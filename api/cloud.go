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

	"github.com/gorilla/mux"
	cmap "github.com/orcaman/concurrent-map"
	etcdv3 "go.etcd.io/etcd/clientv3"

	"github.com/vCloud-DFTBA/faythe/pkg/cloud/store/opensourcemano"
	"github.com/vCloud-DFTBA/faythe/pkg/cloud/store/openstack"
	"github.com/vCloud-DFTBA/faythe/pkg/common"
	"github.com/vCloud-DFTBA/faythe/pkg/metrics"
	"github.com/vCloud-DFTBA/faythe/pkg/model"
)

func (a *API) registerCloud(w http.ResponseWriter, req *http.Request) {
	var (
		vars  map[string]string
		p     string
		ops   *model.OpenStack
		osm   *model.OpenSourceMano
		k     string
		v     []byte
		force bool
	)
	vars = mux.Vars(req)
	p = strings.ToLower(vars["provider"])
	switch p {
	case model.OpenStackType:
		if err := a.receive(req, &ops); err != nil {
			a.respondError(w, apiError{
				code: http.StatusBadRequest,
				err:  err,
			})
			return
		}
		if err := ops.Validate(); err != nil {
			a.respondError(w, apiError{
				code: http.StatusBadRequest,
				err:  err,
			})
			return
		}

		creator := req.Context().Value("user").(map[string]interface{})
		ops.CreatedBy = creator["name"].(string)

		k = common.Path(model.DefaultCloudPrefix, ops.ID)
		if strings.ToLower(req.URL.Query().Get("force")) == "true" {
			force = true
		}
		resp, _ := a.etcdcli.DoGet(k, etcdv3.WithCountOnly())
		if resp.Count > 0 && !force {
			err := fmt.Errorf("the provider with id %s is existed", ops.ID)
			a.respondError(w, apiError{
				code: http.StatusBadRequest,
				err:  err,
			})
			return
		}

		// Register Backend to registry
		err := metrics.Register(ops.Monitor.Backend, string(ops.Monitor.Address),
			ops.Monitor.Username, ops.Monitor.Password.Token)
		if err = ops.Monitor.Password.Encrypt(); err != nil {
			a.respondError(w, apiError{
				code: http.StatusBadRequest,
				err:  fmt.Errorf("cannot encrypt password %s", err.Error()),
			})
			return
		}

		v, _ = json.Marshal(&ops)
		_, err = a.etcdcli.DoPut(k, string(v))
		if err != nil {
			err = fmt.Errorf("error putting a key-value pair into etcd: %s", err.Error())
			a.respondError(w, apiError{
				code: http.StatusInternalServerError,
				err:  err,
			})
			return
		}

		// Set cloud to Store
		store := openstack.Get()
		store.Set(ops.ID, *ops)

		a.respondSuccess(w, http.StatusOK, nil)
		return
	case model.ManoType:
		if err := a.receive(req, &osm); err != nil {
			a.respondError(w, apiError{
				code: http.StatusBadRequest,
				err:  err,
			})
			return
		}
		if err := osm.Validate(); err != nil {
			a.respondError(w, apiError{
				code: http.StatusBadRequest,
				err:  err,
			})
			return
		}
		creator := req.Context().Value("user").(map[string]interface{})
		osm.CreatedBy = creator["name"].(string)

		k = common.Path(model.DefaultCloudPrefix, osm.ID)
		if strings.ToLower(req.URL.Query().Get("force")) == "true" {
			force = true
		}
		resp, _ := a.etcdcli.DoGet(k, etcdv3.WithCountOnly())
		if resp.Count > 0 && !force {
			err := fmt.Errorf("the provider with id %s is existed", osm.ID)
			a.respondError(w, apiError{
				code: http.StatusBadRequest,
				err:  err,
			})
			return
		}
		// Register Backend to registry
		err := metrics.Register(osm.Monitor.Backend, string(osm.Monitor.Address),
			osm.Monitor.Username, osm.Monitor.Password.Token)
		if err != nil {
			a.respondError(w, apiError{
				code: http.StatusBadRequest,
				err:  fmt.Errorf("cannot connect to backend: %s", err.Error()),
			})
			return
		}

		v, _ = json.Marshal(&osm)
		_, err = a.etcdcli.DoPut(k, string(v))
		if err != nil {
			err = fmt.Errorf("error putting a key-value pair into etcd: %s", err.Error())
			a.respondError(w, apiError{
				code: http.StatusInternalServerError,
				err:  err,
			})
			return
		}

		// Set cloud to Store
		store := opensourcemano.Get()
		store.Set(osm.ID, *osm)

		a.respondSuccess(w, http.StatusOK, nil)
		return
	}
}

