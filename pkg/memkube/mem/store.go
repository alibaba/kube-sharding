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
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/ivpusic/grpool"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	glog "k8s.io/klog"
)

var (
	// ErrObjectInvalid the object to add/update is invalid
	ErrObjectInvalid = errors.New("object is invalid")
)

// WatchOptions options for watch
type WatchOptions struct {
	metav1.ListOptions
	// Set to true if watch in same process, default true
	LocalWatch *bool
	// Identify source
	Source string
}

// Store stores memory objects.
type Store interface {
	Start(stopCh <-chan struct{}) error
	List(labels.Selector) ([]runtime.Object, uint64, error)
	Watch(*WatchOptions) (watch.Interface, error)
	Get(name string) (ReadResult, error)
	Peek(name string) (runtime.Object, error)
	Add(obj runtime.Object) (WriteResult, error)
	Update(obj runtime.Object) (WriteResult, error)
	Delete(name string) (WriteResult, error)
}

type persistedStore struct {
	name        string
	indexer     cache.Indexer
	broadcaster WatchBroadcaster
	batcher     Batcher
	persister   Persister
	namespace   string
	scheme      *runtime.Scheme
	gvk         schema.GroupVersionKind
	gvr         schema.GroupVersionResource // to generate api error

	persistPeriod time.Duration

	grpool *grpool.Pool
	meta   *MetaInfo
}

// StoreOption defines the options for store.
type StoreOption func(*persistedStore) *persistedStore

// WithStorePersistPeriod set persist period. Default 500ms
func WithStorePersistPeriod(period time.Duration) StoreOption {
	return func(store *persistedStore) *persistedStore {
		store.persistPeriod = period
		return store
	}
}

// WithStoreConcurrentPersist set persist concurrent options.
func WithStoreConcurrentPersist(worker int, qsize int) StoreOption {
	return func(store *persistedStore) *persistedStore {
		store.grpool = grpool.NewPool(worker, qsize)
		return store
	}
}

// WithSilentWatchConsumer consumes all events from this store and drop the events.
func WithSilentWatchConsumer(stopCh <-chan struct{}) StoreOption {
	return func(store *persistedStore) *persistedStore {
		w, _ := store.Watch(&WatchOptions{})
		ch := w.ResultChan()
		go func() {
			for {
				select {
				case <-stopCh:
					return
				case w := <-ch:
					glog.V(5).Infof("Silient watch event: %+v", w)
				}
			}
		}()
		return store
	}
}

// NewStore creates a store.
func NewStore(scheme *runtime.Scheme, gvk schema.GroupVersionKind, namespace string, batcher Batcher, persister Persister, opts ...StoreOption) Store {
	plural, _ := meta.UnsafeGuessKindToResource(gvk)
	store := &persistedStore{
		name:        namespace + "_" + gvk.Kind,
		scheme:      scheme,
		gvk:         gvk,
		gvr:         plural,
		indexer:     cache.NewIndexer(pendingObjectName, cache.Indexers{}),
		broadcaster: NewCacheWatchBroadcaster(cacheCapacity, gvk.Kind),
		namespace:   namespace,
		batcher:     batcher,
		persister:   persister,
		meta:        NewMetaInfo(),

		persistPeriod: time.Millisecond * 500,
	}
	for _, opt := range opts {
		opt(store)
	}
	if store.grpool == nil {
		store.grpool = grpool.NewPool(100, 50)
	}
	store.negotiateEncoding()
	return store
}

func (store *persistedStore) Start(stopCh <-chan struct{}) error {
	glog.Infof("Start store for (%s) with period %s", store.name, store.persistPeriod)
	if err := store.recover(); err != nil {
		return fmt.Errorf("recover store error: %v", err)
	}
	go func() {
		<-stopCh
		store.grpool.Release()
		store.broadcaster.Shutdown()
	}()
	go func() {
		wait.Until(store.run, store.persistPeriod, stopCh)
		store.persister.Close()
		glog.Infof("Store run stopped, %s", store.name)
	}()
	return nil
}

