package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/example/wishtrack/internal/preview"
	"github.com/example/wishtrack/internal/store"
	"github.com/go-chi/chi/v5"
)

func (s *Server) createWish(w http.ResponseWriter, r *http.Request) {
	user, _ := userFromContext(r.Context())
	var input store.WishInput
	if !decodeJSON(w, r, &input) {
		return
	}
	if cachedURL, err := s.cacheRemoteImage(
		r.Context(), user.ID, input.ImageURL, input.ProductURL,
	); err != nil {
		s.Logger.Warn("cache new wish image", "user_id", user.ID, "error", err)
	} else {
		input.ImageURL = cachedURL
	}
	wish, err := s.Store.CreateWish(r.Context(), user.ID, chi.URLParam(r, "id"),
		input, s.Config.NotificationDigest)
	if err != nil {
		if err == store.ErrNotFound {
			writeStoreError(w, err)
		} else {
			writeError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		}
		return
	}
	writeJSON(w, http.StatusCreated, wish)
}

func (s *Server) getWish(w http.ResponseWriter, r *http.Request) {
	user, _ := userFromContext(r.Context())
	wish, err := s.Store.GetWish(r.Context(), chi.URLParam(r, "wishID"), user.ID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, wish)
}

func (s *Server) updateWish(w http.ResponseWriter, r *http.Request) {
	user, _ := userFromContext(r.Context())
	var input store.WishInput
	if !decodeJSON(w, r, &input) {
		return
	}
	if cachedURL, err := s.cacheRemoteImage(
		r.Context(), user.ID, input.ImageURL, input.ProductURL,
	); err != nil {
		s.Logger.Warn("cache updated wish image", "user_id", user.ID, "error", err)
	} else {
		input.ImageURL = cachedURL
	}
	wish, err := s.Store.UpdateWish(r.Context(), chi.URLParam(r, "wishID"), user.ID, input)
	if err != nil {
		if err == store.ErrNotFound || err == store.ErrVersionConflict {
			writeStoreError(w, err)
		} else {
			writeError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, wish)
}

func (s *Server) deleteWish(w http.ResponseWriter, r *http.Request) {
	user, _ := userFromContext(r.Context())
	if err := s.Store.DeleteWish(r.Context(), chi.URLParam(r, "wishID"), user.ID); err != nil {
		writeStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) previewURL(w http.ResponseWriter, r *http.Request) {
	var request struct {
		URL string `json:"url"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	result, err := s.Preview.Fetch(ctx, strings.TrimSpace(request.URL))
	if err != nil {
		code := "PREVIEW_FAILED"
		message := "Не удалось прочитать страницу. Заполните поля вручную"
		if errors.Is(err, preview.ErrUnsafeURL) {
			code = "UNSAFE_URL"
			message = "Этот адрес недоступен для импорта"
		}
		writeError(w, http.StatusUnprocessableEntity, code, message)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
