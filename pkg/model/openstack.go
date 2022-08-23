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

import (
	"crypto"
	"crypto/tls"
	"net/http"
	"strings"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/pkg/errors"
	"github.com/vCloud-DFTBA/faythe/pkg/common"
)

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

	Password common.FernetString `json:"password"`

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
	_ = op.Auth.Password.Encrypt()
	_ = op.Monitor.Password.Encrypt()
	switch op.Provider {
	case OpenStackType:
	default:
		return errors.Errorf("unsupported cloud provider: %s", op.Provider)
	}
	// Require at least auth_url
	if op.Auth.AuthURL == "" {
		return errors.New("missing `IdentityEndpoint` in OpenStack AuthOpts")
	}

	if _, err := op.BaseClient(); err != nil {
		return err
	}

	op.ID = common.Hash(op.Auth.AuthURL, crypto.MD5)

	return nil
}

func (op *OpenStack) BaseClient() (*gophercloud.ProviderClient, error) {
	op.Auth.Password.Decrypt()
	defer func() { _ = op.Auth.Password.Encrypt() }()
	ao := gophercloud.AuthOptions{
		IdentityEndpoint: op.Auth.AuthURL,
		Username:         op.Auth.Username,
		Password:         op.Auth.Password.Token,
		DomainName:       op.Auth.DomainName,
		DomainID:         op.Auth.DomainID,
		// If OS_PROJECT_NAME is set, overwrite tenantName with the value.
		// https://github.com/gophercloud/gophercloud/blob/master/openstack/auth_env.go#L55
		TenantName: op.Auth.ProjectName,
		// If OS_PROJECT_ID is set, overwrite tenantID with the value.
		// https://github.com/gophercloud/gophercloud/blob/master/openstack/auth_env.go#L50
		TenantID: op.Auth.ProjectID,
	}

	p, err := openstack.NewClient(ao.IdentityEndpoint)

	if err != nil {
		return nil, err
	}

	tlsconfig := &tls.Config{}
	tlsconfig.InsecureSkipVerify = true
	transport := &http.Transport{TLSClientConfig: tlsconfig}

	if strings.Contains(op.Auth.AuthURL, "https") {
		p.HTTPClient = http.Client{
			Transport: transport,
		}
	}

	p.HTTPClient = http.Client{
		Transport: transport,
	}

	err = openstack.Authenticate(p, ao)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (op *OpenStack) NewWorkflowClient() (*gophercloud.ServiceClient, error) {
	p, err := op.BaseClient()
	if err != nil {
		return nil, err
	}
	wc, err := openstack.NewWorkflowV2(p, gophercloud.EndpointOpts{
		Region: op.Auth.RegionName,
	})

	if err != nil {
		return nil, err
	}
	return wc, nil
}
