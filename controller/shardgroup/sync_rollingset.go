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

package shardgroup

import (
	RawErrors "errors"
	"reflect"
	"sort"

	"github.com/alibaba/kube-sharding/common/features"
	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	"github.com/alibaba/kube-sharding/pkg/rollalgorithm"
	apps "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	glog "k8s.io/klog"
	labelsutil "k8s.io/kubernetes/pkg/util/labels"
)

func (c *Controller) syncRollingSetGC(shardgroup *carbonv1.ShardGroup, rollingSets []*carbonv1.RollingSet) error {
	glog.Infof("start delete Operator shardgroup.name: %v, deletionTimestamp: %v", shardgroup.Name, shardgroup.DeletionTimestamp)
	if len(rollingSets) != 0 {
		for i := range rollingSets[:] {
			err := c.deleteUnusedRollingSet(shardgroup, rollingSets[i])
			if err != nil {
				return err
			}
		}
		return RawErrors.New("waiting for rollingSet to be deleted")
	}

	if shardgroup.Finalizers != nil || len(shardgroup.Finalizers) == 0 {
		shardgroup.Finalizers = nil
		err := c.ResourceManager.UpdateShardGroup(shardgroup)
		if nil != err {
			return err
		}
	}
	if err := c.ResourceManager.DeleteShardGroup(shardgroup); err != nil {
		return err
	}
	return nil
}

func (c *Controller) syncRollingSetSpec(shardGroup *carbonv1.ShardGroup,
	rollingSets []*carbonv1.RollingSet, shardGroupVersion string,
) error {
	created, err := c.createRollingsets(shardGroup, rollingSets, shardGroupVersion)
	if nil != err || created {
		return err
	}

	params, err := initGroupScheduleParams(shardGroup, rollingSets, shardGroupVersion)
	if nil != err {
		glog.Errorf("%s: initGroupScheduleParams error %v", shardGroup.Name, err)
		return err
	}

	isSimplifyShardGroup := c.isSimplifyShardGroup(shardGroup, rollingSets)
	if isSimplifyShardGroup {
		return c.simplifySyncRollingset(shardGroup, shardGroupVersion, rollingSets)
	}
	scheduler := rollalgorithm.NewSyncSlidingScheduler()
	shardParams, targetPercent, err := scheduler.Schedule(params)
	if nil != err {
		glog.Errorf("%s: initGroupScheduleParams error %v", shardGroup.Name, err)
		return err
	}
	if glog.V(4) {
		glog.Infof("shardGroup %#v get holdMatrix %s and targetPercent %v", shardGroup.Name, utils.ObjJSON(shardParams), targetPercent)
	}
	for i := range rollingSets[:] {
		rollingSet := rollingSets[i]
		needUpdate := false
		oldRollingSet := rollingSet.DeepCopy()
		for shardKey := range shardGroup.Spec.ShardTemplates {
			shardTemplate := shardGroup.Spec.ShardTemplates[shardKey]
			shardName := carbonv1.GenerateShardName(shardGroup.Name, shardKey)
			if shardName != rollingSet.Name {
				continue
			}
			schedulePlan := shardParams[shardName].Plan
			rollingSet.Spec.SchedulePlan = schedulePlan
			labels := carbonv1.CloneMap(shardGroup.Labels)
			carbonv1.AddGroupUniqueLabelHashKey(labels, shardGroup.Name)
			carbonv1.AddRsUniqueLabelHashKey(labels, oldRollingSet.Name)
			for labelKey, labelValue := range shardTemplate.ObjectMeta.Labels {
				labels = labelsutil.CloneAndAddLabel(labels, labelKey, labelValue)
			}
			rollingSet.ObjectMeta.Labels = labels
			rollingSet.ObjectMeta.Annotations = carbonv1.CloneMap(shardTemplate.ObjectMeta.Annotations)
			newVersionPlan := c.genVersionPlan(shardGroupVersion, shardGroup, &shardTemplate)
			rollingSet.Spec.VersionPlan = newVersionPlan
			rollingSet.Spec.RowComplete = utils.BoolPtr(shardParams[shardName].LatestRowComplete)
			rollingSet.Spec.HealthCheckerConfig = shardTemplate.Spec.HealthCheckerConfig
			if nil == rollingSet.Spec.Selector {
				rollingSet.Spec.Selector = oldRollingSet.Spec.Selector
			}
			needUpdate = c.needUpdate(oldRollingSet, rollingSet, &shardTemplate)
		}

		if needUpdate {
			err := c.updateRollingSet(rollingSet)
			if err != nil {
				glog.Errorf(" update rollingSet %v in namespace %v failed cause %v", rollingSet.Name, shardGroup.Namespace, err)
				return err
			}
		}
	}
	return nil
}

