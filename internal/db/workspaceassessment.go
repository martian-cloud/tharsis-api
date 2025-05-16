package db

//go:generate go tool mockery --name WorkspaceAssessments --inpackage --case underscore

import (
	"context"
	"fmt"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/jackc/pgx/v4"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// WorkspaceAssessments encapsulates the logic to access Tharsis workspaceAssessments from the database.
type WorkspaceAssessments interface {
	GetWorkspaceAssessmentByID(ctx context.Context, id string) (*models.WorkspaceAssessment, error)
	GetWorkspaceAssessmentByTRN(ctx context.Context, trn string) (*models.WorkspaceAssessment, error)
	GetWorkspaceAssessmentByWorkspaceID(ctx context.Context, workspaceID string) (*models.WorkspaceAssessment, error)
	GetWorkspaceAssessments(ctx context.Context, input *GetWorkspaceAssessmentsInput) (*WorkspaceAssessmentsResult, error)
	CreateWorkspaceAssessment(ctx context.Context, assessment *models.WorkspaceAssessment) (*models.WorkspaceAssessment, error)
	UpdateWorkspaceAssessment(ctx context.Context, assessment *models.WorkspaceAssessment) (*models.WorkspaceAssessment, error)
	DeleteWorkspaceAssessment(ctx context.Context, assessment *models.WorkspaceAssessment) error
}

// WorkspaceAssessmentSortableField represents the fields that a assessment can be sorted by
type WorkspaceAssessmentSortableField string

// WorkspaceAssessmentSortableField constants
const (
	WorkspaceAssessmentSortableFieldStartedAtAsc  WorkspaceAssessmentSortableField = "STARTED_AT_ASC"
	WorkspaceAssessmentSortableFieldStartedAtDesc WorkspaceAssessmentSortableField = "STARTED_AT_DESC"
	WorkspaceAssessmentSortableFieldUpdatedAtAsc  WorkspaceAssessmentSortableField = "UPDATED_AT_ASC"
	WorkspaceAssessmentSortableFieldUpdatedAtDesc WorkspaceAssessmentSortableField = "UPDATED_AT_DESC"
)

func (r WorkspaceAssessmentSortableField) getFieldDescriptor() *pagination.FieldDescriptor {
	switch r {
	case WorkspaceAssessmentSortableFieldStartedAtAsc, WorkspaceAssessmentSortableFieldStartedAtDesc:
		return &pagination.FieldDescriptor{Key: "started_at", Table: "workspace_assessments", Col: "started_at"}
	case WorkspaceAssessmentSortableFieldUpdatedAtAsc, WorkspaceAssessmentSortableFieldUpdatedAtDesc:
		return &pagination.FieldDescriptor{Key: "updated_at", Table: "workspace_assessments", Col: "updated_at"}
	default:
		return nil
	}
}

func (r WorkspaceAssessmentSortableField) getSortDirection() pagination.SortDirection {
	if strings.HasSuffix(string(r), "_DESC") {
		return pagination.DescSort
	}
	return pagination.AscSort
}

// WorkspaceAssessmentFilter contains the supported fields for filtering workspace assessment resources
type WorkspaceAssessmentFilter struct {
	WorkspaceIDs []string
	InProgress   *bool
}

// GetWorkspaceAssessmentsInput is the input for listing workspace assessments
type GetWorkspaceAssessmentsInput struct {
	// Sort specifies the field to sort on and direction
	Sort *WorkspaceAssessmentSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// Filter is used to filter the results
	Filter *WorkspaceAssessmentFilter
}

// WorkspaceAssessmentsResult contains the response data and page information
type WorkspaceAssessmentsResult struct {
	PageInfo             *pagination.PageInfo
	WorkspaceAssessments []models.WorkspaceAssessment
}

type workspaceAssessments struct {
	dbClient *Client
}

var workspaceAssessmentsFieldList = append(metadataFieldList, "workspace_id", "started_at", "completed_at", "has_drift", "requires_notification", "completed_run_id")

// NewWorkspaceAssessments returns an instance of the WorkspaceAssessments interface.
func NewWorkspaceAssessments(dbClient *Client) WorkspaceAssessments {
	return &workspaceAssessments{dbClient: dbClient}
}

func (r *workspaceAssessments) GetWorkspaceAssessmentByID(ctx context.Context, id string) (*models.WorkspaceAssessment, error) {
	ctx, span := tracer.Start(ctx, "db.GetWorkspaceAssessmentByID")
	defer span.End()

	return r.getWorkspaceAssessment(ctx, goqu.Ex{"workspace_assessments.id": id})
}

func (r *workspaceAssessments) GetWorkspaceAssessmentByTRN(ctx context.Context, trn string) (*models.WorkspaceAssessment, error) {
	ctx, span := tracer.Start(ctx, "db.GetWorkspaceAssessmentByTRN")
	defer span.End()

	path, err := types.WorkspaceAssessmentModelType.ResourcePathFromTRN(trn)
	if err != nil {
		return nil, err
	}

	lastSlashIndex := strings.LastIndex(path, "/")
	if lastSlashIndex == -1 {
		return nil, errors.New("a workspace assessment TRN must have the workspace path, and assessment GID separated by a forward slash",
			errors.WithErrorCode(errors.EInvalid),
			errors.WithSpan(span),
		)
	}

	return r.getWorkspaceAssessment(ctx, goqu.Ex{
		"workspace_assessments.id": gid.FromGlobalID(path[lastSlashIndex+1:]),
		"namespaces.path":          path[:lastSlashIndex],
	})
}

func (r *workspaceAssessments) GetWorkspaceAssessmentByWorkspaceID(ctx context.Context, workspaceID string) (*models.WorkspaceAssessment, error) {
	ctx, span := tracer.Start(ctx, "db.GetWorkspaceAssessmentByWorkspaceID")
	defer span.End()

	return r.getWorkspaceAssessment(ctx, goqu.Ex{"workspace_assessments.workspace_id": workspaceID})
}

func (r *workspaceAssessments) GetWorkspaceAssessments(ctx context.Context, input *GetWorkspaceAssessmentsInput) (*WorkspaceAssessmentsResult, error) {
	ctx, span := tracer.Start(ctx, "db.GetWorkspaceAssessments")
	defer span.End()

	ex := goqu.And()

	if input.Filter != nil {
		if input.Filter.WorkspaceIDs != nil {
			ex = ex.Append(goqu.I("workspace_assessments.workspace_id").In(input.Filter.WorkspaceIDs))
		}
		if input.Filter.InProgress != nil {
			if *input.Filter.InProgress {
				ex = ex.Append(goqu.I("workspace_assessments.completed_at").IsNull())
			} else {
				ex = ex.Append(goqu.I("workspace_assessments.completed_at").IsNotNull())
			}
		}
	}

	query := dialect.From(goqu.T("workspace_assessments")).
		Select(r.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.I("workspace_assessments.workspace_id").Eq(goqu.I("namespaces.workspace_id")))).
		Where(ex)

	sortDirection := pagination.AscSort

	var sortBy *pagination.FieldDescriptor
	if input.Sort != nil {
		sortDirection = input.Sort.getSortDirection()
		sortBy = input.Sort.getFieldDescriptor()
	}

	qBuilder, err := pagination.NewPaginatedQueryBuilder(
		input.PaginationOptions,
		&pagination.FieldDescriptor{Key: "id", Table: "workspace_assessments", Col: "id"},
		pagination.WithSortByField(sortBy, sortDirection),
	)

	if err != nil {
		tracing.RecordError(span, err, "failed to build query")
		return nil, err
	}

	rows, err := qBuilder.Execute(ctx, r.dbClient.getConnection(ctx), query)
	if err != nil {
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	defer rows.Close()

	// Scan rows
	results := []models.WorkspaceAssessment{}
	for rows.Next() {
		item, err := scanWorkspaceAssessment(rows)
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

	result := WorkspaceAssessmentsResult{
		PageInfo:             rows.GetPageInfo(),
		WorkspaceAssessments: results,
	}

	return &result, nil
}

func (r *workspaceAssessments) CreateWorkspaceAssessment(ctx context.Context, assessment *models.WorkspaceAssessment) (*models.WorkspaceAssessment, error) {
	ctx, span := tracer.Start(ctx, "db.CreateWorkspaceAssessment")
	defer span.End()

	timestamp := currentTime()

	sql, args, err := dialect.From("workspace_assessments").
		Prepared(true).
		With("workspace_assessments",
			dialect.Insert("workspace_assessments").
				Rows(goqu.Record{
					"id":                    newResourceID(),
					"version":               initialResourceVersion,
					"created_at":            timestamp,
					"updated_at":            timestamp,
					"workspace_id":          assessment.WorkspaceID,
					"completed_run_id":      assessment.RunID,
					"has_drift":             assessment.HasDrift,
					"requires_notification": assessment.RequiresNotification,
					"completed_at":          assessment.CompletedAtTimestamp,
					"started_at":            assessment.StartedAtTimestamp,
				}).Returning("*"),
		).Select(r.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.I("workspace_assessments.workspace_id").Eq(goqu.I("namespaces.workspace_id")))).
		ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	createdWorkspaceAssessment, err := scanWorkspaceAssessment(r.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				return nil, errors.New("assessment for workspace %s already exists", assessment.WorkspaceID, errors.WithErrorCode(errors.EConflict), errors.WithSpan(span))
			}
			if isForeignKeyViolation(pgErr) {
				return nil, errors.New("invalid workspace ID", errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
			}
		}
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	return createdWorkspaceAssessment, nil
}

func (r *workspaceAssessments) UpdateWorkspaceAssessment(ctx context.Context, assessment *models.WorkspaceAssessment) (*models.WorkspaceAssessment, error) {
	ctx, span := tracer.Start(ctx, "db.UpdateWorkspaceAssessment")
	defer span.End()

	timestamp := currentTime()

	sql, args, err := dialect.From("workspace_assessments").
		Prepared(true).
		With("workspace_assessments",
			dialect.Update("workspace_assessments").
				Set(
					goqu.Record{
						"version":               goqu.L("? + ?", goqu.C("version"), 1),
						"updated_at":            timestamp,
						"completed_run_id":      assessment.RunID,
						"has_drift":             assessment.HasDrift,
						"requires_notification": assessment.RequiresNotification,
						"completed_at":          assessment.CompletedAtTimestamp,
						"started_at":            assessment.StartedAtTimestamp,
					},
				).Where(goqu.Ex{"id": assessment.Metadata.ID, "version": assessment.Metadata.Version}).Returning("*"),
		).Select(r.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.I("workspace_assessments.workspace_id").Eq(goqu.I("namespaces.workspace_id")))).
		ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	updatedWorkspaceAssessment, err := scanWorkspaceAssessment(r.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return nil, ErrOptimisticLockError
		}
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	return updatedWorkspaceAssessment, nil
}

