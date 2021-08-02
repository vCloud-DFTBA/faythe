// Copyright (c) 2021 Manh Vu Duc <manhvd.hust@gmail.com>

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
	"bytes"
	"crypto"
	"crypto/tls"
	"encoding/json"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"
	"github.com/vCloud-DFTBA/faythe/pkg/common"
)

// OpenSourceMano represents OpenSourceMano information
type OpenSourceMano struct {
	Cloud
	Auth OpenSourceManoAuth `json:"auth"`
}

// OpenSourceManoAuth stores information needed to authenticate an OpenSourceMano
type OpenSourceManoAuth struct {
	AuthURL     string `json:"auth_url"`
	Username    string `json:"username"`
	Password    string `json:"password"`
}

type Roles struct {
	Id   string `yaml:"id"`
	Name string `yaml:"name"`
}

type OSMToken struct {
	RemotePort  int     `yaml:"remote_port"`
	Username    string  `yaml:"username"`
	Id1          string  `yaml:"_id"`
	Admin       bool    `yaml:"admin"`
	IssuedAt    float64 `yaml:"issued_at"`
	RemoteHost  string  `yaml:"remote_host"`
	Roles       []Roles `yaml:"roles"`
	UserId      string  `yaml:"user_id"`
	Expires     float64 `yaml:"expires"`
	Id2         string  `yaml:"id"`
	ProjectId   string  `yaml:"project_id"`
	ProjectName string  `yaml:"project_name"`
}

// Validate returns nil if all fields of the OpenSourceMano have valid values.
func (osm *OpenSourceMano) Validate() error {
	switch osm.Provider {
	case ManoType:
	default:
		return errors.Errorf("Unsupported Mano type: %s", osm.Provider)
	}
	// Require at least token_url
	if osm.Auth.AuthURL == "" {
		return errors.New("missing `IdentityEndpoint` in OpenSourceMano AuthOpts")
	}

	osm.ID = common.Hash(osm.Auth.AuthURL, crypto.MD5)

	return nil
}

// GetToken of OpenSourceMano
func (osm *OpenSourceMano) GetToken() (string, error) {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	basicAuth := map[string]string{"username": osm.Auth.Username, "password": osm.Auth.Password}
	reqBody, err := json.Marshal(basicAuth)
	if err != nil {
		return "", err
	}

	resp, err := http.Post(osm.Auth.AuthURL , "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", err
	}
	// Close the response body
	if resp != nil {
		defer resp.Body.Close()
	}
	// Success is indicated with 2xx status codes
	statusOK := resp.StatusCode >= 200 && resp.StatusCode < 300
	if !statusOK {
		return "",errors.Errorf("non-OK HTTP status: %s", resp.Status)
	}
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	bodyString := string(respBody)
	respYaml := []byte(bodyString)
	osmToken := OSMToken{}
	err = yaml.Unmarshal(respYaml, &osmToken)
	if err != nil {
		return "", err
	}
	return  osmToken.Id1, nil
}