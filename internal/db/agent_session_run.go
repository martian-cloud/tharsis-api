package db

//go:generate go tool mockery --name AgentSessionRuns --inpackage --case underscore

import (
	"context"
	"fmt"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v4"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// AgentSessionRuns encapsulates the logic to access agent session runs from the database
type AgentSessionRuns interface {
	GetAgentSessionRunByID(ctx context.Context, id string) (*models.AgentSessionRun, error)
	GetAgentSessionRunByTRN(ctx context.Context, trn string) (*models.AgentSessionRun, error)
	GetAgentSessionRuns(ctx context.Context, input *GetAgentSessionRunsInput) (*AgentSessionRunsResult, error)
	CreateAgentSessionRun(ctx context.Context, run *models.AgentSessionRun) (*models.AgentSessionRun, error)
	UpdateAgentSessionRun(ctx context.Context, run *models.AgentSessionRun) (*models.AgentSessionRun, error)
}

// AgentSessionRunSortableField represents the fields that a run can be sorted by
type AgentSessionRunSortableField string

// AgentSessionRunSortableField constants
const (
	AgentSessionRunSortableFieldCreatedAtAsc  AgentSessionRunSortableField = "CREATED_AT_ASC"
	AgentSessionRunSortableFieldCreatedAtDesc AgentSessionRunSortableField = "CREATED_AT_DESC"
)

func (sf AgentSessionRunSortableField) getFieldDescriptor() *pagination.FieldDescriptor {
	switch sf {
	case AgentSessionRunSortableFieldCreatedAtAsc, AgentSessionRunSortableFieldCreatedAtDesc:
		return &pagination.FieldDescriptor{Key: "created_at", Table: "agent_session_runs", Col: "created_at"}
	default:
		return nil
	}
}

func (sf AgentSessionRunSortableField) getSortDirection() pagination.SortDirection {
	if strings.HasSuffix(string(sf), "_DESC") {
		return pagination.DescSort
	}
	return pagination.AscSort
}

// AgentSessionRunFilter contains the supported fields for filtering agent session runs.
type AgentSessionRunFilter struct {
	SessionID     *string
	PreviousRunID *string
}

// GetAgentSessionRunsInput is the input for listing agent session runs.
type GetAgentSessionRunsInput struct {
	Sort              *AgentSessionRunSortableField
	PaginationOptions *pagination.Options
	Filter            *AgentSessionRunFilter
}

// AgentSessionRunsResult contains the response data and page information.
type AgentSessionRunsResult struct {
	PageInfo         *pagination.PageInfo
	AgentSessionRuns []models.AgentSessionRun
}

type agentSessionRuns struct {
	dbClient *Client
}

var agentSessionRunFieldList = append(metadataFieldList, "session_id", "previous_run_id", "last_message_id", "status", "error_message", "cancel_requested")

// NewAgentSessionRuns returns an instance of the AgentSessionRuns interface
func NewAgentSessionRuns(dbClient *Client) AgentSessionRuns {
	return &agentSessionRuns{dbClient: dbClient}
}

func (a *agentSessionRuns) GetAgentSessionRunByID(ctx context.Context, id string) (*models.AgentSessionRun, error) {
	ctx, span := tracer.Start(ctx, "db.GetAgentSessionRunByID")
	defer span.End()

	sql, args, err := dialect.From(goqu.T("agent_session_runs")).
		Prepared(true).
		Select(a.getSelectFields()...).
		Where(goqu.Ex{"agent_session_runs.id": id}).
		ToSQL()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	run, err := scanAgentSessionRun(a.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		if pgErr := asPgError(err); pgErr != nil {
			if isInvalidIDViolation(pgErr) {
				return nil, ErrInvalidID
			}
		}
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return run, nil
}

func (a *agentSessionRuns) GetAgentSessionRunByTRN(ctx context.Context, trn string) (*models.AgentSessionRun, error) {
	ctx, span := tracer.Start(ctx, "db.GetAgentSessionRunByTRN")
	defer span.End()

	path, err := types.AgentSessionRunModelType.ResourcePathFromTRN(trn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse TRN", errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
	}

	// TRN path is sessionID/runID — extract the run ID
	parts := strings.Split(path, "/")
	return a.GetAgentSessionRunByID(ctx, gid.FromGlobalID(parts[len(parts)-1]))
}

func (a *agentSessionRuns) GetAgentSessionRuns(ctx context.Context, input *GetAgentSessionRunsInput) (*AgentSessionRunsResult, error) {
	ctx, span := tracer.Start(ctx, "db.GetAgentSessionRuns")
	defer span.End()

	ex := goqu.Ex{}
	if input.Filter != nil {
		if input.Filter.SessionID != nil {
			ex["agent_session_runs.session_id"] = *input.Filter.SessionID
		}
		if input.Filter.PreviousRunID != nil {
			ex["agent_session_runs.previous_run_id"] = *input.Filter.PreviousRunID
		}
	}

	query := dialect.From("agent_session_runs").
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
		&pagination.FieldDescriptor{Key: "id", Table: "agent_session_runs", Col: "id"},
		pagination.WithSortByField(sortBy, sortDirection),
	)
	if err != nil {
		tracing.RecordError(span, err, "failed to build query")
		return nil, err
	}

	rows, err := qBuilder.Execute(ctx, a.dbClient.getConnection(ctx), query)
	if err != nil {
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}
	defer rows.Close()

	results := []models.AgentSessionRun{}
	for rows.Next() {
		item, err := scanAgentSessionRun(rows)
		if err != nil {
			tracing.RecordError(span, err, "failed to scan row")
			return nil, err
		}
		results = append(results, *item)
	}

	if err := rows.Finalize(&results); err != nil {
		tracing.RecordError(span, err, "failed to finalize rows")
		return nil, err
	}

	return &AgentSessionRunsResult{
		PageInfo:         rows.GetPageInfo(),
		AgentSessionRuns: results,
	}, nil
}

func (a *agentSessionRuns) CreateAgentSessionRun(ctx context.Context, run *models.AgentSessionRun) (*models.AgentSessionRun, error) {
	ctx, span := tracer.Start(ctx, "db.CreateAgentSessionRun")
	defer span.End()

	timestamp := currentTime()

	sql, args, err := dialect.Insert("agent_session_runs").Prepared(true).Rows(
		goqu.Record{
			"id":               newResourceID(),
			"version":          initialResourceVersion,
			"created_at":       timestamp,
			"updated_at":       timestamp,
			"session_id":       run.SessionID,
			"previous_run_id":  run.PreviousRunID,
			"last_message_id":  run.LastMessageID,
			"status":           run.Status,
			"error_message":    run.ErrorMessage,
			"cancel_requested": run.CancelRequested,
		},
	).Returning(agentSessionRunFieldList...).ToSQL()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	created, err := scanAgentSessionRun(a.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isForeignKeyViolation(pgErr) {
				return nil, errors.New("invalid session ID", errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
			}
			if isInvalidIDViolation(pgErr) {
				return nil, errors.Wrap(pgErr, "invalid ID; %s", pgErr.Message, errors.WithSpan(span), errors.WithErrorCode(errors.EInvalid))
			}
		}
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return created, nil
}

func (a *agentSessionRuns) UpdateAgentSessionRun(ctx context.Context, run *models.AgentSessionRun) (*models.AgentSessionRun, error) {
	ctx, span := tracer.Start(ctx, "db.UpdateAgentSessionRun")
	defer span.End()

	timestamp := currentTime()

	sql, args, err := dialect.Update("agent_session_runs").
		Prepared(true).
		Set(goqu.Record{
			"version":          goqu.L("? + ?", goqu.C("version"), 1),
			"updated_at":       timestamp,
			"last_message_id":  run.LastMessageID,
			"status":           run.Status,
			"error_message":    run.ErrorMessage,
			"cancel_requested": run.CancelRequested,
		}).
		Where(goqu.Ex{"id": run.Metadata.ID, "version": run.Metadata.Version}).
		Returning(agentSessionRunFieldList...).
		ToSQL()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	updated, err := scanAgentSessionRun(a.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return nil, ErrOptimisticLockError
		}
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return updated, nil
}

func (*agentSessionRuns) getSelectFields() []interface{} {
	selectFields := []interface{}{}
	for _, field := range agentSessionRunFieldList {
		selectFields = append(selectFields, fmt.Sprintf("agent_session_runs.%s", field))
	}
	return selectFields
}

func scanAgentSessionRun(row scanner) (*models.AgentSessionRun, error) {
	run := &models.AgentSessionRun{}

	fields := []interface{}{
		&run.Metadata.ID,
		&run.Metadata.CreationTimestamp,
		&run.Metadata.LastUpdatedTimestamp,
		&run.Metadata.Version,
		&run.SessionID,
		&run.PreviousRunID,
		&run.LastMessageID,
		&run.Status,
		&run.ErrorMessage,
		&run.CancelRequested,
	}

	err := row.Scan(fields...)
	if err != nil {
		return nil, err
	}

	run.Metadata.TRN = types.AgentSessionRunModelType.BuildTRN(gid.ToGlobalID(types.AgentSessionModelType, run.SessionID), run.GetGlobalID())

	return run, nil
}
