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
	"go.etcd.io/etcd/mvcc/mvccpb"

	"github.com/vCloud-DFTBA/faythe/pkg/cluster"
	"github.com/vCloud-DFTBA/faythe/pkg/metrics"
	"github.com/vCloud-DFTBA/faythe/pkg/model"
	"github.com/vCloud-DFTBA/faythe/pkg/utils"
)

// Manager manages a set of Scaler instances.
type Manager struct {
	logger  log.Logger
	rgt     *Registry
	stop    chan struct{}
	etcdcli *etcdv3.Client
	watch   etcdv3.WatchChan
	wg      *sync.WaitGroup
	cluster *cluster.Cluster
}

// NewManager returns an Autoscale Manager
func NewManager(l log.Logger, e *etcdv3.Client, c *cluster.Cluster) *Manager {
	m := &Manager{
		logger:  l,
		rgt:     &Registry{items: make(map[string]*Scaler)},
		stop:    make(chan struct{}),
		etcdcli: e,
		wg:      &sync.WaitGroup{},
		cluster: c,
	}
	// Load at init
	m.load()
	return m
}

// Reload simply stop and start scalers selectively.
func (m *Manager) Reload() {
	level.Info(m.logger).Log("msg", "Reloading...")
	m.rebalance()
	level.Info(m.logger).Log("msg", "Reloaded")
}

// Stop the manager and its scaler cycles.
func (m *Manager) Stop() {
	level.Info(m.logger).Log("msg", "Stopping autoscale manager...")
	close(m.stop)
	m.save()
	// Wait until all scalers shut down
	m.wg.Wait()
	level.Info(m.logger).Log("msg", "Autoscale manager stopped")
}

// Run starts processing of the autoscale manager
func (m *Manager) Run(ctx context.Context) {
	watch := m.etcdcli.Watch(ctx, model.DefaultScalerPrefix, etcdv3.WithPrefix())
	for {
		select {
		case <-m.stop:
			return
		case watchResp := <-watch:
			if watchResp.Err() != nil {
				level.Error(m.logger).Log("msg", "Error watching cluster state", "err", watchResp.Err())
				break
			}
			for _, event := range watchResp.Events {
				sid := string(event.Kv.Key)
				if event.IsCreate() {
					// Create -> simply create and add it to registry
					m.startScaler(sid, event.Kv.Value)
				} else if event.IsModify() {
					// Update -> force recreate scaler
					if _, ok := m.rgt.Get(sid); ok {
						m.stopScaler(sid)
						m.startScaler(sid, event.Kv.Value)
					}
				} else if event.Type == etcdv3.EventTypeDelete {
					// Delete -> remove from registry and stop the goroutine
					if _, ok := m.rgt.Get(sid); ok {
						m.stopScaler(sid)
					}
				}
			}
		default:
		}
	}
}

func (m *Manager) stopScaler(id string) {
	s, _ := m.rgt.Get(id)
	if local, worker, ok := m.cluster.LocalIsWorker(id); !ok {
		level.Debug(m.logger).Log("msg", "Ignoring scaler, another worker node takes it",
			"scaler", id, "local", local, "worker", worker)
		return
	}
	level.Info(m.logger).Log("msg", "Removing scaler", "id", id)
	s.stop()
	m.rgt.Delete(id)
}

func (m *Manager) startScaler(id string, data []byte) {
	if local, worker, ok := m.cluster.LocalIsWorker(id); !ok {
		level.Debug(m.logger).Log("msg", "Ignoring scaler, another worker node takes it",
			"scaler", id, "local", local, "worker", worker)
		return
	}
	level.Info(m.logger).Log("msg", "Creating scaler", "id", id)
	backend, err := m.getBackend(id)
	if err != nil {
		level.Error(m.logger).Log("msg", "Error creating registry backend for scaler",
			"id", id, "err", err)
		return
	}
	s := newScaler(log.With(m.logger, "scaler", id), data, backend)
	m.rgt.Set(id, s)
	go func() {
		m.wg.Add(1)
		s.run(context.Background(), m.wg)
	}()
}

func (m *Manager) getBackend(key string) (metrics.Backend, error) {
	// There is format -> Cloud provider id
	providerID := strings.Split(key, "/")[2]
	resp, err := m.etcdcli.Get(context.Background(), utils.Path(model.DefaultCloudPrefix, providerID))
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
	case model.OpenStackType:
		var ops model.OpenStack
		err = json.Unmarshal(value.Value, &ops)
		if err != nil {
			return nil, err
		}
		// Force register
		err := metrics.Register(ops.Monitor.Backend, string(ops.Monitor.Address),
			ops.Monitor.Username, ops.Monitor.Password)
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
		go func(i RegistryItem) {
			defer func() {
				m.stopScaler(i.Key)
				m.wg.Done()
			}()
			i.Value.Alert = &i.Value.alert.State
			raw, err := json.Marshal(&i.Value)
			if err != nil {
				level.Error(m.logger).Log("msg", "Error marshalling scaler object",
					"id", i.Value.ID, "err", err)
				return
			}
			_, err = m.etcdcli.Put(context.Background(), i.Key, string(raw))
			if err != nil {
				level.Error(m.logger).Log("msg", "Error putting scaler object",
					"key", i.Key, "err", err)
				return
			}
			m.stopScaler(i.Key)
		}(i)
	}
}

func (m *Manager) load() {
	resp, err := m.etcdcli.Get(context.Background(), model.DefaultScalerPrefix,
		etcdv3.WithPrefix(), etcdv3.WithSort(etcdv3.SortByKey, etcdv3.SortAscend))
	if err != nil {
		level.Error(m.logger).Log("msg", "Error getting scalers", "err", err)
		return
	}
	var sid string
	for _, ev := range resp.Kvs {
		sid = string(ev.Key)
		m.startScaler(sid, ev.Value)
	}
}

func (m *Manager) rebalance() {
	resp, err := m.etcdcli.Get(context.Background(), model.DefaultScalerPrefix,
		etcdv3.WithPrefix(), etcdv3.WithSort(etcdv3.SortByKey, etcdv3.SortAscend))
	if err != nil {
		level.Error(m.logger).Log("msg", "Error getting scalers", "err", err)
		return
	}

	var wg sync.WaitGroup
	for _, ev := range resp.Kvs {
		wg.Add(1)
		go func(ev *mvccpb.KeyValue) {
			defer wg.Done()
			id := string(ev.Key)
			local, worker, ok1 := m.cluster.LocalIsWorker(id)
			scaler, ok2 := m.rgt.Get(id)

			if !ok1 {
				if ok2 {
					scaler.Alert = scaler.alert.state
					raw, err := json.Marshal(&scaler)
					if err != nil {
						level.Error(m.logger).Log("msg", "Error serializing scaler object",
							"id", id, "err", err)
						return
					}
					_, err = m.etcdcli.Put(context.Background(), id, string(raw))
					if err != nil {
						level.Error(m.logger).Log("msg", "Error putting scaler object",
							"key", id, "err", err)
						return
					}
					level.Info(m.logger).Log("msg", "Removing scaler, another worker node takes it",
						"scaler", id, "local", local, "worker", worker)
					scaler.stop()
					m.rgt.Delete(id)
				}
			} else if !ok2 {
				m.startScaler(id, ev.Value)
			}
		}(ev)
	}

	wg.Wait()
}
