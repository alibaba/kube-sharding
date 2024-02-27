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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Silent offline
const (
	SilentNodeAnnoKey               = "app.c2.io/silentnode"
	SilentTimeAnnoKey               = "app.c2.io/silent-time"
	ColdResourceUpdateState         = "app.c2.io/cold-resource-update-state"
	ColdResourceUpdateTargetVersion = "app.c2.io/cold-resource-update-target-version"
	Standby2ActiveTimeAnnoKey       = "app.c2.io/standby-to-active-time"
)

const (
	//ColdUpdateStateColdToWarm change cold to warm
	ColdUpdateStateColdToWarm = "cold_to_warm"
	//ColdUpdateStateWaitingWarmAssgin warm resource update
	ColdUpdateStateWaitingWarmAssgin = "waiting_warm_assagin"
	//ColdUpdateStateWarmResourceReplace warm resource replace
	ColdUpdateStateWarmResourceReplace = "warm_resource_replace"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WorkerNode is a abstraction layer for a base worker container
type WorkerNode struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorkerNodeSpec   `json:"spec"`
	Status WorkerNodeStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WorkerNodeList is a list of WorkerNode resources
type WorkerNodeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []WorkerNode `json:"items"`
}

// WorkerNodeSpec is a spec layer for a base workernode resource
type WorkerNodeSpec struct {
	// Selector is a label query over pods that should match the replica count.
	// Label keys and values that must match in order to be controlled by this replica set.
	// It must match the pod template's labels.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors
	Selector *metav1.LabelSelector `json:"selector"`
	// ResoVersion means the signature of Resources.
	ResVersion string `json:"resVersion"`
	// Version means the signature of VersionPlan.
	Version string `json:"version"`
	// config for resource and progress
	VersionPlan `json:",inline"`
	WorkerMode  WorkerModeType `json:"workerMode,omitempty"`
	// DependencyReady 依赖的服务是否ready
	DependencyReady *bool `json:"dependencyReady,omitempty"`
	// Is worker to delete.
	ToDelete bool `json:"toDelete"`

	// Is worker releasing .Different from ToDelete field, may be delete pod and create a new one.
	Releasing bool `json:"releasing"`
	// Is worker reclaim
	Reclaim bool `json:"reclaim"`

	OwnerGeneration int64 `json:"ownerGeneration,omitempty"`

	// specific the resource pool, enum Fixed Cyclical
	ResourcePool string `json:"resourcePool,omitempty"`
	DeletionCost int64  `json:"deletionCost,omitempty"`
	StandbyHours string `json:"standbyHours,omitempty"`

	BackupOfPod BackupOfPod `json:"backupOfPod,omitempty"`

	IsSpot bool `json:"isSpot,omitempty"`
}

// BackupOfPod point to logic current pod
type BackupOfPod struct {
	Name string `json:"name,omitempty"`
	Uid  string `json:"uid,omitempty"`
}

// WorkerModeType WorkerModeType
type WorkerModeType string

// RecycleMode
const (
	WorkerModeTypeActive  WorkerModeType = "active"
	WorkerModeTypeHot     WorkerModeType = "hotStandBy"
	WorkerModeTypeWarm    WorkerModeType = "warmStandBy"
	WorkerModeTypeCold    WorkerModeType = "coldStandBy"
	WorkerModeTypeUnknown WorkerModeType = "unknown"

	WorkerModeTypeInactive WorkerModeType = "inactive"
)

// VersionPlan  is the spec for a VersionPlan resource
type VersionPlan struct {
	SignedVersionPlan
	BroadcastPlan
}

