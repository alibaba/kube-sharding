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

package mem

import (
	"time"

	"github.com/alibaba/kube-sharding/pkg/memkube/testutils"
	"github.com/alibaba/kube-sharding/pkg/memkube/testutils/carbon/spec"
	assert "github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
)

// NewTestStore creates a store for test
func NewTestStore(stopCh chan struct{}, persistPeriod time.Duration) Store {
	opt := WithStorePersistPeriod(persistPeriod)
	persister := NewFakePersister()
	store := NewStore(spec.Scheme, spec.SchemeGroupVersion.WithKind(Kind(&spec.WorkerNode{})), "ns", &OneBatcher{"b"}, persister, opt)
	w, _ := store.Watch(&WatchOptions{})
	testutils.ConsumeWatchEvents(w, stopCh)
	return store
}

func getObject(assert *assert.Assertions, store Store, name string, timeout time.Duration) runtime.Object {
	r, err := store.Get(name)
	assert.Nil(err)
	assert.Nil(r.Wait(timeout))
	return r.List()[0]
}

func deleteObject(assert *assert.Assertions, store Store, name string, timeout time.Duration) {
	result, err := store.Delete(name)
	assert.Nil(err)
	assert.Nil(result.Wait(timeout))
}

func addObject(assert *assert.Assertions, store Store, obj runtime.Object, timeout time.Duration) {
	result, err := store.Add(obj)
	assert.Nil(err)
	assert.Nil(result.Wait(timeout))
}

func tryUpdateObject(assert *assert.Assertions, store Store, obj runtime.Object, timeout time.Duration) bool {
	result, err := store.Update(obj)
	if err == nil {
		assert.Nil(result.Wait(timeout))
		return true
	}
	return false
}
