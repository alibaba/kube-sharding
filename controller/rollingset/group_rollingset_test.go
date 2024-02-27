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

package rollingset

import (
	"reflect"
	"testing"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	listers "github.com/alibaba/kube-sharding/pkg/client/listers/carbon/v1"
	"github.com/alibaba/kube-sharding/pkg/rollalgorithm"
	app "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	intstrutil "k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/cache"
	k8scontroller "k8s.io/kubernetes/pkg/controller"
)

func TestController_getSubRollingSetToUpdate(t *testing.T) {
	type fields struct {
		rollingSetLister  listers.RollingSetLister
		rollingSetSynced  cache.InformerSynced
		workerLister      listers.WorkerNodeLister
		workerSynced      cache.InformerSynced
		expectations      *k8scontroller.UIDTrackingControllerExpectations
		diffLogger        *utils.DiffLogger
		diffLoggerCmpFunc CmpFunc
	}
	type args struct {
		groupRollingSet *carbonv1.RollingSet
		subRollingSets  []*carbonv1.RollingSet
	}
	unavailableStr := intstrutil.FromString("20%")
	unavailableStr1 := intstrutil.FromString("50%")
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []*carbonv1.RollingSet
		want1  int32
	}{
		{
			name: "test",
			args: args{
				groupRollingSet: &carbonv1.RollingSet{
					Spec: carbonv1.RollingSetSpec{
						SubRSLatestVersionRatio: &map[string]int32{
							"sub1": 50,
							"*":    20,
						},
						Version: "1",
						SchedulePlan: rollalgorithm.SchedulePlan{
							Strategy: app.DeploymentStrategy{
								RollingUpdate: &app.RollingUpdateDeployment{
									MaxUnavailable: &unavailableStr,
								},
							},
						},
					},
				},
				subRollingSets: []*carbonv1.RollingSet{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "sub1",
						},
						Spec: carbonv1.RollingSetSpec{
							Version: "0",
							SchedulePlan: rollalgorithm.SchedulePlan{
								Replicas: utils.Int32Ptr(10),
								Strategy: app.DeploymentStrategy{
									RollingUpdate: &app.RollingUpdateDeployment{
										MaxUnavailable: &unavailableStr1,
									},
								},
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "sub2",
						},
						Spec: carbonv1.RollingSetSpec{
							Version: "0",
							SchedulePlan: rollalgorithm.SchedulePlan{
								Replicas: utils.Int32Ptr(10),
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "sub3",
						},
						Spec: carbonv1.RollingSetSpec{
							Version: "0",
							SchedulePlan: rollalgorithm.SchedulePlan{
								Replicas: utils.Int32Ptr(10),
							},
						},
					},
				},
			},
			want: []*carbonv1.RollingSet{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "sub1",
					},
					Spec: carbonv1.RollingSetSpec{
						Version: "0",
						SchedulePlan: rollalgorithm.SchedulePlan{
							Replicas:           utils.Int32Ptr(10),
							LatestVersionRatio: utils.Int32Ptr(50),
							Strategy: app.DeploymentStrategy{
								RollingUpdate: &app.RollingUpdateDeployment{
									MaxUnavailable: &unavailableStr1,
								},
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "sub2",
					},
					Spec: carbonv1.RollingSetSpec{
						Version: "0",
						SchedulePlan: rollalgorithm.SchedulePlan{
							Replicas:           utils.Int32Ptr(10),
							LatestVersionRatio: utils.Int32Ptr(20),
						},
					},
				},
			},
			want1: 30,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Controller{
				rollingSetLister:  tt.fields.rollingSetLister,
				rollingSetSynced:  tt.fields.rollingSetSynced,
				workerLister:      tt.fields.workerLister,
				workerSynced:      tt.fields.workerSynced,
				expectations:      tt.fields.expectations,
				diffLogger:        tt.fields.diffLogger,
				diffLoggerCmpFunc: tt.fields.diffLoggerCmpFunc,
			}
			got, got1, _ := c.getSubRollingSetToUpdate(tt.args.groupRollingSet, tt.args.subRollingSets)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Controller.getSubRollingSetToUpdate() got = %v, want %v", utils.ObjJSON(got), utils.ObjJSON(tt.want))
			}
			if got1 != tt.want1 {
				t.Errorf("Controller.getSubRollingSetToUpdate() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestController_classifyRollingSet(t *testing.T) {
	type fields struct {
		rollingSetLister  listers.RollingSetLister
		rollingSetSynced  cache.InformerSynced
		workerLister      listers.WorkerNodeLister
		workerSynced      cache.InformerSynced
		expectations      *k8scontroller.UIDTrackingControllerExpectations
		diffLogger        *utils.DiffLogger
		diffLoggerCmpFunc CmpFunc
	}
	type args struct {
		groupRollingSet *carbonv1.RollingSet
		subRollingSets  []*carbonv1.RollingSet
	}
	unavailableStr := intstrutil.FromString("20%")
	tests := []struct {
		name          string
		fields        fields
		args          args
		wantInRolling []*carbonv1.RollingSet
		wantToUpdate  []*carbonv1.RollingSet
	}{
		{
			name: "test",
			args: args{
				groupRollingSet: &carbonv1.RollingSet{
					Spec: carbonv1.RollingSetSpec{
						SubRSLatestVersionRatio: &map[string]int32{
							"sub1": 50,
							"*":    20,
						},
						Version:        "1",
						SubrsBlackList: []string{"sub7"},
						SchedulePlan: rollalgorithm.SchedulePlan{
							Strategy: app.DeploymentStrategy{
								RollingUpdate: &app.RollingUpdateDeployment{
									MaxUnavailable: &unavailableStr,
								},
							},
						},
					},
				},
				subRollingSets: []*carbonv1.RollingSet{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "sub1",
						},
						Spec: carbonv1.RollingSetSpec{
							Version: "1",
							SchedulePlan: rollalgorithm.SchedulePlan{
								Replicas:           utils.Int32Ptr(10),
								LatestVersionRatio: utils.Int32Ptr(50),
							},
						},
						Status: carbonv1.RollingSetStatus{
							GroupVersionStatusMap: map[string]*rollalgorithm.VersionStatus{
								"1": {
									ReadyReplicas: 5,
								},
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "sub2",
						},
						Spec: carbonv1.RollingSetSpec{
							Version: "1",
							SchedulePlan: rollalgorithm.SchedulePlan{
								Replicas:           utils.Int32Ptr(10),
								LatestVersionRatio: utils.Int32Ptr(20),
							},
						},
						Status: carbonv1.RollingSetStatus{
							GroupVersionStatusMap: map[string]*rollalgorithm.VersionStatus{
								"1": {
									ReadyReplicas: 2,
								},
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "sub3",
						},
						Spec: carbonv1.RollingSetSpec{
							Version: "1",
							SchedulePlan: rollalgorithm.SchedulePlan{
								Replicas:           utils.Int32Ptr(10),
								LatestVersionRatio: utils.Int32Ptr(20),
							},
						},
						Status: carbonv1.RollingSetStatus{
							GroupVersionStatusMap: map[string]*rollalgorithm.VersionStatus{
								"1": {
									ReadyReplicas: 1,
								},
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "sub4",
						},
						Spec: carbonv1.RollingSetSpec{
							Version: "1",
							SchedulePlan: rollalgorithm.SchedulePlan{
								LatestVersionRatio: utils.Int32Ptr(20),
								Replicas:           utils.Int32Ptr(10),
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "sub5",
						},
						Spec: carbonv1.RollingSetSpec{
							SchedulePlan: rollalgorithm.SchedulePlan{
								Replicas:           utils.Int32Ptr(10),
								LatestVersionRatio: utils.Int32Ptr(20),
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "sub6",
						},
						Spec: carbonv1.RollingSetSpec{
							Version: "1",
							SchedulePlan: rollalgorithm.SchedulePlan{
								Replicas: utils.Int32Ptr(10),
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "sub7",
							Labels: map[string]string{
								carbonv1.DefaultShortSubRSLabelKey: "sub7",
							},
						},
						Spec: carbonv1.RollingSetSpec{
							SchedulePlan: rollalgorithm.SchedulePlan{
								Replicas:           utils.Int32Ptr(10),
								LatestVersionRatio: utils.Int32Ptr(20),
							},
						},
					},
				},
			},
			wantInRolling: []*carbonv1.RollingSet{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "sub3",
					},
					Spec: carbonv1.RollingSetSpec{
						Version: "1",
						SchedulePlan: rollalgorithm.SchedulePlan{
							Replicas:           utils.Int32Ptr(10),
							LatestVersionRatio: utils.Int32Ptr(20),
						},
						VersionPlan: carbonv1.VersionPlan{SignedVersionPlan: carbonv1.SignedVersionPlan{Template: &carbonv1.HippoPodTemplate{}}},
					},
					Status: carbonv1.RollingSetStatus{
						GroupVersionStatusMap: map[string]*rollalgorithm.VersionStatus{
							"1": {
								ReadyReplicas: 1,
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "sub4",
					},
					Spec: carbonv1.RollingSetSpec{
						Version: "1",
						SchedulePlan: rollalgorithm.SchedulePlan{
							LatestVersionRatio: utils.Int32Ptr(20),
							Replicas:           utils.Int32Ptr(10),
						},
						VersionPlan: carbonv1.VersionPlan{SignedVersionPlan: carbonv1.SignedVersionPlan{Template: &carbonv1.HippoPodTemplate{}}},
					},
				},
			},
			wantToUpdate: []*carbonv1.RollingSet{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "sub5",
					},
					Spec: carbonv1.RollingSetSpec{
						SchedulePlan: rollalgorithm.SchedulePlan{
							Replicas:           utils.Int32Ptr(10),
							LatestVersionRatio: utils.Int32Ptr(20),
						},
						VersionPlan: carbonv1.VersionPlan{SignedVersionPlan: carbonv1.SignedVersionPlan{Template: &carbonv1.HippoPodTemplate{}}},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "sub6",
					},
					Spec: carbonv1.RollingSetSpec{
						Version: "1",
						SchedulePlan: rollalgorithm.SchedulePlan{
							Replicas:           utils.Int32Ptr(10),
							LatestVersionRatio: utils.Int32Ptr(20),
						},
						VersionPlan: carbonv1.VersionPlan{SignedVersionPlan: carbonv1.SignedVersionPlan{Template: &carbonv1.HippoPodTemplate{}}},
					},
				},
			},
		},
		{
			name: "test subrs can rollback",
			args: args{
				groupRollingSet: &carbonv1.RollingSet{
					Spec: carbonv1.RollingSetSpec{
						SubRSLatestVersionRatio: &map[string]int32{
							"sub1": 50,
							"*":    20,
						},
						Version: "1",
						SchedulePlan: rollalgorithm.SchedulePlan{
							Strategy: app.DeploymentStrategy{
								RollingUpdate: &app.RollingUpdateDeployment{
									MaxUnavailable: &unavailableStr,
								},
							},
							SubrsCanRollback: true,
						},
					},
				},
				subRollingSets: []*carbonv1.RollingSet{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "sub1",
						},
						Spec: carbonv1.RollingSetSpec{
							Version: "1",
							SchedulePlan: rollalgorithm.SchedulePlan{
								Replicas:           utils.Int32Ptr(10),
								LatestVersionRatio: utils.Int32Ptr(80),
							},
						},
					},
				},
			},
			wantToUpdate: []*carbonv1.RollingSet{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "sub1",
					},
					Spec: carbonv1.RollingSetSpec{
						Version: "1",
						SchedulePlan: rollalgorithm.SchedulePlan{
							Replicas:           utils.Int32Ptr(10),
							LatestVersionRatio: utils.Int32Ptr(50),
						},
						VersionPlan: carbonv1.VersionPlan{SignedVersionPlan: carbonv1.SignedVersionPlan{Template: &carbonv1.HippoPodTemplate{}}},
					},
				},
			},
		},
		{
			name: "test subrs can't rollback",
			args: args{
				groupRollingSet: &carbonv1.RollingSet{
					Spec: carbonv1.RollingSetSpec{
						SubRSLatestVersionRatio: &map[string]int32{
							"sub1": 50,
							"*":    20,
						},
						Version: "1",
						SchedulePlan: rollalgorithm.SchedulePlan{
							Strategy: app.DeploymentStrategy{
								RollingUpdate: &app.RollingUpdateDeployment{
									MaxUnavailable: &unavailableStr,
								},
							},
							SubrsCanRollback: false,
						},
					},
				},
				subRollingSets: []*carbonv1.RollingSet{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "sub1",
						},
						Spec: carbonv1.RollingSetSpec{
							Version: "1",
							SchedulePlan: rollalgorithm.SchedulePlan{
								Replicas:           utils.Int32Ptr(10),
								LatestVersionRatio: utils.Int32Ptr(80),
							},
							VersionPlan: carbonv1.VersionPlan{SignedVersionPlan: carbonv1.SignedVersionPlan{Template: &carbonv1.HippoPodTemplate{}}},
						},
					},
				},
			},
			wantInRolling: []*carbonv1.RollingSet{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "sub1",
					},
					Spec: carbonv1.RollingSetSpec{
						Version: "1",
						SchedulePlan: rollalgorithm.SchedulePlan{
							Replicas:           utils.Int32Ptr(10),
							LatestVersionRatio: utils.Int32Ptr(80),
						},
						VersionPlan: carbonv1.VersionPlan{SignedVersionPlan: carbonv1.SignedVersionPlan{Template: &carbonv1.HippoPodTemplate{}}},
					},
				},
			},
		},
		{
			name: "subrs paused",
			args: args{
				groupRollingSet: &carbonv1.RollingSet{
					Spec: carbonv1.RollingSetSpec{
						SubRSLatestVersionRatio: &map[string]int32{
							"sub1": 50,
							"*":    20,
						},
						Version: "1",
						SchedulePlan: rollalgorithm.SchedulePlan{
							Strategy: app.DeploymentStrategy{
								RollingUpdate: &app.RollingUpdateDeployment{
									MaxUnavailable: &unavailableStr,
								},
							},
							Paused: true,
						},
					},
				},
				subRollingSets: []*carbonv1.RollingSet{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "sub1",
						},
						Spec: carbonv1.RollingSetSpec{
							Version: "1",
							SchedulePlan: rollalgorithm.SchedulePlan{
								Replicas:           utils.Int32Ptr(9),
								LatestVersionRatio: utils.Int32Ptr(80),
							},
						},
						Status: carbonv1.RollingSetStatus{
							UpdatedReplicas: 1,
						},
					},
				},
			},
			wantToUpdate: []*carbonv1.RollingSet{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "sub1",
					},
					Spec: carbonv1.RollingSetSpec{
						Version: "1",
						SchedulePlan: rollalgorithm.SchedulePlan{
							Replicas:           utils.Int32Ptr(9),
							LatestVersionRatio: utils.Int32Ptr(11),
						},
						VersionPlan: carbonv1.VersionPlan{SignedVersionPlan: carbonv1.SignedVersionPlan{Template: &carbonv1.HippoPodTemplate{}}},
					},
					Status: carbonv1.RollingSetStatus{
						UpdatedReplicas: 1,
					},
				},
			},
		},
		{
			name: "subrs non versioned key not match",
			args: args{
				groupRollingSet: &carbonv1.RollingSet{
					Spec: carbonv1.RollingSetSpec{
						SubRSLatestVersionRatio: &map[string]int32{
							"sub1": 50,
							"*":    20,
						},
						Version: "1",
						SchedulePlan: rollalgorithm.SchedulePlan{
							Strategy: app.DeploymentStrategy{
								RollingUpdate: &app.RollingUpdateDeployment{
									MaxUnavailable: &unavailableStr,
								},
							},
						},
					},
				},
				subRollingSets: []*carbonv1.RollingSet{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "sub1",
						},
						Spec: carbonv1.RollingSetSpec{
							Version: "1",
							SchedulePlan: rollalgorithm.SchedulePlan{
								Replicas:           utils.Int32Ptr(9),
								LatestVersionRatio: utils.Int32Ptr(50),
							},
							VersionPlan: carbonv1.VersionPlan{SignedVersionPlan: carbonv1.SignedVersionPlan{Template: &carbonv1.HippoPodTemplate{
								ObjectMeta: metav1.ObjectMeta{
									Labels: map[string]string{carbonv1.LabelServerlessAppName: "aa"},
								},
							}}},
						},
						Status: carbonv1.RollingSetStatus{
							UpdatedReplicas: 1,
						},
					},
				},
			},
			wantInRolling: []*carbonv1.RollingSet{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "sub1",
					},
					Spec: carbonv1.RollingSetSpec{
						Version: "1",
						SchedulePlan: rollalgorithm.SchedulePlan{
							Replicas:           utils.Int32Ptr(9),
							LatestVersionRatio: utils.Int32Ptr(50),
						},
						VersionPlan: carbonv1.VersionPlan{SignedVersionPlan: carbonv1.SignedVersionPlan{Template: &carbonv1.HippoPodTemplate{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{carbonv1.LabelServerlessAppName: "aa"},
							},
						}}},
					},
					Status: carbonv1.RollingSetStatus{
						UpdatedReplicas: 1,
					},
				},
			},
			wantToUpdate: []*carbonv1.RollingSet{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "sub1",
					},
					Spec: carbonv1.RollingSetSpec{
						Version: "1",
						SchedulePlan: rollalgorithm.SchedulePlan{
							Replicas:           utils.Int32Ptr(9),
							LatestVersionRatio: utils.Int32Ptr(50),
						},
						VersionPlan: carbonv1.VersionPlan{SignedVersionPlan: carbonv1.SignedVersionPlan{Template: &carbonv1.HippoPodTemplate{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{carbonv1.LabelServerlessAppName: "aa"},
							},
						}}},
					},
					Status: carbonv1.RollingSetStatus{
						UpdatedReplicas: 1,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Controller{
				rollingSetLister:  tt.fields.rollingSetLister,
				rollingSetSynced:  tt.fields.rollingSetSynced,
				workerLister:      tt.fields.workerLister,
				workerSynced:      tt.fields.workerSynced,
				expectations:      tt.fields.expectations,
				diffLogger:        tt.fields.diffLogger,
				diffLoggerCmpFunc: tt.fields.diffLoggerCmpFunc,
			}
			if tt.args.groupRollingSet.Spec.Template == nil {
				tt.args.groupRollingSet.Spec.VersionPlan = carbonv1.VersionPlan{SignedVersionPlan: carbonv1.SignedVersionPlan{Template: &carbonv1.HippoPodTemplate{}}}
			}
			for i := range tt.args.subRollingSets {
				if tt.args.subRollingSets[i].Spec.Template == nil {
					tt.args.subRollingSets[i].Spec.VersionPlan = carbonv1.VersionPlan{SignedVersionPlan: carbonv1.SignedVersionPlan{Template: &carbonv1.HippoPodTemplate{}}}
				}
			}
			gotInRolling, gotToUpdate, _ := c.computeSubRollingSet(tt.args.groupRollingSet, tt.args.subRollingSets)
			if !reflect.DeepEqual(gotInRolling, tt.wantInRolling) {
				t.Errorf("Controller.computeSubRollingSet() gotInRolling = %v, want %v", utils.ObjJSON(gotInRolling), utils.ObjJSON(tt.wantInRolling))
			}
			if !reflect.DeepEqual(gotToUpdate, tt.wantToUpdate) {
				t.Errorf("Controller.computeSubRollingSet() wantToUpdate = %v, want %v", utils.ObjJSON(gotToUpdate), utils.ObjJSON(tt.wantToUpdate))
			}
		})
	}
}

