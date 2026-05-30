package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"net/url"
	"sort"
	"strings"
	"time"
)

type OAuthConsumer struct {
	Key    string `json:"consumer_key"`
	Secret string `json:"consumer_secret"`
}

type OAuthToken struct {
	Token  string
	Secret string
}

type OAuth1Signer struct {
	Consumer OAuthConsumer
	Token    *OAuthToken
}

type oauthParam struct {
	key   string
	value string
}

func (s OAuth1Signer) Sign(method, rawURL string, extra map[string]string) (string, error) {
	nonce, err := nonce()
	if err != nil {
		return "", err
	}
	return s.SignWithTimestampNonce(method, rawURL, extra, time.Now().Unix(), nonce)
}

func (s OAuth1Signer) SignWithTimestampNonce(method, rawURL string, extra map[string]string, timestamp int64, nonce string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	baseURL := u.Scheme + "://" + u.Host + u.EscapedPath()
	oauth := map[string]string{
		"oauth_consumer_key":     s.Consumer.Key,
		"oauth_nonce":            nonce,
		"oauth_signature_method": "HMAC-SHA1",
		"oauth_timestamp":        strconvFormatInt(timestamp),
		"oauth_version":          "1.0",
	}
	if s.Token != nil {
		oauth["oauth_token"] = s.Token.Token
	}
	params := make([]oauthParam, 0, len(oauth)+len(extra)+len(u.Query()))
	for k, v := range oauth {
		params = append(params, oauthParam{key: k, value: v})
	}
	for k, values := range u.Query() {
		for _, value := range values {
			params = append(params, oauthParam{key: k, value: value})
		}
	}
	for k, v := range extra {
		params = append(params, oauthParam{key: k, value: v})
	}
	oauth["oauth_signature"] = s.signature(method, baseURL, params)
	keys := make([]string, 0, len(oauth))
	for k := range oauth {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, percentEncode(k)+`="`+percentEncode(oauth[k])+`"`)
	}
	return "OAuth " + strings.Join(parts, ", "), nil
}

func (s OAuth1Signer) signature(method, baseURL string, params []oauthParam) string {
	paramString := normalizedParameterString(params)
	base := strings.ToUpper(method) + "&" + percentEncode(baseURL) + "&" + percentEncode(paramString)
	tokenSecret := ""
	if s.Token != nil {
		tokenSecret = s.Token.Secret
	}
	key := percentEncode(s.Consumer.Secret) + "&" + percentEncode(tokenSecret)
	mac := hmac.New(sha1.New, []byte(key))
	mac.Write([]byte(base))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func normalizedParameterString(params []oauthParam) string {
	encoded := make([]oauthParam, 0, len(params))
	for _, param := range params {
		encoded = append(encoded, oauthParam{key: percentEncode(param.key), value: percentEncode(param.value)})
	}
	sort.Slice(encoded, func(i, j int) bool {
		if encoded[i].key == encoded[j].key {
			return encoded[i].value < encoded[j].value
		}
		return encoded[i].key < encoded[j].key
	})
	pairs := make([]string, 0, len(encoded))
	for _, param := range encoded {
		pairs = append(pairs, param.key+"="+param.value)
	}
	return strings.Join(pairs, "&")
}

func ParseOAuthResponse(body string) map[string]string {
	values, err := url.ParseQuery(body)
	if err != nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(values))
	for k, v := range values {
		if len(v) > 0 {
			out[k] = v[0]
		}
	}
	return out
}

func percentEncode(s string) string {
	return strings.ReplaceAll(url.QueryEscape(s), "+", "%20")
}

func nonce() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func strconvFormatInt(n int64) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