// BroadcastPlan BroadcastPlan
type BroadcastPlan struct {
	WorkerSchedulePlan `json:",inline"`
	//CustomInfo suez服务信息
	CustomInfo string `json:"customInfo,omitempty"`
	//json string compressed of CustomInfo
	CompressedCustomInfo string `json:"compressedCustomInfo,omitempty"`
	UserDefVersion       string `json:"userDefVersion,omitempty"`
	// Online means is need to publish
	Online *bool `json:"online,omitempty"`
	// UpdatingGracefully means is need to unpublish before update
	UpdatingGracefully *bool `json:"updatingGracefully,omitempty"`
	//Preload advanced lv7,suez worker 是否做数据预加载
	Preload             bool   `json:"preload,omitempty"`
	IsDaemonSet         bool   `json:"isDaemonSet,omitempty"`
	CPU                 *int32 `json:"-"` // 内部变量，用于计算cpu可用度。不对外展示
	UpdatePlanTimestamp int64  `json:"updatePlanTimestamp,omitempty"`
	RowComplete         *bool  `json:"rowComplete,omitempty"`
}

// WorkerSchedulePlan define worker ready and qouta and other fields for worker schedule
type WorkerSchedulePlan struct {
	// Minimum number of seconds for which a newly created pod should be ready
	// +optional
	MinReadySeconds int32 `json:"minReadySeconds,omitempty"`

	// WarmupSeconds number of seconds for which a pod need warmup seconds
	// +optional
	WarmupSeconds int32 `json:"warmupSeconds,omitempty"`

	// ResourceMatchTimeout describe the max time for resize resource
	// +optional
	ResourceMatchTimeout int64 `json:"resourceMatchTimeout,omitempty"`
	// ProcessMatchTimeout describe the max time for process match
	// +optional
	ProcessMatchTimeout int64 `json:"processMatchTimeout,omitempty"`
	// ResourceMatchTimeout describe the max time for start new process
	// +optional
	WorkerReadyTimeout int64 `json:"workerReadyTimeout,omitempty"`

	// the maximum of creation of workers for recovery for a period of time
	BrokenRecoverQuotaConfig *BrokenRecoverQuotaConfig `json:"brokenRecoverQuotaConfig"`

	// relaiming strategy of workers
	RecoverStrategy RecoverStrategy `json:"recoverStrategy"`

	// Seconds to delay delete backup workernode
	// +optional
	DelayDeleteBackupSeconds int64 `json:"delayDeleteBackupSeconds,omitempty"`
}

// RecoverStrategy how to process with reclaiming worker node
type RecoverStrategy string

const (
	// DefaultRecoverStrategy 默认逻辑, 起新下老
	DefaultRecoverStrategy = ""

	// DirectReleasedRecoverStrategy 下老起新逻辑
	DirectReleasedRecoverStrategy = "directRelease"

	// NotRecoverStrategy 不recover
	NotRecoverStrategy = "notRecover"

	// RebuildDeadStrategy 容器dead时进行rebuild，下老起新
	RebuildDeadStrategy = "rebuildDead"
)

// BrokenRecoverQuotaConfig  is the spec for a SchedulePlan resource
type BrokenRecoverQuotaConfig struct {
	MaxFailedCount *int32 `json:"maxFailedCount"`
	TimeWindow     *int32 `json:"timeWindow"` // sec
}

// SignedVersionPlan SignedVersionPlan
type SignedVersionPlan struct {
	// ShardGroupVersion group version
	ShardGroupVersion string `json:"shardGroupVersion,omitempty"`
	// Template describes basic resource template.
	Template *HippoPodTemplate `json:"template"`
	//Signature
	Signature string `json:"signature,omitempty"`
	// RestartAfterResourceChange means is need to restart, after resource change
	RestartAfterResourceChange *bool             `json:"restartAfterResourceChange,omitempty"`
	BufferSelector             map[string]string `json:"bufferSelector,omitempty"`
	Cm2TopoInfo                string            `json:"cm2TopoInfo,omitempty"`
}

// HippoPodTemplate HippoPodTemplate
type HippoPodTemplate struct {
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              HippoPodSpec `json:"spec,omitempty"`
}

