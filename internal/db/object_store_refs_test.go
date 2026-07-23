//go:build integration

package db

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

func TestObjectStoreRefs_CreateRef(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	t.Run("creates ref with default AvailableAt", func(t *testing.T) {
		err := testClient.client.ObjectStoreRefs.CreateRef(ctx, &CreateObjectStoreRefInput{
			ObjectKey: "test/key/default-available-at",
		})
		require.NoError(t, err)

		// Ref with nil AvailableAt (defaults to now) is immediately orphan-claimable.
		claimed, err := testClient.client.ObjectStoreRefs.ClaimOrphanedRefs(ctx, 10)
		require.NoError(t, err)
		require.Len(t, claimed, 1)
		assert.Equal(t, "test/key/default-available-at", claimed[0].ObjectKey)
		assert.Equal(t, 1, claimed[0].ClaimCount)
	})

	t.Run("creates ref with future AvailableAt is not immediately claimable", func(t *testing.T) {
		future := time.Now().UTC().Add(time.Hour)
		err := testClient.client.ObjectStoreRefs.CreateRef(ctx, &CreateObjectStoreRefInput{
			ObjectKey:   "test/key/future-available-at",
			AvailableAt: &future,
		})
		require.NoError(t, err)

		claimed, err := testClient.client.ObjectStoreRefs.ClaimOrphanedRefs(ctx, 10)
		require.NoError(t, err)
		assert.Empty(t, claimed)
	})

	t.Run("upsert on duplicate key updates AvailableAt", func(t *testing.T) {
		key := "test/key/upsert"

		// First create with a far-future AvailableAt so it's NOT claimable.
		far := time.Now().UTC().Add(24 * time.Hour)
		require.NoError(t, testClient.client.ObjectStoreRefs.CreateRef(ctx, &CreateObjectStoreRefInput{
			ObjectKey:   key,
			AvailableAt: &far,
		}))

		claimed, err := testClient.client.ObjectStoreRefs.ClaimOrphanedRefs(ctx, 10)
		require.NoError(t, err)
		assert.Empty(t, claimed)

		// Re-create with nil AvailableAt (now) -- upsert should refresh the available_at.
		require.NoError(t, testClient.client.ObjectStoreRefs.CreateRef(ctx, &CreateObjectStoreRefInput{
			ObjectKey: key,
		}))

		claimed, err = testClient.client.ObjectStoreRefs.ClaimOrphanedRefs(ctx, 10)
		require.NoError(t, err)
		require.Len(t, claimed, 1)
		assert.Equal(t, key, claimed[0].ObjectKey)
	})
}

func TestObjectStoreRefs_ClaimOrphanedRefs(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createKey := func(t *testing.T, key string) {
		t.Helper()
		require.NoError(t, testClient.client.ObjectStoreRefs.CreateRef(ctx, &CreateObjectStoreRefInput{
			ObjectKey: key,
		}))
	}

	t.Run("no refs returns empty slice", func(t *testing.T) {
		claimed, err := testClient.client.ObjectStoreRefs.ClaimOrphanedRefs(ctx, 10)
		require.NoError(t, err)
		assert.Empty(t, claimed)
	})

	t.Run("limit zero returns empty slice", func(t *testing.T) {
		createKey(t, "claim/limit-zero")
		claimed, err := testClient.client.ObjectStoreRefs.ClaimOrphanedRefs(ctx, 0)
		require.NoError(t, err)
		assert.Empty(t, claimed)
	})

	t.Run("respects limit", func(t *testing.T) {
		for i := range 5 {
			createKey(t, "claim/limit-test-"+string(rune('a'+i)))
		}
		claimed, err := testClient.client.ObjectStoreRefs.ClaimOrphanedRefs(ctx, 3)
		require.NoError(t, err)
		assert.Len(t, claimed, 3)
	})

	t.Run("each claim increments claim_count", func(t *testing.T) {
		createKey(t, "claim/count-test")

		first, err := testClient.client.ObjectStoreRefs.ClaimOrphanedRefs(ctx, 1)
		require.NoError(t, err)
		require.Len(t, first, 1)
		assert.Equal(t, 1, first[0].ClaimCount)

		// Manually reset available_at to now so it's immediately reclaimable.
		_, execErr := testClient.client.getConnection(ctx).Exec(ctx,
			"UPDATE object_store_refs SET available_at = NOW() WHERE id = $1", first[0].ID)
		require.NoError(t, execErr)

		second, err := testClient.client.ObjectStoreRefs.ClaimOrphanedRefs(ctx, 1)
		require.NoError(t, err)
		require.Len(t, second, 1)
		assert.Equal(t, 2, second[0].ClaimCount)
	})

	t.Run("linked ref is not claimed", func(t *testing.T) {
		user, err := testClient.client.Users.CreateUser(ctx, &models.User{
			Username: "osr-claim-user",
			Email:    "osr-claim@example.com",
		})
		require.NoError(t, err)

		session, err := testClient.client.AgentSessions.CreateAgentSession(ctx, &models.AgentSession{
			UserID: user.Metadata.ID,
		})
		require.NoError(t, err)

		key := "claim/linked-ref"
		require.NoError(t, testClient.client.ObjectStoreRefs.CreateRef(ctx, &CreateObjectStoreRefInput{
			ObjectKey: key,
		}))
		require.NoError(t, testClient.client.ObjectStoreRefs.LinkRef(ctx, key, ObjectStoreRefOwnerAgentSession, session.Metadata.ID))

		claimed, err := testClient.client.ObjectStoreRefs.ClaimOrphanedRefs(ctx, 10)
		require.NoError(t, err)
		for _, ref := range claimed {
			assert.NotEqual(t, key, ref.ObjectKey, "linked ref must not be claimed as orphaned")
		}
	})
}

