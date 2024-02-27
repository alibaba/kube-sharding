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

package rollingset

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/alibaba/kube-sharding/controller"
	"github.com/alibaba/kube-sharding/controller/util"
	"github.com/alibaba/kube-sharding/controller/util/mock"
	mock_util "github.com/alibaba/kube-sharding/controller/util/mock"
	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	clientset "github.com/alibaba/kube-sharding/pkg/client/clientset/versioned"
	listers "github.com/alibaba/kube-sharding/pkg/client/listers/carbon/v1"
	mock_v1 "github.com/alibaba/kube-sharding/pkg/client/listers/carbon/v1/mock"
	"github.com/alibaba/kube-sharding/pkg/rollalgorithm"
	"github.com/golang/mock/gomock"
	apps "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	k8scontroller "k8s.io/kubernetes/pkg/controller"
)

func newRollingset(version string, minReadySeconds int) *carbonv1.RollingSet {
	selector := map[string]string{"foo": "bar"}
	metaStr := `{"metadata":{"name":"nginx-deployment","labels":{"app": "nginx"}}}`
	var template carbonv1.HippoPodTemplate
	json.Unmarshal([]byte(metaStr), &template)
	rollingset := carbonv1.RollingSet{
		TypeMeta: metav1.TypeMeta{APIVersion: "carbon/v1", Kind: "Rollingset"},
		ObjectMeta: metav1.ObjectMeta{
			UID:         uuid.NewUUID(),
			Name:        "rollingset-m",
			Namespace:   metav1.NamespaceDefault,
			Annotations: make(map[string]string),
		},
		Spec: carbonv1.RollingSetSpec{
			Version: version,
			SchedulePlan: rollalgorithm.SchedulePlan{
				Strategy: apps.DeploymentStrategy{
					Type: apps.RollingUpdateDeploymentStrategyType,
					RollingUpdate: &apps.RollingUpdateDeployment{
						MaxUnavailable: func() *intstr.IntOrString { i := intstr.FromInt(2); return &i }(),
						MaxSurge:       func() *intstr.IntOrString { i := intstr.FromInt(2); return &i }(),
					},
				},
				Replicas:           func() *int32 { i := int32(10); return &i }(),
				LatestVersionRatio: func() *int32 { i := int32(rollalgorithm.DefaultLatestVersionRatio); return &i }(),
			},
			VersionPlan: carbonv1.VersionPlan{
				BroadcastPlan: carbonv1.BroadcastPlan{
					WorkerSchedulePlan: carbonv1.WorkerSchedulePlan{
						MinReadySeconds:      int32(minReadySeconds),
						ResourceMatchTimeout: 2000,
						WorkerReadyTimeout:   2014,
					},
				},
				SignedVersionPlan: carbonv1.SignedVersionPlan{
					RestartAfterResourceChange: utils.BoolPtr(true),
					Template:                   &template,
				},
			},

			Selector: &metav1.LabelSelector{MatchLabels: selector},
		},
		Status: carbonv1.RollingSetStatus{},
	}
	return &rollingset
}

