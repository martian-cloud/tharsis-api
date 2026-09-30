package db

//go:generate go tool mockery --name Jobs --inpackage --case underscore

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v5"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
)

// jobCleanupClaimLeaseDuration is how long a claimed job is skipped by other runners' cleanup pollers,
// so a stale claim from a crashed runner becomes reclaimable once the lease lapses.
const jobCleanupClaimLeaseDuration = 10 * time.Minute

// Jobs encapsulates the logic to access jobs from the database
type Jobs interface {
	GetJobByID(ctx context.Context, id string) (*models.Job, error)
	GetJobByTRN(ctx context.Context, trnValue string) (*models.Job, error)
	GetLatestJobByType(ctx context.Context, runID string, jobType models.JobType) (*models.Job, error)
	GetJobs(ctx context.Context, input *GetJobsInput) (*JobsResult, error)
	UpdateJob(ctx context.Context, job *models.Job) (*models.Job, error)
	CreateJob(ctx context.Context, job *models.Job) (*models.Job, error)
	GetJobCountForRunner(ctx context.Context, runnerID string) (int, error)
	ClaimJobsForCleanup(ctx context.Context, input *ClaimJobsForCleanupInput) ([]models.Job, error)
	MarkJobsCleanedUp(ctx context.Context, runnerID string, jobIDs []string) error
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
	RunID               *string
	WorkspaceID         *string
	RunnerID            *string
	PolicyCheckID       *string
	JobType             *models.JobType
	JobStatus           *models.JobStatus
	TagFilter           *JobTagFilter
	JobIDs              []string
	NamespacePathPrefix *string
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

// ClaimJobsForCleanupInput is the input for leasing final, dispatched, not-yet-cleaned jobs to a
// runner's cleanup poller with FOR UPDATE SKIP LOCKED.
type ClaimJobsForCleanupInput struct {
	// RunnerID scopes the claim to jobs dispatched by the given runner.
	RunnerID string
	// Limit bounds how many jobs are returned in one call.
	Limit uint
}

type jobs struct {
	dbClient *Client
}

var jobFieldList = append(metadataFieldList,
	"status",
	"type",
	"workspace_id",
	"run_id",
	"cancel_requested_at",
	"runner_id",
	"runner_path",
	"queued_at",
	"pending_at",
	"running_at",
	"finished_at",
	"max_job_duration",
	"force_canceled",
	"tags",
	"properties",
	"outdated_job_protocol_version",
	"job_data",
	"resource_usage_metrics",
	"resource_usage_limits",
	"dispatcher_data",
	"cleanup_claimed_at",
	"cleanup_completed_at",
)

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

func (j *jobs) GetJobByTRN(ctx context.Context, trnValue string) (*models.Job, error) {
	ctx, span := tracer.Start(ctx, "db.GetJobByTRN")
	defer span.End()

	parsed, err := trn.TypeJob.Parse(trnValue)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse TRN", errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
	}

	if !parsed.HasParent() {
		return nil, errors.New("a job TRN must have the workspace path and job GID separated by a forward slash",
			errors.WithErrorCode(errors.EInvalid),
			errors.WithSpan(span),
		)
	}

	return j.getJob(ctx, goqu.Ex{
		"jobs.id":         gid.FromGlobalID(parsed.BaseName()),
		"namespaces.path": parsed.ParentPath(),
	})
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
		return nil, errors.Wrap(err, "failed to get job", errors.WithSpan(span))
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

		if input.Filter.PolicyCheckID != nil {
			// A policy check's jobs (its OPA evaluation jobs, including retries) are matched on the
			// policy check ID stored in the job_data JSONB blob (OPAJobData.policyCheckID).
			ex = ex.Append(goqu.L("jobs.job_data->>'policyCheckID' = ?", *input.Filter.PolicyCheckID))
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
				ex = ex.Append(goqu.L("jobs.tags <@ ?::jsonb", string(json)))
			}
		}

		if input.Filter.NamespacePathPrefix != nil {
			ex = ex.Append(goqu.Or(
				goqu.I("namespaces.path").Eq(*input.Filter.NamespacePathPrefix),
				goqu.I("namespaces.path").Like(escapeLikePattern(*input.Filter.NamespacePathPrefix)+"/%"),
			))
		}
	}

	query := dialect.From(goqu.T("jobs")).
		Select(j.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"jobs.workspace_id": goqu.I("namespaces.workspace_id")})).
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
		pagination.WithQueryTag("jobs.GetJobs"),
	)

	if err != nil {
		return nil, errors.Wrap(err, "failed to build query", errors.WithSpan(span))
	}

	rows, err := qBuilder.Execute(ctx, j.dbClient.getConnection(ctx), query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	defer rows.Close()

	// Scan rows
	results := []models.Job{}
	for rows.Next() {
		item, err := scanJob(rows)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan row", errors.WithSpan(span))
		}

		results = append(results, *item)
	}

	if err := rows.Finalize(&results); err != nil {
		return nil, errors.Wrap(err, "failed to finalize rows", errors.WithSpan(span))
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

	var jobData []byte
	if job.OPAData != nil {
		jobData, err = json.Marshal(job.OPAData)
		if err != nil {
			return nil, err
		}
	}

	timestamp := currentTime()

	// A nil value marshals to nil ([]byte), which the driver writes as SQL NULL.
	resourceUsageMetricsJSON, err := marshalOptionalJSON(job.ResourceUsageMetrics)
	if err != nil {
		return nil, err
	}

	resourceUsageLimitsJSON, err := marshalOptionalJSON(job.ResourceUsageLimits)
	if err != nil {
		return nil, err
	}

	// Marshal dispatcher data to JSON; nil becomes SQL NULL.
	var dispatcherDataJSON []byte
	if job.DispatcherData != nil {
		var mErr error
		dispatcherDataJSON, mErr = json.Marshal(job.DispatcherData)
		if mErr != nil {
			return nil, mErr
		}
	}

	sql, args, err := toSQLWithTag("jobs.UpdateJob", dialect.From("jobs").
		Prepared(true).
		With("jobs",
			dialect.Update("jobs").
				Set(
					goqu.Record{
						"version":                       goqu.L("? + ?", goqu.C("version"), 1),
						"updated_at":                    timestamp,
						"status":                        job.GetStatus(),
						"type":                          job.Type,
						"workspace_id":                  job.WorkspaceID,
						"run_id":                        job.RunID,
						"cancel_requested_at":           job.CancelRequestedTimestamp,
						"queued_at":                     job.Timestamps.QueuedTimestamp,
						"pending_at":                    job.Timestamps.PendingTimestamp,
						"running_at":                    job.Timestamps.RunningTimestamp,
						"finished_at":                   job.Timestamps.FinishedTimestamp,
						"runner_id":                     job.RunnerID,
						"runner_path":                   job.RunnerPath,
						"force_canceled":                job.ForceCanceled,
						"outdated_job_protocol_version": job.OutdatedJobProtocolVersion,
						"tags":                          tags,
						"job_data":                      jobData,
						"resource_usage_metrics":        resourceUsageMetricsJSON,
						"resource_usage_limits":         resourceUsageLimitsJSON,
						"dispatcher_data":               dispatcherDataJSON,
						"cleanup_completed_at":          job.CleanupCompletedAt,
					},
				).Where(goqu.Ex{"id": job.Metadata.ID, "version": job.Metadata.Version}).
				Returning("*"),
		).Select(j.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"jobs.workspace_id": goqu.I("namespaces.workspace_id")})))

	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	updatedJob, err := scanJob(j.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrOptimisticLockError
		}
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
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

	properties, err := json.Marshal(job.Properties)
	if err != nil {
		return nil, err
	}

	var jobData []byte
	if job.OPAData != nil {
		jobData, err = json.Marshal(job.OPAData)
		if err != nil {
			return nil, err
		}
	}

	timestamp := currentTime()

	sql, args, err := toSQLWithTag("jobs.CreateJob", dialect.From("jobs").
		Prepared(true).
		With("jobs",
			dialect.Insert("jobs").
				Rows(goqu.Record{
					"id":                            newResourceID(),
					"version":                       initialResourceVersion,
					"created_at":                    timestamp,
					"updated_at":                    timestamp,
					"status":                        job.GetStatus(),
					"type":                          job.Type,
					"workspace_id":                  job.WorkspaceID,
					"run_id":                        job.RunID,
					"cancel_requested_at":           job.CancelRequestedTimestamp,
					"queued_at":                     job.Timestamps.QueuedTimestamp,
					"pending_at":                    job.Timestamps.PendingTimestamp,
					"running_at":                    job.Timestamps.RunningTimestamp,
					"finished_at":                   job.Timestamps.FinishedTimestamp,
					"max_job_duration":              job.MaxJobDuration,
					"runner_id":                     job.RunnerID,
					"runner_path":                   job.RunnerPath,
					"force_canceled":                job.ForceCanceled,
					"outdated_job_protocol_version": job.OutdatedJobProtocolVersion,
					"tags":                          tags,
					"properties":                    properties,
					"job_data":                      jobData,
				}).Returning("*"),
		).Select(j.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"jobs.workspace_id": goqu.I("namespaces.workspace_id")})))
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	createdJob, err := scanJob(j.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))

	if err != nil {
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
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

	sql, args, err := toSQLWithTag("jobs.GetJobCountForRunner", query)
	if err != nil {
		return 0, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	err = j.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...).Scan(&count)
	if err != nil {
		return 0, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}
	return count, nil
}

