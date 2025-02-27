//go:build integration

package db

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// Some constants and pseudo-constants are declared/defined in dbclient_test.go.

// variableInfo aids convenience in accessing the information
// TestGetVariables needs about the warmup variables.
type variableInfo struct {
	id            string
	key           string
	createTime    time.Time
	namespacePath string
}

// variableInfoIDSlice makes a slice of variableInfo sortable by ID string
type variableInfoIDSlice []variableInfo

// variableInfoKeySlice makes a slice of variableInfo sortable by the key field
type variableInfoKeySlice []variableInfo

// variableInfoCreateTimeSlice makes a slice of variableInfo sortable by creation time
type variableInfoCreateTimeSlice []variableInfo

// variableInfoNamespacePathSlice makes a slice of variableInfo sortable by namespace path
type variableInfoNamespacePathSlice []variableInfo

func TestGetVariables(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupVariables, err := createWarmupVariables(ctx, testClient,
		standardWarmupGroupsForVariables, standardWarmupWorkspacesForVariables,
		standardWarmupVariables)
	require.Nil(t, err)
	allVariableInfos := variableInfoFromVariables(warmupVariables)

	// Sort by variable IDs.
	sort.Sort(variableInfoIDSlice(allVariableInfos))
	allVariableIDs := variableIDsFromVariableInfos(allVariableInfos)

	// Sort by keys.
	sort.Sort(variableInfoKeySlice(allVariableInfos))
	allVariableIDsByKey := variableIDsFromVariableInfos(allVariableInfos)
	reverseVariableIDsByKey := reverseStringSlice(allVariableIDsByKey)

	// Sort by create times.
	sort.Sort(variableInfoCreateTimeSlice(allVariableInfos))
	allVariableIDsByCreateTime := variableIDsFromVariableInfos(allVariableInfos)
	reverseVariableIDsByCreateTime := reverseStringSlice(allVariableIDsByCreateTime)

	// Sort by namespace paths.
	sort.Sort(variableInfoNamespacePathSlice(allVariableInfos))
	allVariableIDsByNamespacePath := variableIDsFromVariableInfos(allVariableInfos)

	dummyCursorFunc := func(cp pagination.CursorPaginatable) (*string, error) { return ptr.String("dummy-cursor-value"), nil }

	type testCase struct {
		expectStartCursorError      error
		expectEndCursorError        error
		input                       *GetVariablesInput
		expectMsg                   *string
		name                        string
		expectPageInfo              pagination.PageInfo
		expectVariableIDs           []string
		getBeforeCursorFromPrevious bool
		sortedDescending            bool
		expectHasStartCursor        bool
		getAfterCursorFromPrevious  bool
		expectHasEndCursor          bool
		externalFilterHcl           bool
	}

	/*
		template test case:

		{
			name: "",
			input: &GetVariablesInput{
				Sort:              nil,
				PaginationOptions: nil,
				Filter:            nil,
			},
			externalFilterHcl            bool
			sortedDescending             bool
			getBeforeCursorFromPrevious: false,
			getAfterCursorFromPrevious:  false,
			expectMsg:                   nil,
			expectVariableIDs:       []string{},
			expectPageInfo: pagination.PageInfo{
				Cursor:          nil,
				TotalCount:      0,
				HasNextPage:     false,
				HasPreviousPage: false,
			},
			expectStartCursorError: nil,
			expectHasStartCursor:   false,
			expectEndCursorError:   nil,
			expectHasEndCursor:     false,
		}
	*/

	testCases := []testCase{
		// nil input likely causes a nil pointer dereference in GetVariables, so don't try it.

		{
			name: "non-nil but mostly empty input",
			input: &GetVariablesInput{
				Sort:              nil,
				PaginationOptions: nil,
				Filter:            nil,
			},
			expectVariableIDs:    allVariableIDs,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allVariableIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "populated sort and pagination, nil filter",
			input: &GetVariablesInput{
				Sort: ptrVariableSortableField(VariableSortableFieldCreatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
				Filter: nil,
			},
			expectVariableIDs:    allVariableIDsByCreateTime,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allVariableIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "sort in ascending order of key",
			input: &GetVariablesInput{
				Sort: ptrVariableSortableField(VariableSortableFieldKeyAsc),
			},
			expectVariableIDs:    allVariableIDsByKey,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allVariableIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "sort in descending order of key",
			input: &GetVariablesInput{
				Sort: ptrVariableSortableField(VariableSortableFieldKeyDesc),
			},
			sortedDescending:     true,
			expectVariableIDs:    reverseVariableIDsByKey,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allVariableIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "sort in ascending order of time of creation",
			input: &GetVariablesInput{
				Sort: ptrVariableSortableField(VariableSortableFieldCreatedAtAsc),
			},
			expectVariableIDs:    allVariableIDsByCreateTime,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allVariableIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "sort in descending order of time of creation",
			input: &GetVariablesInput{
				Sort: ptrVariableSortableField(VariableSortableFieldCreatedAtDesc),
			},
			sortedDescending:     true,
			expectVariableIDs:    reverseVariableIDsByCreateTime,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allVariableIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		// When sorting by namespace path, must filter to one variable per namespace
		// in order to avoid ambiguity in the order of results.
		{
			name: "sort in ascending order of namespace path",
			input: &GetVariablesInput{
				Sort: ptrVariableSortableField(VariableSortableFieldNamespacePathAsc),
			},
			externalFilterHcl:    true,
			expectVariableIDs:    []string{allVariableIDsByNamespacePath[0], allVariableIDsByNamespacePath[3]},
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allVariableIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "sort in descending order of namespace path",
			input: &GetVariablesInput{
				Sort: ptrVariableSortableField(VariableSortableFieldNamespacePathDesc),
			},
			externalFilterHcl:    true,
			sortedDescending:     true,
			expectVariableIDs:    []string{allVariableIDsByNamespacePath[3], allVariableIDsByNamespacePath[0]},
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allVariableIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: everything at once",
			input: &GetVariablesInput{
				Sort: ptrVariableSortableField(VariableSortableFieldCreatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
			},
			expectVariableIDs:    allVariableIDsByCreateTime,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allVariableIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: first two",
			input: &GetVariablesInput{
				Sort: ptrVariableSortableField(VariableSortableFieldCreatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(2),
				},
			},
			expectVariableIDs: allVariableIDsByCreateTime[:2],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allVariableIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: false,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: middle three",
			input: &GetVariablesInput{
				Sort: ptrVariableSortableField(VariableSortableFieldCreatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(3),
				},
			},
			getAfterCursorFromPrevious: true,
			expectVariableIDs:          allVariableIDsByCreateTime[2:5],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allVariableIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: true,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: final one",
			input: &GetVariablesInput{
				Sort: ptrVariableSortableField(VariableSortableFieldCreatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
			},
			getAfterCursorFromPrevious: true,
			expectVariableIDs:          allVariableIDsByCreateTime[5:],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allVariableIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     false,
				HasPreviousPage: true,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		// When Last is supplied, the sort order is intended to be reversed.
		{
			name: "pagination: last three",
			input: &GetVariablesInput{
				Sort: ptrVariableSortableField(VariableSortableFieldCreatedAtAsc),
				PaginationOptions: &pagination.Options{
					Last: ptr.Int32(3),
				},
			},
			sortedDescending:  true,
			expectVariableIDs: reverseVariableIDsByCreateTime[:3],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allVariableIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     false,
				HasPreviousPage: true,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		/*

			The input.PaginationOptions.After field is tested earlier via getAfterCursorFromPrevious.

			The input.PaginationOptions.Before field is not really supported and does not work.
			If it did work, it could be tested by adapting the test cases corresponding to the
			next few cases after a similar block of text from group_test.go

		*/

		{
			name: "pagination, before and after, expect error",
			input: &GetVariablesInput{
				Sort:              ptrVariableSortableField(VariableSortableFieldCreatedAtAsc),
				PaginationOptions: &pagination.Options{},
			},
			getAfterCursorFromPrevious:  true,
			getBeforeCursorFromPrevious: true,
			expectMsg:                   ptr.String("failed to create paginated query builder: only before or after can be defined, not both"),
			expectVariableIDs:           []string{},
			expectPageInfo:              pagination.PageInfo{},
		},

		{
			name: "pagination, first one and last two, expect error",
			input: &GetVariablesInput{
				Sort: ptrVariableSortableField(VariableSortableFieldCreatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(1),
					Last:  ptr.Int32(2),
				},
			},
			expectMsg:         ptr.String("failed to create paginated query builder: only first or last can be defined, not both"),
			expectVariableIDs: allVariableIDs[4:],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allVariableIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: false,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		// If there were more filter fields, this case would allow nothing through the filter.
		{
			name: "fully-populated types, everything allowed through filters",
			input: &GetVariablesInput{
				Sort: ptrVariableSortableField(VariableSortableFieldCreatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
				Filter: &VariableFilter{
					// Passing an empty slice to NamespacePaths likely causes an SQL syntax error ("... IN ()"), so don't try it.
					// NamespacePaths: []string{},
				},
			},
			expectVariableIDs: allVariableIDsByCreateTime,
			expectPageInfo: pagination.PageInfo{
				TotalCount: int32(len(allVariableIDs)),
				Cursor:     dummyCursorFunc,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, namespace paths, positive",
			input: &GetVariablesInput{
				Sort: ptrVariableSortableField(VariableSortableFieldCreatedAtAsc),
				Filter: &VariableFilter{
					NamespacePaths: []string{warmupVariables[0].NamespacePath},
				},
			},
			expectVariableIDs: []string{
				allVariableIDsByCreateTime[0], allVariableIDsByCreateTime[1], allVariableIDsByCreateTime[2],
			},
			expectPageInfo:       pagination.PageInfo{TotalCount: 3, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, namespace paths, non-existent",
			input: &GetVariablesInput{
				Sort: ptrVariableSortableField(VariableSortableFieldCreatedAtAsc),
				Filter: &VariableFilter{
					NamespacePaths: []string{"this-path-does-not-exist"},
				},
			},
			expectVariableIDs:    []string{},
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(0), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, variable IDs, positive",
			input: &GetVariablesInput{
				Sort: ptrVariableSortableField(VariableSortableFieldCreatedAtAsc),
				Filter: &VariableFilter{
					VariableIDs: []string{warmupVariables[0].Metadata.ID},
				},
			},
			expectVariableIDs:    []string{allVariableIDsByCreateTime[0]},
			expectPageInfo:       pagination.PageInfo{TotalCount: 1, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, variable IDs, non-existent",
			input: &GetVariablesInput{
				Sort: ptrVariableSortableField(VariableSortableFieldCreatedAtAsc),
				Filter: &VariableFilter{
					VariableIDs: []string{nonExistentID},
				},
			},
			expectVariableIDs:    []string{},
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(0), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},
		{
			name: "filter, variable IDs, invalid",
			input: &GetVariablesInput{
				Sort: ptrVariableSortableField(VariableSortableFieldCreatedAtAsc),
				Filter: &VariableFilter{
					VariableIDs: []string{invalidID},
				},
			},
			expectMsg:            invalidUUIDMsg2,
			expectVariableIDs:    []string{},
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(0), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},
	}

	var (
		previousEndCursorValue   *string
		previousStartCursorValue *string
	)
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			// For some pagination tests, a previous case's cursor value gets piped into the next case.
			if test.getAfterCursorFromPrevious || test.getBeforeCursorFromPrevious {

				// Make sure there's a place to put it.
				require.NotNil(t, test.input.PaginationOptions)

				if test.getAfterCursorFromPrevious {
					// Make sure there's a previous value to use.
					require.NotNil(t, previousEndCursorValue)
					test.input.PaginationOptions.After = previousEndCursorValue
				}

				if test.getBeforeCursorFromPrevious {
					// Make sure there's a previous value to use.
					require.NotNil(t, previousStartCursorValue)
					test.input.PaginationOptions.Before = previousStartCursorValue
				}

				// Clear the values so they won't be used twice.
				previousEndCursorValue = nil
				previousStartCursorValue = nil
			}

			variablesResult, err := testClient.client.Variables.GetVariables(ctx, test.input)

			checkError(t, test.expectMsg, err)

			// If there was no error, check the results.
			if err == nil {

				// Never returns nil if error is nil.
				require.NotNil(t, variablesResult.PageInfo)
				assert.NotNil(t, variablesResult.Variables)
				pageInfo := variablesResult.PageInfo
				variables := variablesResult.Variables

				// If specified, apply external filter, allowing only Hcl=true variables.
				// This is necessary when sorting by namespace path to avoid ambiguity.
				if test.externalFilterHcl {
					newVariables := []models.Variable{}

					for _, v := range variables {
						if v.Hcl {
							newVariables = append(newVariables, v)
						}
					}

					variables = newVariables
				}

				// Check the variables result by comparing a list of the variable IDs.
				actualVariableIDs := []string{}
				for _, variable := range variables {
					actualVariableIDs = append(actualVariableIDs, variable.Metadata.ID)
				}

				// If no sort direction was specified, sort the results here for repeatability.
				if test.input.Sort == nil {
					sort.Strings(actualVariableIDs)
				}

				assert.Equal(t, len(test.expectVariableIDs), len(actualVariableIDs))
				assert.Equal(t, test.expectVariableIDs, actualVariableIDs)

				assert.Equal(t, test.expectPageInfo.HasNextPage, pageInfo.HasNextPage)
				assert.Equal(t, test.expectPageInfo.HasPreviousPage, pageInfo.HasPreviousPage)
				assert.Equal(t, test.expectPageInfo.TotalCount, pageInfo.TotalCount)
				assert.Equal(t, test.expectPageInfo.Cursor != nil, pageInfo.Cursor != nil)

				// Compare the cursor function results only if there is at least one variable returned.
				// If there are no variables returned, there is no argument to pass to the cursor function.
				// Also, don't try to reverse engineer to compare the cursor string values.
				if len(variables) > 0 {
					resultStartCursor, resultStartCursorError := pageInfo.Cursor(&variables[0])
					resultEndCursor, resultEndCursorError := pageInfo.Cursor(&variables[len(variables)-1])
					assert.Equal(t, test.expectStartCursorError, resultStartCursorError)
					assert.Equal(t, test.expectHasStartCursor, resultStartCursor != nil)
					assert.Equal(t, test.expectEndCursorError, resultEndCursorError)
					assert.Equal(t, test.expectHasEndCursor, resultEndCursor != nil)

					// Capture the ending cursor values for the next case.
					previousEndCursorValue = resultEndCursor
					previousStartCursorValue = resultStartCursor
				}
			}
		})
	}
}

func TestGetVariableByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdLow := currentTime()
	warmupVariables, err := createWarmupVariables(ctx, testClient,
		standardWarmupGroupsForVariables, standardWarmupWorkspacesForVariables,
		standardWarmupVariables)
	createdHigh := currentTime()
	require.Nil(t, err)

	type testCase struct {
		expectMsg      *string
		expectVariable *models.Variable
		name           string
		searchID       string
	}

	positiveVariable := warmupVariables[0]
	now := time.Now()
	testCases := []testCase{
		{
			name:     "positive",
			searchID: positiveVariable.Metadata.ID,
			expectVariable: &models.Variable{
				Metadata: models.ResourceMetadata{
					ID:                positiveVariable.Metadata.ID,
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
				},
				Category:      positiveVariable.Category,
				NamespacePath: positiveVariable.NamespacePath,
				Hcl:           positiveVariable.Hcl,
				Key:           positiveVariable.Key,
				Value:         positiveVariable.Value,
			},
		},

		{
			name:     "negative, non-existent variable ID",
			searchID: nonExistentID,
			// expect variable and error to be nil
		},

		{
			name:      "defective-ID",
			searchID:  invalidID,
			expectMsg: invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualVariable, err := testClient.client.Variables.GetVariableByID(ctx, test.searchID)

			checkError(t, test.expectMsg, err)

			if test.expectVariable != nil {
				require.NotNil(t, actualVariable)
				compareVariables(t, test.expectVariable, actualVariable, false, &timeBounds{
					createLow:  &createdLow,
					createHigh: &createdHigh,
					updateLow:  &createdLow,
					updateHigh: &createdHigh,
				})
			} else {
				assert.Nil(t, actualVariable)
			}
		})
	}
}

func TestCreateVariable(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	_, err := createWarmupVariables(ctx, testClient,
		standardWarmupGroupsForVariables, standardWarmupWorkspacesForVariables,
		[]models.Variable{})
	require.Nil(t, err)

	type testCase struct {
		toCreate      *models.Variable
		expectCreated *models.Variable
		expectMsg     *string
		name          string
	}

	now := time.Now()
	testCases := []testCase{
		{
			name: "positive",
			toCreate: &models.Variable{
				Category:      models.EnvironmentVariableCategory,
				NamespacePath: "top-level-group-0-for-variables/workspace-0-for-variables",
				Hcl:           true,
				Key:           "create-positive-key",
				Value:         ptr.String("create-positive-value"),
			},
			expectCreated: &models.Variable{
				Metadata: models.ResourceMetadata{
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
				},
				Category:      models.EnvironmentVariableCategory,
				NamespacePath: "top-level-group-0-for-variables/workspace-0-for-variables",
				Hcl:           true,
				Key:           "create-positive-key",
				Value:         ptr.String("create-positive-value"),
			},
		},

		{
			name: "duplicate namespace, category, and key",
			toCreate: &models.Variable{
				Category:      models.EnvironmentVariableCategory,
				NamespacePath: "top-level-group-0-for-variables/workspace-0-for-variables",
				Hcl:           false,
				Key:           "create-positive-key",
				Value:         ptr.String("duplicate-value"),
			},
			expectMsg: ptr.String("Variable with key create-positive-key in namespace top-level-group-0-for-variables/workspace-0-for-variables already exists"),
		},

		{
			name: "negative, non-existent namespace path",
			toCreate: &models.Variable{
				Category:      models.EnvironmentVariableCategory,
				NamespacePath: "non-existent-namespace-path",
				Hcl:           true,
				Key:           "non-existent-namespace-key",
				Value:         ptr.String("non-existent-namespace-value"),
			},
			expectMsg: ptr.String("Namespace not found"),
			// expect variable and error to be nil
		},

		// It is the namespace path rather than the ID that gets passed in, so can't do an invalid UUID.

	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualCreated, err := testClient.client.Variables.CreateVariable(ctx, test.toCreate)

			checkError(t, test.expectMsg, err)

			if test.expectCreated != nil {
				require.NotNil(t, actualCreated)

				// The creation process must set the creation and last updated timestamps
				// between when the test case was created and when it the result is checked.
				whenCreated := test.expectCreated.Metadata.CreationTimestamp
				now := time.Now()

				compareVariables(t, test.expectCreated, actualCreated, false, &timeBounds{
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

func TestCreateVariables(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	_, err := createWarmupVariables(ctx, testClient,
		standardWarmupGroupsForVariables, standardWarmupWorkspacesForVariables,
		[]models.Variable{})
	require.Nil(t, err)

	// CreateVariables does not return any objects, only an error, so don't expect any returned objects.
	type testCase struct {
		expectMsg     *string
		namespacePath string
		name          string
		toCreate      []*models.Variable
	}

	testCases := []testCase{
		{
			name:          "positive",
			namespacePath: "top-level-group-0-for-variables/workspace-0-for-variables",
			toCreate: []*models.Variable{
				{
					Category: models.EnvironmentVariableCategory,
					Hcl:      true,
					Key:      "create-positive-key-0",
					Value:    ptr.String("create-positive-value-0"),
				},
				{
					Category: models.EnvironmentVariableCategory,
					Hcl:      true,
					Key:      "create-positive-key-1",
					Value:    ptr.String("create-positive-value-1"),
				},
			},
		},

		{
			name:          "external duplicate namespace, category, and key",
			namespacePath: "top-level-group-0-for-variables/workspace-0-for-variables",
			toCreate: []*models.Variable{
				{
					Category: models.EnvironmentVariableCategory,
					Hcl:      false,
					Key:      "create-positive-key-1",
					Value:    ptr.String("duplicate-value-1"),
				},
			},
			expectMsg: ptr.String("Variable with key already exists in namespace top-level-group-0-for-variables/workspace-0-for-variables"),
		},

		{
			name:          "internal duplicate namespace, category, and key",
			namespacePath: "top-level-group-0-for-variables/workspace-0-for-variables",
			toCreate: []*models.Variable{
				{
					Category: models.EnvironmentVariableCategory,
					Hcl:      false,
					Key:      "internal-duplicate-key",
					Value:    ptr.String("internal-duplicate-value"),
				},
				{
					Category: models.EnvironmentVariableCategory,
					Hcl:      false,
					Key:      "internal-duplicate-key",
					Value:    ptr.String("internal-duplicate-value"),
				},
			},
			expectMsg: ptr.String("Variable with key already exists in namespace top-level-group-0-for-variables/workspace-0-for-variables"),
		},

		{
			name:          "negative, non-existent namespace path",
			namespacePath: "non-existent-namespace-path",
			toCreate: []*models.Variable{
				{
					Category: models.EnvironmentVariableCategory,
					Hcl:      true,
					Key:      "non-existent-namespace-key",
					Value:    ptr.String("non-existent-namespace-value"),
				},
			},
			expectMsg: ptr.String("Namespace not found"),
			// expect variable and error to be nil
		},

		// It is the namespace path rather than the ID that gets passed in, so can't do an invalid UUID.

	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := testClient.client.Variables.CreateVariables(ctx, test.namespacePath, test.toCreate)

			checkError(t, test.expectMsg, err)

			// There are no returned objects to verify.
		})
	}
}

func TestUpdateVariable(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupVariables, err := createWarmupVariables(ctx, testClient,
		standardWarmupGroupsForVariables, standardWarmupWorkspacesForVariables,
		standardWarmupVariables)
	require.Nil(t, err)

	type testCase struct {
		expectMsg     *string
		toUpdate      *models.Variable
		expectUpdated *models.Variable
		name          string
	}

	// Looks up by ID and version.
	// Updates key, value, and Hcl.
	// The NamespacePath field is not updated in the DB, but the value from the argument is returned.
	positiveVariable := warmupVariables[0]
	now := time.Now()
	testCases := []testCase{
		{
			name: "positive",
			toUpdate: &models.Variable{
				Metadata: models.ResourceMetadata{
					ID:      positiveVariable.Metadata.ID,
					Version: initialResourceVersion,
				},
				Category:      "something else",
				NamespacePath: "top-level-group-0-for-variables/workspace-0-for-variables",
				Hcl:           false,
				Key:           "updated-key-0",
				Value:         ptr.String("updated-value-0"),
			},
			expectUpdated: &models.Variable{
				Metadata: models.ResourceMetadata{
					ID:                   positiveVariable.Metadata.ID,
					Version:              initialResourceVersion + 1,
					CreationTimestamp:    positiveVariable.Metadata.CreationTimestamp,
					LastUpdatedTimestamp: &now,
				},
				Category:      positiveVariable.Category,
				NamespacePath: "top-level-group-0-for-variables/workspace-0-for-variables",
				Hcl:           false,
				Key:           "updated-key-0",
				Value:         ptr.String("updated-value-0"),
			},
		},

		{
			name: "would-be-duplicate",
			toUpdate: &models.Variable{
				Metadata: models.ResourceMetadata{
					ID:      positiveVariable.Metadata.ID,
					Version: initialResourceVersion,
				},
				// Would duplicate a different variable.
				Hcl:   false,
				Key:   "key-1",
				Value: ptr.String("value-1"),
			},
			expectMsg: ptr.String("resource version does not match specified version"),
		},

		{
			name: "negative, non-existent variable ID",
			toUpdate: &models.Variable{
				Metadata: models.ResourceMetadata{
					ID:      nonExistentID,
					Version: initialResourceVersion,
				},
			},
			expectMsg: ptr.String("variable does not exist"),
		},

		{
			name: "defective-ID",
			toUpdate: &models.Variable{
				Metadata: models.ResourceMetadata{
					ID:      invalidID,
					Version: initialResourceVersion,
				},
			},
			expectMsg: invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualVariable, err := testClient.client.Variables.UpdateVariable(ctx, test.toUpdate)

			checkError(t, test.expectMsg, err)

			if test.expectUpdated != nil {
				// The creation process must set the creation and last updated timestamps
				// between when the test case was created and when it the result is checked.
				whenCreated := test.expectUpdated.Metadata.CreationTimestamp
				now := currentTime()

				require.NotNil(t, actualVariable)
				compareVariables(t, test.expectUpdated, actualVariable, false, &timeBounds{
					createLow:  whenCreated,
					createHigh: &now,
					updateLow:  whenCreated,
					updateHigh: &now,
				})
			} else {
				assert.Nil(t, actualVariable)
			}
		})
	}
}

func TestDeleteVariable(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupVariables, err := createWarmupVariables(ctx, testClient,
		standardWarmupGroupsForVariables, standardWarmupWorkspacesForVariables,
		standardWarmupVariables)
	require.Nil(t, err)

	type testCase struct {
		expectMsg *string
		toDelete  *models.Variable
		name      string
	}

	// Looks up by ID and version.
	positiveVariable := warmupVariables[0]
	testCases := []testCase{
		{
			name: "positive",
			toDelete: &models.Variable{
				Metadata: models.ResourceMetadata{
					ID:      positiveVariable.Metadata.ID,
					Version: initialResourceVersion,
				},
				Category:      "something else",
				NamespacePath: "some/other/path",
				Hcl:           false,
				Key:           "updated-key-0",
				Value:         ptr.String("updated-value-0"),
			},
		},

		{
			name: "negative, non-existent variable ID",
			toDelete: &models.Variable{
				Metadata: models.ResourceMetadata{
					ID:      nonExistentID,
					Version: initialResourceVersion,
				},
			},
			expectMsg: resourceVersionMismatch,
		},

		{
			name: "defective-ID",
			toDelete: &models.Variable{
				Metadata: models.ResourceMetadata{
					ID:      invalidID,
					Version: initialResourceVersion,
				},
			},
			expectMsg: invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := testClient.client.Variables.DeleteVariable(ctx, test.toDelete)

			checkError(t, test.expectMsg, err)
		})
	}
}

//////////////////////////////////////////////////////////////////////////////

// Common utility structures and functions:

// Standard warmup group(s) for tests in this module:
// The create function will derive the parent path and name from the full path.
var standardWarmupGroupsForVariables = []models.Group{
	{
		Description: "top level group 0 for testing variable functions",
		FullPath:    "top-level-group-0-for-variables",
		CreatedBy:   "someone-g0",
	},
}

// Standard warmup workspace(s) for tests in this module:
var standardWarmupWorkspacesForVariables = []models.Workspace{
	{
		Description: "workspace 0 for testing variable functions",
		FullPath:    "top-level-group-0-for-variables/workspace-0-for-variables",
		CreatedBy:   "someone-w0",
	},
	{
		Description: "workspace 1 for testing variable functions",
		FullPath:    "top-level-group-0-for-variables/workspace-1-for-variables",
		CreatedBy:   "someone-w1",
	},
}

// Standard warmup variables for tests in this module:
// Note: To avoid ambiguity when sorting by namespace paths, facilitate external
// filtering by setting Hcl=true for only one variable per namespace path.
var standardWarmupVariables = []models.Variable{
	{
		Category:      models.EnvironmentVariableCategory,
		NamespacePath: "top-level-group-0-for-variables/workspace-0-for-variables",
		Hcl:           true,
		Key:           "key-0",
		Value:         ptr.String("value-0"),
	},
	{
		Category:      models.EnvironmentVariableCategory,
		NamespacePath: "top-level-group-0-for-variables/workspace-0-for-variables",
		Hcl:           false,
		Key:           "key-1",
		Value:         ptr.String("value-1"),
	},
	{
		Category:      models.EnvironmentVariableCategory,
		NamespacePath: "top-level-group-0-for-variables/workspace-0-for-variables",
		Hcl:           false,
		Key:           "key-2",
		Value:         ptr.String("value-2"),
	},
	{
		Category:      models.EnvironmentVariableCategory,
		NamespacePath: "top-level-group-0-for-variables/workspace-1-for-variables",
		Hcl:           true,
		Key:           "key-3",
		Value:         ptr.String("value-3"),
	},
	{
		Category:      models.EnvironmentVariableCategory,
		NamespacePath: "top-level-group-0-for-variables/workspace-1-for-variables",
		Hcl:           false,
		Key:           "key-4",
		Value:         ptr.String("value-4"),
	},
	{
		Category:      models.EnvironmentVariableCategory,
		NamespacePath: "top-level-group-0-for-variables/workspace-1-for-variables",
		Hcl:           false,
		Key:           "key-5",
		Value:         ptr.String("value-5"),
	},
}

// createWarmupVariables creates some warmup variables for a test
// The warmup variables to create can be standard or otherwise.
func createWarmupVariables(ctx context.Context, testClient *testClient,
	newGroups []models.Group,
	newWorkspaces []models.Workspace,
	newVariables []models.Variable) (
	[]models.Variable,
	error,
) {
	// It is necessary to create at least one group, workspace, and run
	// in order to provide the necessary IDs for the variables.

	_, parentPath2ID, err := createInitialGroups(ctx, testClient, newGroups)
	if err != nil {
		return nil, err
	}

	_, err = createInitialWorkspaces(ctx, testClient, parentPath2ID, newWorkspaces)
	if err != nil {
		return nil, err
	}

	resultVariables, err := createInitialVariables(ctx, testClient, newVariables)
	if err != nil {
		return nil, err
	}

	return resultVariables, nil
}

func ptrVariableSortableField(arg VariableSortableField) *VariableSortableField {
	return &arg
}

func (wis variableInfoIDSlice) Len() int {
	return len(wis)
}

func (wis variableInfoIDSlice) Swap(i, j int) {
	wis[i], wis[j] = wis[j], wis[i]
}

func (wis variableInfoIDSlice) Less(i, j int) bool {
	return wis[i].id < wis[j].id
}

func (wis variableInfoKeySlice) Len() int {
	return len(wis)
}

func (wis variableInfoKeySlice) Swap(i, j int) {
	wis[i], wis[j] = wis[j], wis[i]
}

func (wis variableInfoKeySlice) Less(i, j int) bool {
	return wis[i].key < wis[j].key
}

func (wis variableInfoCreateTimeSlice) Len() int {
	return len(wis)
}

func (wis variableInfoCreateTimeSlice) Swap(i, j int) {
	wis[i], wis[j] = wis[j], wis[i]
}

func (wis variableInfoCreateTimeSlice) Less(i, j int) bool {
	return wis[i].createTime.Before(wis[j].createTime)
}

func (wis variableInfoNamespacePathSlice) Len() int {
	return len(wis)
}

func (wis variableInfoNamespacePathSlice) Swap(i, j int) {
	wis[i], wis[j] = wis[j], wis[i]
}

func (wis variableInfoNamespacePathSlice) Less(i, j int) bool {
	return wis[i].namespacePath < wis[j].namespacePath
}

// variableInfoFromVariables returns a slice of variableInfo, not necessarily sorted in any order.
func variableInfoFromVariables(variables []models.Variable) []variableInfo {
	result := []variableInfo{}

	for _, variable := range variables {
		result = append(result, variableInfo{
			id:            variable.Metadata.ID,
			key:           variable.Key,
			createTime:    *variable.Metadata.CreationTimestamp,
			namespacePath: variable.NamespacePath,
		})
	}

	return result
}

// variableIDsFromVariableInfos preserves order
func variableIDsFromVariableInfos(variableInfos []variableInfo) []string {
	result := []string{}
	for _, variableInfo := range variableInfos {
		result = append(result, variableInfo.id)
	}
	return result
}

// compareVariables compares two variable objects, including bounds for creation and updated times.
// If times is nil, it compares the exact metadata timestamps.
func compareVariables(t *testing.T, expected, actual *models.Variable,
	checkID bool, times *timeBounds,
) {
	assert.Equal(t, expected.Category, actual.Category)
	assert.Equal(t, expected.NamespacePath, actual.NamespacePath)
	assert.Equal(t, expected.Hcl, actual.Hcl)
	assert.Equal(t, expected.Key, actual.Key)
	assert.Equal(t, expected.Value, actual.Value)

	if checkID {
		assert.Equal(t, expected.Metadata.ID, actual.Metadata.ID)
	}
	assert.Equal(t, expected.Metadata.Version, actual.Metadata.Version)

	// Compare timestamps.
	if times != nil {
		compareTime(t, times.createLow, times.createHigh, actual.Metadata.CreationTimestamp)
		compareTime(t, times.updateLow, times.updateHigh, actual.Metadata.LastUpdatedTimestamp)
	} else {
		assert.Equal(t, expected.Metadata.CreationTimestamp, actual.Metadata.CreationTimestamp)
		assert.Equal(t, expected.Metadata.LastUpdatedTimestamp, actual.Metadata.LastUpdatedTimestamp)
	}
}
