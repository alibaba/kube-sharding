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

package transfer

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/alibaba/kube-sharding/pkg/apiserver-client/typespec/carbon"
	"github.com/alibaba/kube-sharding/pkg/apiserver-client/typespec/hippo"
	k8sresource "k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/integer"

	"github.com/alibaba/kube-sharding/pkg/apiserver-client/typespec"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/alibaba/kube-sharding/common/features"
	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	"github.com/alibaba/kube-sharding/pkg/rollalgorithm"
	"github.com/spf13/pflag"
	app "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/validation"
	glog "k8s.io/klog"
)

// c2 keys
const (
	C2PlatformKey         = "app.c2.io/platform"
	platformKey           = "app.c2.io/plaform"
	carbonUserProcessName = "CARBON_USER_PROCESS_NAME"
	defaultPartition      = "default"

	containerConfKeyWorkingDir       = "WORKING_DIRS"
	containerConfKeyVolumeMounts     = "VOLUME_MOUNTS"
	kataVolumeMounts                 = "KATA_VOLUME_MOUNTS"
	containerConfKeyTimezone         = "TIMEZONE"
	containerConfKeyBindMounts       = "BIND_MOUNTS"
	containerConfKeyDownwardAPI      = "DOWNWARD_API"
	containerConfKeyLabels           = "LABELS"
	containerConfKeyEnvs             = "ENVS"
	containerConfKeyUlimits          = "ULIMITS"
	containerConfKeySysctls          = "SYSCTLS"
	containerConfKeyCapability       = "CAPADDS"
	containerConfKeyCapabilityDrop   = "CAPDROPS"
	containerConfKeyDevice           = "DEVICES"
	containerConfKeyPrivileged       = "PRIVILEGED"
	containerConfDependency          = "DEPENDENCY_LEVEL"
	containerConfKeyDNSPolicy        = "DNS_POLICY"
	containerConfKeyVolumeMountsJSON = "JSON_VOLUME_MOUNTS"
	containerConfKeyVolumes          = "JSON_VOLUMES"
	containerConfKeyMemPolicy        = "MEM_POLICY"

	highQualityLabelValue = "HighQuality"
	qualityLabelKey       = "alibabacloud.com/resource-level"

	propUseSmoothRecoverKey      = "smooth_recover" // DEPRECATED (compatible to tf admin)
	propRecoverKey               = "recover_strategy"
	propIsMasterFrameworkModeKey = "master_framework"

	// c2 env key
	envKeyC2Group = "C2_GROUP"
	envKeyC2Role  = "C2_ROLE"

	SchTypeRole  = "role"
	SchTypeGroup = "group"
	SchTypeNode  = "node"
)

var (
	platform    string
	checkDupEnv = true
)

// InitFlags is for explicitly initializing the flags.
func InitFlags(flagset *pflag.FlagSet) {
	flagset.StringVar(&platform, "platform", platform, "Business platform, e.g. d2, igraph, rtp.")
	flagset.BoolVar(&checkDupEnv, "check-dup-env", checkDupEnv, "if check duplicate envkey")
}

// TransSpecResult is the result of trans carbon plan
type TransSpecResult struct {
	Rollingset *carbonv1.RollingSet
	Services   []*carbonv1.ServicePublisher
	ZoneName   string
}

// TransGroupToRollingsets trans carbon plan to rollingsets
func TransGroupToRollingsets(cluster, app, rollingSetName string, groupPlan *typespec.GroupPlan, newCreated bool) ([]*TransSpecResult, error) {
	if nil == groupPlan.RolePlans {
		err := fmt.Errorf("nil role plans %s", groupPlan.GroupID)
		glog.Error(err)
		return nil, err
	}
	shardgroupName := SubGroupName(groupPlan.GroupID)
	var results = make([]*TransSpecResult, 0, len(groupPlan.RolePlans))
	for k, v := range groupPlan.RolePlans {
		result, err := TransRoleToRollingsets(cluster, app, "", groupPlan.GroupID, k, v, SchTypeRole, true, shardgroupName, rollingSetName, newCreated)
		if nil != err {
			glog.Errorf("TransRoleToRollingsets :%s", groupPlan.GroupID)
			return nil, err
		}
		coverGroupField(result, groupPlan)
		result.Rollingset.Labels[carbonv1.ControllersShardKey] = computeControllerShardKey(result.Rollingset.Name)
		result.Rollingset.Labels[carbonv1.GroupTypeKey] = carbonv1.GroupTypeRollingset
		appendPlatform(result.Rollingset.Labels)
		results = append(results, result)
	}
	return results, nil
}

// TransGroupToShardGroup trans carbon plan to shardGroup
func TransGroupToShardGroup(cluster, app string, groupPlan *typespec.GroupPlan, shardgroupName string, newCreated bool) (*carbonv1.ShardGroup, []*carbonv1.ServicePublisher, error) {
	if nil == groupPlan.RolePlans {
		err := fmt.Errorf("nil role plans %s", groupPlan.GroupID)
		glog.Error(err)
		return nil, nil, err
	}
	var shardGroup carbonv1.ShardGroup
	var rollingSet *carbonv1.RollingSet
	var publishers = []*carbonv1.ServicePublisher{}
	var zoneName string
	var isSingleZone = true
	var zonePartCount = map[string]int{}
	shardGroup.Spec.ShardTemplates = map[string]carbonv1.ShardTemplate{}
	for k, v := range groupPlan.RolePlans {
		result, err := TransRoleToRollingsets(cluster, app, "", groupPlan.GroupID, k, v, SchTypeGroup, false, shardgroupName, "", newCreated)
		if nil != err {
			glog.Errorf("TransRoleToRollingsets :%s", groupPlan.GroupID)
			return nil, nil, err
		}

		var template carbonv1.ShardTemplate
		coverGroupField(result, groupPlan)
		coverSingleRoleMaxUnavailable(result, v)
		utils.JSONDeepCopy(result.Rollingset, &template)
		utils.JSONDeepCopy(&result.Rollingset.Spec, &template.Spec)
		shardGroup.Spec.ShardTemplates[result.Rollingset.Name] = template
		publishers = append(publishers, result.Services...)
		rollingSet = result.Rollingset
		if zoneName != result.ZoneName && zoneName != "" {
			isSingleZone = false
		}
		partCount := zonePartCount[result.ZoneName]
		zonePartCount[result.ZoneName] = partCount + 1
		if zoneName == "" {
			zoneName = result.ZoneName
		}
	}
	if !isSingleZone {
		zoneName = ""
	}
	completeGroupField(&shardGroup, groupPlan, zoneName, isSingleZone, rollingSet)
	completeGroupBizDetail(&shardGroup, zonePartCount)
	shardGroup.Name = shardgroupName
	shardGroup.Labels[carbonv1.ControllersShardKey] = computeControllerShardKey(shardGroup.Name)
	carbonv1.CompressShardTemplates(&shardGroup)
	return &shardGroup, publishers, nil
}

// TransGroupToGangRollingset trans carbon plan to gang rollingset
func TransGroupToGangRollingset(cluster, app string, groupPlan *typespec.GroupPlan, shardgroupName string) (*carbonv1.RollingSet, []*carbonv1.ServicePublisher, error) {
	if nil == groupPlan.RolePlans {
		err := fmt.Errorf("nil role plans %s", groupPlan.GroupID)
		glog.Error(err)
		return nil, nil, err
	}
	var rollingSet = carbonv1.RollingSet{}
	rollingSet.Spec.GangVersionPlan = map[string]*carbonv1.VersionPlan{}
	var publishers = []*carbonv1.ServicePublisher{}
	var mainPart string
	for k := range groupPlan.RolePlans {
		if mainPart == "" || mainPart < k {
			mainPart = k
		}
	}
	for k, v := range groupPlan.RolePlans {
		result, err := TransRoleToRollingsets(cluster, app, "", groupPlan.GroupID, k, v, SchTypeGroup, false, shardgroupName, "", true)
		if nil != err {
			glog.Errorf("TransRoleToRollingsets :%s", groupPlan.GroupID)
			return nil, nil, err
		}
		coverGroupField(result, groupPlan)
		result.Rollingset.Labels[carbonv1.ControllersShardKey] = computeControllerShardKey(result.Rollingset.Name)
		result.Rollingset.Labels[carbonv1.GroupTypeKey] = carbonv1.GroupTypeGangRollingset
		appendPlatform(result.Rollingset.Labels)
		coverGroupField(result, groupPlan)
		shardName := EscapeName(k)
		if mainPart == k {
			rollingSet.ObjectMeta = result.Rollingset.ObjectMeta
			gangVersionPlan := rollingSet.Spec.GangVersionPlan
			rollingSet.Spec = result.Rollingset.Spec
			rollingSet.Spec.GangVersionPlan = gangVersionPlan
		}
		delete(rollingSet.Annotations, carbonv1.LabelKeyRoleName)
		delete(rollingSet.Labels, carbonv1.LabelKeyRoleName)
		delete(rollingSet.Labels, carbonv1.LabelKeyRoleNameHash)
		if mainPart == k {
			carbonv1.SetGangMainPart(result.Rollingset.Spec.VersionPlan.Template)
		}
		rollingSet.Spec.GangVersionPlan[shardName] = &result.Rollingset.Spec.VersionPlan
		rollingSet.Name = shardgroupName
		carbonv1.AddGroupUniqueLabelHashKey(rollingSet.Labels, shardgroupName)
		carbonv1.AddRsUniqueLabelHashKey(rollingSet.Labels, shardgroupName)
		for i := range result.Services {
			service := result.Services[i]
			carbonv1.AddRsUniqueLabelHashKey(service.Labels, shardgroupName)
			carbonv1.AddGangPartLabelHashKey(service.Spec.Selector, shardName)
			carbonv1.AddRsUniqueLabelHashKey(service.Spec.Selector, shardgroupName)
			publishers = append(publishers, service)
		}
	}
	return &rollingSet, publishers, nil
}
func subRollingsetName(rollingsetName, groupName string) string {
	subString := utils.GetMaxLengthCommonString(rollingsetName, groupName)
	if strings.HasPrefix(rollingsetName, string(subString)) {
		realName := strings.TrimPrefix(rollingsetName, subString)
		if realName == "" {
			realName = defaultPartition
		}
		if strings.HasPrefix(realName, "_") {
			realName = strings.TrimPrefix(realName, "_")
		}
		if strings.HasPrefix(realName, "-") {
			realName = strings.TrimPrefix(realName, "-")
		}
		return realName
	}
	return rollingsetName
}

func createWorkerNodeName(appName, groupID, rollingsetName string) (string, error) {
	if groupID == appName {
		groupID = ""
	}
	if groupID == rollingsetName {
		groupID = ""
	}
	var err error
	// 最终生成的podname不能过长。不同版本长度限制不一样，目前按照1.14版本，
	if len(appName) > carbonv1.MaxAppNameLength {
		sign, err := utils.SignatureWithMD5(appName)
		if nil != err {
			return "", err
		}
		appName = sign[0:7]
	}
	roleName := appName
	if groupID != "" {
		groupID, err = signName(groupID, carbonv1.MaxRoleNameLength)
		if nil != err {
			return "", err
		}
		roleName = utils.AppendString(appName, ".", groupID)
	}
	rollingsetName, err = signName(rollingsetName, carbonv1.MaxRoleNameLength)
	if nil != err {
		return "", err
	}
	roleName = utils.AppendString(roleName, ".", rollingsetName)
	roleName, err = signName(roleName, carbonv1.MaxNameLength)
	if nil != err {
		return "", err
	}
	roleName = EscapeName(roleName)
	if validation.IsDNS1123Subdomain(roleName) != nil {
		roleName, _ = utils.SignatureWithMD5(roleName)
	}
	return roleName, nil
}

