package stackstorm

import (
	"bytes"
	"crypto/tls"
	"io/ioutil"
	"net/http"
	"os"
)

// TriggerSt2Rule gets Request then create a new request based on it.
// St2-Api-Key is added to New request's header. New request will
// be forwarded to Stackstorm host using Golang http client.
func TriggerSt2Rule(w http.ResponseWriter, req *http.Request) {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req.Body = ioutil.NopCloser(bytes.NewReader(body))
	url := "https://" + os.Getenv("STACKSTORM_HOST") + "/api/webhooks/" + os.Getenv("STACKSTORM_RULE")
	proxyReq, err := http.NewRequest(req.Method, url, bytes.NewReader(body))
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
	resp, err := httpClient.Do(proxyReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
}