// HippoPodSpec defines the hippo pod spec
type HippoPodSpec struct {
	corev1.PodSpec           `json:",inline"`
	Containers               []HippoContainer `json:"containers"`
	HippoPodSpecExtendFields `json:",inline"`
	HippoVolumes             []json.RawMessage `json:"hippoVolumes,omitempty"`
}

// HippoContainer defines the extend hippo container fields
type HippoContainer struct {
	corev1.Container       `json:",inline"`
	ContainerHippoExterned `json:",inline"`
}

// ContainerHippoExterned defines the hippo container
type ContainerHippoExterned struct {
	Labels         map[string]string `json:"Labels,omitempty"`
	PreDeployImage string            `json:"PreDeployImage,omitempty"`
	HostConfig     json.RawMessage   `json:"HostConfig,omitempty"`
	Devices        []Device          `json:"devices,omitempty"`
	Configs        *ContainerConfig  `json:"configs,omitempty"`
	Alias          string            `json:"alias,omitempty"`
}

// ContainerConfig is config of container
type ContainerConfig struct {
	Ulimits           []Ulimit `json:"ulimits,omitempty"`
	RestartCountLimit int      `json:"restartCountLimit,omitempty"`
	StopGracePeriod   int      `json:"stopGracePeriod,omitempty"`

	CPUBvtWarpNs        string `json:"CPU_BVT_WARP_NS,omitempty"`
	CPUllcCache         string `json:"CPU_LLC_CACHE,omitempty"`
	NetPriority         string `json:"NET_PRIORITY,omitempty"`
	NoMemcgReclaim      string `json:"NO_MEMCG_RECLAIM,omitempty"`
	MemWmarkRatio       string `json:"MEM_WMARK_RATIO,omitempty"`
	MemForceEmpty       string `json:"MEM_FORCE_EMPTY,omitempty"`
	MemExtraBytes       string `json:"MEM_EXTRA_BYTES,omitempty"`
	MemExtraRatio       string `json:"MEM_EXTRA_RATIO,omitempty"`
	PredictDelaySeconds string `json:"PREDICT_DELAY_SECONDS,omitempty"`
}

// Ulimit Ulimit
type Ulimit struct {
	Name string `json:"Name,omitempty"`
	Soft int    `json:"Soft,omitempty"`
	Hard int    `json:"Hard,omitempty"`
}

// Device mount host device to container
type Device struct {
	PathOnHost        string
	PathInContainer   string
	CgroupPermissions string
}

// HippoPodSpecExtendFields HippoPodSpecExtendFields
type HippoPodSpecExtendFields struct {
	CpusetMode           string        `json:"cpusetMode,omitempty"`
	CPUShareNum          *int32        `json:"cpuShareNum,omitempty"`
	ContainerModel       string        `json:"containerModel,omitempty"`
	PackageInfos         []PackageInfo `json:"packageInfos,omitempty"`
	PredictDelayTime     int           `json:"predictDelayTime,omitempty"`
	RestartWithoutRemove *bool         `json:"restartWithoutRemove,omitempty"`
	NeedHippoMounts      *bool         `json:"needHippoMounts,omitempty"`
}

// PackageInfo extend spec, load package first
type PackageInfo struct {
	PacakgeType string `json:"pacakgeType"`
	PackageURI  string `json:"packageUri"`
}

