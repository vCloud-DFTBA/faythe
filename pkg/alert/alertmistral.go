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

package alert

import (
	"github.com/gophercloud/gophercloud/openstack/workflow/v2/executions"

	"github.com/vCloud-DFTBA/faythe/pkg/model"
)

func ExecuteWorkflow(os model.OpenStack, a *model.ActionMistral) (*executions.Execution, error) {
	createOpts := &executions.CreateOpts{
		WorkflowID:  a.WorkflowID,
		Input:       a.Input,
		Description: "Create executions for autohealing",
	}

	client, err := os.NewWorkflowClient()
	if err != nil {
		return nil, err
	}

	exec, err := executions.Create(client, createOpts).Extract()
	if err != nil {
		return nil, err
	}

	return exec, nil
}

func GetExecution(os model.OpenStack, execId string) (*executions.Execution, error) {
	client, err := os.NewWorkflowClient()
	if err != nil {
		return nil, err
	}

	exec, err := executions.Get(client, execId).Extract()
	if err != nil {
		return nil, err
	}

	return exec, nil
}
