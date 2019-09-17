package prometheus

import (
	"context"
	"strings"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	prometheusclient "github.com/prometheus/client_golang/api"
	prometheus "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"

	"github.com/ntk148v/faythe/pkg/metrics"
)

// Backend implements a metric backend for Prometheus.
type Backend struct {
	prometheus prometheus.API
	logger     log.Logger
}

const (
	prometheusRequestTimeout = 10 * time.Second
)

// NewClient returns a new client for talking to a Prometheus Backend, or an error
func NewClient(address string, logger log.Logger) (metrics.Backend, error) {
	if logger == nil {
		logger = log.NewNopLogger()
	}
	if address == "" {
		// Under the hood, prometheusclient uses url.Parse() which allows
		// relative URLs, etc. Empty would be allowed, so disallow it
		// explicitly here.
		return nil, errors.New("address must not be empty")
	}

	client, err := prometheusclient.NewClient(prometheusclient.Config{
		Address: address,
	})
	if err != nil {
		return nil, errors.Wrap(err, "instantiating prometheus client")
	}

	api := prometheus.NewAPI(client)
	return Backend{
		prometheus: api,
		logger:     logger,
	}, nil
}

// QueryInstant performs instant query and returns results in model.Vector type.
func (b Backend) QueryInstant(ctx context.Context, query string, ts time.Time) (model.Vector, error) {
	level.Debug(b.logger).Log("msg", "querying instant", "query", query)
	val, warns, err := b.prometheus.Query(ctx, query, ts)
	if err != nil {
		return nil, errors.Wrap(err, "querying instant")
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
