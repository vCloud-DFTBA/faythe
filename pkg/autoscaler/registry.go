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

import "sync"

// Registry is a map type that can be safely shared between
// goroutines that require read/write access to a map.
type Registry struct {
	sync.RWMutex
	items map[string]Scaler
}

// RegistryItem contains a key/value pair item of a registry
type RegistryItem struct {
	Key   string
	Value Scaler
}

// Set adds an item to a registry
func (r *Registry) Set(key string, value Scaler) {
	r.Lock()
	defer r.Unlock()

	r.items[key] = value
}

// Get retrieves the value for a registry
func (r *Registry) Get(key string) (Scaler, bool) {
	r.Lock()
	defer r.Unlock()

	value, ok := r.items[key]
	return value, ok
}

// Iter iterates over the items in a registry
// Each item is sent over a channel, so that
// we can iterate over the registry using the builtin
// range keyword
func (r *Registry) Iter() <-chan RegistryItem {
	c := make(chan RegistryItem)

	go func() {
		r.Lock()
		defer r.Unlock()

		for k, v := range r.items {
			c <- RegistryItem{k, v}
		}
	}()

	return c
}

// Delete deletes the item with the given name from the registry
func (r *Registry) Delete(key string) {
	r.Lock()
	defer r.Unlock()

	delete(r.items, key)
}
