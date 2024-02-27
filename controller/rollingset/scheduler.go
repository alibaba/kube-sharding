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
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"runtime"
	"strings"

	mapset "github.com/deckarep/golang-set"

	"reflect"
	"sort"
	"strconv"
	"time"

	"github.com/alibaba/kube-sharding/common"
	"github.com/alibaba/kube-sharding/common/features"
	"github.com/alibaba/kube-sharding/controller"
	"github.com/alibaba/kube-sharding/controller/util"
	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	v1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	"github.com/alibaba/kube-sharding/pkg/rollalgorithm"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	glog "k8s.io/klog"
	k8scontroller "k8s.io/kubernetes/pkg/controller"
	"k8s.io/utils/integer"
)

type scheduler interface {
	schedule() error
}

var metricDate = ""

// InPlaceScheduler schedule the replicas in place
type InPlaceScheduler struct {
	rsKey              string
	scheduleID         string
	rollingSet         *carbonv1.RollingSet
	groupRollingSet    *carbonv1.RollingSet
	rollingSetStatus   *carbonv1.RollingSetStatus
	replicas           []*carbonv1.Replica
	resourceManager    *util.ResourceManager
	generalLabels      map[string]string
	generalAnnotations map[string]string
	expectations       *k8scontroller.UIDTrackingControllerExpectations
	nodescheduler      rollalgorithm.RollingUpdater
	c                  *Controller
}

func newScheduler(rsKey string, scheduleID string, rollingSet *carbonv1.RollingSet, groupRollingSet *carbonv1.RollingSet,
	replicas []*carbonv1.Replica, resourceManager *util.ResourceManager,
	expectations *k8scontroller.UIDTrackingControllerExpectations, c *Controller,
) *InPlaceScheduler {
	return newInPlaceScheduler(rsKey, scheduleID, rollingSet, groupRollingSet, replicas, resourceManager, expectations, c)
}

func newInPlaceScheduler(rsKey string, scheduleID string, rollingSet *carbonv1.RollingSet, groupRollingSet *carbonv1.RollingSet,
	replicas []*carbonv1.Replica, resourceManager *util.ResourceManager,
	expectations *k8scontroller.UIDTrackingControllerExpectations, c *Controller,
) *InPlaceScheduler {
	return &InPlaceScheduler{
		rsKey:            rsKey,
		rollingSet:       rollingSet,
		groupRollingSet:  groupRollingSet,
		rollingSetStatus: rollingSet.Status.DeepCopy(),
		replicas:         replicas,
		resourceManager:  resourceManager,
		scheduleID:       scheduleID,
		expectations:     expectations,
		nodescheduler:    rollalgorithm.NewDefaultNodeSetUpdater(c.diffLogger),
		c:                c,
	}
}

type rollingsetArgs struct {
	//target算法需要
	latestVersionPlan *carbonv1.VersionPlanWithGangPlan // final plan
	latestResVersion  string
	//	benchmarkVersion  string
	_replicaSet           []*carbonv1.Replica
	_resChangedReplicaSet []*carbonv1.Replica

	versionReplicaMap        map[string][]*carbonv1.Replica
	versionPlanMap           map[string]*carbonv1.VersionPlanWithGangPlan
	resVersionReplicaMap     map[string][]*carbonv1.Replica
	resVersionPlanMap        map[string]*carbonv1.VersionPlan
	groupVersionToVersionMap map[string]string
	//deepcopy 可以直接修改
	supplementReplicaSet []*carbonv1.Replica //升级或者创建的replica对象
	supplements          mapset.Set

	// scale parameters
	standByReplicas   []*carbonv1.Replica
	activeReplicas    []*carbonv1.Replica
	scaledParameters  *carbonv1.ScaleParameters
	releasingReplicas []*carbonv1.Replica
	//version 非version字段变更replicas
	broadcastReplicaSet []*carbonv1.Replica

	replicaNames []string

	*rollalgorithm.ShardScheduleParams
	rollingUpdateArgs    *rollalgorithm.RollingUpdateArgs
	inactiveStandbyNodes []*carbonv1.Replica

	predictInfo      map[int64]int64
	fixedCount       int64
	fixedReplicas    []*carbonv1.Replica
	cyclicalReplicas []*carbonv1.Replica

	spotTarget       int64
	curSpotNames     []string
	nonVersionedKeys mapset.Set
}

func (s *InPlaceScheduler) schedule() error {
	err := s.generateMetas()
	if nil != err {
		return err
	}
	rollingsetArgs, err := s.initArgs()
	if nil != err {
		glog.Errorf("initArgs error : %s, %v", s.rollingSet.Name, err)
		return err
	}
	redundantReplicas := s.adjust(&rollingsetArgs)
	if 0 != len(redundantReplicas) || 0 != len(rollingsetArgs.supplementReplicaSet) {
		glog.Infof("[%s] rollingset schedule, redundantReplica count = %v, supplementReplica count = %v",
			s.scheduleID, len(redundantReplicas), len(rollingsetArgs.supplementReplicaSet))
	}
	//replica, rollingset状态同步
	return s.sync(&rollingsetArgs, &redundantReplicas)
}

func (s *InPlaceScheduler) adjust(rollingsetArgs *rollingsetArgs) []*carbonv1.Replica {
	start := time.Now()
	createNodes, updateNodes, deleteNodes, err := s.nodescheduler.Schedule(rollingsetArgs.rollingUpdateArgs)
	if nil != err {
		glog.Errorf("NodeSetSchedule error %v, and wait for a new changed event", err)
		return nil
	}
	s.addNodeSetMapToSupplement(rollingsetArgs, createNodes, true)
	s.addNodeSetMapToSupplement(rollingsetArgs, updateNodes, false)
	redundantReplicas := nodeSetToReplicas(deleteNodes)
	redundantReplicas = append(redundantReplicas, rollingsetArgs.inactiveStandbyNodes...)
	rollingsetArgs.activeReplicas = carbonv1.SubReplicas(rollingsetArgs._replicaSet, redundantReplicas)
	for k := range createNodes {
		nodes := createNodes[k]
		replicas := nodeSetToReplicas(nodes)
		rollingsetArgs.activeReplicas = append(rollingsetArgs.activeReplicas, replicas...)
	}
	target := len(rollingsetArgs._replicaSet) - len(redundantReplicas)
	s.initSpotArgs(rollingsetArgs, target)
	s.supplementStandby(rollingsetArgs, &redundantReplicas)
	s.calcStandbyHours(rollingsetArgs)
	s.syncSpotReplicas(rollingsetArgs, redundantReplicas)
	elapsed := time.Since(start)
	if glog.V(4) {
		glog.Infof("[%s] rollingset adjust rs.name: %v, replica count: %v, targetReplicas:%d, elapsed: %v",
			s.scheduleID, s.rollingSet.Name, len(rollingsetArgs._replicaSet), target, elapsed)
	}
	if glog.V(5) {
		glog.Infof("[%s] rollingset adjust rs.name: %v, replica names: %v", s.scheduleID, s.rollingSet.Name, rollingsetArgs.replicaNames)
	}
	rollingAdjustLatency.WithLabelValues(
		carbonv1.GetObjectBaseScopes(s.rollingSet)...,
	).Observe(float64(elapsed.Nanoseconds() / 1000000))

	return redundantReplicas
}

func (s *InPlaceScheduler) calcStandbyHours(args *rollingsetArgs) {
	if args.fixedCount == 0 {
		return
	}
	sort.Sort(ByDeletionCost(args.cyclicalReplicas))
	sort.Sort(ByDeletionCost(args.fixedReplicas))
	replicas := v1.MergeReplica(args.fixedReplicas, args.cyclicalReplicas)
	hour := time.Now().Hour()
	standbyHours := make([][]int64, 0, len(replicas))
	for i := 0; i < len(replicas); i++ {
		standbyHours = append(standbyHours, []int64{})
	}
	for i := hour + 1; i <= hour+24; i++ {
		futureHours := int64(i)
		if futureHours >= 24 {
			futureHours -= 24
		}
		desired := args.predictInfo[futureHours]
		if desired == 0 || int(desired) >= len(replicas) {
			continue
		}
		for j := int(desired); j < len(replicas); j++ {
			standbyHours[j] = append(standbyHours[j], futureHours)
		}
	}

	for i, replica := range replicas {
		var statusHours []int64
		specHours := standbyHours[i]
		if replica.Status.StandbyHours != "" {
			_ = json.Unmarshal([]byte(replica.Status.StandbyHours), &statusHours)
			sort.Sort(Int64Slice(statusHours))
		}
		sort.Sort(Int64Slice(specHours))
		if v1.SliceEquals(statusHours, specHours) {
			continue
		}
		replica.Spec.StandbyHours = utils.ObjJSON(specHours)
		addSupplement(args, replica)
	}
}

type poolState struct {
	poolType       string
	curMinCost     int64
	poolMinCost    int64
	targetCnt      int
	currentCnt     int
	suppliedCnt    int
	costBitMap     *util.BitMap
	curBitMinIndex int64
}

func (ps *poolState) getNextCost() int64 {
	if ps.curMinCost > ps.poolMinCost {
		ps.curMinCost--
		ps.costBitMap.Add(uint(ps.curMinCost - ps.poolMinCost))
		return ps.curMinCost
	}
	for i := ps.curBitMinIndex; i >= 0; i-- {
		if !ps.costBitMap.IsExist(uint(i)) {
			ps.curBitMinIndex = i
			ps.costBitMap.Add(uint(i))
			return ps.poolMinCost + ps.curBitMinIndex
		}
	}
	return ps.curMinCost
}

