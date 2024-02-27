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
	"strconv"
	"strings"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	app "k8s.io/api/apps/v1"
	apps "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	intstrutil "k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/integer"
)

var (
	errNilParam           = fmt.Errorf("nil params")
	errBrokenMinAvailable = fmt.Errorf("min available broken")
)

var (
	// Default25IntOrString 默认25%
	Default25IntOrString = intstr.FromString("25%")

	// Default10IntOrString 默认0%
	Default10IntOrString = intstr.FromString("10%")

	//DefaultLatestVersionRatio 默认100
	DefaultLatestVersionRatio = int32(100)
)

func floatRound(src float64) float64 {
	return utils.FloatRound(src, 6)
}

func isZero(src float64) bool {
	return src < 1e-6
}

// ResolveFenceposts resolves both maxSurge and maxUnavailable. This needs to happen in one
// step. For example:
//
// 2 desired, max unavailable 1%, surge 0% - should scale old(-1), then new(+1), then old(-1), then new(+1)
// 1 desired, max unavailable 1%, surge 0% - should scale old(-1), then new(+1)
// 2 desired, max unavailable 25%, surge 1% - should scale new(+1), then old(-1), then new(+1), then old(-1)
// 1 desired, max unavailable 25%, surge 1% - should scale new(+1), then old(-1)
// 2 desired, max unavailable 0%, surge 1% - should scale new(+1), then old(-1), then new(+1), then old(-1)
// 1 desired, max unavailable 0%, surge 1% - should scale new(+1), then old(-1)
func ResolveFenceposts(maxSurge, maxUnavailable *intstrutil.IntOrString, desired int32,
) (int32, int32, error) {
	surge, err := intstrutil.GetValueFromIntOrPercent(
		intstrutil.ValueOrDefault(maxSurge, intstrutil.FromInt(0)), int(desired), true)
	if err != nil {
		return 0, 0, err
	}
	unavailable, err := intstrutil.GetValueFromIntOrPercent(
		intstrutil.ValueOrDefault(maxUnavailable, intstrutil.FromInt(0)), int(desired), false)
	if err != nil {
		return 0, 0, err
	}

	if surge == 0 && unavailable == 0 {
		surge = 1
	}

	return int32(surge), int32(unavailable), nil
}

// ParseScheduleStrategy returns the maxSurge, maxUnavailable, and minAvailable
func ParseScheduleStrategy(plan SchedulePlan) (maxSurge int32, maxUnavailable int32, minAvailable int32,
	maxSurgePercent int32, maxUnavailablePercent int32, minAvailablePercent int32) {
	if IsRecreate(plan.Strategy) || nil == plan.Replicas || *(plan.Replicas) == 0 {
		return
	}
	maxUnavailableIntOrString := &Default25IntOrString
	maxSurgeIntOrString := &Default10IntOrString
	rollingUpdate := plan.Strategy.RollingUpdate
	if rollingUpdate != nil {
		if rollingUpdate.MaxUnavailable != nil {
			maxUnavailableIntOrString = rollingUpdate.MaxUnavailable
		}
		if rollingUpdate.MaxSurge != nil {
			maxSurgeIntOrString = rollingUpdate.MaxSurge
		}
	}
	// Error caught by validation
	maxSurge, maxUnavailable, _ = ResolveFenceposts(
		maxSurgeIntOrString, maxUnavailableIntOrString, *(plan.Replicas))
	if maxUnavailable > *plan.Replicas {
		maxUnavailable = *plan.Replicas
	}
	minAvailable = *(plan.Replicas) - maxUnavailable

	maxSurgePercent, maxUnavailablePercent, _ = ResolveFenceposts(
		maxSurgeIntOrString, maxUnavailableIntOrString, 100)
	if maxUnavailablePercent > 100 {
		maxUnavailablePercent = 100
	}
	minAvailablePercent = 100 - maxUnavailablePercent
	return
}

// IsRecreate returns true if the strategy type is a rolling recreate.
func IsRecreate(strategy apps.DeploymentStrategy) bool {
	return strategy.Type == "Recreate"
}

func getTotalCount(matrix map[string]int32) int32 {
	var sum int32
	for _, v := range matrix {
		sum += v
	}
	return sum
}

func FixStrategy(replicas int32, defaultMaxUnavailable *intstr.IntOrString, strategy *app.DeploymentStrategy) bool {
	defaultUnavailable, err := intstrutil.GetValueFromIntOrPercent(
		intstrutil.ValueOrDefault(defaultMaxUnavailable, intstrutil.FromInt(0)), int(replicas), false)
	if err != nil {
		return false
	}
	unavailable, err := intstrutil.GetValueFromIntOrPercent(
		intstrutil.ValueOrDefault(strategy.RollingUpdate.MaxUnavailable, intstrutil.FromInt(0)), int(replicas), false)
	if err != nil {
		return false
	}
	if defaultUnavailable > 0 && defaultUnavailable != unavailable {
		strategy.RollingUpdate.MaxUnavailable = defaultMaxUnavailable
		return true
	}
	return false
}

func GetIntOrPercentValue(intOrStr *intstr.IntOrString) (int, bool, error) {
	switch intOrStr.Type {
	case intstr.Int:
		return intOrStr.IntValue(), false, nil
	case intstr.String:
		s := strings.Replace(intOrStr.StrVal, "%", "", -1)
		v, err := strconv.Atoi(s)
		if err != nil {
			return 0, false, fmt.Errorf("invalid value %q: %v", intOrStr.StrVal, err)
		}
		return int(v), true, nil
	}
	return 0, false, fmt.Errorf("invalid type: neither int nor percentage")
}

func subIntOrStrPercent(a, b *intstr.IntOrString) (gap int32) {
	percentA, isPercent, err := GetIntOrPercentValue(a)
	if !isPercent || err != nil {
		return
	}
	percentB, isPercent, err := GetIntOrPercentValue(b)
	if !isPercent || err != nil {
		return
	}
	return integer.Int32Max(0, int32(percentA-percentB))
}

func ComputeLatestReplicas(ratio int32, replicas int32, strategy CarryStrategyType) (latestReplicas int32) {
	if strategy == FloorCarryStrategyType {
		//向下取整 为0时补齐
		latestReplicas = int32(math.Floor(float64(replicas) * float64(ratio) / 100))
		if latestReplicas == 0 && ratio > 0 {
			latestReplicas = 1
		}
	} else {
		//向上取整
		latestReplicas = int32(math.Ceil(float64(replicas) * float64(ratio) / 100))
	}
	return
}
