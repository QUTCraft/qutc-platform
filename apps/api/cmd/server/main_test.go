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
	router.PUT("/api/v1/admin/ai/activity-plans/:plan_id/evaluation", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	tests := []struct {
		name        string
		path        string
		method      string
		origin      string
		status      int
		allowOrigin string
	}{
		{name: "allowed post", path: "/api/v1/auth/login", method: http.MethodPost, origin: "https://portal.example.test", status: http.StatusNoContent, allowOrigin: "https://portal.example.test"},
		{name: "allowed put", path: "/api/v1/admin/ai/activity-plans/test-plan/evaluation", method: http.MethodPut, origin: "https://portal.example.test", status: http.StatusNoContent, allowOrigin: "https://portal.example.test"},
		{name: "rejected", path: "/api/v1/auth/login", method: http.MethodPost, origin: "https://evil.example.test", status: http.StatusForbidden},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodOptions, test.path, nil)
			request.Header.Set("Origin", test.origin)
			request.Header.Set("Access-Control-Request-Method", test.method)
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

func TestCheckReadinessRequiresHTTP200(t *testing.T) {
	tests := []struct {
		name   string
		status int
		wantOK bool
	}{
		{name: "ready", status: http.StatusOK, wantOK: true},
		{name: "unavailable", status: http.StatusServiceUnavailable, wantOK: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
				response.WriteHeader(test.status)
			}))
			defer server.Close()
			err := checkReadiness(server.URL)
			if test.wantOK && err != nil {
				t.Fatalf("checkReadiness() error = %v", err)
			}
			if !test.wantOK && err == nil {
				t.Fatal("checkReadiness() accepted a non-200 response")
			}
		})
	}
}
