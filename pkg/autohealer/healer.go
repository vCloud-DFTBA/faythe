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
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/ntk148v/faythe/pkg/alert"
	"github.com/ntk148v/faythe/pkg/metrics"
	"github.com/ntk148v/faythe/pkg/model"
)

const (
	httpTimeout = time.Second * 15
)

type Healer struct {
	model.Healer
	alert   alert.Alert
	logger  log.Logger
	mtx     sync.RWMutex
	done    chan struct{}
	backend metrics.Backend
}

func newHealer(l log.Logger, data []byte, b metrics.Backend) *Healer {
	h := &Healer{
		logger:  l,
		done:    make(chan struct{}),
		backend: b,
	}
	json.Unmarshal(data, h)
	h.Validate()
	return h
}

func (h *Healer) run(ctx context.Context, wg *sync.WaitGroup, nc *chan string) {
	interval, _ := time.ParseDuration(h.Interval)
	cooldown, _ := time.ParseDuration(h.Cooldown)
	duration, _ := time.ParseDuration(h.Duration)
	ticker := time.NewTicker(interval)
	defer func() {
		wg.Done()
		ticker.Stop()
	}()
	for {
		select {
		case <-h.done:
			return
		case <-ticker.C:
			if !h.Active {
				continue
			}
			r, err := h.backend.QueryInstant(ctx, model.DefaultHealerQuery, time.Now())
			if err != nil {
				level.Error(h.logger).Log("msg", "Executing query failed, skip current interval",
					"query", model.DefaultHealerQuery, "err", err)
				continue
			}
			level.Debug(h.logger).Log("msg", "Executing query success", "query", model.DefaultHealerQuery)
			h.mtx.Lock()
			if len(r) == 0 {
				h.alert.Reset()
				continue
			}
			if !h.alert.IsActive() {
				h.alert.Start()
			}
			if h.alert.ShouldFire(duration) && !h.alert.IsCoolingDown(cooldown) {
				h.do()
			}
			h.mtx.Unlock()
		}
	}
}

func (h *Healer) do() {
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

	h.alert.Fire(time.Now())
	for _, a := range h.Actions {
		switch at := a.(type) {
		case *model.ActionHTTP:
			go func(url string) {
				wg.Add(1)
				defer wg.Done()
				if err := alert.SendHTTP(h.logger, cli, at); err != nil {
					level.Error(h.logger).Log("msg", "Error doing HTTP action",
						"url", at.URL.String(), "err", err)
					return
				}
				level.Info(h.logger).Log("msg", "Sending request", "id", h.ID,
					"url", url, "method", at.Method)
			}(string(at.URL))
		case *model.ActionMail:
			go func() {
				wg.Add(1)
				defer wg.Done()
				if err := alert.SendMail(at); err != nil {
					level.Error(h.logger).Log("msg", "Error doing Mail action",
						"err", err)
					return
				}
				level.Info(h.logger).Log("msg", "Sending mail to", strings.Join(at.Receivers, ","),
					"id", h.ID)
			}()
		default:
		}
	}
	wg.Wait()
}

func (h *Healer) Stop() {
	level.Debug(h.logger).Log("msg", "Healer is stopping", "id", h.ID)
	close(h.done)
	level.Debug(h.logger).Log("msg", "Healer is stopped", "id", h.ID)
}
