//go:build integration

package db

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

const (
	testDBActorID      = "3f4a5b6c-7d8e-9f0a-1b2c-3d4e5f6a7b8c"
	testDBActorSubject = "someone@example.com"
)

func TestActorColumns(t *testing.T) {
	tests := []struct {
		name         string
		actor        models.Actor
		expectedType sql.NullString
		expectedID   sql.NullString
		expectedTRN  sql.NullString
		expectError  bool
	}{
		{
			name:         "a user actor is written as type and id",
			actor:        models.NewUserActor(testDBActorID).WithSubject(testDBActorSubject),
			expectedType: sql.NullString{String: "user", Valid: true},
			expectedID:   sql.NullString{String: testDBActorID, Valid: true},
		},
		{
			name:         "a system actor is written as a type with a NULL id",
			actor:        models.SystemActor(),
			expectedType: sql.NullString{String: "system", Valid: true},
			expectedID:   sql.NullString{},
		},
		{
			name:         "a service account actor is written with its TRN",
			actor:        models.NewServiceAccountActor(testDBActorID).WithTRN("trn:service_account:group/sa"),
			expectedType: sql.NullString{String: "service_account", Valid: true},
			expectedID:   sql.NullString{String: testDBActorID, Valid: true},
			expectedTRN:  sql.NullString{String: "trn:service_account:group/sa", Valid: true},
		},
		{
			name:         "a job actor without a TRN is written with a NULL TRN",
			actor:        models.NewJobActor(testDBActorID),
			expectedType: sql.NullString{String: "job", Valid: true},
			expectedID:   sql.NullString{String: testDBActorID, Valid: true},
		},
		{
			name:        "a user actor with a TRN is an error",
			actor:       models.NewUserActor(testDBActorID).WithTRN("someone"),
			expectError: true,
		},
		{
			name:  "a zero actor is written as NULL",
			actor: models.Actor{},
		},
		{
			name:  "a subject only actor is written as NULL",
			actor: models.Actor{Subject: testDBActorSubject},
		},
		{
			name:        "an id bearing actor with no id is an error",
			actor:       models.Actor{Type: models.ActorTypeUser, Subject: testDBActorSubject},
			expectError: true,
		},
		{
			name:        "a valueless actor carrying an id is an error",
			actor:       models.Actor{Type: models.ActorTypeSystem, ID: testDBActorID},
			expectError: true,
		},
		{
			name:        "an unknown actor type is an error",
			actor:       models.Actor{Type: "runner", ID: testDBActorID},
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actorType, actorID, actorTRN, err := actorColumns(test.actor)

			if test.expectError {
				require.Error(t, err)
				assert.Equal(t, errors.EInternal, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.expectedType, actorType)
			assert.Equal(t, test.expectedID, actorID)
			assert.Equal(t, test.expectedTRN, actorTRN)
		})
	}
}

func TestScanActor(t *testing.T) {
	subject := sql.NullString{String: testDBActorSubject, Valid: true}

	tests := []struct {
		name          string
		subjectColumn sql.NullString
		typeColumn    sql.NullString
		idColumn      sql.NullString
		trnColumn     sql.NullString
		expected      models.Actor
	}{
		{
			name:          "a user actor round trips with its subject",
			subjectColumn: subject,
			typeColumn:    sql.NullString{String: "user", Valid: true},
			idColumn:      sql.NullString{String: testDBActorID, Valid: true},
			expected:      models.NewUserActor(testDBActorID).WithSubject(testDBActorSubject),
		},
		{
			name:          "a system actor round trips",
			subjectColumn: sql.NullString{String: "system", Valid: true},
			typeColumn:    sql.NullString{String: "system", Valid: true},
			expected:      models.SystemActor(),
		},
		{
			name:          "a service account actor round trips with its TRN",
			subjectColumn: sql.NullString{String: "group/sa", Valid: true},
			typeColumn:    sql.NullString{String: "service_account", Valid: true},
			idColumn:      sql.NullString{String: testDBActorID, Valid: true},
			trnColumn:     sql.NullString{String: "trn:service_account:group/sa", Valid: true},
			expected:      models.NewServiceAccountActor(testDBActorID).WithTRN("trn:service_account:group/sa").WithSubject("group/sa"),
		},
		{
			name:     "NULL columns yield no actor",
			expected: models.Actor{},
		},
		{
			// A row written before typed actors existed.
			name:          "a subject without a type yields a subject only actor",
			subjectColumn: subject,
			expected:      models.Actor{Subject: testDBActorSubject},
		},
		{
			// The type column is authoritative: without it the id is ignored.
			name:          "an id without a type yields a subject only actor",
			subjectColumn: subject,
			idColumn:      sql.NullString{String: testDBActorID, Valid: true},
			expected:      models.Actor{Subject: testDBActorSubject},
		},
		{
			// Forward compatibility: a type written by a newer version reads as a
			// legacy row rather than failing the query.
			name:          "an unrecognized type yields a subject only actor",
			subjectColumn: subject,
			typeColumn:    sql.NullString{String: "runner", Valid: true},
			idColumn:      sql.NullString{String: testDBActorID, Valid: true},
			expected:      models.Actor{Subject: testDBActorSubject},
		},
		{
			name:       "an unrecognized type with no subject yields no actor",
			typeColumn: sql.NullString{String: "runner", Valid: true},
			idColumn:   sql.NullString{String: testDBActorID, Valid: true},
			expected:   models.Actor{},
		},
		{
			// A malformed row is read like a legacy row rather than failing the
			// query.
			name:          "an id bearing type with a NULL id yields a subject only actor",
			subjectColumn: subject,
			typeColumn:    sql.NullString{String: "user", Valid: true},
			expected:      models.Actor{Subject: testDBActorSubject},
		},
		{
			name:       "a valueless type with an id and no subject yields no actor",
			typeColumn: sql.NullString{String: "system", Valid: true},
			idColumn:   sql.NullString{String: testDBActorID, Valid: true},
			expected:   models.Actor{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, scanActor(test.subjectColumn, test.typeColumn, test.idColumn, test.trnColumn))
		})
	}
}

