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

package shardgroup

import (
	RawErrors "errors"
	"math"
	"testing"

	"github.com/alibaba/kube-sharding/controller"
	"github.com/alibaba/kube-sharding/controller/util"
	"github.com/alibaba/kube-sharding/controller/util/mock"
	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	"github.com/alibaba/kube-sharding/pkg/client/clientset/versioned/fake"
	carboninformers "github.com/alibaba/kube-sharding/pkg/client/informers/externalversions"
	listers "github.com/alibaba/kube-sharding/pkg/client/listers/carbon/v1"
	"github.com/alibaba/kube-sharding/pkg/rollalgorithm"
	"github.com/golang/mock/gomock"
	apps "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"
	k8scontroller "k8s.io/kubernetes/pkg/controller"
)

func newShardTemplate(replicas int32) *carbonv1.ShardTemplate {
	shardTemplate := carbonv1.ShardTemplate{
		Spec: carbonv1.ShardSpec{
			Replicas: &replicas,
		},
	}
	return &shardTemplate
}

func newShardGroupWithShardKey(name string, shardNames []string, replicas int32) *carbonv1.ShardGroup {
	selector := map[string]string{"foo": "bar"}
	maxUnavailable := intstr.FromInt(2)

	shardGroup := carbonv1.ShardGroup{
		TypeMeta: metav1.TypeMeta{APIVersion: "carbon/v1", Kind: "Rollingset"},
		ObjectMeta: metav1.ObjectMeta{
			UID:         uuid.NewUUID(),
			Name:        name,
			Namespace:   metav1.NamespaceDefault,
			Annotations: make(map[string]string),
		},
		Spec: carbonv1.ShardGroupSpec{
			RollingVersion: "21",
			Selector:       &metav1.LabelSelector{MatchLabels: selector},
			MaxUnavailable: &maxUnavailable,
			ShardTemplates: map[string]carbonv1.ShardTemplate{
				"0": *newShardTemplate(1),
			},
			LatestPercent: utils.Int32Ptr(100),
		},
		Status: carbonv1.ShardGroupStatus{
			ObservedGeneration: 1000,
		},
	}
	shardGroup.Spec.ShardTemplates = map[string]carbonv1.ShardTemplate{}
	for i, shardName := range shardNames {
		if replicas < 0 {
			replicas = int32(i + 1)
		}
		shardGroup.Spec.ShardTemplates[shardName] = *newShardTemplate(replicas)
	}
	return &shardGroup
}

func newShardGroup(name string, rollingVersion string, deleted bool) *carbonv1.ShardGroup {
	selector := map[string]string{"foo": "bar"}
	maxUnavailable := intstr.FromInt(2)

	shardGroup := carbonv1.ShardGroup{
		TypeMeta: metav1.TypeMeta{APIVersion: "carbon/v1", Kind: "Rollingset"},
		ObjectMeta: metav1.ObjectMeta{
			UID:         uuid.NewUUID(),
			Name:        name,
			Namespace:   metav1.NamespaceDefault,
			Annotations: make(map[string]string),
			Generation:  1,
		},
		Spec: carbonv1.ShardGroupSpec{
			RollingVersion: rollingVersion,
			Selector:       &metav1.LabelSelector{MatchLabels: selector},
			MaxUnavailable: &maxUnavailable,
			ShardTemplates: map[string]carbonv1.ShardTemplate{
				"0": *newShardTemplate(1),
			},
			LatestPercent: func() *int32 { i := int32(100); return &i }(),
		},
		Status: carbonv1.ShardGroupStatus{
			ObservedGeneration: 1000,
			ShardGroupVersion:  "0-c25ede9bcbdecfa1b6eaa3d664d75d45",
		},
	}

	if deleted {
		now := v1.Now()
		shardGroup.DeletionTimestamp = &now
	}
	return &shardGroup
}
func newShardGroupCompleted(name string, strategy string) *carbonv1.ShardGroup {
	selector := map[string]string{"shardgroup": name}
	maxUnavailable := intstr.FromInt(2)

	shardGroup := carbonv1.ShardGroup{
		TypeMeta: metav1.TypeMeta{APIVersion: "carbon/v1", Kind: "Rollingset"},
		ObjectMeta: metav1.ObjectMeta{
			UID:         uuid.NewUUID(),
			Name:        name,
			Namespace:   metav1.NamespaceDefault,
			Annotations: make(map[string]string),
			Labels: map[string]string{
				"shardgroup": name,
			},
			Finalizers: []string{
				"smoothDeletion",
			},
		},
		Spec: carbonv1.ShardGroupSpec{
			Selector:       &metav1.LabelSelector{MatchLabels: selector},
			MaxUnavailable: &maxUnavailable,
			ShardTemplates: map[string]carbonv1.ShardTemplate{
				"0": *newShardTemplate(1),
			},
			ShardDeployStrategy: carbonv1.ShardDeployType(strategy),
		},
		Status: carbonv1.ShardGroupStatus{
			ObservedGeneration: 1000,
		},
	}

	return &shardGroup
}

