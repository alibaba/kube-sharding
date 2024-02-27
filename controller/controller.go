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

package controller

import (
	"fmt"
	"sync"
	"time"

	"github.com/alibaba/kube-sharding/common"
	runtime2 "k8s.io/apimachinery/pkg/runtime"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/kubernetes/pkg/api/legacyscheme"

	"github.com/alibaba/kube-sharding/pkg/ago-util/monitor/metricgc"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	"github.com/alibaba/kube-sharding/pkg/memkube/mem"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/time/rate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"

	"github.com/alibaba/kube-sharding/controller/util"
	clientset "github.com/alibaba/kube-sharding/pkg/client/clientset/versioned"
	carbonscheme "github.com/alibaba/kube-sharding/pkg/client/clientset/versioned/scheme"
	carboninformers "github.com/alibaba/kube-sharding/pkg/client/informers/externalversions"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	glog "k8s.io/klog"
)

var (
	notSynced = "failed to wait for caches to sync %s"
)

type delayMode string

const (
	enqueueDelayModeDelay         delayMode = "enqueueDelay"
	enqueueDelayModeNow           delayMode = "enqueueNow"
	enqueueDelayModeDelayDuration delayMode = "enqueueDelayDuration"
	enqueueDelayModeIgnoreOwner   delayMode = "enqueueIgnoreOwner"
)

var processErrDelaySeconds = 15

// InitFlags is for explicitly initializing the flags.
func InitFlags(flagset *pflag.FlagSet) {
	flagset.IntVar(&processErrDelaySeconds, "process-err-delay-seconds", processErrDelaySeconds, "delay seconds for process error")
}

// Controller is a general controllers defines, to handle crds
type Controller interface {
	Run(threadiness int, stopCh <-chan struct{}, waiter *sync.WaitGroup) error
	ProcessNextWorkItem() bool
	Enqueue(obj interface{})
	EnqueueWithoutEvent(obj interface{})
	HandleSubObject(obj interface{})
	HandleSubObjectWithoutEvent(obj interface{})
	Sync(key string) error
	// impl by subclass
	// Get obj from apiserver by namespace and key
	GetObj(namespace, key string) (interface{}, error)
	// DeleteSubObj do garbage collection
	DeleteSubObj(namespace, key string) error
	// wait for informers synced
	WaitForCacheSync(stopCh <-chan struct{}) bool
}

// NewDefaultController create DefaultController
func NewDefaultController(
	kubeclientset kubernetes.Interface,
	carbonclientset clientset.Interface,
	controllerAgentName string,
	controllerKind schema.GroupVersionKind,
	controller Controller) *DefaultController {
	// Create event broadcaster
	// Add sample-controller types to the default Kubernetes Scheme so Events can be
	// logged for sample-controller types.
	utilruntime.Must(carbonscheme.AddToScheme(scheme.Scheme))
	return &DefaultController{
		kubeclientset:   kubeclientset,
		carbonclientset: carbonclientset,
		workqueue: workqueue.NewNamedRateLimitingQueue(workqueue.NewMaxOfRateLimiter(
			workqueue.NewItemExponentialFailureRateLimiter(time.Second, 100*time.Second),
			// 10 qps, 100 bucket size.  This is only for retry speed and its only the overall factor (not per item)
			&workqueue.BucketRateLimiter{Limiter: rate.NewLimiter(rate.Limit(10), 100)},
		), controllerAgentName),
		Controller:           controller,
		controllerAgentName:  controllerAgentName,
		GroupVersionKind:     controllerKind,
		start:                make(chan struct{}),
		caches:               make(map[string]map[string]string),
		DelayProcessDuration: time.Second,
		resourceUpdateTime:   make(map[string]time.Time),
	}
}

var _ Controller = &DefaultController{}

// DefaultController is the abstract controller
type DefaultController struct {
	Controller

	controllerAgentName string
	// GroupVersionKind indicates the controller type.
	// Different instances of this struct may handle different GVKs.
	// For example, this struct can be used (with adapters) to handle ReplicationController.
	schema.GroupVersionKind
	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workqueue workqueue.RateLimitingInterface

	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// carbonclientset is a clientset for our own API group
	carbonclientset clientset.Interface

	ResourceManager util.ResourceManager
	start           chan struct{}

	caches               map[string]map[string]string
	lock                 sync.RWMutex
	DelayProcessDuration time.Duration
	Waiter               *sync.WaitGroup
	StopChan             <-chan struct{}

	resourceUpdateTime map[string]time.Time
	recorder           record.EventRecorder
}

