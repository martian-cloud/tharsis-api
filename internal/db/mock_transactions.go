// Code generated by mockery v2.53.0. DO NOT EDIT.

package db

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
)

// MockTransactions is an autogenerated mock type for the Transactions type
type MockTransactions struct {
	mock.Mock
}

// BeginTx provides a mock function with given fields: ctx
func (_m *MockTransactions) BeginTx(ctx context.Context) (context.Context, error) {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for BeginTx")
	}

	var r0 context.Context
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) (context.Context, error)); ok {
		return rf(ctx)
	}
	if rf, ok := ret.Get(0).(func(context.Context) context.Context); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(context.Context)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CommitTx provides a mock function with given fields: ctx
func (_m *MockTransactions) CommitTx(ctx context.Context) error {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for CommitTx")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// RollbackTx provides a mock function with given fields: ctx
func (_m *MockTransactions) RollbackTx(ctx context.Context) error {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for RollbackTx")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewMockTransactions creates a new instance of MockTransactions. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockTransactions(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockTransactions {
	mock := &MockTransactions{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
