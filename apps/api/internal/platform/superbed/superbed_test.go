package superbed

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestUploadPostsTokenAndFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", request.Method)
		}
		if err := request.ParseMultipartForm(4 << 20); err != nil {
			t.Fatalf("ParseMultipartForm: %v", err)
		}
		if got := request.FormValue("token"); got != "test-token" {
			t.Fatalf("token = %q, want test-token", got)
		}
		file, header, err := request.FormFile("file")
		if err != nil {
			t.Fatalf("FormFile: %v", err)
		}
		defer file.Close()
		if header.Filename != "cover.png" {
			t.Fatalf("filename = %q, want cover.png", header.Filename)
		}
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(`{"err":0,"url":"https://img.superbed.example/abc123"}`))
	}))
	defer server.Close()

	uploader := New("test-token", server.URL, time.Second)
	url, err := uploader.Upload(context.Background(), "cover.png", []byte("fake-png-bytes"))
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if url != "https://img.superbed.example/abc123" {
		t.Fatalf("url = %q", url)
	}
}

func TestUploadReportsServiceError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"err":1,"msg":"invalid token"}`))
	}))
	defer server.Close()

	uploader := New("bad-token", server.URL, time.Second)
	if _, err := uploader.Upload(context.Background(), "cover.png", []byte("bytes")); err == nil || !strings.Contains(err.Error(), "invalid token") {
		t.Fatalf("expected invalid token error, got %v", err)
	}
}

func TestUploaderDisabledWithoutToken(t *testing.T) {
	if New("", DefaultUploadURL, time.Second).Enabled() {
		t.Fatal("uploader without token should be disabled")
	}
	if (&Uploader{}).Enabled() {
		t.Fatal("zero value uploader should be disabled")
	}
}
