package stackstorm

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/mux"
	"github.com/prometheus/alertmanager/template"

	"faythe/config"
	"faythe/utils"
)

// TriggerSt2RuleAM gets Request from Prometheus Alertmanager then
// create new request(s). The new request's body will be
// generated based on Alertmanager's request body.
// St2-Api-Key is added to New request's header. New request will
// be forwarded to Stackstorm host using Golang http client.
func TriggerSt2RuleAM() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Declare
		var (
			data template.Data
			wg   sync.WaitGroup
		)

		defer r.Body.Close()
		// Init logger if not initilized yet
		if logger == nil {
			logger = utils.NewFlogger(&once, "stackstorm.log")
		}
		vars := mux.Vars(r)
		confs := config.Get().StackStormConfigs
		conf, ok := confs[vars["st-name"]]
		if !ok {
			supported := make([]string, len(confs))
			for k := range confs {
				supported = append(supported, k)
			}

			err := UnknownStackStormError{
				correct: supported,
				wrong:   vars["st-name"],
			}
			logger.Println(err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Get alerts
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		utils.UpdateExistingAlerts(existingAlerts, &data, logger)
		firingAlerts := data.Alerts.Firing()

		httpClient := newHTTPClient(conf.Scheme)
		computes := make(map[string]bool)
		for _, alert := range firingAlerts {
			// Generate a simple fingerprint aka signature
			// that represents for Alert.
			av := append(alert.Labels.Values(), alert.StartsAt.String())
			fingerprint := utils.Hash(strings.Join(av, "_"))

			// Find hostname by ip address
			hostname, err := utils.LookupAddr(alert.Labels["instance"])
			if err != nil {
				logger.Printf("Get hostname from addr failed because %s.", err.Error())
				continue
			}

			// Check this alert was already received
			_, ok1 := existingAlerts.Get(fingerprint)
			_, ok2 := computes[hostname]
			if ok1 || ok2 {
				logger.Printf("Ignore alert %s/%s from host %s because Faythe already received another alert from this host.",
					alert.Labels["alertname"],
					alert.Labels["instance"],
					hostname)
				// Force add this alert to map(s)
				existingAlerts.Set(fingerprint, true) // Actually, it can be whatever type.
				computes[hostname] = true             // Actually, it can be whatever type.
				continue
			}

			computes[hostname] = true
			existingAlerts.Set(fingerprint, true)
			alert.Labels["compute"] = hostname
			logger.Printf("Processing alert %s from host %s", alert.Labels["alertname"], hostname)
			body, _ := json.Marshal(alert)
			ruler := St2Ruler{
				Rule:       vars["st-rule"],
				Conf:       conf,
				Body:       body,
				HTTPClient: httpClient,
				Req:        r,
				WaitGroup:  &wg,
				Logger:     logger,
			}
			wg.Add(1)
			go ruler.forward()
		}

		defer httpClient.CloseIdleConnections()
		w.WriteHeader(http.StatusAccepted)
	})
}
