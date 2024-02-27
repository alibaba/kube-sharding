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

package shardgroup

import (
	RawErrors "errors"
	"reflect"
	"time"

	"github.com/alibaba/kube-sharding/controller"
	"github.com/alibaba/kube-sharding/controller/util"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	clientset "github.com/alibaba/kube-sharding/pkg/client/clientset/versioned"
	carboncheme "github.com/alibaba/kube-sharding/pkg/client/clientset/versioned/scheme"
	informers "github.com/alibaba/kube-sharding/pkg/client/informers/externalversions/carbon/v1"
	listers "github.com/alibaba/kube-sharding/pkg/client/listers/carbon/v1"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"
	glog "k8s.io/klog"
	k8scontroller "k8s.io/kubernetes/pkg/controller"
)

const controllerAgentName = "shardgroup-controller"

// controllerKind contains the schema.GroupVersionKind for this controller type.
var controllerKind = carbonv1.SchemeGroupVersion.WithKind("ShardGroup")

const (
	// SuccessSynced is used as part of the Event 'reason' when a ReplicaSet is synced
	SuccessSynced = "Synced"
	// ErrResourceExists is used as part of the Event 'reason' when a ReplicaSet fails
	// to sync due to a Deployment of the same name already existing.
	ErrResourceExists = "ErrResourceExists"

	// MessageResourceExists is the message used for Events when a resource
	// fails to sync due to a Deployment already existing
	MessageResourceExists = "Resource %q already exists and is not managed by ShardGroup"
	// MessageResourceSynced is the message used for an Event fired when a ReplicaSet
	// is synced successfully
	MessageResourceSynced = "ShardGroup synced successfully"
)

var (
	// DelayProcessDuration is used to delay controller process
	DelayProcessSeconds = 10
)

// InitFlags is for explicitly initializing the flags.
func InitFlags(flagset *pflag.FlagSet) {
	flagset.IntVar(&DelayProcessSeconds, "shardgroup-delay-seconds", DelayProcessSeconds, "the seconds shardgroup wait to process event")
}

// Controller is the controller implementation for ReplicaSet resources
type Controller struct {
	controller.DefaultController
	shardGroupLister listers.ShardGroupLister
	shardGroupSynced cache.InformerSynced
	rollingSetLister listers.RollingSetLister
	rollingSetSynced cache.InformerSynced
	workerLister     listers.WorkerNodeLister
	workerSynced     cache.InformerSynced
}

// NewController returns a new rollingset controller
func NewController(
	kubeclientset kubernetes.Interface,
	carbonclientset clientset.Interface,
	shardGroupInformer informers.ShardGroupInformer,
	rollingSetInformer informers.RollingSetInformer,
	workerInformer informers.WorkerNodeInformer) *Controller {

	// Create event broadcaster
	// Add sample-controller types to the default Kubernetes Scheme so Events can be
	// logged for sample-controller types.
	utilruntime.Must(carboncheme.AddToScheme(scheme.Scheme))

	uidTrackingControllerexpectations := k8scontroller.NewUIDTrackingControllerExpectations(k8scontroller.NewControllerExpectations())

	sgController := &Controller{
		rollingSetLister: rollingSetInformer.Lister(),
		rollingSetSynced: rollingSetInformer.Informer().HasSynced,
		shardGroupLister: shardGroupInformer.Lister(),
		shardGroupSynced: shardGroupInformer.Informer().HasSynced,
		workerLister:     workerInformer.Lister(),
		workerSynced:     workerInformer.Informer().HasSynced,
	}

	sgController.DefaultController = *controller.NewDefaultController(kubeclientset, carbonclientset, controllerAgentName, controllerKind, sgController)
	sgController.DefaultController.ResourceManager = util.NewSimpleResourceManager(kubeclientset, carbonclientset, workerInformer.Informer().GetIndexer(), nil, uidTrackingControllerexpectations)
	sgController.DefaultController.DelayProcessDuration = time.Duration(DelayProcessSeconds) * time.Second

	glog.Info("Setting up event handlers")
	// Set up an event handler for when shardgroup resources change
	shardGroupInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: sgController.EnqueueNow,
		UpdateFunc: func(old, new interface{}) {
			oldSg := old.(*carbonv1.ShardGroup)
			newSg := new.(*carbonv1.ShardGroup)
			if oldSg.ResourceVersion == newSg.ResourceVersion {
				return
			}
			if glog.V(4) {
				glog.Infof("UpdateEvent, sg.name: %s, ResourceVersion: %v", newSg.Name, newSg.ResourceVersion)
			}
			key, err := cache.MetaNamespaceKeyFunc(new)
			if err != nil {
				glog.Errorf("shardgroups get key error")
				return
			}
			sgController.EnqueueAfter(key, time.Duration(DelayProcessSeconds/2)*time.Second)
		},
		// TODO RollingVersino
		DeleteFunc: sgController.EnqueueWithoutEvent,
	})
	// Set up an event handler for when Deployment resources change. This
	// handler will lookup the owner of the given Deployment, and if it is
	// owned by a ReplicaSet resource will enqueue that ReplicaSet resource for
	// processing. This way, we don't need to implement custom logic for
	// handling Deployment resources. More info on this pattern:
	// https://github.com/kubernetes/community/blob/8cafef897a22026d42f5e5bb3f104febe7e29830/contributors/devel/controllers.md
	rollingSetInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: sgController.addRollingSet,
		UpdateFunc: func(old, new interface{}) {
			sgController.HandleSubObjectDelayDuration(new)
		},
		DeleteFunc: sgController.deleteRollingSet,
	})
	return sgController
}

