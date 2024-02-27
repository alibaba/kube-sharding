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

package util

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/types"

	"github.com/alibaba/kube-sharding/common"
	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	clientset "github.com/alibaba/kube-sharding/pkg/client/clientset/versioned"
	v1 "github.com/alibaba/kube-sharding/pkg/client/listers/carbon/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	glog "k8s.io/klog"
	k8scontroller "k8s.io/kubernetes/pkg/controller"
)

// BatchReplicaTimeOut replica批量操作超时时间
const (
	BatchReplicaTimeOut = time.Minute * 10
	ExecutorSize        = 256
	BatcherPodSize      = 5000
)

var (
	ownerKeys = []string{carbonv1.DefaultRollingsetUniqueLabelKey, carbonv1.LabelKeyCarbonJobName, carbonv1.DefaultSubRSLabelKey}
)

// SimpleResourceManager apiservice 资源处理类
type SimpleResourceManager struct {
	kubeclientset     kubernetes.Interface
	carbonclientset   clientset.Interface
	workerNodeIndexer cache.Indexer
	podIndexer        cache.Indexer
	executor          *utils.AsyncExecutor
	expectations      *k8scontroller.UIDTrackingControllerExpectations
}

// NewResourceManager create SimpleResourceManager
func NewResourceManager(
	kubeclientset kubernetes.Interface,
	carbonclientset clientset.Interface,
) ResourceManager {
	return &SimpleResourceManager{
		kubeclientset:   kubeclientset,
		carbonclientset: carbonclientset,
		executor:        utils.NewExecutor(ExecutorSize),
	}
}

// NewSimpleResourceManager create SimpleResourceManager
func NewSimpleResourceManager(
	kubeclientset kubernetes.Interface,
	carbonclientset clientset.Interface,
	workerNodeIndexer cache.Indexer,
	podIndexer cache.Indexer,
	expectations *k8scontroller.UIDTrackingControllerExpectations) ResourceManager {
	manager := &SimpleResourceManager{
		kubeclientset:     kubeclientset,
		carbonclientset:   carbonclientset,
		workerNodeIndexer: workerNodeIndexer,
		podIndexer:        podIndexer,
		executor:          utils.NewExecutor(ExecutorSize),
		expectations:      expectations,
	}
	if workerNodeIndexer != nil {
		manager.addWorkerNodeIndex(workerNodeIndexer)
	}
	if podIndexer != nil {
		manager.addPodIndex(podIndexer)
	}
	return manager
}

func (a *SimpleResourceManager) addPodIndex(indexer cache.Indexer) {
	key := carbonv1.DefaultSlotsetUniqueLabelKey
	indexer.AddIndexers(cache.Indexers{
		key: func(obj interface{}) ([]string, error) {
			if s, ok := obj.(string); ok {
				return []string{s}, nil
			}
			m, err := meta.Accessor(obj)
			if err != nil {
				return []string{}, err
			}
			if v, ok := m.GetLabels()[key]; ok {
				return []string{v}, nil
			}
			return []string{}, nil
		},
	})
}

func (a *SimpleResourceManager) addWorkerNodeIndex(indexer cache.Indexer) {
	for i := range ownerKeys {
		key := ownerKeys[i]
		indexer.AddIndexers(cache.Indexers{
			key: func(obj interface{}) ([]string, error) {
				if s, ok := obj.(string); ok {
					return []string{s}, nil
				}
				m, err := meta.Accessor(obj)
				if err != nil {
					return []string{}, err
				}
				if v, ok := m.GetLabels()[key]; ok {
					return []string{v}, nil
				}
				return []string{}, nil
			},
		})
	}
}

// ListWorkerNodeByOwner ListWorkerNodeByOwner
func (a *SimpleResourceManager) ListWorkerNodeByOwner(selector map[string]string, ownerKey string) ([]*carbonv1.WorkerNode, error) {
	v, ok := selector[ownerKey]
	if !ok {
		return nil, fmt.Errorf("not found owner label %s", ownerKey)
	}
	items, err := a.workerNodeIndexer.Index(ownerKey, v)
	if err != nil {
		glog.Errorf("Index workernodes failed: %v", err)
		return nil, err
	}
	nodes := make([]*carbonv1.WorkerNode, 0, len(items))
	for _, i := range items {
		if worker, ok := i.(*carbonv1.WorkerNode); ok {
			nodes = append(nodes, worker)
		} else {
			glog.Errorf("not workerNode %s", utils.ObjJSON(worker))
		}
	}
	return nodes, nil
}

// PatchPod PatchPod
func (a *SimpleResourceManager) PatchPod(pod *corev1.Pod, pt types.PatchType, data []byte, subresource []string) error {
	start := time.Now()
	_, err := a.kubeclientset.CoreV1().Pods(pod.Namespace).Patch(context.Background(), pod.Name, pt, data, metav1.PatchOptions{}, subresource...)
	RecordAPICall(CallTypePatchPod, start, &pod.ObjectMeta, common.GetRequestID(), err)
	return err
}

// DeletePod delete pod
func (a *SimpleResourceManager) DeletePod(pod *corev1.Pod, grace bool) error {
	start := time.Now()
	var opt metav1.DeleteOptions
	if pod.UID != "" {
		opt = metav1.DeleteOptions{
			Preconditions: &metav1.Preconditions{
				UID: &pod.UID,
			},
		}
	}
	if !grace {
		var gracePeriod = utils.Int64Ptr(0)
		opt.GracePeriodSeconds = gracePeriod
	} else {
		var gracePeriod = utils.Int64Ptr(1)
		opt.GracePeriodSeconds = gracePeriod
	}
	var err error
	err = a.kubeclientset.CoreV1().Pods(pod.Namespace).Delete(context.Background(), pod.Name, opt)
	RecordAPICall(CallTypeDeletePod, start, &pod.ObjectMeta, common.GetRequestID(), err)
	if nil != err {
		return err
	}
	return nil
}

