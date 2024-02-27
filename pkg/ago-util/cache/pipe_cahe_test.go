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
	"log"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type fakeStore struct {
	lock   *sync.RWMutex
	values map[interface{}]interface{}
}

func newFakeStore() *fakeStore {
	return &fakeStore{lock: new(sync.RWMutex), values: make(map[interface{}]interface{})}
}

func (s *fakeStore) Get(key interface{}) (interface{}, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	val, ok := s.values[key]
	if !ok {
		return nil, ErrKeyNotFound
	}
	return val, nil
}

func (s *fakeStore) SetWithExpire(key, value interface{}, expiration time.Duration) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.values[key] = value
	if value == nil {
		delete(s.values, key)
	}
	return nil
}

func (s *fakeStore) Keys() []interface{} {
	s.lock.RLock()
	defer s.lock.RUnlock()
	keys := make([]interface{}, len(s.values))
	for k := range s.values {
		keys = append(keys, k)
	}
	return keys
}

func TestPipeCache(t *testing.T) {
	assert := assert.New(t)
	store := newFakeStore()
	cache := NewPipeCache(store)
	k := "hello"
	var n uint64
	var wg sync.WaitGroup
	// test concurrent get
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			k0 := "hello"
			v, err := cache.Fetch(k0, time.Second, func(k interface{}) (interface{}, error) {
				time.Sleep(10 * time.Millisecond)
				atomic.AddUint64(&n, 1)
				return "world", nil
			})
			assert.Nil(err)
			assert.Equal("world", v.(string))
		}()
	}
	wg.Wait()
	log.Printf("loader called %d times", int(n))
	assert.Equal(uint64(1), n)
	i, err := store.Get(k)
	assert.Nil(err)
	assert.Equal("world", i.(*pipe).value.(string))

	// test load error
	k2 := "abc"
	_, err = cache.Fetch(k2, time.Second, func(k interface{}) (interface{}, error) {
		return nil, errors.New("fake error")
	})
	assert.NotNil(err)
	// but the pipe created
	i, _ = store.Get(k2)
	assert.NotNil(i)
	assert.Nil(i.(*pipe).value)

	// test expiration: the pipe removed, and the loader expected to be called
	n = 0
	store.SetWithExpire(k, nil, time.Millisecond)
	i, _ = cache.Fetch(k, time.Second, func(k interface{}) (interface{}, error) {
		n++
		return "world", nil
	})
	assert.NotNil(i)
	assert.Equal(uint64(1), n)
}
