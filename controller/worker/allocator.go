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
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"k8s.io/kubernetes/pkg/api/v1/pod"
	"k8s.io/kubernetes/pkg/apis/core"

	"k8s.io/apimachinery/pkg/types"

	"encoding/json"

	"github.com/alibaba/kube-sharding/common/features"
	"github.com/alibaba/kube-sharding/controller"
	"github.com/alibaba/kube-sharding/controller/util"
	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	"github.com/alibaba/kube-sharding/pkg/memkube/mem"
	"github.com/alibaba/kube-sharding/transfer"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	glog "k8s.io/klog"
	"k8s.io/kubernetes/pkg/util/slice"
)

// LabelKeys
const (
	SchedulerNameMock              = "mock"
	HippoPriorityClassPrefix       = "hippo-priority"
	C2DeleteProtectionFinalizer    = "protection-delete.pod.c2/service"
	c2RRCreatorGracefullyDeletePod = "c2-gracefully-delete-pod"
)

var (
	antFinalizer = []string{
		"pod.beta1.sigma.ali/cni-allocated",
		"finalizer.k8s.alipay.com/alibabacloud-cni",
	}
	protectionFinalizerRegexp = regexp.MustCompile(carbonv1.FinalizerPodProtectionFmt)
)

type workerAllocator interface {
	allocate() error
	delete() error
	syncPodSpec() error
	syncPodMetas() error
	syncWorkerStatus() error
	syncWorkerAllocStatus() error
}

const (
	recreateScheduler = "recreate"
	updateScheduler   = "update"
)

type podAllocator struct {
	schedulerType string
	// 基础成员
	target
	current

	parser      podStatusParser
	syncer      podSpecSyncer
	executor    executor
	pSpecMerger *util.PodSpecMerger
	c           *Controller
}

type target struct {
	worker *carbonv1.WorkerNode

	targetPod     *corev1.Pod
	hippoPodSpec  *carbonv1.HippoPodSpec
	podSpec       *corev1.PodSpec
	podSpecExtend *carbonv1.HippoPodSpecExtend

	// pod版本控制
	targetResourceVersion          string            // 目标资源版本，
	targetHippoPodVersion          string            // hippo pod version，
	targetProcessVersion           string            // 目标业务进程版本，
	targetPodVersion               string            // pod版本,用来避免重复更新
	targetContainersProcessVersion map[string]string // 容器目标业务进程版本
}

type current struct {
	currentPod *corev1.Pod
	// pod版本控制
	currentResourceVersion          string            // 当前生效的资源版本，
	currentHippoPodVersion          string            // hippo pod version，
	currentResourceMatched          bool              // 当前生效的资源匹配，
	currentProcessVersion           string            // 当前生效的业务进程版本，
	currentUpdatedResourceVersion   string            // 当前更新的资源版本，用来避免重复更新
	currentUpdatedPodVersion        string            // 当前更新的pod版本，用来避免重复更新
	currentContainersProcessVersion map[string]string // 容器当前业务进程版本
}

type podStatusParser interface {
	getPodIP(pod *corev1.Pod) (hostIP, podIP string)
	isPodReclaimed(pod *corev1.Pod) bool
	getProcessScore(pod *corev1.Pod) int32
	getSlotID(pod *corev1.Pod) carbonv1.HippoSlotID
	isPodProcessFailed(pod *corev1.Pod) bool
	getPackageStatus(pod *corev1.Pod) string
	getCurrentResourceVersions(pod *corev1.Pod) (currentResourceVersion string, resourceMatched bool)
	getCurrentProcessVersions(pod *corev1.Pod) (currentProcessVersion string, currentContainersProcessVersion map[string]string)
}

type podSpecSyncer interface {
	setProhibit(pod *corev1.Pod, prohibit string)
	computeResourceVersion(worker *carbonv1.WorkerNode, hippoPodSpec *carbonv1.HippoPodSpec, podSpecExtend *carbonv1.HippoPodSpecExtend) (string, error)
	computeProcessVersion(worker *carbonv1.WorkerNode, hippoPodSpec *carbonv1.HippoPodSpec, currentPod *corev1.Pod) (string, map[string]string, error)
	updateResource(t *target, currentPod *corev1.Pod) (updateAll bool, err error)
	updateProcess(t *target) error
	preSync(worker *carbonv1.WorkerNode, hippoPodSpec *carbonv1.HippoPodSpec, labels map[string]string) error
	postSync(t *target) error
	MergeWebHooks(targetPod *corev1.Pod, podSpec *corev1.PodSpec, currentPod *corev1.Pod) error
}

var _ workerAllocator = &podAllocator{}

// 初始化各种内部变量
func (a *podAllocator) init() error {
	a.targetPod = a.currentPod.DeepCopy()
	err := a.generateTargets()
	if nil != err {
		glog.Errorf("generatePodSpec with error %s, %v", a.worker.Name, err)
		return err
	}
	if nil != a.currentPod {
		a.currentResourceVersion, a.currentResourceMatched = a.parser.getCurrentResourceVersions(a.currentPod)
		a.currentProcessVersion, a.currentContainersProcessVersion = a.parser.getCurrentProcessVersions(a.currentPod)
		a.currentUpdatedResourceVersion = a.currentPod.Annotations[carbonv1.AnnotationC2ResourceVersion]
		a.currentUpdatedPodVersion = a.currentPod.Annotations[carbonv1.AnnotationC2PodVersion]
	}
	if glog.V(4) {
		glog.Infof("%s init pod allocator : target: %s,%s,%s, current: %s,%t,%s,%s", a.worker.Name, a.targetResourceVersion, a.targetProcessVersion, a.targetContainersProcessVersion,
			a.currentResourceVersion, a.currentResourceMatched, a.currentProcessVersion, a.currentContainersProcessVersion)
	}
	return nil
}

func (a *podAllocator) allocate() error {
	var err error
	err = a.syncer.postSync(&a.target)
	if nil != err {
		return err
	}
	carbonv1.InitAnnotationsLabels(&a.targetPod.ObjectMeta)
	a.targetPod.Annotations[carbonv1.AnnotationC2ResourceVersion] = a.targetResourceVersion
	a.targetPod.Annotations[carbonv1.AnnotationC2PodVersion] = a.targetPodVersion
	if a.worker.Spec.BackupOfPod.Uid != "" && a.worker.Spec.BackupOfPod.Name != "" {
		a.targetPod.Annotations[carbonv1.AnnotationBackupOf] = utils.ObjJSON(a.worker.Spec.BackupOfPod)
		a.targetPod.Labels[carbonv1.LabelIsBackupPod] = "true"
	}
	if err = a.pSpecMerger.InitC2DelcaredKeysAnno(a.targetPod, nil); err != nil {
		return err
	}
	if err = a.pSpecMerger.InitC2DelcaredEnvKeysAnno(a.targetPod, nil); err != nil {
		return err
	}
	if len(a.worker.Status.ServiceConditions) != 0 {
		if !slice.ContainsString(a.targetPod.Finalizers, C2DeleteProtectionFinalizer, nil) {
			a.targetPod.Finalizers = append(a.targetPod.Finalizers, C2DeleteProtectionFinalizer)
		}
	}
	a.simplifyUpdateAnnotations(a.targetPod)
	a.initResourcePool(a.targetPod)
	a.initDeletionCost(a.targetPod)
	a.initStandbyHours(a.targetPod)
	a.initNoMountCgroup(a.targetPod)
	a.updateWorkerInactive(&a.target)
	carbonv1.MapCopy(a.c.writeLabels, a.targetPod.Labels)
	a.currentPod, err = a.executor.createPod(a.targetPod)
	if nil != err {
		return err
	}
	carbonv1.SetWorkerEntity(a.worker, a.currentPod.Name, string(a.currentPod.UID))
	return nil
}

