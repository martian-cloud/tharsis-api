//go:build integration

package db

import (
	"cmp"
	"context"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/doug-martin/goqu/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

const (

	// a contrived bogus ID
	nonExistentID = "12345678-1234-1234-1234-123456789abc"

	// a contrived bogus global ID used for testing TRN functions
	nonExistentGlobalID = "QV84MWI0NDY4OS0wNGExLTRkYTQtOTY1Mi0zYmY4OWE1ZGJkMzU"

	// an invalid ID
	invalidID = "not-a-valid-uuid"

	// must set max job duration when creating a workspace
	forTestMaxJobDuration = time.Hour * 12

	// Maximum number of DB connections--intended to mirror the CI environment
	maxConns = 4
)

var (

	// Used by NewClient to build the DB URI:
	// The script that runs the tests should set these via build flags.
	TestDBHost string
	TestDBPort string // contents of string must be numeric
	TestDBName string
	TestDBMode string
	TestDBUser string
	TestDBPass string

	// constants that cannot be constants

	// Map of names of tables that are excluded from being truncated.
	nonTruncateTables = map[string]interface{}{
		"schema_migrations": nil,
		"resource_limits":   nil,
	}

	// returned for resource version mismatch (or what the DB layer thinks is a version mismatch)
	resourceVersionMismatch = ptr.String(ErrOptimisticLockError.Error())

	// returned for some invalid UUID cases
	invalidUUIDMsg = ptr.String("invalid input syntax for type uuid")
)

type testClient struct {
	logger logger.Logger
	client *Client
}

// time bounds for comparing object metadata
type timeBounds struct {
	createLow  *time.Time
	createHigh *time.Time
	updateLow  *time.Time
	updateHigh *time.Time
}

// newTestClient creates a new DB client to use for the DB integration tests.
// It also wipes all tables empty.
// Based on environment variables, the client could be for a local standalone DB server
// or one created inside the CI/CD pipeline.
func newTestClient(ctx context.Context, t *testing.T) *testClient {
	portNum, err := strconv.Atoi(TestDBPort)
	if err != nil {
		t.Fatal(err)
	}

	logger, _ := logger.NewForTest()

	client, err := NewClient(ctx, TestDBHost, portNum, TestDBName, TestDBMode, TestDBUser, TestDBPass, maxConns, true, logger)
	if err != nil {
		t.Fatal(err)
	}

	result := testClient{
		client: client,
		logger: logger,
	}

	err = result.wipeAllTables(ctx)
	if err != nil {
		t.Fatal(err)
	}

	return &result
}

// close closes the test client but does not terminate the local server
func (tc *testClient) close(ctx context.Context) {
	tc.client.Close(ctx)
}

func (tc *testClient) wipeAllTables(ctx context.Context) error {
	conn := tc.client.getConnection(ctx)

	// Get the names of all tables to wipe.  Sort them to ensure deterministic behavior.
	query := dialect.From(goqu.T("pg_tables")).
		Select("tablename").
		Where(goqu.I("schemaname").Eq("public")).
		Order(goqu.I("tablename").Asc())

	sql, _, err := query.ToSQL()
	if err != nil {
		return err
	}

	rows, err := conn.Query(ctx, sql)
	if err != nil {
		return err
	}
	defer rows.Close()

	model := struct {
		tableName string
	}{}
	fields := []interface{}{
		&model.tableName,
	}
	tableNames := []interface{}{}
	for rows.Next() {
		err = rows.Scan(fields...)
		if err != nil {
			return err
		}
		// Exclude special tables from being wiped.
		if _, ok := nonTruncateTables[model.tableName]; !ok {
			tableNames = append(tableNames, model.tableName)
		}
	}

	if len(tableNames) == 0 {
		return fmt.Errorf("function wipeAllTables found no tables to truncate")
	}

	// Wipe all the tables.
	query2 := dialect.Truncate(tableNames...)

	sql2, _, err := query2.ToSQL()
	if err != nil {
		return err
	}

	_, err = conn.Exec(ctx, sql2)
	if err != nil {
		return err
	}

	return nil
}

type sortableField interface {
	getFieldDescriptor() *pagination.FieldDescriptor
	getSortDirection() pagination.SortDirection
	getValue() string
}

// testCursorPageWalks checks cursor pagination against the unpaginated order for one sort. It walks every
// page forward (First/After) and backward (Last/Before), and steps back one page from the middle
// (First/Before and Last/After), expecting the rows immediately adjacent to the cursor each time. The page
// size is smaller than the remaining rows so a query that returns rows from the wrong end of the list, or in
// the wrong order, is detected. Resources are identified by their cursor, which is unique per row.
func testCursorPageWalks(
	ctx context.Context,
	t *testing.T,
	totalCount int,
	sortByField sortableField,
	getResourcesFunc func(ctx context.Context, sortByField sortableField, paginationOptions *pagination.Options) (*pagination.PageInfo, []pagination.CursorPaginatable, error),
) {
	t.Helper()

	const pageSize = 2
	if totalCount < 2*pageSize+1 {
		return
	}

	getPage := func(opts *pagination.Options) (*pagination.PageInfo, []string, []*string) {
		t.Helper()
		pageInfo, resources, err := getResourcesFunc(ctx, sortByField, opts)
		require.Nil(t, err)

		ids := []string{}
		cursors := []*string{}
		for _, resource := range resources {
			c, err := pageInfo.Cursor(resource)
			require.Nil(t, err)
			ids = append(ids, *c)
			cursors = append(cursors, c)
		}
		return pageInfo, ids, cursors
	}

	reversed := func(s []string) []string {
		out := make([]string, 0, len(s))
		for i := len(s) - 1; i >= 0; i-- {
			out = append(out, s[i])
		}
		return out
	}

	_, expected, expectedCursors := getPage(&pagination.Options{})
	require.Len(t, expected, totalCount)

	// Forward walk.
	walked := []string{}
	var cursor *string
	for range totalCount + 1 {
		pageInfo, ids, cursors := getPage(&pagination.Options{First: ptr.Int32(pageSize), After: cursor})
		walked = append(walked, ids...)
		if len(ids) == 0 || !pageInfo.HasNextPage {
			break
		}
		cursor = cursors[len(cursors)-1]
	}
	assert.Equal(t, expected, walked, "forward walk with sort by %s", sortByField.getValue())

	// Backward walk. Last returns rows in reverse order, so the walk visits the list back to front.
	walked = []string{}
	cursor = nil
	for range totalCount + 1 {
		pageInfo, ids, cursors := getPage(&pagination.Options{Last: ptr.Int32(pageSize), Before: cursor})
		walked = append(walked, ids...)
		if len(ids) == 0 || !pageInfo.HasPreviousPage {
			break
		}
		cursor = cursors[len(cursors)-1]
	}
	assert.Equal(t, expected, reversed(walked), "backward walk with sort by %s", sortByField.getValue())

	// Step back one page from the middle in each direction.
	middle := totalCount / 2

	pageInfo, ids, _ := getPage(&pagination.Options{First: ptr.Int32(pageSize), Before: expectedCursors[middle]})
	assert.Equal(t, expected[middle-pageSize:middle], ids, "first before with sort by %s", sortByField.getValue())
	assert.True(t, pageInfo.HasNextPage, "first before with sort by %s", sortByField.getValue())
	assert.Equal(t, middle-pageSize > 0, pageInfo.HasPreviousPage, "first before with sort by %s", sortByField.getValue())

	pageInfo, ids, _ = getPage(&pagination.Options{Last: ptr.Int32(pageSize), After: expectedCursors[middle]})
	assert.Equal(t, reversed(expected[middle+1:middle+1+pageSize]), ids, "last after with sort by %s", sortByField.getValue())
	assert.True(t, pageInfo.HasPreviousPage, "last after with sort by %s", sortByField.getValue())
	assert.Equal(t, middle+1+pageSize < totalCount, pageInfo.HasNextPage, "last after with sort by %s", sortByField.getValue())
}

func testResourcePaginationAndSorting(
	ctx context.Context,
	t *testing.T,
	totalCount int,
	sortableFields []sortableField,
	getResourcesFunc func(ctx context.Context, sortByField sortableField, paginationOptions *pagination.Options) (*pagination.PageInfo, []pagination.CursorPaginatable, error),
) {
	/* Test pagination in forward direction */
	defaultSortBy := sortableFields[0]

	middleIndex := totalCount / 2
	pageInfo, resources, err := getResourcesFunc(ctx, defaultSortBy, &pagination.Options{
		First: ptr.Int32(int32(middleIndex)),
	})
	require.Nil(t, err)

	assert.Equal(t, middleIndex, len(resources))
	assert.True(t, pageInfo.HasNextPage)
	assert.False(t, pageInfo.HasPreviousPage)

	cursor, err := pageInfo.Cursor(resources[len(resources)-1])
	require.Nil(t, err)

	remaining := totalCount - middleIndex
	pageInfo, resources, err = getResourcesFunc(ctx, defaultSortBy, &pagination.Options{
		First: ptr.Int32(int32(remaining)),
		After: cursor,
	})
	require.Nil(t, err)

	assert.Equal(t, remaining, len(resources))
	assert.True(t, pageInfo.HasPreviousPage)
	assert.False(t, pageInfo.HasNextPage)

	/* Test pagination in reverse direction */

	pageInfo, resources, err = getResourcesFunc(ctx, defaultSortBy, &pagination.Options{
		Last: ptr.Int32(int32(middleIndex)),
	})
	require.Nil(t, err)

	assert.Equal(t, middleIndex, len(resources))
	assert.False(t, pageInfo.HasNextPage)
	assert.True(t, pageInfo.HasPreviousPage)

	cursor, err = pageInfo.Cursor(resources[len(resources)-1])
	require.Nil(t, err)

	remaining = totalCount - middleIndex
	pageInfo, resources, err = getResourcesFunc(ctx, defaultSortBy, &pagination.Options{
		Last:   ptr.Int32(int32(remaining)),
		Before: cursor,
	})
	require.Nil(t, err)

	assert.Equal(t, remaining, len(resources))
	assert.False(t, pageInfo.HasPreviousPage)
	assert.True(t, pageInfo.HasNextPage)

	/* Test walking every page in both directions, and stepping back a page, for each sort */
	for _, sortByField := range sortableFields {
		testCursorPageWalks(ctx, t, totalCount, sortByField, getResourcesFunc)
	}

	/* Test sorting */
	for _, sortByField := range sortableFields {
		_, resources, err = getResourcesFunc(ctx, sortByField, &pagination.Options{})
		require.Nil(t, err)

		values := []*string{}
		for _, resource := range resources {
			value, err := resource.ResolveMetadata(sortByField.getFieldDescriptor().Key)
			require.Nil(t, err)
			values = append(values, value)
		}

		// Must detect whether values are iso8601 timestamps, which are similar enough to RFC 3339 to use that layout.
		// If they are, must convert them and sort as time.Time rather than as strings.
		// That is because truncated trailing zeros cause string comparison to be different vs. time value comparison.
		areAllTimestamps := true
		timeValues := []*time.Time{}
		for _, value := range values {
			if value != nil {
				tv, err := time.Parse(time.RFC3339, *value)
				if err != nil {
					areAllTimestamps = false
					break
				}
				timeValues = append(timeValues, &tv)
			} else {
				timeValues = append(timeValues, nil)
			}
		}

		if areAllTimestamps {
			// Time value sort/comparison.
			expectedTimes := []*time.Time{}
			expectedTimes = append(expectedTimes, timeValues...)

			slices.SortFunc(expectedTimes, func(a, b *time.Time) int {
				if val, ok := cmpNils(a, b, sortByField.getSortDirection()); ok {
					return val
				}
				if sortByField.getSortDirection() == pagination.AscSort {
					return int(a.Sub(*b)) // positive if a is later/greater than g
				}
				return int(b.Sub(*a)) // positive if b is later/greater than a
			})

			assert.Equal(t, expectedTimes, timeValues, "resources are not sorted correctly when using sort by %s", sortByField.getValue())
		} else {
			// Ordinary string sort/comparison.
			expectedValues := []*string{}
			expectedValues = append(expectedValues, values...)

			slices.SortFunc(expectedValues, func(a, b *string) int {
				if val, ok := cmpNils(a, b, sortByField.getSortDirection()); ok {
					return val
				}
				if sortByField.getSortDirection() == pagination.AscSort {
					return cmp.Compare(*a, *b)
				}
				return cmp.Compare(*b, *a)
			})

			assert.Equal(t, expectedValues, values, "resources are not sorted correctly when using sort by %s", sortByField.getValue())
		}
	}
}

//////////////////////////////////////////////////////////////////////////////

// Other utility function(s):

// Compare one actual time vs. an expect interval.
// Use the negative sense, because we want >= and <=, while time gives us > and <.
func compareTime(t *testing.T, expectedLow, expectedHigh, actual *time.Time) {
	assert.False(t, actual.Before(*expectedLow))
	assert.False(t, actual.After(*expectedHigh))
}

// Check whether any actual error contains what was expected.
// If an error was expected but did not occur, the test is terminated, and
// any subsequent test cases that should have run will not be attempted.
func checkError(t *testing.T, expectedMsg *string, actualError error) {
	if expectedMsg == nil {
		assert.Nil(t, actualError)
	} else {
		// Uses require rather than assert to avoid a nil pointer dereference.
		require.NotNil(t, actualError)
		assert.Contains(t, actualError.Error(), *expectedMsg)
	}
}

func cmpNils(a, b any, sortDir pagination.SortDirection) (int, bool) {
	aIsNil := a == nil || reflect.ValueOf(a).IsNil()
	bIsNil := b == nil || reflect.ValueOf(b).IsNil()

	if aIsNil && bIsNil {
		return 0, true
	}

	// If one of the values is nil, then it is greater than the other

	if sortDir == pagination.AscSort {
		if aIsNil {
			return 1, true
		} else if bIsNil {
			return -1, true
		}
	}
	if aIsNil {
		return -1, true
	} else if bIsNil {
		return 1, true
	}
	return 0, false
}

// The End.
