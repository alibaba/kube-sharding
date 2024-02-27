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
	"time"

	"github.com/alibaba/kube-sharding/pkg/memkube/client"
	"github.com/alibaba/kube-sharding/pkg/memkube/testutils/carbon/spec"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

// WorkerNodeInformer informer api
type WorkerNodeInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() WorkerNodeLister
}

type workerNodeInformer struct {
	indexInformer cache.SharedIndexInformer
}

// NewWorkerNodeInformer create a new WorkerNodeInformer
func NewWorkerNodeInformer(client client.Client, namespace string, resyncPeriod time.Duration) WorkerNodeInformer {
	informer := NewFilteredWorkerNodeInformer(client, namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	return &workerNodeInformer{informer}
}

func (i *workerNodeInformer) Informer() cache.SharedIndexInformer {
	return i.indexInformer
}

func (i *workerNodeInformer) Lister() WorkerNodeLister {
	return NewWorkerNodeLister(i.indexInformer.GetIndexer())
}

// NewFilteredWorkerNodeInformer informer sample.
func NewFilteredWorkerNodeInformer(client client.Client, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	wi := NewWorkerNodeInterface(client, namespace)
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return wi.List(options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return wi.Watch(options)
			},
		},
		&spec.WorkerNode{},
		resyncPeriod,
		indexers,
	)
}
