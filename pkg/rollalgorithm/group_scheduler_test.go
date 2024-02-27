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
	"reflect"
	"testing"

	"github.com/alibaba/kube-sharding/common/features"
	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	app "k8s.io/api/apps/v1"
	apps "k8s.io/api/apps/v1"
)

func newGroupScheduleParams(name string, latestVersion, maxSurge, maxUnavailable string, latestPercent int32,
	shards []Shard, targetShardStrategys map[string]*app.DeploymentStrategy,
) *GroupScheduleParams {
	maxSurgePercent, maxUnavailablePercent, _ := ResolveFenceposts(
		getIntstrPtr(maxSurge), getIntstrPtr(maxUnavailable), 100)

	var strategy = apps.DeploymentStrategy{
		RollingUpdate: &apps.RollingUpdateDeployment{
			MaxUnavailable: getIntstrPtr(maxUnavailable),
			MaxSurge:       getIntstrPtr(maxSurge),
		},
	}
	return &GroupScheduleParams{
		Name:                 name,
		TargetShardStrategys: targetShardStrategys,
		DefaultStrategy:      strategy,
		LatestPercent:        latestPercent,
		MaxSurge:             maxSurgePercent,
		MaxUnavailable:       maxUnavailablePercent,
		ShardGroupVersion:    latestVersion,
		ShardStatus:          shards,
	}
}

func newGroupScheduleParamsWithNewShards(name string, latestVersion, maxSurge, maxUnavailable string, latestPercent int32,
	shards []Shard, targetShardStrategys map[string]*app.DeploymentStrategy, newShards []string,
) *GroupScheduleParams {
	maxSurgePercent, maxUnavailablePercent, _ := ResolveFenceposts(
		getIntstrPtr(maxSurge), getIntstrPtr(maxUnavailable), 100)

	var strategy = apps.DeploymentStrategy{
		RollingUpdate: &apps.RollingUpdateDeployment{
			MaxUnavailable: getIntstrPtr(maxUnavailable),
			MaxSurge:       getIntstrPtr(maxSurge),
		},
	}
	return &GroupScheduleParams{
		Name:                 name,
		TargetShardStrategys: targetShardStrategys,
		DefaultStrategy:      strategy,
		LatestPercent:        latestPercent,
		MaxSurge:             maxSurgePercent,
		MaxUnavailable:       maxUnavailablePercent,
		ShardGroupVersion:    latestVersion,
		ShardStatus:          shards,
		NewShards:            newShards,
	}
}

