package auth

import (
	"errors"
	"io"
	"net/http"
	"strings"
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

func TestStartSessionReportsFailure(t *testing.T) {
	client := &SSOClient{http: &http.Client{Transport: roundTripper(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusForbidden,
			Body:       io.NopCloser(strings.NewReader("blocked")),
			Header:     make(http.Header),
			Request:    r,
		}, nil
	})}}
	req, _ := http.NewRequest(http.MethodGet, "https://sso.garmin.com/mobile/sso/en/sign-in", nil)
	err := client.startSession(req)
	if err == nil || !strings.Contains(err.Error(), "HTTP 403") {
		t.Fatalf("expected HTTP 403 error, got %v", err)
	}
}

func TestStartSessionReportsNetworkError(t *testing.T) {
	client := &SSOClient{http: &http.Client{Transport: roundTripper(func(r *http.Request) (*http.Response, error) {
		return nil, errors.New("network down")
	})}}
	req, _ := http.NewRequest(http.MethodGet, "https://sso.garmin.com/mobile/sso/en/sign-in", nil)
	err := client.startSession(req)
	if err == nil || !strings.Contains(err.Error(), "network down") {
		t.Fatalf("expected network error, got %v", err)
	}
}

type roundTripper func(*http.Request) (*http.Response, error)

func (r roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return r(req)
}