func setCostAndPool(args *rollingsetArgs, state *poolState, candidates, replicas []*carbonv1.Replica) {
	if state.currentCnt > state.targetCnt {
		return
	}
	for i := range replicas {
		replica := replicas[i]
		if v1.GetDeletionCost(replica) != 0 {
			continue
		}
		replica.Spec.DeletionCost = state.getNextCost()
		addSupplement(args, replica)
	}
	for i := 0; i < (state.targetCnt-state.currentCnt) && i < len(candidates); i++ {
		replica := candidates[i]
		replica.Spec.ResourcePool = state.poolType
		replica.Spec.DeletionCost = state.getNextCost()
		state.suppliedCnt++
		addSupplement(args, replica)
	}
}

// rollingset, replica状态同步
func (s *InPlaceScheduler) sync(rollingsetArgs *rollingsetArgs, redundantReplicas *[]*carbonv1.Replica) error {
	//释放节点
	if err := s.releaseReplicas(redundantReplicas); err != nil {
		return err
	}
	//移除已释放节点
	if err := s.removeReleasedReplicaNodes(rollingsetArgs); err != nil {
		return err
	}
	rollingsetArgs.supplementReplicaSet = carbonv1.SubReplicas(
		rollingsetArgs.supplementReplicaSet, *redundantReplicas)
	rollingsetArgs.broadcastReplicaSet = carbonv1.SubReplicas(
		rollingsetArgs.broadcastReplicaSet, *redundantReplicas)
	//处理supplementReplicaSet中的replica。 update/create
	if err := s.syncReplicas(rollingsetArgs); err != nil {
		return err
	}
	//更新replica版本
	if err := s.syncReplicaResource(rollingsetArgs, redundantReplicas); err != nil {
		return err
	}
	// 完成同步，记录metric
	compete := s.getCompleteStatus(rollingsetArgs)
	serviceReady := s.getServiceReadyStatus(rollingsetArgs)
	uncompleteStatusMap := s.getUncompleteReplicaStatus(rollingsetArgs._replicaSet, rollingsetArgs.LatestVersion)
	spotStatus := v1.SpotStatus{Target: rollingsetArgs.spotTarget, Instances: rollingsetArgs.curSpotNames}
	return s.syncRollingsetStatus(&rollingsetArgs._replicaSet, compete, serviceReady, spotStatus, uncompleteStatusMap)
}

func (s *InPlaceScheduler) initArgs() (rollingsetArgs, error) {
	rollingsetArgs, err := s.checkArgs() //调度参数check
	if err != nil {
		return rollingsetArgs, err
	}
	rollingsetArgs.nonVersionedKeys, err = v1.GetNonVersionedKeys()
	if err != nil {
		return rollingsetArgs, err
	}
	err = s.initScheduleParams(&rollingsetArgs)
	if err != nil {
		return rollingsetArgs, err
	}
	s.initResVersionMap(&rollingsetArgs)
	err = s.updateReplicaResource(&rollingsetArgs)
	if nil != err {
		return rollingsetArgs, err
	}
	s.initVersionArgs(&rollingsetArgs)
	s.updateBroadcastField(&rollingsetArgs)
	rollingsetArgs.scaledParameters = s.rollingSet.GetScaleParameters()
	if s.rollingSet.Spec.PredictInfo != "" {
		predict := make(map[int64]int64)
		if err = json.Unmarshal([]byte(s.rollingSet.Spec.PredictInfo), &predict); err != nil {
			return rollingsetArgs, err
		}
		rollingsetArgs.predictInfo = predict
	}
	rollingsetArgs.fixedCount = getFixedResourcePoolCnt(rollingsetArgs.predictInfo, s.rollingSet.Spec.Replicas)
	rollingsetArgs.supplements = mapset.NewThreadUnsafeSet()
	return rollingsetArgs, nil
}

func getFixedResourcePoolCnt(predictInfo map[int64]int64, planReplicas *int32) int64 {
	fixedCnt := carbonv1.MaxFixedCost
	if planReplicas != nil {
		fixedCnt = int64(*planReplicas)
	}
	for hour := range predictInfo {
		replicas := predictInfo[hour]
		if fixedCnt > replicas {
			fixedCnt = replicas
		}
	}
	return fixedCnt
}

func (s *InPlaceScheduler) generateMetas() error {
	template, err := carbonv1.GetGeneralTemplate(s.rollingSet.Spec.VersionPlan)
	if err != nil {
		glog.Errorf("generate labels error :%s,%v", s.rollingSet.Name, err)
		return err
	}
	tLabels := template.Labels
	newLabels := util.MergeLabels(s.rollingSet.Labels, tLabels)
	s.generalLabels = newLabels
	s.generalAnnotations = map[string]string{}
	copyGeneralAnnotations(s.rollingSet.Annotations, s.generalAnnotations)
	if features.C2FeatureGate.Enabled(features.DisableLabelInheritance) {
		for k := range s.generalLabels {
			if !s.isC2InternalMeta(k) {
				delete(s.generalLabels, k)
			}
		}
		for k := range s.generalAnnotations {
			if !s.isC2InternalMeta(k) {
				delete(s.generalAnnotations, k)
			}
		}
	}
	for key, val := range s.c.writeLabels {
		s.generalLabels[key] = val
	}
	return nil
}

func (s *InPlaceScheduler) isC2InternalMeta(key string) bool {
	for _, prefix := range carbonv1.C2InternalMetaPrefix {
		if strings.HasPrefix(key, prefix) && !strSliceContains(carbonv1.C2InternalMetaBlackList, key) {
			return true
		}
	}
	if key == carbonv1.DefaultRollingsetUniqueLabelKey || key == carbonv1.DefaultShardGroupUniqueLabelKey {
		return true
	}
	return false
}

// check rollingset参数， 并返回rollingsetArgs
func (s *InPlaceScheduler) checkArgs() (rollingsetArgs, error) {
	//参数校验
	rollingsetArgs := rollingsetArgs{}
	if err := checkParam(s.rollingSet); err != nil {
		return rollingsetArgs, err
	}
	return rollingsetArgs, nil
}

// rollingset 目标参数校验
func checkParam(rs *carbonv1.RollingSet) error {
	if rs.GetReplicas() < 0 {
		return errors.New("illegal parameter Replicas")
	}

	if rs.Spec.LatestVersionRatio != nil && *rs.Spec.LatestVersionRatio < 0 {
		return errors.New("illegal parameter LatestVersionRatio")
	}
	return nil
}

// latestVersionPlan
// latestResVersion
// _replicaSet
// _resChangedReplicaSet
// _resChangedReplicaSet
func (s *InPlaceScheduler) initScheduleParams(rollingsetArgs *rollingsetArgs) error {
	//在最前面resetVersion中已经做重置
	if s.rollingSet.IsRollingSetInRestarting() && s.rollingSet.RollingSetNeedGracefullyRestarting() {
		if nil != s.rollingSet.Spec.Strategy.RollingUpdate {
			zeroMaxUnavailable := intstr.FromString(fmt.Sprintf("%d%%", 0))
			glog.Infof("smooth rolling %s", s.rollingSet.Name)
			s.rollingSet.Spec.Strategy.RollingUpdate.MaxSurge = s.rollingSet.Spec.Strategy.RollingUpdate.MaxUnavailable
			s.rollingSet.Spec.Strategy.RollingUpdate.MaxUnavailable = &zeroMaxUnavailable
			if s.rollingSet.Spec.ScaleSchedulePlan.Strategy.RollingUpdate != nil {
				s.rollingSet.Spec.ScaleSchedulePlan.Strategy.RollingUpdate.MaxUnavailable = &zeroMaxUnavailable
			}
		}
	}
	rollingsetArgs.latestVersionPlan = carbonv1.GetVersionPlanWithGangPlanFromRS(s.rollingSet)
	rollingsetArgs.latestResVersion = s.rollingSet.Spec.ResVersion
	replicas, resChangedReplicas := s.collectResourceChangeReplicas(s.replicas)
	for i := range replicas {
		replicas[i].WorkerNode.Status.Score = carbonv1.ScoreOfWorker(&replicas[i].WorkerNode)
	}
	sort.Stable(ByScore(replicas))
	rollingsetArgs._replicaSet = replicas
	rollingsetArgs._resChangedReplicaSet = resChangedReplicas
	rollingsetArgs.standByReplicas = make([]*carbonv1.Replica, 0, len(replicas))
	rollingsetArgs.activeReplicas = make([]*carbonv1.Replica, 0, len(replicas))
	rollingsetArgs.releasingReplicas = make([]*carbonv1.Replica, 0, len(replicas))
	rollingsetArgs.replicaNames = make([]string, 0, len(replicas))

	groupVersionToVersionMap := make(map[string]string)
	if s.rollingSet.Spec.ShardGroupVersion != "" {
		groupVersionToVersionMap[s.rollingSet.Spec.ShardGroupVersion] = s.rollingSet.Spec.Version
	}
	var replicaSetInWorking = make([]*carbonv1.Replica, 0, len(replicas))
	for _, replica := range replicas {
		version := replica.Spec.Version
		groupVersion := replica.Spec.ShardGroupVersion
		if groupVersion != "" && groupVersion != s.rollingSet.Spec.ShardGroupVersion { // 避免重复添加
			groupVersionToVersionMap[groupVersion] = version
		}
		rollingsetArgs.replicaNames = append(rollingsetArgs.replicaNames, replica.Name)
		if carbonv1.IsReplicaReleasing(replica) || carbonv1.IsReplicaReleased(replica) {
			rollingsetArgs.releasingReplicas = append(rollingsetArgs.releasingReplicas, replica)
			continue
		}
		if carbonv1.IsStandbyReplica(replica) {
			rollingsetArgs.standByReplicas = append(rollingsetArgs.standByReplicas, replica)
			continue
		}
		replicaSetInWorking = append(replicaSetInWorking, replica)
	}
	rollingsetArgs.groupVersionToVersionMap = groupVersionToVersionMap

	timeout, vs := carbonv1.GetGroupVersionStatusMap(replicaSetInWorking)
	if glog.V(4) {
		glog.Infof("[%s] getGroupVersionStatusMap:%s,%d", s.rsKey, utils.ObjJSON(vs), timeout)
	}
	if features.C2FeatureGate.Enabled(features.ValidateRsVersion) {
		err := s.validateRsVersion(s.rollingSet, vs, rollingsetArgs._replicaSet)
		if err != nil {
			return err
		}
	}
	if timeout != 0 && s.c != nil {
		glog.Infof("rollingset: %s, enqueue after %d seconds", s.rsKey, timeout)
		s.c.EnqueueAfter(s.rsKey, time.Duration(timeout)*time.Second)
	}
	rvs := getResourceVersionStatusMap(
		s.rollingSet.Status.ResourceVersionStatusMap, replicaSetInWorking, s.rollingSet)
	s.rollingSet.Status.GroupVersionStatusMap = vs
	s.rollingSet.Status.ResourceVersionStatusMap = rvs

	releasing := s.rollingSet.DeletionTimestamp != nil
	rollingsetArgs.ShardScheduleParams = rollalgorithm.NewShardScheduleParams(
		s.rollingSet, s.rollingSet.GetSchedulePlan(), releasing, s.scheduleID, s.rollingSet.Spec.Version,
		s.rollingSet.Spec.RollingStrategy, groupVersionToVersionMap,
		map[string]string{
			"roleName":  carbonv1.GetRoleName(s.rollingSet),
			"groupName": carbonv1.GetGroupName(s.rollingSet),
			"appName":   carbonv1.GetAppName(s.rollingSet),
		})
	rollingsetArgs.inactiveStandbyNodes = make([]*v1.Replica, 0, len(rollingsetArgs.standByReplicas))
	rollingsetArgs.rollingUpdateArgs = s.newRollingUpdateArgs(rollingsetArgs)
	return nil
}

