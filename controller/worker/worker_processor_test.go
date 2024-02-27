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

package worker

import (
	"reflect"
	"testing"
	"time"

	"github.com/alibaba/kube-sharding/common/features"
	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	"github.com/alibaba/kube-sharding/pkg/client/clientset/versioned/fake"
	carboninformers "github.com/alibaba/kube-sharding/pkg/client/informers/externalversions"
	listers "github.com/alibaba/kube-sharding/pkg/client/listers/carbon/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/kubernetes/pkg/controller"
	labelsutil "k8s.io/kubernetes/pkg/util/labels"

	clientset "github.com/alibaba/kube-sharding/pkg/client/clientset/versioned"
	gomock "github.com/golang/mock/gomock"
	corev1 "k8s.io/api/core/v1"
)

func newTestWorkerWithStatus(name, ip string, status carbonv1.AllocStatus, entityName string) *carbonv1.WorkerNode {
	worker := newTestWorker(name, ip, nil)
	worker.Status.AllocStatus = status
	worker.Status.EntityName = entityName
	worker.Status.EntityUid = "mockUid-" + entityName
	worker.Status.EntityAlloced = true
	return worker
}

func newTestWorkerWithServiceStatus(name, ip string, status carbonv1.AllocStatus, serviceStatus carbonv1.ServiceStatus) *carbonv1.WorkerNode {
	worker := newTestWorker(name, ip, nil)
	worker.Status.AllocStatus = status
	worker.Status.HealthStatus = carbonv1.HealthAlive
	worker.Status.ServiceStatus = serviceStatus
	if serviceStatus == carbonv1.ServiceAvailable {
		worker.Status.ServiceConditions = []carbonv1.ServiceCondition{
			{
				Status: corev1.ConditionTrue,
			},
		}
	}
	return worker
}

func newTestWorkerWithToRelease(name, ip string, status carbonv1.AllocStatus, release bool) *carbonv1.WorkerNode {
	worker := newTestWorker(name, ip, nil)
	worker.Status.AllocStatus = status
	worker.Spec.Releasing = release
	worker.Status.ToRelease = release
	worker.Status.EntityName = "test"
	worker.Status.EntityAlloced = true
	return worker
}

func newTestWorkerWithVersion(name, ip string, status carbonv1.AllocStatus, entityName string, targetVersion, runtimeVersion string) *carbonv1.WorkerNode {
	worker := newTestWorkerWithStatus(name, ip, status, entityName)
	worker.Status.Version = runtimeVersion
	worker.Spec.Version = targetVersion

	return worker
}

func newTestWorkerHealth(name, ip string, status carbonv1.AllocStatus, entityName string, targetVersion, runtimeVersion string) *carbonv1.WorkerNode {
	worker := newTestWorkerWithStatus(name, ip, status, entityName)
	worker.Status.Version = runtimeVersion
	worker.Spec.Version = targetVersion
	worker.Status.AllocStatus = status
	worker.Status.HealthStatus = carbonv1.HealthAlive

	return worker
}

