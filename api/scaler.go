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
	"sync"
	"time"

	"github.com/gorilla/mux"
	etcdv3 "go.etcd.io/etcd/clientv3"

	"github.com/vCloud-DFTBA/faythe/pkg/common"
	"github.com/vCloud-DFTBA/faythe/pkg/metrics"
	"github.com/vCloud-DFTBA/faythe/pkg/model"
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
	path = common.Path(model.DefaultCloudPrefix, vars["provider_id"])
	resp, _ := a.etcdcli.DoGet(path, etcdv3.WithCountOnly())
	if resp.Count == 0 {
		err := fmt.Errorf("unknown provider id: %s", vars["provider_id"])
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
	path = common.Path(model.DefaultScalerPrefix, vars["provider_id"], s.ID)
	if strings.ToLower(req.URL.Query().Get("force")) == "true" {
		force = true
	}
	resp, _ = a.etcdcli.DoGet(path, etcdv3.WithCountOnly())
	if resp.Count > 0 && !force {
		err := fmt.Errorf("the scaler with id %s is existed", s.ID)
		a.respondError(w, apiError{
			code: http.StatusBadRequest,
			err:  err,
		})
		return
	}

	// Validate query format.
	backend, _ := metrics.GetBackend(a.etcdcli, vars["provider_id"])
	_, err := backend.QueryInstant(req.Context(), s.Query, time.Now())
	if err != nil && strings.Contains(err.Error(), "bad_data") {
		err = fmt.Errorf("invalid query: %s", err.Error())
		a.respondError(w, apiError{
			code: http.StatusInternalServerError,
			err:  err,
		})
		return
	}

	v, _ = json.Marshal(&s)
	_, err = a.etcdcli.DoPut(path, string(v))
	if err != nil {
		err = fmt.Errorf("error putting a key-value pair into etcd: %s", err.Error())
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
		wg      sync.WaitGroup
	)
	vars = mux.Vars(req)
	pid = strings.ToLower(vars["provider_id"])
	path = common.Path(model.DefaultScalerPrefix, pid)
	resp, err := a.etcdcli.DoGet(path, etcdv3.WithPrefix(),
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
		wg.Add(1)
		go func(evv []byte, evk string) {
			defer wg.Done()
			var s model.Scaler
			_ = json.Unmarshal(evv, &s)
			// Filter
			// Clouds that match all tags in this list will be returned
			if fTags := req.FormValue("tags"); fTags != "" {
				tags := strings.Split(fTags, ",")
				if !common.Find(s.Tags, tags, "and") {
					return
				}
			}
			// Clouds that match any tags in this list will be returned
			if fTagsAny := req.FormValue("tags-any"); fTagsAny != "" {
				tags := strings.Split(fTagsAny, ",")
				if !common.Find(s.Tags, tags, "or") {
					return
				}
			}
			scalers[evk] = s
		}(ev.Value, string(ev.Key))
	}
	wg.Wait()
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
	path = common.Path(model.DefaultScalerPrefix, pid, sid)
	resp, err := a.etcdcli.DoDelete(path, etcdv3.WithPrefix())
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
