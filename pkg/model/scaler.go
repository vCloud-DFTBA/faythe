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
	"crypto"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/vCloud-DFTBA/faythe/pkg/common"
)

// Scaler represents a Scaler object
type Scaler struct {
	Query       string                     `json:"query"`
	Duration    string                     `json:"duration"`
	Description string                     `json:"description"`
	Interval    string                     `json:"interval"`
	Actions     map[string]ActionInterface `json:"ractions"`
	ActionsRaw  map[string]json.RawMessage `json:"actions"`
	Tags        []string                   `json:"tags"`
	Active      bool                       `json:"active"`
	ID          string                     `json:"id,omitempty"`
	Alert       *Alert                     `json:"alert,omitempty"`
	Cooldown    string                     `json:"cooldown"`
	CloudID     string                     `json:"cloudid"`
	CreatedBy   string                     `json:"created_by"`
}

// Validate returns nil if all fields of the Scaler have valid values.
func (s *Scaler) Validate() error {
	if s.ActionsRaw != nil {
		s.Actions = make(map[string]ActionInterface, len(s.ActionsRaw))
		for k, v := range s.ActionsRaw {
			a := Action{}
			if err := json.Unmarshal(v, &a); err != nil {
				return err
			}
			// TODO(kiennt): Support other action types like mail & mistral.
			switch strings.ToLower(a.Type) {
			case "http":
				ah := &ActionHTTP{}
				if err := json.Unmarshal(v, ah); err != nil {
					return err
				}
				s.Actions[k] = ah
			default:
				return fmt.Errorf("type %s is not supported", a.Type)
			}
			if err := s.Actions[k].Validate(); err != nil {
				return err
			}
		}
	}

	if s.Query == "" {
		return errors.Errorf("required field %+v is missing or invalid", s.Query)
	}

	if _, err := common.ParseDuration(s.Duration); err != nil {
		return err
	}

	if _, err := common.ParseDuration(s.Interval); err != nil {
		return err
	}

	if s.Cooldown == "" {
		s.Cooldown = "10m"
	}
	if _, err := common.ParseDuration(s.Cooldown); err != nil {
		return err
	}

	s.ID = common.Hash(s.Query, crypto.MD5)

	return nil
}
