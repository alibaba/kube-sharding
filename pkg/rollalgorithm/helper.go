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
	"sort"
	"strconv"
	"strings"

	"github.com/alibaba/kube-sharding/pkg/ago-util/logging/zapm"
	"go.uber.org/zap"
)

var (
	divider = "---"
)

// GetReplicas GetReplicas
func (p *SchedulePlan) GetReplicas() int32 {
	if nil == p.Replicas {
		return 0
	}
	return *p.Replicas
}

// GetLatestVersionRatio GetLatestVersionRatio
func (p *SchedulePlan) GetLatestVersionRatio() int32 {
	if nil == p.LatestVersionRatio {
		return 0
	}
	return *p.LatestVersionRatio
}

func (p *SchedulePlan) String() string {
	var versions = make([]string, len(p.VersionHoldMatrix))
	var holdCounts = make([]string, len(p.VersionHoldMatrix))
	var holdPercents = make([]string, len(p.VersionHoldMatrix))
	i := 0
	for k, v := range p.VersionHoldMatrix {
		versions[i] = k
		holdCounts[i] = strconv.Itoa(int(v))
		holdPercents[i] = strconv.FormatFloat(float64(p.VersionHoldMatrixPercent[k]), 'f', 10, 2)
		i++
	}
	return fmt.Sprintf("replicas:%d, latestVersionRatio:%d, versions: %s, holdCounts: %s, holdPercents: %s",
		p.GetReplicas(), p.GetLatestVersionRatio(), strings.Join(versions, "/"), strings.Join(holdCounts, "/"), strings.Join(holdPercents, "/"))
}

func (gsp *GroupScheduleParams) String() string {
	if nil == gsp {
		return ""
	}
	var versionsSet = map[string]string{}
	var shardStatus = map[string]map[string]*VersionStatus{}
	var shardNames = make([]string, len(gsp.ShardStatus))

	for i := range gsp.ShardStatus {
		shard := gsp.ShardStatus[i]
		shardStatus[shard.GetName()] = shard.GetVersionStatus()
		shardNames[i] = shard.GetName()
		for version := range shard.GetVersionStatus() {
			versionsSet[version] = ""
		}
	}

	var versions = make([]string, len(versionsSet))
	var i = 0
	for version := range versionsSet {
		versions[i] = version
		i++
	}

	sort.Sort(newByAvailable(versions, gsp.ShardGroupVersion, nil))
	var versionsStringBuilder strings.Builder
	versionsStringBuilder.WriteString("versions: ")
	for i := range versions {
		versionsStringBuilder.WriteString(versions[i])
		versionsStringBuilder.WriteString(" || ")
	}
	versionsStringBuilder.WriteString(divider)

	sort.Strings(shardNames)
	for i := range shardNames {
		shardStatu := shardStatus[shardNames[i]]
		versionsStringBuilder.WriteString(fmt.Sprintf(" role %s : ", shardNames[i]))
		for j := range versions {
			versionStatus := shardStatu[versions[j]]
			if nil == versionStatus {
				versionsStringBuilder.WriteString("0/0||")
			} else {
				versionsStringBuilder.WriteString(fmt.Sprintf("%d/%d || ", versionStatus.Replicas, versionStatus.ReadyReplicas))
			}
		}
		versionsStringBuilder.WriteString(divider)
	}
	versionInfos := versionsStringBuilder.String()

	output := fmt.Sprintf("ShardGroupVersion: %s, LatestPercent: %d, MaxUnavailable: %d, %s",
		gsp.ShardGroupVersion, gsp.LatestPercent, gsp.MaxUnavailable, versionInfos)
	return output
}

// GroupScheduleTarget is the target of group
type GroupScheduleTarget map[string]ShardScheduleParams

func getGroupScheduleTarget(targets map[string]ShardScheduleParams) *GroupScheduleTarget {
	groupTarget := GroupScheduleTarget(targets)
	return &groupTarget
}

func printGroupScheduleTarget(targets map[string]ShardScheduleParams) string {
	groupTarget := getGroupScheduleTarget(targets)
	return groupTarget.String()
}

