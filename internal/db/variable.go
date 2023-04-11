package db

//go:generate mockery --name Variables --inpackage --case underscore

import (
	"context"
	"fmt"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v4"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// Variables encapsulates the logic to access variables from the database
type Variables interface {
	GetVariables(ctx context.Context, input *GetVariablesInput) (*VariableResult, error)
	GetVariableByID(ctx context.Context, id string) (*models.Variable, error)
	CreateVariable(ctx context.Context, input *models.Variable) (*models.Variable, error)
	CreateVariables(ctx context.Context, namespacePath string, variables []models.Variable) error
	UpdateVariable(ctx context.Context, variable *models.Variable) (*models.Variable, error)
	DeleteVariable(ctx context.Context, variable *models.Variable) error
	DeleteVariables(ctx context.Context, namespacePath string, category models.VariableCategory) error
}

// VariableSortableField represents the fields that a variable can be sorted by
type VariableSortableField string

// VariableSortableField constants
const (
	VariableSortableFieldKeyAsc            VariableSortableField = "KEY_ASC"
	VariableSortableFieldKeyDesc           VariableSortableField = "KEY_DESC"
	VariableSortableFieldCreatedAtAsc      VariableSortableField = "CREATED_AT_ASC"
	VariableSortableFieldCreatedAtDesc     VariableSortableField = "CREATED_AT_DESC"
	VariableSortableFieldNamespacePathAsc  VariableSortableField = "NAMESPACE_PATH_ASC"
	VariableSortableFieldNamespacePathDesc VariableSortableField = "NAMESPACE_PATH_DESC"
)

func (sf VariableSortableField) getFieldDescriptor() *pagination.FieldDescriptor {
	switch sf {
	case VariableSortableFieldKeyAsc, VariableSortableFieldKeyDesc:
		return &pagination.FieldDescriptor{Key: "key", Table: "namespace_variables", Col: "key"}
	case VariableSortableFieldCreatedAtAsc, VariableSortableFieldCreatedAtDesc:
		return &pagination.FieldDescriptor{Key: "created_at", Table: "namespace_variables", Col: "created_at"}
	case VariableSortableFieldNamespacePathAsc, VariableSortableFieldNamespacePathDesc:
		return &pagination.FieldDescriptor{Key: "namespace_path", Table: "namespaces", Col: "path"}
	default:
		return nil
	}
}

func (sf VariableSortableField) getSortDirection() pagination.SortDirection {
	if strings.HasSuffix(string(sf), "_DESC") {
		return pagination.DescSort
	}
	return pagination.AscSort
}

// VariableFilter contains the supported fields for filtering Variable resources
type VariableFilter struct {
	NamespacePaths []string
	VariableIDs    []string
}

// GetVariablesInput is the input for listing variables
type GetVariablesInput struct {
	// Sort specifies the field to sort on and direction
	Sort *VariableSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// Filter is used to filter the results
	Filter *VariableFilter
}

// VariableResult contains the response data and page information
type VariableResult struct {
	PageInfo  *pagination.PageInfo
	Variables []models.Variable
}

type variables struct {
	dbClient *Client
}

var variableFieldList = append(metadataFieldList, "key", "value", "category", "hcl")

// NewVariables returns an instance of the Variables interface
func NewVariables(dbClient *Client) Variables {
	return &variables{dbClient: dbClient}
}

func (m *variables) GetVariableByID(ctx context.Context, id string) (*models.Variable, error) {
	sql, _, err := dialect.From("namespace_variables").
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"namespace_variables.namespace_id": goqu.I("namespaces.id")})).
		Select(m.getSelectFields()...).
		Where(goqu.Ex{"namespace_variables.id": id}).ToSQL()

	if err != nil {
		return nil, err
	}

	variable, err := scanVariable(m.dbClient.getConnection(ctx).QueryRow(ctx, sql), true)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return variable, nil
}

