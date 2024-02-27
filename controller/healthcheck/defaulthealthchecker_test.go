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

package healthcheck

import (
	"testing"
	"time"

	mock "github.com/alibaba/kube-sharding/controller/healthcheck/mock"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	"github.com/golang/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewWorkernodeWithPhase(healthStatus carbonv1.HealthStatus, phase carbonv1.WorkerPhase) *carbonv1.WorkerNode {
	workerNode := carbonv1.WorkerNode{}
	workerNode.Status = carbonv1.WorkerNodeStatus{}
	workerNode.Status.IP = "10.10.2.122"
	workerNode.Status.AllocStatus = carbonv1.WorkerAssigned
	workerNode.Status.HealthStatus = healthStatus
	workerNode.Status.HealthCondition = carbonv1.HealthCondition{}
	workerNode.Status.HealthCondition.Status = healthStatus
	workerNode.Status.HealthCondition.Version = "testversion"
	workerNode.Status.Phase = phase
	workerNode.Status.Version = "testversion"
	workerNode.Status.ProcessReady = true
	second := time.Now().Unix()
	//	time.Now().Unix

	time := time.Unix(second, 0)
	//	workerNode.Status.HealthCondition.LastTransitionTime = v1.Time.Unix(v1.Time.Unix.Now())
	workerNode.Status.HealthCondition.LastTransitionTime = metav1.NewTime(time)
	return &workerNode
}

func Test_needHealthCheck(t *testing.T) {
	type args struct {
		workernode *carbonv1.WorkerNode
	}
	tests := []struct {
		name string
		args args
		want bool
		//	want1        *carbonv1.HealthCondition
		healthStatus carbonv1.HealthStatus
	}{
		{
			name:         "test1",
			args:         args{workernode: NewWorkernodeWithCheck(carbonv1.HealthAlive, carbonv1.WorkerAssigned, carbonv1.Running, 0)},
			want:         true,
			healthStatus: carbonv1.HealthAlive,
		},
		{
			name: "test2",
			args: args{
				workernode: func() *carbonv1.WorkerNode {
					workernode := NewWorkernodeWithCheck(carbonv1.HealthAlive, carbonv1.WorkerAssigned, carbonv1.Running, 0)
					workernode.Spec.WorkerMode = carbonv1.WorkerModeTypeWarm
					return workernode
				}(),
			},
			want:         false,
			healthStatus: carbonv1.HealthUnKnown,
		},
		{
			name: "test2",
			args: args{
				workernode: func() *carbonv1.WorkerNode {
					workernode := NewWorkernodeWithCheck(carbonv1.HealthAlive, carbonv1.WorkerAssigned, carbonv1.Running, 0)
					workernode.Spec.WorkerMode = carbonv1.WorkerModeTypeHot
					return workernode
				}(),
			},
			want:         true,
			healthStatus: carbonv1.HealthAlive,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := needHealthCheck(tt.args.workernode)
			if got != tt.want {
				t.Errorf("needLv7HealthCheck() got = %v, want %v", got, tt.want)
			}
			if got1.Status != tt.healthStatus {
				t.Errorf("needLv7HealthCheck() got = %v, want %v", got1.Status, tt.healthStatus)

			}
		})
	}
}

func TestDefaultHealthChecker_check(t *testing.T) {
	type fields struct {
		helper Helper
	}
	type args struct {
		workernode *carbonv1.WorkerNode
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		status carbonv1.HealthStatus
	}{
		{
			name:   "test1",
			args:   args{workernode: NewWorkernodeWithPhase(carbonv1.HealthAlive, carbonv1.Running)},
			status: carbonv1.HealthAlive,
		},
		{
			name:   "test2",
			args:   args{workernode: NewWorkernodeWithPhase(carbonv1.HealthDead, carbonv1.Running)},
			status: carbonv1.HealthAlive,
		},
		{
			name:   "test2",
			args:   args{workernode: NewWorkernodeWithPhase(carbonv1.HealthDead, carbonv1.Pending)},
			status: carbonv1.HealthUnKnown,
		},
		{
			name:   "test3",
			args:   args{workernode: NewWorkernodeWithPhase(carbonv1.HealthDead, carbonv1.Failed)},
			status: carbonv1.HealthDead,
		},
		{
			name:   "test4",
			args:   args{workernode: NewWorkernodeWithPhase(carbonv1.HealthAlive, carbonv1.Failed)},
			status: carbonv1.HealthDead,
		},
		{
			name:   "test5",
			args:   args{workernode: NewWorkernodeWithPhase(carbonv1.HealthAlive, carbonv1.Running)},
			status: carbonv1.HealthAlive,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &DefaultHealthChecker{
				helper: tt.fields.helper,
			}
			got := d.check(tt.args.workernode)
			if got.Status != tt.status {
				t.Errorf("DefaultHealthChecker.check() = %v, want %v", got.Status, tt.status)
			}

		})
	}
}

func TestDefaultHealthChecker_doCheck(t *testing.T) {
	type fields struct {
		helper Helper
	}
	type args struct {
		workernode *carbonv1.WorkerNode
		config     *carbonv1.HealthCheckerConfig
		rollingset *carbonv1.RollingSet
	}
	ctl := gomock.NewController(t)
	defer ctl.Finish()
	mockHelper := mock.NewMockHelper(ctl)

	mockHelper.EXPECT().Sync(gomock.Any(), gomock.Any()).Return(true).MaxTimes(10)
	mockHelper.EXPECT().GetWorkerNode(gomock.Any()).Return(nil, nil).MaxTimes(10)

	tests := []struct {
		name    string
		fields  fields
		args    args
		status  carbonv1.HealthStatus
		version string
	}{
		{
			name:    "test1",
			fields:  fields{helper: mockHelper},
			args:    args{workernode: NewWorkernodeWithPhase(carbonv1.HealthAlive, carbonv1.Running)},
			status:  carbonv1.HealthAlive,
			version: "testversion",
		},
		{
			name:    "test2",
			fields:  fields{helper: mockHelper},
			args:    args{workernode: NewWorkernodeWithPhase(carbonv1.HealthAlive, carbonv1.Pending)},
			status:  carbonv1.HealthUnKnown,
			version: "testversion",
		},
		{
			name:    "test3",
			fields:  fields{helper: mockHelper},
			args:    args{workernode: NewWorkernodeWithPhase(carbonv1.HealthDead, carbonv1.Running)},
			status:  carbonv1.HealthAlive,
			version: "testversion",
		},
		{
			name:    "test4",
			fields:  fields{helper: mockHelper},
			args:    args{workernode: NewWorkernodeWithPhase(carbonv1.HealthDead, carbonv1.Failed)},
			status:  carbonv1.HealthDead,
			version: "testversion",
		},
		{
			name:    "test5",
			fields:  fields{helper: mockHelper},
			args:    args{workernode: NewWorkernodeWithPhase(carbonv1.HealthLost, carbonv1.Failed)},
			status:  carbonv1.HealthDead,
			version: "testversion",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &DefaultHealthChecker{
				helper: tt.fields.helper,
			}
			got := d.doCheck(tt.args.workernode, tt.args.config, tt.args.rollingset)
			if got.Status != tt.status {
				t.Errorf("DefaultHealthChecker.check() = %v, want %v", got, tt.status)
			}
			if got.Version != tt.version {
				t.Errorf("DefaultHealthChecker.check() = %v, want %v", got.Version, tt.version)
			}

		})
	}
}
