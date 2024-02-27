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

package typespec

import (
	"log"
	"strconv"
	"strings"

	"github.com/alibaba/kube-sharding/pkg/apiserver-client/typespec/carbon"
	"github.com/alibaba/kube-sharding/pkg/apiserver-client/typespec/hippo"
	"github.com/gogo/protobuf/proto"
)

// GroupSchedulerConfig define group schedule config
type GroupSchedulerConfig struct {
	Name      *string `json:"name,omitempty"`
	ConfigStr *string `json:"configStr,omitempty"`
}

// HealthCheckerConfig define health check config
type HealthCheckerConfig struct {
	Name *string           `json:"name,omitempty"`
	Args map[string]string `json:"args,omitempty"`
}

// GetLostTimeout GetLostTimeout
func (h *HealthCheckerConfig) GetLostTimeout() int64 {
	timeout, ok := h.Args[KeyLostTimeout]
	if !ok {
		return 0
	}
	timeoutI, err := strconv.ParseInt(timeout, 10, 0)
	if nil != err {
		log.Printf("parse int error :%s", timeout)
		return 0
	}
	return timeoutI
}

// ServiceAdapterType define service types
type ServiceAdapterType int

// ServiceAdapterTypes
const (
	STNONE       ServiceAdapterType = 0
	STCM2        ServiceAdapterType = 1
	STVIP        ServiceAdapterType = 2
	STLVS        ServiceAdapterType = 3
	STHSF        ServiceAdapterType = 4
	STARMORY     ServiceAdapterType = 5
	STSLB        ServiceAdapterType = 6
	STECSARMORY  ServiceAdapterType = 7
	STVPCSLB     ServiceAdapterType = 8
	STDROGOLVS   ServiceAdapterType = 10 // placeholder for drogo lvs sync
	STSKYLINE    ServiceAdapterType = 11
	STECSSKYLINE ServiceAdapterType = 12
	STANTVIP     ServiceAdapterType = 13
	STLIBRARY    ServiceAdapterType = 0x80
)

// ServiceConfig define service config
type ServiceConfig struct {
	Name      string             `json:"name,omitempty"`
	Type      ServiceAdapterType `json:"type,omitempty"`
	ConfigStr string             `json:"configStr,omitempty"`
	Masked    *bool              `json:"masked,omitempty"`
	MetaStr   *string            `json:"metaStr,omitempty"`
	// no use
	DeleteDelay *int `json:"deleteDelay,omitempty"`
}

// GetMetaStr GetMetaStr
func (s *ServiceConfig) GetMetaStr() string {
	if nil == s || nil == s.MetaStr {
		return ""
	}
	return *s.MetaStr
}

// GetMasked GetMasked
func (s *ServiceConfig) GetMasked() bool {
	if nil == s || nil == s.Masked {
		return false
	}
	return *s.Masked
}

// BrokenRecoverQuotaConfig define BrokenRecoverQuotaConfig
type BrokenRecoverQuotaConfig struct {
	MaxFailedCount *int32 `json:"maxFailedCount,omitempty"`
	TimeWindow     *int32 `json:"timeWindow,omitempty"` // sec
}

// GetMaxFailedCount GetMaxFailedCount
func (b *BrokenRecoverQuotaConfig) GetMaxFailedCount() int32 {
	if nil == b || nil == b.MaxFailedCount {
		return 5
	}
	return *b.MaxFailedCount
}

// GetTimeWindow GetTimeWindow
func (b *BrokenRecoverQuotaConfig) GetTimeWindow() int32 {
	if nil == b || nil == b.TimeWindow {
		return 600
	}
	return *b.TimeWindow
}

// GlobalPlan define group scheduler plan
type GlobalPlan struct {
	Count                    *int32                    `json:"count,omitempty"`
	MinHealthCapacity        *int32                    `json:"minHealthCapacity,omitempty"`
	LatestVersionRatio       *int32                    `json:"latestVersionRatio,omitempty"`
	HealthCheckerConfig      *HealthCheckerConfig      `json:"healthCheckerConfig,omitempty"`
	ServiceConfigs           []ServiceConfig           `json:"serviceConfigs,omitempty"`
	BrokenRecoverQuotaConfig *BrokenRecoverQuotaConfig `json:"brokenRecoverQuotaConfig,omitempty"`
	Properties               *map[string]string        `json:"properties,omitempty"`
}

