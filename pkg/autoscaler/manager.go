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
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/ntk148v/faythe/pkg/model"
	etcdv3 "go.etcd.io/etcd/clientv3"
	"sync"
)

// Manager manages a set of Scaler instances.
type Manager struct {
	logger  log.Logger
	rgt     *Registry
	stop    chan struct{}
	etcdcli *etcdv3.Client
	watch   etcdv3.WatchChan
	mtx     sync.RWMutex
	ctx     context.Context
	wg      *sync.WaitGroup
}

// NewManager returns an Autoscale Manager
func NewManager(l log.Logger, e *etcdv3.Client) *Manager {
	m := &Manager{
		logger:  l,
		rgt:     &Registry{items: make(map[string]*Scaler)},
		stop:    make(chan struct{}),
		etcdcli: e,
		ctx:     context.Background(),
		wg:      &sync.WaitGroup{},
	}
	m.watch = m.etcdcli.Watch(m.ctx, model.DefaultScalerPrefix, etcdv3.WithPrefix())
	// Load at init
	m.Load()
	return m
}

// Stop the manager and its scaler cycles.
func (m *Manager) Stop() {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	level.Info(m.logger).Log("msg", "Stopping autoscale manager...")
	// Wait until all scalers shut down
	m.wg.Wait()
	close(m.stop)
	for s := range m.rgt.Iter() {
		s.Value.stop()
	}

	level.Info(m.logger).Log("msg", "Autoscale manager stopped")
}

// Run starts processing of the autoscale manager
func (m *Manager) Run() {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	defer func() {
		m.Stop()
		// Save all alert state to etcd
		m.Save()
	}()

	for {
		select {
		case <-m.stop:
			return
		default:
			for watchResp := range m.watch {
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
			}
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
	s := newScaler(log.With(m.logger, "scaler", id), data)
	m.rgt.Set(id, s)
	go func() {
		m.wg.Add(1)
		s.run(m.ctx, m.wg)
	}()
}

// Save puts scalers to etcd
func (m *Manager) Save() {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	for i := range m.rgt.Iter() {
		raw, err := json.Marshal(&i.Value)
		if err != nil {
			level.Error(m.logger).Log("msg", "Error serializing scaler object",
				"id", i.Value.ID, "err", err)
			continue
		}
		_, err = m.etcdcli.Put(m.ctx, i.Key, string(raw))
		if err != nil {
			level.Error(m.logger).Log("msg", "Error putting scaler object",
				"key", i.Key, "err", err)
		}
	}
}

func (m *Manager) Load() {
	m.mtx.Lock()
	defer m.mtx.Unlock()

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
