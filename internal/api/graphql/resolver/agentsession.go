package resolver

import (
	"context"
	"encoding/json"

	graphql "github.com/graph-gophers/graphql-go"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/agent"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	agentsvc "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/agent"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

/* AgentSession Resolver */

// AgentSessionResolver resolves an AgentSession
type AgentSessionResolver struct {
	session *models.AgentSession
}

// ID resolver
func (r *AgentSessionResolver) ID() graphql.ID {
	return graphql.ID(r.session.GetGlobalID())
}

// Metadata resolver
func (r *AgentSessionResolver) Metadata() *MetadataResolver {
	return &MetadataResolver{metadata: &r.session.Metadata}
}

// UserID resolver
func (r *AgentSessionResolver) UserID() string {
	return gid.ToGlobalID(types.UserModelType, r.session.UserID)
}

// TotalCredits resolver
func (r *AgentSessionResolver) TotalCredits() float64 {
	return r.session.TotalCredits
}

// Runs resolver
func (r *AgentSessionResolver) Runs(ctx context.Context, args *ConnectionQueryArgs) (*AgentSessionRunConnectionResolver, error) {
	if err := args.Validate(); err != nil {
		return nil, err
	}

	input := agentsvc.GetAgentSessionRunsInput{
		SessionID:         r.session.Metadata.ID,
		PaginationOptions: &pagination.Options{First: args.First, Last: args.Last, After: args.After, Before: args.Before},
	}

	if args.Sort != nil {
		sort := db.AgentSessionRunSortableField(*args.Sort)
		input.Sort = &sort
	}

	return NewAgentSessionRunConnectionResolver(ctx, &input)
}

/* AgentSessionEvent Union Resolver */

// AgentSessionEventResolver resolves the AgentSessionEvent union
type AgentSessionEventResolver struct {
	event agent.Event
}

// ToAgentSessionRunStartedEvent resolves the AgentSessionRunStartedEvent union member
func (r *AgentSessionEventResolver) ToAgentSessionRunStartedEvent() (*AgentSessionRunStartedEventResolver, bool) {
	e, ok := r.event.(*agent.RunStartedEvent)
	if !ok {
		return nil, false
	}
	return &AgentSessionRunStartedEventResolver{event: e}, true
}

// ToAgentSessionRunFinishedEvent resolves the AgentSessionRunFinishedEvent union member
func (r *AgentSessionEventResolver) ToAgentSessionRunFinishedEvent() (*AgentSessionRunFinishedEventResolver, bool) {
	e, ok := r.event.(*agent.RunFinishedEvent)
	if !ok {
		return nil, false
	}
	return &AgentSessionRunFinishedEventResolver{event: e}, true
}

// ToAgentSessionRunErrorEvent resolves the AgentSessionRunErrorEvent union member
func (r *AgentSessionEventResolver) ToAgentSessionRunErrorEvent() (*AgentSessionRunErrorEventResolver, bool) {
	e, ok := r.event.(*agent.RunErrorEvent)
	if !ok {
		return nil, false
	}
	return &AgentSessionRunErrorEventResolver{event: e}, true
}

// ToAgentSessionCustomEvent resolves the AgentSessionCustomEvent union member
func (r *AgentSessionEventResolver) ToAgentSessionCustomEvent() (*AgentSessionCustomEventResolver, bool) {
	e, ok := r.event.(*agent.CustomEvent)
	if !ok {
		return nil, false
	}
	return &AgentSessionCustomEventResolver{event: e}, true
}

// ToAgentSessionStepStartedEvent resolves the AgentSessionStepStartedEvent union member
func (r *AgentSessionEventResolver) ToAgentSessionStepStartedEvent() (*AgentSessionStepStartedEventResolver, bool) {
	e, ok := r.event.(*agent.StepStartedEvent)
	if !ok {
		return nil, false
	}
	return &AgentSessionStepStartedEventResolver{event: e}, true
}

// ToAgentSessionStepFinishedEvent resolves the AgentSessionStepFinishedEvent union member
func (r *AgentSessionEventResolver) ToAgentSessionStepFinishedEvent() (*AgentSessionStepFinishedEventResolver, bool) {
	e, ok := r.event.(*agent.StepFinishedEvent)
	if !ok {
		return nil, false
	}
	return &AgentSessionStepFinishedEventResolver{event: e}, true
}

// ToAgentSessionTextMessageStartEvent resolves the AgentSessionTextMessageStartEvent union member
func (r *AgentSessionEventResolver) ToAgentSessionTextMessageStartEvent() (*AgentSessionTextMessageStartEventResolver, bool) {
	e, ok := r.event.(*agent.TextMessageStartEvent)
	if !ok {
		return nil, false
	}
	return &AgentSessionTextMessageStartEventResolver{event: e}, true
}

// ToAgentSessionTextMessageContentEvent resolves the AgentSessionTextMessageContentEvent union member
func (r *AgentSessionEventResolver) ToAgentSessionTextMessageContentEvent() (*AgentSessionTextMessageContentEventResolver, bool) {
	e, ok := r.event.(*agent.TextMessageContentEvent)
	if !ok {
		return nil, false
	}
	return &AgentSessionTextMessageContentEventResolver{event: e}, true
}

// ToAgentSessionTextMessageEndEvent resolves the AgentSessionTextMessageEndEvent union member
func (r *AgentSessionEventResolver) ToAgentSessionTextMessageEndEvent() (*AgentSessionTextMessageEndEventResolver, bool) {
	e, ok := r.event.(*agent.TextMessageEndEvent)
	if !ok {
		return nil, false
	}
	return &AgentSessionTextMessageEndEventResolver{event: e}, true
}

// ToAgentSessionToolCallStartEvent resolves the AgentSessionToolCallStartEvent union member
func (r *AgentSessionEventResolver) ToAgentSessionToolCallStartEvent() (*AgentSessionToolCallStartEventResolver, bool) {
	e, ok := r.event.(*agent.ToolCallStartEvent)
	if !ok {
		return nil, false
	}
	return &AgentSessionToolCallStartEventResolver{event: e}, true
}

// ToAgentSessionToolCallArgsEvent resolves the AgentSessionToolCallArgsEvent union member
func (r *AgentSessionEventResolver) ToAgentSessionToolCallArgsEvent() (*AgentSessionToolCallArgsEventResolver, bool) {
	e, ok := r.event.(*agent.ToolCallArgsEvent)
	if !ok {
		return nil, false
	}
	return &AgentSessionToolCallArgsEventResolver{event: e}, true
}

// ToAgentSessionToolCallEndEvent resolves the AgentSessionToolCallEndEvent union member
func (r *AgentSessionEventResolver) ToAgentSessionToolCallEndEvent() (*AgentSessionToolCallEndEventResolver, bool) {
	e, ok := r.event.(*agent.ToolCallEndEvent)
	if !ok {
		return nil, false
	}
	return &AgentSessionToolCallEndEventResolver{event: e}, true
}

// ToAgentSessionToolCallResultEvent resolves the AgentSessionToolCallResultEvent union member
func (r *AgentSessionEventResolver) ToAgentSessionToolCallResultEvent() (*AgentSessionToolCallResultEventResolver, bool) {
	e, ok := r.event.(*agent.ToolCallResultEvent)
	if !ok {
		return nil, false
	}
	return &AgentSessionToolCallResultEventResolver{event: e}, true
}

// ToAgentSessionReasoningStartEvent resolves the AgentSessionReasoningStartEvent union member
func (r *AgentSessionEventResolver) ToAgentSessionReasoningStartEvent() (*AgentSessionReasoningStartEventResolver, bool) {
	e, ok := r.event.(*agent.ReasoningStartEvent)
	if !ok {
		return nil, false
	}
	return &AgentSessionReasoningStartEventResolver{event: e}, true
}

// ToAgentSessionReasoningMessageStartEvent resolves the AgentSessionReasoningMessageStartEvent union member
func (r *AgentSessionEventResolver) ToAgentSessionReasoningMessageStartEvent() (*AgentSessionReasoningMessageStartEventResolver, bool) {
	e, ok := r.event.(*agent.ReasoningMessageStartEvent)
	if !ok {
		return nil, false
	}
	return &AgentSessionReasoningMessageStartEventResolver{event: e}, true
}

// ToAgentSessionReasoningMessageContentEvent resolves the AgentSessionReasoningMessageContentEvent union member
func (r *AgentSessionEventResolver) ToAgentSessionReasoningMessageContentEvent() (*AgentSessionReasoningMessageContentEventResolver, bool) {
	e, ok := r.event.(*agent.ReasoningMessageContentEvent)
	if !ok {
		return nil, false
	}
	return &AgentSessionReasoningMessageContentEventResolver{event: e}, true
}

// ToAgentSessionReasoningMessageEndEvent resolves the AgentSessionReasoningMessageEndEvent union member
func (r *AgentSessionEventResolver) ToAgentSessionReasoningMessageEndEvent() (*AgentSessionReasoningMessageEndEventResolver, bool) {
	e, ok := r.event.(*agent.ReasoningMessageEndEvent)
	if !ok {
		return nil, false
	}
	return &AgentSessionReasoningMessageEndEventResolver{event: e}, true
}

// ToAgentSessionReasoningEndEvent resolves the AgentSessionReasoningEndEvent union member
func (r *AgentSessionEventResolver) ToAgentSessionReasoningEndEvent() (*AgentSessionReasoningEndEventResolver, bool) {
	e, ok := r.event.(*agent.ReasoningEndEvent)
	if !ok {
		return nil, false
	}
	return &AgentSessionReasoningEndEventResolver{event: e}, true
}

/* Concrete event resolvers */

func timestampPtr(t *int64) *int32 {
	if t == nil {
		return nil
	}
	v := int32(*t)
	return &v
}

// AgentSessionRunStartedEventResolver resolves an AgentSessionRunStartedEvent
type AgentSessionRunStartedEventResolver struct{ event *agent.RunStartedEvent }

// Type resolver
func (r *AgentSessionRunStartedEventResolver) Type() string { return string(r.event.BaseEvent.Type) }

// ThreadID resolver
func (r *AgentSessionRunStartedEventResolver) ThreadID() string {
	return gid.ToGlobalID(types.AgentSessionModelType, r.event.ThreadID)
}

// RunID resolver
func (r *AgentSessionRunStartedEventResolver) RunID() string {
	return gid.ToGlobalID(types.AgentSessionRunModelType, r.event.RunID)
}

// Timestamp resolver
func (r *AgentSessionRunStartedEventResolver) Timestamp() *int32 {
	return timestampPtr(r.event.Timestamp)
}

// AgentSessionRunFinishedEventResolver resolves an AgentSessionRunFinishedEvent
type AgentSessionRunFinishedEventResolver struct{ event *agent.RunFinishedEvent }

// Type resolver
func (r *AgentSessionRunFinishedEventResolver) Type() string { return string(r.event.BaseEvent.Type) }

// ThreadID resolver
func (r *AgentSessionRunFinishedEventResolver) ThreadID() string {
	return gid.ToGlobalID(types.AgentSessionModelType, r.event.ThreadID)
}

// RunID resolver
func (r *AgentSessionRunFinishedEventResolver) RunID() string {
	return gid.ToGlobalID(types.AgentSessionRunModelType, r.event.RunID)
}

// Timestamp resolver
func (r *AgentSessionRunFinishedEventResolver) Timestamp() *int32 {
	return timestampPtr(r.event.Timestamp)
}

// AgentSessionRunErrorEventResolver resolves an AgentSessionRunErrorEvent
type AgentSessionRunErrorEventResolver struct{ event *agent.RunErrorEvent }

// Type resolver
func (r *AgentSessionRunErrorEventResolver) Type() string { return string(r.event.BaseEvent.Type) }

// ThreadID resolver
func (r *AgentSessionRunErrorEventResolver) ThreadID() string {
	return gid.ToGlobalID(types.AgentSessionModelType, r.event.ThreadID)
}

// RunID resolver
func (r *AgentSessionRunErrorEventResolver) RunID() string {
	return gid.ToGlobalID(types.AgentSessionRunModelType, r.event.RunID)
}

// Message resolver
func (r *AgentSessionRunErrorEventResolver) Message() string { return r.event.Message }

// Timestamp resolver
func (r *AgentSessionRunErrorEventResolver) Timestamp() *int32 {
	return timestampPtr(r.event.Timestamp)
}

// AgentSessionCustomEventResolver resolves an AgentSessionCustomEvent
type AgentSessionCustomEventResolver struct{ event *agent.CustomEvent }

// Type resolver
func (r *AgentSessionCustomEventResolver) Type() string { return string(r.event.BaseEvent.Type) }

// Name resolver
func (r *AgentSessionCustomEventResolver) Name() string { return r.event.Name }

// Value resolver
func (r *AgentSessionCustomEventResolver) Value() *string {
	if r.event.Value == nil {
		return nil
	}
	b, err := json.Marshal(r.event.Value)
	if err != nil {
		return nil
	}
	s := string(b)
	return &s
}

// Timestamp resolver
func (r *AgentSessionCustomEventResolver) Timestamp() *int32 { return timestampPtr(r.event.Timestamp) }

// AgentSessionStepStartedEventResolver resolves an AgentSessionStepStartedEvent
type AgentSessionStepStartedEventResolver struct{ event *agent.StepStartedEvent }

// Type resolver
func (r *AgentSessionStepStartedEventResolver) Type() string { return string(r.event.BaseEvent.Type) }

// StepName resolver
func (r *AgentSessionStepStartedEventResolver) StepName() string { return r.event.StepName }

// Timestamp resolver
func (r *AgentSessionStepStartedEventResolver) Timestamp() *int32 {
	return timestampPtr(r.event.Timestamp)
}

// AgentSessionStepFinishedEventResolver resolves an AgentSessionStepFinishedEvent
type AgentSessionStepFinishedEventResolver struct{ event *agent.StepFinishedEvent }

// Type resolver
func (r *AgentSessionStepFinishedEventResolver) Type() string { return string(r.event.BaseEvent.Type) }

// StepName resolver
func (r *AgentSessionStepFinishedEventResolver) StepName() string { return r.event.StepName }

// Timestamp resolver
func (r *AgentSessionStepFinishedEventResolver) Timestamp() *int32 {
	return timestampPtr(r.event.Timestamp)
}

// AgentSessionTextMessageStartEventResolver resolves an AgentSessionTextMessageStartEvent
type AgentSessionTextMessageStartEventResolver struct{ event *agent.TextMessageStartEvent }

// Type resolver
func (r *AgentSessionTextMessageStartEventResolver) Type() string {
	return string(r.event.BaseEvent.Type)
}

// MessageID resolver
func (r *AgentSessionTextMessageStartEventResolver) MessageID() string {
	return gid.ToGlobalID(types.AgentSessionMessageModelType, r.event.MessageID)
}

// Role resolver
func (r *AgentSessionTextMessageStartEventResolver) Role() string { return r.event.Role }

// Timestamp resolver
func (r *AgentSessionTextMessageStartEventResolver) Timestamp() *int32 {
	return timestampPtr(r.event.Timestamp)
}

// AgentSessionTextMessageContentEventResolver resolves an AgentSessionTextMessageContentEvent
type AgentSessionTextMessageContentEventResolver struct {
	event *agent.TextMessageContentEvent
}

// Type resolver
func (r *AgentSessionTextMessageContentEventResolver) Type() string {
	return string(r.event.BaseEvent.Type)
}

// MessageID resolver
func (r *AgentSessionTextMessageContentEventResolver) MessageID() string {
	return gid.ToGlobalID(types.AgentSessionMessageModelType, r.event.MessageID)
}

// Delta resolver
func (r *AgentSessionTextMessageContentEventResolver) Delta() string { return r.event.Delta }

// Timestamp resolver
func (r *AgentSessionTextMessageContentEventResolver) Timestamp() *int32 {
	return timestampPtr(r.event.Timestamp)
}

// AgentSessionTextMessageEndEventResolver resolves an AgentSessionTextMessageEndEvent
type AgentSessionTextMessageEndEventResolver struct{ event *agent.TextMessageEndEvent }

// Type resolver
func (r *AgentSessionTextMessageEndEventResolver) Type() string {
	return string(r.event.BaseEvent.Type)
}

// MessageID resolver
func (r *AgentSessionTextMessageEndEventResolver) MessageID() string {
	return gid.ToGlobalID(types.AgentSessionMessageModelType, r.event.MessageID)
}

// Timestamp resolver
func (r *AgentSessionTextMessageEndEventResolver) Timestamp() *int32 {
	return timestampPtr(r.event.Timestamp)
}

// AgentSessionToolCallStartEventResolver resolves an AgentSessionToolCallStartEvent
type AgentSessionToolCallStartEventResolver struct{ event *agent.ToolCallStartEvent }

// Type resolver
func (r *AgentSessionToolCallStartEventResolver) Type() string { return string(r.event.BaseEvent.Type) }

// ToolCallID resolver
func (r *AgentSessionToolCallStartEventResolver) ToolCallID() string { return r.event.ToolCallID }

// ToolCallName resolver
func (r *AgentSessionToolCallStartEventResolver) ToolCallName() string { return r.event.ToolCallName }

// ParentMessageID resolver
func (r *AgentSessionToolCallStartEventResolver) ParentMessageID() *string {
	if r.event.ParentMessageID == "" {
		return nil
	}
	id := gid.ToGlobalID(types.AgentSessionMessageModelType, r.event.ParentMessageID)
	return &id
}

// Timestamp resolver
func (r *AgentSessionToolCallStartEventResolver) Timestamp() *int32 {
	return timestampPtr(r.event.Timestamp)
}

// AgentSessionToolCallArgsEventResolver resolves an AgentSessionToolCallArgsEvent
type AgentSessionToolCallArgsEventResolver struct{ event *agent.ToolCallArgsEvent }

// Type resolver
func (r *AgentSessionToolCallArgsEventResolver) Type() string { return string(r.event.BaseEvent.Type) }

// ToolCallID resolver
func (r *AgentSessionToolCallArgsEventResolver) ToolCallID() string { return r.event.ToolCallID }

// Delta resolver
func (r *AgentSessionToolCallArgsEventResolver) Delta() string { return r.event.Delta }

// Timestamp resolver
func (r *AgentSessionToolCallArgsEventResolver) Timestamp() *int32 {
	return timestampPtr(r.event.Timestamp)
}

// AgentSessionToolCallEndEventResolver resolves an AgentSessionToolCallEndEvent
type AgentSessionToolCallEndEventResolver struct{ event *agent.ToolCallEndEvent }

// Type resolver
func (r *AgentSessionToolCallEndEventResolver) Type() string { return string(r.event.BaseEvent.Type) }

// ToolCallID resolver
func (r *AgentSessionToolCallEndEventResolver) ToolCallID() string { return r.event.ToolCallID }

// Timestamp resolver
func (r *AgentSessionToolCallEndEventResolver) Timestamp() *int32 {
	return timestampPtr(r.event.Timestamp)
}

// AgentSessionToolCallResultEventResolver resolves an AgentSessionToolCallResultEvent
type AgentSessionToolCallResultEventResolver struct{ event *agent.ToolCallResultEvent }

// Type resolver
func (r *AgentSessionToolCallResultEventResolver) Type() string {
	return string(r.event.BaseEvent.Type)
}

// MessageID resolver
func (r *AgentSessionToolCallResultEventResolver) MessageID() string {
	return gid.ToGlobalID(types.AgentSessionMessageModelType, r.event.MessageID)
}

// ToolCallID resolver
func (r *AgentSessionToolCallResultEventResolver) ToolCallID() string { return r.event.ToolCallID }

// Content resolver
func (r *AgentSessionToolCallResultEventResolver) Content() string { return r.event.Content }

// Role resolver
func (r *AgentSessionToolCallResultEventResolver) Role() string { return r.event.Role }

// Timestamp resolver
func (r *AgentSessionToolCallResultEventResolver) Timestamp() *int32 {
	return timestampPtr(r.event.Timestamp)
}

// AgentSessionReasoningStartEventResolver resolves an AgentSessionReasoningStartEvent
type AgentSessionReasoningStartEventResolver struct{ event *agent.ReasoningStartEvent }

// Type resolver
func (r *AgentSessionReasoningStartEventResolver) Type() string {
	return string(r.event.BaseEvent.Type)
}

// MessageID resolver
func (r *AgentSessionReasoningStartEventResolver) MessageID() string {
	return gid.ToGlobalID(types.AgentSessionMessageModelType, r.event.MessageID)
}

// Timestamp resolver
func (r *AgentSessionReasoningStartEventResolver) Timestamp() *int32 {
	return timestampPtr(r.event.Timestamp)
}

// AgentSessionReasoningMessageStartEventResolver resolves an AgentSessionReasoningMessageStartEvent
type AgentSessionReasoningMessageStartEventResolver struct {
	event *agent.ReasoningMessageStartEvent
}

// Type resolver
func (r *AgentSessionReasoningMessageStartEventResolver) Type() string {
	return string(r.event.BaseEvent.Type)
}

// MessageID resolver
func (r *AgentSessionReasoningMessageStartEventResolver) MessageID() string {
	return gid.ToGlobalID(types.AgentSessionMessageModelType, r.event.MessageID)
}

// Role resolver
func (r *AgentSessionReasoningMessageStartEventResolver) Role() string { return r.event.Role }

// Timestamp resolver
func (r *AgentSessionReasoningMessageStartEventResolver) Timestamp() *int32 {
	return timestampPtr(r.event.Timestamp)
}

// AgentSessionReasoningMessageContentEventResolver resolves an AgentSessionReasoningMessageContentEvent
type AgentSessionReasoningMessageContentEventResolver struct {
	event *agent.ReasoningMessageContentEvent
}

// Type resolver
func (r *AgentSessionReasoningMessageContentEventResolver) Type() string {
	return string(r.event.BaseEvent.Type)
}

// MessageID resolver
func (r *AgentSessionReasoningMessageContentEventResolver) MessageID() string {
	return gid.ToGlobalID(types.AgentSessionMessageModelType, r.event.MessageID)
}

// Delta resolver
func (r *AgentSessionReasoningMessageContentEventResolver) Delta() string { return r.event.Delta }

// Timestamp resolver
func (r *AgentSessionReasoningMessageContentEventResolver) Timestamp() *int32 {
	return timestampPtr(r.event.Timestamp)
}

// AgentSessionReasoningMessageEndEventResolver resolves an AgentSessionReasoningMessageEndEvent
type AgentSessionReasoningMessageEndEventResolver struct {
	event *agent.ReasoningMessageEndEvent
}

// Type resolver
func (r *AgentSessionReasoningMessageEndEventResolver) Type() string {
	return string(r.event.BaseEvent.Type)
}

// MessageID resolver
func (r *AgentSessionReasoningMessageEndEventResolver) MessageID() string {
	return gid.ToGlobalID(types.AgentSessionMessageModelType, r.event.MessageID)
}

// Timestamp resolver
func (r *AgentSessionReasoningMessageEndEventResolver) Timestamp() *int32 {
	return timestampPtr(r.event.Timestamp)
}

// AgentSessionReasoningEndEventResolver resolves an AgentSessionReasoningEndEvent
type AgentSessionReasoningEndEventResolver struct{ event *agent.ReasoningEndEvent }

// Type resolver
func (r *AgentSessionReasoningEndEventResolver) Type() string { return string(r.event.BaseEvent.Type) }

// MessageID resolver
func (r *AgentSessionReasoningEndEventResolver) MessageID() string {
	return gid.ToGlobalID(types.AgentSessionMessageModelType, r.event.MessageID)
}

// Timestamp resolver
func (r *AgentSessionReasoningEndEventResolver) Timestamp() *int32 {
	return timestampPtr(r.event.Timestamp)
}

/* CreateAgentSession Mutation */

// CreateAgentSessionInput is the input for creating an agent session
type CreateAgentSessionInput struct {
	ClientMutationID *string
}

// CreateAgentSessionPayload is the response payload
type CreateAgentSessionPayload struct {
	ClientMutationID *string
	Session          *models.AgentSession
	Problems         []Problem
}

// CreateAgentSessionPayloadResolver resolves the payload
type CreateAgentSessionPayloadResolver struct {
	CreateAgentSessionPayload
}

// Session field resolver
func (r *CreateAgentSessionPayloadResolver) Session() *AgentSessionResolver {
	if r.CreateAgentSessionPayload.Session == nil {
		return nil
	}
	return &AgentSessionResolver{session: r.CreateAgentSessionPayload.Session}
}

func handleCreateAgentSessionMutationProblem(e error, clientMutationID *string) (*CreateAgentSessionPayloadResolver, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}
	payload := CreateAgentSessionPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}
	return &CreateAgentSessionPayloadResolver{CreateAgentSessionPayload: payload}, nil
}