// TestActorColumns_RoundTripThroughScanActor pins the invariant that
// actorColumns never writes something scanActor cannot read. The
// subject column is written from GetSubject, as the CreateX functions do.
func TestActorColumns_RoundTripThroughScanActor(t *testing.T) {
	tests := []struct {
		name     string
		actor    models.Actor
		expected models.Actor
	}{
		{
			name:     "user",
			actor:    models.NewUserActor(testDBActorID).WithSubject(testDBActorSubject),
			expected: models.NewUserActor(testDBActorID).WithSubject(testDBActorSubject),
		},
		{
			name:     "service account",
			actor:    models.NewServiceAccountActor(testDBActorID).WithTRN("trn:service_account:group/sa").WithSubject("group/sa"),
			expected: models.NewServiceAccountActor(testDBActorID).WithTRN("trn:service_account:group/sa").WithSubject("group/sa"),
		},
		{
			name:     "job",
			actor:    models.NewJobActor(testDBActorID).WithTRN("trn:job:group/ws/job-gid").WithSubject("trn:job:group/ws/x"),
			expected: models.NewJobActor(testDBActorID).WithTRN("trn:job:group/ws/job-gid").WithSubject("trn:job:group/ws/x"),
		},
		{
			name:     "vcs provider",
			actor:    models.NewVCSProviderActor(testDBActorID).WithTRN("trn:vcs_provider:group/vcs").WithSubject("group/vcs"),
			expected: models.NewVCSProviderActor(testDBActorID).WithTRN("trn:vcs_provider:group/vcs").WithSubject("group/vcs"),
		},
		{
			name:     "system",
			actor:    models.SystemActor(),
			expected: models.SystemActor(),
		},
		{
			name:     "scim",
			actor:    models.SCIMActor(),
			expected: models.SCIMActor(),
		},
		{
			name:     "subject only",
			actor:    models.Actor{Subject: testDBActorSubject},
			expected: models.Actor{Subject: testDBActorSubject},
		},
		{
			name:     "zero",
			actor:    models.Actor{},
			expected: models.Actor{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			writtenSubject := sql.NullString{String: test.actor.Subject, Valid: test.actor.Subject != ""}
			writtenType, writtenID, writtenTRN, err := actorColumns(test.actor)
			require.NoError(t, err)
			assert.Equal(t, test.expected.HasType(), writtenType.Valid)
			assert.Equal(t, test.expected.HasType() && test.expected.Type.HasID(), writtenID.Valid)
			assert.Equal(t, test.expected.TRN != "", writtenTRN.Valid)

			assert.Equal(t, test.expected, scanActor(writtenSubject, writtenType, writtenID, writtenTRN))
		})
	}
}

func TestMetadataAndCreatedByFieldList(t *testing.T) {
	assert.Equal(t, []interface{}{
		"id", "created_at", "updated_at", "version",
		"created_by", "created_by_actor_type", "created_by_actor_id", "created_by_actor_trn",
	}, metadataAndCreatedByFieldList, "the created_by columns follow the metadata columns")

	// Tables append their own columns to it, so two appends must never share a
	// backing array.
	first := append(metadataAndCreatedByFieldList, "first")
	second := append(metadataAndCreatedByFieldList, "second")
	assert.Equal(t, "first", first[len(first)-1])
	assert.Equal(t, "second", second[len(second)-1])
	assert.Equal(t, []interface{}{"id", "created_at", "updated_at", "version"}, metadataFieldList)
}

