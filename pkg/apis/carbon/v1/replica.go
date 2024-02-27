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
	glog "k8s.io/klog"
)

// Replica is a specification for a Replica resource
type Replica struct {
	WorkerNode
	Backup WorkerNode
	Gang   []Replica
}

// GetVersion returns Replica spec version
// The implementation of the Node interface
func (o *Replica) GetVersion() string {
	return o.Spec.Version
}

// GetVersionPlan returns Replica spec version plan
// The implementation of the Node interface
func (o *Replica) GetVersionPlan() interface{} {
	return GetVersionPlanWithGangPlanFromReplica(o)
}

// GetScore returns Replica status score
// The implementation of the Node interface
func (o *Replica) GetScore() int {
	return int(ScoreOfWorker(&o.WorkerNode))
}

// GetUpdateScore returns Replica score when upgrading
func (o *Replica) GetUpdateScore() int {
	return int(AggregateFraction(
		bTint32(InStandby2Active(&o.WorkerNode)),
		bTint32(IsReplicaEmpty(o)),
		ReplicaWorkModeScore(o),
		9-ReplicaResourcePoolScore(o),
		bTint32(IsReplicaOnStandByResourcePool(o)),
		bTint32(!o.Spec.IsSpot),
	))
}

// GetDeletionCost returns Replica status score
// The implementation of the Node interface
func (o *Replica) GetDeletionCost() int {
	return int(GetDeletionCost(o))
}

// IsReady returns whether Replica is ready
// The implementation of the Node interface
func (o *Replica) IsReady() bool {
	_, ok := IsReplicaServiceReady(o)
	return ok
}

// IsEmpty returns whether Replica is empty
// The implementation of the Node interface
func (o *Replica) IsEmpty() bool {
	return IsReplicaEmpty(o)
}

// IsRelease returns whether Replica is release
// The implementation of the Node interface
func (o *Replica) IsRelease() bool {
	return IsReplicaReleasing(o) || IsReplicaReleased(o)
}

// SetVersion set Replica spec version and spec version plan
// The implementation of the Node interface
func (o *Replica) SetVersion(version string, versionPlan interface{}) {
	if SetStandby2Active(&o.WorkerNode) {
		glog.V(3).Infof("worker[%v] change standby[%v] to active", o.Name, o.Spec.WorkerMode)
	}
	if !o.IsGangReplica() {
		if !SyncSilence(&o.WorkerNode, WorkerModeTypeActive) {
			if o.Spec.WorkerMode != WorkerModeTypeActive {
				glog.Infof("set workermode %s, active", o.Name)
			}
			o.Spec.WorkerMode = WorkerModeTypeActive
		}
		if o.Spec.WorkerMode == WorkerModeTypeCold && o.Status.IsFedWorker {
			glog.V(4).Infof("worker[%s] is fed pod, can't change to cold now", o.Name)
			o.Spec.WorkerMode = WorkerModeTypeWarm
		}
	}
	plan, _ := versionPlan.(*VersionPlanWithGangPlan)
	SetReplicaVersionPlanWithGangPlan(o, plan, version)
}

// Stop function stop a Replica
// The implementation of the Node interface
func (o *Replica) Stop() {}

// GetDependencyReady returns Replica status dependencyReady
// The implementation of the GroupNode interface
func (o *Replica) GetDependencyReady() bool {
	return IsReplicaDependencyReady(o)
}

// SetDependencyReady set Replica status dependencyReady
// The implementation of the GroupNode interface
func (o *Replica) SetDependencyReady(dependencyReady bool) {
	o.Spec.DependencyReady = &dependencyReady
}

// GetGroupVersion returns Replica spec group version
// The implementation of the GroupNode interface
func (o *Replica) GetGroupVersion() string {
	if o.Spec.ShardGroupVersion == "" {
		return o.GetVersion()
	}
	return o.Spec.ShardGroupVersion
}

// IsGangReplica returns whether Replica is a Gang
func (o *Replica) IsGangReplica() bool {
	return o.Gang != nil
}

func bTint32(b bool) int32 {
	if b {
		return 1
	}
	return 0
}
