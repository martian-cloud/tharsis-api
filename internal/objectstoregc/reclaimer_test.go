package objectstoregc

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	pkgobjectstore "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/objectstore"
)

func TestReclaim(t *testing.T) {
	ref1 := db.ObjectStoreRef{ID: "ref-1", ObjectKey: "key/one", ClaimCount: 1}
	ref2 := db.ObjectStoreRef{ID: "ref-2", ObjectKey: "key/two", ClaimCount: 2}
	deadLetter := db.ObjectStoreRef{ID: "ref-dead", ObjectKey: "key/dead", ClaimCount: maxClaimCount + 1}

	tests := []struct {
		name        string
		setupMocks  func(*db.MockObjectStoreRefs, *pkgobjectstore.MockObjectStore)
		expectError bool
	}{
		{
			name: "no orphaned refs is a no-op",
			setupMocks: func(refs *db.MockObjectStoreRefs, _ *pkgobjectstore.MockObjectStore) {
				refs.On("ClaimOrphanedRefs", mock.Anything, uint(10)).Return(nil, nil)
			},
		},
		{
			name: "claims refs, deletes objects, then batch-deletes refs",
			setupMocks: func(refs *db.MockObjectStoreRefs, store *pkgobjectstore.MockObjectStore) {
				refs.On("ClaimOrphanedRefs", mock.Anything, uint(10)).Return([]db.ObjectStoreRef{ref1, ref2}, nil)
				store.On("DeleteObjects", mock.Anything, []string{ref1.ObjectKey, ref2.ObjectKey}).Return(nil)
				refs.On("DeleteRefs", mock.Anything, []string{ref1.ID, ref2.ID}).Return(nil)
			},
		},
		{
			name: "ClaimOrphanedRefs error is returned",
			setupMocks: func(refs *db.MockObjectStoreRefs, _ *pkgobjectstore.MockObjectStore) {
				refs.On("ClaimOrphanedRefs", mock.Anything, uint(10)).Return(nil, errors.New("db down"))
			},
			expectError: true,
		},
		{
			name: "DeleteObjects failure leaves refs for retry and returns nil",
			setupMocks: func(refs *db.MockObjectStoreRefs, store *pkgobjectstore.MockObjectStore) {
				refs.On("ClaimOrphanedRefs", mock.Anything, uint(10)).Return([]db.ObjectStoreRef{ref1, ref2}, nil)
				store.On("DeleteObjects", mock.Anything, []string{ref1.ObjectKey, ref2.ObjectKey}).Return(errors.New("s3 error"))
				// DeleteRefs must NOT be called -- the batch is kept for retry
			},
		},
		{
			name: "DeleteRefs error is logged but returns nil",
			setupMocks: func(refs *db.MockObjectStoreRefs, store *pkgobjectstore.MockObjectStore) {
				refs.On("ClaimOrphanedRefs", mock.Anything, uint(10)).Return([]db.ObjectStoreRef{ref1, ref2}, nil)
				store.On("DeleteObjects", mock.Anything, []string{ref1.ObjectKey, ref2.ObjectKey}).Return(nil)
				refs.On("DeleteRefs", mock.Anything, []string{ref1.ID, ref2.ID}).Return(errors.New("db error"))
			},
		},
		{
			name: "dead-lettered ref is discarded without batch delete",
			setupMocks: func(refs *db.MockObjectStoreRefs, store *pkgobjectstore.MockObjectStore) {
				refs.On("ClaimOrphanedRefs", mock.Anything, uint(10)).Return([]db.ObjectStoreRef{deadLetter}, nil)
				// best-effort delete for the dead-lettered key
				store.On("DeleteObjects", mock.Anything, []string{deadLetter.ObjectKey}).Return(nil)
				refs.On("DeleteRefs", mock.Anything, []string{deadLetter.ID}).Return(nil)
				// batch delete path must NOT be called (keys slice is empty after filtering)
			},
		},
		{
			name: "dead-lettered ref best-effort delete failure is logged and discard continues",
			setupMocks: func(refs *db.MockObjectStoreRefs, store *pkgobjectstore.MockObjectStore) {
				refs.On("ClaimOrphanedRefs", mock.Anything, uint(10)).Return([]db.ObjectStoreRef{deadLetter}, nil)
				store.On("DeleteObjects", mock.Anything, []string{deadLetter.ObjectKey}).Return(errors.New("s3 error"))
				refs.On("DeleteRefs", mock.Anything, []string{deadLetter.ID}).Return(nil)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockRefs := db.NewMockObjectStoreRefs(t)
			mockStore := pkgobjectstore.NewMockObjectStore(t)
			tc.setupMocks(mockRefs, mockStore)

			logr, _ := logger.NewForTest()
			reclaimer := NewReclaimer(mockRefs, mockStore, logr)
			err := reclaimer.Reclaim(t.Context(), 10)

			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestReclaimContextCancelled(t *testing.T) {
	// If the context is cancelled mid-iteration, Reclaim must return ctx.Err() immediately.
	ctx, cancel := context.WithCancel(t.Context())

	mockRefs := db.NewMockObjectStoreRefs(t)
	mockStore := pkgobjectstore.NewMockObjectStore(t)

	ref := db.ObjectStoreRef{ID: "r1", ObjectKey: "k1", ClaimCount: 1}
	mockRefs.On("ClaimOrphanedRefs", mock.Anything, uint(5)).Return([]db.ObjectStoreRef{ref}, nil)

	// Cancel before Reclaim gets to iterate
	cancel()

	logr, _ := logger.NewForTest()
	reclaimer := NewReclaimer(mockRefs, mockStore, logr)
	err := reclaimer.Reclaim(ctx, 5)

	require.ErrorIs(t, err, context.Canceled)
}
