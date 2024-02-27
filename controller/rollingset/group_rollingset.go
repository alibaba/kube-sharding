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

package rollingset

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/alibaba/kube-sharding/common/router"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	"github.com/alibaba/kube-sharding/pkg/rollalgorithm"
	app "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	intstrutil "k8s.io/apimachinery/pkg/util/intstr"
	glog "k8s.io/klog"
)

func (c *Controller) syncGroupRollingSet(rollingSet *carbonv1.RollingSet) error {
	rollingSetCopy := rollingSet.DeepCopy()
	subRollingSets, bufferRollingSet, err := c.getSubRollingSets(rollingSetCopy)
	if err != nil {
		return err
	}
	if rollingSetCopy.DeletionTimestamp != nil {
		return c.clearGroupRollingSet(append(subRollingSets, bufferRollingSet))
	}
	subRollingSetToUpdate, globalLatestVersionRatio, err := c.getSubRollingSetToUpdate(rollingSetCopy, subRollingSets)
	if err != nil {
		return err
	}
	if err = c.updateSubRollingSets(rollingSetCopy, subRollingSetToUpdate); err != nil {
		return err
	}
	if bufferRollingSet != nil {
		if err = c.updateBufferRollingSets(rollingSetCopy, bufferRollingSet, globalLatestVersionRatio, subRollingSetToUpdate); err != nil {
			return err
		}
	}
	if err = c.computeStatus(append(subRollingSets, bufferRollingSet), rollingSetCopy); err != nil {
		return err
	}
	if before, after := utils.ObjJSON(rollingSet.Status), utils.ObjJSON(rollingSetCopy.Status); before != after {
		glog.Infof("[%s] UpdateRollingSetStatus, before: %s after: %s", rollingSetCopy.Name, before, after)
		if err = c.ResourceManager.UpdateRollingSetStatus(rollingSetCopy); err != nil {
			glog.Warningf("[%s] UpdateRollingSetStatus failure, err: %v", rollingSetCopy.Name, err)
			return err
		}
	}
	return nil
}

func (c *Controller) completeGroupRollingSet(rollingSet *carbonv1.RollingSet) error {
	if carbonv1.IsServerless(rollingSet) {
		if err := c.completeServerless(rollingSet); err != nil {
			return err
		}
	}
	return nil
}

func (c *Controller) computeStatus(subrsList []*carbonv1.RollingSet, rollingSet *carbonv1.RollingSet) error {
	statusMap := make(carbonv1.SubrsVersionStatusMap)
	replicasMap := make(map[string]*int32)
	targetReachedMap := make(map[string]bool)
	complete := true
	for i := range subrsList {
		subrs := subrsList[i]
		if subrs == nil {
			continue
		}
		if !subrs.Status.Complete {
			complete = false
		}
		rawName := carbonv1.GetSubrsShortUniqueKey(subrs)
		replicas := subrs.Spec.Replicas
		if subrs.Spec.ScaleSchedulePlan != nil && subrs.Spec.ScaleSchedulePlan.Replicas != nil &&
			*subrs.Spec.ScaleSchedulePlan.Replicas != 0 {
			replicas = subrs.Spec.ScaleSchedulePlan.Replicas
		}
		if replicas == nil {
			continue
		}
		replicasMap[rawName] = replicas
		if *replicas == 0 {
			targetReachedMap[rawName] = true
			continue
		}
		if subrs.Status.GroupVersionStatusMap == nil {
			statusMap[rawName] = carbonv1.GroupVersionStatusMap{}
		} else {
			statusMap[rawName] = subrs.Status.GroupVersionStatusMap
		}
		if ratio := c.getLatestVersionRatio(rollingSet, subrs); ratio != 0 {
			status := subrs.Status.GroupVersionStatusMap[rollingSet.Spec.Version]
			targetReachedMap[rawName] = status != nil && ratio <= int32(math.Floor(float64(status.ReadyReplicas)/float64(*replicas)*100))
		}
	}
	rollingSet.Status.Complete = complete
	rollingSet.Status.SubrsVersionStatusMap = statusMap
	rollingSet.Status.SubrsTargetReplicas = replicasMap
	rollingSet.Status.SubrsRatioVersion = rollingSet.Spec.SubrsRatioVersion
	rollingSet.Status.SubrsTargetReached = targetReachedMap
	rollingSet.Status.Version = rollingSet.Spec.Version
	return nil
}

