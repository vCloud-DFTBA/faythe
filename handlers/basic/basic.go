package basic

import (
	"io"
	"net/http"
	"sync/atomic"
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
		io.WriteString(w, "Hello, World!")
	})
}

// Healthz handles requests to /healthz and returns uptime.
func Healthz(healthy int32) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.LoadInt32(&healthy) == 1 {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			io.WriteString(w, `{"alive": true}`)
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	})
}
