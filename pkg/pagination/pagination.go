// Package pagination provides functionalities
// related to cursor-based pagination.
package pagination

//go:generate go tool mockery --name Connection --inpackage --case underscore
//go:generate go tool mockery --srcpkg github.com/jackc/pgx/v4 --name Rows

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/jackc/pgx/v4"
	te "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// Constants used for sorting
const (
	gt   = "GT"
	lt   = "LT"
	ne   = "<>"
	asc  = "ASC"
	desc = "DESC"
)

// SortTransformFunc is a function for transforming the sort by field used in the query
type SortTransformFunc func(string) string

// SortDirection indicates the direction for sorting results
type SortDirection string

// SortDirection constants
const (
	AscSort  SortDirection = "ASC"
	DescSort SortDirection = "DESC"
)

// Connection is used to represent a DB connection
type Connection interface {
	Query(ctx context.Context, sql string, optionsAndArgs ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, optionsAndArgs ...any) pgx.Row
}

// CursorPaginatable implements functions needed to resolve fields for cursor pagination
type CursorPaginatable interface {
	ResolveMetadata(key string) (*string, error)
}

// CursorFunc creates an opaque cursor string
type CursorFunc func(cp CursorPaginatable) (*string, error)

// Options contain the cursor based pagination options
type Options struct {
	Before *string
	After  *string
	First  *int32
	Last   *int32
}

// Validate returns an error if the options are not valid
func (o *Options) validate() error {
	// This is also checked in the service layer before getting here.
	if (o.Before != nil) && (o.After != nil) {
		return errors.New("only before or after can be defined, not both")
	}
	if (o.First != nil) && (o.Last != nil) {
		return errors.New("only first or last can be defined, not both")
	}

	return nil
}

// FieldDescriptor defines a field descriptor
type FieldDescriptor struct {
	Key   string
	Table string
	Col   string
}

func (f *FieldDescriptor) getFullColName() string {
	return fmt.Sprintf("%s.%s", f.Table, f.Col)
}

// PageInfo contains the page information
type PageInfo struct {
	Cursor          CursorFunc
	TotalCount      int32
	HasNextPage     bool
	HasPreviousPage bool
}

// PaginatedRows contains the paginated query results
type PaginatedRows interface {
	pgx.Rows
	GetPageInfo() *PageInfo
	Finalize(resultsPtr any) error
}

// cursorPaginatedRows represents DB rows with pagination
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

// Next returns true if there are more rows.
func (c *cursorPaginatedRows) Next() bool {
	next := c.Rows.Next()

	if next {
		c.count++
	}

	return next
}

