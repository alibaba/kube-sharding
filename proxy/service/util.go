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

package service

import (
	"encoding/json"
	"reflect"

	"github.com/openkruise/kruise-api/apps/v1alpha1"

	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	"k8s.io/kubernetes/pkg/util/slice"
)

const inplaceSetProxy = "inplaceset.sigma.ali/proxy"

// ToJSONString to json string
func ToJSONString(obj interface{}) string {
	jsonString, err := json.Marshal(obj)
	if err != nil {
		return "ILLEGAL_JSON_OBJ"
	}
	return string(jsonString)
}

// SliceContainsSlice slice contains slice or not
func SliceContainsSlice(parent, child []string) bool {
	if len(parent) == 0 || len(child) == 0 {
		return reflect.DeepEqual(parent, child)
	}
	for _, c := range child {
		if !slice.ContainsString(parent, c, nil) {
			return false
		}
	}
	return true
}

func fixRsReplicasFromSpread(rs *carbonv1.RollingSet) {
	active, inactive := carbonv1.GetInactiveSpreadReplicas(rs)
	if active == -1 && inactive == -1 {
		if rs.Spec.ScaleSchedulePlan != nil && rs.Spec.ScaleSchedulePlan.RecycleMode == carbonv1.WorkerModeTypeInactive {
			rs.Spec.ScaleSchedulePlan = nil
			rs.Spec.ScaleConfig = nil
		}
		return
	}
	total := active + inactive
	rs.Spec.Replicas = &total
	rs.Spec.ScaleSchedulePlan = &carbonv1.ScaleSchedulePlan{
		Replicas:    &active,
		RecycleMode: carbonv1.WorkerModeTypeInactive,
	}
	rs.Spec.ScaleConfig = &carbonv1.ScaleConfig{
		Enable: true,
	}
}

func getPodKey(namespace, name string) string {
	return namespace + "/" + name
}

func getProxyStatefulSet(cloneSet *v1alpha1.CloneSet) string {
	if cloneSet == nil || cloneSet.Labels == nil || cloneSet.Labels[inplaceSetProxy] != "CloneSet" {
		return ""
	}
	for i := range cloneSet.OwnerReferences {
		if r := cloneSet.OwnerReferences[i]; r.Kind == "StatefulSet" && r.APIVersion == "apps/v1" {
			return r.Name
		}
	}
	return ""
}
