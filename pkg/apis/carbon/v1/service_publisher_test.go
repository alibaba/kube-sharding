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
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSyncWorkerServiceHealthStatus(t *testing.T) {
	type args struct {
		worker *WorkerNode
	}
	tests := []struct {
		name string
		args args
		want ServiceStatus
	}{
		{
			name: "TestUnAvailable",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{ServiceConditions: []ServiceCondition{
				{
					ServiceName: "1",
					Status:      v1.ConditionFalse,
				},
				{
					ServiceName: "2",
					Status:      v1.ConditionFalse,
				},
			}}}},
			want: ServiceUnAvailable,
		},
		{
			name: "ServicePartAvailable",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{ServiceConditions: []ServiceCondition{
				{
					ServiceName: "1",
					Status:      v1.ConditionFalse,
				},
				{
					ServiceName: "2",
					Status:      v1.ConditionTrue,
				},
			}}}},
			want: ServicePartAvailable,
		},
		{
			name: "TestUnAvailable",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{ServiceConditions: []ServiceCondition{
				{
					ServiceName: "1",
					Status:      v1.ConditionFalse,
				},
				{
					ServiceName: "2",
					Status:      v1.ConditionUnknown,
				},
			}}}},
			want: ServiceUnAvailable,
		},
		{
			name: "ServiceAvailable",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{ServiceConditions: []ServiceCondition{
				{
					ServiceName: "1",
					Status:      v1.ConditionTrue,
				},
				{
					ServiceName: "2",
					Status:      v1.ConditionTrue,
				},
			}}}},
			want: ServiceAvailable,
		},
		{
			name: "ServiceAvailable",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{}}},
			want: ServiceAvailable,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SyncWorkerServiceHealthStatus(tt.args.worker); got != tt.want {
				t.Errorf("SyncWorkerServiceHealthStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetWorkerServiceHealthCondition(t *testing.T) {
	type args struct {
		worker        *WorkerNode
		conditionType RegistryType
		name          string
		status        v1.ConditionStatus
		score         int64
		reason        string
		message       string
	}
	worker := &WorkerNode{Status: WorkerNodeStatus{ServiceStatus: ServiceUnAvailable}}
	worker.Status.Version = "v1"
	tests := []struct {
		name           string
		args           args
		want           ServiceStatus
		wantConditions []ServiceCondition
	}{
		{
			name: "TestAvaliable",
			args: args{
				worker:        worker,
				conditionType: "",
				name:          "test",
				status:        v1.ConditionTrue,
				score:         0,
			},
			want: ServiceAvailable,
			wantConditions: []ServiceCondition{
				{
					Name:    "test",
					Status:  "True",
					Message: "Node Published",
					Score:   0,
					Version: "v1",
				},
			},
		},
		{
			name: "TestUnAvaliable",
			args: args{
				worker:        worker,
				conditionType: "",
				name:          "test",
				status:        v1.ConditionFalse,
				score:         1,
			},
			want: ServiceUnAvailable,
			wantConditions: []ServiceCondition{
				{
					Name:    "test",
					Status:  "False",
					Message: "Node Published",
					Score:   1,
					Version: "v1",
				},
			},
		},
		{
			name: "TestUnAvaliable1",
			args: args{
				worker:        worker,
				conditionType: "",
				name:          "test",
				reason:        "Service Check False",
				status:        v1.ConditionFalse,
				score:         1,
			},
			want: ServiceUnAvailable,
			wantConditions: []ServiceCondition{
				{
					Name:    "test",
					Status:  "False",
					Message: "Node Published",
					Score:   1,
					Reason:  "Service Check False",
					Version: "v1",
				},
			},
		},
		{
			name: "TestUnAvaliable2",
			args: args{
				worker:        worker,
				conditionType: "",
				name:          "test",
				reason:        "Query NameService With Error",
				message:       "return error",
				status:        v1.ConditionFalse,
				score:         1,
			},
			want: ServiceUnAvailable,
			wantConditions: []ServiceCondition{
				{
					Name:    "test",
					Status:  "False",
					Message: "Node Published and query with error :  return error",
					Score:   1,
					Reason:  "Query NameService With Error",
					Version: "v1",
				},
			},
		},
		{
			name: "TestPartAvaliable",
			args: args{
				worker:        worker,
				conditionType: "",
				name:          "test1",
				status:        v1.ConditionTrue,
				score:         -1,
			},
			want: ServicePartAvailable,
			wantConditions: []ServiceCondition{
				{
					Name:    "test",
					Status:  "False",
					Message: "Node Published and query with error :  return error",
					Score:   1,
					Reason:  "Query NameService With Error",
					Version: "v1",
				},
				{
					Name:    "test1",
					Status:  "True",
					Message: "Node Published",
					Score:   -1,
					Version: "v1",
				},
			},
		},
		{
			name: "TestAvaliable1",
			args: args{
				worker:        worker,
				conditionType: "",
				name:          "test",
				status:        v1.ConditionTrue,
				score:         0,
			},
			want: ServiceAvailable,
			wantConditions: []ServiceCondition{
				{
					Name:    "test",
					Status:  "True",
					Message: "Node Published",
					Score:   0,
					Reason:  "",
					Version: "v1",
				},
				{
					Name:    "test1",
					Status:  "True",
					Message: "Node Published",
					Score:   -1,
					Version: "v1",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := ServicePublisher{
				ObjectMeta: metav1.ObjectMeta{Name: tt.args.name},
				Spec: ServicePublisherSpec{
					ServiceName: "",
					Type:        tt.args.conditionType,
				},
			}
			got := SetWorkerServiceHealthCondition(tt.args.worker, &service, "", tt.args.status, tt.args.score, tt.args.reason, tt.args.message, false, false)
			if got != tt.want {
				t.Errorf("SetWorkerServiceHealthCondition() = %v, want %v", got, tt.want)
			}
			for i := range tt.args.worker.Status.ServiceConditions {
				tt.args.worker.Status.ServiceConditions[i].LastTransitionTime = metav1.Time{}
			}
			if !reflect.DeepEqual(tt.args.worker.Status.ServiceConditions, tt.wantConditions) {
				t.Errorf("SetWorkerServiceHealthCondition() = %s, want %s", utils.ObjJSON(tt.args.worker.Status.ServiceConditions), utils.ObjJSON(tt.wantConditions))
			}
		})
	}
}

func TestDeleteWorkerServiceHealthCondition(t *testing.T) {
	type args struct {
		worker *WorkerNode
		name   string
		err    error
	}
	worker := &WorkerNode{Status: WorkerNodeStatus{ServiceStatus: ServiceUnAvailable}}
	service0 := &ServicePublisher{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
	}
	service1 := &ServicePublisher{
		ObjectMeta: metav1.ObjectMeta{Name: "test1"},
	}
	SetWorkerServiceHealthCondition(worker, service0, "", v1.ConditionTrue, 0, "", "", false, false)
	fmt.Println(worker.Status.ServiceConditions)
	SetWorkerServiceHealthCondition(worker, service1, "", v1.ConditionFalse, 1, "", "", false, false)
	tests := []struct {
		name           string
		args           args
		want           ServiceStatus
		wantConditions int
	}{
		{
			name: "TestUnAvaliable",
			args: args{
				worker: worker,
				name:   "test",
			},
			want:           ServiceUnAvailable,
			wantConditions: 1,
		},
		{
			name: "TestAvaliable",
			args: args{
				worker: worker,
				name:   "test1",
			},
			want:           ServiceAvailable,
			wantConditions: 0,
		},
		{
			name: "TestAvaliableWithError",
			args: args{
				worker: func() *WorkerNode {
					worker := &WorkerNode{Status: WorkerNodeStatus{ServiceStatus: ServiceUnAvailable}}
					SetWorkerServiceHealthCondition(worker, service0, "", v1.ConditionTrue, 0, "", "", false, false)
					fmt.Println(worker.Status.ServiceConditions)
					SetWorkerServiceHealthCondition(worker, service1, "", v1.ConditionFalse, 1, "Query NameService With Error", "return err", false, false)
					return worker
				}(),
				name: "test",
				err:  errors.New("return err"),
			},
			want:           ServicePartAvailable,
			wantConditions: 2,
		},
		{
			name: "TestAvaliableWithError",
			args: args{
				worker: func() *WorkerNode {
					worker := &WorkerNode{Status: WorkerNodeStatus{ServiceStatus: ServiceUnAvailable}}
					SetWorkerServiceHealthCondition(worker, service0, "", v1.ConditionTrue, 0, "", "", false, false)
					SetWorkerServiceHealthCondition(worker, service0, "", v1.ConditionFalse, 0, "Service Check False", "", false, false)
					SetWorkerServiceHealthCondition(worker, service0, "", v1.ConditionFalse, 0, "Query NameService With Error", "return err", false, false)
					fmt.Println(worker.Status.ServiceConditions)
					SetWorkerServiceHealthCondition(worker, service1, "", v1.ConditionFalse, 1, "Query NameService With Error", "return err", false, false)
					return worker
				}(),
				name: "test",
				err:  errors.New("return err"),
			},
			want:           ServiceUnAvailable,
			wantConditions: 2,
		},
		{
			name: "TestAvaliableWithOutError",
			args: args{
				worker: func() *WorkerNode {
					worker := &WorkerNode{Status: WorkerNodeStatus{ServiceStatus: ServiceUnAvailable}}
					SetWorkerServiceHealthCondition(worker, service0, "", v1.ConditionTrue, 0, "", "", false, false)
					SetWorkerServiceHealthCondition(worker, service0, "", v1.ConditionFalse, 0, "Service Check False", "", false, false)
					SetWorkerServiceHealthCondition(worker, service0, "", v1.ConditionFalse, 0, "Query NameService With Error", "return err", false, false)
					fmt.Println(worker.Status.ServiceConditions)
					SetWorkerServiceHealthCondition(worker, service1, "", v1.ConditionFalse, 1, "Query NameService With Error", "return err", false, false)
					return worker
				}(),
				name: "test",
				err:  nil,
			},
			want:           ServiceUnAvailable,
			wantConditions: 1,
		},
		{
			name: "TestAvaliableWithError",
			args: args{
				worker: func() *WorkerNode {
					worker := &WorkerNode{Status: WorkerNodeStatus{ServiceStatus: ServiceUnAvailable}}
					SetWorkerServiceHealthCondition(worker, service0, "", v1.ConditionTrue, 0, "", "", false, false)
					fmt.Println(worker.Status.ServiceConditions)
					SetWorkerServiceHealthCondition(worker, service1, "", v1.ConditionFalse, 1, "Query NameService With Error", "return err", false, false)
					return worker
				}(),
				name: "test1",
				err:  errors.New("return err"),
			},
			want:           ServiceAvailable,
			wantConditions: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DeleteWorkerServiceHealthCondition(tt.args.worker, tt.args.name, tt.args.err)
			if got != tt.want {
				t.Errorf("SetWorkerServiceHealthCondition() = %v, want %v", got, tt.want)
			}
			if len(tt.args.worker.Status.ServiceConditions) != tt.wantConditions {
				t.Errorf("SetWorkerServiceHealthCondition() = %v, want %v", len(tt.args.worker.Status.ServiceConditions), tt.wantConditions)
			}
		})
	}
}

func TestGetWorkerServiceHealthCondition(t *testing.T) {
	type args struct {
		worker *WorkerNode
	}
	tests := []struct {
		name string
		args args
		want ServiceStatus
	}{
		{
			name: "Test nil worker",
			args: args{worker: nil},
			want: ServiceUnKnown,
		},
		{
			name: "TestUnAvailable",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{ServiceConditions: []ServiceCondition{
				{
					ServiceName: "1",
					Status:      v1.ConditionFalse,
				},
				{
					ServiceName: "2",
					Status:      v1.ConditionFalse,
				},
			}}}},
			want: ServiceUnAvailable,
		},
		{
			name: "ServicePartAvailable",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{ServiceConditions: []ServiceCondition{
				{
					ServiceName: "1",
					Status:      v1.ConditionFalse,
				},
				{
					ServiceName: "2",
					Status:      v1.ConditionTrue,
				},
			}}}},
			want: ServicePartAvailable,
		},
		{
			name: "TestUnAvailable",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{ServiceConditions: []ServiceCondition{
				{
					ServiceName: "1",
					Status:      v1.ConditionFalse,
				},
				{
					ServiceName: "2",
					Status:      v1.ConditionUnknown,
				},
			}}}},
			want: ServiceUnAvailable,
		},
		{
			name: "TestUnAvailable case2 offline and empty ServiceCondition",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{
				ServiceConditions: []ServiceCondition{},
				ServiceOffline:    true,
			}}},
			want: ServiceUnAvailable,
		},
		{
			name: "TestUnAvailable case2 offline and empty ServiceCondition",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{
				ServiceConditions: []ServiceCondition{
					{
						Type:        ServiceSkyline,
						ServiceName: "1",
						Status:      v1.ConditionFalse,
					},
				},
				ServiceOffline: true,
			}}},
			want: ServiceUnAvailable,
		},
		{
			name: "TestUnAvailable case2 offline and empty ServiceCondition",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{
				ServiceConditions: []ServiceCondition{
					{
						Type:        ServiceSkyline,
						ServiceName: "1",
						Status:      v1.ConditionFalse,
					},
					{
						Type:        ServiceSkyline,
						ServiceName: "1",
						Status:      v1.ConditionTrue,
					},
					{
						ServiceName: "1",
						Status:      v1.ConditionFalse,
					},
				},
				ServiceOffline: true,
			}}},
			want: ServiceUnAvailable,
		},
		{
			name: "TestUnAvailable case2 offline and empty ServiceCondition",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{
				ServiceConditions: []ServiceCondition{
					{
						Type:        ServiceSkyline,
						ServiceName: "1",
						Status:      v1.ConditionFalse,
					},
					{
						Type:        ServiceSkyline,
						ServiceName: "1",
						Status:      v1.ConditionTrue,
					},
					{
						ServiceName: "1",
						Status:      v1.ConditionTrue,
					},
				},
				ServiceOffline: true,
			}}},
			want: ServicePartAvailable,
		},
		{
			name: "ServiceAvailable",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{ServiceConditions: []ServiceCondition{
				{
					ServiceName: "1",
					Status:      v1.ConditionTrue,
				},
				{
					ServiceName: "2",
					Status:      v1.ConditionTrue,
				},
			}}}},
			want: ServiceAvailable,
		},
		{
			name: "ServiceAvailable",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{}}},
			want: ServiceAvailable,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetServiceStatusFromCondition(tt.args.worker); got != tt.want {
				t.Errorf("GetServiceStatusFromCondition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNeedUpdateGracefully(t *testing.T) {
	type args struct {
		service *ServicePublisher
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "nil",
			args: args{
				service: nil,
			},
			want: true,
		},
		{
			name: "skyline",
			args: args{
				service: &ServicePublisher{
					Spec: ServicePublisherSpec{
						Type: ServiceSkyline,
					},
				},
			},
			want: false,
		},
		{
			name: "vip",
			args: args{
				service: &ServicePublisher{
					Spec: ServicePublisherSpec{
						Type: ServiceVipserver,
					},
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NeedUpdateGracefully(tt.args.service); got != tt.want {
				t.Errorf("NeedUpdateGracefully() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_isGracefullyServiceType(t *testing.T) {
	type args struct {
		registryType RegistryType
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "skyline",
			args: args{
				registryType: ServiceSkyline,
			},
			want: false,
		},
		{
			name: "vip",
			args: args{
				registryType: ServiceVipserver,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isGracefullyServiceType(tt.args.registryType); got != tt.want {
				t.Errorf("isGracefullyServiceType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsWorkerInWarmup(t *testing.T) {
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
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{
				ServiceConditions: []ServiceCondition{
					{
						Type:        ServiceSkyline,
						ServiceName: "1",
						Status:      v1.ConditionFalse,
					},
					{
						Type:        ServiceSkyline,
						ServiceName: "1",
						Status:      v1.ConditionTrue,
					},
				},
			}}},
			want: false,
		},
		{
			name: "part warmup",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{
				ServiceConditions: []ServiceCondition{
					{
						Type:        ServiceSkyline,
						ServiceName: "1",
						Status:      v1.ConditionFalse,
					},
					{
						Type:        ServiceSkyline,
						ServiceName: "1",
						Status:      v1.ConditionTrue,
					},
					{
						Type:        ServiceCM2,
						ServiceName: "1",
						InWarmup:    true,
					},
					{
						Type:        ServiceCM2,
						ServiceName: "1",
						InWarmup:    false,
					},
				},
			}}},
			want: false,
		},
		{
			name: "part warmup",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{
				ServiceConditions: []ServiceCondition{
					{
						Type:        ServiceCM2,
						ServiceName: "1",
						InWarmup:    true,
					},
					{
						Type:        ServiceCM2,
						ServiceName: "1",
						InWarmup:    false,
					},
				},
			}}},
			want: false,
		},
		{
			name: "warmup",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{
				ServiceConditions: []ServiceCondition{
					{
						Type:        ServiceSkyline,
						ServiceName: "1",
						Status:      v1.ConditionFalse,
					},
					{
						Type:        ServiceSkyline,
						ServiceName: "1",
						Status:      v1.ConditionTrue,
					},
					{
						Type:        ServiceCM2,
						ServiceName: "1",
						InWarmup:    true,
					},
					{
						Type:        ServiceCM2,
						ServiceName: "1",
						InWarmup:    true,
					},
				},
			}}},
			want: true,
		},
		{
			name: "part warmup",
			args: args{worker: &WorkerNode{Status: WorkerNodeStatus{
				ServiceConditions: []ServiceCondition{
					{
						Type:        ServiceCM2,
						ServiceName: "1",
						InWarmup:    true,
					},
					{
						Type:        ServiceCM2,
						ServiceName: "1",
						InWarmup:    true,
					},
				},
			}}},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsWorkerInWarmup(tt.args.worker); got != tt.want {
				t.Errorf("IsWorkerInWarmup() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsWorkerOfflined(t *testing.T) {
	type args struct {
		worker *WorkerNode
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "not offlined",
			args: args{
				worker: &WorkerNode{
					Status: WorkerNodeStatus{
						ServiceStatus: ServiceUnAvailable,
						ServiceConditions: []ServiceCondition{
							{
								Type: ServiceVipserver,
							},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "not offlined",
			args: args{
				worker: &WorkerNode{
					Status: WorkerNodeStatus{
						ServiceStatus: ServiceUnAvailable,
						ServiceConditions: []ServiceCondition{
							{
								Type: ServiceVipserver,
							},
							{
								Type: ServiceSkyline,
							},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "not offlined",
			args: args{
				worker: &WorkerNode{
					Status: WorkerNodeStatus{
						ServiceStatus:     ServiceAvailable,
						ServiceConditions: []ServiceCondition{},
					},
				},
			},
			want: false,
		},
		{
			name: "offlined",
			args: args{
				worker: &WorkerNode{
					Status: WorkerNodeStatus{
						ServiceStatus: ServiceUnAvailable,
						ServiceConditions: []ServiceCondition{
							{
								Type: ServiceSkyline,
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "offlined",
			args: args{
				worker: &WorkerNode{
					Status: WorkerNodeStatus{
						ServiceStatus:     ServiceUnAvailable,
						ServiceConditions: []ServiceCondition{},
					},
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsWorkerOfflined(tt.args.worker); got != tt.want {
				t.Errorf("IsWorkerOfflined() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsWorkerUnpublished(t *testing.T) {
	type args struct {
		worker *WorkerNode
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "not unpublished",
			args: args{
				worker: &WorkerNode{
					Status: WorkerNodeStatus{
						ServiceStatus: ServiceUnAvailable,
						ServiceConditions: []ServiceCondition{
							{
								Type:   ServiceVipserver,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "not unpublished",
			args: args{
				worker: &WorkerNode{
					Status: WorkerNodeStatus{
						ServiceStatus: ServiceUnAvailable,
						ServiceConditions: []ServiceCondition{
							{
								Type:   ServiceVipserver,
								Status: corev1.ConditionFalse,
								Reason: ReasonServiceCheckFalse,
							},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "unpublished",
			args: args{
				worker: &WorkerNode{
					Status: WorkerNodeStatus{
						ServiceStatus: ServiceUnAvailable,
						ServiceConditions: []ServiceCondition{
							{
								Type:   ServiceVipserver,
								Status: corev1.ConditionFalse,
								Reason: ReasonNotExist,
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "unpublished",
			args: args{
				worker: &WorkerNode{
					Status: WorkerNodeStatus{
						ServiceStatus: ServiceUnAvailable,
						ServiceConditions: []ServiceCondition{
							{
								Type:   ServiceVipserver,
								Status: corev1.ConditionFalse,
								Reason: InitServiceFalseState,
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "unpublished",
			args: args{
				worker: &WorkerNode{
					Status: WorkerNodeStatus{
						ServiceStatus: ServiceUnAvailable,
						ServiceConditions: []ServiceCondition{
							{
								Type:   ServiceSkyline,
								Status: corev1.ConditionTrue,
							},
							{
								Type:   ServiceVipserver,
								Status: corev1.ConditionFalse,
								Reason: ReasonNotExist,
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "not unpublished",
			args: args{
				worker: &WorkerNode{
					Status: WorkerNodeStatus{
						ServiceStatus: ServiceUnAvailable,
						ServiceConditions: []ServiceCondition{
							{
								Type:   ServiceSkyline,
								Status: corev1.ConditionTrue,
							},
							{
								Type:   ServiceVipserver,
								Status: corev1.ConditionFalse,
								Reason: ReasonServiceCheckFalse,
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
			if got := IsWorkerUnpublished(tt.args.worker); got != tt.want {
				t.Errorf("IsWorkerUnpublished() = %v, want %v", got, tt.want)
			}
		})
	}
}
