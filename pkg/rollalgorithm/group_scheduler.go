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

	"github.com/alibaba/kube-sharding/pkg/ago-util/logging/zapm"
	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	app "k8s.io/api/apps/v1"
	glog "k8s.io/klog"
)

// GroupScheduleParams GroupScheduleParams
type GroupScheduleParams struct {
	Name                 string
	NewShards            []string
	Metas                map[string]string
	TargetShardStrategys map[string]*app.DeploymentStrategy
	DefaultStrategy      app.DeploymentStrategy
	LatestPercent        int32
	MaxSurge             int32
	MaxUnavailable       int32
	ShardGroupVersion    string
	RollingStrategy      string
	Paused               bool
	ShardStatus          []Shard
}

// GroupScheduler coordinate rolling of all columns
type GroupScheduler interface {
	Schedule(params *GroupScheduleParams) (map[string]ShardScheduleParams, int32, error)
}

// NewSyncSlidingScheduler create GroupSyncSlidingScheduler
func NewSyncSlidingScheduler() GroupScheduler {
	return &GroupSyncSlidingScheduler{}
}

// GroupSyncSlidingScheduler use target available ratio add sliding window to rolling
type GroupSyncSlidingScheduler struct{}

// Schedule Schedule
func (gss *GroupSyncSlidingScheduler) Schedule(params *GroupScheduleParams) (map[string]ShardScheduleParams, int32, error) {
	if nil == params {
		return nil, 0, errNilParam
	}
	shardParams := map[string]ShardScheduleParams{}
	holdMatrixPercent, targetPercent, paddedTargetPercent := calculateHoldMatrix(
		params.ShardStatus, params.LatestPercent, params.RollingStrategy, params.ShardGroupVersion,
		params.MaxUnavailable, params.MaxSurge, params.NewShards)
	var needDependency = needDependency(params.ShardStatus)
	latestRowComplete := isLatestRowComplete(params.ShardStatus, params.ShardGroupVersion)
	for _, shard := range params.ShardStatus {
		holdMatrix := transferHoldMatrix(holdMatrixPercent, shard.GetReplicas())
		strategy := params.DefaultStrategy
		var carryStrategyType CarryStrategyType
		if len(params.TargetShardStrategys) != 0 {
			carryStrategyType = FloorCarryStrategyType
		}
		var maxUnavailableGap int32
		if nil != params.TargetShardStrategys[shard.GetName()] {
			strategy = *params.TargetShardStrategys[shard.GetName()]
			maxUnavailable, _ := strategy.RollingUpdate.MaxUnavailable.MarshalJSON()
			defaultMaxUnavailable, _ := params.DefaultStrategy.RollingUpdate.MaxUnavailable.MarshalJSON()
			if string(maxUnavailable) > string(defaultMaxUnavailable) {
				FixStrategy(shard.GetReplicas(), params.DefaultStrategy.RollingUpdate.MaxUnavailable, &strategy)
				maxUnavailableGap = subIntOrStrPercent(strategy.RollingUpdate.MaxUnavailable, params.DefaultStrategy.RollingUpdate.MaxUnavailable)
			}
		}
		roleTargetPercent := targetPercent + maxUnavailableGap
		if roleTargetPercent > 100 {
			roleTargetPercent = 100
		}
		roleTargetPercent = fixLatestVersionRatio(roleTargetPercent, params.LatestPercent, params.ShardGroupVersion, shard)
		schedulePlan := SchedulePlan{
			Replicas:                 utils.Int32Ptr(shard.GetReplicas()),
			LatestVersionRatio:       &roleTargetPercent,
			PaddedLatestVersionRatio: &paddedTargetPercent,
			Strategy:                 strategy,
			Paused:                   params.Paused,
			VersionHoldMatrix:        holdMatrix,
			VersionHoldMatrixPercent: holdMatrixPercent,
			RollingStrategy:          params.RollingStrategy,
		}
		// 需要额外节点来rolling时向上取整, 避免卡住rolling
		_, maxUnavailbaleInt, _, _, _, _ := ParseScheduleStrategy(schedulePlan)
		if maxUnavailbaleInt == 0 {
			carryStrategyType = CeilCarryStrategyType
		}
		schedulePlan.Replicas = utils.Int32Ptr(shard.GetRawReplicas())
		schedulePlan.LatestVersionCarryStrategy = carryStrategyType
		if needDependency {
			schedulePlan.VersionDependencyMatrixPercent = calculateDependencyMatrix(shard, params.ShardStatus)
		}
		var shardParam ShardScheduleParams
		shardParam.Plan = schedulePlan
		shardParam.LatestRowComplete = latestRowComplete
		shardParams[shard.GetName()] = shardParam
	}
	if nil != zapm.Logger {
		zapm.Logger.Info(fmt.Sprintf("shardgroup Schudule wiht current : %s \n, target : %s",
			params, getGroupScheduleTarget(shardParams)), getZapFields("groupSchLog-"+params.Name, params.Metas)...)
	}
	return shardParams, targetPercent, nil
}

func fixLatestVersionRatio(roleTargetPercent int32, targetPercent int32, latestVersion string, shard Shard) int32 {
	specLatestVersionRatio := shard.GetLatestVersionRatio(latestVersion)
	if roleTargetPercent < specLatestVersionRatio && targetPercent >= specLatestVersionRatio {
		glog.Infof("%s fix latestVersionRatio: %d, %d", shard.GetName(), roleTargetPercent, specLatestVersionRatio)
		roleTargetPercent = specLatestVersionRatio
	}
	return roleTargetPercent
}
