package db

//go:generate mockery --name Jobs --inpackage --case underscore

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/smithy-go/ptr"
	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v4"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// Jobs encapsulates the logic to access jobs from the database
type Jobs interface {
	GetJobByID(ctx context.Context, id string) (*models.Job, error)
	GetLatestJobByType(ctx context.Context, runID string, jobType models.JobType) (*models.Job, error)
	GetJobs(ctx context.Context, input *GetJobsInput) (*JobsResult, error)
	UpdateJob(ctx context.Context, job *models.Job) (*models.Job, error)
	CreateJob(ctx context.Context, job *models.Job) (*models.Job, error)
	GetJobCountForRunner(ctx context.Context, runnerID string) (int, error)
}

// JobSortableField represents the fields that a job can be sorted by
type JobSortableField string

// GroupSortableField constants
const (
	JobSortableFieldCreatedAtAsc  JobSortableField = "CREATED_AT_ASC"
	JobSortableFieldCreatedAtDesc JobSortableField = "CREATED_AT_DESC"
	JobSortableFieldUpdatedAtAsc  JobSortableField = "UPDATED_AT_ASC"
	JobSortableFieldUpdatedAtDesc JobSortableField = "UPDATED_AT_DESC"
)

func (js JobSortableField) getFieldDescriptor() *pagination.FieldDescriptor {
	switch js {
	case JobSortableFieldCreatedAtAsc, JobSortableFieldCreatedAtDesc:
		return &pagination.FieldDescriptor{Key: "created_at", Table: "jobs", Col: "created_at"}
	case JobSortableFieldUpdatedAtAsc, JobSortableFieldUpdatedAtDesc:
		return &pagination.FieldDescriptor{Key: "updated_at", Table: "jobs", Col: "updated_at"}
	default:
		return nil
	}
}

func (js JobSortableField) getSortDirection() pagination.SortDirection {
	if strings.HasSuffix(string(js), "_DESC") {
		return pagination.DescSort
	}
	return pagination.AscSort
}

// JobTagFilter is a filter condition for job tags
type JobTagFilter struct {
	ExcludeUntaggedJobs *bool
	TagSuperset         []string
}

// JobFilter contains the supported fields for filtering Job resources
type JobFilter struct {
	RunID       *string
	WorkspaceID *string
	RunnerID    *string
	JobType     *models.JobType
	JobStatus   *models.JobStatus
	TagFilter   *JobTagFilter
	JobIDs      []string
}

// GetJobsInput is the input for listing jobs
type GetJobsInput struct {
	// Sort specifies the field to sort on and direction
	Sort *JobSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// Filter is used to filter the results
	Filter *JobFilter
}

// JobsResult contains the response data and page information
type JobsResult struct {
	PageInfo *pagination.PageInfo
	Jobs     []models.Job
}

type jobs struct {
	dbClient *Client
}

var jobFieldList = append(metadataFieldList, "status", "type", "workspace_id", "run_id",
	"cancel_requested", "cancel_requested_at",
	"runner_id", "runner_path", "queued_at", "pending_at", "running_at", "finished_at", "max_job_duration", "tags")

// NewJobs returns an instance of the Jobs interface
func NewJobs(dbClient *Client) Jobs {
	return &jobs{dbClient: dbClient}
}

