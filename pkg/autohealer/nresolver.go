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
	"encoding/json"
	"sync"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	
	"github.com/vCloud-DFTBA/faythe/pkg/metrics"
	"github.com/vCloud-DFTBA/faythe/pkg/model"
)

// NResolver stands for name resolver
// it collects information from metrics which map instance IP to instance name
type NResolver struct {
	model.NResolver
	logger  log.Logger
	mtx     sync.RWMutex
	done    chan struct{}
	backend metrics.Backend
}

func newNResolver(l log.Logger, data []byte, b metrics.Backend) *NResolver {
	nr := &NResolver{
		logger:  l,
		done:    make(chan struct{}),
		backend: b,
	}
	json.Unmarshal(data, nr)
	return nr
}

func (nr *NResolver) run(ctx context.Context, wg *sync.WaitGroup, nc *chan NodeMetric) {
	interval, _ := time.ParseDuration(nr.Interval)
	ticker := time.NewTicker(interval)
	defer func() {
		wg.Done()
		ticker.Stop()
	}()
	for {
		select {
		case <-ticker.C:
			result, err := nr.backend.QueryInstant(ctx, model.DefaultNResolverQuery, time.Now())
			if err != nil {
				level.Error(nr.logger).Log("msg", "Executing query failed",
					"query", model.DefaultNResolverQuery, "err", err)
				continue
			}
			level.Debug(nr.logger).Log("msg", "Execcuting query success", "query", model.DefaultNResolverQuery)
			nr.mtx.Lock()
			for _, el := range result {
				j, err := el.MarshalJSON()
				if err != nil {
					level.Error(nr.logger).Log("msg", "Error while unmarshalling metrics result", "err", err)
				}
				nm := NodeMetric{
					CloudID: nr.CloudID,
				}
				err = json.Unmarshal(j, &nm)
				if err != nil {
					level.Error(nr.logger).Log("msg", "Error while unmarshalling metrics result", "err", err)
				}
				*nc <- nm
			}
			nr.mtx.Unlock()
		case <-nr.done:
			return
		}
	}

}

// Stop destroys name resolver instance
func (nr *NResolver) Stop() {
	level.Debug(nr.logger).Log("msg", "NResolver is stopping", "id", nr.ID)
	close(nr.done)
	level.Debug(nr.logger).Log("msg", "NResolver is stopped", "id", nr.ID)
}
