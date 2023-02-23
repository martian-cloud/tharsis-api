package resolver

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/loader"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/namespacemembership"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/user"

	"github.com/graph-gophers/dataloader"
	graphql "github.com/graph-gophers/graphql-go"
)

// UserConnectionQueryArgs are used to query a user connection
type UserConnectionQueryArgs struct {
	ConnectionQueryArgs
	Search *string
}

// UserEdgeResolver resolves user edges
type UserEdgeResolver struct {
	edge Edge
}

// Cursor returns an opaque cursor
func (r *UserEdgeResolver) Cursor() (string, error) {
	user, ok := r.edge.Node.(models.User)
	if !ok {
		return "", errors.NewError(errors.EInternal, "Failed to convert node type")
	}
	cursor, err := r.edge.CursorFunc(&user)
	return *cursor, err
}

// Node returns a user node
func (r *UserEdgeResolver) Node() (*UserResolver, error) {
	user, ok := r.edge.Node.(models.User)
	if !ok {
		return nil, errors.NewError(errors.EInternal, "Failed to convert node type")
	}

	return &UserResolver{user: &user}, nil
}

// UserConnectionResolver resolves a user connection
type UserConnectionResolver struct {
	connection Connection
}

// NewUserConnectionResolver creates a new UserConnectionResolver
func NewUserConnectionResolver(ctx context.Context, input *user.GetUsersInput) (*UserConnectionResolver, error) {
	userService := getUserService(ctx)

	result, err := userService.GetUsers(ctx, input)
	if err != nil {
		return nil, err
	}

	users := result.Users

	// Create edges
	edges := make([]Edge, len(users))
	for i, user := range users {
		edges[i] = Edge{CursorFunc: result.PageInfo.Cursor, Node: user}
	}

	pageInfo := PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
	}

	if len(users) > 0 {
		var err error
		pageInfo.StartCursor, err = result.PageInfo.Cursor(&users[0])
		if err != nil {
			return nil, err
		}

		pageInfo.EndCursor, err = result.PageInfo.Cursor(&users[len(edges)-1])
		if err != nil {
			return nil, err
		}
	}

	connection := Connection{
		TotalCount: result.PageInfo.TotalCount,
		PageInfo:   pageInfo,
		Edges:      edges,
	}

	return &UserConnectionResolver{connection: connection}, nil
}

// TotalCount returns the total result count for the connection
func (r *UserConnectionResolver) TotalCount() int32 {
	return r.connection.TotalCount
}

// PageInfo returns the connection page information
func (r *UserConnectionResolver) PageInfo() *PageInfoResolver {
	return &PageInfoResolver{pageInfo: r.connection.PageInfo}
}

// Edges returns the connection edges
func (r *UserConnectionResolver) Edges() *[]*UserEdgeResolver {
	resolvers := make([]*UserEdgeResolver, len(r.connection.Edges))
	for i, edge := range r.connection.Edges {
		resolvers[i] = &UserEdgeResolver{edge: edge}
	}
	return &resolvers
}

// UserResolver resolves a User type
type UserResolver struct {
	user *models.User
}

// ID resolver
func (r *UserResolver) ID() graphql.ID {
	return graphql.ID(gid.ToGlobalID(gid.UserType, r.user.Metadata.ID))
}

// Username resolver
func (r *UserResolver) Username() string {
	return r.user.Username
}

// Email resolver
func (r *UserResolver) Email() string {
	return r.user.Email
}

// Metadata resolver
func (r *UserResolver) Metadata() *MetadataResolver {
	return &MetadataResolver{metadata: &r.user.Metadata}
}

// NamespaceMemberships resolver
func (r *UserResolver) NamespaceMemberships(ctx context.Context,
	args *ConnectionQueryArgs,
) (*NamespaceMembershipConnectionResolver, error) {
	if err := args.Validate(); err != nil {
		return nil, err
	}

	input := namespacemembership.GetNamespaceMembershipsForSubjectInput{
		PaginationOptions: &db.PaginationOptions{First: args.First, Last: args.Last, After: args.After, Before: args.Before},
		UserID:            &r.user.Metadata.ID,
	}

	if args.Sort != nil {
		sort := db.NamespaceMembershipSortableField(*args.Sort)
		input.Sort = &sort
	}

	return NewNamespaceMembershipConnectionResolver(ctx, &input)
}

// Admin resolver
func (r *UserResolver) Admin() bool {
	return r.user.Admin
}

// Active resolver
func (r *UserResolver) Active() bool {
	return r.user.Active
}

// SCIMExternalID resolver
func (r *UserResolver) SCIMExternalID() *string {
	return &r.user.SCIMExternalID
}

// ActivityEvents resolver
func (r *UserResolver) ActivityEvents(ctx context.Context,
	args *ActivityEventConnectionQueryArgs,
) (*ActivityEventConnectionResolver, error) {
	input, err := getActivityEventsInputFromQueryArgs(ctx, args)
	if err != nil {
		// error is already a Tharsis error
		return nil, err
	}

	// Need to filter to this user.
	input.UserID = &r.user.Metadata.ID

	return NewActivityEventConnectionResolver(ctx, input)
}

func usersQuery(ctx context.Context, args *UserConnectionQueryArgs) (*UserConnectionResolver, error) {
	if err := args.Validate(); err != nil {
		return nil, err
	}

	input := user.GetUsersInput{
		PaginationOptions: &db.PaginationOptions{First: args.First, Last: args.Last, After: args.After, Before: args.Before},
		UsernamePrefix:    args.Search,
	}

	if args.Sort != nil {
		sort := db.UserSortableField(*args.Sort)
		input.Sort = &sort
	}

	return NewUserConnectionResolver(ctx, &input)
}

/* User loader */

const userLoaderKey = "user"

// RegisterUserLoader registers a user loader function
func RegisterUserLoader(collection *loader.Collection) {
	collection.Register(userLoaderKey, userBatchFunc)
}

func loadUser(ctx context.Context, id string) (*models.User, error) {
	ldr, err := loader.Extract(ctx, userLoaderKey)
	if err != nil {
		return nil, err
	}

	data, err := ldr.Load(ctx, dataloader.StringKey(id))()
	if err != nil {
		return nil, err
	}

	user, ok := data.(models.User)
	if !ok {
		return nil, errors.NewError(errors.EInternal, "Wrong type")
	}

	return &user, nil
}

func userBatchFunc(ctx context.Context, ids []string) (loader.DataBatch, error) {
	userService := getUserService(ctx)

	users, err := userService.GetUsersByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	// Build map of results
	batch := loader.DataBatch{}
	for _, result := range users {
		batch[result.Metadata.ID] = result
	}

	return batch, nil
}
