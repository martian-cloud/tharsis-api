package db

import (
	"context"
	"fmt"
	"reflect"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/jackc/pgx/v4"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
)

// CursorFunc creates an opaque cursor string
type CursorFunc func(item interface{}) (*string, error)

// PaginationOptions contain the cursor based pagination options
type PaginationOptions struct {
	Before *string
	After  *string
	First  *int32
	Last   *int32
}

// Validate returns an error if the options are not valid.
func (po *PaginationOptions) Validate() error {
	// This is also checked in the service layer before getting here.
	if (po.Before != nil) && (po.After != nil) {
		return errors.NewError(errors.EInvalid, "only before or after can be defined, not both")
	}
	if (po.First != nil) && (po.Last != nil) {
		return errors.NewError(errors.EInvalid, "only first or last can be defined, not both")
	}

	return nil
}

// PageInfo contains the page information
type PageInfo struct {
	Cursor          CursorFunc
	TotalCount      int32
	HasNextPage     bool
	HasPreviousPage bool
}

// paginatedRows contains the paginated query results
type paginatedRows interface {
	pgx.Rows
	getPageInfo() *PageInfo
	finalize(resultsPtr interface{}) error
}

const (
	gt   = "GT"
	lt   = "LT"
	asc  = "ASC"
	desc = "DESC"
)

type fieldResolverFunc func(key string, model interface{}) (string, error)

type fieldDescriptor struct {
	key   string
	table string
	col   string
}

func (f *fieldDescriptor) getFullColName() string {
	return fmt.Sprintf("%s.%s", f.table, f.col)
}

type cursorPaginatedRows struct {
	pgx.Rows
	limit      *int32
	first      *int32
	last       *int32
	before     *string
	after      *string
	cursorFunc CursorFunc
	totalCount int32
	count      int32
}

func (c *cursorPaginatedRows) Next() bool {
	next := c.Rows.Next()

	if next {
		c.count++
	}

	return next
}

func (c *cursorPaginatedRows) finalize(resultsPtr interface{}) error {
	ptr := reflect.ValueOf(resultsPtr)
	if ptr.Kind() != reflect.Ptr {
		return errors.NewError(errors.EInternal, fmt.Sprintf("expected pointer type, got %T", resultsPtr))
	}

	array := ptr.Elem()
	if array.Kind() != reflect.Array && array.Kind() != reflect.Slice {
		return errors.NewError(errors.EInternal, fmt.Sprintf("expected slice type, got %T", resultsPtr))
	}

	if c.limit != nil && c.count > *c.limit {
		if c.before != nil && c.first != nil {
			// Remove the first element
			array.Set(array.Slice(1, array.Len()))
		} else {
			// Remove the last item
			array.Set(array.Slice(0, int(*c.limit)))
		}
	}

	return nil
}

func (c *cursorPaginatedRows) getPageInfo() *PageInfo {
	pageInfo := PageInfo{TotalCount: c.totalCount}

	// Handle all possible permutations
	// Limit will be non nil if first or last is set
	if c.after != nil && c.first != nil {
		pageInfo.HasNextPage = c.count > *c.limit
		pageInfo.HasPreviousPage = true
	} else if c.after != nil && c.last != nil {
		pageInfo.HasNextPage = false
		pageInfo.HasPreviousPage = true
	} else if c.before != nil && c.first != nil {
		pageInfo.HasNextPage = true
		pageInfo.HasPreviousPage = c.count > *c.limit
	} else if c.before != nil && c.last != nil {
		pageInfo.HasNextPage = true
		pageInfo.HasPreviousPage = false
	} else if c.first != nil {
		pageInfo.HasNextPage = c.count > *c.limit
		pageInfo.HasPreviousPage = false
	} else if c.last != nil {
		pageInfo.HasNextPage = false
		pageInfo.HasPreviousPage = c.count > *c.limit
	} else if c.last == nil && c.first == nil && c.before != nil {
		pageInfo.HasNextPage = true
		pageInfo.HasPreviousPage = false
	} else if c.last == nil && c.first == nil && c.after != nil {
		pageInfo.HasNextPage = false
		pageInfo.HasPreviousPage = true
	}

	pageInfo.Cursor = c.cursorFunc

	return &pageInfo
}

type paginatedQueryBuilder struct {
	options       *PaginationOptions
	primaryKey    *fieldDescriptor
	sortBy        *fieldDescriptor
	limit         *int32
	fieldResolver fieldResolverFunc
	cur           *cursor
	sortDirection SortDirection
}

func newPaginatedQueryBuilder(
	options *PaginationOptions,
	primaryKey *fieldDescriptor,
	sortBy *fieldDescriptor,
	sortDirection SortDirection,
	fieldResolver fieldResolverFunc,
) (*paginatedQueryBuilder, error) {
	if options == nil {
		options = &PaginationOptions{}
	}

	if err := options.Validate(); err != nil {
		return nil, errors.NewError(errors.EInvalid, err.Error())
	}

	var limit *int32
	if options.First != nil {
		limit = options.First
	} else {
		limit = options.Last
	}

	var cur *cursor
	var err error

	if options.After != nil {
		cur, err = newCursor(*options.After)
	}

	if options.Before != nil {
		cur, err = newCursor(*options.Before)
	}

	if err != nil {
		return nil, errors.NewError(
			errors.EInvalid,
			"Failed to decode cursor",
			errors.WithErrorErr(err),
		)
	}

	// Verify sortby matches cursor
	if cur != nil &&
		((cur.secondary != nil && sortBy == nil) ||
			(cur.secondary == nil && sortBy != nil) ||
			(cur.secondary != nil && sortBy != nil && sortBy.key != cur.secondary.name)) {
		return nil, errors.NewError(errors.EInvalid, "Sort by argument does not match cursor")
	}

	return &paginatedQueryBuilder{
		options:       options,
		primaryKey:    primaryKey,
		sortBy:        sortBy,
		sortDirection: sortDirection,
		limit:         limit,
		fieldResolver: fieldResolver,
		cur:           cur,
	}, nil
}

