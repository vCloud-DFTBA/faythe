package stackstorm

import (
	"bytes"
	"crypto/tls"
	"faythe/utils"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
	"github.com/spf13/viper"
)

var (
	logger *utils.Flogger
	once   sync.Once
)

func init() {
	logger = utils.NewFlogger(&once, "stackstorm.go")
}

// TriggerSt2Rule gets Request then create a new request based on it.
// St2-Api-Key is added to New request's header. New request will
// be forwarded to Stackstorm host using Golang http client.
func TriggerSt2Rule() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		host := utils.Getenv("STACKSTORM_HOST", viper.GetString("stackstorm.host"))
		apiKey := utils.Getenv("STACKSTORM_API_KEY", viper.GetString("stackstorm.apiKey"))
		rule := vars["st-rule"]
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			logger.Printf("Stackstorm/TriggerSt2Rule - get request body failed: %s", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		r.Body = ioutil.NopCloser(bytes.NewReader(body))
		url := "https://" + host + "/api/webhooks/" + rule
		proxyReq, err := http.NewRequest(r.Method, url, bytes.NewReader(body))
		if err != nil {
			logger.Printf("Stackstorm/TriggerSt2Rule - create a new request failed: %s", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// Filter some headers, otherwise could just use a shallow copy proxyReq.Header = r.Header
		proxyReq.Header = make(http.Header)
		for h, val := range r.Header {
			proxyReq.Header[h] = val
		}
		// proxyReq.Header = r.Header
		proxyReq.Header.Add("St2-Api-Key", apiKey)
		// Create a httpclient with disabled security checks
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		httpClient := http.Client{Transport: tr}
		logger.Printf("Stackstorm/TriggerSt2Rule - send a POST request to %s...\n", url)
		resp, err := httpClient.Do(proxyReq)
		if err != nil {
			logger.Println(err.Error())
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		logger.Printf("Stackstorm/TriggerSt2Rule - send a POST request to Stackstorm host %s successfully!\n", host)
		defer resp.Body.Close()
	})
}
