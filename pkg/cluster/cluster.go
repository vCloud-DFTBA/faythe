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
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	etcdv3 "go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/clientv3/concurrency"
	"go.etcd.io/etcd/clientv3/namespace"
	"stathat.com/c/consistent"

	"github.com/vCloud-DFTBA/faythe/pkg/common"
	"github.com/vCloud-DFTBA/faythe/pkg/model"
)

// ClusterState is the state of the Cluster instance
type ClusterState int

const (
	// DefaultLeaseTTL etcd lease time-to-live in seconds
	DefaultLeaseTTL int64        = 15
	ClusterAlive    ClusterState = iota
	ClusterLeaving
	ClusterLeft
	ClusterJoining
)

func (s ClusterState) String() string {
	switch s {
	case ClusterAlive:
		return "alive"
	case ClusterJoining:
		return "joining"
	case ClusterLeft:
		return "left"
	case ClusterLeaving:
		return "leaving"
	default:
		return "unknown"
	}
}

// Cluster manages a set of member and the consistent hash ring as well.
type Cluster struct {
	logger     log.Logger
	lease      etcdv3.LeaseID
	local      model.Member
	members    map[string]model.Member
	etcdcli    *common.Etcd
	mtx        *concurrency.Mutex
	ring       *consistent.Consistent
	stopCh     chan struct{}
	state      ClusterState
	stateLock  sync.Mutex
	memberLock sync.Mutex
}

// New creates a new cluster manager instance.
func New(cid, bindAddr string, l log.Logger, e *common.Etcd) (*Cluster, error) {
	c := &Cluster{
		logger:  l,
		etcdcli: e,
		members: make(map[string]model.Member),
		stopCh:  make(chan struct{}),
	}
	if cid == "" {
		cid = common.RandToken()
		level.Info(c.logger).Log("msg", "A new cluster is starting...")
	} else {
		level.Info(c.logger).Log("msg", "A node is joining to existing cluster...")
	}
	level.Info(c.logger).Log("msg", "Use the cluster id to join", "id", cid)
	// Override the client interface with namespace
	c.etcdcli.KV = namespace.NewKV(c.etcdcli.KV, cid)
	c.etcdcli.Watcher = namespace.NewWatcher(c.etcdcli.Watcher, cid)
	c.etcdcli.Lease = namespace.NewLease(c.etcdcli.Lease, cid)

	// Create session
	sess, err := concurrency.NewSession(c.etcdcli.Client)

	if err != nil {
		return nil, err
	}

	c.mtx = concurrency.NewMutex(sess, "cluster-lock/")
	lockCtx, lockCancel := context.WithCancel(context.Background())
	defer func() {
		c.mtx.Unlock(lockCtx)
		lockCancel()
	}()

	_ = c.mtx.Lock(lockCtx)
	// Load the existing cluster
	getResp, _ := c.etcdcli.DoGet(model.DefaultClusterPrefix, etcdv3.WithPrefix())
	for _, kv := range getResp.Kvs {
		var m model.Member
		_ = json.Unmarshal(kv.Value, &m)
		c.members[m.ID] = m
	}

	c.state = ClusterJoining

	// Init a local member
	c.local, err = newLocalMember(bindAddr)
	if err != nil {
		return c, err
	}

	// Grant lease
	leaseResp, err := c.etcdcli.DoGrant(DefaultLeaseTTL)
	if err != nil {
		return c, err
	}
	c.lease = leaseResp.ID

	if _, ok := c.members[c.local.ID]; !ok {
		c.members[c.local.ID] = c.local
		// Add new member
		v, _ := json.Marshal(&c.local)
		_, err := c.etcdcli.DoPut(common.Path(model.DefaultClusterPrefix, c.local.ID),
			string(v), etcdv3.WithLease(c.lease))
		if err != nil {
			return c, err
		}
	} else {
		return c, errors.Errorf("a node %s is already cluster member", c.local.Name)
	}

	c.state = ClusterAlive

	// Init a HashRing
	c.ring = consistent.New()
	for _, m := range c.members {
		c.ring.Add(m.ID)
	}
	return c, nil
}

