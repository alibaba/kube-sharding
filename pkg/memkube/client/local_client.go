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

package client

import (
	"fmt"
	"sync"
	"time"

	"github.com/alibaba/kube-sharding/pkg/memkube/mem"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	glog "k8s.io/klog"
)

// LocalClient used in mem-kube server to operate objects locally.
type LocalClient struct {
	mutex   *sync.RWMutex
	stores  map[string]mem.Store
	timeout time.Duration
}

func storeKey(resource, ns string) string {
	return ns + "/" + resource
}

var _ Client = &LocalClient{}

// NewLocalClient creates a new local client.
func NewLocalClient(timeout time.Duration) *LocalClient {
	return &LocalClient{
		mutex:   new(sync.RWMutex),
		stores:  make(map[string]mem.Store),
		timeout: timeout,
	}
}

// Register regists a store.
func (c *LocalClient) Register(resource, ns string, store mem.Store) {
	k := storeKey(resource, ns)
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.stores[k] = store
}

// Start start all stores.
func (c *LocalClient) Start(stopCh <-chan struct{}) error {
	for _, store := range c.stores {
		err := store.Start(stopCh)
		if err != nil {
			return err
		}
	}
	return nil
}

// List gets a list of objects.
func (c *LocalClient) List(resource, ns string, opts metav1.ListOptions, list runtime.Object) error {
	store, err := c.getStore(resource, ns)
	if err != nil {
		return err
	}
	lblSelector, err := metav1.ParseToLabelSelector(opts.LabelSelector)
	if err != nil {
		return err
	}
	selector, err := metav1.LabelSelectorAsSelector(lblSelector)
	if err != nil {
		return err
	}
	items, resourceVersion, err := store.List(selector)
	if err != nil {
		return err
	}
	if err = meta.SetList(list, items); err != nil {
		return err
	}
	_, err = mem.SetMetaResourceVersion(list, resourceVersion)
	return err
}

// Get gets an object.
// NOTE: the return value is the internal object, not deep copied.
func (c *LocalClient) Get(resource, ns, name string, opts metav1.GetOptions, into runtime.Object) (runtime.Object, error) {
	store, err := c.getStore(resource, ns)
	if err != nil {
		return nil, err
	}
	result, err := store.Get(name)
	if err != nil {
		return nil, err
	}
	err = result.Wait(c.timeout)
	if err != nil {
		return nil, err
	}
	return result.List()[0], nil
}

func (c *LocalClient) getStore(resource, ns string) (mem.Store, error) {
	k := storeKey(resource, ns)
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	store, ok := c.stores[k]
	if !ok {
		return nil, fmt.Errorf("not found store: %s", k)
	}
	return store, nil
}

// Create create an object.
func (c *LocalClient) Create(resource, ns string, object runtime.Object) (runtime.Object, error) {
	store, err := c.getStore(resource, ns)
	if err != nil {
		return nil, err
	}
	result, err := store.Add(object)
	if err != nil {
		return nil, err
	}
	err = result.Wait(c.timeout)
	if err != nil {
		return nil, err
	}
	return result.Object(), nil
}

// Update update an object.
func (c *LocalClient) Update(resource, ns string, object runtime.Object) (runtime.Object, error) {
	store, err := c.getStore(resource, ns)
	if err != nil {
		return nil, err
	}
	result, err := store.Update(object)
	if err != nil {
		return nil, err
	}
	err = result.Wait(c.timeout)
	if err != nil {
		return nil, err
	}
	return result.Object(), nil
}

// Delete deletes an object. If the object has finalizers, update the delete timestamp only.
func (c *LocalClient) Delete(resource, ns, name string, opts *metav1.DeleteOptions) error {
	store, err := c.getStore(resource, ns)
	if err != nil {
		return err
	}
	result, err := store.Get(name)
	if err != nil {
		return err
	}
	err = result.Wait(c.timeout)
	if err != nil {
		return err
	}
	obj := result.List()[0].DeepCopyObject()
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return err
	}
	finalizers := accessor.GetFinalizers()
	if len(finalizers) > 0 {
		if accessor.GetDeletionTimestamp() != nil {
			return nil
		}
		now := metav1.Now()
		accessor.SetDeletionTimestamp(&now)
		glog.Infof("Update object deletion timestamp to delete: %s", mem.Describe(obj))
		_, err := c.Update(resource, ns, obj)
		return err
	}
	wresult, err := store.Delete(name)
	if err != nil {
		return err
	}
	return wresult.Wait(c.timeout)
}

// Watch get a watch.Interface to watch on an object.
func (c *LocalClient) Watch(resource, ns string, opts *mem.WatchOptions) (watch.Interface, error) {
	store, err := c.getStore(resource, ns)
	if err != nil {
		return nil, err
	}
	return store.Watch(opts)
}
