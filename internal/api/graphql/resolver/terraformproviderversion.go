package resolver

import (
	"context"
	"strconv"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/loader"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/providerregistry"

	"github.com/aws/smithy-go/ptr"
	"github.com/graph-gophers/dataloader"
	graphql "github.com/graph-gophers/graphql-go"
)

/* TerraformProviderVersion Query Resolvers */

// TerraformProviderVersionsConnectionQueryArgs are used to query a providerVersion connection
type TerraformProviderVersionsConnectionQueryArgs struct {
	ConnectionQueryArgs
}

// TerraformProviderVersionQueryArgs are used to query a terraform provider version
type TerraformProviderVersionQueryArgs struct {
	Version           *string
	RegistryNamespace string
	ProviderName      string
}

// TerraformProviderVersionEdgeResolver resolves providerVersion edges
type TerraformProviderVersionEdgeResolver struct {
	edge Edge
}

// Cursor returns an opaque cursor
func (r *TerraformProviderVersionEdgeResolver) Cursor() (string, error) {
	providerVersion, ok := r.edge.Node.(models.TerraformProviderVersion)
	if !ok {
		return "", errors.NewError(errors.EInternal, "Failed to convert node type")
	}
	cursor, err := r.edge.CursorFunc(&providerVersion)
	return *cursor, err
}

// Node returns a providerVersion node
func (r *TerraformProviderVersionEdgeResolver) Node(ctx context.Context) (*TerraformProviderVersionResolver, error) {
	providerVersion, ok := r.edge.Node.(models.TerraformProviderVersion)
	if !ok {
		return nil, errors.NewError(errors.EInternal, "Failed to convert node type")
	}

	return &TerraformProviderVersionResolver{providerVersion: &providerVersion}, nil
}

// TerraformProviderVersionConnectionResolver resolves a providerVersion connection
type TerraformProviderVersionConnectionResolver struct {
	connection Connection
}

// NewTerraformProviderVersionConnectionResolver creates a new TerraformProviderVersionConnectionResolver
func NewTerraformProviderVersionConnectionResolver(ctx context.Context, input *providerregistry.GetProviderVersionsInput) (*TerraformProviderVersionConnectionResolver, error) {
	service := getProviderRegistryService(ctx)

	result, err := service.GetProviderVersions(ctx, input)
	if err != nil {
		return nil, err
	}

	providerVersions := result.ProviderVersions

	// Create edges
	edges := make([]Edge, len(providerVersions))
	for i, providerVersion := range providerVersions {
		edges[i] = Edge{CursorFunc: result.PageInfo.Cursor, Node: providerVersion}
	}

	pageInfo := PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
	}

	if len(providerVersions) > 0 {
		var err error
		pageInfo.StartCursor, err = result.PageInfo.Cursor(&providerVersions[0])
		if err != nil {
			return nil, err
		}

		pageInfo.EndCursor, err = result.PageInfo.Cursor(&providerVersions[len(edges)-1])
		if err != nil {
			return nil, err
		}
	}

	connection := Connection{
		TotalCount: result.PageInfo.TotalCount,
		PageInfo:   pageInfo,
		Edges:      edges,
	}

	return &TerraformProviderVersionConnectionResolver{connection: connection}, nil
}

// TotalCount returns the total result count for the connection
func (r *TerraformProviderVersionConnectionResolver) TotalCount() int32 {
	return r.connection.TotalCount
}

// PageInfo returns the connection page information
func (r *TerraformProviderVersionConnectionResolver) PageInfo() *PageInfoResolver {
	return &PageInfoResolver{pageInfo: r.connection.PageInfo}
}

// Edges returns the connection edges
func (r *TerraformProviderVersionConnectionResolver) Edges() *[]*TerraformProviderVersionEdgeResolver {
	resolvers := make([]*TerraformProviderVersionEdgeResolver, len(r.connection.Edges))
	for i, edge := range r.connection.Edges {
		resolvers[i] = &TerraformProviderVersionEdgeResolver{edge: edge}
	}
	return &resolvers
}

// TerraformProviderVersionResolver resolves a providerVersion resource
type TerraformProviderVersionResolver struct {
	providerVersion *models.TerraformProviderVersion
}

// ID resolver
func (r *TerraformProviderVersionResolver) ID() graphql.ID {
	return graphql.ID(gid.ToGlobalID(gid.TerraformProviderVersionType, r.providerVersion.Metadata.ID))
}

