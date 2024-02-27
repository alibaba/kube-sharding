// Code generated by MockGen. DO NOT EDIT.
// Source: proxy/service/k8s_adapter.go

// Package service is a generated GoMock package.
package service

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	v1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	typespec "github.com/alibaba/kube-sharding/pkg/apiserver-client/typespec"
)

// Mockk8sSyncer is a mock of k8sSyncer interface.
type Mockk8sSyncer struct {
	ctrl     *gomock.Controller
	recorder *Mockk8sSyncerMockRecorder
}

// Mockk8sSyncerMockRecorder is the mock recorder for Mockk8sSyncer.
type Mockk8sSyncerMockRecorder struct {
	mock *Mockk8sSyncer
}

// NewMockk8sSyncer creates a new mock instance.
func NewMockk8sSyncer(ctrl *gomock.Controller) *Mockk8sSyncer {
	mock := &Mockk8sSyncer{ctrl: ctrl}
	mock.recorder = &Mockk8sSyncerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *Mockk8sSyncer) EXPECT() *Mockk8sSyncerMockRecorder {
	return m.recorder
}

// deleteShardGroup mocks base method.
func (m *Mockk8sSyncer) deleteShardGroup(appName, groupID string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "deleteShardGroup", appName, groupID)
	ret0, _ := ret[0].(error)
	return ret0
}

// deleteShardGroup indicates an expected call of deleteShardGroup.
func (mr *Mockk8sSyncerMockRecorder) deleteShardGroup(appName, groupID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "deleteShardGroup", reflect.TypeOf((*Mockk8sSyncer)(nil).deleteShardGroup), appName, groupID)
}

// deleteSingleRollingset mocks base method.
func (m *Mockk8sSyncer) deleteSingleRollingset(appName, groupID string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "deleteSingleRollingset", appName, groupID)
	ret0, _ := ret[0].(error)
	return ret0
}

// deleteSingleRollingset indicates an expected call of deleteSingleRollingset.
func (mr *Mockk8sSyncerMockRecorder) deleteSingleRollingset(appName, groupID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "deleteSingleRollingset", reflect.TypeOf((*Mockk8sSyncer)(nil).deleteSingleRollingset), appName, groupID)
}

// syncGangRSPlan mocks base method.
func (m *Mockk8sSyncer) syncGangRSPlan(appName string, groupPlan typespec.GroupPlan, rss []v1.RollingSet) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "syncGangRSPlan", appName, groupPlan, rss)
	ret0, _ := ret[0].(error)
	return ret0
}

// syncGangRSPlan indicates an expected call of syncGangRSPlan.
func (mr *Mockk8sSyncerMockRecorder) syncGangRSPlan(appName, groupPlan, rss interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "syncGangRSPlan", reflect.TypeOf((*Mockk8sSyncer)(nil).syncGangRSPlan), appName, groupPlan, rss)
}

// syncRollingSetPlan mocks base method.
func (m *Mockk8sSyncer) syncRollingSetPlan(appName string, groupPlan typespec.GroupPlan, roles []v1.RollingSet) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "syncRollingSetPlan", appName, groupPlan, roles)
	ret0, _ := ret[0].(error)
	return ret0
}

// syncRollingSetPlan indicates an expected call of syncRollingSetPlan.
func (mr *Mockk8sSyncerMockRecorder) syncRollingSetPlan(appName, groupPlan, roles interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "syncRollingSetPlan", reflect.TypeOf((*Mockk8sSyncer)(nil).syncRollingSetPlan), appName, groupPlan, roles)
}

// syncShardGroupPlan mocks base method.
func (m *Mockk8sSyncer) syncShardGroupPlan(appName string, groupPlan typespec.GroupPlan, groups []v1.ShardGroup) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "syncShardGroupPlan", appName, groupPlan, groups)
	ret0, _ := ret[0].(error)
	return ret0
}

// syncShardGroupPlan indicates an expected call of syncShardGroupPlan.
func (mr *Mockk8sSyncerMockRecorder) syncShardGroupPlan(appName, groupPlan, groups interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "syncShardGroupPlan", reflect.TypeOf((*Mockk8sSyncer)(nil).syncShardGroupPlan), appName, groupPlan, groups)
}