func (m *variables) CreateVariable(ctx context.Context, input *models.Variable) (*models.Variable, error) {
	namespace, err := getNamespaceByPath(ctx, m.dbClient.getConnection(ctx), input.NamespacePath)
	if err != nil {
		return nil, err
	}

	if namespace == nil {
		return nil, errors.New(errors.ENotFound, "Namespace not found")
	}

	timestamp := currentTime()

	record := goqu.Record{
		"id":           newResourceID(),
		"version":      initialResourceVersion,
		"created_at":   timestamp,
		"updated_at":   timestamp,
		"namespace_id": namespace.id,
		"key":          input.Key,
		"value":        input.Value,
		"category":     input.Category,
		"hcl":          input.Hcl,
	}

	sql, args, err := dialect.Insert("namespace_variables").
		Prepared(true).
		Rows(record).
		Returning(variableFieldList...).ToSQL()

	if err != nil {
		return nil, err
	}

	createdVariable, err := scanVariable(m.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...), false)

	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				return nil, errors.New(
					errors.EConflict,
					"Variable with key %s in namespace %s already exists", input.Key, input.NamespacePath,
				)
			}
			if isForeignKeyViolation(pgErr) {
				switch pgErr.ConstraintName {
				case "fk_namespace_variables_namespace_id":
					return nil, errors.New(errors.ENotFound, "namespace does not exist")
				}
			}
		}
		return nil, err
	}

	createdVariable.NamespacePath = input.NamespacePath

	return createdVariable, nil
}

func (m *variables) CreateVariables(ctx context.Context, namespacePath string, variables []models.Variable) error {
	namespace, err := getNamespaceByPath(ctx, m.dbClient.getConnection(ctx), namespacePath)
	if err != nil {
		return err
	}

	if namespace == nil {
		return errors.New(errors.ENotFound, "Namespace not found")
	}

	timestamp := currentTime()

	records := []goqu.Record{}
	for _, v := range variables {
		records = append(records, goqu.Record{
			"id":           newResourceID(),
			"version":      initialResourceVersion,
			"created_at":   timestamp,
			"updated_at":   timestamp,
			"namespace_id": namespace.id,
			"key":          v.Key,
			"value":        v.Value,
			"category":     v.Category,
			"hcl":          v.Hcl,
		})
	}

	sql, args, err := dialect.Insert("namespace_variables").
		Prepared(true).
		Rows(records).
		ToSQL()

	if err != nil {
		return err
	}

	if _, err := m.dbClient.getConnection(ctx).Exec(ctx, sql, args...); err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				return errors.New(
					errors.EConflict,
					"Variable with key already exists in namespace %s", namespacePath,
				)
			}
			if isForeignKeyViolation(pgErr) {
				switch pgErr.ConstraintName {
				case "fk_namespace_variables_namespace_id":
					return errors.New(errors.ENotFound, "namespace does not exist")
				}
			}
		}
		return err
	}

	return nil
}

func (m *variables) UpdateVariable(ctx context.Context, variable *models.Variable) (*models.Variable, error) {
	timestamp := currentTime()

	sql, args, err := dialect.Update("namespace_variables").
		Prepared(true).
		Set(goqu.Record{
			"version":    goqu.L("? + ?", goqu.C("version"), 1),
			"updated_at": timestamp,
			"key":        variable.Key,
			"value":      variable.Value,
			"hcl":        variable.Hcl,
		}).
		Where(goqu.Ex{"id": variable.Metadata.ID, "version": variable.Metadata.Version}).Returning(variableFieldList...).ToSQL()

	if err != nil {
		return nil, err
	}

	updatedVariable, err := scanVariable(m.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...), false)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrOptimisticLockError
		}
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				return nil, errors.New(
					errors.EConflict,
					"Variable with key %s in namespace %s already exists", variable.Key, variable.NamespacePath,
				)
			}
		}
		return nil, err
	}

	updatedVariable.NamespacePath = variable.NamespacePath

	return updatedVariable, nil
}

