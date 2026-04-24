// Package agent provides the agent service for managing AI agent sessions and runs
package agent

import (
	"context"
	"encoding/json"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/m-mizutani/gollem"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/agent"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/asynctask"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/events"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/limits"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

const (
	// MaxUserMessageSize is the maximum allowed size in bytes for a user message including context.
	MaxUserMessageSize = 2048
	// maxAgentSessionsPerUser is the max number of agent sessions per user
	maxAgentSessionsPerUser = 100
)

// CreateAgentSessionInput contains the input for creating an agent session
type CreateAgentSessionInput struct{}

// CreateAgentRunInput contains the input for creating an agent run
type CreateAgentRunInput struct {
	SessionID     string
	PreviousRunID *string
	Message       string
	Context       []string
}

// SubscribeToAgentSessionInput contains the input for subscribing to agent session events
type SubscribeToAgentSessionInput struct {
	SessionID string
}

// CancelAgentRunInput contains the input for cancelling an agent run
type CancelAgentRunInput struct {
	RunID string
}

// GetAgentSessionRunsInput contains the input for listing agent session runs
type GetAgentSessionRunsInput struct {
	SessionID         string
	Sort              *db.AgentSessionRunSortableField
	PaginationOptions *pagination.Options
}

// Service encapsulates the business logic for the agent
type Service interface {
	GetAgentSessionByID(ctx context.Context, id string) (*models.AgentSession, error)
	GetAgentSessionByTRN(ctx context.Context, trn string) (*models.AgentSession, error)
	GetAgentSessionRunByID(ctx context.Context, id string) (*models.AgentSessionRun, error)
	GetAgentSessionRunByTRN(ctx context.Context, trn string) (*models.AgentSessionRun, error)
	CreateAgentSession(ctx context.Context) (*models.AgentSession, error)
	CreateAgentRun(ctx context.Context, input *CreateAgentRunInput) (*models.AgentSessionRun, error)
	CancelAgentRun(ctx context.Context, input *CancelAgentRunInput) (*models.AgentSessionRun, error)
	GetAgentSessionRuns(ctx context.Context, input *GetAgentSessionRunsInput) (*db.AgentSessionRunsResult, error)
	SubscribeToAgentSession(ctx context.Context, input *SubscribeToAgentSessionInput) (<-chan agent.Event, error)
	GetAgentTrace(ctx context.Context, runID string) (json.RawMessage, error)
	GetAgentCreditUsage(ctx context.Context, userID string) (float64, error)
}

// ToolSetFactory creates a gollem ToolSet for an agent run.
type ToolSetFactory func(ctx context.Context) (gollem.ToolSet, error)

type service struct {
	logger         logger.Logger
	dbClient       *db.Client
	systemAgent    *agent.SystemAgent
	agentStore     agent.Store
	eventManager   *events.EventManager
	toolSetFactory ToolSetFactory
	taskManager    asynctask.Manager
	limitChecker   limits.LimitChecker
	aiEnabled      bool
}

// NewService creates an instance of the agent Service
func NewService(logger logger.Logger, dbClient *db.Client, systemAgent *agent.SystemAgent, agentStore agent.Store, eventManager *events.EventManager, toolSetFactory ToolSetFactory, taskManager asynctask.Manager, limitChecker limits.LimitChecker, aiEnabled bool) Service {
	return &service{
		logger:         logger,
		dbClient:       dbClient,
		systemAgent:    systemAgent,
		agentStore:     agentStore,
		eventManager:   eventManager,
		toolSetFactory: toolSetFactory,
		taskManager:    taskManager,
		limitChecker:   limitChecker,
		aiEnabled:      aiEnabled,
	}
}

var errAINotEnabled = errors.New("AI features are not enabled", errors.WithErrorCode(errors.EServiceUnavailable))

func (s *service) requireAI() error {
	if !s.aiEnabled {
		return errAINotEnabled
	}
	return nil
}

func (s *service) authorizeUserCaller(ctx context.Context) (*auth.UserCaller, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	userCaller, ok := caller.(*auth.UserCaller)
	if !ok {
		return nil, errors.New("only user callers are supported", errors.WithErrorCode(errors.EForbidden))
	}
	return userCaller, nil
}

func (s *service) getSessionAssociatedWithUser(ctx context.Context, sessionID string, userID string) (*models.AgentSession, error) {
	session, err := s.dbClient.AgentSessions.GetAgentSessionByID(ctx, sessionID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load agent session")
	}

	if session == nil {
		return nil, errors.New("session not found", errors.WithErrorCode(errors.ENotFound))
	}

	if session.UserID != userID {
		return nil, errors.New("session not found", errors.WithErrorCode(errors.ENotFound))
	}

	return session, nil
}

func (s *service) GetAgentSessionByID(ctx context.Context, id string) (*models.AgentSession, error) {
	if err := s.requireAI(); err != nil {
		return nil, err
	}

	userCaller, err := s.authorizeUserCaller(ctx)
	if err != nil {
		return nil, err
	}
	return s.getSessionAssociatedWithUser(ctx, id, userCaller.User.Metadata.ID)
}

func (s *service) GetAgentSessionByTRN(ctx context.Context, trn string) (*models.AgentSession, error) {
	if err := s.requireAI(); err != nil {
		return nil, err
	}

	userCaller, err := s.authorizeUserCaller(ctx)
	if err != nil {
		return nil, err
	}
	session, err := s.dbClient.AgentSessions.GetAgentSessionByTRN(ctx, trn)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, errors.New("agent session not found", errors.WithErrorCode(errors.ENotFound))
	}

	if session.UserID != userCaller.User.Metadata.ID {
		return nil, errors.New("session not found", errors.WithErrorCode(errors.ENotFound))
	}

	return session, nil
}