// Get all current clouds information from etcdv3
func (a *API) listClouds(w http.ResponseWriter, req *http.Request) {
	var (
		clouds cmap.ConcurrentMap
		wg     sync.WaitGroup
	)
	resp, err := a.etcdcli.DoGet(model.DefaultCloudPrefix, etcdv3.WithPrefix(),
		etcdv3.WithSort(etcdv3.SortByKey, etcdv3.SortAscend))
	if err != nil {
		a.respondError(w, apiError{
			code: http.StatusInternalServerError,
			err:  err,
		})
		return
	}

	clouds = cmap.New()
	for _, ev := range resp.Kvs {
		wg.Add(1)
		go func(evv []byte, evk string) {
			defer wg.Done()
			var cloud model.Cloud
			_ = json.Unmarshal(evv, &cloud)
			// Filter
			if p := strings.ToLower(req.FormValue("provider")); p != "" && p != cloud.Provider {
				return
			}
			if id := strings.ToLower(req.FormValue("id")); id != "" && id != cloud.ID {
				return
			}
			// Clouds that match all tags in this list will be returned
			if fTags := req.FormValue("tags"); fTags != "" {
				tags := strings.Split(fTags, ",")
				if !common.Find(cloud.Tags, tags, "and") {
					return
				}
			}
			// Clouds that match any tags in this list will be returned
			if fTagsAny := req.FormValue("tags-any"); fTagsAny != "" {
				tags := strings.Split(fTagsAny, ",")
				if !common.Find(cloud.Tags, tags, "or") {
					return
				}
			}
			clouds.Set(evk, cloud)
			switch cloud.Provider {
			case model.OpenStackType:
				var ops model.OpenStack
				_ = json.Unmarshal(evv, &ops)
				clouds.Set(evk, ops)
			case model.ManoType:
				var osm model.OpenSourceMano
				_ = json.Unmarshal(evv, &osm)
				clouds.Set(evk, osm)
			}
		}(ev.Value, string(ev.Key))
	}
	wg.Wait()
	a.respondSuccess(w, http.StatusOK, clouds.Items())
}

// Remove the cloud information from etcd3
func (a *API) unregisterCloud(w http.ResponseWriter, req *http.Request) {
	var (
		vars map[string]string
		pid  string
		path string
	)
	vars = mux.Vars(req)
	pid = strings.ToLower(vars["id"])
	path = common.Path(model.DefaultCloudPrefix, pid)
	_, err := a.etcdcli.DoDelete(path, etcdv3.WithPrefix())
	if err != nil {
		a.respondError(w, apiError{
			code: http.StatusInternalServerError,
			err:  err,
		})
		return
	}

	// Remove cloud from store
	store := openstack.Get()
	store.Delete(pid)

	scalerPath := common.Path(model.DefaultScalerPrefix, pid)
	_, err = a.etcdcli.DoDelete(scalerPath, etcdv3.WithPrefix())
	if err != nil {
		a.respondError(w, apiError{
			code: http.StatusInternalServerError,
			err:  err,
		})
		return
	}
	a.respondSuccess(w, http.StatusOK, nil)
}

func (a *API) updateCloud(w http.ResponseWriter, req *http.Request) {
	// Update the existing cloud information
}