func (c *Controller) isSimplifyShardGroup(shardGroup *carbonv1.ShardGroup, rollingSets []*carbonv1.RollingSet) bool {
	if features.C2MutableFeatureGate.Enabled(features.SimplifyShardGroup) {
		return len(shardGroup.Spec.ShardTemplates) == 1 && len(rollingSets) == 1
	}
	return false
}

func (c *Controller) simplifySyncRollingset(shardGroup *carbonv1.ShardGroup, shardGroupVersion string, rollingSets []*carbonv1.RollingSet) error {
	rollingSet := rollingSets[0]
	needUpdate := false
	oldRollingSet := rollingSet.DeepCopy()
	labels := carbonv1.CloneMap(shardGroup.Labels)
	carbonv1.AddGroupUniqueLabelHashKey(labels, shardGroup.Name)
	carbonv1.AddRsUniqueLabelHashKey(labels, oldRollingSet.Name)
	for shardKey := range shardGroup.Spec.ShardTemplates {
		shardTemplate := shardGroup.Spec.ShardTemplates[shardKey]
		for labelKey, labelValue := range shardTemplate.ObjectMeta.Labels {
			labels = labelsutil.CloneAndAddLabel(labels, labelKey, labelValue)
		}
		rollingSet.ObjectMeta.Labels = labels
		rollingSet.ObjectMeta.Annotations = carbonv1.CloneMap(shardTemplate.ObjectMeta.Annotations)
		newVersionPlan := c.genVersionPlan(shardGroupVersion, shardGroup, &shardTemplate)
		rollingSet.Spec.VersionPlan = newVersionPlan
		strategy := shardTemplate.Spec.Strategy
		if strategy == nil || strategy.RollingUpdate == nil {
			strategy = &apps.DeploymentStrategy{
				RollingUpdate: &apps.RollingUpdateDeployment{
					MaxUnavailable: shardGroup.Spec.MaxUnavailable,
					MaxSurge:       shardGroup.Spec.MaxSurge,
				},
				Type: apps.RollingUpdateDeploymentStrategyType,
			}
		}
		schedulePlan := rollalgorithm.SchedulePlan{
			Replicas: shardTemplate.Spec.Replicas,
			Strategy: *strategy,
		}
		schedulePlan.LatestVersionRatio = shardGroup.Spec.LatestPercent
		rollingSet.Spec.SchedulePlan = schedulePlan
		rollingSet.Spec.HealthCheckerConfig = shardTemplate.Spec.HealthCheckerConfig
		if nil == rollingSet.Spec.Selector {
			rollingSet.Spec.Selector = oldRollingSet.Spec.Selector
		}
		needUpdate = c.needUpdate(oldRollingSet, rollingSet, &shardTemplate)
		if needUpdate {
			err := c.updateRollingSet(rollingSet)
			if err != nil {
				glog.Errorf(" update rollingSet %v in namespace %v failed cause %v", rollingSet.Name, shardGroup.Namespace, err)
				return err
			}
		}
		return nil
	}
	return nil
}

func (c *Controller) needUpdate(old, new *carbonv1.RollingSet, template *carbonv1.ShardTemplate) bool {
	var needUpdate = false
	tempTimestamp := new.Spec.UpdatePlanTimestamp
	new.Spec.UpdatePlanTimestamp = old.Spec.UpdatePlanTimestamp
	if utils.ObjJSON(new.Labels) != utils.ObjJSON(old.Labels) ||
		carbonv1.GetExclusiveLableAnno(new) != carbonv1.GetExclusiveLableAnno(old) ||
		utils.ObjJSON(new.Spec.HealthCheckerConfig) != utils.ObjJSON(old.Spec.HealthCheckerConfig) ||
		utils.ObjJSON(new.Spec.VersionPlan) != utils.ObjJSON(old.Spec.VersionPlan) ||
		utils.ObjJSON(new.Spec.SchedulePlan) != utils.ObjJSON(old.Spec.SchedulePlan) ||
		utils.ObjJSON(new.Spec.ScaleConfig) != utils.ObjJSON(old.Spec.ScaleConfig) {
		needUpdate = true
	}
	new.Spec.UpdatePlanTimestamp = tempTimestamp
	if template.Spec.GetReplicas() == 0 && old.GetReplicas() == 0 && template.Spec.UserDefVersion == old.Spec.UserDefVersion {
		if glog.V(4) {
			glog.Infof("rollingSet %v should not update spec repeatedly while replicas is zero ", new.Name)
		}
		needUpdate = false
	}
	return needUpdate
}

