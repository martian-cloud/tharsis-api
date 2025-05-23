// Code generated by mockery v2.53.0. DO NOT EDIT.

package db

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
	models "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// MockRoles is an autogenerated mock type for the Roles type
type MockRoles struct {
	mock.Mock
}

// CreateRole provides a mock function with given fields: ctx, role
func (_m *MockRoles) CreateRole(ctx context.Context, role *models.Role) (*models.Role, error) {
	ret := _m.Called(ctx, role)

	if len(ret) == 0 {
		panic("no return value specified for CreateRole")
	}

	var r0 *models.Role
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *models.Role) (*models.Role, error)); ok {
		return rf(ctx, role)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *models.Role) *models.Role); ok {
		r0 = rf(ctx, role)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.Role)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *models.Role) error); ok {
		r1 = rf(ctx, role)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DeleteRole provides a mock function with given fields: ctx, role
func (_m *MockRoles) DeleteRole(ctx context.Context, role *models.Role) error {
	ret := _m.Called(ctx, role)

	if len(ret) == 0 {
		panic("no return value specified for DeleteRole")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *models.Role) error); ok {
		r0 = rf(ctx, role)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetRoleByID provides a mock function with given fields: ctx, id
func (_m *MockRoles) GetRoleByID(ctx context.Context, id string) (*models.Role, error) {
	ret := _m.Called(ctx, id)

	if len(ret) == 0 {
		panic("no return value specified for GetRoleByID")
	}

	var r0 *models.Role
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*models.Role, error)); ok {
		return rf(ctx, id)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *models.Role); ok {
		r0 = rf(ctx, id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.Role)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetRoleByName provides a mock function with given fields: ctx, name
func (_m *MockRoles) GetRoleByName(ctx context.Context, name string) (*models.Role, error) {
	ret := _m.Called(ctx, name)

	if len(ret) == 0 {
		panic("no return value specified for GetRoleByName")
	}

	var r0 *models.Role
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*models.Role, error)); ok {
		return rf(ctx, name)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *models.Role); ok {
		r0 = rf(ctx, name)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.Role)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetRoleByTRN provides a mock function with given fields: ctx, trn
func (_m *MockRoles) GetRoleByTRN(ctx context.Context, trn string) (*models.Role, error) {
	ret := _m.Called(ctx, trn)

	if len(ret) == 0 {
		panic("no return value specified for GetRoleByTRN")
	}

	var r0 *models.Role
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*models.Role, error)); ok {
		return rf(ctx, trn)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *models.Role); ok {
		r0 = rf(ctx, trn)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.Role)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, trn)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetRoles provides a mock function with given fields: ctx, input
func (_m *MockRoles) GetRoles(ctx context.Context, input *GetRolesInput) (*RolesResult, error) {
	ret := _m.Called(ctx, input)

	if len(ret) == 0 {
		panic("no return value specified for GetRoles")
	}

	var r0 *RolesResult
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *GetRolesInput) (*RolesResult, error)); ok {
		return rf(ctx, input)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *GetRolesInput) *RolesResult); ok {
		r0 = rf(ctx, input)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*RolesResult)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *GetRolesInput) error); ok {
		r1 = rf(ctx, input)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// UpdateRole provides a mock function with given fields: ctx, role
func (_m *MockRoles) UpdateRole(ctx context.Context, role *models.Role) (*models.Role, error) {
	ret := _m.Called(ctx, role)

	if len(ret) == 0 {
		panic("no return value specified for UpdateRole")
	}

	var r0 *models.Role
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *models.Role) (*models.Role, error)); ok {
		return rf(ctx, role)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *models.Role) *models.Role); ok {
		r0 = rf(ctx, role)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.Role)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *models.Role) error); ok {
		r1 = rf(ctx, role)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewMockRoles creates a new instance of MockRoles. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockRoles(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockRoles {
	mock := &MockRoles{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
