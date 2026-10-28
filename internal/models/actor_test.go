package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

const (
	testActorUserID    = "6f2c4d8e-1a3b-4c5d-8e9f-0a1b2c3d4e5f"
	testActorOtherUUID = "11112222-3333-4444-5555-666677778888"
)

func TestActor_String(t *testing.T) {
	tests := []struct {
		name     string
		actor    Actor
		expected string
	}{
		{
			name:     "user actor is type and id",
			actor:    NewUserActor(testActorUserID),
			expected: "user:" + testActorUserID,
		},
		{
			name:     "service account actor is type and id",
			actor:    NewServiceAccountActor(testActorUserID),
			expected: "service_account:" + testActorUserID,
		},
		{
			name:     "job actor is type and id",
			actor:    NewJobActor(testActorUserID),
			expected: "job:" + testActorUserID,
		},
		{
			name:     "vcs provider actor is type and id",
			actor:    NewVCSProviderActor(testActorUserID),
			expected: "vcs_provider:" + testActorUserID,
		},
		{
			name:     "system actor is the bare type name",
			actor:    SystemActor(),
			expected: "system",
		},
		{
			name:     "scim actor is the bare type name",
			actor:    SCIMActor(),
			expected: "scim",
		},
		{
			name:     "subject is not part of the string form",
			actor:    NewUserActor(testActorUserID).WithSubject("user@example.com"),
			expected: "user:" + testActorUserID,
		},
		{
			name:     "untyped actor is empty",
			actor:    Actor{Subject: "user@example.com"},
			expected: "",
		},
		{
			name:     "zero actor is empty",
			actor:    Actor{},
			expected: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, test.actor.String())
		})
	}
}

func TestActorFromParts(t *testing.T) {
	tests := []struct {
		name        string
		subject     string
		actorType   string
		id          string
		trn         string
		expectActor Actor
	}{
		{
			name:        "user actor",
			subject:     "user@example.com",
			actorType:   "user",
			id:          testActorUserID,
			expectActor: Actor{Type: ActorTypeUser, ID: testActorUserID, Subject: "user@example.com"},
		},
		{
			name:        "service account actor",
			subject:     "my-group/my-service-account",
			actorType:   "service_account",
			id:          testActorUserID,
			trn:         "trn:service_account:my-group/my-service-account",
			expectActor: Actor{Type: ActorTypeServiceAccount, ID: testActorUserID, TRN: "trn:service_account:my-group/my-service-account", Subject: "my-group/my-service-account"},
		},
		{
			name:        "system actor",
			subject:     "system",
			actorType:   "system",
			expectActor: Actor{Type: ActorTypeSystem, Subject: "system"},
		},
		{
			name:        "empty type is an untyped actor carrying the subject",
			subject:     "user@example.com",
			expectActor: Actor{Subject: "user@example.com"},
		},
		{
			name:        "empty type ignores a stray id",
			subject:     "user@example.com",
			id:          testActorUserID,
			expectActor: Actor{Subject: "user@example.com"},
		},
		{
			name: "all empty is no actor",
		},
		{
			// Forward compatibility: a type written by a newer version reads like a
			// legacy row rather than failing the request.
			name:        "unrecognized type is an untyped actor carrying the subject",
			subject:     "runner-subject",
			actorType:   "runner",
			id:          testActorUserID,
			expectActor: Actor{Subject: "runner-subject"},
		},
		{
			name:      "unrecognized type without a subject is no actor",
			actorType: "runner",
			id:        testActorUserID,
		},
		{
			// A malformed row reads like a legacy row rather than failing the
			// request.
			name:        "id bearing type without an id is an untyped actor carrying the subject",
			subject:     "user@example.com",
			actorType:   "user",
			expectActor: Actor{Subject: "user@example.com"},
		},
		{
			name:      "valueless type with an id and no subject is no actor",
			actorType: "scim",
			id:        testActorUserID,
		},
		{
			// The write path never stores a user TRN, so one can only come from outside
			// the application; the row reads like a legacy row rather than exposing it.
			name:        "user actor with a TRN is an untyped actor carrying the subject",
			subject:     "user@example.com",
			actorType:   "user",
			id:          testActorUserID,
			trn:         "user",
			expectActor: Actor{Subject: "user@example.com"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expectActor, ActorFromParts(test.subject, test.actorType, test.id, test.trn))
		})
	}
}

