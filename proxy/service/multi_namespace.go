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

package service

import (
	"errors"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

//MultiNamespaceIndexer MultiNamespaceIndexer
type multiNamespaceIndexer struct {
	indexers   map[string]cache.Indexer
	allIndexer cache.Indexer // namespace ""
}

func newMultiNamespaceIndexer(indexers map[string]cache.Indexer) *multiNamespaceIndexer {
	mi := &multiNamespaceIndexer{indexers: make(map[string]cache.Indexer)}
	for ns, indexer := range indexers {
		if ns == v1.NamespaceAll {
			mi.allIndexer = indexer
		} else {
			mi.indexers[ns] = indexer
		}
	}
	return mi
}

// ListIndexFuncValues returns all the indexed values of the given index
func (i *multiNamespaceIndexer) ListIndexFuncValues(indexName string) []string {
	allItem := make([]string, 0)
	if i.allIndexer != nil {
		return i.allIndexer.ListIndexFuncValues(indexName)
	}
	for _, indexer := range i.indexers {
		item := indexer.ListIndexFuncValues(indexName)
		allItem = append(allItem, item...)
	}
	return allItem
}

// ByIndex returns the stored objects whose set of indexed values
// for the named index includes the given indexed value
func (i *multiNamespaceIndexer) ByIndex(indexName, indexedValue string) ([]interface{}, error) {
	allItem := make([]interface{}, 0)
	// First load items in specified namespace indexers
	for _, indexer := range i.indexers {
		item, err := indexer.ByIndex(indexName, indexedValue)
		if err != nil {
			return nil, err
		}
		allItem = append(allItem, item...)
	}
	// If no items found, try the all namespace indexer
	// NOTE: this may miss some items when the items are put in multiple namespaces, but for now it's ok
	// Better to merge specified namespace indexers with all namespace indexers by items' name.
	if i.allIndexer != nil && len(allItem) == 0 {
		return i.allIndexer.ByIndex(indexName, indexedValue)
	}
	return allItem, nil
}

// AddIndexers adds more indexers to this store.  If you call this after you already have data
// in the store, the results are undefined.
func (i *multiNamespaceIndexer) AddIndexers(newIndexers cache.Indexers) error {
	if i.allIndexer != nil {
		if err := i.allIndexer.AddIndexers(newIndexers); err != nil {
			return err
		}
	}
	for _, indexer := range i.indexers {
		if err := indexer.AddIndexers(newIndexers); err != nil {
			return err
		}
	}
	return nil
}

//MultiNamespacePodLister MultiNamespacePodLister
type multiNamespacePodLister struct {
	podListers map[string]corelisters.PodLister
}

// List lists all Pods in the indexer.
func (i *multiNamespacePodLister) List(selector labels.Selector) (ret []*v1.Pod, err error) {
	return nil, errors.New("not support")
}

// Pods returns an object that can list and get Pods.
func (i *multiNamespacePodLister) Pods(namespace string) corelisters.PodNamespaceLister {
	podLister := i.podListers[namespace]
	if podLister == nil {
		podLister = i.podListers[""]
		if podLister == nil {
			return nil
		}
	}
	return podLister.Pods(namespace)
}
