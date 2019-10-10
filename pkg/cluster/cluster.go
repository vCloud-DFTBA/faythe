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

const (
	DefaultGossipInterval   = 200 * time.Millisecond
	DefaultTCPTimeout       = 10 * time.Second
	DefaultProbeTimeout     = 500 * time.Millisecond
	DefaultProbeInterval    = 1 * time.Second
	DefaultPushPullInterval = 30 * time.Second
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
	Peers            string
	PeerTimeout      time.Duration
	GossipInterval   time.Duration
	TCPTimeout       time.Duration
	ProbeTimeout     time.Duration
	ProbeInterval    time.Duration
	PushPullInterval time.Duration
}

// Peer is a single peer in a gossip cluster.
type Peer struct {
	mtx    sync.RWMutex
	logger log.Logger
	opts   *Options
	mlist  *memberlist.Memberlist
}

// Clone from Prometheus alertmanager
// https://github.com/prometheus/alertmanager/blob/master/cluster/cluster.go
// Create creates a new peer.
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
		logger: l,
		opts:   o,
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
	}
	p.mlist = ml
	return p, nil
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
