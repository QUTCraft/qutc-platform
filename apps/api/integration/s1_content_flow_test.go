//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/handler"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/model"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/platform/storage"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type integrationConfig struct {
	apiURL           string
	mysqlDSN         string
	redisAddr        string
	adminEmail       string
	adminPassword    string
	organizationSlug string
	cacheNamespace   string
}

type apiEnvelope[T any] struct {
	Data T `json:"data"`
	Meta struct {
		Page     int   `json:"page"`
		PageSize int   `json:"page_size"`
		Total    int64 `json:"total"`
	} `json:"meta"`
}

type contentDTO struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Type     string `json:"type"`
	Status   string `json:"status"`
	Body     string `json:"body"`
	Category string `json:"category"`
}

type publicPostDTO struct {
	ID string `json:"id"`
}

func TestS1ContentLifecycleAndCacheInvalidation(t *testing.T) {
	cfg := loadIntegrationConfig(t)
	client := &http.Client{Timeout: 10 * time.Second}
	db := openIntegrationDB(t, cfg.mysqlDSN)
	redisClient := redis.NewClient(&redis.Options{Addr: cfg.redisAddr})
	t.Cleanup(func() { _ = redisClient.Close() })

	accessToken := loginAsOwner(t, client, cfg)
	postsCacheKey := fmt.Sprintf("qutc:%s:portal:%s:posts:default", cfg.cacheNamespace, cfg.organizationSlug)

	for iteration := 1; iteration <= 3; iteration++ {
		t.Run(fmt.Sprintf("round_%d", iteration), func(t *testing.T) {
			content := createDraft(t, client, cfg, accessToken, iteration)
			detailCacheKey := fmt.Sprintf("qutc:%s:portal:%s:content:%s", cfg.cacheNamespace, cfg.organizationSlug, content.ID)
			t.Cleanup(func() {
				cleanupContentFixture(t, db, content.ID)
				_ = redisClient.Del(context.Background(), postsCacheKey, detailCacheKey).Err()
			})

			adminDetail := getAdminContent(t, client, cfg, accessToken, content.ID)
			if adminDetail.ID != content.ID || adminDetail.Body != content.Body || adminDetail.Status != "draft" {
				t.Fatalf("admin content detail = %+v, want complete draft %+v", adminDetail, content)
			}

			posts := getPublicPosts(t, client, cfg)
			if containsPost(posts, content.ID) {
				t.Fatalf("draft content %s leaked into Portal posts", content.ID)
			}
			requireCacheKey(t, redisClient, postsCacheKey, true)
			requireStatus(t, client, http.MethodGet, portalContentURL(cfg, content.ID), "", nil, http.StatusNotFound)

			submitted := submitContentReview(t, client, cfg, accessToken, content.ID)
			if submitted.Status != "review" {
				t.Fatalf("submit status = %q, want review", submitted.Status)
			}
			requireStatus(t, client, http.MethodGet, portalContentURL(cfg, content.ID), "", nil, http.StatusNotFound)

			published := changeContentStatus(t, client, cfg, accessToken, content.ID, "publish")
			if published.Status != "published" {
				t.Fatalf("publish status = %q, want published", published.Status)
			}
			requireCacheKey(t, redisClient, postsCacheKey, false)

			posts = getPublicPosts(t, client, cfg)
			if !containsPost(posts, content.ID) {
				t.Fatalf("published content %s missing from Portal posts", content.ID)
			}
			requireCacheKey(t, redisClient, postsCacheKey, true)

			publicBody := getPublicContent(t, client, cfg, content.ID)
			if publicBody["body"] != content.Body {
				t.Fatalf("public body = %#v, want %q", publicBody["body"], content.Body)
			}
			for _, privateField := range []string{"organization_id", "author_user_id", "status"} {
				if _, leaked := publicBody[privateField]; leaked {
					t.Fatalf("Portal detail leaked private field %q", privateField)
				}
			}
			requireCacheKey(t, redisClient, detailCacheKey, true)
			requireStatus(t, client, http.MethodPost, adminContentActionURL(cfg, content.ID, "publish"), accessToken, nil, http.StatusConflict)

			archived := changeContentStatus(t, client, cfg, accessToken, content.ID, "archive")
			if archived.Status != "archived" {
				t.Fatalf("archive status = %q, want archived", archived.Status)
			}
			requireCacheKey(t, redisClient, postsCacheKey, false)
			requireCacheKey(t, redisClient, detailCacheKey, false)

			posts = getPublicPosts(t, client, cfg)
			if containsPost(posts, content.ID) {
				t.Fatalf("archived content %s remained in Portal posts", content.ID)
			}
			requireStatus(t, client, http.MethodGet, portalContentURL(cfg, content.ID), "", nil, http.StatusNotFound)
			requireStatus(t, client, http.MethodPost, adminContentActionURL(cfg, content.ID, "archive"), accessToken, nil, http.StatusConflict)
		})
	}
}

