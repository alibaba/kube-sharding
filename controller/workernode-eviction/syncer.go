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
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/alibaba/kube-sharding/common"
	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

const (
	defaultPrefTTL = 600
	batchTimeout   = 2 * time.Second
	batchSize      = 50
)

func (c *Controller) genEvictLabel(pref *carbonv1.HippoPrefrence) string {
	if nil == pref || nil == pref.Scope || nil == pref.Type {
		return ""
	}
	if nil == pref.TTL {
		pref.TTL = utils.Int32Ptr(defaultPrefTTL)
	}
	return fmt.Sprintf("%s-%s-%d", *pref.Scope, *pref.Type, *pref.TTL)
}

func (c *Controller) getSlotIDFromPod(pod *corev1.Pod) (string, int32, error) {
	if nil == pod {
		return "", 0, nil
	}
	var slaveAddress string
	var slotID int32
	if carbonv1.IsAsiPod(&pod.ObjectMeta) {
		slaveAddress = fmt.Sprintf("%s:%s", pod.Status.HostIP, "7007")
		slotIDStr := pod.Annotations[carbonv1.AnnotationKeySlotID]
		slotIDV, err := strconv.Atoi(slotIDStr)
		if nil != err {
			glog.Errorf("parse slot id error %s, %s, %v", pod.Name, slotIDStr, err)
			return slaveAddress, 0, err
		}
		slotID = int32(slotIDV)
		return slaveAddress, slotID, nil
	}
	var scheduleStatus = carbonv1.ScheduleStatusExtend{}
	for i := range pod.Status.Conditions {
		if string(pod.Status.Conditions[i].Type) == carbonv1.ConditionKeyScheduleStatusExtend {
			err := json.Unmarshal([]byte(pod.Status.Conditions[i].Message), &scheduleStatus)
			if nil != err {
				return "", 0, err
			}
			slotID = int32(scheduleStatus.SlotID.SlotID)
			slaveAddress = scheduleStatus.SlotID.SlaveAddress
			break
		}
	}
	return slaveAddress, slotID, nil
}

func (c *Controller) workerNodeReclaimed(
	namespace, appName string,
	reclaimWorkerNode *carbonv1.ReclaimWorkerNode,
	prefLabel string,
) (bool, *carbonv1.ReclaimWorkerNode, error) {
	if nil == reclaimWorkerNode {
		return true, reclaimWorkerNode, nil
	}
	if nil == reclaimWorkerNode.SlotID.SlaveAddress || nil == reclaimWorkerNode.SlotID.Id {
		return true, reclaimWorkerNode, nil
	}
	if "" == *reclaimWorkerNode.SlotID.SlaveAddress || -1 == *reclaimWorkerNode.SlotID.Id {
		return true, reclaimWorkerNode, nil
	}
	if reclaimWorkerNode.WorkerNodeID == "" && *reclaimWorkerNode.SlotID.Id != 0 {
		workernodeID, err := c.getWorkernodeID(namespace, appName, reclaimWorkerNode)
		if err != nil {
			return false, reclaimWorkerNode, err
		}
		if workernodeID == "" {
			return true, reclaimWorkerNode, nil
		}
		reclaimWorkerNode.WorkerNodeID = workernodeID
	}

	worker, err := c.workerNodeLister.WorkerNodes(namespace).Get(reclaimWorkerNode.WorkerNodeID)
	if nil != err {
		if errors.IsNotFound(err) {
			return true, reclaimWorkerNode, nil
		}
		return false, reclaimWorkerNode, err
	}
	if nil != worker && worker.Spec.Reclaim {
		return true, reclaimWorkerNode, nil
	}

	// 判断slotid, slaveAddress是否匹配
	podName, _ := carbonv1.GetPodName(worker)
	pod, err := c.podLister.Pods(namespace).Get(podName)
	if nil != err {
		if errors.IsNotFound(err) {
			return true, reclaimWorkerNode, nil
		}
		return false, reclaimWorkerNode, err
	}
	currentSlaveAddress, currentSlotID, err := c.getSlotIDFromPod(pod)
	if err == nil {
		if currentSlaveAddress != *reclaimWorkerNode.SlotID.SlaveAddress ||
			currentSlotID != *reclaimWorkerNode.SlotID.Id {
			return true, reclaimWorkerNode, nil
		}
	} else {
		if (currentSlaveAddress != "" && currentSlaveAddress != *reclaimWorkerNode.SlotID.SlaveAddress) ||
			*reclaimWorkerNode.SlotID.Id != 0 || time.Now().Sub(pod.CreationTimestamp.Time) < time.Minute*5 {
			return false, reclaimWorkerNode, err
		}
	}

	//直接release，下老启新
	workerCopy := worker.DeepCopy()
	workerCopy.Spec.Releasing = true
	if "" != prefLabel {
		workerCopy.Labels[carbonv1.LabelKeyHippoPreference] = prefLabel
	}
	glog.Infof("Update worker node to release: %s, %s", workerCopy.Name, prefLabel)
	if err := c.ResourceManager.UpdateWorkerNode(workerCopy); nil != err {
		return false, reclaimWorkerNode, err
	}
	return true, reclaimWorkerNode, nil
}

