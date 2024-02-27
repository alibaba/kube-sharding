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

package carbonjob

import (
	"reflect"
	"time"

	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"

	"github.com/alibaba/kube-sharding/controller"
	"github.com/alibaba/kube-sharding/controller/util"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	clientset "github.com/alibaba/kube-sharding/pkg/client/clientset/versioned"
	carboncheme "github.com/alibaba/kube-sharding/pkg/client/clientset/versioned/scheme"
	informers "github.com/alibaba/kube-sharding/pkg/client/informers/externalversions/carbon/v1"
	listers "github.com/alibaba/kube-sharding/pkg/client/listers/carbon/v1"
	"github.com/alibaba/kube-sharding/transfer"
	coreinformers "k8s.io/client-go/informers/core/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
)

const (
	controllerAgentName = "carbonjob-controller"

	// SuccessSynced is used as part of the Event 'reason' when a ControllersManager is synced
	SuccessSynced = "Synced"

	defaultQueueDelayTimeDuration = 3 * time.Second
)

var controllerKind = carbonv1.SchemeGroupVersion.WithKind("CarbonJob")

// Controller is the controller implementation for ControllersManager resources
type Controller struct {
	schema.GroupVersionKind
	controller.DefaultController

	carbonclientset clientset.Interface
	kubeclientset   kubernetes.Interface

	podLister       corelisters.PodLister
	podListerSynced cache.InformerSynced
	podInformer     coreinformers.PodInformer

	workerNodeLister       listers.WorkerNodeLister
	workerNodeListerSynced cache.InformerSynced
	workerNodeInformer     informers.WorkerNodeInformer

	carbonJobLister   listers.CarbonJobLister
	carbonJobSynced   cache.InformerSynced
	carbonJobInformer informers.CarbonJobInformer

	specConverter transfer.WorkerNodeSpecConverter
}

// NewController returns a new ControllersManager controller
func NewController(
	kubeclientset kubernetes.Interface,
	carbonclientset clientset.Interface,
	podInformer coreinformers.PodInformer,
	workerNodeInformer informers.WorkerNodeInformer,
	carbonJobInformer informers.CarbonJobInformer,
) *Controller {
	utilruntime.Must(carboncheme.AddToScheme(scheme.Scheme))
	c := &Controller{
		carbonclientset:        carbonclientset,
		kubeclientset:          kubeclientset,
		podInformer:            podInformer,
		podListerSynced:        podInformer.Informer().HasSynced,
		podLister:              podInformer.Lister(),
		workerNodeInformer:     workerNodeInformer,
		workerNodeListerSynced: workerNodeInformer.Informer().HasSynced,
		workerNodeLister:       workerNodeInformer.Lister(),
		carbonJobInformer:      carbonJobInformer,
		carbonJobLister:        carbonJobInformer.Lister(),
		carbonJobSynced:        carbonJobInformer.Informer().HasSynced,
	}

	glog.Infof("carbonJob controller Init with carbonclientset type %T, kubeclient type %T", reflect.TypeOf(carbonclientset), reflect.TypeOf(kubeclientset))

	c.DefaultController = *controller.NewDefaultController(kubeclientset, carbonclientset,
		controllerAgentName, controllerKind, c)
	c.DefaultController.ResourceManager = util.NewSimpleResourceManager(kubeclientset, carbonclientset, workerNodeInformer.Informer().GetIndexer(), nil, nil)
	c.DefaultController.DelayProcessDuration = defaultQueueDelayTimeDuration
	c.specConverter = transfer.NewWorkerNodeSpecConverter()

	glog.Info("Setting up carbonJob event handlers")

	carbonJobInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.Enqueue,
		UpdateFunc: func(old, new interface{}) {
			if reflect.DeepEqual(old, new) {
				return
			}
			c.Enqueue(new)
		},
		// 删除操作, 触发新的sync, 同时由于finalizer存在, 有deletionTimestamp
		DeleteFunc: c.EnqueueWithoutEvent,
	})

	workerNodeInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.HandleSubObjectDelayDuration,
		UpdateFunc: func(old, new interface{}) {
			if reflect.DeepEqual(old, new) {
				return
			}
			c.HandleSubObjectDelayDuration(new)
		},
		DeleteFunc: c.HandleSubObjectDelayDuration,
	})
	return c
}

// GetObj GetObj
func (c *Controller) GetObj(namespace, key string) (interface{}, error) {
	cb, err := c.carbonJobLister.CarbonJobs(namespace).Get(key)
	if nil != err {
		glog.Warningf("get carbonjob error: %s, %s, %v", namespace, key, err)
	}
	return cb, err
}

// DeleteSubObj DeleteSubObj
func (c *Controller) DeleteSubObj(namespace, name string) error {
	workerNode, err := c.workerNodeLister.WorkerNodes(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		glog.Warningf("get workerNode error :%s,%s,%v", namespace, name, err)
		return err
	}

	glog.Infof("delete workerNode %s/%s by gc", namespace, name)
	err = c.ResourceManager.DeleteWorkerNode(workerNode)
	if nil != err {
		if errors.IsNotFound(err) {
			return nil
		}
		glog.Warningf("delete workerNode error : %s, %s, %v", namespace, name, err)
		return err
	}
	return nil
}

// WaitForCacheSync WaitForCacheSync
func (c *Controller) WaitForCacheSync(stopCh <-chan struct{}) bool {
	if ok := cache.WaitForCacheSync(stopCh,
		c.carbonJobSynced,
		c.workerNodeListerSynced,
		c.podListerSynced,
	); !ok {
		glog.Error("failed to wait for caches to sync")
		return false
	}
	return true
}