func (a *podAllocator) delete() error {
	carbonv1.SyncWorkerServiceHealthStatus(a.worker)
	if carbonv1.CouldDeletePod(a.worker) {
		if nil == a.currentPod {
			glog.Infof("deleted pod %s", a.worker.Name)
			podName, err := carbonv1.GetPodName(a.worker)
			if err != nil {
				var deletePod corev1.Pod
				deletePod.Name = podName
				deletePod.UID = ""
				err = a.executor.deletePod(&deletePod, false)
				if err != nil && !errors.IsNotFound(err) {
					return err
				}
			}
			a.worker.Status.IP = ""
			a.worker.Status.HostIP = ""
			a.worker.Status.ProcessScore = 0
			a.worker.Status.Phase = carbonv1.Terminated
			carbonv1.SetWorkerEntity(a.worker, "", "")
			return nil
		}
		glog.Infof("to delete pod %s", a.currentPod.Name)
		var err error
		preferValue, prefer := getWorkerProhibit(a.worker)
		if prefer {
			a.syncer.setProhibit(a.currentPod, preferValue)
		}
		err = a.executor.deletePod(a.currentPod, false)
		if nil != err && !errors.IsNotFound(err) {
			glog.Errorf("delete pod %s,%v", a.currentPod.Name, err)
			return err
		}
		return nil
	}
	return nil
}

func (a *podAllocator) couldNotUpdateInplace() bool {
	if len(a.currentPod.Spec.Containers) != len(a.targetPod.Spec.Containers) {
		glog.Infof("couldNotUpdateInplace %s,  current : %s \r\n target: %s",
			a.worker.Name, utils.ObjJSON(a.currentPod.Spec.Containers), utils.ObjJSON(a.currentPod.Spec.Containers))
		return true
	}
	return false
}

func (a *podAllocator) syncPodMetas() error {
	if a.worker == nil || a.currentPod == nil || a.worker.Spec.Template == nil {
		return nil
	}
	patch := &util.CommonPatch{PatchType: types.StrategicMergePatchType, PatchData: map[string]interface{}{}}
	a.addPatchForMetaSync(patch)
	a.addPatchForResourcePoolSync(patch)
	a.addPatchForDeletionCostSync(patch)
	a.addPatchForStandbyHoursSync(patch)
	a.addPatchForBizdetailSync(patch)
	a.addPatchForServerlessApp(patch)
	a.addPatchForUpdateStatus(patch)
	a.addPatchForInplaceUpdate(patch)
	if len(patch.PatchData) > 0 {
		return a.executor.patchPod(a.currentPod, patch)
	}
	return nil
}

func (a *podAllocator) addPatchForInplaceUpdate(patch *util.CommonPatch) {
	if !a.isPodProcessMatch() {
		if a.currentPod.Labels[carbonv1.LabelFinalStateUpgrading] == "true" {
			return
		}
		for _, finalizer := range a.currentPod.ObjectMeta.Finalizers {
			if protectionFinalizerRegexp.MatchString(finalizer) {
				patch.InsertLabel(carbonv1.LabelFinalStateUpgrading, "true")
				break
			}
		}
	} else if a.currentPod.Labels[carbonv1.LabelFinalStateUpgrading] != "" {
		patch.DeleteLabel(carbonv1.LabelFinalStateUpgrading)
	}
}

func (a *podAllocator) addPatchForServerlessApp(patch *util.CommonPatch) {
	appId := a.currentPod.Labels[carbonv1.LabelServerlessAppId]
	if appId == "" {
		return
	}
	var appStatus carbonv1.AppNodeStatus
	for _, condition := range a.worker.Status.ServiceConditions {
		if condition.Type != carbonv1.ServiceGeneral {
			continue
		}
		if condition.Reason == carbonv1.ReasonQueryError {
			return
		}
		var appNodeStatus []carbonv1.AppNodeStatus
		err := json.Unmarshal([]byte(condition.ServiceDetail), &appNodeStatus)
		if err != nil || len(appNodeStatus) != 1 {
			continue
		}
		if appId == appNodeStatus[0].AppId {
			appStatus = appNodeStatus[0]
		}
	}

	if appStatus.AppId == "" {
		// delete old app status
		if a.currentPod.Annotations[carbonv1.AnnoServerlessAppStatus] != "" {
			patch.DeleteAnnotation(carbonv1.AnnoServerlessAppStatus)
			newCond := corev1.PodCondition{
				Type:               carbonv1.ConditionServerlessAppReady,
				Status:             corev1.ConditionFalse,
				LastTransitionTime: metav1.Now(),
				Message:            "serverless app empty",
			}
			patch.UpdatePodCondition(newCond)
		}
		if a.currentPod.Annotations[carbonv1.AnnoServerlessAppPlanStatus] != "" {
			patch.DeleteAnnotation(carbonv1.AnnoServerlessAppPlanStatus)
		}
		return
	}
	statusStr := utils.ObjJSON(appStatus)
	planStatusStr := strconv.Itoa(int(appStatus.PlanStatus))
	if oldStatus := a.currentPod.Annotations[carbonv1.AnnoServerlessAppStatus]; oldStatus != statusStr {
		old := carbonv1.AppNodeStatus{}
		json.Unmarshal([]byte(oldStatus), &old)
		if old.StartedAt == appStatus.StartedAt && old.NeedBackup == appStatus.NeedBackup && old.ServiceStatus == appStatus.ServiceStatus &&
			old.AppId == appStatus.AppId && old.PlanStatus == carbonv1.ServerlessPlanFailed && appStatus.PlanStatus == carbonv1.ServerlessPlanInProgress {
			// When the app fails, it will continue to switch between "inProgress" and "failed". skip it
			return
		}
		patch.InsertAnnotation(carbonv1.AnnoServerlessAppStatus, statusStr)
		status := corev1.ConditionFalse
		if appStatus.PlanStatus == carbonv1.ServerlessPlanSuccess && appStatus.ServiceStatus == carbonv1.ServerlessServiceReady {
			status = corev1.ConditionTrue
		}
		if appStatus.PlanStatus == carbonv1.ServerlessPlanFailed {
			a.c.Eventf(a.currentPod, core.EventTypeWarning, "ServerlessAppFailed", "Deploy serverless app: %s failed", appId)
		}
		_, oldCond := pod.GetPodCondition(&a.currentPod.Status, carbonv1.ConditionServerlessAppReady)
		if oldCond == nil || oldCond.Status != status {
			newCond := corev1.PodCondition{
				Type:               carbonv1.ConditionServerlessAppReady,
				Status:             status,
				LastTransitionTime: metav1.Now(),
				Message:            fmt.Sprintf("ServerlessApp: %s, load status: %s", appStatus.AppId, status),
			}
			if status == corev1.ConditionTrue {
				a.c.Eventf(a.currentPod, core.EventTypeNormal, "ServerlessAppReady", "Deploy serverless app: %s succeed.", appId)
			}
			patch.UpdatePodCondition(newCond)
		}
	}
	if a.currentPod.Annotations != nil && a.currentPod.Annotations[carbonv1.AnnoServerlessAppPlanStatus] != planStatusStr {
		patch.InsertAnnotation(carbonv1.AnnoServerlessAppPlanStatus, planStatusStr)
	}
}

