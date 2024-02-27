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

	"github.com/alibaba/kube-sharding/common"
	"github.com/alibaba/kube-sharding/controller/util"
	mock "github.com/alibaba/kube-sharding/controller/util/mock"
	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	"github.com/alibaba/kube-sharding/pkg/client/clientset/versioned/fake"
	carboninformers "github.com/alibaba/kube-sharding/pkg/client/informers/externalversions"
	listers "github.com/alibaba/kube-sharding/pkg/client/listers/carbon/v1"
	"github.com/golang/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/kubernetes/pkg/controller"
)

func newWorkerNode(name string, selector map[string]string) *carbonv1.WorkerNode {
	var service = carbonv1.WorkerNode{
		ObjectMeta: metav1.ObjectMeta{
			UID:         uuid.NewUUID(),
			Name:        name,
			Namespace:   metav1.NamespaceDefault,
			Annotations: make(map[string]string),
			Labels:      map[string]string{"foo": "bar", carbonv1.DefaultRollingsetUniqueLabelKey: "test1"},
		},
		Spec: carbonv1.WorkerNodeSpec{
			//Selector: selector,
		},
	}
	return &service
}

func TestHealthCheckTask_sync(t *testing.T) {
	type fields struct {
		healthCheckerInfo   *HealthCheckerInfo
		resourceManager     util.ResourceManager
		httpClient          *common.HTTPClient
		executor            *utils.AsyncExecutor
		rollingSet          *carbonv1.RollingSet
		healthCheckerConfig *carbonv1.HealthCheckerConfig
		healthCheckerRunner HealthChecker
	}
	type args struct {
		workernode      *carbonv1.WorkerNode
		healthCondition *carbonv1.HealthCondition
	}

	condition1 := carbonv1.HealthCondition{}
	condition1.Status = carbonv1.HealthAlive

	condition2 := carbonv1.HealthCondition{}
	condition2.Status = carbonv1.HealthLost
	condition2.LostCount = 2

	condition3 := carbonv1.HealthCondition{}
	condition3.Status = carbonv1.HealthLost
	condition3.LostCount = 3

	worknode := carbonv1.WorkerNode{}
	worknode.Status = carbonv1.WorkerNodeStatus{}
	worknode.Status.HealthCondition = condition1

	worknode2 := carbonv1.WorkerNode{}
	worknode2.Status = carbonv1.WorkerNodeStatus{}
	worknode2.Status.HealthCondition = condition2

	ctl := gomock.NewController(t)
	defer ctl.Finish()
	mockResourceManager := mock.NewMockResourceManager(ctl)
	mockResourceManager.EXPECT().UpdateWorkerStatus(gomock.Any()).Return(nil).MaxTimes(10)

	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name:   "test1",
			fields: fields{resourceManager: mockResourceManager},
			args:   args{workernode: &worknode, healthCondition: &condition1},
			want:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ht := &Task{
				healthCheckerInfo: tt.fields.healthCheckerInfo,
				resourceManager:   tt.fields.resourceManager,

				httpClient:          tt.fields.httpClient,
				executor:            tt.fields.executor,
				rollingSet:          tt.fields.rollingSet,
				healthCheckerConfig: tt.fields.healthCheckerConfig,
				healthCheckerRunner: tt.fields.healthCheckerRunner,
			}
			if got := ht.Sync(tt.args.workernode, tt.args.healthCondition); got != tt.want {
				t.Errorf("HealthCheckTask.Sync() = %v, want %v", got, tt.want)
			}
		})
	}
}

func NewConfigWithType(HealthType carbonv1.HealthConditionType) *carbonv1.HealthCheckerConfig {
	config := carbonv1.HealthCheckerConfig{}
	config.Type = HealthType
	config.Lv7Config = &carbonv1.Lv7HealthCheckerConfig{}
	config.Lv7Config.LostCountThreshold = 5
	config.Lv7Config.LostTimeout = 10 //10s

	config.Lv7Config.Path = "test"
	config.Lv7Config.Port = intstr.FromInt(8080)
	return &config
}

