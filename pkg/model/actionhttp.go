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

type ActionHTTP struct {
	Action
	URL    URL    `json:"url"`
	Method string `json:"method,omitempty"`
}

// Validate returns nil if all fields of the Action have valid values.
func (a *ActionHTTP) Validate() error {

	if err := a.Validate(); err != nil {
		return err
	}

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

	return nil
}
