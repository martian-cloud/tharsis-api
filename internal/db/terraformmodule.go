package db

//go:generate mockery --name TerraformModules --inpackage --case underscore

import (
	"context"
	"fmt"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v4"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// TerraformModules encapsulates the logic to access terraform modules from the database
type TerraformModules interface {
	GetModuleByID(ctx context.Context, id string) (*models.TerraformModule, error)
	GetModuleByPath(ctx context.Context, path string) (*models.TerraformModule, error)
	GetModules(ctx context.Context, input *GetModulesInput) (*ModulesResult, error)
	CreateModule(ctx context.Context, module *models.TerraformModule) (*models.TerraformModule, error)
	UpdateModule(ctx context.Context, module *models.TerraformModule) (*models.TerraformModule, error)
	DeleteModule(ctx context.Context, module *models.TerraformModule) error
}

// TerraformModuleSortableField represents the fields that a module can be sorted by
type TerraformModuleSortableField string

// TerraformModuleSortableField constants
const (
	TerraformModuleSortableFieldNameAsc       TerraformModuleSortableField = "NAME_ASC"
	TerraformModuleSortableFieldNameDesc      TerraformModuleSortableField = "NAME_DESC"
	TerraformModuleSortableFieldUpdatedAtAsc  TerraformModuleSortableField = "UPDATED_AT_ASC"
	TerraformModuleSortableFieldUpdatedAtDesc TerraformModuleSortableField = "UPDATED_AT_DESC"
)

func (ts TerraformModuleSortableField) getFieldDescriptor() *fieldDescriptor {
	switch ts {
	case TerraformModuleSortableFieldNameAsc, TerraformModuleSortableFieldNameDesc:
		return &fieldDescriptor{key: "name", table: "terraform_modules", col: "name"}
	case TerraformModuleSortableFieldUpdatedAtAsc, TerraformModuleSortableFieldUpdatedAtDesc:
		return &fieldDescriptor{key: "updated_at", table: "terraform_modules", col: "updated_at"}
	default:
		return nil
	}
}

func (ts TerraformModuleSortableField) getSortDirection() SortDirection {
	if strings.HasSuffix(string(ts), "_DESC") {
		return DescSort
	}
	return AscSort
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
}

// GetModulesInput is the input for listing terraform modules
type GetModulesInput struct {
	// Sort specifies the field to sort on and direction
	Sort *TerraformModuleSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *PaginationOptions
	// Filter is used to filter the results
	Filter *TerraformModuleFilter
}

// ModulesResult contains the response data and page information
type ModulesResult struct {
	PageInfo *PageInfo
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
	return t.getModule(ctx, goqu.Ex{"terraform_modules.id": id})
}

func (t *terraformModules) GetModuleByPath(ctx context.Context, path string) (*models.TerraformModule, error) {
	pathParts := strings.Split(path, "/")

	if len(pathParts) < 3 {
		return nil, errors.NewError(errors.EInvalid, "Invalid resource path for module")
	}

	moduleName := pathParts[len(pathParts)-2]
	moduleSystem := pathParts[len(pathParts)-1]
	namespacePath := strings.Join(pathParts[:len(pathParts)-2], "/")
	return t.getModule(ctx, goqu.Ex{"terraform_modules.name": moduleName, "terraform_modules.system": moduleSystem, "namespaces.path": namespacePath})
}

func (t *terraformModules) GetModules(ctx context.Context, input *GetModulesInput) (*ModulesResult, error) {
	ex := goqu.And()

	if input.Filter != nil {
		if input.Filter.TerraformModuleIDs != nil {
			ex = ex.Append(goqu.I("terraform_modules.id").In(input.Filter.TerraformModuleIDs))
		}
		if input.Filter.Search != nil && *input.Filter.Search != "" {
			search := *input.Filter.Search

			lastDelimiterIndex := strings.LastIndex(search, "/")

			if lastDelimiterIndex != -1 {
				// TODO: do we need to include system in the search?
				registryNamespace := search[:lastDelimiterIndex]
				moduleName := search[lastDelimiterIndex+1:]

				if moduleName != "" {
					// An AND condition is used here since the first part of the search is the registry namespace path
					// and the second part is the module name
					ex = ex.Append(
						goqu.And(
							goqu.I("namespaces.path").Like(registryNamespace+"%"),
							goqu.I("terraform_modules.name").Like(moduleName+"%"),
						),
					)
				} else {
					// We know the search is a namespace path since it ends with a "/"
					ex = ex.Append(goqu.I("namespaces.path").Like(registryNamespace + "%"))
				}
			} else {
				// We don't know if the search is for a namespace path or module name; therefore, use
				// an OR condition to search both
				ex = ex.Append(
					goqu.Or(
						goqu.I("namespaces.path").Like(search+"%"),
						goqu.I("terraform_modules.name").Like(search+"%"),
					),
				)
			}
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

	query := dialect.From(goqu.T("terraform_modules")).
		Select(t.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"terraform_modules.group_id": goqu.I("namespaces.group_id")})).
		Where(ex)

	sortDirection := AscSort

	var sortBy *fieldDescriptor
	if input.Sort != nil {
		sortDirection = input.Sort.getSortDirection()
		sortBy = input.Sort.getFieldDescriptor()
	}

	qBuilder, err := newPaginatedQueryBuilder(
		input.PaginationOptions,
		&fieldDescriptor{key: "id", table: "terraform_modules", col: "id"},
		sortBy,
		sortDirection,
		moduleFieldResolver,
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
	results := []models.TerraformModule{}
	for rows.Next() {
		item, err := scanTerraformModule(rows, true)
		if err != nil {
			return nil, err
		}

		results = append(results, *item)
	}

	if err := rows.finalize(&results); err != nil {
		return nil, err
	}

	result := ModulesResult{
		PageInfo: rows.getPageInfo(),
		Modules:  results,
	}

	return &result, nil
}

func (t *terraformModules) CreateModule(ctx context.Context, module *models.TerraformModule) (*models.TerraformModule, error) {
	timestamp := currentTime()

	tx, err := t.dbClient.getConnection(ctx).Begin(ctx)
	if err != nil {
		return nil, err
	}

	// Rollback is safe to call even if the tx is already closed, so if
	// the tx commits successfully, this is a no-op
	defer func() {
		if txErr := tx.Rollback(ctx); txErr != nil && txErr != pgx.ErrTxClosed {
			t.dbClient.logger.Errorf("failed to rollback tx: %v", txErr)
		}
	}()

	sql, args, err := dialect.Insert("terraform_modules").
		Prepared(true).
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
		}).
		Returning(moduleFieldList...).ToSQL()
	if err != nil {
		return nil, err
	}

	createdModule, err := scanTerraformModule(tx.QueryRow(ctx, sql, args...), false)
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				return nil, errors.NewError(errors.EConflict, fmt.Sprintf("terraform module with name %s and system %s already exists", module.Name, module.System))
			}
		}
		return nil, err
	}

	// Lookup namespace for group
	namespace, err := getNamespaceByGroupID(ctx, tx, module.GroupID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	createdModule.ResourcePath = buildTerraformModuleResourcePath(namespace.path, module.Name, module.System)

	return createdModule, nil
}