func newShardGroupCustominfo(name string, rollingVersion string, custominfo, compressedCustominfo string) *carbonv1.ShardGroup {
	selector := map[string]string{"foo": "bar"}
	maxUnavailable := intstr.FromInt(2)

	shardGroup := carbonv1.ShardGroup{
		TypeMeta: metav1.TypeMeta{APIVersion: "carbon/v1", Kind: "Rollingset"},
		ObjectMeta: metav1.ObjectMeta{
			UID:         uuid.NewUUID(),
			Name:        name,
			Namespace:   metav1.NamespaceDefault,
			Annotations: make(map[string]string),
			Generation:  1,
		},
		Spec: carbonv1.ShardGroupSpec{
			RollingVersion: rollingVersion,
			Selector:       &metav1.LabelSelector{MatchLabels: selector},
			MaxUnavailable: &maxUnavailable,
			ShardTemplates: map[string]carbonv1.ShardTemplate{
				"0": {
					Spec: carbonv1.ShardSpec{
						CustomInfo:           custominfo,
						CompressedCustomInfo: compressedCustominfo,
					},
				},
			},
			LatestPercent: func() *int32 { i := int32(100); return &i }(),
		},
		Status: carbonv1.ShardGroupStatus{
			ObservedGeneration: 1000,
			ShardGroupVersion:  "0-c25ede9bcbdecfa1b6eaa3d664d75d45",
		},
	}

	return &shardGroup
}

