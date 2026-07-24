package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

func (s *Server) middleware(next http.Handler) http.Handler {
	return s.recoverer(s.requestLogger(s.securityHeaders(s.cors(s.globalLimit(next)))))
}

func (s *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; img-src 'self' data: https:; style-src 'self' 'unsafe-inline'; "+
				"script-src 'self' https://telegram.org; connect-src 'self'; frame-ancestors https://web.telegram.org https://*.telegram.org")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && origin == s.Config.FrontendOrigin {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
		}
		if r.Method == http.MethodOptions {
			if origin != s.Config.FrontendOrigin {
				writeError(w, http.StatusForbidden, "ORIGIN_NOT_ALLOWED", "Origin не разрешён")
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := strings.TrimSpace(r.Header.Get("X-Request-ID"))
		if requestID == "" || len(requestID) > 100 {
			var id [8]byte
			_, _ = rand.Read(id[:])
			requestID = hex.EncodeToString(id[:])
		}
		w.Header().Set("X-Request-ID", requestID)
		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r.WithContext(context.WithValue(r.Context(), requestIDKey, requestID)))
		s.Logger.Info("http request",
			"request_id", requestID,
			"method", r.Method,
			"path", r.URL.Path,
			"status", recorder.status,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (w *statusRecorder) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (s *Server) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				s.Logger.Error("panic in request", "panic", recovered, "path", r.URL.Path)
				writeError(w, http.StatusInternalServerError, "INTERNAL", "Что-то пошло не так")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

type visitor struct {
	count int
	reset time.Time
}

type limiter struct {
	mu       sync.Mutex
	visitors map[string]visitor
	limit    int
	window   time.Duration
}

func newLimiter(limit int, window time.Duration) *limiter {
	return &limiter{visitors: make(map[string]visitor), limit: limit, window: window}
}

func (l *limiter) allow(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	item := l.visitors[key]
	if item.reset.Before(now) {
		item = visitor{reset: now.Add(l.window)}
	}
	item.count++
	l.visitors[key] = item
	return item.count <= l.limit
}

func (s *Server) globalLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, _ := net.SplitHostPort(r.RemoteAddr)
		if host == "" {
			host = r.RemoteAddr
		}
		if !s.Limiter.allow(host, time.Now()) {
			w.Header().Set("Retry-After", "60")
			writeError(w, http.StatusTooManyRequests, "RATE_LIMITED", "Слишком много запросов. Попробуйте чуть позже")
			return
		}
		next.ServeHTTP(w, r)
	})
}

var _ = slog.LevelInfo
