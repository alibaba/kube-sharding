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

package rollingset

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"
	glog "k8s.io/klog"
	"k8s.io/utils/integer"

	"github.com/alibaba/kube-sharding/common/features"
	"github.com/alibaba/kube-sharding/controller"
	"github.com/alibaba/kube-sharding/controller/util"
	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	v1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	clientset "github.com/alibaba/kube-sharding/pkg/client/clientset/versioned"
	carboncheme "github.com/alibaba/kube-sharding/pkg/client/clientset/versioned/scheme"
	informers "github.com/alibaba/kube-sharding/pkg/client/informers/externalversions/carbon/v1"
	listers "github.com/alibaba/kube-sharding/pkg/client/listers/carbon/v1"
	corev1 "k8s.io/api/core/v1"
	k8scontroller "k8s.io/kubernetes/pkg/controller"
)

// controllerKind contains the schema.GroupVersionKind for this controller type.
var controllerKind = carbonv1.SchemeGroupVersion.WithKind("RollingSet")

const (
	controllerAgentName = "rollingset-controller"
	// SuccessSynced is used as part of the Event 'reason' when a Rollingset is synced
	SuccessSynced = "Synced"

	historyKeyFileCount   = 30
	historyTotalFileCount = 50
	diffLoggerCache       = historyTotalFileCount * 50
	diffLogRootPath       = "__carbon_record_dir__/"
)

// Controller is the controller implementation for RollingSet resources
type Controller struct {
	controller.DefaultController
	kubeclientset    kubernetes.Interface
	rollingSetLister listers.RollingSetLister
	rollingSetSynced cache.InformerSynced
	workerLister     listers.WorkerNodeLister
	workerSynced     cache.InformerSynced
	// A TTLCache of replica creates/deletes each rc expects to see.
	expectations      *k8scontroller.UIDTrackingControllerExpectations
	diffLogger        *utils.DiffLogger
	diffLoggerCmpFunc CmpFunc
	writeLabels       map[string]string
	cluster           string
}

// NewController returns a new rollingset controller
func NewController(cluster string, kubeclientset kubernetes.Interface, carbonclientset clientset.Interface, rollingSetInformer informers.RollingSetInformer, workerInformer informers.WorkerNodeInformer, writeLabels map[string]string) *Controller {
	utilruntime.Must(carboncheme.AddToScheme(scheme.Scheme))
	uidTrackingControllerexpectations := k8scontroller.NewUIDTrackingControllerExpectations(k8scontroller.NewControllerExpectations())
	rsController := &Controller{
		kubeclientset:    kubeclientset,
		rollingSetLister: rollingSetInformer.Lister(),
		rollingSetSynced: rollingSetInformer.Informer().HasSynced,
		workerLister:     workerInformer.Lister(),
		workerSynced:     workerInformer.Informer().HasSynced,
		expectations:     uidTrackingControllerexpectations,
		writeLabels:      writeLabels,
		cluster:          cluster,
	}

	rsController.DefaultController = *controller.NewDefaultController(
		kubeclientset, carbonclientset, controllerAgentName, controllerKind, rsController)
	rsController.DefaultController.ResourceManager = util.NewSimpleResourceManager(
		kubeclientset, carbonclientset, workerInformer.Informer().GetIndexer(), nil, uidTrackingControllerexpectations)
	rsController.DefaultController.DelayProcessDuration = 1 * time.Second
	diffLogger, err := utils.NewDiffLogger(diffLoggerCache, func(msg string) { glog.Info(msg) }, historyKeyFileCount, historyTotalFileCount)
	if err != nil {
		glog.Fatalf("init difflogger error :%v", err)
		return nil
	}
	rsController.diffLogger = diffLogger
	rsController.diffLoggerCmpFunc = diffLoggerCmpFunc()

	glog.Info("Setting up event handlers")
	// Set up an event handler for when RollingSet resources change
	rollingSetInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: rsController.addRollingset,
		UpdateFunc: func(old, new interface{}) {
			oldRs := old.(*carbonv1.RollingSet)
			newRs := new.(*carbonv1.RollingSet)
			if oldRs.ResourceVersion == newRs.ResourceVersion {
				return
			}
			rsController.enqueueSubrs(new)
			rsController.Enqueue(new)
		},
		DeleteFunc: rsController.deleteRollingset,
	})

	workerInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    rsController.addReplica,
		UpdateFunc: rsController.updateReplica,
		DeleteFunc: rsController.deleteReplica,
	})
	return rsController
}

