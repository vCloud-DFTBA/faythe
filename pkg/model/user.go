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

package model

import (
	"crypto"

	"github.com/vCloud-DFTBA/faythe/pkg/common"
)

// User represents an Faythe user
type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
	ID       string `json:"id,omitempty"`
}

// Validate returns nil if all fields of the User have valid values.
// Nothing to check, just generate to UUID.
func (u *User) Validate() error {
	u.ID = common.Hash(u.Username, crypto.MD5)
	return nil
}
