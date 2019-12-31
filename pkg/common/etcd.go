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

package common

import (
	"context"
	"strings"
	"time"

	etcdv3 "go.etcd.io/etcd/clientv3"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	defaultKvRequestTimeout    = 10 * time.Second
	defaultLeaseRequestTimeout = 2 * time.Second
)

// Etcd is the Etcd v3 client wrapper with addition context.
type Etcd struct {
	*etcdv3.Client
	ErrCh chan error
}

// NewEtcd constructs a new Etcd client
func NewEtcd(cfg etcdv3.Config) (*Etcd, error) {
	cli, err := etcdv3.New(cfg)
	if err != nil {
		return nil, err
	}
	return &Etcd{cli, make(chan error, 1)}, nil
}

// Context returns a cancelable context and its cancel function.
func (e *Etcd) Context() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), defaultKvRequestTimeout)
}

// LeaseContext returns a cancelable context and its cancel function.
func (e *Etcd) LeaseContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), defaultLeaseRequestTimeout)
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
func (e *Etcd) WatchContext() (context.Context, context.CancelFunc) {
	return context.WithCancel(etcdv3.WithRequireLeader(context.Background()))
}

// DoGet retrieves keys.
// More details please refer to etcd clientv3.KV interface.
func (e *Etcd) DoGet(key string, opts ...etcdv3.OpOption) (*etcdv3.GetResponse, error) {
	ctx, cancel := e.Context()
	defer cancel()
	result, err := e.Get(ctx, key, opts...)
	if err != nil {
		e.ErrCh <- err
	}
	return result, err
}

// DoPut puts a key-value pair into etcd.
// More details please refer to etcd clientv3.KV interface.
func (e *Etcd) DoPut(key, val string, opts ...etcdv3.OpOption) (*etcdv3.PutResponse, error) {
	ctx, cancel := e.Context()
	defer cancel()
	result, err := e.Put(ctx, key, val, opts...)
	if err != nil {
		e.ErrCh <- err
	}
	return result, err
}

// DoDelete deletes a key, or optionally using WithRange(end), [key, end).
// More details please refer to etcd clientv3.KV interface.
func (e *Etcd) DoDelete(key string, opts ...etcdv3.OpOption) (*etcdv3.DeleteResponse, error) {
	ctx, cancel := e.Context()
	defer cancel()
	result, err := e.Delete(ctx, key, opts...)
	if err != nil {
		e.ErrCh <- err
	}
	return result, err
}

// DoGrant creates a new lease.
func (e *Etcd) DoGrant(ttl int64) (*etcdv3.LeaseGrantResponse, error) {
	ctx, cancel := e.LeaseContext()
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
func (e *Etcd) DoKeepAliveOnce(id etcdv3.LeaseID) (*etcdv3.LeaseKeepAliveResponse, error) {
	ctx, cancel := e.LeaseContext()
	defer cancel()
	result, err := e.KeepAliveOnce(ctx, id)
	if err != nil {
		e.ErrCh <- err
	}
	return result, err
}

// DoRevoke revokes the given lease.
func (e *Etcd) DoRevoke(id etcdv3.LeaseID) (*etcdv3.LeaseRevokeResponse, error) {
	ctx, cancel := e.LeaseContext()
	defer cancel()
	result, err := e.Revoke(ctx, id)
	if err != nil {
		e.ErrCh <- err
	}
	return result, err
}

// Run waits for Etcd client's error.
func (e *Etcd) Run(stopc chan struct{}) {
	for {
		select {
		case <-e.ErrCh:
			stopc <- struct{}{}
		}
	}
}

// CheckKey accepts a given Etcd key with format:
// then finds the key. Return true if one instance is found,
// otherwise false.
func (e *Etcd) CheckKey(key string) bool {
	resp, err := e.DoGet(key, etcdv3.WithCountOnly())
	if err != nil {
		return false
	}
	if resp.Count == 1 {
		return true
	}
	return false
}

// Stolen from the integration test:
// https://github.com/etcd-io/etcd/blob/master/clientv3/integration/server_shutdown_test.go#L367
// e.g. due to clock drifts in server-side,
// client context times out first in server-side
// while original client-side context is not timed out yet
func isServerCtxTimeout(err error) bool {
	if err == nil {
		return false
	}
	ev, ok := status.FromError(err)
	if !ok {
		return false
	}
	code := ev.Code()
	return code == codes.DeadlineExceeded && strings.Contains(err.Error(), "context deadline exceeded")
}

// In grpc v1.11.3+ dial timeouts can error out with transport.ErrConnClosing. Previously dial timeouts
// would always error out with context.DeadlineExceeded.
func isClientTimeout(err error) bool {
	if err == nil {
		return false
	}
	if err == context.DeadlineExceeded {
		return true
	}
	ev, ok := status.FromError(err)
	if !ok {
		return false
	}
	code := ev.Code()
	return code == codes.DeadlineExceeded
}

func isCanceled(err error) bool {
	if err == nil {
		return false
	}
	if err == context.Canceled {
		return true
	}
	ev, ok := status.FromError(err)
	if !ok {
		return false
	}
	code := ev.Code()
	return code == codes.Canceled
}

func isUnavailable(err error) bool {
	if err == nil {
		return false
	}
	if err == context.Canceled {
		return true
	}
	ev, ok := status.FromError(err)
	if !ok {
		return false
	}
	code := ev.Code()
	return code == codes.Unavailable
}
