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
	"encoding/json"
	"fmt"
	"math/rand"
	"reflect"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"

	goerrors "errors"

	"github.com/alibaba/kube-sharding/pkg/memkube/testutils"
	"github.com/alibaba/kube-sharding/pkg/memkube/testutils/carbon/spec"
	"github.com/golang/mock/gomock"
	assert "github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

func TestPersistedStoreWriteRead(t *testing.T) {
	assert := assert.New(t)
	stopCh := make(chan struct{})
	defer close(stopCh)
	timeout := time.Millisecond * 100
	store0 := NewStore(spec.Scheme, spec.SchemeGroupVersion.WithKind(Kind(&spec.WorkerNode{})), "ns", &OneBatcher{}, NewFakePersister(), WithStorePersistPeriod(time.Millisecond*10))
	store := store0.(*persistedStore)
	w, _ := store.Watch(&WatchOptions{})
	testutils.ConsumeWatchEvents(w, stopCh)

	name := "w1"
	node := &spec.WorkerNode{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"size": "xxl",
			},
		},
		Age:  11,
		Nick: "alice",
	}
	result, err := store.Add(node)
	assert.Nil(err)
	// duplicated add
	_, err = store.Add(node)
	assert.NotNil(err)
	verifyNode := func(name string, age int, exists bool, ver uint64) {
		obj, _ := store.Peek(name)
		if exists {
			assert.NotNil(obj)
			assert.Equal(age, obj.(*spec.WorkerNode).Age)
			ver0, err := MetaResourceVersion(obj)
			assert.Nil(err)
			assert.Equal(ver, ver0)
		} else {
			assert.Nil(obj)
		}
	}
	// nil until commit
	verifyNode(MetaName(node), 0, false, 0)
	// do the commit
	store.run()
	assert.Nil(result.Wait(timeout))

	verifyNode(MetaName(node), 11, true, 1)

	// update the node uid
	obj0, _ := store.Peek(name)
	accessor, _ := meta.Accessor(obj0)
	node.UID = accessor.GetUID()

	// list
	objs, version, _ := store.List(nil)
	assert.Equal(1, len(objs))
	assert.Equal(uint64(1), version)

	// update failed by ResourceVersion conflict
	node.Age = 12
	_, err = store.Update(node)
	assert.True(errors.IsConflict(err))
	// change the version
	node.ResourceVersion = "1"
	result, err = store.Update(node)
	assert.Nil(err)
	store.run()
	assert.Nil(result.Wait(timeout))
	verifyNode(MetaName(node), 12, true, 2)

	// reject if uid not equal
	node.ResourceVersion = "2"
	node.UID = ""
	_, err = store.Update(node)
	fmt.Printf("err: %v\n", err)
	assert.True(errors.IsResourceExpired(err))
	node.UID = accessor.GetUID()

	// wait result timeout
	node.ResourceVersion = "2"
	result, err = store.Update(node)
	assert.Nil(err)
	assert.Equal(ErrWaitTimeout, result.Wait(timeout))

	// overwrite the pending update
	node.ResourceVersion = "3"
	_, err = store.Update(node)
	assert.Nil(err)

	// list
	objs, version, _ = store.List(nil)
	assert.Equal(uint64(4), version) // expect the current max version
	assert.Equal(1, len(objs))
	node0 := objs[0].(*spec.WorkerNode)
	assert.Equal(12, node0.Age)
	v, _ := MetaResourceVersion(node0)
	assert.Equal(uint64(2), v)

	// list by selector
	lblSelector, err := metav1.ParseToLabelSelector("size=xxl")
	assert.Nil(err)
	selector, err := metav1.LabelSelectorAsSelector(lblSelector)
	assert.Nil(err)
	objs, _, _ = store.List(selector)
	assert.Equal(1, len(objs))

	// delete timeout
	result, err = store.Delete(name)
	assert.Nil(err)
	assert.NotNil(result.Wait(timeout))

	// update an deleting object gets conflict error
	_, err = store.Update(node)
	assert.True(errors.IsConflict(err))

	// delete success
	result, err = store.Delete(name)
	store.run()
	assert.Nil(result.Wait(timeout))
	verifyNode(MetaName(node), 0, false, 0)

	// test NotFound error
	_, err = store.Get("non-exists")
	assert.True(errors.IsNotFound(err))
	_, err = store.Delete("non-exists")
	assert.True(errors.IsNotFound(err))

	node = &spec.WorkerNode{
		ObjectMeta: metav1.ObjectMeta{
			Name: "abc",
		},
	}
	_, err = store.Update(node)
	assert.True(errors.IsNotFound(err))
}