func createAgentSessionMutation(ctx context.Context, input *CreateAgentSessionInput) (*CreateAgentSessionPayloadResolver, error) {
	session, err := getServiceCatalog(ctx).AgentService.CreateAgentSession(ctx)
	if err != nil {
		return nil, err
	}
	payload := CreateAgentSessionPayload{
		ClientMutationID: input.ClientMutationID,
		Session:          session,
		Problems:         []Problem{},
	}
	return &CreateAgentSessionPayloadResolver{CreateAgentSessionPayload: payload}, nil
}

/* CreateAgentRun Mutation */

// CreateAgentRunInput is the input for creating an agent run
type CreateAgentRunInput struct {
	ClientMutationID *string
	SessionID        string
	PreviousRunID    *string
	Message          string
	Context          *[]string
}

// CreateAgentRunPayload is the response payload
type CreateAgentRunPayload struct {
	ClientMutationID *string
	Run              *models.AgentSessionRun
	Problems         []Problem
}

// CreateAgentRunPayloadResolver resolves the payload
type CreateAgentRunPayloadResolver struct {
	CreateAgentRunPayload
}

// Run field resolver
func (r *CreateAgentRunPayloadResolver) Run() *AgentSessionRunResolver {
	if r.CreateAgentRunPayload.Run == nil {
		return nil
	}
	return &AgentSessionRunResolver{run: r.CreateAgentRunPayload.Run}
}