// GetObj grep rollingSet
func (c *Controller) addReplica(obj interface{}) {
	object, ok := obj.(metav1.Object)
	if ok {
		ownerKey := carbonv1.GetWorkerOwnerKey(object)
		c.expectations.CreationObserved(ownerKey)
		c.EnqueueAfter(ownerKey, time.Second)
	} else {
		c.HandleSubObject(obj)
	}
}

func (c *Controller) deleteReplica(obj interface{}) {
	if object, ok := obj.(metav1.Object); ok {
		c.addWorkerEvent(object)
	} else {
		c.HandleSubObjectWithoutEvent(obj)
	}
}

func (c *Controller) updateReplica(old, new interface{}) {
	// 通知worker迁移前所属的rs，让来源sbrs即时处理
	if object, ok := new.(metav1.Object); ok {
		c.addWorkerEvent(object)
	} else {
		c.HandleSubObject(new)
	}
}

func (c *Controller) addWorkerEvent(object metav1.Object) {
	// 通知worker迁移前所属的rs，让来源sbrs即时处理
	fromSubrs := carbonv1.GetMigratedFromSubrs(object)
	if fromSubrs != "" && fromSubrs != object.GetName() {
		key := fmt.Sprintf("%s/%s", object.GetNamespace(), fromSubrs)
		glog.V(5).Infof("enqueue rollingset %s, from %s", key, object.GetName())
		c.EnqueueAfter(key, time.Second)
	}
	ownerKey := carbonv1.GetWorkerOwnerKey(object)
	glog.V(5).Infof("enqueue rollingset %s, from %s", ownerKey, object.GetName())
	c.EnqueueAfter(ownerKey, time.Second)
}

func (c *Controller) addRollingset(obj interface{}) {
	c.enqueueSubrs(obj)
	c.EnqueueNow(obj)
}

func (c *Controller) deleteRollingset(obj interface{}) {
	c.enqueueSubrs(obj)
	c.EnqueueWithoutEvent(obj)
}

func (c *Controller) enqueueSubrs(obj interface{}) {
	// subrs 变更触发 grouprs事件
	if rs, ok := obj.(*carbonv1.RollingSet); ok {
		if carbonv1.IsSubRollingSet(rs) {
			key := fmt.Sprintf("%s/%s", rs.Namespace, *rs.Spec.GroupRS)
			glog.V(5).Infof("enqueue group rollingset %s", key)
			c.EnqueueAfter(key, time.Second)
		}
	} else {
		glog.V(5).Infof("not rollingset %s", utils.ObjJSON(obj))
	}
}

// GetObj grep rollingSet
func (c *Controller) GetObj(namespace, key string) (interface{}, error) {
	foo, err := c.rollingSetLister.RollingSets(namespace).Get(key)
	if nil != err {
		glog.Warningf("get rollingSet error :%s,%s,%v", namespace, key, err)
	}
	return foo, err
}

// WaitForCacheSync wait informers synced
func (c *Controller) WaitForCacheSync(stopCh <-chan struct{}) bool {
	if ok := cache.WaitForCacheSync(stopCh, c.rollingSetSynced, c.workerSynced); !ok {
		glog.Error("failed to wait for caches to sync")
		return false
	}
	return true
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
			glog.Warningf("RollingSet '%s' in work queue no longer exists", key)
			c.expectations.DeleteExpectations(key)
			return nil
		}
		return err
	}
	if rollingSet.Spec.DaemonSetReplicas != nil {
		return nil // for daemonset
	}
	return c.sync(key, namespace, name, rollingSet.DeepCopy())
}

func (c *Controller) syncRollingSetPvc(rollingSet *carbonv1.RollingSet) error {
	storagetStrategy, exist := rollingSet.Spec.Template.Annotations[carbonv1.AnnotationAppStoragetStrategy]
	if !exist {
		return nil
	}
	if _, exist := rollingSet.Annotations[carbonv1.AnnotationRolePvcName]; exist {
		return nil
	}
	if storagetStrategy != carbonv1.AppStorageStrategyRole {
		glog.Warningf("can't support starage stragety:%v", storagetStrategy)
		return nil
	}

	storageSizeStr, exist := rollingSet.Spec.Template.Annotations[carbonv1.AnnotationAppStoragetSize]
	if !exist {
		storageSizeStr = "20Gi"
	}
	quantity, err := resource.ParseQuantity(storageSizeStr)
	if err != nil {
		glog.Errorf("invalid format for storage size str:%v", storageSizeStr)
		return err
	}

	pvcName := carbonv1.GetRolePvcName(rollingSet)
	if pvcName == "" {
		return fmt.Errorf("can't get pvc name for rollingSet[%v]", rollingSet.Name)
	}
	var pvc = &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:            pvcName,
			Namespace:       rollingSet.Namespace,
			Labels:          rollingSet.Labels,
			Annotations:     rollingSet.Annotations,
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(rollingSet, controllerKind)},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			Resources:        corev1.ResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceName(corev1.ResourceStorage): quantity}},
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			StorageClassName: utils.StringPtr("csi-ultron-disk"), //暂时写死，故意不透出给用户，否则太自由可能导致云盘没法收费等问题
		},
	}

	_, err = c.kubeclientset.CoreV1().PersistentVolumeClaims(rollingSet.Namespace).Create(context.TODO(), pvc, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		glog.Errorf("can't create pvc:%v for rs[%v/%v], err[%v]", pvc.Name, rollingSet.Namespace, rollingSet.Name, err)
		return err
	}

	glog.Infof("success create pvc[%v] for rs:%v", pvc.Name, rollingSet.Name)
	return nil
}

