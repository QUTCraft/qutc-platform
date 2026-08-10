//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/model"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/platform/storage"
	"github.com/google/uuid"
)

func TestS5S3MediaUploadAndPublicDownload(t *testing.T) {
	endpoint := os.Getenv("S3_TEST_ENDPOINT")
	accessKey := os.Getenv("S3_TEST_ACCESS_KEY")
	secretKey := os.Getenv("S3_TEST_SECRET_KEY")
	if endpoint == "" || accessKey == "" || secretKey == "" {
		t.Skip("S3 test environment is not configured")
	}

	cfg := loadIntegrationConfig(t)
	client := &http.Client{Timeout: 10 * time.Second}
	db := openIntegrationDB(t, cfg.mysqlDSN)
	token := loginAsOwner(t, client, cfg)

	contentResponse := request(t, client, http.MethodPost, cfg.apiURL+"/api/v1/admin/content", token, map[string]any{
		"title":    "S3 媒体闭环 " + uuid.NewString(),
		"type":     "resource",
		"category": "document",
		"excerpt":  "真实 MinIO 上传下载集成测试，结束后自动清理。",
		"body":     "S3 integration fixture",
	}, http.StatusCreated)
	var contentEnvelope apiEnvelope[contentDTO]
	decodeJSON(t, contentResponse, &contentEnvelope)
	contentID := contentEnvelope.Data.ID
	if contentID == "" {
		t.Fatal("created resource did not include id")
	}

	var asset model.MediaAsset
	t.Cleanup(func() {
		if asset.StoragePath != "" {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			store, err := storage.NewS3(ctx, storage.Config{
				Endpoint:  endpoint,
				AccessKey: accessKey,
				SecretKey: secretKey,
				Bucket:    envOrDefault("S3_TEST_BUCKET", "qutcraft-media"),
				Region:    "us-east-1",
			})
			if err != nil {
				t.Errorf("create cleanup S3 client: %v", err)
			} else if err := store.Delete(ctx, asset.StoragePath); err != nil {
				t.Errorf("delete S3 test object: %v", err)
			}
		}
		cleanupContentFixture(t, db, contentID)
	})

	pngBytes, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=")
	if err != nil {
		t.Fatalf("decode PNG fixture: %v", err)
	}
	assetID := uploadAssetFixture(t, client, cfg.apiURL+"/api/v1/admin/assets", token, contentID, pngBytes)
	if err := db.Where("id = ?", assetID).First(&asset).Error; err != nil {
		t.Fatalf("load uploaded asset metadata: %v", err)
	}
	if asset.StorageDriver != "s3" {
		t.Fatalf("storage_driver = %q, want s3", asset.StorageDriver)
	}
	if asset.StoragePath == "" || asset.StoragePath == asset.OriginalName {
		t.Fatalf("unsafe storage path = %q", asset.StoragePath)
	}

	published := changeContentStatus(t, client, cfg, token, contentID, "publish")
	if published.Status != "published" {
		t.Fatalf("publish status = %q, want published", published.Status)
	}
	publicURL := cfg.apiURL + "/api/v1/portal/organizations/" + cfg.organizationSlug + "/assets/" + assetID + "/download"
	responseBody := request(t, client, http.MethodGet, publicURL, "", nil, http.StatusOK)
	if !bytes.Equal(responseBody, pngBytes) {
		t.Fatal("public S3 download differs from uploaded object")
	}
	requireStatus(t, client, http.MethodDelete, cfg.apiURL+"/api/v1/admin/assets/"+assetID, token, nil, http.StatusConflict)

	archived := changeContentStatus(t, client, cfg, token, contentID, "archive")
	if archived.Status != "archived" {
		t.Fatalf("archive status = %q, want archived", archived.Status)
	}
	requireStatus(t, client, http.MethodGet, publicURL, "", nil, http.StatusNotFound)
	requireStatus(t, client, http.MethodDelete, cfg.apiURL+"/api/v1/admin/assets/"+assetID, token, nil, http.StatusOK)
	var remaining int64
	if err := db.Model(&model.MediaAsset{}).Where("id = ?", assetID).Count(&remaining).Error; err != nil {
		t.Fatalf("count deleted asset metadata: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("deleted asset metadata count = %d, want 0", remaining)
	}
	if err := db.Model(&model.Content{}).Where("id = ?", contentID).Count(&remaining).Error; err != nil {
		t.Fatalf("count deleted resource content: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("deleted resource content count = %d, want 0", remaining)
	}
	if err := db.Model(&model.ContentRevision{}).Where("content_id = ?", contentID).Count(&remaining).Error; err != nil {
		t.Fatalf("count deleted resource revisions: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("deleted resource revision count = %d, want 0", remaining)
	}
	asset.StoragePath = ""
}

func uploadAssetFixture(t *testing.T, client *http.Client, url, token, contentID string, payload []byte) string {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	file, err := writer.CreateFormFile("file", "pixel.png")
	if err != nil {
		t.Fatalf("create asset form file: %v", err)
	}
	if _, err := file.Write(payload); err != nil {
		t.Fatalf("write asset form file: %v", err)
	}
	if err := writer.WriteField("content_id", contentID); err != nil {
		t.Fatalf("write content_id: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, &body)
	if err != nil {
		t.Fatalf("create asset upload request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	response, err := client.Do(req)
	if err != nil {
		t.Fatalf("upload asset: %v", err)
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read asset upload response: %v", err)
	}
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("asset upload status = %d, want 201; body=%s", response.StatusCode, responseBody)
	}
	var envelope apiEnvelope[struct {
		ID string `json:"id"`
	}]
	if err := json.Unmarshal(responseBody, &envelope); err != nil {
		t.Fatalf("decode asset upload response: %v", err)
	}
	if envelope.Data.ID == "" {
		t.Fatal("asset upload did not return id")
	}
	return envelope.Data.ID
}
