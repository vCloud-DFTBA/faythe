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
	"net/http"
	"sync"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"

	"github.com/vCloud-DFTBA/faythe/pkg/alert"
	"github.com/vCloud-DFTBA/faythe/pkg/cluster"
	"github.com/vCloud-DFTBA/faythe/pkg/common"
	"github.com/vCloud-DFTBA/faythe/pkg/exporter"
	"github.com/vCloud-DFTBA/faythe/pkg/metrics"
	"github.com/vCloud-DFTBA/faythe/pkg/model"
)

// Scaler does metric polling and executes scale actions.
type Scaler struct {
	model.Scaler
	alert      *alert.Alert
	logger     log.Logger
	mtx        sync.RWMutex
	done       chan struct{}
	terminated chan struct{}
	backend    metrics.Backend
	state      model.State
	httpCli    *http.Client
}

func newScaler(l log.Logger, data []byte, b metrics.Backend) *Scaler {
	s := &Scaler{
		logger:     l,
		done:       make(chan struct{}),
		terminated: make(chan struct{}),
		backend:    b,
		httpCli:    common.NewHTTPClient(),
	}
	_ = json.Unmarshal(data, s)
	if s.Alert == nil {
		s.Alert = &model.Alert{}
	}
	s.alert = &alert.Alert{State: *s.Alert}
	s.state = model.StateActive
	return s
}

func (s *Scaler) Stop() {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	// Ignore close channel if scaler is already stopped/stopping
	if s.state == model.StateStopping || s.state == model.StateStopped {
		return
	}
	level.Debug(s.logger).Log("msg", "Scaler is stopping")
	s.state = model.StateStopping
	close(s.done)
	<-s.terminated
	s.state = model.StateStopped
	exporter.ReportNumScalers(cluster.GetID(), -1)
	level.Debug(s.logger).Log("msg", "Scaler is stopped")
}

func (s *Scaler) run(ctx context.Context, wg *sync.WaitGroup) {
	interval, _ := common.ParseDuration(s.Interval)
	duration, _ := common.ParseDuration(s.Duration)
	cooldown, _ := common.ParseDuration(s.Cooldown)
	ticker := time.NewTicker(interval)
	// Report number of scalers
	exporter.ReportNumScalers(cluster.GetID(), 1)
	defer func() {
		ticker.Stop()
		wg.Done()
		close(s.terminated)
	}()

	for {
		select {
		case <-s.done:
			return
		default:
			select {
			case <-s.done:
				return
			case <-ticker.C:
				if !s.Active {
					continue
				}
				result, err := s.backend.QueryInstant(ctx, s.Query, time.Now())
				if err != nil {
					level.Error(s.logger).Log("msg", "Executing query failed, skip current interval",
						"query", s.Query, "err", err)
					s.state = model.StateFailed
					exporter.ReportMetricQueryFailureCounter(cluster.GetID(),
						s.backend.GetType(), s.backend.GetAddress())
					continue
				}
				level.Debug(s.logger).Log("msg", "Executing query success",
					"query", s.Query)
				s.mtx.Lock()
				if len(result) == 0 {
					s.alert.Reset()
					s.mtx.Unlock()
					continue
				}
				if !s.alert.IsActive() {
					s.alert.Start()
				}
				if s.alert.ShouldFire(duration) && !s.alert.IsCoolingDown(cooldown) {
					s.do()
				}
				s.mtx.Unlock()
			}
		}
	}
}

// do simply creates and executes a POST request
func (s *Scaler) do() {
	var wg sync.WaitGroup

	for _, a := range s.Actions {
		switch at := a.(type) {
		case *model.ActionHTTP:
			wg.Add(1)
			go func(a *model.ActionHTTP) {
				defer wg.Done()
				url := a.URL.String()
				// TODO(kiennt): Check kind of action url -> Authen or not?
				if err := alert.SendHTTP(s.httpCli, a); err != nil {
					level.Error(s.logger).Log("msg", "Error doing HTTP action",
						"url", url, "err", err)
					exporter.ReportFailureScalerActionCounter(cluster.GetID(), "http")
					return
				}

				exporter.ReportSuccessScalerActionCounter(cluster.GetID(), "http")
				level.Info(s.logger).Log("msg", "Sending request",
					"url", url, "method", a.Method)
				s.alert.Fire(time.Now())
			}(at)
		}
	}

	// Wait until all actions were performed
	wg.Wait()
}