func (s *InPlaceScheduler) initSpotArgs(args *rollingsetArgs, target int) {
	if s.rollingSet == nil || s.rollingSet.Spec.Replicas == nil {
		return
	}
	spotTarget := 0
	if features.C2FeatureGate.Enabled(features.OpenRollingSpotPod) {
		replicas := *s.rollingSet.Spec.Replicas
		scalePlan := s.rollingSet.Spec.ScaleSchedulePlan
		if scalePlan != nil && scalePlan.Replicas != nil && *scalePlan.Replicas > replicas {
			replicas = *scalePlan.Replicas
		}
		if target >= int(replicas) {
			spotTarget = target - int(replicas)
		}
	}
	spotTarget += v1.GetSpotTargetFromAnnotation(s.rollingSet)
	args.spotTarget = int64(spotTarget)
}

// versionReplicaMap  map[string][]*carbonv1.Replica
// versionPlanMap    map[string]*carbonv1.VersionPlan
func (s *InPlaceScheduler) initVersionArgs(rollingsetArgs *rollingsetArgs) {
	versionReplicaMap := make(map[string][]*carbonv1.Replica)
	versionPlanMap := make(map[string]*carbonv1.VersionPlanWithGangPlan)
	versionLabelMap := make(map[string]map[string]string)
	//latest初始值设置
	versionReplicaMap[rollingsetArgs.LatestVersion] = make([]*carbonv1.Replica, 0)
	versionPlanMap[rollingsetArgs.LatestVersion] = carbonv1.GetVersionPlanWithGangPlanFromRS(s.rollingSet)
	totalReleasingCount := 0
	standByReplicaCount := 0
	for _, replica := range rollingsetArgs._replicaSet {
		version := replica.Spec.Version
		if _, ok := versionPlanMap[version]; !ok {
			versionPlanMap[version] = carbonv1.GetVersionPlanWithGangPlanFromReplica(replica)
		}
		if _, ok := versionLabelMap[version]; !ok {
			versionLabelMap[version] = replica.Labels
		}
		//排除已释放的节点， 不能再次用作释放/升级
		if carbonv1.IsReplicaReleasing(replica) || carbonv1.IsReplicaReleased(replica) {
			totalReleasingCount++
		} else if carbonv1.IsStandbyReplica(replica) {
			standByReplicaCount++
		} else {
			versionReplica := versionReplicaMap[version]
			if versionReplica == nil {
				versionReplica := make([]*carbonv1.Replica, 0, 1)
				versionReplicaMap[version] = versionReplica
			}
			versionReplica = append(versionReplica, replica)
			versionReplicaMap[version] = versionReplica
		}
	}

	rollingsetArgs.versionReplicaMap = versionReplicaMap
	rollingsetArgs.versionPlanMap = versionPlanMap
	releasingReplicaCounter.WithLabelValues(
		carbonv1.GetObjectBaseScopes(s.rollingSet)...,
	).Set(float64(totalReleasingCount))
	versionReplicaCounter.WithLabelValues(
		carbonv1.GetObjectBaseScopes(s.rollingSet)...,
	).Set(float64(len(versionReplicaMap)))
}

func (s *InPlaceScheduler) updateBroadcastField(rollingsetArgs *rollingsetArgs) {
	replicas := rollingsetArgs.versionReplicaMap[rollingsetArgs.LatestVersion]
	replicas = append(replicas, rollingsetArgs.standByReplicas...)
	var broadReplicas []*carbonv1.Replica
	carbonv1.InitWorkerScheduleTimeout(&rollingsetArgs.latestVersionPlan.BroadcastPlan.WorkerSchedulePlan)
	for i := range rollingsetArgs.latestVersionPlan.GangVersionPlans {
		carbonv1.InitWorkerScheduleTimeout(&rollingsetArgs.latestVersionPlan.GangVersionPlans[i].BroadcastPlan.WorkerSchedulePlan)
	}
	for i := range replicas {
		if carbonv1.IsRrResourceUpdating(&replicas[i].WorkerNode) {
			glog.Infof("workernode[%v] is in rr resource updating, ignore broadcast", replicas[i].WorkerNode.Name)
			continue
		}
		if s.isBroadcastMatched(replicas[i], rollingsetArgs) {
			continue
		}
		replica := replicas[i].DeepCopy()
		if s.isGangRolling() {
			s.createGangMainPart(replica, rollingsetArgs.latestVersionPlan, rollingsetArgs.LatestVersion)
		}
		carbonv1.SetReplicaVersionPlanWithGangPlan(replica, rollingsetArgs.latestVersionPlan, rollingsetArgs.LatestVersion)
		copyGeneralAnnotations(s.generalAnnotations, replica.Annotations)
		syncNonVersionedMetas(s.rollingSet, replica, rollingsetArgs.nonVersionedKeys)
		broadReplicas = append(broadReplicas, replica)
	}
	rollingsetArgs.broadcastReplicaSet = broadReplicas
}

func (s *InPlaceScheduler) isBroadcastMatched(replica *carbonv1.Replica, rollingsetArgs *rollingsetArgs) bool {
	var replicaPlanStr, latestPlanStr = utils.ObjJSON(replica.Spec.BroadcastPlan), utils.ObjJSON(&rollingsetArgs.latestVersionPlan.BroadcastPlan)
	if replicaPlanStr != latestPlanStr {
		glog.V(5).Infof("updateBroadcastField :%s, %s", replicaPlanStr, latestPlanStr)
		return false
	}
	if s.isGangRolling() && !carbonv1.IsGangMainPart(replica.Spec.Template) {
		return false
	}
	if !s.isGangRolling() && !compareGeneralAnnotations(replica.Annotations, s.generalAnnotations) {
		return false
	}
	if !s.isGangRolling() && !carbonv1.IsNonVersionedKeysMatch(s.rollingSet.Spec.Template, replica.Spec.Template, rollingsetArgs.nonVersionedKeys) {
		return false
	}
	return true
}

func addSupplement(args *rollingsetArgs, replicas ...*v1.Replica) {
	pc, _, _, _ := runtime.Caller(1)
	for i := range replicas {
		replica := replicas[i]
		name := replica.Name
		if name == "" && len(replica.Gang) != 0 {
			name = replica.Gang[0].Name
		}
		glog.Infof("%s added to supplement by %s", name, runtime.FuncForPC(pc).Name())
		if name != "" && !args.supplements.Contains(name) {
			args.supplementReplicaSet = append(args.supplementReplicaSet, replica)
			args.supplements.Add(name)
		}
	}
}

func (s *InPlaceScheduler) syncSpotReplicas(args *rollingsetArgs, redundants []*v1.Replica) {
	allReplicas := v1.MergeReplica(args.activeReplicas, args.standByReplicas)
	curCandidatesMap, creatingReplicas := s.classifySpotReplicas(allReplicas)
	spotReplicas, spotCandidates := s.filterSpotReplicasAndCandidates(args, allReplicas, curCandidatesMap)
	redundantCandidates := getRedundantCandidates(redundants, curCandidatesMap)
	curCandidates := make([]*v1.Replica, 0, len(curCandidatesMap))
	for _, replica := range curCandidatesMap {
		curCandidates = append(curCandidates, replica)
	}
	if lackSize := int(args.spotTarget) - len(spotReplicas); lackSize > 0 {
		supplyCnt := supplySpotFromCandidates(args, lackSize, spotCandidates)
		lackSize -= supplyCnt
		supplyCnt = supplySpotFromCreatingReplicas(args, lackSize, creatingReplicas, curCandidates)
		curCandidates = curCandidates[supplyCnt:]
	} else if lackSize < 0 {
		s.releaseSpotReplicas(args, redundantCandidates, spotReplicas, -lackSize)
	}
}

