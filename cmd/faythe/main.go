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
	"runtime"
	"strings"
	"syscall"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/gorilla/mux"
	"github.com/jinzhu/copier"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/promlog"
	logflag "github.com/prometheus/common/promlog/flag"
	"github.com/prometheus/common/version"
	etcdv3 "go.etcd.io/etcd/clientv3"
	"gopkg.in/alecthomas/kingpin.v2"

	// Add version information
	"github.com/vCloud-DFTBA/faythe/api"
	"github.com/vCloud-DFTBA/faythe/config"
	"github.com/vCloud-DFTBA/faythe/pkg/autohealer"
	"github.com/vCloud-DFTBA/faythe/pkg/autoscaler"
	_ "github.com/vCloud-DFTBA/faythe/pkg/build"
	"github.com/vCloud-DFTBA/faythe/pkg/cloud/store/openstack"
	"github.com/vCloud-DFTBA/faythe/pkg/cluster"
	"github.com/vCloud-DFTBA/faythe/pkg/common"
	"github.com/vCloud-DFTBA/faythe/pkg/history"
)

func init() {
	prometheus.MustRegister(version.NewCollector("faythe"))
}

func main() {
	if os.Getenv("DEBUG") != "" {
		runtime.SetBlockProfileRate(20)
		runtime.SetMutexProfileFraction(20)
	}

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
	a.Version(version.Print("faythe"))
	a.HelpFlag.Short('h')
	a.Flag("config.file", "Faythe configuration file path.").
		Default("/etc/faythe/config.yml").StringVar(&cfg.configFile)
	a.Flag("listen-address", "Address to listen on for API.").
		Default("0.0.0.0:8600").StringVar(&cfg.listenAddress)
	a.Flag("external-url",
		"The URL under which Faythe is externally reachable.").
		PlaceHolder("<URL>").StringVar(&cfg.url)
	a.Flag("cluster-id",
		"The unique ID of the cluster, leave it empty to initialize a new cluster. This will be the root prefix for all Faythe keys stored in Etcd.").
		StringVar(&cfg.clusterID)

	logflag.AddFlags(a, &cfg.logConfig)
	_, err := a.Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, errors.Wrapf(err, "Error parsing commandline arguments"))
		a.Usage(os.Args[1:])
		os.Exit(2)
	}

	logger := promlog.New(&cfg.logConfig)
	cfg.externalURL, _ = computeExternalURL(cfg.url, cfg.listenAddress)
	level.Info(logger).Log("msg", "Staring Faythe", "version", version.Info())
	level.Info(logger).Log("build_context", version.BuildContext())
	rtStats := common.RuntimeStats()
	level.Debug(logger).Log("msg", "golang runtime stats")
	for k, v := range rtStats {
		level.Debug(logger).Log(k, v)
	}

	var (
		etcdcfg   = etcdv3.Config{}
		etcdcli   = &common.Etcd{}
		router    = mux.NewRouter()
		fapi      = &api.API{}
		fas       = &autoscaler.Manager{}
		cls       = &cluster.Cluster{}
		clusterID string
	)
	// Load configurations from file
	err = config.Set(cfg.configFile, log.With(logger, "component", "config manager"))
	if err != nil {
		level.Error(logger).Log("msg", "Error loading configuration file", "err", err)
		os.Exit(2)
	}

	config.WatchConfig()

	// Init Etcdv3 client
	_ = copier.Copy(&etcdcfg, config.Get().EtcdConfig)
	// clusterID is the id of cluster. It could be a random string
	// or a user-defined string.
	if cfg.clusterID == "" {
		clusterID = common.RandToken()
		level.Info(logger).Log("msg", "A new cluster is starting...")
	} else {
		clusterID = strings.Trim(cfg.clusterID, "/")
		level.Info(logger).Log("msg", "A node is joining to existing cluster...")
	}
	etcdcli, err = common.NewEtcd(log.With(logger, "component", "etcd wrapper"),
		clusterID, etcdcfg)

	if err != nil {
		level.Error(logger).Log("msg", errors.Wrapf(err, "Error instantiating Etcd V3 client."))
		os.Exit(2)
	}

	// Init cluster
	cls, err = cluster.New(clusterID, cfg.listenAddress,
		log.With(logger, "component", "cluster"), etcdcli)
	if err != nil {
		level.Error(logger).Log("msg", errors.Wrap(err, "Error initializing Cluster"))
		os.Exit(2)
	}
	reloadc := make(chan struct{})
	go cls.Run(reloadc)

	fapi, err = api.New(log.With(logger, "component", "api"), etcdcli)
	if err != nil {
		level.Error(logger).Log("msg", errors.Wrap(err, "Error initializing API"))
		os.Exit(2)
	}
	router.StrictSlash(true)
	fapi.Register(router)

	// Init autoscaler manager
	fas = autoscaler.NewManager(log.With(logger, "component", "autoscaler manager"), etcdcli, cls)
	go fas.Run()

	// Init autohealer manager
	fah := autohealer.NewManager(log.With(logger, "component", "autohealer manager"), etcdcli, cls)
	go fah.Run()

	// Init Cloud store
	openstack.InitStore()
	if err := openstack.Load(etcdcli); err != nil {
		level.Error(logger).Log("msg", "error while loading cloud information", "err", err)
	}

	// Init history engine for saving action histories
	history.Init(log.With(logger, "component", "action history"), etcdcli)

	stopc := make(chan struct{})
	go etcdcli.Run(stopc)
	stopFunc := func() {
		fas.Stop()
		fah.Stop()
		cls.Stop()
		fah.Stop()
		etcdcli.Close()
	}

	// Force clean-up when shutdown.
	defer stopFunc()

	// Init HTTP server
	srv := http.Server{Addr: cfg.listenAddress, Handler: router}
	srvc := make(chan struct{})

	go func() {
		for {
			select {
			case <-reloadc:
				fas.Reload()
				fah.Reload()
			case <-stopc:
				stopFunc()
				level.Info(logger).Log("msg", "Faythe is stopping, bye bye!")
				_ = syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
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

	select {
	case <-term:
		level.Info(logger).Log("msg", "Received SIGTERM, exiting gracefully...")
		return
	case <-srvc:
		return
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
