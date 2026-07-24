package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Env                  string
	Mode                 string
	HTTPAddr             string
	PublicURL            string
	FrontendOrigin       string
	DatabasePath         string
	MigrationsDir        string
	MediaDir             string
	WebDir               string
	BrandName            string
	BrandEmoji           string
	BrandPrimary         string
	BrandAccent          string
	TelegramBotToken     string
	TelegramBotUsername  string
	TelegramAppShortName string
	TelegramWebhookSecret string
	AccessTokenSecret    string
	AccessTokenTTL       time.Duration
	RefreshTokenTTL      time.Duration
	TelegramAuthMaxAge   time.Duration
	NotificationDigest   time.Duration
	WorkerPollInterval   time.Duration
	DevTelegramID        int64
}

func Load() (Config, error) {
	cfg := Config{
		Env:                   env("APP_ENV", "development"),
		Mode:                  env("APP_MODE", "api"),
		HTTPAddr:              env("HTTP_ADDR", ":8080"),
		PublicURL:             strings.TrimRight(env("PUBLIC_URL", "http://localhost:8080"), "/"),
		FrontendOrigin:        strings.TrimRight(env("FRONTEND_ORIGIN", "http://localhost:8080"), "/"),
		DatabasePath:          env("DATABASE_PATH", "./data/wishtrack.db"),
		MigrationsDir:         env("MIGRATIONS_DIR", "./migrations"),
		MediaDir:              env("MEDIA_DIR", "./media"),
		WebDir:                env("WEB_DIR", "./web/dist"),
		BrandName:             env("BRAND_NAME", "WishTrack"),
		BrandEmoji:            env("BRAND_EMOJI", "🎁"),
		BrandPrimary:          env("BRAND_PRIMARY", "#2764ff"),
		BrandAccent:           env("BRAND_ACCENT", "#ff7c66"),
		TelegramBotToken:      os.Getenv("TELEGRAM_BOT_TOKEN"),
		TelegramBotUsername:   strings.TrimPrefix(env("TELEGRAM_BOT_USERNAME", "wishtrack_bot"), "@"),
		TelegramAppShortName:  env("TELEGRAM_APP_SHORT_NAME", "app"),
		TelegramWebhookSecret: os.Getenv("TELEGRAM_WEBHOOK_SECRET"),
		AccessTokenSecret:     os.Getenv("ACCESS_TOKEN_SECRET"),
		AccessTokenTTL:        duration("ACCESS_TOKEN_TTL", 15*time.Minute),
		RefreshTokenTTL:       duration("REFRESH_TOKEN_TTL", 30*24*time.Hour),
		TelegramAuthMaxAge:    duration("TELEGRAM_AUTH_MAX_AGE", 24*time.Hour),
		NotificationDigest:    duration("NOTIFICATION_DIGEST_WINDOW", 5*time.Minute),
		WorkerPollInterval:    duration("WORKER_POLL_INTERVAL", 2*time.Second),
		DevTelegramID:         int64Value("DEV_TELEGRAM_ID", 900000001),
	}

	if cfg.Env == "production" {
		var missing []string
		for name, value := range map[string]string{
			"PUBLIC_URL": cfg.PublicURL,
			"FRONTEND_ORIGIN": cfg.FrontendOrigin,
			"TELEGRAM_BOT_TOKEN": cfg.TelegramBotToken,
			"TELEGRAM_WEBHOOK_SECRET": cfg.TelegramWebhookSecret,
			"ACCESS_TOKEN_SECRET": cfg.AccessTokenSecret,
		} {
			if value == "" {
				missing = append(missing, name)
			}
		}
		if len(missing) > 0 {
			return Config{}, fmt.Errorf("missing production configuration: %s", strings.Join(missing, ", "))
		}
		if !strings.HasPrefix(cfg.PublicURL, "https://") || !strings.HasPrefix(cfg.FrontendOrigin, "https://") {
			return Config{}, errors.New("PUBLIC_URL and FRONTEND_ORIGIN must use https in production")
		}
	} else if cfg.AccessTokenSecret == "" {
		cfg.AccessTokenSecret = "development-only-secret-change-me"
	}

	switch cfg.Mode {
	case "api", "worker", "migrate":
	default:
		return Config{}, fmt.Errorf("unsupported APP_MODE %q", cfg.Mode)
	}
	return cfg, nil
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func duration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func int64Value(key string, fallback int64) int64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fallback
	}
	return parsed
}
