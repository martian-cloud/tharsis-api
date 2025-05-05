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

func (sf FederatedRegistrySortableField) getValue() string {
	return string(sf)
}

func TestGetFederatedRegistryByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name: "test-group",
	})
	require.Nil(t, err)

	federatedRegistry, err := testClient.client.FederatedRegistries.CreateFederatedRegistry(ctx,
		&models.FederatedRegistry{
			Hostname: "remote.registry.host.example.invalid",
			GroupID:  group.Metadata.ID,
		})
	require.Nil(t, err)

	type testCase struct {
		expectErrorCode         errors.CodeType
		name                    string
		id                      string
		expectFederatedRegistry bool
	}

	testCases := []testCase{
		{
			name:                    "get resource by id",
			id:                      federatedRegistry.Metadata.ID,
			expectFederatedRegistry: true,
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

			federatedRegistry, err := testClient.client.FederatedRegistries.GetFederatedRegistryByID(ctx, test.id)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectFederatedRegistry {
				require.NotNil(t, federatedRegistry)
				assert.Equal(t, test.id, federatedRegistry.Metadata.ID)
			} else {
				assert.Nil(t, federatedRegistry)
			}
		})
	}
}

func TestGetFederatedRegistries(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group1, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name: "test-group-1",
	})
	require.Nil(t, err)

	group2, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name: "test-group-2",
	})
	require.Nil(t, err)

	federatedRegistry1, err := testClient.client.FederatedRegistries.
		CreateFederatedRegistry(ctx, &models.FederatedRegistry{
			Hostname: "1.remote.registry.host.example.invalid",
			GroupID:  group1.Metadata.ID,
		})
	require.Nil(t, err)

	federatedRegistry2, err := testClient.client.FederatedRegistries.
		CreateFederatedRegistry(ctx, &models.FederatedRegistry{
			Hostname: "2.remote.registry.host.example.invalid",
			GroupID:  group2.Metadata.ID,
		})
	require.Nil(t, err)

	type testCase struct {
		filter          *FederatedRegistryFilter
		name            string
		expectErrorCode errors.CodeType
		expectResults   []*models.FederatedRegistry
	}

	testCases := []testCase{
		{
			name: "empty filter, return all",
			filter: &FederatedRegistryFilter{
				FederatedRegistryIDs: nil,
				Hostname:             nil,
				GroupID:              nil,
			},
			expectResults: []*models.FederatedRegistry{
				federatedRegistry1,
				federatedRegistry2,
			},
		},
		{
			name: "filter by IDs, return one",
			filter: &FederatedRegistryFilter{
				FederatedRegistryIDs: []string{
					federatedRegistry1.Metadata.ID,
				},
			},
			expectResults: []*models.FederatedRegistry{
				federatedRegistry1,
			},
		},
		{
			name: "filter by IDs, return both",
			filter: &FederatedRegistryFilter{
				FederatedRegistryIDs: []string{
					federatedRegistry1.Metadata.ID,
					federatedRegistry2.Metadata.ID,
				},
			},
			expectResults: []*models.FederatedRegistry{
				federatedRegistry1,
				federatedRegistry2,
			},
		},
		{
			name: "filter by registry endpoint, return one",
			filter: &FederatedRegistryFilter{
				Hostname: &federatedRegistry2.Hostname,
			},
			expectResults: []*models.FederatedRegistry{
				federatedRegistry2,
			},
		},
		{
			name: "filter by group ID, return one",
			filter: &FederatedRegistryFilter{
				GroupID: &federatedRegistry1.GroupID,
			},
			expectResults: []*models.FederatedRegistry{
				federatedRegistry1,
			},
		},
		{
			name: "filter by group paths",
			filter: &FederatedRegistryFilter{
				GroupPaths: []string{
					group1.FullPath,
				},
			},
			expectResults: []*models.FederatedRegistry{
				federatedRegistry1,
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.FederatedRegistries.GetFederatedRegistries(ctx, &GetFederatedRegistriesInput{
				Filter: test.filter,
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)

			assert.ElementsMatch(t, test.expectResults, result.FederatedRegistries)
		})
	}
}

