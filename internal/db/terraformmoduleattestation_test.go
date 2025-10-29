//go:build integration

package db

import (
	"context"
	"fmt"
	"github.com/aws/smithy-go/ptr"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// getValue implements the sortableField interface for TerraformModuleAttestationSortableField
func (tma TerraformModuleAttestationSortableField) getValue() string {
	return string(tma)
}

func TestTerraformModuleAttestations_CreateTerraformModuleAttestation(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-module-attestation",
		Description: "test group for module attestation",
		FullPath:    "test-group-module-attestation",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a terraform module first (required dependency)
	terraformModule, err := testClient.client.TerraformModules.CreateModule(ctx, &models.TerraformModule{
		Name:        "test-module-attestation",
		System:      "terraform",
		GroupID:     group.Metadata.ID,
		RootGroupID: group.Metadata.ID,
		CreatedBy:   "db-integration-tests",
		Private:     false,
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		attestation     *models.TerraformModuleAttestation
	}

	testCases := []testCase{
		{
			name: "successfully create module attestation",
			attestation: &models.TerraformModuleAttestation{
				ModuleID:      terraformModule.Metadata.ID,
				Description:   "test attestation",
				SchemaType:    "https://in-toto.io/Statement/v0.1",
				PredicateType: "https://slsa.dev/provenance/v0.2",
				Data:          `{"test": "data"}`,
				DataSHASum:    []byte("test-sha-sum"),
				CreatedBy:     "db-integration-tests",
				Digests:       []string{"sha256:abc123"},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			attestation, err := testClient.client.TerraformModuleAttestations.CreateModuleAttestation(ctx, test.attestation)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			require.NotNil(t, attestation)
			assert.Equal(t, test.attestation.ModuleID, attestation.ModuleID)
			assert.Equal(t, test.attestation.Description, attestation.Description)
			assert.Equal(t, test.attestation.SchemaType, attestation.SchemaType)
		})
	}
}

func TestTerraformModuleAttestations_UpdateTerraformModuleAttestation(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-module-attestation-update",
		Description: "test group for module attestation update",
		FullPath:    "test-group-module-attestation-update",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a terraform module first (required dependency)
	terraformModule, err := testClient.client.TerraformModules.CreateModule(ctx, &models.TerraformModule{
		Name:        "test-module-attestation-update",
		System:      "terraform",
		GroupID:     group.Metadata.ID,
		RootGroupID: group.Metadata.ID,
		CreatedBy:   "db-integration-tests",
		Private:     false,
	})
	require.NoError(t, err)

	// Create a module attestation to update
	createdAttestation, err := testClient.client.TerraformModuleAttestations.CreateModuleAttestation(ctx, &models.TerraformModuleAttestation{
		ModuleID:      terraformModule.Metadata.ID,
		Description:   "test attestation for update",
		SchemaType:    "https://in-toto.io/Statement/v0.1",
		PredicateType: "https://slsa.dev/provenance/v0.2",
		Data:          `{"test": "data"}`,
		DataSHASum:    []byte("test-sha-sum"),
		CreatedBy:     "db-integration-tests",
		Digests:       []string{"sha256:abc123"},
	})
	require.NoError(t, err)

	type testCase struct {
		name              string
		expectErrorCode   errors.CodeType
		updateAttestation *models.TerraformModuleAttestation
	}

	testCases := []testCase{
		{
			name: "successfully update module attestation",
			updateAttestation: &models.TerraformModuleAttestation{
				Metadata:      createdAttestation.Metadata,
				ModuleID:      createdAttestation.ModuleID,
				Description:   "updated attestation description",
				SchemaType:    createdAttestation.SchemaType,
				PredicateType: createdAttestation.PredicateType,
				Data:          createdAttestation.Data,
				DataSHASum:    createdAttestation.DataSHASum,
				CreatedBy:     createdAttestation.CreatedBy,
				Digests:       createdAttestation.Digests,
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			attestation, err := testClient.client.TerraformModuleAttestations.UpdateModuleAttestation(ctx, test.updateAttestation)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			require.NotNil(t, attestation)
			assert.Equal(t, test.updateAttestation.Description, attestation.Description)
		})
	}
}

func TestTerraformModuleAttestations_GetModuleAttestationByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group and module for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-module-attestation-get-by-id",
		Description: "test group for module attestation get by id",
		FullPath:    "test-group-module-attestation-get-by-id",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	terraformModule, err := testClient.client.TerraformModules.CreateModule(ctx, &models.TerraformModule{
		Name:        "test-module-attestation-get-by-id",
		System:      "aws",
		GroupID:     group.Metadata.ID,
		RootGroupID: group.Metadata.ID,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a module attestation for testing
	createdModuleAttestation, err := testClient.client.TerraformModuleAttestations.CreateModuleAttestation(ctx, &models.TerraformModuleAttestation{
		ModuleID:      terraformModule.Metadata.ID,
		Description:   "test attestation",
		SchemaType:    "https://in-toto.io/Statement/v0.1",
		PredicateType: "https://slsa.dev/provenance/v0.2",
		Data:          `{"test": "data"}`,
		DataSHASum:    []byte("test-sha-sum"),
		Digests:       []string{"sha256:test"},
		CreatedBy:     "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode         errors.CodeType
		name                    string
		id                      string
		expectModuleAttestation bool
	}

	testCases := []testCase{
		{
			name:                    "get resource by id",
			id:                      createdModuleAttestation.Metadata.ID,
			expectModuleAttestation: true,
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
			moduleAttestation, err := testClient.client.TerraformModuleAttestations.GetModuleAttestationByID(ctx, test.id)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectModuleAttestation {
				require.NotNil(t, moduleAttestation)
				assert.Equal(t, test.id, moduleAttestation.Metadata.ID)
			} else {
				assert.Nil(t, moduleAttestation)
			}
		})
	}
}

func TestTerraformModuleAttestations_GetModuleAttestations(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group and module for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-module-attestations-list",
		Description: "test group for module attestations list",
		FullPath:    "test-group-module-attestations-list",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	terraformModule, err := testClient.client.TerraformModules.CreateModule(ctx, &models.TerraformModule{
		Name:        "test-module-attestations-list",
		System:      "aws",
		GroupID:     group.Metadata.ID,
		RootGroupID: group.Metadata.ID,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create test module attestations
	moduleAttestations := []models.TerraformModuleAttestation{
		{
			ModuleID:      terraformModule.Metadata.ID,
			Description:   "test attestation 1",
			SchemaType:    "https://in-toto.io/Statement/v0.1",
			PredicateType: "https://slsa.dev/provenance/v0.2",
			Data:          `{"test": "data"}`,
			DataSHASum:    []byte("test-sha-sum-1"),
			Digests:       []string{"sha256:test1"},
			CreatedBy:     "db-integration-tests",
		},
		{
			ModuleID:      terraformModule.Metadata.ID,
			Description:   "test attestation 2",
			SchemaType:    "https://in-toto.io/Statement/v0.1",
			PredicateType: "https://slsa.dev/provenance/v0.2",
			Data:          `{"test": "data"}`,
			DataSHASum:    []byte("test-sha-sum-2"),
			Digests:       []string{"sha256:test2"},
			CreatedBy:     "db-integration-tests",
		},
	}

	createdModuleAttestations := []models.TerraformModuleAttestation{}
	for _, moduleAttestation := range moduleAttestations {
		created, err := testClient.client.TerraformModuleAttestations.CreateModuleAttestation(ctx, &moduleAttestation)
		require.NoError(t, err)
		createdModuleAttestations = append(createdModuleAttestations, *created)
	}

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		input           *GetModuleAttestationsInput
		expectCount     int
	}

	testCases := []testCase{
		{
			name:        "get all module attestations",
			input:       &GetModuleAttestationsInput{},
			expectCount: len(createdModuleAttestations),
		},
		{
			name: "filter by module ID",
			input: &GetModuleAttestationsInput{
				Filter: &TerraformModuleAttestationFilter{
					ModuleID: &terraformModule.Metadata.ID,
				},
			},
			expectCount: len(createdModuleAttestations),
		},
		{
			name: "filter by digest",
			input: &GetModuleAttestationsInput{
				Filter: &TerraformModuleAttestationFilter{
					Digest: ptr.String("sha256:test1"),
				},
			},
			expectCount: 1,
		},
		{
			name: "filter by module attestation IDs",
			input: &GetModuleAttestationsInput{
				Filter: &TerraformModuleAttestationFilter{
					ModuleAttestationIDs: []string{createdModuleAttestations[0].Metadata.ID},
				},
			},
			expectCount: 1,
		}}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.TerraformModuleAttestations.GetModuleAttestations(ctx, test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Len(t, result.ModuleAttestations, test.expectCount)
		})
	}
}

func TestTerraformModuleAttestations_GetModuleAttestationsWithPaginationAndSorting(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group and module for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-module-attestations-pagination",
		Description: "test group for module attestations pagination",
		FullPath:    "test-group-module-attestations-pagination",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	terraformModule, err := testClient.client.TerraformModules.CreateModule(ctx, &models.TerraformModule{
		Name:        "test-module-attestations-pagination",
		System:      "aws",
		GroupID:     group.Metadata.ID,
		RootGroupID: group.Metadata.ID,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	resourceCount := 10
	for i := 0; i < resourceCount; i++ {
		_, err := testClient.client.TerraformModuleAttestations.CreateModuleAttestation(ctx, &models.TerraformModuleAttestation{
			ModuleID:      terraformModule.Metadata.ID,
			Description:   fmt.Sprintf("test attestation %d", i),
			SchemaType:    "https://in-toto.io/Statement/v0.1",
			PredicateType: "https://slsa.dev/provenance/v0.2",
			Data:          `{"test": "data"}`,
			DataSHASum:    []byte(fmt.Sprintf("test-sha-sum-%d", i)),
			Digests:       []string{fmt.Sprintf("sha256:test%d", i)},
			CreatedBy:     "db-integration-tests",
		})
		require.NoError(t, err)
	}

	sortableFields := []sortableField{
		TerraformModuleAttestationSortableFieldPredicateAsc,
		TerraformModuleAttestationSortableFieldPredicateDesc,
		TerraformModuleAttestationSortableFieldCreatedAtAsc,
		TerraformModuleAttestationSortableFieldCreatedAtDesc,
	}

	testResourcePaginationAndSorting(ctx, t, resourceCount, sortableFields, func(ctx context.Context, sortByField sortableField, paginationOptions *pagination.Options) (*pagination.PageInfo, []pagination.CursorPaginatable, error) {
		sortBy := TerraformModuleAttestationSortableField(sortByField.getValue())

		result, err := testClient.client.TerraformModuleAttestations.GetModuleAttestations(ctx, &GetModuleAttestationsInput{
			Sort:              &sortBy,
			PaginationOptions: paginationOptions,
		})
		if err != nil {
			return nil, nil, err
		}

		resources := []pagination.CursorPaginatable{}
		for _, resource := range result.ModuleAttestations {
			resourceCopy := resource
			resources = append(resources, &resourceCopy)
		}

		return result.PageInfo, resources, nil
	})
}

func TestTerraformModuleAttestations_GetModuleAttestationByTRN(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group and module for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-module-attestation-get-by-trn",
		Description: "test group for module attestation get by trn",
		FullPath:    "test-group-module-attestation-get-by-trn",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	terraformModule, err := testClient.client.TerraformModules.CreateModule(ctx, &models.TerraformModule{
		Name:        "test-module-attestation-get-by-trn",
		System:      "aws",
		GroupID:     group.Metadata.ID,
		RootGroupID: group.Metadata.ID,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a module attestation for testing
	createdModuleAttestation, err := testClient.client.TerraformModuleAttestations.CreateModuleAttestation(ctx, &models.TerraformModuleAttestation{
		ModuleID:      terraformModule.Metadata.ID,
		Description:   "test attestation",
		SchemaType:    "https://in-toto.io/Statement/v0.1",
		PredicateType: "https://slsa.dev/provenance/v0.2",
		Data:          `{"test": "data"}`,
		DataSHASum:    []byte("test-sha-sum"),
		Digests:       []string{"sha256:test"},
		CreatedBy:     "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode         errors.CodeType
		name                    string
		trn                     string
		expectModuleAttestation bool
	}

	testCases := []testCase{
		{
			name:                    "get resource by TRN",
			trn:                     createdModuleAttestation.Metadata.TRN,
			expectModuleAttestation: true,
		},
		{
			name: "resource with TRN not found",
			trn:  "trn:tharsis:terraform_module_attestation:non-existent-id",
		},
		{
			name:            "get resource with invalid TRN will return an error",
			trn:             "invalid-trn",
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			moduleAttestation, err := testClient.client.TerraformModuleAttestations.GetModuleAttestationByTRN(ctx, test.trn)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectModuleAttestation {
				require.NotNil(t, moduleAttestation)
				assert.Equal(t, createdModuleAttestation.Metadata.ID, moduleAttestation.Metadata.ID)
			} else {
				assert.Nil(t, moduleAttestation)
			}
		})
	}
}
