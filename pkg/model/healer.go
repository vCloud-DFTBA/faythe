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

import "time"

const (
	DefaultHealerPrefix string = "/healers"
	DefaultHealerQuery  string = "up{job=~\".*compute-cadvisor.*|.*compute-node.*\"} < 1"
)

// Healer represents a Healer instance
type Healer struct {
	ID          string                     `json:"id,omitempty"`
	Actions     map[string]ActionInterface `json:"actions"`
	Cooldown    string                     `json:"cooldown"`
	Interval    string                     `json:"interval"`
	Duration    string                     `json:"duration"`
	ATEngine    ATEngine                   `json:"atengine"`
	Monitor     Monitor                    `json:"monitor"`
	Description string                     `json:"description,omitempty"`
	Tags        []string                   `json:"tag,omitempty"`
	Active      bool                       `json:"active,omitempty"`
	Alert       Alert                      `json:"alert,omitempty"`
}

func (h *Healer) Validate() error {
	for _, a := range h.Actions {
		if err := a.Validate(); err != nil {
			return err
		}
	}
	if h.Interval == "" {
		h.Interval = "30s"
	}
	if h.Cooldown == "" {
		h.Cooldown = "600s"
	}
	if h.Duration == "" {
		h.Duration = "300s"
	}

	if _, err := time.ParseDuration(h.Duration); err != nil {
		return err
	}

	if _, err := time.ParseDuration(h.Cooldown); err != nil {
		return err
	}
	if _, err := time.ParseDuration(h.Interval); err != nil {
		return err
	}

	return nil
}
