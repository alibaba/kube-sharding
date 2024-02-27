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

package v1

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	mapset "github.com/deckarep/golang-set"

	"github.com/alibaba/kube-sharding/common"
	"github.com/alibaba/kube-sharding/pkg/rollalgorithm"

	"github.com/gogo/protobuf/proto"

	"github.com/alibaba/kube-sharding/common/features"
	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labelsutil "k8s.io/kubernetes/pkg/util/labels"

	"github.com/alibaba/kube-sharding/pkg/memkube/mem"
	v1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/labels"
	glog "k8s.io/klog"
)

// Signable is used to for rollingSet and deployment to sign
type Signable interface {
	Signature() (rollingSign string, refreshSign string, err error)
}

// IsWorkerUnAssigned means is the worker is unassigned
func IsWorkerUnAssigned(worker *WorkerNode) bool {
	if nil == worker {
		return false
	}
	return worker.Status.AllocStatus == WorkerUnAssigned || worker.Status.AllocStatus == ""
}

// SetWorkerUnAssigned setter
func SetWorkerUnAssigned(worker *WorkerNode) {
	if nil == worker {
		return
	}
	worker.Status.AllocStatus = WorkerUnAssigned
}

// IsWorkerAssigned means is worker unassigned
func IsWorkerAssigned(worker *WorkerNode) bool {
	if nil == worker {
		return false
	}
	return worker.Status.AllocStatus == WorkerAssigned
}

// SetWorkerAssigned setter
func SetWorkerAssigned(worker *WorkerNode) {
	if nil == worker {
		return
	}
	worker.Status.AllocStatus = WorkerAssigned
	if 0 == worker.Status.AssignedTime {
		worker.Status.AssignedTime = time.Now().Unix()
	}
}

// RecordPodReadyTime worker
func RecordPodReadyTime(worker *WorkerNode) {
	if nil == worker {
		return
	}
	// 存量worker不记录，避免不合理数据
	if worker.Status.ProcessReady && 0 != worker.Status.AssignedTime && 0 == worker.Status.PodReadyTime {
		worker.Status.PodReadyTime = time.Now().Unix()
	}
}

// RecordWorkerReadyTime worker
func RecordWorkerReadyTime(worker *WorkerNode) {
	if nil == worker {
		return
	}
	// 存量worker不记录，避免不合理数据
	if worker.Status.WorkerReady && 0 != worker.Status.AssignedTime && 0 == worker.Status.WorkerReadyTime {
		worker.Status.WorkerReadyTime = time.Now().Unix()
	}
}

// IsWorkerLost means is worker lost
func IsWorkerLost(worker *WorkerNode) bool {
	if nil == worker {
		return false
	}
	return worker.Status.AllocStatus == WorkerLost
}

// IsWorkerFalied means is worker lost
func IsWorkerFalied(worker *WorkerNode) bool {
	if nil == worker {
		return false
	}
	return worker.Status.Phase == Failed
}

// IsWorkerIrrecoverable IsWorkerIrrecoverable
func IsWorkerIrrecoverable(worker *WorkerNode) bool {
	return IsWorkerLost(worker) || IsWorkerFalied(worker)
}

// IsWorkerDependencyReady IsWorkerDependencyReady
func IsWorkerDependencyReady(worker *WorkerNode) bool {
	if nil == worker || nil == worker.Spec.DependencyReady {
		return true
	}
	return *worker.Spec.DependencyReady
}

// IsStandbyWorker IsStandbyWorker
func IsStandbyWorker(worker *WorkerNode) bool {
	if nil == worker {
		return false
	}
	return IsStandbyWorkerMode(worker.Spec.WorkerMode)
}

// IsInactiveWorker IsInactiveWorker
func IsInactiveWorker(worker *WorkerNode) bool {
	if nil == worker {
		return false
	}
	return worker.Spec.WorkerMode == WorkerModeTypeInactive
}

// IsQuickOnline IsQuickOnline
func IsQuickOnline(object metav1.Object) bool {
	return GetLabelAnnoValue(object, AnnotationC2Spread) != ""
}

// IsStandbyWorkerMode IsStandbyWorkerMode
func IsStandbyWorkerMode(workerMode WorkerModeType) bool {
	return workerMode != "" && workerMode != WorkerModeTypeActive
}

// IsHotStandbyWorker IsHotStandbyWorker
func IsHotStandbyWorker(worker *WorkerNode) bool {
	if nil == worker {
		return false
	}
	return worker.Spec.WorkerMode == WorkerModeTypeHot
}

// IsCurrentHotStandbyWorker current hot
func IsCurrentHotStandbyWorker(worker *WorkerNode) bool {
	if nil == worker {
		return false
	}
	return worker.Status.WorkerMode == WorkerModeTypeHot
}

// IsPodInactive IsPodInactive
func IsPodInactive(pod *corev1.Pod) bool {
	if pod == nil || pod.Labels == nil {
		return false
	}
	if pod.Labels[LabelAllContainersStatus] == "exited" {
		return true
	}
	return false
}

// IsPodPreInactive IsPodPreInactive
func IsPodPreInactive(pod *corev1.Pod) bool {
	if pod == nil || pod.Labels == nil {
		return false
	}
	if pod.Labels[LabelKeyPreInactive] == "true" {
		return true
	}
	return false
}

// GetWorkerTargetModeType GetWorkerTargetModeType
func GetWorkerTargetModeType(worker *WorkerNode) WorkerModeType {
	if nil == worker || worker.Spec.WorkerMode == "" {
		return WorkerModeTypeActive
	}
	return worker.Spec.WorkerMode
}

// GetWorkerCurrentModeType GetWorkerCurrentModeType
func GetWorkerCurrentModeType(worker *WorkerNode) WorkerModeType {
	if nil == worker || worker.Status.WorkerMode == "" {
		return WorkerModeTypeActive
	}
	return worker.Status.WorkerMode
}

// IsColdStandbyWorker IsColdStandbyWorker
func IsColdStandbyWorker(worker *WorkerNode) bool {
	if nil == worker {
		return false
	}
	return worker.Spec.WorkerMode == WorkerModeTypeCold
}

// IsWorkerOnStandByResourcePool IsWorkerOnStandByResourcePool
func IsWorkerOnStandByResourcePool(worker *WorkerNode) bool {
	if nil == worker {
		return false
	}
	return false
}

func GetWorkerStandbyHours(worker *WorkerNode) []int64 {
	if nil == worker {
		return nil
	}
	status := worker.Status.AllocatorSyncedStatus.StandbyHours
	if status == "" {
		return nil
	}
	var hours []int64
	err := json.Unmarshal([]byte(status), &hours)
	if err != nil {
		return nil
	}
	return hours
}

// GetInactiveSpreadReplicas GetInactiveSpreadReplicas
func GetInactiveSpreadReplicas(object metav1.Object) (active, inactive int32) {
	spread := unmarshalSpread(object)
	if spread == nil {
		return -1, -1
	}
	return int32(spread.Active.Replica), int32(spread.Inactive.Replica)
}

func unmarshalSpread(object metav1.Object) *Spread {
	val := object.GetAnnotations()[AnnotationC2Spread]
	if val == "" {
		return nil
	}
	var spread Spread
	if err := json.Unmarshal([]byte(val), &spread); err != nil {
		glog.Errorf("unmarshal spread error, val: %s, err:%v", val, err)
		return nil
	}
	return &spread
}

// GetInactiveInterpose GetInactiveInterpose
func GetInactiveInterpose(object metav1.Object) (labels, annotations map[string]string) {
	spread := unmarshalSpread(object)
	if spread == nil {
		return nil, nil
	}
	return spread.Inactive.Labels, spread.Inactive.Annotations
}

// SetWorkerLost setter
func SetWorkerLost(worker *WorkerNode) {
	if nil == worker {
		return
	}
	worker.Status.AllocStatus = WorkerLost
}

// IsWorkerOfflining means is worker offlining
func IsWorkerOfflining(worker *WorkerNode) bool {
	if nil == worker {
		return false
	}
	return worker.Status.AllocStatus == WorkerOfflining
}

// IsWorkerReclaiming means is worker reclaim
func IsWorkerReclaiming(worker *WorkerNode) bool {
	if nil == worker {
		return false
	}
	if IsWorkerAssigned(worker) {
		return worker.Status.Reclaim || (worker.Status.InternalReclaim && IsWorkerVersionMisMatch(worker)) || worker.Spec.Reclaim
	}
	if IsWorkerUnAssigned(worker) {
		return worker.Status.Reclaim || worker.Status.InternalReclaim || worker.Spec.Reclaim
	}
	return false
}

// SetWorkerOfflining setter
func SetWorkerOfflining(worker *WorkerNode) {
	if nil == worker {
		return
	}
	worker.Status.AllocStatus = WorkerOfflining
}

// IsWorkerReleasing means is worker alloc status is in releasing
func IsWorkerReleasing(worker *WorkerNode) bool {
	if nil == worker {
		return false
	}
	return worker.Status.AllocStatus == WorkerReleasing
}

// SetWorkerReleasing setter
func SetWorkerReleasing(worker *WorkerNode) {
	if nil == worker {
		return
	}
	worker.Status.AllocStatus = WorkerReleasing
}

// IsWorkerReleased means is worker alloc status is in released
func IsWorkerReleased(worker *WorkerNode) bool {
	if nil == worker {
		return false
	}
	return worker.Status.AllocStatus == WorkerReleased
}

// SetWorkerReleased setter
func SetWorkerReleased(worker *WorkerNode) {
	if nil == worker {
		return
	}
	worker.Status.AllocStatus = WorkerReleased
}

// IsWorkerToDelete means is worker to delete
func IsWorkerToDelete(worker *WorkerNode) bool {
	if nil == worker {
		return false
	}
	return worker.Spec.ToDelete || worker.DeletionTimestamp != nil
}

// IsWorkerToRelease means is worker releasing
func IsWorkerToRelease(worker *WorkerNode) bool {
	if nil == worker {
		return false
	}
	return worker.Spec.ToDelete || worker.Status.ToRelease || worker.Spec.Releasing || worker.DeletionTimestamp != nil
}

// SetWorkerToRelease setter
func SetWorkerToRelease(worker *WorkerNode) {
	if nil == worker {
		return
	}
	worker.Status.ToRelease = true
}

// IsWorkerRunning means is worker process running
func IsWorkerRunning(worker *WorkerNode) bool {
	if nil == worker {
		return false
	}
	return worker.Status.Phase == Running ||
		(worker.Status.Phase == Terminated && !IsDaemonWorker(worker))
}

// IsWorkerPending means is worker process pending
func IsWorkerPending(worker *WorkerNode) bool {
	if nil == worker {
		return false
	}
	return worker.Status.Phase == Pending
}

// IsWorkerHealth , process and health check status
func IsWorkerHealth(worker *WorkerNode) bool {
	if nil == worker {
		return false
	}
	return IsWorkerRunning(worker) &&
		worker.Status.HealthStatus == HealthAlive
}

// IsWorkerReady return the ready status provide by allocator
func IsWorkerReady(worker *WorkerNode) bool {
	if nil == worker {
		return false
	}
	return worker.Status.ProcessReady
}

// IsWorkerHealthMatch  worker health for this version
func IsWorkerHealthMatch(worker *WorkerNode) bool {
	if nil == worker {
		return false
	}

	return IsWorkerHealth(worker) && IsWorkerReady(worker) &&
		worker.Status.HealthCondition.Version == worker.Spec.Version
}

// IsWorkerHealthInfoReady version updated and health
func IsWorkerHealthInfoReady(worker *WorkerNode) bool {
	if nil == worker {
		return false
	}

	return IsWorkerAssigned(worker) &&
		worker.Status.ProcessMatch &&
		IsWorkerHealthMatch(worker) &&
		IsWorkerTypeReady(worker)
}

// IsWorkerDead ,process and health check status
func IsWorkerDead(worker *WorkerNode) bool {
	if nil == worker {
		return false
	}
	return worker.Status.HealthStatus == HealthDead || IsWorkerLost(worker) ||
		(worker.Status.Phase == Terminated && isDaemonPlan(worker.Spec.Template)) ||
		worker.Status.Phase == Failed
}

// SetWorkerHealthStatus set workernode HealthStatus and HealthCondition
func SetWorkerHealthStatus(workernode *WorkerNode, healthCondition *HealthCondition) HealthStatus {
	if nil == workernode || nil == healthCondition {
		return HealthUnKnown
	}
	workernode.Status.HealthStatus = healthCondition.Status
	workernode.Status.HealthCondition = *healthCondition
	return healthCondition.Status
}

// IsWorkerLongtimeNotMatch means is worker resource not match for too long time, that exceed the limit
func IsWorkerLongtimeNotMatch(worker *WorkerNode, targetVersion string) bool {
	if nil == worker {
		return false
	}
	if worker.Status.ResourceMatch {
		return false
	}
	if worker.Status.LastResourceNotMatchtime <= 0 {
		return false
	}
	if worker.Spec.ResourceMatchTimeout <= 0 {
		return false
	}
	// 当前正在变更的节点立即处理
	if IsWorkerVersionMisMatch(worker) && worker.Status.Version != "" && worker.Spec.RecoverStrategy != DirectReleasedRecoverStrategy {
		return true
	}
	// 老版本节点不做处理
	if !IsWorkerVersionMisMatch(worker) && NeedRollingAfterResourceChange(&worker.Spec.VersionPlan) {
		return false
	}
	return time.Now().Unix()-worker.Status.LastResourceNotMatchtime > worker.Spec.ResourceMatchTimeout
}

// IsWorkerLongtimeNotSchedule means is worker resource not match for too long time, that exceed the limit
func IsWorkerLongtimeNotSchedule(worker *WorkerNode) bool {
	if nil == worker {
		return false
	}
	if !IsWorkerUnAssigned(worker) {
		return false
	}
	if worker.Status.NotScheduleSeconds <= 0 {
		return false
	}
	if NotScheduleTimeout <= 0 {
		return false
	}
	return worker.Status.NotScheduleSeconds > NotScheduleTimeout
}

// IsWorkerProcessLongtimeNotMatch means is worker process not match for too long time, that exceed the limit
func IsWorkerProcessLongtimeNotMatch(worker *WorkerNode) bool {
	if nil == worker {
		return false
	}
	if worker.Status.ProcessMatch {
		return false
	}
	if worker.Status.LastProcessNotMatchtime <= 0 {
		return false
	}
	if worker.Spec.ProcessMatchTimeout <= 0 {
		return false
	}
	// 老版本节点不做处理
	if !IsWorkerVersionMisMatch(worker) {
		return false
	}
	return time.Now().Unix()-worker.Status.LastProcessNotMatchtime > worker.Spec.ProcessMatchTimeout
}

