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

package jwt

import "errors"

var (
	ErrInvalidSigningMethod    = errors.New("JWT: invalid signing method")
	ErrNoHMACKey               = errors.New("JWT: no a HMAC key")
	ErrNoRSAKey                = errors.New("JWT: no a RSA key")
	ErrNoECKey                 = errors.New("JWT: no a EC key")
	ErrInvalidToken            = errors.New("JWT: invalid token")
	ErrGetTokenId              = errors.New("JWT: can not get id from token")
	ErrGetIssuedTime           = errors.New("JWT: can not get issued time from token")
	ErrGetData                 = errors.New("JWT: can not get data from token")
	ErrNoStore                 = errors.New("JWT: no store provided")
	ErrUnexpectedSigningMethod = errors.New("JWT: unexpected signing method")
	ErrTokenMalformed          = errors.New("JWT: token is malformed")
	ErrTokenNotActive          = errors.New("JWT: token is not valid yet")
	ErrTokenExpired            = errors.New("JWT: token is expired")
)
