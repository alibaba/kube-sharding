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
	"fmt"
	"regexp"
	"strconv"
	"strings"

	mapset "github.com/deckarep/golang-set"

	"github.com/alibaba/kube-sharding/pkg/rollalgorithm"

	"github.com/alibaba/kube-sharding/common"
	"github.com/alibaba/kube-sharding/common/features"
	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	"github.com/alibaba/kube-sharding/pkg/memkube/mem"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/validation"
	labelsutil "k8s.io/kubernetes/pkg/util/labels"
)

// PatchMap merge mainLabels and keys of sub labels to new labels
func PatchMap(mainMaps map[string]string, subMaps map[string]string, keys ...string) map[string]string {
	// Clone.
	newMaps := map[string]string{}
	for key, value := range mainMaps {
		newMaps[key] = value
	}
	for _, key := range keys {
		if value, ok := subMaps[key]; ok && "" != value {
			newMaps[key] = value
		}
	}
	return newMaps
}

// MergeMap merge all kvs to target map
func MergeMap(from map[string]string, to map[string]string) map[string]string {
	if to == nil {
		return nil
	}
	for k, v := range from {
		to[k] = v
	}
	return to
}

// CloneMap Clone the given map and returns a new cloned map
func CloneMap(maps map[string]string) map[string]string {
	// Clone.
	newLabels := map[string]string{}
	for key, value := range maps {
		newLabels[key] = value
	}
	return newLabels
}

// LabelValueMaxLength LabelValueMaxLength
const LabelValueMaxLength int = 63

// MakeNameLegal MakeNameLegal
func MakeNameLegal(name string) string {
	var reg *regexp.Regexp = regexp.MustCompile(`[^a-z0-9-.]`)
	name = strings.ToLower(name)
	name = reg.ReplaceAllString(name, "-")
	if strings.HasPrefix(name, "-") {
		name = "p" + name
	}
	if strings.HasSuffix(name, "-") || strings.HasSuffix(name, ".") {
		name = name + "s"
	}
	var err error
	name, err = ResourceNameHash(name)
	if err != nil { // impossible ?
		panic("hash name error:" + err.Error())
	}
	return name
}

// ResourceNameHash hash resource name, NOTE the name is valid by k8s name conversation
// see https://kubernetes.io/docs/concepts/overview/working-with-objects/names/
func ResourceNameHash(name string) (string, error) {
	var err error
	if len(name) > LabelValueMaxLength {
		name, err = utils.SignatureWithMD5(name)
		if nil != err {
			return "", err
		}
	}
	return name, nil
}

// LabelValueHash hash the label value if it validates failed
func LabelValueHash(v string, enable bool) string {
	if !enable {
		return v
	}
	var err error
	// For compatible reason, we may use a forked apimachinery library which modified the `LabelValueMaxLength`
	if len(v) > LabelValueMaxLength || validation.IsValidLabelValue(v) != nil {
		v, err = utils.SignatureWithMD5(v)
		if err != nil { // impossible ?
			panic("hash label value error:" + err.Error())
		}
	}
	return v
}

// AddUniqueKey AddUniqueKey
func AddUniqueKey(object metav1.Object, key, value string) error {
	setAnnotation(object, key, value)
	labels := object.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}
	AddUniqueLabelHashKey(labels, key, value)
	object.SetLabels(labels)
	return nil
}

// AddSubrsUniqueKey AddSubrsUniqueKey
func AddSubrsUniqueKey(object metav1.Object, value string) error {
	return AddUniqueKey(object, DefaultSubRSLabelKey, value)
}

// AddSubrsShortUniqueKey AddSubrsShortUniqueKey
func AddSubrsShortUniqueKey(object metav1.Object, value string) {
	setAnnotation(object, DefaultShortSubRSLabelKey, value)
}

// SetMigratedFromSubrs SetMigratedFromSubrs
func SetMigratedFromSubrs(object metav1.Object, value string) {
	setAnnotation(object, LabelKeyMigratedFromSubrs, value)
}

func SetBufferSwapRecord(object metav1.Object, value string) {
	setAnnotation(object, AnnotationBufferSwapRecord, value)
}

func GetBufferSwapRecord(object metav1.Object) string {
	return object.GetAnnotations()[AnnotationBufferSwapRecord]
}

func RemoveLabelsAnnos(meta metav1.Object, prefix []string) {
	for i := range prefix {
		for k := range meta.GetLabels() {
			if strings.HasPrefix(k, prefix[i]) {
				delete(meta.GetLabels(), k)
			}
		}
		for k := range meta.GetAnnotations() {
			if strings.HasPrefix(k, prefix[i]) {
				delete(meta.GetAnnotations(), k)
			}
		}
	}
}

func SwapLabelsAnnos(meta1, meta2 metav1.Object, prefix []string) {
	temp := &metav1.ObjectMeta{}
	CopyLabelsAnnos(meta1, temp, prefix)
	RemoveLabelsAnnos(meta1, prefix)
	CopyLabelsAnnos(meta2, meta1, prefix)
	RemoveLabelsAnnos(meta2, prefix)
	CopyLabelsAnnos(temp, meta2, prefix)
}

func CopyLabelsAnnos(src metav1.Object, target metav1.Object, prefix []string) {
	if target.GetLabels() == nil {
		target.SetLabels(map[string]string{})
	}
	if target.GetAnnotations() == nil {
		target.SetAnnotations(map[string]string{})
	}
	for k, v := range src.GetLabels() {
		for i := range prefix {
			if strings.HasPrefix(k, prefix[i]) {
				target.GetLabels()[k] = v
			}
		}
	}
	for k, v := range src.GetAnnotations() {
		for i := range prefix {
			if strings.HasPrefix(k, prefix[i]) {
				target.GetAnnotations()[k] = v
			}
		}
	}
}

// GetMigratedFromSubrs GetMigratedFromSubrs
func GetMigratedFromSubrs(object metav1.Object) string {
	return GetLabelAnnoValue(object, LabelKeyMigratedFromSubrs)
}

// GetSubrsShortUniqueKey GetSubrsShortUniqueKey
func GetSubrsShortUniqueKey(object metav1.Object) string {
	return GetLabelAnnoValue(object, DefaultShortSubRSLabelKey)
}