func (c *Controller) clearGroupRollingSet(subrss []*carbonv1.RollingSet) error {
	for i := range subrss {
		subrs := subrss[i]
		if subrs != nil {
			err := c.ResourceManager.RemoveRollingSet(subrs)
			if err != nil && !errors.IsNotFound(err) {
				return err
			}
		}
	}
	return nil
}

func (c *Controller) updateSubRollingSets(groupRollingSet *carbonv1.RollingSet, subRollingSets []*carbonv1.RollingSet) error {
	for i := range subRollingSets {
		subRollingSet := subRollingSets[i]
		c.fixSubrs(groupRollingSet, subRollingSet)
		_, err := c.ResourceManager.UpdateRollingSet(subRollingSet)
		glog.Infof("updateSubRollingSet %s %v", utils.ObjJSON(subRollingSet), err)
		if nil != err {
			return err
		}
	}
	return nil
}

func (c *Controller) updateBufferRollingSets(groupRollingSet *carbonv1.RollingSet, bufferRollingSet *carbonv1.RollingSet, latestVersionRatio int32, subRollingSets []*carbonv1.RollingSet) error {
	// TOTO 接管后删除
	if bufferRollingSet.Spec.Version == "" && len(subRollingSets) > 0 {
		time.Sleep(time.Second * 30)
		glog.Infof("%s first switch to subrs mode, wait 30s to take over workers", groupRollingSet.Name)
	}
	var bufferRollingSetCopy = bufferRollingSet.DeepCopy()
	subRSLatestVersionRatio := groupRollingSet.Spec.SubRSLatestVersionRatio
	if subRSLatestVersionRatio == nil {
		latestVersionRatio = 100
	} else if ratio, ok := (*subRSLatestVersionRatio)[carbonv1.GroupRollingSetBufferShortName]; ok {
		latestVersionRatio = ratio
	}
	bufferRollingSetCopy.Spec.LatestVersionRatio = utils.Int32Ptr(latestVersionRatio)
	c.fixSubrs(groupRollingSet, bufferRollingSetCopy)
	if utils.ObjJSON(bufferRollingSetCopy.Spec) != utils.ObjJSON(bufferRollingSet.Spec) {
		_, err := c.ResourceManager.UpdateRollingSet(bufferRollingSetCopy)
		glog.Infof("updateSubRollingSet %s, after: %s, %v", utils.ObjJSON(bufferRollingSet.Spec), utils.ObjJSON(bufferRollingSetCopy.Spec), err)
		return err
	}
	return nil
}

func (c *Controller) fixSubrs(groupRollingSet *carbonv1.RollingSet, rollingSet *carbonv1.RollingSet) {
	rollingSet.Spec.VersionPlan = groupRollingSet.Spec.VersionPlan
	rollingSet.Spec.ShardGroupVersion = groupRollingSet.Spec.ShardGroupVersion
	rollingSet.Spec.Version = groupRollingSet.Spec.Version
	rollingSet.Spec.ResVersion = groupRollingSet.Spec.ResVersion
	rollingSet.Spec.InstanceID = groupRollingSet.Spec.InstanceID
	rollingSet.Spec.SubrsPaused = groupRollingSet.Spec.Paused
	rollingSet.Spec.Template = groupRollingSet.Spec.Template.DeepCopy()
}