func (s *service) GetAgentSessionRunByID(ctx context.Context, id string) (*models.AgentSessionRun, error) {
	if err := s.requireAI(); err != nil {
		return nil, err
	}

	userCaller, err := s.authorizeUserCaller(ctx)
	if err != nil {
		return nil, err
	}
	run, err := s.dbClient.AgentSessionRuns.GetAgentSessionRunByID(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get agent run")
	}
	if run == nil {
		return nil, errors.New("run not found", errors.WithErrorCode(errors.ENotFound))
	}

	if _, err := s.getSessionAssociatedWithUser(ctx, run.SessionID, userCaller.User.Metadata.ID); err != nil {
		return nil, err
	}

	return run, nil
}

func (s *service) GetAgentSessionRunByTRN(ctx context.Context, trn string) (*models.AgentSessionRun, error) {
	if err := s.requireAI(); err != nil {
		return nil, err
	}

	userCaller, err := s.authorizeUserCaller(ctx)
	if err != nil {
		return nil, err
	}
	run, err := s.dbClient.AgentSessionRuns.GetAgentSessionRunByTRN(ctx, trn)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, errors.New("run not found", errors.WithErrorCode(errors.ENotFound))
	}

	if _, err := s.getSessionAssociatedWithUser(ctx, run.SessionID, userCaller.User.Metadata.ID); err != nil {
		return nil, err
	}

	return run, nil
}

func (s *service) CreateAgentSession(ctx context.Context) (*models.AgentSession, error) {
	if err := s.requireAI(); err != nil {
		return nil, err
	}

	userCaller, err := s.authorizeUserCaller(ctx)
	if err != nil {
		return nil, err
	}

	userID := userCaller.User.Metadata.ID

	txCtx, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin transaction")
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txCtx); txErr != nil {
			s.logger.Errorf("failed to rollback create agent session tx: %v", txErr)
		}
	}()

	// Prune old sessions
	if err = s.dbClient.AgentSessions.DeleteOldestSessionsByUserID(txCtx, userID, maxAgentSessionsPerUser); err != nil {
		return nil, errors.Wrap(err, "failed to prune old agent sessions")
	}

	session, err := s.dbClient.AgentSessions.CreateAgentSession(txCtx, &models.AgentSession{
		UserID: userID,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create agent session")
	}

	if err := s.dbClient.Transactions.CommitTx(txCtx); err != nil {
		return nil, errors.Wrap(err, "failed to commit transaction")
	}

	return session, nil
}

