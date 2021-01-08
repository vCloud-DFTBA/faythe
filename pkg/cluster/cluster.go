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
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	etcdv3 "go.etcd.io/etcd/clientv3"
	"stathat.com/c/consistent"

	"github.com/vCloud-DFTBA/faythe/pkg/common"
	"github.com/vCloud-DFTBA/faythe/pkg/exporter"
	"github.com/vCloud-DFTBA/faythe/pkg/model"
)

var clusterID string

// State is the state of the Cluster instance
type State int

const (
	// DefaultLeaseTTL etcd lease time-to-live in seconds
	DefaultLeaseTTL int64 = 30
	ClusterAlive    State = iota
	ClusterLeaving
	ClusterLeft
	ClusterJoining
)

func (s State) String() string {
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

// GetID makes cluster id be available
// across modules.
func GetID() string {
	return clusterID
}

// Cluster manages a set of member and the consistent hash ring as well.
type Cluster struct {
	id        string
	logger    log.Logger
	lease     etcdv3.LeaseID
	local     model.Member
	members   map[string]model.Member
	etcdcli   *common.Etcd
	ring      *consistent.Consistent
	stopCh    chan struct{}
	state     State
	stateLock sync.RWMutex
}

// New creates a new cluster manager instance.
func New(id string, bindAddr string, l log.Logger, e *common.Etcd) (*Cluster, error) {
	clusterID = id
	c := &Cluster{
		logger:  l,
		etcdcli: e,
		members: make(map[string]model.Member),
		stopCh:  make(chan struct{}),
	}
	level.Info(c.logger).Log("msg", "Use the cluster id to join", "id", clusterID)
	c.id = id

	// Load the existing cluster
	getResp, _ := c.etcdcli.DoGet(model.DefaultClusterPrefix, etcdv3.WithPrefix())
	for _, kv := range getResp.Kvs {
		var m model.Member
		_ = json.Unmarshal(kv.Value, &m)
		c.members[m.ID] = m
	}
	// Init a local member
	local, err := newLocalMember(bindAddr)
	if err != nil {
		return c, err
	}
	c.local = local
	// Join the cluster
	if err = c.join(); err != nil {
		return c, err
	}
	return c, nil
}

// getState returns the current state of this Cluster instance
func (c *Cluster) getState() State {
	c.stateLock.RLock()
	defer c.stateLock.RUnlock()
	return c.state
}

// setState for safety update the state
func (c *Cluster) setState(new State) {
	c.stateLock.Lock()
	defer c.stateLock.Unlock()
	c.state = new
}

func (c *Cluster) join() error {
	c.setState(ClusterJoining)
	grantResp, err := c.etcdcli.DoGrant(DefaultLeaseTTL)
	if err != nil {
		return err
	}
	c.lease = grantResp.ID
	if _, ok := c.members[c.local.ID]; !ok {
		c.members[c.local.ID] = c.local
		// Add new member
		v, _ := json.Marshal(&c.local)
		_, err := c.etcdcli.DoPut(common.Path(model.DefaultClusterPrefix, c.local.ID),
			string(v), etcdv3.WithLease(c.lease))
		level.Debug(c.logger).Log("msg", "The current cluster state",
			"members", fmt.Sprintf("%+v", c.members))
		if err != nil {
			return err
		}
	} else {
		return errors.Errorf("a node %s is already cluster member", c.local.Name)
	}

	// This will keep the key alive 'forever' or until we revoke it
	// or the connect is canceled.
	err = c.etcdcli.DoKeepAlive(c.lease)
	if err != nil {
		return err
	}

	c.setState(ClusterAlive)
	exporter.RegisterMemberInfo(c.id, c.local)

	// Init a HashRing
	c.ring = consistent.New()
	for _, m := range c.members {
		c.ring.Add(m.ID)
	}
	return nil
}

func (c *Cluster) leave() {
	c.setState(ClusterLeaving)
	level.Info(c.logger).Log("msg", "The local member of cluster is leaving...",
		"name", c.local.Name, "address", c.local.Address)
	_, err := c.etcdcli.DoDelete(common.Path(model.DefaultClusterPrefix, c.local.ID),
		etcdv3.WithIgnoreLease())
	if err != nil {
		level.Error(c.logger).Log("msg", "Error leaving the cluster",
			"name", c.local.Name, "address", c.local.Address)
	}
	c.setState(ClusterLeft)
	level.Info(c.logger).Log("msg", "The local member of cluster is left",
		"name", c.local.Name, "address", c.local.Address)
}

// Run watches the cluster state's changes and does its job
func (c *Cluster) Run(rc chan struct{}) {
	ctx, cancel := c.etcdcli.WatchContext()
	watch := c.etcdcli.Watch(ctx, model.DefaultClusterPrefix,
		etcdv3.WithPrefix(), etcdv3.WithCreatedNotify())
	defer func() { cancel() }()

	for {
		select {
		case <-c.stopCh:
			return
		case <-ctx.Done():
			return
		case watchResp := <-watch:
			reload := false
			if err := watchResp.Err(); err != nil {
				level.Error(c.logger).Log("msg", "Error watching cluster state", "err", err)
				eerr := common.NewEtcdErr(model.DefaultClusterPrefix, "watch", err)
				c.etcdcli.ErrCh <- eerr
				return
			}
			for _, event := range watchResp.Events {
				switch event.Type {
				case etcdv3.EventTypePut:
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
					exporter.ReportClusterJoin()
					c.members[m.ID] = m
				case etcdv3.EventTypeDelete:
					id := strings.TrimPrefix(string(event.Kv.Key), model.DefaultClusterPrefix)
					id = strings.Trim(id, "/")
					level.Info(c.logger).Log("msg", "A member is left",
						"name", c.members[id].Name, "address", c.members[id].Address)
					c.ring.Remove(id)
					exporter.ReportClusterLeave()
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
	if c.getState() == ClusterLeaving || c.getState() == ClusterLeft {
		return
	}
	close(c.stopCh)
	c.leave()
	level.Info(c.logger).Log("msg", "The local member of cluster is stopped",
		"name", c.local.Name, "address", c.local.Address)
}

// LocalIsWorker checks if the local node is the worker which has
// responsibility for the given string key.
func (c *Cluster) LocalIsWorker(key string) (string, string, bool) {
	workerID, _ := c.ring.Get(key)
	worker := c.members[workerID]
	// Return the node name, it will be easier for user.
	return c.local.Name, worker.Name, workerID == c.local.ID
}

// LocalMember returns the local node member.
func (c *Cluster) LocalMember() model.Member {
	return c.local
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