func TestPersistedStoreValidate(t *testing.T) {
	assert := assert.New(t)
	stopCh := make(chan struct{})
	defer close(stopCh)
	store0 := NewStore(spec.Scheme, spec.SchemeGroupVersion.WithKind(Kind(&spec.WorkerNode{})), "ns", &OneBatcher{}, NewFakePersister(), WithStorePersistPeriod(time.Millisecond*10))
	store := store0.(*persistedStore)
	w, _ := store.Watch(&WatchOptions{})
	testutils.ConsumeWatchEvents(w, stopCh)

	// empty add/update
	node := &spec.WorkerNode{}
	_, err := store.Add(node)
	assert.Equal(ErrObjectInvalid, err)

	_, err = store.Update(node)
	assert.Equal(ErrObjectInvalid, err)
}

func TestPersistedStoreSchemeNew(t *testing.T) {
	assert := assert.New(t)
	obj, err := spec.Scheme.New(spec.SchemeGroupVersion.WithKind(Kind(&spec.WorkerNode{})))
	assert.Nil(err)
	fmt.Printf("%v, %v\n", obj, reflect.TypeOf(obj))
}

func TestPersistedStoreCommit(t *testing.T) {
	assert := assert.New(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	stopCh := make(chan struct{})
	defer close(stopCh)
	persister := NewMockPersister(ctrl)
	store := NewStore(spec.Scheme, spec.SchemeGroupVersion.WithKind(Kind(&spec.WorkerNode{})), "ns", &OneBatcher{}, persister).(*persistedStore)
	defer store.broadcaster.Shutdown()

	node1 := &spec.WorkerNode{
		ObjectMeta: metav1.ObjectMeta{
			Name: "n1",
		},
	}
	node2 := &spec.WorkerNode{
		ObjectMeta: metav1.ObjectMeta{
			Name: "n2",
		},
	}
	_, err := store.Add(node1)
	assert.Nil(err)
	_, err = store.Add(node2)
	assert.Nil(err)

	// persist failed
	perr := goerrors.New("persist error")
	persister.EXPECT().PersistMeta(gomock.Any()).Return(perr).Times(1)
	store.run()
	// the objects remain pending
	for _, item := range store.listAll() {
		assert.True(item.IsPending())
	}

	// delete one
	_, err = store.Delete(node2.Name)
	assert.Nil(err)

	// create a watch consumer
	eventsCh := make(chan watch.Event, 100)
	w, _ := store.Watch(&WatchOptions{})
	testutils.CollectWatchEvents(w, eventsCh, stopCh)

	// persist success
	persister.EXPECT().PersistMeta(gomock.Any()).Return(nil).Times(1)
	persister.EXPECT().Persist(testutils.FuncMockMatcher{
		Func: func(x interface{}) bool {
			batch := x.(*BatchObject)
			return 1 == len(batch.deletings) && 1 == len(batch.Items)
		},
		Name: "match deleting",
	}).Return(nil).Times(1)
	store.run()
	// the store.broadcaster async distribute the events.
	time.Sleep(time.Millisecond * 10)
	// all objects committed
	for _, item := range store.listAll() {
		assert.False(item.IsPending())
	}
	assert.Equal(1, len(store.listAll()))
	assert.Equal(2, len(eventsCh))
	// verify the events
	evt := <-eventsCh
	assert.Equal(watch.Added, evt.Type)
	v, _ := MetaResourceVersion(evt.Object)
	assert.Equal(uint64(1), v)
	evt = <-eventsCh
	v, _ = MetaResourceVersion(evt.Object)
	assert.Equal(watch.Deleted, evt.Type)
	assert.Equal(uint64(3), v)
}

func TestPersistedStoreList(t *testing.T) {
	assert := assert.New(t)
	store0 := NewStore(spec.Scheme, spec.SchemeGroupVersion.WithKind(Kind(&spec.WorkerNode{})), "ns", &OneBatcher{}, NewFakePersister(), WithStorePersistPeriod(time.Millisecond*10))
	store := store0.(*persistedStore)
	count := 100
	for i := 0; i < count; i++ {
		node := &spec.WorkerNode{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("node-%d", i),
			},
		}
		_, err := store.Add(node)
		assert.Nil(err)
	}
	pendings := store.listAll()
	bs, err := json.Marshal(pendings)
	assert.Nil(err)
	// verify the stable
	for i := 0; i < 1000; i++ {
		pendings := store.listAll()
		bs0, err := json.Marshal(pendings)
		assert.Nil(err)
		assert.Equal(string(bs), string(bs0))
	}
}