func TestHealthCheckTask_doUpdate(t *testing.T) {
	type fields struct {
		healthCheckerInfo   *HealthCheckerInfo
		workernodeLister    *listers.WorkerNodeLister
		httpClient          *common.HTTPClient
		executor            *utils.AsyncExecutor
		resourceManager     util.ResourceManager
		rollingSet          *carbonv1.RollingSet
		healthCheckerConfig *carbonv1.HealthCheckerConfig
		healthCheckerRunner HealthChecker
		taskCondition       *TaskCondition
	}
	type args struct {
		healthCheckerConfig *carbonv1.HealthCheckerConfig
		rollingSet          *carbonv1.RollingSet
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{
			name: "test1",
			fields: fields{
				taskCondition:     NewTaskCondition("test"),
				rollingSet:        newRollingSet("test"),
				healthCheckerInfo: &HealthCheckerInfo{},
			},
			args: args{rollingSet: newRollingSetWithHealthConfig(NewConfigWithType(carbonv1.Lv7Health))},
			want: "Lv7HealthCheck",
		},

		{
			name: "test2",
			fields: fields{
				taskCondition:     NewTaskCondition("test"),
				rollingSet:        newRollingSet("test"),
				healthCheckerInfo: &HealthCheckerInfo{},
			},
			args: args{rollingSet: newRollingSetWithHealthConfig(nil)},
			want: "DefaultHealthCheck",
		},

		{
			name: "test3",
			fields: fields{
				taskCondition:     NewTaskCondition("test"),
				rollingSet:        newRollingSet("test"),
				healthCheckerInfo: &HealthCheckerInfo{},
			},
			args: args{rollingSet: newRollingSetWithHealthConfig(NewConfigWithType(""))},
			want: "DefaultHealthCheck",
		},
		{
			name: "test3",
			fields: fields{
				taskCondition:     NewTaskCondition("test"),
				rollingSet:        newRollingSet("test"),
				healthCheckerInfo: &HealthCheckerInfo{},
			},
			args: args{rollingSet: newRollingSetWithHealthConfig(NewConfigWithType("dddd"))},
			want: "DefaultHealthCheck",
		},
		{
			name: "test4",
			fields: fields{
				taskCondition:     NewTaskCondition("test"),
				rollingSet:        newRollingSet("test"),
				healthCheckerInfo: &HealthCheckerInfo{},
			},
			args: args{rollingSet: newRollingSetWithHealthConfig(NewConfigWithType(carbonv1.AdvancedLv7Health))},
			want: "AdvancedLv7HealthChecker",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ht := &Task{
				healthCheckerInfo:   tt.fields.healthCheckerInfo,
				httpClient:          tt.fields.httpClient,
				executor:            tt.fields.executor,
				resourceManager:     tt.fields.resourceManager,
				rollingSet:          tt.fields.rollingSet,
				healthCheckerConfig: tt.fields.healthCheckerConfig,
				healthCheckerRunner: tt.fields.healthCheckerRunner,
				taskCondition:       tt.fields.taskCondition,
			}
			ht.doUpdate(tt.args.rollingSet)
			if ht.healthCheckerRunner.getTypeName() != tt.want {
				t.Errorf("TestHealthCheckTask_doUpdate.Sync() = %v, want %v", ht.healthCheckerRunner.getTypeName(), tt.want)
			}

		})
	}
}

