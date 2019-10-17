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
	"os"
	"time"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	"github.com/vCloud-DFTBA/faythe/pkg/utils"
)

// Config is the top-level configuration for Faythe's config file.
type Config struct {
	GlobalConfig GlobalConfig `yaml:"global"`
	EtcdConfig   EtcdConfig   `yaml:"etcd"`
	PeerConfig   PeerConfig   `yaml:"cluster"`
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
	Username string       `yaml:"username"`
	Password utils.Secret `yaml:"password"`
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
	Password utils.Secret `yaml:"password,omitempty"`

	// RejectOldCluster when set will refuse to create a client against an outdated cluster.
	RejectOldCluster bool `yaml:"reject_old_cluster,omitempty"`

	// PermitWithoutStream when set will allow client to send keepalive pings to server without any active streams(RPCs).
	PermitWithoutStream bool `yaml:"permit_without_stream,omitempty"`
}

const (
	etcdDefaultDialTimeout      = 2 * time.Second
	etcdDefaultKeepAliveTime    = 2 * time.Second
	etcdDefaultKeepAliveTimeOut = 6 * time.Second
	// minBroadcastTimeout applies a lower bound to the broadcast timeout interval
	minBroadcastTimeout = time.Second
)

var (
	// DefaultConfig is the default top-level configuration.
	DefaultConfig = Config{
		GlobalConfig: DefaultGlobalConfig,
		EtcdConfig:   DefaultEtcdConfig,
		PeerConfig:   DefaultPeerConfig,
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

	// DefaultPeerConfig is the default Peer configuration
	DefaultPeerConfig = PeerConfig{
		BindAddr:         "0.0.0.0:8601",
		AdvertiseAddr:    "",
		ReplayOnJoin:     false,
		Profile:          "lan",
		BroadcastTimeout: 5 * time.Second,
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

// PeerConfig is the configuration that can be set for an Agent. Some of these
// configurations are exposed as command-line flags to `faythe`, whereas
// many of the more advanced configurations can only be set by creating
// a configuration file.
type PeerConfig struct {
	NodeName string `yaml:"node_name"`

	// BindAddr is the address that the Peer's communication ports
	// will bind to. Peer will use this address to bind for both TCP
	// and UDP connections. If no port is present in the address, the default
	// port will be used.
	BindAddr string `yaml:"bind"`

	// AdvertiseAddr is the address that the Peer will advertise to other
	// members of the cluster. Can be used for basic NAT traversal where
	// where both the internal ip:port and external ip:port are known.
	AdvertiseAddr string `yaml:"advertise"`

	// Tags are used to attach key/value metadata to a node.
	Tags map[string]string `yaml:"tags"`

	// ReplayOnJoin tells Serf to replay past user events
	// when joining based on a `StartJoin`.
	ReplayOnJoin bool `yaml:"replay_on_join"`

	// StartJoin is a list of addresses to attempt to join when the
	// Peer starts. If Peer is unable to communicate with any of these addresses,
	// then the agent will error and exit.
	StartJoin []string `yaml:"start_join"`

	// ReconnectInterval controls how often we attempt to connect to a failed node.
	ReconnectInterval time.Duration `yaml:"reconnect_interval"`

	// ReconnectTimeout controls for how long we attempt to connect to a failed node
	// removing it from the cluster.
	ReconnectTimeout time.Duration `yaml:"reconnect_timeout"`

	// BroadcastTimeout is the string retry interval. This interval
	// controls the timeout for broadcast events. This defaults to 5 seconds.
	BroadcastTimeout time.Duration `yaml:"broadcast_timeout"`

	// Profile is used to select a timing profile for Peer. The supported choices
	// are "wan", "lan", and "local". The default is "lan"
	Profile string `yaml:"profile"`
}

func (c *PeerConfig) Validate() error {
	_, _, err := utils.AddParts(c.BindAddr)
	if err != nil {
		return err
	}
	if c.AdvertiseAddr != "" {
		_, _, err := utils.AddParts(c.AdvertiseAddr)
		if err != nil {
			return err
		}
	}

	if c.NodeName == "" {
		hostname, err := os.Hostname()
		if err != nil {
			return errors.Wrap(err, "setting Peer's node name as hostname")
			c.NodeName = hostname
		}
	}

	// Check for sane broadcast timeout
	if c.BroadcastTimeout < minBroadcastTimeout {
		// If broadcastTimeout is too low, setting to 1s.
		c.BroadcastTimeout = minBroadcastTimeout
	}

	switch c.Profile {
	case "lan":
	case "wan":
	case "local":
	default:
		return errors.Errorf("Unknown profile: %s", c.Profile)
	}

	return nil
}

func (c *PeerConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	*c = DefaultPeerConfig
	// We want to set c to the defaults and then overwrite it with the input.
	// To make unmarshal fill the plain data struct rather than calling UnmarshalYAML
	// again, we have to hide it using a type indirection.
	type plain PeerConfig
	if err := unmarshal((*plain)(c)); err != nil {
		return err
	}
	return c.Validate()
}
