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

package memv1

import (
	"context"
	"errors"

	v1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/client/clientset/versioned/typed/carbon/v1"
	"github.com/alibaba/kube-sharding/pkg/memkube/client"
	"github.com/alibaba/kube-sharding/pkg/memkube/mem"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
)

type workerNodes struct {
	client client.Client
	ns     string
}

// NewWorkerNodeInterface create a workerNodeInterface impl.
func NewWorkerNodeInterface(client client.Client, namespace string) carbonv1.WorkerNodeInterface {
	return &workerNodes{client: client, ns: namespace}
}

func (w *workerNodes) Get(ctx context.Context, name string, options metav1.GetOptions) (result *v1.WorkerNode, err error) {
	result = &v1.WorkerNode{}
	obj, err := w.client.Get("workernodes", w.ns, name, options, result)
	if err != nil {
		return nil, err
	}
	return obj.(*v1.WorkerNode), err
}

// List lists workernodes.
func (w *workerNodes) List(ctx context.Context, opts metav1.ListOptions) (result *v1.WorkerNodeList, err error) {
	// TODO: for local resource, require to initialize fields exclude Items ?
	result = &v1.WorkerNodeList{}
	err = w.client.List("workernodes", w.ns, opts, result)
	return
}

// Watch return watch interface.
func (w *workerNodes) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	return w.client.Watch("workernodes", w.ns, &mem.WatchOptions{ListOptions: opts})
}

func (w *workerNodes) Update(ctx context.Context, workerNode *v1.WorkerNode, opts metav1.UpdateOptions) (*v1.WorkerNode, error) {
	runtimeObj, err := w.client.Update("workernodes", w.ns, workerNode)
	if err != nil {
		return nil, err
	}
	return runtimeObj.(*v1.WorkerNode), nil
}

func (w *workerNodes) Create(ctx context.Context, workerNode *v1.WorkerNode, opts metav1.CreateOptions) (*v1.WorkerNode, error) {
	runtimeObj, err := w.client.Create("workernodes", w.ns, workerNode)
	if err != nil {
		return nil, err
	}
	return runtimeObj.(*v1.WorkerNode), nil
}

func (w *workerNodes) UpdateStatus(ctx context.Context, workerNode *v1.WorkerNode, opts metav1.UpdateOptions) (*v1.WorkerNode, error) {
	runtimeObj, err := w.client.Update("workernodes", w.ns, workerNode)
	if err != nil {
		return nil, err
	}
	return runtimeObj.(*v1.WorkerNode), nil
}

func (w *workerNodes) Delete(ctx context.Context, name string, options metav1.DeleteOptions) error {
	return w.client.Delete("workernodes", w.ns, name, &options)
}

func (w *workerNodes) DeleteCollection(ctx context.Context, options metav1.DeleteOptions, listOptions metav1.ListOptions) error {
	return errors.New("unsupport func")
}

func (w *workerNodes) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.WorkerNode, err error) {
	return nil, errors.New("unsupport func")
}
