// Copyright (c) 2019 Tuan-Dat Vu <tuandatk25a@gmail.com>
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
	numberOfNResolvers = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "faythe",
			Subsystem: "nresolvers",
			Name:      "workers_total",
			Help:      "The total number of nresolvers are currently managed by this cluster member.",
		},
		[]string{"cluster"})
)

func init() {
	prometheus.MustRegister(numberOfNResolvers)
}

func ReportNumberOfNResolvers(clusterID string, val float64) {
	numberOfNResolvers.WithLabelValues(clusterID).Add(val)
}
