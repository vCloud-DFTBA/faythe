package main

import (
	"faythe/config"
	"flag"
	"log"
	"os"
)

const (
	defaultConfigFilePath = "/etc/faythe/config.yml"
	configFilePathUsage   = "config file path."
)

var (
	// Log represents a global Logger.
	Log            *log.Logger
	configFilePath string
	listenAddr     string
)

func init() {
	flag.StringVar(&configFilePath, "conf", defaultConfigFilePath, configFilePathUsage)
	flag.StringVar(&listenAddr, "listen-addr", ":8600", "server listen address.")
	flag.Parse()
	// Log = log.New(os.Stdout, "http: ", log.LstdFlags)
	config.LoadFile(configFilePath)
	config.WatchConfig()
}

func main() {
	// Create a logger, router and server
	Log = log.New(os.Stdout, "http: ", log.LstdFlags)
	router := newRouter(Log)
	server := newServer(
		listenAddr,
		router,
		Log,
	)

	// run our server
	if err := server.run(); err != nil {
		Log.Fatal(err)
	}
}