func (c *Controller) addRollingSet(obj interface{}) {
	c.HandleSubObjectDelayDuration(obj)
}

func (c *Controller) deleteRollingSet(obj interface{}) {
	c.HandleSubObjectDelayDuration(obj)
}

// GetObj for controller manager
func (c *Controller) GetObj(namespace, key string) (interface{}, error) {
	foo, err := c.shardGroupLister.ShardGroups(namespace).Get(key)
	if nil != err {
		glog.Warningf("get shardgroup error :%s,%s,%v", namespace, key, err)
	}
	return foo, err
}

// DeleteSubObj is used for controller to handle gc
func (c *Controller) DeleteSubObj(namespace, key string) error {
	rs, err := c.rollingSetLister.RollingSets(namespace).Get(key)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		glog.Warningf("get rollingSet error :%s,%s,%v", namespace, key, err)
		return err
	}

	err = c.ResourceManager.RemoveRollingSet(rs)
	if nil != err {
		if errors.IsNotFound(err) {
			return nil
		}
		glog.Warningf("delete replic error :%s,%s,%v", namespace, key, err)
		return err
	}
	return nil
}

// WaitForCacheSync wait informers synced
func (c *Controller) WaitForCacheSync(stopCh <-chan struct{}) bool {
	if ok := cache.WaitForCacheSync(stopCh, c.shardGroupSynced, c.rollingSetSynced, c.workerSynced); !ok {
		glog.Error("failed to wait for caches to sync")
		return false
	}
	return true
}

