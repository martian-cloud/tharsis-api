package resolver

import (
	"context"
	"fmt"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/loader"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run"

	"github.com/graph-gophers/dataloader"
	graphql "github.com/graph-gophers/graphql-go"
)

/* Run Query Resolvers */

// RunConnectionQueryArgs are used to query a run connection
type RunConnectionQueryArgs struct {
	ConnectionQueryArgs
	WorkspacePath *string
	WorkspaceID   *string
}

// RunQueryArgs are used to query a single run
type RunQueryArgs struct {
	ID string
}

// RunEdgeResolver resolves run edges
type RunEdgeResolver struct {
	edge Edge
}

// Cursor returns an opaque cursor
func (r *RunEdgeResolver) Cursor() (string, error) {
	run, ok := r.edge.Node.(models.Run)
	if !ok {
		return "", errors.NewError(errors.EInternal, "Failed to convert node type")
	}
	cursor, err := r.edge.CursorFunc(&run)
	return *cursor, err
}

// Node returns a run node
func (r *RunEdgeResolver) Node(ctx context.Context) (*RunResolver, error) {
	run, ok := r.edge.Node.(models.Run)
	if !ok {
		return nil, errors.NewError(errors.EInternal, "Failed to convert node type")
	}

	return &RunResolver{run: &run}, nil
}

// RunConnectionResolver resolves a run connection
type RunConnectionResolver struct {
	connection Connection
}

// NewRunConnectionResolver creates a new RunConnectionResolver
func NewRunConnectionResolver(ctx context.Context, input *run.GetRunsInput) (*RunConnectionResolver, error) {
	runService := getRunService(ctx)

	result, err := runService.GetRuns(ctx, input)
	if err != nil {
		return nil, err
	}

	runs := result.Runs

	// Create edges
	edges := make([]Edge, len(runs))
	for i, run := range runs {
		edges[i] = Edge{CursorFunc: result.PageInfo.Cursor, Node: run}
	}

	pageInfo := PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
	}

	if len(runs) > 0 {
		var err error
		pageInfo.StartCursor, err = result.PageInfo.Cursor(&runs[0])
		if err != nil {
			return nil, err
		}

		pageInfo.EndCursor, err = result.PageInfo.Cursor(&runs[len(edges)-1])
		if err != nil {
			return nil, err
		}
	}

	connection := Connection{
		TotalCount: result.PageInfo.TotalCount,
		PageInfo:   pageInfo,
		Edges:      edges,
	}

	return &RunConnectionResolver{connection: connection}, nil
}

// TotalCount returns the total result count for the connection
func (r *RunConnectionResolver) TotalCount() int32 {
	return r.connection.TotalCount
}

// PageInfo returns the connection page information
func (r *RunConnectionResolver) PageInfo() *PageInfoResolver {
	return &PageInfoResolver{pageInfo: r.connection.PageInfo}
}

// Edges returns the connection edges
func (r *RunConnectionResolver) Edges() *[]*RunEdgeResolver {
	resolvers := make([]*RunEdgeResolver, len(r.connection.Edges))
	for i, edge := range r.connection.Edges {
		resolvers[i] = &RunEdgeResolver{edge: edge}
	}
	return &resolvers
}

// RunResolver resolves a run resource
type RunResolver struct {
	run *models.Run
}

// ID resolver
func (r *RunResolver) ID() graphql.ID {
	return graphql.ID(gid.ToGlobalID(gid.RunType, r.run.Metadata.ID))
}

// Status resolver
func (r *RunResolver) Status() string {
	return string(r.run.Status)
}

// IsDestroy resolver
func (r *RunResolver) IsDestroy() bool {
	return r.run.IsDestroy
}

// Workspace resolver
func (r *RunResolver) Workspace(ctx context.Context) (*WorkspaceResolver, error) {
	workspace, err := loadWorkspace(ctx, r.run.WorkspaceID)
	if err != nil {
		return nil, err
	}

	return &WorkspaceResolver{workspace: workspace}, nil
}