func signName(name string, length int) (string, error) {
	if len(name) > length {
		sign, err := utils.SignatureWithMD5(name)
		if nil != err {
			return "", err
		}
		name = sign
	}
	return name, nil
}

// SubGroupName SubGroupName
func SubGroupName(groupName string) string {
	subString := utils.GetMaxLengthSubRepeatString(groupName)
	if len(subString) > 3 && strings.HasPrefix(groupName, string(subString)) {
		realName := strings.TrimPrefix(groupName, subString)
		realName = trimFixedDecorator(realName)
		return EscapeName(realName)
	}
	return EscapeName(groupName)
}

func trimFixedDecorator(name string) string {
	name = strings.TrimPrefix(name, "_")
	name = strings.TrimPrefix(name, "-")
	name = strings.Replace(name, "_suez_ops_et2_7u", "", -1)
	name = strings.Replace(name, "_suez_ops_na61_7u", "", -1)
	name = strings.Replace(name, "_suez_ops_ea120_7u", "", -1)
	name = strings.Replace(name, "_suez_ops_st3_7u", "", -1)
	return name
}

func completeGroupField(shardGroup *carbonv1.ShardGroup, groupPlan *typespec.GroupPlan, zoneName string, isSingleZone bool, rollingSet *carbonv1.RollingSet) {
	shardGroup.Spec.MaxUnavailable = transferMaxUnavailable(groupPlan.GetMinHealthCapacity())
	maxSurge := intstr.FromString(fmt.Sprintf("%d%%", int(groupPlan.GetExtraRatio())))
	shardGroup.Spec.MaxSurge = &maxSurge
	shardGroup.Labels = map[string]string{
		carbonv1.GroupTypeKey: carbonv1.GroupTypeShardGroup,
	}
	setObjectGroupKeys(shardGroup,
		carbonv1.GetClusterName(rollingSet),
		carbonv1.GetAppName(rollingSet),
		"",
		groupPlan.GroupID,
		zoneName,
	)
	appendPlatform(shardGroup.Labels)
	if nil != groupPlan.SchedulerConfig && nil != groupPlan.SchedulerConfig.ConfigStr && nil != groupPlan.SchedulerConfig.Name {
		if rollalgorithm.SchedulerTypeRolling == *groupPlan.SchedulerConfig.Name {
			shardGroup.Spec.RollingStrategy = *groupPlan.SchedulerConfig.ConfigStr
		}
	}
	if !isSingleZone {
		shardGroup.Spec.RollingStrategy = rollalgorithm.RollingStrategyLessFirst
	}
	shardGroup.Spec.WorkerSchedulePlan = rollingSet.Spec.WorkerSchedulePlan
	shardGroup.Spec.LatestPercent = rollingSet.Spec.LatestVersionRatio
	shardGroup.Spec.RestartAfterResourceChange = rollingSet.Spec.RestartAfterResourceChange
	shardGroup.Spec.ShardDeployStrategy = carbonv1.RollingSetType
}

func completeGroupBizDetail(shardGroup *carbonv1.ShardGroup, zonePartCounts map[string]int) {
	for k := range shardGroup.Spec.ShardTemplates {
		template := shardGroup.Spec.ShardTemplates[k]
		zoneName := carbonv1.GetZoneName(&template)
		zonePartCount := zonePartCounts[zoneName]
		if zonePartCount == 0 {
			zonePartCount = 1
		}
		template.Spec.Template.Annotations[carbonv1.BizDetailKeyPartitionCount] = strconv.Itoa(zonePartCount)
		shardGroup.Spec.ShardTemplates[k] = template
	}
}

func coverGroupField(result *TransSpecResult, groupPlan *typespec.GroupPlan) {
	var rollingargs app.RollingUpdateDeployment
	rollingargs.MaxUnavailable = transferMaxUnavailable(groupPlan.GetMinHealthCapacity())
	maxSurge := intstr.FromString(fmt.Sprintf("%d%%", int(groupPlan.GetExtraRatio())))
	rollingargs.MaxSurge = &maxSurge
	var strategy = app.DeploymentStrategy{
		Type:          app.RollingUpdateDeploymentStrategyType,
		RollingUpdate: &rollingargs,
	}
	result.Rollingset.Spec.Strategy = strategy
}

func coverSingleRoleMaxUnavailable(result *TransSpecResult, rolePlan *typespec.RolePlan) {
	if -1 != rolePlan.Global.GetMinHealthCapacity() {
		result.Rollingset.Spec.Strategy.RollingUpdate.MaxUnavailable = transferMaxUnavailable(rolePlan.Global.GetMinHealthCapacity())
	}
}

var _reg *regexp.Regexp = regexp.MustCompile(`[^a-z0-9-.]`)

// EscapeName for kube-apiserver
func EscapeName(name string) string {
	name = strings.ToLower(name)
	result := _reg.ReplaceAllString(name, "-")
	if strings.HasPrefix(result, "-") {
		result = "p" + result
	}
	if strings.HasSuffix(result, "-") || strings.HasSuffix(result, ".") {
		result = result + "s"
	}
	return result
}

var namespaceReg *regexp.Regexp = regexp.MustCompile(`[^a-z0-9-]`)

// EscapeNamespace for kube-apiserver
func EscapeNamespace(namespace string) string {
	namespace = strings.ToLower(namespace)
	result := namespaceReg.ReplaceAllString(namespace, "-")
	if strings.HasPrefix(result, "-") {
		result = "p" + result
	}
	if strings.HasSuffix(result, "-") {
		result = result + "s"
	}
	return result
}

// EscapeNamespace2 for kube-apiserver
func EscapeNamespace2(namespace string) string {
	namespace = strings.ToLower(namespace)
	result := namespaceReg.ReplaceAllString(namespace, "-")
	for strings.HasPrefix(result, "-") {
		result = strings.TrimPrefix(result, "-")
	}
	for strings.HasSuffix(result, "-") {
		result = strings.TrimSuffix(result, "-")
	}
	return result
}

// GenerateRollingsetName GenerateRollingsetName
func GenerateRollingsetName(appName, groupID, roleID string, schType string) (string, error) {
	name := roleID
	var err error
	if schType == SchTypeRole {
		name = groupID
	} else if schType == SchTypeGroup {
		name = subRollingsetName(roleID, groupID)
	} else if schType == SchTypeNode {
		name, err = createWorkerNodeName(appName, groupID, roleID)
		if nil != err {
			return "", err
		}
	} else {
		err := fmt.Errorf("un supported group type :%s", SchTypeNode)
		return "", err
	}
	name = EscapeName(name)
	if name == "" || len(name) > (carbonv1.MaxNameLength+3) {
		err = fmt.Errorf("rollingset name is too long :%s", name)
		glog.Error(err)
		return "", err
	}
	return name, nil
}

// GenerateHippoRoleName GenerateHippoRoleName
func GenerateHippoRoleName(groupID, roleID string) string {
	tag := utils.AppendString(groupID, ".", roleID)
	return tag
}

// GetGroupAndRoleID GetGroupAndRoleID
func GetGroupAndRoleID(object metav1.Object) (string, string) {
	roleName := carbonv1.GetRoleName(object)
	groupID := carbonv1.GetGroupName(object)
	if "" == groupID {
		groupID = carbonv1.GetShardGroupID(object)
	}
	roleID := strings.TrimPrefix(roleName, groupID+".")
	return groupID, roleID
}

func getRecoverStrategy(globalPlan typespec.GlobalPlan) carbonv1.RecoverStrategy {
	if nil == globalPlan.Properties {
		return carbonv1.DefaultRecoverStrategy
	}
	validValues := map[string]bool{carbonv1.DefaultRecoverStrategy: true,
		carbonv1.DirectReleasedRecoverStrategy: true, carbonv1.NotRecoverStrategy: true}
	props := *globalPlan.Properties
	if st, ok := props[propRecoverKey]; ok && validValues[st] {
		return carbonv1.RecoverStrategy(st)
	}
	if !getPropKeyBool(*(globalPlan.Properties), propUseSmoothRecoverKey, true) {
		return carbonv1.DirectReleasedRecoverStrategy
	}
	return carbonv1.DefaultRecoverStrategy
}

func getPropKeyBool(properties map[string]string, key string, def bool) bool {
	if value, ok := (properties)[key]; ok {
		var err error
		var bValue bool
		if bValue, err = strconv.ParseBool(value); nil == err {
			return bValue
		}
		glog.Errorf("parse prop bool key [%s] failed, value: %s, err: %s", key, value, err)
		return def
	}
	return def
}

type scheduleParams struct {
	exclusives               []string
	schedulerName            string
	zoneName                 string
	delayDeleteBackupSeconds int64
	warmUpSeconds            int64
	dependencyLevel          int
}

// TransRoleToRollingsets trans role plan to rollingsets
func TransRoleToRollingsets(cluster, app, appChecksum, groupID, roleID string, role *typespec.RolePlan,
	schType string, single bool, shardgroupName, rollingSetName string, newCreated bool) (*TransSpecResult, error) {
	hippoPodTemplate, scheduleParams, err := CreateTemplate(app, groupID, roleID, role, schType)
	if nil != err {
		glog.Errorf("create hippoPodTemplate error :%s,%v", roleID, err)
		return nil, err
	}
	// 某些plan里没有roleID，使用map key补齐
	if role.RoleID == "" {
		role.RoleID = roleID
	}
	workdir := GenerateHippoRoleName(groupID, roleID)
	var name = rollingSetName
	if name == "" {
		name, err = GenerateRollingsetName(app, groupID, roleID, schType)
		if nil != err {
			glog.Error(err)
			return nil, err
		}
	}
	compressedCustomInfo := utils.CompressStringToString(role.Version.GetCustomInfo())
	rollingset := generateRollingset(name, compressedCustomInfo, role.Version.GetCm2TopoInfo(), role, scheduleParams.delayDeleteBackupSeconds, scheduleParams.warmUpSeconds, scheduleParams.dependencyLevel, schType)
	if isSysRole(role) {
		daemonSetPercent := intstr.FromString("100%")
		rollingset.Spec.DaemonSetReplicas = &daemonSetPercent
	}
	setObjectRoleKeys(&rollingset, cluster, app, appChecksum, groupID, scheduleParams.zoneName, workdir)
	if single {
		carbonv1.AddGroupUniqueLabelHashKey(rollingset.Labels, rollingset.Name)
		shardgroupName = rollingset.Name
	}
	fillinRollingsetLabels(&rollingset, role, scheduleParams.exclusives, !hippoPodTemplate.Spec.HostNetwork, newCreated)
	err = carbonv1.PatchObject("RollingSet", &rollingset)
	if nil != err {
		glog.Errorf("PatchObject error :%s,%v", roleID, err)
		return nil, err
	}
	isAsiPod := carbonv1.IsAsiPod(&rollingset.ObjectMeta)
	services, err := transServices(app, groupID, name, role, single, shardgroupName, isAsiPod || carbonv1.IsLazyV3(&rollingset))
	if nil != err {
		glog.Errorf("transServices error :%s,%v", roleID, err)
		return nil, err
	}
	err = checkRollingsetValidation(&rollingset, hippoPodTemplate)
	if nil != err {
		return nil, err
	}
	rollingset.Spec.Template = hippoPodTemplate
	var result = TransSpecResult{
		Rollingset: &rollingset,
		Services:   services,
		ZoneName:   scheduleParams.zoneName,
	}
	return &result, nil
}

