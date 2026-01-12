package tools

import (
	"context"
	"testing"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services"
	jobservice "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/job"
	runservice "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestGetJobHandler(t *testing.T) {
	jobID := gid.ToGlobalID(types.JobModelType, "550e8400-e29b-41d4-a716-446655440005")
	runnerID := "runner-456"
	now := time.Now()

	type testCase struct {
		name        string
		jobModel    *models.Job
		returnErr   error
		expectError bool
		validate    func(*testing.T, getJobOutput)
	}

	tests := []testCase{
		{
			name: "successful job retrieval",
			jobModel: &models.Job{
				Metadata:    models.ResourceMetadata{ID: gid.FromGlobalID(jobID)},
				Status:      models.JobRunning,
				Type:        models.JobPlanType,
				WorkspaceID: "ws-123",
				RunID:       "run-123",
				RunnerID:    &runnerID,
				Timestamps: models.JobTimestamps{
					QueuedTimestamp:  &now,
					RunningTimestamp: &now,
				},
			},
			validate: func(t *testing.T, output getJobOutput) {
				assert.Equal(t, models.JobRunning, output.Job.Status)
				assert.Equal(t, models.JobPlanType, output.Job.Type)
				assert.NotNil(t, output.Job.RunnerID)
				assert.False(t, output.Job.CancelRequested)
			},
		},
		{
			name:        "job not found",
			returnErr:   errors.New("not found"),
			expectError: true,
		},
		{
			name: "job with cancel requested",
			jobModel: &models.Job{
				Metadata:                 models.ResourceMetadata{ID: gid.FromGlobalID(jobID)},
				Status:                   models.JobFinished,
				Type:                     models.JobApplyType,
				CancelRequestedTimestamp: &now,
			},
			validate: func(t *testing.T, output getJobOutput) {
				assert.True(t, output.Job.CancelRequested)
				assert.Equal(t, models.JobApplyType, output.Job.Type)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockJobService := jobservice.NewMockService(t)
			mockJobService.On("GetJobByID", mock.Anything, gid.FromGlobalID(jobID)).Return(tt.jobModel, tt.returnErr)

			catalog := &services.Catalog{JobService: mockJobService}
			catalog.Init()

			_, handler := GetJob(&ToolContext{servicesCatalog: catalog})
			_, output, err := handler(context.Background(), nil, getJobInput{ID: jobID})

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, output)
				}
			}
		})
	}
}

func TestGetJobLogsHandler(t *testing.T) {
	jobID := gid.ToGlobalID(types.JobModelType, "550e8400-e29b-41d4-a716-446655440006")
	logs := "terraform plan output"

	type testCase struct {
		name        string
		input       getJobLogsInput
		setup       func(*testing.T) *services.Catalog
		expectError bool
		validate    func(*testing.T, getJobLogsOutput)
	}

	tests := []testCase{
		{
			name:  "successful logs retrieval",
			input: getJobLogsInput{ID: jobID},
			setup: func(t *testing.T) *services.Catalog {
				mockJobService := jobservice.NewMockService(t)
				mockJobService.On("GetJobByID", mock.Anything, gid.FromGlobalID(jobID)).Return(&models.Job{
					Metadata: models.ResourceMetadata{ID: gid.FromGlobalID(jobID)},
				}, nil)
				mockJobService.On("ReadLogs", mock.Anything, gid.FromGlobalID(jobID), 0, defaultLogLimit+1).Return([]byte(logs), nil)

				catalog := &services.Catalog{JobService: mockJobService}
				catalog.Init()
				return catalog
			},
			validate: func(t *testing.T, output getJobLogsOutput) {
				assert.Equal(t, jobID, output.JobID)
				assert.Equal(t, logs, output.Logs)
				assert.Equal(t, len(logs), output.Size)
				assert.False(t, output.HasMore)
			},
		},
		{
			name:  "job not found",
			input: getJobLogsInput{ID: jobID},
			setup: func(t *testing.T) *services.Catalog {
				mockJobService := jobservice.NewMockService(t)
				mockJobService.On("GetJobByID", mock.Anything, gid.FromGlobalID(jobID)).Return(nil, errors.New("not found"))

				catalog := &services.Catalog{JobService: mockJobService}
				catalog.Init()
				return catalog
			},
			expectError: true,
		},
		{
			name:  "custom start and limit",
			input: getJobLogsInput{ID: jobID, Start: ptr.Int(100), Limit: ptr.Int(1000)},
			setup: func(t *testing.T) *services.Catalog {
				mockJobService := jobservice.NewMockService(t)
				mockJobService.On("GetJobByID", mock.Anything, gid.FromGlobalID(jobID)).Return(&models.Job{
					Metadata: models.ResourceMetadata{ID: gid.FromGlobalID(jobID)},
				}, nil)
				mockJobService.On("ReadLogs", mock.Anything, gid.FromGlobalID(jobID), 100, 1001).Return([]byte(logs), nil)

				catalog := &services.Catalog{JobService: mockJobService}
				catalog.Init()
				return catalog
			},
			validate: func(t *testing.T, output getJobLogsOutput) {
				assert.Equal(t, 100, output.Start)
			},
		},
		{
			name:  "limit exceeds maximum",
			input: getJobLogsInput{ID: jobID, Limit: ptr.Int(maxLogLimit + 1)},
			setup: func(t *testing.T) *services.Catalog {
				mockJobService := jobservice.NewMockService(t)
				mockJobService.On("GetJobByID", mock.Anything, gid.FromGlobalID(jobID)).Return(&models.Job{
					Metadata: models.ResourceMetadata{ID: gid.FromGlobalID(jobID)},
				}, nil)

				catalog := &services.Catalog{JobService: mockJobService}
				catalog.Init()
				return catalog
			},
			expectError: true,
		},
		{
			name:  "has more logs",
			input: getJobLogsInput{ID: jobID, Limit: ptr.Int(10)},
			setup: func(t *testing.T) *services.Catalog {
				mockJobService := jobservice.NewMockService(t)
				mockJobService.On("GetJobByID", mock.Anything, gid.FromGlobalID(jobID)).Return(&models.Job{
					Metadata: models.ResourceMetadata{ID: gid.FromGlobalID(jobID)},
				}, nil)
				mockJobService.On("ReadLogs", mock.Anything, gid.FromGlobalID(jobID), 0, 11).Return([]byte("01234567890"), nil)

				catalog := &services.Catalog{JobService: mockJobService}
				catalog.Init()
				return catalog
			},
			validate: func(t *testing.T, output getJobLogsOutput) {
				assert.True(t, output.HasMore)
				assert.Equal(t, 10, output.Size)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, handler := GetJobLogs(&ToolContext{servicesCatalog: tt.setup(t)})
			_, output, err := handler(context.Background(), nil, tt.input)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, output)
				}
			}
		})
	}
}

