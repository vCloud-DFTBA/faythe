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
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	etcdv3 "go.etcd.io/etcd/clientv3"

	"github.com/vCloud-DFTBA/faythe/config"
	"github.com/vCloud-DFTBA/faythe/pkg/alert"
	"github.com/vCloud-DFTBA/faythe/pkg/cloud/store/openstack"
	"github.com/vCloud-DFTBA/faythe/pkg/cluster"
	"github.com/vCloud-DFTBA/faythe/pkg/common"
	"github.com/vCloud-DFTBA/faythe/pkg/exporter"
	"github.com/vCloud-DFTBA/faythe/pkg/metrics"
	"github.com/vCloud-DFTBA/faythe/pkg/model"
)

// Healer scrape metric from metrics backend periodically
// and evaluate whether it is necessary to do healing action
type Healer struct {
	model.Healer
	backend    metrics.Backend
	done       chan struct{}
	logger     log.Logger
	mtx        sync.RWMutex
	state      model.State
	terminated chan struct{}
	silences   map[string]*model.Silence
	httpCli    *http.Client
}

func newHealer(l log.Logger, data []byte, b metrics.Backend) *Healer {
	h := &Healer{
		backend:    b,
		done:       make(chan struct{}),
		logger:     l,
		terminated: make(chan struct{}),
		silences:   make(map[string]*model.Silence),
		httpCli:    common.NewHTTPClient(),
	}
	_ = json.Unmarshal(data, h)
	_ = h.Validate()
	h.state = model.StateActive
	return h
}

func (h *Healer) run(ctx context.Context, e *common.Etcd, wg *sync.WaitGroup, nc chan map[string]string) {
	interval, _ := common.ParseDuration(h.Interval)
	sinterval, _ := common.ParseDuration(model.DefaultSilenceValidationInterval)
	duration, _ := common.ParseDuration(h.Duration)
	ticker := time.NewTicker(interval)
	sticker := time.NewTicker(sinterval)
	chans := make(map[string]*chan struct{})
	whitelist := make(map[string]struct{})
	swatch := e.Watch(ctx, common.Path(model.DefaultSilencePrefix, h.CloudID), etcdv3.WithPrefix())
	h.updateSilence(e)

	// Record the number of healers
	exporter.ReportNumberOfHealers(cluster.GetID(), 1)
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
			case <-sticker.C:
				h.validateSilence()
			case watchResp := <-swatch:
				if watchResp.Err() != nil {
					level.Error(h.logger).Log("msg", "error watching etcd events", "err", watchResp.Err())
					continue
				}
				for _, event := range watchResp.Events {
					sid := string(event.Kv.Key)
					if event.IsCreate() {
						s := model.Silence{}
						_ = json.Unmarshal(event.Kv.Value, &s)
						s.RegexPattern, _ = regexp.Compile(s.Pattern)
						h.silences[sid] = &s
					} else if event.Type == etcdv3.EventTypeDelete {
						delete(h.silences, sid)
					}
				}
			case <-ticker.C:
				if !h.Active {
					continue
				}
				r, err := h.backend.QueryInstant(ctx, h.Query, time.Now())
				if err != nil {
					level.Error(h.logger).Log("msg", "Executing query failed, skip current interval",
						"query", h.Query, "err", err)
					h.state = model.StateFailed
					exporter.ReportMetricQueryFailureCounter(cluster.GetID(),
						h.backend.GetType(), h.backend.GetAddress())
					continue
				}
				level.Debug(h.logger).Log("msg", "Executing query success", "query", h.Query)

				// Make a dict contains list of distinct result Instances
				rIs := make(map[string]int)
				for _, e := range r {
					instance := strings.Split(string(e.Metric["instance"]), ":")[0]
					rIs[instance]++
				}

				// Clear entry in whitelist if instance goes up again
				for k := range whitelist {
					if _, ok := rIs[k]; ok {
						continue
					}
					delete(whitelist, k)
					level.Info(h.logger).Log("msg", fmt.Sprintf("instance %s goes up again, removed from whitelist", k))
				}

				// If number of instance = 0, clear all goroutines
				if len(rIs) == 0 {
					for k, c := range chans {
						close(*c)
						delete(chans, k)
					}
					continue
				}

				// Update existing goroutines
				for k, c := range chans {
					if _, ok := rIs[k]; ok {
						continue
					}
					close(*c)
					delete(chans, k)
				}

				// If number of metrics returned for a instance != EvaluationLevel
				// Or if instances in whitelist, delete from list of Instances, not process it
				// If instance is processing then delete from list of instances
				for k, v := range rIs {
					if _, ok := whitelist[k]; ok || v != h.EvaluationLevel {
						delete(rIs, k)
					}
					if _, ok := chans[k]; ok {
						delete(rIs, k)
					}
				}

				// If number of instances > DefaultMaxNumberOfInstances, clear all goroutines
				// Or number of instances + number of existing instances need to heal > DefaultMaxNumberOfInstances
				if len(rIs) > model.DefaultMaxNumberOfInstances || len(rIs)+len(chans) > model.DefaultMaxNumberOfInstances {
					level.Info(h.logger).Log("msg",
						fmt.Sprintf("not processed because the number of instance needed healing = %d > %d",
							len(rIs), model.DefaultMaxNumberOfInstances))
					for k, c := range chans {
						close(*c)
						delete(chans, k)
					}
					continue
				}

			processing:
				for instance := range rIs {
					for k, v := range h.silences {
						if matched := v.RegexPattern.MatchString(instance); matched {
							level.Info(h.logger).Log("msg", fmt.Sprintf("instance %s is ignored because of silence: %s", instance, k))
							continue processing
						}
					}
					if _, ok := chans[instance]; !ok {
						ci := make(chan struct{})
						chans[instance] = &ci
						go func(ci chan struct{}, instance string) {
							var compute string
							key := common.Path(h.CloudID, instance)
						wait:
							//	wait for correct compute-instance pair
							for {
								select {
								case <-h.done:
									return
								case <-ci:
									return
								case c := <-nc:
									if com, ok := c[key]; ok && com != "" {
										compute = com
										break wait
									}
								default:
									nc <- map[string]string{"instance": key}
								}
							}
							level.Info(h.logger).Log("msg", fmt.Sprintf("Processing instance: %s", instance))
							a := alert.Alert{}
							a.Reset()
							for {
								select {
								case <-h.done:
									return
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
						}(ci, instance)
					}
				}
			}
		}
	}
}

