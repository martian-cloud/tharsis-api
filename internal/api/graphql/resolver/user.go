package resolver

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/loader"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/namespacemembership"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/team"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/user"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"

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
		return "", errors.New("Failed to convert node type")
	}
	cursor, err := r.edge.CursorFunc(&user)
	return *cursor, err
}

// Node returns a user node
func (r *UserEdgeResolver) Node() (*UserResolver, error) {
	user, ok := r.edge.Node.(models.User)
	if !ok {
		return nil, errors.New("Failed to convert node type")
	}

	return &UserResolver{user: &user}, nil
}

// UserConnectionResolver resolves a user connection
type UserConnectionResolver struct {
	connection Connection
}

// NewUserConnectionResolver creates a new UserConnectionResolver
func NewUserConnectionResolver(ctx context.Context, input *user.GetUsersInput) (*UserConnectionResolver, error) {
	userService := getServiceCatalog(ctx).UserService

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
	return graphql.ID(r.user.GetGlobalID())
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
		PaginationOptions: &pagination.Options{First: args.First, Last: args.Last, After: args.After, Before: args.Before},
		UserID:            &r.user.Metadata.ID,
	}

	if args.Sort != nil {
		sort := db.NamespaceMembershipSortableField(*args.Sort)
		input.Sort = &sort
	}

	return NewNamespaceMembershipConnectionResolver(ctx, &input)
}

// Teams resolver
func (r *UserResolver) Teams(ctx context.Context,
	args *ConnectionQueryArgs,
) (*TeamConnectionResolver, error) {
	if err := args.Validate(); err != nil {
		return nil, err
	}

	input := team.GetTeamsInput{
		PaginationOptions: &pagination.Options{First: args.First, Last: args.Last, After: args.After, Before: args.Before},
		UserID:            &r.user.Metadata.ID,
	}

	if args.Sort != nil {
		sort := db.TeamSortableField(*args.Sort)
		input.Sort = &sort
	}

	return NewTeamConnectionResolver(ctx, &input)
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

// UserSessions resolver
func (r *UserResolver) UserSessions(ctx context.Context,
	args *ConnectionQueryArgs,
) (*UserSessionConnectionResolver, error) {
	if err := args.Validate(); err != nil {
		return nil, err
	}

	input := user.GetUserSessionsInput{
		PaginationOptions: &pagination.Options{First: args.First, Last: args.Last, After: args.After, Before: args.Before},
		UserID:            r.user.Metadata.ID,
	}

	if args.Sort != nil {
		sort := db.UserSessionSortableField(*args.Sort)
		input.Sort = &sort
	}

	return NewUserSessionConnectionResolver(ctx, &input)
}

func usersQuery(ctx context.Context, args *UserConnectionQueryArgs) (*UserConnectionResolver, error) {
	if err := args.Validate(); err != nil {
		return nil, err
	}

	input := user.GetUsersInput{
		PaginationOptions: &pagination.Options{First: args.First, Last: args.Last, After: args.After, Before: args.Before},
		Search:            args.Search,
	}

	if args.Sort != nil {
		sort := db.UserSortableField(*args.Sort)
		input.Sort = &sort
	}

	return NewUserConnectionResolver(ctx, &input)
}

/* User Mutation Resolvers */

// UserMutationPayload is the response payload for a user mutation.
type UserMutationPayload struct {
	ClientMutationID *string
	User             *models.User
	Problems         []Problem
}

// RevokeUserSessionPayload is the response payload for revoking a user session.
type RevokeUserSessionPayload struct {
	ClientMutationID *string
	Problems         []Problem
}

// UserMutationPayloadResolver resolves a UserMutationPayload
type UserMutationPayloadResolver struct {
	UserMutationPayload
}

// RevokeUserSessionPayloadResolver resolves a RevokeUserSessionPayload
type RevokeUserSessionPayloadResolver struct {
	RevokeUserSessionPayload
}

// User field resolver
func (r *UserMutationPayloadResolver) User() *UserResolver {
	if r.UserMutationPayload.User == nil {
		return nil
	}

	return &UserResolver{user: r.UserMutationPayload.User}
}

// UpdateUserAdminStatusInput is the input for updating users as admins.
type UpdateUserAdminStatusInput struct {
	ClientMutationID *string
	UserID           string
	Admin            bool
}

// RevokeUserSessionInput is the input for revoking a user session.
type RevokeUserSessionInput struct {
	ClientMutationID *string
	UserSessionID    string
}

// CreateUserInput is the input for creating a user.
type CreateUserInput struct {
	ClientMutationID *string
	Username         string
	Email            string
	Password         *string
	Admin            bool
}

// DeleteUserInput is the input for deleting a user.
type DeleteUserInput struct {
	ClientMutationID *string
	UserID           string
}

// SetUserPasswordInput is the input for setting a user's password.
type SetUserPasswordInput struct {
	ClientMutationID *string
	UserID           string
	CurrentPassword  string
	NewPassword      string
}

func handleUserMutationProblem(e error, clientMutationID *string) (*UserMutationPayloadResolver, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}
	payload := UserMutationPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}
	return &UserMutationPayloadResolver{UserMutationPayload: payload}, nil
}