func handleCreateAgentRunMutationProblem(e error, clientMutationID *string) (*CreateAgentRunPayloadResolver, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}
	payload := CreateAgentRunPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}
	return &CreateAgentRunPayloadResolver{CreateAgentRunPayload: payload}, nil
}

func createAgentRunMutation(ctx context.Context, input *CreateAgentRunInput) (*CreateAgentRunPayloadResolver, error) {
	serviceCatalog := getServiceCatalog(ctx)

	sessionID, err := serviceCatalog.FetchModelID(ctx, input.SessionID)
	if err != nil {
		return nil, err
	}

	svcInput := &agentsvc.CreateAgentRunInput{
		SessionID: sessionID,
		Message:   input.Message,
	}
	if input.PreviousRunID != nil {
		id, err := serviceCatalog.FetchModelID(ctx, *input.PreviousRunID)
		if err != nil {
			return nil, err
		}
		svcInput.PreviousRunID = &id
	}
	if input.Context != nil {
		svcInput.Context = *input.Context
	}

	run, err := serviceCatalog.AgentService.CreateAgentRun(ctx, svcInput)
	if err != nil {
		return nil, err
	}
	payload := CreateAgentRunPayload{
		ClientMutationID: input.ClientMutationID,
		Run:              run,
		Problems:         []Problem{},
	}
	return &CreateAgentRunPayloadResolver{CreateAgentRunPayload: payload}, nil
}