func generateRollingset(name, compressedCustomInfo, cm2TopoInfo string, role *typespec.RolePlan, delayDeleteBackupSeconds, warmUpSeconds int64, dependencyLevel int, schType string) carbonv1.RollingSet {
	var rollingset = carbonv1.RollingSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{},
		},
		Spec: carbonv1.RollingSetSpec{
			SchedulePlan: rollalgorithm.SchedulePlan{
				Replicas:           role.Global.Count,
				LatestVersionRatio: role.Global.GetLatestVersionRatio(),
			},
			VersionPlan: carbonv1.VersionPlan{
				BroadcastPlan: carbonv1.BroadcastPlan{
					CompressedCustomInfo: compressedCustomInfo,
					UserDefVersion:       role.Version.GetUserDefVersion(),
					Online:               utils.BoolPtr(role.Version.GetOnline()),
					UpdatingGracefully:   utils.BoolPtr(role.Version.GetUpdatingGracefully()),
					Preload:              role.Version.GetPreload(),
					WorkerSchedulePlan: carbonv1.WorkerSchedulePlan{
						ProcessMatchTimeout:      role.Version.GetNotMatchTimeout(),
						WorkerReadyTimeout:       role.Version.GetNotReadyTimeout(),
						DelayDeleteBackupSeconds: delayDeleteBackupSeconds,
						WarmupSeconds:            int32(warmUpSeconds),
						BrokenRecoverQuotaConfig: GetBrokenRecoverQuotaConfig(&role.Global),
						RecoverStrategy:          getRecoverStrategy(role.Global),
					},
				},
				SignedVersionPlan: carbonv1.SignedVersionPlan{
					Signature:                  hashSignature(role.Version.GetSignature(), schType),
					RestartAfterResourceChange: utils.BoolPtr(role.Version.GetRestartAfterResourceChange()),
					Cm2TopoInfo:                cm2TopoInfo,
				},
			},
			HealthCheckerConfig: GetHealthCheckerConfig(&role.Global),
		},
	}
	if 0 != dependencyLevel {
		rollingset.Spec.DependencyLevel = utils.Int32Ptr(int32(dependencyLevel))
	}
	return rollingset
}

func hashSignature(signature string, schType string) string {
	if features.C2MutableFeatureGate.Enabled(features.HashSignature) && schType == SchTypeGroup {
		hash, _ := utils.SignatureWithMD5(signature)
		return hash
	}
	return signature
}

func fillinRollingsetLabels(rollingset *carbonv1.RollingSet, role *typespec.RolePlan, exclusives []string, vip bool, newCreated bool) {
	maxInstancePerNode := role.Version.ResourcePlan.GetMaxInstancePerHost()
	if 0 != maxInstancePerNode {
		rollingset.Labels[carbonv1.LabelKeyMaxInstancePerNode] = strconv.Itoa(maxInstancePerNode)
	}
	if nil != role.Version.ResourcePlan.Group && "" != *role.Version.ResourcePlan.Group {
		rollingset.Labels[carbonv1.LabelKeyQuotaGroupID] = *role.Version.ResourcePlan.Group
	}
	if nil != exclusives && 0 != len(exclusives) {
		carbonv1.SetExclusiveLableAnno(rollingset, strings.Join(exclusives, "."))
	}
	rollingset.Labels["app.hippo.io/appname-strategy"] = "without-namespace"
	if nil != role.Version.ResourcePlan.MetaTags {
		podVersion := (*role.Version.ResourcePlan.MetaTags)[carbonv1.LabelKeyPodVersion]
		if "" != podVersion {
			rollingset.ObjectMeta.Labels[carbonv1.LabelKeyPodVersion] = podVersion
		}
	}
	// 用户未明确指定的，可以lazy模式升级v3
	if !newCreated && carbonv1.GetLabelValue(rollingset, carbonv1.LabelHippoPodVersion) == "" && features.C2FeatureGate.Enabled(features.LazyV3) {
		carbonv1.SetLazyV3(rollingset, true)
	}
	if features.C2FeatureGate.Enabled(features.V3ForNewCreate) && newCreated {
		carbonv1.SetLabelsDefaultValue(carbonv1.LabelKeyPodVersion, carbonv1.PodVersion3, &rollingset.ObjectMeta)
	}
	if vip {
		carbonv1.SetDefaultVipPodVersion(&rollingset.ObjectMeta)
	} else {
		carbonv1.SetDefaultPodVersion(&rollingset.ObjectMeta)
	}
	declares := transDeclare(role)
	if nil != declares && 0 != len(declares) {
		// TODO declare
	}
}

// CreateTemplate creates pod template
func CreateTemplate(appName, groupID, roleID string, role *typespec.RolePlan, schType string) (*carbonv1.HippoPodTemplate, *scheduleParams, error) {
	ress, affinity, tolerations, resourceLabels, exclusives, needip, processMode, hippoExtend, err := transResources(role.Version.ResourcePlan.Resources)
	if nil != err {
		glog.Error(err)
		return nil, nil, err
	}
	cconf, err := transContainerConfigs(appName, fmt.Sprintf("%s.%s", groupID, roleID), role.Version.ResourcePlan.ContainerConfigs)
	if nil != err {
		glog.Error(err)
		return nil, nil, err
	}
	cconf.containerEnvs = appendC2ENVs(groupID, roleID, cconf.containerEnvs)
	if needip && cconf.containerModel == "" {
		cconf.containerModel = "VM"
	}
	var pkgs []carbonv1.PackageInfo
	var image string
	if role.Version.LaunchPlan.PodDesc == "" {
		pkgs, image = transPackages(role.RoleID, role.Version.LaunchPlan.PackageInfos)
		if nil != err {
			glog.Error(err)
			return nil, nil, err
		}
	}
	containers, extVolumes, restartPolicy, hostIPC, err := transContainers(
		role, ress, image, cconf.containerEnvs, cconf.volumeMounts, cconf.containerLabels, cconf.config, cconf.workingDirs, cconf.securityContext, cconf.devices)
	if nil != err {
		glog.Error(err)
		return nil, nil, err
	}
	if len(containers) == 0 {
		err = fmt.Errorf("nil containers")
		glog.Errorf("%s,%v", role.RoleID, err)
		return nil, nil, err
	}
	cpuSetMode := role.Version.ResourcePlan.GetCpusetMode().String()
	if cpuSetMode == "-1" {
		cpuSetMode = ""
	}
	priority, err := transPriority(role)
	if nil != err {
		return nil, nil, err
	}
	shareProcessNamespace := features.C2MutableFeatureGate.Enabled(features.ShareProcessNamespace)
	if cconf.shareProcessNamespace != nil {
		shareProcessNamespace = *cconf.shareProcessNamespace
	}
	var hippoPodSpec = carbonv1.HippoPodSpec{
		HippoPodSpecExtendFields: carbonv1.HippoPodSpecExtendFields{
			CpusetMode:           cpuSetMode,
			ContainerModel:       cconf.containerModel,
			PackageInfos:         pkgs,
			RestartWithoutRemove: cconf.restartWithoutRemove,
			NeedHippoMounts:      cconf.needHippoMounts,
		},
		Containers:   containers,
		HippoVolumes: cconf.hippoVolumes,
		PodSpec: corev1.PodSpec{
			Tolerations:                   tolerations,
			Affinity:                      affinity,
			RestartPolicy:                 restartPolicy,
			Priority:                      priority,
			Volumes:                       append(cconf.volumes, extVolumes...),
			ShareProcessNamespace:         &shareProcessNamespace,
			TerminationGracePeriodSeconds: func() *int64 { t := getTerminationGracePeriodSeconds(containers); return &t }(),
			SchedulerName:                 cconf.schedulerName,
			SecurityContext:               cconf.podSecurityContext,
			ServiceAccountName:            cconf.serviceAccount,
			HostIPC:                       hostIPC,
		},
	}
	if cconf.dnsPolicy != "" {
		hippoPodSpec.DNSPolicy = corev1.DNSPolicy(cconf.dnsPolicy)
	}
	carbonv1.FixSchedulerName(&hippoPodSpec)
	if !needip {
		hippoPodSpec.HostNetwork = true
	}
	if cconf.cpuShareNum != 0 {
		cpuShareNum32 := int32(cconf.cpuShareNum)
		hippoPodSpec.CPUShareNum = &cpuShareNum32
	}
	hippoPodSpec.PredictDelayTime = cconf.predictDelayTime

	var hippoPodTemplate = carbonv1.HippoPodTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{},
			Labels:      map[string]string{},
		},
		Spec: hippoPodSpec,
	}
	carbonv1.SetExclusiveLableAnno(&hippoPodTemplate, strings.Join(exclusives, "."))
	roleName := roleID
	// Don't join groupId and roleId only if in master framework mode (also SchTypeNode)
	if !(schType == SchTypeNode && IsMasterFrameworkMode(role)) {
		roleName = utils.AppendString(groupID, ".", roleID)
	}
	carbonv1.SetObjectAppRoleName(&hippoPodTemplate, appName, roleName)
	if cconf.pouchAnnotations != "" {
		hippoPodTemplate.ObjectMeta.Annotations["app.hippo.io/pouch-annotations"] = cconf.pouchAnnotations
	}
	if hippoExtend != nil && (len(hippoExtend.Resources) != 0 || len(hippoExtend.DelayORResources) != 0) {
		hippoExtendStr := utils.ObjJSON(hippoExtend)
		hippoPodTemplate.ObjectMeta.Annotations["app.hippo.io/hippo-extend"] = hippoExtendStr
	}

	setInjectIdentifier(&hippoPodTemplate)
	fillTemplateByMetaTags(&hippoPodTemplate, role.Version.ResourcePlan.MetaTags, resourceLabels)
	carbonv1.FixUnifiedStorageVolumeMounts(&hippoPodTemplate)
	carbonv1.FixRolePvcVolumeMounts(&hippoPodTemplate)
	zoneName := fillTemplateLabelByEnvs(&hippoPodTemplate, containers)
	carbonv1.InjectMetas(&hippoPodTemplate.ObjectMeta)
	injectSystemdToContainer(&hippoPodTemplate)
	if processMode {
		hippoPodTemplate.Annotations[carbonv1.AnnotationC2ContainerRuntime] = "process"
	} else if cconf.containerModel == "VM" {
		hippoPodTemplate.Annotations[carbonv1.AnnotationC2ContainerRuntime] = "vm"
	}
	var params = scheduleParams{
		exclusives:               exclusives,
		schedulerName:            cconf.schedulerName,
		zoneName:                 zoneName,
		delayDeleteBackupSeconds: cconf.delayDeleteBackup,
		warmUpSeconds:            cconf.warmUpSeconds,
		dependencyLevel:          cconf.dependencyLevel,
	}
	return &hippoPodTemplate, &params, nil
}