func TestGetFederatedRegistriesWithPaginationAndSorting(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name: "test-group1",
	})
	require.Nil(t, err)

	resourceCount := 10
	for i := 0; i < resourceCount; i++ {
		_, err := testClient.client.FederatedRegistries.CreateFederatedRegistry(ctx, &models.FederatedRegistry{
			Hostname: fmt.Sprintf("remote-%d.example.invalid", i),
			GroupID:  group.Metadata.ID,
		})
		require.Nil(t, err)
	}

	sortableFields := []sortableField{
		FederatedRegistrySortableFieldUpdatedAtAsc,
		FederatedRegistrySortableFieldUpdatedAtDesc,
	}

	testResourcePaginationAndSorting(ctx, t, resourceCount, sortableFields, func(ctx context.Context, sortByField sortableField, paginationOptions *pagination.Options) (*pagination.PageInfo, []pagination.CursorPaginatable, error) {
		sortBy := FederatedRegistrySortableField(sortByField.getValue())

		result, err := testClient.client.FederatedRegistries.GetFederatedRegistries(ctx, &GetFederatedRegistriesInput{
			Sort:              &sortBy,
			PaginationOptions: paginationOptions,
		})
		if err != nil {
			return nil, nil, err
		}

		resources := []pagination.CursorPaginatable{}
		for _, resource := range result.FederatedRegistries {
			resourceCopy := *resource
			resources = append(resources, &resourceCopy)
		}

		return result.PageInfo, resources, nil
	})
}

func TestCreateFederatedRegistry(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group1, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name: "test-group-1",
	})
	require.Nil(t, err)

	federatedRegistry1 := &models.FederatedRegistry{
		Hostname: "1.remote.registry.host.example.invalid",
		GroupID:  group1.Metadata.ID,
	}

	federatedRegistry2 := &models.FederatedRegistry{
		Hostname: "2.remote.registry.host.example.invalid",
		GroupID:  group1.Metadata.ID,
	}

	type testCase struct {
		input       *models.FederatedRegistry
		expectCode  errors.CodeType
		expectAdded *models.FederatedRegistry
		name        string
	}

	/*
		template test case:

		{
		name        string
		input       *models.FederatedRegistry
		expectCode  errors.CodeType
		expectAdded *models.FederatedRegistry
		}
	*/

	testCases := []testCase{
		{
			name:        "positive, registry 1",
			input:       federatedRegistry1,
			expectAdded: federatedRegistry1,
		},
		{
			name:        "positive, registry 2",
			input:       federatedRegistry2,
			expectAdded: federatedRegistry2,
		},
		{
			name:       "negative, duplicate registry 1",
			input:      federatedRegistry1,
			expectCode: errors.EConflict,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			claimedAdded, err := testClient.client.FederatedRegistries.
				CreateFederatedRegistry(ctx, test.input)

			if test.expectCode != "" {
				assert.Equal(t, test.expectCode, errors.ErrorCode(err))
				return
			}

			if test.expectAdded != nil {
				require.NotNil(t, claimedAdded)

				// Verify that what the CreateFederatedRegistry method claimed was added can fetched.
				fetched, err := testClient.client.FederatedRegistries.
					GetFederatedRegistryByID(ctx, claimedAdded.Metadata.ID)
				assert.Nil(t, err)

				if test.expectAdded != nil {
					require.NotNil(t, fetched)
					assert.Equal(t, claimedAdded, fetched)
				} else {
					assert.Nil(t, fetched)
				}
			} else {
				assert.Nil(t, claimedAdded)
			}
		})
	}
}

