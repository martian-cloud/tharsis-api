package resolver

import (
	"context"
	"strconv"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/loader"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/runner"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/serviceaccount"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"

	"github.com/graph-gophers/dataloader"
	graphql "github.com/graph-gophers/graphql-go"
)

/* Runner Query Resolvers */

// RunnersConnectionQueryArgs are used to query a runner connection
type RunnersConnectionQueryArgs struct {
	ConnectionQueryArgs
	IncludeInherited *bool
}

// RunnerEdgeResolver resolves runner edges
type RunnerEdgeResolver struct {
	edge Edge
}

// Cursor returns an opaque cursor
func (r *RunnerEdgeResolver) Cursor() (string, error) {
	runner, ok := r.edge.Node.(models.Runner)
	if !ok {
		return "", errors.New("Failed to convert node type")
	}
	cursor, err := r.edge.CursorFunc(&runner)
	return *cursor, err
}

// Node returns a runner node
func (r *RunnerEdgeResolver) Node() (*RunnerResolver, error) {
	runner, ok := r.edge.Node.(models.Runner)
	if !ok {
		return nil, errors.New("Failed to convert node type")
	}

	return &RunnerResolver{runner: &runner}, nil
}

// RunnerConnectionResolver resolves a runner connection
type RunnerConnectionResolver struct {
	connection Connection
}

// NewRunnerConnectionResolver creates a new RunnerConnectionResolver
func NewRunnerConnectionResolver(ctx context.Context, input *runner.GetRunnersInput) (*RunnerConnectionResolver, error) {
	service := getRunnerService(ctx)

	result, err := service.GetRunners(ctx, input)
	if err != nil {
		return nil, err
	}

	runners := result.Runners

	// Create edges
	edges := make([]Edge, len(runners))
	for i, runner := range runners {
		edges[i] = Edge{CursorFunc: result.PageInfo.Cursor, Node: runner}
	}

	pageInfo := PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
	}

	if len(runners) > 0 {
		var err error
		pageInfo.StartCursor, err = result.PageInfo.Cursor(&runners[0])
		if err != nil {
			return nil, err
		}

		pageInfo.EndCursor, err = result.PageInfo.Cursor(&runners[len(edges)-1])
		if err != nil {
			return nil, err
		}
	}

	connection := Connection{
		TotalCount: result.PageInfo.TotalCount,
		PageInfo:   pageInfo,
		Edges:      edges,
	}

	return &RunnerConnectionResolver{connection: connection}, nil
}

// TotalCount returns the total result count for the connection
func (r *RunnerConnectionResolver) TotalCount() int32 {
	return r.connection.TotalCount
}

// PageInfo returns the connection page information
func (r *RunnerConnectionResolver) PageInfo() *PageInfoResolver {
	return &PageInfoResolver{pageInfo: r.connection.PageInfo}
}

// Edges returns the connection edges
func (r *RunnerConnectionResolver) Edges() *[]*RunnerEdgeResolver {
	resolvers := make([]*RunnerEdgeResolver, len(r.connection.Edges))
	for i, edge := range r.connection.Edges {
		resolvers[i] = &RunnerEdgeResolver{edge: edge}
	}
	return &resolvers
}

// RunnerResolver resolves a runner resource
type RunnerResolver struct {
	runner *models.Runner
}

// ID resolver
func (r *RunnerResolver) ID() graphql.ID {
	return graphql.ID(gid.ToGlobalID(gid.RunnerType, r.runner.Metadata.ID))
}

// GroupPath resolver
func (r *RunnerResolver) GroupPath() string {
	return r.runner.GetGroupPath()
}

// ResourcePath resolver
func (r *RunnerResolver) ResourcePath() string {
	return r.runner.ResourcePath
}

// Name resolver
func (r *RunnerResolver) Name() string {
	return r.runner.Name
}

// Description resolver
func (r *RunnerResolver) Description() string {
	return r.runner.Description
}

// Type resolver
func (r *RunnerResolver) Type() string {
	return string(r.runner.Type)
}

// Metadata resolver
func (r *RunnerResolver) Metadata() *MetadataResolver {
	return &MetadataResolver{metadata: &r.runner.Metadata}
}