func (s *service) CreateAgentRun(ctx context.Context, input *CreateAgentRunInput) (*models.AgentSessionRun, error) {
	if err := s.requireAI(); err != nil {
		return nil, err
	}

	userCaller, err := s.authorizeUserCaller(ctx)
	if err != nil {
		return nil, err
	}

	session, err := s.getSessionAssociatedWithUser(ctx, input.SessionID, userCaller.User.Metadata.ID)
	if err != nil {
		return nil, err
	}

	if input.Message == "" {
		return nil, errors.New("message is required", errors.WithErrorCode(errors.EInvalid))
	}

	// Calculate total message size including context.
	totalSize := len(input.Message)
	for _, c := range input.Context {
		totalSize += len(c)
	}
	if totalSize > MaxUserMessageSize {
		return nil, errors.New(
			"user message with context exceeds maximum size of %d bytes",
			MaxUserMessageSize,
			errors.WithErrorCode(errors.EInvalid),
		)
	}

	// Check the runs-per-session resource limit.
	existingRuns, err := s.dbClient.AgentSessionRuns.GetAgentSessionRuns(ctx, &db.GetAgentSessionRunsInput{
		Filter:            &db.AgentSessionRunFilter{SessionID: &input.SessionID},
		PaginationOptions: &pagination.Options{First: ptr.Int32(0)},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to count session runs")
	}
	if err = s.limitChecker.CheckLimit(ctx,
		limits.ResourceLimitAgentSessionRunsPerSession, existingRuns.PageInfo.TotalCount); err != nil {
		return nil, err
	}

	var prevRun *models.AgentSessionRun
	if input.PreviousRunID != nil {
		prevRun, err = s.dbClient.AgentSessionRuns.GetAgentSessionRunByID(ctx, *input.PreviousRunID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get previous run")
		}
		if prevRun == nil || prevRun.SessionID != session.Metadata.ID {
			return nil, errors.New("previous run not found", errors.WithErrorCode(errors.ENotFound))
		}

		if prevRun.Status == models.AgentSessionRunRunning {
			return nil, errors.New("previous run is not finished", errors.WithErrorCode(errors.EConflict))
		}

		result, err := s.dbClient.AgentSessionRuns.GetAgentSessionRuns(ctx, &db.GetAgentSessionRunsInput{
			Filter:            &db.AgentSessionRunFilter{PreviousRunID: input.PreviousRunID},
			PaginationOptions: &pagination.Options{First: ptr.Int32(0)},
		})
		if err != nil {
			return nil, errors.Wrap(err, "failed to validate previous run chain")
		}
		if result.PageInfo.TotalCount > 0 {
			return nil, errors.New("previous run is already referenced by another run", errors.WithErrorCode(errors.EInvalid))
		}
	}

	run, err := s.dbClient.AgentSessionRuns.CreateAgentSessionRun(ctx, &models.AgentSessionRun{
		SessionID:     session.Metadata.ID,
		PreviousRunID: input.PreviousRunID,
		Status:        models.AgentSessionRunRunning,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create agent run")
	}

	s.taskManager.StartTask(func(ctx context.Context) {
		// Add caller to context
		ctx = auth.WithCaller(ctx, userCaller)

		toolSet, err := s.toolSetFactory(ctx)
		if err != nil {
			s.logger.Errorf("failed to create toolset: %v", err)
			return
		}

		s.systemAgent.Run(ctx, &agent.RunInput{
			Session:         session,
			Run:             run,
			PreviousRun:     prevRun,
			ToolSets:        []gollem.ToolSet{toolSet},
			Task:            input.Message,
			ContextMessages: input.Context,
			Timeout:         s.taskManager.Timeout() - 20*time.Second, // Add buffer to ensure agent has time to handle timeout
		})
	})

	return run, nil
}

func (s *service) CancelAgentRun(ctx context.Context, input *CancelAgentRunInput) (*models.AgentSessionRun, error) {
	if err := s.requireAI(); err != nil {
		return nil, err
	}

	userCaller, err := s.authorizeUserCaller(ctx)
	if err != nil {
		return nil, err
	}

	run, err := s.dbClient.AgentSessionRuns.GetAgentSessionRunByID(ctx, input.RunID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get agent run")
	}
	if run == nil {
		return nil, errors.New("run not found", errors.WithErrorCode(errors.ENotFound))
	}

	// Verify the session is owned by the caller
	if _, err := s.getSessionAssociatedWithUser(ctx, run.SessionID, userCaller.User.Metadata.ID); err != nil {
		return nil, err
	}

	if run.Status != models.AgentSessionRunRunning {
		return nil, errors.New("run is not in a running state", errors.WithErrorCode(errors.EInvalid))
	}

	run.CancelRequested = true
	updated, err := s.dbClient.AgentSessionRuns.UpdateAgentSessionRun(ctx, run)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update agent run")
	}

	return updated, nil
}

func (s *service) SubscribeToAgentSession(ctx context.Context, input *SubscribeToAgentSessionInput) (<-chan agent.Event, error) {
	if err := s.requireAI(); err != nil {
		return nil, err
	}

	userCaller, err := s.authorizeUserCaller(ctx)
	if err != nil {
		return nil, err
	}

	session, err := s.getSessionAssociatedWithUser(ctx, input.SessionID, userCaller.User.Metadata.ID)
	if err != nil {
		return nil, err
	}

	sessionID := session.Metadata.ID

	// Subscribe BEFORE querying existing state to avoid missing events created between query and subscribe.
	subscriber := s.eventManager.Subscribe([]events.Subscription{
		{
			Type:    events.AgentSessionRunSubscription,
			Actions: []events.SubscriptionAction{events.CreateAction, events.UpdateAction},
			Filter: func(data json.RawMessage) bool {
				var d struct {
					SessionID string `json:"session_id"`
				}
				return json.Unmarshal(data, &d) == nil && d.SessionID == sessionID
			},
		},
	})

	// Query existing runs for replay.
	sortAsc := db.AgentSessionRunSortableFieldCreatedAtAsc
	existingRunsResult, err := s.dbClient.AgentSessionRuns.GetAgentSessionRuns(ctx, &db.GetAgentSessionRunsInput{
		Sort:   &sortAsc,
		Filter: &db.AgentSessionRunFilter{SessionID: &sessionID},
	})
	if err != nil {
		s.eventManager.Unsubscribe(subscriber)
		return nil, errors.Wrap(err, "failed to query existing runs for replay")
	}

	existingRuns := existingRunsResult.AgentSessionRuns

	outgoing := make(chan agent.Event, 16)

	go func() {
		defer close(outgoing)
		defer s.eventManager.Unsubscribe(subscriber)

		var lastSentMessageID *string
		replayedRunIDs := make(map[string]struct{}, len(existingRuns))
		replayedMessageIDs := make(map[string]struct{})

		// Replay existing runs.
		for i := range existingRuns {
			run := &existingRuns[i]
			replayedRunIDs[run.Metadata.ID] = struct{}{}

			replayEvents, msgIDs, err := s.replayRun(ctx, sessionID, run)
			if err != nil {
				s.logger.Errorf("failed to replay run: %v", err)
				return
			}
			for _, ev := range replayEvents {
				select {
				case <-ctx.Done():
					return
				case outgoing <- ev:
				}
			}

			for _, id := range msgIDs {
				replayedMessageIDs[id] = struct{}{}
			}

			if run.LastMessageID != nil {
				lastSentMessageID = run.LastMessageID
			}
		}

		// Live event loop.
		for {
			event, err := subscriber.GetEvent(ctx)
			if err != nil {
				if !errors.IsContextCanceledError(err) {
					s.logger.Errorf("error waiting for agent session events: %v", err)
				}
				return
			}

			runData, err := event.ToAgentSessionRunEventData()
			if err != nil {
				continue
			}

			// Deduplicate: skip CreateAction for runs already replayed.
			if event.Action == string(events.CreateAction) {
				if _, replayed := replayedRunIDs[runData.ID]; replayed {
					continue
				}
			}

			var aguiEvents []agent.Event

			// Emit message events first (before lifecycle) to avoid dropping the final message.
			// Skip messages that were already sent during replay.
			if runData.LastMessageID != nil && (lastSentMessageID == nil || *runData.LastMessageID != *lastSentMessageID) {
				if _, replayed := replayedMessageIDs[*runData.LastMessageID]; !replayed {
					lastSentMessageID = runData.LastMessageID
					msgEvents, err := s.handleMessageEvent(ctx, sessionID, *runData.LastMessageID, false)
					if err != nil {
						s.logger.Errorf("failed to handle message event: %v", err)
						return
					}
					aguiEvents = append(aguiEvents, msgEvents...)
				} else {
					lastSentMessageID = runData.LastMessageID
				}
			}

			// Then emit lifecycle events.
			aguiEvents = append(aguiEvents, s.handleRunEvent(event.Action, runData)...)

			for _, ev := range aguiEvents {
				select {
				case <-ctx.Done():
					return
				case outgoing <- ev:
				}
			}
		}
	}()

	return outgoing, nil
}

// replayRun emits the full event sequence for an existing run: RunStarted, all messages, then terminal event.
// Returns the events and the IDs of all messages that were replayed.
func (s *service) replayRun(ctx context.Context, sessionID string, run *models.AgentSessionRun) ([]agent.Event, []string, error) {
	threadID := sessionID
	runID := run.Metadata.ID

	var evts []agent.Event

	// RunStarted
	evts = append(evts, &agent.RunStartedEvent{
		BaseEvent: agent.BaseEvent{Type: agent.EventTypeRunStarted},
		ThreadID:  threadID,
		RunID:     runID,
	})

	// Replay all messages for this run.
	sortAsc := db.AgentSessionMessageSortableFieldCreatedAtAsc
	result, err := s.dbClient.AgentSessionMessages.GetAgentSessionMessages(ctx, &db.GetAgentSessionMessagesInput{
		Sort:   &sortAsc,
		Filter: &db.AgentSessionMessageFilter{RunID: &runID},
	})
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to query messages for run replay")
	}

	toolContentLoader := func(msgID string) (json.RawMessage, error) {
		return s.agentStore.GetToolContent(ctx, sessionID, msgID)
	}

	var messageIDs []string
	for i := range result.AgentSessionMessages {
		msg := &result.AgentSessionMessages[i]
		messageIDs = append(messageIDs, msg.Metadata.ID)
		msgEvents, err := agent.EventsFromMessage(msg, toolContentLoader, false)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to convert message to events for run replay")
		}
		evts = append(evts, msgEvents...)
	}

	// Terminal lifecycle event (if run is no longer running).
	switch run.Status {
	case models.AgentSessionRunFinished:
		evts = append(evts, &agent.RunFinishedEvent{
			BaseEvent: agent.BaseEvent{Type: agent.EventTypeRunFinished},
			ThreadID:  threadID,
			RunID:     runID,
		})
	case models.AgentSessionRunErrored:
		msg := ""
		if run.ErrorMessage != nil {
			msg = *run.ErrorMessage
		}
		evts = append(evts, &agent.RunErrorEvent{
			BaseEvent: agent.BaseEvent{Type: agent.EventTypeRunError},
			ThreadID:  threadID,
			RunID:     runID,
			Message:   msg,
		})
	case models.AgentSessionRunCancelled:
		evts = append(evts, &agent.CustomEvent{
			BaseEvent: agent.BaseEvent{Type: agent.EventTypeCustom},
			Name:      string(agent.EventTypeRunCancelled),
			Value: map[string]string{
				"threadId": threadID,
				"runId":    runID,
			},
		})
	}

	return evts, messageIDs, nil
}