func TestUpdateFederatedRegistry(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	hostname := "remote.registry.host.example.invalid"
	conflictingURL := "conflicting.remote.registry.host.example.invalid"

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name: "test-group",
	})
	require.Nil(t, err)

	federatedRegistry, err := testClient.client.FederatedRegistries.
		CreateFederatedRegistry(ctx, &models.FederatedRegistry{
			Hostname: hostname,
			GroupID:  group.Metadata.ID,
		})
	require.Nil(t, err)

	conflicting, err := testClient.client.FederatedRegistries.
		CreateFederatedRegistry(ctx, &models.FederatedRegistry{
			Hostname: conflictingURL,
			GroupID:  group.Metadata.ID,
		})
	require.Nil(t, err)
	require.NotNil(t, conflicting)

	type testCase struct {
		input         *models.FederatedRegistry
		expectCode    errors.CodeType
		expectUpdated *models.FederatedRegistry
		name          string
	}

	/*
		template test case:

		{
		name          string
		input         *models.FederatedRegistry
		expectCode    errors.CodeType
		expectUpdated *models.FederatedRegistry
		}
	*/

	testCases := []testCase{
		{
			name: "change would conflict with existing registry", // must run before positive case
			input: &models.FederatedRegistry{
				Metadata: models.ResourceMetadata{
					ID:      federatedRegistry.Metadata.ID,
					Version: federatedRegistry.Metadata.Version,
				},
				Hostname: conflictingURL,
				GroupID:  group.Metadata.ID,
			},
			expectCode: errors.EConflict,
		},
		{
			name: "positive, updated successfully",
			input: &models.FederatedRegistry{
				Metadata: models.ResourceMetadata{
					ID:      federatedRegistry.Metadata.ID,
					Version: federatedRegistry.Metadata.Version,
				},
				Hostname: "updated.remote.registry.host.example.invalid",
				GroupID:  group.Metadata.ID,
			},
			expectUpdated: &models.FederatedRegistry{
				Metadata: models.ResourceMetadata{
					ID:      federatedRegistry.Metadata.ID,
					Version: federatedRegistry.Metadata.Version,
				},
				Hostname: "updated.remote.registry.host.example.invalid",
				GroupID:  group.Metadata.ID,
			},
		},
		{
			name: "federated registry does not exist",
			input: &models.FederatedRegistry{
				Metadata: models.ResourceMetadata{
					ID: nonExistentID,
				},
			},
			expectCode: errors.EOptimisticLock,
		},
		{
			name: "invalid ID",
			input: &models.FederatedRegistry{
				Metadata: models.ResourceMetadata{
					ID: invalidID,
				},
			},
			expectCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			claimedUpdated, err := testClient.client.FederatedRegistries.
				UpdateFederatedRegistry(ctx, test.input)

			if test.expectCode != "" {
				assert.Equal(t, test.expectCode, errors.ErrorCode(err))
				return
			}

			if test.expectUpdated != nil {
				require.NotNil(t, claimedUpdated)

				// Verify that what the UpdateFederatedRegistry method claimed was added can fetched.
				fetched, err := testClient.client.FederatedRegistries.
					GetFederatedRegistryByID(ctx, claimedUpdated.Metadata.ID)
				assert.Nil(t, err)

				if test.expectUpdated != nil {
					require.NotNil(t, fetched)
					assert.Equal(t, claimedUpdated, fetched)
				} else {
					assert.Nil(t, fetched)
				}
			} else {
				assert.Nil(t, claimedUpdated)
			}
		})
	}
}

func TestDeleteFederatedRegistry(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name: "test-group",
	})
	require.Nil(t, err)

	federatedRegistry, err := testClient.client.FederatedRegistries.
		CreateFederatedRegistry(ctx, &models.FederatedRegistry{
			Hostname: "remote.registry.host.example.invalid",
			GroupID:  group.Metadata.ID,
		})
	require.Nil(t, err)

	type testCase struct {
		input      *models.FederatedRegistry
		expectCode errors.CodeType
		name       string
	}

	testCases := []testCase{
		{
			name: "positive, deleted successfully",
			input: &models.FederatedRegistry{
				Metadata: models.ResourceMetadata{
					ID:      federatedRegistry.Metadata.ID,
					Version: federatedRegistry.Metadata.Version,
				},
				Hostname: "remote.registry.host.example.invalid",
				GroupID:  group.Metadata.ID,
			},
		},
		{
			name: "federated registry does not exist",
			input: &models.FederatedRegistry{
				Metadata: models.ResourceMetadata{
					ID: nonExistentID,
				},
			},
			expectCode: errors.EOptimisticLock,
		},
		{
			name: "invalid ID",
			input: &models.FederatedRegistry{
				Metadata: models.ResourceMetadata{
					ID: invalidID,
				},
			},
			expectCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			err := testClient.client.FederatedRegistries.DeleteFederatedRegistry(ctx, test.input)

			if test.expectCode != "" {
				assert.Equal(t, test.expectCode, errors.ErrorCode(err))
				return
			} else {
				assert.Nil(t, err)
			}
		})
	}
}