func supplySpotFromCreatingReplicas(args *rollingsetArgs, lackSize int, creatingReplicas []*v1.Replica, curCandidates []*v1.Replica) int {
	supplyCnt := 0
	for i := 0; i < lackSize && i < len(creatingReplicas) && i < len(curCandidates); i++ {
		replica := curCandidates[i]
		creatingReplicas[i].Spec.IsSpot = true
		creatingReplicas[i].Spec.BackupOfPod.Name = replica.Status.EntityName
		creatingReplicas[i].Spec.BackupOfPod.Uid = replica.Status.EntityUid
		glog.Infof("supply spot replica %s, value:%s", creatingReplicas[i].Name, utils.ObjJSON(creatingReplicas[i].Spec.BackupOfPod))
		addSupplement(args, creatingReplicas[i])
		args.curSpotNames = append(args.curSpotNames, replica.Name)
		supplyCnt++
	}
	return supplyCnt
}

func supplySpotFromCandidates(args *rollingsetArgs, lackSize int, spotCandidates []*v1.Replica) int {
	supplyCnt := 0
	for i := 0; i < lackSize && i < len(spotCandidates); i++ {
		replica := spotCandidates[i]
		replica.Spec.IsSpot = true
		addSupplement(args, replica)
		args.curSpotNames = append(args.curSpotNames, replica.Name)
		supplyCnt++
	}
	return supplyCnt
}

func (s *InPlaceScheduler) releaseSpotReplicas(args *rollingsetArgs, candidateRedundants []*v1.Replica, spotReplicas []*v1.Replica, redundant int) {
	for i := 0; i < len(candidateRedundants) && i < len(spotReplicas) && i < redundant; i++ {
		if uid := v1.GetCurrentPodUid(&spotReplicas[i].WorkerNode); uid == candidateRedundants[i].Status.EntityUid {
			continue
		}
		spot := spotReplicas[i]
		candidate := candidateRedundants[i]
		candidate.Spec.IsSpot = spotReplicas[i].Spec.IsSpot
		candidate.Spec.BackupOfPod.Name = spot.Spec.BackupOfPod.Name
		candidate.Spec.BackupOfPod.Uid = spot.Spec.BackupOfPod.Uid
		spot.Spec.IsSpot = false
		spot.Spec.BackupOfPod.Name = ""
		spot.Spec.BackupOfPod.Uid = ""
		glog.Infof("swap spot instance from %s to %s for release, content: %s", spot.Name, candidate.Name, utils.ObjJSON(candidate.Spec.BackupOfPod))
		addSupplement(args, spotReplicas[i])
		addSupplement(args, candidateRedundants[i])
	}
}

func (s *InPlaceScheduler) filterSpotReplicasAndCandidates(args *rollingsetArgs, allReplicas []*v1.Replica, curCandidateMap map[string]*v1.Replica) ([]*v1.Replica, []*v1.Replica) {
	var spotReplicas []*carbonv1.Replica
	var spotCandidates []*carbonv1.Replica
	for i := range allReplicas {
		replica := allReplicas[i]
		if uid := v1.GetCurrentPodUid(&replica.WorkerNode); uid != "" && curCandidateMap[uid] != nil {
			if replica.Spec.IsSpot {
				spotReplicas = append(spotReplicas, replica)
				args.curSpotNames = append(args.curSpotNames, replica.Name)
			} else {
				spotCandidates = append(spotCandidates, replica)
			}
			delete(curCandidateMap, uid)
		} else if (uid == "" || curCandidateMap[uid] == nil) && replica.Spec.IsSpot {
			replica.Spec.IsSpot = false
			replica.Spec.BackupOfPod.Name = ""
			replica.Spec.BackupOfPod.Uid = ""
			addSupplement(args, replica)
			curCandidateMap[replica.Status.EntityUid] = replica
		}
		if uid := v1.GetCurrentPodUid(&replica.Backup); uid != "" && curCandidateMap[uid] != nil {
			delete(curCandidateMap, uid)
		}
	}
	return spotReplicas, spotCandidates
}

func (s *InPlaceScheduler) classifySpotReplicas(allReplicas []*v1.Replica) (map[string]*v1.Replica, []*v1.Replica) {
	curCandidatesMap := map[string]*carbonv1.Replica{}
	var creatingReplicas []*carbonv1.Replica
	for i := range allReplicas {
		replica := allReplicas[i]
		if !replica.Spec.IsSpot && v1.IsWorkerEntityAlloced(&replica.WorkerNode) {
			curCandidatesMap[replica.Status.EntityUid] = replica
		}
		if !replica.Spec.IsSpot && (replica.ResourceVersion == "" || v1.IsQuotaNotEnough(&replica.WorkerNode)) {
			creatingReplicas = append(creatingReplicas, replica)
		}
	}
	return curCandidatesMap, creatingReplicas
}

func getRedundantCandidates(redundants []*v1.Replica, curCandidateMap map[string]*v1.Replica) []*v1.Replica {
	var candidates []*v1.Replica
	for i := range redundants {
		replica := redundants[i]
		if uid := v1.GetCurrentPodUid(&replica.WorkerNode); v1.IsReplicaAssigned(replica) && (uid == "" || curCandidateMap[uid] == nil) {
			candidates = append(candidates, replica)
		}
	}
	return candidates
}

func activeStandbyStatusChangeLog(replica *carbonv1.Replica, targetMode carbonv1.WorkerModeType, nowHour int64) bool {
	cost := carbonv1.GetDeletionCost(replica)
	if cost == 0 {
		return false
	}
	isStandby := carbonv1.IsStandbyReplica(replica)
	var statusConvertStr = ""
	standby2Active := isStandby && targetMode == carbonv1.WorkerModeTypeActive
	active2Standby := !isStandby && targetMode != carbonv1.WorkerModeTypeActive
	if standby2Active {
		statusConvertStr = "standby2Active"
	} else if active2Standby {
		statusConvertStr = "active2Standby"
	} else {
		return false
	}
	matchCorrect := false
	standby2ActiveWarn := false
	standbyHours := carbonv1.GetReplicaStandbyHours(replica)
	// 状态切换的正确时间点
	if len(standbyHours) > 0 {
		lastHour := standbyHours[len(standbyHours)-1]
		for _, hour := range standbyHours {
			if standby2Active && nowHour == hour {
				standby2ActiveWarn = true
			}
			delta := hour - lastHour
			if delta == 1 || delta == -23 {
				lastHour = hour
				continue
			}
			if (active2Standby && (nowHour == hour-1 || nowHour == hour+23)) ||
				(standby2Active && (nowHour == lastHour+1 || nowHour == lastHour-23)) {
				matchCorrect = true
			}
			lastHour = hour
		}
	}
	if standby2Active {
		year, month, day := time.Now().Date()
		date := strconv.FormatInt(int64(year), 10) + strconv.FormatInt(int64(month), 10) + strconv.FormatInt(int64(day), 10)
		if date != metricDate {
			metricDate = date
		}
	}
	if standby2ActiveWarn {
		glog.Warningf("standby2Active warning, match standby hours replicaName:%s", v1.GetWorkerReplicaName(&replica.WorkerNode))
	}
	glog.Infof("%s, matchCorrect:%t, replicaName:%s, deletionCost:%d, standbyHours:%d", statusConvertStr, matchCorrect, v1.GetWorkerReplicaName(&replica.WorkerNode), cost, standbyHours)
	return matchCorrect
}

func (s *InPlaceScheduler) calStandbyCountsByRatio(standbys int, ratioMap, targetStandbyMap map[carbonv1.WorkerModeType]int) {
	hot := int(math.Ceil(float64(standbys*ratioMap[carbonv1.WorkerModeTypeHot]) / 100.0))
	warm := int(math.Ceil(float64(standbys*ratioMap[carbonv1.WorkerModeTypeWarm]) / 100.0))
	warm = integer.IntMin(standbys-hot, warm)
	cold := standbys - hot - warm
	var inactive int
	if ratioMap[carbonv1.WorkerModeTypeInactive] == 100 {
		inactive = standbys
		cold = 0
	}
	targetStandbyMap[carbonv1.WorkerModeTypeHot] = hot
	targetStandbyMap[carbonv1.WorkerModeTypeWarm] = warm
	targetStandbyMap[carbonv1.WorkerModeTypeCold] = cold
	targetStandbyMap[carbonv1.WorkerModeTypeInactive] = inactive
}

