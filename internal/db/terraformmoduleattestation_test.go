//go:build integration

package db

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// terraformModuleAttestationInfo aids convenience in accessing the information
// TestGetModuleAttestations needs about the warmup objects.
type terraformModuleAttestationInfo struct {
	updateTime time.Time
	id         string
}

// terraformModuleAttestationInfoIDSlice makes a slice of terraformModuleAttestationInfo sortable by ID string
type terraformModuleAttestationInfoIDSlice []terraformModuleAttestationInfo

// terraformModuleAttestationInfoUpdateSlice makes a slice of terraformModuleAttestationInfo sortable by last updated time
type terraformModuleAttestationInfoUpdateSlice []terraformModuleAttestationInfo

// warmupTerraformModuleAttestations holds the inputs to and outputs from createWarmupTerraformModuleAttestations.
type warmupTerraformModuleAttestations struct {
	groups                      []models.Group
	terraformModules            []models.TerraformModule
	terraformModuleAttestations []models.TerraformModuleAttestation
}

func TestGetModuleAttestationByID(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupItems, err := createWarmupTerraformModuleAttestations(ctx, testClient, warmupTerraformModuleAttestations{
		groups:                      standardWarmupGroupsForTerraformModuleAttestations,
		terraformModules:            standardWarmupTerraformModulesForTerraformModulesAttestations,
		terraformModuleAttestations: standardWarmupTerraformModuleAttestations,
	})
	require.Nil(t, err)

	type testCase struct {
		expectMsg                        *string
		expectTerraformModuleAttestation *models.TerraformModuleAttestation
		name                             string
		searchID                         string
	}

	testCases := []testCase{
		{
			name:                             "get module attestation by ID",
			searchID:                         warmupItems.terraformModuleAttestations[0].Metadata.ID,
			expectTerraformModuleAttestation: &warmupItems.terraformModuleAttestations[0],
		},

		{
			name:     "returns nil because module attestation does not exist",
			searchID: nonExistentID,
		},

		{
			name:      "returns an error because the module attestation ID is invalid",
			searchID:  invalidID,
			expectMsg: ptr.String(ErrInvalidID.Error()),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			actualTerraformModuleAttestation, err := testClient.client.TerraformModuleAttestations.GetModuleAttestationByID(ctx, test.searchID)

			checkError(t, test.expectMsg, err)

			if test.expectTerraformModuleAttestation != nil {
				require.NotNil(t, actualTerraformModuleAttestation)
				assert.Equal(t, test.expectTerraformModuleAttestation, actualTerraformModuleAttestation)
			} else {
				assert.Nil(t, actualTerraformModuleAttestation)
			}
		})
	}
}

func TestGetModuleAttestationByTRN(t *testing.T) {
	ctx := t.Context()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name: "test-group",
	})
	require.NoError(t, err)

	module, err := testClient.client.TerraformModules.CreateModule(ctx, &models.TerraformModule{
		Name:        "test-module",
		System:      "aws",
		RootGroupID: group.Metadata.ID,
		GroupID:     group.Metadata.ID,
	})
	require.NoError(t, err)

	attestation, err := testClient.client.TerraformModuleAttestations.CreateModuleAttestation(ctx, &models.TerraformModuleAttestation{
		ModuleID:      module.Metadata.ID,
		Description:   "test attestation",
		Data:          "testdata",
		DataSHASum:    []byte("7ae471ed18395339572f5265b835860e28a2f85016455214cb214bafe442bc34"),
		SchemaType:    "https://in-toto.io/Statement/v0.1",
		PredicateType: "cosign.sigstore.dev/attestation/v1",
		Digests:       []string{"7ae471ed18395339572f5265b835860e28a2f85016455214cb214bafe4422c7d"},
		CreatedBy:     "test",
	})
	require.NoError(t, err)

	type testCase struct {
		name              string
		trn               string
		expectAttestation bool
		expectErrorCode   errors.CodeType
	}

	testCases := []testCase{
		{
			name:              "get attestation by TRN",
			trn:               attestation.Metadata.TRN,
			expectAttestation: true,
		},
		{
			name: "resource with TRN not found",
			trn:  types.TerraformModuleAttestationModelType.BuildTRN(group.FullPath, module.Name, module.System, "sha-sum"),
		},
		{
			name:            "attestation TRN has less than 4 parts",
			trn:             types.TerraformModuleAttestationModelType.BuildTRN("test-group"),
			expectErrorCode: errors.EInvalid,
		},
		{
			name:            "get resource with invalid TRN will return an error",
			trn:             "trn:invalid",
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualAttestation, err := testClient.client.TerraformModuleAttestations.GetModuleAttestationByTRN(ctx, test.trn)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)

			if test.expectAttestation {
				require.NotNil(t, actualAttestation)
				assert.Equal(t,
					types.TerraformModuleAttestationModelType.BuildTRN(
						group.FullPath,
						module.Name,
						module.System,
						string(attestation.DataSHASum),
					),
					actualAttestation.Metadata.TRN,
				)
			} else {
				assert.Nil(t, actualAttestation)
			}
		})
	}
}

