// Code generated by MockGen. DO NOT EDIT.
// Source: controller/worker/worker_processor.go

// Package worker is a generated GoMock package.
package worker

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	v1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	v10 "k8s.io/api/core/v1"
)

// Mockprocessor is a mock of processor interface.
type Mockprocessor struct {
	ctrl     *gomock.Controller
	recorder *MockprocessorMockRecorder
}

// MockprocessorMockRecorder is the mock recorder for Mockprocessor.
type MockprocessorMockRecorder struct {
	mock *Mockprocessor
}

// NewMockprocessor creates a new mock instance.
func NewMockprocessor(ctrl *gomock.Controller) *Mockprocessor {
	mock := &Mockprocessor{ctrl: ctrl}
	mock.recorder = &MockprocessorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *Mockprocessor) EXPECT() *MockprocessorMockRecorder {
	return m.recorder
}

// process mocks base method.
func (m *Mockprocessor) process() (*v1.WorkerNode, *v10.Pod, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "process")
	ret0, _ := ret[0].(*v1.WorkerNode)
	ret1, _ := ret[1].(*v10.Pod)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// process indicates an expected call of process.
func (mr *MockprocessorMockRecorder) process() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "process", reflect.TypeOf((*Mockprocessor)(nil).process))
}

// MockworkerStateMachine is a mock of workerStateMachine interface.
type MockworkerStateMachine struct {
	ctrl     *gomock.Controller
	recorder *MockworkerStateMachineMockRecorder
}

// MockworkerStateMachineMockRecorder is the mock recorder for MockworkerStateMachine.
type MockworkerStateMachineMockRecorder struct {
	mock *MockworkerStateMachine
}

// NewMockworkerStateMachine creates a new mock instance.
func NewMockworkerStateMachine(ctrl *gomock.Controller) *MockworkerStateMachine {
	mock := &MockworkerStateMachine{ctrl: ctrl}
	mock.recorder = &MockworkerStateMachineMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockworkerStateMachine) EXPECT() *MockworkerStateMachineMockRecorder {
	return m.recorder
}

// dealwithAssigned mocks base method.
func (m *MockworkerStateMachine) dealwithAssigned() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "dealwithAssigned")
	ret0, _ := ret[0].(error)
	return ret0
}

// dealwithAssigned indicates an expected call of dealwithAssigned.
func (mr *MockworkerStateMachineMockRecorder) dealwithAssigned() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "dealwithAssigned", reflect.TypeOf((*MockworkerStateMachine)(nil).dealwithAssigned))
}

// dealwithLost mocks base method.
func (m *MockworkerStateMachine) dealwithLost() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "dealwithLost")
	ret0, _ := ret[0].(error)
	return ret0
}

// dealwithLost indicates an expected call of dealwithLost.
func (mr *MockworkerStateMachineMockRecorder) dealwithLost() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "dealwithLost", reflect.TypeOf((*MockworkerStateMachine)(nil).dealwithLost))
}

// dealwithOfflining mocks base method.
func (m *MockworkerStateMachine) dealwithOfflining() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "dealwithOfflining")
	ret0, _ := ret[0].(error)
	return ret0
}

// dealwithOfflining indicates an expected call of dealwithOfflining.
func (mr *MockworkerStateMachineMockRecorder) dealwithOfflining() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "dealwithOfflining", reflect.TypeOf((*MockworkerStateMachine)(nil).dealwithOfflining))
}

// dealwithReleased mocks base method.
func (m *MockworkerStateMachine) dealwithReleased() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "dealwithReleased")
	ret0, _ := ret[0].(error)
	return ret0
}

// dealwithReleased indicates an expected call of dealwithReleased.
func (mr *MockworkerStateMachineMockRecorder) dealwithReleased() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "dealwithReleased", reflect.TypeOf((*MockworkerStateMachine)(nil).dealwithReleased))
}

// dealwithReleasing mocks base method.
func (m *MockworkerStateMachine) dealwithReleasing() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "dealwithReleasing")
	ret0, _ := ret[0].(error)
	return ret0
}

