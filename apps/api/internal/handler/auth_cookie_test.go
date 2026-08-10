package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/middleware"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/service"
	"github.com/gin-gonic/gin"
)

func TestTokenPairResponseSetsTimedBrowserSessionCookies(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)

	sessionExpiry := time.Now().UTC().Add(7 * 24 * time.Hour).Truncate(time.Second)
	handler := &AuthHandler{refreshTTL: 7 * 24 * time.Hour, secureCookie: true}
	payload := handler.tokenPairResponse(context, service.TokenPair{
		AccessToken: "access-token", RefreshToken: "refresh-token", TokenType: "Bearer", ExpiresIn: 900, SessionExpiresAt: sessionExpiry,
	})

	cookies := cookieMap(recorder.Result().Cookies())
	access := cookies[middleware.AccessCookieName]
	if access == nil || access.Value != "access-token" || access.Path != "/api/v1" || !access.HttpOnly || access.MaxAge != 900 {
		t.Fatalf("unexpected access cookie: %#v", access)
	}
	refresh := cookies[refreshCookieName]
	if refresh == nil || refresh.Value != "refresh-token" || refresh.Path != "/api/v1/auth" || !refresh.HttpOnly || refresh.MaxAge < 7*24*60*60-2 {
		t.Fatalf("unexpected refresh cookie: %#v", refresh)
	}
	marker := cookies[sessionExpiryCookieName]
	if marker == nil || marker.Path != "/" || marker.HttpOnly || marker.Value == "" {
		t.Fatalf("unexpected session expiry marker: %#v", marker)
	}
	if access.Secure || refresh.Secure || marker.Secure {
		t.Fatal("plain HTTP request must not receive unusable Secure cookies")
	}
	if payload["session_expires_at"] != sessionExpiry.Format(time.RFC3339) {
		t.Fatalf("session_expires_at = %#v", payload["session_expires_at"])
	}
}

func TestSessionCookiesHonorForwardedHTTPS(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	context.Request.Header.Set("X-Forwarded-Proto", "https")
	handler := &AuthHandler{refreshTTL: time.Hour, secureCookie: true}
	handler.setSessionCookies(context, service.TokenPair{AccessToken: "access", RefreshToken: "refresh", ExpiresIn: 60, SessionExpiresAt: time.Now().UTC().Add(time.Hour)})
	for _, cookie := range recorder.Result().Cookies() {
		if !cookie.Secure {
			t.Fatalf("cookie %s is not Secure behind HTTPS proxy", cookie.Name)
		}
	}
}

func cookieMap(cookies []*http.Cookie) map[string]*http.Cookie {
	result := make(map[string]*http.Cookie, len(cookies))
	for _, cookie := range cookies {
		result[cookie.Name] = cookie
	}
	return result
}
