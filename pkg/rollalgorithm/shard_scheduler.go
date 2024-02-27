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

package rollalgorithm

import (
	"fmt"
	"math"

	"github.com/alibaba/kube-sharding/pkg/ago-util/logging/zapm"
	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	intstrutil "k8s.io/apimachinery/pkg/util/intstr"
	glog "k8s.io/klog"
	"k8s.io/utils/integer"
)

// Shard express a single shard in a group
type Shard interface {
	GetReplicas() int32
	GetLatestVersionRatio(version string) int32
	GetRawReplicas() int32
	GetDependencyLevel() int32
	GetCPU() int32
	GetName() string
	GetCarbonRoleName() string
	GetVersionStatus() map[string]*VersionStatus
	GetResourceVersionStatus() map[string]*ResourceVersionStatus
}

// ShardScheduleParams ShardScheduleParams
type ShardScheduleParams struct {
	Plan SchedulePlan
	//target算法需要
	ShardInternalArgs
	ShardRuntimeArgs
	Metas             map[string]string
	LatestRowComplete bool
}

// ShardInternalArgs ShardInternalArgs
type ShardInternalArgs struct {
	name                     string
	Desired                  int32 //目标值
	maxSurge                 int32 //归一化后的值。最大额外可以存在的replica数， 如果是百分比， 向上取整
	minAvailable             int32
	maxUnavailable           int32 //归一化后的值。最大不可用replica数， 基于replicas， 如果是百分比， 向下取整
	maxSurgePercent          int32
	minAvailablePercent      int32
	maxUnavailablePercent    int32
	latestVersionRatio       int32 //最新版本占比， 用于灰度，百分比， 统一向上取整
	paddedLatestVersionRatio int32 //最新版本占比， 用于灰度，百分比， 统一向上取整
	maxReplicas              int32 //最大replica数
	targetLatestReplicas     int32
	paddedLatestReplicas     int32
	targetOldReplicas        int32
	versionHoldingMap        map[string]int32
	versionHoldMatrixPercent map[string]float64
	scacleUp                 bool
	scacleDown               bool
	scacleOut                bool
	scacleIn                 bool

	CurLatestReplicas      int32 //当前新版replica数
	CurOldReplicas         int32 //当前老版replica数
	CurTotalReplicas       int32 //当前replica数
	VersionReadyCountMap   map[string]int32
	VersionCountMap        map[string]int32
	VersionReadyPercentMap map[string]float64
	VersionSortKeys        []string //versionRepliaMap的key做排序，保证遍历顺序
	GroupVersionSortKeys   []string //versionRepliaMap的key做排序，保证遍历顺序

	// hold cpu  resource version
	minAvailableCPU            int32 //归一化后的值， 最小健康
	maxUnavailableCPU          int32 //归一化后的值， 最小健康
	curTotalCPUCount           int32
	curTotalAvailableCPUCount  int32
	curLatestCPUCount          int32
	curLatestAvailableCPUCount int32
	curOldCPUCount             int32
	curOldAvailableCPUCount    int32
	replicaCPU                 int32
	desireCPUCount             int32
	desireAvailableCPUCount    int32
	resourceVersionStatus      map[string]*ResourceVersionStatus
}

// ShardRuntimeArgs is rolling args runtime
type ShardRuntimeArgs struct {
	Releasing                  bool
	ScheduleID                 string
	LatestVersion              string
	LatestGroupVersion         string
	LatestResourceVersion      string
	RollingStrategy            string
	RestartAfterResourceChange *bool
	EnsureAvailableCPUCores    bool
	GroupVersionToVersionMap   map[string]string
}

// ShardScheduler control one shard rolling
type ShardScheduler interface {
	Schedule(shard Shard, params *ShardScheduleParams) (map[string]int32, map[string]int32, error)
}

// InPlaceShardScheduler control one shard rolling inplace whithout create new replica
// And it can degenerate into normal scheduler if maxUnavailable == 0 and maxSuger > 0.
type InPlaceShardScheduler struct {
	diffLogger *utils.DiffLogger
	debug      bool
}

// NewShardScheduler NewShardScheduler
func NewShardScheduler(diffLogger *utils.DiffLogger) ShardScheduler {
	return &InPlaceShardScheduler{
		diffLogger: diffLogger,
	}
}

