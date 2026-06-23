package pagination

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/doug-martin/goqu/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination/mocks"
)

type testModel struct {
	id   string
	name string
}

func (t testModel) ResolveMetadata(key string) (*string, error) {
	switch key {
	case "id":
		return &t.id, nil
	case "name":
		return &t.name, nil
	default:
		return nil, fmt.Errorf("invalid key requested: %s", key)
	}
}

func TestExecute(t *testing.T) {
	optionsNum := int32(5)

	// Test cases
	tests := []struct {
		paginationOptions   Options
		sortByField         *FieldDescriptor
		name                string
		sortDirection       SortDirection
		expectSQL           string
		expectErrCode       errors.CodeType
		expectArguments     []interface{}
		resultCount         int
		expectedResultCount int
		expectHasNextPage   bool
		expectHasPrevPage   bool
	}{
		{
			name:              "invalid cursor",
			paginationOptions: Options{After: buildTestCursor("1", "test1")},
			expectErrCode:     errors.EInvalid,
		},
		{
			name:                "no pagination or sort by",
			paginationOptions:   Options{},
			resultCount:         10,
			expectSQL:           `SELECT * FROM "tests" ORDER BY "tests"."id" ASC`,
			expectArguments:     nil,
			expectHasNextPage:   false,
			expectHasPrevPage:   false,
			expectedResultCount: 10,
		},
		{
			name:                "limit results by first",
			paginationOptions:   Options{First: &optionsNum},
			resultCount:         6,
			expectSQL:           `SELECT * FROM "tests" ORDER BY "tests"."id" ASC LIMIT ?`,
			expectArguments:     []interface{}{int64(6)},
			expectHasNextPage:   true,
			expectHasPrevPage:   false,
			expectedResultCount: 5,
		},
		{
			name:                "limit results by last",
			paginationOptions:   Options{Last: &optionsNum},
			resultCount:         6,
			expectSQL:           `SELECT * FROM "tests" ORDER BY "tests"."id" DESC LIMIT ?`,
			expectArguments:     []interface{}{int64(6)},
			expectHasNextPage:   false,
			expectHasPrevPage:   true,
			expectedResultCount: 5,
		},
		{
			name:                "limit results by first with after cursor and asc sort",
			paginationOptions:   Options{First: &optionsNum, After: buildTestCursor("1", "test1")},
			sortByField:         &FieldDescriptor{Key: "name", Table: "tests", Col: "name"},
			sortDirection:       AscSort,
			resultCount:         6,
			expectSQL:           `SELECT * FROM "tests" WHERE (("tests"."name" > ?) OR (("tests"."id" > ?) AND ("tests"."name" = ?))) ORDER BY "tests"."name" ASC, "tests"."id" ASC LIMIT ?`,
			expectArguments:     []interface{}{"test1", "1", "test1", int64(6)},
			expectHasNextPage:   true,
			expectHasPrevPage:   true,
			expectedResultCount: 5,
		},
		{
			name:                "limit results by first with after cursor and desc sort",
			paginationOptions:   Options{First: &optionsNum, After: buildTestCursor("1", "test1")},
			sortByField:         &FieldDescriptor{Key: "name", Table: "tests", Col: "name"},
			sortDirection:       DescSort,
			resultCount:         6,
			expectSQL:           `SELECT * FROM "tests" WHERE (("tests"."name" < ?) OR (("tests"."id" < ?) AND ("tests"."name" = ?))) ORDER BY "tests"."name" DESC, "tests"."id" DESC LIMIT ?`,
			expectArguments:     []interface{}{"test1", "1", "test1", int64(6)},
			expectHasNextPage:   true,
			expectHasPrevPage:   true,
			expectedResultCount: 5,
		},
		{
			name:                "limit results by last with before cursor",
			paginationOptions:   Options{Last: &optionsNum, Before: buildTestCursor("1", "")},
			resultCount:         6,
			expectSQL:           `SELECT * FROM "tests" WHERE ("tests"."id" < ?) ORDER BY "tests"."id" ASC LIMIT ?`,
			expectArguments:     []interface{}{"1", int64(6)},
			expectHasNextPage:   true,
			expectHasPrevPage:   false,
			expectedResultCount: 5,
		},
		{
			name:                "limit results by first with before cursor",
			paginationOptions:   Options{First: &optionsNum, Before: buildTestCursor("1", "")},
			resultCount:         6,
			expectSQL:           `SELECT * FROM (SELECT * FROM "tests" WHERE ("tests"."id" < ?) ORDER BY "tests"."id" DESC LIMIT ?) AS "t1" ORDER BY "id" ASC`,
			expectArguments:     []interface{}{"1", int64(6)},
			expectHasNextPage:   true,
			expectHasPrevPage:   true,
			expectedResultCount: 5,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockRows := mocks.Rows{}
			mockRows.Test(t)

			mockRows.On("Next").Return(true).Times(test.resultCount)
			mockRows.On("Next").Return(false)

			mockRows.On("Scan", mock.Anything).Return(nil).Maybe()
			mockRows.On("Close").Return(nil).Maybe()
			mockRows.On("Err").Return(nil).Maybe()

			mockDBConn := MockConnection{}
			mockDBConn.Test(t)

			// Query function expects arguments as individual elements.
			queryArguments := []interface{}{mock.Anything, test.expectSQL}
			queryArguments = append(queryArguments, test.expectArguments...)

			mockDBConn.On("Query", queryArguments...).Return(&mockRows, nil)

			qBuilder, err := NewPaginatedQueryBuilder(
				&test.paginationOptions,
				&FieldDescriptor{Key: "id", Table: "tests", Col: "id"},
				WithSortByField(test.sortByField, test.sortDirection),
			)
			if err != nil {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			query := goqu.From("tests")

			rows, err := qBuilder.Execute(ctx, &mockDBConn, query)
			if err != nil {
				assert.Equal(t, test.expectErrCode, err)
				return
			}

			// Scan rows
			results := []testModel{}
			for rows.Next() {
				results = append(results, testModel{})
			}

			if err = rows.Finalize(&results); err != nil {
				assert.Equal(t, test.expectErrCode, err)
			}

			pageInfo := rows.GetPageInfo()

			assert.Equal(t, test.expectHasNextPage, pageInfo.HasNextPage)
			assert.Equal(t, test.expectHasPrevPage, pageInfo.HasPreviousPage)
			assert.Equal(t, test.expectedResultCount, len(results))

			c, err := pageInfo.Cursor(testModel{id: "1", name: "m1"})
			if err != nil {
				assert.Equal(t, test.expectErrCode, err)
				return
			}

			cursor, err := newCursor(*c)
			if err != nil {
				assert.Equal(t, test.expectErrCode, err)
				return
			}

			assert.Equal(t, "1", cursor.primary.value)

			if test.sortByField != nil {
				assert.Equal(t, ptr.String("m1"), cursor.secondary.value)
			} else {
				assert.Nil(t, cursor.secondary)
			}
		})
	}
}