// ClaimJobsForCleanup leases up to input.Limit final, dispatched jobs that haven't been cleaned up,
// scoped to the given runner. It stamps cleanup_claimed_at and skips rows another runner's poller
// already holds (FOR UPDATE SKIP LOCKED), so two runners never reap the same job and a stale claim
// from a crashed runner is reclaimable once the lease lapses.
func (j *jobs) ClaimJobsForCleanup(ctx context.Context, input *ClaimJobsForCleanupInput) ([]models.Job, error) {
	ctx, span := tracer.Start(ctx, "db.ClaimJobsForCleanup")
	defer span.End()

	if input.Limit == 0 {
		return nil, nil
	}

	now := currentTime()

	finalStatuses := []models.JobStatus{
		models.JobFinished,
		models.JobFailed,
		models.JobCanceled,
	}

	claimable := dialect.From(goqu.T("jobs")).
		Select(goqu.I("jobs.id")).
		Where(goqu.And(
			goqu.I("jobs.runner_id").Eq(input.RunnerID),
			goqu.I("jobs.dispatcher_data").IsNotNull(),
			goqu.I("jobs.cleanup_completed_at").IsNull(),
			goqu.I("jobs.status").In(finalStatuses),
			goqu.Or(
				goqu.I("jobs.cleanup_claimed_at").IsNull(),
				goqu.I("jobs.cleanup_claimed_at").Lt(now.Add(-jobCleanupClaimLeaseDuration)),
			),
		)).
		Order(goqu.I("jobs.created_at").Asc()).
		Limit(input.Limit).
		ForUpdate(goqu.SkipLocked)

	sql, args, err := toSQLWithTag("jobs.ClaimJobsForCleanup", dialect.From("jobs").
		Prepared(true).
		With("claimable", claimable).
		With("jobs",
			dialect.Update("jobs").
				Set(goqu.Record{
					"version":            goqu.L("version + 1"),
					"updated_at":         now,
					"cleanup_claimed_at": now,
				}).
				From(goqu.T("claimable")).
				Where(goqu.I("jobs.id").Eq(goqu.I("claimable.id"))).
				Returning(goqu.T("jobs").All()),
		).Select(j.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"jobs.workspace_id": goqu.I("namespaces.workspace_id")})))
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	rows, err := j.dbClient.getConnection(ctx).Query(ctx, sql, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}
	defer rows.Close()

	results := []models.Job{}
	for rows.Next() {
		item, sErr := scanJob(rows)
		if sErr != nil {
			return nil, errors.Wrap(sErr, "failed to scan row", errors.WithSpan(span))
		}
		results = append(results, *item)
	}

	if err = rows.Err(); err != nil {
		return nil, errors.Wrap(err, "failed to iterate rows", errors.WithSpan(span))
	}

	return results, nil
}

