package db

//go:generate go tool mockery --name Runners --inpackage --case underscore

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v4"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// Runners encapsulates the logic to access runners from the database
type Runners interface {
	GetRunnerByPath(ctx context.Context, path string) (*models.Runner, error)
	GetRunnerByID(ctx context.Context, id string) (*models.Runner, error)
	GetRunners(ctx context.Context, input *GetRunnersInput) (*RunnersResult, error)
	CreateRunner(ctx context.Context, runner *models.Runner) (*models.Runner, error)
	UpdateRunner(ctx context.Context, runner *models.Runner) (*models.Runner, error)
	DeleteRunner(ctx context.Context, runner *models.Runner) error
}

// RunnerSortableField represents the fields that a runners can be sorted by
type RunnerSortableField string

// RunnerSortableField constants
const (
	RunnerSortableFieldUpdatedAtAsc   RunnerSortableField = "UPDATED_AT_ASC"
	RunnerSortableFieldUpdatedAtDesc  RunnerSortableField = "UPDATED_AT_DESC"
	RunnerSortableFieldGroupLevelAsc  RunnerSortableField = "GROUP_LEVEL_ASC"
	RunnerSortableFieldGroupLevelDesc RunnerSortableField = "GROUP_LEVEL_DESC"
)

func (ts RunnerSortableField) getFieldDescriptor() *pagination.FieldDescriptor {
	switch ts {
	case RunnerSortableFieldUpdatedAtAsc, RunnerSortableFieldUpdatedAtDesc:
		return &pagination.FieldDescriptor{Key: "updated_at", Table: "runners", Col: "updated_at"}
	case RunnerSortableFieldGroupLevelAsc, RunnerSortableFieldGroupLevelDesc:
		return &pagination.FieldDescriptor{Key: "group_path", Table: "namespaces", Col: "path"}
	default:
		return nil
	}
}

func (ts RunnerSortableField) getSortDirection() pagination.SortDirection {
	if strings.HasSuffix(string(ts), "_DESC") {
		return pagination.DescSort
	}
	return pagination.AscSort
}

func (ts RunnerSortableField) getTransformFunc() pagination.SortTransformFunc {
	switch ts {
	case RunnerSortableFieldGroupLevelAsc, RunnerSortableFieldGroupLevelDesc:
		return func(s string) string {
			return fmt.Sprintf("array_length(string_to_array(%s, '/'), 1)", s)
		}
	default:
		return nil
	}
}

// RunnerTagFilter is a filter condition for runner tags
type RunnerTagFilter struct {
	RunUntaggedJobs *bool
	TagSubset       []string
}

// RunnerFilter contains the supported fields for filtering Runner resources
type RunnerFilter struct {
	GroupID        *string
	RunnerName     *string
	RunnerID       *string
	Enabled        *bool
	RunnerType     *models.RunnerType
	RunnerIDs      []string
	NamespacePaths []string
	TagFilter      *RunnerTagFilter
}

// GetRunnersInput is the input for listing runners
type GetRunnersInput struct {
	// Sort specifies the field to sort on and direction
	Sort *RunnerSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// Filter is used to filter the results
	Filter *RunnerFilter
}

// RunnersResult contains the response data and page information
type RunnersResult struct {
	PageInfo *pagination.PageInfo
	Runners  []models.Runner
}

type terraformRunners struct {
	dbClient *Client
}

var runnerFieldList = append(metadataFieldList,
	"type", "name", "description", "group_id", "created_by", "disabled",
	"tags", "run_untagged_jobs",
)

// NewRunners returns an instance of the Runners interface
func NewRunners(dbClient *Client) Runners {
	return &terraformRunners{dbClient: dbClient}
}

func (t *terraformRunners) GetRunnerByPath(ctx context.Context, path string) (*models.Runner, error) {
	ctx, span := tracer.Start(ctx, "db.GetRunnerByPath")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	parts := strings.Split(path, "/")
	name := parts[len(parts)-1]

	if len(parts) > 1 {
		namespace := strings.Join(parts[:len(parts)-1], "/")
		return t.getRunner(ctx, goqu.Ex{"runners.name": name, "namespaces.path": namespace})
	}

	return t.getRunner(ctx, goqu.Ex{"runners.name": name})
}