func (store *persistedStore) List(selector labels.Selector) ([]runtime.Object, uint64, error) {
	items := store.indexer.List()
	objs := make([]runtime.Object, 0, len(items))
	resourceVersion := store.meta.CurrentVersion()
	for _, item := range items {
		cur := item.(*pendingObject).current
		if cur != nil {
			accessor, err := meta.Accessor(cur)
			if err != nil {
				glog.Errorf("Get meta accessor error: %v", err)
				continue
			}
			if selector == nil || selector.Matches(labels.Set(accessor.GetLabels())) {
				objs = append(objs, cur)
			}
		}
	}
	return objs, resourceVersion, nil
}

func (store *persistedStore) Watch(opts *WatchOptions) (watch.Interface, error) {
	if opts.LocalWatch == nil {
		b := true
		opts.LocalWatch = &b
	}
	return store.broadcaster.Watch(opts, SelectionPredicate{})
}

func (store *persistedStore) Get(name string) (ReadResult, error) {
	item, exists, err := store.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, kerrors.NewNotFound(store.gvr.GroupResource(), name)
	}
	return newReadResult(store.checkTime(), []*pendingObject{item.(*pendingObject)}), nil
}

func (store *persistedStore) Peek(name string) (runtime.Object, error) {
	item, exists, err := store.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}
	return item.(*pendingObject).current, nil
}

func (store *persistedStore) Add(obj runtime.Object) (WriteResult, error) {
	key := MetaName(obj)
	if key == "" {
		return nil, ErrObjectInvalid
	}
	if _, exists, err := store.indexer.GetByKey(key); err != nil {
		return nil, err
	} else if exists {
		return nil, kerrors.NewAlreadyExists(store.gvr.GroupResource(), key)
	}
	obj0 := obj.DeepCopyObject()
	// NOTE: if failed, the version seed skipped
	ver := store.meta.IncVersionSeed()
	if err := TweakObjectMetas(obj0, ver); err != nil {
		return nil, err
	}
	if err := TweakTypeMetas(store.gvk, obj0); err != nil {
		return nil, err
	}
	pending := newPendingObject(obj0, &store.gvk)
	if err := store.indexer.Add(pending); err != nil {
		return nil, err
	}
	glog.Infof("Add mem object: %s, %s", store.name, Stringify(obj0))
	return newWriteResult(store.gvr.GroupResource(), store.checkTime(), pending, ver), nil
}

func (store *persistedStore) Update(obj runtime.Object) (WriteResult, error) {
	key := MetaName(obj)
	if key == "" {
		return nil, ErrObjectInvalid
	}
	item, exists, err := store.indexer.GetByKey(key)
	if err != nil {
		return nil, fmt.Errorf("get pending object failed: %s, %v/%v", key, err, exists)
	}
	if !exists {
		return nil, kerrors.NewNotFound(store.gvr.GroupResource(), key)
	}
	pending := item.(*pendingObject)
	curObj := pending.GetLatestObject()
	if eq, err := IsUIDEqual(curObj, obj); err != nil || !eq {
		return nil, kerrors.NewResourceExpired(fmt.Sprintf("object uid not equal: %s, err: %v", key, err))
	}
	glog.Infof("Update mem object: %s, %s, obj: %s", store.name, key, Describe(obj))
	retVer, err := pending.RequestUpdate(obj.DeepCopyObject(), store.meta)
	if err != nil {
		if err == errConflict {
			// Operation cannot be fulfilled on ...
			return nil, kerrors.NewConflict(store.gvr.GroupResource(), key, nil)
		}
		return nil, err
	}
	return newWriteResult(store.gvr.GroupResource(), store.checkTime(), pending, retVer), nil
}

func (store *persistedStore) Delete(name string) (WriteResult, error) {
	item, exists, err := store.indexer.GetByKey(name)
	if err != nil {
		return nil, fmt.Errorf("get pending object failed: %s, %v/%v", name, err, exists)
	}
	if !exists {
		return nil, kerrors.NewNotFound(store.gvr.GroupResource(), name)
	}
	// Still to increase the version seeed
	glog.Infof("Delete a mem object: %s, %s", store.name, name)
	pending := item.(*pendingObject)
	pending.RequestDelete(store.meta)
	return newWriteResult(store.gvr.GroupResource(), store.checkTime(), pending, maxVersion), nil
}

