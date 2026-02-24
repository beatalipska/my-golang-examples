package httpapi

import (
	"context"
	"log"
	"net/http"
	"time"

	"tiny-tasks/internal/ids"
)

func withMiddleware(next http.Handler) http.Handler {
	return requestLogging(requestID(timeout(next, 3*time.Second)))
}

func timeout(next http.Handler, d time.Duration) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), d)
		defer cancel()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-Id")
		if reqID == "" {
			reqID = ids.NewID()
		}
		w.Header().Set("X-Request-Id", reqID)
		next.ServeHTTP(w, r)
	})
}

func requestLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("req_id=%s method=%s path=%s dur=%s",
			w.Header().Get("X-Request-Id"), r.Method, r.URL.Path, time.Since(start))
	})
}
