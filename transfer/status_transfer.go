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
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/alibaba/kube-sharding/common/features"
	"github.com/openkruise/kruise-api/apps/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	podutil "k8s.io/kubernetes/pkg/api/v1/pod"

	"github.com/alibaba/kube-sharding/common"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	"github.com/alibaba/kube-sharding/pkg/apiserver-client/typespec/carbon"
	"github.com/alibaba/kube-sharding/pkg/apiserver-client/typespec/hippo"
	corev1 "k8s.io/api/core/v1"
	intstrutil "k8s.io/apimachinery/pkg/util/intstr"
	glog "k8s.io/klog"
)

// ReplicaNodes ReplicaNodes
type ReplicaNodes struct {
	Worker       *carbonv1.WorkerNode
	Pod          *corev1.Pod
	BackupWorker *carbonv1.WorkerNode
	BackupPod    *corev1.Pod
}

// TransRoleStatus TransRoleStatus
func TransRoleStatus(groupID string, rollingSet *carbonv1.RollingSet, replicas []ReplicaNodes) (*carbon.RoleStatusValue, error) {
	var nstatus = make([]*carbon.ReplicaNodeStatus, len(replicas))
	var err error
	var allReplicasReady = true
	for i := range replicas {
		nstatus[i], err = TransReplicaStatus(rollingSet, replicas[i].Worker, replicas[i].Pod, replicas[i].BackupWorker, replicas[i].BackupPod)
		if nil != err {
			err = fmt.Errorf("TransRoleStatus %s,%v", rollingSet.Name, err)
			glog.Error(err)
			return nil, err
		}
		if !nstatus[i].GetReadyForCurVersion() && !carbonv1.IsStandbyWorker(replicas[i].Worker) {
			allReplicasReady = false
		}
	}
	roleID := GetRoleID(groupID, rollingSet)

	adjustedMinHealthCapacity := getAdjustedMinHealthCapacity(rollingSet)
	minHealthCapacity := getMinHealthCapacity(rollingSet)
	versionedPlans := getVersionedPlan(rollingSet.Spec.Template, rollingSet.Spec.Version, carbonv1.GetQuotaID(rollingSet))

	var status = &carbon.RoleStatusValue{
		RoleId: &roleID,
		GlobalPlan: &carbon.GlobalPlan{
			Count:              rollingSet.Spec.Replicas,
			LatestVersionRatio: rollingSet.Spec.LatestVersionRatio,
		},
		LatestVersion:  &rollingSet.Spec.Version,
		VersionedPlans: versionedPlans,
		Nodes:          nstatus,
		UserDefVersion: &rollingSet.Spec.UserDefVersion,
		ReadyForCurVersion: utils.BoolPtr(allReplicasReady &&
			(rollingSet.Status.ServiceReady || rollingSet.Status.Complete) &&
			rollingSet.Status.Version == rollingSet.Spec.Version &&
			rollingSet.Status.UserDefVersion == rollingSet.Spec.UserDefVersion),
		//migrate k8s
		MinHealthCapacity:         &minHealthCapacity,
		AdjustedCount:             rollingSet.Spec.Replicas,
		AdjustedMinHealthCapacity: &minHealthCapacity,
	}

	if nil != rollingSet.Spec.ScaleSchedulePlan && nil != rollingSet.Spec.ScaleSchedulePlan.Replicas {
		status.AdjustedCount = rollingSet.Spec.ScaleSchedulePlan.Replicas
	}
	if 0 != adjustedMinHealthCapacity {
		status.AdjustedMinHealthCapacity = &adjustedMinHealthCapacity
	}
	return status, nil
}

// GetRoleID  get role id from rollingSet
func GetRoleID(groupID string, rollingSet *carbonv1.RollingSet) string {
	roleName := carbonv1.GetRoleName(rollingSet)
	return strings.TrimPrefix(roleName, utils.AppendString(groupID, "."))
}

