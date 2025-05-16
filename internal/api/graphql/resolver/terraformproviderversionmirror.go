package resolver

import (
	"context"
	"strconv"

	"github.com/graph-gophers/dataloader"
	graphql "github.com/graph-gophers/graphql-go"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/loader"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/providermirror"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

/* TerraformProviderVersionMirror Query Resolvers */

// TerraformProviderVersionMirrorConnectionQueryArgs are used to query for a provider version mirror connection.
type TerraformProviderVersionMirrorConnectionQueryArgs struct {
	ConnectionQueryArgs
	IncludeInherited *bool
}

// TerraformProviderVersionMirrorQueryArgs is used to query for a single provider version mirror.
// Deprecated: use node query instead with a TRN
type TerraformProviderVersionMirrorQueryArgs struct {
	RegistryHostname  string
	RegistryNamespace string
	Type              string
	Version           string
	GroupPath         string
}

// TerraformProviderVersionMirrorEdgeResolver resolves providerVersionMirror edges.
type TerraformProviderVersionMirrorEdgeResolver struct {
	edge Edge
}

// Cursor returns an opaque cursor
func (r *TerraformProviderVersionMirrorEdgeResolver) Cursor() (string, error) {
	versionMirror, ok := r.edge.Node.(models.TerraformProviderVersionMirror)
	if !ok {
		return "", errors.New("Failed to convert node type")
	}
	cursor, err := r.edge.CursorFunc(&versionMirror)
	return *cursor, err
}

// Node returns a providerVersion node
func (r *TerraformProviderVersionMirrorEdgeResolver) Node() (*TerraformProviderVersionMirrorResolver, error) {
	versionMirror, ok := r.edge.Node.(models.TerraformProviderVersionMirror)
	if !ok {
		return nil, errors.New("Failed to convert node type")
	}

	return &TerraformProviderVersionMirrorResolver{versionMirror: &versionMirror}, nil
}

// TerraformProviderVersionMirrorConnectionResolver resolves a providerVersionMirror connection
type TerraformProviderVersionMirrorConnectionResolver struct {
	connection Connection
}

// NewTerraformProviderVersionMirrorConnectionResolver creates a new TerraformProviderVersionMirrorConnectionResolver
func NewTerraformProviderVersionMirrorConnectionResolver(ctx context.Context, input *providermirror.GetProviderVersionMirrorsInput) (*TerraformProviderVersionMirrorConnectionResolver, error) {
	service := getServiceCatalog(ctx).TerraformProviderMirrorService

	result, err := service.GetProviderVersionMirrors(ctx, input)
	if err != nil {
		return nil, err
	}

	versionMirrors := result.VersionMirrors

	// Create edges
	edges := make([]Edge, len(versionMirrors))
	for i, providerVersion := range versionMirrors {
		edges[i] = Edge{CursorFunc: result.PageInfo.Cursor, Node: providerVersion}
	}

	pageInfo := PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
	}

	if len(versionMirrors) > 0 {
		var err error
		pageInfo.StartCursor, err = result.PageInfo.Cursor(&versionMirrors[0])
		if err != nil {
			return nil, err
		}

		pageInfo.EndCursor, err = result.PageInfo.Cursor(&versionMirrors[len(edges)-1])
		if err != nil {
			return nil, err
		}
	}

	connection := Connection{
		TotalCount: result.PageInfo.TotalCount,
		PageInfo:   pageInfo,
		Edges:      edges,
	}

	return &TerraformProviderVersionMirrorConnectionResolver{connection: connection}, nil
}

// TotalCount returns the total result count for the connection
func (r *TerraformProviderVersionMirrorConnectionResolver) TotalCount() int32 {
	return r.connection.TotalCount
}

// PageInfo returns the connection page information
func (r *TerraformProviderVersionMirrorConnectionResolver) PageInfo() *PageInfoResolver {
	return &PageInfoResolver{pageInfo: r.connection.PageInfo}
}

// Edges returns the connection edges
func (r *TerraformProviderVersionMirrorConnectionResolver) Edges() *[]*TerraformProviderVersionMirrorEdgeResolver {
	resolvers := make([]*TerraformProviderVersionMirrorEdgeResolver, len(r.connection.Edges))
	for i, edge := range r.connection.Edges {
		resolvers[i] = &TerraformProviderVersionMirrorEdgeResolver{edge: edge}
	}
	return &resolvers
}

// TerraformProviderVersionMirrorResolver resolves a providerVersionMirror resource
type TerraformProviderVersionMirrorResolver struct {
	versionMirror *models.TerraformProviderVersionMirror
}