// IsWorkerProcessLongtimeNotReady means is worker process not ready for too long time, that exceed the limit
func IsWorkerProcessLongtimeNotReady(worker *WorkerNode) bool {
	if nil == worker {
		return false
	}
	if worker.Status.ProcessReady {
		return false
	}
	if worker.Status.LastProcessNotReadytime <= 0 {
		return false
	}
	if worker.Spec.ProcessMatchTimeout <= 0 {
		return false
	}
	return time.Now().Unix()-worker.Status.LastProcessNotReadytime > worker.Spec.ProcessMatchTimeout
}

// IsWorkerLongtimeNotReady means is worker process not ready for too long time, that exceed the limit
func IsWorkerLongtimeNotReady(worker *WorkerNode) bool {
	if nil == worker {
		return false
	}
	if worker.Status.WorkerReady {
		return false
	}
	if worker.Status.LastWorkerNotReadytime <= 0 {
		return false
	}
	if worker.Spec.WorkerReadyTimeout <= 0 {
		return false
	}
	if glog.V(4) {
		glog.Infof("IsWorkerLongtimeNotReady worker.Name:%v %v - %v with %v", worker.Name, time.Now().Unix(), worker.Status.LastWorkerNotReadytime, worker.Spec.WorkerReadyTimeout)
	}
	return time.Now().Unix()-worker.Status.LastWorkerNotReadytime > worker.Spec.WorkerReadyTimeout
}

// IsWorkerBroken means is worker is in wrong status not by unexpected
func IsWorkerBroken(worker *WorkerNode, targetVersion string) BadReasonType {
	if nil == worker {
		return BadReasonNone
	}
	if IsWorkerLost(worker) {
		return BadReasonLost
	}

	if IsWorkerAssigned(worker) {
		reason := BadReasonNone
		switch {
		case IsWorkerDead(worker):
			reason = BadReasonDead
		case IsWorkerLongtimeNotMatch(worker, targetVersion):
			reason = BadReasonNotMatch
		case IsWorkerProcessLongtimeNotMatch(worker):
			reason = BadReasonProcessNotMatch
		case IsWorkerLongtimeNotReady(worker):
			reason = BadReasonNotReady
		case IsWorkerProcessLongtimeNotReady(worker):
			reason = BadReasonProcessNotReady
		case NeedReclaimByService(worker):
			reason = BadReasonServiceReclaim
		}
		if reason > 0 {
			return reason
		}
	}
	if IsWorkerUnAssigned(worker) && IsWorkerLongtimeNotSchedule(worker) {
		return BadReasonNotSchedule
	}
	return BadReasonNone
}

// GetWorkersResyncTime get the min not zero timeout of workers
func GetWorkersResyncTime(workers []*WorkerNode) int64 {
	var timeout = int64(0)
	for i := range workers {
		t := GetWorkerResyncTime(workers[i])
		if timeout == 0 || (t != 0 && t < timeout) {
			timeout = t
		}
	}
	return timeout
}

// GetWorkerResyncTime if worker is not IsWorkerLongtimeNotMatch nor IsWorkerLongtimeNotReady, get the min timeout time
func GetWorkerResyncTime(worker *WorkerNode) int64 {
	if nil == worker {
		return 0
	}

	var timeout = int64(0)
	if IsWorkerAssigned(worker) {
		if IsWorkerDead(worker) {
			return 0
		}
		matchTimeout := worker.Spec.ResourceMatchTimeout - (time.Now().Unix() - worker.Status.LastResourceNotMatchtime)
		processMatchTimeout := worker.Spec.ProcessMatchTimeout - (time.Now().Unix() - worker.Status.LastProcessNotMatchtime)
		readyTimeout := worker.Spec.WorkerReadyTimeout - (time.Now().Unix() - worker.Status.LastWorkerNotReadytime)
		healthTimeout := int64(worker.Spec.MinReadySeconds) - (time.Now().Unix() - worker.Status.HealthCondition.LastTransitionTime.Unix())
		processReadyTimeout := worker.Spec.ProcessMatchTimeout - (time.Now().Unix() - worker.Status.LastProcessNotReadytime)

		if !worker.Status.ResourceMatch && worker.Spec.ResourceMatchTimeout > 0 {
			timeout = syncTimeout(timeout, matchTimeout)
		}
		if !worker.Status.ProcessMatch && worker.Spec.ProcessMatchTimeout > 0 {
			timeout = syncTimeout(timeout, processMatchTimeout)
		}
		if !worker.Status.WorkerReady && worker.Spec.WorkerReadyTimeout > 0 {
			timeout = syncTimeout(timeout, readyTimeout)
		}
		if !worker.Status.ProcessReady && worker.Spec.ProcessMatchTimeout > 0 {
			timeout = syncTimeout(timeout, processReadyTimeout)
		}
		if IsWorkerHealth(worker) && worker.Spec.MinReadySeconds > 0 {
			timeout = syncTimeout(timeout, healthTimeout)
		}
		if worker.Spec.WarmupSeconds > 0 && worker.Status.LastWarmupStartTime > 0 {
			warmupTimeout := int64(worker.Spec.WarmupSeconds) - (time.Now().Unix() - worker.Status.LastWarmupStartTime)
			timeout = syncTimeout(timeout, warmupTimeout)
		}
	}
	if IsWorkerUnAssigned(worker) {
		if NotScheduleTimeout > 0 && worker.Status.NotScheduleSeconds > 0 {
			notScheduleTimeout := int64(NotScheduleTimeout - worker.Status.NotScheduleSeconds)
			timeout = syncTimeout(timeout, notScheduleTimeout)
		}
	}
	return timeout
}

func syncTimeout(timeout, newtimeout int64) int64 {
	if newtimeout <= 0 && timeout <= 0 {
		return 0
	}
	if newtimeout <= 0 {
		return timeout
	}
	if timeout <= 0 {
		return newtimeout
	}
	if newtimeout < timeout {
		return newtimeout
	}
	return timeout
}

// IsWorkerEmpty judge is the worker is allocated
func IsWorkerEmpty(worker *WorkerNode) bool {
	if nil == worker {
		return false
	}
	return worker.Status.EntityName == ""
}

// IsWorkerEntityAlloced judge is the worker entity is allocated
func IsWorkerEntityAlloced(worker *WorkerNode) bool {
	if nil == worker {
		return false
	}
	return worker.Status.EntityName != "" && worker.Status.EntityAlloced == true && worker.Status.EntityUid != ""
}

// SetWorkerEntityAlloced judge is the worker is allocated
func SetWorkerEntityAlloced(worker *WorkerNode, alloced bool) {
	if nil == worker {
		return
	}
	worker.Status.EntityAlloced = alloced
}

// SetWorkerEntity judge is the worker is allocated
func SetWorkerEntity(worker *WorkerNode, name, uid string) {
	if nil == worker {
		return
	}
	if IsWorkerLost(worker) {
		return
	}
	if glog.V(4) {
		glog.Infof("set worker entity name %s,%s", worker.Name, name)
	}
	worker.Status.EntityName = name
	worker.Status.EntityUid = uid
}

// IsWorkerVersionMisMatch means cur version and target version is not same
func IsWorkerVersionMisMatch(worker *WorkerNode) bool {
	if nil == worker {
		return false
	}
	return worker.Spec.Version != worker.Status.Version && !IsWorkerInnerVerMatch(worker)
}

func IsWorkerRollback(w *WorkerNode) bool {
	if w.Spec.Version == "" || w.Status.Version == "" || w.Spec.Version == w.Status.Version {
		return false
	}
	specIndex := strings.Index(w.Spec.Version, "-")
	statusIndex := strings.Index(w.Status.Version, "-")
	if specIndex == -1 || statusIndex == -1 {
		return false
	}
	var err error
	var specVersion, statusVersion int
	if specVersion, err = strconv.Atoi(w.Spec.Version[0:specIndex]); err != nil {
		return false
	}
	if statusVersion, err = strconv.Atoi(w.Status.Version[0:statusIndex]); err != nil {
		return false
	}
	return specVersion < statusVersion
}

func IsWorkerInnerVerMatch(w *WorkerNode) bool {
	return IsVersionInnerMatch(w.Spec.Version, w.Status.Version)
}

func IsVersionInnerMatch(ver1, ver2 string) bool {
	if ver1 == "" || ver2 == "" || ver1 == ver2 {
		return false
	}
	ver1Index := strings.Index(ver1, "-")
	ver2Index := strings.Index(ver2, "-")
	if ver1Index == -1 || ver2Index == -1 {
		return false
	}
	return ver1[ver1Index:] == ver2[ver2Index:]
}

// IsWorkerResVersionMisMatch means cur resversion and target resversion is not same
func IsWorkerResVersionMisMatch(worker *WorkerNode) bool {
	if nil == worker {
		return false
	}
	return worker.Spec.ResVersion != worker.Status.ResVersion
}

// IsWorkerModeMisMatch means worker mode changed
func IsWorkerModeMisMatch(worker *WorkerNode) bool {
	if nil == worker {
		return false
	}
	if IsStandbyWorker(worker) && !IsHotStandbyWorker(worker) {
		return true
	}
	return GetWorkerTargetModeType(worker) != GetWorkerCurrentModeType(worker)
}

func GetSyncMetaPrefix() []string {
	var prefixes []string
	syncSubrsMetas := features.C2FeatureGate.Enabled(features.SyncSubrsMetas)
	if syncSubrsMetas {
		for i := range SubrsInheritanceMetaPrefix {
			prefixes = append(prefixes, SubrsInheritanceMetaPrefix[i])
		}
	}
	for i := range PodInheritWorkerMetaPrefix {
		prefixes = append(prefixes, PodInheritWorkerMetaPrefix[i])
	}
	// NOTE: some non-versioned keys are the same as subrsMetas
	keys, _ := GetNonVersionedKeys()
	for _, key := range keys.ToSlice() {
		prefixes = append(prefixes, key.(string))
	}
	return prefixes
}

func OverrideMap(base, prefer map[string]string) map[string]string {
	result := make(map[string]string, len(base)+len(prefer))
	for key, val := range base {
		result[key] = val
	}
	for key, val := range prefer {
		result[key] = val
	}
	return result
}

// GetWorkerBadState means is worker is in wrong status
func GetWorkerBadState(worker *WorkerNode, targetVersion string) BadReasonType {
	var reason BadReasonType
	if worker == nil {
		return reason
	}
	if broken := IsWorkerBroken(worker, targetVersion); broken != 0 && !IsInactiveWorker(worker) {
		glog.Infof("WorkerInBadState, broken: %s, reason: %d", worker.Name, broken)
		reason = broken
		return reason
	}
	if IsWorkerReclaiming(worker) {
		glog.Infof("WorkerInBadState, reclaiming: %s", worker.Name)
		reason = BadReasonReclaim
		return reason
	}
	if IsWorkerReleased(worker) {
		glog.Infof("WorkerInBadState, released: %s", worker.Name)
		reason = BadReasonRelease
		return reason
	}
	return reason
}

// IsVersionMatch the worker is the current version
func IsVersionMatch(worker *WorkerNode) bool {
	if nil == worker {
		return false
	}
	return worker.Spec.Version == worker.Status.Version
}

// IsResVersionMatch the worker is the current version
func IsResVersionMatch(worker *WorkerNode) bool {
	if nil == worker {
		return false
	}
	return worker.Spec.ResVersion == worker.Status.ResVersion
}

// IsWorkerUnAvaliable means is service status is UNAVAILABLE
func IsWorkerUnAvaliable(worker *WorkerNode) bool {
	if nil == worker {
		return false
	}
	return worker.Status.ServiceStatus == ServiceUnAvailable
}

// CouldDeletePod is pod could delete
func CouldDeletePod(worker *WorkerNode) bool {
	if nil == worker {
		return false
	}
	return IsWorkerUnAvaliable(worker) || // unavailable 已经摘掉流量的
		(!IsWorkerGracefully(worker) && IsWorkerToRelease(worker))
}

// IsWorkerServiceMatch means is service status match plan
func IsWorkerServiceMatch(worker *WorkerNode) bool {
	if nil == worker {
		return false
	}
	if !IsWorkerOnline(worker) {
		return worker.Status.ServiceStatus == ServiceUnAvailable
	}
	return IsWorkerLatestVersionAvailable(worker)
}

// IsWorkerOnline means is worker online field is nil or true
func IsWorkerOnline(worker *WorkerNode) bool {
	return nil == worker.Spec.Online || true == *worker.Spec.Online
}

// IsWorkerGracefully means is worker UpdatingGracefully field is nil or true
func IsWorkerGracefully(worker *WorkerNode) bool {
	return worker.Spec.UpdatingGracefully == nil || *worker.Spec.UpdatingGracefully == true
}

// IsRollingSetGracefully IsRollingSetGracefully
func IsRollingSetGracefully(rs *RollingSet) bool {
	return rs.Spec.UpdatingGracefully == nil || *rs.Spec.UpdatingGracefully == true
}

// IsWorkerMatch means is service status is AVAILABLE
func IsWorkerMatch(worker *WorkerNode) bool {
	if !IsWorkerUpdateMatch(worker) {
		return false
	}
	return IsWorkerServiceMatch(worker)
}

// IsWorkerTypeReady  心跳业务服务正常
func IsWorkerTypeReady(worker *WorkerNode) bool {
	if nil == worker {
		return false
	}
	if worker.Status.HealthCondition.WorkerStatus == "" {
		return true
	}
	if worker.Status.HealthCondition.WorkerStatus == WorkerTypeReady {
		return true
	}
	return false
}

// IsWorkerComplete means is a worker status is ok completely
func IsWorkerComplete(worker *WorkerNode) bool {
	if nil == worker {
		return false
	}
	if !IsResVersionMatch(worker) {
		return false
	}
	if !worker.Status.ResourceMatch {
		return false
	}
	if NeedReclaimByService(worker) {
		return false
	}
	if IsWorkerReclaiming(worker) {
		return false
	}
	if IsStandByWorkerComplete(worker) {
		return true
	}
	return IsWorkerMatch(worker)
}

// IsStandByWorkerComplete IsStandByWorkerComplete
func IsStandByWorkerComplete(worker *WorkerNode) bool {
	if !IsStandbyWorker(worker) {
		return false
	}
	switch worker.Spec.WorkerMode {
	case WorkerModeTypeCold:
		if IsWorkerAssigned(worker) {
			return true
		}
	case WorkerModeTypeWarm:
		if IsWorkerAssigned(worker) &&
			IsWorkerRunning(worker) &&
			IsWorkerReady(worker) {
			return true
		}
	case WorkerModeTypeHot:
		if IsWorkerAssigned(worker) &&
			IsWorkerRunning(worker) &&
			IsWorkerReady(worker) &&
			IsWorkerProcessReady(worker) {
			return true
		}
	case WorkerModeTypeInactive:
		if IsWorkerAssigned(worker) {
			return true
		}
	}
	return false
}

