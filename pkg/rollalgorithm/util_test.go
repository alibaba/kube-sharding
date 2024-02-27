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
	app "k8s.io/api/apps/v1"
	apps "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	intstrutil "k8s.io/apimachinery/pkg/util/intstr"
)

func TestResolveFenceposts(t *testing.T) {
	tests := []struct {
		name           string
		maxSurge       intstrutil.IntOrString
		maxUnavailable intstrutil.IntOrString
		expectErr      bool
		expectVal1     int32
		expectVal2     int32
	}{
		{
			name:           "test1",
			maxSurge:       intstrutil.FromString("10%"),
			maxUnavailable: intstrutil.FromString("100%"),
			expectErr:      false,
			expectVal1:     10,
			expectVal2:     100,
		},
		{
			name:           "test2",
			maxSurge:       intstrutil.FromString("10%"),
			maxUnavailable: intstrutil.FromString("40%"),
			expectErr:      false,
			expectVal1:     10,
			expectVal2:     40,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got1, got2, _ := ResolveFenceposts(&tt.maxSurge, &tt.maxUnavailable, 100)
			if got1 != tt.expectVal1 || got2 != tt.expectVal2 {
				t.Errorf("Controller.calculateHoldMatrix() got1 = %v, want %v", got1, tt.expectVal1)
			}
		})
	}
}

func getIntstrPtr(src string) *intstr.IntOrString {
	v := intstr.FromString(src)
	return &v
}

func newStrategy(maxUnavailable string) *app.DeploymentStrategy {
	var strategy = apps.DeploymentStrategy{
		RollingUpdate: &apps.RollingUpdateDeployment{
			MaxUnavailable: getIntstrPtr(maxUnavailable),
		},
	}
	return &strategy
}

func newStrategy1(maxUnavailable string, maxSurge string) *app.DeploymentStrategy {
	var strategy = apps.DeploymentStrategy{
		RollingUpdate: &apps.RollingUpdateDeployment{
			MaxUnavailable: getIntstrPtr(maxUnavailable),
			MaxSurge:       getIntstrPtr(maxSurge),
		},
	}
	return &strategy
}

func TestParseScheduleStrategy(t *testing.T) {
	type args struct {
		plan SchedulePlan
	}
	tests := []struct {
		name  string
		args  args
		want  int32
		want1 int32
		want2 int32
	}{
		{
			name: "normal",
			args: args{
				plan: SchedulePlan{
					Replicas: utils.Int32Ptr(10),
					Strategy: app.DeploymentStrategy{
						Type: app.RollingUpdateDeploymentStrategyType,
						RollingUpdate: &app.RollingUpdateDeployment{
							MaxUnavailable: getIntstrPtr("30%"),
							MaxSurge:       getIntstrPtr("30%"),
						},
					},
				},
			},
			want:  3,
			want1: 3,
			want2: 7,
		},
		{
			name: "max unavailable Floor",
			args: args{
				plan: SchedulePlan{
					Replicas: utils.Int32Ptr(7),
					Strategy: app.DeploymentStrategy{
						Type: app.RollingUpdateDeploymentStrategyType,
						RollingUpdate: &app.RollingUpdateDeployment{
							MaxUnavailable: getIntstrPtr("10%"),
							MaxSurge:       getIntstrPtr("10%"),
						},
					},
				},
			},
			want:  1,
			want1: 0,
			want2: 7,
		},
		{
			name: "max surge Ceil",
			args: args{
				plan: SchedulePlan{
					Replicas: utils.Int32Ptr(7),
					Strategy: app.DeploymentStrategy{
						Type: app.RollingUpdateDeploymentStrategyType,
						RollingUpdate: &app.RollingUpdateDeployment{
							MaxUnavailable: getIntstrPtr("40%"),
							MaxSurge:       getIntstrPtr("30%"),
						},
					},
				},
			},
			want:  3,
			want1: 2,
			want2: 5,
		},
		{
			name: "max surge Ceil",
			args: args{
				plan: SchedulePlan{
					Replicas: utils.Int32Ptr(3),
					Strategy: app.DeploymentStrategy{
						Type: app.RollingUpdateDeploymentStrategyType,
						RollingUpdate: &app.RollingUpdateDeployment{
							MaxUnavailable: getIntstrPtr("34%"),
							MaxSurge:       getIntstrPtr("10%"),
						},
					},
				},
			},
			want:  1,
			want1: 1,
			want2: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, got2, _, _, _ := ParseScheduleStrategy(tt.args.plan)
			if got != tt.want {
				t.Errorf("ParseScheduleStrategy() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("ParseScheduleStrategy() got1 = %v, want %v", got1, tt.want1)
			}
			if got2 != tt.want2 {
				t.Errorf("ParseScheduleStrategy() got2 = %v, want %v", got2, tt.want2)
			}
		})
	}
}

func Test_floatRound(t *testing.T) {
	type args struct {
		src float64
	}
	tests := []struct {
		name string
		args args
		want float64
	}{
		{
			name: "floor",
			args: args{
				src: 9.9999999,
			},
			want: 10.000000,
		},
		{
			name: "ceil",
			args: args{
				src: 9.9999991,
			},
			want: 9.999999,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := floatRound(tt.args.src); got != tt.want {
				t.Errorf("floatRound() = %v, want %v", got, tt.want)
			}
		})
	}
}
