package providerregistry

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/events"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/maintenance"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

const (
	// workItemEventTimeout bounds how long the consumer waits for a work-item event before
	// falling back to polling the queue, so a missed notification can't leave an item stuck.
	workItemEventTimeout = 1 * time.Minute
	workItemClaimLimit   = 10

	// maxWorkItemClaims bounds redelivery of a failing item before it is dead-lettered, so a
	// poison item can't churn a worker slot and the logs indefinitely.
	maxWorkItemClaims = 10
)

// WorkItemConsumer consumes provider-docs work items from the queue and processes them.
type WorkItemConsumer struct {
	logger             logger.Logger
	dbClient           *db.Client
	eventManager       *events.EventManager
	maintenanceMonitor maintenance.Monitor
	processor          DocsProcessor
}

// NewWorkItemConsumer creates a provider-docs WorkItemConsumer.
func NewWorkItemConsumer(logger logger.Logger, dbClient *db.Client, eventManager *events.EventManager, maintenanceMonitor maintenance.Monitor, processor DocsProcessor) *WorkItemConsumer {
	return &WorkItemConsumer{
		logger:             logger,
		dbClient:           dbClient,
		eventManager:       eventManager,
		maintenanceMonitor: maintenanceMonitor,
		processor:          processor,
	}
}

// Start launches the consumer's background listener.
func (s *WorkItemConsumer) Start(ctx context.Context) {
	go s.listenForWorkItems(ctx)
}

// listenForWorkItems drives processing off the work-item event stream, falling back to polling the
// queue if no event arrives within workItemEventTimeout so a missed event can't strand an item.
func (s *WorkItemConsumer) listenForWorkItems(ctx context.Context) {
	subscriber := s.eventManager.Subscribe([]events.Subscription{
		{
			Type:    events.WorkItemQueueSubscription,
			Actions: []events.SubscriptionAction{events.CreateAction},
			Filter:  matchDocsWorkItemEvent,
		},
	})
	defer s.eventManager.Unsubscribe(subscriber)

	for {
		s.claimAndProcess(ctx)

		waitCtx, cancel := context.WithTimeout(ctx, workItemEventTimeout)
		_, err := subscriber.GetEvent(waitCtx)
		cancel()

		if ctx.Err() != nil {
			return
		}

		if err != nil && !errors.Is(err, context.DeadlineExceeded) {
			s.logger.Errorf("provider docs work items event subscription error: %v", err)
		}
	}
}

// matchDocsWorkItemEvent reports whether a work-item event is for this consumer's own type.
func matchDocsWorkItemEvent(data json.RawMessage) bool {
	var d struct {
		Type string `json:"type"`
	}
	return json.Unmarshal(data, &d) == nil && d.Type == string(db.ProcessTFProviderDocsType)
}

func (s *WorkItemConsumer) claimAndProcess(ctx context.Context) {
	// Claiming is a DB write; idle quietly during maintenance and resume on the next tick.
	inMaintenance, err := s.maintenanceMonitor.InMaintenanceMode(ctx)
	if err != nil {
		s.logger.Errorf("provider docs work item consumer failed to check maintenance mode: %v", err)
		return
	}
	if inMaintenance {
		return
	}

	workItems, err := s.dbClient.WorkItemsQueue.ClaimWorkItems(ctx, &db.ClaimWorkItemsInput{
		Type:          db.ProcessTFProviderDocsType,
		Limit:         workItemClaimLimit,
		MaxClaimCount: maxWorkItemClaims,
	})
	if err != nil {
		s.logger.Errorf("failed to claim provider docs work items: %v", err)
		return
	}

	// Handle each claimed item concurrently, bounded by the claim limit, and wait for the batch before claiming more.
	var wg sync.WaitGroup
	for _, item := range workItems {
		wg.Add(1)
		go func(item db.WorkItem) {
			defer wg.Done()
			s.processWorkItem(ctx, &item)
		}(item)
	}
	wg.Wait()
}

// processWorkItem handles a single claimed item and acknowledges it on success.
func (s *WorkItemConsumer) processWorkItem(ctx context.Context, item *db.WorkItem) {
	if err := s.handleWorkItem(ctx, item); err != nil {
		// Leave the item unacknowledged so the queue redelivers it after its lease; MaxClaimCount
		// bounds redelivery and dead-letters poison items.
		s.logger.Errorf("failed to process provider docs work item %s (attempt %d): %v", item.ID, item.ClaimCount, err)
		return
	}

	// A deleted provider version returns nil from the processor, so it is acked here, not retried.
	if err := s.dbClient.WorkItemsQueue.AcknowledgeWorkItem(ctx, item.ID); err != nil {
		s.logger.Errorf("failed to acknowledge provider docs work item %s: %v", item.ID, err)
	}
}

func (s *WorkItemConsumer) handleWorkItem(ctx context.Context, item *db.WorkItem) error {
	payload, ok := item.ToProcessTFProviderDocsPayload()
	if !ok {
		s.logger.Errorf("invalid payload for work item %s", item.ID)
		return nil
	}
	return s.processor.ProcessProviderVersionDocs(ctx, payload.ProviderVersionID, item.ClaimCount >= maxWorkItemClaims)
}
