package openstack

import (
	"encoding/json"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/orchestration/v1/stacks"
	"github.com/gophercloud/gophercloud/pagination"
	mw "github.com/ntk148v/cloudhotpot-middleware/middlewares"
)

var (
	stackOutputs         map[string]interface{}
	opts                 gophercloud.AuthOptions
	provider             gophercloud.ProviderClient
	orchestractionClient gophercloud.ServiceClient
)

func UpdateStackOutputs(listOpts stacks.ListOpts, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		// Retrieve all OpenStack environment variables
		if opts == nil {
			opts, err := openstack.AuthOptionsFromEnv()
			if err != nil {
				mw.Logger.Printf("UpdateStackOutputs - retrieve OpenStack environment varibales is failed due to %s", err.Error())
				continue
			}
		}

		// Create a general client
		if provider == nil {
			provider, err := openstack.AuthenticatedClient(opts)
			if err != nil {
				mw.Logger.Printf("UpdateStackOutputs - create provider client is failed due to %s", err.Error())
				continue
			}
		}
		// Create a ServiceClient that may be used to access the v1 orchestraction service
		if orchestractionClient == nil {
			orchestractionClient, err := openstack.NewOrchestrationV1(provider, gophercloud.EndpointOpts{
				Region: os.Getenv("OS_REGION_NAME"),
			})
			if err != nil {
				mw.Logger.Printf("UpdateStackOutputs - create Orchestraction client is failed due to %s", err.Error())
				continue
			}
		}

		// List all stacks with given options
		pager := stacks.List(client, listOpts)
		err = pager.EachPage(func(page pagination.Page) (bool, error) {
			stackList, err := stacks.ExtractStacks(page)
			if err != nil {
				return false, err
			}
			for _, s := range stacklist {
				outputValues := make(map[string]interface{})
				stack := stacks.Get(client, s.Name, s.ID)
				stackBody, _ := stack.Extract()
				for _, v := range stackBody.Outputs {
					outputValueRaw := v["output_value"].(string)
					outputValueMap := make(map[string]interface{})
					// Convert output value to map if it is in JSON string format
					if err := json.Unmarshal([]byte(outputValueRaw), &outputValueMap); err != nil {
						outputValues[v["output_key"].(string)] = outputValueRaw
						continue
					}

					outputValues[v["output_key"].(string)] = outputValueMap
				}
				if len(outputValues) != 0 {
					stackOutputs[s.ID] = outputValues
				}
			}
			return true, nil
		})
		if err != nil {
			mw.Logger.Printf("UpdateStackOutputs - get stack outputs is failed due to %s", err.Error())
		}
		time.Sleep(time.Second * 30)
	}
}

func Autoscaling(w http.ResponseWriter, req *http.Request) {

}