func (s *InPlaceScheduler) reportStandbyMetrics(workerMode carbonv1.WorkerModeType, scalerPool string, standbys []*carbonv1.Replica) {
	var assignedNum int64
	var unassignedNum int64
	for _, replica := range standbys {
		if carbonv1.IsWorkerAssigned(&replica.WorkerNode) {
			assignedNum++
		} else {
			unassignedNum++
		}
	}
	standbyReplica.WithLabelValues(
		append(carbonv1.GetObjectBaseScopes(s.rollingSet), string(workerMode), scalerPool, "true")...,
	).Set(float64(assignedNum))
	standbyReplica.WithLabelValues(
		append(carbonv1.GetObjectBaseScopes(s.rollingSet), string(workerMode), scalerPool, "false")...,
	).Set(float64(unassignedNum))
	var cpuSpec int64
	var gpuSpec int64
	var memSpec int64
	for _, container := range s.rollingSet.Spec.Template.Spec.Containers {
		for resourceName, resource := range container.Resources.Limits {
			if resourceName == carbonv1.ResourceCPU {
				cpuSpec += resource.Value()
			}
			if resourceName == carbonv1.ResourceGPU {
				gpuSpec += resource.Value()
			}
			if resourceName == carbonv1.ResourceMEM {
				memSpec += resource.Value()
			}
		}
	}
	if cpuSpec > 0 {
		standbyCPU.WithLabelValues(
			append(carbonv1.GetObjectBaseScopes(s.rollingSet), string(workerMode), scalerPool, "true")...,
		).Set(float64(assignedNum * cpuSpec * 100))
		standbyCPU.WithLabelValues(
			append(carbonv1.GetObjectBaseScopes(s.rollingSet), string(workerMode), scalerPool, "false")...,
		).Set(float64(unassignedNum * cpuSpec * 100))
	}
	if gpuSpec > 0 {
		standbyGPU.WithLabelValues(
			append(carbonv1.GetObjectBaseScopes(s.rollingSet), string(workerMode), scalerPool, "true")...,
		).Set(float64(assignedNum * gpuSpec))
		standbyGPU.WithLabelValues(
			append(carbonv1.GetObjectBaseScopes(s.rollingSet), string(workerMode), scalerPool, "false")...,
		).Set(float64(unassignedNum * gpuSpec))
	}
	if memSpec > 0 {
		standbyMem.WithLabelValues(
			append(carbonv1.GetObjectBaseScopes(s.rollingSet), string(workerMode), scalerPool, "true")...,
		).Set(float64(assignedNum * memSpec))
		standbyMem.WithLabelValues(
			append(carbonv1.GetObjectBaseScopes(s.rollingSet), string(workerMode), scalerPool, "false")...,
		).Set(float64(unassignedNum * memSpec))
	}
}

func (s *InPlaceScheduler) supplementStandby(rollingsetArgs *rollingsetArgs, redundants *[]*carbonv1.Replica) {
	if carbonv1.IsQuickOnline(s.rollingSet) {
		sort.Sort(InactiveStandbyRedundantSort(*redundants))
	} else {
		sort.Sort(RedundantSort(*redundants))
	}
	// 减去由于可用度多要的节点，避免整体多要
	if nil != rollingsetArgs.scaledParameters && rollingsetArgs.scaledParameters.StandByReplicas != 0 {
		standbys := int(rollingsetArgs.scaledParameters.StandByReplicas)
		if len(rollingsetArgs.activeReplicas) > int(rollingsetArgs.scaledParameters.ActiveReplicas) && !carbonv1.IsQuickOnline(s.rollingSet) {
			standbys = standbys - (len(rollingsetArgs.activeReplicas) - int(rollingsetArgs.scaledParameters.ActiveReplicas))
		}
		standByReplicas := make([]*carbonv1.Replica, 0, standbys)
		redundantNodes := make([]*carbonv1.Replica, 0, len(*redundants))
		targetStandbyMap := make(map[carbonv1.WorkerModeType]int)
		standbyReplicaMap := make(map[carbonv1.WorkerModeType]int)
		s.calStandbyCountsByRatio(standbys, rollingsetArgs.scaledParameters.StandByRatioMap, targetStandbyMap)
		for _, replica := range *redundants {
			if !carbonv1.IsStandbyReplica(replica) || replica.Spec.Version != rollingsetArgs.LatestVersion {
				redundantNodes = append(redundantNodes, replica)
				continue
			}
			workermode := replica.Spec.WorkerMode
			targetcnt := targetStandbyMap[workermode]
			if len(standByReplicas) < standbys && standbyReplicaMap[workermode] < targetcnt {
				standByReplicas = append(standByReplicas, replica)
				standbyReplicaMap[workermode]++
			} else {
				redundantNodes = append(redundantNodes, replica)
			}
		}
		glog.V(4).Infof("%s supplementStandby %d, %d,%d,%d, %d,%d,%d targetStandby:%s, standbyReplicaMap:%s, StandByRatioMap:%s",
			s.rsKey, len(*redundants), len(standByReplicas), len(redundantNodes), rollingsetArgs.scaledParameters.StandByReplicas,
			standbys, len(rollingsetArgs.activeReplicas), rollingsetArgs.scaledParameters.ActiveReplicas,
			utils.ObjJSON(targetStandbyMap), utils.ObjJSON(standbyReplicaMap), utils.ObjJSON(rollingsetArgs.scaledParameters.StandByRatioMap))

		*redundants = redundantNodes
		workModeSort := []carbonv1.WorkerModeType{
			carbonv1.WorkerModeTypeHot,
			carbonv1.WorkerModeTypeWarm,
			carbonv1.WorkerModeTypeCold,
			carbonv1.WorkerModeTypeInactive,
		}
		for _, workermode := range workModeSort {
			targetCnt := targetStandbyMap[workermode]
			lackSize := int32(targetCnt - standbyReplicaMap[workermode])
			if lackSize <= 0 {
				continue
			}
			supplementNodes := s.supplementNodesWithPlan(rollingsetArgs, rollingsetArgs.LatestVersion, rollingsetArgs.latestVersionPlan,
				lackSize, workermode, false, redundants)
			if workermode == carbonv1.WorkerModeTypeCold && rollingsetArgs.scaledParameters.ScalerResourcePoolName != "" {
				for _, coldReplica := range supplementNodes {
					glog.V(4).Infof("add cold standby for worker[%v] to scalerResourcePool[%v]", coldReplica.Name, rollingsetArgs.scaledParameters.ScalerResourcePoolName)
					coldReplica.Annotations[carbonv1.AnnotationScalerResourcePool] = rollingsetArgs.scaledParameters.ScalerResourcePoolName
				}
			}
			s.reportStandbyMetrics(workermode, rollingsetArgs.scaledParameters.ScalerResourcePoolName, supplementNodes)
			standByReplicas = append(standByReplicas, supplementNodes...)
			if lackSize != 0 || len(*redundants) != 0 {
				glog.Infof("[%s] supplement %s standby [%s] Replicas, lackSize:%v, redundantSize:%v, supplementSize:%v",
					s.scheduleID, workermode, rollingsetArgs.LatestVersion, lackSize, len(*redundants), len(supplementNodes))
			}
		}
		rollingsetArgs.standByReplicas = standByReplicas
	}
}

// 1)获取可额外增加节点数maxExtraCount
// 2)遍历idlepool， 如果isReleasing=true， 则放到redundantNodes
// 3)如果supplementNodes.size <= lacksize 则放到supplementNodes， 否则放到则放到redundantNodes (应该用 <= )
// 4)maxExtraCount>0, lackcount = min(maxExtraCount, lacksize - supplementNodes.size());
// 5)lackcount>0 需要额外创建lackcount个replica节点， 并放入replicalist和supplementNodes尾部
// 6)设置supplementNodes中的version， versionedpan， timestamp
// 7)idlepool交换为redundantNodes
func (s *InPlaceScheduler) supplementNodesWithPlan(rollingsetArgs *rollingsetArgs, version string,
	versionPlan *carbonv1.VersionPlanWithGangPlan, count int32, workerMode carbonv1.WorkerModeType, hasDependency bool, redundants *[]*carbonv1.Replica,
) []*carbonv1.Replica {
	redundantsNodes := make([]*carbonv1.Replica, 0)
	supplementNodes := make([]*carbonv1.Replica, 0)
	for _, replica := range *redundants {
		if carbonv1.IsReplicaReleasing(replica) {
			redundantsNodes = append(redundantsNodes, replica)
		} else {
			if int32(len(supplementNodes)) < count && isStandbyCanActive(s.rollingSet, replica, workerMode) {
				supplementNodes = append(supplementNodes, replica)
				activeStandbyStatusChangeLog(replica, workerMode, int64(time.Now().Hour()))
			} else {
				redundantsNodes = append(redundantsNodes, replica)
			}
		}
	}
	lackCount := count - int32(len(supplementNodes))
	if lackCount > 0 {
		//限制一次批量创建的最大上限
		lackCount = integer.Int32Min(lackCount, ReplicaMaxBatchSize)
		//额外创建lackCount个replica
		for i := int32(0); i < lackCount; i++ {
			//创建replica, 放入rollingsetArgs.replicaSet, rollingsetArgs.supplementNodes
			replica, err := s.createReplicaTemplate(s.rollingSet)
			if err != nil {
				continue
			}
			rollingsetArgs._replicaSet = append(rollingsetArgs._replicaSet, replica)
			supplementNodes = append(supplementNodes, replica)
		}
		glog.Infof("[%s] create %v Replica Template, namespace:%s, create count:%v, replicaSet.size:%v",
			s.scheduleID, workerMode, s.rollingSet.Namespace, lackCount, len(rollingsetArgs._replicaSet))
	}

	var needUpdateSupplementNode = make([]*carbonv1.Replica, 0, len(supplementNodes))
	//修改更新节点的version， versionPlan
	for i := range supplementNodes {
		supplementNode := supplementNodes[i]
		inSilence := carbonv1.InSilence(&supplementNode.WorkerNode)
		var tmp *carbonv1.Replica
		if inSilence && carbonv1.IsStandbyWorkerMode(workerMode) {
			tmp = supplementNode.DeepCopy()
		}
		if workerMode == carbonv1.WorkerModeTypeActive && carbonv1.SetStandby2Active(&supplementNode.WorkerNode) {
			glog.V(3).Infof("worker[%v] change standby[%v] to active", supplementNode.Name, supplementNode.Spec.WorkerMode)
		}
		if !s.isGangRolling() {
			copyGeneralMetas(supplementNode, s.generalLabels, s.generalAnnotations)
			if !carbonv1.SyncSilence(&supplementNode.WorkerNode, workerMode) {
				if supplementNode.Spec.WorkerMode != workerMode {
					glog.Infof("set workermode %s, %s", supplementNode.Name, workerMode)
				}
				supplementNode.Spec.WorkerMode = workerMode
			}

			// fed scheduler目前不支持fed rr，所以临时黑一下
			if supplementNode.Spec.WorkerMode == v1.WorkerModeTypeCold && supplementNode.Status.IsFedWorker {
				glog.V(4).Infof("worker[%s] is fed pod, can't change to cold now", supplementNode.Name)
				supplementNode.Spec.WorkerMode = v1.WorkerModeTypeWarm
			}
		}
		if carbonv1.IsStandbyWorkerMode(workerMode) && carbonv1.SyncCold2Warm4Update(&supplementNode.WorkerNode) {
			glog.Infof("set worker[%v] to warm for rr resource update", supplementNode.Name)
		} else {
			carbonv1.SetReplicaVersionPlanWithGangPlan(supplementNode, versionPlan, version)
			supplementNode.Spec.DependencyReady = utils.BoolPtr(!hasDependency)
			if tmp != nil && utils.ObjJSON(tmp.Spec) == utils.ObjJSON(supplementNode.Spec) &&
				utils.ObjJSON(tmp.Labels) == utils.ObjJSON(supplementNode.Labels) &&
				utils.ObjJSON(tmp.Labels) == utils.ObjJSON(supplementNode.Labels) {
				continue
			}
			needUpdateSupplementNode = append(needUpdateSupplementNode, supplementNode)
		}
	}
	if len(supplementNodes) != 0 {
		glog.Infof("[%s] supplement %v Replica set version and versionPlan, version:%s, size:%v, needUpdateSize:%v",
			s.scheduleID, workerMode, version, len(supplementNodes), len(needUpdateSupplementNode))
	}

	//置换
	*redundants = redundantsNodes
	//追加
	addSupplement(rollingsetArgs, needUpdateSupplementNode...)
	return supplementNodes
}

