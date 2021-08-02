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
	"github.com/pkg/errors"
)

// Cloud represents Cloud information. Other cloud provider models
// have to inherited this struct
type Cloud struct {
	// The cloud provider type. OpenStack is the only provider supported by now
	Provider  string         `json:"provider"`
	ID        string         `json:"id,omitempty"`
	Endpoints map[string]URL `json:"endpoints"`
	Monitor   Monitor        `json:"monitor"`
	Tags      []string       `json:"tags"`
	CreatedBy string         `json:"created_by"`
}

// Validate cloud information
func (cl *Cloud) Validate() error {
	switch cl.Provider {
	case OpenStackType:
	case ManoType:
	default:
		return errors.Errorf("unsupported provider %s", cl.Provider)
	}
	// Validate endpoints
	for _, e := range cl.Endpoints {
		if err := e.Validate(); err != nil {
			return err
		}
	}

	if err := cl.Monitor.Address.Validate(); err != nil {
		return err
	}
	return nil
}
