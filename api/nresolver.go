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
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/log/level"
	etcdv3 "go.etcd.io/etcd/clientv3"

	"github.com/vCloud-DFTBA/faythe/pkg/model"
)

func (a *API) listNResolvers(rw http.ResponseWriter, req *http.Request) {
	resp, err := a.etcdclient.Get(req.Context(), model.DefaultNResolverPrefix, etcdv3.WithPrefix(),
		etcdv3.WithSort(etcdv3.SortByKey, etcdv3.SortAscend))
	if err != nil {
		err = fmt.Errorf("Error while getting nresolvers objects from etcdv3: %s", err.Error())
		a.respondError(rw, apiError{
			code: http.StatusInternalServerError,
			err:  err,
		})
		return
	}
	nresolvers := make(map[string]model.NResolver, len(resp.Kvs))
	for _, e := range resp.Kvs {
		nrt := model.NResolver{}
		err = json.Unmarshal(e.Value, &nrt)
		if err != nil {
			level.Error(a.logger).Log("msg", "Error getting nresolver from etcd",
				"nrsolvere", e.Key, "err", err.Error())
			continue
		}
		nresolvers[string(e.Key)] = nrt
	}
	a.respondSuccess(rw, http.StatusOK, nresolvers)
	return
}
