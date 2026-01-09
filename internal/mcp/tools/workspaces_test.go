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
	workspaceservice "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/workspace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestGetWorkspaceHandler(t *testing.T) {
	workspaceID := gid.ToGlobalID(types.WorkspaceModelType, "550e8400-e29b-41d4-a716-446655440004")

	type testCase struct {
		name           string
		workspaceModel *models.Workspace
		returnErr      error
		expectError    bool
		validate       func(*testing.T, getWorkspaceOutput)
	}

	tests := []testCase{
		{
			name: "successful workspace retrieval",
			workspaceModel: &models.Workspace{
				Metadata:         models.ResourceMetadata{ID: gid.FromGlobalID(workspaceID)},
				Name:             "test-workspace",
				FullPath:         "group/test-workspace",
				Description:      "Test workspace",
				GroupID:          "group-123",
				TerraformVersion: "1.5.0",
				CreatedBy:        "user@example.com",
				DirtyState:       false,
				Locked:           false,
			},
			validate: func(t *testing.T, output getWorkspaceOutput) {
				assert.Equal(t, "test-workspace", output.Workspace.Name)
				assert.Equal(t, "group/test-workspace", output.Workspace.FullPath)
				assert.Equal(t, "1.5.0", output.Workspace.TerraformVersion)
				assert.False(t, output.Workspace.Locked)
			},
		},
		{
			name:        "workspace not found",
			returnErr:   errors.New("not found"),
			expectError: true,
		},
		{
			name: "workspace with optional fields",
			workspaceModel: &models.Workspace{
				Metadata:             models.ResourceMetadata{ID: gid.FromGlobalID(workspaceID)},
				Name:                 "prod-workspace",
				FullPath:             "org/prod/prod-workspace",
				MaxJobDuration:       ptr.Int32(120),
				EnableDriftDetection: ptr.Bool(true),
				PreventDestroyPlan:   true,
				RunnerTags:           []string{"prod", "us-east-1"},
				Labels:               map[string]string{"env": "production", "team": "platform"},
			},
			validate: func(t *testing.T, output getWorkspaceOutput) {
				assert.Equal(t, "prod-workspace", output.Workspace.Name)
				assert.Equal(t, int32(120), *output.Workspace.MaxJobDuration)
				assert.True(t, *output.Workspace.EnableDriftDetection)
				assert.True(t, output.Workspace.PreventDestroyPlan)
				assert.Len(t, output.Workspace.RunnerTags, 2)
				assert.Len(t, output.Workspace.Labels, 2)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockWorkspaceService := workspaceservice.NewMockService(t)
			mockWorkspaceService.On("GetWorkspaceByID", mock.Anything, gid.FromGlobalID(workspaceID)).Return(tt.workspaceModel, tt.returnErr)

			catalog := &services.Catalog{WorkspaceService: mockWorkspaceService}
			catalog.Init()

			_, handler := GetWorkspace(&ToolContext{servicesCatalog: catalog})
			_, output, err := handler(context.Background(), nil, getWorkspaceInput{ID: workspaceID})

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