func (a *podAllocator) addPatchForMetaSync(patch *util.CommonPatch) {
	srcLabels := carbonv1.OverrideMap(a.worker.Labels, a.worker.Spec.Template.Labels)
	srcAnnos := carbonv1.OverrideMap(a.worker.Annotations, a.worker.Spec.Template.Annotations)
	prefixes := carbonv1.GetSyncMetaPrefix()
	for i := range prefixes {
		prefix := prefixes[i]
		if insert, remove := calcPatch(srcLabels, a.currentPod.Labels, prefix); len(insert) > 0 || len(remove) > 0 {
			for key, val := range insert {
				patch.InsertLabel(key, val)
			}
			for _, key := range remove {
				patch.DeleteLabel(key)
			}
		}
		if insert, remove := calcPatch(srcAnnos, a.currentPod.Annotations, prefix); len(insert) > 0 || len(remove) > 0 {
			for key, val := range insert {
				patch.InsertAnnotation(key, val)
			}
			for _, key := range remove {
				patch.DeleteAnnotation(key)
			}
		}
	}
}

func (a *podAllocator) addPatchForResourcePoolSync(patch *util.CommonPatch) {
	if a.worker.Spec.ResourcePool != "" && a.currentPod.Labels[carbonv1.LabelResourcePool] == "" {
		patch.InsertLabel(carbonv1.LabelResourcePool, a.worker.Spec.ResourcePool)
	}
}

func (a *podAllocator) addPatchForDeletionCostSync(patch *util.CommonPatch) {
	if cost := a.worker.Spec.DeletionCost; cost >= carbonv1.MinCyclicalCost && cost <= carbonv1.MaxFixedCost {
		podCost := int64(0)
		if podCostStr := a.currentPod.Annotations[carbonv1.AnnotationPodDeletionCost]; podCostStr != "" {
			podCost, _ = strconv.ParseInt(podCostStr, 10, 64)
		} else if c2CostStr := a.currentPod.Annotations[carbonv1.AnnotationPodC2DeletionCost]; c2CostStr != "" {
			podCost, _ = strconv.ParseInt(c2CostStr, 10, 64)
		}
		if podCost == 0 || (podCost != cost && podCost >= carbonv1.MinCyclicalCost && podCost <= carbonv1.MaxCyclicalCost) {
			patch.InsertAnnotation(carbonv1.AnnotationPodC2DeletionCost, strconv.FormatInt(cost, 10))
		}
	}
}

func (a *podAllocator) addPatchForStandbyHoursSync(patch *util.CommonPatch) {
	if a.worker.Spec.StandbyHours != "" {
		var hours []int64
		_ = json.Unmarshal([]byte(a.worker.Spec.StandbyHours), &hours)
		status := utils.ObjJSON(carbonv1.PodStandbyStatus{StandbyHours: hours})
		if podStatus := a.currentPod.Annotations[carbonv1.AnnotationStandbyStableStatus]; podStatus != status {
			patch.InsertAnnotation(carbonv1.AnnotationStandbyStableStatus, status)
		}
	}
}

func (a *podAllocator) addPatchForUpdateStatus(patch *util.CommonPatch) {
	updateAt := time.Now().Unix()
	oldStatus := a.currentPod.Annotations[carbonv1.AnnotationWorkerUpdateStatus]
	if oldStatus != "" {
		old := &carbonv1.UpdateStatus{}
		_ = json.Unmarshal([]byte(oldStatus), old)
		if old.TargetVersion == a.worker.Spec.Version ||
			carbonv1.IsVersionInnerMatch(old.TargetVersion, a.worker.Spec.Version) {
			updateAt = old.UpdateAt
		}
	}
	if record := carbonv1.GetBufferSwapRecord(a.worker); record != "" {
		swapRecord := carbonv1.BufferSwapRecord{}
		json.Unmarshal([]byte(record), &swapRecord)
		if swapRecord.SwapAt > updateAt {
			updateAt = swapRecord.SwapAt
		}
	}
	status := carbonv1.UpdateStatus{
		Phase:         a.computeWorkerUpdatePhase(),
		TargetVersion: a.worker.Spec.Version,
		CurVersion:    a.worker.Status.Version,
		HealthStatus:  a.worker.Status.HealthStatus,
		UpdateAt:      updateAt,
	}
	updateStatus := utils.ObjJSON(status)
	if updateStatus != oldStatus {
		patch.InsertAnnotation(carbonv1.AnnotationWorkerUpdateStatus, updateStatus)
	}
	oldPhase := a.currentPod.Annotations[carbonv1.AnnotationWorkerUpdatePhase]
	if strconv.Itoa(int(status.Phase)) != oldPhase {
		patch.InsertAnnotation(carbonv1.AnnotationWorkerUpdatePhase, strconv.Itoa(int(status.Phase)))
		cStatus := corev1.ConditionFalse
		if status.Phase == carbonv1.PhaseReady {
			cStatus = corev1.ConditionTrue
		}
		if status.Phase == carbonv1.PhaseFailed {
			a.c.Eventf(a.currentPod, core.EventTypeNormal, "WorkerNodeFailed", "Deploy workerNode failed.")
		}
		if oldPhase == strconv.Itoa(int(carbonv1.PhaseEvicting)) && status.Phase == carbonv1.PhaseDeploying {
			a.c.Eventf(a.currentPod, core.EventTypeNormal, "LocalServiceEmpty", "Successfully offline local service.")
		}
		if status.Phase == carbonv1.PhaseEvicting {
			a.c.Eventf(a.currentPod, core.EventTypeNormal, "OffliningLocalService", "Offlining local service.")
		}
		_, oldCond := pod.GetPodCondition(&a.currentPod.Status, carbonv1.ConditionWorkerNodeReady)
		if oldCond == nil || oldCond.Status != cStatus {
			newCond := corev1.PodCondition{
				Type:               carbonv1.ConditionWorkerNodeReady,
				Status:             cStatus,
				LastTransitionTime: metav1.Now(),
				Message:            fmt.Sprintf("WorkerNode load status: %s", cStatus),
			}
			if cStatus == corev1.ConditionTrue {
				a.c.Eventf(a.currentPod, core.EventTypeNormal, "WorkerNodeReady", "Deploy workerNode succeed.")
			}
			patch.UpdatePodCondition(newCond)
		}
	}
}

