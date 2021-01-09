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

package prometheus

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	amclient "github.com/prometheus/alertmanager/api/v2/client"
	ammodels "github.com/prometheus/alertmanager/api/v2/models"
	prometheusclient "github.com/prometheus/client_golang/api"
	prometheus "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"

	"github.com/vCloud-DFTBA/faythe/pkg/common"
	fmodel "github.com/vCloud-DFTBA/faythe/pkg/model"
)

// Backend implements a metric backend for Prometheus.
type Backend struct {
	prometheus prometheus.API
	address    string
	logger     log.Logger
}

// New returns a new client for talking to a Prometheus Backend, or an error
func New(logger log.Logger, address, username, password string) (*Backend, error) {
	if logger == nil {
		logger = log.NewNopLogger()
	}

	// Init Promtheus client configuration (with basic auth if provided)
	config := prometheusclient.Config{Address: address}
	if username != "" && password != "" {
		config.RoundTripper = &common.BasicAuthTransport{
			Username: username,
			Password: password,
		}
	}

	client, err := prometheusclient.NewClient(config)
	if err != nil {
		return nil, errors.Wrap(err, "instantiating prometheus client")
	}

	api := prometheus.NewAPI(client)
	return &Backend{
		prometheus: api,
		logger:     logger,
		address:    address,
	}, nil
}

// GetType returns backend type, for example Prometheus.
func (b *Backend) GetType() string {
	return fmodel.PrometheusType
}

// GetAddress returns backend address.
func (b *Backend) GetAddress() string {
	return b.address
}

// QueryInstant performs instant query and returns results in model.Vector type.
func (b *Backend) QueryInstant(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
	level.Debug(b.logger).Log("msg", "querying instant", "query", query)
	val, warns, err := b.prometheus.Query(ctx, query, ts)
	if err != nil {
		return nil, err
	}
	if len(warns) > 0 {
		level.Warn(b.logger).Log("msg", "querying instant warning", strings.Join(warns, ", "), "query", query)
	}

	switch v := val.(type) {
	case model.Vector:
		return v, nil
	default:
		return nil, errors.Errorf("unknown supported type: '%q'", v)
	}
}

// getAlertmanagers returns Alertmanager clients that is associated with the backend.
func (b *Backend) getAlertmanagers(ctx context.Context) ([]*amclient.Alertmanager, error) {
	level.Debug(b.logger).Log("msg", "Get active Alertmanagers")
	amResult, err := b.prometheus.AlertManagers(ctx)
	if err != nil {
		return nil, err
	}
	var ams []*amclient.Alertmanager
	// Get only active Alertmanager instances.
	for _, a := range amResult.Active {
		u, _ := url.Parse(a.URL)
		level.Debug(b.logger).Log("msg", "Setup Alertmanager client", "alertmanager", u.Host)
		// NOTE(kiennt): This is Alertmanager API Ver 2 client.
		// https://github.com/prometheus/alertmanager/blob/master/api/v2/client/alertmanager_client.go
		amCli := amclient.NewHTTPClientWithConfig(nil, &amclient.TransportConfig{
			Host:     u.Host,
			BasePath: "/api/v2/", // hardcode here
			Schemes:  amclient.DefaultSchemes,
		})
		ams = append(ams, amCli)
	}
	return ams, nil
}

// GetAlertmanagerSilences returns silences in Alertmanagers.
func (b *Backend) GetAlertManagerSilences(ctx context.Context, filter []string) (map[string]ammodels.Silence, error) {
	level.Debug(b.logger).Log("msg", "Get Alertmanager silences")
	ams, err := b.getAlertmanagers(ctx)
	if err != nil {
		return nil, err
	}
	result := make(map[string]ammodels.Silence)
	for _, am := range ams {
		silences, err := am.Silence.GetSilences(nil)
		if err != nil {
			fmt.Println(err)
			continue
		}
		for _, s := range silences.GetPayload() {
			if *s.Status.State != ammodels.SilenceStatusStateActive {
				continue
			}
			result[*s.ID] = s.Silence
		}
	}
	return result, err
}
