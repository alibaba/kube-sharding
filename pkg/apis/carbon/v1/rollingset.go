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
	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	"github.com/alibaba/kube-sharding/pkg/rollalgorithm"
	app "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	glog "k8s.io/klog"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RollingSet is a specification for a RollingSet resource
type RollingSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RollingSetSpec   `json:"spec"`
	Status RollingSetStatus `json:"status"`
}

// ScaleSchedulePlan ScaleSchedulePlan
type ScaleSchedulePlan struct {
	//Replicas 目标总数， 不能为nil
	Replicas *int32 `json:"replicas"`

	// The rollingset strategy to use to replace existing replica with new ones.
	// +optional
	// +patchStrategy=retainKeys
	Strategy app.DeploymentStrategy `json:"strategy,omitempty"`

	RecycleMode WorkerModeType `json:"recycleMode,omitempty"`

	RecyclePool string `json:"recyclePool,omitempty"`

	SmoothRolling bool `json:"smoothRolling,omitempty"`

	// vpa scale
	ResourceRequests corev1.ResourceList `json:"ResourceRequests,omitempty"`
}

// BufferPlan BufferPlan
type BufferPlan struct {
	Distribute *BufferDistribute `json:"distribute"`
}

// ActivePlan ActivePlan
type ActivePlan struct {
	ScaleSchedulePlan *ScaleSchedulePlan `json:"scaleSchedulePlan,omitempty"`
	ScaleConfig       *ScaleConfig       `json:"scaleConfig,omitempty"`
}

// Capacity ...
type Capacity struct {
	// 总资源容量（Active+Buffer)
	Total *int32 `json:"total"`
	// Buffer资源相关
	BufferPlan BufferPlan `json:"bufferPlan"`
	// Active资源（弹性）相关
	ActivePlan     ActivePlan `json:"activePlan"`
	ActualReplicas int        `json:"actualReplicas"`
}

// BufferDistribute Buffer配置
type BufferDistribute struct {
	Cold int32 `json:"cold"`
	Warm int32 `json:"warm"`
	Hot  int32 `json:"hot"`
}

func (b *BufferDistribute) isZero() bool {
	return b.Cold == 0 && b.Warm == 0 && b.Hot == 0
}

func (p *ScaleSchedulePlan) getReplicas() int32 {
	if nil == p.Replicas {
		return 0
	}
	return *p.Replicas
}

// ScaleConfig ScaleConfig
type ScaleConfig struct {
	Enable bool `json:"enable"`
}

// RollingSetSpec is the spec for a RollingSetSpec resource
type RollingSetSpec struct {
	// Label selector for pods. Existing Replicas whose pods are
	// selected by this will be the ones affected by this rollingset.
	// It must match the pod template's labels.
	Selector *metav1.LabelSelector `json:"selector" protobuf:"bytes,2,opt,name=selector"`

	// config for schedule
	rollalgorithm.SchedulePlan `json:",inline"`
	// ScaleSchedulePlan ScaleConfig 迁移到 Capacity里去
	ScaleSchedulePlan *ScaleSchedulePlan  `json:"scaleSchedulePlan,omitempty"`
	ScaleConfig       *ScaleConfig        `json:"scaleConfig,omitempty"`
	Capacity          Capacity            `json:"capacity,omitempty"`
	DaemonSetReplicas *intstr.IntOrString `json:"daemonSetReplicas"`
	// latestVersionRatio of subRS
	SubRSLatestVersionRatio *map[string]int32 `json:"subrsLatestVersionRatio"`
	// the owner of subRS
	GroupRS *string `json:"groupRS"`

	// ResVersion means the signature of Resources.
	ResVersion string `json:"resVersion"`
	// Version means the signature of VersionPlan.
	Version string `json:"version"`
	// InstanceID to mark restarting
	InstanceID string `json:"instanceID"`
	// config for resource and progress
	VersionPlan `json:",inline"`

	// CheckSum means the signature of spec.
	CheckSum string `json:"checkSum"`

	//健康检查配置
	HealthCheckerConfig *HealthCheckerConfig `json:"healthCheckerConfig"`
	// 弹性副本数预测 key hour,val replicas
	PredictInfo       string                  `json:"predictInfo,omitempty"`
	GangVersionPlan   map[string]*VersionPlan `json:"gangVersionPlan,omitempty"`
	SubrsRatioVersion string                  `json:"subrsRatioVersion,omitempty"`
	SubrsBlackList    []string                `json:"subrsBlackList,omitempty"`
}

// GroupVersionStatusMap ...
type GroupVersionStatusMap map[string]*rollalgorithm.VersionStatus

// SubrsVersionStatusMap ...
type SubrsVersionStatusMap map[string]GroupVersionStatusMap

