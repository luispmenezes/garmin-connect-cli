package auth

import (
	"net/http"
	"testing"
)

func TestSSOConstantsMatchMobileFlow(t *testing.T) {
	if clientID != "GCM_ANDROID_DARK" {
		t.Fatalf("clientID = %q", clientID)
	}
	if mobileUserAgent != "com.garmin.android.apps.connectmobile" {
		t.Fatalf("mobile user agent = %q", mobileUserAgent)
	}
	req, _ := http.NewRequest(http.MethodGet, "https://sso.garmin.com/mobile/sso/en/sign-in", nil)
	addSSOPageHeaders(req)
	if req.Header.Get("User-Agent") == mobileUserAgent {
		t.Fatal("SSO page should use browser-like user agent, not mobile app user agent")
	}
}
