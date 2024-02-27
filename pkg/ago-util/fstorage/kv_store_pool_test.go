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
	"testing"

	"github.com/abronan/valkeyrie/store"
	assert "github.com/stretchr/testify/require"
)

func TestKvStorePoolGet(t *testing.T) {
	assert := assert.New(t)
	endpoint := "1.2.3.4"
	opts := &options{
		backend:    string(store.ZK),
		poolSize:   5,
		poolLBType: StorePoolLBTypeHash,
	}
	stores, err := createStorePool(opts, endpoint)
	assert.Nil(err)
	defer stores.Close()
	assert.Equal(5, len(stores.stores))
	k := "hello"
	s := stores.get(k)
	// consant
	for i := 0; i < 100; i++ {
		assert.Equal(s, stores.get(k))
	}

	stores.lbType = StorePoolLBTypeRR
	matches := make(map[store.Store]int)
	for i := 0; i < len(stores.stores); i++ {
		s := stores.get(k)
		matches[s] = matches[s] + 1
	}
	assert.Equal(len(matches), len(stores.stores))
	for _, v := range matches {
		assert.True(v > 0)
	}
}
