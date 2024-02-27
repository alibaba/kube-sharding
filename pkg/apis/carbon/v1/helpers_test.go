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
	"fmt"
	"io/ioutil"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	"github.com/alibaba/kube-sharding/pkg/rollalgorithm"
	assert "github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestIsWorkerUnAssigned(t *testing.T) {
	type args struct {
		worker *WorkerNode
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "TestUnAssigned",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{AllocStatus: WorkerUnAssigned}}},
			want: true,
		},
		{
			name: "TestNotUnAssigned",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{AllocStatus: WorkerAssigned}}},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsWorkerUnAssigned(tt.args.worker); got != tt.want {
				t.Errorf("IsWorkerUnAssigned() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetWorkerUnAssigned(t *testing.T) {
	type args struct {
		worker *WorkerNode
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "TestSetUnAssigned",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{}}},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetWorkerUnAssigned(tt.args.worker)
			if got := IsWorkerUnAssigned(tt.args.worker); got != tt.want {
				t.Errorf("IsWorkerUnAssigned() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsWorkerAssigned(t *testing.T) {
	type args struct {
		worker *WorkerNode
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "TestAssigned",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{AllocStatus: WorkerAssigned}}},
			want: true,
		},
		{
			name: "TestNotAssigned",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{AllocStatus: WorkerUnAssigned}}},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsWorkerAssigned(tt.args.worker); got != tt.want {
				t.Errorf("IsWorkerAssigned() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetWorkerAssigned(t *testing.T) {
	type args struct {
		worker *WorkerNode
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "TestSetAssigned",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{}}},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetWorkerAssigned(tt.args.worker)
			if got := IsWorkerAssigned(tt.args.worker); got != tt.want {
				t.Errorf("IsWorkerAssigned() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsWorkerLost(t *testing.T) {
	type args struct {
		worker *WorkerNode
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "TestLost",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{AllocStatus: WorkerLost}}},
			want: true,
		},
		{
			name: "TestNotLost",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{AllocStatus: WorkerUnAssigned}}},
			want: false,
		},
		{
			name: "TestNotLost",
			args: args{worker: &WorkerNode{}},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsWorkerLost(tt.args.worker); got != tt.want {
				t.Errorf("IsWorkerLost() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetWorkerLost(t *testing.T) {
	type args struct {
		worker *WorkerNode
	}

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "TestSetLost",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{}}},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetWorkerLost(tt.args.worker)
			if got := IsWorkerLost(tt.args.worker); got != tt.want {
				t.Errorf("IsWorkerLost() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsWorkerOfflining(t *testing.T) {
	type args struct {
		worker *WorkerNode
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "TestOfflining",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{AllocStatus: WorkerOfflining}}},
			want: true,
		},
		{
			name: "TestNotOflining",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{AllocStatus: WorkerUnAssigned}}},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsWorkerOfflining(tt.args.worker); got != tt.want {
				t.Errorf("IsWorkerOfflining() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetWorkerOfflining(t *testing.T) {
	type args struct {
		worker *WorkerNode
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "TestSetOfflining",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{}}},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetWorkerOfflining(tt.args.worker)
			if got := IsWorkerOfflining(tt.args.worker); got != tt.want {
				t.Errorf("IsWorkerOfflining() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsWorkerToRelease(t *testing.T) {
	type args struct {
		worker *WorkerNode
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "ToRelease",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{ToRelease: true}}},
			want: true,
		},
		{
			name: "NotToRelease",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{ToRelease: false}}},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsWorkerToRelease(tt.args.worker); got != tt.want {
				t.Errorf("IsWorkerToRelease() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetWorkerReleasing(t *testing.T) {
	type args struct {
		worker *WorkerNode
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "TestSetReleasing",
			args: args{worker: &WorkerNode{}},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetWorkerReleasing(tt.args.worker)
			if got := IsWorkerReleasing(tt.args.worker); got != tt.want {
				t.Errorf("IsWorkerReleasing() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsWorkerAllocReleasing(t *testing.T) {
	type args struct {
		worker *WorkerNode
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "TestAllocReleasing",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{AllocStatus: WorkerReleasing}}},
			want: true,
		},
		{
			name: "TestAllocNotReleasing",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{AllocStatus: WorkerUnAssigned}}},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsWorkerReleasing(tt.args.worker); got != tt.want {
				t.Errorf("IsWorkerReleasing() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetWorkerAllocReleasing(t *testing.T) {
	type args struct {
		worker *WorkerNode
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "TestSetAllocReleasing",
			args: args{worker: &WorkerNode{}},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetWorkerReleasing(tt.args.worker)
			if got := IsWorkerReleasing(tt.args.worker); got != tt.want {
				t.Errorf("IsWorkerReleasing() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsWorkerReleased(t *testing.T) {
	type args struct {
		worker *WorkerNode
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "TestReleased",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{AllocStatus: WorkerReleased}}},
			want: true,
		},
		{
			name: "TestNotReleased",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{AllocStatus: WorkerUnAssigned}}},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsWorkerReleased(tt.args.worker); got != tt.want {
				t.Errorf("IsWorkerReleased() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetWorkerReleased(t *testing.T) {
	type args struct {
		worker *WorkerNode
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "TestSetReleased",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{}}},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetWorkerReleased(tt.args.worker)
			if got := IsWorkerReleased(tt.args.worker); got != tt.want {
				t.Errorf("IsWorkerReleased() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsWorkerRunning(t *testing.T) {
	type args struct {
		worker *WorkerNode
	}
	var daemonPod HippoPodTemplate
	daemonPod.Spec.RestartPolicy = corev1.RestartPolicyOnFailure
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "TestRunning",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{AllocatorSyncedStatus: AllocatorSyncedStatus{Phase: Running}}}},
			want: true,
		},
		{
			name: "TerminatedRunning",
			args: args{worker: &WorkerNode{
				Spec: WorkerNodeSpec{VersionPlan: VersionPlan{
					SignedVersionPlan: SignedVersionPlan{
						Template: &daemonPod,
					},
				}},
				Status: WorkerNodeStatus{AllocatorSyncedStatus: AllocatorSyncedStatus{Phase: Terminated}}}},
			want: true,
		},
		{
			name: "TestNotRunning",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{AllocatorSyncedStatus: AllocatorSyncedStatus{Phase: Pending}}}},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsWorkerRunning(tt.args.worker); got != tt.want {
				t.Errorf("IsWorkerRunning() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsWorkerHealth(t *testing.T) {
	type args struct {
		worker *WorkerNode
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "TestHealth",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{HealthStatus: HealthAlive, AllocatorSyncedStatus: AllocatorSyncedStatus{Phase: Running}}}},
			want: true,
		},
		{
			name: "TestHealth",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{HealthStatus: HealthAlive}}},
			want: false,
		},
		{
			name: "TestHealth",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{AllocatorSyncedStatus: AllocatorSyncedStatus{Phase: Running}}}},
			want: false,
		},
		{
			name: "TestNotHealth",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{}}},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsWorkerHealth(tt.args.worker); got != tt.want {
				t.Errorf("IsWorkerHealth() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsWorkerDead(t *testing.T) {
	type args struct {
		worker *WorkerNode
	}
	var daemonPod HippoPodTemplate
	daemonPod.Spec.RestartPolicy = corev1.RestartPolicyAlways
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "TestTerminateDaemon",
			args: args{worker: &WorkerNode{
				Spec: WorkerNodeSpec{VersionPlan: VersionPlan{
					SignedVersionPlan: SignedVersionPlan{
						Template: &daemonPod,
					}}},
				Status: WorkerNodeStatus{AllocatorSyncedStatus: AllocatorSyncedStatus{Phase: Terminated}}}},
			want: true,
		},
		{
			name: "TestTestTerminate",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{AllocatorSyncedStatus: AllocatorSyncedStatus{Phase: Terminated}}}},
			want: false,
		},
		{
			name: "TestDead",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{AllocatorSyncedStatus: AllocatorSyncedStatus{Phase: Failed}}}},
			want: true,
		},
		{
			name: "TestDead",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{AllocatorSyncedStatus: AllocatorSyncedStatus{Phase: Unknown}}}},
			want: false,
		},
		{
			name: "TestDead",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{HealthStatus: HealthDead}}},
			want: true,
		},
		{
			name: "TestNotDead",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{}}},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsWorkerDead(tt.args.worker); got != tt.want {
				t.Errorf("IsWorkerDead() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsWorkerLongtimeNotMatch(t *testing.T) {
	type args struct {
		worker *WorkerNode
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "not match",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{AllocatorSyncedStatus: AllocatorSyncedStatus{LastResourceNotMatchtime: time.Now().Unix() - 6}},
				Spec: WorkerNodeSpec{
					Version: "aaa",
					VersionPlan: VersionPlan{
						BroadcastPlan: BroadcastPlan{
							WorkerSchedulePlan: WorkerSchedulePlan{ResourceMatchTimeout: 5},
						}}}}},
			want: true,
		},
		{
			name: "match",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{AllocatorSyncedStatus: AllocatorSyncedStatus{LastResourceNotMatchtime: time.Now().Unix() - 6}},
				Spec: WorkerNodeSpec{VersionPlan: VersionPlan{
					BroadcastPlan: BroadcastPlan{
						WorkerSchedulePlan: WorkerSchedulePlan{ResourceMatchTimeout: 5},
					}}}}},
			want: false,
		},
		{
			name: "match",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{AllocatorSyncedStatus: AllocatorSyncedStatus{LastResourceNotMatchtime: time.Now().Unix() - 6}},
				Spec: WorkerNodeSpec{VersionPlan: VersionPlan{
					SignedVersionPlan: SignedVersionPlan{
						RestartAfterResourceChange: utils.BoolPtr(true),
					},
					BroadcastPlan: BroadcastPlan{
						WorkerSchedulePlan: WorkerSchedulePlan{ResourceMatchTimeout: 5},
					}}}}},
			want: false,
		},
		{
			name: "match",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{AllocatorSyncedStatus: AllocatorSyncedStatus{LastResourceNotMatchtime: time.Now().Unix()}},
				Spec: WorkerNodeSpec{VersionPlan: VersionPlan{
					BroadcastPlan: BroadcastPlan{
						WorkerSchedulePlan: WorkerSchedulePlan{ResourceMatchTimeout: 5},
					}}}}},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsWorkerLongtimeNotMatch(tt.args.worker, ""); got != tt.want {
				t.Errorf("IsWorkerLongtimeNotMatch() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsWorkerLongtimeNotReady(t *testing.T) {
	type args struct {
		worker *WorkerNode
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "TestLongtimeNotReady",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{LastWorkerNotReadytime: time.Now().Unix() - 6},
				Spec: WorkerNodeSpec{VersionPlan: VersionPlan{
					BroadcastPlan: BroadcastPlan{WorkerSchedulePlan: WorkerSchedulePlan{WorkerReadyTimeout: 5}}}}}},
			want: true,
		},
		{
			name: "TestLongtimeNotReady",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{LastWorkerNotReadytime: time.Now().Unix() - 3},
				Spec: WorkerNodeSpec{VersionPlan: VersionPlan{
					BroadcastPlan: BroadcastPlan{WorkerSchedulePlan: WorkerSchedulePlan{WorkerReadyTimeout: 5}}}}}},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsWorkerLongtimeNotReady(tt.args.worker); got != tt.want {
				t.Errorf("IsWorkerLongtimeNotReady() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsWorkerProcessLongtimeNotReady(t *testing.T) {
	type args struct {
		worker *WorkerNode
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "TestLongtimeNotReady",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{
				AllocatorSyncedStatus: AllocatorSyncedStatus{LastProcessNotReadytime: time.Now().Unix() - 6}},
				Spec: WorkerNodeSpec{VersionPlan: VersionPlan{
					BroadcastPlan: BroadcastPlan{WorkerSchedulePlan: WorkerSchedulePlan{ProcessMatchTimeout: 5}}}}}},
			want: true,
		},
		{
			name: "TestLongtimeNotReady",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{
				AllocatorSyncedStatus: AllocatorSyncedStatus{LastProcessNotReadytime: time.Now().Unix() - 3}},
				Spec: WorkerNodeSpec{VersionPlan: VersionPlan{
					BroadcastPlan: BroadcastPlan{WorkerSchedulePlan: WorkerSchedulePlan{ProcessMatchTimeout: 5}}}}}},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsWorkerProcessLongtimeNotReady(tt.args.worker); got != tt.want {
				t.Errorf("IsWorkerProcessLongtimeNotReady() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsWorkerBroken(t *testing.T) {
	type args struct {
		worker *WorkerNode
	}
	tests := []struct {
		name string
		args args
		want BadReasonType
	}{
		{
			name: "TestLongtimeNotReady",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{AllocStatus: WorkerAssigned, LastWorkerNotReadytime: time.Now().Unix() - 6},
				Spec: WorkerNodeSpec{VersionPlan: VersionPlan{
					BroadcastPlan: BroadcastPlan{WorkerSchedulePlan: WorkerSchedulePlan{WorkerReadyTimeout: 5}}}}}},
			want: 5,
		},
		{
			name: "TestLongtimeNotMatch",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{AllocStatus: WorkerAssigned, AllocatorSyncedStatus: AllocatorSyncedStatus{LastResourceNotMatchtime: time.Now().Unix() - 6}},
				Spec: WorkerNodeSpec{
					Version: "aaa",
					VersionPlan: VersionPlan{
						BroadcastPlan: BroadcastPlan{WorkerSchedulePlan: WorkerSchedulePlan{ResourceMatchTimeout: 5}}}}}},
			want: 3,
		},
		{
			name: "TestLost",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{AllocStatus: WorkerLost}}},
			want: 1,
		},
		{
			name: "TestDead",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{AllocStatus: WorkerAssigned, HealthStatus: HealthDead}}},
			want: 2,
		},
		{
			name: "TestNotBroker",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{}}},
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsWorkerBroken(tt.args.worker, ""); got != tt.want {
				t.Errorf("IsWorkerBroken() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetWorkerBadState(t *testing.T) {
	type args struct {
		worker *WorkerNode
	}
	tests := []struct {
		name string
		args args
		want BadReasonType
	}{
		{
			name: "TestLongtimeNotReady",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{AllocStatus: WorkerAssigned, LastWorkerNotReadytime: time.Now().Unix() - 6},
				Spec: WorkerNodeSpec{VersionPlan: VersionPlan{
					BroadcastPlan: BroadcastPlan{WorkerSchedulePlan: WorkerSchedulePlan{WorkerReadyTimeout: 5}}}}}},
			want: BadReasonNotReady,
		},
		{
			name: "TestLongtimeNotMatch",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{AllocStatus: WorkerAssigned, AllocatorSyncedStatus: AllocatorSyncedStatus{LastResourceNotMatchtime: time.Now().Unix() - 6}},
				Spec: WorkerNodeSpec{
					Version: "aaa",
					VersionPlan: VersionPlan{
						BroadcastPlan: BroadcastPlan{WorkerSchedulePlan: WorkerSchedulePlan{ResourceMatchTimeout: 5}}}}}},
			want: BadReasonNotMatch,
		},
		{
			name: "TestLost",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{AllocStatus: WorkerLost}}},
			want: BadReasonLost,
		},
		{
			name: "TestDead",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{AllocStatus: WorkerAssigned, HealthStatus: HealthDead}}},
			want: BadReasonDead,
		},
		{
			name: "TestOfflining",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{AllocStatus: WorkerOfflining}}},
			want: BadReasonNone,
		},
		{
			name: "TestDead",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{AllocStatus: WorkerReleased}}},
			want: BadReasonRelease,
		},
		{
			name: "TestNotBad",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{}}},
			want: BadReasonNone,
		},
		{
			name: "TestServiceDetail",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{
				AllocStatus: WorkerAssigned,
				ServiceConditions: []ServiceCondition{
					{
						Type:          ServiceGeneral,
						ServiceDetail: "[{\"needBackup\":true}]",
					},
				},
			}}},
			want: BadReasonServiceReclaim,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetWorkerBadState(tt.args.worker, ""); got != tt.want {
				t.Errorf("GetWorkerBadState() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsWorkerUnAvaliable(t *testing.T) {
	type args struct {
		worker *WorkerNode
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "TestUnAvaliable",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{ServiceStatus: ServiceUnAvailable}}},
			want: true,
		},
		{
			name: "TestUnAvaliable",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{ServiceStatus: ServiceUnKnown}}},
			want: false,
		},
		{
			name: "TestAvaliable",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{ServiceStatus: ServiceAvailable}}},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsWorkerUnAvaliable(tt.args.worker); got != tt.want {
				t.Errorf("IsWorkerUnAvaliable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCouldDeletePod(t *testing.T) {
	type args struct {
		worker *WorkerNode
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "TestUnAvaliable",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{ServiceStatus: ServiceUnAvailable}}},
			want: true,
		},
		{
			name: "TestUngraceful",
			args: args{worker: &WorkerNode{
				Spec: WorkerNodeSpec{VersionPlan: VersionPlan{
					BroadcastPlan: BroadcastPlan{
						UpdatingGracefully: utils.BoolPtr(false),
					},
				}},
				Status: WorkerNodeStatus{
					ServiceStatus: ServiceUnKnown,
					ToRelease:     true,
				}}},
			want: true,
		},
		{
			name: "TestUngraceful",
			args: args{worker: &WorkerNode{
				Spec: WorkerNodeSpec{VersionPlan: VersionPlan{
					BroadcastPlan: BroadcastPlan{
						UpdatingGracefully: utils.BoolPtr(false),
					},
				}},
				Status: WorkerNodeStatus{
					ServiceStatus: ServiceUnKnown,
				}}},
			want: false,
		},
		{
			name: "TestUngraceful",
			args: args{worker: &WorkerNode{
				Spec: WorkerNodeSpec{VersionPlan: VersionPlan{}},
				Status: WorkerNodeStatus{
					ServiceStatus: ServiceUnKnown,
					ToRelease:     true,
				}}},
			want: false,
		},
		{
			name: "TestUnAvaliable",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{ServiceStatus: ServiceUnKnown}}},
			want: false,
		},
		{
			name: "TestAvaliable",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{ServiceStatus: ServiceAvailable}}},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CouldDeletePod(tt.args.worker); got != tt.want {
				t.Errorf("IsWorkerUnAvaliable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsWorkerComplete(t *testing.T) {
	type args struct {
		worker *WorkerNode
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "TestComplete",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{

				ServiceStatus: ServiceAvailable,
				AllocStatus:   WorkerAssigned,
				HealthStatus:  HealthAlive,

				HealthCondition: HealthCondition{WorkerStatus: WorkerTypeReady},
				AllocatorSyncedStatus: AllocatorSyncedStatus{
					ProcessMatch:  true,
					ResourceMatch: true,
					Phase:         Running,
					ProcessReady:  true,
				},
			}}},
			want: true,
		},
		{
			name: "TestNotComplete",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{

				ServiceStatus: ServiceAvailable,
				AllocStatus:   WorkerAssigned,
				HealthStatus:  HealthAlive,

				HealthCondition: HealthCondition{WorkerStatus: WorkerTypeNotReady},
				AllocatorSyncedStatus: AllocatorSyncedStatus{
					ProcessMatch:  true,
					ResourceMatch: true,
					Phase:         Running,
					ProcessReady:  true,
				},
			}}},
			want: false,
		},
		{
			name: "TestNotComplete",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{

				ServiceStatus: ServiceAvailable,
				AllocStatus:   WorkerAssigned,
				HealthStatus:  HealthAlive,
				AllocatorSyncedStatus: AllocatorSyncedStatus{
					ProcessMatch: true,
				},
			}}},
			want: false,
		},
		{
			name: "TestNotComplete",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{

				ServiceStatus: ServiceAvailable,
				AllocStatus:   WorkerAssigned,
				HealthStatus:  HealthAlive,

				AllocatorSyncedStatus: AllocatorSyncedStatus{
					ProcessMatch: true,
					Phase:        Running,
					Version:      "1",
				},
			}}},
			want: false,
		},
		{
			name: "TestNotComplete",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{

				ServiceStatus: ServiceAvailable,
				AllocStatus:   WorkerAssigned,

				AllocatorSyncedStatus: AllocatorSyncedStatus{
					ProcessMatch: true,
					Phase:        Running,
				},
			}}},
			want: false,
		},
		{
			name: "TestNotComplete",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{

				AllocStatus:  WorkerAssigned,
				HealthStatus: HealthAlive,

				AllocatorSyncedStatus: AllocatorSyncedStatus{
					ProcessMatch: true,
					Phase:        Running,
				},
			}}},
			want: false,
		},
		{
			name: "TestNotComplete",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{
				ServiceStatus: ServiceAvailable,
				AllocStatus:   WorkerAssigned,
				HealthStatus:  HealthAlive,

				AllocatorSyncedStatus: AllocatorSyncedStatus{

					Phase: Running,
				},
			}}},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsWorkerComplete(tt.args.worker); got != tt.want {
				t.Errorf("IsWorkerComplete() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAllocStatusScore(t *testing.T) {
	type args struct {
		status AllocStatus
	}
	tests := []struct {
		name string
		args args
		want int32
	}{
		{
			name: "TestLostScore",
			args: args{status: WorkerLost},
			want: 0,
		},
		{
			name: "TestLostReleased",
			args: args{status: WorkerReleased},
			want: 1,
		},
		{
			name: "TestOffliningScore",
			args: args{status: WorkerOfflining},
			want: 2,
		},
		{
			name: "TestReleasingScore",
			args: args{status: WorkerReleasing},
			want: 3,
		},
		{
			name: "TestUnAssignedScore",
			args: args{status: WorkerUnAssigned},
			want: 4,
		},
		{
			name: "TestAssignedScore",
			args: args{status: WorkerAssigned},
			want: 5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AllocStatusScore(tt.args.status); got != tt.want {
				t.Errorf("AllocStatusScore() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPhaseScore(t *testing.T) {
	type args struct {
		status WorkerPhase
	}
	tests := []struct {
		name string
		args args
		want int32
	}{
		{
			name: "TestTerminatedScore",
			args: args{status: Terminated},
			want: 0,
		},
		{
			name: "TestFailedReleased",
			args: args{status: Failed},
			want: 1,
		},
		{
			name: "TestUnknownScore",
			args: args{status: Unknown},
			want: 2,
		},
		{
			name: "TestPendingScore",
			args: args{status: Pending},
			want: 3,
		},
		{
			name: "TestRunningScore",
			args: args{status: Running},
			want: 4,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PhaseScore(tt.args.status); got != tt.want {
				t.Errorf("PhaseScore() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestServiceScore(t *testing.T) {
	type args struct {
		worker *WorkerNode
	}
	tests := []struct {
		name string
		args args
		want int32
	}{
		{
			name: "TestServiceUnAvailableScore",
			args: args{
				worker: &WorkerNode{
					Status: WorkerNodeStatus{
						ServiceStatus: ServiceUnAvailable,
					},
				},
			},
			want: 0,
		},
		{
			name: "TestServiceUnKnownScore",
			args: args{
				worker: &WorkerNode{
					Status: WorkerNodeStatus{
						ServiceStatus: ServiceUnKnown,
					},
				},
			},
			want: 1,
		},
		{
			name: "TestServicePartAvailableScore",
			args: args{
				worker: &WorkerNode{
					Status: WorkerNodeStatus{
						ServiceStatus: ServicePartAvailable,
					},
				},
			},
			want: 2,
		},
		{
			name: "TestServiceAvailableScore",
			args: args{
				worker: &WorkerNode{
					Status: WorkerNodeStatus{
						ServiceStatus: ServiceAvailable,
					},
				},
			},
			want: 3,
		},
		{
			name: "TestServiceUnKnownScore",
			args: args{
				worker: &WorkerNode{
					Status: WorkerNodeStatus{
						ServiceStatus: ServiceUnKnown,
						ServiceConditions: []ServiceCondition{
							ServiceCondition{
								ServiceName: "test1",
								Status:      "True",
								Score:       1,
							},
						},
					},
				},
			},
			want: 2,
		},
		{
			name: "TestServicePartAvailableScore",
			args: args{
				worker: &WorkerNode{
					Status: WorkerNodeStatus{
						ServiceStatus: ServicePartAvailable,
						ServiceConditions: []ServiceCondition{
							ServiceCondition{
								ServiceName: "test1",
								Status:      "True",
								Score:       -1,
							},
						},
					},
				},
			},
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ServiceScore(tt.args.worker); got != tt.want {
				t.Errorf("ServiceScore() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHealthScore(t *testing.T) {
	type args struct {
		status HealthStatus
	}
	tests := []struct {
		name string
		args args
		want int32
	}{
		{
			name: "TestHealthDeadScore",
			args: args{status: HealthDead},
			want: 0,
		},
		{
			name: "TestHealthUnKnownScore",
			args: args{status: HealthUnKnown},
			want: 1,
		},
		{
			name: "TestHealthLostScore",
			args: args{status: HealthLost},
			want: 2,
		},
		{
			name: "TestHealthAliveScore",
			args: args{status: HealthAlive},
			want: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HealthScore(tt.args.status); got != tt.want {
				t.Errorf("HealthScore() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWorkerPorcessScore(t *testing.T) {
	type args struct {
		worker *WorkerNode
	}
	tests := []struct {
		name string
		args args
		want int32
	}{
		{
			name: "normal",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{
				AllocStatus:   WorkerAssigned,
				ServiceStatus: ServicePartAvailable,
				HealthStatus:  HealthAlive,
				AllocatorSyncedStatus: AllocatorSyncedStatus{
					Phase:        Running,
					ProcessScore: 5,
				},
			}}},
			want: 5,
		},
		{
			name: "reclaim",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{
				AllocStatus:   WorkerAssigned,
				ServiceStatus: ServicePartAvailable,
				HealthStatus:  HealthAlive,
				AllocatorSyncedStatus: AllocatorSyncedStatus{
					Phase:        Running,
					ProcessScore: -1,
				},
			}}},
			want: -1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := WorkerProcessScore(tt.args.worker); got != tt.want {
				t.Errorf("WorkerProcessScore() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWorkerAvailableScore(t *testing.T) {
	type args struct {
		worker *WorkerNode
	}
	tests := []struct {
		name string
		args args
		want int32
	}{
		{
			name: "not available",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{
				AllocStatus:   WorkerAssigned,
				ServiceStatus: ServicePartAvailable,
				HealthStatus:  HealthAlive,
				AllocatorSyncedStatus: AllocatorSyncedStatus{
					Phase:        Running,
					ProcessScore: 5,
				},
			}}},
			want: 0,
		},
		{
			name: "available",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{
				AllocStatus:   WorkerAssigned,
				ServiceStatus: ServiceAvailable,
				HealthStatus:  HealthAlive,
				AllocatorSyncedStatus: AllocatorSyncedStatus{
					Phase:        Running,
					ProcessScore: -1,
				},
			}}},
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := WorkerAvailableScore(tt.args.worker); got != tt.want {
				t.Errorf("WorkerAvailableScore() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWorkerReclaimScore(t *testing.T) {
	type args struct {
		worker *WorkerNode
	}
	tests := []struct {
		name string
		args args
		want int32
	}{
		{
			name: "reclaim",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{
				AllocStatus:   WorkerAssigned,
				ServiceStatus: ServicePartAvailable,
				HealthStatus:  HealthAlive,
				AllocatorSyncedStatus: AllocatorSyncedStatus{
					Phase:        Running,
					ProcessScore: 5,
					Reclaim:      true,
				},
			}}},
			want: 0,
		},
		{
			name: "not reclaim",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{
				AllocStatus:   WorkerAssigned,
				ServiceStatus: ServiceAvailable,
				HealthStatus:  HealthAlive,
				AllocatorSyncedStatus: AllocatorSyncedStatus{
					Phase:        Running,
					ProcessScore: -1,
				},
			}}},
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := WorkerReclaimScore(tt.args.worker); got != tt.want {
				t.Errorf("WorkerReclaimScore() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWorkerReleaseScore(t *testing.T) {
	type args struct {
		worker *WorkerNode
	}
	tests := []struct {
		name string
		args args
		want int32
	}{
		{
			name: "release",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{
				AllocStatus:   WorkerAssigned,
				ServiceStatus: ServicePartAvailable,
				HealthStatus:  HealthAlive,
				AllocatorSyncedStatus: AllocatorSyncedStatus{
					Phase:        Running,
					ProcessScore: 5,
					Reclaim:      true,
				},
				ToRelease: true,
			}}},
			want: 0,
		},
		{
			name: "not release",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{
				AllocStatus:   WorkerAssigned,
				ServiceStatus: ServiceAvailable,
				HealthStatus:  HealthAlive,
				AllocatorSyncedStatus: AllocatorSyncedStatus{
					Phase:        Running,
					ProcessScore: -1,
				},
			}}},
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := WorkerReleaseScore(tt.args.worker); got != tt.want {
				t.Errorf("WorkerReleaseScore() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWorkerReadyScore(t *testing.T) {
	type args struct {
		worker *WorkerNode
	}
	tests := []struct {
		name string
		args args
		want int32
	}{
		{
			name: "ready",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{
				AllocStatus:   WorkerAssigned,
				ServiceStatus: ServicePartAvailable,
				HealthStatus:  HealthAlive,
				AllocatorSyncedStatus: AllocatorSyncedStatus{
					Phase:        Running,
					ProcessScore: 5,
					Reclaim:      true,
				},
				ToRelease: true,
				HealthCondition: HealthCondition{
					WorkerStatus: WorkerTypeReady,
				},
			}}},
			want: 1,
		},
		{
			name: "not ready",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{
				AllocStatus:   WorkerAssigned,
				ServiceStatus: ServiceAvailable,
				HealthStatus:  HealthAlive,
				AllocatorSyncedStatus: AllocatorSyncedStatus{
					Phase:        Running,
					ProcessScore: -1,
				},
				HealthCondition: HealthCondition{
					WorkerStatus: WorkerTypeNotReady,
				},
			}}},
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := WorkerReadyScore(tt.args.worker); got != tt.want {
				t.Errorf("WorkerReadyScore() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBaseScoreOfWorker(t *testing.T) {
	type args struct {
		worker *WorkerNode
	}
	tests := []struct {
		name string
		args args
		want int64
	}{
		{
			name: "complete",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{
				AllocStatus:   WorkerAssigned,
				ServiceStatus: ServiceAvailable,
				HealthStatus:  HealthAlive,
				WorkerReady:   true,
				AllocatorSyncedStatus: AllocatorSyncedStatus{
					Phase:        Running,
					ProcessScore: 5,
					Reclaim:      false,
					ProcessMatch: true,
					ProcessReady: true,
				},
				HealthCondition: HealthCondition{
					WorkerStatus: WorkerTypeReady,
				},
				WorkerStateChangeRecoder: WorkerStateChangeRecoder{
					LastServiceReadyTime: utils.Int64Ptr(time.Now().Unix() - 40),
				},
				ToRelease: false,
			}}},
			want: 1113134515500,
		},
		{
			name: "to release",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{
				AllocStatus:   WorkerAssigned,
				ServiceStatus: ServiceAvailable,
				HealthStatus:  HealthAlive,
				AllocatorSyncedStatus: AllocatorSyncedStatus{
					Phase:        Running,
					ProcessScore: 5,
					Reclaim:      false,
				},
				HealthCondition: HealthCondition{
					WorkerStatus: WorkerTypeReady,
				},
				ToRelease: true,
			}}},
			want: 13134515500,
		},
		{
			name: "reclaim",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{
				AllocStatus:   WorkerAssigned,
				ServiceStatus: ServiceAvailable,
				HealthStatus:  HealthAlive,
				AllocatorSyncedStatus: AllocatorSyncedStatus{
					Phase:        Running,
					ProcessScore: -1,
					Reclaim:      true,
				},
				HealthCondition: HealthCondition{
					WorkerStatus: WorkerTypeReady,
				},
				ToRelease: false,
			}}},
			want: 1013134499500,
		},
		{
			name: "notready",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{
				AllocStatus:   WorkerAssigned,
				ServiceStatus: ServiceAvailable,
				HealthStatus:  HealthAlive,
				AllocatorSyncedStatus: AllocatorSyncedStatus{
					Phase:        Running,
					ProcessScore: 5,
					Reclaim:      false,
				},
				HealthCondition: HealthCondition{
					WorkerStatus: WorkerTypeNotReady,
				},
				ToRelease: false,
			}}},
			want: 1013034515500,
		},
		{
			name: "raiseServicePriority",
			args: args{worker: &WorkerNode{
				Spec: WorkerNodeSpec{
					Version: "version1",
				},
				Status: WorkerNodeStatus{
					AllocStatus:   WorkerAssigned,
					ServiceStatus: ServiceAvailable,
					HealthStatus:  HealthAlive,
					AllocatorSyncedStatus: AllocatorSyncedStatus{
						Phase:        Running,
						ProcessScore: 5,
						Reclaim:      false,
						Version:      "version2",
					},

					ServiceConditions: []ServiceCondition{
						{
							Type:  ServiceGeneral,
							Score: 3,
						},
					},
					HealthCondition: HealthCondition{
						WorkerStatus: WorkerTypeNotReady,
					},
					ToRelease: false,
				}}},
			want: 1601034515500,
		},
		{
			name: "nothealth",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{
				AllocStatus:   WorkerAssigned,
				ServiceStatus: ServiceAvailable,
				HealthStatus:  HealthLost,
				AllocatorSyncedStatus: AllocatorSyncedStatus{
					Phase:        Running,
					ProcessScore: 5,
					Reclaim:      false,
				},
				HealthCondition: HealthCondition{
					WorkerStatus: WorkerTypeReady,
				},
				ToRelease: false,
			}}},
			want: 1013124515500,
		},
		{
			name: "notavailable",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{
				AllocStatus:   WorkerAssigned,
				ServiceStatus: ServicePartAvailable,
				HealthStatus:  HealthAlive,
				AllocatorSyncedStatus: AllocatorSyncedStatus{
					Phase:        Running,
					ProcessScore: 5,
					Reclaim:      false,
				},
				HealthCondition: HealthCondition{
					WorkerStatus: WorkerTypeReady,
				},
				ToRelease: false,
			}}},
			want: 1002134515500,
		},
		{
			name: "failed",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{
				AllocStatus:   WorkerAssigned,
				ServiceStatus: ServiceAvailable,
				HealthStatus:  HealthAlive,
				AllocatorSyncedStatus: AllocatorSyncedStatus{
					Phase:        Failed,
					ProcessScore: 5,
					Reclaim:      false,
				},
				HealthCondition: HealthCondition{
					WorkerStatus: WorkerTypeReady,
				},
				ToRelease: false,
			}}},
			want: 1013131515500,
		},
		{
			name: "lost",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{
				AllocStatus:   WorkerLost,
				ServiceStatus: ServiceAvailable,
				HealthStatus:  HealthAlive,
				AllocatorSyncedStatus: AllocatorSyncedStatus{
					Phase:        Running,
					ProcessScore: 5,
					Reclaim:      false,
				},
				HealthCondition: HealthCondition{
					WorkerStatus: WorkerTypeReady,
				},
				ToRelease: false,
			}}},
			want: 1013134015500,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BaseScoreOfWorker(tt.args.worker); got != tt.want {
				t.Errorf("BaseScoreOfWorker() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsReplicaUnAssigned(t *testing.T) {
	type args struct {
		replica *Replica
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "TestAssigned",
			args: args{replica: &Replica{
				WorkerNode: WorkerNode{Status: WorkerNodeStatus{AllocStatus: WorkerAssigned}}}},
			want: false,
		},
		{
			name: "TestNotAssigned",
			args: args{replica: &Replica{WorkerNode: WorkerNode{Status: WorkerNodeStatus{AllocStatus: WorkerUnAssigned}}}},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsReplicaUnAssigned(tt.args.replica); got != tt.want {
				t.Errorf("IsReplicaUnAssigned() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsReplicaAssigned(t *testing.T) {
	type args struct {
		replica *Replica
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "TestAssigned",
			args: args{replica: &Replica{
				WorkerNode: WorkerNode{Status: WorkerNodeStatus{AllocStatus: WorkerAssigned}}}},
			want: true,
		},
		{
			name: "TestNotAssigned",
			args: args{replica: &Replica{
				WorkerNode: WorkerNode{Status: WorkerNodeStatus{AllocStatus: WorkerUnAssigned}}}},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsReplicaAssigned(tt.args.replica); got != tt.want {
				t.Errorf("IsReplicaAssigned() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsReplicaLost(t *testing.T) {
	type args struct {
		replica *Replica
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "TestLost",
			args: args{replica: &Replica{
				WorkerNode: WorkerNode{Status: WorkerNodeStatus{AllocStatus: WorkerLost}}}},
			want: true,
		},
		{
			name: "TestNotLost",
			args: args{replica: &Replica{
				WorkerNode: WorkerNode{Status: WorkerNodeStatus{AllocStatus: WorkerUnAssigned}}}},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsReplicaLost(tt.args.replica); got != tt.want {
				t.Errorf("IsReplicaLost() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsReplicaOfflining(t *testing.T) {
	type args struct {
		replica *Replica
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "TestOfflining",
			args: args{replica: &Replica{
				WorkerNode: WorkerNode{Status: WorkerNodeStatus{AllocStatus: WorkerOfflining}}}},
			want: true,
		},
		{
			name: "TestNotOfflining",
			args: args{replica: &Replica{
				WorkerNode: WorkerNode{Status: WorkerNodeStatus{AllocStatus: WorkerUnAssigned}}}},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsReplicaOfflining(tt.args.replica); got != tt.want {
				t.Errorf("IsReplicaOfflining() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsReplicaReleasing(t *testing.T) {
	type args struct {
		replica *Replica
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "TestReleasing",
			args: args{replica: &Replica{
				WorkerNode: WorkerNode{Status: WorkerNodeStatus{ToRelease: true}}}},
			want: true,
		},
		{
			name: "TestNotReleasing",
			args: args{replica: &Replica{
				WorkerNode: WorkerNode{Status: WorkerNodeStatus{ToRelease: false}}}},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsReplicaReleasing(tt.args.replica); got != tt.want {
				t.Errorf("IsReplicaReleasing() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsReplicaAllocReleasing(t *testing.T) {
	type args struct {
		replica *Replica
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "TestReleasing",
			args: args{replica: &Replica{
				WorkerNode: WorkerNode{Status: WorkerNodeStatus{AllocStatus: WorkerReleasing}}}},
			want: true,
		},
		{
			name: "TestNotReleasing",
			args: args{replica: &Replica{
				WorkerNode: WorkerNode{Status: WorkerNodeStatus{AllocStatus: WorkerUnAssigned}}}},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsReplicaAllocReleasing(tt.args.replica); got != tt.want {
				t.Errorf("IsReplicaAllocReleasing() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsWorkerReclaiming(t *testing.T) {
	type args struct {
		worker *WorkerNode
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Reclaiming on status",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{AllocStatus: WorkerAssigned, AllocatorSyncedStatus: AllocatorSyncedStatus{Reclaim: true}}}},
			want: true,
		},
		{
			name: "Reclaiming on spec",
			args: args{
				worker: &WorkerNode{
					Status: WorkerNodeStatus{
						AllocStatus: WorkerAssigned,
					},
					Spec: WorkerNodeSpec{
						Reclaim: true,
					},
				},
			},
			want: true,
		},
		{
			name: "Reclaiming on spec && status",
			args: args{
				worker: &WorkerNode{
					Status: WorkerNodeStatus{

						AllocStatus: WorkerAssigned,
						AllocatorSyncedStatus: AllocatorSyncedStatus{
							Reclaim: true,
						},
					},
					Spec: WorkerNodeSpec{
						Reclaim: true,
					},
				},
			},
			want: true,
		},
		{
			name: "Unassigned Reclaiming",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{AllocatorSyncedStatus: AllocatorSyncedStatus{Reclaim: true}}}},
			want: true,
		},
		{
			name: "Unassigned NotReclaiming",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{}, Spec: WorkerNodeSpec{Reclaim: true}}},
			want: true,
		},
		{
			name: "NotReclaiming",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{AllocStatus: WorkerAssigned}}},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsWorkerReclaiming(tt.args.worker); got != tt.want {
				t.Errorf("IsWorkerReclaiming() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsWorkerHealthMatch(t *testing.T) {
	type args struct {
		worker *WorkerNode
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Match",
			args: args{
				worker: &WorkerNode{
					Status: WorkerNodeStatus{
						HealthStatus:    HealthAlive,
						HealthCondition: HealthCondition{WorkerStatus: WorkerTypeReady},

						AllocatorSyncedStatus: AllocatorSyncedStatus{
							Phase:        Running,
							ProcessReady: true,
						},
					},
				},
			},
			want: true,
		},
		{
			name: "NotMatch",
			args: args{
				worker: &WorkerNode{
					Status: WorkerNodeStatus{
						HealthStatus: HealthAlive,

						HealthCondition: HealthCondition{
							Version: "1",
						},
						AllocatorSyncedStatus: AllocatorSyncedStatus{
							Phase: Running,
						},
					},
				},
			},
			want: false,
		},
		{
			name: "NotMatch",
			args: args{
				worker: &WorkerNode{
					Status: WorkerNodeStatus{
						HealthStatus: HealthAlive,
					},
				},
			},
			want: false,
		},
		{
			name: "NotMatch",
			args: args{
				worker: &WorkerNode{
					Status: WorkerNodeStatus{

						AllocatorSyncedStatus: AllocatorSyncedStatus{
							Phase: Running,
						},
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsWorkerHealthMatch(tt.args.worker); got != tt.want {
				t.Errorf("IsWorkerHealthMatch() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsWorkerHealthInfoReady(t *testing.T) {
	type args struct {
		worker *WorkerNode
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Ready",
			args: args{
				worker: &WorkerNode{
					Status: WorkerNodeStatus{
						AllocStatus: WorkerAssigned,

						HealthStatus: HealthAlive,

						HealthCondition: HealthCondition{WorkerStatus: WorkerTypeReady},
						AllocatorSyncedStatus: AllocatorSyncedStatus{
							Phase:        Running,
							ProcessMatch: true,
							ProcessReady: true,
						},
					},
				},
			},
			want: true,
		},
		{
			name: "Ready",
			args: args{
				worker: &WorkerNode{
					Status: WorkerNodeStatus{
						AllocStatus:  WorkerAssigned,
						HealthStatus: HealthAlive,
						AllocatorSyncedStatus: AllocatorSyncedStatus{
							Phase:        Running,
							ProcessReady: true,
							ProcessMatch: true,
						},
					},
				},
			},
			want: true,
		},
		{
			name: "WorketTypeNotReady",
			args: args{
				worker: &WorkerNode{
					Status: WorkerNodeStatus{
						AllocStatus:     WorkerAssigned,
						HealthStatus:    HealthAlive,
						HealthCondition: HealthCondition{WorkerStatus: WorkerTypeNotReady},
						AllocatorSyncedStatus: AllocatorSyncedStatus{
							Phase:        Running,
							ProcessReady: true,
							ProcessMatch: true,
						},
					},
				},
			},
			want: false,
		},
		{
			name: "NotReady",
			args: args{
				worker: &WorkerNode{
					Status: WorkerNodeStatus{
						AllocStatus: WorkerAssigned,

						HealthStatus: HealthAlive,

						HealthCondition: HealthCondition{
							Version: "1",
						},
						AllocatorSyncedStatus: AllocatorSyncedStatus{
							ResourceMatch: true,
							Phase:         Running,
						},
					},
				},
			},
			want: false,
		},
		{
			name: "NotReady",
			args: args{
				worker: &WorkerNode{
					Status: WorkerNodeStatus{

						HealthStatus: HealthAlive,

						AllocatorSyncedStatus: AllocatorSyncedStatus{
							ResourceMatch: true,
							Phase:         Running,
						},
					},
				},
			},
			want: false,
		},
		{
			name: "NotReady",
			args: args{
				worker: &WorkerNode{
					Status: WorkerNodeStatus{
						AllocStatus:  WorkerAssigned,
						HealthStatus: HealthAlive,

						AllocatorSyncedStatus: AllocatorSyncedStatus{
							Phase: Running,
						},
					},
				},
			},
			want: false,
		},
		{
			name: "NotReady",
			args: args{
				worker: &WorkerNode{
					Status: WorkerNodeStatus{
						AllocStatus: WorkerAssigned,
						AllocatorSyncedStatus: AllocatorSyncedStatus{
							ResourceMatch: true,
							Phase:         Running,
						},
					},
				},
			},
			want: false,
		},
		{
			name: "NotReady",
			args: args{
				worker: &WorkerNode{
					Status: WorkerNodeStatus{
						AllocStatus: WorkerAssigned,

						HealthStatus: HealthAlive,
						AllocatorSyncedStatus: AllocatorSyncedStatus{
							ResourceMatch: true,
						},
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsWorkerHealthInfoReady(tt.args.worker); got != tt.want {
				t.Errorf("IsWorkerHealthInfoReady() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsWorkerEmpty(t *testing.T) {
	type args struct {
		worker *WorkerNode
	}
	tests := []struct {
		name string
		args args
		want bool
	}{

		{
			name: "Empty",
			args: args{
				worker: &WorkerNode{
					Status: WorkerNodeStatus{},
				},
			},
			want: true,
		},
		{
			name: "NotEmpty",
			args: args{
				worker: &WorkerNode{
					Status: WorkerNodeStatus{

						AllocatorSyncedStatus: AllocatorSyncedStatus{
							EntityName:    "pod1",
							EntityAlloced: true,
						},
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsWorkerEmpty(tt.args.worker); got != tt.want {
				t.Errorf("IsWorkerEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsWorkerVersionMisMatch(t *testing.T) {
	type args struct {
		worker *WorkerNode
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Changed",
			args: args{
				worker: &WorkerNode{
					Status: WorkerNodeStatus{

						AllocatorSyncedStatus: AllocatorSyncedStatus{
							Version: "1",
						},
					},
				},
			},
			want: true,
		},
		{
			name: "NotChanged",
			args: args{
				worker: &WorkerNode{
					Status: WorkerNodeStatus{},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsWorkerVersionMisMatch(tt.args.worker); got != tt.want {
				t.Errorf("IsWorkerVersionMisMatch() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsVersionMatch(t *testing.T) {
	type args struct {
		worker *WorkerNode
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Match",
			args: args{
				worker: &WorkerNode{
					Status: WorkerNodeStatus{},
				},
			},
			want: true,
		},
		{
			name: "NotMatch",
			args: args{
				worker: &WorkerNode{
					Status: WorkerNodeStatus{

						AllocatorSyncedStatus: AllocatorSyncedStatus{
							Version: "1",
						},
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsVersionMatch(tt.args.worker); got != tt.want {
				t.Errorf("IsVersionMatch() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsWorkerLatestVersionAvailable(t *testing.T) {
	type args struct {
		worker *WorkerNode
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "available",
			args: args{
				worker: &WorkerNode{
					Spec: WorkerNodeSpec{
						Version: "v2",
					},
					Status: WorkerNodeStatus{
						ServiceConditions: []ServiceCondition{
							{
								ServiceName: "test1",
								Status:      "True",
								Score:       1,
							},
							{
								ServiceName: "test1",
								Status:      "True",
								Score:       1,
								Version:     "v2",
							},
						},
						ServiceStatus: ServiceAvailable,
					},
				},
			},
			want: true,
		},
		{
			name: "unavailable",
			args: args{
				worker: &WorkerNode{
					Spec: WorkerNodeSpec{
						Version: "v2",
					},
					Status: WorkerNodeStatus{
						ServiceConditions: []ServiceCondition{
							{
								ServiceName: "test1",
								Status:      "True",
								Score:       1,
								Version:     "v1",
							},
							{
								ServiceName: "test1",
								Status:      "True",
								Score:       1,
								Version:     "v2",
							},
						},
						ServiceStatus: ServiceAvailable,
					},
				},
			},
			want: false,
		},
		{
			name: "unavailable",
			args: args{
				worker: &WorkerNode{
					Spec: WorkerNodeSpec{
						Version: "v2",
					},
					Status: WorkerNodeStatus{
						ServiceConditions: []ServiceCondition{
							{
								ServiceName: "test1",
								Status:      "True",
								Score:       1,
							},
							{
								ServiceName: "test1",
								Status:      "True",
								Score:       1,
								Version:     "v2",
							},
						},
						ServiceStatus: ServiceUnAvailable,
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsWorkerLatestVersionAvailable(tt.args.worker); got != tt.want {
				t.Errorf("IsWorkerLatestVersionAvailable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsWorkerServiceMatch(t *testing.T) {
	type args struct {
		worker *WorkerNode
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Match",
			args: args{
				worker: &WorkerNode{
					Spec: WorkerNodeSpec{
						VersionPlan: VersionPlan{
							BroadcastPlan: BroadcastPlan{
								Online: utils.BoolPtr(false),
							},
						},
					},
					Status: WorkerNodeStatus{
						ServiceOffline: true,
						ServiceStatus:  ServiceUnAvailable,
					},
				},
			},
			want: true,
		},
		{
			name: "Match",
			args: args{
				worker: &WorkerNode{
					Status: WorkerNodeStatus{
						ServiceOffline: false,
						ServiceStatus:  ServiceAvailable,
					},
				},
			},
			want: true,
		},
		{
			name: "NotMatch",
			args: args{
				worker: &WorkerNode{
					Spec: WorkerNodeSpec{
						VersionPlan: VersionPlan{
							BroadcastPlan: BroadcastPlan{
								Online: utils.BoolPtr(false),
							},
						},
					},
					Status: WorkerNodeStatus{
						ServiceOffline: true,
						ServiceStatus:  ServiceAvailable,
					},
				},
			},
			want: false,
		},
		{
			name: "NotMatch",
			args: args{
				worker: &WorkerNode{
					Status: WorkerNodeStatus{
						ServiceOffline: false,
						ServiceStatus:  ServiceUnAvailable,
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsWorkerServiceMatch(tt.args.worker); got != tt.want {
				t.Errorf("IsWorkerServiceMatch() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsPodPriorityOnline(t *testing.T) {
	type args struct {
		priority int32
	}
	tests := []struct {
		name string
		args args
		want bool
	}{

		{
			name: "Online",
			args: args{
				priority: 100,
			},
			want: true,
		},
		{
			name: "Online",
			args: args{
				priority: 199,
			},
			want: true,
		},
		{
			name: "NotOnline",
			args: args{
				priority: 200,
			},
			want: false,
		}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsPodPriorityOnline(tt.args.priority); got != tt.want {
				t.Errorf("IsPodPriorityOnline() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsPodPriorityOffline(t *testing.T) {
	type args struct {
		priority int32
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Offline",
			args: args{
				priority: 0,
			},
			want: true,
		},
		{
			name: "Offline",
			args: args{
				priority: 99,
			},
			want: true,
		},
		{
			name: "NotOffline",
			args: args{
				priority: 100,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsPodPriorityOffline(tt.args.priority); got != tt.want {
				t.Errorf("IsPodPriorityOffline() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsPodPrioritySys(t *testing.T) {
	type args struct {
		priority int32
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "sys",
			args: args{
				priority: 200,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsPodPrioritySys(tt.args.priority); got != tt.want {
				t.Errorf("IsPodPrioritySys() = %v, want %v", got, tt.want)
			}
		})
	}
}

func testNewVersionPlan(cpumode, containernode, cpus, image string, ip bool, podlabels map[string]string) VersionPlan {
	var hippoPodSpec = &HippoPodSpec{
		HippoPodSpecExtendFields: HippoPodSpecExtendFields{
			CpusetMode:     cpumode,
			ContainerModel: containernode,
		},
		Containers: []HippoContainer{
			{
				Container: corev1.Container{
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							"cpu": resource.MustParse(cpus),
						},
					},
					Image: image,
				},
			},
		},
	}
	if ip {
		hippoPodSpec.Containers[0].Container.Resources.Limits[ResourceIP] = resource.MustParse("1")
	}

	var hippoPodTemplate HippoPodTemplate
	hippoPodTemplate.Spec = *hippoPodSpec
	hippoPodTemplate.Labels = podlabels
	var versionPlan = VersionPlan{
		SignedVersionPlan: SignedVersionPlan{
			Template: &hippoPodTemplate,
		},
	}
	return versionPlan
}

func testNewVersionPlan2Continer(cpumode, containernode, cpus, cpus1, image string, ip bool, podlabels map[string]string) VersionPlan {
	var hippoPodSpec = &HippoPodSpec{
		HippoPodSpecExtendFields: HippoPodSpecExtendFields{
			CpusetMode:     cpumode,
			ContainerModel: containernode,
		},
		Containers: []HippoContainer{
			{
				Container: corev1.Container{
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							"cpu": resource.MustParse(cpus),
						},
					},
					Image: image,
				},
			},
			{
				Container: corev1.Container{
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							"cpu": resource.MustParse(cpus1),
						},
					},
					Image: image,
				},
			},
		},
	}
	if ip {
		hippoPodSpec.Containers[0].Container.Resources.Limits[ResourceIP] = resource.MustParse("1")
	}

	var hippoPodTemplate HippoPodTemplate
	hippoPodTemplate.Spec = *hippoPodSpec
	hippoPodTemplate.Labels = podlabels
	var versionPlan = VersionPlan{
		SignedVersionPlan: SignedVersionPlan{
			Template: &hippoPodTemplate,
		},
	}
	return versionPlan
}

func TestSignVersionPlan(t *testing.T) {
	type args struct {
		versionPlan VersionPlan
		labels      map[string]string
		name        string
	}
	verPlan := testNewVersionPlan("share", "docker", "64m", "regiest", false, nil)
	verPlan2Container := testNewVersionPlan2Continer("share", "docker", "32m", "32m", "regiest", false, nil)
	verPlansharemode := testNewVersionPlan("reserved", "docker", "64m", "regiest", false, nil)
	verPlancontainermode := testNewVersionPlan("share", "other", "64m", "regiest", false, nil)
	verPlancpu := testNewVersionPlan("share", "docker", "32m", "regiest", false, nil)
	verPlanimage := testNewVersionPlan("share", "docker", "64m", "regiest1", false, nil)
	verPlanIP := testNewVersionPlan("share", "docker", "64m", "regiest", true, nil)
	verPlanRestart := testNewVersionPlan("share", "docker", "64m", "regiest", false, nil)
	verPlanRestart.RestartAfterResourceChange = utils.BoolPtr(true)
	verPlanAPP := testNewVersionPlan("share", "docker", "64m", "regiest", true, map[string]string{LabelKeyAppName: "1"})
	verPlanROLE := testNewVersionPlan("share", "docker", "64m", "regiest", true, map[string]string{LabelKeyRoleName: "2"})

	tests := []struct {
		name    string
		args    args
		want    string
		want1   string
		wantErr bool
	}{
		{
			name: "original",
			args: args{
				versionPlan: verPlan,
				labels:      map[string]string{"foo": "bar"},
			},
			want:    "b9e4274dc95d1a452b1ceaf0f995e5ee",
			want1:   "25de86eefe092ac975193dedd0713c17",
			wantErr: false,
		},
		{
			name: "original",
			args: args{
				versionPlan: verPlan2Container,
				labels:      map[string]string{"foo": "bar"},
			},
			want:    "c231817adc7fbbd5f6f86a8954c203c2",
			want1:   "25de86eefe092ac975193dedd0713c17",
			wantErr: false,
		},
		{
			name: "sharemode",
			args: args{
				versionPlan: verPlansharemode,
				labels:      map[string]string{"foo": "bar"},
			},
			want:    "4690137c426d8ccc91af37edf35b2834",
			want1:   "cde9d65b30f46c510bbbeac3b3473698",
			wantErr: false,
		},
		{
			name: "containermode",
			args: args{
				versionPlan: verPlancontainermode,
				labels:      map[string]string{"foo": "bar"},
			},
			want:    "bb21163826ac4fcb1bf4be96d4945aab",
			want1:   "25de86eefe092ac975193dedd0713c17",
			wantErr: false,
		},
		{
			name: "cpu",
			args: args{
				versionPlan: verPlancpu,
				labels:      map[string]string{"foo": "bar"},
			},
			want:    "a7df53fb947b9b32a6540d163221e20c",
			want1:   "13d88b83024b3474870cd3ecd3fa003b",
			wantErr: false,
		},
		{
			name: "image",
			args: args{
				versionPlan: verPlanimage,
				labels:      map[string]string{"foo": "bar"},
			},
			want:    "fd7fa68c882180af37d69f23f0a4b4d7",
			want1:   "25de86eefe092ac975193dedd0713c17",
			wantErr: false,
		},
		{
			name: "labels",
			args: args{
				versionPlan: verPlan,
				labels:      map[string]string{"foo": "bar1"},
			},
			want:    "7745c9572d19d9731cd51a986d68c5e4",
			want1:   "dc6e70564ab9e412f842005d9fa18718",
			wantErr: false,
		},
		{
			name: "ip",
			args: args{
				versionPlan: verPlanIP,
				labels:      map[string]string{"foo": "bar"},
			},
			want:    "8c572550093804a413021f7f8c0eea35",
			want1:   "25de86eefe092ac975193dedd0713c17",
			wantErr: false,
		},
		{
			name: "app",
			args: args{
				versionPlan: verPlanAPP,
				labels:      map[string]string{"foo": "bar"},
			},
			want:    "568f93500151d0415ed9b0e91c93b68c",
			want1:   "25de86eefe092ac975193dedd0713c17",
			wantErr: false,
		},
		{
			name: "role",
			args: args{
				versionPlan: verPlanROLE,
				labels:      map[string]string{"foo": "bar"},
			},
			want:    "503b27a269975a36f1b6e6b79059cdaa",
			want1:   "25de86eefe092ac975193dedd0713c17",
			wantErr: false,
		},
		{
			name: "restart",
			args: args{
				versionPlan: verPlanRestart,
				labels:      map[string]string{"foo": "bar"},
			},
			want:    "5f23a276fa1482b357a62ca237db4970",
			want1:   "25de86eefe092ac975193dedd0713c17",
			wantErr: false,
		},
		{
			name: "filter labels",
			args: args{
				versionPlan: func() VersionPlan {
					plan := verPlan.DeepCopy()
					filterKvs := map[string]string{
						LabelServerlessAppName:       "app1",
						LabelServerlessAppStage:      "PUBLISH",
						LabelServerlessBizPlatform:   "AOP",
						LabelServerlessInstanceGroup: "group1",
						AnnoServerlessAppDetail:      "http://some-domain.com/detail?app=app1&group=group",
					}
					MergeMap(filterKvs, plan.Template.Labels)
					MergeMap(filterKvs, plan.Template.Annotations)
					return *plan
				}(),
				labels: map[string]string{
					"foo": "bar",
				},
			},
			want:    "b9e4274dc95d1a452b1ceaf0f995e5ee",
			want1:   "25de86eefe092ac975193dedd0713c17",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := SignVersionPlan(&tt.args.versionPlan, tt.args.labels, tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("SignVersionPlan() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("SignVersionPlan() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("SignVersionPlan() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_copyPodResource(t *testing.T) {
	type args struct {
		hippoPodSpec *HippoPodSpec
		target       *HippoPodSpec
		copyIP       bool
	}
	tests := []struct {
		name string
		args args
		want *HippoPodSpec
	}{
		{
			name: "copy",
			args: args{
				hippoPodSpec: &HippoPodSpec{
					HippoPodSpecExtendFields: HippoPodSpecExtendFields{
						CpusetMode: "RESERVED",
					},
					Containers: []HippoContainer{
						{
							Container: corev1.Container{
								Name: "a",
								Resources: corev1.ResourceRequirements{
									Limits: corev1.ResourceList{
										"cpu": resource.MustParse("3"),
									},
								},
							},
						},
					},
				},
				target: &HippoPodSpec{
					HippoPodSpecExtendFields: HippoPodSpecExtendFields{
						CpusetMode: "SHARE",
					},
					Containers: []HippoContainer{
						{
							Container: corev1.Container{
								Name: "a",
								Resources: corev1.ResourceRequirements{
									Limits: corev1.ResourceList{
										"cpu": resource.MustParse("2"),
									},
								},
							},
						},
					},
				},
			},
			want: &HippoPodSpec{
				HippoPodSpecExtendFields: HippoPodSpecExtendFields{
					CpusetMode: "SHARE",
				},
				Containers: []HippoContainer{
					{
						Container: corev1.Container{
							Name: "a",
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									"cpu": resource.MustParse("2"),
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			CopyPodResource(tt.args.hippoPodSpec, tt.args.target, tt.args.copyIP)
			if !reflect.DeepEqual(tt.args.hippoPodSpec, tt.want) {
				t.Errorf("CopyPodResource() = %s, want %s", utils.ObjJSON(tt.args.hippoPodSpec), utils.ObjJSON(tt.want))
			}
		})
	}
}

func newTestReplica(name string) *Replica {
	return &Replica{
		WorkerNode: WorkerNode{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		},
	}
}

func TestSubReplicas(t *testing.T) {
	type args struct {
		src []*Replica
		sub []*Replica
	}
	tests := []struct {
		name string
		args args
		want []*Replica
	}{
		{
			name: "sub",
			args: args{
				src: []*Replica{
					newTestReplica("1"),
					newTestReplica("2"),
					newTestReplica("3"),
				},
				sub: []*Replica{
					newTestReplica("1"),
					newTestReplica("4"),
				},
			},
			want: []*Replica{
				newTestReplica("2"),
				newTestReplica("3"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SubReplicas(tt.args.src, tt.args.sub); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SubReplicas() = %v, want %v", got, tt.want)
			}
		})
	}
}

func newTestWorker1(name string, resTimeout, lastNotMatch, readyTimeout, lastNotReady, minReady, readyTime, processNotMatchTimeout, processLastNotReady int64) *WorkerNode {
	var worker = WorkerNode{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   metav1.NamespaceDefault,
			Annotations: make(map[string]string),
		},
		Spec: WorkerNodeSpec{
			VersionPlan: VersionPlan{
				BroadcastPlan: BroadcastPlan{
					WorkerSchedulePlan: WorkerSchedulePlan{
						ResourceMatchTimeout: resTimeout,
						WorkerReadyTimeout:   readyTimeout,
						MinReadySeconds:      int32(minReady),
						ProcessMatchTimeout:  processNotMatchTimeout,
					},
				},
			},
		},
		Status: WorkerNodeStatus{
			WorkerReady:   false,
			AllocStatus:   WorkerAssigned,
			ServiceStatus: ServicePartAvailable,
			HealthStatus:  HealthAlive,

			LastWorkerNotReadytime: lastNotReady,
			HealthCondition: HealthCondition{
				LastTransitionTime: metav1.Time{
					Time: time.Unix(readyTime, 0),
				},
			},
			AllocatorSyncedStatus: AllocatorSyncedStatus{
				ResourceMatch:            false,
				Phase:                    Running,
				LastResourceNotMatchtime: lastNotMatch,
				LastProcessNotReadytime:  processLastNotReady,
			},
		},
	}
	return &worker
}

func newTestWorker(name string, resTimeout, lastNotMatch, readyTimeout, lastNotReady, minReady, readyTime int64) *WorkerNode {
	var worker = WorkerNode{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   metav1.NamespaceDefault,
			Annotations: make(map[string]string),
		},
		Spec: WorkerNodeSpec{
			VersionPlan: VersionPlan{
				BroadcastPlan: BroadcastPlan{
					WorkerSchedulePlan: WorkerSchedulePlan{
						ResourceMatchTimeout: resTimeout,
						WorkerReadyTimeout:   readyTimeout,
						MinReadySeconds:      int32(minReady),
					},
				},
			},
		},
		Status: WorkerNodeStatus{

			WorkerReady:   false,
			AllocStatus:   WorkerAssigned,
			ServiceStatus: ServicePartAvailable,
			HealthStatus:  HealthAlive,

			LastWorkerNotReadytime: lastNotReady,
			HealthCondition: HealthCondition{
				LastTransitionTime: metav1.Time{
					Time: time.Unix(readyTime, 0),
				},
			},
			AllocatorSyncedStatus: AllocatorSyncedStatus{
				ResourceMatch:            false,
				Phase:                    Running,
				LastResourceNotMatchtime: lastNotMatch,
			},
		},
	}
	return &worker
}

func newTestWorkerPrefer(name string, preferStr string) *WorkerNode {
	var worker = WorkerNode{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   metav1.NamespaceDefault,
			Annotations: make(map[string]string),
			Labels:      map[string]string{},
		},
	}
	if "" != preferStr {
		worker.Labels[LabelKeyHippoPreference] = preferStr
	}
	return &worker
}

func TestGetWorkerResyncTime(t *testing.T) {
	type args struct {
		worker *WorkerNode
	}
	tm := time.Now().Unix()
	tests := []struct {
		name string
		args args
		want int64
	}{
		{
			name: "notmatch",
			args: args{
				worker: newTestWorker("test", 60, tm, 70, tm, 80, tm),
			},
			want: 60,
		},
		{
			name: "notready",
			args: args{
				worker: newTestWorker("test", 90, tm, 70, tm, 80, tm),
			},
			want: 70,
		},
		{
			name: "notready",
			args: args{
				worker: newTestWorker("test", 0, tm, 70, tm, 0, tm),
			},
			want: 70,
		},
		{
			name: "notready",
			args: args{
				worker: newTestWorker("test", 90, tm, 90, tm, 80, tm),
			},
			want: 80,
		},
		{
			name: "notready",
			args: args{
				worker: newTestWorker("test", 0, tm, 0, tm, 80, tm),
			},
			want: 80,
		},
		{
			name: "processnotready",
			args: args{
				worker: newTestWorker1("test", 0, tm, 0, tm, 0, tm, 80, tm),
			},
			want: 80,
		},
		{
			name: "notschedule",
			args: args{
				worker: &WorkerNode{
					Status: WorkerNodeStatus{
						AllocatorSyncedStatus: AllocatorSyncedStatus{
							NotScheduleSeconds: 509,
						},
					},
				},
			},
			want: 91,
		},
	}
	for _, tt := range tests {
		NotScheduleTimeout = 600
		t.Run(tt.name, func(t *testing.T) {
			if got := GetWorkerResyncTime(tt.args.worker); got != tt.want {
				t.Errorf("GetWorkerResyncTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAddFinalizer(t *testing.T) {
	type args struct {
		rollingset *RollingSet
		finalizer  string
	}
	tests := []struct {
		name string
		args args
	}{

		{
			name: "test1",
			args: args{rollingset: newRollingset(), finalizer: "aaa"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			AddFinalizer(tt.args.rollingset, tt.args.finalizer)

			if len(tt.args.rollingset.GetFinalizers()) != 1 {
				t.Errorf("len:%v", len(tt.args.rollingset.GetFinalizers()))
			}

			if tt.args.rollingset.GetFinalizers()[0] != "aaa" {
				t.Errorf("finalizer:%v", tt.args.rollingset.GetFinalizers())
			}
		})
	}
}

func newRollingset() *RollingSet {
	rs := RollingSet{}
	return &rs
}

func TestSyncTimeout(t *testing.T) {
	type args struct {
		timeout    int64
		newtimeout int64
	}
	tests := []struct {
		name string
		args args
		want int64
	}{
		{
			name: "test1",
			args: args{timeout: 0, newtimeout: 1},
			want: 1,
		},
		{
			name: "test1",
			args: args{timeout: 0, newtimeout: -1},
			want: 0,
		},
		{
			name: "test1",
			args: args{timeout: 0, newtimeout: 2},
			want: 2,
		},
		{
			name: "test1",
			args: args{timeout: -1, newtimeout: 0},
			want: 0,
		},
		{
			name: "test1",
			args: args{timeout: 1, newtimeout: 2},
			want: 1,
		},
		{
			name: "test1",
			args: args{timeout: 2, newtimeout: 1},
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := syncTimeout(tt.args.timeout, tt.args.newtimeout)
			if got != tt.want {
				t.Errorf("got error:%d,%d", got, tt.want)
			}

		})
	}
}

func Test_SwapAndReleaseBackup(t *testing.T) {
	tm := time.Now().Unix()
	current := newTestWorker("current", 60, tm, 70, tm, 80, tm)
	backup := newTestWorker("backup", 60, tm, 70, tm, 80, tm)
	current.Labels = map[string]string{WorkerRoleKey: CurrentWorkerKey}
	backup.Labels = map[string]string{WorkerRoleKey: BackupWorkerKey}
	type args struct {
		current *WorkerNode
		backup  *WorkerNode
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "swap",
			args: args{
				current: current,
				backup:  backup,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := SwapAndReleaseBackup(tt.args.current, tt.args.backup); (err != nil) != tt.wantErr {
				t.Errorf("SwapAndReleaseBackuperror = %v, wantErr %v", err, tt.wantErr)
			}
			newworker, newbackup, _ := GetCurrentWorkerNodeV2([]*WorkerNode{current, backup}, false)
			if current != newbackup || backup != newworker {
				t.Errorf("SwapAndReleaseBackup swap failed %v,%v,%v",
					!reflect.DeepEqual(current, newbackup), !reflect.DeepEqual(backup, newworker), newbackup.Status.ToRelease == false)
			}
		})
	}
}

func Test_GetCurrentWorkerNodeV2(t *testing.T) {
	tm := time.Now().Unix()
	current := newTestWorker("current", 60, tm, 70, tm, 80, tm)
	backup := newTestWorker("backup", 60, tm, 70, tm, 80, tm)
	current.Labels = map[string]string{WorkerRoleKey: CurrentWorkerKey}
	backup.Labels = map[string]string{WorkerRoleKey: BackupWorkerKey}
	currentToRelease := newTestWorker("current", 60, tm, 70, tm, 80, tm)
	currentToRelease.Labels = map[string]string{WorkerRoleKey: CurrentWorkerKey}
	currentToRelease.Spec.ToDelete = true

	onecurrent := newTestWorker("current", 60, tm, 70, tm, 80, tm)
	onebackup := newTestWorker("backup", 60, tm, 70, tm, 80, tm)
	onecurrent.Labels = map[string]string{WorkerRoleKey: CurrentWorkerKey}
	onebackup.Labels = map[string]string{WorkerRoleKey: BackupWorkerKey}

	dbackup1 := newTestWorker("current", 60, tm, 70, tm, 80, tm)
	dbackup2 := newTestWorker("backup", 60, tm, 70, tm, 80, tm)
	dbackup1.Labels = map[string]string{WorkerRoleKey: BackupWorkerKey}
	dbackup2.Labels = map[string]string{WorkerRoleKey: BackupWorkerKey}

	dcurrent1 := newTestWorker("current", 60, tm, 70, tm, 80, tm)
	dcurrent2 := newTestWorker("backup", 60, tm, 70, tm, 80, tm)
	dcurrent1.Labels = map[string]string{WorkerRoleKey: CurrentWorkerKey}
	dcurrent1.Status.BecomeCurrentTime = metav1.Time{
		Time: time.Now(),
	}
	dcurrent2.Labels = map[string]string{WorkerRoleKey: CurrentWorkerKey}
	type args struct {
		current *WorkerNode
		backup  *WorkerNode
	}
	tests := []struct {
		name        string
		args        args
		wantErr     bool
		toRelease   bool
		wantCurrent *WorkerNode
		wantBackup  *WorkerNode
	}{
		{
			name: "normal",
			args: args{
				current: current,
				backup:  backup,
			},
			wantErr:     false,
			wantCurrent: current,
			wantBackup:  backup,
		},
		{
			name: "one current",
			args: args{
				current: onecurrent,
			},
			wantErr:     false,
			wantCurrent: onecurrent,
			wantBackup:  nil,
		},
		{
			name: "one backup",
			args: args{
				current: onebackup,
			},
			wantErr:     false,
			wantCurrent: onebackup,
			wantBackup:  nil,
		},
		{
			name: "torelease",
			args: args{
				current: currentToRelease,
				backup:  backup,
			},
			wantErr:     false,
			wantCurrent: currentToRelease,
			wantBackup:  backup,
			toRelease:   true,
		},
		{
			name: "two backup",
			args: args{
				current: dbackup1,
				backup:  dbackup2,
			},
			wantErr:     false,
			wantCurrent: dbackup2,
			wantBackup:  dbackup1,
		},
		{
			name: "two current",
			args: args{
				current: dcurrent1,
				backup:  dcurrent2,
			},
			wantErr:     false,
			wantCurrent: dcurrent1,
			wantBackup:  dcurrent2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			current, backup, toRelease := GetCurrentWorkerNodeV2([]*WorkerNode{tt.args.current, tt.args.backup}, false)
			if tt.wantCurrent != current || tt.wantBackup != backup {
				t.Errorf("GetCurrentWorkerNodeV2  failed ")
			}
			if tt.toRelease != toRelease {
				t.Errorf("GetCurrentWorkerNodeV2  failed %v,%v", tt.toRelease, toRelease)
			}
		})
	}
}

func generateRestartRecords(record ...int64) string {
	records := make([]string, 0, len(record))
	for i := range record {
		records = append(records, fmt.Sprintf("%d", record[i]))
	}
	return strings.Join(records, ",")
}

func TestAddRestartRecord(t *testing.T) {
	type args struct {
		pod    *corev1.Pod
		worker *WorkerNode
	}
	timeBase := time.Now()
	timeMillis := timeBase.Unix()
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		{
			name: "add new",
			args: args{
				pod: &corev1.Pod{
					Status: corev1.PodStatus{
						Phase: corev1.PodRunning,
						ContainerStatuses: []corev1.ContainerStatus{
							{
								Name:  "main",
								State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{ExitCode: 1, FinishedAt: metav1.Time{Time: timeBase}}},
							},
						},
					},
					Spec: corev1.PodSpec{
						RestartPolicy: corev1.RestartPolicyAlways,
					},
				},
				worker: &WorkerNode{
					Spec: WorkerNodeSpec{
						VersionPlan: VersionPlan{SignedVersionPlan: SignedVersionPlan{Template: &HippoPodTemplate{
							Spec: HippoPodSpec{Containers: []HippoContainer{{Container: corev1.Container{Name: "main"}}}},
						}}},
					},
				},
			},
			want: map[string]string{
				"main": fmt.Sprintf("%d", timeMillis),
			},
		},
		{
			name: "repeat",
			args: args{
				pod: &corev1.Pod{
					Status: corev1.PodStatus{
						Phase: corev1.PodRunning,
						ContainerStatuses: []corev1.ContainerStatus{
							{
								Name:  "main",
								State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{ExitCode: 1, FinishedAt: metav1.Time{Time: timeBase}}},
							},
						},
					},
					Spec: corev1.PodSpec{
						RestartPolicy: corev1.RestartPolicyAlways,
					},
				},
				worker: &WorkerNode{
					Spec: WorkerNodeSpec{
						VersionPlan: VersionPlan{SignedVersionPlan: SignedVersionPlan{Template: &HippoPodTemplate{
							Spec: HippoPodSpec{Containers: []HippoContainer{{Container: corev1.Container{Name: "main"}}}},
						}}},
					},
					Status: WorkerNodeStatus{
						RestartRecords: map[string]string{
							"main": generateRestartRecords(timeMillis-9, timeMillis-8, timeMillis-7, timeMillis-6, timeMillis-5, timeMillis-4, timeMillis-3, timeMillis-2, timeMillis),
						},
					},
				},
			},
			want: map[string]string{
				"main": generateRestartRecords(timeMillis-9, timeMillis-8, timeMillis-7, timeMillis-6, timeMillis-5, timeMillis-4, timeMillis-3, timeMillis-2, timeMillis),
			},
		},
		{
			name: "over limit",
			args: args{
				pod: &corev1.Pod{
					Status: corev1.PodStatus{
						Phase: corev1.PodRunning,
						ContainerStatuses: []corev1.ContainerStatus{
							{
								Name:  "main",
								State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{ExitCode: 1, FinishedAt: metav1.Time{Time: timeBase}}},
							},
						},
					},
					Spec: corev1.PodSpec{
						RestartPolicy: corev1.RestartPolicyAlways,
					},
				},
				worker: &WorkerNode{
					Spec: WorkerNodeSpec{
						VersionPlan: VersionPlan{SignedVersionPlan: SignedVersionPlan{Template: &HippoPodTemplate{
							Spec: HippoPodSpec{Containers: []HippoContainer{{
								Container:              corev1.Container{Name: "main"},
								ContainerHippoExterned: ContainerHippoExterned{Configs: &ContainerConfig{RestartCountLimit: 8}},
							}}},
						}}},
					},
					Status: WorkerNodeStatus{
						RestartRecords: map[string]string{
							"main": generateRestartRecords(timeMillis-9, timeMillis-8, timeMillis-7, timeMillis-6, timeMillis-5, timeMillis-4, timeMillis-3, timeMillis-2, timeMillis-1),
						},
					},
				},
			},
			want: map[string]string{
				"main": generateRestartRecords(timeMillis-7, timeMillis-6, timeMillis-5, timeMillis-4, timeMillis-3, timeMillis-2, timeMillis-1, timeMillis),
			},
		},
		{
			name: "multi container",
			args: args{
				pod: &corev1.Pod{
					Status: corev1.PodStatus{
						Phase: corev1.PodRunning,
						ContainerStatuses: []corev1.ContainerStatus{
							{
								Name:  "mainx",
								State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{ExitCode: 1, FinishedAt: metav1.Time{Time: timeBase}}},
							},
							{
								Name:  "main",
								State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{ExitCode: 1, FinishedAt: metav1.Time{Time: timeBase}}},
							},
						},
					},
					Spec: corev1.PodSpec{
						RestartPolicy: corev1.RestartPolicyAlways,
					},
				},
				worker: &WorkerNode{
					Spec: WorkerNodeSpec{
						VersionPlan: VersionPlan{SignedVersionPlan: SignedVersionPlan{Template: &HippoPodTemplate{
							Spec: HippoPodSpec{Containers: []HippoContainer{
								{
									Container:              corev1.Container{Name: "main"},
									ContainerHippoExterned: ContainerHippoExterned{Configs: &ContainerConfig{RestartCountLimit: 8}},
								},
								{
									Container: corev1.Container{Name: "mainx"},
								},
							}},
						}}},
					},
					Status: WorkerNodeStatus{
						RestartRecords: map[string]string{
							"main":  generateRestartRecords(timeMillis-9, timeMillis-8, timeMillis-7, timeMillis-6, timeMillis-5, timeMillis-4, timeMillis-3, timeMillis-2, timeMillis-1),
							"mainx": generateRestartRecords(timeMillis-9, timeMillis-8, timeMillis-7, timeMillis-6, timeMillis-5, timeMillis-4, timeMillis-3, timeMillis-2, timeMillis-1),
						},
					},
				},
			},
			want: map[string]string{
				"main":  generateRestartRecords(timeMillis-7, timeMillis-6, timeMillis-5, timeMillis-4, timeMillis-3, timeMillis-2, timeMillis-1, timeMillis),
				"mainx": generateRestartRecords(timeMillis-9, timeMillis-8, timeMillis-7, timeMillis-6, timeMillis-5, timeMillis-4, timeMillis-3, timeMillis-2, timeMillis-1, timeMillis),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			AddRestartRecord(tt.args.pod, tt.args.worker)
			got := tt.args.worker.Status.RestartRecords
			if len(got) != len(tt.want) {
				t.Errorf("AddRestartRecord() = %v, want %v", got, tt.want)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("AddRestartRecord() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestExceedRestartLimit(t *testing.T) {
	assert.False(t, ExceedRestartLimit("", 10))

	timeMillis := time.Now().Unix()
	records := generateRestartRecords(timeMillis-2, timeMillis-1, timeMillis)
	assert.False(t, ExceedRestartLimit(records, 10))

	records = generateRestartRecords(timeMillis-9, timeMillis-8, timeMillis-7, timeMillis-6, timeMillis-5, timeMillis-4, timeMillis-3, timeMillis-2, timeMillis-1, timeMillis)
	assert.True(t, ExceedRestartLimit(records, 10))

	records = generateRestartRecords(timeMillis-(1<<11)*ExponentialBackoff,
		timeMillis-(1<<9)*ExponentialBackoff,
		timeMillis-(1<<8)*ExponentialBackoff,
		timeMillis-(1<<7)*ExponentialBackoff,
		timeMillis-(1<<6)*ExponentialBackoff,
		timeMillis-(1<<5)*ExponentialBackoff,
		timeMillis-(1<<4)*ExponentialBackoff,
		timeMillis-(1<<3)*ExponentialBackoff,
		timeMillis-(1<<2)*ExponentialBackoff,
		timeMillis-(1<<1)*ExponentialBackoff)
	assert.False(t, ExceedRestartLimit(records, 10))

	records = generateRestartRecords(timeMillis-(1<<10)*ExponentialBackoff,
		timeMillis-(1<<9)*ExponentialBackoff,
		timeMillis-(1<<8)*ExponentialBackoff,
		timeMillis-(1<<7)*ExponentialBackoff,
		timeMillis-(1<<6)*ExponentialBackoff,
		timeMillis-(1<<5)*ExponentialBackoff,
		timeMillis-(1<<4)*ExponentialBackoff,
		timeMillis-(1<<3)*ExponentialBackoff,
		timeMillis-(1<<2)*ExponentialBackoff,
		timeMillis-(1<<1)*ExponentialBackoff)

	assert.True(t, ExceedRestartLimit(records, 10))

}

func TestTransPodPhase(t *testing.T) {
	type args struct {
		pod          *corev1.Pod
		processMatch bool
	}
	tests := []struct {
		name string
		args args
		want WorkerPhase
	}{
		{
			name: "pending",
			args: args{
				processMatch: true,
				pod: &corev1.Pod{
					Status: corev1.PodStatus{
						Phase: corev1.PodPending,
					},
				},
			},
			want: Pending,
		},
		{
			name: "PodSucceeded",
			args: args{
				processMatch: false,
				pod: &corev1.Pod{
					Status: corev1.PodStatus{
						Phase: corev1.PodSucceeded,
					},
				},
			},
			want: Pending,
		},
		{
			name: "PodSucceeded",
			args: args{
				processMatch: true,
				pod: &corev1.Pod{
					Status: corev1.PodStatus{
						Phase: corev1.PodSucceeded,
					},
				},
			},
			want: Terminated,
		},
		{
			name: "PodFailed",
			args: args{
				processMatch: true,
				pod: &corev1.Pod{
					Status: corev1.PodStatus{
						Phase: corev1.PodFailed,
					},
				},
			},
			want: Failed,
		},
		{
			name: "PodUnknown",
			args: args{
				processMatch: true,
				pod: &corev1.Pod{
					Status: corev1.PodStatus{
						Phase: corev1.PodUnknown,
					},
				},
			},
			want: Unknown,
		},
		{
			name: "PodRunningTerminate",
			args: args{
				processMatch: true,
				pod: &corev1.Pod{
					Status: corev1.PodStatus{
						Phase: corev1.PodRunning,
						ContainerStatuses: []corev1.ContainerStatus{
							{
								State: corev1.ContainerState{
									Terminated: &corev1.ContainerStateTerminated{},
								},
							},
						},
					},
					Spec: corev1.PodSpec{
						RestartPolicy: corev1.RestartPolicyAlways,
					},
				},
			},
			want: Terminated,
		},
		{
			name: "PodRunningTerminateFailed",
			args: args{
				processMatch: true,
				pod: &corev1.Pod{
					Status: corev1.PodStatus{
						Phase: corev1.PodRunning,
						ContainerStatuses: []corev1.ContainerStatus{
							{
								State: corev1.ContainerState{
									Terminated: &corev1.ContainerStateTerminated{
										ExitCode: 255,
									},
								},
							},
						},
					},
					Spec: corev1.PodSpec{
						RestartPolicy: corev1.RestartPolicyAlways,
					},
				},
			},
			want: Failed,
		},
		{
			name: "PodRunningTerminate",
			args: args{
				processMatch: true,
				pod: &corev1.Pod{
					Status: corev1.PodStatus{
						Phase: corev1.PodRunning,
						ContainerStatuses: []corev1.ContainerStatus{
							{
								State: corev1.ContainerState{
									Terminated: &corev1.ContainerStateTerminated{},
								},
							},
						},
					},
					Spec: corev1.PodSpec{
						RestartPolicy: corev1.RestartPolicyOnFailure,
					},
				},
			},
			want: Terminated,
		},
		{
			name: "PodRunning",
			args: args{
				processMatch: true,
				pod: &corev1.Pod{
					Status: corev1.PodStatus{
						Phase: corev1.PodRunning,
					},
					Spec: corev1.PodSpec{
						RestartPolicy: corev1.RestartPolicyAlways,
					},
				},
			},
			want: Running,
		},
		{
			name: "PodRunningNotMatch",
			args: args{
				processMatch: false,
				pod: &corev1.Pod{
					Status: corev1.PodStatus{
						Phase: corev1.PodRunning,
						ContainerStatuses: []corev1.ContainerStatus{
							{
								State: corev1.ContainerState{
									Terminated: &corev1.ContainerStateTerminated{},
								},
							},
						},
					},
					Spec: corev1.PodSpec{
						RestartPolicy: corev1.RestartPolicyAlways,
					},
				},
			},
			want: Pending,
		},
		{
			name: "PodRunningExitCode1",
			args: args{
				processMatch: true,
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							LabelKeyPodVersion: PodVersion3,
						},
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodRunning,
						ContainerStatuses: []corev1.ContainerStatus{
							{
								State: corev1.ContainerState{
									Terminated: &corev1.ContainerStateTerminated{
										ExitCode: 1,
									},
								},
							},
						},
					},
					Spec: corev1.PodSpec{
						RestartPolicy: corev1.RestartPolicyAlways,
					},
				},
			},
			want: Running,
		},
		{
			name: "BizPodRunningExitCode1",
			args: args{
				processMatch: true,
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							LabelKeyPodVersion: PodVersion3,
							LabelBizPodFlag:    "true",
						},
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodRunning,
						ContainerStatuses: []corev1.ContainerStatus{
							{
								State: corev1.ContainerState{
									Terminated: &corev1.ContainerStateTerminated{
										ExitCode: 1,
									},
								},
							},
						},
					},
					Spec: corev1.PodSpec{
						RestartPolicy: corev1.RestartPolicyAlways,
					},
				},
			},
			want: Failed,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TransPodPhase(tt.args.pod, tt.args.processMatch); got != tt.want {
				t.Errorf("TransPodPhase() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetHealthStatusByPhase(t *testing.T) {
	type args struct {
		worker *WorkerNode
	}
	var daemonPod HippoPodTemplate
	daemonPod.Spec.RestartPolicy = corev1.RestartPolicyAlways
	tests := []struct {
		name string
		args args
		want HealthStatus
	}{
		{
			name: "TestTerminateDaemon",
			args: args{worker: &WorkerNode{
				Spec: WorkerNodeSpec{VersionPlan: VersionPlan{
					SignedVersionPlan: SignedVersionPlan{
						Template: &daemonPod,
					},
				}},
				Status: WorkerNodeStatus{AllocatorSyncedStatus: AllocatorSyncedStatus{Phase: Terminated}}}},
			want: HealthDead,
		},
		{
			name: "TestTestTerminate",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{AllocatorSyncedStatus: AllocatorSyncedStatus{Phase: Terminated}}}},
			want: HealthAlive,
		},
		{
			name: "TestFailed",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{AllocatorSyncedStatus: AllocatorSyncedStatus{Phase: Failed}}}},
			want: HealthDead,
		},
		{
			name: "TestUnknown",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{AllocatorSyncedStatus: AllocatorSyncedStatus{Phase: Unknown}}}},
			want: HealthUnKnown,
		},
		{
			name: "TestPending",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{AllocatorSyncedStatus: AllocatorSyncedStatus{Phase: Pending}}}},
			want: HealthUnKnown,
		},
		{
			name: "TestNotDead",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{AllocatorSyncedStatus: AllocatorSyncedStatus{Phase: Running}}}},
			want: HealthAlive,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetHealthStatusByPhase(tt.args.worker); got != tt.want {
				t.Errorf("GetHealthStatusByPhase() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_IsDaemonPod(t *testing.T) {
	type args struct {
		podSpec corev1.PodSpec
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "PodDaemon",
			args: args{
				podSpec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyAlways,
				},
			},
			want: true,
		},
		{
			name: "NotPodDaemon",
			args: args{
				podSpec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyOnFailure,
				},
			},
			want: false,
		},
		{
			name: "NotPodDaemon",
			args: args{
				podSpec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
			want: false,
		},
		{
			name: "NotPodDaemon",
			args: args{
				podSpec: corev1.PodSpec{},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsDaemonPod(tt.args.podSpec); got != tt.want {
				t.Errorf("IsDaemonPod() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_isDaemonPlan(t *testing.T) {
	type args struct {
		template HippoPodTemplate
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "PodDaemon",
			args: args{
				template: func() HippoPodTemplate {
					var pod = HippoPodTemplate{}
					pod.Spec.RestartPolicy = corev1.RestartPolicyAlways
					return pod
				}(),
			},
			want: true,
		},
		{
			name: "PodDaemon",
			args: args{
				template: func() HippoPodTemplate {
					var pod = HippoPodTemplate{}
					pod.Spec.RestartPolicy = corev1.RestartPolicyOnFailure
					return pod
				}(),
			},
			want: false,
		},
		{
			name: "PodDaemon",
			args: args{
				template: func() HippoPodTemplate {
					var pod = HippoPodTemplate{}
					pod.Spec.RestartPolicy = corev1.RestartPolicyNever
					return pod
				}(),
			},
			want: false,
		},
		{
			name: "PodDaemon",
			args: args{
				template: func() HippoPodTemplate {
					var pod = HippoPodTemplate{}
					return pod
				}(),
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isDaemonPlan(&tt.args.template); got != tt.want {
				t.Errorf("isDaemonPlan() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReleaseWorker(t *testing.T) {
	type args struct {
		worker   *WorkerNode
		prohibit bool
	}
	tests := []struct {
		name string
		args args
		want *WorkerNode
	}{
		{
			name: "prohibit",
			args: args{
				worker:   newTestWorkerPrefer("prohibit", "APP-PROHIBIT-6000"),
				prohibit: true,
			},
			want: newTestWorkerPrefer("prefer", "APP-PROHIBIT-6000"),
		},
		{
			name: "prohibit",
			args: args{
				worker:   newTestWorkerPrefer("prohibit", ""),
				prohibit: true,
			},
			want: newTestWorkerPrefer("prefer", ""),
		},
		{
			name: "prefer",
			args: args{
				worker:   newTestWorkerPrefer("prefer", ""),
				prohibit: false,
			},
			want: newTestWorkerPrefer("prefer", ""),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ReleaseWorker(tt.args.worker, tt.args.prohibit)
			if !reflect.DeepEqual(tt.args.worker.Labels, tt.want.Labels) {
				t.Errorf("ReleaseWorker() = %v, want %v", tt.args.worker.Labels, tt.want.Labels)
			}
		})
	}
}

func TestInitWorkerScheduleTimeout(t *testing.T) {
	type args struct {
		worker *WorkerSchedulePlan
	}
	tests := []struct {
		name string
		args args
		want *WorkerSchedulePlan
	}{
		{
			name: "init",
			args: args{
				worker: &WorkerSchedulePlan{
					ResourceMatchTimeout: 0,
					ProcessMatchTimeout:  0,
					WorkerReadyTimeout:   0,
				},
			},
			want: &WorkerSchedulePlan{
				ResourceMatchTimeout: 60 * 5,
				ProcessMatchTimeout:  60 * 10,
				WorkerReadyTimeout:   60 * 60,
			},
		},

		{
			name: "inited",
			args: args{
				worker: &WorkerSchedulePlan{
					ResourceMatchTimeout: 3,
					ProcessMatchTimeout:  5,
					WorkerReadyTimeout:   -1,
				},
			},
			want: &WorkerSchedulePlan{
				ResourceMatchTimeout: 3,
				ProcessMatchTimeout:  5,
				WorkerReadyTimeout:   -1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			InitWorkerScheduleTimeout(tt.args.worker)
		})
	}
}

func TestIsWorkerTypeReady(t *testing.T) {
	type args struct {
		worker *WorkerNode
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "test1",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{HealthCondition: HealthCondition{WorkerStatus: WorkerTypeReady}}}},
			want: true,
		},
		{
			name: "test2",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{HealthCondition: HealthCondition{WorkerStatus: WorkerTypeUnknow}}}},
			want: false,
		},
		{
			name: "test1",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{HealthCondition: HealthCondition{WorkerStatus: WorkerTypeReady}}}},
			want: true,
		},
		{
			name: "test3",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{HealthCondition: HealthCondition{WorkerStatus: WorkerTypeNotReady}}}},
			want: false,
		},
		{
			name: "test4",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{HealthCondition: HealthCondition{WorkerStatus: ""}}}},
			want: true,
		},
		{
			name: "test4",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{HealthCondition: HealthCondition{Status: HealthAlive}}}},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsWorkerTypeReady(tt.args.worker); got != tt.want {
				t.Errorf("IsWorkerTypeReady() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetHealthMetas(t *testing.T) {
	type args struct {
		worker *WorkerNode
	}
	tests := []struct {
		name           string
		args           args
		want           map[string]string
		wantCustominfo string
	}{
		{
			name: "test1",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{HealthCondition: HealthCondition{
				Metas: map[string]string{"CustomInfo": "11111"},
			}}}},

			want:           map[string]string{"CustomInfo": "11111"},
			wantCustominfo: "11111",
		},
		{
			name: "test2",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{HealthCondition: HealthCondition{
				Metas:           map[string]string{"CustomInfo": "11111", "sign": "1234"},
				CompressedMetas: "invaild",
			}}}},

			want:           map[string]string{"CustomInfo": "11111", "sign": "1234"},
			wantCustominfo: "11111",
		},
		{
			name: "test3",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{HealthCondition: HealthCondition{
				CompressedMetas: "H4sIAAAAAAAA/6pWci4tLsnP9cxLy1eyUkoGAaVaAAAAAP//",
			}}}},

			want:           map[string]string{"CustomInfo": "ccccc"},
			wantCustominfo: "ccccc",
		},
		{
			name: "test4",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{HealthCondition: HealthCondition{
				Metas:           map[string]string{"CustomInfo": "11111"},
				CompressedMetas: "H4sIAAAAAAAA/6pWci4tLsnP9cxLy1eyUkoGAaVaAAAAAP//",
			}}}},

			want:           map[string]string{"CustomInfo": "11111"},
			wantCustominfo: "11111",
		},

		{
			name: "test5",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{HealthCondition: HealthCondition{
				CompressedMetas: "H4sIAAAAAAAA/6pWci4tLsnP9cxLy1eyUkoGAaVaAAAAAP//",
			}}}},

			want:           map[string]string{"CustomInfo": "ccccc"},
			wantCustominfo: "ccccc",
		},
		{
			name: "test6",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{HealthCondition: HealthCondition{
				Metas:           map[string]string{},
				CompressedMetas: "H4sIAAAAAAAA/6pWci4tLsnP9cxLy1eyUkoGAaVaAAAAAP//",
			}}}},

			want:           map[string]string{"CustomInfo": "ccccc"},
			wantCustominfo: "ccccc",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetHealthMetas(tt.args.worker)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetHealthMetas() = %v, want %v", got, tt.want)
			}
			if got["CustomInfo"] != tt.wantCustominfo {
				t.Errorf("TransHealthInfo() CustomInfo = %v, want %v", got, tt.wantCustominfo)
			}
		})

	}
}

func TestIsWorkerResVersionMisMatch(t *testing.T) {
	type args struct {
		worker *WorkerNode
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "changed",
			args: args{
				worker: &WorkerNode{
					Spec: WorkerNodeSpec{
						ResVersion: "a",
					},
					Status: WorkerNodeStatus{
						AllocatorSyncedStatus: AllocatorSyncedStatus{
							ResVersion: "b",
						},
					},
				},
			},
			want: true,
		},
		{
			name: "not changed",
			args: args{
				worker: &WorkerNode{
					Spec: WorkerNodeSpec{
						ResVersion: "a",
					},
					Status: WorkerNodeStatus{
						AllocatorSyncedStatus: AllocatorSyncedStatus{
							ResVersion: "a",
						},
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsWorkerResVersionMisMatch(tt.args.worker); got != tt.want {
				t.Errorf("IsWorkerResVersionMisMatch() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWorkerReplicaName(t *testing.T) {
	assert := assert.New(t)
	w := &WorkerNode{ObjectMeta: metav1.ObjectMeta{Name: "abc-a"}}
	assert.Equal("abc", GetWorkerReplicaName(w))
	w1 := &WorkerNode{ObjectMeta: metav1.ObjectMeta{Name: ""}}
	assert.Equal("", GetWorkerReplicaName(w1))
}

func Test_getMinReadySeconds(t *testing.T) {
	type args struct {
		worker *WorkerNode
	}
	tests := []struct {
		name string
		args args
		want int32
	}{
		{
			name: "got",
			args: args{
				worker: &WorkerNode{
					Spec: WorkerNodeSpec{
						VersionPlan: VersionPlan{
							BroadcastPlan: BroadcastPlan{
								WorkerSchedulePlan: WorkerSchedulePlan{
									MinReadySeconds: 10,
								},
							},
						},
					},
					Status: WorkerNodeStatus{
						AllocStatus:     WorkerAssigned,
						WorkerReady:     true,
						HealthStatus:    HealthAlive,
						ServiceStatus:   ServiceAvailable,
						HealthCondition: HealthCondition{WorkerStatus: WorkerTypeReady},
						AllocatorSyncedStatus: AllocatorSyncedStatus{
							Phase:        Running,
							ProcessMatch: true,
							ProcessReady: true,
						},
						WorkerStateChangeRecoder: WorkerStateChangeRecoder{
							LastServiceReadyTime: utils.Int64Ptr(time.Now().Unix() - 20),
						},
					},
				},
			},
			want: 10,
		},
		{
			name: "default",
			args: args{
				worker: &WorkerNode{
					Spec: WorkerNodeSpec{
						VersionPlan: VersionPlan{
							BroadcastPlan: BroadcastPlan{
								WorkerSchedulePlan: WorkerSchedulePlan{},
							},
						},
					},
					Status: WorkerNodeStatus{
						AllocStatus:     WorkerAssigned,
						WorkerReady:     true,
						HealthStatus:    HealthAlive,
						ServiceStatus:   ServiceAvailable,
						HealthCondition: HealthCondition{WorkerStatus: WorkerTypeReady},
						AllocatorSyncedStatus: AllocatorSyncedStatus{
							Phase:        Running,
							ProcessMatch: true,
							ProcessReady: true,
						},
						WorkerStateChangeRecoder: WorkerStateChangeRecoder{
							LastServiceReadyTime: utils.Int64Ptr(time.Now().Unix() - 20),
						},
					},
				},
			},
			want: 30,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getMinReadySeconds(tt.args.worker); got != tt.want {
				t.Errorf("getMinReadySeconds() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsServiceReadyForMinTime(t *testing.T) {
	type args struct {
		worker *WorkerNode
	}
	tests := []struct {
		name  string
		args  args
		want  int64
		want1 bool
	}{
		{
			name: "ready",
			args: args{
				worker: &WorkerNode{
					Spec: WorkerNodeSpec{
						VersionPlan: VersionPlan{
							BroadcastPlan: BroadcastPlan{
								WorkerSchedulePlan: WorkerSchedulePlan{
									MinReadySeconds: 10,
								},
							},
						},
					},
					Status: WorkerNodeStatus{
						AllocStatus:     WorkerAssigned,
						WorkerReady:     true,
						HealthStatus:    HealthAlive,
						ServiceStatus:   ServiceAvailable,
						HealthCondition: HealthCondition{WorkerStatus: WorkerTypeReady},
						AllocatorSyncedStatus: AllocatorSyncedStatus{
							Phase:        Running,
							ProcessMatch: true,
							ProcessReady: true,
						},
						ServiceReady: true,
						WorkerStateChangeRecoder: WorkerStateChangeRecoder{
							LastServiceReadyTime: utils.Int64Ptr(time.Now().Unix() - 20),
						},
					},
				},
			},
			want:  0,
			want1: true,
		},
		{
			name: "not ready",
			args: args{
				worker: &WorkerNode{
					Spec: WorkerNodeSpec{
						VersionPlan: VersionPlan{
							BroadcastPlan: BroadcastPlan{
								WorkerSchedulePlan: WorkerSchedulePlan{},
							},
						},
					},
					Status: WorkerNodeStatus{
						AllocStatus:     WorkerAssigned,
						WorkerReady:     true,
						HealthStatus:    HealthAlive,
						ServiceStatus:   ServiceAvailable,
						HealthCondition: HealthCondition{WorkerStatus: WorkerTypeReady},
						AllocatorSyncedStatus: AllocatorSyncedStatus{
							Phase:        Running,
							ProcessMatch: true,
							ProcessReady: true,
						},
						ServiceReady: true,
						WorkerStateChangeRecoder: WorkerStateChangeRecoder{
							LastServiceReadyTime: utils.Int64Ptr(time.Now().Unix() - 20),
						},
					},
				},
			},
			want:  20,
			want1: false,
		},
		{
			name: "not ready",
			args: args{
				worker: &WorkerNode{
					Spec: WorkerNodeSpec{
						VersionPlan: VersionPlan{
							BroadcastPlan: BroadcastPlan{
								WorkerSchedulePlan: WorkerSchedulePlan{
									MinReadySeconds: 10,
								},
							},
						},
					},
					Status: WorkerNodeStatus{
						AllocStatus:     WorkerAssigned,
						WorkerReady:     true,
						HealthStatus:    HealthAlive,
						ServiceStatus:   ServiceAvailable,
						HealthCondition: HealthCondition{WorkerStatus: WorkerTypeReady},
						AllocatorSyncedStatus: AllocatorSyncedStatus{
							Phase:        Running,
							ProcessMatch: true,
							ProcessReady: true,
						},
						ServiceReady: false,
						WorkerStateChangeRecoder: WorkerStateChangeRecoder{
							LastServiceReadyTime: utils.Int64Ptr(time.Now().Unix() - 20),
						},
					},
				},
			},
			want:  0,
			want1: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := IsServiceReadyForMinTime(tt.args.worker)
			if got != tt.want {
				t.Errorf("IsServiceReadyForMinTime() = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("IsServiceReadyForMinTime() = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestIsReplicaServiceReady(t *testing.T) {
	type args struct {
		replica *Replica
	}
	tests := []struct {
		name  string
		args  args
		want  int64
		want1 bool
	}{
		{
			name: "Ready",
			args: args{
				replica: &Replica{
					WorkerNode: WorkerNode{
						Status: WorkerNodeStatus{
							AllocStatus:     WorkerAssigned,
							WorkerReady:     true,
							HealthStatus:    HealthAlive,
							ServiceStatus:   ServiceAvailable,
							HealthCondition: HealthCondition{WorkerStatus: WorkerTypeReady},
							AllocatorSyncedStatus: AllocatorSyncedStatus{
								Phase:        Running,
								ProcessMatch: true,
								ProcessReady: true,
							},
							WorkerStateChangeRecoder: WorkerStateChangeRecoder{
								LastServiceReadyTime: utils.Int64Ptr(time.Now().Unix() - 40),
							},
						},
					},
				},
			},
			want:  0,
			want1: true,
		},
		{
			name: "Ready",
			args: args{
				replica: &Replica{
					WorkerNode: WorkerNode{
						Spec: WorkerNodeSpec{
							VersionPlan: VersionPlan{
								BroadcastPlan: BroadcastPlan{
									WorkerSchedulePlan: WorkerSchedulePlan{
										MinReadySeconds: 10,
									},
								},
							},
						},
						Status: WorkerNodeStatus{
							AllocStatus:     WorkerAssigned,
							WorkerReady:     true,
							HealthStatus:    HealthAlive,
							ServiceStatus:   ServiceAvailable,
							HealthCondition: HealthCondition{WorkerStatus: WorkerTypeReady},
							AllocatorSyncedStatus: AllocatorSyncedStatus{
								Phase:        Running,
								ProcessMatch: true,
								ProcessReady: true,
							},
							WorkerStateChangeRecoder: WorkerStateChangeRecoder{
								LastServiceReadyTime: utils.Int64Ptr(time.Now().Unix() - 20),
							},
						},
					},
				},
			},
			want:  0,
			want1: true,
		},
		{
			name: "Ready",
			args: args{
				replica: &Replica{
					WorkerNode: WorkerNode{
						Spec: WorkerNodeSpec{
							VersionPlan: VersionPlan{
								BroadcastPlan: BroadcastPlan{
									WorkerSchedulePlan: WorkerSchedulePlan{
										MinReadySeconds: 10,
									},
								},
							},
						},
						Status: WorkerNodeStatus{
							AllocStatus:     WorkerAssigned,
							WorkerReady:     true,
							HealthStatus:    HealthAlive,
							ServiceStatus:   ServiceAvailable,
							HealthCondition: HealthCondition{WorkerStatus: WorkerTypeReady},
							AllocatorSyncedStatus: AllocatorSyncedStatus{
								Phase:        Running,
								ProcessMatch: true,
								ProcessReady: true,
								Reclaim:      true,
							},
							WorkerStateChangeRecoder: WorkerStateChangeRecoder{
								LastServiceReadyTime: utils.Int64Ptr(time.Now().Unix() - 20),
							},
						},
					},
				},
			},
			want:  0,
			want1: true,
		},
		{
			name: "Not Ready for MinReadySeconds",
			args: args{
				replica: &Replica{
					WorkerNode: WorkerNode{
						Status: WorkerNodeStatus{
							AllocStatus:     WorkerAssigned,
							WorkerReady:     true,
							HealthStatus:    HealthAlive,
							ServiceStatus:   ServiceAvailable,
							HealthCondition: HealthCondition{WorkerStatus: WorkerTypeReady},
							AllocatorSyncedStatus: AllocatorSyncedStatus{
								Phase:        Running,
								ProcessMatch: true,
								ProcessReady: true,
							},
							WorkerStateChangeRecoder: WorkerStateChangeRecoder{
								LastServiceReadyTime: utils.Int64Ptr(time.Now().Unix() - 20),
							},
						},
					},
				},
			},
			want:  20,
			want1: false,
		},
		{
			name: "WorketTypeNotReady",
			args: args{
				replica: &Replica{
					WorkerNode: WorkerNode{
						Status: WorkerNodeStatus{
							AllocStatus:     WorkerAssigned,
							HealthStatus:    HealthAlive,
							HealthCondition: HealthCondition{WorkerStatus: WorkerTypeNotReady},
							AllocatorSyncedStatus: AllocatorSyncedStatus{
								Phase:        Running,
								ProcessReady: true,
								ProcessMatch: true,
							},
						},
					},
				},
			},
			want:  0,
			want1: false,
		},
		{
			name: "NotReady",
			args: args{
				replica: &Replica{
					WorkerNode: WorkerNode{
						Status: WorkerNodeStatus{
							AllocStatus: WorkerAssigned,

							HealthStatus: HealthAlive,

							HealthCondition: HealthCondition{
								Version: "1",
							},
							AllocatorSyncedStatus: AllocatorSyncedStatus{
								ResourceMatch: true,
								Phase:         Running,
							},
						},
					},
				},
			},
			want:  0,
			want1: false,
		},
		{
			name: "NotReady",
			args: args{
				replica: &Replica{
					WorkerNode: WorkerNode{
						Status: WorkerNodeStatus{

							HealthStatus: HealthAlive,

							AllocatorSyncedStatus: AllocatorSyncedStatus{
								ResourceMatch: true,
								Phase:         Running,
							},
						},
					},
				},
			},
			want:  0,
			want1: false,
		},
		{
			name: "NotReady",
			args: args{
				replica: &Replica{
					WorkerNode: WorkerNode{
						Status: WorkerNodeStatus{
							AllocStatus:  WorkerAssigned,
							HealthStatus: HealthAlive,

							AllocatorSyncedStatus: AllocatorSyncedStatus{
								Phase: Running,
							},
						},
					},
				},
			},
			want:  0,
			want1: false,
		},
		{
			name: "NotReady",
			args: args{
				replica: &Replica{
					WorkerNode: WorkerNode{
						Status: WorkerNodeStatus{
							AllocStatus: WorkerAssigned,
							AllocatorSyncedStatus: AllocatorSyncedStatus{
								ResourceMatch: true,
								Phase:         Running,
							},
						},
					},
				},
			},
			want:  0,
			want1: false,
		},
		{
			name: "NotReady",
			args: args{
				replica: &Replica{
					WorkerNode: WorkerNode{
						Status: WorkerNodeStatus{
							AllocStatus: WorkerAssigned,

							HealthStatus: HealthAlive,
							AllocatorSyncedStatus: AllocatorSyncedStatus{
								ResourceMatch: true,
							},
						},
					},
				},
			},
			want:  0,
			want1: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := IsReplicaServiceReady(tt.args.replica)
			if got != tt.want && got < tt.want-2 {
				t.Errorf("IsReplicaServiceReady() = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("IsReplicaServiceReady() = %v, want1 %v", got1, tt.want1)
			}
		})
	}
}

func Test_MegerContainersRess(t *testing.T) {
	type args struct {
		containers []ContainerInstanceField
	}

	res := resource.NewQuantity(int64(1), resource.DecimalExponent)
	res2 := resource.NewQuantity(int64(2), resource.DecimalExponent)
	tests := []struct {
		name string
		args args
		want corev1.ResourceRequirements
	}{
		{
			name: "merge",
			args: args{
				containers: []ContainerInstanceField{
					{
						Resources: corev1.ResourceRequirements{
							Limits: map[corev1.ResourceName]resource.Quantity{
								ResourceIP: *res,
								"cpu":      *res,
							},
						},
					},
					{
						Resources: corev1.ResourceRequirements{
							Limits: map[corev1.ResourceName]resource.Quantity{
								"cpu": *res,
								"mem": *res,
							},
						},
					},
				},
			},
			want: corev1.ResourceRequirements{
				Limits: map[corev1.ResourceName]resource.Quantity{
					ResourceIP: *res,
					"cpu":      *res2,
					"mem":      *res,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MegerContainersRess(tt.args.containers)
			if utils.ObjJSON(got) != utils.ObjJSON(tt.want) {
				t.Errorf("hippoPodAllocator.megerContainersRess() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_copyContainerResource(t *testing.T) {
	type args struct {
		src    corev1.ResourceRequirements
		dst    corev1.ResourceRequirements
		copyIP bool
	}

	res := resource.NewQuantity(int64(1), resource.DecimalExponent)
	res2 := resource.NewQuantity(int64(2), resource.DecimalExponent)
	tests := []struct {
		name string
		args args
		want corev1.ResourceRequirements
	}{
		{
			name: "copy",
			args: args{
				src: corev1.ResourceRequirements{
					Limits: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceName(ResourceIP):  *res,
						corev1.ResourceName(ResourceCPU): *res,
					},
				},
				dst: corev1.ResourceRequirements{
					Limits: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceName(ResourceCPU): *res2,
					},
				},
			},
			want: corev1.ResourceRequirements{
				Limits: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceName(ResourceIP):  *res,
					corev1.ResourceName(ResourceCPU): *res2,
				},
			},
		},
		{
			name: "copy",
			args: args{
				src: corev1.ResourceRequirements{
					Limits: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceName(ResourceCPU): *res,
					},
				},
				dst: corev1.ResourceRequirements{
					Limits: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceName(ResourceIP):  *res,
						corev1.ResourceName(ResourceCPU): *res2,
					},
				},
			},
			want: corev1.ResourceRequirements{
				Limits: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceName(ResourceCPU): *res2,
				},
			},
		},
		{
			name: "copy ip",
			args: args{
				src: corev1.ResourceRequirements{
					Limits: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceName(ResourceCPU): *res,
					},
				},
				dst: corev1.ResourceRequirements{
					Limits: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceName(ResourceIP):  *res,
						corev1.ResourceName(ResourceCPU): *res2,
					},
				},
				copyIP: true,
			},
			want: corev1.ResourceRequirements{
				Limits: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceName(ResourceCPU): *res2,
					corev1.ResourceName(ResourceIP):  *res,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := copyContainerResource(tt.args.src, tt.args.dst, tt.args.copyIP); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("copyContainerResource() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsAsiPod(t *testing.T) {
	type args struct {
		meta                 *metav1.ObjectMeta
		schedulerName        string
		defaultPodVersion    string
		defaultSchedulerName string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "isasi",
			args: args{
				meta: &metav1.ObjectMeta{
					Labels: map[string]string{
						"app.hippo.io/pod-version": "v3.0",
					},
				},
				schedulerName: "default-scheduler",
			},
			want: true,
		},
		{
			name: "isnotasi",
			args: args{},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defaultPodVersion = tt.args.defaultPodVersion
			defaultSchedulerName = tt.args.defaultSchedulerName
			if got := IsAsiPod(tt.args.meta); got != tt.want {
				t.Errorf("IsAsiPod() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSignWorkerRequirementID(t *testing.T) {
	type args struct {
		worker *WorkerNode
		suffix string
	}
	var daemonPod HippoPodTemplate
	daemonPod.Spec.RestartPolicy = corev1.RestartPolicyAlways

	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "test",
			args: args{
				worker: &WorkerNode{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{},
					},
					Spec: WorkerNodeSpec{
						VersionPlan: VersionPlan{
							SignedVersionPlan: SignedVersionPlan{Template: &daemonPod},
						},
					},
				},
			},
			want:    "f54d8ea38a53dc9212ab9fd45e658ca0",
			wantErr: false,
		},
		{
			name: "test",
			args: args{
				worker: &WorkerNode{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"app.hippo.io/pod-version": "v2.0"},
					},
					Spec: WorkerNodeSpec{
						VersionPlan: VersionPlan{
							SignedVersionPlan: SignedVersionPlan{Template: &daemonPod},
						},
					},
				},
			},
			want:    "f815d09554c549a56d7d30824e0eff5e",
			wantErr: false,
		},
		{
			name: "test",
			args: args{
				worker: &WorkerNode{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"app.hippo.io/pod-version": "v2.5"},
					},
					Spec: WorkerNodeSpec{
						VersionPlan: VersionPlan{
							SignedVersionPlan: SignedVersionPlan{Template: &daemonPod},
						},
					},
				},
			},
			want:    "78246c3fa90363a4aab4e4b0c448c229",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SignWorkerRequirementID(tt.args.worker, tt.args.suffix)
			if (err != nil) != tt.wantErr {
				t.Errorf("SignWorkerRequirementID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("SignWorkerRequirementID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsWorkerModeMisMatch(t *testing.T) {
	type args struct {
		worker *WorkerNode
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "normal",
			args: args{
				worker: &WorkerNode{
					Spec:   WorkerNodeSpec{},
					Status: WorkerNodeStatus{},
				},
			},
			want: false,
		},
		{
			name: "normal",
			args: args{
				worker: &WorkerNode{
					Spec: WorkerNodeSpec{
						WorkerMode: WorkerModeTypeActive,
					},
					Status: WorkerNodeStatus{
						AllocatorSyncedStatus: AllocatorSyncedStatus{
							WorkerMode: WorkerModeTypeActive,
						},
					},
				},
			},
			want: false,
		},
		{
			name: "normal",
			args: args{
				worker: &WorkerNode{
					Spec: WorkerNodeSpec{
						WorkerMode: WorkerModeTypeActive,
					},
					Status: WorkerNodeStatus{},
				},
			},
			want: false,
		},
		{
			name: "normal",
			args: args{
				worker: &WorkerNode{
					Spec: WorkerNodeSpec{},
					Status: WorkerNodeStatus{
						AllocatorSyncedStatus: AllocatorSyncedStatus{
							WorkerMode: WorkerModeTypeActive,
						},
					},
				},
			},
			want: false,
		},
		{
			name: "standby",
			args: args{
				worker: &WorkerNode{
					Spec: WorkerNodeSpec{
						WorkerMode: WorkerModeTypeHot,
					},
					Status: WorkerNodeStatus{
						AllocatorSyncedStatus: AllocatorSyncedStatus{
							WorkerMode: WorkerModeTypeActive,
						},
					},
				},
			},
			want: true,
		},
		{
			name: "standby",
			args: args{
				worker: &WorkerNode{
					Spec: WorkerNodeSpec{
						WorkerMode: WorkerModeTypeHot,
					},
					Status: WorkerNodeStatus{
						AllocatorSyncedStatus: AllocatorSyncedStatus{
							WorkerMode: WorkerModeTypeHot,
						},
					},
				},
			},
			want: false,
		},
		{
			name: "standby",
			args: args{
				worker: &WorkerNode{
					Spec: WorkerNodeSpec{
						WorkerMode: WorkerModeTypeWarm,
					},
					Status: WorkerNodeStatus{
						AllocatorSyncedStatus: AllocatorSyncedStatus{
							WorkerMode: WorkerModeTypeWarm,
						},
					},
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsWorkerModeMisMatch(tt.args.worker); got != tt.want {
				t.Errorf("IsWorkerModeMisMatch() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateShardName(t *testing.T) {
	type args struct {
		shardGroupName string
		shardKey       string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "test",
			args: args{
				shardGroupName: "group",
				shardKey:       "shard",
			},
			want: "group.shard",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GenerateShardName(tt.args.shardGroupName, tt.args.shardKey); got != tt.want {
				t.Errorf("GenerateShardName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsWorkerLongtimeNotSchedule(t *testing.T) {
	type args struct {
		worker  *WorkerNode
		timeout int
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "test",
			args: args{
				worker: &WorkerNode{
					Status: WorkerNodeStatus{
						AllocatorSyncedStatus: AllocatorSyncedStatus{
							NotScheduleSeconds: 601,
						},
					},
				},
				timeout: 600,
			},
			want: true,
		},
		{
			name: "test",
			args: args{
				worker: &WorkerNode{
					Status: WorkerNodeStatus{
						AllocatorSyncedStatus: AllocatorSyncedStatus{
							NotScheduleSeconds: 0,
						},
					},
				},
				timeout: 600,
			},
			want: false,
		},

		{
			name: "test",
			args: args{
				worker: &WorkerNode{
					Status: WorkerNodeStatus{
						AllocatorSyncedStatus: AllocatorSyncedStatus{
							NotScheduleSeconds: 509,
						},
					},
				},
				timeout: 600,
			},
			want: false,
		},
		{
			name: "test",
			args: args{
				worker: &WorkerNode{
					Status: WorkerNodeStatus{
						AllocatorSyncedStatus: AllocatorSyncedStatus{
							NotScheduleSeconds: 601,
						},
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		NotScheduleTimeout = tt.args.timeout
		t.Run(tt.name, func(t *testing.T) {
			if got := IsWorkerLongtimeNotSchedule(tt.args.worker); got != tt.want {
				t.Errorf("IsWorkerLongtimeNotSchedule() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetReplicaKey(t *testing.T) {
	type args struct {
		obj metav1.Object
	}
	tests := []struct {
		name  string
		args  args
		want  string
		want1 bool
	}{
		{
			name: "label",
			args: args{
				obj: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"replica-version-hash": "replica-key",
						},
					},
				},
			},
			want:  "replica-key",
			want1: true,
		},
		{
			name: "name",
			args: args{
				obj: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name: "replica-name-a",
					},
				},
			},
			want:  "replica-name",
			want1: true,
		},
		{
			name: "name",
			args: args{
				obj: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name: "replica-name-b",
					},
				},
			},
			want:  "replica-name",
			want1: true,
		},
		{
			name: "name",
			args: args{
				obj: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name: "replica-name-b-odef",
					},
				},
			},
			want:  "replica-name",
			want1: true,
		},
		{
			name: "name",
			args: args{
				obj: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name: "replica-name-a-odef",
					},
				},
			},
			want:  "replica-name",
			want1: true,
		},
		{
			name: "name",
			args: args{
				obj: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name: "replica-name-a-odefa",
					},
				},
			},
			want:  "",
			want1: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := GetReplicaKey(tt.args.obj)
			if got != tt.want {
				t.Errorf("getReplicaKeyFromName() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("getReplicaKeyFromName() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestSyncCrdTime(t *testing.T) {
	type args struct {
		old *metav1.ObjectMeta
		new *metav1.ObjectMeta
	}
	tests := []struct {
		name string
		args args
		want *metav1.ObjectMeta
	}{
		{
			name: "test1",
			args: args{
				new: &metav1.ObjectMeta{
					Annotations: map[string]string{
						"updateVersionTime-default": "2020-07-09T20:14:27.039079728+08:00",
					},
				},
				old: &metav1.ObjectMeta{
					Annotations: map[string]string{
						"updateVersionTime-default": "2020-07-09T20:14:27.197231244+08:00",
					},
				},
			},
			want: &metav1.ObjectMeta{
				Annotations: map[string]string{
					"updateVersionTime-default": "2020-07-09T20:14:27.197231244+08:00",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SyncCrdTime(tt.args.old, tt.args.new)
			if !reflect.DeepEqual(tt.args.new.Annotations, tt.want.Annotations) {
				t.Errorf("SyncCrdTime() = %v, want %v", tt.args.new.Annotations, tt.want.Annotations)
			}
		})
	}
}

func TestIsWorkerRowComplete(t *testing.T) {
	type args struct {
		worker *WorkerNode
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "true",
			args: args{
				worker: &WorkerNode{
					Spec: WorkerNodeSpec{
						Version: "aaa",
						VersionPlan: VersionPlan{
							BroadcastPlan: BroadcastPlan{
								RowComplete: nil,
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "true",
			args: args{
				worker: &WorkerNode{
					Spec: WorkerNodeSpec{
						Version: "aaa",
						VersionPlan: VersionPlan{
							BroadcastPlan: BroadcastPlan{
								RowComplete: utils.BoolPtr(true),
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "false",
			args: args{
				worker: &WorkerNode{
					Spec: WorkerNodeSpec{
						Version: "aaa",
						VersionPlan: VersionPlan{
							BroadcastPlan: BroadcastPlan{
								RowComplete: utils.BoolPtr(false),
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
			if got := IsWorkerRowComplete(tt.args.worker); got != tt.want {
				t.Errorf("IsWorkerRowComplete() = %v, want %v", got, tt.want)
			}
		})
	}
}

func readShardGroup(t *testing.T, file string) *ShardGroup {
	b, err := ioutil.ReadFile(file)
	if err != nil {
		t.Errorf("read file error %s , %v", file, err)
	}
	var group ShardGroup
	err = json.Unmarshal(b, &group)
	if err != nil {
		t.Errorf("unmarshal error %s , %v", string(b), err)
	}
	return &group
}

func TestIsGroupRollingSet(t *testing.T) {
	type args struct {
		rollingSet *RollingSet
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "is group",
			args: args{
				rollingSet: &RollingSet{
					ObjectMeta: metav1.ObjectMeta{
						Name: "group",
						Labels: map[string]string{
							"serverless.io/subrs-enable": "true",
						},
					},
					Spec: RollingSetSpec{
						Version:      "1",
						SchedulePlan: rollalgorithm.SchedulePlan{},
					},
				},
			},
			want: true,
		},
		{
			name: "not group",
			args: args{
				rollingSet: &RollingSet{
					ObjectMeta: metav1.ObjectMeta{
						Name: "group",
					},
					Spec: RollingSetSpec{
						Version: "1",
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsGroupRollingSet(tt.args.rollingSet); got != tt.want {
				t.Errorf("IsGroupRollingSet() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsSubRollingSet(t *testing.T) {
	type args struct {
		rollingSet *RollingSet
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "is sub",
			args: args{
				rollingSet: &RollingSet{
					ObjectMeta: metav1.ObjectMeta{
						Name: "sub2",
					},
					Spec: RollingSetSpec{
						GroupRS: utils.StringPtr("group"),
					},
				},
			},
			want: true,
		},
		{
			name: "not group",
			args: args{
				rollingSet: &RollingSet{
					ObjectMeta: metav1.ObjectMeta{
						Name: "sub",
					},
					Spec: RollingSetSpec{},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsSubRollingSet(tt.args.rollingSet); got != tt.want {
				t.Errorf("IsSubRollingSet() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetBufferName(t *testing.T) {
	type args struct {
		rollingSet *RollingSet
	}
	tests := []struct {
		name string
		args args
		want string
	}{

		{
			name: "group",
			args: args{
				rollingSet: &RollingSet{
					ObjectMeta: metav1.ObjectMeta{
						Name: "rs",
						Labels: map[string]string{
							"serverless.io/subrs-enable": "true",
						},
					},
					Spec: RollingSetSpec{
						Version:      "1",
						SchedulePlan: rollalgorithm.SchedulePlan{},
					},
				},
			},
			want: "rs.grs-buffer",
		},
		{
			name: "sub",
			args: args{
				rollingSet: &RollingSet{
					ObjectMeta: metav1.ObjectMeta{
						Name: "sub2",
					},
					Spec: RollingSetSpec{
						GroupRS: utils.StringPtr("rs"),
						SchedulePlan: rollalgorithm.SchedulePlan{
							Replicas: utils.Int32Ptr(10),
						},
					},
				},
			},
			want: "rs.grs-buffer",
		},
		{
			name: "normal",
			args: args{
				rollingSet: &RollingSet{
					ObjectMeta: metav1.ObjectMeta{
						Name: "normal",
					},
					Spec: RollingSetSpec{
						SchedulePlan: rollalgorithm.SchedulePlan{
							Replicas: utils.Int32Ptr(10),
						},
					},
				},
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetBufferName(tt.args.rollingSet); got != tt.want {
				t.Errorf("GetBufferName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsBufferRollingSet(t *testing.T) {
	type args struct {
		rollingSet *RollingSet
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "test",
			args: args{
				rollingSet: &RollingSet{
					ObjectMeta: metav1.ObjectMeta{
						Name: "rs.grs-buffer",
					},
				},
			},
			want: true,
		},
		{
			name: "test",
			args: args{
				rollingSet: &RollingSet{
					ObjectMeta: metav1.ObjectMeta{
						Name: "rs-sub2",
					},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsBufferRollingSet(tt.args.rollingSet); got != tt.want {
				t.Errorf("Controller.isBufferRollingSet() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSpreadAnno(t *testing.T) {
	type args struct {
		rollingSet *RollingSet
	}
	tests := []struct {
		name       string
		args       args
		wantSpread Spread
	}{
		{
			name: "test",
			args: args{
				rollingSet: &RollingSet{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							AnnotationC2Spread: "{\"active\":{\"replica\":2},\"inactive\":{\"replica\":1,\"labels\":{\"a\":\"a\"},\"annotations\":{\"b\":\"b\"}}}",
						},
					},
				},
			},
			wantSpread: Spread{
				Active: SpreadPatch{Replica: 2},
				Inactive: SpreadPatch{Replica: 1, Labels: map[string]string{
					"a": "a",
				}, Annotations: map[string]string{
					"b": "b",
				}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			active, inactive := GetInactiveSpreadReplicas(tt.args.rollingSet)
			assert.Equal(t, active, int32(tt.wantSpread.Active.Replica))
			assert.Equal(t, inactive, int32(tt.wantSpread.Inactive.Replica))
			labels, annotations := GetInactiveInterpose(tt.args.rollingSet)
			for key, val := range tt.wantSpread.Inactive.Annotations {
				assert.Equal(t, annotations[key], val)
			}
			for key, val := range tt.wantSpread.Inactive.Labels {
				assert.Equal(t, labels[key], val)
			}
		})
	}
}

func TestWorkerNode_IsSlienceNode(t *testing.T) {
	type fields struct {
		TypeMeta   metav1.TypeMeta
		ObjectMeta metav1.ObjectMeta
		Spec       WorkerNodeSpec
		Status     WorkerNodeStatus
	}
	type args struct {
		w *WorkerNode
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name:   "nil",
			fields: fields{},
			want:   false,
		},
		{
			name: "slience",
			args: args{
				w: &WorkerNode{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							SilentNodeAnnoKey: "true",
						},
					},
					Spec: WorkerNodeSpec{
						WorkerMode: WorkerModeTypeHot,
					},
				},
			},
			want: true,
		},
		{
			name: "not slience",
			args: args{
				w: &WorkerNode{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							SilentNodeAnnoKey: "",
						},
					},
					Spec: WorkerNodeSpec{
						WorkerMode: WorkerModeTypeHot,
					},
				},
			},
			want: false,
		},
		{
			name: "not slience",
			args: args{
				w: &WorkerNode{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{},
					},
					Spec: WorkerNodeSpec{
						WorkerMode: WorkerModeTypeHot,
					},
				},
			},
			want: false,
		},
		{
			name: "not slience",
			args: args{
				w: &WorkerNode{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							SilentNodeAnnoKey: "true",
						},
					},
					Spec: WorkerNodeSpec{},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsSilenceNode(tt.args.w); got != tt.want {
				t.Errorf("WorkerNode.IsSlienceNode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWorkerNode_SyncSlience(t *testing.T) {
	type fields struct {
		TypeMeta   metav1.TypeMeta
		ObjectMeta metav1.ObjectMeta
		Spec       WorkerNodeSpec
		Status     WorkerNodeStatus
	}
	type args struct {
		w                *WorkerNode
		targetWorkerMode WorkerModeType
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name:   "nil",
			fields: fields{},
			want:   false,
		},
		{
			name: "inactive",
			args: args{
				w: &WorkerNode{
					ObjectMeta: metav1.ObjectMeta{
						UID: "aaaa",
					},
					Spec: WorkerNodeSpec{
						WorkerMode: WorkerModeTypeActive,
					},
				},
				targetWorkerMode: WorkerModeTypeInactive,
			},
			want: false,
		},
		{
			name: "set slience",
			args: args{
				w: &WorkerNode{
					ObjectMeta: metav1.ObjectMeta{
						UID: "aaaa",
					},
					Spec: WorkerNodeSpec{
						WorkerMode: WorkerModeTypeActive,
					},
				},
				targetWorkerMode: WorkerModeTypeWarm,
			},
			want: true,
		},
		{
			name: "in slience",
			args: args{
				w: &WorkerNode{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							SilentNodeAnnoKey: "true",
							SilentTimeAnnoKey: strconv.FormatInt(time.Now().Unix(), 10),
						},
						UID: "aaaa",
					},
					Spec: WorkerNodeSpec{
						WorkerMode: WorkerModeTypeHot,
					},
					Status: WorkerNodeStatus{
						AllocatorSyncedStatus: AllocatorSyncedStatus{
							WorkerMode: WorkerModeTypeHot,
						},
					},
				},
				targetWorkerMode: WorkerModeTypeWarm,
			},
			want: true,
		},
		{
			name: "not in slience",
			args: args{
				w: &WorkerNode{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							SilentNodeAnnoKey: "true",
						},
						UID: "aaaa",
					},
					Spec: WorkerNodeSpec{
						WorkerMode: WorkerModeTypeHot,
					},
					Status: WorkerNodeStatus{
						AllocatorSyncedStatus: AllocatorSyncedStatus{
							WorkerMode: WorkerModeTypeHot,
						},
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SyncSilence(tt.args.w, tt.args.targetWorkerMode); got != tt.want {
				t.Errorf("WorkerNode.SyncSlience() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWorkerNode_InSilence(t *testing.T) {
	type fields struct {
		TypeMeta   metav1.TypeMeta
		ObjectMeta metav1.ObjectMeta
		Spec       WorkerNodeSpec
		Status     WorkerNodeStatus
	}
	type args struct {
		w *WorkerNode
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name:   "nil",
			fields: fields{},
			want:   false,
		},
		{
			name: "in slience",
			args: args{
				w: &WorkerNode{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							SilentNodeAnnoKey: "true",
							SilentTimeAnnoKey: strconv.FormatInt(time.Now().Unix(), 10),
						},
					},
					Spec: WorkerNodeSpec{
						WorkerMode: WorkerModeTypeHot,
					},
					Status: WorkerNodeStatus{
						AllocatorSyncedStatus: AllocatorSyncedStatus{
							WorkerMode: WorkerModeTypeHot,
						},
					},
				},
			},
			want: true,
		},
		{
			name: "not in slience",
			args: args{
				w: &WorkerNode{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							SilentNodeAnnoKey: "true",
						},
					},
					Spec: WorkerNodeSpec{
						WorkerMode: WorkerModeTypeHot,
					},
					Status: WorkerNodeStatus{
						AllocatorSyncedStatus: AllocatorSyncedStatus{
							WorkerMode: WorkerModeTypeHot,
						},
					},
				},
			},
			want: false,
		},
		{
			name: "not in slience",
			args: args{
				w: &WorkerNode{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							SilentTimeAnnoKey: strconv.FormatInt(time.Now().Unix(), 10),
						},
					},
					Spec: WorkerNodeSpec{
						WorkerMode: WorkerModeTypeHot,
					},
					Status: WorkerNodeStatus{
						AllocatorSyncedStatus: AllocatorSyncedStatus{
							WorkerMode: WorkerModeTypeHot,
						},
					},
				},
			},
			want: true,
		},
		{
			name: "not in slience",
			args: args{
				w: &WorkerNode{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							SilentNodeAnnoKey: "true",
							SilentTimeAnnoKey: strconv.FormatInt(time.Now().Unix(), 10),
						},
					},
					Spec: WorkerNodeSpec{
						WorkerMode: WorkerModeTypeHot,
					},
					Status: WorkerNodeStatus{
						AllocatorSyncedStatus: AllocatorSyncedStatus{
							WorkerMode: WorkerModeTypeWarm,
						},
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := InSilence(tt.args.w); got != tt.want {
				t.Errorf("WorkerNode.InSilence() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWorkerNode_SetBadReason(t *testing.T) {
	type args struct {
		w      *WorkerNode
		reason BadReasonType
	}

	tests := []struct {
		name string
		args args
	}{
		{
			name: "badreason",
			args: args{
				w:      &WorkerNode{},
				reason: BadReasonLost,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetBadReason(tt.args.w, tt.args.reason)
			if tt.args.w.Status.BadReasonMessage != "lost" {
				t.Errorf("SetBadReason = %v, want %v", tt.args.w.Status.BadReasonMessage, "lost")
			}
		})
	}
}

func TestWorkerNode_GetPodName(t *testing.T) {

	type args struct {
		w *WorkerNode
	}
	tests := []struct {
		name     string
		args     args
		wantName string
		wantErr  bool
	}{
		{
			name: "status",
			args: args{
				w: &WorkerNode{
					Status: WorkerNodeStatus{
						AllocatorSyncedStatus: AllocatorSyncedStatus{
							EntityName: "status-pod-name",
						},
					},
				},
			},
			wantName: "status-pod-name",
			wantErr:  false,
		},
		{
			name: "empty",
			args: args{
				w: &WorkerNode{
					ObjectMeta: metav1.ObjectMeta{
						Name: "worker-name",
						UID:  types.UID("odef-acbs"),
					},
					Status: WorkerNodeStatus{
						AllocatorSyncedStatus: AllocatorSyncedStatus{},
					},
				},
			},
			wantName: "worker-name-odef",
			wantErr:  false,
		},
		{
			name: "empty",
			args: args{
				w: &WorkerNode{
					ObjectMeta: metav1.ObjectMeta{
						Name: "worker-name",
					},
					Status: WorkerNodeStatus{
						AllocatorSyncedStatus: AllocatorSyncedStatus{},
					},
				},
			},
			wantName: "",
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, err := GetPodName(tt.args.w)
			if (err != nil) != tt.wantErr {
				t.Errorf("WorkerNode.GetPodName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotName != tt.wantName {
				t.Errorf("WorkerNode.GetPodName() = %v, want %v", gotName, tt.wantName)
			}
		})
	}
}

func TestGetUnassignedResson(t *testing.T) {
	type args struct {
		pod *corev1.Pod
	}
	tests := []struct {
		name string
		args args
		want UnassignedReasonType
	}{
		{
			name: "not schedule",
			args: args{
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						CreationTimestamp: metav1.Time{
							Time: time.Now(),
						},
					},
				},
			},
			want: 2,
		},
		{
			name: "not schedule-1",
			args: args{
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						CreationTimestamp: metav1.Time{
							Time: time.Now().Add(-time.Hour),
						},
					},
				},
			},
			want: 2,
		},
		{
			name: "quota not enough",
			args: args{
				pod: &corev1.Pod{
					Status: corev1.PodStatus{
						Conditions: []corev1.PodCondition{
							{
								Type:    corev1.PodScheduled,
								Message: "quota not enough",
							},
						},
					},
				},
			},
			want: 3,
		},
		{
			name: "quota not exist",
			args: args{
				pod: &corev1.Pod{
					Status: corev1.PodStatus{
						Conditions: []corev1.PodCondition{
							{
								Type:    corev1.PodScheduled,
								Message: "quota not exist",
							},
						},
					},
				},
			},
			want: 4,
		},
		{
			name: "resource not enough",
			args: args{
				pod: &corev1.Pod{
					Status: corev1.PodStatus{
						Conditions: []corev1.PodCondition{
							{
								Type:    corev1.PodScheduled,
								Message: "0/553 nodes are available: 1 had taints:node.kubernetes.io/not-ready, 1 had taints:virtual-kubelet.io/provider, 1 had taints:vk-deploy-node, 11 podSelector not match, 175 node(s) cpu not enough, 2 had taints:hippo_scheduler, 2 had taints:slave_dead, 307 node(s) memory not enough, 35 node(s) sigma/eni not enough, 9 had taints:sigma.ali/resource-pool, 9 had taints:slave_unhealthy.",
							},
						},
					},
				},
			},
			want: 5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetUnassignedResson(tt.args.pod); got != tt.want {
				t.Errorf("GetUnassignedResson() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetUnassignedReason(t *testing.T) {
	type args struct {
		worker *WorkerNode
		reason UnassignedReasonType
	}
	tests := []struct {
		name string
		args args
		want *WorkerNode
	}{
		{
			name: "none",
			args: args{
				worker: &WorkerNode{},
				reason: UnassignedReasonNone,
			},
			want: &WorkerNode{
				Status: WorkerNodeStatus{
					AllocatorSyncedStatus: AllocatorSyncedStatus{
						UnassignedReason:         0,
						UnassignedMessage:        "none",
						HistoryUnassignedMessage: "none",
					},
				},
			},
		},
		{
			name: "quota",
			args: args{
				worker: &WorkerNode{
					Status: WorkerNodeStatus{
						AllocatorSyncedStatus: AllocatorSyncedStatus{
							UnassignedReason:         0,
							UnassignedMessage:        "none",
							HistoryUnassignedMessage: "none",
						},
					},
				},
				reason: UnassignedReasonQuotaNotEnough,
			},
			want: &WorkerNode{
				Status: WorkerNodeStatus{
					AllocatorSyncedStatus: AllocatorSyncedStatus{
						UnassignedReason:         UnassignedReasonQuotaNotEnough,
						UnassignedMessage:        "quota_not_enough",
						HistoryUnassignedMessage: "quota_not_enough",
					},
				},
			},
		},
		{
			name: "none-2",
			args: args{
				worker: &WorkerNode{
					Status: WorkerNodeStatus{
						AllocatorSyncedStatus: AllocatorSyncedStatus{
							UnassignedReason:         UnassignedReasonQuotaNotEnough,
							UnassignedMessage:        "quota_not_enough",
							HistoryUnassignedMessage: "quota_not_enough",
						},
					},
				},
				reason: UnassignedReasonQuotaNotEnough,
			},
			want: &WorkerNode{
				Status: WorkerNodeStatus{
					AllocatorSyncedStatus: AllocatorSyncedStatus{
						UnassignedReason:         0,
						UnassignedMessage:        "none",
						HistoryUnassignedMessage: "quota_not_enough",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetUnassignedReason(tt.args.worker, tt.args.reason)
		})
	}
}

func TestCopySunfireLabels(t *testing.T) {
	src := map[string]string{
		LabelServerlessInstanceGroup: "group",
		LabelServerlessAppName:       "app",
	}
	dst := map[string]string{}
	CopySunfireLabels(src, dst)
	assert.True(t, dst[LabelKeyRoleShortName] == "group")
	assert.True(t, dst[LabelKeyAppShortName] == "app")

	src = map[string]string{
		LabelServerlessInstanceGroup: "group",
	}
	dst = map[string]string{}
	CopySunfireLabels(src, dst)
	assert.True(t, dst[LabelKeyRoleShortName] == "group")

	src = map[string]string{}
	dst = map[string]string{}
	CopySunfireLabels(src, dst)
	assert.True(t, len(dst) == 0)
}

func Test_isRejectByWebhook(t *testing.T) {
	type args struct {
		err error
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "true",
			args: args{
				err: fmt.Errorf(`update pod error:stl-alipay-venue-match.partition-0-4b3f7c0a-a,admission webhook "validating-pod-migrate-from-apiserver.alipay-extensions.k8s.alipay.com" denied the request: labels sigma.ali/app-name, sigma.ali/instance-group can not update`),
			},
			want: true,
		},
		{
			name: "false",
			args: args{
				err: fmt.Errorf(`others `),
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRejectByWebhook(tt.args.err); got != tt.want {
				t.Errorf("isRejectByWebhook() = %v, want %v", got, tt.want)
			}
		})
	}
}
