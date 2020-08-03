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

package autohealer

import (
	"fmt"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/gophercloud/gophercloud/openstack/workflow/v2/executions"
	"github.com/pkg/errors"

	"github.com/vCloud-DFTBA/faythe/pkg/alert"
	"github.com/vCloud-DFTBA/faythe/pkg/cluster"
	"github.com/vCloud-DFTBA/faythe/pkg/common"
	"github.com/vCloud-DFTBA/faythe/pkg/exporter"
	"github.com/vCloud-DFTBA/faythe/pkg/model"
)

const (
	DefaultMistralActionRetries        = 5
	DefaultMistralActionRetryDelay     = 60
	DefaultMistralActionExecutionCheck = 15
)

// There are 6 different states:
// IDLE, RUNNING, SUCCESS, ERROR, PAUSED, CANCELLED
// but we just handle two main states.
const (
	WorkflowExecutionSuccessState = "SUCCESS"
	WorkflowExecutionErrorState   = "ERROR"
)

// WFLExecTracker tracks workflow execution and maxRetries if necessary
type WFLExecTracker struct {
	os         model.OpenStack
	mistralAct model.ActionMistral
	execution  *executions.Execution
	logger     log.Logger
	maxRetries int
	numRetried int
}

// NewTracker spawns new execution tracker instance
func NewTracker(l log.Logger, mistralAct model.ActionMistral, os model.OpenStack) *WFLExecTracker {
	tracker := &WFLExecTracker{
		os:         os,
		mistralAct: mistralAct,
		logger:     l,
		execution:  &executions.Execution{},
		numRetried: 0,
		maxRetries: DefaultMistralActionRetries,
	}

	return tracker
}

// Start starts execution tracker
func (tracker *WFLExecTracker) start() error {
	ticker := time.NewTicker(DefaultMistralActionExecutionCheck * time.Second)
outerloop:
	for {
		if err := tracker.executeWFL(); err != nil {
			return err
		}
		level.Debug(tracker.logger).Log("msg",
			fmt.Sprintf("Execution %d of workflow", tracker.numRetried),
			"workflow", tracker.mistralAct.WorkflowID, "execution", tracker.execution.ID)
		for {
			<-ticker.C
			exec, err := alert.GetExecution(tracker.os, tracker.execution.ID)
			if err != nil {
				level.Error(tracker.logger).Log("msg", "error while getting execution state",
					"err", err)
				continue
			}
			tracker.execution = exec
			if exec.State == WorkflowExecutionErrorState {
				level.Debug(tracker.logger).Log("msg", "execution in error state",
					"execution", tracker.execution.ID)
				time.Sleep(DefaultMistralActionRetryDelay * time.Second)
				continue outerloop
			}
			if exec.State == WorkflowExecutionSuccessState {
				return nil
			}
		}
	}
}

func (tracker *WFLExecTracker) executeWFL() error {
	var msg []interface{}
	tracker.numRetried++
	if tracker.numRetried > tracker.maxRetries {
		msg = common.CnvSliceStrToSliceInf(append([]string{
			"msg", "Retried workflow executions exceeds maxRetries"},
			tracker.mistralAct.InfoLog()...))
		level.Debug(tracker.logger).Log(msg...)
		return errors.Errorf("number of retried reached maximum")
	}
	exec, err := alert.ExecuteWorkflow(tracker.os, &tracker.mistralAct)
	if err != nil {
		msg = common.CnvSliceStrToSliceInf(append([]string{
			"msg", "Exec action failed",
			"err", err.Error()},
			tracker.mistralAct.InfoLog()...))
		level.Error(tracker.logger).Log(msg...)
		exporter.ReportFailureHealerActionCounter(cluster.GetID(), "mistral")
		return err
	}
	exporter.ReportSuccessHealerActionCounter(cluster.GetID(), "mistral")
	tracker.execution = exec
	return nil
}
