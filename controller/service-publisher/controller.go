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

package publisher

import (
	"context"
	"flag"
	"strconv"
	"sync"
	"time"

	"k8s.io/client-go/kubernetes"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/alibaba/kube-sharding/controller"
	"github.com/alibaba/kube-sharding/controller/util"
	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	clientset "github.com/alibaba/kube-sharding/pkg/client/clientset/versioned"
	carboncheme "github.com/alibaba/kube-sharding/pkg/client/clientset/versioned/scheme"
	carboninformers "github.com/alibaba/kube-sharding/pkg/client/informers/externalversions/carbon/v1"
	listers "github.com/alibaba/kube-sharding/pkg/client/listers/carbon/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"
	glog "k8s.io/klog"
)

const (
	controllerAgentName = "service-publisher"
	serviceFinalizer    = "carbon.taobao.com/service-finalizer"
	publishInterval     = 5000
)

var (
	controllerKind        = carbonv1.SchemeGroupVersion.WithKind("ServicePublisher")
	publisherConcurrent   = 50
	updateConcurrent      = 1000
	cm2CacheSeconds       = 3
	publisherQueryTimeout = time.Second * time.Duration(90)
)

func init() {
	flag.IntVar(&publisherConcurrent, "publish-concurrent", publisherConcurrent, "concurrents of publisher syncs")
	flag.IntVar(&updateConcurrent, "publish-update-concurrent", updateConcurrent, "concurrents of publisher update workers")
	flag.IntVar(&cm2CacheSeconds, "cm2-cache-seconds", cm2CacheSeconds, "seconds cm2 nodes cached")
	flag.DurationVar(&publisherQueryTimeout, "publish-query-timeout", publisherQueryTimeout, "query timeout of publisher")
}

// Controller is the controller implementation for ServicePublisher resources
type Controller struct {
	// GroupVersionKind indicates the controller type.
	// Different instances of this struct may handle different GVKs.
	// For example, this struct can be used (with adapters) to handle ReplicationController.
	schema.GroupVersionKind
	controller.DefaultController
	carbonclientset clientset.Interface
	publisherLister listers.ServicePublisherLister
	publisherSynced cache.InformerSynced

	// workerLister is able to list/get workers and is populated by the shared informer passed to
	// NewController.
	workerLister listers.WorkerNodeLister
	// workerSynced returns true if the worker shared informer has been synced at least once.
	// Added as a member to the struct to allow injection for testing.
	workerSynced   cache.InformerSynced
	executor       *utils.AsyncExecutor
	updateExecutor *utils.AsyncExecutor
	taskManager    *taskManager
}

// NewController returns a new rollingset controller
func NewController(
	kubeclientset kubernetes.Interface,
	carbonclientset clientset.Interface,
	workerInformer carboninformers.WorkerNodeInformer,
	publisherInformer carboninformers.ServicePublisherInformer,
	serviceInformer coreinformers.ServiceInformer) *Controller {

	utilruntime.Must(carboncheme.AddToScheme(scheme.Scheme))

	publishController := &Controller{
		publisherLister: publisherInformer.Lister(),
		publisherSynced: publisherInformer.Informer().HasSynced,
		workerLister:    workerInformer.Lister(),
		workerSynced:    workerInformer.Informer().HasSynced,
		taskManager:     newTaskManager(),
		carbonclientset: carbonclientset,
		executor:        utils.NewExecutor(publisherConcurrent),
		updateExecutor:  utils.NewExecutor(updateConcurrent),
	}
	publishController.DefaultController = *controller.NewDefaultController(kubeclientset, carbonclientset,
		controllerAgentName, controllerKind, publishController)
	publishController.DefaultController.ResourceManager = util.NewSimpleResourceManager(kubeclientset, carbonclientset, workerInformer.Informer().GetIndexer(), nil, nil)

	publisherInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: publishController.Enqueue,
		UpdateFunc: func(old, cur interface{}) {
			publishController.Enqueue(cur)
		},
		DeleteFunc: publishController.EnqueueWithoutEvent,
	})
	return publishController
}

// WaitForCacheSync wait for informers synced
func (c *Controller) WaitForCacheSync(stopCh <-chan struct{}) bool {
	if ok := cache.WaitForCacheSync(stopCh, c.publisherSynced, c.workerSynced); !ok {
		glog.Error("failed to wait for caches to sync")
		return false
	}
	return true
}

