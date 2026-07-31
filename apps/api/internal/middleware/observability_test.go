package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/service"
	"github.com/gin-gonic/gin"
)

func TestRequestIDPreservesSafeValueAndReplacesUnsafeValue(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name       string
		requestID  string
		wantSameID bool
	}{
		{name: "safe", requestID: "client-request_2026.07:30-1", wantSameID: true},
		{name: "unsafe newline", requestID: "forged\nlog-entry", wantSameID: false},
		{name: "too long", requestID: strings.Repeat("a", 65), wantSameID: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			router := gin.New()
			router.Use(RequestID())
			router.GET("/probe", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"request_id": EnsureRequestID(c)})
			})
			request := httptest.NewRequest(http.MethodGet, "/probe", nil)
			request.Header.Set("X-Request-ID", test.requestID)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			got := response.Header().Get("X-Request-ID")
			if !requestIDPattern.MatchString(got) {
				t.Fatalf("generated request ID %q is not safe", got)
			}
			if test.wantSameID && got != test.requestID {
				t.Fatalf("request ID = %q, want %q", got, test.requestID)
			}
			if !test.wantSameID && got == test.requestID {
				t.Fatalf("unsafe request ID %q was preserved", got)
			}
		})
	}
}

func TestStructuredLoggerIncludesCorrelationWithoutSensitiveInput(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	router := gin.New()
	router.Use(RequestID(), StructuredLogger(logger))
	router.POST("/probe", func(c *gin.Context) {
		c.Set(principalKey, service.Principal{UserID: "user-1", OrganizationID: "org-1", Email: "private@example.com"})
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodPost, "/probe?secret=query-secret", strings.NewReader(`{"password":"body-secret"}`))
	request.Header.Set("Authorization", "Bearer token-secret")
	request.Header.Set("X-Request-ID", "request-safe-1")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	var entry map[string]any
	if err := json.Unmarshal(output.Bytes(), &entry); err != nil {
		t.Fatalf("decode structured log: %v; output=%s", err, output.String())
	}
	for key, want := range map[string]any{
		"event":           "http_request",
		"request_id":      "request-safe-1",
		"method":          http.MethodPost,
		"path":            "/probe",
		"route":           "/probe",
		"user_id":         "user-1",
		"organization_id": "org-1",
	} {
		if got := entry[key]; got != want {
			t.Fatalf("log field %s = %#v, want %#v", key, got, want)
		}
	}
	logged := output.String()
	for _, secret := range []string{"query-secret", "body-secret", "token-secret", "private@example.com", "Authorization", "password"} {
		if strings.Contains(logged, secret) {
			t.Fatalf("structured log exposed %q: %s", secret, logged)
		}
	}
}
