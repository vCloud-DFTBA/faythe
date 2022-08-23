// Copyright (c) 2022 Tuan Dat Vu <tuandatk25a@gmail.com>
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
	"time"

	"github.com/gorilla/mux"
	cmap "github.com/orcaman/concurrent-map"
	etcdv3 "go.etcd.io/etcd/clientv3"

	"github.com/vCloud-DFTBA/faythe/pkg/common"
	"github.com/vCloud-DFTBA/faythe/pkg/metrics"
	"github.com/vCloud-DFTBA/faythe/pkg/model"
)

type Worker struct {
	Type     string          `json:"type"`
	Worker   json.RawMessage `json:"worker,omitempty"`
	WorkerID string          `json:"id,omitempty"`
}

type Workers []Worker

func (a *API) bulkCreate(w http.ResponseWriter, req *http.Request) {
	// Save workers object in etcd3
	var (
		workers *Workers
		path    string
		vars    map[string]string
		v       []byte
		force   bool
		res     cmap.ConcurrentMap
	)

	vars = mux.Vars(req)
	pid := vars["provider_id"]
	path = common.Path(model.DefaultCloudPrefix, pid)
	resp, _ := a.etcdcli.DoGet(path, etcdv3.WithCountOnly())
	if resp.Count == 0 {
		err := fmt.Errorf("unknown provider id: %s", pid)
		a.respondError(w, apiError{
			code: http.StatusBadRequest,
			err:  err,
		})
		return
	}

	if err := a.receive(req, &workers); err != nil {
		a.respondError(w, apiError{
			code: http.StatusBadRequest,
			err:  err,
		})
		return
	}

	res = cmap.New()

	for _, worker := range *workers {
		switch worker.Type {
		case "scaler":
			var s model.Scaler
			if err := json.Unmarshal(worker.Worker, &s); err != nil {
				a.respondError(w, apiError{
					code: http.StatusInternalServerError,
					err:  err,
				})
			}
			creator := req.Context().Value("user").(map[string]interface{})
			s.CreatedBy = creator["name"].(string)

			if err := s.Validate(); err != nil {
				a.respondError(w, apiError{
					code: http.StatusBadRequest,
					err:  err,
				})
				return
			}
			s.CloudID = pid
			path = common.Path(model.DefaultScalerPrefix, pid, s.ID)
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
			// Check whether query is syntactically correct
			backend, _ := metrics.GetBackend(a.etcdcli, pid)
			_, err := backend.QueryInstant(req.Context(), s.Query, time.Now())
			if err != nil && strings.Contains(err.Error(), "bad_data") {
				err = fmt.Errorf("invalid query: %s", err.Error())
				a.respondError(w, apiError{
					code: http.StatusBadRequest,
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
			res.Set(path, s)
		case "scheduler":
			var sh model.Scheduler
			if err := json.Unmarshal(worker.Worker, &sh); err != nil {
				a.respondError(w, apiError{
					code: http.StatusInternalServerError,
					err:  err,
				})
				return
			}
			creator := req.Context().Value("user").(map[string]interface{})
			sh.CreatedBy = creator["name"].(string)

			if err := sh.Validate(); err != nil {
				a.respondError(w, apiError{
					code: http.StatusBadRequest,
					err:  err,
				})
				return
			}

			sh.CloudID = pid
			path = common.Path(model.DefaultSchedulerPrefix, pid, sh.ID)
			if strings.ToLower(req.URL.Query().Get("force")) == "true" {
				force = true
			}
			resp, _ = a.etcdcli.DoGet(path, etcdv3.WithCountOnly())
			if resp.Count > 0 && !force {
				err := fmt.Errorf("the scheduler with id %s is existed", sh.ID)
				a.respondError(w, apiError{
					code: http.StatusBadRequest,
					err:  err,
				})
				return
			}

			v, _ = json.Marshal(&sh)
			_, err := a.etcdcli.DoPut(path, string(v))
			if err != nil {
				err = fmt.Errorf("error putting a key-value pair into etcd: %s", err.Error())
				a.respondError(w, apiError{
					code: http.StatusInternalServerError,
					err:  err,
				})
				return
			}
			res.Set(path, sh)
		default:
			err := fmt.Errorf("unknown worker type: %s", worker.Type)
			a.respondError(w, apiError{
				code: http.StatusBadRequest,
				err:  err,
			})
		}
	}

	a.respondSuccess(w, http.StatusOK, res.Items())
}

// Delete multiple workers from etcd3
func (a *API) bulkDelete(w http.ResponseWriter, req *http.Request) {
	var (
		workers *Workers
		path    string
		vars    map[string]string
	)

	vars = mux.Vars(req)
	pid := vars["provider_id"]
	path = common.Path(model.DefaultCloudPrefix, pid)
	resp, _ := a.etcdcli.DoGet(path, etcdv3.WithCountOnly())
	if resp.Count == 0 {
		err := fmt.Errorf("unknown provider id: %s", pid)
		a.respondError(w, apiError{
			code: http.StatusBadRequest,
			err:  err,
		})
		return
	}

	if err := a.receive(req, &workers); err != nil {
		a.respondError(w, apiError{
			code: http.StatusBadRequest,
			err:  err,
		})
		return
	}

	for _, worker := range *workers {
		switch worker.Type {
		case "scaler":
			path = common.Path(model.DefaultScalerPrefix, pid, worker.WorkerID)
			_, err := a.etcdcli.DoDelete(path, etcdv3.WithPrefix())
			if err != nil {
				a.respondError(w, apiError{
					code: http.StatusInternalServerError,
					err:  err,
				})
				return
			}
		case "scheduler":
			path = common.Path(model.DefaultSchedulerPrefix, pid, worker.WorkerID)
			_, err := a.etcdcli.DoDelete(path, etcdv3.WithPrefix())
			if err != nil {
				a.respondError(w, apiError{
					code: http.StatusInternalServerError,
					err:  err,
				})
				return
			}
		default:
			err := fmt.Errorf("unknown worker type: %s", worker.Type)
			a.respondError(w, apiError{
				code: http.StatusBadRequest,
				err:  err,
			})
		}
	}

	a.respondSuccess(w, http.StatusOK, nil)
}

func (a *API) bulkUpdate(w http.ResponseWriter, req *http.Request) {
	var (
		workers *Workers
		path    string
		vars    map[string]string
		v       []byte
		res     cmap.ConcurrentMap
	)

	vars = mux.Vars(req)
	pid := vars["provider_id"]
	path = common.Path(model.DefaultCloudPrefix, pid)
	resp, _ := a.etcdcli.DoGet(path, etcdv3.WithCountOnly())
	if resp.Count == 0 {
		err := fmt.Errorf("unknown provider id: %s", pid)
		a.respondError(w, apiError{
			code: http.StatusBadRequest,
			err:  err,
		})
		return
	}

	if err := a.receive(req, &workers); err != nil {
		a.respondError(w, apiError{
			code: http.StatusBadRequest,
			err:  err,
		})
		return
	}

	res = cmap.New()

	for _, worker := range *workers {
		switch worker.Type {
		case "scaler":
			// Delete Scaler in Etcd
			path = common.Path(model.DefaultScalerPrefix, pid, worker.WorkerID)
			_, err := a.etcdcli.DoDelete(path, etcdv3.WithPrefix())
			if err != nil {
				a.respondError(w, apiError{
					code: http.StatusInternalServerError,
					err:  err,
				})
				return
			}

			var s model.Scaler
			if err := json.Unmarshal(worker.Worker, &s); err != nil {
				a.respondError(w, apiError{
					code: http.StatusInternalServerError,
					err:  err,
				})
			}

			creator := req.Context().Value("user").(map[string]interface{})
			s.CreatedBy = creator["name"].(string)

			if err := s.Validate(); err != nil {
				a.respondError(w, apiError{
					code: http.StatusBadRequest,
					err:  err,
				})
				return
			}
			s.CloudID = pid

			// Override ScalerID with old ID
			s.ID = worker.WorkerID

			// Check whether query is syntactically correct
			backend, _ := metrics.GetBackend(a.etcdcli, pid)
			_, err = backend.QueryInstant(req.Context(), s.Query, time.Now())
			if err != nil && strings.Contains(err.Error(), "bad_data") {
				err = fmt.Errorf("invalid query: %s", err.Error())
				a.respondError(w, apiError{
					code: http.StatusBadRequest,
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
			res.Set(path, s)
		case "scheduler":
			// Delete Scheduler in Etcd
			path = common.Path(model.DefaultSchedulerPrefix, pid, worker.WorkerID)
			_, err := a.etcdcli.DoDelete(path, etcdv3.WithPrefix())
			if err != nil {
				a.respondError(w, apiError{
					code: http.StatusInternalServerError,
					err:  err,
				})
				return
			}

			var sh model.Scheduler
			if err := json.Unmarshal(worker.Worker, &sh); err != nil {
				a.respondError(w, apiError{
					code: http.StatusInternalServerError,
					err:  err,
				})
				return
			}
			creator := req.Context().Value("user").(map[string]interface{})
			sh.CreatedBy = creator["name"].(string)

			if err := sh.Validate(); err != nil {
				a.respondError(w, apiError{
					code: http.StatusBadRequest,
					err:  err,
				})
				return
			}
			sh.CloudID = pid

			// Override Scheduler ID with old ID
			sh.ID = worker.WorkerID

			v, _ = json.Marshal(&sh)
			_, err = a.etcdcli.DoPut(path, string(v))
			if err != nil {
				err = fmt.Errorf("error putting a key-value pair into etcd: %s", err.Error())
				a.respondError(w, apiError{
					code: http.StatusInternalServerError,
					err:  err,
				})
				return
			}
			res.Set(path, sh)
		default:
			err := fmt.Errorf("unknown worker type: %s", worker.Type)
			a.respondError(w, apiError{
				code: http.StatusBadRequest,
				err:  err,
			})
		}
	}

	a.respondSuccess(w, http.StatusOK, res.Items())
}