// AllocatorSyncedStatus is worker status synced by allocator
type AllocatorSyncedStatus struct {
	// Ip  address. from pod ip.
	IP string `json:"ip,omitempty"`

	// HostIP  address. from host.
	HostIP string `json:"hostIP,omitempty"`

	// ResourceMatch describes if the running container match the requirement in plan, eg :cpu mem.
	ResourceMatch bool `json:"resourceMatch"`
	// ProcessMatch describes if the running process match the plan
	ProcessMatch bool `json:"processMatch"`

	// LastResourceNotMatchtime describe the last time resource not match
	// +optional
	LastResourceNotMatchtime int64 `json:"lastResourceNotMatchtime,omitempty"`
	// LastProcessNotMatchtime describe the last time process not match
	// +optional
	LastProcessNotMatchtime int64 `json:"lastProcessNotMatchtime,omitempty"`

	ResVersion string `json:"resVersion"`
	// Version means the signature of VersionPlan.
	Version        string `json:"version"`
	UserDefVersion string `json:"userDefVersion,omitempty"`

	// ProcessScore filled by allocators  .
	ProcessScore int32 `json:"processScore"`

	// The phase of a Worker is a simple, high-level summary of where the Worker is in its lifecycle.
	Phase WorkerPhase `json:"phase"`

	PackageStatus string `json:"packageStatus"`

	// Is replica should be reclaimed .
	// +optional
	Reclaim         bool `json:"reclaim"`
	InternalReclaim bool `json:"internalReclaim"`

	// ready status provide by allocator
	ProcessReady            bool           `json:"processReady,omitempty"`
	NamingRegisteredReady   bool           `json:"namingRegisteredReady,omitempty"`
	EntityName              string         `json:"entityName"`
	EntityAlloced           bool           `json:"entityAlloced"`
	ResourcePool            string         `json:"resourcePool"`
	WorkerMode              WorkerModeType `json:"workerMode,omitempty"`
	PodNotStarted           bool           `json:"-"`
	NotScheduleSeconds      int            `json:"-"`
	LastProcessNotReadytime int64          `json:"lastProcessNotReadytime,omitempty"`

	UnassignedReason         UnassignedReasonType `json:"unassignedReason,omitempty"`
	UnassignedMessage        string               `json:"unassignedMessage,omitempty"`
	HistoryUnassignedMessage string               `json:"historyUnassignedMessage,omitempty"`
	DeletionCost             int64                `json:"deletionCost,omitempty"`
	StandbyHours             string               `json:"standbyHours,omitempty"`
	EntityUid                string               `json:"entityUid"`
}

// WorkerPhase is a label for the condition of a worker at the current time.
type WorkerPhase string

const (
	// Pending means the worker has been accepted by the system, but one or more of the process
	// has not been started.
	Pending WorkerPhase = "Pending"
	// Running means that all process in the worker have been started ok.
	Running WorkerPhase = "Running"
	// Terminated means that worker have voluntarily terminated.
	Terminated WorkerPhase = "Terminated"
	// Failed means that all process in the worker terminated and at least one process has
	// terminated in a failure (exited with a non-zero exit code or was stopped by the system).
	Failed WorkerPhase = "Failed"
	// Unknown means that for some reason the state of the worker could not be obtained.
	Unknown WorkerPhase = "Unknown"
)

// AllocStatus define a alloc status type
type AllocStatus string

// These are the valid statuses of worker node.
const (
	WorkerUnAssigned AllocStatus = "UNASSIGNED"
	WorkerAssigned   AllocStatus = "ASSIGNED"
	WorkerLost       AllocStatus = "LOST"
	WorkerOfflining  AllocStatus = "OFFLINING"
	WorkerReleasing  AllocStatus = "RELEASING"
	WorkerReleased   AllocStatus = "RELEASED"
)

