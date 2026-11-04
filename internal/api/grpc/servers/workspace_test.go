package servers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services"
	workspacesvc "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/workspace"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
)

const testWorkspaceID = "8a1b2c3d-4e5f-4a6b-8c9d-0e1f2a3b4c5d"

var (
	inheritDriftDetection = &pb.NamespaceDriftDetectionEnabledInput{Value: &pb.NamespaceDriftDetectionEnabledInput_Inherit{Inherit: true}}
	inheritProviderMirror = &pb.NamespaceProviderMirrorEnabledInput{Value: &pb.NamespaceProviderMirrorEnabledInput_Inherit{Inherit: true}}
	disableDriftDetection = &pb.NamespaceDriftDetectionEnabledInput{Value: &pb.NamespaceDriftDetectionEnabledInput_Enabled{Enabled: false}}
	disableProviderMirror = &pb.NamespaceProviderMirrorEnabledInput{Value: &pb.NamespaceProviderMirrorEnabledInput_Enabled{Enabled: false}}
)

func TestWorkspaceServer_CreateWorkspaceNamespaceSettings(t *testing.T) {
	tests := []struct {
		name                 string
		req                  *pb.CreateWorkspaceRequest
		expectDriftDetection *bool
		expectProviderMirror *bool
	}{
		{
			name: "unset settings are inherited",
			req:  &pb.CreateWorkspaceRequest{},
		},
		{
			name: "inherit leaves the settings unset",
			req:  &pb.CreateWorkspaceRequest{DriftDetectionEnabled: inheritDriftDetection, ProviderMirrorEnabled: inheritProviderMirror},
		},
		{
			name:                 "enabled sets the workspace's own settings",
			req:                  &pb.CreateWorkspaceRequest{DriftDetectionEnabled: disableDriftDetection, ProviderMirrorEnabled: disableProviderMirror},
			expectDriftDetection: new(false),
			expectProviderMirror: new(false),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var created *models.Workspace
			svc := workspacesvc.NewMockService(t)
			svc.On("CreateWorkspace", mock.Anything, mock.Anything).
				Return(func(_ context.Context, ws *models.Workspace) (*models.Workspace, error) {
					created = ws
					ws.Metadata = testMetadata(testWorkspaceID)
					return ws, nil
				})

			catalog := &services.Catalog{WorkspaceService: svc}
			catalog.Init()

			test.req.GroupId = gid.ToGlobalID(types.GroupModelType, testGroupID)
			ctx := auth.WithCaller(context.Background(), &auth.SystemCaller{})

			_, err := NewWorkspaceServer(catalog).CreateWorkspace(ctx, test.req)
			require.NoError(t, err)

			require.NotNil(t, created)
			assert.Equal(t, test.expectDriftDetection, created.EnableDriftDetection)
			assert.Equal(t, test.expectProviderMirror, created.EnableProviderMirror)
		})
	}
}

func TestWorkspaceServer_UpdateWorkspaceNamespaceSettings(t *testing.T) {
	tests := []struct {
		name                 string
		req                  *pb.UpdateWorkspaceRequest
		expectDriftDetection *bool
		expectProviderMirror *bool
	}{
		{
			name:                 "unset settings are left as they are",
			req:                  &pb.UpdateWorkspaceRequest{},
			expectDriftDetection: new(true),
			expectProviderMirror: new(true),
		},
		{
			name: "inherit clears the workspace's own settings",
			req:  &pb.UpdateWorkspaceRequest{DriftDetectionEnabled: inheritDriftDetection, ProviderMirrorEnabled: inheritProviderMirror},
		},
		{
			name:                 "enabled sets the workspace's own settings",
			req:                  &pb.UpdateWorkspaceRequest{DriftDetectionEnabled: disableDriftDetection, ProviderMirrorEnabled: disableProviderMirror},
			expectDriftDetection: new(false),
			expectProviderMirror: new(false),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var updated *models.Workspace
			svc := workspacesvc.NewMockService(t)
			svc.On("GetWorkspaceByID", mock.Anything, testWorkspaceID).Return(&models.Workspace{
				Metadata:             testMetadata(testWorkspaceID),
				EnableDriftDetection: new(true),
				EnableProviderMirror: new(true),
			}, nil)
			svc.On("UpdateWorkspace", mock.Anything, mock.Anything).
				Return(func(_ context.Context, ws *models.Workspace) (*models.Workspace, error) {
					updated = ws
					return ws, nil
				})

			catalog := &services.Catalog{WorkspaceService: svc}
			catalog.Init()

			test.req.Id = gid.ToGlobalID(types.WorkspaceModelType, testWorkspaceID)
			ctx := auth.WithCaller(context.Background(), &auth.SystemCaller{})

			_, err := NewWorkspaceServer(catalog).UpdateWorkspace(ctx, test.req)
			require.NoError(t, err)

			require.NotNil(t, updated)
			assert.Equal(t, test.expectDriftDetection, updated.EnableDriftDetection)
			assert.Equal(t, test.expectProviderMirror, updated.EnableProviderMirror)
		})
	}
}
