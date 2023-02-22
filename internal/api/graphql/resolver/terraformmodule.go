package resolver

import (
	"context"
	"fmt"
	"strconv"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/loader"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/moduleregistry"

	"github.com/aws/smithy-go/ptr"
	"github.com/graph-gophers/dataloader"
	graphql "github.com/graph-gophers/graphql-go"
)

/* TerraformModule Query Resolvers */

// TerraformModuleConnectionQueryArgs are used to query a module connection
type TerraformModuleConnectionQueryArgs struct {
	ConnectionQueryArgs
	Search *string
}

// TerraformModuleQueryArgs are used to query a terraform module
type TerraformModuleQueryArgs struct {
	RegistryNamespace string
	ModuleName        string
	System            string
}

// TerraformModuleEdgeResolver resolves module edges
type TerraformModuleEdgeResolver struct {
	edge Edge
}

// Cursor returns an opaque cursor
func (r *TerraformModuleEdgeResolver) Cursor() (string, error) {
	module, ok := r.edge.Node.(models.TerraformModule)
	if !ok {
		return "", errors.NewError(errors.EInternal, "Failed to convert node type")
	}
	cursor, err := r.edge.CursorFunc(&module)
	return *cursor, err
}

// Node returns a module node
func (r *TerraformModuleEdgeResolver) Node() (*TerraformModuleResolver, error) {
	module, ok := r.edge.Node.(models.TerraformModule)
	if !ok {
		return nil, errors.NewError(errors.EInternal, "Failed to convert node type")
	}

	return &TerraformModuleResolver{module: &module}, nil
}

// TerraformModuleConnectionResolver resolves a module connection
type TerraformModuleConnectionResolver struct {
	connection Connection
}

// NewTerraformModuleConnectionResolver creates a new TerraformModuleConnectionResolver
func NewTerraformModuleConnectionResolver(ctx context.Context, input *moduleregistry.GetModulesInput) (*TerraformModuleConnectionResolver, error) {
	service := getModuleRegistryService(ctx)

	result, err := service.GetModules(ctx, input)
	if err != nil {
		return nil, err
	}

	modules := result.Modules

	// Create edges
	edges := make([]Edge, len(modules))
	for i, module := range modules {
		edges[i] = Edge{CursorFunc: result.PageInfo.Cursor, Node: module}
	}

	pageInfo := PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
	}

	if len(modules) > 0 {
		var err error
		pageInfo.StartCursor, err = result.PageInfo.Cursor(&modules[0])
		if err != nil {
			return nil, err
		}

		pageInfo.EndCursor, err = result.PageInfo.Cursor(&modules[len(edges)-1])
		if err != nil {
			return nil, err
		}
	}

	connection := Connection{
		TotalCount: result.PageInfo.TotalCount,
		PageInfo:   pageInfo,
		Edges:      edges,
	}

	return &TerraformModuleConnectionResolver{connection: connection}, nil
}

// TotalCount returns the total result count for the connection
func (r *TerraformModuleConnectionResolver) TotalCount() int32 {
	return r.connection.TotalCount
}

// PageInfo returns the connection page information
func (r *TerraformModuleConnectionResolver) PageInfo() *PageInfoResolver {
	return &PageInfoResolver{pageInfo: r.connection.PageInfo}
}

// Edges returns the connection edges
func (r *TerraformModuleConnectionResolver) Edges() *[]*TerraformModuleEdgeResolver {
	resolvers := make([]*TerraformModuleEdgeResolver, len(r.connection.Edges))
	for i, edge := range r.connection.Edges {
		resolvers[i] = &TerraformModuleEdgeResolver{edge: edge}
	}
	return &resolvers
}

// TerraformModuleResolver resolves a module resource
type TerraformModuleResolver struct {
	module *models.TerraformModule
}

// ID resolver
func (r *TerraformModuleResolver) ID() graphql.ID {
	return graphql.ID(gid.ToGlobalID(gid.TerraformModuleType, r.module.Metadata.ID))
}

// Name resolver
func (r *TerraformModuleResolver) Name() string {
	return r.module.Name
}

// System resolver
func (r *TerraformModuleResolver) System() string {
	return r.module.System
}

// Private resolver
func (r *TerraformModuleResolver) Private() bool {
	return r.module.Private
}

// CreatedBy resolver
func (r *TerraformModuleResolver) CreatedBy() string {
	return r.module.CreatedBy
}

// GroupPath resolver
func (r *TerraformModuleResolver) GroupPath() string {
	return r.module.GetGroupPath()
}

// ResourcePath resolver
func (r *TerraformModuleResolver) ResourcePath() string {
	return r.module.ResourcePath
}

// RepositoryURL resolver
func (r *TerraformModuleResolver) RepositoryURL() string {
	return r.module.RepositoryURL
}

// RegistryNamespace resolver
func (r *TerraformModuleResolver) RegistryNamespace() string {
	return r.module.GetRegistryNamespace()
}