func TestGetLatestJobHandler(t *testing.T) {
	planID := gid.ToGlobalID(types.PlanModelType, "550e8400-e29b-41d4-a716-446655440007")
	applyID := gid.ToGlobalID(types.ApplyModelType, "550e8400-e29b-41d4-a716-446655440008")
	jobID := "job-123"

	type testCase struct {
		name        string
		input       getLatestJobInput
		setup       func(*testing.T) *services.Catalog
		expectError bool
		validate    func(*testing.T, getLatestJobOutput)
	}

	tests := []testCase{
		{
			name:  "successful plan job retrieval",
			input: getLatestJobInput{ID: planID},
			setup: func(t *testing.T) *services.Catalog {
				mockRunService := runservice.NewMockService(t)
				mockRunService.On("GetPlanByID", mock.Anything, gid.FromGlobalID(planID)).Return(&models.Plan{
					Metadata: models.ResourceMetadata{ID: gid.FromGlobalID(planID)},
				}, nil)
				mockRunService.On("GetLatestJobForPlan", mock.Anything, gid.FromGlobalID(planID)).Return(&models.Job{
					Metadata: models.ResourceMetadata{ID: jobID},
					Status:   models.JobFinished,
					Type:     models.JobPlanType,
				}, nil)

				catalog := &services.Catalog{RunService: mockRunService}
				catalog.Init()
				return catalog
			},
			validate: func(t *testing.T, output getLatestJobOutput) {
				assert.Equal(t, models.JobPlanType, output.Job.Type)
				assert.Equal(t, models.JobFinished, output.Job.Status)
			},
		},
		{
			name:  "successful apply job retrieval",
			input: getLatestJobInput{ID: applyID},
			setup: func(t *testing.T) *services.Catalog {
				mockRunService := runservice.NewMockService(t)
				mockRunService.On("GetApplyByID", mock.Anything, gid.FromGlobalID(applyID)).Return(&models.Apply{
					Metadata: models.ResourceMetadata{ID: gid.FromGlobalID(applyID)},
				}, nil)
				mockRunService.On("GetLatestJobForApply", mock.Anything, gid.FromGlobalID(applyID)).Return(&models.Job{
					Metadata: models.ResourceMetadata{ID: jobID},
					Status:   models.JobRunning,
					Type:     models.JobApplyType,
				}, nil)

				catalog := &services.Catalog{RunService: mockRunService}
				catalog.Init()
				return catalog
			},
			validate: func(t *testing.T, output getLatestJobOutput) {
				assert.Equal(t, models.JobApplyType, output.Job.Type)
				assert.Equal(t, models.JobRunning, output.Job.Status)
			},
		},
		{
			name:  "plan not found",
			input: getLatestJobInput{ID: planID},
			setup: func(t *testing.T) *services.Catalog {
				mockRunService := runservice.NewMockService(t)
				mockRunService.On("GetPlanByID", mock.Anything, gid.FromGlobalID(planID)).Return(nil, errors.New("not found"))

				catalog := &services.Catalog{RunService: mockRunService}
				catalog.Init()
				return catalog
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, handler := GetLatestJob(&ToolContext{servicesCatalog: tt.setup(t)})
			_, output, err := handler(context.Background(), nil, tt.input)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, output)
				}
			}
		})
	}
}
