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
	"encoding/json"

	"github.com/gophercloud/gophercloud"
	"github.com/ntk148v/faythe/pkg/utils"
	"github.com/pkg/errors"
)

// OpenStack represents OpenStack information.
type OpenStack struct {
	Endpoints map[string]URL          `json:"endpoints"`
	Signature uint64                  `json:"signature,omitempty"`
	Auth      gophercloud.AuthOptions `json:"auth"`
}

// MarshalJSON implements the json.Marshaler interface
func (op *OpenStack) MarshalJSON() ([]byte, error) {
	return json.Marshal(op)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (op *OpenStack) UnmarshalJSON(b []byte) error {
	var o = OpenStack{}
	if err := json.Unmarshal(b, &o); err != nil {
		return err
	}
	for _, e := range o.Endpoints {
		if !e.IsValid() {
			return errors.Errorf("invalid endpoint %s", e.String())
		}
	}

	// Require Prometheus URL as prometheus endpoint
	if _, ok := o.Endpoints["prometheus"]; !ok {
		return errors.New("missing `prometheus` endpoint")
	}

	// Require at least auth_url
	if o.Auth.IdentityEndpoint == "" {
		return errors.New("missing `IdentityEndpoint` in OpenStack AuthOpts")
	}

	o.Signature = utils.Hash(o.Auth.IdentityEndpoint)

	return nil
}
