//go:build integration

package db

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// getValue implements the sortableField interface for AsymSigningKeySortableField
func (as AsymSigningKeySortableField) getValue() string {
	return string(as)
}

func TestGetAsymSigningKeyByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	asymSigningKey, err := testClient.client.AsymSigningKeys.CreateAsymSigningKey(ctx, &models.AsymSigningKey{
		PublicKey:  []byte("test-public-key"),
		PluginData: []byte("test-plugin-data"),
		Status:     models.AsymSigningKeyStatusCreating,
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode      errors.CodeType
		name                 string
		id                   string
		expectAsymSigningKey bool
	}

	testCases := []testCase{
		{
			name:                 "get resource by id",
			id:                   asymSigningKey.Metadata.ID,
			expectAsymSigningKey: true,
		},
		{
			name: "resource with id not found",
			id:   nonExistentID,
		},
		{
			name:            "get resource with invalid id will return an error",
			id:              invalidID,
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			asymSigningKey, err := testClient.client.AsymSigningKeys.GetAsymSigningKeyByID(ctx, test.id)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectAsymSigningKey {
				require.NotNil(t, asymSigningKey)
				assert.Equal(t, test.id, asymSigningKey.Metadata.ID)
			} else {
				assert.Nil(t, asymSigningKey)
			}
		})
	}
}

func TestGetAsymSigningKeyByTRN(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	asymSigningKey, err := testClient.client.AsymSigningKeys.CreateAsymSigningKey(ctx, &models.AsymSigningKey{
		PublicKey:  []byte("test-public-key"),
		PluginData: []byte("test-plugin-data"),
		Status:     models.AsymSigningKeyStatusCreating,
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode      errors.CodeType
		name                 string
		trn                  string
		expectAsymSigningKey bool
	}

	testCases := []testCase{
		{
			name:                 "get resource by TRN",
			trn:                  asymSigningKey.Metadata.TRN,
			expectAsymSigningKey: true,
		},
		{
			name: "resource with TRN not found",
			trn:  types.AsymSigningKeyModelType.BuildTRN(nonExistentID),
		},
		{
			name:            "get resource with invalid TRN will return an error",
			trn:             "trn:invalid",
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			asymSigningKey, err := testClient.client.AsymSigningKeys.GetAsymSigningKeyByTRN(ctx, test.trn)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectAsymSigningKey {
				require.NotNil(t, asymSigningKey)
				assert.Equal(t, test.trn, asymSigningKey.Metadata.TRN)
			} else {
				assert.Nil(t, asymSigningKey)
			}
		})
	}
}

func TestCreateAsymSigningKey(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		asymSigningKey  *models.AsymSigningKey
	}

	testCases := []testCase{
		{
			name: "successfully create resource",
			asymSigningKey: &models.AsymSigningKey{
				PublicKey:  []byte("test-public-key"),
				PluginData: []byte("test-plugin-data"),
				Status:     models.AsymSigningKeyStatusCreating,
			},
		},
		{
			name: "successfully create resource with different data",
			asymSigningKey: &models.AsymSigningKey{
				PublicKey:  []byte("another-public-key"),
				PluginData: []byte("another-plugin-data"),
				Status:     models.AsymSigningKeyStatusDecommissioning,
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			asymSigningKey, err := testClient.client.AsymSigningKeys.CreateAsymSigningKey(ctx, test.asymSigningKey)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			require.NotNil(t, asymSigningKey)
			assert.Equal(t, test.asymSigningKey.PublicKey, asymSigningKey.PublicKey)
			assert.Equal(t, test.asymSigningKey.PluginData, asymSigningKey.PluginData)
		})
	}
}

