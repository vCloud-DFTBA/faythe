package utils

import (
	"crypto/sha256"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/pkg/errors"
)

// Secret special type for storing secrets.
type Secret string

// Flogger represents a file logger.
type Flogger struct {
	file string
	*log.Logger
}

func createFlogger(fname string) *Flogger {
	logDir := Getenv("LOG_DIR", "/var/log/faythe")
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		if err := os.MkdirAll(logDir, os.ModePerm); err != nil {
			panic(err)
		}
	}

	fpath := filepath.Join(logDir, fname)
	f, err := os.OpenFile(fpath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		os.Exit(1)
	}

	return &Flogger{
		file:   fname,
		Logger: log.New(f, "handlers: ", log.Lshortfile|log.LstdFlags),
	}
}

// NewFlogger inits a file logger.
func NewFlogger(once *sync.Once, fname string) *Flogger {
	var logger *Flogger
	once.Do(func() {
		logger = createFlogger(fname)
	})
	return logger
}

// MarshalYAML implements the yaml.Marshaler interface for Secrets.
func (s Secret) MarshalYAML() (interface{}, error) {
	if s != "" {
		return "<secret>", nil
	}
	return nil, nil
}

//UnmarshalYAML implements the yaml.Unmarshaler interface for Secrets.
func (s *Secret) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type plain Secret
	return unmarshal((*plain)(s))
}

// Getenv returns default value if environment variable
// doesn't exist.
func Getenv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// Hash compute SHA1 hashes of s given input.
func Hash(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	return string(h.Sum(nil))
}

// LookupAddr performs a reverse lookup for the network address of the form "host:port"/network address/hostname
// returning a name mapping to that address.
func LookupAddr(host string) (string, error) {
	host, _, err := net.SplitHostPort(host)
	if err != nil && !strings.Contains("missingPort", err.Error()) {
		return "", errors.Wrap(err, "parse host from hostport failed")
	}
	// Determine whether a given string is an ip or hostname
	addr := net.ParseIP(host)
	if addr == nil {
		return host, nil
	} else {
		hostname, err := net.LookupAddr(host)
		if err != nil {
			return "", errors.Wrap(err, "lookup adddress failed")
		}
		// Force get the first result, ignore the rest.
		return hostname[0], nil
	}
}