func TestController_getLatestVersionRatio(t *testing.T) {
	type fields struct {
		rollingSetLister  listers.RollingSetLister
		rollingSetSynced  cache.InformerSynced
		workerLister      listers.WorkerNodeLister
		workerSynced      cache.InformerSynced
		expectations      *k8scontroller.UIDTrackingControllerExpectations
		diffLogger        *utils.DiffLogger
		diffLoggerCmpFunc CmpFunc
	}
	type args struct {
		groupRollingSet *carbonv1.RollingSet
		subRollingSet   *carbonv1.RollingSet
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   int32
	}{
		{
			name: "test",
			args: args{
				groupRollingSet: &carbonv1.RollingSet{
					Spec: carbonv1.RollingSetSpec{
						SubRSLatestVersionRatio: &map[string]int32{
							"sub1": 50,
							"*":    20,
						},
					},
				},
				subRollingSet: &carbonv1.RollingSet{
					ObjectMeta: metav1.ObjectMeta{
						Name: "sub1",
					},
				},
			},
			want: 50,
		},
		{
			name: "test",
			args: args{
				groupRollingSet: &carbonv1.RollingSet{
					ObjectMeta: metav1.ObjectMeta{
						Name: "grs",
					},
					Spec: carbonv1.RollingSetSpec{
						SubRSLatestVersionRatio: &map[string]int32{
							"sub1": 50,
							"*":    20,
						},
					},
				},
				subRollingSet: &carbonv1.RollingSet{
					ObjectMeta: metav1.ObjectMeta{
						Name: "grs.sub1",
						Labels: map[string]string{
							carbonv1.DefaultShortSubRSLabelKey: "sub1",
						},
					},
				},
			},
			want: 50,
		},
		{
			name: "test",
			args: args{
				groupRollingSet: &carbonv1.RollingSet{
					Spec: carbonv1.RollingSetSpec{
						SubRSLatestVersionRatio: &map[string]int32{
							"sub2": 50,
							"*":    20,
						},
					},
				},
				subRollingSet: &carbonv1.RollingSet{
					ObjectMeta: metav1.ObjectMeta{
						Name: "sub1",
					},
				},
			},
			want: 20,
		},
		{
			name: "test",
			args: args{
				groupRollingSet: &carbonv1.RollingSet{
					Spec: carbonv1.RollingSetSpec{},
				},
				subRollingSet: &carbonv1.RollingSet{
					ObjectMeta: metav1.ObjectMeta{
						Name: "sub1",
					},
				},
			},
			want: 0,
		},
		{
			name: "test app.*",
			args: args{
				groupRollingSet: &carbonv1.RollingSet{
					Spec: carbonv1.RollingSetSpec{
						SubRSLatestVersionRatio: &map[string]int32{
							"app.*": 50,
						},
					},
				},
				subRollingSet: &carbonv1.RollingSet{
					ObjectMeta: metav1.ObjectMeta{
						Name: "app.xxx",
						Labels: map[string]string{
							carbonv1.DefaultShortSubRSLabelKey: "app.xxx",
						},
					},
				},
			},
			want: 50,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Controller{
				rollingSetLister:  tt.fields.rollingSetLister,
				rollingSetSynced:  tt.fields.rollingSetSynced,
				workerLister:      tt.fields.workerLister,
				workerSynced:      tt.fields.workerSynced,
				expectations:      tt.fields.expectations,
				diffLogger:        tt.fields.diffLogger,
				diffLoggerCmpFunc: tt.fields.diffLoggerCmpFunc,
			}
			if got := c.getLatestVersionRatio(tt.args.groupRollingSet, tt.args.subRollingSet); got != tt.want {
				t.Errorf("Controller.getLatestVersionRatio() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestController_sumReplicas(t *testing.T) {
	type fields struct {
		rollingSetLister  listers.RollingSetLister
		rollingSetSynced  cache.InformerSynced
		workerLister      listers.WorkerNodeLister
		workerSynced      cache.InformerSynced
		expectations      *k8scontroller.UIDTrackingControllerExpectations
		diffLogger        *utils.DiffLogger
		diffLoggerCmpFunc CmpFunc
	}
	type args struct {
		groupRollingSet *carbonv1.RollingSet
		subRollingSets  []*carbonv1.RollingSet
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   int32
		want1  int32
	}{
		{
			name: "test",
			args: args{
				groupRollingSet: &carbonv1.RollingSet{
					Spec: carbonv1.RollingSetSpec{
						SubRSLatestVersionRatio: &map[string]int32{
							"sub1": 50,
							"*":    20,
						},
					},
				},
				subRollingSets: []*carbonv1.RollingSet{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "sub1",
						},
						Spec: carbonv1.RollingSetSpec{
							Version: "1",
							SchedulePlan: rollalgorithm.SchedulePlan{
								Replicas: utils.Int32Ptr(10),
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "sub2",
						},
						Spec: carbonv1.RollingSetSpec{
							Version: "1",
							SchedulePlan: rollalgorithm.SchedulePlan{
								Replicas: utils.Int32Ptr(10),
							},
						},
					},
				},
			},
			want:  20,
			want1: 35,
		},
		{
			name: "test",
			args: args{
				groupRollingSet: &carbonv1.RollingSet{
					Spec: carbonv1.RollingSetSpec{
						SubRSLatestVersionRatio: &map[string]int32{
							"*": 20,
						},
					},
				},
				subRollingSets: []*carbonv1.RollingSet{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "sub1",
						},
						Spec: carbonv1.RollingSetSpec{
							Version: "1",
							SchedulePlan: rollalgorithm.SchedulePlan{
								Replicas: utils.Int32Ptr(10),
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "sub2",
						},
						Spec: carbonv1.RollingSetSpec{
							Version: "1",
							SchedulePlan: rollalgorithm.SchedulePlan{
								Replicas: utils.Int32Ptr(10),
							},
						},
					},
				},
			},
			want:  20,
			want1: 20,
		},
		{
			name: "subrs black list",
			args: args{
				groupRollingSet: &carbonv1.RollingSet{
					Spec: carbonv1.RollingSetSpec{
						SubRSLatestVersionRatio: &map[string]int32{
							"*": 20,
						},
						SubrsBlackList: []string{"sub1"},
					},
				},
				subRollingSets: []*carbonv1.RollingSet{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "sub1",
							Labels: map[string]string{
								"app.c2.io/short-subrs-name": "sub1",
							},
						},
						Spec: carbonv1.RollingSetSpec{
							Version: "1",
							SchedulePlan: rollalgorithm.SchedulePlan{
								Replicas:           utils.Int32Ptr(10),
								LatestVersionRatio: utils.Int32Ptr(60),
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "sub2",
						},
						Spec: carbonv1.RollingSetSpec{
							Version: "1",
							SchedulePlan: rollalgorithm.SchedulePlan{
								Replicas: utils.Int32Ptr(10),
							},
						},
					},
				},
			},
			want:  20,
			want1: 40, // (6+2)/20
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Controller{
				rollingSetLister:  tt.fields.rollingSetLister,
				rollingSetSynced:  tt.fields.rollingSetSynced,
				workerLister:      tt.fields.workerLister,
				workerSynced:      tt.fields.workerSynced,
				expectations:      tt.fields.expectations,
				diffLogger:        tt.fields.diffLogger,
				diffLoggerCmpFunc: tt.fields.diffLoggerCmpFunc,
			}
			got, got1 := c.sumReplicas(tt.args.groupRollingSet, tt.args.subRollingSets)
			if got != tt.want {
				t.Errorf("Controller.sumReplicas() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("Controller.sumReplicas() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestController_isRollingSetMatched(t *testing.T) {
	type fields struct {
		rollingSetLister  listers.RollingSetLister
		rollingSetSynced  cache.InformerSynced
		workerLister      listers.WorkerNodeLister
		workerSynced      cache.InformerSynced
		expectations      *k8scontroller.UIDTrackingControllerExpectations
		diffLogger        *utils.DiffLogger
		diffLoggerCmpFunc CmpFunc
	}
	type args struct {
		rollingSet *carbonv1.RollingSet
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "match",
			args: args{
				rollingSet: &carbonv1.RollingSet{
					ObjectMeta: metav1.ObjectMeta{
						Name: "sub2",
					},
					Spec: carbonv1.RollingSetSpec{
						Version: "1",
						SchedulePlan: rollalgorithm.SchedulePlan{
							Replicas:           utils.Int32Ptr(10),
							LatestVersionRatio: utils.Int32Ptr(20),
						},
					},
					Status: carbonv1.RollingSetStatus{
						GroupVersionStatusMap: map[string]*rollalgorithm.VersionStatus{
							"1": {
								ReadyReplicas: 2,
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "not match",
			args: args{
				rollingSet: &carbonv1.RollingSet{
					ObjectMeta: metav1.ObjectMeta{
						Name: "sub2",
					},
					Spec: carbonv1.RollingSetSpec{
						Version: "1",
						SchedulePlan: rollalgorithm.SchedulePlan{
							Replicas:           utils.Int32Ptr(10),
							LatestVersionRatio: utils.Int32Ptr(20),
						},
					},
					Status: carbonv1.RollingSetStatus{
						GroupVersionStatusMap: map[string]*rollalgorithm.VersionStatus{
							"0": {
								ReadyReplicas: 2,
							},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "not match",
			args: args{
				rollingSet: &carbonv1.RollingSet{
					ObjectMeta: metav1.ObjectMeta{
						Name: "sub2",
					},
					Spec: carbonv1.RollingSetSpec{
						Version: "1",
						SchedulePlan: rollalgorithm.SchedulePlan{
							Replicas:           utils.Int32Ptr(10),
							LatestVersionRatio: utils.Int32Ptr(20),
						},
					},
					Status: carbonv1.RollingSetStatus{
						GroupVersionStatusMap: map[string]*rollalgorithm.VersionStatus{
							"1": {
								ReadyReplicas: 1,
							},
						},
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Controller{
				rollingSetLister:  tt.fields.rollingSetLister,
				rollingSetSynced:  tt.fields.rollingSetSynced,
				workerLister:      tt.fields.workerLister,
				workerSynced:      tt.fields.workerSynced,
				expectations:      tt.fields.expectations,
				diffLogger:        tt.fields.diffLogger,
				diffLoggerCmpFunc: tt.fields.diffLoggerCmpFunc,
			}
			if got := c.isRollingSetMatched(tt.args.rollingSet); got != tt.want {
				t.Errorf("Controller.isRollingSetMatched() = %v, want %v", got, tt.want)
			}
		})
	}
}
