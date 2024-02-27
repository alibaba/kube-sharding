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
	"strconv"
	"strings"
	"time"

	"k8s.io/kubernetes/pkg/util/slice"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	glog "k8s.io/klog"
	podutil "k8s.io/kubernetes/pkg/api/v1/pod"
)

// pod healthscore
const (
	HealthStatusOffline  = -1
	HealthStatusPsFailed = 1

	// for diff log
	historyKeyFileCount   = 20
	historyTotalFileCount = 2000
	diffLogMsgCache       = 20000
)

var (
	diffLogger *utils.DiffLogger
)

func init() {
	var err error
	diffLogger, err = utils.NewDiffLogger(diffLogMsgCache, func(msg string) { glog.Info(msg) }, historyKeyFileCount, historyTotalFileCount)
	if err != nil {
		panic(err)
	}
}

func diffLogKey(name string, evt string) string {
	return name + " : " + evt
}

func isPodOffline(status *carbonv1.ScheduleStatusExtend) bool {
	return status.HealthScore == HealthStatusOffline
}

func isHippoPodFailed(status *carbonv1.ScheduleStatusExtend) bool {
	return status.HealthScore == HealthStatusPsFailed
}

func isWorkerReady(worker *carbonv1.WorkerNode, pod *corev1.Pod) bool {
	if !carbonv1.IsDaemonPod(pod.Spec) && worker.Status.Phase == carbonv1.Terminated {
		return true
	}
	return podutil.IsPodReady(pod) && carbonv1.IsWorkerRunning(worker)
}

func isPodReady(worker *carbonv1.WorkerNode, pod *corev1.Pod) bool {
	if worker.Status.AllocStatus == carbonv1.WorkerLost {
		return false
	}
	if !carbonv1.IsDaemonPod(pod.Spec) && worker.Status.Phase == carbonv1.Terminated {
		return true
	}
	return podutil.IsPodReady(pod)
}

var conditionTypeNamingRegistered corev1.PodConditionType = "NamingRegistered"

func isPodC2NamingRegisteredReady(pod *v1.Pod) bool {
	if slice.ContainsString(pod.Finalizers, carbonv1.FinaliezrKeyC2SkylineRegister, nil) {
		return isPodConditionTrue(pod.Status, carbonv1.ConditionKeyC2Initialize)
	}
	return true
}

func isPodConditionTrue(status v1.PodStatus, conditionType corev1.PodConditionType) bool {
	_, condition := podutil.GetPodCondition(&status, conditionType)
	return condition != nil && condition.Status == v1.ConditionTrue
}

func mockSync(worker *carbonv1.WorkerNode, pod *corev1.Pod) {
	worker.Status.Phase = carbonv1.Running
	worker.Status.EntityName = pod.Name
	worker.Status.EntityAlloced = true
	worker.Status.ProcessReady = true
	worker.Status.ResourceMatch = true
	worker.Status.ProcessMatch = true
	worker.Status.ResVersion = worker.Spec.ResVersion
	worker.Status.ProcessMatch = true
	worker.Status.Version = worker.Spec.Version
}

func getWorkerProhibit(worker *carbonv1.WorkerNode) (preferValue string, prefer bool) {
	if carbonv1.IsDaemonSetWorker(worker) {
		return "", false
	}
	if nil != worker && nil != worker.Labels {
		pre, ok := worker.Labels[carbonv1.LabelKeyHippoPreference]
		return pre, ok
	}
	return "", false
}

func parseProhibits(prohibit string) (scope, op, ttl string, err error) {
	prohibitStrs := strings.Split(prohibit, "-")
	if 3 != len(prohibitStrs) {
		err = fmt.Errorf("error prohibit formart: %s", prohibit)
		return
	}
	scope = prohibitStrs[0]
	op = prohibitStrs[1]
	ttl = prohibitStrs[2]
	return
}

func stringToFloat64(src string) (float64, error) {
	if "" == src {
		return 0, nil
	}

	dst, err := strconv.ParseFloat(src, 64)
	return dst, err
}

func stringToInt64(src string) (int64, error) {
	if "" == src {
		return 0, nil
	}

	dst, err := strconv.ParseInt(src, 10, 64)
	return dst, err
}

func generateName(prefix, suffix string) string {
	return utils.AppendString(prefix, suffix)
}

func getPairWorkerName(currWorkerName string) string {
	replicaName := getReplicaName(currWorkerName)
	if generateName(replicaName, carbonv1.WorkerNameSuffixA) == currWorkerName {
		return generateName(replicaName, carbonv1.WorkerNameSuffixB)
	}
	return generateName(replicaName, carbonv1.WorkerNameSuffixA)
}

func getReplicaName(wrokerName string) string {
	replicaName := strings.TrimSuffix(wrokerName, carbonv1.WorkerNameSuffixB)
	replicaName = strings.TrimSuffix(replicaName, carbonv1.WorkerNameSuffixA)
	return replicaName
}

func reNewWorker(worker *carbonv1.WorkerNode) {
	worker.Spec.Releasing = false
	worker.Spec.Reclaim = false
	worker.Status = carbonv1.WorkerNodeStatus{}
	delete(worker.Labels, carbonv1.LabelKeyHippoPreference)
	carbonv1.SetWorkerUnAssigned(worker)
}

// EscapeGroupID escape groupID
func EscapeGroupID(groupID string) string {
	s := strings.ReplaceAll(groupID, "_", "-")
	s = strings.ReplaceAll(s, "/", "")
	s = strings.Trim(s, "-")
	return s
}

func getNotScheduleTime(pod *corev1.Pod) int {
	if pod == nil || pod.Spec.NodeName != "" {
		return 0
	}
	var scheduled = false
	for i := range pod.Status.Conditions {
		if pod.Status.Conditions[i].Type == corev1.PodScheduled {
			scheduled = true
			break
		}
	}
	if !scheduled {
		return int(time.Since(pod.CreationTimestamp.Time) / time.Second)
	}
	return 0
}

func syncRrResourceUpdateState(worker *carbonv1.WorkerNode) {
	state, exist := worker.Annotations[carbonv1.ColdResourceUpdateState]
	if exist && (state == carbonv1.ColdUpdateStateWaitingWarmAssgin || state == carbonv1.ColdUpdateStateWarmResourceReplace) {
		if state == carbonv1.ColdUpdateStateWaitingWarmAssgin {
			glog.Infof("worker is assigned, waiting for resource update")
			worker.Annotations[carbonv1.ColdResourceUpdateState] = carbonv1.ColdUpdateStateWarmResourceReplace
		}

		targetResVersion, existResVersion := worker.Annotations[carbonv1.ColdResourceUpdateTargetVersion]
		if existResVersion && targetResVersion == worker.Spec.ResVersion && carbonv1.IsWorkerComplete(worker) {
			glog.Infof("worker[%v] resource version[%v] is matched, waiting back to cold", worker.Name, targetResVersion)
		}
	}
}
