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

	gomock "github.com/golang/mock/gomock"
)

func newShardStatus(t *testing.T, name string, replicas int32, versionReplicas map[string]*VersionStatus, dependencyLevel int32, latestVersionRatio int32) Shard {
	ctl := gomock.NewController(t)
	defer ctl.Finish()
	shard := NewMockShard(ctl)
	shard.EXPECT().GetName().Return(name).MinTimes(0)
	shard.EXPECT().GetCarbonRoleName().Return(name).MinTimes(0)
	shard.EXPECT().GetReplicas().Return(replicas).MinTimes(0)
	shard.EXPECT().GetRawReplicas().Return(replicas).MinTimes(0)
	shard.EXPECT().GetVersionStatus().Return(versionReplicas).MinTimes(0)
	shard.EXPECT().GetLatestVersionRatio(gomock.Any()).Return(latestVersionRatio).MinTimes(0)
	shard.EXPECT().GetDependencyLevel().Return(dependencyLevel).MinTimes(0)
	return shard
}

func newVersionStatus(count, readyCount int32) *VersionStatus {
	return &VersionStatus{
		Replicas:      count,
		ReadyReplicas: readyCount,
	}
}

func newDataVersionStatus(count, dataReadyCount int32) *VersionStatus {
	return &VersionStatus{
		Replicas:          count,
		DataReadyReplicas: dataReadyCount,
	}
}

