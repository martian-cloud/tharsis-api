package objectstoregc

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	pkgobjectstore "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/objectstore"
)

func TestTrackedStoreUploadObject(t *testing.T) {
	const key = "some/object/key"

	tests := []struct {
		name        string
		setupMocks  func(*db.MockObjectStoreRefs, *pkgobjectstore.MockObjectStore)
		expectError bool
	}{
		{
			name: "creates ref then uploads",
			setupMocks: func(refs *db.MockObjectStoreRefs, store *pkgobjectstore.MockObjectStore) {
				refs.On("CreateRef", mock.Anything, mock.MatchedBy(func(input *db.CreateObjectStoreRefInput) bool {
					return input.ObjectKey == key &&
						input.AvailableAt != nil &&
						input.AvailableAt.After(time.Now())
				})).Return(nil)
				store.On("UploadObject", mock.Anything, key, mock.Anything).Return(nil)
			},
		},
		{
			name: "CreateRef error aborts upload",
			setupMocks: func(refs *db.MockObjectStoreRefs, _ *pkgobjectstore.MockObjectStore) {
				refs.On("CreateRef", mock.Anything, mock.Anything).Return(errors.New("db error"))
			},
			expectError: true,
		},
		{
			name: "underlying upload error is returned",
			setupMocks: func(refs *db.MockObjectStoreRefs, store *pkgobjectstore.MockObjectStore) {
				refs.On("CreateRef", mock.Anything, mock.Anything).Return(nil)
				store.On("UploadObject", mock.Anything, key, mock.Anything).Return(errors.New("upload error"))
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockRefs := db.NewMockObjectStoreRefs(t)
			mockStore := pkgobjectstore.NewMockObjectStore(t)
			tc.setupMocks(mockRefs, mockStore)

			tracked := New(mockStore, mockRefs)
			err := tracked.UploadObject(t.Context(), key, bytes.NewReader([]byte("content")))

			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestTrackedStoreAvailableAtGracePeriod(t *testing.T) {
	// The pending ref's AvailableAt must be pendingRefGracePeriod from now so the janitor
	// doesn't reclaim it before the FK is linked.
	mockRefs := db.NewMockObjectStoreRefs(t)
	mockStore := pkgobjectstore.NewMockObjectStore(t)

	before := time.Now()
	mockRefs.On("CreateRef", mock.Anything, mock.MatchedBy(func(input *db.CreateObjectStoreRefInput) bool {
		expected := before.Add(pendingRefGracePeriod)
		return input.AvailableAt != nil &&
			!input.AvailableAt.Before(expected.Add(-time.Second)) &&
			!input.AvailableAt.After(expected.Add(time.Second))
	})).Return(nil)
	mockStore.On("UploadObject", mock.Anything, "k", mock.Anything).Return(nil)

	tracked := New(mockStore, mockRefs)
	require.NoError(t, tracked.UploadObject(t.Context(), "k", bytes.NewReader(nil)))
}

func TestTrackedStoreDelegatesNonUpload(t *testing.T) {
	// Non-upload methods must be forwarded to the underlying store without touching refs.
	mockRefs := db.NewMockObjectStoreRefs(t)
	mockStore := pkgobjectstore.NewMockObjectStore(t)

	mockStore.On("DeleteObjects", mock.Anything, []string{"k"}).Return(nil)

	tracked := New(mockStore, mockRefs)
	err := tracked.DeleteObjects(t.Context(), []string{"k"})
	require.NoError(t, err)
	assert.True(t, mockStore.AssertCalled(t, "DeleteObjects", mock.Anything, []string{"k"}))
}
