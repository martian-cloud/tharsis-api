package resolver

import (
	"context"

	"github.com/aws/smithy-go/ptr"
	"github.com/graph-gophers/dataloader"
	graphql "github.com/graph-gophers/graphql-go"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/loader"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/job"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

/* Job Query Resolvers */

// JobConnectionQueryArgs are used to query a list of jobs
type JobConnectionQueryArgs struct {
	ConnectionQueryArgs
	WorkspacePath *string
	JobStatus     *models.JobStatus
	JobType       *models.JobType
}

// JobQueryArgs are used to query a single job
type JobQueryArgs struct {
	ID string
}

// JobLogsQueryArgs contains the options for querying job logs
type JobLogsQueryArgs struct {
	StartOffset int32
	Limit       int32
}

// JobEdgeResolver resolves a job edge
type JobEdgeResolver struct {
	edge Edge
}

// Cursor returns an opaque cursor
func (r *JobEdgeResolver) Cursor() (string, error) {
	job, ok := r.edge.Node.(models.Job)
	if !ok {
		return "", errors.New("Failed to convert node type")
	}
	cursor, err := r.edge.CursorFunc(&job)
	return *cursor, err
}

// Node returns a job node
func (r *JobEdgeResolver) Node() (*JobResolver, error) {
	job, ok := r.edge.Node.(models.Job)
	if !ok {
		return nil, errors.New("Failed to convert node type")
	}

	return &JobResolver{job: &job}, nil
}

// JobConnectionResolver resolves a job connection
type JobConnectionResolver struct {
	connection Connection
}

// NewJobConnectionResolver creates a new JobConnectionResolver
func NewJobConnectionResolver(ctx context.Context, input *job.GetJobsInput) (*JobConnectionResolver, error) {
	jobService := getJobService(ctx)

	result, err := jobService.GetJobs(ctx, input)
	if err != nil {
		return nil, err
	}

	jobs := result.Jobs

	// Create edges
	edges := make([]Edge, len(jobs))
	for i, job := range jobs {
		edges[i] = Edge{CursorFunc: result.PageInfo.Cursor, Node: job}
	}

	pageInfo := PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
	}

	if len(jobs) > 0 {
		var err error
		pageInfo.StartCursor, err = result.PageInfo.Cursor(&jobs[0])
		if err != nil {
			return nil, err
		}

		pageInfo.EndCursor, err = result.PageInfo.Cursor(&jobs[len(edges)-1])
		if err != nil {
			return nil, err
		}
	}

	connection := Connection{
		TotalCount: result.PageInfo.TotalCount,
		PageInfo:   pageInfo,
		Edges:      edges,
	}

	return &JobConnectionResolver{connection: connection}, nil
}

// TotalCount returns the total result count for the connection
func (r *JobConnectionResolver) TotalCount() int32 {
	return r.connection.TotalCount
}

// PageInfo returns the connection page information
func (r *JobConnectionResolver) PageInfo() *PageInfoResolver {
	return &PageInfoResolver{pageInfo: r.connection.PageInfo}
}

// Edges returns the connection edges
func (r *JobConnectionResolver) Edges() *[]*JobEdgeResolver {
	resolvers := make([]*JobEdgeResolver, len(r.connection.Edges))
	for i, edge := range r.connection.Edges {
		resolvers[i] = &JobEdgeResolver{edge: edge}
	}
	return &resolvers
}

// JobTimestampsResolver resolves a job's timestamps
type JobTimestampsResolver struct {
	timestamps *models.JobTimestamps
}

// QueuedAt resolver
func (r *JobTimestampsResolver) QueuedAt() *graphql.Time {
	if r.timestamps.QueuedTimestamp == nil {
		return nil
	}
	return &graphql.Time{Time: *r.timestamps.QueuedTimestamp}
}

// PendingAt resolver
func (r *JobTimestampsResolver) PendingAt() *graphql.Time {
	if r.timestamps.PendingTimestamp == nil {
		return nil
	}
	return &graphql.Time{Time: *r.timestamps.PendingTimestamp}
}

// RunningAt resolver
func (r *JobTimestampsResolver) RunningAt() *graphql.Time {
	if r.timestamps.RunningTimestamp == nil {
		return nil
	}
	return &graphql.Time{Time: *r.timestamps.RunningTimestamp}
}

// FinishedAt resolver
func (r *JobTimestampsResolver) FinishedAt() *graphql.Time {
	if r.timestamps.FinishedTimestamp == nil {
		return nil
	}
	return &graphql.Time{Time: *r.timestamps.FinishedTimestamp}
}

// JobResolver resolves a job resource
type JobResolver struct {
	job *models.Job
}

// ID resolver
func (r *JobResolver) ID() graphql.ID {
	return graphql.ID(gid.ToGlobalID(gid.JobType, r.job.Metadata.ID))
}

// Status resolver
func (r *JobResolver) Status() models.JobStatus {
	return r.job.Status
}

