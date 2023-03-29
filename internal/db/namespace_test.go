//go:build integration

package db

import (
	"context"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

type namespaceWarmupsInput struct {
	groups     []models.Group
	workspaces []models.Workspace
}

type namespaceWarmupsOutput struct {
	groupID2Path     map[string]string
	workspaceID2Path map[string]string
	groups           []models.Group
	workspaces       []models.Workspace
}

// pathChecksType contains maps from group/workspace ID to namespace path and is used for the group migration test.
type pathChecksType struct {
	groups     map[models.Group]string
	workspaces map[models.Workspace]string
}

func TestGetNamespaceByGroupID(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdWarmupOutput, err := createWarmupNamespaces(ctx, testClient, &namespaceWarmupsInput{
		groups: standardWarmupGroupsForNamespaces,
	})
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup objects weren't all created.
		return
	}

	type testCase struct {
		expectMsg       *string
		expectNamespace *namespaceRow
		name            string
		input           string
	}

	/*
		template test case:

		{
		name            string
		input           string
		expectMsg       *string
		expectNamespace *namespaceRow
		}
	*/

	testCases := []testCase{}

	// Positive case, one warmup group at a time.
	for _, toGet := range createdWarmupOutput.groups {
		testCases = append(testCases, testCase{
			name:  "positive-group--" + toGet.FullPath,
			input: toGet.Metadata.ID,
			expectNamespace: &namespaceRow{
				path:    toGet.FullPath,
				groupID: toGet.Metadata.ID,
				version: initialResourceVersion,
			},
		})
	}

	testCases = append(testCases,
		testCase{
			name:  "negative: non-exist",
			input: nonExistentID,
		},
		testCase{
			name:      "negative: invalid",
			input:     invalidID,
			expectMsg: invalidUUIDMsg1,
		},
	)

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			gotNamespace, err := getNamespaceByGroupID(ctx, testClient.client.getConnection(ctx), test.input)

			checkError(t, test.expectMsg, err)

			if test.expectNamespace != nil {
				require.NotNil(t, gotNamespace)
				compareNamespaceRows(t, test.expectNamespace, gotNamespace)
			} else {
				assert.Nil(t, gotNamespace)
			}

		})
	}
}

func TestGetNamespaceByWorkspaceID(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdWarmupOutput, err := createWarmupNamespaces(ctx, testClient, &namespaceWarmupsInput{
		groups:     standardWarmupGroupsForNamespaces,
		workspaces: standardWarmupWorkspacesForNamespaces,
	})
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup objects weren't all created.
		return
	}

	// Creating the groups above is necessary in order to create the workspaces.
	// However, to make sure the groups are not inadvertently used in the test
	// execution, hide them from the rest of this test function.
	createdWarmupOutput.groups = nil

	type testCase struct {
		expectMsg       *string
		expectNamespace *namespaceRow
		name            string
		input           string
	}

	/*
		template test case:

		{
		name            string
		input           string
		expectMsg       *string
		expectNamespace *namespaceRow
		}
	*/

	testCases := []testCase{}

	// Positive case, one warmup workspace at a time.
	for _, toGet := range createdWarmupOutput.workspaces {
		testCases = append(testCases, testCase{
			name:  "positive-workspace--" + toGet.FullPath,
			input: toGet.Metadata.ID,
			expectNamespace: &namespaceRow{
				path:        toGet.FullPath,
				workspaceID: toGet.Metadata.ID,
				version:     initialResourceVersion,
			},
		})
	}

	testCases = append(testCases,
		testCase{
			name:  "negative: non-exist",
			input: nonExistentID,
		},
		testCase{
			name:      "negative: invalid",
			input:     invalidID,
			expectMsg: invalidUUIDMsg1,
		},
	)

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			gotNamespace, err := getNamespaceByWorkspaceID(ctx, testClient.client.getConnection(ctx), test.input)

			checkError(t, test.expectMsg, err)

			if test.expectNamespace != nil {
				require.NotNil(t, gotNamespace)
				compareNamespaceRows(t, test.expectNamespace, gotNamespace)
			} else {
				assert.Nil(t, gotNamespace)
			}

		})
	}
}