// State is the current state of this Cluster instance
func (c *Cluster) State() ClusterState {
	c.stateLock.Lock()
	defer c.stateLock.Unlock()
	return c.state
}

// Run watches the cluster state's changes and does its job
func (c *Cluster) Run(ctx context.Context, rc chan struct{}) {
	watch := c.etcdcli.Watch(ctx, model.DefaultClusterPrefix, etcdv3.WithPrefix())
	ticker := time.NewTicker(time.Duration(DefaultLeaseTTL) * time.Second / 2)
	for {
		select {
		case <-c.stopCh:
			ticker.Stop()
			return
		case <-ticker.C:
			_, err := c.etcdcli.DoKeepAliveOnce(c.lease)
			if err != nil {
				level.Error(c.logger).Log("msg", "Error refreshing lease for cluster member",
					"err", err)
				continue
			}
		case watchResp := <-watch:
			reload := false
			if watchResp.Err() != nil {
				level.Error(c.logger).Log("msg", "Error watching cluster state", "err", watchResp.Err())
				break
			}
			for _, event := range watchResp.Events {
				if event.Type == etcdv3.EventTypePut {
					var m model.Member
					err := json.Unmarshal(event.Kv.Value, &m)
					if err != nil {
						level.Error(c.logger).Log("msg", "Error unmarshaling event value",
							"err", err)
						continue
					}
					level.Info(c.logger).Log("msg", "A new member is joined",
						"name", m.Name, "address", m.Address)
					c.ring.Add(m.ID)
					c.members[m.ID] = m
				}
				if event.Type == etcdv3.EventTypeDelete {
					id := strings.TrimPrefix(string(event.Kv.Key), model.DefaultClusterPrefix)
					id = strings.Trim(id, "/")
					level.Info(c.logger).Log("msg", "A member is left",
						"name", c.members[id].Name, "address", c.members[id].Address)
					c.ring.Remove(id)
					delete(c.members, id)
				}
				level.Debug(c.logger).Log("msg", "The current cluster state",
					"members", fmt.Sprintf("%+v", c.members))
				reload = true
			}
			// Reload only if there is at least one correct event
			if reload {
				rc <- struct{}{}
			}
		}
	}
}

// Stop stops the member as well as the watch process
func (c *Cluster) Stop() {
	if c.state == ClusterLeaving || c.state == ClusterLeft {
		return
	}
	level.Info(c.logger).Log("msg", "A member of cluster is stopping...",
		"name", c.local.Name, "address", c.local.Address)
	c.state = ClusterLeaving
	_, err := c.etcdcli.DoRevoke(c.lease)
	if err != nil {
		level.Error(c.logger).Log("msg", "Error revoking the lease", "id", c.lease)
	}
	close(c.stopCh)
	c.state = ClusterLeft
	level.Info(c.logger).Log("msg", "A member of cluster is stopped",
		"name", c.local.Name, "address", c.local.Address)
}

// LocalIsWorker checks if the local node is the worker which has
// responsibility for the given string key.
func (c *Cluster) LocalIsWorker(key string) (string, string, bool) {
	workerID, _ := c.ring.Get(key)
	worker, _ := c.members[workerID]
	// Return the node name, it will be easier for user.
	return c.local.Name, worker.Name, workerID == c.local.ID
}

func newLocalMember(bindAddr string) (model.Member, error) {
	m := model.Member{}
	hostname, err := os.Hostname()
	if err != nil {
		return m, err
	}
	m.Name = hostname
	host, _, _ := net.SplitHostPort(bindAddr)
	// If there is no bind IP, pick an address
	if host == "0.0.0.0" {
		host, err = common.ExternalIP()
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