func updateUserAdminStatusMutation(ctx context.Context, input *UpdateUserAdminStatusInput) (*UserMutationPayloadResolver, error) {
	serviceCatalog := getServiceCatalog(ctx)

	userID, err := serviceCatalog.FetchModelID(ctx, input.UserID)
	if err != nil {
		return nil, err
	}

	user, err := serviceCatalog.UserService.UpdateAdminStatusForUser(ctx, &user.UpdateAdminStatusForUserInput{
		UserID: userID,
		Admin:  input.Admin,
	})
	if err != nil {
		return nil, err
	}

	payload := UserMutationPayload{ClientMutationID: input.ClientMutationID, User: user, Problems: []Problem{}}
	return &UserMutationPayloadResolver{UserMutationPayload: payload}, nil
}

func handleRevokeUserSessionProblem(e error, clientMutationID *string) (*RevokeUserSessionPayloadResolver, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}
	payload := RevokeUserSessionPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}
	return &RevokeUserSessionPayloadResolver{RevokeUserSessionPayload: payload}, nil
}

func revokeUserSessionMutation(ctx context.Context, input *RevokeUserSessionInput) (*RevokeUserSessionPayloadResolver, error) {
	serviceCatalog := getServiceCatalog(ctx)

	userSessionID, err := serviceCatalog.FetchModelID(ctx, input.UserSessionID)
	if err != nil {
		return nil, err
	}

	err = serviceCatalog.UserService.RevokeUserSession(ctx, &user.RevokeUserSessionInput{
		UserSessionID: userSessionID,
	})
	if err != nil {
		return nil, err
	}

	payload := RevokeUserSessionPayload{ClientMutationID: input.ClientMutationID, Problems: []Problem{}}
	return &RevokeUserSessionPayloadResolver{RevokeUserSessionPayload: payload}, nil
}

func createUserMutation(ctx context.Context, input *CreateUserInput) (*UserMutationPayloadResolver, error) {
	serviceCatalog := getServiceCatalog(ctx)

	user, err := serviceCatalog.UserService.CreateUser(ctx, &user.CreateUserInput{
		Username: input.Username,
		Email:    input.Email,
		Password: input.Password,
		Admin:    input.Admin,
	})
	if err != nil {
		return nil, err
	}

	payload := UserMutationPayload{ClientMutationID: input.ClientMutationID, User: user, Problems: []Problem{}}
	return &UserMutationPayloadResolver{UserMutationPayload: payload}, nil
}

func deleteUserMutation(ctx context.Context, input *DeleteUserInput) (*UserMutationPayloadResolver, error) {
	serviceCatalog := getServiceCatalog(ctx)

	userID, err := serviceCatalog.FetchModelID(ctx, input.UserID)
	if err != nil {
		return nil, err
	}

	err = serviceCatalog.UserService.DeleteUser(ctx, &user.DeleteUserInput{
		UserID: userID,
	})
	if err != nil {
		return nil, err
	}

	payload := UserMutationPayload{ClientMutationID: input.ClientMutationID, Problems: []Problem{}}
	return &UserMutationPayloadResolver{UserMutationPayload: payload}, nil
}

func setUserPasswordMutation(ctx context.Context, input *SetUserPasswordInput) (*UserMutationPayloadResolver, error) {
	serviceCatalog := getServiceCatalog(ctx)

	userID, err := serviceCatalog.FetchModelID(ctx, input.UserID)
	if err != nil {
		return nil, err
	}

	user, err := serviceCatalog.UserService.SetUserPassword(ctx, &user.SetUserPasswordInput{
		UserID:          userID,
		CurrentPassword: input.CurrentPassword,
		NewPassword:     input.NewPassword,
	})
	if err != nil {
		return nil, err
	}

	payload := UserMutationPayload{ClientMutationID: input.ClientMutationID, User: user, Problems: []Problem{}}
	return &UserMutationPayloadResolver{UserMutationPayload: payload}, nil
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
		return nil, errors.New("Wrong type")
	}

	return &user, nil
}

func userBatchFunc(ctx context.Context, ids []string) (loader.DataBatch, error) {
	users, err := getServiceCatalog(ctx).UserService.GetUsersByIDs(ctx, ids)
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