// GPGKeyID resolver
func (r *TerraformProviderVersionResolver) GPGKeyID() *string {
	return r.providerVersion.GetHexGPGKeyID()
}

// GPGASCIIArmor resolver
func (r *TerraformProviderVersionResolver) GPGASCIIArmor() *string {
	return r.providerVersion.GPGASCIIArmor
}

// Version resolver
func (r *TerraformProviderVersionResolver) Version() string {
	return r.providerVersion.SemanticVersion
}

// SHASumsUploaded resolver
func (r *TerraformProviderVersionResolver) SHASumsUploaded() bool {
	return r.providerVersion.SHASumsUploaded
}

// SHASumsSigUploaded resolver
func (r *TerraformProviderVersionResolver) SHASumsSigUploaded() bool {
	return r.providerVersion.SHASumsSignatureUploaded
}

// ReadmeUploaded resolver
func (r *TerraformProviderVersionResolver) ReadmeUploaded() bool {
	return r.providerVersion.ReadmeUploaded
}

// Readme resolver
func (r *TerraformProviderVersionResolver) Readme(ctx context.Context) (string, error) {
	if r.providerVersion.ReadmeUploaded {
		return getProviderRegistryService(ctx).GetProviderVersionReadme(ctx, r.providerVersion)
	}
	return "", nil
}

// Protocols resolver
func (r *TerraformProviderVersionResolver) Protocols() []string {
	return r.providerVersion.Protocols
}

// Latest resolver
func (r *TerraformProviderVersionResolver) Latest() bool {
	return r.providerVersion.Latest
}

// Metadata resolver
func (r *TerraformProviderVersionResolver) Metadata() *MetadataResolver {
	return &MetadataResolver{metadata: &r.providerVersion.Metadata}
}

// Provider resolver
func (r *TerraformProviderVersionResolver) Provider(ctx context.Context) (*TerraformProviderResolver, error) {
	provider, err := loadTerraformProvider(ctx, r.providerVersion.ProviderID)
	if err != nil {
		return nil, err
	}

	return &TerraformProviderResolver{provider: provider}, nil
}

// Platforms resolver
func (r *TerraformProviderVersionResolver) Platforms(ctx context.Context) ([]*TerraformProviderPlatformResolver, error) {
	platforms, err := getProviderRegistryService(ctx).GetProviderPlatforms(ctx, &providerregistry.GetProviderPlatformsInput{
		ProviderVersionID: &r.providerVersion.Metadata.ID,
	})
	if err != nil {
		return nil, err
	}

	resolvers := []*TerraformProviderPlatformResolver{}
	for _, platform := range platforms.ProviderPlatforms {
		platformCopy := platform
		resolvers = append(resolvers, &TerraformProviderPlatformResolver{providerPlatform: &platformCopy})
	}

	return resolvers, nil
}

// CreatedBy resolver
func (r *TerraformProviderVersionResolver) CreatedBy() string {
	return r.providerVersion.CreatedBy
}

func terraformProviderVersionQuery(ctx context.Context, args *TerraformProviderVersionQueryArgs) (*TerraformProviderVersionResolver, error) {
	service := getProviderRegistryService(ctx)

	provider, err := service.GetProviderByAddress(ctx, args.RegistryNamespace, args.ProviderName)
	if err != nil {
		if errors.ErrorCode(err) == errors.ENotFound {
			return nil, nil
		}
		return nil, err
	}

	input := providerregistry.GetProviderVersionsInput{
		PaginationOptions: &db.PaginationOptions{
			First: ptr.Int32(1),
		},
		ProviderID:      provider.Metadata.ID,
		SemanticVersion: args.Version,
	}

	// If version param is not defined then search for latest version
	if args.Version == nil {
		input.Latest = ptr.Bool(true)
	}

	versionsResponse, err := service.GetProviderVersions(ctx, &input)
	if err != nil {
		return nil, err
	}

	if len(versionsResponse.ProviderVersions) == 0 {
		return nil, nil
	}

	return &TerraformProviderVersionResolver{providerVersion: &versionsResponse.ProviderVersions[0]}, nil
}

/* TerraformProviderVersion Mutation Resolvers */

// TerraformProviderVersionMutationPayload is the response payload for a providerVersion mutation
type TerraformProviderVersionMutationPayload struct {
	ClientMutationID *string
	ProviderVersion  *models.TerraformProviderVersion
	Problems         []Problem
}

// TerraformProviderVersionMutationPayloadResolver resolves a TerraformProviderVersionMutationPayload
type TerraformProviderVersionMutationPayloadResolver struct {
	TerraformProviderVersionMutationPayload
}

