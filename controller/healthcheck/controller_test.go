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

	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	"github.com/alibaba/kube-sharding/pkg/client/clientset/versioned/fake"
	carboninformers "github.com/alibaba/kube-sharding/pkg/client/informers/externalversions"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/kubernetes/pkg/controller"
)

func newRollingset(name string, selector map[string]string) *carbonv1.RollingSet {
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
		Status: carbonv1.RollingSetStatus{},
	}
	return &rs
}

func newReplica(name string, selector map[string]string) *carbonv1.Replica {
	ip := "1.1.1.1"
	var replica = carbonv1.Replica{
		WorkerNode: carbonv1.WorkerNode{
			TypeMeta: metav1.TypeMeta{APIVersion: "carbon/v1", Kind: "Replica"},
			ObjectMeta: metav1.ObjectMeta{
				UID:         uuid.NewUUID(),
				Name:        name,
				Namespace:   metav1.NamespaceDefault,
				Annotations: make(map[string]string),
				Labels:      selector,
			},
			Spec: carbonv1.WorkerNodeSpec{
				Selector: &metav1.LabelSelector{MatchLabels: selector},
			},
			Status: carbonv1.WorkerNodeStatus{
				AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{IP: ip},
			},
		},
	}

	return &replica
}

func newWorker(name, ip string, selector map[string]string) *carbonv1.WorkerNode {
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
				IP:    ip,
			},
		},
	}
	return &worker
}

//rollingset， replica 基于标签查询下层worker
func TestController_selectorworker(t *testing.T) {
	client := fake.NewSimpleClientset([]runtime.Object{}...)
	informersFactory := carboninformers.NewSharedInformerFactory(client, controller.NoResyncPeriodFunc())
	workerLister := informersFactory.Carbon().V1().WorkerNodes().Lister()

	rollingset1 := newRollingset("rollingset-m-1",
		map[string]string{"foo": "bar",
			"rs-m-hash": "rollingset1"})
	informersFactory.Carbon().V1().RollingSets().Informer().GetIndexer().Add(rollingset1)

	worker1 := newWorker("work-m-1", "127.0.0.1",
		map[string]string{"foo": "bar", "rs-m-hash": "rollingset1", "replica-m-hash": "replica1",
			"work-m-hash": "work1"})
	worker2 := newWorker("work-m-2", "127.0.0.1",
		map[string]string{"foo": "bar", "rs-m-hash": "rollingset1", "replica-m-hash": "replica1",
			"work-m-hash": "work2"})
	worker3 := newWorker("work-m-3", "127.0.0.1",
		map[string]string{"foo": "bar", "rs-m-hash": "rollingset1", "replica-m-hash": "replica2",
			"work-m-hash": "work3"})
	informersFactory.Carbon().V1().WorkerNodes().Informer().GetIndexer().Add(worker1)
	informersFactory.Carbon().V1().WorkerNodes().Informer().GetIndexer().Add(worker2)
	informersFactory.Carbon().V1().WorkerNodes().Informer().GetIndexer().Add(worker3)
	//selector := map[string]string{"foo": "bar", "replica-m-hash": "replica1"}
	//selector := map[string]string{"foo": "bar"}
	selector := rollingset1.Spec.Selector.MatchLabels
	workers, err := workerLister.WorkerNodes(metav1.NamespaceDefault).List(labels.Set(selector).AsSelectorPreValidated())
	if err != nil {
		t.Error(err)
	}
	if len(workers) != 3 {
		t.Errorf("worker :%v", workers)
	}
}
