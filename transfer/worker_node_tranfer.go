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

package transfer

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	"github.com/alibaba/kube-sharding/pkg/apiserver-client/typespec"
	"github.com/alibaba/kube-sharding/pkg/apiserver-client/typespec/carbon"
	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"

	"github.com/alibaba/kube-sharding/common"
	"github.com/alibaba/kube-sharding/common/features"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Unique Label Keys used to select obj for
const (
	statusConvertTimeout         = 2 * time.Second
	statusConvertConcurrentCount = 200
	schTypeNode                  = "node"
)

var (
	carbonjobKind = carbonv1.SchemeGroupVersion.WithKind("CarbonJob")
)

func aggWorkerNodesByRole(workerNodes []carbonv1.WorkerNode) map[string][]carbonv1.WorkerNode {
	roleWorkerNodes := map[string][]carbonv1.WorkerNode{}
	for i := range workerNodes {
		_, roleID := GetGroupAndRoleID(&workerNodes[i])
		if roleID == "" {
			continue
		}
		if nodes, exist := roleWorkerNodes[roleID]; exist {
			roleWorkerNodes[roleID] = append(nodes, workerNodes[i])
		} else {
			roleWorkerNodes[roleID] = []carbonv1.WorkerNode{workerNodes[i]}
		}
	}
	return roleWorkerNodes
}

func aggWorkerNodesByGroup(workerNodes []carbonv1.WorkerNode) map[string][]carbonv1.WorkerNode {
	groupWorkerNodes := map[string][]carbonv1.WorkerNode{}
	for i := range workerNodes {
		groupID := carbonv1.GetGroupName(&workerNodes[i])
		if groupID == "" {
			continue
		}
		if nodes, exist := groupWorkerNodes[groupID]; exist {
			groupWorkerNodes[groupID] = append(nodes, workerNodes[i])
		} else {
			groupWorkerNodes[groupID] = []carbonv1.WorkerNode{workerNodes[i]}
		}
	}
	return groupWorkerNodes
}

func getCurrentWorkerNodesOnGeneration(workerNodes []carbonv1.WorkerNode) (*carbonv1.WorkerNode, *carbonv1.WorkerNode) {
	if nil == workerNodes || len(workerNodes) == 0 {
		return nil, nil
	}
	sort.Slice(workerNodes, func(i, j int) bool {
		return workerNodes[i].GetGeneration() > workerNodes[j].GetGeneration()
	})
	if len(workerNodes) > 1 {
		return &workerNodes[0], &workerNodes[1]
	}
	return &workerNodes[0], nil
}

func transWorkerNodesToReplicaNodeStatus(
	roleID string,
	currentWorkerNode *carbonv1.WorkerNode,
	currentPod *corev1.Pod,
	backupWorkerNode *carbonv1.WorkerNode,
	backupPod *corev1.Pod,
) (*carbon.ReplicaNodeStatus, error) {
	currentWorkerStatus, err := TransWorkerStatus(nil, currentWorkerNode, currentPod)
	if currentWorkerStatus.GetCurVersion() == "" {
		currentWorkerStatus.CurVersion = &currentWorkerNode.Spec.Version
	}
	if nil != err {
		err = fmt.Errorf("TransWorkerStatus error %s,%v", currentWorkerNode.GetName(), err)
		return nil, err
	}
	var backupWorkerStatus *carbon.WorkerNodeStatus
	if nil != backupWorkerNode {
		backupWorkerStatus, err = TransWorkerStatus(nil, backupWorkerNode, backupPod)
		if nil != err {
			err = fmt.Errorf("TransWorkerStatus error %s,%v", currentWorkerNode.GetName(), err)
			return nil, err
		}
	}
	var status = &carbon.ReplicaNodeStatus{
		ReplicaNodeId:          &roleID,
		CurWorkerNodeStatus:    currentWorkerStatus,
		BackupWorkerNodeStatus: backupWorkerStatus,
		TimeStamp:              utils.Int64Ptr(currentWorkerNode.Status.LastUpdateStatusTime.UnixNano() / 1000),
		UserDefVersion:         &currentWorkerNode.Status.UserDefVersion,
		ReadyForCurVersion:     &currentWorkerNode.Status.Complete,
	}
	return status, nil
}

