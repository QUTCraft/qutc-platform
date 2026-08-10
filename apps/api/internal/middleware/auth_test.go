package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAccessTokenFromRequestPrefersBearerToken(t *testing.T) {
	request := httptest.NewRequest("GET", "/api/v1/admin", nil)
	request.Header.Set("Authorization", "Bearer api-token")
	request.AddCookie(&http.Cookie{Name: AccessCookieName, Value: "cookie-token"})
	if got := accessTokenFromRequest(request); got != "api-token" {
		t.Fatalf("accessTokenFromRequest() = %q, want bearer token", got)
	}
}

func TestAccessTokenFromRequestFallsBackToCookie(t *testing.T) {
	request := httptest.NewRequest("GET", "/api/v1/admin", nil)
	request.AddCookie(&http.Cookie{Name: AccessCookieName, Value: "cookie-token"})
	if got := accessTokenFromRequest(request); got != "cookie-token" {
		t.Fatalf("accessTokenFromRequest() = %q, want cookie token", got)
	}
}

func TestAccessTokenFromRequestRejectsMissingCredentials(t *testing.T) {
	request := httptest.NewRequest("GET", "/api/v1/admin", nil)
	if got := accessTokenFromRequest(request); got != "" {
		t.Fatalf("accessTokenFromRequest() = %q, want empty token", got)
	}
}
