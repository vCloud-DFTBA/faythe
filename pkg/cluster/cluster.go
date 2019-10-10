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
	"crypto"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/hashicorp/memberlist"
	"github.com/ntk148v/faythe/pkg/utils"
	"github.com/pkg/errors"
)

// PeerStatus is the state that a peer is in.
const (
	StatusNone PeerStatus = iota
	StatusAlive
	StatusFailed
)

func (s PeerStatus) String() string {
	switch s {
	case StatusNone:
		return "none"
	case StatusAlive:
		return "alive"
	case StatusFailed:
		return "failed"
	default:
		panic(fmt.Sprintf("unknown PeerStatus: %d", s))
	}
}

const (
	DefaultGossipInterval   = 200 * time.Millisecond
	DefaultProbeTimeout     = 500 * time.Millisecond
	DefaultProbeInterval    = 1 * time.Second
	DefaultPushPullInterval = 30 * time.Second
	DefaultRefreshInterval  = 15 * time.Second
)

type logWriter struct {
	l log.Logger
}

func (l *logWriter) Write(b []byte) (int, error) {
	return len(b), level.Debug(l.l).Log("memberlist", string(b))
}

// Options for the cluster handler
type Options struct {
	BindAddr         string
	AdvertiseAddr    string
	Peers            []string
	PeerTimeout      time.Duration
	GossipInterval   time.Duration
	TCPTimeout       time.Duration
	ProbeTimeout     time.Duration
	ProbeInterval    time.Duration
	PushPullInterval time.Duration
}

// Peer is a single peer in a gossip cluster.
type Peer struct {
	mtx         sync.RWMutex
	logger      log.Logger
	opts        *Options
	mlist       *memberlist.Memberlist
	peers       map[string]peer
	failedPeers []peer
	peerLock    sync.RWMutex
	initPeers   []string
	state       PeerStatus
	shutdownCh  chan struct{}
}

// peer is an internal type used for bookkeeping. It holds the state of peers
// in the cluster.
type peer struct {
	status    PeerStatus
	leaveTime time.Time

	*memberlist.Node
}

// Create creates a new peer.
// Clone from Prometheus alertmanager
// https://github.com/prometheus/alertmanager/blob/master/cluster/cluster.go
func Create(l log.Logger, o *Options) (*Peer, error) {
	bindHost, bindPortStr, err := net.SplitHostPort(o.BindAddr)
	if err != nil {
		return nil, errors.Wrap(err, "invalid listen address")
	}
	bindPort, err := strconv.Atoi(bindPortStr)
	if err != nil {
		return nil, errors.Wrap(err, "invalid listen address, wrong port")
	}
	var (
		advertiseHost string
		advertisePort int
	)

	if o.AdvertiseAddr != "" {
		var advertisePortStr string
		advertiseHost, advertisePortStr, err = net.SplitHostPort(o.AdvertiseAddr)
		if err != nil {
			return nil, errors.Wrap(err, "invalid advertise address")
		}
		advertisePort, err = strconv.Atoi(advertisePortStr)
		if err != nil {
			return nil, errors.Wrap(err, "invalid advertise address, wrong port")
		}
	}

	// Initial validation of user-specified advertise address.
	addr, err := findAdvertiseAddress(bindHost, advertiseHost)
	if err != nil {
		level.Warn(l).Log("err", "couldn't deduce an advertise address: "+err.Error())
	} else if isUnroutable(addr.String()) {
		level.Warn(l).Log("err", "this node advertise itself on an unroutable address", "addr", addr.String())
		level.Warn(l).Log("err", "this node will be unreachable in the cluster")
		level.Warn(l).Log("err", "provide --cluster.advertise-address as a routable IP address or hostname")
	} else if isAny(o.BindAddr) && advertiseHost == "" {
		// memberlist doesn't advertise properly when the bind address is empty or unspecified.
		level.Info(l).Log("msg", "setting advertise address explicitly", "addr", addr.String(), "port", bindPort)
		advertiseHost = addr.String()
		advertisePort = bindPort
	}
	p := &Peer{
		logger:     l,
		opts:       o,
		peers:      map[string]peer{},
		initPeers:  o.Peers,
		state:      StatusAlive,
		shutdownCh: make(chan struct{}),
	}
	cfg := memberlist.DefaultLANConfig()
	cfg.Name = string(utils.Hash(o.BindAddr, crypto.MD5))
	cfg.BindAddr = bindHost
	cfg.BindPort = bindPort
	cfg.GossipInterval = o.GossipInterval
	cfg.ProbeTimeout = o.ProbeTimeout
	cfg.ProbeInterval = o.ProbeInterval
	cfg.PushPullInterval = o.PushPullInterval
	cfg.TCPTimeout = o.TCPTimeout
	cfg.LogOutput = &logWriter{l: l}
	ml, err := memberlist.Create(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "create memberlist")
	}
	if advertiseHost != "" {
		cfg.AdvertiseAddr = advertiseHost
		cfg.AdvertisePort = advertisePort
		p.setInitialFailed(p.initPeers, fmt.Sprintf("%s:%d", advertiseHost, advertisePort))
	} else {
		p.setInitialFailed(p.initPeers, o.BindAddr)
	}
	p.mlist = ml
	return p, nil
}

