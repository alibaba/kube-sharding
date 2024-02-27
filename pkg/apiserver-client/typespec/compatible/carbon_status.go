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

package compatible

import (
	"github.com/alibaba/kube-sharding/pkg/apiserver-client/typespec"
	"github.com/alibaba/kube-sharding/pkg/apiserver-client/typespec/carbon"
	"github.com/alibaba/kube-sharding/pkg/apiserver-client/typespec/hippo"
)

// ReplicaNodeStatus define replica status
type ReplicaNodeStatus struct {
	ReplicaNodeID          *string           `json:"replicaNodeId,omitempty"`
	CurWorkerNodeStatus    *WorkerNodeStatus `json:"curWorkerNodeStatus,omitempty"`
	BackupWorkerNodeStatus *WorkerNodeStatus `json:"backupWorkerNodeStatus,omitempty"`
	TimeStamp              *int64            `json:"timeStamp,omitempty"`
	UserDefVersion         *string           `json:"userDefVersion,omitempty"`
	ReadyForCurVersion     *bool             `json:"readyForCurVersion,omitempty"`
}

// WorkerNodeStatus define worker status
type WorkerNodeStatus struct {
	WorkerNodeID       string          `json:"workerNodeId"`
	CurVersion         string          `json:"curVersion"`
	NextVersion        string          `json:"nextVersion"`
	FinalVersion       string          `json:"finalVersion"`
	Offline            bool            `json:"offline"`
	Releasing          bool            `json:"releasing"`
	Reclaiming         bool            `json:"reclaiming"`
	ReadyForCurVersion bool            `json:"readyForCurVersion"`
	LastNotMatchTime   int64           `json:"lastNotMatchTime"`
	LastNotReadyTime   *int64          `json:"lastNotReadyTime,omitempty"`
	SlotAllocStatus    int             `json:"slotAllocStatus"`
	SlotInfo           *SlotInfo       `json:"slotInfo,omitempty"`
	HealthInfo         *HealthInfo     `json:"healthInfo,omitempty"`
	ServiceInfo        *ServiceInfo    `json:"serviceInfo,omitempty"`
	SlotStatus         *RoleSlotStatus `json:"slotStatus,omitempty"`

	IP               string `json:"ip"`
	UserDefVersion   string `json:"userDefVersion"`
	TargetSignature  string `json:"targetSignature"`
	TargetCustomInfo string `json:"targetCustomInfo"`
}

// HealthInfo define health status
type HealthInfo struct {
	HealthStatus int               `json:"healthStatus"`
	WorkerStatus int               `json:"workerStatus"`
	Metas        map[string]string `json:"metas"`
	Version      string            `json:"version"`
}

// ServiceInfo define service status
type ServiceInfo struct {
	Status int               `json:"status"`
	Metas  map[string]string `json:"metas"`
}

// RoleSlotStatus define slot status
type RoleSlotStatus struct {
	Status int          `json:"status,omitempty"`
	SlotID hippo.SlotId `json:"slotId,omitempty"`
}

// RoleStatus define role status
type RoleStatus struct {
	RoleID             string                             `json:"roleId,omitempty"`
	GlobalPlan         *typespec.GlobalPlan               `json:"globalPlan,omitempty"`
	VersionedPlans     *map[string]typespec.VersionedPlan `json:"versionedPlans,omitempty"`
	LatestVersion      string                             `json:"latestVersion,omitempty"`
	Nodes              []*ReplicaNodeStatus               `json:"nodes,omitempty"`
	UserDefVersion     *string                            `json:"userDefVersion,omitempty"`
	ReadyForCurVersion *bool                              `json:"readyForCurVersion,omitempty"`
	MinHealthCapacity  *int32                             `json:"minHealthCapacity,omitempty"`
}

// GroupStatus define group status
type GroupStatus struct {
	GroupID string                `json:"groupId"`
	Roles   map[string]RoleStatus `json:"roles"`
}

// SlotInfo define slot info
type SlotInfo struct {
	Role                             *string                     `json:"role,omitempty"`
	SlotID                           hippo.SlotId                `json:"slotId,omitempty"`
	Reclaiming                       bool                        `json:"reclaiming"`
	SlotResource                     *carbon.SlotResource        `json:"slotResource,omitempty"`
	SlaveStatus                      *hippo.SlaveStatus_Status   `json:"slaveStatus,omitempty"`
	ProcessStatus                    *[]*hippo.ProcessStatus     `json:"processStatus,omitempty"`
	PackageStatus                    *hippo.PackageStatus_Status `json:"packageStatus,omitempty"`
	PreDeployPackageStatus           *hippo.PackageStatus_Status `json:"preDeployPackageStatus,omitempty"`
	PackageChecksum                  *string                     `json:"packageChecksum,omitempty"`
	PreDeployPackageChecksum         *string                     `json:"preDeployPackageChecksum,omitempty"`
	LaunchSignature                  *int64                      `json:"launchSignature,omitempty"`
	NoLongerMatchQueue               *bool                       `json:"noLongerMatchQueue,omitempty"`
	NoLongerMatchResourceRequirement *bool                       `json:"noLongerMatchResourceRequirement,omitempty"`
	RequirementID                    string                      `json:"requirementId,omitempty"`
	Priority                         *hippo.Priority             `json:"priority,omitempty"`
}