// DeleteResource delete resource record in controller
func (c *DefaultController) DeleteResource(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	delete(c.resourceUpdateTime, key)
}

// AddRateLimited AddRateLimited
func (c *DefaultController) AddRateLimited(key string) {
	c.workqueue.AddRateLimited(key)
}

func (c *DefaultController) getResourceUpdateWaitTimeDuration(key string) time.Duration {
	c.lock.Lock()
	defer c.lock.Unlock()
	updateTime, exist := c.resourceUpdateTime[key]
	now := time.Now()
	if !exist {
		// first time, no delay
		updateTime = now
		c.resourceUpdateTime[key] = now
		return 0
	}
	// | now | updateTime |, should not change updateTime, just wait
	if updateTime.After(now) {
		return updateTime.Sub(now)
	}
	// | updateTime | now | nextUpdateTime, should change updateTime to nextUpdateTime, just wait
	nextUpdateTime := updateTime.Add(c.DelayProcessDuration)
	if nextUpdateTime.After(now) {
		c.resourceUpdateTime[key] = nextUpdateTime
		return nextUpdateTime.Sub(now)
	}
	// | updateTime | nextUpdateTime | now, no delay
	c.resourceUpdateTime[key] = now

	return 0
}

// HandleSubObject will take any resource implementing metav1.Object and attempt
// to find the ReplicaSet resource that 'owns' it. It does this by looking at the
// objects metadata.ownerReferences field for an appropriate OwnerReference.
// It then enqueues that ReplicaSet resource to be processed. If the object does not
// have an appropriate OwnerReference, it will simply be skipped.
func (c *DefaultController) HandleSubObject(obj interface{}) {
	c.handleSubObject(obj, true, enqueueDelayModeNow)
}

// HandleSubObjectWithoutEvent will take any resource implementing metav1.Object and attempt
// to find the ReplicaSet resource that 'owns' it. It does this by looking at the
// objects metadata.ownerReferences field for an appropriate OwnerReference.
// It then enqueues that ReplicaSet resource to be processed. If the object does not
// have an appropriate OwnerReference, it will simply be skipped.
func (c *DefaultController) HandleSubObjectWithoutEvent(obj interface{}) {
	c.handleSubObject(obj, false, enqueueDelayModeNow)
}

// HandleSubObjectDelayDuration  will take any resource implementing metav1.Object and attempt
// to find the owner resource that 'owns' it. It does this by looking at the
// objects metadata.ownerReferences field for an appropriate OwnerReference.
// If the object does not have an appropriate OwnerReference, it will simply be skipped.
// It then enqueues that owner resource with delay duration.
func (c *DefaultController) HandleSubObjectDelayDuration(obj interface{}) {
	c.handleSubObject(obj, true, enqueueDelayModeDelayDuration)
}

// HandlerSubObjectIgnoreOwner will take any resource implementing metav1.Object and attempt
// to find the owner resource that 'owns' it. It does this by looking at the
// objects metadata.ownerReferences field for an appropriate OwnerReference.
// If the object does not have an appropriate OwnerReference, it will simply be skipped.
// It does not enqueues that owner resource.
func (c *DefaultController) HandlerSubObjectIgnoreOwner(obj interface{}) {
	c.handleSubObject(obj, true, enqueueDelayModeIgnoreOwner)
}

