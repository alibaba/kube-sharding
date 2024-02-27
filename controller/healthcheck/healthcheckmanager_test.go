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
	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/uuid"
)

func newRollingSet(path string) *carbonv1.RollingSet {
	selector := map[string]string{"foo": "bar", carbonv1.DefaultRollingsetUniqueLabelKey: "test1"}
	rollingset := carbonv1.RollingSet{
		TypeMeta: metav1.TypeMeta{APIVersion: "carbon/v1", Kind: "Rollingset"},
		ObjectMeta: metav1.ObjectMeta{
			UID:         uuid.NewUUID(),
			Name:        "rollingset-m",
			Namespace:   metav1.NamespaceDefault,
			Annotations: make(map[string]string),
			Labels: map[string]string{
				carbonv1.LabelKeyClusterName: "1",
				carbonv1.LabelKeyAppName:     "2",
			},
		},
		Spec: carbonv1.RollingSetSpec{
			Version:             "version",
			Selector:            &metav1.LabelSelector{MatchLabels: selector},
			HealthCheckerConfig: NewConfigWithPath(path),
		},
		Status: carbonv1.RollingSetStatus{},
	}
	return &rollingset
}

func newRollingSetWithHealthConfig(config *carbonv1.HealthCheckerConfig) *carbonv1.RollingSet {
	selector := map[string]string{"foo": "bar", carbonv1.DefaultRollingsetUniqueLabelKey: "test1"}
	rollingset := carbonv1.RollingSet{
		TypeMeta: metav1.TypeMeta{APIVersion: "carbon/v1", Kind: "Rollingset"},
		ObjectMeta: metav1.ObjectMeta{
			UID:         uuid.NewUUID(),
			Name:        "rollingset-m",
			Namespace:   metav1.NamespaceDefault,
			Annotations: make(map[string]string),
			Labels:      map[string]string{carbonv1.LabelKeyClusterName: "1", carbonv1.LabelKeyAppName: "2"},
		},
		Spec: carbonv1.RollingSetSpec{
			Version: "version",
			VersionPlan: carbonv1.VersionPlan{
				BroadcastPlan: carbonv1.BroadcastPlan{
					WorkerSchedulePlan: carbonv1.WorkerSchedulePlan{
						MinReadySeconds:      1024,
						ResourceMatchTimeout: 2000,
						WorkerReadyTimeout:   2014,
					},
				},
			},

			Selector:            &metav1.LabelSelector{MatchLabels: selector},
			HealthCheckerConfig: config,
		},
		Status: carbonv1.RollingSetStatus{},
	}
	return &rollingset
}

func NewConfigWithPath(path string) *carbonv1.HealthCheckerConfig {
	config := carbonv1.HealthCheckerConfig{}
	config.Lv7Config = &carbonv1.Lv7HealthCheckerConfig{}
	config.Lv7Config.LostCountThreshold = 5
	config.Lv7Config.LostTimeout = 10 //10s

	config.Lv7Config.Path = path
	config.Lv7Config.Port = intstr.FromInt(8080)
	return &config
}

func TestHealthCheckManager_Close(t *testing.T) {
	type fields struct {
		resourceManager    util.ResourceManager
		healthCheckTaskMap map[string]*Task
		httpClient         *common.HTTPClient
		executor           *utils.AsyncExecutor
	}
	healthCheckTaskMap := make(map[string]*Task, 0)
	before := time.Unix(time.Now().Unix()-100, 0)
	healthCheckerConfig := NewConfigWithPath("test")
	//新增，新启线程

	info := HealthCheckerInfo{latestUpdateTime: before, createdTime: before, healthChecker: healthCheckerConfig}
	key := "testkey1"
	task := NewHealthCheckTask(&info, &carbonv1.RollingSet{}, nil, nil, nil, nil)
	healthCheckTaskMap[key] = task
	task.Start()
	task.MarkStop()
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name:    "test1",
			fields:  fields{healthCheckTaskMap: healthCheckTaskMap},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hm := &Manager{
				resourceManager:    tt.fields.resourceManager,
				healthCheckTaskMap: tt.fields.healthCheckTaskMap,
				httpClient:         tt.fields.httpClient,
				executor:           tt.fields.executor,
			}
			if err := hm.Close(); (err != nil) != tt.wantErr {
				t.Errorf("HealthCheckManager.Close() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHealthCheckManager_removeTask(t *testing.T) {
	type fields struct {
		healthCheckTaskMap map[string]*Task
		httpClient         *common.HTTPClient
		executor           *utils.AsyncExecutor
	}
	type args struct {
		key string
	}

	healthCheckTaskMap := make(map[string]*Task, 0)
	before := time.Unix(time.Now().Unix()-100, 0)
	healthCheckerConfig := NewConfigWithPath("test")
	//新增，新启线程
	info := HealthCheckerInfo{latestUpdateTime: before, createdTime: before, healthChecker: healthCheckerConfig}
	key := "testkey1"
	task := NewHealthCheckTask(&info, &carbonv1.RollingSet{}, nil, nil, nil, nil)
	healthCheckTaskMap[key] = task
	task.Start()
	task.MarkStop()
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "test1",
			fields:  fields{healthCheckTaskMap: healthCheckTaskMap},
			args:    args{key: "testkey1"},
			wantErr: false,
		},
		{
			name:    "test2",
			fields:  fields{healthCheckTaskMap: healthCheckTaskMap},
			args:    args{key: "testkey2"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hm := &Manager{

				healthCheckTaskMap: tt.fields.healthCheckTaskMap,
				httpClient:         tt.fields.httpClient,
				executor:           tt.fields.executor,
			}
			if err := hm.removeTask(tt.args.key); (err != nil) != tt.wantErr {
				t.Errorf("HealthCheckManager.removeTask() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
