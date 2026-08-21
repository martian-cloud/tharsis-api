package packageregistry

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	corepkg "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/packageregistry"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/limits"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

func TestCreatePackage(t *testing.T) {
	groupID := "group-id"
	packageID := "package-id"
	packageName := "my-package"

	tests := []struct {
		name                  string
		authError             error
		input                 *CreatePackageInput
		limit                 int
		injectPackagesInGroup int32
		expectErrCode         errors.CodeType
	}{
		{
			name:                  "create package under limit",
			input:                 &CreatePackageInput{Name: packageName, GroupID: groupID},
			limit:                 5,
			injectPackagesInGroup: 5,
		},
		{
			name:                  "exceeds packages per group limit",
			input:                 &CreatePackageInput{Name: packageName, GroupID: groupID},
			limit:                 5,
			injectPackagesInGroup: 6,
			expectErrCode:         errors.EInvalid,
		},
		{
			name:          "subject does not have permission",
			input:         &CreatePackageInput{Name: packageName, GroupID: groupID},
			authError:     errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrCode: errors.EForbidden,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test := test

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockCaller.On("RequirePermission", mock.Anything, models.CreatePackagePermission, mock.Anything).Return(test.authError)
			mockCaller.On("GetSubject").Return("mockSubject").Maybe()

			mockGroups := db.NewMockGroups(t)
			mockPackages := db.NewMockPackages(t)
			mockTransactions := db.NewMockTransactions(t)
			mockResourceLimits := db.NewMockResourceLimits(t)

			if test.authError == nil {
				mockGroups.On("GetGroupByID", mock.Anything, groupID).
					Return(&models.Group{Metadata: models.ResourceMetadata{ID: groupID}}, nil)

				mockTransactions.On("BeginTx", mock.Anything).Return(auth.WithCaller(ctx, mockCaller), nil)
				mockTransactions.On("RollbackTx", mock.Anything).Return(nil)

				mockPackages.On("CreatePackage", mock.Anything, mock.Anything).
					Return(&models.Package{Metadata: models.ResourceMetadata{ID: packageID}, Name: packageName, GroupID: groupID}, nil)

				// Called inside the transaction to check the resource limit.
				mockPackages.On("GetPackages", mock.Anything, mock.Anything).
					Return(&db.PackagesResult{
						PageInfo: &pagination.PageInfo{TotalCount: pagination.StaticCount(test.injectPackagesInGroup)},
					}, nil)

				mockResourceLimits.On("GetResourceLimit", mock.Anything, mock.Anything).
					Return(&models.ResourceLimit{Value: test.limit}, nil)

				if test.expectErrCode == "" {
					mockTransactions.On("CommitTx", mock.Anything).Return(nil)
				}
			}

			dbClient := db.Client{
				Groups:         mockGroups,
				Packages:       mockPackages,
				Transactions:   mockTransactions,
				ResourceLimits: mockResourceLimits,
			}

			testLogger, _ := logger.NewForTest()
			service := NewService(testLogger, &dbClient, limits.NewLimitChecker(&dbClient), nil)

			pkg, err := service.CreatePackage(auth.WithCaller(ctx, mockCaller), test.input)
			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, packageID, pkg.Metadata.ID)
		})
	}
}

