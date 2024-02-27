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

package rollingset

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	mapset "github.com/deckarep/golang-set"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	"github.com/alibaba/kube-sharding/pkg/rollalgorithm"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	glog "k8s.io/klog"
)

const (
	// RevisionAnnotation is the revision annotation of a deployment's replica sets which records its rollout sequence
	RevisionAnnotation = "deployment.kubernetes.io/revision"
	// RevisionHistoryAnnotation maintains the history of all old revisions that a replica set has served for a deployment.
	RevisionHistoryAnnotation = "deployment.kubernetes.io/revision-history"
	// DesiredReplicasAnnotation is the desired replicas for a deployment recorded as an annotation
	// in its replica sets. Helps in separating scaling events from the rollout process and for
	// determining if the new replica set for a deployment is really saturated.
	DesiredReplicasAnnotation = "deployment.kubernetes.io/desired-replicas"
	// MaxReplicasAnnotation is the maximum replicas a deployment can have at a given point, which
	// is deployment.spec.replicas + maxSurge. Used by the underlying replica sets to estimate their
	// proportions in case the deployment has surge replicas.
	MaxReplicasAnnotation = "deployment.kubernetes.io/max-replicas"

	// RollbackRevisionNotFound is not found rollback event reason
	RollbackRevisionNotFound = "DeploymentRollbackRevisionNotFound"
	// RollbackTemplateUnchanged is the template unchanged rollback event reason
	RollbackTemplateUnchanged = "DeploymentRollbackTemplateUnchanged"
	// RollbackDone is the done rollback event reason
	RollbackDone = "DeploymentRollback"

	// Reasons for deployment conditions
	//
	// Progressing:

	// ReplicaUpdatedReason is added in a deployment when one of its replica sets is updated as part
	// of the rollout process.
	ReplicaUpdatedReason = "ReplicaUpdated"
	// FailedReplicaCreateReason is added in a deployment when it cannot create a new replica set.
	FailedReplicaCreateReason = "ReplicaCreateError"
	// NewReplicaReason is added in a deployment when it creates a new replica set.
	NewReplicaReason = "NewReplicaCreated"
	// FoundNewReplicaReason is added in a deployment when it adopts an existing replica set.
	FoundNewReplicaReason = "FoundNewReplica"
	// NewReplicaAvailableReason is added in a deployment when its newest replica set is made available
	// ie. the number of new pods that have passed readiness checks and run for at least minReadySeconds
	// is at least the minimum available pods that need to run for the deployment.
	NewReplicaAvailableReason = "NewReplicaAvailable"
	// TimedOutReason is added in a deployment when its newest replica set fails to show any progress
	// within the given deadline (progressDeadlineSeconds).
	TimedOutReason = "ProgressDeadlineExceeded"
	// PausedDeployReason is added in a deployment when it is paused. Lack of progress shouldn't be
	// estimated once a deployment is paused.
	PausedDeployReason = "RollingsetPaused"
	// ResumedDeployReason is added in a deployment when it is resumed. Useful for not failing accidentally
	// deployments that paused amidst a rollout and are bounded by a deadline.
	ResumedDeployReason = "RollingsetResumed"
	//
	// Available:

	// MinimumReplicasAvailable is added in a deployment when it has its minimum replicas required available.
	MinimumReplicasAvailable = "MinimumReplicasAvailable"
	// MinimumReplicasUnavailable is added in a deployment when it doesn't have the minimum required replicas
	// available.
	MinimumReplicasUnavailable = "MinimumReplicasUnavailable"

	//ReplicaMaxBatchSize rolling处理replica一次批量上限
	ReplicaMaxBatchSize = 4000
	//TimeFormat TimeFormat
	TimeFormat = "2006-01-02-15-04-05"
)

// getReplicasForRollingSet returns the list of Replicas that this RollingSet should manage.
func (c *Controller) getReplicasForRollingSet(rs *carbonv1.RollingSet) ([]*carbonv1.Replica, error) {
	return c.ResourceManager.ListReplicaForRS(rs)
}