// AddUniqueLabelHashKey AddUniqueLabelHashKey
func AddUniqueLabelHashKey(labels map[string]string, key, value string) error {
	if nil == labels {
		return fmt.Errorf("can't add unique keys to nil map")
	}
	hash, err := ResourceNameHash(value)
	if nil != err {
		return err
	}
	labelsutil.AddLabel(labels, key, hash)
	return nil
}

// AddSsUniqueLabelHashKey AddSsUniqueLabelHashKey
func AddSsUniqueLabelHashKey(labels map[string]string, rsName string) error {
	return AddUniqueLabelHashKey(labels, DefaultSlotsetUniqueLabelKey, rsName)
}

// AddRsUniqueLabelHashKey AddRsUniqueLabelHashKey
func AddRsUniqueLabelHashKey(labels map[string]string, rsName string) error {
	return AddUniqueLabelHashKey(labels, DefaultRollingsetUniqueLabelKey, rsName)
}

// AddReplicaUniqueLabelHashKey AddReplicaUniqueLabelHashKey
func AddReplicaUniqueLabelHashKey(labels map[string]string, replicaName string) error {
	return AddUniqueLabelHashKey(labels, DefaultReplicaUniqueLabelKey, replicaName)
}

// AddGangUniqueLabelHashKey AddGangUniqueLabelHashKey
func AddGangUniqueLabelHashKey(labels map[string]string, replicaName string) error {
	return AddUniqueLabelHashKey(labels, DefaultGangUniqueLabelKey, replicaName)
}

// AddGangPartLabelHashKey AddGangPartLabelHashKey
func AddGangPartLabelHashKey(labels map[string]string, partName string) error {
	return AddUniqueLabelHashKey(labels, LabelGangPartName, partName)
}

// AddWorkerUniqueLabelHashKey AddWorkerUniqueLabelHashKey
func AddWorkerUniqueLabelHashKey(labels map[string]string, workerName string) error {
	return AddUniqueLabelHashKey(labels, DefaultWorkernodeUniqueLabelKey, workerName)
}

// AddGroupUniqueLabelHashKey AddGroupUniqueLabelHashKey
func AddGroupUniqueLabelHashKey(labels map[string]string, groupName string) error {
	return AddUniqueLabelHashKey(labels, DefaultShardGroupUniqueLabelKey, groupName)
}

// AddSelectorHashKey AddSelectorHashKey
func AddSelectorHashKey(selector *metav1.LabelSelector, key, value string) error {
	if nil == selector {
		return fmt.Errorf("can't add unique keys to nil selector")
	}
	if "" == value {
		return nil
	}
	hash, err := ResourceNameHash(value)
	if nil != err {
		return err
	}
	labelsutil.AddLabelToSelector(selector, key, hash)
	return nil
}

// AddRsSelectorHashKey AddRsSelectorHashKey
func AddRsSelectorHashKey(selector *metav1.LabelSelector, rsName string) error {
	return AddSelectorHashKey(selector, DefaultRollingsetUniqueLabelKey, rsName)
}

// AddSsSelectorHashKey AddSsSelectorHashKey
func AddSsSelectorHashKey(selector *metav1.LabelSelector, rsName string) error {
	return AddSelectorHashKey(selector, DefaultSlotsetUniqueLabelKey, rsName)
}

// AddReplicaSelectorHashKey AddReplicaSelectorHashKey
func AddReplicaSelectorHashKey(selector *metav1.LabelSelector, replicaName string) error {
	return AddSelectorHashKey(selector, DefaultReplicaUniqueLabelKey, replicaName)
}

// AddWorkerSelectorHashKey AddWorkerSelectorHashKey
func AddWorkerSelectorHashKey(selector *metav1.LabelSelector, workerName string) error {
	return AddSelectorHashKey(selector, DefaultWorkernodeUniqueLabelKey, workerName)
}

// AddGroupSelectorHashKey AddGroupSelectorHashKey
func AddGroupSelectorHashKey(selector *metav1.LabelSelector, groupName string) error {
	return AddSelectorHashKey(selector, DefaultShardGroupUniqueLabelKey, groupName)
}

// AddGangPartSelectorHashKey AddGangPartSelectorHashKey
func AddGangPartSelectorHashKey(selector *metav1.LabelSelector, partName string) error {
	return AddSelectorHashKey(selector, LabelGangPartName, partName)
}

// CopyWorkerAndReplicaLabels CopyWorkerAndReplicaLabels
func CopyWorkerAndReplicaLabels(mainMaps map[string]string, subMaps map[string]string) map[string]string {
	return PatchMap(mainMaps, subMaps, DefaultGangUniqueLabelKey, DefaultReplicaUniqueLabelKey, DefaultWorkernodeUniqueLabelKey, WorkerRoleKey)
}

// GetRsStringSelector GetRsStringSelector
func GetRsStringSelector(rsName string) string {
	hash, err := ResourceNameHash(rsName)
	if err == nil {
		rsName = hash
	}
	return fmt.Sprintf("%s=%s", DefaultRollingsetUniqueLabelKey, rsName)
}

// GetRsSelector GetRsSelector
func GetRsSelector(rsName string) labels.Selector {
	var selector metav1.LabelSelector
	AddRsSelectorHashKey(&selector, rsName)
	labelsSelector, _ := metav1.LabelSelectorAsSelector(&selector)
	return labelsSelector
}

// CreateWorkerLabels CreateWorkerLabels
func CreateWorkerLabels(labels map[string]string, replicaName, workerName string) map[string]string {
	newLabels := CloneMap(labels)
	AddReplicaUniqueLabelHashKey(newLabels, replicaName)
	AddWorkerUniqueLabelHashKey(newLabels, workerName)
	newLabels[WorkerRoleKey] = CurrentWorkerKey
	return newLabels
}

// CreateWorkerSelector CreateWorkerSelector
func CreateWorkerSelector(selector *metav1.LabelSelector, replicaName, workerName string) *metav1.LabelSelector {
	newSelector := selector.DeepCopy()
	AddReplicaSelectorHashKey(newSelector, replicaName)
	AddWorkerSelectorHashKey(newSelector, workerName)
	return newSelector
}

