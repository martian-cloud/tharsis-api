package db

//go:generate mockery --name TerraformModuleVersions --inpackage --case underscore

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

// TerraformModuleVersions encapsulates the logic to access terraform module versions from the database
type TerraformModuleVersions interface {
	GetModuleVersionByID(ctx context.Context, id string) (*models.TerraformModuleVersion, error)
	GetModuleVersions(ctx context.Context, input *GetModuleVersionsInput) (*ModuleVersionsResult, error)
	CreateModuleVersion(ctx context.Context, moduleVersion *models.TerraformModuleVersion) (*models.TerraformModuleVersion, error)
	UpdateModuleVersion(ctx context.Context, moduleVersion *models.TerraformModuleVersion) (*models.TerraformModuleVersion, error)
	DeleteModuleVersion(ctx context.Context, moduleVersion *models.TerraformModuleVersion) error
}

// TerraformModuleVersionSortableField represents the fields that a module version can be sorted by
type TerraformModuleVersionSortableField string

// TerraformModuleVersionSortableField constants
const (
	TerraformModuleVersionSortableFieldUpdatedAtAsc  TerraformModuleVersionSortableField = "UPDATED_AT_ASC"
	TerraformModuleVersionSortableFieldUpdatedAtDesc TerraformModuleVersionSortableField = "UPDATED_AT_DESC"
	TerraformModuleVersionSortableFieldCreatedAtAsc  TerraformModuleVersionSortableField = "CREATED_AT_ASC"
	TerraformModuleVersionSortableFieldCreatedAtDesc TerraformModuleVersionSortableField = "CREATED_AT_DESC"
)

func (ts TerraformModuleVersionSortableField) getFieldDescriptor() *pagination.FieldDescriptor {
	switch ts {
	case TerraformModuleVersionSortableFieldUpdatedAtAsc, TerraformModuleVersionSortableFieldUpdatedAtDesc:
		return &pagination.FieldDescriptor{Key: "updated_at", Table: "terraform_module_versions", Col: "updated_at"}
	case TerraformModuleVersionSortableFieldCreatedAtAsc, TerraformModuleVersionSortableFieldCreatedAtDesc:
		return &pagination.FieldDescriptor{Key: "created_at", Table: "terraform_module_versions", Col: "created_at"}
	default:
		return nil
	}
}

func (ts TerraformModuleVersionSortableField) getSortDirection() pagination.SortDirection {
	if strings.HasSuffix(string(ts), "_DESC") {
		return pagination.DescSort
	}
	return pagination.AscSort
}

// TerraformModuleVersionFilter contains the supported fields for filtering TerraformModuleVersion resources
type TerraformModuleVersionFilter struct {
	ModuleID         *string
	Status           *models.TerraformModuleVersionStatus
	SemanticVersion  *string
	SHASum           *string
	Latest           *bool
	ModuleVersionIDs []string
}

// GetModuleVersionsInput is the input for listing terraform module versions
type GetModuleVersionsInput struct {
	// Sort specifies the field to sort on and direction
	Sort *TerraformModuleVersionSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// Filter is used to filter the results
	Filter *TerraformModuleVersionFilter
}

// ModuleVersionsResult contains the response data and page information
type ModuleVersionsResult struct {
	PageInfo       *pagination.PageInfo
	ModuleVersions []models.TerraformModuleVersion
}

type terraformModuleVersions struct {
	dbClient *Client
}

var moduleVersionFieldList = append(
	metadataFieldList,
	"module_id",
	"semantic_version",
	"sha_sum",
	"status",
	"error",
	"diagnostics",
	"upload_started_at",
	"submodules",
	"examples",
	"latest",
	"created_by",
)

// NewTerraformModuleVersions returns an instance of the TerraformModuleVersions interface
func NewTerraformModuleVersions(dbClient *Client) TerraformModuleVersions {
	return &terraformModuleVersions{dbClient: dbClient}
}