func (t *terraformRunners) GetRunnerByID(ctx context.Context, id string) (*models.Runner, error) {
	ctx, span := tracer.Start(ctx, "db.GetRunnerByID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	return t.getRunner(ctx, goqu.Ex{"runners.id": id})
}

func (t *terraformRunners) GetRunners(ctx context.Context, input *GetRunnersInput) (*RunnersResult, error) {
	ctx, span := tracer.Start(ctx, "db.GetRunners")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	ex := goqu.And()

	if input.Filter != nil {
		if input.Filter.RunnerID != nil {
			ex = ex.Append(goqu.I("runners.id").Eq(*input.Filter.RunnerID))
		}

		if input.Filter.RunnerIDs != nil {
			ex = ex.Append(goqu.I("runners.id").In(input.Filter.RunnerIDs))
		}

		if input.Filter.NamespacePaths != nil {
			ex = ex.Append(goqu.I("namespaces.path").In(input.Filter.NamespacePaths))
		}

		if input.Filter.RunnerName != nil {
			ex = ex.Append(goqu.I("runners.name").Eq(*input.Filter.RunnerName))
		}

		if input.Filter.GroupID != nil {
			ex = ex.Append(goqu.I("runners.group_id").Eq(*input.Filter.GroupID))
		}

		if input.Filter.Enabled != nil {
			ex = ex.Append(goqu.I("runners.disabled").Eq(!(*input.Filter.Enabled)))
		}

		if input.Filter.RunnerType != nil {
			ex = ex.Append(goqu.I("runners.type").Eq(*input.Filter.RunnerType))
		}

		if input.Filter.TagFilter != nil {
			if input.Filter.TagFilter.RunUntaggedJobs != nil {
				ex = ex.Append(goqu.I("runners.run_untagged_jobs").Eq(*input.Filter.TagFilter.RunUntaggedJobs))
			}
			if input.Filter.TagFilter.TagSubset != nil {
				json, err := json.Marshal(input.Filter.TagFilter.TagSubset)
				if err != nil {
					return nil, err
				}
				// This filter condition will only return runners where the runner tags are a superset of the tag
				// subset list specified in the filter
				ex = ex.Append(goqu.L(fmt.Sprintf("runners.tags @> '%s'", json)))
			}
		}
	}

	query := dialect.From(goqu.T("runners")).
		Select(t.getSelectFields()...).
		LeftJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"runners.group_id": goqu.I("namespaces.group_id")})).
		Where(ex)

	sortDirection := pagination.AscSort

	var sortBy *pagination.FieldDescriptor
	var sortTransformFunc pagination.SortTransformFunc
	if input.Sort != nil {
		sortDirection = input.Sort.getSortDirection()
		sortBy = input.Sort.getFieldDescriptor()
		sortTransformFunc = input.Sort.getTransformFunc()
	}

	qBuilder, err := pagination.NewPaginatedQueryBuilder(
		input.PaginationOptions,
		&pagination.FieldDescriptor{Key: "id", Table: "runners", Col: "id"},
		pagination.WithSortByField(sortBy, sortDirection),
		pagination.WithSortByTransform(sortTransformFunc),
	)

	if err != nil {
		tracing.RecordError(span, err, "failed to build query")
		return nil, err
	}

	rows, err := qBuilder.Execute(ctx, t.dbClient.getConnection(ctx), query)
	if err != nil {
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	defer rows.Close()

	// Scan rows
	results := []models.Runner{}
	for rows.Next() {
		item, err := scanRunner(rows, true)
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

	result := RunnersResult{
		PageInfo: rows.GetPageInfo(),
		Runners:  results,
	}

	return &result, nil
}

func (t *terraformRunners) CreateRunner(ctx context.Context, runner *models.Runner) (*models.Runner, error) {
	ctx, span := tracer.Start(ctx, "db.CreateRunner")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	tags, err := json.Marshal(runner.Tags)
	if err != nil {
		return nil, err
	}

	timestamp := currentTime()

	tx, err := t.dbClient.getConnection(ctx).Begin(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return nil, err
	}

	// Rollback is safe to call even if the tx is already closed, so if
	// the tx commits successfully, this is a no-op
	defer func() {
		if txErr := tx.Rollback(ctx); txErr != nil && txErr != pgx.ErrTxClosed {
			t.dbClient.logger.Errorf("failed to rollback tx for CreateRunner: %v", txErr)
		}
	}()

	sql, args, err := dialect.Insert("runners").
		Prepared(true).
		Rows(goqu.Record{
			"id":                newResourceID(),
			"version":           initialResourceVersion,
			"created_at":        timestamp,
			"updated_at":        timestamp,
			"type":              runner.Type,
			"group_id":          runner.GroupID,
			"name":              runner.Name,
			"description":       runner.Description,
			"created_by":        runner.CreatedBy,
			"disabled":          runner.Disabled,
			"tags":              tags,
			"run_untagged_jobs": runner.RunUntaggedJobs,
		}).
		Returning(runnerFieldList...).ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	createdRunner, err := scanRunner(tx.QueryRow(ctx, sql, args...), false)
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				tracing.RecordError(span, nil, "runner with name %s already exists in group", runner.Name)
				return nil, errors.New(
					"runner with name %s already exists in group", runner.Name,
					errors.WithErrorCode(errors.EConflict),
				)
			}
		}
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	if createdRunner.GroupID != nil {
		// Lookup namespace for group
		namespace, err := getNamespaceByGroupID(ctx, tx, *createdRunner.GroupID)
		if err != nil {
			tracing.RecordError(span, err, "failed to get namespace by group ID")
			return nil, err
		}
		createdRunner.ResourcePath = buildGroupRunnerResourcePath(namespace.path, createdRunner.Name)
	} else {
		createdRunner.ResourcePath = createdRunner.Name
	}

	if err := tx.Commit(ctx); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return nil, err
	}

	return createdRunner, nil
}

