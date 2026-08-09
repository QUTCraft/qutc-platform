package service

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/model"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/platform/mailadapter"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/platform/storage"
)

func TestNormalizeS3EndpointForNonTechnicalInput(t *testing.T) {
	tests := []struct {
		value      string
		initialSSL bool
		endpoint   string
		useSSL     bool
		valid      bool
	}{
		{value: "https://minio.example.org:9000", endpoint: "minio.example.org:9000", useSSL: true, valid: true},
		{value: "http://minio:9000", initialSSL: true, endpoint: "minio:9000", useSSL: false, valid: true},
		{value: "minio:9000", endpoint: "minio:9000", useSSL: false, valid: true},
		{value: "https://user@example.org", valid: false},
		{value: "https://example.org/path", valid: false},
	}
	for _, test := range tests {
		endpoint, useSSL, err := normalizeS3Endpoint(test.value, test.initialSSL)
		if (err == nil) != test.valid {
			t.Fatalf("normalizeS3Endpoint(%q) error=%v, valid=%v", test.value, err, test.valid)
		}
		if err == nil && (endpoint != test.endpoint || useSSL != test.useSSL) {
			t.Fatalf("normalizeS3Endpoint(%q)=(%q,%v), want (%q,%v)", test.value, endpoint, useSSL, test.endpoint, test.useSSL)
		}
	}
}

func TestIntegrationViewNeverReturnsPlainCredentials(t *testing.T) {
	key := deriveAgentCredentialKey("integration-test-secret")
	encrypt := func(value string) string {
		result, err := encryptAgentCredential(key, value)
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	service := &IntegrationService{
		defaults: IntegrationDefaults{
			Email:   mailadapter.Config{Driver: "disabled"},
			Storage: storage.Config{Driver: "local", LocalRoot: t.TempDir()},
		},
		credentialKey: key,
	}
	configuration := model.IntegrationConfiguration{
		PublicWebBaseURL: "https://cms.example.org", EmailDriver: "smtp", SMTPHost: "smtp.example.org", SMTPPort: 587,
		SMTPUsername: "mailer", SMTPPassword: encrypt("smtp-super-secret"), SMTPFromAddress: "noreply@example.org",
		SMTPFromName: "Example", SMTPSecurity: "starttls", SMTPTimeoutSeconds: 8,
		StorageDriver: "s3", S3Endpoint: "minio.example.org", S3AccessKey: encrypt("storage-access-secret"),
		S3SecretKey: encrypt("storage-secret-secret"), S3Bucket: "media", S3UseSSL: true, UpdatedAt: time.Now().UTC(),
	}
	view, err := service.view(configuration, true)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(view)
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	for _, secret := range []string{"smtp-super-secret", "storage-access-secret", "storage-secret-secret"} {
		if strings.Contains(text, secret) {
			t.Fatalf("response leaked credential %q: %s", secret, text)
		}
	}
	if !view.Email.PasswordConfigured || !view.Storage.AccessKeyConfigured || !view.Storage.SecretKeyConfigured {
		t.Fatal("configured credential flags were not returned")
	}
}
