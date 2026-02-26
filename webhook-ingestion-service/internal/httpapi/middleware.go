package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"time"
)

type ctxKey string

const (
	requestIDKey    ctxKey = "request_id"
	RequestIDHeader        = "X-Request-Id"
)

// RequestIDFromContext returns request id if present.
func RequestIDFromContext(ctx context.Context) string {
	v := ctx.Value(requestIDKey)
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func WithRequestID(logger *log.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = log.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rid := r.Header.Get(RequestIDHeader)
			if rid == "" {
				rid = newRequestID()
			}

			// attach to context + response header
			ctx := context.WithValue(r.Context(), requestIDKey, rid)
			r = r.WithContext(ctx)
			w.Header().Set(RequestIDHeader, rid)

			next.ServeHTTP(w, r)
		})
	}
}

func Logging(logger *log.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = log.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &statusWriter{ResponseWriter: w, status: 200}

			next.ServeHTTP(sw, r)

			rid := RequestIDFromContext(r.Context())
			dur := time.Since(start)

			logger.Printf(
				"rid=%s method=%s path=%s status=%d dur=%s ua=%q",
				rid,
				r.Method,
				r.URL.Path,
				sw.status,
				dur,
				r.UserAgent(),
			)
		})
	}
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func newRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// fallback: timestamp-based (still ok for dev); but rand.Read should basically never fail
		return hex.EncodeToString([]byte(time.Now().UTC().Format("20060102150405.000000000")))
	}
	return hex.EncodeToString(b[:])
}
