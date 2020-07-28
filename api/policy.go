// Copyright (c) 2020 Kien Nguyen-Tuan <kiennt2609@gmail.com>
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
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"

	"github.com/vCloud-DFTBA/faythe/pkg/common"
	"github.com/vCloud-DFTBA/faythe/pkg/model"
)

// addPolicies allows to add more than one policys at once.
func (a *API) addPolicies(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	user := vars["user"]
	// Check an user is existing.
	path := common.Path(model.DefaultUsersPrefix, common.Hash(user, crypto.MD5))
	resp, err := a.etcdcli.DoGet(path)
	if err != nil {
		a.respondError(w, apiError{
			code: http.StatusInternalServerError,
			err:  err,
		})
		return
	}
	if len(resp.Kvs) == 0 {
		a.respondError(w, apiError{
			code: http.StatusBadRequest,
			err:  errors.New("Unknown user"),
		})
		return
	}

	var (
		pols  model.Polices
		rules [][]string
	)
	if err := a.receive(req, &pols); err != nil {
		a.respondError(w, apiError{
			code: http.StatusBadRequest,
			err:  err,
		})
		return
	}
	for _, p := range pols {
		if err := p.Validate(); err != nil {
			a.respondError(w, apiError{
				code: http.StatusBadRequest,
				err:  err,
			})
			return
		}
		rules = append(rules, []string{user, p.Path, p.Method})
	}
	// Add new policy to Etcd
	if _, err := a.policyEngine.AddPolicies(rules); err != nil {
		a.respondError(w, apiError{
			code: http.StatusInternalServerError,
			err:  err,
		})
		return
	}
	if err := a.policyEngine.SavePolicy(); err != nil {
		a.respondError(w, apiError{
			code: http.StatusInternalServerError,
			err:  err,
		})
		return
	}
	a.respondSuccess(w, http.StatusOK, nil)
}

// removePolicies allows to remove more than one policys at once.
func (a *API) removePolicies(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	user := vars["user"]
	// Check an user is existing.
	path := common.Path(model.DefaultUsersPrefix, common.Hash(user, crypto.MD5))
	resp, err := a.etcdcli.DoGet(path)
	if err != nil {
		a.respondError(w, apiError{
			code: http.StatusInternalServerError,
			err:  err,
		})
		return
	}
	if len(resp.Kvs) == 0 {
		a.respondError(w, apiError{
			code: http.StatusBadRequest,
			err:  errors.New("Unknown user"),
		})
		return
	}
	var (
		pols  model.Polices
		rules [][]string
	)
	if err := a.receive(req, &pols); err != nil {
		a.respondError(w, apiError{
			code: http.StatusBadRequest,
			err:  err,
		})
		return
	}
	for _, p := range pols {
		rules = append(rules, []string{user, p.Path, p.Method})
	}
	// Remove policy from Etcd
	if _, err := a.policyEngine.RemovePolicies(rules); err != nil {
		a.respondError(w, apiError{
			code: http.StatusInternalServerError,
			err:  err,
		})
		return
	}
	if err := a.policyEngine.SavePolicy(); err != nil {
		a.respondError(w, apiError{
			code: http.StatusInternalServerError,
			err:  err,
		})
		return
	}
	a.respondSuccess(w, http.StatusOK, nil)
}