func injectSystemdToContainer(hippoPodTemplate *carbonv1.HippoPodTemplate) {
	if !features.C2MutableFeatureGate.Enabled(features.InjectSystemdToContainer) {
		return
	}
	if len(hippoPodTemplate.Spec.Containers) != 1 {
		return
	}
	hippoPodTemplate.Spec.Containers[0].Name = carbonv1.MainContainer
	hippoPodTemplate.ObjectMeta.Annotations[carbonv1.AnnotationInjectSystemd] = carbonv1.MainContainer
}

func setInjectIdentifier(hippoPodTemplate *carbonv1.HippoPodTemplate) {
	if features.C2MutableFeatureGate.Enabled(features.InjectIdentifier) {
		hippoPodTemplate.ObjectMeta.Labels[carbonv1.LabelKeyIdentifier] = "true"
	}
}

func fillTemplateByMetaTags(hippoPodTemplate *carbonv1.HippoPodTemplate, metaTags *map[string]string, resourceLabels map[string]string) error {
	if nil == metaTags {
		return nil
	}
	labels, annos, err := transMetaTags(metaTags)
	if nil != err {
		glog.Errorf("trans meta tags: %s, error: %v", utils.ObjJSON(metaTags), err)
		return err
	}
	carbonv1.MergeMap(resourceLabels, hippoPodTemplate.ObjectMeta.Labels)
	carbonv1.MergeMap(labels, hippoPodTemplate.ObjectMeta.Labels)
	carbonv1.MergeMap(annos, hippoPodTemplate.ObjectMeta.Annotations)
	return nil
}

const (
	zoneNameArgPrefix   = "zoneName="
	gigPlatformKey      = "PLATFORM"
	gigPlatformLabelKey = "app.c2.io/biz-PLATFORM"
)

func fillTemplateLabelByEnvs(hippoPodTemplate *carbonv1.HippoPodTemplate, containers []carbonv1.HippoContainer) string {
	if nil == hippoPodTemplate || nil == containers {
		return ""
	}
	var zoneName string
	var findGigPlatform, findZoneName bool
	for _, container := range containers {
		for _, env := range container.Env {
			if env.Name == gigPlatformKey {
				hippoPodTemplate.Labels[gigPlatformLabelKey] = env.Value
				findGigPlatform = true
				break
			}
		}
		for _, arg := range container.Args {
			if strings.HasPrefix(arg, zoneNameArgPrefix) {
				zoneName = strings.TrimPrefix(arg, zoneNameArgPrefix)
				carbonv1.SetObjectZoneName(hippoPodTemplate, zoneName)
				findZoneName = true
			}
		}
		if findGigPlatform && findZoneName {
			break
		}
	}
	return zoneName
}

func transMetaTags(metaTags *map[string]string) (map[string]string, map[string]string, error) {
	if nil == metaTags || 0 == len(*metaTags) {
		return nil, nil, nil
	}
	var (
		labels = map[string]string{}
		annos  = map[string]string{}
	)
	// HACK: move portrait keys to annotation
	portraitKs := map[string]bool{
		carbonv1.LabelKeyAppShortName:  true,
		carbonv1.LabelKeyRoleShortName: true,
	}
	// make these label value valid
	validationKs := map[string]bool{
		carbonv1.LabelServerlessAppName:       true,
		carbonv1.LabelServerlessInstanceGroup: true,
		carbonv1.LabelServerlessBizPlatform:   true,
		carbonv1.LabelServerlessAppStage:      true,
		carbonv1.LabelKeyC2GroupName:          true,
	}
	for k, v := range *metaTags {
		parts := strings.Split(k, ":")
		attr := parts[0]
		if len(parts) == 2 {
			k = parts[1]
		}
		errs := validation.IsQualifiedName(k)
		if 0 != len(errs) {
			return nil, nil, fmt.Errorf("metatags key not valid: %s, %s", k, errs)
		}
		if portraitKs[k] || (attr == "annotation" && k != attr) {
			// no need to validate annotation value
			annos[k] = v
			continue
		}
		if validationKs[k] {
			carbonv1.SetMapValidValue(k, v, labels, annos)
			continue
		}
		errs = validation.IsValidLabelValue(v)
		if 0 != len(errs) {
			return nil, nil, fmt.Errorf("metatags value not valid: %s, %s", v, errs)
		}
		labels[k] = v
	}
	return labels, annos, nil
}

func transWorkingDirs(workingDir string) (map[string]string, error) {
	var workingDirs = map[string]string{}
	subdirs := strings.Split(workingDir, ",")
	for i := range subdirs {
		subdir := strings.Split(subdirs[i], ":")
		if len(subdir) < 2 {
			continue
		}
		workingDirs[subdir[0]] = subdir[1]
	}
	return workingDirs, nil
}

func transUlimits(ulimitsStr string, config *carbonv1.ContainerConfig) error {
	ulimitStrs := strings.Split(ulimitsStr, ",")
	var ulimits = make([]carbonv1.Ulimit, len(ulimitStrs))
	for i := range ulimitStrs {
		substr := strings.Split(ulimitStrs[i], ":")
		if len(substr) < 3 {
			err := fmt.Errorf("wrong ulimits format :%s", ulimitsStr)
			return err
		}
		ulimits[i].Name = substr[0]
		hard, err := strconv.Atoi(substr[1])
		if nil != err {
			return err
		}
		soft, err := strconv.Atoi(substr[2])
		if nil != err {
			return err
		}
		ulimits[i].Hard = hard
		ulimits[i].Soft = soft
	}
	config.Ulimits = ulimits
	return nil
}

func transSysctl(sysctlStr string, podSecurityContext *corev1.PodSecurityContext) error {
	sysctlStrs := strings.Split(sysctlStr, ",")
	var sysctls = make([]corev1.Sysctl, len(sysctlStrs))
	for i := range sysctlStrs {
		substr := strings.Split(sysctlStrs[i], "=")
		if len(substr) < 2 {
			err := fmt.Errorf("wrong ulimits format :%s", sysctlStr)
			return err
		}
		sysctls[i].Name = substr[0]
		sysctls[i].Value = substr[1]
	}
	podSecurityContext.Sysctls = sysctls
	return nil
}

func transAddCapability(addCapabilityStr string, securityContext *corev1.SecurityContext) error {
	if nil == securityContext.Capabilities {
		securityContext.Capabilities = &corev1.Capabilities{}
	}
	addCapabilityStrs := strings.Split(addCapabilityStr, ",")
	var add = make([]corev1.Capability, len(addCapabilityStrs))
	for i := range addCapabilityStrs {
		add[i] = corev1.Capability(addCapabilityStrs[i])
	}
	securityContext.Capabilities.Add = add
	return nil
}

func transPrivileged(privilegedStr string, securityContext *corev1.SecurityContext) error {
	if privilegedStr == "true" || privilegedStr == "TRUE" {
		var privileged = true
		securityContext.Privileged = &privileged
	}
	return nil
}

func transDropCapability(dropCapabilityStr string, securityContext *corev1.SecurityContext) error {
	if nil == securityContext.Capabilities {
		securityContext.Capabilities = &corev1.Capabilities{}
	}
	dropCapabilityStrs := strings.Split(dropCapabilityStr, ",")
	var drop = make([]corev1.Capability, len(dropCapabilityStrs))
	for i := range dropCapabilityStrs {
		drop[i] = corev1.Capability(dropCapabilityStrs[i])
	}
	securityContext.Capabilities.Drop = drop
	return nil
}

func appendC2ENVs(groupID, roleID string, envs []corev1.EnvVar) []corev1.EnvVar {
	if nil == envs {
		envs = []corev1.EnvVar{}
	}
	envs = append(envs, corev1.EnvVar{
		Name:  envKeyC2Group,
		Value: groupID,
	}, corev1.EnvVar{
		Name:  envKeyC2Role,
		Value: roleID,
	})
	return envs
}

func getTerminationGracePeriodSeconds(containers []carbonv1.HippoContainer) int64 {
	terminationGracePeriodSeconds := int64(100)
	for _, container := range containers {
		if container.Configs != nil {
			terminationGracePeriodSeconds = integer.Int64Max(terminationGracePeriodSeconds, int64(container.Configs.StopGracePeriod))
		}
	}
	return terminationGracePeriodSeconds
}

func transContaierEnvs(containerConfig string) (containerEnvs []corev1.EnvVar, err error) {
	envstr := containerConfig[6 : len(containerConfig)-1]
	envs := strings.Split(envstr, ",")
	if 0 != len(envs) {
		containerEnvs = make([]corev1.EnvVar, 0, len(envs))
		for j := range envs {
			kvs := strings.SplitN(envs[j], "=", 2)
			if len(kvs) == 2 {
				containerEnv := corev1.EnvVar{
					Name:  carbonv1.ContainerConfigEnvPrefix + strings.TrimSpace(kvs[0]),
					Value: strings.TrimSpace(kvs[1]),
				}
				containerEnvs = append(containerEnvs, containerEnv)
			} else {
				err = fmt.Errorf("not k=v format :%s", envs[j])
				return
			}
		}
	}
	return
}

func transBindMounts(appName, roleName string, containerConfig string) (
	tvolumes []corev1.Volume,
	tvolumeMounts []corev1.VolumeMount,
	err error) {
	volumnstr := containerConfig[13 : len(containerConfig)-1]
	containerVolumes := strings.Split(volumnstr, ",")
	tvolumes = make([]corev1.Volume, len(containerVolumes))
	tvolumeMounts = make([]corev1.VolumeMount, len(containerVolumes))
	var volumeType = corev1.HostPathDirectoryOrCreate
	if 0 != len(containerVolumes) {
		for j := range containerVolumes {
			var config = containerVolumes[j]
			if strings.Contains(containerVolumes[j], "${appName}") {
				config = strings.ReplaceAll(containerVolumes[j], "${appName}", appName)
				fmt.Println("aaa", config)
			}
			if strings.Contains(config, "${roleName}") {
				config = strings.ReplaceAll(config, "${roleName}", roleName)
				fmt.Println("vvv", config)
			}
			fmt.Println(containerVolumes[j], config)
			kvs := strings.Split(config, ":")
			if len(kvs) < 2 {
				continue
			}
			var volume = corev1.Volume{
				Name: fmt.Sprintf("volume-%d", j),
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: kvs[0],
						Type: &volumeType,
					},
				},
			}
			var volumeMount = corev1.VolumeMount{
				Name:      fmt.Sprintf("volume-%d", j),
				MountPath: kvs[1],
			}
			if len(kvs) == 3 && kvs[2] == "ro" {
				volumeMount.ReadOnly = true
			}
			tvolumes[j] = volume
			tvolumeMounts[j] = volumeMount
		}
	}
	return
}

