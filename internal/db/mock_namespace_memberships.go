// Code generated by mockery v2.53.0. DO NOT EDIT.

package db

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
	models "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// MockNamespaceMemberships is an autogenerated mock type for the NamespaceMemberships type
type MockNamespaceMemberships struct {
	mock.Mock
}

// CreateNamespaceMembership provides a mock function with given fields: ctx, input
func (_m *MockNamespaceMemberships) CreateNamespaceMembership(ctx context.Context, input *CreateNamespaceMembershipInput) (*models.NamespaceMembership, error) {
	ret := _m.Called(ctx, input)

	if len(ret) == 0 {
		panic("no return value specified for CreateNamespaceMembership")
	}

	var r0 *models.NamespaceMembership
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *CreateNamespaceMembershipInput) (*models.NamespaceMembership, error)); ok {
		return rf(ctx, input)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *CreateNamespaceMembershipInput) *models.NamespaceMembership); ok {
		r0 = rf(ctx, input)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.NamespaceMembership)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *CreateNamespaceMembershipInput) error); ok {
		r1 = rf(ctx, input)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DeleteNamespaceMembership provides a mock function with given fields: ctx, namespaceMembership
func (_m *MockNamespaceMemberships) DeleteNamespaceMembership(ctx context.Context, namespaceMembership *models.NamespaceMembership) error {
	ret := _m.Called(ctx, namespaceMembership)

	if len(ret) == 0 {
		panic("no return value specified for DeleteNamespaceMembership")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *models.NamespaceMembership) error); ok {
		r0 = rf(ctx, namespaceMembership)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetNamespaceMembershipByID provides a mock function with given fields: ctx, id
func (_m *MockNamespaceMemberships) GetNamespaceMembershipByID(ctx context.Context, id string) (*models.NamespaceMembership, error) {
	ret := _m.Called(ctx, id)

	if len(ret) == 0 {
		panic("no return value specified for GetNamespaceMembershipByID")
	}

	var r0 *models.NamespaceMembership
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*models.NamespaceMembership, error)); ok {
		return rf(ctx, id)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *models.NamespaceMembership); ok {
		r0 = rf(ctx, id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.NamespaceMembership)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetNamespaceMembershipByTRN provides a mock function with given fields: ctx, trn
func (_m *MockNamespaceMemberships) GetNamespaceMembershipByTRN(ctx context.Context, trn string) (*models.NamespaceMembership, error) {
	ret := _m.Called(ctx, trn)

	if len(ret) == 0 {
		panic("no return value specified for GetNamespaceMembershipByTRN")
	}

	var r0 *models.NamespaceMembership
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*models.NamespaceMembership, error)); ok {
		return rf(ctx, trn)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *models.NamespaceMembership); ok {
		r0 = rf(ctx, trn)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.NamespaceMembership)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, trn)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetNamespaceMemberships provides a mock function with given fields: ctx, input
func (_m *MockNamespaceMemberships) GetNamespaceMemberships(ctx context.Context, input *GetNamespaceMembershipsInput) (*NamespaceMembershipResult, error) {
	ret := _m.Called(ctx, input)

	if len(ret) == 0 {
		panic("no return value specified for GetNamespaceMemberships")
	}

	var r0 *NamespaceMembershipResult
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *GetNamespaceMembershipsInput) (*NamespaceMembershipResult, error)); ok {
		return rf(ctx, input)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *GetNamespaceMembershipsInput) *NamespaceMembershipResult); ok {
		r0 = rf(ctx, input)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*NamespaceMembershipResult)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *GetNamespaceMembershipsInput) error); ok {
		r1 = rf(ctx, input)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// UpdateNamespaceMembership provides a mock function with given fields: ctx, namespaceMembership
func (_m *MockNamespaceMemberships) UpdateNamespaceMembership(ctx context.Context, namespaceMembership *models.NamespaceMembership) (*models.NamespaceMembership, error) {
	ret := _m.Called(ctx, namespaceMembership)

	if len(ret) == 0 {
		panic("no return value specified for UpdateNamespaceMembership")
	}

	var r0 *models.NamespaceMembership
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *models.NamespaceMembership) (*models.NamespaceMembership, error)); ok {
		return rf(ctx, namespaceMembership)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *models.NamespaceMembership) *models.NamespaceMembership); ok {
		r0 = rf(ctx, namespaceMembership)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.NamespaceMembership)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *models.NamespaceMembership) error); ok {
		r1 = rf(ctx, namespaceMembership)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewMockNamespaceMemberships creates a new instance of MockNamespaceMemberships. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockNamespaceMemberships(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockNamespaceMemberships {
	mock := &MockNamespaceMemberships{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