func (j *jobs) GetJobByID(ctx context.Context, id string) (*models.Job, error) {
	ctx, span := tracer.Start(ctx, "db.GetJobByID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	return j.getJob(ctx, goqu.Ex{"jobs.id": id})
}

func (j *jobs) GetLatestJobByType(ctx context.Context, runID string, jobType models.JobType) (*models.Job, error) {
	ctx, span := tracer.Start(ctx, "db.GetLatestJobByType")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	sortBy := JobSortableFieldUpdatedAtDesc
	jobResult, err := j.GetJobs(
		ctx,
		&GetJobsInput{
			PaginationOptions: &pagination.Options{First: ptr.Int32(1)},
			Filter:            &JobFilter{RunID: &runID, JobType: &jobType},
			Sort:              &sortBy,
		})
	if err != nil {
		tracing.RecordError(span, err, "failed to get job")
		return nil, errors.Wrap(err, "failed to get job")
	}

	if len(jobResult.Jobs) == 0 {
		return nil, nil
	}

	return &jobResult.Jobs[0], nil
}

func (j *jobs) GetJobs(ctx context.Context, input *GetJobsInput) (*JobsResult, error) {
	ctx, span := tracer.Start(ctx, "db.GetJobs")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	ex := goqu.And()

	if input.Filter != nil {
		if input.Filter.RunID != nil {
			ex = ex.Append(goqu.I("jobs.run_id").Eq(*input.Filter.RunID))
		}

		if input.Filter.WorkspaceID != nil {
			ex = ex.Append(goqu.I("jobs.workspace_id").Eq(*input.Filter.WorkspaceID))
		}

		if input.Filter.RunnerID != nil {
			ex = ex.Append(goqu.I("jobs.runner_id").Eq(*input.Filter.RunnerID))
		}

		if input.Filter.JobType != nil {
			ex = ex.Append(goqu.I("jobs.type").Eq(*input.Filter.JobType))
		}

		if input.Filter.JobStatus != nil {
			ex = ex.Append(goqu.I("jobs.status").Eq(*input.Filter.JobStatus))
		}

		if input.Filter.JobIDs != nil {
			ex = ex.Append(goqu.I("jobs.id").In(input.Filter.JobIDs))
		}

		if input.Filter.TagFilter != nil {
			if input.Filter.TagFilter.ExcludeUntaggedJobs != nil && *input.Filter.TagFilter.ExcludeUntaggedJobs {
				ex = ex.Append(goqu.L("jsonb_array_length(jobs.tags) > 0"))
			}
			if input.Filter.TagFilter.TagSuperset != nil {
				json, err := json.Marshal(input.Filter.TagFilter.TagSuperset)
				if err != nil {
					return nil, err
				}
				// This filter condition will only return jobs where the job tags are a subset of the tag
				// superset list specified in the filter
				ex = ex.Append(goqu.L(fmt.Sprintf("jobs.tags <@ '%s'", json)))
			}
		}
	}

	query := dialect.From(goqu.T("jobs")).
		Select(jobFieldList...).
		Where(ex)

	sortDirection := pagination.AscSort

	var sortBy *pagination.FieldDescriptor
	if input.Sort != nil {
		sortDirection = input.Sort.getSortDirection()
		sortBy = input.Sort.getFieldDescriptor()
	}

	qBuilder, err := pagination.NewPaginatedQueryBuilder(
		input.PaginationOptions,
		&pagination.FieldDescriptor{Key: "id", Table: "jobs", Col: "id"},
		pagination.WithSortByField(sortBy, sortDirection),
	)

	if err != nil {
		tracing.RecordError(span, err, "failed to build query")
		return nil, err
	}

	rows, err := qBuilder.Execute(ctx, j.dbClient.getConnection(ctx), query)
	if err != nil {
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	defer rows.Close()

	// Scan rows
	results := []models.Job{}
	for rows.Next() {
		item, err := scanJob(rows)
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

	result := JobsResult{
		PageInfo: rows.GetPageInfo(),
		Jobs:     results,
	}

	return &result, nil
}

func (j *jobs) UpdateJob(ctx context.Context, job *models.Job) (*models.Job, error) {
	ctx, span := tracer.Start(ctx, "db.UpdateJob")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	tags, err := json.Marshal(job.Tags)
	if err != nil {
		return nil, err
	}

	timestamp := currentTime()

	sql, args, err := dialect.Update("jobs").
		Prepared(true).
		Set(
			goqu.Record{
				"version":             goqu.L("? + ?", goqu.C("version"), 1),
				"updated_at":          timestamp,
				"status":              job.Status,
				"type":                job.Type,
				"workspace_id":        job.WorkspaceID,
				"run_id":              job.RunID,
				"cancel_requested":    job.CancelRequested,
				"cancel_requested_at": job.CancelRequestedTimestamp,
				"queued_at":           job.Timestamps.QueuedTimestamp,
				"pending_at":          job.Timestamps.PendingTimestamp,
				"running_at":          job.Timestamps.RunningTimestamp,
				"finished_at":         job.Timestamps.FinishedTimestamp,
				"runner_id":           job.RunnerID,
				"runner_path":         job.RunnerPath,
				"tags":                tags,
			},
		).Where(goqu.Ex{"id": job.Metadata.ID, "version": job.Metadata.Version}).Returning(jobFieldList...).ToSQL()

	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	updatedJob, err := scanJob(j.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))

	if err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return nil, ErrOptimisticLockError
		}
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	return updatedJob, nil
}

func (j *jobs) CreateJob(ctx context.Context, job *models.Job) (*models.Job, error) {
	ctx, span := tracer.Start(ctx, "db.CreateJob")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	tags, err := json.Marshal(job.Tags)
	if err != nil {
		return nil, err
	}

	timestamp := currentTime()

	sql, args, err := dialect.Insert("jobs").
		Prepared(true).
		Rows(goqu.Record{
			"id":                  newResourceID(),
			"version":             initialResourceVersion,
			"created_at":          timestamp,
			"updated_at":          timestamp,
			"status":              job.Status,
			"type":                job.Type,
			"workspace_id":        job.WorkspaceID,
			"run_id":              job.RunID,
			"cancel_requested":    job.CancelRequested,
			"cancel_requested_at": job.CancelRequestedTimestamp,
			"queued_at":           job.Timestamps.QueuedTimestamp,
			"pending_at":          job.Timestamps.PendingTimestamp,
			"running_at":          job.Timestamps.RunningTimestamp,
			"finished_at":         job.Timestamps.FinishedTimestamp,
			"max_job_duration":    job.MaxJobDuration,
			"runner_id":           job.RunnerID,
			"runner_path":         job.RunnerPath,
			"tags":                tags,
		}).
		Returning(jobFieldList...).ToSQL()

	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	createdJob, err := scanJob(j.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))

	if err != nil {
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	return createdJob, nil
}

func (j *jobs) GetJobCountForRunner(ctx context.Context, runnerID string) (int, error) {
	ctx, span := tracer.Start(ctx, "db.GetJobCountForRunner")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	var count int
	query := dialect.From(goqu.T("jobs")).
		Prepared(true).
		Select(goqu.COUNT("*")).Where(goqu.Ex{
		"runner_id": runnerID,
		"status":    []string{string(models.JobPending), string(models.JobRunning)},
	})

	sql, args, err := query.ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return 0, err
	}

	err = j.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...).Scan(&count)
	if err != nil {
		tracing.RecordError(span, err, "failed to execute query")
		return 0, err
	}
	return count, nil
}

