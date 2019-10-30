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

package model

import (
	"fmt"
	"strings"
)

type ATEngine struct {
	Backend  string            `json:"backend"`
	Address  URL               `json:"address"`
	Metadata map[string]string `json:"metadata"`
	Username string            `json:"username,omitempty"`
	Password string            `json:"password,omitempty"`
	APIKey   string            `json:"apikey,omitempty"`
}

func (at ATEngine) Validate() error {
	if err := at.Address.Validate(); err != nil {
		return err
	}

	switch strings.ToLower(at.Backend) {
	case "stackstorm":
	default:
		return fmt.Errorf("unsupported backend type: %s", at.Backend)
	}

	return nil
}
