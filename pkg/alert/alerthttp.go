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
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/avast/retry-go"
	"github.com/vCloud-DFTBA/faythe/pkg/common"
	"github.com/vCloud-DFTBA/faythe/pkg/history"
	"github.com/vCloud-DFTBA/faythe/pkg/model"
)

func SendHTTP(cli *http.Client, a *model.ActionHTTP, add ...map[string]map[string]string) error {
	actionHistory := history.ActionHistory{}
	actionHistory.Create(a.Type)
	delay, _ := common.ParseDuration(a.Delay)
	err := retry.Do(
		func() error {
			req, err := http.NewRequest(a.Method, string(a.URL), nil)
			if err != nil {
				return err
			}
			if add != nil {
				req.Header.Set("Content-Type", "application/json")
				if header, ok := add[0]["header"]; ok {
					req.SetBasicAuth(header["username"], header["password"])
				}

				if body, ok := add[0]["body"]; ok {
					b, err := json.Marshal(body)
					if err != nil {
						return err
					}

					req.Body = ioutil.NopCloser(bytes.NewReader(b))
					req.ContentLength = int64(len(b))
				}
			}
			resp, err := cli.Do(req)
			// Close the response body
			if resp != nil {
				defer resp.Body.Close()
			}
			if err != nil {
				return err
			}
			// Read the body even the data is not important
			// this must to do
			_, err = io.Copy(ioutil.Discard, resp.Body)
			if err != nil {
				return err
			}
			actionHistory.Update(history.Success, fmt.Sprintf("Sent request to %s", string(a.URL)), "")
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
	actionHistory.Update(history.Error, fmt.Sprintf("Failed request to %s", string(a.URL)), "")
	return err
}
