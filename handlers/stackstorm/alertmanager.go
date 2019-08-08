package stackstorm

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/alertmanager/template"

	"faythe/config"
	"faythe/utils"
)

func updateExistingAlerts(data *template.Data) {
	resolvedAlerts := data.Alerts.Resolved()
	for _, alert := range resolvedAlerts {
		// Generate a simple fingerprint aka signature
		// that represents for Alert.
		av := append(alert.Labels.Values(), alert.StartsAt.String())
		fingerprint := utils.Hash(strings.Join(av, "_"))
		// Remove Alert if it is already resolved.
		if _, ok := existingAlerts.Get(fingerprint); ok {
			logger.Printf("Alert %s/%s was resolved, delete it from existing alerts list.",
				alert.Labels["alertname"],
				alert.Labels["instance"])
			existingAlerts.Delete(fingerprint)
		}
	}
}

// TriggerSt2RuleAM gets Request from Prometheus Alertmanager then
// create new request(s). The new request's body will be
// generated based on Alertmanager's request body.
// St2-Api-Key is added to New request's header. New request will
// be forwarded to Stackstorm host using Golang http client.
func TriggerSt2RuleAM() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		vars := mux.Vars(r)
		conf, ok := config.Get().StackStormConfigs[vars["st-host"]]
		if !ok {
			msg := fmt.Sprintf("Cannot find the configuration of host %s, please check it again", vars["st-host"])
			logger.Println(msg)
			http.Error(w, msg, http.StatusBadRequest)
			return
		}
		host := utils.Getenv("STACKSTORM_HOST", conf.Host)
		apiKey := utils.Getenv("STACKSTORM_API_KEY", conf.Host)
		// TODO(kiennt): Might get ApiKey directly from Stackstorm instead of configure it.
		if host == "" || apiKey == "" {
			logger.Println("Stackstorm host or apikey is missing, please configure these configurations with env or config file.")
			return
		}
		rule := vars["st-rule"]
		if rule == "" {
			logger.Println("Stackstorm rule is missing in request query.")
			http.Error(w, "Stackstorm rule is missing in request query", http.StatusBadRequest)
			return
		}
		url := "https://" + host + "/api/webhooks/" + rule
		// Get alerts
		var (
			data template.Data
			wg   sync.WaitGroup
		)
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		updateExistingAlerts(&data)
		firingAlerts := data.Alerts.Firing()

		// Create a httpclient with disabled security checks
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		httpClient := http.Client{
			Transport: tr,
			Timeout:   3 * time.Second,
		}
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
			body, err := json.Marshal(alert)
			if err != nil {
				logger.Printf("Json marshal Alert %s failed because %s.", alert.GeneratorURL, err.Error())
				continue
			}
			wg.Add(1)
			go forwardReq(r, url, apiKey, body, &httpClient, &wg)
		}

		defer httpClient.CloseIdleConnections()
		w.WriteHeader(http.StatusAccepted)
	})
}
