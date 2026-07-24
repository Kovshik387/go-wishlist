package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	user, _ := userFromContext(r.Context())
	writeJSON(w, http.StatusOK, user)
}

func (s *Server) patchMe(w http.ResponseWriter, r *http.Request) {
	user, _ := userFromContext(r.Context())
	var request struct {
		DisplayName         string `json:"displayName"`
		Timezone            string `json:"timezone"`
		OnboardingCompleted *bool  `json:"onboardingCompleted"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	updated, err := s.Store.PatchUser(r.Context(), user.ID, request.DisplayName,
		request.Timezone, request.OnboardingCompleted)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) deleteMe(w http.ResponseWriter, r *http.Request) {
	user, _ := userFromContext(r.Context())
	if err := s.Store.DeleteAccount(r.Context(), user.ID); err != nil {
		writeStoreError(w, err)
		return
	}
	s.clearRefreshCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) publicUser(w http.ResponseWriter, r *http.Request) {
	viewer, _ := userFromContext(r.Context())
	user, following, err := s.Store.PublicUserByID(r.Context(), chi.URLParam(r, "id"), viewer.ID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": user, "following": following})
}
