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

package nresolver

import "sync"

type Registry struct {
	sync.RWMutex
	items map[string]*NResolver
}

type RegistryItem struct {
	Name  string
	Value *NResolver
}

func (r *Registry) Get(key string) (*NResolver, bool) {
	r.RLock()
	defer r.RUnlock()

	value, ok := r.items[key]
	return value, ok
}

func (r *Registry) Delete(key string) {
	r.Lock()
	defer r.Unlock()

	delete(r.items, key)
}

func (r *Registry) Set(key string, value *NResolver) {
	r.Lock()
	defer r.Unlock()

	r.items[key] = value
}

func (r *Registry) Iter() <-chan RegistryItem {
	c := make(chan RegistryItem)

	go func() {
		r.Lock()
		defer func() {
			r.Unlock()
			close(c)
		}()

		for k, v := range r.items {
			c <- RegistryItem{k, v}
		}
	}()
	return c
}
