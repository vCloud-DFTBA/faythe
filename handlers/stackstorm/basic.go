package stackstorm

import (
	"bytes"
	"crypto/tls"
	"faythe/utils"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/spf13/viper"
)

// TriggerSt2Rule gets Request then create a new request based on it.
// The new request body is be kept as the origin request.
// St2-Api-Key is added to New request's header. New request will
// be forwarded to Stackstorm host using Golang http client.
func TriggerSt2Rule() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		frChan := make(chan forwardResult, 1)
		forwardReq(frChan, r, url, apiKey, body, &httpClient)
		frs := <-frChan
		if frs.err != nil {
			logger.Printf("Sent request %s failed because %s.", string(frs.body), frs.err)
		} else {
			logger.Printf("Sent request %s successfully.", string(frs.body))
		}
		w.WriteHeader(http.StatusAccepted)
	})
}
