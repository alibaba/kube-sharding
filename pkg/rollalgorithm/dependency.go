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

func calculateDependencyMatrix(shard Shard, group []Shard) map[string]float64 {
	dependencyShards := getDependencyShards(shard, group)
	replicas := shard.GetReplicas()
	if replicas == 0 || len(dependencyShards) == 0 {
		return nil
	}
	dependencyMatrix := initShardMatrix(dependencyShards, getVersionReadyReplicas, getCompactReplicas, nil)
	return dependencyMatrix
}

func getDependencyShards(shard Shard, group []Shard) []Shard {
	var dependencyShards = make([]Shard, 0, len(group))
	dependencyLevel := shard.GetDependencyLevel()
	for i := range group {
		if dependencyLevel > group[i].GetDependencyLevel() &&
			shard.GetName() != group[i].GetName() &&
			group[i].GetReplicas() > 0 {
			dependencyShards = append(dependencyShards, group[i])
		}
	}
	return dependencyShards
}

func needDependency(group []Shard) bool {
	for i := range group {
		if group[i].GetDependencyLevel() != 0 {
			return true
		}
	}
	return false
}