// GetObj grep replica
func (c *Controller) GetObj(namespace, key string) (interface{}, error) {
	foo, err := c.publisherLister.ServicePublishers(namespace).Get(key)
	if nil != err {
		glog.Errorf("get replica error :%s,%s,%v", namespace, key, err)
	}
	return foo, err
}

// Sync compares the actual state with the desired, and attempts to converge the two.
func (c *Controller) Sync(key string) error {
	c.taskManager.add(key, c)
	return nil
}

// Deletes the given service publisher.
func (c *Controller) deleteService(service *carbonv1.ServicePublisher) error {
	err := c.ResourceManager.DeleteServicePublisher(service)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	return nil
}

// finalizeService removes the specified finalizerToken and finalizes the service
func (c *Controller) finalizeService(service *carbonv1.ServicePublisher) error {
	service.Finalizers = nil
	err := c.ResourceManager.UpdateServicePublisher(service)
	return err
}

// initFinalizeService add the specified finalizerToken and finalizes the service
func (c *Controller) initFinalizeService(service *carbonv1.ServicePublisher) error {
	if service.Finalizers != nil {
		return nil
	}
	if glog.V(4) {
		glog.Infof("init finalizer %s,%s", service.Namespace, service.Name)
	}
	service.Finalizers = []string{serviceFinalizer}
	err := c.ResourceManager.UpdateServicePublisher(service)
	if err != nil {
		// it was removed already, so life is good
		if errors.IsNotFound(err) {
			return nil
		}
	}
	return err
}

func (c *Controller) updateServiceStatus(service *carbonv1.ServicePublisher, rawservice *carbonv1.ServicePublisher, nodes []carbonv1.IPNode) error {
	var ips []string
	if nil != nodes {
		ips = make([]string, len(nodes))
		for i := range nodes {
			ips[i] = nodes[i].IP
		}
	}
	service.Status.IPs = ips
	service.Status.IPList = nil
	if !isServiceStatusEqual(service.Status, rawservice.Status) {
		if glog.V(4) {
			glog.Infof("update service status :%s", utils.ObjJSON(service))
		}
		err := c.ResourceManager.UpdateServicePublisherStatus(service)
		if err != nil {
			// it was removed already, so life is good
			if errors.IsNotFound(err) {
				return nil
			}
		}
		return err
	}
	return nil
}

func isSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func isServiceStatusEqual(a, b carbonv1.ServicePublisherStatus) bool {
	if !isSliceEqual(a.IPs, b.IPs) {
		return false
	}
	if a.Reason != b.Reason {
		return false
	}
	if a.Message != b.Message {
		return false
	}
	return true
}

func (c *Controller) updateWorkers(service *carbonv1.ServicePublisher, workers []*carbonv1.WorkerNode,
	remotes []carbonv1.IPNode, serviceReleasing, softDelete bool, serviceErr error) error {
	var remoteIps = make(map[string]carbonv1.IPNode)
	for i := range remotes {
		remoteIps[remotes[i].IP] = remotes[i]
	}

	batcher := utils.NewBatcherWithExecutor(c.updateExecutor)
	commands := make([]*utils.Command, 0, len(workers))
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*300)
	defer cancel()
	for i := range workers {
		worker := workers[i]
		command, err := batcher.Go(
			ctx, false, c.updateWorker, service, worker, remoteIps, serviceReleasing, softDelete, &serviceErr)
		if nil != err {
			return err
		}
		commands = append(commands, command)
	}
	batcher.Wait()
	var errs = utils.NewErrors()
	for _, command := range commands {
		if command != nil {
			errs.Add(command.FuncError)
			errs.Add(command.ExecutorError)
		}
	}
	return errs.Error()
}

func (c *Controller) updateWorker(service *carbonv1.ServicePublisher, oldWorker *carbonv1.WorkerNode,
	remoteIps map[string]carbonv1.IPNode, serviceReleasing, softDelete bool, serviceErr *error) error {
	worker := oldWorker.DeepCopy()
	c.syncWorkerServiceStatus(service, worker, remoteIps, serviceReleasing, softDelete, *serviceErr)
	if oldWorker.Status.ServiceStatus != worker.Status.ServiceStatus {
		glog.Infof("[%s] worker ServiceStatus change %s,%s",
			worker.GetQualifiedName(), oldWorker.Status.ServiceStatus, worker.Status.ServiceStatus)
	}
	if utils.ObjJSON(worker.Status) != utils.ObjJSON(oldWorker.Status) {
		err := c.ResourceManager.UpdateWorkerStatus(worker)
		if nil != err {
			if errors.IsNotFound(err) {
				glog.Infof("worker has been deleted %v", worker.Name)
				return err
			}
			glog.Warningf("update worker error :%s %s %v", service.Name, worker.Name, err)
			return err
		}
	}
	return nil
}

