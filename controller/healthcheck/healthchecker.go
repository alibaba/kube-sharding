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
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
)

//HealthChecker 健康检查接口
type HealthChecker interface {
	//	doCheck(workernode *carbonv1.WorkerNode, healthCheckerConfig *carbonv1.HealthCheckerConfig) *carbonv1.HealthCondition

	//	doCheckBatch(workernodes []*carbonv1.WorkerNode, healthCheckerConfig *carbonv1.HealthCheckerConfig) []*carbonv1.HealthCondition

	//心跳检查，workernode状态变更
	doCheck(workernode *carbonv1.WorkerNode, healthCheckerConfig *carbonv1.HealthCheckerConfig, rollingset *carbonv1.RollingSet) *carbonv1.HealthCondition

	//获取HealthChecker的typename
	getTypeName() string
}
