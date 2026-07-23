package workspace

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/objectstore"
)

var (
	errNotFound = errors.New(
		"not Found",
		errors.WithErrorCode(errors.ENotFound))
	errInternal = errors.New(
		"internal Error",
		errors.WithErrorCode(errors.EInternal))
)

type fakeWriterAt struct {
	w io.Writer
}

func (fw fakeWriterAt) WriteAt(p []byte, _ int64) (n int, err error) {
	// Ignore offset
	return fw.w.Write(p)
}

func TestDownloadConfigurationVersion(t *testing.T) {
	// Test cases
	tests := []struct {
		name          string
		expectContent string
		retErr        error
		expectErrCode errors.CodeType
	}{
		{
			name:          "success",
			expectContent: "test payload",
		},
		{
			name:          "not found",
			retErr:        errNotFound,
			expectErrCode: errors.ENotFound,
		},
		{
			name:          "internal error",
			retErr:        errInternal,
			expectErrCode: errors.EInternal,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockResult := func(_ context.Context, _ string, w io.WriterAt, _ *objectstore.DownloadOptions) error {
				if test.retErr != nil {
					return test.retErr
				}

				_, _ = w.WriteAt([]byte("test payload"), 0)

				return nil
			}

			mockObjectStore := objectstore.MockObjectStore{}

			var buf bytes.Buffer
			writer := fakeWriterAt{w: &buf}

			key := "workspaces/ws-1/configuration_versions/cv-key.tar.gz"
			cv := models.ConfigurationVersion{Metadata: models.ResourceMetadata{ID: "1"}, WorkspaceID: "ws-1", ObjectStoreKey: key}

			mockObjectStore.On("DownloadObject", mock.Anything, key, writer, mock.Anything).Return(mockResult)

			err := NewArtifactStore(&mockObjectStore, nil).DownloadConfigurationVersion(ctx, &cv, writer)
			if err != nil {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err), "Unexpected error occurred")
				return
			}

			mockObjectStore.AssertExpectations(t)

			assert.Equal(t, test.expectContent, buf.String())
		})
	}
}

func TestDownloadStateVersion(t *testing.T) {
	// Test cases
	tests := []struct {
		name          string
		expectContent string
		retErr        error
		expectErrCode errors.CodeType
	}{
		{
			name:          "success",
			expectContent: "test payload",
		},
		{
			name:          "not found",
			retErr:        errNotFound,
			expectErrCode: errors.ENotFound,
		},
		{
			name:          "internal error",
			retErr:        errInternal,
			expectErrCode: errors.EInternal,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockResult := func(_ context.Context, _ string, w io.WriterAt, _ *objectstore.DownloadOptions) error {
				if test.retErr != nil {
					return test.retErr
				}

				_, _ = w.WriteAt([]byte("test payload"), 0)

				return nil
			}

			mockObjectStore := objectstore.MockObjectStore{}

			var buf bytes.Buffer
			writer := fakeWriterAt{w: &buf}

			key := "workspaces/ws-1/state_versions/sv-key.json"
			sv := models.StateVersion{Metadata: models.ResourceMetadata{ID: "1"}, WorkspaceID: "ws-1", ObjectStoreKey: key}

			mockObjectStore.On("DownloadObject", mock.Anything, key, writer, mock.Anything).Return(mockResult)

			err := NewArtifactStore(&mockObjectStore, nil).DownloadStateVersion(ctx, &sv, writer)
			if err != nil {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err), "Unexpected error occurred")
				return
			}

			mockObjectStore.AssertExpectations(t)

			assert.Equal(t, test.expectContent, buf.String())
		})
	}
}

func TestGetStateVersion(t *testing.T) {
	// Test cases
	tests := []struct {
		name          string
		expectContent string
		retErr        error
		expectErrCode errors.CodeType
	}{
		{
			name:          "success",
			expectContent: "test payload",
		},
		{
			name:          "not found",
			retErr:        errNotFound,
			expectErrCode: errors.ENotFound,
		},
		{
			name:          "internal error",
			retErr:        errInternal,
			expectErrCode: errors.EInternal,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockObjectStore := objectstore.MockObjectStore{}

			key := "workspaces/ws-1/state_versions/sv-key.json"
			sv := models.StateVersion{Metadata: models.ResourceMetadata{ID: "1"}, WorkspaceID: "ws-1", ObjectStoreKey: key}

			mockObjectStore.On("GetObjectStream", mock.Anything, key, mock.Anything).Return(&objectstore.GetObjectStreamOutput{Body: io.NopCloser(strings.NewReader("test payload"))}, test.retErr)

			resp, err := NewArtifactStore(&mockObjectStore, nil).GetStateVersion(ctx, &sv)
			if err != nil {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err), "Unexpected error occurred")
				return
			}
			defer resp.Close()

			mockObjectStore.AssertExpectations(t)

			buf, err := io.ReadAll(resp)
			assert.Nil(t, err, fmt.Sprintf("Unexpected error occurred: %v", err))
			assert.Equal(t, test.expectContent, string(buf))
		})
	}
}

