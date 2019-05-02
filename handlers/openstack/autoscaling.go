package openstack

import (
	"encoding/json"
	"faythe/handlers/openstack/stack"
	"faythe/utils"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/alertmanager/template"
	"github.com/spf13/viper"
)

// ScaleResult stores scale request call information.
type ScaleResult struct {
	stackID string
	action  string
	result  string
	reason  string
}

var (
	sos         atomic.Value
	mu          sync.Mutex // used only by writers
	data        template.Data
	scaleAlerts map[string]template.Alert
	scaleURLKey string
	scaleURL    string
	logger      *utils.Flogger
	once        sync.Once
)

func init() {
	logger = utils.NewFlogger(&once, "autoscaling.log")
}

// UpdateStacksOutputs queries the outputs of stacks that was filters with a given listOpts periodically.
func UpdateStacksOutputs(wg *sync.WaitGroup) {
	defer wg.Done()
	sos.Store(make(stack.Outputs))

	for {
		mu.Lock() // synchronize with other potential writers
		_ = sos.Load().(stack.Outputs)
		stacksOp, err := stack.GetOutputs()
		if err != nil {
			logger.Println("Cannot update stacks outputs: ", err)
		} else {
			logger.Println("Stacks outputs: ", stacksOp)
			sos.Store(stacksOp)
		}
		mu.Unlock()
		time.Sleep(time.Second * viper.GetDuration("openstack.stackQuery.updateInterval"))
	}
}

func doScale(scaleResults chan<- ScaleResult, stack map[string]string, stackID, action, microservice string) {
	if len(stack) == 0 {
		scaleResults <- ScaleResult{
			stackID: stackID,
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
	if microservice != "" {
		// Convert output value (JSON string) to Map to able to index
		stackMap := make(map[string]string)
		json.Unmarshal([]byte(stack[microservice]), &stackMap)
		scaleURL = stackMap[scaleURLKey]
	} else {
		scaleURL = stack[scaleURLKey]
	}

	if scaleURL == "" {
		scaleResults <- ScaleResult{
			stackID: stackID,
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
			stackID: stackID,
			action:  action,
			result:  "failed",
			reason:  err.Error(),
		}
		return
	}
	scaleResults <- ScaleResult{
		stackID: stackID,
		action:  action,
		result:  "success",
	}
	defer resp.Body.Close()
}

// Autoscaling get Webhook be trigered from Prometheus Alertmanager.
func Autoscaling() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Get the updated stacksOutputs
		stacksOutputs := sos.Load().(stack.Outputs)
		if len(stacksOutputs) == 0 {
			logger.Println("stacksOutput is empty now!")
			w.WriteHeader(http.StatusAccepted)
			return
		}

		alerts := data.Alerts.Firing()
		scaleAlerts = make(map[string]template.Alert)
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
			logger.Printf("Alert: status=%s,Labels=%v,Annotations=%v", alert.Status, alert.Labels, alert.Annotations)
			stack := stacksOutputs[alert.Labels["stack_id"]]
			go doScale(scaleResults, stack, alert.Labels["stack_id"], strings.ToLower(alert.Labels["scale_action"]), alert.Labels["microservice"])
		}

		for i := 0; i < len(scaleAlerts); i++ {
			rs := <-scaleResults
			msg := fmt.Sprintf("Send request scale %s to stack %s: %s", rs.action, rs.stackID, rs.result)
			if rs.reason != "" {
				msg += "because " + rs.reason
			}
			logger.Printf(msg)
		}
		w.WriteHeader(http.StatusAccepted)
	})
}