func TestGroupSyncSlidingScheduler_Schedule(t *testing.T) {
	type args struct {
		params            *GroupScheduleParams
		groupVersionMap   map[string]string
		rollingStrategy   string
		disableMissColume bool
	}
	scaleOutWithLatestVersionCount = true
	tests := []struct {
		name    string
		args    args
		want    map[string]ShardScheduleParams
		want1   int32
		want2   map[string]map[string]int32
		want3   map[string]map[string]int32
		wantErr bool
	}{
		{
			name: "miss colume",
			args: args{
				params: newGroupScheduleParams(
					"group", "453-2c0bc462f0e41079ffc6bc3307c1f0e4", "20%", "0%", 100,
					[]Shard{
						newShardStatus(t, "test1", 1, map[string]*VersionStatus{
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(1, 0),
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 0),
						}, 0, 0),
						newShardStatus(t, "test2", 1, map[string]*VersionStatus{
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(1, 0),
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 0),
						}, 0, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test1": newStrategy1("0%", "20%"),
						"test2": newStrategy1("0%", "20%"),
					},
				),
				groupVersionMap: map[string]string{
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
				},
				rollingStrategy:   RollingStrategyOldFirst,
				disableMissColume: true,
			},
			want1: 50,
			want2: map[string]map[string]int32{
				"test1": {"451": 1, "453": 1},
				"test2": {"451": 1, "453": 1},
			},
			wantErr: false,
		},
		{
			name: "miss colume",
			args: args{
				params: newGroupScheduleParams(
					"group", "453-2c0bc462f0e41079ffc6bc3307c1f0e4", "20%", "0%", 100,
					[]Shard{
						newShardStatus(t, "test1", 1, map[string]*VersionStatus{
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(1, 0),
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 0),
						}, 0, 0),
						newShardStatus(t, "test2", 1, map[string]*VersionStatus{
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(1, 0),
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 0),
						}, 0, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test1": newStrategy1("0%", "20%"),
						"test2": newStrategy1("0%", "20%"),
					},
				),
				groupVersionMap: map[string]string{
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
				},
				rollingStrategy: RollingStrategyOldFirst,
			},
			want1: 50,
			want2: map[string]map[string]int32{
				"test1": {"451": 0, "453": 2},
				"test2": {"451": 0, "453": 2},
			},
			wantErr: false,
		},
		{
			name: "oldFirst1",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "10%", "25%", 100,
					[]Shard{
						newShardStatus(t, "test1", 4, map[string]*VersionStatus{
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(2, 0),
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 0),
							"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 1),
						}, 0, 0),
						newShardStatus(t, "test2", 4, map[string]*VersionStatus{
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(2, 2),
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 0),
							"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 0),
						}, 0, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test1": newStrategy("50%"),
						"test2": newStrategy("50%"),
					},
				),
				groupVersionMap: map[string]string{
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
				},
				rollingStrategy: RollingStrategyOldFirst,
			},
			want1: 25,
			want2: map[string]map[string]int32{
				"test1": {"451": 2, "453": 1, "455": 0, "458": 1},
				"test2": {"451": 2, "453": 1, "455": 0, "458": 1},
			},
			wantErr: false,
		},
		{
			name: "oldFirst2",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "10%", "25%", 100,
					[]Shard{
						newShardStatus(t, "test1", 4, map[string]*VersionStatus{
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(1, 0),
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 0),
							"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(2, 2),
						}, 0, 0),
						newShardStatus(t, "test2", 4, map[string]*VersionStatus{
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(1, 1),
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
							"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(2, 0),
						}, 0, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test1": newStrategy("50%"),
						"test2": newStrategy("50%"),
					},
				),
				groupVersionMap: map[string]string{
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
				},
				rollingStrategy: RollingStrategyOldFirst,
			},
			want1: 25,
			want2: map[string]map[string]int32{
				"test1": {"451": 1, "453": 1, "455": 1, "458": 1},
				"test2": {"451": 1, "453": 1, "455": 1, "458": 1},
			},
			wantErr: false,
		},
		{
			name: "oldFirst3",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "10%", "25%", 100,
					[]Shard{
						newShardStatus(t, "test1", 4, map[string]*VersionStatus{
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(1, 1),
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
							"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(2, 1),
						}, 0, 0),
						newShardStatus(t, "test2", 4, map[string]*VersionStatus{
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(1, 1),
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
							"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(2, 1),
						}, 0, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test1": newStrategy("50%"),
						"test2": newStrategy("50%"),
					},
				),
				groupVersionMap: map[string]string{
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
				},
				rollingStrategy: RollingStrategyOldFirst,
			},
			want1: 25,
			want2: map[string]map[string]int32{
				"test1": {"451": 1, "453": 1, "455": 1, "458": 1},
				"test2": {"451": 1, "453": 1, "455": 1, "458": 1},
			},
			wantErr: false,
		},
		{
			name: "oldFirst4",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "10%", "50%", 100,
					[]Shard{
						newShardStatus(t, "test1", 2, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 0),
							"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 0),
						}, 0, 0),
						newShardStatus(t, "test2", 2, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 0),
							"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 0),
						}, 0, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test1": newStrategy("50%"),
						"test2": newStrategy("50%"),
					},
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
				},
				rollingStrategy: RollingStrategyOldFirst,
			},
			want1: 50,
			want2: map[string]map[string]int32{
				"test1": {"453": 1, "455": 0, "458": 1},
				"test2": {"453": 1, "455": 0, "458": 1},
			},
			wantErr: false,
		},
		{
			name: "hold",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "10%", "40%", 100,
					[]Shard{
						newShardStatus(t, "test1", 4, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 0),
							"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 0),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(1, 0),
						}, 0, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test1": newStrategy("40%"),
					},
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
				},
			},
			want1: 40,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"453": 2, "455": 1, "458": 1},
			},
			wantErr: false,
		},
		{
			name: "hold",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "10%", "20%", 100,
					[]Shard{
						newShardStatus(t, "test1", 8, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 2),
							"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(6, 5),
						}, 0, 0),
						newShardStatus(t, "test2", 8, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 2),
							"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(6, 6),
						}, 1, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test1": newStrategy("20%"),
						"test2": newStrategy("20%"),
					},
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
				},
			},
			want1: 20,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"453": 2, "455": 5, "458": 1},
				"test2": map[string]int32{"453": 2, "455": 5, "458": 1},
			},
			want3: map[string]map[string]int32{
				"test1": nil,
				"test2": map[string]int32{"453": 2, "455": 5},
			},
			wantErr: false,
		},
		{
			name: "hold",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "10%", "20%", 100,
					[]Shard{
						newShardStatus(t, "test1", 8, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 2),
							"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(6, 6),
						}, 0, 0),
						newShardStatus(t, "test2", 8, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 1),
							"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(6, 6),
						}, 1, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test1": newStrategy("20%"),
						"test2": newStrategy("20%"),
					},
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
				},
			},
			want1: 20,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"453": 1, "455": 6, "458": 1},
				"test2": map[string]int32{"453": 1, "455": 6, "458": 1},
			},
			want3: map[string]map[string]int32{
				"test1": nil,
				"test2": map[string]int32{"453": 1, "455": 6},
			},
			wantErr: false,
		},
		{
			name: "3 and 5 rows round 1",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "10%", "20%", 100,
					[]Shard{
						newShardStatus(t, "test1", 3, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(3, 3),
						}, 0, 0),
						newShardStatus(t, "test2", 5, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(5, 5),
						}, 1, 0),
					},
					nil,
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 20,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"453": 3, "458": 1},
				"test2": map[string]int32{"453": 4, "458": 1, "455": 0, "451": 0},
			},
			want3: map[string]map[string]int32{
				"test1": nil,
				"test2": map[string]int32{"453": 4},
			},
			wantErr: false,
		},
		{
			name: "3 and 5 rows round 2",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "10%", "20%", 100,
					[]Shard{
						newShardStatus(t, "test1", 3, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(3, 3),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(1, 1),
						}, 0, 0),
						newShardStatus(t, "test2", 5, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(4, 4),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(1, 1),
						}, 1, 0),
					},
					nil,
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 40,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"453": 2, "458": 2, "455": 0, "451": 0},
				"test2": map[string]int32{"453": 3, "458": 2, "455": 0, "451": 0},
			},
			want3: map[string]map[string]int32{
				"test1": nil,
				"test2": map[string]int32{"453": 3, "458": 2},
			},
			wantErr: false,
		},
		{
			name: "3 and 5 rows round 3",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "10%", "20%", 100,
					[]Shard{
						newShardStatus(t, "test1", 3, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 2),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(2, 2),
						}, 0, 0),
						newShardStatus(t, "test2", 5, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(3, 3),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(2, 2),
						}, 1, 0),
					},
					nil,
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 60,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"453": 1, "458": 3, "455": 0, "451": 0},
				"test2": map[string]int32{"453": 2, "458": 3, "455": 0, "451": 0},
			},
			want3: map[string]map[string]int32{
				"test1": nil,
				"test2": map[string]int32{"453": 2, "458": 3},
			},
			wantErr: false,
		},
		{
			name: "3 and 200 rows round 4",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "10%", "20%", 100,
					[]Shard{
						newShardStatus(t, "test1", 3, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(3, 3),
						}, 0, 0),
						newShardStatus(t, "test2", 5, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 2),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(3, 3),
						}, 1, 0),
					},
					nil,
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 80,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"453": 1, "458": 3, "455": 0, "451": 0},
				"test2": map[string]int32{"453": 1, "458": 4, "455": 0, "451": 0},
			},
			want3: map[string]map[string]int32{
				"test1": nil,
				"test2": map[string]int32{"453": 1, "458": 4},
			},
			wantErr: false,
		},
		{
			name: "3 and 200 rows round 5",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "10%", "20%", 100,
					[]Shard{
						newShardStatus(t, "test1", 3, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(3, 3),
						}, 0, 0),
						newShardStatus(t, "test2", 5, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(4, 4),
						}, 1, 0),
					},
					nil,
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 100,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"453": 0, "458": 4, "455": 0, "451": 0},
				"test2": map[string]int32{"453": 0, "458": 5, "455": 0, "451": 0},
			},
			want3: map[string]map[string]int32{
				"test1": nil,
				"test2": map[string]int32{"453": 0, "458": 5},
			},
			wantErr: false,
		},
		{
			name: "3 and 200 rows round 6",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "10%", "20%", 100,
					[]Shard{
						newShardStatus(t, "test1", 3, map[string]*VersionStatus{
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(4, 4),
						}, 0, 0),
						newShardStatus(t, "test2", 5, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(4, 4),
						}, 1, 0),
					},
					nil,
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 100,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"458": 3},
				"test2": map[string]int32{"453": 0, "458": 5, "455": 0, "451": 0},
			},
			want3: map[string]map[string]int32{
				"test1": nil,
				"test2": map[string]int32{"458": 5},
			},
			wantErr: false,
		},
		{
			name: "3 and 5 rows round 1",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "10%", "20%", 100,
					[]Shard{
						newShardStatus(t, "test1", 3, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(3, 3),
						}, 0, 0),
						newShardStatus(t, "test2", 5, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(5, 5),
						}, 1, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test1": newStrategy("34%"),
					},
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 20,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"453": 2, "458": 1, "455": 0, "451": 0},
				"test2": map[string]int32{"453": 4, "458": 1, "455": 0, "451": 0},
			},
			want3: map[string]map[string]int32{
				"test1": nil,
				"test2": map[string]int32{"453": 4},
			},
			wantErr: false,
		},
		{
			name: "3 and 5 rows round 2",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "10%", "20%", 100,
					[]Shard{
						newShardStatus(t, "test1", 3, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 2),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(1, 1),
						}, 0, 0),
						newShardStatus(t, "test2", 5, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(4, 4),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(1, 1),
						}, 1, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test1": newStrategy("34%"),
					},
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 40,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"453": 2, "458": 1},
				"test2": map[string]int32{"453": 3, "458": 2, "455": 0, "451": 0},
			},
			want3: map[string]map[string]int32{
				"test1": nil,
				"test2": map[string]int32{"453": 3, "458": 2},
			},
			wantErr: false,
		},
		{
			name: "3 and 5 rows round 3",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "10%", "20%", 100,
					[]Shard{
						newShardStatus(t, "test1", 3, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 2),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(1, 1),
						}, 0, 0),
						newShardStatus(t, "test2", 5, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(3, 3),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(2, 2),
						}, 1, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test1": newStrategy("34%"),
					},
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 54,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"451": 0, "453": 1, "455": 0, "458": 2},
				"test2": map[string]int32{"453": 3, "458": 2},
			},
			want3: map[string]map[string]int32{
				"test1": nil,
				"test2": map[string]int32{"453": 3, "458": 2},
			},
			wantErr: false,
		},
		{
			name: "3 and 5 rows round 4",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "10%", "20%", 100,
					[]Shard{
						newShardStatus(t, "test1", 3, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(2, 2),
						}, 0, 0),
						newShardStatus(t, "test2", 5, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(3, 3),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(2, 2),
						}, 1, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test1": newStrategy("34%"),
					},
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 60,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"453": 1, "458": 2},
				"test2": map[string]int32{"453": 2, "458": 3, "455": 0, "451": 0},
			},
			want3: map[string]map[string]int32{
				"test1": nil,
				"test2": map[string]int32{"453": 2, "458": 3},
			},
			wantErr: false,
		},
		{
			name: "3 and 5 rows round 5",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "10%", "20%", 100,
					[]Shard{
						newShardStatus(t, "test1", 3, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(2, 2),
						}, 0, 0),
						newShardStatus(t, "test2", 5, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 2),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(3, 3),
						}, 1, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test1": newStrategy("34%"),
					},
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 80,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"453": 1, "458": 2},
				"test2": map[string]int32{"453": 1, "458": 4, "455": 0, "451": 0},
			},
			want3: map[string]map[string]int32{
				"test1": nil,
				"test2": map[string]int32{"453": 1, "458": 4},
			},
			wantErr: false,
		},
		{
			name: "3 and 5 rows round 6",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "10%", "20%", 100,
					[]Shard{
						newShardStatus(t, "test1", 3, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(2, 2),
						}, 0, 0),
						newShardStatus(t, "test2", 5, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(4, 4),
						}, 1, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test1": newStrategy("34%"),
					},
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 87,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"451": 0, "453": 0, "455": 0, "458": 3},
				"test2": map[string]int32{"453": 1, "458": 4},
			},
			want3: map[string]map[string]int32{
				"test1": nil,
				"test2": map[string]int32{"453": 1, "458": 4},
			},
			wantErr: false,
		},
		{
			name: "3 and 5 rows round 5",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "10%", "20%", 100,
					[]Shard{
						newShardStatus(t, "test1", 3, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(0, 0),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(3, 3),
						}, 0, 0),
						newShardStatus(t, "test2", 5, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(4, 4),
						}, 1, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test1": newStrategy("34%"),
					},
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 100,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"453": 0, "458": 3},
				"test2": map[string]int32{"453": 0, "458": 5, "455": 0, "451": 0},
			},
			want3: map[string]map[string]int32{
				"test1": nil,
				"test2": map[string]int32{"453": 0, "458": 5},
			},
			wantErr: false,
		},
		{
			name: "init",
			args: args{
				params: nil,
			},
			wantErr: true,
		},
		{
			name: "scacle to zero",
			args: args{
				params: newGroupScheduleParams(
					"group", "453-2c0bc462f0e41079ffc6bc3307c1f0e4", "50%", "50%", 100,
					[]Shard{
						newShardStatus(t, "test1", 0, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
						}, 0, 0),
						newShardStatus(t, "test2", 0, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
						}, 1, 0),
					},
					nil,
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 100,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"453": 0},
				"test2": map[string]int32{"453": 0},
			},
			want3: map[string]map[string]int32{
				"test1": nil,
				"test2": nil,
			},
			wantErr: false,
		},
		{
			name: "scacle to zero",
			args: args{
				params: newGroupScheduleParams(
					"group", "453-2c0bc462f0e41079ffc6bc3307c1f0e4", "50%", "50%", 100,
					[]Shard{
						newShardStatus(t, "test1", 0, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(1, 1),
						}, 0, 0),
						newShardStatus(t, "test2", 0, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(1, 1),
						}, 1, 0),
					},
					nil,
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 50,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"453": 0, "458": 0, "451": 0, "455": 0},
				"test2": map[string]int32{"453": 0, "458": 0, "451": 0, "455": 0},
			},
			want3: map[string]map[string]int32{
				"test1": nil,
				"test2": nil,
			},
			wantErr: false,
		},
		{
			name: "one role zero",
			args: args{
				params: newGroupScheduleParams(
					"group", "453-2c0bc462f0e41079ffc6bc3307c1f0e4", "50%", "50%", 100,
					[]Shard{
						newShardStatus(t, "test1", 2, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(1, 1),
						}, 0, 0),
						newShardStatus(t, "test2", 0, map[string]*VersionStatus{}, 1, 0),
					},
					nil,
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 100,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"453": 2, "458": 0, "451": 0, "455": 0},
				"test2": map[string]int32{},
			},
			want3: map[string]map[string]int32{
				"test1": nil,
				"test2": nil,
			},
			wantErr: false,
		},
		{
			name: "one role zero",
			args: args{
				params: newGroupScheduleParams(
					"group", "453-2c0bc462f0e41079ffc6bc3307c1f0e4", "50%", "50%", 100,
					[]Shard{
						newShardStatus(t, "test1", 2, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(1, 1),
						}, 0, 0),
						newShardStatus(t, "test2", 0, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
							"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 1),
						}, 1, 0),
					},
					nil,
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 100,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"453": 2, "458": 0, "451": 0, "455": 0},
				"test2": map[string]int32{"453": 0, "455": 0, "451": 0, "458": 0},
			},
			want3: map[string]map[string]int32{
				"test1": nil,
				"test2": nil,
			},
			wantErr: false,
		},
		{
			name: "one role",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "10%", "8%", 100,
					[]Shard{
						newShardStatus(t, "test1", 10, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(10, 10),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(1, 1),
						}, 0, 0),
					},
					nil,
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
				},
			},
			want1: 18,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"453": 9, "458": 2},
			},
			want3: map[string]map[string]int32{
				"test1": nil,
				"test2": nil,
			},
			wantErr: false,
		},
		{
			name: "roll",
			args: args{
				params: newGroupScheduleParams(
					"group", "451-4abe5b7fe1a035cac4d1f7e6c13bed9a", "30%", "50%", 50,
					[]Shard{
						newShardStatus(t, "test1", 2, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(1, 1),
						}, 0, 0),
						newShardStatus(t, "test2", 2, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
							"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 1),
						}, 1, 0),
					},
					nil,
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 50,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"451": 1, "453": 1, "458": 0, "455": 0},
				"test2": map[string]int32{"451": 1, "453": 1, "455": 0, "458": 0},
			},
			want3: map[string]map[string]int32{
				"test1": nil,
				"test2": map[string]int32{"453": 1, "458": 0},
			},
			wantErr: false,
		},
		{
			name: "roll 2",
			args: args{
				params: newGroupScheduleParams(
					"group", "451-4abe5b7fe1a035cac4d1f7e6c13bed9a", "30%", "50%", 50,
					[]Shard{
						newShardStatus(t, "test1", 2, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 0),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(1, 1),
						}, 0, 0),
						newShardStatus(t, "test2", 2, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
							"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 1),
						}, 1, 0),
					},
					nil,
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 50,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"451": 1, "453": 1, "458": 0, "455": 0},
				"test2": map[string]int32{"451": 1, "453": 1, "455": 0, "458": 0},
			},
			want3: map[string]map[string]int32{
				"test1": nil,
				"test2": map[string]int32{"453": 0, "458": 0},
			},
			wantErr: false,
		},
		{
			name: "3 versions select new version first",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "30%", "20%", 50,
					[]Shard{
						newShardStatus(t, "test1", 7, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(5, 5),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(1, 1),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(1, 1),
						}, 0, 0),
						newShardStatus(t, "test2", 7, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(5, 5),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(1, 1),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(1, 1),
						}, 1, 0),
					},
					nil,
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 35,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"453": 5, "451": 0, "458": 2, "455": 0},
				"test2": map[string]int32{"453": 5, "451": 0, "458": 2, "455": 0},
			},
			want3: map[string]map[string]int32{
				"test1": nil,
				"test2": map[string]int32{"451": 0, "453": 5, "458": 1},
			},
			wantErr: false,
		},
		{
			name: "3 versions not alignment",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "30%", "20%", 50,
					[]Shard{
						newShardStatus(t, "test1", 7, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(5, 5),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(1, 1),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(1, 1),
						}, 0, 0),
						newShardStatus(t, "test2", 7, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(5, 5),
							"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 1),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(1, 1),
						}, 1, 0),
					},
					nil,
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 35,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"453": 5, "451": 0, "455": 0, "458": 2},
				"test2": map[string]int32{"451": 0, "453": 5, "455": 0, "458": 2},
			},
			want3: map[string]map[string]int32{
				"test1": nil,
				"test2": map[string]int32{"451": 0, "453": 5, "458": 1},
			},
			wantErr: false,
		},
		{
			name: "3 versions block by maxunavailable",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "30%", "20%", 50,
					[]Shard{
						newShardStatus(t, "test1", 10, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(5, 5),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(3, 3),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(2, 0),
						}, 0, 0),
						newShardStatus(t, "test2", 10, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(5, 5),
							"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(3, 3),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(2, 2),
						}, 1, 0),
					},
					nil,
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 20,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"453": 5, "451": 2, "458": 3},
				"test2": map[string]int32{"453": 5, "455": 2, "458": 3},
			},
			want3: map[string]map[string]int32{
				"test1": nil,
				"test2": map[string]int32{"451": 0, "453": 5, "458": 0},
			},
			wantErr: false,
		},
		{
			name: "3 versions block by maxunavailable",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "30%", "30%", 50,
					[]Shard{
						newShardStatus(t, "test1", 10, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(5, 5),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(3, 3),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(2, 0),
						}, 0, 0),
						newShardStatus(t, "test2", 10, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(5, 5),
							"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(3, 3),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(2, 2),
						}, 1, 0),
					},
					nil,
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 30,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"453": 5, "451": 1, "455": 0, "458": 4},
				"test2": map[string]int32{"453": 5, "455": 1, "451": 0, "458": 4},
			},
			want3: map[string]map[string]int32{
				"test1": nil,
				"test2": map[string]int32{"451": 0, "453": 5, "458": 0},
			},
			wantErr: false,
		},
		{
			name: "3 versions block by latestversion ratio",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "30%", "50%", 40,
					[]Shard{
						newShardStatus(t, "test1", 10, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(5, 5),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(3, 3),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(2, 0),
						}, 0, 0),
						newShardStatus(t, "test2", 10, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(5, 5),
							"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(3, 3),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(2, 2),
						}, 1, 0),
					},
					nil,
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 40,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"453": 5, "451": 1, "455": 0, "458": 4},
				"test2": map[string]int32{"453": 5, "455": 1, "451": 0, "458": 4},
			},
			want3: map[string]map[string]int32{
				"test1": nil,
				"test2": map[string]int32{"451": 0, "453": 5, "458": 0},
			},
			wantErr: false,
		},
		{
			name: "3 versions roll and scale in",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "30%", "20%", 50,
					[]Shard{
						newShardStatus(t, "test1", 7, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(50, 50),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(10, 10),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(10, 10),
						}, 0, 0),
						newShardStatus(t, "test2", 7, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(50, 50),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(10, 10),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(10, 10),
						}, 1, 0),
					},
					nil,
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 50,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"453": 0, "451": 3, "455": 0, "458": 4},
				"test2": map[string]int32{"453": 0, "451": 3, "455": 0, "458": 4},
			},
			want3: map[string]map[string]int32{
				"test1": nil,
				"test2": map[string]int32{"451": 3, "453": 0, "458": 4},
			},
			wantErr: false,
		},
		{
			name: "3 versions roll and scale in 2",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "30%", "20%", 50,
					[]Shard{
						newShardStatus(t, "test1", 7, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(50, 0),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(10, 5),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(10, 10),
						}, 0, 0),
						newShardStatus(t, "test2", 7, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(50, 10),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(10, 0),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(10, 10),
						}, 1, 0),
					},
					nil,
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 50,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"453": 0, "451": 3, "455": 0, "458": 4},
				"test2": map[string]int32{"453": 0, "451": 3, "455": 0, "458": 4},
			},
			want3: map[string]map[string]int32{
				"test1": nil,
				"test2": map[string]int32{"451": 3, "453": 0, "458": 4},
			},
			wantErr: false,
		},
		{
			name: "3 versions roll and scale in",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "30%", "20%", 50,
					[]Shard{
						newShardStatus(t, "test1", 7, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(50, 0),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(10, 0),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(10, 0),
						}, 0, 0),
						newShardStatus(t, "test2", 7, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(50, 0),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(10, 0),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(10, 0),
						}, 1, 0),
					},
					nil,
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 20,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"453": 0, "451": 5, "455": 0, "458": 2},
				"test2": map[string]int32{"453": 0, "451": 5, "455": 0, "458": 2},
			},
			want3: map[string]map[string]int32{
				"test1": nil,
				"test2": map[string]int32{"451": 0, "453": 0, "458": 0},
			},
			wantErr: false,
		},
		{
			name: "3 versions roll and scale in one row",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "30%", "20%", 50,
					[]Shard{
						newShardStatus(t, "test1", 7, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(50, 50),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(10, 10),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(10, 10),
						}, 0, 0),
						newShardStatus(t, "test2", 70, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(50, 50),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(10, 10),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(10, 10),
						}, 1, 0),
					},
					nil,
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 35,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"453": 4, "451": 0, "455": 0, "458": 3},
				"test2": map[string]int32{"453": 46, "451": 0, "455": 0, "458": 24},
			},
			want3: map[string]map[string]int32{
				"test1": nil,
				"test2": map[string]int32{"451": 0, "453": 46, "458": 24},
			},
			wantErr: false,
		},
		{
			name: "3 versions roll and scale out",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "30%", "20%", 50,
					[]Shard{
						newShardStatus(t, "test1", 70, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(5, 5),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(1, 1),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(1, 1),
						}, 0, 0),
						newShardStatus(t, "test2", 70, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(5, 5),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(1, 1),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(1, 1),
						}, 1, 0),
					},
					nil,
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 50,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"453": 34, "451": 1, "458": 35},
				"test2": map[string]int32{"453": 34, "451": 1, "458": 35},
			},
			want3: map[string]map[string]int32{
				"test1": nil,
				"test2": map[string]int32{"451": 1, "453": 34, "458": 10},
			},
			wantErr: false,
		},
		{
			name: "3 versions roll and scale out one row",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "30%", "20%", 50,
					[]Shard{
						newShardStatus(t, "test1", 70, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(5, 5),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(1, 1),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(1, 1),
						}, 0, 0),
						newShardStatus(t, "test2", 7, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(5, 5),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(1, 1),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(1, 1),
						}, 1, 0),
					},
					nil,
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 35,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"453": 44, "451": 1, "458": 25},
				"test2": map[string]int32{"453": 5, "451": 0, "455": 0, "458": 2},
			},
			want3: map[string]map[string]int32{
				"test1": nil,
				"test2": map[string]int32{"451": 0, "453": 5, "458": 1},
			},
			wantErr: false,
		},
		{
			name: "3 versions roll back",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "30%", "20%", 0,
					[]Shard{
						newShardStatus(t, "test1", 7, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 2),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(1, 1),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(4, 4),
						}, 0, 0),
						newShardStatus(t, "test2", 7, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 2),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(1, 1),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(4, 4),
						}, 1, 0),
					},
					nil,
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 38,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"453": 2, "451": 2, "458": 3},
				"test2": map[string]int32{"453": 2, "451": 2, "458": 3},
			},
			want3: map[string]map[string]int32{
				"test1": nil,
				"test2": map[string]int32{"451": 1, "453": 2, "458": 3},
			},
			wantErr: false,
		},
		{
			name: "3 versions roll back 2",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "30%", "20%", 0,
					[]Shard{
						newShardStatus(t, "test1", 7, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 2),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(1, 1),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(4, 2),
						}, 0, 0),
						newShardStatus(t, "test2", 7, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 2),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(1, 1),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(4, 1),
						}, 1, 0),
					},
					nil,
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 38,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"453": 2, "451": 2, "458": 3},
				"test2": map[string]int32{"453": 2, "451": 2, "458": 3},
			},
			want3: map[string]map[string]int32{
				"test1": nil,
				"test2": map[string]int32{"451": 1, "453": 2, "458": 3},
			},
			wantErr: false,
		},
		{
			name: "3 versions roll back and block by maxunavaiable",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "30%", "20%", 0,
					[]Shard{
						newShardStatus(t, "test1", 7, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 2),
							"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 1),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(4, 4),
						}, 0, 0),
						newShardStatus(t, "test2", 7, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 2),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(1, 1),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(4, 4),
						}, 1, 0),
					},
					nil,
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 52,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"453": 2, "455": 1, "458": 4},
				"test2": map[string]int32{"453": 2, "451": 1, "458": 4},
			},
			want3: map[string]map[string]int32{
				"test1": nil,
				"test2": map[string]int32{"455": 0, "453": 2, "458": 4},
			},
			wantErr: false,
		},
		{
			name: "3 and 200 rows floor round 1",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "30%", "34%", 40,
					[]Shard{
						newShardStatus(t, "test1", 3, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 2),
							"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 1),
						}, 0, 0),
						newShardStatus(t, "test2", 200, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(190, 190),
							"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(10, 10),
						}, 1, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test2": newStrategy("10%"),
					},
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 34,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"453": 2, "455": 0, "451": 0, "458": 1},
				"test2": map[string]int32{"453": 180, "455": 0, "451": 0, "458": 20},
			},
			want3: map[string]map[string]int32{
				"test1": nil,
				"test2": map[string]int32{"455": 0, "453": 134},
			},
			wantErr: false,
		},

		{
			name: "3 and 200 rows floor round 2",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "30%", "34%", 100,
					[]Shard{
						newShardStatus(t, "test1", 3, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 2),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(1, 1),
						}, 0, 0),
						newShardStatus(t, "test2", 200, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(180, 180),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(20, 20),
						}, 1, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test2": newStrategy("10%"),
					},
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 44,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"453": 2, "458": 1},
				"test2": map[string]int32{"453": 160, "458": 40, "455": 0, "451": 0},
			},
			want3: map[string]map[string]int32{
				"test1": nil,
				"test2": map[string]int32{"453": 134, "458": 40},
			},
			wantErr: false,
		},

		{
			name: "3 and 200 rows floor round 3",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "30%", "34%", 100,
					[]Shard{
						newShardStatus(t, "test1", 3, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 2),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(1, 1),
						}, 0, 0),
						newShardStatus(t, "test2", 200, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(160, 160),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(40, 40),
						}, 1, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test2": newStrategy("10%"),
					},
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 54,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"453": 2, "458": 1},
				"test2": map[string]int32{"453": 140, "458": 60, "455": 0, "451": 0},
			},
			want3: map[string]map[string]int32{
				"test1": nil,
				"test2": map[string]int32{"453": 134, "458": 60},
			},
			wantErr: false,
		},
		{
			name: "3 and 200 rows floor round 4",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "30%", "34%", 100,
					[]Shard{
						newShardStatus(t, "test1", 3, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 2),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(1, 1),
						}, 0, 0),
						newShardStatus(t, "test2", 200, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(140, 140),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(60, 60),
						}, 1, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test2": newStrategy("10%"),
					},
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 64,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"453": 2, "458": 1},
				"test2": map[string]int32{"453": 120, "458": 80, "455": 0, "451": 0},
			},
			want3: map[string]map[string]int32{
				"test1": nil,
				"test2": map[string]int32{"453": 120, "458": 67},
			},
			wantErr: false,
		},
		{
			name: "3 and 200 rows floor round 5",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "30%", "34%", 100,
					[]Shard{
						newShardStatus(t, "test1", 3, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 2),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(1, 1),
						}, 0, 0),
						newShardStatus(t, "test2", 200, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(133, 133),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(67, 67),
						}, 1, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test2": newStrategy("10%"),
					},
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 68,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"453": 1, "458": 2, "455": 0, "451": 0},
				"test2": map[string]int32{"453": 113, "458": 87, "455": 0, "451": 0},
			},
			want3: map[string]map[string]int32{
				"test1": nil,
				"test2": map[string]int32{"453": 113, "458": 67},
			},
			wantErr: false,
		},
		{
			name: "3 and 200 rows round 1",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "30%", "10%", 100,
					[]Shard{
						newShardStatus(t, "test1", 3, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(3, 3),
						}, 0, 0),
						newShardStatus(t, "test2", 200, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(200, 200),
						}, 1, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test1": newStrategy("34%"),
					},
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 10,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"453": 2, "458": 1, "455": 0, "451": 0},
				"test2": map[string]int32{"453": 180, "458": 20, "455": 0, "451": 0},
			},
			want3: map[string]map[string]int32{
				"test1": nil,
				"test2": map[string]int32{"453": 180},
			},
			wantErr: false,
		},
		{
			name: "3 and 200 rows round 2",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "30%", "10%", 100,
					[]Shard{
						newShardStatus(t, "test1", 3, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 2),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(1, 1),
						}, 0, 0),
						newShardStatus(t, "test2", 200, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(180, 180),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(20, 20),
						}, 1, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test1": newStrategy("34%"),
					},
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 20,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"453": 2, "458": 1},
				"test2": map[string]int32{"453": 160, "458": 40, "455": 0, "451": 0},
			},
			want3: map[string]map[string]int32{
				"test1": nil,
				"test2": map[string]int32{"453": 134, "458": 40},
			},
			wantErr: false,
		},
		{
			name: "3 and 200 rows round 3",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "30%", "10%", 100,
					[]Shard{
						newShardStatus(t, "test1", 3, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 2),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(1, 1),
						}, 0, 0),
						newShardStatus(t, "test2", 200, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(160, 160),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(40, 40),
						}, 1, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test1": newStrategy("34%"),
					},
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 30,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"453": 2, "458": 1},
				"test2": map[string]int32{"453": 140, "458": 60, "455": 0, "451": 0},
			},
			want3: map[string]map[string]int32{
				"test1": nil,
				"test2": map[string]int32{"453": 134, "458": 60},
			},
			wantErr: false,
		},
		{
			name: "3 and 200 rows round 4",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "30%", "10%", 100,
					[]Shard{
						newShardStatus(t, "test1", 3, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 2),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(1, 1),
						}, 0, 0),
						newShardStatus(t, "test2", 200, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(133, 133),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(67, 67),
						}, 1, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test1": newStrategy("34%"),
					},
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 44,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"453": 1, "458": 2, "455": 0, "451": 0},
				"test2": map[string]int32{"453": 113, "458": 87, "451": 0, "455": 0},
			},
			want3: map[string]map[string]int32{
				"test1": nil,
				"test2": map[string]int32{"453": 113, "458": 67},
			},
			wantErr: false,
		},
		{
			name: "padded 1",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "10%", "30%", 100,
					[]Shard{
						newShardStatus(t, "test1", 5, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(0, 0),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(3, 2),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(2, 2),
						}, 0, 0),
						newShardStatus(t, "test2", 7, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(4, 4),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(2, 2),
						}, 1, 0),
						newShardStatus(t, "test3", 40, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(28, 28),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(11, 11),
						}, 2, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test1": newStrategy("34%"),
					},
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 70,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"453": 0, "458": 3, "451": 2},
				"test2": map[string]int32{"453": 0, "458": 5, "451": 2},
				"test3": map[string]int32{"453": 0, "458": 29, "451": 11},
			},
			want3: map[string]map[string]int32{
				"test1": nil,
				"test2": map[string]int32{"453": 0, "458": 3, "451": 2},
				"test3": map[string]int32{"453": 0, "458": 16, "451": 11},
			},
			wantErr: false,
		},
		{
			name: "padded with new shards",
			args: args{
				params: newGroupScheduleParamsWithNewShards(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "10%", "30%", 100,
					[]Shard{
						newShardStatus(t, "test1", 5, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(0, 0),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(3, 2),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(2, 2),
						}, 0, 0),
						newShardStatus(t, "test2", 7, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(4, 4),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(2, 2),
						}, 1, 0),
						newShardStatus(t, "test3", 40, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(28, 28),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(11, 11),
						}, 2, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test1": newStrategy("34%"),
					},
					[]string{"test1"},
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 70,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"453": 0, "458": 3, "451": 2},
				"test2": map[string]int32{"453": 1, "458": 4, "451": 2},
				"test3": map[string]int32{"453": 1, "458": 28, "451": 11},
			},
			want3: map[string]map[string]int32{
				"test1": nil,
				"test2": map[string]int32{"453": 0, "458": 3, "451": 2},
				"test3": map[string]int32{"453": 0, "458": 16, "451": 11},
			},
			wantErr: false,
		},
		{
			name: "padded with new shards 2",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "10%", "30%", 100,
					[]Shard{
						newShardStatus(t, "test1", 5, map[string]*VersionStatus{
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(5, 2),
						}, 0, 0),
						newShardStatus(t, "test2", 7, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(4, 4),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(2, 2),
						}, 1, 0),
						newShardStatus(t, "test3", 40, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(28, 28),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(11, 11),
						}, 2, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test1": newStrategy("34%"),
					},
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 70,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"458": 5},
				"test2": map[string]int32{"453": 0, "458": 6, "451": 1},
				"test3": map[string]int32{"453": 0, "458": 36, "451": 4},
			},
			want3: map[string]map[string]int32{
				"test1": nil,
				"test2": map[string]int32{"458": 3},
				"test3": map[string]int32{"453": 0, "458": 16, "451": 0},
			},
			wantErr: false,
		},
		{
			name: "padded with error new shards",
			args: args{
				params: newGroupScheduleParamsWithNewShards(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "10%", "30%", 100,
					[]Shard{
						newShardStatus(t, "test1", 5, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 0),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(3, 0),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(0, 0),
						}, 0, 0),
						newShardStatus(t, "test2", 7, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(4, 4),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(2, 2),
						}, 1, 0),
						newShardStatus(t, "test3", 40, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(28, 28),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(11, 11),
						}, 2, 0),
					},
					map[string]*app.DeploymentStrategy{},
					[]string{"test1"},
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 58,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"453": 2, "458": 3, "451": 0},
				"test2": map[string]int32{"453": 0, "458": 5, "455": 0, "451": 2},
				"test3": map[string]int32{"453": 1, "458": 24, "451": 15},
			},
			wantErr: false,
		},
		{
			name: "padded with error new shards 2",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "10%", "30%", 100,
					[]Shard{
						newShardStatus(t, "test1", 5, map[string]*VersionStatus{
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(5, 2),
						}, 0, 0),
						newShardStatus(t, "test2", 7, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(4, 4),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(2, 2),
						}, 1, 0),
						newShardStatus(t, "test3", 40, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(28, 28),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(11, 11),
						}, 2, 0),
					},
					map[string]*app.DeploymentStrategy{},
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 70,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"458": 5},
				"test2": map[string]int32{"453": 0, "455": 0, "458": 7, "451": 0},
				"test3": map[string]int32{"453": 0, "458": 36, "451": 4},
			},
			wantErr: false,
		},
		{
			name: "padded",
			args: args{
				params: newGroupScheduleParamsWithNewShards(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "10%", "25%", 100,
					[]Shard{
						newShardStatus(t, "test1", 5, map[string]*VersionStatus{
							"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 1),
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 2),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(1, 0),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(1, 1),
						}, 0, 0),
						newShardStatus(t, "test2", 16, map[string]*VersionStatus{
							"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(2, 0),
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(8, 8),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(4, 0),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(2, 2),
						}, 1, 0),
						newShardStatus(t, "test3", 4, map[string]*VersionStatus{
							"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(0, 0),
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 2),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(1, 0),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(1, 1),
						}, 2, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test1": newStrategy("34%"),
					},
					[]string{},
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 25,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"453": 2, "455": 0, "458": 2, "451": 1},
				"test2": map[string]int32{"453": 8, "455": 0, "458": 6, "451": 2},
				"test3": map[string]int32{"453": 2, "455": 0, "458": 1, "451": 1},
			},
			want3: map[string]map[string]int32{
				"test1": nil,
				"test2": map[string]int32{"453": 7, "455": 0, "458": 0, "451": 2},
				"test3": map[string]int32{"453": 2, "455": 0, "458": 0, "451": 1},
			},
			wantErr: false,
		},
		{
			name: "hold more version",
			args: args{
				params: newGroupScheduleParamsWithNewShards(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "10%", "25%", 100,
					[]Shard{
						newShardStatus(t, "test1", 6, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(5, 4),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(1, 1),
						}, 0, 0),
						newShardStatus(t, "test2", 16, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(14, 14),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(2, 2),
						}, 1, 0),
						newShardStatus(t, "test3", 4, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(3, 3),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(1, 1),
						}, 2, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test1": newStrategy("34%"),
					},
					[]string{},
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 25,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"453": 4, "455": 0, "458": 1, "451": 1},
				"test2": map[string]int32{"453": 10, "455": 0, "458": 4, "451": 2},
				"test3": map[string]int32{"453": 2, "455": 0, "458": 1, "451": 1},
			},
			want3: map[string]map[string]int32{
				"test1": nil,
				"test2": map[string]int32{"453": 10, "451": 2},
				"test3": map[string]int32{"453": 2, "451": 1},
			},
			wantErr: false,
		},
		{
			name: "carry strategy",
			args: args{
				params: newGroupScheduleParams(
					"group", "453-2c0bc462f0e41079ffc6bc3307c1f0e4", "10%", "20%", 100,
					[]Shard{
						newShardStatus(t, "test1", 3, map[string]*VersionStatus{
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(2, 2),
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 1),
						}, 0, 0),
						newShardStatus(t, "test2", 3, map[string]*VersionStatus{
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(2, 2),
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 2),
						}, 0, 0),
					},
					nil,
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 54,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"453": 2, "451": 2},
				"test2": map[string]int32{"453": 2, "451": 2, "455": 0, "458": 0},
			},
			want3:   map[string]map[string]int32{},
			wantErr: false,
		},
		{
			name: "carry strategy",
			args: args{
				params: newGroupScheduleParams(
					"group", "453-2c0bc462f0e41079ffc6bc3307c1f0e4", "10%", "20%", 100,
					[]Shard{
						newShardStatus(t, "test1", 3, map[string]*VersionStatus{
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(2, 2),
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 2),
						}, 0, 0),
						newShardStatus(t, "test2", 3, map[string]*VersionStatus{
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(2, 2),
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 2),
						}, 0, 0),
					},
					nil,
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 87,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"453": 3, "451": 1, "455": 0, "458": 0},
				"test2": map[string]int32{"453": 3, "451": 1, "455": 0, "458": 0},
			},
			want3:   map[string]map[string]int32{},
			wantErr: false,
		},
		{
			name: "carry strategy",
			args: args{
				params: newGroupScheduleParams(
					"group", "453-2c0bc462f0e41079ffc6bc3307c1f0e4", "10%", "34%", 100,
					[]Shard{
						newShardStatus(t, "test1", 3, map[string]*VersionStatus{
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(1, 1),
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 1),
						}, 0, 0),
						newShardStatus(t, "test2", 2, map[string]*VersionStatus{
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(1, 1),
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 2),
						}, 0, 0),
					},
					nil,
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 68,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"453": 2, "451": 1},
				"test2": map[string]int32{"453": 2, "451": 1, "455": 0, "458": 0},
			},
			want3:   map[string]map[string]int32{},
			wantErr: false,
		},
		{
			name: "carry strategy",
			args: args{
				params: newGroupScheduleParams(
					"group", "453-2c0bc462f0e41079ffc6bc3307c1f0e4", "10%", "34%", 100,
					[]Shard{
						newShardStatus(t, "test1", 3, map[string]*VersionStatus{
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(1, 1),
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 1),
						}, 0, 0),
						newShardStatus(t, "test2", 3, map[string]*VersionStatus{
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(1, 1),
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 2),
						}, 0, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test1": newStrategy("34%"),
						"test2": newStrategy("34%"),
					},
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 68,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"453": 2, "451": 1},
				"test2": map[string]int32{"453": 2, "451": 1},
			},
			want3:   map[string]map[string]int32{},
			wantErr: false,
		},
		{
			name: "carry strategy",
			args: args{
				params: newGroupScheduleParams(
					"group", "453-2c0bc462f0e41079ffc6bc3307c1f0e4", "10%", "34%", 100,
					[]Shard{
						newShardStatus(t, "test1", 3, map[string]*VersionStatus{
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(1, 1),
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 2),
						}, 0, 0),
						newShardStatus(t, "test2", 3, map[string]*VersionStatus{
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(1, 1),
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 2),
						}, 0, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test1": newStrategy("34%"),
						"test2": newStrategy("34%"),
					},
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 100,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"453": 3, "451": 0},
				"test2": map[string]int32{"453": 3, "451": 0},
			},
			want3:   map[string]map[string]int32{},
			wantErr: false,
		},
		{
			name: "scale up more than double",
			args: args{
				params: newGroupScheduleParams(
					"group", "453-2c0bc462f0e41079ffc6bc3307c1f0e4", "10%", "20%", 100,
					[]Shard{
						newShardStatus(t, "test1", 32, map[string]*VersionStatus{
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(10, 10),
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(5, 0),
						}, 0, 0),
					},
					nil,
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 74,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"453": 22, "451": 10},
			},
			want3:   map[string]map[string]int32{},
			wantErr: false,
		},
		{
			name: "scale up ",
			args: args{
				params: newGroupScheduleParams(
					"group", "453-2c0bc462f0e41079ffc6bc3307c1f0e4", "10%", "20%", 100,
					[]Shard{
						newShardStatus(t, "test1", 16, map[string]*VersionStatus{
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(10, 10),
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(5, 3),
						}, 0, 0),
					},
					nil,
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 47,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"453": 6, "451": 10},
			},
			want3:   map[string]map[string]int32{},
			wantErr: false,
		},
		{
			name: "scale up more than double",
			args: args{
				params: newGroupScheduleParams(
					"group", "453-2c0bc462f0e41079ffc6bc3307c1f0e4", "10%", "20%", 100,
					[]Shard{
						newShardStatus(t, "test1", 32, map[string]*VersionStatus{
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(14, 14),
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 0),
						}, 0, 0),
					},
					nil,
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 74,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"453": 18, "451": 14},
			},
			want3:   map[string]map[string]int32{},
			wantErr: false,
		},
		{
			name: "4 and 25",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "10%", "25%", 100,
					[]Shard{
						newShardStatus(t, "test1", 4, map[string]*VersionStatus{
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(1, 1),
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 0),
							"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 0),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(0, 0),
						}, 0, 0),
						newShardStatus(t, "test2", 25, map[string]*VersionStatus{
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(7, 7),
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(12, 0),
							"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(6, 0),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(0, 0),
						}, 0, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test1": newStrategy("50%"),
					},
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 25,
			want2: map[string]map[string]int32{
				"test1": {"451": 1, "453": 2, "455": 0, "458": 1},
				"test2": {"451": 7, "453": 12, "455": 0, "458": 6},
			},
			want3:   map[string]map[string]int32{},
			wantErr: false,
		},
		{
			name: "4 and 25",
			args: args{
				rollingStrategy: "oldFirst",
				params: newGroupScheduleParams(
					"group", "459-dfasderw0628e0853754ae8a5a0a8c45", "10%", "20%", 100,
					[]Shard{
						newShardStatus(t, "test1", 4, map[string]*VersionStatus{
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(1, 1),
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
							"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 1),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(1, 1),
						}, 0, 0),
						newShardStatus(t, "test2", 25, map[string]*VersionStatus{
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(7, 7),
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(4, 4),
							"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(9, 9),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(6, 0),
						}, 0, 0),
						newShardStatus(t, "test3", 6, map[string]*VersionStatus{
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(2, 2),
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
							"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(2, 2),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(1, 0),
						}, 0, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test1": newStrategy("25%"),
					},
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
					"459-dfasderw0628e0853754ae8a5a0a8c45": "459",
				},
			},
			want1: 20,
			want2: map[string]map[string]int32{
				"test1": {"451": 1, "453": 1, "455": 1, "458": 0, "459": 1},
				"test2": {"451": 7, "453": 4, "455": 9, "458": 0, "459": 5},
				"test3": {"451": 2, "453": 1, "455": 2, "458": 0, "459": 1},
			},
			want3:   map[string]map[string]int32{},
			wantErr: false,
		},
		{
			name: "4 and 25",
			args: args{
				params: newGroupScheduleParams(
					"group", "459-dfasderw0628e0853754ae8a5a0a8c45", "10%", "25%", 100,
					[]Shard{
						newShardStatus(t, "test1", 4, map[string]*VersionStatus{
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(1, 1),
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
							"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 1),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(1, 1),
						}, 0, 0),
						newShardStatus(t, "test2", 25, map[string]*VersionStatus{
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(7, 7),
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(6, 6),
							"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(6, 0),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(6, 0),
						}, 0, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test1": newStrategy("50%"),
					},
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
					"459-dfasderw0628e0853754ae8a5a0a8c45": "459",
				},
			},
			want1: 25,
			want2: map[string]map[string]int32{
				"test1": {"451": 1, "453": 1, "455": 1, "458": 0, "459": 1},
				"test2": {"451": 7, "453": 6, "455": 6, "458": 0, "459": 6},
			},
			want3:   map[string]map[string]int32{},
			wantErr: false,
		},
		{
			name: "hold more than need",
			args: args{
				params: newGroupScheduleParams(
					"group", "453-2c0bc462f0e41079ffc6bc3307c1f0e4", "10%", "30%", 100,
					[]Shard{
						newShardStatus(t, "test1", 9, map[string]*VersionStatus{
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(7, 7),
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(3, 0),
						}, 0, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test1": newStrategy("30%"),
					},
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
					"459-dfasderw0628e0853754ae8a5a0a8c45": "459",
				},
			},
			want1: 30,
			want2: map[string]map[string]int32{
				"test1": {"451": 7, "453": 2},
			},
			want3:   map[string]map[string]int32{},
			wantErr: false,
		},
		{
			name: "2 nodes",
			args: args{
				params: newGroupScheduleParams(
					"group", "453-2c0bc462f0e41079ffc6bc3307c1f0e4", "10%", "50%", 100,
					[]Shard{
						newShardStatus(t, "test1", 2, map[string]*VersionStatus{
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(2, 2),
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(0, 0),
						}, 0, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test1": newStrategy("50%"),
					},
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
					"459-dfasderw0628e0853754ae8a5a0a8c45": "459",
				},
			},
			want1: 50,
			want2: map[string]map[string]int32{
				"test1": {"451": 1, "453": 1, "455": 0, "458": 0, "459": 0},
			},
			want3:   map[string]map[string]int32{},
			wantErr: false,
		},
		{
			name: "2 nodes",
			args: args{
				params: newGroupScheduleParams(
					"group", "453-2c0bc462f0e41079ffc6bc3307c1f0e4", "10%", "50%", 100,
					[]Shard{
						newShardStatus(t, "test1", 2, map[string]*VersionStatus{
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(1, 1),
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 0),
						}, 0, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test1": newStrategy("50%"),
					},
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
					"459-dfasderw0628e0853754ae8a5a0a8c45": "459",
				},
			},
			want1: 50,
			want2: map[string]map[string]int32{
				"test1": {"451": 1, "453": 1},
			},
			want3:   map[string]map[string]int32{},
			wantErr: false,
		},
		{
			name: "2 nodes",
			args: args{
				params: newGroupScheduleParams(
					"group", "455-761efa07c0f73213a3a8704a55e60330", "10%", "50%", 100,
					[]Shard{
						newShardStatus(t, "test1", 2, map[string]*VersionStatus{
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(1, 1),
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
						}, 0, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test1": newStrategy("50%"),
					},
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
					"459-dfasderw0628e0853754ae8a5a0a8c45": "459",
				},
			},
			want1: 50,
			want2: map[string]map[string]int32{
				"test1": {"451": 1, "455": 1, "453": 0, "458": 0, "459": 0},
			},
			want3:   map[string]map[string]int32{},
			wantErr: false,
		},
		{
			name: "3 nodes",
			args: args{
				params: newGroupScheduleParams(
					"group", "453-2c0bc462f0e41079ffc6bc3307c1f0e4", "10%", "34%", 100,
					[]Shard{
						newShardStatus(t, "test1", 3, map[string]*VersionStatus{
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(3, 3),
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(0, 0),
						}, 0, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test1": newStrategy("34%"),
					},
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
					"459-dfasderw0628e0853754ae8a5a0a8c45": "459",
				},
			},
			want1: 34,
			want2: map[string]map[string]int32{
				"test1": {"451": 2, "453": 1, "455": 0, "458": 0, "459": 0},
			},
			want3:   map[string]map[string]int32{},
			wantErr: false,
		},
		{
			name: "3 nodes",
			args: args{
				params: newGroupScheduleParams(
					"group", "453-2c0bc462f0e41079ffc6bc3307c1f0e4", "10%", "34%", 100,
					[]Shard{
						newShardStatus(t, "test1", 3, map[string]*VersionStatus{
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(2, 2),
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 0),
						}, 0, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test1": newStrategy("34%"),
					},
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
					"459-dfasderw0628e0853754ae8a5a0a8c45": "459",
				},
			},
			want1: 34,
			want2: map[string]map[string]int32{
				"test1": {"451": 2, "453": 1},
			},
			want3:   map[string]map[string]int32{},
			wantErr: false,
		},
		{
			name: "3 nodes",
			args: args{
				params: newGroupScheduleParams(
					"group", "453-2c0bc462f0e41079ffc6bc3307c1f0e4", "10%", "34%", 100,
					[]Shard{
						newShardStatus(t, "test1", 3, map[string]*VersionStatus{
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(2, 2),
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
						}, 0, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test1": newStrategy("34%"),
					},
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
					"459-dfasderw0628e0853754ae8a5a0a8c45": "459",
				},
			},
			want1: 68,
			want2: map[string]map[string]int32{
				"test1": {"451": 1, "453": 2, "455": 0, "458": 0, "459": 0},
			},
			want3:   map[string]map[string]int32{},
			wantErr: false,
		},
		{
			name: "3 nodes",
			args: args{
				params: newGroupScheduleParams(
					"group", "453-2c0bc462f0e41079ffc6bc3307c1f0e4", "10%", "34%", 100,
					[]Shard{
						newShardStatus(t, "test1", 3, map[string]*VersionStatus{
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(1, 1),
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 1),
						}, 0, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test1": newStrategy("34%"),
					},
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
					"459-dfasderw0628e0853754ae8a5a0a8c45": "459",
				},
			},
			want1: 68,
			want2: map[string]map[string]int32{
				"test1": {"451": 1, "453": 2},
			},
			want3:   map[string]map[string]int32{},
			wantErr: false,
		},
		{
			name: "3 nodes",
			args: args{
				params: newGroupScheduleParams(
					"group", "453-2c0bc462f0e41079ffc6bc3307c1f0e4", "10%", "34%", 100,
					[]Shard{
						newShardStatus(t, "test1", 3, map[string]*VersionStatus{
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(1, 1),
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 2),
						}, 0, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test1": newStrategy("34%"),
					},
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
					"459-dfasderw0628e0853754ae8a5a0a8c45": "459",
				},
			},
			want1: 100,
			want2: map[string]map[string]int32{
				"test1": {"451": 0, "453": 3, "455": 0, "458": 0, "459": 0},
			},
			want3:   map[string]map[string]int32{},
			wantErr: false,
		},
		{
			name: "oldFirst",
			args: args{
				rollingStrategy: "oldFirst",
				params: newGroupScheduleParams(
					"group", "459-dfasderw0628e0853754ae8a5a0a8c45", "10%", "20%", 100,
					[]Shard{
						newShardStatus(t, "test1", 40, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(20, 0),
							"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(12, 0),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(8, 0),
						}, 0, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test1": newStrategy("25%"),
					},
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
					"459-dfasderw0628e0853754ae8a5a0a8c45": "459",
				},
			},
			want1: 20,
			want2: map[string]map[string]int32{
				"test1": {"451": 0, "453": 20, "455": 12, "458": 0, "459": 8},
			},
			want3:   map[string]map[string]int32{},
			wantErr: false,
		},
		{
			name: "oldFirst",
			args: args{
				rollingStrategy: "oldFirst",
				params: newGroupScheduleParams(
					"group", "459-dfasderw0628e0853754ae8a5a0a8c45", "10%", "20%", 100,
					[]Shard{
						newShardStatus(t, "test1", 40, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(20, 20),
							"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(12, 12),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(8, 8),
						}, 0, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test1": newStrategy("25%"),
					},
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
					"459-dfasderw0628e0853754ae8a5a0a8c45": "459",
				},
			},
			want1: 20,
			want2: map[string]map[string]int32{
				"test1": {"451": 0, "453": 12, "455": 12, "458": 8, "459": 8},
			},
			want3:   map[string]map[string]int32{},
			wantErr: false,
		},
		{
			name: "oldFirst",
			args: args{
				rollingStrategy: "oldFirst",
				params: newGroupScheduleParams(
					"group", "459-dfasderw0628e0853754ae8a5a0a8c45", "10%", "20%", 100,
					[]Shard{
						newShardStatus(t, "test1", 40, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(20, 20),
							"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(12, 6),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(8, 8),
						}, 0, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test1": newStrategy("25%"),
					},
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
					"459-dfasderw0628e0853754ae8a5a0a8c45": "459",
				},
			},
			want1: 20,
			want2: map[string]map[string]int32{
				"test1": {"451": 0, "453": 18, "455": 6, "458": 8, "459": 8},
			},
			want3:   map[string]map[string]int32{},
			wantErr: false,
		},
		{
			name: "lessFirst",
			args: args{
				rollingStrategy: "lessFirst",
				params: newGroupScheduleParams(
					"group", "459-dfasderw0628e0853754ae8a5a0a8c45", "10%", "20%", 100,
					[]Shard{
						newShardStatus(t, "test1", 40, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(20, 20),
							"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(12, 6),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(8, 8),
						}, 0, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test1": newStrategy("25%"),
					},
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
					"459-dfasderw0628e0853754ae8a5a0a8c45": "459",
				},
			},
			want1: 20,
			want2: map[string]map[string]int32{
				"test1": {"451": 0, "453": 20, "455": 4, "458": 8, "459": 8},
			},
			want3:   map[string]map[string]int32{},
			wantErr: false,
		},
		{
			name: "4 and 12 rows",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "10%", "25%", 100,
					[]Shard{
						newShardStatus(t, "test1", 4, map[string]*VersionStatus{
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(1, 1),
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 2),
							"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 0),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(0, 0),
						}, 0, 0),
						newShardStatus(t, "test2", 12, map[string]*VersionStatus{
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(2, 2),
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(5, 5),
							"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(5, 2),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(0, 0),
						}, 0, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test1": newStrategy("34%"),
					},
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 25,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"451": 1, "453": 2, "455": 0, "458": 1},
				"test2": map[string]int32{"451": 2, "453": 5, "455": 2, "458": 3},
			},
			want3:   map[string]map[string]int32{},
			wantErr: false,
		},
		{
			name: "4 and 12 rows 1",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "10%", "25%", 100,
					[]Shard{
						newShardStatus(t, "test1", 4, map[string]*VersionStatus{
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(1, 1),
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 2),
							"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(0, 0),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(1, 0),
						}, 0, 0),
						newShardStatus(t, "test2", 12, map[string]*VersionStatus{
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(2, 2),
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(5, 5),
							"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 1),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(4, 0),
						}, 0, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test1": newStrategy("34%"),
					},
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 25,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"451": 1, "453": 2, "455": 0, "458": 1},
				"test2": map[string]int32{"451": 2, "453": 5, "455": 0, "458": 5},
			},
			want3:   map[string]map[string]int32{},
			wantErr: false,
		}, {
			name: "surge",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "10%", "20%", 100,
					[]Shard{
						newShardStatus(t, "test1", 5, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(2, 0),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(2, 2),
						}, 0, 0),
						newShardStatus(t, "test2", 8, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 2),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(3, 0),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(3, 3),
						}, 0, 0),
						newShardStatus(t, "test3", 2, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(1, 0),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(1, 1),
						}, 0, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test1": newStrategy("20%"),
						"test2": newStrategy("20%"),
						"test3": newStrategy("20%"),
					},
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 34,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"451": 2, "453": 1, "458": 2},
				"test2": map[string]int32{"451": 3, "453": 2, "458": 3},
				"test3": map[string]int32{"451": 1, "453": 1, "458": 1},
			},
			wantErr: false,
		},
		{
			name: "surge",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "10%", "20%", 100,
					[]Shard{
						newShardStatus(t, "test1", 5, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(2, 0),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(2, 2),
						}, 0, 0),
						newShardStatus(t, "test2", 8, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 2),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(3, 0),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(3, 3),
						}, 0, 0),
						newShardStatus(t, "test3", 2, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(1, 1),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(1, 1),
						}, 0, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test1": newStrategy("20%"),
						"test2": newStrategy("20%"),
						"test3": newStrategy("20%"),
					},
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 34,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"451": 2, "453": 1, "458": 2},
				"test2": map[string]int32{"451": 3, "453": 2, "458": 3},
				"test3": map[string]int32{"451": 1, "453": 1, "458": 1},
			},
			wantErr: false,
		},
		{
			name: "surge",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "10%", "20%", 100,
					[]Shard{
						newShardStatus(t, "test1", 5, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(2, 2),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(2, 2),
						}, 0, 0),
						newShardStatus(t, "test2", 8, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 2),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(3, 3),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(3, 3),
						}, 0, 0),
						newShardStatus(t, "test3", 2, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(1, 1),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(1, 1),
						}, 0, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test1": newStrategy("20%"),
						"test2": newStrategy("20%"),
						"test3": newStrategy("20%"),
					},
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 58,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"451": 2, "453": 1, "458": 2},
				"test2": map[string]int32{"451": 3, "453": 1, "458": 4},
				"test3": map[string]int32{"451": 1, "453": 0, "458": 2},
			},
			wantErr: false,
		},
		{
			name: "surge",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "10%", "20%", 100,
					[]Shard{
						newShardStatus(t, "test1", 5, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(2, 2),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(2, 2),
						}, 0, 0),
						newShardStatus(t, "test2", 8, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(4, 4),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(3, 3),
						}, 0, 0),
						newShardStatus(t, "test3", 2, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(0, 0),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(2, 2),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(1, 1),
						}, 0, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test1": newStrategy("20%"),
						"test2": newStrategy("20%"),
						"test3": newStrategy("20%"),
					},
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 60,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"451": 2, "453": 0, "458": 3},
				"test2": map[string]int32{"451": 3, "453": 0, "458": 5},
				"test3": map[string]int32{"451": 1, "453": 0, "458": 2},
			},
			wantErr: false,
		},
		{
			name: "surge",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "10%", "20%", 100,
					[]Shard{
						newShardStatus(t, "test1", 5, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(0, 0),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(3, 3),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(2, 2),
						}, 0, 0),
						newShardStatus(t, "test2", 8, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(0, 0),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(5, 5),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(3, 3),
						}, 0, 0),
						newShardStatus(t, "test3", 2, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(0, 0),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(2, 2),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(1, 1),
						}, 0, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test1": newStrategy("20%"),
						"test2": newStrategy("20%"),
						"test3": newStrategy("20%"),
					},
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 80,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"451": 1, "453": 0, "458": 4},
				"test2": map[string]int32{"451": 2, "453": 0, "458": 6},
				"test3": map[string]int32{"451": 1, "453": 0, "458": 2},
			},
			wantErr: false,
		},
		{
			name: "surge",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "10%", "20%", 100,
					[]Shard{
						newShardStatus(t, "test1", 5, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(0, 0),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(4, 4),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(1, 1),
						}, 0, 0),
						newShardStatus(t, "test2", 8, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(0, 0),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(6, 6),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(2, 2),
						}, 0, 0),
						newShardStatus(t, "test3", 2, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(0, 0),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(2, 2),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(1, 1),
						}, 0, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test1": newStrategy("20%"),
						"test2": newStrategy("20%"),
						"test3": newStrategy("20%"),
					},
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 95,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"451": 1, "453": 0, "458": 4},
				"test2": map[string]int32{"451": 1, "453": 0, "458": 7},
				"test3": map[string]int32{"451": 0, "453": 0, "458": 3},
			},
			wantErr: false,
		},
		{
			name: "surge",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "10%", "20%", 100,
					[]Shard{
						newShardStatus(t, "test1", 5, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(0, 0),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(4, 4),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(1, 1),
						}, 0, 0),
						newShardStatus(t, "test2", 8, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(0, 0),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(7, 7),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(1, 1),
						}, 0, 0),
						newShardStatus(t, "test3", 2, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(0, 0),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(3, 2),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(0, 0),
						}, 0, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test1": newStrategy("20%"),
						"test2": newStrategy("20%"),
						"test3": newStrategy("20%"),
					},
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 100,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"451": 0, "453": 0, "458": 5},
				"test2": map[string]int32{"451": 0, "453": 0, "458": 8},
				"test3": map[string]int32{"451": 0, "453": 0, "458": 3},
			},
			wantErr: false,
		},
		{
			name: "surge",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "10%", "20%", 100,
					[]Shard{
						newShardStatus(t, "test1", 5, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(0, 0),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(5, 5),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(0, 0),
						}, 0, 0),
						newShardStatus(t, "test2", 8, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(0, 0),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(8, 7),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(0, 0),
						}, 0, 0),
						newShardStatus(t, "test3", 2, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(0, 0),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(3, 2),
							"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(0, 0),
						}, 0, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test1": newStrategy("20%"),
						"test2": newStrategy("20%"),
						"test3": newStrategy("20%"),
					},
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
			},
			want1: 100,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"451": 0, "453": 0, "458": 5},
				"test2": map[string]int32{"451": 0, "453": 0, "458": 8},
				"test3": map[string]int32{"451": 0, "453": 0, "458": 3},
			},
			wantErr: false,
		},
		{
			name: "pingpang",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "10%", "34%", 100,
					[]Shard{
						newShardStatus(t, "test1", 3, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(2, 0),
						}, 0, 0),
						newShardStatus(t, "test2", 3, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 1),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(1, 0),
						}, 0, 0),
					},
					map[string]*app.DeploymentStrategy{
						"test1": newStrategy("34%"),
						"test2": newStrategy("34%"),
					},
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
				},
			},
			want1: 34,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"453": 2, "458": 1},
				"test2": map[string]int32{"453": 2, "458": 1},
			},
			wantErr: false,
		},
		{
			name: "pingpang",
			args: args{
				params: newGroupScheduleParams(
					"group", "458-95a70e220628e0853754ae8a5a0a8c45", "10%", "34%", 100,
					[]Shard{
						newShardStatus(t, "test1", 3, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(2, 0),
						}, 0, 67),
						newShardStatus(t, "test2", 3, map[string]*VersionStatus{
							"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 1),
							"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(1, 0),
						}, 0, 67),
					},
					map[string]*app.DeploymentStrategy{
						"test1": newStrategy("34%"),
						"test2": newStrategy("34%"),
					},
				),
				groupVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
				},
			},
			want1: 34,
			want2: map[string]map[string]int32{
				"test1": map[string]int32{"453": 1, "458": 2},
				"test2": map[string]int32{"453": 2, "458": 1},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.rollingStrategy != "" {
				defaultRollingStrategy = tt.args.rollingStrategy
				defer func() {
					defaultRollingStrategy = RollingStrategyLessFirst
				}()
			}
			if tt.args.disableMissColume {
				features.C2MutableFeatureGate.Set("EnableMissingColumn=false")
			} else {
				features.C2MutableFeatureGate.Set("EnableMissingColumn=true")
			}
			gss := &GroupSyncSlidingScheduler{}
			got, got1, err := gss.Schedule(tt.args.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("GroupSyncSlidingScheduler.Schedule() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if nil != err {
				return
			}
			fmt.Println(utils.ObjJSON(got))
			for _, shard := range tt.args.params.ShardStatus {
				params := got[shard.GetName()]
				params.GroupVersionToVersionMap = tt.args.groupVersionMap
				params.LatestVersion = tt.args.groupVersionMap[tt.args.params.ShardGroupVersion]
				s := &InPlaceShardScheduler{
					debug: true,
				}
				params.initScheduleArgs(shard)
				gotShard, gotDependency, _ := s.Schedule(shard, &params)
				wantShard := tt.want2[shard.GetName()]
				if !reflect.DeepEqual(gotShard, wantShard) {
					t.Errorf("shard %s InPlaceShardScheduler.Schedule() got = %v, want %v", shard.GetName(), gotShard, wantShard)
				}
				if nil != tt.want3 {
					wantDependency := tt.want3[shard.GetName()]
					if !reflect.DeepEqual(gotDependency, wantDependency) {
						t.Errorf("shard %s InPlaceShardScheduler.Schedule() got = %v, want %v", shard.GetName(), gotDependency, wantDependency)
					}
				}
			}
			if got1 != tt.want1 {
				t.Errorf("GroupSyncSlidingScheduler.Schedule() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