// Schedule Schedule
func (s *InPlaceShardScheduler) Schedule(shard Shard, params *ShardScheduleParams) (map[string]int32, map[string]int32, error) {
	if nil == params {
		return nil, nil, errNilParam
	}
	if params.isOneTargetVersion() {
		versionTargetMatrix := map[string]int32{params.LatestVersion: params.Desired}
		dependencyMatrix := s.calcDependencyMatrix(params, versionTargetMatrix)
		return versionTargetMatrix, dependencyMatrix, nil
	}
	s.initTarget(params)
	s.calcHold(params, shard)
	s.calcTarget(params)
	versionTargetMatrix, err := s.calculateVersionTargetMatrix(params)
	s.paddLatestVersion(params, versionTargetMatrix)
	if nil != zapm.Logger {
		zapm.Logger.Info(fmt.Sprintf("shard Schedule with params: %s, \n current versions: %s, \n targets: %s",
			params, getversionsStatus(shard.GetVersionStatus()), printShardTargetMatrix(versionTargetMatrix,
				params.GroupVersionToVersionMap, params.GroupVersionSortKeys, params.VersionSortKeys)),
			getZapFields("roleSchLog-"+shard.GetName(), params.Metas)...)
	}
	if nil != err {
		return nil, nil, err
	}
	dependencyMatrix := s.calcDependencyMatrix(params, versionTargetMatrix)
	return versionTargetMatrix, dependencyMatrix, nil
}

func (s *InPlaceShardScheduler) calculateVersionTargetMatrix(params *ShardScheduleParams) (map[string]int32, error) {
	var versionTargetMatrix = params.VersionCountMap
	latestVersionOversize, oldVersionOversize := s.calaulateOversize(params)
	// 老版本目标数量
	var redudantCount int32 // 回收数量
	// 老版本缩容
	redudantCount = s.oldVersionScaleIn(params, oldVersionOversize, versionTargetMatrix)
	// 新版本缩容
	redudantCount = s.latestVersionScaleIn(params, latestVersionOversize, redudantCount, versionTargetMatrix)
	// 可新增的节点数量
	extraCount := params.maxReplicas - params.CurTotalReplicas
	supplement := redudantCount + extraCount
	// 新版本扩容
	supplement = s.latestVersionScaleOut(
		params, -latestVersionOversize, supplement, versionTargetMatrix)
	s.oldVersionScaleOut(params, -oldVersionOversize, supplement, versionTargetMatrix)

	return versionTargetMatrix, nil
}

func (s *InPlaceShardScheduler) paddLatestVersion(params *ShardScheduleParams, versionTargetMatrix map[string]int32) {
	if params.paddedLatestReplicas > params.targetLatestReplicas && params.Plan.getPaddLatestVersionRatio() > params.Plan.getLatestVersionRatio() {
		gap := params.paddedLatestReplicas - params.targetLatestReplicas
		for i := len(params.VersionSortKeys) - 1; i >= 0; i-- {
			if i < 0 || gap <= 0 {
				break
			}
			version := params.VersionSortKeys[i]
			if version == params.LatestVersion {
				continue
			}
			groupVersion := versionToGroupVersion(version, params.GroupVersionToVersionMap)
			percent := params.versionHoldMatrixPercent[groupVersion]
			if isZero(percent) && gap > 0 {
				shardVersion := groupVersionToVersion(version, params.GroupVersionToVersionMap)
				versionedCount := versionTargetMatrix[shardVersion]
				if versionedCount > 0 {
					paddCount := integer.Int32Min(versionedCount, gap)
					versionTargetMatrix[shardVersion] = versionedCount - paddCount
					gap -= paddCount
					versionTargetMatrix[params.LatestVersion] = versionTargetMatrix[params.LatestVersion] + paddCount
				}
			}
		}
	}
}

func (s *InPlaceShardScheduler) calcDependencyMatrix(args *ShardScheduleParams, targetMap map[string]int32) map[string]int32 {
	if 0 == len(args.Plan.VersionDependencyMatrixPercent) {
		return nil
	}
	var dependencyMatrix = map[string]int32{}
	for groupVersion, percent := range args.Plan.VersionDependencyMatrixPercent {
		version := groupVersionToVersion(groupVersion, args.GroupVersionToVersionMap)
		count := targetMap[version]
		dependency := int32(math.Ceil(float64(args.Desired) * percent / 100.0))
		if dependency > count {
			dependency = count
		}
		dependencyMatrix[version] = dependency
	}
	return dependencyMatrix
}

func (s *InPlaceShardScheduler) calaulateOversize(params *ShardScheduleParams) (int32, int32) {
	oldVersionOversize := params.CurOldReplicas - params.targetOldReplicas
	latestVersionOversize := params.CurLatestReplicas - params.targetLatestReplicas
	return latestVersionOversize, oldVersionOversize
}

func (s *InPlaceShardScheduler) latestVersionScaleIn(params *ShardScheduleParams,
	oversize int32, redudantCount int32, versionTargetMatrix map[string]int32,
) int32 {
	if oversize <= 0 {
		return redudantCount
	}
	redudant := s.versionScacleIn(params, params.LatestVersion, oversize, versionTargetMatrix)
	redudantCount += redudant
	return redudantCount
}

