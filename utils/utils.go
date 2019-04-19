package utils

import (
	"crypto/sha1"
	"os"
)

// Secret special type for storing secrets.
type Secret string

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
