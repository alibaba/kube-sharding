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
	"errors"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/alibaba/kube-sharding/common/features"
	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	glog "k8s.io/klog"
)

// single worker scheduler
type workerAdjuster interface {
	adjust() (*carbonv1.WorkerNode, *carbonv1.WorkerNode, bool, error)
}

func newDefaultAdjuster(worker, pair *carbonv1.WorkerNode, pod *corev1.Pod, c *Controller) workerAdjuster {
	return &defaultAdjuster{
		worker: worker,
		pair:   pair,
		pod:    pod,
		c:      c,
	}
}

type defaultAdjuster struct {
	c      *Controller
	worker *carbonv1.WorkerNode
	pair   *carbonv1.WorkerNode
	pod    *corev1.Pod
}

func isWorkersSame(current, backup *carbonv1.WorkerNode) bool {
	return nil != current && nil != backup &&
		(backup.Spec.Version == current.Spec.Version &&
			backup.Spec.ResVersion == current.Spec.ResVersion &&
			backup.Spec.UserDefVersion == current.Spec.UserDefVersion &&
			backup.Spec.CustomInfo == current.Spec.CustomInfo &&
			backup.Spec.CompressedCustomInfo == current.Spec.CompressedCustomInfo &&
			isWorkerRowCompleteSame(backup, current) &&
			carbonv1.IsWorkerDependencyReady(backup) == carbonv1.IsWorkerDependencyReady(current) &&
			backup.Spec.WorkerMode == current.Spec.WorkerMode &&
			carbonv1.GetQuotaID(current) == carbonv1.GetQuotaID(backup) &&
			carbonv1.GetGangPlan(current) == carbonv1.GetGangPlan(backup) &&
			carbonv1.GetGangInfo(current) == carbonv1.GetGangInfo(backup))
}

func isWorkerRowCompleteSame(current, backup *carbonv1.WorkerNode) bool {
	if current.Spec.RowComplete == nil && backup.Spec.RowComplete == nil {
		return true
	}
	if current.Spec.RowComplete == nil || backup.Spec.RowComplete == nil {
		return false
	}
	return *current.Spec.RowComplete == *backup.Spec.RowComplete
}

func syncWorkers(current, backup *carbonv1.WorkerNode) {
	var new, old *carbonv1.WorkerNode
	if current.Spec.OwnerGeneration >= backup.Spec.OwnerGeneration {
		new = current
		old = backup
	} else {
		new = backup
		old = current
	}
	old.Spec.ResVersion = new.Spec.ResVersion
	old.Spec.Version = new.Spec.Version
	old.Spec.VersionPlan = new.Spec.VersionPlan
	old.Spec.WorkerMode = new.Spec.WorkerMode
	old.Spec.DependencyReady = new.Spec.DependencyReady
	old.Spec.CompressedCustomInfo = new.Spec.CompressedCustomInfo
	old.Spec.Cm2TopoInfo = new.Spec.Cm2TopoInfo
	carbonv1.SetQuotaID(old, carbonv1.GetQuotaID(new))
	carbonv1.SetGangInfo(old, carbonv1.GetGangInfo(new))
	if carbonv1.IsGangMainPart(new.Spec.Template) {
		carbonv1.SetGangPlan(old, carbonv1.GetGangPlan(new))
	} else {
		delete(old.Annotations, carbonv1.AnnoGangPlan)
	}
	for k, v := range new.Labels {
		if k == carbonv1.WorkerRoleKey || k == carbonv1.DefaultWorkernodeUniqueLabelKey {
			continue
		}
		old.Labels[k] = v
	}
}

func syncBackupState(current, backup *carbonv1.WorkerNode) {
	if backup == nil || carbonv1.IsWorkerToRelease(backup) || carbonv1.IsWorkerComplete(current) {
		carbonv1.RemoveEvictFailedMsg(current)
		return
	}
	state := carbonv1.GetWorkerStatus(backup)
	carbonv1.SetEvictFailedMsg(current, state)
}