// Group resolver
func (r *RunnerResolver) Group(ctx context.Context) (*GroupResolver, error) {
	if r.runner.GroupID == nil {
		return nil, nil
	}

	group, err := loadGroup(ctx, *r.runner.GroupID)
	if err != nil {
		return nil, err
	}

	return &GroupResolver{group: group}, nil
}

// AssignedServiceAccounts resolver
func (r *RunnerResolver) AssignedServiceAccounts(ctx context.Context, args *ConnectionQueryArgs) (*ServiceAccountConnectionResolver, error) {
	if err := args.Validate(); err != nil {
		return nil, err
	}

	input := &serviceaccount.GetServiceAccountsInput{
		PaginationOptions: &pagination.Options{
			First:  args.First,
			Last:   args.Last,
			Before: args.Before,
			After:  args.After,
		},
		RunnerID:         &r.runner.Metadata.ID,
		NamespacePath:    r.runner.GetGroupPath(),
		IncludeInherited: true,
	}

	return NewServiceAccountConnectionResolver(ctx, input)
}

// CreatedBy resolver
func (r *RunnerResolver) CreatedBy() string {
	return r.runner.CreatedBy
}

/* Runner Mutation Resolvers */

// RunnerMutationPayload is the response payload for a runner mutation
type RunnerMutationPayload struct {
	ClientMutationID *string
	Runner           *models.Runner
	Problems         []Problem
}

// RunnerMutationPayloadResolver resolves a RunnerMutationPayload
type RunnerMutationPayloadResolver struct {
	RunnerMutationPayload
}

// Runner field resolver
func (r *RunnerMutationPayloadResolver) Runner() *RunnerResolver {
	if r.RunnerMutationPayload.Runner == nil {
		return nil
	}
	return &RunnerResolver{runner: r.RunnerMutationPayload.Runner}
}

// CreateRunnerInput contains the input for creating a new runner
type CreateRunnerInput struct {
	ClientMutationID *string
	Name             string
	Description      string
	GroupPath        string
}

// UpdateRunnerInput contains the input for updating a runner
type UpdateRunnerInput struct {
	ClientMutationID *string
	ID               string
	Metadata         *MetadataInput
	Description      string
}

// DeleteRunnerInput contains the input for deleting a runner
type DeleteRunnerInput struct {
	ClientMutationID *string
	Metadata         *MetadataInput
	ID               string
}

// AssignServiceAccountToRunnerInput is used to assign a service account to a runner
type AssignServiceAccountToRunnerInput struct {
	ClientMutationID   *string
	RunnerPath         string
	ServiceAccountPath string
}

func handleRunnerMutationProblem(e error, clientMutationID *string) (*RunnerMutationPayloadResolver, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}

	payload := RunnerMutationPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}
	return &RunnerMutationPayloadResolver{RunnerMutationPayload: payload}, nil
}

func createRunnerMutation(ctx context.Context, input *CreateRunnerInput) (*RunnerMutationPayloadResolver, error) {
	group, err := getGroupService(ctx).GetGroupByFullPath(ctx, input.GroupPath)
	if err != nil {
		return nil, err
	}

	service := getRunnerService(ctx)

	createdRunner, err := service.CreateRunner(ctx, &runner.CreateRunnerInput{
		Name:        input.Name,
		Description: input.Description,
		GroupID:     group.Metadata.ID,
	})
	if err != nil {
		return nil, err
	}

	payload := RunnerMutationPayload{ClientMutationID: input.ClientMutationID, Runner: createdRunner, Problems: []Problem{}}
	return &RunnerMutationPayloadResolver{RunnerMutationPayload: payload}, nil
}

func updateRunnerMutation(ctx context.Context, input *UpdateRunnerInput) (*RunnerMutationPayloadResolver, error) {
	service := getRunnerService(ctx)

	runner, err := service.GetRunnerByID(ctx, gid.FromGlobalID(input.ID))
	if err != nil {
		return nil, err
	}

	// Check if resource version is specified
	if input.Metadata != nil {
		v, cErr := strconv.Atoi(input.Metadata.Version)
		if cErr != nil {
			return nil, cErr
		}

		runner.Metadata.Version = v
	}

	// Update fields
	runner.Description = input.Description

	runner, err = service.UpdateRunner(ctx, runner)
	if err != nil {
		return nil, err
	}

	payload := RunnerMutationPayload{ClientMutationID: input.ClientMutationID, Runner: runner, Problems: []Problem{}}
	return &RunnerMutationPayloadResolver{RunnerMutationPayload: payload}, nil
}

