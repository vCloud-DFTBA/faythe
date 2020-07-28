// Copyright (c) 2020 Dat Vu Tuan <tuandatk25a@gmail.com>
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
	cmap "github.com/orcaman/concurrent-map"
	etcdv3 "go.etcd.io/etcd/clientv3"

	"github.com/vCloud-DFTBA/faythe/pkg/history"
)

func (a *API) listActionHistory(rw http.ResponseWriter, req *http.Request) {
	resp, err := a.etcdcli.DoGet(history.DefaultActionHistoryPrefix, etcdv3.WithPrefix(),
		etcdv3.WithSort(etcdv3.SortByKey, etcdv3.SortAscend))
	if err != nil {
		err = fmt.Errorf("Error while getting history objects from etcdv3: %s", err.Error())
		a.respondError(rw, apiError{
			code: http.StatusInternalServerError,
			err:  err,
		})
		return
	}
	actionHistories := cmap.New()
	for _, e := range resp.Kvs {
		actHis := history.ActionHistory{}
		err = json.Unmarshal(e.Value, &actHis)
		if err != nil {
			level.Error(a.logger).Log("msg", "Error getting action history from etcd", "err", err.Error())
			continue
		}
		actionHistories.Set(string(e.Key), actHis)
	}
	a.respondSuccess(rw, http.StatusOK, actionHistories.Items())
	return
}