func (c *Controller) getSubRollingSetToUpdate(groupRollingSet *carbonv1.RollingSet, subRollingSets []*carbonv1.RollingSet) ([]*carbonv1.RollingSet, int32, error) {
	var subRSToUpdate []*carbonv1.RollingSet
	// rolling中的，需要被更新的
	inRolling, toUpdate, err := c.computeSubRollingSet(groupRollingSet, subRollingSets)
	if err != nil {
		return nil, 0, err
	}
	// 统计rs的总节点数
	totalCount, globalLatestVersionRatio := c.sumReplicas(groupRollingSet, subRollingSets)
	// 允许更新的所有节点
	var maxUnavailable *intstrutil.IntOrString
	if groupRollingSet.Spec.Strategy.RollingUpdate != nil && groupRollingSet.Spec.Strategy.RollingUpdate.MaxUnavailable != nil {
		maxUnavailable = groupRollingSet.Spec.Strategy.RollingUpdate.MaxUnavailable
	}
	totalCountAllowUpdate, err := intstrutil.GetValueFromIntOrPercent(
		intstrutil.ValueOrDefault(maxUnavailable, rollalgorithm.Default25IntOrString), int(totalCount), false)
	if nil != err {
		glog.Errorf("GetValueFromIntOrPercent error: %s, %v", groupRollingSet.Name, err)
		return nil, globalLatestVersionRatio, nil
	}
	// 排除正在rolling中的节点数
	var totalCountInUpdate int32
	for i := range inRolling {
		_, countInUpdate, _, _, _, _ := rollalgorithm.ParseScheduleStrategy(inRolling[i].Spec.SchedulePlan)
		totalCountInUpdate += countInUpdate
	}
	// 选择新的rs，可以参与rolling的
	totalCountNeedToUpdate := int32(totalCountAllowUpdate) - totalCountInUpdate
	for i := 0; i < len(toUpdate) && totalCountNeedToUpdate >= 0 && !groupRollingSet.Spec.Paused; i++ {
		subRSToUpdate = append(subRSToUpdate, toUpdate[i])
		_, countToRolling, _, _, _, _ := rollalgorithm.ParseScheduleStrategy(toUpdate[i].Spec.SchedulePlan)
		totalCountNeedToUpdate -= countToRolling
	}
	if groupRollingSet.Spec.Paused {
		subRSToUpdate = toUpdate
	}
	glog.Infof("getSubRollingSetToUpdate %s, subRollingSets: %d, subRSToUpdate: %d, toUpdate: %d, inRolling: %d, paused: %v",
		groupRollingSet.Name, len(subRollingSets), len(subRSToUpdate), len(toUpdate), len(inRolling), groupRollingSet.Spec.Paused)
	return subRSToUpdate, globalLatestVersionRatio, nil
}

func (c *Controller) computeSubRollingSet(groupRs *carbonv1.RollingSet, subRollingSets []*carbonv1.RollingSet) (inRolling, toUpdate []*carbonv1.RollingSet, err error) {
	keys, err := carbonv1.GetNonVersionedKeys()
	if err != nil {
		return
	}
	for i := range subRollingSets {
		subrs := subRollingSets[i]
		rawName := carbonv1.GetSubrsShortUniqueKey(subrs)
		if strSliceContains(groupRs.Spec.SubrsBlackList, rawName) || (subrs.Spec.SubrsPaused && groupRs.Spec.Paused) {
			continue
		}
		ratio := c.getLatestVersionRatio(groupRs, subrs)
		match := ratio == 0 || (ratio != 0 && carbonv1.IsNonVersionedKeysMatch(groupRs.Spec.Template, subrs.Spec.Template, keys))
		specRatio := subrs.Spec.LatestVersionRatio
		// subrs can't rollback
		if !groupRs.Spec.SubrsCanRollback && subrs.Spec.Version == groupRs.Spec.Version && ratio > 0 &&
			specRatio != nil && ratio < *specRatio {
			ratio = *specRatio
		}
		pausedRatio := false
		if groupRs.Spec.Paused && subrs.Spec.Replicas != nil && !subrs.Spec.SubrsPaused {
			ratio = subrs.Status.UpdatedReplicas * 100 / *subrs.Spec.Replicas
			pausedRatio = true
		}
		if subrs.Spec.Version == "" || specRatio == nil || pausedRatio || !match ||
			(*specRatio != ratio && ratio != 0) ||
			(subrs.Spec.Version != groupRs.Spec.Version && ratio != 0) {
			subrs.Spec.LatestVersionRatio = &ratio
			toUpdate = append(toUpdate, subrs)
		}
		if subrs.Spec.Version == groupRs.Spec.Version && specRatio != nil && *specRatio == ratio && !c.isRollingSetMatched(subrs) {
			inRolling = append(inRolling, subrs)
		}
	}
	return
}