// TestExecute_HasResultsReflectsRowsNotCount documents that HasResults is derived from the
// rows actually returned, not from COUNT, which Execute runs as a separate lazy statement. A
// concurrent commit (or a mid-stream error) between the two can leave COUNT > 0 while the row
// set is empty; HasResults stays false, so callers that gate on it never index an empty slice.
// TotalCount remains available but is computed lazily on demand.
func TestExecute_HasResultsReflectsRowsNotCount(t *testing.T) {
	ctx := t.Context()

	// SELECT returns zero rows (the row was filtered out by a concurrent commit).
	selectRows := mocks.Rows{}
	selectRows.Test(t)
	selectRows.On("Next").Return(false)
	selectRows.On("Err").Return(nil).Maybe()
	selectRows.On("Close").Return().Maybe()

	// COUNT returns 1, evaluated only when TotalCount is invoked.
	countRow := mocks.Rows{}
	countRow.Test(t)
	countRow.On("Scan", mock.Anything).Run(func(args mock.Arguments) {
		*(args.Get(0).(*int32)) = 1
	}).Return(nil)

	conn := MockConnection{}
	conn.Test(t)
	conn.On("Query", mock.Anything, mock.Anything, mock.Anything).Return(&selectRows, nil)
	conn.On("QueryRow", mock.Anything, mock.Anything).Return(&countRow)

	qBuilder, err := NewPaginatedQueryBuilder(
		&Options{First: ptr.Int32(1)},
		&FieldDescriptor{Key: "id", Table: "runs", Col: "id"},
	)
	if !assert.NoError(t, err) {
		return
	}

	rows, err := qBuilder.Execute(ctx, &conn, goqu.From("runs"))
	if !assert.NoError(t, err) {
		return
	}

	results := []testModel{}
	for rows.Next() {
		results = append(results, testModel{})
	}
	assert.NoError(t, rows.Finalize(&results))

	pageInfo := rows.GetPageInfo()

	// No rows were returned, so HasResults is false even though COUNT says 1.
	assert.False(t, pageInfo.HasResults)
	assert.Len(t, results, 0)

	// TotalCount is computed lazily and still reports the COUNT value.
	totalCount, err := pageInfo.TotalCount(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int32(1), totalCount)

	// Gating existence on HasResults is safe; the old TotalCount-based check was not.
	assert.NotPanics(t, func() {
		if !pageInfo.HasResults {
			return
		}
		_ = results[0]
	})
}

