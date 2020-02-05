// Copyright (c) 2019 Dat Vu Tuan <tuandatk25a@gmail.com>
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
	"crypto"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	cmap "github.com/orcaman/concurrent-map"
	etcdv3 "go.etcd.io/etcd/clientv3"

	"github.com/vCloud-DFTBA/faythe/pkg/common"
	"github.com/vCloud-DFTBA/faythe/pkg/metrics"
	"github.com/vCloud-DFTBA/faythe/pkg/model"
)

func (a *API) createHealer(rw http.ResponseWriter, req *http.Request) {
	if req.Method == "OPTIONS" {
		return
	}
	h := &model.Healer{}
	vars := mux.Vars(req)
	path := common.Path(model.DefaultCloudPrefix, vars["provider_id"])
	resp, _ := a.etcdcli.DoGet(path)
	if resp.Count == 0 {
		err := fmt.Errorf("unknown provider id: %s", vars["provider_id"])
		a.respondError(rw, apiError{
			code: http.StatusBadRequest,
			err:  err,
		})
		return
	}
	c := model.Cloud{}
	json.Unmarshal(resp.Kvs[0].Value, &c)

	if err := a.receive(req, &h); err != nil {
		a.respondError(rw, apiError{
			code: http.StatusBadRequest,
			err:  err,
		})
		return
	}

	if err := h.Validate(); err != nil {
		a.respondError(rw, apiError{
			code: http.StatusBadRequest,
			err:  err,
		})
		return
	}

	path = common.Path(model.DefaultHealerPrefix, vars["provider_id"])
	resp, _ = a.etcdcli.DoGet(path, etcdv3.WithPrefix(), etcdv3.WithCountOnly())
	if resp.Count > 0 {
		err := fmt.Errorf("there is only 1 healer can be existed for 1 cloud provider")
		a.respondError(rw, apiError{
			code: http.StatusBadRequest,
			err:  err,
		})
		return
	}

	// Validate query format.
	backend, _ := metrics.GetBackend(a.etcdcli, vars["provider_id"])
	_, err := backend.QueryInstant(req.Context(), h.Query, time.Now())
	if err != nil && strings.Contains(err.Error(), "bad_data") {
		err = fmt.Errorf("invalid query: %s", err.Error())
		a.respondError(rw, apiError{
			code: http.StatusBadRequest,
			err:  err,
		})
		return
	}

	h.ID = common.Hash(c.ID, crypto.MD5)
	h.CloudID = c.ID

	r, _ := json.Marshal(&h)
	_, err = a.etcdcli.DoPut(common.Path(path, h.ID), string(r))
	if err != nil {
		err = fmt.Errorf("error putting a key-value pair into etcd: %s", err.Error())
		a.respondError(rw, apiError{
			code: http.StatusInternalServerError,
			err:  err,
		})
		return
	}
	a.respondSuccess(rw, http.StatusOK, nil)
	return
}

func (a *API) listHealers(rw http.ResponseWriter, req *http.Request) {
	if req.Method == "OPTIONS" {
		return
	}
	vars := mux.Vars(req)
	pid := strings.ToLower(vars["provider_id"])
	path := common.Path(model.DefaultHealerPrefix, pid)

	resp, err := a.etcdcli.DoGet(path, etcdv3.WithPrefix(),
		etcdv3.WithSort(etcdv3.SortByKey, etcdv3.SortAscend))
	if err != nil {
		a.respondError(rw, apiError{
			code: http.StatusInternalServerError,
			err:  err,
		})
		return
	}
	healers := cmap.New()
	for _, e := range resp.Kvs {
		h := model.Healer{}
		_ = json.Unmarshal(e.Value, &h)
		healers.Set(string(e.Key), h)
	}
	a.respondSuccess(rw, http.StatusOK, healers.Items())
	return
}

func (a *API) deleteHealer(w http.ResponseWriter, req *http.Request) {
	if req.Method == "OPTIONS" {
		return
	}
	vars := mux.Vars(req)
	pid := strings.ToLower(vars["provider_id"])
	sid := strings.ToLower(vars["id"])
	path := common.Path(model.DefaultHealerPrefix, pid, sid)
	_, err := a.etcdcli.DoDelete(path, etcdv3.WithPrefix())
	if err != nil {
		a.respondError(w, apiError{
			code: http.StatusInternalServerError,
			err:  err,
		})
		return
	}
	a.respondSuccess(w, http.StatusOK, nil)
	return
}
