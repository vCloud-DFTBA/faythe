package openstack

import (
	"encoding/json"
	"faythe/utils"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/orchestration/v1/stacks"
	"github.com/gophercloud/gophercloud/pagination"
	"github.com/prometheus/alertmanager/template"
	"github.com/spf13/viper"
)

// StacksOutputs represents the outputs of a list of stacks.
// map[stack_id:map[output_key:output_value]]
type StacksOutputs map[string]map[string]string

// ScaleResult stores scale request call information.
type ScaleResult struct {
	stackID string
	action  string
	result  string
	reason  string
}

var (
	sos           atomic.Value
	mu            sync.Mutex // used only by writers
	data          template.Data
	stacksOutputs StacksOutputs
	scaleResults  chan ScaleResult
	scaleAlerts   map[string]template.Alert
	scaleURLKey   string
	scaleURL      string
)

// UpdateStacksOutputs queries the outputs of stacks that was filters with a given listOpts periodically.
func UpdateStacksOutputs(logger *log.Logger, wg *sync.WaitGroup) {
	defer wg.Done()
	sos.Store(make(StacksOutputs))
	opts := gophercloud.AuthOptions{
		IdentityEndpoint: utils.Getenv("OS_AUTH_URL", viper.GetString("openstack.authURL")),
		Username:         utils.Getenv("OS_USERNAME", viper.GetString("openstack.username")),
		Password:         utils.Getenv("OS_PASSWORD", viper.GetString("openstack.password")),
		DomainName:       utils.Getenv("OS_DOMAIN_NAME", viper.GetString("openstack.domainName")),
		DomainID:         utils.Getenv("OS_DOMAIN_ID", viper.GetString("openstack.domainID")),
		TenantID:         utils.Getenv("OS_TENANT_ID", viper.GetString("openstack.projectID")),
		TenantName:       utils.Getenv("OS_TENANT_NAME", viper.GetString("openstack.projectName")),
	}

	// Create a general client
	provider, err := openstack.AuthenticatedClient(opts)
	if err != nil {
		logger.Printf("OpenStack/UpdateStacksOutputs - create provider client failed due to %s", err.Error())
		return
	}

	orchestractionClient, err := openstack.NewOrchestrationV1(provider, gophercloud.EndpointOpts{
		Region: utils.Getenv("OS_REGION_NAME", viper.GetString("openstack.regionName")),
	})
	if err != nil {
		logger.Printf("OpenStack/UpdateStacksOutputs - create Orchestraction client failed due to %s", err.Error())
		return
	}

	listOpts := stacks.ListOpts{
		Tags: viper.GetString("openstack.stackTags"),
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
			if len(stacksOutputs) != 0 {
				logger.Println("OpenStack/UpdateStacksOutputs - the stacks outputs: ", stacksOutputs)
			}
			sos.Store(stacksOutputs)
			return true, nil
		})
		if err != nil {
			logger.Printf("OpenStack/UpdateStacksOutputs - get stack outputs is failed due to %s", err.Error())
		}

		time.Sleep(time.Second * time.Duration(viper.GetInt("openstack.updateInterval")))
	}
}

func doScale(alert *template.Alert) {
	stack := stacksOutputs[alert.Labels["stack_id"]]
	action := strings.ToLower(alert.Labels["scale_action"])
	if len(stack) == 0 {
		scaleResults <- ScaleResult{
			stackID: alert.Labels["stack_id"],
			action:  action,
			result:  "failed",
			reason:  "Couldn't find stack",
		}
		return
	}

	// scale_action must be one of two values: `up` and `down`.
	// TODO: add check format later.
	scaleURLKey = strings.Join([]string{"scale", action, "url"}, "_")

	// There are two potential output format.
	// It might be a simple map with two keys: `scale_down_url` and `scale_up_url`.
	// It can also be a nested map which its keys are microservice name.
	if microservice, ok := alert.Labels["microservice"]; ok {
		// Convert output value (JSON string) to Map to able to index
		stackMap := make(map[string]string)
		json.Unmarshal([]byte(stack[microservice]), &stackMap)
		scaleURL = stackMap[scaleURLKey]
	} else {
		scaleURL = stack[scaleURLKey]
	}

	if scaleURL == "" {
		scaleResults <- ScaleResult{
			stackID: alert.Labels["stack_id"],
			action:  action,
			result:  "failed",
			reason:  "Couldn't find scale url in stack's output",
		}
		return
	}

	// Good now, create a POST request to scale URL
	resp, err := http.Post(scaleURL, "application/json", nil)
	if err != nil {
		scaleResults <- ScaleResult{
			stackID: alert.Labels["stack_id"],
			action:  action,
			result:  "failed",
			reason:  err.Error(),
		}
		return
	}
	scaleResults <- ScaleResult{
		stackID: alert.Labels["stack_id"],
		action:  action,
		result:  "success",
	}
	defer resp.Body.Close()
}

// Autoscaling get Webhook be trigered from Prometheus Alertmanager.
func Autoscaling(logger *log.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Get the updated stacksOutputs
		stacksOutputs := sos.Load().(StacksOutputs)
		if len(stacksOutputs) == 0 {
			logger.Println("OpenStack/Autoscaling - stacksOutput is empty now!")
			w.WriteHeader(http.StatusAccepted)
			return
		}

		alerts := data.Alerts.Firing()
		// Get alerts with scale action only, ignore the rest
		// TODO: Could reduce this step by grouping alerts from Prometheus
		// alertmanager side.
		for _, alert := range alerts {
			if _, ok := alert.Labels["scale_action"]; ok {
				// Deduce alerts. If alerts which is firing by multiple instances
				// with the same stack_id, microservice, scale_action, use just one.
				key := utils.Hash(strings.Join([]string{alert.Labels["stack_id"], alert.Labels["microservice"], alert.Labels["scale_action"]}, "_"))
				if _, ok := scaleAlerts[key]; ok {
					continue
				}
				scaleAlerts[key] = alert
			}
		}

		scaleResults := make(chan ScaleResult, len(scaleAlerts))

		for _, alert := range scaleAlerts {
			logger.Printf("OpenStack/Autoscaling - Alert: status=%s,Labels=%v,Annotations=%v", alert.Status, alert.Labels, alert.Annotations)
			go doScale(&alert)
		}

		for i := 0; i < len(scaleAlerts); i++ {
			rs := <-scaleResults
			msg := fmt.Sprintf("OpenStack/Autoscaling - Send request scale %s to stack %s: %s", rs.action, rs.stackID, rs.result)
			if rs.reason != "" {
				msg += "because " + rs.reason
			}
			logger.Printf(msg)
		}
		w.WriteHeader(http.StatusAccepted)
	})
}