func TestController_updateVersion(t *testing.T) {
	type fields struct {
		kubeclientset    kubernetes.Interface
		carbonclientset  clientset.Interface
		rollingSetLister listers.RollingSetLister
		rollingSetSynced cache.InformerSynced

		replicaSynced cache.InformerSynced
		workqueue     workqueue.RateLimitingInterface
		recorder      record.EventRecorder
	}
	type args struct {
		rollingset *carbonv1.RollingSet
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		version string
	}{
		{
			name: "test1",
			args: args{
				rollingset: newRollingset("version-1", 1024),
			},
			version: "0-49aaee12ea1535921bc8a056cb467c69",
		},
		{
			name: "test2",
			args: args{
				rollingset: newRollingset("c56910e3d14ba83dc9bd3e42795dad61", 1024),
			},
			version: "0-49aaee12ea1535921bc8a056cb467c69",
		},
		{
			name: "test3",
			args: args{
				rollingset: func() *carbonv1.RollingSet {
					rollingset := newRollingset("3-49aaee12ea1535921bc8a056cb467c69", 1024)
					rollingset.Generation = 3
					return rollingset
				}(),
			},
			version: "3-49aaee12ea1535921bc8a056cb467c69",
		},
		{
			name: "test4",
			args: args{
				rollingset: func() *carbonv1.RollingSet {
					rollingset := newRollingset("3-dcef", 1024)
					rollingset.Generation = 3
					return rollingset
				}(),
			},
			version: "3-49aaee12ea1535921bc8a056cb467c69",
		},
		{
			name: "test5",
			args: args{
				rollingset: func() *carbonv1.RollingSet {
					rollingset := newRollingset("3-49aaee12ea1535921bc8a056cb467c69", 1024)
					rollingset.Generation = 4
					return rollingset
				}(),
			},
			version: "3-49aaee12ea1535921bc8a056cb467c69",
		},
		{
			name: "test6",
			args: args{
				rollingset: func() *carbonv1.RollingSet {
					rollingset := newRollingset("3-dcef", 1024)
					rollingset.Generation = 4
					return rollingset
				}(),
			},
			version: "4-49aaee12ea1535921bc8a056cb467c69",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Controller{
				rollingSetLister: tt.fields.rollingSetLister,
				rollingSetSynced: tt.fields.rollingSetSynced,
			}
			c.updateVersion(tt.args.rollingset)
			if tt.args.rollingset.Spec.Version != tt.version {
				t.Errorf("InPlaceScheduler.releaseReplicaReplicas() = %v, want %v", tt.args.rollingset.Spec.Version, tt.version)
			}
		})
	}
}

func newRollingsetFinalizer(finalizer string) *carbonv1.RollingSet {
	selector := map[string]string{"foo": "bar"}
	rollingset := carbonv1.RollingSet{
		TypeMeta: metav1.TypeMeta{APIVersion: "carbon/v1", Kind: "Rollingset"},
		ObjectMeta: metav1.ObjectMeta{
			UID:         uuid.NewUUID(),
			Name:        "rollingset-m",
			Namespace:   metav1.NamespaceDefault,
			Annotations: make(map[string]string),
			Finalizers:  []string{finalizer, "ssa"},
		},
		Spec: carbonv1.RollingSetSpec{
			Version: "version1",
			SchedulePlan: rollalgorithm.SchedulePlan{
				Strategy: apps.DeploymentStrategy{
					Type: apps.RollingUpdateDeploymentStrategyType,
					RollingUpdate: &apps.RollingUpdateDeployment{
						MaxUnavailable: func() *intstr.IntOrString { i := intstr.FromInt(2); return &i }(),
						MaxSurge:       func() *intstr.IntOrString { i := intstr.FromInt(2); return &i }(),
					},
				},
				Replicas:           func() *int32 { i := int32(10); return &i }(),
				LatestVersionRatio: func() *int32 { i := int32(rollalgorithm.DefaultLatestVersionRatio); return &i }(),
			},
			VersionPlan: carbonv1.VersionPlan{
				BroadcastPlan: carbonv1.BroadcastPlan{
					WorkerSchedulePlan: carbonv1.WorkerSchedulePlan{
						MinReadySeconds:      1000,
						ResourceMatchTimeout: 2000,
						WorkerReadyTimeout:   2014,
					},
				},
			},
			Selector: &metav1.LabelSelector{MatchLabels: selector},
		},
		Status: carbonv1.RollingSetStatus{},
	}
	return &rollingset
}

