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
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/alibaba/kube-sharding/common/features"
	glog "k8s.io/klog"
	"k8s.io/utils/integer"
)

func getRollingStrategy(strategy string, available map[string]float64, maxUnavailable float64) string {
	var totalAvailable = 0.0
	for i := range available {
		totalAvailable = totalAvailable + available[i]
	}
	if ((totalAvailable < (100.0 - maxUnavailable - 1.0)) || maxUnavailable > float64(maxUnavailableOldFirstStrategy)) && // 可用度足够，且批次较多时才能使用RollingStrategyOldFirst策略
		(strategy == RollingStrategyOldFirst || (defaultRollingStrategy == RollingStrategyOldFirst && strategy == "")) {
		return RollingStrategyLatestFirst
	}
	if strategy == "" {
		return defaultRollingStrategy
	}
	return strategy
}

func calculateHoldMatrix(shards []Shard, latestPercent int32, strategy string,
	targetShardGroupVersion string, maxUnavailable int32, maxSurge int32, newShards []string,
) (map[string]float64, int32, int32) {
	stepRatio := float64(maxUnavailable)
	maxRatio := 100.0
	minReplicas := getMinTargetReplicas(shards)
	if maxUnavailable == 0 && maxSurge != 0 {
		stepRatio = float64(maxSurge)
		maxRatio += stepRatio
	}

	// 使用 availableMatrix 初始化 holdMatrix
	holdMatrix := initShardMatrix(shards, getVersionReadyReplicas, getCompactReplicas, newShards)
	grayHoldMatrix := initShardMatrix(shards, getVersionAllReplicas, getCompactReplicas, newShards)
	curMatrix := initShardMatrix(shards, getVersionAllReplicas, getCurReplicas, newShards)
	latestAvailable := holdMatrix[targetShardGroupVersion]

	// 只有一个版本直接返回
	if onlyOneVersion(curMatrix, targetShardGroupVersion) {
		return holdMatrix, latestPercent, latestPercent
	}
	// 计算targetPercent
	extraRatio := getExtraRatio(shards)
	targetRatio, forward, latestRatio, curLatestRatio := calculateTargetRatio(latestPercent, curMatrix[targetShardGroupVersion],
		holdMatrix, targetShardGroupVersion, stepRatio, extraRatio)
	targetPercent := int32(math.Ceil(targetRatio))
	// 调整holdMatrix 使整体hold比例低于100
	adjustHoldMatrix(holdMatrix, grayHoldMatrix, targetShardGroupVersion, targetRatio, maxRatio, stepRatio, forward, strategy, minReplicas)
	// 可用老版本加上新版本比例不足100时，补齐targetRatio
	var paddedTargetPercent int32
	if len(shards) > 1 && features.C2FeatureGate.Enabled(features.EnableMissingColumn) {
		paddedTargetRatio := paddTargetRatio(targetRatio, maxRatio, getMaxStepRatio(shards), latestAvailable, forward, curMatrix, newShards, targetShardGroupVersion)
		paddedTargetRatio = borderTargetRatio(paddedTargetRatio, latestRatio, curLatestRatio, forward)
		paddedTargetPercent = int32(math.Ceil(paddedTargetRatio))
	} else {
		paddedTargetPercent = 0
	}
	return holdMatrix, targetPercent, paddedTargetPercent
}

// 调整holdMatrix 使整体hold比例低于100, 不足可用度要补齐
func adjustHoldMatrix(holdMatrix map[string]float64, curMatrix map[string]float64, targetShardGroupVersion string,
	targetRatio float64, maxRatio, stepRatio float64, forward bool, strategy string, minReplicas int32,
) {
	var availMatrix = map[string]float64{}
	for k := range holdMatrix {
		v := holdMatrix[k]
		availMatrix[k] = v
	}
	holdMatrix[targetShardGroupVersion] = targetRatio
	if targetRatio == 100.0 {
		maxRatio = 100.0
	}
	keys := []string{}
	var allRatio float64
	for key, val := range holdMatrix {
		keys = append(keys, key)
		allRatio += val
	}
	sortVersions(keys, targetShardGroupVersion, availMatrix, stepRatio, strategy)
	//hold值+目标值超过100的时候， 优先从次新的version减值
	for i := len(keys) - 1; i >= 0; i-- {
		// 如果向后rolling，则从最新版开始减值
		if keys[i] == targetShardGroupVersion && forward {
			continue
		}
		if allRatio-maxRatio < 0 {
			break
		}
		if allRatio-holdMatrix[keys[i]]-maxRatio < 0.0 {
			holdMatrix[keys[i]] = floatRound(holdMatrix[keys[i]] - (allRatio - maxRatio))
			break
		} else {
			allRatio -= holdMatrix[keys[i]]
			holdMatrix[keys[i]] = 0
		}
	}

	// 如果hold不足可用度，找从最老版本开始找完整行补齐
	var minTotalHoldReplicas = getTotalHoldReplicas(holdMatrix, minReplicas)
	if allRatio < maxRatio && minTotalHoldReplicas < minReplicas {
		needExtraHold := maxRatio - allRatio
		for i := 0; i < len(keys); i++ {
			if needExtraHold <= 0 {
				break
			}
			minTotalHoldReplicas = getTotalHoldReplicas(holdMatrix, minReplicas)
			if minTotalHoldReplicas >= minReplicas {
				break
			}
			maxStep := (100.0 / float64(minReplicas))
			maxExtraHold := (maxStep * float64(minReplicas-minTotalHoldReplicas))
			if needExtraHold > maxExtraHold {
				needExtraHold = maxExtraHold
			}
			if keys[i] == targetShardGroupVersion {
				continue
			}
			needExtraHold = supplementHoldPercent(needExtraHold, keys[i], holdMatrix, curMatrix)
		}
	}
}

