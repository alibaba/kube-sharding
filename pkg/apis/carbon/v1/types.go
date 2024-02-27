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

	"github.com/spf13/pflag"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// InitFlags is for explicitly initializing the flags.
func InitFlags(flagset *pflag.FlagSet) {
	flagset.StringVar(&SystemNamespace, "system-namespace", SystemNamespace, "system-namespace define the system namespace for c2")
	flagset.IntVar(&defaultMinReadySeconds, "spec:min-ready-seconds", defaultMinReadySeconds, "default MinReadySeconds, used for app to check health")
	flagset.IntVar(&defaultWarmupSeconds, "spec:warmup-seconde", defaultWarmupSeconds, "default WarmupSeconds, used for app to warmup")
	flagset.StringVar(&defaultSchedulerName, "spec:scheduler-name", defaultSchedulerName, "Default SchedulerName for proxy transter. hippo/unifiedScheduler/ack")
	flagset.StringVar(&defaultPodVersion, "spec:pod-version", defaultPodVersion, "Default Pod Version. v2.0/v2.5/v3.0")
	flagset.StringVar(&defaultVipPodVersion, "spec:vip-pod-version", defaultVipPodVersion, "Default Pod Version for pod with vip. v2.0/v2.5/v3.0")
	flagset.StringVar(&forcePodVersion, "spec:force-pod-version", forcePodVersion, "Force Pod Version. v2.0/v2.5/v3.0")
	flagset.StringVar(&defaultMetaSite, "spec:meta-site", defaultMetaSite, "default meta: sigma.ali/site")
	flagset.StringVar(&defaultMetaZone, "spec:meta-zone", defaultMetaZone, "default meta: meta.k8s.alipay.com/zone")
	flagset.StringVar(&defaultMetaInstanceGroup, "spec:instance-group", defaultMetaInstanceGroup, "default meta: sigma.ali/instance-group")
	flagset.StringVar(&defaultMetaAppName, "spec:app-name", defaultMetaAppName, "default meta: sigma.ali/app-name")
	flagset.StringVar(&defaultDeployUnit, "spec:deploy-unit", defaultDeployUnit, "default meta: sigma.ali/deploy-unit")
	flagset.StringVar(&injectLabels, "spec:inject-labels", injectLabels, "labels need to inject")
	flagset.StringVar(&HomeDiskPath, "spec:home-disk-path", HomeDiskPath, "default local home disk path")
	flagset.IntVar(&DefaultMaxFailedCount, "spec:max-recover-count", DefaultMaxFailedCount, "max recover failed worker count per 10 mins")
	flagset.IntVar(&DefaultMaxInstancePerNode, "spec:max-instance-per-node", DefaultMaxInstancePerNode, "")
	flagset.IntVar(&MaxNameLength, "spec:max-name-length", MaxNameLength, "")
	flagset.IntVar(&MaxGroupsPerApp, "spec:max-groups-per-app", MaxGroupsPerApp, "")
	flagset.IntVar(&MaxAppNameLength, "spec:max-app-name-length", MaxAppNameLength, "")
	flagset.IntVar(&MaxRoleNameLength, "spec:max-role-name-length", MaxRoleNameLength, "")
	flagset.StringVar(&DefaultExclusiveMode, "spec:exclusive-mode", DefaultExclusiveMode, "default value for app.hippo.io/exclusiveMode")
	flagset.IntVar(&NotScheduleTimeout, "spec:schedule-timeout-seconds", NotScheduleTimeout, "default value for scheduler to allocate pod")
	flagset.IntVar(&CompresseShardThreshold, "spec:compress-shard-threshold", CompresseShardThreshold, "if shardgroup has more shards than threshold, compress ShardTemplates")
	flagset.IntVar(&defaultResourceMatchTimeout, "spec:default-resource-match-timeout", defaultResourceMatchTimeout, "the max seconds wait pod resource match")
	flagset.IntVar(&defaultProcessMatchTimeout, "spec:default-process-match-timeout", defaultProcessMatchTimeout, "the max seconds wait pod process match")
	flagset.IntVar(&defaultSilenceSeconds, "spec:default-silence-seconds", defaultSilenceSeconds, "the default silence seconds before a node deleted")
	flagset.IntVar(&DefaultAutoReserveMinutes, "spec:default-auto-reserve-minutes", DefaultAutoReserveMinutes, "the default auto rr minutes")
	flagset.IntVar(&RecordNotScheduleMetricThreshold, "spec:record-not-schedule-threshold", RecordNotScheduleMetricThreshold, "the threshold for reocrd not schedule metric")
}