// 追加Replica对象
func appendReplicas(target []*carbonv1.Replica, replicas []*carbonv1.Replica) []*carbonv1.Replica {
	if target == nil {
		return replicas
	}
	target = append(target, replicas...)
	return target
}

func (s *InPlaceScheduler) releaseReplicas(redundants *[]*carbonv1.Replica) error {
	if redundants == nil || len(*redundants) == 0 {
		return nil
	}
	if len(*redundants) > ReplicaMaxBatchSize {
		*redundants = (*redundants)[:ReplicaMaxBatchSize]
	}

	var names = make([]string, len(*redundants))
	for i := range *redundants {
		names[i] = (*redundants)[i].Name
	}
	start := time.Now()
	if cnt, err := (*s.resourceManager).BatchReleaseReplica(s.rollingSet, *redundants); err != nil {
		glog.Warningf("[%s] rollingset sync releaseReplicas failed, rs.name: %v, target:%v, replica names: %s, success:%v, err:%v",
			s.scheduleID, s.rollingSet.Name, len(*redundants), names, cnt, err)
		return err
	}
	elapsed := time.Since(start)
	glog.Infof("[%s] rollingset sync releaseReplicas rs.name: %v, release count: %v, replica names: %s, elapsed: %v",
		s.scheduleID, s.rollingSet.Name, len(*redundants), names, elapsed)
	return nil
}

func (s *InPlaceScheduler) removeReleasedReplicaNodes(rollingsetArgs *rollingsetArgs) error {
	tmp := make([]*carbonv1.Replica, 0)
	removeReplicas := make([]*carbonv1.Replica, 0)
	for _, replica := range rollingsetArgs._replicaSet {
		if !carbonv1.IsReplicaReleased(replica) {
			tmp = append(tmp, replica)
		} else {
			removeReplicas = append(removeReplicas, replica)
		}
	}
	rollingsetArgs._replicaSet = tmp
	rollingsetArgs.supplementReplicaSet = carbonv1.SubReplicas(
		rollingsetArgs.supplementReplicaSet, removeReplicas)
	rollingsetArgs.broadcastReplicaSet = carbonv1.SubReplicas(
		rollingsetArgs.broadcastReplicaSet, removeReplicas)
	if len(removeReplicas) > 0 {
		if len(removeReplicas) > ReplicaMaxBatchSize {
			removeReplicas = removeReplicas[:ReplicaMaxBatchSize]
		}
		start := time.Now()
		if cnt, err := (*s.resourceManager).BatchRemoveReplica(s.rollingSet, removeReplicas); err != nil {
			glog.Warningf("[%s] removeReleasedReplicaNodes remove replica failed, rs.name: %v, target:%d, success:%d, err:%v",
				s.scheduleID, s.rollingSet.Name, len(removeReplicas), cnt, err)
			return err
		}
		elapsed := time.Since(start)
		glog.Infof("[%s] removeReleasedReplicaNodes remove replica rs.name: %v, remove count: %v, elapsed: %v",
			s.scheduleID, s.rollingSet.Name, len(removeReplicas), elapsed)
	}
	return nil
}

func (s *InPlaceScheduler) syncReplicas(rollingsetArgs *rollingsetArgs) error {
	if len(rollingsetArgs.broadcastReplicaSet) > 0 {
		start := time.Now()
		if cnt, err := (*s.resourceManager).BatchUpdateReplica(s.rollingSet, rollingsetArgs.broadcastReplicaSet); err != nil {
			glog.Warningf("[%s] rollingset rs.name:%s batch update replica failed: target:%d, success:%d, err:%v",
				s.scheduleID, s.rollingSet.Name, len(rollingsetArgs.broadcastReplicaSet), cnt, err)
			return err
		}
		elapsed := time.Since(start)
		glog.Infof("[%s] rollingset  syncReplicas rs.name: %v, broadcast update count: %v, elapsed: %v",
			s.scheduleID, s.rollingSet.Name, len(rollingsetArgs.broadcastReplicaSet), elapsed)
	}

	if len(rollingsetArgs.supplementReplicaSet) == 0 {
		return nil
	}
	createReplicas := make([]*carbonv1.Replica, 0)
	updateReplicas := make([]*carbonv1.Replica, 0)

	for _, replica := range rollingsetArgs.supplementReplicaSet {
		// 认为是创建
		if replica.ResourceVersion == "" {
			carbonv1.InitAnnotationsLabels(&replica.ObjectMeta)
			carbonv1.MapCopy(s.c.writeLabels, replica.Labels)
			createReplicas = append(createReplicas, replica)
			glog.V(5).Infof("create replica: %s", utils.ObjJSON(replica))
		} else {
			updateReplicas = append(updateReplicas, replica)
			glog.Infof("%s: sp %s to version %s,%s,%s,%s",
				s.rsKey, replica.GetQualifiedName(), replica.Spec.UserDefVersion,
				replica.Spec.Version, replica.Spec.ShardGroupVersion, replica.Spec.WorkerMode)
		}
		v1.SetReplicaAnnotation(replica, "scheduleID", s.scheduleID)
	}
	if len(createReplicas) > 0 {
		start := time.Now()
		if cnt, err := (*s.resourceManager).BatchCreateReplica(s.rollingSet, createReplicas); err != nil {
			glog.Warningf("[%s] rollingset rs.name:%s batch create replica, target:%d, success:%d, error:%v",
				s.scheduleID, s.rollingSet.Name, len(createReplicas), cnt, err)
			return err
		}
		elapsed := time.Since(start)
		glog.Infof("[%s] rollingset syncReplicas rs.name: %v, create count: %v, elapsed: %v",
			s.scheduleID, s.rollingSet.Name, len(createReplicas), elapsed)
	}
	if len(updateReplicas) > 0 {
		start := time.Now()
		for i := range updateReplicas {
			updateReplicas[i].OwnerReferences = s.getOwner()
		}
		if cnt, err := (*s.resourceManager).BatchUpdateReplica(s.rollingSet, updateReplicas); err != nil {
			glog.Warningf("[%s] rollingset rs.name:%s batch update replica failed: target:%d, success:%d, err:%v",
				s.scheduleID, s.rollingSet.Name, len(updateReplicas), cnt, err)
			return err
		}
		elapsed := time.Since(start)
		glog.Infof("[%s] rollingset  syncReplicas rs.name: %v, update count: %v, elapsed: %v",
			s.scheduleID, s.rollingSet.Name, len(updateReplicas), elapsed)
	}
	return nil
}

func (s *InPlaceScheduler) getOwner() []metav1.OwnerReference {
	var rs = s.rollingSet
	if carbonv1.IsSubRollingSet(rs) {
		rs = s.groupRollingSet
		if rs == nil {
			glog.Infof("%s owner rs is nil", s.rollingSet.Name)
			return nil
		}
	}
	return []metav1.OwnerReference{*metav1.NewControllerRef(rs, controllerKind)}
}

