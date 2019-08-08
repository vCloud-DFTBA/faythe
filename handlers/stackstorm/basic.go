package stackstorm

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/gorilla/mux"

	"faythe/config"
	"faythe/utils"
)

// TriggerSt2Rule gets Request then create a new request based on it.
// The new request body is be kept as the origin request.
// St2-Api-Key is added to New request's header. New request will
// be forwarded to Stackstorm host using Golang http client.
func TriggerSt2Rule() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if logger == nil {
			logger = utils.NewFlogger(&once, "stackstorm.log")
		}

		vars := mux.Vars(r)
		conf, ok := config.Get().StackStormConfigs[vars["st-host"]]
		if !ok {
			msg := fmt.Sprintf("Cannot find the configuration of host %s, please check it again", vars["st-host"])
			logger.Println(msg)
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		host := utils.Getenv("STACKSTORM_HOST", conf.Host)
		apiKey := utils.Getenv("STACKSTORM_API_KEY", conf.APIKey)
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
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			logger.Printf("Get request body failed: %s", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		r.Body = ioutil.NopCloser(bytes.NewReader(body))
		url := "https://" + host + "/api/webhooks/" + rule
		// Create a httpclient with disabled security checks
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		httpClient := http.Client{Transport: tr}
		var wg sync.WaitGroup
		wg.Add(1)
		forwardReq(r, url, apiKey, body, &httpClient, &wg)
		w.WriteHeader(http.StatusAccepted)
	})
}