func couldSwapWorkers(current, backup *carbonv1.WorkerNode) bool {
	if current.Spec.RecoverStrategy == carbonv1.DirectReleasedRecoverStrategy {
		return false
	}
	return (carbonv1.IsWorkerComplete(backup) || (carbonv1.IsWorkerIrrecoverable(current) && carbonv1.IsWorkerHealthInfoReady(backup) && carbonv1.IsWorkerServiceOffline(backup))) &&
		(nil == current || !carbonv1.IsWorkerComplete(current)) && (isWorkersSame(current, backup) || carbonv1.IsWorkerIrrecoverable(current))
}

func (a *defaultAdjuster) adjust() (*carbonv1.WorkerNode, *carbonv1.WorkerNode, bool, error) {
	var err error
	current, backup, toRelease := carbonv1.GetCurrentWorkerNodeV2([]*carbonv1.WorkerNode{a.worker, a.pair}, false)
	if nil == current {
		err := fmt.Errorf("no current worker :%s", a.worker.Name)
		return a.worker, a.pair, false, err
	}
	if carbonv1.IsWorkerComplete(current) {
		giveBackRecoverQuota(a.worker)
	}
	if nil != a.pair && nil == backup {
		glog.Warningf("do not find backup %s", a.worker.Name)
	}
	if nil != backup && !isWorkersSame(current, backup) {
		glog.Infof("sync worker :%s, %s", current.Name, backup.Name)
		syncWorkers(current, backup)
		return a.worker, a.pair, true, nil
	}
	syncBackupState(current, backup)
	if toRelease {
		diffLogger.Log(diffLogKey(a.worker.Name, "DoRelease"), "release current and backup worker")
		carbonv1.SetWorkerToRelease(a.worker)
		carbonv1.SetWorkerToRelease(a.pair)
		giveBackRecoverQuota(a.worker)
	}
	var targetVersion string
	if len(a.worker.OwnerReferences) == 1 && a.worker.OwnerReferences[0].Kind == "RollingSet" {
		rs, _ := a.c.rollingSetLister.RollingSets(a.worker.Namespace).Get(a.worker.OwnerReferences[0].Name)
		if rs != nil {
			targetVersion = rs.Spec.Version
		}
	}
	currentBadReason := carbonv1.GetWorkerBadState(current, targetVersion)
	carbonv1.SetBadReason(current, currentBadReason)
	if backup != nil {
		backupBadReason := carbonv1.GetWorkerBadState(backup, targetVersion)
		carbonv1.SetBadReason(backup, backupBadReason)
		if couldSwapWorkers(current, backup) {
			glog.Infof("swap backup worker :%s", backup.Name)
			carbonv1.SwapAndReleaseBackup(current, backup)
			recordLatencyMetric(failoverLatency, backup)
		}
		if backupBadReason != carbonv1.BadReasonNone || carbonv1.IsWorkerComplete(current) {
			glog.Infof("release backup worker.name :%s backupBad: %v, currentComplete: %v ,%s",
				backup.Name, backupBadReason != carbonv1.BadReasonNone, carbonv1.IsWorkerComplete(current), utils.ObjJSON(backup))
			timeout, releaseNow := a.shouldReleaseBackupNow(backup)
			if releaseNow {
				glog.Infof("release backup now :%s", backup.Name)
				carbonv1.ReleaseWorker(backup, true)
			}
			if currentBadReason == carbonv1.BadReasonNone && current.Labels != nil {
				delete(current.Labels, carbonv1.LabelKeyHippoPreference)
			}
			key := current.Namespace + "/" + current.Name
			a.c.EnqueueAfter(key, timeout)
			glog.Infof("release worker and enqueue pair :%s", key)
		} else {
			backup.Status.LastDeleteBackupTime = 0
		}
	}
	if currentBadReason != carbonv1.BadReasonNone {
		carbonv1.SetWorkerProhibit(current, true)
	}
	if shouldRecover(current, backup, targetVersion) {
		glog.Infof("bad worker : %s", current.Name)
		a.c.diffLogWorker(current)
		a.c.diffLogPod(a.pod)
		key := current.Namespace + "/" + current.Name
		err = hasRecoverQuota(current)
		if nil != err {
			a.c.EnqueueAfter(key, time.Second*90)
			return a.worker, a.pair, false, nil
		}
		if recoverWithoutCreatebackup(current, a.pod) {
			glog.Infof("release current worker for recover :%s", key)
			if current != nil && carbonv1.GetGangID(current) != "" {
				backup, err := createBackup(current, backup, a.pod)
				if nil != err {
					return a.worker, a.pair, false, err
				}
				carbonv1.SwapAndReleaseBackup(current, backup)
				backup.Annotations[carbonv1.AnnoBecomeCurrentTime] = time.Now().Format(time.RFC3339)
				a.pair = backup
			} else {
				carbonv1.ReleaseWorker(current, true)
			}
		} else {
			backup, err = createBackup(current, backup, a.pod)
			if nil != err {
				return a.worker, a.pair, false, err
			}
			a.c.Eventf(a.pod, corev1.EventTypeNormal, "CreateBackupPod", "Backup Pod created because of %s.", current.Status.BadReasonMessage)
			a.pair = backup
		}
	}
	return a.worker, a.pair, false, nil
}

