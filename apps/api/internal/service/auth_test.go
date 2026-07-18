package service

import "testing"

func TestTokenHashIsStableAndDoesNotExposeToken(t *testing.T) {
	raw := "a-refresh-token-that-must-not-be-stored-directly"
	first := tokenHash(raw)
	second := tokenHash(raw)
	if first != second {
		t.Fatal("expected token hashes to be stable")
	}
	if first == raw {
		t.Fatal("refresh token must not be stored directly")
	}
}
