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

package autoscaler

import (
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	etcdv3 "go.etcd.io/etcd/clientv3"
	"sync"
)

// Manager manages a set of Scaler instances.
type Manager struct {
	logger   log.Logger
	rgt      *Registry
	stopChan chan struct{}
	etcdcli  *etcdv3.Client
}

// NewManager returns an Autoscale Manager
func NewManager(l log.Logger, stopChan chan struct{}, e *etcdv3.Client) *Manager {
	return &Manager{
		logger:   l,
		rgt:      &Registry{items: make(map[string]Scaler)},
		stopChan: stopChan,
		etcdcli:  e,
	}
}

func (m *Manager) run() {
	// This stop channel tells the rgt to stop if the manager is
	// shutting down for any reason. It must be closed if this function exits.
	scalerStopCh := make(chan struct{})

	var wg sync.WaitGroup

	for {
		select {
		case <-scalerStopCh:
			wg.Wait()
		}
	}
	for i := range m.rgt.Iter() {
		wg.Add(1)
		go i.Value.run(&wg, scalerStopCh)
	}

	defer func() {
		close(scalerStopCh)

		// Wait for all scalers to shut down
		wg.Wait()
		level.Info(m.logger).Log("msg", "Autoscale Manager shuts down success")
	}()
}