// MarkJobsCleanedUp stamps cleanup_completed_at on the given jobs, scoped to the runner that owns them
// and skipping any already marked. The runner_id predicate ensures a runner can only complete its own jobs.
func (j *jobs) MarkJobsCleanedUp(ctx context.Context, runnerID string, jobIDs []string) error {
	ctx, span := tracer.Start(ctx, "db.MarkJobsCleanedUp")
	defer span.End()

	if len(jobIDs) == 0 {
		return nil
	}

	now := currentTime()

	sql, args, err := toSQLWithTag("jobs.MarkJobsCleanedUp", dialect.Update("jobs").
		Prepared(true).
		Set(goqu.Record{
			"version":              goqu.L("version + 1"),
			"updated_at":           now,
			"cleanup_completed_at": now,
		}).
		Where(goqu.And(
			goqu.I("jobs.id").In(jobIDs),
			goqu.I("jobs.runner_id").Eq(runnerID),
			goqu.I("jobs.cleanup_completed_at").IsNull(),
		)))
	if err != nil {
		return errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	if _, err := j.dbClient.getConnection(ctx).Exec(ctx, sql, args...); err != nil {
		return errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return nil
}

func (j *jobs) getJob(ctx context.Context, exp goqu.Ex) (*models.Job, error) {
	ctx, span := tracer.Start(ctx, "db.getJob")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	query := dialect.From(goqu.T("jobs")).
		Prepared(true).
		Select(j.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"jobs.workspace_id": goqu.I("namespaces.workspace_id")})).
		Where(exp)

	sql, args, err := toSQLWithTag("jobs.getJob", query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	job, err := scanJob(j.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))

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

	return job, nil
}