func TestS1ContentOwnershipAndReviewWorkflow(t *testing.T) {
	cfg := loadIntegrationConfig(t)
	client := &http.Client{Timeout: 10 * time.Second}
	db := openIntegrationDB(t, cfg.mysqlDSN)

	var organization model.Organization
	if err := db.Where("slug = ?", cfg.organizationSlug).First(&organization).Error; err != nil {
		t.Fatalf("load organization: %v", err)
	}
	var editorRole model.Role
	if err := db.Where("`key` = ?", "editor").First(&editorRole).Error; err != nil {
		t.Fatalf("load editor role: %v", err)
	}
	password := "S1-Editor-Password-2026!"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash editor password: %v", err)
	}

	createEditor := func(label string) model.User {
		user := model.User{ID: uuid.NewString(), Email: "s1-editor-" + label + "-" + uuid.NewString() + "@example.test", DisplayName: "S1 Editor " + label, PasswordHash: string(hash), State: "active", DefaultOrganizationID: organization.ID}
		if err := db.Create(&user).Error; err != nil {
			t.Fatalf("create editor %s: %v", label, err)
		}
		membership := model.Membership{ID: uuid.NewString(), OrganizationID: organization.ID, UserID: user.ID, State: "active"}
		if err := db.Create(&membership).Error; err != nil {
			t.Fatalf("create editor membership %s: %v", label, err)
		}
		if err := db.Create(&model.MembershipRole{MembershipID: membership.ID, RoleID: editorRole.ID}).Error; err != nil {
			t.Fatalf("assign editor role %s: %v", label, err)
		}
		return user
	}

	author := createEditor("author")
	other := createEditor("other")
	t.Cleanup(func() {
		for _, user := range []model.User{author, other} {
			_ = db.Where("user_id = ?", user.ID).Delete(&model.RefreshToken{}).Error
			var memberships []model.Membership
			_ = db.Where("user_id = ?", user.ID).Find(&memberships).Error
			for _, membership := range memberships {
				_ = db.Where("membership_id = ?", membership.ID).Delete(&model.MembershipRole{}).Error
			}
			_ = db.Where("user_id = ?", user.ID).Delete(&model.Membership{}).Error
			_ = db.Where("id = ?", user.ID).Delete(&model.User{}).Error
		}
	})

	authorToken := loginWithCredentials(t, client, cfg, author.Email, password)
	otherToken := loginWithCredentials(t, client, cfg, other.Email, password)
	ownerToken := loginAsOwner(t, client, cfg)
	content := createDraft(t, client, cfg, authorToken, 99)
	t.Cleanup(func() { cleanupContentFixture(t, db, content.ID) })

	updatePayload := map[string]any{"title": content.Title + " updated", "type": "news", "category": "integration", "excerpt": "ownership test", "body": "author update"}
	requireStatus(t, client, http.MethodPatch, cfg.apiURL+"/api/v1/admin/content/"+content.ID, otherToken, updatePayload, http.StatusForbidden)
	requireStatus(t, client, http.MethodPost, adminContentActionURL(cfg, content.ID, "submit"), otherToken, map[string]string{"note": "not mine"}, http.StatusForbidden)
	requireStatus(t, client, http.MethodPatch, cfg.apiURL+"/api/v1/admin/content/"+content.ID, authorToken, updatePayload, http.StatusOK)

	submitted := submitContentReview(t, client, cfg, authorToken, content.ID)
	if submitted.Status != "review" {
		t.Fatalf("author submit status = %q, want review", submitted.Status)
	}
	requireStatus(t, client, http.MethodPatch, cfg.apiURL+"/api/v1/admin/content/"+content.ID, authorToken, updatePayload, http.StatusConflict)
	published := changeContentStatus(t, client, cfg, ownerToken, content.ID, "publish")
	if published.Status != "published" {
		t.Fatalf("review approval status = %q, want published", published.Status)
	}
	requireStatus(t, client, http.MethodPost, adminContentActionURL(cfg, content.ID, "request-archive"), otherToken, map[string]string{"note": "not mine"}, http.StatusForbidden)
	requireStatus(t, client, http.MethodPost, adminContentActionURL(cfg, content.ID, "request-archive"), authorToken, map[string]string{"note": "needs correction"}, http.StatusOK)
	archived := changeContentStatus(t, client, cfg, ownerToken, content.ID, "archive")
	if archived.Status != "archived" {
		t.Fatalf("archive approval status = %q, want archived", archived.Status)
	}
}