func TestGetNamespaceByPath(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdWarmupOutput, err := createWarmupNamespaces(ctx, testClient, &namespaceWarmupsInput{
		groups:     standardWarmupGroupsForNamespaces,
		workspaces: standardWarmupWorkspacesForNamespaces,
	})
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup objects weren't all created.
		return
	}

	type testCase struct {
		expectMsg       *string
		expectNamespace *namespaceRow
		name            string
		input           string
	}

	/*
		template test case:

		{
		name            string
		input           string
		expectMsg       *string
		expectNamespace *namespaceRow
		}
	*/

	testCases := []testCase{}

	// Positive cases, one warmup group at a time.
	for _, group := range createdWarmupOutput.groups {
		testCases = append(testCases, testCase{
			name:  "positive-group-path--" + group.FullPath,
			input: group.FullPath,
			expectNamespace: &namespaceRow{
				path:    group.FullPath,
				groupID: group.Metadata.ID,
				version: initialResourceVersion,
			},
		})
	}

	// Positive cases, one warmup workspace at a time.
	for _, workspace := range createdWarmupOutput.workspaces {
		testCases = append(testCases, testCase{
			name:  "positive-workspace-path--" + workspace.FullPath,
			input: workspace.FullPath,
			expectNamespace: &namespaceRow{
				path:        workspace.FullPath,
				workspaceID: workspace.Metadata.ID,
				version:     initialResourceVersion,
			},
		})
	}

	// Negative cases for paths that do not exist.
	nonExistPaths := []string{
		"non-exist-top-level",
		"top-level-group-0-for-namespaces/non-exist-sub-path",
		"top-level-group-1-for-namespaces/workspace-2/non-exist-below-workspace",
	}
	for _, nonExistPath := range nonExistPaths {
		testCases = append(testCases,
			testCase{
				name:  "negative: non-exist--" + nonExistPath,
				input: nonExistPath,
			},
		)
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			gotNamespace, err := getNamespaceByPath(ctx, testClient.client.getConnection(ctx), test.input)

			checkError(t, test.expectMsg, err)

			if test.expectNamespace != nil {
				require.NotNil(t, gotNamespace)
				compareNamespaceRows(t, test.expectNamespace, gotNamespace)
			} else {
				assert.Nil(t, gotNamespace)
			}

		})
	}
}

func TestCreateNamespace(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdWarmupOutput, err := createWarmupNamespaces(ctx, testClient, &namespaceWarmupsInput{
		groups:     standardWarmupGroupsForNamespaces,
		workspaces: standardWarmupWorkspacesForNamespaces,
	})
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup objects weren't all created.
		return
	}

	type testCase struct {
		input           *namespaceRow
		expectMsg       *string
		expectNamespace *namespaceRow
		name            string
	}

	/*
		template test case:

		{
		name            string
		input           *namespaceRow
		expectMsg       *string
		expectNamespace *namespaceRow
		}
	*/

	testCases := []testCase{}

	// It is not feasible to make a direct positive test case here, because the ID fields
	// would have to match existing groups and workspaces.  The other test functions
	// in this module serve as indirect tests of createNamespace.

	testCases = append(testCases,

		testCase{
			name: "negative: duplicate group",
			input: &namespaceRow{
				path:    "would/duplicate/a/group",
				groupID: createdWarmupOutput.groups[0].Metadata.ID,
			},
			expectMsg: ptr.String("namespace would/duplicate/a/group already exists"),
		},

		testCase{
			name: "negative: duplicate workspace",
			input: &namespaceRow{
				path:        "would/duplicate/a/workspace",
				workspaceID: createdWarmupOutput.workspaces[0].Metadata.ID,
			},
			expectMsg: ptr.String("namespace would/duplicate/a/workspace already exists"),
		},

		testCase{
			name: "negative: non-exist group ID",
			input: &namespaceRow{
				path:    "group/ID/does/not/exist",
				groupID: nonExistentID,
			},
			expectMsg: ptr.String("ERROR: insert or update on table \"namespaces\" violates foreign key constraint \"fk_group_id\" (SQLSTATE 23503)"),
		},

		testCase{
			name: "negative: non-exist workspace ID",
			input: &namespaceRow{
				path:        "workspace/ID/does/not/exist",
				workspaceID: nonExistentID,
			},
			expectMsg: ptr.String("ERROR: insert or update on table \"namespaces\" violates foreign key constraint \"fk_workspace_id\" (SQLSTATE 23503)"),
		},
		testCase{
			name: "negative: invalid group ID",
			input: &namespaceRow{
				path:    "group/ID/invalid",
				groupID: invalidID,
			},
			expectMsg: invalidUUIDMsg1,
		},

		testCase{
			name: "negative: invalid workspace ID",
			input: &namespaceRow{
				path:        "workspace/ID/invalid",
				workspaceID: invalidID,
			},
			expectMsg: invalidUUIDMsg1,
		},
	)

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			gotNamespace, err := createNamespace(ctx, testClient.client.getConnection(ctx), test.input)

			checkError(t, test.expectMsg, err)

			if test.expectNamespace != nil {
				require.NotNil(t, gotNamespace)
				compareNamespaceRows(t, test.expectNamespace, gotNamespace)
			} else {
				assert.Nil(t, gotNamespace)
			}

		})
	}
}

