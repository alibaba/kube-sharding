// Code generated by MockGen. DO NOT EDIT.
// Source: status_converter.go

// Package mock_carbonjob is a generated GoMock package.
package mock_carbonjob

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	carbon "github.com/alibaba/kube-sharding/pkg/apiserver-client/typespec/carbon"
	v1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	v10 "k8s.io/api/core/v1"
)

// MockgroupStatusConverter is a mock of groupStatusConverter interface
type MockgroupStatusConverter struct {
	ctrl     *gomock.Controller
	recorder *MockgroupStatusConverterMockRecorder
}

// MockgroupStatusConverterMockRecorder is the mock recorder for MockgroupStatusConverter
type MockgroupStatusConverterMockRecorder struct {
	mock *MockgroupStatusConverter
}

// NewMockgroupStatusConverter creates a new mock instance
func NewMockgroupStatusConverter(ctrl *gomock.Controller) *MockgroupStatusConverter {
	mock := &MockgroupStatusConverter{ctrl: ctrl}
	mock.recorder = &MockgroupStatusConverterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockgroupStatusConverter) EXPECT() *MockgroupStatusConverterMockRecorder {
	return m.recorder
}

// Convert mocks base method
func (m *MockgroupStatusConverter) Convert(carbonJob *v1.CarbonJob, workers []*v1.WorkerNode, pods []*v10.Pod) (map[string]*carbon.GroupStatus, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Convert", carbonJob, workers, pods)
	ret0, _ := ret[0].(map[string]*carbon.GroupStatus)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Convert indicates an expected call of Convert
func (mr *MockgroupStatusConverterMockRecorder) Convert(carbonJob, workers, pods interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Convert", reflect.TypeOf((*MockgroupStatusConverter)(nil).Convert), carbonJob, workers, pods)
}
