package stack

import (
	"faythe/handlers/openstack/auth"
	"faythe/utils"
	"github.com/gophercloud/gophercloud/pagination"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/orchestration/v1/stacks"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

// Outputs represents the outputs of stacks.
// map[stack_id:map[output_key:output_value]]
type Outputs map[string]map[string]string

func createClient() (*gophercloud.ServiceClient, error) {
	provider, err := auth.CreateProvider()
	if err != nil {
		return nil, err
	}
	return openstack.NewOrchestrationV1(provider, gophercloud.EndpointOpts{
		Region: utils.Getenv("OS_REGION_NAME", viper.GetString("openstack.regionName")),
	})
}

// GetOutputs return Outputs
func GetOutputs() (Outputs, error) {
	listOpts := stacks.ListOpts{
		TenantID:   viper.GetString("openstack.stackQuery.listOpts.projectID"),
		ID:         viper.GetString("openstack.stackQuery.listOpts.id"),
		Status:     viper.GetString("openstack.stackQuery.listOpts.status"),
		Name:       viper.GetString("openstack.stackQuery.listOpts.name"),
		AllTenants: viper.GetBool("openstack.stackQuery.listOpts.allTenants"),
		Tags:       viper.GetString("openstack.stackQuery.listOpts.tags"),
		TagsAny:    viper.GetString("openstack.stackQuery.listOpts.tagsAny"),
		NotTags:    viper.GetString("openstack.stackQuery.listOpts.notTags"),
		NotTagsAny: viper.GetString("openstack.stackQuery.listOpts.notTagsAny"),
	}

	client, err := createClient()
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