func TestTask_processBatch(t *testing.T) {
	type fields struct {
		healthCheckerInfo   *HealthCheckerInfo
		httpClient          *common.HTTPClient
		executor            *utils.AsyncExecutor
		resourceManager     util.ResourceManager
		rollingSet          *carbonv1.RollingSet
		healthCheckerConfig *carbonv1.HealthCheckerConfig
		healthCheckerRunner HealthChecker
		taskCondition       *TaskCondition
	}
	executor := utils.NewExecutor(10)
	defer executor.Close()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	healthCheckerInfo := &HealthCheckerInfo{}
	client := fake.NewSimpleClientset([]runtime.Object{}...)
	informersFactory := carboninformers.NewSharedInformerFactory(client, controller.NoResyncPeriodFunc())
	workerLister := informersFactory.Carbon().V1().WorkerNodes().Lister()
	rs1 := newRollingSet("")

	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "empty nodes",
			fields: fields{executor: executor,
				healthCheckerRunner: &TestTimeOutChecker{},
				rollingSet:          rs1,
				healthCheckerInfo:   healthCheckerInfo,
				resourceManager: func() util.ResourceManager {
					mockResourceManager := mock.NewMockResourceManager(ctl)
					mockResourceManager.EXPECT().ListWorkerNodeForRS(map[string]string{"foo": "bar", carbonv1.DefaultRollingsetUniqueLabelKey: "test1"}).
						Return(nil, nil).Times(1)
					return mockResourceManager
				}(),
				taskCondition: NewTaskCondition("test"),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ht := &Task{
				healthCheckerInfo: tt.fields.healthCheckerInfo,
				httpClient:        tt.fields.httpClient,

				resourceManager:     tt.fields.resourceManager,
				rollingSet:          tt.fields.rollingSet,
				healthCheckerConfig: tt.fields.healthCheckerConfig,

				taskCondition:    tt.fields.taskCondition,
				workernodeLister: workerLister,
			}
			if err := ht.processBatch(); (err != nil) != tt.wantErr {
				t.Errorf("Task.processBatch() = %v", err)
			}
		})
	}
}

func TestTask_processBatch_timeout(t *testing.T) {
	type fields struct {
		healthCheckerInfo   *HealthCheckerInfo
		httpClient          *common.HTTPClient
		executor            *utils.AsyncExecutor
		resourceManager     util.ResourceManager
		rollingSet          *carbonv1.RollingSet
		healthCheckerConfig *carbonv1.HealthCheckerConfig
		healthCheckerRunner HealthChecker
		taskCondition       *TaskCondition
	}
	executor := utils.NewExecutor(10)
	defer executor.Close()

	ctl := gomock.NewController(t)
	defer ctl.Finish()
	mockResourceManager := mock.NewMockResourceManager(ctl)
	worker1 := newWorkerNode("worker1", nil)
	worker2 := newWorkerNode("worker2", nil)
	workers := make([]*carbonv1.WorkerNode, 0)
	workers = append(workers, worker1)
	workers = append(workers, worker2)
	mockResourceManager.EXPECT().ListWorkerNodeForRS(gomock.Any()).Return(workers, nil).MaxTimes(10)

	client := fake.NewSimpleClientset([]runtime.Object{}...)
	informersFactory := carboninformers.NewSharedInformerFactory(client, controller.NoResyncPeriodFunc())
	informersFactory.Carbon().V1().WorkerNodes().Informer().GetIndexer().Add(worker1)
	informersFactory.Carbon().V1().WorkerNodes().Informer().GetIndexer().Add(worker2)
	workerLister := informersFactory.Carbon().V1().WorkerNodes().Lister()

	healthCheckerInfo := &HealthCheckerInfo{}

	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "test timeout",
			fields: fields{executor: executor,
				healthCheckerRunner: &TestTimeOutChecker{},
				rollingSet:          newRollingSet(""),
				healthCheckerInfo:   healthCheckerInfo,
				resourceManager:     mockResourceManager,
				taskCondition:       NewTaskCondition("test"),
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ht := &Task{
				healthCheckerInfo:   tt.fields.healthCheckerInfo,
				httpClient:          tt.fields.httpClient,
				executor:            tt.fields.executor,
				resourceManager:     tt.fields.resourceManager,
				rollingSet:          tt.fields.rollingSet,
				healthCheckerConfig: tt.fields.healthCheckerConfig,
				healthCheckerRunner: tt.fields.healthCheckerRunner,
				taskCondition:       tt.fields.taskCondition,
				workernodeLister:    workerLister,
			}
			go ht.processBatch()
			time.Sleep(10 * time.Millisecond)

			if executor.ActiveCount() != 2 {
				t.Errorf("Task.processBatch() activeCount:%v", executor.ActiveCount())
			}
			time.Sleep(1 * time.Second)
			if executor.ActiveCount() != 2 {
				t.Errorf("Task.processBatch() activeCount:%v", executor.ActiveCount())
			}
			time.Sleep(5 * time.Second)
			if executor.ActiveCount() != 0 {
				t.Errorf("Task.processBatch() activeCount:%v", executor.ActiveCount())
			}
		})
	}
}