func (c *Controller) syncWorkerServiceStatus(service *carbonv1.ServicePublisher, worker *carbonv1.WorkerNode,
	remoteIps map[string]carbonv1.IPNode, serviceReleasing, softDelete bool, serviceErr error) {
	if worker == nil || service == nil {
		return
	}
	errReson := ""
	errMessage := ""
	remoteIP, ok := remoteIps[worker.Status.IP]
	health := remoteIP.Health
	if (serviceReleasing || softDelete || carbonv1.IsWorkerToRelease(worker)) && !ok {
		carbonv1.DeleteWorkerServiceHealthCondition(worker, service.Name, serviceErr)
	} else if !softDelete {
		if !ok {
			errReson = carbonv1.ReasonNotExist
			health = false
		} else if !health {
			errReson = carbonv1.ReasonServiceCheckFalse
			if remoteIP.InWarmup {
				errReson = carbonv1.ReasonWarmUP
			}
		}
		if nil != serviceErr {
			errReson = carbonv1.ReasonQueryError
			errMessage = serviceErr.Error()
		}
		status := v1.ConditionTrue
		if serviceReleasing && health {
			status = v1.ConditionFalse
		} else if !serviceReleasing && !health {
			status = v1.ConditionFalse
		}
		carbonv1.SetWorkerServiceHealthCondition(worker, service, remoteIP.ServiceDetail, status, remoteIP.Score,
			errReson, errMessage, remoteIP.InWarmup, remoteIP.StartWarmup)
		if service.Spec.Type == carbonv1.ServiceCM2 && serviceErr == nil {
			serverInfoMetas := make(map[string]string)
			if !remoteIP.NewAdded {
				worker.Status.ServiceInfoMetas = utils.ObjJSON(serverInfoMetas)
			}
			worker.Status.ServiceInfoMetasRecoverd = true
		} else if service.Spec.Type == carbonv1.ServiceCM2 {
			glog.Infof("do not update ServiceInfoMetas %s, %v, %v", worker.Name, serviceErr, remoteIP.NewAdded)
		}
	}
	if worker.Status.Warmup && carbonv1.IsWorkerInWarmup(worker) && worker.Status.LastWarmupStartTime == 0 {
		worker.Status.LastWarmupStartTime = time.Now().Unix()
		worker.Status.InWarmup = true
	}
}

type publishTask struct {
	c *Controller
	utils.LoopJob
	key       string
	publisher *abstractPublisher
}

func (t *publishTask) start(interval int) {
	glog.Infof("start task %s", t.key)
	t.LoopJob.Start()
	go func() {
		defer t.MarkStop()
		for !t.CheckStop(interval) {
			// 并发控制，避免把后端打挂
			t.c.executor.Do(context.Background(), false, t.sync, t.key)
		}
	}()
}

type taskManager struct {
	tasks map[string]*publishTask
	sync.RWMutex
}

func newTaskManager() *taskManager {
	return &taskManager{
		tasks: make(map[string]*publishTask),
	}
}

func (m *taskManager) add(key string, c *Controller) {
	m.Lock()
	defer m.Unlock()
	if _, ok := m.tasks[key]; ok {
		return
	}
	var task = publishTask{
		key: key,
		c:   c,
	}
	task.start(publishInterval)
	m.tasks[key] = &task
}

func (m *taskManager) remove(key string) {
	m.Lock()
	defer m.Unlock()
	if glog.V(4) {
		glog.Infof("remove task manager %s", key)
	}
	task, ok := m.tasks[key]
	if !ok {
		return
	}
	task.LoopJob.Stop()
	delete(m.tasks, key)
}