func Test_calculateHoldMatrix(t *testing.T) {
	type args struct {
		shards                  []Shard
		latestPercent           int32
		targetShardGroupVersion string
		strategy                string
		maxUnavailable          int32
		maxSurge                int32
	}
	tests := []struct {
		name  string
		args  args
		want  map[string]float64
		want1 int32
		want2 int32
	}{
		{
			name: "2 2 test",
			args: args{
				shards: []Shard{
					newShardStatus(t, "test1", 2, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
						"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(1, 1),
					}, 0, 0),
					newShardStatus(t, "test2", 2, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
						"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 1),
					}, 1, 0),
				},
				latestPercent:           100,
				targetShardGroupVersion: "541-4abe5b7fe1a035cac4d1f7e6c13bed9a",
				maxUnavailable:          50,
				maxSurge:                50,
			},
			want: map[string]float64{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": 50,
				"455-761efa07c0f73213a3a8704a55e60330": 0,
				"458-95a70e220628e0853754ae8a5a0a8c45": 0,
				"541-4abe5b7fe1a035cac4d1f7e6c13bed9a": 50,
			},
			want1: 50,
			want2: 50,
		},
		{
			name: "2 2 test",
			args: args{
				shards: []Shard{
					newShardStatus(t, "test1", 2, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
						"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(1, 1),
					}, 0, 0),
					newShardStatus(t, "test2", 2, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
						"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 1),
					}, 1, 0),
				},
				latestPercent:           100,
				targetShardGroupVersion: "541-4abe5b7fe1a035cac4d1f7e6c13bed9a",
				maxUnavailable:          50,
				maxSurge:                50,
			},
			want: map[string]float64{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": 50,
				"455-761efa07c0f73213a3a8704a55e60330": 0,
				"458-95a70e220628e0853754ae8a5a0a8c45": 0,
				"541-4abe5b7fe1a035cac4d1f7e6c13bed9a": 50,
			},
			want1: 50,
			want2: 50,
		},
		{
			name: "2 2 test",
			args: args{
				shards: []Shard{
					newShardStatus(t, "test1", 2, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
						"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(3, 3),
					}, 0, 0),
					newShardStatus(t, "test2", 2, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
						"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(3, 3),
					}, 1, 0),
				},
				latestPercent:           100,
				targetShardGroupVersion: "541-4abe5b7fe1a035cac4d1f7e6c13bed9a",
				maxUnavailable:          25,
				maxSurge:                50,
			},
			want: map[string]float64{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": 0,
				"458-95a70e220628e0853754ae8a5a0a8c45": 75,
				"541-4abe5b7fe1a035cac4d1f7e6c13bed9a": 25,
			},
			want1: 25,
			want2: 25,
		},
		{
			name: "2 2 test1",
			args: args{
				shards: []Shard{
					newShardStatus(t, "test1", 2, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
						"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(1, 1),
					}, 0, 0),
					newShardStatus(t, "test2", 2, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 0),
						"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 1),
					}, 1, 0),
				},
				latestPercent:           100,
				targetShardGroupVersion: "541-4abe5b7fe1a035cac4d1f7e6c13bed9a",
				maxUnavailable:          50,
				maxSurge:                50,
			},
			want: map[string]float64{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": 50,
				"455-761efa07c0f73213a3a8704a55e60330": 0,
				"458-95a70e220628e0853754ae8a5a0a8c45": 0,
				"541-4abe5b7fe1a035cac4d1f7e6c13bed9a": 50,
			},
			want1: 50,
			want2: 50,
		},
		{
			name: "3 2 test",
			args: args{
				shards: []Shard{
					newShardStatus(t, "test1", 3, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
						"456-5f3dc3f6ee51f00b65238cdd4579618f": newVersionStatus(1, 1),
						"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(1, 1),
					}, 0, 0),
					newShardStatus(t, "test2", 2, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
						"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 1),
					}, 1, 0),
				},
				latestPercent:           100,
				targetShardGroupVersion: "541-4abe5b7fe1a035cac4d1f7e6c13bed9a",
				maxUnavailable:          50,
				maxSurge:                50,
			},
			want: map[string]float64{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": 33.333333,
				"455-761efa07c0f73213a3a8704a55e60330": 0,
				"456-5f3dc3f6ee51f00b65238cdd4579618f": 0,
				"458-95a70e220628e0853754ae8a5a0a8c45": 0,
				"541-4abe5b7fe1a035cac4d1f7e6c13bed9a": 50,
			},
			want1: 50,
			want2: 67,
		},

		{
			name: "3 2 test3",
			args: args{
				shards: []Shard{
					newShardStatus(t, "test1", 3, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
						"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 1),
						"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(1, 1),
					}, 0, 0),
					newShardStatus(t, "test2", 2, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
						"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 1),
					}, 1, 0),
				},
				latestPercent:           100,
				targetShardGroupVersion: "541-4abe5b7fe1a035cac4d1f7e6c13bed9a",
				maxUnavailable:          50,
				maxSurge:                50,
				strategy:                "oldFirst",
			},
			want: map[string]float64{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": 33.333333,
				"455-761efa07c0f73213a3a8704a55e60330": 16.666667,
				"458-95a70e220628e0853754ae8a5a0a8c45": 0,
				"541-4abe5b7fe1a035cac4d1f7e6c13bed9a": 50,
			},
			want1: 50,
			want2: 50,
		},
		{
			name: "3 2 test4",
			args: args{
				shards: []Shard{
					newShardStatus(t, "test1", 3, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 2),
						"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 1),
					}, 0, 0),
					newShardStatus(t, "test2", 2, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 2),
						"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(0, 0),
					}, 1, 0),
				},
				latestPercent:           100,
				targetShardGroupVersion: "541-4abe5b7fe1a035cac4d1f7e6c13bed9a",
				maxUnavailable:          50,
				maxSurge:                50,
			},
			want: map[string]float64{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": 50,
				"455-761efa07c0f73213a3a8704a55e60330": 0,
				"541-4abe5b7fe1a035cac4d1f7e6c13bed9a": 50,
			},
			want1: 50,
			want2: 50,
		},
		{
			name: "3 200 34",
			args: args{
				shards: []Shard{
					newShardStatus(t, "test1", 3, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 2),
						"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 1),
					}, 0, 0),
					newShardStatus(t, "test2", 200, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(200, 200),
						"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(0, 0),
					}, 1, 0),
				},
				latestPercent:           100,
				targetShardGroupVersion: "541-4abe5b7fe1a035cac4d1f7e6c13bed9a",
				maxUnavailable:          34,
				maxSurge:                50,
			},
			want: map[string]float64{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": 66,
				"455-761efa07c0f73213a3a8704a55e60330": 0,
				"541-4abe5b7fe1a035cac4d1f7e6c13bed9a": 34,
			},
			want1: 34,
			want2: 34,
		},
		{
			name: "old first",
			args: args{
				shards: []Shard{
					newShardStatus(t, "test1", 3, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
						"456-5f3dc3f6ee51f00b65238cdd4579618f": newVersionStatus(1, 1),
						"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(1, 1),
					}, 0, 0),
					newShardStatus(t, "test2", 3, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
						"456-5f3dc3f6ee51f00b65238cdd4579618f": newVersionStatus(1, 1),
						"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(1, 1),
					}, 1, 0),
				},
				latestPercent:           100,
				targetShardGroupVersion: "541-4abe5b7fe1a035cac4d1f7e6c13bed9a",
				maxUnavailable:          34,
				maxSurge:                10,
				strategy:                "oldFirst",
			},
			want: map[string]float64{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": 33.333333,
				"456-5f3dc3f6ee51f00b65238cdd4579618f": 32.666667,
				"458-95a70e220628e0853754ae8a5a0a8c45": 0,
				"541-4abe5b7fe1a035cac4d1f7e6c13bed9a": 34,
			},
			want1: 34,
			want2: 34,
		},
		{
			name: "old first1",
			args: args{
				shards: []Shard{
					newShardStatus(t, "test1", 4, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
						"456-5f3dc3f6ee51f00b65238cdd4579618f": newVersionStatus(1, 1),
						"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(2, 2),
					}, 0, 0),
					newShardStatus(t, "test2", 4, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
						"456-5f3dc3f6ee51f00b65238cdd4579618f": newVersionStatus(1, 1),
						"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(2, 2),
					}, 1, 0),
				},
				latestPercent:           100,
				targetShardGroupVersion: "541-4abe5b7fe1a035cac4d1f7e6c13bed9a",
				maxUnavailable:          25,
				maxSurge:                10,
				strategy:                "oldFirst",
			},
			want: map[string]float64{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": 0,
				"456-5f3dc3f6ee51f00b65238cdd4579618f": 25,
				"458-95a70e220628e0853754ae8a5a0a8c45": 50,
				"541-4abe5b7fe1a035cac4d1f7e6c13bed9a": 25,
			},
			want1: 25,
			want2: 25,
		},
		{
			name: "old first2",
			args: args{
				shards: []Shard{
					newShardStatus(t, "test1", 4, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
						"456-5f3dc3f6ee51f00b65238cdd4579618f": newVersionStatus(1, 1),
						"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(2, 0),
					}, 0, 0),
					newShardStatus(t, "test2", 4, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 0),
						"456-5f3dc3f6ee51f00b65238cdd4579618f": newVersionStatus(1, 1),
						"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(2, 2),
					}, 1, 0),
				},
				latestPercent:           100,
				targetShardGroupVersion: "541-4abe5b7fe1a035cac4d1f7e6c13bed9a",
				maxUnavailable:          25,
				maxSurge:                10,
				strategy:                "oldFirst",
			},
			want: map[string]float64{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": 25,
				"456-5f3dc3f6ee51f00b65238cdd4579618f": 25,
				"458-95a70e220628e0853754ae8a5a0a8c45": 25,
				"541-4abe5b7fe1a035cac4d1f7e6c13bed9a": 25,
			},
			want1: 25,
			want2: 25,
		},
		{
			name: "latest first",
			args: args{
				shards: []Shard{
					newShardStatus(t, "test1", 3, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
						"456-5f3dc3f6ee51f00b65238cdd4579618f": newVersionStatus(1, 1),
						"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(1, 1),
					}, 0, 0),
					newShardStatus(t, "test2", 3, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
						"456-5f3dc3f6ee51f00b65238cdd4579618f": newVersionStatus(1, 1),
						"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(1, 1),
					}, 1, 0),
				},
				latestPercent:           100,
				targetShardGroupVersion: "541-4abe5b7fe1a035cac4d1f7e6c13bed9a",
				maxUnavailable:          34,
				maxSurge:                10,
				strategy:                "latestFirst",
			},
			want: map[string]float64{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": 33.333333,
				"456-5f3dc3f6ee51f00b65238cdd4579618f": 32.666667,
				"458-95a70e220628e0853754ae8a5a0a8c45": 0,
				"541-4abe5b7fe1a035cac4d1f7e6c13bed9a": 34,
			},
			want1: 34,
			want2: 34,
		},
		{
			name: "less first",
			args: args{
				shards: []Shard{
					newShardStatus(t, "test1", 5, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 2),
						"456-5f3dc3f6ee51f00b65238cdd4579618f": newVersionStatus(1, 1),
						"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(2, 2),
					}, 0, 0),
					newShardStatus(t, "test2", 5, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 2),
						"456-5f3dc3f6ee51f00b65238cdd4579618f": newVersionStatus(1, 1),
						"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(2, 2),
					}, 1, 0),
				},
				latestPercent:           100,
				targetShardGroupVersion: "541-4abe5b7fe1a035cac4d1f7e6c13bed9a",
				maxUnavailable:          34,
				maxSurge:                10,
				strategy:                "lessFirst",
			},
			want: map[string]float64{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": 40,
				"456-5f3dc3f6ee51f00b65238cdd4579618f": 0,
				"458-95a70e220628e0853754ae8a5a0a8c45": 26,
				"541-4abe5b7fe1a035cac4d1f7e6c13bed9a": 34,
			},
			want1: 34,
			want2: 34,
		},
		{
			name: "roll back",
			args: args{
				shards: []Shard{
					newShardStatus(t, "test6", 12, map[string]*VersionStatus{
						"541-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(7, 7),
						"540-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(5, 5),
					}, 0, 0),
				},
				latestPercent:           0,
				targetShardGroupVersion: "541-4abe5b7fe1a035cac4d1f7e6c13bed9a",
				maxUnavailable:          20,
				maxSurge:                50,
			},
			want: map[string]float64{
				"540-4abe5b7fe1a035cac4d1f7e6c13bed9a": 41.666667,
				"541-4abe5b7fe1a035cac4d1f7e6c13bed9a": 38.333333,
			},
			want1: 39,
			want2: 0,
		},
		{
			name: "one version",
			args: args{
				shards: []Shard{
					newShardStatus(t, "test7", 10, map[string]*VersionStatus{
						"541-4abe5b7fe1a035cac4d1f7e6c13bed9a": newVersionStatus(10, 10),
					}, 0, 0),
				},
				latestPercent:           0,
				targetShardGroupVersion: "541-4abe5b7fe1a035cac4d1f7e6c13bed9a",
				maxUnavailable:          20,
				maxSurge:                50,
			},
			want: map[string]float64{
				"541-4abe5b7fe1a035cac4d1f7e6c13bed9a": 100,
			},
			want1: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, got2 := calculateHoldMatrix(tt.args.shards, tt.args.latestPercent, tt.args.strategy, tt.args.targetShardGroupVersion, tt.args.maxUnavailable, tt.args.maxSurge, nil)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("calculateHoldMatrix() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("calculateHoldMatrix() got1 = %v, want %v", got1, tt.want1)
			}
			if got2 != tt.want2 {
				t.Errorf("calculateHoldMatrix() got2 = %v, want %v", got2, tt.want2)
			}
		})
	}
}

