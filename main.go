package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync/atomic"
	"syscall"
	"time"

	mw ".github.com/ntk148v/cloudhotpot-middleware/middlewares"
	"github.com/ntk148v/cloudhotpot-middleware/handlers/stackstorm"
)

var (
	listenAddr string
	healthy    int64
)

func main() {
	flag.StringVar(&listenAddr, "listen-addr", ":5000", "server listen address")
	flag.Parse()

	mw.Logger = log.New(os.Stdout, "http: ", log.LstdFlags)
	mw.Logger.Println("Simple Go Server")
	mw.Logger.Println("Server is starting...")

	router := http.NewServeMux()
	// routing
	router.HandleFunc("/", index)
	router.HandleFunc("/healthz", healthz)
	router.HandleFunc("/stackstorm", stackstorm.TriggerSt2Rule)
	// Add more routes here
	// 1. Create a new handler, for example: handlers/new/new.go
	// 2. import "./handlers/new"
	// 3. router.HandleFunc("/new", new.NewHandle)

	nextRequestID := func() string {
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}

	server := &http.Server{
		Addr:         listenAddr,
		Handler:      mw.Tracing(nextRequestID)(mw.Logging(mw.Logger)(router)),
		ErrorLog:     mw.Logger,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}
	ctx := shutdown(context.Background(), server)
	atomic.StoreInt64(&healthy, time.Now().UnixNano())

	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		mw.Logger.Fatalf("Could not listen on %q: %s\n", listenAddr, err)
	}
	<-ctx.Done()
	mw.Logger.Printf("Server stopped\n")
}

func shutdown(ctx context.Context, server *http.Server) context.Context {
	ctx, done := context.WithCancel(ctx)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		defer done()

		<-quit
		signal.Stop(quit)
		close(quit)

		atomic.StoreInt64(&healthy, 0)
		server.ErrorLog.Printf("Server is shutting down...\n")

		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		server.SetKeepAlivesEnabled(false)
		if err := server.Shutdown(ctx); err != nil {
			server.ErrorLog.Fatalf("Could not gracefully shutdown the server: %s\n", err)
		}
	}()

	return ctx
}

func index(w http.ResponseWriter, req *http.Request) {
	if req.URL.Path != "/" {
		http.NotFound(w, req)
	}
	fmt.Fprintf(w, "Hello stranger! Welcome to Simple Go Server\n")
}

func healthz(w http.ResponseWriter, req *http.Request) {
	if h := atomic.LoadInt64(&healthy); h == 0 {
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		fmt.Fprintf(w, "uptime: %s\n", time.Since(time.Unix(0, h)))
	}
}
