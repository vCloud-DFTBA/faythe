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
	"strconv"
	"strings"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	etcdv3 "go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/clientv3/namespace"
	"go.etcd.io/etcd/etcdserver/api/v3rpc/rpctypes"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Set of raw Prometheus metrics.
// NOTE(kiennt): To prevent import cycle, have to move this part from the exporter
// common etcd.
// Do not increment directly, use Report* methods.
var (
	failureEtcdRequestsCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "faythe",
			Subsystem: "etcd",
			Name:      "request_failures_total",
			Help:      "The total number of Etcd request failures (not retryable).",
		},
		[]string{"cluster", "action", "path"},
	)
)

func init() {
	prometheus.MustRegister(failureEtcdRequestsCounter)
}

func ReportFailureEtcdRequestCounter(clusterID, action, path string) {
	failureEtcdRequestsCounter.WithLabelValues(clusterID, action, path).Inc()
}

const (
	defaultKvRequestTimeout    = 10 * time.Second
	defaultLeaseRequestTimeout = 2 * time.Second
	// DefaultEtcdRetryCount for Etcd operations
	DefaultEtcdRetryCount = 3
	// DefaultEtcdtIntervalBetweenRetries for Etcd failed operations
	DefaultEtcdtIntervalBetweenRetries = time.Second * 5
)

// EtcdError is the Etcd error wrapper
type EtcdError struct {
	action string
	path   string
	err    error
}

func (ee EtcdError) Error() string {
	return ee.err.Error()
}

// NewEtcdErr returns an Etcd error.
func NewEtcdErr(path, action string, err error) EtcdError {
	return EtcdError{action, path, err}
}

// Etcd is the Etcd v3 client wrapper with addition context.
type Etcd struct {
	*etcdv3.Client
	namespace string
	logger    log.Logger
	ErrCh     chan EtcdError
}

// NewEtcd constructs a new Etcd client
func NewEtcd(l log.Logger, ns string, cfg etcdv3.Config) (*Etcd, error) {
	cli, err := etcdv3.New(cfg)
	if err != nil {
		return nil, err
	}
	// Override the client interface with namespace
	cli.Watcher = namespace.NewWatcher(cli.Watcher, ns)
	cli.Lease = namespace.NewLease(cli.Lease, ns)
	cli.KV = namespace.NewKV(cli.KV, ns)
	return &Etcd{cli, ns, l, make(chan EtcdError, 1)}, nil
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
	var (
		result *etcdv3.GetResponse
		err    error
		retry  bool
	)
	for i := 1; i < DefaultEtcdRetryCount+1; i++ {
		ctx, cancel := e.Context()
		result, err = e.Get(ctx, key, opts...)
		cancel()
		retry = e.isRetryNeeded(err, "get", key, i)
		if retry {
			time.Sleep(DefaultEtcdtIntervalBetweenRetries)
			continue
		}
		break
	}
	if err != nil {
		eerr := NewEtcdErr(key, "get", err)
		e.ErrCh <- eerr
	}
	return result, err
}

// DoPut puts a key-value pair into etcd.
// More details please refer to etcd clientv3.KV interface.
func (e *Etcd) DoPut(key, val string, opts ...etcdv3.OpOption) (*etcdv3.PutResponse, error) {
	var (
		result *etcdv3.PutResponse
		err    error
		retry  bool
	)
	for i := 1; i < DefaultEtcdRetryCount+1; i++ {
		ctx, cancel := e.Context()
		result, err = e.Put(ctx, key, val, opts...)
		cancel()
		retry = e.isRetryNeeded(err, "put", key, i)
		if retry {
			time.Sleep(DefaultEtcdtIntervalBetweenRetries)
			continue
		}
		break
	}
	if err != nil {
		eerr := NewEtcdErr(key, "put", err)
		e.ErrCh <- eerr
	}
	return result, err
}

// DoDelete deletes a key, or optionally using WithRange(end), [key, end).
// More details please refer to etcd clientv3.KV interface.
func (e *Etcd) DoDelete(key string, opts ...etcdv3.OpOption) (*etcdv3.DeleteResponse, error) {
	var (
		result *etcdv3.DeleteResponse
		err    error
		retry  bool
	)
	for i := 1; i < DefaultEtcdRetryCount+1; i++ {
		ctx, cancel := e.Context()
		result, err = e.Delete(ctx, key, opts...)
		cancel()
		retry = e.isRetryNeeded(err, "delete", key, i)
		if retry {
			time.Sleep(DefaultEtcdtIntervalBetweenRetries)
			continue
		}
		break
	}
	if err != nil {
		eerr := NewEtcdErr(key, "delete", err)
		e.ErrCh <- eerr
	}
	return result, err
}