// TransReplicaStatus TransReplicaStatus
func TransReplicaStatus(rollingSet *carbonv1.RollingSet, worker *carbonv1.WorkerNode, pod *corev1.Pod,
	backupworker *carbonv1.WorkerNode, backuppod *corev1.Pod) (*carbon.ReplicaNodeStatus, error) {
	var currentWorkerStatus *carbon.WorkerNodeStatus
	var backupWorkerStatus *carbon.WorkerNodeStatus
	var err error
	if nil == worker {
		return &carbon.ReplicaNodeStatus{}, nil
	}
	currentWorkerStatus, err = TransWorkerStatus(rollingSet, worker, pod)
	if currentWorkerStatus.GetCurVersion() == "" {
		currentWorkerStatus.CurVersion = &worker.Spec.Version
	}
	if nil != err {
		err = fmt.Errorf("TransWorkerStatus error %s,%v", carbonv1.GetReplicaID(worker), err)
		return nil, err
	}

	if nil != backupworker {
		backupWorkerStatus, err = TransWorkerStatus(rollingSet, backupworker, backuppod)
		if nil != err {
			err = fmt.Errorf("TransWorkerStatus error %s,%v", carbonv1.GetReplicaID(worker), err)
			return nil, err
		}
	}
	var status = &carbon.ReplicaNodeStatus{
		ReplicaNodeId:          utils.StringPtr(carbonv1.GetReplicaID(worker)),
		CurWorkerNodeStatus:    currentWorkerStatus,
		BackupWorkerNodeStatus: backupWorkerStatus,
		TimeStamp:              utils.Int64Ptr(worker.Status.LastUpdateStatusTime.UnixNano() / 1000),
		UserDefVersion:         &worker.Status.UserDefVersion,
		ReadyForCurVersion:     isWorkerReadyForCurVersion(rollingSet, worker),
	}
	return status, nil
}

func TransCloneSetToGroupStatus(cloneSet *v1alpha1.CloneSet, pods []*corev1.Pod) (*carbon.GroupStatus, error) {
	runtimeID := ""
	if cloneSet.Labels != nil {
		runtimeID = cloneSet.Labels[carbonv1.LabelRuntimeID]
	}
	if runtimeID == "" {
		return nil, nil
	}
	var nodes []*carbon.ReplicaNodeStatus
	for i := range pods {
		pod := pods[i]
		replica := &carbon.ReplicaNodeStatus{
			ReplicaNodeId:       &pod.Name,
			CurWorkerNodeStatus: transCloneSetPodStatus(cloneSet, pod),
		}
		nodes = append(nodes, replica)
	}
	status := &carbon.GroupStatus{
		GroupId: &runtimeID,
		Roles: map[string]*carbon.RoleStatusValue{
			runtimeID: {
				RoleId: &runtimeID,
				Nodes:  nodes,
			},
		},
	}
	return status, nil
}

func transFakeWorkerByCloneSetPod(cloneSet *v1alpha1.CloneSet, pod *corev1.Pod) *carbonv1.WorkerNode {
	workerPhase := carbonv1.TransPodPhase(pod, true)
	podReady := podutil.IsPodReady(pod)
	health := carbonv1.HealthUnKnown
	if podReady {
		health = carbonv1.HealthAlive
	}
	reclaimed := carbonv1.IsPodReclaimed(pod)
	slaveAddress := fmt.Sprintf("%s:%s", pod.Status.HostIP, "7007")
	return &carbonv1.WorkerNode{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      map[string]string{},
			Annotations: map[string]string{},
			Namespace:   pod.Namespace,
		},
		Status: carbonv1.WorkerNodeStatus{
			AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
				Phase:      workerPhase,
				Reclaim:    reclaimed,
				EntityName: pod.Name,
			},
			SlotID: carbonv1.HippoSlotID{
				SlaveAddress: slaveAddress,
				SlotID:       0,
			},
			HealthStatus:   health,
			ServiceOffline: !podReady,
			WorkerReady:    podReady,
		},
	}
}

func transCloneSetPodStatus(cloneSet *v1alpha1.CloneSet, pod *corev1.Pod) *carbon.WorkerNodeStatus {
	worker := transFakeWorkerByCloneSetPod(cloneSet, pod)
	healthInfo, _ := TransHealthInfo(worker)
	slotInfo, restarting, _ := TransSlotInfo(worker, pod)
	slotStatus, _ := TransRoleSlotStatus(worker, restarting)
	labels, _ := TransWorkerNodeStatusLabels(worker, pod)
	slotStatus.SlotId = slotInfo.SlotId
	return &carbon.WorkerNodeStatus{
		WorkerNodeId: &pod.Name,
		Offline:      &worker.Status.ServiceOffline,
		Reclaiming:   &worker.Status.Reclaim,
		SlotStatus:   slotStatus,
		HealthInfo:   healthInfo,
		Labels:       labels,
		Ip:           &pod.Status.PodIP,
	}
}