func transDownwardAPI(containerConfig string) (
	tvolumes []corev1.Volume,
	tvolumeMounts []corev1.VolumeMount,
	err error) {
	volumnstr := containerConfig[14 : len(containerConfig)-1]
	kvs := strings.Split(volumnstr, ":")
	if len(kvs) != 2 {
		return
	}
	volumnstr = kvs[0]
	mountPath := kvs[1]
	containerVolumes := strings.Split(volumnstr, ",")
	var volume = corev1.Volume{
		Name: "podinfo-downward",
		VolumeSource: corev1.VolumeSource{
			DownwardAPI: &corev1.DownwardAPIVolumeSource{
				Items: []corev1.DownwardAPIVolumeFile{},
			},
		},
	}
	for i := range containerVolumes {
		podpath := containerVolumes[i]
		var path = podpath
		slices := strings.Split(podpath, ".")
		if len(slices) > 0 {
			path = slices[len(slices)-1]
		}
		volume.VolumeSource.DownwardAPI.Items = append(volume.VolumeSource.DownwardAPI.Items,
			corev1.DownwardAPIVolumeFile{
				Path: path,
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: podpath,
				},
			},
		)
	}
	var volumeMount = corev1.VolumeMount{
		Name:      "podinfo-downward",
		MountPath: mountPath,
	}
	tvolumes = []corev1.Volume{volume}
	tvolumeMounts = []corev1.VolumeMount{volumeMount}
	return
}

type kataVolumn struct {
	Name      string `json:"name,omitempty"`
	Type      string `json:"type,omitempty"`
	Capacity  string `json:"capacity,omitempty"`
	Format    string `json:"format,omitempty"`
	MountPath string `json:"mountPath,omitempty"`
}

// hippoExtend defines hippo extend
type hippoExtend struct {
	Resources        []hippo.Resource     `json:"resources,omitempty"`
	DelayORResources []hippo.SlotResource `json:"delayORResources,omitempty"`
}

func transKataMounts(containerConfig string) (
	katavolumns []json.RawMessage,
	tvolumes []corev1.Volume,
	tvolumeMounts []corev1.VolumeMount,
	err error) {
	volumnstr := containerConfig[len(kataVolumeMounts)+1:]
	err = json.Unmarshal([]byte(volumnstr), &katavolumns)
	if nil != err {
		glog.Errorf("%s, %v", volumnstr, err)
		return
	}
	var kvolumns []kataVolumn
	err = json.Unmarshal([]byte(volumnstr), &kvolumns)
	if nil != err {
		glog.Error(err)
		return
	}
	tvolumeMounts = make([]corev1.VolumeMount, len(kvolumns))
	tvolumes = make([]corev1.Volume, len(kvolumns))
	for i := range kvolumns {
		var volume = corev1.Volume{
			Name: kvolumns[i].Name,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		}
		tvolumes[i] = volume
		var volumeMount = corev1.VolumeMount{
			MountPath: kvolumns[i].MountPath,
			Name:      kvolumns[i].Name,
		}
		tvolumeMounts[i] = volumeMount
	}
	return
}

func transLabels(containerConfig string) (
	containerLabels map[string]string,
	err error,
) {
	labelstr := containerConfig[8 : len(containerConfig)-1]
	labels := strings.Split(labelstr, ",")
	containerLabels = make(map[string]string)
	if 0 != len(labels) {
		for j := range labels {
			kvs := strings.SplitN(labels[j], "=", 2)
			if len(kvs) == 2 {
				containerLabels[kvs[0]] = kvs[1]
			} else {
				containerLabels[kvs[0]] = ""
			}
		}
	}
	return
}

func transTimeZone(containerConfig string) (
	volume1 corev1.Volume,
	volume2 corev1.Volume,
	volumeMount1 corev1.VolumeMount,
	volumeMount2 corev1.VolumeMount,
	err error,
) {
	kvs := strings.SplitN(containerConfig, "=", 2)
	if len(kvs) < 2 {
		err = fmt.Errorf("timezone format error :%s", containerConfig)
		return
	}
	var volumeTypeFile = corev1.HostPathFile
	timezone := kvs[1]
	timezonepath := path.Join("/usr/share/zoneinfo/", timezone)
	volume1 = corev1.Volume{
		Name: "timezone-autotrans-0",
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: timezonepath,
				Type: &volumeTypeFile,
			},
		},
	}
	volume2 = corev1.Volume{
		Name: "timezone-autotrans-1",
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: timezonepath,
				Type: &volumeTypeFile,
			},
		},
	}
	volumeMount1 = corev1.VolumeMount{
		Name:      "timezone-autotrans-0",
		MountPath: "/etc/localtime",
		ReadOnly:  true,
	}
	volumeMount2 = corev1.VolumeMount{
		Name:      "timezone-autotrans-1",
		MountPath: timezonepath,
		ReadOnly:  true,
	}
	return
}

type containerStConfigs struct {
	devices         *[]carbonv1.Device
	config          *carbonv1.ContainerConfig
	containerEnvs   []corev1.EnvVar
	containerLabels map[string]string

	hippoVolumes []json.RawMessage
	volumes      []corev1.Volume
	volumeMounts []corev1.VolumeMount

	cpuShareNum           int32
	containerModel        string
	predictDelayTime      int
	pouchAnnotations      string
	serviceAccount        string
	needHippoMounts       *bool
	restartWithoutRemove  *bool
	delayDeleteBackup     int64
	warmUpSeconds         int64
	schedulerName         string
	workingDirs           map[string]string
	dependencyLevel       int
	shareProcessNamespace *bool
	dnsPolicy             string

	securityContext    *corev1.SecurityContext
	podSecurityContext *corev1.PodSecurityContext
}

func transContainerConfigs(appName, roleName string, containerConfigs []string) (*containerStConfigs, error) {
	var (
		conf containerStConfigs
		err  error
	)
	for i := range containerConfigs {
		if strings.HasPrefix(containerConfigs[i], containerConfKeyEnvs) {
			conf.containerEnvs, err = transContaierEnvs(containerConfigs[i])
		} else if strings.HasPrefix(containerConfigs[i], containerConfKeyLabels) {
			conf.containerLabels, err = transLabels(containerConfigs[i])
		} else if strings.HasPrefix(containerConfigs[i], containerConfKeyBindMounts) {
			var tvolumes []corev1.Volume
			var tvolumeMounts []corev1.VolumeMount
			tvolumes, tvolumeMounts, err = transBindMounts(appName, roleName, containerConfigs[i])
			conf.volumes = append(conf.volumes, tvolumes...)
			conf.volumeMounts = append(conf.volumeMounts, tvolumeMounts...)
		} else if strings.HasPrefix(containerConfigs[i], containerConfKeyDownwardAPI) {
			var tvolumes []corev1.Volume
			var tvolumeMounts []corev1.VolumeMount
			tvolumes, tvolumeMounts, err = transDownwardAPI(containerConfigs[i])
			conf.volumes = append(conf.volumes, tvolumes...)
			conf.volumeMounts = append(conf.volumeMounts, tvolumeMounts...)
		} else if strings.HasPrefix(containerConfigs[i], kataVolumeMounts) {
			var kvolumes []corev1.Volume
			var kvolumeMounts []corev1.VolumeMount
			conf.hippoVolumes, kvolumes, kvolumeMounts, err = transKataMounts(containerConfigs[i])
			conf.volumes = append(conf.volumes, kvolumes...)
			conf.volumeMounts = append(conf.volumeMounts, kvolumeMounts...)
		} else if strings.HasPrefix(containerConfigs[i], containerConfKeyTimezone) {
			var volume1, volume2 corev1.Volume
			var volumeMount1, volumeMount2 corev1.VolumeMount
			volume1, volume2, volumeMount1, volumeMount2, err = transTimeZone(containerConfigs[i])
			conf.volumes = append(conf.volumes, volume1, volume2)
			conf.volumeMounts = append(conf.volumeMounts, volumeMount1, volumeMount2)
		} else if strings.HasPrefix(containerConfigs[i], containerConfKeyVolumeMounts) {
			volumeStr := containerConfigs[i][2+len(containerConfKeyVolumeMounts) : len(containerConfigs[i])-1]
			vols, volms := transVolumeMounts(volumeStr)
			conf.volumes = append(conf.volumes, vols...)
			conf.volumeMounts = append(conf.volumeMounts, volms...)
		} else if strings.HasPrefix(containerConfigs[i], containerConfKeyWorkingDir) {
			workingDirStr := containerConfigs[i][2+len(containerConfKeyWorkingDir) : len(containerConfigs[i])-1]
			conf.workingDirs, err = transWorkingDirs(workingDirStr)
		} else if strings.HasPrefix(containerConfigs[i], containerConfKeyUlimits) {
			if conf.config == nil {
				conf.config = &carbonv1.ContainerConfig{}
			}
			ulimitsStr := containerConfigs[i][2+len(containerConfKeyUlimits) : len(containerConfigs[i])-1]
			err = transUlimits(ulimitsStr, conf.config)
		} else if strings.HasPrefix(containerConfigs[i], containerConfKeySysctls) {
			conf.podSecurityContext = &corev1.PodSecurityContext{}
			systctlStr := containerConfigs[i][2+len(containerConfKeySysctls) : len(containerConfigs[i])-1]
			err = transSysctl(systctlStr, conf.podSecurityContext)
		} else if strings.HasPrefix(containerConfigs[i], containerConfKeyCapability) {
			if nil == conf.securityContext {
				conf.securityContext = &corev1.SecurityContext{}
			}
			capabilityStr := containerConfigs[i][2+len(containerConfKeyCapability) : len(containerConfigs[i])-1]
			err = transAddCapability(capabilityStr, conf.securityContext)
		} else if strings.HasPrefix(containerConfigs[i], containerConfKeyCapabilityDrop) {
			if nil == conf.securityContext {
				conf.securityContext = &corev1.SecurityContext{}
			}
			dropCapabilityStr := containerConfigs[i][2+len(containerConfKeyCapabilityDrop) : len(containerConfigs[i])-1]
			err = transDropCapability(dropCapabilityStr, conf.securityContext)
		} else if strings.HasPrefix(containerConfigs[i], containerConfKeyPrivileged) {
			if nil == conf.securityContext {
				conf.securityContext = &corev1.SecurityContext{}
			}
			privilegedStr := containerConfigs[i][1+len(containerConfKeyPrivileged) : len(containerConfigs[i])]
			err = transPrivileged(privilegedStr, conf.securityContext)
		} else if strings.HasPrefix(containerConfigs[i], containerConfKeyDevice) {
			deviceStr := containerConfigs[i][1+len(containerConfKeyDevice) : len(containerConfigs[i])]
			devicesConfig := strings.Split(deviceStr, ";")
			var devices []carbonv1.Device = make([]carbonv1.Device, len(devicesConfig))
			for i := range devicesConfig {
				path := devicesConfig[i]
				devices[i].PathOnHost = path
				devices[i].PathInContainer = path
				devices[i].CgroupPermissions = "mrw"
			}
			conf.devices = &devices
		} else if strings.HasPrefix(containerConfigs[i], containerConfKeyVolumeMountsJSON) {
			volumeMountsStr := containerConfigs[i][len(containerConfKeyVolumeMountsJSON)+1:]
			var volumeMounts []corev1.VolumeMount
			err = json.Unmarshal([]byte(volumeMountsStr), &volumeMounts)
			if err != nil {
				return nil, err
			}
			conf.volumeMounts = append(conf.volumeMounts, volumeMounts...)
		} else if strings.HasPrefix(containerConfigs[i], containerConfKeyVolumes) {
			volumesStr := containerConfigs[i][len(containerConfKeyVolumes)+1:]
			var volumes []corev1.Volume
			err = json.Unmarshal([]byte(volumesStr), &volumes)
			if err != nil {
				return nil, err
			}
			conf.volumes = append(conf.volumes, volumes...)
		} else if strings.HasPrefix(containerConfigs[i], containerConfKeyMemPolicy) {
			kvs := strings.SplitN(containerConfigs[i], "=", 2)
			if len(kvs) < 2 {
				err = fmt.Errorf("invalid extra container conf: %s", containerConfigs[i])
				return nil, err
			}
			if kvs[1] == "MEM_INTERLEAVE" {
				if conf.containerEnvs == nil {
					conf.containerEnvs = []corev1.EnvVar{}
				}
				conf.containerEnvs = append(conf.containerEnvs, corev1.EnvVar{
					Name:  "NUMA_POLICY",
					Value: "MPOL_INTERLEAVE:65535",
				})
			} else {
				err = fmt.Errorf("invalid MEM_POLICY: %s", kvs[1])
				return nil, err
			}
		} else {
			if conf.config == nil {
				conf.config = &carbonv1.ContainerConfig{}
			}
			var tmpneedHippoMounts, tmprestartWithoutRemove, share *bool
			if tmpneedHippoMounts, tmprestartWithoutRemove, share, err =
				transExtraContainerConfig(
					containerConfigs[i],
					conf.config,
					&conf.cpuShareNum,
					&conf.containerModel,
					&conf.predictDelayTime,
					&conf.pouchAnnotations,
					&conf.serviceAccount,
					&conf.delayDeleteBackup,
					&conf.warmUpSeconds,
					&conf.schedulerName,
					&conf.dnsPolicy,
					&conf.dependencyLevel); err != nil {
				return nil, err
			}
			if nil != tmpneedHippoMounts {
				conf.needHippoMounts = tmpneedHippoMounts
			}
			if nil != tmprestartWithoutRemove {
				conf.restartWithoutRemove = tmprestartWithoutRemove
			}
			if nil != share {
				conf.shareProcessNamespace = share
			}
		}
		if nil != err {
			return nil, err
		}
	}
	return &conf, nil
}