// UpdateWorkerStatus update worker status
func (a *SimpleResourceManager) UpdateWorkerStatus(worker *carbonv1.WorkerNode) error {
	start := time.Now()
	worker.Status.LastUpdateStatusTime = metav1.NewTime(time.Now())
	_, err := a.carbonclientset.CarbonV1().WorkerNodes(worker.Namespace).UpdateStatus(context.Background(), worker, metav1.UpdateOptions{})
	if glog.V(4) {
		glog.Infof("call UpdateWorkerStatus %s, %s", worker.Name, utils.ObjJSON(worker.Status))
	}
	RecordAPICall(CallTypeUpdateWorker, start, &worker.ObjectMeta, common.GetRequestID(), err)
	if nil != err {
		return err
	}
	return nil
}

// UpdatePodSpec update pod spec
func (a *SimpleResourceManager) UpdatePodSpec(pod *corev1.Pod) error {
	start := time.Now()
	UpdateVersionTime(&pod.ObjectMeta, "", start)
	var err error
	_, err = a.kubeclientset.CoreV1().Pods(pod.Namespace).Update(context.Background(), pod, metav1.UpdateOptions{})
	RecordAPICall(CallTypeUpdatePod, start, &pod.ObjectMeta, common.GetRequestID(), err)
	if nil != err {
		return err
	}
	return nil
}

// CreatePod create pod
func (a SimpleResourceManager) CreatePod(pod *corev1.Pod) (*corev1.Pod, error) {
	start := time.Now()
	if glog.V(4) {
		glog.Infof("create pod: %s", utils.ObjJSON(pod))
	}
	UpdateVersionTime(&pod.ObjectMeta, "", start)
	var newPod *corev1.Pod
	var err error
	newPod, err = a.kubeclientset.CoreV1().Pods(pod.Namespace).Create(context.Background(), pod, metav1.CreateOptions{})
	RecordAPICall(CallTypeCreatePod, start, &pod.ObjectMeta, common.GetRequestID(), err)
	if nil != err {
		return nil, err
	}
	return newPod, nil
}

// CreateReplica 创建replica资源,  增加了创建约束，不能直接调用。 改用批量接口
func (a *SimpleResourceManager) CreateReplica(rs *carbonv1.RollingSet, newR *carbonv1.Replica) (*carbonv1.Replica, error) {
	start := time.Now()
	if len(newR.Gang) != 0 {
		return a.syncGangReplica(rs, newR, true, start)
	}
	UpdateCrdTime(&newR.ObjectMeta, start)
	UpdateVersionTime(&newR.ObjectMeta, newR.Spec.Version, start)
	carbonv1.InitWorkerScheduleTimeout(&(newR.WorkerNode.Spec.WorkerSchedulePlan))
	createRs, err := a.carbonclientset.CarbonV1().WorkerNodes(rs.Namespace).Create(context.Background(), &newR.WorkerNode, metav1.CreateOptions{})
	RecordAPICall(CallTypeCreateReplica, start, &newR.ObjectMeta, common.GetRequestID(), err)
	if err != nil {
		if a.expectations != nil {
			a.expectations.CreationObserved(RollingSetKey(rs))
		}
		switch {
		case errors.IsAlreadyExists(err):
			lister := v1.NewWorkerNodeLister(a.workerNodeIndexer).WorkerNodes(newR.Namespace)
			replica, rErr := lister.Get(newR.Name)
			if rErr != nil {
				glog.Warningf("CreateReplica failure, err: %v", err)
				return nil, rErr
			}
			controllerRef := metav1.GetControllerOf(replica)
			//已存在， 返回
			carbonv1.InitWorkerScheduleTimeout(&(rs.Spec.WorkerSchedulePlan))
			if controllerRef != nil && controllerRef.UID == rs.UID && carbonv1.EqualIgnoreHash(&rs.Spec.VersionPlan, &replica.Spec.VersionPlan) {
				createRs = replica
				err = nil
				break
			}
			return nil, err
		case err != nil:
			return nil, err
		}
	}
	return &carbonv1.Replica{WorkerNode: *createRs}, err
}

// UpdateReplica 更新replica
func (a *SimpleResourceManager) UpdateReplica(rs *carbonv1.RollingSet, r *carbonv1.Replica) error {
	start := time.Now()
	if len(r.Gang) != 0 {
		_, err := a.syncGangReplica(rs, r, true, start)
		return err
	}
	UpdateCrdTime(&r.ObjectMeta, start)
	UpdateVersionTime(&r.ObjectMeta, r.Spec.Version, start)
	if nil != rs {
		r.WorkerNode.Spec.OwnerGeneration = rs.Generation
	} else {
		r.WorkerNode.Spec.OwnerGeneration = time.Now().Unix()
	}
	glog.Infof("%s: to version %s,%s,%s,%s", r.GetQualifiedName(), r.Spec.UserDefVersion, r.Spec.Version, r.Spec.ShardGroupVersion, r.Spec.WorkerMode)
	carbonv1.InitWorkerScheduleTimeout(&(r.WorkerNode.Spec.WorkerSchedulePlan))
	_, err := a.carbonclientset.CarbonV1().WorkerNodes(r.Namespace).Update(context.Background(), &r.WorkerNode, metav1.UpdateOptions{})
	RecordAPICall(CallTypeUpdateReplica, start, &r.ObjectMeta, common.GetRequestID(), err)
	if err != nil {
		glog.Warningf("UpdateReplica failure, err: %v", err)
		return err
	}
	return nil
}

// UpdateReplicaStatus 更新replica
func (a *SimpleResourceManager) UpdateReplicaStatus(r *carbonv1.Replica) error {
	start := time.Now()
	r.Status.LastUpdateStatusTime = metav1.NewTime(time.Now())
	_, err := a.carbonclientset.CarbonV1().WorkerNodes(r.Namespace).UpdateStatus(context.Background(), &r.WorkerNode, metav1.UpdateOptions{})
	RecordAPICall(CallTypeUpdateReplica, start, &r.ObjectMeta, common.GetRequestID(), err)
	if err != nil {
		glog.Warningf("UpdateReplicaStatus failure, err: %v", err)
		return err
	}
	return nil
}

