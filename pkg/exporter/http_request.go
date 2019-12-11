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

package exporter

import "github.com/prometheus/client_golang/prometheus"

var (
	InFlightGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "faythe",
			Subsystem: "api",
			Name:      "in_flight_requests",
			Help:      "A gauge of requests currently being served by the wrapper handler.",
		},
	)

	RequestsCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "faythe",
			Subsystem: "api",
			Name:      "requests_total",
			Help:      "A counter for requests to the wrapped handler.",
		},
		[]string{"handler", "code", "method"},
	)

	RequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "faythe",
			Subsystem: "api",
			Name:      "request_duration_seconds",
			Help:      "A histogram of latencies for requests.",
			Buckets:   []float64{.05, 0.1, .25, .5, .75, 1, 2, 5, 20, 60},
		},
		[]string{"handler", "code", "method"},
	)

	RequestSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "faythe",
			Subsystem: "api",
			Name:      "request_size_bytes",
			Help:      "A histogram of request sizes for requests.",
			Buckets:   prometheus.ExponentialBuckets(100, 10, 7),
		},
		[]string{"handler", "code", "method"},
	)

	ResponseSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "faythe",
			Subsystem: "api",
			Name:      "response_size_bytes",
			Help:      "A histogram of response sizes for requests.",
			Buckets:   prometheus.ExponentialBuckets(100, 10, 7),
		},
		[]string{"handler", "code", "method"},
	)
)

func init() {
	prometheus.MustRegister(InFlightGauge, RequestsCount, RequestDuration, RequestSize, ResponseSize)
}
