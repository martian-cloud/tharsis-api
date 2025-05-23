// Code generated by mockery v2.53.0. DO NOT EDIT.

package namespace

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
)

// MockNotificationManager is an autogenerated mock type for the NotificationManager type
type MockNotificationManager struct {
	mock.Mock
}

// GetUsersToNotify provides a mock function with given fields: ctx, input
func (_m *MockNotificationManager) GetUsersToNotify(ctx context.Context, input *GetUsersToNotifyInput) ([]string, error) {
	ret := _m.Called(ctx, input)

	if len(ret) == 0 {
		panic("no return value specified for GetUsersToNotify")
	}

	var r0 []string
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *GetUsersToNotifyInput) ([]string, error)); ok {
		return rf(ctx, input)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *GetUsersToNotifyInput) []string); ok {
		r0 = rf(ctx, input)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *GetUsersToNotifyInput) error); ok {
		r1 = rf(ctx, input)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewMockNotificationManager creates a new instance of MockNotificationManager. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockNotificationManager(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockNotificationManager {
	mock := &MockNotificationManager{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