var (
	// SystemNamespace is used for c2 system
	SystemNamespace              = "kube-system"
	defaultMinReadySeconds       = 30 // asi metas
	defaultSilenceSeconds        = 300
	defaultStandby2ActiveSeconds = 10800
	defaultWarmupSeconds         int
	//DefaultAutoReserveMinutes is reserve time for delete pod
	DefaultAutoReserveMinutes = 3
	// DefaultMaxFailedCount is max recover failed count
	DefaultMaxFailedCount = 5
	// DefaultExclusiveMode  DefaultExclusiveMode
	DefaultExclusiveMode = ExclusiveModeTAG
	// DefaultMaxInstancePerNode DefaultMaxInstancePerNode
	DefaultMaxInstancePerNode = 1

	// 调度版本
	defaultSchedulerName string
	defaultPodVersion    string
	defaultVipPodVersion string
	forcePodVersion      string

	// 注入labels
	injectLabels = ""
	// 蚂蚁五元组
	defaultMetaSite          string
	defaultMetaZone          string
	defaultMetaInstanceGroup string
	defaultMetaAppName       string
	defaultDeployUnit        string

	// MaxNameLength 最终生成的podname不能过长。不同版本长度限制不一样，目前按照1.14版本:253，留10个字节作为buffer, kubelet 创建目录 namespace + name + uid 不能超过 256 反推回来podname应在155以内
	MaxNameLength = 150
	// MaxNamespaceLength 最终生成的namespacee不能过长。不同版本长度限制不一样，目前按照1.14版本:253，留10个字节作为buffer
	MaxNamespaceLength = 63
	// MaxAppNameLength MaxAppNameLength
	MaxAppNameLength = 140
	// MaxRoleNameLength MaxRoleNameLength
	MaxRoleNameLength = 120

	// MaxGroupsPerApp is used to limit the maximum number of app
	MaxGroupsPerApp = 40000

	// HomeDiskPath HomeDiskPath
	HomeDiskPath = "/home,/home/admin,/home/admin/hippo"
	// NotScheduleTimeout for recover
	NotScheduleTimeout = 0

	CompresseShardThreshold = 256

	// RecordNotScheduleMetricThreshold for record metric
	RecordNotScheduleMetricThreshold = 300
)