// ID resolver
func (r *TerraformProviderVersionMirrorResolver) ID() graphql.ID {
	return graphql.ID(r.versionMirror.GetGlobalID())
}

// Metadata resolver
func (r *TerraformProviderVersionMirrorResolver) Metadata() *MetadataResolver {
	return &MetadataResolver{metadata: &r.versionMirror.Metadata}
}

// CreatedBy resolver
func (r *TerraformProviderVersionMirrorResolver) CreatedBy() string {
	return r.versionMirror.CreatedBy
}

// Version resolver
func (r *TerraformProviderVersionMirrorResolver) Version() string {
	return r.versionMirror.SemanticVersion
}

// Type resolver
func (r *TerraformProviderVersionMirrorResolver) Type() string {
	return r.versionMirror.Type
}

// RegistryNamespace resolver
func (r *TerraformProviderVersionMirrorResolver) RegistryNamespace() string {
	return r.versionMirror.RegistryNamespace
}

// RegistryHostname resolver
func (r *TerraformProviderVersionMirrorResolver) RegistryHostname() string {
	return r.versionMirror.RegistryHostname
}

// Group resolver
func (r *TerraformProviderVersionMirrorResolver) Group(ctx context.Context) (*GroupResolver, error) {
	group, err := loadGroup(ctx, r.versionMirror.GroupID)
	if err != nil {
		return nil, err
	}

	return &GroupResolver{group: group}, nil
}

// PlatformMirrors resolver
func (r *TerraformProviderVersionMirrorResolver) PlatformMirrors(ctx context.Context) ([]*TerraformProviderPlatformMirrorResolver, error) {
	result, err := getServiceCatalog(ctx).TerraformProviderMirrorService.GetProviderPlatformMirrors(ctx, &providermirror.GetProviderPlatformMirrorsInput{
		VersionMirrorID: r.versionMirror.Metadata.ID,
	})
	if err != nil {
		return nil, err
	}

	resolvers := []*TerraformProviderPlatformMirrorResolver{}
	for _, platform := range result.PlatformMirrors {
		platformCopy := platform
		resolvers = append(resolvers, &TerraformProviderPlatformMirrorResolver{platformMirror: &platformCopy})
	}

	return resolvers, nil
}

// Deprecated: use node query instead
func terraformProviderVersionMirrorQuery(ctx context.Context, args *TerraformProviderVersionMirrorQueryArgs) (*TerraformProviderVersionMirrorResolver, error) {
	trn := types.TerraformProviderVersionMirrorModelType.BuildTRN(args.GroupPath, args.RegistryHostname, args.RegistryNamespace, args.Type, args.Version)
	versionMirror, err := getServiceCatalog(ctx).TerraformProviderMirrorService.GetProviderVersionMirrorByTRN(ctx, trn)
	if err != nil {
		if errors.ErrorCode(err) == errors.ENotFound {
			return nil, nil
		}
		return nil, err
	}

	return &TerraformProviderVersionMirrorResolver{versionMirror: versionMirror}, nil
}

/* TerraformProviderVersionMirror Mutation Resolvers */

// TerraformProviderVersionMirrorMutationPayload is the response payload for a providerVersionMirror mutation
type TerraformProviderVersionMirrorMutationPayload struct {
	ClientMutationID *string
	VersionMirror    *models.TerraformProviderVersionMirror
	Problems         []Problem
}

// TerraformProviderVersionMirrorMutationPayloadResolver resolves a TerraformProviderVersionMirrorMutationPayload
type TerraformProviderVersionMirrorMutationPayloadResolver struct {
	TerraformProviderVersionMirrorMutationPayload
}

// VersionMirror field resolver
func (r *TerraformProviderVersionMirrorMutationPayloadResolver) VersionMirror() *TerraformProviderVersionMirrorResolver {
	if r.TerraformProviderVersionMirrorMutationPayload.VersionMirror == nil {
		return nil
	}

	return &TerraformProviderVersionMirrorResolver{versionMirror: r.TerraformProviderVersionMirrorMutationPayload.VersionMirror}
}

// CreateTerraformProviderVersionMirrorInput is the input for creating a TerraformProviderVersionMirror.
type CreateTerraformProviderVersionMirrorInput struct {
	ClientMutationID  *string
	GroupPath         *string // Deprecated: use GroupID instead with a TRN
	GroupID           *string
	Type              string
	RegistryNamespace string
	RegistryHostname  string
	SemanticVersion   string
}

