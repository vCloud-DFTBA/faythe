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
	"time"

	"github.com/ntk148v/faythe/pkg/utils"
	"github.com/pkg/errors"
)

const (
	DefaultNResolverPrefix = "/nresolvers"
	DefaultNResolverQuery  = "node_uname_info"
)

type NResolver struct {
	Address  URL    `json:"address"`
	Name     string `json:"name"`
	Interval string `json:"interval"`
}

func (nr *NResolver) Validate() error {
	if nr.Address == "" {
		return errors.New("Missing `Address` option")
	}

	if err := nr.Address.Validate(); err != nil {
		return err
	}

	if nr.Interval == "" {
		nr.Interval = "600s"
	}

	if _, err := time.ParseDuration(nr.Interval); err != nil {
		return err
	}

	if nr.Name == "" {
		nr.Name = utils.HashFNV(nr.Address.String())
	}
	return nil
}
