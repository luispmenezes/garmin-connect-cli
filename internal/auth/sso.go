package auth

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

const (
	defaultDomain    = "garmin.com"
	clientID         = "GCM_ANDROID_DARK"
	mobileUserAgent  = "com.garmin.android.apps.connectmobile"
	ssoUserAgent     = "Mozilla/5.0 (iPhone; CPU iPhone OS 18_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148"
	consumerCredsURL = "https://thegarth.s3.amazonaws.com/oauth_consumer.json"
)

var ErrMFARequired = errors.New("MFA required")

type MFAFunc func(method string) (string, error)

type SSOClient struct {
	http     *http.Client
	domain   string
	consumer string
}

func NewSSOClient() (*SSOClient, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	return &SSOClient{
		http:     &http.Client{Jar: jar, Timeout: 30 * time.Second},
		domain:   defaultDomain,
		consumer: consumerCredsURL,
	}, nil
}

func (c *SSOClient) Login(email, password string, mfa MFAFunc) (OAuth1Token, OAuth2Token, error) {
	serviceURL := fmt.Sprintf("https://mobile.integration.%s/gcm/android", c.domain)
	query := url.Values{"clientId": {clientID}, "locale": {"en-US"}, "service": {serviceURL}}
	signInURL := fmt.Sprintf("https://sso.%s/mobile/sso/en/sign-in?clientId=%s", c.domain, url.QueryEscape(clientID))
	req, _ := http.NewRequest(http.MethodGet, signInURL, nil)
	addSSOPageHeaders(req)
	if err := c.startSession(req); err != nil {
		return OAuth1Token{}, OAuth2Token{}, err
	}

	loginBody := map[string]any{
		"username":     email,
		"password":     password,
		"rememberMe":   false,
		"captchaToken": "",
	}
	var resp ssoResponse
	if err := c.postJSON(fmt.Sprintf("https://sso.%s/mobile/api/login?%s", c.domain, query.Encode()), loginBody, &resp); err != nil {
		return OAuth1Token{}, OAuth2Token{}, err
	}

	statusType := resp.Type()
	ticket := resp.ServiceTicketID
	if statusType == "MFA_REQUIRED" {
		if mfa == nil {
			return OAuth1Token{}, OAuth2Token{}, ErrMFARequired
		}
		method := "email"
		if resp.CustomerMFAInfo.MFALastMethodUsed != "" {
			method = resp.CustomerMFAInfo.MFALastMethodUsed
		}
		code, err := mfa(method)
		if err != nil {
			return OAuth1Token{}, OAuth2Token{}, err
		}
		ticket, err = c.submitMFA(query, method, code)
		if err != nil {
			return OAuth1Token{}, OAuth2Token{}, err
		}
	} else if statusType != "SUCCESSFUL" {
		msg := resp.ResponseStatus.Message
		if msg == "" {
			msg = statusType
		}
		return OAuth1Token{}, OAuth2Token{}, fmt.Errorf("SSO login failed: %s", msg)
	}
	if ticket == "" {
		return OAuth1Token{}, OAuth2Token{}, errors.New("SSO login response did not include service ticket")
	}
	return c.completeLogin(ticket)
}

func (c *SSOClient) startSession(req *http.Request) error {
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("start SSO session: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("start SSO session: HTTP %d: %s", resp.StatusCode, truncateBody(data))
	}
	return nil
}

func (c *SSOClient) RefreshOAuth2(oauth1 OAuth1Token) (OAuth2Token, error) {
	return c.exchangeOAuth1ForOAuth2(oauth1, false)
}

func (c *SSOClient) submitMFA(query url.Values, method, code string) (string, error) {
	body := map[string]any{
		"mfaMethod":           method,
		"mfaVerificationCode": code,
		"rememberMyBrowser":   false,
		"reconsentList":       []string{},
		"mfaSetup":            false,
	}
	var resp ssoResponse
	if err := c.postJSON(fmt.Sprintf("https://sso.%s/mobile/api/mfa/verifyCode?%s", c.domain, query.Encode()), body, &resp); err != nil {
		return "", err
	}
	if resp.Type() != "SUCCESSFUL" {
		return "", fmt.Errorf("MFA verification failed: %s", resp.ResponseStatus.Message)
	}
	if resp.ServiceTicketID == "" {
		return "", errors.New("MFA response did not include service ticket")
	}
	return resp.ServiceTicketID, nil
}

func (c *SSOClient) completeLogin(ticket string) (OAuth1Token, OAuth2Token, error) {
	portalURL := fmt.Sprintf("https://sso.%s/portal/sso/embed", c.domain)
	req, _ := http.NewRequest(http.MethodGet, portalURL, nil)
	addSSOPageHeaders(req)
	_, _ = c.http.Do(req)
	oauth1, err := c.oauth1FromTicket(ticket)
	if err != nil {
		return OAuth1Token{}, OAuth2Token{}, err
	}
	oauth2, err := c.exchangeOAuth1ForOAuth2(oauth1, true)
	if err != nil {
		return OAuth1Token{}, OAuth2Token{}, err
	}
	return oauth1, oauth2, nil
}

