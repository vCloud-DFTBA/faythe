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

package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/prometheus/common/model"

	"github.com/vCloud-DFTBA/faythe/pkg/common"
	fmodel "github.com/vCloud-DFTBA/faythe/pkg/model"
)

// Backend is used to interface with a metrics backend
type Backend interface {
	// GetType returns backend type, for example Prometheus.
	GetType() string
	// GetAddress returns backend address.
	GetAddress() string
	// QueryInstant performs instant query and returns results in model.Vector type.
	QueryInstant(ctx context.Context, query string, ts time.Time) (model.Vector, error)
}

// GetBackend retrieves backend instance from a given provider id.
func GetBackend(e *common.Etcd, pid string) (Backend, error) {
	resp, err := e.DoGet(common.Path(fmodel.DefaultCloudPrefix, pid))
	if err != nil {
		return nil, err
	}
	value := resp.Kvs[0]
	var (
		cloud   fmodel.Cloud
		backend Backend
	)
	err = json.Unmarshal(value.Value, &cloud)
	if err != nil {
		return nil, err
	}
	switch cloud.Provider {
	case fmodel.OpenStackType:
		var ops fmodel.OpenStack
		err = json.Unmarshal(value.Value, &ops)
		if err != nil {
			return nil, err
		}
		// Force register
		ops.Monitor.Password.Decrypt()
		defer ops.Monitor.Password.Encrypt()
		err := Register(ops.Monitor.Backend, string(ops.Monitor.Address),
			ops.Monitor.Username, ops.Monitor.Password.Token)
		if err != nil {
			return nil, err
		}
		backend, _ = Get(fmt.Sprintf("%s-%s", ops.Monitor.Backend, ops.Monitor.Address))
	case fmodel.ManoType:
		var osm fmodel.OpenSourceMano
		err = json.Unmarshal(value.Value, &osm)
		if err != nil {
			return nil, err
		}
		// Force register
		osm.Monitor.Password.Decrypt()
		defer osm.Monitor.Password.Encrypt()
		err := Register(osm.Monitor.Backend, string(osm.Monitor.Address),
			osm.Monitor.Username, osm.Monitor.Password.Token)
		if err != nil {
			return nil, err
		}
		backend, _ = Get(fmt.Sprintf("%s-%s", osm.Monitor.Backend, osm.Monitor.Address))
	}
	return backend, nil
}
