/*
Copyright The Kubernetes Authors.

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

// Code generated by lister-gen. DO NOT EDIT.

package v1

import (
	v1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// WorkerNodeLister helps list WorkerNodes.
type WorkerNodeLister interface {
	// List lists all WorkerNodes in the indexer.
	List(selector labels.Selector) (ret []*v1.WorkerNode, err error)
	// WorkerNodes returns an object that can list and get WorkerNodes.
	WorkerNodes(namespace string) WorkerNodeNamespaceLister
	WorkerNodeListerExpansion
}

// workerNodeLister implements the WorkerNodeLister interface.
type workerNodeLister struct {
	indexer cache.Indexer
}

// NewWorkerNodeLister returns a new WorkerNodeLister.
func NewWorkerNodeLister(indexer cache.Indexer) WorkerNodeLister {
	return &workerNodeLister{indexer: indexer}
}

// List lists all WorkerNodes in the indexer.
func (s *workerNodeLister) List(selector labels.Selector) (ret []*v1.WorkerNode, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.WorkerNode))
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
	List(selector labels.Selector) (ret []*v1.WorkerNode, err error)
	// Get retrieves the WorkerNode from the indexer for a given namespace and name.
	Get(name string) (*v1.WorkerNode, error)
	WorkerNodeNamespaceListerExpansion
}

// workerNodeNamespaceLister implements the WorkerNodeNamespaceLister
// interface.
type workerNodeNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all WorkerNodes in the indexer for a given namespace.
func (s workerNodeNamespaceLister) List(selector labels.Selector) (ret []*v1.WorkerNode, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.WorkerNode))
	})
	return ret, err
}

// Get retrieves the WorkerNode from the indexer for a given namespace and name.
func (s workerNodeNamespaceLister) Get(name string) (*v1.WorkerNode, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1.Resource("workernode"), name)
	}
	return obj.(*v1.WorkerNode), nil
}