func TestUpdateAsymSigningKey(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	asymSigningKey, err := testClient.client.AsymSigningKeys.CreateAsymSigningKey(ctx, &models.AsymSigningKey{
		PublicKey:  []byte("original-public-key"),
		PluginData: []byte("original-plugin-data"),
		Status:     models.AsymSigningKeyStatusCreating,
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		version         int
		publicKey       []byte
		pluginData      []byte
		status          models.AsymSigningKeyStatus
	}

	testCases := []testCase{
		{
			name:       "successfully update resource",
			version:    1,
			publicKey:  []byte("updated-public-key"),
			pluginData: []byte("updated-plugin-data"),
			status:     models.AsymSigningKeyStatusActive,
		},
		{
			name:            "update will fail because resource version doesn't match",
			version:         -1,
			publicKey:       []byte("updated-public-key"),
			pluginData:      []byte("updated-plugin-data"),
			status:          models.AsymSigningKeyStatusActive,
			expectErrorCode: errors.EOptimisticLock,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualAsymSigningKey, err := testClient.client.AsymSigningKeys.UpdateAsymSigningKey(ctx, &models.AsymSigningKey{
				Metadata: models.ResourceMetadata{
					ID:      asymSigningKey.Metadata.ID,
					Version: test.version,
				},
				PublicKey:  test.publicKey,
				PluginData: test.pluginData,
				Status:     test.status,
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			require.NotNil(t, actualAsymSigningKey)
			assert.Equal(t, test.publicKey, actualAsymSigningKey.PublicKey)
			assert.Equal(t, test.pluginData, actualAsymSigningKey.PluginData)
			assert.Equal(t, test.status, actualAsymSigningKey.Status)
		})
	}
}

func TestDeleteAsymSigningKey(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	asymSigningKey, err := testClient.client.AsymSigningKeys.CreateAsymSigningKey(ctx, &models.AsymSigningKey{
		PublicKey:  []byte("test-public-key"),
		PluginData: []byte("test-plugin-data"),
		Status:     models.AsymSigningKeyStatusCreating,
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		id              string
		version         int
	}

	testCases := []testCase{
		{
			name:            "delete will fail because resource version doesn't match",
			id:              asymSigningKey.Metadata.ID,
			expectErrorCode: errors.EOptimisticLock,
			version:         -1,
		},
		{
			name:    "successfully delete resource",
			id:      asymSigningKey.Metadata.ID,
			version: 1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := testClient.client.AsymSigningKeys.DeleteAsymSigningKey(ctx, &models.AsymSigningKey{
				Metadata: models.ResourceMetadata{
					ID:      test.id,
					Version: test.version,
				},
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestGetAsymSigningKeys(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create test asymmetric signing keys
	_, err := testClient.client.AsymSigningKeys.CreateAsymSigningKey(ctx, &models.AsymSigningKey{
		PublicKey:  []byte("test-public-key-1"),
		PluginData: []byte("test-plugin-data-1"),
		Status:     models.AsymSigningKeyStatusCreating,
	})
	require.NoError(t, err)

	_, err = testClient.client.AsymSigningKeys.CreateAsymSigningKey(ctx, &models.AsymSigningKey{
		PublicKey:  []byte("test-public-key-2"),
		PluginData: []byte("test-plugin-data-2"),
		Status:     models.AsymSigningKeyStatusDecommissioning,
	})
	require.NoError(t, err)

	type testCase struct {
		filter            *AsymSigningKeyFilter
		name              string
		expectErrorCode   errors.CodeType
		expectResultCount int
	}

	testCases := []testCase{
		{
			name:              "return all asymmetric signing keys",
			expectResultCount: 2,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.AsymSigningKeys.GetAsymSigningKeys(ctx, &GetAsymSigningKeysInput{
				Filter: test.filter,
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.expectResultCount, len(result.AsymSigningKeys))
		})
	}
}

func TestGetAsymSigningKeysWithPaginationAndSorting(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	resourceCount := 10

	for i := range resourceCount {
		_, err := testClient.client.AsymSigningKeys.CreateAsymSigningKey(ctx, &models.AsymSigningKey{
			PublicKey:  []byte(fmt.Sprintf("test-public-key-%d", i)),
			PluginData: []byte(fmt.Sprintf("test-plugin-data-%d", i)),
			Status:     models.AsymSigningKeyStatusDecommissioning,
		})
		require.NoError(t, err)
	}

	sortableFields := []sortableField{
		AsymSigningKeySortableFieldCreatedAtAsc,
		AsymSigningKeySortableFieldCreatedAtDesc,
	}

	testResourcePaginationAndSorting(ctx, t, resourceCount, sortableFields, func(ctx context.Context, sortByField sortableField, paginationOptions *pagination.Options) (*pagination.PageInfo, []pagination.CursorPaginatable, error) {
		sortBy := AsymSigningKeySortableField(sortByField.getValue())

		result, err := testClient.client.AsymSigningKeys.GetAsymSigningKeys(ctx, &GetAsymSigningKeysInput{
			Sort:              &sortBy,
			PaginationOptions: paginationOptions,
		})
		if err != nil {
			return nil, nil, err
		}

		resources := []pagination.CursorPaginatable{}
		for _, resource := range result.AsymSigningKeys {
			resourceCopy := resource
			resources = append(resources, &resourceCopy)
		}

		return result.PageInfo, resources, nil
	})
}