func (t *terraformRunners) UpdateRunner(ctx context.Context, runner *models.Runner) (*models.Runner, error) {
	ctx, span := tracer.Start(ctx, "db.UpdateRunner")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	tags, err := json.Marshal(runner.Tags)
	if err != nil {
		return nil, err
	}

	timestamp := currentTime()

	tx, err := t.dbClient.getConnection(ctx).Begin(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return nil, err
	}

	// Rollback is safe to call even if the tx is already closed, so if
	// the tx commits successfully, this is a no-op
	defer func() {
		if txErr := tx.Rollback(ctx); txErr != nil && txErr != pgx.ErrTxClosed {
			t.dbClient.logger.Errorf("failed to rollback tx for UpdateRunner: %v", txErr)
		}
	}()

	sql, args, err := dialect.Update("runners").
		Prepared(true).
		Set(goqu.Record{
			"version":           goqu.L("? + ?", goqu.C("version"), 1),
			"updated_at":        timestamp,
			"description":       runner.Description,
			"disabled":          runner.Disabled,
			"tags":              tags,
			"run_untagged_jobs": runner.RunUntaggedJobs,
		}).
		Where(goqu.Ex{"id": runner.Metadata.ID, "version": runner.Metadata.Version}).
		Returning(runnerFieldList...).ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	updatedRunner, err := scanRunner(tx.QueryRow(ctx, sql, args...), false)
	if err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return nil, ErrOptimisticLockError
		}
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	if updatedRunner.GroupID != nil {
		// Lookup namespace for group
		namespace, err := getNamespaceByGroupID(ctx, tx, *updatedRunner.GroupID)
		if err != nil {
			tracing.RecordError(span, err, "failed to get namespace by group ID")
			return nil, err
		}
		updatedRunner.ResourcePath = buildGroupRunnerResourcePath(namespace.path, updatedRunner.Name)
	} else {
		updatedRunner.ResourcePath = updatedRunner.Name
	}

	if err := tx.Commit(ctx); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return nil, err
	}

	return updatedRunner, nil
}

func (t *terraformRunners) DeleteRunner(ctx context.Context, runner *models.Runner) error {
	ctx, span := tracer.Start(ctx, "db.DeleteRunner")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	sql, args, err := dialect.Delete("runners").
		Prepared(true).
		Where(
			goqu.Ex{
				"id":      runner.Metadata.ID,
				"version": runner.Metadata.Version,
			},
		).Returning(runnerFieldList...).ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return err
	}

	_, err = scanRunner(t.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...), false)
	if err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return ErrOptimisticLockError
		}
		tracing.RecordError(span, err, "failed to execute query")
		return err
	}

	return nil
}

func (t *terraformRunners) getRunner(ctx context.Context, exp goqu.Ex) (*models.Runner, error) {
	ctx, span := tracer.Start(ctx, "db.getRunner")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	query := dialect.From(goqu.T("runners")).
		Prepared(true).
		Select(t.getSelectFields()...).
		LeftJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"runners.group_id": goqu.I("namespaces.group_id")})).
		Where(exp)

	sql, args, err := query.ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	runner, err := scanRunner(t.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...), true)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	return runner, nil
}

func (t *terraformRunners) getSelectFields() []interface{} {
	selectFields := []interface{}{}
	for _, field := range runnerFieldList {
		selectFields = append(selectFields, fmt.Sprintf("runners.%s", field))
	}

	selectFields = append(selectFields, "namespaces.path")

	return selectFields
}

func buildGroupRunnerResourcePath(groupPath string, name string) string {
	return fmt.Sprintf("%s/%s", groupPath, name)
}

func scanRunner(row scanner, withResourcePath bool) (*models.Runner, error) {
	runner := &models.Runner{}

	fields := []interface{}{
		&runner.Metadata.ID,
		&runner.Metadata.CreationTimestamp,
		&runner.Metadata.LastUpdatedTimestamp,
		&runner.Metadata.Version,
		&runner.Type,
		&runner.Name,
		&runner.Description,
		&runner.GroupID,
		&runner.CreatedBy,
		&runner.Disabled,
		&runner.Tags,
		&runner.RunUntaggedJobs,
	}
	var path sql.NullString
	if withResourcePath {
		fields = append(fields, &path)
	}

	err := row.Scan(fields...)
	if err != nil {
		return nil, err
	}

	if withResourcePath {
		if path.Valid {
			runner.ResourcePath = buildGroupRunnerResourcePath(path.String, runner.Name)
		} else {
			runner.ResourcePath = runner.Name
		}
	}

	return runner, nil
}
