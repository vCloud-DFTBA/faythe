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
	"crypto/tls"
	"fmt"

	"gopkg.in/mail.v2"

	"github.com/vCloud-DFTBA/faythe/config"
	"github.com/vCloud-DFTBA/faythe/pkg/model"
)

func SendMail(a *model.ActionMail) error {
	m := mail.NewMessage()
	mc := config.Get().MailConfig
	if a.Body == "" || a.Subject == "" {
		return fmt.Errorf("mail subject or body cannot be empty")
	}

	m.SetHeaders(map[string][]string{
		"From":    {mc.Username},
		"To":      a.Receivers,
		"Subject": {a.Subject},
	})
	m.SetBody("text/html", a.Body)

	d := mail.NewDialer(mc.Host, mc.Port, mc.Username, string(mc.Password))
	d.TLSConfig = &tls.Config{InsecureSkipVerify: true}

	if err := d.DialAndSend(m); err != nil {
		return err
	}
	return nil
}