func TestPersistedStoreRecoverCompatible(t *testing.T) {
	assert := assert.New(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	store0 := NewStore(spec.Scheme, spec.SchemeGroupVersion.WithKind(Kind(&spec.WorkerNode{})), "ns", &OneBatcher{}, NewFakePersister(), WithStorePersistPeriod(time.Millisecond*10))
	store := store0.(*persistedStore)
	persister := NewMockPersister(ctrl)
	store.persister = persister
	mi := NewMetaInfo()
	batches := newTestBatchObject("b1", 2, func(idx int, node *spec.WorkerNode) {
		node.SetResourceVersion(strconv.Itoa(idx + 1))
	})
	persister.EXPECT().Recover(gomock.Any()).Return([]*BatchObject{batches}, mi, nil).Times(1)
	assert.Nil(store.recover())
	// auto fix to the max version in all items
	assert.Equal(uint64(2), store.meta.VersionSeed)
}

// If we launch concurrent Updates, we expect the result is ok, no timeout error.
// If WriteResult.Wait check object by IsPending, concurrent Updates will make timeout error, because the object manybe
// set pending by other threads. To fix this, i don't checking pending, but checking if the object version matches.
func TestPersistedStoreWaitResult(t *testing.T) {
	assert := assert.New(t)
	persistPeriod := time.Millisecond * 10
	timeout := persistPeriod * 2
	opt := WithStorePersistPeriod(persistPeriod)
	store0 := NewStore(spec.Scheme, spec.SchemeGroupVersion.WithKind(Kind(&spec.WorkerNode{})), "ns", &OneBatcher{}, NewFakePersister(), opt)
	store := store0.(*persistedStore)
	stopCh := make(chan struct{})
	defer close(stopCh)
	w, _ := store.Watch(&WatchOptions{})
	testutils.ConsumeWatchEvents(w, stopCh)
	assert.Nil(store.Start(stopCh))

	name := "n1"
	node := &spec.WorkerNode{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	store.Add(node)
	var wg sync.WaitGroup
	// The writer launchs many write operations.
	writer := func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			r, err := store.Get(name)
			assert.Nil(err)
			assert.Nil(r.Wait(timeout))
			store.Update(r.List()[0])
		}
	}
	verifier := func() {
		defer wg.Done()
		r, err := store.Get(name)
		assert.Nil(err)
		assert.Nil(r.Wait(timeout))
		w, err := store.Update(r.List()[0])
		if err == nil { // otherwise, maybe conflict error
			// Maybe the object is still pending, but the target version committed.
			assert.Nil(w.Wait(timeout))
		}
	}
	for i := 0; i < 10; i++ {
		wg.Add(2)
		go writer()
		go verifier()
	}
	wg.Wait()
}