func TestCreatePackageVersion(t *testing.T) {
	groupID := "group-id"
	packageID := "package-id"
	packageVersionID := "package-version-id"

	tests := []struct {
		name                   string
		authError              error
		limit                  int
		injectVersionsInPeriod int32
		expectErrCode          errors.CodeType
	}{
		{
			name:                   "create version under limit",
			limit:                  5,
			injectVersionsInPeriod: 5,
		},
		{
			name:                   "exceeds versions per package per time period limit",
			limit:                  5,
			injectVersionsInPeriod: 6,
			expectErrCode:          errors.EInvalid,
		},
		{
			name:          "subject does not have permission",
			authError:     errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrCode: errors.EForbidden,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test := test

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockCaller.On("RequirePermission", mock.Anything, models.UpdatePackagePermission, mock.Anything).Return(test.authError)
			mockCaller.On("GetSubject").Return("mockSubject").Maybe()

			mockPackages := db.NewMockPackages(t)
			mockPackageVersions := db.NewMockPackageVersions(t)
			mockTransactions := db.NewMockTransactions(t)
			mockResourceLimits := db.NewMockResourceLimits(t)

			mockPackages.On("GetPackageByID", mock.Anything, packageID).
				Return(&models.Package{Metadata: models.ResourceMetadata{ID: packageID, TRN: "trn:package:test-group/test-package"}, GroupID: groupID}, nil)

			if test.authError == nil {
				// GetPackageVersions is called twice in order: first the latest-lookup before the
				// transaction (no prior versions), then the in-transaction count for the limit check.
				mockPackageVersions.On("GetPackageVersions", mock.Anything, mock.Anything).
					Return(&db.PackageVersionsResult{PageInfo: &pagination.PageInfo{TotalCount: pagination.StaticCount(0)}}, nil).Once()
				mockPackageVersions.On("GetPackageVersions", mock.Anything, mock.Anything).
					Return(&db.PackageVersionsResult{PageInfo: &pagination.PageInfo{TotalCount: pagination.StaticCount(test.injectVersionsInPeriod)}}, nil).Once()

				mockTransactions.On("BeginTx", mock.Anything).Return(auth.WithCaller(ctx, mockCaller), nil)
				mockTransactions.On("RollbackTx", mock.Anything).Return(nil)

				mockPackageVersions.On("CreatePackageVersion", mock.Anything, mock.Anything).
					Return(&models.PackageVersion{Metadata: models.ResourceMetadata{ID: packageVersionID, CreationTimestamp: ptr.Time(time.Now())}, PackageID: packageID}, nil)

				mockResourceLimits.On("GetResourceLimit", mock.Anything, mock.Anything).
					Return(&models.ResourceLimit{Value: test.limit}, nil)

				if test.expectErrCode == "" {
					mockTransactions.On("CommitTx", mock.Anything).Return(nil)
				}
			}

			dbClient := db.Client{
				Packages:        mockPackages,
				PackageVersions: mockPackageVersions,
				Transactions:    mockTransactions,
				ResourceLimits:  mockResourceLimits,
			}

			testLogger, _ := logger.NewForTest()
			service := NewService(testLogger, &dbClient, limits.NewLimitChecker(&dbClient), nil)

			pv, err := service.CreatePackageVersion(auth.WithCaller(ctx, mockCaller), &CreatePackageVersionInput{
				SemanticVersion: "1.0.0",
				PackageID:       packageID,
			})
			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, packageVersionID, pv.Metadata.ID)
		})
	}
}

func TestUploadPackageVersion(t *testing.T) {
	groupID := "group-id"
	packageID := "package-id"

	tests := []struct {
		name                 string
		authError            error
		versionStatus        models.PackageVersionStatus
		allowMutableVersions bool
		expectErrCode        errors.CodeType
	}{
		{
			name:          "subject does not have permission",
			authError:     errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			versionStatus: models.PackageVersionStatusPending,
			expectErrCode: errors.EForbidden,
		},
		{
			name:          "upload already in progress",
			versionStatus: models.PackageVersionStatusUploadInProgress,
			expectErrCode: errors.EConflict,
		},
		{
			name:                 "re-upload rejected when package does not allow mutable versions",
			versionStatus:        models.PackageVersionStatusUploaded,
			allowMutableVersions: false,
			expectErrCode:        errors.EConflict,
		},
		{
			name:                 "re-upload of errored version rejected when package does not allow mutable versions",
			versionStatus:        models.PackageVersionStatusErrored,
			allowMutableVersions: false,
			expectErrCode:        errors.EConflict,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test := test

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockCaller.On("RequirePermission", mock.Anything, models.UpdatePackagePermission, mock.Anything).Return(test.authError)

			mockPackages := db.NewMockPackages(t)
			mockPackages.On("GetPackageByID", mock.Anything, packageID).
				Return(&models.Package{
					Metadata:             models.ResourceMetadata{ID: packageID, TRN: "trn:package:test-group/test-package"},
					GroupID:              groupID,
					AllowMutableVersions: test.allowMutableVersions,
				}, nil)

			dbClient := db.Client{Packages: mockPackages}

			testLogger, _ := logger.NewForTest()
			service := NewService(testLogger, &dbClient, limits.NewLimitChecker(&dbClient), nil)

			err := service.UploadPackageVersion(auth.WithCaller(ctx, mockCaller), &models.PackageVersion{
				Metadata:        models.ResourceMetadata{ID: "package-version-id"},
				PackageID:       packageID,
				SemanticVersion: "1.0.0",
				Status:          test.versionStatus,
			}, strings.NewReader("bundle"))

			assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
		})
	}
}

