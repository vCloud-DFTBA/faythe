// Copyright (c) 2019 Tuan-Dat Vu<tuandatk25a@gmail.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
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
	cmap "github.com/orcaman/concurrent-map"
	etcdv3 "go.etcd.io/etcd/clientv3"

	"github.com/vCloud-DFTBA/faythe/pkg/common"
	"github.com/vCloud-DFTBA/faythe/pkg/model"
)

func (a *API) createSilence(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	pid := strings.ToLower(vars["provider_id"])

	path := common.Path(model.DefaultCloudPrefix, pid)
	resp, err := a.etcdcli.DoGet(path)
	if err != nil {
		a.respondError(rw, apiError{code: http.StatusInternalServerError, err: err})
		return
	}

	if resp.Count == 0 {
		a.respondError(rw, apiError{
			code: http.StatusBadRequest,
			err:  fmt.Errorf("unknown provider_id: %s", pid),
		})
		return
	}

	s := &model.Silence{}
	if err := a.receive(req, &s); err != nil {
		a.respondError(rw, apiError{code: http.StatusBadRequest, err: err})
		return
	}

	if err := s.Validate(); err != nil {
		a.respondError(rw, apiError{code: http.StatusBadRequest, err: err})
		return
	}

	creator := req.Context().Value("user").(map[string]interface{})
	s.CreatedBy = creator["name"].(string)

	path = common.Path(model.DefaultSilencePrefix, pid, s.ID)
	resp, err = a.etcdcli.DoGet(path, etcdv3.WithCountOnly())
	if err != nil {
		a.respondError(rw, apiError{code: http.StatusInternalServerError, err: err})
		return
	}
	if resp.Count > 0 {
		a.respondError(rw, apiError{
			code: http.StatusBadRequest,
			err:  fmt.Errorf("there is an exist silence with the exact the same pattern and expiration time"),
		})
		return
	}

	t, _ := common.ParseDuration(s.TTL)

	r, err := a.etcdcli.DoGrant(int64(t.Seconds()))
	if err != nil {
		a.respondError(rw, apiError{
			code: http.StatusInternalServerError,
			err:  fmt.Errorf("error while getting grant for silence: %s", err),
		})
		return
	}

	raw, _ := json.Marshal(&s)
	if _, err := a.etcdcli.DoPut(path, string(raw), etcdv3.WithLease(r.ID)); err != nil {
		a.respondError(rw, apiError{
			code: http.StatusInternalServerError,
			err:  fmt.Errorf("error putting a key-value pair into etcd: %s", err.Error()),
		})
		return
	}

	a.respondSuccess(rw, http.StatusOK, nil)
}

func (a *API) listSilences(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	pid := strings.ToLower(vars["provider_id"])
	path := common.Path(model.DefaultSilencePrefix, pid)

	resp, err := a.etcdcli.DoGet(path, etcdv3.WithPrefix(),
		etcdv3.WithSort(etcdv3.SortByKey, etcdv3.SortAscend))
	if err != nil {
		a.respondError(rw, apiError{
			code: http.StatusInternalServerError,
			err:  err,
		})
		return
	}
	silences := cmap.New()
	for _, e := range resp.Kvs {
		s := model.Silence{}
		_ = json.Unmarshal(e.Value, &s)
		silences.Set(string(e.Key), s)
	}
	a.respondSuccess(rw, http.StatusOK, silences.Items())
}

func (a *API) expireSilence(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	pid := strings.ToLower(vars["provider_id"])
	id := strings.ToLower(vars["id"])
	path := common.Path(model.DefaultSilencePrefix, pid, id)
	_, err := a.etcdcli.DoDelete(path, etcdv3.WithPrefix())
	if err != nil {
		a.respondError(w, apiError{
			code: http.StatusInternalServerError,
			err:  err,
		})
		return
	}
	a.respondSuccess(w, http.StatusOK, nil)
}
