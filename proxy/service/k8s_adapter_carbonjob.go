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

package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/alibaba/kube-sharding/common"
	"github.com/alibaba/kube-sharding/common/features"
	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	v1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	"github.com/alibaba/kube-sharding/pkg/apiserver-client/typespec"
	"github.com/alibaba/kube-sharding/pkg/apiserver-client/typespec/carbon"
	"github.com/alibaba/kube-sharding/proxy/apiset"
	"github.com/alibaba/kube-sharding/transfer"
	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *k8sAdapter) setCarbonJobGroups(appName, checksum string, groupPlans map[string]*typespec.GroupPlan) error {
	if nil == groupPlans {
		return fmt.Errorf("setCarbonJobGroups failed, groupPlans is nil")
	}
	if features.C2MutableFeatureGate.Enabled(features.GroupCarbonJobByRoleShortName) {
		return c.setCarbonJobGroupByRoleShortName(appName, checksum, groupPlans)
	}
	return c.setCarbonJobGroupByAPP(appName, checksum, groupPlans)
}

func (c *k8sAdapter) setCarbonJobGroupByAPP(appName, checksum string, groupPlans map[string]*typespec.GroupPlan) error {
	namespace, exclusiveMode, quota, err := c.getAdminInfos(appName, checksum)
	if err != nil {
		return err
	}
	carbonJob, err := c.getCarbonJobByAppName(appName, namespace)
	if nil != err && !errors.IsNotFound(err) {
		return err
	}
	if errors.IsNotFound(err) {
		if len(groupPlans) == 0 {
			return nil
		}
		name := transfer.EscapeName(appName)
		carbonJob := c.generateCarbonJob(appName, name, namespace, exclusiveMode, quota, checksum, "", groupPlans)
		err = convertCarbonJob(appName, carbonJob)
		if err != nil {
			return err
		}
		err = c.doCreateCarbonJob(appName, checksum, carbonJob)
		if nil != err {
			return fmt.Errorf("setCarbonJobGroups failed, create err: %v", err)
		}
		return nil
	}

	newCarbonJob := carbonJob.DeepCopy()
	newCarbonJob.Spec.AppPlan = &carbonv1.CarbonAppPlan{
		Groups: groupPlans,
	}
	if newCarbonJob.Labels == nil {
		newCarbonJob.Labels = map[string]string{} // for case
	}
	newCarbonJob.Labels[v1.LabelKeyAppChecksum] = checksum
	err = convertCarbonJob(appName, newCarbonJob)
	if err != nil {
		return err
	}
	if ((newCarbonJob.Spec.JobPlan == nil) != (carbonJob.Spec.JobPlan == nil)) ||
		newCarbonJob.Spec.Checksum != carbonJob.Spec.Checksum {
		if err = c.manager.getCarbonJobAPIs().UpdateCarbonJob(newCarbonJob); nil != err {
			return fmt.Errorf("setCarbonJobGroups failed, update err: %s", err)
		}
	}
	return nil
}

func (c *k8sAdapter) setCarbonJobGroupByRoleShortName(appName, checksum string, groupPlans map[string]*typespec.GroupPlan) error {
	namespace, targetCarbonJobs, err := c.generateCarbonJobs(appName, checksum, groupPlans)
	if nil != err {
		return err
	}
	carbonJobs, err := c.getCarbonJobsByAppName(appName, namespace)
	if nil != err && !errors.IsNotFound(err) {
		glog.Errorf("get carbonjobs error: %s, %v", appName, err)
		return err
	}
	return c.syncCarbonJobs(appName, checksum, carbonJobs, targetCarbonJobs)
}

func (c *k8sAdapter) getNamespace(appName string) (string, error) {
	// namespace only connect to appName in new router, no need pass groupplan
	namespace, err := c.manager.getTargetNamespace(appName)
	if nil != err {
		glog.Warningf("get namespace failed, appName: %s, err: %v", appName, err)
		return "", err
	}
	if err := c.manager.createNamespace(namespace); nil != err {
		glog.Errorf("create namespace failed, appName: %s, namespace: %s, err: %v", appName, namespace, err)
		return "", err
	}
	return namespace, nil
}

func (c *k8sAdapter) getCarbonJobByAppName(appName, namespace string) (*carbonv1.CarbonJob, error) {
	if namespace == "" {
		var err error
		namespace, err = c.getNamespace(appName)
		if nil != err {
			glog.Errorf("getCarbonJobByAppName failed, appName: %s getNamespace failed, err: %s", appName, err)
			return nil, err
		}
	}
	carbonJob, err := c.manager.getCarbonJobAPIs().GetCarbonJob(namespace, appName)
	if nil != err {
		if errors.IsNotFound(err) {
			return nil, err
		}
		glog.Errorf("getCarbonJobByAppName getCarbonJob faield, namespace: %s, app: %s, err: %s", namespace, appName, err)
		return nil, err
	}
	return carbonJob, nil
}

