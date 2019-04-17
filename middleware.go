package main

import (
	"context"
	"log"
	"net/http"
	"time"
)

type middleware func(http.Handler) http.Handler

type key int

const (
	requestIDKey key = 0
)

// logging logs all requests with its information and the time it took to process
func logging(logger *log.Logger) middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				requestID, ok := r.Context().Value(requestIDKey).(string)
				if !ok {
					requestID = "unknown"
				}
				start := time.Now()
				logger.Println(requestID, r.Method, r.URL.Path, r.RemoteAddr, r.UserAgent(), time.Since(start))
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// tracing appends a ID to each request
func tracing(nextRequestID func() string) middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get("X-Request-Id")
			if requestID == "" {
				requestID = nextRequestID()
			}
			ctx := context.WithValue(r.Context(), requestIDKey, requestID)
			w.Header().Set("X-Request-Id", requestID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
