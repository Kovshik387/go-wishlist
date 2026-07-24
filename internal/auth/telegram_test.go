package auth

import (
	"net/url"
	"strconv"
	"testing"
	"time"
)

func TestTelegramValidator(t *testing.T) {
	now := time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC)
	token := "123456:telegram-test-token"
	base := url.Values{
		"auth_date": {strconv.FormatInt(now.Unix(), 10)},
		"query_id":  {"AAExample"},
		"user":      {`{"id":922337203685477000,"first_name":"Аня","username":"anya"}`},
	}

	t.Run("valid", func(t *testing.T) {
		raw := SignInitDataForTest(token, base)
		result, err := (TelegramValidator{BotToken: token, MaxAge: time.Hour, Now: func() time.Time { return now }}).Validate(raw)
		if err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
		if result.User.ID != 922337203685477000 {
			t.Fatalf("telegram id truncated: %d", result.User.ID)
		}
	})

	t.Run("wrong signature", func(t *testing.T) {
		raw := SignInitDataForTest("wrong-token", base)
		_, err := (TelegramValidator{BotToken: token, MaxAge: time.Hour, Now: func() time.Time { return now }}).Validate(raw)
		if err != ErrInvalidInitData {
			t.Fatalf("error = %v, want %v", err, ErrInvalidInitData)
		}
	})

	t.Run("expired", func(t *testing.T) {
		expired := cloneValues(base)
		expired.Set("auth_date", strconv.FormatInt(now.Add(-2*time.Hour).Unix(), 10))
		raw := SignInitDataForTest(token, expired)
		_, err := (TelegramValidator{BotToken: token, MaxAge: time.Hour, Now: func() time.Time { return now }}).Validate(raw)
		if err != ErrExpiredInitData {
			t.Fatalf("error = %v, want %v", err, ErrExpiredInitData)
		}
	})
}
