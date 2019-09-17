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

	"github.com/pkg/errors"
)

// RegistryInterface is an interface to a Backend registry.
// See function comments for implementation limitations.
type RegistryInterface interface {
	Get(name string) (Backend, error)
	Delete(name string)
	Put(name string, backend Backend)
}

type registry struct {
	sync.RWMutex
	items map[string]Backend
}

var reg = registry{
	items: make(map[string]Backend),
}

// Registry provides an interface to the single Backend registry.
func Registry() RegistryInterface {
	return &reg
}

// Get returns the Backend with the given name from the registry, or an error
// if it does not exist. A Backend returned is not guaranteed to be valid
// still; it's assumed that the caller will handle Backend errors and delete it
// from the Registry if appropriate.
func (r *registry) Get(name string) (Backend, error) {
	r.RLock()
	defer r.RUnlock()

	var backend Backend
	var ok bool
	if backend, ok = r.items[name]; !ok {
		return nil, errors.Errorf("backend %q does not exist", name)
	}

	return backend, nil
}

// Delete deletes the Backend with the given name from the registry, or noops
// if the Backend doesn't exist. It only deletes it from the registry; it does
// not clean up the underlying type.
func (r *registry) Delete(name string) {
	r.Lock()
	defer r.Unlock()

	delete(r.items, name)
}

// Put puts a Backend with the given name into the registry. If a Backend
// already exists with the given name, it will simply be overwritten.
func (r *registry) Put(name string, backend Backend) {
	r.Lock()
	defer r.Unlock()

	r.items[name] = backend
}
