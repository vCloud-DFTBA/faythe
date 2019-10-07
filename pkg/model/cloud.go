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
	"fmt"
	"github.com/pkg/errors"

	"github.com/ntk148v/faythe/pkg/utils"
)

var (
	// DefaultCloudPrefix is the default etcd prefix for Cloud data
	DefaultCloudPrefix = "/clouds"
)

const (
	// OpenStackType represents a OpenStack type
	OpenStackType string = "openstack"
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
}

func (cl *Cloud) Validate() error {
	switch cl.Provider {
	case "openstack":
	default:
		return errors.Errorf("unsupported provider %s", cl.Provider)
	}
	// Validate endpoints
	for _, e := range cl.Endpoints {
		if err := e.Validate(); err != nil {
			return err
		}
	}
	// Require Monitor backend
	if &cl.Monitor == nil {
		return errors.New("missing `Monitor` option")
	}
	if err := cl.Monitor.Address.Validate(); err != nil {
		return err
	}
	return nil
}

// OpenStack represents OpenStack information.
type OpenStack struct {
	Cloud
	Auth OpenStackAuth `json:"auth"`
}

// OpenStackAuth stores information needed to authenticate to an OpenStack Cloud.
type OpenStackAuth struct {
	// AuthURL specifies the HTTP endpoint that is required to work with
	// the Identity API of the appropriate version. While it's ultimately needed by
	// all of the identity services, it will often be populated by a provider-level
	// function.
	AuthURL    string `json:"auth_url"`
	RegionName string `json:"region_name"`

	// Username is required if using Identity V2 API. Consult with your provider's
	// control panel to discover your account's username. In Identity V3, either
	// UserID or a combination of Username and DomainID or DomainName are needed.
	Username string `json:"username"`
	UserID   string `json:"userid"`

	Password utils.Secret `json:"password"`

	// At most one of DomainID and DomainName must be provided if using Username
	// with Identity V3. Otherwise, either are optional.
	DomainName string `json:"domain_name"`
	DomainID   string `json:"domain_id"`

	// The ProjectID and ProjectName fields are optional for the Identity V2 API.
	// The same fields are known as project_id and project_name in the Identity
	// V3 API, but are collected as ProjectID and ProjectName here in both cases.
	// Some providers allow you to specify a ProjectName instead of the ProjectId.
	// Some require both. Your provider's authentication policies will determine
	// how these fields influence authentication.
	// If DomainID or DomainName are provided, they will also apply to ProjectName.
	// It is not currently possible to authenticate with Username and a Domain
	// and scope to a Project in a different Domain by using ProjectName. To
	// accomplish that, the ProjectID will need to be provided as the ProjectID
	// option.
	ProjectName string `json:"project_name"`
	ProjectID   string `json:"project_id"`
}

// Validate returns nil if all fields of the OpenStack have valid values.
func (op *OpenStack) Validate() error {
	switch op.Provider {
	case "openstack":
	default:
		return errors.Errorf("unsupported cloud provider: %s", op.Provider)
	}
	// Require at least auth_url
	if op.Auth.AuthURL == "" {
		return errors.New("missing `IdentityEndpoint` in OpenStack AuthOpts")
	}

	op.ID = fmt.Sprintf("%x", utils.HashSHA(op.Auth.AuthURL))

	return nil
}
