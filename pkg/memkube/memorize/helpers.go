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

package memorize

import (
	"errors"
	"fmt"

	"github.com/alibaba/kube-sharding/pkg/memkube/mem"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// FuncReaderWriter wraps functions as a ReaderWriter.
type FuncReaderWriter struct {
	WriteFunc func(runtime.Object) error
	WrapFunc  func(parent runtime.Object, parentGvk schema.GroupVersionKind, obj runtime.Object) (runtime.Object, error)
	ListFunc  func(runtime.Object, ListMetas) ([]runtime.Object, error)
	Name      string
}

// Write writes an object.
func (f FuncReaderWriter) Write(obj runtime.Object) error {
	return f.WriteFunc(obj)
}

// Wrap wraps an objects.
func (f FuncReaderWriter) Wrap(parent runtime.Object, parentGvk schema.GroupVersionKind, obj runtime.Object) (runtime.Object, error) {
	return f.WrapFunc(parent, parentGvk, obj)
}

// List lists objects.
func (f FuncReaderWriter) List(parent runtime.Object, metas ListMetas) ([]runtime.Object, error) {
	return f.ListFunc(parent, metas)
}

// WrapKube wraps an object into kubernetes object.
func WrapKube(parent runtime.Object, parentGvk schema.GroupVersionKind, obj runtime.Object) (runtime.Object, error) {
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return nil, err
	}
	if err := mem.Unmemorize(obj); err != nil {
		return nil, err
	}
	if err = mem.SetOwnerReferences(parent, parentGvk, accessor); err != nil {
		return nil, err
	}
	return obj, nil
}

// WrapMem wraps an object into an memorized object.
func WrapMem(parent runtime.Object, parentGvk schema.GroupVersionKind, obj runtime.Object) (runtime.Object, error) {
	if err := mem.Memorize(obj); err != nil {
		return nil, err
	}
	return obj, nil
}

const (
	// KubeType read/write from/to kubenetes.
	KubeType = "kube"
	// MemType read/write from/to memory.
	MemType = "mem"
	// NopType no operation
	NopType = "nop"
)

// Descriptor the resource/object descrptor.
type Descriptor struct {
	Resource     string    `json:"resource"`
	Kind         string    `json:"kind"`
	Source       string    `json:"source"`
	Destination  string    `json:"destination"`
	Metas        ListMetas `json:"metas"`
	GroupVersion string    `json:"groupVersion,omitempty"` // if empty, use default
}

// ReaderWriterPair holds ReaderWriters
type ReaderWriterPair struct {
	KubeReaderWriter ReaderWriter
	MemReaderWriter  ReaderWriter
}

// ReaderWriterCreator used to create ReaderWriters
type ReaderWriterCreator func(gvr schema.GroupVersionResource) ReaderWriterPair

// CreateNodeProcessors create node processors.
// gv is the default GroupVersion, use descriptor GroupVersion if it's configed.
func CreateNodeProcessors(gv schema.GroupVersion, descriptors []*Descriptor, creator ReaderWriterCreator) (*NodeProcessor, error) {
	if len(descriptors) == 0 {
		return nil, errors.New("empty descriptors")
	}
	node := createNode(gv, descriptors[0], creator)
	processor := Root(node)
	root := processor
	for _, desc := range descriptors[1:] {
		node := createNode(gv, desc, creator)
		processor = processor.Child(node)
	}
	return root, nil
}

func createNode(gv schema.GroupVersion, desc *Descriptor, creator ReaderWriterCreator) Node {
	var err error
	if desc.GroupVersion != "" {
		gv, err = schema.ParseGroupVersion(desc.GroupVersion)
		if err != nil {
			panic(fmt.Sprintf("Parse GroupVersion failed: %s, %v", desc.GroupVersion, err))
		}
	}
	node := Node{
		Gvk:   gv.WithKind(desc.Kind),
		Metas: desc.Metas,
	}
	apis := creator(gv.WithResource(desc.Resource))
	if desc.Source == KubeType {
		node.Reader = apis.KubeReaderWriter
	} else {
		node.Reader = apis.MemReaderWriter
	}
	if desc.Destination == KubeType {
		node.Writer = apis.KubeReaderWriter
	} else if desc.Destination == NopType {
		node.Writer = &NopReaderWriter{}
	} else {
		node.Writer = apis.MemReaderWriter
	}
	return node
}
