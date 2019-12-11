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
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"

	"github.com/vCloud-DFTBA/faythe/pkg/alert"
	"github.com/vCloud-DFTBA/faythe/pkg/metrics"
	"github.com/vCloud-DFTBA/faythe/pkg/model"
	"github.com/vCloud-DFTBA/faythe/pkg/utils"
)

type healerState int

const (
	httpTimeout             = time.Second * 15
	stateNone   healerState = iota
	stateStopping
	stateStopped
	stateFailed
	stateActive
)

func (s healerState) String() string {
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
		return "active"
	default:
		panic(fmt.Sprintf("unknown healer state: %d", s))
	}
}

// Healer scrape metrics from metrics backend priodically
// and evaluate whether it is necessary to do healing action
type Healer struct {
	model.Healer
	at         model.ATEngine
	backend    metrics.Backend
	done       chan struct{}
	logger     log.Logger
	mtx        sync.RWMutex
	state      healerState
	terminated chan struct{}
}

func newHealer(l log.Logger, data []byte, b metrics.Backend, atengine model.ATEngine) *Healer {
	h := &Healer{
		at:         atengine,
		backend:    b,
		done:       make(chan struct{}),
		logger:     l,
		terminated: make(chan struct{}),
	}
	json.Unmarshal(data, h)
	h.Validate()
	h.state = stateActive
	return h
}

func (h *Healer) run(ctx context.Context, wg *sync.WaitGroup, nc chan map[string]string) {
	interval, _ := time.ParseDuration(h.Interval)
	duration, _ := time.ParseDuration(h.Duration)
	ticker := time.NewTicker(interval)
	chans := make(map[string]*chan struct{})
	whitelist := make(map[string]struct{})
	defer func() {
		wg.Done()
		ticker.Stop()
		close(h.terminated)
	}()
	for {
		select {
		case <-h.done:
			return
		default:
			select {
			case <-h.done:
				return
			case <-ticker.C:
				if !h.Active {
					continue
				}
				r, err := h.backend.QueryInstant(ctx, h.Query, time.Now())
				if err != nil {
					level.Error(h.logger).Log("msg", "Executing query failed, skip current interval",
						"query", h.Query, "err", err)
					h.state = stateFailed
					if utils.RetryableError(err) {
						continue
					} else {
						return
					}
				}
				level.Debug(h.logger).Log("msg", "Executing query success", "query", h.Query)

				// Make a dict contains list of distinct result Instances
				rIs := make(map[string]int)
				for _, e := range r {
					instance := strings.Split(string(e.Metric["instance"]), ":")[0]
					rIs[instance]++
				}
				
				for k, v := range rIs {
					if v != h.EvaluationLevel {
						delete(rIs, k)
					}
				}

				// If no of instance = 0, clear all goroutines an whitelist
				if len(rIs) == 0 {
					for k, c := range chans {
						close(*c)
						delete(chans, k)
					}
					for k := range whitelist {
						delete(whitelist, k)
					}
					continue
				}

				// If no of instance > 3, clear all goroutines
				if len(rIs) > 3 {
					level.Info(h.logger).Log("msg", fmt.Sprintf("Not processed because the number of instance needed healing > %d",len(rIs)))
					for k, c := range chans {
						close(*c)
						delete(chans, k)
					}
					continue
				}

				// Remove redundant goroutine if exists
				for k, c := range chans {
					if _, ok := rIs[k]; ok {
						continue
					}
					close(*c)
					delete(chans, k)
				}

				// Clear entry in whitelist if instance goes up again
				for k := range whitelist {
					if _, ok := rIs[k]; ok {
						continue
					}
					delete(whitelist, k)
					level.Info(h.logger).Log("msg", fmt.Sprintf("instance %s goes up again, removed from whitelist", k))
				}

				for instance := range rIs {
					if _, ok := whitelist[instance]; ok {
						continue
					}
					if _, ok := chans[instance]; !ok {
						ci := make(chan struct{})
						chans[instance] = &ci
						go func(ci chan struct{}, instance string, nc chan map[string]string) {
							var compute string
							key := MakeKey(h.CloudID, instance)
						wait:
							//	wait for correct compute-instance pair
							for {
								select {
								case <-ci:
									return
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
							level.Info(h.logger).Log("msg", fmt.Sprintf("Processing instance: %s", instance))
							a := alert.Alert{}
							a.Reset()
							for {
								select {
								case <-ci:
									return
								default:
									if !a.IsActive() {
										a.Start()
									}
									if a.ShouldFire(duration) {
										level.Info(h.logger).Log("msg", fmt.Sprintf("Fired alert for instance: %s", instance))
										h.do(compute)
										// if healing for compute is fired, store it in a whitelist
										whitelist[instance] = struct{}{}
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
}

func (h *Healer) do(compute string) {
	var (
		wg  sync.WaitGroup
		tr  *http.Transport
		cli *http.Client
	)

	tr = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
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
				level.Info(h.logger).Log("msg", "Sending request",
					"url", url, "method", at.Method)
			}(string(at.URL), compute)
		case *model.ActionMail:
			go func(compute string) {
				wg.Add(1)
				defer wg.Done()
				at.Subject = "Node down, triggering autohealing"
				at.Body = fmt.Sprintf("Node %s has been down for more than %s.", compute, h.Duration)
				if err := alert.SendMail(at); err != nil {
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
	h.mtx.Lock()
	defer h.mtx.Unlock()

	if h.state == stateStopping || h.state == stateStopped {
		return
	}
	level.Debug(h.logger).Log("msg", "Healer is stopping")
	h.state = stateStopping
	close(h.done)
	<-h.terminated
	h.state = stateStopped
	level.Debug(h.logger).Log("msg", "Healer is stopped")
}