// TransWorkerStatus TransWorkerStatus
func TransWorkerStatus(rollingSet *carbonv1.RollingSet, worker *carbonv1.WorkerNode, pod *corev1.Pod) (*carbon.WorkerNodeStatus, error) {
	allocStatus, _ := TransAllocStatus(worker)
	slotInfo, restarting, _ := TransSlotInfo(worker, pod)
	healthInfo, _ := TransHealthInfo(worker)
	serviceInfo, _ := TransServiceInfo(worker)
	slotStatus, _ := TransRoleSlotStatus(worker, restarting)
	labels, _ := TransWorkerNodeStatusLabels(worker, pod)
	slotStatus.SlotId = slotInfo.SlotId

	var status = carbon.WorkerNodeStatus{
		WorkerNodeId:       &worker.Name,
		CurVersion:         &worker.Status.Version,
		NextVersion:        &worker.Spec.Version,
		Offline:            &worker.Status.ServiceOffline,
		Releasing:          utils.BoolPtr(carbonv1.IsWorkerToRelease(worker)),
		Reclaiming:         &worker.Status.Reclaim,
		ReadyForCurVersion: isWorkerReadyForCurVersion(rollingSet, worker),
		LastNotMatchTime:   &worker.Status.LastProcessNotMatchtime,
		LastNotReadyTime:   &worker.Status.LastWorkerNotReadytime,
		SlotAllocStatus:    allocStatus,
		SlotInfo:           slotInfo,
		HealthInfo:         healthInfo,
		ServiceInfo:        serviceInfo,
		SlotStatus:         slotStatus,
		Ip:                 &worker.Status.IP,
		UserDefVersion:     &worker.Status.UserDefVersion,
		TargetSignature:    &worker.Spec.Signature,
		TargetCustomInfo:   utils.StringPtr(carbonv1.GetCustomInfo(&worker.Spec.VersionPlan)),
		Labels:             labels,
		WorkerMode:         utils.StringPtr(string(carbonv1.GetWorkerTargetModeType(worker))),
	}
	return &status, nil
}

func isWorkerReadyForCurVersion(rollingSet *carbonv1.RollingSet, worker *carbonv1.WorkerNode) *bool {
	readyForCurVersion := (worker.Status.ServiceReady || worker.Status.Complete) && (worker.Status.Version == worker.Spec.Version)
	if nil != rollingSet {
		readyForCurVersion = readyForCurVersion && (worker.Status.Version == rollingSet.Spec.Version) &&
			(worker.Status.UserDefVersion == rollingSet.Spec.UserDefVersion)
	}
	return &readyForCurVersion
}

//enum SlotAllocStatus {
//    SAS_UNASSIGNED = 0,
//    SAS_ASSIGNED,
//    SAS_LOST,
//    SAS_OFFLINING,
//    SAS_RELEASING,
//    SAS_RELEASED
//};

// TransAllocStatus TransAllocStatus
func TransAllocStatus(worker *carbonv1.WorkerNode) (*carbon.SlotAllocStatus, error) {
	var status carbon.SlotAllocStatus
	switch worker.Status.AllocStatus {
	case carbonv1.WorkerUnAssigned:
		status = carbon.SlotAllocStatus_SAS_UNASSIGNED
	case carbonv1.WorkerAssigned:
		status = carbon.SlotAllocStatus_SAS_ASSIGNED
	case carbonv1.WorkerLost:
		status = carbon.SlotAllocStatus_SAS_LOST
	case carbonv1.WorkerOfflining:
		status = carbon.SlotAllocStatus_SAS_OFFLINING
	case carbonv1.WorkerReleasing:
		status = carbon.SlotAllocStatus_SAS_RELEASING
	case carbonv1.WorkerReleased:
		status = carbon.SlotAllocStatus_SAS_RELEASED
	}
	return &status, nil
}

