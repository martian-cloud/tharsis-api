package resolver

import (
	"context"

	"github.com/aws/smithy-go/ptr"
	graphql "github.com/graph-gophers/graphql-go"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/namespace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/group"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/user"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/workspace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// UserNamespacePreferenceConnectionQueryArgs are used to query a user namespace preference connection
type UserNamespacePreferenceConnectionQueryArgs struct {
	ConnectionQueryArgs
	Path *string
}

// UserNamespacePreferenceEdgeResolver resolves user namespace preference edges
type UserNamespacePreferenceEdgeResolver struct {
	edge Edge
}

// Cursor returns an opaque cursor
func (r *UserNamespacePreferenceEdgeResolver) Cursor() (string, error) {
	pref, ok := r.edge.Node.(*UserNamespacePreferencesResolver)
	if !ok {
		return "", errors.New("Failed to convert node type")
	}
	cursor, err := r.edge.CursorFunc(pref)
	return *cursor, err
}

// Node returns a user namespace preference node
func (r *UserNamespacePreferenceEdgeResolver) Node() (*UserNamespacePreferencesResolver, error) {
	node, ok := r.edge.Node.(*UserNamespacePreferencesResolver)
	if !ok {
		return nil, errors.New("Failed to convert node type")
	}

	return node, nil
}

// UserNamespacePreferenceConnectionResolver resolves a role connection
type UserNamespacePreferenceConnectionResolver struct {
	connection Connection
}

// NewUserGroupPreferenceConnectionResolver creates a new namespace connection
func NewUserGroupPreferenceConnectionResolver(ctx context.Context, input *group.GetGroupsInput) (*UserNamespacePreferenceConnectionResolver, error) {
	service := getServiceCatalog(ctx).GroupService

	result, err := service.GetGroups(ctx, input)
	if err != nil {
		return nil, err
	}

	namespaces := result.Groups

	// Create edges
	edges := make([]Edge, len(namespaces))
	for i, ns := range namespaces {
		ns := ns
		edges[i] = Edge{CursorFunc: result.PageInfo.Cursor, Node: &UserNamespacePreferencesResolver{namespace: &ns}}
	}

	pageInfo := PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
	}

	if len(namespaces) > 0 {
		var err error
		pageInfo.StartCursor, err = result.PageInfo.Cursor(&namespaces[0])
		if err != nil {
			return nil, err
		}

		pageInfo.EndCursor, err = result.PageInfo.Cursor(&namespaces[len(edges)-1])
		if err != nil {
			return nil, err
		}
	}

	connection := Connection{
		TotalCount: result.PageInfo.TotalCount,
		PageInfo:   pageInfo,
		Edges:      edges,
	}

	return &UserNamespacePreferenceConnectionResolver{connection: connection}, nil
}

// NewUserWorkspacePreferenceConnectionResolver creates a new namespace connection
func NewUserWorkspacePreferenceConnectionResolver(ctx context.Context, input *workspace.GetWorkspacesInput) (*UserNamespacePreferenceConnectionResolver, error) {
	service := getServiceCatalog(ctx).WorkspaceService

	result, err := service.GetWorkspaces(ctx, input)
	if err != nil {
		return nil, err
	}

	namespaces := result.Workspaces

	// Create edges
	edges := make([]Edge, len(namespaces))
	for i, ns := range namespaces {
		ns := ns
		edges[i] = Edge{CursorFunc: result.PageInfo.Cursor, Node: &UserNamespacePreferencesResolver{namespace: &ns}}
	}

	pageInfo := PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
	}

	if len(namespaces) > 0 {
		var err error
		pageInfo.StartCursor, err = result.PageInfo.Cursor(&namespaces[0])
		if err != nil {
			return nil, err
		}

		pageInfo.EndCursor, err = result.PageInfo.Cursor(&namespaces[len(edges)-1])
		if err != nil {
			return nil, err
		}
	}

	connection := Connection{
		TotalCount: result.PageInfo.TotalCount,
		PageInfo:   pageInfo,
		Edges:      edges,
	}

	return &UserNamespacePreferenceConnectionResolver{connection: connection}, nil
}

