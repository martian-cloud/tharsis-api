// Code generated by mockery v2.20.0. DO NOT EDIT.

package db

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
	models "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// MockApplies is an autogenerated mock type for the Applies type
type MockApplies struct {
	mock.Mock
}

// CreateApply provides a mock function with given fields: ctx, apply
func (_m *MockApplies) CreateApply(ctx context.Context, apply *models.Apply) (*models.Apply, error) {
	ret := _m.Called(ctx, apply)

	var r0 *models.Apply
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *models.Apply) (*models.Apply, error)); ok {
		return rf(ctx, apply)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *models.Apply) *models.Apply); ok {
		r0 = rf(ctx, apply)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.Apply)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *models.Apply) error); ok {
		r1 = rf(ctx, apply)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetApplies provides a mock function with given fields: ctx, input
func (_m *MockApplies) GetApplies(ctx context.Context, input *GetAppliesInput) (*AppliesResult, error) {
	ret := _m.Called(ctx, input)

	var r0 *AppliesResult
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *GetAppliesInput) (*AppliesResult, error)); ok {
		return rf(ctx, input)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *GetAppliesInput) *AppliesResult); ok {
		r0 = rf(ctx, input)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*AppliesResult)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *GetAppliesInput) error); ok {
		r1 = rf(ctx, input)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetApply provides a mock function with given fields: ctx, id
func (_m *MockApplies) GetApply(ctx context.Context, id string) (*models.Apply, error) {
	ret := _m.Called(ctx, id)

	var r0 *models.Apply
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*models.Apply, error)); ok {
		return rf(ctx, id)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *models.Apply); ok {
		r0 = rf(ctx, id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.Apply)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// UpdateApply provides a mock function with given fields: ctx, apply
func (_m *MockApplies) UpdateApply(ctx context.Context, apply *models.Apply) (*models.Apply, error) {
	ret := _m.Called(ctx, apply)

	var r0 *models.Apply
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *models.Apply) (*models.Apply, error)); ok {
		return rf(ctx, apply)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *models.Apply) *models.Apply); ok {
		r0 = rf(ctx, apply)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.Apply)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *models.Apply) error); ok {
		r1 = rf(ctx, apply)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewMockApplies interface {
	mock.TestingT
	Cleanup(func())
}

// NewMockApplies creates a new instance of MockApplies. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewMockApplies(t mockConstructorTestingTNewMockApplies) *MockApplies {
	mock := &MockApplies{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
