package basic

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"time"
)

// Index handles request to / url.
func Index() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintf(w, "Hello stranger! Welcome to Cloudhotpot-middleware")
	})
}

// Healthz handles requests to /healthz and returns uptime.
func Healthz(healthy *int64) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h := atomic.LoadInt64(healthy); h == 0 {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			fmt.Fprintf(w, "uptime: %s\n", time.Since(time.Unix(0, h)))
		}
	})
}