// AllocStatusScore compute the score of AllocStatus
func AllocStatusScore(status AllocStatus) int32 {
	switch status {
	case WorkerLost:
		return 0
	case WorkerReleased:
		return 1
	case WorkerOfflining:
		return 2
	case WorkerReleasing:
		return 3
	case WorkerUnAssigned:
		return 4
	case WorkerAssigned:
		return 5
	default:
		return 0
	}
}

// PhaseScore compute the score of WorkerPhase
func PhaseScore(status WorkerPhase) int32 {
	switch status {
	case Terminated:
		return 0
	case Failed:
		return 1
	case Unknown:
		return 2
	case Pending:
		return 3
	case Running:
		return 4
	default:
		return 0
	}
}

// ServiceScore ServiceScore
func ServiceScore(worker *WorkerNode) int32 {
	baseScore := ServiceStatusScore(worker.Status.ServiceStatus)
	for i := range worker.Status.ServiceConditions {
		baseScore += int32(worker.Status.ServiceConditions[i].Score)
	}
	return baseScore
}

// ServiceStatusScore compute the score of ServiceStatus
func ServiceStatusScore(status ServiceStatus) int32 {
	switch status {
	case ServiceUnAvailable:
		return 0
	case ServiceUnKnown:
		return 1
	case ServicePartAvailable:
		return 2
	case ServiceAvailable:
		return 3
	default:
		return 0
	}
}

// HealthScore compute the score of HealthStatus
func HealthScore(status HealthStatus) int32 {
	switch status {
	case HealthDead:
		return 0
	case HealthUnKnown:
		return 1
	case HealthLost:
		return 2
	case HealthAlive:
		return 3
	default:
		return 0
	}
}

// WorkerReadyScore compute the score of Worker Ready
func WorkerReadyScore(worker *WorkerNode) int32 {
	if IsWorkerTypeReady(worker) {
		return 1
	}
	return 0
}

// WorkerReleaseScore compute the score of Worker Release
func WorkerReleaseScore(worker *WorkerNode) int32 {
	if !IsWorkerToRelease(worker) {
		return 1
	}
	return 0
}

// WorkerReclaimScore compute the score of Worker reclaim
func WorkerReclaimScore(worker *WorkerNode) int32 {
	if !IsWorkerReclaiming(worker) {
		return 1
	}
	return 0
}

// WorkerAvailableScore compute the score of Worker available
func WorkerAvailableScore(worker *WorkerNode) int32 {
	if IsWorkerLatestVersionAvailable(worker) {
		return 1
	}
	return 0
}

// WorkerProcessScore compute the score of Worker process
func WorkerProcessScore(worker *WorkerNode) int32 {
	processScore := worker.Status.ProcessScore
	return processScore
}

// WorkerCompleteScore compute the score of Worker complete
func WorkerCompleteScore(worker *WorkerNode) int32 {
	complete := IsWorkerComplete(worker)
	if complete {
		return 1
	}
	return 0
}

func WorkerResourcePoolScore(worker *WorkerNode) int32 {
	return 0
}

// WorkerResourceScore compute the score of Worker resource match
func WorkerResourceScore(worker *WorkerNode) int32 {
	resourceMatch := worker.Status.ResourceMatch
	if resourceMatch {
		return 1
	}
	return 0
}

// WorkerModeScore compute the score of Worker mode
func WorkerModeScore(worker *WorkerNode) int32 {
	if !IsStandbyWorker(worker) {
		return 5
	}
	if InSilence(worker) {
		return 4
	}
	if !IsWorkerOnStandByResourcePool(worker) {
		return 3
	}
	if worker.Spec.WorkerMode == WorkerModeTypeHot {
		return 2
	}
	if worker.Spec.WorkerMode == WorkerModeTypeWarm {
		return 1
	}
	return 0
}

// DaemonSetHostScore compute the score of host for daemonset
func DaemonSetHostScore(worker *WorkerNode) int32 {
	if !IsDaemonSetWorker(worker) {
		return 0
	}
	// 未分配，认为无法分配，最后更新
	if IsWorkerUnAssigned(worker) {
		return 9
	}
	// 宿主机挂了
	if worker.Status.ProcessScore == HostDead {
		return 8
	}
	// 宿主机未启动pod
	if worker.Status.PodNotStarted {
		return 7
	}
	return 0
}

// WorkerServiceReadyScore WorkerServiceReadyScore
func WorkerServiceReadyScore(worker *WorkerNode) int32 {
	if _, ok := IsWorkerServiceReady(worker); ok {
		return 1
	}
	return 0
}

// DaemonsetScoreOfWorker compute the score of a worker for daemonset
func DaemonsetScoreOfWorker(worker *WorkerNode) int64 {
	if nil == worker {
		return 0
	}
	return AggregateFraction(
		WorkerReleaseScore(worker),
		DaemonSetHostScore(worker),
		WorkerAvailableScore(worker),
		ServiceScore(worker),
		WorkerReadyScore(worker),
		HealthScore(worker.Status.HealthStatus),
		PhaseScore(worker.Status.Phase),
		AllocStatusScore(worker.Status.AllocStatus),
		WorkerReclaimScore(worker),
		WorkerProcessScore(worker),
		WorkerModeScore(worker),
		WorkerCompleteScore(worker),
		WorkerResourcePoolScore(worker),
	)
}

// BaseScoreOfWorker compute the score of a worker
func BaseScoreOfWorker(worker *WorkerNode) int64 {
	if nil == worker {
		return 0
	}
	fractions := make([]int32, 0)
	fractions = append(fractions, WorkerReleaseScore(worker))
	if raiseServicePriority(worker) {
		fractions = append(fractions,
			ServiceScore(worker),
			WorkerServiceReadyScore(worker),
			WorkerAvailableScore(worker))
	} else {
		fractions = append(fractions,
			WorkerServiceReadyScore(worker),
			WorkerAvailableScore(worker),
			ServiceScore(worker))
	}
	fractions = append(fractions, WorkerReadyScore(worker),
		HealthScore(worker.Status.HealthStatus),
		PhaseScore(worker.Status.Phase),
		AllocStatusScore(worker.Status.AllocStatus),
		WorkerReclaimScore(worker),
		WorkerProcessScore(worker),
		WorkerModeScore(worker),
		WorkerCompleteScore(worker),
		WorkerResourcePoolScore(worker))
	return AggregateFraction(fractions...)
}

// ScoreOfWorker  compute the score of a worker
func ScoreOfWorker(worker *WorkerNode) int64 {
	var score int64
	if IsDaemonSetWorker(worker) {
		score = DaemonsetScoreOfWorker(worker)
	} else {
		score = BaseScoreOfWorker(worker)
	}
	return score
}

// AggregateFraction AggregateFraction
func AggregateFraction(fractions ...int32) (score int64) {
	for i := range fractions {
		score = score*10 + int64(fractions[i])
	}
	return
}

// IsDaemonSetWorker is this worker for daemonset
func IsDaemonSetWorker(worker *WorkerNode) bool {
	return worker.Spec.IsDaemonSet
}

func raiseServicePriority(worker *WorkerNode) bool {
	for i := range worker.Status.ServiceConditions {
		if worker.Status.ServiceConditions[i].Type == ServiceGeneral {
			return true
		}
	}
	return false
}

// IsWorkerNotStarted just is worker not started
func IsWorkerNotStarted(worker *WorkerNode) bool {
	return IsWorkerAssigned(worker) && (worker.Status.Phase == Unknown || worker.Status.Phase == Pending)
}

// Hippo Health Scores
const (
	HostDead = 2
)

// IsCurrentWorker judge is the worker is current worker preliminarily
func IsCurrentWorker(worker *WorkerNode) bool {
	if nil == worker {
		return false
	}
	return worker.Labels[WorkerRoleKey] == CurrentWorkerKey
}

// IsBackupWorker judge is the worker is current worker preliminarily
func IsBackupWorker(worker *WorkerNode) bool {
	if nil == worker {
		return false
	}
	return worker.Labels[WorkerRoleKey] == BackupWorkerKey
}

// GetWorkerReplicaName gets replica name for a worker node
func GetWorkerReplicaName(w *WorkerNode) string {
	if len(w.Name) < 2 { // expect empty
		return w.Name
	}
	return w.Name[:len(w.Name)-2]
}

// AppendWorkerCurrentLabel appends current worker label, used to select only current worker nodes.
func AppendWorkerCurrentLabel(lbls labels.Set) labels.Set {
	lbls[WorkerRoleKey] = CurrentWorkerKey
	return lbls
}

func getWorkerCopy(worker *WorkerNode, deepcopy bool) *WorkerNode {
	if deepcopy {
		return worker.DeepCopy()
	}
	return worker
}

// FixWorkerRole FixWorkerRole
func FixWorkerRole(current, backup *WorkerNode, deepcopy bool) (newCurrent *WorkerNode, newBackup *WorkerNode) {
	newCurrent = current
	newBackup = backup
	if nil != current && !IsCurrentWorker(current) || current.Status.BecomeCurrentTime.IsZero() {
		newCurrent = getWorkerCopy(current, deepcopy)
		SetWorkerCurrent(newCurrent)
	}
	if nil != backup && !IsBackupWorker(backup) {
		newBackup = getWorkerCopy(backup, deepcopy)
		SetWorkerBackup(newBackup)
	}
	return newCurrent, newBackup
}

// GetCurrentWorkerNodeV2 getter
func GetCurrentWorkerNodeV2(workers []*WorkerNode, deepcopy bool) (*WorkerNode, *WorkerNode, bool) {
	var current, backup *WorkerNode
	var toRelease bool
	for i := range workers {
		if nil == workers[i] {
			continue
		}
		if IsBackupWorker(workers[i]) {
			if nil == backup {
				backup = workers[i]
			} else {
				current = workers[i]
			}
		} else {
			if IsWorkerToDelete(workers[i]) {
				toRelease = true
			}
			if current == nil {
				current = workers[i]
			} else {
				// 由于不同步出现两个current，最近一个更新的是真current
				FixWorkerBecomeCurrentTime(current)
				FixWorkerBecomeCurrentTime(workers[i])
				if current.Status.BecomeCurrentTime.Unix() >
					workers[i].Status.BecomeCurrentTime.Unix() {
					backup = workers[i]
					//SetWorkerBackup(backup)
				} else {
					backup = current
					current = workers[i]
					//SetWorkerBackup(backup)
				}
			}
		}
	}
	if nil == current && nil != backup {
		current = backup
		backup = nil
	}
	newCurrent, newBackup := FixWorkerRole(current, backup, deepcopy)
	return newCurrent, newBackup, toRelease
}

func FixWorkerBecomeCurrentTime(current *WorkerNode) {
	if current.Status.BecomeCurrentTime.IsZero() {
		if timeStr, ok := current.Annotations[AnnoBecomeCurrentTime]; ok {
			t, _ := time.Parse(time.RFC3339, timeStr)
			current.Status.BecomeCurrentTime = metav1.NewTime(t)
		}
	}
}

// SwapAndReleaseBackup swap current and backup , release backup
func SwapAndReleaseBackup(current, backup *WorkerNode) error {
	SetWorkerBackup(current)
	SetWorkerCurrent(backup)
	return nil
}

// SetWorkerCurrent SetWorkerCurrent
func SetWorkerCurrent(worker *WorkerNode) {
	if nil == worker {
		return
	}
	if worker.Labels == nil {
		worker.Labels = map[string]string{}
	}
	worker.Labels[WorkerRoleKey] = CurrentWorkerKey
	worker.Status.BecomeCurrentTime = metav1.NewTime(time.Now())
	SetBadReason(worker, BadReasonNone)
}

// SetWorkerBackup SetWorkerBackup
func SetWorkerBackup(worker *WorkerNode) {
	if nil == worker {
		return
	}
	if worker.Labels == nil {
		worker.Labels = map[string]string{}
	}
	worker.Labels[WorkerRoleKey] = BackupWorkerKey
}

// ReleaseWorker ReleaseWorker
func ReleaseWorker(worker *WorkerNode, prohibit bool) {
	worker.Status.ToRelease = true
	worker.Status.ServiceOffline = true

	SetWorkerProhibit(worker, prohibit)
}

// SetWorkerProhibit SetWorkerProhibit
func SetWorkerProhibit(worker *WorkerNode, prohibit bool) {
	// 首先使用外部配置prefer
	if _, ok := worker.Labels[LabelKeyHippoPreference]; ok {
		return
	}
	if IsWorkerBroken(worker, worker.Spec.Version) != BadReasonNone || worker.Status.Reclaim {
		// 没有外部配置prefer时，使用默认prefer
		preValue := generatePreference(prohibit, defaultPreTTL, preScopeAPP)
		worker.Labels = labelsutil.CloneAndAddLabel(worker.Labels, LabelKeyHippoPreference, preValue)
	}
}

const (
	preKeyProhibit = "PROHIBIT"
	preKeyPrefer   = "PREFER"
	defaultPreTTL  = 600
	preScopeRole   = "ROLE"
	preScopeAPP    = "APP"
)

func generatePreference(prohibit bool, ttl int64, scope string) string {
	if prohibit {
		return fmt.Sprintf("%s-%s-%d", scope, preKeyProhibit, ttl)
	}
	return fmt.Sprintf("%s-%s-%d", scope, preKeyPrefer, ttl)
}

// IsReplicaUnAssigned get current state
func IsReplicaUnAssigned(replica *Replica) bool {
	if nil == replica {
		return false
	}
	return replica.Status.AllocStatus == WorkerUnAssigned
}

// IsReplicaEmpty get current state
func IsReplicaEmpty(replica *Replica) bool {
	return IsReplicaUnAssigned(replica) || IsReplicaNotInited(replica) || IsReplicaLost(replica)
}

// IsReplicaNotInited IsReplicaNotInited
func IsReplicaNotInited(replica *Replica) bool {
	if nil == replica {
		return false
	}
	return replica.Status.AllocStatus == ""
}

// IsReplicaAssigned get current state
func IsReplicaAssigned(replica *Replica) bool {
	if nil == replica {
		return false
	}
	return replica.Status.AllocStatus == WorkerAssigned
}

// IsReplicaLost get current state
func IsReplicaLost(replica *Replica) bool {
	if nil == replica {
		return false
	}
	return replica.Status.AllocStatus == WorkerLost
}

// IsReplicaOfflining get current state
func IsReplicaOfflining(replica *Replica) bool {
	if nil == replica {
		return false
	}
	return replica.Status.AllocStatus == WorkerOfflining
}

// IsReplicaReleasing get current state
func IsReplicaReleasing(replica *Replica) bool {
	if nil == replica {
		return false
	}
	if len(replica.Gang) != 0 {
		return IsWorkerToDelete(&replica.WorkerNode)
	}
	return IsWorkerToRelease(&replica.WorkerNode)
}

