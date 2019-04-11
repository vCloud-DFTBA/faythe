package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"
)

// Log represents a global Logger.
var Log *log.Logger

func main() {
	var listenAddr string
	flag.StringVar(&listenAddr, "listen-addr", ":8600", "server listen address")
	flag.Parse()

	// Create a logger, router and server
	Log = log.New(os.Stdout, "http: ", log.LstdFlags)
	router := newRouter()
	server := newServer(
		listenAddr,
		(middlewares{tracing(func() string { return fmt.Sprintf("%d", time.Now().UnixNano()) }), logging(Log)}).apply(router),
		Log,
	)

	// run our server
	if err := server.run(); err != nil {
		Log.Fatal(err)
	}
}