// Sync compares the actual state with the desired, and attempts to
// converge the two. It then updates the spec of the RollingSet resource
// with the current spec of the resource.
func (c *Controller) Sync(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		glog.Warningf("invalid resource key: %s", key)
		return nil
	}

	// Get the shardGroup resource with this namespace/name
	shardGroup, err := c.shardGroupLister.ShardGroups(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			glog.Warningf("shardgroup '%v' in work queue no longer exists", key)
			return nil
		}
		return c.delayProcessError(key, err)
	}
	shardGroupCopy := shardGroup.DeepCopy()
	if isReturn, err := c.autoComplete(shardGroupCopy); isReturn || err != nil {
		return c.delayProcessError(key, err)
	}
	if !reflect.DeepEqual(shardGroupCopy, shardGroup) {
		err = c.ResourceManager.UpdateShardGroup(shardGroupCopy)
		if nil != err {
			glog.Warningf("update rollingset spec error :%s,%v", shardGroup.Name, err)
		}
		return c.delayProcessError(key, err)
	}

	errorOccured := false
	errMsg := RawErrors.New("error reason")
	switch shardGroup.Spec.ShardDeployStrategy {
	case carbonv1.RollingSetType:
		rollingSets, err := c.getRollingSetForShardGroup(shardGroup)
		if err != nil {
			if errors.IsNotFound(err) {
				glog.Warningf("shardgroup '%v' in work queue no longer exists", key)
			}
			return c.delayProcessError(key, err)
		}

		if !shardGroupCopy.DeletionTimestamp.IsZero() {
			return c.delayProcessError(key, c.syncRollingSetGC(shardGroupCopy, rollingSets))
		}

		err = carbonv1.DecompressShardTemplates(shardGroupCopy)
		if err != nil {
			glog.Warningf("shardgroup DecompressShardTemplates err %s, %v", key, err)
			return c.delayProcessError(key, err)
		}

		for k := range shardGroupCopy.Spec.ShardTemplates {
			template := shardGroupCopy.Spec.ShardTemplates[k]
			carbonv1.SetDefaultPodVersion(&template.ObjectMeta)
			shardGroupCopy.Spec.ShardTemplates[k] = template
		}
		shardGroupVersion, err := c.calculateShardGroupSign(shardGroupCopy)
		if err != nil {
			return c.delayProcessError(key, err)
		}

		err = c.syncRollingSetSpec(shardGroupCopy, rollingSets, shardGroupVersion)
		if err != nil {
			errorOccured = true
			errMsg = err
		}
		err = c.syncRollingSetGroupStatus(shardGroup, shardGroupCopy, rollingSets, shardGroupVersion)
		if err != nil {
			errorOccured = true
			errMsg = err
		}
	case carbonv1.DeploymentType:
		if !shardGroupCopy.DeletionTimestamp.IsZero() {
			return c.delayProcessError(key, c.syncDeploymentGC(shardGroupCopy))
		}

		if shardGroupCopy.Spec.Paused == true {
			return nil
		}

		err = carbonv1.DecompressShardTemplates(shardGroupCopy)
		if err != nil {
			glog.Warningf("shardgroup DecompressShardTemplates err %s, %v", key, err)
			return c.delayProcessError(key, err)
		}

		err = c.syncDeploymentSpec(shardGroupCopy)
		if err != nil {
			errorOccured = true
			errMsg = err
		}
		err = c.syncDeploymentGroupStatus(shardGroupCopy)
		if err != nil {
			errorOccured = true
			errMsg = err
		}
	}
	if errorOccured {
		return c.delayProcessError(key, errMsg)
	}
	return nil
}

func (c *Controller) delayProcessError(key string, err error) error {
	if err != nil {
		c.EnqueueAfter(key, time.Duration(DelayProcessSeconds/2)*time.Second)
	}
	return nil
}

// 如果不存在SmoothFinalizer，则自动补全
func (c *Controller) autoComplete(shardgroup *carbonv1.ShardGroup) (bool, error) {

	if shardgroup.DeletionTimestamp != nil {
		return false, nil
	}
	if shardgroup.Finalizers == nil {
		shardgroup.Finalizers = []string{}
	}
	if len(shardgroup.Finalizers) == 0 {
		shardgroup.Finalizers = append(shardgroup.Finalizers, carbonv1.FinalizerSmoothDeletion)
	}

	if shardgroup.Labels == nil {
		shardgroup.Labels = map[string]string{}
	}
	carbonv1.AddGroupUniqueLabelHashKey(shardgroup.Labels, shardgroup.Name)

	if shardgroup.Spec.Selector == nil {
		shardgroup.Spec.Selector = new(metav1.LabelSelector)
	}
	newSelector := carbonv1.CloneSelector(shardgroup.Spec.Selector)
	carbonv1.AddGroupSelectorHashKey(newSelector, shardgroup.Name)
	shardgroup.Spec.Selector = newSelector
	return false, nil
}

func (c *Controller) syncDeploymentGC(shardGroup *carbonv1.ShardGroup) error {
	return nil
}

func (c *Controller) syncDeploymentSpec(shardGroup *carbonv1.ShardGroup) error {
	return nil
}

func (c *Controller) syncDeploymentGroupStatus(shardGroup *carbonv1.ShardGroup) error {
	return nil
}
