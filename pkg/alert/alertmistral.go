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
	"fmt"

	"github.com/gophercloud/gophercloud/openstack/workflow/v2/executions"

	"github.com/vCloud-DFTBA/faythe/pkg/history"
	"github.com/vCloud-DFTBA/faythe/pkg/model"
)

func ExecuteWorkflow(os model.OpenStack, a *model.ActionMistral) error {
	actionHistory := history.ActionHistory{}
	actionHistory.Create(a.Type)
	createOpts := &executions.CreateOpts{
		WorkflowID:  a.WorkflowID,
		Input:       a.Input,
		Description: "Create executions for autohealing",
	}

	client, err := os.NewWorkflowClient()
	if err != nil {
		actionHistory.Update(history.Error, fmt.Sprintf("Failed mistral action %s", a.WorkflowID), "")
		return err
	}

	_, err = executions.Create(client, createOpts).Extract()
	if err != nil {
		actionHistory.Update(history.Error, fmt.Sprintf("Failed mistral action %s", a.WorkflowID), "")
		return err
	}
	actionHistory.Update(history.Success, fmt.Sprintf("Executed mistral action %s", a.WorkflowID), "")
	return nil
}
