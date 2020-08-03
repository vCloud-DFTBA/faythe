// Copyright (c) 2020 Dat Vu Tuan <tuandatk25a@gmail.com>
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

import "github.com/pkg/errors"

// ActionMistral is the special action contains
// OpenStack Mistral workflow information.
type ActionMistral struct {
	Action
	WorkflowID string                 `json:"workflow_id"`
	Input      map[string]interface{} `json:"-"`
}

// Validate returns nil if all fields of the Action have valid values.
func (a *ActionMistral) Validate() error {
	if a.WorkflowID == "" {
		return errors.Errorf("Missing workflow_id")
	}
	if err := a.validate(); err != nil {
		return err
	}
	return nil
}
