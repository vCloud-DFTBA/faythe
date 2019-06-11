package stackstorm

import (
	"faythe/utils"
	"io"
	"net/http"
	"sync"

	"github.com/pkg/errors"
)

type forwardResult struct {
	reqDump []byte
	err     error
}

var (
	logger *utils.Flogger
	once   sync.Once
	host   string
	apiKey string
)

func init() {
	logger = utils.NewFlogger(&once, "stackstorm.log")
}

func forwardReq(fResults chan<- forwardResult, r *http.Request, url, apiKey string, body io.Reader, httpClient *http.Client) {
	proxyReq, err := http.NewRequest(r.Method, url, body)
	if err != nil {
		fResults <- forwardResult{[]byte(url), errors.Wrap(err, "create a new request failed")}
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
		fResults <- forwardResult{[]byte(url), errors.Wrap(err, "send a POST request failed")}
		return
	}
	fResults <- forwardResult{[]byte(url), nil}
	defer resp.Body.Close()
	return
}
