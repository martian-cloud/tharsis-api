package resolver

import (
	"context"
	"strconv"

	"github.com/graph-gophers/dataloader"
	graphql "github.com/graph-gophers/graphql-go"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/loader"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/federatedregistry"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

/* FederatedRegistry Query Resolvers */

// FederatedRegistryConnectionQueryArgs are used to query a federated registry connection
type FederatedRegistryConnectionQueryArgs struct {
	ConnectionQueryArgs
}

// FederatedRegistryEdgeResolver resolves federated registry edges
type FederatedRegistryEdgeResolver struct {
	edge Edge
}

// Cursor returns an opaque cursor
func (r *FederatedRegistryEdgeResolver) Cursor() (string, error) {
	federatedRegistry, ok := r.edge.Node.(models.FederatedRegistry)
	if !ok {
		return "", errors.New("failed to convert node type")
	}
	cursor, err := r.edge.CursorFunc(&federatedRegistry)
	return *cursor, err
}

// Node returns a federatedRegistry node
func (r *FederatedRegistryEdgeResolver) Node() (*FederatedRegistryResolver, error) {
	federatedRegistry, ok := r.edge.Node.(models.FederatedRegistry)
	if !ok {
		return nil, errors.New("failed to convert node type")
	}

	return &FederatedRegistryResolver{federatedRegistry: &federatedRegistry}, nil
}

// FederatedRegistryConnectionResolver resolves a federated registry connection
type FederatedRegistryConnectionResolver struct {
	connection Connection
}

// NewFederatedRegistryConnectionResolver creates a new FederatedRegistryConnectionResolver
func NewFederatedRegistryConnectionResolver(ctx context.Context,
	input *federatedregistry.GetFederatedRegistriesInput,
) (*FederatedRegistryConnectionResolver, error) {
	federatedRegistryService := getServiceCatalog(ctx).FederatedRegistryService

	result, err := federatedRegistryService.GetFederatedRegistries(ctx, input)
	if err != nil {
		return nil, err
	}

	federatedRegistries := result.FederatedRegistries

	// Create edges
	edges := make([]Edge, len(federatedRegistries))
	for i, federatedRegistry := range federatedRegistries {
		edges[i] = Edge{CursorFunc: result.PageInfo.Cursor, Node: *federatedRegistry}
	}

	pageInfo := PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
	}

	if len(federatedRegistries) > 0 {
		var err error
		pageInfo.StartCursor, err = result.PageInfo.Cursor(federatedRegistries[0])
		if err != nil {
			return nil, err
		}

		pageInfo.EndCursor, err = result.PageInfo.Cursor(federatedRegistries[len(edges)-1])
		if err != nil {
			return nil, err
		}
	}

	connection := Connection{
		TotalCount: result.PageInfo.TotalCount,
		PageInfo:   pageInfo,
		Edges:      edges,
	}

	return &FederatedRegistryConnectionResolver{connection: connection}, nil
}

// TotalCount returns the total result count for the connection
func (r *FederatedRegistryConnectionResolver) TotalCount() int32 {
	return r.connection.TotalCount
}

// PageInfo returns the connection page information
func (r *FederatedRegistryConnectionResolver) PageInfo() *PageInfoResolver {
	return &PageInfoResolver{pageInfo: r.connection.PageInfo}
}

// Edges returns the connection edges
func (r *FederatedRegistryConnectionResolver) Edges() *[]*FederatedRegistryEdgeResolver {
	resolvers := make([]*FederatedRegistryEdgeResolver, len(r.connection.Edges))
	for i, edge := range r.connection.Edges {
		resolvers[i] = &FederatedRegistryEdgeResolver{edge: edge}
	}
	return &resolvers
}

// FederatedRegistryResolver resolves a federatedRegistry resource
type FederatedRegistryResolver struct {
	federatedRegistry *models.FederatedRegistry
}

// ID resolver
func (r *FederatedRegistryResolver) ID() graphql.ID {
	return graphql.ID(r.federatedRegistry.GetGlobalID())
}

// Metadata resolver
func (r *FederatedRegistryResolver) Metadata() *MetadataResolver {
	return &MetadataResolver{metadata: &r.federatedRegistry.Metadata}
}

// Hostname resolver
func (r *FederatedRegistryResolver) Hostname() string {
	return r.federatedRegistry.Hostname
}

// Audience resolver
func (r *FederatedRegistryResolver) Audience() string {
	return r.federatedRegistry.Audience
}

// CreatedBy resolver
func (r *FederatedRegistryResolver) CreatedBy() string {
	return r.federatedRegistry.CreatedBy
}