const (
	defalutDelayDeleteBackupSeconds int64 = 30
)

func (a *defaultAdjuster) shouldReleaseBackupNow(backup *carbonv1.WorkerNode) (time.Duration, bool) {
	// 默认5min后重新入队，避免丢失事件
	timeout := time.Second * 0
	releaseNow := false
	delayDeleteBackupSeconds := backup.Spec.DelayDeleteBackupSeconds
	// 如果不是dead过的节点，就不可能是core， 不需要等太久
	if backup.Status.BadReason != carbonv1.BadReasonDead && 0 != delayDeleteBackupSeconds {
		delayDeleteBackupSeconds = defalutDelayDeleteBackupSeconds
	}
	// 服务正常的节点，为了服务平滑，给新节点一个预热时间，延迟30s删除。
	if carbonv1.IsWorkerServiceMatch(backup) {
		if 0 == delayDeleteBackupSeconds {
			delayDeleteBackupSeconds = defalutDelayDeleteBackupSeconds
		}
	}
	// 未分配的节点直接释放
	if carbonv1.IsWorkerUnAssigned(backup) {
		delayDeleteBackupSeconds = 0
	}
	if delayDeleteBackupSeconds <= 0 {
		releaseNow = true
	} else {
		if 0 == backup.Status.LastDeleteBackupTime {
			backup.Status.LastDeleteBackupTime = time.Now().Unix()
		}
		if time.Now().Unix()-backup.Status.LastDeleteBackupTime >= delayDeleteBackupSeconds {
			releaseNow = true
		} else {
			// 到delay时间后重新入队
			timeout = time.Second * time.Duration(delayDeleteBackupSeconds-(time.Now().Unix()-backup.Status.LastDeleteBackupTime)+10)
		}
	}
	return timeout, releaseNow
}

func recoverWithoutCreatebackup(current *carbonv1.WorkerNode, pod *corev1.Pod) bool {
	if nil == current {
		return false
	}
	if current.Spec.RecoverStrategy == carbonv1.DirectReleasedRecoverStrategy ||
		current.Spec.RecoverStrategy == carbonv1.RebuildDeadStrategy ||
		carbonv1.IsWorkerUnAssigned(current) ||
		carbonv1.IsWorkerIrrecoverable(current) ||
		isPodStartedWithError(current) {
		return true
	}
	// 3.0切换可选下老起新，避免资源不够
	if features.C2MutableFeatureGate.Enabled(features.UpdatePodVersion3WithRelease) && carbonv1.IsWorkerVersionMisMatch(current) &&
		(pod != nil && carbonv1.GetPodVersion(current) != carbonv1.GetPodVersion(pod)) {
		return true
	}
	return false
}

func isPodStartedWithError(worker *carbonv1.WorkerNode) bool {
	if nil == worker {
		return false
	}
	// 刚刚启动的worker，分配到节点，确没有IP，说明pod同步错误，需要换节点，限制时间是怕底层bug引起二层错误删pod
	return carbonv1.IsWorkerNotStarted(worker) && worker.Status.IP == "" && time.Now().Unix()-worker.CreationTimestamp.Time.Unix() < 1200
}

