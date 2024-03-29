// Code generated by MockGen. DO NOT EDIT.
// Source: ./httpclient.go

// Package mock_utils is a generated GoMock package.
package mock_utils

import (
	gomock "github.com/golang/mock/gomock"
	utils "github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	reflect "reflect"
)

// MockHTTPClient is a mock of HTTPClient interface
type MockHTTPClient struct {
	ctrl     *gomock.Controller
	recorder *MockHTTPClientMockRecorder
}

// MockHTTPClientMockRecorder is the mock recorder for MockHTTPClient
type MockHTTPClientMockRecorder struct {
	mock *MockHTTPClient
}

// NewMockHTTPClient creates a new mock instance
func NewMockHTTPClient(ctrl *gomock.Controller) *MockHTTPClient {
	mock := &MockHTTPClient{ctrl: ctrl}
	mock.recorder = &MockHTTPClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockHTTPClient) EXPECT() *MockHTTPClientMockRecorder {
	return m.recorder
}

// Get mocks base method
func (m *MockHTTPClient) Get(url string) (*utils.LiteHTTPResp, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", url)
	ret0, _ := ret[0].(*utils.LiteHTTPResp)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get
func (mr *MockHTTPClientMockRecorder) Get(url interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockHTTPClient)(nil).Get), url)
}

// Post mocks base method
func (m *MockHTTPClient) Post(url, body string) (*utils.LiteHTTPResp, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Post", url, body)
	ret0, _ := ret[0].(*utils.LiteHTTPResp)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Post indicates an expected call of Post
func (mr *MockHTTPClientMockRecorder) Post(url, body interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Post", reflect.TypeOf((*MockHTTPClient)(nil).Post), url, body)
}

// Put mocks base method
func (m *MockHTTPClient) Put(url, body string) (*utils.LiteHTTPResp, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Put", url, body)
	ret0, _ := ret[0].(*utils.LiteHTTPResp)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Put indicates an expected call of Put
func (mr *MockHTTPClientMockRecorder) Put(url, body interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Put", reflect.TypeOf((*MockHTTPClient)(nil).Put), url, body)
}

// Delete mocks base method
func (m *MockHTTPClient) Delete(url string) (*utils.LiteHTTPResp, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", url)
	ret0, _ := ret[0].(*utils.LiteHTTPResp)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Delete indicates an expected call of Delete
func (mr *MockHTTPClientMockRecorder) Delete(url interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockHTTPClient)(nil).Delete), url)
}
