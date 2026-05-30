package garmin

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/luispmenezes/garmin-connect-cli/internal/auth"
)

func TestClientGetJSONAndStatuses(t *testing.T) {
	client := &Client{
		BaseURL: "https://example.test",
		http: &http.Client{Transport: roundTripper(func(r *http.Request) (*http.Response, error) {
			if r.Header.Get("Authorization") != "Bearer token" {
				t.Fatalf("missing auth header: %s", r.Header.Get("Authorization"))
			}
			status := http.StatusOK
			body := `{"ok":true}`
			switch r.URL.Path {
			case "/ok":
			case "/rate":
				status = http.StatusTooManyRequests
				body = ""
			default:
				status = http.StatusNotFound
				body = ""
			}
			return &http.Response{
				StatusCode: status,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     make(http.Header),
				Request:    r,
			}, nil
		})},
	}

	token := auth.OAuth2Token{TokenType: "Bearer", AccessToken: "token"}
	data, err := client.GetJSON(token, "/ok")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"ok":true}` {
		t.Fatalf("unexpected body: %s", data)
	}
	if _, err := client.GetJSON(token, "/rate"); err == nil || !strings.Contains(err.Error(), "rate limited") {
		t.Fatalf("expected rate limit error, got %v", err)
	}
	if _, err := client.GetJSON(token, "/missing"); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not found error, got %v", err)
	}
}

type roundTripper func(*http.Request) (*http.Response, error)

func (r roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return r(req)
}