func newTestPackageService(dbClient *db.Client, store corepkg.PackageStore) Service {
	testLogger, _ := logger.NewForTest()
	return NewService(testLogger, dbClient, limits.NewLimitChecker(dbClient), store)
}

func globalPackage() *models.Package {
	return &models.Package{
		Name:        "my-package",
		GroupID:     "group-1",
		RootGroupID: "root-1",
		Kind:        models.PackageKindOPAPolicy,
		Visibility:  models.PackageVisibilityGlobal,
		Metadata:    models.ResourceMetadata{ID: "pkg-1", TRN: "trn:package:root/team/my-package"},
	}
}

func TestGetPackageByID(t *testing.T) {
	tests := []struct {
		name            string
		setupMocks      func(*db.MockPackages)
		wantID          string
		expectErrorCode errors.CodeType
	}{
		{
			name: "returns the package when found",
			setupMocks: func(mockPackages *db.MockPackages) {
				mockPackages.On("GetPackageByID", mock.Anything, "pkg-1").Return(globalPackage(), nil)
			},
			wantID: "pkg-1",
		},
		{
			name: "returns not found when the package does not exist",
			setupMocks: func(mockPackages *db.MockPackages) {
				mockPackages.On("GetPackageByID", mock.Anything, "pkg-1").Return(nil, nil)
			},
			expectErrorCode: errors.ENotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockCaller := auth.NewMockCaller(t)
			mockPackages := db.NewMockPackages(t)
			tt.setupMocks(mockPackages)

			dbClient := &db.Client{Packages: mockPackages}
			got, err := newTestPackageService(dbClient, nil).GetPackageByID(auth.WithCaller(ctx, mockCaller), "pkg-1")

			if tt.expectErrorCode != "" {
				require.Error(t, err)
				assert.Equal(t, tt.expectErrorCode, errors.ErrorCode(err))
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tt.wantID, got.Metadata.ID)
		})
	}
}

func TestGetPackageByTRN(t *testing.T) {
	const packageTRN = "trn:package:root/team/my-package"

	tests := []struct {
		name            string
		setupMocks      func(*db.MockPackages)
		wantID          string
		expectErrorCode errors.CodeType
	}{
		{
			name: "returns the package when found",
			setupMocks: func(mockPackages *db.MockPackages) {
				mockPackages.On("GetPackageByTRN", mock.Anything, packageTRN).Return(globalPackage(), nil)
			},
			wantID: "pkg-1",
		},
		{
			name: "returns not found when the package does not exist",
			setupMocks: func(mockPackages *db.MockPackages) {
				mockPackages.On("GetPackageByTRN", mock.Anything, packageTRN).Return(nil, nil)
			},
			expectErrorCode: errors.ENotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockCaller := auth.NewMockCaller(t)
			mockPackages := db.NewMockPackages(t)
			tt.setupMocks(mockPackages)

			dbClient := &db.Client{Packages: mockPackages}
			got, err := newTestPackageService(dbClient, nil).GetPackageByTRN(auth.WithCaller(ctx, mockCaller), packageTRN)

			if tt.expectErrorCode != "" {
				require.Error(t, err)
				assert.Equal(t, tt.expectErrorCode, errors.ErrorCode(err))
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tt.wantID, got.Metadata.ID)
		})
	}
}

func TestGetPackagesByIDs(t *testing.T) {
	privatePackage := &models.Package{
		Name:       "private-package",
		GroupID:    "group-1",
		Kind:       models.PackageKindOPAPolicy,
		Visibility: models.PackageVisibilityPrivate,
		Metadata:   models.ResourceMetadata{ID: "pkg-2", TRN: "trn:package:root/team/private-package"},
	}

	tests := []struct {
		name            string
		packages        []models.Package
		authError       error
		expectErrorCode errors.CodeType
	}{
		{
			name:     "returns globally visible packages without an additional permission check",
			packages: []models.Package{*globalPackage()},
		},
		{
			name:            "returns forbidden when the caller cannot view a private package",
			packages:        []models.Package{*privatePackage},
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockCaller := auth.NewMockCaller(t)
			mockPackages := db.NewMockPackages(t)

			ids := make([]string, len(tt.packages))
			for i := range tt.packages {
				ids[i] = tt.packages[i].Metadata.ID
			}

			mockPackages.On("GetPackages", mock.Anything, &db.GetPackagesInput{
				Filter: &db.PackageFilter{PackageIDs: ids},
			}).Return(&db.PackagesResult{Packages: tt.packages}, nil)

			for _, p := range tt.packages {
				if p.Visibility == models.PackageVisibilityPrivate {
					mockCaller.On("RequireAccessToInheritableResource", mock.Anything, types.PackageModelType, mock.Anything).Return(tt.authError)
				}
			}

			dbClient := &db.Client{Packages: mockPackages}
			got, err := newTestPackageService(dbClient, nil).GetPackagesByIDs(auth.WithCaller(ctx, mockCaller), ids)

			if tt.expectErrorCode != "" {
				require.Error(t, err)
				assert.Equal(t, tt.expectErrorCode, errors.ErrorCode(err))
				return
			}
			require.NoError(t, err)
			assert.Len(t, got, len(tt.packages))
		})
	}
}

