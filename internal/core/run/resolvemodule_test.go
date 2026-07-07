package run

import (
	"context"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/registry"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	modeltypes "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

const resolveWorkspaceID = "ws-1"

// withWorkspace stubs a successful workspace lookup on a fresh Workspaces mock.
func withWorkspace(ctx context.Context, t *testing.T) *db.MockWorkspaces {
	t.Helper()
	mockWorkspaces := db.NewMockWorkspaces(t)
	mockWorkspaces.On("GetWorkspaceByID", ctx, resolveWorkspaceID).
		Return(&models.Workspace{FullPath: "group/ws"}, nil)
	return mockWorkspaces
}

func TestResolveModule_NilSource(t *testing.T) {
	ctx := context.Background()
	// No DB or resolver calls expected for a configuration-version run.
	resolved, err := ResolveModule(ctx, &db.Client{}, registry.NewMockModuleResolver(t),
		resolveWorkspaceID, nil, nil, false, nil)
	require.NoError(t, err)
	require.NotNil(t, resolved)
	assert.Nil(t, resolved.Source)
	assert.Nil(t, resolved.Version)
	assert.Nil(t, resolved.Digest)
}

func TestResolveModule_WorkspaceErrors(t *testing.T) {
	ctx := context.Background()
	source := "registry.example.com/namespace/name/aws"

	t.Run("workspace lookup fails", func(t *testing.T) {
		mockWorkspaces := db.NewMockWorkspaces(t)
		mockWorkspaces.On("GetWorkspaceByID", ctx, resolveWorkspaceID).
			Return(nil, errors.New("boom", errors.WithErrorCode(errors.EInternal)))

		resolved, err := ResolveModule(ctx, &db.Client{Workspaces: mockWorkspaces},
			registry.NewMockModuleResolver(t), resolveWorkspaceID, &source, nil, false, nil)
		require.Error(t, err)
		assert.Equal(t, errors.EInternal, errors.ErrorCode(err))
		assert.Nil(t, resolved)
	})

	t.Run("workspace not found", func(t *testing.T) {
		mockWorkspaces := db.NewMockWorkspaces(t)
		mockWorkspaces.On("GetWorkspaceByID", ctx, resolveWorkspaceID).Return(nil, nil)

		resolved, err := ResolveModule(ctx, &db.Client{Workspaces: mockWorkspaces},
			registry.NewMockModuleResolver(t), resolveWorkspaceID, &source, nil, false, nil)
		require.Error(t, err)
		assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
		assert.Nil(t, resolved)
	})
}

func TestResolveModule_RemoteNonRegistrySource(t *testing.T) {
	ctx := context.Background()
	source := "git::https://example.com/repo.git"

	mockResolver := registry.NewMockModuleResolver(t)
	mockResolver.On("ParseModuleRegistrySource", ctx, source, mock.Anything, mock.Anything).
		Return(nil, registry.ErrRemoteModuleSource)

	resolved, err := ResolveModule(ctx, &db.Client{Workspaces: withWorkspace(ctx, t)},
		mockResolver, resolveWorkspaceID, &source, nil, false, nil)
	require.NoError(t, err)
	require.NotNil(t, resolved)
	assert.Nil(t, resolved.Source)
	assert.Nil(t, resolved.Version)
}

func TestResolveModule_NilRegistrySourceWithoutError(t *testing.T) {
	ctx := context.Background()
	source := "registry.example.com/namespace/name/aws"

	// Parse can return a nil source with no error for a remote source that doesn't
	// use the registry protocol; ResolveModule treats it as a zero-value result.
	mockResolver := registry.NewMockModuleResolver(t)
	mockResolver.On("ParseModuleRegistrySource", ctx, source, mock.Anything, mock.Anything).
		Return(nil, nil)

	resolved, err := ResolveModule(ctx, &db.Client{Workspaces: withWorkspace(ctx, t)},
		mockResolver, resolveWorkspaceID, &source, nil, false, nil)
	require.NoError(t, err)
	require.NotNil(t, resolved)
	assert.Nil(t, resolved.Source)
}

func TestResolveModule_ParseError(t *testing.T) {
	ctx := context.Background()
	source := "registry.example.com/namespace/name/aws"

	mockResolver := registry.NewMockModuleResolver(t)
	mockResolver.On("ParseModuleRegistrySource", ctx, source, mock.Anything, mock.Anything).
		Return(nil, errors.New("bad source"))

	resolved, err := ResolveModule(ctx, &db.Client{Workspaces: withWorkspace(ctx, t)},
		mockResolver, resolveWorkspaceID, &source, nil, false, nil)
	require.Error(t, err)
	assert.Equal(t, errors.EInvalid, errors.ErrorCode(err))
	assert.Nil(t, resolved)
}

func TestResolveModule_PublicModule_NormalizesVersion(t *testing.T) {
	ctx := context.Background()
	source := "registry.example.com/namespace/name/aws"
	digest := []byte{0x01, 0x02}

	mockSource := registry.NewMockModuleRegistrySource(t)
	mockSource.On("LocalRegistryModule", ctx).Return(&models.TerraformModule{Private: false}, nil)
	// The leading "v" before a digit must be stripped before resolution.
	mockSource.On("ResolveSemanticVersion", ctx, mock.MatchedBy(func(v *string) bool {
		return v != nil && *v == "1.2.3"
	}), false).Return("1.2.3", nil)
	mockSource.On("ResolveDigest", ctx, "1.2.3").Return(digest, nil)

	mockResolver := registry.NewMockModuleResolver(t)
	mockResolver.On("ParseModuleRegistrySource", ctx, source, mock.Anything, mock.Anything).
		Return(mockSource, nil)

	resolved, err := ResolveModule(ctx, &db.Client{Workspaces: withWorkspace(ctx, t)},
		mockResolver, resolveWorkspaceID, &source, ptr.String("v1.2.3"), false, nil)
	require.NoError(t, err)
	require.NotNil(t, resolved)
	assert.Equal(t, mockSource, resolved.Source)
	require.NotNil(t, resolved.Version)
	assert.Equal(t, "1.2.3", *resolved.Version)
	assert.Equal(t, digest, resolved.Digest)
}

func TestResolveModule_PrivateModule_Authorization(t *testing.T) {
	source := "registry.example.com/namespace/name/aws"
	module := &models.TerraformModule{Private: true, GroupID: "group-1"}

	newResolver := func(t *testing.T, ctx context.Context, mockSource registry.ModuleRegistrySource) registry.ModuleResolver {
		mockResolver := registry.NewMockModuleResolver(t)
		mockResolver.On("ParseModuleRegistrySource", ctx, source, mock.Anything, mock.Anything).
			Return(mockSource, nil)
		return mockResolver
	}

	t.Run("authorized caller proceeds to resolve version", func(t *testing.T) {
		mockCaller := auth.NewMockCaller(t)
		mockCaller.On("RequireAccessToInheritableResource", mock.Anything,
			modeltypes.TerraformModuleModelType, mock.Anything).Return(nil)
		ctx := auth.WithCaller(context.Background(), mockCaller)

		mockSource := registry.NewMockModuleRegistrySource(t)
		mockSource.On("LocalRegistryModule", ctx).Return(module, nil)
		mockSource.On("ResolveSemanticVersion", ctx, mock.Anything, false).Return("2.0.0", nil)
		mockSource.On("ResolveDigest", ctx, "2.0.0").Return([]byte{0xAA}, nil)

		resolved, err := ResolveModule(ctx, &db.Client{Workspaces: withWorkspace(ctx, t)},
			newResolver(t, ctx, mockSource), resolveWorkspaceID, &source, nil, false, nil)
		require.NoError(t, err)
		require.NotNil(t, resolved.Version)
		assert.Equal(t, "2.0.0", *resolved.Version)
	})

	t.Run("unauthorized caller is rejected", func(t *testing.T) {
		mockCaller := auth.NewMockCaller(t)
		mockCaller.On("RequireAccessToInheritableResource", mock.Anything,
			modeltypes.TerraformModuleModelType, mock.Anything).
			Return(errors.New("forbidden", errors.WithErrorCode(errors.EForbidden)))
		ctx := auth.WithCaller(context.Background(), mockCaller)

		mockSource := registry.NewMockModuleRegistrySource(t)
		mockSource.On("LocalRegistryModule", ctx).Return(module, nil)

		resolved, err := ResolveModule(ctx, &db.Client{Workspaces: withWorkspace(ctx, t)},
			newResolver(t, ctx, mockSource), resolveWorkspaceID, &source, nil, false, nil)
		require.Error(t, err)
		assert.Equal(t, errors.EForbidden, errors.ErrorCode(err))
		assert.Nil(t, resolved)
	})

	t.Run("no caller on context is rejected", func(t *testing.T) {
		ctx := context.Background()
		mockSource := registry.NewMockModuleRegistrySource(t)
		mockSource.On("LocalRegistryModule", ctx).Return(module, nil)

		resolved, err := ResolveModule(ctx, &db.Client{Workspaces: withWorkspace(ctx, t)},
			newResolver(t, ctx, mockSource), resolveWorkspaceID, &source, nil, false, nil)
		require.Error(t, err)
		assert.Nil(t, resolved)
	})
}

func TestResolveModule_ResolveErrors(t *testing.T) {
	ctx := context.Background()
	source := "registry.example.com/namespace/name/aws"

	newResolver := func(mockSource registry.ModuleRegistrySource) registry.ModuleResolver {
		mockResolver := registry.NewMockModuleResolver(t)
		mockResolver.On("ParseModuleRegistrySource", ctx, source, mock.Anything, mock.Anything).
			Return(mockSource, nil)
		return mockResolver
	}

	t.Run("LocalRegistryModule error propagates", func(t *testing.T) {
		mockSource := registry.NewMockModuleRegistrySource(t)
		mockSource.On("LocalRegistryModule", ctx).Return(nil, errors.New("boom"))

		resolved, err := ResolveModule(ctx, &db.Client{Workspaces: withWorkspace(ctx, t)},
			newResolver(mockSource), resolveWorkspaceID, &source, nil, false, nil)
		require.Error(t, err)
		assert.Nil(t, resolved)
	})

	t.Run("ResolveSemanticVersion error is invalid", func(t *testing.T) {
		mockSource := registry.NewMockModuleRegistrySource(t)
		mockSource.On("LocalRegistryModule", ctx).Return(&models.TerraformModule{}, nil)
		mockSource.On("ResolveSemanticVersion", ctx, mock.Anything, false).
			Return("", errors.New("no such version"))

		resolved, err := ResolveModule(ctx, &db.Client{Workspaces: withWorkspace(ctx, t)},
			newResolver(mockSource), resolveWorkspaceID, &source, nil, false, nil)
		require.Error(t, err)
		assert.Equal(t, errors.EInvalid, errors.ErrorCode(err))
		assert.Nil(t, resolved)
	})

	t.Run("ResolveDigest error propagates", func(t *testing.T) {
		mockSource := registry.NewMockModuleRegistrySource(t)
		mockSource.On("LocalRegistryModule", ctx).Return(&models.TerraformModule{}, nil)
		mockSource.On("ResolveSemanticVersion", ctx, mock.Anything, false).Return("1.0.0", nil)
		mockSource.On("ResolveDigest", ctx, "1.0.0").Return(nil, errors.New("digest failure"))

		resolved, err := ResolveModule(ctx, &db.Client{Workspaces: withWorkspace(ctx, t)},
			newResolver(mockSource), resolveWorkspaceID, &source, nil, false, nil)
		require.Error(t, err)
		assert.Nil(t, resolved)
	})
}
