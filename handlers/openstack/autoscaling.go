package openstack

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/orchestration/v1/stacks"
	"github.com/gophercloud/gophercloud/pagination"
	"github.com/prometheus/alertmanager/template"
)

// StacksOutputs represents the outputs of a list of stacks.
// map[stack_id:map[output_key:output_value]]
type StacksOutputs map[string]map[string]string

var sos atomic.Value

// UpdateStacksOutputs queries the outputs of stacks that was filters with a given listOpts periodically.
func UpdateStacksOutputs(logger *log.Logger, wg *sync.WaitGroup) {
	defer wg.Done()
	sos.Store(make(StacksOutputs))
	var mu sync.Mutex // used only by writers
	// Retrieve all OpenStack environment variables
	opts, err := openstack.AuthOptionsFromEnv()
	if err != nil {
		logger.Printf("UpdateStacksOutputs - retrieve OpenStack environment varibales is failed due to %s", err.Error())
		return
	}

	// Create a general client
	provider, err := openstack.AuthenticatedClient(opts)
	if err != nil {
		logger.Printf("UpdateStacksOutputs - create provider client is failed due to %s", err.Error())
		return
	}

	orchestractionClient, err := openstack.NewOrchestrationV1(provider, gophercloud.EndpointOpts{
		Region: os.Getenv("OS_REGION_NAME"),
	})
	if err != nil {
		logger.Printf("UpdateStacksOutputs - create Orchestraction client is failed due to %s", err.Error())
		return
	}

	listOpts := stacks.ListOpts{
		Tags: "scale",
	}

	for {
		// List all stacks with given options
		pager := stacks.List(orchestractionClient, listOpts)
		err := pager.EachPage(func(page pagination.Page) (bool, error) {
			mu.Lock()
			defer mu.Unlock()
			_ = sos.Load().(StacksOutputs)
			stacksOutputs := make(StacksOutputs)
			stackList, err := stacks.ExtractStacks(page)
			if err != nil {
				return false, err
			}
			for _, s := range stackList {
				outputValues := make(map[string]string)
				stack := stacks.Get(orchestractionClient, s.Name, s.ID)
				stackBody, _ := stack.Extract()
				for _, v := range stackBody.Outputs {
					outputValues[v["output_key"].(string)] = v["output_value"].(string)
				}
				if len(outputValues) != 0 {
					stacksOutputs[s.ID] = outputValues
				}
			}
			sos.Store(stacksOutputs)
			return true, nil
		})
		if err != nil {
			logger.Printf("UpdateStackOutputs - get stack outputs is failed due to %s", err.Error())
		}
		time.Sleep(time.Second * 30)
	}
}

// Autoscaling gets Webhook be triggered from Prometheus Alertmanager.
func Autoscaling(logger *log.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		defer req.Body.Close()
		data := template.Data{}
		if err := json.NewDecoder(req.Body).Decode(&data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		logger.Printf("Alerts: GroupLabels=%v, CommonLabels=%v", data.GroupLabels, data.CommonLabels)
		for _, alert := range data.Alerts {
			logger.Printf("Alert: status=%s,Labels=%v,Annotations=%v", alert.Status, alert.Labels, alert.Annotations)
			stacksOutputs := sos.Load().(StacksOutputs)
			stackID := alert.Labels["stack_id"]
			// scale_action must be one of two values: `up` and `down`.
			/// TODO: add check later.
			scaleURLKey := "scale_" + strings.ToLower(alert.Labels["scale_action"]) + "_url"
			var scaleURL string
			stack := stacksOutputs[stackID]
			if microservice, ok := alert.Labels["microservice"]; ok {
				// Convert output value (JSON string) to Map to able to index
				stackMap := make(map[string]string)
				json.Unmarshal([]byte(stack[microservice]), &stackMap)
				scaleURL = stackMap[scaleURLKey]
			} else {
				scaleURL = stack[scaleURLKey]
			}

			// Good now, create a POST request to scale URL
			resp, err := http.Post(scaleURL, "application/json", nil)
			if err != nil {
				logger.Println(err.Error())
				http.Error(w, err.Error(), http.StatusBadGateway)
				return
			}
			defer resp.Body.Close()
		}
	})
}
