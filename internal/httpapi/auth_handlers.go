package httpapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/example/wishtrack/internal/auth"
	"github.com/example/wishtrack/internal/platform"
	"github.com/example/wishtrack/internal/store"
)

type telegramAuthRequest struct {
	InitData string `json:"initData"`
}

func (s *Server) telegramAuth(w http.ResponseWriter, r *http.Request) {
	var request telegramAuthRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	authData, err := s.Telegram.Validate(request.InitData)
	if err != nil {
		code := "INVALID_TELEGRAM_AUTH"
		message := "Не удалось проверить данные Telegram"
		if errors.Is(err, auth.ErrExpiredInitData) {
			code = "TELEGRAM_AUTH_EXPIRED"
			message = "Сессия Telegram устарела. Откройте приложение заново"
		}
		writeError(w, http.StatusUnauthorized, code, message)
		return
	}
	telegramUser := authData.User
	remoteAvatarURL := telegramUser.PhotoURL
	telegramUser.PhotoURL = ""
	user, err := s.Store.UpsertTelegramUser(r.Context(), telegramUser)
	if err != nil {
		s.Logger.Error("upsert authenticated user", "error", err)
		writeStoreError(w, err)
		return
	}
	if remoteAvatarURL == "" && isRemoteImageURL(user.AvatarURL) {
		remoteAvatarURL = user.AvatarURL
	}
	if remoteAvatarURL != "" {
		avatarContext, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		localAvatarURL, avatarErr := s.cacheTelegramAvatar(avatarContext, user.ID, remoteAvatarURL)
		cancel()
		if avatarErr != nil {
			s.Logger.Warn("cache telegram avatar", "user_id", user.ID, "error", avatarErr)
		} else if avatarErr = s.Store.UpdateUserAvatar(r.Context(), user.ID, localAvatarURL); avatarErr != nil {
			s.Logger.Warn("save telegram avatar", "user_id", user.ID, "error", avatarErr)
		} else {
			user.AvatarURL = localAvatarURL
		}
	}
	s.issueSession(w, r, user)
}

func (s *Server) devAuth(w http.ResponseWriter, r *http.Request) {
	if s.Config.Env == "production" {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Маршрут не найден")
		return
	}
	var request struct {
		DisplayName string `json:"displayName"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	if request.DisplayName == "" {
		request.DisplayName = "Аня"
	}
	user, err := s.Store.UpsertTelegramUser(r.Context(), store.TelegramUser{
		ID: s.Config.DevTelegramID, FirstName: request.DisplayName, Username: "dev_friend",
		LanguageCode: "ru",
	})
	if err != nil {
		writeStoreError(w, err)
		return
	}
	s.issueSession(w, r, user)
}

func (s *Server) refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("wishtrack_refresh")
	if err != nil || cookie.Value == "" {
		writeError(w, http.StatusUnauthorized, "REFRESH_REQUIRED", "Сессия завершена")
		return
	}
	newRefresh, err := platform.RandomToken(48)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	user, err := s.Store.RotateSession(r.Context(), cookie.Value, newRefresh,
		time.Now().Add(s.Config.RefreshTokenTTL))
	if err != nil {
		s.clearRefreshCookie(w)
		writeError(w, http.StatusUnauthorized, "INVALID_REFRESH_TOKEN", "Сессия завершена")
		return
	}
	s.setRefreshCookie(w, newRefresh)
	token, expiresAt, err := s.Tokens.Issue(user.ID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"accessToken": token, "expiresAt": expiresAt.Format(time.RFC3339), "user": user,
	})
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("wishtrack_refresh"); err == nil {
		if err := s.Store.RevokeSession(r.Context(), cookie.Value); err != nil {
			s.Logger.Warn("revoke refresh session", "error", err)
		}
	}
	s.clearRefreshCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) issueSession(w http.ResponseWriter, r *http.Request, user store.User) {
	refresh, err := platform.RandomToken(48)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	if err := s.Store.CreateSession(r.Context(), user.ID, refresh, time.Now().Add(s.Config.RefreshTokenTTL)); err != nil {
		writeStoreError(w, err)
		return
	}
	access, expiresAt, err := s.Tokens.Issue(user.ID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	s.setRefreshCookie(w, refresh)
	writeJSON(w, http.StatusOK, map[string]any{
		"accessToken": access, "expiresAt": expiresAt.Format(time.RFC3339), "user": user,
	})
}

func (s *Server) setRefreshCookie(w http.ResponseWriter, value string) {
	http.SetCookie(w, &http.Cookie{
		Name: "wishtrack_refresh", Value: value, Path: "/api/v1/auth",
		HttpOnly: true, Secure: s.Config.Env == "production", SameSite: http.SameSiteLaxMode,
		MaxAge: int(s.Config.RefreshTokenTTL.Seconds()), Expires: time.Now().Add(s.Config.RefreshTokenTTL),
	})
}

func (s *Server) clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: "wishtrack_refresh", Value: "", Path: "/api/v1/auth",
		HttpOnly: true, Secure: s.Config.Env == "production", SameSite: http.SameSiteLaxMode,
		MaxAge: -1, Expires: time.Unix(1, 0),
	})
}
