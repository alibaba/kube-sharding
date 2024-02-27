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
	"reflect"
	"testing"

	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	"github.com/alibaba/kube-sharding/pkg/client/clientset/versioned/fake"
	carboninformers "github.com/alibaba/kube-sharding/pkg/client/informers/externalversions"
	listers "github.com/alibaba/kube-sharding/pkg/client/listers/carbon/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/kubernetes/pkg/controller"
	labelsutil "k8s.io/kubernetes/pkg/util/labels"
)

func newReplica(name string, selector map[string]string) (*carbonv1.Replica, []*carbonv1.WorkerNode) {
	ip := "1.1.1.1"
	var currWorker = newWorker(name+"-curr", ip, selector)
	var backupWorker = newWorker(name+"-backup", ip+"1", selector)
	var replica = carbonv1.Replica{
		WorkerNode: carbonv1.WorkerNode{
			TypeMeta: metav1.TypeMeta{APIVersion: "carbon/v1", Kind: "Replica"},
			ObjectMeta: metav1.ObjectMeta{
				UID:         uuid.NewUUID(),
				Name:        name,
				Namespace:   metav1.NamespaceDefault,
				Annotations: make(map[string]string),
			},
			Spec: carbonv1.WorkerNodeSpec{
				Selector: &metav1.LabelSelector{MatchLabels: selector},
			},
			Status: carbonv1.WorkerNodeStatus{
				AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{IP: ip},
			},
		},
	}
	var workers = []*carbonv1.WorkerNode{currWorker, backupWorker}
	return &replica, workers
}

func newWorker(name, ip string, selector map[string]string) *carbonv1.WorkerNode {
	var worker = carbonv1.WorkerNode{
		ObjectMeta: metav1.ObjectMeta{
			UID:         uuid.NewUUID(),
			Name:        name,
			Namespace:   metav1.NamespaceDefault,
			Annotations: make(map[string]string),
		},
		Spec: carbonv1.WorkerNodeSpec{},
		Status: carbonv1.WorkerNodeStatus{
			AllocStatus:   carbonv1.WorkerAssigned,
			ServiceStatus: carbonv1.ServicePartAvailable,
			HealthStatus:  carbonv1.HealthAlive,
			AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
				Phase: carbonv1.Running,
				IP:    ip,
			},
		},
	}
	return &worker
}

