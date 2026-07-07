package transformers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/statemachine"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/namespace"
)

func maxJobDuration(v int32) *int32 { return &v }

func TestJobCreationTransformer_Transform_CreatesPlanJob(t *testing.T) {
	ctx := context.Background()

	run := &models.Run{
		Metadata:    models.ResourceMetadata{ID: "run-1"},
		WorkspaceID: "ws-1",
		Plan:        models.Plan{Status: models.PlanQueued},
	}

	ws := &models.Workspace{
		Metadata:       models.ResourceMetadata{ID: "ws-1"},
		MaxJobDuration: maxJobDuration(60),
	}

	mockWorkspaces := db.NewMockWorkspaces(t)
	mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "ws-1").Return(ws, nil)

	mockJobs := db.NewMockJobs(t)
	mockJobs.On("CreateJob", mock.Anything, mock.MatchedBy(func(j *models.Job) bool {
		return j.Type == models.JobPlanType && j.WorkspaceID == "ws-1" && j.RunID == "run-1"
	})).Return(&models.Job{Metadata: models.ResourceMetadata{ID: "job-1"}}, nil)

	mockLogStreams := db.NewMockLogStreams(t)
	mockLogStreams.On("CreateLogStream", mock.Anything, mock.MatchedBy(func(ls *models.LogStream) bool {
		return ls.JobID != nil && *ls.JobID == "job-1"
	})).Return(&models.LogStream{}, nil)

	resolver := namespace.NewMockInheritedSettingResolver(t)
	resolver.On("GetRunnerTags", mock.Anything, ws).Return(&namespace.RunnerTagsSetting{Value: []string{"tag1"}}, nil)
	resolver.On("GetProviderMirrorEnabled", mock.Anything, ws).Return(&namespace.ProviderMirrorEnabledSetting{Value: true}, nil)

	dbClient := &db.Client{
		Workspaces: mockWorkspaces,
		Jobs:       mockJobs,
		LogStreams: mockLogStreams,
	}

	transformer := NewJobCreationTransformer(dbClient, resolver)

	change := types.RunChange{
		Run:               run,
		NodeStatusChanges: []statemachine.NodeStatusChange{statemachine.PlanStatusChange{NewStatus: models.PlanQueued}},
	}

	err := transformer.Transform(ctx, []types.RunChange{change}, nil)
	assert.NoError(t, err)

	// The created job's ID is linked to the plan node.
	assert.NotNil(t, run.Plan.LatestJobID)
	assert.Equal(t, "job-1", *run.Plan.LatestJobID)
}

func TestJobCreationTransformer_Transform_CreatesApplyJob(t *testing.T) {
	ctx := context.Background()

	run := &models.Run{
		Metadata:    models.ResourceMetadata{ID: "run-1"},
		WorkspaceID: "ws-1",
		Apply:       &models.Apply{Status: models.ApplyQueued},
	}

	ws := &models.Workspace{
		Metadata:       models.ResourceMetadata{ID: "ws-1"},
		MaxJobDuration: maxJobDuration(60),
	}

	mockWorkspaces := db.NewMockWorkspaces(t)
	mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "ws-1").Return(ws, nil)

	mockJobs := db.NewMockJobs(t)
	mockJobs.On("CreateJob", mock.Anything, mock.MatchedBy(func(j *models.Job) bool {
		return j.Type == models.JobApplyType
	})).Return(&models.Job{Metadata: models.ResourceMetadata{ID: "job-apply"}}, nil)

	mockLogStreams := db.NewMockLogStreams(t)
	mockLogStreams.On("CreateLogStream", mock.Anything, mock.Anything).Return(&models.LogStream{}, nil)

	resolver := namespace.NewMockInheritedSettingResolver(t)
	resolver.On("GetRunnerTags", mock.Anything, ws).Return(&namespace.RunnerTagsSetting{Value: []string{}}, nil)
	resolver.On("GetProviderMirrorEnabled", mock.Anything, ws).Return(&namespace.ProviderMirrorEnabledSetting{Value: false}, nil)

	dbClient := &db.Client{
		Workspaces: mockWorkspaces,
		Jobs:       mockJobs,
		LogStreams: mockLogStreams,
	}

	transformer := NewJobCreationTransformer(dbClient, resolver)

	change := types.RunChange{
		Run:               run,
		NodeStatusChanges: []statemachine.NodeStatusChange{statemachine.ApplyStatusChange{NewStatus: models.ApplyQueued}},
	}

	err := transformer.Transform(ctx, []types.RunChange{change}, nil)
	assert.NoError(t, err)

	assert.NotNil(t, run.Apply.LatestJobID)
	assert.Equal(t, "job-apply", *run.Apply.LatestJobID)
}

