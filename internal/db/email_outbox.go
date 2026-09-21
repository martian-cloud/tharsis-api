package db

//go:generate go tool mockery --name EmailOutboxItems --inpackage --case underscore

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v5"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
)

// emailOutboxClaimLeaseDuration is how long a claimed outbox item is skipped by other claimers.
const emailOutboxClaimLeaseDuration = 10 * time.Minute

// EmailOutboxItems encapsulates the logic to access email outbox rows from the database.
type EmailOutboxItems interface {
	GetOutboxItemByID(ctx context.Context, id string) (*models.EmailOutboxItem, error)
	GetOutboxItemByTRN(ctx context.Context, trnValue string) (*models.EmailOutboxItem, error)
	GetOutboxItems(ctx context.Context, input *GetEmailOutboxItemsInput) (*EmailOutboxItemsResult, error)
	CreateOutboxItem(ctx context.Context, outbox *models.EmailOutboxItem) (*models.EmailOutboxItem, error)
	UpdateOutboxItem(ctx context.Context, item *models.EmailOutboxItem) (*models.EmailOutboxItem, error)
	DeleteOutboxItems(ctx context.Context, ids []string) error
	ClaimOutboxItems(ctx context.Context, input *ClaimOutboxItemsInput) ([]*models.EmailOutboxItem, error)
}

// EmailOutboxItemSortableField represents the fields that email outbox rows can be sorted by.
type EmailOutboxItemSortableField string

// EmailOutboxItemSortableField constants.
const (
	EmailOutboxItemSortableFieldCreatedAtAsc  EmailOutboxItemSortableField = "CREATED_AT_ASC"
	EmailOutboxItemSortableFieldCreatedAtDesc EmailOutboxItemSortableField = "CREATED_AT_DESC"
)

func (sf EmailOutboxItemSortableField) getFieldDescriptor() *pagination.FieldDescriptor {
	switch sf {
	case EmailOutboxItemSortableFieldCreatedAtAsc, EmailOutboxItemSortableFieldCreatedAtDesc:
		return &pagination.FieldDescriptor{Key: "created_at", Table: "email_outbox_items", Col: "created_at"}
	default:
		return nil
	}
}

func (sf EmailOutboxItemSortableField) getSortDirection() pagination.SortDirection {
	if strings.HasSuffix(string(sf), "_DESC") {
		return pagination.DescSort
	}
	return pagination.AscSort
}

// ClaimOutboxItemsInput is the input for locking outbox items with FOR UPDATE SKIP LOCKED.
type ClaimOutboxItemsInput struct {
	// Status, when set, claims outboxes in the given status.
	Status *models.EmailOutboxItemStatus
	// Ephemeral, when set, claims only outboxes whose ephemeral flag matches.
	Ephemeral *bool
	// UpdatedBefore, when set, claims only outboxes last updated before the given time (retention cutoff).
	UpdatedBefore *time.Time
	// Limit bounds how many outboxes are returned in one call.
	Limit uint
}

// EmailOutboxItemFilter contains the supported fields for filtering EmailOutboxItems resources.
type EmailOutboxItemFilter struct {
	Ephemeral *bool
	// SubjectSearch filters outboxes whose subject contains the given substring (case-insensitive).
	SubjectSearch *string
	// OutboxItemIDs filters to the given outbox item IDs.
	OutboxItemIDs []string
}

// GetEmailOutboxItemsInput is the input for listing email outbox rows.
type GetEmailOutboxItemsInput struct {
	Sort              *EmailOutboxItemSortableField
	PaginationOptions *pagination.Options
	Filter            *EmailOutboxItemFilter
}

// EmailOutboxItemsResult contains the response data and page information.
type EmailOutboxItemsResult struct {
	PageInfo    *pagination.PageInfo
	OutboxItems []models.EmailOutboxItem
}

type emailOutboxItems struct {
	dbClient *Client
}

var emailOutboxItemsFieldList = append(metadataFieldList,
	"ephemeral",
	"email_type",
	"subject",
	"payload",
	"payload_object_store_key",
	"status",
	"send_to_all_users",
	"recipient_user_ids",
	"recipient_team_ids",
	"send_at",
	"claimed_at",
)