// DeleteTerraformProviderVersionMirrorInput is the input for deleting a TerraformProviderVersionMirror.
type DeleteTerraformProviderVersionMirrorInput struct {
	ClientMutationID *string
	Metadata         *MetadataInput
	Force            *bool
	ID               string
}

func handleTerraformProviderVersionMirrorMutationProblem(e error, clientMutationID *string) (*TerraformProviderVersionMirrorMutationPayloadResolver, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}
	payload := TerraformProviderVersionMirrorMutationPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}
	return &TerraformProviderVersionMirrorMutationPayloadResolver{TerraformProviderVersionMirrorMutationPayload: payload}, nil
}

func createTerraformProviderVersionMirrorMutation(ctx context.Context, input *CreateTerraformProviderVersionMirrorInput) (*TerraformProviderVersionMirrorMutationPayloadResolver, error) {
	groupID, err := toModelID(ctx, input.GroupPath, input.GroupID, types.GroupModelType)
	if err != nil {
		return nil, err
	}

	createdMirror, err := getServiceCatalog(ctx).TerraformProviderMirrorService.CreateProviderVersionMirror(ctx, &providermirror.CreateProviderVersionMirrorInput{
		Type:              input.Type,
		RegistryNamespace: input.RegistryNamespace,
		RegistryHostname:  input.RegistryHostname,
		GroupID:           groupID,
		SemanticVersion:   input.SemanticVersion,
	})
	if err != nil {
		return nil, err
	}

	payload := TerraformProviderVersionMirrorMutationPayload{ClientMutationID: input.ClientMutationID, VersionMirror: createdMirror, Problems: []Problem{}}
	return &TerraformProviderVersionMirrorMutationPayloadResolver{TerraformProviderVersionMirrorMutationPayload: payload}, nil
}

func deleteTerraformProviderVersionMirrorMutation(ctx context.Context, input *DeleteTerraformProviderVersionMirrorInput) (*TerraformProviderVersionMirrorMutationPayloadResolver, error) {
	serviceCatalog := getServiceCatalog(ctx)

	versionMirrorID, err := serviceCatalog.FetchModelID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	gotMirror, err := serviceCatalog.TerraformProviderMirrorService.GetProviderVersionMirrorByID(ctx, versionMirrorID)
	if err != nil {
		return nil, err
	}

	// Check if resource version is specified
	if input.Metadata != nil {
		v, err := strconv.Atoi(input.Metadata.Version)
		if err != nil {
			return nil, err
		}

		gotMirror.Metadata.Version = v
	}

	toDelete := &providermirror.DeleteProviderVersionMirrorInput{
		VersionMirror: gotMirror,
	}

	if input.Force != nil {
		toDelete.Force = *input.Force
	}

	if err := serviceCatalog.TerraformProviderMirrorService.DeleteProviderVersionMirror(ctx, toDelete); err != nil {
		return nil, err
	}

	payload := TerraformProviderVersionMirrorMutationPayload{ClientMutationID: input.ClientMutationID, VersionMirror: gotMirror, Problems: []Problem{}}
	return &TerraformProviderVersionMirrorMutationPayloadResolver{TerraformProviderVersionMirrorMutationPayload: payload}, nil
}

/* TerraformProviderVersionMirror loader */

const providerVersionMirrorLoaderKey = "providerVersionMirror"

// RegisterTerraformProviderVersionMirrorLoader registers a providerVersionMirror loader function
func RegisterTerraformProviderVersionMirrorLoader(collection *loader.Collection) {
	collection.Register(providerVersionMirrorLoaderKey, providerVersionMirrorBatchFunc)
}

func loadTerraformProviderVersionMirror(ctx context.Context, id string) (*models.TerraformProviderVersionMirror, error) {
	ldr, err := loader.Extract(ctx, providerVersionMirrorLoaderKey)
	if err != nil {
		return nil, err
	}

	data, err := ldr.Load(ctx, dataloader.StringKey(id))()
	if err != nil {
		return nil, err
	}

	versionMirror, ok := data.(models.TerraformProviderVersionMirror)
	if !ok {
		return nil, errors.New("Wrong type")
	}

	return &versionMirror, nil
}

func providerVersionMirrorBatchFunc(ctx context.Context, ids []string) (loader.DataBatch, error) {
	versionMirrors, err := getServiceCatalog(ctx).TerraformProviderMirrorService.GetProviderVersionMirrorsByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	// Build map of results
	batch := loader.DataBatch{}
	for _, result := range versionMirrors {
		batch[result.Metadata.ID] = result
	}

	return batch, nil
}
