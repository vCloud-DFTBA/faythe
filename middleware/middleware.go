// Copyright (c) 2019 Kien Nguyen-Tuan <kiennt2609@gmail.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package middleware

import (
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/vCloud-DFTBA/faythe/config"
	"github.com/vCloud-DFTBA/faythe/pkg/exporter"
)

// Middleware represents middleware handlers.
type Middleware struct {
	logger log.Logger
	auth   config.BasicAuthentication
	regexp *regexp.Regexp
}

// New returns a new Middleware.
func New(l log.Logger) *Middleware {
	if l == nil {
		l = log.NewNopLogger()
	}

	cfg := config.Get().GlobalConfig
	a := cfg.BasicAuthentication
	r, _ := regexp.Compile(cfg.RemoteHostPattern)

	return &Middleware{
		logger: l,
		auth:   a,
		regexp: r,
	}
}

type instrumentHandler struct {
	handler http.Handler
}

func (h instrumentHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// NOTE(kiennt): The path may contain the component uuid -> cardinality explosion?
	handlerName := req.URL.Path
	if !strings.HasPrefix(handlerName, "/metrics") {
		h.handler = promhttp.InstrumentHandlerInFlight(exporter.InFlightGauge,
			promhttp.InstrumentHandlerDuration(exporter.RequestDuration.MustCurryWith(prometheus.Labels{"handler": handlerName}),
				promhttp.InstrumentHandlerCounter(exporter.RequestsCount.MustCurryWith(prometheus.Labels{"handler": handlerName}),
					promhttp.InstrumentHandlerRequestSize(exporter.RequestSize.MustCurryWith(prometheus.Labels{"handler": handlerName}),
						promhttp.InstrumentHandlerResponseSize(exporter.ResponseSize.MustCurryWith(prometheus.Labels{"handler": handlerName}),
							h.handler)))))
	}
	h.handler.ServeHTTP(w, req)
}

// Instrument is a middleware that wraps the provided http.Handler
// to observe the request result.
func (m *Middleware) Instrument(next http.Handler) http.Handler {
	return instrumentHandler{handler: next}
}

// Logging logs all requests with its information and the time it took to process
func (m *Middleware) Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		defer func() {
			start := time.Now()
			level.Info(m.logger).Log("msg", "Receiving request", "method", req.Method, "url",
				req.URL, "remote-addr", req.RemoteAddr,
				"user-agent", req.UserAgent(),
				"time", time.Since(start))
		}()
		next.ServeHTTP(w, req)
	})
}

// Authenticate verifies authentication provided in the request's Authorization
// header if the request uses JSON web tokens.
func (m *Middleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		c, err := req.Cookie("api-token")
		if err != nil {
			if err == http.ErrNoCookie {
				level.Error(m.logger).Log("msg", "Unauthorized request",
					"endpoint", req.RequestURI, "addr", req.RemoteAddr)
				http.Error(w, "Login Required!", http.StatusUnauthorized)
				return
			}

			level.Error(m.logger).Log("msg", "Error while getting request cookie",
				"err", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		tokenString := c.Value
		claims := &jwt.StandardClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (i interface{}, err error) {
			return []byte(m.auth.SecretKey), nil
		})

		if err != nil {
			if err == jwt.ErrSignatureInvalid {
				level.Error(m.logger).Log("msg", "Unauthorized request",
					"endpoint", req.RequestURI, "addr", req.RemoteAddr)
				http.Error(w, "Login Required!", http.StatusUnauthorized)
				return
			}

			level.Error(m.logger).Log("msg", "Error while verifying token",
				"err", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if !token.Valid {
			level.Error(m.logger).Log("msg", "Unauthorized request",
				"endpoint", req.RequestURI, "addr", req.RemoteAddr)
			http.Error(w, "Login Required!", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, req)
	})
}

// RestrictDomain checks whehter request's remote address was matched
// a defined host pattern or not.
func (m *Middleware) RestrictDomain(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		matched := m.regexp.MatchString(req.RemoteAddr)
		if !matched {
			level.Error(m.logger).Log("msg", "Remote address is not matched restricted domain pattern")
			http.Error(w, "Remote address is not matched restricted domain pattern", http.StatusNotFound)
			return
		}

		next.ServeHTTP(w, req)
	})
}
