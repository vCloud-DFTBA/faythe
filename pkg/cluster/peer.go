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
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/hashicorp/memberlist"
	"github.com/hashicorp/serf/serf"
	"github.com/pkg/errors"

	"github.com/ntk148v/faythe/config"
	"github.com/ntk148v/faythe/pkg/utils"
)

// Peer starts and manage a Serf instance
type Peer struct {
	// Stores the serf configuration
	conf *serf.Config
	// This is underlying Serf we are wrapping
	serf *serf.Serf

	// eventCh is used for Serf to deliver events on
	eventCh chan serf.Event
	// eventHandlers is the registered handlers for events
	eventHandlers     map[EventHandler]struct{}
	eventHandlerList  []EventHandler
	eventHandlersLock sync.Mutex

	logger log.Logger

	// shutdownCh is used for shutdowns
	shutdown     bool
	shutdownCh   chan struct{}
	shutdownLock sync.Mutex
}

// Create creates a new peer, potentially returning an error
func Create(c config.PeerConfig, l log.Logger, lo io.Writer) *Peer {
	if l == nil {
		l = log.NewNopLogger()
	}
	if lo == nil {
		lo = os.Stderr
	}

	// Setup the underlying loggers
	sc := serf.DefaultConfig()
	sc.LogOutput = lo
	switch c.Profile {
	case "lan":
		sc.MemberlistConfig = memberlist.DefaultLANConfig()
	case "wan":
		sc.MemberlistConfig = memberlist.DefaultWANConfig()
	case "local":
		sc.MemberlistConfig = memberlist.DefaultLocalConfig()
	}
	sc.MemberlistConfig.LogOutput = lo
	bindIP, bindPort, _ := utils.AddParts(c.BindAddr)
	sc.MemberlistConfig.BindAddr = bindIP
	sc.MemberlistConfig.BindPort = bindPort
	if c.AdvertiseAddr != "" {
		advertiseIP, advertisePort, _ := utils.AddParts(c.AdvertiseAddr)
		sc.MemberlistConfig.AdvertiseAddr = advertiseIP
		sc.MemberlistConfig.AdvertisePort = advertisePort
	}
	sc.Tags = c.Tags
	if c.ReconnectInterval != 0 {
		sc.ReconnectInterval = c.ReconnectInterval
	}
	if c.ReconnectInterval != 0 {
		sc.ReconnectTimeout = c.ReconnectTimeout
	}
	if c.BroadcastTimeout != 0 {
		sc.BroadcastTimeout = c.BroadcastTimeout
	}

	// Create a channel to listen for events from Serf
	eventCh := make(chan serf.Event, 64)
	sc.EventCh = eventCh

	// Setup the peer
	p := &Peer{
		conf:          sc,
		logger:        l,
		eventCh:       eventCh,
		eventHandlers: make(map[EventHandler]struct{}),
		shutdownCh:    make(chan struct{}),
	}
	return p
}

// Start is used to initiate the event handlers. It is separate from
// create so that there isn't a race condition between creating the
// peer and registering handlers.
func (p *Peer) Start() error {
	level.Info(p.logger).Log("msg", "serf agent is starting")
	// Create serf first
	serf, err := serf.Create(p.conf)
	if err != nil {
		return errors.Wrap(err, "error creating Serf")
	}

	p.serf = serf
	go p.eventLoop()
	level.Info(p.logger).Log("msg", fmt.Sprintf("peer stats: %v", p.Stats()))
	return nil
}

// Leave prepares for a graceful shutdown of the peer and its processes.
func (p *Peer) Leave() error {
	if p.serf == nil {
		return nil
	}

	level.Info(p.logger).Log("msg", "requesting graceful leave from Serf")
	return p.serf.Leave()
}

// Shutdown closes this peer and all of its processes. Should be preceded
// by a Leave for a graceful shutdown.
func (p *Peer) Shutdown() error {
	p.shutdownLock.Lock()
	defer p.shutdownLock.Unlock()

	if p.shutdown {
		return nil
	}

	if p.serf == nil {
		goto EXIT
	}

	level.Info(p.logger).Log("msg", "requesting serf shutdown")
	if err := p.serf.Shutdown(); err != nil {
		return err
	}
EXIT:
	level.Info(p.logger).Log("msg", "shutdown complete")
	p.shutdown = true
	close(p.shutdownCh)
	return nil
}

