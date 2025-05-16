package resolver

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/loader"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/logstream"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/runner"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"

	"github.com/aws/smithy-go/ptr"
	"github.com/graph-gophers/dataloader"
	graphql "github.com/graph-gophers/graphql-go"
)

/* RunnerSession Query Resolvers */

// RunnerSessionEdgeResolver resolves session edges
type RunnerSessionEdgeResolver struct {
	edge Edge
}

// Cursor returns an opaque cursor
func (r *RunnerSessionEdgeResolver) Cursor() (string, error) {
	session, ok := r.edge.Node.(models.RunnerSession)
	if !ok {
		return "", errors.New("Failed to convert node type")
	}
	cursor, err := r.edge.CursorFunc(&session)
	return *cursor, err
}

// Node returns a session node
func (r *RunnerSessionEdgeResolver) Node() (*RunnerSessionResolver, error) {
	session, ok := r.edge.Node.(models.RunnerSession)
	if !ok {
		return nil, errors.New("Failed to convert node type")
	}

	return &RunnerSessionResolver{session: &session}, nil
}

// RunnerSessionConnectionResolver resolves a session connection
type RunnerSessionConnectionResolver struct {
	connection Connection
}

// NewRunnerSessionConnectionResolver creates a new RunnerSessionConnectionResolver
func NewRunnerSessionConnectionResolver(ctx context.Context, input *runner.GetRunnerSessionsInput) (*RunnerSessionConnectionResolver, error) {
	service := getServiceCatalog(ctx).RunnerService

	result, err := service.GetRunnerSessions(ctx, input)
	if err != nil {
		return nil, err
	}

	sessions := result.RunnerSessions

	// Create edges
	edges := make([]Edge, len(sessions))
	for i, session := range sessions {
		edges[i] = Edge{CursorFunc: result.PageInfo.Cursor, Node: session}
	}

	pageInfo := PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
	}

	if len(sessions) > 0 {
		var err error
		pageInfo.StartCursor, err = result.PageInfo.Cursor(&sessions[0])
		if err != nil {
			return nil, err
		}

		pageInfo.EndCursor, err = result.PageInfo.Cursor(&sessions[len(edges)-1])
		if err != nil {
			return nil, err
		}
	}

	connection := Connection{
		TotalCount: result.PageInfo.TotalCount,
		PageInfo:   pageInfo,
		Edges:      edges,
	}

	return &RunnerSessionConnectionResolver{connection: connection}, nil
}

// TotalCount returns the total result count for the connection
func (r *RunnerSessionConnectionResolver) TotalCount() int32 {
	return r.connection.TotalCount
}

// PageInfo returns the connection page information
func (r *RunnerSessionConnectionResolver) PageInfo() *PageInfoResolver {
	return &PageInfoResolver{pageInfo: r.connection.PageInfo}
}

// Edges returns the connection edges
func (r *RunnerSessionConnectionResolver) Edges() *[]*RunnerSessionEdgeResolver {
	resolvers := make([]*RunnerSessionEdgeResolver, len(r.connection.Edges))
	for i, edge := range r.connection.Edges {
		resolvers[i] = &RunnerSessionEdgeResolver{edge: edge}
	}
	return &resolvers
}

// RunnerSessionResolver resolves a session resource
type RunnerSessionResolver struct {
	session *models.RunnerSession
}

// ID resolver
func (r *RunnerSessionResolver) ID() graphql.ID {
	return graphql.ID(r.session.GetGlobalID())
}

// Metadata resolver
func (r *RunnerSessionResolver) Metadata() *MetadataResolver {
	return &MetadataResolver{metadata: &r.session.Metadata}
}

// LastContacted resolver
func (r *RunnerSessionResolver) LastContacted() graphql.Time {
	return graphql.Time{Time: r.session.LastContactTimestamp}
}

// Active resolver
func (r *RunnerSessionResolver) Active() bool {
	return r.session.Active()
}

// Internal resolver
func (r *RunnerSessionResolver) Internal() bool {
	return r.session.Internal
}

// ErrorCount resolver
func (r *RunnerSessionResolver) ErrorCount() int32 {
	return int32(r.session.ErrorCount)
}