func (gst *GroupScheduleTarget) String() string {
	if nil == gst {
		return ""
	}
	var versionsSet = map[string]string{}
	var shardNames = make([]string, len(*gst))
	var i = 0
	for shardName, shardParams := range *gst {
		shardNames[i] = shardName
		i++
		for version := range shardParams.Plan.VersionHoldMatrix {
			versionsSet[version] = ""
		}
	}

	var versions = make([]string, len(versionsSet))
	i = 0
	for version := range versionsSet {
		versions[i] = version
		i++
	}

	sort.Sort(newByAvailable(versions, "", nil))
	var versionsStringBuilder strings.Builder
	versionsStringBuilder.WriteString("versions: ")
	for i := range versions {
		versionsStringBuilder.WriteString(versions[i])
		versionsStringBuilder.WriteString(" || ")
	}
	versionsStringBuilder.WriteString(divider)
	sort.Strings(shardNames)

	for i := range shardNames {
		shardParam := (*gst)[shardNames[i]]
		versionsStringBuilder.WriteString(fmt.Sprintf("role %s:, replicas: %d, versions: ", shardNames[i], shardParam.Plan.GetReplicas()))
		for j := range versions {
			versionHold := shardParam.Plan.VersionHoldMatrix[versions[j]]
			versionHoldPercent := shardParam.Plan.VersionHoldMatrixPercent[versions[j]]
			versionsStringBuilder.WriteString(fmt.Sprintf("%d/%f || ", versionHold, versionHoldPercent))
		}
		versionsStringBuilder.WriteString(divider)
	}
	versionHoldInfos := versionsStringBuilder.String()
	return versionHoldInfos
}

type versionsStatus map[string]*VersionStatus

func getversionsStatus(status map[string]*VersionStatus) *versionsStatus {
	versionsStatus := versionsStatus(status)
	return &versionsStatus
}

func (s *versionsStatus) String() string {
	var versions = make([]string, len(*s))
	var i = 0
	for version := range *s {
		versions[i] = version
		i++
	}
	sort.Sort(newByAvailable(versions, "", nil))
	var versionsStringBuilder strings.Builder
	versionsStringBuilder.WriteString("versions: ")
	for i := range versions {
		versionsStringBuilder.WriteString(versions[i])
		versionsStringBuilder.WriteString(fmt.Sprintf(": %d/%d || ", (*s)[versions[i]].Replicas, (*s)[versions[i]].ReadyReplicas))
	}
	return versionsStringBuilder.String()
}

func (s *ShardScheduleParams) String() string {

	return fmt.Sprintf("[%s] rollingset initScheduleArgs latestVersion: %s,%s curReplicas: %v, curLatestReplicas: %v, curOldReplicas: %v,"+
		"Desired: %v, maxSurge: %v,  minAvailable: %v, maxUnavailable: %v, maxReplicas: %v, latestVersionRatio: %v, targetLatestReplicas: %v, targetOldReplicas: %v",
		s.name, s.LatestVersion, s.LatestGroupVersion, s.CurTotalReplicas, s.CurLatestReplicas, s.CurOldReplicas, s.Desired, s.maxSurge,
		s.minAvailable, s.maxUnavailable, s.maxReplicas, s.latestVersionRatio, s.targetLatestReplicas, s.targetOldReplicas)
}

func printShardTargetMatrix(targets map[string]int32, groupVersionToVersionMap map[string]string, sortedKeys []string, versionSortedKeys []string) string {
	var groupVersionTargets = map[string]int32{}
	if 0 == len(groupVersionToVersionMap) {
		groupVersionTargets = targets
	} else {
		for groupVersion, version := range groupVersionToVersionMap {
			groupVersionTargets[groupVersion] = targets[version]
		}
	}
	var versionsStringBuilder strings.Builder
	if 0 != len(sortedKeys) {
		versionsStringBuilder.WriteString("versions: ")
		for i := range sortedKeys {
			versionsStringBuilder.WriteString(sortedKeys[i])
			versionsStringBuilder.WriteString(fmt.Sprintf(": %d || ", groupVersionTargets[sortedKeys[i]]))
		}
	} else if 0 != len(versionSortedKeys) {
		versionsStringBuilder.WriteString("versions: ")
		for i := range versionSortedKeys {
			versionsStringBuilder.WriteString(versionSortedKeys[i])
			versionsStringBuilder.WriteString(fmt.Sprintf(": %d || ", targets[versionSortedKeys[i]]))
		}
	}
	return versionsStringBuilder.String()
}

func printVersionStatus(status map[string]*VersionStatus) string {
	versionStatus := getversionsStatus(status)
	return versionStatus.String()
}

func getZapFields(name string, metas map[string]string) []zap.Field {
	var fields = make([]zap.Field, 0, len(metas)+1)
	field := zap.String(zapm.MsgUKey, name)
	fields = append(fields, field)
	for k, v := range metas {
		field = zap.String(k, v)
		fields = append(fields, field)
	}
	return fields
}