func (t *terraformModules) UpdateModule(ctx context.Context, module *models.TerraformModule) (*models.TerraformModule, error) {
	timestamp := currentTime()

	tx, err := t.dbClient.getConnection(ctx).Begin(ctx)
	if err != nil {
		return nil, err
	}

	// Rollback is safe to call even if the tx is already closed, so if
	// the tx commits successfully, this is a no-op
	defer func() {
		if txErr := tx.Rollback(ctx); txErr != nil && txErr != pgx.ErrTxClosed {
			t.dbClient.logger.Errorf("failed to rollback tx: %v", txErr)
		}
	}()

	sql, args, err := dialect.Update("terraform_modules").
		Prepared(true).
		Set(
			goqu.Record{
				"version":    goqu.L("? + ?", goqu.C("version"), 1),
				"updated_at": timestamp,
				"name":       module.Name,
				"private":    module.Private,
				"repo_url":   module.RepositoryURL,
			},
		).Where(goqu.Ex{"id": module.Metadata.ID, "version": module.Metadata.Version}).Returning(moduleFieldList...).ToSQL()

	if err != nil {
		return nil, err
	}

	updatedModule, err := scanTerraformModule(tx.QueryRow(ctx, sql, args...), false)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrOptimisticLockError
		}
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				return nil, errors.NewError(errors.EConflict, fmt.Sprintf("terraform module with name %s and system %s already exists", module.Name, module.System))
			}
		}
		return nil, err
	}

	// Lookup namespace for group
	namespace, err := getNamespaceByGroupID(ctx, tx, module.GroupID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	updatedModule.ResourcePath = buildTerraformModuleResourcePath(namespace.path, module.Name, module.System)

	return updatedModule, nil
}

func (t *terraformModules) DeleteModule(ctx context.Context, module *models.TerraformModule) error {

	sql, args, err := dialect.Delete("terraform_modules").
		Prepared(true).
		Where(
			goqu.Ex{
				"id":      module.Metadata.ID,
				"version": module.Metadata.Version,
			},
		).Returning(moduleFieldList...).ToSQL()
	if err != nil {
		return err
	}

	_, err = scanTerraformModule(t.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...), false)
	if err != nil {
		if err == pgx.ErrNoRows {
			return ErrOptimisticLockError
		}
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

	module, err := scanTerraformModule(t.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...), true)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
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

func buildTerraformModuleResourcePath(groupPath string, name string, system string) string {
	return fmt.Sprintf("%s/%s/%s", groupPath, name, system)
}

func scanTerraformModule(row scanner, withResourcePath bool) (*models.TerraformModule, error) {
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
	}

	var path string
	if withResourcePath {
		fields = append(fields, &path)
	}

	err := row.Scan(fields...)
	if err != nil {
		return nil, err
	}

	if withResourcePath {
		module.ResourcePath = buildTerraformModuleResourcePath(path, module.Name, module.System)
	}

	return module, nil
}

func moduleFieldResolver(key string, model interface{}) (string, error) {
	module, ok := model.(*models.TerraformModule)
	if !ok {
		return "", errors.NewError(errors.EInternal, fmt.Sprintf("Expected module type, got %T", model))
	}

	val, ok := metadataFieldResolver(key, &module.Metadata)
	if !ok {
		switch key {
		case "name":
			val = module.Name
		default:
			return "", errors.NewError(errors.EInternal, fmt.Sprintf("Invalid field key requested %s", key))
		}
	}

	return val, nil
}