/* CancelAgentRun Mutation */

// CancelAgentRunInput is the input for cancelling an agent run
type CancelAgentRunInput struct {
	ClientMutationID *string
	RunID            string
}

// CancelAgentRunPayload is the response payload
type CancelAgentRunPayload struct {
	ClientMutationID *string
	Run              *models.AgentSessionRun
	Problems         []Problem
}

// CancelAgentRunPayloadResolver resolves the payload
type CancelAgentRunPayloadResolver struct {
	CancelAgentRunPayload
}

// Run field resolver
func (r *CancelAgentRunPayloadResolver) Run() *AgentSessionRunResolver {
	if r.CancelAgentRunPayload.Run == nil {
		return nil
	}
	return &AgentSessionRunResolver{run: r.CancelAgentRunPayload.Run}
}

func handleCancelAgentRunMutationProblem(e error, clientMutationID *string) (*CancelAgentRunPayloadResolver, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}
	payload := CancelAgentRunPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}
	return &CancelAgentRunPayloadResolver{CancelAgentRunPayload: payload}, nil
}

func cancelAgentRunMutation(ctx context.Context, input *CancelAgentRunInput) (*CancelAgentRunPayloadResolver, error) {
	serviceCatalog := getServiceCatalog(ctx)

	runID, err := serviceCatalog.FetchModelID(ctx, input.RunID)
	if err != nil {
		return nil, err
	}

	run, err := serviceCatalog.AgentService.CancelAgentRun(ctx, &agentsvc.CancelAgentRunInput{
		RunID: runID,
	})
	if err != nil {
		return nil, err
	}
	payload := CancelAgentRunPayload{
		ClientMutationID: input.ClientMutationID,
		Run:              run,
		Problems:         []Problem{},
	}
	return &CancelAgentRunPayloadResolver{CancelAgentRunPayload: payload}, nil
}

