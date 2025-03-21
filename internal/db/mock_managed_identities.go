// Code generated by mockery v2.53.0. DO NOT EDIT.

package db

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
	models "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// MockManagedIdentities is an autogenerated mock type for the ManagedIdentities type
type MockManagedIdentities struct {
	mock.Mock
}

// AddManagedIdentityToWorkspace provides a mock function with given fields: ctx, managedIdentityID, workspaceID
func (_m *MockManagedIdentities) AddManagedIdentityToWorkspace(ctx context.Context, managedIdentityID string, workspaceID string) error {
	ret := _m.Called(ctx, managedIdentityID, workspaceID)

	if len(ret) == 0 {
		panic("no return value specified for AddManagedIdentityToWorkspace")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) error); ok {
		r0 = rf(ctx, managedIdentityID, workspaceID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CreateManagedIdentity provides a mock function with given fields: ctx, managedIdentity
func (_m *MockManagedIdentities) CreateManagedIdentity(ctx context.Context, managedIdentity *models.ManagedIdentity) (*models.ManagedIdentity, error) {
	ret := _m.Called(ctx, managedIdentity)

	if len(ret) == 0 {
		panic("no return value specified for CreateManagedIdentity")
	}

	var r0 *models.ManagedIdentity
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *models.ManagedIdentity) (*models.ManagedIdentity, error)); ok {
		return rf(ctx, managedIdentity)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *models.ManagedIdentity) *models.ManagedIdentity); ok {
		r0 = rf(ctx, managedIdentity)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.ManagedIdentity)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *models.ManagedIdentity) error); ok {
		r1 = rf(ctx, managedIdentity)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CreateManagedIdentityAccessRule provides a mock function with given fields: ctx, rule
func (_m *MockManagedIdentities) CreateManagedIdentityAccessRule(ctx context.Context, rule *models.ManagedIdentityAccessRule) (*models.ManagedIdentityAccessRule, error) {
	ret := _m.Called(ctx, rule)

	if len(ret) == 0 {
		panic("no return value specified for CreateManagedIdentityAccessRule")
	}

	var r0 *models.ManagedIdentityAccessRule
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *models.ManagedIdentityAccessRule) (*models.ManagedIdentityAccessRule, error)); ok {
		return rf(ctx, rule)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *models.ManagedIdentityAccessRule) *models.ManagedIdentityAccessRule); ok {
		r0 = rf(ctx, rule)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.ManagedIdentityAccessRule)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *models.ManagedIdentityAccessRule) error); ok {
		r1 = rf(ctx, rule)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DeleteManagedIdentity provides a mock function with given fields: ctx, managedIdentity
func (_m *MockManagedIdentities) DeleteManagedIdentity(ctx context.Context, managedIdentity *models.ManagedIdentity) error {
	ret := _m.Called(ctx, managedIdentity)

	if len(ret) == 0 {
		panic("no return value specified for DeleteManagedIdentity")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *models.ManagedIdentity) error); ok {
		r0 = rf(ctx, managedIdentity)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeleteManagedIdentityAccessRule provides a mock function with given fields: ctx, rule
func (_m *MockManagedIdentities) DeleteManagedIdentityAccessRule(ctx context.Context, rule *models.ManagedIdentityAccessRule) error {
	ret := _m.Called(ctx, rule)

	if len(ret) == 0 {
		panic("no return value specified for DeleteManagedIdentityAccessRule")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *models.ManagedIdentityAccessRule) error); ok {
		r0 = rf(ctx, rule)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetManagedIdentities provides a mock function with given fields: ctx, input
func (_m *MockManagedIdentities) GetManagedIdentities(ctx context.Context, input *GetManagedIdentitiesInput) (*ManagedIdentitiesResult, error) {
	ret := _m.Called(ctx, input)

	if len(ret) == 0 {
		panic("no return value specified for GetManagedIdentities")
	}

	var r0 *ManagedIdentitiesResult
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *GetManagedIdentitiesInput) (*ManagedIdentitiesResult, error)); ok {
		return rf(ctx, input)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *GetManagedIdentitiesInput) *ManagedIdentitiesResult); ok {
		r0 = rf(ctx, input)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*ManagedIdentitiesResult)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *GetManagedIdentitiesInput) error); ok {
		r1 = rf(ctx, input)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetManagedIdentitiesForWorkspace provides a mock function with given fields: ctx, workspaceID
func (_m *MockManagedIdentities) GetManagedIdentitiesForWorkspace(ctx context.Context, workspaceID string) ([]models.ManagedIdentity, error) {
	ret := _m.Called(ctx, workspaceID)

	if len(ret) == 0 {
		panic("no return value specified for GetManagedIdentitiesForWorkspace")
	}

	var r0 []models.ManagedIdentity
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) ([]models.ManagedIdentity, error)); ok {
		return rf(ctx, workspaceID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) []models.ManagedIdentity); ok {
		r0 = rf(ctx, workspaceID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]models.ManagedIdentity)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, workspaceID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetManagedIdentityAccessRule provides a mock function with given fields: ctx, ruleID
