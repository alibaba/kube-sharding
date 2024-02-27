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
	"sync"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	glog "k8s.io/klog"
)

var (
	errConflict = errors.New("target pendingObject conflict")
)

type pendingObject struct {
	name    string
	kind    string // for log only
	mutex   *sync.RWMutex
	event   *watch.Event
	current runtime.Object
	deleted bool
}

type pendingList []*pendingObject

func pendingObjectName(i interface{}) (string, error) {
	return i.(*pendingObject).name, nil
}

func newPendingObject(obj runtime.Object, gvk *schema.GroupVersionKind) *pendingObject {
	kind := ""
	if gvk != nil {
		kind = gvk.Kind
	}
	name := MetaName(obj)
	return &pendingObject{
		name:    name,
		kind:    kind,
		mutex:   new(sync.RWMutex),
		current: nil,
		event:   &watch.Event{Type: watch.Added, Object: obj},
		deleted: false,
	}
}

func initPendingObject(obj runtime.Object) *pendingObject {
	name := MetaName(obj)
	return &pendingObject{
		name:    name,
		kind:    Kind(obj),
		mutex:   new(sync.RWMutex),
		current: obj,
	}
}

func (obj *pendingObject) IsPending() bool {
	return obj.GetEvent() != nil
}

func (obj *pendingObject) RequestUpdate(targetObj runtime.Object, meta *MetaInfo) (uint64, error) {
	nextVer, _ := MetaResourceVersion(targetObj)
	obj.mutex.Lock()
	defer obj.mutex.Unlock()
	curObj := obj.getLatest()
	curVer, _ := MetaResourceVersion(curObj)
	deleting := obj.event != nil && obj.event.Type == watch.Deleted
	if nextVer < curVer || deleting { // if deleting, reject any other Update requests.
		return 0, errConflict
	}
	mergeReadOnlyFields(curObj, targetObj)
	retVer := meta.IncVersionSeed()
	_, err := SetMetaResourceVersion(targetObj, retVer)
	if err != nil {
		return 0, err
	}
	evt := &watch.Event{Type: watch.Modified, Object: targetObj}
	obj.event = evt
	glog.Infof("RequestUpdate to pending object: %s/%s, details: %s", obj.kind, obj.name, Describe(evt.Object))
	return retVer, nil
}

// Don't check pending state, `Deleted` event overwrites all other events.
func (obj *pendingObject) RequestDelete(meta *MetaInfo) (uint64, error) {
	obj.mutex.Lock()
	defer obj.mutex.Unlock()
	cur := obj.current
	// Delete before Add committed.
	// NOTE: If we add and delete an object, there will be 1 event generated.
	if cur == nil {
		// if `current==nil` which means the object is awaiting `Add`.
		// Set object in Delete event as the pending object.
		cur = obj.event.Object
	}
	retVer := meta.IncVersionSeed()
	_, err := SetMetaResourceVersion(cur, retVer)
	if err != nil {
		return 0, err
	}
	evt := &watch.Event{Type: watch.Deleted, Object: cur}
	obj.event = evt
	glog.Infof("RequestDelete to pending object: %s/%s, details: %s", obj.kind, obj.name, Describe(evt.Object))
	return retVer, nil
}

func (obj *pendingObject) GetEvent() *watch.Event {
	obj.mutex.RLock()
	defer obj.mutex.RUnlock()
	return obj.event
}

func (obj *pendingObject) Deleting() bool {
	evt := obj.GetEvent()
	return evt != nil && evt.Type == watch.Deleted
}

func (obj *pendingObject) Deleted() bool {
	obj.mutex.RLock()
	defer obj.mutex.RUnlock()
	return obj.deleted
}

// Deprecated, only used in test now.
func (obj *pendingObject) GetObject() runtime.Object {
	obj.mutex.RLock()
	defer obj.mutex.RUnlock()
	if obj.current != nil {
		return obj.current
	}
	return obj.event.Object
}

func (obj *pendingObject) GetLatestObject() runtime.Object {
	obj.mutex.RLock()
	defer obj.mutex.RUnlock()
	return obj.getLatest()
}

func (obj *pendingObject) getLatest() runtime.Object {
	if obj.event != nil && obj.event.Object != nil {
		return obj.event.Object
	}
	return obj.current
}

func (obj *pendingObject) GetCurrent() runtime.Object {
	obj.mutex.RLock()
	defer obj.mutex.RUnlock()
	return obj.current
}

func (obj *pendingObject) Commit(expectDeleting bool) *watch.Event {
	obj.mutex.Lock()
	defer obj.mutex.Unlock()
	evt := obj.event
	if evt.Type == watch.Deleted && !expectDeleting {
		glog.Warningf("Pending object commit failed by expection fail: %s", obj.name)
		return nil
	}
	if evt.Type == watch.Added || evt.Type == watch.Modified {
		obj.current = evt.Object
	}
	obj.deleted = evt.Type == watch.Deleted
	obj.event = nil
	glog.Infof("Pending object committed: %s/%s, %s, details: %s", obj.kind, obj.name, evt.Type, Describe(obj.current))
	return evt
}

func (obj *pendingObject) MarshalJSON() ([]byte, error) {
	o := obj.GetLatestObject()
	return FilterMarshal(o)
}
