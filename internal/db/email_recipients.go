package db

//go:generate go tool mockery --name EmailRecipients --inpackage --case underscore

import (
	"context"
	"fmt"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/jackc/pgx/v5"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
)

// emailRecipientInsertChunkSize bounds the number of rows per multi-row INSERT so a
// large fan-out (e.g. a 20,000-recipient broadcast) is split into manageable chunks.
const emailRecipientInsertChunkSize = 1000

// EmailRecipients encapsulates the logic to access email recipient rows from the database.
type EmailRecipients interface {
	GetRecipientByID(ctx context.Context, id string) (*models.EmailRecipient, error)
	GetRecipientByTRN(ctx context.Context, trnValue string) (*models.EmailRecipient, error)
	GetRecipients(ctx context.Context, input *GetEmailRecipientsInput) (*EmailRecipientsResult, error)
	GetRecipientStats(ctx context.Context, emailOutboxItemID string) (*EmailRecipientStatsResult, error)
	CreateRecipients(ctx context.Context, recipients []*models.EmailRecipient) error
	UpdateRecipient(ctx context.Context, recipient *models.EmailRecipient) (*models.EmailRecipient, error)
	ClaimRecipients(ctx context.Context, input *ClaimRecipientsInput) ([]*models.EmailRecipient, error)
	AllRecipientsFinal(ctx context.Context, emailOutboxItemID string) (bool, error)
}

// ClaimRecipientsInput is the input for claiming recipients the sender should (re)attempt to send.
type ClaimRecipientsInput struct {
	// Statuses are the delivery statuses eligible to be claimed.
	Statuses []models.EmailDeliveryStatus
	// Limit is the maximum number of recipients to claim.
	Limit uint
}

// EmailRecipientSortableField represents the fields that email recipients can be sorted by.
type EmailRecipientSortableField string

// EmailRecipientSortableField constants.
const (
	EmailRecipientSortableFieldCreatedAtAsc  EmailRecipientSortableField = "CREATED_AT_ASC"
	EmailRecipientSortableFieldCreatedAtDesc EmailRecipientSortableField = "CREATED_AT_DESC"
)

func (sf EmailRecipientSortableField) getFieldDescriptor() *pagination.FieldDescriptor {
	switch sf {
	case EmailRecipientSortableFieldCreatedAtAsc, EmailRecipientSortableFieldCreatedAtDesc:
		return &pagination.FieldDescriptor{Key: "created_at", Table: "email_recipients", Col: "created_at"}
	default:
		return nil
	}
}

func (sf EmailRecipientSortableField) getSortDirection() pagination.SortDirection {
	if strings.HasSuffix(string(sf), "_DESC") {
		return pagination.DescSort
	}
	return pagination.AscSort
}

// EmailRecipientFilter contains the supported fields for filtering EmailRecipient resources.
type EmailRecipientFilter struct {
	// EmailOutboxItemID filters to the recipient's outbox item id.
	EmailOutboxItemID *string
	// DeliveryStatuses filters by delivery statuses.
	DeliveryStatuses []models.EmailDeliveryStatus
	// Search filters recipients whose address contains the given substring (case-insensitive).
	Search *string
	// HasOpened filters to recipients that have (true) or have not (false) opened the email.
	HasOpened *bool
	// HasIssues filters to recipients in the "issues" bucket: bounced, abandoned, failed, or complained.
	HasIssues *bool
}

// EmailRecipientStatsResult is the delivery breakdown for one outbox item's recipients.
type EmailRecipientStatsResult struct {
	Total     int
	Delivered int
	Opened    int
	Clicked   int
	Issues    int
}

// GetEmailRecipientsInput is the input for listing email recipients.
type GetEmailRecipientsInput struct {
	Sort              *EmailRecipientSortableField
	PaginationOptions *pagination.Options
	Filter            *EmailRecipientFilter
}

// EmailRecipientsResult contains the response data and page information.
type EmailRecipientsResult struct {
	PageInfo   *pagination.PageInfo
	Recipients []models.EmailRecipient
}

type emailRecipients struct {
	dbClient *Client
}

var emailRecipientsFieldList = append(metadataFieldList,
	"email_outbox_item_id",
	"address",
	"delivery_status",
	"attempt_count",
	"available_at",
	"last_attempt_at",
	"opened_at",
	"clicked_at",
	"complained_at",
	"failure_reason",
)

// NewEmailRecipients returns an instance of the EmailRecipients interface.
func NewEmailRecipients(dbClient *Client) EmailRecipients {
	return &emailRecipients{dbClient: dbClient}
}

