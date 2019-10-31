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

	"github.com/ntk148v/faythe/pkg/cluster"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/gorilla/mux"
	"github.com/jinzhu/copier"
	"github.com/pkg/errors"
	"github.com/prometheus/common/promlog"
	logflag "github.com/prometheus/common/promlog/flag"
	etcdv3 "go.etcd.io/etcd/clientv3"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/vCloud-DFTBA/faythe/api"
	"github.com/vCloud-DFTBA/faythe/config"
	"github.com/vCloud-DFTBA/faythe/middleware"
	"github.com/vCloud-DFTBA/faythe/pkg/autoscaler"
	"github.com/vCloud-DFTBA/faythe/pkg/cluster"
)

func main() {
	cfg := struct {
		configFile    string
		listenAddress string
		url           string
		externalURL   *url.URL
		logConfig     promlog.Config
		clusterID     string
	}{
		logConfig: promlog.Config{},
	}

	a := kingpin.New(filepath.Base(os.Args[0]), "The Faythe server")
	a.HelpFlag.Short('h')
	a.Flag("config.file", "Faythe configuration file path.").
		Default("/etc/faythe/config.yml").StringVar(&cfg.configFile)
	a.Flag("listen-address", "Address to listen on for API.").
		Default("0.0.0.0:8600").StringVar(&cfg.listenAddress)
	a.Flag("external-url",
		"The URL under which Faythe is externally reachable.").
		PlaceHolder("<URL>").StringVar(&cfg.url)
	a.Flag("cluster-id", "The unique ID of the cluster, leave it empty to initialize a new cluster.").
		StringVar(&cfg.clusterID)

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

	var (
		etcdConf = etcdv3.Config{}
		etcdCli  = &etcdv3.Client{}
		router   = mux.NewRouter()
		fmw      = &middleware.Middleware{}
		fapi     = &api.API{}
		fas      = &autoscaler.Manager{}
		cls      = &cluster.Cluster{}
	)
	// Load configurations from file
	err = config.Set(cfg.configFile, log.With(logger, "component", "config manager"))
	if err != nil {
		level.Error(logger).Log("msg", "Error loading configuration file", "err", err)
		os.Exit(2)
	}

	config.WatchConfig()

	// Init Etcdv3 client
	copier.Copy(&etcdConf, config.Get().EtcdConfig)
	etcdCli, err = etcdv3.New(etcdConf)

	if err != nil {
		level.Error(logger).Log("err", errors.Wrapf(err, "Error instantiating Etcd client."))
		os.Exit(2)
	}

	// Init cluster
	cls, err = cluster.New(cfg.clusterID, cfg.listenAddress,
		log.With(logger, "component", "cluster"), etcdCli)
	if err != nil {
		level.Error(logger).Log("err", errors.Wrap(err, "Error initializing Cluster"))
		os.Exit(2)
	}
	reloadCh := make(chan bool)
	go cls.Run(reloadCh)

	fmw = middleware.New(log.With(logger, "component", "transport middleware"))

	fapi = api.New(log.With(logger, "component", "api"), etcdCli)
	fapi.Register(router)
	router.Use(fmw.Logging, fmw.RestrictDomain, fmw.Authenticate)

	fas = autoscaler.NewManager(log.With(logger, "component", "autoscale manager"), etcdCli)
	go fas.Run()
	defer func() {
		fas.Stop()
		cls.Stop()
		etcdCli.Close()
		level.Info(logger).Log("msg", "Faythe is stopped, bye bye!")
	}()

	// Init HTTP server
	srv := http.Server{Addr: cfg.listenAddress, Handler: router}
	srvc := make(chan struct{})

	go func() {
		for {
			select {
			case <-reloadCh:
				fas.Reload()
			}
		}
	}()

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
			return
		case <-srvc:
			return
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