func TestGetPackages(t *testing.T) {
	group := &models.Group{Metadata: models.ResourceMetadata{ID: "group-1"}, FullPath: "root/team"}

	tests := []struct {
		name          string
		input         *GetPackagesInput
		setupMocks    func(*auth.MockCaller, *db.MockPackages)
		wantPackageID string
	}{
		{
			name:  "lists packages for a group",
			input: &GetPackagesInput{Group: group},
			setupMocks: func(mockCaller *auth.MockCaller, mockPackages *db.MockPackages) {
				mockCaller.On("RequirePermission", mock.Anything, mock.Anything, mock.Anything).Return(nil)
				mockPackages.On("GetPackages", mock.Anything, mock.Anything).Return(&db.PackagesResult{
					Packages: []models.Package{{Metadata: models.ResourceMetadata{ID: "pkg-1"}}},
				}, nil)
			},
			wantPackageID: "pkg-1",
		},
		{
			name:  "global listing for an admin",
			input: &GetPackagesInput{},
			setupMocks: func(mockCaller *auth.MockCaller, mockPackages *db.MockPackages) {
				mockCaller.On("IsAdminModeActivated", mock.Anything).Return(true)
				mockPackages.On("GetPackages", mock.Anything, mock.Anything).Return(&db.PackagesResult{
					Packages: []models.Package{{Metadata: models.ResourceMetadata{ID: "pkg-1"}}},
				}, nil)
			},
			wantPackageID: "pkg-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockCaller := auth.NewMockCaller(t)
			mockPackages := db.NewMockPackages(t)
			tt.setupMocks(mockCaller, mockPackages)

			dbClient := &db.Client{Packages: mockPackages}
			result, err := newTestPackageService(dbClient, nil).GetPackages(auth.WithCaller(ctx, mockCaller), tt.input)

			require.NoError(t, err)
			require.NotNil(t, result)
			require.Len(t, result.Packages, 1)
			assert.Equal(t, tt.wantPackageID, result.Packages[0].Metadata.ID)
		})
	}
}

