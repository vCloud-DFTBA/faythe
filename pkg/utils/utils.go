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
	"crypto"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"hash"
	"hash/fnv"
	"reflect"
	"strings"
)

// HashFNV generates a new 64-bit number from a given string
// using 64-bit FNV-1a hash function.
func HashFNV(s string) string {
	h := fnv.New64a()
	h.Write([]byte(s))
	return string(h.Sum64())
}

// Hash generates a new slice of bytee hash from a given string
// using a given hash algorithms.
func Hash(s string, f crypto.Hash) []byte {
	var h hash.Hash
	switch f {
	case crypto.MD5:
		h = md5.New()
	case crypto.SHA1:
		h = sha1.New()
	case crypto.SHA256:
		h = sha256.New()
	case crypto.SHA512:
		h = sha512.New()
	default:
	}
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

// Find tells whether string contains x.
// op - boolean operator, expected `AND` `OR` string value.
// x - could be string or slice of string.
func Find(a []string, x interface{}, op string) bool {
	var r bool
	switch reflect.TypeOf(x).Kind() {
	case reflect.String:
		for _, n := range a {
			if x == n {
				r = true
				break
			}
		}
	case reflect.Slice:
		v := reflect.ValueOf(x)
		for i := 0; i < v.Len(); i++ {
			r = Find(a, v.Index(i).String(), op)
			// If operator is OR, break the loop immediately when found the first match
			if strings.ToLower(op) == "or" && r {
				break
			}
		}
	}
	return r
}