// Group resolver
func (r *FederatedRegistryResolver) Group(ctx context.Context) (*GroupResolver, error) {
	group, err := loadGroup(ctx, r.federatedRegistry.GroupID)
	if err != nil {
		return nil, err
	}

	return &GroupResolver{group: group}, nil
}

/* FederatedRegistry Mutation Resolvers */

// FederatedRegistryMutationPayload is the response payload for a federated registry mutation
type FederatedRegistryMutationPayload struct {
	ClientMutationID  *string
	FederatedRegistry *models.FederatedRegistry
	Problems          []Problem
}

// FederatedRegistryMutationPayloadResolver resolves a FederatedRegistryMutationPayload
type FederatedRegistryMutationPayloadResolver struct {
	FederatedRegistryMutationPayload
}

// FederatedRegistry field resolver
func (r *FederatedRegistryMutationPayloadResolver) FederatedRegistry() *FederatedRegistryResolver {
	if r.FederatedRegistryMutationPayload.FederatedRegistry == nil {
		return nil
	}
	return &FederatedRegistryResolver{federatedRegistry: r.FederatedRegistryMutationPayload.FederatedRegistry}
}

// CreateFederatedRegistryInput contains the input for creating a new federated registry
type CreateFederatedRegistryInput struct {
	ClientMutationID *string
	Hostname         string
	Audience         string
	GroupID          *string
	GroupPath        *string // DEPRECATED: use GroupID instead with a TRN
}

// UpdateFederatedRegistryInput contains the input for updating a federated registry
type UpdateFederatedRegistryInput struct {
	ClientMutationID *string
	Metadata         *MetadataInput
	ID               string
	Hostname         *string
	Audience         *string
}

// DeleteFederatedRegistryInput contains the input for deleting a federated registry
type DeleteFederatedRegistryInput struct {
	ClientMutationID *string
	Metadata         *MetadataInput
	ID               string
}

func handleFederatedRegistryMutationProblem(e error, clientMutationID *string) (*FederatedRegistryMutationPayloadResolver, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}
	payload := FederatedRegistryMutationPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}
	return &FederatedRegistryMutationPayloadResolver{FederatedRegistryMutationPayload: payload}, nil
}

func createFederatedRegistryMutation(ctx context.Context,
	input *CreateFederatedRegistryInput,
) (*FederatedRegistryMutationPayloadResolver, error) {
	groupID, err := toModelID(ctx, input.GroupPath, input.GroupID, types.GroupModelType)
	if err != nil {
		return nil, err
	}

	federatedRegistryCreateOptions := models.FederatedRegistry{
		Hostname: input.Hostname,
		Audience: input.Audience,
		GroupID:  groupID,
	}

	createdFederatedRegistry, err := getServiceCatalog(ctx).FederatedRegistryService.CreateFederatedRegistry(ctx,
		&federatedRegistryCreateOptions,
	)
	if err != nil {
		return nil, err
	}

	payload := FederatedRegistryMutationPayload{ClientMutationID: input.ClientMutationID, FederatedRegistry: createdFederatedRegistry, Problems: []Problem{}}
	return &FederatedRegistryMutationPayloadResolver{FederatedRegistryMutationPayload: payload}, nil
}

func updateFederatedRegistryMutation(ctx context.Context,
	input *UpdateFederatedRegistryInput,
) (*FederatedRegistryMutationPayloadResolver, error) {
	serviceCatalog := getServiceCatalog(ctx)

	registryID, err := serviceCatalog.FetchModelID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	federatedRegistry, err := serviceCatalog.FederatedRegistryService.GetFederatedRegistryByID(ctx, registryID)
	if err != nil {
		return nil, err
	}

	// Check if resource version is specified
	if input.Metadata != nil {
		v, cErr := strconv.Atoi(input.Metadata.Version)
		if cErr != nil {
			return nil, cErr
		}

		federatedRegistry.Metadata.Version = v
	}

	// Update fields
	if input.Hostname != nil {
		federatedRegistry.Hostname = *input.Hostname
	}

	if input.Audience != nil {
		federatedRegistry.Audience = *input.Audience
	}

	federatedRegistry, err = serviceCatalog.FederatedRegistryService.UpdateFederatedRegistry(ctx, federatedRegistry)
	if err != nil {
		return nil, err
	}

	payload := FederatedRegistryMutationPayload{ClientMutationID: input.ClientMutationID, FederatedRegistry: federatedRegistry, Problems: []Problem{}}
	return &FederatedRegistryMutationPayloadResolver{FederatedRegistryMutationPayload: payload}, nil
}