// TestGetVisiblePackages covers the listing of every package a group may reference. Unlike
// TestGetPackages above it asserts the filter the service hands the DB layer, because that filter is
// the whole behavior of this method: which packages come back is decided entirely by it.
func TestGetVisiblePackages(t *testing.T) {
	group := &models.Group{Metadata: models.ResourceMetadata{ID: "group-1"}, FullPath: "root/mid/team"}

	t.Run("scopes the query to everything visible to the group", func(t *testing.T) {
		ctx := context.Background()
		mockCaller := auth.NewMockCaller(t)
		mockPackages := db.NewMockPackages(t)

		mockCaller.On("RequirePermission", mock.Anything, models.ViewPackagePermission, mock.Anything).Return(nil)

		var captured *db.GetPackagesInput
		mockPackages.On("GetPackages", mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				captured = args.Get(1).(*db.GetPackagesInput)
			}).
			Return(&db.PackagesResult{
				Packages: []models.Package{{Metadata: models.ResourceMetadata{ID: "pkg-1"}}},
			}, nil)

		dbClient := &db.Client{Packages: mockPackages}
		search := "base"
		result, err := newTestPackageService(dbClient, nil).GetVisiblePackages(
			auth.WithCaller(ctx, mockCaller),
			&GetVisiblePackagesInput{Group: group, Search: &search},
		)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, result.Packages, 1)

		require.NotNil(t, captured)
		require.NotNil(t, captured.Filter)
		assert.Equal(t, &search, captured.Filter.Search)
		// The visibility filter is what widens the listing; the plain group and ancestry filters must
		// not also be set, or they would AND it back down to the group's own subtree.
		assert.Nil(t, captured.Filter.GroupID)
		assert.Nil(t, captured.Filter.NamespacePaths)
		// The group's own path is all the DB layer needs; it expands the ancestry and takes the root
		// segment itself, so those derivations are covered by the DB integration tests.
		require.NotNil(t, captured.Filter.VisibleToGroup)
		assert.Equal(t, group.FullPath, *captured.Filter.VisibleToGroup)
	})

	t.Run("requires view access to the group", func(t *testing.T) {
		ctx := context.Background()
		mockCaller := auth.NewMockCaller(t)
		mockPackages := db.NewMockPackages(t)

		mockCaller.On("RequirePermission", mock.Anything, models.ViewPackagePermission, mock.Anything).
			Return(errors.New("forbidden", errors.WithErrorCode(errors.EForbidden)))

		dbClient := &db.Client{Packages: mockPackages}
		_, err := newTestPackageService(dbClient, nil).GetVisiblePackages(
			auth.WithCaller(ctx, mockCaller),
			&GetVisiblePackagesInput{Group: group},
		)

		require.Error(t, err)
		assert.Equal(t, errors.EForbidden, errors.ErrorCode(err))
	})

	t.Run("rejects a missing group", func(t *testing.T) {
		ctx := context.Background()
		mockCaller := auth.NewMockCaller(t)

		// Visibility is relative to a namespace, so there is no cross-group form of this listing.
		dbClient := &db.Client{Packages: db.NewMockPackages(t)}
		_, err := newTestPackageService(dbClient, nil).GetVisiblePackages(
			auth.WithCaller(ctx, mockCaller),
			&GetVisiblePackagesInput{},
		)

		require.Error(t, err)
		assert.Equal(t, errors.EInvalid, errors.ErrorCode(err))
	})
}

func TestUpdatePackage(t *testing.T) {
	tests := []struct {
		name            string
		setupMocks      func(*auth.MockCaller, *db.MockPackages, *db.MockTransactions)
		wantID          string
		expectErrorCode errors.CodeType
	}{
		{
			name: "updates the package",
			setupMocks: func(mockCaller *auth.MockCaller, mockPackages *db.MockPackages, mockTransactions *db.MockTransactions) {
				mockPackages.On("GetPackageByID", mock.Anything, "pkg-1").Return(globalPackage(), nil)
				mockCaller.On("RequirePermission", mock.Anything, mock.Anything, mock.Anything).Return(nil)
				mockTransactions.On("BeginTx", mock.Anything).Return(auth.WithCaller(context.Background(), mockCaller), nil)
				mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
				mockTransactions.On("CommitTx", mock.Anything).Return(nil)
				mockPackages.On("UpdatePackage", mock.Anything, mock.Anything).
					Return(&models.Package{Metadata: models.ResourceMetadata{ID: "pkg-1"}}, nil)
			},
			wantID: "pkg-1",
		},
		{
			name: "returns not found when the package does not exist",
			setupMocks: func(_ *auth.MockCaller, mockPackages *db.MockPackages, _ *db.MockTransactions) {
				mockPackages.On("GetPackageByID", mock.Anything, "pkg-1").Return(nil, nil)
			},
			expectErrorCode: errors.ENotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockCaller := auth.NewMockCaller(t)
			mockCaller.On("GetSubject").Return("mockSubject").Maybe()
			mockPackages := db.NewMockPackages(t)
			mockTransactions := db.NewMockTransactions(t)
			tt.setupMocks(mockCaller, mockPackages, mockTransactions)

			dbClient := &db.Client{Packages: mockPackages, Transactions: mockTransactions}
			got, err := newTestPackageService(dbClient, nil).UpdatePackage(
				auth.WithCaller(ctx, mockCaller),
				&UpdatePackageInput{ID: "pkg-1", Description: ptr.String("updated")},
			)

			if tt.expectErrorCode != "" {
				require.Error(t, err)
				assert.Equal(t, tt.expectErrorCode, errors.ErrorCode(err))
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tt.wantID, got.Metadata.ID)
		})
	}
}