func (t *terraformModuleVersions) GetModuleVersionByID(ctx context.Context, id string) (*models.TerraformModuleVersion, error) {
	ctx, span := tracer.Start(ctx, "db.GetModuleVersionByID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	return t.getModuleVersion(ctx, goqu.Ex{"terraform_module_versions.id": id})
}

func (t *terraformModuleVersions) GetModuleVersions(ctx context.Context, input *GetModuleVersionsInput) (*ModuleVersionsResult, error) {
	ctx, span := tracer.Start(ctx, "db.GetModuleVersions")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	ex := goqu.Ex{}

	if input.Filter != nil {
		if input.Filter.ModuleVersionIDs != nil {
			ex["terraform_module_versions.id"] = input.Filter.ModuleVersionIDs
		}
		if input.Filter.ModuleID != nil {
			ex["terraform_module_versions.module_id"] = *input.Filter.ModuleID
		}
		if input.Filter.Status != nil {
			ex["terraform_module_versions.status"] = string(*input.Filter.Status)
		}
		if input.Filter.SHASum != nil {
			ex["terraform_module_versions.sha_sum"] = *input.Filter.SHASum
		}
		if input.Filter.SemanticVersion != nil {
			ex["terraform_module_versions.semantic_version"] = *input.Filter.SemanticVersion
		}
		if input.Filter.Latest != nil {
			ex["terraform_module_versions.latest"] = *input.Filter.Latest
		}
	}

	query := dialect.From(goqu.T("terraform_module_versions")).
		Select(moduleVersionFieldList...).
		Where(ex)

	sortDirection := pagination.AscSort

	var sortBy *pagination.FieldDescriptor
	if input.Sort != nil {
		sortDirection = input.Sort.getSortDirection()
		sortBy = input.Sort.getFieldDescriptor()
	}

	qBuilder, err := pagination.NewPaginatedQueryBuilder(
		input.PaginationOptions,
		&pagination.FieldDescriptor{Key: "id", Table: "terraform_module_versions", Col: "id"},
		sortBy,
		sortDirection,
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
	results := []models.TerraformModuleVersion{}
	for rows.Next() {
		item, err := scanTerraformModuleVersion(rows)
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

	result := ModuleVersionsResult{
		PageInfo:       rows.GetPageInfo(),
		ModuleVersions: results,
	}

	return &result, nil
}

func (t *terraformModuleVersions) CreateModuleVersion(ctx context.Context, moduleVersion *models.TerraformModuleVersion) (*models.TerraformModuleVersion, error) {
	ctx, span := tracer.Start(ctx, "db.CreateModuleVersion")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	timestamp := currentTime()

	submodules, err := json.Marshal(moduleVersion.Submodules)
	if err != nil {
		tracing.RecordError(span, err, "failed to marshal module version submodules")
		return nil, err
	}

	examples, err := json.Marshal(moduleVersion.Examples)
	if err != nil {
		tracing.RecordError(span, err, "failed to marshal module version examples")
		return nil, err
	}

	record := goqu.Record{
		"id":                newResourceID(),
		"version":           initialResourceVersion,
		"created_at":        timestamp,
		"updated_at":        timestamp,
		"module_id":         moduleVersion.ModuleID,
		"semantic_version":  moduleVersion.SemanticVersion,
		"sha_sum":           moduleVersion.SHASum,
		"status":            moduleVersion.Status,
		"error":             nullableString(moduleVersion.Error),
		"diagnostics":       nullableString(moduleVersion.Diagnostics),
		"upload_started_at": moduleVersion.UploadStartedTimestamp,
		"submodules":        submodules,
		"examples":          examples,
		"created_by":        moduleVersion.CreatedBy,
		"latest":            moduleVersion.Latest,
	}

	sql, args, err := dialect.Insert("terraform_module_versions").
		Prepared(true).
		Rows(record).
		Returning(moduleVersionFieldList...).
		ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	createdTerraformModuleVersion, err := scanTerraformModuleVersion(t.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				switch pgErr.ConstraintName {
				case "index_terraform_module_versions_on_latest":
					tracing.RecordError(span, nil,
						"another terraform module version is already marked as the latest for the same module")
					return nil, errors.New(errors.EConflict, "another terraform module version is already marked as the latest for the same module")
				case "index_terraform_module_versions_on_semantic_version":
					tracing.RecordError(span, nil,
						"terraform module version %s already exists", moduleVersion.SemanticVersion)
					return nil, errors.New(errors.EConflict, "terraform module version %s already exists", moduleVersion.SemanticVersion)
				default:
					tracing.RecordError(span, nil,
						"database constraint violated: %s", pgErr.ConstraintName)
					return nil, errors.New(errors.EConflict, "database constraint violated: %s", pgErr.ConstraintName)
				}
			}
		}
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	return createdTerraformModuleVersion, nil
}

func (t *terraformModuleVersions) UpdateModuleVersion(ctx context.Context, moduleVersion *models.TerraformModuleVersion) (*models.TerraformModuleVersion, error) {
	ctx, span := tracer.Start(ctx, "db.UpdateModuleVersion")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	timestamp := currentTime()

	submodules, err := json.Marshal(moduleVersion.Submodules)
	if err != nil {
		tracing.RecordError(span, err, "failed to marshal module version submodules")
		return nil, err
	}

	examples, err := json.Marshal(moduleVersion.Examples)
	if err != nil {
		tracing.RecordError(span, err, "failed to marshal module version examples")
		return nil, err
	}

	record := goqu.Record{
		"version":           goqu.L("? + ?", goqu.C("version"), 1),
		"updated_at":        timestamp,
		"sha_sum":           moduleVersion.SHASum,
		"status":            moduleVersion.Status,
		"error":             nullableString(moduleVersion.Error),
		"diagnostics":       nullableString(moduleVersion.Diagnostics),
		"upload_started_at": moduleVersion.UploadStartedTimestamp,
		"submodules":        submodules,
		"examples":          examples,
		"latest":            moduleVersion.Latest,
	}

	sql, args, err := dialect.Update("terraform_module_versions").
		Prepared(true).
		Set(record).
		Where(goqu.Ex{"id": moduleVersion.Metadata.ID, "version": moduleVersion.Metadata.Version}).Returning(moduleVersionFieldList...).ToSQL()

	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	updatedTerraformModuleVersion, err := scanTerraformModuleVersion(t.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))

	if err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return nil, ErrOptimisticLockError
		}
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				switch pgErr.ConstraintName {
				case "index_terraform_module_versions_on_latest":
					tracing.RecordError(span, nil,
						"another terraform module version is already marked as the latest for the same module")
					return nil, errors.New(errors.EConflict, "another terraform module version is already marked as the latest for the same module")
				default:
					tracing.RecordError(span, nil,
						"database constraint violated: %s", pgErr.ConstraintName)
					return nil, errors.New(errors.EConflict, "database constraint violated: %s", pgErr.ConstraintName)
				}
			}
		}
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	return updatedTerraformModuleVersion, nil
}

