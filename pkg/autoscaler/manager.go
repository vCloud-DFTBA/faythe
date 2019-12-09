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
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	etcdv3 "go.etcd.io/etcd/clientv3"

	"github.com/ntk148v/faythe/pkg/metrics"
	"github.com/ntk148v/faythe/pkg/model"
	"github.com/ntk148v/faythe/pkg/utils"
)

// Manager manages a set of Scaler instances.
type Manager struct {
	logger  log.Logger
	rgt     *Registry
	stop    chan struct{}
	etcdcli *etcdv3.Client
	watch   etcdv3.WatchChan
	ctx     context.Context
	cancel  context.CancelFunc
	wg      *sync.WaitGroup
}

// NewManager returns an Autoscale Manager
func NewManager(l log.Logger, e *etcdv3.Client) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	m := &Manager{
		logger:  l,
		rgt:     &Registry{items: make(map[string]*Scaler)},
		stop:    make(chan struct{}),
		etcdcli: e,
		ctx:     ctx,
		wg:      &sync.WaitGroup{},
		cancel:  cancel,
	}
	m.watch = m.etcdcli.Watch(m.ctx, model.DefaultScalerPrefix, etcdv3.WithPrefix())
	// Load at init
	m.load()
	return m
}

// Stop the manager and its scaler cycles.
func (m *Manager) Stop() {
	level.Info(m.logger).Log("msg", "Stopping autoscale manager...")
	m.save()
	// Wait until all scalers shut down
	m.wg.Wait()
	m.cancel()
	close(m.stop)
	level.Info(m.logger).Log("msg", "Autoscale manager stopped")
}

// Run starts processing of the autoscale manager
func (m *Manager) Run() {
	for {
		select {
		case <-m.stop:
			return
		case watchResp := <-m.watch:
			for _, event := range watchResp.Events {
				sid := string(event.Kv.Key)
				// Create -> simply create and add it to registry
				if event.IsCreate() {
					m.startScaler(sid, event.Kv.Value)
				}
				// Update -> force recreate scaler
				if event.IsModify() {
					m.stopScaler(sid)
					m.startScaler(sid, event.Kv.Value)
				}
				// Delete -> remove from registry and stop the goroutine
				if event.Type == etcdv3.EventTypeDelete {
					m.stopScaler(sid)
				}
			}
		default:
		}
	}
}

func (m *Manager) stopScaler(id string) {
	if s, ok := m.rgt.Get(id); ok {
		level.Info(m.logger).Log("msg", "Removing scaler", "id", id)
		s.stop()
		m.rgt.Delete(id)
	}
}

func (m *Manager) startScaler(id string, data []byte) {
	level.Info(m.logger).Log("msg", "Creating scaler", "id", id)
	backend, err := m.getBackend(id)
	if err != nil {
		level.Error(m.logger).Log("msg", "Error creating registry backend for scaler",
			"id", id)
		return
	}
	s := newScaler(log.With(m.logger, "scaler", id), data, backend)
	m.rgt.Set(id, s)
	go func() {
		m.wg.Add(1)
		s.run(m.ctx, m.wg)
	}()
}

func (m *Manager) getBackend(key string) (metrics.Backend, error) {
	// There is format -> Cloud provider id
	providerID := strings.Split(key, "/")[2]
	resp, err := m.etcdcli.Get(m.ctx, utils.Path(model.DefaultCloudPrefix, providerID))
	if err != nil {
		return nil, err
	}
	value := resp.Kvs[0]
	var (
		cloud   model.Cloud
		backend metrics.Backend
	)
	err = json.Unmarshal(value.Value, &cloud)
	if err != nil {
		return nil, err
	}
	switch cloud.Provider {
	case "openstack":
		var ops model.OpenStack
		err = json.Unmarshal(value.Value, &ops)
		if err != nil {
			return nil, err
		}
		// Force register
		err := metrics.Register(ops.Monitor.Backend, string(ops.Monitor.Address))
		if err != nil {
			return nil, err
		}
		backend, _ = metrics.Get(fmt.Sprintf("%s-%s", ops.Monitor.Backend, ops.Monitor.Address))
	default:
	}
	return backend, nil
}

// save puts scalers to etcd
func (m *Manager) save() {
	for i := range m.rgt.Iter() {
		m.wg.Add(1)
		go func(wg *sync.WaitGroup) {
			defer func() {
				m.stopScaler(i.Key)
				wg.Done()
			}()
			i.Value.Alert = i.Value.alert.state
			raw, err := json.Marshal(&i.Value)
			if err != nil {
				level.Error(m.logger).Log("msg", "Error serializing scaler object",
					"id", i.Value.ID, "err", err)
				return
			}
			_, err = m.etcdcli.Put(m.ctx, i.Key, string(raw))
			if err != nil {
				level.Error(m.logger).Log("msg", "Error putting scaler object",
					"key", i.Key, "err", err)
				return
			}
		}(m.wg)
	}
}

func (m *Manager) load() {
	resp, err := m.etcdcli.Get(m.ctx, model.DefaultScalerPrefix,
		etcdv3.WithPrefix(), etcdv3.WithSort(etcdv3.SortByKey, etcdv3.SortAscend))
	if err != nil {
		level.Error(m.logger).Log("msg", "Error getting scalers", "err", err)
		return
	}
	for _, ev := range resp.Kvs {
		m.startScaler(string(ev.Key), ev.Value)
	}
}
