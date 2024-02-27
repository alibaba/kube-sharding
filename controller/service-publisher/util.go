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

package publisher

import (
	reflect "reflect"
	"strings"

	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// getBizMetas get biz metas
func getBizMetas(worker *carbonv1.WorkerNode) map[string]string {
	if nil != worker {
		template, _ := carbonv1.GetGeneralTemplate(worker.Spec.VersionPlan)
		bizMetas := map[string]string{}
		setMeta(bizMetas, carbonv1.BizMeatsKeyHIPPOCluster, getFromMultiObject(carbonv1.GetClusterName, worker, template))
		setMeta(bizMetas, carbonv1.BizMeatsKeyHIPPOAPP, getFromMultiObject(carbonv1.GetAppName, worker, template))
		setMeta(bizMetas, carbonv1.BizMeatsKeyHIPPORole, getFromMultiObject(carbonv1.GetRoleName, worker, worker, template))
		setMeta(bizMetas, carbonv1.BizMeatsKeyC2Group, getFromMultiObject(carbonv1.GetGroupName, worker, worker, template))
		setMeta(bizMetas, carbonv1.BizMeatsKeyC2Role, getFromMultiObject(carbonv1.GetC2RoleName, worker, worker, template))
		copyWorkerBizMetas(bizMetas, worker)
		if worker.Status.Warmup {
			setMeta(bizMetas, carbonv1.CM2MetaWeightKey, carbonv1.CM2MetaWeightWarmupValue)
		}
		if worker.Spec.Cm2TopoInfo != "" {
			setMeta(bizMetas, carbonv1.CM2MetaTopoInfo, worker.Spec.Cm2TopoInfo)
		}
		return bizMetas
	}
	return nil
}

func copyWorkerBizMetas(bizMetas map[string]string, worker *carbonv1.WorkerNode) {
	copyBizMetas(worker.Labels, bizMetas, carbonv1.BizKeyPrefix)
	template, _ := carbonv1.GetGeneralTemplate(worker.Spec.VersionPlan)
	copyBizMetas(template.Labels, bizMetas, carbonv1.BizKeyPrefix)
	copyBizMetas(template.Annotations, bizMetas, carbonv1.BizCm2MetasKeyPrefix)
}

func copyBizMetas(from, to map[string]string, prefix string) {
	if from == nil || to == nil {
		return
	}

	for k, v := range from {
		if strings.HasPrefix(k, prefix) {
			to[strings.TrimPrefix(k, prefix)] = v
		}
	}
}

func getFromMultiObject(getObjMeta func(object metav1.Object) string, objs ...metav1.Object) string {
	for i := range objs {
		if nil == objs[i] || reflect.ValueOf(objs[i]).IsNil() {
			continue
		}
		if "" != getObjMeta(objs[i]) {
			return getObjMeta(objs[i])
		}
	}
	return ""
}

func setMeta(meta map[string]string, k, v string) {
	if "" == v {
		return
	}
	meta[k] = v
}
