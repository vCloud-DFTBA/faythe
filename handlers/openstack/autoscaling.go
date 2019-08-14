package openstack

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/alertmanager/template"
	"github.com/prometheus/common/model"

	"faythe/config"
	"faythe/handlers/openstack/auth"
	"faythe/utils"
)

// ScalePolicy describes the name of service (type OS::Heat::ScalingPolicy)
// that is defined in template.
type ScalePolicy string

const (
	heatSvcName ServiceName    = "heat"
	heatSvcPort ServicePort    = "8004"
	heatSvcVer  ServiceVersion = "v1"
	scaleOut    ScalePolicy    = "scaleout_policy"
	scaleIn     ScalePolicy    = "scalein_policy"
)

// Scaler does scale action with input attributes.
type Scaler struct {
	Conf       *config.OpenStackConfig
	Alert      template.Alert
	HTTPClient *http.Client
	WaitGroup  *sync.WaitGroup
	Logger     *utils.Flogger
	Policy     ScalePolicy
}

func (s *Scaler) defineAction() {
	if s.Alert.Status == string(model.AlertFiring) {
		s.Policy = scaleOut
	} else {
		s.Policy = scaleIn
	}
}

func (s *Scaler) genSignalURL() string {
	// TODO(kiennt): Check key in labels.
	labels := s.Alert.Labels
	signalURL := fmt.Sprintf("%s/%s/stacks/%s/%s/resources/%s/signal",
		s.Conf.Endpoints[string(heatSvcName)],
		labels["project_id"],
		labels["stack_asg_name"],
		labels["stack_asg_id"],
		s.Policy)
	return signalURL
}

func (s *Scaler) printLog(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a)
	s.Logger.Printf("Stack %s/%s - %s\n",
		s.Alert.Labels["stack_asg_name"],
		s.Policy,
		msg)
}

func (s *Scaler) do() {
	defer s.WaitGroup.Done()
	labels := s.Alert.Labels
	// Generate a simple fingerprint aka signature
	// that represents for Alert.
	av := append(labels.Values(), s.Alert.StartsAt.String())
	fingerprint := utils.Hash(strings.Join(av, "_"))

	// Check this alert was already received
	if _, ok := existingAlerts.Get(fingerprint); ok {
		if s.Alert.Status == string(model.AlertFiring) {
			s.printLog("Ignore existing alert %s from host %s",
				labels["alertname"], labels["instance"])
			return
		} else if s.Alert.Status == string(model.AlertResolved) {
			s.printLog("Alert %s/%s was resolved, delete it from existing alerts list.",
				labels["alertname"],
				labels["instance"])
			existingAlerts.Delete(fingerprint)
		}
	}

	s.defineAction()
	signalURL := s.genSignalURL()
	authPC, err := auth.CreateProvider(s.Conf)
	if err != nil {
		s.printLog("Invalid OpenStack configuration: %s", err)
		return
	}
	req, err := http.NewRequest("POST", signalURL, nil)
	if err != nil {
		s.printLog("Create request for url %s failed: ", err)
		return
	}
	// Good now, create a POST request to scale URL
	if token, ok := authPC.AuthenticatedHeaders()["X-Auth-Token"]; ok {
		req.Header.Set("X-Auth-Token", token)
	}
	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		s.printLog("Send POST request %s  to %s failed: %s\n",
			signalURL, labels["stack_asg_name"], err)
		return
	}
	defer resp.Body.Close()
}

// AutoScaling get information from Prometheus Alertmanager webhook to trigger
// OpenStack autoscale action.
func AutoScaling() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if logger == nil {
			logger = utils.NewFlogger(&once, "autoscaling.log")
		}

		// Generate Endpoints if not be confiured.
		generateEndpoints(heatSvcName, heatSvcPort, heatSvcVer)

		defer r.Body.Close()
		var data template.Data
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		vars := mux.Vars(r)
		confs := config.Get().OpenStackConfigs
		conf, ok := confs[vars["ops-name"]]
		if !ok {
			supported := make([]string, len(confs))
			for k := range confs {
				supported = append(supported, k)
			}

			err := UnknownOpenStackError{
				correct: supported,
				wrong:   vars["ops-name"],
			}
			logger.Println(err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		alerts := data.Alerts
		// Create client & set timeout
		client := &http.Client{}
		client.Timeout = time.Second * 15
		var wg sync.WaitGroup
		for _, alert := range alerts {
			s := Scaler{
				Conf:       conf,
				Alert:      alert,
				HTTPClient: client,
				WaitGroup:  &wg,
				Logger:     logger,
			}
			wg.Add(1)
			go s.do()
		}
		wg.Wait()
		w.WriteHeader(http.StatusAccepted)
	})
}
