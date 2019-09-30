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
	"fmt"
	"net/http"
	"strings"

	"github.com/go-kit/kit/log/level"
	"github.com/gorilla/mux"
	etcdv3 "go.etcd.io/etcd/clientv3"

	"github.com/ntk148v/faythe/pkg/model"
	"github.com/ntk148v/faythe/pkg/utils"
)

func (a *API) createScaler(w http.ResponseWriter, req *http.Request) {
	// Save a Scaler object in etcd3
	var (
		s     *model.Scaler
		path  string
		vars  map[string]string
		v     []byte
		force bool
	)
	vars = mux.Vars(req)
	path = utils.Path(model.DefaultOpenStackPrefix, vars["provider_id"])
	resp, _ := a.etcdclient.Get(req.Context(), path, etcdv3.WithCountOnly())
	if resp.Count == 0 {
		err := fmt.Errorf("Unknown provider id: %s", vars["provider_id"])
		a.respondError(w, apiError{
			code: http.StatusBadRequest,
			err:  err,
		})
		return
	}
	if err := a.receive(req, &s); err != nil {
		a.respondError(w, apiError{
			code: http.StatusBadRequest,
			err:  err,
		})
		return
	}
	if err := s.Validate(); err != nil {
		a.respondError(w, apiError{
			code: http.StatusBadRequest,
			err:  err,
		})
		return
	}
	path = utils.Path(model.DefaultScalerPrefix, vars["provider_id"], s.ID)
	if strings.ToLower(req.URL.Query().Get("force")) == "true" {
		force = true
	}
	resp, _ = a.etcdclient.Get(req.Context(), path, etcdv3.WithCountOnly())
	if resp.Count > 0 && !force {
		err := fmt.Errorf("The scaler with id %s is existed", s.ID)
		a.respondError(w, apiError{
			code: http.StatusBadRequest,
			err:  err,
		})
		return
	}
	v, _ = json.Marshal(&s)
	_, err := a.etcdclient.Put(req.Context(), path, string(v))
	if err != nil {
		err = fmt.Errorf("Error putting a key-value pair into etcd: %s", err.Error())
		a.respondError(w, apiError{
			code: http.StatusInternalServerError,
			err:  err,
		})
		return
	}

	a.respondSuccess(w, http.StatusOK, nil)
	return
}

// List all current Scalers from etcd3
func (a *API) listScalers(w http.ResponseWriter, req *http.Request) {
	var (
		vars    map[string]string
		pid     string
		path    string
		scalers map[string]model.Scaler
	)
	vars = mux.Vars(req)
	pid = strings.ToLower(vars["provider_id"])
	path = utils.Path(model.DefaultScalerPrefix, pid)
	resp, err := a.etcdclient.Get(req.Context(), path, etcdv3.WithPrefix(),
		etcdv3.WithSort(etcdv3.SortByKey, etcdv3.SortAscend))
	if err != nil {
		a.respondError(w, apiError{
			code: http.StatusInternalServerError,
			err:  err,
		})
		return
	}

	scalers = make(map[string]model.Scaler, len(resp.Kvs))
	for _, ev := range resp.Kvs {
		var s model.Scaler
		err = json.Unmarshal(ev.Value, &s)
		if err != nil {
			level.Error(a.logger).Log("msg", "Error getting scaler from etcd",
				"scaler", ev.Key, "err", err)
			continue
		}
		scalers[string(ev.Key)] = s
	}
	a.respondSuccess(w, http.StatusOK, scalers)
	return
}

// Delete a Scaler from etcd3
func (a *API) deleteScaler(w http.ResponseWriter, req *http.Request) {
	var (
		vars map[string]string
		pid  string
		sid  string
		path string
	)

	vars = mux.Vars(req)
	pid = strings.ToLower(vars["provider_id"])
	sid = strings.ToLower(vars["id"])
	path = utils.Path(model.DefaultScalerPrefix, pid, sid)
	resp, err := a.etcdclient.Delete(req.Context(), path, etcdv3.WithPrefix())
	if err != nil {
		a.respondError(w, apiError{
			code: http.StatusInternalServerError,
			err:  err,
		})
		return
	}
	fmt.Printf("%+v", resp)
	a.respondSuccess(w, http.StatusOK, nil)
	return
}

func (a *API) updateScaler(w http.ResponseWriter, req *http.Request) {
	// Update a existed Scaler information
}