// Type resolver
func (r *JobResolver) Type() string {
	return string(r.job.Type)
}

// RunnerPath resolver
func (r *JobResolver) RunnerPath() *string {
	return r.job.RunnerPath
}

// Runner resolver
func (r *JobResolver) Runner(ctx context.Context) (*RunnerResolver, error) {
	if r.job.RunnerID == nil {
		return nil, nil
	}
	runner, err := loadRunner(ctx, *r.job.RunnerID)
	if err != nil {
		// Check for not found since runner may have been deleted
		if errors.ErrorCode(err) == errors.ENotFound {
			return nil, nil
		}
		return nil, err
	}

	return &RunnerResolver{runner: runner}, nil
}

// Run resolver
func (r *JobResolver) Run(ctx context.Context) (*RunResolver, error) {
	run, err := loadRun(ctx, r.job.RunID)
	if err != nil {
		return nil, err
	}

	return &RunResolver{run: run}, nil
}

// Workspace resolver
func (r *JobResolver) Workspace(ctx context.Context) (*WorkspaceResolver, error) {
	workspace, err := loadWorkspace(ctx, r.job.WorkspaceID)
	if err != nil {
		return nil, err
	}

	return &WorkspaceResolver{workspace: workspace}, nil
}

// CancelRequested resolver
func (r *JobResolver) CancelRequested() bool {
	return r.job.CancelRequested
}

// Metadata resolver
func (r *JobResolver) Metadata() *MetadataResolver {
	return &MetadataResolver{metadata: &r.job.Metadata}
}

// Timestamps resolver
func (r *JobResolver) Timestamps() *JobTimestampsResolver {
	return &JobTimestampsResolver{timestamps: &r.job.Timestamps}
}

// LogLastUpdatedAt resolver
func (r *JobResolver) LogLastUpdatedAt(ctx context.Context) (*graphql.Time, error) {
	descriptor, err := getJobService(ctx).GetJobLogDescriptor(ctx, r.job)
	if err != nil {
		return nil, err
	}
	if descriptor == nil {
		return nil, nil
	}
	return &graphql.Time{Time: *descriptor.Metadata.LastUpdatedTimestamp}, nil
}

// MaxJobDuration resolver
func (r *JobResolver) MaxJobDuration() int32 {
	return r.job.MaxJobDuration
}

// LogSize resolver
func (r *JobResolver) LogSize(ctx context.Context) (int32, error) {
	descriptor, err := getJobService(ctx).GetJobLogDescriptor(ctx, r.job)
	if err != nil {
		return 0, err
	}
	if descriptor == nil {
		return 0, nil
	}
	return int32(descriptor.Size), nil
}

// Logs resolver
func (r *JobResolver) Logs(ctx context.Context, args *JobLogsQueryArgs) (string, error) {
	buffer, err := getJobService(ctx).GetLogs(ctx, r.job.Metadata.ID, int(args.StartOffset), int(args.Limit))
	if err != nil {
		return "", err
	}
	return string(buffer), nil
}

/* Job Subscriptions */

// JobLogEventResolver resolves a job log event
type JobLogEventResolver struct {
	event *job.LogEvent
}

// Action resolver
func (j *JobLogEventResolver) Action() string {
	return j.event.Action
}

// Size resolver
func (j *JobLogEventResolver) Size() int32 {
	return int32(j.event.Size)
}

func jobQuery(ctx context.Context, args *JobQueryArgs) (*JobResolver, error) {
	jobService := getJobService(ctx)

	job, err := jobService.GetJob(ctx, gid.FromGlobalID(args.ID))
	if err != nil {
		if errors.ErrorCode(err) == errors.ENotFound {
			return nil, nil
		}
		return nil, err
	}

	return &JobResolver{job: job}, nil
}

func jobsQuery(ctx context.Context, args *JobConnectionQueryArgs) (*JobConnectionResolver, error) {
	if err := args.Validate(); err != nil {
		return nil, err
	}

	input := &job.GetJobsInput{
		PaginationOptions: &pagination.Options{First: args.First, Last: args.Last, After: args.After, Before: args.Before},
		Status:            args.JobStatus,
		Type:              args.JobType,
	}

	if args.WorkspacePath != nil {
		workspace, err := getWorkspaceService(ctx).GetWorkspaceByFullPath(ctx, *args.WorkspacePath)
		if err != nil {
			return nil, err
		}

		input.WorkspaceID = &workspace.Metadata.ID
	}

	if args.Sort != nil {
		sort := db.JobSortableField(*args.Sort)
		input.Sort = &sort
	}

	return NewJobConnectionResolver(ctx, input)
}

// JobCancellationEventResolver resolves a job cancellation event
type JobCancellationEventResolver struct {
	event *job.CancellationEvent
}

// Job resolves a job
func (r *JobCancellationEventResolver) Job() *JobResolver {
	return &JobResolver{job: &r.event.Job}
}

// JobLogSubscriptionInput is the input for subscribing to job log events
type JobLogSubscriptionInput struct {
	LastSeenLogSize *int32
	JobID           string
}

