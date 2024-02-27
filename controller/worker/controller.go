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

package worker

import (
	"context"
	"fmt"
	"reflect"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	glog "k8s.io/klog"
	"k8s.io/kubernetes/pkg/util/slice"

	"github.com/alibaba/kube-sharding/controller"
	"github.com/alibaba/kube-sharding/controller/util"
	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	clientset "github.com/alibaba/kube-sharding/pkg/client/clientset/versioned"
	carboncheme "github.com/alibaba/kube-sharding/pkg/client/clientset/versioned/scheme"
	informers "github.com/alibaba/kube-sharding/pkg/client/informers/externalversions/carbon/v1"
	listers "github.com/alibaba/kube-sharding/pkg/client/listers/carbon/v1"
)

const (
	controllerAgentName = "carbon-worker-controller"
)

var controllerKind = carbonv1.SchemeGroupVersion.WithKind("WorkerNode")

// Controller is the controller implementation for workernode resources
type Controller struct {
	schema.GroupVersionKind
	controller.DefaultController

	kubeclientset   kubernetes.Interface
	carbonclientset clientset.Interface

	configMapLister       corelisters.ConfigMapLister
	configMapListerSynced cache.InformerSynced
	podLister             corelisters.PodLister
	podListerSynced       cache.InformerSynced
	podInformer           coreinformers.PodInformer
	workerLister          listers.WorkerNodeLister
	workerListerSynced    cache.InformerSynced
	workerInformer        informers.WorkerNodeInformer

	serviceLister       listers.ServicePublisherLister
	serviceListerSynced cache.InformerSynced
	rollingSetLister    listers.RollingSetLister
	rollingSetSynced    cache.InformerSynced

	executor    executor
	writeLabels map[string]string
}

// NewController returns a new workerallocator controller
func NewController(
	carbonclientset clientset.Interface,
	kubeclientset kubernetes.Interface,
	workerInformer informers.WorkerNodeInformer,
	serviceInformer informers.ServicePublisherInformer,
	rollingSetInformer informers.RollingSetInformer,
	podInformer coreinformers.PodInformer,
	configMapInformer coreinformers.ConfigMapInformer,
	writeLabels map[string]string) *Controller {

	utilruntime.Must(carboncheme.AddToScheme(scheme.Scheme))
	c := &Controller{
		kubeclientset:         kubeclientset,
		carbonclientset:       carbonclientset,
		podLister:             podInformer.Lister(),
		podListerSynced:       podInformer.Informer().HasSynced,
		workerLister:          workerInformer.Lister(),
		workerListerSynced:    workerInformer.Informer().HasSynced,
		workerInformer:        workerInformer,
		rollingSetLister:      rollingSetInformer.Lister(),
		rollingSetSynced:      rollingSetInformer.Informer().HasSynced,
		serviceLister:         serviceInformer.Lister(),
		serviceListerSynced:   serviceInformer.Informer().HasSynced,
		configMapLister:       configMapInformer.Lister(),
		configMapListerSynced: configMapInformer.Informer().HasSynced,
		writeLabels:           writeLabels,
	}
	c.DefaultController = *controller.NewDefaultController(kubeclientset, carbonclientset,
		controllerAgentName, controllerKind, c)
	c.InitRecorder()
	c.DefaultController.ResourceManager = util.NewSimpleResourceManager(kubeclientset, carbonclientset, nil, workerInformer.Informer().GetIndexer(), nil)
	executor := &updateExecutor{
		baseExecutor: baseExecutor{
			controller: c,
		},
	}
	c.executor = executor
	workerInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.EnqueueNow,
		UpdateFunc: func(old, new interface{}) {
			oldW := old.(*carbonv1.WorkerNode)
			newW := new.(*carbonv1.WorkerNode)
			if oldW.ResourceVersion == newW.ResourceVersion {
				if glog.V(4) {
					glog.Infof("UpdateEvent resourceVersion equels, worker.name: %s, ResourceVersion: %v", newW.Name, newW.ResourceVersion)
				}
				return
			}
			c.EnqueueNow(new)
		},
		DeleteFunc: c.onDeleteWorker,
	})

	podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.updatePod,
		UpdateFunc: func(old, new interface{}) {
			newDepl := new.(*corev1.Pod)
			oldDepl := old.(*corev1.Pod)
			if newDepl.ResourceVersion == oldDepl.ResourceVersion {
				return
			}
			c.updatePod(new)
		},
		DeleteFunc: c.deletePod,
	})

	return c
}

