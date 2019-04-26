package utils

import (
	"crypto/sha1"
	"log"
	"os"
	"path/filepath"
	"sync"
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
		Logger: log.New(f, "handlers", log.Lshortfile),
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
	h := sha1.New()
	h.Write([]byte(s))
	return string(h.Sum(nil))
}
