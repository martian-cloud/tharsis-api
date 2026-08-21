package packageregistry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestIsPackageVisibleToNamespace(t *testing.T) {
	tests := []struct {
		name          string
		visibility    models.PackageVisibility
		ownerTRN      string
		namespacePath string
		want          bool
	}{
		{
			name:          "global is visible everywhere",
			visibility:    models.PackageVisibilityGlobal,
			ownerTRN:      "trn:package:acme/team/my-package",
			namespacePath: "unrelated/group",
			want:          true,
		},
		{
			name:          "root_group visible within same root group",
			visibility:    models.PackageVisibilityRootGroup,
			ownerTRN:      "trn:package:acme/team/my-package",
			namespacePath: "acme/other",
			want:          true,
		},
		{
			name:          "root_group not visible in a different root group",
			visibility:    models.PackageVisibilityRootGroup,
			ownerTRN:      "trn:package:acme/team/my-package",
			namespacePath: "beta/team",
			want:          false,
		},
		{
			name:          "private visible in the owning group itself",
			visibility:    models.PackageVisibilityPrivate,
			ownerTRN:      "trn:package:acme/team/my-package",
			namespacePath: "acme/team",
			want:          true,
		},
		{
			name:          "private visible in a descendant group",
			visibility:    models.PackageVisibilityPrivate,
			ownerTRN:      "trn:package:acme/team/my-package",
			namespacePath: "acme/team/sub",
			want:          true,
		},
		{
			name:          "private not visible in an ancestor group",
			visibility:    models.PackageVisibilityPrivate,
			ownerTRN:      "trn:package:acme/team/my-package",
			namespacePath: "acme",
			want:          false,
		},
		{
			name:          "unknown visibility is not visible",
			visibility:    models.PackageVisibility("bogus"),
			ownerTRN:      "trn:package:acme/team/my-package",
			namespacePath: "acme/team",
			want:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg := &models.Package{
				Visibility: tt.visibility,
				Metadata:   models.ResourceMetadata{TRN: tt.ownerTRN},
			}
			assert.Equal(t, tt.want, IsPackageVisibleToNamespace(pkg, tt.namespacePath))
		})
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
				mockPackages.On("GetPackageByID", mock.Anything, "package-1").Return(&models.Package{
					Name:     "my-package",
					Metadata: models.ResourceMetadata{ID: "package-1"},
				}, nil)
			},
			wantID: "package-1",
		},
		{
			name: "returns not found when package does not exist",
			setupMocks: func(mockPackages *db.MockPackages) {
				mockPackages.On("GetPackageByID", mock.Anything, "package-1").Return(nil, nil)
			},
			expectErrorCode: errors.ENotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPackages := db.NewMockPackages(t)
			tt.setupMocks(mockPackages)

			dbClient := &db.Client{Packages: mockPackages}

			got, err := GetPackageByID(context.Background(), dbClient, "package-1")
			if tt.expectErrorCode != "" {
				require.Error(t, err)
				assert.Equal(t, tt.expectErrorCode, errors.ErrorCode(err))
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tt.wantID, got.GetID())
		})
	}
}

// versionsResult builds a GetPackageVersions result whose rows are all uploaded, with each version's
// id derived from its semantic version so assertions can name the row that was selected.
func versionsResult(versions ...string) *db.PackageVersionsResult {
	rows := make([]models.PackageVersion, 0, len(versions))
	for _, v := range versions {
		rows = append(rows, models.PackageVersion{
			SemanticVersion: v,
			Status:          models.PackageVersionStatusUploaded,
			Metadata:        models.ResourceMetadata{ID: "version-" + v},
		})
	}
	return &db.PackageVersionsResult{PackageVersions: rows}
}

// latestQuery matches the targeted query for the row flagged latest; scanQuery matches the unfiltered
// listing of every uploaded version, and exactQuery the lookup of one pinned version.
func latestQuery() interface{} {
	return mock.MatchedBy(func(in *db.GetPackageVersionsInput) bool {
		return in.Filter != nil && in.Filter.Latest != nil && *in.Filter.Latest
	})
}

func scanQuery() interface{} {
	return mock.MatchedBy(func(in *db.GetPackageVersionsInput) bool {
		return in.Filter != nil && in.Filter.Latest == nil && in.Filter.SemanticVersion == nil
	})
}

func exactQuery(version string) interface{} {
	return mock.MatchedBy(func(in *db.GetPackageVersionsInput) bool {
		return in.Filter != nil && in.Filter.SemanticVersion != nil && *in.Filter.SemanticVersion == version
	})
}