// carbon health checker keys
const (
	DefaultHealthChecker     = "default"
	Lv7HealthChecker         = "lv7"
	AdvancedLv7HealthChecker = "adv_lv7"
	KeyPort                  = "PORT"
	KeyCheckPath             = "CHECK_PATH"
	KeyLostTimeout           = "LOST_TIMEOUT"
)

// GetLatestVersionRatio get
func (g *GlobalPlan) GetLatestVersionRatio() *int32 {
	if nil == g || nil == g.LatestVersionRatio {
		return func() *int32 { i := int32(100); return &i }()
	}
	return g.LatestVersionRatio
}

// GetMinHealthCapacity get
func (g *GlobalPlan) GetMinHealthCapacity() int32 {
	if nil == g || nil == g.MinHealthCapacity {
		return -1
	}
	return *g.MinHealthCapacity
}

// GetBrokenRecoverQuotaConfig get
func (g *GlobalPlan) GetBrokenRecoverQuotaConfig() *BrokenRecoverQuotaConfig {
	return g.BrokenRecoverQuotaConfig
}

type ResourceRequest_CpusetMode int32

const (
	ResourceRequest_NONE      ResourceRequest_CpusetMode = 0
	ResourceRequest_SHARE     ResourceRequest_CpusetMode = 1
	ResourceRequest_RESERVED  ResourceRequest_CpusetMode = 2
	ResourceRequest_EXCLUSIVE ResourceRequest_CpusetMode = 3
)

var ResourceRequest_CpusetMode_name = map[int32]string{
	0: "NONE",
	1: "SHARE",
	2: "RESERVED",
	3: "EXCLUSIVE",
}

var ResourceRequest_CpusetMode_value = map[string]int32{
	"NONE":      0,
	"SHARE":     1,
	"RESERVED":  2,
	"EXCLUSIVE": 3,
}

func (x ResourceRequest_CpusetMode) Enum() *ResourceRequest_CpusetMode {
	p := new(ResourceRequest_CpusetMode)
	*p = x
	return p
}

func (x ResourceRequest_CpusetMode) String() string {
	return proto.EnumName(ResourceRequest_CpusetMode_name, int32(x))
}

func (x *ResourceRequest_CpusetMode) UnmarshalJSON(data []byte) error {
	value, err := proto.UnmarshalJSONEnum(ResourceRequest_CpusetMode_value, data, "ResourceRequest_CpusetMode")
	if err != nil {
		return err
	}
	*x = ResourceRequest_CpusetMode(value)
	return nil
}

// ResourcePlan define role resource plan
type ResourcePlan struct {
	Resources        []carbon.SlotResource       `json:"resources,omitempty"`
	Declarations     *[]hippo.Resource           `json:"declarations,omitempty"`
	AllocateMode     *string                     `json:"allocateMode,omitempty"`
	Queue            *string                     `json:"queue,omitempty"`
	Priority         *carbon.CarbonPriority      `json:"priority,omitempty"`
	Group            *string                     `json:"group,omitempty"`
	CpusetMode       *ResourceRequest_CpusetMode `json:"cpusetMode,omitempty"`
	Constraints      *ConstraintConfig           `json:"constraints,omitempty"`
	ContainerConfigs []string                    `json:"containerConfigs,omitempty"`
	MetaTags         *map[string]string          `json:"metaTags,omitempty"`
}

// GetMaxInstancePerHost GetMaxInstancePerHost
func (r *ResourcePlan) GetMaxInstancePerHost() int {
	if nil == r.Constraints || nil == r.Constraints.MaxInstancePerHost {
		return 0
	}
	return int(*r.Constraints.MaxInstancePerHost)
}

// GetCpusetMode GetCpusetMode
func (r *ResourcePlan) GetCpusetMode() ResourceRequest_CpusetMode {
	if nil == r || nil == r.CpusetMode {
		return -1
	}
	return *r.CpusetMode
}

// GetAllocateMode GetAllocateMode
func (r *ResourcePlan) GetAllocateMode() string {
	if nil == r || nil == r.AllocateMode {
		return "AUTO"
	}
	return (*r.AllocateMode)
}