type TestTimeOutChecker struct {
}

func (t *TestTimeOutChecker) doCheck(workernode *carbonv1.WorkerNode, config *carbonv1.HealthCheckerConfig, rollingSet *carbonv1.RollingSet) *carbonv1.HealthCondition {
	time.Sleep(5 * time.Second)
	return nil
}

func (t *TestTimeOutChecker) getTypeName() string {
	return "TestHealthCheck"
}

func TestTask_decompressVersionPlan(t *testing.T) {
	type fields struct {
		healthCheckerInfo   *HealthCheckerInfo
		httpClient          *common.HTTPClient
		executor            *utils.AsyncExecutor
		resourceManager     util.ResourceManager
		rollingSet          *carbonv1.RollingSet
		healthCheckerConfig *carbonv1.HealthCheckerConfig
		workernodeLister    listers.WorkerNodeLister
		healthCheckerRunner HealthChecker
		taskCondition       *TaskCondition
	}
	type args struct {
		rollingSet *carbonv1.RollingSet
	}
	tests := []struct {
		name                     string
		fields                   fields
		args                     args
		wantCustominfo           string
		wantCompressedCustominfo string
	}{
		{
			name:                     "test1",
			args:                     args{rollingSet: newRollingSetWithFinal("", "")},
			wantCustominfo:           "",
			wantCompressedCustominfo: "",
		},
		{
			name:                     "test2",
			args:                     args{rollingSet: newRollingSetWithFinal("Custominfo", "")},
			wantCustominfo:           "Custominfo",
			wantCompressedCustominfo: "",
		},
		{
			name:                     "test3",
			args:                     args{rollingSet: newRollingSetWithFinal("", "H4sIAAAAAAAA/3IuLS7Jz/XMS8vXNQIAAAD//w==")},
			wantCustominfo:           "CustomInfo-2",
			wantCompressedCustominfo: "",
		},
		{
			name:                     "test4",
			args:                     args{rollingSet: newRollingSetWithFinal("Custominfo", "H4sIAAAAAAAA/3IuLS7Jz/XMS8vXNQIAAAD//w==")},
			wantCustominfo:           "Custominfo",
			wantCompressedCustominfo: "",
		},

		{
			name:                     "test5",
			args:                     args{rollingSet: newRollingSetWithFinal("", "invaild value")},
			wantCustominfo:           "",
			wantCompressedCustominfo: "invaild value",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ht := &Task{
				healthCheckerInfo:   tt.fields.healthCheckerInfo,
				httpClient:          tt.fields.httpClient,
				executor:            tt.fields.executor,
				resourceManager:     tt.fields.resourceManager,
				rollingSet:          tt.fields.rollingSet,
				healthCheckerConfig: tt.fields.healthCheckerConfig,
				workernodeLister:    tt.fields.workernodeLister,
				healthCheckerRunner: tt.fields.healthCheckerRunner,
				taskCondition:       tt.fields.taskCondition,
			}
			ht.decompressVersionPlan(tt.args.rollingSet)
			if tt.args.rollingSet.Spec.VersionPlan.CustomInfo != tt.wantCustominfo {
				t.Errorf("decompressVersionPlan CustomInfo %v want:%v", tt.args.rollingSet.Spec.VersionPlan.CustomInfo, tt.wantCustominfo)
			}
			if tt.args.rollingSet.Spec.VersionPlan.CompressedCustomInfo != tt.wantCompressedCustominfo {
				t.Errorf("decompressVersionPlan CompressedCustomInfo %v want:%v", tt.args.rollingSet.Spec.VersionPlan.CompressedCustomInfo, tt.wantCompressedCustominfo)

			}
		})
	}
}
