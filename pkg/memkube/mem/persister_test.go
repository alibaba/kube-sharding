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
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	fsutil "github.com/alibaba/kube-sharding/pkg/ago-util/fstorage/testutil"
	"github.com/alibaba/kube-sharding/pkg/memkube/testutils"
	"github.com/alibaba/kube-sharding/pkg/memkube/testutils/carbon/spec"
	"github.com/stretchr/testify/mock"
	assert "github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
)

func workerNodeCreator() runtime.Object {
	return &spec.WorkerNode{}
}

func TestFsPersister(t *testing.T) {
	assert := assert.New(t)
	fsurl := "file:///tmp/fstore"
	fsurl2 := "file:///tmp/fstore2"
	ns := "ns1"

	persister, err := NewFsPersister(fsurl, fsurl2, ns, spec.SchemeGroupVersion.WithKind("WorkerNode"), nil)
	assert.Nil(err)
	defer testutils.ClearLocation(persister.fsl)

	batch0 := newTestBatchObject("b", 2, nil)
	assert.Nil(persister.Persist(batch0))

	// verify last hash
	h, ok := persister.lastHashs[batch0.Name]
	assert.True(ok)
	fmt.Printf("persisted hash: %s\n", h)
	assert.Nil(persister.Persist(batch0))

	batch1 := newTestBatchObject("b1", 3, nil)
	assert.Nil(persister.Persist(batch1))

	// test recover
	persister.lastHashs = map[string]string{}
	batches, _, err := persister.Recover(workerNodeCreator)
	assert.Nil(err)
	// verity the persisted hash
	assert.Equal(2, len(persister.lastHashs))
	assert.Equal(h, persister.lastHashs[batch0.Name])
	batchMap := map[string]*BatchObject{}
	for _, b := range batches {
		batchMap[b.Name] = b
	}
	eq, err := batch0.DeepEqual(batchMap[batch0.Name])
	assert.Nil(err)
	assert.True(eq)
	eq, err = batch1.DeepEqual(batchMap[batch1.Name])
	assert.Nil(err)
	assert.True(eq)
}

func TestFsPersisterRemove(t *testing.T) {
	assert := assert.New(t)
	fsurl := "file:///tmp/fstore"
	ns := "ns1"
	persister, err := NewFsPersister(fsurl, "", ns, spec.SchemeGroupVersion.WithKind("WorkerNode"), nil)
	assert.Nil(err)
	defer testutils.ClearLocation(persister.fsl)

	batch0 := newTestBatchObject("b", 2, nil)
	assert.Nil(persister.Persist(batch0))

	batches, _, err := persister.Recover(workerNodeCreator)
	assert.Nil(err)
	assert.Equal(1, len(batches))

	batch0.Items = make(pendingList, 0)
	assert.Nil(persister.Persist(batch0))
	batches, _, err = persister.Recover(workerNodeCreator)
	assert.Nil(err)
	assert.Equal(0, len(batches))
}

func TestFsPersistBenchmark(t *testing.T) {
	assert := assert.New(t)
	fsurl := "file:///tmp/fstore"
	ns := "ns1"
	persister, err := NewFsPersister(fsurl, "", ns, spec.SchemeGroupVersion.WithKind("WorkerNode"), nil)
	assert.Nil(err)
	mockLoc := &fsutil.Location{}
	persister.fsl = mockLoc
	mockFs := &fsutil.FStorage{}

	count := 1000
	size := 1024 * 100 // Kb
	times := 10
	str := strings.Repeat("A", size)
	mockLoc.On("Create", mock.Anything).Return(mockFs, nil).Times(times)
	mockFs.On("Write", mock.Anything).Return(nil).Times(times)
	mockFs.On("Close", mock.Anything).Return(nil).Times(times)
	var totalTime int64
	for i := 0; i < times; i++ {
		batch := newTestBatchObject("b1", count, func(idx int, node *spec.WorkerNode) {
			node.Kvs = map[string]string{
				"str":   str,
				"times": strconv.Itoa(i), // to make different hash in persister.
			}
		})
		start := time.Now()
		err = persister.Persist(batch)
		assert.Nil(err)
		totalTime += time.Now().Sub(start).Nanoseconds()
	}
	avgTime := totalTime / int64(times) / 1000 / 1000
	// NOTE: the time cost here includes mock argument compare.
	fmt.Printf("Benchmark on persist, times %d, avg %dms, count in batch %d, each batch size %dKb\n",
		times, avgTime, count, count*size/1024)
	mockLoc.AssertExpectations(t)
	mockFs.AssertExpectations(t)
}