func (a *podAllocator) computeWorkerUpdatePhase() carbonv1.UpdatePhase {
	if carbonv1.IsWorkerIrrecoverable(a.worker) || carbonv1.IsWorkerDead(a.worker) {
		return carbonv1.PhaseFailed
	}
	if carbonv1.IsPodReclaimed(a.currentPod) {
		return carbonv1.PhaseReclaim
	}
	if carbonv1.IsWorkerToRelease(a.worker) {
		if carbonv1.IsGeneralServiceReady(a.worker) {
			return carbonv1.PhaseEvicting
		}
		return carbonv1.PhaseStopping
	}
	if a.worker.Status.Version == a.worker.Spec.Version {
		if carbonv1.IsWorkerUpdateReady(a.worker) {
			return carbonv1.PhaseReady
		}
		return carbonv1.PhaseDeploying
	}
	if a.worker.Status.Version != a.worker.Spec.Version {
		if a.worker.Status.ServiceOffline && carbonv1.IsGeneralServiceReady(a.worker) {
			return carbonv1.PhaseEvicting
		}
		return carbonv1.PhaseDeploying
	}
	return carbonv1.PhaseUnknown
}

func (a *podAllocator) addPatchForBizdetailSync(patch *util.CommonPatch) {
	srcAnnos := carbonv1.OverrideMap(a.worker.Annotations, a.worker.Spec.Template.Annotations)
	for k := range srcAnnos {
		if strings.HasPrefix(k, carbonv1.BizDetailKeyPrefix) {
			if val, ok := a.currentPod.Annotations[k]; !ok || val != srcAnnos[k] {
				val := srcAnnos[k]
				patch.InsertAnnotation(k, val)
			}
		}
	}
	if a.worker.Status.Complete && a.currentPod.Annotations[carbonv1.BizDetailKeyReady] != "true" {
		patch.InsertAnnotation(carbonv1.BizDetailKeyReady, "true")
	}
	if !a.worker.Status.Complete && a.currentPod.Annotations[carbonv1.BizDetailKeyReady] != "false" {
		patch.InsertAnnotation(carbonv1.BizDetailKeyReady, "false")
	}
}

func calcPatch(src, target map[string]string, prefix string) (insert map[string]string, remove []string) {
	insert = map[string]string{}
	remove = []string{}
	for key, val := range src {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		if val != target[key] {
			insert[key] = val
		}
	}
	for key := range target {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		if _, ok := src[key]; !ok {
			remove = append(remove, key)
		}
	}
	return
}

func (a *podAllocator) syncPodSpec() (err error) {
	if nil == a.currentPod {
		return
	}
	if (!carbonv1.IsWorkerVersionMisMatch(a.worker) && !carbonv1.IsWorkerResVersionMisMatch(a.worker) &&
		!carbonv1.IsWorkerModeMisMatch(a.worker)) || (a.podUpdated() && a.resourceUpdated()) {
		a.worker.Status.InternalReclaim = false
		return
	}

	var updated = false
	var updateAll bool
	if !a.resourceUpdated() {
		if glog.V(4) {
			glog.Infof("updateResource %s", a.worker.Name)
		}
		if carbonv1.IsWorkerResVersionMisMatch(a.worker) || carbonv1.IsWorkerModeMisMatch(a.worker) {
			if carbonv1.IsQuickOnline(a.target.worker) {
				a.worker.Status.InternalReclaim = true
				glog.Infof("%s quick online worker update resource, set reclaim", a.worker.Name)
				return
			}
			updateAll, err = a.syncer.updateResource(&a.target, a.currentPod)
			if nil != err {
				glog.Errorf("updateResource %s with error : %v", a.worker.Name, err)
				return
			}
			updated = true
		}
		if updated {
			a.targetPod.Annotations[carbonv1.AnnotationC2ResourceVersion] = a.targetResourceVersion
			if nil != a.currentPod {
				carbonv1.SetIfExist(a.currentPod.Annotations, a.targetPod.Annotations, carbonv1.AnnotationC2PodVersion)
				carbonv1.SetIfExist(a.currentPod.Annotations, a.targetPod.Annotations, carbonv1.AnnotationC2DeclaredEnvKeys)
				if a.targetPod.Annotations[carbonv1.AnnotationC2DeclaredEnvKeys] == "" {
					if err = a.pSpecMerger.InitC2DelcaredEnvKeysAnno(a.targetPod, nil); err != nil {
						return err
					}
				}
			}
		}
	}
	// 资源目标到达后再更新pod
	if ((!carbonv1.IsWorkerResVersionMisMatch(a.worker) || updateAll || carbonv1.IsWorkerUnAssigned(a.worker)) &&
		carbonv1.IsWorkerVersionMisMatch(a.worker) && !a.podUpdated()) ||
		carbonv1.IsWorkerModeMisMatch(a.worker) {
		// 需要重启
		if !a.isPodProcessMatch() {
			a.worker.Status.InRestarting = true
			// 预热
			if features.C2MutableFeatureGate.Enabled(features.WarmupNewStartingPod) {
				glog.V(5).Infof("need warmup worker after restart %s", a.worker.Name)
				a.worker.Status.NeedWarmup = carbonv1.NeedWarmup(a.worker)
			}
			// 摘流
			if shouldGracefullyRestart(a.worker) {
				glog.V(5).Infof("need offline worker before restart %s", a.worker.Name)
				if !carbonv1.IsWorkerUnAvaliable(a.worker) && !a.worker.Status.ServiceOffline {
					glog.V(5).Infof("wait offline worker before restart %s", a.worker.Name)
					return
				}
			}
		}
		glog.V(4).Infof("updateProcess %s", a.worker.Name)
		err = a.syncer.updateProcess(&a.target)
		if nil != err {
			return err
		}
		err = a.syncer.postSync(&a.target)
		if nil != err {
			return err
		}
		if err := a.MergeEnvs(a.targetPod, a.podSpec, a.currentPod); err != nil {
			glog.Warningf("merge envs err:%v in updateProcess %s", err, a.worker.Name)
			return err
		}
		updated = true
		a.targetPod.Annotations[carbonv1.AnnotationC2ResourceVersion] = a.targetResourceVersion
		a.targetPod.Annotations[carbonv1.AnnotationC2PodVersion] = a.targetPodVersion
	}
	if !updated {
		return
	}
	err = a.MergeWebHooks(a.targetPod, a.podSpec, a.currentPod)
	if nil != err {
		glog.Warningf("merge web hooks err:%v in updateProcess %s", err, a.worker.Name)
		return err
	}
	err = a.syncer.MergeWebHooks(a.targetPod, a.podSpec, a.currentPod)
	if nil != err {
		glog.Warningf("merge web hooks err:%v in updateProcess %s", err, a.worker.Name)
		return err
	}

	if carbonv1.IsWorkerVersionMisMatch(a.worker) {
		if a.couldNotUpdateInplace() {
			a.worker.Status.InternalReclaim = true
			glog.Infof("could not update pod inplace, and recreate: %s", a.worker.Name)
			return
		}
	}
	a.worker.Status.InternalReclaim = false
	a.currentPod, err = a.executor.updatePod(a.worker, a.currentPod, a.targetPod)
	return
}