func newRollingSetDeleteOperate(finalizer string) *carbonv1.RollingSet {
	selector := map[string]string{"foo": "bar"}
	rollingset := carbonv1.RollingSet{
		TypeMeta: metav1.TypeMeta{APIVersion: "carbon/v1", Kind: "Rollingset"},
		ObjectMeta: metav1.ObjectMeta{
			UID:               uuid.NewUUID(),
			Name:              "rollingset-m",
			Namespace:         metav1.NamespaceDefault,
			Annotations:       make(map[string]string),
			Finalizers:        []string{finalizer},
			DeletionTimestamp: func() *metav1.Time { time := metav1.Now(); return &time }(),
		},
		Spec: carbonv1.RollingSetSpec{
			Version: "version-1",
			SchedulePlan: rollalgorithm.SchedulePlan{
				Strategy: apps.DeploymentStrategy{
					Type: apps.RollingUpdateDeploymentStrategyType,
					RollingUpdate: &apps.RollingUpdateDeployment{
						MaxUnavailable: func() *intstr.IntOrString { i := intstr.FromInt(2); return &i }(),
						MaxSurge:       func() *intstr.IntOrString { i := intstr.FromInt(2); return &i }(),
					},
				},
				Replicas:           func() *int32 { i := int32(10); return &i }(),
				LatestVersionRatio: func() *int32 { i := int32(rollalgorithm.DefaultLatestVersionRatio); return &i }(),
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
		Status: carbonv1.RollingSetStatus{},
	}
	return &rollingset
}

func newRollingSetDeleteOperateNof(finalizer string) *carbonv1.RollingSet {
	selector := map[string]string{"foo": "bar"}
	rollingset := carbonv1.RollingSet{
		TypeMeta: metav1.TypeMeta{APIVersion: "carbon/v1", Kind: "Rollingset"},
		ObjectMeta: metav1.ObjectMeta{
			UID:         uuid.NewUUID(),
			Name:        "rollingset-m",
			Namespace:   metav1.NamespaceDefault,
			Annotations: make(map[string]string),
			//	Finalizers:        []string{finalizer},
			DeletionTimestamp: func() *metav1.Time { time := metav1.Now(); return &time }(),
		},
		Spec: carbonv1.RollingSetSpec{
			Version: "version-1",
			SchedulePlan: rollalgorithm.SchedulePlan{
				Strategy: apps.DeploymentStrategy{
					Type: apps.RollingUpdateDeploymentStrategyType,
					RollingUpdate: &apps.RollingUpdateDeployment{
						MaxUnavailable: func() *intstr.IntOrString { i := intstr.FromInt(2); return &i }(),
						MaxSurge:       func() *intstr.IntOrString { i := intstr.FromInt(2); return &i }(),
					},
				},
				Replicas:           func() *int32 { i := int32(10); return &i }(),
				LatestVersionRatio: func() *int32 { i := int32(rollalgorithm.DefaultLatestVersionRatio); return &i }(),
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
		Status: carbonv1.RollingSetStatus{},
	}
	return &rollingset
}

func TestController_deleteOperate(t *testing.T) {
	ctl := gomock.NewController(t)
	defer ctl.Finish()
	mockResourceManager := mock.NewMockResourceManager(ctl)
	mockResourceManager.EXPECT().UpdateRollingSet(gomock.Any()).Return(nil, nil)
	mockResourceManager.EXPECT().RemoveRollingSet(gomock.Any()).Return(nil).MaxTimes(2)

	type fields struct {
		kubeclientset    kubernetes.Interface
		carbonclientset  clientset.Interface
		rollingSetLister listers.RollingSetLister
		rollingSetSynced cache.InformerSynced

		replicaSynced   cache.InformerSynced
		workqueue       workqueue.RateLimitingInterface
		recorder        record.EventRecorder
		resourceManager util.ResourceManager
	}
	type args struct {
		rollingset *carbonv1.RollingSet
		replicas   *[]*carbonv1.Replica
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		isReturn bool
		wantErr  bool
	}{
		{
			name: "test1",
			args: args{rollingset: newRollingSetDeleteOperate(""),
				replicas: newReplicasBase("", 10)},
			isReturn: false,
			wantErr:  false,
		},
		{
			name: "test2",
			args: args{rollingset: newRollingSetDeleteOperate("smoothDeletion"),
				replicas: newReplicasBase("", 10)},
			isReturn: false,
			wantErr:  false,
		},
		{
			name: "test3",
			args: args{rollingset: newRollingSetDeleteOperate("smoothDeletion"),
				replicas: newReplicasBase("", 0)},
			isReturn: true,
			wantErr:  false,
		},
		{
			name: "test4",
			args: args{rollingset: newRollingSetDeleteOperateNof(""),
				replicas: newReplicasBase("", 10)},
			isReturn: false,
			wantErr:  false,
		},

		{
			name: "test5",
			args: args{rollingset: newRollingSetDeleteOperateNof(""),
				replicas: newReplicasBase("", 0)},
			isReturn: true,
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Controller{
				rollingSetLister: tt.fields.rollingSetLister,
				rollingSetSynced: tt.fields.rollingSetSynced,
			}
			c.ResourceManager = mockResourceManager

			got, err := c.deleteOperate(tt.args.rollingset, tt.args.replicas)
			if (err != nil) != tt.wantErr {
				t.Errorf("Controller.deleteOperate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.isReturn {
				t.Errorf("Controller.deleteOperate() = %v, want %v", got, tt.isReturn)
			}

		})
	}
}

func TestController_fillFinalizer(t *testing.T) {
	ctl := gomock.NewController(t)
	defer ctl.Finish()
	mockResourceManager := mock.NewMockResourceManager(ctl)
	mockResourceManager.EXPECT().UpdateRollingSet(gomock.Any()).Return(nil, nil).MaxTimes(10)

	type fields struct {
		kubeclientset    kubernetes.Interface
		carbonclientset  clientset.Interface
		rollingSetLister listers.RollingSetLister
		rollingSetSynced cache.InformerSynced

		replicaSynced   cache.InformerSynced
		workqueue       workqueue.RateLimitingInterface
		recorder        record.EventRecorder
		resourceManager util.ResourceManager
	}
	type args struct {
		rollingset *carbonv1.RollingSet
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		{
			name:   "test1",
			fields: fields{resourceManager: mockResourceManager},
			args:   args{rollingset: newRollingsetFinalizer("abc")},
			want:   false,
		},
		{
			name:   "test2",
			fields: fields{resourceManager: mockResourceManager},
			args:   args{rollingset: newRollingsetFinalizer(carbonv1.FinalizerSmoothDeletion)},
			want:   false,
		},
		{
			name:   "test3",
			fields: fields{resourceManager: mockResourceManager},
			args:   args{rollingset: newRollingset("abc", 1)},
			want:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Controller{
				rollingSetLister: tt.fields.rollingSetLister,
				rollingSetSynced: tt.fields.rollingSetSynced,
			}
			c.ResourceManager = mockResourceManager

			got, err := c.completeFields(tt.args.rollingset)
			if (err != nil) != tt.wantErr {
				t.Errorf("Controller.fillFinalizer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Controller.fillFinalizer() = %v, want %v", got, tt.want)
			}
			if !HasSmoothFinalizer(tt.args.rollingset) {
				t.Errorf("Controller.fillFinalizer() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestController_DeleteSubObj(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	name := "test"
	namespace := "namespae"

	workerNode := &carbonv1.WorkerNode{}
	workerNode.Name = "test"

	type fields struct {
		resourceManager  util.ResourceManager
		rollingSetLister listers.RollingSetLister
		rollingSetSynced cache.InformerSynced
		workerLister     listers.WorkerNodeLister
		workerSynced     cache.InformerSynced
		expectations     *k8scontroller.UIDTrackingControllerExpectations
	}
	type args struct {
		namespace string
		name      string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "get failed, error",
			fields: fields{
				resourceManager: func() util.ResourceManager {
					rm := mock_util.NewMockResourceManager(ctrl)
					return rm
				}(),
				workerLister: func() listers.WorkerNodeLister {
					namespaceLister := mock_v1.NewMockWorkerNodeNamespaceLister(ctrl)
					namespaceLister.EXPECT().Get(name).Return(nil, fmt.Errorf("failed"))
					lister := mock_v1.NewMockWorkerNodeLister(ctrl)
					lister.EXPECT().WorkerNodes(namespace).Return(namespaceLister)
					return lister
				}(),
			},
			args: args{
				namespace: namespace,
				name:      name,
			},
			wantErr: true,
		},
		{
			name: "get failed, not exist",
			fields: fields{
				resourceManager: func() util.ResourceManager {
					rm := mock_util.NewMockResourceManager(ctrl)
					return rm
				}(),
				workerLister: func() listers.WorkerNodeLister {
					namespaceLister := mock_v1.NewMockWorkerNodeNamespaceLister(ctrl)
					namespaceLister.EXPECT().Get(name).Return(nil, errors.NewNotFound(schema.GroupResource{}, ""))
					lister := mock_v1.NewMockWorkerNodeLister(ctrl)
					lister.EXPECT().WorkerNodes(namespace).Return(namespaceLister)
					return lister
				}(),
			},
			args: args{
				namespace: namespace,
				name:      name,
			},
			wantErr: false,
		},
		{
			name: "get success, delete failed",
			fields: fields{
				resourceManager: func() util.ResourceManager {
					rm := mock_util.NewMockResourceManager(ctrl)
					rm.EXPECT().DeleteWorkerNode(workerNode).Return(fmt.Errorf("failed"))
					return rm
				}(),
				workerLister: func() listers.WorkerNodeLister {
					namespaceLister := mock_v1.NewMockWorkerNodeNamespaceLister(ctrl)
					namespaceLister.EXPECT().Get(name).Return(workerNode, nil)
					lister := mock_v1.NewMockWorkerNodeLister(ctrl)
					lister.EXPECT().WorkerNodes(namespace).Return(namespaceLister)
					return lister
				}(),
			},
			args: args{
				namespace: namespace,
				name:      name,
			},
			wantErr: true,
		},
		{
			name: "get success, delete not exist",
			fields: fields{
				resourceManager: func() util.ResourceManager {
					rm := mock_util.NewMockResourceManager(ctrl)
					rm.EXPECT().DeleteWorkerNode(workerNode).Return(errors.NewNotFound(schema.GroupResource{}, ""))
					return rm
				}(),
				workerLister: func() listers.WorkerNodeLister {
					namespaceLister := mock_v1.NewMockWorkerNodeNamespaceLister(ctrl)
					namespaceLister.EXPECT().Get(name).Return(workerNode, nil)
					lister := mock_v1.NewMockWorkerNodeLister(ctrl)
					lister.EXPECT().WorkerNodes(namespace).Return(namespaceLister)
					return lister
				}(),
			},
			args: args{
				namespace: namespace,
				name:      name,
			},
			wantErr: false,
		},
		{
			name: "get success, delete success",
			fields: fields{
				resourceManager: func() util.ResourceManager {
					rm := mock_util.NewMockResourceManager(ctrl)
					rm.EXPECT().DeleteWorkerNode(workerNode).Return(nil)
					return rm
				}(),
				workerLister: func() listers.WorkerNodeLister {
					namespaceLister := mock_v1.NewMockWorkerNodeNamespaceLister(ctrl)
					namespaceLister.EXPECT().Get(name).Return(workerNode, nil)
					lister := mock_v1.NewMockWorkerNodeLister(ctrl)
					lister.EXPECT().WorkerNodes(namespace).Return(namespaceLister)
					return lister
				}(),
			},
			args: args{
				namespace: namespace,
				name:      name,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Controller{
				DefaultController: controller.DefaultController{
					ResourceManager: tt.fields.resourceManager,
				},
				rollingSetLister: tt.fields.rollingSetLister,
				rollingSetSynced: tt.fields.rollingSetSynced,
				workerLister:     tt.fields.workerLister,
				workerSynced:     tt.fields.workerSynced,
				expectations:     tt.fields.expectations,
			}
			if err := c.DeleteSubObj(tt.args.namespace, tt.args.name); (err != nil) != tt.wantErr {
				t.Errorf("Controller.DeleteSubObj() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