// WorkerNodeStatus is a status for a base workernode resource
type WorkerNodeStatus struct {
	AllocatorSyncedStatus `json:",inline"`

	// Notify publisher if it is needed to publish
	ServiceOffline bool `json:"serviceOffline"`
	Warmup         bool `json:"warmup"`
	InWarmup       bool `json:"inWarmup,omitempty"`
	NeedWarmup     bool `json:"needWarmup"`
	InUpdating     bool `json:"inUpdating"`
	InRestarting   bool `json:"inRestarting"`

	// WorkerReady describes if the running process is the health in plan.
	WorkerReady bool `json:"workerReady"`

	// ResourceMatchTimeout describe the last time process not match
	// +optional
	LastWorkerNotReadytime int64 `json:"lastWorkerNotReadytime,omitempty"`

	// Scores calculated by various indicators, for sort and decide which to release(choose the worst state) .
	Score int64 `json:"score"`

	// The phase of a Worker is a simple, high-level summary of where the Worker is in its lifecycle.
	AllocStatus AllocStatus `json:"allocStatus"`

	// The HealthStatus of a Worker is a simple, high-level summary of the health state of worker.
	HealthStatus HealthStatus `json:"healthStatus"`

	// ProcessStep describe the process step of replica controller
	ProcessStep ProcessStep `json:"processStep"`

	// HealthCondition, require to persist in memekube, see issue #96779
	// +optional
	HealthCondition HealthCondition `json:"healthCondition,omitempty"`

	// The HealthStatus of a Worker is a simple, high-level summary of the service state of worker.
	ServiceStatus ServiceStatus `json:"serviceStatus"`

	// list of ServiceConditions
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	ServiceConditions      []ServiceCondition `json:"serviceConditions,omitempty"`
	ServiceReady           bool               `json:"serviceReady,omitempty"`
	ServiceReadyForMinTime bool               `json:"serviceReadyForMinTime,omitempty"`

	// ServiceInfoMetas  service meta信息 存储cm2信息
	ServiceInfoMetas string `json:"serviceInfoMetas,omitempty" until:"1.0.0"`
	// The conditions array, the reason and message fields contain more detail about the worker's status.
	// list of Conditions
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []WorkerCondition `json:"conditions,omitempty"`

	Complete bool `json:"complete"`

	// Is worker releasing .Different from ToDelete field, may be delete pod and create a new one.
	ToRelease bool `json:"toRelease"`
	//LastUpdateStatusTime 最近一次更新时间
	BadReason BadReasonType `json:"badReason,omitempty"`
	SlotID    HippoSlotID   `json:"slotId,omitempty"`

	WorkerStateChangeRecoder `json:",inline"`

	// PodStandbyStatus standby 顺序信息 //DEPRECATED
	PodStandbyStatus PodStandbyStatus `json:"podStandbyStatus,omitempty"`

	// ServiceInfoMetas不持久化，看是否重启后load到了新值
	ServiceInfoMetasRecoverd bool `json:"serviceInfoMetasRecoverd,omitempty" until:"1.0.0"`
	// PodReady pod conditions type = Ready and status = True
	PodReady bool `json:"podReady"`
	// Id workerNodeId
	Name             string            `json:"name"`
	BadReasonMessage string            `json:"badReasonMessage,omitempty"`
	RestartRecords   map[string]string `json:"restartRecords,omitempty"`

	// isFedWorker
	IsFedWorker bool `json:"isFedWorker"`
}

// PodStandbyStatus PodStandbyStatus
type PodStandbyStatus struct {
	UseOrder     int64   `json:"useOrder,omitempty"`
	StandbyHours []int64 `json:"standbyHours,omitempty"`
}

// WorkerStateChangeRecoder WorkerStateChangeRecoder
type WorkerStateChangeRecoder struct {
	LastServiceReadyTime *int64      `json:"lastServiceReadyTime,omitempty"`
	LastUpdateStatusTime metav1.Time `json:"lastUpdateStatusTime,omitempty" until:"1.0.0"`
	BecomeCurrentTime    metav1.Time `json:"becomeCurrentTime,omitempty"`
	LastDeleteBackupTime int64       `json:"backupDelayDeleteTime,omitempty"`
	AssignedTime         int64       `json:"assignedTime,omitempty"`
	PodReadyTime         int64       `json:"podReadyTime,omitempty"`
	WorkerReadyTime      int64       `json:"lastWorkerReadyTime,omitempty"`
	LastWarmupStartTime  int64       `json:"lastWarmupStartTime,omitempty"`
	LastWarmupEndTime    int64       `json:"lastWarmupEndTime,omitempty"`
}

