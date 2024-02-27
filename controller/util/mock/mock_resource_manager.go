// Code generated by MockGen. DO NOT EDIT.
// Source: controller/util/resource_manager.go

// Package mock is a generated GoMock package.
package mock

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	v1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	v10 "k8s.io/api/core/v1"
	types "k8s.io/apimachinery/pkg/types"
)

// MockResourceManager is a mock of ResourceManager interface.
type MockResourceManager struct {
	ctrl     *gomock.Controller
	recorder *MockResourceManagerMockRecorder
}

// MockResourceManagerMockRecorder is the mock recorder for MockResourceManager.
type MockResourceManagerMockRecorder struct {
	mock *MockResourceManager
}

// NewMockResourceManager creates a new mock instance.
func NewMockResourceManager(ctrl *gomock.Controller) *MockResourceManager {
	mock := &MockResourceManager{ctrl: ctrl}
	mock.recorder = &MockResourceManagerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockResourceManager) EXPECT() *MockResourceManagerMockRecorder {
	return m.recorder
}

// BatchCreateReplica mocks base method.
func (m *MockResourceManager) BatchCreateReplica(rs *v1.RollingSet, rList []*v1.Replica) (int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BatchCreateReplica", rs, rList)
	ret0, _ := ret[0].(int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// BatchCreateReplica indicates an expected call of BatchCreateReplica.
func (mr *MockResourceManagerMockRecorder) BatchCreateReplica(rs, rList interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BatchCreateReplica", reflect.TypeOf((*MockResourceManager)(nil).BatchCreateReplica), rs, rList)
}

// BatchDoWorkerNodes mocks base method.
func (m *MockResourceManager) BatchDoWorkerNodes(workersToCreate, workersToUpdate, workersToRelease []*v1.WorkerNode) (int, error, int, error, int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BatchDoWorkerNodes", workersToCreate, workersToUpdate, workersToRelease)
	ret0, _ := ret[0].(int)
	ret1, _ := ret[1].(error)
	ret2, _ := ret[2].(int)
	ret3, _ := ret[3].(error)
	ret4, _ := ret[4].(int)
	ret5, _ := ret[5].(error)
	return ret0, ret1, ret2, ret3, ret4, ret5
}

// BatchDoWorkerNodes indicates an expected call of BatchDoWorkerNodes.
func (mr *MockResourceManagerMockRecorder) BatchDoWorkerNodes(workersToCreate, workersToUpdate, workersToRelease interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BatchDoWorkerNodes", reflect.TypeOf((*MockResourceManager)(nil).BatchDoWorkerNodes), workersToCreate, workersToUpdate, workersToRelease)
}

// BatchReleaseReplica mocks base method.
func (m *MockResourceManager) BatchReleaseReplica(rs *v1.RollingSet, rList []*v1.Replica) (int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BatchReleaseReplica", rs, rList)
	ret0, _ := ret[0].(int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// BatchReleaseReplica indicates an expected call of BatchReleaseReplica.
func (mr *MockResourceManagerMockRecorder) BatchReleaseReplica(rs, rList interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BatchReleaseReplica", reflect.TypeOf((*MockResourceManager)(nil).BatchReleaseReplica), rs, rList)
}

// BatchRemoveReplica mocks base method.
func (m *MockResourceManager) BatchRemoveReplica(rs *v1.RollingSet, rList []*v1.Replica) (int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BatchRemoveReplica", rs, rList)
	ret0, _ := ret[0].(int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// BatchRemoveReplica indicates an expected call of BatchRemoveReplica.
func (mr *MockResourceManagerMockRecorder) BatchRemoveReplica(rs, rList interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BatchRemoveReplica", reflect.TypeOf((*MockResourceManager)(nil).BatchRemoveReplica), rs, rList)
}

// BatchUpdateReplica mocks base method.
func (m *MockResourceManager) BatchUpdateReplica(rs *v1.RollingSet, rList []*v1.Replica) (int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BatchUpdateReplica", rs, rList)
	ret0, _ := ret[0].(int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// BatchUpdateReplica indicates an expected call of BatchUpdateReplica.
func (mr *MockResourceManagerMockRecorder) BatchUpdateReplica(rs, rList interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BatchUpdateReplica", reflect.TypeOf((*MockResourceManager)(nil).BatchUpdateReplica), rs, rList)
}

// CreatePod mocks base method.
func (m *MockResourceManager) CreatePod(pod *v10.Pod) (*v10.Pod, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreatePod", pod)
	ret0, _ := ret[0].(*v10.Pod)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreatePod indicates an expected call of CreatePod.
func (mr *MockResourceManagerMockRecorder) CreatePod(pod interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreatePod", reflect.TypeOf((*MockResourceManager)(nil).CreatePod), pod)
}

// CreateRollingSet mocks base method.
func (m *MockResourceManager) CreateRollingSet(sg *v1.ShardGroup, rs *v1.RollingSet) (*v1.RollingSet, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateRollingSet", sg, rs)
	ret0, _ := ret[0].(*v1.RollingSet)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateRollingSet indicates an expected call of CreateRollingSet.
func (mr *MockResourceManagerMockRecorder) CreateRollingSet(sg, rs interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateRollingSet", reflect.TypeOf((*MockResourceManager)(nil).CreateRollingSet), sg, rs)
}

// CreateServicePublisher mocks base method.
func (m *MockResourceManager) CreateServicePublisher(p *v1.ServicePublisher) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateServicePublisher", p)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateServicePublisher indicates an expected call of CreateServicePublisher.
func (mr *MockResourceManagerMockRecorder) CreateServicePublisher(p interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateServicePublisher", reflect.TypeOf((*MockResourceManager)(nil).CreateServicePublisher), p)
}

// CreateWorkerNode mocks base method.
func (m *MockResourceManager) CreateWorkerNode(worker *v1.WorkerNode) (*v1.WorkerNode, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateWorkerNode", worker)
	ret0, _ := ret[0].(*v1.WorkerNode)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateWorkerNode indicates an expected call of CreateWorkerNode.
func (mr *MockResourceManagerMockRecorder) CreateWorkerNode(worker interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateWorkerNode", reflect.TypeOf((*MockResourceManager)(nil).CreateWorkerNode), worker)
}

// DeletePod mocks base method.
func (m *MockResourceManager) DeletePod(pod *v10.Pod, grace bool) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeletePod", pod, grace)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeletePod indicates an expected call of DeletePod.
func (mr *MockResourceManagerMockRecorder) DeletePod(pod, grace interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeletePod", reflect.TypeOf((*MockResourceManager)(nil).DeletePod), pod, grace)
}

// DeleteServicePublisher mocks base method.
func (m *MockResourceManager) DeleteServicePublisher(p *v1.ServicePublisher) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteServicePublisher", p)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteServicePublisher indicates an expected call of DeleteServicePublisher.
func (mr *MockResourceManagerMockRecorder) DeleteServicePublisher(p interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteServicePublisher", reflect.TypeOf((*MockResourceManager)(nil).DeleteServicePublisher), p)
}

// DeleteServicePublisherForRs mocks base method.
func (m *MockResourceManager) DeleteServicePublisherForRs(rs *v1.RollingSet) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteServicePublisherForRs", rs)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteServicePublisherForRs indicates an expected call of DeleteServicePublisherForRs.
func (mr *MockResourceManagerMockRecorder) DeleteServicePublisherForRs(rs interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteServicePublisherForRs", reflect.TypeOf((*MockResourceManager)(nil).DeleteServicePublisherForRs), rs)
}

// DeleteShardGroup mocks base method.
func (m *MockResourceManager) DeleteShardGroup(sg *v1.ShardGroup) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteShardGroup", sg)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteShardGroup indicates an expected call of DeleteShardGroup.
func (mr *MockResourceManagerMockRecorder) DeleteShardGroup(sg interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteShardGroup", reflect.TypeOf((*MockResourceManager)(nil).DeleteShardGroup), sg)
}

// DeleteWorkerNode mocks base method.
func (m *MockResourceManager) DeleteWorkerNode(worker *v1.WorkerNode) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteWorkerNode", worker)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteWorkerNode indicates an expected call of DeleteWorkerNode.
func (mr *MockResourceManagerMockRecorder) DeleteWorkerNode(worker interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteWorkerNode", reflect.TypeOf((*MockResourceManager)(nil).DeleteWorkerNode), worker)
}

// ListReplicaForRS mocks base method.
func (m *MockResourceManager) ListReplicaForRS(rs *v1.RollingSet) ([]*v1.Replica, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListReplicaForRS", rs)
	ret0, _ := ret[0].([]*v1.Replica)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListReplicaForRS indicates an expected call of ListReplicaForRS.
func (mr *MockResourceManagerMockRecorder) ListReplicaForRS(rs interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListReplicaForRS", reflect.TypeOf((*MockResourceManager)(nil).ListReplicaForRS), rs)
}

// ListWorkerNodeByOwner mocks base method.
func (m *MockResourceManager) ListWorkerNodeByOwner(selector map[string]string, ownerKey string) ([]*v1.WorkerNode, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListWorkerNodeByOwner", selector, ownerKey)
	ret0, _ := ret[0].([]*v1.WorkerNode)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListWorkerNodeByOwner indicates an expected call of ListWorkerNodeByOwner.
func (mr *MockResourceManagerMockRecorder) ListWorkerNodeByOwner(selector, ownerKey interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListWorkerNodeByOwner", reflect.TypeOf((*MockResourceManager)(nil).ListWorkerNodeByOwner), selector, ownerKey)
}

// ListWorkerNodeForRS mocks base method.
func (m *MockResourceManager) ListWorkerNodeForRS(selector map[string]string) ([]*v1.WorkerNode, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListWorkerNodeForRS", selector)
	ret0, _ := ret[0].([]*v1.WorkerNode)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListWorkerNodeForRS indicates an expected call of ListWorkerNodeForRS.
func (mr *MockResourceManagerMockRecorder) ListWorkerNodeForRS(selector interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListWorkerNodeForRS", reflect.TypeOf((*MockResourceManager)(nil).ListWorkerNodeForRS), selector)
}

// PatchPod mocks base method.
func (m *MockResourceManager) PatchPod(pod *v10.Pod, pt types.PatchType, data []byte, subresource []string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PatchPod", pod, pt, data, subresource)
	ret0, _ := ret[0].(error)
	return ret0
}

// PatchPod indicates an expected call of PatchPod.
func (mr *MockResourceManagerMockRecorder) PatchPod(pod, pt, data, subresource interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PatchPod", reflect.TypeOf((*MockResourceManager)(nil).PatchPod), pod, pt, data, subresource)
}

// ReleaseReplica mocks base method.
func (m *MockResourceManager) ReleaseReplica(rs *v1.RollingSet, r *v1.Replica) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReleaseReplica", rs, r)
	ret0, _ := ret[0].(error)
	return ret0
}

// ReleaseReplica indicates an expected call of ReleaseReplica.
func (mr *MockResourceManagerMockRecorder) ReleaseReplica(rs, r interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReleaseReplica", reflect.TypeOf((*MockResourceManager)(nil).ReleaseReplica), rs, r)
}

// ReleaseWorkerNode mocks base method.
func (m *MockResourceManager) ReleaseWorkerNode(worker *v1.WorkerNode) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReleaseWorkerNode", worker)
	ret0, _ := ret[0].(error)
	return ret0
}

// ReleaseWorkerNode indicates an expected call of ReleaseWorkerNode.
func (mr *MockResourceManagerMockRecorder) ReleaseWorkerNode(worker interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReleaseWorkerNode", reflect.TypeOf((*MockResourceManager)(nil).ReleaseWorkerNode), worker)
}

// RemoveReplica mocks base method.
func (m *MockResourceManager) RemoveReplica(rs *v1.RollingSet, r *v1.Replica) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RemoveReplica", rs, r)
	ret0, _ := ret[0].(error)
	return ret0
}

// RemoveReplica indicates an expected call of RemoveReplica.
func (mr *MockResourceManagerMockRecorder) RemoveReplica(rs, r interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RemoveReplica", reflect.TypeOf((*MockResourceManager)(nil).RemoveReplica), rs, r)
}

// RemoveRollingSet mocks base method.
func (m *MockResourceManager) RemoveRollingSet(rs *v1.RollingSet) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RemoveRollingSet", rs)
	ret0, _ := ret[0].(error)
	return ret0
}

// RemoveRollingSet indicates an expected call of RemoveRollingSet.
func (mr *MockResourceManagerMockRecorder) RemoveRollingSet(rs interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RemoveRollingSet", reflect.TypeOf((*MockResourceManager)(nil).RemoveRollingSet), rs)
}

// UpdatePodSpec mocks base method.
func (m *MockResourceManager) UpdatePodSpec(pod *v10.Pod) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdatePodSpec", pod)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdatePodSpec indicates an expected call of UpdatePodSpec.
func (mr *MockResourceManagerMockRecorder) UpdatePodSpec(pod interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdatePodSpec", reflect.TypeOf((*MockResourceManager)(nil).UpdatePodSpec), pod)
}

// UpdatePodStatus mocks base method.
func (m *MockResourceManager) UpdatePodStatus(pod *v10.Pod) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdatePodStatus", pod)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdatePodStatus indicates an expected call of UpdatePodStatus.
func (mr *MockResourceManagerMockRecorder) UpdatePodStatus(pod interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdatePodStatus", reflect.TypeOf((*MockResourceManager)(nil).UpdatePodStatus), pod)
}

// UpdateReplica mocks base method.
func (m *MockResourceManager) UpdateReplica(rs *v1.RollingSet, r *v1.Replica) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateReplica", rs, r)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateReplica indicates an expected call of UpdateReplica.
func (mr *MockResourceManagerMockRecorder) UpdateReplica(rs, r interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateReplica", reflect.TypeOf((*MockResourceManager)(nil).UpdateReplica), rs, r)
}

// UpdateReplicaStatus mocks base method.
func (m *MockResourceManager) UpdateReplicaStatus(r *v1.Replica) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateReplicaStatus", r)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateReplicaStatus indicates an expected call of UpdateReplicaStatus.
func (mr *MockResourceManagerMockRecorder) UpdateReplicaStatus(r interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateReplicaStatus", reflect.TypeOf((*MockResourceManager)(nil).UpdateReplicaStatus), r)
}

// UpdateRollingSet mocks base method.
func (m *MockResourceManager) UpdateRollingSet(rs *v1.RollingSet) (*v1.RollingSet, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateRollingSet", rs)
	ret0, _ := ret[0].(*v1.RollingSet)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateRollingSet indicates an expected call of UpdateRollingSet.
func (mr *MockResourceManagerMockRecorder) UpdateRollingSet(rs interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateRollingSet", reflect.TypeOf((*MockResourceManager)(nil).UpdateRollingSet), rs)
}

// UpdateRollingSetStatus mocks base method.
func (m *MockResourceManager) UpdateRollingSetStatus(rs *v1.RollingSet) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateRollingSetStatus", rs)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateRollingSetStatus indicates an expected call of UpdateRollingSetStatus.
func (mr *MockResourceManagerMockRecorder) UpdateRollingSetStatus(rs interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateRollingSetStatus", reflect.TypeOf((*MockResourceManager)(nil).UpdateRollingSetStatus), rs)
}

// UpdateServicePublisher mocks base method.
func (m *MockResourceManager) UpdateServicePublisher(p *v1.ServicePublisher) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateServicePublisher", p)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateServicePublisher indicates an expected call of UpdateServicePublisher.
func (mr *MockResourceManagerMockRecorder) UpdateServicePublisher(p interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateServicePublisher", reflect.TypeOf((*MockResourceManager)(nil).UpdateServicePublisher), p)
}

// UpdateServicePublisherStatus mocks base method.
func (m *MockResourceManager) UpdateServicePublisherStatus(p *v1.ServicePublisher) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateServicePublisherStatus", p)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateServicePublisherStatus indicates an expected call of UpdateServicePublisherStatus.
func (mr *MockResourceManagerMockRecorder) UpdateServicePublisherStatus(p interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateServicePublisherStatus", reflect.TypeOf((*MockResourceManager)(nil).UpdateServicePublisherStatus), p)
}

// UpdateShardGroup mocks base method.
func (m *MockResourceManager) UpdateShardGroup(sg *v1.ShardGroup) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateShardGroup", sg)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateShardGroup indicates an expected call of UpdateShardGroup.
func (mr *MockResourceManagerMockRecorder) UpdateShardGroup(sg interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateShardGroup", reflect.TypeOf((*MockResourceManager)(nil).UpdateShardGroup), sg)
}

// UpdateShardGroupStatus mocks base method.
func (m *MockResourceManager) UpdateShardGroupStatus(sg *v1.ShardGroup) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateShardGroupStatus", sg)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateShardGroupStatus indicates an expected call of UpdateShardGroupStatus.
func (mr *MockResourceManagerMockRecorder) UpdateShardGroupStatus(sg interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateShardGroupStatus", reflect.TypeOf((*MockResourceManager)(nil).UpdateShardGroupStatus), sg)
}

// UpdateWorkerNode mocks base method.
func (m *MockResourceManager) UpdateWorkerNode(wn *v1.WorkerNode) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateWorkerNode", wn)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateWorkerNode indicates an expected call of UpdateWorkerNode.
func (mr *MockResourceManagerMockRecorder) UpdateWorkerNode(wn interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateWorkerNode", reflect.TypeOf((*MockResourceManager)(nil).UpdateWorkerNode), wn)
}

// UpdateWorkerStatus mocks base method.
func (m *MockResourceManager) UpdateWorkerStatus(worker *v1.WorkerNode) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateWorkerStatus", worker)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateWorkerStatus indicates an expected call of UpdateWorkerStatus.
func (mr *MockResourceManagerMockRecorder) UpdateWorkerStatus(worker interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateWorkerStatus", reflect.TypeOf((*MockResourceManager)(nil).UpdateWorkerStatus), worker)
}