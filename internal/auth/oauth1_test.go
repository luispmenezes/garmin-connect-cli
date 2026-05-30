package auth

import (
	"strings"
	"testing"
)

func TestOAuth1SignerIncludesExpectedFields(t *testing.T) {
	signer := OAuth1Signer{Consumer: OAuthConsumer{Key: "consumer", Secret: "secret"}}
	header, err := signer.SignWithTimestampNonce("GET", "https://example.com/path?ticket=ST-123", nil, 1234567890, "nonce")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`oauth_consumer_key="consumer"`,
		`oauth_nonce="nonce"`,
		`oauth_signature_method="HMAC-SHA1"`,
		`oauth_timestamp="1234567890"`,
		`oauth_version="1.0"`,
		`oauth_signature=`,
	} {
		if !strings.Contains(header, want) {
			t.Fatalf("header missing %s: %s", want, header)
		}
	}
}

func TestParseOAuthResponse(t *testing.T) {
	got := ParseOAuthResponse("oauth_token=abc123&oauth_token_secret=xyz789&mfa_token=mfa%20456")
	if got["oauth_token"] != "abc123" || got["oauth_token_secret"] != "xyz789" || got["mfa_token"] != "mfa 456" {
		t.Fatalf("unexpected response: %#v", got)
	}
}

func TestNormalizedParameterStringPreservesDuplicateParameters(t *testing.T) {
	got := normalizedParameterString([]oauthParam{
		{key: "a", value: "2"},
		{key: "oauth_nonce", value: "nonce"},
		{key: "a", value: "1"},
		{key: "space", value: "a b"},
	})
	want := "a=1&a=2&oauth_nonce=nonce&space=a%20b"
	if got != want {
		t.Fatalf("normalized params = %q, want %q", got, want)
	}
}
