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
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Healer represents a Healer instance
type Healer struct {
	ID          string                     `json:"id,omitempty"`
	Actions     map[string]ActionInterface `json:"ractions"`
	ActionsRaw  map[string]json.RawMessage `json:"actions"`
	Query       string                     `json:"query"`
	Interval    string                     `json:"interval"`
	Duration    string                     `json:"duration"`
	ATEngine    ATEngine                   `json:"atengine"`
	Monitor     Monitor                    `json:"monitor"`
	Description string                     `json:"description,omitempty"`
	Tags        []string                   `json:"tags,omitempty"`
	Active      bool                       `json:"active,omitempty"`
	Alert       Alert                      `json:"alert,omitempty"`
	CloudID     string                     `json:"cloudid"`
}

// Validate healher model
func (h *Healer) Validate() error {
	if h.Interval == "" {
		h.Interval = DefaultHealerInterval
	}

	if h.Duration == "" {
		h.Duration = DefaultHealerDuration
	}

	if _, err := time.ParseDuration(h.Duration); err != nil {
		return err
	}

	if _, err := time.ParseDuration(h.Interval); err != nil {
		return err
	}

	if h.Query == "" {
		h.Query = DefaultHealerQuery
	}

	if h.ActionsRaw != nil {
		h.Actions = make(map[string]ActionInterface, len(h.ActionsRaw))
		for k, v := range h.ActionsRaw {
			a := Action{}
			if err := json.Unmarshal(v, &a); err != nil {
				return err
			}
			switch strings.ToLower(a.Type) {
			case "mail":
				am := &ActionMail{}
				if err := json.Unmarshal(v, am); err != nil {
					return err
				}
				h.Actions[k] = am
			case "http":
				ah := &ActionHTTP{}
				if err := json.Unmarshal(v, ah); err != nil {
					return err
				}
				h.Actions[k] = ah
			default:
				return fmt.Errorf("Type %s is not supported", a.Type)
			}
			if err := h.Actions[k].Validate(); err != nil {
				return err
			}
		}
	}

	return nil
}
