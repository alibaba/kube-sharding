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

package v1

import (
	"encoding/json"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	glog "k8s.io/klog"
)

// VersionPlanWithGangPlan contains replica versionPlan and gang versionPlan
type VersionPlanWithGangPlan struct {
	*VersionPlan
	GangVersionPlans map[string]*VersionPlan
}

// GetVersionPlanWithGangPlanFromRS returns rollingset VersionPlanWithGangPlan
func GetVersionPlanWithGangPlanFromRS(rollingSet *RollingSet) *VersionPlanWithGangPlan {
	return &VersionPlanWithGangPlan{
		VersionPlan:      &rollingSet.Spec.VersionPlan,
		GangVersionPlans: rollingSet.Spec.GangVersionPlan,
	}
}

// GetVersionPlanWithGangPlanFromReplica returns replica VersionPlanWithGangPlan
func GetVersionPlanWithGangPlanFromReplica(replica *Replica) *VersionPlanWithGangPlan {
	if !replica.IsGangReplica() {
		return &VersionPlanWithGangPlan{
			VersionPlan: &replica.Spec.VersionPlan,
		}
	}
	var gangVersionPlans = map[string]*VersionPlan{}
	for k := range replica.Gang {
		worker := replica.Gang[k].WorkerNode
		plan := GetGangPlan(&worker)
		if plan == "" {
			continue
		}
		err := json.Unmarshal([]byte(plan), &gangVersionPlans)
		if err != nil {
			glog.Errorf("unmarshal with error %v", err)
			return nil
		}
		return &VersionPlanWithGangPlan{
			GangVersionPlans: gangVersionPlans,
		}
	}
	return nil
}

// SetReplicaVersionPlanWithGangPlan set replica VersionPlanWithGangPlan
func SetReplicaVersionPlanWithGangPlan(replica *Replica, versionPlans *VersionPlanWithGangPlan, version string) error {
	if !replica.IsGangReplica() {
		replica.Spec.Version = version
		replica.Spec.VersionPlan = *versionPlans.VersionPlan
		return nil
	}
	gangPlan := utils.ObjJSON(versionPlans.GangVersionPlans)
	for i := range replica.Gang {
		node := &replica.Gang[i]
		partName := GetGangPartName(node)
		if versionPlan, ok := versionPlans.GangVersionPlans[partName]; ok {
			if IsGangMainPart(versionPlan.Template) {
				SetGangPlan(node, gangPlan)
			}
			node.Spec.VersionPlan = *versionPlan
			node.Spec.Version = version
		} else {
			node.Spec.ToDelete = true
			glog.Infof("remove node %s,%s, because of delete part", replica.Name, partName)
		}
		syncGangNodeMetas(&node.WorkerNode)
	}
	return nil
}

func syncGangNodeMetas(node *WorkerNode) error {
	node.Labels = PatchMap(node.Labels, node.Spec.Template.Labels, LabelKeyRoleName, LabelKeyRoleNameHash)
	node.Annotations = PatchMap(node.Annotations, node.Spec.Template.Annotations, LabelKeyRoleName, LabelKeyRoleNameHash)
	SetRoleName(node, GetRoleName(node.Spec.Template))
	SetObjectRoleHash(node, GetAppName(node.Spec.Template), GetRoleName(node.Spec.Template))
	return nil
}
