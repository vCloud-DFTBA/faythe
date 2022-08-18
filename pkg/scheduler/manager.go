// Copyright (c) 2022 Dat Vu Tuan <tuandatk25a@gmail.com>
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

package scheduler

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	etcdv3 "go.etcd.io/etcd/clientv3"

	"github.com/vCloud-DFTBA/faythe/pkg/cluster"
	"github.com/vCloud-DFTBA/faythe/pkg/common"
	"github.com/vCloud-DFTBA/faythe/pkg/model"
)

const Inverval = "5s"

// Manager manages a set of scheduler instances.
type Manager struct {
	logger  log.Logger
	rgt     *common.Registry
	stop    chan struct{}
	etcdcli *common.Etcd
	cluster *cluster.Cluster
	state   model.State
}

// NewManager returns a scheduler Manager
func NewManager(l log.Logger, e *common.Etcd, c *cluster.Cluster) *Manager {
	m := &Manager{
		logger:  l,
		rgt:     &common.Registry{Items: make(map[string]common.Worker)},
		stop:    make(chan struct{}),
		etcdcli: e,
		cluster: c,
	}
	m.load()
	m.state = model.StateActive
	return m
}

// Reload simply stop and start schedulers selectively.
func (m *Manager) Reload() {
	level.Info(m.logger).Log("msg", "Reloading...")
	m.rebalance()
	level.Info(m.logger).Log("msg", "Reloaded")
}

// Stop the manager
func (m *Manager) Stop() {
	// Ignore close channel if manager is already stopped/stopping
	if m.state == model.StateStopping || m.state == model.StateStopped {
		return
	}
	level.Info(m.logger).Log("msg", "Stopping scheduler manager...")
	m.state = model.StateStopping
	close(m.stop)
	m.save()
	m.state = model.StateStopped
	level.Info(m.logger).Log("msg", "Scheduler manager stopped")
}

// Run starts processing of the scheduler manager
func (m *Manager) Run() {
	interval, _ := common.ParseDuration(Inverval)
	ticker := time.NewTicker(interval)

	defer ticker.Stop()

	ctx, cancel := m.etcdcli.WatchContext()
	watch := m.etcdcli.Watch(ctx, model.DefaultSchedulerPrefix, etcdv3.WithPrefix())
	defer func() { cancel() }()

	for {
		select {
		case <-m.stop:
			return
		case watchResp := <-watch:
			if err := watchResp.Err(); err != nil {
				level.Error(m.logger).Log("msg", "Error watching etcd scheduler keys", "err", err)
				eerr := common.NewEtcdErr(model.DefaultSchedulerPrefix, "watch", err)
				m.etcdcli.ErrCh <- eerr
				return
			}

			for _, event := range watchResp.Events {
				sname := string(event.Kv.Key)
				if event.IsCreate() {
					// Create -> simply create and add it to registry
					m.storeScheduler(sname, event.Kv.Value)
				} else if event.IsModify() {
					// Update -> force recreate schedule
					if _, ok := m.rgt.Get(sname); ok {
						m.removeScheduler(sname)
						m.storeScheduler(sname, event.Kv.Value)
					}
				} else if event.Type == etcdv3.EventTypeDelete {
					// Delete -> remove from registry and stop the goroutine
					if _, ok := m.rgt.Get(sname); ok {
						m.removeScheduler(sname)
					}
				}
			}
		case <-ticker.C:
			for s := range m.rgt.Iter() {
				switch it := s.Value.(type) {
				case *Scheduler:
					scheduler := it.Scheduler
					if scheduler.FromNextExec.Sub(time.Now()) < interval {
						it.Do()
					}
					if scheduler.ToNextExec.Sub(time.Now()) < interval {
						it.Do()
					}

				default:
					level.Error(m.logger).Log("msg", "Registry can contains only Schedulers",
						"name", s.Name)
					return
				}
			}
		}
	}
}

func (m *Manager) removeScheduler(name string) {
	level.Info(m.logger).Log("msg", "Removing schedule", "name", name)
	m.rgt.Delete(name)
}

func (m *Manager) storeScheduler(name string, data []byte) {
	if local, worker, ok := m.cluster.LocalIsWorker(name); !ok {
		level.Debug(m.logger).Log("msg", "Ignoring scheduler, another worker node takes it",
			"scheduler", name, "local", local, "worker", worker)
		return
	}
	level.Info(m.logger).Log("msg", "Creating schedule", "name", name)
	// Extract Cloud provider id from Etcd key
	providerID := strings.Split(name, "/")[2]
	s := newScheduler(log.With(m.logger, "scheduler", name), data)
	// For backward compability, insert CloudID if isn't existing.
	if s.CloudID == "" {
		s.CloudID = providerID
	}
	m.rgt.Set(name, s)
}

// save puts schedulers to etcd
func (m *Manager) save() {
	for i := range m.rgt.Iter() {
		m.removeScheduler(i.Name)
		raw, err := json.Marshal(&i.Value)
		if err != nil {
			level.Error(m.logger).Log("msg", "Error serializing scheduler object", "name", i.Name, "err", err)
			return
		}
		_, err = m.etcdcli.DoPut(i.Name, string(raw))
		if err != nil {
			level.Error(m.logger).Log("msg", "Error putting scheduler object", "name", i.Name, "err", err)
			return
		}
	}
}

func (m *Manager) load() {
	resp, err := m.etcdcli.DoGet(model.DefaultSchedulerPrefix, etcdv3.WithPrefix(),
		etcdv3.WithSort(etcdv3.SortByKey, etcdv3.SortAscend))
	if err != nil {
		level.Error(m.logger).Log("msg", "error getting scheduler", "err", err)
		return
	}
	var sname string
	for _, ev := range resp.Kvs {
		sname = string(ev.Key)
		// Extract Cloud provider id from Etcd key
		providerID := strings.Split(sname, "/")[2]
		if ok := m.etcdcli.CheckKey(common.Path(model.DefaultCloudPrefix, providerID)); !ok {
			err = errors.Errorf("unable to find provider %s for scheduler %s", providerID, sname)
			level.Error(m.logger).Log("msg", err.Error())
			continue
		}
		m.storeScheduler(sname, ev.Value)
	}
}

func (m *Manager) rebalance() {
	resp, err := m.etcdcli.DoGet(model.DefaultSchedulerPrefix, etcdv3.WithPrefix(),
		etcdv3.WithSort(etcdv3.SortByKey, etcdv3.SortAscend))
	if err != nil {
		level.Error(m.logger).Log("msg", "Error getting schedulers", "err", err)
		return
	}

	for _, ev := range resp.Kvs {
		name := string(ev.Key)
		local, worker, ok1 := m.cluster.LocalIsWorker(name)
		scheduler, ok2 := m.rgt.Get(name)

		if !ok1 {
			if ok2 {
				raw, err := json.Marshal(&scheduler)
				if err != nil {
					level.Error(m.logger).Log("msg", "Error serializing scheduler object",
						"name", name, "err", err)
					return
				}
				_, err = m.etcdcli.DoPut(name, string(raw))
				if err != nil {
					level.Error(m.logger).Log("msg", "Error putting scheduler object",
						"name", name, "err", err)
					return
				}
				level.Info(m.logger).Log("msg", "Removing scheduler, another worker node takes it",
					"scheduler", name, "local", local, "worker", worker)
				scheduler.Stop()
				m.rgt.Delete(name)
			}
		} else if !ok2 {
			m.storeScheduler(name, ev.Value)
		}
	}
}
