//go:build integration

package db

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeClaimable resets a work item's available_at into the past so it can be claimed
// again in the same test without waiting out the claim lease.
func makeClaimable(ctx context.Context, t *testing.T, tc *testClient, id string) {
	t.Helper()
	_, err := tc.client.getConnection(ctx).Exec(ctx,
		"UPDATE work_items_queue SET available_at = $1 WHERE id = $2",
		time.Now().UTC().Add(-time.Hour), id)
	require.NoError(t, err)
}

func countWorkItems(ctx context.Context, t *testing.T, tc *testClient, id string) int {
	t.Helper()
	var count int
	err := tc.client.getConnection(ctx).QueryRow(ctx,
		"SELECT COUNT(*) FROM work_items_queue WHERE id = $1", id).Scan(&count)
	require.NoError(t, err)
	return count
}

func TestWorkItemsQueue_AcknowledgeWorkItem(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	added, err := testClient.client.WorkItemsQueue.AddWorkItemToQueue(ctx, &AddWorkItemToQueueInput{
		Type:    QueuePendingRunsForWorkspaceType,
		Payload: &QueuePendingRunsForWorkspacePayload{WorkspaceID: "ws-ack"},
	})
	require.NoError(t, err)
	require.Equal(t, 1, countWorkItems(ctx, t, testClient, added.ID))

	// Acknowledging a queued work item deletes it from the queue.
	err = testClient.client.WorkItemsQueue.AcknowledgeWorkItem(ctx, added.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, countWorkItems(ctx, t, testClient, added.ID), "acknowledged item must be removed from the queue")

	// Acknowledging an id that matches no rows (already acknowledged, or never
	// existed) affects zero rows and surfaces an optimistic-lock error.
	err = testClient.client.WorkItemsQueue.AcknowledgeWorkItem(ctx, nonExistentID)
	assert.Equal(t, ErrOptimisticLockError, err)
}

func TestWorkItemsQueue_ClaimWorkItems_MaxClaimCount(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	const maxClaims = 3

	added, err := testClient.client.WorkItemsQueue.AddWorkItemToQueue(ctx, &AddWorkItemToQueueInput{
		Type:    QueuePendingRunsForWorkspaceType,
		Payload: &QueuePendingRunsForWorkspacePayload{WorkspaceID: "ws-max-claims"},
	})
	require.NoError(t, err)

	// The item should be delivered exactly maxClaims times. Don't acknowledge it
	// (simulating a poison item); reset availability between claims so the lease
	// isn't what blocks redelivery.
	for attempt := 1; attempt <= maxClaims; attempt++ {
		items, claimErr := testClient.client.WorkItemsQueue.ClaimWorkItems(ctx, &ClaimWorkItemsInput{
			Type:          QueuePendingRunsForWorkspaceType,
			Limit:         10,
			MaxClaimCount: maxClaims,
		})
		require.NoError(t, claimErr)
		require.Len(t, items, 1, "expected the item to be claimable on attempt %d", attempt)
		assert.Equal(t, added.ID, items[0].ID)
		assert.Equal(t, attempt, items[0].ClaimCount)

		makeClaimable(ctx, t, testClient, added.ID)
	}

	// The next claim must not deliver the exhausted item, and must reap it.
	items, err := testClient.client.WorkItemsQueue.ClaimWorkItems(ctx, &ClaimWorkItemsInput{
		Type:          QueuePendingRunsForWorkspaceType,
		Limit:         10,
		MaxClaimCount: maxClaims,
	})
	require.NoError(t, err)
	assert.Empty(t, items, "exhausted item must not be delivered again")
	assert.Equal(t, 0, countWorkItems(ctx, t, testClient, added.ID), "exhausted item must be reaped")
}

func TestWorkItemsQueue_ClaimWorkItems_UnlimitedClaims(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	added, err := testClient.client.WorkItemsQueue.AddWorkItemToQueue(ctx, &AddWorkItemToQueueInput{
		Type:    QueuePendingRunsForWorkspaceType,
		Payload: &QueuePendingRunsForWorkspacePayload{WorkspaceID: "ws-unlimited"},
	})
	require.NoError(t, err)

	// With MaxClaimCount == 0 the item is never dropped, no matter how many times it
	// is claimed without being acknowledged.
	for attempt := 1; attempt <= 5; attempt++ {
		items, claimErr := testClient.client.WorkItemsQueue.ClaimWorkItems(ctx, &ClaimWorkItemsInput{
			Type:  QueuePendingRunsForWorkspaceType,
			Limit: 10,
			// MaxClaimCount omitted (0) -> unlimited.
		})
		require.NoError(t, claimErr)
		require.Len(t, items, 1, "item must remain claimable on attempt %d when unlimited", attempt)
		assert.Equal(t, attempt, items[0].ClaimCount)

		makeClaimable(ctx, t, testClient, added.ID)
	}

	assert.Equal(t, 1, countWorkItems(ctx, t, testClient, added.ID), "unlimited item must never be reaped")
}
