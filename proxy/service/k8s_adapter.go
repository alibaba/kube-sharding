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
	"strings"
	"sync"
	"time"

	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	corev1 "k8s.io/api/core/v1"
	glog "k8s.io/klog"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	"github.com/alibaba/kube-sharding/pkg/apiserver-client/typespec"
	"github.com/alibaba/kube-sharding/pkg/apiserver-client/typespec/carbon"
	"github.com/alibaba/kube-sharding/transfer"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/labels"
)

const (
	alreadyExsitErr                   = "group name [%s] already exist"
	lruSize                           = 1024
	lruTimeout                        = 600 * time.Second
	defaultResourceMatchTimeout int64 = 60 * 5
	defaultProcessMatchTimeout  int64 = 60 * 10
	defaultWorkerReadyTimeout   int64 = 60 * 60
)

// SchTypes
const (
	SchTypeRole  = "role"
	SchTypeGroup = "group"
	SchTypeNode  = "node"
)

const (
	cloneSetKind       = "CloneSet"
	cloneSetApiVersion = "apps.kruise.io/v1alpha1"
)

// SchOptions options for scheduler.
type SchOptions struct {
	SchType     string `json:"schType"`
	AppChecksum string `json:"appChecksum"`
	ZKAddress   string `json:"zkAddress"`
}

func (s *SchOptions) validate() error {
	match := false
	for _, t := range []string{SchTypeRole, SchTypeGroup, SchTypeNode} {
		if t == s.SchType {
			match = true
			break
		}
	}
	if !match {
		return fmt.Errorf("invalid schType: %s", s.SchType)
	}
	return nil
}

// ParseSchOptions parse schedule options.
func ParseSchOptions(str string) (*SchOptions, error) {
	var opts SchOptions
	if "" == str {
		return &opts, nil
	}
	mapOpts := make(map[string]string)
	for _, kvs := range strings.Split(str, ";") {
		kvs := strings.Split(kvs, "=")
		if len(kvs) != 2 {
			continue
		}
		mapOpts[kvs[0]] = kvs[1]
	}
	if err := utils.JSONDeepCopy(mapOpts, &opts); err != nil {
		return nil, err
	}
	if err := opts.validate(); err != nil {
		return nil, err
	}
	return &opts, nil
}

type k8sAdapter struct {
	cluster string
	manager resourceManager
	syncer  k8sSyncer
}

func newGroupService(cluster string, manager resourceManager) GroupService {
	var k8sAdapter = k8sAdapter{
		cluster: cluster,
		manager: manager,
	}
	k8sAdapter.syncer = &k8sAdapter
	return &k8sAdapter
}

// only for mock test now
type k8sSyncer interface {
	deleteSingleRollingset(appName, groupID string) error
	deleteShardGroup(appName, groupID string) error
	syncShardGroupPlan(appName string, groupPlan typespec.GroupPlan, groups []carbonv1.ShardGroup) error
	syncRollingSetPlan(appName string, groupPlan typespec.GroupPlan, roles []carbonv1.RollingSet) error
	syncGangRSPlan(appName string, groupPlan typespec.GroupPlan, rss []carbonv1.RollingSet) error
}

func (c *k8sAdapter) DeleteGroup(appName, schType, groupID string) error {
	if SchTypeRole == schType {
		return c.deleteSingleRollingset(appName, groupID)
	} else if SchTypeNode == schType {
		return c.deleteCarbonJobGroup(appName, groupID)
	}
	return c.deleteShardGroup(appName, groupID)
}

