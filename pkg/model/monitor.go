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

package model

import "github.com/vCloud-DFTBA/faythe/pkg/common"

// Monitor represents a monitor backend
type Monitor struct {
	Backend  string              `json:"backend"`
	Address  URL                 `json:"address"`
	Metadata map[string]string   `json:"metadata"`
	Username string              `json:"username,omitempty"`
	Password common.FernetString `json:"password,omitempty"`
}