func (c *DefaultController) handleSubObject(obj interface{}, event bool, mode delayMode) {
	<-c.start
	var object metav1.Object
	var ok bool
	if object, ok = obj.(metav1.Object); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			runtime.HandleError(fmt.Errorf("error decoding object, invalid type"))
			return
		}
		object, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			runtime.HandleError(fmt.Errorf("error decoding object tombstone, invalid type"))
			return
		}
		glog.Infof("Recovered deleted object '%s' from tombstone", object.GetName())
	}
	if objectType, ok := obj.(carbonv1.ObjKind); ok && event {
		c.RecordCrdChangeLatency(object.GetNamespace()+"/"+object.GetName(), objectType.GetKind(), object)
	}

	if glog.V(4) {
		glog.Infof("%s: Processing sub object: %s", c.controllerAgentName, object.GetName())
	}

	var ownerRef *metav1.OwnerReference
	var err error
	if ownerRef, err = mem.GetControllerOf(object); ownerRef != nil {
		if ownerRef.Kind != c.GroupVersionKind.Kind { // multiple resource kind owns the same kind child resource
			return
		}
		foo, err := c.Controller.GetObj(object.GetNamespace(), ownerRef.Name)
		if err != nil {
			if errors.IsNotFound(err) {
				var kind string
				accessor, err := meta.TypeAccessor(obj)
				if err == nil {
					kind = accessor.GetKind()
				}
				metricgc.WithLabelValuesCounter(gcCounter, object.GetNamespace(), object.GetName()).Inc()
				err = c.Controller.DeleteSubObj(object.GetNamespace(), object.GetName())
				glog.Infof("owner %s/%s/%s not exist, do garbage collection for %s/%s, result %v",
					object.GetNamespace(), ownerRef.Kind, ownerRef.Name, kind, object.GetName(), err)
				return
			}
			glog.Errorf("get obj error '%s' of foo '%s' %v", object.GetSelfLink(), ownerRef.Name, err)
			return
		}
		if mode == enqueueDelayModeIgnoreOwner {
			return
		}
		if glog.V(4) {
			glog.Infof("%s: add obj from sub obj :%s", c.controllerAgentName, object.GetName())
		}
		c.enqueue(foo, false, mode)
		return
	}
	if err != nil {
		glog.Warningf("Get ControllerOf is nil, obj: %s,%s err:%v", object.GetNamespace(), object.GetName(), err)
	}
}

// Enqueue takes a obj resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than ReplicaSet.
func (c *DefaultController) Enqueue(obj interface{}) {
	c.enqueue(obj, true, enqueueDelayModeDelay)
}

// EnqueueNow Enqueue immediately
func (c *DefaultController) EnqueueNow(obj interface{}) {
	c.enqueue(obj, true, enqueueDelayModeNow)
}

// EnqueueWithoutEvent takes a obj resource and converts it into a namespace/name
func (c *DefaultController) EnqueueWithoutEvent(obj interface{}) {
	c.enqueue(obj, false, enqueueDelayModeNow)
}

// EnqueueDelayDuration takes a obj resource and compute a delay to add into
func (c *DefaultController) EnqueueDelayDuration(obj interface{}) {
	c.enqueue(obj, true, enqueueDelayModeDelayDuration)
}