// Unique Label Keys used to select obj for
const (
	// SchedulerName
	SchedulerNameHippo   = "hippo"
	SchedulerNameAsi     = "default-scheduler"
	SchedulerNameDefault = "default-scheduler"
	SchedulerNameAck     = "ack"
	SchedulerNameC2      = "c2-allocator"

	// subrs
	LabelKeySchedulerName     = "app.c2.io/scheduler-name"
	LabelKeyMigratedFromSubrs = "app.c2.io/migrated-from-subrs"
	DefaultShortSubRSLabelKey = "app.c2.io/short-subrs-name"
	DefaultSubRSLabelKey      = "app.c2.io/subrs-name"
	SubrsTextPrefix           = "subrs.c2.io/"

	// quickonline
	LabelKeyPreInactive      = "app.c2.io/pre-inactive"
	LabelKeyInactive         = "app.c2.io/inactive"
	LabelKeyOnlined          = "app.c2.io/onlined"
	LabelAllContainersStatus = "alibabacloud.com/all-containers-status"

	LabelBizPodFlag = "alibabacloud.com/biz-pod"

	// unassigned info
	LabelKeyUnassignedReason = "app.c2.io/unassigned-reason"
	LabelKeyUnassignedMsg    = "app.c2.io/unassigned-msg"
	// DefaultRollingsetUniqueLabelKey is the default key of the selector that is added
	// to existing RCs (and label key that is added to its pods) to prevent the existing RCs
	// to select new pods (and old pods being select by new RC).
	DefaultRollingsetUniqueLabelKey string = "rs-version-hash"
	DefaultReplicaUniqueLabelKey    string = "replica-version-hash"
	DefaultGangUniqueLabelKey       string = "gang-version-hash"
	DefaultWorkernodeUniqueLabelKey string = "worker-version-hash"
	DefaultShardGroupUniqueLabelKey string = "shardgroup" // memkube default batchkey
	DefaultSlotsetUniqueLabelKey    string = "ss-version-hash"

	// hippo label keys
	LabelAsiOwner         = "alibabacloud.com/owner"
	LabelKeyAppName       = "app.hippo.io/app-name"
	LabelKeyZoneName      = "app.hippo.io/zone-name"
	LabelKeyAppChecksum   = "app.hippo.io/app-checksum" // for admin-manager
	LabelKeyClusterName   = "app.hippo.io/cluster-name"
	LabelKeyRoleName      = "app.hippo.io/role-name"
	LabelKeySubrsName     = "app.c2.io/subrs-short-name"
	LabelGangPartName     = "app.c2.io/gang-part-name"
	LabelGangMainPart     = "app.c2.io/main-part"
	AnnoGangPlan          = "app.c2.io/gang-plan"
	AnnoBecomeCurrentTime = "app.hippo.io/become-current-time"
	// NOTE: this key is only used in c2, bad key name
	LabelKeyGroupName          = "app.hippo.io/group-name"
	LabelKeyRequirementID      = "app.hippo.io/requirementId"
	LabelKeyExlusive           = "app.hippo.io/exclusive-labels"
	LabelKeyDeclare            = "app.hippo.io/declare-label"
	LabelKeyQuotaGroupID       = "app.hippo.io/quotaGroupId"
	LabelKeyMaxInstancePerNode = "app.hippo.io/maxInstancePerNode"
	LabelKeyExclusiveMode      = "app.hippo.io/exclusiveMode"
	LabelKeyHippoPreference    = "app.hippo.io/preference"
	LabelKeyCarbonJobName      = "app.hippo.io/carbonjob-name" // NOTE: the value maybe a hash by LabelHashValue
	LabelKeyRoleShortName      = "app.hippo.io/roleShortName"
	LabelKeyAppShortName       = "app.hippo.io/appShortName"
	LabelKeyAutoUpdate         = "app.hippo.io/autoUpdate"
	LabelKeyBackupPod          = "app.hippo.io/backupPod"
	LabelKeyPodVersion         = "app.hippo.io/pod-version"
	LabelKeyPreemptable        = "app.hippo.io/preemptable"
	LabelKeyLaunchSignature    = "app.hippo.io/launch-signature"
	LabelKeyPackageChecksum    = "app.hippo.io/package-checksum"
	LabelKeyServiceVersion     = "app.c2.io/service-version"

	LabelKeyInstanceGroup        = "sigma.ali/instance-group"
	LabelKeyAppStage             = "sigma.alibaba-inc.com/app-stage"
	LabelKeyAppUnit              = "sigma.alibaba-inc.com/app-unit"
	LabelKeySkylineInitContainer = "app.c2.io/skyline-init-container"
	LabelKeyDependentSkyline     = "app.c2.io/dependent-skyline"
	LabelKeySigmaAppName         = "sigma.ali/app-name"    // compatible to old sigma protocol
	LabelKeySigmaDeplyUnit       = "sigma.ali/deploy-unit" // compatible
	LabelInjectSidecar           = "alibabacloud.com/inject-staragent-sidecar"
	LabelSigmaInjectSidecar      = "sigma.ali/inject-staragent-sidecar"
	LabelKeyReplaceNodeVersion   = "app.c2.io/replace-node-version"

	LabelKeyFedInstanceGroup = "sigma.ali/fed-instance-group"
	LabelKeyFedAppStage      = "sigma.alibaba-inc.com/fed-app-stage"
	LabelKeyFedAppUnit       = "sigma.alibaba-inc.com/fed-app-unit"

	// asi
	LabelAsiAppName             = "app.kubernetes.io/name"
	LabelDeployUnit             = "app.alibabacloud.com/deploy-unit"
	LabelObjSource              = "app.hippo.com/object-source"
	LabelConstraint             = "alibabacloud.com/pod-constraint-name"
	LabelHippoPodVersion        = "app.hippo.io/pod-version"
	LabelDisableAdmission       = "alibabacloud.com/disable-admission"
	LabelAntBizGroup            = "meta.k8s.alipay.com/biz-group-id"
	LabelAntBizName             = "meta.k8s.alipay.com/biz-name"
	LabelCPUBindStrategy        = "alibabacloud.com/cpuBindStrategy"
	LabelAsiMountGPU            = "alibabacloud.com/mount-nvidia-driver-compatible"
	LabelInplaceUpdateForbidden = "alibabacloud.com/inplace-update-forbidden"
	LabelKeyNoMountCgroup       = "alibabacloud.com/no-mount-cgroup"

	AnnotationCPUQuota = "pod.beta1.alibabacloud.com/container-cpu-quota-unlimit"
	AnnotationSSH      = "pod.beta1.alibabacloud.com/sshd-in-staragent"
	AnnotationSN       = "pods.sigma.alibaba-inc.com/inject-pod-sn"

	// c2 skyline register
	AnnotationKeyC2HostnameTemplate = "app.c2.io/hostname-template"
	LabelKeyC2InstanceGroup         = "app.c2.io/instance-group"
	LabelKeyC2AppStage              = "app.c2.io/app-stage"
	LabelKeyC2AppUnit               = "app.c2.io/app-unit"

	FinaliezrKeyC2SkylineRegister      = "app.c2.io/skyline-register"
	FinaliezrKeyC2SkylinePreProtection = "protection-delete.pod.c2/naming-pre-register"
	FinaliezrKeyC2SkylineProtection    = "protection-delete.pod.c2/naming"
	ConditionKeyC2Initialize           = "app.c2.io/initialize"

	// serverless
	LabelPoolStatus              = "serverless.io/pool-status"
	LabelRuntimeAppName          = "serverless.io/runtime-app-name"
	AnnotationSubrsMetas         = "serverless.io/subrs-metas"
	LabelSubrsEnable             = "serverless.io/subrs-enable"
	LabelServerlessAppName       = "serverless.io/app-name"
	LabelServerlessInstanceGroup = "serverless.io/instance-group"
	LabelServerlessAppId         = "serverless.io/app-id"
	LabelServerlessRecycleModel  = "serverless.io/recycle-mode"
	AnnotationPoolSpec           = "serverless.io/pool-spec"

	AnnoServerlessAppPlanStatus = "serverless.io/app-plan-status"
	AnnoServerlessAppStatus     = "serverless.io/app-status"

	ServerlessPlatform = "serverless"

	// Pool Status
	PoolStatusWaiting = "waiting"
	PoolStatusCreated = "created"
	PoolStatusDeleted = "deleted"

	// asi label keys
	LabelKeyPlatform            = "alibabacloud.com/platform"
	LabelKeyBU                  = "alibabacloud.com/bu"
	LabelPodEvictionProcessor   = "pod.sigma.ali/eviction-processor"
	LabelPodEvictionLegacySigma = "pod.sigma.ali/eviction" // legacy sigma eviction protocol
	LabelKeyPodSN               = "sigma.ali/sn"
	LabelGpuCardModel           = "alibabacloud.com/gpu-card-model"
	LabelFinalStateUpgrading    = "inplaceset.beta1.sigma.ali/final-state-upgrading"

	// These *Hash below are used to select in proxy
	LabelKeyC2GroupName   = "app.c2.io/group-name"
	LabelKeyRoleNameHash  = "app.c2.io/role-name-hash"
	LabelKeyGroupNameHash = "app.c2.io/group-name-hash"
	LabelKeyZoneNameHash  = "app.c2.io/zone-name-hash"
	LabelKeyAppNameHash   = "app.c2.io/app-name-hash"

	LabelKeyBufferCapacityNameHash = "app.c2.io/buffer-capacity-name-hash"

	// hippo condition keys
	ConditionKeyScheduleStatusExtend = "app.hippo.io/ScheduleStatusExtend"
	ConditionKeyPodStatusExtend      = "app.hippo.io/PodStatusExtend"
	ConditionKeyPodPreference        = "app.hippo.io/PodPreference"
	ConditionKeyPodNetworkStatus     = "app.hippo.io/PodNetworkStatus"

	// ResourceIP means ip resource
	HippoResourcePrefix = "resource.hippo.io/"
	ResourceIP          = "resource.hippo.io/ip"
	ResourceCPU         = "cpu"
	ResourceMEM         = "memory"
	ResourceGPUMEM      = "gpu-memory"
	ResourceGPUCore     = "gpu-core"
	ResourceENI         = "eni"
	ResourceDISKPrefix  = "resource.hippo.io/disk_size_"
	ResourceDISK        = "resource.hippo.io/disk_size"
	ResourceGPU         = "nvidia.com/gpu"
	ResourceFPGA        = "xilinx.com/fpga"
	ResourceNPU         = "alibaba.com/npu"
	ResourcePOV         = "alibaba.com/pov"
	ResourceDiskRatio   = "alibabacloud.com/disk_ratio"
	ResourceRequestCpu  = "requests.cpu"

	ASIResourceKeyIP       corev1.ResourceName = "alibabacloud.com/ip"
	ASIResourceKeyGPU      corev1.ResourceName = "alibabacloud.com/gpu"
	ASIResourceKeyGPUMEM   corev1.ResourceName = "alibabacloud.com/gpu-mem"
	ASIResourceKeyGPUCore  corev1.ResourceName = "alibabacloud.com/gpu-core"
	ASIResourceKeyFPGA     corev1.ResourceName = "alibabacloud.com/fpga"
	ASIResourceKeyNPU      corev1.ResourceName = "alibabacloud.com/npu"
	ASIResourceKeyPOV      corev1.ResourceName = "alibabacloud.com/pov"
	ASIResourceKeyDISK     corev1.ResourceName = "ephemeral-storage"
	ASIResourceKeyUnderlay corev1.ResourceName = "alibabacloud.com/underlay"

	// c2 extend labels keys
	ControllersShardKey   string = "app.c2.io/controllers-shard"
	WorkerRoleKey         string = "app.c2.io/worker-role"
	GroupTypeKey          string = "app.c2.io/group-type"
	WorkerReferReplicaKey string = "app.c2.io/worker-refer"
	BizKeyPrefix          string = "app.c2.io/biz-"
	AnnoKeyRoleRequest    string = "app.c2.io/role-request"
	// 次前缀开头的annotation不参与rolling计算
	BizDetailKeyPrefix         string = "app.c2.io/biz-detail"
	BizDetailKeyPartitionCount string = "app.c2.io/biz-detail-partition-count"
	BizDetailKeyReady          string = "app.c2.io/biz-detail-ready"
	// cm2 metas
	BizCm2MetasKeyPrefix string = "app.c2.io/biz-cm2-metas-"

	AnnoKeyEvictFailedMsg string = "alibabacloud.com/evict-failed-msg"

	BizDetailKeyGangInfo string = "app.c2.io/biz-detail-ganginfo"
	// BizMeatsKeys
	BizMeatsKeyHIPPOCluster     = "HIPPO_CLUSTER"
	BizMeatsKeyHIPPOAPP         = "HIPPO_APP"
	BizMeatsKeyHIPPORole        = "HIPPO_ROLE"
	BizMeatsKeyC2Group          = "C2_GROUP"
	BizMeatsKeyC2Role           = "C2_ROLE"
	CM2MetaWeightKey            = "target_weight"
	CM2MetaTopoInfo             = "topo_info"
	CM2MetaWeightWarmupValue    = "-100"
	CM2MetaWeightWarmupValueInt = -100

	// c2 envs
	C2EnvsRestart            = "C2_ENVS_RESTART__"
	C2EnvContainerWorkingDir = "WORKING_DIR"

	ContainerConfigEnvPrefix = "containerConfig_"

	// c2 extend labels values
	CurrentWorkerKey        string = "current"
	BackupWorkerKey         string = "backup"
	GroupTypeShardGroup     string = "group"
	GroupTypeRollingset     string = "role"
	GroupTypeGangRollingset string = "gang-role"
	GroupTypeReplica        string = "replica"

	// FinalizerSmoothDeletion Finalizer字段
	FinalizerSmoothDeletion string = "smoothDeletion"

	// FinalizerPoolDeletion 删除Pool
	FinalizerPoolDeletion string = "serverless.io.pool"

	// To store the original hippo application description
	AnnotationKeyApplicationDescription = "app.c2.io/application-description"
	// To store the user keys
	AnnotationC2DeclaredKeys    = "app.c2.io/c2-declared-keys"
	AnnotationC2DeclaredEnvKeys = "app.c2.io/c2-declared-env-keys"

	AnnotationKeySpecExtend     = "app.hippo.io/spec-extend"
	AnnotationKeySlotID         = "app.hippo.io/slotID"
	AnnotationKeyOldPriority    = "app.hippo.io/old-priority"
	AnnotationC2ResourceVersion = "app.c2.io/pod-resource-version"
	AnnotationC2PodVersion      = "app.c2.io/pod-version"
	AnnotationKeyPriorityV1     = "app.hippo.io/priority-v1.0"
	// For bizpod only right now
	AnnotationC2ContainerRuntime = "app.c2.io/container-runtime"
	AnnotationC2Spread           = "app.c2.io/spread"

	AnnotationOldPodLevelStorage = "sigma.ali/use-unified-pv"
	// 配置文档：https://yuque.antfin-inc.com/docs/share/cb242f8e-5a41-447d-8023-de1bbf029419?#
	AnnotationPodLevelStorageMode = "sigma.ali/app-storage-mode"

	// asi annotation key
	AnnotationStandbyStableStatus = "alibabacloud.com/standby-stable-status"

	AnnotationBufferCapacityResVersion   = "app.c2.io/buffer-capacity-res-version"
	AnnotationBufferCapacityMainRoleName = "app.c2.io/buffer-capacity-main-role-name"

	AnnotationInjectSystemd = "alibabacloud.com/need-inject-systemd-containers"

	AnnotationHostNameTemplate = "pod.beta1.sigma.ali/hostname-template"

	AnnotationScalerResourcePool = "alibabacloud.com/scaler-resource-pool"

	AnnotationPodDeletionCost   = "controller.kubernetes.io/pod-deletion-cost"
	AnnotationPodC2DeletionCost = "controller.kubernetes.io/original-pod-deletion-cost"

	// AnnotationCandidateClusters aligns with fed clusters filtering key.
	AnnotationCandidateClusters = "alibabacloud.com/cluster"
	// node-topology
	AnnotationNodeTopologyScope = "app.c2.io/node-topology-scope"

	//用户指定需要使用的云盘的策略，目前只有role-storage，即按照role级别分配一个，所有pod均使用该pvc
	AppStorageStrategyRole         = "role-storage"
	AnnotationAppStoragetStrategy  = "app.c2.io/app-storage-strategy"
	AnnotationAppStoragetMountPath = "app.c2.io/app-storage-mount-path"
	AnnotationAppStoragetSize      = "app.c2.io/app-storage-size"
	AnnotationRolePvcName          = "app.c2.io/app-role-pvc-name"
	StorageMountEnvKey             = "C2_NET_DISK_MOUNT_PATH"

	AnnotationWorkerUpdateStatus = "app.c2.io/update-status"
	AnnotationWorkerUpdatePhase  = "app.c2.io/update-phase-status"
	AnnotationBufferSwapRecord   = "app.c2.io/buffer-swap-record"
	// TopologyScop types
	TopologyScopGang = "gang"

	PodVersion2     = "v2.0"
	PodVersion25    = "v2.5"
	PodVersion3     = "v3.0"
	LazyPodVersion3 = "app.c2.io/lazy-v3.0"

	// inject identifier
	LabelKeyIdentifier = "app.c2.io/inject-identifier"
	IdentifierEnvKey   = "WORKER_IDENTIFIER_FOR_CARBON"

	C2AdminRoleTag    = "internal_appmaster_resource_tag"
	HippoAdminRoleTag = "__internal_appmaster_resource_tag__"

	WorkerNameSuffixA = "-a"
	WorkerNameSuffixB = "-b"

	UnifiedStorageInjectedVolumeName = "autogen"
	//
	GroupRollingSetBufferShortName = "grs-buffer"

	MainContainer = "main"

	AnnotationInitSpec         = "alibabacloud.com/init-spec"
	AnnotationInitResourceSpec = "alibabacloud.com/init-resource-spec"

	DefaultNamespace = "default"

	DirectLabelCopyMappingKey = "_direct"

	ExponentialBackoff = 10

	DefaultRestartCountLimit = 10

	// serverless & AIOS app meta info, see doc https://aliyuque.antfin.com/search-infra/pufssx/bux3yx7bw9626qsc for details
	LabelServerlessBizPlatform = "serverless.io/biz-platform"
	LabelServerlessAppStage    = "serverless.io/app-stage"
	AnnoServerlessAppDetail    = "serverless.io/app-detail-url" // only in annotation

	MaxFixedCost    int64 = 9999
	MinFixedCost    int64 = 5000
	MaxCyclicalCost int64 = 4999
	MinCyclicalCost int64 = 1

	AnnotationFiberScenes = "app.c2.io/fiber-scenes"
	LabelResourcePool     = "app.alibabacloud.com/pod-scale-kind"

	// spot-instance
	AnnotationC2SpotTarget = "app.c2.io/spot-target"
	LabelIsBackupPod       = "alibabacloud.com/is-backup"
	AnnotationBackupOf     = "alibabacloud.com/backup-of"
	// gang
	ScheduleGangRolling = "gang_rolling"

	LabelRuntimeID = "sigma.ali/runtime-id"

	C2PlatformKey = "app.c2.io/platform"

	ConditionServerlessAppReady = "ServerlessAppReady"
	ConditionWorkerNodeReady    = "WorkerNodeReady"

	ServerlessPlanSuccess    int32 = 1
	ServerlessPlanInProgress int32 = 2
	ServerlessPlanFailed     int32 = 3
	ServerlessServiceReady   int32 = 3

	FinalizerPodProtectionFmt = "protection.pod*"
)

