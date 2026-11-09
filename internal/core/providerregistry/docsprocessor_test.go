package providerregistry

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/slug"
)

// packDocs writes the given relative-path -> contents files under a temp docs/ tree, packs it
// into a tar.gz using the same slug format the CLI uploads, and returns the package file path.
func packDocs(t *testing.T, files map[string]string) string {
	t.Helper()

	docsDir := t.TempDir()
	for relPath, contents := range files {
		fullPath := filepath.Join(docsDir, relPath)
		require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0700))
		require.NoError(t, os.WriteFile(fullPath, []byte(contents), 0600))
	}

	packaged, err := slug.New(docsDir)
	require.NoError(t, err)
	t.Cleanup(func() { os.Remove(packaged.SlugPath) })

	return packaged.SlugPath
}

// updatedToStatus matches the UpdateProviderVersion call that records the given terminal doc status.
// A non-nil wantMessage also asserts the stored error message equals it; a nil wantMessage only
// asserts a message is present for the failed status and absent otherwise.
func updatedToStatus(status models.TerraformProviderVersionDocsProcessingStatus, wantMessage *string) interface{} {
	return mock.MatchedBy(func(pv *models.TerraformProviderVersion) bool {
		if pv.DocsProcessingStatus != status {
			return false
		}
		if wantMessage != nil {
			return pv.DocsProcessingErrorMessage != nil && *pv.DocsProcessingErrorMessage == *wantMessage
		}
		hasErr := status == models.TerraformProviderVersionDocsProcessingStatusFailed
		return (pv.DocsProcessingErrorMessage != nil) == hasErr
	})
}