/* AgentSession Events Subscription */

// AgentSessionEventSubscriptionInput is the input for subscribing to agent session events
type AgentSessionEventSubscriptionInput struct {
	SessionID string
}

func (r RootResolver) agentSessionEventsSubscription(ctx context.Context, input *AgentSessionEventSubscriptionInput) (<-chan *AgentSessionEventResolver, error) {
	serviceCatalog := getServiceCatalog(ctx)

	sessionID, err := serviceCatalog.FetchModelID(ctx, input.SessionID)
	if err != nil {
		return nil, err
	}

	events, err := serviceCatalog.AgentService.SubscribeToAgentSession(ctx, &agentsvc.SubscribeToAgentSessionInput{
		SessionID: sessionID,
	})
	if err != nil {
		return nil, err
	}

	outgoing := make(chan *AgentSessionEventResolver)

	go func() {
		defer close(outgoing)

		for event := range events {
			select {
			case <-ctx.Done():
				return
			case outgoing <- &AgentSessionEventResolver{event: event}:
			}
		}
	}()

	return outgoing, nil
}

// AgentTraceInput represents the input for querying an agent trace
type AgentTraceInput struct {
	RunID string
}

func agentTraceQuery(ctx context.Context, input *AgentTraceInput) (*string, error) {
	serviceCatalog := getServiceCatalog(ctx)

	runID, err := serviceCatalog.FetchModelID(ctx, input.RunID)
	if err != nil {
		return nil, err
	}

	data, err := serviceCatalog.AgentService.GetAgentTrace(ctx, runID)
	if err != nil {
		return nil, err
	}

	s := string(data)
	return &s, nil
}