func (c *Controller) sync(key, namespace, name string, rollingSet *carbonv1.RollingSet) error {
	ScheduleID := getScheduleID(name)
	glog.V(5).Infof("sync rollingset: %s", utils.ObjJSON(rollingSet))
	rollingSetCopy := rollingSet.DeepCopy()
	carbonv1.SetDefaultPodVersion(&rollingSetCopy.ObjectMeta)
	if isReturn, err := c.completeFields(rollingSetCopy); isReturn || err != nil {
		return err
	}
	// copy replicas obj
	replicas, err := c.getReplicasForRollingSet(rollingSetCopy)
	if err != nil {
		if errors.IsNotFound(err) {
			glog.Warningf("replicas '%s' in work queue no longer exists", key)
			return nil
		}
		return err
	}
	// 删除保护
	if rollingSetCopy.DeletionTimestamp != nil {
		if isReturn, err := c.deleteOperate(rollingSetCopy, &replicas); isReturn || err != nil {
			glog.Warningf("deleteOperate err, rollingset.name:%s, isReturn:%v, err:%v", rollingSetCopy.Name, isReturn, err)
			return err
		}
	}
	var groupRollingSet *carbonv1.RollingSet
	if carbonv1.IsSubRollingSet(rollingSetCopy) {
		err = c.completeSubRollingSet(rollingSetCopy)
		if err != nil {
			glog.Warningf("completeSubRollingSet err, rollingset.name:%s, err:%s", rollingSetCopy.Name, err.Error())
			return err
		}
		if rollingSetCopy.Spec.Version == "" {
			return nil
		}
		groupRollingSet, err = c.rollingSetLister.RollingSets(rollingSet.Namespace).Get(*rollingSet.Spec.GroupRS)
		if err != nil {
			glog.Warningf("get groupRollingSet err, rollingset.name:%s, %s, err:%s", rollingSetCopy.Name, *rollingSet.Spec.GroupRS, err.Error())
			return err
		}
	} else {
		if rollingSet.Spec.Replicas == nil && !carbonv1.IsSubrsEnable(rollingSet) {
			err = fmt.Errorf("%s replicas is nil", rollingSet.Name)
			glog.Warning(err)
			return err
		}
		if rollingSet.Spec.Template == nil {
			err = fmt.Errorf("%s template is nil", rollingSet.Name)
			glog.Warning(err)
			return err
		}
		err = c.syncRollingSetPvc(rollingSet)
		if err != nil {
			glog.Warningf("sync pvc for rs[%s] err[%v]", rollingSet.Name, err)
			return err
		}
		if err := c.updateVersion(rollingSetCopy); err != nil {
			glog.Warningf("updateVersion err, rollingset.name:%s, err:%s", rollingSetCopy.Name, err.Error())
			return err
		}
	}
	if err := c.updateRollingset(rollingSetCopy, rollingSet); err != nil {
		glog.Warningf("updateRollingset err, rollingset.name:%s, err:%s", rollingSetCopy.Name, err.Error())
		return err
	}
	// 调度groupRollingSet
	if carbonv1.IsGroupRollingSet(rollingSetCopy) {
		return c.syncGroupRollingSet(rollingSetCopy)
	}
	rsNeedsSync := c.expectations.SatisfiedExpectations(key)
	if rsNeedsSync && rollingSetCopy.Spec.Paused == true {
		rsNeedsSync = false
	}
	if rsNeedsSync {
		var scheduler scheduler = newScheduler(key, ScheduleID, rollingSetCopy, groupRollingSet, replicas, &c.ResourceManager, c.expectations, c)
		if err = scheduler.schedule(); err != nil {
			glog.Warningf("rollingset schedule error, rs:%v, error:%v", key, err)
			return err
		}
	} else {
		e, exist, _ := c.expectations.GetExpectations(key)
		if exist && e != nil {
			add, del := e.GetExpectations()
			glog.Warningf("rsNeedsSync is false, not need schedule, rollingset:%v, rs.size:%v, add:%v, del:%v",
				key, len(replicas), add, del)
		}
		glog.V(4).Infof("Enqueue rollingset for period syncing: %s", key)
		// TODO: remove this, a controller will NOT drop events actually.
		// If enqueue with `event=true`, it will report unnecessary UpdateCrdLatency metric. see issue #106095
		c.Enqueue(rollingSetCopy)
	}
	return nil
}

