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
	pmodel "github.com/prometheus/common/model"
)

type NResolver struct {
	model.NResolver
	logger     log.Logger
	mtx        sync.RWMutex
	done       chan struct{}
	terminated chan struct{}
}

func newNResolver(l log.Logger, data []byte) *NResolver {
	nr := &NResolver{
		logger:     l,
		done:       make(chan struct{}),
		terminated: make(chan struct{}),
	}
	err := json.Unmarshal(data, nr)
	if err != nil {
		level.Error(nr.logger).Log("msg", "Error while unmarhshaling data", "err", err)
	}
	return nr
}

func (nr *NResolver) run(ctx context.Context, wg *sync.WaitGroup) {
	err := metrics.Register("prometheus", nr.Address.String())
	if err != nil {
		level.Error(nr.logger).Log("msg", "Error loading metrics backend, stopping..")
		return
	}
	backend, _ := metrics.Get(fmt.Sprintf("%s-%s", "prometheus", nr.Address.String()))
	interval, _ := time.ParseDuration(nr.Interval)
	defer func() {
		close(nr.terminated)
	}()
	for ticker := time.Tick(interval); ; {
		result, err := backend.QueryInstant(ctx, model.DefaultNResolverQuery, time.Now())
		if err != nil {
			level.Error(nr.logger).Log("msg", "Executing query failed",
				"query", model.DefaultNResolverQuery, "err", err)
			return
		}
		level.Info(nr.logger).Log("msg", "Execcuting query success", "query", model.DefaultNResolverQuery)
		nr.mtx.Lock()
		nr.parseQueryResult(result)
		nr.mtx.Unlock()
		select {
		case <-ticker:
			continue
		case <-nr.done:
			return
		}
	}
}

func (nr *NResolver) parseQueryResult(pm pmodel.Vector) {
	for _, el := range pm {
		j, err := el.MarshalJSON()
		if err != nil {
			level.Error(nr.logger).Log("msg", "Erorr while json-izing metrics result", "err", err)
		}

		nm := NodeMetric{}
		err = json.Unmarshal(j, &nm)
		if err != nil {
			level.Error(nr.logger).Log("msg", "Erorr while json-izing metrics result", "err", err)
		}
		fmt.Println(nm)
	}
}

func (nr *NResolver) stop() {
	level.Debug(nr.logger).Log("msg", "NResolver is stopping", "name", nr.Name)
	close(nr.done)
	<-nr.terminated
	level.Debug(nr.logger).Log("msg", "NResolver is stopped", "name", nr.Name)
}
