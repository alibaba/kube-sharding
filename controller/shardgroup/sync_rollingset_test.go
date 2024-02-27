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

package shardgroup

import (
	"testing"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	"github.com/alibaba/kube-sharding/pkg/rollalgorithm"
)

func Test_isRollingsetNotComplete(t *testing.T) {
	type args struct {
		shardGroup        *carbonv1.ShardGroup
		rollingSet        *carbonv1.RollingSet
		shardGroupVersion string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "completed",
			args: args{
				shardGroup: &carbonv1.ShardGroup{
					Spec: carbonv1.ShardGroupSpec{
						LatestPercent: utils.Int32Ptr(100),
					},
				},
				rollingSet: &carbonv1.RollingSet{
					Spec: carbonv1.RollingSetSpec{
						Version: "role-1111",
						VersionPlan: carbonv1.VersionPlan{
							SignedVersionPlan: carbonv1.SignedVersionPlan{
								ShardGroupVersion: "group-1111",
							},
						},
						SchedulePlan: rollalgorithm.SchedulePlan{
							Replicas:           utils.Int32Ptr(5),
							LatestVersionRatio: utils.Int32Ptr(100),
						},
					},
					Status: carbonv1.RollingSetStatus{
						Version:  "role-1111",
						Complete: true,
						GroupVersionStatusMap: map[string]*rollalgorithm.VersionStatus{
							"group-1111": {
								Replicas:      5,
								ReadyReplicas: 5,
							},
						},
					},
				},
				shardGroupVersion: "group-1111",
			},
			want: false,
		},
		{
			name: "completed",
			args: args{
				shardGroup: &carbonv1.ShardGroup{
					Spec: carbonv1.ShardGroupSpec{
						LatestPercent: utils.Int32Ptr(0),
					},
				},
				rollingSet: &carbonv1.RollingSet{
					Spec: carbonv1.RollingSetSpec{
						Version: "role-1111",
						VersionPlan: carbonv1.VersionPlan{
							SignedVersionPlan: carbonv1.SignedVersionPlan{
								ShardGroupVersion: "group-1111",
							},
						},
						SchedulePlan: rollalgorithm.SchedulePlan{
							Replicas:           utils.Int32Ptr(5),
							LatestVersionRatio: utils.Int32Ptr(0),
						},
					},
					Status: carbonv1.RollingSetStatus{
						Version:  "role-1111",
						Complete: true,
						GroupVersionStatusMap: map[string]*rollalgorithm.VersionStatus{
							"group-1111": {
								Replicas:      5,
								ReadyReplicas: 5,
							},
						},
					},
				},
				shardGroupVersion: "group-1111",
			},
			want: false,
		},
		{
			name: "not completed latestpercent",
			args: args{
				shardGroup: &carbonv1.ShardGroup{
					Spec: carbonv1.ShardGroupSpec{
						LatestPercent: utils.Int32Ptr(50),
					},
				},
				rollingSet: &carbonv1.RollingSet{
					Spec: carbonv1.RollingSetSpec{
						Version: "role-1111",
						VersionPlan: carbonv1.VersionPlan{
							SignedVersionPlan: carbonv1.SignedVersionPlan{
								ShardGroupVersion: "group-1111",
							},
						},
						SchedulePlan: rollalgorithm.SchedulePlan{
							Replicas:           utils.Int32Ptr(5),
							LatestVersionRatio: utils.Int32Ptr(50),
						},
					},
					Status: carbonv1.RollingSetStatus{
						Version:  "role-1111",
						Complete: true,
						GroupVersionStatusMap: map[string]*rollalgorithm.VersionStatus{
							"group-1111": {
								Replicas:      5,
								ReadyReplicas: 5,
							},
						},
					},
				},
				shardGroupVersion: "group-1111",
			},
			want: true,
		},
		{
			name: "not completed groupVerson",
			args: args{
				shardGroup: &carbonv1.ShardGroup{
					Spec: carbonv1.ShardGroupSpec{
						LatestPercent: utils.Int32Ptr(50),
					},
				},
				rollingSet: &carbonv1.RollingSet{
					Spec: carbonv1.RollingSetSpec{
						Version: "role-1111",
						VersionPlan: carbonv1.VersionPlan{
							SignedVersionPlan: carbonv1.SignedVersionPlan{
								ShardGroupVersion: "group-1111",
							},
						},
						SchedulePlan: rollalgorithm.SchedulePlan{
							Replicas:           utils.Int32Ptr(5),
							LatestVersionRatio: utils.Int32Ptr(50),
						},
					},
					Status: carbonv1.RollingSetStatus{
						Version:  "role-1111",
						Complete: true,
						GroupVersionStatusMap: map[string]*rollalgorithm.VersionStatus{
							"group-1111": {
								Replicas:      5,
								ReadyReplicas: 4,
							},
						},
					},
				},
				shardGroupVersion: "group-1111",
			},
			want: true,
		},
		{
			name: "not completed groupVerson",
			args: args{
				shardGroup: &carbonv1.ShardGroup{
					Spec: carbonv1.ShardGroupSpec{
						LatestPercent: utils.Int32Ptr(50),
					},
				},
				rollingSet: &carbonv1.RollingSet{
					Spec: carbonv1.RollingSetSpec{
						Version: "role-1111",
						VersionPlan: carbonv1.VersionPlan{
							SignedVersionPlan: carbonv1.SignedVersionPlan{
								ShardGroupVersion: "group-1111",
							},
						},
						SchedulePlan: rollalgorithm.SchedulePlan{
							Replicas:           utils.Int32Ptr(5),
							LatestVersionRatio: utils.Int32Ptr(50),
						},
					},
					Status: carbonv1.RollingSetStatus{
						Version:  "role-1111",
						Complete: true,
						GroupVersionStatusMap: map[string]*rollalgorithm.VersionStatus{
							"group-1111": {
								Replicas:      5,
								ReadyReplicas: 4,
							},
						},
					},
				},
				shardGroupVersion: "group-1112",
			},
			want: true,
		},
		{
			name: "not completed roleVerson",
			args: args{
				shardGroup: &carbonv1.ShardGroup{
					Spec: carbonv1.ShardGroupSpec{
						LatestPercent: utils.Int32Ptr(50),
					},
				},
				rollingSet: &carbonv1.RollingSet{
					Spec: carbonv1.RollingSetSpec{
						Version: "role-1111",
						VersionPlan: carbonv1.VersionPlan{
							SignedVersionPlan: carbonv1.SignedVersionPlan{
								ShardGroupVersion: "group-1111",
							},
						},
						SchedulePlan: rollalgorithm.SchedulePlan{
							Replicas:           utils.Int32Ptr(5),
							LatestVersionRatio: utils.Int32Ptr(50),
						},
					},
					Status: carbonv1.RollingSetStatus{
						Version:  "role-1112",
						Complete: true,
						GroupVersionStatusMap: map[string]*rollalgorithm.VersionStatus{
							"group-1111": {
								Replicas:      5,
								ReadyReplicas: 4,
							},
						},
					},
				},
				shardGroupVersion: "group-1111",
			},
			want: true,
		},
		{
			name: "not completed roleVerson",
			args: args{
				shardGroup: &carbonv1.ShardGroup{
					Spec: carbonv1.ShardGroupSpec{},
				},
				rollingSet: &carbonv1.RollingSet{
					Spec: carbonv1.RollingSetSpec{
						Version: "role-1111",
						VersionPlan: carbonv1.VersionPlan{
							SignedVersionPlan: carbonv1.SignedVersionPlan{
								ShardGroupVersion: "group-1111",
							},
						},
						SchedulePlan: rollalgorithm.SchedulePlan{},
					},
					Status: carbonv1.RollingSetStatus{
						Version:  "role-1112",
						Complete: true,
					},
				},
				shardGroupVersion: "group-1111",
			},
			want: true,
		},
		{
			name: "completed standby",
			args: args{
				shardGroup: &carbonv1.ShardGroup{
					Spec: carbonv1.ShardGroupSpec{
						LatestPercent: utils.Int32Ptr(100),
					},
				},
				rollingSet: &carbonv1.RollingSet{
					Spec: carbonv1.RollingSetSpec{
						Version: "role-1111",
						VersionPlan: carbonv1.VersionPlan{
							SignedVersionPlan: carbonv1.SignedVersionPlan{
								ShardGroupVersion: "group-1111",
							},
						},
						SchedulePlan: rollalgorithm.SchedulePlan{
							Replicas:           utils.Int32Ptr(5),
							LatestVersionRatio: utils.Int32Ptr(100),
						},
						Capacity: carbonv1.Capacity{
							ActivePlan: carbonv1.ActivePlan{
								ScaleSchedulePlan: &carbonv1.ScaleSchedulePlan{
									Replicas: utils.Int32Ptr(3),
								},
								ScaleConfig: &carbonv1.ScaleConfig{Enable: true},
							},
						},
					},
					Status: carbonv1.RollingSetStatus{
						Version:  "role-1111",
						Complete: true,
						GroupVersionStatusMap: map[string]*rollalgorithm.VersionStatus{
							"group-1111": {
								Replicas:      3,
								ReadyReplicas: 3,
							},
						},
					},
				},
				shardGroupVersion: "group-1111",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRollingsetNotComplete(tt.args.shardGroup, tt.args.rollingSet, tt.args.shardGroupVersion); got != tt.want {
				t.Errorf("test[%v] isRollingsetNotComplete() = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}
