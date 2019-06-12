package stackstorm

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"faythe/utils"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/prometheus/alertmanager/template"
	"github.com/spf13/viper"
)

var (
	data          template.Data
	existedAlerts map[string]bool
)

func updateExistedAlerts(data *template.Data) {
	if existedAlerts == nil {
		existedAlerts = make(map[string]bool)
	}

	resolvedAlerts := data.Alerts.Resolved()
	for _, alert := range resolvedAlerts {
		// Generate a simple fingerprint aka signature
		// that represents for Alert.
		av := append(alert.Labels.Values(), alert.StartsAt.String())
		fingerprint := utils.Hash(strings.Join(av, "_"))
		// Remove Alert if it is already resolved.
		if _, ok := existedAlerts[fingerprint]; ok {
			delete(existedAlerts, fingerprint)
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
		host := utils.Getenv("STACKSTORM_HOST", viper.GetString("stackstorm.host"))
		apiKey := utils.Getenv("STACKSTORM_API_KEY", viper.GetString("stackstorm.apiKey"))
		// TODO(kiennt): Might get ApiKey directly from Stackstorm instead of configure it.
		if host == "" || apiKey == "" {
			logger.Println("Stackstorm host or apikey is missing, please configure these configurations with env or config file.")
			http.Error(w, "Stackstorm host or apikey is missing", http.StatusInternalServerError)
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
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		updateExistedAlerts(&data)
		firingAlerts := data.Alerts.Firing()

		// Create a httpclient with disabled security checks
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		httpClient := http.Client{Transport: tr}
		frChan := make(chan forwardResult, 0)
		computes := make(map[string]bool)
		for _, alert := range firingAlerts {
			// Generate a simple fingerprint aka signature
			// that represents for Alert.
			av := append(alert.Labels.Values(), alert.StartsAt.String())
			fingerprint := utils.Hash(strings.Join(av, "_"))
			// Check this alert was already received
			if _, ok := existedAlerts[fingerprint]; ok {
				logger.Printf("Alert %s from host %s was received, ignore it.", alert.Labels["alertname"], alert.Labels["instance"])
				continue
			}
			hostname, err := utils.LookupAddr(alert.Labels["instance"])
			if err != nil {
				logger.Printf("Get hostname from addr failed because %s.", err.Error())
				continue
			}

			// Deduplicate alert from the same host
			if _, ok := computes[hostname]; ok {
				logger.Printf("Alert %s from host %s was received, ignore it.", alert.Labels["alertname"], alert.Labels["instance"])
				continue
			}
			computes[hostname] = true // Actually, it can be whatever type.
			alert.Labels["compute"] = hostname
			body, err := json.Marshal(alert)
			if err != nil {
				logger.Printf("Json marshal Alert %s failed because %s.", alert.GeneratorURL, err.Error())
				continue
			}
			go forwardReq(frChan, r, url, apiKey, bytes.NewBuffer(body), &httpClient)
			existedAlerts[fingerprint] = true // Actually, it can be whatever type.
		}

		for i := 0; i < len(firingAlerts); i++ {
			frs := <-frChan
			if frs.err != nil {
				logger.Printf("Sent request %s failed because %s.", string(frs.reqDump), frs.err)
			} else {
				logger.Printf("Sent request %s successfully.", string(frs.reqDump))
			}
		}
		w.WriteHeader(http.StatusAccepted)
	})
}