// HippoSlotID defines hippo slot id
type HippoSlotID struct {
	SlaveAddress string `json:"slave_address"`
	SlotID       int    `json:"slot_id"`
}

// WorkerCondition describes the state of a worker at a certain point.
type WorkerCondition struct {
	// Type of worker node condition.
	Type WorkerConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus `json:"status"`
	// The last time the condition transitioned from one status to another.
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// The reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty"`
	// A human readable message indicating details about the transition.
	// +optional
	Message string `json:"message,omitempty"`
}

// WorkerConditionType define a worker condition status type
type WorkerConditionType string

// ServiceCondition describes the service publish state of a worker at a certain point.
type ServiceCondition struct {
	// Type of worker node condition.
	Type RegistryType `json:"type"`
	// ServiceName of service to publish.
	ServiceName string `json:"serviceName"`
	// Status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus `json:"status"`
	Score  int64                  `json:"score"`
	// The last time the condition transitioned from one status to another.
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// The reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty"`
	// A human readable message indicating details about the transition.
	// +optional
	Message       string `json:"message,omitempty"`
	InWarmup      bool   `json:"inWarmup,omitempty"`
	DeleteCount   int    `json:"deleteCount,omitempty"`
	Name          string `json:"name"`
	StartWarmup   bool   `json:"startWarmup,omitempty"`
	Version       string `json:"version"`
	ServiceDetail string `json:"serviceDetail,omitempty"`
}

// RegistryType define a service condition status type
type RegistryType string

// These are valid conditions of a service.
const (
	ServiceNone      RegistryType = ""
	ServiceVipserver RegistryType = "Vipserver"
	ServiceCM2       RegistryType = "CM2"
	ServiceLVS       RegistryType = "LVS"
	ServiceSLB       RegistryType = "SLB"
	ServiceVPCSLB    RegistryType = "VPC_SLB"
	ServiceALB       RegistryType = "ALB"
	ServiceArmory    RegistryType = "Armory"
	ServiceSkyline   RegistryType = "Skyline"
	ServiceAntvip    RegistryType = "Antvip"
	ServiceMesh      RegistryType = "ServiceMesh"
	ServiceGeneral   RegistryType = "General"
)

// ServiceStatus define a service status type
type ServiceStatus string

// These are the valid statuses of worker service health.
const (
	ServiceUnKnown       ServiceStatus = "SVT_UNKNOWN"
	ServiceUnAvailable   ServiceStatus = "SVT_UNAVAILABLE"
	ServicePartAvailable ServiceStatus = "SVT_PART_AVAILABLE"
	ServiceAvailable     ServiceStatus = "SVT_AVAILABLE"
)

// HealthCondition describes the health state of a worker at a certain point.
type HealthCondition struct {
	// Type of worker node condition.
	Type HealthConditionType `json:"type"`
	// Status of the condition.
	Status HealthStatus `json:"status"`
	// The last time the condition transitioned from one status to another.
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// The reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty"`
	// A human readable message indicating details about the transition.
	// +optional
	Message string `json:"message,omitempty"`

	//时间戳，单位秒
	LastLostTime int64 `json:"lastLostTime,omitempty"`

	LostCount int32 `json:"lostCount,omitempty"`

	//Metas advanced lv7记录的节点返回信息
	Metas   map[string]string `json:"metas,omitempty" until:"1.0.0"`
	Checked bool              `json:"checked,omitempty" until:"1.0.0"`
	//json string compressed of Metas
	CompressedMetas string `json:"compressedMetas,omitempty" until:"1.0.0"`

	//Version workernode版本号
	// +optional
	Version string `json:"version,omitempty"`

	//WorkerStatus advanced lv7判断签名是否匹配
	WorkerStatus WorkerType `json:"workerStatus,omitempty"`
}

// WorkerType define a worker status type
type WorkerType string