// GetAvailableReplicaCount returns the number of available pods corresponding to the given replica sets.
func GetAvailableReplicaCount(replicaSet []*carbonv1.Replica) int32 {
	totalAvailableReplicas := int32(0)
	for _, replica := range replicaSet {
		if replica != nil && carbonv1.IsReplicaAvailable(replica) {
			totalAvailableReplicas++
		}
	}
	return totalAvailableReplicas
}

// GetTotalWorkers GetTotalWorkers
func GetTotalWorkers(replicaSet []*carbonv1.Replica) int32 {
	totalWorkers := int32(0)
	for _, replica := range replicaSet {
		if replica != nil {
			totalWorkers++
		}
		if replica.Backup.UID != "" {
			totalWorkers++
		}
	}
	return totalWorkers
}

// GetActiveReplicaCount returns the number of active pods corresponding to the given replica sets.
func GetActiveReplicaCount(replicaSet []*carbonv1.Replica) (activeReplicas, notAssignedReplicas, workerModeNotMatchReplicas int32) {
	for _, replica := range replicaSet {
		if replica != nil {
			if !carbonv1.IsStandbyReplica(replica) {
				activeReplicas++
			} else if !carbonv1.IsReplicaAssigned(replica) {
				notAssignedReplicas++
			}
			if carbonv1.IsWorkerModeMisMatch(&replica.WorkerNode) {
				workerModeNotMatchReplicas++
			}
		}
	}
	return
}

func getUnassignedWorkerCount(replicaSet []*carbonv1.Replica) map[string]int32 {
	var workers = []*carbonv1.WorkerNode{}
	for _, replica := range replicaSet {
		if replica != nil {
			workers = append(workers, &replica.WorkerNode)
			if replica.Backup.UID != "" {
				workers = append(workers, &replica.Backup)
			}
		}
	}
	var unassignedClassify = map[string]int32{}
	for i := range workers {
		if carbonv1.IsWorkerUnAssigned(workers[i]) {
			count := unassignedClassify[workers[i].Status.UnassignedMessage]
			unassignedClassify[workers[i].Status.UnassignedMessage] = count + 1
		}
	}
	return unassignedClassify
}

// GroupByStandbyCount return cold warm hot standby count
func GroupByStandbyCount(replicaSet []*carbonv1.Replica) (cold, warm, hot int32) {
	for _, replica := range replicaSet {
		if replica != nil && carbonv1.IsStandbyReplica(replica) {
			switch replica.Status.WorkerMode {
			case carbonv1.WorkerModeTypeCold:
				cold++
			case carbonv1.WorkerModeTypeWarm:
				warm++
			case carbonv1.WorkerModeTypeHot:
				hot++
			default:
			}
		}
	}
	return
}

// GetAllocatedReplicaCount returns the number of allocated pods corresponding to the given replica sets.
func GetAllocatedReplicaCount(replicaSet []*carbonv1.Replica) int32 {
	totalAllocatedReplicas := int32(0)
	for _, replica := range replicaSet {
		if replica != nil && carbonv1.IsReplicaAssigned(replica) {
			totalAllocatedReplicas++
		}
	}
	return totalAllocatedReplicas
}

// GetUpdatedReplicaCount 已更新的replica数
func GetUpdatedReplicaCount(replicaSet []*carbonv1.Replica, latestVersion string) int32 {
	totalUpdatedReplicas := int32(0)
	for _, replica := range replicaSet {
		if replica != nil {
			if replica.Spec.Version == latestVersion {
				totalUpdatedReplicas++
			}
		}
	}
	return totalUpdatedReplicas
}

// GetCompleteReplicaCount 已完成的replica数
func GetCompleteReplicaCount(replicaSet []*carbonv1.Replica) int32 {
	totalCompleteReplicas := int32(0)
	for _, replica := range replicaSet {
		if replica != nil {
			if carbonv1.IsReplicaComplete(replica) {
				totalCompleteReplicas++
			}
		}
	}
	return totalCompleteReplicas
}

