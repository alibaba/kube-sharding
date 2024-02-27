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
	"reflect"
	"testing"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	"github.com/alibaba/kube-sharding/pkg/rollalgorithm"
	apps "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func getIntstrPtr(src string) *intstr.IntOrString {
	v := intstr.FromString(src)
	return &v
}
func Test_initGroupScheduleParams(t *testing.T) {
	type args struct {
		shardGroup        *carbonv1.ShardGroup
		roles             []*carbonv1.RollingSet
		shardGroupVersion string
	}
	tests := []struct {
		name    string
		args    args
		want    *rollalgorithm.GroupScheduleParams
		wantErr bool
	}{
		{
			name: "test",
			args: args{
				shardGroup: &carbonv1.ShardGroup{
					ObjectMeta: metav1.ObjectMeta{
						Name: "group",
						Labels: map[string]string{
							"app.hippo.io/app-name":   "app",
							"app.hippo.io/group-name": "group",
						},
					},
					Spec: carbonv1.ShardGroupSpec{
						LatestPercent:  utils.Int32Ptr(100),
						MaxUnavailable: getIntstrPtr("10"),
						ShardTemplates: map[string]carbonv1.ShardTemplate{
							"role-1": carbonv1.ShardTemplate{
								Spec: carbonv1.ShardSpec{
									Replicas: utils.Int32Ptr(10),
								},
							},
							"role-2": carbonv1.ShardTemplate{
								Spec: carbonv1.ShardSpec{
									Replicas: utils.Int32Ptr(10),
								},
							},
						},
					},
					Status: carbonv1.ShardGroupStatus{
						OnceCompletedShardNames: []string{"group.role-2"},
					},
				},
				roles: []*carbonv1.RollingSet{
					&carbonv1.RollingSet{
						ObjectMeta: metav1.ObjectMeta{
							Name: "group.role-2",
						},
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
				},
				shardGroupVersion: "group-1111",
			},
			want: &rollalgorithm.GroupScheduleParams{
				Name:              "group",
				ShardGroupVersion: "group-1111",
				LatestPercent:     100,
				MaxUnavailable:    10,
				MaxSurge:          0,
				NewShards:         []string{"group.role-1"},
				Metas: map[string]string{
					"appName":   "app",
					"groupName": "group",
				},
				ShardStatus: []rollalgorithm.Shard{
					&carbonv1.RollingSet{
						ObjectMeta: metav1.ObjectMeta{
							Name: "group.role-2",
						},
						Spec: carbonv1.RollingSetSpec{
							Version: "role-1111",
							VersionPlan: carbonv1.VersionPlan{
								SignedVersionPlan: carbonv1.SignedVersionPlan{
									ShardGroupVersion: "group-1111",
								},
							},
							SchedulePlan: rollalgorithm.SchedulePlan{
								Replicas:           utils.Int32Ptr(10),
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
				},
				TargetShardStrategys: map[string]*apps.DeploymentStrategy{},
				DefaultStrategy: apps.DeploymentStrategy{
					RollingUpdate: &apps.RollingUpdateDeployment{
						MaxUnavailable: getIntstrPtr("10"),
					},
				},
				//versions:          []string{"group-1111"},
			},
			wantErr: false,
		},
		{
			name: "test",
			args: args{
				shardGroup: &carbonv1.ShardGroup{
					ObjectMeta: metav1.ObjectMeta{
						Name: "group",
						Labels: map[string]string{
							"app.hippo.io/app-name":   "app",
							"app.hippo.io/group-name": "group",
						},
					},
					Spec: carbonv1.ShardGroupSpec{
						LatestPercent:  utils.Int32Ptr(100),
						MaxUnavailable: getIntstrPtr("10"),
						ShardTemplates: map[string]carbonv1.ShardTemplate{
							"role-1": carbonv1.ShardTemplate{
								Spec: carbonv1.ShardSpec{
									Replicas: utils.Int32Ptr(10),
								},
							},
						},
					},
					Status: carbonv1.ShardGroupStatus{
						OnceCompletedShardNames: []string{"group.role-2"},
					},
				},
				roles: []*carbonv1.RollingSet{
					&carbonv1.RollingSet{
						ObjectMeta: metav1.ObjectMeta{
							Name: "group.role-1",
						},
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
					&carbonv1.RollingSet{
						ObjectMeta: metav1.ObjectMeta{
							Name: "group.role-2",
						},
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
				},
				shardGroupVersion: "group-1111",
			},
			want: &rollalgorithm.GroupScheduleParams{
				Name:              "group",
				ShardGroupVersion: "group-1111",
				LatestPercent:     100,
				MaxUnavailable:    10,
				MaxSurge:          0,
				NewShards:         []string{"group.role-1"},
				Metas: map[string]string{
					"appName":   "app",
					"groupName": "group",
				},
				ShardStatus: []rollalgorithm.Shard{
					&carbonv1.RollingSet{
						ObjectMeta: metav1.ObjectMeta{
							Name: "group.role-1",
						},
						Spec: carbonv1.RollingSetSpec{
							Version: "role-1111",
							VersionPlan: carbonv1.VersionPlan{
								SignedVersionPlan: carbonv1.SignedVersionPlan{
									ShardGroupVersion: "group-1111",
								},
							},
							SchedulePlan: rollalgorithm.SchedulePlan{
								Replicas:           utils.Int32Ptr(10),
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
				},
				TargetShardStrategys: map[string]*apps.DeploymentStrategy{},
				DefaultStrategy: apps.DeploymentStrategy{
					RollingUpdate: &apps.RollingUpdateDeployment{
						MaxUnavailable: getIntstrPtr("10"),
					},
				},
				//versions:          []string{"group-1111"},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := initGroupScheduleParams(tt.args.shardGroup, tt.args.roles, tt.args.shardGroupVersion)
			if (err != nil) != tt.wantErr {
				t.Errorf("initGroupScheduleParams() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("initGroupScheduleParams() = %+v, want %+v", *got, *tt.want)
			}
		})
	}
}
