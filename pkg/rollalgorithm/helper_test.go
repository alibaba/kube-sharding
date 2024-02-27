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
	"testing"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
)

func TestGroupScheduleParams_String(t *testing.T) {
	type fields struct {
		params *GroupScheduleParams
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "string",
			fields: fields{
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
						}, 0, 0),
					},
					nil,
				),
			},
			want: "ShardGroupVersion: 458-95a70e220628e0853754ae8a5a0a8c45, LatestPercent: 100, MaxUnavailable: 20, versions: 453-2c0bc462f0e41079ffc6bc3307c1f0e4 || 458-95a70e220628e0853754ae8a5a0a8c45 || --- role test1 : 3/3 || 1/1 || --- role test2 : 4/4 || 1/1 || ---",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gsp := tt.fields.params
			if got := gsp.String(); got != tt.want {
				t.Errorf("GroupScheduleParams.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGroupScheduleTarget_String(t *testing.T) {
	tests := []struct {
		name string
		gst  *GroupScheduleTarget
		want string
	}{
		{
			name: "string",
			gst: func() *GroupScheduleTarget {
				targets := map[string]ShardScheduleParams{
					"shard1": ShardScheduleParams{
						Plan: SchedulePlan{
							Replicas: utils.Int32Ptr(10),
							VersionHoldMatrix: map[string]int32{
								"453-2c0bc462f0e41079ffc6bc3307c1f0e4": 3,
								"458-95a70e220628e0853754ae8a5a0a8c45": 3,
							},
							VersionHoldMatrixPercent: map[string]float64{
								"453-2c0bc462f0e41079ffc6bc3307c1f0e4": 0.3,
								"458-95a70e220628e0853754ae8a5a0a8c45": 0.3,
							},
						},
					},
					"shard2": ShardScheduleParams{
						Plan: SchedulePlan{
							Replicas: utils.Int32Ptr(10),
							VersionHoldMatrix: map[string]int32{
								"453-2c0bc462f0e41079ffc6bc3307c1f0e4": 3,
								"457-95a70e220628e0853754ae8a5a0a8c45": 1,
							},
							VersionHoldMatrixPercent: map[string]float64{
								"453-2c0bc462f0e41079ffc6bc3307c1f0e4": 0.3,
								"457-95a70e220628e0853754ae8a5a0a8c45": 0.1,
							},
						},
					},
				}
				groupTarget := GroupScheduleTarget(targets)
				return &groupTarget
			}(),
			want: "versions: 453-2c0bc462f0e41079ffc6bc3307c1f0e4 || 457-95a70e220628e0853754ae8a5a0a8c45 || 458-95a70e220628e0853754ae8a5a0a8c45 || ---role shard1:, replicas: 10, versions: 3/0.300000 || 0/0.000000 || 3/0.300000 || ---role shard2:, replicas: 10, versions: 3/0.300000 || 1/0.100000 || 0/0.000000 || ---",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.gst.String(); got != tt.want {
				t.Errorf("GroupScheduleTarget.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_versionsStatus_String(t *testing.T) {
	tests := []struct {
		name string
		s    *versionsStatus
		want string
	}{
		{
			name: "string",
			s: getversionsStatus(map[string]*VersionStatus{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": &VersionStatus{
					Replicas:      3,
					ReadyReplicas: 2,
				},
				"457-95a70e220628e0853754ae8a5a0a8c45": &VersionStatus{
					Replicas:      5,
					ReadyReplicas: 3,
				},
			}),
			want: "versions: 453-2c0bc462f0e41079ffc6bc3307c1f0e4: 3/2 || 457-95a70e220628e0853754ae8a5a0a8c45: 5/3 || ",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.String(); got != tt.want {
				t.Errorf("versionsStatus.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShardScheduleParams_String(t *testing.T) {
	type fields struct {
		Plan              SchedulePlan
		ShardInternalArgs ShardInternalArgs
		ShardRuntimeArgs  ShardRuntimeArgs
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "string",
			fields: fields{
				Plan: SchedulePlan{
					Replicas:           utils.Int32Ptr(10),
					LatestVersionRatio: utils.Int32Ptr(100),
					VersionHoldMatrix: map[string]int32{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": 3,
						"457-95a70e220628e0853754ae8a5a0a8c45": 1,
					},
					VersionHoldMatrixPercent: map[string]float64{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": 0.3,
						"457-95a70e220628e0853754ae8a5a0a8c45": 0.1,
					},
				},
				ShardInternalArgs: ShardInternalArgs{
					name:                 "test-role",
					CurLatestReplicas:    8,
					CurOldReplicas:       2,
					CurTotalReplicas:     10,
					Desired:              10,
					maxSurge:             0,
					minAvailable:         8,
					maxUnavailable:       2,
					latestVersionRatio:   100,
					maxReplicas:          10,
					targetLatestReplicas: 8,
					targetOldReplicas:    2,
				},
				ShardRuntimeArgs: ShardRuntimeArgs{
					LatestVersion: "457-95a70e220628e0853754ae8a5a0a8c45",
				},
			},
			want: "[test-role] rollingset initScheduleArgs latestVersion: 457-95a70e220628e0853754ae8a5a0a8c45, curReplicas: 10, curLatestReplicas: 8, curOldReplicas: 2,Desired: 10, maxSurge: 0,  minAvailable: 8, maxUnavailable: 2, maxReplicas: 10, latestVersionRatio: 100, targetLatestReplicas: 8, targetOldReplicas: 2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &ShardScheduleParams{
				Plan:              tt.fields.Plan,
				ShardInternalArgs: tt.fields.ShardInternalArgs,
				ShardRuntimeArgs:  tt.fields.ShardRuntimeArgs,
			}
			if got := s.String(); got != tt.want {
				t.Errorf("ShardScheduleParams.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_printShardTargetMatrix(t *testing.T) {
	type args struct {
		targets                  map[string]int32
		groupVersionToVersionMap map[string]string
		sortedKeys               []string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "string",
			args: args{
				targets: map[string]int32{
					"453": 3,
					"458": 1,
				},
				groupVersionToVersionMap: map[string]string{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
					"458-95a70e220628e0853754ae8a5a0a8c45": "458",
					"455-761efa07c0f73213a3a8704a55e60330": "455",
					"451-4abe5b7fe1a035cac4d1f7e6c13bed9a": "451",
				},
				sortedKeys: []string{
					"458-95a70e220628e0853754ae8a5a0a8c45", "453-2c0bc462f0e41079ffc6bc3307c1f0e4",
				},
			},
			want: "versions: 458-95a70e220628e0853754ae8a5a0a8c45: 1 || 453-2c0bc462f0e41079ffc6bc3307c1f0e4: 3 || ",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := printShardTargetMatrix(tt.args.targets, tt.args.groupVersionToVersionMap, tt.args.sortedKeys, nil); got != tt.want {
				t.Errorf("printShardTargetMatrix() = %v, want %v", got, tt.want)
			}
		})
	}
}