// ReleaseReplica 释放replica
func (a *SimpleResourceManager) ReleaseReplica(rs *carbonv1.RollingSet, r *carbonv1.Replica) error {
	start := time.Now()
	r.Spec.ToDelete = true
	if len(r.Gang) != 0 {
		_, err := a.releaseGangReplica(rs, r, start)
		return err
	}
	UpdateCrdTime(&r.ObjectMeta, start)
	_, err := a.carbonclientset.CarbonV1().WorkerNodes(rs.Namespace).Update(context.Background(), &r.WorkerNode, metav1.UpdateOptions{})
	RecordAPICall(CallTypeReleaseReplica, start, &r.ObjectMeta, common.GetRequestID(), err)
	if err != nil {
		glog.Warningf("ReleaseReplica failure, err: %v", err)
		return err
	}
	return nil
}

// RemoveReplica 移除
func (a *SimpleResourceManager) RemoveReplica(rs *carbonv1.RollingSet, r *carbonv1.Replica) error {
	start := time.Now()
	err := a.carbonclientset.CarbonV1().WorkerNodes(r.Namespace).Delete(context.Background(), r.Name, metav1.DeleteOptions{})
	RecordAPICall(CallTypeDeleteReplica, start, &r.ObjectMeta, common.GetRequestID(), err)
	if err != nil {
		//资源已被删除
		if errors.IsNotFound(err) {
			return nil
		}
		glog.Warningf("RemoveReplicaError, name=%v, err=%v", ReplicaKey(r), err)
		return err
	}
	return err
}

// UpdateRollingSet spec  更新
func (a *SimpleResourceManager) UpdateRollingSet(rs *carbonv1.RollingSet) (*carbonv1.RollingSet, error) {
	start := time.Now()
	//update spec version
	var err error
	UpdateCrdTime(&rs.ObjectMeta, start)
	UpdateVersionTime(&rs.ObjectMeta, rs.Spec.Version, start)
	rsResult, err := a.carbonclientset.CarbonV1().RollingSets(rs.Namespace).Update(context.Background(), rs, metav1.UpdateOptions{})
	RecordAPICall(CallTypeUpdateRollingset, start, &rs.ObjectMeta, common.GetRequestID(), err)
	if err != nil {
		glog.Warningf("UpdateRollingSet failure, err: %v", err)
		return rsResult, err
	}
	return rsResult, err
}

// UpdateRollingSetStatus status  更新
func (a *SimpleResourceManager) UpdateRollingSetStatus(rs *carbonv1.RollingSet) error {
	start := time.Now()
	//update status
	var err error
	rs.Status.LastUpdateStatusTime = metav1.NewTime(time.Now())
	_, err = a.carbonclientset.CarbonV1().RollingSets(rs.Namespace).UpdateStatus(context.Background(), rs, metav1.UpdateOptions{})
	RecordAPICall(CallTypeUpdateRollingset, start, &rs.ObjectMeta, common.GetRequestID(), err)
	if err != nil {
		glog.Warningf("UpdateRollingSetStatus failure, err: %v", err)
		return err
	}
	return err
}

// RemoveRollingSet 移除RollingSet
func (a *SimpleResourceManager) RemoveRollingSet(rs *carbonv1.RollingSet) error {
	if err := a.carbonclientset.CarbonV1().RollingSets(rs.Namespace).Delete(context.Background(), rs.Name, metav1.DeleteOptions{}); err != nil {
		return err
	}
	return nil
}

// UpdateWorkerNode 更新workernode
func (a *SimpleResourceManager) UpdateWorkerNode(workernode *carbonv1.WorkerNode) error {
	start := time.Now()
	UpdateCrdTime(&workernode.ObjectMeta, start)
	UpdateVersionTime(&workernode.ObjectMeta, workernode.Spec.Version, start)
	_, err := a.carbonclientset.CarbonV1().WorkerNodes(workernode.Namespace).Update(context.Background(), workernode, metav1.UpdateOptions{})
	RecordAPICall(CallTypeUpdateWorker, start, &workernode.ObjectMeta, common.GetRequestID(), err)
	if nil != err {
		glog.Warningf("UpdateWorkerNode failure, err: %v", err)
		return err
	}
	return nil
}

// CreateWorkerNode 创建workernode
func (a *SimpleResourceManager) CreateWorkerNode(worker *carbonv1.WorkerNode) (*carbonv1.WorkerNode, error) {
	start := time.Now()
	UpdateCrdTime(&worker.ObjectMeta, start)
	UpdateVersionTime(&worker.ObjectMeta, worker.Spec.Version, start)
	newWorker, err := a.carbonclientset.CarbonV1().WorkerNodes(worker.Namespace).Create(context.Background(), worker, metav1.CreateOptions{})
	RecordAPICall(CallTypeCreateWorker, start, &worker.ObjectMeta, common.GetRequestID(), err)
	if nil != err {
		return nil, err
	}
	return newWorker, err
}

// DeleteWorkerNode 删除workernode
func (a *SimpleResourceManager) DeleteWorkerNode(worker *carbonv1.WorkerNode) error {
	start := time.Now()
	err := a.carbonclientset.CarbonV1().WorkerNodes(worker.Namespace).Delete(context.Background(), worker.Name, metav1.DeleteOptions{})
	RecordAPICall(CallTypeDeleteWorker, start, &worker.ObjectMeta, common.GetRequestID(), err)
	if nil != err {
		return err
	}
	return nil
}