func (a *podAllocator) MergeWebHooks(targetPod *corev1.Pod, podSpec *corev1.PodSpec, currentPod *corev1.Pod) error {
	return a.pSpecMerger.MergeTargetPods(targetPod, a.podSpec, currentPod)
}

func (a *podAllocator) MergeEnvs(targetPod *corev1.Pod, podSpec *corev1.PodSpec, currentPod *corev1.Pod) error {
	return a.pSpecMerger.MergeEnvs(targetPod, podSpec, currentPod)
}

func (a *podAllocator) syncWorkerAllocStatus() error {
	if nil != a.currentPod && a.currentPod.Spec.SchedulerName == SchedulerNameMock {
		mockSync(a.worker, a.currentPod)
		return nil
	}
	carbonv1.InitWorkerNode(a.worker)
	a.worker.Status.NotScheduleSeconds = getNotScheduleTime(a.currentPod)
	if a.currentPod != nil {
		a.worker.Status.Reclaim = a.parser.isPodReclaimed(a.currentPod)
	}
	if a.currentPod == nil || a.currentPod.Spec.NodeName == "" {
		if a.currentPod == nil {
			carbonv1.SetWorkerEntity(a.worker, "", "")
		}
		glog.Infof("empty pod %s", a.worker.Name)
		phase := a.worker.Status.Phase
		if a.currentPod != nil && a.currentPod.Status.PodIP != "" && a.currentPod.Status.Phase == corev1.PodFailed {
			a.worker.Status.Phase = carbonv1.Failed
		}
		if phase != carbonv1.Failed && a.worker.Status.Phase == carbonv1.Failed {
			a.c.Eventf(a.currentPod, corev1.EventTypeWarning, "WorkerNodeFailed", "Failed to deploy workerNode.")
		}
		return nil
	}
	carbonv1.SetWorkerEntityAlloced(a.worker, a.currentPod.Spec.NodeName != "")
	if a.worker.Status.AllocStatus == carbonv1.WorkerAssigned && a.currentPod.DeletionTimestamp != nil {
		carbonv1.SetWorkerEntityAlloced(a.worker, false)
	}
	carbonv1.SetWorkerEntity(a.worker, a.currentPod.Name, string(a.currentPod.UID))
	a.worker.Status.HostIP, a.worker.Status.IP = a.parser.getPodIP(a.currentPod)
	if a.currentPod.Status.PodIP == "" {
		a.worker.Status.PodNotStarted = true
	}
	if a.schedulerType == recreateScheduler && !carbonv1.IsWorkerEntityAlloced(a.worker) && carbonv1.IsWorkerAssigned(a.worker) {
		a.worker.Spec.Releasing = false
		a.worker.Spec.Reclaim = false
		a.worker.Status = carbonv1.WorkerNodeStatus{}
		delete(a.worker.Labels, carbonv1.LabelKeyHippoPreference)
		carbonv1.SetWorkerUnAssigned(a.worker)
	}
	preferValue, prefer := getWorkerProhibit(a.worker)
	if prefer {
		glog.Infof("setProhibit %s, %s, %s", a.currentPod.Name, preferValue, a.currentPod.Status.HostIP)
		a.syncer.setProhibit(a.currentPod, preferValue)
	}
	return nil
}

func (a *podAllocator) syncWorkerStatus() error {
	//增加mock功能，压测apiserver
	if nil != a.currentPod && a.currentPod.Spec.SchedulerName == SchedulerNameMock {
		mockSync(a.worker, a.currentPod)
		return nil
	}
	a.syncWorkerAllocStatus()
	if nil == a.currentPod {
		return nil
	}
	a.worker.Status.ProcessScore = a.parser.getProcessScore(a.currentPod)
	a.syncWorkerResourceStatus()
	a.syncWorkerProcessStatus()
	a.syncWorkerPodReady()
	a.syncWorkerModeStatus()
	a.syncStandbyStableStatus()
	a.worker.Status.SlotID = a.parser.getSlotID(a.currentPod)
	a.worker.Status.Reclaim = a.parser.isPodReclaimed(a.currentPod)
	if glog.V(4) {
		glog.Infof("syncWorkerStatus worker status:%s", utils.ObjJSON(a.worker.Status))
	}
	return nil
}

func (a *podAllocator) syncWorkerResourceStatus() error {
	resourceAccepted, resourceMatched := a.isPodResourceMatch()
	// 同步 ResourceMatch resVersion
	if resourceAccepted {
		a.worker.Status.ResourceMatch = resourceMatched
		if !a.worker.Status.ResourceMatch && a.worker.Status.LastResourceNotMatchtime == 0 {
			a.worker.Status.LastResourceNotMatchtime = time.Now().Unix()
		} else if a.worker.Status.ResourceMatch {
			a.worker.Status.LastResourceNotMatchtime = 0
			a.worker.Status.ResVersion = a.worker.Spec.ResVersion
		}
	} else {
		a.worker.Status.ResourceMatch = false
	}
	if !carbonv1.IsWorkerResVersionMisMatch(a.worker) {
		a.worker.Status.ResourceMatch = true
	}

	diffLogger.Log(diffLogKey(a.worker.Name, "syncResourceStatus"), "targetResVer: %s, curResVer: %s, resourceMatch: %v, notMatchTime: %v",
		a.targetResourceVersion, a.currentResourceVersion, a.worker.Status.ResourceMatch, a.worker.Status.LastResourceNotMatchtime)
	return nil
}

func (a *podAllocator) isPodProcessFailed() bool {
	if nil == a.hippoPodSpec {
		return false
	}
	asiPod := a.isAsiPod(a.currentPod)
	for _, container := range a.hippoPodSpec.Containers {
		limit := 0
		if nil != container.Configs && 0 != container.Configs.RestartCountLimit {
			limit = container.Configs.RestartCountLimit
		}
		if limit == 0 && asiPod {
			limit = carbonv1.DefaultRestartCountLimit
		}
		if limit == 0 {
			continue
		}
		for _, status := range a.currentPod.Status.ContainerStatuses {
			if status.Name == container.Name && !status.Ready && nil != status.LastTerminationState.Terminated {
				if asiPod && carbonv1.ExceedRestartLimit(a.worker.Status.RestartRecords[container.Name], limit) {
					return true
				}
				if !asiPod && status.RestartCount >= int32(limit) {
					return true
				}
			}
		}
	}
	return false
}

