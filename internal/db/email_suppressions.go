package db

//go:generate go tool mockery --name EmailSuppressions --inpackage --case underscore

import (
	"context"
	"fmt"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v5"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
)

// EmailSuppressions encapsulates the logic to access the email suppression list from the database.
type EmailSuppressions interface {
	GetSuppressionByID(ctx context.Context, id string) (*models.EmailSuppression, error)
	GetSuppressionByTRN(ctx context.Context, trn string) (*models.EmailSuppression, error)
	GetSuppressions(ctx context.Context, input *GetEmailSuppressionsInput) (*EmailSuppressionsResult, error)
	CreateSuppression(ctx context.Context, entry *models.EmailSuppression) (*models.EmailSuppression, error)
	DeleteSuppression(ctx context.Context, entry *models.EmailSuppression) error
}

// EmailSuppressionSortableField represents the fields that suppression entries can be sorted by.
type EmailSuppressionSortableField string

// EmailSuppressionSortableField constants.
const (
	EmailSuppressionSortableFieldCreatedAtAsc  EmailSuppressionSortableField = "CREATED_AT_ASC"
	EmailSuppressionSortableFieldCreatedAtDesc EmailSuppressionSortableField = "CREATED_AT_DESC"
)

func (sf EmailSuppressionSortableField) getFieldDescriptor() *pagination.FieldDescriptor {
	switch sf {
	case EmailSuppressionSortableFieldCreatedAtAsc, EmailSuppressionSortableFieldCreatedAtDesc:
		return &pagination.FieldDescriptor{Key: "created_at", Table: "email_suppressions", Col: "created_at"}
	default:
		return nil
	}
}

func (sf EmailSuppressionSortableField) getSortDirection() pagination.SortDirection {
	if strings.HasSuffix(string(sf), "_DESC") {
		return pagination.DescSort
	}
	return pagination.AscSort
}

// EmailSuppressionFilter contains the supported fields for filtering suppression entries.
type EmailSuppressionFilter struct {
	// Addresses restricts the results to entries matching these addresses (case-insensitive).
	Addresses []string
	// Search filters suppressions whose address contains the given substring (case-insensitive).
	Search *string
}

// GetEmailSuppressionsInput is the input for listing suppression entries.
type GetEmailSuppressionsInput struct {
	Sort              *EmailSuppressionSortableField
	PaginationOptions *pagination.Options
	Filter            *EmailSuppressionFilter
}

// EmailSuppressionsResult contains the response data and page information.
type EmailSuppressionsResult struct {
	PageInfo     *pagination.PageInfo
	Suppressions []*models.EmailSuppression
}

type emailSuppressions struct {
	dbClient *Client
}

var emailSuppressionsFieldList = append(metadataFieldList, "address", "cause")

// NewEmailSuppressions returns an instance of the EmailSuppressions interface.
func NewEmailSuppressions(dbClient *Client) EmailSuppressions {
	return &emailSuppressions{dbClient: dbClient}
}

func (e *emailSuppressions) GetSuppressionByID(ctx context.Context, id string) (*models.EmailSuppression, error) {
	ctx, span := tracer.Start(ctx, "db.GetSuppressionByID")
	defer span.End()

	return e.getSuppression(ctx, goqu.Ex{"email_suppressions.id": id})
}

func (e *emailSuppressions) GetSuppressionByTRN(ctx context.Context, trnValue string) (*models.EmailSuppression, error) {
	ctx, span := tracer.Start(ctx, "db.GetSuppressionByTRN")
	defer span.End()

	parsed, err := trn.TypeEmailSuppression.Parse(trnValue)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse TRN", errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
	}

	return e.getSuppression(ctx, goqu.Ex{"email_suppressions.address": parsed.Path()})
}

