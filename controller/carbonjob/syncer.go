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
	"context"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/alibaba/kube-sharding/pkg/ago-util/monitor/metricgc"
	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	"github.com/alibaba/kube-sharding/transfer"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

func (c *Controller) getCarbonJobByKey(key string) (*carbonv1.CarbonJob, error) {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if nil != err {
		return nil, fmt.Errorf("getCarbonJobByKey failed, split key: %s, err: %v", key, err)
	}
	cb, err := c.carbonJobLister.CarbonJobs(namespace).Get(name)
	if nil != err {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("getCarbonJobByKey failed, get err: %v", err)
	}
	return cb, nil
}

func (c *Controller) prepareUpdateWorkerNode(
	old, new *carbonv1.WorkerNode,
) *carbonv1.WorkerNode {
	if new.Spec.Version == old.Spec.Version && new.Spec.ResVersion == old.Spec.ResVersion &&
		new.Spec.UserDefVersion == old.Spec.UserDefVersion && new.Spec.RecoverStrategy == old.Spec.RecoverStrategy {
		return nil
	}
	labels := new.Labels
	new.ObjectMeta = *old.ObjectMeta.DeepCopy()
	new.TypeMeta = old.TypeMeta
	for key, value := range new.Labels {
		labels[key] = value
	}
	new.Labels = labels
	new.Spec.Selector = old.Spec.Selector
	// The name of new here is always `xxx-a`, but the actually old name maybe `xxx-b` by worker node recover.
	// So rewrite the real worker name.
	new.Name = old.Name
	if reflect.DeepEqual(old.Spec, new.Spec) && reflect.DeepEqual(old.Labels, new.Labels) {
		return nil
	}
	new.Status = old.Status
	return new
}

func (c *Controller) doSyncWorkerSpec(
	oldWorkers map[string]*carbonv1.WorkerNode,
	newWorkers map[string]*transfer.WorkerNodeConvertResult,
	noExistGroupWorkers []*carbonv1.WorkerNode,
) error {
	workersToRelease := noExistGroupWorkers
	workersToUpdate := make([]*carbonv1.WorkerNode, 0)
	workersToCreate := make([]*carbonv1.WorkerNode, 0)
	for workerName, worker := range newWorkers {
		oldWorker, exist := oldWorkers[workerName]
		if exist {
			if !worker.Immutable {
				w := c.prepareUpdateWorkerNode(oldWorker, worker.WorkerNode)
				if nil != w {
					workersToUpdate = append(workersToUpdate, worker.WorkerNode)
				}
			}
			continue
		}
		workersToCreate = append(workersToCreate, worker.WorkerNode)
	}

	for workerName, oldWorker := range oldWorkers {
		if _, exist := newWorkers[workerName]; !exist {
			workersToRelease = append(workersToRelease, oldWorker)
		}
	}
	releaseNames := make([]string, 0)
	for _, w := range workersToRelease {
		releaseNames = append(releaseNames, w.Name)
	}
	if len(releaseNames) > 0 {
		glog.Infof("Release workerNodes: %s", releaseNames)
	}
	successCreate, errsCreate, successUpdate, errsUpdate, succcessRelease, errsRelease := c.ResourceManager.BatchDoWorkerNodes(
		workersToCreate, workersToUpdate, workersToRelease,
	)
	if nil != errsCreate || nil != errsUpdate || nil != errsRelease {
		glog.Infof(
			"carbonJobController doSyncWorkerSpec failed, create total: %d, success: %d, err: %v; update total: %d, success: %d, err: %v, release total: %d, success: %d, err: %v",
			len(workersToCreate), successCreate, errsCreate, len(workersToUpdate), successUpdate, errsUpdate, len(workersToRelease), succcessRelease, errsRelease,
		)
		errs := utils.NewErrors()
		errs.Add(errsCreate)
		errs.Add(errsUpdate)
		errs.Add(errsRelease)
		return errs.Error()
	}
	glog.Infof(
		"carbonJobController doSyncWorkerSpec success, create total: %d, success: %d; update total: %d, success: %d, release total: %d, success: %d",
		len(workersToCreate), successCreate, len(workersToUpdate), successUpdate, len(workersToRelease), succcessRelease,
	)
	return nil
}