func (c *k8sAdapter) deleteSingleRollingset(
	appName, groupID string) error {
	roleName := utils.AppendString(groupID, ".", groupID)
	rollingSets, err := c.manager.listRollingSet(appName, groupID, true)
	if nil != err && !errors.IsNotFound(err) {
		glog.Error(err)
		return err
	}
	for _, rollingSet := range rollingSets {
		if roleName != carbonv1.GetRoleName(&rollingSet) {
			glog.Errorf("roleName not match :%s, %s",
				rollingSet.Namespace, carbonv1.GetRoleName(&rollingSet))
			return nil
		}
		rollingsetName := rollingSet.Name
		services, err := c.manager.
			getRollingsetServices(appName, rollingSet.Namespace, rollingsetName)
		if nil != err {
			return err
		}
		for i := range services {
			err = c.manager.deleteService(services[i].Namespace, services[i].Name)
			if nil != err && !errors.IsNotFound(err) {
				return err
			}
		}
		err = c.manager.deleteRollingSet(rollingSet.Namespace, rollingSet.Name)
		glog.Infof("delete rollingsets :%s , %s , %v",
			rollingSet.Namespace, rollingSet.Name, err)
		if nil != err && !errors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func (c *k8sAdapter) deleteShardGroup(
	appName, groupID string) error {
	groups, err := c.manager.listGroup(appName, groupID)
	if nil != err && !errors.IsNotFound(err) {
		glog.Error(err)
		return err
	}
	if len(groups) == 0 && groupID != "" {
		rollingSets, err := c.manager.listRollingSet(appName, groupID, false)
		if nil != err && !errors.IsNotFound(err) {
			glog.Error(err)
			return err
		}
		if len(rollingSets) == 1 {
			rollingSet := rollingSets[0]
			if len(rollingSet.Spec.GangVersionPlan) != 0 {
				rollingsetName := rollingSet.Name
				services, err := c.manager.
					getRollingsetServices(appName, rollingSet.Namespace, rollingsetName)
				if nil != err {
					return err
				}
				for i := range services {
					err = c.manager.deleteService(services[i].Namespace, services[i].Name)
					if nil != err && !errors.IsNotFound(err) {
						return err
					}
				}
				err = c.manager.deleteRollingSet(rollingSet.Namespace, rollingSet.Name)
				glog.Infof("delete rollingsets :%s , %s , %v",
					rollingSet.Namespace, rollingSet.Name, err)
				if nil != err && !errors.IsNotFound(err) {
					return err
				}
			}
		}
	}
	for _, group := range groups {
		services, err := c.manager.getServiceByGroupname(appName, group.Namespace, group.Name)
		if nil != err {
			return err
		}
		for i := range services {
			err = c.manager.deleteService(services[i].Namespace, services[i].Name)
			if nil != err && !errors.IsNotFound(err) {
				return err
			}
		}
		err = c.manager.deleteShardGroup(group.Namespace, group.Name)
		glog.Infof("delete shard group :%s , %s , %v",
			group.Namespace, group.Name, err)
		if nil != err && !errors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func (c *k8sAdapter) NewGroup(
	appName string, schOpts *SchOptions, groupPlan *typespec.GroupPlan) error {
	c.tweakGroupPlan(schOpts.ZKAddress, groupPlan)
	if SchTypeRole == schOpts.SchType {
		return c.createSingleRollingset(appName, groupPlan)
	} else if SchTypeNode == schOpts.SchType {
		return c.createCarbonJobGroup(appName, schOpts.AppChecksum, groupPlan)
	}
	return c.createShardGroup(appName, groupPlan)
}

func (c *k8sAdapter) createSingleRollingset(appName string, groupPlan *typespec.GroupPlan) error {
	if nil == groupPlan {
		err := fmt.Errorf("nil group plan")
		return err
	}
	rollingSets, err := c.manager.
		listRollingSet(appName, groupPlan.GroupID, true)
	if nil != err && !errors.IsNotFound(err) {
		glog.Error(err)
		return err
	}
	if 0 != len(rollingSets) {
		err = fmt.Errorf(alreadyExsitErr, groupPlan.GroupID)
		return err
	}
	err = c.syncRollingSetPlan(appName, *groupPlan, rollingSets)
	return err
}

func (c *k8sAdapter) createShardGroup(appName string, groupPlan *typespec.GroupPlan) error {
	if groupPlan.SchedulerConfig != nil && groupPlan.SchedulerConfig.Name != nil && *groupPlan.SchedulerConfig.Name == carbonv1.ScheduleGangRolling {
		return c.syncGangRSPlan(appName, *groupPlan, nil)
	}
	err := c.syncShardGroupPlan(appName, *groupPlan, nil)
	return err
}

func (c *k8sAdapter) createGangRS(appName string, groupPlan *typespec.GroupPlan) error {
	err := c.syncGangRSPlan(appName, *groupPlan, nil)
	return err
}

func (c *k8sAdapter) SetGroups(appName string, schOpts *SchOptions, groupPlans map[string]*typespec.GroupPlan) error {
	c.tweakGroupPlans(schOpts.ZKAddress, groupPlans)
	if SchTypeRole == schOpts.SchType {
		return c.setSingleRollingsets(appName, groupPlans)
	} else if SchTypeNode == schOpts.SchType {
		return c.setCarbonJobGroups(appName, schOpts.AppChecksum, groupPlans)
	}
	return c.setShardGroups(appName, groupPlans)
}

func (c *k8sAdapter) setSingleRollingsets(appName string, groupPlans map[string]*typespec.GroupPlan) error {
	exists, err := c.manager.listRollingSet(appName, "", true)
	if nil != err {
		glog.Error(err)
		return err
	}
	roles := aggRollingSets(exists)
	executor := utils.NewExecutor(50)
	batcher := utils.NewBatcherWithExecutor(executor)
	commands := make([]*utils.Command, 0, len(groupPlans))
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()

	for _, rs := range exists {
		_, ok := groupPlans[carbonv1.GetGroupName(&rs)]
		if !ok {
			command, err := batcher.Go(
				ctx, false, c.syncer.deleteSingleRollingset, appName, carbonv1.GetGroupName(&rs))
			if nil != err {
				return err
			}
			commands = append(commands, command)
		}
	}
	for k := range groupPlans {
		hippoRoleName := transfer.GenerateHippoRoleName(k, k)
		command, err := batcher.Go(
			ctx, false, c.syncer.syncRollingSetPlan,
			appName, *groupPlans[k], roles[hippoRoleName])
		if nil != err {
			return err
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

func (c *k8sAdapter) tweakGroupPlans(zkAddr string, groups map[string]*typespec.GroupPlan) error {
	for _, group := range groups {
		err := c.tweakGroupPlan(zkAddr, group)
		if nil != err {
			return err
		}
	}
	return nil
}

func (c *k8sAdapter) tweakGroupPlan(zkAddr string, group *typespec.GroupPlan) error {
	return nil
}

func aggRollingSets(roles []carbonv1.RollingSet) map[string][]carbonv1.RollingSet {
	var result = make(map[string][]carbonv1.RollingSet)
	for i := range roles {
		name := carbonv1.GetRoleName(&roles[i])
		if exits, ok := result[name]; ok {
			exits = append(exits, roles[i])
			result[name] = exits
		} else {
			result[name] = []carbonv1.RollingSet{roles[i]}
		}
	}
	return result
}

func aggShardGroups(groups []carbonv1.ShardGroup) (map[string][]carbonv1.ShardGroup, error) {
	var result = make(map[string][]carbonv1.ShardGroup)
	for i := range groups {
		group := groups[i]
		groupID := carbonv1.GetGroupName(&group)
		if groupID == "" {
			err := fmt.Errorf("get not groupid %s", utils.ObjJSON(&group))
			glog.Error(err)
			return nil, err
		}
		if exits, ok := result[groupID]; ok {
			exits = append(exits, group)
			result[groupID] = exits
		} else {
			result[groupID] = []carbonv1.ShardGroup{group}
		}
	}
	return result, nil
}

func (c *k8sAdapter) setShardGroups(appName string, groupPlans map[string]*typespec.GroupPlan) error {
	exists, err := c.manager.listGroup(appName, "")
	if nil != err {
		glog.Error(err)
		return err
	}
	groups, err := aggShardGroups(exists)
	if err != nil {
		return err
	}
	executor := utils.NewExecutor(50)
	batcher := utils.NewBatcherWithExecutor(executor)
	commands := make([]*utils.Command, 0, len(groupPlans))
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()

	for _, rs := range exists {
		_, ok := groupPlans[carbonv1.GetGroupName(&rs)]
		if !ok {
			command, err := batcher.Go(
				ctx, false, c.syncer.deleteShardGroup,
				appName, carbonv1.GetGroupName(&rs))
			if nil != err {
				return err
			}
			commands = append(commands, command)
		}
	}
	for k := range groupPlans {
		command, err := batcher.Go(ctx, false, c.syncer.syncShardGroupPlan, appName, *groupPlans[k], groups[k])
		if nil != err {
			return err
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

func (c *k8sAdapter) SetGroup(appName, groupID string, schOpts *SchOptions, groupPlan *typespec.GroupPlan) error {
	c.tweakGroupPlan(schOpts.ZKAddress, groupPlan)
	if SchTypeRole == schOpts.SchType {
		return c.setSingleRollingset(appName, groupID, groupPlan)
	} else if SchTypeNode == schOpts.SchType {
		return c.setCarbonJobGroup(appName, groupID, groupPlan)
	}
	return c.setShardGroup(appName, groupID, groupPlan)
}

func (c *k8sAdapter) ScaleGroup(appName, groupID, zoneName string, scalePlan *carbonv1.ScaleSchedulePlan) error {
	rss, err := c.manager.listZoneRollingSet(appName, groupID, zoneName)
	if err != nil {
		return err
	}
	if len(rss) == 0 {
		err := fmt.Errorf("%s, %s, %s cat not match rollingsets", appName, groupID, zoneName)
		return err
	}
	for i := range rss {
		rs := rss[i].DeepCopy()
		if rs.Spec.ScaleSchedulePlan == nil {
			rs.Spec.ScaleSchedulePlan = &carbonv1.ScaleSchedulePlan{}
		}
		rs.Spec.ScaleSchedulePlan.Replicas = scalePlan.Replicas
		rs.Spec.ScaleSchedulePlan.RecycleMode = scalePlan.RecycleMode
		rs.Spec.ScaleConfig = &carbonv1.ScaleConfig{
			Enable: true,
		}
		if utils.ObjJSON(rs.Spec) != utils.ObjJSON(rss[i]) {
			err = c.manager.updateRollingset(rs.Namespace, rs)
			if err != nil {
				glog.Errorf("updateRollingset %s with error :%v ", rs.Name, err)
				return err
			}
		}
	}
	return nil
}

func (c *k8sAdapter) DisableScale(appName, groupID, zoneName string) error {
	rss, err := c.manager.listZoneRollingSet(appName, groupID, zoneName)
	if err != nil {
		return err
	}
	if len(rss) == 0 {
		err := fmt.Errorf("%s, %s, %s cat not match rollingsets", appName, groupID, zoneName)
		return err
	}
	for i := range rss {
		rs := rss[i].DeepCopy()
		rs.Spec.ScaleConfig = &carbonv1.ScaleConfig{
			Enable: false,
		}
		if utils.ObjJSON(rs.Spec) != utils.ObjJSON(rss[i]) {
			err = c.manager.updateRollingset(rs.Namespace, rs)
			if err != nil {
				glog.Errorf("updateRollingset %s with error :%v ", rs.Name, err)
				return err
			}
		}
	}
	return nil
}

func (c *k8sAdapter) setSingleRollingset(appName, groupID string, groupPlan *typespec.GroupPlan) error {
	rollingSets, err := c.manager.
		listRollingSet(appName, groupID, true)
	if nil != err && !errors.IsNotFound(err) {
		glog.Error(err)
		return err
	}

	err = c.syncRollingSetPlan(appName, *groupPlan, rollingSets)
	return err
}

func (c *k8sAdapter) setShardGroup(appName, groupID string, groupPlan *typespec.GroupPlan) error {
	if groupPlan.SchedulerConfig != nil && groupPlan.SchedulerConfig.Name != nil && *groupPlan.SchedulerConfig.Name == carbonv1.ScheduleGangRolling {
		return c.setGangRS(appName, groupID, groupPlan)
	}
	groups, err := c.manager.listGroup(appName, groupID)
	if nil != err && !errors.IsNotFound(err) {
		glog.Error(err)
		return err
	}
	err = c.syncShardGroupPlan(appName, *groupPlan, groups)
	return err
}

func (c *k8sAdapter) setGangRS(appName, groupID string, groupPlan *typespec.GroupPlan) error {
	roles, err := c.manager.listRollingSet(appName, groupID, false)
	if nil != err {
		glog.Error(err)
		return err
	}
	err = c.syncGangRSPlan(appName, *groupPlan, roles)
	return err
}

func (c *k8sAdapter) GetGroup(
	appName, schType string, groupIds []string,
) ([]*carbon.GroupStatus, error) {
	if SchTypeRole == schType {
		return c.getSingleRoleStatus(appName, groupIds)
	} else if SchTypeNode == schType {
		return c.getWorkerNode(appName, groupIds)
	}
	return c.getShardGroup(appName, groupIds)
}

func (c *k8sAdapter) getSingleRoleStatus(appName string, groupIds []string) ([]*carbon.GroupStatus, error) {
	rStatuses, err := c.getSingleRollingset(appName, groupIds)
	if err != nil {
		return nil, err
	}
	return rStatuses, nil
}

func (c *k8sAdapter) getSingleRollingset(
	appName string, groupIds []string) ([]*carbon.GroupStatus, error) {
	rollingSets, err := c.manager.getRollingsetByGroupIDs(appName, groupIds, true)
	if nil != err {
		return nil, err
	}
	var statuses = make([]*carbon.GroupStatus, 0, len(rollingSets))
	for i := range rollingSets {
		role := &rollingSets[i]
		groupID, roleID := transfer.GetGroupAndRoleID(role)
		status, err := c.getRoleStatus(role.Namespace, groupID, role)
		if nil != err {
			glog.Errorf("Get role status error: %s/%s %v", role.Namespace, groupID, err)
			return nil, err
		}
		var groupStatus = &carbon.GroupStatus{
			GroupId: &groupID,
			Roles: map[string]*carbon.RoleStatusValue{
				roleID: status,
			},
		}
		statuses = append(statuses, groupStatus)
	}
	return statuses, nil
}

func (c *k8sAdapter) getShardGroup(appName string, groupIDs []string) ([]*carbon.GroupStatus, error) {
	start := time.Now()
	rollingSets, err := c.manager.getRollingsetByGroupIDs(appName, groupIDs, false)
	if nil != err {
		return nil, err
	}
	var rollingSetsWithGang []carbonv1.RollingSet
	for i := range rollingSets {
		role := rollingSets[i]
		if len(role.Spec.GangVersionPlan) != 0 {
			for shardName := range role.Spec.GangVersionPlan {
				versionPlan := role.Spec.GangVersionPlan[shardName]
				rs := role.DeepCopy()
				carbonv1.AddGangPartSelectorHashKey(rs.Spec.Selector, shardName)
				roleName := carbonv1.GetRoleName(versionPlan.Template)
				carbonv1.SetRoleName(rs, roleName)
				rollingSetsWithGang = append(rollingSetsWithGang, *rs)
			}
			continue
		}
		rollingSetsWithGang = append(rollingSetsWithGang, role)
	}
	rollingSets = rollingSetsWithGang
	var statuss = make([]*carbon.GroupStatus, 0, len(rollingSets))
	var groupStatuss = make(map[string]*carbon.GroupStatus)
	var groupStatusLock sync.Mutex
	executor := utils.NewExecutor(256)
	batcher := utils.NewBatcherWithExecutor(executor)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(opts.timeout))
	defer cancel()
	commands := make([]*utils.Command, 0)
	for i := range rollingSets {
		role := &rollingSets[i]
		command, err := batcher.Go(ctx, false, func() error {
			groupID, roleID := transfer.GetGroupAndRoleID(role)
			status, err := c.getRoleStatus(role.Namespace, groupID, role)
			if nil != err {
				glog.Error(err)
				return err
			}
			groupStatusLock.Lock()
			if groupStatus, ok := groupStatuss[groupID]; ok {
				groupStatus.Roles[roleID] = status
			} else {
				var groupStatus = carbon.GroupStatus{
					GroupId: &groupID,
					Roles: map[string]*carbon.RoleStatusValue{
						roleID: status,
					},
				}
				groupStatuss[groupID] = &groupStatus
			}
			groupStatusLock.Unlock()
			return nil
		})
		if nil != err {
			glog.Errorf("Get rollingset error %s, %v", rollingSets[i].Name, err)
			return nil, err
		}
		commands = append(commands, command)
	}
	batcher.Wait()
	if glog.V(4) {
		glog.Infof("get shardgroups %s, used %d ms", groupIDs, time.Now().Sub(start).Nanoseconds()/1000/1000)
	}
	var errs = utils.NewErrors()
	for _, command := range commands {
		errs.Add(command.FuncError)
		errs.Add(command.ExecutorError)
	}
	if nil != errs.Error() {
		glog.Errorf("get shardgroups error %s,%v", groupIDs, errs.Error())
		return nil, errs.Error()
	}
	for k := range groupStatuss {
		groupStatus := groupStatuss[k]
		statuss = append(statuss, groupStatus)
	}
	return statuss, nil
}

/*
	func (c *k8sAdapter) getGangRs(appName string, rollingSet *carbonv1.RollingSet) ([]*carbon.GroupStatus, error) {
		start := time.Now()
		var shardRses = make([]*carbonv1.RollingSet, 0, len(rollingSet.Spec.GangVersionPlan))
		for shardName := range rollingSet.Spec.GangVersionPlan {
			rs := rollingSet.DeepCopy()
			carbonv1.AddGangPartSelectorHashKey(rs.Spec.Selector, shardName)
			shardRses = append(shardRses, rs)
		}
		var statuss = make([]*carbon.GroupStatus, 0, len(shardRses))
		var groupStatuss = make(map[string]*carbon.GroupStatus)
		var groupStatusLock sync.Mutex
		executor := utils.NewExecutor(256)
		batcher := utils.NewBatcherWithExecutor(executor)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(opts.timeout))
		defer cancel()
		commands := make([]*utils.Command, 0)
		for i := range shardRses {
			role := shardRses[i]
			command, err := batcher.Go(ctx, false, func() error {
				groupID, roleID := transfer.GetGroupAndRoleID(role)
				status, err := c.getRoleStatus(role.Namespace, groupID, role)
				if nil != err {
					glog.Error(err)
					return err
				}
				groupStatusLock.Lock()
				if groupStatus, ok := groupStatuss[groupID]; ok {
					groupStatus.Roles[roleID] = status
				} else {
					var groupStatus = carbon.GroupStatus{
						GroupId: &groupID,
						Roles: map[string]*carbon.RoleStatusValue{
							roleID: status,
						},
					}
					groupStatuss[groupID] = &groupStatus
				}
				groupStatusLock.Unlock()
				return nil
			})
			if nil != err {
				glog.Errorf("Get rollingset error %s, %v", shardRses[i].Name, err)
				return nil, err
			}
			commands = append(commands, command)
		}
		batcher.Wait()
		if glog.V(4) {
			glog.Infof("get shardgroups %s, used %d ms", rollingSet.Name, time.Now().Sub(start).Nanoseconds()/1000/1000)
		}
		var errs = utils.NewErrors()
		for _, command := range commands {
			errs.Add(command.FuncError)
			errs.Add(command.ExecutorError)
		}
		if nil != errs.Error() {
			glog.Errorf("get shardgroups error %s,%v", rollingSet.Name, errs.Error())
			return nil, errs.Error()
		}
		for k := range groupStatuss {
			groupStatus := groupStatuss[k]
			statuss = append(statuss, groupStatus)
		}
		return statuss, nil
	}
*/
func (c *k8sAdapter) getRoleStatus(namespace, groupID string, rollingSet *carbonv1.RollingSet) (*carbon.RoleStatusValue, error) {
	start := time.Now()
	selector, err := metav1.LabelSelectorAsSelector(rollingSet.Spec.Selector)
	if err != nil {
		return nil, err
	}

	var workerSets = map[string][]*carbonv1.WorkerNode{}
	var podSets = map[string]*corev1.Pod{}
	executor := utils.NewExecutor(50)
	batcher := utils.NewBatcherWithExecutor(executor)
	ctx, cancel := context.WithTimeout(
		context.Background(), time.Second*time.Duration(opts.timeout))
	defer cancel()
	commands := make([]*utils.Command, 0)
	command, err := batcher.Go(ctx, false, func() error {
		workers, err := c.manager.listWorkerNode(namespace, selector)
		if nil != err && !errors.IsNotFound(err) {
			glog.Errorf("List workerNodes failed: %s, %v", namespace, err)
			return err
		}
		for i := range workers {
			replicaName := carbonv1.GetReplicaID(&workers[i])
			workerPairs := workerSets[replicaName]
			if nil == workerPairs {
				workerPairs = []*carbonv1.WorkerNode{workers[i].DeepCopy()}
			} else {
				workerPairs = append(workerPairs, workers[i].DeepCopy())
			}
			workerSets[replicaName] = workerPairs
		}
		if glog.V(4) {
			glog.Infof("get role workers %s, %s, %s, %d, used %d ms",
				groupID, namespace, selector.String(), len(workers), time.Now().Sub(start).Nanoseconds()/1000/1000)
		}
		return nil
	})
	if nil != err {
		glog.Errorf("Get workers error %s, %v", namespace, err)
		return nil, err
	}
	commands = append(commands, command)
	command, err = batcher.Go(ctx, false, func() error {
		listPodStart := time.Now()
		pods, err := c.manager.listPod(namespace, selector)
		if nil != err && !errors.IsNotFound(err) {
			glog.Errorf("List pod error: %s, %v", namespace, err)
			return err
		}
		for i := range pods {
			podSets[pods[i].Name] = &pods[i]
		}
		if glog.V(4) {
			glog.Infof("get role pods %s, %s, %s, %d, used %d ms",
				groupID, namespace, selector.String(), len(pods), time.Now().Sub(listPodStart).Nanoseconds()/1000/1000)
		}
		return nil
	})
	if nil != err {
		glog.Errorf("Get pods error %s, %v", namespace, err)
		return nil, err
	}
	commands = append(commands, command)
	batcher.Wait()
	if glog.V(4) {
		glog.Infof("get role replicas %s, used %d ms", groupID, time.Now().Sub(start).Nanoseconds()/1000/1000)
	}
	var errs = utils.NewErrors()
	for _, command := range commands {
		errs.Add(command.FuncError)
		errs.Add(command.ExecutorError)
	}
	if nil != errs.Error() {
		glog.Errorf("get replicas and workers error %s,%v", namespace, err)
		return nil, errs.Error()
	}

	var replicaNodes = make([]transfer.ReplicaNodes, 0, len(workerSets))
	for _, workers := range workerSets {
		currentWorker, backupWorker, _ := carbonv1.GetCurrentWorkerNodeV2(workers, true)
		currPodName, _ := carbonv1.GetPodName(currentWorker)
		pod := podSets[currPodName]
		var backupPod *corev1.Pod
		if backupWorker != nil {
			backPodName, _ := carbonv1.GetPodName(backupWorker)
			backupPod = podSets[backPodName]
		}
		var replicaNode = transfer.ReplicaNodes{
			Worker:       currentWorker,
			Pod:          pod,
			BackupWorker: backupWorker,
			BackupPod:    backupPod,
		}
		replicaNodes = append(replicaNodes, replicaNode)
	}
	status, err := transfer.TransRoleStatus(groupID, rollingSet, replicaNodes)
	if glog.V(4) {
		glog.Infof("get role transed %s, used %d ms", groupID, time.Now().Sub(start).Nanoseconds()/1000/1000)
	}
	return status, err
}

// TODO 目前是多次并行查询，遇到大app还是比较慢，需要改成一次批量查询
func (c *k8sAdapter) GetAppGroup(appName, schType string) ([]*carbon.GroupStatus, error) {
	if SchTypeRole == schType {
		return c.getAppAllSingleRollingset(appName)
	}
	return c.getShardGroup(appName, []string{})
}

func (c *k8sAdapter) getAppAllSingleRollingset(appName string) ([]*carbon.GroupStatus, error) {
	rollingSets, err := c.manager.getRollingsetByGroupIDs(appName, []string{}, true)
	if nil != err {
		return nil, err
	}
	var roleStatusLock sync.Mutex
	var statuses = make([]*carbon.GroupStatus, 0, len(rollingSets))
	executor := utils.NewExecutor(200)
	batcher := utils.NewBatcherWithExecutor(executor)
	ctx, cancel := context.WithTimeout(
		context.Background(), time.Second*time.Duration(opts.timeout))
	defer cancel()
	commands := make([]*utils.Command, 0)
	for i := range rollingSets {
		role := &rollingSets[i]
		command, err := batcher.Go(ctx, false, func() error {
			groupID, roleID := transfer.GetGroupAndRoleID(role)
			status, err := c.getRoleStatus(role.Namespace, groupID, role)
			if nil != err {
				glog.Errorf("Get role status error: %s/%s %v", role.Namespace, groupID, err)
				return err
			}
			var groupStatus = &carbon.GroupStatus{
				GroupId: &groupID,
				Roles: map[string]*carbon.RoleStatusValue{
					roleID: status,
				},
			}
			roleStatusLock.Lock()
			glog.V(5).Infof("get group status %s, %s", groupID, roleID)
			statuses = append(statuses, groupStatus)
			roleStatusLock.Unlock()
			return nil
		})
		if nil != err {
			glog.Errorf("Get rollingset error %s, %v", role.Name, err)
			return nil, err
		}
		commands = append(commands, command)
	}
	batcher.Wait()
	var errs = utils.NewErrors()
	for _, command := range commands {
		errs.Add(command.FuncError)
		errs.Add(command.ExecutorError)
	}
	if nil != errs.Error() {
		glog.Errorf("get roles error %s,  %v", appName, err)
		return nil, errs.Error()
	}
	return statuses, nil
}

func (c *k8sAdapter) getNamespaces(kind, appName string, namespaces []string, groupPlan typespec.GroupPlan) ([]string, error) {
	var err error
	namespaces, err = c.manager.checkNamespace(appName, namespaces)
	if nil != err {
		glog.Error(err)
		return nil, err
	}
	return namespaces, nil
}

func (c *k8sAdapter) syncShardGroupPlan(appName string, groupPlan typespec.GroupPlan, groups []carbonv1.ShardGroup) error {
	var namespaces = []string{}
	for _, obj := range groups {
		namespaces = append(namespaces, obj.GetNamespace())
	}
	namespaces, err := c.getNamespaces("ShardGroup", appName, namespaces, groupPlan)
	if err != nil {
		return err
	}
	shardgroupName, err := c.manager.getShardGroupName(appName, namespaces[0], groupPlan.GroupID)
	if err != nil {
		return err
	}
	shardGroup, services, err := transfer.
		TransGroupToShardGroup(c.cluster, appName, &groupPlan, shardgroupName, len(groups) == 0 || c.isGroupPodVersionV3(&groups[0]))
	if nil != err {
		glog.Error(err)
		return err
	}

	start := time.Now()
	for _, namespace := range namespaces {
		key := namespace + "/" + groupPlan.GroupID
		err = c.manager.createNamespace(namespace)
		if nil != err {
			return err
		}
		if glog.V(4) {
			glog.Infof("complete trans %s rollingset after %d ms",
				key, time.Now().Sub(start).Nanoseconds()/1000/1000)
		}
		oldServices, err := c.manager.
			getServiceByGroupname(appName, namespace, shardGroup.Name)
		if nil != err {
			return err
		}

		if glog.V(4) {
			glog.Infof("complete sync %s rollingset after %d ms",
				key, time.Now().Sub(start).Nanoseconds()/1000/1000)
		}
		err = c.manager.syncShardGroup(namespace, shardGroup)
		if nil != err {
			glog.Error(err)
			return err
		}

		oldServices = c.filterDeleteRoleService(oldServices, shardGroup)
		err = c.manager.syncServices(namespace, oldServices, services)
		if nil != err {
			glog.Error(err)
			return err
		}
		if glog.V(4) {
			glog.Infof("complete sync %s shardGroup after %d ms",
				key, time.Now().Sub(start).Nanoseconds()/1000/1000)
		}
	}
	return nil
}

func (c *k8sAdapter) syncGangRSPlan(appName string, groupPlan typespec.GroupPlan, rss []carbonv1.RollingSet) error {
	var namespaces = []string{}
	for _, obj := range rss {
		namespaces = append(namespaces, obj.GetNamespace())
	}
	namespaces, err := c.getNamespaces("ShardGroup", appName, namespaces, groupPlan)
	if err != nil {
		return err
	}
	rsName, err := c.manager.getGangRollingSetName(appName, namespaces[0], groupPlan.GroupID)
	if err != nil {
		return err
	}
	rs, services, err := transfer.
		TransGroupToGangRollingset(c.cluster, appName, &groupPlan, rsName)
	if nil != err {
		glog.Error(err)
		return err
	}

	start := time.Now()
	for _, namespace := range namespaces {
		key := namespace + "/" + groupPlan.GroupID
		err = c.manager.createNamespace(namespace)
		if nil != err {
			return err
		}
		if glog.V(4) {
			glog.Infof("complete trans %s rollingset after %d ms",
				key, time.Now().Sub(start).Nanoseconds()/1000/1000)
		}
		oldServices, err := c.manager.getServiceByGroupname(appName, namespace, rs.Name)
		if nil != err {
			return err
		}

		if glog.V(4) {
			glog.Infof("complete sync %s rollingset after %d ms",
				key, time.Now().Sub(start).Nanoseconds()/1000/1000)
		}
		err = c.manager.syncRollingset(namespace, rs)
		if nil != err {
			glog.Error(err)
			return err
		}
		err = c.manager.syncServices(namespace, oldServices, services)
		if nil != err {
			glog.Error(err)
			return err
		}
		if glog.V(4) {
			glog.Infof("complete sync %s shardGroup after %d ms",
				key, time.Now().Sub(start).Nanoseconds()/1000/1000)
		}
	}
	return nil
}

func (c *k8sAdapter) isGroupPodVersionV3(group *carbonv1.ShardGroup) bool {
	shardGroupCopy := group.DeepCopy()
	err := carbonv1.DecompressShardTemplates(shardGroupCopy)
	if err != nil {
		glog.Warningf("shardgroup DecompressShardTemplates err %s, %v", group.Name, err)
		return false
	}
	for k := range shardGroupCopy.Spec.ShardTemplates {
		temp := shardGroupCopy.Spec.ShardTemplates[k]
		if carbonv1.GetPodVersion(&temp) == carbonv1.PodVersion3 {
			return true
		}
	}
	return false
}

func (c *k8sAdapter) filterDeleteRoleService(services []carbonv1.ServicePublisher, group *carbonv1.ShardGroup) []carbonv1.ServicePublisher {
	var filteredService = []carbonv1.ServicePublisher{}
	roleNames := carbonv1.GetRollingsetNameHashs(group)
	for i := range services {
		service := services[i]
		roleName := carbonv1.GetRsID(&service)
		if roleNames[roleName] {
			filteredService = append(filteredService, service)
		} else {
			glog.Infof("filter service: %s, %s", service.Name, roleName)
		}
	}
	return filteredService
}

func (c *k8sAdapter) syncRollingSetPlan(appName string, groupPlan typespec.GroupPlan, roles []carbonv1.RollingSet) error {
	var namespaces = []string{}
	for _, obj := range roles {
		namespaces = append(namespaces, obj.GetNamespace())
	}
	namespaces, err := c.getNamespaces("RollingSet", appName, namespaces, groupPlan)
	if err != nil {
		return err
	}
	rollingSetName, err := c.manager.getRollingSetName(appName, namespaces[0], groupPlan.GroupID)
	if err != nil {
		return err
	}
	specs, err := transfer.TransGroupToRollingsets(c.cluster, appName, rollingSetName, &groupPlan, len(roles) == 0 || carbonv1.IsV3(&roles[0]))
	if nil != err {
		glog.Error(err)
		return err
	}
	for i := range specs {
		start := time.Now()
		for _, namespace := range namespaces {
			err = c.manager.createNamespace(namespace)
			if nil != err {
				return err
			}
			if glog.V(4) {
				glog.Infof("complete trans %s,%s,%s rollingset after %d ms",
					appName, namespace, groupPlan.GroupID,
					time.Now().Sub(start).Nanoseconds()/1000/1000)
			}
			oldServices, err := c.manager.
				getRollingsetServices(appName, namespace, specs[i].Rollingset.Name)
			if nil != err {
				return err
			}
			if glog.V(4) {
				glog.Infof("complete sync %s,%s,%s rollingset after %d ms",
					appName, namespace, groupPlan.GroupID,
					time.Now().Sub(start).Nanoseconds()/1000/1000)
			}
			err = c.manager.syncRollingset(namespace, specs[i].Rollingset)
			if nil != err {
				glog.Error(err)
				return err
			}
			err = c.manager.syncServices(namespace, oldServices, specs[i].Services)
			if nil != err {
				glog.Error(err)
				return err
			}
			if glog.V(4) {
				glog.Infof("complete sync %s,%s,%s services after %d ms",
					appName, namespace, groupPlan.GroupID,
					time.Now().Sub(start).Nanoseconds()/1000/1000)
			}
		}
	}
	return nil
}

func parseStrSelector(selectorStr string) (labels.Selector, error) {
	labelselector, err := metav1.ParseToLabelSelector(selectorStr)
	if nil != err {
		glog.Warningf("create service lister err: %v", err)
		return nil, err
	}
	selector, err := metav1.LabelSelectorAsSelector(labelselector)
	if nil != err {
		glog.Warningf("create service lister err: %v", err)
		return nil, err
	}
	return selector, nil
}
