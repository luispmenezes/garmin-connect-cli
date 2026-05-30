package garmin

import (
	"encoding/json"
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

func TestClientJSONWriteMethods(t *testing.T) {
	token := auth.OAuth2Token{TokenType: "Bearer", AccessToken: "token"}
	tests := []struct {
		name        string
		call        func(*Client) (json.RawMessage, error)
		wantMethod  string
		wantPath    string
		wantBody    string
		wantContent string
	}{
		{
			name: "post raw json",
			call: func(c *Client) (json.RawMessage, error) {
				return c.PostRawJSON(token, "/raw", []byte(`{"raw":true}`))
			},
			wantMethod:  http.MethodPost,
			wantPath:    "/raw",
			wantBody:    `{"raw":true}`,
			wantContent: "application/json",
		},
		{
			name: "put json",
			call: func(c *Client) (json.RawMessage, error) {
				return c.PutJSON(token, "/put", map[string]any{"ok": true})
			},
			wantMethod:  http.MethodPut,
			wantPath:    "/put",
			wantBody:    `{"ok":true}`,
			wantContent: "application/json",
		},
		{
			name: "post string json",
			call: func(c *Client) (json.RawMessage, error) {
				return c.PostJSON(token, "/schedule", "2026-06-01")
			},
			wantMethod:  http.MethodPost,
			wantPath:    "/schedule",
			wantBody:    `"2026-06-01"`,
			wantContent: "application/json",
		},
		{
			name: "delete json",
			call: func(c *Client) (json.RawMessage, error) {
				return c.DeleteJSON(token, "/delete")
			},
			wantMethod: http.MethodDelete,
			wantPath:   "/delete",
			wantBody:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				BaseURL: "https://example.test",
				http: &http.Client{Transport: roundTripper(func(r *http.Request) (*http.Response, error) {
					if r.Method != tt.wantMethod {
						t.Fatalf("method = %s, want %s", r.Method, tt.wantMethod)
					}
					if r.URL.Path != tt.wantPath {
						t.Fatalf("path = %s, want %s", r.URL.Path, tt.wantPath)
					}
					if r.Header.Get("Content-Type") != tt.wantContent {
						t.Fatalf("content type = %q, want %q", r.Header.Get("Content-Type"), tt.wantContent)
					}
					var body []byte
					if r.Body != nil {
						var err error
						body, err = io.ReadAll(r.Body)
						if err != nil {
							t.Fatal(err)
						}
					}
					if string(body) != tt.wantBody {
						t.Fatalf("body = %q, want %q", body, tt.wantBody)
					}
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
						Header:     make(http.Header),
						Request:    r,
					}, nil
				})},
			}
			data, err := tt.call(client)
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != `{"ok":true}` {
				t.Fatalf("unexpected response: %s", data)
			}
		})
	}
}

type roundTripper func(*http.Request) (*http.Response, error)

func (r roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return r(req)
}