// GetVersionCount 获取rollingset的版本数
func GetVersionCount(replicaSet []*carbonv1.Replica, latestVersion string) int32 {
	versionMap := make(map[string]struct{})
	versionMap[latestVersion] = struct{}{}
	for _, replica := range replicaSet {
		versionMap[replica.Spec.Version] = struct{}{}
	}
	return int32(len(versionMap))
}

// GetReleasingReplicaCount 获取rs释放中的replica数
func GetReleasingReplicaCount(replicaSet []*carbonv1.Replica) (int32, int32) {
	totalReleasingReplicas := int32(0)
	totalReleasingWorkers := int32(0)
	for _, replica := range replicaSet {
		if replica != nil {
			if carbonv1.IsReplicaReleasing(replica) {
				totalReleasingReplicas++
				totalReleasingWorkers++
			}
			if carbonv1.IsWorkerToRelease(&replica.Backup) {
				totalReleasingWorkers++
			}
		}
	}
	return totalReleasingReplicas, totalReleasingWorkers
}

// getResourceVersionStatusMap 返回resourceverison的状态
func getResourceVersionStatusMap(
	resourceVersionStatusMap map[string]*rollalgorithm.ResourceVersionStatus,
	replicaSet []*carbonv1.Replica, rollingSet *carbonv1.RollingSet,
) map[string]*rollalgorithm.ResourceVersionStatus {
	if nil == resourceVersionStatusMap {
		resourceVersionStatusMap = map[string]*rollalgorithm.ResourceVersionStatus{}
	}
	resourceVersionStatusMap[rollingSet.Spec.ResVersion] = &rollalgorithm.ResourceVersionStatus{
		ResourceVersion: rollingSet.Spec.ResVersion,
	}
	var newMap = map[string]*rollalgorithm.ResourceVersionStatus{}
	for _, replica := range replicaSet {
		resVersion := replica.Status.ResVersion
		versionStatus := newMap[resVersion]
		if nil != versionStatus {
			versionStatus.Replicas++
			if _, ok := carbonv1.IsReplicaServiceReady(replica); ok {
				versionStatus.AvailableReplicas++
			}
		} else {
			status := resourceVersionStatusMap[resVersion]
			if nil != status {
				newStatus := &rollalgorithm.ResourceVersionStatus{
					CPU:             status.CPU,
					ResourceVersion: status.ResourceVersion,
				}
				newStatus.Replicas++
				if _, ok := carbonv1.IsReplicaServiceReady(replica); ok {
					newStatus.AvailableReplicas++
				}
				newMap[resVersion] = newStatus
			}
		}
	}
	return newMap
}

// Signature 计算签名
func Signature(a interface{}) (string, error) {
	data, err := json.Marshal(a)
	if err != nil {
		return "", err
	}
	m := md5.New()
	m.Write(data)
	sign := hex.EncodeToString(m.Sum(nil))
	return sign, nil
}

