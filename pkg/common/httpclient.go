// Copyright (c) 2020 Kien Nguyen-Tuan <kiennt2609@gmail.com>
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
	"crypto/tls"
	"net"
	"net/http"
	"time"
)

const (
	defaultKeepAliveTimeout    = 600 * time.Second
	defaultTimeout             = 15 * time.Second
	defaultMaxIdleConns        = 100
	defaultMaxIdleConnsPerHost = 100
)

// NewHTTPClient returns the net/http.Client with a
// custom set of configs.
func NewHTTPClient() *http.Client {
	defaultTransport := &http.Transport{
		Dial: (&net.Dialer{
			KeepAlive: defaultKeepAliveTimeout,
		}).Dial,
		MaxIdleConns:        defaultMaxIdleConns,
		MaxIdleConnsPerHost: defaultMaxIdleConnsPerHost,
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: defaultTransport,
		Timeout:   defaultTimeout,
	}
	return client
}
