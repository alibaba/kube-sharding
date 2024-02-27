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
	"fmt"
	"reflect"

	"github.com/alibaba/kube-sharding/pkg/memkube/mem"
	"github.com/ivpusic/grpool"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	glog "k8s.io/klog"
)

// Writer writes object.
type Writer interface {
	Write(runtime.Object) error
	Wrap(parent runtime.Object, parentGvk schema.GroupVersionKind, obj runtime.Object) (runtime.Object, error)
}

// ListMetas payload arguments to List.
type ListMetas struct {
	Selectors map[string]string `json:"selectors,omitempty"`
	Config    map[string]string `json:"config,omitempty"`
}

// Reader reads/lists objects.
type Reader interface {
	List(parent runtime.Object, metas ListMetas) ([]runtime.Object, error)
}

// ReaderWriter read and write objects.
type ReaderWriter interface {
	Writer
	Reader
}

// NopReaderWriter nop
type NopReaderWriter struct {
}

// Write writes an object
func (rw *NopReaderWriter) Write(runtime.Object) error {
	return nil
}

// Wrap wrap an object
func (rw *NopReaderWriter) Wrap(parent runtime.Object, parentGvk schema.GroupVersionKind, obj runtime.Object) (runtime.Object, error) {
	return obj, nil
}

// List list objects
func (rw *NopReaderWriter) List(runtime.Object, ListMetas) ([]runtime.Object, error) {
	return nil, nil
}

// Node represents a type of object in the transition chain.
type Node struct {
	Reader Reader
	Writer Writer
	Gvk    schema.GroupVersionKind
	Metas  ListMetas
}

// Root create a root node processor.
func Root(n Node) *NodeProcessor {
	return &NodeProcessor{Node: n}
}

// NodeProcessor process one node.
type NodeProcessor struct {
	Node
	child *NodeProcessor
}

// ProcessOptions options for Process.
type ProcessOptions struct {
	Dry        bool
	NumWorkers int
}

// Child creates a child node from this node.
func (p *NodeProcessor) Child(n Node) *NodeProcessor {
	p.child = &NodeProcessor{Node: n}
	return p.child
}

// Process starts to process the transition.
func (p *NodeProcessor) Process(opts *ProcessOptions) error {
	if opts == nil {
		opts = &ProcessOptions{}
	}
	return p.process(nil, schema.GroupVersionKind{}, opts)
}

func (p *NodeProcessor) process(parent runtime.Object, gvk schema.GroupVersionKind, opts *ProcessOptions) error {
	items, err := p.Reader.List(parent, p.Metas)
	if err != nil {
		return fmt.Errorf("list items error for %s, %v", mem.MetaName(parent), err)
	}
	if len(items) == 0 {
		return nil
	}
	numWorkers := opts.NumWorkers
	if numWorkers == 0 {
		numWorkers = 5
	}
	grpool := grpool.NewPool(numWorkers, 1)
	defer grpool.Release()

	grpool.WaitCount(len(items))
	errs := make([]error, len(items))
	for i, item := range items {
		item2 := item
		index := i
		grpool.JobQueue <- func() {
			defer grpool.JobDone()
			err := p.processItem(item2, parent, gvk, opts)
			if err != nil {
				errs[index] = err
				glog.Errorf("ProcessItem err: [%s], item: %s, err: %v", p.Gvk.String(), mem.Stringify(item), err)
			}
		}
	}
	grpool.WaitAll()
	if err := p.joinedError(errs); err != nil {
		return err
	}
	return nil
}

func (p *NodeProcessor) processItem(item, parent runtime.Object, gvk schema.GroupVersionKind, opts *ProcessOptions) error {
	item, err := p.Writer.Wrap(parent, gvk, item)
	if err != nil {
		return fmt.Errorf("wrap object error: %s, %v", mem.MetaName(item), err)
	}
	ws := reflect.TypeOf(p.Writer)
	glog.Infof("Write object: [%s], writer: %s, %s", p.Gvk.String(), ws.String(), mem.Stringify(item))
	if !opts.Dry {
		if err = p.Writer.Write(item); err != nil {
			return err
		}
	}
	if p.child != nil {
		if err = p.child.process(item, p.Gvk, opts); err != nil {
			return err
		}
	}
	return nil
}

func (p *NodeProcessor) joinedError(errs []error) error {
	errStr := ""
	isErr := false
	for _, err := range errs {
		if err != nil {
			isErr = true
			errStr += err.Error() + ";"
		}
	}
	if !isErr {
		return nil
	}
	return fmt.Errorf("process joined error: %s", errStr)
}