func (e *emailRecipients) GetRecipientByID(ctx context.Context, id string) (*models.EmailRecipient, error) {
	ctx, span := tracer.Start(ctx, "db.GetRecipientByID")
	defer span.End()

	return e.getRecipient(ctx, goqu.Ex{"email_recipients.id": id})
}

func (e *emailRecipients) GetRecipientByTRN(ctx context.Context, trnValue string) (*models.EmailRecipient, error) {
	ctx, span := tracer.Start(ctx, "db.GetRecipientByTRN")
	defer span.End()

	parsed, err := trn.TypeEmailRecipient.Parse(trnValue)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse TRN", errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
	}

	return e.getRecipient(ctx, goqu.Ex{"email_recipients.id": gid.FromGlobalID(parsed.Path())})
}

func (e *emailRecipients) GetRecipients(ctx context.Context, input *GetEmailRecipientsInput) (*EmailRecipientsResult, error) {
	ctx, span := tracer.Start(ctx, "db.GetRecipients")
	defer span.End()

	ex := goqu.And()

	if input.Filter != nil {
		if input.Filter.EmailOutboxItemID != nil {
			ex = ex.Append(goqu.I("email_recipients.email_outbox_item_id").Eq(*input.Filter.EmailOutboxItemID))
		}

		if len(input.Filter.DeliveryStatuses) > 0 {
			ex = ex.Append(goqu.I("email_recipients.delivery_status").In(input.Filter.DeliveryStatuses))
		}

		if input.Filter.Search != nil && *input.Filter.Search != "" {
			ex = ex.Append(goqu.I("email_recipients.address").ILike("%" + escapeLikePattern(*input.Filter.Search) + "%"))
		}

		if input.Filter.HasOpened != nil {
			if *input.Filter.HasOpened {
				ex = ex.Append(goqu.I("email_recipients.opened_at").IsNotNull())
			} else {
				ex = ex.Append(goqu.I("email_recipients.opened_at").IsNull())
			}
		}

		if input.Filter.HasIssues != nil && *input.Filter.HasIssues {
			ex = ex.Append(goqu.Or(
				goqu.I("email_recipients.delivery_status").In(models.EmailDeliveryStatusesMatching(models.EmailDeliveryStatus.IsIssueStatus)),
				goqu.I("email_recipients.complained_at").IsNotNull(),
			))
		}
	}

	query := dialect.From(goqu.T("email_recipients")).
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
		&pagination.FieldDescriptor{Key: "id", Table: "email_recipients", Col: "id"},
		pagination.WithSortByField(sortBy, sortDirection),
		pagination.WithQueryTag("emailRecipients.GetRecipients"),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build query", errors.WithSpan(span))
	}

	rows, err := qBuilder.Execute(ctx, e.dbClient.getConnection(ctx), query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}
	defer rows.Close()

	results := []models.EmailRecipient{}
	for rows.Next() {
		item, sErr := scanEmailRecipient(rows)
		if sErr != nil {
			return nil, errors.Wrap(sErr, "failed to scan row", errors.WithSpan(span))
		}
		results = append(results, *item)
	}

	if err = rows.Finalize(&results); err != nil {
		return nil, errors.Wrap(err, "failed to finalize rows", errors.WithSpan(span))
	}

	return &EmailRecipientsResult{
		PageInfo:   rows.GetPageInfo(),
		Recipients: results,
	}, nil
}

// GetRecipientStats returns the delivery breakdown for one outbox item's recipients in a single aggregate query.
func (e *emailRecipients) GetRecipientStats(ctx context.Context, emailOutboxItemID string) (*EmailRecipientStatsResult, error) {
	ctx, span := tracer.Start(ctx, "db.GetRecipientStats")
	defer span.End()

	countWhen := func(cond exp.Expression) exp.SQLFunctionExpression {
		// The THEN value is an inline literal (goqu.L("1")), not a bound Go int — mixed with the
		// adjacent text-column comparisons as a parameter, pgx's type inference for the prepared
		// statement fails with "unable to encode 1 into text format".
		return goqu.COUNT(goqu.Case().When(cond, goqu.L("1")))
	}

	sql, args, err := toSQLWithTag("emailRecipients.GetRecipientStats", dialect.From(goqu.T("email_recipients")).
		Prepared(true).
		Select(
			goqu.COUNT("*"),
			countWhen(goqu.I("delivery_status").Eq(models.EmailDeliveryCompleted)),
			countWhen(goqu.I("opened_at").IsNotNull()),
			countWhen(goqu.I("clicked_at").IsNotNull()),
			countWhen(goqu.Or(
				goqu.I("delivery_status").In(models.EmailDeliveryStatusesMatching(models.EmailDeliveryStatus.IsIssueStatus)),
				goqu.I("complained_at").IsNotNull(),
			)),
		).
		Where(goqu.I("email_outbox_item_id").Eq(emailOutboxItemID)))
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	result := &EmailRecipientStatsResult{}
	if err := e.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...).Scan(
		&result.Total,
		&result.Delivered,
		&result.Opened,
		&result.Clicked,
		&result.Issues,
	); err != nil {
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return result, nil
}