func (c *Controller) syncWorkerSpecFromCarbonJob(carbonJob *carbonv1.CarbonJob, workers []*carbonv1.WorkerNode) error {
	// 考虑所有的workerNode都在一个namespace下, 不区分Group处理了.
	workerMap := map[string]*carbonv1.WorkerNode{}
	noExistGroupWorkers := make([]*carbonv1.WorkerNode, 0)
	var newWorkerMap = map[string]*transfer.WorkerNodeConvertResult{}
	var err error
	if carbonJob.Spec.AppPlan != nil && carbonJob.Spec.AppPlan.Groups != nil {
		newWorkerMap, err = c.specConverter.Convert(carbonJob)
		if nil != err {
			return fmt.Errorf("syncWorkerSpecFromCarbonJob failed, convert spec err: %v", err)
		}
	}

	if carbonJob.Spec.JobPlan != nil && carbonJob.Spec.JobPlan.WorkerNodes != nil {
		for replicaName := range carbonJob.Spec.JobPlan.WorkerNodes {
			template := carbonJob.Spec.JobPlan.WorkerNodes[replicaName]
			worker := transfer.WorkerNodeConvertResult{
				WorkerNode: &carbonv1.WorkerNode{
					ObjectMeta: *template.ObjectMeta.DeepCopy(),
					Spec:       template.Spec,
				},
				Immutable: template.Immutable,
			}
			newWorkerMap[replicaName] = &worker
		}
	}
	for _, worker := range newWorkerMap {
		if len(worker.WorkerNode.OwnerReferences) != 0 {
			for i := range worker.WorkerNode.OwnerReferences {
				if worker.WorkerNode.OwnerReferences[i].Kind == controllerKind.Kind && worker.WorkerNode.OwnerReferences[i].Name == carbonJob.Name {
					worker.WorkerNode.OwnerReferences[i].UID = carbonJob.UID
					glog.V(4).Infof("%s, %s complete ownerReferences UID %s", worker.WorkerNode.Name, carbonJob.Name, carbonJob.UID)
				} else {
					glog.V(4).Infof("%s not complete ownerReferences %s,%s,%s,%s", worker.WorkerNode.Name, worker.WorkerNode.OwnerReferences[i].Kind, controllerKind.Kind, worker.WorkerNode.OwnerReferences[i].Name, carbonJob.Name)
				}
			}
		}
	}

	for i := range workers {
		// 去掉不存在的group对应的workerNode
		if _, exist := newWorkerMap[carbonv1.GetWorkerReplicaName(workers[i])]; !exist {
			noExistGroupWorkers = append(noExistGroupWorkers, workers[i])
			continue
		}
		workerMap[carbonv1.GetWorkerReplicaName(workers[i])] = workers[i]
	}

	if err := c.doSyncWorkerSpec(workerMap, newWorkerMap, noExistGroupWorkers); nil != err {
		return fmt.Errorf("syncWorkerSpecFromCarbonJob failed, sync spec err: %s", err)
	}
	return nil
}

func (c *Controller) getWorkerNodesFromCarbonJob(carbonJob *carbonv1.CarbonJob) ([]*carbonv1.WorkerNode, error) {
	workers, err := c.ResourceManager.ListWorkerNodeByOwner(carbonJob.GetLabels(), carbonv1.LabelKeyCarbonJobName)
	if nil != err {
		return nil, fmt.Errorf("list workernodes for carbonjob failed: %s, %v", carbonJob.Name, err)
	}
	return workers, nil
}

func (c *Controller) syncWorkerStatusToCarbonJobStatus(carbonJob *carbonv1.CarbonJob, workers []*carbonv1.WorkerNode) error {
	var successCount, deadCount int
	for i := range workers {
		if nil == workers[i] {
			continue
		}
		if carbonv1.IsWorkerReady(workers[i]) {
			successCount++
			continue
		}
		if carbonv1.IsWorkerDead(workers[i]) {
			deadCount++
			continue
		}
	}
	status := carbonv1.CarbonJobStatus{
		SuccessCount: successCount,
		DeadCount:    deadCount,
		TotalCount:   len(workers),
		TargetCount:  carbonJob.GetTaskSize(),
	}

	if !reflect.DeepEqual(carbonJob.Status, status) {
		carbonJobCopy := carbonJob.DeepCopy()
		carbonJobCopy.Status = status
		if _, err := c.carbonclientset.CarbonV1().CarbonJobs(carbonJobCopy.Namespace).UpdateStatus(context.Background(), carbonJobCopy, metav1.UpdateOptions{}); nil != err {
			glog.Errorf("update carbonJobStatus %s failed, err: %v", carbonJob.Spec.AppName, err)
			return err
		}
	}
	return nil
}