func (c *k8sAdapter) getCarbonJobsByAppName(appName, namespace string) ([]carbonv1.CarbonJob, error) {
	if namespace != "" {
		var err error
		namespace, err = c.getNamespace(appName)
		if nil != err {
			glog.Errorf("getCarbonJobsByAppName failed, appName: %s getNamespace failed, err: %s", appName, err)
			return nil, err
		}
	}
	carbonJobs, err := c.manager.getCarbonJobAPIs().GetCarbonJobs(namespace, appName)
	if nil != err {
		if errors.IsNotFound(err) {
			return nil, err
		}
		glog.Errorf("getCarbonJobsByAppName getCarbonJob faield, namespace: %s, app: %s, err: %s", namespace, appName, err)
		return nil, err
	}
	return carbonJobs, nil
}

func (c *k8sAdapter) deleteCarbonJobGroup(appName, groupID string) error {
	// not used
	return nil
}

func (c *k8sAdapter) doCreateCarbonJob(appName, checksum string, carbonJob *carbonv1.CarbonJob) error {
	carbonJob.Spec.AppName = appName
	_, err := c.manager.getCarbonJobAPIs().CreateCarbonJob(carbonJob)
	if nil != err {
		glog.Errorf("doCreateCarbonJob failed, err: %v", err)
		return err
	}
	return nil
}

func (c *k8sAdapter) getAdminInfos(appName, checksum string) (namespace, exclusiveMode, quota string, err error) {
	if c.manager == nil {
		return
	}
	namespace, err = c.getNamespace(appName)
	if err != nil {
		glog.Errorf("%s getNamespace failed, err: %s", appName, err)
	}
	return
}

func (c *k8sAdapter) createCarbonJobGroup(appName, checksum string, groupPlan *typespec.GroupPlan) error {
	// not used
	return nil
}

func (c *k8sAdapter) syncCarbonJobs(appName, checksum string, carbonJobs []carbonv1.CarbonJob, targetCarbonJobs map[string]*v1.CarbonJob) error {
	var err error
	var currentCarbonJobs = map[string]*v1.CarbonJob{}
	// 同步carbonjob
	for i := range carbonJobs {
		carbonJob := &carbonJobs[i]
		currentCarbonJobs[carbonJob.Name] = carbonJob
		targetCarbonJob, ok := targetCarbonJobs[carbonJob.Name]
		if !ok {
			if err = c.manager.getCarbonJobAPIs().DeleteCarbonJob(carbonJob); nil != err {
				return fmt.Errorf("DeleteCarbonJob failed, update err: %s", err)
			}
		} else if ((targetCarbonJob.Spec.JobPlan == nil) != (carbonJob.Spec.JobPlan == nil)) ||
			targetCarbonJob.Spec.Checksum != carbonJob.Spec.Checksum ||
			targetCarbonJob.Labels[carbonv1.LabelKeyQuotaGroupID] != carbonJob.Labels[carbonv1.LabelKeyQuotaGroupID] {
			if carbonJob.DeletionTimestamp != nil {
				return fmt.Errorf("CarbonJob %s in deleting", carbonJob.Name)
			}
			targetMetas := targetCarbonJob.ObjectMeta
			targetCarbonJob.ObjectMeta = carbonJob.ObjectMeta
			targetCarbonJob.Labels = targetMetas.Labels
			targetCarbonJob.Annotations = targetMetas.Annotations
			if err = c.manager.getCarbonJobAPIs().UpdateCarbonJob(targetCarbonJob); nil != err {
				return fmt.Errorf("UpdateCarbonJob failed, update err: %s", err)
			}
		}
	}
	for name := range targetCarbonJobs {
		carbonJob := targetCarbonJobs[name]
		_, ok := currentCarbonJobs[name]
		if !ok {
			if err = c.doCreateCarbonJob(appName, checksum, carbonJob); nil != err {
				return fmt.Errorf("doCreateCarbonJob failed, update err: %s", err)
			}
		}
	}
	return nil
}

func (c *k8sAdapter) generateCarbonJobs(appName, checksum string, groupPlans map[string]*typespec.GroupPlan) (string, map[string]*v1.CarbonJob, error) {
	// 按roleShortName聚合
	namespace, targetCarbonJobs, err := c.groupCarbonJobByRoleShortName(appName, checksum, groupPlans)
	if nil != err {
		return namespace, nil, err
	}
	// 生成workernodes template
	err = c.generateWorkerTemplates(appName, targetCarbonJobs)
	if nil != err {
		return namespace, nil, err
	}
	return namespace, targetCarbonJobs, nil
}

