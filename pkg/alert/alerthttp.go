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
	"bytes"
	"crypto/tls"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/avast/retry-go"
	"github.com/pkg/errors"

	"github.com/vCloud-DFTBA/faythe/pkg/common"
	"github.com/vCloud-DFTBA/faythe/pkg/model"
)

// SendHTTP constructs and sends a HTTP request.
func SendHTTP(cli *http.Client, a *model.ActionHTTP) error {
	delay, _ := common.ParseDuration(a.Delay)
	err := retry.Do(
		func() error {
			req, err := http.NewRequest(a.Method, string(a.URL), nil)
			if err != nil {
				return err
			}
			if a.Header != nil {
				for k, v := range a.Header {
					req.Header.Set(k, v)
				}
			}
			if a.Body != nil {
				b, err := json.Marshal(a.Body)
				if err != nil {
					return err
				}

				req.Body = ioutil.NopCloser(bytes.NewReader(b))
				req.ContentLength = int64(len(b))
			}

			tr := &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}

			if strings.Contains(a.URL.String(), "https") {
				cli.Transport = tr
			}

			resp, err := cli.Do(req)
			if err != nil {
				return err
			}
			// Close the response body
			if resp != nil {
				defer resp.Body.Close()
			}
			// Success is indicated with 2xx status codes
			statusOK := resp.StatusCode >= 200 && resp.StatusCode < 300
			if !statusOK {
				return errors.Errorf("non-OK HTTP status: %s", resp.Status)
			}
			// Read the body even the data is not important
			// this must to do
			_, err = io.Copy(ioutil.Discard, resp.Body)
			if err != nil {
				return err
			}
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
			return common.RetryableError(err)
		}),
	)
	return err
}
