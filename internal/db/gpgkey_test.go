//go:build integration

package db

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// getValue implements the sortableField interface for GPGKeySortableField
func (g GPGKeySortableField) getValue() string {
	return string(g)
}

func TestGPGKeys_CreateGPGKey(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-gpg",
		Description: "test group for gpg key",
		FullPath:    "test-group-gpg",
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		keyID           uint64
		groupID         string
		asciiArmor      string
	}

	testCases := []testCase{
		{
			name:       "create gpg key",
			keyID:      12345,
			groupID:    group.Metadata.ID,
			asciiArmor: "-----BEGIN PGP PUBLIC KEY BLOCK-----\ntest key data\n-----END PGP PUBLIC KEY BLOCK-----",
		},
		{
			name:            "create gpg key with invalid group ID",
			keyID:           67890,
			groupID:         invalidID,
			asciiArmor:      "-----BEGIN PGP PUBLIC KEY BLOCK-----\ntest key data\n-----END PGP PUBLIC KEY BLOCK-----",
			expectErrorCode: errors.EInternal,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			gpgKey, err := testClient.client.GPGKeys.CreateGPGKey(ctx, &models.GPGKey{
				GPGKeyID:   test.keyID,
				GroupID:    test.groupID,
				ASCIIArmor: test.asciiArmor,
				CreatedBy:  "db-integration-tests",
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, gpgKey)

			assert.Equal(t, test.keyID, gpgKey.GPGKeyID)
			assert.Equal(t, test.groupID, gpgKey.GroupID)
			assert.Equal(t, test.asciiArmor, gpgKey.ASCIIArmor)
			assert.NotEmpty(t, gpgKey.Metadata.ID)
		})
	}
}

func TestGPGKeys_DeleteGPGKey(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group and gpg key for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-gpg-delete",
		Description: "test group for gpg key delete",
		FullPath:    "test-group-gpg-delete",
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	createdGPGKey, err := testClient.client.GPGKeys.CreateGPGKey(ctx, &models.GPGKey{
		GPGKeyID:   98765,
		GroupID:    group.Metadata.ID,
		ASCIIArmor: "-----BEGIN PGP PUBLIC KEY BLOCK-----\nkey to delete\n-----END PGP PUBLIC KEY BLOCK-----",
		CreatedBy:  "db-integration-tests",
	})
	require.Nil(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		id              string
		version         int
	}

	testCases := []testCase{
		{
			name:    "delete gpg key",
			id:      createdGPGKey.Metadata.ID,
			version: createdGPGKey.Metadata.Version,
		},
		{
			name:            "delete will fail because resource version doesn't match",
			id:              createdGPGKey.Metadata.ID,
			expectErrorCode: errors.EOptimisticLock,
			version:         -1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := testClient.client.GPGKeys.DeleteGPGKey(ctx, &models.GPGKey{
				Metadata: models.ResourceMetadata{
					ID:      test.id,
					Version: test.version,
				},
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)

			// Verify gpg key was deleted
			gpgKey, err := testClient.client.GPGKeys.GetGPGKeyByID(ctx, test.id)
			assert.Nil(t, gpgKey)
			assert.Nil(t, err)
		})
	}
}

func TestGPGKeys_GetGPGKeyByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-gpg-key-get-by-id",
		Description: "test group for gpg key get by id",
		FullPath:    "test-group-gpg-key-get-by-id",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a GPG key for testing
	createdGPGKey, err := testClient.client.GPGKeys.CreateGPGKey(ctx, &models.GPGKey{
		GroupID:     group.Metadata.ID,
		ASCIIArmor:  "-----BEGIN PGP PUBLIC KEY BLOCK-----\ntest-key-content\n-----END PGP PUBLIC KEY BLOCK-----",
		Fingerprint: "1234567890ABCDEF1234567890ABCDEF12345678",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		id              string
		expectGPGKey    bool
	}

	testCases := []testCase{
		{
			name:         "get resource by id",
			id:           createdGPGKey.Metadata.ID,
			expectGPGKey: true,
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
			gpgKey, err := testClient.client.GPGKeys.GetGPGKeyByID(ctx, test.id)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectGPGKey {
				require.NotNil(t, gpgKey)
				assert.Equal(t, test.id, gpgKey.Metadata.ID)
			} else {
				assert.Nil(t, gpgKey)
			}
		})
	}
}

