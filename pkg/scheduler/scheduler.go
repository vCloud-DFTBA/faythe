package scheduler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/vCloud-DFTBA/faythe/pkg/alert"
	"github.com/vCloud-DFTBA/faythe/pkg/cloud/store/openstack"
	"github.com/vCloud-DFTBA/faythe/pkg/common"
	"github.com/vCloud-DFTBA/faythe/pkg/model"
)

// Scheduler does execution time and executes scale actions.
type Scheduler struct {
	model.Scheduler
	logger  log.Logger
	mtx     sync.RWMutex
	state   model.State
	httpCli *http.Client
}

func newScheduler(l log.Logger, data []byte) *Scheduler {
	s := &Scheduler{
		logger:  l,
		httpCli: common.NewHTTPClient(),
	}
	_ = json.Unmarshal(data, s)
	// Force validate for backward compatible
	_ = s.Validate()
	s.state = model.StateActive
	return s
}

// Stop declared to meet Worker interface
func (s *Scheduler) Stop() {}

// Do executes actions
func (s *Scheduler) Do() {
	var wg sync.WaitGroup
	store := openstack.Get()
	os, ok1 := store.Get(s.CloudID)
	if !ok1 {
		level.Error(s.logger).Log("msg",
			fmt.Sprintf("cannot find cloud key %s in store", s.CloudID))
		return
	}

	for _, a := range s.Actions {
		switch at := a.(type) {
		case *model.ActionHTTP:
			wg.Add(1)
			var msg []interface{}
			go func(a *model.ActionHTTP) {
				defer wg.Done()
				if a.CloudAuthToken && os.Provider == model.OpenStackType {
					// If HTTP uses cloud auth token, let's get it from Cloud base client.
					// Only OpenStack provider is supported at this time.
					baseCli, _ := os.BaseClient()
					if token, ok := baseCli.AuthenticatedHeaders()["X-Auth-Token"]; ok {
						if a.Header == nil {
							a.Header = make(map[string]string)
						}
						a.Header["X-Auth-Token"] = token
					}
				}
				if err := alert.SendHTTP(s.httpCli, a); err != nil {
					msg = common.CnvSliceStrToSliceInf(append([]string{
						"msg", "Execute action failed",
						"err", err.Error()},
						at.InfoLog()...))
					level.Error(s.logger).Log(msg...)
					return
				}

				msg = common.CnvSliceStrToSliceInf(append([]string{"msg", "Execute action success"}, at.InfoLog()...))
				level.Info(s.logger).Log(msg...)
			}(at)
		}
	}

	// Wait until all actions were performed
	wg.Wait()
}