func (r *workspaceAssessments) DeleteWorkspaceAssessment(ctx context.Context, assessment *models.WorkspaceAssessment) error {
	ctx, span := tracer.Start(ctx, "db.DeleteWorkspaceAssessment")
	defer span.End()

	sql, args, err := dialect.From("workspace_assessments").
		Prepared(true).
		With("workspace_assessments",
			dialect.Delete("workspace_assessments").
				Where(
					goqu.Ex{
						"id":      assessment.Metadata.ID,
						"version": assessment.Metadata.Version,
					},
				).Returning("*"),
		).Select(r.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.I("workspace_assessments.workspace_id").Eq(goqu.I("namespaces.workspace_id")))).
		ToSQL()

	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return err
	}

	if _, err := scanWorkspaceAssessment(r.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...)); err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return ErrOptimisticLockError
		}

		tracing.RecordError(span, err, "failed to execute query")
		return err
	}

	return nil
}

func (r *workspaceAssessments) getWorkspaceAssessment(ctx context.Context, exp exp.Ex) (*models.WorkspaceAssessment, error) {
	ctx, span := tracer.Start(ctx, "db.getWorkspaceAssessment")
	defer span.End()

	sql, args, err := dialect.From("workspace_assessments").
		Prepared(true).
		Select(r.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.I("workspace_assessments.workspace_id").Eq(goqu.I("namespaces.workspace_id")))).
		Where(exp).
		ToSQL()

	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	assessment, err := scanWorkspaceAssessment(r.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}

		if pgErr := asPgError(err); pgErr != nil {
			if isInvalidIDViolation(pgErr) {
				return nil, ErrInvalidID
			}
		}

		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	return assessment, nil
}

func (r *workspaceAssessments) getSelectFields() []interface{} {
	selectFields := []interface{}{}
	for _, field := range workspaceAssessmentsFieldList {
		selectFields = append(selectFields, fmt.Sprintf("workspace_assessments.%s", field))
	}

	selectFields = append(selectFields, "namespaces.path")

	return selectFields
}

func scanWorkspaceAssessment(row scanner) (*models.WorkspaceAssessment, error) {
	var workspacePath string
	wa := &models.WorkspaceAssessment{}

	fields := []interface{}{
		&wa.Metadata.ID,
		&wa.Metadata.CreationTimestamp,
		&wa.Metadata.LastUpdatedTimestamp,
		&wa.Metadata.Version,
		&wa.WorkspaceID,
		&wa.StartedAtTimestamp,
		&wa.CompletedAtTimestamp,
		&wa.HasDrift,
		&wa.RequiresNotification,
		&wa.RunID,
		&workspacePath,
	}

	if err := row.Scan(fields...); err != nil {
		return nil, err
	}

	wa.Metadata.TRN = types.WorkspaceAssessmentModelType.BuildTRN(workspacePath, wa.GetGlobalID())

	return wa, nil
}