// ReleaseWorkerNode 释放WorkerNode和对应的pod
func (a *SimpleResourceManager) ReleaseWorkerNode(worker *carbonv1.WorkerNode) error {
	start := time.Now()
	worker.Spec.ToDelete = true
	glog.Infof("Set workerNode ToDelete: %s", worker.Name)
	_, err := a.carbonclientset.CarbonV1().WorkerNodes(worker.Namespace).Update(context.Background(), worker, metav1.UpdateOptions{})
	RecordAPICall(CallTypeDeleteWorker, start, &worker.ObjectMeta, common.GetRequestID(), err)
	return err
}

// ListWorkerNodeForRS list workernodes for rollingset
func (a *SimpleResourceManager) ListWorkerNodeForRS(selector map[string]string) ([]*carbonv1.WorkerNode, error) {
	ownerKey := carbonv1.DefaultRollingsetUniqueLabelKey
	if _, ok := selector[carbonv1.DefaultSubRSLabelKey]; ok {
		ownerKey = carbonv1.DefaultSubRSLabelKey
	}
	return a.ListWorkerNodeByOwner(selector, ownerKey)
}

// CreateServicePublisher create obj
func (a *SimpleResourceManager) CreateServicePublisher(p *carbonv1.ServicePublisher) error {
	_, err := a.carbonclientset.CarbonV1().ServicePublishers(p.Namespace).Create(context.Background(), p, metav1.CreateOptions{})
	if nil != err {
		glog.Errorf("create service publisher err, service:%s, err:%v", utils.ObjJSON(p), err)
		return err
	}
	return nil
}

// DeleteServicePublisher delete obj
func (a *SimpleResourceManager) DeleteServicePublisher(p *carbonv1.ServicePublisher) error {
	err := a.carbonclientset.CarbonV1().ServicePublishers(p.Namespace).Delete(context.Background(), p.Name, metav1.DeleteOptions{})
	if nil != err {
		return err
	}
	return nil
}

// DeleteServicePublisherForRs delete obj
func (a *SimpleResourceManager) DeleteServicePublisherForRs(rs *carbonv1.RollingSet) error {
	if nil == rs {
		return nil
	}
	services, err := a.carbonclientset.CarbonV1().
		ServicePublishers(rs.Namespace).
		List(context.Background(), metav1.ListOptions{LabelSelector: carbonv1.GetRsStringSelector(rs.Name)})
	if nil == services || nil != err {
		return err
	}
	for i := range services.Items {
		service := services.Items[i]
		err := a.carbonclientset.CarbonV1().ServicePublishers(service.Namespace).Delete(context.Background(), service.Name, metav1.DeleteOptions{})
		glog.Infof("delete service %s, %s, %v", service.Namespace, service.Name, err)
		if nil != err {
			return err
		}
	}
	return nil
}

// UpdateServicePublisher update obj
func (a *SimpleResourceManager) UpdateServicePublisher(p *carbonv1.ServicePublisher) error {
	UpdateCrdTime(&p.ObjectMeta, time.Now())
	_, err := a.carbonclientset.CarbonV1().ServicePublishers(p.Namespace).Update(context.Background(), p, metav1.UpdateOptions{})
	if nil != err {
		return err
	}
	return nil
}

// UpdateServicePublisherStatus update obj status
func (a *SimpleResourceManager) UpdateServicePublisherStatus(p *carbonv1.ServicePublisher) error {
	p.Status.LastUpdateStatusTime = metav1.NewTime(time.Now())
	_, err := a.carbonclientset.CarbonV1().ServicePublishers(p.Namespace).UpdateStatus(context.Background(), p, metav1.UpdateOptions{})
	if nil != err {
		return err
	}
	return nil
}

// ListReplicaForRS list replicas for rollingset
func (a *SimpleResourceManager) ListReplicaForRS(rs *carbonv1.RollingSet) ([]*carbonv1.Replica, error) {
	selector, err := metav1.LabelSelectorAsMap(rs.Spec.Selector)
	if err != nil {
		glog.Errorf("No slector in rollingset spec: %s", rs.Name)
		return nil, err
	}
	workers, err := a.ListWorkerNodeForRS(selector)
	if err != nil {
		glog.Warningf("List workernodes failed, err: %v", err)
		return nil, err
	}
	var replicas = make([]*carbonv1.Replica, 0, len(workers))
	var workerPairs map[string][]*carbonv1.WorkerNode = make(map[string][]*carbonv1.WorkerNode)
	for i := range workers {
		paris := workerPairs[carbonv1.GetReplicaID(workers[i])]
		if nil != paris {
			paris = append(paris, workers[i])
		} else {
			paris = []*carbonv1.WorkerNode{workers[i]}
		}
		workerPairs[carbonv1.GetReplicaID(workers[i])] = paris
	}
	for _, pairs := range workerPairs {
		current, backup, _ := carbonv1.GetCurrentWorkerNodeV2(pairs, true)
		if current == nil {
			glog.Errorf("get current %s", utils.ObjJSON(pairs))
			continue
		}
		replica := &carbonv1.Replica{WorkerNode: *current}
		if backup != nil {
			replica.Backup = *backup.DeepCopy()
		}
		replicas = append(replicas, replica)
	}
	if len(rs.Spec.GangVersionPlan) == 0 {
		return replicas, nil
	}
	return a.listGangReplicas(rs, replicas)
}

