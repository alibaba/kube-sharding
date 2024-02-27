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
	"testing"

	"github.com/alibaba/kube-sharding/pkg/memkube/mem"
	"github.com/alibaba/kube-sharding/pkg/memkube/testutils"
	"github.com/alibaba/kube-sharding/pkg/memkube/testutils/carbon"
	"github.com/alibaba/kube-sharding/pkg/memkube/testutils/carbon/spec"
	"github.com/golang/mock/gomock"
	assert "github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/errors"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestNodeProcessorsProcess(t *testing.T) {
	assert := assert.New(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	kind := "WorkerNode"
	gr := schema.GroupResource{Group: spec.SchemeGroupVersion.Group, Resource: "workernodes"}

	descriptors := []*Descriptor{
		{
			Kind:        kind,
			Source:      KubeType,
			Destination: MemType,
		},
	}
	wi := carbon.NewMockWorkerNodeInterface(ctrl)
	creator := func(gvr schema.GroupVersionResource) ReaderWriterPair {
		readerWriter := FuncReaderWriter{
			WriteFunc: func(obj runtime.Object) error {
				node := obj.(*spec.WorkerNode)
				_, err := wi.Get(node.Name, metav1.GetOptions{})
				if err == nil {
					_, err = wi.Update(node)
				} else if errors.IsNotFound(err) {
					_, err = wi.Create(node)
				}
				return err
			},
			WrapFunc: WrapMem,
			ListFunc: func(parent runtime.Object, metas ListMetas) ([]runtime.Object, error) {
				nodes, err := wi.List(metav1.ListOptions{})
				if err != nil {
					return nil, err
				}
				if nodes == nil {
					return nil, nil
				}
				items := make([]runtime.Object, 0, len(nodes.Items))
				for _, node := range nodes.Items {
					items = append(items, &node)
				}
				return items, nil
			},
		}
		return ReaderWriterPair{
			MemReaderWriter:  readerWriter,
			KubeReaderWriter: readerWriter,
		}
	}
	processors, err := CreateNodeProcessors(spec.SchemeGroupVersion, descriptors, creator)
	assert.Nil(err)
	// List empty
	wi.EXPECT().List(gomock.Any()).Return(nil, nil).Times(1)
	assert.Nil(processors.Process(nil))

	// List 1
	node := &spec.WorkerNode{
		ObjectMeta: metav1.ObjectMeta{
			Name: "n1",
		},
	}
	wi.EXPECT().List(gomock.Any()).Return(&spec.WorkerNodeList{Items: []spec.WorkerNode{*node}}, nil).Times(1)
	wi.EXPECT().Get(node.Name, gomock.Any()).Return(nil, kerrors.NewNotFound(gr, node.Name)).Times(1)
	matcher := testutils.FuncMockMatcher{
		Name: "match mem object",
		Func: func(x interface{}) bool {
			n := x.(*spec.WorkerNode)
			return mem.IsMemObject(n) && n.Name == node.Name
		},
	}
	wi.EXPECT().Create(matcher).Return(nil, nil).Times(1)
	assert.Nil(processors.Process(nil))
}

func TestCreateNodeProcessors(t *testing.T) {
	assert := assert.New(t)
	gv := spec.SchemeGroupVersion
	descriptors := []*Descriptor{
		// Do nothing (Kube -> Nop)
		{
			Kind:        "ShardGroup",
			Source:      KubeType,
			Destination: NopType,
		},
		// Rollback (Mem -> Kube)
		{
			Kind:        "Rollingset",
			Source:      MemType,
			Destination: KubeType,
		},
		// Memorize (Kube -> Mem)
		{
			Kind:        "WorkerNode",
			Source:      KubeType,
			Destination: MemType,
		},
		// Update only (Kube -> Kube)
		{
			Kind:        "Pod",
			Source:      KubeType,
			Destination: KubeType,
		},
	}
	memReaderWriter := FuncReaderWriter{Name: "mem"}
	kubeReaderWriter := FuncReaderWriter{Name: "kube"}
	creator := func(schema.GroupVersionResource) ReaderWriterPair {
		return ReaderWriterPair{
			MemReaderWriter:  memReaderWriter,
			KubeReaderWriter: kubeReaderWriter,
		}
	}
	processors, err := CreateNodeProcessors(spec.SchemeGroupVersion, descriptors, creator)
	assert.Nil(err)
	// ShardGroup
	assert.Equal(kubeReaderWriter, processors.Reader)
	fmt.Printf("Type of: %s\n", reflect.TypeOf(&NopReaderWriter{}).String())
	assert.Equal(reflect.TypeOf(&NopReaderWriter{}).String(), reflect.TypeOf(processors.Writer).String())
	assert.Equal(gv.WithKind("ShardGroup"), processors.Gvk)
	// Rollingset
	processor := processors.child
	assert.Equal(memReaderWriter, processor.Reader)
	assert.Equal(kubeReaderWriter, processor.Writer)
	assert.Equal(gv.WithKind("Rollingset"), processor.Gvk)
	// WorkerNode
	processor = processor.child
	assert.Equal(kubeReaderWriter, processor.Reader)
	assert.Equal(memReaderWriter, processor.Writer)
	assert.Equal(gv.WithKind("WorkerNode"), processor.Gvk)
	// Pod
	processor = processor.child
	assert.Equal(kubeReaderWriter, processor.Reader)
	assert.Equal(kubeReaderWriter, processor.Writer)
	assert.Equal(gv.WithKind("Pod"), processor.Gvk)
}