func (s *service) handleRunEvent(action string, data *db.AgentSessionRunEventData) []agent.Event {
	threadID := data.SessionID
	runID := data.ID

	if action == string(events.CreateAction) {
		return []agent.Event{&agent.RunStartedEvent{
			BaseEvent: agent.BaseEvent{Type: agent.EventTypeRunStarted},
			ThreadID:  threadID,
			RunID:     runID,
		}}
	}

	// UPDATE action — check terminal statuses
	switch data.Status {
	case string(models.AgentSessionRunFinished):
		return []agent.Event{&agent.RunFinishedEvent{
			BaseEvent: agent.BaseEvent{Type: agent.EventTypeRunFinished},
			ThreadID:  threadID,
			RunID:     runID,
		}}
	case string(models.AgentSessionRunErrored):
		msg := ""
		if data.ErrorMessage != nil {
			msg = *data.ErrorMessage
		}
		return []agent.Event{&agent.RunErrorEvent{
			BaseEvent: agent.BaseEvent{Type: agent.EventTypeRunError},
			ThreadID:  threadID,
			RunID:     runID,
			Message:   msg,
		}}
	case string(models.AgentSessionRunCancelled):
		return []agent.Event{&agent.CustomEvent{
			BaseEvent: agent.BaseEvent{Type: agent.EventTypeCustom},
			Name:      string(agent.EventTypeRunCancelled),
			Value: map[string]string{
				"threadId": threadID,
				"runId":    runID,
			},
		}}
	}

	return nil
}