func TestGetModuleAttestationsWithPagination(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupItems, err := createWarmupTerraformModuleAttestations(ctx, testClient, warmupTerraformModuleAttestations{
		groups:                      standardWarmupGroupsForTerraformModuleAttestations,
		terraformModules:            standardWarmupTerraformModulesForTerraformModulesAttestations,
		terraformModuleAttestations: standardWarmupTerraformModuleAttestations,
	})
	require.Nil(t, err)

	// Query for first page
	middleIndex := len(warmupItems.terraformModuleAttestations) / 2
	page1, err := testClient.client.TerraformModuleAttestations.GetModuleAttestations(ctx, &GetModuleAttestationsInput{
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(int32(middleIndex)),
		},
	})
	require.Nil(t, err)

	assert.Equal(t, middleIndex, len(page1.ModuleAttestations))
	assert.True(t, page1.PageInfo.HasNextPage)
	assert.False(t, page1.PageInfo.HasPreviousPage)

	cursor, err := page1.PageInfo.Cursor(&page1.ModuleAttestations[len(page1.ModuleAttestations)-1])
	require.Nil(t, err)

	remaining := len(warmupItems.terraformModuleAttestations) - middleIndex
	page2, err := testClient.client.TerraformModuleAttestations.GetModuleAttestations(ctx, &GetModuleAttestationsInput{
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(int32(remaining)),
			After: cursor,
		},
	})
	require.Nil(t, err)

	assert.Equal(t, remaining, len(page2.ModuleAttestations))
	assert.True(t, page2.PageInfo.HasPreviousPage)
	assert.False(t, page2.PageInfo.HasNextPage)
}