// Make share volumes from image by all containers.
// Declare `VOLUME` in your image, and this function mount that volume. It's different from `BIND_MOUNTS` which mounts
// directory from host file system.
func transVolumeMounts(vols string) (volumes []corev1.Volume, volumeMounts []corev1.VolumeMount) {
	volumes, volumeMounts = doTransPodVolumes(strings.Split(vols, ","))
	return
}

func transExtraContainerConfig(
	kvstr string,
	config *carbonv1.ContainerConfig,
	cpuShareNum *int32,
	containerModel *string,
	predictDelayTime *int,
	pouchAnnotations *string,
	serviceAccount *string,
	delayDeleteBackup *int64,
	warmUpSeconds *int64,
	schedulerName *string,
	dnsPolicy *string,
	dependencyLevel *int) (
	needHippoMounts *bool,
	restartWithoutRemove *bool,
	shareProcessNamespace *bool,
	err error) {
	kvs := strings.SplitN(kvstr, "=", 2)
	if len(kvs) < 2 {
		glog.Warningf("invalid extra container conf: %s", kvstr)
		return
	}
	switch kvs[0] {
	case "NO_MEM_LIMIT":
	case "NO_MEMCG_RECLAIM":
		config.NoMemcgReclaim = kvs[1]
	case "MEM_WMARK_RATIO":
		config.MemWmarkRatio = kvs[1]
	case "MEM_FORCE_EMPTY":
		config.MemForceEmpty = kvs[1]
	case "MEM_EXTRA_BYTES":
		config.MemExtraBytes = kvs[1]
	case "MEM_EXTRA_RATIO":
		config.MemExtraRatio = kvs[1]
	case "OOM_CONTAINER_STOP":
	case "OOM_KILL_DISABLE":
	case "NET_PRIORITY":
		config.NetPriority = kvs[1]
	case "CPU_BVT_WARP_NS":
		config.CPUBvtWarpNs = kvs[1]
	case "CPU_LLC_CACHE":
		config.CPUllcCache = kvs[1]
	case "CONTAINER_MODEL":
		*containerModel = kvs[1]
	case "CPU_RESERVED_TO_SHARE_NUM":
		shareNum, _ := strconv.Atoi(kvs[1])
		*cpuShareNum = int32(shareNum)
	case "PREDICT_DELAY_SECONDS": // TODO:
		delayTime, _ := strconv.Atoi(kvs[1])
		*predictDelayTime = delayTime
	case "POUCH_ANNOTATIONS":
		*pouchAnnotations = kvs[1]
	case "SERVICE_ACCOUNT":
		*serviceAccount = kvs[1]
	case "RESTART_WITHOUT_REMOVE":
		restart, _ := strconv.ParseBool(kvs[1])
		restartWithoutRemove = &restart
	case "NEED_HIPPO_MOUNT":
		needmount, _ := strconv.ParseBool(kvs[1])
		needHippoMounts = &needmount
	case "SHARE_PROCESS_NAMESPACE":
		share, _ := strconv.ParseBool(kvs[1])
		shareProcessNamespace = &share
	case "DELAY_DELETE_BACKUP_SECONDS":
		delaySeconds, _ := strconv.Atoi(kvs[1])
		*delayDeleteBackup = int64(delaySeconds)
	case "WARM_UP_SECONDS":
		warmSeconds, _ := strconv.Atoi(kvs[1])
		*warmUpSeconds = int64(warmSeconds)
	case "SCHEDULER_NAME":
		*schedulerName = kvs[1]
	case "DEPENDENCY_LEVEL":
		level, _ := strconv.Atoi(kvs[1])
		*dependencyLevel = level
	case containerConfKeyDNSPolicy:
		*dnsPolicy = kvs[1]
	default:
		err = fmt.Errorf("not support containerConfig [%s]", kvstr)
		return
	}
	return
}

func transPackages(id string, pkgs []typespec.PackageInfo) ([]carbonv1.PackageInfo, string) {
	var packages = make([]carbonv1.PackageInfo, 0, len(pkgs))
	var image string
	for i := range pkgs {
		if "IMAGE" != pkgs[i].Type {
			var pkg = carbonv1.PackageInfo{
				PacakgeType: pkgs[i].Type,
				PackageURI:  pkgs[i].URI,
			}
			packages = append(packages, pkg)
		} else {
			if image == "" {
				image = pkgs[i].URI
			} else {
				glog.Errorf("ignore image :%s with exist image :%s", pkgs[i].URI, image)
			}
		}
	}
	if image == "" {
		glog.Errorf("%s empty image, and auto set to reg.docker.alibaba-inc.com/hippo/hippo_alios5u7_base:1.0", id)
		image = "reg.docker.alibaba-inc.com/hippo/hippo_alios5u7_base:1.0"
	}
	return packages, image
}

func MergeEnvs(containerEnvs, processEnvs []corev1.EnvVar) []corev1.EnvVar {
	var results = processEnvs
	for i := range containerEnvs {
		name := containerEnvs[i].Name
		conflict := false
		for j := range processEnvs {
			if name == processEnvs[j].Name {
				if name == "PATH" || name == "LD_LIBRARY_PATH" {
					results[j].Value = processEnvs[j].Value + ":" + containerEnvs[i].Value
				}
				conflict = true
				break
			}
		}
		if !conflict {
			results = append(results, containerEnvs[i])
		}
	}
	return results
}

func transScalarResource(name string, amount int32,
	requireMatchExpressions *[]corev1.NodeSelectorRequirement,
	resourceRequirements *corev1.ResourceRequirements) (ip, hippoExtend bool, err error) {
	var quantityPtr *k8sresource.Quantity
	var requestQuantityPtr *k8sresource.Quantity
	var k8sname string
	switch name {
	case "cpu":
		var quantity k8sresource.Quantity
		err = quantity.UnmarshalJSON([]byte(fmt.Sprintf("\"%dm\"", amount*10)))
		if nil != err {
			err = fmt.Errorf("ParseQuantity error %v", err)
			glog.Error(err)
			return
		}
		quantityPtr = &quantity
		k8sname = carbonv1.ResourceCPU
	case carbonv1.ResourceRequestCpu:
		var quantity k8sresource.Quantity
		err = quantity.UnmarshalJSON([]byte(fmt.Sprintf("\"%dm\"", amount*10)))
		if nil != err {
			err = fmt.Errorf("ParseQuantity error %v", err)
			glog.Error(err)
			return
		}
		requestQuantityPtr = &quantity
		k8sname = carbonv1.ResourceCPU
	case "ip":
		quantityPtr = k8sresource.NewQuantity(int64(1), k8sresource.DecimalExponent)
		k8sname = carbonv1.ResourceIP
		ip = true
	case "mem":
		var quantity k8sresource.Quantity
		quantity, err = k8sresource.ParseQuantity(fmt.Sprintf("%dMi", amount))
		if nil != err {
			err = fmt.Errorf("ParseQuantity error %v", err)
			glog.Error(err)
			return
		}
		quantityPtr = &quantity
		k8sname = carbonv1.ResourceMEM
	case "gpu-mem":
		var quantity k8sresource.Quantity
		quantity, err = k8sresource.ParseQuantity(fmt.Sprintf("%dMi", amount))
		if nil != err {
			err = fmt.Errorf("ParseQuantity error %v", err)
			glog.Error(err)
			return
		}
		quantityPtr = &quantity
		k8sname = carbonv1.ResourceGPUMEM
	case "gpu-core":
		quantityPtr = k8sresource.NewQuantity(int64(amount), k8sresource.DecimalExponent)
		k8sname = carbonv1.ResourceGPUCore
	case "gpu":
		quantityPtr = k8sresource.NewQuantity(int64(amount), k8sresource.DecimalExponent)
		k8sname = carbonv1.ResourceGPU
	case "fpga":
		quantityPtr = k8sresource.NewQuantity(int64(amount), k8sresource.DecimalExponent)
		k8sname = carbonv1.ResourceFPGA
	case "npu":
		quantityPtr = k8sresource.NewQuantity(int64(amount), k8sresource.DecimalExponent)
		k8sname = "alibaba.com/npu"
	case "pov":
		quantityPtr = k8sresource.NewQuantity(int64(amount), k8sresource.DecimalExponent)
		k8sname = "alibaba.com/pov"
	case "eni":
		quantityPtr = k8sresource.NewQuantity(int64(int64(1)), k8sresource.DecimalExponent)
		k8sname = carbonv1.ResourceENI
	case "T4":
		var nodeRequirement = corev1.NodeSelectorRequirement{
			Key:      name,
			Operator: corev1.NodeSelectorOpIn,
			Values:   []string{name},
		}
		*requireMatchExpressions = append(*requireMatchExpressions, nodeRequirement)
	default:
		if strings.HasPrefix(name, "disk_size") {
			var quantity k8sresource.Quantity
			quantity, err = k8sresource.ParseQuantity(fmt.Sprintf("%dMi", amount))
			if nil != err {
				err = fmt.Errorf("ParseQuantity error %v", err)
				glog.Error(err)
				return
			}
			quantityPtr = &quantity
			k8sname = carbonv1.HippoResourcePrefix + name
		} else if strings.HasPrefix(name, "alibabacloud.com") || strings.HasPrefix(name, "nvidia") {
			// 透传字段
			k8sname = name
			quantityPtr = k8sresource.NewQuantity(int64(amount), k8sresource.DecimalExponent)
		} else if strings.HasPrefix(name, "disk_ratio") && features.C2MutableFeatureGate.Enabled(features.SupportDiskRatio) {
			// 透传字段
			k8sname = carbonv1.ResourceDiskRatio
			quantityPtr = k8sresource.NewQuantity(int64(amount), k8sresource.DecimalExponent)
		} else {
			hippoExtend = true
		}
	}
	if quantityPtr != nil {
		resourceRequirements.Limits[corev1.ResourceName(k8sname)] = *quantityPtr
	}
	if requestQuantityPtr != nil {
		if resourceRequirements.Requests == nil {
			resourceRequirements.Requests = make(map[corev1.ResourceName]k8sresource.Quantity)
		}
		resourceRequirements.Requests[corev1.ResourceName(k8sname)] = *requestQuantityPtr
	}
	return
}