func TestS1PublishedAssetDownloadBoundary(t *testing.T) {
	cfg := loadIntegrationConfig(t)
	db := openIntegrationDB(t, cfg.mysqlDSN)
	gin.SetMode(gin.TestMode)

	var organization model.Organization
	if err := db.Where("slug = ?", cfg.organizationSlug).First(&organization).Error; err != nil {
		t.Fatalf("load organization: %v", err)
	}
	var owner model.User
	if err := db.Where("email = ?", cfg.adminEmail).First(&owner).Error; err != nil {
		t.Fatalf("load owner: %v", err)
	}

	contentID := uuid.NewString()
	assetID := uuid.NewString()
	pngBytes, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=")
	if err != nil {
		t.Fatalf("decode PNG fixture: %v", err)
	}
	assetRoot := t.TempDir()
	assetPath := filepath.Join(assetRoot, assetID+".png")
	if err := os.WriteFile(assetPath, pngBytes, 0o600); err != nil {
		t.Fatalf("write PNG fixture: %v", err)
	}

	content := model.Content{
		ID:             contentID,
		OrganizationID: organization.ID,
		AuthorUserID:   owner.ID,
		Title:          "S1 资源下载集成测试",
		Type:           "resource",
		Category:       "document",
		Status:         "draft",
		Body:           "仅用于自动化测试。",
	}
	asset := model.MediaAsset{
		ID:             assetID,
		OrganizationID: organization.ID,
		ContentID:      contentID,
		UploadedBy:     owner.ID,
		OriginalName:   "pixel.png",
		StoredName:     assetID + ".png",
		MimeType:       "image/png",
		SizeBytes:      int64(len(pngBytes)),
		StoragePath:    assetPath,
	}
	if err := db.Create(&content).Error; err != nil {
		t.Fatalf("create content fixture: %v", err)
	}
	if err := db.Create(&asset).Error; err != nil {
		_ = db.Delete(&content).Error
		t.Fatalf("create asset fixture: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Where("id = ?", assetID).Delete(&model.MediaAsset{}).Error
		_ = db.Where("id = ?", contentID).Delete(&model.Content{}).Error
	})

	mediaStorage, err := storage.NewLocal(assetRoot)
	if err != nil {
		t.Fatalf("create integration media storage: %v", err)
	}
	workspace := handler.NewWorkspaceHandlerWithDependencies(db, nil, "integration", mediaStorage)
	if response := downloadPublicAsset(workspace, cfg.organizationSlug, assetID); response.Code != http.StatusNotFound {
		t.Fatalf("draft asset status = %d, want 404", response.Code)
	}

	publishedAt := time.Now().UTC()
	if err := db.Model(&model.Content{}).Where("id = ?", contentID).Updates(map[string]any{
		"status":       "published",
		"published_at": &publishedAt,
	}).Error; err != nil {
		t.Fatalf("publish fixture: %v", err)
	}
	response := downloadPublicAsset(workspace, cfg.organizationSlug, assetID)
	if response.Code != http.StatusOK {
		t.Fatalf("published asset status = %d, want 200; body=%s", response.Code, response.Body.String())
	}
	if !bytes.Equal(response.Body.Bytes(), pngBytes) {
		t.Fatal("downloaded asset bytes differ from stored file")
	}
	if contentType := response.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "image/png") {
		t.Fatalf("Content-Type = %q, want image/png", contentType)
	}
	if disposition := response.Header().Get("Content-Disposition"); !strings.HasPrefix(disposition, "inline;") {
		t.Fatalf("Content-Disposition = %q, want inline", disposition)
	}

	if response := downloadPublicAsset(workspace, "wrong-organization", assetID); response.Code != http.StatusNotFound {
		t.Fatalf("cross-organization asset status = %d, want 404", response.Code)
	}
	if err := db.Model(&model.Content{}).Where("id = ?", contentID).Updates(map[string]any{
		"status":       "archived",
		"published_at": nil,
	}).Error; err != nil {
		t.Fatalf("archive fixture: %v", err)
	}
	if response := downloadPublicAsset(workspace, cfg.organizationSlug, assetID); response.Code != http.StatusNotFound {
		t.Fatalf("archived asset status = %d, want 404", response.Code)
	}
}