func (s *InPlaceScheduler) getUncompleteReplicaStatus(replicaSet []*carbonv1.Replica, latestVersion string) map[string]carbonv1.ProcessStep {
	uncompleteStatusMap := make(map[string]carbonv1.ProcessStep)
	limit := 10
	for _, replica := range replicaSet {
		if replica.Spec.Version == latestVersion && !replica.Status.Complete {
			uncompleteStatusMap[replica.WorkerNode.Status.IP+"_"+replica.WorkerNode.Name] = replica.Status.ProcessStep
		}
		if limit <= 0 {
			break
		}
		limit--
	}
	return uncompleteStatusMap
}

func (s *InPlaceScheduler) syncRollingsetStatus(replicaSet *[]*carbonv1.Replica, complete, serviceReady bool, spotStatus v1.SpotStatus, uncompleteStatusMap map[string]carbonv1.ProcessStep) error {
	status := s.rollingSetStatus
	newStatus, timeout := CalculateStatus(s.rollingSet, *replicaSet)
	if timeout != 0 && s.c != nil {
		glog.Infof("rollingset: %s, enqueue after %d seconds", s.rsKey, timeout)
		s.c.EnqueueAfter(s.rsKey, time.Duration(timeout)*time.Second)
	}
	newStatus.ReleasingReplicas, newStatus.ReleasingWorkers = GetReleasingReplicaCount(s.replicas)
	newStatus.Complete = complete
	if newStatus.Complete {
		newStatus.InstanceID = s.rollingSet.Spec.InstanceID
	} else {
		newStatus.InstanceID = status.InstanceID
	}
	newStatus.ServiceReady = serviceReady
	newStatus.UncompleteReplicaInfo = uncompleteStatusMap
	newStatus.SpotInstanceStatus = spotStatus
	controller.RecordCompleteLatency(&s.rollingSet.ObjectMeta, s.rollingSet.Kind,
		controllerAgentName, s.rollingSet.Spec.Version, s.rollingSet.Status.Complete)
	s.rollingSet.Status = newStatus
	status.LastUpdateStatusTime = newStatus.LastUpdateStatusTime
	//status.GroupVersionStatusMap = newStatus.GroupVersionStatusMap
	if utils.ObjJSON(status) != utils.ObjJSON(s.rollingSet.Status) {
		glog.Infof("[%s] call update rollingset status. rollingset.name:%s, version:%s, old:%s, new:%s",
			s.scheduleID, s.rollingSet.Name, s.rollingSet.Spec.Version, utils.ObjJSON(status),
			utils.ObjJSON(s.rollingSet.Status))
		err := (*s.resourceManager).UpdateRollingSetStatus(s.rollingSet)
		if err != nil {
			glog.Warningf("[%s] UpdateRollingSetStatus failure, err: %v", s.scheduleID, err)
		}
		return err
	}
	return nil
}

// createReplicaTemplate 只创建replica模版
func (s *InPlaceScheduler) createReplicaTemplate(rs *carbonv1.RollingSet) (*carbonv1.Replica, error) {
	rsName := rs.Name
	if carbonv1.IsSubRollingSet(rs) {
		rsName = carbonv1.GetRsID(rs)
	}
	// Create new Replica
	newReplica := carbonv1.Replica{}
	if !s.isGangRolling() {
		worker := s.createWorker(rsName, "", "", rs)
		newReplica.WorkerNode = *worker
	} else {
		newReplica.Gang = make([]carbonv1.Replica, 0, len(rs.Spec.GangVersionPlan))
		var gangID string
		for k := range rs.Spec.GangVersionPlan {
			worker := s.createWorker(rsName, k, gangID, rs)
			gangID = carbonv1.GetGangID(worker)
			newReplica.Gang = append(newReplica.Gang, carbonv1.Replica{WorkerNode: *worker})
		}
	}
	return &newReplica, nil
}

// getScheduleID 生成一个调度任务的id
func getScheduleID(name string) string {
	id := "s-" + name + "-" + common.GetCommonID()
	return id
}

// resource update by pass rolling
func (s *InPlaceScheduler) collectResourceChangeReplicas(replicas []*carbonv1.Replica) (copy []*carbonv1.Replica, changed []*carbonv1.Replica) {
	copy = make([]*carbonv1.Replica, 0, len(replicas))
	changed = make([]*carbonv1.Replica, 0, len(replicas))
	for _, replica := range replicas {
		if carbonv1.IsReplicaReleasing(replica) || carbonv1.IsReplicaReleased(replica) {
			continue
		}
		//TODO 跳过coldstandby
		if replica.Spec.ResVersion != s.rollingSet.Spec.ResVersion &&
			s.rollingSet.Spec.Version != s.rollingSet.Spec.ResVersion {
			replica = replica.DeepCopy()
			changed = append(changed, replica)
			glog.V(4).Infof("collectResourceChangeReplicas %s,%s,%s", replica.Name, replica.Spec.ResVersion, s.rollingSet.Spec.ResVersion)
		}
		copyReplica := replica.DeepCopy()
		if copyReplica.Spec.Version == "" && s.isGangRolling() {
			copyReplica.Spec.Version = s.rollingSet.Spec.Version
		}
		copy = append(copy, copyReplica)
	}
	return
}

func (s *InPlaceScheduler) initResVersionMap(rollingsetArgs *rollingsetArgs) {
	if s.rollingSet.Spec.Version == s.rollingSet.Spec.ResVersion {
		return
	}
	resVersionReplicaMap := make(map[string][]*carbonv1.Replica)
	resVersionPlanMap := make(map[string]*carbonv1.VersionPlan)
	for _, replica := range rollingsetArgs._resChangedReplicaSet {
		version := replica.Spec.ResVersion
		versionPlan := resVersionPlanMap[version]
		if versionPlan == nil {
			resVersionPlanMap[version] = &replica.Spec.VersionPlan
		}
		if !carbonv1.IsReplicaReleasing(replica) {
			//排除已释放的节点， 不能再次用作释放/升级
			versionReplica := resVersionReplicaMap[version]
			if versionReplica == nil {
				versionReplica := make([]*carbonv1.Replica, 0, 1)
				resVersionReplicaMap[version] = versionReplica
			}
			versionReplica = append(versionReplica, replica)
			resVersionReplicaMap[version] = versionReplica
		}
	}
	rollingsetArgs.resVersionReplicaMap = resVersionReplicaMap
	rollingsetArgs.resVersionPlanMap = resVersionPlanMap
}

func (s *InPlaceScheduler) updateReplicaResource(rollingsetArgs *rollingsetArgs) error {
	if s.rollingSet.Spec.Version == s.rollingSet.Spec.ResVersion {
		return nil
	}

	hippoPodSpec, _, err := carbonv1.GetPodSpecFromTemplate(s.rollingSet.Spec.VersionPlan.Template)
	if err != nil {
		glog.Errorf("Unmarshal error :%s,%v", s.rollingSet.Name, err)
		return err
	}

	var canNotChange = map[string]bool{}
	for k, v := range rollingsetArgs.resVersionPlanMap {
		oldHippoPodSpec, _, err := carbonv1.GetPodSpecFromTemplate(v.Template)
		if err != nil {
			glog.Errorf("Unmarshal error :%s,%v", k, err)
			return err
		}
		if k == s.rollingSet.Spec.ResVersion && 0 != len(oldHippoPodSpec.Containers) {
			continue
		}
		if len(hippoPodSpec.Containers) != len(oldHippoPodSpec.Containers) {
			continue
		}
		if couldRollingByPass(hippoPodSpec, oldHippoPodSpec) {
			replicas := rollingsetArgs.resVersionReplicaMap[k]
			for i := range replicas {
				err = carbonv1.UpdateReplicaResource(
					replicas[i],
					hippoPodSpec,
					s.generalLabels,
					s.rollingSet.Spec.ResVersion,
					s.rollingSet.Spec.WorkerSchedulePlan)
				glog.Infof("update replica resource : %s, %s, %v", replicas[i].Name, k, err)
				if err != nil {
					canNotChange[replicas[i].Name] = true
				}
			}
		}
	}
	var resChangedReplicaSet = make([]*carbonv1.Replica, 0, len(rollingsetArgs._resChangedReplicaSet))
	for i := range rollingsetArgs._resChangedReplicaSet {
		replica := rollingsetArgs._resChangedReplicaSet[i]
		if !canNotChange[replica.Name] {
			resChangedReplicaSet = append(resChangedReplicaSet, replica)
		}
	}
	rollingsetArgs._resChangedReplicaSet = resChangedReplicaSet
	return nil
}

func (s *InPlaceScheduler) syncReplicaResource(rollingsetArgs *rollingsetArgs, redundantReplicas *[]*carbonv1.Replica) error {
	if s.rollingSet.Spec.Version == s.rollingSet.Spec.ResVersion {
		return nil
	}
	rollingsetArgs._resChangedReplicaSet = carbonv1.SubReplicas(
		rollingsetArgs._resChangedReplicaSet, rollingsetArgs.supplementReplicaSet)
	rollingsetArgs._resChangedReplicaSet = carbonv1.SubReplicas(
		rollingsetArgs._resChangedReplicaSet, *redundantReplicas)
	rollingsetArgs._resChangedReplicaSet = carbonv1.SubReplicas(
		rollingsetArgs._resChangedReplicaSet, rollingsetArgs.broadcastReplicaSet)

	var err error
	if nil != rollingsetArgs._resChangedReplicaSet && 0 != len(rollingsetArgs._resChangedReplicaSet) {
		start := time.Now()
		_, err = (*s.resourceManager).BatchUpdateReplica(s.rollingSet, rollingsetArgs._resChangedReplicaSet)
		elapsed := time.Since(start)
		glog.Infof("[%s] rollingset  syncReplicaResource rs.name: %v, update count: %v, elapsed: %v",
			s.scheduleID, s.rollingSet.Name, len(rollingsetArgs._resChangedReplicaSet), elapsed)
	}
	return err
}

