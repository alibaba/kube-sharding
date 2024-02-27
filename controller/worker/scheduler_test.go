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
	"fmt"
	"testing"

	"github.com/alibaba/kube-sharding/controller"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	gomock "github.com/golang/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func newTestWorker(name, ip string, selector map[string]string) *carbonv1.WorkerNode {
	var worker = carbonv1.WorkerNode{
		ObjectMeta: metav1.ObjectMeta{
			UID:         types.UID("abcd"),
			Name:        name,
			Namespace:   metav1.NamespaceDefault,
			Annotations: make(map[string]string),
			Labels:      make(map[string]string),
		},
		Spec: carbonv1.WorkerNodeSpec{},
		Status: carbonv1.WorkerNodeStatus{
			AllocStatus:     carbonv1.WorkerAssigned,
			ServiceStatus:   carbonv1.ServicePartAvailable,
			HealthStatus:    carbonv1.HealthAlive,
			HealthCondition: carbonv1.HealthCondition{WorkerStatus: carbonv1.WorkerTypeReady},
			AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
				Phase: carbonv1.Running,
				IP:    ip,
			},
		},
	}
	return &worker
}

func newTestWorkerPair(name, ip string, selector map[string]string) (*carbonv1.WorkerNode, *carbonv1.WorkerNode) {
	var worker = carbonv1.WorkerNode{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name + "-a",
			Namespace:   metav1.NamespaceDefault,
			Annotations: make(map[string]string),
			Labels: map[string]string{
				carbonv1.DefaultReplicaUniqueLabelKey: name,
			},
		},
		Spec: carbonv1.WorkerNodeSpec{},
		Status: carbonv1.WorkerNodeStatus{
			AllocStatus:     carbonv1.WorkerAssigned,
			ServiceStatus:   carbonv1.ServicePartAvailable,
			HealthStatus:    carbonv1.HealthAlive,
			HealthCondition: carbonv1.HealthCondition{WorkerStatus: carbonv1.WorkerTypeReady},
			AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
				Phase: carbonv1.Running,
				IP:    ip,
			},
		},
	}
	carbonv1.SetWorkerCurrent(&worker)
	var pair = carbonv1.WorkerNode{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name + "-b",
			Namespace:   metav1.NamespaceDefault,
			Annotations: make(map[string]string),
			Labels: map[string]string{
				carbonv1.DefaultReplicaUniqueLabelKey: name,
			},
		},
		Spec: carbonv1.WorkerNodeSpec{},
		Status: carbonv1.WorkerNodeStatus{
			AllocStatus:     carbonv1.WorkerAssigned,
			ServiceStatus:   carbonv1.ServicePartAvailable,
			HealthStatus:    carbonv1.HealthAlive,
			HealthCondition: carbonv1.HealthCondition{WorkerStatus: carbonv1.WorkerTypeReady},
			AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
				Phase: carbonv1.Running,
				IP:    ip,
			},
		},
	}
	carbonv1.SetWorkerBackup(&pair)
	return &worker, &pair
}

func setWorkerComplete(worker *carbonv1.WorkerNode) {
	worker.Status.ServiceStatus = carbonv1.ServiceAvailable
	worker.Status.AllocStatus = carbonv1.WorkerAssigned
	worker.Status.HealthStatus = carbonv1.HealthAlive
	worker.Status.Phase = carbonv1.Running
	worker.Status.ResourceMatch = true
	worker.Status.ProcessMatch = true
	worker.Status.ProcessReady = true
}

func Test_simpleReplicaScheduler_schedule(t *testing.T) {
	type fields struct {
		worker    *carbonv1.WorkerNode
		pair      *carbonv1.WorkerNode
		pod       *corev1.Pod
		adjuster  workerAdjuster
		c         *Controller
		processor processor
	}
	worker := newTestWorker("test", "1.1.1.1", map[string]string{"foo": "bar"})
	pair := newTestWorker("test", "2.2.2.2", map[string]string{"foo": "bar"})
	worker.Spec.Releasing = true
	worker.Status.ToRelease = true
	worker.Labels[carbonv1.WorkerRoleKey] = carbonv1.CurrentWorkerKey

	ctl := gomock.NewController(t)
	defer ctl.Finish()
	adjuster := NewMockworkerAdjuster(ctl)
	adjuster.EXPECT().adjust().Return(nil, nil, false, nil).MinTimes(1)
	processor := NewMockprocessor(ctl)
	processor.EXPECT().process().Return(worker, nil, nil).MinTimes(1)

	var c = &Controller{}
	c.DefaultController = *controller.NewDefaultController(nil, nil, "a", controllerKind, c)
	adjusterNormalErr := NewMockworkerAdjuster(ctl)
	adjusterNormalErr.EXPECT().adjust().Return(nil, nil, false, fmt.Errorf("normal error")).MinTimes(1)

	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "scheduleReleasing",
			fields: fields{
				worker:    worker,
				pair:      pair,
				adjuster:  adjuster,
				processor: processor,
			},
			wantErr: false,
		},
		{
			name: "normal err",
			fields: fields{
				worker:    worker,
				pair:      pair,
				adjuster:  adjusterNormalErr,
				processor: processor,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &defaultWorkerScheduler{
				worker:    tt.fields.worker,
				pair:      tt.fields.pair,
				adjuster:  tt.fields.adjuster,
				processor: tt.fields.processor,
				c:         tt.fields.c,
			}
			if _, _, _, err := s.schedule(); (err != nil) != tt.wantErr {
				t.Errorf("defaultWorkerScheduler.schedule() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