func (c *Controller) createRollingsets(shardGroup *carbonv1.ShardGroup, rollingSets []*carbonv1.RollingSet, shardGroupVersion string) (needCreate bool, err error) {
	for shardKey, shardTemplate := range shardGroup.Spec.ShardTemplates {
		shardName := carbonv1.GenerateShardName(shardGroup.Name, shardKey)
		createIfNotExist := true
		for _, rollingSet := range rollingSets {
			if rollingSet.Name == shardName {
				createIfNotExist = false
			}
		}

		if createIfNotExist {
			needCreate = true
			err = c.createRollingSet(shardGroup, &shardTemplate, shardName, shardGroupVersion)
			if err != nil {
				glog.Errorf("shardgroup %v create rollingSet %v failed cause %v",
					shardGroup.Namespace+"-"+shardGroup.Name, shardName, err)
				return
			}
		}
	}
	return
}

func (c *Controller) updateRollingSet(rollingSet *carbonv1.RollingSet) error {
	_, err := c.ResourceManager.UpdateRollingSet(rollingSet)
	glog.Infof("update rollingSet %v in namespace %v, %s, err: %v ", rollingSet.Name, rollingSet.Namespace, utils.ObjJSON(rollingSet), err)
	return err
}

func (c *Controller) deleteUnusedRollingSet(shardGroup *carbonv1.ShardGroup, rollingSet *carbonv1.RollingSet) error {
	if rollingSet.Spec.SchedulePlan.VersionHoldMatrix != nil {
		rollingSet.Spec.SchedulePlan.VersionHoldMatrix = map[string]int32{}
		err := c.updateRollingSet(rollingSet)
		if err != nil {
			glog.Errorf(" update rollingSet %v in namespace %v failed cause %v", rollingSet.Name, shardGroup.Namespace, err)
			return err
		}
	}
	err := c.ResourceManager.DeleteServicePublisherForRs(rollingSet)
	glog.Infof("delete rollingSet services %v by shardGroup %v in namespace %v, %v", rollingSet.Name, shardGroup.Name, rollingSet.Namespace, err)
	if nil != err {
		return err
	}

	err = c.ResourceManager.RemoveRollingSet(rollingSet)
	glog.Infof("delete rollingSet %v by shardGroup %v in namespace %v, %v", rollingSet.Name, shardGroup.Name, rollingSet.Namespace, err)
	if nil != err {
		return err
	}
	return err
}

func (c *Controller) genVersionPlan(shardGroupVersion string, shardGroup *carbonv1.ShardGroup, shardTemplate *carbonv1.ShardTemplate) carbonv1.VersionPlan {
	return carbonv1.VersionPlan{
		SignedVersionPlan: carbonv1.SignedVersionPlan{
			ShardGroupVersion:          shardGroupVersion,
			Template:                   &shardTemplate.Spec.Template,
			Cm2TopoInfo:                shardTemplate.Spec.Cm2TopoInfo,
			Signature:                  shardTemplate.Spec.Signature,
			RestartAfterResourceChange: shardGroup.Spec.RestartAfterResourceChange,
		},
		BroadcastPlan: carbonv1.BroadcastPlan{
			WorkerSchedulePlan:   shardGroup.Spec.WorkerSchedulePlan,
			CustomInfo:           shardTemplate.Spec.CustomInfo,
			CompressedCustomInfo: shardTemplate.Spec.CompressedCustomInfo,
			UserDefVersion:       shardTemplate.Spec.UserDefVersion,
			Online:               shardTemplate.Spec.Online,
			UpdatingGracefully: func() *bool {
				if nil != shardTemplate.Spec.UpdatingGracefully {
					return shardTemplate.Spec.UpdatingGracefully
				}
				return shardGroup.Spec.UpdatingGracefully // 兼容线上老plan
			}(),
			Preload:             shardTemplate.Spec.Preload,
			UpdatePlanTimestamp: shardGroup.Spec.UpdatePlanTimestamp,
		},
	}
}