// GetPriority GetPriority
func (r *ResourcePlan) GetPriority() *carbon.CarbonPriority {
	if nil == r || nil == r.Priority {
		return nil
	}
	return (r.Priority)
}

// LaunchPlan define role process plan
type LaunchPlan struct {
	PackageInfos []PackageInfo  `json:"packageInfos,omitempty"`
	ProcessInfos []ProcessInfo  `json:"processInfos,omitempty"`
	DataInfos    *[]interface{} `json:"dataInfos,omitempty"`
	PodDesc      string         `json:"podDesc,omitempty"`
}

// VersionedPlan define role resource plan and process plan
type VersionedPlan struct {
	ResourcePlan ResourcePlan `json:"resourcePlan,omitempty"`
	LaunchPlan   LaunchPlan   `json:"launchPlan,omitempty"`
	Signature    *string      `json:"signature,omitempty"`

	//not used when generate version
	UserDefVersion             *string `json:"userDefVersion,omitempty"`
	CustomInfo                 *string `json:"customInfo,omitempty"`
	Online                     *bool   `json:"online,omitempty"`
	NotMatchTimeout            *int64  `json:"notMatchTimeout,omitempty"`
	NotReadyTimeout            *int64  `json:"notReadyTimeout,omitempty"`
	UpdatingGracefully         *bool   `json:"updatingGracefully,omitempty"`
	RestartAfterResourceChange *bool   `json:"restartAfterResourceChange,omitempty"`
	// no use
	Preload     *bool   `json:"preload,omitempty"`
	Cm2TopoInfo *string `json:"cm2TopoInfo,omitempty"`
}

// GetCustomInfo GetCustomInfo
func (v *VersionedPlan) GetCustomInfo() string {
	if nil == v.CustomInfo {
		return ""
	}
	return *v.CustomInfo
}

// GetCm2TopoInfo GetCm2TopoInfo
func (v *VersionedPlan) GetCm2TopoInfo() string {
	if nil == v.Cm2TopoInfo {
		return ""
	}
	return *v.Cm2TopoInfo
}

// GetSignature GetSignature
func (v *VersionedPlan) GetSignature() string {
	if nil == v.Signature {
		return ""
	}
	return *v.Signature
}

// GetUserDefVersion GetUserDefVersion
func (v *VersionedPlan) GetUserDefVersion() string {
	if nil == v.UserDefVersion {
		return ""
	}
	return *v.UserDefVersion
}

// GetNotMatchTimeout GetNotMatchTimeout
func (v *VersionedPlan) GetNotMatchTimeout() int64 {
	if nil == v.NotMatchTimeout {
		return 900
	}
	return *v.NotMatchTimeout
}

// GetNotReadyTimeout GetNotReadyTimeout
func (v *VersionedPlan) GetNotReadyTimeout() int64 {
	if nil == v.NotReadyTimeout {
		return 0
	}
	return *v.NotReadyTimeout
}

// GetOnline GetOnline
func (v *VersionedPlan) GetOnline() bool {
	if nil == v.Online {
		return true
	}
	return *v.Online
}

// GetRestartAfterResourceChange GetRestartAfterResourceChange
func (v *VersionedPlan) GetRestartAfterResourceChange() bool {
	if nil == v.RestartAfterResourceChange {
		return true
	}
	return *v.RestartAfterResourceChange
}

// GetUpdatingGracefully GetUpdatingGracefully
func (v *VersionedPlan) GetUpdatingGracefully() bool {
	if nil == v.UpdatingGracefully {
		return true
	}
	return *v.UpdatingGracefully
}

// GetPreload GetPreload
func (v *VersionedPlan) GetPreload() bool {
	if nil == v.Preload {
		return false
	}
	return *v.Preload
}

// RoleSchedulerConfig define role scheduler plan
type RoleSchedulerConfig struct {
	MinHealthCapacity *int32                `json:"minHealthCapacity,omitempty"`
	ExtraRatio        *int32                `json:"extraRatio,omitempty"`
	SchedulerConfig   *GroupSchedulerConfig `json:"schedulerConfig,omitempty"`
}

// RolePlan define role target
type RolePlan struct {
	RoleID              string               `json:"roleId,omitempty"`
	Global              GlobalPlan           `json:"global,omitempty"`
	RoleSchedulerConfig *RoleSchedulerConfig `json:"roleSchedulerConfig,omitempty"`
	Version             VersionedPlan        `json:"version,omitempty"`
}

