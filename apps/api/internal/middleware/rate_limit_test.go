package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestRateLimiterRejectsAndResets(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Date(2026, time.July, 28, 12, 0, 0, 0, time.UTC)
	limiter := NewRateLimiter(2, time.Minute)
	limiter.now = func() time.Time { return now }

	router := gin.New()
	if err := router.SetTrustedProxies(nil); err != nil {
		t.Fatalf("SetTrustedProxies: %v", err)
	}
	router.GET("/limited", limiter.Middleware(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	for index, expected := range []int{http.StatusNoContent, http.StatusNoContent, http.StatusTooManyRequests} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/limited", nil)
		request.RemoteAddr = "192.0.2.10:1234"
		router.ServeHTTP(recorder, request)
		if recorder.Code != expected {
			t.Fatalf("request %d status = %d, want %d", index+1, recorder.Code, expected)
		}
		if recorder.Header().Get("X-RateLimit-Limit") != "2" {
			t.Fatalf("request %d missing limit header", index+1)
		}
	}

	now = now.Add(time.Minute)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/limited", nil)
	request.RemoteAddr = "192.0.2.10:1234"
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("request after reset status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
}

func TestRateLimiterSeparatesRoutesAndClients(t *testing.T) {
	limiter := NewRateLimiter(1, time.Minute)
	now := time.Now().UTC()

	if allowed, _, _ := limiter.consume("198.51.100.1:/a", now); !allowed {
		t.Fatal("first client should be allowed")
	}
	if allowed, _, _ := limiter.consume("198.51.100.1:/b", now); !allowed {
		t.Fatal("different route should have an independent window")
	}
	if allowed, _, _ := limiter.consume("198.51.100.2:/a", now); !allowed {
		t.Fatal("different client should have an independent window")
	}
	if allowed, _, _ := limiter.consume("198.51.100.1:/a", now); allowed {
		t.Fatal("same client and route should be limited")
	}
}
