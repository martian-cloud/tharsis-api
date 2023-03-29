package db

//go:generate mockery --name Runners --inpackage --case underscore

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v4"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
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
	RunnerSortableFieldUpdatedAtAsc  RunnerSortableField = "UPDATED_AT_ASC"
	RunnerSortableFieldUpdatedAtDesc RunnerSortableField = "UPDATED_AT_DESC"
)

func (ts RunnerSortableField) getFieldDescriptor() *fieldDescriptor {
	switch ts {
	case RunnerSortableFieldUpdatedAtAsc, RunnerSortableFieldUpdatedAtDesc:
		return &fieldDescriptor{key: "updated_at", table: "runners", col: "updated_at"}
	default:
		return nil
	}
}

func (ts RunnerSortableField) getSortDirection() SortDirection {
	if strings.HasSuffix(string(ts), "_DESC") {
		return DescSort
	}
	return AscSort
}

// RunnerFilter contains the supported fields for filtering Runner resources
type RunnerFilter struct {
	GroupID        *string
	RunnerName     *string
	RunnerID       *string
	RunnerIDs      []string
	NamespacePaths []string
}

// GetRunnersInput is the input for listing runners
type GetRunnersInput struct {
	// Sort specifies the field to sort on and direction
	Sort *RunnerSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *PaginationOptions
	// Filter is used to filter the results
	Filter *RunnerFilter
}

// RunnersResult contains the response data and page information
type RunnersResult struct {
	PageInfo *PageInfo
	Runners  []models.Runner
}

type terraformRunners struct {
	dbClient *Client
}

var runnerFieldList = append(metadataFieldList, "type", "name", "description", "group_id", "created_by")

// NewRunners returns an instance of the Runners interface
func NewRunners(dbClient *Client) Runners {
	return &terraformRunners{dbClient: dbClient}
}

func (t *terraformRunners) GetRunnerByPath(ctx context.Context, path string) (*models.Runner, error) {
	parts := strings.Split(path, "/")
	name := parts[len(parts)-1]

	if len(parts) > 1 {
		namespace := strings.Join(parts[:len(parts)-1], "/")
		return t.getRunner(ctx, goqu.Ex{"runners.name": name, "namespaces.path": namespace})
	}

	return t.getRunner(ctx, goqu.Ex{"runners.name": name})
}

func (t *terraformRunners) GetRunnerByID(ctx context.Context, id string) (*models.Runner, error) {
	return t.getRunner(ctx, goqu.Ex{"runners.id": id})
}

func (t *terraformRunners) GetRunners(ctx context.Context, input *GetRunnersInput) (*RunnersResult, error) {
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
	}

	query := dialect.From(goqu.T("runners")).
		Select(t.getSelectFields()...).
		LeftJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"runners.group_id": goqu.I("namespaces.group_id")})).
		Where(ex)

	sortDirection := AscSort

	var sortBy *fieldDescriptor
	if input.Sort != nil {
		sortDirection = input.Sort.getSortDirection()
		sortBy = input.Sort.getFieldDescriptor()
	}

	qBuilder, err := newPaginatedQueryBuilder(
		input.PaginationOptions,
		&fieldDescriptor{key: "id", table: "runners", col: "id"},
		sortBy,
		sortDirection,
		runnerFieldResolver,
	)

	if err != nil {
		return nil, err
	}

	rows, err := qBuilder.execute(ctx, t.dbClient.getConnection(ctx), query)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	// Scan rows
	results := []models.Runner{}
	for rows.Next() {
		item, err := scanRunner(rows, true)
		if err != nil {
			return nil, err
		}

		results = append(results, *item)
	}

	if err := rows.finalize(&results); err != nil {
		return nil, err
	}

	result := RunnersResult{
		PageInfo: rows.getPageInfo(),
		Runners:  results,
	}

	return &result, nil
}

