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
	"github.com/vCloud-DFTBA/faythe/pkg/model"
)

const (
	DefaultMistralActionRetries        = 5
	DefaultMistralActionRetryDelay     = 60
	DefaultMistralActionExecutionCheck = 15
)

const (
	WorkflowExecutionSuccessState = "SUCCESS"
	WorkflowExecutionErrorState   = "ERROR"
)

// WFLTracker tracks workflow execution and maxRetries if necessary
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
outerloop:
	for {
		ticker := time.NewTicker(DefaultMistralActionExecutionCheck * time.Second)
		if err := tracker.executeWFL(); err != nil {
			return err
		}
		level.Debug(tracker.logger).Log("msg",
			fmt.Sprintf("Execution %d of workflow", tracker.numRetried),
			"workflow", tracker.mistralAct.WorkflowID, "execution", tracker.execution.ID)
		for {
			select {
			case <-ticker.C:
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
					ticker.Stop()
					time.Sleep((DefaultMistralActionRetryDelay - DefaultMistralActionExecutionCheck) * time.Second)
					continue outerloop
				}
				if exec.State == WorkflowExecutionSuccessState {
					return nil
				}
			}
		}
	}
}

func (tracker *WFLExecTracker) executeWFL() error {
	tracker.numRetried += 1
	if tracker.numRetried > tracker.maxRetries {
		level.Debug(tracker.logger).Log("msg", "Retried workflow executions exceeds maxRetries",
			"workflow", tracker.mistralAct.WorkflowID)
		return errors.Errorf("number of retried reached maximum")
	}
	exec, err := alert.ExecuteWorkflow(tracker.os, &tracker.mistralAct)
	if err != nil {
		level.Error(tracker.logger).Log("msg", "Error while executing workflow", "err", err)
		return err
	}
	tracker.execution = exec
	return nil
}
