package resolver

import (
	"context"
	"encoding/hex"
	"strconv"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/loader"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/moduleregistry"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"

	"github.com/aws/smithy-go/ptr"
	"github.com/graph-gophers/dataloader"
	graphql "github.com/graph-gophers/graphql-go"
)

/* TerraformModuleVersion Query Resolvers */

// TerraformModuleVersionsConnectionQueryArgs is used to query a moduleVersion connection
type TerraformModuleVersionsConnectionQueryArgs struct {
	ConnectionQueryArgs
}

// TerraformModuleVersionConfigurationDetailsQueryArgs is used to query configuration details
type TerraformModuleVersionConfigurationDetailsQueryArgs struct {
	Path string
}

// TerraformModuleVersionQueryArgs are used to query a terraform module version
type TerraformModuleVersionQueryArgs struct {
	Version           *string
	RegistryNamespace string
	ModuleName        string
	System            string
}

// TerraformModuleVersionEdgeResolver resolves moduleVersion edges
type TerraformModuleVersionEdgeResolver struct {
	edge Edge
}

// Cursor returns an opaque cursor
func (r *TerraformModuleVersionEdgeResolver) Cursor() (string, error) {
	moduleVersion, ok := r.edge.Node.(models.TerraformModuleVersion)
	if !ok {
		return "", errors.New(errors.EInternal, "Failed to convert node type")
	}
	cursor, err := r.edge.CursorFunc(&moduleVersion)
	return *cursor, err
}

// Node returns a moduleVersion node
func (r *TerraformModuleVersionEdgeResolver) Node() (*TerraformModuleVersionResolver, error) {
	moduleVersion, ok := r.edge.Node.(models.TerraformModuleVersion)
	if !ok {
		return nil, errors.New(errors.EInternal, "Failed to convert node type")
	}

	return &TerraformModuleVersionResolver{moduleVersion: &moduleVersion}, nil
}

// TerraformModuleVersionConnectionResolver resolves a moduleVersion connection
type TerraformModuleVersionConnectionResolver struct {
	connection Connection
}

// NewTerraformModuleVersionConnectionResolver creates a new TerraformModuleVersionConnectionResolver
func NewTerraformModuleVersionConnectionResolver(ctx context.Context, input *moduleregistry.GetModuleVersionsInput) (*TerraformModuleVersionConnectionResolver, error) {
	service := getModuleRegistryService(ctx)

	result, err := service.GetModuleVersions(ctx, input)
	if err != nil {
		return nil, err
	}

	moduleVersions := result.ModuleVersions

	// Create edges
	edges := make([]Edge, len(moduleVersions))
	for i, moduleVersion := range moduleVersions {
		edges[i] = Edge{CursorFunc: result.PageInfo.Cursor, Node: moduleVersion}
	}

	pageInfo := PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
	}

	if len(moduleVersions) > 0 {
		var err error
		pageInfo.StartCursor, err = result.PageInfo.Cursor(&moduleVersions[0])
		if err != nil {
			return nil, err
		}

		pageInfo.EndCursor, err = result.PageInfo.Cursor(&moduleVersions[len(edges)-1])
		if err != nil {
			return nil, err
		}
	}

	connection := Connection{
		TotalCount: result.PageInfo.TotalCount,
		PageInfo:   pageInfo,
		Edges:      edges,
	}

	return &TerraformModuleVersionConnectionResolver{connection: connection}, nil
}

// TotalCount returns the total result count for the connection
func (r *TerraformModuleVersionConnectionResolver) TotalCount() int32 {
	return r.connection.TotalCount
}

// PageInfo returns the connection page information
func (r *TerraformModuleVersionConnectionResolver) PageInfo() *PageInfoResolver {
	return &PageInfoResolver{pageInfo: r.connection.PageInfo}
}

// Edges returns the connection edges
func (r *TerraformModuleVersionConnectionResolver) Edges() *[]*TerraformModuleVersionEdgeResolver {
	resolvers := make([]*TerraformModuleVersionEdgeResolver, len(r.connection.Edges))
	for i, edge := range r.connection.Edges {
		resolvers[i] = &TerraformModuleVersionEdgeResolver{edge: edge}
	}
	return &resolvers
}

// TerraformModuleVersionResolver resolves a moduleVersion resource
type TerraformModuleVersionResolver struct {
	moduleVersion *models.TerraformModuleVersion
}

// ID resolver
func (r *TerraformModuleVersionResolver) ID() graphql.ID {
	return graphql.ID(gid.ToGlobalID(gid.TerraformModuleVersionType, r.moduleVersion.Metadata.ID))
}

