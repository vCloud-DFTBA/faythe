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
	"os"
	"path/filepath"

	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	log "github.com/prometheus/common/promlog"
	logflag "github.com/prometheus/common/promlog/flag"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

func main() {
	cfg := struct {
		configFile string
		logConfig  log.Config
	}{
		logConfig: log.Config{},
	}

	a := kingpin.New(filepath.Base(os.Args[0]), "The Faythe server")
	a.HelpFlag.Short('h')
	a.Flag("config.file", "Faythe configuration file path.").
		Default("faythe.yml").StringVar(&cfg.configFile)

	logflag.AddFlags(a, &cfg.logConfig)
	_, err := a.Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, errors.Wrapf(err, "Error parsing commandline arguments"))
		a.Usage(os.Args[1:])
		os.Exit(2)
	}

	logger := log.New(&cfg.logConfig)
	level.Info(logger).Log("msg", "Staring Faythe")
}
