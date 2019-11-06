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
        "crypto/tls"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/vCloud-DFTBA/faythe/pkg/alert"
	"github.com/vCloud-DFTBA/faythe/pkg/metrics"
	"github.com/vCloud-DFTBA/faythe/pkg/model"
)

const (
	httpTimeout = time.Second * 15
)

// Healer scrape metrics from metrics backend priodically
// and evaluate whether it is necessary to do healing action
type Healer struct {
	model.Healer
	logger  log.Logger
	done    chan struct{}
	backend metrics.Backend
	at      model.ATEngine
	wl      map[string]struct{}
}

func newHealer(l log.Logger, data []byte, b metrics.Backend, atengine model.ATEngine) *Healer {
	h := &Healer{
		logger:  l,
		done:    make(chan struct{}),
		backend: b,
		at:      atengine,
		wl:      make(map[string]struct{}),
	}
	json.Unmarshal(data, h)
	h.Validate()
	return h
}

func (h *Healer) run(ctx context.Context, wg *sync.WaitGroup, nc chan map[string]string) {
	interval, _ := time.ParseDuration(h.Interval)
	duration, _ := time.ParseDuration(h.Duration)
	ticker := time.NewTicker(interval)
	chans := make(map[string]chan struct{})
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
			if len(r) == 0 || len(r) > 3 {
				for _, c := range chans {
					close(c)
				}
				continue
			}

			// Make a dict contains list of instances
			ris := make(map[string]struct{})
			for _, e := range r {
				instance := strings.Split(string(e.Metric["instance"]), ":")[0]
				if _, ok := ris[instance]; !ok {
					ris[instance] = struct{}{}
				}
			}

			// Remove redundant goroutine if exists
			for k, c := range chans {
				if _, ok := ris[k]; ok {
					continue
				}
				close(c)
				delete(chans, k)
			}

			// Clear entry in whitelist if instance goes up again
			for k := range h.wl {
				if _, ok := ris[k]; ok {
					continue
				}
				delete(h.wl, k)
			}

			for _, e := range r {
				instance := strings.Split(string(e.Metric["instance"]), ":")[0]
				if _, ok := h.wl[instance]; ok {
					continue
				}
				if _, ok := chans[instance]; !ok {
					ci := make(chan struct{})
					chans[instance] = ci
					go func(ch chan struct{}, instance string, nc chan map[string]string) {
						var compute string
						key := MakeKey(h.CloudID, instance)
					wait:
						//	wait for correct compute-instance pair
						for {
							select {
							case c := <-nc:
								if com, ok := c[key]; ok {
									compute = com
									break wait
								}
								continue
							default:
								nc <- map[string]string{"instance": key}
								continue
							}
						}
						a := alert.Alert{}
						a.Reset()
						for {
							select {
							case <-ch:
								return
							default:
								if !a.IsActive() {
									a.Start()
								}
								if a.ShouldFire(duration) {
									h.do(compute)
									// if healing for compute is fired, store it in a whitelist
									h.wl[instance] = struct{}{}
									delete(chans, instance)
									return
								}

							}
						}
					}(ci, instance, nc)
				}
			}
		}
	}
}

func (h *Healer) do(compute string) {
	var (
		wg  sync.WaitGroup
		tr  *http.Transport
		cli *http.Client
	)

	tr = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true},}
	cli = &http.Client{
		Transport: tr,
		Timeout:   httpTimeout,
	}

	for _, a := range h.Actions {
		switch at := a.(type) {
		case *model.ActionHTTP:
			go func(url string, compute string) {
				wg.Add(1)
				defer wg.Done()
				params := make(map[string]map[string]string)
				switch h.at.Backend {
				case "stackstorm":
					params["header"] = make(map[string]string)
					params["body"] = make(map[string]string)
					if apikey := h.at.APIKey; apikey != "" {
						params["header"]["apikey"] = string(apikey)
					} else if username := h.at.Username; username != "" {
						params["header"]["username"] = h.at.Username
						params["header"]["password"] = string(h.at.Password)
					}
					params["body"]["compute"] = compute
				}
				if err := alert.SendHTTP(h.logger, cli, at, params); err != nil {
					level.Error(h.logger).Log("msg", "Error doing HTTP action",
						"url", at.URL.String(), "err", err)
					return
				}
				level.Info(h.logger).Log("msg", "Sending request", "id", h.ID,
					"url", url, "method", at.Method)
			}(string(at.URL), compute)
		case *model.ActionMail:
			go func(compute string) {
				wg.Add(1)
				defer wg.Done()
				if err := alert.SendMail(at, compute, h.Duration); err != nil {
					level.Error(h.logger).Log("msg", "Error doing Mail action",
						"err", err)
					return
				}
				level.Info(h.logger).Log("msg", "Sending mail to", "receivers", strings.Join(at.Receivers, ","))
			}(compute)
		default:
		}
	}
	wg.Wait()
}

// Stop Healer worker
func (h *Healer) Stop() {
	level.Debug(h.logger).Log("msg", "Healer is stopping", "id", h.ID)
	close(h.done)
	level.Debug(h.logger).Log("msg", "Healer is stopped", "id", h.ID)
}
