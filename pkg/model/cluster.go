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
	"crypto"
	"net"

	"github.com/pkg/errors"

	"github.com/vCloud-DFTBA/faythe/pkg/common"
)

const (
	// DefaultClusterPrefix is the etcd default prefix for cluster
	DefaultClusterPrefix string = "/cluster"
)

// Member is the logical node
type Member struct {
	Name    string `json:"name"`
	ID      string `json:"id"`
	Address string `json:"address"`
}

// Validate returns nil if all fields of the member have valid values.
func (m *Member) Validate() error {
	if ip := net.ParseIP(m.Address); ip == nil {
		return errors.Errorf("member's address: %s is not a valid textual representation of an IP address", m.Address)
	}
	m.ID = common.Hash(m.Name+m.Address, crypto.MD5)
	return nil
}
