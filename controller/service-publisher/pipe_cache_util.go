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

package publisher

import (
	"sync"
	"time"

	"github.com/alibaba/kube-sharding/pkg/ago-util/cache"
	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	"github.com/bluele/gcache"
)

// adapter gcache.Cache to cache.Store
type gcacheStore struct {
	gcache gcache.Cache
}

func (s *gcacheStore) Get(key interface{}) (interface{}, error) {
	v, err := s.gcache.Get(key)
	if err == gcache.KeyNotFoundError {
		return nil, cache.ErrKeyNotFound
	}
	return v, err
}

func (s *gcacheStore) SetWithExpire(key, value interface{}, expiration time.Duration) error {
	return s.gcache.SetWithExpire(key, value, expiration)
}

func (s *gcacheStore) Keys() []interface{} {
	return s.gcache.Keys(true)
}

var pipeCacheOncer sync.Once
var pipeCache *cache.PipeCache

func getPipeCache() *cache.PipeCache {
	initPipeCache := func() {
		store := &gcacheStore{gcache.New(100).LRU().Build()}
		pipeCache = cache.NewPipeCache(store)
	}
	pipeCacheOncer.Do(initPipeCache)
	return pipeCache
}

func getPipeCacheKey(ptype, namespace, name string) string {
	return utils.AppendString(ptype, ":", namespace, ":", name)
}
