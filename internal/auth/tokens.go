package auth

import "time"

type OAuth1Token struct {
	OAuthToken             string     `json:"oauth_token"`
	OAuthTokenSecret       string     `json:"oauth_token_secret"`
	MFAToken               string     `json:"mfa_token,omitempty"`
	MFAExpirationTimestamp *time.Time `json:"mfa_expiration_timestamp,omitempty"`
	Domain                 string     `json:"domain"`
}

func NewOAuth1Token(token, secret string) OAuth1Token {
	return OAuth1Token{OAuthToken: token, OAuthTokenSecret: secret, Domain: "garmin.com"}
}

type OAuth2Token struct {
	Scope                 string `json:"scope"`
	JTI                   string `json:"jti"`
	TokenType             string `json:"token_type"`
	AccessToken           string `json:"access_token"`
	RefreshToken          string `json:"refresh_token"`
	ExpiresIn             int64  `json:"expires_in"`
	ExpiresAt             int64  `json:"expires_at"`
	RefreshTokenExpiresIn int64  `json:"refresh_token_expires_in"`
	RefreshTokenExpiresAt int64  `json:"refresh_token_expires_at"`
}

func (t OAuth2Token) Expired(now time.Time) bool {
	return t.ExpiresAt <= now.Unix()
}

func (t OAuth2Token) RefreshExpired(now time.Time) bool {
	return t.RefreshTokenExpiresAt <= now.Unix()
}

func (t OAuth2Token) AuthorizationHeader() string {
	tokenType := t.TokenType
	if tokenType == "" {
		tokenType = "Bearer"
	}
	return tokenType + " " + t.AccessToken
}
