package main

import (
	"faythe/config"
	"flag"
	"log"
	"os"
	"runtime"
	"strconv"
	"time"
)

const (
	defaultConfigFilePath = "./etc/"
	configFilePathUsage   = "config file directory (eg. '/etc/faythe/'). Config file must be named 'config.yml'."
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
	config.Load(configFilePath)
}

func main() {
	if os.Getenv("GOMAXPROCS") == "" {
		runtime.GOMAXPROCS(runtime.NumCPU())
	}

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