// CreatedBy resolver
func (r *RunResolver) CreatedBy() string {
	return r.run.CreatedBy
}

// Metadata resolver
func (r *RunResolver) Metadata() *MetadataResolver {
	return &MetadataResolver{metadata: &r.run.Metadata}
}

// ConfigurationVersion resolver
func (r *RunResolver) ConfigurationVersion(ctx context.Context) (*ConfigurationVersionResolver, error) {
	if r.run.ConfigurationVersionID == nil {
		return nil, nil
	}

	cv, err := loadConfigurationVersion(ctx, *r.run.ConfigurationVersionID)
	if err != nil {
		return nil, err
	}

	return &ConfigurationVersionResolver{configurationVersion: cv}, nil
}

// Apply resolver
func (r *RunResolver) Apply(ctx context.Context) (*ApplyResolver, error) {
	if r.run.ApplyID == "" {
		return nil, nil
	}

	apply, err := loadApply(ctx, r.run.ApplyID)
	if err != nil {
		return nil, err
	}

	return &ApplyResolver{apply: apply}, nil
}

// Plan resolver
func (r *RunResolver) Plan(ctx context.Context) (*PlanResolver, error) {
	plan, err := loadPlan(ctx, r.run.PlanID)
	if err != nil {
		return nil, err
	}

	return &PlanResolver{plan: plan}, nil
}

// Variables resolver
func (r *RunResolver) Variables(ctx context.Context) ([]*RunVariableResolver, error) {
	resolvers := []*RunVariableResolver{}

	service := getRunService(ctx)

	variables, err := service.GetRunVariables(ctx, r.run.Metadata.ID)
	if err != nil {
		return nil, err
	}

	for _, v := range variables {
		varCopy := v
		resolvers = append(resolvers, &RunVariableResolver{variable: &varCopy})
	}

	return resolvers, nil
}

// ModuleSource resolver
func (r *RunResolver) ModuleSource(ctx context.Context) *string {
	return r.run.ModuleSource
}

// ModuleVersion resolver
func (r *RunResolver) ModuleVersion(ctx context.Context) *string {
	return r.run.ModuleVersion
}

// ForceCanceledBy resolver
func (r *RunResolver) ForceCanceledBy() *string {
	return r.run.ForceCanceledBy
}

// ForceCanceled resolver
func (r *RunResolver) ForceCanceled() bool {
	return r.run.ForceCanceled
}

// ForceCancelAvailableAt resolver
func (r *RunResolver) ForceCancelAvailableAt() *graphql.Time {
	if r.run.ForceCancelAvailableAt == nil {
		return nil
	}
	return &graphql.Time{Time: *r.run.ForceCancelAvailableAt}
}

// Comment resolver
func (r *RunResolver) Comment() string {
	return r.run.Comment
}

// TerraformVersion resolver
func (r *RunResolver) TerraformVersion() string {
	return r.run.TerraformVersion
}

// RunVariableResolver resolves a variable resource
type RunVariableResolver struct {
	variable *run.Variable
}

// Category resolver
func (r *RunVariableResolver) Category() string {
	return string(r.variable.Category)
}

// Hcl resolver
func (r *RunVariableResolver) Hcl() bool {
	return r.variable.Hcl
}

// NamespacePath resolver
func (r *RunVariableResolver) NamespacePath() *string {
	return r.variable.NamespacePath
}

// Key resolver
func (r *RunVariableResolver) Key() string {
	return r.variable.Key
}

// Value resolver
func (r *RunVariableResolver) Value() *string {
	return r.variable.Value
}

func runQuery(ctx context.Context, args *RunQueryArgs) (*RunResolver, error) {
	runService := getRunService(ctx)

	run, err := runService.GetRun(ctx, gid.FromGlobalID(args.ID))
	if err != nil {
		if errors.ErrorCode(err) == errors.ENotFound {
			return nil, nil
		}
		return nil, err
	}

	if run == nil {
		return nil, nil
	}

	return &RunResolver{run: run}, nil
}