// MaxControllerShards descirbe max controllers in one namespace
const (
	MaxControllerShards = 16
)

// StripVersionLabelKeys represents label/annotation keys which will be ignored in version computation
var StripVersionLabelKeys []string = []string{
	LabelServerlessAppName,
	LabelServerlessInstanceGroup,
	LabelServerlessBizPlatform,
	LabelServerlessAppStage,
	AnnoServerlessAppDetail,
}

// C2InternalMetaPrefix  This is meant to be constant! Please don't mutate it!
var C2InternalMetaPrefix = []string{"serverless.io", "app.hippo.io", "app.c2.io"}
var C2InternalMetaBlackList = []string{"serverless.io/rollout-id", "serverless.io/rollout-batch-id"}

var SubrsInheritanceMetaPrefix = []string{LabelServerlessAppId, LabelServerlessInstanceGroup, LabelServerlessAppName, LabelRuntimeAppName,
	"serverless.io/env-name", "app.hippo.io/appShortName", "app.hippo.io/roleShortName"}

var PodInheritWorkerMetaPrefix = []string{AnnoKeyEvictFailedMsg}

// ProvidorType is a type for ProvidorType
type ProvidorType string

const (
	// ProvidorTypePod is a type for pod providor
	ProvidorTypePod ProvidorType = "pod"
	// ProvidorTypeReplica is a type for replica providor
	ProvidorTypeReplica ProvidorType = "replica"
)