func (store *persistedStore) run() {
	startTime := time.Now()
	mi := store.meta.DeepCopy() // thread-safe deepcopy
	if err := store.persister.PersistMeta(mi); err != nil {
		glog.Errorf("Persist meta info failed: %s, %v", store.name, err)
		return
	}
	objs := store.listAll()
	batchObjs, _ := store.batcher.Batch(objs)
	store.grpool.WaitCount(len(batchObjs))
	for _, batchObj0 := range batchObjs {
		batchObj := batchObj0
		store.grpool.JobQueue <- func() {
			defer store.grpool.JobDone()
			if nil == store.persistBatch(batchObj) {
				store.commit(batchObj.Items, batchObj.deletings)
			}
		}
	}
	store.grpool.WaitAll()
	if elapsed := time.Since(startTime); elapsed > store.persistPeriod*2 {
		glog.Warningf("Store run once slow: %s, %v", store.name, elapsed)
	}
	Metrics.Get(MetricStoreRunTime).Elapsed(startTime, store.name, store.namespace, store.gvk.Kind)
}

// shadow copy objects
func (store *persistedStore) listAll() []*pendingObject {
	items := store.indexer.List()
	objs := make([]*pendingObject, 0, len(items))
	for _, item := range items {
		objs = append(objs, item.(*pendingObject))
	}
	// Make a stable sorting, to fix random order in persister marshal.
	sort.SliceStable(objs, func(i, j int) bool {
		return objs[i].name < objs[j].name
	})
	return objs
}

func (store *persistedStore) persistBatch(batchObj *BatchObject) error {
	if err := store.persister.Persist(batchObj); err != nil {
		glog.Errorf("Persist object error: %s, %s, %v", store.namespace, batchObj.Name, err)
		return err
	}
	return nil
}

func (store *persistedStore) commit(objs []*pendingObject, deletings []*pendingObject) {
	for _, obj := range objs {
		store.commitOne(obj, false)
	}
	for _, obj := range deletings {
		store.commitOne(obj, true)
	}
}

func (store *persistedStore) commitOne(obj *pendingObject, deleting bool) {
	if !obj.IsPending() {
		return
	}
	evt := obj.Commit(deleting)
	if evt == nil {
		return
	}
	if evt.Type == watch.Deleted {
		if err := store.indexer.Delete(obj); err != nil {
			glog.Errorf("Delete object error: %v, store %s", err, store.name)
			return
		}
	}
	startTime := time.Now()
	store.broadcaster.Action(*evt)
	if elapsed := time.Since(startTime); elapsed > time.Second {
		glog.Warningf("Broadcast watch events slow: %s, %v", store.name, elapsed)
	}
}

func (store *persistedStore) checkTime() time.Duration {
	return store.persistPeriod / 2
}

func (store *persistedStore) recover() error {
	batches, mi, err := store.persister.Recover(func() runtime.Object {
		obj, err := store.scheme.New(store.gvk)
		if err != nil {
			glog.Fatalf("New object from scheme error: %v", err)
		}
		return obj
	})
	if err != nil {
		return err
	}
	store.meta = mi
	cnt := 0
	maxVersion := mi.VersionSeed
	for _, batch := range batches {
		cnt += len(batch.Items)
		for _, item := range batch.Items {
			if err := store.indexer.Add(item); err != nil {
				glog.Errorf("Add recovered pendingObject error: %s, %v", item.name, err)
				return err
			}
			// for compatible reason, the MetaInfo maybe empty
			ver, _ := MetaResourceVersion(item.current)
			if ver > maxVersion {
				maxVersion = ver
			}
		}
	}
	if maxVersion > mi.VersionSeed {
		glog.Infof("Fix MetaInfo.MaxVersion by %v -> %v", mi.VersionSeed, maxVersion)
		mi.VersionSeed = maxVersion
	}
	glog.Infof("Recover store [%s] %d items in %d batches, with meta: %+v", store.name, cnt, len(batches), *mi)
	return nil
}

func (store *persistedStore) negotiateEncoding() {
	np, ok := store.persister.(negotiatedPersister)
	if !ok {
		return
	}
	obj, err := store.scheme.New(store.gvk)
	if err != nil {
		glog.Fatalf("New object from scheme error: %v", err)
	}
	np.NegotiateEncoding(obj)
}