/* AgentSessionRun Connection */

// AgentSessionRunConnectionResolver resolves an AgentSessionRun connection
type AgentSessionRunConnectionResolver struct {
	connection Connection
}

// NewAgentSessionRunConnectionResolver creates a new AgentSessionRunConnectionResolver
func NewAgentSessionRunConnectionResolver(ctx context.Context, input *agentsvc.GetAgentSessionRunsInput) (*AgentSessionRunConnectionResolver, error) {
	result, err := getServiceCatalog(ctx).AgentService.GetAgentSessionRuns(ctx, input)
	if err != nil {
		return nil, err
	}

	runs := result.AgentSessionRuns

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
		pageInfo.EndCursor, err = result.PageInfo.Cursor(&runs[len(runs)-1])
		if err != nil {
			return nil, err
		}
	}

	return &AgentSessionRunConnectionResolver{connection: Connection{
		TotalCount: result.PageInfo.TotalCount,
		PageInfo:   pageInfo,
		Edges:      edges,
	}}, nil
}

// TotalCount returns the total result count for the connection
func (r *AgentSessionRunConnectionResolver) TotalCount() int32 {
	return r.connection.TotalCount
}

// PageInfo returns the connection page information
func (r *AgentSessionRunConnectionResolver) PageInfo() *PageInfoResolver {
	return &PageInfoResolver{pageInfo: r.connection.PageInfo}
}