// listGangReplicas list gang replicas for rollingset
func (a *SimpleResourceManager) listGangReplicas(rs *carbonv1.RollingSet, replicas []*carbonv1.Replica) ([]*carbonv1.Replica, error) {
	var gangReplicas = []*carbonv1.Replica{}
	var gangPartReplicas = map[string][]*carbonv1.Replica{}
	for i := range replicas {
		replica := replicas[i]
		if carbonv1.IsGangMainPart(replica.Spec.Template) {
			gangReplicas = append(gangReplicas, replica.DeepCopy())
		} else {
			gangID := carbonv1.GetGangID(replica)
			if gangID != "" {
				partReplicas := gangPartReplicas[gangID]
				if partReplicas == nil {
					partReplicas = []*carbonv1.Replica{replica}
				} else {
					partReplicas = append(partReplicas, replica)
				}
				gangPartReplicas[gangID] = partReplicas
			}
		}
		replicas = append(replicas, replica)
	}
	for i := range gangReplicas {
		replica := gangReplicas[i]
		replica.Gang = []carbonv1.Replica{*replica}
		gangID := carbonv1.GetGangID(replica)
		partReplicas := gangPartReplicas[gangID]
		for j := range partReplicas {
			partReplica := partReplicas[j]
			replica.Gang = append(replica.Gang, *partReplica)
		}
	}
	for gangID := range gangPartReplicas {
		var found = false
		for i := range gangReplicas {
			replica := gangReplicas[i]
			replicaGangID := carbonv1.GetGangID(replica)
			if gangID == replicaGangID {
				found = true
				break
			}
		}
		if !found {
			partReplicas := gangPartReplicas[gangID]
			glog.Infof("patch gangReplicas not found %s, %s", gangID, partReplicas)
			var replica *carbonv1.Replica
			for j := range partReplicas {
				partReplica := partReplicas[j]
				if replica == nil {
					replica = partReplica.DeepCopy()
					replica.Gang = []carbonv1.Replica{}
				}
				replica.Gang = append(replica.Gang, *partReplica)
			}
			gangReplicas = append(gangReplicas, replica)
		}
	}
	for i := range gangReplicas {
		synced := a.aggregateGangStatus(gangReplicas[i])
		if !synced {
			glog.Infof("sync gang replica %s", gangReplicas[i].Name)
			a.syncGangReplica(rs, gangReplicas[i], false, time.Now())
		}
	}
	return gangReplicas, nil
}

func (a *SimpleResourceManager) aggregateGangStatus(replica *carbonv1.Replica) bool {
	var versionPlans map[string]*carbonv1.VersionPlan
	var synced = true
	var gangInfos = map[string]string{}
	for i := range replica.Gang {
		worker := &replica.Gang[i].WorkerNode
		part := carbonv1.GetC2RoleName(worker)
		gangInfos[part] = worker.Status.IP
	}
	if !carbonv1.IsGangMainPart(replica.Spec.Template) {
		if time.Now().Sub(replica.CreationTimestamp.Time) > time.Minute*3 {
			glog.Infof("not mainpart %s ", utils.ObjJSON(replica))
			replica.WorkerNode.Status.AllocStatus = carbonv1.WorkerLost
			replica.Spec.Version = ""
		}
	} else {
		plan := carbonv1.GetGangPlan(replica)
		err := json.Unmarshal([]byte(plan), &versionPlans)
		if err != nil {
			glog.Errorf("unmarshal with error %s, %v", replica.Name, err)
		}
		var gangStatus *carbonv1.WorkerNodeStatus
		var generation int
		for i := range replica.Gang {
			gangReplica := replica.Gang[i]
			info := carbonv1.GetGangInfo(&gangReplica)
			var replicaGangInfo map[string]carbonv1.GangInfo
			json.Unmarshal([]byte(info), &replicaGangInfo)
			if len(replicaGangInfo) != len(gangInfos) {
				synced = false
			}
			for k := range replicaGangInfo {
				if replicaGangInfo[k].IP != gangInfos[k] {
					synced = false
					break
				}
			}
			if gangStatus == nil {
				gangStatus = gangReplica.Status.DeepCopy()
				generation = int(gangReplica.Spec.OwnerGeneration)
			} else {
				if int(gangReplica.Spec.OwnerGeneration) < generation {
					gangStatus = gangReplica.Status.DeepCopy()
					generation = int(gangReplica.Spec.OwnerGeneration)
				} else if int(gangReplica.Spec.OwnerGeneration) == generation && gangReplica.Status.Score < gangStatus.Score {
					gangStatus = gangReplica.Status.DeepCopy()
					generation = int(gangReplica.Spec.OwnerGeneration)
				}
			}
			if gangReplica.Spec.Version != replica.Spec.Version || carbonv1.GetGangInfo(&gangReplica) != carbonv1.GetGangInfo(replica) {
				synced = false
			}
		}
		if versionPlans != nil && len(versionPlans) != len(replica.Gang) {
			if time.Now().Sub(replica.CreationTimestamp.Time) > time.Minute*3 {
				replica.WorkerNode.Status.AllocStatus = carbonv1.WorkerLost
			}
			synced = false
		}
		if gangStatus != nil {
			glog.V(5).Infof("use %s as gang status for %s, %d", utils.ObjJSON(gangStatus), replica.Name, len(replica.Gang))
			replica.WorkerNode.Status = *gangStatus
		}
	}

	return synced
}

