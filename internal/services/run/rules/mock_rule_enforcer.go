// Code generated by mockery v2.53.0. DO NOT EDIT.

package rules

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
	models "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// MockRuleEnforcer is an autogenerated mock type for the RuleEnforcer type
type MockRuleEnforcer struct {
	mock.Mock
}

// EnforceRules provides a mock function with given fields: ctx, managedIdentity, input
func (_m *MockRuleEnforcer) EnforceRules(ctx context.Context, managedIdentity *models.ManagedIdentity, input *RunDetails) error {
	ret := _m.Called(ctx, managedIdentity, input)

	if len(ret) == 0 {
		panic("no return value specified for EnforceRules")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *models.ManagedIdentity, *RunDetails) error); ok {
		r0 = rf(ctx, managedIdentity, input)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewMockRuleEnforcer creates a new instance of MockRuleEnforcer. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockRuleEnforcer(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockRuleEnforcer {
	mock := &MockRuleEnforcer{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
