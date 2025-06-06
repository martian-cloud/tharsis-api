// Code generated by mockery v2.53.0. DO NOT EDIT.

package kubernetes

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
	v1 "k8s.io/api/batch/v1"
)

// mockClient is an autogenerated mock type for the client type
type mockClient struct {
	mock.Mock
}

// CreateJob provides a mock function with given fields: _a0, _a1
func (_m *mockClient) CreateJob(_a0 context.Context, _a1 *v1.Job) (*v1.Job, error) {
	ret := _m.Called(_a0, _a1)

	if len(ret) == 0 {
		panic("no return value specified for CreateJob")
	}

	var r0 *v1.Job
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Job) (*v1.Job, error)); ok {
		return rf(_a0, _a1)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Job) *v1.Job); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.Job)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *v1.Job) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// newMockClient creates a new instance of mockClient. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func newMockClient(t interface {
	mock.TestingT
	Cleanup(func())
}) *mockClient {
	mock := &mockClient{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
