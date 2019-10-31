// Copyright (c) 2019 Dat Vu Tuan <tuandatk25a@gmail.com>
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

package autohealer

// NodeMetric contains and ID and a metric
type NodeMetric struct {
	CloudID string   `json:"cloudid"`
	Metric  NodeInfo `json:"metric"`
}

// NodeInfo contains information of node name and ip
type NodeInfo struct {
	Instance string `json:"instance"`
	Nodename string `json:"nodename"`
}
