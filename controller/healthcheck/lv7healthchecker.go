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
	"bytes"
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/alibaba/kube-sharding/common"
	"github.com/alibaba/kube-sharding/pkg/ago-util/monitor/metricgc"
	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	"github.com/bluele/gcache"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	glog "k8s.io/klog"
)

// Lv7HealthChecker 心跳执行器
type Lv7HealthChecker struct {
	helper     Helper
	rollingSet *carbonv1.RollingSet
}

// NewLv7HealthChecker 创建一个
func NewLv7HealthChecker(helper Helper, rollingSet *carbonv1.RollingSet) *Lv7HealthChecker {
	return &Lv7HealthChecker{
		helper:     helper,
		rollingSet: rollingSet}
}

// 根据_healthChecker组装心跳请求， 判断workernode状态， 如果与apiserver返回不一致则修改
func (l *Lv7HealthChecker) doCheck(workernode *carbonv1.WorkerNode, config *carbonv1.HealthCheckerConfig, rollingset *carbonv1.RollingSet) *carbonv1.HealthCondition {
	start := time.Now()
	needLv7Check, condition := needHealthCheck(workernode)
	if needLv7Check {
		condition = l.query(workernode, config)
		if condition != nil {
			reportScheduleMetrics(workernode, condition)
		}
	}
	updateConditionVersion(condition, workernode)
	needUpdate := !IsEquals(&workernode.Status.HealthCondition, condition)
	if needUpdate || bool(glog.V(5)) {
		glog.Infof("Lv7HealthChecker checkout result workernode name: %s, %s, %s, %v, %d",
			workernode.GetQualifiedName(), workernode.Status.IP, &workernode.Status.HealthCondition, condition, time.Now().Sub(start).Nanoseconds()/1000000)
	}
	if needUpdate {
		l.helper.Sync(workernode, condition)
	}
	return condition
}

func (l *Lv7HealthChecker) query(workernode *carbonv1.WorkerNode, config *carbonv1.HealthCheckerConfig) *carbonv1.HealthCondition {
	if !isValidConfig(config) {
		err := errors.New("lv7 config invaild")
		_, healthCondition := transfer(nil, err, workernode, config)
		healthCondition.WorkerStatus = carbonv1.WorkerTypeReady
		return healthCondition
	}
	url := getQueryURL(workernode, config)
	ctx, cancel := context.WithTimeout(context.Background(), common.GetBatcherTimeout(time.Second*time.Duration(healthCheckHTTPTimeout)))
	defer cancel()
	response, err := l.excuteHTTPGetQuery(ctx, url)
	if response != nil {
		response.Body.Close()
	}
	if glog.V(5) {
		glog.Infof("excuteHTTPGetQuery url: %s, %s, response: %v", workernode.GetQualifiedName(), url, response)
	}
	if err != nil {
		glog.Warningf("excuteHTTPGetQuery %s, %s, err: %v", workernode.GetQualifiedName(), url, err)
	}
	// 获取最新worker
	worker, _ := l.helper.GetWorkerNode(workernode)
	if nil != worker {
		workernode = worker
	}
	_, healthCondition := transfer(response, err, workernode, config)
	healthCondition.WorkerStatus = carbonv1.WorkerTypeReady
	healthCondition.Checked = true
	return healthCondition
}

func isValidConfig(config *carbonv1.HealthCheckerConfig) bool {
	if config.Lv7Config == nil || config.Lv7Config.Path == "" {
		return false
	}
	return true
}

func getQueryURL(workernode *carbonv1.WorkerNode, healthCheckerConfig *carbonv1.HealthCheckerConfig) string {
	ip := workernode.Status.IP

	port := healthCheckerConfig.Lv7Config.Port
	checkPath := healthCheckerConfig.Lv7Config.Path
	if checkPath[0] != '/' {
		checkPath = "/" + checkPath
	}
	var buf bytes.Buffer
	buf.WriteString("http://")
	buf.WriteString(ip)
	buf.WriteString(":")
	buf.WriteString(port.String())
	buf.WriteString(checkPath)
	return buf.String()
}

func (l *Lv7HealthChecker) excuteHTTPGetQuery(ctx context.Context, url string) (*http.Response, error) {
	headers := map[string]string{"Content-Type": utils.HTTPContentTypeJSON, "method": "GET", "Connection": "close"}
	response, err := l.doGet(ctx, url, nil, headers)
	return response, err
}

func (l *Lv7HealthChecker) doGet(ctx context.Context, url string, req interface{}, headers map[string]string) (*http.Response, error) {
	start := time.Now()
	response, err := l.helper.GetHTTPClient().SendJSONRequest(ctx, "GET", url, nil, headers)
	elapsed := time.Since(start)
	costMiss := float64(elapsed.Nanoseconds() / 1000000)
	metricgc.WithLabelValuesSummary(Lv7HealthCheckHTTPLatency,
		carbonv1.GetObjectExtendScopes(l.rollingSet, strconv.FormatBool(err == nil))...,
	).Observe(costMiss)
	if response != nil {
		Lv7HealthCheckHTTPPV.WithLabelValues(
			carbonv1.GetObjectExtendScopes(l.rollingSet,
				strconv.FormatBool(err == nil),
				strconv.Itoa(response.StatusCode))...,
		).Inc()
	}
	return response, err
}

