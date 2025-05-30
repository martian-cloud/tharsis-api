// Code generated by mockery v2.53.0. DO NOT EDIT.

package run

import (
	context "context"
	io "io"

	db "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"

	mock "github.com/stretchr/testify/mock"

	models "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"

	plan "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan"

	tfjson "github.com/hashicorp/terraform-json"
)

// MockService is an autogenerated mock type for the Service type
type MockService struct {
	mock.Mock
}

// ApplyRun provides a mock function with given fields: ctx, runID, comment
func (_m *MockService) ApplyRun(ctx context.Context, runID string, comment *string) (*models.Run, error) {
	ret := _m.Called(ctx, runID, comment)

	if len(ret) == 0 {
		panic("no return value specified for ApplyRun")
	}

	var r0 *models.Run
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, *string) (*models.Run, error)); ok {
		return rf(ctx, runID, comment)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, *string) *models.Run); ok {
		r0 = rf(ctx, runID, comment)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.Run)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, *string) error); ok {
		r1 = rf(ctx, runID, comment)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CancelRun provides a mock function with given fields: ctx, options
func (_m *MockService) CancelRun(ctx context.Context, options *CancelRunInput) (*models.Run, error) {
	ret := _m.Called(ctx, options)

	if len(ret) == 0 {
		panic("no return value specified for CancelRun")
	}

	var r0 *models.Run
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *CancelRunInput) (*models.Run, error)); ok {
		return rf(ctx, options)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *CancelRunInput) *models.Run); ok {
		r0 = rf(ctx, options)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.Run)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *CancelRunInput) error); ok {
		r1 = rf(ctx, options)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CreateAssessmentRunForWorkspace provides a mock function with given fields: ctx, options
func (_m *MockService) CreateAssessmentRunForWorkspace(ctx context.Context, options *CreateAssessmentRunForWorkspaceInput) (*models.Run, error) {
	ret := _m.Called(ctx, options)

	if len(ret) == 0 {
		panic("no return value specified for CreateAssessmentRunForWorkspace")
	}

	var r0 *models.Run
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *CreateAssessmentRunForWorkspaceInput) (*models.Run, error)); ok {
		return rf(ctx, options)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *CreateAssessmentRunForWorkspaceInput) *models.Run); ok {
		r0 = rf(ctx, options)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.Run)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *CreateAssessmentRunForWorkspaceInput) error); ok {
		r1 = rf(ctx, options)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CreateDestroyRunForWorkspace provides a mock function with given fields: ctx, options
func (_m *MockService) CreateDestroyRunForWorkspace(ctx context.Context, options *CreateDestroyRunForWorkspaceInput) (*models.Run, error) {
	ret := _m.Called(ctx, options)

	if len(ret) == 0 {
		panic("no return value specified for CreateDestroyRunForWorkspace")
	}

	var r0 *models.Run
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *CreateDestroyRunForWorkspaceInput) (*models.Run, error)); ok {
		return rf(ctx, options)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *CreateDestroyRunForWorkspaceInput) *models.Run); ok {
		r0 = rf(ctx, options)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.Run)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *CreateDestroyRunForWorkspaceInput) error); ok {
		r1 = rf(ctx, options)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CreateRun provides a mock function with given fields: ctx, options
func (_m *MockService) CreateRun(ctx context.Context, options *CreateRunInput) (*models.Run, error) {
	ret := _m.Called(ctx, options)

	if len(ret) == 0 {
		panic("no return value specified for CreateRun")
	}

	var r0 *models.Run
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *CreateRunInput) (*models.Run, error)); ok {
		return rf(ctx, options)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *CreateRunInput) *models.Run); ok {
		r0 = rf(ctx, options)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.Run)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *CreateRunInput) error); ok {
		r1 = rf(ctx, options)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DownloadPlan provides a mock function with given fields: ctx, planID
func (_m *MockService) DownloadPlan(ctx context.Context, planID string) (io.ReadCloser, error) {
	ret := _m.Called(ctx, planID)

	if len(ret) == 0 {
		panic("no return value specified for DownloadPlan")
	}

	var r0 io.ReadCloser
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (io.ReadCloser, error)); ok {
		return rf(ctx, planID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) io.ReadCloser); ok {
		r0 = rf(ctx, planID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(io.ReadCloser)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, planID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetAppliesByIDs provides a mock function with given fields: ctx, idList
func (_m *MockService) GetAppliesByIDs(ctx context.Context, idList []string) ([]models.Apply, error) {
	ret := _m.Called(ctx, idList)

	if len(ret) == 0 {
		panic("no return value specified for GetAppliesByIDs")
	}

	var r0 []models.Apply
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, []string) ([]models.Apply, error)); ok {
		return rf(ctx, idList)
	}
	if rf, ok := ret.Get(0).(func(context.Context, []string) []models.Apply); ok {
		r0 = rf(ctx, idList)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]models.Apply)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, []string) error); ok {
		r1 = rf(ctx, idList)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetApplyByID provides a mock function with given fields: ctx, applyID
func (_m *MockService) GetApplyByID(ctx context.Context, applyID string) (*models.Apply, error) {
	ret := _m.Called(ctx, applyID)

	if len(ret) == 0 {
		panic("no return value specified for GetApplyByID")
	}

	var r0 *models.Apply
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*models.Apply, error)); ok {
		return rf(ctx, applyID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *models.Apply); ok {
		r0 = rf(ctx, applyID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.Apply)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, applyID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetApplyByTRN provides a mock function with given fields: ctx, trn
func (_m *MockService) GetApplyByTRN(ctx context.Context, trn string) (*models.Apply, error) {
	ret := _m.Called(ctx, trn)

	if len(ret) == 0 {
		panic("no return value specified for GetApplyByTRN")
	}

	var r0 *models.Apply
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*models.Apply, error)); ok {
		return rf(ctx, trn)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *models.Apply); ok {
		r0 = rf(ctx, trn)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.Apply)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, trn)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetLatestJobForApply provides a mock function with given fields: ctx, applyID