// CloneSelector Clones the given selector and returns a new cloned selector
func CloneSelector(selector *metav1.LabelSelector) *metav1.LabelSelector {
	// Clone.
	newSelector := new(metav1.LabelSelector)

	// TODO(madhusudancs): Check if you can use deepCopy_extensions_LabelSelector here.
	newSelector.MatchLabels = make(map[string]string)
	if selector.MatchLabels != nil {
		for key, val := range selector.MatchLabels {
			newSelector.MatchLabels[key] = val
		}
	}

	if selector.MatchExpressions != nil {
		newMExps := make([]metav1.LabelSelectorRequirement, len(selector.MatchExpressions))
		for i, me := range selector.MatchExpressions {
			newMExps[i].Key = me.Key
			newMExps[i].Operator = me.Operator
			if me.Values != nil {
				newMExps[i].Values = make([]string, len(me.Values))
				copy(newMExps[i].Values, me.Values)
			} else {
				newMExps[i].Values = nil
			}
		}
		newSelector.MatchExpressions = newMExps
	} else {
		newSelector.MatchExpressions = nil
	}

	return newSelector
}

// GetReplicaID GetReplicaID
func GetReplicaID(object metav1.Object) string {
	labels := object.GetLabels()

	return labels[DefaultReplicaUniqueLabelKey]
}

// GetGangID GetGangID
func GetGangID(object metav1.Object) string {
	labels := object.GetLabels()

	return labels[DefaultGangUniqueLabelKey]
}

// GetRsID GetRsID
func GetRsID(object metav1.Object) string {
	labels := object.GetLabels()
	return labels[DefaultRollingsetUniqueLabelKey]
}

func GetRsName(worker *WorkerNode) string {
	if worker == nil || worker.OwnerReferences == nil || len(worker.OwnerReferences) != 1 ||
		worker.OwnerReferences[0].Kind != "RollingSet" {
		return ""
	}
	return worker.OwnerReferences[0].Name
}

// GetSsID GetSsID
func GetSsID(object metav1.Object) string {
	return GetLabelAnnoValue(object, DefaultSlotsetUniqueLabelKey)
}

// SetSsID SetSsID
func SetSsID(object metav1.Object, id string) {
	AddSsUniqueLabelHashKey(object.GetLabels(), id)
	object.GetAnnotations()[DefaultSlotsetUniqueLabelKey] = id
}

// GetCarbonJobID GetCarbonJobID
func GetCarbonJobID(object metav1.Object) string {
	labels := object.GetLabels()
	return labels[LabelKeyCarbonJobName]
}

// GetReportRsID 用来汇报controller指标中rollingset key的value
func GetReportRsID(object metav1.Object) string {
	value := GetRsID(object)
	if "" != value {
		return value
	}
	value = GetCarbonJobID(object)
	if "" != value {
		return value
	}
	return "none"
}

// GetShardGroupID GetShardGroupID
func GetShardGroupID(object metav1.Object) string {
	labels := object.GetLabels()
	return labels[DefaultShardGroupUniqueLabelKey]
}

// GetClusterName GetClusterName
func GetClusterName(object metav1.Object) string {
	labels := object.GetLabels()
	return labels[LabelKeyClusterName]
}

// GetPodVersion GetPodVersion
func GetPodVersion(object metav1.Object) string {
	labels := object.GetLabels()
	if labels == nil {
		return ""
	}
	return labels[LabelKeyPodVersion]
}

// GetExclusiveMode GetExclusiveMode
func GetExclusiveMode(object metav1.Object) string {
	exclusiveMode := GetLabelValue(object, LabelKeyExclusiveMode)
	if "" == exclusiveMode {
		exclusiveMode = DefaultExclusiveMode
	}
	return exclusiveMode
}

// SetPodVersion SetPodVersion
func SetPodVersion(object metav1.Object, podVersion string) {
	setLabel(object, LabelKeyPodVersion, podVersion)
}

// GetLabelValue GetLabelValue
func GetLabelValue(object metav1.Object, key string) string {
	if common.IsInterfaceNil(object) {
		return ""
	}
	lbls := object.GetLabels()
	if lbls != nil && lbls[key] != "" {
		return lbls[key]
	}
	return ""
}

func GetAnnotationValue(object metav1.Object, key string) string {
	if common.IsInterfaceNil(object) {
		return ""
	}
	annos := object.GetAnnotations()
	if annos != nil && annos[key] != "" {
		return annos[key]
	}
	return ""
}

// GetLabelAnnoValue see issue #101090
func GetLabelAnnoValue(object metav1.Object, key string) string {
	if common.IsInterfaceNil(object) {
		return ""
	}
	annos := object.GetAnnotations()
	if annos != nil && annos[key] != "" {
		return annos[key]
	}
	lbls := object.GetLabels()
	if lbls != nil && lbls[key] != "" {
		return lbls[key]
	}
	return ""
}

// GetReplaceNodeVersion GetReplaceNodeVersion
func GetReplaceNodeVersion(object metav1.Object) string {
	return GetLabelAnnoValue(object, LabelKeyReplaceNodeVersion)
}

// GetAppName GetAppName
func GetAppName(object metav1.Object) string {
	return GetLabelAnnoValue(object, LabelKeyAppName)
}

// GetAppChecksum returns appChecksum label value.
func GetAppChecksum(object metav1.Object) string {
	labels := object.GetLabels()
	return labels[LabelKeyAppChecksum]
}

// GetGroupName GetGroupName
func GetGroupName(object metav1.Object) string {
	return GetLabelAnnoValue(object, LabelKeyGroupName)
}

// GetRoleName GetRoleName
func GetRoleName(object metav1.Object) string {
	return GetLabelAnnoValue(object, LabelKeyRoleName)
}

// GetGangPartName GetGangPartName
func GetGangPartName(object metav1.Object) string {
	return GetLabelAnnoValue(object, LabelGangPartName)
}

// IsGangMainPart IsGangMainPart
func IsGangMainPart(object metav1.Object) bool {
	return GetLabelAnnoValue(object, LabelGangMainPart) == "true"
}

// GetGangPlan GetGangPlan
func GetGangPlan(object metav1.Object) string {
	return GetLabelAnnoValue(object, AnnoGangPlan)
}

// GetGangInfo GetGangInfo
func GetGangInfo(object metav1.Object) string {
	return GetLabelAnnoValue(object, BizDetailKeyGangInfo)
}

// GetZoneName GetZoneName
func GetZoneName(object metav1.Object) string {
	return GetLabelAnnoValue(object, LabelKeyZoneName)
}

// GetRoleNameAndName GetRoleNameAndName
func GetRoleNameAndName(object metav1.Object) string {
	labels := object.GetLabels()
	return utils.AppendString(object.GetName(), "::", labels[LabelKeyRoleName])
}

