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
	"fmt"
	"sync"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/ntk148v/faythe/pkg/metrics"
	"github.com/ntk148v/faythe/pkg/model"
)

type NResolver struct {
	model.NResolver
	logger log.Logger
	mtx    sync.RWMutex
	done   chan struct{}
}

func newNResolver(l log.Logger, data []byte) *NResolver {
	nr := &NResolver{
		logger: l,
		done:   make(chan struct{}),
	}
	err := json.Unmarshal(data, nr)
	if err != nil {
		level.Error(nr.logger).Log("msg", "Error while unmarhshaling data", "err", err.Error())
	}
	return nr
}

func (nr *NResolver) run(ctx context.Context, wg *sync.WaitGroup, nc *chan NodeMetric) {
	interval, _ := time.ParseDuration(nr.Interval)
	err := metrics.Register(nr.Monitor.Backend, nr.Monitor.Address.String())
	if err != nil {
		level.Error(nr.logger).Log("msg", "Error loading metrics backend, stopping..")
		return
	}
	backend, _ := metrics.Get(fmt.Sprintf("%s-%s", nr.Monitor.Backend, nr.Monitor.Address.String()))
	ticker := time.NewTicker(interval)
	defer func() {
		wg.Done()
		ticker.Stop()
	}()
	for {
		select {
		case <-ticker.C:
			result, err := backend.QueryInstant(ctx, model.DefaultNResolverQuery, time.Now())
			if err != nil {
				level.Error(nr.logger).Log("msg", "Executing query failed",
					"query", model.DefaultNResolverQuery, "err", err.Error())
				continue
			}
			level.Info(nr.logger).Log("msg", "Execcuting query success", "query", model.DefaultNResolverQuery)
			nr.mtx.Lock()
			for _, el := range result {
				j, err := el.MarshalJSON()
				if err != nil {
					level.Error(nr.logger).Log("msg", "Erorr while json-izing metrics result", "err", err.Error())
				}
				nm := NodeMetric{}
				err = json.Unmarshal(j, &nm)
				if err != nil {
					level.Error(nr.logger).Log("msg", "Erorr while json-izing metrics result", "err", err.Error())
				}
				*nc <- nm
			}
			nr.mtx.Unlock()
		case <-nr.done:
			return
		}
	}

}

func (nr *NResolver) Stop() {
	level.Debug(nr.logger).Log("msg", "NResolver is stopping", "name", nr.Name)
	close(nr.done)
	level.Debug(nr.logger).Log("msg", "NResolver is stopped", "name", nr.Name)
}