func (a *SimpleResourceManager) syncGangReplica(rs *carbonv1.RollingSet, replica *carbonv1.Replica, syncPlan bool, start time.Time) (*carbonv1.Replica, error) {
	var versionPlans map[string]*carbonv1.VersionPlan
	var mainWorker *carbonv1.WorkerNode
	var gangID string
	var gangInfos = map[string]carbonv1.GangInfo{}

	for i := range replica.Gang {
		worker := &replica.Gang[i].WorkerNode
		part := carbonv1.GetC2RoleName(worker)
		var gangInfo = carbonv1.GangInfo{
			Name: part,
			IP:   worker.Status.IP,
		}
		gangInfos[part] = gangInfo
		if carbonv1.IsGangMainPart(worker.Spec.Template) {
			mainWorker = worker
			plan := carbonv1.GetGangPlan(worker)
			err := json.Unmarshal([]byte(plan), &versionPlans)
			if err != nil {
				glog.Errorf("unmarshal with error %s, %v", worker.Name, err)
			}
			r := replica.Gang[0]
			replica.Gang[0] = replica.Gang[i]
			replica.Gang[i] = r
			gangID = carbonv1.GetGangID(worker)
		}
	}
	if mainWorker == nil {
		return replica, fmt.Errorf("not found main worker in %s", carbonv1.GetGangID(replica))
	}
	gangInfo := utils.ObjJSON(gangInfos)
	for i := range replica.Gang {
		old := &replica.Gang[i].WorkerNode
		worker := old.DeepCopy()
		carbonv1.SetGangInfo(worker, gangInfo)
		if worker.UID == "" {
			UpdateCrdTime(&worker.ObjectMeta, start)
			UpdateVersionTime(&worker.ObjectMeta, worker.Spec.Version, start)
			carbonv1.InitWorkerScheduleTimeout(&(worker.Spec.WorkerSchedulePlan))
			glog.Infof("create worker %s: to version %s,%s,%s,%s", worker.GetQualifiedName(), worker.Spec.UserDefVersion, worker.Spec.Version, worker.Spec.ShardGroupVersion, worker.Spec.WorkerMode)
			_, err := a.carbonclientset.CarbonV1().WorkerNodes(rs.Namespace).Create(context.Background(), worker, metav1.CreateOptions{})
			RecordAPICall(CallTypeCreateReplica, start, &worker.ObjectMeta, common.GetRequestID(), err)
			if err != nil {
				glog.Warningf("CreateReplica failure, err: %s, %v", worker.Name, err)
				if carbonv1.IsGangMainPart(worker.Spec.Template) {
					return nil, err
				}
			}
		} else if old.Spec.Version != mainWorker.Spec.Version || carbonv1.GetGangInfo(old) != gangInfo || syncPlan {
			if old.Spec.Version != mainWorker.Spec.Version && !syncPlan {
				plan, ok := versionPlans[carbonv1.GetGangPartName(worker)]
				if ok {
					worker.Spec.Version = mainWorker.Spec.Version
					worker.Spec.ResVersion = mainWorker.Spec.ResVersion
					worker.Spec.WorkerMode = mainWorker.Spec.WorkerMode
					worker.Spec.DependencyReady = mainWorker.Spec.DependencyReady
					worker.Spec.OwnerGeneration = mainWorker.Spec.OwnerGeneration
					worker.Spec.VersionPlan = *plan
				}
			}
			worker.Spec.ResourcePool = replica.Spec.ResourcePool
			worker.Spec.DeletionCost = replica.Spec.DeletionCost
			worker.Spec.StandbyHours = replica.Spec.StandbyHours
			worker.Spec.WorkerMode = replica.Spec.WorkerMode
			UpdateCrdTime(&worker.ObjectMeta, start)
			UpdateVersionTime(&worker.ObjectMeta, worker.Spec.Version, start)
			if nil != rs {
				worker.Spec.OwnerGeneration = rs.Generation
			} else {
				worker.Spec.OwnerGeneration = time.Now().Unix()
			}
			glog.Infof("%s: to version %s,%s,%s,%s", worker.GetQualifiedName(), worker.Spec.UserDefVersion, worker.Spec.Version, worker.Spec.ShardGroupVersion, worker.Spec.WorkerMode)
			_, err := a.carbonclientset.CarbonV1().WorkerNodes(worker.Namespace).Update(context.Background(), worker, metav1.UpdateOptions{})
			RecordAPICall(CallTypeUpdateReplica, start, &worker.ObjectMeta, common.GetRequestID(), err)
			if err != nil {
				glog.Warningf("UpdateReplica failure, err: %s, %v", worker.Name, err)
				if carbonv1.IsGangMainPart(worker.Spec.Template) {
					return nil, err
				}
			}
		}
	}

	for k := range versionPlans {
		found := false
		for i := range replica.Gang {
			worker := &replica.Gang[i].WorkerNode
			partName := carbonv1.GetGangPartName(worker)
			if partName == k {
				found = true
			}
		}
		if !found {
			worker := NewWorker(rs.Name, k, gangID, rs, []metav1.OwnerReference{*metav1.NewControllerRef(rs, carbonv1.SchemeGroupVersion.WithKind("RollingSet"))}, rs.Labels, rs.Annotations)
			plan := versionPlans[k]
			worker.Spec.VersionPlan = *plan
			worker.Spec.Version = mainWorker.Spec.Version
			worker.Labels = carbonv1.PatchMap(worker.Labels, worker.Spec.Template.Labels, carbonv1.LabelKeyRoleName, carbonv1.LabelKeyRoleNameHash)
			worker.Annotations = carbonv1.PatchMap(worker.Annotations, worker.Spec.Template.Annotations, carbonv1.LabelKeyRoleName, carbonv1.LabelKeyRoleNameHash)
			UpdateCrdTime(&worker.ObjectMeta, start)
			UpdateVersionTime(&worker.ObjectMeta, worker.Spec.Version, start)
			carbonv1.InitWorkerScheduleTimeout(&(worker.Spec.WorkerSchedulePlan))
			glog.Infof("fix worker %s: to version %s,%s,%s,%s", worker.GetQualifiedName(), worker.Spec.UserDefVersion, worker.Spec.Version, worker.Spec.ShardGroupVersion, worker.Spec.WorkerMode)
			_, err := a.carbonclientset.CarbonV1().WorkerNodes(rs.Namespace).Create(context.Background(), worker, metav1.CreateOptions{})
			RecordAPICall(CallTypeCreateReplica, start, &worker.ObjectMeta, common.GetRequestID(), err)
			if err != nil {
				glog.Warningf("CreateReplica failure, err: %s, %v", worker.Name, err)
			}
		}
	}
	return nil, nil
}