func (t *terraformModuleVersions) DeleteModuleVersion(ctx context.Context, moduleVersion *models.TerraformModuleVersion) error {
	ctx, span := tracer.Start(ctx, "db.DeleteModuleVersion")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	sql, args, err := dialect.Delete("terraform_module_versions").
		Prepared(true).
		Where(
			goqu.Ex{
				"id":      moduleVersion.Metadata.ID,
				"version": moduleVersion.Metadata.Version,
			},
		).Returning(moduleVersionFieldList...).ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return err
	}

	_, err = scanTerraformModuleVersion(t.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
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

func (t *terraformModuleVersions) getModuleVersion(ctx context.Context, exp goqu.Ex) (*models.TerraformModuleVersion, error) {
	query := dialect.From(goqu.T("terraform_module_versions")).
		Prepared(true).
		Select(t.getSelectFields()...).
		Where(exp)

	sql, args, err := query.ToSQL()
	if err != nil {
		return nil, err
	}

	moduleVersion, err := scanTerraformModuleVersion(t.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return moduleVersion, nil
}

func (t *terraformModuleVersions) getSelectFields() []interface{} {
	selectFields := []interface{}{}
	for _, field := range moduleVersionFieldList {
		selectFields = append(selectFields, fmt.Sprintf("terraform_module_versions.%s", field))
	}

	return selectFields
}

func scanTerraformModuleVersion(row scanner) (*models.TerraformModuleVersion, error) {
	moduleVersion := &models.TerraformModuleVersion{}

	moduleVersion.Submodules = []string{}
	moduleVersion.Examples = []string{}
	var errorMessage, diagnostics sql.NullString
	var uploadStartedAt sql.NullTime

	fields := []interface{}{
		&moduleVersion.Metadata.ID,
		&moduleVersion.Metadata.CreationTimestamp,
		&moduleVersion.Metadata.LastUpdatedTimestamp,
		&moduleVersion.Metadata.Version,
		&moduleVersion.ModuleID,
		&moduleVersion.SemanticVersion,
		&moduleVersion.SHASum,
		&moduleVersion.Status,
		&errorMessage,
		&diagnostics,
		&uploadStartedAt,
		&moduleVersion.Submodules,
		&moduleVersion.Examples,
		&moduleVersion.Latest,
		&moduleVersion.CreatedBy,
	}

	err := row.Scan(fields...)
	if err != nil {
		return nil, err
	}

	if errorMessage.Valid {
		moduleVersion.Error = errorMessage.String
	}

	if diagnostics.Valid {
		moduleVersion.Diagnostics = diagnostics.String
	}

	if uploadStartedAt.Valid {
		moduleVersion.UploadStartedTimestamp = &uploadStartedAt.Time
	}

	return moduleVersion, nil
}