func TestResolveVersionConstraint(t *testing.T) {
	tests := []struct {
		name       string
		constraint string
		setupMocks func(*db.MockPackageVersions)
		wantID     string
		wantNil    bool
	}{
		{
			name:       "empty constraint takes the version flagged latest",
			constraint: "",
			setupMocks: func(mockVersions *db.MockPackageVersions) {
				mockVersions.On("GetPackageVersions", mock.Anything, latestQuery()).
					Return(versionsResult("1.2.0"), nil)
			},
			wantID: "version-1.2.0",
		},
		{
			// The latest flag is assigned at registration and is not cleared if that upload later
			// fails, so an empty constraint must fall back to the highest version actually uploaded.
			name:       "empty constraint falls back to the highest uploaded when latest is not uploaded",
			constraint: "",
			setupMocks: func(mockVersions *db.MockPackageVersions) {
				mockVersions.On("GetPackageVersions", mock.Anything, latestQuery()).
					Return(&db.PackageVersionsResult{}, nil)
				mockVersions.On("GetPackageVersions", mock.Anything, scanQuery()).
					Return(versionsResult("1.0.0", "1.2.0", "1.1.0"), nil)
			},
			wantID: "version-1.2.0",
		},
		{
			name:       "an exact version is queried directly rather than scanned",
			constraint: "1.1.0",
			setupMocks: func(mockVersions *db.MockPackageVersions) {
				mockVersions.On("GetPackageVersions", mock.Anything, exactQuery("1.1.0")).
					Return(versionsResult("1.1.0"), nil)
			},
			wantID: "version-1.1.0",
		},
		{
			name:       "an exact version that is not uploaded resolves to nothing",
			constraint: "9.9.9",
			setupMocks: func(mockVersions *db.MockPackageVersions) {
				mockVersions.On("GetPackageVersions", mock.Anything, exactQuery("9.9.9")).
					Return(&db.PackageVersionsResult{}, nil)
			},
			wantNil: true,
		},
		{
			name:       "a range takes the highest matching version",
			constraint: "~> 1.0",
			setupMocks: func(mockVersions *db.MockPackageVersions) {
				mockVersions.On("GetPackageVersions", mock.Anything, scanQuery()).
					Return(versionsResult("1.0.0", "1.9.0", "2.0.0", "1.2.0"), nil)
			},
			wantID: "version-1.9.0",
		},
		{
			name:       "a range no uploaded version satisfies resolves to nothing",
			constraint: ">= 3.0.0",
			setupMocks: func(mockVersions *db.MockPackageVersions) {
				mockVersions.On("GetPackageVersions", mock.Anything, scanQuery()).
					Return(versionsResult("1.0.0", "2.0.0"), nil)
			},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockVersions := db.NewMockPackageVersions(t)
			tt.setupMocks(mockVersions)

			dbClient := &db.Client{PackageVersions: mockVersions}

			got, err := ResolveVersionConstraint(context.Background(), dbClient, "package-1", tt.constraint)
			require.NoError(t, err)
			if tt.wantNil {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			assert.Equal(t, tt.wantID, got.GetID())
		})
	}
}

func TestGetUploadedPackageVersion(t *testing.T) {
	tests := []struct {
		name       string
		setupMocks func(*db.MockPackageVersions)
		wantID     string
		wantNil    bool
	}{
		{
			name: "returns the uploaded version when present",
			setupMocks: func(mockVersions *db.MockPackageVersions) {
				mockVersions.On("GetPackageVersions", mock.Anything, mock.Anything).Return(&db.PackageVersionsResult{
					PackageVersions: []models.PackageVersion{
						{
							SemanticVersion: "1.0.0",
							Status:          models.PackageVersionStatusUploaded,
							Metadata:        models.ResourceMetadata{ID: "version-1"},
						},
					},
				}, nil)
			},
			wantID: "version-1",
		},
		{
			name: "returns nil when no uploaded version exists",
			setupMocks: func(mockVersions *db.MockPackageVersions) {
				mockVersions.On("GetPackageVersions", mock.Anything, mock.Anything).
					Return(&db.PackageVersionsResult{PackageVersions: nil}, nil)
			},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockVersions := db.NewMockPackageVersions(t)
			tt.setupMocks(mockVersions)

			dbClient := &db.Client{PackageVersions: mockVersions}

			got, err := GetUploadedPackageVersion(context.Background(), dbClient, "package-1", "1.0.0")
			require.NoError(t, err)
			if tt.wantNil {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			assert.Equal(t, tt.wantID, got.GetID())
		})
	}
}
