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

	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"gopkg.in/yaml.v2"
)

// Config is the top-level configuration for Faythe's config file.
type Config struct {
	ServerConfig ServerConfig `yaml:"server_config"`
	EtcdConfig   EtcdConfig   `yaml:"etcd"`
	JWTConfig    JWTConfig    `yaml:"jwt"`
	MailConfig   MailConfig   `yaml:"mail,omitempty"`
	// RemoteHostPattern can define an optional regexp pattern to be matched:
	//
	// - {name} matches anything until the next dot.
	//
	// - {name:pattern} matches the given regexp pattern.
	RemoteHostPattern string `yaml:"remote_host_pattern,omitempty"`
	// PasswordHashingCost is the cost to hash the user password.
	// Check bcrypt for details: https://godoc.org/golang.org/x/crypto/bcrypt#pkg-constants
	PasswordHashingCost int    `yaml:"password_hashing_cost"`
	FernetKey           string `yaml:"fernet_key"`
	// EnableProfiling enables profiling via web interface host:port/debug/pprof/
	EnableProfiling     bool                `yaml:"enable_profiling"`
	AdminAuthentication AdminAuthentication `yaml:"admin_authentication"`
}

// ServerConfig stores configs to setup HTTP Server
type ServerConfig struct {
	EnableTLS      bool          `yaml:"enable_tls"`
	CertFile       string        `yaml:"cert_file"`
	CertKey        string        `yaml:"cert_key"`
	ReadTimeout    time.Duration `yaml:"read_timeout"`
	WriteTimeout   time.Duration `yaml:"write_timeout"`
	MaxHeaderBytes int           `yaml:"max_header_bytes"`
}

// AdminAuthentication represents the `root/admin` user authentication
type AdminAuthentication struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// MailConfig stores configs to setup a SNMP client.
type MailConfig struct {
	Host     string `yaml:"host"`
	Protocol string `yaml:"protocol"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// JWTConfig is a struct for specifying JWT configuration options
// A clone of Options struct: https://github.com/adam-hanna/jwt-auth/blob/develop/jwt/auth.go#L31
type JWTConfig struct {
	// SigningMethodString is a string to define the signing method
	// Valid signing methods:
	// - "RS256","RS384","RS512" (RSA signing method)
	// - "ES256","ES384","ES512" (ECDSA signing method)
	// More details here: https://github.com/dgrijalva/jwt-go#signing-methods-and-key-types
	SigningMethod string `yaml:"signing_method"`
	// PrivateKeyLocation is the private key path
	// $ openssl genrsa -out faythe.rsa 2048
	PrivateKeyLocation string `yaml:"private_key_location"`
	// PublicKeyLocation is the public key path
	// $ openssl rsa -in faythe.rsa -pubout > faythe.rsa.pub
	PublicKeyLocation string `yaml:"public_key_location"`
	// TTL - Token time to live
	TTL           time.Duration `yaml:"ttl"`
	IsBearerToken bool          `yaml:"is_bearer_token"`
	// Header is a name of the custom request header.
	// If IsBearerToken is set, the header name will be `Authorization`
	// with value format `Bearer <token-string>`.
	Header string `yaml:"header"`
	// The name of the property in the request where the user information
	// from the JWT will be stored.
	UserProperty string `yaml:"user_property"`
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

	// DialOptions is a list of dial options for the grpc client (e.g., for interceptors).
	// For example, pass "grpc.WithBlock()" to block until the underlying connection is up.
	// Without this, Dial returns immediately and connecting the server happens in background.
	DialOptions []grpc.DialOption
}

const (
	etcdDefaultDialTimeout       = 5 * time.Second
	etcdDefaultKeepAliveTime     = 5 * time.Second
	etcdDefaultKeepAliveTimeOut  = 6 * time.Second
	jwtDefaultSigningMethod      = "RS256"
	jwtDefaultTTL                = 60 * time.Minute
	jwtDefaultIsBearerToken      = true
	jwtDefaultUserProperty       = "user"
	jwtDefaultPrivateKeyLocation = "/etc/faythe/keys/faythe.rsa"
	jwtDefaultPublicKeyLocation  = "/etc/faythe/keys/faythe.rsa.pub"
	serverDefaultCertFile        = "/etc/faythe/certs/faythe.crt"
	serverDefaultCertKey         = "/etc/faythe/certs/faythe.key"
	serverDefaultReadTimeout     = 5 * time.Second
	serverDefaultWriteTimeout    = 5 * time.Second
	serverDefaultMaxHeaderBytes  = 1048576
)

var (
	// DefaultConfig is the default top-level configuration.
	DefaultConfig = Config{
		ServerConfig:        DefaultServerConfig,
		EtcdConfig:          DefaultEtcdConfig,
		JWTConfig:           DefaultJWTConfig,
		RemoteHostPattern:   ".*",
		EnableProfiling:     false,
		PasswordHashingCost: bcrypt.DefaultCost,
		FernetKey:           "RSkt1yqhOp9znrUzeCQRybYdRVqQGfO5G2VR-wF8OKc=",
	}

	DefaultServerConfig = ServerConfig{
		EnableTLS:      false,
		CertFile:       serverDefaultCertFile,
		CertKey:        serverDefaultCertKey,
		ReadTimeout:    serverDefaultReadTimeout,
		WriteTimeout:   serverDefaultWriteTimeout,
		MaxHeaderBytes: serverDefaultMaxHeaderBytes,
	}

	// DefaultEtcdConfig is the default Etcd configuration.
	DefaultEtcdConfig = EtcdConfig{
		Endpoints:            []string{"127.0.0.1:2379"},
		DialTimeout:          etcdDefaultDialTimeout,
		DialKeepAliveTime:    etcdDefaultKeepAliveTime,
		DialKeepAliveTimeout: etcdDefaultKeepAliveTimeOut,
		DialOptions:          []grpc.DialOption{grpc.WithBlock()}, // block until the underlying connection is up
	}

	// DefaultJWTConfig is the default JWT configuration.
	DefaultJWTConfig = JWTConfig{
		SigningMethod:      jwtDefaultSigningMethod,
		TTL:                jwtDefaultTTL,
		IsBearerToken:      jwtDefaultIsBearerToken,
		UserProperty:       jwtDefaultUserProperty,
		PrivateKeyLocation: jwtDefaultPrivateKeyLocation,
		PublicKeyLocation:  jwtDefaultPublicKeyLocation,
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

// UnmarshalYAML implements the yaml.Unmarshaler interface
func (c *JWTConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	*c = DefaultJWTConfig
	// We want to set c to the defaults and then overwrite it with the input.
	// To make unmarshal fill the plain data struct rather than calling UnmarshalYAML
	// again, we have to hide it using a type indirection.
	type plain JWTConfig
	if err := unmarshal((*plain)(c)); err != nil {
		return err
	}
	return nil
}
