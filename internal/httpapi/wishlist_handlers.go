package httpapi

import (
	"net/http"

	"github.com/example/wishtrack/internal/store"
	"github.com/go-chi/chi/v5"
)

func (s *Server) listWishlists(w http.ResponseWriter, r *http.Request) {
	user, _ := userFromContext(r.Context())
	lists, err := s.Store.ListWishlists(r.Context(), user.ID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	saved, err := s.Store.ListSavedWishlists(r.Context(), user.ID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": lists, "saved": saved})
}

func (s *Server) createWishlist(w http.ResponseWriter, r *http.Request) {
	user, _ := userFromContext(r.Context())
	var input store.WishlistInput
	if !decodeJSON(w, r, &input) {
		return
	}
	list, err := s.Store.CreateWishlist(r.Context(), user.ID, input)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, list)
}

func (s *Server) getWishlist(w http.ResponseWriter, r *http.Request) {
	user, _ := userFromContext(r.Context())
	list, err := s.Store.GetWishlist(r.Context(), chi.URLParam(r, "id"), user.ID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) forgetWishlist(w http.ResponseWriter, r *http.Request) {
	user, _ := userFromContext(r.Context())
	if err := s.Store.ForgetWishlist(r.Context(), chi.URLParam(r, "id"), user.ID); err != nil {
		writeStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) updateWishlist(w http.ResponseWriter, r *http.Request) {
	user, _ := userFromContext(r.Context())
	var input store.WishlistInput
	if !decodeJSON(w, r, &input) {
		return
	}
	list, err := s.Store.UpdateWishlist(r.Context(), chi.URLParam(r, "id"), user.ID, input)
	if err != nil {
		if err == store.ErrVersionConflict || err == store.ErrNotFound {
			writeStoreError(w, err)
		} else {
			writeError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) deleteWishlist(w http.ResponseWriter, r *http.Request) {
	user, _ := userFromContext(r.Context())
	if err := s.Store.DeleteWishlist(r.Context(), chi.URLParam(r, "id"), user.ID); err != nil {
		writeStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) rotateWishlistLink(w http.ResponseWriter, r *http.Request) {
	user, _ := userFromContext(r.Context())
	list, err := s.Store.RotateWishlistLink(r.Context(), chi.URLParam(r, "id"), user.ID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) publicWishlist(w http.ResponseWriter, r *http.Request) {
	viewer, _ := userFromContext(r.Context())
	list, err := s.Store.PublicWishlist(r.Context(), chi.URLParam(r, "token"), viewer.ID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}
