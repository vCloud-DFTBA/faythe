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
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/avast/retry-go"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"go.etcd.io/etcd/clientv3/concurrency"

	"github.com/vCloud-DFTBA/faythe/pkg/alert"
	
	"github.com/vCloud-DFTBA/faythe/pkg/common"
	"github.com/vCloud-DFTBA/faythe/pkg/cluster"
	"github.com/vCloud-DFTBA/faythe/pkg/exporter"
	"github.com/vCloud-DFTBA/faythe/pkg/metrics"
	"github.com/vCloud-DFTBA/faythe/pkg/model"
)

// scalerState is the state that a scaler is in.
type scalerState int

const (
	httpTimeout             = time.Second * 15
	stateNone   scalerState = iota
	stateStopping
	stateStopped
	stateFailed
	stateActive
)

func (s scalerState) String() string {
	switch s {
	case stateNone:
		return "none"
	case stateStopping:
		return "stopping"
	case stateStopped:
		return "stopped"
	case stateFailed:
		return "failed"
	case stateActive:
		return "acitve"
	default:
		panic(fmt.Sprintf("unknown scaler state: %d", s))
	}
}

// Scaler does metric polling and executes scale actions.
type Scaler struct {
	model.Scaler
	alert      *alert.Alert
	logger     log.Logger
	mtx        sync.RWMutex
	done       chan struct{}
	terminated chan struct{}
	backend    metrics.Backend
	dlock      concurrency.Mutex
	state      scalerState
}

func newScaler(l log.Logger, data []byte, b metrics.Backend) *Scaler {
	s := &Scaler{
		logger:     l,
		done:       make(chan struct{}),
		terminated: make(chan struct{}),
		backend:    b,
	}
	_ = json.Unmarshal(data, s)
	if s.Alert == nil {
		s.Alert = &model.Alert{}
	}
	s.alert = &alert.Alert{State: *s.Alert}
	s.state = stateActive
	return s
}

func (s *Scaler) Stop() {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	// Ignore close channel if scaler is already stopped/stopping
	if s.state == stateStopping || s.state == stateStopped {
		return
	}
	level.Debug(s.logger).Log("msg", "Scaler is stopping")
	s.state = stateStopping
	close(s.done)
	<-s.terminated
	s.state = stateStopped
	level.Debug(s.logger).Log("msg", "Scaler is stopped")
}

func (s *Scaler) run(ctx context.Context, wg *sync.WaitGroup) {
	interval, _ := time.ParseDuration(s.Interval)
	duration, _ := time.ParseDuration(s.Duration)
	cooldown, _ := time.ParseDuration(s.Cooldown)
	ticker := time.NewTicker(interval)
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
					s.state = stateFailed
					exporter.ReportMetricQueryFailureCounter(cluster.ClusterID,
						s.backend.GetType(), s.backend.GetAddress())
					if common.RetryableError(err) {
						continue
					} else {
						return
					}
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
	var (
		wg  sync.WaitGroup
		tr  *http.Transport
		cli *http.Client
	)
	tr = &http.Transport{}
	cli = &http.Client{
		Transport: tr,
		Timeout:   httpTimeout,
	}

	for _, a := range s.Actions {
		go func(a *model.ActionHTTP) {
			wg.Add(1)
			delay, _ := time.ParseDuration(a.Delay)
			url := a.URL.String()
			err := retry.Do(
				func() error {
					// TODO(kiennt): Check kind of action url -> Authen or not?
					req, err := http.NewRequest(a.Method, url, nil)
					if err != nil {
						return err
					}
					resp, err := cli.Do(req)
					if err != nil {
						return err
					}
					defer resp.Body.Close()
					return nil
				},
				retry.DelayType(func(n uint, config *retry.Config) time.Duration {
					var f retry.DelayTypeFunc
					switch a.DelayType {
					case "fixed":
						f = retry.FixedDelay
					case "backoff":
						f = retry.BackOffDelay
					}
					return f(n, config)
				}),
				retry.Attempts(a.Attempts),
				retry.Delay(delay),
				retry.RetryIf(func(err error) bool {
					return common.RetryableError(err)
				}),
			)
			if err != nil {
				level.Error(s.logger).Log("msg", "Error doing scale action", "url", url, "err", err)
				exporter.ReportFailureScalerActionCounter(cluster.ClusterID, "http")
				return
			}
			level.Info(s.logger).Log("msg", "Sending request",
				"url", url, "method", a.Method)
			s.alert.Fire(time.Now())
			defer wg.Done()
		}(a)
	}
	// Wait until all actions were performed
	wg.Wait()
}
