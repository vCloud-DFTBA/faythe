package main

import (
	"flag"
	"log"
	"os"
	"strconv"
	"time"
)

// Log represents a global Logger.
var Log *log.Logger

func main() {
	var listenAddr string
	flag.StringVar(&listenAddr, "listen-addr", ":8600", "server listen address")
	flag.Parse()

	// Create nextRequestID
	nextRequestID := func() string {
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}

	// Create a logger, router and server
	Log = log.New(os.Stdout, "http: ", log.LstdFlags)
	router := newRouter()
	server := newServer(
		listenAddr,
		tracing(nextRequestID)(logging(Log)(router)),
		Log,
	)

	// run our server
	if err := server.run(); err != nil {
		Log.Fatal(err)
	}
}