func (t *terraformRunners) CreateRunner(ctx context.Context, runner *models.Runner) (*models.Runner, error) {
	timestamp := currentTime()

	tx, err := t.dbClient.getConnection(ctx).Begin(ctx)
	if err != nil {
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
			"id":          newResourceID(),
			"version":     initialResourceVersion,
			"created_at":  timestamp,
			"updated_at":  timestamp,
			"type":        runner.Type,
			"group_id":    runner.GroupID,
			"name":        runner.Name,
			"description": runner.Description,
			"created_by":  runner.CreatedBy,
		}).
		Returning(runnerFieldList...).ToSQL()
	if err != nil {
		return nil, err
	}

	createdRunner, err := scanRunner(tx.QueryRow(ctx, sql, args...), false)
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				return nil, errors.NewError(
					errors.EConflict,
					fmt.Sprintf("runner with name %s already exists in group", runner.Name),
				)
			}
		}
		return nil, err
	}

	if createdRunner.GroupID != nil {
		// Lookup namespace for group
		namespace, err := getNamespaceByGroupID(ctx, tx, *createdRunner.GroupID)
		if err != nil {
			return nil, err
		}
		createdRunner.ResourcePath = buildGroupRunnerResourcePath(namespace.path, createdRunner.Name)
	} else {
		createdRunner.ResourcePath = createdRunner.Name
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return createdRunner, nil
}

func (t *terraformRunners) UpdateRunner(ctx context.Context, runner *models.Runner) (*models.Runner, error) {
	timestamp := currentTime()

	tx, err := t.dbClient.getConnection(ctx).Begin(ctx)
	if err != nil {
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
			"version":     goqu.L("? + ?", goqu.C("version"), 1),
			"updated_at":  timestamp,
			"description": runner.Description,
		}).
		Where(goqu.Ex{"id": runner.Metadata.ID, "version": runner.Metadata.Version}).
		Returning(runnerFieldList...).ToSQL()
	if err != nil {
		return nil, err
	}

	updatedRunner, err := scanRunner(tx.QueryRow(ctx, sql, args...), false)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrOptimisticLockError
		}
		return nil, err
	}

	if updatedRunner.GroupID != nil {
		// Lookup namespace for group
		namespace, err := getNamespaceByGroupID(ctx, tx, *updatedRunner.GroupID)
		if err != nil {
			return nil, err
		}
		updatedRunner.ResourcePath = buildGroupRunnerResourcePath(namespace.path, updatedRunner.Name)
	} else {
		updatedRunner.ResourcePath = updatedRunner.Name
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return updatedRunner, nil
}

func (t *terraformRunners) DeleteRunner(ctx context.Context, runner *models.Runner) error {

	sql, args, err := dialect.Delete("runners").
		Prepared(true).
		Where(
			goqu.Ex{
				"id":      runner.Metadata.ID,
				"version": runner.Metadata.Version,
			},
		).Returning(runnerFieldList...).ToSQL()
	if err != nil {
		return err
	}

	_, err = scanRunner(t.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...), false)
	if err != nil {
		if err == pgx.ErrNoRows {
			return ErrOptimisticLockError
		}
		return err
	}

	return nil
}

func (t *terraformRunners) getRunner(ctx context.Context, exp goqu.Ex) (*models.Runner, error) {
	query := dialect.From(goqu.T("runners")).
		Prepared(true).
		Select(t.getSelectFields()...).
		LeftJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"runners.group_id": goqu.I("namespaces.group_id")})).
		Where(exp)

	sql, args, err := query.ToSQL()
	if err != nil {
		return nil, err
	}

	runner, err := scanRunner(t.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...), true)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
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

func runnerFieldResolver(key string, model interface{}) (string, error) {
	runner, ok := model.(*models.Runner)
	if !ok {
		return "", errors.NewError(errors.EInternal, fmt.Sprintf("Expected runner type, got %T", model))
	}

	val, ok := metadataFieldResolver(key, &runner.Metadata)
	if !ok {
		return "", errors.NewError(errors.EInternal, fmt.Sprintf("Invalid field key requested %s", key))
	}

	return val, nil
}
