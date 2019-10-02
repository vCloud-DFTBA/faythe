package stack

import (
	"github.com/gophercloud/gophercloud/pagination"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/orchestration/v1/stacks"
	"github.com/pkg/errors"

	"github.com/ntk148v/faythe/config"
	"github.com/ntk148v/faythe/handlers/openstack/auth"
)

// Outputs represents the outputs of stacks.
// map[stack_id:map[output_key:output_value]]
type Outputs map[string]map[string]string

func createClient(opsConf *config.OpenStackConfig) (*gophercloud.ServiceClient, error) {
	provider, err := auth.CreateProvider(opsConf)
	if err != nil {
		return nil, err
	}
	return openstack.NewOrchestrationV1(provider, gophercloud.EndpointOpts{Region: opsConf.RegionName})
}

// GetOutputs return Outputs
func GetOutputs(opsConf *config.OpenStackConfig) (Outputs, error) {
	filterOpts := opsConf.StackQuery.ListOpts
	listOpts := stacks.ListOpts{
		TenantID:   filterOpts.ProjectID,
		ID:         filterOpts.ID,
		Status:     filterOpts.Status,
		Name:       filterOpts.Name,
		AllTenants: filterOpts.AllTenants,
		Tags:       filterOpts.Tags,
		TagsAny:    filterOpts.TagsAny,
		NotTags:    filterOpts.NotTags,
		NotTagsAny: filterOpts.NotTagsAny,
	}

	client, err := createClient(opsConf)
	if err != nil {
		return nil, err
	}

	// List all stacks with given options
	pager := stacks.List(client, listOpts)
	op := make(Outputs)
	err = pager.EachPage(func(page pagination.Page) (bool, error) {
		stackList, err := stacks.ExtractStacks(page)
		if err != nil {
			return false, err
		}

		for _, s := range stackList {
			opv := make(map[string]string)
			stack := stacks.Get(client, s.Name, s.ID)
			body, _ := stack.Extract()
			for _, v := range body.Outputs {
				if ov, ok := v["output_value"].(string); ok {
					opv[v["output_key"].(string)] = ov
				}
			}
			if len(opv) != 0 {
				op[s.ID] = opv
			}
		}
		return true, nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "get outputs failed")
	}
	return op, nil
}
