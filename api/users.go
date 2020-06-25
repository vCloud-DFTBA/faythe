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
	"encoding/json"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
	cmap "github.com/orcaman/concurrent-map"
	"github.com/pkg/errors"
	etcdv3 "go.etcd.io/etcd/clientv3"

	"github.com/vCloud-DFTBA/faythe/config"
	"github.com/vCloud-DFTBA/faythe/pkg/common"
	"github.com/vCloud-DFTBA/faythe/pkg/model"
)

func (a *API) createAdminUser() error {
	// Check to see if the user is already taken
	authCfg := config.Get().AdminAuthentication
	path := common.Path(model.DefaultUsersPrefix, common.Hash(authCfg.Username, crypto.MD5))
	resp, err := a.etcdcli.DoGet(path)
	if err != nil {
		return err
	}
	var user model.User
	if len(resp.Kvs) > 0 {
		// Get only the first item
		_ = json.Unmarshal(resp.Kvs[0].Value, &user)
		if ok := common.CheckPasswordAgainstHash(authCfg.Password, user.Password); !ok {
			return errors.New("An user is existing but the given password is wrong")
		}
	} else {
		// Do not store the plain text password, encrypt it!
		hashed, err := common.GenerateBcryptHash(authCfg.Password, config.Get().PasswordHashingCost)
		if err != nil {
			return err
		}
		user.Username = authCfg.Username
		user.Password = hashed
		_ = user.Validate()
		r, _ := json.Marshal(&user)
		_, err = a.etcdcli.DoPut(path, string(r))
		if err != nil {
			return err
		}
	}
	// Add admin permissions
	if ok, err := a.policyEngine.AddPolicy(authCfg.Username, "/*", "(GET)|(POST)|(DELETE)|(PUT)"); !ok || err != nil {
		return err
	}
	return nil
}

// addUser creates a new user
func (a *API) addUser(w http.ResponseWriter, req *http.Request) {
	// NOTE(kiennt): Who can signup (create new user)?
	if err := req.ParseForm(); err != nil {
		a.respondError(w, apiError{
			code: http.StatusBadRequest,
			err:  err,
		})
		return
	}
	username := req.Form.Get("username")
	password := req.Form.Get("password")
	if username == "" || password == "" {
		a.respondError(w, apiError{
			code: http.StatusBadRequest,
			err:  errors.New("Incorrect sign up form"),
		})
		return
	}

	// Check to see if the user is already taken
	path := common.Path(model.DefaultUsersPrefix, common.Hash(username, crypto.MD5))
	resp, err := a.etcdcli.DoGet(path)
	if err != nil {
		a.respondError(w, apiError{
			code: http.StatusInternalServerError,
			err:  err,
		})
		return
	}
	if len(resp.Kvs) != 0 {
		a.respondError(w, apiError{
			code: http.StatusBadRequest,
			err:  errors.New("The username is already taken"),
		})
		return
	}
	// Do not store the plain text password, encrypt it!
	hashed, err := common.GenerateBcryptHash(password, config.Get().PasswordHashingCost)
	if err != nil {
		a.respondError(w, apiError{
			code: http.StatusInternalServerError,
			err:  errors.Wrap(err, "Something went wrong"),
		})
		return
	}

	user := &model.User{
		Username: username,
		Password: hashed,
	}
	_ = user.Validate()
	r, _ := json.Marshal(&user)
	_, err = a.etcdcli.DoPut(path, string(r))
	if err != nil {
		a.respondError(w, apiError{
			code: http.StatusInternalServerError,
			err:  errors.Wrap(err, "Unable to put a key-value pair into etcd"),
		})
		return
	}
	// Add user permission to view clouds
	if ok, err := a.policyEngine.AddPolicy(username, "/clouds", "GET"); !ok || err != nil {
		a.respondError(w, apiError{
			code: http.StatusInternalServerError,
			err:  errors.Wrap(err, "Unable to add view cloud permission"),
		})
		return
	}
	a.respondSuccess(w, http.StatusOK, nil)
	return
}

