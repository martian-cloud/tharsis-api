package db

//go:generate go tool mockery --name TerraformModules --inpackage --case underscore

import (
	"context"
	"fmt"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v4"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// TerraformModules encapsulates the logic to access terraform modules from the database
type TerraformModules interface {
	GetModuleByID(ctx context.Context, id string) (*models.TerraformModule, error)
	GetModuleByTRN(ctx context.Context, trn string) (*models.TerraformModule, error)
	GetModules(ctx context.Context, input *GetModulesInput) (*ModulesResult, error)
	CreateModule(ctx context.Context, module *models.TerraformModule) (*models.TerraformModule, error)
	UpdateModule(ctx context.Context, module *models.TerraformModule) (*models.TerraformModule, error)
	DeleteModule(ctx context.Context, module *models.TerraformModule) error
}

// TerraformModuleSortableField represents the fields that a module can be sorted by
type TerraformModuleSortableField string

// TerraformModuleSortableField constants
const (
	TerraformModuleSortableFieldNameAsc             TerraformModuleSortableField = "NAME_ASC"
	TerraformModuleSortableFieldNameDesc            TerraformModuleSortableField = "NAME_DESC"
	TerraformModuleSortableFieldUpdatedAtAsc        TerraformModuleSortableField = "UPDATED_AT_ASC"
	TerraformModuleSortableFieldUpdatedAtDesc       TerraformModuleSortableField = "UPDATED_AT_DESC"
	TerraformModuleSortableFieldFieldGroupLevelAsc  TerraformModuleSortableField = "GROUP_LEVEL_ASC"
	TerraformModuleSortableFieldFieldGroupLevelDesc TerraformModuleSortableField = "GROUP_LEVEL_DESC"
)

func (ts TerraformModuleSortableField) getFieldDescriptor() *pagination.FieldDescriptor {
	switch ts {
	case TerraformModuleSortableFieldNameAsc, TerraformModuleSortableFieldNameDesc:
		return &pagination.FieldDescriptor{Key: "name", Table: "terraform_modules", Col: "name"}
	case TerraformModuleSortableFieldUpdatedAtAsc, TerraformModuleSortableFieldUpdatedAtDesc:
		return &pagination.FieldDescriptor{Key: "updated_at", Table: "terraform_modules", Col: "updated_at"}
	case TerraformModuleSortableFieldFieldGroupLevelAsc, TerraformModuleSortableFieldFieldGroupLevelDesc:
		return &pagination.FieldDescriptor{Key: "group_path", Table: "namespaces", Col: "path"}
	default:
		return nil
	}
}

func (ts TerraformModuleSortableField) getSortDirection() pagination.SortDirection {
	if strings.HasSuffix(string(ts), "_DESC") {
		return pagination.DescSort
	}
	return pagination.AscSort
}

func (ts TerraformModuleSortableField) getTransformFunc() pagination.SortTransformFunc {
	switch ts {
	case TerraformModuleSortableFieldFieldGroupLevelAsc, TerraformModuleSortableFieldFieldGroupLevelDesc:
		return func(s string) string {
			return fmt.Sprintf("array_length(string_to_array(%s, '/'), 1)", s)
		}
	default:
		return nil
	}
}

// TerraformModuleFilter contains the supported fields for filtering TerraformModule resources
type TerraformModuleFilter struct {
	Search             *string
	Name               *string
	System             *string
	RootGroupID        *string
	GroupID            *string
	UserID             *string
	ServiceAccountID   *string
	TerraformModuleIDs []string
	NamespacePaths     []string
}

// GetModulesInput is the input for listing terraform modules
type GetModulesInput struct {
	// Sort specifies the field to sort on and direction
	Sort *TerraformModuleSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// Filter is used to filter the results
	Filter *TerraformModuleFilter
}

// ModulesResult contains the response data and page information
type ModulesResult struct {
	PageInfo *pagination.PageInfo
	Modules  []models.TerraformModule
}

type terraformModules struct {
	dbClient *Client
}

var moduleFieldList = append(metadataFieldList, "group_id", "root_group_id", "name", "system", "private", "repo_url", "created_by")

// NewTerraformModules returns an instance of the TerraformModules interface
func NewTerraformModules(dbClient *Client) TerraformModules {
	return &terraformModules{dbClient: dbClient}
}

func (t *terraformModules) GetModuleByID(ctx context.Context, id string) (*models.TerraformModule, error) {
	ctx, span := tracer.Start(ctx, "db.GetModuleByID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	return t.getModule(ctx, goqu.Ex{"terraform_modules.id": id})
}

func (t *terraformModules) GetModuleByTRN(ctx context.Context, trn string) (*models.TerraformModule, error) {
	ctx, span := tracer.Start(ctx, "db.GetModuleByTRN")
	defer span.End()

	path, err := types.TerraformModuleModelType.ResourcePathFromTRN(trn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse TRN", errors.WithSpan(span))
	}

	pathParts := strings.Split(path, "/")

	if len(pathParts) < 3 {
		return nil, errors.New("a Terraform module TRN must have the namespacePath, module name, and module system separated by a forward slash",
			errors.WithErrorCode(errors.EInvalid),
			errors.WithSpan(span),
		)
	}

	return t.getModule(ctx, goqu.Ex{
		"terraform_modules.system": pathParts[len(pathParts)-1],
		"terraform_modules.name":   pathParts[len(pathParts)-2],
		"namespaces.path":          strings.Join(pathParts[:len(pathParts)-2], "/"),
	})
}

func (t *terraformModules) GetModules(ctx context.Context, input *GetModulesInput) (*ModulesResult, error) {
	ctx, span := tracer.Start(ctx, "db.GetModules")
	defer span.End()

	query := dialect.From(goqu.T("terraform_modules")).
		Select(t.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"terraform_modules.group_id": goqu.I("namespaces.group_id")}))

	ex := goqu.And()

	if input.Filter != nil {
		if input.Filter.TerraformModuleIDs != nil {
			ex = ex.Append(goqu.I("terraform_modules.id").In(input.Filter.TerraformModuleIDs))
		}
		if input.Filter.Search != nil && *input.Filter.Search != "" {
			// Add join on root group ID so that we can include it in the search
			query = query.InnerJoin(goqu.T("namespaces").As("root_namespace"), goqu.On(goqu.Ex{"terraform_modules.root_group_id": goqu.I("root_namespace.group_id")}))

			searchExp := goqu.L("CONCAT(root_namespace.path, '/', terraform_modules.name, '/', terraform_modules.system)")
			ex = ex.Append(searchExp.ILike("%" + *input.Filter.Search + "%"))
		}
		if input.Filter.NamespacePaths != nil {
			ex = ex.Append(goqu.I("namespaces.path").In(input.Filter.NamespacePaths))
		}
		if input.Filter.GroupID != nil {
			ex = ex.Append(goqu.I("terraform_modules.group_id").Eq(*input.Filter.GroupID))
		}
		if input.Filter.RootGroupID != nil {
			ex = ex.Append(goqu.I("terraform_modules.root_group_id").Eq(*input.Filter.RootGroupID))
		}
		if input.Filter.Name != nil {
			ex = ex.Append(goqu.I("terraform_modules.name").Eq(*input.Filter.Name))
		}
		if input.Filter.System != nil {
			ex = ex.Append(goqu.I("terraform_modules.system").Eq(*input.Filter.System))
		}
		if input.Filter.UserID != nil {
			ex = ex.Append(
				goqu.Or(
					goqu.I("terraform_modules.private").Eq(false),
					namespaceMembershipExpressionBuilder{
						userID: input.Filter.UserID,
					}.build(),
				))
		}
		if input.Filter.ServiceAccountID != nil {
			ex = ex.Append(
				goqu.Or(
					goqu.I("terraform_modules.private").Eq(false),
					namespaceMembershipExpressionBuilder{
						serviceAccountID: input.Filter.ServiceAccountID,
					}.build(),
				))
		}
	}

	query = query.Where(ex)

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
		&pagination.FieldDescriptor{Key: "id", Table: "terraform_modules", Col: "id"},
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
	results := []models.TerraformModule{}
	for rows.Next() {
		item, err := scanTerraformModule(rows)
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

	result := ModulesResult{
		PageInfo: rows.GetPageInfo(),
		Modules:  results,
	}

	return &result, nil
}

func (t *terraformModules) CreateModule(ctx context.Context, module *models.TerraformModule) (*models.TerraformModule, error) {
	ctx, span := tracer.Start(ctx, "db.CreateModule")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	timestamp := currentTime()

	sql, args, err := dialect.From("terraform_modules").
		Prepared(true).
		With("terraform_modules",
			dialect.Insert("terraform_modules").
				Rows(goqu.Record{
					"id":            newResourceID(),
					"version":       initialResourceVersion,
					"created_at":    timestamp,
					"updated_at":    timestamp,
					"group_id":      module.GroupID,
					"root_group_id": module.RootGroupID,
					"name":          module.Name,
					"system":        module.System,
					"private":       module.Private,
					"repo_url":      module.RepositoryURL,
					"created_by":    module.CreatedBy,
				}).Returning("*"),
		).Select(t.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"terraform_modules.group_id": goqu.I("namespaces.group_id")})).
		ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	createdModule, err := scanTerraformModule(t.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				tracing.RecordError(span, nil,
					"terraform module with name %s and system %s already exists", module.Name, module.System)
				return nil, errors.New("terraform module with name %s and system %s already exists", module.Name, module.System, errors.WithErrorCode(errors.EConflict))
			}
		}
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	return createdModule, nil
}

