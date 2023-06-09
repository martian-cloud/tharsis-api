// Code generated by mockery v2.20.0. DO NOT EDIT.

package db

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
	models "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// MockGroups is an autogenerated mock type for the Groups type
type MockGroups struct {
	mock.Mock
}

// CreateGroup provides a mock function with given fields: ctx, group
func (_m *MockGroups) CreateGroup(ctx context.Context, group *models.Group) (*models.Group, error) {
	ret := _m.Called(ctx, group)

	var r0 *models.Group
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *models.Group) (*models.Group, error)); ok {
		return rf(ctx, group)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *models.Group) *models.Group); ok {
		r0 = rf(ctx, group)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.Group)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *models.Group) error); ok {
		r1 = rf(ctx, group)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DeleteGroup provides a mock function with given fields: ctx, group
func (_m *MockGroups) DeleteGroup(ctx context.Context, group *models.Group) error {
	ret := _m.Called(ctx, group)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *models.Group) error); ok {
		r0 = rf(ctx, group)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetChildDepth provides a mock function with given fields: ctx, group
func (_m *MockGroups) GetChildDepth(ctx context.Context, group *models.Group) (int, error) {
	ret := _m.Called(ctx, group)

	var r0 int
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *models.Group) (int, error)); ok {
		return rf(ctx, group)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *models.Group) int); ok {
		r0 = rf(ctx, group)
	} else {
		r0 = ret.Get(0).(int)
	}

	if rf, ok := ret.Get(1).(func(context.Context, *models.Group) error); ok {
		r1 = rf(ctx, group)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetGroupByFullPath provides a mock function with given fields: ctx, path
func (_m *MockGroups) GetGroupByFullPath(ctx context.Context, path string) (*models.Group, error) {
	ret := _m.Called(ctx, path)

	var r0 *models.Group
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*models.Group, error)); ok {
		return rf(ctx, path)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *models.Group); ok {
		r0 = rf(ctx, path)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.Group)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, path)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetGroupByID provides a mock function with given fields: ctx, id
func (_m *MockGroups) GetGroupByID(ctx context.Context, id string) (*models.Group, error) {
	ret := _m.Called(ctx, id)

	var r0 *models.Group
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*models.Group, error)); ok {
		return rf(ctx, id)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *models.Group); ok {
		r0 = rf(ctx, id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.Group)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetGroups provides a mock function with given fields: ctx, input
func (_m *MockGroups) GetGroups(ctx context.Context, input *GetGroupsInput) (*GroupsResult, error) {
	ret := _m.Called(ctx, input)

	var r0 *GroupsResult
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *GetGroupsInput) (*GroupsResult, error)); ok {
		return rf(ctx, input)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *GetGroupsInput) *GroupsResult); ok {
		r0 = rf(ctx, input)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*GroupsResult)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *GetGroupsInput) error); ok {
		r1 = rf(ctx, input)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MigrateGroup provides a mock function with given fields: ctx, group, newParentGroup
func (_m *MockGroups) MigrateGroup(ctx context.Context, group *models.Group, newParentGroup *models.Group) (*models.Group, error) {
	ret := _m.Called(ctx, group, newParentGroup)

	var r0 *models.Group
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *models.Group, *models.Group) (*models.Group, error)); ok {
		return rf(ctx, group, newParentGroup)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *models.Group, *models.Group) *models.Group); ok {
		r0 = rf(ctx, group, newParentGroup)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.Group)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *models.Group, *models.Group) error); ok {
		r1 = rf(ctx, group, newParentGroup)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// UpdateGroup provides a mock function with given fields: ctx, group
func (_m *MockGroups) UpdateGroup(ctx context.Context, group *models.Group) (*models.Group, error) {
	ret := _m.Called(ctx, group)

	var r0 *models.Group
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *models.Group) (*models.Group, error)); ok {
		return rf(ctx, group)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *models.Group) *models.Group); ok {
		r0 = rf(ctx, group)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.Group)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *models.Group) error); ok {
		r1 = rf(ctx, group)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewMockGroups interface {
	mock.TestingT
	Cleanup(func())
}

// NewMockGroups creates a new instance of MockGroups. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewMockGroups(t mockConstructorTestingTNewMockGroups) *MockGroups {
	mock := &MockGroups{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
