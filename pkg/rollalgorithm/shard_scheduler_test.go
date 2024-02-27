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
	"reflect"
	"testing"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	app "k8s.io/api/apps/v1"
)

func TestInPlaceShardScheduler_Schedule(t *testing.T) {
	type args struct {
		shard  Shard
		params *ShardScheduleParams
	}

	fCreateParams := func(maxUnavaliable, maxSurge, latestVersion string, latestVersionRatio int32, groupVersionMap map[string]string,
		replicas int32, versionHoldMatrix map[string]int32, versionHoldMatrixPercent map[string]float64, paddedLatestVersionRatio int32, strategy CarryStrategyType,
	) *ShardScheduleParams {
		var params = &ShardScheduleParams{
			Plan: SchedulePlan{
				Replicas: &replicas,
				Strategy: app.DeploymentStrategy{
					Type: app.RollingUpdateDeploymentStrategyType,
					RollingUpdate: &app.RollingUpdateDeployment{
						MaxUnavailable: getIntstrPtr(maxUnavaliable),
						MaxSurge:       getIntstrPtr(maxSurge),
					},
				},
				PaddedLatestVersionRatio:   &paddedLatestVersionRatio,
				VersionHoldMatrix:          versionHoldMatrix,
				VersionHoldMatrixPercent:   versionHoldMatrixPercent,
				LatestVersionRatio:         utils.Int32Ptr(latestVersionRatio),
				LatestVersionCarryStrategy: strategy,
			},
			ShardRuntimeArgs: ShardRuntimeArgs{
				LatestVersion:            latestVersion,
				GroupVersionToVersionMap: groupVersionMap,
			},
		}
		return params
	}

	fCreateArgs := func(t *testing.T, name string, replicas int32, versionReplicas map[string]*VersionStatus,
		maxUnavaliable, maxSurge, latestVersion string, latestVersionRatio int32, groupVersionMap map[string]string,
		versionHoldMatrix map[string]int32, versionHoldMatrixPercent map[string]float64, paddedLatestVersionRatio int32, strategy CarryStrategyType,
	) args {
		return args{shard: newShardStatus(t, name, replicas, versionReplicas, 0, 0),
			params: fCreateParams(maxUnavaliable, maxSurge, latestVersion, latestVersionRatio, groupVersionMap, replicas, versionHoldMatrix, versionHoldMatrixPercent, paddedLatestVersionRatio, strategy)}
	}

	tests := []struct {
		name    string
		args    args
		want    map[string]int32
		want1   map[string]int32
		wantErr bool
	}{

		{
			name: "pingpang",
			args: fCreateArgs(t, "test1", 6, map[string]*VersionStatus{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(3, 3),
				"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 1),
				"457-aa1efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 0),
				"458-bb1efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 0),
			}, "25%", "10%", "458", 25, map[string]string{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
				"455-761efa07c0f73213a3a8704a55e60330": "455",
				"457-aa1efa07c0f73213a3a8704a55e60330": "457",
				"458-bb1efa07c0f73213a3a8704a55e60330": "458",
			}, map[string]int32{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": 3,
				"455-761efa07c0f73213a3a8704a55e60330": 1,
				"457-aa1efa07c0f73213a3a8704a55e60330": 0,
				"458-bb1efa07c0f73213a3a8704a55e60330": 1,
			}, map[string]float64{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": 50,
				"455-761efa07c0f73213a3a8704a55e60330": 16,
				"457-aa1efa07c0f73213a3a8704a55e60330": 0,
				"458-bb1efa07c0f73213a3a8704a55e60330": 25,
			}, 34, FloorCarryStrategyType),
			wantErr: false,
			want: map[string]int32{
				"453": 3,
				"455": 1,
				"457": 0,
				"458": 2,
			},
		},
		{
			name: "pingpang",
			args: fCreateArgs(t, "test1", 6, map[string]*VersionStatus{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(3, 3),
				"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 1),
				"458-bb1efa07c0f73213a3a8704a55e60330": newVersionStatus(2, 0),
			}, "25%", "10%", "458", 25, map[string]string{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
				"455-761efa07c0f73213a3a8704a55e60330": "455",
				"458-bb1efa07c0f73213a3a8704a55e60330": "458",
			}, map[string]int32{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": 3,
				"455-761efa07c0f73213a3a8704a55e60330": 1,
				"457-aa1efa07c0f73213a3a8704a55e60330": 0,
				"458-bb1efa07c0f73213a3a8704a55e60330": 1,
			}, map[string]float64{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": 50,
				"455-761efa07c0f73213a3a8704a55e60330": 16,
				"457-aa1efa07c0f73213a3a8704a55e60330": 0,
				"458-bb1efa07c0f73213a3a8704a55e60330": 25,
			}, 34, FloorCarryStrategyType),
			wantErr: false,
			want: map[string]int32{
				"453": 3,
				"455": 1,
				"458": 2,
			},
		},

		{
			name: "not perfect available",
			args: fCreateArgs(t, "test1", 3, map[string]*VersionStatus{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(3, 3),
				"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 1),
			}, "20%", "10%", "455", 50, map[string]string{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
				"455-761efa07c0f73213a3a8704a55e60330": "455",
			}, nil, nil, 0, CeilCarryStrategyType),
			wantErr: false,
			want: map[string]int32{
				"455": 2,
				"453": 2,
			},
		},
		{
			name: "not perfect available",
			args: fCreateArgs(t, "test1", 3, map[string]*VersionStatus{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 2),
				"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(2, 2),
			}, "20%", "10%", "455", 89, map[string]string{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
				"455-761efa07c0f73213a3a8704a55e60330": "455",
			}, nil, nil, 0, CeilCarryStrategyType),
			wantErr: false,
			want: map[string]int32{
				"455": 3,
				"453": 1,
			},
		},
		{
			name: "not perfect available",
			args: fCreateArgs(t, "test1", 3, map[string]*VersionStatus{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
				"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(3, 3),
			}, "20%", "10%", "455", 100, map[string]string{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
				"455-761efa07c0f73213a3a8704a55e60330": "455",
			}, nil, nil, 0, CeilCarryStrategyType),
			wantErr: false,
			want: map[string]int32{
				"455": 4,
				"453": 0,
			},
		},
		{
			name: "not perfect available",
			args: fCreateArgs(t, "test1", 3, map[string]*VersionStatus{
				"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(4, 4),
			}, "20%", "10%", "455", 100, map[string]string{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
				"455-761efa07c0f73213a3a8704a55e60330": "455",
			}, nil, nil, 0, CeilCarryStrategyType),
			wantErr: false,
			want: map[string]int32{
				"455": 3,
			},
		},

		{
			name: "not perfect available",
			args: fCreateArgs(t, "test1", 3, map[string]*VersionStatus{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(3, 3),
				"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(0, 0),
			}, "20%", "10%", "455", 59, map[string]string{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
				"455-761efa07c0f73213a3a8704a55e60330": "455",
			}, nil, nil, 0, CeilCarryStrategyType),
			wantErr: false,
			want: map[string]int32{
				"455": 1,
				"453": 3,
			},
		},
		{
			name: "not perfect available",
			args: fCreateArgs(t, "test1", 3, map[string]*VersionStatus{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 2),
				"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 1),
			}, "20%", "10%", "455", 59, map[string]string{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
				"455-761efa07c0f73213a3a8704a55e60330": "455",
			}, nil, nil, 0, CeilCarryStrategyType),
			wantErr: false,
			want: map[string]int32{
				"455": 2,
				"453": 2,
			},
		},
		{
			name: "nil",
			args: args{
				shard:  nil,
				params: nil,
			},
			wantErr: true,
		},
		{
			name: "one version scale",
			args: fCreateArgs(t, "test1", 3, map[string]*VersionStatus{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(4, 4),
			}, "50%", "30%", "453", 50, map[string]string{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
			}, nil, nil, 0, CeilCarryStrategyType),
			wantErr: false,
			want: map[string]int32{
				"453": 3,
			},
		},
		{
			name: "one version scale",
			args: fCreateArgs(t, "test1", 3, map[string]*VersionStatus{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(4, 0),
			}, "50%", "30%", "453", 50, map[string]string{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
			}, nil, nil, 0, CeilCarryStrategyType),
			wantErr: false,
			want: map[string]int32{
				"453": 3,
			},
		},
		{
			name: "one version scale",
			args: fCreateArgs(t, "test1", 10, map[string]*VersionStatus{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(4, 4),
			}, "50%", "30%", "453", 50, map[string]string{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
			}, nil, nil, 0, CeilCarryStrategyType),
			wantErr: false,
			want: map[string]int32{
				"453": 10,
			},
		},
		{
			name: "one version scale",
			args: fCreateArgs(t, "test1", 0, map[string]*VersionStatus{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(4, 4),
			}, "50%", "30%", "453", 50, map[string]string{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
			}, nil, nil, 0, CeilCarryStrategyType),
			wantErr: false,
			want: map[string]int32{
				"453": 0,
			},
		},
		{
			name: "normal",
			args: fCreateArgs(t, "test1", 5, map[string]*VersionStatus{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(4, 4),
				"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 1),
			}, "50%", "30%", "455", 50, map[string]string{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
				"455-761efa07c0f73213a3a8704a55e60330": "455",
			}, nil, nil, 0, CeilCarryStrategyType),
			wantErr: false,
			want: map[string]int32{
				"455": 3,
				"453": 2,
			},
		},
		{
			name: "normal",
			args: fCreateArgs(t, "test1", 5, map[string]*VersionStatus{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(5, 5),
			}, "100%", "30%", "455", 100, map[string]string{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
			}, nil, nil, 0, CeilCarryStrategyType),
			wantErr: false,
			want: map[string]int32{
				"455": 5,
				"453": 0,
			},
		},
		{
			name: "no change",
			args: fCreateArgs(t, "test1", 5, map[string]*VersionStatus{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(4, 4),
				"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 1),
			}, "50%", "30%", "455", 20, map[string]string{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
				"455-761efa07c0f73213a3a8704a55e60330": "455",
			}, nil, nil, 0, CeilCarryStrategyType),
			wantErr: false,
			want: map[string]int32{
				"455": 1,
				"453": 4,
			},
		},
		{
			name: "block by unavailable",
			args: fCreateArgs(t, "test1", 5, map[string]*VersionStatus{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(4, 4),
				"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 1),
			}, "30%", "30%", "455", 50, map[string]string{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
				"455-761efa07c0f73213a3a8704a55e60330": "455",
			}, nil, nil, 0, CeilCarryStrategyType),
			wantErr: false,
			want: map[string]int32{
				"455": 2,
				"453": 3,
			},
		},
		{
			name: "block by unavailable",
			args: fCreateArgs(t, "test1", 5, map[string]*VersionStatus{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(4, 4),
				"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 0),
			}, "30%", "30%", "455", 50, map[string]string{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
				"455-761efa07c0f73213a3a8704a55e60330": "455",
			}, nil, nil, 0, CeilCarryStrategyType),
			wantErr: false,
			want: map[string]int32{
				"455": 1,
				"453": 4,
			},
		},
		{
			name: "block by latestversion",
			args: fCreateArgs(t, "test1", 5, map[string]*VersionStatus{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(4, 4),
				"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 1),
			}, "30%", "30%", "455", 20, map[string]string{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
				"455-761efa07c0f73213a3a8704a55e60330": "455",
			}, nil, nil, 0, CeilCarryStrategyType),
			wantErr: false,
			want: map[string]int32{
				"455": 1,
				"453": 4,
			},
		},
		{
			name: "block by latestversion ceil",
			args: fCreateArgs(t, "test1", 7, map[string]*VersionStatus{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(6, 6),
				"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 1),
			}, "70%", "30%", "455", 30, map[string]string{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
				"455-761efa07c0f73213a3a8704a55e60330": "455",
			}, nil, nil, 0, CeilCarryStrategyType),
			wantErr: false,
			want: map[string]int32{
				"455": 3,
				"453": 4,
			},
		},
		{
			name: "roll back",
			args: fCreateArgs(t, "test1", 7, map[string]*VersionStatus{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(3, 3),
				"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(4, 4),
			}, "70%", "30%", "455", 30, map[string]string{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
				"455-761efa07c0f73213a3a8704a55e60330": "455",
			}, nil, nil, 0, CeilCarryStrategyType),
			wantErr: false,
			want: map[string]int32{
				"455": 3,
				"453": 4,
			},
		},
		{
			name: "roll back block",
			args: fCreateArgs(t, "test1", 7, map[string]*VersionStatus{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(3, 1),
				"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(4, 4),
			}, "30%", "30%", "455", 0, map[string]string{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
				"455-761efa07c0f73213a3a8704a55e60330": "455",
			}, nil, nil, 0, CeilCarryStrategyType),
			wantErr: false,
			want: map[string]int32{
				"455": 4,
				"453": 3,
			},
		},
		{
			name: "roll back block by latestversion",
			args: fCreateArgs(t, "test1", 7, map[string]*VersionStatus{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(3, 3),
				"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(4, 4),
			}, "100%", "30%", "455", 20, map[string]string{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
				"455-761efa07c0f73213a3a8704a55e60330": "455",
			}, nil, nil, 0, CeilCarryStrategyType),
			wantErr: false,
			want: map[string]int32{
				"455": 2,
				"453": 5,
			},
		},
		{
			name: "scale out",
			args: fCreateArgs(t, "test1", 17, map[string]*VersionStatus{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(4, 4),
				"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 1),
			}, "50%", "30%", "455", 20, map[string]string{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
				"455-761efa07c0f73213a3a8704a55e60330": "455",
			}, nil, nil, 0, CeilCarryStrategyType),
			wantErr: false,
			want: map[string]int32{
				"455": 4,
				"453": 13,
			},
		},
		{
			name: "scale out and role",
			args: fCreateArgs(t, "test1", 20, map[string]*VersionStatus{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(4, 4),
				"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 1),
			}, "50%", "30%", "455", 50, map[string]string{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
				"455-761efa07c0f73213a3a8704a55e60330": "455",
			}, nil, nil, 0, CeilCarryStrategyType),
			wantErr: false,
			want: map[string]int32{
				"455": 10,
				"453": 10,
			},
		},
		{
			name: "scale in and role",
			args: fCreateArgs(t, "test1", 3, map[string]*VersionStatus{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(7, 7),
				"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(3, 0),
			}, "50%", "30%", "455", 100, map[string]string{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
				"455-761efa07c0f73213a3a8704a55e60330": "455",
			}, nil, nil, 0, CeilCarryStrategyType),
			wantErr: false,
			want: map[string]int32{
				"455": 3,
				"453": 2,
			},
		},
		{
			name: "scale in and role",
			args: fCreateArgs(t, "test1", 3, map[string]*VersionStatus{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 2),
				"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(3, 0),
			}, "50%", "30%", "455", 100, map[string]string{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
				"455-761efa07c0f73213a3a8704a55e60330": "455",
			}, nil, nil, 0, CeilCarryStrategyType),
			wantErr: false,
			want: map[string]int32{
				"455": 3,
				"453": 2,
			},
		},
		{
			name: "scale in and role",
			args: fCreateArgs(t, "test1", 3, map[string]*VersionStatus{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 0),
				"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(3, 0),
			}, "50%", "30%", "455", 100, map[string]string{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
				"455-761efa07c0f73213a3a8704a55e60330": "455",
			}, nil, nil, 0, CeilCarryStrategyType),
			wantErr: false,
			want: map[string]int32{
				"455": 3,
				"453": 1,
			},
		},
		{
			name: "scale in and role",
			args: fCreateArgs(t, "test1", 3, map[string]*VersionStatus{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 0),
				"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(3, 0),
			}, "100%", "30%", "455", 100, map[string]string{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
				"455-761efa07c0f73213a3a8704a55e60330": "455",
			}, nil, nil, 0, CeilCarryStrategyType),
			wantErr: false,
			want: map[string]int32{
				"455": 3,
				"453": 0,
			},
		},
		{
			name: "scale in and role",
			args: fCreateArgs(t, "test1", 3, map[string]*VersionStatus{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 0),
				"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(3, 0),
			}, "50%", "30%", "455", 100, map[string]string{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
				"455-761efa07c0f73213a3a8704a55e60330": "455",
			}, nil, nil, 0, CeilCarryStrategyType),
			wantErr: false,
			want: map[string]int32{
				"455": 3,
				"453": 1,
			},
		},
		{
			name: "scale in and role",
			args: fCreateArgs(t, "test1", 0, map[string]*VersionStatus{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 0),
				"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(3, 0),
			}, "50%", "30%", "455", 100, map[string]string{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
				"455-761efa07c0f73213a3a8704a55e60330": "455",
			}, nil, nil, 0, CeilCarryStrategyType),
			wantErr: false,
			want: map[string]int32{
				"455": 0,
				"453": 0,
			},
		},
		{
			name: "scale in",
			args: fCreateArgs(t, "test1", 3, map[string]*VersionStatus{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(4, 4),
				"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 1),
			}, "50%", "30%", "455", 20, map[string]string{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
				"455-761efa07c0f73213a3a8704a55e60330": "455",
			}, nil, nil, 0, CeilCarryStrategyType),
			wantErr: false,
			want: map[string]int32{
				"455": 1,
				"453": 2,
			},
		},
		{
			name: "scale in",
			args: fCreateArgs(t, "test1", 3, map[string]*VersionStatus{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(4, 0),
				"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 0),
			}, "50%", "30%", "455", 20, map[string]string{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
				"455-761efa07c0f73213a3a8704a55e60330": "455",
			}, nil, nil, 0, CeilCarryStrategyType),
			wantErr: false,
			want: map[string]int32{
				"455": 1,
				"453": 2,
			},
		},
		{
			name: "role with 3 versions",
			args: fCreateArgs(t, "test1", 7, map[string]*VersionStatus{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(4, 4),
				"454-761efa07c0f73213a3a8704a55e60330": newVersionStatus(2, 2),
				"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 1),
			}, "50%", "30%", "455", 40, map[string]string{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
				"454-761efa07c0f73213a3a8704a55e60330": "454",
				"455-761efa07c0f73213a3a8704a55e60330": "455",
			}, nil, nil, 0, CeilCarryStrategyType),
			wantErr: false,
			want: map[string]int32{
				"455": 3,
				"454": 0,
				"453": 4,
			},
		},
		{
			name: "role with 3 versions",
			args: fCreateArgs(t, "test1", 7, map[string]*VersionStatus{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(4, 2),
				"454-761efa07c0f73213a3a8704a55e60330": newVersionStatus(2, 2),
				"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 1),
			}, "30%", "30%", "455", 40, map[string]string{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
				"454-761efa07c0f73213a3a8704a55e60330": "454",
				"455-761efa07c0f73213a3a8704a55e60330": "455",
			}, nil, nil, 0, CeilCarryStrategyType),
			wantErr: false,
			want: map[string]int32{
				"455": 3,
				"454": 2,
				"453": 2,
			},
		},
		{
			name: "role with 3 versions",
			args: fCreateArgs(t, "test1", 7, map[string]*VersionStatus{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(4, 3),
				"454-761efa07c0f73213a3a8704a55e60330": newVersionStatus(2, 2),
				"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 1),
			}, "30%", "30%", "455", 40, map[string]string{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
				"454-761efa07c0f73213a3a8704a55e60330": "454",
				"455-761efa07c0f73213a3a8704a55e60330": "455",
			}, nil, nil, 0, CeilCarryStrategyType),
			wantErr: false,
			want: map[string]int32{
				"455": 3,
				"454": 1,
				"453": 3,
			},
		},
		{
			name: "role with 3 versions and scale in",
			args: fCreateArgs(t, "test1", 3, map[string]*VersionStatus{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(4, 3),
				"454-761efa07c0f73213a3a8704a55e60330": newVersionStatus(2, 2),
				"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 1),
			}, "50%", "30%", "455", 40, map[string]string{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
				"454-761efa07c0f73213a3a8704a55e60330": "454",
				"455-761efa07c0f73213a3a8704a55e60330": "455",
			}, nil, nil, 0, CeilCarryStrategyType),
			wantErr: false,
			want: map[string]int32{
				"455": 2,
				"454": 0,
				"453": 1,
			},
		},
		{
			name: "role with 3 versions and scale out",
			args: fCreateArgs(t, "test1", 13, map[string]*VersionStatus{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(4, 3),
				"454-761efa07c0f73213a3a8704a55e60330": newVersionStatus(2, 2),
				"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 1),
			}, "50%", "30%", "455", 40, map[string]string{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
				"454-761efa07c0f73213a3a8704a55e60330": "454",
				"455-761efa07c0f73213a3a8704a55e60330": "455",
			}, nil, nil, 0, CeilCarryStrategyType),
			wantErr: false,
			want: map[string]int32{
				"455": 6,
				"454": 2,
				"453": 5,
			},
		},
		{
			name: "roll by surge",
			args: fCreateArgs(t, "test1", 2, map[string]*VersionStatus{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
				"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 1),
			}, "30%", "30%", "457", 50, map[string]string{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
				"455-761efa07c0f73213a3a8704a55e60330": "455",
			}, nil, nil, 0, CeilCarryStrategyType),
			wantErr: false,
			want: map[string]int32{
				"457": 1,
				"455": 1,
				"453": 1,
			},
		},
		{
			name: "block by unavailable",
			args: fCreateArgs(t, "test2", 2, map[string]*VersionStatus{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
				"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 1),
			}, "30%", "0%", "457", 50, map[string]string{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
				"455-761efa07c0f73213a3a8704a55e60330": "455",
			}, nil, nil, 0, CeilCarryStrategyType),
			wantErr: false,
			want: map[string]int32{
				"457": 1,
				"455": 1,
				"453": 1,
			},
		},
		{
			name: "scale",
			args: fCreateArgs(t, "test2", 2, map[string]*VersionStatus{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 0),
			}, "30%", "0%", "457", 80, map[string]string{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
				"455-761efa07c0f73213a3a8704a55e60330": "455",
			}, nil, nil, 0, CeilCarryStrategyType),
			wantErr: false,
			want: map[string]int32{
				"457": 2,
				"453": 1,
			},
		},
		{
			name: "max unavailable 0",
			args: fCreateArgs(t, "test1", 5, map[string]*VersionStatus{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
				"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(5, 5),
			}, "0%", "10%", "455", 100, map[string]string{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
				"455-761efa07c0f73213a3a8704a55e60330": "455",
			}, nil, nil, 0, CeilCarryStrategyType),
			wantErr: false,
			want: map[string]int32{
				"455": 6,
				"453": 0,
			},
		},
		{
			name: "scale out",
			args: fCreateArgs(t, "test1", 947, map[string]*VersionStatus{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(381, 341),
				"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(315, 67),
			}, "20%", "10%", "455", 100, map[string]string{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
				"455-761efa07c0f73213a3a8704a55e60330": "455",
			}, nil, nil, 0, CeilCarryStrategyType),
			wantErr: false,
			want: map[string]int32{
				"455": 507,
				"453": 440,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &InPlaceShardScheduler{
				debug: true,
			}
			tt.args.params.initScheduleArgs(tt.args.shard)
			got, _, err := s.Schedule(tt.args.shard, tt.args.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("InPlaceShardScheduler.Schedule() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("InPlaceShardScheduler.Schedule() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInPlaceShardScheduler_calaulateOversize(t *testing.T) {
	type args struct {
		params *ShardScheduleParams
	}
	tests := []struct {
		name  string
		s     *InPlaceShardScheduler
		args  args
		want  int32
		want1 int32
	}{
		{
			args: args{
				params: &ShardScheduleParams{
					ShardInternalArgs: ShardInternalArgs{
						CurOldReplicas:       100,
						targetOldReplicas:    33,
						CurLatestReplicas:    19,
						targetLatestReplicas: 6,
					},
				},
			},
			want:  13,
			want1: 67,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &InPlaceShardScheduler{}
			got, got1 := s.calaulateOversize(tt.args.params)
			if got != tt.want {
				t.Errorf("InPlaceShardScheduler.calaulateOversize() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("InPlaceShardScheduler.calaulateOversize() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestInPlaceShardScheduler_calcTarget(t *testing.T) {
	type args struct {
		args *ShardScheduleParams
	}
	tests := []struct {
		name  string
		args  args
		want  int32
		want1 int32
	}{
		{
			name: "backward",
			args: args{
				args: &ShardScheduleParams{
					ShardInternalArgs: ShardInternalArgs{
						Desired:     7,
						maxReplicas: 7,
						VersionCountMap: map[string]int32{
							"a": 1,
							"b": 6,
						},
						VersionReadyCountMap: map[string]int32{
							"a": 1,
							"b": 6,
						},
						CurTotalReplicas:   7,
						latestVersionRatio: 0,
						maxUnavailable:     1,
						VersionSortKeys:    []string{"a", "b"},
					},
					ShardRuntimeArgs: ShardRuntimeArgs{
						LatestVersion: "a",
					},
				},
			},
			want:  0,
			want1: 7,
		},
		{
			name: "forward",
			args: args{
				args: &ShardScheduleParams{
					ShardInternalArgs: ShardInternalArgs{
						Desired:     7,
						maxReplicas: 7,
						VersionCountMap: map[string]int32{
							"a": 1,
							"b": 6,
						},
						VersionReadyCountMap: map[string]int32{
							"a": 1,
							"b": 6,
						},
						CurTotalReplicas:   7,
						latestVersionRatio: 60,
						maxUnavailable:     1,
						VersionSortKeys:    []string{"a", "b"},
					},
					ShardRuntimeArgs: ShardRuntimeArgs{
						LatestVersion: "a",
					},
				},
			},
			want:  2,
			want1: 5,
		},

		{
			name: "block",
			args: args{
				args: &ShardScheduleParams{
					ShardInternalArgs: ShardInternalArgs{
						Desired:     7,
						maxReplicas: 7,
						VersionCountMap: map[string]int32{
							"a": 4,
							"b": 3,
						},
						VersionReadyCountMap: map[string]int32{
							"a": 1,
							"b": 3,
						},
						CurTotalReplicas:   7,
						latestVersionRatio: 60,
						maxUnavailable:     1,
						VersionSortKeys:    []string{"a", "b"},
					},
					ShardRuntimeArgs: ShardRuntimeArgs{
						LatestVersion: "a",
					},
				},
			},
			want:  4,
			want1: 3,
		},
		{
			name: "block",
			args: args{
				args: &ShardScheduleParams{
					ShardInternalArgs: ShardInternalArgs{
						Desired:     7,
						maxReplicas: 7,
						VersionCountMap: map[string]int32{
							"a": 3,
							"b": 4,
						},
						VersionReadyCountMap: map[string]int32{
							"a": 3,
							"b": 1,
						},
						CurTotalReplicas:   7,
						latestVersionRatio: 40,
						maxUnavailable:     1,
						VersionSortKeys:    []string{"a", "b"},
					},
					ShardRuntimeArgs: ShardRuntimeArgs{
						LatestVersion: "a",
					},
				},
			},
			want:  3,
			want1: 4,
		},
		{
			name: "target block",
			args: args{
				args: &ShardScheduleParams{
					ShardInternalArgs: ShardInternalArgs{
						Desired:     7,
						maxReplicas: 7,
						VersionCountMap: map[string]int32{
							"a": 1,
							"b": 6,
						},
						VersionReadyCountMap: map[string]int32{
							"a": 1,
							"b": 6,
						},
						CurTotalReplicas:   7,
						latestVersionRatio: 20,
						maxUnavailable:     5,
						VersionSortKeys:    []string{"a", "b"},
					},
					ShardRuntimeArgs: ShardRuntimeArgs{
						LatestVersion: "a",
					},
				},
			},
			want:  2,
			want1: 5,
		},

		{
			name: "target block",
			args: args{
				args: &ShardScheduleParams{
					ShardInternalArgs: ShardInternalArgs{
						Desired:     7,
						maxReplicas: 7,
						VersionCountMap: map[string]int32{
							"a": 6,
							"b": 1,
						},
						VersionReadyCountMap: map[string]int32{
							"a": 6,
							"b": 1,
						},
						CurTotalReplicas:   7,
						latestVersionRatio: 70,
						maxUnavailable:     5,
						VersionSortKeys:    []string{"a", "b"},
					},
					ShardRuntimeArgs: ShardRuntimeArgs{
						LatestVersion: "a",
					},
				},
			},
			want:  5,
			want1: 2,
		},

		{
			name: "one version",
			args: args{
				args: &ShardScheduleParams{
					ShardInternalArgs: ShardInternalArgs{
						Desired:     7,
						maxReplicas: 7,
						VersionCountMap: map[string]int32{
							"a": 1,
							"b": 6,
						},
						VersionReadyCountMap: map[string]int32{
							"a": 1,
							"b": 6,
						},
						CurTotalReplicas:   7,
						latestVersionRatio: 60,
						maxUnavailable:     1,
						VersionSortKeys:    []string{"a"},
					},
					ShardRuntimeArgs: ShardRuntimeArgs{
						LatestVersion: "a",
					},
				},
			},
			want:  7,
			want1: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &InPlaceShardScheduler{}
			s.calcTarget(tt.args.args)
			if tt.args.args.targetLatestReplicas != tt.want || tt.args.args.targetOldReplicas != tt.want1 {
				t.Errorf("calcTarget() = %v,%v, want %v,%v ", tt.args.args.targetLatestReplicas,
					tt.args.args.targetOldReplicas, tt.want, tt.want1)
			}
		})
	}
}

func TestInPlaceShardScheduler_filterVersionHoldingMap(t *testing.T) {
	type args struct {
		args              *ShardScheduleParams
		versionHoldingMap map[string]int32
	}
	tests := []struct {
		name string
		s    *InPlaceShardScheduler
		args args
		want map[string]int32
	}{
		{
			name: "test1",
			args: args{
				args: &ShardScheduleParams{
					ShardInternalArgs: ShardInternalArgs{
						minAvailable:         10,
						targetLatestReplicas: 3,
						VersionReadyCountMap: map[string]int32{
							"latest-version": 6,
							"version1":       2, "version2": 2, "version3": 0},
						VersionSortKeys: []string{"latest-version", "version1", "version2", "version3"},
					},
					ShardRuntimeArgs: ShardRuntimeArgs{
						LatestVersion: "latest-version",
					},
				},
				versionHoldingMap: map[string]int32{
					"latest-version": 6,
					"version1":       2, "version2": 2, "version3": 0,
				},
			},
			want: map[string]int32{
				"latest-version": 6,
				"version1":       2, "version2": 2, "version3": 0},
		},
		{
			name: "test2",
			args: args{
				args: &ShardScheduleParams{
					ShardInternalArgs: ShardInternalArgs{
						minAvailable:         6,
						targetLatestReplicas: 3,
						VersionReadyCountMap: map[string]int32{
							"latest-version": 6,
							"version1":       2, "version2": 1, "version3": 0},
						VersionCountMap: map[string]int32{
							"latest-version": 6,
							"version1":       2, "version2": 1, "version3": 0},
						VersionSortKeys: []string{"latest-version", "version1", "version2", "version3"},
					},
					ShardRuntimeArgs: ShardRuntimeArgs{
						LatestVersion: "latest-version",
					},
				},
				versionHoldingMap: map[string]int32{
					"latest-version": 2,
					"version1":       2, "version2": 2, "version3": 1,
				},
			},
			want: map[string]int32{
				"latest-version": 2,
				"version1":       2, "version2": 1, "version3": 0},
		},
		{
			name: "test3",
			args: args{
				args: &ShardScheduleParams{
					ShardInternalArgs: ShardInternalArgs{
						minAvailable:         6,
						targetLatestReplicas: 3,
						VersionReadyCountMap: map[string]int32{
							"latest-version": 6,
							"version1":       2, "version2": 1},

						VersionCountMap: map[string]int32{
							"latest-version": 6,
							"version1":       2, "version2": 1},
						VersionSortKeys: []string{"latest-version", "version1", "version2", "version3"},
					},
					ShardRuntimeArgs: ShardRuntimeArgs{
						LatestVersion: "latest-version",
					},
				},
				versionHoldingMap: map[string]int32{
					"latest-version": 2,
					"version1":       2, "version2": 2, "version3": 1,
				},
			},
			want: map[string]int32{
				"latest-version": 2,
				"version1":       2, "version2": 1, "version3": 0},
		},

		{
			name: "test4",
			args: args{
				args: &ShardScheduleParams{
					ShardInternalArgs: ShardInternalArgs{
						minAvailable:         10,
						targetLatestReplicas: 3,
						VersionReadyCountMap: map[string]int32{
							"latest-version": 6,
							"version1":       2, "version2": 1},
						VersionSortKeys: []string{"latest-version", "version1", "version2", "version3"},
					},
					ShardRuntimeArgs: ShardRuntimeArgs{
						LatestVersion: "latest-version",
					},
				},
				versionHoldingMap: map[string]int32{
					"latest-version": 2,
					"version1":       2, "version3": 0,
				},
			},
			want: map[string]int32{
				"latest-version": 2,
				"version1":       2, "version3": 0},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &InPlaceShardScheduler{}
			s.filterVersionHoldingMap(tt.args.args, tt.args.versionHoldingMap)
			if !reflect.DeepEqual(tt.args.versionHoldingMap, tt.want) {
				t.Errorf("filterVersionHoldingMap() = %v, want %v ", tt.args.versionHoldingMap, tt.want)
			}
		})
	}
}

func TestInPlaceShardScheduler_completeAvailableHold(t *testing.T) {
	type args struct {
		args              *ShardScheduleParams
		versionHoldingMap map[string]int32
	}
	tests := []struct {
		name string
		s    *InPlaceShardScheduler
		args args
		want map[string]int32
	}{
		{
			name: "test2",
			args: args{
				args: &ShardScheduleParams{
					ShardInternalArgs: ShardInternalArgs{
						minAvailable:         3,
						targetLatestReplicas: 3,
						VersionReadyCountMap: map[string]int32{
							"latest-version": 3,
							"version1":       1},
					},
					ShardRuntimeArgs: ShardRuntimeArgs{
						LatestVersion: "latest-version",
					},
				},
				versionHoldingMap: map[string]int32{
					"latest-version": 2,
					"version1":       0,
				},
			},
			want: map[string]int32{
				"latest-version": 3,
				"version1":       0,
			},
		},
		{
			name: "test2",
			args: args{
				args: &ShardScheduleParams{
					ShardInternalArgs: ShardInternalArgs{
						minAvailable:         3,
						targetLatestReplicas: 3,
						VersionReadyCountMap: map[string]int32{
							"latest-version": 4,
							"version1":       1},
					},
					ShardRuntimeArgs: ShardRuntimeArgs{
						LatestVersion: "latest-version",
					},
				},
				versionHoldingMap: map[string]int32{
					"latest-version": 2,
					"version1":       0,
				},
			},
			want: map[string]int32{
				"latest-version": 3,
				"version1":       0},
		},
		{
			name: "test2",
			args: args{
				args: &ShardScheduleParams{
					ShardInternalArgs: ShardInternalArgs{
						minAvailable:         2,
						targetLatestReplicas: 2,
						VersionReadyCountMap: map[string]int32{
							"latest-version": 3,
							"version1":       1},
					},
					ShardRuntimeArgs: ShardRuntimeArgs{
						LatestVersion: "latest-version",
					},
				},
				versionHoldingMap: map[string]int32{
					"latest-version": 2,
					"version1":       0},
			},
			want: map[string]int32{
				"latest-version": 2,
				"version1":       0},
		},
		{
			name: "test3",
			args: args{
				args: &ShardScheduleParams{
					ShardInternalArgs: ShardInternalArgs{
						minAvailable:         3,
						targetLatestReplicas: 3,
						VersionReadyCountMap: map[string]int32{
							"latest-version": 2,
							"version1":       0},
					},
					ShardRuntimeArgs: ShardRuntimeArgs{
						LatestVersion: "latest-version",
					},
				},
				versionHoldingMap: map[string]int32{
					"latest-version": 2,
					"version1":       0},
			},
			want: map[string]int32{
				"latest-version": 2,
				"version1":       0},
		},
		{
			name: "test1",
			args: args{
				args: &ShardScheduleParams{
					ShardInternalArgs: ShardInternalArgs{
						minAvailable:         2,
						targetLatestReplicas: 3,
						VersionReadyCountMap: map[string]int32{
							"latest-version": 0,
							"version1":       3},
					},
					ShardRuntimeArgs: ShardRuntimeArgs{
						LatestVersion: "latest-version",
					},
				},
				versionHoldingMap: map[string]int32{
					"latest-version": 2,
					"version1":       1},
			},
			want: map[string]int32{
				"latest-version": 2,
				"version1":       1},
		},
		{
			name: "test1",
			args: args{
				args: &ShardScheduleParams{
					ShardInternalArgs: ShardInternalArgs{
						minAvailable:         2,
						targetLatestReplicas: 3,
						VersionReadyCountMap: map[string]int32{
							"latest-version": 0,
							"version1":       3},
						VersionSortKeys: []string{"version1"},
					},
					ShardRuntimeArgs: ShardRuntimeArgs{
						LatestVersion: "latest-version",
					},
				},
				versionHoldingMap: map[string]int32{
					"latest-version": 0,
					"version1":       1},
			},
			want: map[string]int32{
				"latest-version": 0,
				"version1":       2},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &InPlaceShardScheduler{}
			s.completeAvailableHold(tt.args.args, tt.args.versionHoldingMap)
			if !reflect.DeepEqual(tt.args.versionHoldingMap, tt.want) {
				t.Errorf("ShardScheduleParams.calcResourceVersionHold() = %v, want %v", tt.args.versionHoldingMap, tt.want)
			}
		})
	}
}

func TestShardScheduleParams_initScheduleArgs(t *testing.T) {
	type fields struct {
		Plan              SchedulePlan
		ShardInternalArgs ShardInternalArgs
		ShardRuntimeArgs  ShardRuntimeArgs
	}
	type args struct {
		shard Shard
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *ShardScheduleParams
	}{
		{
			name: "init",
			args: args{
				shard: newShardStatus(t, "test1", 3, map[string]*VersionStatus{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(3, 2),
					"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 1),
				}, 0, 25),
			},
			fields: fields{
				Plan: SchedulePlan{
					Replicas: utils.Int32Ptr(3),
					Strategy: app.DeploymentStrategy{
						Type: app.RollingUpdateDeploymentStrategyType,
						RollingUpdate: &app.RollingUpdateDeployment{
							MaxUnavailable: getIntstrPtr("30%"),
							MaxSurge:       getIntstrPtr("30%"),
						},
					},
					LatestVersionRatio: utils.Int32Ptr(33),
				},
				ShardRuntimeArgs: ShardRuntimeArgs{
					LatestVersion: "453-2c0bc462f0e41079ffc6bc3307c1f0e4",
					ScheduleID:    "test",
				},
			},
			want: &ShardScheduleParams{
				Plan: SchedulePlan{
					Replicas: utils.Int32Ptr(3),
					Strategy: app.DeploymentStrategy{
						Type: app.RollingUpdateDeploymentStrategyType,
						RollingUpdate: &app.RollingUpdateDeployment{
							MaxUnavailable: getIntstrPtr("30%"),
							MaxSurge:       getIntstrPtr("30%"),
						},
					},
					LatestVersionRatio: utils.Int32Ptr(33),
				},
				ShardInternalArgs: ShardInternalArgs{
					name:                  "test1",
					Desired:               3, //目标值
					maxSurge:              1,
					minAvailable:          3,
					maxUnavailable:        0,
					maxSurgePercent:       30,
					minAvailablePercent:   70,
					maxUnavailablePercent: 30,
					latestVersionRatio:    33,
					maxReplicas:           4,
					targetLatestReplicas:  0,
					targetOldReplicas:     0,
					CurLatestReplicas:     3,
					CurOldReplicas:        1,
					CurTotalReplicas:      4,
					GroupVersionSortKeys:  []string{},
					VersionReadyCountMap: map[string]int32{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": 2,
						"455-761efa07c0f73213a3a8704a55e60330": 1,
					},
					VersionReadyPercentMap: map[string]float64{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": 50,
						"455-761efa07c0f73213a3a8704a55e60330": 25,
					},
					VersionCountMap: map[string]int32{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": 3,
						"455-761efa07c0f73213a3a8704a55e60330": 1,
					},
					VersionSortKeys: []string{"455-761efa07c0f73213a3a8704a55e60330", "453-2c0bc462f0e41079ffc6bc3307c1f0e4"},
				},
				ShardRuntimeArgs: ShardRuntimeArgs{
					Releasing:                false,
					ScheduleID:               "test",
					LatestVersion:            "453-2c0bc462f0e41079ffc6bc3307c1f0e4",
					GroupVersionToVersionMap: nil,
				},
			},
		},
		{
			name: "init",
			args: args{
				shard: newShardStatus(t, "test1", 3, map[string]*VersionStatus{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(3, 2),
					"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 1),
				}, 0, 25),
			},
			fields: fields{
				Plan: SchedulePlan{
					Replicas: utils.Int32Ptr(3),
					Strategy: app.DeploymentStrategy{
						Type: app.RollingUpdateDeploymentStrategyType,
						RollingUpdate: &app.RollingUpdateDeployment{
							MaxUnavailable: getIntstrPtr("30%"),
							MaxSurge:       getIntstrPtr("30%"),
						},
					},
					LatestVersionRatio: utils.Int32Ptr(33),
				},
				ShardRuntimeArgs: ShardRuntimeArgs{
					LatestVersion: "455-761efa07c0f73213a3a8704a55e60330",
					ScheduleID:    "test",
				},
			},
			want: &ShardScheduleParams{
				Plan: SchedulePlan{
					Replicas: utils.Int32Ptr(3),
					Strategy: app.DeploymentStrategy{
						Type: app.RollingUpdateDeploymentStrategyType,
						RollingUpdate: &app.RollingUpdateDeployment{
							MaxUnavailable: getIntstrPtr("30%"),
							MaxSurge:       getIntstrPtr("30%"),
						},
					},
					LatestVersionRatio: utils.Int32Ptr(33),
				},
				ShardInternalArgs: ShardInternalArgs{
					name:                  "test1",
					Desired:               3, //目标值
					maxSurge:              1,
					minAvailable:          3,
					maxUnavailable:        0,
					maxSurgePercent:       30,
					minAvailablePercent:   70,
					maxUnavailablePercent: 30,
					latestVersionRatio:    33,
					maxReplicas:           4,
					targetLatestReplicas:  0,
					targetOldReplicas:     0,
					CurLatestReplicas:     1,
					CurOldReplicas:        3,
					CurTotalReplicas:      4,
					GroupVersionSortKeys:  []string{},
					VersionReadyCountMap: map[string]int32{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": 2,
						"455-761efa07c0f73213a3a8704a55e60330": 1,
					},
					VersionReadyPercentMap: map[string]float64{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": 50,
						"455-761efa07c0f73213a3a8704a55e60330": 25,
					},
					VersionCountMap: map[string]int32{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": 3,
						"455-761efa07c0f73213a3a8704a55e60330": 1,
					},
					VersionSortKeys: []string{"453-2c0bc462f0e41079ffc6bc3307c1f0e4", "455-761efa07c0f73213a3a8704a55e60330"},
				},
				ShardRuntimeArgs: ShardRuntimeArgs{
					Releasing:                false,
					ScheduleID:               "test",
					LatestVersion:            "455-761efa07c0f73213a3a8704a55e60330",
					GroupVersionToVersionMap: nil,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := &ShardScheduleParams{
				Plan:              tt.fields.Plan,
				ShardInternalArgs: tt.fields.ShardInternalArgs,
				ShardRuntimeArgs:  tt.fields.ShardRuntimeArgs,
			}
			args.initScheduleArgs(tt.args.shard)
			if utils.ObjJSON(args) != utils.ObjJSON(tt.want) {
				t.Errorf("ShardScheduleParams.initScheduleArgs() = %v, want %v , got: %s , want:%s", (args), (tt.want), utils.ObjJSON(args), utils.ObjJSON(tt.want))
			}
		})
	}
}

func TestShardScheduleParams_getHoldCount(t *testing.T) {
	type fields struct {
		Plan              SchedulePlan
		ShardInternalArgs ShardInternalArgs
		ShardRuntimeArgs  ShardRuntimeArgs
	}
	type args struct {
		version string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   int32
	}{
		{
			fields: fields{
				ShardInternalArgs: ShardInternalArgs{
					versionHoldingMap: map[string]int32{
						"a": 1,
						"b": 6,
						"c": 7,
						"d": 9,
					},
				},
				ShardRuntimeArgs: ShardRuntimeArgs{
					LatestVersion: "a",
				},
			},
			args: args{
				version: "a",
			},
			want: 1,
		},
		{
			fields: fields{
				ShardInternalArgs: ShardInternalArgs{
					versionHoldingMap: map[string]int32{
						"a": 1,
						"b": 6,
						"c": 7,
						"d": 9,
					},
				},
				ShardRuntimeArgs: ShardRuntimeArgs{
					LatestVersion: "a",
				},
			},
			args: args{
				version: "b",
			},
			want: 6,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := &ShardScheduleParams{
				Plan:              tt.fields.Plan,
				ShardInternalArgs: tt.fields.ShardInternalArgs,
				ShardRuntimeArgs:  tt.fields.ShardRuntimeArgs,
			}
			if got := args.getHoldCount(tt.args.version); got != tt.want {
				t.Errorf("ShardScheduleParams.getHoldCount() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShardScheduleParams_getLatestVersionHoldCount(t *testing.T) {
	type fields struct {
		Plan              SchedulePlan
		ShardInternalArgs ShardInternalArgs
		ShardRuntimeArgs  ShardRuntimeArgs
	}
	tests := []struct {
		name   string
		fields fields
		want   int32
	}{
		{
			fields: fields{
				ShardInternalArgs: ShardInternalArgs{
					versionHoldingMap: map[string]int32{
						"a": 1,
						"b": 6,
						"c": 7,
						"d": 9,
					},
				},
				ShardRuntimeArgs: ShardRuntimeArgs{
					LatestVersion: "a",
				},
			},
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := &ShardScheduleParams{
				Plan:              tt.fields.Plan,
				ShardInternalArgs: tt.fields.ShardInternalArgs,
				ShardRuntimeArgs:  tt.fields.ShardRuntimeArgs,
			}
			if got := args.getLatestVersionHoldCount(); got != tt.want {
				t.Errorf("ShardScheduleParams.getLatestVersionHoldCount() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShardScheduleParams_getOldHoldCount(t *testing.T) {
	type fields struct {
		Plan              SchedulePlan
		ShardInternalArgs ShardInternalArgs
		ShardRuntimeArgs  ShardRuntimeArgs
	}
	tests := []struct {
		name   string
		fields fields
		want   int32
	}{
		{
			fields: fields{
				ShardInternalArgs: ShardInternalArgs{
					versionHoldingMap: map[string]int32{
						"a": 1,
						"b": 6,
						"c": 7,
						"d": 9,
					},
				},
				ShardRuntimeArgs: ShardRuntimeArgs{
					LatestVersion: "a",
				},
			},
			want: 22,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := &ShardScheduleParams{
				Plan:              tt.fields.Plan,
				ShardInternalArgs: tt.fields.ShardInternalArgs,
				ShardRuntimeArgs:  tt.fields.ShardRuntimeArgs,
			}
			if got := args.getOldHoldCount(); got != tt.want {
				t.Errorf("ShardScheduleParams.getOldHoldCount() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShardScheduleParams_getLatestReadyCount(t *testing.T) {
	type fields struct {
		Plan              SchedulePlan
		ShardInternalArgs ShardInternalArgs
		ShardRuntimeArgs  ShardRuntimeArgs
	}
	tests := []struct {
		name   string
		fields fields
		want   int32
	}{
		{
			fields: fields{
				ShardInternalArgs: ShardInternalArgs{
					VersionReadyCountMap: map[string]int32{
						"a": 1,
						"b": 6,
						"c": 7,
						"d": 9,
					},
				},
				ShardRuntimeArgs: ShardRuntimeArgs{
					LatestVersion: "a",
				},
			},
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := &ShardScheduleParams{
				Plan:              tt.fields.Plan,
				ShardInternalArgs: tt.fields.ShardInternalArgs,
				ShardRuntimeArgs:  tt.fields.ShardRuntimeArgs,
			}
			if got := args.getLatestReadyCount(); got != tt.want {
				t.Errorf("ShardScheduleParams.getLatestReadyCount() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShardScheduleParams_getOldReadyCount(t *testing.T) {
	type fields struct {
		Plan              SchedulePlan
		ShardInternalArgs ShardInternalArgs
		ShardRuntimeArgs  ShardRuntimeArgs
	}
	tests := []struct {
		name   string
		fields fields
		want   int32
	}{
		{
			fields: fields{
				ShardInternalArgs: ShardInternalArgs{
					VersionReadyCountMap: map[string]int32{
						"a": 1,
						"b": 6,
						"c": 7,
						"d": 9,
					},
				},
				ShardRuntimeArgs: ShardRuntimeArgs{
					LatestVersion: "a",
				},
			},
			want: 22,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := &ShardScheduleParams{
				Plan:              tt.fields.Plan,
				ShardInternalArgs: tt.fields.ShardInternalArgs,
				ShardRuntimeArgs:  tt.fields.ShardRuntimeArgs,
			}
			if got := args.getOldReadyCount(); got != tt.want {
				t.Errorf("ShardScheduleParams.getOldReadyCount() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShardScheduleParams_getLatestCount(t *testing.T) {
	type fields struct {
		Plan              SchedulePlan
		ShardInternalArgs ShardInternalArgs
		ShardRuntimeArgs  ShardRuntimeArgs
	}
	tests := []struct {
		name   string
		fields fields
		want   int32
	}{
		{
			fields: fields{
				ShardInternalArgs: ShardInternalArgs{
					VersionCountMap: map[string]int32{
						"a": 1,
						"b": 6,
						"c": 7,
						"d": 9,
					},
				},
				ShardRuntimeArgs: ShardRuntimeArgs{
					LatestVersion: "a",
				},
			},
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := &ShardScheduleParams{
				Plan:              tt.fields.Plan,
				ShardInternalArgs: tt.fields.ShardInternalArgs,
				ShardRuntimeArgs:  tt.fields.ShardRuntimeArgs,
			}
			if got := args.getLatestCount(); got != tt.want {
				t.Errorf("ShardScheduleParams.getLatestCount() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShardScheduleParams_getOldCount(t *testing.T) {
	type fields struct {
		Plan              SchedulePlan
		ShardInternalArgs ShardInternalArgs
		ShardRuntimeArgs  ShardRuntimeArgs
	}
	tests := []struct {
		name   string
		fields fields
		want   int32
	}{
		{
			fields: fields{
				ShardInternalArgs: ShardInternalArgs{
					VersionCountMap: map[string]int32{
						"a": 1,
						"b": 6,
						"c": 7,
						"d": 9,
					},
				},
				ShardRuntimeArgs: ShardRuntimeArgs{
					LatestVersion: "a",
				},
			},
			want: 22,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := &ShardScheduleParams{
				Plan:              tt.fields.Plan,
				ShardInternalArgs: tt.fields.ShardInternalArgs,
				ShardRuntimeArgs:  tt.fields.ShardRuntimeArgs,
			}
			if got := args.getOldCount(); got != tt.want {
				t.Errorf("ShardScheduleParams.getOldCount() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShardScheduleParams_isForward(t *testing.T) {
	type fields struct {
		Plan              SchedulePlan
		ShardInternalArgs ShardInternalArgs
		ShardRuntimeArgs  ShardRuntimeArgs
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "backward",
			fields: fields{
				ShardInternalArgs: ShardInternalArgs{
					VersionCountMap: map[string]int32{
						"a": 1,
						"b": 6,
					},
					CurTotalReplicas:   7,
					latestVersionRatio: 0,
				},
				ShardRuntimeArgs: ShardRuntimeArgs{
					LatestVersion: "a",
				},
			},
			want: false,
		},
		{
			name: "forward",
			fields: fields{
				ShardInternalArgs: ShardInternalArgs{
					VersionCountMap: map[string]int32{
						"a": 1,
						"b": 6,
					},
					CurTotalReplicas:   7,
					latestVersionRatio: 30,
				},
				ShardRuntimeArgs: ShardRuntimeArgs{
					LatestVersion: "a",
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := &ShardScheduleParams{
				Plan:              tt.fields.Plan,
				ShardInternalArgs: tt.fields.ShardInternalArgs,
				ShardRuntimeArgs:  tt.fields.ShardRuntimeArgs,
			}
			if got := args.isForward(); got != tt.want {
				t.Errorf("ShardScheduleParams.isForward() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShardScheduleParams_completeCPUCount(t *testing.T) {
	type fields struct {
		Plan              SchedulePlan
		ShardInternalArgs ShardInternalArgs
		ShardRuntimeArgs  ShardRuntimeArgs
	}
	type args struct {
		shard Shard
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := &ShardScheduleParams{
				Plan:              tt.fields.Plan,
				ShardInternalArgs: tt.fields.ShardInternalArgs,
				ShardRuntimeArgs:  tt.fields.ShardRuntimeArgs,
			}
			args.completeCPUCount(tt.args.shard)
		})
	}
}

func TestShardScheduleParams_calcResourceVersionHold(t *testing.T) {
	type fields struct {
		Plan              SchedulePlan
		ShardInternalArgs ShardInternalArgs
		ShardRuntimeArgs  ShardRuntimeArgs
	}
	type args struct {
		shard Shard
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   map[string]int32
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := &ShardScheduleParams{
				Plan:              tt.fields.Plan,
				ShardInternalArgs: tt.fields.ShardInternalArgs,
				ShardRuntimeArgs:  tt.fields.ShardRuntimeArgs,
			}
			if got := args.calcResourceVersionHold(tt.args.shard); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ShardScheduleParams.calcResourceVersionHold() = %v, want %v", got, tt.want)
			}
		})
	}
}