func (a *podAllocator) syncWorkerPodReady() {
	a.worker.Status.PodReady = isPodReady(a.worker, a.currentPod)
}

func (a *podAllocator) syncWorkerProcessStatus() error {
	// 同步 ProcessMatch
	processMatched := a.isPodProcessMatch()
	a.worker.Status.Phase = carbonv1.TransPodPhase(a.currentPod, processMatched)
	carbonv1.AddRestartRecord(a.currentPod, a.worker)
	if a.isPodProcessFailed() || a.parser.isPodProcessFailed(a.currentPod) {
		a.worker.Status.Phase = carbonv1.Failed
	}
	a.worker.Status.PackageStatus = a.parser.getPackageStatus(a.currentPod)
	controller.RecordCompleteLatency(&a.currentPod.ObjectMeta, a.currentPod.Kind, controllerAgentName, "", a.worker.Status.ProcessReady)
	a.worker.Status.ProcessReady = isWorkerReady(a.worker, a.currentPod)
	if processMatched && a.podUpdated() {
		// not NeedRestartAfterResourceChange 或者 ResourceMatch version才匹配
		if !carbonv1.NeedRollingAfterResourceChange(&a.worker.Spec.VersionPlan) || a.worker.Status.ResourceMatch {
			a.worker.Status.Version = a.worker.Spec.Version
			a.worker.Status.UserDefVersion = a.worker.Spec.UserDefVersion
		}
	}
	if processMatched && carbonv1.IsWorkerRunning(a.worker) && isPodReady(a.worker, a.currentPod) {
		a.worker.Status.ProcessMatch = true
		a.worker.Status.LastProcessNotMatchtime = 0
	} else {
		a.worker.Status.ProcessMatch = false
		if a.worker.Status.LastProcessNotMatchtime == 0 {
			a.worker.Status.LastProcessNotMatchtime = time.Now().Unix()
		}
		diffLogger.Log(diffLogKey(a.worker.Name, "processNotMatch"), "set ProcessMatch=false, %s", utils.ObjJSON(&a.currentPod.Status))
	}

	if a.worker.Status.ProcessReady {
		a.worker.Status.LastProcessNotReadytime = 0
	} else if a.worker.Status.LastProcessNotReadytime == 0 {
		a.worker.Status.LastProcessNotReadytime = time.Now().Unix()
	}

	diffLogger.Log(diffLogKey(a.worker.Name, "processStatusResult"), "procMatch: %v, procReady: %v, target (procVer: %s, containerProcVer: %s), current (procVer: %s, containerProcVer: %s)",
		processMatched, a.worker.Status.ProcessReady, a.targetProcessVersion, a.targetContainersProcessVersion, a.currentProcessVersion, a.currentContainersProcessVersion)
	return nil
}

func (a *podAllocator) syncStandbyStableStatus() error {
	if nil == a.currentPod || nil == a.currentPod.Annotations {
		return nil
	}
	anno := a.currentPod.Annotations[carbonv1.AnnotationStandbyStableStatus]
	if anno != "" {
		var status carbonv1.PodStandbyStatus
		err := json.Unmarshal([]byte(anno), &status)
		if nil != err {
			glog.Errorf("parse AnnotationStandbyStableStatus err: %s, %s, %v", a.worker.Name, anno, err)
			return err
		}
		a.worker.Status.StandbyHours = utils.ObjJSON(status.StandbyHours)
	} else {
		a.worker.Status.StandbyHours = ""
	}
	cost := a.currentPod.Annotations[carbonv1.AnnotationPodDeletionCost]
	if cost != "" {
		costInt, err := strconv.ParseInt(cost, 10, 64)
		if err != nil {
			glog.Errorf("parse AnnotationPodDeletionCost err: %s, %s, %v", a.worker.Name, cost, err)
			return err
		}
		if costInt == 0 {
			glog.Warningf("pod deletion cost set default value: 0, worker: %s, val: %s", a.worker.Name, cost)
		}
		a.worker.Status.DeletionCost = costInt
	}
	return nil
}

func (a *podAllocator) syncWorkerModeStatus() error {
	return nil
}

func (a *podAllocator) isPodProcessMatch() bool {
	if a.isPodSimplifyed(a.currentPod, true) {
		if !carbonv1.IsWorkerVersionMisMatch(a.worker) || a.worker.Status.Version == "" || (a.podUpdated() && carbonv1.IsWorkerVersionMisMatch(a.worker)) {
			return true
		}
		return false
	}
	glog.V(4).Infof("isPodProcessMatch %s, %s, %s, %s, %s", a.worker.Name, a.targetProcessVersion, a.currentProcessVersion, a.targetContainersProcessVersion, a.currentContainersProcessVersion)
	if a.targetProcessVersion != a.currentProcessVersion {
		return false
	}
	// 去除注入的container version
	for k, v := range a.currentContainersProcessVersion {
		if v == "" && a.targetContainersProcessVersion[k] == "" {
			delete(a.currentContainersProcessVersion, k)
		}
	}
	if len(a.targetContainersProcessVersion) != len(a.currentContainersProcessVersion) {
		return false
	}
	for k, v := range a.targetContainersProcessVersion {
		if a.currentContainersProcessVersion[k] != v {
			return false
		}
	}
	return true
}

func (a *podAllocator) isPodResourceMatch() (accepted, matched bool) {
	accepted = a.targetResourceVersion == a.currentResourceVersion
	matched = accepted && a.currentResourceMatched
	return
}

