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
	"fmt"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"

	"github.com/vCloud-DFTBA/faythe/config"
)

const TokenExpirationTime = 60

func (a *API) getToken(rw http.ResponseWriter, req *http.Request) {
	user, pass, _ := req.BasicAuth()
	creds := config.Get().GlobalConfig.BasicAuthentication

	if creds.Password != pass || creds.Username != user {
		a.respondError(rw, apiError{
			code: http.StatusUnauthorized,
			err:  fmt.Errorf("invalid user or password"),
		})
		return
	}

	expTime := time.Now().Add(TokenExpirationTime * time.Minute)

	claims := jwt.StandardClaims{ExpiresAt: expTime.Unix()}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(creds.SecretKey))

	if err != nil {
		a.respondError(rw, apiError{
			code: http.StatusInternalServerError,
			err:  err,
		})
		return
	}

	http.SetCookie(rw, &http.Cookie{
		Name:     "api-token",
		Value:    tokenString,
		Path:     "/",
		Expires:  expTime,
		HttpOnly: true,
	})

	a.respondSuccess(rw, http.StatusOK, "")
}