func TestActor_Validate(t *testing.T) {
	tests := []struct {
		name            string
		actor           Actor
		expectErrorCode errors.CodeType
	}{
		{
			name:  "valid user actor",
			actor: NewUserActor(testActorUserID),
		},
		{
			name:  "valid service account actor",
			actor: NewServiceAccountActor(testActorUserID),
		},
		{
			name:  "valid job actor",
			actor: NewJobActor(testActorUserID),
		},
		{
			name:  "valid vcs provider actor",
			actor: NewVCSProviderActor(testActorUserID),
		},
		{
			name:  "valid system actor",
			actor: SystemActor(),
		},
		{
			name:  "valid scim actor",
			actor: SCIMActor(),
		},
		{
			name:  "subject does not affect validity",
			actor: NewUserActor(testActorUserID).WithSubject("user@example.com"),
		},
		{
			name:            "zero actor is invalid",
			actor:           Actor{},
			expectErrorCode: errors.EInvalid,
		},
		{
			name:            "untyped actor is invalid",
			actor:           Actor{Subject: "user@example.com"},
			expectErrorCode: errors.EInvalid,
		},
		{
			name:            "unknown type is invalid",
			actor:           Actor{Type: ActorType("runner"), ID: testActorUserID},
			expectErrorCode: errors.EInvalid,
		},
		{
			name:            "user actor without an id is invalid",
			actor:           Actor{Type: ActorTypeUser},
			expectErrorCode: errors.EInvalid,
		},
		{
			name:            "system actor with an id is invalid",
			actor:           Actor{Type: ActorTypeSystem, ID: testActorUserID},
			expectErrorCode: errors.EInvalid,
		},
		{
			name:  "service account actor with a TRN is valid",
			actor: NewServiceAccountActor(testActorUserID).WithTRN("trn:service_account:group/sa"),
		},
		{
			name:  "job actor with a TRN is valid",
			actor: NewJobActor(testActorUserID).WithTRN("trn:job:group/ws/job-gid"),
		},
		{
			name:  "vcs provider actor with a TRN is valid",
			actor: NewVCSProviderActor(testActorUserID).WithTRN("trn:vcs_provider:group/vcs"),
		},
		{
			name:            "user actor with a TRN is invalid",
			actor:           NewUserActor(testActorUserID).WithTRN("someone"),
			expectErrorCode: errors.EInvalid,
		},
		{
			name:            "system actor with a TRN is invalid",
			actor:           SystemActor().WithTRN("system"),
			expectErrorCode: errors.EInvalid,
		},
		{
			name:            "scim actor with a TRN is invalid",
			actor:           SCIMActor().WithTRN("scim"),
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.actor.Validate()

			if test.expectErrorCode != "" {
				require.Error(t, err)
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestActor_GlobalID(t *testing.T) {
	tests := []struct {
		name     string
		actor    Actor
		expected string
	}{
		{
			name:     "user actor",
			actor:    NewUserActor(testActorUserID),
			expected: gid.ToGlobalID(types.UserModelType, testActorUserID),
		},
		{
			name:     "service account actor",
			actor:    NewServiceAccountActor(testActorOtherUUID),
			expected: gid.ToGlobalID(types.ServiceAccountModelType, testActorOtherUUID),
		},
		{
			name:     "job actor",
			actor:    NewJobActor(testActorUserID),
			expected: gid.ToGlobalID(types.JobModelType, testActorUserID),
		},
		{
			name:     "vcs provider actor",
			actor:    NewVCSProviderActor(testActorUserID),
			expected: gid.ToGlobalID(types.VCSProviderModelType, testActorUserID),
		},
		{
			name:     "system actor has no global id",
			actor:    SystemActor(),
			expected: "",
		},
		{
			name:     "scim actor has no global id",
			actor:    SCIMActor(),
			expected: "",
		},
		{
			name:     "zero actor has no global id",
			actor:    Actor{},
			expected: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, test.actor.GlobalID())
		})
	}
}

func TestActorType_HasID(t *testing.T) {
	withID := []ActorType{
		ActorTypeUser,
		ActorTypeServiceAccount,
		ActorTypeJob,
		ActorTypeVCSProvider,
	}
	withoutID := []ActorType{
		ActorTypeSystem,
		ActorTypeSCIM,
	}

	for _, actorType := range withID {
		t.Run(string(actorType)+" has an id", func(t *testing.T) {
			assert.True(t, actorType.HasID())
		})
	}

	for _, actorType := range withoutID {
		t.Run(string(actorType)+" has no id", func(t *testing.T) {
			assert.False(t, actorType.HasID())
		})
	}
}

func TestActor_ModelType(t *testing.T) {
	mt, ok := NewUserActor(testActorUserID).ModelType()
	require.True(t, ok)
	assert.True(t, mt.Equals(types.UserModelType))

	_, ok = SystemActor().ModelType()
	assert.False(t, ok)
}

func TestActor_HasType(t *testing.T) {
	var nilActor Actor
	assert.False(t, nilActor.HasType())
	assert.False(t, (Actor{}).HasType())
	assert.False(t, (Actor{Subject: "user@example.com"}).HasType())
	assert.True(t, SystemActor().HasType())
	assert.True(t, NewUserActor(testActorUserID).HasType())
}

func TestActor_UnmappedActorIDIsValid(t *testing.T) {
	assert.NoError(t, NewServiceAccountActor(UnmappedActorID).Validate(), "an unmapped actor is still a valid actor")
}

func TestActor_IsZero(t *testing.T) {
	assert.True(t, (Actor{}).IsZero())
	assert.False(t, (Actor{Subject: "legacy@example.com"}).IsZero(), "a legacy actor records a principal")
	assert.False(t, NewUserActor(testActorUserID).IsZero())
	assert.False(t, SystemActor().IsZero())
}

func TestActorSubjectPtr(t *testing.T) {
	assert.Nil(t, ActorSubjectPtr(nil))
	assert.Nil(t, ActorSubjectPtr(&Actor{}))
	subject := ActorSubjectPtr(new(NewUserActor(testActorUserID).WithSubject("user@example.com")))
	require.NotNil(t, subject)
	assert.Equal(t, "user@example.com", *subject)
}

func TestActor_WithDoesNotModifyTheOriginal(t *testing.T) {
	original := NewUserActor(testActorUserID)
	withSubject := original.WithSubject("user@example.com")
	assert.Equal(t, "", original.Subject)
	assert.Equal(t, "user@example.com", withSubject.Subject)
}
