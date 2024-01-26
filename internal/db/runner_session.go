package db

//go:generate mockery --name RunnerSessions --inpackage --case underscore

import (
	"context"
	"fmt"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v4"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// RunnerSessions encapsulates the logic to access sessions from the database
type RunnerSessions interface {
	GetRunnerSessionByID(ctx context.Context, id string) (*models.RunnerSession, error)
	GetRunnerSessions(ctx context.Context, input *GetRunnerSessionsInput) (*RunnerSessionsResult, error)
	CreateRunnerSession(ctx context.Context, session *models.RunnerSession) (*models.RunnerSession, error)
	UpdateRunnerSession(ctx context.Context, session *models.RunnerSession) (*models.RunnerSession, error)
	DeleteRunnerSession(ctx context.Context, session *models.RunnerSession) error
}

// RunnerSessionSortableField represents the fields that sessions can be sorted by
type RunnerSessionSortableField string

// RunnerSessionSortableField constants
const (
	RunnerSessionSortableFieldCreatedAtAsc        RunnerSessionSortableField = "CREATED_AT_ASC"
	RunnerSessionSortableFieldCreatedAtDesc       RunnerSessionSortableField = "CREATED_AT_DESC"
	RunnerSessionSortableFieldLastContactedAtAsc  RunnerSessionSortableField = "LAST_CONTACTED_AT_ASC"
	RunnerSessionSortableFieldLastContactedAtDesc RunnerSessionSortableField = "LAST_CONTACTED_AT_DESC"
)

func (as RunnerSessionSortableField) getValue() string {
	return string(as)
}

func (as RunnerSessionSortableField) getFieldDescriptor() *pagination.FieldDescriptor {
	switch as {
	case RunnerSessionSortableFieldCreatedAtAsc, RunnerSessionSortableFieldCreatedAtDesc:
		return &pagination.FieldDescriptor{Key: "created_at", Table: "runner_sessions", Col: "created_at"}
	case RunnerSessionSortableFieldLastContactedAtAsc, RunnerSessionSortableFieldLastContactedAtDesc:
		return &pagination.FieldDescriptor{Key: "last_contacted_at", Table: "runner_sessions", Col: "last_contacted_at"}
	default:
		return nil
	}
}

func (as RunnerSessionSortableField) getSortDirection() pagination.SortDirection {
	if strings.HasSuffix(string(as), "_DESC") {
		return pagination.DescSort
	}
	return pagination.AscSort
}

// RunnerSessionFilter contains the supported fields for filtering RunnerSession resources
type RunnerSessionFilter struct {
	RunnerID         *string
	RunnerSessionIDs []string
}

// GetRunnerSessionsInput is the input for listing sessions
type GetRunnerSessionsInput struct {
	// Sort specifies the field to sort on and direction
	Sort *RunnerSessionSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// Filter is used to filter the results
	Filter *RunnerSessionFilter
}

// RunnerSessionsResult contains the response data and page information
type RunnerSessionsResult struct {
	PageInfo       *pagination.PageInfo
	RunnerSessions []models.RunnerSession
}

type sessions struct {
	dbClient *Client
}

var sessionFieldList = append(metadataFieldList, "runner_id", "last_contacted_at", "error_count", "internal")

// NewRunnerSessions returns an instance of the RunnerSessions interface
func NewRunnerSessions(dbClient *Client) RunnerSessions {
	return &sessions{dbClient: dbClient}
}

func (a *sessions) GetRunnerSessionByID(ctx context.Context, id string) (*models.RunnerSession, error) {
	ctx, span := tracer.Start(ctx, "db.GetRunnerSessionByID")
	defer span.End()

	return a.getRunnerSession(ctx, goqu.Ex{"runner_sessions.id": id})
}

func (a *sessions) GetRunnerSessions(ctx context.Context, input *GetRunnerSessionsInput) (*RunnerSessionsResult, error) {
	ctx, span := tracer.Start(ctx, "db.GetRunnerSessions")
	defer span.End()

	ex := goqu.And()

	if input.Filter != nil {
		if input.Filter.RunnerID != nil {
			ex = ex.Append(goqu.I("runner_sessions.runner_id").Eq(*input.Filter.RunnerID))
		}

		if len(input.Filter.RunnerSessionIDs) > 0 {
			ex = ex.Append(goqu.I("runner_sessions.id").In(input.Filter.RunnerSessionIDs))
		}
	}

	query := dialect.From(goqu.T("runner_sessions")).
		Select(a.getSelectFields()...).
		Where(ex)

	sortDirection := pagination.AscSort

	var sortBy *pagination.FieldDescriptor
	if input.Sort != nil {
		sortDirection = input.Sort.getSortDirection()
		sortBy = input.Sort.getFieldDescriptor()
	}

	qBuilder, err := pagination.NewPaginatedQueryBuilder(
		input.PaginationOptions,
		&pagination.FieldDescriptor{Key: "id", Table: "runner_sessions", Col: "id"},
		sortBy,
		sortDirection,
	)

	if err != nil {
		return nil, errors.Wrap(err, "failed to build query", errors.WithSpan(span))
	}

	rows, err := qBuilder.Execute(ctx, a.dbClient.getConnection(ctx), query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	defer rows.Close()

	// Scan rows
	results := []models.RunnerSession{}
	for rows.Next() {
		item, err := scanRunnerSession(rows)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan row", errors.WithSpan(span))
		}

		results = append(results, *item)
	}

	if err := rows.Finalize(&results); err != nil {
		return nil, errors.Wrap(err, "failed to finalize rows", errors.WithSpan(span))
	}

	result := RunnerSessionsResult{
		PageInfo:       rows.GetPageInfo(),
		RunnerSessions: results,
	}

	return &result, nil
}

func (a *sessions) CreateRunnerSession(ctx context.Context, session *models.RunnerSession) (*models.RunnerSession, error) {
	ctx, span := tracer.Start(ctx, "db.CreateRunnerSession")
	defer span.End()

	timestamp := currentTime()

	sql, args, err := dialect.From("runner_sessions").
		Prepared(true).
		With("runner_sessions",
			dialect.Insert("runner_sessions").Rows(
				goqu.Record{
					"id":                newResourceID(),
					"version":           initialResourceVersion,
					"created_at":        timestamp,
					"updated_at":        timestamp,
					"runner_id":         session.RunnerID,
					"last_contacted_at": session.LastContactTimestamp,
					"error_count":       session.ErrorCount,
					"internal":          session.Internal,
				}).Returning("*"),
		).Select(a.getSelectFields()...).
		ToSQL()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	createdRunnerSession, err := scanRunnerSession(a.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isForeignKeyViolation(pgErr) {
				return nil, errors.New("invalid runner ID", errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
			}
		}
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return createdRunnerSession, nil
}

func (a *sessions) UpdateRunnerSession(ctx context.Context, session *models.RunnerSession) (*models.RunnerSession, error) {
	ctx, span := tracer.Start(ctx, "db.UpdateRunnerSession")
	defer span.End()

	timestamp := currentTime()

	sql, args, err := dialect.From("runner_sessions").
		Prepared(true).
		With("runner_sessions",
			dialect.Update("runner_sessions").
				Set(goqu.Record{
					"version":           goqu.L("? + ?", goqu.C("version"), 1),
					"updated_at":        timestamp,
					"last_contacted_at": session.LastContactTimestamp,
					"error_count":       session.ErrorCount,
				}).Where(goqu.Ex{"id": session.Metadata.ID, "version": session.Metadata.Version}).
				Returning("*"),
		).Select(a.getSelectFields()...).
		ToSQL()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	updatedRunnerSession, err := scanRunnerSession(a.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return nil, ErrOptimisticLockError
		}
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return updatedRunnerSession, nil
}

func (a *sessions) DeleteRunnerSession(ctx context.Context, session *models.RunnerSession) error {
	ctx, span := tracer.Start(ctx, "db.DeleteRunnerSession")
	defer span.End()

	sql, args, err := dialect.From("runner_sessions").
		Prepared(true).
		With("runner_sessions",
			dialect.Delete("runner_sessions").
				Where(goqu.Ex{"id": session.Metadata.ID, "version": session.Metadata.Version}).
				Returning("*"),
		).Select(a.getSelectFields()...).
		ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	_, err = scanRunnerSession(a.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return ErrOptimisticLockError
		}
		return errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return nil
}

func (a *sessions) getRunnerSession(ctx context.Context, exp goqu.Ex) (*models.RunnerSession, error) {
	ctx, span := tracer.Start(ctx, "db.getRunnerSession")
	defer span.End()

	query := dialect.From(goqu.T("runner_sessions")).
		Prepared(true).
		Select(a.getSelectFields()...).
		Where(exp)

	sql, args, err := query.ToSQL()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	session, err := scanRunnerSession(a.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}

		if pgErr := asPgError(err); pgErr != nil {
			if isInvalidIDViolation(pgErr) {
				return nil, errors.Wrap(pgErr, "invalid ID; %s", pgErr.Message, errors.WithSpan(span), errors.WithErrorCode(errors.EInvalid))
			}
		}

		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return session, nil
}

func (*sessions) getSelectFields() []interface{} {
	selectFields := []interface{}{}
	for _, field := range sessionFieldList {
		selectFields = append(selectFields, fmt.Sprintf("runner_sessions.%s", field))
	}

	return selectFields
}

func scanRunnerSession(row scanner) (*models.RunnerSession, error) {
	session := &models.RunnerSession{}

	fields := []interface{}{
		&session.Metadata.ID,
		&session.Metadata.CreationTimestamp,
		&session.Metadata.LastUpdatedTimestamp,
		&session.Metadata.Version,
		&session.RunnerID,
		&session.LastContactTimestamp,
		&session.ErrorCount,
		&session.Internal,
	}

	err := row.Scan(fields...)
	if err != nil {
		return nil, err
	}

	return session, nil
}