func TestDeletePackage(t *testing.T) {
	t.Run("deletes the package", func(t *testing.T) {
		ctx := context.Background()
		mockCaller := auth.NewMockCaller(t)
		mockCaller.On("GetSubject").Return("mockSubject").Maybe()
		mockCaller.On("RequirePermission", mock.Anything, mock.Anything, mock.Anything).Return(nil)

		mockPackages := db.NewMockPackages(t)
		mockTransactions := db.NewMockTransactions(t)
		mockTransactions.On("BeginTx", mock.Anything).Return(auth.WithCaller(context.Background(), mockCaller), nil)
		mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
		mockTransactions.On("CommitTx", mock.Anything).Return(nil)
		mockPackages.On("DeletePackage", mock.Anything, mock.Anything).Return(nil)

		dbClient := &db.Client{Packages: mockPackages, Transactions: mockTransactions}
		err := newTestPackageService(dbClient, nil).DeletePackage(auth.WithCaller(ctx, mockCaller), globalPackage())
		require.NoError(t, err)
	})
}

// TestDeletePackageVersion covers the latest-flag promotion, which is the only part of the delete that
// isn't a straight pass-through: deleting the latest version has to hand the flag to the highest
// remaining semantic version, and deleting the only version must not try to promote anything.
func TestDeletePackageVersion(t *testing.T) {
	packageVersion := func(id, semanticVersion string, latest bool) *models.PackageVersion {
		return &models.PackageVersion{
			Metadata:        models.ResourceMetadata{ID: id, TRN: "trn:package_version:root/team/my-package/" + semanticVersion},
			PackageID:       "pkg-1",
			SemanticVersion: semanticVersion,
			Latest:          latest,
		}
	}

	tests := []struct {
		name string
		// toDelete is the version handed to the service.
		toDelete *models.PackageVersion
		// remaining is what the DB reports for the package, including the version being deleted. It is
		// only read when the deleted version holds the latest flag.
		remaining []models.PackageVersion
		// wantPromotedID is the version expected to be updated as the new latest, empty for none.
		wantPromotedID  string
		authError       error
		expectErrorCode errors.CodeType
	}{
		{
			name:     "deleting a non-latest version promotes nothing",
			toDelete: packageVersion("pv-1", "1.0.0", false),
		},
		{
			name:     "deleting the latest version promotes the highest remaining version",
			toDelete: packageVersion("pv-3", "2.0.0", true),
			remaining: []models.PackageVersion{
				*packageVersion("pv-1", "1.0.0", false),
				*packageVersion("pv-3", "2.0.0", true),
				*packageVersion("pv-2", "1.10.0", false),
			},
			wantPromotedID: "pv-2",
		},
		{
			name:      "deleting the only version promotes nothing",
			toDelete:  packageVersion("pv-1", "1.0.0", true),
			remaining: []models.PackageVersion{*packageVersion("pv-1", "1.0.0", true)},
		},
		{
			name:            "subject does not have permission",
			toDelete:        packageVersion("pv-1", "1.0.0", false),
			authError:       errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()

			mockCaller := auth.NewMockCaller(t)
			mockCaller.On("GetSubject").Return("mockSubject").Maybe()
			mockCaller.On("RequirePermission", mock.Anything, models.UpdatePackagePermission, mock.Anything).Return(test.authError)

			mockPackages := db.NewMockPackages(t)
			mockPackages.On("GetPackageByID", mock.Anything, "pkg-1").Return(globalPackage(), nil)

			mockPackageVersions := db.NewMockPackageVersions(t)
			mockTransactions := db.NewMockTransactions(t)

			if test.authError == nil {
				if test.toDelete.Latest {
					mockPackageVersions.On("GetPackageVersions", mock.Anything, mock.Anything).
						Return(&db.PackageVersionsResult{PackageVersions: test.remaining}, nil).Once()
				}

				mockTransactions.On("BeginTx", mock.Anything).Return(auth.WithCaller(ctx, mockCaller), nil)
				mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
				mockTransactions.On("CommitTx", mock.Anything).Return(nil)

				mockPackageVersions.On("DeletePackageVersion", mock.Anything, test.toDelete).Return(nil)

				if test.wantPromotedID != "" {
					mockPackageVersions.On("UpdatePackageVersion", mock.Anything, mock.MatchedBy(
						func(v *models.PackageVersion) bool {
							return v.Metadata.ID == test.wantPromotedID && v.Latest
						})).Return(&models.PackageVersion{}, nil).Once()
				}
			}

			dbClient := &db.Client{
				Packages:        mockPackages,
				PackageVersions: mockPackageVersions,
				Transactions:    mockTransactions,
			}

			err := newTestPackageService(dbClient, nil).DeletePackageVersion(auth.WithCaller(ctx, mockCaller), test.toDelete)

			if test.expectErrorCode != "" {
				require.Error(t, err)
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestGetPackageVersionByID(t *testing.T) {
	tests := []struct {
		name            string
		setupMocks      func(*db.MockPackageVersions, *db.MockPackages)
		wantID          string
		expectErrorCode errors.CodeType
	}{
		{
			name: "returns the version when found",
			setupMocks: func(mockVersions *db.MockPackageVersions, mockPackages *db.MockPackages) {
				mockVersions.On("GetPackageVersionByID", mock.Anything, "version-1").Return(&models.PackageVersion{
					PackageID: "pkg-1",
					Metadata:  models.ResourceMetadata{ID: "version-1"},
				}, nil)
				mockPackages.On("GetPackageByID", mock.Anything, "pkg-1").Return(globalPackage(), nil)
			},
			wantID: "version-1",
		},
		{
			name: "returns not found when the version does not exist",
			setupMocks: func(mockVersions *db.MockPackageVersions, _ *db.MockPackages) {
				mockVersions.On("GetPackageVersionByID", mock.Anything, "version-1").Return(nil, nil)
			},
			expectErrorCode: errors.ENotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockCaller := auth.NewMockCaller(t)
			mockVersions := db.NewMockPackageVersions(t)
			mockPackages := db.NewMockPackages(t)
			tt.setupMocks(mockVersions, mockPackages)

			dbClient := &db.Client{PackageVersions: mockVersions, Packages: mockPackages}
			got, err := newTestPackageService(dbClient, nil).GetPackageVersionByID(auth.WithCaller(ctx, mockCaller), "version-1")

			if tt.expectErrorCode != "" {
				require.Error(t, err)
				assert.Equal(t, tt.expectErrorCode, errors.ErrorCode(err))
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tt.wantID, got.Metadata.ID)
		})
	}
}

func TestGetPackageVersionByTRN(t *testing.T) {
	const versionTRN = "trn:package_version:root/team/my-package/1.0.0"

	tests := []struct {
		name            string
		setupMocks      func(*db.MockPackageVersions, *db.MockPackages)
		wantID          string
		expectErrorCode errors.CodeType
	}{
		{
			name: "returns the version when found",
			setupMocks: func(mockVersions *db.MockPackageVersions, mockPackages *db.MockPackages) {
				mockVersions.On("GetPackageVersionByTRN", mock.Anything, versionTRN).Return(&models.PackageVersion{
					PackageID: "pkg-1",
					Metadata:  models.ResourceMetadata{ID: "version-1", TRN: versionTRN},
				}, nil)
				mockPackages.On("GetPackageByID", mock.Anything, "pkg-1").Return(globalPackage(), nil)
			},
			wantID: "version-1",
		},
		{
			name: "returns not found when the version does not exist",
			setupMocks: func(mockVersions *db.MockPackageVersions, _ *db.MockPackages) {
				mockVersions.On("GetPackageVersionByTRN", mock.Anything, versionTRN).Return(nil, nil)
			},
			expectErrorCode: errors.ENotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockCaller := auth.NewMockCaller(t)
			mockVersions := db.NewMockPackageVersions(t)
			mockPackages := db.NewMockPackages(t)
			tt.setupMocks(mockVersions, mockPackages)

			dbClient := &db.Client{PackageVersions: mockVersions, Packages: mockPackages}
			got, err := newTestPackageService(dbClient, nil).GetPackageVersionByTRN(auth.WithCaller(ctx, mockCaller), versionTRN)

			if tt.expectErrorCode != "" {
				require.Error(t, err)
				assert.Equal(t, tt.expectErrorCode, errors.ErrorCode(err))
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tt.wantID, got.Metadata.ID)
		})
	}
}

func TestGetPackageVersionsByIDs(t *testing.T) {
	privatePackage := &models.Package{
		Name:       "private-package",
		GroupID:    "group-1",
		Kind:       models.PackageKindOPAPolicy,
		Visibility: models.PackageVisibilityPrivate,
		Metadata:   models.ResourceMetadata{ID: "pkg-2", TRN: "trn:package:root/team/private-package"},
	}

	tests := []struct {
		name            string
		versions        []models.PackageVersion
		pkg             *models.Package
		authError       error
		expectErrorCode errors.CodeType
	}{
		{
			name: "deduplicates the owning package lookup and its permission check across versions of the same package",
			versions: []models.PackageVersion{
				{PackageID: "pkg-2", Metadata: models.ResourceMetadata{ID: "version-1"}},
				{PackageID: "pkg-2", Metadata: models.ResourceMetadata{ID: "version-2"}},
			},
			pkg: privatePackage,
		},
		{
			name: "returns forbidden when the caller cannot view the owning package",
			versions: []models.PackageVersion{
				{PackageID: "pkg-2", Metadata: models.ResourceMetadata{ID: "version-1"}},
			},
			pkg:             privatePackage,
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockCaller := auth.NewMockCaller(t)
			mockVersions := db.NewMockPackageVersions(t)
			mockPackages := db.NewMockPackages(t)

			ids := make([]string, len(tt.versions))
			for i := range tt.versions {
				ids[i] = tt.versions[i].Metadata.ID
			}

			mockVersions.On("GetPackageVersions", mock.Anything, &db.GetPackageVersionsInput{
				Filter: &db.PackageVersionFilter{PackageVersionIDs: ids},
			}).Return(&db.PackageVersionsResult{PackageVersions: tt.versions}, nil)

			// .Once() enforces that the owning-package lookup is deduplicated across versions
			// sharing the same PackageID rather than queried once per version.
			mockPackages.On("GetPackageByID", mock.Anything, tt.pkg.Metadata.ID).Return(tt.pkg, nil).Once()

			if tt.pkg.Visibility == models.PackageVisibilityPrivate {
				// .Once() enforces that the permission check itself is deduplicated across versions
				// sharing the same PackageID rather than repeated once per version.
				mockCaller.On("RequireAccessToInheritableResource", mock.Anything, types.PackageModelType, mock.Anything).Return(tt.authError).Once()
			}

			dbClient := &db.Client{PackageVersions: mockVersions, Packages: mockPackages}
			got, err := newTestPackageService(dbClient, nil).GetPackageVersionsByIDs(auth.WithCaller(ctx, mockCaller), ids)

			if tt.expectErrorCode != "" {
				require.Error(t, err)
				assert.Equal(t, tt.expectErrorCode, errors.ErrorCode(err))
				return
			}
			require.NoError(t, err)
			assert.Len(t, got, len(tt.versions))
		})
	}
}

func TestGetPackageVersions(t *testing.T) {
	t.Run("lists versions for an authorized package", func(t *testing.T) {
		ctx := context.Background()
		mockCaller := auth.NewMockCaller(t)
		mockPackages := db.NewMockPackages(t)
		mockVersions := db.NewMockPackageVersions(t)

		mockPackages.On("GetPackageByID", mock.Anything, "pkg-1").Return(globalPackage(), nil)
		mockVersions.On("GetPackageVersions", mock.Anything, mock.Anything).Return(&db.PackageVersionsResult{
			PackageVersions: []models.PackageVersion{{Metadata: models.ResourceMetadata{ID: "version-1"}}},
		}, nil)

		dbClient := &db.Client{Packages: mockPackages, PackageVersions: mockVersions}
		result, err := newTestPackageService(dbClient, nil).GetPackageVersions(
			auth.WithCaller(ctx, mockCaller),
			&GetPackageVersionsInput{PackageID: "pkg-1"},
		)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, result.PackageVersions, 1)
		assert.Equal(t, "version-1", result.PackageVersions[0].Metadata.ID)
	})
}

func TestDownloadPackageVersion(t *testing.T) {
	t.Run("returns a presigned download URL", func(t *testing.T) {
		ctx := context.Background()
		const wantURL = "https://example.com/download"

		mockCaller := auth.NewMockCaller(t)
		mockPackages := db.NewMockPackages(t)
		mockStore := corepkg.NewMockPackageStore(t)

		packageVersion := &models.PackageVersion{PackageID: "pkg-1", Metadata: models.ResourceMetadata{ID: "version-1"}}
		mockPackages.On("GetPackageByID", mock.Anything, "pkg-1").Return(globalPackage(), nil)
		mockStore.On("GetPackageVersionPresignedURL", mock.Anything, packageVersion).Return(wantURL, nil)

		dbClient := &db.Client{Packages: mockPackages}
		got, err := newTestPackageService(dbClient, mockStore).DownloadPackageVersion(auth.WithCaller(ctx, mockCaller), packageVersion)

		require.NoError(t, err)
		assert.Equal(t, wantURL, got)
	})
}
