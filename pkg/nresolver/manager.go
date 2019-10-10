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

package nresolver

import (
	"context"
	"encoding/json"
	"strings"
	"sync"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/ntk148v/faythe/pkg/model"
	"github.com/ntk148v/faythe/pkg/utils"
	etcdv3 "go.etcd.io/etcd/clientv3"
)

type Manager struct {
	logger  log.Logger
	rqt     *utils.Registry
	stop    chan struct{}
	etcdcli *etcdv3.Client
	watch   etcdv3.WatchChan
	ctx     context.Context
	cancel  context.CancelFunc
	wg      *sync.WaitGroup
	nodes   map[string]string
	nc      chan NodeMetric
}

func NewManager(l log.Logger, e *etcdv3.Client) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	nrm := &Manager{
		logger:  l,
		rqt:     &utils.Registry{Items: make(map[string]utils.Worker)},
		stop:    make(chan struct{}),
		etcdcli: e,
		ctx:     ctx,
		cancel:  cancel,
		wg:      &sync.WaitGroup{},
		nodes:   make(map[string]string),
		nc:      make(chan NodeMetric),
	}
	nrm.watch = nrm.etcdcli.Watch(nrm.ctx, model.DefaultNResolverPrefix, etcdv3.WithPrefix())
	nrm.load()
	return nrm
}

func (nrm *Manager) load() {
	r, err := nrm.etcdcli.Get(nrm.ctx, model.DefaultNResolverPrefix, etcdv3.WithPrefix())
	if err != nil {
		level.Error(nrm.logger).Log("msg", "Error getting list NResolver", "err", err)
		return
	}
	for _, e := range r.Kvs {
		nrm.startNResolver(string(e.Key), e.Value)
	}
}

func (nrm *Manager) startNResolver(name string, data []byte) {
	level.Info(nrm.logger).Log("msg", "Creating name resovler", "name", name)
	nr := newNResolver(log.With(nrm.logger, "nresolver", name), data)
	nrm.rqt.Set(name, nr)
	go func() {
		nrm.wg.Add(1)
		nr.run(nrm.ctx, nrm.wg, &nrm.nc)
	}()
}

func (nrm *Manager) stopNResolver(name string) {
	if nr, ok := nrm.rqt.Get(name); ok {
		level.Info(nrm.logger).Log("msg", "Removing name resolver", "name", name)
		nr.Stop()
		nrm.rqt.Delete(name)
	}
}

func (nrm *Manager) Stop() {
	level.Info(nrm.logger).Log("msg", "Cleaning before stopping name resolver managger")
	nrm.save()
	nrm.wg.Wait()
	close(nrm.stop)
	nrm.cancel()
	level.Info(nrm.logger).Log("msg", "Name resolver manager is stopped!")
}

func (nrm *Manager) save() {
	for e := range nrm.rqt.Iter() {
		nrm.wg.Add(1)
		go func(name string) {
			defer func() {
				nrm.stopNResolver(name)
				nrm.wg.Done()
			}()

			raw, err := json.Marshal(&e.Value)
			if err != nil {
				level.Error(nrm.logger).Log("msg", "Error while serializing name resolver object",
					"name", e.Name, "err", err)
				return
			}
			_, err = nrm.etcdcli.Put(nrm.ctx, e.Name, string(raw))
			if err != nil {
				level.Error(nrm.logger).Log("msg", "Error putting name resolver object",
					"name", e.Name, "err", err)
				return
			}
		}(e.Name)
	}
}

func (nrm *Manager) Run() {
	for {
		select {
		case <-nrm.stop:
			return
		case watchResp := <-nrm.watch:
			for _, event := range watchResp.Events {
				name := string(event.Kv.Key)
				if event.IsCreate() {
					nrm.startNResolver(name, event.Kv.Value)
				}
				if event.Type == etcdv3.EventTypeDelete {
					nrm.stopNResolver(name)
				}
			}
		case nm := <-nrm.nc:
			nrm.nodes[strings.Split(nm.Metric.Instance, ":")[0]] = nm.Metric.Nodename
		}
	}
}