func supplementHoldPercent(needExtraHold float64, key string, holdMatrix map[string]float64, curMatrix map[string]float64) float64 {
	if holdMatrix[key] < curMatrix[key] {
		extraHold := curMatrix[key] - holdMatrix[key]
		if needExtraHold < extraHold {
			extraHold = needExtraHold
			needExtraHold = 0
		} else {
			needExtraHold -= extraHold
		}
		holdMatrix[key] = holdMatrix[key] + extraHold
	}
	return needExtraHold
}

// 可用老版本加上新版本比例不足100时，补齐targetRatio
func paddTargetRatio(targetRatio, maxRatio, stepRatio, latestAvailable float64, forward bool, holdMatrix map[string]float64, newShards []string, targetShardGroupVersion string) float64 {
	var tunedTargetRatio = targetRatio
	var oldVersionRatio float64
	for k, v := range holdMatrix {
		if k != targetShardGroupVersion {
			oldVersionRatio += v
		}
	}
	// 可用老版本加上新版本比例不足100
	if oldVersionRatio+targetRatio < maxRatio && forward {
		// 存在新建列的情况下不做调整，因为新建列老版本为0，可用度必然不足
		if len(newShards) != 0 {
			glog.Infof("exist newShards and don't padd latest version : %s", newShards)
			return tunedTargetRatio
		}
		maxGap := 25.0
		if maxGap > stepRatio {
			maxGap = stepRatio
		}
		gap := maxRatio - oldVersionRatio - targetRatio // 缺了多少
		if gap > maxGap {
			gap = maxGap // gap 不能大于stepRatio/2
		}
		tunedTargetRatio = targetRatio + gap
		glog.Infof("fineTuneTargetRatio, tunedTargetRatio: %v, targetRatio: %v, oldVersionRatio: %v, maxRatio: %v, gap: %v, holdMatrix: %v",
			tunedTargetRatio, targetRatio, oldVersionRatio, maxRatio, gap, holdMatrix)
	}
	return tunedTargetRatio
}

func calculateTargetRatio(latestPercent int32, curLatestRatio float64,
	holdMatrix map[string]float64, targetShardGroupVersion string, stepRatio, extraRatio float64,
) (float64, bool, float64, float64) {
	latestRatio := floatRound(float64(latestPercent))
	var targetRatio float64
	var availableLatestRatio = holdMatrix[targetShardGroupVersion]
	var availableOldRatio = getTotalRatio(holdMatrix) - availableLatestRatio
	targetRatio = availableLatestRatio + stepRatio + extraRatio
	var forward = true
	if curLatestRatio > latestRatio {
		targetRatio = 100.0 - (availableOldRatio + stepRatio)
		forward = false
	}
	targetRatio = borderTargetRatio(targetRatio, latestRatio, curLatestRatio, forward)
	return targetRatio, forward, latestRatio, curLatestRatio
}

func borderTargetRatio(targetRatio, latestRatio, curLatestRatio float64, forward bool) float64 {
	// 正向以及反向的step保护，不能超过latestRatio限制
	if (forward && targetRatio > latestRatio) || (!forward && targetRatio < latestRatio) {
		targetRatio = latestRatio
	}
	// 避免错误回退
	if (forward && targetRatio < curLatestRatio) || (!forward && targetRatio > curLatestRatio) {
		targetRatio = curLatestRatio
	}
	targetRatio = floatRound(targetRatio)
	return targetRatio
}

func getTotalRatio(matrix map[string]float64) float64 {
	var totalRatio float64
	for _, ratio := range matrix {
		totalRatio += ratio
	}
	return totalRatio
}