// Concurrent Update object, and expect the committed object matches the persisted object.
// See issue #93078 for details.
func TestPersistedStoreUpdateConsistence(t *testing.T) {
	assert := assert.New(t)
	persistPeriod := time.Millisecond * 10
	timeout := persistPeriod * 2
	opt := WithStorePersistPeriod(persistPeriod)
	persister := NewFakePersister()
	store0 := NewStore(spec.Scheme, spec.SchemeGroupVersion.WithKind(Kind(&spec.WorkerNode{})), "ns", &OneBatcher{"b"}, persister, opt)
	store := store0.(*persistedStore)
	stopCh := make(chan struct{})
	defer close(stopCh)
	w, _ := store.Watch(&WatchOptions{})
	testutils.ConsumeWatchEvents(w, stopCh)
	assert.Nil(store.Start(stopCh))

	name := "n1"
	node := &spec.WorkerNode{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Age: 1,
	}
	store.Add(node)
	var wg sync.WaitGroup
	for i := 0; i < 300; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n := 0; n < 100; n++ {
				r, err := store.Get(name)
				assert.Nil(err)
				assert.Nil(r.Wait(timeout))
				node0 := r.List()[0].DeepCopyObject().(*spec.WorkerNode)
				node0.Age = rand.Intn(100000)
				w, err := store.Update(node0)
				if err == nil {
					assert.Nil(w.Wait(timeout))
				}
			}
		}()
	}
	wg.Wait()
	r, err := store.Get(name)
	assert.Nil(err)
	assert.Nil(r.Wait(timeout))
	node0 := r.List()[0].(*spec.WorkerNode)
	nodep := persister.items["b"][0].(*spec.WorkerNode)
	assert.Equal(node0.Age, nodep.Age)
}

func TestPersistedStoreParallelRunDelete(t *testing.T) {
	assert := assert.New(t)
	stopCh := make(chan struct{})
	defer close(stopCh)
	persister := NewFakePersister()
	funcBatcher := FuncBatcher{}
	var n = 10000
	runtime.GOMAXPROCS(128)                                   // 多线程跑
	funcBatcher.KeyFunc = HashedNameFuncLabelValue("size", 1) // 尽量多的batcher， 增加出问题概率
	store0 := NewStore(spec.Scheme, spec.SchemeGroupVersion.WithKind(Kind(&spec.WorkerNode{})), "ns", &funcBatcher, persister, WithStorePersistPeriod(time.Millisecond*10))
	store := store0.(*persistedStore)
	w, _ := store.Watch(&WatchOptions{})
	testutils.ConsumeWatchEvents(w, stopCh)
	var nodes = make([]*spec.WorkerNode, 0, n)
	for i := 0; i < n; i++ {
		name := fmt.Sprintf("w.%d", i)
		label := fmt.Sprintf("xxl.%d", i)
		age := i
		node := &spec.WorkerNode{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
				Labels: map[string]string{
					"size": label,
				},
			},
			Age:  age,
			Nick: "alice",
		}
		_, err := store.Add(node)
		assert.Nil(err)
		nodes = append(nodes, node)
	}
	store.run() // 添加完成
	objs := store.listAll()
	assert.Equal(n, len(objs))
	assert.Equal(n, len(persister.items))
	for i := range nodes {
		node := nodes[i]
		node.Age++
		store.Update(node) // 增加一堆更新事件
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		fmt.Println("run")
		store.run()
		wg.Done()
	}()
	for i := 0; i < n; i++ {
		name := fmt.Sprintf("w.%d", i)
		wg.Add(1)
		go func(name string) {
			_, err := store.Delete(name)
			assert.Nil(err)
			wg.Done()
		}(name)
	}
	wg.Wait()
	store.run()
	objs = store.listAll()
	assert.Equal(0, len(objs))
	assert.Equal(n, len(persister.items)) // persister 确认删除batch
	for i := range persister.items {
		assert.Equal(0, len(persister.items[i])) // persister 确认删除batch
	}
}
