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
<<<<<<< HEAD
"github.com/vCloud-DFTBA/faythe/pkg/common"
=======
"github.com/vCloud-DFTBA/faythe/pkg/exporter"
>>>>>>> Implement Exporter module
"github.com/vCloud-DFTBA/faythe/pkg/metrics"
"github.com/vCloud-DFTBA/faythe/pkg/model"
)

// Manager manages a set of Scaler instances.
type Manager struct {
	logger  log.Logger
	rgt     *common.Registry
	stop    chan struct{}
	etcdcli *common.Etcd
	watch   etcdv3.WatchChan
	wg      *sync.WaitGroup
	cluster *cluster.Cluster
	state   model.State
}

// NewManager returns an Autoscale Manager
func NewManager(l log.Logger, e *common.Etcd, c *cluster.Cluster) *Manager {
	m := &Manager{
		logger:  l,
		rgt:     &common.Registry{Items: make(map[string]common.Worker)},
		stop:    make(chan struct{}),
		etcdcli: e,
		wg:      &sync.WaitGroup{},
		cluster: c,
	}
	// Init with 0
	exporter.ReportNumScalers(cluster.ClusterID, 0)
	// Load at init
	m.load()
	m.state = model.StateActive
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
	// Ignore close channel if manager is already stopped/stopping
	if m.state == model.StateStopping || m.state == model.StateStopped {
		return
	}
	level.Info(m.logger).Log("msg", "Stopping autoscale manager...")
	m.state = model.StateStopping
	close(m.stop)
	m.save()
	// Wait until all scalers shut down
	m.wg.Wait()
	m.state = model.StateStopped
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
				sname := string(event.Kv.Key)
				if event.IsCreate() {
					// Create -> simply create and add it to registry
					m.startScaler(sname, event.Kv.Value)
				} else if event.IsModify() {
					// Update -> force recreate scaler
					if _, ok := m.rgt.Get(sname); ok {
						m.stopScaler(sname)
						m.startScaler(sname, event.Kv.Value)
					}
				} else if event.Type == etcdv3.EventTypeDelete {
					// Delete -> remove from registry and stop the goroutine
					if _, ok := m.rgt.Get(sname); ok {
						m.stopScaler(sname)
					}
				}
			}
		default:
		}
	}
}

func (m *Manager) stopScaler(name string) {
	s, _ := m.rgt.Get(name)
	if local, worker, ok := m.cluster.LocalIsWorker(name); !ok {
		level.Debug(m.logger).Log("msg", "Ignoring scaler, another worker node takes it",
			"scaler", name, "local", local, "worker", worker)
		return
	}
	level.Info(m.logger).Log("msg", "Removing scaler", "name", name)
	s.Stop()
	m.rgt.Delete(name)
	exporter.ReportNumScalers(cluster.ClusterID, -1)
}

func (m *Manager) startScaler(name string, data []byte) {
	if local, worker, ok := m.cluster.LocalIsWorker(name); !ok {
		level.Debug(m.logger).Log("msg", "Ignoring scaler, another worker node takes it",
			"scaler", name, "local", local, "worker", worker)
		return
	}
	level.Info(m.logger).Log("msg", "Creating scaler", "name", name)
	backend, err := m.getBackend(name)
	if err != nil {
		level.Error(m.logger).Log("msg", "Error creating registry backend for scaler",
			"name", name, "err", err)
		return
	}
	s := newScaler(log.With(m.logger, "scaler", name), data, backend)
	m.rgt.Set(name, s)
	go func() {
		m.wg.Add(1)
		s.run(context.Background(), m.wg)
		exporter.ReportNumScalers(cluster.ClusterID, 1)
	}()
}

func (m *Manager) getBackend(key string) (metrics.Backend, error) {
	// There is format -> Cloud provider id
	providerID := strings.Split(key, "/")[2]
	resp, err := m.etcdcli.DoGet(common.Path(model.DefaultCloudPrefix, providerID))
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
		go func(i common.RegistryItem) {
			defer func() {
				m.stopScaler(i.Name)
				m.wg.Done()
			}()
			switch it := i.Value.(type) {
			case *Scaler:
				it.Alert = &it.alert.State
			default:
				level.Error(m.logger).Log("msg", "Registry can contains only Scalers",
					"name", i.Name)
				return
			}
			raw, err := json.Marshal(&i.Value)
			if err != nil {
				level.Error(m.logger).Log("msg", "Error serializing scaler object",
					"name", i.Name, "err", err)
				return
			}
			_, err = m.etcdcli.DoPut(i.Name, string(raw))
			if err != nil {
				level.Error(m.logger).Log("msg", "Error putting scaler object",
					"name", i.Name, "err", err)
				return
			}
		}(i)
	}
}

func (m *Manager) load() {
	resp, err := m.etcdcli.DoGet(model.DefaultScalerPrefix, etcdv3.WithPrefix(),
		etcdv3.WithSort(etcdv3.SortByKey, etcdv3.SortAscend))
	if err != nil {
		level.Error(m.logger).Log("msg", "Error getting scalers", "err", err)
		return
	}
	var sname string
	for _, ev := range resp.Kvs {
		sname = string(ev.Key)
		m.startScaler(sname, ev.Value)
	}
}

func (m *Manager) rebalance() {
	resp, err := m.etcdcli.DoGet(model.DefaultScalerPrefix, etcdv3.WithPrefix(),
		etcdv3.WithSort(etcdv3.SortByKey, etcdv3.SortAscend))
	if err != nil {
		level.Error(m.logger).Log("msg", "Error getting scalers", "err", err)
		return
	}

	var wg sync.WaitGroup
	for _, ev := range resp.Kvs {
		wg.Add(1)
		go func(ev *mvccpb.KeyValue) {
			defer wg.Done()
			name := string(ev.Key)
			local, worker, ok1 := m.cluster.LocalIsWorker(name)
			scaler, ok2 := m.rgt.Get(name)

			if !ok1 {
				if ok2 {
					switch s := scaler.(type) {
					case *Scaler:
						s.Alert = &s.alert.State
					default:
						level.Error(m.logger).Log("msg", "Registry can contains only Scalers",
							"name", name)
						return
					}
					raw, err := json.Marshal(&scaler)
					if err != nil {
						level.Error(m.logger).Log("msg", "Error serializing scaler object",
							"name", name, "err", err)
						return
					}
					_, err = m.etcdcli.DoPut(name, string(raw))
					if err != nil {
						level.Error(m.logger).Log("msg", "Error putting scaler object",
							"name", name, "err", err)
						return
					}
					level.Info(m.logger).Log("msg", "Removing scaler, another worker node takes it",
						"scaler", name, "local", local, "worker", worker)
					scaler.Stop()
					m.rgt.Delete(name)
				}
			} else if !ok2 {
				m.startScaler(name, ev.Value)
			}
		}(ev)
	}

	wg.Wait()
}
