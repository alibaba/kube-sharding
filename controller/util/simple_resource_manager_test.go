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

package util

import (
	"context"
	"testing"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	clientset "github.com/alibaba/kube-sharding/pkg/client/clientset/versioned"
	"github.com/alibaba/kube-sharding/pkg/client/clientset/versioned/fake"
	carboninformers "github.com/alibaba/kube-sharding/pkg/client/informers/externalversions"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"
	"k8s.io/kubernetes/pkg/controller"
	k8scontroller "k8s.io/kubernetes/pkg/controller"
)

func newRollingset2(name string, selector map[string]string) *carbonv1.RollingSet {
	var rs = carbonv1.RollingSet{
		ObjectMeta: metav1.ObjectMeta{
			UID:         uuid.NewUUID(),
			Name:        name,
			Namespace:   metav1.NamespaceDefault,
			Annotations: make(map[string]string),
			Labels:      selector,
		},
		Spec: carbonv1.RollingSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: selector},
		},
		Status: carbonv1.RollingSetStatus{
			Replicas: 2,
		},
	}
	return &rs
}

func newReplica2(name string, selector map[string]string, rs *carbonv1.RollingSet) *carbonv1.Replica {
	ip := "1.1.1.1"
	var controllerKind = carbonv1.SchemeGroupVersion.WithKind("Rollingset")

	var replica = carbonv1.Replica{
		WorkerNode: carbonv1.WorkerNode{
			TypeMeta: metav1.TypeMeta{APIVersion: "carbon/v1", Kind: "Replica"},
			ObjectMeta: metav1.ObjectMeta{
				UID:             uuid.NewUUID(),
				Name:            name,
				Namespace:       metav1.NamespaceDefault,
				Annotations:     make(map[string]string),
				Labels:          selector,
				OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(rs, controllerKind)},
			},
			Spec: carbonv1.WorkerNodeSpec{
				Selector: &metav1.LabelSelector{MatchLabels: selector},
			},
			Status: carbonv1.WorkerNodeStatus{
				AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{IP: ip},
				HealthStatus:          carbonv1.HealthAlive,
			},
		},
	}

	return &replica
}

func newWorker2(name, ip string, selector map[string]string) *carbonv1.WorkerNode {
	var worker = carbonv1.WorkerNode{
		ObjectMeta: metav1.ObjectMeta{
			UID:         uuid.NewUUID(),
			Name:        name,
			Namespace:   metav1.NamespaceDefault,
			Annotations: make(map[string]string),
			Labels:      selector,
		},
		Spec: carbonv1.WorkerNodeSpec{},
		Status: carbonv1.WorkerNodeStatus{
			AllocStatus:   carbonv1.WorkerAssigned,
			ServiceStatus: carbonv1.ServicePartAvailable,
			HealthStatus:  carbonv1.HealthAlive,

			AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
				Phase: carbonv1.Running,
				IP:    ip},
		},
	}
	return &worker
}

const controllerAgentName = "test-controller"