func (j *jobs) getJob(ctx context.Context, exp goqu.Ex) (*models.Job, error) {
	ctx, span := tracer.Start(ctx, "db.getJob")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	query := dialect.From(goqu.T("jobs")).
		Prepared(true).
		Select(jobFieldList...).
		Where(exp)

	sql, args, err := query.ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	job, err := scanJob(j.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	return job, nil
}

func scanJob(row scanner) (*models.Job, error) {
	var cancelRequestedAt sql.NullTime
	var queuedAt sql.NullTime
	var pendingAt sql.NullTime
	var runningAt sql.NullTime
	var finishedAt sql.NullTime

	job := &models.Job{}

	fields := []interface{}{
		&job.Metadata.ID,
		&job.Metadata.CreationTimestamp,
		&job.Metadata.LastUpdatedTimestamp,
		&job.Metadata.Version,
		&job.Status,
		&job.Type,
		&job.WorkspaceID,
		&job.RunID,
		&job.CancelRequested,
		&cancelRequestedAt,
		&job.RunnerID,
		&job.RunnerPath,
		&queuedAt,
		&pendingAt,
		&runningAt,
		&finishedAt,
		&job.MaxJobDuration,
		&job.Tags,
	}

	err := row.Scan(fields...)

	if err != nil {
		return nil, err
	}

	if cancelRequestedAt.Valid {
		job.CancelRequestedTimestamp = &cancelRequestedAt.Time
	}

	if queuedAt.Valid {
		job.Timestamps.QueuedTimestamp = &queuedAt.Time
	}

	if pendingAt.Valid {
		job.Timestamps.PendingTimestamp = &pendingAt.Time
	}

	if runningAt.Valid {
		job.Timestamps.RunningTimestamp = &runningAt.Time
	}

	if finishedAt.Valid {
		job.Timestamps.FinishedTimestamp = &finishedAt.Time
	}

	return job, nil
}