func deleteFederatedRegistryMutation(ctx context.Context,
	input *DeleteFederatedRegistryInput,
) (*FederatedRegistryMutationPayloadResolver, error) {
	serviceCatalog := getServiceCatalog(ctx)

	registryID, err := serviceCatalog.FetchModelID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	federatedRegistryToDelete, err := serviceCatalog.FederatedRegistryService.GetFederatedRegistryByID(ctx, registryID)
	if err != nil {
		return nil, err
	}

	// Check if resource version is specified
	if input.Metadata != nil {
		v, err := strconv.Atoi(input.Metadata.Version)
		if err != nil {
			return nil, err
		}

		federatedRegistryToDelete.Metadata.Version = v
	}

	err = serviceCatalog.FederatedRegistryService.DeleteFederatedRegistry(ctx, federatedRegistryToDelete)
	if err != nil {
		return nil, err
	}

	payload := FederatedRegistryMutationPayload{
		ClientMutationID:  input.ClientMutationID,
		FederatedRegistry: federatedRegistryToDelete,
		Problems:          []Problem{},
	}
	return &FederatedRegistryMutationPayloadResolver{FederatedRegistryMutationPayload: payload}, nil
}

/////////////////////////////////////////////////////////////////////////////

// CreateFederatedRegistryTokensInput contains the input for creating federated registry tokens
type CreateFederatedRegistryTokensInput struct {
	ClientMutationID *string
	JobID            string
}

/* FederatedRegistryTokens Mutation Resolvers */

// FederatedRegistryTokensMutationPayload is the response payload for federated registry token mutation
type FederatedRegistryTokensMutationPayload struct {
	ClientMutationID *string
	Tokens           []*federatedregistry.Token
	Problems         []Problem
}

// FederatedRegistryTokensMutationPayloadResolver resolves a FederatedRegistryTokensMutationPayload
type FederatedRegistryTokensMutationPayloadResolver struct {
	FederatedRegistryTokensMutationPayload
}

func handleCreateFederatedRegistryTokensMutationProblem(e error,
	clientMutationID *string,
) (*FederatedRegistryTokensMutationPayloadResolver, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}

	payload := FederatedRegistryTokensMutationPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}
	return &FederatedRegistryTokensMutationPayloadResolver{FederatedRegistryTokensMutationPayload: payload}, nil
}

func createFederatedRegistryTokensMutation(ctx context.Context,
	input *CreateFederatedRegistryTokensInput,
) (*FederatedRegistryTokensMutationPayloadResolver, error) {
	serviceCatalog := getServiceCatalog(ctx)

	jobID, err := serviceCatalog.FetchModelID(ctx, input.JobID)
	if err != nil {
		return nil, err
	}

	tokens, err := serviceCatalog.FederatedRegistryService.CreateFederatedRegistryTokensForJob(ctx, jobID)
	if err != nil {
		return nil, err
	}

	payload := FederatedRegistryTokensMutationPayload{
		ClientMutationID: input.ClientMutationID,
		Tokens:           tokens,
		Problems:         []Problem{},
	}

	return &FederatedRegistryTokensMutationPayloadResolver{FederatedRegistryTokensMutationPayload: payload}, nil
}

/* FederatedRegistry loader */

const federatedRegistryLoaderKey = "federatedRegistry"

// RegisterFederatedRegistryLoader registers a federated registry loader function
func RegisterFederatedRegistryLoader(collection *loader.Collection) {
	collection.Register(federatedRegistryLoaderKey, federatedRegistryBatchFunc)
}

func loadFederatedRegistry(ctx context.Context, id string) (*models.FederatedRegistry, error) {
	ldr, err := loader.Extract(ctx, federatedRegistryLoaderKey)
	if err != nil {
		return nil, err
	}

	data, err := ldr.Load(ctx, dataloader.StringKey(id))()
	if err != nil {
		return nil, err
	}

	registry, ok := data.(*models.FederatedRegistry)
	if !ok {
		return nil, errors.New("Wrong type")
	}

	return registry, nil
}

func federatedRegistryBatchFunc(ctx context.Context, ids []string) (loader.DataBatch, error) {
	federatedRegistries, err := getServiceCatalog(ctx).FederatedRegistryService.GetFederatedRegistriesByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	// Build map of results
	batch := loader.DataBatch{}
	for _, result := range federatedRegistries {
		batch[result.Metadata.ID] = result
	}

	return batch, nil
}