func (a *SimpleResourceManager) releaseGangWorker(rs *carbonv1.RollingSet, worker *carbonv1.WorkerNode, start time.Time) error {
	UpdateCrdTime(&worker.ObjectMeta, start)
	UpdateVersionTime(&worker.ObjectMeta, worker.Spec.Version, start)
	if nil != rs {
		worker.Spec.OwnerGeneration = rs.Generation
	} else {
		worker.Spec.OwnerGeneration = time.Now().Unix()
	}
	worker.Spec.ToDelete = true
	glog.Infof("release %s: to version %s,%s,%s,%s", worker.GetQualifiedName(), worker.Spec.UserDefVersion, worker.Spec.Version, worker.Spec.ShardGroupVersion, worker.Spec.WorkerMode)
	_, err := a.carbonclientset.CarbonV1().WorkerNodes(worker.Namespace).Update(context.Background(), worker, metav1.UpdateOptions{})
	RecordAPICall(CallTypeReleaseReplica, start, &worker.ObjectMeta, common.GetRequestID(), err)
	if err != nil {
		glog.Warningf("Release replica failure, err: %s, %v", worker.Name, err)
		if carbonv1.IsGangMainPart(worker.Spec.Template) {
			return err
		}
	}
	return nil
}

func (a *SimpleResourceManager) releaseGangReplica(rs *carbonv1.RollingSet, replica *carbonv1.Replica, start time.Time) (*carbonv1.Replica, error) {
	// 节点删除顺序，后删除mainpart
	var mainWorker *carbonv1.WorkerNode
	for i := range replica.Gang {
		worker := &replica.Gang[i].WorkerNode
		if carbonv1.IsGangMainPart(worker.Spec.Template) {
			mainWorker = worker
			continue
		}
	}
	if mainWorker != nil {
		err := a.releaseGangWorker(rs, mainWorker, start)
		if err != nil {
			return nil, err
		}
	}
	for i := range replica.Gang {
		worker := &replica.Gang[i].WorkerNode
		mainWorker = worker
		if carbonv1.IsGangMainPart(worker.Spec.Template) {
			continue
		}
		err := a.releaseGangWorker(rs, worker, start)
		if err != nil {
			glog.Warningf("Release replica failure, err: %s, %v", worker.Name, err)
			if carbonv1.IsGangMainPart(worker.Spec.Template) {
				return nil, err
			}
		}
	}
	return nil, nil
}

func (a *SimpleResourceManager) listPodForSS(selector map[string]string) ([]*corev1.Pod, error) {
	ownerKey := carbonv1.DefaultSlotsetUniqueLabelKey
	return a.listPodByOwner(selector, ownerKey)
}

func (a *SimpleResourceManager) listPodByOwner(selector map[string]string, ownerKey string) ([]*corev1.Pod, error) {
	v, ok := selector[ownerKey]
	if !ok {
		return nil, fmt.Errorf("not found owner label %s", ownerKey)
	}
	items, err := a.podIndexer.Index(ownerKey, v)
	if err != nil {
		glog.Errorf("Index workernodes failed: %v", err)
		return nil, err
	}
	pods := make([]*corev1.Pod, 0, len(items))
	for _, i := range items {
		if pod, ok := i.(*corev1.Pod); ok {
			pods = append(pods, pod)
		} else {
			glog.Errorf("not pods %s", utils.ObjJSON(pod))
		}
	}
	return pods, nil
}

// BatchCreateReplica 批量创建replica
func (a *SimpleResourceManager) BatchCreateReplica(rs *carbonv1.RollingSet, rList []*carbonv1.Replica) (int, error) {
	f := func(replica *carbonv1.Replica) error {
		_, err := a.CreateReplica(rs, replica)
		return err
	}
	if a.expectations != nil {
		a.expectations.ExpectCreations(RollingSetKey(rs), len(rList))
	}
	successCount, err := a.batchReplicaOperate(f, rList)
	return successCount, err
}

func (a *SimpleResourceManager) batchReplicaOperate(f interface{}, rList []*carbonv1.Replica) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), common.GetBatcherTimeout(BatchReplicaTimeOut))
	defer cancel()

	batcher := utils.NewBatcherWithExecutor(a.executor)
	commands := make([]*utils.Command, 0, len(rList))
	for _, replica := range rList {
		if replica == nil {
			continue
		}
		command, err := batcher.Go(ctx, false, f, replica)
		if err != nil {
			glog.Warning("batcher.Go err: ", err)
		}
		commands = append(commands, command)
	}
	batcher.Wait()
	errs := utils.NewErrors()
	successCount := 0 //成功执行个数
	for _, command := range commands {
		if nil != command && nil != command.FuncError {
			if nil != command.FuncError {
				errs.Add(command.FuncError)
			} else {
				successCount++
			}
		}
	}
	return successCount, errs.Error()
}

// BatchUpdateReplica 批量更新replica
func (a *SimpleResourceManager) BatchUpdateReplica(rs *carbonv1.RollingSet, rList []*carbonv1.Replica) (int, error) {
	f := func(replica *carbonv1.Replica) error {
		return a.UpdateReplica(rs, replica)
	}
	success, err := a.batchReplicaOperate(f, rList)
	return success, err
}

// BatchReleaseReplica 批量释放replica
func (a *SimpleResourceManager) BatchReleaseReplica(rs *carbonv1.RollingSet, rList []*carbonv1.Replica) (int, error) {
	f := func(replica *carbonv1.Replica) error {
		return a.ReleaseReplica(rs, replica)
	}
	success, err := a.batchReplicaOperate(f, rList)
	return success, err
}

// BatchRemoveReplica 批量删除replica
func (a *SimpleResourceManager) BatchRemoveReplica(rs *carbonv1.RollingSet, rList []*carbonv1.Replica) (int, error) {
	f := func(replica *carbonv1.Replica) error {
		return a.RemoveReplica(rs, replica)
	}
	success, err := a.batchReplicaOperate(f, rList)
	return success, err
}

