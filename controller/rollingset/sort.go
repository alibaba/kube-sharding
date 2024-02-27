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
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
)

// ByScore 按照score升序排序, 根据score从小到大
type ByScore []*carbonv1.Replica

func (a ByScore) Len() int { return len(a) }
func (a ByScore) Less(i, j int) bool {
	if a[i].Status.Score == a[j].Status.Score {
		iCost := carbonv1.GetDeletionCost(a[i])
		jCost := carbonv1.GetDeletionCost(a[j])
		if iCost != jCost {
			return iCost < jCost
		}
		if a[i].Name > a[j].Name {
			return true
		}
		return false
	}
	return a[i].Status.Score < a[j].Status.Score
}
func (a ByScore) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

// ByUnAssignedSlot  按照AssignedSlot排序, AssignedSolt排前面
type ByUnAssignedSlot []*carbonv1.Replica

func (a ByUnAssignedSlot) Len() int { return len(a) }
func (a ByUnAssignedSlot) Less(i, j int) bool {
	return bTint(carbonv1.IsReplicaUnAssigned(a[i])) < bTint(carbonv1.IsReplicaUnAssigned(a[j]))
}
func (a ByUnAssignedSlot) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

func bTint(b bool) uint8 {
	if b {
		return 1
	}
	return 0
}

func bTint32(b bool) int32 {
	if b {
		return 1
	}
	return 0
}

// RedundantSort  Assigned/OnActive排前面
type RedundantSort []*carbonv1.Replica

func (a RedundantSort) Len() int { return len(a) }
func (a RedundantSort) Less(i, j int) bool {
	scoreI := carbonv1.AggregateFraction(
		bTint32(carbonv1.InStandby2Active(&a[i].WorkerNode)),
		bTint32(carbonv1.IsReplicaEmpty(a[i])),
		carbonv1.ReplicaWorkModeScore(a[i]),
		9-carbonv1.ReplicaResourcePoolScore(a[i]),
		bTint32(carbonv1.IsReplicaOnStandByResourcePool(a[i])),
	)
	scoreJ := carbonv1.AggregateFraction(
		bTint32(carbonv1.InStandby2Active(&a[j].WorkerNode)),
		bTint32(carbonv1.IsReplicaEmpty(a[j])),
		carbonv1.ReplicaWorkModeScore(a[j]),
		9-carbonv1.ReplicaResourcePoolScore(a[j]),
		bTint32(carbonv1.IsReplicaOnStandByResourcePool(a[j])),
	)
	iCost := carbonv1.GetDeletionCost(a[i])
	jCost := carbonv1.GetDeletionCost(a[j])
	if scoreI == scoreJ && iCost != jCost {
		return iCost > jCost
	}
	if scoreI == scoreJ {
		if a[i].Status.Score != a[j].Status.Score {
			return a[i].Status.Score > a[j].Status.Score
		}
		if a[i].Spec.IsSpot != a[j].Spec.IsSpot {
			return a[i].Spec.IsSpot
		}
		return a[i].Name > a[j].Name
	}
	return scoreI < scoreJ
}
func (a RedundantSort) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

// InactiveStandbyRedundantSort  Assigned/OnActive排前面
type InactiveStandbyRedundantSort []*carbonv1.Replica

func (a InactiveStandbyRedundantSort) Len() int { return len(a) }
func (a InactiveStandbyRedundantSort) Less(i, j int) bool {
	scoreI := carbonv1.AggregateFraction(
		bTint32(carbonv1.IsReplicaUnAssigned(a[i]) || carbonv1.IsReplicaNotInited(a[i])),
		9-carbonv1.ReplicaWorkModeScore(a[i]),
		9-carbonv1.ReplicaResourcePoolScore(a[i]),
		bTint32(carbonv1.IsReplicaOnStandByResourcePool(a[i])),
	)
	scoreJ := carbonv1.AggregateFraction(
		bTint32(carbonv1.IsReplicaUnAssigned(a[j]) || carbonv1.IsReplicaNotInited(a[i])),
		9-carbonv1.ReplicaWorkModeScore(a[j]),
		9-carbonv1.ReplicaResourcePoolScore(a[j]),
		bTint32(carbonv1.IsReplicaOnStandByResourcePool(a[j])),
	)
	iCost := carbonv1.GetDeletionCost(a[i])
	jCost := carbonv1.GetDeletionCost(a[i])
	if scoreI == scoreJ && iCost != jCost {
		return iCost > jCost
	}
	if scoreI == scoreJ {
		if a[i].Status.Score != a[j].Status.Score {
			return a[i].Status.Score > a[j].Status.Score
		}
		if a[i].Spec.IsSpot != a[j].Spec.IsSpot {
			return a[i].Spec.IsSpot
		}
		return a[i].Name > a[j].Name
	}
	return scoreI < scoreJ
}
func (a InactiveStandbyRedundantSort) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

// ByAvailableAndName  按照可用度排序, UnAvaiable,UnAssignedSolt排前面
type ByAvailableAndName []*carbonv1.Replica

func (a ByAvailableAndName) Len() int { return len(a) }
func (a ByAvailableAndName) Less(i, j int) bool {
	if carbonv1.IsReplicaComplete(a[i]) == carbonv1.IsReplicaComplete(a[j]) {
		if carbonv1.IsReplicaUnAssigned(a[i]) == carbonv1.IsReplicaUnAssigned(a[j]) {
			if a[i].Name > a[j].Name {
				return true
			}
			return false
		}
		return bTint(carbonv1.IsReplicaUnAssigned(a[i])) > bTint(carbonv1.IsReplicaUnAssigned(a[j]))
	}
	return bTint(carbonv1.IsReplicaComplete(a[i])) < bTint(carbonv1.IsReplicaComplete(a[j]))
}
func (a ByAvailableAndName) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

type ByDeletionCost []*carbonv1.Replica

func (a ByDeletionCost) Len() int { return len(a) }
func (a ByDeletionCost) Less(i, j int) bool {
	return carbonv1.GetDeletionCost(a[i]) > carbonv1.GetDeletionCost(a[j])
}
func (a ByDeletionCost) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

type Int64Slice []int64

func (x Int64Slice) Len() int           { return len(x) }
func (x Int64Slice) Less(i, j int) bool { return x[i] < x[j] }
func (x Int64Slice) Swap(i, j int)      { x[i], x[j] = x[j], x[i] }
