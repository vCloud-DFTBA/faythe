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
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"

	"github.com/ntk148v/faythe/config"
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
// header if the request uses HTTP Basic Authentication.
func (m *Middleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		user, pass, _ := req.BasicAuth()
		if m.auth.Username != user || string(m.auth.Password) != pass {
			level.Error(m.logger).Log("msg", "Unauthorized request")
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "Unauthorized.", http.StatusUnauthorized)
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