func newService(name string, selector map[string]string) *carbonv1.ServicePublisher {
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

func TestGetWorkerServices(t *testing.T) {
	type args struct {
		s      listers.ServicePublisherLister
		worker *carbonv1.WorkerNode
	}

	client := fake.NewSimpleClientset([]runtime.Object{}...)
	informersFactory := carboninformers.NewSharedInformerFactory(client, controller.NoResyncPeriodFunc())
	serviceLister := informersFactory.Carbon().V1().ServicePublishers().Lister()

	service := newService("test", map[string]string{"foo": "bar"})
	serviceOther := newService("test1", map[string]string{"foo": "bars"})
	informersFactory.Carbon().V1().ServicePublishers().Informer().GetIndexer().Add(service)
	informersFactory.Carbon().V1().ServicePublishers().Informer().GetIndexer().Add(serviceOther)

	worker := newWorker("test", "1.1.1.1", nil)
	worker.Labels = labelsutil.CloneAndAddLabel(nil, "foo", "bar")

	tests := []struct {
		name    string
		args    args
		want    []*carbonv1.ServicePublisher
		wantErr bool
	}{
		{
			name:    "TestGetService",
			args:    args{s: serviceLister, worker: worker},
			want:    []*carbonv1.ServicePublisher{service},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetWorkerServices(tt.args.s, tt.args.worker)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetWorkerServices() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetWorkerServices() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetWorkerServiceMemberships(t *testing.T) {
	type args struct {
		s      listers.ServicePublisherLister
		worker *carbonv1.WorkerNode
	}

	client := fake.NewSimpleClientset([]runtime.Object{}...)
	informersFactory := carboninformers.NewSharedInformerFactory(client, controller.NoResyncPeriodFunc())
	serviceLister := informersFactory.Carbon().V1().ServicePublishers().Lister()

	service := newService("test", map[string]string{"foo": "bar"})
	serviceOther := newService("test1", map[string]string{"foo": "bars"})
	informersFactory.Carbon().V1().ServicePublishers().Informer().GetIndexer().Add(service)
	informersFactory.Carbon().V1().ServicePublishers().Informer().GetIndexer().Add(serviceOther)

	worker := newWorker("test", "1.1.1.1", nil)
	worker.Labels = labelsutil.CloneAndAddLabel(nil, "foo", "bar")

	var want = sets.String{}
	want.Insert("default/test")
	tests := []struct {
		name    string
		args    args
		want    sets.String
		wantErr bool
	}{
		{
			name:    "TestGetService",
			args:    args{s: serviceLister, worker: worker},
			want:    want,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetWorkerServiceMemberships(tt.args.s, tt.args.worker)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetWorkerServiceMemberships() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetWorkerServiceMemberships() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsObjDelete(t *testing.T) {
	type args struct {
		obj interface{}
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsObjDelete(tt.args.obj); got != tt.want {
				t.Errorf("IsObjDelete() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_mergeLabels(t *testing.T) {
	type args struct {
		labels  map[string]string
		labels2 map[string]string
	}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		{
			name: "test2",
			args: args{labels: map[string]string{"foo": "bar",
				"rs-m-hash": "rollingset1"}, labels2: map[string]string{"foo": "bar",
				"rs-m-hash": "rollingset2"}},
			want: map[string]string{"foo": "bar",
				"rs-m-hash": "rollingset2"},
		},
		{
			name: "test2",
			args: args{labels: map[string]string{"foo": "bar",
				"rs-m-hash": "rollingset1"}, labels2: map[string]string{"foo": "bar",
				"rs-a-hash": "rollingset2"}},
			want: map[string]string{"foo": "bar",
				"rs-m-hash": "rollingset1", "rs-a-hash": "rollingset2"},
		},
		{
			name: "test2",
			args: args{labels2: map[string]string{"foo": "bar",
				"rs-a-hash": "rollingset2"}},
			want: map[string]string{"foo": "bar",
				"rs-a-hash": "rollingset2"},
		},
		{
			name: "test2",
			args: args{labels: map[string]string{"foo": "bar",
				"rs-a-hash": "rollingset2"}},
			want: map[string]string{"foo": "bar",
				"rs-a-hash": "rollingset2"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MergeLabels(tt.args.labels, tt.args.labels2); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mergeLabels() = %v, want %v", got, tt.want)
			}

		})
	}
}

func newRollingSet(version string) *carbonv1.RollingSet {

	rollingset := carbonv1.RollingSet{
		TypeMeta: metav1.TypeMeta{APIVersion: "carbon/v1", Kind: "Rollingset"},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "rollingset-m",
			Namespace:   metav1.NamespaceDefault,
			Annotations: make(map[string]string),
		},
		Spec: carbonv1.RollingSetSpec{
			Version: version,
		},
		Status: carbonv1.RollingSetStatus{},
	}
	return &rollingset
}

/*
func TestUpdateCrdTime(t *testing.T) {
	type args struct {
		meta *metav1.ObjectMeta
		now  time.Time
	}
	rs := newRollingSet("test")
	rs.ObjectMeta.Annotations = nil

	now := time.Now()

	tests := []struct {
		name string
		args args
	}{
		{
			name: "test",
			args: args{meta: &rs.ObjectMeta, now: now},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			UpdateCrdTime(tt.args.meta, tt.args.now)
			if tt.args.meta.Annotations["updateCrdTime"] == "" {
				t.Errorf("annotations=%v", tt.args.meta.Annotations)
			}
			if rs.Annotations["updateCrdTime"] == "" {
				t.Errorf("annotations=%v", rs.Annotations)
			}

		})
	}
}

func TestCalcCrdChangeLatency(t *testing.T) {
	type args struct {
		annotations map[string]string
	}
	annotations := make(map[string]string)
	now := time.Now()
	annotations["updateCrdTime"] = now.Format(time.RFC3339Nano)
	tests := []struct {
		name string
		args args
		want float64
	}{
		{
			name: "test",
			args: args{annotations: annotations},
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CalcCrdChangeLatency(tt.args.annotations); got != tt.want && got != tt.want+1 {
				t.Errorf("CalcCrdChangeLatency() = %v, want %v", got, tt.want)
			}
		})
	}
}

/*

// GetReplicaRefers get the replicas which ref the replica
func GetReplicaRefers(s listers.ReplicaLister, replica *carbonv1.Replica) ([]*carbonv1.Replica, error) {
	if nil == s {
		return nil, errNilLister
	}
	allReplicas, err := s.Replicas(replica.Namespace).List(labels.Everything())
	if err != nil {
		return nil, err
	}

	var replicas []*carbonv1.Replica
	for i := range allReplicas {
		treplica := allReplicas[i]
		if treplica.Spec.Selector == nil {
			continue
		}
		selector, _ := metav1.LabelSelectorAsSelector(treplica.Spec.Selector)
		if selector.Matches(labels.Set(replica.Labels)) && treplica.Name != replica.Name {
			replicas = append(replicas, treplica)
		}
	}

	return replicas, nil
}


func GetReplicaPods(carbonclientset clientset.Interface, replica *carbonv1.Replica,
	podLister corelisters.PodLister) ([]*corev1.Pod, error) {

	selector, err := metav1.LabelSelectorAsSelector(replica.Spec.Selector)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("Error converting pod selector to selector: %v", err))
		return nil, err
	}

	// list all pods to include the pods that don't match the rs`s selector
	// anymore but has the stale controller ref.
	// TODO: Do the List and Filter in a single pass, or use an index.
	allPods, err := podLister.Pods(replica.Namespace).List(labels.Everything())
	if err != nil {
		return nil, err
	}

	// NOTE: filteredPods are pointing to objects from cache - if you need to
	// modify them, you need to copy it first.
	filteredPods, err := ClaimPods(replica, selector, allPods)
	if err != nil {
		return nil, err
	}
	return allPods, nil
}


func TestGetReplicaRefers(t *testing.T) {
	type args struct {
		s       listers.ReplicaLister
		replica *carbonv1.Replica
	}

	client := fake.NewSimpleClientset([]runtime.Object{}...)
	informersFactory := carboninformers.NewSharedInformerFactory(client, controller.NoResyncPeriodFunc())
	replicaLister := informersFactory.Carbon().V1().Replicas().Lister()

	informersFactoryWithref := carboninformers.NewSharedInformerFactory(client, controller.NoResyncPeriodFunc())
	replicaListerWithref := informersFactoryWithref.Carbon().V1().Replicas().Lister()
	r, _ := newReplica("test1", map[string]string{"foo": "bar"})
	informersFactoryWithref.Carbon().V1().Replicas().Informer().GetIndexer().Add(r)

	informersFactoryWithrefSelf := carboninformers.NewSharedInformerFactory(client, controller.NoResyncPeriodFunc())
	replicaListerWithrefSelf := informersFactoryWithrefSelf.Carbon().V1().Replicas().Lister()
	r1, _ := newReplica("test", map[string]string{"foo": "bar"})
	informersFactoryWithrefSelf.Carbon().V1().Replicas().Informer().GetIndexer().Add(r1)

	replica, _ := newReplica("test", map[string]string{"foo": "bar"})
	replica.Labels = labelsutil.CloneAndAddLabel(nil, "foo", "bar")
	replica1, _ := newReplica("test", map[string]string{"foo": "bar"})

	tests := []struct {
		name    string
		args    args
		want    []*carbonv1.Replica
		wantErr bool
	}{
		{
			name:    "TestNil",
			args:    args{replica: replica, s: replicaLister},
			want:    nil,
			wantErr: false,
		},
		{
			name:    "TestWithRef",
			args:    args{replica: replica, s: replicaListerWithref},
			want:    []*carbonv1.Replica{r},
			wantErr: false,
		},
		{
			name:    "TestWithRefself",
			args:    args{replica: replica, s: replicaListerWithrefSelf},
			want:    nil,
			wantErr: false,
		},
		{
			name:    "TestWithWrongSelector",
			args:    args{replica: replica1, s: replicaListerWithrefSelf},
			want:    nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetReplicaRefers(tt.args.s, tt.args.replica)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetReplicaRefers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetReplicaRefers() = %v, want %v", got, tt.want)
			}
		})
	}
}

*/

func Test_map(t *testing.T) {
	map1 := map[string]string{"a": "a1", "b": "b1", "c": "c1"}
	map2 := map[string]string{"b": "b1", "a": "a1", "c": "c1"}

	if !reflect.DeepEqual(map1, map2) {
		t.Errorf("map1=%v, map2=%v", map1, map2)
	}
}