func TestSimpleResourceManager_CreateReplica_success(t *testing.T) {
	client := fake.NewSimpleClientset([]runtime.Object{}...)
	informersFactory := carboninformers.NewSharedInformerFactory(client, controller.NoResyncPeriodFunc())
	replicaInformer := informersFactory.Carbon().V1().WorkerNodes()

	rollingset1 := newRollingset2("rollingset-m-1",
		map[string]string{"foo": "bar",
			"rs-m-hash": "rollingset1"})
	informersFactory.Carbon().V1().RollingSets().Informer().GetIndexer().Add(rollingset1)

	replica1 := newReplica2("replica-m-1",
		map[string]string{"foo": "bar", "rs-m-hash": "rollingset1",
			"replica-m-hash": "replica1"}, rollingset1)
	informersFactory.Carbon().V1().WorkerNodes().Informer().GetIndexer().Add(&replica1.WorkerNode)

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	type fields struct {
		carbonclientset   clientset.Interface
		workernodeIndexer cache.Indexer
		recorder          record.EventRecorder
		replicaInformer   cache.SharedIndexInformer
	}
	type args struct {
		rs   *carbonv1.RollingSet
		newR *carbonv1.Replica
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *carbonv1.Replica
		wantErr bool
	}{
		{
			name: "test2",
			fields: fields{
				carbonclientset:   client,
				workernodeIndexer: replicaInformer.Informer().GetIndexer(),
				replicaInformer:   replicaInformer.Informer(),
				recorder:          recorder,
			},
			args: args{rs: rollingset1, newR: replica1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &SimpleResourceManager{
				carbonclientset:   tt.fields.carbonclientset,
				workerNodeIndexer: tt.fields.workernodeIndexer,
			}
			got, err := h.CreateReplica(tt.args.rs, tt.args.newR)
			if (err != nil) != tt.wantErr {
				t.Errorf("SimpleResourceManager.CreateReplica() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got.Name != replica1.Name {
				t.Errorf("SimpleResourceManager.CreateReplica() got.Name = %v, want %v", got.Name, tt.args.rs.Name)
				return
			}

			t.Log(got)
		})
	}
}

func TestSimpleResourceManager_CreateReplica_exists(t *testing.T) {
	client := fake.NewSimpleClientset([]runtime.Object{}...)
	informersFactory := carboninformers.NewSharedInformerFactory(client, controller.NoResyncPeriodFunc())
	replicaInformer := informersFactory.Carbon().V1().WorkerNodes()

	rollingset1 := newRollingset2("rollingset-m-1",
		map[string]string{"foo": "bar",
			"rs-m-hash": "rollingset1"})
	rollingset2 := newRollingset2("rollingset-m-2",
		map[string]string{"foo": "bar",
			"rs-m-hash": "rollingset2"})
	informersFactory.Carbon().V1().RollingSets().Informer().GetIndexer().Add(rollingset1)

	replica1 := newReplica2("replica-m-1",
		map[string]string{"foo": "bar", "rs-m-hash": "rollingset1",
			"replica-m-hash": "replica1"}, rollingset2)
	informersFactory.Carbon().V1().WorkerNodes().Informer().GetIndexer().Add(&replica1.WorkerNode)

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	type fields struct {
		carbonclientset   clientset.Interface
		workernodeIndexer cache.Indexer
		recorder          record.EventRecorder
		replicaInformer   cache.SharedIndexInformer
	}
	type args struct {
		rs   *carbonv1.RollingSet
		newR *carbonv1.Replica
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *carbonv1.Replica
		wantErr bool
	}{
		{
			name: "test2",
			fields: fields{
				carbonclientset:   client,
				workernodeIndexer: replicaInformer.Informer().GetIndexer(),
				recorder:          recorder,
				replicaInformer:   replicaInformer.Informer(),
			},
			args: args{rs: rollingset1, newR: replica1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &SimpleResourceManager{
				carbonclientset:   tt.fields.carbonclientset,
				workerNodeIndexer: tt.fields.workernodeIndexer,
			}
			h.CreateReplica(tt.args.rs, tt.args.newR)

			got, err := h.CreateReplica(tt.args.rs, tt.args.newR)
			if err == nil {
				t.Errorf("SimpleResourceManager.CreateReplica() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !errors.IsAlreadyExists(err) {
				t.Errorf("SimpleResourceManager.CreateReplica() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			t.Log(got)
		})
	}
}

func TestSimpleResourceManager_CreateReplica_exists2(t *testing.T) {
	client := fake.NewSimpleClientset([]runtime.Object{}...)
	informersFactory := carboninformers.NewSharedInformerFactory(client, controller.NoResyncPeriodFunc())
	replicaInformer := informersFactory.Carbon().V1().WorkerNodes()

	rollingset1 := newRollingset2("rollingset-m-1",
		map[string]string{"foo": "bar",
			"rs-m-hash": "rollingset1"})
	informersFactory.Carbon().V1().RollingSets().Informer().GetIndexer().Add(rollingset1)

	replica1 := newReplica2("replica-m-1",
		map[string]string{"foo": "bar", "rs-m-hash": "rollingset1",
			"replica-m-hash": "replica1"}, rollingset1)
	informersFactory.Carbon().V1().WorkerNodes().Informer().GetIndexer().Add(&replica1.WorkerNode)

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	type fields struct {
		carbonclientset   clientset.Interface
		workernodeIndexer cache.Indexer
		recorder          record.EventRecorder
		replicaInformer   cache.SharedIndexInformer
	}
	type args struct {
		rs   *carbonv1.RollingSet
		newR *carbonv1.Replica
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *carbonv1.Replica
		wantErr bool
	}{
		{
			name: "test2",
			fields: fields{
				carbonclientset:   client,
				workernodeIndexer: replicaInformer.Informer().GetIndexer(),
				recorder:          recorder,
				replicaInformer:   replicaInformer.Informer(),
			},
			args: args{rs: rollingset1, newR: replica1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &SimpleResourceManager{
				carbonclientset:   tt.fields.carbonclientset,
				workerNodeIndexer: tt.fields.workernodeIndexer,
			}
			h.CreateReplica(tt.args.rs, tt.args.newR)

			got, err := h.CreateReplica(tt.args.rs, tt.args.newR)

			if (err != nil) != tt.wantErr {
				t.Errorf("SimpleResourceManager.CreateReplica() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got.Name != replica1.Name {
				t.Errorf("SimpleResourceManager.CreateReplica() got.Name = %v, want %v", got.Name, tt.args.rs.Name)
				return
			}

			t.Log(got)
		})
	}
}

func TestSimpleResourceManager_UpdateReplica(t *testing.T) {
	client := fake.NewSimpleClientset([]runtime.Object{}...)
	informersFactory := carboninformers.NewSharedInformerFactory(client, controller.NoResyncPeriodFunc())
	replicaInformer := informersFactory.Carbon().V1().WorkerNodes()
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	rollingset1 := newRollingset2("rollingset-m-1",
		map[string]string{"foo": "bar",
			"rs-m-hash": "rollingset1"})
	informersFactory.Carbon().V1().RollingSets().Informer().GetIndexer().Add(rollingset1)

	replica1 := newReplica2("replica-m-1",
		map[string]string{"foo": "bar", "rs-m-hash": "rollingset1",
			"replica-m-hash": "replica1"}, rollingset1)
	informersFactory.Carbon().V1().WorkerNodes().Informer().GetIndexer().Add(&replica1.WorkerNode)

	type fields struct {
		carbonclientset clientset.Interface
		recorder        record.EventRecorder
		replicaInformer cache.SharedIndexInformer
	}
	type args struct {
		rs *carbonv1.RollingSet
		r  *carbonv1.Replica
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
				carbonclientset: client,
				recorder:        recorder,
				replicaInformer: replicaInformer.Informer(),
			},
			args: args{rs: rollingset1, r: replica1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &SimpleResourceManager{
				carbonclientset: tt.fields.carbonclientset,
			}
			h.CreateReplica(tt.args.rs, tt.args.r.DeepCopy())
			tt.args.r.Status.HealthStatus = carbonv1.HealthLost
			if err := h.UpdateReplica(tt.args.rs, tt.args.r.DeepCopy()); err != nil {
				t.Errorf("SimpleResourceManager.UpdateReplica() error = %v, wantErr %v", err, tt.wantErr)
			}
			replica, _ := replicaInformer.Lister().WorkerNodes(metav1.NamespaceDefault).Get(tt.args.r.Name)
			if replica.Status.HealthStatus != carbonv1.HealthLost {
				t.Errorf("SimpleResourceManager.UpdateReplica() replica.Status.HealthStatus = %v, carbonv1.HealthLost", replica.Status.HealthStatus)
			}
		})
	}
}

func TestSimpleResourceManager_UpdateReplica_Notfount(t *testing.T) {
	client := fake.NewSimpleClientset([]runtime.Object{}...)
	informersFactory := carboninformers.NewSharedInformerFactory(client, controller.NoResyncPeriodFunc())
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	rollingset1 := newRollingset2("rollingset-m-1",
		map[string]string{"foo": "bar",
			"rs-m-hash": "rollingset1"})
	informersFactory.Carbon().V1().RollingSets().Informer().GetIndexer().Add(rollingset1)

	replica1 := newReplica2("replica-m-1",
		map[string]string{"foo": "bar", "rs-m-hash": "rollingset1",
			"replica-m-hash": "replica1"}, rollingset1)
	informersFactory.Carbon().V1().WorkerNodes().Informer().GetIndexer().Add(&replica1.WorkerNode)

	type fields struct {
		carbonclientset clientset.Interface
		recorder        record.EventRecorder
	}
	type args struct {
		rs *carbonv1.RollingSet
		r  *carbonv1.Replica
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
				carbonclientset: client,
				recorder:        recorder,
			},
			args: args{rs: rollingset1, r: replica1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &SimpleResourceManager{
				carbonclientset: tt.fields.carbonclientset,
			}

			if err := h.UpdateReplica(tt.args.rs, tt.args.r); !errors.IsNotFound(err) {
				t.Errorf("SimpleResourceManager.UpdateReplica() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSimpleResourceManager_ReleaseReplica(t *testing.T) {
	client := fake.NewSimpleClientset([]runtime.Object{}...)
	informersFactory := carboninformers.NewSharedInformerFactory(client, controller.NoResyncPeriodFunc())
	replicaInformer := informersFactory.Carbon().V1().WorkerNodes()
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	rollingset1 := newRollingset2("rollingset-m-1",
		map[string]string{"foo": "bar",
			"rs-m-hash": "rollingset1"})
	informersFactory.Carbon().V1().RollingSets().Informer().GetIndexer().Add(rollingset1)

	replica1 := newReplica2("replica-m-1",
		map[string]string{"foo": "bar", "rs-m-hash": "rollingset1",
			"replica-m-hash": "replica1"}, rollingset1)
	informersFactory.Carbon().V1().WorkerNodes().Informer().GetIndexer().Add(&replica1.WorkerNode)

	type fields struct {
		carbonclientset clientset.Interface
		recorder        record.EventRecorder
	}
	type args struct {
		rs *carbonv1.RollingSet
		r  *carbonv1.Replica
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
				carbonclientset: client,
				recorder:        recorder,
			},
			args: args{rs: rollingset1, r: replica1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &SimpleResourceManager{
				carbonclientset: tt.fields.carbonclientset,
			}
			h.CreateReplica(tt.args.rs, tt.args.r)
			if err := h.ReleaseReplica(tt.args.rs, tt.args.r); (err != nil) != tt.wantErr {
				t.Errorf("SimpleResourceManager.ReleaseReplica() error = %v, wantErr %v", err, tt.wantErr)
			}

			replica, _ := replicaInformer.Lister().WorkerNodes(metav1.NamespaceDefault).Get(tt.args.r.Name)
			if replica.Spec.ToDelete != true {
				t.Error("SimpleResourceManager.UpdateReplica() replica.Spec.ToDelete != true")
			}
		})
	}
}

func TestSimpleResourceManager_RemoveReplica(t *testing.T) {
	client := fake.NewSimpleClientset([]runtime.Object{}...)
	informersFactory := carboninformers.NewSharedInformerFactory(client, controller.NoResyncPeriodFunc())
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	rollingset1 := newRollingset2("rollingset-m-1",
		map[string]string{"foo": "bar",
			"rs-m-hash": "rollingset1"})
	informersFactory.Carbon().V1().RollingSets().Informer().GetIndexer().Add(rollingset1)

	replica1 := newReplica2("replica-m-1",
		map[string]string{"foo": "bar", "rs-m-hash": "rollingset1",
			"replica-m-hash": "replica1"}, rollingset1)
	informersFactory.Carbon().V1().WorkerNodes().Informer().GetIndexer().Add(&replica1.WorkerNode)

	type fields struct {
		carbonclientset clientset.Interface
		recorder        record.EventRecorder
	}
	type args struct {
		rs *carbonv1.RollingSet
		r  *carbonv1.Replica
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
				carbonclientset: client,
				recorder:        recorder,
			},
			args: args{rs: rollingset1, r: replica1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &SimpleResourceManager{
				carbonclientset: tt.fields.carbonclientset,
			}
			h.CreateReplica(tt.args.rs, tt.args.r)
			if err := h.RemoveReplica(tt.args.rs, tt.args.r); (err != nil) != tt.wantErr {
				t.Errorf("SimpleResourceManager.RemoveReplica() error = %v, wantErr %v", err, tt.wantErr)
			}
			err := h.UpdateReplica(tt.args.rs, tt.args.r)
			if err == nil {
				t.Error("SimpleResourceManager.RemoveReplica() error is nil")
			}
		})
	}
}

func TestSimpleResourceManager_UpdateRollingSet(t *testing.T) {
	client := fake.NewSimpleClientset([]runtime.Object{}...)
	informersFactory := carboninformers.NewSharedInformerFactory(client, controller.NoResyncPeriodFunc())
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	rollingset1 := newRollingset2("rollingset-m-1",
		map[string]string{"foo": "bar",
			"rs-m-hash": "rollingset1"})
	informersFactory.Carbon().V1().RollingSets().Informer().GetIndexer().Add(rollingset1)

	type fields struct {
		carbonclientset clientset.Interface
		recorder        record.EventRecorder
	}
	type args struct {
		rs *carbonv1.RollingSet
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
				carbonclientset: client,
				recorder:        recorder,
			},
			args: args{rs: rollingset1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &SimpleResourceManager{
				carbonclientset: tt.fields.carbonclientset,
			}
			_, err := h.UpdateRollingSet(tt.args.rs)

			if !errors.IsNotFound(err) {
				t.Errorf("SimpleResourceManager.UpdateRollingSet() error = %v, wantErr %v", err, tt.wantErr)

			}
		})
	}
}

