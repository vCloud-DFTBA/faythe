package stackstorm

import (
	"bytes"
	"crypto/tls"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

// TriggerSt2Rule gets Request then create a new request based on it.
// St2-Api-Key is added to New request's header. New request will
// be forwarded to Stackstorm host using Golang http client.
func TriggerSt2Rule(logger *log.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			logger.Printf("Stackstorm/TriggerSt2Rule - get request body failed: %s", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		req.Body = ioutil.NopCloser(bytes.NewReader(body))
		url := "https://" + os.Getenv("STACKSTORM_HOST") + "/api/webhooks/" + os.Getenv("STACKSTORM_RULE")
		proxyReq, err := http.NewRequest(req.Method, url, bytes.NewReader(body))
		if err != nil {
			logger.Printf("Stackstorm/TriggerSt2Rule - create a new request failed: %s", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// Filter some headers, otherwise could just use a shallow copy proxyReq.Header = req.Header
		proxyReq.Header = make(http.Header)
		for h, val := range req.Header {
			proxyReq.Header[h] = val
		}
		// proxyReq.Header = req.Header
		proxyReq.Header.Add("St2-Api-Key", os.Getenv("STACKSTORM_API_KEY"))
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
		logger.Printf("Stackstorm/TriggerSt2Rule - send a POST request to Stackstorm host %s successfully!\n", os.Getenv("STACKSTORM_HOST"))
		defer resp.Body.Close()
	})
}
