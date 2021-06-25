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

	"github.com/vCloud-DFTBA/faythe/pkg/cluster"
	"github.com/vCloud-DFTBA/faythe/pkg/common"
	"github.com/vCloud-DFTBA/faythe/pkg/exporter"
	"github.com/vCloud-DFTBA/faythe/pkg/metrics"
	"github.com/vCloud-DFTBA/faythe/pkg/model"
)

// NResolver stands for name resolver
// it collects information from metrics backend which map instance IP to instance name
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
	_ = json.Unmarshal(data, nr)
	return nr
}

func (nr *NResolver) run(ctx context.Context, nc chan map[string]string) {
	interval, _ := common.ParseDuration(nr.Interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	// Record the number of healers
	exporter.ReportNumberOfNResolvers(cluster.GetID(), 1)

	// A trick to run it immediately to fetch the list of nodes
	// before the corresponding healer is up.
	// There is no way to create ticker with instance first tick
	// https://github.com/golang/go/issues/17601
	doWork := func() {
		result, err := nr.backend.QueryInstant(ctx, model.DefaultNResolverQuery, time.Now())
		if err != nil {
			level.Error(nr.logger).Log("msg", "Execute query failed",
				"query", model.DefaultNResolverQuery, "err", err)
			exporter.ReportMetricQueryFailureCounter(cluster.GetID(),
				nr.backend.GetType(), nr.backend.GetAddress())
			return
		}
		level.Debug(nr.logger).Log("msg", "Execute query success", "query", model.DefaultNResolverQuery)
		nr.mtx.Lock()
		for _, el := range result {
			j, err := el.MarshalJSON()
			if err != nil {
				level.Error(nr.logger).Log("msg", "Error while unmarshalling metrics result", "err", err)
				return
			}
			nm := NodeMetric{
				CloudID: nr.CloudID,
			}
			err = json.Unmarshal(j, &nm)
			if err != nil {
				level.Error(nr.logger).Log("msg", "Error while unmarshalling metrics result", "err", err)
				return
			}
			nc <- map[string]string{nm.Metric.Instance: nm.Metric.Nodename}
		}
		defer nr.mtx.Unlock()
	}
	doWork()
	for {
		select {
		case <-ticker.C:
			doWork()
		case <-nr.done:
			return
		}
	}

}

// Stop destroys name resolver instance
func (nr *NResolver) Stop() {
	level.Debug(nr.logger).Log("msg", "NResolver is stopping")
	close(nr.done)
	exporter.ReportNumberOfNResolvers(cluster.GetID(), -1)
	level.Debug(nr.logger).Log("msg", "NResolver is stopped")
}
