//go:build integration

package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
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

func TestGetNamespaceByGroupID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdWarmupOutput, err := createWarmupNamespaces(ctx, testClient, &namespaceWarmupsInput{
		groups: standardWarmupGroupsForNamespaces,
	})
	require.Nil(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		expectNamespace *namespaceRow
		name            string
		input           string
	}

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
			name:            "negative: invalid",
			input:           invalidID,
			expectErrorCode: errors.EInternal,
		},
	)

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			gotNamespace, err := getNamespaceByGroupID(ctx, testClient.client.getConnection(ctx), test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

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
	require.Nil(t, err)

	// Creating the groups above is necessary in order to create the workspaces.
	// However, to make sure the groups are not inadvertently used in the test
	// execution, hide them from the rest of this test function.
	createdWarmupOutput.groups = nil

	type testCase struct {
		expectErrorCode errors.CodeType
		expectNamespace *namespaceRow
		name            string
		input           string
	}

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
			name:            "negative: invalid",
			input:           invalidID,
			expectErrorCode: errors.EInternal,
		},
	)

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			gotNamespace, err := getNamespaceByWorkspaceID(ctx, testClient.client.getConnection(ctx), test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

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
	require.Nil(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		expectNamespace *namespaceRow
		name            string
		input           string
	}

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

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

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
	require.Nil(t, err)

	type testCase struct {
		input           *namespaceRow
		expectErrorCode errors.CodeType
		expectNamespace *namespaceRow
		name            string
	}

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
			expectErrorCode: errors.EConflict,
		},

		testCase{
			name: "negative: duplicate workspace",
			input: &namespaceRow{
				path:        "would/duplicate/a/workspace",
				workspaceID: createdWarmupOutput.workspaces[0].Metadata.ID,
			},
			expectErrorCode: errors.EConflict,
		},

		testCase{
			name: "negative: non-exist group ID",
			input: &namespaceRow{
				path:    "group/ID/does/not/exist",
				groupID: nonExistentID,
			},
			expectErrorCode: errors.EInternal,
		},

		testCase{
			name: "negative: non-exist workspace ID",
			input: &namespaceRow{
				path:        "workspace/ID/does/not/exist",
				workspaceID: nonExistentID,
			},
			expectErrorCode: errors.EInternal,
		},
		testCase{
			name: "negative: invalid group ID",
			input: &namespaceRow{
				path:    "group/ID/invalid",
				groupID: invalidID,
			},
			expectErrorCode: errors.EInternal,
		},

		testCase{
			name: "negative: invalid workspace ID",
			input: &namespaceRow{
				path:        "workspace/ID/invalid",
				workspaceID: invalidID,
			},
			expectErrorCode: errors.EInternal,
		},
	)

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			gotNamespace, err := createNamespace(ctx, testClient.client.getConnection(ctx), test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectNamespace != nil {
				require.NotNil(t, gotNamespace)
				compareNamespaceRows(t, test.expectNamespace, gotNamespace)
			} else {
				assert.Nil(t, gotNamespace)
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

// createWarmupNamespaces creates some warmup groups and workspaces
// and thus their associated namespaces for a test.
// The warmup groups and workspaces to create can be standard or otherwise.
//
// NOTE: Due to the need to supply the parent ID for non-top-level groups,
// the groups must be created in a top-down manner.
func createWarmupNamespaces(ctx context.Context, testClient *testClient,
	input *namespaceWarmupsInput,
) (*namespaceWarmupsOutput, error) {
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

// compareNamespaceRows compares two namespace row objects.
// Because there's no way to find the expected ID, it cannot be checked.
func compareNamespaceRows(t *testing.T, expected, actual *namespaceRow) {
	assert.Equal(t, expected.path, actual.path)
	assert.Equal(t, expected.groupID, actual.groupID)
	assert.Equal(t, expected.workspaceID, actual.workspaceID)
	assert.Equal(t, expected.version, actual.version)
}