func TestGPGKeys_GetGPGKeys(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-gpg-keys-list",
		Description: "test group for gpg keys list",
		FullPath:    "test-group-gpg-keys-list",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create test GPG keys
	gpgKeys := []models.GPGKey{
		{
			GroupID:     group.Metadata.ID,
			ASCIIArmor:  "-----BEGIN PGP PUBLIC KEY BLOCK-----\ntest-key-1\n-----END PGP PUBLIC KEY BLOCK-----",
			Fingerprint: "1111111111111111111111111111111111111111",
			CreatedBy:   "db-integration-tests",
		},
		{
			GroupID:     group.Metadata.ID,
			ASCIIArmor:  "-----BEGIN PGP PUBLIC KEY BLOCK-----\ntest-key-2\n-----END PGP PUBLIC KEY BLOCK-----",
			Fingerprint: "2222222222222222222222222222222222222222",
			CreatedBy:   "db-integration-tests",
		},
	}

	createdGPGKeys := []models.GPGKey{}
	for _, gpgKey := range gpgKeys {
		created, err := testClient.client.GPGKeys.CreateGPGKey(ctx, &gpgKey)
		require.NoError(t, err)
		createdGPGKeys = append(createdGPGKeys, *created)
	}

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		input           *GetGPGKeysInput
		expectCount     int
	}

	testCases := []testCase{
		{
			name:        "get all gpg keys",
			input:       &GetGPGKeysInput{},
			expectCount: len(createdGPGKeys),
		},
		{
			name: "filter by namespace paths",
			input: &GetGPGKeysInput{
				Filter: &GPGKeyFilter{
					NamespacePaths: []string{group.FullPath},
				},
			},
			expectCount: len(createdGPGKeys),
		}}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.GPGKeys.GetGPGKeys(ctx, test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Len(t, result.GPGKeys, test.expectCount)
		})
	}
}

func TestGPGKeys_GetGPGKeysWithPaginationAndSorting(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-gpg-keys-pagination",
		Description: "test group for gpg keys pagination",
		FullPath:    "test-group-gpg-keys-pagination",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	resourceCount := 10
	for i := 0; i < resourceCount; i++ {
		_, err := testClient.client.GPGKeys.CreateGPGKey(ctx, &models.GPGKey{
			GroupID:     group.Metadata.ID,
			ASCIIArmor:  fmt.Sprintf("-----BEGIN PGP PUBLIC KEY BLOCK-----\ntest-key-%d\n-----END PGP PUBLIC KEY BLOCK-----", i),
			Fingerprint: fmt.Sprintf("%040d", i), // 40-character fingerprint
			CreatedBy:   "db-integration-tests",
		})
		require.NoError(t, err)
	}

	// Only test UpdatedAt fields to avoid GROUP_LEVEL complexity
	sortableFields := []sortableField{
		GPGKeySortableFieldUpdatedAtAsc,
		GPGKeySortableFieldUpdatedAtDesc,
	}

	testResourcePaginationAndSorting(ctx, t, resourceCount, sortableFields, func(ctx context.Context, sortByField sortableField, paginationOptions *pagination.Options) (*pagination.PageInfo, []pagination.CursorPaginatable, error) {
		sortBy := GPGKeySortableField(sortByField.getValue())

		result, err := testClient.client.GPGKeys.GetGPGKeys(ctx, &GetGPGKeysInput{
			Sort:              &sortBy,
			PaginationOptions: paginationOptions,
		})
		if err != nil {
			return nil, nil, err
		}

		resources := []pagination.CursorPaginatable{}
		for _, resource := range result.GPGKeys {
			resourceCopy := resource
			resources = append(resources, &resourceCopy)
		}

		return result.PageInfo, resources, nil
	})
}

func TestGPGKeys_GetGPGKeyByTRN(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-gpg-key-get-by-trn",
		Description: "test group for gpg key get by trn",
		FullPath:    "test-group-gpg-key-get-by-trn",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a GPG key for testing
	createdGPGKey, err := testClient.client.GPGKeys.CreateGPGKey(ctx, &models.GPGKey{
		GroupID:     group.Metadata.ID,
		ASCIIArmor:  "-----BEGIN PGP PUBLIC KEY BLOCK-----\ntest-key-trn\n-----END PGP PUBLIC KEY BLOCK-----",
		Fingerprint: "ABCDABCDABCDABCDABCDABCDABCDABCDABCDABCD",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		trn             string
		expectGPGKey    bool
	}

	testCases := []testCase{
		{
			name:         "get resource by TRN",
			trn:          createdGPGKey.Metadata.TRN,
			expectGPGKey: true,
		},
		{
			name: "resource with TRN not found",
			trn:  "trn:tharsis:gpg_key:non-existent-id",
		},
		{
			name:            "get resource with invalid TRN will return an error",
			trn:             "invalid-trn",
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			gpgKey, err := testClient.client.GPGKeys.GetGPGKeyByTRN(ctx, test.trn)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectGPGKey {
				require.NotNil(t, gpgKey)
				assert.Equal(t, createdGPGKey.Metadata.ID, gpgKey.Metadata.ID)
			} else {
				assert.Nil(t, gpgKey)
			}
		})
	}
}
