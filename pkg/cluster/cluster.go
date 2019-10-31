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
	"fmt"
	"net"
	"os"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/serialx/hashring"
	etcdv3 "go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/clientv3/concurrency"
	"go.etcd.io/etcd/clientv3/namespace"

	"github.com/vCloud-DFTBA/faythe/pkg/model"
	"github.com/vCloud-DFTBA/faythe/pkg/utils"
)

// DefaultLeaseTTL etcd lease time-to-live in seconds
const DefaultLeaseTTL int64 = 30

// Cluster manages a set of member and the consistent hash ring as well.
type Cluster struct {
	logger  log.Logger
	lease   etcdv3.LeaseID
	local   model.Member
	members map[string]model.Member
	etcdcli *etcdv3.Client
	watch   etcdv3.WatchChan
	mtx     *concurrency.Mutex
	ring    *hashring.HashRing
	ctx     context.Context
	cancel  context.CancelFunc
	stopCh  chan struct{}
}

// New creates a new cluster manager instance
func New(cid, bindAddr string, l log.Logger, e *etcdv3.Client) (*Cluster, error) {
	ctx, cancel := context.WithCancel(context.Background())
	c := &Cluster{
		logger:  l,
		etcdcli: e,
		ctx:     ctx,
		cancel:  cancel,
		members: make(map[string]model.Member),
		stopCh:  make(chan struct{}),
	}
	if cid == "" {
		cid = utils.RandToken()
	}
	level.Info(c.logger).Log("msg", "A new cluster is starting... Use the cluster id to join",
		"id", cid)
	// Override the client interface with namespace
	c.etcdcli.KV = namespace.NewKV(c.etcdcli.KV, cid)
	c.etcdcli.Watcher = namespace.NewWatcher(c.etcdcli.Watcher, cid)
	c.etcdcli.Lease = namespace.NewLease(c.etcdcli.Lease, cid)
	c.watch = c.etcdcli.Watch(c.ctx, model.DefaultClusterPrefix, etcdv3.WithPrefix())
	sess, err := concurrency.NewSession(c.etcdcli)
	if err != nil {
		return nil, err
	}
	c.mtx = concurrency.NewMutex(sess, "cluster-lock/")
	_ = c.mtx.Lock(c.ctx)
	// Load the existing cluster
	getResp, _ := c.etcdcli.Get(c.ctx, model.DefaultClusterPrefix, etcdv3.WithPrefix())
	for _, kv := range getResp.Kvs {
		var m model.Member
		_ = json.Unmarshal(kv.Value, &m)
		c.members[m.ID] = m
	}

	// Init a local member
	c.local, err = newLocalMember(bindAddr)
	if err != nil {
		return c, err
	}

	// Grant lease
	leaseResp, err := c.etcdcli.Grant(c.ctx, DefaultLeaseTTL)
	if err != nil {
		return c, err
	}
	c.lease = leaseResp.ID

	if _, ok := c.members[c.local.ID]; !ok {
		c.members[c.local.ID] = c.local
		// Add new member
		v, _ := json.Marshal(&c.local)
		_, err := c.etcdcli.Put(c.ctx, utils.Path(model.DefaultClusterPrefix, c.local.ID),
			string(v), etcdv3.WithLease(c.lease))
		if err != nil {
			return c, err
		}
	}
	defer c.mtx.Unlock(c.ctx)

	// Init a HashRing
	nodes := make([]string, len(c.members))
	for _, m := range c.members {
		// Use node's name/id/address?
		nodes = append(nodes, m.ID)
	}
	c.ring = hashring.New(nodes)
	return c, nil
}

// Run watches the cluster state's changes and does its job
func (c *Cluster) Run(rc chan bool) {
	ticker := time.NewTicker(time.Duration(DefaultLeaseTTL) * time.Second / 2)
	for {
		select {
		case <-c.stopCh:
			ticker.Stop()
			return
		case <-ticker.C:
			_, err := c.etcdcli.KeepAliveOnce(c.ctx, c.lease)
			if err != nil {
				level.Error(c.logger).Log("msg", "Error refreshing lease for cluster member",
					"err", err)
				continue
			}
			level.Debug(c.logger).Log("msg", "Renew lease for cluster member")
		case watchResp := <-c.watch:
			for _, event := range watchResp.Events {
				var m model.Member
				err := json.Unmarshal(event.Kv.Value, &m)
				if err != nil {
					continue
				}
				if event.Type == etcdv3.EventTypePut {
					level.Debug(c.logger).Log("msg", "A new member is joined",
						"name", m.Name, "address", m.Address)
					c.ring.AddNode(m.ID)
					c.members[m.ID] = m
				}
				if event.Type == etcdv3.EventTypeDelete {
					level.Debug(c.logger).Log("msg", "A new member is left",
						"name", m.Name, "address", m.Address)
					c.ring.RemoveNode(m.ID)
					delete(c.members, m.ID)
				}
				level.Debug(c.logger).Log("msg", "The current cluster state",
					"members", fmt.Sprintf("%+v", c.members))
			}
			rc <- true
		}
	}
}

// Stop stops the member as well as the watch process
func (c *Cluster) Stop() {
	level.Info(c.logger).Log("msg", "A member of cluster is stopping...",
		"name", c.local.Name, "address", c.local.Address)
	close(c.stopCh)
	c.cancel()
	level.Info(c.logger).Log("msg", "A member of cluster is stopped",
		"name", c.local.Name, "address", c.local.Address)
}

// LocalMember returns the current local node
func (c *Cluster) LocalMember() model.Member {
	return c.local
}

// HashRing returns the cluster's consistent hash ring
func (c *Cluster) HashRing() *hashring.HashRing {
	return c.ring
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
		host, err = utils.ExternalIP()
		if err != nil {
			return m, err
		}
	}
	m.Address = host
	if err := m.Validate(); err != nil {
		return m, err
	}
	return m, nil
}