func loadIntegrationConfig(t *testing.T) integrationConfig {
	t.Helper()
	cfg := integrationConfig{
		apiURL:           strings.TrimRight(strings.TrimSpace(os.Getenv("QUTC_INTEGRATION_API_URL")), "/"),
		mysqlDSN:         strings.TrimSpace(os.Getenv("QUTC_INTEGRATION_MYSQL_DSN")),
		redisAddr:        strings.TrimSpace(os.Getenv("QUTC_INTEGRATION_REDIS_ADDR")),
		adminEmail:       strings.TrimSpace(os.Getenv("QUTC_INTEGRATION_ADMIN_EMAIL")),
		adminPassword:    os.Getenv("QUTC_INTEGRATION_ADMIN_PASSWORD"),
		organizationSlug: envOrDefault("QUTC_INTEGRATION_ORGANIZATION_SLUG", "qutcraft"),
		cacheNamespace:   envOrDefault("QUTC_INTEGRATION_CACHE_NAMESPACE", "development"),
	}
	if cfg.apiURL == "" {
		t.Skip("set QUTC_INTEGRATION_API_URL to run Compose integration tests")
	}
	for name, value := range map[string]string{
		"QUTC_INTEGRATION_MYSQL_DSN":      cfg.mysqlDSN,
		"QUTC_INTEGRATION_REDIS_ADDR":     cfg.redisAddr,
		"QUTC_INTEGRATION_ADMIN_EMAIL":    cfg.adminEmail,
		"QUTC_INTEGRATION_ADMIN_PASSWORD": cfg.adminPassword,
	} {
		if value == "" {
			t.Fatalf("%s is required", name)
		}
	}
	return cfg
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func openIntegrationDB(t *testing.T, dsn string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("connect integration database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("open integration database handle: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db
}

func loginAsOwner(t *testing.T, client *http.Client, cfg integrationConfig) string {
	t.Helper()
	return loginWithCredentials(t, client, cfg, cfg.adminEmail, cfg.adminPassword)
}

func submitContentReview(t *testing.T, client *http.Client, cfg integrationConfig, token, contentID string) contentDTO {
	t.Helper()
	responseBody := request(t, client, http.MethodPost, adminContentActionURL(cfg, contentID, "submit"), token, map[string]string{"note": "S1 integration review"}, http.StatusOK)
	var envelope apiEnvelope[contentDTO]
	decodeJSON(t, responseBody, &envelope)
	return envelope.Data
}

func createDraft(t *testing.T, client *http.Client, cfg integrationConfig, token string, iteration int) contentDTO {
	t.Helper()
	bodyText := fmt.Sprintf("S1 自动化内容闭环第 %d 轮。", iteration)
	responseBody := request(t, client, http.MethodPost, cfg.apiURL+"/api/v1/admin/content", token, map[string]any{
		"title":    fmt.Sprintf("S1 内容闭环 %d %s", iteration, uuid.NewString()),
		"type":     "news",
		"category": "integration",
		"excerpt":  "自动化集成测试，执行后会清理。",
		"body":     bodyText,
	}, http.StatusCreated)
	var envelope apiEnvelope[contentDTO]
	decodeJSON(t, responseBody, &envelope)
	if envelope.Data.ID == "" || envelope.Data.Status != "draft" {
		t.Fatalf("created content = %+v, want draft with id", envelope.Data)
	}
	return envelope.Data
}

func changeContentStatus(t *testing.T, client *http.Client, cfg integrationConfig, token, contentID, action string) contentDTO {
	t.Helper()
	responseBody := request(t, client, http.MethodPost, adminContentActionURL(cfg, contentID, action), token, nil, http.StatusOK)
	var envelope apiEnvelope[contentDTO]
	decodeJSON(t, responseBody, &envelope)
	return envelope.Data
}

func getPublicPosts(t *testing.T, client *http.Client, cfg integrationConfig) []publicPostDTO {
	t.Helper()
	responseBody := request(t, client, http.MethodGet, cfg.apiURL+"/api/v1/portal/organizations/"+cfg.organizationSlug+"/posts", "", nil, http.StatusOK)
	var envelope apiEnvelope[[]publicPostDTO]
	decodeJSON(t, responseBody, &envelope)
	return envelope.Data
}

func getAdminContent(t *testing.T, client *http.Client, cfg integrationConfig, token, contentID string) contentDTO {
	t.Helper()
	responseBody := request(t, client, http.MethodGet, cfg.apiURL+"/api/v1/admin/content/"+contentID, token, nil, http.StatusOK)
	var envelope apiEnvelope[contentDTO]
	decodeJSON(t, responseBody, &envelope)
	return envelope.Data
}

func getPublicContent(t *testing.T, client *http.Client, cfg integrationConfig, contentID string) map[string]any {
	t.Helper()
	responseBody := request(t, client, http.MethodGet, portalContentURL(cfg, contentID), "", nil, http.StatusOK)
	var envelope apiEnvelope[map[string]any]
	decodeJSON(t, responseBody, &envelope)
	return envelope.Data
}

func containsPost(posts []publicPostDTO, contentID string) bool {
	for _, post := range posts {
		if post.ID == contentID {
			return true
		}
	}
	return false
}

func portalContentURL(cfg integrationConfig, contentID string) string {
	return cfg.apiURL + "/api/v1/portal/organizations/" + cfg.organizationSlug + "/content/" + contentID
}

func adminContentActionURL(cfg integrationConfig, contentID, action string) string {
	return cfg.apiURL + "/api/v1/admin/content/" + contentID + "/" + action
}

func request(t *testing.T, client *http.Client, method, url, token string, payload any, expectedStatus int) []byte {
	t.Helper()
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("encode request body: %v", err)
		}
		body = bytes.NewReader(encoded)
	}
	request, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	request.Header.Set("Accept", "application/json")
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	if response.StatusCode != expectedStatus {
		t.Fatalf("%s %s status = %d, want %d; body=%s", method, url, response.StatusCode, expectedStatus, responseBody)
	}
	return responseBody
}

