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
	"sync"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/ntk148v/faythe/pkg/model"
	etcdv3 "go.etcd.io/etcd/clientv3"
)

type NRManager struct {
	logger  log.Logger
	rqt     *Registry
	stop    chan struct{}
	etcdcli *etcdv3.Client
	watch   etcdv3.WatchChan
	ctx     context.Context
	cancel  context.CancelFunc
	wg      *sync.WaitGroup
}

func NewNRManager(l log.Logger, e *etcdv3.Client) *NRManager {
	ctx, cancel := context.WithCancel(context.Background())
	nrm := &NRManager{
		logger:  l,
		rqt:     &Registry{items: make(map[string]*NResolver)},
		stop:    make(chan struct{}),
		etcdcli: e,
		ctx:     ctx,
		cancel:  cancel,
		wg:      &sync.WaitGroup{},
	}
	nrm.watch = nrm.etcdcli.Watch(nrm.ctx, model.DefaultNResolverPrefix, etcdv3.WithPrefix())
	nrm.load()
	return nrm
}

func (nrm *NRManager) load() {
	r, err := nrm.etcdcli.Get(nrm.ctx, model.DefaultNResolverPrefix, etcdv3.WithPrefix())
	if err != nil {
		level.Error(nrm.logger).Log("msg", "Error getting list NResolver", "err", err)
		return
	}
	for _, e := range r.Kvs {
		nrm.startNResolver(string(e.Key), e.Value)
	}
}

func (nrm *NRManager) startNResolver(name string, data []byte) {
	level.Info(nrm.logger).Log("msg", "Creating name resovler", "name", name)
	nr := newNResolver(log.With(nrm.logger, "nresolver", name), data)
	nrm.rqt.Set(name, nr)
	go func() {
		nrm.wg.Add(1)
		nr.run(nrm.ctx, nrm.wg)
	}()
}

func (nrm *NRManager) stopNResolver(name string) {
	if nr, ok := nrm.rqt.Get(name); ok {
		level.Info(nrm.logger).Log("msg", "Removing name resolver", "name", name)
		nr.stop()
		nrm.rqt.Delete(name)
	}
}

func (nrm *NRManager) Stop() {
	level.Info(nrm.logger).Log("msg", "Cleaning before stopping name resolver managger")
	nrm.save()
	nrm.cancel()
	close(nrm.stop)
	level.Info(nrm.logger).Log("msg", "Name resolver manager is stopped!")
}

func (nrm *NRManager) save() {
	for e := range nrm.rqt.Iter() {
		nrm.wg.Add(1)
		go func() {
			defer func() {
				nrm.stopNResolver(e.Name)
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
		}()
	}
}

func (nrm *NRManager) Run() {
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
		}
	}
}
