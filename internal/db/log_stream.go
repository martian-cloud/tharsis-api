package db

//go:generate go tool mockery --name LogStreams --inpackage --case underscore

import (
	"context"
	"fmt"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/jackc/pgx/v4"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// LogStreamSortableField represents the fields that a log stream can be sorted by
type LogStreamSortableField string

// LogStreamSortableField constants
const (
	LogStreamSortableFieldUpdatedAtAsc  LogStreamSortableField = "UPDATED_AT_ASC"
	LogStreamSortableFieldUpdatedAtDesc LogStreamSortableField = "UPDATED_AT_DESC"
)

func (lf LogStreamSortableField) getFieldDescriptor() *pagination.FieldDescriptor {
	switch lf {
	case LogStreamSortableFieldUpdatedAtAsc, LogStreamSortableFieldUpdatedAtDesc:
		return &pagination.FieldDescriptor{Key: "updated_at", Table: "log_streams", Col: "updated_at"}
	default:
		return nil
	}
}

func (lf LogStreamSortableField) getSortDirection() pagination.SortDirection {
	if strings.HasSuffix(string(lf), "_DESC") {
		return pagination.DescSort
	}
	return pagination.AscSort
}

// LogStreamFilter contains the supported fields for filtering log stream resources
type LogStreamFilter struct {
	RunnerSessionIDs []string
	JobIDs           []string
}

// GetLogStreamsInput is the input for listing log streams
type GetLogStreamsInput struct {
	// Sort specifies the field to sort on and direction
	Sort *LogStreamSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// Filter is used to filter the results
	Filter *LogStreamFilter
}

// LogStreamsResult contains the response data and page information
type LogStreamsResult struct {
	PageInfo   *pagination.PageInfo
	LogStreams []models.LogStream
}

// LogStreams encapsulates the logic to access LogStreams from the database
type LogStreams interface {
	GetLogStreamByID(ctx context.Context, id string) (*models.LogStream, error)
	GetLogStreamByJobID(ctx context.Context, jobID string) (*models.LogStream, error)
	GetLogStreamByRunnerSessionID(ctx context.Context, sessionID string) (*models.LogStream, error)
	GetLogStreams(ctx context.Context, input *GetLogStreamsInput) (*LogStreamsResult, error)
	CreateLogStream(ctx context.Context, logStream *models.LogStream) (*models.LogStream, error)
	UpdateLogStream(ctx context.Context, logStream *models.LogStream) (*models.LogStream, error)
}

var logStreamFieldList = append(metadataFieldList, "size", "job_id", "runner_session_id", "completed")

type logStreams struct {
	dbClient *Client
}

// NewLogStreams returns an instance of the LogStreams interface
func NewLogStreams(dbClient *Client) LogStreams {
	return &logStreams{dbClient: dbClient}
}

func (l *logStreams) GetLogStreamByID(ctx context.Context, id string) (*models.LogStream, error) {
	ctx, span := tracer.Start(ctx, "db.GetLogStreamByID")
	defer span.End()

	return l.getLogStream(ctx, goqu.Ex{"log_streams.id": id})
}

func (l *logStreams) GetLogStreamByJobID(ctx context.Context, jobID string) (*models.LogStream, error) {
	ctx, span := tracer.Start(ctx, "db.GetLogStreamByJobID")
	defer span.End()

	return l.getLogStream(ctx, goqu.Ex{"log_streams.job_id": jobID})
}

func (l *logStreams) GetLogStreamByRunnerSessionID(ctx context.Context, sessionID string) (*models.LogStream, error) {
	ctx, span := tracer.Start(ctx, "db.GetLogStreamByRunnerSessionID")
	defer span.End()

	return l.getLogStream(ctx, goqu.Ex{"log_streams.runner_session_id": sessionID})
}

func (l *logStreams) GetLogStreams(ctx context.Context, input *GetLogStreamsInput) (*LogStreamsResult, error) {
	ctx, span := tracer.Start(ctx, "db.GetLogStreams")
	defer span.End()

	ex := goqu.Ex{}

	if input.Filter != nil {
		if input.Filter.JobIDs != nil {
			ex["log_streams.job_id"] = input.Filter.JobIDs
		}
		if input.Filter.RunnerSessionIDs != nil {
			ex["log_streams.runner_session_id"] = input.Filter.RunnerSessionIDs
		}
	}

	query := dialect.From(goqu.T("log_streams")).
		Select(l.getSelectFields()...).
		Where(ex)

	sortDirection := pagination.AscSort

	var sortBy *pagination.FieldDescriptor
	if input.Sort != nil {
		sortDirection = input.Sort.getSortDirection()
		sortBy = input.Sort.getFieldDescriptor()
	}

	qBuilder, err := pagination.NewPaginatedQueryBuilder(
		input.PaginationOptions,
		&pagination.FieldDescriptor{Key: "id", Table: "log_streams", Col: "id"},
		pagination.WithSortByField(sortBy, sortDirection),
	)

	if err != nil {
		return nil, errors.Wrap(err, "failed to build query", errors.WithSpan(span))
	}

	rows, err := qBuilder.Execute(ctx, l.dbClient.getConnection(ctx), query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	defer rows.Close()

	// Scan rows
	results := []models.LogStream{}
	for rows.Next() {
		item, err := scanLogStream(rows)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan row", errors.WithSpan(span))
		}

		results = append(results, *item)
	}

	if err := rows.Finalize(&results); err != nil {
		return nil, errors.Wrap(err, "failed to finalize rows", errors.WithSpan(span))
	}

	result := LogStreamsResult{
		PageInfo:   rows.GetPageInfo(),
		LogStreams: results,
	}

	return &result, nil
}

func (l *logStreams) CreateLogStream(ctx context.Context, logStream *models.LogStream) (*models.LogStream, error) {
	ctx, span := tracer.Start(ctx, "db.CreateLogStream")
	defer span.End()

	timestamp := currentTime()

	sql, args, err := dialect.Insert("log_streams").
		Prepared(true).
		Rows(goqu.Record{
			"id":                newResourceID(),
			"version":           initialResourceVersion,
			"created_at":        timestamp,
			"updated_at":        timestamp,
			"size":              logStream.Size,
			"job_id":            logStream.JobID,
			"runner_session_id": logStream.RunnerSessionID,
			"completed":         logStream.Completed,
		}).
		Returning(logStreamFieldList...).ToSQL()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	createdLogStream, err := scanLogStream(l.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isForeignKeyViolation(pgErr) {
				switch pgErr.ConstraintName {
				case "fk_job_id":
					return nil, errors.New("job does not exist", errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
				case "fk_log_streams_runner_session_id":
					return nil, errors.New("runner session does not exist", errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
				}
			}
			if isUniqueViolation(pgErr) {
				switch pgErr.ConstraintName {
				case "index_log_streams_on_job_id":
					return nil, errors.New("log stream already exists for job %s", *logStream.JobID,
						errors.WithErrorCode(errors.EConflict), errors.WithSpan(span))
				case "index_log_streams_on_runner_session_id":
					return nil, errors.New("log stream already exists for runner session %s", *logStream.RunnerSessionID,
						errors.WithErrorCode(errors.EConflict), errors.WithSpan(span))
				}
			}
		}

		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return createdLogStream, nil
}

func (l *logStreams) UpdateLogStream(ctx context.Context, logStream *models.LogStream) (*models.LogStream, error) {
	ctx, span := tracer.Start(ctx, "db.UpdateLogStream")
	defer span.End()

	timestamp := currentTime()

	sql, args, err := dialect.Update("log_streams").
		Prepared(true).
		Set(
			goqu.Record{
				"version":    goqu.L("? + ?", goqu.C("version"), 1),
				"updated_at": timestamp,
				"size":       logStream.Size,
				"completed":  logStream.Completed,
			},
		).Where(goqu.Ex{"id": logStream.Metadata.ID, "version": logStream.Metadata.Version}).
		Returning(logStreamFieldList...).ToSQL()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	updatedLogStream, err := scanLogStream(l.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrOptimisticLockError
		}
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return updatedLogStream, nil
}

func (l *logStreams) getLogStream(ctx context.Context, exp exp.Expression) (*models.LogStream, error) {
	query := dialect.From(goqu.T("log_streams")).
		Prepared(true).
		Select(l.getSelectFields()...).
		Where(exp)

	sql, args, err := query.ToSQL()
	if err != nil {
		return nil, err
	}

	logStream, err := scanLogStream(l.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}

		if pgErr := asPgError(err); pgErr != nil {
			if isInvalidIDViolation(pgErr) {
				return nil, errors.Wrap(pgErr, pgErr.Message, errors.WithErrorCode(errors.EInvalid))
			}
		}

		return nil, err
	}

	return logStream, nil
}

func (*logStreams) getSelectFields() []interface{} {
	selectFields := []interface{}{}
	for _, field := range logStreamFieldList {
		selectFields = append(selectFields, fmt.Sprintf("log_streams.%s", field))
	}

	return selectFields
}

func scanLogStream(row scanner) (*models.LogStream, error) {
	logStream := &models.LogStream{}

	fields := []interface{}{
		&logStream.Metadata.ID,
		&logStream.Metadata.CreationTimestamp,
		&logStream.Metadata.LastUpdatedTimestamp,
		&logStream.Metadata.Version,
		&logStream.Size,
		&logStream.JobID,
		&logStream.RunnerSessionID,
		&logStream.Completed,
	}

	if err := row.Scan(fields...); err != nil {
		return nil, err
	}

	return logStream, nil
}