// TestExecute_TotalCountLazyAndMemoized verifies the COUNT runs only when TotalCount is
// invoked, and at most once across repeated calls.
func TestExecute_TotalCountLazyAndMemoized(t *testing.T) {
	ctx := t.Context()

	selectRows := mocks.Rows{}
	selectRows.Test(t)
	selectRows.On("Next").Return(true).Once()
	selectRows.On("Next").Return(false)
	selectRows.On("Scan", mock.Anything).Return(nil).Maybe()
	selectRows.On("Err").Return(nil).Maybe()
	selectRows.On("Close").Return().Maybe()

	countRow := mocks.Rows{}
	countRow.Test(t)
	countRow.On("Scan", mock.Anything).Run(func(args mock.Arguments) {
		*(args.Get(0).(*int32)) = 7
	}).Return(nil)

	conn := MockConnection{}
	conn.Test(t)
	conn.On("Query", mock.Anything, mock.Anything, mock.Anything).Return(&selectRows, nil)
	// .Once() makes a second COUNT a test failure, proving memoization.
	conn.On("QueryRow", mock.Anything, mock.Anything).Return(&countRow).Once()

	qBuilder, err := NewPaginatedQueryBuilder(&Options{First: ptr.Int32(1)}, &FieldDescriptor{Key: "id", Table: "runs", Col: "id"})
	if !assert.NoError(t, err) {
		return
	}

	rows, err := qBuilder.Execute(ctx, &conn, goqu.From("runs"))
	if !assert.NoError(t, err) {
		return
	}

	results := []testModel{}
	for rows.Next() {
		results = append(results, testModel{})
	}
	assert.NoError(t, rows.Finalize(&results))

	pageInfo := rows.GetPageInfo()

	// Lazy: no COUNT until TotalCount is invoked.
	conn.AssertNotCalled(t, "QueryRow", mock.Anything, mock.Anything)

	for range 3 {
		count, err := pageInfo.TotalCount(ctx)
		assert.NoError(t, err)
		assert.Equal(t, int32(7), count)
	}

	conn.AssertExpectations(t)
}

// TestExecute_TotalCountError surfaces a COUNT scan error.
func TestExecute_TotalCountError(t *testing.T) {
	ctx := t.Context()

	selectRows := mocks.Rows{}
	selectRows.Test(t)
	selectRows.On("Next").Return(false)
	selectRows.On("Err").Return(nil).Maybe()
	selectRows.On("Close").Return().Maybe()

	countRow := mocks.Rows{}
	countRow.Test(t)
	countRow.On("Scan", mock.Anything).Return(fmt.Errorf("boom"))

	conn := MockConnection{}
	conn.Test(t)
	conn.On("Query", mock.Anything, mock.Anything, mock.Anything).Return(&selectRows, nil)
	conn.On("QueryRow", mock.Anything, mock.Anything).Return(&countRow)

	qBuilder, err := NewPaginatedQueryBuilder(&Options{First: ptr.Int32(1)}, &FieldDescriptor{Key: "id", Table: "runs", Col: "id"})
	if !assert.NoError(t, err) {
		return
	}

	rows, err := qBuilder.Execute(ctx, &conn, goqu.From("runs"))
	if !assert.NoError(t, err) {
		return
	}

	_, err = rows.GetPageInfo().TotalCount(ctx)
	assert.Error(t, err)
}
