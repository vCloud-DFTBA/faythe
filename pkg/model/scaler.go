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

import (
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/ntk148v/faythe/pkg/utils"
)

const (
	// DefaultScalerPrefix is the etcd default prefix for scaler
	DefaultScalerPrefix string = "/scalers"
)

// Scaler represents a Scaler object
type Scaler struct {
	Monitor     Monitor           `json:"monitor"`
	Query       string            `json:"query"`
	Duration    string            `json:"duration"`
	Description string            `json:"description,omitempty"`
	Interval    string            `json:"interval"`
	Actions     map[string]Action `json:"actions"`
	Metadata    map[string]string `json:"metadata"`
	Active      bool              `json:"active"`
	ID          string            `json:"id,omitempty"`
	Alert       *Alert            `json:"alert,omitempty"`
}

// Validate returns nil if all fields of the Scaler have valid values.
func (s *Scaler) Validate() error {
	for _, a := range s.Actions {
		if err := a.Validate(); err != nil {
			return err
		}
	}

	// Require Monitor backend
	if &s.Monitor == nil {
		return errors.New("missing `Monitor` option")
	}
	if err := s.Monitor.Address.Validate(); err != nil {
		return err
	}

	if s.Query == "" {
		return errors.Errorf("required field %+v is missing or invalid", s.Query)
	}

	if _, err := time.ParseDuration(s.Duration); err != nil {
		return err
	}

	if _, err := time.ParseDuration(s.Interval); err != nil {
		return err
	}

	s.ID = fmt.Sprintf("%x", utils.HashSHA(s.Query))

	return nil
}