func runsQuery(ctx context.Context, args *RunConnectionQueryArgs) (*RunConnectionResolver, error) {
	if err := args.Validate(); err != nil {
		return nil, err
	}

	input := run.GetRunsInput{
		PaginationOptions: &db.PaginationOptions{First: args.First, Last: args.Last, After: args.After, Before: args.Before},
	}

	if args.WorkspaceID != nil && args.WorkspacePath != nil {
		return nil, fmt.Errorf("only workspaceId or workspacePath can be set")
	} else if args.WorkspacePath != nil {
		// Find workspace with path
		ws, err := getWorkspaceService(ctx).GetWorkspaceByFullPath(ctx, *args.WorkspacePath)
		if err != nil {
			return nil, err
		}

		input.Workspace = ws
	} else if args.WorkspaceID != nil {
		// Find workspace with ID
		ws, err := getWorkspaceService(ctx).GetWorkspaceByID(ctx, gid.FromGlobalID(*args.WorkspaceID))
		if err != nil {
			return nil, err
		}

		input.Workspace = ws
	}

	if args.Sort != nil {
		sort := db.RunSortableField(*args.Sort)
		input.Sort = &sort
	}

	return NewRunConnectionResolver(ctx, &input)
}

/* Run Mutations */

// RunMutationPayload is the response payload for a run mutation
type RunMutationPayload struct {
	ClientMutationID *string
	Run              *models.Run
	Problems         []Problem
}

// RunMutationPayloadResolver resolves a RunMutationPayload
type RunMutationPayloadResolver struct {
	RunMutationPayload
}

// Run field resolver
func (r *RunMutationPayloadResolver) Run(ctx context.Context) *RunResolver {
	if r.RunMutationPayload.Run == nil {
		return nil
	}
	return &RunResolver{run: r.RunMutationPayload.Run}
}

// CreateRunInput is the input for creating a run
type CreateRunInput struct {
	ClientMutationID       *string
	ConfigurationVersionID *string
	IsDestroy              *bool
	ModuleSource           *string
	ModuleVersion          *string
	Comment                *string
	Variables              *[]struct {
		Key      string
		Value    string
		Category string
		Hcl      bool
	}
	TerraformVersion *string
	WorkspacePath    string
}

// ApplyRunInput is the input for applying a run
type ApplyRunInput struct {
	ClientMutationID *string
	Comment          *string
	RunID            string
}

// CancelRunInput is the input for cancelling a run
type CancelRunInput struct {
	ClientMutationID *string
	Comment          *string
	Force            *bool
	RunID            string
}

func handleRunMutationProblem(e error, clientMutationID *string) (*RunMutationPayloadResolver, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}
	payload := RunMutationPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}
	return &RunMutationPayloadResolver{RunMutationPayload: payload}, nil
}

func createRunMutation(ctx context.Context, input *CreateRunInput) (*RunMutationPayloadResolver, error) {
	ws, err := getWorkspaceService(ctx).GetWorkspaceByFullPath(ctx, input.WorkspacePath)
	if err != nil {
		return nil, err
	}

	var cvID *string
	if input.ConfigurationVersionID != nil {
		id := gid.FromGlobalID(*input.ConfigurationVersionID)
		cvID = &id
	}

	var terraformVersion string
	if input.TerraformVersion != nil {
		terraformVersion = *input.TerraformVersion
	}

	runOptions := &run.CreateRunInput{
		WorkspaceID:            ws.Metadata.ID,
		ConfigurationVersionID: cvID,
		ModuleSource:           input.ModuleSource,
		ModuleVersion:          input.ModuleVersion,
		Comment:                input.Comment,
		TerraformVersion:       terraformVersion,
	}

	if input.Variables != nil {
		variables := []run.Variable{}

		for _, v := range *input.Variables {
			vCopy := v
			variables = append(variables, run.Variable{
				Key:      v.Key,
				Value:    &vCopy.Value,
				Hcl:      v.Hcl,
				Category: models.VariableCategory(v.Category),
			})
		}

		runOptions.Variables = variables
	}

	if input.IsDestroy != nil {
		runOptions.IsDestroy = *input.IsDestroy
	}

	run, err := getRunService(ctx).CreateRun(ctx, runOptions)
	if err != nil {
		return nil, err
	}

	payload := RunMutationPayload{ClientMutationID: input.ClientMutationID, Run: run, Problems: []Problem{}}
	return &RunMutationPayloadResolver{RunMutationPayload: payload}, nil
}

