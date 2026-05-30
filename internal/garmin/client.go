package garmin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/luispmenezes/garmin-connect-cli/internal/auth"
)

const apiUserAgent = "GCM-iOS-5.7.2.1"

type Client struct {
	http    *http.Client
	BaseURL string
}

func NewClient() *Client {
	return &Client{http: &http.Client{Timeout: 30 * time.Second}, BaseURL: "https://connectapi.garmin.com"}
}

func NewClientWithBaseURL(baseURL string) *Client {
	return &Client{http: &http.Client{Timeout: 30 * time.Second}, BaseURL: baseURL}
}

func (c *Client) GetJSON(token auth.OAuth2Token, path string) (json.RawMessage, error) {
	resp, err := c.do(token, http.MethodGet, path, nil, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func (c *Client) PostJSON(token auth.OAuth2Token, path string, body any) (json.RawMessage, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	resp, err := c.do(token, http.MethodPost, path, bytes.NewReader(data), "application/json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func (c *Client) Download(token auth.OAuth2Token, path string) ([]byte, error) {
	resp, err := c.do(token, http.MethodGet, path, nil, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func (c *Client) Upload(token auth.OAuth2Token, path, file string) (json.RawMessage, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filepath.Base(file))
	if err != nil {
		return nil, err
	}
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(part, f); err != nil {
		f.Close()
		return nil, err
	}
	if err := f.Close(); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	resp, err := c.do(token, http.MethodPost, path, &body, writer.FormDataContentType())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func (c *Client) do(token auth.OAuth2Token, method, path string, body io.Reader, contentType string) (*http.Response, error) {
	u, err := c.url(path)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(method, u, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", apiUserAgent)
	req.Header.Set("Authorization", token.AuthorizationHeader())
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		return resp, nil
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return nil, fmt.Errorf("not authenticated: Garmin returned HTTP 401")
	case http.StatusNotFound:
		return nil, fmt.Errorf("not found: %s", u)
	case http.StatusTooManyRequests:
		return nil, fmt.Errorf("rate limited by Garmin: HTTP 429")
	default:
		return nil, fmt.Errorf("Garmin API HTTP %d: %s", resp.StatusCode, truncate(data, 400))
	}
}

func (c *Client) url(path string) (string, error) {
	if _, err := url.Parse(c.BaseURL); err != nil {
		return "", err
	}
	if path == "" || path[0] != '/' {
		return "", fmt.Errorf("API path must start with /: %q", path)
	}
	return c.BaseURL + path, nil
}

func truncate(data []byte, max int) string {
	s := string(data)
	if len(s) > max {
		return s[:max]
	}
	return s
}
