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
	"context"
	"crypto"
	"encoding/json"
	"errors"
	"net"
	"os"

	"github.com/go-kit/kit/log"
	"github.com/serialx/hashring"
	etcdv3 "go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/clientv3/concurrency"
	"go.etcd.io/etcd/clientv3/namespace"

	"github.com/ntk148v/faythe/pkg/model"
	"github.com/ntk148v/faythe/pkg/utils"
)

// Manager manages a set of member and the consistent hash ring as well.
type Manager struct {
	logger  log.Logger
	cluster model.Cluster
	etcdcli *etcdv3.Client
	watch   etcdv3.WatchChan
	mtx     *concurrency.Mutex
	ring    *hashring.HashRing
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewManager creates a new cluster manager instance
func NewManager(cid, bindAddr string, l log.Logger, e *etcdv3.Client) (*Manager, error) {
	ctx, cancel := context.WithCancel(context.Background())
	m := &Manager{
		logger:  l,
		etcdcli: e,
		ctx:     ctx,
		cancel:  cancel,
		cluster: model.Cluster{},
	}
	if cid == "" {
		cid = string(utils.RandToken())
	}
	// Override the client interface with namespace
	m.etcdcli.KV = namespace.NewKV(m.etcdcli.KV, cid)
	m.etcdcli.Watcher = namespace.NewWatcher(m.etcdcli.Watcher, cid)
	m.etcdcli.Lease = namespace.NewLease(m.etcdcli.Lease, cid)
	m.watch = m.etcdcli.Watch(m.ctx, model.DefaultClusterPrefix, etcdv3.WithPrefix())
	sess, err := concurrency.NewSession(m.etcdcli)
	if err != nil {
		return nil, err
	}
	m.mtx = concurrency.NewMutex(sess, "cluster-lock/")
	// Load the existing cluster
	resp, _ := m.etcdcli.Get(m.ctx, model.DefaultClusterPrefix, etcdv3.WithPrefix())
	if len(resp.Kvs) == 1 {
		_ = json.Unmarshal(resp.Kvs[0].Value, &m.cluster)
	}
	local, err := newLocalMember(bindAddr)
	if err != nil {
		return m, err
	}
	if _, ok := m.cluster.Members[local.ID]; !ok {
		m.
	}
	return m, nil
}

func newLocalMember(bindAddr string) (model.Member, error) {
	m := model.Member{}
	hostname, err := os.Hostname()
	if err != nil {
		return m, err
	}
	m.Name = hostname
	m.ID = string(utils.Hash(m.Name, crypto.MD5))
	host, _, _ := net.SplitHostPort(bindAddr)
	// If there is no bind IP, pick an address
	if host == "0.0.0.0" {
		found := false
		addrs, err := net.InterfaceAddrs()
		if err != nil {
			return m, err
		}
		for _, a := range addrs {
			var addrIP net.IP
			// Linux only
			addr, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			addrIP = addr.IP
			// Skip self-assigned IPs
			if addrIP.IsLinkLocalUnicast() {
				continue
			}
			// Found an IP
			found = true
			host = addrIP.String()
			break
		}
		if !found {
			return m, errors.New("Failed to find usable address for local member")
		}
	}
	m.Address = host
	return m, nil
}
