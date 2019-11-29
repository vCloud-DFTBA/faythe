// Copyright (c) 2019 Dat Vu Tuan <tuandatk25a@gmail.com>
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

package autohealer

import (
	"context"
	"crypto"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/vCloud-DFTBA/faythe/pkg/metrics"
	"github.com/vCloud-DFTBA/faythe/pkg/model"
	"github.com/vCloud-DFTBA/faythe/pkg/utils"
	etcdv3 "go.etcd.io/etcd/clientv3"
)

// Manager controls name resolver and healer instances
type Manager struct {
	logger  log.Logger
	rqt     *utils.Registry
	stop    chan struct{}
	etcdcli *etcdv3.Client
	watchc  etcdv3.WatchChan
	watchh  etcdv3.WatchChan
	ctx     context.Context
	cancel  context.CancelFunc
	wg      *sync.WaitGroup
	nodes   map[string]string
	ncin    chan NodeMetric
	ncout   chan map[string]string
}

// NewManager create new Manager for name resolver and healer
func NewManager(l log.Logger, e *etcdv3.Client) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	hm := &Manager{
		logger:  l,
		rqt:     &utils.Registry{Items: make(map[string]utils.Worker)},
		stop:    make(chan struct{}),
		etcdcli: e,
		ctx:     ctx,
		cancel:  cancel,
		wg:      &sync.WaitGroup{},
		nodes:   make(map[string]string),
		ncin:    make(chan NodeMetric),
		ncout:   make(chan map[string]string),
	}
	hm.watchc = hm.etcdcli.Watch(hm.ctx, model.DefaultCloudPrefix, etcdv3.WithPrefix())
	hm.watchh = hm.etcdcli.Watch(hm.ctx, model.DefaultHealerPrefix, etcdv3.WithPrefix())
	hm.load()
	return hm
}

func (hm *Manager) load() {
	for _, p := range []string{model.DefaultNResolverPrefix, model.DefaultHealerPrefix} {
		r, err := hm.etcdcli.Get(hm.ctx, p, etcdv3.WithPrefix())
		if err != nil {
			level.Error(hm.logger).Log("msg", "Error getting list Workers", "err", err)
			return
		}
		for _, e := range r.Kvs {
			hm.startWorker(p, string(e.Key), e.Value)
		}
	}
}

func (hm *Manager) startWorker(p string, name string, data []byte) {
	level.Info(hm.logger).Log("msg", "Creating worker", "name", name)
	backend, err := hm.getBackend(name)
	if err != nil {
		level.Error(hm.logger).Log("msg", "Error creating registry backend for worker",
			"id", name, "err", err)
		return
	}
	if p == model.DefaultNResolverPrefix {
		nr := newNResolver(log.With(hm.logger, "nresolver", name), data, backend)
		hm.rqt.Set(name, nr)
		go func() {
			hm.wg.Add(1)
			nr.run(hm.ctx, hm.wg, &hm.ncin)
		}()
	} else {
		atengine, err := hm.getATEngine(name)
		if err != nil {
			level.Error(hm.logger).Log("msg", "Error getting automation engine for worker",
				"id", name, "err", err)
		}
		h := newHealer(log.With(hm.logger, "healer", name), data, backend, atengine)
		hm.rqt.Set(name, h)
		go func() {
			hm.wg.Add(1)
			h.run(hm.ctx, hm.wg, hm.ncout)
		}()
	}
}

func (hm *Manager) stopWorker(name string) {
	if w, ok := hm.rqt.Get(name); ok {
		level.Info(hm.logger).Log("msg", "Removing worker", "name", name)
		w.Stop()
		hm.rqt.Delete(name)
	}
}

// Stop destroy name resolver, healer and itself
func (hm *Manager) Stop() {
	level.Info(hm.logger).Log("msg", "Cleaning before stopping autohealer managger")
	hm.save()
	hm.wg.Wait()
	close(hm.stop)
	hm.cancel()
	level.Info(hm.logger).Log("msg", "Autohealer manager is stopped!")
}