// DoGrant creates a new lease.
func (e *Etcd) DoGrant(ttl int64) (*etcdv3.LeaseGrantResponse, error) {
	var (
		result *etcdv3.LeaseGrantResponse
		err    error
		retry  bool
	)
	for i := 1; i < DefaultEtcdRetryCount+1; i++ {
		ctx, cancel := e.LeaseContext()
		result, err = e.Grant(ctx, ttl)
		cancel()
		retry = e.isRetryNeeded(err, "grant", strconv.FormatInt(ttl, 10), i)
		if retry {
			time.Sleep(DefaultEtcdtIntervalBetweenRetries)
			continue
		}
		break
	}
	if err != nil {
		eerr := NewEtcdErr("", "grant", err)
		e.ErrCh <- eerr
	}
	return result, err
}

// DoKeepAliveOnce renews the lease once. The response corresponds to the
// first message from calling KeepAlive. If the response has a recoverable
// error, KeepAliveOnce will retry the RPC with a new keep alive message.
func (e *Etcd) DoKeepAliveOnce(id etcdv3.LeaseID) (*etcdv3.LeaseKeepAliveResponse, error) {
	var (
		result *etcdv3.LeaseKeepAliveResponse
		err    error
		retry  bool
	)
	for i := 1; i < DefaultEtcdRetryCount+1; i++ {
		ctx, cancel := e.LeaseContext()
		result, err = e.KeepAliveOnce(ctx, id)
		cancel()
		retry = e.isRetryNeeded(err, "keep-alive-once", strconv.FormatInt(int64(id), 10), i)
		if retry {
			time.Sleep(DefaultEtcdtIntervalBetweenRetries)
			continue
		}
		break
	}
	if err != nil && !IsNotFound(err) {
		eerr := NewEtcdErr("", "keep-alive-once", err)
		e.ErrCh <- eerr
	}
	return result, err
}

// DoKeepAlive attempts to keep the given lease alive forever. If the keepalive responses posted
// to the channel are not consumed promptly the channel may become full. When full, the lease
// client will continue sending keep alive requests to the etcd server, but will drop responses
// until there is capacity on the channel to send more responses.
//
// If client keep alive loop halts with an unexpected error (e.g. "etcdserver: no leader") or
// canceled by the caller (e.g. context.Canceled), KeepAlive returns a ErrKeepAliveHalted error
// containing the error reason.
//
// The returned "LeaseKeepAliveResponse" channel closes if underlying keep
// alive stream is interrupted in some way the client cannot handle itself;
// given context "ctx" is canceled or timed out.
func (e *Etcd) DoKeepAlive(id etcdv3.LeaseID) (<-chan *etcdv3.LeaseKeepAliveResponse, error) {
	var (
		result <-chan *etcdv3.LeaseKeepAliveResponse
		err    error
	)
	result, err = e.KeepAlive(context.Background(), id)
	if err != nil {
		eerr := NewEtcdErr("", "keep-alive", err)
		e.ErrCh <- eerr
	}
	return result, err
}

// DoRevoke revokes the given lease.
func (e *Etcd) DoRevoke(id etcdv3.LeaseID) (*etcdv3.LeaseRevokeResponse, error) {
	var (
		result *etcdv3.LeaseRevokeResponse
		err    error
		retry  bool
	)
	for i := 1; i < DefaultEtcdRetryCount+1; i++ {
		ctx, cancel := e.LeaseContext()
		result, err = e.Revoke(ctx, id)
		cancel()
		retry = e.isRetryNeeded(err, "keep-alive-once", string(id), i)
		if retry {
			time.Sleep(DefaultEtcdtIntervalBetweenRetries)
			continue
		}
		break
	}
	if err != nil {
		eerr := NewEtcdErr("", "revoke", err)
		e.ErrCh <- eerr
	}
	return result, err
}

// Run waits for Etcd client's error.
func (e *Etcd) Run(stopc chan struct{}) {
	for {
		err := <-e.ErrCh
		ReportFailureEtcdRequestCounter(e.namespace, err.action, err.path)
		stopc <- struct{}{}
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

// isRetryNeeded checks if for the given error does a retry needed.
func (e *Etcd) isRetryNeeded(err error, fn string, key string, retryCount int) bool {
	if isClientTimeout(err) || isServerCtxTimeout(err) ||
		err == rpctypes.ErrTimeout || err == rpctypes.ErrTimeoutDueToLeaderFail {
		level.Debug(e.logger).Log("msg", "retry execute", "action", fn,
			"err", err, "key", key, "count", retryCount)
		return true
	}
	// NOTE(kiennt): Check isUnavailable or isCanceled?
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

// IsNotFound verifies the type of given error is NotFound or not.
func IsNotFound(err error) bool {
	if err == nil || err == context.Canceled {
		return false
	}
	if eerr, ok := err.(rpctypes.EtcdError); ok {
		return eerr.Code() == codes.NotFound
	}
	return false
}