func (c *k8sAdapter) generateWorkerTemplates(appName string, targetCarbonJobs map[string]*v1.CarbonJob) error {
	ctx, cancel := context.WithTimeout(context.Background(), common.GetBatcherTimeout(2*time.Second))
	defer cancel()
	batcher := utils.NewBatcher(1000)
	commands := make([]*utils.Command, 0, len(targetCarbonJobs))
	for shortName := range targetCarbonJobs {
		targetCarbonJob := targetCarbonJobs[shortName]
		command, err := batcher.Go(ctx, false, convertCarbonJob, appName, targetCarbonJob)
		if err != nil {
			return fmt.Errorf("specConverter batcher go failed, err: %v", err)
		}
		commands = append(commands, command)
	}
	batcher.Wait()
	var errs = utils.NewErrors()
	for _, command := range commands {
		if command != nil {
			errs.Add(command.FuncError)
			errs.Add(command.ExecutorError)
		}
	}
	return errs.Error()
}

func convertCarbonJob(appName string, targetCarbonJob *v1.CarbonJob) error {
	var converter = transfer.NewWorkerNodeSpecConverter()
	workerResults, err := converter.Convert(targetCarbonJob)
	if err != nil {
		glog.Errorf("convert carbonjob to worker error: %s, %v", appName, err)
		return err
	}
	targetCarbonJob.Spec.JobPlan = &carbonv1.CarbonJobPlan{
		WorkerNodes: map[string]carbonv1.WorkerNodeTemplate{},
	}
	var workerNames = []string{}
	for workerName := range workerResults {
		worker := workerResults[workerName]
		workerNames = append(workerNames, workerName)
		targetCarbonJob.Spec.JobPlan.WorkerNodes[workerName] = carbonv1.WorkerNodeTemplate{
			ObjectMeta: worker.WorkerNode.ObjectMeta,
			Spec:       worker.WorkerNode.Spec,
			Immutable:  worker.Immutable,
		}
	}
	sort.StringSlice(workerNames).Sort()
	var versions = []string{}
	for i := range workerNames {
		versions = append(versions, workerResults[workerNames[i]].WorkerNode.Spec.Version)
	}
	targetCarbonJob.Spec.Checksum, _ = utils.SignatureWithMD5(versions)
	targetCarbonJob.Spec.AppPlan = nil
	return nil
}

func (c *k8sAdapter) groupCarbonJobByRoleShortName(appName, checksum string, groupPlans map[string]*typespec.GroupPlan) (string, map[string]*v1.CarbonJob, error) {
	var targetCarbonJobs = map[string]*v1.CarbonJob{}
	namespace, exclusiveMode, quota, err := c.getAdminInfos(appName, checksum)
	if err != nil {
		glog.Errorf("%s groupCarbonJobByRoleShortName with error: %v", appName, err)
		return namespace, nil, err
	}
	for groupName := range groupPlans {
		group := groupPlans[groupName]
		for roleName := range group.RolePlans {
			role := group.RolePlans[roleName]
			roleShortName := appName
			name := appName
			if nil != role.Version.ResourcePlan.MetaTags {
				if _, ok := (*role.Version.ResourcePlan.MetaTags)[carbonv1.LabelKeyRoleShortName]; ok {
					roleShortName = (*role.Version.ResourcePlan.MetaTags)[carbonv1.LabelKeyRoleShortName]
					name = appName + "." + roleShortName
				}
			}
			name = transfer.EscapeName(name)
			targetCarbonJob, ok := targetCarbonJobs[name]
			if ok {
				targetCarbonJob.Spec.AppPlan.Groups[groupName] = group
			} else {
				groups := map[string]*typespec.GroupPlan{groupName: group}
				targetCarbonJob = c.generateCarbonJob(appName, name, namespace, exclusiveMode, quota, checksum, roleShortName, groups)
				targetCarbonJobs[name] = targetCarbonJob
			}
		}
	}
	return namespace, targetCarbonJobs, nil
}