func TestJobCreationTransformer_Transform_IgnoresNonQueuedTransition(t *testing.T) {
	ctx := context.Background()

	run := &models.Run{
		Metadata:    models.ResourceMetadata{ID: "run-1"},
		WorkspaceID: "ws-1",
		Plan:        models.Plan{Status: models.PlanPending},
	}

	// No mock expectations: a non-queued transition must not touch the db.
	dbClient := &db.Client{
		Workspaces: db.NewMockWorkspaces(t),
		Jobs:       db.NewMockJobs(t),
		LogStreams: db.NewMockLogStreams(t),
	}
	resolver := namespace.NewMockInheritedSettingResolver(t)

	transformer := NewJobCreationTransformer(dbClient, resolver)

	change := types.RunChange{
		Run:               run,
		NodeStatusChanges: []statemachine.NodeStatusChange{statemachine.PlanStatusChange{NewStatus: models.PlanPending}},
	}

	err := transformer.Transform(ctx, []types.RunChange{change}, nil)
	assert.NoError(t, err)
	assert.Nil(t, run.Plan.LatestJobID)
}

func TestJobCreationTransformer_Transform_ErrorsWhenMaxJobDurationMissing(t *testing.T) {
	ctx := context.Background()

	run := &models.Run{
		Metadata:    models.ResourceMetadata{ID: "run-1"},
		WorkspaceID: "ws-1",
		Plan:        models.Plan{Status: models.PlanQueued},
	}

	mockWorkspaces := db.NewMockWorkspaces(t)
	mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "ws-1").
		Return(&models.Workspace{Metadata: models.ResourceMetadata{ID: "ws-1"}}, nil) // no MaxJobDuration

	dbClient := &db.Client{
		Workspaces: mockWorkspaces,
		Jobs:       db.NewMockJobs(t),
		LogStreams: db.NewMockLogStreams(t),
	}
	resolver := namespace.NewMockInheritedSettingResolver(t)

	transformer := NewJobCreationTransformer(dbClient, resolver)

	change := types.RunChange{
		Run:               run,
		NodeStatusChanges: []statemachine.NodeStatusChange{statemachine.PlanStatusChange{NewStatus: models.PlanQueued}},
	}

	err := transformer.Transform(ctx, []types.RunChange{change}, nil)
	assert.Error(t, err)
}

func TestJobCreationTransformer_Transform_ErrorsWhenWorkspaceNotFound(t *testing.T) {
	ctx := context.Background()

	run := &models.Run{
		Metadata:    models.ResourceMetadata{ID: "run-1"},
		WorkspaceID: "ws-1",
		Plan:        models.Plan{Status: models.PlanQueued},
	}

	mockWorkspaces := db.NewMockWorkspaces(t)
	mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "ws-1").Return(nil, nil)

	dbClient := &db.Client{
		Workspaces: mockWorkspaces,
		Jobs:       db.NewMockJobs(t),
		LogStreams: db.NewMockLogStreams(t),
	}
	resolver := namespace.NewMockInheritedSettingResolver(t)

	transformer := NewJobCreationTransformer(dbClient, resolver)

	change := types.RunChange{
		Run:               run,
		NodeStatusChanges: []statemachine.NodeStatusChange{statemachine.PlanStatusChange{NewStatus: models.PlanQueued}},
	}

	err := transformer.Transform(ctx, []types.RunChange{change}, nil)
	assert.Error(t, err)
}