// NewEmailOutboxItems returns an instance of the EmailOutboxItems interface.
func NewEmailOutboxItems(dbClient *Client) EmailOutboxItems {
	return &emailOutboxItems{dbClient: dbClient}
}

func (e *emailOutboxItems) GetOutboxItemByID(ctx context.Context, id string) (*models.EmailOutboxItem, error) {
	ctx, span := tracer.Start(ctx, "db.GetOutboxItemByID")
	defer span.End()

	return e.getOutboxItem(ctx, goqu.Ex{"email_outbox_items.id": id})
}

func (e *emailOutboxItems) GetOutboxItemByTRN(ctx context.Context, trnValue string) (*models.EmailOutboxItem, error) {
	ctx, span := tracer.Start(ctx, "db.GetOutboxItemByTRN")
	defer span.End()

	parsed, err := trn.TypeEmailOutboxItem.Parse(trnValue)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse TRN", errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
	}

	return e.getOutboxItem(ctx, goqu.Ex{"email_outbox_items.id": gid.FromGlobalID(parsed.Path())})
}

func (e *emailOutboxItems) GetOutboxItems(ctx context.Context, input *GetEmailOutboxItemsInput) (*EmailOutboxItemsResult, error) {
	ctx, span := tracer.Start(ctx, "db.GetOutboxItems")
	defer span.End()

	ex := goqu.And()

	if input.Filter != nil {
		if input.Filter.Ephemeral != nil {
			ex = ex.Append(goqu.I("email_outbox_items.ephemeral").Eq(*input.Filter.Ephemeral))
		}

		if input.Filter.SubjectSearch != nil && *input.Filter.SubjectSearch != "" {
			ex = ex.Append(goqu.I("email_outbox_items.subject").ILike("%" + escapeLikePattern(*input.Filter.SubjectSearch) + "%"))
		}

		if len(input.Filter.OutboxItemIDs) > 0 {
			ex = ex.Append(goqu.I("email_outbox_items.id").In(input.Filter.OutboxItemIDs))
		}
	}

	query := dialect.From(goqu.T("email_outbox_items")).
		Select(e.getSelectFields()...).
		Where(ex)

	sortDirection := pagination.AscSort

	var sortBy *pagination.FieldDescriptor
	if input.Sort != nil {
		sortDirection = input.Sort.getSortDirection()
		sortBy = input.Sort.getFieldDescriptor()
	}

	qBuilder, err := pagination.NewPaginatedQueryBuilder(
		input.PaginationOptions,
		&pagination.FieldDescriptor{Key: "id", Table: "email_outbox_items", Col: "id"},
		pagination.WithSortByField(sortBy, sortDirection),
		pagination.WithQueryTag("emailOutboxItems.GetOutboxItems"),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build query", errors.WithSpan(span))
	}

	rows, err := qBuilder.Execute(ctx, e.dbClient.getConnection(ctx), query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}
	defer rows.Close()

	results := []models.EmailOutboxItem{}
	for rows.Next() {
		item, sErr := scanEmailOutboxItem(rows)
		if sErr != nil {
			return nil, errors.Wrap(sErr, "failed to scan row", errors.WithSpan(span))
		}
		results = append(results, *item)
	}

	if err = rows.Finalize(&results); err != nil {
		return nil, errors.Wrap(err, "failed to finalize rows", errors.WithSpan(span))
	}

	return &EmailOutboxItemsResult{
		PageInfo:    rows.GetPageInfo(),
		OutboxItems: results,
	}, nil
}

