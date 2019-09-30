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
	"strings"
	"time"

	"github.com/pkg/errors"
)

// Action represents an scale action
type Action struct {
	URL       URL    `json:"url"`
	Type      string `json:"type,omitempty"`
	Method    string `json:"method,omitempty"`
	Attempts  uint   `json:"attempts,omitempty"`
	Delay     string `json:"delay,omitempty"`
	DelayType string `json:"delay_type,omitempty"`
}

// Validate returns nil if all fields of the Action have valid values.
func (a *Action) Validate() error {
	if err := a.URL.Validate(); err != nil {
		return err
	}
	if a.Delay == "" {
		a.Delay = "100ms"
	}
	if a.DelayType == "" {
		a.DelayType = "fixed"
	}
	if a.Method == "" {
		a.Method = "POST"
	}
	if a.Attempts == 0 {
		a.Attempts = 10
	}
	if a.Type == "" {
		a.Type = "http"
	}
	switch strings.ToLower(a.Type) {
	case "http":
	default:
		return errors.Errorf("unsupported action type: %s", a.Type)
	}
	if _, err := time.ParseDuration(a.Delay); err != nil {
		return err
	}
	switch strings.ToLower(a.DelayType) {
	case "backoff":
		// BackOffDelay is a DelayType which increases delay between consecutive retries
	case "fixed":
		// FixedDelay is a DelayType which keeps delay the same through all iterations
	default:
		return errors.Errorf("unsupported delay type: %s", a.DelayType)
	}
	return nil
}
