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
	"github.com/gorilla/mux"
	cmap "github.com/orcaman/concurrent-map"
	etcdv3 "go.etcd.io/etcd/clientv3"
	"net/http"
	"strings"

	"github.com/vCloud-DFTBA/faythe/pkg/common"
	"github.com/vCloud-DFTBA/faythe/pkg/model"
)

func (a *API) createScheduler(w http.ResponseWriter, req *http.Request) {
	// Save a Scheduler object in etcd3
	var (
		sh    *model.Scheduler
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

	if err := a.receive(req, &sh); err != nil {
		a.respondError(w, apiError{
			code: http.StatusBadRequest,
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

	sh.CloudID = vars["provider_id"]
	path = common.Path(model.DefaultSchedulerPrefix, vars["provider_id"], sh.ID)
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

	a.respondSuccess(w, http.StatusOK, sh)
}

// List all current Schedulers from etcd3
func (a *API) listSchedulers(w http.ResponseWriter, req *http.Request) {
	var (
		vars       map[string]string
		pid        string
		path       string
		schedulers cmap.ConcurrentMap
	)

	vars = mux.Vars(req)
	pid = strings.ToLower(vars["provider_id"])
	path = common.Path(model.DefaultSchedulerPrefix, pid)
	resp, err := a.etcdcli.DoGet(path, etcdv3.WithPrefix(),
		etcdv3.WithSort(etcdv3.SortByKey, etcdv3.SortAscend))
	if err != nil {
		a.respondError(w, apiError{
			code: http.StatusInternalServerError,
			err:  err,
		})
		return
	}

	schedulers = cmap.New()
	fTags := req.FormValue("tags")
	fTagsAny := req.FormValue("tags-any")
	for _, ev := range resp.Kvs {
		var sh model.Scheduler
		_ = json.Unmarshal(ev.Value, &sh)
		if sh.CloudID == "" {
			sh.CloudID = pid
		}
		// Filter
		// Schedulers that match all tags in this list will be returned
		if fTags != "" {
			tags := strings.Split(fTags, ",")
			if !common.Find(sh.Tags, tags, "and") {
				continue
			}
		}
		// Clouds that match any tags in this list will be returned
		if fTagsAny != "" {
			tags := strings.Split(fTagsAny, ",")
			if !common.Find(sh.Tags, tags, "or") {
				continue
			}
		}
		schedulers.Set(string(ev.Key), sh)
	}
	a.respondSuccess(w, http.StatusOK, schedulers.Items())
}

// Delete a Scheduler from etcd3
func (a *API) deleteScheduler(w http.ResponseWriter, req *http.Request) {
	var (
		vars map[string]string
		pid  string
		shid string
		path string
	)

	vars = mux.Vars(req)
	pid = strings.ToLower(vars["provider_id"])
	shid = strings.ToLower(vars["id"])
	path = common.Path(model.DefaultSchedulerPrefix, pid, shid)
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
}

func (a *API) updateScheduler(w http.ResponseWriter, req *http.Request) {
	// Update an existed Scheduler information
	var (
		vars map[string]string
		pid  string
		shid string
		path string
		sh   *model.Scheduler
		v    []byte
	)

	vars = mux.Vars(req)
	pid = strings.ToLower(vars["provider_id"])
	shid = strings.ToLower(vars["id"])
	path = common.Path(model.DefaultSchedulerPrefix, pid, shid)

	// Delete Scheduler in Etcd
	_, err := a.etcdcli.DoDelete(path, etcdv3.WithPrefix())
	if err != nil {
		a.respondError(w, apiError{
			code: http.StatusInternalServerError,
			err:  err,
		})
		return
	}

	// Update with new data
	if err := a.receive(req, &sh); err != nil {
		a.respondError(w, apiError{
			code: http.StatusBadRequest,
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
	sh.ID = shid

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

	a.respondSuccess(w, http.StatusOK, sh)
}