// GetC2RoleName GetC2RoleName
func GetC2RoleName(object metav1.Object) string {
	groupName := GetGroupName(object)
	roleName := GetRoleName(object)
	return strings.TrimPrefix(roleName, utils.AppendString(groupName, "."))
}

func setLabel(object metav1.Object, key, value string) {
	if value == "" {
		return
	}
	labels := object.GetLabels()
	labels = labelsutil.AddLabel(labels, key, value)
	object.SetLabels(labels)
}

func setAnnotation(object metav1.Object, key, value string) {
	if value == "" {
		return
	}
	annos := object.GetAnnotations()
	if annos == nil {
		annos = make(map[string]string)
	}
	annos[key] = value
	object.SetAnnotations(annos)
}

func removeAnnotation(object metav1.Object, key string) {
	if key == "" {
		return
	}
	annos := object.GetAnnotations()
	if annos == nil {
		annos = make(map[string]string)
	}
	delete(annos, key)
	object.SetAnnotations(annos)
}

// SetClusterName SetClusterName
func SetClusterName(object metav1.Object, clusterName string) {
	setLabel(object, LabelKeyClusterName, clusterName)
}

// SetAppName SetAppName
func SetAppName(object metav1.Object, appName string) {
	setLabel(object, LabelKeyAppName, appName)
}

// SetAppChecksum set app/admin checksum to a resource.
// NOTE: Its admin-manager responsibility to generate app checksum when it deploy admin on k8s.
func SetAppChecksum(object metav1.Object, appChecksum string) {
	setLabel(object, LabelKeyAppChecksum, appChecksum)
}

// SetGroupName SetGroupName
func SetGroupName(object metav1.Object, groupName string) {
	setLabel(object, LabelKeyGroupName, groupName)
}

// SetAnnoGroupName set group name
func SetAnnoGroupName(object metav1.Object, groupName string) {
	setAnnotation(object, LabelKeyGroupName, groupName)
}

// SetObjectAppRoleName set app & role to this object
// App & Role are used by hippo for 2 purposes: a) group by ops; b) create working dir
func SetObjectAppRoleName(object metav1.Object, app, role string) {
	setAnnotation(object, LabelKeyAppName, app)
	setAnnotation(object, LabelKeyRoleName, role)
	if !features.C2MutableFeatureGate.Enabled(features.HashLabelValue) { // old version, set value on label
		setLabel(object, LabelKeyAppName, app)
		setLabel(object, LabelKeyRoleName, role)
	}
}

func SetObjectAppRoleNameHash(object metav1.Object, app, role, group string) {
	setAnnotation(object, LabelKeyAppName, app)
	setAnnotation(object, LabelKeyRoleName, role)
	setAnnotation(object, LabelKeyGroupName, group)
	if features.C2MutableFeatureGate.Enabled(features.HashLabelValue) {
		setLabel(object, LabelKeyAppNameHash, LabelValueHash(app, true))
		setLabel(object, LabelKeyRoleNameHash, LabelValueHash(role, true))
		setLabel(object, LabelKeyGroupNameHash, LabelValueHash(group, true))
	}
}

// SetRoleName set  role to this object
func SetRoleName(object metav1.Object, role string) {
	setAnnotation(object, LabelKeyRoleName, role)
	if !features.C2MutableFeatureGate.Enabled(features.HashLabelValue) { // old version, set value on label
		setLabel(object, LabelKeyRoleName, role)
	}
}

// SetGangPartName SetGangPartName
func SetGangPartName(object metav1.Object, part string) {
	setAnnotation(object, LabelGangPartName, part)
	setLabel(object, LabelGangPartName, LabelValueHash(part, true))
}

// SetGangPlan SetGangPlan
func SetGangPlan(object metav1.Object, plan string) {
	setAnnotation(object, AnnoGangPlan, plan)
}

// SetGangInfo SetGangInfo
func SetGangInfo(object metav1.Object, info string) {
	setAnnotation(object, BizDetailKeyGangInfo, info)
}

// SetGangMainPart SetGangMainPart
func SetGangMainPart(object metav1.Object) {
	setAnnotation(object, LabelGangMainPart, "true")
}

// SetObjectAppName set appname for to this object.
func SetObjectAppName(object metav1.Object, app string) {
	setAnnotation(object, LabelKeyAppName, app)
	if !features.C2MutableFeatureGate.Enabled(features.HashLabelValue) { // old version, set value on label
		setLabel(object, LabelKeyAppName, app)
	}
}

// SetObjectZoneName set appname for to this object.
func SetObjectZoneName(object metav1.Object, zone string) {
	if zone == "" {
		return
	}
	setAnnotation(object, LabelKeyZoneName, zone)
	setLabel(object, LabelKeyZoneNameHash, LabelValueHash(zone, true))
}

// SetObjectValidLabel set label if the value is valid, otherwise write the raw value in the annotation
func SetObjectValidLabel(object metav1.Object, k, v string) {
	hv := LabelValueHash(v, true)
	setLabel(object, k, hv)
	if hv != v {
		setAnnotation(object, k, v)
	}
}

// SetDefaultObjectValidLabel set label if the value is valid and not exist, otherwise write the raw value in the annotation if not exist
func SetDefaultObjectValidLabel(meta metav1.Object, k, v string) {
	hv := LabelValueHash(v, true)
	if meta.GetLabels() == nil {
		meta.SetLabels(map[string]string{})
	}
	if meta.GetAnnotations() == nil {
		meta.SetAnnotations(map[string]string{})
	}
	oldVal := SetIfEmptyAndReturnOld(meta.GetLabels(), k, hv)
	if oldVal == "" && hv != v {
		SetIfEmptyAndReturnOld(meta.GetAnnotations(), k, v)
	}
}

// SetMapValidValue set v to kv1 if it's valid, otherwise hash it and copy raw value to kv2
func SetMapValidValue(k, v string, kv1, kv2 map[string]string) {
	hv := LabelValueHash(v, true)
	kv1[k] = hv
	if hv != v {
		kv2[k] = v
	}
}

// SetObjectRoleHash role hash is used for role selection
func SetObjectRoleHash(object metav1.Object, app, role string) {
	setLabel(object, LabelKeyRoleNameHash, LabelValueHash(role, true))
	setLabel(object, LabelKeyAppNameHash, LabelValueHash(app, true))
}