// ClaimOutboxItems claims up to input.Limit outboxes the cleanup poller can delete, locking
// them with FOR UPDATE SKIP LOCKED so the caller can delete them in the same transaction.
func (e *emailOutboxItems) ClaimOutboxItems(ctx context.Context, input *ClaimOutboxItemsInput) ([]*models.EmailOutboxItem, error) {
	ctx, span := tracer.Start(ctx, "db.ClaimOutboxItems")
	defer span.End()

	// With no filter selected there is nothing to claim; return before doing any work.
	if input.Limit == 0 || (input.Status == nil && input.Ephemeral == nil) {
		return nil, nil
	}

	var conditions []goqu.Expression

	if input.Status != nil {
		conditions = append(conditions, goqu.I("email_outbox_items.status").Eq(string(*input.Status)))
	}

	if input.Ephemeral != nil {
		conditions = append(conditions, goqu.I("email_outbox_items.ephemeral").Eq(*input.Ephemeral))
	}

	if input.UpdatedBefore != nil {
		conditions = append(conditions, goqu.I("email_outbox_items.updated_at").Lt(*input.UpdatedBefore))
	}

	// Every claim stamps claimed_at and skips rows claimed within the cooldown, so two instances never process the same item and
	// a stale claim (from a crashed instance) is reclaimable once the cooldown lapses.
	now := currentTime()
	conditions = append(conditions,
		goqu.Or(
			goqu.I("email_outbox_items.claimed_at").IsNull(),
			goqu.I("email_outbox_items.claimed_at").Lt(now.Add(-emailOutboxClaimLeaseDuration)),
		),
	)

	claimable := dialect.From(goqu.T("email_outbox_items")).
		Select(goqu.I("email_outbox_items.id")).
		Where(goqu.And(conditions...)).
		Order(goqu.I("email_outbox_items.created_at").Asc()).
		Limit(input.Limit).
		ForUpdate(goqu.SkipLocked)

	query := dialect.Update(goqu.T("email_outbox_items")).
		Prepared(true).
		With("claimable", claimable).
		Set(goqu.Record{
			"version":    goqu.L("version + 1"),
			"updated_at": now,
			"claimed_at": now,
		}).
		From(goqu.T("claimable")).
		Where(goqu.I("email_outbox_items.id").Eq(goqu.I("claimable.id"))).
		Returning(e.getSelectFields()...)

	sql, args, err := toSQLWithTag("emailOutboxItems.ClaimOutboxItems", query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	rows, err := e.dbClient.getConnection(ctx).Query(ctx, sql, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}
	defer rows.Close()

	var outboxes []*models.EmailOutboxItem
	for rows.Next() {
		outbox, sErr := scanEmailOutboxItem(rows)
		if sErr != nil {
			return nil, errors.Wrap(sErr, "failed to scan row", errors.WithSpan(span))
		}

		outboxes = append(outboxes, outbox)
	}

	if err = rows.Err(); err != nil {
		return nil, errors.Wrap(err, "failed to iterate rows", errors.WithSpan(span))
	}

	return outboxes, nil
}

// UpdateOutboxItem updates the mutable fields of the given item using optimistic locking.
func (e *emailOutboxItems) UpdateOutboxItem(ctx context.Context, item *models.EmailOutboxItem) (*models.EmailOutboxItem, error) {
	ctx, span := tracer.Start(ctx, "db.UpdateOutboxItem")
	defer span.End()

	sql, args, err := toSQLWithTag("emailOutboxItems.UpdateOutboxItem", dialect.From("email_outbox_items").
		Prepared(true).
		With("email_outbox_items",
			dialect.Update("email_outbox_items").
				Set(goqu.Record{
					"version":    goqu.L("? + ?", goqu.C("version"), 1),
					"updated_at": currentTime(),
					"status":     item.Status,
				}).Where(goqu.Ex{"id": item.Metadata.ID, "version": item.Metadata.Version}).
				Returning("*"),
		).Select(e.getSelectFields()...))
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	updated, err := scanEmailOutboxItem(e.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrOptimisticLockError
		}

		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return updated, nil
}

func (e *emailOutboxItems) CreateOutboxItem(ctx context.Context, outbox *models.EmailOutboxItem) (*models.EmailOutboxItem, error) {
	ctx, span := tracer.Start(ctx, "db.CreateOutboxItem")
	defer span.End()

	timestamp := currentTime()

	recipientUserIDs, err := json.Marshal(outbox.RecipientUserIDs)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal recipient user IDs", errors.WithSpan(span))
	}

	recipientTeamIDs, err := json.Marshal(outbox.RecipientTeamIDs)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal recipient team IDs", errors.WithSpan(span))
	}

	sql, args, err := toSQLWithTag("emailOutboxItems.CreateOutboxItem", dialect.Insert("email_outbox_items").
		Prepared(true).
		Rows(goqu.Record{
			"id":                       newResourceID(),
			"version":                  initialResourceVersion,
			"created_at":               timestamp,
			"updated_at":               timestamp,
			"ephemeral":                outbox.Ephemeral,
			"email_type":               outbox.EmailType,
			"subject":                  outbox.Subject,
			"payload":                  nullableRawJSON(outbox.Payload),
			"payload_object_store_key": outbox.PayloadObjectStoreKey,
			"status":                   outbox.Status,
			"send_to_all_users":        outbox.SendToAllUsers,
			"recipient_user_ids":       recipientUserIDs,
			"recipient_team_ids":       recipientTeamIDs,
			"send_at":                  outbox.SendAt,
		}).
		Returning(emailOutboxItemsFieldList...))
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	created, err := scanEmailOutboxItem(e.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return created, nil
}

