package resolver

import (
	"context"
	"encoding/base64"
	"io"

	"github.com/graph-gophers/dataloader"
	graphql "github.com/graph-gophers/graphql-go"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/loader"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/workspace"
)

/* State Version Query Resolvers */

// StateVersionConnectionQueryArgs are used to query a state version connection
type StateVersionConnectionQueryArgs struct {
	ConnectionQueryArgs
}

// StateVersionEdgeResolver resolves state version edges
type StateVersionEdgeResolver struct {
	edge Edge
}

// Cursor returns an opaque cursor
func (r *StateVersionEdgeResolver) Cursor() (string, error) {
	stateVersion, ok := r.edge.Node.(models.StateVersion)
	if !ok {
		return "", errors.NewError(errors.EInternal, "Failed to convert node type")
	}
	cursor, err := r.edge.CursorFunc(&stateVersion)
	return *cursor, err
}

// Node returns a state version node
func (r *StateVersionEdgeResolver) Node() (*StateVersionResolver, error) {
	stateVersion, ok := r.edge.Node.(models.StateVersion)
	if !ok {
		return nil, errors.NewError(errors.EInternal, "Failed to convert node type")
	}

	return &StateVersionResolver{stateVersion: &stateVersion}, nil
}

// StateVersionConnectionResolver resolves a stateVersion connection
type StateVersionConnectionResolver struct {
	connection Connection
}

// NewStateVersionConnectionResolver creates a new StateVersionConnectionResolver
func NewStateVersionConnectionResolver(ctx context.Context, input *workspace.GetStateVersionsInput) (*StateVersionConnectionResolver, error) {
	service := getWorkspaceService(ctx)

	result, err := service.GetStateVersions(ctx, input)
	if err != nil {
		return nil, err
	}

	stateVersions := result.StateVersions

	// Create edges
	edges := make([]Edge, len(stateVersions))
	for i, stateVersion := range stateVersions {
		edges[i] = Edge{CursorFunc: result.PageInfo.Cursor, Node: stateVersion}
	}

	pageInfo := PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
	}

	if len(stateVersions) > 0 {
		var err error
		pageInfo.StartCursor, err = result.PageInfo.Cursor(&stateVersions[0])
		if err != nil {
			return nil, err
		}

		pageInfo.EndCursor, err = result.PageInfo.Cursor(&stateVersions[len(edges)-1])
		if err != nil {
			return nil, err
		}
	}

	connection := Connection{
		TotalCount: result.PageInfo.TotalCount,
		PageInfo:   pageInfo,
		Edges:      edges,
	}

	return &StateVersionConnectionResolver{connection: connection}, nil
}

// TotalCount returns the total result count for the connection
func (r *StateVersionConnectionResolver) TotalCount() int32 {
	return r.connection.TotalCount
}

// PageInfo returns the connection page information
func (r *StateVersionConnectionResolver) PageInfo() *PageInfoResolver {
	return &PageInfoResolver{pageInfo: r.connection.PageInfo}
}

// Edges returns the connection edges
func (r *StateVersionConnectionResolver) Edges() *[]*StateVersionEdgeResolver {
	resolvers := make([]*StateVersionEdgeResolver, len(r.connection.Edges))
	for i, edge := range r.connection.Edges {
		resolvers[i] = &StateVersionEdgeResolver{edge: edge}
	}
	return &resolvers
}

// StateVersionDependencyResolver resolves a state version dependency
type StateVersionDependencyResolver struct {
	dependency *workspace.StateVersionDependency
}

// WorkspacePath resolver
func (r *StateVersionDependencyResolver) WorkspacePath() string {
	return r.dependency.WorkspacePath
}

