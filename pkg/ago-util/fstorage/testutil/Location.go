// Code generated by mockery v0.0.0-dev. DO NOT EDIT.

package testutil

import (
	mock "github.com/stretchr/testify/mock"
	fstorage "github.com/alibaba/kube-sharding/pkg/ago-util/fstorage"
)

// Location is an autogenerated mock type for the Location type
type Location struct {
	mock.Mock
}

// Close provides a mock function with given fields:
func (_m *Location) Close() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Create provides a mock function with given fields: sub
func (_m *Location) Create(sub string) (fstorage.FStorage, error) {
	ret := _m.Called(sub)

	var r0 fstorage.FStorage
	if rf, ok := ret.Get(0).(func(string) fstorage.FStorage); ok {
		r0 = rf(sub)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(fstorage.FStorage)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(sub)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// List provides a mock function with given fields:
func (_m *Location) List() ([]string, error) {
	ret := _m.Called()

	var r0 []string
	if rf, ok := ret.Get(0).(func() []string); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// New provides a mock function with given fields: sub
func (_m *Location) New(sub string) (fstorage.FStorage, error) {
	ret := _m.Called(sub)

	var r0 fstorage.FStorage
	if rf, ok := ret.Get(0).(func(string) fstorage.FStorage); ok {
		r0 = rf(sub)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(fstorage.FStorage)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(sub)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewSubLocation provides a mock function with given fields: sub
func (_m *Location) NewSubLocation(sub ...string) (fstorage.Location, error) {
	_va := make([]interface{}, len(sub))
	for _i := range sub {
		_va[_i] = sub[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 fstorage.Location
	if rf, ok := ret.Get(0).(func(...string) fstorage.Location); ok {
		r0 = rf(sub...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(fstorage.Location)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(...string) error); ok {
		r1 = rf(sub...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
