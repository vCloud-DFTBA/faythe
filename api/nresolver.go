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
	"io/ioutil"
	"net/http"

	"github.com/go-kit/kit/log/level"
	"github.com/ntk148v/faythe/pkg/model"
	"github.com/ntk148v/faythe/pkg/utils"
	etcdv3 "go.etcd.io/etcd/clientv3"
)

func (a *API) createNResolver(rw http.ResponseWriter, req *http.Request) {
	nr := model.NResolver{}
	err := a.parseNResolverRequest(&nr, req)
	if err != nil {
		err = fmt.Errorf("Error while parsing NResolver object: %s", err.Error())
		a.respondError(rw, apiError{
			code: http.StatusInternalServerError,
			err:  err,
		})
		return
	}
	path := utils.Path(model.DefaultNResolverPrefix, nr.Name)
	resp, _ := a.etcdclient.Get(req.Context(), path, etcdv3.WithCountOnly())
	if resp.Count > 0 {
		err := fmt.Errorf("The address exists", nr.Address.String())
		a.respondError(rw, apiError{
			code: http.StatusBadRequest,
			err:  err,
		})
		return
	}
	b, err := json.Marshal(&nr)
	if err != nil {
		err = fmt.Errorf("Error while serializing NResolver object: %s", err.Error())
		a.respondError(rw, apiError{
			code: http.StatusInternalServerError,
			err:  err,
		})
		return
	}
	_, err = a.etcdclient.Put(req.Context(), path, string(b))
	if err != nil {
		err = fmt.Errorf("Error while putting NResolver object to etcd: %s", err.Error())
		a.respondError(rw, apiError{
			code: http.StatusInternalServerError,
			err:  err,
		})
		return
	}

	a.respondSuccess(rw, http.StatusOK, nil)
}

func (a *API) deleteNResolver(rw http.ResponseWriter, req *http.Request) {
	nr := model.NResolver{}
	err := a.parseNResolverRequest(&nr, req)
	if err != nil {
		err = fmt.Errorf("Error while parsing NResolver object: %s", err.Error())
		a.respondError(rw, apiError{
			code: http.StatusInternalServerError,
			err:  err,
		})
		return
	}
	path := utils.Path(model.DefaultNResolverPrefix, nr.Name)
	_, err = a.etcdclient.Delete(req.Context(), path, etcdv3.WithPrefix())
	if err != nil {
		err = fmt.Errorf("Error while deleting NResolver object to etcd: %s", err.Error())
		a.respondError(rw, apiError{
			code: http.StatusInternalServerError,
			err:  err,
		})
		return
	}
	a.respondSuccess(rw, http.StatusOK, nil)
	return
}

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
				"nrsolvere", e.Key, "err", err)
			continue
		}
		nresolvers[string(e.Key)] = nrt
	}
	a.respondSuccess(rw, http.StatusOK, nresolvers)
	return
}

func (a API) parseNResolverRequest(nr *model.NResolver, req *http.Request) error {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		level.Error(a.logger).Log("msg", "Error while reading NResolver body",
			"err", err)
		return err
	}
	err = json.Unmarshal(body, nr)
	if err != nil {
		level.Error(a.logger).Log("msg", "Error while json-izing NResolver object",
			"err", err)
		return err
	}
	err = nr.Validate()
	if err != nil {
		level.Error(a.logger).Log("msg", "Error while parsing NResolver object",
			"err", err)
		return err
	}
	return nil
}