// Metadata resolver
func (r *TerraformModuleResolver) Metadata() *MetadataResolver {
	return &MetadataResolver{metadata: &r.module.Metadata}
}

// Group resolver
func (r *TerraformModuleResolver) Group(ctx context.Context) (*GroupResolver, error) {
	group, err := loadGroup(ctx, r.module.GroupID)
	if err != nil {
		return nil, err
	}
	return &GroupResolver{group: group}, nil
}

// Source resolver
func (r *TerraformModuleResolver) Source(ctx context.Context) string {
	cfg := getConfig(ctx)
	return fmt.Sprintf("%s/%s/%s/%s", cfg.ServiceDiscoveryHost, r.module.GetRegistryNamespace(), r.module.Name, r.module.System)
}

// Versions resolver
func (r *TerraformModuleResolver) Versions(ctx context.Context, args *TerraformModuleVersionsConnectionQueryArgs) (*TerraformModuleVersionConnectionResolver, error) {
	input := &moduleregistry.GetModuleVersionsInput{
		PaginationOptions: &db.PaginationOptions{
			First:  args.First,
			Last:   args.Last,
			Before: args.Before,
			After:  args.After,
		},
		ModuleID: r.module.Metadata.ID,
	}

	if args.Sort != nil {
		sort := db.TerraformModuleVersionSortableField(*args.Sort)
		input.Sort = &sort
	}

	return NewTerraformModuleVersionConnectionResolver(ctx, input)
}

// Attestations resolver
func (r *TerraformModuleResolver) Attestations(ctx context.Context, args *TerraformModuleAttestationConnectionQueryArgs) (*TerraformModuleAttestationConnectionResolver, error) {
	input := &moduleregistry.GetModuleAttestationsInput{
		PaginationOptions: &db.PaginationOptions{
			First:  args.First,
			Last:   args.Last,
			Before: args.Before,
			After:  args.After,
		},
		ModuleID: r.module.Metadata.ID,
		Digest:   args.Digest,
	}

	if args.Sort != nil {
		sort := db.TerraformModuleAttestationSortableField(*args.Sort)
		input.Sort = &sort
	}

	return NewTerraformModuleAttestationConnectionResolver(ctx, input)
}

// LatestVersion resolver
func (r *TerraformModuleResolver) LatestVersion(ctx context.Context) (*TerraformModuleVersionResolver, error) {
	versionsResp, err := getModuleRegistryService(ctx).GetModuleVersions(ctx, &moduleregistry.GetModuleVersionsInput{
		PaginationOptions: &db.PaginationOptions{
			First: ptr.Int32(1),
		},
		ModuleID: r.module.Metadata.ID,
		Latest:   ptr.Bool(true),
	})
	if err != nil {
		return nil, err
	}

	if len(versionsResp.ModuleVersions) == 0 {
		return nil, nil
	}

	return &TerraformModuleVersionResolver{moduleVersion: &versionsResp.ModuleVersions[0]}, nil
}

func terraformModulesQuery(ctx context.Context, args *TerraformModuleConnectionQueryArgs) (*TerraformModuleConnectionResolver, error) {
	if err := args.Validate(); err != nil {
		return nil, err
	}

	input := moduleregistry.GetModulesInput{
		PaginationOptions: &db.PaginationOptions{First: args.First, Last: args.Last, After: args.After, Before: args.Before},
		Search:            args.Search,
	}

	if args.Sort != nil {
		sort := db.TerraformModuleSortableField(*args.Sort)
		input.Sort = &sort
	}

	return NewTerraformModuleConnectionResolver(ctx, &input)
}

func terraformModuleQuery(ctx context.Context, args *TerraformModuleQueryArgs) (*TerraformModuleResolver, error) {
	module, err := getModuleRegistryService(ctx).GetModuleByAddress(ctx, args.RegistryNamespace, args.ModuleName, args.System)
	if err != nil {
		if errors.ErrorCode(err) == errors.ENotFound {
			return nil, nil
		}
		return nil, err
	}
	return &TerraformModuleResolver{module: module}, nil
}

/* TerraformModule Mutation Resolvers */

// TerraformModuleMutationPayload is the response payload for module mutation
type TerraformModuleMutationPayload struct {
	ClientMutationID *string
	Module           *models.TerraformModule
	Problems         []Problem
}

// TerraformModuleMutationPayloadResolver resolves a TerraformModuleMutationPayload
type TerraformModuleMutationPayloadResolver struct {
	TerraformModuleMutationPayload
}

// Module field resolver
func (r *TerraformModuleMutationPayloadResolver) Module() *TerraformModuleResolver {
	if r.TerraformModuleMutationPayload.Module == nil {
		return nil
	}
	return &TerraformModuleResolver{module: r.TerraformModuleMutationPayload.Module}
}