// CalculateStatus calculates the latest status for the provided deployment by looking into the provided replica sets.
func CalculateStatus(rollingset *carbonv1.RollingSet, replicaSet []*carbonv1.Replica) (carbonv1.RollingSetStatus, int64) {
	availableReplicas := GetAvailableReplicaCount(replicaSet)
	totalReplicas := int32(len(replicaSet))
	unavailableReplicas := totalReplicas - availableReplicas
	activeReplicas, unAssignedStandbyReplicas, workerModeMisMatch := GetActiveReplicaCount(replicaSet)
	coldStandbys, warmStandbys, hotStandbys := GroupByStandbyCount(replicaSet)
	unassigned := getUnassignedWorkerCount(replicaSet)
	// If unavailableReplicas is negative, then that means the Deployment has more available replicas running than
	// desired, e.g. whenever it scales down. In such a case we should simply default unavailableReplicas to zero.
	if unavailableReplicas < 0 {
		unavailableReplicas = 0
	}
	timeout, groupVersionStatusMap := carbonv1.GetGroupVersionStatusMap(replicaSet)
	resVersionStatusMap := getResourceVersionStatusMap(
		rollingset.Status.ResourceVersionStatusMap, replicaSet, rollingset)
	status := carbonv1.RollingSetStatus{
		// TODO: Ensure that if we start retrying status updates, we won't pick up a new Generation value.
		Replicas: totalReplicas,
		Workers:  GetTotalWorkers(replicaSet),
		Complete: rollingset.Status.Complete,
		//TODO 下发的最新版本replica 数
		UpdatedReplicas:            GetUpdatedReplicaCount(replicaSet, rollingset.Spec.Version),
		ReadyReplicas:              GetCompleteReplicaCount(replicaSet),
		ActiveReplicas:             activeReplicas,
		StandbyReplicas:            totalReplicas - activeReplicas,
		ColdStandbys:               coldStandbys,
		WarmStandbys:               warmStandbys,
		HotStandbys:                hotStandbys,
		WorkerModeMisMatchReplicas: workerModeMisMatch,
		UnAssignedStandbyReplicas:  unAssignedStandbyReplicas,
		AvailableReplicas:          availableReplicas,
		UnavailableReplicas:        unavailableReplicas,
		UnAssignedWorkers:          unassigned,
		AllocatedReplicas:          GetAllocatedReplicaCount(replicaSet),
		VersionCount:               GetVersionCount(replicaSet, rollingset.Spec.Version),
		GroupVersionStatusMap:      groupVersionStatusMap,
		ResourceVersionStatusMap:   resVersionStatusMap,
		Version:                    rollingset.Spec.Version,
		UserDefVersion:             rollingset.Spec.UserDefVersion,
	}
	selector, err := metav1.LabelSelectorAsSelector(rollingset.Spec.Selector)
	if err == nil && selector != nil {
		status.LabelSelector = selector.String()
	}
	// Copy conditions one by one so we won't mutate the original object.
	conditions := rollingset.Status.Conditions
	for i := range conditions {
		status.Conditions = append(status.Conditions, conditions[i])
	}

	return status, timeout
}

// HasSmoothFinalizer 是否带了SmoothFinalizer
func HasSmoothFinalizer(rollingset *carbonv1.RollingSet) bool {
	finalizers := rollingset.GetFinalizers()
	for _, finalizer := range finalizers {
		if finalizer == carbonv1.FinalizerSmoothDeletion {
			return true
		}
	}
	return false
}

// CheckSum rollingset checksum
func CheckSum(rollingset *carbonv1.RollingSet) (string, error) {
	checkSum, err := utils.SignatureWithMD5(rollingset.Spec.VersionPlan, rollingset.Spec.SchedulePlan, rollingset.Spec.HealthCheckerConfig, rollingset.Labels)
	if err != nil {
		glog.Errorf("CheckSum error :%s,%v", rollingset.Name, err)
		return "", err
	}
	return checkSum, nil
}

// RecordRollingsetHistory 记录rollingset版本变更信息
func RecordRollingsetHistory(rollingset *carbonv1.RollingSet) {
	mask := syscall.Umask(0)
	defer syscall.Umask(mask)
	dir := "__carbon_record_dir__/" + rollingset.Name
	err := os.MkdirAll(dir, 0777)
	if err != nil {
		glog.Warning("RecordRollingsetHistory mkdir error", err)
		return
	}

	fileTime := time.Now().Format(TimeFormat)
	fileName := dir + "/" + fileTime + "_target"
	glog.Infof("RecordRollingsetHistory file:" + fileName)
	data, err := json.MarshalIndent(rollingset, "", "      ")
	if err != nil {
		glog.Warning("RecordRollingsetHistory error, rollingset:"+rollingset.Name+". ", err)
		return
	}
	err = ioutil.WriteFile(fileName, data, 0777)
	if err != nil {
		glog.Warning("RecordRollingsetHistory error, rollingset:"+rollingset.Name+". ", err)
	}
}

// CmpFunc ...
type CmpFunc func(prev, obj interface{}) (bool, error)