// StateVersion resolver
func (r *StateVersionDependencyResolver) StateVersion(ctx context.Context) (*StateVersionResolver, error) {
	sv, err := loadStateVersion(ctx, r.dependency.StateVersionID)
	if errors.ErrorCode(err) == errors.ENotFound {
		// Return nil if state version is not found since it may have been deleted
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &StateVersionResolver{stateVersion: sv}, nil
}

// Workspace resolver
func (r *StateVersionDependencyResolver) Workspace(ctx context.Context) (*WorkspaceResolver, error) {
	ws, err := loadWorkspace(ctx, r.dependency.WorkspaceID)
	// Return nil if workspace is not found since it may have been deleted
	if errors.ErrorCode(err) == errors.ENotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &WorkspaceResolver{workspace: ws}, nil
}

// StateVersionResolver resolves a state version resource
type StateVersionResolver struct {
	stateVersion *models.StateVersion
}

// ID resolver
func (r *StateVersionResolver) ID() graphql.ID {
	return graphql.ID(gid.ToGlobalID(gid.StateVersionType, r.stateVersion.Metadata.ID))
}

// Run resolver
func (r *StateVersionResolver) Run(ctx context.Context) (*RunResolver, error) {
	if r.stateVersion.RunID == nil {
		return nil, nil
	}

	run, err := loadRun(ctx, *r.stateVersion.RunID)
	if err != nil {
		return nil, err
	}

	return &RunResolver{run: run}, nil
}

// Metadata resolver
func (r *StateVersionResolver) Metadata() *MetadataResolver {
	return &MetadataResolver{metadata: &r.stateVersion.Metadata}
}

// Outputs resolver (does not need connection resolver, because it's not doing pagination)
func (r *StateVersionResolver) Outputs(ctx context.Context) ([]*StateVersionOutputResolver, error) {
	return getStateVersionOutputs(ctx, r.stateVersion.Metadata.ID)
}

// Resources resolver
func (r *StateVersionResolver) Resources(ctx context.Context) ([]*workspace.StateVersionResource, error) {
	service := getWorkspaceService(ctx)

	resources, err := service.GetStateVersionResources(ctx, r.stateVersion)
	if err != nil {
		return nil, err
	}

	response := []*workspace.StateVersionResource{}

	for _, resource := range resources {
		resourceCopy := resource
		response = append(response, &resourceCopy)
	}

	return response, nil
}

// Dependencies resolver
func (r *StateVersionResolver) Dependencies(ctx context.Context) ([]*StateVersionDependencyResolver, error) {
	service := getWorkspaceService(ctx)

	dependencies, err := service.GetStateVersionDependencies(ctx, r.stateVersion)
	if err != nil {
		return nil, err
	}

	resolvers := []*StateVersionDependencyResolver{}

	for _, dependency := range dependencies {
		dependencyCopy := dependency
		resolvers = append(resolvers, &StateVersionDependencyResolver{
			dependency: &dependencyCopy,
		})
	}

	return resolvers, nil
}

// Data resolver
func (r *StateVersionResolver) Data(ctx context.Context) (string, error) {
	service := getWorkspaceService(ctx)

	reader, err := service.GetStateVersionContent(ctx, r.stateVersion.Metadata.ID)
	if err != nil {
		return "", err
	}

	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}

	encoded := base64.StdEncoding.EncodeToString(data)

	return encoded, err
}

// CreatedBy resolver.
func (r *StateVersionResolver) CreatedBy() string {
	return r.stateVersion.CreatedBy
}

/* State Version Mutation Resolvers */

// StateVersionMutationPayload is the response payload for state version mutation
type StateVersionMutationPayload struct {
	ClientMutationID *string
	StateVersion     *models.StateVersion
	Problems         []Problem
}

// StateVersionMutationPayloadResolver resolves StateVersionMutationPayload
type StateVersionMutationPayloadResolver struct {
	StateVersionMutationPayload
}

// StateVersion field resolver
func (r *StateVersionMutationPayloadResolver) StateVersion() *StateVersionResolver {
	if r.StateVersionMutationPayload.StateVersion == nil {
		return nil
	}
	return &StateVersionResolver{stateVersion: r.StateVersionMutationPayload.StateVersion}
}

// CreateStateVersionInput contains the input for creating a state version
type CreateStateVersionInput struct {
	ClientMutationID *string
	Metadata         *MetadataInput
	State            string
	RunID            string
}

func handleStateVersionMutationProblem(e error, clientMutationID *string) (*StateVersionMutationPayloadResolver, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}
	payload := StateVersionMutationPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}
	return &StateVersionMutationPayloadResolver{StateVersionMutationPayload: payload}, nil
}

func createStateVersionMutation(ctx context.Context, input *CreateStateVersionInput) (*StateVersionMutationPayloadResolver, error) {
	run, err := getRunService(ctx).GetRun(ctx, gid.FromGlobalID(input.RunID))
	if err != nil {
		return nil, err
	}

	stateVersionCreateOptions := models.StateVersion{
		WorkspaceID: run.WorkspaceID,
		RunID:       &run.Metadata.ID,
	}

	workspaceService := getWorkspaceService(ctx)

	stateVersion, err := workspaceService.CreateStateVersion(ctx, &stateVersionCreateOptions, &input.State)
	if err != nil {
		return nil, err
	}

	payload := StateVersionMutationPayload{ClientMutationID: input.ClientMutationID, StateVersion: stateVersion, Problems: []Problem{}}
	return &StateVersionMutationPayloadResolver{StateVersionMutationPayload: payload}, nil
}

/* StateVersion loader */

const stateVersionLoaderKey = "stateVersion"

// RegisterStateVersionLoader registers a state version loader function
func RegisterStateVersionLoader(collection *loader.Collection) {
	collection.Register(stateVersionLoaderKey, stateVersionBatchFunc)
}

func loadStateVersion(ctx context.Context, id string) (*models.StateVersion, error) {
	ldr, err := loader.Extract(ctx, stateVersionLoaderKey)
	if err != nil {
		return nil, err
	}

	data, err := ldr.Load(ctx, dataloader.StringKey(id))()
	if err != nil {
		return nil, err
	}

	sv, ok := data.(models.StateVersion)
	if !ok {
		return nil, errors.NewError(errors.EInternal, "Wrong type")
	}

	return &sv, nil
}

func stateVersionBatchFunc(ctx context.Context, ids []string) (loader.DataBatch, error) {
	wsService := getWorkspaceService(ctx)

	stateVersions, err := wsService.GetStateVersionsByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	// Build map of results
	batch := loader.DataBatch{}
	for _, result := range stateVersions {
		batch[result.Metadata.ID] = result
	}

	return batch, nil
}
