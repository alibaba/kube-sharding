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
	"context"
	"errors"

	"github.com/alibaba/kube-sharding/pkg/memkube/client"
	"github.com/alibaba/kube-sharding/pkg/memkube/mem"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	glog "k8s.io/klog"
)

var (
	// ErrWrongTypeObject the object is not unstructured object
	ErrWrongTypeObject = errors.New("runtime.Object is not unstructured object")
)

func convertUnstructuredItems(list *unstructured.UnstructuredList) []runtime.Object {
	objs := make([]runtime.Object, 0, len(list.Items))
	for i := range list.Items {
		objs = append(objs, &list.Items[i])
	}
	return objs
}

func copyResourceVersion(src runtime.Object, dst runtime.Object) error {
	return mem.MetaAccessors(func(accessors ...metav1.Object) error {
		accessors[1].SetResourceVersion(accessors[0].GetResourceVersion())
		return nil
	}, src, dst)
}

type writerLister struct {
	name     string
	selector SelectorProvider
	gvr      schema.GroupVersionResource
	client   dynamic.ResourceInterface
}

func (wl *writerLister) Write(obj runtime.Object) error {
	uobj, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return ErrWrongTypeObject
	}
	name := mem.MetaName(obj)
	remote, err := wl.client.Get(context.Background(), name, metav1.GetOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		return err
	}
	if kerrors.IsNotFound(err) {
		_, err := wl.client.Create(context.Background(), uobj, metav1.CreateOptions{})
		glog.Infof("Do create a %s resource [%s]: %s, %v", wl.name, name, wl.gvr.String(), err)
		return err
	}
	// Always update the resourceVersion to update success
	if err = copyResourceVersion(remote, uobj); err != nil {
		glog.Errorf("Copy resource version error [%s], %s, %v", wl.gvr.String(), name, err)
		return err
	}
	_, err = wl.client.Update(context.Background(), uobj, metav1.UpdateOptions{})
	glog.Infof("Do update a %s resource [%s] %s, %v", wl.name, name, wl.gvr.String(), err)
	return err
}

func (wl *writerLister) List(parent runtime.Object, metas ListMetas) ([]runtime.Object, error) {
	labels, err := wl.selector.Selector(metas, parent)
	if err != nil {
		return nil, err
	}
	list, err := wl.client.List(context.Background(), metav1.ListOptions{LabelSelector: labels})

	if err != nil {
		return nil, err
	}
	glog.V(4).Infof("List %s resources [%s], selector: %s, items: %d", wl.name, wl.gvr.String(), labels, len(list.Items))
	return convertUnstructuredItems(list), nil
}

// NewDynamicReaderWriterCreator setup a creator
func NewDynamicReaderWriterCreator(ns string, kubeapis dynamic.Interface, memclient *client.DynamicClient) ReaderWriterCreator {
	sp := &nameSelector{}
	return func(gvr schema.GroupVersionResource) ReaderWriterPair {
		kube := &writerLister{
			name:     "kube",
			client:   kubeapis.Resource(gvr).Namespace(ns),
			selector: sp,
			gvr:      gvr,
		}
		kubeRW := &FuncReaderWriter{
			WriteFunc: kube.Write,
			ListFunc:  kube.List,
			WrapFunc:  WrapKube,
		}
		mem := &writerLister{
			name:     "mem",
			client:   client.NewDynamicResource(memclient).Resource(gvr).Namespace(ns),
			selector: sp,
			gvr:      gvr,
		}
		memRW := &FuncReaderWriter{
			WriteFunc: mem.Write,
			ListFunc:  mem.List,
			WrapFunc:  WrapMem,
		}
		return ReaderWriterPair{
			KubeReaderWriter: kubeRW,
			MemReaderWriter:  memRW,
		}
	}
}
