package main

import (
	"context"
	"log"
	"net/http"
	"time"
)

type key int

const (
	requestIDKey key = 0
)

type middleware struct {
	logger        *log.Logger
	nextRequestID func() string
}

// logging logs all requests with its information and the time it took to process
func (mw *middleware) logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			requestID, ok := r.Context().Value(requestIDKey).(string)
			if !ok {
				requestID = "unknown"
			}
			start := time.Now()
			mw.logger.Println(requestID, r.Method, r.URL.Path, r.RemoteAddr, r.UserAgent(), time.Since(start))
		}()
		next.ServeHTTP(w, r)
	})
}

// tracing appends a ID to each request
func (mw *middleware) tracing(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-Id")
		if requestID == "" {
			requestID = mw.nextRequestID()
		}
		ctx := context.WithValue(r.Context(), requestIDKey, requestID)
		w.Header().Set("X-Request-Id", requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
