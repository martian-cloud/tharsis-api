package tools

import (
	"context"
	"io"
	"strings"
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
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// jobWithStatus sets the initial status on a freshly constructed test job. It is
// valid from the job's zero value, so the error is intentionally ignored.
func jobWithStatus(j *models.Job, status models.JobStatus) *models.Job {
	_ = j.SetStatus(status)
	return j
}

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
			jobModel: jobWithStatus(&models.Job{
				Metadata:    models.ResourceMetadata{ID: gid.FromGlobalID(jobID)},
				Type:        models.JobPlanType,
				WorkspaceID: "ws-123",
				RunID:       "run-123",
				RunnerID:    &runnerID,
				Timestamps: models.JobTimestamps{
					QueuedTimestamp:  &now,
					RunningTimestamp: &now,
				},
			}, models.JobRunning),
			validate: func(t *testing.T, output getJobOutput) {
				assert.Equal(t, models.JobRunning, output.Job.Status)
				assert.Equal(t, models.JobPlanType, output.Job.Type)
				assert.NotNil(t, output.Job.RunnerID)
			},
		},
		{
			name:        "job not found",
			returnErr:   errors.New("not found"),
			expectError: true,
		},
		{
			name: "job with cancel requested",
			jobModel: jobWithStatus(&models.Job{
				Metadata: models.ResourceMetadata{ID: gid.FromGlobalID(jobID)},
				Type:     models.JobApplyType,
			}, models.JobCanceling),
			validate: func(t *testing.T, output getJobOutput) {
				assert.Equal(t, models.JobCanceling, output.Job.Status)
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
				mockJobService.On("ReadLogs", mock.Anything, gid.FromGlobalID(jobID), 0, defaultLogLimit+1).Return(io.NopCloser(strings.NewReader(logs)), nil)

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
				mockJobService.On("ReadLogs", mock.Anything, gid.FromGlobalID(jobID), 100, 1001).Return(io.NopCloser(strings.NewReader(logs)), nil)

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
				mockJobService.On("ReadLogs", mock.Anything, gid.FromGlobalID(jobID), 0, 11).Return(io.NopCloser(strings.NewReader("01234567890")), nil)

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