func transPackageStatus(pod *corev1.Pod) *hippo.PackageStatus {
	status := hippo.PackageStatus_IS_INSTALLED
	if corev1.PodUnknown == pod.Status.Phase {
		status = hippo.PackageStatus_IS_UNKNOWN
	} else if corev1.PodPending == pod.Status.Phase {
		if "" == pod.Status.HostIP {
			status = hippo.PackageStatus_IS_UNKNOWN
		} else {
			status = hippo.PackageStatus_IS_INSTALLING
		}
	}
	var packageStatus hippo.PackageStatus
	packageStatus.Status = &status
	return &packageStatus
}

func transSlaveStatus(pod *corev1.Pod) *hippo.SlaveStatus {
	status := hippo.SlaveStatus_UNKNOWN
	if "" != pod.Status.HostIP {
		status = hippo.SlaveStatus_ALIVE
	}
	var slaveStatus hippo.SlaveStatus
	slaveStatus.Status = &status
	return &slaveStatus
}

func transSlotResource(pod *corev1.Pod, containers []corev1.Container) *carbon.SlotResource {
	if len(containers) == 0 {
		return nil
	}
	resourceAmountMap := make(map[string]int64)
	for _, container := range containers {
		resourceList := container.Resources.Limits
		for k, v := range resourceList {
			ammount, ok := resourceAmountMap[k.String()]
			var value int64
			switch k.String() {
			case carbonv1.ResourceMEM:
				value = v.Value() / 1024 / 1024
			case carbonv1.ResourceCPU:
				value = v.MilliValue() / 10
			case carbonv1.ResourceDISK:
				value = v.Value() / 1024 / 1024
			case "ephemeral-storage":
				value = v.Value() / 1024 / 1024
			default:
				if strings.HasPrefix(k.String(), carbonv1.ResourceDISKPrefix) {
					value = v.Value() / 1024 / 1024
				} else {
					value = v.Value()
				}
			}
			if ok {
				resourceAmountMap[k.String()] = ammount + value
			} else {
				resourceAmountMap[k.String()] = value
			}

		}
	}
	ip := getPodIP(pod)
	if pod != nil && !pod.Spec.HostNetwork && ip != "" {
		resourceAmountMap["ip_"+ip+":"] = 1
	}

	sortKey := sortKey(resourceAmountMap)
	resources := []*hippo.Resource{}
	for _, key := range sortKey {
		rv := resourceAmountMap[key]
		rType := hippo.Resource_SCALAR
		name := key
		if key == carbonv1.ResourceMEM {
			name = "mem"
		}
		resource := &hippo.Resource{Name: utils.StringPtr(name), Amount: utils.Int32Ptr(int32(rv)), Type: &rType}
		resources = append(resources, resource)
	}
	return &carbon.SlotResource{SlotResources: resources}
}