func (_m *MockService) GetLatestJobForApply(ctx context.Context, applyID string) (*models.Job, error) {
	ret := _m.Called(ctx, applyID)

	if len(ret) == 0 {
		panic("no return value specified for GetLatestJobForApply")
	}

	var r0 *models.Job
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*models.Job, error)); ok {
		return rf(ctx, applyID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *models.Job); ok {
		r0 = rf(ctx, applyID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.Job)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, applyID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetLatestJobForPlan provides a mock function with given fields: ctx, planID
func (_m *MockService) GetLatestJobForPlan(ctx context.Context, planID string) (*models.Job, error) {
	ret := _m.Called(ctx, planID)

	if len(ret) == 0 {
		panic("no return value specified for GetLatestJobForPlan")
	}

	var r0 *models.Job
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*models.Job, error)); ok {
		return rf(ctx, planID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *models.Job); ok {
		r0 = rf(ctx, planID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.Job)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, planID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetPlanByID provides a mock function with given fields: ctx, planID
func (_m *MockService) GetPlanByID(ctx context.Context, planID string) (*models.Plan, error) {
	ret := _m.Called(ctx, planID)

	if len(ret) == 0 {
		panic("no return value specified for GetPlanByID")
	}

	var r0 *models.Plan
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*models.Plan, error)); ok {
		return rf(ctx, planID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *models.Plan); ok {
		r0 = rf(ctx, planID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.Plan)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, planID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetPlanByTRN provides a mock function with given fields: ctx, trn
func (_m *MockService) GetPlanByTRN(ctx context.Context, trn string) (*models.Plan, error) {
	ret := _m.Called(ctx, trn)

	if len(ret) == 0 {
		panic("no return value specified for GetPlanByTRN")
	}

	var r0 *models.Plan
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*models.Plan, error)); ok {
		return rf(ctx, trn)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *models.Plan); ok {
		r0 = rf(ctx, trn)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.Plan)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, trn)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetPlanDiff provides a mock function with given fields: ctx, planID
func (_m *MockService) GetPlanDiff(ctx context.Context, planID string) (*plan.Diff, error) {
	ret := _m.Called(ctx, planID)

	if len(ret) == 0 {
		panic("no return value specified for GetPlanDiff")
	}

	var r0 *plan.Diff
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*plan.Diff, error)); ok {
		return rf(ctx, planID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *plan.Diff); ok {
		r0 = rf(ctx, planID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*plan.Diff)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, planID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetPlansByIDs provides a mock function with given fields: ctx, idList
func (_m *MockService) GetPlansByIDs(ctx context.Context, idList []string) ([]models.Plan, error) {
	ret := _m.Called(ctx, idList)

	if len(ret) == 0 {
		panic("no return value specified for GetPlansByIDs")
	}

	var r0 []models.Plan
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, []string) ([]models.Plan, error)); ok {
		return rf(ctx, idList)
	}
	if rf, ok := ret.Get(0).(func(context.Context, []string) []models.Plan); ok {
		r0 = rf(ctx, idList)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]models.Plan)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, []string) error); ok {
		r1 = rf(ctx, idList)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetRunByID provides a mock function with given fields: ctx, runID
func (_m *MockService) GetRunByID(ctx context.Context, runID string) (*models.Run, error) {
	ret := _m.Called(ctx, runID)

	if len(ret) == 0 {
		panic("no return value specified for GetRunByID")
	}

	var r0 *models.Run
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*models.Run, error)); ok {
		return rf(ctx, runID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *models.Run); ok {
		r0 = rf(ctx, runID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.Run)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, runID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetRunByTRN provides a mock function with given fields: ctx, trn
func (_m *MockService) GetRunByTRN(ctx context.Context, trn string) (*models.Run, error) {
	ret := _m.Called(ctx, trn)

	if len(ret) == 0 {
		panic("no return value specified for GetRunByTRN")
	}

	var r0 *models.Run
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*models.Run, error)); ok {
		return rf(ctx, trn)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *models.Run); ok {
		r0 = rf(ctx, trn)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.Run)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, trn)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetRunVariables provides a mock function with given fields: ctx, runID, includeSensitiveValues