// TotalCount returns the total result count for the connection
func (r *UserNamespacePreferenceConnectionResolver) TotalCount() int32 {
	return r.connection.TotalCount
}

// PageInfo returns the connection page information
func (r *UserNamespacePreferenceConnectionResolver) PageInfo() *PageInfoResolver {
	return &PageInfoResolver{pageInfo: r.connection.PageInfo}
}

// Edges returns the connection edges
func (r *UserNamespacePreferenceConnectionResolver) Edges() *[]*UserNamespacePreferenceEdgeResolver {
	resolvers := make([]*UserNamespacePreferenceEdgeResolver, len(r.connection.Edges))
	for i, edge := range r.connection.Edges {
		resolvers[i] = &UserNamespacePreferenceEdgeResolver{edge: edge}
	}
	return &resolvers
}

// UserNamespacePreferencesResolver resolves a user namespace preference
type UserNamespacePreferencesResolver struct {
	namespace namespace.Namespace
}

// ID resolver
func (r *UserNamespacePreferencesResolver) ID() graphql.ID {
	return graphql.ID(r.namespace.GetGlobalID())
}

// Path resolver
func (r *UserNamespacePreferencesResolver) Path() string {
	return r.namespace.GetPath()
}

// NotificationPreference resolver
func (r *UserNamespacePreferencesResolver) NotificationPreference(ctx context.Context) (*UserNotificationPreferenceResolver, error) {
	pref, err := getServiceCatalog(ctx).UserService.GetNotificationPreference(ctx, &user.GetNotificationPreferenceInput{
		NamespacePath: ptr.String(r.namespace.GetPath()),
	})
	if err != nil {
		return nil, err
	}
	return &UserNotificationPreferenceResolver{preference: pref}, nil
}

// ResolveMetadata is used to resolve metadata for the namespace
func (r *UserNamespacePreferencesResolver) ResolveMetadata(key string) (string, error) {
	return r.namespace.ResolveMetadata(key)
}

// UserNotificationPreferenceResolver resolves a user notification preference
type UserNotificationPreferenceResolver struct {
	preference *namespace.NotificationPreferenceSetting
}

// Inherited resolver
func (r *UserNotificationPreferenceResolver) Inherited() bool {
	return r.preference.Inherited
}

// Global resolver
func (r *UserNotificationPreferenceResolver) Global() bool {
	return r.preference.NamespacePath == nil
}

// NamespacePath resolver
func (r *UserNotificationPreferenceResolver) NamespacePath() *string {
	return r.preference.NamespacePath
}

// Scope resolver
func (r *UserNotificationPreferenceResolver) Scope() models.NotificationPreferenceScope {
	return r.preference.Scope
}

// CustomEvents resolver
func (r *UserNotificationPreferenceResolver) CustomEvents() *models.NotificationPreferenceCustomEvents {
	return r.preference.CustomEvents
}

// GlobalUserPreferencesResolver resolves global user preferences
type GlobalUserPreferencesResolver struct{}

// NotificationPreference resolver
func (r *GlobalUserPreferencesResolver) NotificationPreference(ctx context.Context) (*UserNotificationPreferenceResolver, error) {
	pref, err := getServiceCatalog(ctx).UserService.GetNotificationPreference(ctx, &user.GetNotificationPreferenceInput{})
	if err != nil {
		return nil, err
	}
	return &UserNotificationPreferenceResolver{preference: pref}, nil
}

// UserPreferencesResolver resolves user preferences
type UserPreferencesResolver struct{}

// GlobalPreferences resolver
func (r *UserPreferencesResolver) GlobalPreferences() (*GlobalUserPreferencesResolver, error) {
	return &GlobalUserPreferencesResolver{}, nil
}