// IsReplicaAllocReleasing get current state
func IsReplicaAllocReleasing(replica *Replica) bool {
	if nil == replica {
		return false
	}
	return replica.Status.AllocStatus == WorkerReleasing
}

// IsReplicaReleased get current state
func IsReplicaReleased(replica *Replica) bool {
	if nil == replica {
		return false
	}
	return replica.Status.AllocStatus == WorkerReleased
}

// IsReplicaComplete get current state
func IsReplicaComplete(replica *Replica) bool {
	if nil == replica {
		return false
	}
	if IsReplicaReleasing(replica) {
		return false
	}
	if !IsReplicaAvailable(replica) {
		return false
	}
	if replica.Spec.Version != replica.Status.Version && !IsWorkerInnerVerMatch(&replica.WorkerNode) {
		return false
	}
	if isBackupReleaseWithGeneralService(replica) {
		return false
	}
	return replica.Status.Complete
}

func isBackupReleaseWithGeneralService(replica *Replica) bool {
	if replica == nil {
		return false
	}
	if replica.Backup.UID == "" {
		return false
	}
	if !IsWorkerToRelease(&replica.Backup) {
		return false
	}
	var conditions []ServiceCondition
	conditions = append(conditions, replica.Status.ServiceConditions...)
	conditions = append(conditions, replica.Backup.Status.ServiceConditions...)
	for i := range conditions {
		if conditions[i].Type == ServiceGeneral {
			return true
		}
	}
	return false
}

// IsReplicaServiceReady IsReplicaServiceReady
func IsReplicaServiceReady(replica *Replica) (int64, bool) {
	timeout, ok := IsWorkerServiceReady(&replica.WorkerNode)
	return timeout, ok && !isBackupReleaseWithGeneralService(replica)
}

// IsStandbyReplica is standby
func IsStandbyReplica(replica *Replica) bool {
	if nil == replica {
		return false
	}
	return IsStandbyWorker(&replica.WorkerNode)
}

// ReplicaWorkModeScore compute the score of workModeType
func ReplicaWorkModeScore(replica *Replica) int32 {
	if !IsStandbyReplica(replica) {
		return 0
	}
	if InSilence(&replica.WorkerNode) {
		return 1
	}
	switch replica.Spec.WorkerMode {
	case WorkerModeTypeUnknown:
		return 9
	case WorkerModeTypeCold:
		return 4
	case WorkerModeTypeWarm:
		return 3
	case WorkerModeTypeHot:
		return 2
	}
	return 9
}

func ReplicaResourcePoolScore(replica *Replica) int32 {
	if nil == replica {
		return 0
	}
	return WorkerResourcePoolScore(&replica.WorkerNode)
}

// GetReplicaStandbyHours get standby hours
func GetReplicaStandbyHours(replica *Replica) []int64 {
	if nil == replica {
		return nil
	}
	return GetWorkerStandbyHours(&replica.WorkerNode)
}

// IsReplicaOnStandByResourcePool IsReplicaOnStandByResourcePool
func IsReplicaOnStandByResourcePool(replica *Replica) bool {
	if nil == replica {
		return false
	}
	return IsWorkerOnStandByResourcePool(&replica.WorkerNode)
}

// IsReplicaDependencyReady IsReplicaDependencyReady
func IsReplicaDependencyReady(replica *Replica) bool {
	if nil == replica {
		return false
	}
	return IsWorkerDependencyReady(&replica.WorkerNode)
}

// IsWorkerServiceReady IsWorkerServiceReady
func IsWorkerServiceReady(worker *WorkerNode) (int64, bool) {
	if nil == worker {
		return 0, false
	}
	if IsWorkerToRelease(worker) {
		return 0, false
	}
	if worker.Spec.Version != worker.Status.Version && !IsWorkerInnerVerMatch(worker) {
		return 0, false
	}
	worker.Status.ServiceReady = IsWorkerLatestVersionAvailable(worker) && IsWorkerHealth(worker) &&
		IsWorkerHealthInfoReady(worker) && IsWorkerProcessReady(worker)
	if !worker.Status.ServiceReady {
		worker.Status.LastServiceReadyTime = utils.Int64Ptr(0)
	} else if worker.Status.LastServiceReadyTime == nil {
		minReadySeconds := int64(getMinReadySeconds(worker))
		worker.Status.LastServiceReadyTime = utils.Int64Ptr(time.Now().Unix() - minReadySeconds - 1)
	} else if *worker.Status.LastServiceReadyTime == 0 {
		worker.Status.LastServiceReadyTime = utils.Int64Ptr(time.Now().Unix())
	}

	return IsServiceReadyForMinTime(worker)
}

// GetWorkerSilenceTimeout GetWorkerSilenceTimeout
func GetWorkerSilenceTimeout(worker *WorkerNode) int64 {
	if IsSilenceNode(worker) {
		if worker == nil || worker.Annotations == nil {
			return 0
		}

		startTime, err := strconv.ParseInt(worker.Annotations[SilentTimeAnnoKey], 10, 64)
		if err != nil {
			return 0
		}

		timeout := int64(defaultSilenceSeconds) - (time.Now().Unix() - startTime)
		if timeout < 0 {
			return 0
		}
		return timeout
	}
	return 0
}

func getMinReadySeconds(worker *WorkerNode) int32 {
	if 0 == worker.Spec.MinReadySeconds {
		return int32(defaultMinReadySeconds)
	}
	return worker.Spec.MinReadySeconds
}

// IsServiceReadyForMinTime IsServiceReadyForMinTime
func IsServiceReadyForMinTime(worker *WorkerNode) (int64, bool) {
	ready := worker.Status.ServiceReady
	readySeconds := getMinReadySeconds(worker)
	if glog.V(4) {
		glog.Infof("[%s] getMinReadySeconds %v,%d", worker.Name, ready, readySeconds)
	}
	if !ready || 0 == readySeconds {
		return 0, ready
	}
	var lastServiceReadyTime int64
	if nil == worker.Status.LastServiceReadyTime {
		lastServiceReadyTime = time.Now().Unix() - int64(readySeconds) - 1
	} else {
		lastServiceReadyTime = *worker.Status.LastServiceReadyTime
	}
	return int64(readySeconds+10) - (time.Now().Unix() - lastServiceReadyTime), time.Now().Unix()-lastServiceReadyTime > int64(readySeconds)
}

// IsReplicaProcessReady if service process ready
func IsReplicaProcessReady(replica *Replica) bool {
	return replica.Status.WorkerReady
}

// IsReplicaAvailable means is the replica service is avaiable
func IsReplicaAvailable(replica *Replica) bool {
	if nil == replica {
		return false
	}
	return IsWorkerLatestVersionAvailable(&replica.WorkerNode)
}

// IsWorkerLatestVersionAvailable means is the worker service is avaiable
func IsWorkerLatestVersionAvailable(worker *WorkerNode) bool {
	if nil == worker {
		return false
	}
	for i := range worker.Status.ServiceConditions {
		if isGracefullyServiceType(worker.Status.ServiceConditions[i].Type) &&
			worker.Status.ServiceConditions[i].Version != worker.Spec.Version &&
			worker.Status.ServiceConditions[i].Version != "" {
			return false
		}
	}
	return worker.Status.ServiceStatus == ServiceAvailable
}

// IsReplicaHealth means is the replica service is avaiable
func IsReplicaHealth(replica *Replica) bool {
	if nil == replica {
		return false
	}
	return replica.Status.HealthStatus == HealthAlive
}

// IsAllWorkersReleased means if all workers of this replica is released
func IsAllWorkersReleased(workers []*WorkerNode) bool {
	if nil == workers || 0 == len(workers) {
		return true
	}
	for i := range workers {
		if !IsWorkerReleased(workers[i]) {
			return false
		}
	}
	return true
}

// SetReplicaAnnotation set Annotation
func SetReplicaAnnotation(replica *Replica, key string, value string) {
	if nil == replica {
		return
	}
	if replica.Annotations == nil {
		replica.Annotations = make(map[string]string, 0)
	}
	replica.Annotations[key] = value
}

// RemoveFinalizer 移除Finalizer
func RemoveFinalizer(rollingset *RollingSet, finalizer string) {
	rollingset.Finalizers = common.DeleteFromSlice(finalizer, rollingset.GetFinalizers())
}

// AddFinalizer 添加Finalizer
func AddFinalizer(rollingset *RollingSet, finalizer string) {
	for _, element := range rollingset.Finalizers {
		if element == finalizer {
			return
		}
	}
	rollingset.Finalizers = append(rollingset.Finalizers, finalizer)
}

// GetHippoPodTemplate get hippo template from raw template
func GetHippoPodTemplate(version *VersionPlan) (*HippoPodTemplate, error) {
	return version.Template.DeepCopy(), nil
}

// GetGeneralTemplate get general template from raw template
func GetGeneralTemplate(version VersionPlan) (*GeneralTemplate, error) {
	var general GeneralTemplate
	var meta *metav1.ObjectMeta
	if version.Template != nil {
		meta = version.Template.ObjectMeta.DeepCopy()
	}
	if nil != meta {
		general.ObjectMeta = *meta
	}
	return &general, nil
}

// GetTemplateLabelSet copy labels
func GetTemplateLabelSet(template *GeneralTemplate) labels.Set {
	desiredLabels := make(labels.Set)
	for k, v := range template.Labels {
		desiredLabels[k] = v
	}
	return desiredLabels
}

// GetTemplateFinalizers copy finalizers
func GetTemplateFinalizers(template *GeneralTemplate) []string {
	desiredFinalizers := make([]string, len(template.Finalizers))
	copy(desiredFinalizers, template.Finalizers)
	return desiredFinalizers
}

// GetTemplateAnnotationSet copy annotations
func GetTemplateAnnotationSet(template *GeneralTemplate) labels.Set {
	desiredAnnotations := make(labels.Set)
	for k, v := range template.Annotations {
		desiredAnnotations[k] = v
	}
	return desiredAnnotations
}

// EqualIgnoreHash returns true if two given podTemplateSpec are equal, ignoring the diff in value of Labels[pod-template-hash]
// We ignore pod-template-hash because:
//  1. The hash result would be different upon podTemplateSpec API changes
//     (e.g. the addition of a new field will cause the hash code to change)
//  2. The deployment template won't have hash labels
func EqualIgnoreHash(plan1, plan2 *VersionPlan) bool {
	p1Copy := plan1.DeepCopy()
	p2Copy := plan2.DeepCopy()
	// Remove hash labels from template.Labels before comparing
	// delete(p1Copy.Labels, apps.DefaultDeploymentUniqueLabelKey)
	// delete(p2Copy.Labels, apps.DefaultDeploymentUniqueLabelKey)
	return apiequality.Semantic.DeepEqual(p1Copy, p2Copy)
}

// IsPodPriorityOnline online pod
func IsPodPriorityOnline(priority int32) bool {
	if priority >= 100 && priority < 200 {
		return true
	}
	return false
}

// IsPodPriorityMid mid pod
func IsPodPriorityMid(priority int32) bool {
	if priority >= 72 && priority < 96 {
		return true
	}
	return false
}

// IsPodPriorityOffline offline pod
func IsPodPriorityOffline(priority int32) bool {
	if priority >= 0 && priority < 100 {
		return true
	}
	return false
}

// IsPodPrioritySys sys pod
func IsPodPrioritySys(priority int32) bool {
	if priority >= 200 {
		return true
	}
	return false
}

// IsWorkerServiceOffline if offline service
func IsWorkerServiceOffline(worker *WorkerNode) bool {
	if worker == nil {
		return false
	}
	return worker.Status.ServiceOffline
}

// IsWorkerProcessReady if service process ready
func IsWorkerProcessReady(worker *WorkerNode) bool {
	return worker.Status.WorkerReady
}

// InitWorkerNode 初始化默认值
func InitWorkerNode(worker *WorkerNode) {
	if "" == worker.Status.AllocStatus {
		worker.Status.AllocStatus = WorkerUnAssigned
	}
	if "" == worker.Status.HealthStatus {
		worker.Status.HealthStatus = HealthUnKnown
	}
	if "" == worker.Status.ServiceStatus {
		worker.Status.ServiceStatus = ServiceUnAvailable
	}
	if "" == worker.Status.Phase {
		worker.Status.Phase = Unknown
	}
	worker.Status.Name = worker.Name
}

// GetReplicaReferWorkerNums 获取replica上refer worker数量
func GetReplicaReferWorkerNums(replica *Replica) int {
	var refers int
	for k := range replica.Labels {
		if strings.HasPrefix(k, WorkerReferReplicaKey) {
			refers++
		}
	}
	return refers
}

// IsReplicaNoRefers 判断是否没有引用
func IsReplicaNoRefers(replica *Replica) bool {
	return GetReplicaReferWorkerNums(replica) == 0
}

func ExceedRestartLimit(restartRecord string, limitCnt int) bool {
	if restartRecord == "" {
		return false
	}
	records := strings.Split(restartRecord, ",")
	if len(records) < limitCnt {
		return false
	}
	return records[len(records)-limitCnt] >= fmt.Sprintf("%d", time.Now().Unix()-(1<<limitCnt)*ExponentialBackoff)
}

func AddRestartRecord(pod *corev1.Pod, worker *WorkerNode) {
	for i := range pod.Status.ContainerStatuses {
		name := pod.Status.ContainerStatuses[i].Name
		limit := DefaultRestartCountLimit
		if configs := GetContainersConfigs(name, worker); configs != nil && configs.RestartCountLimit > 0 {
			limit = configs.RestartCountLimit
		}
		state := pod.Status.ContainerStatuses[i].State
		if state.Terminated != nil && state.Terminated.ExitCode != 0 {
			record := fmt.Sprintf("%d", state.Terminated.FinishedAt.Unix())
			if worker.Status.RestartRecords == nil {
				worker.Status.RestartRecords = make(map[string]string)
			}
			var records []string
			if worker.Status.RestartRecords[name] != "" {
				records = strings.Split(worker.Status.RestartRecords[name], ",")
			}
			newRecord := true
			for j := len(records) - 1; j >= 0; j-- {
				if records[j] == record {
					newRecord = false
					break
				}
			}
			if newRecord {
				records = append(records, record)
				if len(records) > limit {
					records = records[len(records)-limit:]
				}
				worker.Status.RestartRecords[name] = strings.Join(records, ",")
			}
		}
	}
}

func GetContainersConfigs(name string, worker *WorkerNode) *ContainerConfig {
	for i := range worker.Spec.Template.Spec.Containers {
		container := worker.Spec.Template.Spec.Containers[i]
		if name == container.Name {
			return container.Configs
		}
	}
	return nil
}