func (c *Controller) onDeleteWorker(obj interface{}) {
	worker, ok := obj.(*carbonv1.WorkerNode)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("Couldn't get object from tombstone %#v", obj))
			return
		}
		worker, ok = tombstone.Obj.(*carbonv1.WorkerNode)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("Tombstone contained object that is not a WorkerNode %#v", obj))
			return
		}
	}
	if nil == worker.DeletionTimestamp {
		now := metav1.Now()
		worker.DeletionTimestamp = &now
	}
	glog.Infof("on deleted worker: %s %s", worker.Namespace, worker.Name)
	if glog.V(4) {
		glog.Infof("on deleted worker:  %s", worker.String())
	}
	if carbonv1.IsBackupWorker(worker) {
		pairName := getPairWorkerName(worker.Name)
		pairKey := worker.Namespace + "/" + pairName
		err := c.Sync(pairKey)
		if err != nil {
			c.EnqueueAfter(pairKey, time.Duration(10)*time.Second)
		}
		glog.Infof("worker deleted enqueue pair :%s,%v", pairKey, err)
	}
	_, err := c.workerLister.WorkerNodes(worker.Namespace).Get(worker.Name)
	if err != nil && errors.IsNotFound(err) {
		// Request to delete pod manually because there may never pod events later
		glog.Infof("Delete pod by NotFound workerNode %s", worker.Name)
		podName, err := carbonv1.GetPodName(worker)
		if nil != err {
			glog.Errorf("onDeleteWorker with error %s, %v", worker.Name, err)
		}
		c.DeleteSubObj(worker.Namespace, podName)
	}
}

// GetObj grep worker
func (c *Controller) GetObj(namespace, key string) (interface{}, error) {
	foo, err := c.workerLister.WorkerNodes(namespace).Get(key)
	if nil != err {
		glog.Warningf("get workernode error :%s,%s,%v", namespace, key, err)
	}
	return foo, err
}

// WaitForCacheSync wait informers synced
func (c *Controller) WaitForCacheSync(stopCh <-chan struct{}) bool {
	if ok := cache.WaitForCacheSync(stopCh, c.podListerSynced,
		c.workerListerSynced, c.rollingSetSynced, c.serviceListerSynced, c.configMapListerSynced); !ok {
		glog.Error("failed to wait for caches to sync")
		return false
	}
	return true
}

const (
	// SuccessSynced is used as part of the Event 'reason' when a WorkerNode is synced
	SuccessSynced = "Synced"
)

// Sync compares the actual state with the desired, and attempts to converge the two.
func (c *Controller) Sync(key string) error {
	worker, pairWorker, pod, err := c.getResources(key)
	if nil != err {
		if errors.IsNotFound(err) {
			glog.Infof("get resources not found :%s,%v", key, err)
			return nil
		}
		glog.Warningf("get resources error :%s,%v", key, err)
		return err
	}
	scheduler := newWorkerScheduler(worker.DeepCopy(), pairWorker.DeepCopy(), pod, c)
	if scheduler == nil {
		glog.Infof("create worker scheduler error :%s", key)
		return nil
	}

	targetWorker, targetPairWorker, _, err := scheduler.schedule()
	if err != nil {
		glog.Warningf("schedule error :%v", err)
		return err
	}

	err = c.updateWorkers(worker, targetWorker, pairWorker, targetPairWorker)
	if nil != err {
		glog.Warningf("update workers error :%s,%v", targetWorker.Name, err)
		return err
	}

	delay := carbonv1.GetWorkersResyncTime([]*carbonv1.WorkerNode{targetWorker})
	if 0 != delay {
		c.EnqueueAfter(key, time.Duration(delay+10)*time.Second)
		if glog.V(4) {
			glog.Infof("resync workernode %s after %d seconds", key, delay)
		}
	}
	return nil
}

