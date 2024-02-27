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

package healthcheck

import (
	"flag"

	"github.com/alibaba/kube-sharding/controller"
	"github.com/alibaba/kube-sharding/controller/util"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"
	glog "k8s.io/klog"

	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	clientset "github.com/alibaba/kube-sharding/pkg/client/clientset/versioned"
	carboncheme "github.com/alibaba/kube-sharding/pkg/client/clientset/versioned/scheme"
	informers "github.com/alibaba/kube-sharding/pkg/client/informers/externalversions/carbon/v1"
	listers "github.com/alibaba/kube-sharding/pkg/client/listers/carbon/v1"
)

const controllerAgentName = "healthcheck-controller"

var controllerKind = carbonv1.SchemeGroupVersion.WithKind("RollingSet")

const (
	// SuccessSynced is used as part of the Event 'reason' when a HealthChecker is synced
	SuccessSynced = "Synced"
)

var healthCheckHTTPTimeout int64 = 30

// InitFlags is for explicitly initializing the flags.
func init() {
	flag.Int64Var(&healthCheckHTTPTimeout, "health-check-timeout", healthCheckHTTPTimeout, "http query timeout for healthcheck in second")
	flag.IntVar(&healthcheckInterval, "health-check-interval", healthcheckInterval, "health check interval in millisecond")
}

// Controller is the controller implementation for HealthChecker resources
type Controller struct {
	schema.GroupVersionKind
	controller.DefaultController

	rollingSetLister listers.RollingSetLister
	rollingSetSync   cache.InformerSynced
	workernodeLister listers.WorkerNodeLister
	workernodeSynced cache.InformerSynced

	healthCheckManager Manager
}

// NewController returns a new healthcheck controller
func NewController(
	kubeclientset kubernetes.Interface,
	carbonclientset clientset.Interface, rollingSetInformer informers.RollingSetInformer,
	workernodeInformer informers.WorkerNodeInformer) *Controller {

	utilruntime.Must(carboncheme.AddToScheme(scheme.Scheme))
	healthController := &Controller{
		rollingSetLister: rollingSetInformer.Lister(),
		rollingSetSync:   rollingSetInformer.Informer().HasSynced,
		workernodeLister: workernodeInformer.Lister(),
		workernodeSynced: workernodeInformer.Informer().HasSynced,
	}

	healthController.DefaultController = *controller.NewDefaultController(kubeclientset, carbonclientset,
		controllerAgentName, controllerKind, healthController)
	healthController.DefaultController.ResourceManager = util.NewSimpleResourceManager(kubeclientset, carbonclientset, workernodeInformer.Informer().GetIndexer(), nil, nil)

	healthController.healthCheckManager = NewHealthCheckManager(healthController.ResourceManager, healthController.workernodeLister)

	// Set up an event handler for when ReplicaSet resources change
	rollingSetInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: healthController.Enqueue,
		UpdateFunc: func(old, new interface{}) {
			oldHc := old.(*carbonv1.RollingSet)
			newHc := new.(*carbonv1.RollingSet)
			if oldHc.ResourceVersion == newHc.ResourceVersion {
				return
			}
			healthController.Enqueue(newHc)
		},
		DeleteFunc: healthController.EnqueueWithoutEvent,
	})

	return healthController
}

// Sync compares the actual state with the desired, and attempts to converge the two.
func (c *Controller) Sync(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		glog.Warningf("invalid resource key: %s", key)
		return nil
	}
	rollingSet, err := c.rollingSetLister.RollingSets(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			c.healthCheckManager.removeTask(key)
			glog.Warningf("remove HealthCheckerTask '%s'", key)
			return nil
		}
		return err
	}
	if rollingSet == nil || rollingSet.DeletionTimestamp != nil || carbonv1.IsSubRollingSet(rollingSet) {
		c.healthCheckManager.removeTask(key)
		glog.Warningf("remove HealthCheckerTask '%s'", key)
		return nil
	}

	if err := c.healthCheckManager.Process(key, rollingSet.DeepCopy()); err != nil {
		glog.Warningf("HealthChecker Process error key: %s", key)
	}
	return nil
}

// WaitForCacheSync wait for informers synced
func (c *Controller) WaitForCacheSync(stopCh <-chan struct{}) bool {
	if ok := cache.WaitForCacheSync(stopCh, c.rollingSetSync, c.workernodeSynced); !ok {
		glog.Error("failed to wait for caches to sync")
		return false
	}
	return true
}
