/*
Copyright 2024 The Alibaba Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cache

import (
	"errors"
	"sync"
	"time"
)

var (
	// ErrKeyNotFound indicates the cache not found.
	ErrKeyNotFound = errors.New("key not found")
)

// Store represents a cache implementation which stores key value.
type Store interface {
	Get(key interface{}) (interface{}, error)
	SetWithExpire(key, value interface{}, expiration time.Duration) error
}

type pipe struct {
	value interface{}
	lock  *sync.RWMutex
}

// LoaderFunc load values for a cache key.
type LoaderFunc func(key interface{}) (interface{}, error)

// PipeCache limits cache loader (load values from remote services) in a pipe, which means limit the concurrence.
// NOTE: The refactor version here only allows 1 concurrent to load for simple implementation reason.
type PipeCache struct {
	lock  *sync.RWMutex
	store Store
}

// NewPipeCache create a new piped cache.
func NewPipeCache(store Store) *PipeCache {
	return &PipeCache{
		lock:  new(sync.RWMutex),
		store: store,
	}
}

// Fetch get the cache value, if the value not found, call loader to load. This function limits the cocurrent loader count.
func (pc *PipeCache) Fetch(key interface{}, expiration time.Duration, loader LoaderFunc) (interface{}, error) {
	i, err := pc.store.Get(key)
	if err == nil {
		return pc.loadValue(key, loader, i.(*pipe))
	}
	if err != ErrKeyNotFound {
		return nil, err
	}
	pc.lock.Lock()
	i, _ = pc.store.Get(key) // double check
	if i != nil {
		pc.lock.Unlock()
		return pc.loadValue(key, loader, i.(*pipe))
	}
	pipe := &pipe{lock: new(sync.RWMutex)}
	pc.store.SetWithExpire(key, pipe, expiration)
	pc.lock.Unlock()

	return pc.loadValue(key, loader, pipe)
}

func (pc *PipeCache) loadValue(key interface{}, loader LoaderFunc, pipe *pipe) (interface{}, error) {
	pipe.lock.RLock()
	v := pipe.value
	pipe.lock.RUnlock()
	if v != nil {
		return v, nil
	}
	pipe.lock.Lock()
	defer pipe.lock.Unlock()
	if pipe.value != nil { // double check
		return pipe.value, nil
	}
	v, err := loader(key)
	if err != nil {
		return nil, err
	}
	pipe.value = v
	return v, nil
}
