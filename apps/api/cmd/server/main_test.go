package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/config"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func TestCORSConfigAllowsConfiguredOriginAndRejectsOthers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(cors.New(corsConfig(config.Config{
		CORSAllowedOrigins: []string{"https://portal.example.test"},
	})))
	router.POST("/api/v1/auth/login", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	tests := []struct {
		name        string
		origin      string
		status      int
		allowOrigin string
	}{
		{name: "allowed", origin: "https://portal.example.test", status: http.StatusNoContent, allowOrigin: "https://portal.example.test"},
		{name: "rejected", origin: "https://evil.example.test", status: http.StatusForbidden},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/login", nil)
			request.Header.Set("Origin", test.origin)
			request.Header.Set("Access-Control-Request-Method", http.MethodPost)
			router.ServeHTTP(recorder, request)
			if recorder.Code != test.status {
				t.Fatalf("status = %d, want %d", recorder.Code, test.status)
			}
			if actual := recorder.Header().Get("Access-Control-Allow-Origin"); actual != test.allowOrigin {
				t.Fatalf("Access-Control-Allow-Origin = %q, want %q", actual, test.allowOrigin)
			}
		})
	}
}
