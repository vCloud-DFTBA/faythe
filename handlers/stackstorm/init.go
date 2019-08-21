package stackstorm

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"faythe/config"
	"faythe/utils"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/alertmanager/template"
)

var (
	logger         *utils.Flogger
	once           sync.Once
	existingAlerts utils.SharedValue
)

// St2Ruler processes StackStorm requests.
type St2Ruler struct {
	Rule       string
	Conf       *config.StackStormConfig
	Body       []byte
	HTTPClient http.Client
	Req        *http.Request
	WaitGroup  *sync.WaitGroup
	Logger     *utils.Flogger
}

func (r *St2Ruler) genWebhookURL() string {
	return fmt.Sprintf("%s/%s/api/webhooks/%s", r.Conf.Scheme,
		r.Conf.Host, r.Rule)
}

func (r *St2Ruler) printLog(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a)
	r.Logger.Printf("Stackstorm %s - %s\n",
		r.Conf.Host,
		msg)
}

func (r *St2Ruler) forward() {
	defer r.WaitGroup.Done()
	host := r.Req.RemoteAddr
	var bodymap template.Alert
	err := json.Unmarshal(r.Body, &bodymap)
	if err == nil {
		host = bodymap.Labels["compute"]
	}

	url := r.genWebhookURL()
	nreq, err := http.NewRequest(r.Req.Method, url, bytes.NewBuffer(r.Body))
	if err != nil {
		r.printLog("Send request from %s failed because %s",
			host, err.Error())
		return
	}

	// Filter some headers, otherwise could just use a shallow copy
	// nreq.Header = r.Req.header
	nreq.Header = make(http.Header)
	for h, v := range r.Req.Header {
		nreq.Header[h] = v
	}
	nreq.Header.Add("St2-Api-Key", r.Conf.APIKey)
	resp, err := r.HTTPClient.Do(nreq)
	if err != nil {
		r.printLog("Send request from %s failed because %s",
			host, err.Error())
		return
	}
	r.printLog("Send request from %s success", host)
	defer resp.Body.Close()
}

func newHTTPClient(scheme string) http.Client {
	var client http.Client
	switch scheme {
	case "https":
		// Create a httpclient with disabled security checks
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client = http.Client{
			Transport: tr,
			Timeout:   3 * time.Second,
		}
	case "http":
		client = http.Client{Timeout: 3 * time.Second}
	}
	return client
}
