package resolver

import (
	"context"
	"fmt"
	"strconv"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/loader"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/providerregistry"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"

	"github.com/aws/smithy-go/ptr"
	"github.com/graph-gophers/dataloader"
	graphql "github.com/graph-gophers/graphql-go"
)

/* TerraformProvider Query Resolvers */

// TerraformProviderConnectionQueryArgs are used to query a provider connection
type TerraformProviderConnectionQueryArgs struct {
	ConnectionQueryArgs
	Search *string
}

// TerraformProviderQueryArgs are used to query a terraform provider
// DEPRECATED: use node query instead with a TRN
type TerraformProviderQueryArgs struct {
	RegistryNamespace string
	ProviderName      string
}

// TerraformProviderEdgeResolver resolves provider edges
type TerraformProviderEdgeResolver struct {
	edge Edge
}

// Cursor returns an opaque cursor
func (r *TerraformProviderEdgeResolver) Cursor() (string, error) {
	provider, ok := r.edge.Node.(models.TerraformProvider)
	if !ok {
		return "", errors.New("Failed to convert node type")
	}
	cursor, err := r.edge.CursorFunc(&provider)
	return *cursor, err
}

// Node returns a provider node
func (r *TerraformProviderEdgeResolver) Node() (*TerraformProviderResolver, error) {
	provider, ok := r.edge.Node.(models.TerraformProvider)
	if !ok {
		return nil, errors.New("Failed to convert node type")
	}

	return &TerraformProviderResolver{provider: &provider}, nil
}

// TerraformProviderConnectionResolver resolves a provider connection
type TerraformProviderConnectionResolver struct {
	connection Connection
}

// NewTerraformProviderConnectionResolver creates a new TerraformProviderConnectionResolver
func NewTerraformProviderConnectionResolver(ctx context.Context, input *providerregistry.GetProvidersInput) (*TerraformProviderConnectionResolver, error) {
	service := getServiceCatalog(ctx).TerraformProviderRegistryService

	result, err := service.GetProviders(ctx, input)
	if err != nil {
		return nil, err
	}

	providers := result.Providers

	// Create edges
	edges := make([]Edge, len(providers))
	for i, provider := range providers {
		edges[i] = Edge{CursorFunc: result.PageInfo.Cursor, Node: provider}
	}

	pageInfo := PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
	}

	if len(providers) > 0 {
		var err error
		pageInfo.StartCursor, err = result.PageInfo.Cursor(&providers[0])
		if err != nil {
			return nil, err
		}

		pageInfo.EndCursor, err = result.PageInfo.Cursor(&providers[len(edges)-1])
		if err != nil {
			return nil, err
		}
	}

	connection := Connection{
		TotalCount: result.PageInfo.TotalCount,
		PageInfo:   pageInfo,
		Edges:      edges,
	}

	return &TerraformProviderConnectionResolver{connection: connection}, nil
}

// TotalCount returns the total result count for the connection
func (r *TerraformProviderConnectionResolver) TotalCount() int32 {
	return r.connection.TotalCount
}

// PageInfo returns the connection page information
func (r *TerraformProviderConnectionResolver) PageInfo() *PageInfoResolver {
	return &PageInfoResolver{pageInfo: r.connection.PageInfo}
}

// Edges returns the connection edges
func (r *TerraformProviderConnectionResolver) Edges() *[]*TerraformProviderEdgeResolver {
	resolvers := make([]*TerraformProviderEdgeResolver, len(r.connection.Edges))
	for i, edge := range r.connection.Edges {
		resolvers[i] = &TerraformProviderEdgeResolver{edge: edge}
	}
	return &resolvers
}

// TerraformProviderResolver resolves a provider resource
type TerraformProviderResolver struct {
	provider *models.TerraformProvider
}

// ID resolver
func (r *TerraformProviderResolver) ID() graphql.ID {
	return graphql.ID(r.provider.GetGlobalID())
}

// Name resolver
func (r *TerraformProviderResolver) Name() string {
	return string(r.provider.Name)
}

// Private resolver
func (r *TerraformProviderResolver) Private() bool {
	return r.provider.Private
}

// CreatedBy resolver
func (r *TerraformProviderResolver) CreatedBy() string {
	return r.provider.CreatedBy
}

// GroupPath resolver
func (r *TerraformProviderResolver) GroupPath() string {
	return r.provider.GetGroupPath()
}

// ResourcePath resolver
func (r *TerraformProviderResolver) ResourcePath() string {
	return r.provider.GetResourcePath()
}

// RepositoryURL resolver
func (r *TerraformProviderResolver) RepositoryURL() string {
	return r.provider.RepositoryURL
}

