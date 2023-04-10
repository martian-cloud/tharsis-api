package resolver

import (
	"fmt"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

const maxQueryLimit = 100

// ConnectionQueryArgs are used to query a connection
type ConnectionQueryArgs struct {
	After  *string
	Before *string
	First  *int32
	Last   *int32
	Sort   *string
}

// Validate query args
func (c ConnectionQueryArgs) Validate() error {
	if c.First != nil && c.Last != nil {
		return errors.NewError(errors.EInvalid, "invalid args: only first or last can be used")
	}

	if c.First == nil && c.Last == nil {
		return errors.NewError(errors.EInvalid, "invalid args: either first or last must be specified")
	}

	if c.First != nil && (*c.First < 0 || *c.First > maxQueryLimit) {
		return errors.NewError(errors.EInvalid, fmt.Sprintf("invalid args: first must be between 0-%d", maxQueryLimit))
	}

	if c.Last != nil && (*c.Last < 0 || *c.Last > maxQueryLimit) {
		return errors.NewError(errors.EInvalid, fmt.Sprintf("invalid args: last must be between 0-%d", maxQueryLimit))
	}

	return nil
}

// PageInfo represents the page information for a connection
type PageInfo struct {
	EndCursor       *string
	StartCursor     *string
	HasNextPage     bool
	HasPreviousPage bool
}

// PageInfoResolver resolves the PageInfo type
type PageInfoResolver struct {
	pageInfo PageInfo
}

// EndCursor resolver
func (r *PageInfoResolver) EndCursor() *string {
	return r.pageInfo.EndCursor
}

// StartCursor resolver
func (r *PageInfoResolver) StartCursor() *string {
	return r.pageInfo.StartCursor
}

// HasNextPage resolver
func (r *PageInfoResolver) HasNextPage() bool {
	return r.pageInfo.HasNextPage
}

// HasPreviousPage resolver
func (r *PageInfoResolver) HasPreviousPage() bool {
	return r.pageInfo.HasPreviousPage
}

// Edge type
type Edge struct {
	CursorFunc pagination.CursorFunc
	Node       interface{}
}

// Connection type
type Connection struct {
	PageInfo   PageInfo
	Edges      []Edge
	TotalCount int32
}