func TestSimpleResourceManager_UpdateRollingSet2(t *testing.T) {
	client := fake.NewSimpleClientset([]runtime.Object{}...)
	informersFactory := carboninformers.NewSharedInformerFactory(client, controller.NoResyncPeriodFunc())
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	rollingset1 := newRollingset2("rollingset-m-1",
		map[string]string{"foo": "bar",
			"rs-m-hash": "rollingset1"})
	informersFactory.Carbon().V1().RollingSets().Informer().GetIndexer().Add(rollingset1)

	type fields struct {
		carbonclientset clientset.Interface
		recorder        record.EventRecorder
	}
	type args struct {
		rs *carbonv1.RollingSet
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
				carbonclientset: client,
				recorder:        recorder,
			},
			args: args{rs: rollingset1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &SimpleResourceManager{
				carbonclientset: tt.fields.carbonclientset,
			}
			client.CarbonV1().RollingSets(tt.args.rs.Namespace).Create(context.Background(), tt.args.rs, metav1.CreateOptions{})
			_, err := h.UpdateRollingSet(tt.args.rs)
			if errors.IsNotFound(err) {
				t.Errorf("SimpleResourceManager.UpdateRollingSet() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSimpleResourceManager_RemoveRollingSet(t *testing.T) {
	client := fake.NewSimpleClientset([]runtime.Object{}...)
	informersFactory := carboninformers.NewSharedInformerFactory(client, controller.NoResyncPeriodFunc())
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	rollingset1 := newRollingset2("rollingset-m-1",
		map[string]string{"foo": "bar",
			"rs-m-hash": "rollingset1"})
	informersFactory.Carbon().V1().RollingSets().Informer().GetIndexer().Add(rollingset1)

	type fields struct {
		carbonclientset clientset.Interface
		recorder        record.EventRecorder
	}
	type args struct {
		rs *carbonv1.RollingSet
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
				carbonclientset: client,
				recorder:        recorder,
			},
			args: args{rs: rollingset1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &SimpleResourceManager{
				carbonclientset: tt.fields.carbonclientset,
			}
			client.CarbonV1().RollingSets(tt.args.rs.Namespace).Create(context.Background(), tt.args.rs, metav1.CreateOptions{})

			if err := h.RemoveRollingSet(tt.args.rs); (err != nil) != tt.wantErr {
				t.Errorf("SimpleResourceManager.RemoveRollingSet() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSimpleResourceManager_UpdateWorkerNode(t *testing.T) {
	client := fake.NewSimpleClientset([]runtime.Object{}...)
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})
	work := newWorker2("workernode-m-1", "1.1.1.1", map[string]string{"foo": "bar",
		"work-m-hash": "workernode1"})

	type fields struct {
		carbonclientset clientset.Interface
		recorder        record.EventRecorder
	}
	type args struct {
		workernode *carbonv1.WorkerNode
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{

		{name: "test1",
			fields: fields{
				carbonclientset: client,
				recorder:        recorder,
			},
			args: args{workernode: work},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &SimpleResourceManager{
				carbonclientset: tt.fields.carbonclientset,
			}
			client.CarbonV1().WorkerNodes(metav1.NamespaceDefault).Create(context.Background(), work, metav1.CreateOptions{})
			if err := h.UpdateWorkerNode(tt.args.workernode); (err != nil) != tt.wantErr {
				t.Errorf("SimpleResourceManager.UpdateWorkerNode() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSimpleResourceManager_ListWorkerNode(t *testing.T) {
	client := fake.NewSimpleClientset([]runtime.Object{}...)
	informersFactory := carboninformers.NewSharedInformerFactory(client, controller.NoResyncPeriodFunc())

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})
	worker1 := newWorker2("work-m-1", "127.0.0.1",
		map[string]string{"foo": "bar", "replica-m-hash": "replica1", carbonv1.DefaultRollingsetUniqueLabelKey: "rollingset1",
			"work-m-hash": "work1"})
	worker2 := newWorker2("work-m-2", "127.0.0.1",
		map[string]string{"foo": "bar", "replica-m-hash": "replica1", carbonv1.DefaultRollingsetUniqueLabelKey: "rollingset1",
			"work-m-hash": "work2"})
	worker3 := newWorker2("work-m-3", "127.0.0.1",
		map[string]string{"foo": "bar", "replica-m-hash": "replica2", carbonv1.DefaultRollingsetUniqueLabelKey: "rollingset2",
			"work-m-hash": "work3"})

	(&SimpleResourceManager{}).addWorkerNodeIndex(informersFactory.Carbon().V1().WorkerNodes().Informer().GetIndexer())
	informersFactory.Carbon().V1().WorkerNodes().Informer().GetIndexer().Add(worker1)
	informersFactory.Carbon().V1().WorkerNodes().Informer().GetIndexer().Add(worker2)
	informersFactory.Carbon().V1().WorkerNodes().Informer().GetIndexer().Add(worker3)

	type fields struct {
		carbonclientset   clientset.Interface
		workernodeIndexer cache.Indexer
		recorder          record.EventRecorder
	}
	type args struct {
		namespace string
		selector  map[string]string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		count   int
		wantErr bool
	}{
		{
			name: "match nodes",
			fields: fields{
				carbonclientset:   client,
				workernodeIndexer: informersFactory.Carbon().V1().WorkerNodes().Informer().GetIndexer(),
				recorder:          recorder,
			},
			args:  args{namespace: metav1.NamespaceDefault, selector: map[string]string{"foo": "bar", carbonv1.DefaultRollingsetUniqueLabelKey: "rollingset1"}},
			count: 2,
		},
		{
			name: "no match node",
			fields: fields{
				carbonclientset:   client,
				workernodeIndexer: informersFactory.Carbon().V1().WorkerNodes().Informer().GetIndexer(),
				recorder:          recorder,
			},
			args:  args{namespace: metav1.NamespaceDefault, selector: map[string]string{carbonv1.DefaultRollingsetUniqueLabelKey: "rollingset3"}},
			count: 0,
		},
		{
			name: "invalid selector",
			fields: fields{
				carbonclientset:   client,
				workernodeIndexer: informersFactory.Carbon().V1().WorkerNodes().Informer().GetIndexer(),
				recorder:          recorder,
			},
			args:    args{namespace: metav1.NamespaceDefault, selector: map[string]string{"foo": "bar"}},
			count:   0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &SimpleResourceManager{
				carbonclientset:   tt.fields.carbonclientset,
				workerNodeIndexer: tt.fields.workernodeIndexer,
			}
			gotRet, err := h.ListWorkerNodeForRS(tt.args.selector)
			if (err != nil) != tt.wantErr {
				t.Errorf("SimpleResourceManager.ListWorkerNode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(gotRet) != tt.count {
				t.Errorf("SimpleResourceManager.ListWorkerNode() len(gotRet) = %v, tt.count %v", len(gotRet), tt.count)
				return
			}

		})
	}
}

func TestSimpleResourceManager_BatchCreateReplica(t *testing.T) {
	client := fake.NewSimpleClientset([]runtime.Object{}...)
	informersFactory := carboninformers.NewSharedInformerFactory(client, controller.NoResyncPeriodFunc())

	rollingset1 := newRollingset2("rollingset-m-1",
		map[string]string{"foo": "bar",
			"rs-m-hash": "rollingset1"})
	informersFactory.Carbon().V1().RollingSets().Informer().GetIndexer().Add(rollingset1)

	replica1 := newReplica2("replica-m-1",
		map[string]string{"foo": "bar", "rs-m-hash": "rollingset1",
			"replica-m-hash": "replica1"}, rollingset1)
	informersFactory.Carbon().V1().WorkerNodes().Informer().GetIndexer().Add(&replica1.WorkerNode)
	replica2 := newReplica2("replica-m-2",
		map[string]string{"foo": "bar", "rs-m-hash": "rollingset1",
			"replica-m-hash": "replica2"}, rollingset1)
	informersFactory.Carbon().V1().WorkerNodes().Informer().GetIndexer().Add(replica2)

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	type fields struct {
		kubeclientset   kubernetes.Interface
		carbonclientset clientset.Interface
		recorder        record.EventRecorder
		executor        *utils.AsyncExecutor
	}
	type args struct {
		rs    *carbonv1.RollingSet
		rList []*carbonv1.Replica
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "test2",
			fields: fields{
				carbonclientset: client,
				recorder:        recorder,
				executor:        utils.NewExecutor(10),
			},
			args: args{rs: rollingset1, rList: []*carbonv1.Replica{replica1, replica2}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &SimpleResourceManager{
				kubeclientset:   tt.fields.kubeclientset,
				carbonclientset: tt.fields.carbonclientset,
				executor:        tt.fields.executor,
			}
			if _, err := a.BatchCreateReplica(tt.args.rs, tt.args.rList); (err != nil) != tt.wantErr {
				t.Errorf("SimpleResourceManager.BatchCreateReplica() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSimpleResourceManager_aggregateGangStatus(t *testing.T) {
	type fields struct {
		kubeclientset     kubernetes.Interface
		carbonclientset   clientset.Interface
		workerNodeIndexer cache.Indexer
		podIndexer        cache.Indexer
		executor          *utils.AsyncExecutor
		expectations      *k8scontroller.UIDTrackingControllerExpectations
	}
	type args struct {
		replica *carbonv1.Replica
	}
	tests := []struct {
		name       string
		fields     fields
		args       args
		want       *carbonv1.Replica
		wantSynced bool
	}{
		{
			name: "agg",
			args: args{
				replica: &carbonv1.Replica{
					WorkerNode: carbonv1.WorkerNode{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{
								carbonv1.BizDetailKeyGangInfo: `{"role1":{"name":"role1","ip":"1.1"},"role2":{"name":"role2","ip":"1.2"},"role0":{"name":"role0","ip":"1.0"}}`,
							},
						},
						Spec: carbonv1.WorkerNodeSpec{
							VersionPlan: carbonv1.VersionPlan{
								SignedVersionPlan: carbonv1.SignedVersionPlan{
									Template: &carbonv1.HippoPodTemplate{
										ObjectMeta: metav1.ObjectMeta{
											Labels: map[string]string{
												carbonv1.LabelGangMainPart: "true",
											},
										},
									},
								},
							},
						},
					},
					Gang: []carbonv1.Replica{
						{
							WorkerNode: carbonv1.WorkerNode{
								ObjectMeta: metav1.ObjectMeta{
									Labels: map[string]string{
										carbonv1.LabelKeyGroupName: "group",
										carbonv1.LabelKeyRoleName:  "group.role1",
									},
									Annotations: map[string]string{
										carbonv1.BizDetailKeyGangInfo: `{"role1":{"name":"role1","ip":"1.1"},"role2":{"name":"role2","ip":"1.2"},"role0":{"name":"role0","ip":"1.0"}}`,
									},
								},
								Status: carbonv1.WorkerNodeStatus{
									Score: 101,
									AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
										IP:            "1.1",
										EntityAlloced: true,
										Phase:         carbonv1.Running,
									},
								},
							},
						},
						{
							WorkerNode: carbonv1.WorkerNode{
								ObjectMeta: metav1.ObjectMeta{
									Labels: map[string]string{
										carbonv1.LabelKeyGroupName: "group",
										carbonv1.LabelKeyRoleName:  "group.role2",
									},
									Annotations: map[string]string{
										carbonv1.BizDetailKeyGangInfo: `{"role1":{"name":"role1","ip":"1.1"},"role2":{"name":"role2","ip":"1.2"},"role0":{"name":"role0","ip":"1.0"}}`,
									},
								},
								Status: carbonv1.WorkerNodeStatus{
									Score: 102,
									AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
										IP:            "1.2",
										EntityAlloced: true,
										Phase:         carbonv1.Terminated,
									},
								},
							},
						},
						{
							WorkerNode: carbonv1.WorkerNode{
								ObjectMeta: metav1.ObjectMeta{
									Labels: map[string]string{
										carbonv1.LabelKeyGroupName: "group",
										carbonv1.LabelKeyRoleName:  "group.role0",
									},
									Annotations: map[string]string{
										carbonv1.BizDetailKeyGangInfo: `{"role1":{"name":"role1","ip":"1.1"},"role2":{"name":"role2","ip":"1.2"},"role0":{"name":"role0","ip":"1.0"}}`,
									},
								},
								Status: carbonv1.WorkerNodeStatus{
									Score: 100,
									AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
										IP:            "1.0",
										EntityAlloced: true,
										Phase:         carbonv1.Failed,
									},
								},
							},
						},
					},
				},
			},
			want: &carbonv1.Replica{
				WorkerNode: carbonv1.WorkerNode{
					Status: carbonv1.WorkerNodeStatus{
						Score: 100,
						AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
							IP:            "1.0",
							EntityAlloced: true,
							Phase:         carbonv1.Failed,
						},
					},
				},
			},
			wantSynced: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &SimpleResourceManager{
				kubeclientset:     tt.fields.kubeclientset,
				carbonclientset:   tt.fields.carbonclientset,
				workerNodeIndexer: tt.fields.workerNodeIndexer,
				podIndexer:        tt.fields.podIndexer,
				executor:          tt.fields.executor,
				expectations:      tt.fields.expectations,
			}
			if got := a.aggregateGangStatus(tt.args.replica); got != tt.wantSynced {
				t.Errorf("SimpleResourceManager.aggregateGangStatus() = %v, want %v", got, tt.want)
			}
			if utils.ObjJSON(tt.args.replica.Status) != utils.ObjJSON(tt.want.Status) {
				t.Errorf("SimpleResourceManager.aggregateGangStatus() = %v, want %v", utils.ObjJSON(tt.args.replica.Status), utils.ObjJSON(tt.want.Status))
			}
		})
	}
}
