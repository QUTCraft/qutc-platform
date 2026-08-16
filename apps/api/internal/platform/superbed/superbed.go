package superbed

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

const DefaultUploadURL = "https://api.superbed.cn/upload"

// Uploader forwards image bytes to the Superbed image hosting service. It is
// intentionally a small, dependency-free adapter so the workspace handler can
// fall back to local storage when it is disabled or the remote upload fails.
type Uploader struct {
	token     string
	uploadURL string
	client    *http.Client
}

type uploadResponse struct {
	Err int    `json:"err"`
	URL string `json:"url"`
	Msg string `json:"msg"`
}

func New(token, uploadURL string, timeout time.Duration) *Uploader {
	if strings.TrimSpace(uploadURL) == "" {
		uploadURL = DefaultUploadURL
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &Uploader{
		token:     strings.TrimSpace(token),
		uploadURL: uploadURL,
		client:    &http.Client{Timeout: timeout},
	}
}

func (u *Uploader) Enabled() bool {
	return u != nil && u.token != "" && u.uploadURL != ""
}

func (u *Uploader) Upload(ctx context.Context, filename string, data []byte) (string, error) {
	if !u.Enabled() {
		return "", fmt.Errorf("superbed uploader is not configured")
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("token", u.token); err != nil {
		return "", err
	}
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return "", err
	}
	if _, err := part.Write(data); err != nil {
		return "", err
	}
	if err := writer.Close(); err != nil {
		return "", err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, u.uploadURL, &body)
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("User-Agent", "qutc-platform/1.0")

	response, err := u.client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("superbed returned status %d", response.StatusCode)
	}

	var result uploadResponse
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&result); err != nil {
		return "", err
	}
	if result.Err != 0 {
		message := strings.TrimSpace(result.Msg)
		if message == "" {
			message = "unknown superbed error"
		}
		return "", fmt.Errorf("superbed upload failed: %s", message)
	}
	url := strings.TrimSpace(result.URL)
	if url == "" {
		return "", fmt.Errorf("superbed returned an empty url")
	}
	return url, nil
}
