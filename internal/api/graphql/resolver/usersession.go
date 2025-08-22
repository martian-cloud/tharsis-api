package resolver

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/user"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"

	graphql "github.com/graph-gophers/graphql-go"
)

// UserSessionEdgeResolver resolves user session edges
type UserSessionEdgeResolver struct {
	edge Edge
}

// Cursor returns an opaque cursor
func (r *UserSessionEdgeResolver) Cursor() (string, error) {
	userSession, ok := r.edge.Node.(models.UserSession)
	if !ok {
		return "", errors.New("Failed to convert node type")
	}
	cursor, err := r.edge.CursorFunc(&userSession)
	return *cursor, err
}

// Node returns a user session node
func (r *UserSessionEdgeResolver) Node() (*UserSessionResolver, error) {
	userSession, ok := r.edge.Node.(models.UserSession)
	if !ok {
		return nil, errors.New("Failed to convert node type")
	}

	return &UserSessionResolver{userSession: &userSession}, nil
}

// UserSessionConnectionResolver resolves a user session connection
type UserSessionConnectionResolver struct {
	connection Connection
}

// NewUserSessionConnectionResolver creates a new UserSessionConnectionResolver
func NewUserSessionConnectionResolver(ctx context.Context, input *user.GetUserSessionsInput) (*UserSessionConnectionResolver, error) {
	userService := getServiceCatalog(ctx).UserService

	result, err := userService.GetUserSessions(ctx, input)
	if err != nil {
		return nil, err
	}

	userSessions := result.UserSessions

	// Create edges
	edges := make([]Edge, len(userSessions))
	for i, userSession := range userSessions {
		edges[i] = Edge{CursorFunc: result.PageInfo.Cursor, Node: userSession}
	}

	pageInfo := PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
	}

	if len(userSessions) > 0 {
		var err error
		pageInfo.StartCursor, err = result.PageInfo.Cursor(&userSessions[0])
		if err != nil {
			return nil, err
		}

		pageInfo.EndCursor, err = result.PageInfo.Cursor(&userSessions[len(edges)-1])
		if err != nil {
			return nil, err
		}
	}

	connection := Connection{
		TotalCount: result.PageInfo.TotalCount,
		PageInfo:   pageInfo,
		Edges:      edges,
	}

	return &UserSessionConnectionResolver{connection: connection}, nil
}

// TotalCount returns the total result count for the connection
func (r *UserSessionConnectionResolver) TotalCount() int32 {
	return r.connection.TotalCount
}

// PageInfo returns the connection page information
func (r *UserSessionConnectionResolver) PageInfo() *PageInfoResolver {
	return &PageInfoResolver{pageInfo: r.connection.PageInfo}
}

// Edges returns the connection edges
func (r *UserSessionConnectionResolver) Edges() *[]*UserSessionEdgeResolver {
	resolvers := make([]*UserSessionEdgeResolver, len(r.connection.Edges))
	for i, edge := range r.connection.Edges {
		resolvers[i] = &UserSessionEdgeResolver{edge: edge}
	}
	return &resolvers
}

// UserSessionResolver resolves a UserSession type
type UserSessionResolver struct {
	userSession *models.UserSession
}

// ID resolver
func (r *UserSessionResolver) ID() graphql.ID {
	return graphql.ID(r.userSession.GetGlobalID())
}

// Metadata resolver
func (r *UserSessionResolver) Metadata() *MetadataResolver {
	return &MetadataResolver{metadata: &r.userSession.Metadata}
}

// UserAgent resolver
func (r *UserSessionResolver) UserAgent() string {
	return r.userSession.UserAgent
}

// Expiration resolver
func (r *UserSessionResolver) Expiration() graphql.Time {
	return graphql.Time{Time: r.userSession.Expiration}
}

// Expired resolver
func (r *UserSessionResolver) Expired() bool {
	return r.userSession.IsExpired()
}