// 重置rollingset的version， 根据versionplan计算
func (c *Controller) updateRollingset(rollingSetCopy, rollingSet *carbonv1.RollingSet) error {
	if utils.ObjJSON(rollingSetCopy.Spec) != utils.ObjJSON(rollingSet.Spec) ||
		utils.ObjJSON(rollingSetCopy.ObjectMeta) != utils.ObjJSON(rollingSet.ObjectMeta) {
		if glog.V(4) {
			glog.Infof("update rollingset before: %s, after: %s", utils.ObjJSON(rollingSet), utils.ObjJSON(rollingSetCopy))
		}
		_, err := c.ResourceManager.UpdateRollingSet(rollingSetCopy)
		if nil != err {
			glog.Warningf("update rollingset spec error :%s,%v", rollingSetCopy.Name, err)
			return err
		}
		//更新过rollingset，需要退出
		return nil
	}
	return nil
}

// 重置rollingset的version， 根据versionplan计算
func (c *Controller) updateVersion(rollingset *carbonv1.RollingSet) error {
	oldVersion := rollingset.Spec.Version
	oldRVersion := rollingset.Spec.ResVersion
	oldInstanceID := rollingset.Spec.InstanceID
	var err error
	rollingset.Spec.Version, rollingset.Spec.ResVersion, err = c.computeVersion(rollingset)
	if oldVersion != rollingset.Spec.Version || oldRVersion != rollingset.Spec.ResVersion {
		glog.V(4).Infof("reset rollingset: %s version, newVersion = '%s', oldVersion = '%s', "+
			"newResourceVersion = '%s', oldResourceVersion = '%s', err = %v",
			rollingset.Name, rollingset.Spec.Version, oldVersion, rollingset.Spec.ResVersion, oldRVersion, err)
	}

	newInstanceID, containersInstanceID, err := carbonv1.ComputeInstanceID(
		carbonv1.NeedRestartAfterResourceChange(&rollingset.Spec.VersionPlan), &rollingset.Spec.Template.Spec)
	for i := range rollingset.Spec.Template.Spec.Containers {
		name := rollingset.Spec.Template.Spec.Containers[i].Name
		v := containersInstanceID[name]
		newInstanceID = newInstanceID + "-" + v
	}
	rollingset.Spec.InstanceID = newInstanceID
	if oldInstanceID != rollingset.Spec.InstanceID {
		glog.Infof("reset rollingset: %s InstanceID, newInstanceID = '%s', oldInstanceID = '%s', error: %v", rollingset.Name, oldInstanceID, rollingset.Spec.InstanceID, err)
	}

	rollingset.Spec.CheckSum, err = CheckSum(rollingset)
	if c.diffLogger != nil {
		diffLoggerErr := c.diffLogger.WriteHistoryFile(diffLogRootPath+"/"+rollingset.Name, rollingset.Name, "_target", rollingset, c.diffLoggerCmpFunc)
		if diffLoggerErr != nil {
			glog.Infof("write diffLog error %s, %v", utils.ObjJSON(rollingset), err)
		}
	}
	return err
}

func (c *Controller) computeVersion(rollingset *carbonv1.RollingSet) (string, string, error) {
	var labels = rollingset.Labels
	if features.C2FeatureGate.Enabled(features.DisableLabelInheritance) {
		labels = nil
	}
	var sign, resSign string
	var err error
	if rollingset.Spec.GangVersionPlan != nil && len(rollingset.Spec.GangVersionPlan) != 0 {
		sign, resSign, err = carbonv1.SignGangVersionPlan(rollingset.Spec.GangVersionPlan, labels, rollingset.Name)
	} else {
		sign, resSign, err = carbonv1.SignVersionPlan(&rollingset.Spec.VersionPlan, labels, rollingset.Name)
	}
	version := c.getGenerationVersion(rollingset.Generation, sign, rollingset.Spec.Version)
	resVersion := c.getGenerationVersion(rollingset.Generation, resSign, rollingset.Spec.ResVersion)
	return version, resVersion, err
}