// SetObjectGroupHash group hash is used for group selection
func SetObjectGroupHash(object metav1.Object, app, group string) {
	setLabel(object, LabelKeyGroupNameHash, LabelValueHash(group, true))
	setLabel(object, LabelKeyAppNameHash, LabelValueHash(app, true))
}

// SetExclusiveLableAnno SetExclusiveLableAnno
func SetExclusiveLableAnno(object metav1.Object, exclusive string) {
	setLabel(object, LabelKeyExlusive, LabelValueHash(exclusive, true))
	setAnnotation(object, LabelKeyExlusive, exclusive)
}

// GetExclusiveLableAnno GetExclusiveLableAnno
func GetExclusiveLableAnno(object metav1.Object) string {
	if common.IsInterfaceNil(object) {
		return ""
	}
	return GetLabelAnnoValue(object, LabelKeyExlusive)
}

// GetQuotaID GetQuotaID
func GetQuotaID(object metav1.Object) string {
	labels := object.GetLabels()
	return labels[LabelKeyQuotaGroupID]
}

// SetQuotaID SetQuotaID
func SetQuotaID(object metav1.Object, quotaID string) {
	setLabel(object, LabelKeyQuotaGroupID, quotaID)
}

// SetBackupPod SetBackupPod
func SetBackupPod(object metav1.Object, uid string) {
	setLabel(object, LabelKeyBackupPod, uid)
}

// GetEviction GetEviction
func GetEviction(object metav1.Object) string {
	labels := object.GetLabels()
	return labels["alibabacloud.com/eviction"]
}

// SetEviction SetEviction
func SetEviction(object metav1.Object) {
	setLabel(object, "alibabacloud.com/eviction", "true")
}

// SetHippoLabelSelector SetHippoLabelSelector
func SetHippoLabelSelector(selector *metav1.LabelSelector, key string, value string) error {
	if nil == selector {
		return fmt.Errorf("can't add unique keys to nil selector")
	}
	if value != "" {
		labelsutil.AddLabelToSelector(selector, key, value)
	}
	return nil
}

// SetClusterSelector SetClusterSelector
func SetClusterSelector(selector *metav1.LabelSelector, clusterName string) error {
	return SetHippoLabelSelector(selector, LabelKeyClusterName, clusterName)
}

// SetAppSelector SetAppSelector
func SetAppSelector(selector *metav1.LabelSelector, appName string) error {
	if features.C2MutableFeatureGate.Enabled(features.HashLabelValue) {
		return SetHippoLabelSelector(selector, LabelKeyAppNameHash, LabelValueHash(appName, true))
	}
	return SetHippoLabelSelector(selector, LabelKeyAppName, appName)
}

// SetHashAppSelector use LabelKeyAppNameHash
func SetHashAppSelector(selector *metav1.LabelSelector, appName string) error {
	return SetHippoLabelSelector(selector, LabelKeyAppNameHash, LabelValueHash(appName, true))
}

// SetNormalAppSelector use LabelKeyAppName
func SetNormalAppSelector(selector *metav1.LabelSelector, appName string) error {
	return SetHippoLabelSelector(selector, LabelKeyAppName, appName)
}

// SetGroupSelector SetGroupSelector
func SetGroupSelector(selector *metav1.LabelSelector, groupName string) error {
	if features.C2MutableFeatureGate.Enabled(features.HashLabelValue) {
		return SetHippoLabelSelector(selector, LabelKeyGroupNameHash, LabelValueHash(groupName, true))
	}
	return SetHippoLabelSelector(selector, LabelKeyGroupName, groupName)
}

// SetZoneSelector SetZoneSelector
func SetZoneSelector(selector *metav1.LabelSelector, zoneName string) error {
	if features.C2MutableFeatureGate.Enabled(features.HashLabelValue) {
		return SetHippoLabelSelector(selector, LabelKeyZoneNameHash, LabelValueHash(zoneName, true))
	}
	return SetHippoLabelSelector(selector, LabelKeyZoneNameHash, zoneName)
}

// SetRoleSelector SetRoleSelector
func SetRoleSelector(selector *metav1.LabelSelector, roleName string) error {
	if features.C2MutableFeatureGate.Enabled(features.HashLabelValue) {
		return SetHippoLabelSelector(selector, LabelKeyRoleNameHash, LabelValueHash(roleName, true))
	}
	return SetHippoLabelSelector(selector, LabelKeyRoleName, roleName)
}

// SetGroupsSelector SetGroupsSelector
func SetGroupsSelector(selector *metav1.LabelSelector, groupIDs ...string) error {
	key := LabelKeyGroupName
	if features.C2MutableFeatureGate.Enabled(features.HashLabelValue) {
		key = LabelKeyGroupNameHash
	}
	if len(groupIDs) == 1 {
		SetGroupSelector(selector, groupIDs[0])
	} else if len(groupIDs) > 1 {
		// make hash first
		for i := range groupIDs {
			groupIDs[i] = LabelValueHash(groupIDs[i], features.C2MutableFeatureGate.Enabled(features.HashLabelValue))
		}
		// multi groupIDs
		if nil == selector.MatchExpressions {
			selector.MatchExpressions = []metav1.LabelSelectorRequirement{}
		}
		if nil != groupIDs && 0 < len(groupIDs) {
			selector.MatchExpressions = append(
				selector.MatchExpressions,
				metav1.LabelSelectorRequirement{
					Key:      key,
					Operator: metav1.LabelSelectorOpIn,
					Values:   groupIDs,
				},
			)
		}
	}
	return nil
}

// SetRolesSelector SetRolesSelector
func SetRolesSelector(selector *metav1.LabelSelector, roleNames ...string) error {
	key := LabelKeyRoleName
	if features.C2MutableFeatureGate.Enabled(features.HashLabelValue) {
		key = LabelKeyRoleNameHash
	}
	if len(roleNames) == 1 {
		SetRoleSelector(selector, roleNames[0])
	} else if len(roleNames) > 1 {
		// make hash first
		for i := range roleNames {
			roleNames[i] = LabelValueHash(roleNames[i], features.C2MutableFeatureGate.Enabled(features.HashLabelValue))
		}
		// multi roleNames
		if nil == selector.MatchExpressions {
			selector.MatchExpressions = []metav1.LabelSelectorRequirement{}
		}
		if nil != roleNames && 0 < len(roleNames) {
			selector.MatchExpressions = append(
				selector.MatchExpressions,
				metav1.LabelSelectorRequirement{
					Key:      key,
					Operator: metav1.LabelSelectorOpIn,
					Values:   roleNames,
				},
			)
		}
	}
	return nil
}

