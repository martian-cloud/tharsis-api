package db

//go:generate go tool mockery --name WorkItemsQueue --inpackage --case underscore

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/doug-martin/goqu/v9"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

const (
	// workItemClaimLeaseDuration is the time a work item is unavailable after being claimed.
	workItemClaimLeaseDuration = 5 * time.Minute
)

// WorkItemType represents the work item type.
type WorkItemType string

// WorkItemType constants.
const (
	QueuePendingRunsForWorkspaceType        WorkItemType = "QUEUE_PENDING_RUNS_FOR_WORKSPACE"
	DiscardStalePlannedRunsForWorkspaceType WorkItemType = "DISCARD_STALE_PLANNED_RUNS_FOR_WORKSPACE"
)

// workItemPayloadFactories maps each work-item type to a constructor for its payload,
// used when scanning rows to unmarshal the payload column into the right type. A new
// work-item type must be registered here (and given a handler in the run work item consumer)
// or scanning rows of that type fails.
var workItemPayloadFactories = map[WorkItemType]func() any{
	QueuePendingRunsForWorkspaceType:        func() any { return &QueuePendingRunsForWorkspacePayload{} },
	DiscardStalePlannedRunsForWorkspaceType: func() any { return &DiscardStalePlannedRunsForWorkspacePayload{} },
}

// QueuePendingRunsForWorkspacePayload represents the payload for a QueuePendingRunsForWorkspace work item.
type QueuePendingRunsForWorkspacePayload struct {
	WorkspaceID string `json:"workspaceId"`
}

// DiscardStalePlannedRunsForWorkspacePayload represents the payload for a
// DiscardStalePlannedRunsForWorkspace work item. ApplyCompletedAt is the time the
// apply that made the planned runs stale completed; every planned run in the
// workspace last updated (i.e. that entered the planned state) before it had its
// plan computed against pre-apply state and is discarded, while a run that entered
// planned after the apply is left untouched.
type DiscardStalePlannedRunsForWorkspacePayload struct {
	WorkspaceID      string    `json:"workspaceId"`
	ApplyCompletedAt time.Time `json:"applyCompletedAt"`
}

// ClaimWorkItemsInput is the input for claiming work items.
type ClaimWorkItemsInput struct {
	Limit uint
	Type  WorkItemType
	// MaxClaimCount is the maximum number of times a work item may be claimed before it
	// is dropped as undeliverable (dead-lettered), so a permanently-failing ("poison")
	// item can't be redelivered forever. 0 means unlimited (never dropped).
	MaxClaimCount uint
}

// AddWorkItemToQueueInput is the input for adding a work item to the queue.
type AddWorkItemToQueueInput struct {
	Type        WorkItemType
	Payload     any
	AvailableAt *time.Time
}

// WorkItem represents a unit of work in the queue.
type WorkItem struct {
	ID                 string
	CreationTimestamp  *time.Time
	AvailableTimestamp *time.Time
	ClaimCount         int
	Type               WorkItemType
	Payload            any
}

// ToQueuePendingRunsForWorkspacePayload returns the payload as a QueuePendingRunsForWorkspacePayload.
func (w *WorkItem) ToQueuePendingRunsForWorkspacePayload() (*QueuePendingRunsForWorkspacePayload, bool) {
	payload, ok := w.Payload.(*QueuePendingRunsForWorkspacePayload)
	return payload, ok
}

// ToDiscardStalePlannedRunsForWorkspacePayload returns the payload as a DiscardStalePlannedRunsForWorkspacePayload.
func (w *WorkItem) ToDiscardStalePlannedRunsForWorkspacePayload() (*DiscardStalePlannedRunsForWorkspacePayload, bool) {
	payload, ok := w.Payload.(*DiscardStalePlannedRunsForWorkspacePayload)
	return payload, ok
}

// WorkItemsQueue provides an interface for the persistent work items queue.
type WorkItemsQueue interface {
	ClaimWorkItems(ctx context.Context, input *ClaimWorkItemsInput) ([]WorkItem, error)
	AcknowledgeWorkItem(ctx context.Context, workItemID string) error
	AddWorkItemToQueue(ctx context.Context, item *AddWorkItemToQueueInput) (*WorkItem, error)
}

type workItemsQueue struct {
	dbClient *Client
}

var workItemsQueueFieldList = []any{"id", "created_at", "available_at", "claim_count", "type", "payload"}

// NewWorkItemsQueue returns a new WorkItemsQueue instance.
func NewWorkItemsQueue(dbClient *Client) WorkItemsQueue {
	return &workItemsQueue{dbClient: dbClient}
}