func (c *k8sAdapter) generateCarbonJob(appName, name, namespace, exclusiveMode, quota, checksum, roleShortName string, groupPlans map[string]*typespec.GroupPlan) *v1.CarbonJob {
	targetCarbonJob := &v1.CarbonJob{}
	targetCarbonJob.ObjectMeta = metav1.ObjectMeta{
		Name:      name,
		Namespace: namespace,
		Labels: map[string]string{
			carbonv1.LabelKeyClusterName:   c.cluster,
			carbonv1.LabelKeyAppNameHash:   carbonv1.LabelValueHash(appName, true),
			carbonv1.LabelKeyAppChecksum:   checksum,
			carbonv1.LabelKeyCarbonJobName: carbonv1.LabelValueHash(name, features.C2MutableFeatureGate.Enabled(features.HashLabelValue)),
		},
		Annotations: map[string]string{
			carbonv1.LabelKeyAppName:       appName,
			carbonv1.LabelKeyCarbonJobName: name,
		},
	}
	if quota != "" {
		targetCarbonJob.Labels[carbonv1.LabelKeyQuotaGroupID] = quota
	}
	if roleShortName != "" {
		targetCarbonJob.Annotations[carbonv1.LabelKeyRoleShortName] = roleShortName
	}
	if exclusiveMode != "" {
		targetCarbonJob.Labels[carbonv1.LabelKeyExclusiveMode] = exclusiveMode
	}
	carbonv1.SetObjectAppName(targetCarbonJob, appName)
	targetCarbonJob.Spec.AppName = appName
	targetCarbonJob.Spec.AppPlan = &carbonv1.CarbonAppPlan{
		Groups: groupPlans,
	}
	return targetCarbonJob
}

func (c *k8sAdapter) setCarbonJobGroup(appName, groupID string, groupPlan *typespec.GroupPlan) error {
	// not used
	return nil
}

// ReclaimWorkerNodePreference preference to reclaim worker node
type ReclaimWorkerNodePreference struct {
	Scope *string `json:"scope"`
	Type  *string `json:"type"`
	TTL   *int32  `json:"ttl"`
}

// ReclaimWorkerNodeRequest reclaim a list of worker nodes
type ReclaimWorkerNodeRequest struct {
	WorkerNodes []*carbonv1.ReclaimWorkerNode `json:"workerNodes"`
	Preference  *ReclaimWorkerNodePreference  `json:"preference"`
}

func (c *k8sAdapter) ReclaimWorkerNodesOnEviction(appName string, req *ReclaimWorkerNodeRequest) error {
	namespace, err := c.getNamespace(appName)
	if nil != err {
		glog.Errorf("reclaimWorkerNodesOnEviction failed, appName: %s getNamespace failed, err: %s", appName, err)
		return err
	}
	var pref *carbonv1.HippoPrefrence
	if nil != req.Preference {
		pref = &carbonv1.HippoPrefrence{
			Scope: req.Preference.Scope,
			Type:  req.Preference.Type,
			TTL:   req.Preference.TTL,
		}
	}
	spec, err := apiset.NewWorkerNodeEvictionSpec(appName, namespace, c.cluster, req.WorkerNodes, pref)
	if err != nil {
		glog.Errorf("Create new WorkerNodeEviction spec error: %s, %v", appName, err)
		return err
	}
	err = c.manager.getWorkerEvictionAPIs().SetWorkerNodeEviction(spec)
	return err
}

func (c *k8sAdapter) getWorkerNode(appName string, groupIDs []string) ([]*carbon.GroupStatus, error) {
	namespace, err := c.getNamespace(appName)
	if nil != err {
		return nil, err
	}
	workerNodes, err := c.loadCurrentWorkerNode(appName, groupIDs, namespace)
	if nil != err {
		return nil, err
	}
	pods, err := c.loadCurrentPod(appName, groupIDs, namespace)
	if nil != err {
		return nil, err
	}
	var groupStatusList []*carbon.GroupStatus
	groupStatusList, err = transfer.TransWorkerNodesToGroupStatusList(appName, workerNodes, pods)
	if nil != err {
		return nil, err
	}
	return groupStatusList, nil
}

func (c *k8sAdapter) loadCurrentPod(appName string, groupIDs []string, namespace string) ([]corev1.Pod, error) {
	selector, err := newHippoKeySelector(c.cluster, appName, "", "", groupIDs, nil)
	if nil != err {
		return nil, fmt.Errorf(
			"load exist pods make selector failed, app: %s, labels: %s",
			appName, utils.ObjJSON(selector),
		)
	}

	pods, err := c.manager.listPod(namespace, selector)
	if nil != err && !errors.IsNotFound(err) {
		return nil, fmt.Errorf("load exist pods failed, labels: %v, err: %s", selector, err)
	}
	return pods, nil
}

func (c *k8sAdapter) loadCurrentWorkerNode(appName string, groupIDs []string, namespace string) ([]carbonv1.WorkerNode, error) {
	selector, err := newHippoKeySelector(c.cluster, appName, "", "", groupIDs, nil)
	if nil != err {
		return nil, fmt.Errorf(
			"load exist workerNode make selector failed, app: %s, labels: %s",
			appName, utils.ObjJSON(selector),
		)
	}

	workerNodes, err := c.manager.listWorkerNode(namespace, selector)
	if nil != err && !errors.IsNotFound(err) {
		return nil, fmt.Errorf("load exist workerNode failed, labels: %v, err: %s", selector, err)
	}
	return workerNodes, nil
}
