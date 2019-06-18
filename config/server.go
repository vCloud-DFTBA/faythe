package config

import "faythe/utils"

// ServerConfiguration represents server-side configurations.
type ServerConfiguration struct {
	// RemoteHostPattern can define an optional regexp pattern to be matched:
	//
	// - {name} matches anything until the next dot.
	//
	// - {name:pattern} matches the given regexp pattern.
	RemoteHostPattern   string              `yaml:"remoteHostPattern,omitempty"`
	BasicAuthentication BasicAuthentication `yaml:"basicAuth,omitempty"`
}

// BasicAuthentication - HTTP Basic authentication
type BasicAuthentication struct {
	// Usenname, Password to implement HTTP basic authentication
	Username string       `yaml:"username"`
	Password utils.Secret `yaml:"password"`
}