// UpdateTerraformModuleInput contains the input for updating a module
type UpdateTerraformModuleInput struct {
	ClientMutationID *string
	Metadata         *MetadataInput
	Name             *string
	System           *string
	RepositoryURL    *string
	Private          *bool
	ID               string
}

// CreateTerraformModuleInput contains the input for creating a module
type CreateTerraformModuleInput struct {
	ClientMutationID *string
	Private          *bool
	RepositoryURL    *string
	Name             string
	System           string
	GroupPath        string
}

// DeleteTerraformModuleInput contains the input for deleting a module
type DeleteTerraformModuleInput struct {
	ClientMutationID *string
	ID               string
}

func handleTerraformModuleMutationProblem(e error, clientMutationID *string) (*TerraformModuleMutationPayloadResolver, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}
	payload := TerraformModuleMutationPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}
	return &TerraformModuleMutationPayloadResolver{TerraformModuleMutationPayload: payload}, nil
}

func createTerraformModuleMutation(ctx context.Context, input *CreateTerraformModuleInput) (*TerraformModuleMutationPayloadResolver, error) {
	service := getModuleRegistryService(ctx)

	group, err := getGroupService(ctx).GetGroupByFullPath(ctx, input.GroupPath)
	if err != nil {
		return nil, err
	}

	createOptions := moduleregistry.CreateModuleInput{
		GroupID: group.Metadata.ID,
		Name:    input.Name,
		System:  input.System,
		Private: true,
	}

	if input.Private != nil {
		createOptions.Private = *input.Private
	}

	if input.RepositoryURL != nil {
		createOptions.RepositoryURL = *input.RepositoryURL
	}

	module, err := service.CreateModule(ctx, &createOptions)
	if err != nil {
		return nil, err
	}

	payload := TerraformModuleMutationPayload{ClientMutationID: input.ClientMutationID, Module: module, Problems: []Problem{}}
	return &TerraformModuleMutationPayloadResolver{TerraformModuleMutationPayload: payload}, nil
}

func updateTerraformModuleMutation(ctx context.Context, input *UpdateTerraformModuleInput) (*TerraformModuleMutationPayloadResolver, error) {
	service := getModuleRegistryService(ctx)

	module, err := service.GetModuleByID(ctx, gid.FromGlobalID(input.ID))
	if err != nil {
		return nil, err
	}

	// Check if resource version is specified
	if input.Metadata != nil {
		v, cErr := strconv.Atoi(input.Metadata.Version)
		if cErr != nil {
			return nil, cErr
		}

		module.Metadata.Version = v
	}

	// Update fields
	if input.Name != nil {
		module.Name = *input.Name
	}

	if input.System != nil {
		module.System = *input.System
	}

	if input.Private != nil {
		module.Private = *input.Private
	}

	if input.RepositoryURL != nil {
		module.RepositoryURL = *input.RepositoryURL
	}

	module, err = service.UpdateModule(ctx, module)
	if err != nil {
		return nil, err
	}

	payload := TerraformModuleMutationPayload{ClientMutationID: input.ClientMutationID, Module: module, Problems: []Problem{}}
	return &TerraformModuleMutationPayloadResolver{TerraformModuleMutationPayload: payload}, nil
}

func deleteTerraformModuleMutation(ctx context.Context, input *DeleteTerraformModuleInput) (*TerraformModuleMutationPayloadResolver, error) {
	service := getModuleRegistryService(ctx)

	module, err := service.GetModuleByID(ctx, gid.FromGlobalID(input.ID))
	if err != nil {
		return nil, err
	}

	if err := service.DeleteModule(ctx, module); err != nil {
		return nil, err
	}

	payload := TerraformModuleMutationPayload{ClientMutationID: input.ClientMutationID, Module: module, Problems: []Problem{}}
	return &TerraformModuleMutationPayloadResolver{TerraformModuleMutationPayload: payload}, nil
}

/* TerraformModule loader */

const moduleLoaderKey = "terraformModule"

// RegisterTerraformModuleLoader registers a module loader function
func RegisterTerraformModuleLoader(collection *loader.Collection) {
	collection.Register(moduleLoaderKey, moduleBatchFunc)
}

func loadTerraformModule(ctx context.Context, id string) (*models.TerraformModule, error) {
	ldr, err := loader.Extract(ctx, moduleLoaderKey)
	if err != nil {
		return nil, err
	}

	data, err := ldr.Load(ctx, dataloader.StringKey(id))()
	if err != nil {
		return nil, err
	}

	ws, ok := data.(models.TerraformModule)
	if !ok {
		return nil, errors.NewError(errors.EInternal, "Wrong type")
	}

	return &ws, nil
}

func moduleBatchFunc(ctx context.Context, ids []string) (loader.DataBatch, error) {
	service := getModuleRegistryService(ctx)

	modules, err := service.GetModulesByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	// Build map of results
	batch := loader.DataBatch{}
	for _, result := range modules {
		batch[result.Metadata.ID] = result
	}

	return batch, nil
}
