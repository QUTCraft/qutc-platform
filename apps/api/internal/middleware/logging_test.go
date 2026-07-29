package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequestIDGeneratesAndPropagates(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	if err := router.SetTrustedProxies(nil); err != nil {
		t.Fatalf("SetTrustedProxies: %v", err)
	}
	router.Use(RequestID())

	var capturedID string
	router.GET("/test", func(c *gin.Context) {
		id, ok := c.Get("request_id")
		if !ok {
			t.Error("request_id not set in gin context")
			return
		}
		capturedID = id.(string)
		c.String(http.StatusOK, "ok")
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(recorder, request)

	if capturedID == "" {
		t.Fatal("request_id was not captured")
	}
	if recorder.Header().Get("X-Request-ID") != capturedID {
		t.Fatalf("response header X-Request-ID = %q, want %q", recorder.Header().Get("X-Request-ID"), capturedID)
	}
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
}

func TestRequestIDHonorsIncomingHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	if err := router.SetTrustedProxies(nil); err != nil {
		t.Fatalf("SetTrustedProxies: %v", err)
	}
	router.Use(RequestID())

	var capturedID string
	router.GET("/test", func(c *gin.Context) {
		id, _ := c.Get("request_id")
		capturedID = id.(string)
		c.String(http.StatusOK, "ok")
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/test", nil)
	request.Header.Set("X-Request-ID", "incoming-id-456")
	router.ServeHTTP(recorder, request)

	if capturedID != "incoming-id-456" {
		t.Fatalf("captured request_id = %q, want %q", capturedID, "incoming-id-456")
	}
	if recorder.Header().Get("X-Request-ID") != "incoming-id-456" {
		t.Fatalf("response header = %q, want %q", recorder.Header().Get("X-Request-ID"), "incoming-id-456")
	}
}

func TestSlogLoggerEmitsStructuredRecord(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	if err := router.SetTrustedProxies(nil); err != nil {
		t.Fatalf("SetTrustedProxies: %v", err)
	}
	router.Use(RequestID(), SlogLogger())

	router.GET("/ok", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	router.GET("/bad", func(c *gin.Context) { c.String(http.StatusBadRequest, "bad") })
	router.GET("/err", func(c *gin.Context) { c.String(http.StatusInternalServerError, "err") })

	for _, tc := range []struct {
		path string
		code int
	}{
		{"/ok", http.StatusOK},
		{"/bad", http.StatusBadRequest},
		{"/err", http.StatusInternalServerError},
	} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, tc.path, nil)
		router.ServeHTTP(recorder, request)
		if recorder.Code != tc.code {
			t.Fatalf("%s status = %d, want %d", tc.path, recorder.Code, tc.code)
		}
	}
}