func shouldRecover(current, backup *carbonv1.WorkerNode, targetVersion string) bool {
	if nil == current || nil != backup {
		return false
	}

	if carbonv1.IsWorkerToRelease(current) {
		return false
	}
	switch current.Spec.RecoverStrategy {
	case carbonv1.NotRecoverStrategy:
		return false
	case carbonv1.RebuildDeadStrategy:
		if carbonv1.IsWorkerDead(current) {
			return true
		}
	default:
		if carbonv1.GetWorkerBadState(current, targetVersion) != carbonv1.BadReasonNone {
			return true
		}
	}
	return false
}

var errBackupExist = errors.New("backup worker already exist, can't add repeatly")
var errNoThisBackup = errors.New("no this backup , can't swap")
var defaultMaxFailedCount int32 = 5

func createBackup(current, backup *carbonv1.WorkerNode, pod *corev1.Pod) (worker *carbonv1.WorkerNode, err error) {
	if nil != backup {
		worker = backup
		glog.Warningf("add backup err %s,%v", backup.Name, errBackupExist)
		return
	}
	worker = current.DeepCopy()
	labels := worker.Labels
	refer := worker.OwnerReferences
	worker.ObjectMeta = metav1.ObjectMeta{Annotations: worker.Annotations}
	worker.Namespace = current.Namespace
	worker.OwnerReferences = refer
	workerName := getPairWorkerName(current.Name)
	if nil == worker.Spec.Selector {
		worker.Spec.Selector = &metav1.LabelSelector{}
	}
	carbonv1.AddWorkerSelectorHashKey(worker.Spec.Selector, workerName)
	carbonv1.AddWorkerUniqueLabelHashKey(labels, workerName)
	labels[carbonv1.WorkerRoleKey] = carbonv1.BackupWorkerKey
	delete(labels, carbonv1.LabelKeyHippoPreference)
	worker.Labels = labels
	worker.Name = workerName
	worker.Spec.Reclaim = false
	worker.Status = carbonv1.WorkerNodeStatus{}
	carbonv1.SetWorkerBackup(worker)
	if nil != pod {
		worker.Spec.BackupOfPod.Name = pod.Name
		worker.Spec.BackupOfPod.Uid = string(pod.UID)
	}
	carbonv1.FixNewCreateLazyV3PodVersion(worker)
	glog.Infof("add backup worker :%s", worker.Name)
	return
}

const (
	quotaErr = "require recover quota failed"
)

func giveBackRecoverQuota(current *carbonv1.WorkerNode) {
	if current != nil && carbonv1.GetGangID(current) != "" {
		rsKey := current.Labels[carbonv1.DefaultRollingsetUniqueLabelKey]
		defaultRecoverQuotas.givebackParallel(rsKey, carbonv1.GetReplicaID(current))
	}
}

func hasRecoverQuota(current *carbonv1.WorkerNode) (err error) {
	if carbonv1.IsWorkerDead(current) || (carbonv1.GetGangID(current) != "" && carbonv1.IsWorkerAssigned(current)) {
		quota := current.Spec.BrokenRecoverQuotaConfig
		if nil == quota || nil == quota.MaxFailedCount || *quota.MaxFailedCount == defaultMaxFailedCount { // 默认值
			quota = &carbonv1.BrokenRecoverQuotaConfig{
				MaxFailedCount: utils.Int32Ptr(int32(carbonv1.DefaultMaxFailedCount)),
				TimeWindow:     utils.Int32Ptr(int32(600)),
			}
		}
		rsKey := current.Labels[carbonv1.DefaultRollingsetUniqueLabelKey]
		if rsKey == "" {
			rsKey = current.Name
		}
		var required = false
		if carbonv1.GetGangID(current) != "" {
			required = defaultRecoverQuotas.requireParallel(rsKey, carbonv1.GetReplicaID(current), quota.MaxFailedCount)
		} else {
			required = defaultRecoverQuotas.require(rsKey, quota.MaxFailedCount, quota.TimeWindow)
		}
		if !required {
			err = fmt.Errorf("%s: %s: %s,%d,%d", quotaErr, rsKey, carbonv1.GetGangID(current), getInt32Value(quota.MaxFailedCount), getInt32Value(quota.TimeWindow))
			glog.Error(err)
			return
		}
	}
	return
}

func getInt32Value(ptr *int32) int32 {
	if nil == ptr {
		return 0
	}
	return *ptr
}
