package main

import (
	"context"
	"log"
	"net/http"
	"regexp"
	"time"

	"github.com/ntk148v/faythe/config"
)

type key int

const (
	requestIDKey key = 0
)

type middleware struct {
	logger        *log.Logger
	nextRequestID func() string
	regexp        *regexp.Regexp
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

// authencating verify authentication provided in the request's Authorization header
// if the request uses HTTP Basic Authentication.
func (mw *middleware) authenticating(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, _ := r.BasicAuth()
		basicAuth := config.Get().ServerConfig.BasicAuthentication
		correctUsr := basicAuth.Username
		correctPwd := string(basicAuth.Password)

		if correctUsr != user || correctPwd != pass {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "Unauthorized.", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// restrictingDomain checks whehter request's remote address was matched
// a defined host pattern or not.
func (mw *middleware) restrictingDomain(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		matched := mw.regexp.MatchString(r.RemoteAddr)
		if !matched {
			http.Error(w, "Remote address is not matched restricted domain pattern", http.StatusNotFound)
			return
		}

		next.ServeHTTP(w, r)
	})
}
