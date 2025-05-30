// Code generated by mockery v2.53.0. DO NOT EDIT.

package jobclient

import (
	context "context"
	io "io"

	mock "github.com/stretchr/testify/mock"

	tfjson "github.com/hashicorp/terraform-json"

	time "time"

	types "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/types"
)

// MockClient is an autogenerated mock type for the Client type
type MockClient struct {
	mock.Mock
}

// Close provides a mock function with no fields
func (_m *MockClient) Close() error {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Close")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CreateFederatedRegistryTokens provides a mock function with given fields: ctx, input
func (_m *MockClient) CreateFederatedRegistryTokens(ctx context.Context, input *types.CreateFederatedRegistryTokensInput) ([]types.FederatedRegistryToken, error) {
	ret := _m.Called(ctx, input)

	if len(ret) == 0 {
		panic("no return value specified for CreateFederatedRegistryTokens")
	}

	var r0 []types.FederatedRegistryToken
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *types.CreateFederatedRegistryTokensInput) ([]types.FederatedRegistryToken, error)); ok {
		return rf(ctx, input)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *types.CreateFederatedRegistryTokensInput) []types.FederatedRegistryToken); ok {
		r0 = rf(ctx, input)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]types.FederatedRegistryToken)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *types.CreateFederatedRegistryTokensInput) error); ok {
		r1 = rf(ctx, input)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CreateManagedIdentityCredentials provides a mock function with given fields: ctx, managedIdentityID
func (_m *MockClient) CreateManagedIdentityCredentials(ctx context.Context, managedIdentityID string) ([]byte, error) {
	ret := _m.Called(ctx, managedIdentityID)

	if len(ret) == 0 {
		panic("no return value specified for CreateManagedIdentityCredentials")
	}

	var r0 []byte
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) ([]byte, error)); ok {
		return rf(ctx, managedIdentityID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) []byte); ok {
		r0 = rf(ctx, managedIdentityID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, managedIdentityID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CreateServiceAccountToken provides a mock function with given fields: ctx, serviceAccountPath, token
func (_m *MockClient) CreateServiceAccountToken(ctx context.Context, serviceAccountPath string, token string) (string, *time.Duration, error) {
	ret := _m.Called(ctx, serviceAccountPath, token)

	if len(ret) == 0 {
		panic("no return value specified for CreateServiceAccountToken")
	}

	var r0 string
	var r1 *time.Duration
	var r2 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) (string, *time.Duration, error)); ok {
		return rf(ctx, serviceAccountPath, token)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string) string); ok {
		r0 = rf(ctx, serviceAccountPath, token)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string) *time.Duration); ok {
		r1 = rf(ctx, serviceAccountPath, token)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*time.Duration)
		}
	}

	if rf, ok := ret.Get(2).(func(context.Context, string, string) error); ok {
		r2 = rf(ctx, serviceAccountPath, token)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// CreateStateVersion provides a mock function with given fields: ctx, runID, body
func (_m *MockClient) CreateStateVersion(ctx context.Context, runID string, body io.Reader) (*types.StateVersion, error) {
	ret := _m.Called(ctx, runID, body)

	if len(ret) == 0 {
		panic("no return value specified for CreateStateVersion")
	}

	var r0 *types.StateVersion
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, io.Reader) (*types.StateVersion, error)); ok {
		return rf(ctx, runID, body)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, io.Reader) *types.StateVersion); ok {
		r0 = rf(ctx, runID, body)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.StateVersion)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, io.Reader) error); ok {
		r1 = rf(ctx, runID, body)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CreateTerraformCLIDownloadURL provides a mock function with given fields: ctx, input
func (_m *MockClient) CreateTerraformCLIDownloadURL(ctx context.Context, input *types.CreateTerraformCLIDownloadURLInput) (string, error) {
	ret := _m.Called(ctx, input)

	if len(ret) == 0 {
		panic("no return value specified for CreateTerraformCLIDownloadURL")
	}

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *types.CreateTerraformCLIDownloadURLInput) (string, error)); ok {
		return rf(ctx, input)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *types.CreateTerraformCLIDownloadURLInput) string); ok {
		r0 = rf(ctx, input)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(context.Context, *types.CreateTerraformCLIDownloadURLInput) error); ok {
		r1 = rf(ctx, input)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DownloadConfigurationVersion provides a mock function with given fields: ctx, configurationVersion, writer
func (_m *MockClient) DownloadConfigurationVersion(ctx context.Context, configurationVersion *types.ConfigurationVersion, writer io.WriterAt) error {
	ret := _m.Called(ctx, configurationVersion, writer)

	if len(ret) == 0 {
		panic("no return value specified for DownloadConfigurationVersion")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *types.ConfigurationVersion, io.WriterAt) error); ok {
		r0 = rf(ctx, configurationVersion, writer)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DownloadPlanCache provides a mock function with given fields: ctx, planID, writer
func (_m *MockClient) DownloadPlanCache(ctx context.Context, planID string, writer io.WriterAt) error {
	ret := _m.Called(ctx, planID, writer)

	if len(ret) == 0 {
		panic("no return value specified for DownloadPlanCache")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, io.WriterAt) error); ok {
		r0 = rf(ctx, planID, writer)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DownloadStateVersion provides a mock function with given fields: ctx, stateVersion, writer
func (_m *MockClient) DownloadStateVersion(ctx context.Context, stateVersion *types.StateVersion, writer io.WriterAt) error {
	ret := _m.Called(ctx, stateVersion, writer)

	if len(ret) == 0 {
		panic("no return value specified for DownloadStateVersion")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *types.StateVersion, io.WriterAt) error); ok {
		r0 = rf(ctx, stateVersion, writer)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetAssignedManagedIdentities provides a mock function with given fields: ctx, workspaceID
func (_m *MockClient) GetAssignedManagedIdentities(ctx context.Context, workspaceID string) ([]types.ManagedIdentity, error) {
	ret := _m.Called(ctx, workspaceID)

	if len(ret) == 0 {
		panic("no return value specified for GetAssignedManagedIdentities")
	}

	var r0 []types.ManagedIdentity
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) ([]types.ManagedIdentity, error)); ok {
		return rf(ctx, workspaceID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) []types.ManagedIdentity); ok {
		r0 = rf(ctx, workspaceID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]types.ManagedIdentity)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, workspaceID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetConfigurationVersion provides a mock function with given fields: ctx, id
func (_m *MockClient) GetConfigurationVersion(ctx context.Context, id string) (*types.ConfigurationVersion, error) {
	ret := _m.Called(ctx, id)

	if len(ret) == 0 {
		panic("no return value specified for GetConfigurationVersion")
	}

	var r0 *types.ConfigurationVersion
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*types.ConfigurationVersion, error)); ok {
		return rf(ctx, id)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *types.ConfigurationVersion); ok {
		r0 = rf(ctx, id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.ConfigurationVersion)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetJob provides a mock function with given fields: ctx, id
func (_m *MockClient) GetJob(ctx context.Context, id string) (*types.Job, error) {
	ret := _m.Called(ctx, id)

	if len(ret) == 0 {
		panic("no return value specified for GetJob")
	}

	var r0 *types.Job
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*types.Job, error)); ok {
		return rf(ctx, id)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *types.Job); ok {
		r0 = rf(ctx, id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Job)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetRun provides a mock function with given fields: ctx, id
func (_m *MockClient) GetRun(ctx context.Context, id string) (*types.Run, error) {
	ret := _m.Called(ctx, id)

	if len(ret) == 0 {
		panic("no return value specified for GetRun")
	}

	var r0 *types.Run
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*types.Run, error)); ok {
		return rf(ctx, id)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *types.Run); ok {
		r0 = rf(ctx, id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Run)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetRunVariables provides a mock function with given fields: ctx, runID
func (_m *MockClient) GetRunVariables(ctx context.Context, runID string) ([]types.RunVariable, error) {
	ret := _m.Called(ctx, runID)

	if len(ret) == 0 {
		panic("no return value specified for GetRunVariables")
	}

	var r0 []types.RunVariable
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) ([]types.RunVariable, error)); ok {
		return rf(ctx, runID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) []types.RunVariable); ok {
		r0 = rf(ctx, runID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]types.RunVariable)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, runID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetWorkspace provides a mock function with given fields: ctx, id
func (_m *MockClient) GetWorkspace(ctx context.Context, id string) (*types.Workspace, error) {
	ret := _m.Called(ctx, id)

	if len(ret) == 0 {
		panic("no return value specified for GetWorkspace")
	}

	var r0 *types.Workspace
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*types.Workspace, error)); ok {
		return rf(ctx, id)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *types.Workspace); ok {
		r0 = rf(ctx, id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Workspace)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SaveJobLogs provides a mock function with given fields: ctx, jobID, startOffset, buffer
func (_m *MockClient) SaveJobLogs(ctx context.Context, jobID string, startOffset int, buffer []byte) error {
	ret := _m.Called(ctx, jobID, startOffset, buffer)

	if len(ret) == 0 {
		panic("no return value specified for SaveJobLogs")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, int, []byte) error); ok {
		r0 = rf(ctx, jobID, startOffset, buffer)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SetVariablesIncludedInTFConfig provides a mock function with given fields: ctx, runID, variableKeys
func (_m *MockClient) SetVariablesIncludedInTFConfig(ctx context.Context, runID string, variableKeys []string) error {
	ret := _m.Called(ctx, runID, variableKeys)

	if len(ret) == 0 {
		panic("no return value specified for SetVariablesIncludedInTFConfig")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, []string) error); ok {
		r0 = rf(ctx, runID, variableKeys)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SubscribeToJobCancellationEvent provides a mock function with given fields: ctx, jobID
func (_m *MockClient) SubscribeToJobCancellationEvent(ctx context.Context, jobID string) (<-chan *types.CancellationEvent, error) {
	ret := _m.Called(ctx, jobID)

	if len(ret) == 0 {
		panic("no return value specified for SubscribeToJobCancellationEvent")
	}

	var r0 <-chan *types.CancellationEvent
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (<-chan *types.CancellationEvent, error)); ok {
		return rf(ctx, jobID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) <-chan *types.CancellationEvent); ok {
		r0 = rf(ctx, jobID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(<-chan *types.CancellationEvent)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, jobID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// UpdateApply provides a mock function with given fields: ctx, apply
func (_m *MockClient) UpdateApply(ctx context.Context, apply *types.Apply) (*types.Apply, error) {
	ret := _m.Called(ctx, apply)

	if len(ret) == 0 {
		panic("no return value specified for UpdateApply")
	}

	var r0 *types.Apply
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *types.Apply) (*types.Apply, error)); ok {
		return rf(ctx, apply)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *types.Apply) *types.Apply); ok {
		r0 = rf(ctx, apply)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Apply)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *types.Apply) error); ok {
		r1 = rf(ctx, apply)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// UpdatePlan provides a mock function with given fields: ctx, apply
func (_m *MockClient) UpdatePlan(ctx context.Context, apply *types.Plan) (*types.Plan, error) {
	ret := _m.Called(ctx, apply)

	if len(ret) == 0 {
		panic("no return value specified for UpdatePlan")
	}

	var r0 *types.Plan
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *types.Plan) (*types.Plan, error)); ok {
		return rf(ctx, apply)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *types.Plan) *types.Plan); ok {
		r0 = rf(ctx, apply)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Plan)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *types.Plan) error); ok {
		r1 = rf(ctx, apply)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// UploadPlanCache provides a mock function with given fields: ctx, plan, body
func (_m *MockClient) UploadPlanCache(ctx context.Context, plan *types.Plan, body io.Reader) error {
	ret := _m.Called(ctx, plan, body)

	if len(ret) == 0 {
		panic("no return value specified for UploadPlanCache")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *types.Plan, io.Reader) error); ok {
		r0 = rf(ctx, plan, body)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UploadPlanData provides a mock function with given fields: ctx, plan, tfPlan, tfProviderScheams
func (_m *MockClient) UploadPlanData(ctx context.Context, plan *types.Plan, tfPlan *tfjson.Plan, tfProviderScheams *tfjson.ProviderSchemas) error {
	ret := _m.Called(ctx, plan, tfPlan, tfProviderScheams)

	if len(ret) == 0 {
		panic("no return value specified for UploadPlanData")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *types.Plan, *tfjson.Plan, *tfjson.ProviderSchemas) error); ok {
		r0 = rf(ctx, plan, tfPlan, tfProviderScheams)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewMockClient creates a new instance of MockClient. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockClient(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockClient {
	mock := &MockClient{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