// DeleteOutboxItems deletes the given outbox rows by ID, returning ErrOptimisticLockError if fewer rows are deleted than requested.
func (e *emailOutboxItems) DeleteOutboxItems(ctx context.Context, ids []string) error {
	ctx, span := tracer.Start(ctx, "db.DeleteOutboxItems")
	defer span.End()

	if len(ids) == 0 {
		return nil
	}

	sql, args, err := toSQLWithTag("emailOutboxItems.DeleteOutboxItems", dialect.Delete("email_outbox_items").
		Prepared(true).
		Where(goqu.C("id").In(ids)))
	if err != nil {
		return errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	result, err := e.dbClient.getConnection(ctx).Exec(ctx, sql, args...)
	if err != nil {
		return errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	if result.RowsAffected() != int64(len(ids)) {
		return ErrOptimisticLockError
	}

	return nil
}

func (e *emailOutboxItems) getOutboxItem(ctx context.Context, exp goqu.Ex) (*models.EmailOutboxItem, error) {
	ctx, span := tracer.Start(ctx, "db.getOutboxItem")
	defer span.End()

	sql, args, err := toSQLWithTag("emailOutboxItems.getOutboxItem", dialect.From(goqu.T("email_outbox_items")).
		Prepared(true).
		Select(e.getSelectFields()...).
		Where(exp))
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	outbox, err := scanEmailOutboxItem(e.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}

		if pgErr := asPgError(err); pgErr != nil && isInvalidIDViolation(pgErr) {
			return nil, ErrInvalidID
		}

		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return outbox, nil
}

func (*emailOutboxItems) getSelectFields() []interface{} {
	selectFields := []interface{}{}
	for _, field := range emailOutboxItemsFieldList {
		selectFields = append(selectFields, fmt.Sprintf("email_outbox_items.%s", field))
	}
	return selectFields
}

func scanEmailOutboxItem(row scanner) (*models.EmailOutboxItem, error) {
	outbox := &models.EmailOutboxItem{}

	fields := []interface{}{
		&outbox.Metadata.ID,
		&outbox.Metadata.CreationTimestamp,
		&outbox.Metadata.LastUpdatedTimestamp,
		&outbox.Metadata.Version,
		&outbox.Ephemeral,
		&outbox.EmailType,
		&outbox.Subject,
		&outbox.Payload,
		&outbox.PayloadObjectStoreKey,
		&outbox.Status,
		&outbox.SendToAllUsers,
		&outbox.RecipientUserIDs,
		&outbox.RecipientTeamIDs,
		&outbox.SendAt,
		&outbox.ClaimedAt,
	}

	if err := row.Scan(fields...); err != nil {
		return nil, err
	}

	outbox.Metadata.TRN = trn.TypeEmailOutboxItem.Build(outbox.GetGlobalID())

	return outbox, nil
}