func (c *Controller) updateWorkers(worker, targetWorker,
	pairWorker, targetPairWorker *carbonv1.WorkerNode) error {
	// 优先更新current worker
	var current, currnetTarget, backup, backupTarget *carbonv1.WorkerNode
	if carbonv1.IsCurrentWorker(targetWorker) {
		current, currnetTarget = worker, targetWorker
		backup, backupTarget = pairWorker, targetPairWorker
	} else {
		current, currnetTarget = pairWorker, targetPairWorker
		backup, backupTarget = worker, targetWorker
	}
	err := c.updateWorker(current, currnetTarget)
	err1 := c.updateWorker(backup, backupTarget)
	if nil != err {
		return err
	}
	if nil != err1 {
		return err1
	}
	return nil
}

func (c *Controller) updateWorker(before, after *carbonv1.WorkerNode) error {
	if nil == after {
		return nil
	}

	if carbonv1.IsWorkerReleased(after) {
		err := c.ResourceManager.DeleteWorkerNode(after)
		glog.Infof("delete worker %s,%v", after.Name, err)
		if glog.V(4) {
			glog.Infof("delete worker %s,%v", utils.ObjJSON(after), err)
		}
		if nil != err && !errors.IsNotFound(err) {
			return err
		}
		defaultRecoverQuotas.requireWorkerQuota(after.Name)
		return nil
	}

	if nil == before {
		ret := defaultRecoverQuotas.requireWorkerQuota(after.Name)
		if !ret {
			err := fmt.Errorf("%s :%s recover too frequently", quotaErr, after.Name)
			glog.Error(err)
			return err
		}
		_, err := c.ResourceManager.CreateWorkerNode(after)
		glog.Infof("create worker :%s,%v", after.Name, err)
		if err != nil && !errors.IsAlreadyExists(err) {
			return err
		}
		return nil
	}

	if !reflect.DeepEqual(before.Labels, after.Labels) || !reflect.DeepEqual(before.Spec, after.Spec) ||
		carbonv1.GetGangInfo(before) != carbonv1.GetGangInfo(after) || carbonv1.GetGangPlan(before) != carbonv1.GetGangPlan(after) {
		if glog.V(5) {
			glog.Infof("update worker spec before: %s, after: %s", before, after)
		}
		err := c.ResourceManager.UpdateWorkerNode(after)
		if nil != err && !errors.IsNotFound(err) {
			glog.Errorf("update worker spec %s,%v", after.Name, err)
			return err
		}
		return nil
	}

	if utils.ObjJSON(before.Status) != utils.ObjJSON(after.Status) {
		if glog.V(5) {
			glog.Infof("update worker status before: %s, after: %s", before, after)
		}
		err := c.ResourceManager.UpdateWorkerStatus(after)
		if err != nil && !errors.IsNotFound(err) {
			glog.Warningf("update worker status error :%s,%v", after.Name, err)
			return err
		}
	}
	return nil
}

func (c *Controller) getPodsForWorker(worker *carbonv1.WorkerNode) (*corev1.Pod, error) {
	podName, err := carbonv1.GetPodName(worker)
	if nil != err {
		return nil, err
	}
	pod, err := c.podLister.Pods(worker.Namespace).Get(podName)
	if err != nil {
		if !errors.IsNotFound(err) {
			glog.Warningf("get pods error :%s,%v", worker.Name, err)
			return nil, err
		} else {
			// TODO pod 命名规则变化过程兼容代码， 切换完成后去除
			pod, err = c.podLister.Pods(worker.Namespace).Get(worker.Name)
			if err != nil && !errors.IsNotFound(err) {
				glog.Warningf("get pods error :%s,%v", worker.Name, err)
				return nil, err
			}
			if pod != nil {
				carbonv1.SetWorkerEntity(worker, pod.Name, string(pod.UID))
			}
			return pod, nil
		}
	}
	return pod, nil
}

