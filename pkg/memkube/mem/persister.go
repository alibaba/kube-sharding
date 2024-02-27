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
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/alibaba/kube-sharding/pkg/ago-util/fstorage"
	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	"github.com/gogo/protobuf/proto"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	glog "k8s.io/klog"
)

const (
	metaFileName = "__metainfo__"
)

// NewObjectFunc create an empty object.
type NewObjectFunc func() runtime.Object

// Persister persists a list of batch objects.
type Persister interface {
	Persist(*BatchObject) error
	PersistMeta(*MetaInfo) error
	Recover(fn NewObjectFunc) ([]*BatchObject, *MetaInfo, error)
	Close() error
}

type negotiatedPersister interface {
	NegotiateEncoding(o runtime.Object)
}

// FakePersister is a persister.
type FakePersister struct {
	items map[string][]runtime.Object
	mutex *sync.RWMutex
}

// NewFakePersister create a fake persister
func NewFakePersister() *FakePersister {
	return &FakePersister{
		items: make(map[string][]runtime.Object),
		mutex: new(sync.RWMutex),
	}
}

// PersistMeta persist meta info
func (p *FakePersister) PersistMeta(*MetaInfo) error {
	return nil
}

// Persist persists batch object.
func (p *FakePersister) Persist(b *BatchObject) error {
	objs := make([]runtime.Object, 0)
	for _, pending := range b.Items {
		objs = append(objs, pending.GetLatestObject().DeepCopyObject())
	}
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.items[b.Name] = objs
	return nil
}

// Recover reload all objects.
func (p *FakePersister) Recover(fn NewObjectFunc) ([]*BatchObject, *MetaInfo, error) {
	return []*BatchObject{}, &MetaInfo{}, nil
}

// Close close persister
func (p *FakePersister) Close() error {
	return nil
}

// TraitsMatcher check if traits matches in persiter recover.
type TraitsMatcher func(map[string]string) bool

// FsPersister is a file system persister which used fstorage.
type FsPersister struct {
	fsl fstorage.Location

	matcher   TraitsMatcher
	lastHashs map[string]string
	lastMeta  *MetaInfo
	mutex     *sync.RWMutex
	namespace string
	gvk       schema.GroupVersionKind
	encoding  encodingType
}

// NewFsPersister create a new FsPersister.
func NewFsPersister(url, backupURL string, namespace string, gvk schema.GroupVersionKind, matcher TraitsMatcher) (*FsPersister, error) {
	fsl, err := fstorage.CreateWithBackup(url, backupURL)
	if err != nil {
		glog.Errorf("Create fstorage from url error: %s, %v", url, err)
		return nil, err
	}
	fsl, err = fsl.NewSubLocation(namespace)
	if err != nil {
		glog.Errorf("Create sub location at namespace error: %s, %s, %v", url, namespace, err)
		return nil, err
	}

	fsl, err = fsl.NewSubLocation(strings.ToLower(gvk.Kind))
	if err != nil {
		glog.Errorf("Create sub location at kind (%s) error: %s, %s, %v", gvk.Kind, url, namespace, err)
		return nil, err
	}

	return &FsPersister{
		fsl:       fsl,
		matcher:   matcher,
		lastHashs: map[string]string{},
		namespace: namespace,
		mutex:     new(sync.RWMutex),
		gvk:       gvk,
		encoding:  encodingJSON,
	}, nil
}

// PersistMeta persist meta info
func (p *FsPersister) PersistMeta(meta *MetaInfo) error {
	if p.lastMeta != nil && p.lastMeta.Equal(meta) {
		return nil
	}
	bs, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("marshal meta error: %v", err)
	}
	fs, err := p.fsl.Create(metaFileName)
	if err != nil {
		return fmt.Errorf("create meta file fstorage error: %v", err)
	}
	defer fs.Close()
	if err := fs.Write(bs); err != nil {
		return err
	}
	p.lastMeta = meta
	glog.V(4).Infof("Write meta success: %+v", *meta)
	return nil
}

