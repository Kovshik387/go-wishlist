package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/example/wishtrack/internal/store"
)

var (
	ErrInvalidInitData = errors.New("invalid Telegram init data")
	ErrExpiredInitData = errors.New("Telegram init data has expired")
)

type TelegramValidator struct {
	BotToken string
	MaxAge   time.Duration
	Now      func() time.Time
}

type TelegramAuthData struct {
	User      store.TelegramUser
	AuthDate  time.Time
	QueryID   string
	StartParam string
}

func (v TelegramValidator) Validate(raw string) (TelegramAuthData, error) {
	if v.BotToken == "" || raw == "" {
		return TelegramAuthData{}, ErrInvalidInitData
	}
	values, err := url.ParseQuery(raw)
	if err != nil {
		return TelegramAuthData{}, ErrInvalidInitData
	}
	providedHash := values.Get("hash")
	if len(providedHash) != sha256.Size*2 {
		return TelegramAuthData{}, ErrInvalidInitData
	}
	values.Del("hash")

	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	pairs := make([]string, 0, len(keys))
	for _, key := range keys {
		pairs = append(pairs, key+"="+values.Get(key))
	}
	dataCheckString := strings.Join(pairs, "\n")

	secretMAC := hmac.New(sha256.New, []byte("WebAppData"))
	_, _ = secretMAC.Write([]byte(v.BotToken))
	checkMAC := hmac.New(sha256.New, secretMAC.Sum(nil))
	_, _ = checkMAC.Write([]byte(dataCheckString))
	expected := checkMAC.Sum(nil)
	actual, err := hex.DecodeString(providedHash)
	if err != nil || !hmac.Equal(expected, actual) {
		return TelegramAuthData{}, ErrInvalidInitData
	}

	authUnix, err := strconv.ParseInt(values.Get("auth_date"), 10, 64)
	if err != nil {
		return TelegramAuthData{}, ErrInvalidInitData
	}
	authDate := time.Unix(authUnix, 0)
	now := time.Now()
	if v.Now != nil {
		now = v.Now()
	}
	maxAge := v.MaxAge
	if maxAge <= 0 {
		maxAge = 24 * time.Hour
	}
	if authDate.After(now.Add(2*time.Minute)) || now.Sub(authDate) > maxAge {
		return TelegramAuthData{}, ErrExpiredInitData
	}

	var user store.TelegramUser
	if err := json.Unmarshal([]byte(values.Get("user")), &user); err != nil || user.ID == 0 {
		return TelegramAuthData{}, ErrInvalidInitData
	}
	return TelegramAuthData{
		User: user, AuthDate: authDate, QueryID: values.Get("query_id"),
		StartParam: values.Get("start_param"),
	}, nil
}

func SignInitDataForTest(botToken string, values url.Values) string {
	values = cloneValues(values)
	values.Del("hash")
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var pairs []string
	for _, key := range keys {
		pairs = append(pairs, fmt.Sprintf("%s=%s", key, values.Get(key)))
	}
	secret := hmac.New(sha256.New, []byte("WebAppData"))
	_, _ = secret.Write([]byte(botToken))
	signature := hmac.New(sha256.New, secret.Sum(nil))
	_, _ = signature.Write([]byte(strings.Join(pairs, "\n")))
	values.Set("hash", hex.EncodeToString(signature.Sum(nil)))
	return values.Encode()
}

func cloneValues(source url.Values) url.Values {
	target := make(url.Values, len(source))
	for key, values := range source {
		target[key] = append([]string(nil), values...)
	}
	return target
}