func (s *service) handleMessageEvent(ctx context.Context, sessionID, messageID string, includeReasoning bool) ([]agent.Event, error) {
	msg, err := s.dbClient.AgentSessionMessages.GetAgentSessionMessageByID(ctx, sessionID, messageID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get agent session message")
	}
	if msg == nil {
		return nil, nil
	}

	evts, err := agent.EventsFromMessage(msg, func(msgID string) (json.RawMessage, error) {
		return s.agentStore.GetToolContent(ctx, sessionID, msgID)
	}, includeReasoning)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert message to events")
	}

	return evts, nil
}

func (s *service) GetAgentSessionRuns(ctx context.Context, input *GetAgentSessionRunsInput) (*db.AgentSessionRunsResult, error) {
	if err := s.requireAI(); err != nil {
		return nil, err
	}

	caller, err := s.authorizeUserCaller(ctx)
	if err != nil {
		return nil, err
	}

	if _, err := s.getSessionAssociatedWithUser(ctx, input.SessionID, caller.User.Metadata.ID); err != nil {
		return nil, err
	}

	return s.dbClient.AgentSessionRuns.GetAgentSessionRuns(ctx, &db.GetAgentSessionRunsInput{
		Sort:              input.Sort,
		PaginationOptions: input.PaginationOptions,
		Filter:            &db.AgentSessionRunFilter{SessionID: &input.SessionID},
	})
}