func (c *Controller) createRollingSet(shardGroup *carbonv1.ShardGroup, shardTemplate *carbonv1.ShardTemplate, shardName string, shardGroupSign string) error {
	labels := carbonv1.CloneMap(shardGroup.Labels)
	carbonv1.AddGroupUniqueLabelHashKey(labels, shardGroup.Name)
	for labelKey, labelValue := range shardTemplate.ObjectMeta.Labels {
		labels = labelsutil.CloneAndAddLabel(labels, labelKey, labelValue)
	}
	// Create new RollingSet
	selector := carbonv1.CloneSelector(shardGroup.Spec.Selector)
	carbonv1.AddGroupSelectorHashKey(selector, shardGroup.Name)
	shardSpec := shardTemplate.Spec
	newRollingSet := &carbonv1.RollingSet{
		ObjectMeta: metav1.ObjectMeta{
			// Make the name deterministic, to ensure idempotence
			Name:            shardName,
			Namespace:       shardGroup.Namespace,
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(shardGroup, controllerKind)},
			Labels:          labels,
			Annotations:     carbonv1.CloneMap(shardTemplate.ObjectMeta.Annotations),
		},
		Spec: carbonv1.RollingSetSpec{
			Selector: selector,
			SchedulePlan: rollalgorithm.SchedulePlan{
				Replicas:           shardSpec.Replicas,
				LatestVersionRatio: shardGroup.Spec.LatestPercent,
				Strategy:           c.getSchedulePlanStrategy(shardGroup, shardTemplate),
			},
			VersionPlan:         c.genVersionPlan(shardGroupSign, shardGroup, shardTemplate),
			HealthCheckerConfig: shardSpec.HealthCheckerConfig,
		},
	}
	_, err := c.ResourceManager.CreateRollingSet(shardGroup, newRollingSet)
	glog.Infof("shardGroup %v create rollingset %v in %v, %v", shardGroup.Name, newRollingSet.Name, shardGroup.Namespace, err)
	return err
}

// It returns the list of RollingSets that this shardgroup should manage.
func (c *Controller) getRollingSetForShardGroup(shardGroup *carbonv1.ShardGroup) ([]*carbonv1.RollingSet, error) {
	selector, err := metav1.LabelSelectorAsSelector(shardGroup.Spec.Selector)
	if err != nil {
		return nil, err
	}
	rollingSets, err := c.rollingSetLister.RollingSets(shardGroup.Namespace).List(selector)
	if err != nil {
		return nil, err
	}
	rollingSetsCopy := make([]*carbonv1.RollingSet, len(rollingSets))
	for i := range rollingSets[:] {
		rollingSetsCopy[i] = rollingSets[i].DeepCopy()
	}
	for i := range rollingSetsCopy {
		rs := rollingSetsCopy[i]
		replicas, err := c.ResourceManager.ListReplicaForRS(rs)
		if err != nil {
			return nil, err
		}
		_, groupVersionStatusMap := carbonv1.GetGroupVersionStatusMap(replicas)
		rs.Status.GroupVersionStatusMap = groupVersionStatusMap
	}
	return rollingSetsCopy, nil
}

func (c *Controller) syncRollingSetGroupStatus(rawShardGroup *carbonv1.ShardGroup, shardGroup *carbonv1.ShardGroup, rollingSets []*carbonv1.RollingSet, shardGroupVersion string) error {
	currentShardNames := []string{}
	readyShardNames := []string{}
	complete := false
	phase := carbonv1.ShardGroupRunning
	readyShard := 0
	unUsedRollingSet := []*carbonv1.RollingSet{}
	for _, rollingSet := range rollingSets[:] {
		shardComplete := true
		if isRollingsetNotComplete(shardGroup, rollingSet, shardGroupVersion) {
			phase = carbonv1.ShardGroupRolling
			shardComplete = false
		}
		unUsed := true
		for shardKey := range (*shardGroup).Spec.ShardTemplates {
			if rollingSet.Name == carbonv1.GenerateShardName(shardGroup.Name, shardKey) {
				currentShardNames = append(currentShardNames, rollingSet.Name)
				if shardComplete {
					readyShard++
					readyShardNames = append(readyShardNames, rollingSet.Name)
				}
				unUsed = false
				break
			}
		}
		if unUsed {
			glog.Infof("find unused rollingset %v in namespace %v", rollingSet.Name, rollingSet.Namespace)
			unUsedRollingSet = append(unUsedRollingSet, rollingSet)
		}
	}

	if (len(unUsedRollingSet) > 0) && (len(unUsedRollingSet)+readyShard == len(rollingSets)) {
		glog.Infof("readyShard %v and unusedShard %v and current rollingSet %#v", readyShard, len(unUsedRollingSet), currentShardNames)
		for _, rollingSet := range unUsedRollingSet {
			glog.Infof("try to delete unused rollingSet %v in namespace %v", rollingSet.Name, rollingSet.Namespace)
			c.deleteUnusedRollingSet(shardGroup, rollingSet)
		}
	}
	sort.Strings(currentShardNames)
	sort.Strings(readyShardNames)
	// 所有列全部完成
	if len(shardGroup.Spec.ShardTemplates) == readyShard {
		complete = true
	}

	if phase != shardGroup.Status.Phase || !reflect.DeepEqual(shardGroup.Status.ShardNames, currentShardNames) ||
		shardGroupVersion != shardGroup.Status.ShardGroupVersion || shardGroup.Status.Complete != complete {
		shardGroup.Status.ShardGroupVersion = shardGroupVersion
		shardGroup.Status.Phase = phase
		shardGroup.Status.ShardNames = currentShardNames
		shardGroup.Status.ObservedGeneration = shardGroup.Generation
		shardGroup.Status.Complete = complete
		if complete {
			shardGroup.Status.OnceCompletedShardNames = readyShardNames
		}
		shardGroup.Spec = rawShardGroup.Spec
		err := c.ResourceManager.UpdateShardGroupStatus(shardGroup)
		if err != nil {
			glog.Warningf("[%s] UpdateShardgroupStatus failure, err: %v", shardGroup.Namespace+"/"+shardGroup.Name, err)
		}
		return err
	}
	return nil
}

