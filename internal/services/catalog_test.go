package services

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
)

func TestFetchModel(t *testing.T) {
	sampleRunGID := gid.ToGlobalID(types.RunModelType, uuid.NewString())

	type testCase struct {
		name            string
		searchValue     string
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:        "fetch by TRN",
			searchValue: trn.TypeRun.Build(sampleRunGID),
		},
		{
			name:        "fetch by GID",
			searchValue: sampleRunGID,
		},
		{
			name:            "TRN resource type not supported",
			searchValue:     "trn:invalid:some/path",
			expectErrorCode: errors.EInternal,
		},
		{
			name:            "gid code not supported",
			searchValue:     gid.ToGlobalID(types.ModelType{}, uuid.NewString()),
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			// We use just one service for testing since logic for all others is the same
			mockRunService := run.NewMockService(t)

			if trn.IsTRN(test.searchValue) && trn.MustParseAny(test.searchValue).Type().String() == types.RunModelType.Name() {
				mockRunService.On("GetRunByTRN", mock.Anything, test.searchValue).Return(&models.Run{}, nil)
			} else {
				mockRunService.On("GetRunByID", mock.Anything, gid.FromGlobalID(test.searchValue)).Return(&models.Run{}, nil).Maybe()
			}

			catalog := &Catalog{
				RunService: mockRunService,
			}

			catalog.Init()

			actualModel, err := catalog.FetchModel(t.Context(), test.searchValue)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			require.NotNil(t, actualModel)
			require.IsType(t, &models.Run{}, actualModel)
		})
	}
}

func TestFetchModelID(t *testing.T) {
	sampleRun := &models.Run{
		Metadata: models.ResourceMetadata{
			ID: uuid.NewString(),
		},
	}

	sampleRunGID := gid.ToGlobalID(types.RunModelType, sampleRun.Metadata.ID)

	type testCase struct {
		name            string
		searchValue     string
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:        "fetch by TRN",
			searchValue: trn.TypeRun.Build(sampleRunGID),
		},
		{
			name:        "fetch by GID",
			searchValue: sampleRunGID,
		},
		{
			name:            "TRN resource type not supported",
			searchValue:     "trn:invalid:some/path",
			expectErrorCode: errors.EInternal,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockRunService := run.NewMockService(t)

			if trn.IsTRN(test.searchValue) && trn.MustParseAny(test.searchValue).Type().String() == types.RunModelType.Name() {
				mockRunService.On("GetRunByTRN", mock.Anything, test.searchValue).Return(sampleRun, nil)
			} else {
				mockRunService.On("GetRunByID", mock.Anything, gid.FromGlobalID(test.searchValue)).Return(sampleRun, nil).Maybe()
			}

			catalog := &Catalog{
				RunService: mockRunService,
			}

			catalog.Init()

			actualID, err := catalog.FetchModelID(t.Context(), test.searchValue)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			require.NotEmpty(t, actualID)
		})
	}
}

func Test_Init(t *testing.T) {
	// We use just one service for testing since logic for all others is the same
	mockRunService := run.NewMockService(t)

	catalog := &Catalog{
		RunService: mockRunService,
	}

	catalog.Init()

	require.NotEmpty(t, catalog.gidFetchers)
	require.NotEmpty(t, catalog.trnFetchers)
}

func Test_addFetchers(t *testing.T) {
	mockRunService := run.NewMockService(t)

	catalog := &Catalog{
		RunService: mockRunService,
	}

	catalog.addModelFetchers(types.RunModelType,
		func(ctx context.Context, value string) (models.Model, error) {
			return mockRunService.GetRunByNodeID(ctx, value)
		},
		func(ctx context.Context, value string) (models.Model, error) {
			return mockRunService.GetRunByTRN(ctx, value)
		},
	)

	require.NotEmpty(t, catalog.gidFetchers)
	require.NotEmpty(t, catalog.trnFetchers)
}

func Test_getModelFetcherByTRNType(t *testing.T) {
	mockRunService := run.NewMockService(t)

	catalog := &Catalog{}

	catalog.addModelFetchers(types.RunModelType,
		func(ctx context.Context, value string) (models.Model, error) {
			return mockRunService.GetRunByNodeID(ctx, value)
		},
		func(ctx context.Context, value string) (models.Model, error) {
			return mockRunService.GetRunByTRN(ctx, value)
		},
	)

	// By ResourceType
	actualFetchMethods, ok := catalog.getModelFetcherByTRNType(types.RunModelType.TRNType())
	require.True(t, ok)
	require.NotNil(t, actualFetchMethods)
}

func Test_getModelFetcherByGIDCode(t *testing.T) {
	mockRunService := run.NewMockService(t)

	catalog := &Catalog{
		RunService: mockRunService,
	}

	catalog.addModelFetchers(types.RunModelType,
		func(ctx context.Context, value string) (models.Model, error) {
			return mockRunService.GetRunByNodeID(ctx, value)
		},
		func(ctx context.Context, value string) (models.Model, error) {
			return mockRunService.GetRunByTRN(ctx, value)
		},
	)

	// By GID
	actualFetchMethods, ok := catalog.getModelFetcherByGIDCode(types.RunModelType.GIDCode())
	require.True(t, ok)
	require.NotNil(t, actualFetchMethods)
}
