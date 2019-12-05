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

type Receivers []string

type ActionMail struct {
	Action
	Receivers Receivers `json:"receivers"`
	Subject   string    `json:"subject"`
	Body      string    `json:"body"`
}

func (a *ActionMail) Validate() error {
	if err := a.validate(); err != nil {
		return err
	}
	if a.Delay == "" {
		a.Delay = "100ms"
	}
	if a.DelayType == "" {
		a.DelayType = "fixed"
	}
	if a.Attempts == 0 {
		a.Attempts = 10
	}
	return nil
}
