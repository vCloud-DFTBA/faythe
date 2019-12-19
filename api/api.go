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
	"encoding/json"
	"net/http"
	"net/http/pprof"
	"sync"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/vCloud-DFTBA/faythe/config"
	"github.com/vCloud-DFTBA/faythe/pkg/common"
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
	logger  log.Logger
	uptime  time.Time
	etcdcli *common.Etcd
	mtx     sync.RWMutex
}

// New returns a new API.
func New(l log.Logger, e *common.Etcd) *API {
	if l == nil {
		l = log.NewNopLogger()
	}

	return &API{
		logger:  l,
		uptime:  time.Now(),
		etcdcli: e,
	}
}

// Register registers the API handlers under their correct routes
// in the given router.
func (a *API) Register(r *mux.Router) {
	wrap := func(f http.HandlerFunc) http.HandlerFunc {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			setCORS(w)
			f(w, r)
		})
	}
	// Prometheus golang metrics
	r.Handle("/metrics", promhttp.Handler()).Methods("GET")
	r.Handle("/", wrap(a.index)).Methods("GET")
	r.Handle("/status", wrap(a.status)).Methods("GET")
	// Cloud endpoints
	r.Handle("/clouds/{provider}", wrap(a.registerCloud)).Methods("POST")
	r.Handle("/clouds", wrap(a.listClouds)).Methods("GET")
	r.Handle("/clouds/{id:[a-z 0-9]+}", wrap(a.unregisterCloud)).Methods("DELETE")
	r.Handle("/clouds/{id:[a-z 0-9]+}", wrap(a.updateCloud)).Methods("PUT")

	// Scaler endpoints
	r.Handle("/scalers/{provider_id:[a-z 0-9]+}", wrap(a.createScaler)).Methods("POST")
	r.Handle("/scalers/{provider_id:[a-z 0-9]+}", wrap(a.listScalers)).Methods("GET")
	r.Handle("/scalers/{provider_id:[a-z 0-9]+}/{id:[a-z 0-9]+}",
		wrap(a.deleteScaler)).Methods("DELETE")
	r.Handle("/scalers/{provider_id:[a-z 0-9]+}/{id:[a-z 0-9]+}",
		wrap(a.updateScaler)).Methods("PUT")

	// Name Resolver endpoints
	r.Handle("/nresolvers", wrap(a.listNResolvers)).Methods("GET")
	// Healer endpoints
	r.Handle("/healers/{provider_id:[a-z 0-9]+}", wrap(a.createHealer)).Methods("POST")
	r.Handle("/healers/{provider_id:[a-z 0-9]+}", wrap(a.listHealers)).Methods("GET")
	r.Handle("/healers/{provider_id:[a-z 0-9]+}/{id:[a-z 0-9]+}",
		wrap(a.deleteHealer)).Methods("DELETE")

	// Profiling endpoints
	cfg := config.Get().GlobalConfig
	if cfg.EnableProfiling {
		r.Handle("/debug/pprof", http.HandlerFunc(pprof.Index))
		r.Handle("/debug/pprof/{profile}", http.DefaultServeMux)
	}
}

func (a *API) receive(req *http.Request, v interface{}) error {
	dec := json.NewDecoder(req.Body)
	defer req.Body.Close()
	err := dec.Decode(v)
	if err != nil {
		level.Debug(a.logger).Log("msg", "Decoding request failed", "err", err)
	}
	return err
}

func (a *API) respondError(w http.ResponseWriter, e apiError) {
	w.Header().Set("Content-Type", "application/json")
	level.Error(a.logger).Log("msg", "API error", "err", e.Error())

	b, err := json.Marshal(&response{
		Status: http.StatusText(e.code),
		Err:    e.err.Error(),
	})

	if err != nil {
		level.Error(a.logger).Log("msg", "Error marshalling JSON", "err", err)
	} else {
		if _, err := w.Write(b); err != nil {
			level.Error(a.logger).Log("msg", "failed to write data to connection", "err", err)
		}
	}

	http.Error(w, e.Error(), e.code)
}

type response struct {
	Status string
	Data   interface{}
	Err    string
}

func (a *API) respondSuccess(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	status := http.StatusText(code)
	b, err := json.Marshal(&response{
		Status: status,
		Data:   data,
	})

	if err != nil {
		level.Error(a.logger).Log("msg", "Error marshalling JSON", "err", err)
		return
	}
	if _, err := w.Write(b); err != nil {
		level.Error(a.logger).Log("msg", "failed to write data to connection", "err", err)
	}
}