func (a *podAllocator) generateTargets() error {
	hippoPodSpec, _, err := carbonv1.GetPodSpecFromTemplate(a.worker.Spec.Template)
	if nil != err {
		glog.Errorf("GetPodSpecFromTemplate error :%v", err)
		return err
	}
	a.addSuezWorkerEnv(a.worker, hippoPodSpec)
	a.resetWarmStandbyContainer(a.worker, hippoPodSpec)
	processVersion, containersProcessVersions, err := a.syncer.computeProcessVersion(a.worker, hippoPodSpec, a.currentPod)
	if nil != err {
		glog.Errorf("computeProcessVersion error :%s,%v", a.worker.Name, err)
		return err
	}

	podSpec, podSpecExtend, labels, annotations, err := a.generatePodSpec(a.worker, hippoPodSpec, processVersion, containersProcessVersions)
	if nil != err {
		glog.Errorf("generatePodSpec error :%s,%v", a.worker.Name, err)
		return err
	}

	resourceVersion, err := a.syncer.computeResourceVersion(a.worker, hippoPodSpec, podSpecExtend)
	if nil != err {
		glog.Errorf("computeResourceVersion error :%s,%v", a.worker.Name, err)
		return err
	}

	extend := utils.ObjJSON(podSpecExtend)
	podVersion, err := carbonv1.SignWorkerRequirementID(a.worker, extend)
	if nil != err {
		glog.Errorf("compute pod version error :%s,%v", a.worker.Name, err)
		return err
	}

	a.targetPodVersion = podVersion
	a.targetResourceVersion = resourceVersion
	a.targetProcessVersion = processVersion
	a.targetContainersProcessVersion = containersProcessVersions
	a.hippoPodSpec = hippoPodSpec
	a.podSpec = podSpec
	a.podSpecExtend = podSpecExtend
	if nil == a.targetPod {
		a.targetPod = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: a.worker.Namespace,
			},
			Spec: *a.podSpec,
		}
	}
	podName, err := carbonv1.GetPodName(a.worker)
	if nil != err {
		return err
	}
	if a.currentPod != nil {
		podName = a.currentPod.Name
	}
	a.targetPod.Name = podName
	a.targetPod.Labels = labels
	a.targetPod.Annotations = annotations
	a.clearLazyV3Metas(a.targetPod)
	a.setPodC2Labels(a.targetPod)
	a.updateWorkerMode(&a.target)
	err = mem.SetOwnerReferences(a.worker, controllerKind, a.targetPod)
	return err
}

func (a *podAllocator) clearLazyV3Metas(pod *corev1.Pod) {
	if carbonv1.IsLazyV3(pod) && carbonv1.IsV25(pod) {
		delete(pod.Labels, carbonv1.LabelKeyInstanceGroup)
		delete(pod.Labels, carbonv1.LabelKeyAppStage)
		delete(pod.Labels, carbonv1.LabelKeyAppUnit)
		delete(pod.Annotations, "pod.beta1.sigma.ali/hostname-template")
	}
}

func (a *podAllocator) generatePodSpec(worker *carbonv1.WorkerNode, hippoPodSpec *carbonv1.HippoPodSpec, version string, containersVersion map[string]string) (
	*corev1.PodSpec, *carbonv1.HippoPodSpecExtend, map[string]string, map[string]string, error) {
	general, err := carbonv1.GetGeneralTemplate(worker.Spec.VersionPlan)
	if nil != err {
		glog.Errorf("GetGeneralTemplate error %s,%v", worker.Name, err)
		return nil, nil, nil, nil, err
	}
	err = a.syncer.preSync(worker, hippoPodSpec, general.Labels)
	if nil != err {
		glog.Errorf("preSync error %s,%v", worker.Name, err)
		return nil, nil, nil, nil, err
	}
	podSpec, podSpecExtend, err := transfer.GeneratePodSpec(hippoPodSpec, version, containersVersion)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	appName := a.getAppName(worker, general)
	workerDir := a.getWorkerDir(worker, general)
	labels := carbonv1.CloneMap(worker.Labels)
	labels = util.MergeLabels(labels, general.Labels)
	if !features.C2MutableFeatureGate.Enabled(features.HashLabelValue) {
		labels[carbonv1.LabelKeyRoleName] = carbonv1.LabelValueHash(workerDir, false)
		labels[carbonv1.LabelKeyAppName] = carbonv1.LabelValueHash(appName, false)
	}
	annotations := carbonv1.GetTemplateAnnotationSet(general)
	annotations[carbonv1.LabelKeyRoleName] = carbonv1.LabelValueHash(workerDir, false)
	annotations[carbonv1.LabelKeyAppName] = carbonv1.LabelValueHash(appName, false)

	// https://yuque.antfin-inc.com/asidocs/wi0ad5/viq0hu sn 生成
	if _, ok := labels[carbonv1.LabelKeyInstanceGroup]; ok {
		annotations[carbonv1.AnnotationSN] = "true"
	}
	return podSpec, podSpecExtend, labels, annotations, nil
}

func (a *podAllocator) addSuezWorkerEnv(worker *carbonv1.WorkerNode, hippoPodSpec *carbonv1.HippoPodSpec) {
	if worker.Spec.Template == nil || worker.Spec.Template.Labels == nil || worker.Spec.Template.Labels[carbonv1.LabelKeyIdentifier] != "true" {
		return
	}
	for i := range hippoPodSpec.Containers {
		env := corev1.EnvVar{
			Name:  carbonv1.IdentifierEnvKey,
			Value: carbonv1.GenUniqIdentifier(worker),
		}
		hippoPodSpec.Containers[i].Env = append(hippoPodSpec.Containers[i].Env, env)
	}
}

// warm standby 切换状态时需要重启，用来重置内存
func (a *podAllocator) resetWarmStandbyContainer(worker *carbonv1.WorkerNode, hippoPodSpec *carbonv1.HippoPodSpec) error {
	if nil != hippoPodSpec && nil != worker && carbonv1.WorkerModeTypeWarm == worker.Spec.WorkerMode {
		for i := range hippoPodSpec.Containers {
			env := []corev1.EnvVar{
				{
					Name:  "IS_WARM_STANDBY",
					Value: "true",
				},
			}
			hippoPodSpec.Containers[i].Env = append(env, hippoPodSpec.Containers[i].Env...)
			if features.C2MutableFeatureGate.Enabled(features.ResetWarmStandbyCommand) {
				hippoPodSpec.Containers[i].Args = nil
				hippoPodSpec.Containers[i].Command = []string{"sleep", "1000d"}
			}
		}
	}
	return nil
}

func (a *podAllocator) getAppName(worker *carbonv1.WorkerNode, general *carbonv1.GeneralTemplate) string {
	appName := carbonv1.GetAppName(&general.ObjectMeta)
	if "" == appName {
		//从carbon-proxy发起的指令， carbon2协议过滤
		appName = carbonv1.GetAppName(worker)
	}
	return appName
}

func (a *podAllocator) getWorkerDir(worker *carbonv1.WorkerNode, general *carbonv1.GeneralTemplate) string {
	workerDir := carbonv1.GetRoleName(&general.ObjectMeta)
	if "" == workerDir {
		workerDir = carbonv1.GetRsID(worker)
		if roleName := carbonv1.GetRoleName(worker); roleName != "" {
			workerDir = roleName
		}
	}
	return workerDir
}

func (a *podAllocator) setPodC2Labels(pod *corev1.Pod) error {
	pod.Labels[carbonv1.LabelKeyPlatform] = "c2"
	return nil
}

func (a *podAllocator) resourceUpdated() bool {
	return a.currentUpdatedResourceVersion == a.targetResourceVersion &&
		a.targetHippoPodVersion == a.currentHippoPodVersion
}

func (a *podAllocator) podUpdated() bool {
	return a.currentUpdatedPodVersion == a.targetPodVersion
}

func (a *podAllocator) updateWorkerMode(t *target) {
}