// Finalize finalizes the result set
func (c *cursorPaginatedRows) Finalize(resultsPtr any) error {
	ptr := reflect.ValueOf(resultsPtr)
	if ptr.Kind() != reflect.Ptr {
		return fmt.Errorf("expected pointer type, got %T", resultsPtr)
	}

	array := ptr.Elem()
	if array.Kind() != reflect.Array && array.Kind() != reflect.Slice {
		return fmt.Errorf("expected slice type, got %T", resultsPtr)
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

// GetPageInfo returns the PageInfo
func (c *cursorPaginatedRows) GetPageInfo() *PageInfo {
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

type extraOptions struct {
	sortBy            *FieldDescriptor
	sortTransformFunc SortTransformFunc
	sortDirection     SortDirection
}

// ExtraOptionFunc is a function to set extra options
type ExtraOptionFunc func(*extraOptions)

// WithSortByField sets the sort by field and direction
func WithSortByField(field *FieldDescriptor, direction SortDirection) ExtraOptionFunc {
	return func(o *extraOptions) {
		o.sortBy = field
		o.sortDirection = direction
	}
}

// WithSortByTransform sets the optional sort by transform function
func WithSortByTransform(f SortTransformFunc) ExtraOptionFunc {
	return func(o *extraOptions) {
		o.sortTransformFunc = f
	}
}

// PaginatedQueryBuilder represents a paginated DB query
type PaginatedQueryBuilder struct {
	options           *Options
	primaryKey        *FieldDescriptor
	sortBy            *FieldDescriptor
	limit             *int32
	cur               *cursor
	sortDirection     SortDirection
	sortTransformFunc SortTransformFunc
}

// NewPaginatedQueryBuilder returns a PaginatedQueryBuilder
func NewPaginatedQueryBuilder(
	options *Options,
	primaryKey *FieldDescriptor,
	extraOpts ...ExtraOptionFunc,
) (*PaginatedQueryBuilder, error) {
	extra := &extraOptions{}
	for _, o := range extraOpts {
		o(extra)
	}

	if options == nil {
		options = &Options{}
	}

	if err := options.validate(); err != nil {
		return nil, err
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
		return nil, err
	}

	// Verify sortBy matches cursor
	if cur != nil &&
		((cur.secondary != nil && extra.sortBy == nil) ||
			(cur.secondary == nil && extra.sortBy != nil) ||
			(cur.secondary != nil && extra.sortBy != nil && extra.sortBy.Key != cur.secondary.name)) {
		return nil, te.New("sort by argument does not match cursor", te.WithErrorCode(te.EInvalid))
	}

	return &PaginatedQueryBuilder{
		options:           options,
		primaryKey:        primaryKey,
		sortBy:            extra.sortBy,
		sortDirection:     extra.sortDirection,
		sortTransformFunc: extra.sortTransformFunc,
		limit:             limit,
		cur:               cur,
	}, nil
}

// Execute executes the paginated query using the DB Connection
func (p *PaginatedQueryBuilder) Execute(ctx context.Context, conn Connection, query *goqu.SelectDataset) (PaginatedRows, error) {
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
		return nil, fmt.Errorf("failed to scan query count result: %w", err)
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
		cursorFunc: func(cp CursorPaginatable) (*string, error) {
			pKeyVal, err := cp.ResolveMetadata(p.primaryKey.Key)
			if err != nil {
				return nil, err
			}

			if pKeyVal == nil {
				return nil, te.New("failed to create cursor: primary key value cannot be null", te.WithErrorCode(te.EInvalid))
			}

			cur := cursor{primary: &primaryCursorField{name: p.primaryKey.Key, value: *pKeyVal}}

			if p.sortBy != nil {
				sKeyVal, frErr := cp.ResolveMetadata(p.sortBy.Key)
				if frErr != nil {
					return nil, frErr
				}

				cur.secondary = &secondaryCursorField{name: p.sortBy.Key, value: sKeyVal}
			}

			encodedCursor, err := cur.encode()
			if err != nil {
				return nil, err
			}

			return &encodedCursor, nil
		},
	}, nil
}

func (p *PaginatedQueryBuilder) buildWhereCondition() goqu.Expression {
	var op string

	afterOp := gt
	beforeOp := lt
	if p.sortDirection == DescSort {
		afterOp = lt
		beforeOp = gt
	}

	op = beforeOp
	if p.options.After != nil {
		op = afterOp
	}

	if p.cur != nil {
		if p.cur.secondary != nil {
			// If secondary cursor value is nil, we need to find rows where the sort by is not null or rows
			// where the sort value is null but the primary key value is gt/lt the primary cursor value
			// depending on the sort direction.
			if p.cur.secondary.value == nil {
				return goqu.Or(
					goqu.I(p.sortBy.getFullColName()).IsNotNull(),
					goqu.Ex{
						p.sortBy.getFullColName():     nil,
						p.primaryKey.getFullColName(): goqu.Op{op: p.cur.primary.value},
					},
				)
			}
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

func (p *PaginatedQueryBuilder) buildOrderBy() []exp.OrderedExpression {
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

	if p.sortBy != nil {
		expressions = append(expressions, p.buildOrderByExpression(p.buildSortByExpr(p.sortBy.getFullColName()), direction))
	}
	expressions = append(expressions, p.buildOrderByExpression(goqu.I(p.primaryKey.getFullColName()), direction))

	return expressions
}

func (p *PaginatedQueryBuilder) buildOuterReverseOrderBy() []exp.OrderedExpression {
	expressions := []exp.OrderedExpression{}

	if p.sortBy != nil {
		expressions = append(expressions, p.buildOrderByExpression(p.buildSortByExpr(p.sortBy.Col), asc))
	}
	expressions = append(expressions, p.buildOrderByExpression(goqu.I(p.primaryKey.Col), asc))

	return expressions
}

func (p *PaginatedQueryBuilder) buildOrderByExpression(ex exp.Orderable, direction string) exp.OrderedExpression {
	if direction == desc {
		return ex.Desc()
	}
	return ex.Asc()
}

func (p *PaginatedQueryBuilder) buildSortByExpr(field string) exp.Orderable {
	if p.sortTransformFunc != nil {
		return goqu.L(p.sortTransformFunc(field))
	}
	return goqu.I(field)
}