// HealthCheckerConfig 健康检查配置
type HealthCheckerConfig struct {
	// // Label selector for workernode.
	// // It must match the workernode's labels.
	// Selector map[string]string `json:"selector"`

	// Type of worker node condition.
	Type HealthConditionType `json:"type"`

	// lv7 HealthChecker 规则参数
	// +optional
	Lv7Config *Lv7HealthCheckerConfig `json:"lv7Config"`
}

// Lv7HealthCheckerConfig lv7配置
type Lv7HealthCheckerConfig struct {
	// +optional
	LostCountThreshold int32 `json:"lostCountThreshold,omitempty" protobuf:"varint,4,opt,name=lostCountThreshold"`

	// 超时时间， 单位秒
	// +optional
	LostTimeout int64 `json:"lostTimeout,omitempty" protobuf:"varint,8,opt,name=lostTimeout"`

	// Path to access on the HTTP server.
	// +optional
	Path string `json:"path,omitempty"`

	// Name or number of the port to access on the container.
	// Number must be in the range 1 to 65535.
	// Name must be an IANA_SVC_NAME.
	Port intstr.IntOrString `json:"port" protobuf:"bytes,2,opt,name=port"`

	// // http请求超时事件, 单位ms
	// // default 100
	// // +optional
	// HttpTimeout int64 `json:"httpTimeout,omitempty" protobuf:"varint,8,opt,name=httpTimeout"`
}