// Join is used to take an init peers and attempt to join a cluster.
func (p *Peer) Join() error {
	// Do a quick state check
	if p.State() != StatusAlive {
		return fmt.Errorf("Peer can't join after leave or shutdown")
	}
	n, err := p.mlist.Join(p.initPeers)
	// TODO(kiennt): Add retry
	// reconnectInterval & reconnectTimeout.
	if err != nil {
		level.Warn(p.logger).Log("msg", "failed to join cluster", "err", err)
		return err
	}
	level.Debug(p.logger).Log("msg", "joined cluster", "peers", n)
	return nil
}

// ClusterSize returns the current number of alive members in the cluster.
func (p *Peer) ClusterSize() int {
	return p.mlist.NumMembers()
}

// Leave the cluster, waiting up to timeout.
func (p *Peer) Leave(timeout time.Duration) error {
	level.Debug(p.logger).Log("msg", "leaving cluster")
	return p.mlist.Leave(timeout)
}

// State is the current state of this Peer instance.
func (p *Peer) State() PeerStatus {
	return p.state
}

// All peers are initially added to the failed list. They will be removed from
// this list in peerJoin when making their initial connection.
func (p *Peer) setInitialFailed(peers []string, myAddr string) {
	if len(peers) == 0 {
		return
	}

	p.peerLock.RLock()
	defer p.peerLock.RUnlock()

	now := time.Now()
	for _, peerAddr := range peers {
		if peerAddr == myAddr {
			// Don't add ourselves to the initially failing list,
			// we don't connect to ourselves.
			continue
		}
		host, port, err := net.SplitHostPort(peerAddr)
		if err != nil {
			continue
		}
		ip := net.ParseIP(host)
		if ip == nil {
			// Don't add textual addresses since memberlist only advertises
			// dotted decimal or IPv6 addresses.
			continue
		}
		portUint, err := strconv.ParseUint(port, 10, 16)
		if err != nil {
			continue
		}

		pr := peer{
			status:    StatusFailed,
			leaveTime: now,
			Node: &memberlist.Node{
				Addr: ip,
				Port: uint16(portUint),
			},
		}
		p.failedPeers = append(p.failedPeers, pr)
		p.peers[peerAddr] = pr
	}
}

func (p *Peer) removeFailedPeers(timeout time.Duration) {
	p.peerLock.Lock()
	defer p.peerLock.Unlock()

	now := time.Now()

	keep := make([]peer, 0, len(p.failedPeers))
	for _, pr := range p.failedPeers {
		if pr.leaveTime.Add(timeout).After(now) {
			keep = append(keep, pr)
		} else {
			level.Debug(p.logger).Log("msg", "failed peer has timed out", "peer", pr.Node, "addr", pr.Address())
			delete(p.peers, pr.Name)
		}
	}

	p.failedPeers = keep
}

// Name returns the unique ID of this peer in the cluster.
func (p *Peer) Name() string {
	return p.mlist.LocalNode().Name
}

// Peers returns the peers in the cluster.
func (p *Peer) Peers() []*memberlist.Node {
	return p.mlist.Members()
}

func isUnroutable(addr string) bool {
	if host, _, err := net.SplitHostPort(addr); err == nil {
		addr = host
	}
	if ip := net.ParseIP(addr); ip != nil && (ip.IsUnspecified() || ip.IsLoopback()) {
		return true // typically 0.0.0.0 or localhost
	} else if ip == nil && strings.ToLower(addr) == "localhost" {
		return true
	}
	return false
}

func isAny(addr string) bool {
	if host, _, err := net.SplitHostPort(addr); err == nil {
		addr = host
	}
	return addr == "" || net.ParseIP(addr).IsUnspecified()
}
