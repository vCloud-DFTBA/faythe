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

package cluster

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/vCloud-DFTBA/faythe/pkg/model"
)

// Set of raw Prometheus metrics.
// Do not increment directly, use Report* methods.
var (
	memberJoinCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "faythe_cluster_member_join_total",
			Help: "A counter of the number of members that have joined.",
		})
	memberLeaveCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "faythe_cluster_member_leave_total",
			Help: "A counter of the number of members that have left.",
		})
)

func init() {
	prometheus.MustRegister(memberJoinCounter)
	prometheus.MustRegister(memberLeaveCounter)
}

func registerMemberInfo(clusterID string, member model.Member) {
	memberInfo := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "faythe_cluster_member_info",
			Help: "A metric with constant '1' value labeled by cluster id and member information",
			ConstLabels: prometheus.Labels{
				"cluster": clusterID,
				"name":    member.Name,
				"id":      member.ID,
				"address": member.Address,
			},
		})
	prometheus.MustRegister(memberInfo)
	memberInfo.Set(1)
}

func reportClusterJoin() {
	memberJoinCounter.Inc()
}

func reportClusterLeave() {
	memberLeaveCounter.Inc()
}