func (t *publishTask) sync(key string) error {
	startTime := time.Now()
	defer recordSummaryMetric("null", key, syncLatency, startTime, nil)

	select {
	case <-t.c.StopChan:
		return nil
	default:
	}
	if nil != t.c.Waiter {
		t.c.Waiter.Add(1)
		defer t.c.Waiter.Done()
	}
	rawservice, workers, valid, err := t.getResources(key)
	if nil != err || !valid {
		return err
	}
	recordSummaryMetric(string(rawservice.Spec.Type), key, getResourceLatency, startTime, err)

	service := rawservice.DeepCopy()
	var toPlublisNodes []carbonv1.IPNode
	var localNodes []carbonv1.IPNode
	var remotes []carbonv1.IPNode
	serviceReleasing := service.DeletionTimestamp != nil
	softDelete := service.Spec.SoftDelete
	if !serviceReleasing {
		t.c.initFinalizeService(service)
		toPlublisNodes, localNodes = getLocalNodes(workers, service)
	}
	publishedNodes := getPublishedNodes(service)

	// publisher 创建错误，所有节点unavailable
	publisher := t.getPublisher(key, service, workers)
	if nil != publisher {
		remotes, err = publisher.sync(publishedNodes, toPlublisNodes, localNodes, softDelete)
		glog.V(3).Infof("publisher sync %s used %d seconds", key, int(time.Since(startTime).Seconds()))
		if err != nil {
			glog.Errorf("publisher with error :%s,%v", key, err)
			if t.isNameServerUnavailable(err) {
				glog.Errorf("NameServerUnavailable don't update workers :%s,%v", key, err)
				return err
			}
		} else {
			t.c.saveRemoteNodes(service, rawservice, remotes)
		}
	} else {
		glog.Errorf("nil publisher error :%s", key)
	}
	if glog.V(4) {
		glog.Infof("published remotes key: %s, topublish: %s ,local: %s, remotes :%s",
			key, getIPs(toPlublisNodes), getIPs(localNodes), getIPs(remotes))
	}
	recordSummaryMetric(string(rawservice.Spec.Type), key, publishLatency, startTime, err)

	err = t.c.updateWorkers(service, workers, remotes, serviceReleasing, softDelete, err)
	if nil != err {
		glog.Warningf("update worker error %s, %v", key, err)
		return err
	}
	glog.V(3).Infof("publisher update workers %s used %d seconds", key, int(time.Since(startTime).Seconds()))
	recordSummaryMetric(string(rawservice.Spec.Type), key, updateWorkerLatency, startTime, err)

	if serviceReleasing && len(publishedNodes) == 0 && len(remotes) == 0 && t.conditionRemoved(service.Name, workers) {
		return t.deleteService(key, service)
	}

	if softDelete {
		return t.softDelete(key, service, workers, remotes)
	}

	return nil
}

func (t *publishTask) getResources(key string) (
	service *carbonv1.ServicePublisher,
	workers []*carbonv1.WorkerNode, valid bool, err error) {
	var namespace, name string
	valid = true
	namespace, name, err = cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return
	}
	service, err = t.c.publisherLister.ServicePublishers(namespace).Get(name)
	if err != nil {
		if !errors.IsNotFound(err) {
			glog.Errorf("get service error :%s,%s,%v", namespace, name, err)
			return
		}
		glog.Errorf("nil service :%s,%s,%v", namespace, name, err)
		// http://github.com/alibaba/kube-sharding/issues/115717
		go t.c.taskManager.remove(key)
		err = nil
		valid = false
		return
	}
	if service.Spec.Selector == nil {
		glog.Errorf("selectorless service :%s,%s", namespace, name)
		valid = false
		return
	}

	_, isGang := service.Spec.Selector[carbonv1.LabelGangPartName]
	if _, ok := service.Spec.Selector[carbonv1.DefaultRollingsetUniqueLabelKey]; ok && !isGang {
		workers, err = t.c.ResourceManager.ListWorkerNodeForRS(service.Spec.Selector)
		if err != nil {
			glog.Infof("List workers error %v, %v, %v", key, service.Spec.Selector, err)
			return
		}
	} else {
		workers, err = t.c.workerLister.List(labels.Set(service.Spec.Selector).AsSelectorPreValidated())
		if err != nil {
			glog.Infof("List workers error %v, %v, %v", key, service.Spec.Selector, err)
			return
		}
	}
	return
}