func Test_getTotalRatio(t *testing.T) {
	type args struct {
		matrix map[string]float64
	}
	tests := []struct {
		name string
		args args
		want float64
	}{
		{
			name: "nil",
			args: args{},
			want: 0.0,
		},
		{
			name: "normal",
			args: args{
				matrix: map[string]float64{
					"a": 1.1,
					"b": 1.1,
				},
			},
			want: 2.2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getTotalRatio(tt.args.matrix); got != tt.want {
				t.Errorf("getTotalRatio() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getVersionReadyReplicas(t *testing.T) {
	type args struct {
		vs *VersionStatus
	}
	tests := []struct {
		name string
		args args
		want int32
	}{
		{
			name: "nil",
			args: args{},
			want: 0,
		},
		{
			name: "normal",
			args: args{
				vs: &VersionStatus{
					ReadyReplicas: 3,
				},
			},
			want: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getVersionReadyReplicas(tt.args.vs); got != tt.want {
				t.Errorf("getVersionReadyReplicas() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getVersionAllReplicas(t *testing.T) {
	type args struct {
		vs *VersionStatus
	}
	tests := []struct {
		name string
		args args
		want int32
	}{
		{
			name: "nil",
			args: args{},
			want: 0,
		},
		{
			name: "normal",
			args: args{
				vs: &VersionStatus{
					Replicas: 3,
				},
			},
			want: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getVersionAllReplicas(tt.args.vs); got != tt.want {
				t.Errorf("getVersionAllReplicas() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_shouldIgnoreNewShard(t *testing.T) {
	type args struct {
		shard     Shard
		newShards []string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "test",
			args: args{
				shard: newShardStatus(t, "test1", 4, map[string]*VersionStatus{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 1),
					"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(2, 2),
				}, 0, 0),
			},
			want: false,
		},
		{
			name: "test",
			args: args{
				shard: newShardStatus(t, "test1", 4, map[string]*VersionStatus{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 1),
					"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(2, 2),
				}, 0, 0),
				newShards: []string{"test2"},
			},
			want: false,
		},
		{
			name: "test",
			args: args{
				shard: newShardStatus(t, "test1", 4, map[string]*VersionStatus{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 1),
					"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(2, 0),
				}, 0, 0),
				newShards: []string{"test1"},
			},
			want: false,
		},
		{
			name: "test",
			args: args{
				shard: newShardStatus(t, "test1", 4, map[string]*VersionStatus{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 0),
					"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(2, 0),
				}, 0, 0),
				newShards: []string{"test1"},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldIgnoreNewShard(tt.args.shard, tt.args.newShards); got != tt.want {
				t.Errorf("shouldIgnoreNewShard() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_initShardMatrix(t *testing.T) {
	type args struct {
		shards    []Shard
		f         getVersionStatusReplicas
		newShards []string
	}
	tests := []struct {
		name string
		args args
		want map[string]float64
	}{
		{
			name: "50",
			args: args{
				shards: []Shard{
					newShardStatus(t, "test1", 4, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 1),
						"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(2, 2),
					}, 0, 0),
					newShardStatus(t, "test2", 4, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 2),
						"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(2, 1),
					}, 1, 0),
				},
				f: getVersionAllReplicas,
			},
			want: map[string]float64{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": 50,
				"455-761efa07c0f73213a3a8704a55e60330": 0,
				"458-95a70e220628e0853754ae8a5a0a8c45": 0,
			},
		},
		{
			name: "25",
			args: args{
				shards: []Shard{
					newShardStatus(t, "test1", 4, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
						"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(2, 2),
					}, 0, 0),
					newShardStatus(t, "test2", 4, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 2),
						"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(2, 1),
					}, 1, 0),
				},
				f: getVersionAllReplicas,
			},
			want: map[string]float64{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": 25,
				"455-761efa07c0f73213a3a8704a55e60330": 0,
				"458-95a70e220628e0853754ae8a5a0a8c45": 0,
			},
		},
		{
			name: "33.3333",
			args: args{
				shards: []Shard{
					newShardStatus(t, "test1", 3, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
						"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(2, 2),
					}, 0, 0),
					newShardStatus(t, "test2", 3, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 2),
						"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(2, 1),
					}, 1, 0),
				},
				f: getVersionAllReplicas,
			},
			want: map[string]float64{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": 33.333333,
				"455-761efa07c0f73213a3a8704a55e60330": 0,
				"458-95a70e220628e0853754ae8a5a0a8c45": 0,
			},
		},
		{
			name: "0",
			args: args{
				shards: []Shard{
					newShardStatus(t, "test1", 3, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
						"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(2, 2),
					}, 0, 0),
					newShardStatus(t, "test2", 3, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(0, 0),
						"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(3, 0),
					}, 1, 0),
				},
				f: getVersionAllReplicas,
			},
			want: map[string]float64{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": 0,
				"455-761efa07c0f73213a3a8704a55e60330": 0,
				"458-95a70e220628e0853754ae8a5a0a8c45": 0,
			},
		},
		{
			name: "newshard",
			args: args{
				shards: []Shard{
					newShardStatus(t, "test1", 3, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
						"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(2, 2),
					}, 0, 0),
					newShardStatus(t, "test2", 3, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(0, 0),
						"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(3, 0),
					}, 1, 0),
				},
				f:         getVersionAllReplicas,
				newShards: []string{"test2"},
			},
			want: map[string]float64{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": 33.333333,
				"455-761efa07c0f73213a3a8704a55e60330": 0,
				"458-95a70e220628e0853754ae8a5a0a8c45": 66.666667,
			},
		},
		{
			name: "newshard",
			args: args{
				shards: []Shard{
					newShardStatus(t, "test1", 3, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
						"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(2, 2),
					}, 0, 0),
					newShardStatus(t, "test2", 3, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(0, 0),
						"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(3, 0),
					}, 1, 0),
				},
				f:         getVersionAllReplicas,
				newShards: []string{"test2"},
			},
			want: map[string]float64{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": 33.333333,
				"458-95a70e220628e0853754ae8a5a0a8c45": 66.666667,
			},
		},
		{
			name: "newshard",
			args: args{
				shards: []Shard{
					newShardStatus(t, "test1", 3, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
						"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(2, 2),
					}, 0, 0),
					newShardStatus(t, "test2", 3, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(0, 0),
						"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(3, 3),
					}, 1, 0),
				},
				f:         getVersionAllReplicas,
				newShards: []string{"test1"},
			},
			want: map[string]float64{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": 0,
				"455-761efa07c0f73213a3a8704a55e60330": 100,
				"458-95a70e220628e0853754ae8a5a0a8c45": 0,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := initShardMatrix(tt.args.shards, tt.args.f, getTargetReplicas, tt.args.newShards); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("initShardMatrix() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_calculateHoldCount(t *testing.T) {
	type args struct {
		replicas int32
		percent  float64
	}
	tests := []struct {
		name string
		args args
		want int32
	}{
		{
			name: "0",
			args: args{
				replicas: 0,
				percent:  0.0,
			},
			want: 0,
		},
		{
			name: "floor",
			args: args{
				replicas: 10,
				percent:  33.33333,
			},
			want: 4,
		},
		{
			name: "ceil",
			args: args{
				replicas: 10,
				percent:  66.6666,
			},
			want: 7,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := calculateHoldCount(tt.args.replicas, tt.args.percent); got != tt.want {
				t.Errorf("calculateHoldCount() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_transferHoldMatrix(t *testing.T) {
	type args struct {
		holdMatrixPercent map[string]float64
		replicas          int32
	}
	tests := []struct {
		name string
		args args
		want map[string]int32
	}{
		{
			name: "normal",
			args: args{
				holdMatrixPercent: map[string]float64{
					"a": 33.3333,
					"b": 66.6666,
					"c": 0,
				},
				replicas: 7,
			},
			want: map[string]int32{
				"a": 3,
				"b": 5,
				"c": 0,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := transferHoldMatrix(tt.args.holdMatrixPercent, tt.args.replicas); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("transferHoldMatrix() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestController_transferHoldMatrix(t *testing.T) {
	type args struct {
		holdMatrixPercent map[string]float64
		replicas          int32
	}
	tests := []struct {
		name string
		args args
		want map[string]int32
	}{
		{
			name: "test1",
			args: args{
				holdMatrixPercent: map[string]float64{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": 50,
					"455-761efa07c0f73213a3a8704a55e60330": 50},
				replicas: 3,
			},
			want: map[string]int32{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": 2,
				"455-761efa07c0f73213a3a8704a55e60330": 2},
		},
		{
			name: "test2",
			args: args{
				holdMatrixPercent: map[string]float64{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": 33.333333,
					"455-761efa07c0f73213a3a8704a55e60330": 50},
				replicas: 3,
			},
			want: map[string]int32{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": 1,
				"455-761efa07c0f73213a3a8704a55e60330": 2},
		},
		{
			name: "test3",
			args: args{
				holdMatrixPercent: map[string]float64{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": 66.666666,
					"455-761efa07c0f73213a3a8704a55e60330": 50},
				replicas: 3,
			},
			want: map[string]int32{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": 2,
				"455-761efa07c0f73213a3a8704a55e60330": 2},
		},
		{
			name: "test4",
			args: args{
				holdMatrixPercent: map[string]float64{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": 50,
					"455-761efa07c0f73213a3a8704a55e60330": 50},
				replicas: 3,
			},
			want: map[string]int32{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": 2,
				"455-761efa07c0f73213a3a8704a55e60330": 2},
		},
		{
			name: "test5",
			args: args{
				holdMatrixPercent: map[string]float64{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": 50},
				replicas: 3,
			},
			want: map[string]int32{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": 2},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := transferHoldMatrix(tt.args.holdMatrixPercent, tt.args.replicas); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("transferHoldMatrix() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_sortVersions(t *testing.T) {
	type args struct {
		versions  []string
		latest    string
		available map[string]float64
		strategy  string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "latestFirst",
			args: args{
				versions: []string{
					"2-bbb",
					"1-aaa",
					"3-ccc",
				},
				latest:    "3-ccc",
				available: nil,
				strategy:  "latestFirst",
			},
			want: []string{
				"1-aaa",
				"2-bbb",
				"3-ccc",
			},
		},
		{
			name: "oldFirst",
			args: args{
				versions: []string{
					"2-bbb",
					"1-aaa",
					"3-ccc",
				},
				latest:    "3-ccc",
				available: map[string]float64{"1-aaa": 100.0},
				strategy:  "oldFirst",
			},
			want: []string{
				"3-ccc",
				"2-bbb",
				"1-aaa",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sortVersions(tt.args.versions, tt.args.latest, tt.args.available, 20.0, tt.args.strategy)
			fmt.Println(tt.want)
			if !reflect.DeepEqual(tt.want, tt.args.versions) {
				t.Errorf("sortVersions() = %v, want %v", tt.args.versions, tt.want)
			}
		})
	}
}

func Test_isLatestRowComplete(t *testing.T) {
	type args struct {
		shards                  []Shard
		targetShardGroupVersion string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "false",
			args: args{
				shards: []Shard{
					newShardStatus(t, "test1", 3, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newDataVersionStatus(1, 1),
						"458-95a70e220628e0853754ae8a5a0a8c45": newDataVersionStatus(2, 2),
					}, 0, 0),
					newShardStatus(t, "test2", 3, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newDataVersionStatus(0, 0),
						"458-95a70e220628e0853754ae8a5a0a8c45": newDataVersionStatus(3, 0),
					}, 1, 0),
				},
				targetShardGroupVersion: "458-95a70e220628e0853754ae8a5a0a8c45",
			},
			want: false,
		},
		{
			name: "false",
			args: args{
				shards: []Shard{
					newShardStatus(t, "test1", 3, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newDataVersionStatus(1, 1),
						"458-95a70e220628e0853754ae8a5a0a8c45": newDataVersionStatus(2, 2),
					}, 0, 0),
					newShardStatus(t, "test2", 3, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newDataVersionStatus(0, 0),
						"455-761efa07c0f73213a3a8704a55e60330": newDataVersionStatus(3, 0),
					}, 1, 0),
				},
				targetShardGroupVersion: "458-95a70e220628e0853754ae8a5a0a8c45",
			},
			want: false,
		},
		{
			name: "true",
			args: args{
				shards: []Shard{
					newShardStatus(t, "test1", 3, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newDataVersionStatus(1, 1),
						"458-95a70e220628e0853754ae8a5a0a8c45": newDataVersionStatus(2, 2),
					}, 0, 0),
					newShardStatus(t, "test2", 3, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newDataVersionStatus(0, 0),
						"458-95a70e220628e0853754ae8a5a0a8c45": newDataVersionStatus(3, 1),
					}, 1, 0),
				},
				targetShardGroupVersion: "458-95a70e220628e0853754ae8a5a0a8c45",
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isLatestRowComplete(tt.args.shards, tt.args.targetShardGroupVersion); got != tt.want {
				t.Errorf("isLatestRowComplete() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getRollingStrategy(t *testing.T) {
	type args struct {
		strategy        string
		defaultStrategy string
		available       map[string]float64
		maxUnavailable  float64
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "default",
			args: args{
				strategy:       "",
				maxUnavailable: 20.0,
				available: map[string]float64{
					"1": 80.0,
				},
			},
			want: RollingStrategyLessFirst,
		},
		{
			name: "old",
			args: args{
				strategy:       RollingStrategyOldFirst,
				maxUnavailable: 20.0,
				available: map[string]float64{
					"1": 80.0,
				},
			},
			want: RollingStrategyOldFirst,
		},
		{
			name: "latest",
			args: args{
				strategy:       RollingStrategyOldFirst,
				maxUnavailable: 20.0,
				available: map[string]float64{
					"1": 70.0,
				},
			},
			want: RollingStrategyLatestFirst,
		},
		{
			name: "latest",
			args: args{
				strategy:       RollingStrategyOldFirst,
				maxUnavailable: 50.0,
				available: map[string]float64{
					"1": 80.0,
				},
			},
			want: RollingStrategyLatestFirst,
		},
		{
			name: "less",
			args: args{
				strategy:       "",
				maxUnavailable: 50.0,
				available: map[string]float64{
					"1": 60.0,
				},
				defaultStrategy: RollingStrategyOldFirst,
			},
			want: RollingStrategyLatestFirst,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.defaultStrategy != "" {
				defaultRollingStrategy = tt.args.defaultStrategy
				defer func() {
					defaultRollingStrategy = RollingStrategyLessFirst
				}()
			}
			if got := getRollingStrategy(tt.args.strategy, tt.args.available, tt.args.maxUnavailable); got != tt.want {
				t.Errorf("getRollingStrategy() = %v, want %v", got, tt.want)
			}
		})
	}
}