func (m *variables) DeleteVariable(ctx context.Context, variable *models.Variable) error {
	sql, args, err := dialect.Delete("namespace_variables").
		Prepared(true).
		Where(
			goqu.Ex{
				"id":      variable.Metadata.ID,
				"version": variable.Metadata.Version,
			},
		).Returning(variableFieldList...).ToSQL()

	if err != nil {
		return err
	}

	if _, err := scanVariable(m.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...), false); err != nil {
		if err == pgx.ErrNoRows {
			return ErrOptimisticLockError
		}

		return err
	}

	return nil
}

func (m *variables) DeleteVariables(ctx context.Context, namespacePath string, category models.VariableCategory) error {
	sql, args, err := dialect.Delete("namespace_variables").
		Prepared(true).
		Where(goqu.Ex{
			"namespace_id": dialect.From("namespaces").Select("id").Where(goqu.Ex{"path": namespacePath}),
			"category":     string(category),
		}).ToSQL()

	if err != nil {
		return err
	}

	if _, err := m.dbClient.getConnection(ctx).Exec(ctx, sql, args...); err != nil {
		return err
	}

	return nil
}

func (m *variables) GetVariables(ctx context.Context, input *GetVariablesInput) (*VariableResult, error) {
	ex := goqu.And()

	if input.Filter != nil {
		if input.Filter.NamespacePaths != nil {
			ex = ex.Append(goqu.I("namespaces.path").In(input.Filter.NamespacePaths))
		}

		if input.Filter.VariableIDs != nil {
			// This check avoids an SQL syntax error if an empty slice is provided.
			if len(input.Filter.VariableIDs) > 0 {
				ex = ex.Append(goqu.I("namespace_variables.id").In(input.Filter.VariableIDs))
			}
		}
	}

	query := dialect.From("namespace_variables").
		Select(m.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"namespace_variables.namespace_id": goqu.I("namespaces.id")})).
		Where(ex)

	sortDirection := pagination.AscSort

	var sortBy *pagination.FieldDescriptor
	if input.Sort != nil {
		sortDirection = input.Sort.getSortDirection()
		sortBy = input.Sort.getFieldDescriptor()
	}

	qBuilder, err := pagination.NewPaginatedQueryBuilder(
		input.PaginationOptions,
		&pagination.FieldDescriptor{Key: "id", Table: "namespace_variables", Col: "id"},
		sortBy,
		sortDirection,
	)

	if err != nil {
		return nil, err
	}

	rows, err := qBuilder.Execute(ctx, m.dbClient.getConnection(ctx), query)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	// Scan rows
	results := []models.Variable{}
	for rows.Next() {
		item, err := scanVariable(rows, true)
		if err != nil {
			return nil, err
		}

		results = append(results, *item)
	}

	if err := rows.Finalize(&results); err != nil {
		return nil, err
	}

	result := VariableResult{
		PageInfo:  rows.GetPageInfo(),
		Variables: results,
	}

	return &result, nil
}

func (m *variables) getSelectFields() []interface{} {
	selectFields := []interface{}{}
	for _, field := range variableFieldList {
		selectFields = append(selectFields, fmt.Sprintf("namespace_variables.%s", field))
	}

	selectFields = append(selectFields, "namespaces.path")

	return selectFields
}

func scanVariable(row scanner, withNamespacePath bool) (*models.Variable, error) {
	variable := &models.Variable{}

	var namespacePath string

	fields := []interface{}{
		&variable.Metadata.ID,
		&variable.Metadata.CreationTimestamp,
		&variable.Metadata.LastUpdatedTimestamp,
		&variable.Metadata.Version,
		&variable.Key,
		&variable.Value,
		&variable.Category,
		&variable.Hcl,
	}

	if withNamespacePath {
		fields = append(fields, &namespacePath)
	}

	err := row.Scan(fields...)
	if err != nil {
		return nil, err
	}

	if withNamespacePath {
		variable.NamespacePath = namespacePath
	}

	return variable, nil
}