// Persist persists batch object.
func (p *FsPersister) Persist(obj *BatchObject) error {
	if obj.Name == "" {
		return errors.New("batchObject name is empty")
	}
	if len(obj.Items) == 0 {
		// All items in the batch will be mark deleted first, that's the chance we can remove the fstorage file.
		return p.removeOne(obj.Name)
	}
	Metrics.Get(MetricPersistCountInBatch).Observe(float64(len(obj.Items)), p.namespace, obj.Name, p.gvk.Kind)
	// marshal if the batch is dirty, to optimize cpu usage
	if p.lastHash(obj.Name) != "" && !obj.IsDrity() {
		return nil
	}
	startTime := time.Now()
	bs, err := obj.Marshal(p.encoding)
	if err != nil {
		return fmt.Errorf("marshal batchObject error: %v", err)
	}
	sig, err := utils.Signature(bs)
	if err != nil {
		return fmt.Errorf("hash signature error: %v", err)
	}
	Metrics.Get(MetricPersistMarshalTime).Elapsed(startTime, p.namespace, obj.Name, p.gvk.Kind)
	if sig == p.lastHash(obj.Name) {
		return nil
	}
	startTime = time.Now()
	defer Metrics.Get(MetricPersistWriteTime).Elapsed(startTime, p.namespace, obj.Name, p.gvk.Kind)
	fs, err := p.fsl.Create(obj.Name)
	if err != nil {
		return fmt.Errorf("create fstorage error: %v", err)
	}
	defer fs.Close()
	glog.V(4).Infof("Write batchObject to fstorage: %s, count: %d, uncompressed bytes: %d", obj.Name, len(obj.Items), len(bs))
	fs = fstorage.MakeCompressiveNotify(fs, func(bs []byte) {
		Metrics.Get(MetricPersistFileSize).Observe(float64(len(bs)), p.namespace, obj.Name, p.gvk.Kind)
	})
	if err := fs.Write(bs); err != nil {
		return err
	}
	// otherwise update the persisted hash signature.
	p.setLastHash(obj.Name, sig)
	glog.V(3).Infof("Persist batchObject %s success, items: %v", obj.Name, len(obj.Items))
	return nil
}

// Recover reload all objects.
func (p *FsPersister) Recover(fn NewObjectFunc) ([]*BatchObject, *MetaInfo, error) {
	meta, err := p.recoverMeta()
	if err != nil {
		return nil, nil, err
	}
	names, err := p.fsl.List()
	if err != nil {
		return nil, nil, err
	}
	batches := make([]*BatchObject, 0)
	for _, name := range names {
		if strings.HasPrefix(name, ".") || (strings.HasPrefix(name, "__") && strings.HasSuffix(name, "__")) {
			continue
		}
		batch, err := p.recoverOne(fn, name)
		if err != nil {
			return nil, nil, err
		}
		if p.matcher == nil || p.matcher(batch.Traits) {
			batches = append(batches, batch)
		}
	}
	return batches, meta, nil
}

func (p *FsPersister) recoverMeta() (*MetaInfo, error) {
	fs, err := p.fsl.New(metaFileName)
	if err != nil {
		return nil, fmt.Errorf("new meta fstorage error: %v", err)
	}
	defer fs.Close()
	if exists, err := fs.Exists(); err != nil {
		return nil, fmt.Errorf("check meta exists failed: %v", err)
	} else if !exists {
		return NewMetaInfo(), nil
	}
	bs, err := fs.Read()
	if err != nil {
		return nil, fmt.Errorf("read meta data failed: %v", err)
	}
	meta := NewMetaInfo()
	if err = json.Unmarshal(bs, meta); err != nil {
		return nil, fmt.Errorf("unmarshal meta failed: %v", err)
	}
	return meta, nil
}

func (p *FsPersister) recoverOne(fn NewObjectFunc, name string) (*BatchObject, error) {
	fs, err := p.fsl.Create(name)
	if err != nil {
		return nil, fmt.Errorf("create fstorage error: %v", err)
	}
	fs = fstorage.MakeCompressive(fs)
	bs, err := fs.Read()
	if err != nil {
		return nil, fmt.Errorf("read data from storage error: %s, %v", name, err)
	}
	us := &batchObjectUnserializer{creator: fn}
	batch, err := us.unmarshal(bs, p.encoding)
	if err != nil {
		return nil, fmt.Errorf("unmarshal batchObject error: %s, %v", us.Name, err)
	}
	sig, err := utils.Signature(bs)
	if err != nil {
		return nil, fmt.Errorf("hash signature error: %v", err)
	}
	// set the initialized persisted hash.
	p.setLastHash(name, sig)
	return batch, err
}

// Do the real remove or just rename ?
func (p *FsPersister) removeOne(name string) error {
	fs, err := p.fsl.Create(name)
	if err != nil {
		return fmt.Errorf("create fstorage error: %v", err)
	}
	exist, err := fs.Exists()
	if err != nil {
		return fmt.Errorf("check fstorage exists error: %s, %v", name, err)
	}
	if exist {
		glog.Infof("Remove fstorage file: %s, %s", p.gvk.String(), name)
		return fs.Remove()
	}
	return nil
}

func (p *FsPersister) lastHash(name string) string {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.lastHashs[name]
}

func (p *FsPersister) setLastHash(name, h string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.lastHashs[name] = h
}

// Close close persister
func (p *FsPersister) Close() error {
	return p.fsl.Close()
}

// NegotiateEncoding negotiate encoding by object
func (p *FsPersister) NegotiateEncoding(o runtime.Object) {
	p.encoding = encodingJSON
	if _, ok := o.(proto.Message); ok {
		p.encoding = encodingPB
	}
	glog.Infof("Negotiate persister encoding to [%d] for %s", p.encoding, Kind(o))
}
