package auth

import (
	"testing"
	"time"
)

func TestOAuth2Expiry(t *testing.T) {
	now := time.Unix(100, 0)
	token := OAuth2Token{ExpiresAt: 99, RefreshTokenExpiresAt: 101}
	if !token.Expired(now) {
		t.Fatal("expected token to be expired")
	}
	if token.RefreshExpired(now) {
		t.Fatal("refresh token should still be valid")
	}
}
