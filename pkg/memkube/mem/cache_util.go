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

package mem

import (
	"time"

	"k8s.io/client-go/tools/cache"
	glog "k8s.io/klog"
)

// DuplicateInformerCache duplicates cache from informer, it binds event handlers to the informer, and update items in duplicated cache.
// NOTE: If use this cache to get objects, but register handlers on the raw informer, data mismatched (notified by v2, but only get v1)
func DuplicateInformerCache(informer cache.SharedIndexInformer, newIndexers cache.Indexers) cache.Indexer {
	// NOTE: the keyFunc must be same as NewSharedIndexInformer
	indexer := cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, informer.GetIndexer().GetIndexers())
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			indexer.Add(obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			startTime := time.Now()
			indexer.Update(newObj)
			if elapsed := time.Since(startTime); elapsed > time.Millisecond*500 {
				name, _ := cache.DeletionHandlingMetaNamespaceKeyFunc(newObj)
				glog.Warningf("Update indexer items slow: %s, %v", name, elapsed)
			}
		},
		DeleteFunc: func(obj interface{}) {
			indexer.Delete(obj)
		},
	})
	if newIndexers != nil {
		indexer.AddIndexers(newIndexers)
	}
	return indexer
}
