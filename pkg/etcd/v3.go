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

package etcd

import (
	"context"
	"time"

	etcdv3 "go.etcd.io/etcd/clientv3"
)

const (
	defaultKvRequestTimeout    = 10 * time.Second
	defaultLeaseRequestTimeout = 2 * time.Second
)

// V3 is the Etcd v3 client wrapper with addition context.
type V3 struct {
	*etcdv3.Client
	ErrCh chan error
}

// New constructs a new V3 client
func New(cfg etcdv3.Config) (*V3, error) {
	cli, err := etcdv3.New(cfg)
	if err != nil {
		return nil, err
	}
	return &V3{cli, make(chan error, 1)}, nil
}

// Context returns a cancelable context and its cancel function.
func (e *V3) Context(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithTimeout(ctx, defaultKvRequestTimeout)
}

// LeaseContext returns a cancelable context and its cancel function.
func (e *V3) LeaseContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithTimeout(ctx, defaultLeaseRequestTimeout)
}

// WatchContext wraps context with the WithRequireLeader
// If the context is "context.Background/TODO", returned "WatchChan" will
// not be closed and block until event is triggered, except when server
// returns a non-recoverable error (e.g. ErrCompacted).
// For example, when context passed with "WithRequireLeader" and the
// connected server has no leader (e.g. due to network partition),
// error "etcdserver: no leader" (ErrNoLeader) will be returned,
// and then "WatchChan" is closed with non-nil "Err()".
// In order to prevent a watch stream being stuck in a partitioned node,
// make sure to wrap context with "WithRequireLeader".
func (e *V3) WatchContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithCancel(etcdv3.WithRequireLeader(ctx))
}

// DoGet retrieves keys.
// More details please refer to etcd clientv3.KV interface.
func (e *V3) DoGet(ctx context.Context, key string, opts ...etcdv3.OpOption) (*etcdv3.GetResponse, error) {
	ctx, cancel := e.Context(ctx)
	defer cancel()
	result, err := e.Get(ctx, key, opts...)
	if err != nil {
		e.ErrCh <- err
	}
	return result, err
}

// DoPut puts a key-value pair into etcd.
// More details please refer to etcd clientv3.KV interface.
func (e *V3) DoPut(ctx context.Context, key, val string, opts ...etcdv3.OpOption) (*etcdv3.PutResponse, error) {
	ctx, cancel := e.Context(ctx)
	defer cancel()
	result, err := e.Put(ctx, key, val, opts...)
	if err != nil {
		e.ErrCh <- err
	}
	return result, err
}

// DoDelete deletes a key, or optionally using WithRange(end), [key, end).
// More details please refer to etcd clientv3.KV interface.
func (e *V3) DoDelete(ctx context.Context, key string, opts ...etcdv3.OpOption) (*etcdv3.DeleteResponse, error) {
	ctx, cancel := e.Context(ctx)
	defer cancel()
	result, err := e.Delete(ctx, key, opts...)
	if err != nil {
		e.ErrCh <- err
	}
	return result, err
}

// DoGrant creates a new lease.
func (e *V3) DoGrant(ctx context.Context, ttl int64) (*etcdv3.LeaseGrantResponse, error) {
	ctx, cancel := e.LeaseContext(ctx)
	defer cancel()
	result, err := e.Grant(ctx, ttl)
	if err != nil {
		e.ErrCh <- err
	}
	return result, err
}

// DoKeepAliveOnce renews the lease once. The response corresponds to the
// first message from calling KeepAlive. If the response has a recoverable
// error, KeepAliveOnce will retry the RPC with a new keep alive message.
func (e *V3) DoKeepAliveOnce(ctx context.Context, id etcdv3.LeaseID) (*etcdv3.LeaseKeepAliveResponse, error) {
	ctx, cancel := e.LeaseContext(ctx)
	defer cancel()
	result, err := e.KeepAliveOnce(ctx, id)
	if err != nil {
		e.ErrCh <- err
	}
	return result, err
}

// DoRevoke revokes the given lease.
func (e *V3) DoRevoke(ctx context.Context, id etcdv3.LeaseID) (*etcdv3.LeaseRevokeResponse, error) {
	ctx, cancel := e.LeaseContext(ctx)
	defer cancel()
	result, err := e.Revoke(ctx, id)
	if err != nil {
		e.ErrCh <- err
	}
	return result, err
}

// Run waits for Etcd client's error.
func (e *V3) Run(stopc chan struct{}) {
	for {
		select {
		case err := <-e.ErrCh:
			if err == context.Canceled || err == context.DeadlineExceeded {
				stopc <- struct{}{}
			}
		}
	}
}