// TransPodPhase trans pod phase to worker phase
func TransPodPhase(pod *corev1.Pod, processMatch bool) WorkerPhase {
	if nil == pod {
		glog.Errorf("unkonwn pod")
		return Unknown
	}
	switch pod.Status.Phase {
	case corev1.PodPending:
		return Pending
	case corev1.PodRunning:
		if terminated, failed := IsPodHasTerminatedContainer(pod); terminated {
			if !processMatch {
				return Pending
			} else if !IsBizPod(&pod.ObjectMeta) && IsAsiPod(&pod.ObjectMeta) && failed {
				return Running
			} else if failed {
				return Failed
			}
			return Terminated
		}
		if IsPodHasWaitingContainer(pod) {
			return Pending
		}
		return Running
	case corev1.PodSucceeded:
		if !processMatch {
			return Pending
		}
		return Terminated
	case corev1.PodFailed:
		return Failed
	case corev1.PodUnknown:
		return Unknown
	}
	glog.Errorf("unkonwn pod phase :%s,%s", pod.Name, pod.Status.Phase)
	return Unknown
}

// IsPodHasTerminatedContainer if a container of pod is terminated return true
func IsPodHasTerminatedContainer(pod *corev1.Pod) (bool, bool) {
	if nil == pod {
		return false, false
	}
	for i := range pod.Status.ContainerStatuses {
		if nil != pod.Status.ContainerStatuses[i].State.Terminated {
			return true, pod.Status.ContainerStatuses[i].State.Terminated.ExitCode != 0
		}
	}
	return false, false
}

// IsPodHasWaitingContainer if a container of pod is waiting return true
func IsPodHasWaitingContainer(pod *corev1.Pod) bool {
	if nil == pod {
		return false
	}
	for i := range pod.Status.ContainerStatuses {
		if nil != pod.Status.ContainerStatuses[i].State.Waiting {
			return true
		}
	}
	return false
}

// GetPodSpecFromTemplate GetPodSpecFromTemplate
func GetPodSpecFromTemplate(hippoPodTemplate *HippoPodTemplate) (*HippoPodSpec, map[string]string, error) {
	if hippoPodTemplate == nil {
		return nil, nil, nil
	}
	var hippoPodSpec = hippoPodTemplate.Spec.DeepCopy()
	return hippoPodSpec, hippoPodTemplate.Labels, nil
}

// IsDaemonWorker IsDaemonWorker
func IsDaemonWorker(worker *WorkerNode) bool {
	if worker == nil || worker.Spec.Template == nil {
		return false
	}
	return isDaemonPlan(worker.Spec.Template)
}

func isDaemonPlan(hippoPodTemplate *HippoPodTemplate) bool {
	if hippoPodTemplate == nil {
		return false
	}
	return IsDaemonPod(hippoPodTemplate.Spec.PodSpec)
}

// IsDaemonPod IsDaemonPod
func IsDaemonPod(podSpec corev1.PodSpec) bool {
	return podSpec.RestartPolicy == corev1.RestartPolicyAlways
}

// NeedRestartAfterResourceChange NeedRestartAfterResourceChange
func NeedRestartAfterResourceChange(versionPlan *VersionPlan) bool {
	if nil == versionPlan.RestartAfterResourceChange {
		return false
	}
	return *versionPlan.RestartAfterResourceChange
}

// NeedRollingAfterResourceChange NeedRollingAfterResourceChange
func NeedRollingAfterResourceChange(versionPlan *VersionPlan) bool {
	return features.C2MutableFeatureGate.Enabled(features.UpdateResourceRolling) || NeedRestartAfterResourceChange(versionPlan)
}

func filterNonVersioned(src map[string]string, keys mapset.Set) map[string]string {
	if src == nil {
		return nil
	}
	target := make(map[string]string)
	for k := range src {
		filtered := false
		if keys != nil && keys.Contains(k) {
			filtered = true
		}
		if strings.HasPrefix(k, BizDetailKeyPrefix) {
			filtered = true
		}
		if !filtered {
			target[k] = src[k]
		}
	}
	return target
}

// SignGangVersionPlan sign gang version plan
func SignGangVersionPlan(versionPlans map[string]*VersionPlan, labels map[string]string, name string) (string, string, error) {
	var versions = []string{}
	var keys = []string{}
	for k := range versionPlans {
		keys = append(keys, k)
	}
	sort.Sort(sort.StringSlice(keys))
	for _, k := range keys {
		versionPlan := versionPlans[k]
		version, _, err := SignVersionPlan(versionPlan, labels, name)
		if err != nil {
			return "", "", err
		}
		versions = append(versions, version)
	}
	version, _ := utils.SignatureWithMD5(versions)
	return version, version, nil
}

// SignVersionPlan sign version plan
func SignVersionPlan(versionPlan *VersionPlan, labels map[string]string, name string) (string, string, error) {
	versionPlan = versionPlan.DeepCopy()
	nonVersionedKeys, err := GetNonVersionedKeys()
	if err != nil {
		return "", "", err
	}
	versionPlan.Template.Labels = filterNonVersioned(versionPlan.Template.Labels, nonVersionedKeys)
	versionPlan.Template.Annotations = filterNonVersioned(versionPlan.Template.Annotations, nonVersionedKeys)
	hippoPodTemplate, err := GetHippoPodTemplate(versionPlan)
	if err != nil {
		glog.Errorf("Unmarshal error :%s,%v", name, err)
		return "", "", err
	}

	rversion, err := GetResVersion(name, hippoPodTemplate.Spec, versionPlan.WorkerSchedulePlan, labels)
	if nil != err {
		return "", "", err
	}
	var version string
	if !NeedRollingAfterResourceChange(versionPlan) {
		var vSignSources = make([]interface{}, 0, 20)
		for i := range hippoPodTemplate.Spec.Containers {
			var ip resource.Quantity
			for k, v := range hippoPodTemplate.Spec.Containers[i].Resources.Limits {
				if k == ResourceIP {
					ip = v
				}
			}
			vSignSources = append(vSignSources, ip)
		}
		clearPodResource(&hippoPodTemplate.Spec)
		versionPlan.Template = hippoPodTemplate
		vSignSources = append(vSignSources, versionPlan.SignedVersionPlan, labels)
		version, err = utils.SignatureWithMD5(vSignSources...)
		if nil != err {
			glog.Errorf("Signature error :%s,%v", name, err)
			return "", "", err
		}
	} else {
		version, err = utils.SignatureWithMD5(versionPlan.SignedVersionPlan, labels)
	}
	return version, rversion, nil
}

// GetResVersion GetResVersion
func GetResVersion(name string, hippoPodSpec HippoPodSpec, ins ...interface{}) (string, error) {
	var podResource PodResource
	var err error
	var rversion string
	err = utils.JSONDeepCopy(&hippoPodSpec, &podResource)
	if err != nil {
		glog.Errorf("Unmarshal error :%s,%v", name, err)
		return "", err
	}
	ress := megerContainersRess1(podResource.Containers)
	if nil != ress.Limits {
		delete(ress.Limits, ResourceIP)
	}
	podResource.Containers = nil
	if len(ins) == 0 {
		rversion, err = utils.SignatureWithMD5(podResource, ress)
	} else {
		var newIns []interface{}
		newIns = append(newIns, podResource, ress)
		newIns = append(newIns, ins...)
		rversion, err = utils.SignatureWithMD5(newIns...)
	}
	if nil != err {
		glog.Errorf("Signature error :%s,%v", name, err)
		return "", err
	}
	return rversion, nil
}

func clearPodResource(hippoPodSpec *HippoPodSpec) {
	for i := range hippoPodSpec.Containers {
		hippoPodSpec.Containers[i].Resources = corev1.ResourceRequirements{}
	}

	hippoPodSpec.NodeSelector = nil
	hippoPodSpec.NodeName = ""
	hippoPodSpec.Affinity = nil
	hippoPodSpec.Tolerations = nil
	hippoPodSpec.Priority = nil
	hippoPodSpec.CpusetMode = ""
	hippoPodSpec.CPUShareNum = nil
}

// CopyPodResource copy resource fields 把目标资源copy到原pod上
func CopyPodResource(hippoPodSpec *HippoPodSpec, target *HippoPodSpec, copyIP bool) error {
	if 0 == len(hippoPodSpec.Containers) {
		hippoPodSpec.Containers = target.Containers
	}
	err := CopyHippoContainersResource(hippoPodSpec.Containers, target.Containers, copyIP)
	CopyCorePodResource(&hippoPodSpec.PodSpec, &target.PodSpec, copyIP)
	hippoPodSpec.CpusetMode = target.CpusetMode
	hippoPodSpec.CPUShareNum = target.CPUShareNum
	return err
}

// CopyContainersResource copy resource fields 把目标资源copy到原pod上
func CopyContainersResource(current, target []corev1.Container, copyIP bool) error {
	for i := range current {
		found := false
		for j := range target {
			if current[i].Name == target[j].Name {
				current[i].Resources = copyContainerResource(current[i].Resources, target[j].Resources, copyIP)
				found = true
				break
			}
		}
		if !found {
			err := fmt.Errorf("container not matched: %s", current[i].Name)
			return err
		}
	}
	return nil
}

// CopyHippoContainersResource copy resource fields 把目标资源copy到原pod上
func CopyHippoContainersResource(current, target []HippoContainer, copyIP bool) error {
	for i := range current {
		found := false
		for j := range target {
			if current[i].Name == target[j].Name {
				current[i].Resources = copyContainerResource(current[i].Resources, target[j].Resources, copyIP)
				found = true
				break
			}
		}
		if !found {
			err := fmt.Errorf("container not matched: %s", current[i].Name)
			return err
		}
	}
	return nil
}

// CopyCorePodResource copy resource fields 把目标资源copy到原pod上
func CopyCorePodResource(podSpec *corev1.PodSpec, target *corev1.PodSpec, copyIP bool) {
	podSpec.NodeSelector = target.NodeSelector
	podSpec.Affinity = target.Affinity
	podSpec.Tolerations = target.Tolerations
	if nil != target.Priority {
		podSpec.Priority = target.Priority
	}
	if copyIP {
		podSpec.HostNetwork = target.HostNetwork
	}
}

func copyContainerResource(src, dst corev1.ResourceRequirements, copyIP bool) corev1.ResourceRequirements {
	src.Limits = copyContainerResourceList(src.Limits, dst.Limits, copyIP)
	src.Requests = copyContainerResourceList(src.Requests, dst.Requests, copyIP)
	return src
}

// 把目标资源copy到原pod上
func copyContainerResourceList(src, dst corev1.ResourceList, copyIP bool) corev1.ResourceList {
	if dst != nil {
		ip, ok := src[ResourceIP] // 默认ip不能变更
		src = dst
		if !copyIP {
			if !ok {
				delete(src, ResourceIP)
			} else {
				src[ResourceIP] = ip
			}
		}
	}
	return src
}

// UpdateReplicaResource update replica resource fields
func UpdateReplicaResource(replica *Replica, target *HippoPodSpec,
	labels map[string]string, resourceVersion string, workerSchedulePlan WorkerSchedulePlan) error {

	var template = replica.Spec.VersionPlan.Template
	if len(template.Spec.Containers) != len(target.Containers) {
		err := fmt.Errorf("containers not matched : %s,%d,%d", replica.Name, len(template.Spec.Containers), len(target.Containers))
		glog.Error(err)
		return err
	}
	cold2WarmState, exist := replica.Annotations[ColdResourceUpdateState]
	if exist {
		replica.Annotations[ColdResourceUpdateTargetVersion] = resourceVersion
		if cold2WarmState == ColdUpdateStateColdToWarm {
			if IsWorkerAssigned(&replica.WorkerNode) {
				replica.Annotations[ColdResourceUpdateState] = ColdUpdateStateWaitingWarmAssgin
				glog.Infof("replica[%v] is assigned, need update resource now, replica.Spec.ResVersion[%v] target resourceVersion[%v]",
					replica.Name, replica.Spec.ResVersion, resourceVersion)
			} else {
				glog.Infof("replica[%v] is cold update state[%v], waiting assign", replica.Name, ColdUpdateStateColdToWarm)
				return fmt.Errorf("replica[%v] keep warm for resource update", replica.Name)
			}
		}
	}
	if replica.Spec.WorkerMode == WorkerModeTypeCold {
		glog.Infof("need change replica[%v] workmode to warm for resource updating, cur resource[%v]", replica.Name, utils.ObjJSON(replica.WorkerNode.Spec.Template.Spec.Containers[0].Resources))
		replica.Spec.WorkerMode = WorkerModeTypeWarm
		replica.Annotations[ColdResourceUpdateState] = ColdUpdateStateColdToWarm
		replica.Annotations[ColdResourceUpdateTargetVersion] = resourceVersion
		return nil
	}
	err := CopyPodResource(&template.Spec, target, false)
	if nil != err {
		glog.Error(err)
		return err
	}
	replica.Spec.ResVersion = resourceVersion
	replica.Spec.WorkerSchedulePlan = workerSchedulePlan
	oldlabels := replica.Labels
	replica.Labels = CopyWorkerAndReplicaLabels(labels, oldlabels)
	// pod version 保持不变
	if podVersion, ok := oldlabels[LabelKeyPodVersion]; ok {
		replica.Labels[LabelKeyPodVersion] = podVersion
	} else {
		delete(replica.Labels, LabelKeyPodVersion)
	}

	return nil
}

// SubReplicas sub replicas
func SubReplicas(src []*Replica, sub []*Replica) []*Replica {
	var subMap = make(map[string]string)
	for i := range sub {
		subMap[sub[i].Name] = sub[i].Name
	}
	var result = make([]*Replica, 0, len(src))
	for i := range src {
		if _, ok := subMap[src[i].Name]; !ok {
			result = append(result, src[i])
		}
	}
	return result
}

// ObjKind get kind of obj
type ObjKind interface {
	GetKind() string
}

// GetKind get kind of obj
func (w *WorkerNode) GetKind() string {
	return w.Kind
}

// GetKind get kind of obj
func (o *RollingSet) GetKind() string {
	return o.Kind
}

// GetKind get kind of obj
func (o *Replica) GetKind() string {
	return o.Kind
}

// GetKind get kind of obj
func (o *ShardGroup) GetKind() string {
	return o.Kind
}

const (
	complete     = "Complete"
	notcomplete  = "NotComplete"
	rolling      = "Rolling"
	reclaim      = "Reclaim"
	ready        = "Ready"
	notRunning   = "NotRunning"
	notReady     = "NotReady"
	notHealth    = "NotHealth"
	notAvailable = "NotAvailable"
	notAssigned  = "NotAssigned"
)

// GetRollingSetStatus get status description
func GetRollingSetStatus(r *RollingSet) string {
	if r.Status.Complete {
		return complete
	}
	return notcomplete
}