func TestGetModuleAttestations(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupItems, err := createWarmupTerraformModuleAttestations(ctx, testClient, warmupTerraformModuleAttestations{
		groups:                      standardWarmupGroupsForTerraformModuleAttestations,
		terraformModules:            standardWarmupTerraformModulesForTerraformModulesAttestations,
		terraformModuleAttestations: standardWarmupTerraformModuleAttestations,
	})
	require.Nil(t, err)

	allTerraformModuleAttestationInfos := terraformModuleAttestationInfoFromTerraformModuleAttestations(warmupItems.terraformModuleAttestations)

	// Sort by Terraform module attestation IDs.
	sort.Sort(terraformModuleAttestationInfoIDSlice(allTerraformModuleAttestationInfos))
	allTerraformModuleAttestationIDs := terraformModuleAttestationIDsFromTerraformModuleAttestationInfos(allTerraformModuleAttestationInfos)

	// Sort by last update times.
	sort.Sort(terraformModuleAttestationInfoUpdateSlice(allTerraformModuleAttestationInfos))
	allTerraformModuleAttestationIDsByTime := terraformModuleAttestationIDsFromTerraformModuleAttestationInfos(allTerraformModuleAttestationInfos)
	reverseTerraformModuleAttestationIDsByTime := reverseStringSlice(allTerraformModuleAttestationIDsByTime)

	type testCase struct {
		input                               *GetModuleAttestationsInput
		expectMsg                           *string
		name                                string
		expectTerraformModuleAttestationIDs []string
	}

	testCases := []testCase{
		{
			name: "non-nil but mostly empty input",
			input: &GetModuleAttestationsInput{
				Sort:              nil,
				PaginationOptions: nil,
				Filter:            nil,
			},
			expectTerraformModuleAttestationIDs: allTerraformModuleAttestationIDs,
		},

		{
			name: "populated sort and pagination, nil filter",
			input: &GetModuleAttestationsInput{
				Sort: ptrTerraformModuleAttestationSortableField(TerraformModuleAttestationSortableFieldCreatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
				Filter: nil,
			},
			expectTerraformModuleAttestationIDs: allTerraformModuleAttestationIDsByTime,
		},

		{
			name: "sort in ascending order of created at time",
			input: &GetModuleAttestationsInput{
				Sort: ptrTerraformModuleAttestationSortableField(TerraformModuleAttestationSortableFieldCreatedAtAsc),
			},
			expectTerraformModuleAttestationIDs: allTerraformModuleAttestationIDsByTime,
		},

		{
			name: "sort in descending order of created time",
			input: &GetModuleAttestationsInput{
				Sort: ptrTerraformModuleAttestationSortableField(TerraformModuleAttestationSortableFieldCreatedAtDesc),
			},
			expectTerraformModuleAttestationIDs: reverseTerraformModuleAttestationIDsByTime,
		},

		{
			name: "pagination, first one and last two, expect error",
			input: &GetModuleAttestationsInput{
				Sort: ptrTerraformModuleAttestationSortableField(TerraformModuleAttestationSortableFieldCreatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(1),
					Last:  ptr.Int32(2),
				},
			},
			expectMsg:                           ptr.String("only first or last can be defined, not both"),
			expectTerraformModuleAttestationIDs: allTerraformModuleAttestationIDs[4:],
		},

		{
			name: "filter, terraform module attestation IDs, positive",
			input: &GetModuleAttestationsInput{
				Sort: ptrTerraformModuleAttestationSortableField(TerraformModuleAttestationSortableFieldCreatedAtAsc),
				Filter: &TerraformModuleAttestationFilter{
					ModuleAttestationIDs: []string{
						allTerraformModuleAttestationIDsByTime[0], allTerraformModuleAttestationIDsByTime[1], allTerraformModuleAttestationIDsByTime[3]},
				},
			},
			expectTerraformModuleAttestationIDs: []string{
				allTerraformModuleAttestationIDsByTime[0], allTerraformModuleAttestationIDsByTime[1], allTerraformModuleAttestationIDsByTime[3],
			},
		},

		{
			name: "filter by digest",
			input: &GetModuleAttestationsInput{
				Sort: ptrTerraformModuleAttestationSortableField(TerraformModuleAttestationSortableFieldCreatedAtAsc),
				Filter: &TerraformModuleAttestationFilter{
					Digest: &standardWarmupTerraformModuleAttestations[0].Digests[0],
				},
			},
			expectTerraformModuleAttestationIDs: []string{
				allTerraformModuleAttestationIDsByTime[0],
				allTerraformModuleAttestationIDsByTime[1],
			},
		},

		{
			name: "filter, terraform module attestation IDs, non-existent",
			input: &GetModuleAttestationsInput{
				Sort: ptrTerraformModuleAttestationSortableField(TerraformModuleAttestationSortableFieldCreatedAtAsc),
				Filter: &TerraformModuleAttestationFilter{
					ModuleAttestationIDs: []string{nonExistentID},
				},
			},
			expectTerraformModuleAttestationIDs: []string{},
		},

		{
			name: "filter, terraform module attestation IDs, invalid ID",
			input: &GetModuleAttestationsInput{
				Sort: ptrTerraformModuleAttestationSortableField(TerraformModuleAttestationSortableFieldCreatedAtAsc),
				Filter: &TerraformModuleAttestationFilter{
					ModuleAttestationIDs: []string{invalidID},
				},
			},
			expectMsg:                           invalidUUIDMsg,
			expectTerraformModuleAttestationIDs: []string{},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			terraformModuleAttestationsResult, err := testClient.client.TerraformModuleAttestations.GetModuleAttestations(ctx, test.input)

			checkError(t, test.expectMsg, err)

			if err == nil {
				// Never returns nil if error is nil.
				require.NotNil(t, terraformModuleAttestationsResult.PageInfo)

				terraformModuleAttestations := terraformModuleAttestationsResult.ModuleAttestations

				// Check the terraform moduleAttestations result by comparing a list of the terraform module attestation IDs.
				actualTerraformModuleAttestationIDs := []string{}
				for _, terraformModuleAttestation := range terraformModuleAttestations {
					actualTerraformModuleAttestationIDs = append(actualTerraformModuleAttestationIDs, terraformModuleAttestation.Metadata.ID)
				}

				// If no sort direction was specified, sort the results here for repeatability.
				if test.input.Sort == nil {
					sort.Strings(actualTerraformModuleAttestationIDs)
				}

				assert.Equal(t, len(test.expectTerraformModuleAttestationIDs), len(actualTerraformModuleAttestationIDs))
				assert.Equal(t, test.expectTerraformModuleAttestationIDs, actualTerraformModuleAttestationIDs)
			}
		})
	}
}