func (p *paginatedQueryBuilder) execute(ctx context.Context, conn connection, query *goqu.SelectDataset) (paginatedRows, error) {
	// Copy original query which will be used to get the total count
	originalQuery := *query

	where := p.buildWhereCondition()
	if where != nil {
		query = query.Where(where)
	}

	query = query.Order(p.buildOrderBy()...)

	if p.limit != nil {
		// Add one to the limit to query an additional row to determine if there is a next page
		query = query.Limit(uint(*p.limit) + 1)
	}

	if p.options.Before != nil && p.options.First != nil {
		// When using a before with the first field, we need to reverse the query results
		query = goqu.From(query).Order(p.buildOuterReverseOrderBy()...)
	}

	sql, args, err := query.Prepared(true).ToSQL()
	if err != nil {
		return nil, err
	}

	// Build count query
	countSQL, countArgs, err := originalQuery.Prepared(true).Select(goqu.COUNT("*")).ToSQL()
	if err != nil {
		return nil, err
	}

	row := conn.QueryRow(ctx, countSQL, countArgs...)

	var count int32
	if err = row.Scan(&count); err != nil {
		return nil, errors.NewError(errors.EInternal, "Failed to scan query count result", errors.WithErrorErr(err))
	}

	rows, err := conn.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}

	return &cursorPaginatedRows{
		Rows:       rows,
		totalCount: count,
		limit:      p.limit,
		first:      p.options.First,
		last:       p.options.Last,
		before:     p.options.Before,
		after:      p.options.After,
		cursorFunc: func(item interface{}) (*string, error) {
			pKeyVal, err := p.fieldResolver(p.primaryKey.key, item)
			if err != nil {
				return nil, err
			}

			cur := cursor{primary: &cursorField{name: p.primaryKey.key, value: pKeyVal}}

			if p.sortBy != nil {
				sKeyVal, frErr := p.fieldResolver(p.sortBy.key, item)
				if frErr != nil {
					return nil, frErr
				}

				cur.secondary = &cursorField{name: p.sortBy.key, value: sKeyVal}
			}

			encodedCursor, err := cur.encode()
			if err != nil {
				return nil, err
			}

			return &encodedCursor, nil
		},
	}, nil
}

func (p *paginatedQueryBuilder) buildWhereCondition() goqu.Expression {
	var op string

	afterOp := gt
	beforeOp := lt

	if p.sortDirection == DescSort {
		afterOp = lt
		beforeOp = gt
	}

	if p.options.After != nil {
		op = afterOp
	} else {
		op = beforeOp
	}

	if p.cur != nil {
		if p.cur.secondary != nil {
			return goqu.Or(
				goqu.Ex{
					p.sortBy.getFullColName(): goqu.Op{op: p.cur.secondary.value},
				},
				goqu.Ex{
					p.sortBy.getFullColName():     p.cur.secondary.value,
					p.primaryKey.getFullColName(): goqu.Op{op: p.cur.primary.value},
				},
			)
		}
		return goqu.Ex{p.primaryKey.getFullColName(): goqu.Op{op: p.cur.primary.value}}
	}

	return nil
}

func (p *paginatedQueryBuilder) buildOrderBy() []exp.OrderedExpression {
	expressions := []exp.OrderedExpression{}

	forward := asc
	backward := desc

	if p.sortDirection == DescSort {
		forward = desc
		backward = asc
	}

	direction := forward

	if (p.options.Before == nil && p.options.Last != nil) || (p.options.Before != nil && p.options.First != nil) {
		direction = backward
	}

	idCol := p.primaryKey.getFullColName()
	if p.sortBy != nil {
		expressions = append(
			expressions,
			p.buildOrderByExpression(goqu.I(p.sortBy.getFullColName()), direction),
		)
	}
	expressions = append(expressions, p.buildOrderByExpression(goqu.I(idCol), direction))

	return expressions
}

func (p *paginatedQueryBuilder) buildOuterReverseOrderBy() []exp.OrderedExpression {
	expressions := []exp.OrderedExpression{}

	direction := asc

	idCol := p.primaryKey.col
	if p.sortBy != nil {
		expressions = append(
			expressions,
			p.buildOrderByExpression(goqu.I(p.sortBy.col), direction),
		)
	}
	expressions = append(expressions, p.buildOrderByExpression(goqu.I(idCol), direction))

	return expressions
}

func (p *paginatedQueryBuilder) buildOrderByExpression(ex exp.IdentifierExpression, dir string) exp.OrderedExpression {
	if dir == desc {
		return ex.Desc()
	}
	return ex.Asc()
}
