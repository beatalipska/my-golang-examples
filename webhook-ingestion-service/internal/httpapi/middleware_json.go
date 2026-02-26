package httpapi

import (
	"context"
	"io"
	"net/http"
	"time"

	"webhook-ingestion-service/internal/observability/jsonlog"
)

func WithRequestIDJSON(_ *jsonlog.Logger) func(http.Handler) http.Handler {
	// logger not used here, but kept for symmetry / future extension
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rid := r.Header.Get(RequestIDHeader)
			if rid == "" {
				rid = newRequestID()
			}
			ctx := context.WithValue(r.Context(), requestIDKey, rid)
			r = r.WithContext(ctx)
			w.Header().Set(RequestIDHeader, rid)
			next.ServeHTTP(w, r)
		})
	}
}

func LoggingJSON(logger *jsonlog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = jsonlog.New(io.Discard) // should not happen; but safe
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &statusWriter{ResponseWriter: w, status: 200}

			next.ServeHTTP(sw, r)

			logger.Info("http_request", map[string]any{
				"rid":    RequestIDFromContext(r.Context()),
				"method": r.Method,
				"path":   r.URL.Path,
				"status": sw.status,
				"dur_ms": time.Since(start).Milliseconds(),
				"ua":     r.UserAgent(),
			})
		})
	}
}
