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
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
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

func (ts TerraformModuleVersionSortableField) getFieldDescriptor() *fieldDescriptor {
	switch ts {
	case TerraformModuleVersionSortableFieldUpdatedAtAsc, TerraformModuleVersionSortableFieldUpdatedAtDesc:
		return &fieldDescriptor{key: "updated_at", table: "terraform_module_versions", col: "updated_at"}
	case TerraformModuleVersionSortableFieldCreatedAtAsc, TerraformModuleVersionSortableFieldCreatedAtDesc:
		return &fieldDescriptor{key: "created_at", table: "terraform_module_versions", col: "created_at"}
	default:
		return nil
	}
}

func (ts TerraformModuleVersionSortableField) getSortDirection() SortDirection {
	if strings.HasSuffix(string(ts), "_DESC") {
		return DescSort
	}
	return AscSort
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
	PaginationOptions *PaginationOptions
	// Filter is used to filter the results
	Filter *TerraformModuleVersionFilter
}

// ModuleVersionsResult contains the response data and page information
type ModuleVersionsResult struct {
	PageInfo       *PageInfo
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
	return t.getModuleVersion(ctx, goqu.Ex{"terraform_module_versions.id": id})
}

func (t *terraformModuleVersions) GetModuleVersions(ctx context.Context, input *GetModuleVersionsInput) (*ModuleVersionsResult, error) {
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

	query := dialect.From(goqu.T("terraform_module_versions")).Select(moduleVersionFieldList...).Where(ex)

	sortDirection := AscSort

	var sortBy *fieldDescriptor
	if input.Sort != nil {
		sortDirection = input.Sort.getSortDirection()
		sortBy = input.Sort.getFieldDescriptor()
	}

	qBuilder, err := newPaginatedQueryBuilder(
		input.PaginationOptions,
		&fieldDescriptor{key: "id", table: "terraform_module_versions", col: "id"},
		sortBy,
		sortDirection,
		moduleVersionFieldResolver,
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
	results := []models.TerraformModuleVersion{}
	for rows.Next() {
		item, err := scanTerraformModuleVersion(rows)
		if err != nil {
			return nil, err
		}

		results = append(results, *item)
	}

	if err := rows.finalize(&results); err != nil {
		return nil, err
	}

	result := ModuleVersionsResult{
		PageInfo:       rows.getPageInfo(),
		ModuleVersions: results,
	}

	return &result, nil
}

func (t *terraformModuleVersions) CreateModuleVersion(ctx context.Context, moduleVersion *models.TerraformModuleVersion) (*models.TerraformModuleVersion, error) {
	timestamp := currentTime()

	submodules, err := json.Marshal(moduleVersion.Submodules)
	if err != nil {
		return nil, err
	}

	examples, err := json.Marshal(moduleVersion.Examples)
	if err != nil {
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
		return nil, err
	}

	createdTerraformModuleVersion, err := scanTerraformModuleVersion(t.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				switch pgErr.ConstraintName {
				case "index_terraform_module_versions_on_latest":
					return nil, errors.NewError(errors.EConflict, "another terraform module version is already marked as the latest for the same module")
				case "index_terraform_module_versions_on_semantic_version":
					return nil, errors.NewError(errors.EConflict, fmt.Sprintf("terraform module version %s already exists", moduleVersion.SemanticVersion))
				default:
					return nil, errors.NewError(errors.EConflict, fmt.Sprintf("database constraint violated: %s", pgErr.ConstraintName))
				}
			}
		}
		return nil, err
	}

	return createdTerraformModuleVersion, nil
}

func (t *terraformModuleVersions) UpdateModuleVersion(ctx context.Context, moduleVersion *models.TerraformModuleVersion) (*models.TerraformModuleVersion, error) {
	timestamp := currentTime()

	submodules, err := json.Marshal(moduleVersion.Submodules)
	if err != nil {
		return nil, err
	}

	examples, err := json.Marshal(moduleVersion.Examples)
	if err != nil {
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
		return nil, err
	}

	updatedTerraformModuleVersion, err := scanTerraformModuleVersion(t.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrOptimisticLockError
		}
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				switch pgErr.ConstraintName {
				case "index_terraform_module_versions_on_latest":
					return nil, errors.NewError(errors.EConflict, "another terraform module version is already marked as the latest for the same module")
				default:
					return nil, errors.NewError(errors.EConflict, fmt.Sprintf("database constraint violated: %s", pgErr.ConstraintName))
				}
			}
		}
		return nil, err
	}

	return updatedTerraformModuleVersion, nil
}

func (t *terraformModuleVersions) DeleteModuleVersion(ctx context.Context, moduleVersion *models.TerraformModuleVersion) error {

	sql, _, err := dialect.Delete("terraform_module_versions").Where(
		goqu.Ex{
			"id":      moduleVersion.Metadata.ID,
			"version": moduleVersion.Metadata.Version,
		},
	).Returning(moduleVersionFieldList...).ToSQL()
	if err != nil {
		return err
	}

	_, err = scanTerraformModuleVersion(t.dbClient.getConnection(ctx).QueryRow(ctx, sql))
	if err != nil {
		if err == pgx.ErrNoRows {
			return ErrOptimisticLockError
		}
		return err
	}

	return nil
}

func (t *terraformModuleVersions) getModuleVersion(ctx context.Context, exp goqu.Ex) (*models.TerraformModuleVersion, error) {
	query := dialect.From(goqu.T("terraform_module_versions")).
		Select(t.getSelectFields()...).Where(exp)

	sql, _, err := query.ToSQL()
	if err != nil {
		return nil, err
	}

	moduleVersion, err := scanTerraformModuleVersion(t.dbClient.getConnection(ctx).QueryRow(ctx, sql))
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

func moduleVersionFieldResolver(key string, model interface{}) (string, error) {
	moduleVersion, ok := model.(*models.TerraformModuleVersion)
	if !ok {
		return "", errors.NewError(errors.EInternal, fmt.Sprintf("Expected module version type, got %T", model))
	}

	val, ok := metadataFieldResolver(key, &moduleVersion.Metadata)
	if !ok {
		return "", errors.NewError(errors.EInternal, fmt.Sprintf("Invalid field key requested %s", key))
	}

	return val, nil
}