func (h *Healer) do(compute string) {
	var wg sync.WaitGroup
	for _, a := range h.Actions {
		switch at := a.(type) {
		case *model.ActionHTTP:
			wg.Add(1)
			go func(url string, compute string) {
				defer wg.Done()
				params := make(map[string]map[string]string)
				if err := alert.SendHTTP(h.logger, h.httpCli, at, params); err != nil {
					level.Error(h.logger).Log("msg", "Error doing HTTP action",
						"url", at.URL.String(), "err", err)
					exporter.ReportFailureHealerActionCounter(cluster.GetID(), "http")
					m := &model.ActionMail{
						Receivers: h.Receivers,
						Subject:   fmt.Sprintf("[autohealing] Node %s down, failed to trigger http request", compute),
						Body: fmt.Sprintf("Node %s is down for more than %s.\nBut failed to trigger autohealing, due to %s",
							compute, h.Duration, err.Error()),
					}
					_ = m.Validate()
					if err := alert.SendMail(m); err != nil {
						level.Error(h.logger).Log("msg", "error doing Mail action",
							"err", err)
						return
					}
					return
				}
				exporter.ReportSuccessHealerActionCounter(cluster.GetID(), "http")
				level.Info(h.logger).Log("msg", "Sending request",
					"url", url, "method", at.Method)
			}(string(at.URL), compute)
		case *model.ActionMail:
			wg.Add(1)
			go func(compute string) {
				defer wg.Done()
				at.Receivers = h.Receivers
				at.Subject = fmt.Sprintf("[autohealing] Node %s down, trigger autohealing", compute)
				at.Body = fmt.Sprintf("Node %s has been down for more than %s.", compute, h.Duration)
				if err := alert.SendMail(at); err != nil {
					level.Error(h.logger).Log("msg", "error doing Mail action",
						"err", err)
					exporter.ReportFailureHealerActionCounter(cluster.GetID(), "mail")
					return
				}
				exporter.ReportSuccessHealerActionCounter(cluster.GetID(), "mail")
				level.Info(h.logger).Log("msg", "Sending mail to", "receivers", strings.Join(at.Receivers, ","))
			}(compute)
		case *model.ActionMistral:
			wg.Add(1)
			go func(compute string) {
				defer wg.Done()
				mc := config.Get().MailConfig
				at.Input = map[string]interface{}{
					"compute":       compute,
					"smtp_server":   fmt.Sprintf("%s:%d", mc.Host, mc.Port),
					"smtp_username": mc.Username,
					"smtp_password": mc.Password,
					"to_addrs":      h.Receivers,
				}
				store := openstack.Get()
				os, ok := store.Get(h.CloudID)
				if !ok {
					level.Error(h.logger).Log("msg",
						fmt.Sprintf("cannot find cloud key %s in store", h.CloudID))
					exporter.ReportFailureHealerActionCounter(cluster.GetID(), "mistral")
					return
				}
				tracker := NewTracker(log.With(h.logger), *at, os)
				if err := tracker.start(); err != nil {
					level.Error(h.logger).Log("msg", "error doing Mistral action", "err", err)
					exporter.ReportFailureHealerActionCounter(cluster.GetID(), "mistral")
					return
				}
				exporter.ReportSuccessHealerActionCounter(cluster.GetID(), "mistral")
				level.Info(h.logger).Log("msg", "Workflow execution succeeded",
					"workflow", at.WorkflowID, "compute", compute)
			}(compute)
		}
	}
	wg.Wait()
}

// Stop Healer worker
func (h *Healer) Stop() {
	h.mtx.Lock()
	defer h.mtx.Unlock()

	if h.state == model.StateStopping || h.state == model.StateStopped {
		return
	}
	level.Debug(h.logger).Log("msg", "Healer is stopping")
	h.state = model.StateStopping
	close(h.done)
	<-h.terminated
	h.state = model.StateStopped
	// Record the number of healers
	exporter.ReportNumberOfHealers(cluster.GetID(), -1)
	level.Debug(h.logger).Log("msg", "Healer is stopped")
}

func (h *Healer) validateSilence() {
	now := time.Now()
	for k, s := range h.silences {
		if s.ExpiredAt.Before(now) || s.ExpiredAt.Equal(now) {
			delete(h.silences, k)
		}
	}
}

func (h *Healer) updateSilence(e *common.Etcd) {
	resp, err := e.DoGet(common.Path(model.DefaultSilencePrefix, h.CloudID), etcdv3.WithPrefix())
	if err != nil {
		level.Error(h.logger).Log("msg", "error while getting information from etcd", "err", err)
		return
	}
	// Best way to clear a map https://github.com/golang/go/blob/master/doc/go1.11.html#L447
	for k := range h.silences {
		delete(h.silences, k)
	}
	for _, v := range resp.Kvs {
		sid := string(v.Key)
		s := model.Silence{}
		_ = json.Unmarshal(v.Value, &s)
		s.RegexPattern, _ = regexp.Compile(s.Pattern)
		h.silences[sid] = &s
	}
}
