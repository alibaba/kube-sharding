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

package publisher

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_isSliceEqual(t *testing.T) {
	type args struct {
		a []string
		b []string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "equal",
			args: args{
				a: nil,
				b: []string{},
			},
			want: true,
		},
		{
			name: "equal",
			args: args{
				a: []string{},
				b: []string{},
			},
			want: true,
		},
		{
			name: "equal",
			args: args{
				a: []string{"a"},
				b: []string{"a", "b"},
			},
			want: false,
		},
		{
			name: "equal",
			args: args{
				a: []string{"a"},
				b: []string{"a"},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSliceEqual(tt.args.a, tt.args.b)
			if got != tt.want {
				t.Errorf("got error %t,%t", got, tt.want)
				return
			}
		})
	}
}

func TestController_syncWorkerServiceStatus(t *testing.T) {
	type fields struct {
	}
	type args struct {
		service           *carbonv1.ServicePublisher
		worker            *carbonv1.WorkerNode
		remoteIps         map[string]carbonv1.IPNode
		serviceReleaseing bool
		softDelete        bool
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   carbonv1.WorkerNodeStatus
	}{
		{
			name: "release worker",
			args: args{
				service: &carbonv1.ServicePublisher{
					ObjectMeta: metav1.ObjectMeta{
						Name: "service1",
					},
					Spec: carbonv1.ServicePublisherSpec{
						Type:        carbonv1.ServiceCM2,
						ServiceName: "serviceName",
					},
				},
				worker: &carbonv1.WorkerNode{
					ObjectMeta: metav1.ObjectMeta{
						Name: "worker1",
					},
					Spec: carbonv1.WorkerNodeSpec{
						ToDelete: true,
					},
					Status: carbonv1.WorkerNodeStatus{
						ServiceStatus:  carbonv1.ServiceAvailable,
						ServiceOffline: true,
						ServiceConditions: []carbonv1.ServiceCondition{
							{
								Type:        carbonv1.ServiceCM2,
								Name:        "service1",
								ServiceName: "serviceName",
								Status:      corev1.ConditionTrue,
								DeleteCount: 1,
							},
						},
						AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
							IP: "1.2.1.2",
						},
					},
				},
				remoteIps: map[string]carbonv1.IPNode{},
			},
			want: carbonv1.WorkerNodeStatus{
				ServiceStatus:  carbonv1.ServiceAvailable,
				ServiceOffline: true,
				ServiceConditions: []carbonv1.ServiceCondition{
					{
						Type:        carbonv1.ServiceCM2,
						Name:        "service1",
						ServiceName: "serviceName",
						Status:      corev1.ConditionTrue,
						DeleteCount: 0,
					},
				},
				AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
					IP: "1.2.1.2",
				},
			},
		},
		{
			name: "release worker",
			args: args{
				service: &carbonv1.ServicePublisher{
					ObjectMeta: metav1.ObjectMeta{
						Name: "service1",
					},
					Spec: carbonv1.ServicePublisherSpec{
						Type:        carbonv1.ServiceCM2,
						ServiceName: "serviceName",
					},
				},
				worker: &carbonv1.WorkerNode{
					ObjectMeta: metav1.ObjectMeta{
						Name: "worker1",
					},
					Spec: carbonv1.WorkerNodeSpec{
						ToDelete: true,
					},
					Status: carbonv1.WorkerNodeStatus{
						ServiceStatus:  carbonv1.ServiceAvailable,
						ServiceOffline: true,
						ServiceConditions: []carbonv1.ServiceCondition{
							{
								Type:        carbonv1.ServiceCM2,
								ServiceName: "service1",
								Status:      corev1.ConditionTrue,
								DeleteCount: 0,
							},
						},
						AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
							IP: "1.2.1.2",
						},
					},
				},
				remoteIps: map[string]carbonv1.IPNode{},
			},
			want: carbonv1.WorkerNodeStatus{
				ServiceStatus:     carbonv1.ServiceUnAvailable,
				ServiceOffline:    true,
				ServiceConditions: []carbonv1.ServiceCondition{},
				AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
					IP: "1.2.1.2",
				},
			},
		},
		{
			name: "unpublish service",
			args: args{
				service: &carbonv1.ServicePublisher{
					ObjectMeta: metav1.ObjectMeta{
						Name: "service1",
					},
					Spec: carbonv1.ServicePublisherSpec{
						Type:        carbonv1.ServiceCM2,
						ServiceName: "serviceName",
					},
				},
				worker: &carbonv1.WorkerNode{
					ObjectMeta: metav1.ObjectMeta{
						Name: "worker1",
					},
					Spec: carbonv1.WorkerNodeSpec{},
					Status: carbonv1.WorkerNodeStatus{
						ServiceStatus:  carbonv1.ServiceAvailable,
						ServiceOffline: true,
						ServiceConditions: []carbonv1.ServiceCondition{
							{
								Type:        carbonv1.ServiceCM2,
								ServiceName: "service1",
								Status:      corev1.ConditionTrue,
							},
						},
						AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
							IP: "1.2.1.2",
						},
					},
				},
				remoteIps:         map[string]carbonv1.IPNode{},
				serviceReleaseing: true,
			},
			want: carbonv1.WorkerNodeStatus{
				ServiceStatus:     carbonv1.ServiceUnAvailable,
				ServiceOffline:    true,
				ServiceConditions: []carbonv1.ServiceCondition{},
				AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
					IP: "1.2.1.2",
				},
			},
		},
		{
			name: "rolling worker",
			args: args{
				service: &carbonv1.ServicePublisher{
					ObjectMeta: metav1.ObjectMeta{
						Name: "service1",
					},
					Spec: carbonv1.ServicePublisherSpec{
						Type:        carbonv1.ServiceCM2,
						ServiceName: "serviceName",
					},
				},
				worker: &carbonv1.WorkerNode{
					ObjectMeta: metav1.ObjectMeta{
						Name: "worker1",
					},
					Spec: carbonv1.WorkerNodeSpec{},
					Status: carbonv1.WorkerNodeStatus{
						ServiceStatus:  carbonv1.ServiceAvailable,
						ServiceOffline: true,
						ServiceConditions: []carbonv1.ServiceCondition{
							{
								Type:        carbonv1.ServiceCM2,
								ServiceName: "service1",
								Status:      corev1.ConditionTrue,
							},
						},
						AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
							IP: "1.2.1.2",
						},
					},
				},
				remoteIps: map[string]carbonv1.IPNode{},
			},
			want: carbonv1.WorkerNodeStatus{
				ServiceStatus:  carbonv1.ServiceUnAvailable,
				ServiceOffline: true,
				ServiceConditions: []carbonv1.ServiceCondition{
					{
						Type:        carbonv1.ServiceCM2,
						Name:        "service1",
						ServiceName: "serviceName",
						Reason:      "Service Not Exist Node",
						Status:      corev1.ConditionFalse,
					},
				},
				ServiceInfoMetasRecoverd: true,
				AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
					IP: "1.2.1.2",
				},
				ServiceInfoMetas: "{}",
			},
		},
		{
			name: "new worker",
			args: args{
				service: &carbonv1.ServicePublisher{
					ObjectMeta: metav1.ObjectMeta{
						Name: "service1",
					},
					Spec: carbonv1.ServicePublisherSpec{
						Type:        carbonv1.ServiceGeneral,
						ServiceName: "serviceName",
					},
				},
				worker: &carbonv1.WorkerNode{
					ObjectMeta: metav1.ObjectMeta{
						Name: "worker1",
					},
					Spec: carbonv1.WorkerNodeSpec{},
					Status: carbonv1.WorkerNodeStatus{
						ServiceStatus: carbonv1.ServiceAvailable,
						ServiceConditions: []carbonv1.ServiceCondition{
							{
								Type:        carbonv1.ServiceGeneral,
								ServiceName: "service1",
								Status:      corev1.ConditionFalse,
							},
						},
						AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
							IP: "1.2.1.2",
						},
					},
				},
				remoteIps: map[string]carbonv1.IPNode{
					"1.2.1.2": carbonv1.IPNode{
						IP:     "1.2.1.2",
						Health: true,
					},
				},
			},
			want: carbonv1.WorkerNodeStatus{
				ServiceStatus: carbonv1.ServiceAvailable,
				ServiceConditions: []carbonv1.ServiceCondition{
					{
						Type:        carbonv1.ServiceGeneral,
						Name:        "service1",
						ServiceName: "serviceName",
						Message:     "Node Published",
						Status:      corev1.ConditionTrue,
						DeleteCount: 1,
					},
				},
				AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
					IP: "1.2.1.2",
				},
			},
		},
		{
			name: "new worker",
			args: args{
				service: &carbonv1.ServicePublisher{
					ObjectMeta: metav1.ObjectMeta{
						Name: "service1",
					},
					Spec: carbonv1.ServicePublisherSpec{
						Type:        carbonv1.ServiceCM2,
						ServiceName: "serviceName",
					},
				},
				worker: &carbonv1.WorkerNode{
					ObjectMeta: metav1.ObjectMeta{
						Name: "worker1",
					},
					Spec: carbonv1.WorkerNodeSpec{},
					Status: carbonv1.WorkerNodeStatus{
						ServiceConditions: []carbonv1.ServiceCondition{},
						AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
							IP: "1.2.1.2",
						},
					},
				},
				remoteIps: map[string]carbonv1.IPNode{
					"1.2.1.2": carbonv1.IPNode{
						IP:     "1.2.1.2",
						Health: true,
					},
				},
			},
			want: carbonv1.WorkerNodeStatus{
				ServiceStatus: carbonv1.ServiceAvailable,
				ServiceConditions: []carbonv1.ServiceCondition{
					{
						Type:        carbonv1.ServiceCM2,
						Name:        "service1",
						ServiceName: "serviceName",
						Message:     "Node Published",
						Status:      corev1.ConditionTrue,
					},
				},
				ServiceInfoMetasRecoverd: true,
				AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
					IP: "1.2.1.2",
				},
				ServiceInfoMetas: "{}",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Controller{}
			c.syncWorkerServiceStatus(tt.args.service, tt.args.worker, tt.args.remoteIps, tt.args.serviceReleaseing, tt.args.softDelete, nil)
			for i := range tt.args.worker.Status.ServiceConditions {
				tt.args.worker.Status.ServiceConditions[i].LastTransitionTime = metav1.Time{}
			}
			if !reflect.DeepEqual(tt.want, tt.args.worker.Status) {
				t.Errorf("syncWorkerServiceStatus() got = %v, want %v", utils.ObjJSON(tt.args.worker.Status), utils.ObjJSON(tt.want))
			}
		})
	}
}