func (c *Controller) getLatestVersionRatio(groupRollingSet *carbonv1.RollingSet, subRollingSet *carbonv1.RollingSet) int32 {
	if groupRollingSet.Spec.SubRSLatestVersionRatio == nil {
		return 0
	}
	rawName := carbonv1.GetSubrsShortUniqueKey(subRollingSet)
	if strSliceContains(groupRollingSet.Spec.SubrsBlackList, rawName) {
		return 0
	}
	if latestVersionRatio, ok := (*groupRollingSet.Spec.SubRSLatestVersionRatio)[subRollingSet.Name]; ok {
		return latestVersionRatio
	}
	subrsName := carbonv1.GetSubrsShortUniqueKey(subRollingSet)
	ratioMap := *groupRollingSet.Spec.SubRSLatestVersionRatio
	if ratio, ok := ratioMap[subrsName]; ok {
		return ratio
	}
	if split := strings.Split(subrsName, "."); len(split) == 2 {
		if ratio, ok := ratioMap[split[0]+".*"]; ok {
			return ratio
		}
	}
	if ratio, ok := ratioMap["*"]; ok {
		return ratio
	}
	return 0
}

func (c *Controller) sumReplicas(groupRollingSet *carbonv1.RollingSet, subRollingSets []*carbonv1.RollingSet) (int32, int32) {
	var totalReplicas int32
	var totalLatestReplicas int32
	var globalLatestVersionRatio int32
	for i := range subRollingSets {
		totalReplicas += *subRollingSets[i].Spec.Replicas
		latestVersionRatio := c.getLatestVersionRatio(groupRollingSet, subRollingSets[i])
		rawName := carbonv1.GetSubrsShortUniqueKey(subRollingSets[i])
		if strSliceContains(groupRollingSet.Spec.SubrsBlackList, rawName) {
			latestVersionRatio = *subRollingSets[i].Spec.LatestVersionRatio
		}
		latestReplicas := rollalgorithm.ComputeLatestReplicas(
			latestVersionRatio, subRollingSets[i].GetReplicas(), rollalgorithm.CeilCarryStrategyType)
		totalLatestReplicas += latestReplicas
	}
	if totalReplicas == 0 {
		globalLatestVersionRatio = 100
	} else {
		globalLatestVersionRatio = int32(float32(totalLatestReplicas) * 100.0 / float32(totalReplicas))
	}
	return totalReplicas, globalLatestVersionRatio
}

func (c *Controller) getSubRollingSets(rollingSet *carbonv1.RollingSet) ([]*carbonv1.RollingSet, *carbonv1.RollingSet, error) {
	selector, err := metav1.LabelSelectorAsSelector(rollingSet.Spec.Selector)
	if err != nil {
		glog.Errorf("No slector in rollingset spec: %s", rollingSet.Name)
		return nil, nil, err
	}
	rollingSets, err := c.rollingSetLister.RollingSets(rollingSet.Namespace).List(selector)
	if err != nil {
		return nil, nil, err
	}
	var subRollingsets = []*carbonv1.RollingSet{}
	var bufferRollingset *carbonv1.RollingSet
	for i := range rollingSets {
		if !carbonv1.IsSubRollingSet(rollingSets[i]) {
			continue
		}
		subRollingSet := rollingSets[i].DeepCopy()
		if subRollingSet.Name == rollingSet.Name {
			continue
		}
		c.completeSubRollingSet(subRollingSet)
		if carbonv1.IsBufferRollingSet(subRollingSet) {
			bufferRollingset = subRollingSet
		} else {
			subRollingsets = append(subRollingsets, subRollingSet)
		}
	}
	return subRollingsets, bufferRollingset, nil
}