func transWorkerNodesToRoleStatus(roleID string, workerNodes []carbonv1.WorkerNode, podSet map[string]*corev1.Pod,
) (*carbon.RoleStatusValue, error) {
	currentWorkerNode, backupWorkerNode := getCurrentWorkerNodesOnGeneration(workerNodes)
	if nil == currentWorkerNode {
		return nil, nil
	}
	var currentPod, backupPod *corev1.Pod
	currentPodName, _ := carbonv1.GetPodName(currentWorkerNode)
	currentPod = podSet[currentPodName]
	if nil != backupWorkerNode {
		backupPodName, _ := carbonv1.GetPodName(backupWorkerNode)
		backupPod = podSet[backupPodName]
	}
	replicaNodeStatus, err := transWorkerNodesToReplicaNodeStatus(
		roleID, currentWorkerNode, currentPod, backupWorkerNode, backupPod)
	if nil != err {
		return nil, err
	}

	versionedPlans := getVersionedPlan(currentWorkerNode.Spec.Template, currentWorkerNode.Spec.Version, carbonv1.GetQuotaID(currentWorkerNode))
	roleStatus := carbon.RoleStatusValue{
		RoleId: &roleID,
		GlobalPlan: &carbon.GlobalPlan{
			Count: utils.Int32Ptr(1),
		},
		Nodes:              []*carbon.ReplicaNodeStatus{replicaNodeStatus},
		LatestVersion:      &currentWorkerNode.Spec.Version,
		UserDefVersion:     &currentWorkerNode.Spec.UserDefVersion,
		ReadyForCurVersion: &currentWorkerNode.Status.Complete,
		VersionedPlans:     versionedPlans,
	}
	return &roleStatus, nil
}

func transWorkerNodesToGroupStatus(workerNodes []carbonv1.WorkerNode, podSet map[string]*corev1.Pod) (*carbon.GroupStatus, error) {
	if nil == workerNodes || len(workerNodes) == 0 {
		return nil, nil
	}
	groupID := carbonv1.GetGroupName(&workerNodes[0])
	groupStatus := carbon.GroupStatus{
		GroupId: &groupID,
		Roles:   map[string]*carbon.RoleStatusValue{},
	}
	// Group下按role聚合.
	roleWorkerNodes := aggWorkerNodesByRole(workerNodes)
	for roleID, workerNodes := range roleWorkerNodes {
		// 每个role下的workerNode聚合成role的Status
		roleStatus, err := transWorkerNodesToRoleStatus(roleID, workerNodes, podSet)
		if nil != err {
			return nil, err
		}
		if nil != roleStatus {
			groupStatus.Roles[roleID] = roleStatus
		}
	}
	return &groupStatus, nil
}

// TransWorkerNodesToGroupStatusList ...用于生成Status
func TransWorkerNodesToGroupStatusList(appName string, workerNodes []carbonv1.WorkerNode, pods []corev1.Pod) ([]*carbon.GroupStatus, error) {
	if nil == workerNodes || len(workerNodes) == 0 {
		return nil, nil
	}
	podSet := map[string]*corev1.Pod{}
	for i := range pods {
		podSet[pods[i].GetName()] = &pods[i]
	}

	ctx, cancel := context.WithTimeout(context.Background(), statusConvertTimeout)
	defer cancel()

	// 按Group聚合
	groupWorkerNodes := aggWorkerNodesByGroup(workerNodes)

	batcher := utils.NewBatcher(statusConvertConcurrentCount)
	commands := make([]*utils.Command, 0, len(groupWorkerNodes))
	for _, workerNodes := range groupWorkerNodes {
		command, err := batcher.Go(ctx, false, transWorkerNodesToGroupStatus, workerNodes, podSet)
		if err != nil {
			return nil, fmt.Errorf("statusConverter batcher go failed, err: %v", err)
		}
		commands = append(commands, command)
	}

	batcher.Wait()

	groupStatusList := []*carbon.GroupStatus{}
	for i := range commands {
		if nil == commands[i] {
			continue
		}
		if nil != commands[i].FuncError {
			return nil, commands[i].FuncError
		}
		if len(commands[i].Results) != 2 {
			return nil, fmt.Errorf("statusConverter batcher result len err")
		}
		if groupStatus, ok := commands[i].Results[0].(*carbon.GroupStatus); ok {
			if nil != groupStatus {
				groupStatusList = append(groupStatusList, groupStatus)
			}
			continue
		} else {
			return nil, fmt.Errorf("statusConverter batcher result type err, %v", commands[i].Results)
		}
	}
	return groupStatusList, nil
}