func (s *service) GetAgentTrace(ctx context.Context, runID string) (json.RawMessage, error) {
	if err := s.requireAI(); err != nil {
		return nil, err
	}

	userCaller, err := s.authorizeUserCaller(ctx)
	if err != nil {
		return nil, err
	}

	if !userCaller.IsAdmin() {
		return nil, errors.New("only admins can view trace data", errors.WithErrorCode(errors.EForbidden))
	}

	s.logger.Infow("accessing agent session run trace", "caller", userCaller.GetSubject(), "runId", runID)

	run, err := s.dbClient.AgentSessionRuns.GetAgentSessionRunByID(ctx, runID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get agent run")
	}
	if run == nil {
		return nil, errors.New("run not found", errors.WithErrorCode(errors.ENotFound))
	}

	data, err := s.agentStore.GetTrace(ctx, run.SessionID, run.Metadata.ID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get agent trace")
	}

	return json.RawMessage(data), nil
}

func (s *service) GetAgentCreditUsage(ctx context.Context, userID string) (float64, error) {
	if err := s.requireAI(); err != nil {
		return 0, err
	}

	userCaller, err := s.authorizeUserCaller(ctx)
	if err != nil {
		return 0, err
	}

	if !userCaller.IsAdmin() && userCaller.User.Metadata.ID != userID {
		return 0, errors.New("cannot view credit usage for other users", errors.WithErrorCode(errors.EForbidden))
	}

	monthDate := time.Date(time.Now().UTC().Year(), time.Now().UTC().Month(), 1, 0, 0, 0, 0, time.UTC)

	quota, err := s.dbClient.AgentCreditQuotas.GetAgentCreditQuota(ctx, userID, monthDate)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get credit quota")
	}
	if quota == nil {
		return 0, nil
	}

	return quota.TotalCredits, nil
}