// dealwithReleasing indicates an expected call of dealwithReleasing.
func (mr *MockworkerStateMachineMockRecorder) dealwithReleasing() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "dealwithReleasing", reflect.TypeOf((*MockworkerStateMachine)(nil).dealwithReleasing))
}

// dealwithUnAssigned mocks base method.
func (m *MockworkerStateMachine) dealwithUnAssigned() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "dealwithUnAssigned")
	ret0, _ := ret[0].(error)
	return ret0
}

// dealwithUnAssigned indicates an expected call of dealwithUnAssigned.
func (mr *MockworkerStateMachineMockRecorder) dealwithUnAssigned() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "dealwithUnAssigned", reflect.TypeOf((*MockworkerStateMachine)(nil).dealwithUnAssigned))
}

// postSync mocks base method.
func (m *MockworkerStateMachine) postSync() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "postSync")
	ret0, _ := ret[0].(error)
	return ret0
}

// postSync indicates an expected call of postSync.
func (mr *MockworkerStateMachineMockRecorder) postSync() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "postSync", reflect.TypeOf((*MockworkerStateMachine)(nil).postSync))
}

// prepare mocks base method.
func (m *MockworkerStateMachine) prepare() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "prepare")
	ret0, _ := ret[0].(error)
	return ret0
}

// prepare indicates an expected call of prepare.
func (mr *MockworkerStateMachineMockRecorder) prepare() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "prepare", reflect.TypeOf((*MockworkerStateMachine)(nil).prepare))
}

// MockworkerUpdater is a mock of workerUpdater interface.
type MockworkerUpdater struct {
	ctrl     *gomock.Controller
	recorder *MockworkerUpdaterMockRecorder
}

// MockworkerUpdaterMockRecorder is the mock recorder for MockworkerUpdater.
type MockworkerUpdaterMockRecorder struct {
	mock *MockworkerUpdater
}

// NewMockworkerUpdater creates a new mock instance.
func NewMockworkerUpdater(ctrl *gomock.Controller) *MockworkerUpdater {
	mock := &MockworkerUpdater{ctrl: ctrl}
	mock.recorder = &MockworkerUpdaterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockworkerUpdater) EXPECT() *MockworkerUpdaterMockRecorder {
	return m.recorder
}

// processHealthInfo mocks base method.
func (m *MockworkerUpdater) processHealthInfo(worker *v1.WorkerNode) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "processHealthInfo", worker)
	ret0, _ := ret[0].(bool)
	return ret0
}

// processHealthInfo indicates an expected call of processHealthInfo.
func (mr *MockworkerUpdaterMockRecorder) processHealthInfo(worker interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "processHealthInfo", reflect.TypeOf((*MockworkerUpdater)(nil).processHealthInfo), worker)
}

// processPlan mocks base method.
func (m *MockworkerUpdater) processPlan(worker *v1.WorkerNode) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "processPlan", worker)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// processPlan indicates an expected call of processPlan.
func (mr *MockworkerUpdaterMockRecorder) processPlan(worker interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "processPlan", reflect.TypeOf((*MockworkerUpdater)(nil).processPlan), worker)
}

// processServiceInfo mocks base method.
func (m *MockworkerUpdater) processServiceInfo(worker *v1.WorkerNode) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "processServiceInfo", worker)
	ret0, _ := ret[0].(bool)
	return ret0
}

// processServiceInfo indicates an expected call of processServiceInfo.
func (mr *MockworkerUpdaterMockRecorder) processServiceInfo(worker interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "processServiceInfo", reflect.TypeOf((*MockworkerUpdater)(nil).processServiceInfo), worker)
}

// processUpdateGracefully mocks base method.
func (m *MockworkerUpdater) processUpdateGracefully(worker *v1.WorkerNode) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "processUpdateGracefully", worker)
	ret0, _ := ret[0].(bool)
	return ret0
}

// processUpdateGracefully indicates an expected call of processUpdateGracefully.
func (mr *MockworkerUpdaterMockRecorder) processUpdateGracefully(worker interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "processUpdateGracefully", reflect.TypeOf((*MockworkerUpdater)(nil).processUpdateGracefully), worker)
}
