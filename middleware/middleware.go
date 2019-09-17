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
	"fmt"
	"net/http"
	"net/http/httputil"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

// Middleware represents middleware handlers.
type Middleware struct {
	logger log.Logger
}

// New returns a new Middleware.
func New(l log.Logger) *Middleware {
	if l == nil {
		l = log.NewNopLogger()
	}

	return &Middleware{logger: l}
}

// Logging logs all requests with its information and the time it took to process
func (m *Middleware) Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		defer func() {
			dump, err := httputil.DumpRequest(req, true)
			if err != nil {
				http.Error(w, fmt.Sprint(err), http.StatusInternalServerError)
				return
			}
			level.Info(m.logger).Log("req", fmt.Sprintf("%s", dump))
		}()
		next.ServeHTTP(w, req)
	})
}