func (hm *Manager) save() {
	for e := range hm.rqt.Iter() {
		hm.wg.Add(1)
		go func(e utils.RegistryItem) {
			defer func() {
				hm.stopWorker(e.Name)
				hm.wg.Done()
			}()

			raw, err := json.Marshal(&e.Value)
			if err != nil {
				level.Error(hm.logger).Log("msg", "Error while marshalling worker object",
					"name", e.Name, "err", err)
				return
			}
			_, err = hm.etcdcli.Put(hm.ctx, e.Name, string(raw))
			if err != nil {
				level.Error(hm.logger).Log("msg", "Error putting worker object",
					"name", e.Name, "err", err)
				return
			}
		}(e)
	}
}

// Run start healer mamaner instance
func (hm *Manager) Run() {
	for {
		select {
		case <-hm.stop:
			return
		case watchResp := <-hm.watchc:
			for _, event := range watchResp.Events {
				name := utils.Path(model.DefaultNResolverPrefix, strings.Split(string(event.Kv.Key), "/")[2],
					utils.Hash(strings.Split(string(event.Kv.Key), "/")[2], crypto.MD5))
				if event.IsCreate() {
					cloud := model.Cloud{}
					err := json.Unmarshal(event.Kv.Value, &cloud)
					if err != nil {
						level.Error(hm.logger).Log("msg", "Error while unmarshalling cloud object", "err", err)
					}
					// NResolver
					nr := model.NResolver{
						ID:      utils.Hash(cloud.ID, crypto.MD5),
						Monitor: cloud.Monitor,
						CloudID: cloud.ID,
					}
					nr.Validate()
					raw, err := json.Marshal(nr)
					if err != nil {
						level.Error(hm.logger).Log("msg", "Error while marshalling nresolver object", "err", err)
					}
					hm.etcdcli.Put(hm.ctx, name, string(raw))
					hm.startWorker(model.DefaultNResolverPrefix, name, raw)
				}
				if event.Type == etcdv3.EventTypeDelete {
					hm.stopWorker(name)
					hm.etcdcli.Delete(hm.ctx, name, etcdv3.WithPrefix())
					hname := strings.ReplaceAll(name, model.DefaultNResolverPrefix, model.DefaultHealerPrefix)
					hm.stopWorker(hname)
					hm.etcdcli.Delete(hm.ctx, hname, etcdv3.WithPrefix())
				}
			}
		case watchResp := <-hm.watchh:
			for _, event := range watchResp.Events {
				name := utils.Path(model.DefaultHealerPrefix, strings.Split(string(event.Kv.Key), "/")[2],
					utils.Hash(strings.Split(string(event.Kv.Key), "/")[2], crypto.MD5))
				if event.IsCreate() {
					hm.startWorker(model.DefaultHealerPrefix, name, event.Kv.Value)
				}
				if event.Type == etcdv3.EventTypeDelete {
					hm.stopWorker(name)
				}
			}
		case nm := <-hm.ncin:
			hm.nodes[MakeKey(nm.CloudID, strings.Split(nm.Metric.Instance, ":")[0])] = nm.Metric.Nodename
		case nm := <-hm.ncout:
			if len(hm.nodes) != 0 {
				if m, ok := hm.nodes[nm["instance"]]; ok {
					hm.ncout <- map[string]string{nm["instance"]: m}
				}
			}
		default:
		}
	}
}

func (hm *Manager) getBackend(key string) (metrics.Backend, error) {
	// There is format -> Cloud provider id
	providerID := strings.Split(key, "/")[2]
	resp, err := hm.etcdcli.Get(hm.ctx, utils.Path(model.DefaultCloudPrefix, providerID))
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

func (hm *Manager) getATEngine(key string) (model.ATEngine, error) {
	providerID := strings.Split(key, "/")[2]
	resp, err := hm.etcdcli.Get(hm.ctx, utils.Path(model.DefaultCloudPrefix, providerID))
	if err != nil {
		return model.ATEngine{}, err
	}
	value := resp.Kvs[0]
	var (
		cloud    model.Cloud
		atengine model.ATEngine
	)
	err = json.Unmarshal(value.Value, &cloud)
	if err != nil {
		return model.ATEngine{}, err
	}
	switch cloud.Provider {
	case model.OpenStackType:
		var ops model.OpenStack
		err = json.Unmarshal(value.Value, &ops)
		if err != nil {
			return model.ATEngine{}, err
		}

		atengine = ops.ATEngine
	default:
	}
	return atengine, nil
}

// MakeKey creates key from id and instance
func MakeKey(id string, instance string) string {
	return fmt.Sprintf("%s/%s", id, instance)
}