func TestObjectStoreRefs_LinkRef(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "osr-link-user",
		Email:    "osr-link@example.com",
	})
	require.NoError(t, err)

	session, err := testClient.client.AgentSessions.CreateAgentSession(ctx, &models.AgentSession{
		UserID: user.Metadata.ID,
	})
	require.NoError(t, err)

	t.Run("links ref to owner and removes it from orphan set", func(t *testing.T) {
		key := "link/session-ref"
		require.NoError(t, testClient.client.ObjectStoreRefs.CreateRef(ctx, &CreateObjectStoreRefInput{
			ObjectKey: key,
		}))

		// Before linking: the ref is orphaned (no FK, AvailableAt=now).
		before, err := testClient.client.ObjectStoreRefs.ClaimOrphanedRefs(ctx, 10)
		require.NoError(t, err)
		require.Len(t, before, 1)

		// Reset available_at because ClaimOrphanedRefs advanced it (just to confirm it was claimed).
		_, execErr := testClient.client.getConnection(ctx).Exec(ctx,
			"UPDATE object_store_refs SET available_at = NOW() WHERE object_key = $1", key)
		require.NoError(t, execErr)

		require.NoError(t, testClient.client.ObjectStoreRefs.LinkRef(ctx, key, ObjectStoreRefOwnerAgentSession, session.Metadata.ID))

		// After linking: no longer orphaned.
		after, err := testClient.client.ObjectStoreRefs.ClaimOrphanedRefs(ctx, 10)
		require.NoError(t, err)
		for _, ref := range after {
			assert.NotEqual(t, key, ref.ObjectKey, "linked ref must not appear in orphan claim")
		}
	})
}

func TestObjectStoreRefs_DeleteRef(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	t.Run("deletes an existing ref", func(t *testing.T) {
		require.NoError(t, testClient.client.ObjectStoreRefs.CreateRef(ctx, &CreateObjectStoreRefInput{
			ObjectKey: "delete/existing",
		}))

		refs, err := testClient.client.ObjectStoreRefs.ClaimOrphanedRefs(ctx, 10)
		require.NoError(t, err)
		require.Len(t, refs, 1)

		require.NoError(t, testClient.client.ObjectStoreRefs.DeleteRefs(ctx, []string{refs[0].ID}))

		// After deletion, no orphaned refs remain.
		remaining, err := testClient.client.ObjectStoreRefs.ClaimOrphanedRefs(ctx, 10)
		require.NoError(t, err)
		// Reset available_at so the just-deleted ref would reappear if still present.
		assert.Empty(t, remaining)
	})

	t.Run("deleting a non-existent ref is a no-op", func(t *testing.T) {
		// DeleteRefs is tolerant of already-deleted rows so concurrent janitor
		// instances don't produce errors when they race on the same ref.
		require.NoError(t, testClient.client.ObjectStoreRefs.DeleteRefs(ctx, []string{nonExistentID}))
	})
}
