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

func TestGetApplyHandler(t *testing.T) {
	applyID := gid.ToGlobalID(types.ApplyModelType, "550e8400-e29b-41d4-a716-446655440001")

	type testCase struct {
		name        string
		applyModel  *models.Apply
		returnErr   error
		expectError bool
		validate    func(*testing.T, getApplyOutput)
	}

	tests := []testCase{
		{
			name: "successful apply retrieval",
			applyModel: &models.Apply{
				Metadata:    models.ResourceMetadata{ID: gid.FromGlobalID(applyID)},
				Status:      models.ApplyFinished,
				WorkspaceID: "ws-123",
				TriggeredBy: "user@example.com",
			},
			validate: func(t *testing.T, output getApplyOutput) {
				assert.Equal(t, models.ApplyFinished, output.Apply.Status)
				assert.Equal(t, "user@example.com", output.Apply.TriggeredBy)
			},
		},
		{
			name:        "apply not found",
			returnErr:   errors.New("not found"),
			expectError: true,
		},
		{
			name: "apply with error message",
			applyModel: &models.Apply{
				Metadata:     models.ResourceMetadata{ID: gid.FromGlobalID(applyID)},
				Status:       models.ApplyErrored,
				ErrorMessage: ptr.String("terraform apply failed"),
				TriggeredBy:  "user@example.com",
			},
			validate: func(t *testing.T, output getApplyOutput) {
				assert.Equal(t, models.ApplyErrored, output.Apply.Status)
				assert.Equal(t, "terraform apply failed", *output.Apply.ErrorMessage)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRunService := runservice.NewMockService(t)
			mockRunService.On("GetApplyByID", mock.Anything, gid.FromGlobalID(applyID)).Return(tt.applyModel, tt.returnErr)

			catalog := &services.Catalog{RunService: mockRunService}
			catalog.Init()

			_, handler := GetApply(&ToolContext{servicesCatalog: catalog})
			_, output, err := handler(context.Background(), nil, getApplyInput{ID: applyID})

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
