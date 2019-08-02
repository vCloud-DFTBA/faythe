package main

import (
	"log"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/mux"

	"faythe/config"
	"faythe/handlers/basic"
	"faythe/handlers/openstack"
	"faythe/handlers/stackstorm"
)

func newRouter(logger *log.Logger) *mux.Router {
	router := mux.NewRouter()

	// TODO(kiennt): Might be this is not the best way to place the
	// 				 follow. Update later.
	var wg sync.WaitGroup
	wg.Add(1)
	conf := config.Get()
	for _, opsConf := range conf.OpenStackConfigs {
		go openstack.UpdateStacksOutputs(opsConf, &wg)
	}
	// Create nextRequestID
	nextRequestID := func() string {
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}

	r, _ := regexp.Compile(conf.ServerConfig.RemoteHostPattern)

	// Init middleware
	mw := &middleware{
		logger:        logger,
		nextRequestID: nextRequestID,
		regexp:        r,
	}

	// Routing
	router.Handle("/", basic.Index()).Methods("GET")
	router.Handle("/healthz", basic.Healthz(&healthy)).Methods("GET")
	router.Handle("/stackstorm/{st-host}/{st-rule}", stackstorm.TriggerSt2Rule()).
		Methods("POST")
	router.Handle("/stackstorm/alertmanager/{st-host}/{st-rule}", stackstorm.TriggerSt2RuleAM()).
		Methods("POST")
	router.Handle("/openstack/autoscaling", openstack.Autoscaling()).
		Methods("POST")

	// Appends a Middlewarefunc to the chain.
	router.Use(mw.tracing, mw.logging, mw.restrictingDomain, mw.authenticating)

	return router
}
