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
	"crypto"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"hash"
	"hash/fnv"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/pkg/errors"
	prommodel "github.com/prometheus/common/model"
	"golang.org/x/crypto/bcrypt"
)

// BasicAuthTransport is an http.RoundTripper that authenticates all requests
// using HTTP Basic Authentication with the provided username and password
type BasicAuthTransport struct {
	Username string
	Password string
	// Transport is the underlying HTTP transport to use when making requests.
	// It will default to http.DefaultTransport if nil
	Transport http.RoundTripper
}

// RoundTrip implements the RoundTripper interface.
func (t *BasicAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// To set extra headers, we must make a copy of the Request so
	// that we don't modify the Request we were given. This is required by the
	// specification of http.RoundTripper.
	//
	// Since we are going to modify only req.Header here, we only need a deep copy
	// of req.Header.
	clnReq := new(http.Request)
	*clnReq = *req
	clnReq.Header = make(http.Header, len(req.Header))
	for k, s := range req.Header {
		clnReq.Header[k] = append([]string(nil), s...)
	}

	clnReq.SetBasicAuth(t.Username, t.Password)
	return t.transport().RoundTrip(clnReq)
}

func (t *BasicAuthTransport) transport() http.RoundTripper {
	if t.Transport == nil {
		return http.DefaultTransport
	}
	return t.Transport
}

// GenerateBcryptHash return the hash password
func GenerateBcryptHash(password string, cost int) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	return string(bytes), err
}

// CheckPasswordAgainstHash compares the password with the hashed password
func CheckPasswordAgainstHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// HashFNV generates a new 64-bit number from a given string
// using 64-bit FNV-1a hash function.
func HashFNV(s string) string {
	h := fnv.New64a()
	h.Write([]byte(s))
	return string(h.Sum64())
}

// Hash generates a new slice of bytee hash from a given string
// using a given hash algorithms.
func Hash(s string, f crypto.Hash) string {
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
	}
	h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// RandToken generates a random 16-bit token
func RandToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}

// Path returns a etcd key path.
func Path(keys ...string) string {
	for index, key := range keys {
		keys[index] = key
	}
	return strings.Join(append([]string{}, keys...), "/")
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

// AddParts returns the parts of the address
func AddParts(address string) (string, int, error) {
	_, _, err := net.SplitHostPort(address)
	if err != nil {
		return "", 0, err
	}

	// Get the address
	addr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		return "", 0, err
	}

	return addr.IP.String(), addr.Port, nil
}

// RuntimeStats is used to return various runtime information
func RuntimeStats() map[string]string {
	return map[string]string{
		"os":         runtime.GOOS,
		"arch":       runtime.GOARCH,
		"version":    runtime.Version(),
		"max_procs":  strconv.FormatInt(int64(runtime.GOMAXPROCS(0)), 10),
		"goroutines": strconv.FormatInt(int64(runtime.NumGoroutine()), 10),
		"cpu_count":  strconv.FormatInt(int64(runtime.NumCPU()), 10),
	}
}

// ExternalIP returns an external ip address of the current host
func ExternalIP() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			return ip.String(), nil
		}
	}
	return "", errors.New("a node didn't connect to any networks")
}

// RetryableError determines that given error is retryable or not.
func RetryableError(err error) bool {
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return true
	}

	switch t := err.(type) {
	case *net.OpError:
		if t.Op == "dial" {
			return false
		}
		if t.Op == "read" {
			// Accept retry if there is connection refused error
			return true
		}
	case syscall.Errno:
		if t == syscall.ECONNREFUSED {
			// Accept retry if there is connection refused error
			return true
		}
	}
	return false
}

// ReachableTCP checks an address is reachable via TCP.
func ReachableTCP(addr string) error {
	u, _ := url.Parse(addr)
	_, err := net.DialTimeout("tcp", u.Host, 3*time.Second)
	return err
}

// ParseDuration parses a string into a time.Duration, assuming that a year
// always has 365d, a week always has 7d, and a day always has 24h.
func ParseDuration(durationStr string) (time.Duration, error) {
	duration, err := prommodel.ParseDuration(durationStr)
	return time.Duration(duration), err
}