func diffLoggerCmpFunc() CmpFunc {
	return func(prev, obj interface{}) (bool, error) {
		oldRs := prev.(*carbonv1.RollingSet)
		newRs := obj.(*carbonv1.RollingSet)
		return oldRs.Spec.CheckSum != newRs.Spec.CheckSum, nil
	}
}

var generalAnnotationTags = []string{
	carbonv1.LabelKeyAppName,
	carbonv1.LabelKeyGroupName,
	carbonv1.LabelKeyRoleName,
	carbonv1.DefaultSubRSLabelKey,
	carbonv1.DefaultShortSubRSLabelKey,
	carbonv1.AnnotationC2Spread,
}

func copyGeneralAnnotations(src, dst map[string]string) {
	for i := range generalAnnotationTags {
		if value, ok := src[generalAnnotationTags[i]]; ok {
			dst[generalAnnotationTags[i]] = value
		}
	}
}

func compareGeneralAnnotations(src, dst map[string]string) bool {
	for i := range generalAnnotationTags {
		if getAnnotations(generalAnnotationTags[i], src) != getAnnotations(generalAnnotationTags[i], dst) {
			return false
		}
	}
	return true
}

func getAnnotations(key string, anno map[string]string) string {
	if nil == anno {
		return ""
	}
	return anno[key]
}

func copyGeneralMetas(replica *carbonv1.Replica, generalLabels, generalAnnotations map[string]string) {
	oldlabels := replica.Labels
	replica.Labels = carbonv1.CopyWorkerAndReplicaLabels(generalLabels, oldlabels)
	lazyUpdatePodV3(replica, oldlabels)
	copyGeneralAnnotations(generalAnnotations, replica.Annotations)
}

func lazyUpdatePodV3(replica *carbonv1.Replica, oldlabels map[string]string) {
	if carbonv1.IsLazyV3(replica) && oldlabels[carbonv1.LabelKeyPodVersion] == carbonv1.PodVersion3 && carbonv1.IsV25(replica) {
		carbonv1.SetPodVersion(replica, carbonv1.PodVersion3)
	}
}

func getVersionGeneration(version string) (int, error) {
	index := strings.Index(version, "-")
	if index == -1 {
		return 0, nil
	}
	generationChars := version[:index]
	iGeneration, err := strconv.Atoi(generationChars)
	return iGeneration, err
}

func isStandbyCanActive(rs *carbonv1.RollingSet, replica *carbonv1.Replica, workerMode carbonv1.WorkerModeType) bool {
	if rs == nil || replica == nil {
		return false
	}
	if (carbonv1.IsQuickOnline(rs) || replica.Spec.WorkerMode == carbonv1.WorkerModeTypeInactive || replica.Status.WorkerMode == carbonv1.WorkerModeTypeInactive) &&
		carbonv1.IsReplicaOnStandByResourcePool(replica) && !carbonv1.IsStandbyWorkerMode(workerMode) {
		return false
	}
	return true
}

func syncNonVersionedMetas(rs *carbonv1.RollingSet, replica *carbonv1.Replica, keys mapset.Set) {
	if rs == nil || replica == nil || rs.Spec.Template == nil || replica.Spec.Template == nil {
		return
	}
	labels := carbonv1.InitIfNil(rs.Spec.Template.Labels)
	annos := carbonv1.InitIfNil(rs.Spec.Template.Annotations)
	rLabels := carbonv1.InitIfNil(replica.Spec.Template.Labels)
	rAnnos := carbonv1.InitIfNil(replica.Spec.Template.Annotations)
	for _, k := range keys.ToSlice() {
		key := k.(string)
		if val, ok := labels[key]; ok {
			rLabels[key] = val
		} else {
			delete(rLabels, key)
		}
		if val, ok := annos[key]; ok {
			rAnnos[key] = val
		} else {
			delete(rAnnos, key)
		}
	}
	replica.Spec.Template.Labels = rLabels
	replica.Spec.Template.Annotations = rAnnos
}

func strSliceContains(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}
