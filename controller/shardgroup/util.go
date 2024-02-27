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
	"fmt"
	"math"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	"github.com/alibaba/kube-sharding/pkg/rollalgorithm"
	apps "k8s.io/api/apps/v1"
	glog "k8s.io/klog"
)

func calculateHoldCount(replicas int32, percent float64) int32 {
	return int32(math.Floor(float64(percent)*float64(replicas)+0.5) / 100)
}

// initGroupScheduleParams initGroupScheduleParams
func initGroupScheduleParams(shardGroup *carbonv1.ShardGroup, roles []*carbonv1.RollingSet, shardGroupVersion string,
) (*rollalgorithm.GroupScheduleParams, error) {
	var latestPercent int32
	if nil != shardGroup.Spec.LatestPercent {
		latestPercent = *shardGroup.Spec.LatestPercent
	}

	defaultStrategy := shardGroup.Spec.GetStrategy()
	// Error caught by validation
	maxSurge, maxUnavailable, err := rollalgorithm.ResolveFenceposts(shardGroup.Spec.MaxSurge, defaultStrategy.RollingUpdate.MaxUnavailable, 100)
	if err != nil {
		err := fmt.Errorf("shardgroup %v GetMaxUnavailable failed casue %v", shardGroup.Namespace+"-"+shardGroup.Name, err)
		return nil, err
	}

	var newShards = []string{}
	var targetShardReplicas = map[string]int32{}
	var targetDependencyLevel = map[string]*int32{}
	var targetShardStrategys = map[string]*apps.DeploymentStrategy{}
	for shardKey, shardTemplate := range shardGroup.Spec.ShardTemplates {
		name := carbonv1.GenerateShardName(shardGroup.Name, shardKey)
		targetShardReplicas[name] = shardTemplate.Spec.GetReplicas()
		targetDependencyLevel[name] = shardTemplate.Spec.DependencyLevel
		if nil != shardTemplate.Spec.Strategy {
			targetShardStrategys[name] = shardTemplate.Spec.Strategy
		}
		var newCreate = true
		for i := range shardGroup.Status.OnceCompletedShardNames {
			if name == shardGroup.Status.OnceCompletedShardNames[i] {
				newCreate = false
				break
			}
		}
		if newCreate {
			newShards = append(newShards, name)
		}
	}

	var shards = make([]rollalgorithm.Shard, 0, len(roles))
	for i := range roles {
		role := roles[i].DeepCopy()
		replicas, ok := targetShardReplicas[role.Name]
		role.Spec.Replicas = utils.Int32Ptr(replicas)
		role.Spec.DependencyLevel = targetDependencyLevel[role.Name]
		scaleStrategy := roles[i].GetScaleStrategy()
		if scaleStrategy != nil && ok {
			targetShardStrategys[role.Name] = scaleStrategy.DeepCopy()
		}
		if ok {
			shards = append(shards, role)
		}
	}

	var params = &rollalgorithm.GroupScheduleParams{
		Name: shardGroup.Name,
		Metas: map[string]string{
			"groupName": carbonv1.GetGroupName(shardGroup),
			"appName":   carbonv1.GetAppName(shardGroup),
		},
		TargetShardStrategys: targetShardStrategys,
		DefaultStrategy:      shardGroup.Spec.GetStrategy(),
		LatestPercent:        latestPercent,
		MaxSurge:             maxSurge,
		MaxUnavailable:       maxUnavailable,
		ShardGroupVersion:    shardGroupVersion,
		ShardStatus:          shards,
		Paused:               shardGroup.Spec.Paused,
		NewShards:            newShards,
		RollingStrategy:      shardGroup.Spec.RollingStrategy,
	}
	return params, nil
}

func deepEqual(src, dst interface{}) bool {
	equal := utils.ObjJSON(src) == utils.ObjJSON(dst)
	if !equal {
		if glog.V(5) {
			glog.Infof("not equal src: %s dst: %s", utils.ObjJSON(src), utils.ObjJSON(dst))
		}
	}
	return equal
}
