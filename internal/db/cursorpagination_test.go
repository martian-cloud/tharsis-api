package db

import (
	"context"
	"testing"

	"github.com/doug-martin/goqu/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db/mocks"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
)

type testModel struct {
	id   string
	name string
}

func TestExecute(t *testing.T) {
	// Test cases
	tests := []struct {
		paginationOptions   PaginationOptions
		sortByField         *fieldDescriptor
		name                string
		sortDirection       SortDirection
		expectSQL           string
		expectCountSQL      string
		expectErrCode       string
		expectArguments     []interface{}
		resultCount         int
		expectedResultCount int
		expectHasNextPage   bool
		expectHasPrevPage   bool
	}{
		{
			name:              "invalid cursor",
			paginationOptions: PaginationOptions{After: buildTestCursor("1", "test1")},
			expectErrCode:     errors.EInvalid,
		},
		{
			name:                "no pagination or sort by",
			paginationOptions:   PaginationOptions{},
			resultCount:         10,
			expectSQL:           `SELECT * FROM "tests" ORDER BY "tests"."id" ASC`,
			expectArguments:     nil,
			expectCountSQL:      `SELECT COUNT(*) FROM "tests"`,
			expectHasNextPage:   false,
			expectHasPrevPage:   false,
			expectedResultCount: 10,
		},
		{
			name:                "limit results by first",
			paginationOptions:   PaginationOptions{First: int32Ptr(5)},
			resultCount:         6,
			expectSQL:           `SELECT * FROM "tests" ORDER BY "tests"."id" ASC LIMIT ?`,
			expectArguments:     []interface{}{int64(6)},
			expectCountSQL:      `SELECT COUNT(*) FROM "tests"`,
			expectHasNextPage:   true,
			expectHasPrevPage:   false,
			expectedResultCount: 5,
		},
		{
			name:                "limit results by last",
			paginationOptions:   PaginationOptions{Last: int32Ptr(5)},
			resultCount:         6,
			expectSQL:           `SELECT * FROM "tests" ORDER BY "tests"."id" DESC LIMIT ?`,
			expectArguments:     []interface{}{int64(6)},
			expectCountSQL:      `SELECT COUNT(*) FROM "tests"`,
			expectHasNextPage:   false,
			expectHasPrevPage:   true,
			expectedResultCount: 5,
		},
		{
			name:                "limit results by first with after cursor and asc sort",
			paginationOptions:   PaginationOptions{First: int32Ptr(5), After: buildTestCursor("1", "test1")},
			sortByField:         &fieldDescriptor{key: "name", table: "tests", col: "name"},
			sortDirection:       AscSort,
			resultCount:         6,
			expectSQL:           `SELECT * FROM "tests" WHERE (("tests"."name" > ?) OR (("tests"."id" > ?) AND ("tests"."name" = ?))) ORDER BY "tests"."name" ASC, "tests"."id" ASC LIMIT ?`,
			expectArguments:     []interface{}{"test1", "1", "test1", int64(6)},
			expectCountSQL:      `SELECT COUNT(*) FROM "tests"`,
			expectHasNextPage:   true,
			expectHasPrevPage:   true,
			expectedResultCount: 5,
		},
		{
			name:                "limit results by first with after cursor and desc sort",
			paginationOptions:   PaginationOptions{First: int32Ptr(5), After: buildTestCursor("1", "test1")},
			sortByField:         &fieldDescriptor{key: "name", table: "tests", col: "name"},
			sortDirection:       DescSort,
			resultCount:         6,
			expectSQL:           `SELECT * FROM "tests" WHERE (("tests"."name" < ?) OR (("tests"."id" < ?) AND ("tests"."name" = ?))) ORDER BY "tests"."name" DESC, "tests"."id" DESC LIMIT ?`,
			expectArguments:     []interface{}{"test1", "1", "test1", int64(6)},
			expectCountSQL:      `SELECT COUNT(*) FROM "tests"`,
			expectHasNextPage:   true,
			expectHasPrevPage:   true,
			expectedResultCount: 5,
		},
		{
			name:                "limit results by last with before cursor",
			paginationOptions:   PaginationOptions{Last: int32Ptr(5), Before: buildTestCursor("1", "")},
			resultCount:         6,
			expectSQL:           `SELECT * FROM "tests" WHERE ("tests"."id" < ?) ORDER BY "tests"."id" ASC LIMIT ?`,
			expectArguments:     []interface{}{"1", int64(6)},
			expectCountSQL:      `SELECT COUNT(*) FROM "tests"`,
			expectHasNextPage:   true,
			expectHasPrevPage:   false,
			expectedResultCount: 5,
		},
		{
			name:                "limit results by first with before cursor",
			paginationOptions:   PaginationOptions{First: int32Ptr(5), Before: buildTestCursor("1", "")},
			resultCount:         6,
			expectSQL:           `SELECT * FROM (SELECT * FROM "tests" WHERE ("tests"."id" < ?) ORDER BY "tests"."id" DESC LIMIT ?) AS "t1" ORDER BY "id" ASC`,
			expectArguments:     []interface{}{"1", int64(6)},
			expectCountSQL:      `SELECT COUNT(*) FROM "tests"`,
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

			mockCountRows := mocks.Rows{}
			mockCountRows.Test(t)

			mockCountRows.On("Scan", mock.Anything).Return(nil)

			mockDBConn := Mockconnection{}
			mockDBConn.Test(t)

			// Query function expects arguments as individual elements.
			queryArguments := []interface{}{mock.Anything, test.expectSQL}
			queryArguments = append(queryArguments, test.expectArguments...)

			mockDBConn.On("Query", queryArguments...).Return(&mockRows, nil)
			mockDBConn.On("QueryRow", mock.Anything, mock.Anything).Return(&mockCountRows, nil)

			qBuilder, err := newPaginatedQueryBuilder(
				&test.paginationOptions,
				&fieldDescriptor{key: "id", table: "tests", col: "id"},
				test.sortByField,
				test.sortDirection,
				func(key string, model interface{}) (string, error) {
					tm := model.(testModel)
					switch key {
					case "id":
						return tm.id, nil
					case "name":
						return tm.name, nil
					default:
						return "", errors.NewError(errors.EInternal, "Invalid key")
					}
				},
			)
			if err != nil {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err), "Unexpected error occurred")
				return
			}

			query := goqu.From("tests")

			rows, err := qBuilder.execute(ctx, &mockDBConn, query)
			if err != nil {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err), "Unexpected error occurred")
				return
			}

			// Scan rows
			results := []testModel{}
			for rows.Next() {
				results = append(results, testModel{})
			}

			if err = rows.finalize(&results); err != nil {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err), "Unexpected error occurred")
			}

			pageInfo := rows.getPageInfo()

			assert.Equal(t, test.expectHasNextPage, pageInfo.HasNextPage)
			assert.Equal(t, test.expectHasPrevPage, pageInfo.HasPreviousPage)
			assert.Equal(t, test.expectedResultCount, len(results))

			c, err := pageInfo.Cursor(testModel{id: "1", name: "m1"})
			if err != nil {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err), "Unexpected error occurred")
				return
			}

			cursor, err := newCursor(*c)
			if err != nil {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err), "Unexpected error occurred")
				return
			}

			assert.Equal(t, "1", cursor.primary.value)

			if test.sortByField != nil {
				assert.Equal(t, "m1", cursor.secondary.value)
			} else {
				assert.Nil(t, cursor.secondary)
			}
		})
	}
}

func int32Ptr(val int) *int32 {
	resp := int32(val)
	return &resp
}