// SetPodVersionSelector setPodVersionSelector
func SetPodVersionSelector(selector *metav1.LabelSelector, podVersion string) error {
	return SetHippoLabelSelector(selector, LabelKeyPodVersion, podVersion)
}

// SetPodVersionsSelector SetPodVersionsSelector
func SetPodVersionsSelector(selector *metav1.LabelSelector, podVersions ...string) error {
	if len(podVersions) == 1 {
		SetGroupSelector(selector, podVersions[0])
	} else if len(podVersions) > 1 {
		// multi groupIDs
		if nil == selector.MatchExpressions {
			selector.MatchExpressions = []metav1.LabelSelectorRequirement{}
		}
		if nil != podVersions && 0 < len(podVersions) {
			selector.MatchExpressions = append(
				selector.MatchExpressions,
				metav1.LabelSelectorRequirement{
					Key:      LabelKeyPodVersion,
					Operator: metav1.LabelSelectorOpIn,
					Values:   podVersions,
				},
			)
		}
	}
	return nil
}

// GetObjectScaleScopes ..
func GetObjectScaleScopes(object metav1.Object, extends ...string) []string {
	tags := []string{GetClusterName(object), GetAppName(object), GetGroupName(object), GetRoleName(object)}
	tags = append(tags, extends...)
	return fixEmptyTag(tags)
}

func isFromRollingSet(obj metav1.Object) bool {
	owners := obj.GetOwnerReferences()
	for i := range owners {
		if owners[i].Kind == "RollingSet" {
			return true
		}
	}
	value := GetRsID(obj)
	return value != ""
}

// GetSubObjectBaseScopes GetSubObjectBaseScopes
func GetSubObjectBaseScopes(object metav1.Object, extends ...string) []string {
	var rsId, groupName, roleName string
	if isFromRollingSet(object) {
		rsId = GetReportRsID(object)
		groupName = GetGroupName(object)
		roleName = GetRoleName(object)
	}
	tags := []string{GetClusterName(object), GetAppName(object), rsId, groupName, roleName}
	return fixEmptyTag(tags)
}

// GetSubObjectExtendScopes GetSubObjectExtendScopes
func GetSubObjectExtendScopes(object metav1.Object, extends ...string) []string {
	return append(GetSubObjectBaseScopes(object), extends...)
}

// GetObjectBaseScopes GetObjectBaseScopes
func GetObjectBaseScopes(object metav1.Object, extends ...string) []string {
	tags := []string{GetClusterName(object), GetAppName(object), object.GetName(), GetGroupName(object), GetRoleName(object)}
	return fixEmptyTag(tags)
}

// GetObjectExtendScopes GetObjectExtendScopes
func GetObjectExtendScopes(object metav1.Object, extends ...string) []string {
	return append(GetObjectBaseScopes(object), extends...)
}

func fixEmptyTag(tags []string) []string {
	for i := range tags {
		if "" == tags[i] {
			tags[i] = "unknown"
		}
	}
	return tags
}

// SetLabelsDefaultValue SetLabelsDefaultValue
func SetLabelsDefaultValue(key, value string, meta *metav1.ObjectMeta) {
	if "" != value && "" == meta.Labels[key] {
		if meta.Labels == nil {
			meta.Labels = map[string]string{}
		}
		meta.Labels[key] = value
	}
}

func SetIfEmptyAndReturnOld(m map[string]string, key, val string) string {
	if m == nil {
		return ""
	}
	oldVal := m[key]
	if val != "" && oldVal == "" {
		m[key] = val
	}
	return oldVal
}

// SetAntMetas SetAntMetas
func SetAntMetas(meta *metav1.ObjectMeta) {
	SetLabelsDefaultValue("sigma.ali/site", defaultMetaSite, meta)
	SetLabelsDefaultValue("meta.k8s.alipay.com/zone", defaultMetaZone, meta)
	SetLabelsDefaultValue(LabelKeyInstanceGroup, defaultMetaInstanceGroup, meta)
	SetLabelsDefaultValue(LabelKeySigmaAppName, defaultMetaAppName, meta)
	SetLabelsDefaultValue(LabelKeySigmaDeplyUnit, defaultDeployUnit, meta)
}

// SetDefaultPodVersion SetDefaultPodVersion
func SetDefaultPodVersion(meta *metav1.ObjectMeta) {
	if nil == meta.Labels {
		meta.Labels = map[string]string{}
	}

	SetLabelsDefaultValue(LabelKeyPodVersion, defaultPodVersion, meta)
	setLabel(meta, LabelKeyPodVersion, forcePodVersion)
}

// SetDefaultVipPodVersion SetDefaultVipPodVersion
func SetDefaultVipPodVersion(meta *metav1.ObjectMeta) {
	if nil == meta.Labels {
		meta.Labels = map[string]string{}
	}
	if defaultVipPodVersion == "" {
		defaultVipPodVersion = defaultPodVersion
	}

	SetLabelsDefaultValue(LabelKeyPodVersion, defaultVipPodVersion, meta)
	setLabel(meta, LabelKeyPodVersion, forcePodVersion)
}

// GetInstanceGroup GetInstanceGroup
func GetInstanceGroup(object metav1.Object) string {
	return getMultyLabels(object, LabelKeyInstanceGroup, LabelKeyFedInstanceGroup)
}

// GetAppUnit GetAppUnit
func GetAppUnit(object metav1.Object) string {
	return getMultyLabels(object, LabelKeyAppUnit, LabelKeyFedAppUnit)
}

// GetAppStage GetAppStage
func GetAppStage(object metav1.Object) string {
	return getMultyLabels(object, LabelKeyAppStage, LabelKeyFedAppStage)
}

func getMultyLabels(object metav1.Object, keys ...string) string {
	labels := object.GetLabels()
	for i := range keys {
		value := labels[keys[i]]
		if value != "" {
			return value
		}
	}
	return ""
}

// InjectMetas InjectMetas
func InjectMetas(meta *metav1.ObjectMeta) {
	SetAntMetas(meta)
	//injectLabels
	injects := parseInjectLabels(injectLabels)
	for k, v := range injects {
		SetLabelsDefaultValue(k, v, meta)
	}
}

