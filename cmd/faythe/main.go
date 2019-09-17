// Copyright (c) 2019 Kien Nguyen-Tuan <kiennt2609@gmail.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/prometheus/common/promlog"
	logflag "github.com/prometheus/common/promlog/flag"
	etcdv3 "go.etcd.io/etcd/clientv3"
	etcdv3config "go.etcd.io/etcd/clientv3/yaml"
	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.com/ntk148v/faythe/api"
	"github.com/ntk148v/faythe/middleware"
	"github.com/ntk148v/faythe/pkg/metrics"
)

func main() {
	cfg := struct {
		configFile     string
		listenAddress  string
		url            string
		externalURL    *url.URL
		logConfig      promlog.Config
		etcdConfigFile string
	}{
		logConfig: promlog.Config{},
	}

	a := kingpin.New(filepath.Base(os.Args[0]), "The Faythe server")
	a.HelpFlag.Short('h')
	a.Flag("config.file", "Faythe configuration file path.").
		Default("faythe.yml").StringVar(&cfg.configFile)
	a.Flag("listen-address", "Address to listen on for API.").
		Default("0.0.0.0:8600").StringVar(&cfg.listenAddress)
	a.Flag("external-url",
		"The URL under which Faythe is externally reachable.").
		PlaceHolder("<URL>").StringVar(&cfg.url)
	// etcdv3 flags - To minize effort, pass just a config file path.
	a.Flag("etcd-config.file", "Etcd configuration file path.").
		Default("etcd.yml").StringVar(&cfg.etcdConfigFile)

	logflag.AddFlags(a, &cfg.logConfig)
	_, err := a.Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, errors.Wrapf(err, "Error parsing commandline arguments"))
		a.Usage(os.Args[1:])
		os.Exit(2)
	}

	logger := promlog.New(&cfg.logConfig)
	cfg.externalURL, err = computeExternalURL(cfg.url, cfg.listenAddress)
	level.Info(logger).Log("msg", "Staring Faythe")

	// Init etcdclientv3 with the given config file.
	etcdConf, err := etcdv3config.NewConfig(cfg.etcdConfigFile)
	if err != nil {
		level.Error(logger).Log("err", errors.Wrapf(err, "Error parsing Etcd config file."))
		os.Exit(2)
	}

	etcdCli, err := etcdv3.New(etcdv3.Config(*etcdConf))

	if err != nil {
		level.Error(logger).Log("err", errors.Wrapf(err, "Error instantiating Etcd client."))
		os.Exit(2)
	}

	defer etcdCli.Close()

	var (
		metricsManager = metrics.NewManager(log.With(logger, "component", "metric backend manager"))
		middleware     = middleware.New(log.With(logger, "transport middleware"))
		api            = api.New(log.With(logger, "component", "api"), etcdCli)
		mux            = mux.NewRouter()
	)

	// To ignore error
	fmt.Println(metricsManager)

	api.Register(mux)
	mux.Use(middleware.Logging)
	srv := http.Server{Addr: cfg.listenAddress, Handler: mux}
	srvc := make(chan struct{})

	go func() {
		level.Info(logger).Log("msg", "Listening", "address", cfg.listenAddress)
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			level.Error(logger).Log("msg", "Listen error", "err", err)
			close(srvc)
		}
		defer func() {
			if err := srv.Close(); err != nil {
				level.Error(logger).Log("msg", "Error on closing the server", "err", err)
			}
		}()
	}()

	var (
		hup      = make(chan os.Signal, 1)
		hupReady = make(chan bool)
		term     = make(chan os.Signal, 1)
	)
	signal.Notify(hup, syscall.SIGHUP)
	signal.Notify(term, os.Interrupt, syscall.SIGTERM)

	// Wait for reload or termination signals.
	close(hupReady) // Unblock SIGHUP handler.

	for {
		select {
		case <-term:
			level.Info(logger).Log("msg", "Received SIGTERM, exiting gracefully...")
			os.Exit(0)
		case <-srvc:
			os.Exit(1)
		}
	}
}

// A clone of Prometheus computeExternalURL, because it is a internal function:
// https://github.com/prometheus/prometheus/blob/master/cmd/prometheus/main.go#L791
func computeExternalURL(u, listenAddr string) (*url.URL, error) {
	if u == "" {
		hostname, err := os.Hostname()
		if err != nil {
			return nil, err
		}
		_, port, err := net.SplitHostPort(listenAddr)
		if err != nil {
			return nil, err
		}
		u = fmt.Sprintf("http://%s:%s/", hostname, port)
	}

	// starts or ends with quote
	if strings.HasPrefix(u, "\"") || strings.HasPrefix(u, "'") ||
		strings.HasSuffix(u, "\"") || strings.HasSuffix(u, "'") {
		return nil, errors.New("URL must not begin or end with quotes")
	}

	eu, err := url.Parse(u)
	if err != nil {
		return nil, err
	}

	ppref := strings.TrimRight(eu.Path, "/")
	if ppref != "" && !strings.HasPrefix(ppref, "/") {
		ppref = "/" + ppref
	}
	eu.Path = ppref

	return eu, nil
}