// GroupPlan define group target
type GroupPlan struct {
	GroupID           string                `json:"groupId,omitempty"`
	MinHealthCapacity *int32                `json:"minHealthCapacity,omitempty"`
	ExtraRatio        *int32                `json:"extraRatio,omitempty"`
	RolePlans         map[string]*RolePlan  `json:"rolePlans,omitempty"`
	SchedulerConfig   *GroupSchedulerConfig `json:"schedulerConfig,omitempty"`
}

// GetMinHealthCapacity GetMinHealthCapacity
func (g *GroupPlan) GetMinHealthCapacity() int32 {
	if nil == g || nil == g.MinHealthCapacity {
		return 80
	}
	return *g.MinHealthCapacity
}

// GetExtraRatio GetExtraRatio
func (g *GroupPlan) GetExtraRatio() int32 {
	if nil == g || nil == g.ExtraRatio {
		return 10
	}
	return *g.ExtraRatio
}

// ConstraintConfig defines MaxInstancePerHost
type ConstraintConfig struct {
	Level               *int32    `json:"level,omitempty"`
	Strictly            *bool     `json:"strictly,omitempty"`
	UseHostWorkDir      *bool     `json:"useHostWorkDir,omitempty"`
	MaxInstancePerHost  *int32    `json:"maxInstancePerHost,omitempty"`
	MaxInstancePerFrame *int32    `json:"maxInstancePerFrame,omitempty"`
	MaxInstancePerRack  *int32    `json:"maxInstancePerRack,omitempty"`
	MaxInstancePerASW   *int32    `json:"maxInstancePerASW,omitempty"`
	MaxInstancePerPSW   *int32    `json:"maxInstancePerPSW,omitempty"`
	SpecifiedIps        *[]string `json:"specifiedIps,omitempty"`
	ProhibitedIps       *[]string `json:"prohibitedIps,omitempty"`
}

// ProcessInfo define process info
type ProcessInfo struct {
	IsDaemon          *bool       `json:"isDaemon,omitempty"`
	Name              *string     `json:"processName,omitempty"`
	Cmd               *string     `json:"cmd,omitempty"`
	Args              *[][]string `json:"args,omitempty"`
	Envs              *[][]string `json:"envs,omitempty"`
	OtherInfos        *[][]string `json:"otherInfos,omitempty"`
	InstanceID        *int64      `json:"instanceId,omitempty"`
	StopTimeout       *int        `json:"stopTimeout,omitempty"`
	RestartInterval   *int64      `json:"restartInterval,omitempty"`
	RestartCountLimit *int        `json:"restartCountLimit,omitempty"`
	ProcStopSig       *int32      `json:"procStopSig,omitempty"`
}

// GetIsDaemon GetIsDaemon
func (p *ProcessInfo) GetIsDaemon() bool {
	if nil == p.IsDaemon {
		return false
	}
	return *p.IsDaemon
}

// GetRestartCountLimit GetRestartCountLimit
func (p *ProcessInfo) GetRestartCountLimit() int {
	if nil == p.RestartCountLimit {
		return 0
	}
	return *p.RestartCountLimit
}

// GetStopTimeout GetStopTimeout
func (p *ProcessInfo) GetStopTimeout() int {
	if nil == p.StopTimeout {
		return 0
	}
	return *p.StopTimeout
}

// GetName GetName
func (p *ProcessInfo) GetName() string {
	if nil == p.Name {
		return ""
	}
	return *p.Name
}

// GetCmd GetCmd
func (p *ProcessInfo) GetCmd() []string {
	if nil == p.Cmd {
		return nil
	}
	cmds := strings.Fields(*p.Cmd)
	return cmds
}

// GetArgs GetArgs
func (p *ProcessInfo) GetArgs() []string {
	if nil == p.Args {
		return nil
	}
	var args = make([]string, 0, len(*p.Args)*2)
	for i := range *p.Args {
		for j := range (*p.Args)[i] {
			args = append(args, (*p.Args)[i][j])
		}
	}
	return args
}

// PackageInfo define package infos
type PackageInfo struct {
	URI  string `json:"packageURI,omitempty"`
	Type string `json:"type,omitempty"`
}