// GeneralTemplate describes a general template
type GeneralTemplate struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`
}

// ContainerHippoExternedWithName used for commit pod to hippo
type ContainerHippoExternedWithName struct {
	Name                   string `json:"name,omitempty"`
	InstanceID             string `json:"instanceId,omitempty"`
	ContainerHippoExterned `json:",inline"`
}

// RollingsetHippoExterned defines the hippo extend fields
type RollingsetHippoExterned struct {
	ApplicationID string `json:"applicationId"`
	QuotaGroupID  string `json:"quotaGroupId"`
}

// CpusetModeType defines cpu mode
type CpusetModeType string

// cpu modes enum
const (
	CpusetReserved  CpusetModeType = "RESERVED"
	CpusetExclusive CpusetModeType = "EXCLUSIVE"
	CpusetShare     CpusetModeType = "SHARE"
	CpusetNone      CpusetModeType = "NONE"
)

// ContainerModelType defines container mode
type ContainerModelType string

// containers modes enum
const (
	ContainerDocker        ContainerModelType = "DOCKER"
	ContainerCgroupProcess ContainerModelType = "CGROUPPROCESS"
	ContainerDockerVM      ContainerModelType = "DOCKERVM"
	ContainerKVM           ContainerModelType = "KVM"
)

// HippoPodSpecExtend defines hippos pods spec extend
type HippoPodSpecExtend struct {
	Containers               []ContainerHippoExternedWithName `json:"containers"`
	InstanceID               string                           `json:"instanceId,omitempty"`
	HippoPodSpecExtendFields `json:",inline"`
	Devices                  []Device          `json:"device,omitempty"`
	Volumes                  []json.RawMessage `json:"volumes,omitempty"`
}

// ScheduleStatusExtend hippo extend fileds
type ScheduleStatusExtend struct {
	HealthScore int32 `json:"healthScore"`
	DeclareTime int64 `json:"declareTime"`
	//ScheduleDiagnosis string      `json:"scheduleDiagnosis"`
	RequirementID    string      `json:"app.hippo.io/requirementId"`
	RequirementMatch bool        `json:"app.hippo.io/requirementMatch"`
	NodeDomain       string      `json:"nodeDomain"`
	SlotID           HippoSlotID `json:"slotId"`
	UID              string      `json:"uid"`
	NodeName         string      `json:"nodeName"`
}

// ContainerStatusHippoExterned defines the hippo container
type ContainerStatusHippoExterned struct {
	Name       string `json:"name,omitempty"`
	InstanceID string `json:"instanceId,omitempty"`
}

// PodStatusExtend defines the hippo container
type PodStatusExtend struct {
	InstanceID     string                         `json:"instanceId,omitempty"`
	CpusetMode     string                         `json:"cpusetMode,omitempty"`
	ContainerModel string                         `json:"containerModel,omitempty"`
	Containers     []ContainerStatusHippoExterned `json:"containers"`
}

// ContainerResource means container resource require
type ContainerResource struct {
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}

// PodResource means pod resource require
type PodResource struct {
	Containers   []ContainerResource `json:"containers" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,2,rep,name=containers"`
	NodeSelector map[string]string   `json:"nodeSelector,omitempty" protobuf:"bytes,7,rep,name=nodeSelector"`
	NodeName     string              `json:"nodeName,omitempty" protobuf:"bytes,10,opt,name=nodeName"`
	Affinity     *corev1.Affinity    `json:"affinity,omitempty" protobuf:"bytes,18,opt,name=affinity"`
	Tolerations  []corev1.Toleration `json:"tolerations,omitempty" protobuf:"bytes,22,opt,name=tolerations"`
	Priority     *int32              `json:"priority,omitempty" protobuf:"bytes,25,opt,name=priority"`
	CpusetMode   string              `json:"cpusetMode,omitempty"`
	CPUShareNum  *int32              `json:"cpuShareNum,omitempty"`
}