type getVersionStatusReplicas func(vs *VersionStatus) int32

func getVersionReadyReplicas(vs *VersionStatus) int32 {
	if nil == vs {
		return 0
	}
	return vs.ReadyReplicas
}

func getVersionAllReplicas(vs *VersionStatus) int32 {
	if nil == vs {
		return 0
	}
	return vs.Replicas
}

type getReplicas func(shard Shard) int32

func getCurReplicas(shard Shard) int32 {
	var currentReplicas int32
	for _, svDetails := range shard.GetVersionStatus() {
		currentReplicas = currentReplicas + getVersionAllReplicas(svDetails)
	}
	return currentReplicas
}

func getTargetReplicas(shard Shard) int32 {
	return shard.GetReplicas()
}

func getMinTargetReplicas(shards []Shard) int32 {
	var minReplicas int32
	for i := range shards {
		replias := getTargetReplicas(shards[i])
		if replias < minReplicas || minReplicas == 0 {
			minReplicas = replias
		}
	}
	return minReplicas
}

func getCompactReplicas(shard Shard) int32 {
	return integer.Int32Min(getCurReplicas(shard), getTargetReplicas(shard))
}

func initShardMatrix(shards []Shard, fGetVersionReplicas getVersionStatusReplicas,
	fGetShardReplicas getReplicas, newShards []string) map[string]float64 {
	//key： groupVersion
	availableMatrix := map[string]float64{}
	//计算不同列各个version的最小占比
	for _, shard := range shards {
		replicas := fGetShardReplicas(shard)
		if replicas == 0 {
			continue
		}
		for shardGroupVersion, svDetails := range shard.GetVersionStatus() {
			versionReplicas := fGetVersionReplicas(svDetails)
			if svDetails.Replicas == 0 && inNewShard(shard, newShards) {
				continue
			}
			// 向下取整
			if val, ok := availableMatrix[shardGroupVersion]; !ok ||
				val >= float64(versionReplicas)*100/float64(replicas) {
				availableMatrix[shardGroupVersion] = floatRound(float64(versionReplicas) * 100 / float64(replicas))
			}
		}
	}

	// if shardGroupVerison not exist in this rollingSet
	// 有列不存在的情况
	for key := range availableMatrix {
		for _, shard := range shards {
			if _, ok := shard.GetVersionStatus()[key]; !ok && 0 != fGetShardReplicas(shard) {
				if inNewShard(shard, newShards) {
					continue
				}
				availableMatrix[key] = 0.0
			}
		}
	}
	// 缩容时可能超过100
	for key := range availableMatrix {
		if availableMatrix[key] > 100.0 {
			availableMatrix[key] = 100.0
		}
	}
	return availableMatrix
}

func inNewShard(shard Shard, newShards []string) bool {
	name := shard.GetName()
	for i := range newShards {
		if name == newShards[i] {
			// 新增列，并且所有版本不可用，忽略此列
			return true
		}
	}
	// 不是新增列，不需要忽略
	return false
}

func shouldIgnoreNewShard(shard Shard, newShards []string) bool {
	name := shard.GetName()
	for i := range newShards {
		if name == newShards[i] {
			for _, svDetails := range shard.GetVersionStatus() {
				if svDetails.ReadyReplicas != 0 {
					// 新增列，有版本不可用，不忽略此列
					return false
				}
			}
			// 新增列，并且所有版本不可用，忽略此列
			return true
		}
	}
	// 不是新增列，不需要忽略
	return false
}

func getExtraRatio(shards []Shard) float64 {
	if !scaleOutWithLatestVersionCount {
		return 0.0
	}
	var minExtraRatio float64
	for _, shard := range shards {
		replicas := getTargetReplicas(shard)
		cur := getCurReplicas(shard)
		if replicas <= cur {
			return 0.0
		}
		extraRatio := 100.0 * float64(replicas-cur) / float64(replicas)
		if extraRatio <= 0 {
			return 0.0
		}
		if extraRatio > 0 && (extraRatio < minExtraRatio || minExtraRatio <= 0) {
			minExtraRatio = extraRatio
		}
	}
	return minExtraRatio
}

func getMaxStepRatio(shards []Shard) float64 {
	var minReplicas int32
	for _, shard := range shards {
		replicas := getTargetReplicas(shard)
		if 0 == minReplicas || (0 != replicas && replicas < minReplicas) {
			minReplicas = replicas
		}
	}
	if 0 == minReplicas {
		return 0.0
	}
	return 100.0 / float64(minReplicas)
}

