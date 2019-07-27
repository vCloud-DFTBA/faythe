package stackstorm

import (
	"bytes"
	"encoding/json"
	"faythe/utils"
	"net/http"
	"sync"

	"github.com/prometheus/alertmanager/template"
)

var (
	logger         *utils.Flogger
	once           sync.Once
	existingAlerts utils.SharedValue
)

func forwardReq(r *http.Request, url, apiKey string, body []byte, httpClient *http.Client, wg *sync.WaitGroup) {
	host := r.RemoteAddr
	var bodymap template.Alert
	err := json.Unmarshal(body, &bodymap)
	if err == nil {
		host = bodymap.Labels["compute"]
	}
	proxyReq, err := http.NewRequest(r.Method, url, bytes.NewBuffer(body))
	if err != nil {
		logger.Printf("Sent request from %s failed because %s.", host, "create a new request failed")
		return
	}
	// Filter some headers, otherwise could just use a shallow copy proxyReq.Header = r.Header
	proxyReq.Header = make(http.Header)
	for h, val := range r.Header {
		proxyReq.Header[h] = val
	}
	// proxyReq.Header = r.Header
	proxyReq.Header.Add("St2-Api-Key", apiKey)
	resp, err := httpClient.Do(proxyReq)
	if err != nil {
		logger.Printf("Sent request from %s failed because %s.", host, "send a POST request failed")
		return
	}
	logger.Printf("Sent request from %s successfully.", host)
	defer resp.Body.Close()
	defer wg.Done()
	return
}