// ClaimWorkItems fetches a batch of work items and marks them as unavailable.
func (w *workItemsQueue) ClaimWorkItems(ctx context.Context, input *ClaimWorkItemsInput) ([]WorkItem, error) {
	ctx, span := tracer.Start(ctx, "db.ClaimWorkItems")
	defer span.End()

	now := time.Now().UTC()

	// Drop ("dead-letter") items that have already been claimed the maximum number of
	// times so a permanently-failing item can't churn forever. The claimable filter
	// below also excludes them, so reaping is housekeeping; it is idempotent and safe
	// under concurrent instances.
	if input.MaxClaimCount > 0 {
		if err := w.reapExhaustedWorkItems(ctx, input.Type, input.MaxClaimCount); err != nil {
			return nil, errors.Wrap(err, "failed to reap exhausted work items", errors.WithSpan(span))
		}
	}

	claimablePredicate := goqu.And(
		goqu.I("work_items_queue.type").Eq(input.Type),
		goqu.I("work_items_queue.available_at").Lte(now),
	)
	if input.MaxClaimCount > 0 {
		claimablePredicate = claimablePredicate.Append(
			goqu.I("work_items_queue.claim_count").Lt(input.MaxClaimCount),
		)
	}

	claimableWorkItems := dialect.From(goqu.T("work_items_queue")).
		Select(goqu.I("work_items_queue.id")).
		Where(claimablePredicate).
		// Order by available_at (then created_at to break ties deterministically) so the
		// claim matches the (type, available_at, created_at) index prefix exactly.
		Order(
			goqu.I("work_items_queue.available_at").Asc(),
			goqu.I("work_items_queue.created_at").Asc(),
		).
		Limit(input.Limit).
		ForUpdate(goqu.SkipLocked)

	sql, args, err := dialect.Update(goqu.T("work_items_queue")).
		Prepared(true).
		With("claimable_work_items", claimableWorkItems).
		Set(goqu.Record{
			"available_at": now.Add(workItemClaimLeaseDuration),
			"claim_count":  goqu.L("claim_count + 1"),
		}).
		From(goqu.T("claimable_work_items")).
		Where(goqu.I("work_items_queue.id").Eq(goqu.I("claimable_work_items.id"))).
		Returning(w.getSelectFields()...).
		ToSQL()
	if err != nil {
		return nil, errors.Wrap(err, "failed to build query", errors.WithSpan(span))
	}

	rows, err := w.dbClient.getConnection(ctx).Query(ctx, sql, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}
	defer rows.Close()

	var workItems []WorkItem
	for rows.Next() {
		item, err := w.scanWorkItem(rows)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan row", errors.WithSpan(span))
		}
		workItems = append(workItems, *item)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "failed to iterate rows", errors.WithSpan(span))
	}

	return workItems, nil
}

// reapExhaustedWorkItems deletes work items of the given type that have reached the
// maximum claim count, logging the dropped IDs so the dead-lettering is observable.
func (w *workItemsQueue) reapExhaustedWorkItems(ctx context.Context, workItemType WorkItemType, maxClaimCount uint) error {
	sql, args, err := dialect.Delete("work_items_queue").
		Prepared(true).
		Where(goqu.And(
			goqu.C("type").Eq(workItemType),
			goqu.C("claim_count").Gte(maxClaimCount),
		)).
		Returning("id").
		ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to build query")
	}

	rows, err := w.dbClient.getConnection(ctx).Query(ctx, sql, args...)
	if err != nil {
		return errors.Wrap(err, "failed to execute query")
	}
	defer rows.Close()

	var droppedIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return errors.Wrap(err, "failed to scan row")
		}
		droppedIDs = append(droppedIDs, id)
	}
	if err := rows.Err(); err != nil {
		return errors.Wrap(err, "failed to iterate rows")
	}

	if len(droppedIDs) > 0 {
		w.dbClient.logger.WithContextFields(ctx).Errorf(
			"dropped %d undeliverable work item(s) of type %s after reaching the max claim count of %d: %v",
			len(droppedIDs), workItemType, maxClaimCount, droppedIDs,
		)
	}

	return nil
}

// AcknowledgeWorkItem removes a completed work item from the queue.
func (w *workItemsQueue) AcknowledgeWorkItem(ctx context.Context, workItemID string) error {
	ctx, span := tracer.Start(ctx, "db.AcknowledgeWorkItem")
	defer span.End()

	sql, args, err := dialect.Delete("work_items_queue").
		Prepared(true).
		Where(goqu.C("id").Eq(workItemID)).
		ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	result, err := w.dbClient.getConnection(ctx).Exec(ctx, sql, args...)
	if err != nil {
		return errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	if result.RowsAffected() == 0 {
		return ErrOptimisticLockError
	}

	return nil
}

// AddWorkItemToQueue inserts a new work item into the queue.
func (w *workItemsQueue) AddWorkItemToQueue(ctx context.Context, item *AddWorkItemToQueueInput) (*WorkItem, error) {
	ctx, span := tracer.Start(ctx, "db.AddWorkItemToQueue")
	defer span.End()

	timestamp := currentTime()

	availableAt := timestamp
	if item.AvailableAt != nil {
		availableAt = *item.AvailableAt
	}

	payloadBytes, err := json.Marshal(item.Payload)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal payload", errors.WithSpan(span))
	}

	sql, args, err := dialect.Insert("work_items_queue").
		Prepared(true).
		Rows(goqu.Record{
			"id":           newResourceID(),
			"created_at":   timestamp,
			"available_at": availableAt,
			"type":         item.Type,
			"payload":      payloadBytes,
		}).Returning(w.getSelectFields()...).
		ToSQL()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	workItem, err := w.scanWorkItem(w.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return workItem, nil
}

func (*workItemsQueue) getSelectFields() []any {
	selectFields := make([]any, len(workItemsQueueFieldList))
	for i, field := range workItemsQueueFieldList {
		selectFields[i] = fmt.Sprintf("work_items_queue.%s", field)
	}
	return selectFields
}

func (*workItemsQueue) scanWorkItem(row scanner) (*WorkItem, error) {
	workItem := &WorkItem{}

	var payloadBytes []byte

	fields := []any{
		&workItem.ID,
		&workItem.CreationTimestamp,
		&workItem.AvailableTimestamp,
		&workItem.ClaimCount,
		&workItem.Type,
		&payloadBytes,
	}

	if err := row.Scan(fields...); err != nil {
		return nil, err
	}

	newPayload, ok := workItemPayloadFactories[workItem.Type]
	if !ok {
		return nil, fmt.Errorf("unknown work item type: %s", workItem.Type)
	}
	payload := newPayload()

	if err := json.Unmarshal(payloadBytes, payload); err != nil {
		return nil, err
	}

	workItem.Payload = payload

	return workItem, nil
}
