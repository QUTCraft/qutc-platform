package handler

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/model"
	"github.com/gin-gonic/gin"
)

func TestUploadAssetRejectsOversizedRequestBeforePersistence(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	file, err := writer.CreateFormFile("file", "oversized.png")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := file.Write(bytes.Repeat([]byte{0x89}, maxAssetRequestSize)); err != nil {
		t.Fatalf("write multipart payload: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	handler := &WorkspaceHandler{}
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/assets", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	context.Request = request

	handler.UploadAsset(context)
	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusRequestEntityTooLarge, recorder.Body.String())
	}
	var response struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Error.Code != "asset.request_too_large" {
		t.Fatalf("error code = %q, want asset.request_too_large", response.Error.Code)
	}
}

func TestNormalizeAssetResourceInput(t *testing.T) {
	asset := model.MediaAsset{OriginalName: "社团资料包.zip", MimeType: "application/zip"}
	input, err := normalizeAssetResourceInput(asset, publishAssetResourceRequest{Title: "社团资料包"})
	if err != nil {
		t.Fatalf("normalize resource input: %v", err)
	}
	if input.Type != "resource" || input.Category != "package" {
		t.Fatalf("normalized type/category = %q/%q, want resource/package", input.Type, input.Category)
	}
	if input.Excerpt != "公开文件：社团资料包.zip" || input.Body != input.Excerpt {
		t.Fatalf("unexpected generated description: %#v", input)
	}
}

func TestNormalizeAssetResourceInputRejectsInvalidMetadata(t *testing.T) {
	asset := model.MediaAsset{OriginalName: "guide.pdf", MimeType: "application/pdf"}
	for name, request := range map[string]publishAssetResourceRequest{
		"missing title": {Kind: "document"},
		"invalid kind":  {Title: "Guide", Kind: "executable"},
		"long summary":  {Title: "Guide", Kind: "document", Description: strings.Repeat("长", 501)},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := normalizeAssetResourceInput(asset, request); err == nil {
				t.Fatal("invalid resource metadata was accepted")
			}
		})
	}
}

func TestInferResourceKind(t *testing.T) {
	tests := map[string]string{
		"application/pdf":              "document",
		"image/png":                    "document",
		"application/zip":              "package",
		"application/x-zip-compressed": "package",
		"video/mp4":                    "video",
	}
	for mimeType, want := range tests {
		if got := inferResourceKind(mimeType); got != want {
			t.Fatalf("inferResourceKind(%q) = %q, want %q", mimeType, got, want)
		}
	}
}
