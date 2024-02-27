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

package controller

import (
	"fmt"
	"math"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/alibaba/kube-sharding/controller/util"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	clientset "github.com/alibaba/kube-sharding/pkg/client/clientset/versioned"
	"github.com/alibaba/kube-sharding/pkg/rollalgorithm"
	apps "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
)

func TestDefaultController_GetLatestUpdateTime(t *testing.T) {
	type fields struct {
		Controller          Controller
		controllerAgentName string
		GroupVersionKind    schema.GroupVersionKind
		workqueue           workqueue.RateLimitingInterface
		kubeclientset       kubernetes.Interface
		carbonclientset     clientset.Interface
		ResourceManager     util.ResourceManager
		start               chan struct{}
		caches              map[string]map[string]string
	}
	type args struct {
		obj interface{}
	}
	rs := newRollingSet(nil, time.Time{})
	rs2 := newRollingSet(map[string]string{"foo": "bar"}, time.Time{})
	rs3 := newRollingSet(map[string]string{"updateCrdTime": "nil"}, time.Time{})
	now1 := time.Now()
	time.Sleep(2 * time.Second)
	now2 := time.Now()
	rs4 := newRollingSet(map[string]string{"updateCrdTime": "nil"}, now1)

	rs5 := newRollingSet(map[string]string{"updateCrdTime": "nil"}, now2)
	timeString1 := now1.Format(time.RFC3339Nano)
	timeString2 := now2.Format(time.RFC3339Nano)
	time1, _ := time.Parse(time.RFC3339Nano, timeString1)
	time2, _ := time.Parse(time.RFC3339Nano, timeString2)

	rs6 := newRollingSet(map[string]string{"updateCrdTime": timeString1}, time.Time{})
	rs7 := newRollingSet(map[string]string{"updateCrdTime": timeString2}, time.Time{})

	rs8 := newRollingSet(map[string]string{"updateCrdTime": timeString1}, time1)
	rs9 := newRollingSet(map[string]string{"updateCrdTime": timeString1}, time2)
	rs10 := newRollingSet(map[string]string{"updateCrdTime": timeString2}, time1)

	tests := []struct {
		name   string
		fields fields
		args   args
		want   time.Time
	}{
		{
			name:   "test1",
			fields: fields{},
			args:   args{obj: rs},
			want:   time.Time{},
		},
		{
			name:   "test2",
			fields: fields{},
			args:   args{obj: rs2},
			want:   time.Time{},
		},
		{
			name:   "test3",
			fields: fields{},
			args:   args{obj: rs3},
			want:   time.Time{},
		},
		{
			name:   "test4",
			fields: fields{},
			args:   args{obj: rs4},
			want:   now1,
		},
		{
			name:   "test5",
			fields: fields{},
			args:   args{obj: rs5},
			want:   now2,
		},
		{
			name:   "test6",
			fields: fields{},
			args:   args{obj: rs6},
			want:   time1,
		},

		{
			name:   "test7",
			fields: fields{},
			args:   args{obj: rs7},
			want:   time2,
		},

		{
			name:   "test8",
			fields: fields{},
			args:   args{obj: rs8},
			want:   time1,
		},

		{
			name:   "test9",
			fields: fields{},
			args:   args{obj: rs9},
			want:   time2,
		},

		{
			name:   "test10",
			fields: fields{},
			args:   args{obj: rs10},
			want:   time2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &DefaultController{
				Controller:          tt.fields.Controller,
				controllerAgentName: tt.fields.controllerAgentName,
				GroupVersionKind:    tt.fields.GroupVersionKind,
				workqueue:           tt.fields.workqueue,
				kubeclientset:       tt.fields.kubeclientset,
				carbonclientset:     tt.fields.carbonclientset,
				ResourceManager:     tt.fields.ResourceManager,
				start:               tt.fields.start,
				caches:              tt.fields.caches,
			}
			if got := c.GetLatestUpdateTime(tt.args.obj); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DefaultController.GetLatestUpdateTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func newRollingSet(annotations map[string]string, updateStatusTime time.Time) *carbonv1.RollingSet {
	selector := map[string]string{"foo": "bar"}

	rollingset := carbonv1.RollingSet{
		TypeMeta: metav1.TypeMeta{APIVersion: "carbon/v1", Kind: "Rollingset"},
		ObjectMeta: metav1.ObjectMeta{
			UID:         "acdd",
			Name:        "rollingset-m",
			Namespace:   metav1.NamespaceDefault,
			Annotations: annotations,
		},
		Spec: carbonv1.RollingSetSpec{
			Version: "11111",
			SchedulePlan: rollalgorithm.SchedulePlan{
				Strategy: apps.DeploymentStrategy{
					Type: apps.RollingUpdateDeploymentStrategyType,
					RollingUpdate: &apps.RollingUpdateDeployment{
						MaxUnavailable: func() *intstr.IntOrString { i := intstr.FromInt(2); return &i }(),
						MaxSurge:       func() *intstr.IntOrString { i := intstr.FromInt(2); return &i }(),
					},
				},
				Replicas:           func() *int32 { i := int32(10); return &i }(),
				LatestVersionRatio: func() *int32 { i := int32(0); return &i }(),
			},

			VersionPlan: carbonv1.VersionPlan{
				BroadcastPlan: carbonv1.BroadcastPlan{
					WorkerSchedulePlan: carbonv1.WorkerSchedulePlan{
						MinReadySeconds:      1024,
						ResourceMatchTimeout: 2000,
						WorkerReadyTimeout:   2014,
					},
				},
			},

			Selector: &metav1.LabelSelector{MatchLabels: selector},
		},
		Status: carbonv1.RollingSetStatus{
			LastUpdateStatusTime: metav1.NewTime(updateStatusTime),
		},
	}
	return &rollingset
}

type testController struct {
	wantErr  error
	errTimes int
	runTimes int
	key      string
	DefaultController
}

// GetObj grep replica
func (c *testController) GetObj(namespace, key string) (interface{}, error) {
	return nil, nil
}

// WaitForCacheSync wait informers synced
func (c *testController) WaitForCacheSync(stopCh <-chan struct{}) bool {
	return true
}

// DeleteSubObj do garbage collect
func (c *testController) DeleteSubObj(namespace, key string) error {
	return nil
}

// Sync compares the actual state with the desired, and attempts to converge the two.
func (c *testController) Sync(key string) (err error) {
	// Convert the namespace/name string into a distinct namespace and name
	c.runTimes++
	if c.runTimes < c.errTimes {
		return c.wantErr
	}
	return nil
}

func TestProcess(t *testing.T) {
	c := &testController{
		wantErr:  fmt.Errorf("err"),
		errTimes: 2,
		key:      "namespace/name",
	}
	var controllerKind = carbonv1.SchemeGroupVersion.WithKind("Admin")
	c.DefaultController = *NewDefaultController(nil, nil, "testagent", controllerKind, c)
	c.workqueue.Add("namespace/name")
	for i := 0; i < 2; i++ {
		c.processNextWorkItem()
	}

	if c.runTimes != 2 {
		t.Fatalf("runtimes error :want %d,get %d", 2, c.runTimes)
	}
}

func TestDefaultController_getResourceUpdateWaitTimeDuration(t *testing.T) {
	key := "key"

	type fields struct {
		Controller           Controller
		controllerAgentName  string
		GroupVersionKind     schema.GroupVersionKind
		workqueue            workqueue.RateLimitingInterface
		kubeclientset        kubernetes.Interface
		carbonclientset      clientset.Interface
		Recorder             record.EventRecorder
		ResourceManager      util.ResourceManager
		start                chan struct{}
		caches               map[string]map[string]string
		DelayProcessDuration time.Duration
		Waiter               *sync.WaitGroup
		resourceUpdateTime   map[string]time.Time
	}
	type args struct {
		key string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   time.Duration
	}{
		{
			name: "new resource",
			fields: fields{
				resourceUpdateTime:   map[string]time.Time{},
				DelayProcessDuration: 5 * time.Second,
			},
			args: args{
				key: key,
			},
			want: 0,
		},
		{
			name: "| now | updateTime |",
			fields: fields{
				resourceUpdateTime: map[string]time.Time{
					key: time.Now().Add(4 * time.Second),
				},
				DelayProcessDuration: 5 * time.Second,
			},
			args: args{
				key: key,
			},
			want: 4 * time.Second,
		},
		{
			name: "| updateTime | now | nextUpdateTime",
			fields: fields{
				resourceUpdateTime: map[string]time.Time{
					key: time.Now().Add(-2 * time.Second),
				},
				DelayProcessDuration: 5 * time.Second,
			},
			args: args{
				key: key,
			},
			want: 3 * time.Second,
		},
		{
			name: " | updateTime | nextUpdateTime | now",
			fields: fields{
				resourceUpdateTime: map[string]time.Time{
					key: time.Now().Add(-8 * time.Second),
				},
				DelayProcessDuration: 5 * time.Second,
			},
			args: args{
				key: key,
			},
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &DefaultController{
				Controller:           tt.fields.Controller,
				controllerAgentName:  tt.fields.controllerAgentName,
				GroupVersionKind:     tt.fields.GroupVersionKind,
				workqueue:            tt.fields.workqueue,
				kubeclientset:        tt.fields.kubeclientset,
				carbonclientset:      tt.fields.carbonclientset,
				ResourceManager:      tt.fields.ResourceManager,
				start:                tt.fields.start,
				caches:               tt.fields.caches,
				DelayProcessDuration: tt.fields.DelayProcessDuration,
				Waiter:               tt.fields.Waiter,
				resourceUpdateTime:   tt.fields.resourceUpdateTime,
			}
			if got := c.getResourceUpdateWaitTimeDuration(tt.args.key); math.Round(got.Seconds()) != tt.want.Seconds() {
				t.Errorf("DefaultController.getResourceUpdateWaitTimeDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}
