package httpapi

import (
	"crypto/subtle"
	"database/sql"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/example/wishtrack/internal/auth"
	"github.com/example/wishtrack/internal/bot"
	"github.com/example/wishtrack/internal/config"
	"github.com/example/wishtrack/internal/preview"
	"github.com/example/wishtrack/internal/store"
	"github.com/go-chi/chi/v5"
)

type Server struct {
	Config   config.Config
	DB       *sql.DB
	Store    *store.Store
	Tokens   auth.TokenManager
	Telegram auth.TelegramValidator
	Preview  preview.Fetcher
	Bot      *bot.Client
	Logger   *slog.Logger
	Limiter  *limiter
}

func New(cfg config.Config, db *sql.DB, dataStore *store.Store, logger *slog.Logger) *Server {
	return &Server{
		Config: cfg,
		DB: db,
		Store: dataStore,
		Tokens: auth.TokenManager{Secret: []byte(cfg.AccessTokenSecret), TTL: cfg.AccessTokenTTL},
		Telegram: auth.TelegramValidator{
			BotToken: cfg.TelegramBotToken, MaxAge: cfg.TelegramAuthMaxAge,
		},
		Preview: preview.Fetcher{},
		Bot: &bot.Client{Token: cfg.TelegramBotToken},
		Logger: logger,
		Limiter: newLimiter(180, time.Minute),
	}
}

func (s *Server) Handler() http.Handler {
	router := chi.NewRouter()
	router.Use(s.middleware)

	router.Get("/healthz", s.health)
	router.Get("/readyz", s.ready)
	router.Get("/api/v1/config", s.brandConfig)
	router.Post("/api/v1/auth/telegram", s.telegramAuth)
	router.Post("/api/v1/auth/refresh", s.refresh)
	router.Post("/api/v1/auth/logout", s.logout)
	if s.Config.Env != "production" {
		router.Post("/api/v1/auth/dev", s.devAuth)
	}
	router.With(s.optionalAuth).Get("/api/v1/public/wishlists/{token}", s.publicWishlist)
	router.Post("/api/v1/telegram/webhook", s.telegramWebhook)

	router.Group(func(api chi.Router) {
		api.Use(s.authRequired)
		api.Get("/api/v1/me", s.me)
		api.Patch("/api/v1/me", s.patchMe)
		api.Delete("/api/v1/me", s.deleteMe)
		api.Get("/api/v1/feed", s.feed)
		api.Get("/api/v1/wishlists", s.listWishlists)
		api.Post("/api/v1/wishlists", s.createWishlist)
		api.Get("/api/v1/wishlists/{id}", s.getWishlist)
		api.Patch("/api/v1/wishlists/{id}", s.updateWishlist)
		api.Delete("/api/v1/wishlists/{id}", s.deleteWishlist)
		api.Post("/api/v1/wishlists/{id}/rotate-link", s.rotateWishlistLink)
		api.Post("/api/v1/wishlists/{id}/wishes", s.createWish)
		api.Get("/api/v1/wishlists/{id}/wishes/{wishID}", s.getWish)
		api.Patch("/api/v1/wishlists/{id}/wishes/{wishID}", s.updateWish)
		api.Delete("/api/v1/wishlists/{id}/wishes/{wishID}", s.deleteWish)
		api.Post("/api/v1/wishes/preview-url", s.previewURL)
		api.Post("/api/v1/users/{id}/follow", s.follow)
		api.Delete("/api/v1/users/{id}/follow", s.unfollow)
		api.Get("/api/v1/users/{id}", s.publicUser)
		api.Patch("/api/v1/users/{id}/notification-settings", s.updateFollowSettings)
		api.Post("/api/v1/wishes/{id}/reservation", s.reserve)
		api.Delete("/api/v1/wishes/{id}/reservation", s.cancelReservation)
		api.Get("/api/v1/reservations", s.reservations)
		api.Get("/api/v1/notification-settings", s.notificationSettings)
		api.Patch("/api/v1/notification-settings", s.updateNotificationSettings)
		api.Post("/api/v1/media", s.uploadMedia)
	})

	if err := os.MkdirAll(s.Config.MediaDir, 0o750); err == nil {
		router.Handle("/media/*", http.StripPrefix("/media/",
			http.FileServer(http.Dir(s.Config.MediaDir))))
	}
	router.NotFound(s.frontend)
	return router
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) ready(w http.ResponseWriter, r *http.Request) {
	if err := s.DB.PingContext(r.Context()); err != nil {
		writeError(w, http.StatusServiceUnavailable, "NOT_READY", "База данных недоступна")
		return
	}
	var version int
	if err := s.DB.QueryRowContext(r.Context(),
		`SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&version); err != nil || version < 1 {
		writeError(w, http.StatusServiceUnavailable, "MIGRATIONS_PENDING", "Миграции не применены")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ready", "migrationVersion": version})
}

func (s *Server) brandConfig(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"name": s.Config.BrandName, "emoji": s.Config.BrandEmoji,
		"primary": s.Config.BrandPrimary, "accent": s.Config.BrandAccent,
		"botUsername": s.Config.TelegramBotUsername,
		"appShortName": s.Config.TelegramAppShortName,
	})
}

func (s *Server) frontend(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Маршрут не найден")
		return
	}
	clean := filepath.Clean(strings.TrimPrefix(r.URL.Path, "/"))
	if clean == "." {
		clean = "index.html"
	}
	path := filepath.Join(s.Config.WebDir, clean)
	webRoot, _ := filepath.Abs(s.Config.WebDir)
	resolved, _ := filepath.Abs(path)
	if strings.HasPrefix(resolved, webRoot+string(os.PathSeparator)) {
		if info, err := os.Stat(resolved); err == nil && !info.IsDir() {
			http.ServeFile(w, r, resolved)
			return
		}
	}
	indexPath := filepath.Join(s.Config.WebDir, "index.html")
	if _, err := os.Stat(indexPath); err == nil {
		http.ServeFile(w, r, indexPath)
		return
	}
	writeError(w, http.StatusNotFound, "FRONTEND_NOT_BUILT", "Frontend ещё не собран")
}

func secureEqual(left, right string) bool {
	return len(left) == len(right) &&
		subtle.ConstantTimeCompare([]byte(left), []byte(right)) == 1
}

var _ fs.FileInfo
var _ = errors.Is
