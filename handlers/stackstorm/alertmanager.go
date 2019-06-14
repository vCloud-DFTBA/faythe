package stackstorm

import (
	"crypto/tls"
	"encoding/json"
	"faythe/utils"
	"net/http"
	"strings"
	"time"

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
		httpClient := http.Client{
			Transport: tr,
			Timeout:   30 * time.Second,
		}
		frChan := make(chan forwardResult, 0)
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
			_, ok1 := existedAlerts[fingerprint]
			_, ok2 := computes[hostname]
			if ok1 || ok2 {
				logger.Printf("Alert %s from host %s was received, ignore it.", alert.Labels["alertname"], hostname)
				// Force add this alert to map(s)
				existedAlerts[fingerprint] = true // Actually, it can be whatever type.
				computes[hostname] = true         // Actually, it can be whatever type.
				continue
			}

			computes[hostname] = true // Actually, it can be whatever type.
			alert.Labels["compute"] = hostname
			logger.Printf("Processing alert %s from host %s", alert.Labels["alertname"], hostname)
			body, err := json.Marshal(alert)
			if err != nil {
				logger.Printf("Json marshal Alert %s failed because %s.", alert.GeneratorURL, err.Error())
				continue
			}
			go forwardReq(frChan, r, url, apiKey, body, &httpClient)
			existedAlerts[fingerprint] = true // Actually, it can be whatever type.
		}

		for i := 0; i < len(firingAlerts); i++ {
			frs := <-frChan
			var bodymap template.Alert
			_ = json.Unmarshal(frs.body, &bodymap)
			if frs.err != nil {
				logger.Printf("Sent request from %s failed because %s.", bodymap.Labels["compute"], frs.err)
			} else {
				logger.Printf("Sent request from %s successfully.", bodymap.Labels["compute"])
			}
		}
		defer httpClient.CloseIdleConnections()
		w.WriteHeader(http.StatusAccepted)
	})
}