func sortKey(vMap map[string]int64) []string {
	keys := make([]string, 0)
	for k := range vMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

//ProcessStatus_PS_UNKNOWN    ProcessStatus_Status = 0
//ProcessStatus_PS_RUNNING    ProcessStatus_Status = 1
//ProcessStatus_PS_RESTARTING ProcessStatus_Status = 2
//ProcessStatus_PS_STOPPING   ProcessStatus_Status = 3
//ProcessStatus_PS_STOPPED    ProcessStatus_Status = 4
//ProcessStatus_PS_FAILED     ProcessStatus_Status = 5
//ProcessStatus_PS_TERMINATED ProcessStatus_Status = 6

// TransSlotInfo TransSlotInfo
func TransSlotInfo(worker *carbonv1.WorkerNode, pod *corev1.Pod) (*carbon.SlotInfo, bool, error) {
	var slotInfo = carbon.SlotInfo{
		Role:       utils.StringPtr(carbonv1.GetRoleName(worker)),
		Reclaiming: &worker.Status.Reclaim,
	}
	var restarting bool
	var processStatus []*hippo.ProcessStatus
	if nil != pod {
		processStatus, restarting = transProcessStatus(pod)
		slotInfo.SlotId = &hippo.SlotId{}
		slotInfo.SlotId.SlaveAddress = &worker.Status.SlotID.SlaveAddress
		slotInfo.SlotId.Id = utils.Int32Ptr(int32(worker.Status.SlotID.SlotID))
		slotInfo.SlotId.DeclareTime = utils.Int64Ptr(pod.CreationTimestamp.Time.Unix() * 1000 * 1000)

		slotInfo.ProcessStatus = processStatus
		slotInfo.PackageStatus = transPackageStatus(pod)
		slotInfo.SlaveStatus = transSlaveStatus(pod)

		slotInfo.SlotResource = transSlotResource(pod, pod.Spec.Containers) //pod.spec
	}
	return &slotInfo, restarting, nil
}

func transProcessStatus(pod *corev1.Pod) ([]*hippo.ProcessStatus, bool) {
	var restarting bool
	var processStatus = make([]*hippo.ProcessStatus, 0, len(pod.Status.ContainerStatuses))
	for i := range pod.Status.ContainerStatuses {
		if pod.Status.ContainerStatuses[i].Name == "pause" || pod.Status.ContainerStatuses[i].Name == "staragent" {
			continue
		}
		processName := pod.Status.ContainerStatuses[i].Name
		alias := getAlias(pod.Status.ContainerStatuses[i].Name, pod)
		if "" != alias {
			processName = alias
		}
		var processStatu = &hippo.ProcessStatus{
			IsDaemon:     utils.BoolPtr(transDaemon(pod)),
			ProcessName:  &processName,
			RestartCount: utils.Int32Ptr(pod.Status.ContainerStatuses[i].RestartCount),
		}
		var status hippo.ProcessStatus_Status
		if pod.Status.ContainerStatuses[i].State.Terminated != nil {
			processStatu.ExitCode = utils.Int32Ptr(pod.Status.ContainerStatuses[i].State.Terminated.ExitCode)
			processStatu.StartTime = utils.Int64Ptr(pod.Status.ContainerStatuses[i].State.Terminated.StartedAt.UnixNano() / 1000)
			status = hippo.ProcessStatus_PS_TERMINATED
		} else if pod.Status.ContainerStatuses[i].State.Waiting != nil && pod.Status.ContainerStatuses[i].LastTerminationState.Terminated != nil {
			processStatu.ExitCode = utils.Int32Ptr(pod.Status.ContainerStatuses[i].LastTerminationState.Terminated.ExitCode)
			processStatu.StartTime = utils.Int64Ptr(pod.Status.ContainerStatuses[i].LastTerminationState.Terminated.StartedAt.UnixNano() / 1000)
			status = hippo.ProcessStatus_PS_RESTARTING
			if pod.Status.ContainerStatuses[i].RestartCount >= 10 {
				status = hippo.ProcessStatus_PS_FAILED
			}
		} else if pod.Status.ContainerStatuses[i].State.Running != nil {
			processStatu.StartTime = utils.Int64Ptr(pod.Status.ContainerStatuses[i].State.Running.StartedAt.UnixNano() / 1000)
			if pod.Status.Phase == corev1.PodSucceeded {
				status = hippo.ProcessStatus_PS_TERMINATED
			} else if pod.Status.ContainerStatuses[i].Ready {
				status = hippo.ProcessStatus_PS_RUNNING
			} else {
				status = hippo.ProcessStatus_PS_RESTARTING
				restarting = true
			}
		} else {
			status = hippo.ProcessStatus_PS_UNKNOWN
		}
		processStatu.Status = &status
		processStatus = append(processStatus, processStatu)
	}
	// pod status被清空
	if pod.Status.Phase == corev1.PodFailed && len(pod.Status.ContainerStatuses) == 0 && len(processStatus) == 0 {
	}
	return processStatus, restarting
}

func getAlias(containerName string, pod *corev1.Pod) string {
	for k, v := range pod.Annotations {
		if k == carbonv1.AnnotationKeySpecExtend {
			var podSpecExtend carbonv1.HippoPodSpecExtend
			err := json.Unmarshal([]byte(v), &podSpecExtend)
			if nil != err {
				glog.Errorf("unmarshal error :%s,%v", v, err)
			}
			for _, container := range podSpecExtend.Containers {
				if container.Name == containerName {
					return container.Alias
				}
			}
		}
	}
	if carbonv1.IsAsiPod(&pod.ObjectMeta) {
	}
	return ""
}

func transDaemon(pod *corev1.Pod) bool {
	if pod.Spec.RestartPolicy == corev1.RestartPolicyAlways {
		return true
	}
	return false
}

//enum HealthType {
//    HT_UNKNOWN = 0,
//    HT_LOST,
//    HT_ALIVE,
//    HT_DEAD
//};

// TransHealthType TransHealthType
func TransHealthType(worker *carbonv1.WorkerNode) (*carbon.HealthType, error) {
	var healthType carbon.HealthType
	switch worker.Status.HealthStatus {
	case carbonv1.HealthUnKnown:
		healthType = carbon.HealthType_HT_UNKNOWN
	case carbonv1.HealthLost:
		healthType = carbon.HealthType_HT_LOST
	case carbonv1.HealthAlive:
		healthType = carbon.HealthType_HT_ALIVE
	case carbonv1.HealthDead:
		healthType = carbon.HealthType_HT_DEAD
	}
	return &healthType, nil
}

//enum WorkerType {
//    WT_UNKNOWN = 0,
//    WT_NOT_READY,
//    WT_READY
//};

// TransWorkerType TransWorkerType
func TransWorkerType(worker *carbonv1.WorkerNode) (*carbon.WorkerType, error) {
	var workerType carbon.WorkerType
	switch worker.Status.HealthCondition.WorkerStatus {
	case carbonv1.WorkerTypeUnknow:
		workerType = carbon.WorkerType_WT_UNKNOWN
	case carbonv1.WorkerTypeNotReady:
		workerType = carbon.WorkerType_WT_NOT_READY
	case carbonv1.WorkerTypeReady:
		workerType = carbon.WorkerType_WT_READY
	}
	return &workerType, nil
}

//enum ServiceType {
//    SVT_UNKNOWN = 0,
//    SVT_UNAVAILABLE,
//    SVT_PART_AVAILABLE,
//    SVT_AVAILABLE,
//};

// TransServiceType TransServiceType
func TransServiceType(worker *carbonv1.WorkerNode) (*carbon.ServiceType, error) {
	var serviceType carbon.ServiceType
	switch worker.Status.ServiceStatus {
	case carbonv1.ServiceUnKnown:
		serviceType = carbon.ServiceType_SVT_UNKNOWN
	case carbonv1.ServiceUnAvailable:
		serviceType = carbon.ServiceType_SVT_UNAVAILABLE
	case carbonv1.ServicePartAvailable:
		serviceType = carbon.ServiceType_SVT_PART_AVAILABLE
	case carbonv1.ServiceAvailable:
		serviceType = carbon.ServiceType_SVT_AVAILABLE
	}
	return &serviceType, nil
}

// TransHealthInfo TransHealthInfo
func TransHealthInfo(worker *carbonv1.WorkerNode) (*carbon.HealthInfo, error) {
	healthStatus, _ := TransHealthType(worker)
	workerStatus, _ := TransWorkerType(worker)

	var healthInfo = carbon.HealthInfo{
		HealthStatus: healthStatus,
		WorkerStatus: workerStatus,
		Metas:        carbonv1.GetHealthMetas(worker),
		Version:      &worker.Status.HealthCondition.Version,
	}
	if nil == healthInfo.Metas {
		healthInfo.Metas = make(map[string]string)
	}
	return &healthInfo, nil
}

// TransServiceInfo TransServiceInfo
func TransServiceInfo(worker *carbonv1.WorkerNode) (*carbon.ServiceStatus, error) {
	serviceStatus, _ := TransServiceType(worker)
	var serviceInfo = carbon.ServiceStatus{
		Status: serviceStatus,
	}

	if nil == serviceInfo.Metas {
		serviceInfo.Metas = make(map[string]string)
	}
	return &serviceInfo, nil
}

//	enum SlotType {
//	   ST_PKG_FAILED = 0,
//	   ST_UNKNOWN,
//	   ST_DEAD,
//	   ST_PROC_FAILED,
//	   ST_PROC_RESTARTING,
//	   ST_PROC_RUNNING
//	};
const (
	StPkgFailed      = 0
	StUnKnown        = 1
	StDead           = 2
	StProcFailed     = 3
	StProcRestarting = 4
	StRunning        = 5
	StProcTerminated = 6
)

// TransWorkerNodeStatusLabels TransWorkerNodeStatusLabels
func TransWorkerNodeStatusLabels(worker *carbonv1.WorkerNode, pod *corev1.Pod) (map[string]string, error) {
	labels := make(map[string]string)
	labels["app.c2.io/pod-name"] = worker.Status.EntityName
	subrsName := carbonv1.GetSubrsShortUniqueKey(worker)
	if subrsName != "" {
		labels["subrs"] = carbonv1.SubrsTextPrefix + subrsName
	}
	if worker.Status.WorkerMode == carbonv1.WorkerModeTypeInactive {
		labels[carbonv1.LabelKeyInactive] = "true"
	}
	labels["namespace"] = worker.Namespace
	if reason := worker.Status.UnassignedReason; reason != carbonv1.UnassignedReasonNone {
		labels[carbonv1.LabelKeyUnassignedReason] = strconv.Itoa(int(reason))
	}
	if msg := worker.Status.UnassignedMessage; msg != "" {
		labels[carbonv1.LabelKeyUnassignedMsg] = worker.Status.UnassignedMessage
	}
	if pod != nil {
		if s := pod.Labels[carbonv1.LabelKeyPodSN]; s != "" {
			labels[carbonv1.LabelKeyPodSN] = s
		}
		if s := pod.Labels[carbonv1.LabelKeyPreInactive]; s != "" {
			labels[carbonv1.LabelKeyPreInactive] = s
		}
		if poolScore := carbonv1.WorkerResourcePoolScore(worker); poolScore != 0 {
			labels["app.c2.io/pod-pool-score"] = strconv.FormatInt(int64(poolScore), 10)
		}
		if cost := carbonv1.GetWorkerDeletionCost(worker); cost != 0 {
			labels["app.c2.io/pod-deletion-cost"] = strconv.FormatInt(cost, 10)
		}
		if features.C2MutableFeatureGate.Enabled(features.EnableTransContainers) {
			labels["app.c2.io/container-statuses"] = simplifyContainerStatus(&pod.Status)
		}
	}
	err := copyStatusLabelsByConfig(worker, pod, labels)
	if err != nil {
		glog.Errorf("copy worker status labels by config error:%v", err)
	}
	return labels, err
}

func copyStatusLabelsByConfig(worker *carbonv1.WorkerNode, pod *corev1.Pod, labels map[string]string) error {
	lc := carbonv1.LabelsCopy{}
	val, err := common.GetGlobalConfigVal(common.ConfProxyCopyLabels, "")
	if err != nil || val == "" {
		return err
	}
	if err = json.Unmarshal([]byte(val), &lc); err != nil {
		return err
	}
	if pod != nil {
		copy2Labels(pod.Labels, lc.PodLabels, labels)
		copy2Labels(pod.Annotations, lc.PodAnnotations, labels)
	}
	if worker != nil {
		copy2Labels(worker.Labels, lc.WorkerLabels, labels)
		copy2Labels(worker.Annotations, lc.WorkerAnnotations, labels)
	}
	return nil
}

func copy2Labels(src map[string]string, ct carbonv1.CopyTarget, labels map[string]string) {
	if len(src) == 0 || len(ct) == 0 {
		return
	}
	for prefix, fullKeys := range ct {
		for i := range fullKeys {
			fullKey := fullKeys[i]
			val := src[fullKey]
			if val == "" {
				continue
			}
			if prefix == carbonv1.DirectLabelCopyMappingKey {
				labels[fullKey] = val
				continue
			}
			split := strings.Split(fullKey, "/")
			key := ""
			if len(split) == 1 {
				key = fmt.Sprintf("%s/%s", prefix, fullKey)
			} else if len(split) == 2 {
				split[0] = prefix
				key = strings.Join(split, "/")
			}
			if key != "" {
				labels[key] = val
			}
		}
	}
}

// TransRoleSlotStatus TransRoleSlotStatus
func TransRoleSlotStatus(worker *carbonv1.WorkerNode, restarting bool) (*carbon.SlotStatus, error) {
	var slotType carbon.SlotType
	if worker.Status.Phase == carbonv1.Running {
		slotType = carbon.SlotType_ST_PROC_RUNNING
	} else if worker.Status.Phase == carbonv1.Terminated {
		if carbonv1.IsDaemonWorker(worker) {
			slotType = carbon.SlotType_ST_PROC_FAILED
		} else {
			slotType = carbon.SlotType_ST_PROC_TERMINATED
		}
	} else if worker.Status.Phase == carbonv1.Failed {
		slotType = carbon.SlotType_ST_PROC_FAILED
	} else {
		slotType = carbon.SlotType_ST_UNKNOWN
	}
	if restarting {
		slotType = carbon.SlotType_ST_PROC_RESTARTING
	}
	var roleSlotStatus = carbon.SlotStatus{Status: &slotType}
	return &roleSlotStatus, nil
}

func getIntOrPercentValue(intOrStr *intstrutil.IntOrString) (int, bool, error) {
	switch intOrStr.Type {
	case intstrutil.Int:
		return intOrStr.IntValue(), false, nil
	case intstrutil.String:
		s := strings.Replace(intOrStr.StrVal, "%", "", -1)
		v, err := strconv.Atoi(s)
		if err != nil {
			return 0, false, fmt.Errorf("invalid value %q: %v", intOrStr.StrVal, err)
		}
		return int(v), true, nil
	}
	return 0, false, fmt.Errorf("invalid type: neither int nor percentage")
}

const (
	defaultMinHealthCapacity = 75
)

func getAdjustedMinHealthCapacity(rollingSet *carbonv1.RollingSet) int32 {
	if rollingSet.Spec.ScaleSchedulePlan == nil || rollingSet.Spec.ScaleSchedulePlan.Strategy.RollingUpdate == nil ||
		rollingSet.Spec.ScaleSchedulePlan.Strategy.RollingUpdate.MaxUnavailable == nil {
		return 0
	}
	return getMihHealthFromMaxUnavailable(rollingSet.Spec.ScaleSchedulePlan.Strategy.RollingUpdate.MaxUnavailable)
}

func getMinHealthCapacity(rollingSet *carbonv1.RollingSet) int32 {
	if rollingSet.Spec.Strategy.RollingUpdate == nil ||
		rollingSet.Spec.Strategy.RollingUpdate.MaxUnavailable == nil {
		return defaultMinHealthCapacity
	}
	return getMihHealthFromMaxUnavailable(rollingSet.Spec.Strategy.RollingUpdate.MaxUnavailable)
}
func getMihHealthFromMaxUnavailable(maxUnavailable *intstrutil.IntOrString) int32 {
	maxUnavailableInt, _, err := getIntOrPercentValue(maxUnavailable)
	if err != nil {
		return defaultMinHealthCapacity
	}
	return int32(100 - maxUnavailableInt)
}

// 只保存latestVersion的Resources, Priority
func getVersionedPlan(template *carbonv1.HippoPodTemplate, version, quotaID string) map[string]*carbon.VersionedPlan {
	versionedPlans := make(map[string]*carbon.VersionedPlan)
	latestVersion := version
	hippoPodSpec, _, err := carbonv1.GetPodSpecFromTemplate(template)
	if hippoPodSpec == nil || err != nil {
		glog.Errorf("unmarshal error :%v", err)
		return nil
	}
	containers := make([]corev1.Container, len(hippoPodSpec.Containers))
	for i, container := range hippoPodSpec.Containers {
		containers[i] = container.Container
	}
	slotResource := transSlotResource(nil, containers)
	var resources []*carbon.SlotResource
	if slotResource != nil {
		resources = make([]*carbon.SlotResource, 0)
		resources = append(resources, slotResource)
	}
	latestVersionedPlan := carbon.VersionedPlan{
		ResourcePlan: &carbon.ResourcePlan{
			Resources: resources,
			Group:     utils.StringPtr(quotaID),
		},
	}
	versionedPlans[latestVersion] = &latestVersionedPlan
	return versionedPlans
}

type minimalContainerStatus struct {
	Name         string `json:"name"`
	Ready        bool   `json:"ready"`
	RestartCount int32  `json:"restartCount"`
	Started      *bool  `json:"started"`
}

func simplifyContainerStatus(podStatus *corev1.PodStatus) string {
	var statuses []minimalContainerStatus
	for _, status := range podStatus.ContainerStatuses {
		statuses = append(statuses, minimalContainerStatus{
			Name:         status.Name,
			Ready:        status.Ready,
			RestartCount: status.RestartCount,
			Started:      status.Started,
		})
	}
	return utils.ObjJSON(statuses)
}

func getPodIP(pod *corev1.Pod) string {
	if pod == nil {
		return ""
	}
	if pod.Status.PodIP != "" {
		return pod.Status.PodIP
	}
	return ""
}