// GetReplicaStatus get status description
func GetReplicaStatus(r *Replica) string {
	if IsReplicaComplete(r) {
		return complete
	}
	if !IsReplicaAssigned(r) {
		return string(r.Status.AllocStatus)
	}
	if r.Status.Reclaim || r.Status.InternalReclaim {
		return reclaim
	}
	if r.Status.Phase != Running {
		return string(r.Status.Phase)
	}
	if r.Status.HealthStatus != HealthAlive {
		return string(r.Status.HealthStatus)
	}
	return string(r.Status.ServiceStatus)
}

// GetWorkerStatus get status description
func GetWorkerStatus(r *WorkerNode) string {
	if IsWorkerComplete(r) {
		return complete
	}
	if !IsWorkerAssigned(r) {
		return notAssigned
	}
	if r.Status.Reclaim || r.Status.InternalReclaim {
		return reclaim
	}
	if r.Status.Phase != Running {
		return notRunning
	}
	if r.Status.HealthStatus != HealthAlive {
		return notHealth
	}
	if r.Status.HealthCondition.WorkerStatus != WorkerTypeReady {
		return notReady
	}
	return notAvailable
}

// GetPodStatus get status description
func GetPodStatus(r *corev1.Pod) string {
	return string(r.Status.Phase)
}

// GetGroupStatus get status description
func GetGroupStatus(g *ShardGroup) string {
	return string(g.Status.Phase)
}

// SignWorkerRequirementID SignWorkerRequirementID
func SignWorkerRequirementID(worker *WorkerNode, suffix string) (string, error) {
	return utils.SignatureWithMD5(
		worker.Spec.Template,
		worker.Labels[DefaultRollingsetUniqueLabelKey],
		worker.Labels[LabelKeyAppName],
		worker.Labels[LabelKeyExlusive],
		worker.Labels[LabelKeyDeclare],
		worker.Labels[LabelKeyQuotaGroupID],
		worker.Labels[LabelKeyMaxInstancePerNode],
		worker.Labels[LabelKeyPodVersion],
		worker.Namespace,
		suffix,
		worker.Spec.WorkerMode,
	)
}

// GetHealthStatusByPhase GetHealthStatusByPhase
func GetHealthStatusByPhase(worker *WorkerNode) HealthStatus {
	if nil == worker {
		return HealthUnKnown
	}

	switch worker.Status.Phase {
	case Failed:
		return HealthDead
	case Terminated:
		if isDaemonPlan(worker.Spec.Template) {
			return HealthDead
		}
		return HealthAlive
	case Pending:
		return HealthUnKnown
	case Running:
		return HealthAlive
	default:
		return HealthUnKnown
	}
}

// IsHealthCheckDone IsHealthCheckDone
func IsHealthCheckDone(worker *WorkerNode) bool {
	return worker.Status.HealthCondition.Checked
}

// GetHealthMetas get health metas
func GetHealthMetas(worker *WorkerNode) map[string]string {
	if len(worker.Status.HealthCondition.Metas) > 0 {
		return worker.Status.HealthCondition.Metas
	}
	if worker.Status.HealthCondition.CompressedMetas != "" {
		metaStr, err := utils.DecompressStringToString(worker.Status.HealthCondition.CompressedMetas)
		if err != nil {
			glog.Errorf("fillHealthMetas err %v", err)
			return worker.Status.HealthCondition.Metas
		}
		metas := make(map[string]string)
		err = json.Unmarshal([]byte(metaStr), &metas)
		if err != nil {
			glog.Errorf("fillHealthMetas err %v", err)
			return worker.Status.HealthCondition.Metas
		}
		return metas
	}
	return worker.Status.HealthCondition.Metas
}

// GetCustomInfo get custominfo
func GetCustomInfo(versionPlan *VersionPlan) string {
	if versionPlan.CustomInfo != "" {
		return versionPlan.CustomInfo
	}
	if versionPlan.CompressedCustomInfo != "" {
		customInfo, err := utils.DecompressStringToString(versionPlan.CompressedCustomInfo)
		if err != nil || customInfo == "" {
			glog.Errorf("get getCustomInfo error  %v", err)
			return versionPlan.CustomInfo
		}
		return customInfo
	}
	return versionPlan.CustomInfo
}

// GenUniqIdentifier GenUniqIdentifier
func GenUniqIdentifier(workernode *WorkerNode) string {
	hippoClusterName := GetClusterName(workernode)
	identifier := utils.AppendString(hippoClusterName, ":", workernode.Namespace, ":", workernode.Name)
	return identifier
}

// Default Timeout
var (
	defaultResourceMatchTimeout = 60 * 5
	defaultProcessMatchTimeout  = 60 * 10
)

// InitWorkerScheduleTimeout InitWorkerScheduleTimeout
func InitWorkerScheduleTimeout(worker *WorkerSchedulePlan) {
	if worker.ResourceMatchTimeout == 0 {
		worker.ResourceMatchTimeout = int64(defaultResourceMatchTimeout)
	}
	if worker.ProcessMatchTimeout == 0 {
		worker.ProcessMatchTimeout = int64(defaultProcessMatchTimeout)
	}
	if worker.WarmupSeconds == 0 {
		worker.WarmupSeconds = int32(defaultWarmupSeconds)
	}
}

// GetPodStatusFromCondition GetPodStatusFromCondition
func GetPodStatusFromCondition(pod *corev1.Pod, conditionType corev1.PodConditionType) corev1.ConditionStatus {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == conditionType {
			return condition.Status
		}
	}
	return corev1.ConditionUnknown
}

// ClearObjectMeta clear meta to init state
func ClearObjectMeta(meta *metav1.ObjectMeta) {
	meta.SelfLink = ""
	meta.UID = ""
	meta.ResourceVersion = ""
	meta.Generation = 0
	meta.CreationTimestamp = metav1.Time{}
	meta.DeletionTimestamp = nil
	meta.DeletionGracePeriodSeconds = nil
}

// PodIsEvicted returns true if the reported pod status is due to an eviction.
func PodIsEvicted(podStatus v1.PodStatus) bool {
	return podStatus.Phase == v1.PodFailed && podStatus.Reason == "Evicted"
}

// MegerContainersRess MegerContainersRess
func MegerContainersRess(containers []ContainerInstanceField) corev1.ResourceRequirements {
	var ress = make([]corev1.ResourceRequirements, len(containers))
	for i := range containers {
		ress[i] = containers[i].Resources
	}
	return megerContainersRess(ress)
}

func megerContainersRess1(containers []ContainerResource) corev1.ResourceRequirements {
	var ress = make([]corev1.ResourceRequirements, len(containers))
	for i := range containers {
		ress[i] = containers[i].Resources
	}
	return megerContainersRess(ress)
}

// megerContainersRess megerContainersRess
func megerContainersRess(ress []corev1.ResourceRequirements) corev1.ResourceRequirements {
	if len(ress) == 1 {
		return ress[0]
	}

	var limits corev1.ResourceList = make(map[corev1.ResourceName]resource.Quantity)
	for i := range ress {
		for k, v := range ress[i].Limits {
			if k == "cpu" || k == "memory" {
				old := limits[k]
				new := v.MilliValue() + old.MilliValue()
				v.SetMilli(new)
			}
			limits[k] = v
		}
	}
	return corev1.ResourceRequirements{Limits: limits}
}

func (c *HealthCondition) String() string {
	var versionsStringBuilder strings.Builder
	versionsStringBuilder.WriteString("Type: ")
	versionsStringBuilder.WriteString(string(c.Type))
	versionsStringBuilder.WriteString(", Status: ")
	versionsStringBuilder.WriteString(string(c.Status))
	versionsStringBuilder.WriteString(", LastTransitionTime ")
	versionsStringBuilder.WriteString(c.LastTransitionTime.String())
	versionsStringBuilder.WriteString(", Reason: ")
	versionsStringBuilder.WriteString(c.Reason)
	versionsStringBuilder.WriteString(", LostCount")
	versionsStringBuilder.WriteString(strconv.Itoa(int(c.LostCount)))
	versionsStringBuilder.WriteString(", Version")
	versionsStringBuilder.WriteString(c.Version)
	versionsStringBuilder.WriteString(", WorkerStatus")
	versionsStringBuilder.WriteString(string(c.WorkerStatus))
	return versionsStringBuilder.String()
}

func (w *WorkerNode) String() string {
	if nil == w {
		return "nil"
	}
	w = w.DeepCopy()
	w.Spec.VersionPlan.CustomInfo = ""

	w.Status.HealthCondition.Metas = nil
	w.Status.HealthCondition.CompressedMetas = ""
	w.Status.ServiceInfoMetas = ""
	return utils.ObjJSON(w)
}

// GetQualifiedName GetQualifiedName
func (w *WorkerNode) GetQualifiedName() string {
	if nil == w {
		return ""
	}
	return fmt.Sprintf("%s::%s::%s", GetRoleNameAndName(w), w.Status.IP, w.Status.HostIP)
}

func (w *WorkerNode) Copy4PBPersist() (proto.Message, error) {
	out := w.DeepCopy()
	out.Status.HealthCondition.Metas = nil
	out.Status.HealthCondition.CompressedMetas = ""
	out.Status.ServiceInfoMetas = ""
	out.Status.HealthCondition.Checked = false
	out.Status.LastUpdateStatusTime = metav1.Time{}
	return out, nil
}

// FixSchedulerName FixSchedulerName
func FixSchedulerName(spec *HippoPodSpec) {
	if "" == spec.SchedulerName && "" != defaultSchedulerName {
		spec.SchedulerName = defaultSchedulerName
	}
}

// GetDefaultSchedulerName GetDefaultSchedulerName
func GetDefaultSchedulerName() string {
	if "" == defaultSchedulerName {
		return SchedulerNameDefault
	}
	return defaultSchedulerName
}

// IsAsiPod IsAsiPod
func IsAsiPod(meta *metav1.ObjectMeta) bool {
	podVersion := defaultPodVersion
	if nil != meta && nil != meta.Labels && "" != meta.Labels[LabelKeyPodVersion] {
		podVersion = meta.Labels[LabelKeyPodVersion]
	}
	return podVersion == PodVersion3
}

// GetHippoPodTemplateFromPod GetHippoPodTemplateFromPod
func GetHippoPodTemplateFromPod(pod interface{}) HippoPodTemplate {
	var template HippoPodTemplate
	if nil == pod {
		return template
	}
	utils.JSONDeepCopy(pod, &template)
	return template
}

// GenerateShardName  is used to generate a special unique shardName
func GenerateShardName(shardGroupName string, shardKey string) string {
	return shardGroupName + "." + shardKey
}

// GetRollingsetNameHashs GetRollingsetNameHashs
func GetRollingsetNameHashs(group *ShardGroup) map[string]bool {
	compressedShardTemplates := group.Spec.CompressedShardTemplates
	DecompressShardTemplates(group)
	if nil == group || nil == group.Spec.ShardTemplates {
		return nil
	}
	var names = map[string]bool{}
	for k := range group.Spec.ShardTemplates {
		name, _ := ResourceNameHash(GenerateShardName(group.Name, k))
		names[name] = true
	}
	if len(group.Spec.ShardTemplates) >= CompresseShardThreshold && compressedShardTemplates != "" {
		group.Spec.CompressedShardTemplates = compressedShardTemplates
		group.Spec.ShardTemplates = nil
	}
	return names
}

// NeedWarmup NeedWarmup
func NeedWarmup(worker *WorkerNode) bool {
	if nil == worker {
		return false
	}
	if worker.Spec.WarmupSeconds > 0 {
		for i := range worker.Status.ServiceConditions {
			if worker.Status.ServiceConditions[i].Type == ServiceCM2 {
				return true
			}
		}
	}
	return false
}

// SyncWorkerInUpdating sync worker InUpdating status
func SyncWorkerInUpdating(worker *WorkerNode) {
	if nil == worker {
		return
	}
	if worker.Spec.Version != worker.Status.Version {
		worker.Status.InUpdating = true
	} else if worker.Status.Complete {
		worker.Status.InUpdating = false
	}
}

func IsWorkerRowComplete(worker *WorkerNode) bool {
	if worker == nil {
		return false
	}
	if worker.Spec.RowComplete == nil {
		return true
	}
	return *worker.Spec.RowComplete
}

// SyncCrdTime sync crd time //concurrent map writes 问题无法解决
func SyncCrdTime(old *metav1.ObjectMeta, new *metav1.ObjectMeta) {
	if old.Annotations == nil || new.Annotations == nil {
		return
	}
	if value, ok := old.Annotations["updateCrdTime"]; !ok {
		delete(new.Annotations, "updateCrdTime")
	} else {
		new.Annotations["updateCrdTime"] = value
	}
	for k, v := range old.Annotations {
		if strings.HasPrefix(k, "updateVersionTime") {
			new.Annotations[k] = v
		}
	}
}

func MarshalStablePodStatus(worker *WorkerNode) string {
	status := worker.Status.PodStandbyStatus
	if status.UseOrder == 0 && status.StandbyHours == nil {
		return ""
	}
	marshal, err := json.Marshal(status)
	if err != nil {
		glog.Errorf("marshal pod standby status error %v", err)
		return ""
	}
	if marshal == nil {
		return ""
	}
	return string(marshal)
}

func GetReplicaKey(obj metav1.Object) (string, bool) {
	labels := obj.GetLabels()
	if labels != nil {
		replicaHash := labels[DefaultReplicaUniqueLabelKey]
		if replicaHash != "" {
			return replicaHash, true
		}
	}
	objName := obj.GetName()
	replicaName, ok := getReplicaKeyFromName(objName)
	if ok {
		return replicaName, true
	}
	workerName := objName[0 : len(objName)-4]
	if strings.HasSuffix(workerName, "-") {
		workerName = strings.TrimSuffix(workerName, "-")
	} else {
		return "", false
	}
	replicaName, ok = getReplicaKeyFromName(workerName)
	if ok {
		return replicaName, true
	}
	return "", false
}

func getReplicaKeyFromName(name string) (string, bool) {
	if strings.HasSuffix(name, WorkerNameSuffixA) {
		return strings.TrimSuffix(name, WorkerNameSuffixA), true
	} else if strings.HasSuffix(name, WorkerNameSuffixB) {
		return strings.TrimSuffix(name, WorkerNameSuffixB), true
	}
	return "", false
}

func getWorkerNameFromName(name string) (string, bool) {
	if strings.HasSuffix(name, WorkerNameSuffixA) {
		return name, true
	} else if strings.HasSuffix(name, WorkerNameSuffixB) {
		return name, true
	}
	return "", false
}

func GetPairWorkerName(currWorkerName string) string {
	replicaName, ok := getReplicaKeyFromName(currWorkerName)
	if !ok {
		return ""
	}
	if (replicaName + WorkerNameSuffixA) == currWorkerName {
		return replicaName + WorkerNameSuffixB
	}
	return replicaName + WorkerNameSuffixA
}

