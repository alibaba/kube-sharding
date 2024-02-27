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
	"sort"
	"testing"

	gomock "github.com/golang/mock/gomock"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func newMockNode(ctl *gomock.Controller, name, version string, ready, empty, release bool, score, cost int) *MockNode {
	node := NewMockNode(ctl)
	node.EXPECT().GetName().Return(name).MinTimes(0)
	node.EXPECT().GetVersion().Return(version).MinTimes(0)
	node.EXPECT().GetVersionPlan().Return(version).MinTimes(0)
	node.EXPECT().IsReady().Return(ready).MinTimes(0)
	node.EXPECT().IsEmpty().Return(empty).MinTimes(0)
	node.EXPECT().IsRelease().Return(release).MinTimes(0)
	node.EXPECT().GetScore().Return(score).MinTimes(0)
	node.EXPECT().GetDeletionCost().Return(cost).MinTimes(0)
	node.EXPECT().SetVersion(gomock.Any(), gomock.Any()).MinTimes(0)
	node.EXPECT().SetDependencyReady(gomock.Any()).MinTimes(0)
	node.EXPECT().Stop().MinTimes(0)
	return node
}

func newMockNodeList(t *testing.T, versionReplicas map[string]*VersionStatus) []Node {
	ctl := gomock.NewController(t)
	defer ctl.Finish()
	nodes := make([]Node, 0)
	for version, status := range versionReplicas {
		for i := 0; i < int(status.ReadyReplicas); i++ {
			node := newMockNode(ctl, "", version, true, false, false, 0, 0)
			nodes = append(nodes, node)
		}
		for i := 0; i < int(status.Replicas-status.ReadyReplicas); i++ {
			node := newMockNode(ctl, "", version, false, false, false, 0, 0)
			nodes = append(nodes, node)
		}
	}
	return nodes
}