type SpotStatus struct {
	Target    int64    `json:"target,omitempty"`
	Instances []string `json:"instances,omitempty"`
}

// RollingSetStatus is the status for a RollingSet resource
type RollingSetStatus struct {
	// The generation observed by the rollingset controller.
	// +optional

	Complete     bool `json:"complete"`
	ServiceReady bool `json:"serviceReady"`

	Rolling  bool `json:"rolling,omitempty"`
	Reducing bool `json:"reducing,omitempty"`
	// Total number of non-terminated Replica targeted by this rollingset (their labels match the selector).
	// +optional
	Replicas int32 `json:"replicas"`
	Workers  int32 `json:"workers"`

	ActiveReplicas             int32            `json:"activeReplicas"`
	StandbyReplicas            int32            `json:"standbyReplicas"`
	ColdStandbys               int32            `json:"coldStandbys"`
	WarmStandbys               int32            `json:"warmStandbys"`
	HotStandbys                int32            `json:"hotStandbys"`
	WorkerModeMisMatchReplicas int32            `json:"workerModeMisMatchReplicas"`
	UnAssignedStandbyReplicas  int32            `json:"unAssignedStandbyReplicas"`
	UpdatedReplicas            int32            `json:"updatedReplicas"`
	ReadyReplicas              int32            `json:"readyReplicas"`
	AvailableReplicas          int32            `json:"availableReplicas"`
	UnavailableReplicas        int32            `json:"unavailableReplicas"`
	AllocatedReplicas          int32            `json:"allocatedReplicas,omitempty"`
	ReleasingReplicas          int32            `json:"releasingReplicas"`
	ReleasingWorkers           int32            `json:"releasingWorkers"`
	UnAssignedWorkers          map[string]int32 `json:"unAssignedWorkers"`

	// rollingset 下的version数，包含rollingset和replica
	// +optional
	VersionCount int32 `json:"versionCount"`

	// Represents the latest available observations of a rollingset's current state.
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []app.DeploymentCondition `json:"conditions,omitempty"`

	//LastUpdateStatusTime 最近一次更新时间
	LastUpdateStatusTime metav1.Time `json:"lastUpdateStatusTime,omitempty"`

	//GroupVersionStatusMap 每个版本的服务状态,key是ShardGroupVersion
	GroupVersionStatusMap GroupVersionStatusMap `json:"groupVersionStatusMap,omitempty"`

	//ResourceVersionStatusMap 每个版本的服务状态,key是ShardGroupVersion
	ResourceVersionStatusMap map[string]*rollalgorithm.ResourceVersionStatus `json:"resourceVersionStatusMap,omitempty"`
	//Version 当前正在处理的version
	Version        string `json:"version"`
	UserDefVersion string `json:"userDefVersion,omitempty"`
	// InstanceID to mark restarting
	InstanceID            string                 `json:"instanceID"`
	UncompleteReplicaInfo map[string]ProcessStep `json:"uncompleteReplicaInfo"`

	// subrs 中每个Rollingset 的versionStatusMap, key是subrsName, value 是这个subrs 的GroupVersionStatusMap
	SubrsVersionStatusMap SubrsVersionStatusMap `json:"subrsVersionStatusMap,omitempty"`
	SubrsTargetReplicas   map[string]*int32     `json:"subrsTargetReplicas,omitempty"`
	// LabelSelector is label selectors for query over pods that should match the replica count used by HPA.
	LabelSelector      string          `json:"labelSelector,omitempty"`
	SpotInstanceStatus SpotStatus      `json:"spotInstanceStatus"`
	SubrsRatioVersion  string          `json:"subrsRatioVersion"`
	SubrsTargetReached map[string]bool `json:"subrsTargetReached,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RollingSetList is a list of RollingSet resources
type RollingSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []RollingSet `json:"items"`
}

// BufferDistributeRatio buffer分池比例配置
type BufferDistributeRatio BufferDistribute

// GetRawReplicas GetRawReplicas
func (r *RollingSet) GetRawReplicas() int32 {
	if nil == r.Spec.SchedulePlan.Replicas {
		return 0
	}
	return *r.Spec.SchedulePlan.Replicas
}

// GetReplicas GetReplicas
func (r *RollingSet) GetReplicas() int32 {
	if nil == r.Spec.SchedulePlan.Replicas {
		return 0
	}
	scaleParameters := r.GetScaleParameters()
	if nil == scaleParameters || !r.scaleEnable() {
		return *r.Spec.SchedulePlan.Replicas
	}
	return scaleParameters.ActiveReplicas
}

// GetSchedulePlan GetSchedulePlan
func (r *RollingSet) GetSchedulePlan() rollalgorithm.SchedulePlan {
	if nil == r.Spec.SchedulePlan.Replicas {
		return r.Spec.SchedulePlan
	}
	scaleParameters := r.GetScaleParameters()
	if nil == scaleParameters || !r.scaleEnable() {
		return r.Spec.SchedulePlan
	}
	schedulePlan := r.Spec.SchedulePlan.DeepCopy()
	schedulePlan.Replicas = &scaleParameters.ActiveReplicas
	if nil != r.Spec.ScaleSchedulePlan.Strategy.RollingUpdate {
		schedulePlan.Strategy = *r.Spec.ScaleSchedulePlan.Strategy.DeepCopy()
		fixed := rollalgorithm.FixStrategy(*schedulePlan.Replicas, r.Spec.Strategy.RollingUpdate.MaxUnavailable, &schedulePlan.Strategy)
		if fixed {
			unavailable, _ := schedulePlan.Strategy.RollingUpdate.MaxUnavailable.MarshalJSON()
			glog.Infof("fix rollingsets strategy %s , %s", r.Name, unavailable)
		}
	}
	return *schedulePlan
}

// GetScaleStrategy GetScaleStrategy
func (r *RollingSet) GetScaleStrategy() *app.DeploymentStrategy {
	if !r.scaleEnable() {
		return nil
	}
	if nil == r.Spec.SchedulePlan.Replicas {
		return nil
	}
	if nil == r.Spec.SchedulePlan.Strategy.RollingUpdate {
		return nil
	}
	if r.Spec.ScaleSchedulePlan == nil || r.Spec.ScaleSchedulePlan.Replicas == nil || r.Spec.ScaleSchedulePlan.Strategy.RollingUpdate == nil {
		return nil
	}
	return &r.Spec.ScaleSchedulePlan.Strategy
}

// GetDependencyLevel GetDependencyLevel
func (r *RollingSet) GetDependencyLevel() int32 {
	if nil == r.Spec.SchedulePlan.DependencyLevel {
		return 0
	}
	return *r.Spec.SchedulePlan.DependencyLevel
}

// GetName GetName
func (r *RollingSet) GetName() string {
	if nil == r {
		return ""
	}
	return r.Name
}

// GetCarbonRoleName GetCarbonRoleName
func (r *RollingSet) GetCarbonRoleName() string {
	if nil == r || nil == r.Labels {
		return ""
	}
	return GetRoleName(r)
}

// GetVersionStatus GetVersionStatus
func (r *RollingSet) GetVersionStatus() map[string]*rollalgorithm.VersionStatus {
	return r.Status.GroupVersionStatusMap
}

// GetResourceVersionStatus GetResourceVersionStatus
func (r *RollingSet) GetResourceVersionStatus() map[string]*rollalgorithm.ResourceVersionStatus {
	return r.Status.ResourceVersionStatusMap
}

// GetCPU GetCPU
func (r *RollingSet) GetCPU() int32 {
	if nil != r.Spec.CPU {
		return *r.Spec.CPU
	}
	podSpec, _, err := GetPodSpecFromTemplate(r.Spec.Template)
	if nil != err {
		return 0
	}
	var cpu int32
	for _, container := range podSpec.Containers {
		for name, resource := range container.Resources.Limits {
			if "cpu" == name {
				cpuInt64, _ := resource.AsInt64()
				cpu += int32(cpuInt64)
			}
		}
	}
	r.Spec.CPU = utils.Int32Ptr(cpu)
	return cpu
}

// CreatePod CreatePod
func (r *RollingSet) CreatePod() (*corev1.Pod, error) {
	var pod corev1.Pod
	err := utils.JSONDeepCopy(r.Spec.Template, &pod)
	return &pod, err
}

// GetTotal ...
func (r *RollingSet) GetTotal() int32 {
	if r.Spec.Capacity.Total == nil {
		return r.GetRawReplicas()
	}
	return *r.Spec.Capacity.Total
}

// GetLatestVersionRatio GetLatestVersionRatio
func (r *RollingSet) GetLatestVersionRatio(version string) int32 {
	if (r.Spec.ShardGroupVersion == version || r.Spec.Version == version) && r.Spec.LatestVersionRatio != nil {
		return *r.Spec.LatestVersionRatio
	}
	return 0
}

// ScaleParameters ScaleParameters
type ScaleParameters struct {
	ActiveReplicas         int32
	StandByReplicas        int32
	StandByRatioMap        map[WorkerModeType]int
	ScalerResourcePoolName string
}

// GetScaleParameters GetScaleParameters
func (r *RollingSet) GetScaleParameters() *ScaleParameters {
	if (!r.scaleEnable() || r.Spec.Capacity.ActivePlan.ScaleSchedulePlan.getReplicas() == 0) && r.Spec.Capacity.Total == nil {
		return nil
	}
	var parameter ScaleParameters
	coldratio, warmratio, hotratio, inactiveRatio, standbys, actives := r.getReserveCapacity()
	parameter.ActiveReplicas = actives
	parameter.StandByReplicas = standbys
	if standbys > 0 {
		parameter.StandByRatioMap = make(map[WorkerModeType]int)
		parameter.StandByRatioMap[WorkerModeTypeCold] = coldratio
		parameter.StandByRatioMap[WorkerModeTypeWarm] = warmratio
		parameter.StandByRatioMap[WorkerModeTypeHot] = hotratio
		parameter.StandByRatioMap[WorkerModeTypeInactive] = inactiveRatio
	}
	if r.DeletionTimestamp != nil {
		parameter.ActiveReplicas = 0
		parameter.StandByReplicas = 0
		parameter.StandByRatioMap = nil
	}
	if r.Spec.Capacity.ActivePlan.ScaleSchedulePlan != nil {
		parameter.ScalerResourcePoolName = r.Spec.Capacity.ActivePlan.ScaleSchedulePlan.RecyclePool
	}
	return &parameter
}

func (r RollingSet) scaleEnable() bool {
	if nil == r.Spec.Capacity.ActivePlan.ScaleSchedulePlan {
		return false
	}
	if nil == r.Spec.Capacity.ActivePlan.ScaleConfig || !r.Spec.Capacity.ActivePlan.ScaleConfig.Enable {
		return false
	}
	return true
}

func (r *RollingSet) getReserveCapacity() (int, int, int, int, int32, int32) {
	capacity := r.Spec.Capacity
	var actives, total int32
	var coldratio, warmratio, hotratio, inactiveRatio int
	/** 计算active **/
	if r.scaleEnable() && capacity.ActivePlan.ScaleSchedulePlan.getReplicas() > 0 {
		actives = capacity.ActivePlan.ScaleSchedulePlan.getReplicas()
		// 默认buffer比例使用弹性的
		recycleMode := capacity.ActivePlan.ScaleSchedulePlan.RecycleMode
		if capacity.ActivePlan.ScaleSchedulePlan.RecyclePool != "" {
			glog.V(4).Infof("set rollingset[%v/%v] to cold standby with RecyclePool[%v]",
				r.Namespace, r.Name, capacity.ActivePlan.ScaleSchedulePlan.RecyclePool)
			recycleMode = WorkerModeTypeCold //目前安pool预留只支持cold，即rr
		}
		switch recycleMode {
		case WorkerModeTypeCold:
			coldratio = 100
		case WorkerModeTypeWarm:
			warmratio = 100
		case WorkerModeTypeHot:
			hotratio = 100
		case WorkerModeTypeInactive:
			inactiveRatio = 100
		}
	} else {
		// 未开启弹性，全是active
		actives = r.GetRawReplicas()
	}

	/** 更新分池比例 **/
	// 使用预留buffer比例
	if capacity.BufferPlan.Distribute != nil && !capacity.BufferPlan.Distribute.isZero() {
		coldratio = int(capacity.BufferPlan.Distribute.Cold)
		warmratio = int(capacity.BufferPlan.Distribute.Warm)
		hotratio = int(capacity.BufferPlan.Distribute.Hot)
	}

	// 异常情况
	if coldratio == 0 && warmratio == 0 && hotratio == 0 && inactiveRatio == 0 {
		hotratio = 100
	}

	/** 计算replica总数 **/
	if capacity.Total != nil {
		total = *capacity.Total
	} else {
		total = r.GetRawReplicas()
	}

	buffers := total - actives
	if buffers < 0 {
		buffers = 0
	}
	return coldratio, warmratio, hotratio, inactiveRatio, buffers, actives
}

// IsRollingSetInRestarting IsRollingSetInRestarting
func (r *RollingSet) IsRollingSetInRestarting() bool {
	return r.Spec.InstanceID != r.Status.InstanceID && r.Status.InstanceID != ""
}

// RollingSetNeedGracefullyRestarting RollingSetNeedGracefullyRestarting
func (r *RollingSet) RollingSetNeedGracefullyRestarting() bool {
	if nil == r.Spec.ScaleSchedulePlan {
		return false
	}
	if nil != r.Spec.ScaleConfig && !r.Spec.ScaleConfig.Enable {
		return false
	}
	return r.Spec.ScaleSchedulePlan.SmoothRolling
}

// SubrsMetas ...
type SubrsMetas map[string]map[string]string

// RollingsetVerbose ...
type RollingsetVerbose struct {
	SubrsMetas SubrsMetas        `json:"subrsMetas"`
	Status     *RollingSetStatus `json:"status"`
	Replicas   []*ReplicaStatus  `json:"replicas"`
}
