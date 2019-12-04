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

package autoscaler

import "github.com/prometheus/client_golang/prometheus"

// Set of raw Prometheus metrics.
// Do not increment directly, use Report* methods.
var (
	numberOfScalers = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "faythe_scalers_total",
			Help: "Total number of scalers are currently managed by this cluster member.",
		},
		[]string{"cluster", "member"})
)

func init() {
	prometheus.MustRegister(numberOfScalers)
}

func reportNumScalers(clusterID, memberName string, val float64) {
	if val == 0 {
		numberOfScalers.WithLabelValues(clusterID, memberName).Set(val)
	} else if val < 0 {
		numberOfScalers.WithLabelValues(clusterID, memberName).Sub(val)
	} else {
		numberOfScalers.WithLabelValues(clusterID, memberName).Add(val)
	}
}
