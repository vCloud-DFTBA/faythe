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

package jwt

import (
	"time"
)

// Options is a struct for specifying configuration options
type Options struct {
	PrivateKeyLocation string
	PublicKeyLocation  string
	HMACKey            []byte
	SigningMethod      string
	TTL                time.Duration
	IsBearerToken      bool
	Header             string
	UserProperty       string
}