func Test_defaultProcessor_process(t *testing.T) {
	type fields struct {
		worker  *carbonv1.WorkerNode
		pod     *corev1.Pod
		machine workerStateMachine
	}

	ctl := gomock.NewController(t)
	defer ctl.Finish()
	unassignedMachine := NewMockworkerStateMachine(ctl)
	unassignedMachine.EXPECT().prepare().Return(nil).MinTimes(1)
	unassignedMachine.EXPECT().dealwithUnAssigned().Return(nil).MinTimes(1)
	unassignedMachine.EXPECT().postSync().Return(nil).MinTimes(1)

	assignedMachine := NewMockworkerStateMachine(ctl)
	assignedMachine.EXPECT().prepare().Return(nil).MinTimes(1)
	assignedMachine.EXPECT().dealwithAssigned().Return(nil).MinTimes(1)
	assignedMachine.EXPECT().postSync().Return(nil).MinTimes(1)

	lostMachine := NewMockworkerStateMachine(ctl)
	lostMachine.EXPECT().prepare().Return(nil).MinTimes(1)
	lostMachine.EXPECT().dealwithLost().Return(nil).MinTimes(1)
	lostMachine.EXPECT().postSync().Return(nil).MinTimes(1)

	offliningMachine := NewMockworkerStateMachine(ctl)
	offliningMachine.EXPECT().prepare().Return(nil).MinTimes(1)
	offliningMachine.EXPECT().dealwithOfflining().Return(nil).MinTimes(1)
	offliningMachine.EXPECT().postSync().Return(nil).MinTimes(1)

	releasingMachine := NewMockworkerStateMachine(ctl)
	releasingMachine.EXPECT().prepare().Return(nil).MinTimes(1)
	releasingMachine.EXPECT().dealwithReleasing().Return(nil).MinTimes(1)
	releasingMachine.EXPECT().postSync().Return(nil).MinTimes(1)

	releasedMachine := NewMockworkerStateMachine(ctl)
	releasedMachine.EXPECT().prepare().Return(nil).MinTimes(1)
	releasedMachine.EXPECT().dealwithReleased().Return(nil).MinTimes(1)
	releasedMachine.EXPECT().postSync().Return(nil).MinTimes(1)

	tests := []struct {
		name       string
		fields     fields
		wantWorker *carbonv1.WorkerNode
		wantPair   *carbonv1.WorkerNode
		wantPod    *corev1.Pod
		wantErr    bool
	}{
		{
			name: "unassigned",
			fields: fields{
				machine: unassignedMachine,
				worker:  newTestWorkerWithStatus("unassinged", "ip-1", carbonv1.WorkerUnAssigned, "pod1"),
			},
			wantErr: false,
		},
		{
			name: "assigned",
			fields: fields{
				machine: assignedMachine,
				worker:  newTestWorkerWithStatus("assinged", "ip-1", carbonv1.WorkerAssigned, "pod1"),
			},
			wantErr: false,
		},
		{
			name: "lost",
			fields: fields{
				machine: lostMachine,
				worker:  newTestWorkerWithStatus("lost", "ip-1", carbonv1.WorkerLost, "pod1"),
			},
			wantErr: false,
		},
		{
			name: "offlining",
			fields: fields{
				machine: offliningMachine,
				worker:  newTestWorkerWithStatus("offlining", "ip-1", carbonv1.WorkerOfflining, "pod1"),
			},
			wantErr: false,
		},
		{
			name: "releasing",
			fields: fields{
				machine: releasingMachine,
				worker:  newTestWorkerWithStatus("releasing", "ip-1", carbonv1.WorkerReleasing, "pod1"),
			},
			wantErr: false,
		},
		{
			name: "released",
			fields: fields{
				machine: releasedMachine,
				worker:  newTestWorkerWithStatus("released", "ip-1", carbonv1.WorkerReleased, "pod1"),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &defaultProcessor{
				worker:  tt.fields.worker,
				pod:     tt.fields.pod,
				machine: tt.fields.machine,
			}
			_, _, err := p.process()
			if (err != nil) != tt.wantErr {
				t.Errorf("defaultProcessor.process() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func Test_defaultStateMachine_prepare(t *testing.T) {
	type fields struct {
		worker    *carbonv1.WorkerNode
		pod       *corev1.Pod
		allocator workerAllocator
		u         workerUpdater
		c         *Controller
	}

	ctl := gomock.NewController(t)
	defer ctl.Finish()
	allocator := NewMockworkerAllocator(ctl)
	allocator.EXPECT().syncWorkerAllocStatus().Return(nil).MinTimes(1)
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "sync",
			fields: fields{
				allocator: allocator,
				worker:    newTestWorkerWithStatus("assinged", "ip-1", carbonv1.WorkerAssigned, "pod1"),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &defaultStateMachine{
				worker:    tt.fields.worker,
				pod:       tt.fields.pod,
				allocator: tt.fields.allocator,
				u:         tt.fields.u,
				c:         tt.fields.c,
			}
			if err := m.prepare(); (err != nil) != tt.wantErr {
				t.Errorf("defaultStateMachine.prepare() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_defaultStateMachine_dealwithReleased(t *testing.T) {
	type fields struct {
		worker    *carbonv1.WorkerNode
		pod       *corev1.Pod
		allocator workerAllocator
		u         workerUpdater
		c         *Controller
	}
	directReleaseWorker := newTestWorkerWithStatus("released", "ip-1", carbonv1.WorkerReleased, "pod1")
	directReleaseWorker.Spec.RecoverStrategy = carbonv1.DirectReleasedRecoverStrategy
	directReleaseWorker.Spec.Reclaim = true
	directReleaseWorker.Spec.Releasing = true
	tests := []struct {
		name            string
		fields          fields
		wantErr         bool
		wantAllocStatus carbonv1.AllocStatus
	}{
		{
			name: "release",
			fields: fields{
				worker: newTestWorkerWithStatus("released", "ip-1", carbonv1.WorkerReleased, "pod1"),
			},
			wantErr:         false,
			wantAllocStatus: carbonv1.WorkerReleased,
		},
		{
			name: "direct recover",
			fields: fields{
				worker: directReleaseWorker,
			},
			wantErr:         false,
			wantAllocStatus: carbonv1.WorkerReleased,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &defaultStateMachine{
				worker:    tt.fields.worker,
				pod:       tt.fields.pod,
				allocator: tt.fields.allocator,
				u:         tt.fields.u,
				c:         tt.fields.c,
			}
			if err := m.dealwithReleased(); (err != nil) != tt.wantErr {
				t.Errorf("defaultStateMachine.dealwithReleased() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.fields.worker.Status.AllocStatus != tt.wantAllocStatus {
				t.Errorf("simpleWorkerScheduler.dealwithReleasing() Status = %v, wantStatus %v", tt.fields.worker.Status.AllocStatus, tt.wantAllocStatus)
			}
		})
	}
}

func Test_defaultStateMachine_dealwithReleasing(t *testing.T) {
	type fields struct {
		worker    *carbonv1.WorkerNode
		pod       *corev1.Pod
		allocator workerAllocator
		u         workerUpdater
		c         *Controller
	}

	ctl := gomock.NewController(t)
	defer ctl.Finish()
	allocator := NewMockworkerAllocator(ctl)
	allocator.EXPECT().delete().Return(nil).MinTimes(1)

	tests := []struct {
		name       string
		fields     fields
		wantErr    bool
		wantStatus carbonv1.AllocStatus
	}{
		{
			name: "empty",
			fields: fields{
				allocator: nil,
				worker:    newTestWorkerWithStatus("releasing", "", carbonv1.WorkerReleasing, ""),
			},
			wantErr:    false,
			wantStatus: carbonv1.WorkerReleased,
		},
		{
			name: "notempty",
			fields: fields{
				allocator: allocator,
				worker:    newTestWorkerWithStatus("releasing", "ip-1", carbonv1.WorkerReleasing, "pod1"),
				pod:       newPod("test", "1", corev1.PodPending, ""),
			},
			wantErr:    false,
			wantStatus: carbonv1.WorkerReleased,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &defaultStateMachine{
				worker:    tt.fields.worker,
				pod:       tt.fields.pod,
				allocator: tt.fields.allocator,
				u:         tt.fields.u,
				c:         tt.fields.c,
			}
			err := m.dealwithReleasing()
			if (err != nil) != tt.wantErr {
				t.Errorf("defaultStateMachine.dealwithReleasing() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.fields.worker.Status.AllocStatus != tt.wantStatus {
				t.Errorf("simpleWorkerScheduler.dealwithReleasing() Status = %v, wantStatus %v", tt.fields.worker.Status.AllocStatus, tt.wantStatus)
			}
		})
	}
}

func Test_defaultStateMachine_dealwithOfflining(t *testing.T) {
	type fields struct {
		worker    *carbonv1.WorkerNode
		pod       *corev1.Pod
		allocator workerAllocator
		u         workerUpdater
		c         *Controller
	}
	tests := []struct {
		name       string
		fields     fields
		wantErr    bool
		wantStatus carbonv1.AllocStatus
	}{
		{
			name: "servicematch",
			fields: fields{
				worker: newTestWorkerWithServiceStatus("releasing", "", carbonv1.WorkerOfflining, carbonv1.ServiceUnAvailable),
			},
			wantErr:    false,
			wantStatus: carbonv1.WorkerReleasing,
		},
		{
			name: "servicenotmatch",
			fields: fields{
				worker: func() *carbonv1.WorkerNode {
					worker := newTestWorker("name", "ip", nil)
					worker.Status.AllocStatus = carbonv1.WorkerOfflining
					worker.Status.ServiceStatus = carbonv1.ServiceUnAvailable
					worker.Status.ServiceConditions = []carbonv1.ServiceCondition{
						{
							Status: corev1.ConditionFalse,
						},
					}
					return worker
				}(),
			},
			wantErr:    false,
			wantStatus: carbonv1.WorkerOfflining,
		},
		{
			name: "serviceNotmatch",
			fields: fields{
				worker: newTestWorkerWithServiceStatus("releasing", "ip-1", carbonv1.WorkerOfflining, carbonv1.ServiceAvailable),
			},
			wantErr:    false,
			wantStatus: carbonv1.WorkerOfflining,
		},
		{
			name: "failed",
			fields: fields{
				worker: func() *carbonv1.WorkerNode {
					worker := newTestWorkerWithServiceStatus("releasing", "ip-1", carbonv1.WorkerOfflining, carbonv1.ServiceAvailable)
					worker.Status.Phase = carbonv1.Failed
					return worker
				}(),
			},
			wantErr:    false,
			wantStatus: carbonv1.WorkerReleasing,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &defaultStateMachine{
				worker:    tt.fields.worker,
				pod:       tt.fields.pod,
				allocator: tt.fields.allocator,
				u:         tt.fields.u,
				c:         tt.fields.c,
			}
			if err := m.dealwithOfflining(); (err != nil) != tt.wantErr {
				t.Errorf("defaultStateMachine.dealwithOfflining() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.fields.worker.Status.AllocStatus != tt.wantStatus {
				t.Errorf("simpleWorkerScheduler.dealwithOfflining() Status = %v, wantStatus %v", tt.fields.worker.Status.AllocStatus, tt.wantStatus)
			}
		})
	}
}

func Test_defaultStateMachine_dealwithLost(t *testing.T) {
	type fields struct {
		worker    *carbonv1.WorkerNode
		pod       *corev1.Pod
		allocator workerAllocator
		u         workerUpdater
		c         *Controller
	}
	tests := []struct {
		name       string
		fields     fields
		wantErr    bool
		wantStatus carbonv1.AllocStatus
	}{
		{
			name: "torelease",
			fields: fields{
				worker: newTestWorkerWithToRelease("lost", "", carbonv1.WorkerLost, true),
			},
			wantErr:    false,
			wantStatus: carbonv1.WorkerOfflining,
		},
		{
			name: "nottorelease",
			fields: fields{
				worker: newTestWorkerWithToRelease("lost", "ip-1", carbonv1.WorkerLost, false),
			},
			wantErr:    false,
			wantStatus: carbonv1.WorkerLost,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &defaultStateMachine{
				worker:    tt.fields.worker,
				pod:       tt.fields.pod,
				allocator: tt.fields.allocator,
				u:         tt.fields.u,
				c:         tt.fields.c,
			}
			if err := m.dealwithLost(); (err != nil) != tt.wantErr {
				t.Errorf("defaultStateMachine.dealwithLost() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.fields.worker.Status.AllocStatus != tt.wantStatus {
				t.Errorf("simpleWorkerScheduler.dealwithLost() Status = %v, wantStatus %v", tt.fields.worker.Status.AllocStatus, tt.wantStatus)
			}
		})
	}
}

func Test_defaultStateMachine_dealwithUnAssigned(t *testing.T) {
	type fields struct {
		worker    *carbonv1.WorkerNode
		pod       *corev1.Pod
		allocator workerAllocator
		u         workerUpdater
		c         *Controller
	}

	ctl := gomock.NewController(t)
	defer ctl.Finish()
	allocator := NewMockworkerAllocator(ctl)
	allocator.EXPECT().allocate().Return(nil).MinTimes(1)
	torelease := newTestWorkerWithToRelease("lost", "ip-1", carbonv1.WorkerUnAssigned, true)
	torelease.Status.EntityAlloced = false
	torelease.Status.EntityName = ""

	emptyWorker := newTestWorkerWithStatus("notempty", "ip1", carbonv1.WorkerUnAssigned, "pod1")
	emptyWorker.Status.EntityAlloced = false
	emptyWorker.Status.EntityName = ""

	notAllocedWorker := newTestWorkerWithStatus("notalloced", "ip1", carbonv1.WorkerUnAssigned, "pod1")
	notAllocedWorker.Status.EntityAlloced = false
	notAllocedWorker.Status.EntityName = "notalloced"

	allocatorNotAlloced := NewMockworkerAllocator(ctl)
	allocatorNotAlloced.EXPECT().syncPodSpec().Return(nil).MinTimes(1)
	podNotAlloced := newPod("test", "1", corev1.PodPending, "")
	tests := []struct {
		name       string
		fields     fields
		wantErr    bool
		wantStatus carbonv1.AllocStatus
	}{
		{
			name: "empty",
			fields: fields{
				worker:    emptyWorker,
				allocator: allocator,
			},
			wantErr:    false,
			wantStatus: carbonv1.WorkerUnAssigned,
		},
		{
			name: "notempty",
			fields: fields{
				worker: newTestWorkerWithStatus("notempty", "ip1", carbonv1.WorkerUnAssigned, "pod1"),
			},
			wantErr:    false,
			wantStatus: carbonv1.WorkerAssigned,
		},
		{
			name: "nottorelease",
			fields: fields{
				worker:    torelease,
				allocator: allocator,
			},
			wantErr:    false,
			wantStatus: carbonv1.WorkerReleased,
		},
		{
			name: "not alloced",
			fields: fields{
				worker:    notAllocedWorker,
				allocator: allocatorNotAlloced,
				pod:       podNotAlloced,
			},
			wantErr:    false,
			wantStatus: carbonv1.WorkerUnAssigned,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &defaultStateMachine{
				worker:    tt.fields.worker,
				pod:       tt.fields.pod,
				allocator: tt.fields.allocator,
				u:         tt.fields.u,
				c:         tt.fields.c,
			}
			if err := m.dealwithUnAssigned(); (err != nil) != tt.wantErr {
				t.Errorf("defaultStateMachine.dealwithUnAssigned() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.fields.worker.Status.AllocStatus != tt.wantStatus {
				t.Errorf("simpleWorkerScheduler.dealwithLost() Status = %v, wantStatus %v", tt.fields.worker.Status.AllocStatus, tt.wantStatus)
			}
		})
	}
}

func Test_defaultStateMachine_dealwithAssigned(t *testing.T) {
	type fields struct {
		worker    *carbonv1.WorkerNode
		pod       *corev1.Pod
		allocator workerAllocator
		u         workerUpdater
		c         *Controller
	}

	ctl := gomock.NewController(t)
	defer ctl.Finish()
	updater := NewMockworkerUpdater(ctl)
	updater.EXPECT().processUpdateGracefully(gomock.Any()).Return(false).MinTimes(1)
	updaterprocessplan := NewMockworkerUpdater(ctl)
	updaterprocessplan.EXPECT().processUpdateGracefully(gomock.Any()).Return(true).MinTimes(1)
	updaterprocessplan.EXPECT().processPlan(gomock.Any()).Return(false, nil).MinTimes(1)
	updaterprocessHealthInfo := NewMockworkerUpdater(ctl)
	updaterprocessHealthInfo.EXPECT().processUpdateGracefully(gomock.Any()).Return(true).MinTimes(1)
	updaterprocessHealthInfo.EXPECT().processPlan(gomock.Any()).Return(true, nil).MinTimes(1)
	updaterprocessHealthInfo.EXPECT().processHealthInfo(gomock.Any()).Return(false).MinTimes(1)
	updaterprocessServiceInfo := NewMockworkerUpdater(ctl)
	updaterprocessServiceInfo.EXPECT().processUpdateGracefully(gomock.Any()).Return(true).MinTimes(1)
	updaterprocessServiceInfo.EXPECT().processPlan(gomock.Any()).Return(true, nil).MinTimes(1)
	updaterprocessServiceInfo.EXPECT().processHealthInfo(gomock.Any()).Return(true).MinTimes(1)
	updaterprocessServiceInfo.EXPECT().processServiceInfo(gomock.Any()).Return(true).MinTimes(1)

	serviceavailable := newTestWorkerWithServiceStatus("available", "ip-1", carbonv1.WorkerAssigned, carbonv1.ServiceAvailable)
	serviceavailable.Status.EntityName = "available"
	serviceavailable.Status.EntityAlloced = true
	serviceavailable.Status.EntityUid = "available-uid"
	tests := []struct {
		name     string
		fields   fields
		wantErr  bool
		wantStep carbonv1.ProcessStep
	}{
		{
			name: "lost",
			fields: fields{
				worker: newTestWorkerWithStatus("lost", "ip1", carbonv1.WorkerAssigned, ""),
				u:      updater,
			},
			wantErr:  false,
			wantStep: carbonv1.StepLost,
		},
		{
			name: "release",
			fields: fields{
				u:      updater,
				worker: newTestWorkerWithToRelease("release", "ip-1", carbonv1.WorkerAssigned, true),
			},
			wantErr:  false,
			wantStep: carbonv1.StepToRelease,
		},
		{
			name: "updategracefully",
			fields: fields{
				u:      updater,
				worker: newTestWorkerWithVersion("updategracefully", "ip-1", carbonv1.WorkerAssigned, "podname", "2", "1"),
			},
			wantErr:  false,
			wantStep: carbonv1.StepBegin,
		},
		{
			name: "processplan",
			fields: fields{
				u: updaterprocessplan,
				worker: func() *carbonv1.WorkerNode {
					worker := newTestWorkerWithVersion("updategracefully", "ip-1", carbonv1.WorkerAssigned, "podname", "2", "2")
					worker.Status.ServiceOffline = true
					return worker
				}(),
			},
			wantErr:  false,
			wantStep: carbonv1.StepProcessUpdateGracefully,
		},
		{
			name: "processHealthInfo",
			fields: fields{
				u:      updaterprocessHealthInfo,
				worker: newTestWorkerHealth("updategracefully", "ip-1", carbonv1.WorkerAssigned, "podname", "2", "2"),
			},
			wantErr:  false,
			wantStep: carbonv1.StepProcessPlan,
		},
		{
			name: "processServiceInfo",
			fields: fields{
				u:      updaterprocessServiceInfo,
				worker: serviceavailable,
			},
			wantErr:  false,
			wantStep: carbonv1.StepProcessServiceInfo,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &defaultStateMachine{
				worker:    tt.fields.worker,
				pod:       tt.fields.pod,
				allocator: tt.fields.allocator,
				u:         tt.fields.u,
				c:         tt.fields.c,
			}
			if err := m.dealwithAssigned(); (err != nil) != tt.wantErr {
				t.Errorf("defaultStateMachine.dealwithAssigned() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantStep != tt.fields.worker.Status.ProcessStep {
				t.Errorf("defaultStateMachine.dealwithAssigned() step wantStep = %s, step %s", tt.wantStep, tt.fields.worker.Status.ProcessStep)
			}
		})
	}
}

func Test_defaultStateMachine_processPlan(t *testing.T) {
	type fields struct {
		worker    *carbonv1.WorkerNode
		pod       *corev1.Pod
		allocator workerAllocator
		u         workerUpdater
		c         *Controller
	}

	ctl := gomock.NewController(t)
	defer ctl.Finish()
	allocator := NewMockworkerAllocator(ctl)
	allocator.EXPECT().syncPodSpec().Return(nil).MinTimes(1)
	allocator.EXPECT().syncWorkerStatus().Return(nil).MinTimes(1)

	features.C2MutableFeatureGate.Set("UnPubRestartingNode=true")
	type args struct {
		worker *carbonv1.WorkerNode
	}
	tests := []struct {
		name           string
		fields         fields
		wantWorkerSpec carbonv1.WorkerNodeSpec
		args           args
		want           bool
		want1          bool
		wantVersion    string
	}{
		{
			name: "changed",
			fields: fields{
				allocator: allocator,
			},
			args: args{
				worker: &carbonv1.WorkerNode{
					Spec: carbonv1.WorkerNodeSpec{
						Version: "replica-version2",
					},
				},
			},
			wantWorkerSpec: carbonv1.WorkerNodeSpec{
				Version: "replica-version1",
				VersionPlan: carbonv1.VersionPlan{
					BroadcastPlan: carbonv1.BroadcastPlan{
						WorkerSchedulePlan: carbonv1.WorkerSchedulePlan{
							ResourceMatchTimeout: 30,
							WorkerReadyTimeout:   60,
						},
					},
				},
			},
			want: false,
		},
		{
			name: "changedmatch",
			fields: fields{
				allocator: allocator,
			},
			args: args{
				worker: &carbonv1.WorkerNode{
					Spec: carbonv1.WorkerNodeSpec{
						Version: "replica-version2",
					},
					Status: carbonv1.WorkerNodeStatus{
						AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
							ResourceMatch: true,
							ProcessMatch:  true,
						},
					},
				},
			},
			wantWorkerSpec: carbonv1.WorkerNodeSpec{
				Version: "replica-version1",
				VersionPlan: carbonv1.VersionPlan{
					BroadcastPlan: carbonv1.BroadcastPlan{
						WorkerSchedulePlan: carbonv1.WorkerSchedulePlan{
							ResourceMatchTimeout: 30,
							WorkerReadyTimeout:   60,
						},
					},
				},
			},
			want: false,
		},
		{
			name: "versionchange",
			fields: fields{
				allocator: allocator,
			},
			args: args{
				worker: &carbonv1.WorkerNode{
					Spec: carbonv1.WorkerNodeSpec{
						Version: "replica-version2",
						VersionPlan: carbonv1.VersionPlan{
							BroadcastPlan: carbonv1.BroadcastPlan{
								UpdatingGracefully: utils.BoolPtr(false),
							},
						},
					},
					Status: carbonv1.WorkerNodeStatus{
						AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
							Version: "replica-version1",
						},
						InRestarting: true,
					},
				},
			},
			wantWorkerSpec: carbonv1.WorkerNodeSpec{
				Version: "replica-version1",
				VersionPlan: carbonv1.VersionPlan{
					BroadcastPlan: carbonv1.BroadcastPlan{
						WorkerSchedulePlan: carbonv1.WorkerSchedulePlan{
							ResourceMatchTimeout: 30,
							WorkerReadyTimeout:   60,
						},
					},
				},
			},
			want: false,
		},
		{
			name: "versionchange",
			fields: fields{
				allocator: allocator,
			},
			args: args{
				worker: &carbonv1.WorkerNode{
					Spec: carbonv1.WorkerNodeSpec{
						Version: "replica-version2",
					},
					Status: carbonv1.WorkerNodeStatus{
						AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
							Version: "replica-version1",
						},
					},
				},
			},
			wantWorkerSpec: carbonv1.WorkerNodeSpec{
				Version: "replica-version1",
				VersionPlan: carbonv1.VersionPlan{
					BroadcastPlan: carbonv1.BroadcastPlan{
						WorkerSchedulePlan: carbonv1.WorkerSchedulePlan{
							ResourceMatchTimeout: 30,
							WorkerReadyTimeout:   60,
						},
					},
				},
			},
			want: false,
		},
		{
			name: "notchanged1",
			fields: fields{
				allocator: allocator,
			},
			args: args{
				worker: &carbonv1.WorkerNode{
					Spec: carbonv1.WorkerNodeSpec{
						Version: "replica-version1",
					},
					Status: carbonv1.WorkerNodeStatus{
						AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
							Version: "replica-version1",
						},
						ServiceOffline: true,
					},
				},
			},
			wantWorkerSpec: carbonv1.WorkerNodeSpec{
				Version: "replica-version1",
				VersionPlan: carbonv1.VersionPlan{
					BroadcastPlan: carbonv1.BroadcastPlan{
						WorkerSchedulePlan: carbonv1.WorkerSchedulePlan{
							ResourceMatchTimeout: 30,
							WorkerReadyTimeout:   60,
						},
					},
				},
			},
			want: true,
		},
		{
			name: "notchanged2",
			fields: fields{
				allocator: allocator,
			},
			args: args{
				worker: &carbonv1.WorkerNode{
					Spec: carbonv1.WorkerNodeSpec{
						Version: "replica-version1",
						VersionPlan: carbonv1.VersionPlan{
							BroadcastPlan: carbonv1.BroadcastPlan{
								UpdatingGracefully: utils.BoolPtr(false),
							},
						},
					},
					Status: carbonv1.WorkerNodeStatus{
						AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
							Version: "replica-version1",
						},
						InRestarting:   true,
						ServiceOffline: true,
					},
				},
			},
			wantWorkerSpec: carbonv1.WorkerNodeSpec{
				Version: "replica-version1",
				VersionPlan: carbonv1.VersionPlan{
					BroadcastPlan: carbonv1.BroadcastPlan{
						WorkerSchedulePlan: carbonv1.WorkerSchedulePlan{
							ResourceMatchTimeout: 30,
							WorkerReadyTimeout:   60,
						},
					},
				},
			},
			want: true,
		},
		{
			name: "notchanged3",
			fields: fields{
				allocator: allocator,
			},
			args: args{
				worker: &carbonv1.WorkerNode{
					Spec: carbonv1.WorkerNodeSpec{
						Version: "replica-version1",
						VersionPlan: carbonv1.VersionPlan{
							BroadcastPlan: carbonv1.BroadcastPlan{
								UpdatingGracefully: utils.BoolPtr(false),
							},
						},
					},
					Status: carbonv1.WorkerNodeStatus{
						AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
							Version: "replica-version1",
						},
						InRestarting: true,
					},
				},
			},
			wantWorkerSpec: carbonv1.WorkerNodeSpec{
				Version: "replica-version1",
				VersionPlan: carbonv1.VersionPlan{
					BroadcastPlan: carbonv1.BroadcastPlan{
						WorkerSchedulePlan: carbonv1.WorkerSchedulePlan{
							ResourceMatchTimeout: 30,
							WorkerReadyTimeout:   60,
						},
					},
				},
			},
			want: true,
		},
		{
			name: "notchanged4",
			fields: fields{
				allocator: allocator,
			},
			args: args{
				worker: &carbonv1.WorkerNode{
					Spec: carbonv1.WorkerNodeSpec{
						Version: "replica-version1",
					},
					Status: carbonv1.WorkerNodeStatus{
						AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
							Version: "replica-version1",
						},
					},
				},
			},
			wantWorkerSpec: carbonv1.WorkerNodeSpec{
				Version: "replica-version1",
				VersionPlan: carbonv1.VersionPlan{
					BroadcastPlan: carbonv1.BroadcastPlan{
						WorkerSchedulePlan: carbonv1.WorkerSchedulePlan{
							ResourceMatchTimeout: 30,
							WorkerReadyTimeout:   60,
						},
					},
				},
			},
			want: true,
		},
		{
			name: "workermodechanged",
			fields: fields{
				allocator: allocator,
			},
			args: args{
				worker: &carbonv1.WorkerNode{
					Spec: carbonv1.WorkerNodeSpec{
						Version:    "replica-version1",
						WorkerMode: carbonv1.WorkerModeTypeHot,
					},
					Status: carbonv1.WorkerNodeStatus{
						AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
							Version: "replica-version1",
						},
					},
				},
			},
			wantWorkerSpec: carbonv1.WorkerNodeSpec{
				Version: "replica-version1",
				VersionPlan: carbonv1.VersionPlan{
					BroadcastPlan: carbonv1.BroadcastPlan{
						WorkerSchedulePlan: carbonv1.WorkerSchedulePlan{
							ResourceMatchTimeout: 30,
							WorkerReadyTimeout:   60,
						},
					},
				},
			},
			want: false,
		},
		{
			name: "worker version rollback",
			fields: fields{
				allocator: allocator,
			},
			args: args{
				worker: &carbonv1.WorkerNode{
					Spec: carbonv1.WorkerNodeSpec{
						Version:    "11-2203",
						WorkerMode: carbonv1.WorkerModeTypeActive,
					},
					Status: carbonv1.WorkerNodeStatus{
						AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
							Version: "12-2203",
						},
					},
				},
			},
			want: false,
		},
		{
			name: "worker version inner match",
			fields: fields{
				allocator: allocator,
			},
			args: args{
				worker: &carbonv1.WorkerNode{
					Spec: carbonv1.WorkerNodeSpec{
						Version:    "13-2203",
						WorkerMode: carbonv1.WorkerModeTypeActive,
					},
					Status: carbonv1.WorkerNodeStatus{
						AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
							Version: "12-2203",
						},
					},
				},
			},
			want:        true,
			wantVersion: "13-2203",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &defaultStateMachine{
				worker:    tt.fields.worker,
				pod:       tt.fields.pod,
				allocator: tt.fields.allocator,
				u:         tt.fields.u,
				c:         tt.fields.c,
			}
			got, got1 := m.processPlan(tt.args.worker)
			if got != tt.want {
				t.Errorf("defaultStateMachine.processPlan() = %v, want %v", got, tt.want)
			}
			if (got1 != nil) != tt.want1 {
				t.Errorf("defaultStateMachine.processPlan() = %v, want %v", got, tt.want)
			}
			if tt.wantVersion != "" && tt.wantVersion != tt.args.worker.Status.Version {
				t.Errorf("defaultStateMachine.processPlan() = %v, want %v", tt.args.worker.Status.Version, tt.wantVersion)
			}
		})
	}
}