func TestDownloadPlanCache(t *testing.T) {
	// Test cases
	tests := []struct {
		name          string
		expectContent string
		retErr        error
		expectErrCode errors.CodeType
	}{
		{
			name:          "success",
			expectContent: "test payload",
		},
		{
			name:          "not found",
			retErr:        errNotFound,
			expectErrCode: errors.ENotFound,
		},
		{
			name:          "internal error",
			retErr:        errInternal,
			expectErrCode: errors.EInternal,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockResult := func(_ context.Context, _ string, w io.WriterAt, _ *objectstore.DownloadOptions) error {
				if test.retErr != nil {
					return test.retErr
				}

				_, _ = w.WriteAt([]byte("test payload"), 0)

				return nil
			}

			mockObjectStore := objectstore.MockObjectStore{}

			var buf bytes.Buffer
			writer := fakeWriterAt{w: &buf}

			key := fmt.Sprintf("workspaces/%s/runs/%s/plan/%s", "ws-1", "run-1", "plan-1")
			run := models.Run{
				Metadata:    models.ResourceMetadata{ID: "run-1"},
				WorkspaceID: "ws-1",
				Plan:        models.Plan{ID: "plan-1", CacheObjectStoreKey: ptr.String(key)},
			}
			mockObjectStore.On("DownloadObject", mock.Anything, key, writer, mock.Anything).Return(mockResult)

			err := NewArtifactStore(&mockObjectStore, nil).DownloadPlanCache(ctx, &run, writer)
			if err != nil {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err), "Unexpected error occurred")
				return
			}

			mockObjectStore.AssertExpectations(t)

			assert.Equal(t, test.expectContent, buf.String())
		})
	}
}

func TestUploadConfigurationVersion(t *testing.T) {
	tests := []struct {
		name          string
		retErr        error
		expectErrCode errors.CodeType
	}{
		{
			name: "success",
		},
		{
			name:          "internal error",
			retErr:        errInternal,
			expectErrCode: errors.EInternal,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			buf := bytes.NewBufferString("test data")
			cv := models.ConfigurationVersion{Metadata: models.ResourceMetadata{ID: "1"}, WorkspaceID: "ws-1"}

			mockObjectStore := objectstore.MockObjectStore{}
			mockObjectStore.On("UploadObject", mock.Anything, mock.Anything, mock.Anything).Return(test.retErr)

			mockRefs := db.NewMockObjectStoreRefs(t)
			if test.retErr == nil {
				mockRefs.On("LinkRef", mock.Anything, mock.Anything, db.ObjectStoreRefOwnerConfigurationVersion, cv.Metadata.ID).Return(nil)
			}

			retainFn, key, err := NewArtifactStore(&mockObjectStore, mockRefs).UploadConfigurationVersion(ctx, &cv, buf)
			if err != nil {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err), "Unexpected error occurred")
				return
			}

			assert.NotNil(t, retainFn)
			assert.NotEmpty(t, key)
			assert.NoError(t, retainFn(ctx, cv.Metadata.ID))
		})
	}
}

func TestUploadStateVersion(t *testing.T) {
	tests := []struct {
		name          string
		retErr        error
		expectErrCode errors.CodeType
	}{
		{
			name: "success",
		},
		{
			name:          "internal error",
			retErr:        errInternal,
			expectErrCode: errors.EInternal,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			buf := bytes.NewBufferString("test data")
			sv := models.StateVersion{Metadata: models.ResourceMetadata{ID: "1"}, WorkspaceID: "ws-1"}

			mockObjectStore := objectstore.MockObjectStore{}
			mockObjectStore.On("UploadObject", mock.Anything, mock.Anything, mock.Anything).Return(test.retErr)

			mockRefs := db.NewMockObjectStoreRefs(t)
			if test.retErr == nil {
				mockRefs.On("LinkRef", mock.Anything, mock.Anything, db.ObjectStoreRefOwnerStateVersion, sv.Metadata.ID).Return(nil)
			}

			retainFn, key, err := NewArtifactStore(&mockObjectStore, mockRefs).UploadStateVersion(ctx, &sv, buf)
			if err != nil {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err), "Unexpected error occurred")
				return
			}

			assert.NotNil(t, retainFn)
			assert.NotEmpty(t, key)
			assert.NoError(t, retainFn(ctx, sv.Metadata.ID))
		})
	}
}

func TestUploadPlanCache(t *testing.T) {
	tests := []struct {
		name          string
		retErr        error
		expectErrCode errors.CodeType
	}{
		{
			name: "success",
		},
		{
			name:          "internal error",
			retErr:        errInternal,
			expectErrCode: errors.EInternal,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			run := models.Run{
				Metadata:    models.ResourceMetadata{ID: "1"},
				WorkspaceID: "ws-1",
				Plan:        models.Plan{ID: "plan-1"},
			}
			key := fmt.Sprintf("workspaces/%s/runs/%s/plan/%s", run.WorkspaceID, run.Metadata.ID, run.Plan.GetID())

			buf := bytes.NewBufferString("test data")
			mockObjectStore := objectstore.MockObjectStore{}
			mockObjectStore.On("UploadObject", mock.Anything, mock.Anything, mock.Anything).Return(test.retErr)

			mockRefs := db.NewMockObjectStoreRefs(t)
			if test.retErr == nil {
				mockRefs.On("LinkRef", mock.Anything, key, db.ObjectStoreRefOwnerRun, run.Metadata.ID).Return(nil)
			}

			retainFn, retKey, err := NewArtifactStore(&mockObjectStore, mockRefs).UploadPlanCache(ctx, &run, buf)
			if err != nil {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err), "Unexpected error occurred")
				return
			}

			assert.NotNil(t, retainFn)
			assert.Equal(t, key, retKey)
			assert.NoError(t, retainFn(ctx, run.Metadata.ID))
		})
	}
}