// WorkerNodeConvertResult WorkerNodeConvertResult
type WorkerNodeConvertResult struct {
	WorkerNode *carbonv1.WorkerNode
	Immutable  bool
}

func transRollingsetToWorkerNode(carbonJob *carbonv1.CarbonJob, rollingset *carbonv1.RollingSet) (*carbonv1.WorkerNode, error) {
	version, resVersion, err := carbonv1.SignVersionPlan(
		&rollingset.Spec.VersionPlan, rollingset.Labels, rollingset.Name)
	if nil != err {
		return nil, fmt.Errorf("transRollingsetToWorkerNode failed, SignVersionPlan failed, err: %v", err)
	}

	worker := carbonv1.WorkerNode{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   carbonJob.GetNamespace(),
			Name:        fmt.Sprintf("%s-a", rollingset.Name),
			Labels:      rollingset.Labels,
			Annotations: rollingset.Annotations,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         carbonjobKind.Version,
					Kind:               carbonjobKind.Kind,
					Name:               carbonJob.Name,
					UID:                carbonJob.UID,
					Controller:         utils.BoolPtr(true),
					BlockOwnerDeletion: utils.BoolPtr(true),
				},
			},
		},
		Spec: carbonv1.WorkerNodeSpec{
			ResVersion:  resVersion,
			Version:     version,
			VersionPlan: rollingset.Spec.VersionPlan,
		},
	}
	exclusiveMode := carbonv1.GetExclusiveMode(carbonJob)
	carbonv1.InitWorkerScheduleTimeout(&worker.Spec.WorkerSchedulePlan)
	worker.Labels[carbonv1.GroupTypeKey] = carbonv1.GroupTypeReplica
	worker.Labels[carbonv1.LabelKeyExclusiveMode] = exclusiveMode
	return &worker, nil
}

func transRolePlanToWorkerNode(carbonJob *carbonv1.CarbonJob, hippoCluster, appName, groupID, roleID string, rolePlan *typespec.RolePlan,
) (*carbonv1.WorkerNode, bool, error) {
	shardgroupName := SubGroupName(groupID)
	transSpecResult, err := TransRoleToRollingsets(hippoCluster, appName, "", groupID, roleID, rolePlan, schTypeNode, true, shardgroupName, "", false)
	if err == nil { // to append group name, see transRolePlanToWorkerNode
		carbonv1.SetAnnoGroupName(transSpecResult.Rollingset, groupID)
	}
	if nil != err || nil == transSpecResult || nil == transSpecResult.Rollingset {
		return nil, false, fmt.Errorf("transRolePlanToWorkerNode failed, trans %s %s %s %s, rolePlan: %s to rollingset err: %v",
			hippoCluster, appName, groupID, roleID, utils.ObjJSON(rolePlan), err)
	}

	worker, err := transRollingsetToWorkerNode(carbonJob, transSpecResult.Rollingset)
	if nil != err {
		return nil, false, fmt.Errorf("transRolePlanToWorkerNode failed, trans rollingset err: %v", err)
	}
	worker.Labels[carbonv1.LabelKeyClusterName] = hippoCluster
	cjName := carbonJob.Labels[carbonv1.LabelKeyCarbonJobName]
	worker.Labels[carbonv1.LabelKeyCarbonJobName] = cjName
	if carbonv1.GetQuotaID(worker) == "" && carbonv1.GetQuotaID(carbonJob) != "" {
		worker.Labels[carbonv1.LabelKeyQuotaGroupID] = carbonv1.GetQuotaID(carbonJob)
	}
	// api request from masterframework, some hack for compatible
	masterFrameworkMode := IsMasterFrameworkMode(rolePlan)
	return worker, masterFrameworkMode && features.C2MutableFeatureGate.Enabled(features.ImmutableWorker), nil
}

