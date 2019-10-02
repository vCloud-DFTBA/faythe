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

package metrics

import (
	"sync"
)

type Registry struct {
	sync.RWMutex
	items map[string]Backend
}

type RegistryItem struct {
	Key   string
	Value Backend
}

// Get returns the Backend with the given name from the Registry
func (r *Registry) Get(key string) (Backend, bool) {
	r.RLock()
	defer r.RUnlock()

	value, ok := r.items[key]
	return value, ok
}

// Delete deletes the Backend with the given name from the Registry, or noops
// if the Backend doesn't exist. It only deletes it from the Registry; it does
// not clean up the underlying type.
func (r *Registry) Delete(key string) {
	r.Lock()
	defer r.Unlock()

	delete(r.items, key)
}

// Put puts a Backend with the given name into the Registry. If a Backend
// already exists with the given name, it will simply be overwritten.
func (r *Registry) Set(key string, value Backend) {
	r.Lock()
	defer r.Unlock()

	r.items[key] = value
}

// Iter iterates over the items in a Registry
// Each item is sent over a channel, so that
// we can iterate over the Registry using the builtin
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
