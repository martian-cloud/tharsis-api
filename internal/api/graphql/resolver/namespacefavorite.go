package resolver

import (
	"context"

	graphql "github.com/graph-gophers/graphql-go"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/namespace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/user"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// NamespaceFavoriteResolver resolves a namespace favorite
type NamespaceFavoriteResolver struct {
	namespaceFavorite *models.NamespaceFavorite
}

// NamespaceFavoritesConnectionQueryArgs are used to query namespace favorites
type NamespaceFavoritesConnectionQueryArgs struct {
	ConnectionQueryArgs
	NamespacePath *string
}

// ID resolver
func (r *NamespaceFavoriteResolver) ID() graphql.ID {
	return graphql.ID(r.namespaceFavorite.GetGlobalID())
}

// Metadata resolver
func (r *NamespaceFavoriteResolver) Metadata() *MetadataResolver {
	return &MetadataResolver{metadata: &r.namespaceFavorite.Metadata}
}

// Namespace resolver
func (r *NamespaceFavoriteResolver) Namespace(ctx context.Context) (*NamespaceResolver, error) {
	if r.namespaceFavorite.GroupID != nil {
		group, err := loadGroup(ctx, *r.namespaceFavorite.GroupID)
		if err != nil {
			return nil, err
		}
		return &NamespaceResolver{&GroupResolver{group: group}}, nil
	}

	workspace, err := loadWorkspace(ctx, *r.namespaceFavorite.WorkspaceID)
	if err != nil {
		return nil, err
	}
	return &NamespaceResolver{&WorkspaceResolver{workspace: workspace}}, nil
}

// ResolveMetadata is used to resolve metadata for the namespace favorite
func (r *NamespaceFavoriteResolver) ResolveMetadata(key string) (*string, error) {
	return r.namespaceFavorite.ResolveMetadata(key)
}

// NamespaceFavoriteEdgeResolver resolves namespace favorite edges
type NamespaceFavoriteEdgeResolver struct {
	edge Edge
}

// Cursor returns an opaque cursor
func (r *NamespaceFavoriteEdgeResolver) Cursor() (string, error) {
	favorite, ok := r.edge.Node.(models.NamespaceFavorite)
	if !ok {
		return "", errors.New("Failed to convert node type")
	}
	cursor, err := r.edge.CursorFunc(&favorite)
	return *cursor, err
}

// Node returns a namespace favorite node
func (r *NamespaceFavoriteEdgeResolver) Node() (*NamespaceFavoriteResolver, error) {
	favorite, ok := r.edge.Node.(models.NamespaceFavorite)
	if !ok {
		return nil, errors.New("Failed to convert node type")
	}
	return &NamespaceFavoriteResolver{namespaceFavorite: &favorite}, nil
}

// NamespaceFavoriteConnectionResolver resolves a namespace favorite connection
type NamespaceFavoriteConnectionResolver struct {
	connection Connection
}

// NewNamespaceFavoriteConnectionResolver creates a new NamespaceFavoriteConnectionResolver
func NewNamespaceFavoriteConnectionResolver(ctx context.Context, input *user.GetNamespaceFavoritesInput) (*NamespaceFavoriteConnectionResolver, error) {
	userService := getServiceCatalog(ctx).UserService

	result, err := userService.GetNamespaceFavorites(ctx, input)
	if err != nil {
		return nil, err
	}

	favorites := result.NamespaceFavorites

	// Create edges
	edges := make([]Edge, len(favorites))
	for i, favorite := range favorites {
		edges[i] = Edge{CursorFunc: result.PageInfo.Cursor, Node: favorite}
	}

	pageInfo := PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
	}

	if len(favorites) > 0 {
		var err error
		pageInfo.StartCursor, err = result.PageInfo.Cursor(&favorites[0])
		if err != nil {
			return nil, err
		}

		pageInfo.EndCursor, err = result.PageInfo.Cursor(&favorites[len(edges)-1])
		if err != nil {
			return nil, err
		}
	}

	connection := Connection{
		TotalCount: result.PageInfo.TotalCount,
		PageInfo:   pageInfo,
		Edges:      edges,
	}

	return &NamespaceFavoriteConnectionResolver{connection: connection}, nil
}

// TotalCount returns the total result count for the connection
func (r *NamespaceFavoriteConnectionResolver) TotalCount() int32 {
	return r.connection.TotalCount
}

