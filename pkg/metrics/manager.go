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
	"fmt"
	"net/url"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"

	"github.com/ntk148v/faythe/pkg/metrics/backends/prometheus"
)

// Manager maintains a set of Backends.
type Manager struct {
	logger log.Logger
	rgt    RegistryInterface
}

// NewManager is the MetricsManager constructor.
func NewManager(logger log.Logger, options ...func(*Manager)) *Manager {
	if logger == nil {
		logger = log.NewNopLogger()
	}

	mgr := &Manager{
		logger: logger,
		rgt:    Registry(),
	}
	return mgr
}

func (m *Manager) initBackend(btype string, address url.URL) (Backend, error) {
	switch btype {
	case "promtheus":
		return prometheus.NewClient(address, log.With(m.logger, fmt.Sprintf("%s-%s", btype, address.String())))
	default:
		return nil, errors.Errorf("unknown backend type %q", btype)
	}
}

// Register inits Backend with input Type and address, puts the instantiated
// backend to registry.
func (m *Manager) Register(btype string, address url.URL) error {
	name := fmt.Sprintf("%s-%s", btype, address.String())
	// If the instantiated metrics backend already exists, let's just
	// ignore it.
	if _, err := m.rgt.Get(name); err == nil {
		return nil
	}

	level.Info(m.logger).Log("msg", "Instantiating backend client for MetricsBackend", btype)
	b, err := m.initBackend(btype, address)
	if err != nil {
		return errors.Wrap(err, "instantiating backend client for MetricsBackend %q", btype)
	}
	m.rgt.Put(name, b)
	level.Info(m.logger).Log("msg", "Backend", name, "instantiated successfully")

	return nil
}