func (c *SSOClient) oauth1FromTicket(ticket string) (OAuth1Token, error) {
	consumer, err := c.fetchConsumer()
	if err != nil {
		return OAuth1Token{}, err
	}
	loginURL := fmt.Sprintf("https://mobile.integration.%s/gcm/android", c.domain)
	u := fmt.Sprintf("https://connectapi.%s/oauth-service/oauth/preauthorized?ticket=%s&login-url=%s&accepts-mfa-tokens=true", c.domain, ticket, loginURL)
	header, err := (OAuth1Signer{Consumer: consumer}).Sign(http.MethodGet, u, nil)
	if err != nil {
		return OAuth1Token{}, err
	}
	req, _ := http.NewRequest(http.MethodGet, u, nil)
	req.Header.Set("User-Agent", mobileUserAgent)
	req.Header.Set("Authorization", header)
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return OAuth1Token{}, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return OAuth1Token{}, fmt.Errorf("OAuth1 exchange failed: HTTP %d: %s", resp.StatusCode, truncateBody(data))
	}
	params := ParseOAuthResponse(string(data))
	token, secret := params["oauth_token"], params["oauth_token_secret"]
	if token == "" || secret == "" {
		return OAuth1Token{}, errors.New("OAuth1 exchange response missing token or secret")
	}
	oauth1 := NewOAuth1Token(token, secret)
	oauth1.MFAToken = params["mfa_token"]
	return oauth1, nil
}

func (c *SSOClient) exchangeOAuth1ForOAuth2(oauth1 OAuth1Token, login bool) (OAuth2Token, error) {
	consumer, err := c.fetchConsumer()
	if err != nil {
		return OAuth2Token{}, err
	}
	u := fmt.Sprintf("https://connectapi.%s/oauth-service/oauth/exchange/user/2.0", c.domain)
	form := map[string]string{}
	if login {
		form["audience"] = "GARMIN_CONNECT_MOBILE_ANDROID_DI"
	}
	if oauth1.MFAToken != "" {
		form["mfa_token"] = oauth1.MFAToken
	}
	header, err := (OAuth1Signer{
		Consumer: consumer,
		Token:    &OAuthToken{Token: oauth1.OAuthToken, Secret: oauth1.OAuthTokenSecret},
	}).Sign(http.MethodPost, u, form)
	if err != nil {
		return OAuth2Token{}, err
	}
	values := url.Values{}
	for k, v := range form {
		values.Set(k, v)
	}
	req, _ := http.NewRequest(http.MethodPost, u, strings.NewReader(values.Encode()))
	req.Header.Set("User-Agent", mobileUserAgent)
	req.Header.Set("Authorization", header)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return OAuth2Token{}, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return OAuth2Token{}, fmt.Errorf("OAuth2 exchange failed: HTTP %d: %s", resp.StatusCode, truncateBody(data))
	}
	var token OAuth2Token
	if err := json.Unmarshal(data, &token); err != nil {
		return OAuth2Token{}, err
	}
	now := time.Now().Unix()
	token.ExpiresAt = now + token.ExpiresIn
	token.RefreshTokenExpiresAt = now + token.RefreshTokenExpiresIn
	return token, nil
}

func (c *SSOClient) fetchConsumer() (OAuthConsumer, error) {
	resp, err := c.http.Get(c.consumer)
	if err != nil {
		return OAuthConsumer{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return OAuthConsumer{}, fmt.Errorf("fetch OAuth consumer failed: HTTP %d", resp.StatusCode)
	}
	var consumer OAuthConsumer
	return consumer, json.NewDecoder(resp.Body).Decode(&consumer)
}

func (c *SSOClient) postJSON(rawURL string, body any, out any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, rawURL, bytes.NewReader(data))
	if err != nil {
		return err
	}
	addSSOAPIHeaders(req)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respData, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusTooManyRequests {
		return fmt.Errorf("rate limited by Garmin: HTTP 429: %s", truncateBody(respData))
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("SSO HTTP %d: %s", resp.StatusCode, truncateBody(respData))
	}
	if err := json.Unmarshal(respData, out); err != nil {
		return fmt.Errorf("parse SSO response: %w", err)
	}
	return nil
}

type ssoResponse struct {
	ResponseStatus struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"responseStatus"`
	ServiceTicketID string `json:"serviceTicketId"`
	CustomerMFAInfo struct {
		MFALastMethodUsed string `json:"mfaLastMethodUsed"`
	} `json:"customerMfaInfo"`
}

func (r ssoResponse) Type() string {
	if r.ResponseStatus.Type == "" {
		return "UNKNOWN"
	}
	return r.ResponseStatus.Type
}

func addSSOPageHeaders(req *http.Request) {
	req.Header.Set("User-Agent", ssoUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Dest", "document")
}

func addSSOAPIHeaders(req *http.Request) {
	req.Header.Set("User-Agent", mobileUserAgent)
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://sso.garmin.com")
	req.Header.Set("Referer", "https://sso.garmin.com/mobile/sso/en/sign-in")
}

func truncateBody(data []byte) string {
	s := string(data)
	if len(s) > 400 {
		return s[:400]
	}
	return s
}
