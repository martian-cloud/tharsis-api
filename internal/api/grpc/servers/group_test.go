package servers

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/group"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
)

const testGroupID = "5f9c4f27-9d47-4d8a-9b0a-3f4c1a2b3c4d"

// testMetadata returns the metadata a model fetched from the database carries.
func testMetadata(id string) models.ResourceMetadata {
	now := time.Now()
	return models.ResourceMetadata{ID: id, CreationTimestamp: &now, LastUpdatedTimestamp: &now}
}

// groupUpdateStub returns a group whose namespace settings are all set and records the group it is asked
// to update. The group service has no mockery mock, so it embeds the interface: any other method panics.
type groupUpdateStub struct {
	group.Service
	updated *models.Group
}

func (s *groupUpdateStub) GetGroupByID(_ context.Context, id string) (*models.Group, error) {
	return &models.Group{
		Metadata:             testMetadata(id),
		Name:                 "group",
		FullPath:             "group",
		EnableDriftDetection: new(true),
		EnableProviderMirror: new(true),
	}, nil
}

func (s *groupUpdateStub) UpdateGroup(_ context.Context, g *models.Group) (*models.Group, error) {
	s.updated = g
	return g, nil
}

func TestGroupServer_UpdateGroupNamespaceSettings(t *testing.T) {
	tests := []struct {
		name                 string
		req                  *pb.UpdateGroupRequest
		expectDriftDetection *bool
		expectProviderMirror *bool
	}{
		{
			name:                 "unset settings are left as they are",
			req:                  &pb.UpdateGroupRequest{},
			expectDriftDetection: new(true),
			expectProviderMirror: new(true),
		},
		{
			name: "inherit clears the group's own settings",
			req: &pb.UpdateGroupRequest{
				DriftDetectionEnabled: &pb.NamespaceDriftDetectionEnabledInput{Value: &pb.NamespaceDriftDetectionEnabledInput_Inherit{Inherit: true}},
				ProviderMirrorEnabled: &pb.NamespaceProviderMirrorEnabledInput{Value: &pb.NamespaceProviderMirrorEnabledInput_Inherit{Inherit: true}},
			},
		},
		{
			name: "enabled sets the group's own settings",
			req: &pb.UpdateGroupRequest{
				DriftDetectionEnabled: &pb.NamespaceDriftDetectionEnabledInput{Value: &pb.NamespaceDriftDetectionEnabledInput_Enabled{Enabled: false}},
				ProviderMirrorEnabled: &pb.NamespaceProviderMirrorEnabledInput{Value: &pb.NamespaceProviderMirrorEnabledInput_Enabled{Enabled: false}},
			},
			expectDriftDetection: new(false),
			expectProviderMirror: new(false),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			svc := &groupUpdateStub{}
			catalog := &services.Catalog{GroupService: svc}
			catalog.Init()

			test.req.Id = gid.ToGlobalID(types.GroupModelType, testGroupID)
			ctx := auth.WithCaller(context.Background(), &auth.SystemCaller{})

			_, err := NewGroupServer(catalog).UpdateGroup(ctx, test.req)
			require.NoError(t, err)

			require.NotNil(t, svc.updated)
			assert.Equal(t, test.expectDriftDetection, svc.updated.EnableDriftDetection)
			assert.Equal(t, test.expectProviderMirror, svc.updated.EnableProviderMirror)
		})
	}
}
