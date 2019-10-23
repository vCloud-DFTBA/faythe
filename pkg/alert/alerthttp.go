// Copyright (c) 2019 Dat Vu Tuan <tuandatk25a@gmail.com>
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

package alert

import (
	"net"
	"net/http"
	"time"

	"github.com/avast/retry-go"
	"github.com/go-kit/kit/log"
	"github.com/ntk148v/faythe/pkg/model"
)

func Send(l log.Logger, cli *http.Client, a *model.ActionHTTP) error {
	delay, _ := time.ParseDuration(a.Delay)
	err := retry.Do(
		func() error {
			req, err := http.NewRequest(a.Method, string(a.URL), nil)
			if err != nil {
				return err
			}
			resp, err := cli.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			return nil
		},
		retry.DelayType(func(n uint, config *retry.Config) time.Duration {
			var f retry.DelayTypeFunc
			switch a.DelayType {
			case "fixed":
				f = retry.FixedDelay
			case "backoff":
				f = retry.BackOffDelay
			}
			return f(n, config)
		}),
		retry.Attempts(a.Attempts),
		retry.Delay(delay),
		retry.RetryIf(func(err error) bool {
			if err, ok := err.(net.Error); ok && err.Timeout() {
				return true
			}
			return false
		}),
	)
	return err
}
