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

package api

import "net/http"

func (a *API) createScaler(w http.ResponseWriter, req *http.Request) {
	// Save a Scaler object in etcd3
}

func (a *API) listScalers(w http.ResponseWriter, req *http.Request) {
	// List all current Scalers from etcd3
}

func (a *API) deleteScaler(w http.ResponseWriter, req *http.Request) {
	// Delete a Scaler from etcd3
}

func (a *API) updateScaler(w http.ResponseWriter, req *http.Request) {
	// Update a existed Scaler information
}