func (s *InPlaceShardScheduler) oldVersionScaleIn(params *ShardScheduleParams,
	oversize int32, versionTargetMatrix map[string]int32,
) int32 {
	if oversize <= 0 {
		return 0
	}
	var redudantCount int32
	for i := len(params.VersionSortKeys) - 1; oversize >= 0 && i >= 0; i-- {
		version := params.VersionSortKeys[i]
		if version == params.LatestVersion {
			continue
		}
		redudant := s.versionScacleIn(params, version, oversize, versionTargetMatrix)
		oversize -= redudant
		redudantCount += redudant
	}
	return redudantCount
}

func (s *InPlaceShardScheduler) versionScacleIn(params *ShardScheduleParams,
	version string, oversize int32, versionTargetMatrix map[string]int32,
) int32 {
	versionHoldCount := params.getHoldCount(version)
	versionCurCount := params.VersionCountMap[version]
	redudant := integer.Int32Min(oversize, integer.Int32Max(0, versionCurCount-versionHoldCount))
	versionTargetMatrix[version] = versionCurCount - redudant
	return redudant
}

func (s *InPlaceShardScheduler) latestVersionScaleOut(params *ShardScheduleParams,
	lack int32, supplement int32, versionTargetMatrix map[string]int32,
) int32 {
	if lack <= 0 || supplement <= 0 {
		return supplement
	}

	supplement = s.versionScacleOut(params, params.LatestVersion, lack, supplement, versionTargetMatrix)
	return supplement
}

func (s *InPlaceShardScheduler) oldVersionScaleOut(params *ShardScheduleParams,
	lack int32, supplement int32, versionTargetMatrix map[string]int32,
) int32 {
	if lack <= 0 || supplement <= 0 {
		return supplement
	}

	for _, version := range params.VersionSortKeys {
		if version == params.LatestVersion {
			continue
		}
		replica := versionTargetMatrix[version]
		if replica < params.versionHoldingMap[version] {
			versionLack := params.versionHoldingMap[version] - replica
			supplement = s.versionScacleOut(params, version, versionLack, supplement, versionTargetMatrix)
			if supplement <= 0 {
				return supplement
			}
		}
	}

	var version string
	for i := range params.VersionSortKeys {
		if params.VersionSortKeys[i] != params.LatestVersion {
			version = params.VersionSortKeys[i]
			break
		}
	}
	if version == params.LatestVersion {
		return supplement
	}
	supplement = s.versionScacleOut(params, version, lack, supplement, versionTargetMatrix)
	return supplement
}

func (s *InPlaceShardScheduler) versionScacleOut(params *ShardScheduleParams, version string,
	lackCount, totalSupplement int32, versionTargetMatrix map[string]int32,
) int32 {
	supplement := integer.Int32Min(lackCount, totalSupplement)
	totalSupplement = totalSupplement - supplement
	versionCurCount := params.VersionCountMap[version]
	versionTargetMatrix[version] = versionCurCount + supplement
	return totalSupplement
}

//versionRepliaMap  = > versionHoldingMap
//hold节点都是可用状态, rollingSet.Spec.VersionHoldMatrix有值，则直接使用
func (s *InPlaceShardScheduler) calcHold(args *ShardScheduleParams, shard Shard) {
	holdMatrix := args.Plan.VersionHoldMatrix
	args.versionHoldMatrixPercent = args.Plan.VersionHoldMatrixPercent
	if holdMatrix == nil || len(holdMatrix) == 0 {
		holdMatrix = s.getRollingsetHoldingMap(args, shard)
		glog.V(4).Infof("[%s] getRollingsetHoldingMap, versionHoldingMap:%v", args.ScheduleID, holdMatrix)
	}
	// groupversion=>替换成version
	versionHoldingMap := make(map[string]int32)
	totalHoldCount := int32(0)
	for groupVersion, holdCount := range holdMatrix {
		version := groupVersionToVersion(groupVersion, args.GroupVersionToVersionMap)
		versionHoldingMap[version] = holdCount
		totalHoldCount += holdCount
	}

	s.diffLog(fmt.Sprintf("calcHold rollingset versionHoldingMap : %s ", args.getQualifiedName()),
		fmt.Sprintf("%v", versionHoldingMap))
	//过滤无效的hold
	s.filterVersionHoldingMap(args, versionHoldingMap)
	s.diffLog(fmt.Sprintf("calcHold rollingset from filterVersionHoldingMap : %s ", args.getQualifiedName()),
		fmt.Sprintf("%v", versionHoldingMap))

	//根据可用度完善hold
	s.completeAvailableHold(args, versionHoldingMap)
	s.diffLog(fmt.Sprintf("calcHold rollingset from completeAvailableHold : %s ", args.getQualifiedName()),
		fmt.Sprintf("%v", versionHoldingMap))

	args.versionHoldingMap = versionHoldingMap
}