func requireStatus(t *testing.T, client *http.Client, method, url, token string, payload any, expectedStatus int) {
	t.Helper()
	request(t, client, method, url, token, payload, expectedStatus)
}

func decodeJSON(t *testing.T, value []byte, destination any) {
	t.Helper()
	if err := json.Unmarshal(value, destination); err != nil {
		t.Fatalf("decode response JSON: %v; body=%s", err, value)
	}
}

func requireCacheKey(t *testing.T, client *redis.Client, key string, expected bool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	count, err := client.Exists(ctx, key).Result()
	if err != nil {
		t.Fatalf("inspect Redis key %s: %v", key, err)
	}
	if exists := count > 0; exists != expected {
		t.Fatalf("Redis key %s exists = %t, want %t", key, exists, expected)
	}
}

func cleanupContentFixture(t *testing.T, db *gorm.DB, contentID string) {
	t.Helper()
	var reviews []model.ContentReviewRequest
	if err := db.Where("content_id = ?", contentID).Find(&reviews).Error; err != nil {
		t.Errorf("find content reviews for %s: %v", contentID, err)
	}
	for _, review := range reviews {
		if err := db.Where("target_type = ? AND target_id = ?", "content_review", review.ID).Delete(&model.NotificationOutbox{}).Error; err != nil {
			t.Errorf("cleanup content notifications for %s: %v", contentID, err)
		}
	}
	if err := db.Where("content_id = ?", contentID).Delete(&model.ContentReviewRequest{}).Error; err != nil {
		t.Errorf("cleanup content reviews for %s: %v", contentID, err)
	}
	if err := db.Where("target_type = ? AND target_id = ?", "content", contentID).Delete(&model.AuditEvent{}).Error; err != nil {
		t.Errorf("cleanup audit events for %s: %v", contentID, err)
	}
	if err := db.Where("content_id = ?", contentID).Delete(&model.MediaAsset{}).Error; err != nil {
		t.Errorf("cleanup media assets for %s: %v", contentID, err)
	}
	if err := db.Where("id = ?", contentID).Delete(&model.Content{}).Error; err != nil {
		t.Errorf("cleanup content %s: %v", contentID, err)
	}
}

func downloadPublicAsset(workspace *handler.WorkspaceHandler, slug, assetID string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/v1/portal/organizations/"+slug+"/assets/"+assetID+"/download", nil)
	context.Params = gin.Params{{Key: "slug", Value: slug}, {Key: "id", Value: assetID}}
	workspace.DownloadAsset(context)
	return recorder
}