func TestCreateModuleAttestation(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupItems, err := createWarmupTerraformModuleAttestations(ctx, testClient, warmupTerraformModuleAttestations{
		groups:           standardWarmupGroupsForTerraformModuleAttestations,
		terraformModules: standardWarmupTerraformModulesForTerraformModulesAttestations,
	})
	require.Nil(t, err)

	type testCase struct {
		toCreate      *models.TerraformModuleAttestation
		expectCreated *models.TerraformModuleAttestation
		expectMsg     *string
		name          string
	}

	now := time.Now()
	testCases := []testCase{
		{
			name: "positive",
			toCreate: &models.TerraformModuleAttestation{
				ModuleID:      warmupItems.terraformModules[0].Metadata.ID,
				Description:   "test 1",
				Data:          "testdata1",
				DataSHASum:    []byte("7ae471ed18395339572f5265b835860e28a2f85016455214cb214bafe442bc45"),
				SchemaType:    "https://in-toto.io/Statement/v0.1",
				PredicateType: "cosign.sigstore.dev/attestation/v1",
				Digests:       []string{"7ae471ed18395339572f5265b835860e28a2f85016455214cb214bafe4422c7d"},
				CreatedBy:     "TestCreateModuleAttestation",
			},
			expectCreated: &models.TerraformModuleAttestation{
				Metadata: models.ResourceMetadata{
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
				},
				ModuleID:      warmupItems.terraformModules[0].Metadata.ID,
				Description:   "test 1",
				Data:          "testdata1",
				DataSHASum:    []byte("7ae471ed18395339572f5265b835860e28a2f85016455214cb214bafe442bc45"),
				SchemaType:    "https://in-toto.io/Statement/v0.1",
				PredicateType: "cosign.sigstore.dev/attestation/v1",
				Digests:       []string{"7ae471ed18395339572f5265b835860e28a2f85016455214cb214bafe4422c7d"},
				CreatedBy:     "TestCreateModuleAttestation",
			},
		},

		{
			name: "duplicate attestation",
			toCreate: &models.TerraformModuleAttestation{
				ModuleID:      warmupItems.terraformModules[0].Metadata.ID,
				Description:   "test 1",
				Data:          "testdata1",
				DataSHASum:    []byte("7ae471ed18395339572f5265b835860e28a2f85016455214cb214bafe442bc45"),
				SchemaType:    "https://in-toto.io/Statement/v0.1",
				PredicateType: "cosign.sigstore.dev/attestation/v1",
				Digests:       []string{"7ae471ed18395339572f5265b835860e28a2f85016455214cb214bafe4422c7d"},
				CreatedBy:     "TestCreateModuleAttestation",
			},
			expectMsg: ptr.String("another module attestation with the same data already exists for this module"),
		},

		{
			name: "module does not exist",
			toCreate: &models.TerraformModuleAttestation{
				ModuleID:      nonExistentID,
				Description:   "test 1",
				Data:          "testdata1",
				DataSHASum:    []byte("7ae471ed18395339572f5265b835860e28a2f85016455214cb214bafe442bc34"),
				SchemaType:    "https://in-toto.io/Statement/v0.1",
				PredicateType: "cosign.sigstore.dev/attestation/v1",
				Digests:       []string{"7ae471ed18395339572f5265b835860e28a2f85016455214cb214bafe4422c7d"},
				CreatedBy:     "TestCreateModuleAttestation",
			},
			expectMsg: ptr.String("ERROR: insert or update on table \"terraform_module_attestations\" violates foreign key constraint \"fk_module_id\" (SQLSTATE 23503)"),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			actualCreated, err := testClient.client.TerraformModuleAttestations.CreateModuleAttestation(ctx, test.toCreate)

			checkError(t, test.expectMsg, err)

			if test.expectCreated != nil {
				require.NotNil(t, actualCreated)

				// The creation process must set the creation and last updated timestamps
				// between when the test case was created and when it the result is checked.
				whenCreated := test.expectCreated.Metadata.CreationTimestamp
				now := time.Now()

				compareTerraformModuleAttestations(t, test.expectCreated, actualCreated, false, &timeBounds{
					createLow:  whenCreated,
					createHigh: &now,
					updateLow:  whenCreated,
					updateHigh: &now,
				})
			} else {
				assert.Nil(t, actualCreated)
			}
		})
	}
}

