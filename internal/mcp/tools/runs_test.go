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
	runservice "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestGetRunHandler(t *testing.T) {
	runID := gid.ToGlobalID(types.RunModelType, "550e8400-e29b-41d4-a716-446655440000")
	now := time.Now()

	type testCase struct {
		name        string
		runModel    *models.Run
		returnErr   error
		expectError bool
		validate    func(*testing.T, getRunOutput)
	}

	tests := []testCase{
		{
			name: "successful run retrieval",
			runModel: &models.Run{
				Metadata:         models.ResourceMetadata{ID: gid.FromGlobalID(runID)},
				Status:           models.RunApplied,
				CreatedBy:        "user@example.com",
				TerraformVersion: "1.5.0",
				HasChanges:       true,
			},
			validate: func(t *testing.T, output getRunOutput) {
				assert.Equal(t, models.RunApplied, output.Run.Status)
				assert.Equal(t, "user@example.com", output.Run.CreatedBy)
				assert.Equal(t, "1.5.0", output.Run.TerraformVersion)
				assert.True(t, output.Run.HasChanges)
			},
		},
		{
			name:        "run not found",
			returnErr:   errors.New("not found"),
			expectError: true,
		},
		{
			name: "run with optional fields",
			runModel: &models.Run{
				Metadata:               models.ResourceMetadata{ID: gid.FromGlobalID(runID)},
				Status:                 models.RunCanceled,
				ModuleSource:           ptr.String("registry.terraform.io/hashicorp/aws"),
				ModuleVersion:          ptr.String("5.0.0"),
				ForceCanceledBy:        ptr.String("admin@example.com"),
				ForceCanceled:          true,
				ForceCancelAvailableAt: &now,
				TargetAddresses:        []string{"aws_instance.example"},
			},
			validate: func(t *testing.T, output getRunOutput) {
				assert.Equal(t, models.RunCanceled, output.Run.Status)
				assert.Equal(t, "registry.terraform.io/hashicorp/aws", *output.Run.ModuleSource)
				assert.Equal(t, "5.0.0", *output.Run.ModuleVersion)
				assert.Equal(t, "admin@example.com", *output.Run.ForceCanceledBy)
				assert.True(t, output.Run.ForceCanceled)
				assert.Len(t, output.Run.TargetAddresses, 1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRunService := runservice.NewMockService(t)
			mockRunService.On("GetRunByID", mock.Anything, gid.FromGlobalID(runID)).Return(tt.runModel, tt.returnErr)

			catalog := &services.Catalog{RunService: mockRunService}
			catalog.Init()

			_, handler := GetRun(&ToolContext{servicesCatalog: catalog})
			_, output, err := handler(context.Background(), nil, getRunInput{ID: runID})

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
