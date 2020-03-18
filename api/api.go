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

// RegisterPublicRouter registers the Authentication API which has no
// Authentication middleware
func (a *API) RegisterPublicRouter(r *mux.Router) {
	r.HandleFunc("/login", a.getToken).Methods("OPTIONS", "GET")

	// Prometheus golang metrics
	r.Handle("/metrics", promhttp.Handler()).Methods("GET")

	r.HandleFunc("/", a.index).Methods("GET")
	r.HandleFunc("/status", a.status).Methods("GET")

	// Profiling endpoints
	cfg := config.Get().GlobalConfig
	if cfg.EnableProfiling {
		r.HandleFunc("/debug/pprof/", pprof.Index)
		r.Handle("/debug/pprof/{profile}", http.DefaultServeMux)
	}
}

// Register registers the API handlers under their correct routes
// in the given router.
func (a *API) Register(r *mux.Router) {
	// Cloud endpoints
	r.HandleFunc("/clouds/{provider}", a.registerCloud).Methods("OPTIONS", "POST")
	r.HandleFunc("/clouds", a.listClouds).Methods("OPTIONS", "GET")
	r.HandleFunc("/clouds/{id:[a-z 0-9]+}", a.unregisterCloud).Methods("OPTIONS", "DELETE")
	r.HandleFunc("/clouds/{id:[a-z 0-9]+}", a.updateCloud).Methods("OPTIONS", "PUT")

	// Scaler endpoints
	r.HandleFunc("/scalers/{provider_id:[a-z 0-9]+}", a.createScaler).Methods("OPTIONS", "POST")
	r.HandleFunc("/scalers/{provider_id:[a-z 0-9]+}", a.listScalers).Methods("OPTIONS", "GET")
	r.HandleFunc("/scalers/{provider_id:[a-z 0-9]+}/{id:[a-z 0-9]+}",
		a.deleteScaler).Methods("OPTIONS", "DELETE")
	r.HandleFunc("/scalers/{provider_id:[a-z 0-9]+}/{id:[a-z 0-9]+}",
		a.updateScaler).Methods("OPTIONS", "PUT")

	// Name Resolver endpoints
	r.HandleFunc("/nresolvers", a.listNResolvers).Methods("OPTIONS", "GET")

	// Healer endpoints
	r.HandleFunc("/healers/{provider_id:[a-z 0-9]+}", a.createHealer).Methods("OPTIONS", "POST")
	r.HandleFunc("/healers/{provider_id:[a-z 0-9]+}", a.listHealers).Methods("OPTIONS", "GET")
	r.HandleFunc("/healers/{provider_id:[a-z 0-9]+}/{id:[a-z 0-9]+}",
		a.deleteHealer).Methods("OPTIONS", "DELETE")

	// Silences endpoints
	r.HandleFunc("/silences/{provider_id:[a-z 0-9]+}", a.createSilence).Methods("OPTIONS", "POST")
	r.HandleFunc("/silences/{provider_id:[a-z 0-9]+}", a.listSilences).Methods("OPTIONS", "GET")
	r.HandleFunc("/silences/{provider_id:[a-z 0-9]+}/{id:[a-z 0-9]+}", a.expireSilence).Methods("OPTIONS", "DELETE")
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else {
		if _, err := w.Write(b); err != nil {
			level.Error(a.logger).Log("msg", "Failed to write data to connection", "err", err)
		}
	}
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
