package httpapi

import (
	"net/http"

	"github.com/example/wishtrack/internal/store"
	"github.com/go-chi/chi/v5"
)

func (s *Server) feed(w http.ResponseWriter, r *http.Request) {
	user, _ := userFromContext(r.Context())
	items, err := s.Store.Feed(r.Context(), user.ID, 30)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) follow(w http.ResponseWriter, r *http.Request) {
	user, _ := userFromContext(r.Context())
	if err := s.Store.Follow(r.Context(), user.ID, chi.URLParam(r, "id")); err != nil {
		if err == store.ErrNotFound {
			writeStoreError(w, err)
		} else {
			writeError(w, http.StatusUnprocessableEntity, "FOLLOW_FAILED", err.Error())
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) unfollow(w http.ResponseWriter, r *http.Request) {
	user, _ := userFromContext(r.Context())
	if err := s.Store.Unfollow(r.Context(), user.ID, chi.URLParam(r, "id")); err != nil {
		writeStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) updateFollowSettings(w http.ResponseWriter, r *http.Request) {
	user, _ := userFromContext(r.Context())
	var request struct {
		Muted bool `json:"muted"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	if err := s.Store.SetFollowMuted(r.Context(), user.ID, chi.URLParam(r, "id"), request.Muted); err != nil {
		writeStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) reserve(w http.ResponseWriter, r *http.Request) {
	user, _ := userFromContext(r.Context())
	reservation, err := s.Store.ReserveWish(r.Context(), chi.URLParam(r, "id"), user.ID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, reservation)
}

func (s *Server) cancelReservation(w http.ResponseWriter, r *http.Request) {
	user, _ := userFromContext(r.Context())
	if err := s.Store.CancelReservation(r.Context(), chi.URLParam(r, "id"), user.ID); err != nil {
		writeStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) reservations(w http.ResponseWriter, r *http.Request) {
	user, _ := userFromContext(r.Context())
	items, err := s.Store.Reservations(r.Context(), user.ID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) notificationSettings(w http.ResponseWriter, r *http.Request) {
	user, _ := userFromContext(r.Context())
	prefs, err := s.Store.NotificationPreferences(r.Context(), user.ID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, prefs)
}

func (s *Server) updateNotificationSettings(w http.ResponseWriter, r *http.Request) {
	user, _ := userFromContext(r.Context())
	var prefs store.NotificationPreferences
	if !decodeJSON(w, r, &prefs) {
		return
	}
	updated, err := s.Store.UpdateNotificationPreferences(r.Context(), user.ID, prefs)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updated)
}