// GetWorkerNameFromPod get the worker name from pod info
func GetWorkerNameFromPod(obj metav1.Object) (name string, err error) {
	if ownerRef, _ := mem.GetControllerOf(obj); ownerRef != nil {
		if ownerRef.Kind == "WorkerNode" { // multiple resource kind owns the same kind child resource
			name = ownerRef.Name
			return
		}
	}
	for i := range obj.GetOwnerReferences() {
		ref := obj.GetOwnerReferences()[i]
		if ref.Kind == "WorkerNode" { // multiple resource kind owns the same kind child resource
			name = ref.Name
			return
		}
	}
	var ok bool
	name, ok = getWorkerNameFromName(obj.GetName())
	if ok {
		return
	}
	workerName := obj.GetName()[0 : len(obj.GetName())-4]
	if strings.HasSuffix(workerName, "-") {
		workerName = strings.TrimSuffix(workerName, "-")
		name, ok = getWorkerNameFromName(workerName)
		if ok {
			return
		}
	}
	err = fmt.Errorf("can't get worker name from pod %s", obj.GetName())
	return
}

func DecompressShardTemplates(shardgroup *ShardGroup) error {
	if len(shardgroup.Spec.ShardTemplates) == 0 && shardgroup.Spec.CompressedShardTemplates != "" {
		shardTemplatesJson, err := utils.DecompressStringToString(shardgroup.Spec.CompressedShardTemplates)
		if err != nil {
			return err
		}
		var templates map[string]ShardTemplate
		err = json.Unmarshal([]byte(shardTemplatesJson), &templates)
		if err != nil {
			return err
		}
		shardgroup.Spec.ShardTemplates = templates
		shardgroup.Spec.CompressedShardTemplates = ""
	}
	return nil
}

func CompressShardTemplates(shardgroup *ShardGroup) {
	if len(shardgroup.Spec.ShardTemplates) >= CompresseShardThreshold {
		shardgroup.Spec.CompressedShardTemplates = utils.CompressStringToString(utils.ObjJSON(shardgroup.Spec.ShardTemplates))
		shardgroup.Spec.ShardTemplates = nil
	}
}

func IsWorkerNodeUseUnifiedStorageVolume(worker *WorkerNode) bool {
	if worker == nil || worker.Spec.Template == nil || worker.Spec.Template.Annotations == nil {
		return false
	}
	return UseUnifiedStorageVolume(&worker.Spec.Template.ObjectMeta)
}

// UseUnifiedStorageVolume UseUnifiedStorageVolume
func UseUnifiedStorageVolume(meta *metav1.ObjectMeta) bool {
	if meta.Annotations[AnnotationOldPodLevelStorage] == "true" {
		return true
	}
	if _, exist := meta.Annotations[AnnotationPodLevelStorageMode]; exist {
		return true
	}
	return false
}

// FixUnifiedStorageVolumeMounts FixUnifiedStorageVolumeMounts
func FixUnifiedStorageVolumeMounts(pod *HippoPodTemplate) {
	if !UseUnifiedStorageVolume(&pod.ObjectMeta) {
		return
	}
	podOldVolumes := pod.Spec.Volumes
	emptyDirVolumeNames := make(map[string]struct{}, len(podOldVolumes))

	// 首先找到使用本地存储的所有emptydir volumes
	// pod.Spec.Volumes = make([]v1.Volume, 0, len(podOldVolumes))
	for _, v := range podOldVolumes {
		if v.EmptyDir != nil && v.EmptyDir.Medium == corev1.StorageMediumDefault {
			emptyDirVolumeNames[v.Name] = struct{}{}
			// continue
		}
		// pod.Spec.Volumes = append(pod.Spec.Volumes, v)
	}

	// 将使用本地存储emptydir volumes的容器的mount关系改为PodSpec中指定的csi volume上的mount
	// 关系，并且使用原来emptydir volume name作为subPath来隔离目录，这样原来被多个容器共享的emptydir
	// volume在csi volume上对应同一个目录从而达到共享的目的。
	for i, c := range pod.Spec.InitContainers {
		for j, vm := range c.VolumeMounts {
			if _, ok := emptyDirVolumeNames[vm.Name]; ok {
				pod.Spec.InitContainers[i].VolumeMounts[j].Name = UnifiedStorageInjectedVolumeName
				pod.Spec.InitContainers[i].VolumeMounts[j].SubPath = vm.Name
			}
		}
	}
	for i, c := range pod.Spec.Containers {
		for j, vm := range c.VolumeMounts {
			if _, ok := emptyDirVolumeNames[vm.Name]; ok {
				pod.Spec.Containers[i].VolumeMounts[j].Name = UnifiedStorageInjectedVolumeName
				pod.Spec.Containers[i].VolumeMounts[j].SubPath = vm.Name
			}
		}
	}
}

var (
	c2NetDiskName string = "c2-netdisk"
)

// GetRolePvcName GetRolePvcName
func GetRolePvcName(object metav1.Object) string {
	appName := GetLabelValue(object, LabelKeyAppNameHash)
	if appName == "" {
		appName = GetAppName(object)
	}
	roleName := GetLabelValue(object, LabelKeyRoleNameHash)
	if roleName == "" {
		roleName = GetRoleName(object)
	}
	if roleName == "" {
		glog.Warningf("empty roleName for pvc")
		return ""
	}

	return c2NetDiskName + "-" + EscapeName(appName) + "-" + EscapeName(roleName)
}

// FixRolePvcVolumeMounts FixRolePvcVolumeMounts
func FixRolePvcVolumeMounts(pod *HippoPodTemplate) {
	storagetStrategy, exist := pod.ObjectMeta.Annotations[AnnotationAppStoragetStrategy]
	if !exist {
		return
	}
	if storagetStrategy != AppStorageStrategyRole {
		glog.Warningf("can't support storage stragety:%v", storagetStrategy)
		return
	}

	mountPath := "/c2NetDisk"
	if path, exist := pod.ObjectMeta.Annotations[AnnotationAppStoragetMountPath]; exist {
		mountPath = path
	}
	pvcName := GetRolePvcName(pod)
	if pvcName == "" {
		glog.Warningf("can't get pvc name for pod[%v]", pod.Name)
		return
	}
	pod.Spec.Volumes = append(pod.Spec.Volumes, v1.Volume{
		Name: c2NetDiskName,
		VolumeSource: v1.VolumeSource{
			PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
				ClaimName: pvcName,
			},
		},
	})

	for i := range pod.Spec.InitContainers {
		pod.Spec.InitContainers[i].VolumeMounts = append(pod.Spec.InitContainers[i].VolumeMounts,
			v1.VolumeMount{MountPath: mountPath, Name: c2NetDiskName, SubPath: "empty-dir-volume"})
		pod.Spec.Containers[i].Env = append(pod.Spec.Containers[i].Env, corev1.EnvVar{Name: StorageMountEnvKey, Value: mountPath})
	}
	for i := range pod.Spec.Containers {
		pod.Spec.Containers[i].VolumeMounts = append(pod.Spec.Containers[i].VolumeMounts,
			v1.VolumeMount{MountPath: mountPath, Name: c2NetDiskName, SubPath: "empty-dir-volume"})
		pod.Spec.Containers[i].Env = append(pod.Spec.Containers[i].Env, corev1.EnvVar{Name: StorageMountEnvKey, Value: mountPath})
	}
}

// ComputeInstanceID ComputeInstanceID
func ComputeInstanceID(needRestartAfterResourceChange bool, hippoPodSpec *HippoPodSpec) (string, map[string]string, error) {
	var hippoPodSpecInstance PodInstanceField
	var processVersion string
	var containerProcessVersion = map[string]string{}
	err := utils.JSONDeepCopy(&hippoPodSpec, &hippoPodSpecInstance)
	if nil != err {
		return processVersion, containerProcessVersion, err
	}
	var mergedResource corev1.ResourceRequirements
	if needRestartAfterResourceChange && len(hippoPodSpecInstance.Containers) > 1 {
		mergedResource = MegerContainersRess(hippoPodSpecInstance.Containers)
	}
	delete(mergedResource.Limits, ResourceDiskRatio)
	delete(mergedResource.Requests, ResourceDiskRatio)
	for i := range hippoPodSpecInstance.Containers {
		var ip resource.Quantity
		for k, v := range hippoPodSpecInstance.Containers[i].Resources.Limits {
			if k == ResourceIP {
				ip = v
			}
		}
		if needRestartAfterResourceChange {
			delete(hippoPodSpecInstance.Containers[i].Resources.Limits, ResourceDiskRatio)
			delete(hippoPodSpecInstance.Containers[i].Resources.Requests, ResourceDiskRatio)
			if len(hippoPodSpec.Containers) > 1 {
				hippoPodSpecInstance.Containers[i].Resources = mergedResource
			}
			containerProcessVersion[hippoPodSpecInstance.Containers[i].Name], _ = utils.SignatureShort(
				hippoPodSpecInstance.Containers[i],
			)
		} else {
			hippoPodSpecInstance.Containers[i].Resources = corev1.ResourceRequirements{}
			hippoPodSpecInstance.CPUShareNum = nil
			containerProcessVersion[hippoPodSpecInstance.Containers[i].Name], _ = utils.SignatureShort(
				hippoPodSpecInstance.Containers[i],
				ip,
			)
		}
	}
	if hippoPodSpec.ContainerModel != "VM" {
		hippoPodSpecInstance.Containers = nil
	}
	if !needRestartAfterResourceChange {
		hippoPodSpecInstance.CpusetMode = ""
		hippoPodSpecInstance.CPUShareNum = nil
	}
	processVersion, err = utils.SignatureShort(hippoPodSpecInstance)
	return processVersion, containerProcessVersion, err
}

// IsGroupRollingSet IsGroupRollingSet
func IsGroupRollingSet(rollingSet *RollingSet) bool {
	return IsSubrsEnable(rollingSet) && rollingSet.Spec.GroupRS == nil
}

// IsSubRollingSet IsSubRollingSet
func IsSubRollingSet(rollingSet *RollingSet) bool {
	return rollingSet.Spec.GroupRS != nil
}

// IsBufferRollingSet IsBufferRollingSet
func IsBufferRollingSet(rollingSet *RollingSet) bool {
	return IsBufferRollingSetName(rollingSet.Name)
}

// IsBufferRollingSetName IsBufferRollingSetName
func IsBufferRollingSetName(rsName string) bool {
	return strings.HasSuffix(rsName, GroupRollingSetBufferShortName)
}

// GetBufferName GetBufferName
func GetBufferName(rollingSet *RollingSet) string {
	if IsGroupRollingSet(rollingSet) {
		return GenerateBufferName(rollingSet.Name)
	}
	if IsSubRollingSet(rollingSet) {
		return GenerateBufferName(*rollingSet.Spec.GroupRS)
	}
	glog.Infof("getBufferName null %s", utils.ObjJSON(rollingSet))
	return ""
}

// GenerateBufferName GenerateBufferName
func GenerateBufferName(grsName string) string {
	return fmt.Sprintf("%s.%s", grsName, GroupRollingSetBufferShortName)
}

// MapCopy MapCopy
func MapCopy(src map[string]string, dst map[string]string) {
	if src == nil || dst == nil {
		return
	}
	for key, val := range src {
		dst[key] = val
	}
}

func CopySpecificKeys(src map[string]string, dst map[string]string, keys []string) {
	if src == nil || dst == nil || len(keys) == 0 {
		return
	}
	for _, key := range keys {
		if val, ok := src[key]; ok {
			dst[key] = val
		}
	}
}

func CopySunfireLabels(src map[string]string, dst map[string]string) {
	appName := src[LabelServerlessAppName]
	group := src[LabelServerlessInstanceGroup]
	if appName != "" {
		dst[LabelKeyAppShortName] = appName
	}
	if group != "" {
		dst[LabelKeyRoleShortName] = group
	}
}

// GenerateSubrsName GenerateSubrsName
func GenerateSubrsName(grsName, shortSubrsName string) string {
	return fmt.Sprintf("%s.%s", grsName, shortSubrsName)
}

// GenAppMetaID gen ID
func GenAppMetaID(app, group, role string) string {
	rawID := fmt.Sprintf("%s-%s", group, role)
	escapedID := EscapeName(rawID)
	return LabelValueHash(escapedID, true)
}

var _reg *regexp.Regexp = regexp.MustCompile(`[^a-z0-9-.]`)

// EscapeName for kube-apiserver
func EscapeName(name string) string {
	name = strings.ToLower(name)
	result := _reg.ReplaceAllString(name, "-")
	if strings.HasPrefix(result, "-") {
		result = "p" + result
	}
	if strings.HasSuffix(result, "-") || strings.HasSuffix(result, ".") {
		result = result + "s"
	}
	return result
}

// GenAppBatchID gen batch ID
func GenAppBatchID(app, group, batchID string) string {
	id := fmt.Sprintf("%s-%s", app, batchID)
	return LabelValueHash(id, true)
}

// GetDefaultPodVersion default pod version
func GetDefaultPodVersion() string {
	return defaultPodVersion
}

// GetGroupVersionStatusMap 返回groupverison的状态
func GetGroupVersionStatusMap(replicaSet []*Replica) (int64, map[string]*rollalgorithm.VersionStatus) {
	groupVersionStatusMap := make(map[string]*rollalgorithm.VersionStatus)
	var timeout int64
	for _, replica := range replicaSet {
		silenceTimeout := GetWorkerSilenceTimeout(&replica.WorkerNode)
		if silenceTimeout > 0 && (silenceTimeout < timeout || timeout == 0) {
			timeout = silenceTimeout
		}
		if IsStandbyReplica(replica) {
			continue
		}
		version := getGroupVersion(replica)
		versionStatus, ok := groupVersionStatusMap[version]
		if !ok {
			versionStatus = &rollalgorithm.VersionStatus{Version: version,
				Replicas:      0,
				ReadyReplicas: 0,
			}
			groupVersionStatusMap[version] = versionStatus
		}
		versionStatus.Replicas++
		replicatimeout, ok := IsReplicaServiceReady(replica)
		if glog.V(4) {
			glog.Infof("[%s] IsReplicaServiceReady %v,%d", replica.Name, ok, replicatimeout)
		}
		if ok {
			versionStatus.ReadyReplicas++
		} else {
			if 0 < replicatimeout && (replicatimeout < timeout || timeout == 0) {
				timeout = replicatimeout
			}
		}
		if IsWorkerHealthInfoReady(&replica.WorkerNode) || (replica.Backup.UID != "" && IsWorkerHealthInfoReady(&replica.Backup)) {
			versionStatus.DataReadyReplicas++
		}
	}
	return timeout, groupVersionStatusMap
}

func getGroupVersion(rs *Replica) string {
	if nil == rs {
		return ""
	}
	if rs.Spec.ShardGroupVersion != "" {
		return rs.Spec.ShardGroupVersion
	}
	return rs.Spec.Version
}