func parseInjectLabels(injectLabels string) map[string]string {
	var labels = map[string]string{}
	if "" == injectLabels {
		return labels
	}
	fields := strings.Split(injectLabels, ",")
	for i := range fields {
		items := strings.Split(fields[i], "=")
		if 2 != len(items) {
			continue
		}
		labels[items[0]] = items[1]
	}
	return labels
}

// IsC2Scheduler IsC2Scheduler
func IsC2Scheduler(object metav1.Object) bool {
	if common.IsInterfaceNil(object) {
		return false
	}
	lbls := object.GetLabels()
	if lbls != nil && lbls[LabelKeySchedulerName] == SchedulerNameC2 {
		return true
	}
	return false
}

// SetC2Scheduler SetC2Scheduler
func SetC2Scheduler(object metav1.Object, isC2Scheduler bool) {
	if !isC2Scheduler {
		labels := object.GetLabels()
		if labels != nil {
			delete(labels, LabelKeySchedulerName)
		}
	} else {
		setLabel(object, LabelKeySchedulerName, SchedulerNameC2)
	}
}

// GetSubrs GetSubrs
func GetSubrs(object metav1.Object) string {
	return GetLabelAnnoValue(object, DefaultSubRSLabelKey)
}

// GetWorkerOwnerKey GetWorkerOwnerKey
func GetWorkerOwnerKey(object metav1.Object) string {
	var ownerName string
	namespace := object.GetNamespace()
	subrs := GetSubrs(object)
	if subrs != "" {
		ownerName = subrs
	} else {
		if ownerRef, err := mem.GetControllerOf(object); ownerRef != nil && err == nil {
			ownerName = ownerRef.Name
		}
	}
	return fmt.Sprintf("%s/%s", namespace, ownerName)
}

// GetSSPodOwnerKey GetSSPodOwnerKey
func GetSSPodOwnerKey(object metav1.Object) string {
	var ownerName string
	namespace := object.GetNamespace()
	if ownerRef, err := mem.GetControllerOf(object); ownerRef != nil && err == nil {
		ownerName = ownerRef.Name
	}
	return fmt.Sprintf("%s/%s", namespace, ownerName)
}

func NeedCreatePool(rs *RollingSet, spec *PoolSpec, status *PoolStatus) bool {
	if rs.DeletionTimestamp != nil {
		return false
	}
	if fiberId := GetAppName(rs); fiberId == "" {
		return false
	}
	if spec == nil {
		return status != nil && status.Status == PoolStatusWaiting
	}
	return status == nil || status.Status == PoolStatusWaiting
}

func NeedDeletePool(rs *RollingSet, spec *PoolSpec, status *PoolStatus) bool {
	if rs.DeletionTimestamp == nil {
		return false
	}
	if fiberId := GetAppName(rs); fiberId == "" {
		return false
	}
	if spec == nil {
		return status != nil && status.Status == PoolStatusCreated
	}
	return status == nil || status.Status == PoolStatusCreated
}

func NeedUpdatePool(rs *RollingSet, spec *PoolSpec, status *PoolStatus) bool {
	if rs.DeletionTimestamp != nil {
		return false
	}
	if fiberId := GetAppName(rs); fiberId == "" {
		return false
	}
	if spec == nil || status == nil {
		return false
	}
	return status.Status == PoolStatusCreated && status.Version != spec.Version
}

func SetPoolStatus(rs *RollingSet, status, version string) {
	if version != "" {
		setAnnotation(rs, LabelPoolStatus, utils.ObjJSON(PoolStatus{Status: status, Version: version}))
	}
	// for compatible reason
	setLabel(rs, LabelPoolStatus, status)
}

func IsSubrsEnable(rs *RollingSet) bool {
	return GetLabelValue(rs, LabelSubrsEnable) == "true"
}

func SetSubrsMetas(rs *RollingSet, metas string) {
	setAnnotation(rs, AnnotationSubrsMetas, metas)
}

func GetSubrsMetas(rs *RollingSet) string {
	return GetLabelAnnoValue(rs, AnnotationSubrsMetas)
}

// InitAnnotationsLabels InitAnnotationsLabels
func InitAnnotationsLabels(meta *metav1.ObjectMeta) {
	if meta.Labels == nil {
		meta.Labels = make(map[string]string)
	}
	if meta.Annotations == nil {
		meta.Annotations = make(map[string]string)
	}
}

func IsBizPod(meta *metav1.ObjectMeta) bool {
	return meta != nil && meta.Labels != nil && meta.Labels[LabelBizPodFlag] == "true"
}

// GetC2PodVersion GetPodVersion
func GetC2PodVersion(object metav1.Object) string {
	return GetLabelAnnoValue(object, AnnotationC2PodVersion)
}

// GetC2PodResVersion GetC2PodResVersion
func GetC2PodResVersion(object metav1.Object) string {
	return GetLabelAnnoValue(object, AnnotationC2ResourceVersion)
}

// GetRequirementID GetRequirementID
func GetRequirementID(object metav1.Object) string {
	return GetLabelAnnoValue(object, LabelKeyRequirementID)
}

// GetLaunchSignature GetLaunchSignature
func GetLaunchSignature(object metav1.Object) int64 {
	sigStr := GetLabelAnnoValue(object, LabelKeyLaunchSignature)
	sig, _ := strconv.ParseInt(sigStr, 10, 0)
	return sig
}

// GetServiceVersion GetServiceVersion
func GetServiceVersion(object metav1.Object) string {
	return GetLabelAnnoValue(object, LabelKeyServiceVersion)
}

// GetPriorityV1 GetPriorityV1
func GetPriorityV1(object metav1.Object) int32 {
	priorityStr := GetLabelAnnoValue(object, AnnotationKeyPriorityV1)
	priority, _ := strconv.Atoi(priorityStr)
	return int32(priority)
}

// GetPackageChecksum GetPackageChecksum
func GetPackageChecksum(object metav1.Object) string {
	return GetLabelAnnoValue(object, LabelKeyPackageChecksum)
}

// SetC2PodVersion SetC2PodVersion
func SetC2PodVersion(object metav1.Object, version string) {
	setAnnotation(object, AnnotationC2PodVersion, version)
}