func (c *Controller) getSchedulePlanStrategy(shardGroup *carbonv1.ShardGroup, shardTemplate *carbonv1.ShardTemplate) apps.DeploymentStrategy {
	var strategy = apps.DeploymentStrategy{
		RollingUpdate: &apps.RollingUpdateDeployment{
			MaxUnavailable: shardGroup.Spec.MaxUnavailable,
			MaxSurge:       shardGroup.Spec.MaxSurge,
		},
	}
	if nil != shardTemplate && nil != shardTemplate.Spec.Strategy {
		strategy = *shardTemplate.Spec.Strategy
	}
	return strategy
}

func isRollingsetNotComplete(shardGroup *carbonv1.ShardGroup, rollingSet *carbonv1.RollingSet, shardGroupVersion string) bool {
	// 空目标
	if nil == rollingSet.Spec.LatestVersionRatio || nil == shardGroup.Spec.LatestPercent {
		if glog.V(5) {
			glog.Infof("get rollingSet  %v , LatestVersionRatio nil", rollingSet.Name)
		}
		return true
	}
	// rollingset 最新Version未调度
	if rollingSet.Status.Version != rollingSet.Spec.Version || rollingSet.Spec.Version == "" {
		if glog.V(5) {
			glog.Infof("get rollingSet  %v Spec.Version %v and Status.Version %v ", rollingSet.Name, rollingSet.Spec.Version, rollingSet.Status.Version)
		}
		return true
	}
	// rollingset 最新Version调度状态未完成
	if rollingSet.Status.Complete != true || rollingSet.Spec.ShardGroupVersion != shardGroupVersion ||
		*(rollingSet.Spec.LatestVersionRatio) != *(shardGroup.Spec.LatestPercent) ||
		(*(rollingSet.Spec.LatestVersionRatio) != 0 && *(rollingSet.Spec.LatestVersionRatio) < 100) {
		if glog.V(5) {
			glog.Infof("get rollingSet  %v Status.Complete %v and Spec.LatestVersionRatio %v and .Spec.ShardGroupVersion %v", rollingSet.Name, rollingSet.Status.Complete, *(rollingSet.Spec.LatestVersionRatio), rollingSet.Spec.ShardGroupVersion)
			glog.Infof("get shardgroup  %v Spec.LatestPercent %v and shardGroupVersion %v", shardGroup.Name, *shardGroup.Spec.LatestPercent, shardGroupVersion)
		}
		return true
	}
	// groupVersionStatus 未同步
	groupVersionStatus := rollingSet.Status.GroupVersionStatusMap[shardGroupVersion]
	if nil == groupVersionStatus || nil == rollingSet.Spec.Replicas ||
		groupVersionStatus.Replicas != rollingSet.GetReplicas() || groupVersionStatus.ReadyReplicas != rollingSet.GetReplicas() {
		if glog.V(5) {
			glog.Infof("get rollingSet  %v groupStatus not ready", rollingSet.Name)
		}
		return true
	}
	return false
}
