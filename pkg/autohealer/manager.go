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
	"strings"
	"sync"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	etcdv3 "go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/mvcc/mvccpb"

	"github.com/vCloud-DFTBA/faythe/pkg/cluster"
	"github.com/vCloud-DFTBA/faythe/pkg/common"
	"github.com/vCloud-DFTBA/faythe/pkg/exporter"
	"github.com/vCloud-DFTBA/faythe/pkg/metrics"
	"github.com/vCloud-DFTBA/faythe/pkg/model"
)

// Manager controls name resolver and healer instances
type Manager struct {
	logger  log.Logger
	rqt     *common.Registry
	stop    chan struct{}
	etcdcli *common.Etcd
	watchc  etcdv3.WatchChan
	watchh  etcdv3.WatchChan
	ncin    map[string]chan map[string]string
	ncout   map[string]chan map[string]string
	cluster *cluster.Cluster
	state   model.State
}

// NewManager create new Manager for name resolver and healer
func NewManager(l log.Logger, e *common.Etcd, c *cluster.Cluster) *Manager {
	hm := &Manager{
		logger:  l,
		rqt:     &common.Registry{Items: make(map[string]common.Worker)},
		stop:    make(chan struct{}),
		etcdcli: e,
		ncin:    make(map[string]chan map[string]string),
		ncout:   make(map[string]chan map[string]string),
		cluster: c,
	}
	exporter.ReportNumberOfHealers(cluster.GetID(), 0)
	hm.load()
	hm.state = model.StateActive
	return hm
}

// Reload stops and starts healers
func (hm *Manager) Reload() {
	level.Info(hm.logger).Log("msg", "Reloading...")
	hm.rebalance()
	level.Info(hm.logger).Log("msg", "Reloaded")
}

func (hm *Manager) load() {
	for _, p := range []string{model.DefaultNResolverPrefix, model.DefaultHealerPrefix} {
		r, err := hm.etcdcli.DoGet(p, etcdv3.WithPrefix())
		if err != nil {
			level.Error(hm.logger).Log("msg", "Error getting list Workers", "err", err)
			return
		}
		var hname string
		for _, e := range r.Kvs {
			hname = string(e.Key)
			providerID := strings.Split(hname, "/")[2]
			if ok := hm.etcdcli.CheckKey(common.Path(model.DefaultCloudPrefix, providerID)); !ok {
				err = errors.Errorf("unable to find provider %s for healer worker %s", providerID, hname)
				level.Error(hm.logger).Log("msg", err.Error())
				continue
			}
			hm.startWorker(p, string(e.Key), e.Value)
		}
	}
}

func (hm *Manager) startWorker(p string, name string, data []byte) {
	if local, worker, ok := hm.cluster.LocalIsWorker(name); !ok && p == model.DefaultHealerPrefix {
		level.Debug(hm.logger).Log("msg", "Ignoring healer, another worker node takes it",
			"healer", name, "local", local, "worker", worker)
		return
	}
	level.Info(hm.logger).Log("msg", "Creating worker", "name", name)
	backend, err := metrics.GetBackend(hm.etcdcli, strings.Split(name, "/")[2])
	if err != nil {
		level.Error(hm.logger).Log("msg", "Error creating registry backend for worker",
			"name", name, "err", err)
		return
	}
	if p == model.DefaultNResolverPrefix {
		nr := newNResolver(log.With(hm.logger, "nresolver", name), data, backend)
		hm.rqt.Set(name, nr)
		hm.ncin[nr.CloudID] = make(chan map[string]string)
		go nr.run(context.Background(), hm.ncin[nr.CloudID])
	} else {
		h := newHealer(log.With(hm.logger, "healer", name), data, backend)
		hm.rqt.Set(name, h)
		hm.ncout[h.CloudID] = make(chan map[string]string)
		go h.run(context.Background(), hm.etcdcli, hm.ncout[h.CloudID])
		go hm.pubSubNodes(h.CloudID)
	}
}

func (hm *Manager) stopWorker(name string) {
	w, _ := hm.rqt.Get(name)
	if local, worker, ok := hm.cluster.LocalIsWorker(name); !ok && strings.Contains(name, model.DefaultHealerPrefix) {
		level.Debug(hm.logger).Log("msg", "Ignoring healing worker, another worker node takes it",
			"healing_worker", name, "local", local, "worker", worker)
		return
	}
	level.Info(hm.logger).Log("msg", "Removing healing worker", "name", name)

	w.Stop()
	hm.rqt.Delete(name)
}

// Stop destroy name resolver, healer and itself
func (hm *Manager) Stop() {
	// Ignore close channel if manager is already stopped/stopping
	if hm.state == model.StateStopping || hm.state == model.StateStopped {
		return
	}
	level.Info(hm.logger).Log("msg", "Cleaning before stopping autohealer managger")
	hm.state = model.StateStopping
	close(hm.stop)
	hm.save()
	hm.state = model.StateStopped
	level.Info(hm.logger).Log("msg", "Autohealer manager is stopped!")
}

func (hm *Manager) save() {
	var wg sync.WaitGroup
	for e := range hm.rqt.Iter() {
		wg.Add(1)
		go func(e common.RegistryItem) {
			defer func() {
				hm.stopWorker(e.Name)
				wg.Done()
			}()

			raw, err := json.Marshal(&e.Value)
			if err != nil {
				level.Error(hm.logger).Log("msg", "Error while marshalling worker object",
					"name", e.Name, "err", err)
				return
			}
			_, err = hm.etcdcli.DoPut(e.Name, string(raw))
			if err != nil {
				level.Error(hm.logger).Log("msg", "Error putting worker object",
					"name", e.Name, "err", err)
				return
			}
		}(e)
	}

	// Wait for all healer workers to shut down.
	wg.Wait()
}