func (c *Controller) getGenerationVersion(generation int64, sign, currentVersion string) string {
	startIndex := strings.Index(currentVersion, "-")
	if startIndex > 0 {
		curSign := currentVersion[startIndex+1:]
		if curSign == sign {
			return currentVersion
		}
	}
	return strconv.FormatInt(generation, 10) + "-" + sign
}

// rollingset删除流程
func (c *Controller) deleteOperate(rollingset *carbonv1.RollingSet, replicas *[]*carbonv1.Replica) (bool, error) {
	//TODO check netdisk
	if rollingset.DeletionTimestamp == nil {
		return false, nil
	}
	glog.Infof("start deleteOperate rollingset.name: %s, deletionTimestamp: %v, replicas: %d",
		rollingset.Name, rollingset.DeletionTimestamp, len(*replicas))

	if delete := c.hasCanDelete(rollingset, replicas); delete {
		if HasSmoothFinalizer(rollingset) {
			//移除Finalizer
			v1.RemoveFinalizer(rollingset, v1.FinalizerSmoothDeletion)
			//更新状态
			glog.Infof("RemoveFinalizer rollingset.name: %s", rollingset.Name)
			_, err := c.ResourceManager.UpdateRollingSet(rollingset)
			if nil != err {
				return true, err
			}
		}
		if rollingset.Finalizers == nil || len(rollingset.Finalizers) == 0 {
			//达到删除rollingset的状态
			if delete := c.hasCanDelete(rollingset, replicas); delete {
				glog.Infof("call remove rollingset , name = '%s', version = '%s'",
					rollingset.Name, rollingset.Spec.Version)
				if err := c.ResourceManager.RemoveRollingSet(rollingset); err != nil {
					return true, err
				}
			}
		}
		return true, nil
	}

	//未达到删除状态，则继续走缩容流程， schedule内部设置0
	return false, nil
}

func (c *Controller) hasCanDelete(rollingset *carbonv1.RollingSet, replicas *[]*carbonv1.Replica) bool {
	if carbonv1.IsGroupRollingSet(rollingset) {
		subrs, bufferrs, err := c.getSubRollingSets(rollingset)
		if len(subrs) == 0 && bufferrs == nil && err == nil {
			return true
		}
		return false
	}
	if rollingset.DeletionTimestamp != nil && len(*replicas) == 0 {
		return true
	}
	return false
}

// 如果不存在SmoothFinalizer，则自动补全
func (c *Controller) completeFields(rollingset *carbonv1.RollingSet) (bool, error) {
	if rollingset.DeletionTimestamp == nil && !HasSmoothFinalizer(rollingset) {
		v1.AddFinalizer(rollingset, v1.FinalizerSmoothDeletion)
	}
	if nil == rollingset.Labels {
		rollingset.Labels = make(map[string]string)
	}
	// ScaleSchedulePlan和ScaleConfig打算迁移到Capacity，这里做个兼容
	rollingset.Spec.Capacity.ActivePlan.ScaleSchedulePlan = rollingset.Spec.ScaleSchedulePlan
	rollingset.Spec.Capacity.ActivePlan.ScaleConfig = rollingset.Spec.ScaleConfig
	rollingset.Spec.Capacity.ActualReplicas = int(integer.Int32Max(rollingset.GetReplicas(), rollingset.GetRawReplicas()))
	if !carbonv1.IsSubRollingSet(rollingset) {
		carbonv1.AddRsUniqueLabelHashKey(rollingset.Labels, rollingset.Name)
		if nil == rollingset.Spec.Selector {
			rollingset.Spec.Selector = new(metav1.LabelSelector)
		}
		carbonv1.AddRsSelectorHashKey(rollingset.Spec.Selector, rollingset.Name)
	}
	if carbonv1.IsGroupRollingSet(rollingset) {
		if err := c.completeGroupRollingSet(rollingset); err != nil {
			return true, err
		}
	}
	return false, nil
}

// DeleteSubObj do garbage collect
func (c *Controller) DeleteSubObj(namespace, name string) error {
	workerNode, err := c.workerLister.WorkerNodes(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		glog.Warningf("get workerNode error :%s,%s,%v", namespace, name, err)
		return err
	}

	err = c.ResourceManager.DeleteWorkerNode(workerNode)
	if nil != err {
		if errors.IsNotFound(err) {
			return nil
		}
	}
	glog.Warningf("delete workerNode : %s, %s, %v", namespace, name, err)
	return err
}
