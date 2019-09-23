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
	"github.com/pkg/errors"

	"github.com/ntk148v/faythe/pkg/metrics"
	"github.com/ntk148v/faythe/pkg/model"
)

// Scaler does metric polling and executes scale actions.
type Scaler struct {
	model.Scaler
	backend metrics.Backend
	alert   *alert
	logger  log.Logger
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

func (s *Scaler) doScale() {
	var (
		wg  sync.WaitGroup
		cli http.Client
	)
	defer wg.Done()
	cli = http.Client{Timeout: time.Second * 15}
	for _, a := range s.Actions {
		wg.Add(1)
		go func(url string) {
			req, err := http.NewRequest("POST", url, nil)
			if err != nil {
				level.Info(s.logger).Log("msg", "Error creating scale request",
					"req", req.URL, "err", err)
			}
			resp, err := cli.Do(req)
			if err != nil {
				level.Info(s.logger).Log("msg", "Error sends scale request",
					"req", req.URL, "err", err)
			}
			defer resp.Body.Close()
		}(string(a))
	}
}

func newScaler(l log.Logger, b metrics.Backend, data []byte) *Scaler {
	s := &Scaler{
		backend: b,
		logger:  l,
	}
	_ = json.Unmarshal(data, s)
	return s
}

func (s *Scaler) run(wg *sync.WaitGroup, stopChan <-chan struct{}) {
	defer wg.Done()

	interval, _ := time.ParseDuration(s.Interval)
	duration, _ := time.ParseDuration(s.Duration)
	ticker := time.NewTicker(interval)
	s.alert = &alert{}
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if !s.Active {
				continue
			}
			result, err := s.backend.QueryInstant(context.Background(), s.Query, time.Now())
			if err != nil {
				err = errors.Wrapf(err, "querying %s", s.Query)
				return
			}

			level.Debug(s.logger).Log("msg", "Scaler is querying successfully", "id", s.ID,
				"query", s.Query, "result", result.String())
			if len(result) == 0 {
				continue
			}
			s.alert.start()
			if s.alert.shouldFire(duration) {
				s.doScale()
			}
		case <-stopChan:
			level.Debug(s.logger).Log("msg", "Scaler is shutting down", "id", s.ID)
			return
		}
	}
}