// TestMigrateNamespace tests the namespace module's migrateNamespaces method,
// which is called by group modules MigrateGroup method before actually re-parenting the group.
func TestMigrateNamespace(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupOutput, err := createWarmupNamespaces(ctx, testClient, &namespaceWarmupsInput{
		groups:     append(standardWarmupGroupsForNamespaces, warmupGroupsForNamespaceMigration...),
		workspaces: append(standardWarmupWorkspacesForNamespaces, warmupWorkspacesForNamespaceMigration...),
	})
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup objects weren't all created.
		return
	}

	type testCase struct {
		expectMsg  *string
		pathChecks *pathChecksType
		name       string
		oldPath    string
		newPath    string
	}

	/*
		template test case:

		{
		name       string
		oldPath    string
		newPath    string
		expectMsg  *string
		pathChecks *pathChecksType
		}
	*/

	// Because each test case starts with the results of the previous case,
	// exceptions accumulate from case to case.
	testCases := []testCase{}
	testCases = append(testCases,

		testCase{
			name:       "null move",
			oldPath:    "top-level-group-0-for-namespaces",
			newPath:    "top-level-group-0-for-namespaces",
			pathChecks: buildPathChecks(warmupOutput, &pathChecksType{}),
		},

		testCase{
			name:    "root to root",
			oldPath: "top-level-group-3-for-nothing",
			newPath: "migrated-group-3",
			pathChecks: buildPathChecks(warmupOutput, &pathChecksType{
				groups: map[models.Group]string{
					warmupOutput.groups[3]: "migrated-group-3",
					warmupOutput.groups[8]: "migrated-group-3/2nd-level-group-30",
				},
				workspaces: map[models.Workspace]string{
					warmupOutput.workspaces[9]: "migrated-group-3/2nd-level-group-30/workspace-30x",
				},
			}),
		},

		testCase{
			name:    "root to lower",
			oldPath: "migrated-group-3",
			newPath: "top-level-group-0-for-namespaces/double-migrated-group-3",
			pathChecks: buildPathChecks(warmupOutput, &pathChecksType{
				groups: map[models.Group]string{
					warmupOutput.groups[3]: "top-level-group-0-for-namespaces/double-migrated-group-3",
					warmupOutput.groups[8]: "top-level-group-0-for-namespaces/double-migrated-group-3/2nd-level-group-30",
				},
				workspaces: map[models.Workspace]string{
					warmupOutput.workspaces[9]: "top-level-group-0-for-namespaces/double-migrated-group-3/2nd-level-group-30/workspace-30x",
				},
			}),
		},

		testCase{
			name:    "lower 10 to root",
			oldPath: "top-level-group-1-for-namespaces/2nd-level-group-10",
			newPath: "migrated-2nd-level-group-10-now-root",
			pathChecks: buildPathChecks(warmupOutput, &pathChecksType{
				groups: map[models.Group]string{
					warmupOutput.groups[3]: "top-level-group-0-for-namespaces/double-migrated-group-3",
					warmupOutput.groups[8]: "top-level-group-0-for-namespaces/double-migrated-group-3/2nd-level-group-30",
					warmupOutput.groups[4]: "migrated-2nd-level-group-10-now-root",
					warmupOutput.groups[5]: "migrated-2nd-level-group-10-now-root/3rd-level-group-100",
				},
				workspaces: map[models.Workspace]string{
					warmupOutput.workspaces[9]: "top-level-group-0-for-namespaces/double-migrated-group-3/2nd-level-group-30/workspace-30x",
					warmupOutput.workspaces[5]: "migrated-2nd-level-group-10-now-root/workspace-10x",
					warmupOutput.workspaces[6]: "migrated-2nd-level-group-10-now-root/3rd-level-group-100/workspace-100x",
				},
			}),
		},

		testCase{
			name:    "lower 20 to lower",
			oldPath: "top-level-group-2-for-namespaces/2nd-level-group-20",
			newPath: "top-level-group-1-for-namespaces/2nd-level-group-20",
			pathChecks: buildPathChecks(warmupOutput, &pathChecksType{
				groups: map[models.Group]string{
					warmupOutput.groups[3]: "top-level-group-0-for-namespaces/double-migrated-group-3",
					warmupOutput.groups[8]: "top-level-group-0-for-namespaces/double-migrated-group-3/2nd-level-group-30",
					warmupOutput.groups[4]: "migrated-2nd-level-group-10-now-root",
					warmupOutput.groups[5]: "migrated-2nd-level-group-10-now-root/3rd-level-group-100",
					warmupOutput.groups[6]: "top-level-group-1-for-namespaces/2nd-level-group-20",
					warmupOutput.groups[7]: "top-level-group-1-for-namespaces/2nd-level-group-20/3rd-level-group-200",
				},
				workspaces: map[models.Workspace]string{
					warmupOutput.workspaces[9]: "top-level-group-0-for-namespaces/double-migrated-group-3/2nd-level-group-30/workspace-30x",
					warmupOutput.workspaces[5]: "migrated-2nd-level-group-10-now-root/workspace-10x",
					warmupOutput.workspaces[6]: "migrated-2nd-level-group-10-now-root/3rd-level-group-100/workspace-100x",
					warmupOutput.workspaces[7]: "top-level-group-1-for-namespaces/2nd-level-group-20/workspace-20x",
					warmupOutput.workspaces[8]: "top-level-group-1-for-namespaces/2nd-level-group-20/3rd-level-group-200/workspace-200x",
				},
			}),
		},

		// There are no other negative test cases here, because all the checks are done in earlier methods/functions.
	)

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			err := migrateNamespaces(ctx, testClient.client.getConnection(ctx), test.oldPath, test.newPath)

			checkError(t, test.expectMsg, err)

			if test.pathChecks != nil {
				for g1, expectPath := range test.pathChecks.groups {
					// Must fetch the group by ID to get the updated full path.
					g2, err := testClient.client.Groups.GetGroupByID(ctx, g1.Metadata.ID)
					require.Nil(t, err)
					assert.Equal(t, expectPath, g2.FullPath)
				}
				for w1, expectPath := range test.pathChecks.workspaces {
					// Must fetch the workspace by ID to get the updated full path.
					w2, err := testClient.client.Workspaces.GetWorkspaceByID(ctx, w1.Metadata.ID)
					require.Nil(t, err)
					assert.Equal(t, expectPath, w2.FullPath)
				}
			}
		})
	}
}