func newRollingSet(name string, version string) *carbonv1.RollingSet {
	selector := map[string]string{"foo": "bar"}
	rollingset := carbonv1.RollingSet{
		TypeMeta: metav1.TypeMeta{APIVersion: "carbon/v1", Kind: "Rollingset"},
		ObjectMeta: metav1.ObjectMeta{
			UID:         uuid.NewUUID(),
			Name:        name,
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
				LatestVersionRatio: func() *int32 { i := int32(100); return &i }(),
			},
			VersionPlan: carbonv1.VersionPlan{
				SignedVersionPlan: carbonv1.SignedVersionPlan{
					RestartAfterResourceChange: utils.BoolPtr(true),
				},
				BroadcastPlan: carbonv1.BroadcastPlan{
					WorkerSchedulePlan: carbonv1.WorkerSchedulePlan{
						MinReadySeconds:      100,
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

func newRollingSetCustominfo(name string, version string, custominfo, compressedCustominfo string) *carbonv1.RollingSet {
	selector := map[string]string{"foo": "bar"}
	rollingset := carbonv1.RollingSet{
		TypeMeta: metav1.TypeMeta{APIVersion: "carbon/v1", Kind: "Rollingset"},
		ObjectMeta: metav1.ObjectMeta{
			UID:         uuid.NewUUID(),
			Name:        name,
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
				LatestVersionRatio: func() *int32 { i := int32(100); return &i }(),
			},
			VersionPlan: carbonv1.VersionPlan{
				SignedVersionPlan: carbonv1.SignedVersionPlan{
					RestartAfterResourceChange: utils.BoolPtr(true),
				},
				BroadcastPlan: carbonv1.BroadcastPlan{
					CustomInfo:           custominfo,
					CompressedCustomInfo: compressedCustominfo,
					WorkerSchedulePlan: carbonv1.WorkerSchedulePlan{
						MinReadySeconds:      100,
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

func newRollingSetStatus(name string, replicas int32, lastestVeersionRatio int32, versionReplicas map[string]int32) *carbonv1.RollingSet {
	selector := map[string]string{"foo": "bar"}
	rollingset := carbonv1.RollingSet{
		TypeMeta: metav1.TypeMeta{APIVersion: "carbon/v1", Kind: "Rollingset"},
		ObjectMeta: metav1.ObjectMeta{
			UID:         uuid.NewUUID(),
			Name:        name,
			Namespace:   metav1.NamespaceDefault,
			Annotations: make(map[string]string),
		},
		Spec: carbonv1.RollingSetSpec{
			Version: "default",
			SchedulePlan: rollalgorithm.SchedulePlan{
				Strategy: apps.DeploymentStrategy{
					Type: apps.RollingUpdateDeploymentStrategyType,
					RollingUpdate: &apps.RollingUpdateDeployment{
						MaxUnavailable: func() *intstr.IntOrString { i := intstr.FromInt(2); return &i }(),
						MaxSurge:       func() *intstr.IntOrString { i := intstr.FromInt(2); return &i }(),
					},
				},
				Replicas:           func() *int32 { return &replicas }(),
				LatestVersionRatio: func() *int32 { return &lastestVeersionRatio }(),
			},
			VersionPlan: carbonv1.VersionPlan{
				SignedVersionPlan: carbonv1.SignedVersionPlan{
					RestartAfterResourceChange: utils.BoolPtr(true),
				},
				BroadcastPlan: carbonv1.BroadcastPlan{
					WorkerSchedulePlan: carbonv1.WorkerSchedulePlan{
						MinReadySeconds:      100,
						ResourceMatchTimeout: 2000,
						WorkerReadyTimeout:   2014,
					},
				},
			},
			Selector: &metav1.LabelSelector{MatchLabels: selector},
		},
	}

	groupVersionStatusMap := map[string]*rollalgorithm.VersionStatus{}
	for key, val := range versionReplicas {
		groupVersionStatusMap[key] = &rollalgorithm.VersionStatus{
			ReadyReplicas: val,
			Version:       key,
			Replicas:      val,
		}
	}
	rollingset.Status.GroupVersionStatusMap = groupVersionStatusMap
	return &rollingset
}

func TestController_syncRollingSetGC(t *testing.T) {
	ctl := gomock.NewController(t)
	defer ctl.Finish()
	mockResourceManager := mock.NewMockResourceManager(ctl)
	mockResourceManager.EXPECT().UpdateShardGroup(gomock.Any()).Return(nil).MaxTimes(1)
	mockResourceManager.EXPECT().RemoveRollingSet(gomock.Any()).Return(nil).MaxTimes(10)
	mockResourceManager.EXPECT().DeleteServicePublisherForRs(gomock.Any()).Return(nil).MaxTimes(10)
	mockResourceManager.EXPECT().DeleteShardGroup(gomock.Any()).Return(nil).MaxTimes(1)
	type fields struct {
		shardGroupLister listers.ShardGroupLister
		shardGroupSynced cache.InformerSynced
		rollingSetLister listers.RollingSetLister
		rollingSetSynced cache.InformerSynced
	}
	type args struct {
		shardgroup  *carbonv1.ShardGroup
		rollingSets []*carbonv1.RollingSet
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "test1",
			args: args{
				shardgroup: newShardGroup("sg1", "1", true),
			},
			wantErr: false,
		},
		{
			name: "test2",
			args: args{
				shardgroup: newShardGroup("sg2", "2", true),
				rollingSets: []*carbonv1.RollingSet{
					newRollingSet("rs1", "21"),
				},
			},
			wantErr: true,
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Controller{
				shardGroupLister: tt.fields.shardGroupLister,
				shardGroupSynced: tt.fields.shardGroupSynced,
				rollingSetLister: tt.fields.rollingSetLister,
				rollingSetSynced: tt.fields.rollingSetSynced,
			}
			c.DefaultController = controller.DefaultController{
				ResourceManager: mockResourceManager,
			}

			if err := c.syncRollingSetGC(tt.args.shardgroup, tt.args.rollingSets); (err != nil) != tt.wantErr {
				t.Errorf("Controller.syncRollingSetGC() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestController_createRollingSet(t *testing.T) {
	ctl := gomock.NewController(t)
	defer ctl.Finish()
	mockResourceManager := mock.NewMockResourceManager(ctl)
	mockResourceManager.EXPECT().CreateRollingSet(gomock.Any(), gomock.Any()).Return(nil, nil).MaxTimes(1)
	mockResourceManager.EXPECT().CreateRollingSet(gomock.Any(), gomock.Any()).Return(nil, RawErrors.New("ok")).MaxTimes(1)

	type fields struct {
		shardGroupLister listers.ShardGroupLister
		shardGroupSynced cache.InformerSynced
		rollingSetLister listers.RollingSetLister
		rollingSetSynced cache.InformerSynced
	}
	type args struct {
		shardGroup     *carbonv1.ShardGroup
		shardTemplate  *carbonv1.ShardTemplate
		shardName      string
		shardGroupSign string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "test1",
			args: args{
				shardGroup:    newShardGroup("sg1", "123", false),
				shardTemplate: newShardTemplate(1),
			},
			wantErr: false,
		},
		{
			name: "test1",
			args: args{
				shardGroup:    newShardGroup("sg1", "123", false),
				shardTemplate: newShardTemplate(1),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Controller{
				shardGroupLister: tt.fields.shardGroupLister,
				shardGroupSynced: tt.fields.shardGroupSynced,
				rollingSetLister: tt.fields.rollingSetLister,
				rollingSetSynced: tt.fields.rollingSetSynced,
			}
			c.DefaultController = controller.DefaultController{
				ResourceManager: mockResourceManager,
			}
			err := c.createRollingSet(tt.args.shardGroup, tt.args.shardTemplate, tt.args.shardName, tt.args.shardGroupSign)
			if (err != nil) != tt.wantErr {
				t.Errorf("Controller.createRollingSet() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestController_autoComplete(t *testing.T) {
	type fields struct {
		shardGroupLister listers.ShardGroupLister
		shardGroupSynced cache.InformerSynced
		rollingSetLister listers.RollingSetLister
		rollingSetSynced cache.InformerSynced
	}
	type args struct {
		shardgroup *carbonv1.ShardGroup
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		{
			name:    "test1",
			args:    args{shardgroup: newShardGroup("abc", "1", false)},
			want:    false,
			wantErr: false,
		},
		{
			name:    "test2",
			args:    args{shardgroup: newShardGroup("cde", "2", true)},
			want:    false,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Controller{
				shardGroupLister: tt.fields.shardGroupLister,
				shardGroupSynced: tt.fields.shardGroupSynced,
				rollingSetLister: tt.fields.rollingSetLister,
				rollingSetSynced: tt.fields.rollingSetSynced,
			}
			got, err := c.autoComplete(tt.args.shardgroup)
			if (err != nil) != tt.wantErr {
				t.Errorf("Controller.autoComplete() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Controller.autoComplete() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestController_syncRollingSetGroupStatus(t *testing.T) {
	ctl := gomock.NewController(t)
	defer ctl.Finish()
	mockResourceManager := mock.NewMockResourceManager(ctl)
	mockResourceManager.EXPECT().UpdateShardGroupStatus(gomock.Any()).Return(nil).MaxTimes(1)

	type fields struct {
		shardGroupLister listers.ShardGroupLister
		shardGroupSynced cache.InformerSynced
		rollingSetLister listers.RollingSetLister
		rollingSetSynced cache.InformerSynced
	}
	type args struct {
		shardGroup        *carbonv1.ShardGroup
		rollingSets       []*carbonv1.RollingSet
		shardGroupVersion string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "test1",
			args: args{
				shardGroup: newShardGroup("sg1", "1", false),
				/*rollingSets: []*carbonv1.RollingSet{
					newRollingSet("sg1.rs1", "12"),
				},*/
				shardGroupVersion: "123",
			},
			wantErr: false,
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Controller{
				shardGroupLister: tt.fields.shardGroupLister,
				shardGroupSynced: tt.fields.shardGroupSynced,
				rollingSetLister: tt.fields.rollingSetLister,
				rollingSetSynced: tt.fields.rollingSetSynced,
			}
			c.DefaultController = controller.DefaultController{
				ResourceManager: mockResourceManager,
			}

			if err := c.syncRollingSetGroupStatus(tt.args.shardGroup, tt.args.shardGroup, tt.args.rollingSets, tt.args.shardGroupVersion); (err != nil) != tt.wantErr {
				t.Errorf("Controller.syncRollingSetGroupStatus() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestController_calculateShardGroupSign(t *testing.T) {
	type fields struct {
		shardGroupLister listers.ShardGroupLister
		shardGroupSynced cache.InformerSynced
		rollingSetLister listers.RollingSetLister
		rollingSetSynced cache.InformerSynced
	}
	type args struct {
		shardgroup *carbonv1.ShardGroup
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "test1",
			args: args{
				shardgroup: newShardGroup("sg1", "123", false),
			},
			want:    "1-652da08ffd1c6e59832d903ad3d4d188",
			wantErr: false,
		},

		{
			name: "test2",
			args: args{
				shardgroup: newShardGroupCustominfo("sg1", "123", "", "abc"),
			},
			want:    "1-652da08ffd1c6e59832d903ad3d4d188",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Controller{
				shardGroupLister: tt.fields.shardGroupLister,
				shardGroupSynced: tt.fields.shardGroupSynced,
				rollingSetLister: tt.fields.rollingSetLister,
				rollingSetSynced: tt.fields.rollingSetSynced,
			}
			got, err := c.calculateShardGroupSign(tt.args.shardgroup)
			if (err != nil) != tt.wantErr {
				t.Errorf("Controller.calculateShardGroupSign() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Controller.calculateShardGroupSign() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestController_syncRollingSetSpec(t *testing.T) {
	ctl := gomock.NewController(t)
	defer ctl.Finish()
	mockResourceManagerCreate := mock.NewMockResourceManager(ctl)
	mockResourceManagerCreate.EXPECT().CreateRollingSet(gomock.Any(), gomock.Any()).Return(nil, nil).MaxTimes(1)

	mockResourceManagerUpdate := mock.NewMockResourceManager(ctl)
	mockResourceManagerUpdate.EXPECT().UpdateRollingSet(gomock.Any()).Return(nil, nil).MaxTimes(1)

	mockResourceManagerScaleZero := mock.NewMockResourceManager(ctl)
	mockResourceManagerScaleZero.EXPECT().UpdateRollingSet(gomock.Any()).Return(nil, nil).MaxTimes(1)

	type fields struct {
		shardGroupLister listers.ShardGroupLister
		shardGroupSynced cache.InformerSynced
		rollingSetLister listers.RollingSetLister
		rollingSetSynced cache.InformerSynced
		resourceManager  util.ResourceManager
	}
	type args struct {
		shardGroup        *carbonv1.ShardGroup
		rollingSets       []*carbonv1.RollingSet
		shardGroupVersion string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "create not exist",
			fields: fields{
				resourceManager: mockResourceManagerCreate,
			},
			args: args{
				shardGroup: newShardGroup("sg1", "1", false),
				rollingSets: []*carbonv1.RollingSet{
					newRollingSet("rs1", "12"),
				},
				shardGroupVersion: "sgversion1",
			},
			wantErr: false,
		},
		{
			name: "update",
			fields: fields{
				resourceManager: mockResourceManagerUpdate,
			},
			args: args{
				shardGroup: newShardGroupWithShardKey("sg2", []string{"ok"}, -1),
				rollingSets: []*carbonv1.RollingSet{
					newRollingSet("sg2.ok", "12"),
				},
				shardGroupVersion: "sgversion2",
			},
			wantErr: false,
		},
		{
			name: "scale to zero",
			fields: fields{
				resourceManager: mockResourceManagerScaleZero,
			},
			args: args{
				shardGroup: newShardGroupWithShardKey("sg2", []string{"ok"}, 0),
				rollingSets: []*carbonv1.RollingSet{
					newRollingSet("sg2.ok", "12"),
				},
				shardGroupVersion: "sgversion2",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Controller{
				shardGroupLister: tt.fields.shardGroupLister,
				shardGroupSynced: tt.fields.shardGroupSynced,
				rollingSetLister: tt.fields.rollingSetLister,
				rollingSetSynced: tt.fields.rollingSetSynced,
			}
			c.DefaultController = controller.DefaultController{
				ResourceManager: tt.fields.resourceManager,
			}
			if err := c.syncRollingSetSpec(tt.args.shardGroup, tt.args.rollingSets, tt.args.shardGroupVersion); (err != nil) != tt.wantErr {
				t.Errorf("Controller.syncRollingSetSpec() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestController_Sync(t *testing.T) {
	client := fake.NewSimpleClientset([]runtime.Object{}...)
	informersFactory := carboninformers.NewSharedInformerFactory(client, k8scontroller.NoResyncPeriodFunc())
	shardGroupInformer := informersFactory.Carbon().V1().ShardGroups()
	rollingSetInformer := informersFactory.Carbon().V1().RollingSets()

	shardgroup0 := newShardGroupCompleted("test0", "rollingSet")
	informersFactory.Carbon().V1().ShardGroups().Informer().GetIndexer().Add(shardgroup0)

	shardgroup1 := newShardGroup("test1", "123", false)
	informersFactory.Carbon().V1().ShardGroups().Informer().GetIndexer().Add(shardgroup1)

	shardgroup2 := newShardGroup("test2", "124", true)
	informersFactory.Carbon().V1().ShardGroups().Informer().GetIndexer().Add(shardgroup2)

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	//	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})

	ctl := gomock.NewController(t)
	defer ctl.Finish()
	mockResourceManager := mock.NewMockResourceManager(ctl)
	mockResourceManager.EXPECT().UpdateShardGroup(gomock.Any()).Return(nil).MaxTimes(10)
	mockResourceManager.EXPECT().UpdateShardGroupStatus(gomock.Any()).Return(nil).MaxTimes(10)
	mockResourceManager.EXPECT().CreateRollingSet(gomock.Any(), gomock.Any()).Return(nil, nil).MaxTimes(10)

	type fields struct {
		shardGroupLister listers.ShardGroupLister
		shardGroupSynced cache.InformerSynced
		rollingSetLister listers.RollingSetLister
		rollingSetSynced cache.InformerSynced
	}
	type args struct {
		key string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "test1",
			fields: fields{
				shardGroupLister: shardGroupInformer.Lister(),
			},
			args: args{
				key: "test/sg",
			}, //default
			wantErr: false,
		},
		{
			name: "test2",
			fields: fields{
				shardGroupLister: shardGroupInformer.Lister(),
				rollingSetLister: rollingSetInformer.Lister(),
			},
			args: args{
				key: "default/test1",
			}, //default
			wantErr: false,
		},
		{
			name: "test0",
			fields: fields{
				shardGroupLister: shardGroupInformer.Lister(),
				rollingSetLister: rollingSetInformer.Lister(),
			},
			args: args{
				key: "default/test0",
			}, //default
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Controller{
				shardGroupLister: tt.fields.shardGroupLister,
				shardGroupSynced: tt.fields.shardGroupSynced,
				rollingSetLister: tt.fields.rollingSetLister,
				rollingSetSynced: tt.fields.rollingSetSynced,
			}
			c.DefaultController = controller.DefaultController{
				ResourceManager: mockResourceManager,
			}
			if err := c.Sync(tt.args.key); (err != nil) != tt.wantErr {
				t.Errorf("Controller.Sync() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

const float64EqualityThreshold = 1e-2

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) <= float64EqualityThreshold
}
