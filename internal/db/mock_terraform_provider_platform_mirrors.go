// Code generated by mockery v2.20.0. DO NOT EDIT.

package db

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
	models "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// MockTerraformProviderPlatformMirrors is an autogenerated mock type for the TerraformProviderPlatformMirrors type
type MockTerraformProviderPlatformMirrors struct {
	mock.Mock
}

// CreatePlatformMirror provides a mock function with given fields: ctx, platformMirror
func (_m *MockTerraformProviderPlatformMirrors) CreatePlatformMirror(ctx context.Context, platformMirror *models.TerraformProviderPlatformMirror) (*models.TerraformProviderPlatformMirror, error) {
	ret := _m.Called(ctx, platformMirror)

	var r0 *models.TerraformProviderPlatformMirror
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *models.TerraformProviderPlatformMirror) (*models.TerraformProviderPlatformMirror, error)); ok {
		return rf(ctx, platformMirror)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *models.TerraformProviderPlatformMirror) *models.TerraformProviderPlatformMirror); ok {
		r0 = rf(ctx, platformMirror)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.TerraformProviderPlatformMirror)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *models.TerraformProviderPlatformMirror) error); ok {
		r1 = rf(ctx, platformMirror)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DeletePlatformMirror provides a mock function with given fields: ctx, platformMirror
func (_m *MockTerraformProviderPlatformMirrors) DeletePlatformMirror(ctx context.Context, platformMirror *models.TerraformProviderPlatformMirror) error {
	ret := _m.Called(ctx, platformMirror)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *models.TerraformProviderPlatformMirror) error); ok {
		r0 = rf(ctx, platformMirror)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetPlatformMirrorByID provides a mock function with given fields: ctx, id
func (_m *MockTerraformProviderPlatformMirrors) GetPlatformMirrorByID(ctx context.Context, id string) (*models.TerraformProviderPlatformMirror, error) {
	ret := _m.Called(ctx, id)

	var r0 *models.TerraformProviderPlatformMirror
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*models.TerraformProviderPlatformMirror, error)); ok {
		return rf(ctx, id)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *models.TerraformProviderPlatformMirror); ok {
		r0 = rf(ctx, id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.TerraformProviderPlatformMirror)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetPlatformMirrors provides a mock function with given fields: ctx, input
func (_m *MockTerraformProviderPlatformMirrors) GetPlatformMirrors(ctx context.Context, input *GetProviderPlatformMirrorsInput) (*ProviderPlatformMirrorsResult, error) {
	ret := _m.Called(ctx, input)

	var r0 *ProviderPlatformMirrorsResult
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *GetProviderPlatformMirrorsInput) (*ProviderPlatformMirrorsResult, error)); ok {
		return rf(ctx, input)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *GetProviderPlatformMirrorsInput) *ProviderPlatformMirrorsResult); ok {
		r0 = rf(ctx, input)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*ProviderPlatformMirrorsResult)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *GetProviderPlatformMirrorsInput) error); ok {
		r1 = rf(ctx, input)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewMockTerraformProviderPlatformMirrors interface {
	mock.TestingT
	Cleanup(func())
}

// NewMockTerraformProviderPlatformMirrors creates a new instance of MockTerraformProviderPlatformMirrors. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewMockTerraformProviderPlatformMirrors(t mockConstructorTestingTNewMockTerraformProviderPlatformMirrors) *MockTerraformProviderPlatformMirrors {
	mock := &MockTerraformProviderPlatformMirrors{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}