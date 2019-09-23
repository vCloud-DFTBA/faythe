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
	"github.com/go-kit/kit/log/level"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	etcdv3 "go.etcd.io/etcd/clientv3"

	"github.com/ntk148v/faythe/pkg/model"
)

func (a *API) registerCloud(w http.ResponseWriter, req *http.Request) {
	var (
		vars  map[string]string
		p     string
		ops   *model.OpenStack
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
				code: http.StatusInternalServerError,
				err:  err,
			})
			return
		}
		k = fmt.Sprintf("%s/%s", model.DefaultOpenStackPrefix, ops.ID)
		if strings.ToLower(req.URL.Query().Get("force")) == "true" {
			force = true
		}
		resp, _ := a.etcdclient.Get(req.Context(), k, etcdv3.WithCountOnly())
		if resp.Count > 0 && !force {
			err := fmt.Errorf("The provider with id %s is existed", ops.ID)
			a.respondError(w, apiError{
				code: http.StatusBadRequest,
				err:  err,
			})
			return
		}

		v, _ = json.Marshal(&ops)
		_, err := a.etcdclient.Put(req.Context(), k, string(v))
		if err != nil {
			err = fmt.Errorf("Error putting a key-value pair into etcd: %s", err.Error())
			a.respondError(w, apiError{
				code: http.StatusInternalServerError,
				err:  err,
			})
			return
		}

		a.respondSuccess(w, http.StatusOK, nil)
	default:
		err := fmt.Errorf("The provider %s is unsupported", p)
		a.respondError(w, apiError{
			code: http.StatusBadRequest,
			err:  err,
		})
		return
	}
}

func (a *API) listClouds(w http.ResponseWriter, req *http.Request) {
	// Get all current clouds information from etcd3
	var (
		vars   map[string]string
		p      string
		clouds map[string]model.OpenStack
	)
	vars = mux.Vars(req)
	p = strings.ToLower(vars["provider"])
	switch p {
	case "openstack":
		resp, err := a.etcdclient.Get(req.Context(), model.DefaultOpenStackPrefix,
			etcdv3.WithPrefix(), etcdv3.WithSort(etcdv3.SortByKey, etcdv3.SortAscend))
		if err != nil {
			a.respondError(w, apiError{
				code: http.StatusInternalServerError,
				err:  err,
			})
			return
		}

		clouds = make(map[string]model.OpenStack, len(resp.Kvs))

		for _, ev := range resp.Kvs {
			var cloud model.OpenStack
			err = json.Unmarshal(ev.Value, &cloud)
			if err != nil {
				level.Error(a.logger).Log("msg", "Error getting cloud from etcd",
					"cloud", ev.Key, "err", err)
				continue
			}
			clouds[string(ev.Key)] = cloud
		}
		a.respondSuccess(w, http.StatusOK, clouds)
		return
	default:
		err := fmt.Errorf("The provider %s is unsupported", p)
		a.respondError(w, apiError{
			code: http.StatusBadRequest,
			err:  err,
		})
		return
	}
}

func (a *API) unregisterCloud(w http.ResponseWriter, req *http.Request) {
	// Remove the cloud information from etcd3
}

func (a *API) updateCloud(w http.ResponseWriter, req *http.Request) {
	// Update the existing cloud information
}
