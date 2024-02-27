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

package healthcheck

import (
	"fmt"
	"time"

	glog "k8s.io/klog"

	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DefaultHealthChecker 结构体
type DefaultHealthChecker struct {
	helper Helper
}

// NewDefaultHealthChecker 创建一个新的NewDefaultHealthChecker
func NewDefaultHealthChecker(helper Helper) *DefaultHealthChecker {
	return &DefaultHealthChecker{helper: helper}
}

// 根据_healthChecker组装心跳请求， 判断workernode状态， 如果与apiserver返回不一致则修改
func (d *DefaultHealthChecker) doCheck(workernode *carbonv1.WorkerNode, config *carbonv1.HealthCheckerConfig, rollingset *carbonv1.RollingSet) *carbonv1.HealthCondition {
	needCheck, condition := needHealthCheck(workernode)
	if needCheck {
		condition = d.check(workernode)
		condition.Checked = true
	}
	updateConditionVersion(condition, workernode)
	if !IsEquals(condition, &workernode.Status.HealthCondition) {
		glog.Infof("sync worker node healthstatus, workernode name:%v, condition:%v, originCondition:%v", workernode.Name, condition, workernode.Status.HealthCondition)
		d.helper.Sync(workernode, condition)
	}

	return condition
}

func needHealthCheck(workernode *carbonv1.WorkerNode) (bool, *carbonv1.HealthCondition) {
	originalCondition := workernode.Status.HealthCondition
	// #536295, 切rr失败时worker.status其实还是hot，所以这时应该认为hot而有心跳
	if carbonv1.IsStandbyWorker(workernode) && !carbonv1.IsHotStandbyWorker(workernode) && !carbonv1.IsCurrentHotStandbyWorker(workernode) {
		return false, buildDefaultHealthCondition(&originalCondition, carbonv1.HealthUnKnown, "", "", 0, 0, time.Now())
	}
	// lost worker 不做检查
	if carbonv1.IsWorkerLost(workernode) {
		return false, &originalCondition
	}
	if workernode.Status.Phase != carbonv1.Running {
		if glog.V(4) {
			glog.Infof("worker phase %s,%s,%s", workernode.Name, workernode.Status.IP, workernode.Status.Phase)
		}
	}
	switch workernode.Status.Phase {
	case carbonv1.Running:
		if !workernode.Status.ProcessReady {
			return false, buildDefaultHealthCondition(&originalCondition, carbonv1.HealthUnKnown, "", "", 0, 0, time.Now())
		}
		return true, &originalCondition
	case carbonv1.Terminated:
		if carbonv1.IsDaemonWorker(workernode) {
			return false, buildDefaultHealthCondition(&originalCondition, carbonv1.HealthDead, "", "worker phase is terminated", 0, 0, time.Now())
		}
		return false, buildDefaultHealthCondition(&originalCondition, carbonv1.HealthAlive, "", "worker phase is terminated", 0, 0, time.Now())
	case carbonv1.Failed:
		return false, buildDefaultHealthCondition(&originalCondition, carbonv1.HealthDead, "", "worker phase is failed", 0, 0, time.Now())
	case carbonv1.Pending:
		fallthrough
	case carbonv1.Unknown:
		fallthrough
	default:
		return false, buildDefaultHealthCondition(&originalCondition, carbonv1.HealthUnKnown, "", "", 0, 0, time.Now())
	}
	//之前状态也是dead， 不做修改 //出现未分配就dead，应该是unknown 不做检查则不修改状态，
	//if originalCondition.Status == carbonv1.HealthDead {
	//}
	//之前状态不是dead， 修改状态
	//return false, buildDefaultHealthCondition(&originalCondition, carbonv1.HealthDead, "", "worker node allocStatus is not assigned or offlining", 0, 0, time.Now())
}

func (d *DefaultHealthChecker) check(workernode *carbonv1.WorkerNode) *carbonv1.HealthCondition {
	workerPhase := workernode.Status.Phase
	originalCondition := workernode.Status.HealthCondition
	originalStatus := originalCondition.Status

	curStatus := carbonv1.GetHealthStatusByPhase(workernode)

	if curStatus == originalStatus {
		//状态不变
		return &originalCondition
	}
	return buildDefaultHealthCondition(&originalCondition, curStatus, "", fmt.Sprintf("worker node phase is %v", workerPhase), 0, 0, time.Now())
}

func buildDefaultHealthCondition(healthCondition *carbonv1.HealthCondition, curStatus carbonv1.HealthStatus, msg, reason string, lostCount int32, lastLostTime int64, lastTransitionTime time.Time) *carbonv1.HealthCondition {
	newHealthCondition := &carbonv1.HealthCondition{}
	newHealthCondition.Type = carbonv1.DefaultHealth
	newHealthCondition.Status = curStatus
	newHealthCondition.LastLostTime = lastLostTime
	newHealthCondition.LostCount = lostCount
	newHealthCondition.Reason = reason
	newHealthCondition.Message = msg
	if nil == healthCondition || healthCondition.Status != curStatus {
		newHealthCondition.LastTransitionTime = metav1.NewTime(lastTransitionTime)
	} else {
		newHealthCondition.LastTransitionTime = healthCondition.LastTransitionTime
	}
	if curStatus != carbonv1.HealthAlive {
		newHealthCondition.WorkerStatus = carbonv1.WorkerTypeUnknow
	} else {
		newHealthCondition.WorkerStatus = carbonv1.WorkerTypeReady
	}

	return newHealthCondition
}

func (d *DefaultHealthChecker) getTypeName() string {
	return "DefaultHealthCheck"
}

// 更新healthCondition为当前节点的version状态
func updateConditionVersion(condition *carbonv1.HealthCondition, workerNode *carbonv1.WorkerNode) {
	if workerNode.Status.Version != workerNode.Status.HealthCondition.Version {
		// NOTE: when version changed, set Status Unknown, see: https://aone.alibaba-inc.com/v2/project/2001745/req/51455921
		condition.Status = carbonv1.HealthUnKnown
	}
	condition.Version = workerNode.Status.Version
}
