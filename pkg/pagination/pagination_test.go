package pagination

import (
	"context"
	"fmt"
	"testing"

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

func (t testModel) ResolveMetadata(key string) (string, error) {
	switch key {
	case "id":
		return t.id, nil
	case "name":
		return t.name, nil
	default:
		return "", fmt.Errorf("invalid key requested: %s", key)
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
		expectCountSQL      string
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
			expectCountSQL:      `SELECT COUNT(*) FROM "tests"`,
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
			expectCountSQL:      `SELECT COUNT(*) FROM "tests"`,
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
			expectCountSQL:      `SELECT COUNT(*) FROM "tests"`,
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
			expectCountSQL:      `SELECT COUNT(*) FROM "tests"`,
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
			expectCountSQL:      `SELECT COUNT(*) FROM "tests"`,
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
			expectCountSQL:      `SELECT COUNT(*) FROM "tests"`,
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

			mockDBConn := MockConnection{}
			mockDBConn.Test(t)

			// Query function expects arguments as individual elements.
			queryArguments := []interface{}{mock.Anything, test.expectSQL}
			queryArguments = append(queryArguments, test.expectArguments...)

			mockDBConn.On("Query", queryArguments...).Return(&mockRows, nil)
			mockDBConn.On("QueryRow", mock.Anything, mock.Anything).Return(&mockCountRows, nil)

			qBuilder, err := NewPaginatedQueryBuilder(
				&test.paginationOptions,
				&FieldDescriptor{Key: "id", Table: "tests", Col: "id"},
				test.sortByField,
				test.sortDirection,
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
				assert.Equal(t, "m1", cursor.secondary.value)
			} else {
				assert.Nil(t, cursor.secondary)
			}
		})
	}
}
