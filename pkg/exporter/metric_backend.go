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

// Set of raw Prometheus metrics.
// Do not increment directly, use Report* methods.
var (
	metricQueryFailureCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "faythe",
			Subsystem: "metric_backend",
			Name:      "query_failures_total",
			Help:      "The total number of metric backend query failures total.",
		},
		[]string{"cluster", "type", "endpoint"})
)

func init() {
	prometheus.MustRegister(metricQueryFailureCounter)
}

func ReportMetricQueryFailureCounter(clusterID, backendType, backendEndpoint string) {
	metricQueryFailureCounter.WithLabelValues(clusterID, backendType, backendEndpoint).Inc()
}
