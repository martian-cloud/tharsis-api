//go:build integration

package db

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func createTestLogStreamForChunks(ctx context.Context, t *testing.T, testClient *testClient) *models.LogStream {
	logStream, err := testClient.client.LogStreams.CreateLogStream(ctx, &models.LogStream{})
	require.Nil(t, err)
	return logStream
}

func TestLogStreams_ClaimLogStreamsForCompaction(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Completed + not compacted: the compaction candidate.
	candidate, err := testClient.client.LogStreams.CreateLogStream(ctx, &models.LogStream{})
	require.Nil(t, err)
	candidate.Completed = true
	candidate, err = testClient.client.LogStreams.UpdateLogStream(ctx, candidate)
	require.Nil(t, err)
	assert.False(t, candidate.Compacted) // compacted round-trips as false

	// Completed + compacted: already done, must not be claimed.
	done, err := testClient.client.LogStreams.CreateLogStream(ctx, &models.LogStream{})
	require.Nil(t, err)
	done.Completed = true
	done.Compacted = true
	_, err = testClient.client.LogStreams.UpdateLogStream(ctx, done)
	require.Nil(t, err)

	// Not completed: must not be claimed.
	_, err = testClient.client.LogStreams.CreateLogStream(ctx, &models.LogStream{})
	require.Nil(t, err)

	// Only the candidate is eligible; it comes back stamped with a claim time and a bumped version.
	claimableBefore := time.Now().Add(-time.Minute)
	claimed, err := testClient.client.LogStreams.ClaimLogStreamsForCompaction(ctx, 10, claimableBefore)
	require.Nil(t, err)
	require.Len(t, claimed, 1)
	assert.Equal(t, candidate.Metadata.ID, claimed[0].Metadata.ID)
	require.NotNil(t, claimed[0].CompactionStartedAt)
	assert.Greater(t, claimed[0].Metadata.Version, candidate.Metadata.Version)

	// Re-claiming with the same cutoff skips the freshly-claimed stream (its claim is newer than the
	// cutoff), so nothing is returned. This is the SKIP-stale-claims behavior the scheduler relies on.
	again, err := testClient.client.LogStreams.ClaimLogStreamsForCompaction(ctx, 10, claimableBefore)
	require.Nil(t, err)
	assert.Empty(t, again)

	// With a cutoff after the claim time, the now-stale claim becomes reclaimable again.
	reclaimed, err := testClient.client.LogStreams.ClaimLogStreamsForCompaction(ctx, 10, time.Now().Add(time.Minute))
	require.Nil(t, err)
	require.Len(t, reclaimed, 1)
	assert.Equal(t, candidate.Metadata.ID, reclaimed[0].Metadata.ID)
}

func TestLogStreamChunks_CreateAndGet(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	logStream := createTestLogStreamForChunks(ctx, t, testClient)

	created, err := testClient.client.LogStreamChunks.CreateLogStreamChunk(ctx, &models.LogStreamChunk{
		LogStreamID: logStream.Metadata.ID,
		ChunkIndex:  0,
		StartOffset: 0,
		Size:        4,
		ObjectKey:   "logstreams/x/0.txt",
	})
	require.Nil(t, err)
	assert.NotEmpty(t, created.Metadata.ID)
	assert.Equal(t, 1, created.Metadata.Version)
	assert.Equal(t, "logstreams/x/0.txt", created.ObjectKey)

	chunks, err := testClient.client.LogStreamChunks.GetOverlappingChunks(ctx, logStream.Metadata.ID, 0, 4)
	require.Nil(t, err)
	require.Len(t, chunks, 1)
	assert.Equal(t, created.Metadata.ID, chunks[0].Metadata.ID)
}

func TestLogStreamChunks_CreateNonExistentStream(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	_, err := testClient.client.LogStreamChunks.CreateLogStreamChunk(ctx, &models.LogStreamChunk{
		LogStreamID: "00000000-0000-0000-0000-000000000000",
		ChunkIndex:  0,
		ObjectKey:   "k",
	})
	require.NotNil(t, err)
	assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
}

func TestLogStreamChunks_GetActiveChunk(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	logStream := createTestLogStreamForChunks(ctx, t, testClient)

	// No chunks yet.
	active, err := testClient.client.LogStreamChunks.GetActiveChunk(ctx, logStream.Metadata.ID)
	require.Nil(t, err)
	assert.Nil(t, active)

	for i := 0; i < 3; i++ {
		_, err = testClient.client.LogStreamChunks.CreateLogStreamChunk(ctx, &models.LogStreamChunk{
			LogStreamID: logStream.Metadata.ID,
			ChunkIndex:  i,
			StartOffset: i * 10,
			Size:        10,
			ObjectKey:   "k",
		})
		require.Nil(t, err)
	}

	active, err = testClient.client.LogStreamChunks.GetActiveChunk(ctx, logStream.Metadata.ID)
	require.Nil(t, err)
	require.NotNil(t, active)
	assert.Equal(t, 2, active.ChunkIndex)
	assert.Equal(t, 20, active.StartOffset)
}

func TestLogStreamChunks_GetOverlappingChunks(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	logStream := createTestLogStreamForChunks(ctx, t, testClient)

	specs := []struct {
		index       int
		startOffset int
		size        int
	}{
		{0, 0, 10},
		{1, 10, 10},
		{2, 20, 5},
	}
	for _, s := range specs {
		_, err := testClient.client.LogStreamChunks.CreateLogStreamChunk(ctx, &models.LogStreamChunk{
			LogStreamID: logStream.Metadata.ID,
			ChunkIndex:  s.index,
			StartOffset: s.startOffset,
			Size:        s.size,
			ObjectKey:   "k",
		})
		require.Nil(t, err)
	}

	tests := []struct {
		name            string
		start           int
		end             int
		expectedOffsets []int
	}{
		{"spans first two chunks", 5, 15, []int{0, 10}},
		{"single chunk at boundary", 10, 20, []int{10}},
		{"tail chunk", 20, 100, []int{20}},
		{"empty range", 10, 10, nil},
		{"covers all", 0, 100, []int{0, 10, 20}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			chunks, err := testClient.client.LogStreamChunks.GetOverlappingChunks(ctx, logStream.Metadata.ID, test.start, test.end)
			require.Nil(t, err)

			offsets := []int{}
			for _, c := range chunks {
				offsets = append(offsets, c.StartOffset)
			}
			if test.expectedOffsets == nil {
				assert.Empty(t, offsets)
				return
			}
			assert.Equal(t, test.expectedOffsets, offsets)
		})
	}
}

func TestLogStreamChunks_UpdateOptimisticLock(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	logStream := createTestLogStreamForChunks(ctx, t, testClient)

	created, err := testClient.client.LogStreamChunks.CreateLogStreamChunk(ctx, &models.LogStreamChunk{
		LogStreamID: logStream.Metadata.ID,
		ChunkIndex:  0,
		Size:        4,
		ObjectKey:   "k",
	})
	require.Nil(t, err)

	created.Size = 8
	created.Sealed = true
	updated, err := testClient.client.LogStreamChunks.UpdateLogStreamChunk(ctx, created)
	require.Nil(t, err)
	assert.Equal(t, 8, updated.Size)
	assert.True(t, updated.Sealed)
	assert.Equal(t, created.Metadata.Version+1, updated.Metadata.Version)

	// Updating with the stale version should fail the optimistic lock.
	created.Size = 12
	_, err = testClient.client.LogStreamChunks.UpdateLogStreamChunk(ctx, created)
	require.NotNil(t, err)
	assert.Equal(t, ErrOptimisticLockError, err)
}
