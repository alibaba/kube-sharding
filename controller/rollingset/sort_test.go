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
	"sort"
	"testing"

	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
)

func TestByUnAssignedSlot(t *testing.T) {
	replicas := make([]*carbonv1.Replica, 0)
	replica1 := newReplica("a", 1, 1)
	replica1.Status.AllocStatus = carbonv1.WorkerUnAssigned
	replicas = append(replicas, replica1)
	replica2 := newReplica("b", 2, 1)
	replica2.Status.AllocStatus = carbonv1.WorkerAssigned
	replicas = append(replicas, replica2)
	replica3 := newReplica("c", 3, 1)
	replica3.Status.AllocStatus = carbonv1.WorkerUnAssigned
	replicas = append(replicas, replica3)
	sort.Sort(ByUnAssignedSlot(replicas))
	if replicas[0].Name != "b" {
		t.Errorf(" error %v", replicas[0].Name)
	}

	if replicas[1].Name != "a" {
		t.Errorf(" error %v", replicas[1].Name)

	}
	if replicas[2].Name != "c" {
		t.Errorf(" error %v", replicas[2].Name)

	}
}

func TestByScore(t *testing.T) {
	replicas := make([]*carbonv1.Replica, 0)
	replica1 := newReplica("a", 10, 1)
	replica1.Status.AllocStatus = carbonv1.WorkerUnAssigned
	replica1.Status.ServiceStatus = carbonv1.ServiceUnAvailable
	replicas = append(replicas, replica1)
	replica2 := newReplica("b", 2, 1)
	replica2.Status.AllocStatus = carbonv1.WorkerAssigned
	replica2.Status.ServiceStatus = carbonv1.ServiceAvailable
	replicas = append(replicas, replica2)
	replica3 := newReplica("c", 3, 1)
	replica3.Status.AllocStatus = carbonv1.WorkerUnAssigned
	replica3.Status.ServiceStatus = carbonv1.ServiceUnAvailable
	replicas = append(replicas, replica3)
	sort.Sort(ByScore(replicas))
	if replicas[0].Name != "b" {
		t.Errorf(" error %v", replicas[0].Name)
	}

	if replicas[1].Name != "c" {
		t.Errorf(" error %v", replicas[1].Name)

	}
	if replicas[2].Name != "a" {
		t.Errorf(" error %v", replicas[2].Name)

	}
}

func TestByUnAssignedSlotAndName(t *testing.T) {
	replicas := make([]*carbonv1.Replica, 0)
	replica1 := newReplica("a", 1, 1)
	replica1.Status.AllocStatus = carbonv1.WorkerUnAssigned
	replica1.Status.Complete = false
	replicas = append(replicas, replica1)
	replica2 := newReplica("b", 2, 1)
	replica2.Status.AllocStatus = carbonv1.WorkerAssigned
	replica2.Status.Complete = true
	replicas = append(replicas, replica2)
	replica3 := newReplica("c", 3, 1)
	replica3.Status.AllocStatus = carbonv1.WorkerUnAssigned
	replica3.Status.Complete = false
	replicas = append(replicas, replica3)
	sort.Sort(ByAvailableAndName(replicas))
	if replicas[0].Name != "c" {
		t.Errorf(" error %v", replicas[0].Name)
	}

	if replicas[1].Name != "a" {
		t.Errorf(" error %v", replicas[1].Name)

	}
	if replicas[2].Name != "b" {
		t.Errorf(" error %v", replicas[2].Name)

	}
}

func TestByUnAssignedSlotAndName2(t *testing.T) {
	replicas := make([]*carbonv1.Replica, 0)
	replica1 := newReplica("a", 1, 1)
	replica1.Status.AllocStatus = carbonv1.WorkerAssigned
	replica1.Status.Complete = false
	replicas = append(replicas, replica1)
	replica2 := newReplica("b", 2, 1)
	replica2.Status.AllocStatus = carbonv1.WorkerAssigned
	replica2.Status.ServiceStatus = carbonv1.ServiceAvailable
	replica2.Status.Complete = true
	replica2.Spec.Version = "a"
	replica2.Status.Version = "a"
	replicas = append(replicas, replica2)
	replica3 := newReplica("c", 3, 1)
	replica3.Status.AllocStatus = carbonv1.WorkerAssigned
	replica3.Status.Complete = false
	replicas = append(replicas, replica3)
	sort.Sort(ByAvailableAndName(replicas))
	if replicas[0].Name != "c" {
		t.Errorf(" error %v", replicas[0].Name)
	}

	if replicas[1].Name != "a" {
		t.Errorf(" error %v", replicas[1].Name)

	}
	if replicas[2].Name != "b" {
		t.Errorf(" error %v", replicas[2].Name)

	}
}

func TestByRedudant(t *testing.T) {
	replicas := make([]*carbonv1.Replica, 0)
	replica2 := newReplica("b", 2, 2)
	replica2.Status.AllocStatus = carbonv1.WorkerAssigned
	replica2.Status.Complete = true
	replicas = append(replicas, replica2)
	replica3 := newReplica("c", 3, 10001)
	replica3.Status.AllocStatus = carbonv1.WorkerUnAssigned
	replica3.Status.Complete = false
	replicas = append(replicas, replica3)
	replica4 := newReplica("d", 3, 2)
	replica4.Status.AllocStatus = carbonv1.WorkerAssigned
	replica4.Spec.WorkerMode = carbonv1.WorkerModeTypeHot
	replica4.Status.Complete = true
	replicas = append(replicas, replica4)
	replica6 := newReplica("f", 3, -100)
	replica6.Status.AllocStatus = carbonv1.WorkerUnAssigned
	replica6.Status.Complete = false
	replicas = append(replicas, replica6)
	replica7 := newReplica("g", 3, 2)
	replica7.Status.AllocStatus = carbonv1.WorkerAssigned
	replica7.Spec.WorkerMode = carbonv1.WorkerModeTypeWarm
	replica7.Status.Complete = true
	replicas = append(replicas, replica7)
	sort.Sort(RedundantSort(replicas))
	if replicas[0].Name != "b" {
		t.Errorf(" error %v", replicas[0].Name)
	}
	if replicas[1].Name != "d" {
		t.Errorf(" error %v", replicas[1].Name)
	}
	if replicas[2].Name != "g" {
		t.Errorf(" error %v", replicas[2].Name)
	}
	if replicas[3].Name != "c" {
		t.Errorf(" error %v", replicas[3].Name)
	}
	if replicas[4].Name != "f" {
		t.Errorf(" error %v", replicas[4].Name)
	}
}

func newReplica(name string, score int, deletionCost int64) *carbonv1.Replica {
	replica := &carbonv1.Replica{
		WorkerNode: carbonv1.WorkerNode{
			TypeMeta: metav1.TypeMeta{Kind: "ReplicaSet"},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				UID:       uuid.NewUUID(),
				Namespace: metav1.NamespaceDefault,
			},
			Spec: carbonv1.WorkerNodeSpec{
				Version: name,
			},
			Status: carbonv1.WorkerNodeStatus{
				Complete: true,
				Score:    int64(score),
				AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
					DeletionCost: deletionCost,
				},
			},
		},
	}
	return replica
}