func (c *Controller) getWorkernodeID(namespace, appName string,
	reclaimWorkerNode *carbonv1.ReclaimWorkerNode) (string, error) {
	labelSelector := metav1.SetAsLabelSelector(labels.Set{})
	carbonv1.SetAppSelector(labelSelector, appName)
	selector, err := metav1.LabelSelectorAsSelector(labelSelector)
	if err != nil {
		return "", err
	}
	workernodes, err := c.workerNodeLister.WorkerNodes(namespace).List(selector)
	if err != nil {
		return "", err
	}
	for _, workernode := range workernodes {
		if reclaimWorkerNode.SlotID.Id != nil && int32(workernode.Status.SlotID.SlotID) == *reclaimWorkerNode.SlotID.Id &&
			reclaimWorkerNode.SlotID.SlaveAddress != nil && workernode.Status.SlotID.SlaveAddress == *reclaimWorkerNode.SlotID.SlaveAddress {
			glog.Infof("GetWorkernodeID workernodeID:%s from %v workernodes", workernode.Name, len(workernodes))
			return workernode.Name, nil
		}
	}
	return "", nil
}

func (c *Controller) doSyncWorkerNodesPref(evict *carbonv1.WorkerNodeEviction) ([]*carbonv1.ReclaimWorkerNode, error) {
	prefLabel := c.genEvictLabel(evict.Spec.Pref)

	ctx, cancel := context.WithTimeout(context.Background(), common.GetBatcherTimeout(batchTimeout))
	defer cancel()

	batcher := utils.NewBatcher(batchSize)
	commands := make([]*utils.Command, 0, len(evict.Spec.WorkerNodes))
	remainWorkers := []*carbonv1.ReclaimWorkerNode{}
	for i := range evict.Spec.WorkerNodes {
		command, err := batcher.Go(ctx, false, c.workerNodeReclaimed, evict.Namespace, evict.Spec.AppName, evict.Spec.WorkerNodes[i], prefLabel)
		if nil != err {
			glog.Warningf("doSyncWorkerNodesPref %s failed, err: %v",
				utils.ObjJSON(evict.Spec.WorkerNodes[i]), err)
			remainWorkers = append(remainWorkers, evict.Spec.WorkerNodes[i])
			continue
		}
		commands = append(commands, command)
	}

	batcher.Wait()

	for i := range commands {
		if nil == commands[i] {
			continue
		}
		if nil != commands[i].FuncError {
			glog.Warningf("doSyncWorkerNodesPref reclaimed failed, err: %v", commands[i].FuncError)
			continue
		}
		if 3 != len(commands[i].Results) {
			glog.Warningf("doSyncWorkerNodesPref batcher result len err")
			continue
		}
		if reclaimed, ok := commands[i].Results[0].(bool); ok && reclaimed {
			continue
		}
		if worker, ok := commands[i].Results[1].(*carbonv1.ReclaimWorkerNode); ok {
			remainWorkers = append(remainWorkers, worker)
		} else {
			glog.Warningf("doSyncWorkerNodesPref batcher result value err, %v", commands[i].Results...)
		}
	}
	return remainWorkers, nil
}

// Sync sync
func (c *Controller) Sync(key string) error {
	if glog.V(4) {
		glog.Infof("workerNodeEviction controller sync %s", key)
	}
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if nil != err {
		return err
	}

	evict, err := c.evictionLister.WorkerNodeEvictions(namespace).Get(name)
	if nil != err {
		return err
	}

	if 0 == len(evict.Spec.WorkerNodes) {
		if err := c.carbonclientset.CarbonV1().WorkerNodeEvictions(evict.Namespace).Delete(context.Background(), evict.Name, metav1.DeleteOptions{}); nil != err {
			return err
		}
		glog.Infof("clear useless workernode eviction success, %s", utils.ObjJSON(evict))
		return nil
	}

	remainWorkers, err := c.doSyncWorkerNodesPref(evict)
	if nil != err {
		return err
	}
	if glog.V(4) {
		glog.Infof("workerNodeEviction %s after sync remainWorkers: %v", key, remainWorkers)
	}
	// if delete failed, how to process reclaim
	if 0 == len(remainWorkers) {
		if err := c.carbonclientset.CarbonV1().WorkerNodeEvictions(evict.Namespace).Delete(context.Background(), evict.Name, metav1.DeleteOptions{}); nil != err {
			return err
		}
		glog.Infof("clear workernode eviction success, %s", utils.ObjJSON(evict))
		return nil
	}

	evictCopy := evict.DeepCopy()
	evictCopy.Spec.WorkerNodes = remainWorkers
	if _, err := c.carbonclientset.CarbonV1().WorkerNodeEvictions(evict.Namespace).Update(context.Background(), evictCopy, metav1.UpdateOptions{}); nil != err {
		return err
	}
	glog.Infof("update workernode eviction success, %s to %s", utils.ObjJSON(evict), utils.ObjJSON(evictCopy))
	return nil
}