func (_m *MockManagedIdentities) GetManagedIdentityAccessRule(ctx context.Context, ruleID string) (*models.ManagedIdentityAccessRule, error) {
	ret := _m.Called(ctx, ruleID)

	if len(ret) == 0 {
		panic("no return value specified for GetManagedIdentityAccessRule")
	}

	var r0 *models.ManagedIdentityAccessRule
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*models.ManagedIdentityAccessRule, error)); ok {
		return rf(ctx, ruleID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *models.ManagedIdentityAccessRule); ok {
		r0 = rf(ctx, ruleID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.ManagedIdentityAccessRule)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, ruleID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetManagedIdentityAccessRules provides a mock function with given fields: ctx, input
func (_m *MockManagedIdentities) GetManagedIdentityAccessRules(ctx context.Context, input *GetManagedIdentityAccessRulesInput) (*ManagedIdentityAccessRulesResult, error) {
	ret := _m.Called(ctx, input)

	if len(ret) == 0 {
		panic("no return value specified for GetManagedIdentityAccessRules")
	}

	var r0 *ManagedIdentityAccessRulesResult
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *GetManagedIdentityAccessRulesInput) (*ManagedIdentityAccessRulesResult, error)); ok {
		return rf(ctx, input)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *GetManagedIdentityAccessRulesInput) *ManagedIdentityAccessRulesResult); ok {
		r0 = rf(ctx, input)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*ManagedIdentityAccessRulesResult)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *GetManagedIdentityAccessRulesInput) error); ok {
		r1 = rf(ctx, input)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetManagedIdentityByID provides a mock function with given fields: ctx, id
func (_m *MockManagedIdentities) GetManagedIdentityByID(ctx context.Context, id string) (*models.ManagedIdentity, error) {
	ret := _m.Called(ctx, id)

	if len(ret) == 0 {
		panic("no return value specified for GetManagedIdentityByID")
	}

	var r0 *models.ManagedIdentity
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*models.ManagedIdentity, error)); ok {
		return rf(ctx, id)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *models.ManagedIdentity); ok {
		r0 = rf(ctx, id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.ManagedIdentity)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetManagedIdentityByPath provides a mock function with given fields: ctx, path
func (_m *MockManagedIdentities) GetManagedIdentityByPath(ctx context.Context, path string) (*models.ManagedIdentity, error) {
	ret := _m.Called(ctx, path)

	if len(ret) == 0 {
		panic("no return value specified for GetManagedIdentityByPath")
	}

	var r0 *models.ManagedIdentity
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*models.ManagedIdentity, error)); ok {
		return rf(ctx, path)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *models.ManagedIdentity); ok {
		r0 = rf(ctx, path)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.ManagedIdentity)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, path)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// RemoveManagedIdentityFromWorkspace provides a mock function with given fields: ctx, managedIdentityID, workspaceID
func (_m *MockManagedIdentities) RemoveManagedIdentityFromWorkspace(ctx context.Context, managedIdentityID string, workspaceID string) error {
	ret := _m.Called(ctx, managedIdentityID, workspaceID)

	if len(ret) == 0 {
		panic("no return value specified for RemoveManagedIdentityFromWorkspace")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) error); ok {
		r0 = rf(ctx, managedIdentityID, workspaceID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpdateManagedIdentity provides a mock function with given fields: ctx, managedIdentity
func (_m *MockManagedIdentities) UpdateManagedIdentity(ctx context.Context, managedIdentity *models.ManagedIdentity) (*models.ManagedIdentity, error) {
	ret := _m.Called(ctx, managedIdentity)

	if len(ret) == 0 {
		panic("no return value specified for UpdateManagedIdentity")
	}

	var r0 *models.ManagedIdentity
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *models.ManagedIdentity) (*models.ManagedIdentity, error)); ok {
		return rf(ctx, managedIdentity)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *models.ManagedIdentity) *models.ManagedIdentity); ok {
		r0 = rf(ctx, managedIdentity)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.ManagedIdentity)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *models.ManagedIdentity) error); ok {
		r1 = rf(ctx, managedIdentity)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// UpdateManagedIdentityAccessRule provides a mock function with given fields: ctx, rule
func (_m *MockManagedIdentities) UpdateManagedIdentityAccessRule(ctx context.Context, rule *models.ManagedIdentityAccessRule) (*models.ManagedIdentityAccessRule, error) {
	ret := _m.Called(ctx, rule)

	if len(ret) == 0 {
		panic("no return value specified for UpdateManagedIdentityAccessRule")
	}

	var r0 *models.ManagedIdentityAccessRule
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *models.ManagedIdentityAccessRule) (*models.ManagedIdentityAccessRule, error)); ok {
		return rf(ctx, rule)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *models.ManagedIdentityAccessRule) *models.ManagedIdentityAccessRule); ok {
		r0 = rf(ctx, rule)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.ManagedIdentityAccessRule)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *models.ManagedIdentityAccessRule) error); ok {
		r1 = rf(ctx, rule)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewMockManagedIdentities creates a new instance of MockManagedIdentities. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockManagedIdentities(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockManagedIdentities {
	mock := &MockManagedIdentities{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