// TestCreatedByActorRoundTrip verifies that an actor survives a write and read
// through a real database.
//
// The other integration tests mostly write subject-only actors, so they leave the
// created_by_actor columns NULL and only prove the columns are wired into the
// field list and scan destinations without error. This exercises the values
// themselves: every actor type with its Subject, the legacy subject-only case,
// and the invalid actors that actorColumns refuses to persist.
func TestCreatedByActorRoundTrip(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	const (
		userID  = "b1c2d3e4-f5a6-4b7c-8d9e-0f1a2b3c4d5e"
		saID    = "c2d3e4f5-a6b7-4c8d-9e0f-1a2b3c4d5e6f"
		subject = "creator@example.com"
	)

	tests := []struct {
		name          string
		actor         models.Actor
		expectedActor models.Actor
		// expectErrorCode is set when the write must fail.
		expectErrorCode errors.CodeType
	}{
		{
			name:          "user actor",
			actor:         models.NewUserActor(userID).WithSubject(subject),
			expectedActor: models.NewUserActor(userID).WithSubject(subject),
		},
		{
			name:          "service account actor",
			actor:         models.NewServiceAccountActor(saID).WithTRN("trn:service_account:group/sa").WithSubject("group/sa"),
			expectedActor: models.NewServiceAccountActor(saID).WithTRN("trn:service_account:group/sa").WithSubject("group/sa"),
		},
		{
			name:          "job actor",
			actor:         models.NewJobActor(userID).WithTRN("trn:job:group/ws/job-gid").WithSubject("trn:job:group/ws/job"),
			expectedActor: models.NewJobActor(userID).WithTRN("trn:job:group/ws/job-gid").WithSubject("trn:job:group/ws/job"),
		},
		{
			name:          "vcs provider actor",
			actor:         models.NewVCSProviderActor(userID).WithTRN("trn:vcs_provider:group/vcs-provider").WithSubject("group/vcs-provider"),
			expectedActor: models.NewVCSProviderActor(userID).WithTRN("trn:vcs_provider:group/vcs-provider").WithSubject("group/vcs-provider"),
		},
		{
			name:          "system actor",
			actor:         models.SystemActor(),
			expectedActor: models.SystemActor(),
		},
		{
			name:          "scim actor",
			actor:         models.SCIMActor(),
			expectedActor: models.SCIMActor(),
		},
		{
			// How the backfill records a principal it could not identify.
			name:          "unmapped actor",
			actor:         models.NewUserActor(models.UnmappedActorID).WithSubject(subject),
			expectedActor: models.NewUserActor(models.UnmappedActorID).WithSubject(subject),
		},
		{
			// How every row written before the type columns existed behaves.
			name:          "legacy subject only actor",
			actor:         models.Actor{Subject: subject},
			expectedActor: models.Actor{Subject: subject},
		},
		{
			// Validate leaves the ID's format to the database: the UUID column
			// rejects it, so the write still fails.
			name:            "an actor with a non uuid id is rejected by the database",
			actor:           models.Actor{Type: models.ActorTypeUser, ID: "not-a-uuid", Subject: subject},
			expectErrorCode: errors.EInvalid,
		},
		{
			name:            "an actor with a missing id is rejected",
			actor:           models.Actor{Type: models.ActorTypeUser, Subject: subject},
			expectErrorCode: errors.EInternal,
		},
		{
			name:            "an unknown actor type is rejected",
			actor:           models.Actor{Type: "runner", ID: userID, Subject: subject},
			expectErrorCode: errors.EInternal,
		},
	}

	for i, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			created, cErr := testClient.client.Groups.CreateGroup(ctx, &models.Group{
				Name:        groupNameForActorTest(i),
				Description: "actor round trip",
				CreatedBy:   test.actor,
			})
			if test.expectErrorCode != "" {
				require.Error(t, cErr)
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(cErr))
				return
			}
			require.NoError(t, cErr)
			require.NotNil(t, created)

			// The value returned by the insert comes from RETURNING, so this already
			// reflects what the database stored.
			assert.Equal(t, test.expectedActor, created.CreatedBy)

			// Re-reading exercises the SELECT field list and scan path.
			reread, gErr := testClient.client.Groups.GetGroupByID(ctx, created.Metadata.ID)
			require.NoError(t, gErr)
			require.NotNil(t, reread)
			assert.Equal(t, test.expectedActor, reread.CreatedBy)
		})
	}
}

// TestCreatedByActorRoundTrip_MalformedRow verifies that a row whose actor
// columns were written outside the application, and do not form a valid actor,
// reads back as a subject-only actor instead of failing the query.
func TestCreatedByActorRoundTrip_MalformedRow(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	const subject = "creator@example.com"

	created, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:      "actor-malformed",
		CreatedBy: models.Actor{Subject: subject},
	})
	require.NoError(t, err)

	// A user actor needs an id, so a NULL id makes the row malformed.
	_, err = testClient.client.getConnection(ctx).Exec(ctx,
		"UPDATE groups SET created_by_actor_type = 'user', created_by_actor_id = NULL WHERE id = $1",
		created.Metadata.ID)
	require.NoError(t, err)

	reread, err := testClient.client.Groups.GetGroupByID(ctx, created.Metadata.ID)
	require.NoError(t, err)
	require.NotNil(t, reread)
	assert.Equal(t, models.Actor{Subject: subject}, reread.CreatedBy)
}

func groupNameForActorTest(i int) string {
	return "actor-roundtrip-" + string(rune('a'+i))
}
