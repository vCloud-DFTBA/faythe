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

package metrics

import (
	"context"
	"time"

	"github.com/prometheus/common/model"
)

const (
	// Prometheus is a Prometheus backend
	Prometheus string = "prometheus"
)

// Backend is used to interface with a metrics backend
type Backend interface {
	// QueryInstant performs instant query and returns results in model.Vector type.
	QueryInstant(ctx context.Context, query string, ts time.Time) (model.Vector, error)
}