func (t *publishTask) getPublisher(key string,
	service *carbonv1.ServicePublisher,
	workers []*carbonv1.WorkerNode) *abstractPublisher {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return nil
	}
	publisher := t.publisher
	if nil == publisher {
		glog.Infof("create publisher :%s,%s", key, utils.ObjJSON(service.Spec))
		publisher, err = newAbstractPublisher(key, service.Spec)
		if nil != err {
			glog.Errorf("newAbstractPublisher error :%s,%s,%v", namespace, name, err)
		}
		t.publisher = publisher
	}

	var isHostIP = isWorkersHostIP(workers)
	if isHostIP && service.Spec.Type == carbonv1.ServiceSkyline {
		glog.Warningf("Skyline publisher %s with host worker is invalide ", service.Name)
		publisher = nil
		return publisher
	}

	serviceReleasing := service.DeletionTimestamp != nil
	if publisher != nil && publisher.publisher != nil {
		if setter, ok := publisher.publisher.(releasingSetter); ok {
			setter.setReleasing(serviceReleasing)
		}
	}
	return publisher
}

func (t *publishTask) deleteService(key string, service *carbonv1.ServicePublisher) error {
	err := t.c.finalizeService(service)
	if err != nil && !errors.IsNotFound(err) {
		glog.Infof("finalize service error %s,%v", key, err)
		return err
	}

	err = t.c.deleteService(service)
	if err != nil && !errors.IsNotFound(err) {
		glog.Errorf("deleteService error %s,%v", key, err)
		return err
	}
	// http://github.com/alibaba/kube-sharding/issues/115717
	go t.c.taskManager.remove(key)
	return err
}

func (c *Controller) saveRemoteNodes(
	service *carbonv1.ServicePublisher,
	rawservice *carbonv1.ServicePublisher,
	remotes []carbonv1.IPNode) error {
	namespace := service.Namespace
	name := service.Name
	for i := 0; i < 10; i++ {
		err := c.updateServiceStatus(service, rawservice, remotes)
		if nil != err {
			glog.Warningf("update service %s error: %v", service.Name, err)
			if errors.IsNotFound(err) {
				return nil
			}
			if errors.IsConflict(err) {
				service, err = c.carbonclientset.CarbonV1().
					ServicePublishers(namespace).Get(context.Background(), name, metav1.GetOptions{})
				if nil != err && !errors.IsNotFound(err) {
					glog.Infof("get service error :%s , %s , %v", namespace, name, err)
					return err
				}
				continue
			}
			continue
		}
		return nil
	}
	return nil
}

func (t *publishTask) softDelete(key string, service *carbonv1.ServicePublisher, workers []*carbonv1.WorkerNode, remotes []carbonv1.IPNode) error {
	var cleared = true
	for i := range workers {
		for j := range workers[i].Status.ServiceConditions {
			if workers[i].Status.ServiceConditions[j].ServiceName == service.Name &&
				workers[i].Status.ServiceConditions[j].Type == service.Spec.Type {
				cleared = false
				break
			}
		}
	}
	if cleared && len(remotes) == 0 {
		return t.deleteService(key, service)
	}
	return nil
}

func (t *publishTask) conditionRemoved(name string, workers []*carbonv1.WorkerNode) bool {
	for i := range workers {
		for j := range workers[i].Status.ServiceConditions {
			if workers[i].Status.ServiceConditions[j].Name == name || workers[i].Status.ServiceConditions[j].ServiceName == name {
				return false
			}
		}
	}
	return true
}

func (t *publishTask) isNameServerUnavailable(err error) bool {
	if code, ok := err.(codeError); ok {
		if code.GetCode() == ErrorCodeServerUnvailable {
			return true
		}
	}
	return false
}

func isWorkersHostIP(workers []*carbonv1.WorkerNode) bool {
	for _, worker := range workers {
		if worker.Status.IP == "" || worker.Status.HostIP == "" {
			continue
		}
		if worker.Status.IP == worker.Status.HostIP {
			return true
		}
	}
	return false
}

func getPublishedNodes(service *carbonv1.ServicePublisher) []carbonv1.IPNode {
	var publishedNodes = service.Status.IPList
	if nil == publishedNodes {
		publishedNodes = make([]carbonv1.IPNode, 0, len(service.Status.IPs))
	}
	for i := range service.Status.IPs {
		var ipnode = carbonv1.IPNode{
			IP: service.Status.IPs[i],
		}
		publishedNodes = append(publishedNodes, ipnode)
	}
	return publishedNodes
}

func getIPs(nodes []carbonv1.IPNode) []string {
	var ips = make([]string, len(nodes))
	for i := range nodes {
		ip := nodes[i].IP
		for j := range nodes[i].Port {
			ip = utils.AppendString(ip, ":", strconv.Itoa(int(nodes[i].Port[j])))
		}
		ips[i] = ip
	}
	return ips
}
