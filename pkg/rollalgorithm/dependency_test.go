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
)

func Test_calculateDependencyMatrix(t *testing.T) {
	type args struct {
		shard Shard
		group []Shard
	}
	tests := []struct {
		name string
		args args
		want map[string]float64
	}{
		{
			name: "normal",
			args: args{
				shard: newShardStatus(t, "test2", 5, map[string]*VersionStatus{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
					"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(4, 4),
				}, 1, 0),
				group: []Shard{
					newShardStatus(t, "test1", 3, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
						"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(3, 3),
					}, 0, 0),
					newShardStatus(t, "test2", 5, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
						"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(4, 4),
					}, 1, 0),
				},
			},
			want: map[string]float64{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": 33.333333,
				"458-95a70e220628e0853754ae8a5a0a8c45": 100,
			},
		},
		{
			name: "normal",
			args: args{
				shard: newShardStatus(t, "test2", 5, map[string]*VersionStatus{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
					"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(4, 4),
				}, 1, 0),
				group: []Shard{
					newShardStatus(t, "test1", 4, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
						"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(3, 3),
					}, 0, 0),
					newShardStatus(t, "test2", 5, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
						"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(4, 4),
					}, 1, 0),
				},
			},
			want: map[string]float64{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": 25,
				"458-95a70e220628e0853754ae8a5a0a8c45": 75,
			},
		},
		{
			name: "normal",
			args: args{
				shard: newShardStatus(t, "test0", 4, map[string]*VersionStatus{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 1),
					"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(2, 2),
				}, 2, 0),
				group: []Shard{
					newShardStatus(t, "test1", 4, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 1),
						"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(2, 2),
					}, 0, 0),
					newShardStatus(t, "test2", 4, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 2),
						"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(2, 1),
					}, 1, 0),
				},
			},
			want: map[string]float64{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": 25,
				"455-761efa07c0f73213a3a8704a55e60330": 0,
				"458-95a70e220628e0853754ae8a5a0a8c45": 0,
			},
		},
		{
			name: "normal",
			args: args{
				shard: newShardStatus(t, "test0", 4, map[string]*VersionStatus{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 1),
					"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(2, 2),
				}, 2, 0),
				group: []Shard{
					newShardStatus(t, "test1", 4, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 1),
						"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(2, 1),
					}, 0, 0),
					newShardStatus(t, "test2", 4, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 2),
						"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(2, 1),
					}, 1, 0),
				},
			},
			want: map[string]float64{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": 25,
				"455-761efa07c0f73213a3a8704a55e60330": 25,
			},
		},
		{
			name: "normal",
			args: args{
				shard: newShardStatus(t, "test0", 4, map[string]*VersionStatus{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 1),
					"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(2, 2),
				}, 0, 0),
				group: []Shard{
					newShardStatus(t, "test1", 4, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 1),
						"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(2, 1),
					}, 0, 0),
					newShardStatus(t, "test2", 4, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 2),
						"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(2, 1),
					}, 1, 0),
				},
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := calculateDependencyMatrix(tt.args.shard, tt.args.group); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("calculateDependencyMatrix() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getDependencyShards(t *testing.T) {
	type args struct {
		shard Shard
		group []Shard
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "normal",
			args: args{
				shard: newShardStatus(t, "test0", 4, map[string]*VersionStatus{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 1),
					"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(2, 2),
				}, 0, 0),
				group: []Shard{
					newShardStatus(t, "test1", 4, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 1),
						"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(2, 2),
					}, 0, 0),
					newShardStatus(t, "test2", 4, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 2),
						"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(2, 1),
					}, 0, 0),
				},
			},
			want: []string{},
		},
		{
			name: "normal",
			args: args{
				shard: newShardStatus(t, "test0", 4, map[string]*VersionStatus{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 1),
					"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(2, 2),
				}, 1, 0),
				group: []Shard{
					newShardStatus(t, "test1", 4, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 1),
						"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(2, 2),
					}, 0, 0),
					newShardStatus(t, "test2", 4, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 2),
						"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(2, 1),
					}, 1, 0),
				},
			},
			want: []string{"test1"},
		},
		{
			name: "normal",
			args: args{
				shard: newShardStatus(t, "test0", 4, map[string]*VersionStatus{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 1),
					"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(2, 2),
				}, 2, 0),
				group: []Shard{
					newShardStatus(t, "test1", 4, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 1),
						"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(2, 2),
					}, 0, 0),
					newShardStatus(t, "test2", 4, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 2),
						"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(2, 1),
					}, 1, 0),
				},
			},
			want: []string{"test1", "test2"},
		},
		{
			name: "normal",
			args: args{
				shard: newShardStatus(t, "test2", 5, map[string]*VersionStatus{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
					"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(4, 4),
				}, 1, 0),
				group: []Shard{
					newShardStatus(t, "test1", 3, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
						"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(3, 3),
					}, 0, 0),
					newShardStatus(t, "test2", 5, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(1, 1),
						"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(4, 4),
					}, 1, 0),
				},
			},
			want: []string{"test1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getDependencyShards(tt.args.shard, tt.args.group)
			if len(got) != len(tt.want) {
				t.Errorf("getDependencyShards() = %v, want %v", len(got), len(tt.want))
			}
			for i := range tt.want {
				found := false
				for j := range got {
					if tt.want[i] == got[j].GetName() {
						found = true
					}
				}
				if !found {
					t.Errorf("not found %s", (tt.want[i]))
				}
			}
		})
	}
}

func Test_needDependency(t *testing.T) {
	type args struct {
		group []Shard
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "normal",
			args: args{
				group: []Shard{
					newShardStatus(t, "test1", 4, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 1),
						"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(2, 2),
					}, 0, 0),
					newShardStatus(t, "test2", 4, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 2),
						"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(2, 1),
					}, 0, 0),
				},
			},
			want: false,
		},
		{
			name: "normal",
			args: args{
				group: []Shard{
					newShardStatus(t, "test1", 4, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 1),
						"458-95a70e220628e0853754ae8a5a0a8c45": newVersionStatus(2, 2),
					}, 0, 0),
					newShardStatus(t, "test2", 4, map[string]*VersionStatus{
						"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(2, 2),
						"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(2, 1),
					}, 1, 0),
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := needDependency(tt.args.group); got != tt.want {
				t.Errorf("needDependency() = %v, want %v", got, tt.want)
			}
		})
	}
}
