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

package v1

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	"github.com/alibaba/kube-sharding/pkg/rollalgorithm"
	app "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestScacledSchedulePlan_getReplicas(t *testing.T) {
	type fields struct {
		Replicas         *int32
		Strategy         app.DeploymentStrategy
		RecycleMode      WorkerModeType
		ResourceRequests corev1.ResourceList
	}
	tests := []struct {
		name   string
		fields fields
		want   int32
	}{
		{
			name: "get",
			fields: fields{
				Replicas: utils.Int32Ptr(1),
			},
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &ScaleSchedulePlan{
				Replicas:         tt.fields.Replicas,
				Strategy:         tt.fields.Strategy,
				RecycleMode:      tt.fields.RecycleMode,
				ResourceRequests: tt.fields.ResourceRequests,
			}
			if got := p.getReplicas(); got != tt.want {
				t.Errorf("ScaleSchedulePlan.getReplicas() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRollingSet_GetRawReplicas(t *testing.T) {
	type fields struct {
		TypeMeta   metav1.TypeMeta
		ObjectMeta metav1.ObjectMeta
		Spec       RollingSetSpec
		Status     RollingSetStatus
	}
	tests := []struct {
		name   string
		fields fields
		want   int32
	}{
		{
			name: "get",
			fields: fields{
				Spec: RollingSetSpec{},
			},
			want: 0,
		},
		{
			name: "get",
			fields: fields{
				Spec: RollingSetSpec{
					SchedulePlan: rollalgorithm.SchedulePlan{
						Replicas: utils.Int32Ptr(1),
					},
				},
			},
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RollingSet{
				TypeMeta:   tt.fields.TypeMeta,
				ObjectMeta: tt.fields.ObjectMeta,
				Spec:       tt.fields.Spec,
				Status:     tt.fields.Status,
			}
			if got := r.GetRawReplicas(); got != tt.want {
				t.Errorf("RollingSet.GetRawReplicas() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRollingSet_GetReplicas(t *testing.T) {
	type fields struct {
		TypeMeta   metav1.TypeMeta
		ObjectMeta metav1.ObjectMeta
		Spec       RollingSetSpec
		Status     RollingSetStatus
	}
	tests := []struct {
		name   string
		fields fields
		want   int32
	}{
		{
			name: "get",
			fields: fields{
				Spec: RollingSetSpec{},
			},
			want: 0,
		},
		{
			name: "get",
			fields: fields{
				Spec: RollingSetSpec{
					SchedulePlan: rollalgorithm.SchedulePlan{
						Replicas: utils.Int32Ptr(1),
					},
				},
			},
			want: 1,
		},
		{
			name: "get",
			fields: fields{
				Spec: RollingSetSpec{
					SchedulePlan: rollalgorithm.SchedulePlan{
						Replicas: utils.Int32Ptr(10),
					},
					Capacity: Capacity{
						ActivePlan: ActivePlan{
							ScaleSchedulePlan: &ScaleSchedulePlan{
								Replicas: utils.Int32Ptr(3),
							},
							ScaleConfig: &ScaleConfig{Enable: true},
						},
					},
				},
			},
			want: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RollingSet{
				TypeMeta:   tt.fields.TypeMeta,
				ObjectMeta: tt.fields.ObjectMeta,
				Spec:       tt.fields.Spec,
				Status:     tt.fields.Status,
			}
			if got := r.GetReplicas(); got != tt.want {
				t.Errorf("RollingSet.GetReplicas() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRollingSet_GetScaleParameters(t *testing.T) {
	type fields struct {
		TypeMeta   metav1.TypeMeta
		ObjectMeta metav1.ObjectMeta
		Spec       RollingSetSpec
		Status     RollingSetStatus
	}
	tests := []struct {
		name   string
		fields fields
		want   *ScaleParameters
	}{
		{
			name: "get",
			fields: fields{
				Spec: RollingSetSpec{},
			},
			want: nil,
		},
		{
			name: "get",
			fields: fields{
				Spec: RollingSetSpec{
					SchedulePlan: rollalgorithm.SchedulePlan{
						Replicas: utils.Int32Ptr(1),
					},
				},
			},
			want: nil,
		},
		{
			name: "scale & reserve",
			fields: fields{
				Spec: RollingSetSpec{
					SchedulePlan: rollalgorithm.SchedulePlan{
						Replicas: utils.Int32Ptr(40),
					},
					ScaleSchedulePlan: &ScaleSchedulePlan{
						Replicas:    utils.Int32Ptr(30),
						RecycleMode: WorkerModeTypeHot,
						RecyclePool: "testaop",
					},
					ScaleConfig: &ScaleConfig{
						Enable: true,
					},
					Capacity: Capacity{
						Total:      utils.Int32Ptr(130),
						ActivePlan: ActivePlan{ScaleConfig: &ScaleConfig{Enable: true}},
						BufferPlan: BufferPlan{
							Distribute: &BufferDistribute{
								Cold: 30,
								Warm: 50,
								Hot:  20,
							},
						},
					},
				},
			},
			want: &ScaleParameters{
				ActiveReplicas:  30,
				StandByReplicas: 100,
				StandByRatioMap: map[WorkerModeType]int{
					WorkerModeTypeInactive: 0,
					WorkerModeTypeCold:     30,
					WorkerModeTypeWarm:     50,
					WorkerModeTypeHot:      20,
				},
				ScalerResourcePoolName: "testaop",
			},
		},
		{
			name: "no scale & reserve",
			fields: fields{
				Spec: RollingSetSpec{
					SchedulePlan: rollalgorithm.SchedulePlan{
						Replicas: utils.Int32Ptr(40),
					},
					ScaleSchedulePlan: &ScaleSchedulePlan{
						Replicas:    utils.Int32Ptr(30),
						RecycleMode: WorkerModeTypeHot,
						RecyclePool: "testaop2",
					},
					ScaleConfig: &ScaleConfig{
						Enable: false,
					},
					Capacity: Capacity{
						Total: utils.Int32Ptr(140),
						BufferPlan: BufferPlan{
							Distribute: &BufferDistribute{
								Cold: 30,
								Warm: 50,
								Hot:  20,
							},
						},
					},
				},
			},
			want: &ScaleParameters{
				ActiveReplicas:  40,
				StandByReplicas: 100,
				StandByRatioMap: map[WorkerModeType]int{
					WorkerModeTypeInactive: 0,
					WorkerModeTypeCold:     30,
					WorkerModeTypeWarm:     50,
					WorkerModeTypeHot:      20,
				},
				ScalerResourcePoolName: "testaop2",
			},
		},
		{
			name: "scale & no reserve",
			fields: fields{
				Spec: RollingSetSpec{
					SchedulePlan: rollalgorithm.SchedulePlan{
						Replicas: utils.Int32Ptr(10),
					},
					ScaleSchedulePlan: &ScaleSchedulePlan{
						Replicas:    utils.Int32Ptr(3),
						RecycleMode: WorkerModeTypeHot,
					},
					Capacity: Capacity{
						Total: nil,
						BufferPlan: BufferPlan{
							Distribute: &BufferDistribute{},
						},
					},
					ScaleConfig: &ScaleConfig{
						Enable: true,
					},
				},
			},
			want: &ScaleParameters{
				ActiveReplicas:  3,
				StandByReplicas: 7,
				StandByRatioMap: map[WorkerModeType]int{
					WorkerModeTypeInactive: 0,
					WorkerModeTypeCold:     0,
					WorkerModeTypeWarm:     0,
					WorkerModeTypeHot:      100,
				},
			},
		},
		{
			name: "inactive",
			fields: fields{
				Spec: RollingSetSpec{
					SchedulePlan: rollalgorithm.SchedulePlan{
						Replicas: utils.Int32Ptr(10),
					},
					ScaleSchedulePlan: &ScaleSchedulePlan{
						Replicas:    utils.Int32Ptr(3),
						RecycleMode: WorkerModeTypeInactive,
					},
					Capacity: Capacity{
						Total: nil,
						BufferPlan: BufferPlan{
							Distribute: &BufferDistribute{},
						},
					},
					ScaleConfig: &ScaleConfig{
						Enable: true,
					},
				},
			},
			want: &ScaleParameters{
				ActiveReplicas:  3,
				StandByReplicas: 7,
				StandByRatioMap: map[WorkerModeType]int{
					WorkerModeTypeInactive: 100,
					WorkerModeTypeCold:     0,
					WorkerModeTypeWarm:     0,
					WorkerModeTypeHot:      0,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RollingSet{
				TypeMeta:   tt.fields.TypeMeta,
				ObjectMeta: tt.fields.ObjectMeta,
				Spec:       tt.fields.Spec,
				Status:     tt.fields.Status,
			}
			r.Spec.Capacity.ActivePlan = ActivePlan{
				ScaleConfig:       r.Spec.ScaleConfig,
				ScaleSchedulePlan: r.Spec.ScaleSchedulePlan,
			}
			if got := r.GetScaleParameters(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("case[%s] RollingSet.GetScaleParameters() = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func Test_RollingsetVerbose(t *testing.T) {
	verbose := RollingsetVerbose{
		Status: &RollingSetStatus{
			Complete: false,
		},
		Replicas: []*ReplicaStatus{
			{
				Name: "Name",
			},
		},
	}
	marshal, err := json.Marshal(verbose)
	assert.Nil(t, err)
	res := &RollingsetVerbose{}
	assert.Nil(t, json.Unmarshal(marshal, res))
	assert.False(t, res.Status.Complete)
	assert.True(t, len(res.Replicas) == 1)
	assert.Equal(t, "Name", res.Replicas[0].Name)
}