// Run start healer mananer instance
func (hm *Manager) Run() {
	ctxc, cancelc := hm.etcdcli.WatchContext()
	hm.watchc = hm.etcdcli.Watch(ctxc, model.DefaultCloudPrefix, etcdv3.WithPrefix())
	ctxh, cancelh := hm.etcdcli.WatchContext()
	hm.watchh = hm.etcdcli.Watch(ctxh, model.DefaultHealerPrefix, etcdv3.WithPrefix())
	defer func() {
		cancelc()
		cancelh()
	}()

	for {
		select {
		case <-hm.stop:
			return
		case watchResp := <-hm.watchc:
			if err := watchResp.Err(); err != nil {
				level.Error(hm.logger).Log("msg", "Error watching etcd cloud provider keys", "err", err)
				eerr := common.NewEtcdErr(model.DefaultCloudPrefix, "watch", err)
				hm.etcdcli.ErrCh <- eerr
				break
			}

			for _, event := range watchResp.Events {
				name := common.Path(model.DefaultNResolverPrefix, strings.Split(string(event.Kv.Key), "/")[2],
					common.Hash(strings.Split(string(event.Kv.Key), "/")[2], crypto.MD5))
				if event.IsCreate() {
					cloud := model.Cloud{}
					err := json.Unmarshal(event.Kv.Value, &cloud)
					if err != nil {
						level.Error(hm.logger).Log("msg", "Error while unmarshalling cloud object", "err", err)
					}
					// NResolver
					nr := model.NResolver{
						ID:      common.Hash(cloud.ID, crypto.MD5),
						Monitor: cloud.Monitor,
						CloudID: cloud.ID,
					}
					_ = nr.Validate()
					raw, err := json.Marshal(nr)
					if err != nil {
						level.Error(hm.logger).Log("msg", "Error while marshalling nresolver object", "err", err)
					}
					_, err = hm.etcdcli.DoPut(name, string(raw))
					if err != nil {
						level.Error(hm.logger).Log("msg", "Error putting nresolver object", "err", err)
					}
					hm.startWorker(model.DefaultNResolverPrefix, name, raw)
				}
				if event.Type == etcdv3.EventTypeDelete {
					if _, ok := hm.rqt.Get(name); ok {
						hm.stopWorker(name)
						_, err := hm.etcdcli.DoDelete(name, etcdv3.WithPrefix())
						if err != nil {
							level.Error(hm.logger).Log("msg", "Error deleting nresolver object", "err", err)
						}
					}
					hname := strings.ReplaceAll(name, model.DefaultNResolverPrefix, model.DefaultHealerPrefix)
					_, err := hm.etcdcli.DoDelete(hname, etcdv3.WithPrefix())
					if err != nil {
						level.Error(hm.logger).Log("msg", "Error deleting healer object", "err", err)
					}
				}
			}
		case watchResp := <-hm.watchh:
			if err := watchResp.Err(); err != nil {
				level.Error(hm.logger).Log("msg", "Error watching etcd healer keys", "err", err)
				eerr := common.NewEtcdErr(model.DefaultHealerPrefix, "watch", err)
				hm.etcdcli.ErrCh <- eerr
				break
			}

			for _, event := range watchResp.Events {
				name := common.Path(model.DefaultHealerPrefix, strings.Split(string(event.Kv.Key), "/")[2],
					common.Hash(strings.Split(string(event.Kv.Key), "/")[2], crypto.MD5))
				if event.IsCreate() {
					hm.startWorker(model.DefaultHealerPrefix, name, event.Kv.Value)
				}
				if event.Type == etcdv3.EventTypeDelete {
					if _, ok := hm.rqt.Get(name); ok {
						hm.stopWorker(name)
					}
				}
			}
		}
	}
}

// pubSubNodes receives the map info ({instance-ip: compute-name} from NResolver
// and sends it to Healer
func (hm *Manager) pubSubNodes(cloudID string) {
	for {
		select {
		case <-hm.stop:
			return
		default:
			nm := <-hm.ncin[cloudID]
			if _, ok := hm.ncout[cloudID]; ok {
				hm.ncout[cloudID] <- nm
			}
		}
	}
}

func (hm *Manager) rebalance() {
	resp, err := hm.etcdcli.DoGet(model.DefaultHealerPrefix, etcdv3.WithPrefix(),
		etcdv3.WithSort(etcdv3.SortByKey, etcdv3.SortAscend))
	if err != nil {
		level.Error(hm.logger).Log("msg", "Error getting healers", "err", err)
		return
	}

	var wg sync.WaitGroup
	for _, ev := range resp.Kvs {
		wg.Add(1)
		go func(ev *mvccpb.KeyValue) {
			defer wg.Done()
			name := string(ev.Key)
			local, worker, ok1 := hm.cluster.LocalIsWorker(name)
			healer, ok2 := hm.rqt.Get(name)

			if !ok1 {
				if ok2 {
					raw, err := json.Marshal(&healer)
					if err != nil {
						level.Error(hm.logger).Log("msg", "Error serializing healer object",
							"name", name, "err", err)
						return
					}
					_, err = hm.etcdcli.DoPut(name, string(raw))
					if err != nil {
						level.Error(hm.logger).Log("msg", "Error putting healer object",
							"key", name, "err", err)
						return
					}
					level.Info(hm.logger).Log("msg", "Removing healer, another worker node takes it",
						"healer", name, "local", local, "worker", worker)
					healer.Stop()
					hm.rqt.Delete(name)
				}
			} else if !ok2 {
				hm.startWorker(model.DefaultHealerPrefix, name, ev.Value)
			}
		}(ev)
	}
	wg.Wait()
}
