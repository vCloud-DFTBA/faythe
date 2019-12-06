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
	"github.com/pkg/errors"
)

type NResolver struct {
	Monitor  Monitor `json:"address"`
	ID       string  `json:"ID"`
	Interval string  `json:"interval"`
	CloudID  string  `json:"cloudid"`
}

func (nr *NResolver) Validate() error {
	if &nr.Monitor == nil {
		return errors.New("missing `Monitor` option")
	}
	if err := nr.Monitor.Address.Validate(); err != nil {
		return err
	}

	if nr.Interval == "" {
		nr.Interval = DefaultNResolverInterval
	}
	return nil
}