func (c *DefaultController) enqueue(obj interface{}, event bool, mode delayMode) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}

	if object, ok := obj.(metav1.Object); ok {
		labels := object.GetLabels()
		addQueueCounter.WithLabelValues(
			carbonv1.GetSubObjectExtendScopes(object, c.controllerAgentName)...,
		).Inc()

		queueLengthGauge.With(prometheus.Labels{
			"controller": c.controllerAgentName,
		}).Set(float64(c.workqueue.Len()))
		var newLabels = make(map[string]string)
		if event {
			c.RecordCrdChangeLatency(key, c.GroupVersionKind.Kind, object)
		}
		timeString := time.Now().Format(time.RFC3339Nano)
		for k, v := range labels {
			newLabels[k] = v
		}
		newLabels["queueTime"] = timeString
		c.lock.Lock()
		c.caches[key] = newLabels
		c.lock.Unlock()
	}
	if glog.V(4) {
		glog.Infof("%s: Enqueue obj :%s, mode: %s", c.controllerAgentName, key, mode)
	}
	if enqueueDelayModeNow == mode {
		c.workqueue.Add(key)
		return
	}
	if enqueueDelayModeDelayDuration == mode {
		duration := c.getResourceUpdateWaitTimeDuration(key)
		c.workqueue.AddAfter(key, duration)
		return
	}
	c.workqueue.AddAfter(key, c.DelayProcessDuration)
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *DefaultController) Run(threadiness int, stopCh <-chan struct{}, waiter *sync.WaitGroup) error {
	defer runtime.HandleCrash()
	defer c.workqueue.ShutDown()

	c.Waiter = waiter
	c.StopChan = stopCh
	// Start the informer factories to begin populating the informer caches
	glog.Infof("starting %s controller", c.controllerAgentName)
	// Wait for the caches to be synced before starting workers
	glog.Infof("Waiting for informer caches to sync : %s ", c.controllerAgentName)
	if ok := c.WaitForCacheSync(stopCh); !ok {
		err := fmt.Errorf(notSynced, c.controllerAgentName)
		glog.Error(err)
		return err
	}
	glog.Infof("informer caches has synced : %s ", c.controllerAgentName)

	glog.Infof("Starting workers %s", c.controllerAgentName)
	// Launch two workers to process ReplicaSet resources
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.run, time.Second, stopCh)
	}
	close(c.start)
	glog.Infof("Started workers %s", c.controllerAgentName)
	<-stopCh
	glog.Info("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *DefaultController) run() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *DefaultController) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()
	if shutdown {
		return false
	}
	select {
	case <-c.StopChan:
		return false
	default:
	}
	if nil != c.Waiter {
		c.Waiter.Add(1)
		defer c.Waiter.Done()
	}

	// We wrap this block in a func so we can defer c.workqueue.Done.
	func(obj interface{}) error {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off
		// period.
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the
		// workqueue.
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			c.workqueue.Forget(obj)
			glog.Error(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		if glog.V(4) {
			glog.Infof("%s : get obj from queue :%v", c.controllerAgentName, obj)
		}
		c.lock.Lock()
		labels := c.caches[key]
		delete(c.caches, key)
		c.lock.Unlock()
		reportQueueTime(labels, key, c.controllerAgentName)
		// Run the syncHandler, passing it the namespace/name string of the
		// ReplicaSet resource to be synced.
		startTime := time.Now()
		err := c.Sync(key)
		reportSyncMetric(labels, key, c.controllerAgentName, startTime, err)
		if err != nil {
			if !errors.IsNotFound(err) && !errors.IsConflict(err) {
				glog.Errorf("%s sync error %s, %v", c.controllerAgentName, key, err)
			}
			// Put the item back on the workqueue to handle any transient errors.
			c.workqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		c.workqueue.Forget(obj)
		//	glog.Infof("%s successfully synced '%s'", c.controllerAgentName, key)
		return nil
	}(obj)

	queueLengthGauge.With(prometheus.Labels{
		"controller": c.controllerAgentName,
	}).Set(float64(c.workqueue.Len()))
	processCounter.With(prometheus.Labels{
		"controller": c.controllerAgentName,
	}).Inc()
	return true
}

// EnqueueAfter enqueue key to the queue after the indicated duration has passed
func (c *DefaultController) EnqueueAfter(key string, delay time.Duration) {
	if delay != 0 && c.workqueue != nil {
		// Put the item back on the workqueue to handle any transient errors.
		if glog.V(4) {
			glog.Infof("enqueue after %s,%d", key, delay)
		}
		c.workqueue.AddAfter(key, delay)
	}
}

var (
	errUnImpl = fmt.Errorf("unimplemented interface")
)

// DeleteSubObj DeleteSubObj
func (c *DefaultController) DeleteSubObj(namespace, key string) error {
	return errUnImpl
}

// GetObj GetObj
func (c *DefaultController) GetObj(namespace, key string) (interface{}, error) {
	return nil, errUnImpl
}

// GetLatestUpdateTime 获取对象最近的更新时间
func (c *DefaultController) GetLatestUpdateTime(obj interface{}) time.Time {
	var updateCrdTime time.Time
	if object, ok := obj.(metav1.Object); ok {
		annotations := object.GetAnnotations()
		if annotations != nil {
			updateCrdTimestampString := annotations["updateCrdTime"]
			if updateCrdTimestampString != "" {
				crdTime, err := time.Parse(time.RFC3339Nano, updateCrdTimestampString)
				if err != nil {
					updateCrdTime = time.Time{}
				} else {
					updateCrdTime = crdTime
				}
			}
		}
	}
	var lastUpdateStatusTime metav1.Time
	if object, ok := obj.(*carbonv1.RollingSet); ok {
		lastUpdateStatusTime = object.Status.LastUpdateStatusTime
	} else if object, ok := obj.(*carbonv1.Replica); ok {
		lastUpdateStatusTime = object.Status.LastUpdateStatusTime
	} else if object, ok := obj.(*carbonv1.WorkerNode); ok {
		lastUpdateStatusTime = object.Status.LastUpdateStatusTime
	} else if object, ok := obj.(*carbonv1.ServicePublisher); ok {
		lastUpdateStatusTime = object.Status.LastUpdateStatusTime
	} else if object, ok := obj.(*carbonv1.ShardGroup); ok {
		lastUpdateStatusTime = object.Status.LastUpdateStatusTime
	}

	if lastUpdateStatusTime.IsZero() && updateCrdTime.IsZero() {
		return updateCrdTime
	}
	if !lastUpdateStatusTime.IsZero() && !updateCrdTime.IsZero() {
		if lastUpdateStatusTime.Time.Before(updateCrdTime) {
			return updateCrdTime
		}
		return lastUpdateStatusTime.Time
	}
	if lastUpdateStatusTime.IsZero() && !updateCrdTime.IsZero() {
		return updateCrdTime
	}

	if !lastUpdateStatusTime.IsZero() && updateCrdTime.IsZero() {
		return lastUpdateStatusTime.Time
	}

	return updateCrdTime
}

// RecordCrdChangeLatency 记录crd变更到实际处理的延迟
func (c *DefaultController) RecordCrdChangeLatency(key string, kind string, obj metav1.Object) {
	lastUpdateTime := c.GetLatestUpdateTime(obj)
	latency := util.CalcCrdChangeLatency(lastUpdateTime)
	if latency == 0 {
		return
	}
	metricgc.WithLabelValuesSummary(UpdateCrdLatency,
		carbonv1.GetSubObjectExtendScopes(obj, kind, c.controllerAgentName)...,
	).Observe(latency)
	if glog.V(4) {
		glog.Infof("RecordCrdChangeLatency, latency = %v ms, controller = %s", latency, c.controllerAgentName)
	}
}

// RecordCompleteLatency 记录crd变更到实际处理的延迟
func RecordCompleteLatency(meta *metav1.ObjectMeta, kind string, controller string, version string, currCompleted bool) {
	var latency float64
	if !currCompleted {
		latency = util.CalcCompleteLatency(meta.Annotations, version)
	}
	metricgc.WithLabelValuesSummary(CompleteLatency,
		carbonv1.GetSubObjectExtendScopes(meta, kind, controller)...,
	).Observe(latency)
	if glog.V(4) {
		key := meta.Namespace + "/" + meta.Name
		glog.Infof("RecordCompleteLatency, latency = %v ms, controller = %s, key = %s, kind = %s", latency, controller, key, kind)
	}
}

// StartMonitor 启动基础监控
func StartMonitor(carbonInformerFactory carboninformers.SharedInformerFactory,
	informerFactory informers.SharedInformerFactory, namespace string, selector labels.Selector) {
	if namespace == "" && selector == nil {
		return
	}
	glog.Infof("StartMonitor, namespace:%v selector:%v", namespace, selector)
	for range time.Tick(time.Second * 30) {
		recordMonitorInfo(carbonInformerFactory, informerFactory, namespace, selector)
	}
}

func (c *DefaultController) InitRecorder() {
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(&v1.EventSinkImpl{Interface: c.kubeclientset.CoreV1().Events("")})
	c.recorder = eventBroadcaster.NewRecorder(legacyscheme.Scheme, corev1.EventSource{Component: c.controllerAgentName})
}

func (c *DefaultController) Event(obj runtime2.Object, eventtype, reason, message string) {
	if c.recorder != nil && !common.IsInterfaceNil(obj) {
		c.recorder.Event(obj, eventtype, reason, message)
	}
}

func (c *DefaultController) Eventf(obj runtime2.Object, eventtype, reason, messageFmt string, args ...interface{}) {
	if c.recorder != nil && !common.IsInterfaceNil(obj) {
		c.recorder.Eventf(obj, eventtype, reason, messageFmt, args...)
	}
}

func recordMonitorInfo(carbonInformerFactory carboninformers.SharedInformerFactory,
	informerFactory informers.SharedInformerFactory,
	namespace string, selector labels.Selector) {
	rollingsetLister := carbonInformerFactory.Carbon().V1().RollingSets().Lister()
	podLister := informerFactory.Core().V1().Pods().Lister()
	groupLister := carbonInformerFactory.Carbon().V1().ShardGroups().Lister()
	workerLister := carbonInformerFactory.Carbon().V1().WorkerNodes().Lister()
	go func() {
		var groups []*carbonv1.ShardGroup
		var err error
		if namespace != "" {
			groups, err = groupLister.ShardGroups(namespace).List(labels.Set(map[string]string{}).AsSelectorPreValidated())
		} else if selector != nil {
			groups, err = groupLister.List(selector)
		}
		if err == nil {
			groupCounter.Reset()
			for _, group := range groups {
				groupCounter.WithLabelValues(
					carbonv1.GetObjectExtendScopes(group, carbonv1.GetGroupStatus(group))...,
				).Inc()
			}
		}
	}()
	go func() {
		var rollingsets []*carbonv1.RollingSet
		var err error
		if namespace != "" {
			rollingsets, err = rollingsetLister.RollingSets(namespace).List(labels.Set(map[string]string{}).AsSelectorPreValidated())
		} else if selector != nil {
			rollingsets, err = rollingsetLister.List(selector)
		}
		if err == nil {
			rollingsetCounter.WithLabelValues().Set(float64(len(rollingsets)))
			for _, rollingset := range rollingsets {
				statusReplicas := rollingset.Status.Replicas
				activeReplicas := rollingset.Status.ActiveReplicas
				standbyReplicas := rollingset.Status.StandbyReplicas
				availableReplicas := rollingset.Status.AvailableReplicas
				readyReplicas := rollingset.Status.ReadyReplicas
				assignedReplicas := rollingset.Status.AllocatedReplicas
				updatedReplicas := rollingset.Status.UpdatedReplicas
				releasingReplicas := rollingset.Status.ReleasingReplicas
				releasingWorkers := rollingset.Status.ReleasingWorkers
				if !rollingset.Status.Complete {
					latency := util.CalcCompleteLatency(rollingset.Annotations, rollingset.Spec.Version)
					meta := &rollingset.ObjectMeta
					CompleteLatency.WithLabelValues(
						carbonv1.GetSubObjectExtendScopes(meta, rollingset.Kind, "rollingset")...,
					).Observe(latency)
				}

				metricgc.WithLabelValuesGauge(rollingsetRepleasingReplica,
					carbonv1.GetObjectBaseScopes(rollingset)...,
				).Set(float64(releasingReplicas))
				metricgc.WithLabelValuesGauge(rollingsetRepleasingWroker,
					carbonv1.GetObjectBaseScopes(rollingset)...,
				).Set(float64(releasingWorkers))

				uncompleteReplicaCounter.WithLabelValues(
					carbonv1.GetObjectBaseScopes(rollingset)...,
				).Set(float64(rollingset.Status.Replicas - rollingset.Status.ReadyReplicas))

				replicaCounter.WithLabelValues(
					carbonv1.GetObjectBaseScopes(rollingset)...,
				).Set(float64(statusReplicas))

				rollingSetSpotCounter.WithLabelValues(
					carbonv1.GetObjectBaseScopes(rollingset)...,
				).Set(float64(len(rollingset.Status.SpotInstanceStatus.Instances)))
				activeReplicaCounter.WithLabelValues(
					carbonv1.GetObjectBaseScopes(rollingset)...,
				).Set(float64(activeReplicas))

				rollingsetAvailableReplicaRate.WithLabelValues(
					carbonv1.GetObjectBaseScopes(rollingset)...,
				).Set(float64(availableReplicas) / float64(activeReplicas))

				availableReplicaCounter.WithLabelValues(
					carbonv1.GetObjectBaseScopes(rollingset)...,
				).Set(float64(availableReplicas))

				rollingsetReadyReplicaRate.WithLabelValues(
					carbonv1.GetObjectBaseScopes(rollingset)...,
				).Set(float64(readyReplicas) / float64(activeReplicas))

				readyReplicaCounter.WithLabelValues(
					carbonv1.GetObjectBaseScopes(rollingset)...,
				).Set(float64(readyReplicas))

				assignedReplicaCounter.WithLabelValues(
					carbonv1.GetObjectBaseScopes(rollingset)...,
				).Set(float64(assignedReplicas))

				rollingsetLatestReplicaRate.WithLabelValues(
					carbonv1.GetObjectBaseScopes(rollingset)...,
				).Set(float64(updatedReplicas) / float64(activeReplicas))

				belowMinHealth := float64(0)
				if carbonv1.IsBelowMinHealth(rollingset) {
					belowMinHealth = float64(1)
				}
				metricgc.WithLabelValuesGauge(belowMinHealthGauge,
					carbonv1.GetObjectBaseScopes(rollingset)...,
				).Set(belowMinHealth)

				for k, v := range rollingset.Status.UnAssignedWorkers {
					metricgc.WithLabelValuesGauge(rollingsetUnassignedWorker,
						carbonv1.GetObjectExtendScopes(rollingset, k)...,
					).Set(float64(v))
				}

				workerModeMisMatchReplicas := rollingset.Status.WorkerModeMisMatchReplicas
				workeModeMisMatchReplicaCounter.WithLabelValues(
					carbonv1.GetObjectBaseScopes(rollingset)...,
				).Set(float64(workerModeMisMatchReplicas))
				unAssignedStandbyReplicas := rollingset.Status.UnAssignedStandbyReplicas
				standbyReplicaCounter.WithLabelValues(
					carbonv1.GetObjectBaseScopes(rollingset)...,
				).Set(float64(standbyReplicas))
				unAssignedStandbyReplicaCounter.WithLabelValues(
					carbonv1.GetObjectBaseScopes(rollingset)...,
				).Set(float64(unAssignedStandbyReplicas))
				rollingsetUnssignedStandbyReplicaRate.WithLabelValues(
					carbonv1.GetObjectBaseScopes(rollingset)...,
				).Set(float64(unAssignedStandbyReplicas) / float64(standbyReplicas))
				workers, err := workerLister.WorkerNodes(namespace).List(carbonv1.GetRsSelector(rollingset.Name))
				if err == nil {
					metricgc.WithLabelValuesGauge(rollingSetWorkerCounter, carbonv1.GetObjectBaseScopes(rollingset)...).Set(float64(len(workers)))
				}
				existStandbyGPU := false
				if rollingset.Spec.ScaleConfig != nil && rollingset.Spec.ScaleConfig.Enable && rollingset.Spec.Template != nil {
					for _, container := range rollingset.Spec.Template.Spec.Containers {
						if _, existStandbyGPU = container.Resources.Limits[carbonv1.ResourceGPU]; existStandbyGPU {
							break
						}
					}
				}
				pods, err := podLister.Pods(namespace).List(carbonv1.GetRsSelector(rollingset.Name))
				if err == nil {
					rollingsetPodCounter.WithLabelValues(carbonv1.GetObjectBaseScopes(rollingset)...).Set(float64(len(pods)))
				}
			}
		}
	}()

	go func() {
		var workers []*carbonv1.WorkerNode
		var err error
		if namespace != "" {
			workers, err = workerLister.WorkerNodes(namespace).List(labels.Set(map[string]string{}).AsSelectorPreValidated())
		} else if selector != nil {
			workers, err = workerLister.List(selector)
		}
		if err == nil {
			workerCounter.WithLabelValues().Set(float64(len(workers)))
		}
	}()

	go func() {
		var pods []*corev1.Pod
		var err error
		if namespace != "" {
			pods, err = podLister.Pods(namespace).List(labels.Set(map[string]string{}).AsSelectorPreValidated())
		} else if selector != nil {
			pods, err = podLister.List(selector)
		}
		if err == nil {
			podCounter.WithLabelValues().Set(float64(len(pods)))
		}
	}()
}
