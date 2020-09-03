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
	"strings"

	"github.com/pkg/errors"

	"github.com/vCloud-DFTBA/faythe/pkg/common"
)

// ActionHTTP represents a HTTP request with retry logic.
type ActionHTTP struct {
	Action
	URL            URL               `json:"url"`
	CloudAuthToken bool              `json:"cloud_auth_token"`
	Method         string            `json:"method,omitempty"`
	Attempts       uint              `json:"attempts,omitempty"`
	Delay          string            `json:"delay,omitempty"`
	DelayType      string            `json:"delay_type,omitempty"`
	Header         map[string]string `json:"header,omitempty"`
	Body           interface{}       `json:"body,omitempty"`
}

// Validate returns nil if all fields of the Action have valid values.
func (a *ActionHTTP) Validate() error {
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
	if _, err := common.ParseDuration(a.Delay); err != nil {
		return err
	}
	switch strings.ToLower(a.DelayType) {
	case BackoffDelay:
		// BackOffDelay is a DelayType which increases delay between consecutive retries
	case FixedDelay:
		// FixedDelay is a DelayType which keeps delay the same through all iterations
	default:
		return errors.Errorf("unsupported delay type: %s", a.DelayType)
	}
	if err := a.validate(); err != nil {
		return err
	}
	if err := a.URL.Validate(); err != nil {
		return err
	}
	return nil
}

// InfoLog returns type, url & method information.
func (a *ActionHTTP) InfoLog() []string {
	return []string{
		"type", a.Type,
		"url", a.URL.String(),
		"method", a.Method,
	}
}
