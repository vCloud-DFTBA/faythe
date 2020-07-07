// Copyright (c) 2020 Dat Vu Tuan <tuandatk25a@gmail.com>
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

package history

import (
	"encoding/json"
	"github.com/go-kit/kit/log/level"
	"time"

	"github.com/go-kit/kit/log"
	etcdv3 "go.etcd.io/etcd/clientv3"

	"github.com/vCloud-DFTBA/faythe/pkg/common"
	"github.com/vCloud-DFTBA/faythe/pkg/model"
)

type ActionState int

const (
	Success ActionState = iota
	Error
)

const (
	DefaultActionHistoryPrefix = "/actions"
	// keep history of action in etcd for 30 days (converted to second unit)
	DefaultActionHistoryRetentionTime = 30 * 24 * 3600
)

type ActionHistory struct {
	ID        string      `json:"id"`
	Type      string      `json:"type"`
	State     ActionState `json:"state"`
	Message   string      `json:"message"`
	Ref       model.URL   `json:"ref"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

type History struct {
	log     log.Logger
	etcdcli *common.Etcd
}

var history History

func Init(log log.Logger, e *common.Etcd) {
	history = History{
		log: log,
		etcdcli: e,
	}
}

func (history History) grantLease() (etcdv3.LeaseID, error) {
	resp, err := history.etcdcli.DoGrant(DefaultActionHistoryRetentionTime)
	if err != nil {
		return -1, err
	}
	return resp.ID, nil
}

func (history History) doPutWithLease(action ActionHistory) error {
	bytes, err := json.Marshal(action)
	if err != nil {
		return err
	}

	lease, err := history.grantLease()
	if err != nil {
		return err
	}

	_, err = history.etcdcli.DoPut(common.Path(DefaultActionHistoryPrefix, action.ID),
		string(bytes), etcdv3.WithLease(lease))

	if err != nil {
		return err
	}

	return nil
}

func (action ActionHistory) Save() error {
	if err := history.doPutWithLease(action); err != nil {
		return err
	}
	return nil
}

func (action *ActionHistory) Create(actionType string) {
	action.ID = common.RandToken()
	action.Type = actionType
	action.CreatedAt = time.Now()
}

func (action *ActionHistory) Update(state ActionState, message string, ref model.URL) {
	action.State = state
	action.Message = message
	action.Ref = ref
	action.UpdatedAt = time.Now()
	if err := action.Save(); err != nil {
		level.Error(history.log).Log("msg", "Saving action error")
	}
}