func applyRunMutation(ctx context.Context, input *ApplyRunInput) (*RunMutationPayloadResolver, error) {
	run, err := getRunService(ctx).ApplyRun(ctx, gid.FromGlobalID(input.RunID), input.Comment)
	if err != nil {
		return nil, err
	}

	payload := RunMutationPayload{ClientMutationID: input.ClientMutationID, Run: run, Problems: []Problem{}}
	return &RunMutationPayloadResolver{RunMutationPayload: payload}, nil
}

func cancelRunMutation(ctx context.Context, input *CancelRunInput) (*RunMutationPayloadResolver, error) {
	force := false
	if input.Force != nil {
		force = *input.Force
	}
	run, err := getRunService(ctx).CancelRun(ctx, &run.CancelRunInput{
		RunID:   gid.FromGlobalID(input.RunID),
		Comment: input.Comment,
		Force:   force,
	})
	if err != nil {
		return nil, err
	}

	payload := RunMutationPayload{ClientMutationID: input.ClientMutationID, Run: run, Problems: []Problem{}}
	return &RunMutationPayloadResolver{RunMutationPayload: payload}, nil
}

/* Run Subscriptions */

// RunEventResolver resolves a run event
type RunEventResolver struct {
	event *run.Event
}

// Action resolves the event action
func (r *RunEventResolver) Action() string {
	return r.event.Action
}

// Run resolves the run
func (r *RunEventResolver) Run() *RunResolver {
	return &RunResolver{run: &r.event.Run}
}

// RunSubscriptionInput is the input for subscribing to run events
type RunSubscriptionInput struct {
	RunID         *string
	WorkspacePath string
}

func (r RootResolver) workspaceRunEventsSubscription(ctx context.Context, input *RunSubscriptionInput) (<-chan *RunEventResolver, error) {
	runService := getRunService(ctx)

	workspace, err := getWorkspaceService(ctx).GetWorkspaceByFullPath(ctx, input.WorkspacePath)
	if err != nil {
		return nil, err
	}
	workspaceID := workspace.Metadata.ID

	var runID *string
	if input.RunID != nil {
		id := gid.FromGlobalID(*input.RunID)
		runID = &id
	}

	events, err := runService.SubscribeToRunEvents(ctx, &run.EventSubscriptionOptions{
		WorkspaceID: &workspaceID,
		RunID:       runID,
	})
	if err != nil {
		return nil, err
	}

	outgoing := make(chan *RunEventResolver)

	go func() {
		for event := range events {
			outgoing <- &RunEventResolver{event: event}
		}

		close(outgoing)
	}()

	return outgoing, nil
}

/* Run loader */

const runLoaderKey = "run"

// RegisterRunLoader registers a run loader function
func RegisterRunLoader(collection *loader.Collection) {
	collection.Register(runLoaderKey, runBatchFunc)
}

func loadRun(ctx context.Context, id string) (*models.Run, error) {
	ldr, err := loader.Extract(ctx, runLoaderKey)
	if err != nil {
		return nil, err
	}

	data, err := ldr.Load(ctx, dataloader.StringKey(id))()
	if err != nil {
		return nil, err
	}

	sv, ok := data.(models.Run)
	if !ok {
		return nil, errors.NewError(errors.EInternal, "Wrong type")
	}

	return &sv, nil
}

func runBatchFunc(ctx context.Context, ids []string) (loader.DataBatch, error) {
	runService := getRunService(ctx)

	runs, err := runService.GetRunsByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	// Build map of results
	batch := loader.DataBatch{}
	for _, result := range runs {
		batch[result.Metadata.ID] = result
	}

	return batch, nil
}

// The End.