func Test_publishTask_conditionRemoved(t *testing.T) {
	type fields struct {
		c         *Controller
		key       string
		publisher *abstractPublisher
	}
	type args struct {
		name    string
		workers []*carbonv1.WorkerNode
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "true",
			args: args{
				name: "test1",
				workers: []*carbonv1.WorkerNode{
					{
						Status: carbonv1.WorkerNodeStatus{
							ServiceConditions: []carbonv1.ServiceCondition{
								{
									Name: "test2",
								},
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
				name: "test1",
				workers: []*carbonv1.WorkerNode{
					{
						Status: carbonv1.WorkerNodeStatus{},
					},
				},
			},
			want: true,
		},
		{
			name: "false",
			args: args{
				name: "test1",
				workers: []*carbonv1.WorkerNode{
					{
						Status: carbonv1.WorkerNodeStatus{
							ServiceConditions: []carbonv1.ServiceCondition{
								{
									Name: "test2",
								},
								{
									Name: "test1",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "false",
			args: args{
				name: "test1",
				workers: []*carbonv1.WorkerNode{
					{
						Status: carbonv1.WorkerNodeStatus{
							ServiceConditions: []carbonv1.ServiceCondition{
								{
									Name: "test1",
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
			tr := &publishTask{
				c:         tt.fields.c,
				key:       tt.fields.key,
				publisher: tt.fields.publisher,
			}
			if got := tr.conditionRemoved(tt.args.name, tt.args.workers); got != tt.want {
				t.Errorf("publishTask.conditionRemoved() = %v, want %v", got, tt.want)
			}
		})
	}
}

type testcodeErrorImpl struct {
	code int
	err  string
}

func (e *testcodeErrorImpl) GetCode() int {
	return e.code
}

func (e *testcodeErrorImpl) Error() string {
	return e.err
}

func newTestCodeErrorImpl(err string, code int) *testcodeErrorImpl {
	return &testcodeErrorImpl{
		code: code,
		err:  err,
	}
}

func Test_publishTask_isNameServerUnavailable(t *testing.T) {
	type fields struct {
		c   *Controller
		key string
	}
	type args struct {
		err error
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "not code error",
			args: args{
				err: fmt.Errorf("test error"),
			},
			want: false,
		},
		{
			name: "is code error, not unavailable",
			args: args{
				err: newTestCodeErrorImpl("test error", 500),
			},
			want: false,
		},
		{
			name: "is code error, is unavailable",
			args: args{
				err: newTestCodeErrorImpl("test error", 503),
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &publishTask{
				c:   tt.fields.c,
				key: tt.fields.key,
			}
			if got := tr.isNameServerUnavailable(tt.args.err); got != tt.want {
				t.Errorf("publishTask.isNameServerUnavailable() = %v, want %v", got, tt.want)
			}
		})
	}
}
