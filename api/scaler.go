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

	"github.com/gorilla/mux"
	etcdv3 "go.etcd.io/etcd/clientv3"

	"github.com/ntk148v/faythe/pkg/model"
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
	path = fmt.Sprintf("%s/%s", model.DefaultOpenStackPrefix, vars["provider_id"])
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
	path = fmt.Sprintf("%s/%s/%s", model.DefaultScalerPrefix, vars["provider_id"], s.ID)
	if strings.ToLower(req.URL.Query().Get("force")) == "true" {
		force = true
	}
	resp, _ = a.etcdclient.Get(req.Context(), path+"/scaler", etcdv3.WithCountOnly())
	if resp.Count > 0 && !force {
		err := fmt.Errorf("The provider with id %s is existed", s.ID)
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
}

func (a *API) listScalers(w http.ResponseWriter, req *http.Request) {
	// List all current Scalers from etcd3
}

func (a *API) deleteScaler(w http.ResponseWriter, req *http.Request) {
	// Delete a Scaler from etcd3
}

func (a *API) updateScaler(w http.ResponseWriter, req *http.Request) {
	// Update a existed Scaler information
}