func TestNodesUpdater_calcTargetMatrix(t *testing.T) {
	fcreateRollingArgs := func(t *testing.T, name string, replicas int32, versionReplicas map[string]*VersionStatus,
		maxUnavaliable, maxSurge, latestVersion string, latestVersionRatio int32, groupVersionMap map[string]string,
		versionHoldMatrix map[string]int32, versionHoldMatrixPercent map[string]float64, paddedLatestVersionRatio int32, strategy CarryStrategyType,
	) *RollingUpdateArgs {
		return &RollingUpdateArgs{
			Nodes:              newMockNodeList(t, versionReplicas),
			Replicas:           &replicas,
			LatestVersionRatio: &latestVersionRatio,
			MaxUnavailable:     getIntstrPtr(maxUnavaliable),
			MaxSurge:           getIntstrPtr(maxSurge),
			LatestVersion:      latestVersion,
			GroupNodeSetArgs: GroupNodeSetArgs{
				LatestVersionCarryStrategy: strategy,
				GroupVersionMap:            groupVersionMap,
				VersionHoldMatrixPercent:   versionHoldMatrixPercent,
				PaddedLatestVersionRatio:   &paddedLatestVersionRatio,
			},
			NodeSetName: name,
		}
	}

	tests := []struct {
		name    string
		args    *RollingUpdateArgs
		want    map[string]int32
		wantErr bool
	}{
		{
			name: "test1",
			args: fcreateRollingArgs(t, "test1", 6, map[string]*VersionStatus{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(3, 3),
				"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 1),
				"457-aa1efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 0),
				"458-bb1efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 0),
			}, "22%", "10%", "458", 25, map[string]string{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
				"455-761efa07c0f73213a3a8704a55e60330": "455",
				"457-aa1efa07c0f73213a3a8704a55e60330": "457",
				"458-bb1efa07c0f73213a3a8704a55e60330": "458",
			}, nil, nil, 34, FloorCarryStrategyType),
			want: map[string]int32{
				"453": 3,
				"455": 1,
				"457": 0,
				"458": 2,
			},
		},
		{
			name: "PaddedLatestVersionTest",
			args: fcreateRollingArgs(t, "test1", 5, map[string]*VersionStatus{
				"457-aa1efa07c0f73213a3a8704a55e60330": newVersionStatus(3, 2),
				"458-bb1efa07c0f73213a3a8704a55e60330": newVersionStatus(2, 0),
			}, "22%", "10%", "458", 80, map[string]string{
				"457-aa1efa07c0f73213a3a8704a55e60330": "457",
				"458-bb1efa07c0f73213a3a8704a55e60330": "458",
			}, nil, nil, 100, CeilCarryStrategyType),
			want: map[string]int32{
				"457": 3,
				"458": 2,
			},
		},
		{
			name: "pingpang",
			args: fcreateRollingArgs(t, "test1", 6, map[string]*VersionStatus{
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
			want: map[string]int32{
				"453": 3,
				"455": 1,
				"457": 0,
				"458": 2,
			},
		},
		{
			name: "pingpang",
			args: fcreateRollingArgs(t, "test1", 6, map[string]*VersionStatus{
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
			args: fcreateRollingArgs(t, "test1", 3, map[string]*VersionStatus{
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
			args: fcreateRollingArgs(t, "test1", 3, map[string]*VersionStatus{
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
			args: fcreateRollingArgs(t, "test1", 3, map[string]*VersionStatus{
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
			args: fcreateRollingArgs(t, "test1", 3, map[string]*VersionStatus{
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
			args: fcreateRollingArgs(t, "test1", 3, map[string]*VersionStatus{
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
			args: fcreateRollingArgs(t, "test1", 3, map[string]*VersionStatus{
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
		// {
		// 	name: "nil",
		// 	args: args{
		// 		shard:  nil,
		// 		params: nil,
		// 	},
		// 	wantErr: true,
		// },
		{
			name: "one version scale",
			args: fcreateRollingArgs(t, "test1", 3, map[string]*VersionStatus{
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
			args: fcreateRollingArgs(t, "test1", 3, map[string]*VersionStatus{
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
			args: fcreateRollingArgs(t, "test1", 10, map[string]*VersionStatus{
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
			args: fcreateRollingArgs(t, "test1", 0, map[string]*VersionStatus{
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
			args: fcreateRollingArgs(t, "test1", 5, map[string]*VersionStatus{
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
			args: fcreateRollingArgs(t, "test1", 5, map[string]*VersionStatus{
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
			args: fcreateRollingArgs(t, "test1", 5, map[string]*VersionStatus{
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
			args: fcreateRollingArgs(t, "test1", 5, map[string]*VersionStatus{
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
			args: fcreateRollingArgs(t, "test1", 5, map[string]*VersionStatus{
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
			args: fcreateRollingArgs(t, "test1", 5, map[string]*VersionStatus{
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
			args: fcreateRollingArgs(t, "test1", 7, map[string]*VersionStatus{
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
			args: fcreateRollingArgs(t, "test1", 7, map[string]*VersionStatus{
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
			args: fcreateRollingArgs(t, "test1", 7, map[string]*VersionStatus{
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
			args: fcreateRollingArgs(t, "test1", 7, map[string]*VersionStatus{
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
			args: fcreateRollingArgs(t, "test1", 17, map[string]*VersionStatus{
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
			args: fcreateRollingArgs(t, "test1", 20, map[string]*VersionStatus{
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
			args: fcreateRollingArgs(t, "test1", 3, map[string]*VersionStatus{
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
			args: fcreateRollingArgs(t, "test1", 3, map[string]*VersionStatus{
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
			args: fcreateRollingArgs(t, "test1", 3, map[string]*VersionStatus{
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
			args: fcreateRollingArgs(t, "test1", 3, map[string]*VersionStatus{
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
			args: fcreateRollingArgs(t, "test1", 3, map[string]*VersionStatus{
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
			args: fcreateRollingArgs(t, "test1", 0, map[string]*VersionStatus{
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
			args: fcreateRollingArgs(t, "test1", 3, map[string]*VersionStatus{
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
			args: fcreateRollingArgs(t, "test1", 3, map[string]*VersionStatus{
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
			args: fcreateRollingArgs(t, "test1", 7, map[string]*VersionStatus{
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
			args: fcreateRollingArgs(t, "test1", 7, map[string]*VersionStatus{
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
			args: fcreateRollingArgs(t, "test1", 7, map[string]*VersionStatus{
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
			args: fcreateRollingArgs(t, "test1", 3, map[string]*VersionStatus{
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
			args: fcreateRollingArgs(t, "test1", 13, map[string]*VersionStatus{
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
			args: fcreateRollingArgs(t, "test1", 2, map[string]*VersionStatus{
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
			args: fcreateRollingArgs(t, "test2", 2, map[string]*VersionStatus{
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
			args: fcreateRollingArgs(t, "test2", 2, map[string]*VersionStatus{
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
			args: fcreateRollingArgs(t, "test1", 5, map[string]*VersionStatus{
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
			args: fcreateRollingArgs(t, "test1", 947, map[string]*VersionStatus{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(381, 341),
				"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(315, 67),
			}, "20%", "10%", "455", 100, map[string]string{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
				"455-761efa07c0f73213a3a8704a55e60330": "455",
			}, nil, nil, 0, CeilCarryStrategyType),

			want: map[string]int32{
				"455": 507,
				"453": 440,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &NodeSetUpdater{
				shardScheduler: &InPlaceShardScheduler{
					debug: true,
				},
				debug: true,
			}
			if got, _, _ := n.calcTargetMatrix(tt.args); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NodesUpdater.calcTargetMatrix() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRollingUpdateArgs_GetVersionStatus(t *testing.T) {
	type fields struct {
		Nodes                      []Node
		CreateNode                 func() Node
		Replicas                   *int32
		LatestVersionRatio         *int32
		MaxUnavailable             *intstr.IntOrString
		MaxSurge                   *intstr.IntOrString
		LatestVersion              string
		LatestVersionPlan          interface{}
		LatestVersionCarryStrategy CarryStrategyType
		GroupVersionMap            map[string]string
		VersionHoldMatrix          map[string]int32
		VersionHoldMatrixPercent   map[string]float64
		PaddedLatestVersionRatio   *int32
	}
	tests := []struct {
		name   string
		fields fields
		want   map[string]*VersionStatus
	}{
		{
			name: "test1",
			fields: fields{
				Nodes: newMockNodeList(t, map[string]*VersionStatus{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(3, 3),
					"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 1),
					"457-aa1efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 0),
					"458-bb1efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 0),
				}),
			},
			want: map[string]*VersionStatus{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(3, 3),
				"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 1),
				"457-aa1efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 0),
				"458-bb1efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 0),
			},
		},
		{
			name: "test2",
			fields: fields{
				Nodes: newMockNodeList(t, map[string]*VersionStatus{
					"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(3, 3),
					"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 1),
					"458-bb1efa07c0f73213a3a8704a55e60330": newVersionStatus(2, 0),
				}),
			},
			want: map[string]*VersionStatus{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": newVersionStatus(3, 3),
				"455-761efa07c0f73213a3a8704a55e60330": newVersionStatus(1, 1),
				"458-bb1efa07c0f73213a3a8704a55e60330": newVersionStatus(2, 0),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := &RollingUpdateArgs{
				Nodes:              tt.fields.Nodes,
				CreateNode:         tt.fields.CreateNode,
				Replicas:           tt.fields.Replicas,
				LatestVersionRatio: tt.fields.LatestVersionRatio,
				MaxUnavailable:     tt.fields.MaxUnavailable,
				MaxSurge:           tt.fields.MaxSurge,
				LatestVersion:      tt.fields.LatestVersion,
				LatestVersionPlan:  tt.fields.LatestVersionPlan,
			}
			got := args.GetVersionStatus()
			if len(got) != len(tt.want) {
				t.Errorf("RollingUpdateArgs.GetVersionStatus() = %v, want %v", got, tt.want)
			}
			for version, status := range got {
				wantstatus := tt.want[version]
				if status.Replicas != wantstatus.Replicas || status.ReadyReplicas != status.ReadyReplicas {
					t.Errorf("RollingUpdateArgs.GetVersionStatus() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestNodeSetUpdater_rollingUpdateNodeSet(t *testing.T) {
	ctl := gomock.NewController(t)
	defer ctl.Finish()
	type args struct {
		versionNodeMap map[string]NodeSet
		TargetMatrix   map[string]int32
	}
	fcreateNodeSet := func(all, ready, empty, release int, version string) NodeSet {
		nodes := make(NodeSet, 0)
		for i := 0; i < ready; i++ {
			score := 1000 + i
			name := fmt.Sprintf("R-%s-%d", version, score)
			node := newMockNode(ctl, name, version, true, false, false, score, 0)
			nodes = append(nodes, node)
		}
		for i := 0; i < empty; i++ {
			score := 10 + i
			name := fmt.Sprintf("E-%s-%d", version, score)
			node := newMockNode(ctl, name, version, false, true, false, score, 0)
			nodes = append(nodes, node)
		}
		for i := 0; i < release; i++ {
			name := fmt.Sprintf("D-%s", version)
			node := newMockNode(ctl, name, version, false, false, true, 0, 0)
			nodes = append(nodes, node)
		}
		all -= (ready + empty + release)
		for i := 0; i < all; i++ {
			score := 100 + i
			name := fmt.Sprintf("O-%s-%d", version, score)
			node := newMockNode(ctl, name, version, false, false, false, score, 0)
			nodes = append(nodes, node)
		}
		sort.Stable(sort.Reverse(nodes))
		return nodes
	}
	tests := []struct {
		name  string
		args  args
		want  map[string]int
		want1 map[string][]string
		want2 []string
	}{
		{
			name: "扩容1",
			args: args{
				versionNodeMap: map[string]NodeSet{
					"453": fcreateNodeSet(3, 3, 0, 0, "453"),
				},
				TargetMatrix: map[string]int32{
					"453": 4,
				},
			},
			want: map[string]int{
				"453": 1,
			},
			want1: map[string][]string{},
			want2: []string{},
		},
		{
			name: "扩容2",
			args: args{
				versionNodeMap: map[string]NodeSet{
					"453": fcreateNodeSet(3, 3, 0, 0, "453"),
					"455": fcreateNodeSet(2, 2, 0, 0, "455"),
				},
				TargetMatrix: map[string]int32{
					"453": 3,
					"455": 3,
				},
			},
			want: map[string]int{
				"455": 1,
			},
			want1: map[string][]string{},
			want2: []string{},
		},
		{
			name: "缩容1",
			args: args{
				versionNodeMap: map[string]NodeSet{
					"453": fcreateNodeSet(3, 3, 0, 0, "453"),
				},
				TargetMatrix: map[string]int32{
					"453": 2,
				},
			},
			want:  map[string]int{},
			want1: map[string][]string{},
			want2: []string{"R-453-1000"},
		},
		{
			name: "缩容2",
			args: args{
				versionNodeMap: map[string]NodeSet{
					"455": fcreateNodeSet(1, 1, 0, 0, "455"),
					"453": fcreateNodeSet(3, 3, 0, 0, "453"),
				},
				TargetMatrix: map[string]int32{
					"455": 1,
					"453": 2,
				},
			},
			want:  map[string]int{},
			want1: map[string][]string{},
			want2: []string{"R-453-1000"},
		},
		{
			name: "rolling1",
			args: args{
				versionNodeMap: map[string]NodeSet{
					"453": fcreateNodeSet(3, 3, 0, 0, "453"),
				},
				TargetMatrix: map[string]int32{
					"455": 1,
					"453": 2,
				},
			},
			want: map[string]int{},
			want1: map[string][]string{
				"455": {"R-453-1000"},
			},
			want2: []string{},
		},
		{
			name: "rolling2",
			args: args{
				versionNodeMap: map[string]NodeSet{
					"453": fcreateNodeSet(2, 2, 0, 0, "453"),
					"455": fcreateNodeSet(1, 1, 0, 0, "455"),
				},
				TargetMatrix: map[string]int32{
					"455": 2,
					"453": 1,
				},
			},
			want: map[string]int{},
			want1: map[string][]string{
				"455": {"R-453-1000"},
			},
			want2: []string{},
		},
		{
			name: "rolling3",
			args: args{
				versionNodeMap: map[string]NodeSet{
					"453": fcreateNodeSet(1, 1, 0, 0, "453"),
					"455": fcreateNodeSet(2, 2, 0, 0, "455"),
				},
				TargetMatrix: map[string]int32{
					"455": 1,
					"453": 1,
					"457": 1,
				},
			},
			want: map[string]int{},
			want1: map[string][]string{
				"457": {"R-455-1000"},
			},
			want2: []string{},
		},
		{
			name: "empty1",
			args: args{
				versionNodeMap: map[string]NodeSet{
					"453": fcreateNodeSet(2, 2, 0, 0, "453"),
					"455": fcreateNodeSet(2, 0, 2, 0, "455"),
					"457": fcreateNodeSet(2, 2, 0, 0, "457"),
				},
				TargetMatrix: map[string]int32{
					"455": 2,
					"453": 2,
					"457": 0,
				},
			},
			want: map[string]int{},
			want1: map[string][]string{
				"455": {"R-457-1001", "R-457-1000"},
			},
			want2: []string{"E-455-10", "E-455-11"},
		},
		{
			name: "empty2",
			args: args{
				versionNodeMap: map[string]NodeSet{
					"453": fcreateNodeSet(2, 2, 0, 0, "453"),
					"455": fcreateNodeSet(2, 1, 1, 0, "455"),
					"457": fcreateNodeSet(2, 2, 0, 0, "457"),
				},
				TargetMatrix: map[string]int32{
					"455": 2,
					"453": 2,
					"457": 0,
				},
			},
			want: map[string]int{},
			want1: map[string][]string{
				"455": {"R-457-1001"},
			},
			want2: []string{"E-455-10", "R-457-1000"},
		},
		{
			name: "empty3",
			args: args{
				versionNodeMap: map[string]NodeSet{
					"453": fcreateNodeSet(2, 1, 1, 0, "453"),
					"455": fcreateNodeSet(2, 1, 1, 0, "455"),
					"457": fcreateNodeSet(2, 2, 0, 0, "457"),
				},
				TargetMatrix: map[string]int32{
					"453": 2,
					"455": 2,
					"457": 0,
				},
			},
			want: map[string]int{},
			want1: map[string][]string{
				"453": {"R-457-1000"},
				"455": {"R-457-1001"},
			},
			want2: []string{"E-455-10", "E-453-10"},
		},
		{
			name: "score1",
			args: args{
				versionNodeMap: map[string]NodeSet{
					"453": fcreateNodeSet(3, 3, 0, 0, "453"),
				},
				TargetMatrix: map[string]int32{
					"453": 2,
					"455": 1,
				},
			},
			want: map[string]int{},
			want1: map[string][]string{
				"455": {"R-453-1000"},
			},
			want2: []string{},
		},
		{
			name: "score2",
			args: args{
				versionNodeMap: map[string]NodeSet{
					"453": fcreateNodeSet(3, 2, 1, 0, "453"),
					"455": fcreateNodeSet(1, 1, 0, 0, "455"),
					"457": fcreateNodeSet(2, 1, 1, 0, "457"),
					"458": fcreateNodeSet(1, 0, 1, 0, "458"),
				},
				TargetMatrix: map[string]int32{
					"453": 2,
					"455": 1,
					"457": 2,
					"458": 2,
				},
			},
			want: map[string]int{},
			want1: map[string][]string{
				"458": {"E-453-10"},
			},
			want2: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &NodeSetUpdater{
				debug: true,
			}
			args := &RollingUpdateArgs{}
			got, got1, got2 := n.rollingUpdateNodeSet(tt.args.versionNodeMap, tt.args.TargetMatrix, args)
			strgot1 := make(map[string][]string)
			strgot2 := make([]string, 0)
			for version, nodes := range got1 {
				for _, n := range nodes {
					strgot1[version] = append(strgot1[version], n.GetName())
				}
			}
			for _, n := range got2 {
				strgot2 = append(strgot2, n.GetName())
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NodeSetUpdater.NodeSetVersionUpdate() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(strgot1, tt.want1) {
				t.Errorf("NodeSetUpdater.NodeSetVersionUpdate() got1 = %v, want %v", strgot1, tt.want1)
			}
			if !reflect.DeepEqual(strgot2, tt.want2) {
				t.Errorf("NodeSetUpdater.NodeSetVersionUpdate() got2 = %v, want %v", strgot2, tt.want2)
			}
		})
	}
}

func TestNodeSetUpdater_Schedule(t *testing.T) {
	ctl := gomock.NewController(t)
	defer ctl.Finish()
	fcreateNodeSet := func(info map[string][]int) NodeSet {
		nodes := make(NodeSet, 0)
		for version, status := range info {
			all, ready, empty, release := status[0], status[1], status[2], status[3]
			for i := 0; i < ready; i++ {
				score := 1000 + i
				name := fmt.Sprintf("R-%s-%d", version, score)
				node := newMockNode(ctl, name, version, true, false, false, score, 0)
				nodes = append(nodes, node)
			}
			for i := 0; i < empty; i++ {
				score := 10 + i
				name := fmt.Sprintf("E-%s-%d", version, score)
				node := newMockNode(ctl, name, version, false, true, false, score, 0)
				nodes = append(nodes, node)
			}
			for i := 0; i < release; i++ {
				name := fmt.Sprintf("D-%s", version)
				node := newMockNode(ctl, name, version, false, false, true, 0, 0)
				nodes = append(nodes, node)
			}
			all -= (ready + empty + release)
			for i := 0; i < all; i++ {
				score := 100 + i
				name := fmt.Sprintf("O-%s-%d", version, score)
				node := newMockNode(ctl, name, version, false, false, false, score, 0)
				nodes = append(nodes, node)
			}
		}
		sort.Stable(sort.Reverse(nodes))
		return nodes
	}
	fcreateRollingArgs := func(nodes map[string][]int, replicas int32, latestVersionRatio int32, release bool,
		maxUnavaliable, maxSurge, latestVersion, rollingStrategy string) *RollingUpdateArgs {
		fcreateNode := func() Node {
			return newMockNode(ctl, "N", "init", false, false, false, 0, 0)
		}
		args := &RollingUpdateArgs{
			Nodes:              fcreateNodeSet(nodes),
			CreateNode:         fcreateNode,
			Replicas:           &replicas,
			LatestVersionRatio: &latestVersionRatio,
			MaxUnavailable:     getIntstrPtr(maxUnavaliable),
			MaxSurge:           getIntstrPtr(maxSurge),
			LatestVersion:      latestVersion,
			RollingStrategy:    rollingStrategy,
			Release:            release,
		}
		return args
	}
	fcreateGroupArgs := func(groupVersionMap map[string]string, holdMatrixPercent map[string]float64, paddedLatestVersionRatio int32,
		dependencyMatrixPercent map[string]float64, latestVersionCarryStrategy CarryStrategyType) *GroupNodeSetArgs {
		return &GroupNodeSetArgs{
			GroupVersionMap:                groupVersionMap,
			VersionHoldMatrixPercent:       holdMatrixPercent,
			PaddedLatestVersionRatio:       &paddedLatestVersionRatio,
			VersionDependencyMatrixPercent: dependencyMatrixPercent,
			LatestVersionCarryStrategy:     latestVersionCarryStrategy,
		}
	}

	tests := []struct {
		name      string
		args      *RollingUpdateArgs
		groupArgs *GroupNodeSetArgs
		want      map[string]int
		want1     map[string][]string
		want2     []string
		wantErr   bool
	}{
		{
			name: "expand1",
			args: fcreateRollingArgs(map[string][]int{
				"453": {3, 3, 0, 0},
			}, 4, 100, false, "20%", "10%", "453", RollingStrategyOldFirst),
			want: map[string]int{
				"453": 1,
			},
			want1:   map[string][]string{},
			want2:   []string{},
			wantErr: false,
		},
		{
			name: "expand2",
			args: fcreateRollingArgs(map[string][]int{
				"453": {3, 2, 1, 0},
			}, 4, 100, false, "20%", "10%", "453", RollingStrategyOldFirst),
			want: map[string]int{
				"453": 1,
			},
			want1:   map[string][]string{},
			want2:   []string{},
			wantErr: false,
		},
		{
			name: "expand3",
			args: fcreateRollingArgs(map[string][]int{
				"453": {3, 2, 0, 1},
			}, 4, 100, false, "20%", "10%", "453", RollingStrategyOldFirst),
			want: map[string]int{
				"453": 2,
			},
			want1:   map[string][]string{},
			want2:   []string{"D-453"},
			wantErr: false,
		},
		{
			name: "expand4",
			args: fcreateRollingArgs(map[string][]int{
				"453": {3, 2, 0, 0},
			}, 4, 100, false, "20%", "10%", "453", RollingStrategyOldFirst),
			want: map[string]int{
				"453": 1,
			},
			want1:   map[string][]string{},
			want2:   []string{},
			wantErr: false,
		},
		{
			name: "reduce1",
			args: fcreateRollingArgs(map[string][]int{
				"453": {5, 5, 0, 0},
			}, 4, 100, false, "20%", "10%", "453", RollingStrategyOldFirst),
			want:    map[string]int{},
			want1:   map[string][]string{},
			want2:   []string{"R-453-1000"},
			wantErr: false,
		},
		{
			name: "reduce2",
			args: fcreateRollingArgs(map[string][]int{
				"453": {5, 4, 1, 0},
			}, 4, 100, false, "20%", "10%", "453", RollingStrategyOldFirst),
			want:    map[string]int{},
			want1:   map[string][]string{},
			want2:   []string{"E-453-10"},
			wantErr: false,
		},
		{
			name: "reduce3",
			args: fcreateRollingArgs(map[string][]int{
				"453": {5, 4, 0, 0},
			}, 4, 100, false, "20%", "10%", "453", RollingStrategyOldFirst),
			want:    map[string]int{},
			want1:   map[string][]string{},
			want2:   []string{"O-453-100"},
			wantErr: false,
		},
		{
			name: "reduce4",
			args: fcreateRollingArgs(map[string][]int{
				"453": {5, 4, 0, 1},
			}, 4, 100, false, "20%", "10%", "453", RollingStrategyOldFirst),
			want:    map[string]int{},
			want1:   map[string][]string{},
			want2:   []string{"D-453"},
			wantErr: false,
		},
		{
			name: "reduce5",
			args: fcreateRollingArgs(map[string][]int{
				"453": {5, 3, 1, 1},
			}, 4, 100, false, "20%", "10%", "453", RollingStrategyOldFirst),
			want:    map[string]int{},
			want1:   map[string][]string{},
			want2:   []string{"D-453"},
			wantErr: false,
		},
		{
			name: "reduce6",
			args: fcreateRollingArgs(map[string][]int{
				"453": {5, 3, 0, 1},
			}, 4, 100, false, "20%", "10%", "453", RollingStrategyOldFirst),
			want:    map[string]int{},
			want1:   map[string][]string{},
			want2:   []string{"D-453"},
			wantErr: false,
		},
		{
			name: "reduce7",
			args: fcreateRollingArgs(map[string][]int{
				"453": {5, 3, 1, 0},
			}, 4, 100, false, "20%", "10%", "453", RollingStrategyOldFirst),
			want:    map[string]int{},
			want1:   map[string][]string{},
			want2:   []string{"E-453-10"},
			wantErr: false,
		},
		{
			name: "reduce8",
			args: fcreateRollingArgs(map[string][]int{
				"453": {5, 2, 2, 0},
			}, 4, 100, false, "20%", "10%", "453", RollingStrategyOldFirst),
			want:    map[string]int{},
			want1:   map[string][]string{},
			want2:   []string{"E-453-10"},
			wantErr: false,
		},
		{
			name: "reduce9",
			args: fcreateRollingArgs(map[string][]int{
				"453": {5, 5, 0, 0},
			}, 3, 100, false, "20%", "10%", "453", RollingStrategyOldFirst),
			want:    map[string]int{},
			want1:   map[string][]string{},
			want2:   []string{"R-453-1001", "R-453-1000"},
			wantErr: false,
		},
		{
			name: "update1",
			args: fcreateRollingArgs(map[string][]int{
				"453": {3, 3, 0, 0},
			}, 3, 100, false, "20%", "10%", "455", RollingStrategyOldFirst),
			want: map[string]int{
				"455": 1,
			},
			want1:   map[string][]string{},
			want2:   []string{},
			wantErr: false,
		},
		{
			name: "update2",
			args: fcreateRollingArgs(map[string][]int{
				"453": {5, 5, 0, 0},
			}, 5, 100, false, "20%", "10%", "455", RollingStrategyOldFirst),
			want: map[string]int{},
			want1: map[string][]string{
				"455": {"R-453-1000"},
			},
			want2:   []string{},
			wantErr: false,
		},
		{
			name: "update3",
			args: fcreateRollingArgs(map[string][]int{
				"453": {5, 4, 0, 0},
			}, 5, 100, false, "20%", "10%", "455", RollingStrategyOldFirst),
			want: map[string]int{},
			want1: map[string][]string{
				"455": {"O-453-100"},
			},
			want2:   []string{},
			wantErr: false,
		},
		{
			name: "update4",
			args: fcreateRollingArgs(map[string][]int{
				"453": {5, 4, 1, 0},
			}, 5, 100, false, "20%", "10%", "455", RollingStrategyOldFirst),
			want: map[string]int{},
			want1: map[string][]string{
				"455": {"E-453-10"},
			},
			want2:   []string{},
			wantErr: false,
		},
		{
			name: "update5",
			args: fcreateRollingArgs(map[string][]int{
				"453": {5, 4, 0, 1},
			}, 5, 100, false, "20%", "10%", "455", RollingStrategyOldFirst),
			want: map[string]int{
				"455": 1,
			},
			want1:   map[string][]string{},
			want2:   []string{"D-453"},
			wantErr: false,
		},
		{
			name: "update6",
			args: fcreateRollingArgs(map[string][]int{
				"453": {3, 3, 0, 0},
				"455": {2, 2, 0, 0},
				"457": {2, 2, 0, 0},
			}, 7, 100, false, "30%", "10%", "457", RollingStrategyOldFirst),
			want: map[string]int{},
			want1: map[string][]string{
				"457": {"R-453-1001", "R-453-1000"},
			},
			want2:   []string{},
			wantErr: false,
		},
		{
			name: "update_reduce1",
			args: fcreateRollingArgs(map[string][]int{
				"453": {3, 3, 0, 0},
				"455": {2, 2, 0, 0},
			}, 4, 100, false, "20%", "10%", "455", RollingStrategyOldFirst),
			want: map[string]int{},
			want1: map[string][]string{
				"455": {"R-453-1000"},
			},
			want2:   []string{},
			wantErr: false,
		},
		{
			name: "update_reduce2",
			args: fcreateRollingArgs(map[string][]int{
				"453": {3, 3, 0, 0},
				"455": {2, 2, 0, 0},
			}, 3, 100, false, "20%", "10%", "455", RollingStrategyOldFirst),
			want: map[string]int{},
			want1: map[string][]string{
				"455": {"R-453-1001"},
			},
			want2:   []string{"R-453-1000"},
			wantErr: false,
		},
		{
			name: "update_reduce3",
			args: fcreateRollingArgs(map[string][]int{
				"453": {3, 1, 1, 0},
				"455": {2, 2, 0, 0},
			}, 3, 100, false, "20%", "10%", "455", RollingStrategyOldFirst),
			want: map[string]int{},
			want1: map[string][]string{
				"455": {"O-453-100"},
			},
			want2:   []string{"E-453-10"},
			wantErr: false,
		},
		{
			name: "update_reduce4",
			args: fcreateRollingArgs(map[string][]int{
				"453": {3, 1, 1, 1},
				"455": {2, 2, 0, 0},
			}, 3, 100, false, "20%", "10%", "455", RollingStrategyOldFirst),
			want: map[string]int{},
			want1: map[string][]string{
				"455": {"E-453-10"},
			},
			want2:   []string{"D-453"},
			wantErr: false,
		},
		{
			name: "update_reduce5",
			args: fcreateRollingArgs(map[string][]int{
				"453": {4, 4, 0, 0},
				"455": {1, 0, 0, 0},
			}, 3, 100, false, "20%", "10%", "455", RollingStrategyOldFirst),
			want:    map[string]int{},
			want1:   map[string][]string{},
			want2:   []string{"R-453-1000"},
			wantErr: false,
		},
		{
			name: "update_reduce6",
			args: fcreateRollingArgs(map[string][]int{
				"453": {4, 4, 0, 0},
				"455": {1, 0, 1, 0},
			}, 3, 100, false, "20%", "10%", "455", RollingStrategyOldFirst),
			want: map[string]int{},
			want1: map[string][]string{
				"455": {"R-453-1000"},
			},
			want2:   []string{"E-455-10"},
			wantErr: false,
		},
		{
			name: "update_expand1",
			args: fcreateRollingArgs(map[string][]int{
				"453": {3, 3, 0, 0},
				"455": {2, 2, 0, 0},
			}, 6, 100, false, "20%", "10%", "455", RollingStrategyOldFirst),
			want: map[string]int{
				"455": 1,
			},
			want1:   map[string][]string{},
			want2:   []string{},
			wantErr: false,
		},
		{
			name: "update_expand2",
			args: fcreateRollingArgs(map[string][]int{
				"453": {3, 3, 0, 0},
				"455": {2, 2, 0, 0},
			}, 6, 100, false, "40%", "10%", "455", RollingStrategyOldFirst),
			want: map[string]int{
				"455": 1,
			},
			want1: map[string][]string{
				"455": {"R-453-1000"},
			},
			want2:   []string{},
			wantErr: false,
		},
		{
			name: "update_expand3",
			args: fcreateRollingArgs(map[string][]int{
				"453": {3, 3, 0, 0},
				"455": {2, 1, 0, 0},
			}, 6, 100, false, "20%", "10%", "455", RollingStrategyOldFirst),
			want: map[string]int{
				"455": 1,
			},
			want1:   map[string][]string{},
			want2:   []string{},
			wantErr: false,
		},
		{
			name: "update_expand4",
			args: fcreateRollingArgs(map[string][]int{
				"453": {3, 3, 0, 0},
				"455": {2, 1, 1, 0},
			}, 6, 100, false, "20%", "10%", "455", RollingStrategyOldFirst),
			want: map[string]int{
				"455": 1,
			},
			want1:   map[string][]string{},
			want2:   []string{},
			wantErr: false,
		},
		{
			name: "group1",
			args: fcreateRollingArgs(map[string][]int{
				"453": {3, 3, 0, 0},
				"455": {1, 1, 0, 0},
				"457": {1, 0, 0, 0},
				"458": {1, 0, 0, 0},
			}, 6, 25, false, "25%", "10%", "458", RollingStrategyOldFirst),
			groupArgs: fcreateGroupArgs(map[string]string{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
				"455-761efa07c0f73213a3a8704a55e60330": "455",
				"457-aa1efa07c0f73213a3a8704a55e60330": "457",
				"458-bb1efa07c0f73213a3a8704a55e60330": "458",
			}, map[string]float64{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": 50,
				"455-761efa07c0f73213a3a8704a55e60330": 16,
				"457-aa1efa07c0f73213a3a8704a55e60330": 0,
				"458-bb1efa07c0f73213a3a8704a55e60330": 25,
			}, 34, nil, FloorCarryStrategyType),
			want: map[string]int{},
			want1: map[string][]string{
				"458": {"O-457-100"},
			},
			want2:   []string{},
			wantErr: false,
		},
		{
			name: "group2",
			args: fcreateRollingArgs(map[string][]int{
				"453": {3, 3, 0, 0},
				"455": {1, 1, 0, 0},
				"458": {2, 0, 0, 0},
			}, 6, 25, false, "25%", "10%", "458", RollingStrategyOldFirst),
			groupArgs: fcreateGroupArgs(map[string]string{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
				"455-761efa07c0f73213a3a8704a55e60330": "455",
				"458-bb1efa07c0f73213a3a8704a55e60330": "458",
			}, map[string]float64{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": 50,
				"455-761efa07c0f73213a3a8704a55e60330": 16,
				"457-aa1efa07c0f73213a3a8704a55e60330": 0,
				"458-bb1efa07c0f73213a3a8704a55e60330": 25,
			}, 34, nil, FloorCarryStrategyType),
			want:    map[string]int{},
			want1:   map[string][]string{},
			want2:   []string{},
			wantErr: false,
		},
		{
			name: "group3",
			args: fcreateRollingArgs(map[string][]int{
				"453": {1, 1, 0, 0},
				"455": {3, 2, 1, 0},
				"458": {2, 2, 0, 0},
			}, 5, 40, false, "25%", "10%", "458", RollingStrategyOldFirst),
			groupArgs: fcreateGroupArgs(map[string]string{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": "453",
				"455-761efa07c0f73213a3a8704a55e60330": "455",
				"458-bb1efa07c0f73213a3a8704a55e60330": "458",
			}, map[string]float64{
				"453-2c0bc462f0e41079ffc6bc3307c1f0e4": 0,
				"455-761efa07c0f73213a3a8704a55e60330": 60,
				"458-bb1efa07c0f73213a3a8704a55e60330": 40,
			}, 0, nil, ""),
			want: map[string]int{},
			want1: map[string][]string{
				"455": {"R-453-1000"},
			},
			want2:   []string{"E-455-10"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &NodeSetUpdater{
				shardScheduler: &InPlaceShardScheduler{
					debug: true,
				},
				debug: true,
			}
			if tt.groupArgs != nil {
				tt.args.GroupVersionMap = tt.groupArgs.GroupVersionMap
				tt.args.VersionHoldMatrixPercent = tt.groupArgs.VersionHoldMatrixPercent
				tt.args.PaddedLatestVersionRatio = tt.groupArgs.PaddedLatestVersionRatio
				tt.args.VersionDependencyMatrixPercent = tt.groupArgs.VersionDependencyMatrixPercent
				tt.args.LatestVersionCarryStrategy = tt.groupArgs.LatestVersionCarryStrategy
			}
			got, got1, got2, err := n.Schedule(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("NodeSetUpdater.Schedule() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			intgot := make(map[string]int)
			strgot1 := make(map[string][]string)
			strgot2 := make([]string, 0)
			for version, nodes := range got {
				intgot[version] = len(nodes)
			}
			for version, nodes := range got1 {
				for i := range nodes {
					strgot1[version] = append(strgot1[version], nodes[i].GetName())
				}
			}
			for i := range got2 {
				strgot2 = append(strgot2, got2[i].GetName())
			}
			if !reflect.DeepEqual(intgot, tt.want) {
				t.Errorf("NodeSetUpdater.Schedule() got = %v, want %v", intgot, tt.want)
			}
			if !reflect.DeepEqual(strgot1, tt.want1) {
				t.Errorf("NodeSetUpdater.Schedule() got1 = %v, want %v", strgot1, tt.want1)
			}
			if !reflect.DeepEqual(strgot2, tt.want2) {
				t.Errorf("NodeSetUpdater.Schedule() got2 = %v, want %v", strgot2, tt.want2)
			}
		})
	}
}
