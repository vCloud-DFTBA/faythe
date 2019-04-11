package main

import (
	"cloudhotpot-middleware/handlers/basic"
	"cloudhotpot-middleware/handlers/openstack"
	"cloudhotpot-middleware/handlers/stackstorm"
	"net/http"
	"sync"
)

func newRouter() *http.ServeMux {
	router := http.NewServeMux()

	var wg sync.WaitGroup
	wg.Add(1)
	go openstack.UpdateStacksOutputs(&Log, &wg)

	// routing
	router.Handle("/", basic.Index())
	router.Handle("/healthz", basic.Healthz(healthy))
	router.Handle("/stackstorm", stackstorm.TriggerSt2Rule(&Log))
	router.Handle("/autoscaling", openstack.Autoscaling(&Log))

	return router
}