// RegistryNamespace resolver
func (r *TerraformProviderResolver) RegistryNamespace() string {
	return r.provider.GetRegistryNamespace()
}

// Source resolver
func (r *TerraformProviderResolver) Source(ctx context.Context) string {
	cfg := getConfig(ctx)
	return fmt.Sprintf("%s/%s/%s", cfg.ServiceDiscoveryHost, r.provider.GetRegistryNamespace(), r.provider.Name)
}

// Metadata resolver
func (r *TerraformProviderResolver) Metadata() *MetadataResolver {
	return &MetadataResolver{metadata: &r.provider.Metadata}
}

// Group resolver
func (r *TerraformProviderResolver) Group(ctx context.Context) (*GroupResolver, error) {
	group, err := loadGroup(ctx, r.provider.GroupID)
	if err != nil {
		return nil, err
	}
	return &GroupResolver{group: group}, nil
}

// Versions resolver
func (r *TerraformProviderResolver) Versions(ctx context.Context, args *TerraformProviderVersionsConnectionQueryArgs) (*TerraformProviderVersionConnectionResolver, error) {
	input := &providerregistry.GetProviderVersionsInput{
		PaginationOptions: &pagination.Options{
			First:  args.First,
			Last:   args.Last,
			Before: args.Before,
			After:  args.After,
		},
		ProviderID: r.provider.Metadata.ID,
	}

	if args.Sort != nil {
		sort := db.TerraformProviderVersionSortableField(*args.Sort)
		input.Sort = &sort
	}

	return NewTerraformProviderVersionConnectionResolver(ctx, input)
}

// LatestVersion resolver
func (r *TerraformProviderResolver) LatestVersion(ctx context.Context) (*TerraformProviderVersionResolver, error) {
	versionsResp, err := getServiceCatalog(ctx).TerraformProviderRegistryService.GetProviderVersions(ctx, &providerregistry.GetProviderVersionsInput{
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(1),
		},
		ProviderID: r.provider.Metadata.ID,
		Latest:     ptr.Bool(true),
	})
	if err != nil {
		return nil, err
	}

	if len(versionsResp.ProviderVersions) == 0 {
		return nil, nil
	}

	return &TerraformProviderVersionResolver{providerVersion: &versionsResp.ProviderVersions[0]}, nil
}

func terraformProvidersQuery(ctx context.Context, args *TerraformProviderConnectionQueryArgs) (*TerraformProviderConnectionResolver, error) {
	if err := args.Validate(); err != nil {
		return nil, err
	}

	input := providerregistry.GetProvidersInput{
		PaginationOptions: &pagination.Options{First: args.First, Last: args.Last, After: args.After, Before: args.Before},
		Search:            args.Search,
	}

	if args.Sort != nil {
		sort := db.TerraformProviderSortableField(*args.Sort)
		input.Sort = &sort
	}

	return NewTerraformProviderConnectionResolver(ctx, &input)
}

// DEPRECATED: use node query instead
func terraformProviderQuery(ctx context.Context, args *TerraformProviderQueryArgs) (*TerraformProviderResolver, error) {
	provider, err := getServiceCatalog(ctx).TerraformProviderRegistryService.GetProviderByAddress(ctx, args.RegistryNamespace, args.ProviderName)
	if err != nil {
		if errors.ErrorCode(err) == errors.ENotFound {
			return nil, nil
		}
		return nil, err
	}
	return &TerraformProviderResolver{provider: provider}, nil
}

/* TerraformProvider Mutation Resolvers */

// TerraformProviderMutationPayload is the response payload for provider mutation
type TerraformProviderMutationPayload struct {
	ClientMutationID *string
	Provider         *models.TerraformProvider
	Problems         []Problem
}

// TerraformProviderMutationPayloadResolver resolves a TerraformProviderMutationPayload
type TerraformProviderMutationPayloadResolver struct {
	TerraformProviderMutationPayload
}

// Provider field resolver
func (r *TerraformProviderMutationPayloadResolver) Provider() *TerraformProviderResolver {
	if r.TerraformProviderMutationPayload.Provider == nil {
		return nil
	}
	return &TerraformProviderResolver{provider: r.TerraformProviderMutationPayload.Provider}
}

// UpdateTerraformProviderInput contains the input for updating a provider
type UpdateTerraformProviderInput struct {
	ClientMutationID *string
	Metadata         *MetadataInput
	RepositoryURL    *string
	Private          *bool
	ID               string
}

// CreateTerraformProviderInput contains the input for creating a provider
type CreateTerraformProviderInput struct {
	ClientMutationID *string
	Private          *bool
	RepositoryURL    *string
	Name             string
	GroupPath        *string // DEPRECATED: use GroupID instead with a TRN
	GroupID          *string
}