func TestUpdateModuleAttestation(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupItems, err := createWarmupTerraformModuleAttestations(ctx, testClient, warmupTerraformModuleAttestations{
		groups:                      standardWarmupGroupsForTerraformModuleAttestations,
		terraformModules:            standardWarmupTerraformModulesForTerraformModulesAttestations,
		terraformModuleAttestations: standardWarmupTerraformModuleAttestations,
	})
	require.Nil(t, err)

	type testCase struct {
		expectMsg     *string
		toUpdate      *models.TerraformModuleAttestation
		expectUpdated *models.TerraformModuleAttestation
		name          string
	}

	now := time.Now()
	testCases := []testCase{

		{
			name: "positive",
			toUpdate: &models.TerraformModuleAttestation{
				Metadata: models.ResourceMetadata{
					ID:      warmupItems.terraformModuleAttestations[0].Metadata.ID,
					Version: initialResourceVersion,
				},
				Description: "test update",
			},
			expectUpdated: &models.TerraformModuleAttestation{
				Metadata: models.ResourceMetadata{
					ID:                   warmupItems.terraformModuleAttestations[0].Metadata.ID,
					Version:              initialResourceVersion + 1,
					CreationTimestamp:    warmupItems.terraformModuleAttestations[0].Metadata.CreationTimestamp,
					LastUpdatedTimestamp: &now,
				},
				ModuleID:      warmupItems.terraformModuleAttestations[0].ModuleID,
				Description:   "test update",
				Data:          "testdata1",
				DataSHASum:    []byte("7ae471ed18395339572f5265b835860e28a2f85016455214cb214bafe442bc34"),
				SchemaType:    "https://in-toto.io/Statement/v0.1",
				PredicateType: "cosign.sigstore.dev/attestation/v1",
				Digests:       []string{"7ae471ed18395339572f5265b835860e28a2f85016455214cb214bafe4422c7d"},
				CreatedBy:     "someone-0",
			},
		},

		{
			name: "negative, non-existent Terraform module attestation ID",
			toUpdate: &models.TerraformModuleAttestation{
				Metadata: models.ResourceMetadata{
					ID:      nonExistentID,
					Version: initialResourceVersion,
				},
			},
			expectMsg: resourceVersionMismatch,
		},

		{
			name: "defective-ID",
			toUpdate: &models.TerraformModuleAttestation{
				Metadata: models.ResourceMetadata{
					ID:      invalidID,
					Version: initialResourceVersion,
				},
			},
			expectMsg: invalidUUIDMsg,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			actualTerraformModuleAttestation, err := testClient.client.TerraformModuleAttestations.UpdateModuleAttestation(ctx, test.toUpdate)

			checkError(t, test.expectMsg, err)

			if test.expectUpdated != nil {
				// The creation process must set the creation and last updated timestamps
				// between when the test case was created and when it the result is checked.
				whenCreated := test.expectUpdated.Metadata.CreationTimestamp
				now := currentTime()

				require.NotNil(t, actualTerraformModuleAttestation)
				compareTerraformModuleAttestations(t, test.expectUpdated, actualTerraformModuleAttestation, false, &timeBounds{
					createLow:  whenCreated,
					createHigh: &now,
					updateLow:  whenCreated,
					updateHigh: &now,
				})
			} else {
				assert.Nil(t, actualTerraformModuleAttestation)
			}
		})
	}
}

