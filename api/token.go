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
	"fmt"
	"net/http"

	"github.com/pkg/errors"

	"github.com/vCloud-DFTBA/faythe/pkg/common"
	"github.com/vCloud-DFTBA/faythe/pkg/model"
)

const (
	bearerFormat string = "Bearer %s"
)

// issueToken by username password.
func (a *API) issueToken(w http.ResponseWriter, req *http.Request) {
	username, password, _ := req.BasicAuth()
	// Check user in Etcd
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
			code: http.StatusUnauthorized,
			err:  errors.New("Invalid credentials"),
		})
		return
	}
	// Get only the first item
	user := model.User{}
	_ = json.Unmarshal(resp.Kvs[0].Value, &user)
	// Validate password
	if ok := common.CheckPasswordAgainstHash(password, user.Password); ok {
		data := make(map[string]interface{})
		data["name"] = username

		tokenString, err := a.jwtToken.GenerateToken(data)
		if err != nil {
			a.respondError(w, apiError{
				code: http.StatusInternalServerError,
				err:  errors.Wrap(err, "Something went wrong"),
			})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Add("Authorization", fmt.Sprintf(bearerFormat, tokenString))
		a.respondSuccess(w, http.StatusOK, nil)
		return
	}
	a.respondError(w, apiError{
		code: http.StatusUnauthorized,
		err:  errors.New("Invalid credentials"),
	})
}
