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
	"errors"

	"github.com/alibaba/kube-sharding/pkg/memkube/testutils/carbon/spec"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// NewWorkerNodeNamespaceLister create a lister.
func NewWorkerNodeNamespaceLister(ns string, indexer cache.Indexer) WorkerNodeNamespaceLister {
	return &workerNodeNamespaceLister{indexer: indexer, namespace: ns}
}

// These codes below are copied from code-gen for workerNodes in c2 project

// WorkerNodeLister lister api
type WorkerNodeLister interface {
	// List lists all WorkerNodes in the indexer.
	List(selector labels.Selector) ([]*spec.WorkerNode, error)
	// WorkerNodes returns an object that can list and get WorkerNodes.
	WorkerNodes(namespace string) WorkerNodeNamespaceLister
}

type workerNodeLister struct {
	indexer cache.Indexer
}

// NewWorkerNodeLister returns a new WorkerNodeLister.
func NewWorkerNodeLister(indexer cache.Indexer) WorkerNodeLister {
	return &workerNodeLister{indexer: indexer}
}

// List lists all WorkerNodes in the indexer.
func (s *workerNodeLister) List(selector labels.Selector) (ret []*spec.WorkerNode, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*spec.WorkerNode))
	})
	return ret, err
}

// WorkerNodes returns an object that can list and get WorkerNodes.
func (s *workerNodeLister) WorkerNodes(namespace string) WorkerNodeNamespaceLister {
	return workerNodeNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// WorkerNodeNamespaceLister helps list and get WorkerNodes.
type WorkerNodeNamespaceLister interface {
	// List lists all WorkerNodes in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*spec.WorkerNode, err error)
	// Get retrieves the WorkerNode from the indexer for a given namespace and name.
	Get(name string) (*spec.WorkerNode, error)
}

// workerNodeNamespaceLister implements the WorkerNodeNamespaceLister
// interface.
type workerNodeNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all WorkerNodes in the indexer for a given namespace.
func (s workerNodeNamespaceLister) List(selector labels.Selector) (ret []*spec.WorkerNode, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*spec.WorkerNode))
	})
	return ret, err
}

// Get retrieves the WorkerNode from the indexer for a given namespace and name.
func (s workerNodeNamespaceLister) Get(name string) (*spec.WorkerNode, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New("wrokerNode not found")
	}
	return obj.(*spec.WorkerNode), nil
}
