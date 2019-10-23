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

package alert

import (
	"time"

	"github.com/ntk148v/faythe/pkg/model"
)

type Alert struct {
	State   model.Alert
	cooling bool
}

func (a *Alert) ShouldFire(duration time.Duration) bool {
	return a.State.Active && time.Now().Sub(a.State.StartedAt) >= duration
}

func (a *Alert) IsCoolingDown(cooldown time.Duration) bool {
	a.cooling = time.Now().Sub(a.State.FiredAt) <= cooldown
	return a.cooling
}

func (a *Alert) Start() {
	a.State.StartedAt = time.Now()
	a.State.Active = true
}

func (a *Alert) Fire(firedAt time.Time) {
	if a.State.FiredAt.IsZero() || !a.cooling {
		a.State.FiredAt = firedAt
	}
}

func (a *Alert) Reset() {
	a.State.StartedAt = time.Time{}
	a.State.Active = false
	a.State.FiredAt = time.Time{}
}

func (a *Alert) IsActive() bool {
	return a.State.Active
}
