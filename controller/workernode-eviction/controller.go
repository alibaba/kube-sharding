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

package workernodeeviction

import (
	"fmt"
	"reflect"

	"github.com/golang/glog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"

	"github.com/alibaba/kube-sharding/controller"
	"github.com/alibaba/kube-sharding/controller/util"
	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	clientset "github.com/alibaba/kube-sharding/pkg/client/clientset/versioned"
	carboncheme "github.com/alibaba/kube-sharding/pkg/client/clientset/versioned/scheme"
	informers "github.com/alibaba/kube-sharding/pkg/client/informers/externalversions/carbon/v1"
	listers "github.com/alibaba/kube-sharding/pkg/client/listers/carbon/v1"
	coreinformers "k8s.io/client-go/informers/core/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
)

const (
	controllerAgentName = "workernodeeviction"

	// SuccessSynced is used as part of the Event 'reason' when a ControllersManager is synced
	SuccessSynced = "Synced"
)

var controllerKind = carbonv1.SchemeGroupVersion.WithKind("WorkerNodeEviction")

// Controller is the controller implementation for ControllersManager resources
type Controller struct {
	schema.GroupVersionKind
	controller.DefaultController

	carbonclientset clientset.Interface
	kubeclientset   kubernetes.Interface

	podLister      corelisters.PodLister
	podListerSyncd cache.InformerSynced
	podInformer    coreinformers.PodInformer

	workerNodeLister       listers.WorkerNodeLister
	workerNodeListerSynced cache.InformerSynced
	workerNodeInformer     informers.WorkerNodeInformer

	evictionLister       listers.WorkerNodeEvictionLister
	evictionInformer     informers.WorkerNodeEvictionInformer
	evictionListerSynced cache.InformerSynced
}

// NewController returns a new ControllersManager controller
func NewController(
	kubeclientset kubernetes.Interface,
	carbonclientset clientset.Interface,
	podInformer coreinformers.PodInformer,
	workerNodeInformer informers.WorkerNodeInformer,
	evictionInformer informers.WorkerNodeEvictionInformer,
) *Controller {
	utilruntime.Must(carboncheme.AddToScheme(scheme.Scheme))
	c := &Controller{
		carbonclientset:        carbonclientset,
		kubeclientset:          kubeclientset,
		podListerSyncd:         podInformer.Informer().HasSynced,
		podLister:              podInformer.Lister(),
		workerNodeInformer:     workerNodeInformer,
		workerNodeListerSynced: workerNodeInformer.Informer().HasSynced,
		workerNodeLister:       workerNodeInformer.Lister(),
		evictionInformer:       evictionInformer,
		evictionLister:         evictionInformer.Lister(),
		evictionListerSynced:   evictionInformer.Informer().HasSynced,
	}

	c.DefaultController = *controller.NewDefaultController(kubeclientset, carbonclientset,
		controllerAgentName, controllerKind, c)
	c.DefaultController.ResourceManager = util.NewSimpleResourceManager(kubeclientset, carbonclientset, nil, nil, nil)

	glog.Info("Setting up workerNodeEviction event handlers")

	evictionInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.EnqueueNow,
		UpdateFunc: func(old, new interface{}) {
			if reflect.DeepEqual(old, new) {
				return
			}
			c.EnqueueNow(new)
		},
		DeleteFunc: c.deleteEviction,
	})
	return c
}

func (c *Controller) deleteEviction(obj interface{}) {
	evict, ok := obj.(*carbonv1.WorkerNodeEviction)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("Couldn't get object from tombstone %#v", obj))
			return
		}
		evict, ok = tombstone.Obj.(*carbonv1.WorkerNodeEviction)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("Tombstone contained object that is not a WorkerNodeEviction %#v", obj))
			return
		}
	}

	if nil == evict.DeletionTimestamp {
		now := metav1.Now()
		evict.DeletionTimestamp = &now
	}
	glog.Infof("on deleted worker eviction : %s", utils.ObjJSON(evict))
}

// GetObj GetObj
func (c *Controller) GetObj(namespace, key string) (interface{}, error) {
	e, err := c.evictionLister.WorkerNodeEvictions(namespace).Get(key)
	if nil != err {
		glog.Warningf("get workernode error: %s, %s, %v", namespace, key, err)
	}
	return e, err
}

// DeleteSubObj DeleteSubObj
func (c *Controller) DeleteSubObj(namespace, key string) error {
	// No sub obj of this
	return nil
}

// WaitForCacheSync WaitForCacheSync
func (c *Controller) WaitForCacheSync(stopCh <-chan struct{}) bool {
	if ok := cache.WaitForCacheSync(stopCh,
		c.podListerSyncd,
		c.evictionListerSynced,
		c.workerNodeListerSynced,
	); !ok {
		glog.Error("failed to wait for caches to sync")
		return false
	}
	return true
}
