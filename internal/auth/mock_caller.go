// Code generated by mockery v2.53.0. DO NOT EDIT.

package auth

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
	models "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	types "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
)

// MockCaller is an autogenerated mock type for the Caller type
type MockCaller struct {
	mock.Mock
}

// GetNamespaceAccessPolicy provides a mock function with given fields: ctx
func (_m *MockCaller) GetNamespaceAccessPolicy(ctx context.Context) (*NamespaceAccessPolicy, error) {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for GetNamespaceAccessPolicy")
	}

	var r0 *NamespaceAccessPolicy
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) (*NamespaceAccessPolicy, error)); ok {
		return rf(ctx)
	}
	if rf, ok := ret.Get(0).(func(context.Context) *NamespaceAccessPolicy); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*NamespaceAccessPolicy)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetSubject provides a mock function with no fields
func (_m *MockCaller) GetSubject() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetSubject")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// IsAdmin provides a mock function with no fields
func (_m *MockCaller) IsAdmin() bool {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for IsAdmin")
	}

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// RequireAccessToInheritableResource provides a mock function with given fields: ctx, modelType, checks
func (_m *MockCaller) RequireAccessToInheritableResource(ctx context.Context, modelType types.ModelType, checks ...func(*constraints)) error {
	_va := make([]interface{}, len(checks))
	for _i := range checks {
		_va[_i] = checks[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, modelType)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for RequireAccessToInheritableResource")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, types.ModelType, ...func(*constraints)) error); ok {
		r0 = rf(ctx, modelType, checks...)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// RequirePermission provides a mock function with given fields: ctx, perms, checks
func (_m *MockCaller) RequirePermission(ctx context.Context, perms models.Permission, checks ...func(*constraints)) error {
	_va := make([]interface{}, len(checks))
	for _i := range checks {
		_va[_i] = checks[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, perms)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for RequirePermission")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, models.Permission, ...func(*constraints)) error); ok {
		r0 = rf(ctx, perms, checks...)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UnauthorizedError provides a mock function with given fields: ctx, hasViewerAccess
func (_m *MockCaller) UnauthorizedError(ctx context.Context, hasViewerAccess bool) error {
	ret := _m.Called(ctx, hasViewerAccess)

	if len(ret) == 0 {
		panic("no return value specified for UnauthorizedError")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, bool) error); ok {
		r0 = rf(ctx, hasViewerAccess)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewMockCaller creates a new instance of MockCaller. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockCaller(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockCaller {
	mock := &MockCaller{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
