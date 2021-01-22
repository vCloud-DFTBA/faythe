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
	cmap "github.com/orcaman/concurrent-map"
	etcdv3 "go.etcd.io/etcd/clientv3"

	"github.com/vCloud-DFTBA/faythe/config"
	"github.com/vCloud-DFTBA/faythe/pkg/alert"
	"github.com/vCloud-DFTBA/faythe/pkg/cloud/store/openstack"
	"github.com/vCloud-DFTBA/faythe/pkg/cluster"
	"github.com/vCloud-DFTBA/faythe/pkg/common"
	"github.com/vCloud-DFTBA/faythe/pkg/exporter"
	"github.com/vCloud-DFTBA/faythe/pkg/metrics"
	"github.com/vCloud-DFTBA/faythe/pkg/metrics/backends/prometheus"
	"github.com/vCloud-DFTBA/faythe/pkg/model"
)

// Healer scrape metric from metrics backend periodically
// and evaluate whether it is necessary to do healing action
type Healer struct {
	model.Healer
	backend    metrics.Backend
	nodes      cmap.ConcurrentMap
	done       chan struct{}
	logger     log.Logger
	mtx        sync.RWMutex
	state      model.State
	terminated chan struct{}
	silences   cmap.ConcurrentMap
	httpCli    *http.Client
}

func newHealer(l log.Logger, data []byte, b metrics.Backend) *Healer {
	h := &Healer{
		backend:    b,
		done:       make(chan struct{}),
		nodes:      cmap.New(),
		logger:     l,
		terminated: make(chan struct{}),
		silences:   cmap.New(),
		httpCli:    common.NewHTTPClient(),
	}
	_ = json.Unmarshal(data, h)
	_ = h.Validate()
	h.state = model.StateActive
	return h
}