// removeUser deletes an user
func (a *API) removeUser(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	username := vars["user"]
	// Prevent mistakenly deleting administrator
	if username == config.Get().AdminAuthentication.Username {
		a.respondError(w, apiError{
			code: http.StatusForbidden,
			err:  errors.New("Cannot remove administrator"),
		})
		return
	}
	// Check an user is existing.
	path := common.Path(model.DefaultUsersPrefix, common.Hash(username, crypto.MD5))
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
	_, err = a.etcdcli.DoDelete(path, etcdv3.WithPrefix())
	if err != nil {
		a.respondError(w, apiError{
			code: http.StatusInternalServerError,
			err:  err,
		})
		return
	}
	// Get & remove user permissions
	rules := a.policyEngine.GetFilteredPolicy(0, username)
	if ok, err := a.policyEngine.RemovePolicies(rules); !ok || err != nil {
		a.respondError(w, apiError{
			code: http.StatusInternalServerError,
			err:  errors.Wrap(err, "Unable to remove user associated permissions"),
		})
		return
	}
	a.respondSuccess(w, http.StatusOK, nil)
	return
}

// listUsers returns a list of current Faythe users with associated policies.
func (a *API) listUsers(w http.ResponseWriter, req *http.Request) {
	var (
		path  string
		users cmap.ConcurrentMap
		wg    sync.WaitGroup
	)
	// Force reload policy to get the newest
	_ = a.policyEngine.LoadPolicy()
	if username := req.FormValue("name"); username != "" {
		path = common.Path(model.DefaultUsersPrefix, common.Hash(username, crypto.MD5))
	} else {
		path = common.Path(model.DefaultUsersPrefix)
	}
	resp, err := a.etcdcli.DoGet(path, etcdv3.WithPrefix(),
		etcdv3.WithSort(etcdv3.SortByKey, etcdv3.SortAscend))
	if err != nil {
		a.respondError(w, apiError{
			code: http.StatusInternalServerError,
			err:  err,
		})
		return
	}
	users = cmap.New()
	for _, ev := range resp.Kvs {
		wg.Add(1)
		go func(evv []byte) {
			defer wg.Done()
			var (
				u model.User
				p [][]string
			)
			_ = json.Unmarshal(evv, &u)
			p = a.policyEngine.GetFilteredPolicy(0, u.Username)
			for i, v := range p {
				// The first element is username, so just remove it.
				p[i] = v[1:]
			}
			users.Set(u.Username, p)
		}(ev.Value)
	}
	wg.Wait()
	a.respondSuccess(w, http.StatusOK, users)
	return
}

// changePassword updates the new password for a given user
func (a *API) changePassword(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	username := vars["user"]
	if username == config.Get().AdminAuthentication.Username {
		a.respondError(w, apiError{
			code: http.StatusForbidden,
			err:  errors.New("Cannot change administrator's password"),
		})
		return
	}
	if err := req.ParseForm(); err != nil {
		a.respondError(w, apiError{
			code: http.StatusBadRequest,
			err:  err,
		})
		return
	}
	password := req.Form.Get("password")
	if password == "" {
		a.respondError(w, apiError{
			code: http.StatusBadRequest,
			err:  errors.New("Incorrect change password form"),
		})
		return
	}
	// Check an user is existing
	path := common.Path(model.DefaultUsersPrefix, common.Hash(username, crypto.MD5))
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
	// Hash for new password
	hashedpw, err := common.GenerateBcryptHash(password, config.Get().PasswordHashingCost)
	if err != nil {
		a.respondError(w, apiError{
			code: http.StatusInternalServerError,
			err:  errors.Wrap(err, "Something went wrong"),
		})
		return
	}

	user := &model.User{
		Username: username,
		Password: hashedpw,
	}
	_ = user.Validate()
	r, _ := json.Marshal(&user)
	_, err = a.etcdcli.DoPut(path, string(r))
	if err != nil {
		a.respondError(w, apiError{
			code: http.StatusInternalServerError,
			err:  errors.Wrap(err, "Unable to put a key-value pair into etcd"),
		})
		return
	}
	a.respondSuccess(w, http.StatusOK, nil)
	return
}