func (c *Controller) addFinalizer(carbonJob *carbonv1.CarbonJob) (*carbonv1.CarbonJob, error) {
	carbonJob.Finalizers = []string{carbonv1.FinalizerSmoothDeletion}
	if _, err := c.carbonclientset.CarbonV1().CarbonJobs(carbonJob.Namespace).Update(context.Background(), carbonJob, metav1.UpdateOptions{}); nil != err {
		return nil, err
	}
	return c.carbonJobLister.CarbonJobs(carbonJob.Namespace).Get(carbonJob.Name)
}

// Sync sync
func (c *Controller) Sync(key string) error {
	carbonJob, err := c.getCarbonJobByKey(key)
	if nil != err {
		return err
	}
	if nil == carbonJob {
		return nil
	}

	if len(carbonJob.Finalizers) == 0 {
		carbonJob, err = c.addFinalizer(carbonJob)
		if nil != err {
			return err
		}
	}

	if nil != carbonJob.DeletionTimestamp {
		return c.doDeleteCarbonJob(carbonJob)
	}

	startTime := time.Now()
	currentWorkers, err := c.getWorkerNodesFromCarbonJob(carbonJob)
	if nil != err {
		return err
	}

	glog.V(4).Infof("Found %d currentWorkers", len(currentWorkers))
	startTime = time.Now()
	if err := c.syncWorkerSpecFromCarbonJob(carbonJob, currentWorkers); nil != err {
		glog.Warningf("carbonJobController syncWorkerSpecFromCarbonJob failed, cost: %s, err: %v", time.Now().Sub(startTime), err)
		return err
	}
	metricgc.WithLabelValuesSummary(SyncWorkerSpecFromCarbonJobLatency,
		carbonJob.Namespace, carbonJob.Name, strconv.FormatBool(err == nil),
	).Observe(float64(time.Since(startTime).Nanoseconds() / 1000000))
	return nil
}

func (c *Controller) doDeleteCarbonJob(carbonJob *carbonv1.CarbonJob) error {
	if nil == carbonJob {
		return nil
	}

	// clear all sub resource
	workers, err := c.getWorkerNodesFromCarbonJob(carbonJob)
	if nil != err && !errors.IsNotFound(err) {
		return err
	}

	// workers all released, release carbonJob
	if len(workers) == 0 || errors.IsNotFound(err) {
		carbonJobCopy := carbonJob.DeepCopy()
		carbonJobCopy.Finalizers = nil

		if _, err := c.carbonclientset.CarbonV1().CarbonJobs(carbonJobCopy.Namespace).Update(context.Background(), carbonJobCopy, metav1.UpdateOptions{}); nil != err {
			return err
		}
		glog.Infof("caronJobController release carbonJob %s", utils.ObjJSON(carbonJobCopy))

		if err = c.carbonclientset.CarbonV1().CarbonJobs(carbonJobCopy.Namespace).Delete(context.Background(), carbonJobCopy.Name, metav1.DeleteOptions{}); nil != err && !errors.IsNotFound(err) {
			return err
		}
		glog.Infof("caronJobController delete carbonJob %s", utils.ObjJSON(carbonJobCopy))
		return nil
	}

	_, _, _, _, success, err := c.ResourceManager.BatchDoWorkerNodes(
		[]*carbonv1.WorkerNode{},
		[]*carbonv1.WorkerNode{},
		workers,
	)
	if nil != err {
		glog.Errorf("carbonJobController batch release worker node failed, err: %s", err)
		return err
	}
	glog.Infof("caronJobController batch release worker node %d success %d", len(workers), success)
	// wait workerNode change trigger resync
	return nil
}
