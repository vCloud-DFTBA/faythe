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

package config

import (
	"crypto/tls"
	"fmt"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/vCloud-DFTBA/faythe/pkg/utils"
)

// Config is the top-level configuration for Faythe's config file.
type Config struct {
	GlobalConfig GlobalConfig `yaml:"global"`
	EtcdConfig   EtcdConfig   `yaml:"etcd"`
	MailConfig   MailConfig   `yaml:"mail,omitempty"`
}

type MailConfig struct {
	Host     string       `yaml:"host"`
	Protocol string       `yaml:"protocol"`
	Port     int          `yaml:"port"`
	Username string       `yaml:"username"`
	Password string       `yaml:"password"`
}

// GlobalConfig configures values that are used to config Faythe HTTP server
type GlobalConfig struct {
	// RemoteHostPattern can define an optional regexp pattern to be matched:
	//
	// - {name} matches anything until the next dot.
	//
	// - {name:pattern} matches the given regexp pattern.
	RemoteHostPattern string `yaml:"remote_host_pattern,omitempty"`
	// BasicAuthentication - HTTP Basic authentication.
	BasicAuthentication BasicAuthentication `yaml:"basic_auth,omitempty"`
}

// BasicAuthentication - HTTP Basic authentication.
type BasicAuthentication struct {
	// Usename, Password to implement HTTP basic authentication
	Username string  `yaml:"username"`
	Password string  `yaml:"password"`
}

// EtcdConfig stores Etcd related configurations.
type EtcdConfig struct {
	// Endpoints is a list of URLs.
	Endpoints []string `yaml:"endpoints"`

	// AutoSyncInterval is the interval to update endpoints with its latest members.
	// 0 disables auto-sync. By default auto-sync is disabled.
	AutoSyncInterval time.Duration `yaml:"auto_sync_interval,omitempty"`

	// DialTimeout is the timeout for failing to establish a connection.
	DialTimeout time.Duration `yaml:"dial_timeout,omitempty"`

	// DialKeepAliveTime is the time after which client pings the server to see if
	// transport is alive.
	DialKeepAliveTime time.Duration `yaml:"dial_keep_alive_time,omitempty"`

	// DialKeepAliveTimeout is the time that the client waits for a response for the
	// keep-alive probe. If the response is not received in this time, the connection is closed.
	DialKeepAliveTimeout time.Duration `yaml:"dial_keep_alive_timeout,omitempty"`

	// MaxCallSendMsgSize is the client-side request send limit in bytes.
	// If 0, it defaults to 2.0 MiB (2 * 1024 * 1024).
	// Make sure that "MaxCallSendMsgSize" < server-side default send/recv limit.
	// ("--max-request-bytes" flag to etcd or "embed.Config.MaxRequestBytes").
	MaxCallSendMsgSize int `yaml:"max_call_send_msg_size,omitempty"`

	// MaxCallRecvMsgSize is the client-side response receive limit.
	// If 0, it defaults to "math.MaxInt32", because range response can
	// easily exceed request send limits.
	// Make sure that "MaxCallRecvMsgSize" >= server-side default send/recv limit.
	// ("--max-request-bytes" flag to etcd or "embed.Config.MaxRequestBytes").
	MaxCallRecvMsgSize int `yaml:"max_call_recv_msg_size,omitempty"`

	// TLS holds the client secure credentials, if any.
	TLS *tls.Config `yaml:"tls,omitempty"`

	// Username is a user name for authentication.
	Username string `yaml:"username,omitempty"`

	// Password is a password for authentication.
	Password string `yaml:"password,omitempty"`

	// RejectOldCluster when set will refuse to create a client against an outdated cluster.
	RejectOldCluster bool `yaml:"reject_old_cluster,omitempty"`

	// PermitWithoutStream when set will allow client to send keepalive pings to server without any active streams(RPCs).
	PermitWithoutStream bool `yaml:"permit_without_stream,omitempty"`
}

const (
	etcdDefaultDialTimeout      = 2 * time.Second
	etcdDefaultKeepAliveTime    = 2 * time.Second
	etcdDefaultKeepAliveTimeOut = 6 * time.Second
)

var (
	// DefaultConfig is the default top-level configuration.
	DefaultConfig = Config{
		GlobalConfig: DefaultGlobalConfig,
		EtcdConfig:   DefaultEtcdConfig,
	}

	// DefaultGlobalConfig is the default global configuration.
	DefaultGlobalConfig = GlobalConfig{
		RemoteHostPattern:   ".*",
		BasicAuthentication: BasicAuthentication{},
	}

	// DefaultEtcdConfig is the default Etcd configuration.
	DefaultEtcdConfig = EtcdConfig{
		Endpoints:            []string{"127.0.0.1:2379"},
		DialTimeout:          etcdDefaultDialTimeout,
		DialKeepAliveTime:    etcdDefaultKeepAliveTime,
		DialKeepAliveTimeout: etcdDefaultKeepAliveTimeOut,
	}
)

// UnmarshalYAML implements the yaml.Unmarshaler interface
func (c *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	*c = DefaultConfig
	// We want to set c to the defaults and then overwrite it with the input.
	// To make unmarshal fill the plain data struct rather than calling UnmarshalYAML
	// again, we have to hide it using a type indirection.
	type plain Config
	if err := unmarshal((*plain)(c)); err != nil {
		return err
	}
	return nil
}

// String represents Configuration instance as string.
func (c *Config) String() string {
	b, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Sprintf("<error creating config string: %s>", err)
	}
	return string(b)
}

// UnmarshalYAML implements the yaml.Unmarshaler interface
func (c *GlobalConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	*c = DefaultGlobalConfig
	// We want to set c to the defaults and then overwrite it with the input.
	// To make unmarshal fill the plain data struct rather than calling UnmarshalYAML
	// again, we have to hide it using a type indirection.
	type plain GlobalConfig
	if err := unmarshal((*plain)(c)); err != nil {
		return err
	}
	return nil
}

// UnmarshalYAML implements the yaml.Unmarshaler interface
func (c *EtcdConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	*c = DefaultEtcdConfig
	// We want to set c to the defaults and then overwrite it with the input.
	// To make unmarshal fill the plain data struct rather than calling UnmarshalYAML
	// again, we have to hide it using a type indirection.
	type plain EtcdConfig
	if err := unmarshal((*plain)(c)); err != nil {
		return err
	}
	return nil
}

// UnmarshalYAML implements the yaml.Unmarshaler interface
func (c *MailConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type plain MailConfig
	if err := unmarshal((*plain)(c)); err != nil {
		return err
	}
	return nil
}