// These are the valid statuses of worker health.
const (
	WorkerTypeUnknow   WorkerType = "WT_UNKNOW"
	WorkerTypeNotReady WorkerType = "WT_NOT_READY"
	WorkerTypeReady    WorkerType = "WT_READY"
)

// HealthConditionType define a health condition status type
type HealthConditionType string

// These are valid conditions of a health.
const (
	DefaultHealth     HealthConditionType = "DefaultHealth"
	Lv7Health         HealthConditionType = "Lv7Health"
	AdvancedLv7Health HealthConditionType = "AdvanceLv7Health"
)

// ProcessStep  describe the process step of controller
type ProcessStep string

// ProcessSteps
const (
	StepBegin                   ProcessStep = "StepBegin"
	StepProcessUpdateGracefully ProcessStep = "StepProcessUpdateGracefully"
	StepProcessPlan             ProcessStep = "StepProcessPlan"
	StepProcessHealthInfo       ProcessStep = "StepProcessHealthInfo"
	StepProcessServiceInfo      ProcessStep = "StepProcessServiceInfo"
	StepLost                    ProcessStep = "StepLost"
	StepToRelease               ProcessStep = "StepToRelease"
)

// HealthStatus define a health status type
type HealthStatus string

// These are the valid statuses of worker health.
const (
	HealthUnKnown HealthStatus = "HT_UNKNOWN"
	HealthLost    HealthStatus = "HT_LOST"
	HealthAlive   HealthStatus = "HT_ALIVE"
	HealthDead    HealthStatus = "HT_DEAD"
)

// BadReasonType BadReasonType
type BadReasonType int

// BadReasons
const (
	BadReasonNone            BadReasonType = 0
	BadReasonLost            BadReasonType = 1
	BadReasonDead            BadReasonType = 2
	BadReasonNotMatch        BadReasonType = 3
	BadReasonProcessNotMatch BadReasonType = 4
	BadReasonNotReady        BadReasonType = 5
	BadReasonReclaim         BadReasonType = 6
	BadReasonRelease         BadReasonType = 7
	BadReasonNotSchedule     BadReasonType = 8
	BadReasonProcessNotReady BadReasonType = 9
	BadReasonServiceReclaim  BadReasonType = 10
)

var badReasonMessage = map[BadReasonType]string{
	BadReasonNone:            "",
	BadReasonLost:            "lost",
	BadReasonDead:            "dead",
	BadReasonNotMatch:        "resourceNotMatch",
	BadReasonProcessNotMatch: "processNotMatch",
	BadReasonNotReady:        "notReady",
	BadReasonReclaim:         "reclaim",
	BadReasonRelease:         "release",
	BadReasonNotSchedule:     "notSchedule",
	BadReasonProcessNotReady: "processNotReady",
	BadReasonServiceReclaim:  "serviceReclaim",
}

// UnassignedReasonType UnassignedReasonType
type UnassignedReasonType int

// UnassignedReasons
const (
	UnassignedReasonNone              UnassignedReasonType = 0
	UnassignedReasonCreatePodErr      UnassignedReasonType = 1
	UnassignedReasonNotSchedule       UnassignedReasonType = 2
	UnassignedReasonQuotaNotEnough    UnassignedReasonType = 3
	UnassignedReasonQuotaNotExist     UnassignedReasonType = 4
	UnassignedReasonResourceNotEnough UnassignedReasonType = 5
)

var unassignedReasonMessage = map[UnassignedReasonType]string{
	UnassignedReasonNone:              "none",
	UnassignedReasonCreatePodErr:      "create_pod_error",
	UnassignedReasonNotSchedule:       "not_scheduled",
	UnassignedReasonQuotaNotEnough:    "quota_not_enough",
	UnassignedReasonQuotaNotExist:     "quota_not_exist",
	UnassignedReasonResourceNotEnough: "resource_not_enough",
}