// Version resolver
func (r *TerraformModuleVersionResolver) Version() string {
	return r.moduleVersion.SemanticVersion
}

// SHASum resolver
func (r *TerraformModuleVersionResolver) SHASum() string {
	return r.moduleVersion.GetSHASumHex()
}

// Status resolver
func (r *TerraformModuleVersionResolver) Status() string {
	return string(r.moduleVersion.Status)
}

// Error resolver
func (r *TerraformModuleVersionResolver) Error() string {
	return r.moduleVersion.Error
}

// Diagnostics resolver
func (r *TerraformModuleVersionResolver) Diagnostics() string {
	return r.moduleVersion.Diagnostics
}

// Submodules resolver
func (r *TerraformModuleVersionResolver) Submodules() []string {
	return r.moduleVersion.Submodules
}

// Examples resolver
func (r *TerraformModuleVersionResolver) Examples() []string {
	return r.moduleVersion.Examples
}

// Latest resolver
func (r *TerraformModuleVersionResolver) Latest() bool {
	return r.moduleVersion.Latest
}

// Metadata resolver
func (r *TerraformModuleVersionResolver) Metadata() *MetadataResolver {
	return &MetadataResolver{metadata: &r.moduleVersion.Metadata}
}

// Module resolver
func (r *TerraformModuleVersionResolver) Module(ctx context.Context) (*TerraformModuleResolver, error) {
	module, err := loadTerraformModule(ctx, r.moduleVersion.ModuleID)
	if err != nil {
		return nil, err
	}

	return &TerraformModuleResolver{module: module}, nil
}

// ConfigurationDetails resolver
func (r *TerraformModuleVersionResolver) ConfigurationDetails(ctx context.Context, args *TerraformModuleVersionConfigurationDetailsQueryArgs) (*moduleregistry.ModuleConfigurationDetails, error) {
	if r.moduleVersion.Status == models.TerraformModuleVersionStatusPending || r.moduleVersion.Status == models.TerraformModuleVersionStatusUploadInProgress {
		return nil, nil
	}

	metadata, err := getModuleRegistryService(ctx).GetModuleConfigurationDetails(ctx, r.moduleVersion, args.Path)
	if err != nil {
		if errors.ErrorCode(err) == errors.ENotFound {
			return nil, nil
		}
		return nil, err
	}

	return metadata, nil
}

// Attestations resolver
func (r *TerraformModuleVersionResolver) Attestations(ctx context.Context, args *TerraformModuleAttestationConnectionQueryArgs) (*TerraformModuleAttestationConnectionResolver, error) {
	digest := r.moduleVersion.GetSHASumHex()
	input := &moduleregistry.GetModuleAttestationsInput{
		PaginationOptions: &pagination.Options{
			First:  args.First,
			Last:   args.Last,
			Before: args.Before,
			After:  args.After,
		},
		ModuleID: r.moduleVersion.ModuleID,
		Digest:   &digest,
	}

	if args.Sort != nil {
		sort := db.TerraformModuleAttestationSortableField(*args.Sort)
		input.Sort = &sort
	}

	return NewTerraformModuleAttestationConnectionResolver(ctx, input)
}

// CreatedBy resolver
func (r *TerraformModuleVersionResolver) CreatedBy() string {
	return r.moduleVersion.CreatedBy
}

func terraformModuleVersionQuery(ctx context.Context, args *TerraformModuleVersionQueryArgs) (*TerraformModuleVersionResolver, error) {
	service := getModuleRegistryService(ctx)

	module, err := service.GetModuleByAddress(ctx, args.RegistryNamespace, args.ModuleName, args.System)
	if err != nil {
		if errors.ErrorCode(err) == errors.ENotFound {
			return nil, nil
		}
		return nil, err
	}

	input := moduleregistry.GetModuleVersionsInput{
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(1),
		},
		ModuleID:        module.Metadata.ID,
		SemanticVersion: args.Version,
	}

	// If version param is not defined then search for latest version
	if args.Version == nil {
		input.Latest = ptr.Bool(true)
	}

	versionsResponse, err := service.GetModuleVersions(ctx, &input)
	if err != nil {
		return nil, err
	}

	if len(versionsResponse.ModuleVersions) == 0 {
		return nil, nil
	}

	return &TerraformModuleVersionResolver{moduleVersion: &versionsResponse.ModuleVersions[0]}, nil
}

/* TerraformModuleVersion Mutation Resolvers */

// TerraformModuleVersionMutationPayload is the response payload for a moduleVersion mutation
type TerraformModuleVersionMutationPayload struct {
	ClientMutationID *string
	ModuleVersion    *models.TerraformModuleVersion
	Problems         []Problem
}

