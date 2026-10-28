package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
)

const (
	testActorUserID           = "0d4e1a2b-3c4d-5e6f-7a8b-9c0d1e2f3a4b"
	testActorServiceAccountID = "1e5f2b3c-4d5e-6f7a-8b9c-0d1e2f3a4b5c"
	testActorJobID            = "2f6a3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d"
	testActorVCSProviderID    = "3a7b4d5e-6f7a-8b9c-0d1e-2f3a4b5c6d7e"
)

// TestCaller_GetActor covers every concrete Caller implementation. Adding a new
// caller type is already a compile error because GetActor is on the interface;
// this pins the actor each one reports, asserts its Subject is GetSubject so the
// legacy created_by column keeps its value, asserts it is valid, and asserts that
// changing a returned actor does not affect the caller.
func TestCaller_GetActor(t *testing.T) {
	tests := []struct {
		name     string
		caller   Caller
		expected models.Actor
	}{
		{
			name: "user caller reports a user actor",
			caller: &UserCaller{
				User: &models.User{Metadata: models.ResourceMetadata{ID: testActorUserID}, Email: "user@example.com"},
			},
			expected: models.NewUserActor(testActorUserID).WithSubject("user@example.com"),
		},
		{
			name:     "service account caller reports a service account actor",
			caller:   &ServiceAccountCaller{ServiceAccountID: testActorServiceAccountID, ServiceAccountPath: "my-group/my-sa"},
			expected: models.NewServiceAccountActor(testActorServiceAccountID).WithTRN("trn:service_account:my-group/my-sa").WithSubject("my-group/my-sa"),
		},
		{
			name: "job caller reports a job actor",
			caller: &JobCaller{
				Job:    &models.Job{Metadata: models.ResourceMetadata{ID: testActorJobID}},
				JobTRN: "trn:job:my-group/my-workspace/job-gid",
			},
			expected: models.NewJobActor(testActorJobID).WithTRN("trn:job:my-group/my-workspace/job-gid").WithSubject("trn:job:my-group/my-workspace/job-gid"),
		},
		{
			name: "vcs workspace link caller reports a vcs provider actor",
			caller: &VCSWorkspaceLinkCaller{
				Provider: &models.VCSProvider{Metadata: models.ResourceMetadata{
					ID:  testActorVCSProviderID,
					TRN: trn.TypeVCSProvider.Build("my-group/my-provider"),
				}},
			},
			expected: models.NewVCSProviderActor(testActorVCSProviderID).WithTRN("trn:vcs_provider:my-group/my-provider").WithSubject("my-group/my-provider"),
		},
		{
			name:     "system caller reports a system actor",
			caller:   &SystemCaller{},
			expected: models.SystemActor(),
		},
		{
			name:     "scim caller reports a scim actor",
			caller:   &SCIMCaller{},
			expected: models.SCIMActor(),
		},
		{
			// A read-only caller that is never recorded as a creator.
			name:     "federated registry caller reports a subject only actor",
			caller:   &FederatedRegistryCaller{subject: "federated-subject"},
			expected: models.Actor{Subject: "federated-subject"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actor := test.caller.GetActor()
			require.NotNil(t, actor)
			assert.Equal(t, test.expected, actor)
			assert.Equal(t, test.caller.GetSubject(), actor.Subject)
			if test.expected.HasType() {
				require.NoError(t, actor.Validate())
			}

			// Mutating a returned actor must not affect the caller's state.
			other := test.caller.GetActor()
			actor.ID = "mutated"
			assert.Equal(t, test.expected.ID, other.ID)
		})
	}
}