func TestDeleteModuleAttestation(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupItems, err := createWarmupTerraformModuleAttestations(ctx, testClient, warmupTerraformModuleAttestations{
		groups:                      standardWarmupGroupsForTerraformModuleAttestations,
		terraformModules:            standardWarmupTerraformModulesForTerraformModulesAttestations,
		terraformModuleAttestations: standardWarmupTerraformModuleAttestations,
	})
	require.Nil(t, err)

	type testCase struct {
		expectMsg *string
		toDelete  *models.TerraformModuleAttestation
		name      string
	}

	testCases := []testCase{

		{
			name: "positive",
			toDelete: &models.TerraformModuleAttestation{
				Metadata: models.ResourceMetadata{
					ID:      warmupItems.terraformModuleAttestations[0].Metadata.ID,
					Version: initialResourceVersion,
				},
			},
		},

		{
			name: "negative, non-existent Terraform module ID",
			toDelete: &models.TerraformModuleAttestation{
				Metadata: models.ResourceMetadata{
					ID:      nonExistentID,
					Version: initialResourceVersion,
				},
			},
			expectMsg: resourceVersionMismatch,
		},

		{
			name: "defective-ID",
			toDelete: &models.TerraformModuleAttestation{
				Metadata: models.ResourceMetadata{
					ID:      invalidID,
					Version: initialResourceVersion,
				},
			},
			expectMsg: invalidUUIDMsg,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			err := testClient.client.TerraformModuleAttestations.DeleteModuleAttestation(ctx, test.toDelete)

			checkError(t, test.expectMsg, err)
		})
	}
}

//////////////////////////////////////////////////////////////////////////////

// Common utility structures and functions:

// Standard warmup group(s) for tests in this moduleAttestation:
// The create function will derive the parent path and name from the full path.
var standardWarmupGroupsForTerraformModuleAttestations = []models.Group{
	// Top-level groups:
	{
		Description: "top level group 0 for testing terraform module attestation functions",
		FullPath:    "top-level-group-0-for-terraform-module-attestations",
		CreatedBy:   "someone-g0",
	},
	{
		Description: "top level group 1 for testing terraform module attestation functions",
		FullPath:    "top-level-group-1-for-terraform-module-attestations",
		CreatedBy:   "someone-g1",
	},
	{
		Description: "top level group 2 for testing terraform module attestation functions",
		FullPath:    "top-level-group-2-for-terraform-module-attestations",
		CreatedBy:   "someone-g2",
	},
	{
		Description: "top level group 3 for testing terraform module attestation functions",
		FullPath:    "top-level-group-3-for-terraform-module-attestations",
		CreatedBy:   "someone-g3",
	},
	{
		Description: "top level group 4 for testing terraform module attestation functions",
		FullPath:    "top-level-group-4-for-terraform-module-attestations",
		CreatedBy:   "someone-g4",
	},
}

// The ID fields will be replaced by the real IDs during the create function.
var standardWarmupTerraformModulesForTerraformModulesAttestations = []models.TerraformModule{
	{
		Name:        "module-a",
		System:      "aws",
		RootGroupID: "top-level-group-0-for-terraform-module-attestations",
		GroupID:     "top-level-group-0-for-terraform-module-attestations",
		Private:     false,
		CreatedBy:   "someone-tp0",
	},
	{
		Name:        "module-b",
		System:      "azure",
		RootGroupID: "top-level-group-0-for-terraform-module-attestations",
		GroupID:     "top-level-group-0-for-terraform-module-attestations",
		Private:     false,
		CreatedBy:   "someone-tp1",
	},
	{
		Name:        "module-c",
		System:      "azure",
		RootGroupID: "top-level-group-0-for-terraform-module-attestations",
		GroupID:     "top-level-group-0-for-terraform-module-attestations",
		Private:     false,
		CreatedBy:   "someone-tp1",
	},
}

