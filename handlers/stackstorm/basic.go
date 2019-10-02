package stackstorm

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/gorilla/mux"

	"github.com/ntk148v/faythe/config"
	"github.com/ntk148v/faythe/utils"
)

// TriggerSt2Rule gets Request then create a new request based on it.
// The new request body is be kept as the origin request.
// St2-Api-Key is added to New request's header. New request will
// be forwarded to Stackstorm host using Golang http client.
func TriggerSt2Rule() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var wg sync.WaitGroup
		defer r.Body.Close()
		// Init logger if not initilized yet
		if logger == nil {
			logger = utils.NewFlogger(&once, "stackstorm.log")
		}

		vars := mux.Vars(r)
		confs := config.Get().StackStormConfigs
		conf, ok := confs[vars["st-name"]]
		if !ok {
			supported := make([]string, 0)
			for k := range confs {
				supported = append(supported, k)
			}

			err := UnknownStackStormError{
				correct: supported,
				wrong:   vars["st-name"],
			}
			logger.Println(err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			logger.Printf("Get request body failed: %s", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		r.Body = ioutil.NopCloser(bytes.NewReader(body))
		httpClient := newHTTPClient(conf.Scheme)
		ruler := St2Ruler{
			Rule:       vars["st-rule"],
			Conf:       conf,
			Body:       body,
			HTTPClient: httpClient,
			Req:        r,
			WaitGroup:  &wg,
			Logger:     logger,
		}
		wg.Add(1)
		ruler.forward()
		w.WriteHeader(http.StatusAccepted)
	})
}