func (_m *MockService) GetRunVariables(ctx context.Context, runID string, includeSensitiveValues bool) ([]Variable, error) {
	ret := _m.Called(ctx, runID, includeSensitiveValues)

	if len(ret) == 0 {
		panic("no return value specified for GetRunVariables")
	}

	var r0 []Variable
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, bool) ([]Variable, error)); ok {
		return rf(ctx, runID, includeSensitiveValues)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, bool) []Variable); ok {
		r0 = rf(ctx, runID, includeSensitiveValues)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]Variable)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, bool) error); ok {
		r1 = rf(ctx, runID, includeSensitiveValues)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetRuns provides a mock function with given fields: ctx, input
func (_m *MockService) GetRuns(ctx context.Context, input *GetRunsInput) (*db.RunsResult, error) {
	ret := _m.Called(ctx, input)

	if len(ret) == 0 {
		panic("no return value specified for GetRuns")
	}

	var r0 *db.RunsResult
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *GetRunsInput) (*db.RunsResult, error)); ok {
		return rf(ctx, input)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *GetRunsInput) *db.RunsResult); ok {
		r0 = rf(ctx, input)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*db.RunsResult)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *GetRunsInput) error); ok {
		r1 = rf(ctx, input)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetRunsByIDs provides a mock function with given fields: ctx, idList
func (_m *MockService) GetRunsByIDs(ctx context.Context, idList []string) ([]models.Run, error) {
	ret := _m.Called(ctx, idList)

	if len(ret) == 0 {
		panic("no return value specified for GetRunsByIDs")
	}

	var r0 []models.Run
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, []string) ([]models.Run, error)); ok {
		return rf(ctx, idList)
	}
	if rf, ok := ret.Get(0).(func(context.Context, []string) []models.Run); ok {
		r0 = rf(ctx, idList)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]models.Run)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, []string) error); ok {
		r1 = rf(ctx, idList)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetStateVersionsByRunIDs provides a mock function with given fields: ctx, idList
func (_m *MockService) GetStateVersionsByRunIDs(ctx context.Context, idList []string) ([]models.StateVersion, error) {
	ret := _m.Called(ctx, idList)

	if len(ret) == 0 {
		panic("no return value specified for GetStateVersionsByRunIDs")
	}

	var r0 []models.StateVersion
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, []string) ([]models.StateVersion, error)); ok {
		return rf(ctx, idList)
	}
	if rf, ok := ret.Get(0).(func(context.Context, []string) []models.StateVersion); ok {
		r0 = rf(ctx, idList)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]models.StateVersion)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, []string) error); ok {
		r1 = rf(ctx, idList)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ProcessPlanData provides a mock function with given fields: ctx, planID, _a2, providerSchemas
func (_m *MockService) ProcessPlanData(ctx context.Context, planID string, _a2 *tfjson.Plan, providerSchemas *tfjson.ProviderSchemas) error {
	ret := _m.Called(ctx, planID, _a2, providerSchemas)

	if len(ret) == 0 {
		panic("no return value specified for ProcessPlanData")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, *tfjson.Plan, *tfjson.ProviderSchemas) error); ok {
		r0 = rf(ctx, planID, _a2, providerSchemas)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SetVariablesIncludedInTFConfig provides a mock function with given fields: ctx, input
func (_m *MockService) SetVariablesIncludedInTFConfig(ctx context.Context, input *SetVariablesIncludedInTFConfigInput) error {
	ret := _m.Called(ctx, input)

	if len(ret) == 0 {
		panic("no return value specified for SetVariablesIncludedInTFConfig")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *SetVariablesIncludedInTFConfigInput) error); ok {
		r0 = rf(ctx, input)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SubscribeToRunEvents provides a mock function with given fields: ctx, options
func (_m *MockService) SubscribeToRunEvents(ctx context.Context, options *EventSubscriptionOptions) (<-chan *Event, error) {
	ret := _m.Called(ctx, options)

	if len(ret) == 0 {
		panic("no return value specified for SubscribeToRunEvents")
	}

	var r0 <-chan *Event
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *EventSubscriptionOptions) (<-chan *Event, error)); ok {
		return rf(ctx, options)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *EventSubscriptionOptions) <-chan *Event); ok {
		r0 = rf(ctx, options)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(<-chan *Event)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *EventSubscriptionOptions) error); ok {
		r1 = rf(ctx, options)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// UpdateApply provides a mock function with given fields: ctx, apply
func (_m *MockService) UpdateApply(ctx context.Context, apply *models.Apply) (*models.Apply, error) {
	ret := _m.Called(ctx, apply)

	if len(ret) == 0 {
		panic("no return value specified for UpdateApply")
	}

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

// UpdatePlan provides a mock function with given fields: ctx, _a1
func (_m *MockService) UpdatePlan(ctx context.Context, _a1 *models.Plan) (*models.Plan, error) {
	ret := _m.Called(ctx, _a1)

	if len(ret) == 0 {
		panic("no return value specified for UpdatePlan")
	}

	var r0 *models.Plan
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *models.Plan) (*models.Plan, error)); ok {
		return rf(ctx, _a1)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *models.Plan) *models.Plan); ok {
		r0 = rf(ctx, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.Plan)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *models.Plan) error); ok {
		r1 = rf(ctx, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// UploadPlanBinary provides a mock function with given fields: ctx, planID, reader
func (_m *MockService) UploadPlanBinary(ctx context.Context, planID string, reader io.Reader) error {
	ret := _m.Called(ctx, planID, reader)

	if len(ret) == 0 {
		panic("no return value specified for UploadPlanBinary")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, io.Reader) error); ok {
		r0 = rf(ctx, planID, reader)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewMockService creates a new instance of MockService. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockService(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockService {
	mock := &MockService{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