func transTEXTResource(name string, requireMatchExpressions *[]corev1.NodeSelectorRequirement) {
	key, values := getKVFromText(name)
	var nodeRequirement = corev1.NodeSelectorRequirement{
		Key:      key,
		Operator: corev1.NodeSelectorOpIn,
		Values:   values,
	}
	*requireMatchExpressions = append(*requireMatchExpressions, nodeRequirement)
}

func getKVFromText(name string) (key string, value []string) {
	key = name
	value = []string{name}
	fields := strings.Split(name, "=")
	if len(fields) == 2 {
		key = fields[0]
		value = strings.Split(fields[1], "/")
	}
	return
}

func transExcludeTextResource(name string, tolerations *[]corev1.Toleration, requireMatchExpressions *[]corev1.NodeSelectorRequirement) {
	key, values := getKVFromText(name)
	var value string
	if len(values) > 0 {
		value = values[0]
	}
	var isToleration bool
	fields := strings.Split(key, ":")
	if len(fields) == 2 && fields[0] == "toleration" {
		if value == key {
			value = fields[1]
		}
		key = fields[1]
		isToleration = true
	}
	var toleration = corev1.Toleration{
		Key:      key,
		Value:    value,
		Operator: corev1.TolerationOpEqual,
		Effect:   corev1.TaintEffectNoSchedule,
	}
	*tolerations = append(*tolerations, toleration)

	if !isToleration {
		var nodeRequirement = corev1.NodeSelectorRequirement{
			Key:      key,
			Operator: corev1.NodeSelectorOpIn,
			Values:   []string{value},
		}
		*requireMatchExpressions = append(*requireMatchExpressions, nodeRequirement)
	}
}

func transPreferTextResource(name string, prefers *[]corev1.PreferredSchedulingTerm, labels map[string]string) {
	key, values := getKVFromText(name)
	var nodeRequirement = corev1.NodeSelectorRequirement{
		Key:      key,
		Operator: corev1.NodeSelectorOpIn,
		Values:   values,
	}
	prefer := corev1.PreferredSchedulingTerm{
		Weight: 1,
		Preference: corev1.NodeSelectorTerm{
			MatchExpressions: []corev1.NodeSelectorRequirement{nodeRequirement},
		},
	}
	for _, value := range values {
		if value == highQualityLabelValue {
			labels[qualityLabelKey] = highQualityLabelValue
		}
	}
	*prefers = append(*prefers, prefer)
}

func transPreferProhibitTextResource(name string, prefers *[]corev1.PreferredSchedulingTerm, labels map[string]string) {
	key, values := getKVFromText(name)
	var nodeRequirement = corev1.NodeSelectorRequirement{
		Key:      key,
		Operator: corev1.NodeSelectorOpNotIn,
		Values:   values,
	}
	prefer := corev1.PreferredSchedulingTerm{
		Weight: 1,
		Preference: corev1.NodeSelectorTerm{
			MatchExpressions: []corev1.NodeSelectorRequirement{nodeRequirement},
		},
	}
	for _, value := range values {
		if value == highQualityLabelValue {
			labels[qualityLabelKey] = highQualityLabelValue
		}
	}
	*prefers = append(*prefers, prefer)
}

func transProhibitResource(name string, requireMatchExpressions *[]corev1.NodeSelectorRequirement) {
	key, value := getKVFromText(name)
	var nodeRequirement = corev1.NodeSelectorRequirement{
		Key:      key,
		Operator: corev1.NodeSelectorOpNotIn,
		Values:   value,
	}
	*requireMatchExpressions = append(*requireMatchExpressions, nodeRequirement)
}

func transResources(resourcess []carbon.SlotResource) (*corev1.ResourceRequirements, *corev1.Affinity, []corev1.Toleration, map[string]string, []string, bool, bool, *hippoExtend, error) {
	var resourceRequirements corev1.ResourceRequirements
	var requireMatchExpressions []corev1.NodeSelectorRequirement = make([]corev1.NodeSelectorRequirement, 0)
	var prefers []corev1.PreferredSchedulingTerm = make([]corev1.PreferredSchedulingTerm, 0)
	var tolerations = make([]corev1.Toleration, 0)
	var exclisives = make([]string, 0)
	var labels = map[string]string{}
	var hasip = false
	var processMode = true
	resourceRequirements.Limits = make(map[corev1.ResourceName]k8sresource.Quantity)
	var hippoExtend hippoExtend

	for _, resource := range resourcess {
		if nil == resource.SlotResources || 0 == len(resource.SlotResources) {
			err := fmt.Errorf("nil declarations")
			glog.Error(err)
			return nil, nil, nil, nil, nil, hasip, processMode, nil, err
		}
		for _, res := range resource.SlotResources {
			name := res.GetName()
			amount := res.GetAmount()
			resType := res.GetType()
			if name == "T4" {
				processMode = false
			}
			switch resType {
			case hippo.Resource_SCALAR:
				isip, isHippoExtend, err := transScalarResource(name, amount, &requireMatchExpressions, &resourceRequirements)
				if nil != err {
					return nil, nil, nil, nil, nil, hasip, processMode, nil, err
				}
				if isip {
					processMode = false
					hasip = true
				}
				if isHippoExtend {
					hippoExtend.Resources = append(hippoExtend.Resources, *res)
				}
			case hippo.Resource_TEXT:
				transTEXTResource(name, &requireMatchExpressions)
			case hippo.Resource_EXCLUDE_TEXT:
				transExcludeTextResource(name, &tolerations, &requireMatchExpressions)
			case hippo.Resource_QUEUE_NAME:
			case hippo.Resource_EXCLUSIVE:
				newName := strings.Replace(name, ":", "-", -1)
				exclisives = append(exclisives, newName)
			case hippo.Resource_PREFER_TEXT:
				transPreferTextResource(name, &prefers, labels)
			case hippo.Resource_PREFER_PROHIBIT_TEXT:
				transPreferProhibitTextResource(name, &prefers, labels)
			case hippo.Resource_PROHIBIT_TEXT:
				transProhibitResource(name, &requireMatchExpressions)
			case hippo.Resource_SCALAR_CMP:
			}
		}
		break
	}

	var orResourceRequirements = [][]corev1.NodeSelectorRequirement{}
	if len(resourcess) > 1 {
		for i := 1; i < len(resourcess); i++ {
			var hippoRes = hippo.SlotResource{Resources: resourcess[i].SlotResources}
			hippoExtend.DelayORResources = append(hippoExtend.DelayORResources, hippoRes)
			var tmpRequireMatchExpressions = []corev1.NodeSelectorRequirement{}
			for _, res := range resourcess[i].SlotResources {
				name := res.GetName()
				resType := res.GetType()
				switch resType {
				case hippo.Resource_TEXT:
					transTEXTResource(name, &tmpRequireMatchExpressions)
				case hippo.Resource_EXCLUDE_TEXT:
					transExcludeTextResource(name, &tolerations, &tmpRequireMatchExpressions)
				}
			}
			orResourceRequirements = append(orResourceRequirements, tmpRequireMatchExpressions)
		}
	}

	var affinity corev1.Affinity
	if 0 != len(requireMatchExpressions) || 0 != len(prefers) || 0 != len(orResourceRequirements) {
		affinity.NodeAffinity = &corev1.NodeAffinity{}
		if 0 != len(requireMatchExpressions) || 0 != len(orResourceRequirements) {
			var nodeSelector = corev1.NodeSelector{
				NodeSelectorTerms: []corev1.NodeSelectorTerm{
					{
						MatchExpressions: requireMatchExpressions,
					},
				},
			}
			for i := range orResourceRequirements {
				tmpRequireMatchExpressions := orResourceRequirements[i]
				nodeSelector.NodeSelectorTerms = append(nodeSelector.NodeSelectorTerms, corev1.NodeSelectorTerm{
					MatchExpressions: tmpRequireMatchExpressions,
				})
			}
			affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = &nodeSelector
		}
		if 0 != len(prefers) {
			affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution = prefers
		}
	}

	if resourceRequirements.Requests != nil {
		request := resourceRequirements.Requests
		resourceRequirements.Requests = resourceRequirements.Limits.DeepCopy()
		for k := range request {
			v := request[k]
			resourceRequirements.Requests[k] = v
		}
	}
	return &resourceRequirements, &affinity, tolerations, labels, exclisives, hasip, processMode, &hippoExtend, nil
}

var registryTypeMap = map[typespec.ServiceAdapterType]carbonv1.RegistryType{
	typespec.STNONE:       carbonv1.ServiceNone,
	typespec.STCM2:        carbonv1.ServiceCM2,
	typespec.STVIP:        carbonv1.ServiceVipserver,
	typespec.STLVS:        carbonv1.ServiceLVS,
	typespec.STARMORY:     carbonv1.ServiceArmory,
	typespec.STSLB:        carbonv1.ServiceSLB,
	typespec.STHSF:        carbonv1.ServiceNone,
	typespec.STECSARMORY:  carbonv1.ServiceNone,
	typespec.STVPCSLB:     carbonv1.ServiceVPCSLB,
	typespec.STDROGOLVS:   carbonv1.ServiceNone,
	typespec.STSKYLINE:    carbonv1.ServiceSkyline,
	typespec.STECSSKYLINE: carbonv1.ServiceNone,
	typespec.STANTVIP:     carbonv1.ServiceAntvip,
	typespec.STLIBRARY:    carbonv1.ServiceGeneral,
}