// ContainerInstanceField means container instance resource
type ContainerInstanceField struct {
	Name           string                      `json:"name"`
	Alias          string                      `json:"alias,omitempty"`
	Image          string                      `json:"image,omitempty"`
	Command        []string                    `json:"command,omitempty"`
	Args           []string                    `json:"args,omitempty"`
	WorkingDir     string                      `json:"workingDir,omitempty"`
	Ports          []corev1.ContainerPort      `json:"ports,omitempty"`
	EnvFrom        []corev1.EnvFromSource      `json:"envFrom,omitempty"`
	Env            []corev1.EnvVar             `json:"env,omitempty"`
	VolumeMounts   []corev1.VolumeMount        `json:"volumeMounts,omitempty"`
	VolumeDevices  []corev1.VolumeDevice       `json:"volumeDevices,omitempty"`
	Lifecycle      *LifecycleInstanceField     `json:"lifecycle,omitempty"`
	Resources      corev1.ResourceRequirements `json:"resources,omitempty"`
	Labels         map[string]string           `json:"Labels,omitempty"`
	PreDeployImage string                      `json:"PreDeployImage,omitempty"`
	HostConfig     json.RawMessage             `json:"HostConfig,omitempty"`
	Devices        []Device                    `json:"device,omitempty"`
	Configs        *ContainerConfig            `json:"configs,omitempty"`
}

// LifecycleInstanceField used for compute instanceid
type LifecycleInstanceField struct {
	PostStart *corev1.LifecycleHandler `json:"postStart,omitempty" protobuf:"bytes,1,opt,name=postStart"`
}

// PodInstanceField means container instance resource
type PodInstanceField struct {
	Containers     []ContainerInstanceField `json:"containers,omitempty"`
	ContainerModel string                   `json:"containerModel,omitempty"`
	PackageInfos   []PackageInfo            `json:"packageInfos,omitempty"`
	Volumes        []corev1.Volume          `json:"volumes,omitempty"`
	CpusetMode     string                   `json:"cpusetMode,omitempty"`
	CPUShareNum    *int32                   `json:"cpuShareNum,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TemporaryConstraint is nodeaffinity with ttl
type TemporaryConstraint struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TemporaryConstraintSpec   `json:"spec"`
	Status TemporaryConstraintStatus `json:"status"`
}

// TemporaryConstraintSpec TemporaryConstraintSpec
type TemporaryConstraintSpec struct {
	Rule *TemporaryConstraintRule `json:"temporaryConstraintRule,omitempty"`
	APP  string                   `json:"app,omitempty"`
	Role string                   `json:"role,omitempty"`
	IP   string                   `json:"ip,omitempty"`
}

// TemporaryConstraintStatus TemporaryConstraintStatus
type TemporaryConstraintStatus struct{}

// TemporaryConstraintRule TemporaryConstraintRule
type TemporaryConstraintRule struct {
	NodeAffinity *corev1.NodeAffinity `json:"nodeAffinity,omitempty"`
	DeadTime     int64                `json:"deadTime,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TemporaryConstraintList is a list of WorkerNode resources
type TemporaryConstraintList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []TemporaryConstraint `json:"items"`
}