// Runner resolver
func (r *RunnerSessionResolver) Runner(ctx context.Context) (*RunnerResolver, error) {
	runner, err := loadRunner(ctx, r.session.RunnerID)
	if err != nil {
		return nil, err
	}
	return &RunnerResolver{runner: runner}, nil
}

// ErrorLog resolver
func (r *RunnerSessionResolver) ErrorLog(ctx context.Context) (*RunnerSessionErrorLogResolver, error) {
	logStream, err := loadRunnerSessionLogStream(ctx, r.session.Metadata.ID)
	if err != nil {
		return nil, err
	}

	return &RunnerSessionErrorLogResolver{stream: logStream}, nil
}

// RunnerSessionErrorLogResolver resolves a session error log
type RunnerSessionErrorLogResolver struct {
	stream *models.LogStream
}

// LastUpdatedAt resolver
func (r *RunnerSessionErrorLogResolver) LastUpdatedAt() (*graphql.Time, error) {
	return &graphql.Time{Time: *r.stream.Metadata.LastUpdatedTimestamp}, nil
}

// Size resolver
func (r *RunnerSessionErrorLogResolver) Size() (int32, error) {
	return int32(r.stream.Size), nil
}

// Data resolver
func (r *RunnerSessionErrorLogResolver) Data(ctx context.Context, args *JobLogsQueryArgs) (string, error) {
	buffer, err := getServiceCatalog(ctx).RunnerService.ReadRunnerSessionErrorLog(ctx, *r.stream.RunnerSessionID, int(args.StartOffset), int(args.Limit))
	if err != nil {
		return "", err
	}
	return string(buffer), nil
}

/* Create Runner Session Mutation */

// CreateRunnerSessionMutationPayload is the response payload for a create runner session mutation.
type CreateRunnerSessionMutationPayload struct {
	ClientMutationID *string
	RunnerSession    *models.RunnerSession
	Problems         []Problem
}

// CreateRunnerSessionMutationPayloadResolver resolves a CreateRunnerSessionMutationPayload.
type CreateRunnerSessionMutationPayloadResolver struct {
	CreateRunnerSessionMutationPayload
}

// RunnerSession field resolver
func (s *CreateRunnerSessionMutationPayloadResolver) RunnerSession() *RunnerSessionResolver {
	if s.CreateRunnerSessionMutationPayload.RunnerSession == nil {
		return nil
	}
	return &RunnerSessionResolver{
		session: s.CreateRunnerSessionMutationPayload.RunnerSession,
	}
}

// CreateRunnerSessionInput is the input for a create runner session mutation.
type CreateRunnerSessionInput struct {
	ClientMutationID *string
	RunnerPath       string
}

func handleCreateRunnerSessionMutationProblem(e error,
	clientMutationID *string) (*CreateRunnerSessionMutationPayloadResolver, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}

	payload := CreateRunnerSessionMutationPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}
	return &CreateRunnerSessionMutationPayloadResolver{CreateRunnerSessionMutationPayload: payload}, nil
}

func createRunnerSessionMutation(ctx context.Context,
	input *CreateRunnerSessionInput) (*CreateRunnerSessionMutationPayloadResolver, error) {
	createdRunnerSession, err := getServiceCatalog(ctx).RunnerService.CreateRunnerSession(ctx, &runner.CreateRunnerSessionInput{
		RunnerPath: input.RunnerPath,
	})
	if err != nil {
		return nil, err
	}

	payload := CreateRunnerSessionMutationPayload{
		ClientMutationID: input.ClientMutationID,
		RunnerSession:    createdRunnerSession,
		Problems:         []Problem{},
	}
	return &CreateRunnerSessionMutationPayloadResolver{CreateRunnerSessionMutationPayload: payload}, nil
}

/* Send Runner Session Heartbeat and Send Runner Session Error Mutations */

// RunnerSessionHeartbeatErrorMutationPayload is the response payload for a sending a heartbeat or error.
type RunnerSessionHeartbeatErrorMutationPayload struct {
	ClientMutationID *string
	Problems         []Problem
}