func (j *jobs) getSelectFields() []interface{} {
	selectFields := []interface{}{}
	for _, field := range jobFieldList {
		selectFields = append(selectFields, fmt.Sprintf("jobs.%s", field))
	}
	selectFields = append(selectFields, "namespaces.path")

	return selectFields
}

func scanJob(row scanner) (*models.Job, error) {
	var workspacePath string
	var status models.JobStatus
	var rawJobData []byte

	job := &models.Job{}

	fields := []any{
		&job.Metadata.ID,
		&job.Metadata.CreationTimestamp,
		&job.Metadata.LastUpdatedTimestamp,
		&job.Metadata.Version,
		&status,
		&job.Type,
		&job.WorkspaceID,
		&job.RunID,
		&job.CancelRequestedTimestamp,
		&job.RunnerID,
		&job.RunnerPath,
		&job.Timestamps.QueuedTimestamp,
		&job.Timestamps.PendingTimestamp,
		&job.Timestamps.RunningTimestamp,
		&job.Timestamps.FinishedTimestamp,
		&job.MaxJobDuration,
		&job.ForceCanceled,
		&job.Tags,
		&job.Properties,
		&job.OutdatedJobProtocolVersion,
		&rawJobData,
		&job.ResourceUsageMetrics,
		&job.ResourceUsageLimits,
		&job.DispatcherData,
		&job.CleanupClaimedAt,
		&job.CleanupCompletedAt,
		&workspacePath,
	}

	err := row.Scan(fields...)

	if err != nil {
		return nil, err
	}

	// Hydrate the status from the persisted value (allowed from the job's zero value).
	if err := job.SetStatus(status); err != nil {
		return nil, err
	}

	if rawJobData != nil {
		switch job.Type {
		case models.JobOPAType:
			job.OPAData = &models.OPAJobData{}
			if err := json.Unmarshal(rawJobData, job.OPAData); err != nil {
				return nil, err
			}
		}
	}

	job.Metadata.TRN = trn.TypeJob.Build(workspacePath, job.GetGlobalID())

	return job, nil
}