// Edges returns the connection edges
func (r *AgentSessionRunConnectionResolver) Edges() *[]*AgentSessionRunEdgeResolver {
	resolvers := make([]*AgentSessionRunEdgeResolver, len(r.connection.Edges))
	for i, edge := range r.connection.Edges {
		resolvers[i] = &AgentSessionRunEdgeResolver{edge: edge}
	}
	return &resolvers
}

// AgentSessionRunEdgeResolver resolves AgentSessionRun edges
type AgentSessionRunEdgeResolver struct {
	edge Edge
}

// Cursor returns an opaque cursor
func (r *AgentSessionRunEdgeResolver) Cursor() (string, error) {
	run, ok := r.edge.Node.(models.AgentSessionRun)
	if !ok {
		return "", errors.New("Failed to convert node type")
	}
	cursor, err := r.edge.CursorFunc(&run)
	return *cursor, err
}

// Node returns an AgentSessionRun node
func (r *AgentSessionRunEdgeResolver) Node() (*AgentSessionRunResolver, error) {
	run, ok := r.edge.Node.(models.AgentSessionRun)
	if !ok {
		return nil, errors.New("Failed to convert node type")
	}
	return &AgentSessionRunResolver{run: &run}, nil
}

/* AgentSessionRun Resolver */

// AgentSessionRunResolver resolves an AgentSessionRun
type AgentSessionRunResolver struct {
	run *models.AgentSessionRun
}

