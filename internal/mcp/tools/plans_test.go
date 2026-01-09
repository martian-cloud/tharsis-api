package tools

import (
	"context"
	"testing"

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

func TestGetPlanHandler(t *testing.T) {
	planID := gid.ToGlobalID(types.PlanModelType, "550e8400-e29b-41d4-a716-446655440002")

	type testCase struct {
		name        string
		planModel   *models.Plan
		returnErr   error
		expectError bool
		validate    func(*testing.T, getPlanOutput)
	}

	tests := []testCase{
		{
			name: "successful plan retrieval",
			planModel: &models.Plan{
				Metadata:    models.ResourceMetadata{ID: gid.FromGlobalID(planID)},
				Status:      models.PlanFinished,
				WorkspaceID: "ws-123",
				HasChanges:  true,
				Summary: models.PlanSummary{
					ResourceAdditions:    5,
					ResourceChanges:      2,
					ResourceDestructions: 1,
				},
			},
			validate: func(t *testing.T, output getPlanOutput) {
				assert.Equal(t, models.PlanFinished, output.Plan.Status)
				assert.True(t, output.Plan.HasChanges)
				assert.Equal(t, int32(5), output.Plan.ResourceAdditions)
				assert.Equal(t, int32(2), output.Plan.ResourceChanges)
				assert.Equal(t, int32(1), output.Plan.ResourceDestructions)
			},
		},
		{
			name:        "plan not found",
			returnErr:   errors.New("not found"),
			expectError: true,
		},
		{
			name: "plan with error message",
			planModel: &models.Plan{
				Metadata:     models.ResourceMetadata{ID: gid.FromGlobalID(planID)},
				Status:       models.PlanErrored,
				ErrorMessage: ptr.String("terraform plan failed"),
				HasChanges:   false,
			},
			validate: func(t *testing.T, output getPlanOutput) {
				assert.Equal(t, models.PlanErrored, output.Plan.Status)
				assert.Equal(t, "terraform plan failed", *output.Plan.ErrorMessage)
				assert.False(t, output.Plan.HasChanges)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRunService := runservice.NewMockService(t)
			mockRunService.On("GetPlanByID", mock.Anything, gid.FromGlobalID(planID)).Return(tt.planModel, tt.returnErr)

			catalog := &services.Catalog{RunService: mockRunService}
			catalog.Init()

			_, handler := GetPlan(&ToolContext{servicesCatalog: catalog})
			_, output, err := handler(context.Background(), nil, getPlanInput{ID: planID})

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