func transServices(appName, groupID, roleName string, role *typespec.RolePlan, single bool, shardgroupName string, isAsiPod bool) ([]*carbonv1.ServicePublisher, error) {
	if 0 == len(role.Global.ServiceConfigs) {
		return nil, nil
	}
	if !single {
		roleName = utils.AppendString(shardgroupName, ".", roleName)
	}
	var services = make([]*carbonv1.ServicePublisher, 0, len(role.Global.ServiceConfigs))
	for i := range role.Global.ServiceConfigs {
		var config interface{}
		if role.Global.ServiceConfigs[i].GetMasked() {
			continue
		}
		err := json.Unmarshal([]byte(role.Global.ServiceConfigs[i].ConfigStr), &config)
		if nil != err {
			err = fmt.Errorf("unmarshal error :%s,%v", role.Global.ServiceConfigs[i].ConfigStr, err)
			glog.Error(err)
			return nil, err
		}
		b, _ := json.Marshal(&config)

		suffix, err := utils.SignatureShort(role.Global.ServiceConfigs[i])
		if nil != err {
			return nil, err
		}
		name := utils.AppendString(roleName, "-", suffix)
		name = EscapeName(name)
		var service = &carbonv1.ServicePublisher{
			ObjectMeta: metav1.ObjectMeta{
				Name:   name,
				Labels: map[string]string{},
			},
			Spec: carbonv1.ServicePublisherSpec{
				ServiceName:  role.Global.ServiceConfigs[i].Name,
				MetaStr:      role.Global.ServiceConfigs[i].GetMetaStr(),
				Type:         registryTypeMap[role.Global.ServiceConfigs[i].Type],
				RegistryConf: json.RawMessage(b),
				Selector:     map[string]string{},
			},
		}
		carbonv1.AddRsUniqueLabelHashKey(service.Labels, roleName)
		carbonv1.AddRsUniqueLabelHashKey(service.Spec.Selector, roleName)
		carbonv1.AddGroupUniqueLabelHashKey(service.Labels, shardgroupName)
		carbonv1.SetAppName(service, appName)
		if single {
			service.Labels[carbonv1.ControllersShardKey] =
				computeControllerShardKey(roleName)
			service.Labels[carbonv1.GroupTypeKey] = carbonv1.GroupTypeRollingset
		} else {
			service.Labels[carbonv1.ControllersShardKey] =
				computeControllerShardKey(shardgroupName)
			service.Labels[carbonv1.GroupTypeKey] = carbonv1.GroupTypeShardGroup
		}
		services = append(services, service)
	}
	return services, nil
}

func transPriority(role *typespec.RolePlan) (*int32, error) {
	return transMajorPriority(role.Version.ResourcePlan.GetPriority().GetMajorPriority())
}

func transMajorPriority(majorPriority int32) (*int32, error) {
	var priority int32
	if 0 == majorPriority {
		priority = 200
	} else if majorPriority <= 1 {
		priority = 195
	} else if majorPriority <= 32 {
		priority = 3*(65-int32(majorPriority)) + 1
	} else if majorPriority <= 33 {
		priority = 95
	} else if majorPriority <= 64 {
		priority = 3 * (64 - int32(majorPriority))
	} else {
		err := fmt.Errorf("invalid priority %d", majorPriority)
		return nil, err
	}

	return &priority, nil
}

func transDeclare(role *typespec.RolePlan) []string {
	if nil == role.Version.ResourcePlan.Declarations ||
		0 == len(*role.Version.ResourcePlan.Declarations) {
		return nil
	}
	var declares = make([]string, 0, len(*role.Version.ResourcePlan.Declarations))
	for _, res := range *role.Version.ResourcePlan.Declarations {
		declares = append(declares, res.GetName())
	}
	return declares
}

// GetBrokenRecoverQuotaConfig GetBrokenRecoverQuotaConfig
func GetBrokenRecoverQuotaConfig(g *typespec.GlobalPlan) *carbonv1.BrokenRecoverQuotaConfig {
	return &carbonv1.BrokenRecoverQuotaConfig{
		MaxFailedCount: utils.Int32Ptr(g.GetBrokenRecoverQuotaConfig().GetMaxFailedCount()),
		TimeWindow:     utils.Int32Ptr(g.GetBrokenRecoverQuotaConfig().GetTimeWindow()),
	}
}

// GetHealthCheckerConfig GetHealthCheckerConfig
func GetHealthCheckerConfig(g *typespec.GlobalPlan) *carbonv1.HealthCheckerConfig {
	if nil == g.HealthCheckerConfig || nil == g.HealthCheckerConfig.Name {
		return nil
	}

	switch *g.HealthCheckerConfig.Name {
	case typespec.DefaultHealthChecker:
		fallthrough
	case "":
		return nil
	case typespec.Lv7HealthChecker:
		return &carbonv1.HealthCheckerConfig{
			Type: carbonv1.Lv7Health,
			Lv7Config: &carbonv1.Lv7HealthCheckerConfig{
				LostTimeout: g.HealthCheckerConfig.GetLostTimeout(),
				Path:        g.HealthCheckerConfig.Args[typespec.KeyCheckPath],
				Port:        intstr.FromString(g.HealthCheckerConfig.Args[typespec.KeyPort]),
			},
		}
	case typespec.AdvancedLv7HealthChecker:
		return &carbonv1.HealthCheckerConfig{
			Type: carbonv1.AdvancedLv7Health,
			Lv7Config: &carbonv1.Lv7HealthCheckerConfig{
				LostTimeout: g.HealthCheckerConfig.GetLostTimeout(),
				Path:        g.HealthCheckerConfig.Args[typespec.KeyCheckPath],
				Port:        intstr.FromString(g.HealthCheckerConfig.Args[typespec.KeyPort]),
			},
		}
	}
	err := fmt.Errorf("tran healthcheck error with error name:%s", *g.HealthCheckerConfig.Name)
	glog.Error(err)
	return nil
}

// GetEnvs GetEnvs
func GetEnvs(p *typespec.ProcessInfo) ([]corev1.EnvVar, error) {
	if nil == p.Envs {
		return nil, nil
	}
	var envs = make([]corev1.EnvVar, 0, len(*p.Envs))
	for i := range *p.Envs {
		if 2 == len((*p.Envs)[i]) {
			env := corev1.EnvVar{
				Name:  (*p.Envs)[i][0],
				Value: (*p.Envs)[i][1],
			}
			envs = append(envs, env)
		} else if 0 == len((*p.Envs)[i]) {
			continue
		} else {
			err := fmt.Errorf("error env :%s,%s", *p.Name, (*p.Envs)[i])
			glog.Error(err)
			return nil, err
		}
	}
	return envs, nil
}

// GetEnvsFromHippoProcessInfo GetEnvsFromHippoProcessInfo
func GetEnvsFromHippoProcessInfo(p *hippo.ProcessInfo) ([]corev1.EnvVar, error) {
	if nil == p.Envs {
		return nil, nil
	}
	var envs = make([]corev1.EnvVar, len(p.Envs))
	for i := range p.Envs {
		env := corev1.EnvVar{
			Name:  (p.Envs)[i].GetKey(),
			Value: (p.Envs)[i].GetValue(),
		}
		envs[i] = env
	}
	return envs, nil
}

// GetArgsFromHippoProcessInfo GetArgsFromHippoProcessInfo
func GetArgsFromHippoProcessInfo(p *hippo.ProcessInfo) []string {
	if nil == p.Args {
		return nil
	}
	var args = make([]string, 0, 2*len(p.Args))
	for i := range p.Args {
		if nil != p.Args[i].Key {
			args = append(args, p.Args[i].GetKey())
		}
		if nil != p.Args[i].Value {
			args = append(args, p.Args[i].GetValue())
		}
	}
	return args
}

// GetRoleNameSpace GetRoleNameSpace
func GetRoleNameSpace(admin, quota, name string) string {
	return ""
}

// GetGroupNameSpace GetGroupNameSpace
func GetGroupNameSpace(admin, quota, name string) string {
	return ""
}

func computeControllerShardKey(name string) string {
	hash := fnvHash(name)
	index := hash % carbonv1.MaxControllerShards
	if index < 0 {
		index = -index
	}
	return strconv.Itoa(index)
}

func fnvHash(src string) int {
	tableFnv := fnv.New32a()
	tableFnv.Write([]byte(src))
	result := int(tableFnv.Sum32())
	return result
}

func appendPlatform(labels map[string]string) {
	if "" != platform {
		// 修正拼写错误，重要，不可随意删除
		labels[platformKey] = platform
		labels[C2PlatformKey] = platform
	}
}

func appendPlatformV1(labels map[string]string) {
	if "" != platform {
		labels[C2PlatformKey] = platform
	}
}

// AppendPlatfrom ...
func AppendPlatfrom(labels map[string]string) {
	appendPlatform(labels)
}

func setObjectGroupKeys(object metav1.Object, cluster, app, appChecksum, group, zoneName string) {
	carbonv1.SetClusterName(object, cluster)
	carbonv1.SetAppChecksum(object, appChecksum)
	carbonv1.SetObjectAppRoleName(object, app, "")
	carbonv1.SetObjectGroupHash(object, app, group)
	carbonv1.SetObjectZoneName(object, zoneName)
	carbonv1.SetAnnoGroupName(object, group) // status transfer requires this
	if !features.C2MutableFeatureGate.Enabled(features.HashLabelValue) {
		carbonv1.SetGroupName(object, group)
	}
}

func setObjectRoleKeys(object metav1.Object, cluster, app, appChecksum, group, zoneName, role string) {
	setObjectGroupKeys(object, cluster, app, appChecksum, group, zoneName)
	carbonv1.SetObjectAppRoleName(object, app, role)
	carbonv1.SetObjectRoleHash(object, app, role)
}

const (
	systemPriority = 0
	systemQueue    = "system"
)

func isSystemPriority(role *typespec.RolePlan) bool {
	if nil == role {
		return false
	}
	if nil == role.Version.ResourcePlan.Priority ||
		nil == role.Version.ResourcePlan.Priority.MajorPriority {
		return false
	}
	return systemPriority == *role.Version.ResourcePlan.Priority.MajorPriority
}

func isSystemQueue(role *typespec.RolePlan) bool {
	if nil == role {
		return false
	}
	return nil != role.Version.ResourcePlan.Queue && systemQueue == *role.Version.ResourcePlan.Queue
}

func isSysRole(role *typespec.RolePlan) bool {
	if nil == role {
		return false
	}
	if isSystemQueue(role) && isSystemPriority(role) {
		return true
	}
	return false
}

func transferMaxUnavailable(minHealthCapacity int32) *intstr.IntOrString {
	maxUnavailable := intstr.FromString(fmt.Sprintf("%d%%", 100-int(minHealthCapacity)))
	return &maxUnavailable
}

func checkRollingsetValidation(rollingset *carbonv1.RollingSet, template *carbonv1.HippoPodTemplate) error {
	// 容器重复env
	for i := range template.Spec.Containers {
		var envKeys = map[string]bool{}
		for j := range template.Spec.Containers[i].Env {
			name := template.Spec.Containers[i].Env[j].Name
			errs := validation.IsEnvVarName(name)
			if nil != errs {
				err := fmt.Errorf("%s: invalid envkey: %s , %v", rollingset.Name, name, errs)
				return err
			}
			if checkDupEnv {
				exist := envKeys[name]
				if exist {
					err := fmt.Errorf("%s: duplicate envkey: %s", rollingset.Name, name)
					return err
				}
				envKeys[name] = true
			}
		}
	}
	return nil
}

// IsMasterFrameworkMode 为了兼容masterFramework中特殊且待废弃的逻辑, 对这部分服务提供的特殊标识
// 1) 用于bs监控(?); 2) 用于兼容master framework 进程instanceId 问题
func IsMasterFrameworkMode(rolePlan *typespec.RolePlan) bool {
	if nil == rolePlan || nil == rolePlan.Global.Properties {
		return false
	}
	return getPropKeyBool(*rolePlan.Global.Properties, propIsMasterFrameworkModeKey, false)
}
