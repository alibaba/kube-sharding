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

package dummy

import (
	"errors"

	"github.com/alibaba/kube-sharding/pkg/memkube/client"
	"github.com/alibaba/kube-sharding/pkg/memkube/testutils/carbon"
	"github.com/alibaba/kube-sharding/pkg/memkube/testutils/carbon/spec"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// Dummy informer & listers are api compatible but no real client-go informers created.

type workerNodeInformer struct {
	client client.Client
}

func (i *workerNodeInformer) Informer() cache.SharedIndexInformer {
	panic("dummy informer has no informer actually.")
}

func (i *workerNodeInformer) Lister() carbon.WorkerNodeLister {
	return newWorkerNodeLister(i.client)
}

// NewWorkerNodeInformer create a new WorkerNodeInformer
func NewWorkerNodeInformer(client client.Client) carbon.WorkerNodeInformer {
	return &workerNodeInformer{client}
}

type workerNodeLister struct {
	client client.Client
}

func newWorkerNodeLister(client client.Client) carbon.WorkerNodeLister {
	return &workerNodeLister{client}
}

// List lists all WorkerNodes in the indexer.
// NOTE: List is actually a namespace listers because mem-kube always manage objects in 1 namespace.
func (l *workerNodeLister) List(selector labels.Selector) ([]*spec.WorkerNode, error) {
	return nil, errors.New("not support list all namespaces")
}

func (l *workerNodeLister) Get(name string) (*spec.WorkerNode, error) {
	return nil, errors.New("not implemented")
}

// WorkerNodes returns an object that can list and get WorkerNodes.
func (l *workerNodeLister) WorkerNodes(namespace string) carbon.WorkerNodeNamespaceLister {
	wi := carbon.NewWorkerNodeInterface(l.client, namespace)
	return workerNodeNamespaceLister{wi: wi}
}

type workerNodeNamespaceLister struct {
	wi carbon.WorkerNodeInterface
}

func (l workerNodeNamespaceLister) List(selector labels.Selector) (ret []*spec.WorkerNode, err error) {
	nodeList, err := l.wi.List(metav1.ListOptions{}) // TODO: list options from selector
	if err != nil {
		return nil, err
	}
	items := make([]*spec.WorkerNode, 0)
	for i := range nodeList.Items {
		items = append(items, &nodeList.Items[i])
	}
	return items, nil
}

func (l workerNodeNamespaceLister) Get(name string) (*spec.WorkerNode, error) {
	return l.wi.Get(name, metav1.GetOptions{})
}