// RunnerSessionHeartbeatErrorMutationPayloadResolver resolves
// a RunnerSessionHeartbeatErrorMutationPayload.
type RunnerSessionHeartbeatErrorMutationPayloadResolver struct {
	RunnerSessionHeartbeatErrorMutationPayload
}

// RunnerSessionHeartbeatInput is the input for sending a heartbeat.
type RunnerSessionHeartbeatInput struct {
	ClientMutationID *string
	RunnerSessionID  string
}

// CreateRunnerSessionErrorInput is the input for sending an error.
type CreateRunnerSessionErrorInput struct {
	ClientMutationID *string
	RunnerSessionID  string
	ErrorMessage     string
}

func handleRunnerSessionHeartbeatErrorMutationProblem(e error,
	clientMutationID *string) (*RunnerSessionHeartbeatErrorMutationPayloadResolver, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}

	payload := RunnerSessionHeartbeatErrorMutationPayload{
		ClientMutationID: clientMutationID,
		Problems:         []Problem{*problem},
	}
	return &RunnerSessionHeartbeatErrorMutationPayloadResolver{
		RunnerSessionHeartbeatErrorMutationPayload: payload,
	}, nil
}

func runnerSessionHeartbeatMutation(ctx context.Context,
	input *RunnerSessionHeartbeatInput) (*RunnerSessionHeartbeatErrorMutationPayloadResolver, error) {
	serviceCatalog := getServiceCatalog(ctx)

	sessionID, err := serviceCatalog.FetchModelID(ctx, input.RunnerSessionID)
	if err != nil {
		return nil, err
	}

	err = serviceCatalog.RunnerService.AcceptRunnerSessionHeartbeat(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	payload := RunnerSessionHeartbeatErrorMutationPayload{
		Problems: []Problem{},
	}
	return &RunnerSessionHeartbeatErrorMutationPayloadResolver{
		RunnerSessionHeartbeatErrorMutationPayload: payload,
	}, nil
}

func createRunnerSessionErrorMutation(ctx context.Context,
	input *CreateRunnerSessionErrorInput) (*RunnerSessionHeartbeatErrorMutationPayloadResolver, error) {
	serviceCatalog := getServiceCatalog(ctx)

	sessionID, err := serviceCatalog.FetchModelID(ctx, input.RunnerSessionID)
	if err != nil {
		return nil, err
	}

	err = serviceCatalog.RunnerService.CreateRunnerSessionError(ctx, sessionID, input.ErrorMessage)
	if err != nil {
		return nil, err
	}

	payload := RunnerSessionHeartbeatErrorMutationPayload{
		Problems: []Problem{},
	}
	return &RunnerSessionHeartbeatErrorMutationPayloadResolver{
		RunnerSessionHeartbeatErrorMutationPayload: payload,
	}, nil
}

/* RunnerSession Subscriptions */

// RunnerSessionEventResolver resolves a runner session event
type RunnerSessionEventResolver struct {
	event *runner.SessionEvent
}

// Action resolves the event action
func (r *RunnerSessionEventResolver) Action() string {
	return r.event.Action
}

// RunnerSession resolves the runner session
func (r *RunnerSessionEventResolver) RunnerSession() *RunnerSessionResolver {
	return &RunnerSessionResolver{session: r.event.RunnerSession}
}

// RunnerSessionErrorLogEventResolver resolves a session error log event
type RunnerSessionErrorLogEventResolver struct {
	event *logstream.LogEvent
}

// Completed resolver
func (j *RunnerSessionErrorLogEventResolver) Completed() bool {
	return j.event.Completed
}

// Size resolver
func (j *RunnerSessionErrorLogEventResolver) Size() int32 {
	return int32(j.event.Size)
}

// RunnerSessionEventSubscriptionInput is the input for subscribing to runner sessions
type RunnerSessionEventSubscriptionInput struct {
	GroupID    *string
	RunnerID   *string
	RunnerType *string
}

func (r RootResolver) runnerSessionEventsSubscription(ctx context.Context,
	input *RunnerSessionEventSubscriptionInput) (<-chan *RunnerSessionEventResolver, error) {
	serviceCatalog := getServiceCatalog(ctx)

	var groupID, runnerID *string
	var runnerType *models.RunnerType

	if input.GroupID != nil {
		id, err := serviceCatalog.FetchModelID(ctx, *input.GroupID)
		if err != nil {
			return nil, err
		}

		groupID = &id
	}

	if input.RunnerID != nil {
		id, err := serviceCatalog.FetchModelID(ctx, *input.RunnerID)
		if err != nil {
			return nil, err
		}

		runnerID = &id
	}

	if input.RunnerType != nil {
		typ := models.RunnerType(*input.RunnerType)
		runnerType = &typ
	}

	events, err := serviceCatalog.RunnerService.SubscribeToRunnerSessions(ctx, &runner.SubscribeToRunnerSessionsInput{
		GroupID:    groupID,
		RunnerID:   runnerID,
		RunnerType: runnerType,
	})
	if err != nil {
		return nil, err
	}

	outgoing := make(chan *RunnerSessionEventResolver)

	go func() {
		for event := range events {
			select {
			case <-ctx.Done():
			case outgoing <- &RunnerSessionEventResolver{event: event}:
			}
		}

		close(outgoing)
	}()

	return outgoing, nil
}

// RunnerSessionErrorLogSubscriptionInput is the input for subscribing to runner session error logs
type RunnerSessionErrorLogSubscriptionInput struct {
	LastSeenLogSize *int32
	RunnerSessionID string
}

func (r RootResolver) runnerSessionErrorLogSubscription(ctx context.Context, input *RunnerSessionErrorLogSubscriptionInput) (<-chan *RunnerSessionErrorLogEventResolver, error) {
	serviceCatalog := getServiceCatalog(ctx)

	sessionID, err := serviceCatalog.FetchModelID(ctx, input.RunnerSessionID)
	if err != nil {
		return nil, err
	}

	options := &runner.SubscribeToRunnerSessionErrorLogInput{
		RunnerSessionID: sessionID,
	}
	if input.LastSeenLogSize != nil {
		options.LastSeenLogSize = ptr.Int(int(*input.LastSeenLogSize))
	}

	events, err := serviceCatalog.RunnerService.SubscribeToRunnerSessionErrorLog(ctx, options)
	if err != nil {
		return nil, err
	}

	outgoing := make(chan *RunnerSessionErrorLogEventResolver)

	go func() {
		for event := range events {
			select {
			case <-ctx.Done():
			case outgoing <- &RunnerSessionErrorLogEventResolver{event: event}:
			}
		}

		close(outgoing)
	}()

	return outgoing, nil
}

/* RunnerSessionLogStream loader */

const runnerSessionLogStreamLoaderKey = "runnerSessionLogStream"

// RegisterRunnerSessionLogStreamLoader registers a runnerSessionLogStream loader function
func RegisterRunnerSessionLogStreamLoader(collection *loader.Collection) {
	collection.Register(runnerSessionLogStreamLoaderKey, runnerSessionLogStreamBatchFunc)
}

func loadRunnerSessionLogStream(ctx context.Context, id string) (*models.LogStream, error) {
	ldr, err := loader.Extract(ctx, runnerSessionLogStreamLoaderKey)
	if err != nil {
		return nil, err
	}

	data, err := ldr.Load(ctx, dataloader.StringKey(id))()
	if err != nil {
		return nil, err
	}

	runnerSessionLogStream, ok := data.(models.LogStream)
	if !ok {
		return nil, errors.New("Wrong type")
	}

	return &runnerSessionLogStream, nil
}

func runnerSessionLogStreamBatchFunc(ctx context.Context, ids []string) (loader.DataBatch, error) {
	runnerSessionLogStreams, err := getServiceCatalog(ctx).RunnerService.GetLogStreamsByRunnerSessionIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	// Build map of results
	batch := loader.DataBatch{}
	for _, result := range runnerSessionLogStreams {
		// Use runner session ID as the key since that is the ID which was
		// used to query the data
		batch[*result.RunnerSessionID] = result
	}

	return batch, nil
}
