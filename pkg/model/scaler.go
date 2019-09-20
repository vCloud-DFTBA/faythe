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

	"github.com/ntk148v/faythe/pkg/metrics"
	"github.com/ntk148v/faythe/pkg/utils"
)

// Scaler represents a Scaler object
type Scaler struct {
	Backend     string            `json:"backend"`
	Query       string            `json:"query"`
	Duration    string            `json:"duration"`
	Description string            `json:"description,omitempty"`
	Interval    string            `json:"interval"`
	Actions     map[string]URL    `json:"actions"`
	Metadata    map[string]string `json:"metadata"`
	ID          uint64            `json:"-,omitempty"`
}

// Validate returns nil if all fields of the Scaler have valid values.
func (s *Scaler) Validate() error {
	as := make([]string, len(s.Actions))
	for _, a := range s.Actions {
		if err := a.Validate(); err != nil {
			return errors.Errorf("invalid action url: %s", a.String())
		}
		as = append(as, a.String())
	}

	switch s.Backend {
	case metrics.Prometheus:
		// Ignore this case, it's good
	default:
		return errors.Errorf("invalid metric backend: %s", s.Backend)
	}

	if s.Query == "" {
		return errors.Errorf("required field %+v is missing or invalid", s.Query)
	}

	if _, err := time.ParseDuration(s.Duration); err != nil {
		return errors.Errorf("required field %+v is missing or invalid: %s", s.Duration, err.Error())
	}

	if _, err := time.ParseDuration(s.Interval); err != nil {
		return errors.Errorf("required field %+v is missing or invalid: %s", s.Interval, err.Error())
	}

	s.ID = utils.Hash(strings.Join(as, ""))

	return nil
}
