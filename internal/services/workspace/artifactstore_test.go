package workspace

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

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

			cv := models.ConfigurationVersion{Metadata: models.ResourceMetadata{ID: "1"}, WorkspaceID: "ws-1"}

			key := fmt.Sprintf("workspaces/%s/configuration_versions/%s.tar.gz", cv.WorkspaceID, cv.Metadata.ID)
			mockObjectStore.On("DownloadObject", mock.Anything, key, writer, mock.Anything).Return(mockResult)

			err := NewArtifactStore(&mockObjectStore).DownloadConfigurationVersion(ctx, &cv, writer)
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

			sv := models.StateVersion{Metadata: models.ResourceMetadata{ID: "1"}, WorkspaceID: "ws-1"}

			key := fmt.Sprintf("workspaces/%s/state_versions/%s.json", sv.WorkspaceID, sv.Metadata.ID)
			mockObjectStore.On("DownloadObject", mock.Anything, key, writer, mock.Anything).Return(mockResult)

			err := NewArtifactStore(&mockObjectStore).DownloadStateVersion(ctx, &sv, writer)
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

			sv := models.StateVersion{Metadata: models.ResourceMetadata{ID: "1"}, WorkspaceID: "ws-1"}

			key := fmt.Sprintf("workspaces/%s/state_versions/%s.json", sv.WorkspaceID, sv.Metadata.ID)
			mockObjectStore.On("GetObjectStream", mock.Anything, key, mock.Anything).Return(io.NopCloser(strings.NewReader("test payload")), test.retErr)

			resp, err := NewArtifactStore(&mockObjectStore).GetStateVersion(ctx, &sv)
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

			run := models.Run{Metadata: models.ResourceMetadata{ID: "run-1"}, WorkspaceID: "ws-1", PlanID: "plan-1"}

			key := fmt.Sprintf("workspaces/%s/runs/%s/plan/%s", run.WorkspaceID, run.Metadata.ID, run.PlanID)
			mockObjectStore.On("DownloadObject", mock.Anything, key, writer, mock.Anything).Return(mockResult)

			err := NewArtifactStore(&mockObjectStore).DownloadPlanCache(ctx, &run, writer)
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
			buf := bytes.NewBufferString("test data")
			cv := models.ConfigurationVersion{Metadata: models.ResourceMetadata{ID: "1"}, WorkspaceID: "ws-1"}

			key := fmt.Sprintf("workspaces/%s/configuration_versions/%s.tar.gz", cv.WorkspaceID, cv.Metadata.ID)
			mockObjectStore.On("UploadObject", mock.Anything, key, buf).Return(test.retErr)

			err := NewArtifactStore(&mockObjectStore).UploadConfigurationVersion(ctx, &cv, buf)
			if err != nil {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err), "Unexpected error occurred")
				return
			}

			mockObjectStore.AssertExpectations(t)
		})
	}
}

func TestUploadStateVersion(t *testing.T) {
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
			buf := bytes.NewBufferString("test data")
			sv := models.StateVersion{Metadata: models.ResourceMetadata{ID: "1"}, WorkspaceID: "ws-1"}

			key := fmt.Sprintf("workspaces/%s/state_versions/%s.json", sv.WorkspaceID, sv.Metadata.ID)
			mockObjectStore.On("UploadObject", mock.Anything, key, buf).Return(test.retErr)

			err := NewArtifactStore(&mockObjectStore).UploadStateVersion(ctx, &sv, buf)
			if err != nil {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err), "Unexpected error occurred")
				return
			}

			mockObjectStore.AssertExpectations(t)
		})
	}
}

func TestUploadPlanCache(t *testing.T) {
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
			buf := bytes.NewBufferString("test data")
			run := models.Run{Metadata: models.ResourceMetadata{ID: "1"}, WorkspaceID: "ws-1", PlanID: "plan-1"}

			key := fmt.Sprintf("workspaces/%s/runs/%s/plan/%s", run.WorkspaceID, run.Metadata.ID, run.PlanID)
			mockObjectStore.On("UploadObject", mock.Anything, key, buf).Return(test.retErr)

			err := NewArtifactStore(&mockObjectStore).UploadPlanCache(ctx, &run, buf)
			if err != nil {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err), "Unexpected error occurred")
				return
			}

			mockObjectStore.AssertExpectations(t)
		})
	}
}