func (s *InPlaceShardScheduler) initTarget(args *ShardScheduleParams) {
	if nil == args {
		return
	}
	// 计算target
	targetLatestReplicas, targetOldReplicas := int32(0), int32(0)
	targetLatestReplicas = ComputeLatestReplicas(args.latestVersionRatio, args.maxReplicas, args.Plan.LatestVersionCarryStrategy)
	targetOldReplicas = args.maxReplicas - targetLatestReplicas
	args.targetLatestReplicas = targetLatestReplicas
	args.targetOldReplicas = targetOldReplicas
	args.paddedLatestReplicas = ComputeLatestReplicas(args.paddedLatestVersionRatio, args.maxReplicas, args.Plan.LatestVersionCarryStrategy)
	s.diffLog(fmt.Sprintf("initTarget rollingset: %s ", args.getQualifiedName()),
		fmt.Sprintf("curLatestReplicas:%v, targetLatestReplicas:%v, curOldReplicas:%v, targetOldReplicas:%v, paddedLatestReplicas:%v",
			args.CurLatestReplicas, args.targetLatestReplicas, args.CurOldReplicas, args.targetOldReplicas, args.paddedLatestReplicas))
}

func (s *InPlaceShardScheduler) calcTarget(args *ShardScheduleParams) {
	if nil == args {
		return
	}
	targetLatestReplicas, targetOldReplicas := int32(0), int32(0)
	if len(args.VersionSortKeys) == 1 {
		//只有一个版本时， latestVersionRatio没有意义
		targetLatestReplicas = args.Desired
		targetOldReplicas = args.Desired - targetLatestReplicas
	} else if args.latestVersionRatio == 0 {
		targetLatestReplicas = int32(0)
		targetOldReplicas = args.Desired - targetLatestReplicas
	} else {
		// 计算target
		s.initTarget(args)
		targetLatestReplicas = args.targetLatestReplicas
		targetOldReplicas = args.targetOldReplicas
		extraCount := integer.Int32Max(args.maxReplicas-args.CurTotalReplicas, args.maxReplicas-args.Desired)
		if extraCount < 0 {
			extraCount = 0
		}
		// adjust with maxUnavailable as slide window
		if args.isForward() {
			threshHold := integer.Int32Max(args.getLatestReadyCount()+args.maxUnavailable+extraCount, args.getLatestCount())
			targetLatestReplicas = integer.Int32Min(targetLatestReplicas, threshHold)
			if args.paddedLatestReplicas > targetLatestReplicas && args.getLatestCount() > targetLatestReplicas && args.paddedLatestReplicas >= args.getLatestCount() {
				targetLatestReplicas = args.getLatestCount()
			}
			targetOldReplicas = args.maxReplicas - targetLatestReplicas
			s.diffLog(fmt.Sprintf("calcTarget rollingset forward: %s ", args.getQualifiedName()),
				fmt.Sprintf("threshHold: %d, targetLatestReplicas: %d, targetOldReplicas: %d",
					threshHold, targetLatestReplicas, targetOldReplicas))
		} else {
			threshHold := integer.Int32Max(args.getOldReadyCount()+args.maxUnavailable, args.getOldCount())
			targetOldReplicas = integer.Int32Min(targetOldReplicas, threshHold)
			targetLatestReplicas = args.maxReplicas - targetOldReplicas
			s.diffLog(fmt.Sprintf("calcTarget rollingset backward: %s ", args.getQualifiedName()),
				fmt.Sprintf("threshHold: %d, targetLatestReplicas: %d, targetOldReplicas: %d",
					threshHold, targetLatestReplicas, targetOldReplicas))
		}
	}
	args.targetLatestReplicas = targetLatestReplicas
	args.targetOldReplicas = targetOldReplicas
	s.diffLog(fmt.Sprintf("calcTarget rollingset: %s ", args.getQualifiedName()),
		fmt.Sprintf("curLatestReplicas:%v, targetLatestReplicas:%v, curOldReplicas:%v, targetOldReplicas:%v",
			args.CurLatestReplicas, args.targetLatestReplicas, args.CurOldReplicas, args.targetOldReplicas))
}