// CreateRecipients inserts recipients in chunked multi-row INSERTs. It uses the connection on the
// context, so callers that need the inserts to commit atomically with the owning email_outbox_items row
// (the enqueue path) should run this inside a transaction.
func (e *emailRecipients) CreateRecipients(ctx context.Context, recipients []*models.EmailRecipient) error {
	ctx, span := tracer.Start(ctx, "db.CreateRecipients")
	defer span.End()

	if len(recipients) == 0 {
		return nil
	}

	timestamp := currentTime()

	for start := 0; start < len(recipients); start += emailRecipientInsertChunkSize {
		end := min(start+emailRecipientInsertChunkSize, len(recipients))

		records := make([]goqu.Record, 0, end-start)
		for _, r := range recipients[start:end] {
			records = append(records, goqu.Record{
				"id":                   newResourceID(),
				"version":              initialResourceVersion,
				"created_at":           timestamp,
				"updated_at":           timestamp,
				"email_outbox_item_id": r.EmailOutboxItemID,
				"address":              r.Address,
				"delivery_status":      r.DeliveryStatus,
				"attempt_count":        r.AttemptCount,
				"available_at":         r.AvailableAt,
			})
		}

		sql, args, err := toSQLWithTag("emailRecipients.CreateRecipients", dialect.Insert("email_recipients").
			Prepared(true).
			Rows(records))
		if err != nil {
			return errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
		}

		if _, err = e.dbClient.getConnection(ctx).Exec(ctx, sql, args...); err != nil {
			if pgErr := asPgError(err); pgErr != nil && isForeignKeyViolation(pgErr) {
				return errors.New("email outbox does not exist",
					errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
			}
			return errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
		}
	}

	return nil
}

// UpdateRecipient updates the mutable status/feedback fields of a recipient, enforcing optimistic
// locking on the version the caller loaded. Callers (sender, feedback processor) must read the
// current row before updating, since the claim advances the version. Immutable fields
// (email_outbox_item_id, email, created_at) are never updated.
func (e *emailRecipients) UpdateRecipient(ctx context.Context, recipient *models.EmailRecipient) (*models.EmailRecipient, error) {
	ctx, span := tracer.Start(ctx, "db.UpdateRecipient")
	defer span.End()

	sql, args, err := toSQLWithTag("emailRecipients.UpdateRecipient", dialect.Update("email_recipients").
		Prepared(true).
		Set(goqu.Record{
			"version":         goqu.L("version + 1"),
			"updated_at":      currentTime(),
			"delivery_status": recipient.DeliveryStatus,
			"attempt_count":   recipient.AttemptCount,
			"available_at":    recipient.AvailableAt,
			"opened_at":       recipient.OpenedAt,
			"clicked_at":      recipient.ClickedAt,
			"complained_at":   recipient.ComplainedAt,
			"failure_reason":  recipient.FailureReason,
		}).
		Where(goqu.Ex{"id": recipient.Metadata.ID, "version": recipient.Metadata.Version}).
		Returning(e.getReturningFields()...))
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	updated, err := scanEmailRecipient(e.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrOptimisticLockError
		}

		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return updated, nil
}

// ClaimRecipients atomically claims a batch of due recipients in one of input.Statuses, FIFO, via FOR UPDATE SKIP LOCKED.
func (e *emailRecipients) ClaimRecipients(ctx context.Context, input *ClaimRecipientsInput) ([]*models.EmailRecipient, error) {
	ctx, span := tracer.Start(ctx, "db.ClaimRecipients")
	defer span.End()

	if input.Limit == 0 || len(input.Statuses) == 0 {
		return nil, nil
	}

	now := currentTime()

	claimablePredicate := goqu.And(
		goqu.I("email_recipients.delivery_status").In(input.Statuses),
		goqu.I("email_recipients.available_at").Lte(now),
	)

	claimable := dialect.From(goqu.T("email_recipients")).
		Select(goqu.I("email_recipients.id")).
		Where(claimablePredicate).
		Order(
			goqu.I("email_recipients.available_at").Asc(),
			goqu.I("email_recipients.created_at").Asc(),
		).
		Limit(input.Limit).
		ForUpdate(goqu.SkipLocked)

	sql, args, err := toSQLWithTag("emailRecipients.ClaimRecipients", dialect.Update(goqu.T("email_recipients")).
		Prepared(true).
		With("claimable", claimable).
		Set(goqu.Record{
			"version":         goqu.L("version + 1"),
			"updated_at":      now,
			"attempt_count":   goqu.L("attempt_count + 1"),
			"available_at":    now.Add(models.EmailRecipientRetryDelay),
			"last_attempt_at": now,
		}).
		From(goqu.T("claimable")).
		Where(goqu.I("email_recipients.id").Eq(goqu.I("claimable.id"))).
		Returning(e.getSelectFields()...))
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	rows, err := e.dbClient.getConnection(ctx).Query(ctx, sql, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}
	defer rows.Close()

	var recipients []*models.EmailRecipient
	for rows.Next() {
		item, sErr := scanEmailRecipient(rows)
		if sErr != nil {
			return nil, errors.Wrap(sErr, "failed to scan row", errors.WithSpan(span))
		}
		recipients = append(recipients, item)
	}

	if err = rows.Err(); err != nil {
		return nil, errors.Wrap(err, "failed to iterate rows", errors.WithSpan(span))
	}

	return recipients, nil
}

// AllRecipientsFinal reports whether the outbox item has no recipient left in a non-final status, using NOT EXISTS so it short-circuits on the first match rather than counting.
func (e *emailRecipients) AllRecipientsFinal(ctx context.Context, emailOutboxItemID string) (bool, error) {
	ctx, span := tracer.Start(ctx, "db.AllRecipientsFinal")
	defer span.End()

	nonFinalStatuses := models.EmailDeliveryStatusesMatching(func(s models.EmailDeliveryStatus) bool {
		return !s.IsFinal()
	})

	inner := dialect.From(goqu.T("email_recipients")).
		Select(goqu.L("1")).
		Where(goqu.And(
			goqu.I("email_recipients.email_outbox_item_id").Eq(emailOutboxItemID),
			goqu.I("email_recipients.delivery_status").In(nonFinalStatuses),
		))

	sql, args, err := toSQLWithTag("emailRecipients.AllRecipientsFinal", dialect.
		Select(goqu.L("not exists ?", inner)).
		Prepared(true))
	if err != nil {
		return false, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	var allFinal bool
	if err := e.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...).Scan(&allFinal); err != nil {
		return false, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return allFinal, nil
}

func (e *emailRecipients) getRecipient(ctx context.Context, exp goqu.Ex) (*models.EmailRecipient, error) {
	ctx, span := tracer.Start(ctx, "db.getRecipient")
	defer span.End()

	sql, args, err := toSQLWithTag("emailRecipients.getRecipient", dialect.From(goqu.T("email_recipients")).
		Prepared(true).
		Select(e.getSelectFields()...).
		Where(exp))
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	recipient, err := scanEmailRecipient(e.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}

		if pgErr := asPgError(err); pgErr != nil && isInvalidIDViolation(pgErr) {
			return nil, ErrInvalidID
		}

		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return recipient, nil
}

func (*emailRecipients) getSelectFields() []interface{} {
	selectFields := []any{}
	for _, field := range emailRecipientsFieldList {
		selectFields = append(selectFields, fmt.Sprintf("email_recipients.%s", field))
	}
	return selectFields
}

// getReturningFields returns bare column names for RETURNING clauses (no table prefix).
func (*emailRecipients) getReturningFields() []interface{} {
	return emailRecipientsFieldList
}

func scanEmailRecipient(row scanner) (*models.EmailRecipient, error) {
	recipient := &models.EmailRecipient{}

	fields := []interface{}{
		&recipient.Metadata.ID,
		&recipient.Metadata.CreationTimestamp,
		&recipient.Metadata.LastUpdatedTimestamp,
		&recipient.Metadata.Version,
		&recipient.EmailOutboxItemID,
		&recipient.Address,
		&recipient.DeliveryStatus,
		&recipient.AttemptCount,
		&recipient.AvailableAt,
		&recipient.LastAttemptAt,
		&recipient.OpenedAt,
		&recipient.ClickedAt,
		&recipient.ComplainedAt,
		&recipient.FailureReason,
	}

	if err := row.Scan(fields...); err != nil {
		return nil, err
	}

	recipient.Metadata.TRN = trn.TypeEmailRecipient.Build(recipient.GetGlobalID())

	return recipient, nil
}