// IsSilenceNode get the pod name of worker
func IsSilenceNode(w *WorkerNode) bool {
	if w == nil || w.Annotations == nil {
		return false
	}
	return w.Annotations[SilentNodeAnnoKey] == "true" && (w.Spec.WorkerMode == WorkerModeTypeHot || w.Status.WorkerMode == WorkerModeTypeHot)
}

// SyncSilence get the pod name of worker
func SyncSilence(w *WorkerNode, targetWorkerMode WorkerModeType) bool {
	if w == nil || w.UID == "" {
		return false
	}
	if targetWorkerMode == WorkerModeTypeInactive {
		return false
	}
	if !IsStandbyWorkerMode(w.Spec.WorkerMode) && IsStandbyWorkerMode(targetWorkerMode) {
		if !IsSilenceNode(w) {
			SetSilence(w, true)
			return true
		}
	}
	if !IsStandbyWorkerMode(targetWorkerMode) {
		SetSilence(w, false)
		return false
	}
	return InSilence(w)
}

// SetSilence get the pod name of worker
func SetSilence(w *WorkerNode, silence bool) bool {
	if w.Annotations == nil {
		w.Annotations = map[string]string{}
	}
	if silence {
		if w.Annotations[SilentNodeAnnoKey] != "true" {
			w.Annotations[SilentTimeAnnoKey] = strconv.FormatInt(time.Now().Unix(), 10)
		}
		w.Annotations[SilentNodeAnnoKey] = "true"
		if w.Spec.WorkerMode != WorkerModeTypeHot {
			glog.Infof("set workermode %s WorkerModeTypeHot for silence", w.Name)
		}
		w.Spec.WorkerMode = WorkerModeTypeHot
	} else {
		delete(w.Annotations, SilentNodeAnnoKey)
	}
	return false
}

// InSilence get the pod name of worker
func InSilence(w *WorkerNode) bool {
	if w == nil || w.Annotations == nil {
		return false
	}
	startTime, err := strconv.ParseInt(w.Annotations[SilentTimeAnnoKey], 10, 64)
	if err != nil {
		return false
	}
	if time.Now().Unix()-startTime < int64(defaultSilenceSeconds) &&
		(w.Status.WorkerMode == WorkerModeTypeActive || w.Status.WorkerMode == WorkerModeTypeHot || w.Status.WorkerMode == "") {
		return true
	}
	return false
}

// InStandby2Active in
func InStandby2Active(w *WorkerNode) bool {
	if w == nil || w.Annotations == nil {
		return false
	}
	startTimeStr, exist := w.Annotations[Standby2ActiveTimeAnnoKey]
	if !exist {
		return false
	}
	startTime, err := strconv.ParseInt(startTimeStr, 10, 64)
	if err != nil {
		return false
	}
	if time.Now().Unix()-startTime < int64(defaultStandby2ActiveSeconds) {
		return true
	}

	glog.Warningf("worker[%v] standby2active timeout", w.Name)
	UnsetStandby2Active(w)

	return false
}

// SetStandby2Active set
func SetStandby2Active(w *WorkerNode) bool {
	// 目前仅有cold切active需要保护不被 #538079
	if w.Spec.WorkerMode != WorkerModeTypeCold {
		return false
	}

	if w.Annotations == nil {
		w.Annotations = map[string]string{}
	}
	if InStandby2Active(w) {
		return false
	}
	w.Annotations[Standby2ActiveTimeAnnoKey] = strconv.FormatInt(time.Now().Unix(), 10)
	if glog.V(4) {
		glog.Infof("worker[%v]begin standby to active now", w.Name)
	}
	return true
}

// UnsetStandby2Active unset
func UnsetStandby2Active(w *WorkerNode) bool {
	if w.Annotations == nil {
		return false
	}
	startTimeStr, exist := w.Annotations[Standby2ActiveTimeAnnoKey]
	if !exist {
		return false
	}

	delete(w.Annotations, Standby2ActiveTimeAnnoKey)

	startTime, err := strconv.ParseInt(startTimeStr, 10, 64)
	if err == nil {
		if time.Now().Unix()-startTime > 120 {
			glog.Warningf("worker[%v] standby to active cost too long: %vs", w.Name, time.Now().Unix()-startTime)
		} else {
			glog.V(3).Infof("worker[%v]standby to active cost %vs", w.Name, time.Now().Unix()-startTime)
		}
	}

	return true
}

// SetBadReason SetBadReason
func SetBadReason(w *WorkerNode, reason BadReasonType) {
	if w == nil {
		return
	}
	w.Status.BadReason = reason
	w.Status.BadReasonMessage = badReasonMessage[reason]
}

// GetPodName get the pod name of worker
func GetPodName(w *WorkerNode) (name string, err error) {
	// 兼容已创建的pod
	if w.Status.EntityName != "" {
		return w.Status.EntityName, nil
	}
	// 避免同名workernode出现同名pod
	var suffix string
	if w.UID != "" {
		suffix = string(w.UID)[0:4]
		name = utils.AppendString(w.Name, "-", suffix)
	} else {
		err = fmt.Errorf("generate pod name error, please wait worker uid")
	}
	return
}

// GetUnassignedResson GetUnassignedResson
func GetUnassignedResson(pod *corev1.Pod) UnassignedReasonType {
	var reason = UnassignedReasonNone
	var podScheduleCondition *corev1.PodCondition
	if pod != nil {
		for i := range pod.Status.Conditions {
			if pod.Status.Conditions[i].Type == corev1.PodScheduled {
				podScheduleCondition = &pod.Status.Conditions[i]
				break
			}
		}
	}
	if podScheduleCondition == nil || podScheduleCondition.Message == "" {
		reason = UnassignedReasonNotSchedule
	} else if strings.Contains(podScheduleCondition.Message, "quota not enough") {
		reason = UnassignedReasonQuotaNotEnough
	} else if strings.Contains(podScheduleCondition.Message, "quota not exist") {
		reason = UnassignedReasonQuotaNotExist
	} else {
		reason = UnassignedReasonResourceNotEnough
	}
	return reason
}

// SetUnassignedReason SetUnassignedReason
func SetUnassignedReason(worker *WorkerNode, reason UnassignedReasonType) {
	var reasonMessage = unassignedReasonMessage[reason]
	worker.Status.UnassignedReason = reason
	worker.Status.UnassignedMessage = reasonMessage
	if !IsWorkerHaveBeenUnassignException(worker) {
		worker.Status.HistoryUnassignedMessage = reasonMessage
	}
}

// IsWorkerHaveBeenUnassignException worker是否分配异常（未调度除外）
func IsWorkerHaveBeenUnassignException(worker *WorkerNode) bool {
	if worker.Status.HistoryUnassignedMessage == "" ||
		worker.Status.HistoryUnassignedMessage == unassignedReasonMessage[UnassignedReasonNone] ||
		worker.Status.HistoryUnassignedMessage == unassignedReasonMessage[UnassignedReasonNotSchedule] {
		return false
	}
	return true
}

// SyncCold2Warm4Update get the pod name of worker
func SyncCold2Warm4Update(worker *WorkerNode) bool {
	if !IsRrResourceUpdating(worker) {
		return false
	}

	targetResVersion, exist := worker.Annotations[ColdResourceUpdateTargetVersion]
	if exist && targetResVersion == worker.Status.ResVersion && IsWorkerComplete(worker) {
		glog.Infof("worker[%v] resource version[%v] is matched, change back to cold now", worker.Name, targetResVersion)
		delete(worker.Annotations, ColdResourceUpdateState)
		delete(worker.Annotations, ColdResourceUpdateTargetVersion)
		return false
	}

	if worker.Spec.WorkerMode == WorkerModeTypeCold {
		worker.Spec.WorkerMode = WorkerModeTypeWarm
		return true
	}

	return false
}

// IsRrResourceUpdating get the pod name of worker
func IsRrResourceUpdating(worker *WorkerNode) bool {
	if worker == nil || worker.Annotations == nil {
		return false
	}
	if _, exist := worker.Annotations[ColdResourceUpdateState]; exist {
		return true
	}
	return false
}

// IsAdminRole IsAdminRole
func IsAdminRole(role string) bool {
	return role == C2AdminRoleTag || role == HippoAdminRoleTag
}

const (
	webHookRejectErrStr = "admission webhook"
)

// IsRejectByWebhook IsRejectByWebhook
func IsRejectByWebhook(err error) bool {
	return strings.Contains(err.Error(), webHookRejectErrStr)
}

// IsSkylineChanged IsSkylineChanged
func IsSkylineChanged(before, after *corev1.Pod) bool {
	if valueChanged(GetInstanceGroup(before), GetInstanceGroup(after)) ||
		valueChanged(GetAppUnit(before), GetAppUnit(after)) ||
		valueChanged(GetAppStage(before), GetAppStage(after)) {
		return true
	}
	return false
}

func valueChanged(before, after string) bool {
	return before != after && before != ""
}

// IsNetworkChanged IsNetworkChanged
func IsNetworkChanged(before, after *corev1.Pod) bool {
	if before.Spec.HostNetwork != after.Spec.HostNetwork {
		return true
	}
	return false
}

func PopSliceElement(elements []string, element string) ([]string, bool) {
	result := make([]string, 0, len(elements))
	pop := false
	for i := range elements {
		if elements[i] == element {
			pop = true
			continue
		}
		result = append(result, elements[i])
	}
	return result, pop
}

// GetPodSlaveIP GetPodSlaveIP
func GetPodSlaveIP(pod *corev1.Pod) string {
	if pod.Spec.NodeName == "" {
		return ""
	}
	if pod.Status.HostIP != "" {
		return pod.Status.HostIP
	}
	for i := range pod.Spec.Containers {
		for j := range pod.Spec.Containers[i].Env {
			if pod.Spec.Containers[i].Env[j].Name == "HIPPO_SLAVE_IP" {
				return pod.Spec.Containers[i].Env[j].Value
			}
		}
	}
	return ""
}

// GetPodSlotID GetPodSlotID
func GetPodSlotID(pod *corev1.Pod) (int32, bool) {
	if pod.Spec.NodeName == "" {
		return 0, false
	}
	slotIDStr := pod.Annotations[AnnotationKeySlotID]
	if slotIDStr == "" {
		for i := range pod.Spec.Containers {
			found := false
			for j := range pod.Spec.Containers[i].Env {
				if pod.Spec.Containers[i].Env[j].Name == "HIPPO_SLOT_ID" {
					slotIDStr = pod.Spec.Containers[i].Env[j].Value
					found = true
					break
				}
			}
			if found {
				break
			}
		}
	}
	if slotIDStr == "" {
		return 0, false
	}
	slotID, err := strconv.Atoi(slotIDStr)
	if nil != err {
		glog.Errorf("parse slot id error %s, %s, %v", pod.Name, slotIDStr, err)
	}
	return int32(slotID), true
}

// GetPodRecycleMode get pod recycle mod
func GetPodRecycleMode(pod *corev1.Pod) string {
	if pod != nil {
		return pod.Labels[LabelServerlessRecycleModel]
	}
	return ""
}

func PredictInfoEqual(before, after map[int64]int64) bool {
	if len(before) != len(after) {
		return false
	}
	if len(before) == 0 {
		return true
	}
	for hours, replicas := range before {
		if after[hours] != replicas {
			return false
		}
	}
	return true
}

func GetDeletionCost(replica *Replica) int64 {
	if replica == nil {
		return 0
	}
	return GetWorkerDeletionCost(&replica.WorkerNode)
}

func GetReplicaPoolType(replica *Replica) string {
	pool := ""
	return pool
}

func GetWorkerDeletionCost(worker *WorkerNode) int64 {
	statusCost := worker.Status.DeletionCost
	specCost := worker.Spec.DeletionCost
	if statusCost == 0 && specCost != 0 {
		return specCost
	}
	return statusCost
}

func MergeReplica(replicas ...[]*Replica) []*Replica {
	var res []*Replica
	for i := range replicas {
		for j := range replicas[i] {
			res = append(res, replicas[i][j])
		}
	}
	return res
}

func SliceEquals(s1, s2 []int64) bool {
	if len(s1) != len(s2) {
		return false
	}
	for i := 0; i < len(s1); i++ {
		if s1[i] != s2[i] {
			return false
		}
	}
	return true
}

func GetSpotTargetFromAnnotation(rs *RollingSet) int {
	spot := rs.GetAnnotations()[AnnotationC2SpotTarget]
	if spot != "" {
		if cnt, err := strconv.ParseInt(spot, 10, 64); err == nil {
			return int(cnt)
		}
	}
	return 0
}

func IsQuotaNotEnough(w *WorkerNode) bool {
	if nil == w {
		return false
	}
	return w.Status.UnassignedReason == UnassignedReasonQuotaNotEnough
}

func GetCurrentPodUid(w *WorkerNode) string {
	return w.Spec.BackupOfPod.Uid
}

func GetNonVersionedKeys() (mapset.Set, error) {
	nonVersionedKeys := mapset.NewThreadUnsafeSet()
	if val, err := common.GetGlobalConfigVal(common.ConfNonVersionedKeys, ""); err != nil {
		return nil, err
	} else if val != "" {
		if err = json.Unmarshal([]byte(val), &nonVersionedKeys); err != nil {
			return nil, err
		}
	}
	for i := range StripVersionLabelKeys {
		nonVersionedKeys.Add(StripVersionLabelKeys[i])
	}
	return nonVersionedKeys, nil
}

func BackupOfPodNotEmpty(worker *WorkerNode) bool {
	return worker != nil && worker.Spec.BackupOfPod.Uid != "" && worker.Spec.BackupOfPod.Name != ""
}

func IsGeneralServiceReady(worker *WorkerNode) bool {
	for i := range worker.Status.ServiceConditions {
		condition := worker.Status.ServiceConditions[i]
		if condition.Type == ServiceGeneral && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func IsWorkerUpdateReady(worker *WorkerNode) bool {
	if nil == worker {
		return false
	}
	if !IsResVersionMatch(worker) {
		return false
	}
	if !worker.Status.ResourceMatch {
		return false
	}
	if NeedReclaimByService(worker) {
		return false
	}
	if IsWorkerReclaiming(worker) {
		return false
	}
	if IsStandByWorkerComplete(worker) {
		return true
	}
	return IsWorkerUpdateMatch(worker)
}

func IsWorkerUpdateMatch(worker *WorkerNode) bool {
	if nil == worker {
		return false
	}
	if !IsVersionMatch(worker) {
		return false
	}
	if !IsWorkerHealthInfoReady(worker) {
		return false
	}
	if !IsWorkerTypeReady(worker) {
		return false
	}
	if !IsWorkerHealth(worker) {
		return false
	}
	return true
}