// TerraformModuleVersionMutationPayloadResolver resolves a TerraformModuleVersionMutationPayload
type TerraformModuleVersionMutationPayloadResolver struct {
	TerraformModuleVersionMutationPayload
}

// ModuleVersion field resolver
func (r *TerraformModuleVersionMutationPayloadResolver) ModuleVersion() *TerraformModuleVersionResolver {
	if r.TerraformModuleVersionMutationPayload.ModuleVersion == nil {
		return nil
	}
	return &TerraformModuleVersionResolver{moduleVersion: r.TerraformModuleVersionMutationPayload.ModuleVersion}
}

// CreateTerraformModuleVersionInput contains the input for creating a new moduleVersion
type CreateTerraformModuleVersionInput struct {
	ClientMutationID *string
	ModulePath       string
	Version          string
	SHASum           string
}

// DeleteTerraformModuleVersionInput contains the input for deleting a moduleVersion
type DeleteTerraformModuleVersionInput struct {
	ClientMutationID *string
	Metadata         *MetadataInput
	ID               string
}

func handleTerraformModuleVersionMutationProblem(e error, clientMutationID *string) (*TerraformModuleVersionMutationPayloadResolver, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}
	payload := TerraformModuleVersionMutationPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}
	return &TerraformModuleVersionMutationPayloadResolver{TerraformModuleVersionMutationPayload: payload}, nil
}

func createTerraformModuleVersionMutation(ctx context.Context, input *CreateTerraformModuleVersionInput) (*TerraformModuleVersionMutationPayloadResolver, error) {
	service := getModuleRegistryService(ctx)

	module, err := service.GetModuleByPath(ctx, input.ModulePath)
	if err != nil {
		return nil, err
	}

	shaSum, err := hex.DecodeString(input.SHASum)
	if err != nil {
		return nil, err
	}

	createdModuleVersion, err := service.CreateModuleVersion(ctx, &moduleregistry.CreateModuleVersionInput{
		ModuleID:        module.Metadata.ID,
		SemanticVersion: input.Version,
		SHASum:          shaSum,
	})
	if err != nil {
		return nil, err
	}

	payload := TerraformModuleVersionMutationPayload{ClientMutationID: input.ClientMutationID, ModuleVersion: createdModuleVersion, Problems: []Problem{}}
	return &TerraformModuleVersionMutationPayloadResolver{TerraformModuleVersionMutationPayload: payload}, nil
}

func deleteTerraformModuleVersionMutation(ctx context.Context, input *DeleteTerraformModuleVersionInput) (*TerraformModuleVersionMutationPayloadResolver, error) {
	service := getModuleRegistryService(ctx)

	moduleVersion, err := service.GetModuleVersionByID(ctx, gid.FromGlobalID(input.ID))
	if err != nil {
		return nil, err
	}

	// Check if resource version is specified
	if input.Metadata != nil {
		v, err := strconv.Atoi(input.Metadata.Version)
		if err != nil {
			return nil, err
		}

		moduleVersion.Metadata.Version = v
	}

	if err := service.DeleteModuleVersion(ctx, moduleVersion); err != nil {
		return nil, err
	}

	payload := TerraformModuleVersionMutationPayload{ClientMutationID: input.ClientMutationID, ModuleVersion: moduleVersion, Problems: []Problem{}}
	return &TerraformModuleVersionMutationPayloadResolver{TerraformModuleVersionMutationPayload: payload}, nil
}

/* TerraformModuleVersion loader */

const moduleVersionLoaderKey = "moduleVersion"

// RegisterTerraformModuleVersionLoader registers a moduleVersion loader function
func RegisterTerraformModuleVersionLoader(collection *loader.Collection) {
	collection.Register(moduleVersionLoaderKey, moduleVersionBatchFunc)
}

func loadTerraformModuleVersion(ctx context.Context, id string) (*models.TerraformModuleVersion, error) {
	ldr, err := loader.Extract(ctx, moduleVersionLoaderKey)
	if err != nil {
		return nil, err
	}

	data, err := ldr.Load(ctx, dataloader.StringKey(id))()
	if err != nil {
		return nil, err
	}

	moduleVersion, ok := data.(models.TerraformModuleVersion)
	if !ok {
		return nil, errors.New(errors.EInternal, "Wrong type")
	}

	return &moduleVersion, nil
}

func moduleVersionBatchFunc(ctx context.Context, ids []string) (loader.DataBatch, error) {
	service := getModuleRegistryService(ctx)

	moduleVersions, err := service.GetModuleVersionsByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	// Build map of results
	batch := loader.DataBatch{}
	for _, result := range moduleVersions {
		batch[result.Metadata.ID] = result
	}

	return batch, nil
}