// Standard warmup terraform moduleAttestations for tests in this moduleAttestation:
// The ID fields will be replaced by the real IDs during the create function.
var standardWarmupTerraformModuleAttestations = []models.TerraformModuleAttestation{
	{
		ModuleID:      "top-level-group-0-for-terraform-module-attestations/module-a/aws",
		Description:   "test 1",
		Data:          "testdata1",
		DataSHASum:    []byte("7ae471ed18395339572f5265b835860e28a2f85016455214cb214bafe442bc34"),
		SchemaType:    "https://in-toto.io/Statement/v0.1",
		PredicateType: "cosign.sigstore.dev/attestation/v1",
		Digests:       []string{"7ae471ed18395339572f5265b835860e28a2f85016455214cb214bafe4422c7d"},
		CreatedBy:     "someone-0",
	},
	{
		ModuleID:      "top-level-group-0-for-terraform-module-attestations/module-a/aws",
		Description:   "test 2",
		Data:          "testdata2",
		DataSHASum:    []byte("7ae471ed18395339572f5265b835860e28a2f85016455214cb214bafe442bc35"),
		SchemaType:    "https://in-toto.io/Statement/v0.1",
		PredicateType: "cosign.sigstore.dev/attestation/v1",
		Digests:       []string{"7ae471ed18395339572f5265b835860e28a2f85016455214cb214bafe4422c7d", "7ae471ed18395339572f5265b835860e28a2f85016455214cb214bafe4422c7a"},
		CreatedBy:     "someone-0",
	},
	{
		ModuleID:      "top-level-group-0-for-terraform-module-attestations/module-b/azure",
		Description:   "test 3",
		Data:          "testdata3",
		DataSHASum:    []byte("7ae471ed18395339572f5265b835860e28a2f85016455214cb214bafe442bc36"),
		SchemaType:    "https://in-toto.io/Statement/v0.1",
		PredicateType: "cosign.sigstore.dev/attestation/v1",
		Digests:       []string{"7ae471ed18395339572f5265b835860e28a2f85016455214cb214bafe4422c7f"},
		CreatedBy:     "someone-0",
	},
	{
		ModuleID:      "top-level-group-0-for-terraform-module-attestations/module-c/azure",
		Description:   "test 4",
		Data:          "testdata4",
		DataSHASum:    []byte("7ae471ed18395339572f5265b835860e28a2f85016455214cb214bafe442bc37"),
		SchemaType:    "https://in-toto.io/Statement/v0.1",
		PredicateType: "cosign.sigstore.dev/attestation/v1",
		Digests:       []string{"7ae471ed18395339572f5265b835860e28a2f85016455214cb214bafe4422c7c"},
		CreatedBy:     "someone-0",
	},
}

// createWarmupTerraformModuleAttestations creates some warmup terraform moduleAttestations for a test
// The warmup terraform moduleAttestations to create can be standard or otherwise.
func createWarmupTerraformModuleAttestations(ctx context.Context, testClient *testClient,
	input warmupTerraformModuleAttestations) (*warmupTerraformModuleAttestations, error) {

	// It is necessary to create at least one group in order to
	// provide the necessary IDs for the terraform module attestations.

	resultGroups, parentPath2ID, err := createInitialGroups(ctx, testClient, input.groups)
	if err != nil {
		return nil, err
	}

	resultTerraformProviders, moduleResourcePath2ID, err := createInitialTerraformModules(ctx, testClient,
		input.terraformModules, parentPath2ID)
	if err != nil {
		return nil, err
	}

	resultTerraformProviderAttestations, err := createInitialTerraformModuleAttestations(ctx, testClient,
		input.terraformModuleAttestations, moduleResourcePath2ID)
	if err != nil {
		return nil, err
	}

	return &warmupTerraformModuleAttestations{
		groups:                      resultGroups,
		terraformModules:            resultTerraformProviders,
		terraformModuleAttestations: resultTerraformProviderAttestations,
	}, nil
}

func ptrTerraformModuleAttestationSortableField(arg TerraformModuleAttestationSortableField) *TerraformModuleAttestationSortableField {
	return &arg
}

func (wis terraformModuleAttestationInfoIDSlice) Len() int {
	return len(wis)
}

func (wis terraformModuleAttestationInfoIDSlice) Swap(i, j int) {
	wis[i], wis[j] = wis[j], wis[i]
}

