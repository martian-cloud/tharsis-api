// Code generated by mockery v2.53.0. DO NOT EDIT.

package db

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
	models "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// MockResourceLimits is an autogenerated mock type for the ResourceLimits type
type MockResourceLimits struct {
	mock.Mock
}

// GetResourceLimit provides a mock function with given fields: ctx, name
func (_m *MockResourceLimits) GetResourceLimit(ctx context.Context, name string) (*models.ResourceLimit, error) {
	ret := _m.Called(ctx, name)

	if len(ret) == 0 {
		panic("no return value specified for GetResourceLimit")
	}

	var r0 *models.ResourceLimit
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*models.ResourceLimit, error)); ok {
		return rf(ctx, name)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *models.ResourceLimit); ok {
		r0 = rf(ctx, name)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.ResourceLimit)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetResourceLimits provides a mock function with given fields: ctx
func (_m *MockResourceLimits) GetResourceLimits(ctx context.Context) ([]models.ResourceLimit, error) {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for GetResourceLimits")
	}

	var r0 []models.ResourceLimit
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) ([]models.ResourceLimit, error)); ok {
		return rf(ctx)
	}
	if rf, ok := ret.Get(0).(func(context.Context) []models.ResourceLimit); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]models.ResourceLimit)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// UpdateResourceLimit provides a mock function with given fields: ctx, resourceLimit
func (_m *MockResourceLimits) UpdateResourceLimit(ctx context.Context, resourceLimit *models.ResourceLimit) (*models.ResourceLimit, error) {
	ret := _m.Called(ctx, resourceLimit)

	if len(ret) == 0 {
		panic("no return value specified for UpdateResourceLimit")
	}

	var r0 *models.ResourceLimit
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *models.ResourceLimit) (*models.ResourceLimit, error)); ok {
		return rf(ctx, resourceLimit)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *models.ResourceLimit) *models.ResourceLimit); ok {
		r0 = rf(ctx, resourceLimit)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.ResourceLimit)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *models.ResourceLimit) error); ok {
		r1 = rf(ctx, resourceLimit)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewMockResourceLimits creates a new instance of MockResourceLimits. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockResourceLimits(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockResourceLimits {
	mock := &MockResourceLimits{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
