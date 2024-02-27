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
	"hash/fnv"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	glog "k8s.io/klog"
)

type encodingType int

const (
	// NOTE: for pb encoding, filter fields not work
	encodingPB   = encodingType(1)
	encodingJSON = encodingType(2)
)

// BatchObject is a batch of object.
type BatchObject struct {
	Name string `json:"name"`
	// Traits are some special properties parsed from object, to identify this batchObject. E.g: shardId in c2.
	Traits map[string]string `json:"traits,omitempty"`
	Items  pendingList       `json:"items"`

	deletings pendingList
}

// NewBatchObject creates a new batch object.
func NewBatchObject(name string) *BatchObject {
	return &BatchObject{
		Name:      name,
		Items:     make(pendingList, 0),
		deletings: make(pendingList, 0),
	}
}

// IsDrity persist only if the batch is dirty
func (b *BatchObject) IsDrity() bool {
	if len(b.deletings) > 0 {
		return true
	}
	for _, items := range b.Items {
		if items.IsPending() {
			return true
		}
	}
	return false
}

type batchObjectUnserializer struct {
	Name    string            `json:"name"`
	Items   []json.RawMessage `json:"items"`
	Traits  map[string]string `json:"traits,omitempty"`
	creator NewObjectFunc
}

func (s *batchObjectUnserializer) unmarshal(bs []byte, encoding encodingType) (*BatchObject, error) {
	// check to be compatible
	if encoding == encodingPB && isPBBytes(bs) {
		return s.unmarshalPB(bs)
	}
	return s.unmarshalJSON(bs)
}

func (s *batchObjectUnserializer) unmarshalJSON(bs []byte) (*BatchObject, error) {
	if err := json.Unmarshal(bs, s); err != nil {
		return nil, err
	}
	pendings := make([]*pendingObject, 0)
	for _, raw := range s.Items {
		obj := s.creator()
		if err := json.Unmarshal(raw, obj); err != nil {
			return nil, err
		}
		pending := initPendingObject(obj)
		pendings = append(pendings, pending)
	}
	return &BatchObject{Name: s.Name, Items: pendings, Traits: s.Traits}, nil
}

// Batcher batch a list of objects into batches.
type Batcher interface {
	Batch([]*pendingObject) ([]*BatchObject, error)
}

// OneBatcher batch all object into one batch.
type OneBatcher struct {
	Name string
}

// Batch batch objects.
func (b *OneBatcher) Batch(items []*pendingObject) ([]*BatchObject, error) {
	name := b.Name
	if name == "" {
		name = "oneBatch"
	}
	batch := NewBatchObject(name)
	for _, item := range items {
		if item.Deleting() {
			batch.deletings = append(batch.deletings, item)
		} else {
			batch.Items = append(batch.Items, item)
		}
	}
	return []*BatchObject{batch}, nil
}

// KeyFunc format a key for an object.
type KeyFunc func(runtime.Object) (string, error)

// TraitsFunc parse traits from object.
type TraitsFunc func(runtime.Object) (map[string]string, error)

// KeyFuncLabelValue create a KeyFunc by label value.
func KeyFuncLabelValue(key string) KeyFunc {
	return func(obj runtime.Object) (string, error) {
		meta, err := meta.Accessor(obj)
		if err != nil {
			return "", fmt.Errorf("object has no meta: %v", err)
		}
		name := meta.GetName()
		lbls := meta.GetLabels()
		if lbls == nil || lbls[key] == "" {
			return "", fmt.Errorf("object has no label: %s, %s", name, key)
		}
		return lbls[key], nil
	}
}

// HashedNameFuncLabelValue labelValue+hash
func HashedNameFuncLabelValue(key string, hashNum uint32) KeyFunc {
	return KeyFuncHashName(hashNum, func(meta metav1.Object) (string, error) {
		lbls := meta.GetLabels()
		if lbls == nil || lbls[key] == "" {
			return "", fmt.Errorf("object has no label: %s, %s", meta.GetName(), key)
		}
		return lbls[key], nil
	})
}

// KeyFuncHashName return $prefixFn(obj)-hash($name) % hashNum
func KeyFuncHashName(hashNum uint32, prefixFn func(metav1.Object) (string, error)) KeyFunc {
	return func(obj runtime.Object) (string, error) {
		meta, err := meta.Accessor(obj)
		if err != nil {
			return "", fmt.Errorf("object has no meta: %v", err)
		}
		name := meta.GetName()
		hash := fnv.New32a()
		hash.Write([]byte(name))
		hashValue := hash.Sum32()
		hashRemainder := hashValue % hashNum
		prefix, err := prefixFn(meta)
		if err != nil {
			return "", fmt.Errorf("get meta prefix string error: %v", err)
		}
		return fmt.Sprintf("%s-%d", prefix, hashRemainder), nil
	}
}

// TraitsFuncLabels create a TraitsFunc by a list of labels.
func TraitsFuncLabels(keys ...string) TraitsFunc {
	return func(obj runtime.Object) (map[string]string, error) {
		meta, err := meta.Accessor(obj)
		if err != nil {
			return nil, fmt.Errorf("object has no meta: %v", err)
		}
		name := meta.GetName()
		lbls := meta.GetLabels()
		if lbls == nil {
			return nil, fmt.Errorf("object has empty labels: %s", name)
		}
		kvs := map[string]string{}
		for _, k := range keys {
			if lbls[k] == "" {
				return nil, fmt.Errorf("object has no label: %s, %s", name, k)
			}
			kvs[k] = lbls[k]
		}
		return kvs, nil
	}
}

// FuncBatcher batch objects by a custom function.
type FuncBatcher struct {
	TraitsFunc TraitsFunc
	KeyFunc    KeyFunc
}

// Batch batch objects. Try the best to batch all objects, skip these failed objects.
func (b *FuncBatcher) Batch(items []*pendingObject) ([]*BatchObject, error) {
	batches := map[string]*BatchObject{}
	for _, item := range items {
		obj := item.GetLatestObject()
		k, err := b.KeyFunc(obj)
		if err != nil {
			glog.Errorf("Batch object by keyFunc error: %s, %v", item.name, err)
			continue
		}
		batch, ok := batches[k]
		if !ok {
			batch = NewBatchObject(k)
			if b.TraitsFunc != nil {
				// NOTE: parse the traits by the first object, we assume all objects in 1 batch must have the same traits.
				traits, err := b.TraitsFunc(obj)
				if err != nil {
					glog.Errorf("Parse traits from object error: %s, %v", item.name, err)
					continue
				}
				batch.Traits = traits
			}
			batches[k] = batch
		}
		if item.Deleting() {
			batch.deletings = append(batch.deletings, item)
		} else {
			batch.Items = append(batch.Items, item)
		}
	}
	return b.toBatchList(batches), nil
}

func (b *FuncBatcher) toBatchList(batches map[string]*BatchObject) []*BatchObject {
	batchList := make([]*BatchObject, 0, len(batches))
	for _, batch := range batches {
		batchList = append(batchList, batch)
	}
	return batchList
}