func (h *Healer) run(ctx context.Context, e *common.Etcd, nc chan map[string]string) {
	interval, _ := common.ParseDuration(h.Interval)
	sinterval, _ := common.ParseDuration(model.DefaultSilenceValidationInterval)
	duration, _ := common.ParseDuration(h.Duration)
	ticker := time.NewTicker(interval)
	sticker := time.NewTicker(sinterval)
	chans := make(map[string]*chan struct{})
	whitelist := make(map[string]struct{})
	swatch := e.Watch(ctx, common.Path(model.DefaultSilencePrefix, h.CloudID), etcdv3.WithPrefix())
	h.updateSilence(e)
	// Sync silences from Alertmanager
	go h.syncSilencesFromBackend(ctx, e)

	// Record the number of healers
	exporter.ReportNumberOfHealers(cluster.GetID(), 1)
	defer func() {
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
			case node := <-nc:
				h.updateNodes(node)
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
						h.silences.Set(sid, &s)
					} else if event.Type == etcdv3.EventTypeDelete {
						h.silences.Remove(sid)
					}
				}
			case <-ticker.C:
				if !h.Active {
					continue
				}
				r, err := h.backend.QueryInstant(ctx, h.Query, time.Now())
				if err != nil {
					level.Error(h.logger).Log("msg", "Execute query failed, skip current interval",
						"query", h.Query, "err", err)
					h.state = model.StateFailed
					exporter.ReportMetricQueryFailureCounter(cluster.GetID(),
						h.backend.GetType(), h.backend.GetAddress())
					continue
				}
				level.Debug(h.logger).Log("msg", "Execute query success", "query", h.Query)

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
					// Check silenced instances
					for ks, vs := range h.silences.Items() {
						sil := vs.(*model.Silence)
						if matched := sil.RegexPattern.MatchString(k); matched {
							level.Info(h.logger).Log("msg", fmt.Sprintf("instance %s is ignored because of silence: %s", k, ks))
							delete(rIs, k)
						}
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

				for instance := range rIs {
					if _, ok := chans[instance]; !ok {
						ci := make(chan struct{})
						chans[instance] = &ci
						go func(ci chan struct{}, instance string) {
							var compute string
							// Rest your goroutine, prevent CPU spike
							rTicker := time.NewTicker(100 * time.Millisecond)
							for {
								<-rTicker.C
								if com, ok := h.nodes.Get(instance); ok {
									compute = com.(string)
									break
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
								case <-rTicker.C:
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

func (h *Healer) updateNodes(n map[string]string) {
	for ip, hostname := range n {
		h.nodes.Set(strings.Split(ip, ":")[0], hostname)
	}
}

func (h *Healer) do(compute string) {
	var wg sync.WaitGroup
	store := openstack.Get()
	os, ok := store.Get(h.CloudID)
	if !ok {
		level.Error(h.logger).Log("msg",
			fmt.Sprintf("cannot find cloud key %s in store", h.CloudID))
		return
	}
	// Create a email subject prefix
	subject := "[autohealing]"
	if h.Description != "" {
		subject = fmt.Sprintf("%s[%s]", subject, h.Description)
	}
	if len(h.Tags) > 0 {
		for _, t := range h.Tags {
			subject = fmt.Sprintf("%s[%s]", subject, t)
		}
	}
	if h.Description == "" && len(h.Tags) == 0 {
		subject = fmt.Sprintf("%s[%s]", subject, os.ID)
	}

	for _, a := range h.Actions {
		switch at := a.(type) {
		case *model.ActionHTTP:
			wg.Add(1)
			go func(url string, compute string) {
				defer wg.Done()
				var msg []interface{}
				if at.CloudAuthToken {
					// If HTTP uses cloud auth token, let's get it from Cloud base client.
					// Only OpenStack provider is supported at this time.
					baseCli, _ := os.BaseClient()
					if token, ok := baseCli.AuthenticatedHeaders()["X-Auth-Token"]; ok {
						if at.Header == nil {
							at.Header = make(map[string]string)
						}
						at.Header["X-Auth-Token"] = token
					}
				}
				if err := alert.SendHTTP(h.httpCli, at); err != nil {
					msg = common.CnvSliceStrToSliceInf(append([]string{
						"msg", "Execute action failed",
						"err", err.Error()},
						at.InfoLog()...))
					level.Error(h.logger).Log(msg...)
					exporter.ReportFailureHealerActionCounter(cluster.GetID(), "http")
					m := &model.ActionMail{
						Receivers: h.Receivers,
						Subject:   fmt.Sprintf("%s Node %s down, failed to trigger http request", subject, compute),
						Body: fmt.Sprintf("Node %s is down for more than %s.\nBut failed to trigger autohealing, due to %s",
							compute, h.Duration, err.Error()),
					}
					_ = m.Validate()
					if err := alert.SendMail(m); err != nil {
						msg = common.CnvSliceStrToSliceInf(append([]string{
							"msg", "Execute action failed",
							"err", err.Error()},
							at.InfoLog()...))
						level.Error(h.logger).Log(msg...)
						return
					}
					return
				}
				exporter.ReportSuccessHealerActionCounter(cluster.GetID(), "http")

				msg = common.CnvSliceStrToSliceInf(append([]string{
					"msg", "Execute action success"},
					at.InfoLog()...))
				level.Info(h.logger).Log(msg...)
			}(string(at.URL), compute)
		case *model.ActionMail:
			wg.Add(1)
			go func(compute string) {
				defer wg.Done()
				var msg []interface{}
				at.Receivers = h.Receivers
				at.Subject = fmt.Sprintf("%s Node %s down, trigger autohealing", subject, compute)
				at.Body = fmt.Sprintf("Node %s has been down for more than %s.", compute, h.Duration)
				if err := alert.SendMail(at); err != nil {
					msg = common.CnvSliceStrToSliceInf(append([]string{
						"msg", "Execute action failed",
						"err", err.Error()},
						at.InfoLog()...))
					level.Error(h.logger).Log(msg...)
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
				var msg []interface{}
				mc := config.Get().MailConfig
				at.Input = map[string]interface{}{
					"compute":       compute,
					"smtp_server":   fmt.Sprintf("%s:%d", mc.Host, mc.Port),
					"smtp_username": mc.Username,
					"smtp_password": mc.Password,
					"to_addrs":      h.Receivers,
				}
				tracker := NewTracker(log.With(h.logger), *at, os)
				if err := tracker.start(); err != nil {
					level.Error(h.logger).Log("msg", "error doing Mistral action", "err", err)
					exporter.ReportFailureHealerActionCounter(cluster.GetID(), "mistral")
					m := &model.ActionMail{
						Receivers: h.Receivers,
						Subject:   fmt.Sprintf("%s Node %s down, mistral workflow execution failed", subject, compute),
						Body: fmt.Sprintf("Node %s is down for more than %s.\nMistral workflow executions has exceeded maxinum number of retry.",
							compute, h.Duration),
					}
					_ = m.Validate()
					if err := alert.SendMail(m); err != nil {
						msg = common.CnvSliceStrToSliceInf(append([]string{
							"msg", "Error while sending email notifying mistral action failed",
							"err", err.Error()},
							at.InfoLog()...))
						level.Error(h.logger).Log(msg...)
						return
					}
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
	for k, s := range h.silences.Items() {
		sil := s.(*model.Silence)
		if sil.ExpiredAt.Before(now) || sil.ExpiredAt.Equal(now) {
			h.silences.Remove(k)
		}
	}
}

func (h *Healer) updateSilence(e *common.Etcd) {
	resp, err := e.DoGet(common.Path(model.DefaultSilencePrefix, h.CloudID), etcdv3.WithPrefix())
	if err != nil {
		level.Error(h.logger).Log("msg", "error while getting information from etcd", "err", err)
		return
	}
	// Force init silence map
	h.silences = cmap.New()
	for _, v := range resp.Kvs {
		sid := string(v.Key)
		s := model.Silence{}
		_ = json.Unmarshal(v.Value, &s)
		s.RegexPattern, _ = regexp.Compile(s.Pattern)
		h.silences.Set(sid, &s)
	}
}

// syncSilencesFromBackend queries silences from the metric backend. Only Prometheus Alertmanager
// is supported at this time.
func (h *Healer) syncSilencesFromBackend(ctx context.Context, e *common.Etcd) {
	if h.backend.GetType() != model.PrometheusType || !h.SyncSilences {
		level.Debug(h.logger).Log("msg", "Skip sync silences",
			"backend", h.backend.GetAddress())
		return
	}
	level.Debug(h.logger).Log("msg", "Sync silences from metric backend",
		"backend", h.backend.GetAddress())
	syncIntenval, _ := common.ParseDuration(model.DefaultSyncSilencesInterval)
	syncTicker := time.NewTicker(syncIntenval)

	for {
		select {
		case <-h.done:
			return
		case <-syncTicker.C:
			silencesMap, err := h.backend.(*prometheus.Backend).GetAlertManagerSilences(ctx, []string{"instance=~\".+\""})
			if err != nil {
				level.Error(h.logger).Log("msg", "Error retrieving silences from Alertmanager",
					"backend", h.backend.GetAddress())
				continue
			}
			for id, silence := range silencesMap {
				// If silence's comment doesn't
				// - Start with `[faythe]` prefix,
				// - Contain the Healer tags,
				// ignore it!
				// For example: [faythe][openstack-hlct5] Silence due to maintenance
				var ignore bool
				if !strings.HasPrefix(strings.ToLower(*silence.Comment), model.DefaultSyncSilencePrefix) {
					ignore = true
				}
				if len(h.Tags) != 0 {
					for _, t := range h.Tags {
						if !strings.Contains(strings.ToLower(*silence.Comment), "["+t+"]") {
							ignore = true
							break
						}
					}
				}
				if ignore {
					level.Debug(h.logger).Log("msg",
						"Ignoring silence doesn't satisfy the comment's format condition", "id", id)
					continue
				}
				s := &model.Silence{
					ID:          id, // Force use AM silence's ID
					Name:        model.DefaultSyncedSilenceName,
					CreatedBy:   *silence.CreatedBy,
					CreatedAt:   time.Time(*silence.StartsAt),
					ExpiredAt:   time.Time(*silence.EndsAt),
					Tags:        []string{"auto-sync", "alertmanager"},
					Description: *silence.Comment,
				}
				for _, matcher := range silence.Matchers {
					// Only care about matcher with name 'instance'
					if *matcher.Name == "instance" {
						s.Pattern = *matcher.Value
						break
					}
				}
				if s.Pattern == "" {
					continue
				}
				if err := s.Validate(); err != nil {
					level.Error(h.logger).Log("msg", "Error when validating silence",
						"id", s.ID, "err", err)
					continue
				}
				// Check if the silence exists
				path := common.Path(model.DefaultSilencePrefix, h.CloudID, s.ID)
				if existSilItf, ok := h.silences.Get(s.ID); ok {
					existSil := existSilItf.(*model.Silence)
					if existSil.ExpiredAt != s.ExpiredAt || existSil.Pattern != s.Pattern {
						level.Debug(h.logger).Log("msg", "Delete the outdated synced silence", "id", s.ID)
						_, err := e.DoDelete(path, etcdv3.WithPrefix())
						if err != nil {
							level.Error(h.logger).Log("msg", "Error when deleting the outdated synced silence",
								"id", s.ID, "err", err)
							continue
						}
						// Force calculate silence's TTL
						s.CreatedAt = time.Now()
						_ = s.Validate()
						h.silences.Remove(s.ID)
					}
				}

				t, _ := common.ParseDuration(s.TTL)
				grantr, err := e.DoGrant(int64(t.Seconds()))
				if err != nil {
					level.Error(h.logger).Log("msg", "Error when getting grant for silence",
						"id", s.ID, "err", err)
					continue
				}
				raw, _ := json.Marshal(&s)
				if _, err := e.DoPut(path, string(raw), etcdv3.WithLease(grantr.ID)); err != nil {
					level.Error(h.logger).Log("msg", "Error when creating silence",
						"id", s.ID, "err", err)
					continue
				}
				h.silences.Set(s.ID, s)
				level.Info(h.logger).Log("msg", "Create a silence successfully", "id", s.ID)
			}

			for id, silence := range h.silences.Items() {
				sil := silence.(*model.Silence)
				// If there is a synced silence exist on Faythe but is no longer
				// available on Alertmanager, delete it.
				if sil.Name != model.DefaultSyncedSilenceName {
					continue
				}
				if _, ok := silencesMap[id]; !ok {
					level.Debug(h.logger).Log("msg", "Delete the outdated synced silence", "id", id)
					path := common.Path(model.DefaultSilencePrefix, h.CloudID, id)
					_, err := e.DoDelete(path, etcdv3.WithPrefix())
					if err != nil {
						level.Error(h.logger).Log("msg", "Error when deleting the outdated synced silence",
							"id", id, "err", err)
						continue
					}
					h.silences.Remove(id)
				}
			}
		}
	}
}
