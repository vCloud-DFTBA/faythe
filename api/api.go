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

package api

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/gorilla/mux"
	etcdv3 "go.etcd.io/etcd/clientv3"
)

var corsHeaders = map[string]string{
	"Access-Control-Allow-Headers":  "Accept, Authorization, Content-Type, Origin",
	"Access-Control-Allow-Methods":  "GET, POST",
	"Access-Control-Allow-Origin":   "*",
	"Access-Control-Expose-Headers": "Date",
	"Cache-Control":                 "no-cache, no-store, must-revalidate",
}

// Enables cross-site script calls.
func setCORS(w http.ResponseWriter) {
	for h, v := range corsHeaders {
		w.Header().Set(h, v)
	}
}

// API provides registration of handlers for API routes
type API struct {
	logger     log.Logger
	uptime     time.Time
	etcdclient *etcdv3.Client
	mtx        sync.RWMutex
}

// New returns a new API.
func New(l log.Logger, e *etcdv3.Client) *API {
	if l == nil {
		l = log.NewNopLogger()
	}

	return &API{
		logger:     l,
		uptime:     time.Now(),
		etcdclient: e,
	}
}

// Register registers the API handlers under their correct routes
// in the given router.
func (api *API) Register(r *mux.Router) {
	wrap := func(f http.HandlerFunc) http.HandlerFunc {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			setCORS(w)
			f(w, r)
		})
	}
	r.Handle("/", wrap(api.index)).Methods("GET")
	r.Handle("/status", wrap(api.status)).Methods("GET")
}

func (api *API) index(w http.ResponseWriter, req *http.Request) {
	if req.URL.Path != "/" {
		http.NotFound(w, req)
		return
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "Hello stranger! Welcome to Faythe!")
}

func (api *API) status(w http.ResponseWriter, req *http.Request) {
	// A placeholder here
	// Represents Faythe status (version, uptime...) here.
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "Status here - This is just a placeholder")
}
