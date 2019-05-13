package stackstorm

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"faythe/utils"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/prometheus/alertmanager/template"
	"github.com/spf13/viper"
)

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
		var data template.Data
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		alerts := data.Alerts.Firing()

		// Create a httpclient with disabled security checks
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		httpClient := http.Client{Transport: tr}
		frChan := make(chan forwardResult, 0)
		for _, alert := range alerts {
			hostname, err := utils.LookupAddr(alert.Labels["instance"])
			if err != nil {
				logger.Printf("Get hostname from addr failed because %s", err.Error())
				continue
			}
			alert.Labels["compute"] = hostname
			body, err := json.Marshal(alert)
			if err != nil {
				logger.Printf("Json marshal Alert %s failed because %s.", alert.GeneratorURL, err.Error())
				continue
			}
			go forwardReq(frChan, r, url, apiKey, bytes.NewBuffer(body), &httpClient)
		}

		for i := 0; i < len(alerts); i++ {
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
