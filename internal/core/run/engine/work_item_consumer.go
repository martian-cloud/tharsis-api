package engine

import (
	"context"
	"errors"
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/admission"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/commands"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/events"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/maintenance"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

const (
	// workItemEventTimeout bounds how long the work item consumer waits for a work-item
	// event before falling back to polling the queue directly, so a missed
	// notification can't leave a work item stuck.
	workItemEventTimeout = 1 * time.Minute
	workItemClaimLimit   = 1000

	// maxWorkItemClaims bounds how many times a failing work item is redelivered
	// before it is dropped (dead-lettered), so a permanently-failing ("poison")
	// item can't churn a worker slot and the logs indefinitely. It is passed to
	// ClaimWorkItems, which enforces the cap and reaps exhausted items.
	maxWorkItemClaims = 10
)

// workItemHandler processes a single claimed work item of one type.
type workItemHandler func(ctx context.Context, item *db.WorkItem) error

// WorkItemConsumer consumes work items from the run work-items queue and processes them.
type WorkItemConsumer struct {
	logger             logger.Logger
	dbClient           *db.Client
	eventManager       *events.EventManager
	processor          CmdProcessor
	factory            *commands.Factory
	maintenanceMonitor maintenance.Monitor
	handlers           map[db.WorkItemType]workItemHandler
}

// NewWorkItemConsumer creates a new run WorkItemConsumer.
func NewWorkItemConsumer(logger logger.Logger, dbClient *db.Client, eventManager *events.EventManager, processor CmdProcessor, factory *commands.Factory, maintenanceMonitor maintenance.Monitor) *WorkItemConsumer {
	s := &WorkItemConsumer{
		processor:          processor,
		dbClient:           dbClient,
		eventManager:       eventManager,
		logger:             logger,
		factory:            factory,
		maintenanceMonitor: maintenanceMonitor,
	}

	// One handler per work-item type; Start and handleWorkItem are driven by this map.
	// To add a work-item type: add its payload to db's workItemPayloadFactories and
	// register its handler here — nothing else in the work item consumer changes.
	s.handlers = map[db.WorkItemType]workItemHandler{
		db.QueuePendingRunsForWorkspaceType: func(ctx context.Context, item *db.WorkItem) error {
			payload, ok := item.ToQueuePendingRunsForWorkspacePayload()
			if !ok {
				s.logger.Errorf("invalid payload for work item %s", item.ID)
				return nil
			}
			return s.handleQueuePendingRunsForWorkspace(ctx, payload)
		},
		db.DiscardStalePlannedRunsForWorkspaceType: func(ctx context.Context, item *db.WorkItem) error {
			payload, ok := item.ToDiscardStalePlannedRunsForWorkspacePayload()
			if !ok {
				s.logger.Errorf("invalid payload for work item %s", item.ID)
				return nil
			}
			return s.handleDiscardStalePlannedRunsForWorkspace(ctx, payload)
		},
	}

	return s
}

// Start launches a work item consumer background goroutine per handled work-item type.
// ClaimWorkItems filters by a single type, so each type gets its own listener: a
// backlog of one type can't starve another, and the types are processed
// concurrently.
func (s *WorkItemConsumer) Start(ctx context.Context) {
	for workItemType := range s.handlers {
		go s.listenForWorkItems(ctx, workItemType)
	}
}

// listenForWorkItems drives processing of a single work-item type off the event
// stream, but with a timeout so it falls back to polling the queue if no event
// arrives within workItemEventTimeout. This replaces a separate polling loop:
// events make processing responsive, and the timeout is the safety net for missed
// events.
func (s *WorkItemConsumer) listenForWorkItems(ctx context.Context, workItemType db.WorkItemType) {
	subscriber := s.eventManager.Subscribe([]events.Subscription{
		{
			Type:    events.WorkItemQueueSubscription,
			Actions: []events.SubscriptionAction{events.CreateAction},
		},
	})
	defer s.eventManager.Unsubscribe(subscriber)

	for {
		// Process any available work items on startup and after each event or
		// fallback timeout.
		s.claimAndProcess(ctx, workItemType)

		// Wait for the next work-item event, falling back to a poll if none
		// arrives within the timeout.
		waitCtx, cancel := context.WithTimeout(ctx, workItemEventTimeout)
		_, err := subscriber.GetEvent(waitCtx)
		cancel()

		// Stop when the work item consumer's context is canceled (shutdown).
		if ctx.Err() != nil {
			return
		}

		// A deadline-exceeded error just means no event arrived in time, which is
		// the expected fallback-poll trigger; log anything else (the loop polls
		// again on the next iteration regardless).
		if err != nil && !errors.Is(err, context.DeadlineExceeded) {
			s.logger.Errorf("work items event subscription error: %v", err)
		}
	}
}

func (s *WorkItemConsumer) claimAndProcess(ctx context.Context, workItemType db.WorkItemType) {
	// Claiming a work item is a DB write. Skip it while in maintenance mode so the
	// consumer idles quietly instead of repeatedly failing to claim; the listener loop
	// re-checks on its next event/timeout tick and resumes when maintenance ends.
	inMaintenance, err := s.maintenanceMonitor.InMaintenanceMode(ctx)
	if err != nil {
		s.logger.Errorf("work item consumer failed to check maintenance mode: %v", err)
		return
	}
	if inMaintenance {
		return
	}

	workItems, err := s.dbClient.WorkItemsQueue.ClaimWorkItems(ctx, &db.ClaimWorkItemsInput{
		Type:          workItemType,
		Limit:         workItemClaimLimit,
		MaxClaimCount: maxWorkItemClaims,
	})
	if err != nil {
		s.logger.Errorf("failed to claim work items: %v", err)
		return
	}

	for _, item := range workItems {
		if err := s.handleWorkItem(ctx, &item); err != nil {
			// Leave the item unacknowledged so it is redelivered after its lease. The
			// queue bounds redelivery via MaxClaimCount and drops poison items, so the
			// consumer doesn't need to track claim counts itself.
			s.logger.Errorf("failed to process work item %s (attempt %d): %v", item.ID, item.ClaimCount, err)
			continue
		}

		if err := s.dbClient.WorkItemsQueue.AcknowledgeWorkItem(ctx, item.ID); err != nil {
			s.logger.Errorf("failed to acknowledge work item %s: %v", item.ID, err)
		}
	}
}

func (s *WorkItemConsumer) handleWorkItem(ctx context.Context, item *db.WorkItem) error {
	handler, ok := s.handlers[item.Type]
	if !ok {
		s.logger.Errorf("unknown work item type: %s", item.Type)
		return nil
	}
	return handler(ctx, item)
}

func (s *WorkItemConsumer) handleQueuePendingRunsForWorkspace(ctx context.Context, payload *db.QueuePendingRunsForWorkspacePayload) error {
	// Only queuing and queuing_apply runs are actionable here: a queuing run's plan
	// is pending admission (speculative runs start immediately, non-speculative when
	// the workspace is free) and a queuing_apply run holds an approved apply parked
	// waiting for the workspace. Runs in any other status are no-ops for this
	// handler, so filter them out DB-side rather than fetching the workspace's
	// entire run history.
	sort := db.RunSortableFieldCreatedAtAsc
	result, err := s.dbClient.Runs.GetRuns(ctx, &db.GetRunsInput{
		Sort: &sort,
		Filter: &db.RunFilter{
			WorkspaceID: &payload.WorkspaceID,
			Statuses:    []models.RunStatus{models.RunQueuing, models.RunQueuingApply},
		},
	})
	if err != nil {
		return err
	}

	ws, err := s.dbClient.Workspaces.GetWorkspaceByID(ctx, payload.WorkspaceID)
	if err != nil {
		return err
	}
	if ws == nil {
		return nil
	}

	// Only one non-speculative run proceeds per workspace at a time; the admitter
	// re-validates per run inside the queue command, so this is purely a pre-filter
	// to avoid issuing commands that cannot be admitted.
	if !admission.WorkspaceAvailable(ws) {
		return nil
	}

	for _, run := range result.Runs {
		if run.Speculative() {
			continue
		}

		if run.Status == models.RunQueuing || run.Status == models.RunQueuingApply {
			if err := s.processor.ProcessCommand(ctx, s.factory.NewQueueRun(run.Metadata.ID)); err != nil {
				s.logger.Errorf("failed to resume apply for run %s: %v", run.Metadata.ID, err)
				return err
			}
			// Return here since we only want to queue one run per work item to ensure runs are queued in order
			return nil
		}
	}

	return nil
}

// handleDiscardStalePlannedRunsForWorkspace discards the workspace's runs that are still
// parked at planned and entered that state before the apply that triggered this work item
// completed. Those plans were computed against pre-apply state and are stale; a run that
// entered planned after the apply is newer than the applied state and must not be discarded.
func (s *WorkItemConsumer) handleDiscardStalePlannedRunsForWorkspace(ctx context.Context, payload *db.DiscardStalePlannedRunsForWorkspacePayload) error {
	// The payload carries the apply-completion time as the cutoff, so only planned runs last
	// updated before it (i.e. that entered planned before the apply finished) are discarded.
	updatedBefore := payload.ApplyCompletedAt
	result, err := s.dbClient.Runs.GetRuns(ctx, &db.GetRunsInput{
		Filter: &db.RunFilter{
			WorkspaceID:   &payload.WorkspaceID,
			Statuses:      []models.RunStatus{models.RunPlanned},
			UpdatedBefore: &updatedBefore,
		},
	})
	if err != nil {
		return err
	}

	for _, run := range result.Runs {
		// These system-initiated discards aren't attributed to a user, so they record no
		// activity event (and need no caller on the context). Return any error so the work
		// item is redelivered rather than acked; discard is idempotent — an already-discarded
		// run is no longer planned, so the status filter excludes it on the retry.
		if err := s.processor.ProcessCommand(ctx, s.factory.NewDiscardRun(&commands.DiscardRunInput{
			RunID:             run.Metadata.ID,
			SkipActivityEvent: true,
		})); err != nil {
			s.logger.Errorf("failed to discard stale planned run %s: %v", run.Metadata.ID, err)
			return err
		}
	}

	return nil
}