// SetC2PodResVersion SetC2PodResVersion
func SetC2PodResVersion(object metav1.Object, version string) {
	setAnnotation(object, AnnotationC2ResourceVersion, version)
}

// SetRequirementID SetRequirementID
func SetRequirementID(object metav1.Object, version string) {
	setAnnotation(object, LabelKeyRequirementID, version)
}

// SetLaunchSignature SetLaunchSignature
func SetLaunchSignature(object metav1.Object, sig int64) {
	sigStr := strconv.FormatInt(sig, 10)
	setAnnotation(object, LabelKeyLaunchSignature, sigStr)
}

// SetServiceVersion SetServiceVersion
func SetServiceVersion(object metav1.Object, v string) {
	setAnnotation(object, LabelKeyServiceVersion, v)
}

// SetPackageChecksum SetPackageChecksum
func SetPackageChecksum(object metav1.Object, version string) {
	setAnnotation(object, LabelKeyPackageChecksum, version)
}

// IsPodReclaimed IsPodReclaimed
func IsPodReclaimed(pod *corev1.Pod) bool {
	if nil == pod || nil == pod.Labels {
		return false
	}
	return pod.Labels["alibabacloud.com/eviction"] == "true" || pod.Labels[LabelPodEvictionLegacySigma] == "true"
}

// IsPodCreateByC2 IsPodCreateByC2
func IsPodCreateByC2(pod *corev1.Pod) bool {
	if pod != nil && pod.Labels != nil && pod.Labels[LabelKeyPlatform] == "c2" {
		return true
	}
	return false
}

// SetIfExist SetIfExist
func SetIfExist(from, to map[string]string, key string) {
	if nil == from || nil == to {
		return
	}
	if v, ok := from[key]; ok {
		to[key] = v
	}
}

func IsBelowMinHealth(rs *RollingSet) bool {
	if rs == nil || rs.Spec.Strategy.RollingUpdate == nil || rs.Spec.Strategy.RollingUpdate.MaxUnavailable == nil {
		return false
	}
	replicas := int32(0)
	if rs.Spec.Replicas != nil {
		replicas = *rs.Spec.Replicas
	}
	scalePlan := rs.Spec.ScaleSchedulePlan
	if scalePlan != nil && scalePlan.Replicas != nil {
		replicas = *scalePlan.Replicas
	}
	if replicas == 0 {
		return false
	}
	ratio, _, err := rollalgorithm.GetIntOrPercentValue(rs.Spec.Strategy.RollingUpdate.MaxUnavailable)
	if err != nil {
		return false
	}
	minHealth := replicas - int32(float32(replicas)*float32(ratio)/100.0)
	return rs.Status.AvailableReplicas < minHealth
}

// IsV25 IsV25
func IsV25(object metav1.Object) bool {
	return GetPodVersion(object) == PodVersion25
}

// IsV3 IsV3
func IsV3(object metav1.Object) bool {
	return GetPodVersion(object) == PodVersion3
}

// IsLazyV3 IsLazyV3
func IsLazyV3(object metav1.Object) bool {
	return GetLabelValue(object, LazyPodVersion3) == "true"
}

// SetLazyV3 SetLazyV3
func SetLazyV3(object metav1.Object, lazyV3 bool) {
	labels := object.GetLabels()
	if lazyV3 {
		labels[LazyPodVersion3] = "true"
	} else {
		delete(labels, LazyPodVersion3)
	}
}

// FixNewCreateLazyV3PodVersion FixNewCreateLazyV3PodVersion
func FixNewCreateLazyV3PodVersion(object metav1.Object) {
	if IsLazyV3(object) && IsV25(object) {
		SetPodVersion(object, PodVersion3)
	}
}

func GetPoolSpec(rs *RollingSet) (*PoolSpec, error) {
	value := GetAnnotationValue(rs, AnnotationPoolSpec)
	if value == "" {
		return nil, nil
	}
	spec := &PoolSpec{}
	if err := json.Unmarshal([]byte(value), spec); err != nil {
		return nil, err
	}
	spec.Version, _ = utils.SignatureWithMD5(value)
	return spec, nil
}

func GetPoolStatus(rs *RollingSet) (*PoolStatus, error) {
	value := GetAnnotationValue(rs, LabelPoolStatus)
	if value == "" {
		// for compatible reason
		if status := GetLabelValue(rs, LabelPoolStatus); status == "" {
			return nil, nil
		} else {
			return &PoolStatus{Status: status}, nil
		}
	}
	status := &PoolStatus{}
	if err := json.Unmarshal([]byte(value), status); err != nil {
		return nil, err
	}
	return status, nil
}

func IsServerless(rs *RollingSet) bool {
	return GetLabelValue(rs, C2PlatformKey) == ServerlessPlatform
}

// GetEvictFailedMsg GetEvictFailedMsg
func GetEvictFailedMsg(object metav1.Object) string {
	return GetLabelAnnoValue(object, AnnoKeyEvictFailedMsg)
}

// SetEvictFailedMsg SetEvictFailedMsg
func SetEvictFailedMsg(object metav1.Object, state string) {
	setAnnotation(object, AnnoKeyEvictFailedMsg, "backup "+state)
}

// RemoveEvictFailedMsg RemoveEvictFailedMsg
func RemoveEvictFailedMsg(object metav1.Object) {
	removeAnnotation(object, AnnoKeyEvictFailedMsg)
}

func IsAsiNaming(rs *RollingSet) bool {
	if rs == nil || rs.Spec.Template == nil || rs.Spec.Template.Labels == nil {
		return false
	}
	return rs.Spec.Template.Labels["meta.k8s.alipay.com/ignore-naming"] != "true"
}

func IsNonVersionedKeysMatch(src metav1.Object, target metav1.Object, keys mapset.Set) bool {
	if common.IsInterfaceNil(src) || common.IsInterfaceNil(target) {
		return false
	}
	labels := InitIfNil(src.GetLabels())
	annos := InitIfNil(src.GetAnnotations())
	rLabels := InitIfNil(target.GetLabels())
	rAnnos := InitIfNil(target.GetAnnotations())
	for _, k := range keys.ToSlice() {
		key := k.(string)
		if labels[key] != rLabels[key] {
			return false
		}
		if annos[key] != rAnnos[key] {
			return false
		}
	}
	return true
}

func InitIfNil(m map[string]string) map[string]string {
	if m == nil {
		m = map[string]string{}
	}
	return m
}