// ID resolver
func (r *AgentSessionRunResolver) ID() graphql.ID {
	return graphql.ID(r.run.GetGlobalID())
}

// Metadata resolver
func (r *AgentSessionRunResolver) Metadata() *MetadataResolver {
	return &MetadataResolver{metadata: &r.run.Metadata}
}

// SessionID resolver
func (r *AgentSessionRunResolver) SessionID() string {
	return gid.ToGlobalID(types.AgentSessionModelType, r.run.SessionID)
}

// PreviousRunID resolver
func (r *AgentSessionRunResolver) PreviousRunID() *string {
	if r.run.PreviousRunID == nil {
		return nil
	}
	id := gid.ToGlobalID(types.AgentSessionRunModelType, *r.run.PreviousRunID)
	return &id
}

// LastMessageID resolver
func (r *AgentSessionRunResolver) LastMessageID() *string {
	if r.run.LastMessageID == nil {
		return nil
	}
	id := gid.ToGlobalID(types.AgentSessionMessageModelType, *r.run.LastMessageID)
	return &id
}

// Status resolver
func (r *AgentSessionRunResolver) Status() string {
	return string(r.run.Status)
}

// ErrorMessage resolver
func (r *AgentSessionRunResolver) ErrorMessage() *string {
	return r.run.ErrorMessage
}

// CancelRequested resolver
func (r *AgentSessionRunResolver) CancelRequested() bool {
	return r.run.CancelRequested
}
