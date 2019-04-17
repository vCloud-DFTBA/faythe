package main

import (
	"faythe/handlers/basic"
	"faythe/handlers/openstack"
	"faythe/handlers/stackstorm"
	"sync"

	"github.com/gorilla/mux"
)

func newRouter() *mux.Router {
	router := mux.NewRouter()

	var wg sync.WaitGroup
	wg.Add(1)
	go openstack.UpdateStacksOutputs(Log, &wg)

	// routing
	router.Handle("/", basic.Index())
	router.Handle("/healthz", basic.Healthz(&healthy))
	router.Handle("/stackstorm/{st-rule}", stackstorm.TriggerSt2Rule(Log))
	router.Handle("/autoscaling", openstack.Autoscaling(Log))

	return router
}
