// Copyright (c) 2021 Kien Nguyen-Tuan <kiennt2609@gmail.com>
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

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Set of raw Prometheus metrics.
// Do not increment directly, use Report* methods.
var (
	numberOfClouds = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "faythe",
			Subsystem: "",
			Name:      "clouds_total",
			Help:      "The total number of clouds are currently registered.",
		},
		[]string{"cluster"})
)

func init() {
	prometheus.MustRegister(numberOfClouds)
}

func ReportNumberOfClouds(clusterID string, val float64) {
	numberOfClouds.WithLabelValues(clusterID).Add(val)
}
