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

package utils

import (
	"crypto/sha256"
	"hash/fnv"
	"strings"
)

// HashFNV generates a new 64-bit number from a given string
// using 64-bit FNV-1a hash function.
func HashFNV(s string) string {
	h := fnv.New64a()
	h.Write([]byte(s))
	return string(h.Sum64())
}

// HashSHA generates a new slice of byte hash from a given string
// using SHA256 hash algorithms.
func HashSHA(s string) []byte {
	h := sha256.New()
	h.Write([]byte(s))
	return h.Sum(nil)
}

// Path returns a etcd key path.
func Path(keys ...string) string {
	return strings.Join(append([]string{}, keys...), "/")
}

// Secret special type for storing secrets.
type Secret string

// MarshalYAML implements the yaml.Marshaler interface for Secrets.
func (s Secret) MarshalYAML() (interface{}, error) {
	if s != "" {
		return "<secret>", nil
	}
	return nil, nil
}

// UnmarshalYAML implements the yaml.Unmarshaler interface for Secrets.
func (s *Secret) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type plain Secret
	return unmarshal((*plain)(s))
}

func (s Secret) MarshalJSON() ([]byte, error) {
	if s != "" {
		return []byte(`"<secret>"`), nil
	}
	return nil, nil
}
