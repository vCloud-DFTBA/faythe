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
	Type      string `json:"type"`
	Attempts  uint   `json:"attempts,omitempty"`
	Delay     string `json:"delay,omitempty"`
	DelayType string `json:"delay_type,omitempty"`
}

type ActionInterface interface {
	Validate() error
}

func (a Action) validate() error {
	if a.Type == "" {
		return errors.Errorf("Missing action type")
	}

	if _, err := time.ParseDuration(a.Delay); err != nil {
		return err
	}

	switch strings.ToLower(a.Type) {
	case "http", "mail":
	default:
		return errors.Errorf("unsupported action type: %s", a.Type)
	}

	switch strings.ToLower(a.DelayType) {
	case BackoffDelay:
		// BackOffDelay is a DelayType which increases delay between consecutive retries
	case FixedDelay:
		// FixedDelay is a DelayType which keeps delay the same through all iterations
	default:
		return errors.Errorf("unsupported delay type: %s", a.DelayType)
	}

	return nil
}