func Test_defaultStateMachine_processHealthInfo(t *testing.T) {
	type fields struct {
		worker    *carbonv1.WorkerNode
		pod       *corev1.Pod
		allocator workerAllocator
		u         workerUpdater
		c         *Controller
	}
	type args struct {
		worker *carbonv1.WorkerNode
	}
	workerUnavailable := newTestWorkerWithVersion("assinged", "ip-1", carbonv1.WorkerAssigned, "pod1", "version1", "version2")
	workerUnavailable.Status.ServiceStatus = carbonv1.ServiceUnAvailable
	tests := []struct {
		name      string
		fields    fields
		args      args
		want      bool
		offlining bool
	}{
		{
			name: "Ready",
			args: args{
				worker: &carbonv1.WorkerNode{
					Status: carbonv1.WorkerNodeStatus{
						AllocStatus: carbonv1.WorkerAssigned,
						AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
							ProcessMatch: true,
							Phase:        carbonv1.Running,
							ProcessReady: true,
						},
						HealthStatus:    carbonv1.HealthAlive,
						HealthCondition: carbonv1.HealthCondition{WorkerStatus: carbonv1.WorkerTypeReady},
					},
				},
			},
			want: true,
		},
		{
			name: "NotReady",
			args: args{
				worker: &carbonv1.WorkerNode{
					Status: carbonv1.WorkerNodeStatus{
						AllocStatus:     carbonv1.WorkerAssigned,
						HealthCondition: carbonv1.HealthCondition{WorkerStatus: carbonv1.WorkerTypeReady},

						HealthStatus: carbonv1.HealthAlive,
						AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
							ResourceMatch: false,
							Phase:         carbonv1.Running,
						},
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &defaultStateMachine{
				worker:    tt.fields.worker,
				pod:       tt.fields.pod,
				allocator: tt.fields.allocator,
				u:         tt.fields.u,
				c:         tt.fields.c,
			}
			if got := m.processHealthInfo(tt.args.worker); got != tt.want {
				t.Errorf("defaultStateMachine.processHealthInfo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_defaultStateMachine_processServiceInfo(t *testing.T) {
	type fields struct {
		worker    *carbonv1.WorkerNode
		pod       *corev1.Pod
		allocator workerAllocator
		u         workerUpdater
		c         *Controller
	}
	type args struct {
		worker *carbonv1.WorkerNode
	}

	unavailable := newTestWorkerWithServiceStatus("releasing", "ip-1", carbonv1.WorkerOfflining, carbonv1.ServiceUnAvailable)
	unavailable.Spec.Online = utils.BoolPtr(false)

	offlineavailable := newTestWorkerWithServiceStatus("releasing", "ip-1", carbonv1.WorkerOfflining, carbonv1.ServiceAvailable)
	offlineavailable.Spec.Online = utils.BoolPtr(false)

	onlineunavailable := newTestWorkerWithServiceStatus("releasing", "ip-1", carbonv1.WorkerOfflining, carbonv1.ServiceUnAvailable)
	onlineunavailable.Status.ServiceConditions = []carbonv1.ServiceCondition{
		{
			Status: corev1.ConditionFalse,
		},
	}
	tests := []struct {
		name       string
		fields     fields
		args       args
		want       bool
		wantWarmup bool
	}{
		{
			name: "match offline",
			args: args{
				worker: unavailable,
			},
			want: true,
		},
		{
			name: "not match offline",
			args: args{
				worker: offlineavailable,
			},
			want: false,
		},
		{
			name: "online match",
			args: args{
				worker: newTestWorkerWithServiceStatus("releasing", "ip-1", carbonv1.WorkerOfflining, carbonv1.ServiceAvailable),
			},
			want: true,
		},
		{
			name: "online not match",
			args: args{
				worker: onlineunavailable,
			},
			want: false,
		},
		{
			name: "warmup-start",
			args: args{
				worker: &carbonv1.WorkerNode{
					Spec: carbonv1.WorkerNodeSpec{
						VersionPlan: carbonv1.VersionPlan{
							BroadcastPlan: carbonv1.BroadcastPlan{
								WorkerSchedulePlan: carbonv1.WorkerSchedulePlan{
									WarmupSeconds: 60,
								},
							},
						},
					},
					Status: carbonv1.WorkerNodeStatus{
						NeedWarmup:     true,
						ServiceOffline: true,
					},
				},
			},
			want:       true,
			wantWarmup: true,
		},
		{
			name: "warmup-start",
			args: args{
				worker: &carbonv1.WorkerNode{
					Spec: carbonv1.WorkerNodeSpec{
						VersionPlan: carbonv1.VersionPlan{
							BroadcastPlan: carbonv1.BroadcastPlan{
								WorkerSchedulePlan: carbonv1.WorkerSchedulePlan{
									WarmupSeconds: 60,
								},
								RowComplete: utils.BoolPtr(false),
							},
						},
					},
					Status: carbonv1.WorkerNodeStatus{
						NeedWarmup:     true,
						ServiceOffline: true,
					},
				},
			},
			want:       false,
			wantWarmup: false,
		},
		{
			name: "warmup-end",
			args: args{
				worker: &carbonv1.WorkerNode{
					Spec: carbonv1.WorkerNodeSpec{
						VersionPlan: carbonv1.VersionPlan{
							BroadcastPlan: carbonv1.BroadcastPlan{
								WorkerSchedulePlan: carbonv1.WorkerSchedulePlan{
									WarmupSeconds: 60,
								},
							},
						},
					},
					Status: carbonv1.WorkerNodeStatus{
						Warmup:         true,
						ServiceOffline: false,
						WorkerStateChangeRecoder: carbonv1.WorkerStateChangeRecoder{
							LastWarmupStartTime: 123,
						},
					},
				},
			},
			want:       true,
			wantWarmup: false,
		},
		{
			name: "warmup-end-not-started",
			args: args{
				worker: &carbonv1.WorkerNode{
					Spec: carbonv1.WorkerNodeSpec{
						VersionPlan: carbonv1.VersionPlan{
							BroadcastPlan: carbonv1.BroadcastPlan{
								WorkerSchedulePlan: carbonv1.WorkerSchedulePlan{
									WarmupSeconds: 60,
								},
							},
						},
					},
					Status: carbonv1.WorkerNodeStatus{
						Warmup:         true,
						ServiceOffline: false,
					},
				},
			},
			want:       true,
			wantWarmup: true,
		},
		{
			name: "warmup-end-not-end",
			args: args{
				worker: &carbonv1.WorkerNode{
					Spec: carbonv1.WorkerNodeSpec{
						VersionPlan: carbonv1.VersionPlan{
							BroadcastPlan: carbonv1.BroadcastPlan{
								WorkerSchedulePlan: carbonv1.WorkerSchedulePlan{
									WarmupSeconds: 60,
								},
							},
						},
					},
					Status: carbonv1.WorkerNodeStatus{
						Warmup:         true,
						ServiceOffline: false,
						WorkerStateChangeRecoder: carbonv1.WorkerStateChangeRecoder{
							LastWarmupStartTime: time.Now().Unix(),
						},
					},
				},
			},
			want:       true,
			wantWarmup: true,
		},
	}
	for _, tt := range tests {
		features.C2MutableFeatureGate.Set("WarmupUnavailablePod=true")
		t.Run(tt.name, func(t *testing.T) {
			m := &defaultStateMachine{
				worker:    tt.fields.worker,
				pod:       tt.fields.pod,
				allocator: tt.fields.allocator,
				u:         tt.fields.u,
				c:         tt.fields.c,
			}
			got := m.processServiceInfo(tt.args.worker)
			if got != tt.want {
				t.Errorf("defaultStateMachine.processServiceInfo() = %v, want %v", got, tt.want)
			}
			if tt.wantWarmup != tt.args.worker.Status.Warmup {
				t.Errorf("defaultStateMachine.processServiceInfo() = %v, want %v", tt.args.worker.Status.Warmup, tt.wantWarmup)
			}
		})
	}
}

func Test_defaultStateMachine_processUpdateGracefully(t *testing.T) {
	type fields struct {
		worker    *carbonv1.WorkerNode
		pod       *corev1.Pod
		allocator workerAllocator
		u         workerUpdater
		c         *Controller
	}
	type args struct {
		worker *carbonv1.WorkerNode
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		want      bool
		offlining bool
	}{
		{
			name: "versionChanged",
			args: args{
				worker: func() *carbonv1.WorkerNode {
					workerChange := newTestWorkerWithVersion("assinged", "ip-1", carbonv1.WorkerAssigned, "pod1", "version1", "version2")
					return workerChange
				}(),
			},
			want:      true,
			offlining: true,
		},
		{
			name: "versionChangedUnavailable",
			args: args{
				worker: func() *carbonv1.WorkerNode {
					workerUnavailable := newTestWorkerWithVersion("assinged", "ip-1", carbonv1.WorkerAssigned, "pod1", "version1", "version2")
					workerUnavailable.Status.ServiceStatus = carbonv1.ServiceUnAvailable
					return workerUnavailable
				}(),
			},
			want:      true,
			offlining: true,
		},
		{
			name: "versionChangedUnpublished",
			args: args{
				worker: func() *carbonv1.WorkerNode {
					workerUnavailable := newTestWorkerWithVersion("assinged", "ip-1", carbonv1.WorkerAssigned, "pod1", "version1", "version2")
					workerUnavailable.Status.ServiceStatus = carbonv1.ServiceUnAvailable
					workerUnavailable.Status.ServiceConditions = []carbonv1.ServiceCondition{
						{
							Type:   carbonv1.ServiceSkyline,
							Status: corev1.ConditionTrue,
						},
						{
							Type:   carbonv1.ServiceVipserver,
							Status: corev1.ConditionFalse,
							Reason: carbonv1.ReasonServiceCheckFalse,
						},
					}
					return workerUnavailable
				}(),
			},
			want:      false,
			offlining: true,
		},
		{
			name: "versionChangedUnpublished",
			args: args{
				worker: func() *carbonv1.WorkerNode {
					workerUnavailable := newTestWorkerWithVersion("assinged", "ip-1", carbonv1.WorkerAssigned, "pod1", "version1", "version2")
					workerUnavailable.Status.ServiceStatus = carbonv1.ServiceUnAvailable
					workerUnavailable.Status.ServiceConditions = []carbonv1.ServiceCondition{
						{
							Type:   carbonv1.ServiceSkyline,
							Status: corev1.ConditionTrue,
						},
						{
							Type:   carbonv1.ServiceVipserver,
							Status: corev1.ConditionFalse,
							Reason: carbonv1.ReasonNotExist,
						},
					}
					return workerUnavailable
				}(),
			},
			want:      true,
			offlining: true,
		},
		{
			name: "versionMatch",
			args: args{
				worker: func() *carbonv1.WorkerNode {
					workerSame := newTestWorkerWithVersion("assinged", "ip-1", carbonv1.WorkerAssigned, "pod1", "version1", "version1")
					return workerSame
				}(),
			},
			want:      true,
			offlining: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &defaultStateMachine{
				worker:    tt.fields.worker,
				pod:       tt.fields.pod,
				allocator: tt.fields.allocator,
				u:         tt.fields.u,
				c:         tt.fields.c,
			}
			if got := m.processUpdateGracefully(tt.args.worker); got != tt.want {
				t.Errorf("defaultStateMachine.processUpdateGracefully() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_defaultStateMachine_tryToReleaseWorker(t *testing.T) {
	type fields struct {
		worker    *carbonv1.WorkerNode
		pod       *corev1.Pod
		allocator workerAllocator
		u         workerUpdater
		c         *Controller
	}
	type args struct {
		worker *carbonv1.WorkerNode
	}
	tests := []struct {
		name       string
		fields     fields
		args       args
		wantStatus carbonv1.AllocStatus
	}{
		{
			name: "release",
			args: args{
				worker: newTestWorkerWithStatus("assinged", "ip-1", carbonv1.WorkerAssigned, "pod1"),
			},
			wantStatus: carbonv1.WorkerOfflining,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &defaultStateMachine{
				worker:    tt.fields.worker,
				pod:       tt.fields.pod,
				allocator: tt.fields.allocator,
				u:         tt.fields.u,
				c:         tt.fields.c,
			}
			m.tryToReleaseWorker(tt.args.worker)
			if tt.args.worker.Status.AllocStatus != tt.wantStatus {
				t.Errorf("simpleWorkerScheduler.tryToReleaseWorker() Status = %v, wantStatus %v", tt.args.worker.Status.AllocStatus, tt.wantStatus)
			}
		})
	}
}

func Test_defaultStateMachine_initServiceConditions(t *testing.T) {
	type fields struct {
		worker    *carbonv1.WorkerNode
		pod       *corev1.Pod
		allocator workerAllocator
		u         workerUpdater
		c         *Controller
	}

	client := fake.NewSimpleClientset([]runtime.Object{}...)
	informersFactory := carboninformers.NewSharedInformerFactory(client, controller.NoResyncPeriodFunc())
	serviceLister := informersFactory.Carbon().V1().ServicePublishers().Lister()

	service := newTestService("test", map[string]string{"foo": "bar"})
	service.Labels = map[string]string{"rs-version-hash": "test"}
	serviceOther := newTestService("test1", map[string]string{"foo": "bars"})
	informersFactory.Carbon().V1().ServicePublishers().Informer().GetIndexer().Add(service)
	informersFactory.Carbon().V1().ServicePublishers().Informer().GetIndexer().Add(serviceOther)

	worker := newTestWorker("test", "1.1.1.1", nil)
	worker.Labels = labelsutil.CloneAndAddLabel(nil, "foo", "bar")
	worker.Labels["rs-version-hash"] = "test"
	c := &Controller{
		carbonclientset: client,
		serviceLister:   serviceLister,
	}
	type args struct {
		worker *carbonv1.WorkerNode
	}
	tests := []struct {
		name           string
		fields         fields
		args           args
		wantErr        bool
		wantConditions []carbonv1.ServiceCondition
	}{
		{
			name: "init",
			fields: fields{
				c: c,
			},
			args: args{
				worker: worker,
			},
			wantErr: false,
			wantConditions: []carbonv1.ServiceCondition{
				{
					Name:   "test",
					Status: corev1.ConditionFalse,
					Reason: "init with false",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &defaultStateMachine{
				worker:    tt.fields.worker,
				pod:       tt.fields.pod,
				allocator: tt.fields.allocator,
				u:         tt.fields.u,
				c:         tt.fields.c,
			}
			if err := m.initServiceConditions(tt.args.worker); (err != nil) != tt.wantErr {
				t.Errorf("defaultStateMachine.initServiceConditions() error = %v, wantErr %v", err, tt.wantErr)
			}
			tt.wantConditions[0].LastTransitionTime = tt.args.worker.Status.ServiceConditions[0].LastTransitionTime
			if !reflect.DeepEqual(tt.wantConditions, tt.args.worker.Status.ServiceConditions) {
				t.Errorf("defaultStateMachine.initServiceConditions() conditions = %s, wantConditions %s",
					utils.ObjJSON(tt.args.worker.Status.ServiceConditions), utils.ObjJSON(tt.wantConditions))
			}
		})
	}
}

func newTestService(name string, selector map[string]string) *carbonv1.ServicePublisher {
	var service = carbonv1.ServicePublisher{
		ObjectMeta: metav1.ObjectMeta{
			UID:         uuid.NewUUID(),
			Name:        name,
			Namespace:   metav1.NamespaceDefault,
			Annotations: make(map[string]string),
		},
		Spec: carbonv1.ServicePublisherSpec{
			Selector: selector,
		},
	}
	return &service
}

func Test_getWorkerServices(t *testing.T) {
	type args struct {
		rslister  listers.RollingSetLister
		clientset clientset.Interface
		lister    listers.ServicePublisherLister
		worker    *carbonv1.WorkerNode
	}

	client := fake.NewSimpleClientset([]runtime.Object{}...)
	informersFactory := carboninformers.NewSharedInformerFactory(client, controller.NoResyncPeriodFunc())
	serviceLister := informersFactory.Carbon().V1().ServicePublishers().Lister()

	service := newTestService("test", map[string]string{"foo": "bar"})
	service.Labels = map[string]string{"rs-version-hash": "test"}
	serviceOther := newTestService("test1", map[string]string{"foo": "bars"})
	informersFactory.Carbon().V1().ServicePublishers().Informer().GetIndexer().Add(service)
	informersFactory.Carbon().V1().ServicePublishers().Informer().GetIndexer().Add(serviceOther)

	worker := newTestWorker("test", "1.1.1.1", nil)
	worker.Labels = labelsutil.CloneAndAddLabel(nil, "foo", "bar")
	worker.Labels["rs-version-hash"] = "test"
	worker.Labels["gang-version-hash"] = "test-gang"

	tests := []struct {
		name    string
		args    args
		want    []*carbonv1.ServicePublisher
		wantErr bool
	}{
		{
			name: "TestGetService",
			args: args{
				lister: serviceLister, worker: worker,
			},
			want:    []*carbonv1.ServicePublisher{service},
			wantErr: false,
		},
		{
			name: "TestGangGetService",
			args: args{
				lister: serviceLister, worker: worker,
			},
			want:    []*carbonv1.ServicePublisher{service},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getWorkerServices(tt.args.rslister, tt.args.clientset, tt.args.lister, tt.args.worker)
			cache := append(got, got[0])
			serviceCache.SetWithExpire("test", cache, lruTimeout)
			if (err != nil) != tt.wantErr {
				t.Errorf("getWorkerServices() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			tt.want[0].UID = got[0].UID
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getWorkerServices() = %v, want %v", utils.ObjJSON(got), utils.ObjJSON(tt.want))
			}
		})
	}
}
