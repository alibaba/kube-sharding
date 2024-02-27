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

package fstorage

import (
	"strings"
	"sync/atomic"
	"time"

	"github.com/abronan/valkeyrie"
	"github.com/abronan/valkeyrie/store"
	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	glog "k8s.io/klog"
)

// kvStorePool makes a pool of stores (usually represents a remote store network connection), to write/read fast.
type kvStorePool struct {
	stores []store.Store
	lbType int
	idx    uint64
}

var _ storeapi = &kvStorePool{}

// Store pool load balancing type
const (
	// For these you don't care the write/read sequence
	StorePoolLBTypeRR   = 1
	StorePoolLBTypeHash = 2
)

func createStorePool(opt *options, endpoint string) (*kvStorePool, error) {
	size := opt.poolSize
	if size <= 0 {
		size = 1
	}
	glog.Infof("Create kv store pool: %s, %+v", endpoint, *opt)
	stores := make([]store.Store, size)
	for i := 0; i < size; i++ {
		s, err := valkeyrie.NewStore(store.Backend(opt.backend), strings.Split(endpoint, ","),
			&store.Config{
				ConnectionTimeout: time.Duration(opt.connectTimeout) * time.Second,
				MaxBufferSize:     opt.maxBufferSize,
				Bucket:            opt.bucket,
			})
		if err != nil {
			return nil, err
		}
		stores[i] = s
	}
	return &kvStorePool{
		stores: stores,
		lbType: opt.poolLBType,
	}, nil
}

func (sp *kvStorePool) Put(key string, value []byte, options *store.WriteOptions) error {
	return sp.get(key).Put(key, value, options)
}

func (sp *kvStorePool) Get(key string, options *store.ReadOptions) (*store.KVPair, error) {
	return sp.get(key).Get(key, options)
}

func (sp *kvStorePool) Delete(key string) error {
	return sp.get(key).Delete(key)
}

func (sp *kvStorePool) Exists(key string, options *store.ReadOptions) (bool, error) {
	return sp.get(key).Exists(key, options)
}

func (sp *kvStorePool) List(directory string, options *store.ReadOptions) ([]*store.KVPair, error) {
	return sp.get(directory).List(directory, options)
}

func (sp *kvStorePool) Close() {
	glog.Info("Close kv store pool.")
	for i := range sp.stores {
		sp.stores[i].Close()
	}
}

func (sp *kvStorePool) get(key string) store.Store {
	if len(sp.stores) == 1 {
		return sp.stores[0]
	}
	if sp.lbType == StorePoolLBTypeHash {
		// Select store by constant hash. Write/Read a same key on same connection
		h := utils.HashString64([]byte(key))
		return sp.stores[h%uint64(len(sp.stores))]
	}
	// else StorePoolLBTypeRR or default
	atomic.AddUint64(&sp.idx, 1)
	idx := atomic.LoadUint64(&sp.idx)
	return sp.stores[idx%uint64(len(sp.stores))]
}
