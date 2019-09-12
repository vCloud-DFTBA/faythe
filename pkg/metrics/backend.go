package metrics

import (
	"context"
	"time"

	"github.com/prometheus/common/model"
)

// Backend is used to interface with a metrics backend
type Backend interface {
	// QueryInstant performs instant query and returns results in model.Vector type.
	QueryInstant(ctx context.Context, query string, ts time.Time) (model.Vector, error)
}