func (r RootResolver) jobLogEventsSubscription(ctx context.Context, input *JobLogSubscriptionInput) (<-chan *JobLogEventResolver, error) {
	service := getJobService(ctx)

	j, err := service.GetJob(ctx, gid.FromGlobalID(input.JobID))
	if err != nil {
		return nil, err
	}

	options := &job.LogEventSubscriptionOptions{}
	if input.LastSeenLogSize != nil {
		options.LastSeenLogSize = ptr.Int(int(*input.LastSeenLogSize))
	}

	events, err := service.SubscribeToJobLogEvents(ctx, j, options)
	if err != nil {
		return nil, err
	}

	outgoing := make(chan *JobLogEventResolver)

	go func() {
		for event := range events {
			select {
			case <-ctx.Done():
			case outgoing <- &JobLogEventResolver{event: event}:
			}
		}

		close(outgoing)
	}()

	return outgoing, nil
}

// JobCancellationEventSubscriptionInput is the input for subscribing to job cancellation events
type JobCancellationEventSubscriptionInput struct {
	JobID string
}

func (r RootResolver) jobCancellationEventSubscription(ctx context.Context, input *JobCancellationEventSubscriptionInput) (<-chan *JobCancellationEventResolver, error) {
	service := getJobService(ctx)

	events, err := service.SubscribeToCancellationEvent(ctx, &job.CancellationSubscriptionsOptions{
		JobID: gid.FromGlobalID(input.JobID),
	})
	if err != nil {
		return nil, err
	}

	outgoing := make(chan *JobCancellationEventResolver)

	go func() {
		for event := range events {
			select {
			case <-ctx.Done():
			case outgoing <- &JobCancellationEventResolver{event: event}:
			}
		}

		close(outgoing)
	}()

	return outgoing, nil
}

/* Job Mutation Resolvers */

// ClaimJobMutationPayload is the response payload for the claim job mutation
type ClaimJobMutationPayload struct {
	ClientMutationID *string
	Token            *string
	JobID            *string
	Problems         []Problem
}

// ClaimJobInput is the input for claiming a job
type ClaimJobInput struct {
	ClientMutationID *string
	RunnerPath       string
}

// SaveJobLogsPayload is the response payload for a save logs mutation
type SaveJobLogsPayload struct {
	ClientMutationID *string
	Problems         []Problem
}

// SaveJobLogsInput is the input for saving logs
type SaveJobLogsInput struct {
	ClientMutationID *string
	Logs             string
	JobID            string
	StartOffset      int32
}

func handleClaimJobMutationProblem(e error, clientMutationID *string) (*ClaimJobMutationPayload, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}

	payload := ClaimJobMutationPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}
	return &payload, nil
}

func handleSaveJobLogsMutationProblem(e error, clientMutationID *string) (*SaveJobLogsPayload, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}

	return &SaveJobLogsPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}, nil
}

func claimJobMutation(ctx context.Context, input *ClaimJobInput) (*ClaimJobMutationPayload, error) {
	jobService := getJobService(ctx)

	resp, err := jobService.ClaimJob(ctx, input.RunnerPath)
	if err != nil {
		return nil, err
	}

	payload := ClaimJobMutationPayload{
		ClientMutationID: input.ClientMutationID,
		JobID:            ptr.String(gid.ToGlobalID(gid.JobType, resp.JobID)),
		Token:            &resp.Token,
		Problems:         []Problem{},
	}
	return &payload, nil
}

func saveJobLogsMutation(ctx context.Context, input *SaveJobLogsInput) (*SaveJobLogsPayload, error) {
	jobService := getJobService(ctx)
	logs := []byte(input.Logs)

	err := jobService.SaveLogs(ctx, gid.FromGlobalID(input.JobID), int(input.StartOffset), logs)
	if err != nil {
		return nil, err
	}

	return &SaveJobLogsPayload{ClientMutationID: input.ClientMutationID, Problems: []Problem{}}, nil
}

/* Job loader */

const jobLoaderKey = "job"

// RegisterJobLoader registers a job loader function
func RegisterJobLoader(collection *loader.Collection) {
	collection.Register(jobLoaderKey, jobBatchFunc)
}

func loadJob(ctx context.Context, id string) (*models.Job, error) {
	ldr, err := loader.Extract(ctx, jobLoaderKey)
	if err != nil {
		return nil, err
	}

	data, err := ldr.Load(ctx, dataloader.StringKey(id))()
	if err != nil {
		return nil, err
	}

	job, ok := data.(models.Job)
	if !ok {
		return nil, errors.New("Wrong type")
	}

	return &job, nil
}

func jobBatchFunc(ctx context.Context, ids []string) (loader.DataBatch, error) {
	groups, err := getJobService(ctx).GetJobsByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	// Build map of results
	batch := loader.DataBatch{}
	for _, result := range groups {
		batch[result.Metadata.ID] = result
	}

	return batch, nil
}