func deleteRunnerMutation(ctx context.Context, input *DeleteRunnerInput) (*RunnerMutationPayloadResolver, error) {
	service := getRunnerService(ctx)

	runner, err := service.GetRunnerByID(ctx, gid.FromGlobalID(input.ID))
	if err != nil {
		return nil, err
	}

	// Check if resource version is specified
	if input.Metadata != nil {
		v, err := strconv.Atoi(input.Metadata.Version)
		if err != nil {
			return nil, err
		}

		runner.Metadata.Version = v
	}

	if err := service.DeleteRunner(ctx, runner); err != nil {
		return nil, err
	}

	payload := RunnerMutationPayload{ClientMutationID: input.ClientMutationID, Runner: runner, Problems: []Problem{}}
	return &RunnerMutationPayloadResolver{RunnerMutationPayload: payload}, nil
}

func assignServiceAccountToRunnerMutation(ctx context.Context, input *AssignServiceAccountToRunnerInput) (*RunnerMutationPayloadResolver, error) {
	saService := getSAService(ctx)
	runnerService := getRunnerService(ctx)

	runner, err := runnerService.GetRunnerByPath(ctx, input.RunnerPath)
	if err != nil {
		return nil, err
	}

	serviceAccount, err := saService.GetServiceAccountByPath(ctx, input.ServiceAccountPath)
	if err != nil {
		return nil, err
	}

	if err := runnerService.AssignServiceAccountToRunner(ctx, serviceAccount.Metadata.ID, runner.Metadata.ID); err != nil {
		return nil, err
	}

	payload := RunnerMutationPayload{ClientMutationID: input.ClientMutationID, Runner: runner, Problems: []Problem{}}
	return &RunnerMutationPayloadResolver{RunnerMutationPayload: payload}, nil
}

func unassignServiceAccountFromRunnerMutation(ctx context.Context, input *AssignServiceAccountToRunnerInput) (*RunnerMutationPayloadResolver, error) {
	saService := getSAService(ctx)
	runnerService := getRunnerService(ctx)

	runner, err := runnerService.GetRunnerByPath(ctx, input.RunnerPath)
	if err != nil {
		return nil, err
	}

	serviceAccount, err := saService.GetServiceAccountByPath(ctx, input.ServiceAccountPath)
	if err != nil {
		return nil, err
	}

	if err := runnerService.UnassignServiceAccountFromRunner(ctx, serviceAccount.Metadata.ID, runner.Metadata.ID); err != nil {
		return nil, err
	}

	payload := RunnerMutationPayload{ClientMutationID: input.ClientMutationID, Runner: runner, Problems: []Problem{}}
	return &RunnerMutationPayloadResolver{RunnerMutationPayload: payload}, nil
}

/* Runner loader */

const runnerLoaderKey = "runner"

// RegisterRunnerLoader registers a runner loader function
func RegisterRunnerLoader(collection *loader.Collection) {
	collection.Register(runnerLoaderKey, runnerBatchFunc)
}

func loadRunner(ctx context.Context, id string) (*models.Runner, error) {
	ldr, err := loader.Extract(ctx, runnerLoaderKey)
	if err != nil {
		return nil, err
	}

	data, err := ldr.Load(ctx, dataloader.StringKey(id))()
	if err != nil {
		return nil, err
	}

	runner, ok := data.(models.Runner)
	if !ok {
		return nil, errors.New("Wrong type")
	}

	return &runner, nil
}

func runnerBatchFunc(ctx context.Context, ids []string) (loader.DataBatch, error) {
	runnerService := getRunnerService(ctx)

	runners, err := runnerService.GetRunnersByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	// Build map of results
	batch := loader.DataBatch{}
	for _, result := range runners {
		batch[result.Metadata.ID] = result
	}

	return batch, nil
}
