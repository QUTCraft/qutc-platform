package logging

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
)

func TestSanitizingHandlerRedactsSensitiveKeys(t *testing.T) {
	tests := []struct {
		key   string
		value string
		redact bool
	}{
		{"password", "secret123", true},
		{"user_password", "secret123", true},
		{"token", "abc.def.ghi", true},
		{"refresh_token", "abc.def.ghi", true},
		{"authorization", "Bearer xyz", true},
		{"auth_code", "smtp-code-456", true},
		{"rcon_password", "minecraft-pass", true},
		{"minio_secret", "minio-key", true},
		{"jwt_secret", "jwt-key-789", true},
		{"api_key", "sk-abcdef", true},
		{"private_key", "-----BEGIN RSA PRIVATE KEY-----", true},
		{"smtp_password", "mail-pass", true},
		{"email", "user@example.com", false},
		{"display_name", "Yukino", false},
		{"organization_id", "org-123", false},
		{"action", "content.publish", false},
		{"request_id", "req-abc", false},
		{"status", "published", false},
	}

	for _, test := range tests {
		t.Run(test.key, func(t *testing.T) {
			var buf bytes.Buffer
			handler := newSanitizingHandler(slog.NewJSONHandler(&buf, nil))
			logger := slog.New(handler)
			logger.Info("test message", test.key, test.value)

			output := buf.String()
			if test.redact && strings.Contains(output, test.value) {
				t.Fatalf("expected sensitive key %q to be redacted, but value appeared in output: %s", test.key, output)
			}
			if test.redact && !strings.Contains(output, "[REDACTED]") {
				t.Fatalf("expected sensitive key %q to be replaced with [REDACTED]: %s", test.key, output)
			}
			if !test.redact && !strings.Contains(output, test.value) {
				t.Fatalf("expected non-sensitive key %q to keep its value: %s", test.key, output)
			}
		})
	}
}

func TestWithRequestIDAndCtx(t *testing.T) {
	Init("development")
	ctx := WithRequestID(context.Background(), "req-test-123")
	logger := Ctx(ctx)

	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	logger = Ctx(ctx)
	logger.Info("test")

	output := buf.String()
	if !strings.Contains(output, "request_id=req-test-123") {
		t.Fatalf("expected log to include request_id: %s", output)
	}
}
