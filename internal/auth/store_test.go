package auth

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStoreProfilePathsAndPermissions(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStoreInDir("work", dir)
	if err != nil {
		t.Fatal(err)
	}
	oauth1 := NewOAuth1Token("one", "secret")
	oauth2 := OAuth2Token{TokenType: "Bearer", AccessToken: "two", ExpiresAt: time.Now().Add(time.Hour).Unix()}
	if err := store.SaveTokens(oauth1, oauth2); err != nil {
		t.Fatal(err)
	}
	if store.Dir() != filepath.Join(dir, "work") {
		t.Fatalf("unexpected dir: %s", store.Dir())
	}
	_, _, ok, err := store.LoadTokens()
	if err != nil || !ok {
		t.Fatalf("load tokens ok=%v err=%v", ok, err)
	}
	info, err := os.Stat(filepath.Join(store.Dir(), oauth1File))
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("token file perms = %o, want 600", got)
	}
}

func TestInvalidProfileRejected(t *testing.T) {
	if _, err := NewStoreInDir("../bad", t.TempDir()); err == nil {
		t.Fatal("expected invalid profile error")
	}
}
