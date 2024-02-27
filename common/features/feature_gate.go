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

package features

import (
	"github.com/spf13/pflag"
	"k8s.io/component-base/featuregate"
)

// default c2 FeatureGates
var (
	C2MutableFeatureGate featuregate.MutableFeatureGate = featuregate.NewFeatureGate()
	C2FeatureGate        featuregate.FeatureGate        = C2MutableFeatureGate
)

// InitFlags is for explicitly initializing the flags.
func InitFlags(flagset *pflag.FlagSet) {
	C2MutableFeatureGate.AddFlag(flagset)
}

// features
const (
	ResetWarmStandbyCommand       = "ResetWarmStandbyCommand"
	UseDeployment                 = "UseDeployment"
	AsiAPIServer                  = "AsiAPIServer"
	AsiConstraint                 = "AsiConstraint"
	ResourceReserve               = "ResourceReserve"
	HashLabelValue                = "HashLabelValue"
	AdminClearPrivateResources    = "AdminClearPrivateResources"
	UsePriorityClassName          = "UsePriorityClassName"
	ShareProcessNamespace         = "ShareProcessNamespace"
	CompletePlanFromAdmin         = "CompletePlanFromAdmin"
	UseOfflineBindAllCPUs         = "UseOfflineBindAllCPUs"
	WarmupNewStartingPod          = "WarmupNewStartingPod"
	WarmupUnavailablePod          = "WarmupUnavailablePod"
	UnPubRestartingNode           = "UnPubRestartingNode"
	GroupCarbonJobByRoleShortName = "GroupCarbonJobByRoleShortName"
	ImmutableWorker               = "ImmutableWorker"
	UpdateResourceRolling         = "UpdateResourceRolling"
	DisablePathAppend             = "DisablePathAppend"
	SimplifyUpdateField           = "SimplifyUpdateField"
	InjectIdentifier              = "InjectIdentifier"
	UpdatePodVersion3WithRelease  = "UpdatePodVersion3WithRelease"
	InjectSystemdToContainer      = "InjectSystemdToContainer"
	DisableLabelInheritance       = "DisableLabelInheritance"
	CloseStandbySortUseOrder      = "CloseStandbySortUseOrder"
	HashSignature                 = "HashSignature"
	SupportDiskRatio              = "SupportDiskRatio"
	V3ForNewCreate                = "V3ForNewCreate"
	LazyV3                        = "LazyV3"
	DeletePodGracefully           = "DeletePodGracefully"
	SyncSubrsMetas                = "SyncSubrsMetas"
	SimplifySlotSet               = "SimplifySlotSet"
	SimplifyShardGroup            = "SimplifyShardGroup"
	ValidateRsVersion             = "ValidateRsVersion"
	HealthCheckWhenUpdatePod      = "HealthCheckWhenUpdatePod"
	OpenRollingSpotPod            = "OpenRollingSpotPod"
	DisableNoMountCgroup          = "DisableNoMountCgroup"
	DisablePodOwnerHippo          = "DisablePodOwnerHippo"
	EnableMissingColumn           = "EnableMissingColumn"
	DisableLaunchSlot             = "DisableLaunchSlot"
	EnableTransContainers         = "EnableTransContainers"
)

// defaultKubernetesFeatureGates consists of all known Kubernetes-specific feature keys.
// To add a new feature, define a key for it above and add it here. The features will be
// available throughout Kubernetes binaries.
var defaultKubernetesFeatureGates = map[featuregate.Feature]featuregate.FeatureSpec{
	ResetWarmStandbyCommand:       {Default: true, PreRelease: featuregate.Beta},
	UseDeployment:                 {Default: false, PreRelease: featuregate.Beta},
	AsiAPIServer:                  {Default: false, PreRelease: featuregate.Beta},
	AsiConstraint:                 {Default: false, PreRelease: featuregate.Beta},
	ResourceReserve:               {Default: false, PreRelease: featuregate.Beta},
	HashLabelValue:                {Default: false, PreRelease: featuregate.Beta},
	AdminClearPrivateResources:    {Default: false, PreRelease: featuregate.Beta},
	UsePriorityClassName:          {Default: false, PreRelease: featuregate.Beta},
	ShareProcessNamespace:         {Default: true, PreRelease: featuregate.Beta},
	CompletePlanFromAdmin:         {Default: false, PreRelease: featuregate.Beta},
	UseOfflineBindAllCPUs:         {Default: false, PreRelease: featuregate.Beta},
	WarmupNewStartingPod:          {Default: false, PreRelease: featuregate.Beta},
	WarmupUnavailablePod:          {Default: false, PreRelease: featuregate.Beta},
	UnPubRestartingNode:           {Default: false, PreRelease: featuregate.Beta},
	GroupCarbonJobByRoleShortName: {Default: false, PreRelease: featuregate.Beta},
	ImmutableWorker:               {Default: false, PreRelease: featuregate.Beta},
	UpdateResourceRolling:         {Default: true, PreRelease: featuregate.Beta},
	DisablePathAppend:             {Default: false, PreRelease: featuregate.Beta},
	SimplifyUpdateField:           {Default: false, PreRelease: featuregate.Beta},
	InjectIdentifier:              {Default: false, PreRelease: featuregate.Beta},
	UpdatePodVersion3WithRelease:  {Default: false, PreRelease: featuregate.Beta},
	InjectSystemdToContainer:      {Default: false, PreRelease: featuregate.Beta},
	DisableLabelInheritance:       {Default: false, PreRelease: featuregate.Beta},
	CloseStandbySortUseOrder:      {Default: false, PreRelease: featuregate.Beta},
	HashSignature:                 {Default: false, PreRelease: featuregate.Beta},
	SupportDiskRatio:              {Default: true, PreRelease: featuregate.Beta},
	V3ForNewCreate:                {Default: false, PreRelease: featuregate.Beta},
	LazyV3:                        {Default: false, PreRelease: featuregate.Beta},
	DeletePodGracefully:           {Default: false, PreRelease: featuregate.Beta},
	SyncSubrsMetas:                {Default: false, PreRelease: featuregate.Beta},
	SimplifySlotSet:               {Default: false, PreRelease: featuregate.Beta},
	ValidateRsVersion:             {Default: true, PreRelease: featuregate.Beta},
	HealthCheckWhenUpdatePod:      {Default: false, PreRelease: featuregate.Beta},
	SimplifyShardGroup:            {Default: false, PreRelease: featuregate.Beta},
	OpenRollingSpotPod:            {Default: false, PreRelease: featuregate.Beta},
	DisableNoMountCgroup:          {Default: false, PreRelease: featuregate.Beta},
	EnableMissingColumn:           {Default: true, PreRelease: featuregate.Beta},
	DisableLaunchSlot:             {Default: false, PreRelease: featuregate.Beta},
	DisablePodOwnerHippo:          {Default: false, PreRelease: featuregate.Beta},
	EnableTransContainers:         {Default: false, PreRelease: featuregate.Beta},
}

func init() {
	C2MutableFeatureGate.Add(defaultKubernetesFeatureGates)
}