func (t *terraformModules) UpdateModule(ctx context.Context, module *models.TerraformModule) (*models.TerraformModule, error) {
	ctx, span := tracer.Start(ctx, "db.UpdateModule")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	timestamp := currentTime()

	sql, args, err := dialect.From("terraform_modules").
		Prepared(true).
		With("terraform_modules",
			dialect.Update("terraform_modules").
				Set(
					goqu.Record{
						"version":    goqu.L("? + ?", goqu.C("version"), 1),
						"updated_at": timestamp,
						"private":    module.Private,
						"repo_url":   module.RepositoryURL,
					},
				).Where(goqu.Ex{"id": module.Metadata.ID, "version": module.Metadata.Version}).
				Returning("*"),
		).Select(t.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"terraform_modules.group_id": goqu.I("namespaces.group_id")})).
		ToSQL()

	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	updatedModule, err := scanTerraformModule(t.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))

	if err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return nil, ErrOptimisticLockError
		}
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				tracing.RecordError(span, nil,
					"terraform module with name %s and system %s already exists", module.Name, module.System)
				return nil, errors.New("terraform module with name %s and system %s already exists", module.Name, module.System, errors.WithErrorCode(errors.EConflict))
			}
		}
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	return updatedModule, nil
}

func (t *terraformModules) DeleteModule(ctx context.Context, module *models.TerraformModule) error {
	ctx, span := tracer.Start(ctx, "db.DeleteModule")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	sql, args, err := dialect.From("terraform_modules").
		Prepared(true).
		With("terraform_modules",
			dialect.Delete("terraform_modules").
				Where(
					goqu.Ex{
						"id":      module.Metadata.ID,
						"version": module.Metadata.Version,
					},
				).Returning("*"),
		).Select(t.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"terraform_modules.group_id": goqu.I("namespaces.group_id")})).
		ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return err
	}

	_, err = scanTerraformModule(t.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
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

func (t *terraformModules) getModule(ctx context.Context, exp goqu.Ex) (*models.TerraformModule, error) {
	query := dialect.From(goqu.T("terraform_modules")).
		Prepared(true).
		Select(t.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"terraform_modules.group_id": goqu.I("namespaces.group_id")})).
		Where(exp)

	sql, args, err := query.ToSQL()
	if err != nil {
		return nil, err
	}

	module, err := scanTerraformModule(t.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}

		if pgErr := asPgError(err); pgErr != nil {
			if isInvalidIDViolation(pgErr) {
				return nil, ErrInvalidID
			}
		}

		return nil, err
	}

	return module, nil
}

func (t *terraformModules) getSelectFields() []interface{} {
	selectFields := []interface{}{}
	for _, field := range moduleFieldList {
		selectFields = append(selectFields, fmt.Sprintf("terraform_modules.%s", field))
	}

	selectFields = append(selectFields, "namespaces.path")

	return selectFields
}

func scanTerraformModule(row scanner) (*models.TerraformModule, error) {
	var groupPath string
	module := &models.TerraformModule{}

	fields := []interface{}{
		&module.Metadata.ID,
		&module.Metadata.CreationTimestamp,
		&module.Metadata.LastUpdatedTimestamp,
		&module.Metadata.Version,
		&module.GroupID,
		&module.RootGroupID,
		&module.Name,
		&module.System,
		&module.Private,
		&module.RepositoryURL,
		&module.CreatedBy,
		&groupPath,
	}

	err := row.Scan(fields...)
	if err != nil {
		return nil, err
	}

	module.Metadata.TRN = types.TerraformModuleModelType.BuildTRN(groupPath, module.Name, module.System)

	return module, nil
}
