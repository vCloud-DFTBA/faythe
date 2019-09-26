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

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"

	"github.com/ntk148v/faythe/pkg/metrics"
	"github.com/ntk148v/faythe/pkg/model"
)

const (
	httpTimeout = time.Second * 15
)

// Scaler does metric polling and executes scale actions.
type Scaler struct {
	model.Scaler
	alert      *alert
	logger     log.Logger
	mtx        sync.RWMutex
	done       chan struct{}
	terminated chan struct{}
}

func newScaler(l log.Logger, data []byte) *Scaler {
	s := &Scaler{
		logger:     l,
		done:       make(chan struct{}),
		terminated: make(chan struct{}),
		alert:      &alert{},
	}
	_ = json.Unmarshal(data, s)
	return s
}

func (s *Scaler) stop() {
	close(s.done)
	<-s.terminated
}

func (s *Scaler) run(ctx context.Context, wg *sync.WaitGroup) {
	defer func() {
		wg.Done()
		close(s.terminated)
	}()
	interval, _ := time.ParseDuration(s.Interval)
	duration, _ := time.ParseDuration(s.Duration)
	wg.Add(1)
	ticker := time.NewTicker(interval)
	// Force register
	err := metrics.Register(s.Monitor.Backend, string(s.Monitor.Address))
	if err != nil {
		level.Error(s.logger).Log("msg", "Error loading metric backend, cancel scaler")
		return
	}
	backend, _ := metrics.Get(fmt.Sprintf("%s-%s", s.Monitor.Backend, s.Monitor.Address))

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
				result, err := backend.QueryInstant(ctx, s.Query, time.Now())
				if err != nil {
					level.Error(s.logger).Log("msg", "Executing query failed, skip current interval",
						"query", s.Query, "err", err)
					continue
				}
				level.Debug(s.logger).Log("msg", "Executing query success",
					"query", s.Query)
				s.mtx.Lock()
				if len(result) == 0 {
					s.alert.reset()
					continue
				}
				if !s.alert.isActive() {
					s.alert.start()
				}
				if s.alert.shouldFire(duration) {
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
		go func(url string) {
			// TODO(kiennt): Check kind of action url -> Authen or not?
			req, err := http.NewRequest("POST", url, nil)
			if err != nil {
				level.Error(s.logger).Log("msg", "Error creating scale request",
					"req", req.URL, "err", err)
			}
			resp, err := cli.Do(req)
			if err != nil {
				level.Error(s.logger).Log("msg", "Error performing scale request",
					"req", req.URL, "err", err)
			}
			defer func() {
				resp.Body.Close()
				wg.Done()
			}()
		}(string(a))
	}
	// Wait until all actions were performed
	wg.Wait()
}

type alert struct {
	model.Alert
}

func (a *alert) shouldFire(duration time.Duration) bool {
	return a.Active && time.Now().Sub(a.StartedAt) >= duration
}

func (a *alert) start() {
	a.StartedAt = time.Now()
	a.Active = true
}

func (a *alert) reset() {
	a.StartedAt = time.Time{}
	a.Active = false
}

func (a *alert) isActive() bool {
	return a.Active
}
