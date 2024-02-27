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

	app "k8s.io/api/apps/v1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ShardGroup is a specification for shardgroup resource
type ShardGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ShardGroupSpec   `json:"spec"`
	Status            ShardGroupStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ShardGroupList is a list of ShardGroup resources
type ShardGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []ShardGroup `json:"items"`
}

// ShardDeployType is the type for shardDeploy
type ShardDeployType string

const (
	// RollingSetType for rollingSet rolling
	RollingSetType ShardDeployType = "rollingSet"
	// DeploymentType for deployment rolling
	DeploymentType ShardDeployType = "deployment"
)

// ShardGroupSpec is the spec for a ShardGroup resource
type ShardGroupSpec struct {
	Selector       *metav1.LabelSelector `json:"selector,omitempty"`
	RollingVersion string                `json:"rollingVersion,omitempty"`
	Paused         bool                  `json:"paused,omitempty"`
	//Strict         bool                  `json:"strict,omitempty"`

	LatestPercent  *int32              `json:"latestPercent,omitempty"` //对齐目标比例，取值范围1-100
	MaxUnavailable *intstr.IntOrString `json:"maxUnavailable,omitempty"`
	MaxSurge       *intstr.IntOrString `json:"maxSurge,omitempty"`

	// scaleConfig
	ScaleStrategy   *app.DeploymentStrategy `json:"scaleStrategy,omitempty"`
	ScaleConfig     *ScaleConfig            `json:"scaleConfig,omitempty"`
	RollingStrategy string                  `json:"rollingStrategy,omitempty"`

	AppStopCmd      string `json:"appStopCmd,omitempty"`
	AppStopTimeout  int32  `json:"appStopTimeout,omitempty"`
	AppReadyTimeout int64  `json:"appReadyTimeout,omitempty"`
	AutoRecovery    bool   `json:"autoRecovery,omitempty"`

	// from where to allocate entity. for example, "replica","rollingset-name" or  "pod",""
	WorkerSchedulePlan `json:",inline"`

	// UpdatingGracefully means is need to unpublish before update
	UpdatingGracefully         *bool `json:"updatingGracefully,omitempty"`
	RestartAfterResourceChange *bool `json:"restartAfterResourceChange,omitempty"`

	ShardTemplates           map[string]ShardTemplate   `json:"shardTemplates,omitempty"`
	CompressedShardTemplates string                     `json:"compressedShardTemplates,omitempty"`
	ShardDeployStrategy      ShardDeployType            `json:"strategy,omitempty"`
	Extension                map[string]json.RawMessage `json:"extension,omitempty"`

	UpdatePlanTimestamp int64 `json:"updatePlanTimestamp,omitempty"`
}

// GetStrategy GetStrategy
func (s *ShardGroupSpec) GetStrategy() appsv1.DeploymentStrategy {
	var strategy = appsv1.DeploymentStrategy{
		RollingUpdate: &appsv1.RollingUpdateDeployment{
			MaxUnavailable: s.MaxUnavailable,
			MaxSurge:       s.MaxSurge,
		},
	}
	if nil != s.ScaleStrategy && (nil == s.ScaleConfig || s.ScaleConfig.Enable) {
		return *s.ScaleStrategy
	}
	return strategy
}

// ShardTemplate is the template of Shard
type ShardTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ShardSpec `json:"shardSpec,omitempty"`
}

// ShardSpec is the spec for shard resource
type ShardSpec struct {
	//Selector            *metav1.LabelSelector `json:"selector,omitempty"`
	Replicas             *int32                  `json:"replicas"`
	DependencyLevel      *int32                  `json:"dependencyLevel"`
	Strategy             *app.DeploymentStrategy `json:"strategy,omitempty"`
	CustomInfo           string                  `json:"customInfo,omitempty"`
	CompressedCustomInfo string                  `json:"compressedCustomInfo,omitempty"`
	Signature            string                  `json:"signature,omitempty"`
	UserDefVersion       string                  `json:"userDefVersion,omitempty"`
	Preload              bool                    `json:"preload,omitempty"`
	// Template describes basic resource template.
	Template            HippoPodTemplate     `json:"template"`
	HealthCheckerConfig *HealthCheckerConfig `json:"healthCheckerConfig"`
	Online              *bool                `json:"online,omitempty"`
	// UpdatingGracefully means is need to unpublish before update
	UpdatingGracefully *bool  `json:"updatingGracefully,omitempty"`
	Cm2TopoInfo        string `json:"cm2TopoInfo,omitempty"`
}

// GetReplicas GetReplicas
func (s *ShardSpec) GetReplicas() int32 {
	if nil == s.Replicas {
		return 0
	}
	return *s.Replicas
}

// ShardSignSpec is the spec for shard resource
type ShardSignSpec struct {
	Signature   string           `json:"signature,omitempty"`
	Cm2TopoInfo string           `json:"cm2TopoInfo,omitempty"`
	Template    HippoPodTemplate `json:"template"`
}

// ShardSignTemplate is used for the sign of the shardGroup
type ShardSignTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ShardSignSpec `json:"shardSpec,omitempty"`
}

// ShardGroupSignSpec is the spec for sign  shardGroup
type ShardGroupSignSpec struct {
	RestartAfterResourceChange *bool                        `json:"restartAfterResourceChange,omitempty"`
	ShardTemplates             map[string]ShardSignTemplate `json:"shardTemplates,omitempty"`
}

// ShardGroupPhase is the phase of the shardGroup resouce
type ShardGroupPhase string

const (
	// ShardGroupUpdating means the shardgroup is updating
	ShardGroupUpdating ShardGroupPhase = "Updating"
	// ShardGroupRolling means the shardgroup is updating
	ShardGroupRolling ShardGroupPhase = "Rolling"
	// ShardGroupRunning means that the rolling is done
	ShardGroupRunning ShardGroupPhase = "Running"
	// ShardGroupFailed means that the rolling update is failed
	ShardGroupFailed ShardGroupPhase = "Failed"
	// ShardGroupUnknown means that for some reason the state of the shardGroup could not be obtained.
	ShardGroupUnknown ShardGroupPhase = "Unknown"
)

//ShardGroupStatus is the status for a ShardGroup resour
type ShardGroupStatus struct {
	// ObservedGeneration reflects the generation of the most recently observed shardGroup
	// +optional
	ObservedGeneration int64 `json:"observedGeneration"`
	// ShardGroupVersion is the version for shardGroup
	ShardGroupVersion string `json:"shardGroupVersion"`
	// phase of shardGroup status
	Phase ShardGroupPhase `json:"phase"` // Updating Running Failed
	// return all the rollingset or deployment name owned by shardGroup
	ShardNames []string `json:"shardNames,omitempty"`
	// 完成过目标的列，区别于新建列
	OnceCompletedShardNames []string `json:"onceCompletedShardNames,omitempty"`
	Complete                bool     `json:"complete,omitempty"`
	// update time by shardGroup resource
	LastUpdateStatusTime metav1.Time `json:"lastUpdateStatusTime,omitempty"`
}