// 基于本次心跳加healthcondition信息计算healthstatus状态
func transfer(response *http.Response, err error, workernode *carbonv1.WorkerNode, config *carbonv1.HealthCheckerConfig) (bool, *carbonv1.HealthCondition) {
	originalCondition := workernode.Status.HealthCondition
	originalStatus := workernode.Status.HealthCondition.Status

	lastLostTime := originalCondition.LastLostTime
	lostCount := originalCondition.LostCount

	curTime := time.Now()
	curTimeSecond := curTime.Unix()

	var isSuccess bool //心跳请求成功
	var reason, msg string
	if response == nil {
		isSuccess = false
		msg = "response is nil"
	} else {
		isSuccess = response.StatusCode == 200
		msg = response.Status
	}
	if err != nil {
		reason = err.Error()
	}
	curStatus := carbonv1.HealthAlive
	if isSuccess {
		lastLostTime = 0
		lostCount = 0
	} else {
		lostCount++
	}
	if originalStatus == carbonv1.HealthAlive {
		if isSuccess {
			//状态仍然是alive，不需要更新
			return true, buildSimpleLv7HealthCondition(&originalCondition, msg, reason, lostCount)
		}

		if lostCount < config.Lv7Config.LostCountThreshold {
			//alive节点超过LostCountThreshold， 才变成lost
			return false, buildSimpleLv7HealthCondition(&originalCondition, msg, reason, lostCount)
		}
		curStatus = carbonv1.HealthLost
		lastLostTime = curTimeSecond
		return true, buildLv7HealthCondition(&originalCondition, curStatus, msg, reason, lostCount, lastLostTime, curTime)
	}

	if originalStatus == carbonv1.HealthUnKnown || originalStatus == "" {
		if isSuccess {
			curStatus = carbonv1.HealthAlive
			return true, buildLv7HealthCondition(&originalCondition, curStatus, msg, reason, lostCount, lastLostTime, curTime)
		}
		if lostCount < config.Lv7Config.LostCountThreshold {
			return false, buildSimpleLv7HealthCondition(&originalCondition, msg, reason, lostCount)
		}
		curStatus = carbonv1.HealthLost
		lastLostTime = curTimeSecond
		return true, buildLv7HealthCondition(&originalCondition, curStatus, msg, reason, lostCount, lastLostTime, curTime)
	}

	if originalStatus == carbonv1.HealthLost {
		if isSuccess {
			curStatus = carbonv1.HealthAlive
			return true, buildLv7HealthCondition(&originalCondition, curStatus, msg, reason, lostCount, lastLostTime, curTime)
		}
		if curTimeSecond-lastLostTime < config.Lv7Config.LostTimeout {
			return false, &originalCondition
		}
		curStatus = carbonv1.HealthDead
		return true, buildLv7HealthCondition(&originalCondition, curStatus, msg, reason, lostCount, lastLostTime, curTime)
	}

	if originalStatus == carbonv1.HealthDead {
		if isSuccess {
			curStatus = carbonv1.HealthAlive
			return true, buildLv7HealthCondition(&originalCondition, curStatus, msg, reason, lostCount, lastLostTime, curTime)
		}
		return false, &originalCondition
	}

	return false, &originalCondition
}

func buildLv7HealthCondition(healthCondition *carbonv1.HealthCondition, curStatus carbonv1.HealthStatus, msg, reason string, lostCount int32, lastLostTime int64, lastTransitionTime time.Time) *carbonv1.HealthCondition {
	newHealthCondition := &carbonv1.HealthCondition{}
	newHealthCondition.Type = carbonv1.Lv7Health
	newHealthCondition.Status = curStatus
	newHealthCondition.LastLostTime = lastLostTime
	newHealthCondition.LostCount = lostCount
	newHealthCondition.Reason = reason
	newHealthCondition.Message = msg

	if nil == healthCondition || healthCondition.Status != curStatus {
		newHealthCondition.LastTransitionTime = metav1.NewTime(lastTransitionTime)
	}

	return newHealthCondition
}

// 不变更status的HealthCondition
func buildSimpleLv7HealthCondition(healthCondition *carbonv1.HealthCondition, msg, reason string, lostCount int32) *carbonv1.HealthCondition {
	newHealthCondition := healthCondition.DeepCopy()
	newHealthCondition.Type = carbonv1.Lv7Health
	newHealthCondition.LostCount = lostCount
	newHealthCondition.Reason = reason
	newHealthCondition.Message = msg
	return newHealthCondition
}

func (l *Lv7HealthChecker) getTypeName() string {
	return "Lv7HealthCheck"
}

const (
	lruSize    = 1024 * 10
	lruTimeout = 10 * time.Minute
)

var updatePlanCache = gcache.New(lruSize).LRU().Build()

func reportScheduleMetrics(workernode *carbonv1.WorkerNode, condition *carbonv1.HealthCondition) {
	startTime := workernode.Spec.UpdatePlanTimestamp
	if startTime == 0 {
		return
	}
	shardgroupKey := carbonv1.GetShardGroupID(workernode)
	if shardgroupKey == "" {
		return
	}
	cacheKey := shardgroupKey + "/" + workernode.Spec.UserDefVersion
	cacheV, err := updatePlanCache.Get(cacheKey)
	//已发过指标
	if nil == err && nil != cacheV {
		return
	}
	latency := float64((time.Now().UnixNano() - startTime) / 1000000)
	if time.Duration(latency)*time.Millisecond > lruTimeout {
		return
	}
	if glog.V(4) {
		glog.Infof("reportScheduleMetrics cacheKey:%s, latency:%v, startTime:%v", cacheKey, latency, startTime)
	}
	SendTargetE2ELatency.WithLabelValues(
		carbonv1.GetClusterName(workernode),
		carbonv1.GetAppName(workernode),
		workernode.Labels[carbonv1.DefaultShardGroupUniqueLabelKey],
		string(condition.Status),
	).Set(latency)
	updatePlanCache.SetWithExpire(cacheKey, true, lruTimeout)
}