// DeleteTerraformProviderInput contains the input for deleting a provider
type DeleteTerraformProviderInput struct {
	ClientMutationID *string
	ID               string
}

func handleTerraformProviderMutationProblem(e error, clientMutationID *string) (*TerraformProviderMutationPayloadResolver, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}
	payload := TerraformProviderMutationPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}
	return &TerraformProviderMutationPayloadResolver{TerraformProviderMutationPayload: payload}, nil
}

func createTerraformProviderMutation(ctx context.Context, input *CreateTerraformProviderInput) (*TerraformProviderMutationPayloadResolver, error) {
	groupID, err := toModelID(ctx, input.GroupPath, input.GroupID, types.GroupModelType)
	if err != nil {
		return nil, err
	}

	createOptions := providerregistry.CreateProviderInput{
		GroupID: groupID,
		Name:    input.Name,
		Private: true,
	}

	if input.Private != nil {
		createOptions.Private = *input.Private
	}

	if input.RepositoryURL != nil {
		createOptions.RepositoryURL = *input.RepositoryURL
	}

	provider, err := getServiceCatalog(ctx).TerraformProviderRegistryService.CreateProvider(ctx, &createOptions)
	if err != nil {
		return nil, err
	}

	payload := TerraformProviderMutationPayload{ClientMutationID: input.ClientMutationID, Provider: provider, Problems: []Problem{}}
	return &TerraformProviderMutationPayloadResolver{TerraformProviderMutationPayload: payload}, nil
}

func updateTerraformProviderMutation(ctx context.Context, input *UpdateTerraformProviderInput) (*TerraformProviderMutationPayloadResolver, error) {
	serviceCatalog := getServiceCatalog(ctx)

	providerID, err := serviceCatalog.FetchModelID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	provider, err := serviceCatalog.TerraformProviderRegistryService.GetProviderByID(ctx, providerID)
	if err != nil {
		return nil, err
	}

	// Check if resource version is specified
	if input.Metadata != nil {
		v, cErr := strconv.Atoi(input.Metadata.Version)
		if cErr != nil {
			return nil, cErr
		}

		provider.Metadata.Version = v
	}

	// Update fields
	if input.Private != nil {
		provider.Private = *input.Private
	}

	if input.RepositoryURL != nil {
		provider.RepositoryURL = *input.RepositoryURL
	}

	provider, err = serviceCatalog.TerraformProviderRegistryService.UpdateProvider(ctx, provider)
	if err != nil {
		return nil, err
	}

	payload := TerraformProviderMutationPayload{ClientMutationID: input.ClientMutationID, Provider: provider, Problems: []Problem{}}
	return &TerraformProviderMutationPayloadResolver{TerraformProviderMutationPayload: payload}, nil
}

func deleteTerraformProviderMutation(ctx context.Context, input *DeleteTerraformProviderInput) (*TerraformProviderMutationPayloadResolver, error) {
	serviceCatalog := getServiceCatalog(ctx)

	providerID, err := serviceCatalog.FetchModelID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	provider, err := serviceCatalog.TerraformProviderRegistryService.GetProviderByID(ctx, providerID)
	if err != nil {
		return nil, err
	}

	if err := serviceCatalog.TerraformProviderRegistryService.DeleteProvider(ctx, provider); err != nil {
		return nil, err
	}

	payload := TerraformProviderMutationPayload{ClientMutationID: input.ClientMutationID, Provider: provider, Problems: []Problem{}}
	return &TerraformProviderMutationPayloadResolver{TerraformProviderMutationPayload: payload}, nil
}

/* TerraformProvider loader */

const providerLoaderKey = "terraformProvider"

// RegisterTerraformProviderLoader registers a provider loader function
func RegisterTerraformProviderLoader(collection *loader.Collection) {
	collection.Register(providerLoaderKey, providerBatchFunc)
}

func loadTerraformProvider(ctx context.Context, id string) (*models.TerraformProvider, error) {
	ldr, err := loader.Extract(ctx, providerLoaderKey)
	if err != nil {
		return nil, err
	}

	data, err := ldr.Load(ctx, dataloader.StringKey(id))()
	if err != nil {
		return nil, err
	}

	ws, ok := data.(models.TerraformProvider)
	if !ok {
		return nil, errors.New("Wrong type")
	}

	return &ws, nil
}

func providerBatchFunc(ctx context.Context, ids []string) (loader.DataBatch, error) {
	providers, err := getServiceCatalog(ctx).TerraformProviderRegistryService.GetProvidersByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	// Build map of results
	batch := loader.DataBatch{}
	for _, result := range providers {
		batch[result.Metadata.ID] = result
	}

	return batch, nil
}
