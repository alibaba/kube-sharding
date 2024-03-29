// Code generated by MockGen. DO NOT EDIT.
// Source: nodeset_scheduler.go

// Package mock_rollalgorithm is a generated GoMock package.
package rollalgorithm

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockNode is a mock of Node interface.
type MockNode struct {
	ctrl     *gomock.Controller
	recorder *MockNodeMockRecorder
}

// MockNodeMockRecorder is the mock recorder for MockNode.
type MockNodeMockRecorder struct {
	mock *MockNode
}

// NewMockNode creates a new mock instance.
func NewMockNode(ctrl *gomock.Controller) *MockNode {
	mock := &MockNode{ctrl: ctrl}
	mock.recorder = &MockNodeMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockNode) EXPECT() *MockNodeMockRecorder {
	return m.recorder
}

// GetDeletionCost mocks base method.
func (m *MockNode) GetDeletionCost() int {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetDeletionCost")
	ret0, _ := ret[0].(int)
	return ret0
}

// GetDeletionCost indicates an expected call of GetDeletionCost.
func (mr *MockNodeMockRecorder) GetDeletionCost() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDeletionCost", reflect.TypeOf((*MockNode)(nil).GetDeletionCost))
}

// GetDependencyReady mocks base method.
func (m *MockNode) GetDependencyReady() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetDependencyReady")
	ret0, _ := ret[0].(bool)
	return ret0
}

// GetDependencyReady indicates an expected call of GetDependencyReady.
func (mr *MockNodeMockRecorder) GetDependencyReady() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDependencyReady", reflect.TypeOf((*MockNode)(nil).GetDependencyReady))
}

// GetName mocks base method.
func (m *MockNode) GetName() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetName")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetName indicates an expected call of GetName.
func (mr *MockNodeMockRecorder) GetName() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetName", reflect.TypeOf((*MockNode)(nil).GetName))
}

// GetScore mocks base method.
func (m *MockNode) GetScore() int {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetScore")
	ret0, _ := ret[0].(int)
	return ret0
}

// GetScore indicates an expected call of GetScore.
func (mr *MockNodeMockRecorder) GetScore() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetScore", reflect.TypeOf((*MockNode)(nil).GetScore))
}

// GetVersion mocks base method.
func (m *MockNode) GetVersion() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetVersion")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetVersion indicates an expected call of GetVersion.
func (mr *MockNodeMockRecorder) GetVersion() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetVersion", reflect.TypeOf((*MockNode)(nil).GetVersion))
}

// GetVersionPlan mocks base method.
func (m *MockNode) GetVersionPlan() interface{} {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetVersionPlan")
	ret0, _ := ret[0].(interface{})
	return ret0
}

// GetVersionPlan indicates an expected call of GetVersionPlan.
func (mr *MockNodeMockRecorder) GetVersionPlan() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetVersionPlan", reflect.TypeOf((*MockNode)(nil).GetVersionPlan))
}

// IsEmpty mocks base method.
func (m *MockNode) IsEmpty() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsEmpty")
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsEmpty indicates an expected call of IsEmpty.
func (mr *MockNodeMockRecorder) IsEmpty() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsEmpty", reflect.TypeOf((*MockNode)(nil).IsEmpty))
}

// IsReady mocks base method.
func (m *MockNode) IsReady() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsReady")
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsReady indicates an expected call of IsReady.
func (mr *MockNodeMockRecorder) IsReady() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsReady", reflect.TypeOf((*MockNode)(nil).IsReady))
}

// IsRelease mocks base method.
func (m *MockNode) IsRelease() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsRelease")
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsRelease indicates an expected call of IsRelease.
func (mr *MockNodeMockRecorder) IsRelease() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsRelease", reflect.TypeOf((*MockNode)(nil).IsRelease))
}

// SetDependencyReady mocks base method.
func (m *MockNode) SetDependencyReady(arg0 bool) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetDependencyReady", arg0)
}

// SetDependencyReady indicates an expected call of SetDependencyReady.
func (mr *MockNodeMockRecorder) SetDependencyReady(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetDependencyReady", reflect.TypeOf((*MockNode)(nil).SetDependencyReady), arg0)
}

// SetVersion mocks base method.
func (m *MockNode) SetVersion(arg0 string, arg1 interface{}) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetVersion", arg0, arg1)
}

// SetVersion indicates an expected call of SetVersion.
func (mr *MockNodeMockRecorder) SetVersion(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetVersion", reflect.TypeOf((*MockNode)(nil).SetVersion), arg0, arg1)
}

// Stop mocks base method.
func (m *MockNode) Stop() {
	m.ctrl.Call(m, "Stop")
}

// Stop indicates an expected call of Stop.
func (mr *MockNodeMockRecorder) Stop() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Stop", reflect.TypeOf((*MockNode)(nil).Stop))
}