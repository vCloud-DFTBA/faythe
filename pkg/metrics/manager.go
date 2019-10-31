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
	"fmt"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"

	"github.com/vCloud-DFTBA/faythe/pkg/metrics/backends/prometheus"
)

// Manager maintains a set of Backends.
type Manager struct {
	logger log.Logger
	rgt    *Registry
}

// NewManager is the MetricsManager constructor.
func NewManager() *Manager {
	mgr := &Manager{
		logger: log.NewNopLogger(),
		rgt:    &Registry{items: make(map[string]Backend)},
	}
	return mgr
}

var m *Manager

func init() {
	m = NewManager()
}

func (m *Manager) initBackend(btype string, address string) (Backend, error) {
	switch btype {
	case Prometheus:
		return prometheus.New(address, log.With(m.logger, "component", "metric backend",
			"name", fmt.Sprintf("%s-%s", btype, address)))
	default:
		return nil, errors.Errorf("unknown backend type %q", btype)
	}
}

// Register inits Backend with input Type and address, puts the instantiated
// backend to Registry.
func (m *Manager) Register(btype, address string) error {
	name := fmt.Sprintf("%s-%s", btype, address)
	// If the instantiated metrics backend already exists, let's just
	// ignore it.
	if _, ok := m.rgt.Get(name); ok {
		return nil
	}

	level.Info(m.logger).Log("msg", "Instantiating backend client for MetricsBackend", btype)
	b, err := m.initBackend(btype, address)
	if err != nil {
		return errors.Wrapf(err, "instantiating backend client for MetricsBackend %q", btype)
	}
	m.rgt.Set(name, b)
	level.Info(m.logger).Log("msg", "Backend", name, "instantiated successfully")

	return nil
}

// Register inits Backend with input Type and address, puts the instantiated
// backend to Registry.
func Register(btype, address string) error {
	return m.Register(btype, address)
}

// Unregister removes Backend from registry
func Unregister(name string) {
	m.rgt.Delete(name)
}

// Get returns a Backend with a given name
func Get(name string) (Backend, bool) {
	return m.rgt.Get(name)
}
