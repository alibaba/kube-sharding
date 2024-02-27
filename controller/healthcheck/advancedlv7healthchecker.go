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
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/alibaba/kube-sharding/common"
	"github.com/alibaba/kube-sharding/common/features"
	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	glog "k8s.io/klog"
)

// Healthcheck meta keys
const (
	HealthcheckPayloadSignature        = "signature"
	HealthcheckPayloadCustominfo       = "customInfo"
	HealthcheckPayloadGlobalCustominfo = "globalCustomInfo"
	HealthcheckPayloadIdentifier       = "identifier"
	HealthcheckPayloadProcessVersion   = "porcessVersion"
	HealthcheckPayloadSchedulerInfo    = "schedulerInfo"
	HealthcheckPayloadPreload          = "preload"
)

// AdvancedLv7HealthChecker adv心跳执行器
type AdvancedLv7HealthChecker struct {
	helper     Helper
	rollingSet *carbonv1.RollingSet
}

// NewAdvancedLv7HealthChecker 创建一个
func NewAdvancedLv7HealthChecker(helper Helper, rollingSet *carbonv1.RollingSet) *AdvancedLv7HealthChecker {
	return &AdvancedLv7HealthChecker{
		helper:     helper,
		rollingSet: rollingSet,
	}
}

// 获取HealthChecker的typename
func (a *AdvancedLv7HealthChecker) getTypeName() string {
	return "AdvancedLv7HealthChecker"
}

// 根据_healthChecker组装心跳请求， 判断workernode状态， 如果与apiserver返回不一致则修改
func (a *AdvancedLv7HealthChecker) doCheck(workernode *carbonv1.WorkerNode, config *carbonv1.HealthCheckerConfig, rollingset *carbonv1.RollingSet) *carbonv1.HealthCondition {
	needLv7Check, condition := needHealthCheckAdvlv7(workernode)
	if needLv7Check {
		_, condition = a.query(workernode, config, rollingset)
		if condition != nil {
			reportScheduleMetrics(workernode, condition)
		}
	}
	if condition == nil {
		return nil
	}
	updateConditionVersion(condition, workernode)
	needUpdate := !IsEquals(&workernode.Status.HealthCondition, condition)
	if bool(glog.V(5)) || workernode.Status.HealthCondition.String() != condition.String() {
		glog.Infof("AdvLv7 sync worker condition, workernode name: %s, condition: %s, %s",
			workernode.GetQualifiedName(), &workernode.Status.HealthCondition, condition)
	}
	if needUpdate {
		a.helper.Sync(workernode, condition)
	}

	return condition
}

func (a *AdvancedLv7HealthChecker) query(workernode *carbonv1.WorkerNode, config *carbonv1.HealthCheckerConfig, rollingset *carbonv1.RollingSet) (bool, *carbonv1.HealthCondition) {
	start := time.Now()
	url := getQueryURL(workernode, config)
	queryData := getQueryData(workernode, rollingset)
	ctx, cancel := context.WithTimeout(context.Background(), common.GetBatcherTimeout(time.Second*time.Duration(healthCheckHTTPTimeout)))
	defer cancel()
	response, err := a.excuteHTTPPostQuery(ctx, url, queryData)
	if response != nil {
		defer response.Body.Close()
	}
	if glog.V(5) {
		glog.Infof("AdvLv7 excuteHTTPPostQuery workernode: %s, url: %s, response: %v, queryData: %v, delay: %d",
			workernode.GetQualifiedName(),
			url, response,
			utils.ObjJSON(queryData),
			time.Now().Sub(start).Nanoseconds()/1000000)
	}
	if err != nil {
		glog.Warningf("AdvLv7 excuteHTTPPostQuery %s, %s, err: %v", workernode.GetQualifiedName(), url, err)
	}
	// 获取最新worker
	worker, _ := a.helper.GetWorkerNode(workernode)
	if nil != worker {
		workernode = worker
	}
	needUpdate, healthCondition := transfer(response, err, workernode, config)
	healthCondition.Type = carbonv1.AdvancedLv7Health
	if needUpdate && nil == err && nil != response && response.StatusCode == 200 && nil != response.Body {
		body, err := ioutil.ReadAll(response.Body)
		if nil != err {
			glog.Warningf("AdvLv7 read error :%v", err)
			return false, nil
		}
		resultMetas, err := parseHealthMetas(body)
		if err != nil {
			glog.Warningf("AdvLv7 parseHealthMetas error workernode:%v, err:%v", workernode.Name, err)
			return false, nil
		}
		if glog.V(5) {
			glog.Infof("AdvLv7 %s resultMetas:%v", workernode.Name, resultMetas)
		}
		healthCondition.Metas = resultMetas
		healthCondition.WorkerStatus = checkWorkerReady(workernode.Spec.Signature, resultMetas[HealthcheckPayloadSignature])
		healthCondition.Checked = true
	}
	return needUpdate, healthCondition
}