func (c *Controller) isRollingSetMatched(rollingSet *carbonv1.RollingSet) bool {
	version := rollingSet.Spec.Version
	if rollingSet.Spec.ShardGroupVersion != "" {
		version = rollingSet.Spec.ShardGroupVersion
	}
	latestVersionStatus := rollingSet.Status.GroupVersionStatusMap[version]
	if latestVersionStatus != nil {
		targetLatestReplicas := rollalgorithm.ComputeLatestReplicas(*rollingSet.Spec.LatestVersionRatio, rollingSet.GetReplicas(), rollalgorithm.CeilCarryStrategyType)
		if latestVersionStatus.ReadyReplicas >= targetLatestReplicas {
			return true
		}
	}
	return false
}

func (c *Controller) completeSubRollingSet(rollingSet *carbonv1.RollingSet) error {
	// 补齐template
	groupRollingSet, err := c.rollingSetLister.RollingSets(rollingSet.Namespace).Get(*rollingSet.Spec.GroupRS)
	if err != nil {
		return err
	}
	if !carbonv1.IsSubrsEnable(groupRollingSet) {
		err = fmt.Errorf("%s subrs not enable", groupRollingSet.Name)
		return err
	}
	// for compatibility
	if rollingSet.Spec.Template == nil {
		rollingSet.Spec.Template = groupRollingSet.Spec.Template.DeepCopy()
	}
	if rollingSet.Spec.Strategy.RollingUpdate == nil {
		rollingSet.Spec.Strategy.RollingUpdate = &app.RollingUpdateDeployment{}
	}
	unavailableInt := 10
	if rollingSet.GetReplicas() != 0 && (int(100.0/float32(rollingSet.GetReplicas()))+1) >= unavailableInt {
		unavailableInt = int(100.0/float32(rollingSet.GetReplicas())) + 1
	}
	groupStrategy := groupRollingSet.Spec.SchedulePlan.Strategy.RollingUpdate
	if groupStrategy != nil && groupStrategy.MaxUnavailable != nil {
		if group, _, err := rollalgorithm.GetIntOrPercentValue(groupStrategy.MaxUnavailable); err != nil {
			return err
		} else if group > unavailableInt {
			unavailableInt = group
		}
	}
	unavailableStr := intstrutil.FromString(fmt.Sprintf("%d%%", unavailableInt))
	rollingSet.Spec.Strategy.RollingUpdate.MaxUnavailable = &unavailableStr
	return nil
}

func (c *Controller) completeServerless(rollingSet *carbonv1.RollingSet) error {
	if carbonv1.GetAppName(rollingSet) != "" {
		return nil
	}
	r := router.NewGlobalConfigRouter()
	appName := carbonv1.GetLabelValue(rollingSet, carbonv1.LabelKeySigmaAppName)
	stage := carbonv1.GetLabelValue(rollingSet, carbonv1.LabelKeyAppStage)
	fiberId, err := r.GetFiberId(appName, "", "", stage, "")
	if err != nil || fiberId == "" {
		return err
	}
	roleName := utils.AppendString(rollingSet.Name, ".", rollingSet.Name)
	carbonv1.SetObjectAppRoleNameHash(rollingSet, fiberId, roleName, rollingSet.Name)
	carbonv1.SetObjectAppRoleNameHash(rollingSet.Spec.Template, fiberId, roleName, rollingSet.Name)
	carbonv1.SetClusterName(rollingSet, c.cluster)
	carbonv1.SetClusterName(rollingSet.Spec.Template, c.cluster)
	return nil
}