// ConvertWorkerFromGroupPlan ConvertWorkerFromGroupPlan
func ConvertWorkerFromGroupPlan(carbonJob *carbonv1.CarbonJob, hippoCluster, appName, groupID string, groupPlan *typespec.GroupPlan) (map[string]*WorkerNodeConvertResult, error) {
	newWorkers := make(map[string]*WorkerNodeConvertResult, len(groupPlan.RolePlans))
	for roleID, rolePlan := range groupPlan.RolePlans {
		worker, immutableWorker, err := transRolePlanToWorkerNode(
			carbonJob, hippoCluster, appName, groupID, roleID, rolePlan)
		if nil != err {
			// if gen worker failed, worker should be delete
			glog.Errorf("convertWorkerFromGroupPlan failed, trans %s %s %s %s rolePlan err: %v", hippoCluster, appName, groupID, roleID, err)
			continue
		}
		if nil == worker {
			continue
		}
		newWorkers[carbonv1.GetWorkerReplicaName(worker)] = &WorkerNodeConvertResult{
			WorkerNode: worker,
			Immutable:  immutableWorker,
		}
	}
	return newWorkers, nil
}

const (
	specConvertTimeout         = 30 * time.Second
	specConvertConcurrentCount = 2000
)

// workerNodeSpecConverter 用于将GroupPlan转换为workerNodeSpec
type WorkerNodeSpecConverter interface {
	Convert(*carbonv1.CarbonJob) (map[string]*WorkerNodeConvertResult, error)
}

type DefaultWorkerNodeSpecConverter struct{}

func NewWorkerNodeSpecConverter() WorkerNodeSpecConverter {
	return &DefaultWorkerNodeSpecConverter{}
}

func (c *DefaultWorkerNodeSpecConverter) Convert(carbonJob *carbonv1.CarbonJob) (map[string]*WorkerNodeConvertResult, error) {
	hippoCluster := carbonJob.Labels[carbonv1.LabelKeyClusterName]
	appName := carbonJob.Spec.AppName
	ctx, cancel := context.WithTimeout(context.Background(), common.GetBatcherTimeout(specConvertTimeout))
	defer cancel()
	batcher := utils.NewBatcher(specConvertConcurrentCount)
	commands := make([]*utils.Command, 0, len(carbonJob.Spec.AppPlan.Groups))
	for groupID, groupPlan := range carbonJob.Spec.AppPlan.Groups {
		if nil == groupPlan {
			continue
		}
		command, err := batcher.Go(ctx, false, ConvertWorkerFromGroupPlan, carbonJob, hippoCluster, appName, groupID, groupPlan)
		if err != nil {
			return nil, fmt.Errorf("specConverter batcher go failed, err: %v", err)
		}
		commands = append(commands, command)
	}
	batcher.Wait()

	newWorkers := make(map[string]*WorkerNodeConvertResult)
	for i := range commands {
		if nil == commands[i] {
			continue
		}
		if nil != commands[i].FuncError {
			return nil, commands[i].FuncError
		}
		if len(commands[i].Results) != 2 {
			return nil, fmt.Errorf("specConverter batcher result len err")
		}
		if workers, ok := commands[i].Results[0].(map[string]*WorkerNodeConvertResult); ok {
			for name, worker := range workers {
				newWorkers[name] = worker
			}
			continue
		} else {
			return nil, fmt.Errorf("specConverter batcher result type err, %v", workers)
		}
	}
	return newWorkers, nil
}