func (wis terraformModuleAttestationInfoIDSlice) Less(i, j int) bool {
	return wis[i].id < wis[j].id
}

func (wis terraformModuleAttestationInfoUpdateSlice) Len() int {
	return len(wis)
}

func (wis terraformModuleAttestationInfoUpdateSlice) Swap(i, j int) {
	wis[i], wis[j] = wis[j], wis[i]
}

func (wis terraformModuleAttestationInfoUpdateSlice) Less(i, j int) bool {
	return wis[i].updateTime.Before(wis[j].updateTime)
}

// terraformModuleAttestationInfoFromTerraformModuleAttestations returns a slice of terraformModuleAttestationInfo, not necessarily sorted in any order.
func terraformModuleAttestationInfoFromTerraformModuleAttestations(terraformModuleAttestations []models.TerraformModuleAttestation) []terraformModuleAttestationInfo {
	result := []terraformModuleAttestationInfo{}

	for _, tp := range terraformModuleAttestations {
		result = append(result, terraformModuleAttestationInfo{
			id:         tp.Metadata.ID,
			updateTime: *tp.Metadata.LastUpdatedTimestamp,
		})
	}

	return result
}

// terraformModuleAttestationIDsFromTerraformModuleAttestationInfos preserves order
func terraformModuleAttestationIDsFromTerraformModuleAttestationInfos(terraformModuleAttestationInfos []terraformModuleAttestationInfo) []string {
	result := []string{}
	for _, terraformModuleAttestationInfo := range terraformModuleAttestationInfos {
		result = append(result, terraformModuleAttestationInfo.id)
	}
	return result
}

// compareTerraformModuleAttestations compares two terraform module attestation objects, including bounds for creation and updated times.
// If times is nil, it compares the exact metadata timestamps.
func compareTerraformModuleAttestations(t *testing.T, expected, actual *models.TerraformModuleAttestation,
	checkID bool, times *timeBounds) {

	assert.Equal(t, expected.ModuleID, actual.ModuleID)
	assert.Equal(t, expected.Data, actual.Data)
	assert.Equal(t, expected.DataSHASum, actual.DataSHASum)
	assert.Equal(t, expected.Description, actual.Description)
	assert.Equal(t, expected.Digests, actual.Digests)
	assert.Equal(t, expected.SchemaType, actual.SchemaType)
	assert.Equal(t, expected.PredicateType, actual.PredicateType)
	assert.Equal(t, expected.CreatedBy, actual.CreatedBy)

	if checkID {
		assert.Equal(t, expected.Metadata.ID, actual.Metadata.ID)
	}
	assert.Equal(t, expected.Metadata.Version, actual.Metadata.Version)
	assert.NotEmpty(t, actual.Metadata.TRN)

	// Compare timestamps.
	if times != nil {
		compareTime(t, times.createLow, times.createHigh, actual.Metadata.CreationTimestamp)
		compareTime(t, times.updateLow, times.updateHigh, actual.Metadata.LastUpdatedTimestamp)
	} else {
		assert.Equal(t, expected.Metadata.CreationTimestamp, actual.Metadata.CreationTimestamp)
		assert.Equal(t, expected.Metadata.LastUpdatedTimestamp, actual.Metadata.LastUpdatedTimestamp)
	}
}

// createInitialTerraformModuleAttestations creates some warmup Terraform moduleAttestations for a test.
func createInitialTerraformModuleAttestations(ctx context.Context, testClient *testClient,
	toCreate []models.TerraformModuleAttestation, moduleResourcePath2ID map[string]string) ([]models.TerraformModuleAttestation, error) {
	result := []models.TerraformModuleAttestation{}

	for _, input := range toCreate {

		moduleResourcePath := input.ModuleID
		moduleID, ok := moduleResourcePath2ID[moduleResourcePath]
		if !ok {
			return nil,
				fmt.Errorf("createInitialTerraformModuleAttestations failed to look up module resource path: %s",
					moduleResourcePath)
		}
		input.ModuleID = moduleID

		created, err := testClient.client.TerraformModuleAttestations.CreateModuleAttestation(ctx, &input)
		if err != nil {
			return nil, err
		}

		result = append(result, *created)
	}

	return result, nil
}
