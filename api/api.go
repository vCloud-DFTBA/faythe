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
	"time"

	"github.com/casbin/casbin/v2"
	casbinmodel "github.com/casbin/casbin/v2/model"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/gorilla/mux"
	"github.com/jinzhu/copier"
	etcdadapter "github.com/ntk148v/etcd-adapter"
	"github.com/ntk148v/jwt-middleware"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	etcdv3 "go.etcd.io/etcd/clientv3"

	"github.com/vCloud-DFTBA/faythe/config"
	"github.com/vCloud-DFTBA/faythe/middleware"
	"github.com/vCloud-DFTBA/faythe/pkg/cluster"
	"github.com/vCloud-DFTBA/faythe/pkg/common"
	"github.com/vCloud-DFTBA/faythe/pkg/model"
)

// API provides registration of handlers for API routes
type API struct {
	logger       log.Logger
	uptime       time.Time
	etcdcli      *common.Etcd
	jwtToken     *jwt.Token
	policyEngine *casbin.Enforcer
}

// New returns a new API.
func New(l log.Logger, e *common.Etcd) (*API, error) {
	jwtCfg := config.Get().JWTConfig
	token, err := jwt.NewToken(jwt.Options{
		SigningMethod:      jwtCfg.SigningMethod,
		PrivateKeyLocation: jwtCfg.PrivateKeyLocation,
		PublicKeyLocation:  jwtCfg.PublicKeyLocation,
		IsBearerToken:      jwtCfg.IsBearerToken,
		UserProperty:       jwtCfg.UserProperty,
		TTL:                jwtCfg.TTL,
	}, nil)

	if err != nil {
		// Exit immediately
		return nil, errors.Wrapf(err, "Error initializing the JWT instance")
	}
	if l == nil {
		l = log.NewNopLogger()
	}

	// Init Policy engine
	var etcdCfg etcdv3.Config
	copier.Copy(&etcdCfg, config.Get().EtcdConfig)
	adapter := etcdadapter.NewAdapter(etcdCfg, cluster.GetID(), model.DefaultPoliciesPrefix)
	policyModel := casbinmodel.NewModel()
	policyModel.AddDef("r", "r", "sub, obj, act")
	policyModel.AddDef("p", "p", "sub, obj, act")
	policyModel.AddDef("e", "e", "some(where (p.eft == allow))")
	policyModel.AddDef("m", "m", "r.sub == p.sub && keyMatch(r.obj, p.obj) && regexMatch(r.act, p.act)")
	policyEngine, err := casbin.NewEnforcer(policyModel, adapter)

	a := &API{
		logger:       l,
		uptime:       time.Now(),
		etcdcli:      e,
		jwtToken:     token,
		policyEngine: policyEngine,
	}
	// Create an admin user & grant permissions
	if err = a.createAdminUser(); err != nil {
		return nil, errors.Wrap(err, "Error creating admin user")
	}
	return a, nil
}

// Register registers the API handlers under their correct routes
// in the given router.
func (a *API) Register(r *mux.Router) {
	mw := middleware.New(log.With(a.logger, "component", "transport middleware"))
	// General middleware
	r.Use(mw.Instrument, mw.Logging, mw.RestrictDomain, mw.HandleCors)
	// Unrestricted subrouter
	r.HandleFunc("/tokens", a.issueToken).Methods("OPTIONS", "POST")
	// Prometheus golang metrics
	r.Handle("/metrics", promhttp.Handler()).Methods("GET")
	r.HandleFunc("/", a.index).Methods("GET")
	r.HandleFunc("/status", a.status).Methods("GET")

	// Profiling endpoints
	cfg := config.Get()
	if cfg.EnableProfiling {
		r.HandleFunc("/debug/pprof/", pprof.Index)
		r.Handle("/debug/pprof/{profile}", http.DefaultServeMux)
	}

	// Restricted subrouter
	resr := r.PathPrefix("/").Subrouter()
	resr.Use(jwt.Authenticator(a.jwtToken), middleware.Authorizer(a.policyEngine))

	// User endpoints
	resr.HandleFunc("/users", a.addUser).Methods("OPTIONS", "POST")
	resr.HandleFunc("/users/{user:[a-z 0-9]+}", a.removeUser).Methods("OPTIONS", "DELETE")
	resr.HandleFunc("/users", a.listUsers).Methods("OPTIONS", "GET")
	resr.HandleFunc("/users/{user:[a-z 0-9]+}/change_password", a.changePassword).Methods("OPTIONS", "PUT")

	// Policy endpoints
	resr.HandleFunc("/policies/{user:[a-z 0-9]+}", a.addPolicies).Methods("OPTIONS", "POST")
	resr.HandleFunc("/policies/{user:[a-z 0-9]+}", a.removePolicies).Methods("OPTIONS", "DELETE")
	// Cloud endpoints
	resr.HandleFunc("/clouds/{provider}", a.registerCloud).Methods("OPTIONS", "POST")
	resr.HandleFunc("/clouds", a.listClouds).Methods("OPTIONS", "GET")
	resr.HandleFunc("/clouds/{id:[a-z 0-9]+}", a.unregisterCloud).Methods("OPTIONS", "DELETE")
	resr.HandleFunc("/clouds/{id:[a-z 0-9]+}", a.updateCloud).Methods("OPTIONS", "PUT")

	// Scaler endpoints
	resr.HandleFunc("/scalers/{provider_id:[a-z 0-9]+}", a.createScaler).Methods("OPTIONS", "POST")
	resr.HandleFunc("/scalers/{provider_id:[a-z 0-9]+}", a.listScalers).Methods("OPTIONS", "GET")
	resr.HandleFunc("/scalers/{provider_id:[a-z 0-9]+}/{id:[a-z 0-9]+}",
		a.deleteScaler).Methods("OPTIONS", "DELETE")
	resr.HandleFunc("/scalers/{provider_id:[a-z 0-9]+}/{id:[a-z 0-9]+}",
		a.updateScaler).Methods("OPTIONS", "PUT")

	// Name Resolver endpoints
	resr.HandleFunc("/nresolvers", a.listNResolvers).Methods("OPTIONS", "GET")

	// Healer endpoints
	resr.HandleFunc("/healers/{provider_id:[a-z 0-9]+}", a.createHealer).Methods("OPTIONS", "POST")
	resr.HandleFunc("/healers/{provider_id:[a-z 0-9]+}", a.listHealers).Methods("OPTIONS", "GET")
	resr.HandleFunc("/healers/{provider_id:[a-z 0-9]+}/{id:[a-z 0-9]+}",
		a.deleteHealer).Methods("OPTIONS", "DELETE")

	// Silences endpoints
	resr.HandleFunc("/silences/{provider_id:[a-z 0-9]+}", a.createSilence).Methods("OPTIONS", "POST")
	resr.HandleFunc("/silences/{provider_id:[a-z 0-9]+}", a.listSilences).Methods("OPTIONS", "GET")
	resr.HandleFunc("/silences/{provider_id:[a-z 0-9]+}/{id:[a-z 0-9]+}", a.expireSilence).Methods("OPTIONS", "DELETE")
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
	w.WriteHeader(e.code)
	level.Error(a.logger).Log("msg", "API error", "err", e.Error())

	b, err := json.Marshal(&response{
		Status: http.StatusText(e.code),
		Err:    e.err.Error(),
	})

	if err != nil {
		level.Error(a.logger).Log("msg", "Error marshalling JSON", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
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

func (a *API) unauthorizedHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		a.respondError(w, apiError{
			code: http.StatusUnauthorized,
			err:  errors.New("Invalid credentials"),
		})
		return
	})
}