// GroupPreferences resolver
func (r *UserPreferencesResolver) GroupPreferences(ctx context.Context, args *UserNamespacePreferenceConnectionQueryArgs) (*UserNamespacePreferenceConnectionResolver, error) {
	if err := args.Validate(); err != nil {
		return nil, err
	}

	sort := db.GroupSortableFieldFullPathAsc
	input := group.GetGroupsInput{
		PaginationOptions: &pagination.Options{First: args.First, Last: args.Last, After: args.After, Before: args.Before},
		Sort:              &sort,
		GroupPath:         args.Path,
	}

	return NewUserGroupPreferenceConnectionResolver(ctx, &input)
}

// WorkspacePreferences resolver
func (r *UserPreferencesResolver) WorkspacePreferences(ctx context.Context, args *UserNamespacePreferenceConnectionQueryArgs) (*UserNamespacePreferenceConnectionResolver, error) {
	if err := args.Validate(); err != nil {
		return nil, err
	}

	sort := db.WorkspaceSortableFieldFullPathAsc
	input := workspace.GetWorkspacesInput{
		PaginationOptions: &pagination.Options{First: args.First, Last: args.Last, After: args.After, Before: args.Before},
		Sort:              &sort,
		WorkspacePath:     args.Path,
	}

	return NewUserWorkspacePreferenceConnectionResolver(ctx, &input)
}

func userPreferencesQuery() (*UserPreferencesResolver, error) {
	return &UserPreferencesResolver{}, nil
}

/* User Preference Mutation Resolvers */

// UserNotificationPreferenceMutationPayload is the response payload for a user preference mutation
type UserNotificationPreferenceMutationPayload struct {
	ClientMutationID *string
	Preference       *namespace.NotificationPreferenceSetting
	Problems         []Problem
}

// UserNotificationPreferenceMutationPayloadResolver resolves a UserNotificationPreferenceMutationPayload
type UserNotificationPreferenceMutationPayloadResolver struct {
	UserNotificationPreferenceMutationPayload
}

// Preference resolver
func (r *UserNotificationPreferenceMutationPayloadResolver) Preference() *UserNotificationPreferenceResolver {
	if r.UserNotificationPreferenceMutationPayload.Preference == nil {
		return nil
	}
	return &UserNotificationPreferenceResolver{preference: r.UserNotificationPreferenceMutationPayload.Preference}
}

// SetUserNotificationPreferenceInput contains the input for setting a user notification preference
type SetUserNotificationPreferenceInput struct {
	ClientMutationID *string
	Inherit          *bool
	NamespacePath    *string
	Scope            *models.NotificationPreferenceScope
	CustomEvents     *models.NotificationPreferenceCustomEvents
}

func handleUserNotificationPreferenceMutationProblem(e error, clientMutationID *string) (*UserNotificationPreferenceMutationPayloadResolver, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}

	payload := UserNotificationPreferenceMutationPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}
	return &UserNotificationPreferenceMutationPayloadResolver{UserNotificationPreferenceMutationPayload: payload}, nil
}

func setUserNotificationPreferenceMutation(ctx context.Context, input *SetUserNotificationPreferenceInput) (*UserNotificationPreferenceMutationPayloadResolver, error) {
	setPreferenceOptions := &user.SetNotificationPreferenceInput{
		NamespacePath: input.NamespacePath,
		Scope:         input.Scope,
		CustomEvents:  input.CustomEvents,
	}

	if input.Inherit != nil {
		setPreferenceOptions.Inherit = *input.Inherit
	}

	setting, err := getServiceCatalog(ctx).UserService.SetNotificationPreference(ctx, setPreferenceOptions)
	if err != nil {
		return nil, err
	}

	payload := UserNotificationPreferenceMutationPayload{ClientMutationID: input.ClientMutationID, Preference: setting, Problems: []Problem{}}
	return &UserNotificationPreferenceMutationPayloadResolver{UserNotificationPreferenceMutationPayload: payload}, nil
}
