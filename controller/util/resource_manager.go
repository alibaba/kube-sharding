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
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// ResourceManager is interface to manage resource
type ResourceManager interface {
	CreatePod(pod *corev1.Pod) (*corev1.Pod, error)
	UpdatePodSpec(pod *corev1.Pod) error
	UpdatePodStatus(pod *corev1.Pod) error
	DeletePod(pod *corev1.Pod, grace bool) error
	PatchPod(pod *corev1.Pod, pt types.PatchType, data []byte, subresource []string) error

	CreateWorkerNode(worker *carbonv1.WorkerNode) (*carbonv1.WorkerNode, error)
	DeleteWorkerNode(worker *carbonv1.WorkerNode) error
	ReleaseWorkerNode(worker *carbonv1.WorkerNode) error
	UpdateWorkerStatus(worker *carbonv1.WorkerNode) error
	UpdateWorkerNode(wn *carbonv1.WorkerNode) error
	ListWorkerNodeForRS(selector map[string]string) ([]*carbonv1.WorkerNode, error)
	ListWorkerNodeByOwner(selector map[string]string, ownerKey string) ([]*carbonv1.WorkerNode, error)

	CreateServicePublisher(p *carbonv1.ServicePublisher) error
	DeleteServicePublisher(p *carbonv1.ServicePublisher) error
	UpdateServicePublisher(p *carbonv1.ServicePublisher) error
	UpdateServicePublisherStatus(p *carbonv1.ServicePublisher) error
	DeleteServicePublisherForRs(rs *carbonv1.RollingSet) error

	//CreateReplica(rs *carbonv1.RollingSet, r *carbonv1.Replica) (*carbonv1.Replica, error)
	UpdateReplica(rs *carbonv1.RollingSet, r *carbonv1.Replica) error
	ReleaseReplica(rs *carbonv1.RollingSet, r *carbonv1.Replica) error
	RemoveReplica(rs *carbonv1.RollingSet, r *carbonv1.Replica) error

	BatchCreateReplica(rs *carbonv1.RollingSet, rList []*carbonv1.Replica) (int, error)
	BatchUpdateReplica(rs *carbonv1.RollingSet, rList []*carbonv1.Replica) (int, error)
	BatchReleaseReplica(rs *carbonv1.RollingSet, rList []*carbonv1.Replica) (int, error)
	BatchRemoveReplica(rs *carbonv1.RollingSet, rList []*carbonv1.Replica) (int, error)

	UpdateReplicaStatus(r *carbonv1.Replica) error
	UpdateRollingSet(rs *carbonv1.RollingSet) (*carbonv1.RollingSet, error)
	UpdateRollingSetStatus(rs *carbonv1.RollingSet) error
	RemoveRollingSet(rs *carbonv1.RollingSet) error
	ListReplicaForRS(rs *carbonv1.RollingSet) ([]*carbonv1.Replica, error)

	CreateRollingSet(sg *carbonv1.ShardGroup, rs *carbonv1.RollingSet) (*carbonv1.RollingSet, error)
	UpdateShardGroup(sg *carbonv1.ShardGroup) error
	UpdateShardGroupStatus(sg *carbonv1.ShardGroup) error
	DeleteShardGroup(sg *carbonv1.ShardGroup) error

	BatchDoWorkerNodes(
		workersToCreate []*carbonv1.WorkerNode,
		workersToUpdate []*carbonv1.WorkerNode,
		workersToRelease []*carbonv1.WorkerNode,
	) (int, error, int, error, int, error)
}
