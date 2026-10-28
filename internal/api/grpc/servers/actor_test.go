package servers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
	"google.golang.org/protobuf/proto"
)

const testActorID = "5c9d0e1f-2a3b-4c5d-6e7f-8a9b0c1d2e3f"

func TestToPBActor(t *testing.T) {
	tests := []struct {
		name     string
		actor    models.Actor
		expected *pb.Actor
	}{
		{
			name:  "user actor carries the user global id",
			actor: models.NewUserActor(testActorID),
			expected: &pb.Actor{Actor: &pb.Actor_User{User: &pb.UserActor{
				PrincipalId: gid.ToGlobalID(types.UserModelType, testActorID),
			}}},
		},
		{
			name:  "service account actor carries its global id and trn snapshot",
			actor: models.NewServiceAccountActor(testActorID).WithTRN("trn:service_account:group/sa"),
			expected: &pb.Actor{Actor: &pb.Actor_ServiceAccount{ServiceAccount: &pb.ServiceAccountActor{
				PrincipalId: gid.ToGlobalID(types.ServiceAccountModelType, testActorID),
				TrnSnapshot: ptrString("trn:service_account:group/sa"),
			}}},
		},
		{
			name:  "job actor carries its global id and trn snapshot",
			actor: models.NewJobActor(testActorID).WithTRN("trn:job:group/ws/job-gid"),
			expected: &pb.Actor{Actor: &pb.Actor_Job{Job: &pb.JobActor{
				PrincipalId: gid.ToGlobalID(types.JobModelType, testActorID),
				TrnSnapshot: ptrString("trn:job:group/ws/job-gid"),
			}}},
		},
		{
			name:  "job actor without a trn has no trn snapshot",
			actor: models.NewJobActor(testActorID),
			expected: &pb.Actor{Actor: &pb.Actor_Job{Job: &pb.JobActor{
				PrincipalId: gid.ToGlobalID(types.JobModelType, testActorID),
			}}},
		},
		{
			name:  "vcs provider actor carries its global id and trn snapshot",
			actor: models.NewVCSProviderActor(testActorID).WithTRN("trn:vcs_provider:group/vcs"),
			expected: &pb.Actor{Actor: &pb.Actor_VcsProvider{VcsProvider: &pb.VCSProviderActor{
				PrincipalId: gid.ToGlobalID(types.VCSProviderModelType, testActorID),
				TrnSnapshot: ptrString("trn:vcs_provider:group/vcs"),
			}}},
		},
		{
			name:     "system actor sets the system case",
			actor:    models.SystemActor(),
			expected: &pb.Actor{Actor: &pb.Actor_System{System: &pb.SystemActor{}}},
		},
		{
			name:     "scim actor sets the scim case",
			actor:    models.SCIMActor(),
			expected: &pb.Actor{Actor: &pb.Actor_Scim{Scim: &pb.SCIMActor{}}},
		},
		{
			name:  "zero actor converts to nil",
			actor: models.Actor{},
		},
		{
			name:  "untyped actor converts to nil",
			actor: models.Actor{Subject: "legacy@example.com"},
		},
		{
			name:  "actor type unknown to the proto converts to nil",
			actor: models.Actor{Type: "future", ID: testActorID},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := toPBActor(test.actor)

			if test.expected == nil {
				assert.Nil(t, result)
				return
			}

			assert.True(t, proto.Equal(test.expected, result), "expected %v, got %v", test.expected, result)
		})
	}
}

func TestToPBGroup_CreatedByActor(t *testing.T) {
	now := time.Now()
	metadata := models.ResourceMetadata{
		ID:                   "1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d",
		TRN:                  "trn:group:my-group",
		CreationTimestamp:    &now,
		LastUpdatedTimestamp: &now,
	}

	t.Run("created by is the actor subject and the actor is populated", func(t *testing.T) {
		group := &models.Group{
			Metadata:  metadata,
			Name:      "my-group",
			FullPath:  "my-group",
			CreatedBy: models.NewUserActor(testActorID).WithSubject("user@example.com"),
		}

		result := toPBGroup(group)

		// createdBy keeps its existing value; the actor is additive.
		assert.Equal(t, "user@example.com", result.CreatedBy)
		require.NotNil(t, result.CreatedByActor)
		require.NotNil(t, result.CreatedByActor.GetUser())
		assert.Equal(t, gid.ToGlobalID(types.UserModelType, testActorID), result.CreatedByActor.GetUser().PrincipalId)
	})

	t.Run("an untyped actor keeps created by and reports no actor", func(t *testing.T) {
		group := &models.Group{
			Metadata:  metadata,
			Name:      "my-group",
			FullPath:  "my-group",
			CreatedBy: models.Actor{Subject: "legacy@example.com"},
		}

		result := toPBGroup(group)

		assert.Equal(t, "legacy@example.com", result.CreatedBy)
		assert.Nil(t, result.CreatedByActor)
	})
}

func TestToPBWorkspaceRoleBinding_CreatedByActor(t *testing.T) {
	now := time.Now()
	binding := &models.WorkspaceRoleBinding{
		Metadata: models.ResourceMetadata{
			ID:                   "1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d",
			TRN:                  "trn:workspace_role_binding:my-group/my-workspace",
			CreationTimestamp:    &now,
			LastUpdatedTimestamp: &now,
		},
		WorkspaceID: "2b3c4d5e-6f7a-8b9c-0d1e-2f3a4b5c6d7e",
		RoleID:      "3c4d5e6f-7a8b-9c0d-1e2f-3a4b5c6d7e8f",
		CreatedBy:   models.NewServiceAccountActor(testActorID).WithSubject("my-group/my-sa"),
	}

	result := toPBWorkspaceRoleBinding(binding)

	assert.Equal(t, "my-group/my-sa", result.CreatedBy)
	require.NotNil(t, result.CreatedByActor)
	require.NotNil(t, result.CreatedByActor.GetServiceAccount())
	assert.Equal(t, gid.ToGlobalID(types.ServiceAccountModelType, testActorID), result.CreatedByActor.GetServiceAccount().PrincipalId)
}

func ptrString(s string) *string {
	return &s
}
