package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

var ErrInvalidToken = errors.New("invalid access token")

type TokenManager struct {
	Secret []byte
	TTL    time.Duration
	Now    func() time.Time
}

type claims struct {
	Subject string `json:"sub"`
	Issued  int64  `json:"iat"`
	Expires int64  `json:"exp"`
	Issuer  string `json:"iss"`
}

func (m TokenManager) Issue(userID string) (string, time.Time, error) {
	now := time.Now().UTC()
	if m.Now != nil {
		now = m.Now().UTC()
	}
	ttl := m.TTL
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	expires := now.Add(ttl)
	header, _ := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT"})
	payload, _ := json.Marshal(claims{
		Subject: userID, Issued: now.Unix(), Expires: expires.Unix(), Issuer: "wishtrack",
	})
	unsigned := encode(header) + "." + encode(payload)
	signature := m.sign(unsigned)
	return unsigned + "." + encode(signature), expires, nil
}

func (m TokenManager) Parse(token string) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 || len(m.Secret) < 16 {
		return "", ErrInvalidToken
	}
	expected := m.sign(parts[0] + "." + parts[1])
	actual, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !hmac.Equal(expected, actual) {
		return "", ErrInvalidToken
	}
	body, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", ErrInvalidToken
	}
	var parsed claims
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", ErrInvalidToken
	}
	now := time.Now()
	if m.Now != nil {
		now = m.Now()
	}
	if parsed.Issuer != "wishtrack" || parsed.Subject == "" || parsed.Expires <= now.Unix() {
		return "", ErrInvalidToken
	}
	return parsed.Subject, nil
}

func (m TokenManager) sign(value string) []byte {
	mac := hmac.New(sha256.New, m.Secret)
	_, _ = mac.Write([]byte(value))
	return mac.Sum(nil)
}

func encode(value []byte) string {
	return base64.RawURLEncoding.EncodeToString(value)
}
