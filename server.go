package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"
)

var healthy int64

// Server implements HTTP server
type Server struct {
	server *http.Server
}

// newServer creates a new HTTP server
func newServer(listenAddr string, h http.Handler, l *log.Logger) *Server {
	return &Server{
		server: &http.Server{
			Addr:           listenAddr,
			Handler:        h, // pass in mux/router
			ErrorLog:       l,
			ReadTimeout:    5 * time.Second,
			WriteTimeout:   10 * time.Second,
			IdleTimeout:    30 * time.Second,
			MaxHeaderBytes: 1 << 20,
		},
	}
}

// run starts the HTTP server
func (s *Server) run() error {
	// Get hostname
	hostname, err := os.Hostname()
	if err != nil {
		return err
	}

	done := make(chan bool)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		fmt.Println("")
		s.server.ErrorLog.Printf("%s - Shutdown signal received...\n", hostname)
		atomic.StoreInt64(&healthy, 0)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		s.server.SetKeepAlivesEnabled(false)
		if err := s.server.Shutdown(ctx); err != nil {
			s.server.ErrorLog.Fatalf("Could not gracefully shutdown the server: %v\n", err)
		}
		close(done)
	}()

	s.server.ErrorLog.Println("** Cloudhotpot-middle **")
	s.server.ErrorLog.Printf("%s - Starting server on %v", hostname, s.server.Addr)
	atomic.StoreInt64(&healthy, time.Now().UnixNano())

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		s.server.ErrorLog.Fatalf("Could not listen on %s: %v", s.server.Addr, err)
	}

	<-done
	s.server.ErrorLog.Printf("%s - Server gracefully stopped.\n", hostname)
	return nil
}
