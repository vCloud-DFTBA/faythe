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

package cluster

import (
	"sync"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/hashicorp/serf/serf"
	"github.com/serialx/hashring"
)

// EventHandler is a handler that does things when events happen.
type EventHandler interface {
	HandleEvent(e serf.Event)
}

// ReloadHandler is a handler that reload Faythe component's.
type ReloadHandler struct {
	logger     log.Logger
	reloadCh   chan bool
	lock       sync.Mutex
	consistent *hashring.HashRing
}

// HandleEvent triggers component reloads when event is coming.
func (h *ReloadHandler) HandleEvent(e serf.Event) {
	me := e.(serf.MemberEvent)
	h.lock.Lock()
	defer h.lock.Unlock()
	ms := make([]string, len(me.Members))
	for _, i := range me.Members {
		if i.Status != serf.StatusAlive {
			continue
		}
		// NOTE(kiennt): Use Name or may be use Addr:Port?
		ms = append(ms, i.Name)
	}
	switch e.EventType() {
	case serf.EventMemberJoin:
		if h.consistent == nil {
			h.consistent = hashring.New(ms)
		} else {
			for _, m := range ms {
				// TODO(kiennt): Use AddWeightNode (calculate weight based on node resource)
				h.consistent.AddNode(m)
				level.Info(h.logger).Log("msg", "add node to consistent hash ring", "node", m)
			}
		}
	case serf.EventMemberLeave, serf.EventMemberFailed:
		if h.consistent != nil {
			for _, m := range ms {
				h.consistent.RemoveNode(m)
				level.Info(h.logger).Log("msg", "remove node to consistent hash ring", "node", m)
			}
		}
	case serf.EventMemberUpdate, serf.EventMemberReap:
	default:
		level.Error(h.logger).Log("msg", "unknown event", "event", e.String())
		return
	}
	h.reloadCh <- true
}