//计算rollingset级别的versionhold
func (s *InPlaceShardScheduler) getRollingsetHoldingMap(args *ShardScheduleParams, shard Shard) map[string]int32 {
	latestGroupVersion := versionToGroupVersion(args.LatestVersion, args.GroupVersionToVersionMap)
	holdMatrixPercent, _, _ := calculateHoldMatrix([]Shard{shard}, args.latestVersionRatio, args.RollingStrategy,
		latestGroupVersion, args.maxUnavailablePercent, args.maxSurgePercent, []string{})
	args.versionHoldMatrixPercent = holdMatrixPercent
	holdMatrix := transferHoldMatrix(holdMatrixPercent, args.Desired)
	return holdMatrix
}

func (s *InPlaceShardScheduler) filterVersionHoldingMap(args *ShardScheduleParams, versionHoldingMap map[string]int32) {
	var latestHold = versionHoldingMap[args.LatestVersion]
	var latestReady = args.VersionReadyCountMap[args.LatestVersion]
	if args.targetLatestReplicas >= args.CurLatestReplicas && latestHold > latestReady {
		versionHoldingMap[args.LatestVersion] = args.VersionReadyCountMap[args.LatestVersion]
		latestHold = versionHoldingMap[args.LatestVersion]
	}
	if latestHold > latestReady && latestHold > args.targetLatestReplicas {
		versionHoldingMap[args.LatestVersion] = integer.Int32Min(latestHold, integer.Int32Max(args.targetLatestReplicas, latestReady))
	}
	needHoldCount := args.minAvailable
	if args.minAvailable == args.Desired {
		needHoldCount++
	} else if args.targetLatestReplicas-versionHoldingMap[args.LatestVersion] < args.maxUnavailable {
		needHoldCount += args.maxUnavailable - (args.targetLatestReplicas - versionHoldingMap[args.LatestVersion])
	}
	totalHoldCount := getTotalCount(versionHoldingMap)
	if totalHoldCount <= needHoldCount {
		return
	}
	latestVersionSortKeys := make([]string, len(args.VersionSortKeys))
	for i := range args.VersionSortKeys {
		latestVersionSortKeys[i] = args.VersionSortKeys[i]
	}
	sortVersions(latestVersionSortKeys, args.LatestVersion, args.VersionReadyPercentMap, float64(args.maxUnavailablePercent), RollingStrategyLatestFirst)
	for _, version := range latestVersionSortKeys {
		groupHold, ok := versionHoldingMap[version]
		versionCount := args.VersionCountMap[version]
		//hold值以rollingset实际情况为准， 不超过实际的值
		if ok && groupHold > versionCount {
			versionHoldingMap[version] = versionCount
		}
	}
	totalHoldCount = getTotalCount(versionHoldingMap)
	if totalHoldCount > needHoldCount {
		//处理old version
		//优先去除不ready的
		for i := len(latestVersionSortKeys) - 1; totalHoldCount > needHoldCount && i >= 0; i-- {
			version := latestVersionSortKeys[i]
			if version == args.LatestVersion {
				continue
			}
			count := args.VersionCountMap[version]
			readyCount := args.VersionReadyCountMap[version]
			notReadyCount := integer.Int32Max(0, count-readyCount)
			groupHold := versionHoldingMap[version]
			overSize := totalHoldCount - needHoldCount
			overSize = integer.Int32Min(notReadyCount, overSize)
			overSize = integer.Int32Min(groupHold, overSize)
			s.diffLog(fmt.Sprintf("%s,%s,overSize: ", args.ShardInternalArgs.name, version), fmt.Sprintf("%d,%d,%d", notReadyCount, totalHoldCount, needHoldCount))
			versionHoldingMap[version] = groupHold - overSize
			totalHoldCount -= overSize
		}
		for i := len(latestVersionSortKeys) - 1; totalHoldCount > needHoldCount && i >= 0; i-- {
			version := latestVersionSortKeys[i]
			if version == args.LatestVersion {
				continue
			}
			groupHold := versionHoldingMap[version]
			overSize := totalHoldCount - needHoldCount
			overSize = integer.Int32Min(groupHold, overSize)
			s.diffLog(fmt.Sprintf("%s,%s,overSize: ", args.ShardInternalArgs.name, version), fmt.Sprintf("%d,%d,%d", groupHold, totalHoldCount, needHoldCount))
			versionHoldingMap[version] = groupHold - overSize
			totalHoldCount -= overSize
		}
		//处理latest version
		if totalHoldCount > needHoldCount {
			groupHold := versionHoldingMap[args.LatestVersion]
			overSize := totalHoldCount - needHoldCount
			overSize = integer.Int32Min(groupHold, overSize)
			versionHoldingMap[args.LatestVersion] = groupHold - overSize
			totalHoldCount -= overSize
		}
	}
}