// CreateRollingSet  is used by shardGroup to crate rollingSet
func (a *SimpleResourceManager) CreateRollingSet(sg *carbonv1.ShardGroup, rs *carbonv1.RollingSet) (*carbonv1.RollingSet, error) {
	start := time.Now()
	UpdateCrdTime(&rs.ObjectMeta, start)
	createRs, err := a.carbonclientset.CarbonV1().RollingSets(rs.Namespace).Create(context.Background(), rs, metav1.CreateOptions{})
	RecordAPICall(CallTypeCreateRollingSet, start, &rs.ObjectMeta, common.GetRequestID(), err)
	if err != nil {
		switch {
		// We may end up hitting this due to a slow cache or a fast resync of the ShardGroup.
		case errors.IsAlreadyExists(err):
			rollingSet, rErr := a.carbonclientset.CarbonV1().RollingSets(rs.Namespace).Get(context.Background(), rs.Name, metav1.GetOptions{})
			if rErr != nil {
				glog.Warningf("CreateReplica failure, err: %v", err)
				return nil, rErr
			}
			controllerRef := metav1.GetControllerOf(rollingSet)
			//已存在， 返回
			if controllerRef != nil && (sg == nil || controllerRef.UID == sg.UID) {
				createRs = rollingSet
				err = nil
				break
			}
			return nil, err
		case err != nil:
			return nil, err
		}
	}
	return createRs, err
}

// UpdateShardGroup is used by shardGroup to update spec
func (a *SimpleResourceManager) UpdateShardGroup(sg *carbonv1.ShardGroup) error {
	start := time.Now()
	UpdateCrdTime(&sg.ObjectMeta, start)
	sg.Spec.RollingVersion = start.Format(time.RFC3339Nano)
	_, err := a.carbonclientset.CarbonV1().ShardGroups(sg.Namespace).Update(context.Background(), sg, metav1.UpdateOptions{})
	RecordAPICall(CallTypeUpdateShardGroup, start, &sg.ObjectMeta, common.GetRequestID(), err)
	if err != nil {
		return err
	}
	return err
}

// UpdateShardGroupStatus  is uesd by shardGroupController to update status
func (a *SimpleResourceManager) UpdateShardGroupStatus(sg *carbonv1.ShardGroup) error {
	start := time.Now()
	var err error
	sg.Status.LastUpdateStatusTime = metav1.Now()
	_, err = a.carbonclientset.CarbonV1().ShardGroups(sg.Namespace).UpdateStatus(context.Background(), sg, metav1.UpdateOptions{})
	RecordAPICall(CallTypeUpdateShardGroupStatus, start, &sg.ObjectMeta, common.GetRequestID(), err)
	if err != nil {
		return err
	}
	return err
}

// DeleteShardGroup is to delete a shardGroup
func (a *SimpleResourceManager) DeleteShardGroup(sg *carbonv1.ShardGroup) error {
	if err := a.carbonclientset.CarbonV1().ShardGroups(sg.Namespace).Delete(context.Background(), sg.Name, metav1.DeleteOptions{}); err != nil {
		return err
	}
	return nil
}

// UpdatePodStatus update pod status only for memkube pod
func (a *SimpleResourceManager) UpdatePodStatus(pod *corev1.Pod) error {
	start := time.Now()

	var err error
	_, err = a.kubeclientset.CoreV1().Pods(pod.GetNamespace()).UpdateStatus(context.Background(), pod, metav1.UpdateOptions{})
	RecordAPICall(CallTypeUpdatePod, start, &pod.ObjectMeta, common.GetRequestID(), err)
	return err
}

func (a *SimpleResourceManager) startBatchWorkerNodes(
	ctx context.Context,
	batcher *utils.Batcher,
	f interface{},
	wList []*carbonv1.WorkerNode,
) []*utils.Command {
	commands := make([]*utils.Command, 0, len(wList))
	for i := range wList {
		if wList[i] == nil {
			continue
		}
		command, err := batcher.Go(ctx, false, f, wList[i])
		if err != nil {
			glog.Warning("batcher.Go err: ", err)
			continue
		}
		commands = append(commands, command)
	}
	return commands
}

func (a *SimpleResourceManager) collectBatcherCommandResult(
	commands []*utils.Command,
) (int, error) {
	errs := utils.NewErrors()
	successCount := 0
	for i := range commands {
		if nil == commands[i] {
			continue
		}
		if nil != commands[i].FuncError {
			errs.Add(commands[i].FuncError)
		} else {
			successCount++
		}
	}
	return successCount, errs.Error()
}

// BatchDoWorkerNodes BatchDoWorkerNodes bacther复用 for memkube
func (a *SimpleResourceManager) BatchDoWorkerNodes(
	workersToCreate []*carbonv1.WorkerNode,
	workersToUpdate []*carbonv1.WorkerNode,
	workersToRelease []*carbonv1.WorkerNode,
) (int, error, int, error, int, error) {
	fCreate := func(workerNode *carbonv1.WorkerNode) error {
		_, err := a.CreateWorkerNode(workerNode)
		return err
	}

	fUpdate := func(workerNode *carbonv1.WorkerNode) error {
		return a.UpdateWorkerNode(workerNode)
	}

	fRelease := func(workerNode *carbonv1.WorkerNode) error {
		return a.ReleaseWorkerNode(workerNode)
	}

	ctx, cancel := context.WithTimeout(context.Background(), common.GetBatcherTimeout(BatchReplicaTimeOut))
	defer cancel()

	batcher := utils.NewBatcher(BatcherPodSize)

	createCommands := a.startBatchWorkerNodes(ctx, batcher, fCreate, workersToCreate)
	updateCommands := a.startBatchWorkerNodes(ctx, batcher, fUpdate, workersToUpdate)
	releaseCommands := a.startBatchWorkerNodes(ctx, batcher, fRelease, workersToRelease)

	batcher.Wait()
	successCreate, errsCreate := a.collectBatcherCommandResult(createCommands)
	successUpdate, errsUpdate := a.collectBatcherCommandResult(updateCommands)
	successRelease, errsRelease := a.collectBatcherCommandResult(releaseCommands)
	return successCreate, errsCreate, successUpdate, errsUpdate, successRelease, errsRelease
}