//////////////////////////////////////////////////////////////////////////////

// Common utility structures and functions:

// Standard warmup groups for tests in this module:
// The create function will derive the parent path and name from the full path.
var standardWarmupGroupsForNamespaces = []models.Group{
	{
		Description: "top level group 0 for testing namespace functions",
		FullPath:    "top-level-group-0-for-namespaces",
		CreatedBy:   "someone-1",
	},
	{
		Description: "top level group 1 for testing namespace functions",
		FullPath:    "top-level-group-1-for-namespaces",
		CreatedBy:   "someone-2",
	},
	{
		Description: "top level group 2 for testing namespace functions",
		FullPath:    "top-level-group-2-for-namespaces",
		CreatedBy:   "someone-3",
	},
	{
		Description: "top level group 3 for nothing",
		FullPath:    "top-level-group-3-for-nothing",
		CreatedBy:   "someone-4",
	},
}

// Semi-standard warmup groups for migration testing:
// The create function will derive the parent path and name from the full path.
var warmupGroupsForNamespaceMigration = []models.Group{
	{
		Description: "2nd-level group 10",
		FullPath:    "top-level-group-1-for-namespaces/2nd-level-group-10",
		CreatedBy:   "someone-10",
	},
	{
		Description: "3rd-level group 100",
		FullPath:    "top-level-group-1-for-namespaces/2nd-level-group-10/3rd-level-group-100",
		CreatedBy:   "someone-100",
	},
	{
		Description: "2nd-level group 20",
		FullPath:    "top-level-group-2-for-namespaces/2nd-level-group-20",
		CreatedBy:   "someone-20",
	},
	{
		Description: "3rd-level group 200",
		FullPath:    "top-level-group-2-for-namespaces/2nd-level-group-20/3rd-level-group-200",
		CreatedBy:   "someone-200",
	},
	{
		Description: "2nd-level group 30",
		FullPath:    "top-level-group-3-for-nothing/2nd-level-group-30",
		CreatedBy:   "someone-30",
	},
}