func getTotalHoldReplicas(holdMatrix map[string]float64, replicas int32) int32 {
	holdReplicasMatrix := transferHoldMatrix(holdMatrix, replicas)
	var totalHoldReplicas int32
	for _, count := range holdReplicasMatrix {
		totalHoldReplicas += count
	}
	return totalHoldReplicas
}

func calculateHoldCount(replicas int32, percent float64) int32 {
	return int32(math.Floor(float64(percent)*float64(replicas)-0.5)/100 + 0.7)
}

func transferHoldMatrix(holdMatrixPercent map[string]float64, replicas int32) map[string]int32 {
	holdMatrix := map[string]int32{}
	for key, val := range holdMatrixPercent {
		holdMatrix[key] = calculateHoldCount(replicas, val)
	}
	return holdMatrix
}

func onlyOneVersion(matrix map[string]float64, targetShardGroupVersion string) bool {
	return matrix[targetShardGroupVersion] == 100 && len(matrix) == 1
}

func newByAvailable(versions []string, latest string, available map[string]float64) ByAvailable {
	return ByAvailable{
		versions:  versions,
		latest:    latest,
		available: available,
	}
}

//ByAvailable is used to sort  sharGroupVersion by generation
type ByAvailable struct {
	versions  []string
	latest    string
	available map[string]float64
}

func (a ByAvailable) Len() int { return len(a.versions) }
func (a ByAvailable) Less(i, j int) bool {
	if a.versions[i] == a.latest {
		return false
	}
	if a.versions[j] == a.latest {
		return true
	}
	if nil != a.available {
		availableI := a.available[a.versions[i]]
		availableJ := a.available[a.versions[j]]
		if availableJ != availableI {
			return availableJ < availableI
		}
	}
	index1 := strings.Index(a.versions[i], "-")
	index2 := strings.Index(a.versions[j], "-")
	if index1 == -1 || index2 == -1 {
		return a.versions[i] < a.versions[j]
	}
	iGenerationChars := a.versions[i][:index1]
	jGenerationChars := a.versions[j][:index2]
	iGeneration, err1 := strconv.Atoi(iGenerationChars)
	jGeneration, err2 := strconv.Atoi(jGenerationChars)
	if err1 != nil || err2 != nil {
		return a.versions[i] < a.versions[j]
	}
	return iGeneration < jGeneration
}
func (a ByAvailable) Swap(i, j int) { a.versions[i], a.versions[j] = a.versions[j], a.versions[i] }

func newByGeneration(versions []string, latest string) ByGeneration {
	return ByGeneration{
		versions: versions,
		latest:   latest,
	}
}

//ByGeneration is used to sort  sharGroupVersion by generation
type ByGeneration struct {
	versions []string
	latest   string
}

func (a ByGeneration) Len() int { return len(a.versions) }
func (a ByGeneration) Less(i, j int) bool {
	if a.versions[i] == a.latest {
		return false
	}
	if a.versions[j] == a.latest {
		return true
	}
	index1 := strings.Index(a.versions[i], "-")
	index2 := strings.Index(a.versions[j], "-")
	if index1 == -1 || index2 == -1 {
		return a.versions[i] < a.versions[j]
	}
	iGenerationChars := a.versions[i][:index1]
	jGenerationChars := a.versions[j][:index2]
	iGeneration, err1 := strconv.Atoi(iGenerationChars)
	jGeneration, err2 := strconv.Atoi(jGenerationChars)
	if err1 != nil || err2 != nil {
		return a.versions[i] < a.versions[j]
	}
	return iGeneration < jGeneration
}
func (a ByGeneration) Swap(i, j int) { a.versions[i], a.versions[j] = a.versions[j], a.versions[i] }

func sortVersions(versions []string, latest string, available map[string]float64, maxUnavailable float64, strategy string) {
	strategy = getRollingStrategy(strategy, available, maxUnavailable)
	switch strategy {
	case RollingStrategyOldFirst:
		sort.Sort(sort.Reverse(newByGeneration(versions, latest)))
	case RollingStrategyLatestFirst:
		sort.Sort(newByGeneration(versions, latest))
	case RollingStrategyLessFirst:
		fallthrough
	default:
		sort.Sort(newByAvailable(versions, latest, available))
	}
}

func isLatestRowComplete(shards []Shard, targetShardGroupVersion string) bool {
	// 使用 availableMatrix 初始化 holdMatrix
	for i := range shards {
		versionStatus := shards[i].GetVersionStatus()
		if versionStatus != nil {
			latestVersonStatus := versionStatus[targetShardGroupVersion]
			if latestVersonStatus != nil {
				if latestVersonStatus.DataReadyReplicas == 0 {
					return false
				}
			} else {
				return false
			}
		} else {
			return false
		}
	}
	return true
}