// ShutdownCh returns a channel that can be selected to wait
// for the peer to perform a shutdown.
func (p *Peer) ShutdownCh() <-chan struct{} {
	return p.shutdownCh
}

// Serf returns the Serf agent of the running Peer.
func (p *Peer) Serf() *serf.Serf {
	return p.serf
}

// SerfConfig returns the Serf config of the running Peer.
func (p *Peer) SerfConfig() *serf.Config {
	return p.conf
}

// Join asks the Serf instance to join. See serf.Join function
func (p *Peer) Join(addrs []string, replay bool) (n int, err error) {
	level.Info(p.logger).Log("msg", fmt.Sprintf("joining: %v,  replay: %v", addrs, replay))
	ignoreOld := !replay
	n, err = p.serf.Join(addrs, ignoreOld)
	if n > 0 {
		level.Info(p.logger).Log("msg", fmt.Sprintf("joined:  %d nodes", n))
	}
	if err != nil {
		level.Warn(p.logger).Log("msg", "error joining", "err", err)
	}
	return
}

// ForceLeave is used to eject a failed node from the cluster
func (p *Peer) ForceLeave(node string) error {
	level.Info(p.logger).Log("msg", "force leaving node", "node", node)
	err := p.serf.RemoveFailedNode(node)
	if err != nil {
		level.Warn(p.logger).Log("msg", "failed to remove node", "err", err)
	}
	return err
}

// ForceLeavePrune completely removes a failed node from the
// member list entirely
func (p *Peer) ForceLeavePrune(node string) error {
	level.Info(p.logger).Log("msg", "force leaving node (prune)", "node", node)
	err := p.serf.RemoveFailedNodePrune(node)
	if err != nil {
		level.Warn(p.logger).Log("msg", "failed to remove node (prune)", "err", err)
	}
	return err
}

// UserEvent sends a UserEvent on Serf, see Serf.UserEvent
func (p *Peer) UserEvent(name string, payload []byte, coalesce bool) error {
	level.Debug(p.logger).Log("msg", "requesting user event send", "name",
		name, "coalesced", coalesce, "payload", string(payload))
	err := p.serf.UserEvent(name, payload, coalesce)
	if err != nil {
		level.Warn(p.logger).Log("msg", "failed to send user event", "err", err)
	}
	return err
}

// RegisterEventHandler adds an event handler to receive event notifications
func (p *Peer) RegisterEventHandler(eh EventHandler) {
	p.eventHandlersLock.Lock()
	defer p.eventHandlersLock.Unlock()

	p.eventHandlers[eh] = struct{}{}
	p.eventHandlerList = nil
	for eh := range p.eventHandlers {
		p.eventHandlerList = append(p.eventHandlerList, eh)
	}
}

// DeregisterEventHandler removes an EventHandler and prevents more invocations
func (p *Peer) DeregisterEventHandler(eh EventHandler) {
	p.eventHandlersLock.Lock()
	defer p.eventHandlersLock.Unlock()

	delete(p.eventHandlers, eh)
	p.eventHandlerList = nil
	for eh := range p.eventHandlers {
		p.eventHandlerList = append(p.eventHandlerList, eh)
	}
}

// eventLoop listens to events from Serf and fans out to event handlers
func (p *Peer) eventLoop() {
	serfShutdownCh := p.serf.ShutdownCh()
	for {
		select {
		case e := <-p.eventCh:
			level.Info(p.logger).Log("msg", "received event", "event", e.String())
			p.eventHandlersLock.Lock()
			handlers := p.eventHandlerList
			p.eventHandlersLock.Unlock()
			for _, eh := range handlers {
				eh.HandleEvent(e)
			}
		case <-serfShutdownCh:
			level.Warn(p.logger).Log("msg", "serf shutdown detected, quitting")
			p.Shutdown()
			return
		case <-p.shutdownCh:
			return
		}
	}
}

// Stats is used to get various runtime information and stats
func (p *Peer) Stats() map[string]map[string]string {
	local := p.serf.LocalMember()
	output := map[string]map[string]string{
		"peer": map[string]string{
			"name": local.Name,
		},
		"runtime": utils.RuntimeStats(),
		"serf":    p.serf.Stats(),
		"tags":    local.Tags,
	}
	return output
}