// Standard warmup workspaces for tests in this module:
// Make sure the order in this slice is _NOT_ exactly full path or name order.
// The create function will derive the group ID and name from the full path.
var standardWarmupWorkspacesForNamespaces = []models.Workspace{
	{
		Description: "workspace 1 for testing namespace functions",
		FullPath:    "top-level-group-0-for-namespaces/workspace-1",
		CreatedBy:   "someone-1",
	},
	{
		Description: "workspace 5 for testing namespace functions",
		FullPath:    "top-level-group-1-for-namespaces/workspace-5",
		CreatedBy:   "someone-6",
	},
	{
		Description: "workspace 3 for testing namespace functions",
		FullPath:    "top-level-group-2-for-namespaces/workspace-3",
		CreatedBy:   "someone-5",
	},
	{
		Description: "workspace 4 for testing namespace functions",
		FullPath:    "top-level-group-0-for-namespaces/workspace-4",
		CreatedBy:   "someone-3",
	},
	{
		Description: "workspace 2 for testing namespace functions",
		FullPath:    "top-level-group-1-for-namespaces/workspace-2",
		CreatedBy:   "someone-2",
	},
}

// Semi-standard warmup workspaces for group migration testing:
// The create function will derive the group ID and name from the full path.
var warmupWorkspacesForNamespaceMigration = []models.Workspace{
	{
		Description: "workspace under 2nd-level group 10",
		FullPath:    "top-level-group-1-for-namespaces/2nd-level-group-10/workspace-10x",
		CreatedBy:   "someone-10x",
	},
	{
		Description: "workspace under 3rd-level group 100",
		FullPath:    "top-level-group-1-for-namespaces/2nd-level-group-10/3rd-level-group-100/workspace-100x",
		CreatedBy:   "someone-100x",
	},
	{
		Description: "workspace under 2nd-level group 20",
		FullPath:    "top-level-group-2-for-namespaces/2nd-level-group-20/workspace-20x",
		CreatedBy:   "someone-20x",
	},
	{
		Description: "workspace under 3rd-level group 200",
		FullPath:    "top-level-group-2-for-namespaces/2nd-level-group-20/3rd-level-group-200/workspace-200x",
		CreatedBy:   "someone-200x",
	},
	{
		Description: "workspace under 2nd-level group 30",
		FullPath:    "top-level-group-3-for-nothing/2nd-level-group-30/workspace-30x",
		CreatedBy:   "someone-30x",
	},
}

// createWarmupNamespaces creates some warmup groups and workspaces
// and thus their associated namespaces for a test.
// The warmup groups and workspaces to create can be standard or otherwise.
//
// NOTE: Due to the need to supply the parent ID for non-top-level groups,
// the groups must be created in a top-down manner.
func createWarmupNamespaces(ctx context.Context, testClient *testClient,
	input *namespaceWarmupsInput) (*namespaceWarmupsOutput, error) {

	resultGroups, parentPath2ID, err := createInitialGroups(ctx, testClient, input.groups)
	if err != nil {
		return nil, err
	}

	groupMap := make(map[string]string)
	for _, group := range resultGroups {
		groupMap[group.Metadata.ID] = group.FullPath
	}

	resultWorkspaces, err := createInitialWorkspaces(ctx, testClient, parentPath2ID, input.workspaces)
	if err != nil {
		return nil, err
	}

	workspaceMap := make(map[string]string)
	for _, workspace := range resultWorkspaces {
		workspaceMap[workspace.Metadata.ID] = workspace.FullPath
	}

	return &namespaceWarmupsOutput{
		groups:           resultGroups,
		workspaces:       resultWorkspaces,
		groupID2Path:     groupMap,
		workspaceID2Path: workspaceMap,
	}, nil
}

// buildPathChecks builds a pathChecksType struct from a namespaceWarmupsOutput and a block of exceptions.
func buildPathChecks(base *namespaceWarmupsOutput, exceptions *pathChecksType) *pathChecksType {
	result := pathChecksType{
		groups:     map[models.Group]string{},
		workspaces: map[models.Workspace]string{},
	}

	// Build the base.
	for _, g := range base.groups {
		result.groups[g] = g.FullPath
	}
	for _, w := range base.workspaces {
		result.workspaces[w] = w.FullPath
	}

	// Apply the exceptions.
	for g, p := range exceptions.groups {
		result.groups[g] = p
	}
	for w, p := range exceptions.workspaces {
		result.workspaces[w] = p
	}

	return &result
}

// compareNamespaceRows compares two namespace row objects.
// Because there's no way to find the expected ID, it cannot be checked.
func compareNamespaceRows(t *testing.T, expected, actual *namespaceRow) {
	assert.Equal(t, expected.path, actual.path)
	assert.Equal(t, expected.groupID, actual.groupID)
	assert.Equal(t, expected.workspaceID, actual.workspaceID)
	assert.Equal(t, expected.version, actual.version)
}

// The End.