// KeysMayBeAdjust the keys c2 declared
type KeysMayBeAdjust struct {
	VolumeKeys               []string `json:"volumeKeys,omitempty"`
	TolerationKeys           []string `json:"tolerationKeys,omitempty"`
	LableKeys                []string `json:"lableKeys,omitempty"`
	AnnotationKeys           []string `json:"annotationKeys,omitempty"`
	ResKeys                  []string `json:"resourceKeys,omitempty"`
	ContainerKeys            []string `json:"containerKeys,omitempty"`
	InitContainerKeys        []string `json:"initContainerKeys,omitempty"`
	WithWorkingDirContainers []string `json:"withWorkingDirContainers,omitempty"`
	SchedulerName            string   `json:"schedulerName,omitempty"`
}

// ExclusviceMode
const (
	ExclusiveModeAPP  = "APP"
	ExclusiveModeTAG  = "TAG"
	ExclusiveModeNONE = "NONE"
)

// ReplicaStatus is a pair of workernode status
type ReplicaStatus struct {
	Name                string             `json:"name"`
	CurWorkerNode       *ReplicaWorkerNode `json:"curWorkerNode,omitempty"`
	BackupWorkerNode    *ReplicaWorkerNode `json:"backupWorkerNode,omitempty"`
	Subrs               string             `json:"subrs"`
	CurTargetVersion    string             `json:"curTargetVersion"`
	BackupTargetVersion string             `json:"backupTargetVersion"`
	Detail              Detail             `json:"detail"`
}

type ReplicaWorkerNode struct {
	*WorkerNodeStatus `json:",inline"`
	TargetVersion     string `json:"targetVersion"`
}

type Detail struct {
	Scenes []SceneNode `json:"scenes"`
}

type SceneNode struct {
	Name   string `json:"name"`
	Status int    `json:"status"`
	IP     string `json:"ip"`
}

type Subrs struct {
	Name        string            `json:"name"`
	Replica     int32             `json:"replica"`
	Metas       map[string]string `json:"metas"`
	Annotations map[string]string `json:"annotations"`
	Labels      map[string]string `json:"labels"`
}

type RollingsetPatch struct {
	FiberId string          `json:"fiberId"`
	RoleId  string          `json:"roleId"`
	Patch   json.RawMessage `json:"patch"`
}

type Spread struct {
	Active   SpreadPatch `json:"active"`
	Inactive SpreadPatch `json:"inactive"`
}

type SpreadPatch struct {
	Replica     int               `json:"replica"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// CopyTarget key is prefix (e.g. app.c2.io), val is sourceFullKey (e.g. app.hippo.io/pod-version), result is app.c2.io/pod-version
type CopyTarget map[string][]string

type LabelsCopy struct {
	WorkerLabels      CopyTarget `json:"workerLabels,omitempty"`
	WorkerAnnotations CopyTarget `json:"workerAnnotations,omitempty"`
	PodLabels         CopyTarget `json:"podLabels,omitempty"`
	PodAnnotations    CopyTarget `json:"podAnnotations,omitempty"`
}

type PackageStatus struct {
	PackageDetails []PackageDetail `json:"packageDetails"`
}

type PackageDetail struct {
	Status      string `json:"status"`
	PackageName string `json:"packageName"`
	Extra       Extra  `json:"extra"`
}

type Extra struct {
	Package string `json:"package"`
}

// GangInfo GangInfo
type GangInfo struct {
	Name string `json:"name"`
	IP   string `json:"ip"`
}

type UpdatePhase int

const (
	PhaseUnknown   UpdatePhase = 0
	PhaseReady     UpdatePhase = 1
	PhaseEvicting  UpdatePhase = 2
	PhaseDeploying UpdatePhase = 3
	PhaseStopping  UpdatePhase = 4
	PhaseFailed    UpdatePhase = 5
	PhaseReclaim   UpdatePhase = 6
)

type UpdateStatus struct {
	TargetVersion string       `json:"targetVersion"`
	CurVersion    string       `json:"curVersion"`
	UpdateAt      int64        `json:"updateAt"`
	Phase         UpdatePhase  `json:"phase"`
	HealthStatus  HealthStatus `json:"healthStatus"`
}

type BufferSwapRecord struct {
	Version  string `json:"version"`
	SwapAt   int64  `json:"swapAt"`
	Previous string `json:"previous"`
}

type PoolSpec struct {
	ZoomRatio int `json:"zoomRatio"`
	Addon     int `json:"addon"`
	SubTimeMs int `json:"subTimeMs"`
	Version   string
}

type PoolStatus struct {
	Version string `json:"version"`
	Status  string `json:"status"`
}
