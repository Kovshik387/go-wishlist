package httpapi

import (
	"context"
	"net/http"
	"strings"

	"github.com/example/wishtrack/internal/store"
)

type contextKey string

const (
	userContextKey contextKey = "user"
	requestIDKey   contextKey = "request_id"
)

func userFromContext(ctx context.Context) (store.User, bool) {
	user, ok := ctx.Value(userContextKey).(store.User)
	return user, ok
}

func (s *Server) authRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := s.authenticate(r)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Нужно войти в приложение")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userContextKey, user)))
	})
}

func (s *Server) optionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := s.authenticate(r)
		if err == nil {
			r = r.WithContext(context.WithValue(r.Context(), userContextKey, user))
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) authenticate(r *http.Request) (store.User, error) {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(header, "Bearer ") {
		return store.User{}, errUnauthorized
	}
	userID, err := s.Tokens.Parse(strings.TrimSpace(strings.TrimPrefix(header, "Bearer ")))
	if err != nil {
		return store.User{}, err
	}
	return s.Store.UserByID(r.Context(), userID)
}
