// Code generated by MockGen. DO NOT EDIT.
// Source: workernode.go

// Package mock_v1 is a generated GoMock package.
package mock_v1

import (
	gomock "github.com/golang/mock/gomock"
	v1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	v10 "github.com/alibaba/kube-sharding/pkg/client/listers/carbon/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	reflect "reflect"
)

// MockWorkerNodeLister is a mock of WorkerNodeLister interface
type MockWorkerNodeLister struct {
	ctrl     *gomock.Controller
	recorder *MockWorkerNodeListerMockRecorder
}

// MockWorkerNodeListerMockRecorder is the mock recorder for MockWorkerNodeLister
type MockWorkerNodeListerMockRecorder struct {
	mock *MockWorkerNodeLister
}

// NewMockWorkerNodeLister creates a new mock instance
func NewMockWorkerNodeLister(ctrl *gomock.Controller) *MockWorkerNodeLister {
	mock := &MockWorkerNodeLister{ctrl: ctrl}
	mock.recorder = &MockWorkerNodeListerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockWorkerNodeLister) EXPECT() *MockWorkerNodeListerMockRecorder {
	return m.recorder
}

// List mocks base method
func (m *MockWorkerNodeLister) List(selector labels.Selector) ([]*v1.WorkerNode, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List", selector)
	ret0, _ := ret[0].([]*v1.WorkerNode)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// List indicates an expected call of List
func (mr *MockWorkerNodeListerMockRecorder) List(selector interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockWorkerNodeLister)(nil).List), selector)
}

// WorkerNodes mocks base method
func (m *MockWorkerNodeLister) WorkerNodes(namespace string) v10.WorkerNodeNamespaceLister {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WorkerNodes", namespace)
	ret0, _ := ret[0].(v10.WorkerNodeNamespaceLister)
	return ret0
}

// WorkerNodes indicates an expected call of WorkerNodes
func (mr *MockWorkerNodeListerMockRecorder) WorkerNodes(namespace interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WorkerNodes", reflect.TypeOf((*MockWorkerNodeLister)(nil).WorkerNodes), namespace)
}

// MockWorkerNodeNamespaceLister is a mock of WorkerNodeNamespaceLister interface
type MockWorkerNodeNamespaceLister struct {
	ctrl     *gomock.Controller
	recorder *MockWorkerNodeNamespaceListerMockRecorder
}

// MockWorkerNodeNamespaceListerMockRecorder is the mock recorder for MockWorkerNodeNamespaceLister
type MockWorkerNodeNamespaceListerMockRecorder struct {
	mock *MockWorkerNodeNamespaceLister
}

// NewMockWorkerNodeNamespaceLister creates a new mock instance
func NewMockWorkerNodeNamespaceLister(ctrl *gomock.Controller) *MockWorkerNodeNamespaceLister {
	mock := &MockWorkerNodeNamespaceLister{ctrl: ctrl}
	mock.recorder = &MockWorkerNodeNamespaceListerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockWorkerNodeNamespaceLister) EXPECT() *MockWorkerNodeNamespaceListerMockRecorder {
	return m.recorder
}

// List mocks base method
func (m *MockWorkerNodeNamespaceLister) List(selector labels.Selector) ([]*v1.WorkerNode, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List", selector)
	ret0, _ := ret[0].([]*v1.WorkerNode)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// List indicates an expected call of List
func (mr *MockWorkerNodeNamespaceListerMockRecorder) List(selector interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockWorkerNodeNamespaceLister)(nil).List), selector)
}

// Get mocks base method
func (m *MockWorkerNodeNamespaceLister) Get(name string) (*v1.WorkerNode, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", name)
	ret0, _ := ret[0].(*v1.WorkerNode)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get
func (mr *MockWorkerNodeNamespaceListerMockRecorder) Get(name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockWorkerNodeNamespaceLister)(nil).Get), name)
}
