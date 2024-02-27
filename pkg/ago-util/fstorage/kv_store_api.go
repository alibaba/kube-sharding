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
	"sync/atomic"

	"github.com/abronan/valkeyrie/store"
	glog "k8s.io/klog"
)

// storeapi compatible with valkeyrie.Store
type storeapi interface {
	Put(key string, value []byte, options *store.WriteOptions) error
	Get(key string, options *store.ReadOptions) (*store.KVPair, error)
	Delete(key string) error
	Exists(key string, options *store.ReadOptions) (bool, error)
	List(directory string, options *store.ReadOptions) ([]*store.KVPair, error)
	Close()
}

// DEPRECATED
type refStore struct {
	storeapi
	refs *int64
}

func newRefStore(t storeapi) *refStore {
	r := int64(1)
	return &refStore{
		storeapi: t,
		refs:     &r,
	}
}

func (s *refStore) Clone() storeapi {
	other := &refStore{storeapi: s.storeapi, refs: s.refs}
	atomic.AddInt64(s.refs, 1)
	return other
}

func (s *refStore) Close() {
	if i := atomic.AddInt64(s.refs, -1); i <= 0 {
		glog.V(4).Infof("Do close the storeapi by refs %d", i)
		s.storeapi.Close()
	}
}
