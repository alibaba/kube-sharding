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

package carbon

import (
	"github.com/alibaba/kube-sharding/pkg/memkube/client"
	"github.com/alibaba/kube-sharding/pkg/memkube/mem"
	"github.com/alibaba/kube-sharding/pkg/memkube/testutils/carbon/spec"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
)

// WorkerNodeInterface sample workerNodeInterface impl.
type WorkerNodeInterface interface {
	List(opts metav1.ListOptions) (*spec.WorkerNodeList, error)
	Watch(opts metav1.ListOptions) (watch.Interface, error)
	Update(*spec.WorkerNode) (*spec.WorkerNode, error)
	Get(name string, options metav1.GetOptions) (*spec.WorkerNode, error)
	Create(*spec.WorkerNode) (*spec.WorkerNode, error)
	// TODO: add full api set
}

type workerNodes struct {
	client client.Client
	ns     string
}

// NewWorkerNodeInterface create a workerNodeInterface impl.
func NewWorkerNodeInterface(client client.Client, namespace string) WorkerNodeInterface {
	return &workerNodes{client: client, ns: namespace}
}

// List lists workernodes.
func (w *workerNodes) List(opts metav1.ListOptions) (*spec.WorkerNodeList, error) {
	// TODO: for local resource, require to initialize fields exclude Items ?
	result := &spec.WorkerNodeList{}
	err := w.client.List("workernodes", w.ns, opts, result)
	return result, err
}

// Watch return watch interface.
func (w *workerNodes) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	return w.client.Watch("workernodes", w.ns, &mem.WatchOptions{ListOptions: opts})
}

func (w *workerNodes) Update(workerNode *spec.WorkerNode) (result *spec.WorkerNode, err error) {
	var obj runtime.Object
	obj, err = w.client.Update("workernodes", w.ns, workerNode)
	if err != nil {
		return
	}
	result = obj.(*spec.WorkerNode)
	return
}

func (w *workerNodes) Get(name string, options metav1.GetOptions) (*spec.WorkerNode, error) {
	result := &spec.WorkerNode{}
	obj, err := w.client.Get("workernodes", w.ns, name, options, result)
	return obj.(*spec.WorkerNode), err
}

func (w *workerNodes) Create(workerNode *spec.WorkerNode) (result *spec.WorkerNode, err error) {
	var obj runtime.Object
	obj, err = w.client.Create("workernodes", w.ns, workerNode)
	result = obj.(*spec.WorkerNode)
	return
}
