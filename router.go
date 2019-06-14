package main

import (
	"faythe/handlers/basic"
	"faythe/handlers/openstack"
	"faythe/handlers/stackstorm"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/spf13/viper"
)

func newRouter(logger *log.Logger) *mux.Router {
	router := mux.NewRouter()

	var wg sync.WaitGroup
	wg.Add(1)
	go openstack.UpdateStacksOutputs(&wg)

	// Create nextRequestID
	nextRequestID := func() string {
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}

	// Init middleware
	mw := &middleware{
		logger:        logger,
		nextRequestID: nextRequestID,
	}

	// Routing
	router.Handle("/", basic.Index()).Methods("GET")
	router.Handle("/healthz", basic.Healthz(&healthy)).Methods("GET")
	router.Handle("/stackstorm/{st-rule}", stackstorm.TriggerSt2Rule()).
		Methods("POST").
		Host(viper.GetString("restrictedDomain"))
	router.Handle("/stackstorm/alertmanager/{st-rule}", stackstorm.TriggerSt2RuleAM()).
		Methods("POST").
		Host(viper.GetString("restrictedDomain"))
	router.Handle("/autoscaling", openstack.Autoscaling()).
		Methods("POST").
		Host(viper.GetString("restrictedDomain"))

	// Appends a Middlewarefunc to the chain.
	router.Use(mw.tracing, mw.logging)

	return router
}