func (e *emailSuppressions) GetSuppressions(ctx context.Context, input *GetEmailSuppressionsInput) (*EmailSuppressionsResult, error) {
	ctx, span := tracer.Start(ctx, "db.GetSuppressions")
	defer span.End()

	ex := goqu.And()

	if input.Filter != nil {
		if len(input.Filter.Addresses) > 0 {
			// The address column is stored and uniquely indexed on lower(address), so match case-insensitively.
			lowered := make([]string, len(input.Filter.Addresses))
			for i, a := range input.Filter.Addresses {
				lowered[i] = strings.ToLower(a)
			}

			ex = ex.Append(goqu.L("lower(email_suppressions.address)").In(lowered))
		}

		if input.Filter.Search != nil && *input.Filter.Search != "" {
			ex = ex.Append(goqu.I("email_suppressions.address").ILike("%" + escapeLikePattern(*input.Filter.Search) + "%"))
		}
	}

	query := dialect.From(goqu.T("email_suppressions")).
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
		&pagination.FieldDescriptor{Key: "id", Table: "email_suppressions", Col: "id"},
		pagination.WithSortByField(sortBy, sortDirection),
		pagination.WithQueryTag("emailSuppressions.GetSuppressions"),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build query", errors.WithSpan(span))
	}

	rows, err := qBuilder.Execute(ctx, e.dbClient.getConnection(ctx), query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}
	defer rows.Close()

	results := []*models.EmailSuppression{}
	for rows.Next() {
		item, sErr := scanEmailSuppression(rows)
		if sErr != nil {
			return nil, errors.Wrap(sErr, "failed to scan row", errors.WithSpan(span))
		}
		results = append(results, item)
	}

	if err = rows.Finalize(&results); err != nil {
		return nil, errors.Wrap(err, "failed to finalize rows", errors.WithSpan(span))
	}

	return &EmailSuppressionsResult{
		PageInfo:     rows.GetPageInfo(),
		Suppressions: results,
	}, nil
}

func (e *emailSuppressions) CreateSuppression(ctx context.Context, entry *models.EmailSuppression) (*models.EmailSuppression, error) {
	ctx, span := tracer.Start(ctx, "db.CreateSuppression")
	defer span.End()

	timestamp := currentTime()

	sql, args, err := toSQLWithTag("emailSuppressions.CreateSuppression", dialect.Insert("email_suppressions").
		Prepared(true).
		Rows(goqu.Record{
			"id":         newResourceID(),
			"version":    initialResourceVersion,
			"created_at": timestamp,
			"updated_at": timestamp,
			"address":    strings.ToLower(entry.Address),
			"cause":      entry.Cause,
		}).
		Returning(emailSuppressionsFieldList...))
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	created, err := scanEmailSuppression(e.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil && isUniqueViolation(pgErr) {
			return nil, errors.New("email address is already suppressed", errors.WithErrorCode(errors.EConflict), errors.WithSpan(span))
		}

		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return created, nil
}

func (e *emailSuppressions) DeleteSuppression(ctx context.Context, entry *models.EmailSuppression) error {
	ctx, span := tracer.Start(ctx, "db.DeleteSuppression")
	defer span.End()

	sql, args, err := toSQLWithTag("emailSuppressions.DeleteSuppression", dialect.From("email_suppressions").
		Prepared(true).
		With("email_suppressions",
			dialect.Delete("email_suppressions").
				Where(goqu.Ex{"id": entry.Metadata.ID, "version": entry.Metadata.Version}).
				Returning("*"),
		).Select(e.getSelectFields()...))
	if err != nil {
		return errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	if _, err = scanEmailSuppression(e.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...)); err != nil {
		if err == pgx.ErrNoRows {
			return ErrOptimisticLockError
		}

		return errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return nil
}

func (e *emailSuppressions) getSuppression(ctx context.Context, exp goqu.Ex) (*models.EmailSuppression, error) {
	ctx, span := tracer.Start(ctx, "db.getSuppression")
	defer span.End()

	sql, args, err := toSQLWithTag("emailSuppressions.getSuppression", dialect.From(goqu.T("email_suppressions")).
		Prepared(true).
		Select(e.getSelectFields()...).
		Where(exp))
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	entry, err := scanEmailSuppression(e.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}

		if pgErr := asPgError(err); pgErr != nil && isInvalidIDViolation(pgErr) {
			return nil, ErrInvalidID
		}

		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return entry, nil
}

func (*emailSuppressions) getSelectFields() []interface{} {
	selectFields := []any{}
	for _, field := range emailSuppressionsFieldList {
		selectFields = append(selectFields, fmt.Sprintf("email_suppressions.%s", field))
	}

	return selectFields
}

func scanEmailSuppression(row scanner) (*models.EmailSuppression, error) {
	entry := &models.EmailSuppression{}

	fields := []any{
		&entry.Metadata.ID,
		&entry.Metadata.CreationTimestamp,
		&entry.Metadata.LastUpdatedTimestamp,
		&entry.Metadata.Version,
		&entry.Address,
		&entry.Cause,
	}

	if err := row.Scan(fields...); err != nil {
		return nil, err
	}

	entry.Metadata.TRN = trn.TypeEmailSuppression.Build(entry.Address)

	return entry, nil
}