func TestProcessProviderVersionDocs(t *testing.T) {
	const (
		providerID = "provider-1"
		versionID  = "version-1"
	)

	// Shared across subtests; the loop reassigns them per subtest so recordResult's in-place status
	// mutation on one case can't bleed into the next.
	var providerVersion *models.TerraformProviderVersion
	var provider *models.TerraformProvider

	tests := []struct {
		name          string
		finalAttempt  bool
		setupMocks    func(*db.MockTerraformProviderVersions, *db.MockTerraformProviders, *db.MockTerraformProviderDocItems, *MockRegistryStore, *db.MockTransactions)
		expectErrCode errors.CodeType
	}{
		{
			name: "acks without processing when the provider version no longer exists",
			setupMocks: func(versions *db.MockTerraformProviderVersions, _ *db.MockTerraformProviders, _ *db.MockTerraformProviderDocItems, _ *MockRegistryStore, _ *db.MockTransactions) {
				versions.On("GetProviderVersionByID", mock.Anything, versionID).Return(nil, nil)
			},
		},
		{
			name: "returns a transient error when the docs package download fails",
			setupMocks: func(versions *db.MockTerraformProviderVersions, providers *db.MockTerraformProviders, _ *db.MockTerraformProviderDocItems, store *MockRegistryStore, _ *db.MockTransactions) {
				versions.On("GetProviderVersionByID", mock.Anything, versionID).Return(providerVersion, nil)
				providers.On("GetProviderByID", mock.Anything, providerID).Return(provider, nil)
				store.On("GetProviderVersionDocsPackage", mock.Anything, providerVersion, provider).
					Return(nil, errors.New("object store unavailable"))
			},
			expectErrCode: errors.EInternal,
		},
		{
			name: "acks when the version is deleted during processing",
			setupMocks: func(versions *db.MockTerraformProviderVersions, providers *db.MockTerraformProviders, _ *db.MockTerraformProviderDocItems, store *MockRegistryStore, _ *db.MockTransactions) {
				versions.On("GetProviderVersionByID", mock.Anything, versionID).Return(providerVersion, nil).Once()
				providers.On("GetProviderByID", mock.Anything, providerID).Return(provider, nil)
				store.On("GetProviderVersionDocsPackage", mock.Anything, providerVersion, provider).
					Return(nil, errors.New("object store unavailable"))
				versions.On("GetProviderVersionByID", mock.Anything, versionID).Return(nil, nil).Once()
			},
		},
		{
			name: "records the failed status when the docs package has no recognized doc files",
			setupMocks: func(versions *db.MockTerraformProviderVersions, providers *db.MockTerraformProviders, _ *db.MockTerraformProviderDocItems, store *MockRegistryStore, transactions *db.MockTransactions) {
				packagePath := packDocs(t, map[string]string{
					filepath.Join("docs", "index.md"): "# wrapped in an extra docs dir",
				})

				versions.On("GetProviderVersionByID", mock.Anything, versionID).Return(providerVersion, nil)
				providers.On("GetProviderByID", mock.Anything, providerID).Return(provider, nil)
				store.On("GetProviderVersionDocsPackage", mock.Anything, providerVersion, provider).
					Return(func(_ context.Context, _ *models.TerraformProviderVersion, _ *models.TerraformProvider) (io.ReadCloser, error) {
						return os.Open(packagePath) // nosemgrep: gosec.G304-1
					})

				transactions.On("BeginTx", mock.Anything).Return(context.Background(), nil)
				transactions.On("CommitTx", mock.Anything).Return(nil)
				transactions.On("RollbackTx", mock.Anything).Return(nil)

				versions.On("UpdateProviderVersion", mock.Anything,
					updatedToStatus(models.TerraformProviderVersionDocsProcessingStatusFailed,
						ptr.String("no documentation files found; expected an index.md at the root or Markdown files under resources/, data-sources/, ephemeral-resources/, functions/, or guides/"))).Return(providerVersion, nil)
			},
		},
		{
			name: "records the completed status when the docs package is processed",
			setupMocks: func(versions *db.MockTerraformProviderVersions, providers *db.MockTerraformProviders, docItems *db.MockTerraformProviderDocItems, store *MockRegistryStore, transactions *db.MockTransactions) {
				packagePath := packDocs(t, map[string]string{
					"index.md":                            "# overview",
					filepath.Join("resources", "a.md"):    "# resource a",
					filepath.Join("data-sources", "b.md"): "# data source b",
				})

				versions.On("GetProviderVersionByID", mock.Anything, versionID).Return(providerVersion, nil)
				providers.On("GetProviderByID", mock.Anything, providerID).Return(provider, nil)
				store.On("GetProviderVersionDocsPackage", mock.Anything, providerVersion, provider).
					Return(func(_ context.Context, _ *models.TerraformProviderVersion, _ *models.TerraformProvider) (io.ReadCloser, error) {
						return os.Open(packagePath) // nosemgrep: gosec.G304-1
					})

				noopRetain := func(_ context.Context, _ string) error { return nil }
				store.On("UploadProviderVersionDocFile", mock.Anything, providerVersion, provider, mock.Anything, mock.Anything, mock.Anything).
					Return(db.RetainObjectRefFunc(noopRetain), "doc-file-key", nil)

				transactions.On("BeginTx", mock.Anything).Return(context.Background(), nil)
				transactions.On("CommitTx", mock.Anything).Return(nil)
				transactions.On("RollbackTx", mock.Anything).Return(nil)
				docItems.On("CreateProviderDocItems", mock.Anything, mock.Anything).Return(nil)
				versions.On("UpdateProviderVersion", mock.Anything,
					updatedToStatus(models.TerraformProviderVersionDocsProcessingStatusCompleted, nil)).Return(providerVersion, nil)
			},
		},
		{
			name: "records the failed status on the first delivery when a doc file exceeds the size cap",
			setupMocks: func(versions *db.MockTerraformProviderVersions, providers *db.MockTerraformProviders, _ *db.MockTerraformProviderDocItems, store *MockRegistryStore, transactions *db.MockTransactions) {
				// ETooLarge is bad content like EInvalid, so it must be recorded as failed rather than retried.
				packagePath := packDocs(t, map[string]string{
					filepath.Join("resources", "big.md"): strings.Repeat("a", maxProviderDocFileSize+1),
				})

				versions.On("GetProviderVersionByID", mock.Anything, versionID).Return(providerVersion, nil)
				providers.On("GetProviderByID", mock.Anything, providerID).Return(provider, nil)
				store.On("GetProviderVersionDocsPackage", mock.Anything, providerVersion, provider).
					Return(func(_ context.Context, _ *models.TerraformProviderVersion, _ *models.TerraformProvider) (io.ReadCloser, error) {
						return os.Open(packagePath) // nosemgrep: gosec.G304-1
					})

				transactions.On("BeginTx", mock.Anything).Return(context.Background(), nil)
				transactions.On("CommitTx", mock.Anything).Return(nil)
				transactions.On("RollbackTx", mock.Anything).Return(nil)

				versions.On("UpdateProviderVersion", mock.Anything,
					updatedToStatus(models.TerraformProviderVersionDocsProcessingStatusFailed,
						ptr.String(fmt.Sprintf("documentation file %q exceeds the maximum size of %d bytes", "big.md", maxProviderDocFileSize)))).
					Return(providerVersion, nil)
			},
		},
		{
			name: "records the failed status without uploading when a doc file name is invalid",
			setupMocks: func(versions *db.MockTerraformProviderVersions, providers *db.MockTerraformProviders, _ *db.MockTerraformProviderDocItems, store *MockRegistryStore, transactions *db.MockTransactions) {
				// The parser accepts the name; the model's Validate rejects the space, before any upload, as terminal.
				packagePath := packDocs(t, map[string]string{
					filepath.Join("resources", "bad name.md"): "# bad name",
				})

				versions.On("GetProviderVersionByID", mock.Anything, versionID).Return(providerVersion, nil)
				providers.On("GetProviderByID", mock.Anything, providerID).Return(provider, nil)
				store.On("GetProviderVersionDocsPackage", mock.Anything, providerVersion, provider).
					Return(func(_ context.Context, _ *models.TerraformProviderVersion, _ *models.TerraformProvider) (io.ReadCloser, error) {
						return os.Open(packagePath) // nosemgrep: gosec.G304-1
					})

				transactions.On("BeginTx", mock.Anything).Return(context.Background(), nil)
				transactions.On("CommitTx", mock.Anything).Return(nil)
				transactions.On("RollbackTx", mock.Anything).Return(nil)

				versions.On("UpdateProviderVersion", mock.Anything,
					updatedToStatus(models.TerraformProviderVersionDocsProcessingStatusFailed, nil)).
					Return(providerVersion, nil)
			},
		},
		{
			name: "returns a transient error without recording failed when a doc file upload fails",
			setupMocks: func(versions *db.MockTerraformProviderVersions, providers *db.MockTerraformProviders, _ *db.MockTerraformProviderDocItems, store *MockRegistryStore, _ *db.MockTransactions) {
				packagePath := packDocs(t, map[string]string{"index.md": "# overview"})

				// Initial fetch, plus the isSettled re-fetch after the upload failure (still pending).
				versions.On("GetProviderVersionByID", mock.Anything, versionID).Return(providerVersion, nil)
				providers.On("GetProviderByID", mock.Anything, providerID).Return(provider, nil)
				store.On("GetProviderVersionDocsPackage", mock.Anything, providerVersion, provider).
					Return(func(_ context.Context, _ *models.TerraformProviderVersion, _ *models.TerraformProvider) (io.ReadCloser, error) {
						return os.Open(packagePath) // nosemgrep: gosec.G304-1
					})
				store.On("UploadProviderVersionDocFile", mock.Anything, providerVersion, provider, mock.Anything, mock.Anything, mock.Anything).
					Return(db.RetainObjectRefFunc(nil), "", errors.New("object store unavailable"))
			},
			expectErrCode: errors.EInternal,
		},
		{
			name: "acks when the version is deleted before the result is recorded",
			setupMocks: func(versions *db.MockTerraformProviderVersions, providers *db.MockTerraformProviders, _ *db.MockTerraformProviderDocItems, store *MockRegistryStore, transactions *db.MockTransactions) {
				packagePath := packDocs(t, map[string]string{"index.md": "# overview"})

				versions.On("GetProviderVersionByID", mock.Anything, versionID).Return(providerVersion, nil).Once()
				providers.On("GetProviderByID", mock.Anything, providerID).Return(provider, nil)
				store.On("GetProviderVersionDocsPackage", mock.Anything, providerVersion, provider).
					Return(func(_ context.Context, _ *models.TerraformProviderVersion, _ *models.TerraformProvider) (io.ReadCloser, error) {
						return os.Open(packagePath) // nosemgrep: gosec.G304-1
					})
				noopRetain := func(_ context.Context, _ string) error { return nil }
				store.On("UploadProviderVersionDocFile", mock.Anything, providerVersion, provider, mock.Anything, mock.Anything, mock.Anything).
					Return(db.RetainObjectRefFunc(noopRetain), "doc-file-key", nil)

				transactions.On("BeginTx", mock.Anything).Return(context.Background(), nil)
				transactions.On("RollbackTx", mock.Anything).Return(nil)
				versions.On("GetProviderVersionByID", mock.Anything, versionID).Return(nil, nil).Once()
			},
		},
		{
			name: "no-ops when the version is already settled before recording the result",
			setupMocks: func(versions *db.MockTerraformProviderVersions, providers *db.MockTerraformProviders, _ *db.MockTerraformProviderDocItems, store *MockRegistryStore, transactions *db.MockTransactions) {
				packagePath := packDocs(t, map[string]string{"index.md": "# overview"})

				pending := &models.TerraformProviderVersion{
					Metadata:             models.ResourceMetadata{ID: versionID},
					ProviderID:           providerID,
					DocsProcessingStatus: models.TerraformProviderVersionDocsProcessingStatusPending,
				}
				completed := &models.TerraformProviderVersion{
					Metadata:             models.ResourceMetadata{ID: versionID},
					ProviderID:           providerID,
					DocsProcessingStatus: models.TerraformProviderVersionDocsProcessingStatusCompleted,
				}
				versions.On("GetProviderVersionByID", mock.Anything, versionID).Return(pending, nil).Once()
				providers.On("GetProviderByID", mock.Anything, providerID).Return(provider, nil)
				store.On("GetProviderVersionDocsPackage", mock.Anything, pending, provider).
					Return(func(_ context.Context, _ *models.TerraformProviderVersion, _ *models.TerraformProvider) (io.ReadCloser, error) {
						return os.Open(packagePath) // nosemgrep: gosec.G304-1
					})
				noopRetain := func(_ context.Context, _ string) error { return nil }
				store.On("UploadProviderVersionDocFile", mock.Anything, pending, provider, mock.Anything, mock.Anything, mock.Anything).
					Return(db.RetainObjectRefFunc(noopRetain), "doc-file-key", nil)

				transactions.On("BeginTx", mock.Anything).Return(context.Background(), nil)
				transactions.On("RollbackTx", mock.Anything).Return(nil)
				versions.On("GetProviderVersionByID", mock.Anything, versionID).Return(completed, nil).Once()
			},
		},
		{
			name:         "records failed on the final attempt when the download keeps failing",
			finalAttempt: true,
			setupMocks: func(versions *db.MockTerraformProviderVersions, providers *db.MockTerraformProviders, _ *db.MockTerraformProviderDocItems, store *MockRegistryStore, transactions *db.MockTransactions) {
				versions.On("GetProviderVersionByID", mock.Anything, versionID).Return(providerVersion, nil)
				providers.On("GetProviderByID", mock.Anything, providerID).Return(provider, nil)
				store.On("GetProviderVersionDocsPackage", mock.Anything, providerVersion, provider).
					Return(nil, errors.New("object store unavailable"))

				transactions.On("BeginTx", mock.Anything).Return(context.Background(), nil)
				transactions.On("CommitTx", mock.Anything).Return(nil)
				transactions.On("RollbackTx", mock.Anything).Return(nil)
				versions.On("UpdateProviderVersion", mock.Anything,
					updatedToStatus(models.TerraformProviderVersionDocsProcessingStatusFailed,
						ptr.String(errors.InternalErrorMessage))).Return(providerVersion, nil)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Fresh fixtures per subtest so recordResult's in-place status mutation can't bleed across cases.
			providerVersion = &models.TerraformProviderVersion{
				Metadata:             models.ResourceMetadata{ID: versionID},
				ProviderID:           providerID,
				DocsProcessingStatus: models.TerraformProviderVersionDocsProcessingStatusPending,
			}
			provider = &models.TerraformProvider{Metadata: models.ResourceMetadata{ID: providerID}}

			mockVersions := db.NewMockTerraformProviderVersions(t)
			mockProviders := db.NewMockTerraformProviders(t)
			mockDocItems := db.NewMockTerraformProviderDocItems(t)
			mockStore := NewMockRegistryStore(t)
			mockTransactions := db.NewMockTransactions(t)
			test.setupMocks(mockVersions, mockProviders, mockDocItems, mockStore, mockTransactions)

			testLogger, _ := logger.NewForTest()
			dbClient := &db.Client{
				TerraformProviderVersions: mockVersions,
				TerraformProviders:        mockProviders,
				TerraformProviderDocItems: mockDocItems,
				Transactions:              mockTransactions,
			}

			processor := NewDocsProcessor(dbClient, mockStore, testLogger)
			err := processor.ProcessProviderVersionDocs(context.Background(), versionID, test.finalAttempt)

			if test.expectErrCode != "" {
				require.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}
			require.NoError(t, err)
		})
	}
}