func (s *InPlaceShardScheduler) completeAvailableHold(args *ShardScheduleParams, versionHoldingMap map[string]int32,
) {
	needHoldCount := args.minAvailable
	totalHoldCount := getTotalCount(versionHoldingMap)
	if needHoldCount > totalHoldCount {
		//保证可用度， 需要额外hold
		extraHoldCount := needHoldCount - totalHoldCount
		//处理latestVersion
		latestHold := versionHoldingMap[args.LatestVersion]
		latestCompleteCount := args.VersionReadyCountMap[args.LatestVersion]
		extraCount := latestCompleteCount - latestHold
		targetExtra := args.targetLatestReplicas - latestHold
		if targetExtra >= 0 && extraCount > targetExtra {
			extraCount = targetExtra
		}
		if extraCount > 0 {
			versionHoldingMap[args.LatestVersion] = latestHold + extraCount
			extraHoldCount -= extraCount
			if extraHoldCount <= 0 {
				return
			}
		}

		//处理其他version
		for _, version := range args.VersionSortKeys {
			if version == args.LatestVersion {
				continue
			}
			groupHold := versionHoldingMap[version]
			completeVersionCount := args.VersionReadyCountMap[version]
			if groupHold >= completeVersionCount {
				continue
			}
			versionExtraCount := completeVersionCount - groupHold
			extraCount := integer.Int32Min(extraHoldCount, versionExtraCount)
			if extraCount <= 0 {
				continue
			}
			versionHoldingMap[version] = groupHold + extraCount
			extraHoldCount -= extraCount
			if extraHoldCount <= 0 {
				break
			}
		}
	}
}

func getLatestVersionRatio(ratio *int32) int32 {
	var result int32
	if ratio == nil {
		result = DefaultLatestVersionRatio
	} else {
		if *ratio > 100 {
			result = 100
		} else {
			result = *ratio
		}
	}
	return result
}

func getPaddedLatestVersionRatio(ratio *int32) int32 {
	var result int32
	if ratio == nil {
		result = 0
	} else {
		if *ratio > 100 {
			result = 100
		} else {
			result = *ratio
		}
	}
	return result
}

func (s *InPlaceShardScheduler) diffLog(key, msg string) {
	if nil != s.diffLogger {
		s.diffLogger.Log(key, msg)
	}
	if s.debug {
		glog.Infof("%s: %s", key, msg)
	}
}

func (args *ShardScheduleParams) initScheduleArgs(shard Shard) {
	if nil == shard {
		return
	}
	args.name = shard.GetName()
	args.VersionSortKeys = []string{}
	args.GroupVersionSortKeys = []string{}
	args.VersionReadyCountMap = map[string]int32{}
	args.VersionReadyPercentMap = map[string]float64{}
	args.VersionCountMap = map[string]int32{}
	//rollingset被标记为删除，直接缩容到0个replica
	if args.Releasing {
		args.Desired = 0
		args.maxSurge = 0
		args.minAvailable = 0
		args.latestVersionRatio = DefaultLatestVersionRatio
		args.maxReplicas = 0
		args.targetLatestReplicas = 0
		args.targetOldReplicas = 0
		glog.Infof("[%s] initScheduleArgs rollingset[Deletion]", shard.GetName())
		return
	}
	args.Desired = shard.GetReplicas()
	args.maxSurge, args.maxUnavailable, args.minAvailable, args.maxSurgePercent,
		args.maxUnavailablePercent, args.minAvailablePercent = ParseScheduleStrategy(args.Plan)
	//args.completeCPUCount(shard)
	args.latestVersionRatio = getLatestVersionRatio(args.Plan.LatestVersionRatio)
	args.paddedLatestVersionRatio = getPaddedLatestVersionRatio(args.Plan.PaddedLatestVersionRatio)
	//计算最大replica数量
	if args.minAvailable < args.Desired {
		args.maxReplicas = args.Desired
	} else {
		args.maxReplicas = args.Desired + args.maxSurge
	}

	for groupVersion := range args.GroupVersionToVersionMap {
		args.GroupVersionSortKeys = append(args.GroupVersionSortKeys, groupVersion)
	}
	latestGroupVersion := versionToGroupVersion(args.LatestVersion, args.GroupVersionToVersionMap)
	sortVersions(args.GroupVersionSortKeys, latestGroupVersion, args.VersionReadyPercentMap, float64(args.maxUnavailablePercent), RollingStrategyLatestFirst)
	for _, groupVersion := range args.GroupVersionSortKeys {
		args.VersionSortKeys = append(args.VersionSortKeys, groupVersionToVersion(groupVersion, args.GroupVersionToVersionMap))
	}
	if len(args.VersionSortKeys) == 0 {
		args.VersionSortKeys = append(args.VersionSortKeys, args.LatestVersion)
		vs := shard.GetVersionStatus()
		for groupVersion := range vs {
			version := groupVersionToVersion(groupVersion, args.GroupVersionToVersionMap)
			if version != args.LatestVersion {
				args.VersionSortKeys = append(args.VersionSortKeys, version)
			}
		}
		sortVersions(args.VersionSortKeys, args.LatestVersion, args.VersionReadyPercentMap, float64(args.maxUnavailablePercent), RollingStrategyLatestFirst)
	}
	// 计算当前status
	vs := shard.GetVersionStatus()
	for groupVersion, versionStatus := range vs {
		version := groupVersionToVersion(groupVersion, args.GroupVersionToVersionMap)
		if version == args.LatestVersion {
			args.CurLatestReplicas = getVersionAllReplicas(versionStatus)
		} else {
			args.CurOldReplicas += getVersionAllReplicas(versionStatus)
		}
		args.CurTotalReplicas += getVersionAllReplicas(versionStatus)
		args.VersionReadyCountMap[version] = getVersionReadyReplicas(versionStatus)
		args.VersionCountMap[version] = getVersionAllReplicas(versionStatus)
	}
	for groupVersion, versionStatus := range vs {
		if args.CurTotalReplicas != 0 {
			version := groupVersionToVersion(groupVersion, args.GroupVersionToVersionMap)
			args.VersionReadyPercentMap[version] = float64(getVersionReadyReplicas(versionStatus)) * 100.0 / float64(args.CurTotalReplicas)
		}
	}

	if glog.V(4) {
		glog.Infof("[%s] initScheduleArgs rollingset: %v, latestVersion: %s, curReplicas: %v, curLatestReplicas: %v, curOldReplicas: %v,"+
			"Desired: %v, maxSurge: %v,  minAvailable: %v, maxUnavailable: %v, latestVersionRatio: %v, maxReplicas: %v , groupVersionToVersionMap: %s, versionStatus: %s",
			args.ScheduleID, shard.GetName(), args.LatestVersion, args.CurTotalReplicas, args.CurLatestReplicas, args.CurOldReplicas, args.Desired, args.maxSurge,
			args.minAvailable, args.maxUnavailable, args.latestVersionRatio, args.maxReplicas, args.GroupVersionToVersionMap, getversionsStatus(vs))
	}
}