func parseHealthMetas(body []byte) (map[string]string, error) {
	metas := make(map[string]string)
	if body == nil {
		return metas, nil
	}
	err := json.Unmarshal(body, &metas)
	if err != nil {
		return nil, err
	}
	return metas, nil
}

func checkWorkerReady(signature, signature2 string) carbonv1.WorkerType {
	if signature == signature2 {
		return carbonv1.WorkerTypeReady
	}
	return carbonv1.WorkerTypeNotReady
}

func getQueryData(workernode *carbonv1.WorkerNode, rollingset *carbonv1.RollingSet) map[string]string {
	plan := workernode.Spec.VersionPlan
	porcessVersion := workernode.Status.Version

	finalPlan := rollingset.Spec.VersionPlan

	//ServiceInfoMetas from cm2 servicemeta with cm2_topo_info
	serviceMetas := workernode.Status.ServiceInfoMetas
	signature := plan.Signature
	customInfo := carbonv1.GetCustomInfo(&plan)

	metas := make(map[string]string)

	metas[HealthcheckPayloadSignature] = signature
	metas[HealthcheckPayloadCustominfo] = customInfo
	metas[HealthcheckPayloadGlobalCustominfo] = utils.ObjJSON(finalPlan)
	metas[HealthcheckPayloadIdentifier] = carbonv1.GenUniqIdentifier(workernode)
	metas[HealthcheckPayloadProcessVersion] = porcessVersion
	metas[HealthcheckPayloadSchedulerInfo] = serviceMetas
	preload := "false"
	if finalPlan.Preload {
		preload = "true"
	}
	metas[HealthcheckPayloadPreload] = preload
	return metas
}

func (a *AdvancedLv7HealthChecker) excuteHTTPPostQuery(ctx context.Context, url string, data interface{}) (*http.Response, error) {
	headers := map[string]string{"Content-Type": utils.HTTPContentTypeWWW, "method": "POST"}
	response, err := a.doPost(ctx, url, data, headers)
	return response, err
}

func (a *AdvancedLv7HealthChecker) doPost(ctx context.Context, url string, req interface{}, headers map[string]string) (*http.Response, error) {
	response, err := a.helper.GetHTTPClient().SendJSONRequest(ctx, "POST", url, req, headers)
	return response, err
}

func genUniqIdentifier(workernode *carbonv1.WorkerNode) string {
	hippoClusterName := carbonv1.GetClusterName(workernode)
	identifier := utils.AppendString(hippoClusterName, ":", workernode.Namespace, ":", workernode.Name)
	return identifier
}

func needHealthCheckAdvlv7(worker *carbonv1.WorkerNode) (bool, *carbonv1.HealthCondition) {
	ok, condition := needHealthCheck(worker)
	if !ok {
		return ok, condition
	}
	originalCondition := worker.Status.HealthCondition
	if !features.C2MutableFeatureGate.Enabled(features.HealthCheckWhenUpdatePod) && carbonv1.IsWorkerGracefully(worker) && worker.Status.ServiceStatus != carbonv1.ServiceUnAvailable && // gracefully 未摘流
		(carbonv1.IsWorkerVersionMisMatch(worker) && worker.Status.Version != "") && worker.Status.HealthCondition.Version != worker.Spec.Version { // rolling 中
		return false, &originalCondition
	}
	if !features.C2MutableFeatureGate.Enabled(features.HealthCheckWhenUpdatePod) && carbonv1.IsWorkerVersionMisMatch(worker) && worker.Status.Version != "" {
		return false, &originalCondition
	}
	if !carbonv1.IfWorkerServiceMetasRecovered(worker) {
		glog.Infof("%s not recovered, %s", worker.Name, utils.ObjJSON(worker.Status.ServiceConditions))
		return false, &originalCondition
	}
	return ok, condition
}