func couldRollingByPass(hippoPodSpec *carbonv1.HippoPodSpec, oldHippoPodSpec *carbonv1.HippoPodSpec) bool {
	for i := range hippoPodSpec.Containers {
		var ip resource.Quantity
		for k, v := range hippoPodSpec.Containers[i].Resources.Limits {
			if k == carbonv1.ResourceIP {
				ip = v
			}
		}
		var name = hippoPodSpec.Containers[i].Name
		for j := range oldHippoPodSpec.Containers {
			if oldHippoPodSpec.Containers[j].Name == name {
				var oldIP resource.Quantity
				for ipk, ipv := range oldHippoPodSpec.Containers[j].Resources.Limits {
					if ipk == carbonv1.ResourceIP {
						oldIP = ipv
					}
				}
				// container ip 资源变化
				if !reflect.DeepEqual(ip, oldIP) {
					return false
				}
			}
		}
	}
	return true
}

func (s *InPlaceScheduler) getCompleteStatus(rollingsetArgs *rollingsetArgs) bool {
	return s.comeToSomeState(rollingsetArgs, func(replica *carbonv1.Replica) bool {
		return replica.Status.Complete
	})
}

func (s *InPlaceScheduler) getServiceReadyStatus(rollingsetArgs *rollingsetArgs) bool {
	return s.comeToSomeState(rollingsetArgs, func(replica *carbonv1.Replica) bool {
		return replica.Status.ServiceReady
	})
}

func (s *InPlaceScheduler) comeToSomeState(rollingsetArgs *rollingsetArgs, f func(replica *carbonv1.Replica) bool) bool {
	if len(rollingsetArgs.versionPlanMap) != 1 {
		return false
	}
	var activeReplicas = make([]*carbonv1.Replica, 0, len(rollingsetArgs._replicaSet))
	for i := range rollingsetArgs._replicaSet {
		if !carbonv1.IsStandbyReplica(rollingsetArgs._replicaSet[i]) {
			activeReplicas = append(activeReplicas, rollingsetArgs._replicaSet[i])
		}
	}
	if len(activeReplicas) != int(rollingsetArgs.Desired) {
		return false
	}

	for i := range activeReplicas {
		if !f(activeReplicas[i]) {
			return false
		}
		if activeReplicas[i].Status.Version != rollingsetArgs.LatestVersion {
			return false
		}
	}

	return true
}

func (s *InPlaceScheduler) validateRsVersion(rs *carbonv1.RollingSet, vmaps map[string]*rollalgorithm.VersionStatus, replicas []*carbonv1.Replica) error {
	if rs.DeletionTimestamp != nil {
		return nil
	}
	tversion := rs.Spec.Version
	if rs.Spec.ShardGroupVersion != "" {
		tversion = rs.Spec.ShardGroupVersion
	}
	tgeneration, err := getVersionGeneration(tversion)
	if err != nil {
		return nil
	}
	for cversion := range vmaps {
		cgeneration, err := getVersionGeneration(cversion)
		if err != nil {
			return nil
		}
		if tgeneration < cgeneration {
			return fmt.Errorf("error target version, target version: %s, current version: %s", tversion, cversion)
		}
	}
	for i := range replicas {
		if replicas[i].Spec.Version != rs.Spec.Version && replicas[i].Spec.ShardGroupVersion == rs.Spec.ShardGroupVersion &&
			(replicas[i].Spec.Signature != rs.Spec.Signature || rs.Spec.InstanceID != rs.Status.InstanceID || carbonv1.IsRollingSetGracefully(rs)) &&
			rs.Spec.ShardGroupVersion != "" && rs.Spec.ShardGroupVersion != rs.Spec.Version { // 只校验shardgroup的rollingsets
			return fmt.Errorf("invalid version, has changed while groupVersion has not changed : %s, %s, %s", replicas[i].Spec.Version, rs.Spec.Version, rs.Spec.ShardGroupVersion)
		}
	}
	return nil
}

func (s *InPlaceScheduler) isGangRolling() bool {
	if s.rollingSet == nil || len(s.rollingSet.Spec.GangVersionPlan) == 0 {
		return false
	}
	return true
}

func (s *InPlaceScheduler) createWorker(rsName, gangPartName, gangID string, rs *carbonv1.RollingSet) *carbonv1.WorkerNode {
	return util.NewWorker(rsName, gangPartName, gangID, rs, s.getOwner(), s.generalLabels, s.generalAnnotations)
}

func (s *InPlaceScheduler) isDisableReplaceEmptyReplica() bool {
	if s.isGangRolling() {
		return true
	}
	if carbonv1.IsQuickOnline(s.rollingSet) {
		return true
	}
	if nil != s.rollingSet && s.rollingSet.Spec.IsDaemonSet {
		return true
	}
	return false
}

func (s *InPlaceScheduler) newRollingUpdateArgs(rollingsetArgs *rollingsetArgs) *rollalgorithm.RollingUpdateArgs {
	nodes := make(rollalgorithm.NodeSet, 0)
	extraNodes := make(rollalgorithm.NodeSet, 0)
	for i := range rollingsetArgs._replicaSet {
		replica := rollingsetArgs._replicaSet[i]
		if carbonv1.IsReplicaReleasing(replica) || carbonv1.IsReplicaReleased(replica) {
			continue
		} else if carbonv1.IsStandbyReplica(replica) {
			if isStandbyCanActive(s.rollingSet, replica, carbonv1.WorkerModeTypeActive) {
				extraNodes = append(extraNodes, replica)
			} else {
				rollingsetArgs.inactiveStandbyNodes = append(rollingsetArgs.inactiveStandbyNodes, replica)
			}
		} else {
			nodes = append(nodes, replica)
		}
	}
	createNode := func() rollalgorithm.Node {
		replica, err := s.createReplicaTemplate(s.rollingSet)
		if err != nil {
			return nil
		}
		return replica
	}
	plan := s.rollingSet.GetSchedulePlan()
	return &rollalgorithm.RollingUpdateArgs{
		Nodes:              nodes,
		CreateNode:         createNode,
		ExtraNodes:         extraNodes,
		Replicas:           plan.Replicas,
		LatestVersionRatio: plan.LatestVersionRatio,
		MaxUnavailable:     plan.Strategy.RollingUpdate.MaxUnavailable,
		MaxSurge:           plan.Strategy.RollingUpdate.MaxSurge,
		LatestVersion:      s.rollingSet.Spec.Version,
		LatestVersionPlan:  carbonv1.GetVersionPlanWithGangPlanFromRS(s.rollingSet),
		Release:            s.rollingSet.DeletionTimestamp != nil,
		RollingStrategy:    plan.RollingStrategy,
		GroupNodeSetArgs: rollalgorithm.GroupNodeSetArgs{
			GroupVersionMap:                rollingsetArgs.groupVersionToVersionMap,
			VersionHoldMatrixPercent:       plan.VersionHoldMatrixPercent,
			PaddedLatestVersionRatio:       plan.PaddedLatestVersionRatio,
			VersionDependencyMatrixPercent: plan.VersionDependencyMatrixPercent,
			LatestVersionCarryStrategy:     plan.LatestVersionCarryStrategy,
		},
		DisableReplaceEmpty: s.isDisableReplaceEmptyReplica(),
		NodeSetName:         s.rollingSet.Name,
	}
}

func (s *InPlaceScheduler) addNodeSetMapToSupplement(rollingsetArgs *rollingsetArgs, nodeSetMap map[string]rollalgorithm.NodeSet, isCreate bool) int {
	replicas := make([]*v1.Replica, 0)
	for _, nodes := range nodeSetMap {
		for _, node := range nodes {
			replica, ok := node.(*v1.Replica)
			if ok {
				if !s.isGangRolling() {
					copyGeneralMetas(replica, s.generalLabels, s.generalAnnotations)
				}
				replicas = append(replicas, replica)
			}
		}
	}
	if isCreate {
		rollingsetArgs._replicaSet = append(rollingsetArgs._replicaSet, replicas...)
	}
	addSupplement(rollingsetArgs, replicas...)
	return len(replicas)
}

func nodeSetToReplicas(nodes rollalgorithm.NodeSet) []*v1.Replica {
	replicas := make([]*v1.Replica, 0)
	for _, node := range nodes {
		replicas = append(replicas, node.(*v1.Replica))
	}
	return replicas
}

func (s *InPlaceScheduler) createGangMainPart(replica *carbonv1.Replica, versionPlans *carbonv1.VersionPlanWithGangPlan, version string) {
	gangPlan := utils.ObjJSON(versionPlans.GangVersionPlans)
	partName, plan := s.getMainPart(versionPlans)
	if partName != "" {
		worker := s.createWorker(s.rollingSet.Name, partName, carbonv1.GetGangID(replica), s.rollingSet)
		worker.Spec.VersionPlan = *plan
		worker.Spec.Version = version
		carbonv1.SetGangPlan(worker, gangPlan)
		replica.Gang = append(replica.Gang, carbonv1.Replica{WorkerNode: *worker})
		glog.Infof("patch mainpart %s", worker.Name)
	}
}

func (s *InPlaceScheduler) getMainPart(versionPlans *carbonv1.VersionPlanWithGangPlan) (string, *carbonv1.VersionPlan) {
	for k := range versionPlans.GangVersionPlans {
		if carbonv1.IsGangMainPart(versionPlans.GangVersionPlans[k].Template) {
			return k, versionPlans.GangVersionPlans[k]
		}
	}
	return "", nil
}