func (args *ShardScheduleParams) isOneTargetVersion() bool {
	_, latestVersionExist := args.VersionCountMap[args.LatestVersion]
	if 1 == len(args.VersionCountMap) && latestVersionExist {
		return true
	}
	return false
}

func (args *ShardScheduleParams) getHoldCount(version string) int32 {
	return args.versionHoldingMap[version]
}

func (args *ShardScheduleParams) getLatestVersionHoldCount() int32 {
	return args.getHoldCount(args.LatestVersion)
}

func (args *ShardScheduleParams) getOldHoldCount() int32 {
	var holdCount int32
	for version, count := range args.versionHoldingMap {
		if version == args.LatestVersion {
			continue
		}
		holdCount += count
	}
	return holdCount
}

func (args *ShardScheduleParams) getLatestReadyCount() int32 {
	return args.VersionReadyCountMap[args.LatestVersion]
}

func (args *ShardScheduleParams) getOldReadyCount() int32 {
	var holdCount int32
	for version, count := range args.VersionReadyCountMap {
		if version == args.LatestVersion {
			continue
		}
		holdCount += count
	}
	return holdCount
}

func (args *ShardScheduleParams) getLatestCount() int32 {
	return args.VersionCountMap[args.LatestVersion]
}

func (args *ShardScheduleParams) getOldCount() int32 {
	var holdCount int32
	for version, count := range args.VersionCountMap {
		if version == args.LatestVersion {
			continue
		}
		holdCount += count
	}
	return holdCount
}

func (args *ShardScheduleParams) isForward() bool {
	latestCount := args.VersionCountMap[args.LatestVersion]
	currentRatio := int32(floatRound(100.0 * float64(latestCount) / float64(args.CurTotalReplicas)))
	if args.Desired == args.CurTotalReplicas {

	}
	return args.latestVersionRatio >= currentRatio || args.paddedLatestVersionRatio >= currentRatio
}

// NewShardScheduleParams create new ShardScheduleParams
func NewShardScheduleParams(shard Shard, plan SchedulePlan, releasing bool, scheduleID string, latestVersion string,
	rollingStrategy string, groupVersionToVersionMap map[string]string, metas map[string]string) *ShardScheduleParams {
	latestGroupVersion := versionToGroupVersion(latestVersion, groupVersionToVersionMap)
	params := &ShardScheduleParams{
		ShardRuntimeArgs: ShardRuntimeArgs{
			Releasing:                releasing,
			ScheduleID:               scheduleID,
			LatestVersion:            latestVersion,
			LatestGroupVersion:       latestGroupVersion,
			GroupVersionToVersionMap: groupVersionToVersionMap,
			RollingStrategy:          rollingStrategy,
		},
		Plan:  plan,
		Metas: metas,
	}
	params.initScheduleArgs(shard)
	return params
}