// ProviderVersion field resolver
func (r *TerraformProviderVersionMutationPayloadResolver) ProviderVersion(ctx context.Context) *TerraformProviderVersionResolver {
	if r.TerraformProviderVersionMutationPayload.ProviderVersion == nil {
		return nil
	}
	return &TerraformProviderVersionResolver{providerVersion: r.TerraformProviderVersionMutationPayload.ProviderVersion}
}

// CreateTerraformProviderVersionInput contains the input for creating a new providerVersion
type CreateTerraformProviderVersionInput struct {
	ClientMutationID *string
	ProviderPath     string
	Version          string
	Protocols        []string
}

// DeleteTerraformProviderVersionInput contains the input for deleting a providerVersion
type DeleteTerraformProviderVersionInput struct {
	ClientMutationID *string
	Metadata         *MetadataInput
	ID               string
}

func handleTerraformProviderVersionMutationProblem(e error, clientMutationID *string) (*TerraformProviderVersionMutationPayloadResolver, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}
	payload := TerraformProviderVersionMutationPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}
	return &TerraformProviderVersionMutationPayloadResolver{TerraformProviderVersionMutationPayload: payload}, nil
}

func createTerraformProviderVersionMutation(ctx context.Context, input *CreateTerraformProviderVersionInput) (*TerraformProviderVersionMutationPayloadResolver, error) {
	service := getProviderRegistryService(ctx)

	provider, err := service.GetProviderByPath(ctx, input.ProviderPath)
	if err != nil {
		return nil, err
	}

	createdProviderVersion, err := service.CreateProviderVersion(ctx, &providerregistry.CreateProviderVersionInput{
		ProviderID:      provider.Metadata.ID,
		SemanticVersion: input.Version,
		Protocols:       input.Protocols,
	})
	if err != nil {
		return nil, err
	}

	payload := TerraformProviderVersionMutationPayload{ClientMutationID: input.ClientMutationID, ProviderVersion: createdProviderVersion, Problems: []Problem{}}
	return &TerraformProviderVersionMutationPayloadResolver{TerraformProviderVersionMutationPayload: payload}, nil
}

func deleteTerraformProviderVersionMutation(ctx context.Context, input *DeleteTerraformProviderVersionInput) (*TerraformProviderVersionMutationPayloadResolver, error) {
	service := getProviderRegistryService(ctx)

	providerVersion, err := service.GetProviderVersionByID(ctx, gid.FromGlobalID(input.ID))
	if err != nil {
		return nil, err
	}

	// Check if resource version is specified
	if input.Metadata != nil {
		v, err := strconv.Atoi(input.Metadata.Version)
		if err != nil {
			return nil, err
		}

		providerVersion.Metadata.Version = v
	}

	if err := service.DeleteProviderVersion(ctx, providerVersion); err != nil {
		return nil, err
	}

	payload := TerraformProviderVersionMutationPayload{ClientMutationID: input.ClientMutationID, ProviderVersion: providerVersion, Problems: []Problem{}}
	return &TerraformProviderVersionMutationPayloadResolver{TerraformProviderVersionMutationPayload: payload}, nil
}

/* TerraformProviderVersion loader */

const providerVersionLoaderKey = "providerVersion"

// RegisterTerraformProviderVersionLoader registers a providerVersion loader function
func RegisterTerraformProviderVersionLoader(collection *loader.Collection) {
	collection.Register(providerVersionLoaderKey, providerVersionBatchFunc)
}

func loadTerraformProviderVersion(ctx context.Context, id string) (*models.TerraformProviderVersion, error) {
	ldr, err := loader.Extract(ctx, providerVersionLoaderKey)
	if err != nil {
		return nil, err
	}

	data, err := ldr.Load(ctx, dataloader.StringKey(id))()
	if err != nil {
		return nil, err
	}

	providerVersion, ok := data.(models.TerraformProviderVersion)
	if !ok {
		return nil, errors.NewError(errors.EInternal, "Wrong type")
	}

	return &providerVersion, nil
}

func providerVersionBatchFunc(ctx context.Context, ids []string) (loader.DataBatch, error) {
	service := getProviderRegistryService(ctx)

	providerVersions, err := service.GetProviderVersionsByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	// Build map of results
	batch := loader.DataBatch{}
	for _, result := range providerVersions {
		batch[result.Metadata.ID] = result
	}

	return batch, nil
}