func (c *Controller) getPairWorker(worker *carbonv1.WorkerNode) (*carbonv1.WorkerNode, error) {
	pairName := getPairWorkerName(worker.Name)
	worker, err := c.workerLister.WorkerNodes(worker.Namespace).Get(pairName)
	if err != nil && !errors.IsNotFound(err) {
		return nil, err
	}

	return worker, nil
}

func (c *Controller) getResources(key string) (
	worker *carbonv1.WorkerNode, pair *carbonv1.WorkerNode, pod *corev1.Pod, err error) {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		glog.Warningf("invalid resource key: %s", key)
		return
	}

	worker, err = c.workerLister.WorkerNodes(namespace).Get(name)
	if err != nil {
		return
	}
	worker = worker.DeepCopy()

	pair, err = c.getPairWorker(worker)
	if err != nil {
		return
	}
	pair = pair.DeepCopy()

	pod, err = c.getPodsForWorker(worker)
	if err != nil {
		return
	}
	pod = pod.DeepCopy()

	return
}

// DeleteSubObj do garbage collect
func (c *Controller) DeleteSubObj(namespace, key string) error {
	err := c.deleteSubPod(namespace, key)
	if nil != err {
		return err
	}
	err = c.deleteSubReserve(namespace, key)
	if nil != err {
		return err
	}
	return err
}

func (c *Controller) deleteSubPod(namespace, key string) error {
	// Don't get pod by bizpod api, because the addon may gc before
	pod, err := c.podLister.Pods(namespace).Get(key)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		glog.Warningf("get pod error :%s,%s,%v", namespace, key, err)
		return err
	}
	if nil != c.executor {
		var grace = true
		executor := &updateExecutor{
			baseExecutor: baseExecutor{
				controller: c,
				grace:      grace,
			},
		}
		glog.Infof("delete pod for garbage collection :%s,%s,%v", namespace, key, err)
		err = executor.deletePod(pod, false)
	}
	return err
}

func (c *Controller) deleteSubReserve(namespace, key string) error {
	return nil
}

func (c *Controller) diffLogWorker(objs ...*carbonv1.WorkerNode) {
	for _, obj := range objs {
		if nil == obj {
			continue
		}
		key := "WorkerNode" + "-" + obj.GetNamespace() + "-" + obj.GetName()
		diffLogger.Log(key+"-log", "[%s] diff log, %s", key, obj.String())
	}
}

func (c *Controller) diffLogPod(objs ...*corev1.Pod) {
	for _, obj := range objs {
		if nil == obj {
			continue
		}
		key := "Pod" + "-" + obj.GetNamespace() + "-" + obj.GetName()
		diffLogger.Log(key+"-log", "[%s] diff log, %s", key, utils.ObjJSON(obj))
	}
}

func (c *Controller) updatePod(obj interface{}) {
	c.handelPodEvent(obj)
	c.HandleSubObject(obj)
}

func (c *Controller) deletePod(obj interface{}) {
	c.handelPodEvent(obj)
	c.HandleSubObjectWithoutEvent(obj)
}

func (c *Controller) handelPodEvent(obj interface{}) {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("Couldn't get object from tombstone %#v", obj))
			return
		}
		pod, ok = tombstone.Obj.(*corev1.Pod)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("Tombstone contained object that is not a Pod %#v", obj))
			return
		}
	}
	c.clearPodFinalizer(pod)
	return
}

func (c *Controller) clearPodFinalizer(pod *corev1.Pod) error {
	if pod.DeletionTimestamp != nil {
		if slice.ContainsString(pod.Finalizers, C2DeleteProtectionFinalizer, nil) {
			patchReq := util.NewStrategicPatch().RemoveFinalizer(C2DeleteProtectionFinalizer)
			_, err := c.kubeclientset.CoreV1().Pods(pod.Namespace).Patch(context.TODO(), pod.Name, patchReq.Type(), patchReq.SimpleData(), metav1.PatchOptions{})
			glog.Infof("clearPodFinalizer %s", pod.Name)
			return err
		}
	}
	return nil
}