// PageInfo returns the connection page information
func (r *NamespaceFavoriteConnectionResolver) PageInfo() *PageInfoResolver {
	return &PageInfoResolver{pageInfo: r.connection.PageInfo}
}

// Edges returns the connection edges
func (r *NamespaceFavoriteConnectionResolver) Edges() *[]*NamespaceFavoriteEdgeResolver {
	resolvers := make([]*NamespaceFavoriteEdgeResolver, len(r.connection.Edges))
	for i, edge := range r.connection.Edges {
		resolvers[i] = &NamespaceFavoriteEdgeResolver{edge: edge}
	}
	return &resolvers
}

/* NamespaceFavorite Mutations */

// NamespaceFavoriteMutationPayload is the response payload for namespace favorite mutations
type NamespaceFavoriteMutationPayload struct {
	ClientMutationID  *string
	NamespaceFavorite *models.NamespaceFavorite
	Problems          []Problem
}

// NamespaceFavoriteMutationPayloadResolver resolves a NamespaceFavoriteMutationPayload
type NamespaceFavoriteMutationPayloadResolver struct {
	NamespaceFavoriteMutationPayload
}

// NamespaceFavorite resolver
func (r *NamespaceFavoriteMutationPayloadResolver) NamespaceFavorite() *NamespaceFavoriteResolver {
	if r.NamespaceFavoriteMutationPayload.NamespaceFavorite == nil {
		return nil
	}
	return &NamespaceFavoriteResolver{namespaceFavorite: r.NamespaceFavoriteMutationPayload.NamespaceFavorite}
}

// NamespaceUnfavoriteMutationPayload is the response payload for namespace unfavorite mutations
type NamespaceUnfavoriteMutationPayload struct {
	ClientMutationID *string
	Problems         []Problem
}

// NamespaceUnfavoriteMutationPayloadResolver resolves a NamespaceUnfavoriteMutationPayload
type NamespaceUnfavoriteMutationPayloadResolver struct {
	NamespaceUnfavoriteMutationPayload
}

// NamespaceFavoriteInput contains the input for namespace favorite mutations
type NamespaceFavoriteInput struct {
	ClientMutationID *string
	NamespacePath    string
	NamespaceType    namespace.Type
}

func handleNamespaceFavoriteMutationProblem(e error, clientMutationID *string) (*NamespaceFavoriteMutationPayloadResolver, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}

	payload := NamespaceFavoriteMutationPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}
	return &NamespaceFavoriteMutationPayloadResolver{NamespaceFavoriteMutationPayload: payload}, nil
}

func handleNamespaceUnfavoriteMutationProblem(e error, clientMutationID *string) (*NamespaceUnfavoriteMutationPayloadResolver, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}

	payload := NamespaceUnfavoriteMutationPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}
	return &NamespaceUnfavoriteMutationPayloadResolver{NamespaceUnfavoriteMutationPayload: payload}, nil
}

func favoriteNamespaceMutation(ctx context.Context, input *NamespaceFavoriteInput) (*NamespaceFavoriteMutationPayloadResolver, error) {
	serviceCatalog := getServiceCatalog(ctx)
	favorite, err := serviceCatalog.UserService.CreateNamespaceFavorite(ctx, &user.CreateNamespaceFavoriteInput{
		NamespacePath: input.NamespacePath,
		NamespaceType: input.NamespaceType,
	})
	if err != nil {
		return nil, err
	}

	payload := NamespaceFavoriteMutationPayload{ClientMutationID: input.ClientMutationID, NamespaceFavorite: favorite, Problems: []Problem{}}
	return &NamespaceFavoriteMutationPayloadResolver{NamespaceFavoriteMutationPayload: payload}, nil
}

func unfavoriteNamespaceMutation(ctx context.Context, input *NamespaceFavoriteInput) (*NamespaceUnfavoriteMutationPayloadResolver, error) {
	serviceCatalog := getServiceCatalog(ctx)
	err := serviceCatalog.UserService.DeleteNamespaceFavorite(ctx, &user.DeleteNamespaceFavoriteInput{
		NamespacePath: input.NamespacePath,
		NamespaceType: input.NamespaceType,
	})
	if err != nil {
		return nil, err
	}

	payload := NamespaceUnfavoriteMutationPayload{ClientMutationID: input.ClientMutationID, Problems: []Problem{}}
	return &NamespaceUnfavoriteMutationPayloadResolver{NamespaceUnfavoriteMutationPayload: payload}, nil
}