func (a *podAllocator) updateWorkerInactive(t *target) {
	if t.worker.Spec.WorkerMode == carbonv1.WorkerModeTypeInactive {
		labels, annotations := carbonv1.GetInactiveInterpose(t.worker)
		carbonv1.MergeMap(labels, t.targetPod.Labels)
		carbonv1.MergeMap(annotations, t.targetPod.Annotations)
	}
}

func (a *podAllocator) initResourcePool(pod *corev1.Pod) {
	if a.worker.Spec.ResourcePool != "" {
		carbonv1.SetLabelsDefaultValue(carbonv1.LabelResourcePool, a.worker.Spec.ResourcePool, &pod.ObjectMeta)
	}
}

func (a *podAllocator) initDeletionCost(pod *corev1.Pod) {
	if cost := a.worker.Spec.DeletionCost; cost >= carbonv1.MinCyclicalCost && cost <= carbonv1.MaxFixedCost {
		pod.Annotations[carbonv1.AnnotationPodC2DeletionCost] = strconv.FormatInt(cost, 10)
	}
}

func (a *podAllocator) initStandbyHours(pod *corev1.Pod) {
	if a.worker.Spec.StandbyHours != "" {
		var hours []int64
		_ = json.Unmarshal([]byte(a.worker.Spec.StandbyHours), &hours)
		status := carbonv1.PodStandbyStatus{StandbyHours: hours}
		pod.Annotations[carbonv1.AnnotationStandbyStableStatus] = utils.ObjJSON(status)
	}
}

func (a *podAllocator) initNoMountCgroup(pod *corev1.Pod) {
	if features.C2FeatureGate.Enabled(features.DisableNoMountCgroup) && carbonv1.IsV3(pod) {
		pod.Labels[carbonv1.LabelKeyNoMountCgroup] = "false"
	}
}

func (a *podAllocator) simplifyUpdateAnnotations(pod *corev1.Pod) {
}

func (a *podAllocator) isPodSimplifyed(pod *corev1.Pod, process bool) bool {
	return false
}

func (a *podAllocator) isAsiPod(pod *corev1.Pod) bool {
	if pod.Labels != nil {
		podVersion := pod.Labels[carbonv1.LabelKeyPodVersion]
		return carbonv1.PodVersion3 == podVersion
	}
	return false
}

func (a *podAllocator) shouldReserve(worker *carbonv1.WorkerNode, pod *corev1.Pod, prefer bool) bool {
	if !features.C2FeatureGate.Enabled(features.DeletePodGracefully) {
		return false
	}
	if pod == nil {
		return false
	}
	if pod.Spec.NodeName == "" {
		return false
	}
	if prefer {
		return false
	}
	if a.parser.isPodReclaimed(pod) {
		return false
	}
	if worker != nil && worker.Status.IsFedWorker {
		return false
	}
	for i := range pod.Finalizers {
		if pod.Finalizers[i] == "protection-delete.pod.hippo/rr-set-controller-from-release" {
			return false
		}
	}
	if !a.currentResourceMatched {
		return false
	}
	return true
}

const (
	resourceVersionLabelKey = "c2-resource-version"
)

func createAllocator(worker *carbonv1.WorkerNode, pod *corev1.Pod, c *Controller) (workerAllocator, error) {
	var allocator = podAllocator{
		c: c,
	}
	var merger = &util.PodSpecMerger{}

	hippoPodSpec, _, err := carbonv1.GetPodSpecFromTemplate(worker.Spec.Template)
	if nil != err {
		glog.Errorf("GetPodSpecFromTemplate error :%v", err)
		return nil, err
	}
	syncerScheduler := hippoPodSpec.SchedulerName
	if syncerScheduler == "" || carbonv1.IsC2Scheduler(worker) {
		syncerScheduler = carbonv1.GetDefaultSchedulerName()
	}
	parserScheduler := syncerScheduler
	podVersion := carbonv1.GetPodVersion(worker)
	currentPodVersion := podVersion
	if pod != nil {
		if parserScheduler != carbonv1.SchedulerNameAck {
			parserScheduler = pod.Spec.SchedulerName
		}
		currentPodVersion = carbonv1.GetPodVersion(pod)
	}
	lazyV3 := carbonv1.IsLazyV3(worker)
	// lazy模式新建节点直接创建为3.0，存量pod不变
	if lazyV3 && podVersion == carbonv1.PodVersion25 && (pod == nil || currentPodVersion == carbonv1.PodVersion3) {
		podVersion = carbonv1.PodVersion3
		currentPodVersion = carbonv1.PodVersion3
		carbonv1.SetPodVersion(worker, carbonv1.PodVersion3)
	}
	allocator.syncer, allocator.parser = getSyncer(syncerScheduler, parserScheduler, podVersion, worker, pod)
	if syncerScheduler != parserScheduler || podVersion != currentPodVersion {
		parser := getPaser(syncerScheduler, parserScheduler, currentPodVersion, worker, pod)
		if nil != parser {
			allocator.parser = parser
		}
	}
	allocator.executor = getExecutor(syncerScheduler, parserScheduler, currentPodVersion, c)
	allocator.worker = worker
	allocator.currentPod = pod
	allocator.pSpecMerger = merger
	allocator.targetHippoPodVersion = podVersion
	allocator.currentHippoPodVersion = currentPodVersion
	err = allocator.init()
	if glog.V(4) {
		glog.Infof("%s create pod allocator, paser: %s, %T, syncer: %s, %T, %T, podVersion: %s, currentPodVersion: %s",
			worker.Name, parserScheduler, allocator.parser, syncerScheduler, allocator.syncer, allocator.executor, podVersion, currentPodVersion)
	}
	return &allocator, err
}

func getSyncer(syncerScheduler, parserScheduler, podVersion string, worker *carbonv1.WorkerNode, pod *corev1.Pod) (podSpecSyncer, podStatusParser) {
	var parser podStatusParser
	var syncer podSpecSyncer
	var podAllocator = &ackPodAllocator{}
	parser = podAllocator
	syncer = podAllocator
	return syncer, parser
}

func getPaser(syncerScheduler, parserScheduler, currentPodVersion string, worker *carbonv1.WorkerNode, pod *corev1.Pod) podStatusParser {
	var parser podStatusParser
	var podAllocator = &ackPodAllocator{}
	parser = podAllocator
	return parser
}

func getExecutor(syncerScheduler, parserScheduler, currentPodVersion string, c *Controller) executor {
	if syncerScheduler == carbonv1.SchedulerNameAck {
		return &recreateExecutor{
			baseExecutor: baseExecutor{
				controller: c,
				grace:      true,
			},
		}
	}
	executor := &updateExecutor{
		baseExecutor: baseExecutor{
			controller: c,
			grace:      true,
		},
	}
	if glog.V(4) {
		glog.Infof("scheduler: %s, currentPodVersion: %s, grace: %v", parserScheduler, currentPodVersion, executor.grace)
	}
	return executor
}