// 计算最小cpu, 控制扩核缩容中缩容的进度
func (args *ShardScheduleParams) completeCPUCount(shard Shard) {
	if !args.EnsureAvailableCPUCores {
		return
	}
	args.resourceVersionStatus = shard.GetResourceVersionStatus()
	if nil == args.resourceVersionStatus {
		return
	}
	args.replicaCPU = shard.GetCPU()
	var curTotalReplicas int32
	var curTotalAvailableReplicas int32
	for resourceVersion, status := range args.resourceVersionStatus {
		if resourceVersion == args.LatestResourceVersion {
			args.curLatestCPUCount += status.Replicas * status.CPU
			args.curLatestAvailableCPUCount += status.AvailableReplicas * status.CPU
		}
		args.curTotalCPUCount += status.Replicas * status.CPU
		args.curTotalAvailableCPUCount += status.AvailableReplicas * status.CPU
		curTotalReplicas += status.Replicas
		curTotalAvailableReplicas += status.AvailableReplicas
		if args.replicaCPU > status.CPU {
			args.scacleUp = true
		}
		if args.replicaCPU < status.CPU {
			args.scacleDown = true
		}
	}
	if shard.GetReplicas() > curTotalReplicas {
		args.scacleOut = true
	} else if shard.GetReplicas() < curTotalReplicas {
		args.scacleIn = true
	}
	args.desireCPUCount = shard.GetReplicas() * shard.GetCPU()
	unavailable, _ := intstrutil.GetValueFromIntOrPercent(
		intstrutil.ValueOrDefault(args.Plan.Strategy.RollingUpdate.MaxUnavailable,
			intstrutil.FromInt(0)), int(args.desireCPUCount), false)
	args.desireAvailableCPUCount = args.desireCPUCount - int32(unavailable)
}

// 计算最小hold cpu, 控制缩核扩容中缩容的进度
func (args *ShardScheduleParams) calcResourceVersionHold(shard Shard) map[string]int32 {
	// 不保障cpu可用度，不需要hold
	if !args.EnsureAvailableCPUCores || nil == args.resourceVersionStatus {
		return nil
	}
	// 只有同时变更core数和count 才需要hold
	if !((args.scacleUp && args.scacleIn) || (args.scacleDown && args.scacleOut)) {
		return nil
	}
	needAvailableOldVersionCPU := args.desireAvailableCPUCount - args.curLatestAvailableCPUCount
	if needAvailableOldVersionCPU <= 0 {
		return nil
	}
	overCPUCount := args.curTotalAvailableCPUCount - args.desireAvailableCPUCount
	var holdResourceVersionMap = map[string]int32{}
	for resourceVersion, status := range args.resourceVersionStatus {
		if resourceVersion == args.LatestResourceVersion {
			continue
		}
		scacleCPU := status.CPU - args.replicaCPU
		var scacleReplica int32
		if scacleCPU <= 0 {
			scacleCPU = -scacleCPU
		}
		if scacleCPU > 0 && overCPUCount > 0 {
			scacleReplica := overCPUCount / scacleCPU
			scacleReplica = integer.Int32Max(0, scacleReplica)
			scacleReplica = integer.Int32Min(status.AvailableReplicas, scacleReplica)
		}
		needAvailableOldVersionCPU -= (status.AvailableReplicas - scacleReplica) * args.replicaCPU
		needAvailableOldVersionCPU -= -scacleReplica * status.CPU
		holdResourceVersionMap[resourceVersion] = status.AvailableReplicas - scacleReplica
		if needAvailableOldVersionCPU <= 0 {
			break
		}
		if overCPUCount <= 0 {
			break
		}
	}
	return holdResourceVersionMap
}

func (args *ShardScheduleParams) getQualifiedName() string {
	return fmt.Sprintf("%s::%s", args.name, args.Metas)
}

func groupVersionToVersion(groupVersion string, groupVersionToVersion map[string]string) string {
	if version, ok := groupVersionToVersion[groupVersion]; ok {
		return version
	}
	return groupVersion
}

func versionToGroupVersion(version string, groupVersionToVersion map[string]string) string {
	for groupVersion, v := range groupVersionToVersion {
		if version == v {
			if groupVersion == "" {
				return version
			}
			return groupVersion
		}
	}
	return version
}
