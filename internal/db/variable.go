package db

//go:generate go tool mockery --name Variables --inpackage --case underscore

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

// Variables encapsulates the logic to access variables from the database
type Variables interface {
	GetVariables(ctx context.Context, input *GetVariablesInput) (*VariableResult, error)
	GetVariableByID(ctx context.Context, id string) (*models.Variable, error)
	GetVariableByTRN(ctx context.Context, trn string) (*models.Variable, error)
	CreateVariable(ctx context.Context, input *models.Variable) (*models.Variable, error)
	CreateVariables(ctx context.Context, namespacePath string, variables []*models.Variable) error
	UpdateVariable(ctx context.Context, variable *models.Variable) (*models.Variable, error)
	DeleteVariable(ctx context.Context, variable *models.Variable) error
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
	Category       *models.VariableCategory
	Key            *string
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

var variableFieldList = append(metadataFieldList, "key", "category", "sensitive")

// NewVariables returns an instance of the Variables interface
func NewVariables(dbClient *Client) Variables {
	return &variables{dbClient: dbClient}
}

func (m *variables) GetVariableByID(ctx context.Context, id string) (*models.Variable, error) {
	ctx, span := tracer.Start(ctx, "db.GetVariableByID")
	defer span.End()

	return m.getVariable(ctx, goqu.Ex{"namespace_variables.id": id})
}

func (m *variables) GetVariableByTRN(ctx context.Context, trn string) (*models.Variable, error) {
	_, span := tracer.Start(ctx, "db.GetVariableByTRN")
	defer span.End()

	path, err := types.VariableModelType.ResourcePathFromTRN(trn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse TRN", errors.WithSpan(span))
	}

	parts := strings.Split(path, "/")

	if len(parts) < 3 {
		return nil, errors.New("a variable TRN must have namespace path, variable category, and key separated by a forward slash",
			errors.WithErrorCode(errors.EInvalid),
			errors.WithSpan(span),
		)
	}

	return m.getVariable(ctx, goqu.Ex{
		"namespace_variables.key":      parts[len(parts)-1],
		"namespace_variables.category": parts[len(parts)-2],
		"namespaces.path":              strings.Join(parts[:len(parts)-2], "/"),
	})
}

func (m *variables) CreateVariable(ctx context.Context, input *models.Variable) (*models.Variable, error) {
	ctx, span := tracer.Start(ctx, "db.CreateVariable")
	defer span.End()

	namespace, err := getNamespaceByPath(ctx, m.dbClient.getConnection(ctx), input.NamespacePath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get namespace by path", errors.WithSpan(span))
	}

	if namespace == nil {
		return nil, errors.New("Namespace not found", errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	timestamp := currentTime()
	variableID := newResourceID()

	createVariableSQL, createVariableSQLArgs, err := dialect.Insert("namespace_variables").
		Prepared(true).
		Rows(goqu.Record{
			"id":           variableID,
			"version":      initialResourceVersion,
			"created_at":   timestamp,
			"updated_at":   timestamp,
			"namespace_id": namespace.id,
			"key":          input.Key,
			"category":     input.Category,
			"sensitive":    input.Sensitive,
		}).ToSQL()

	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	variableVersionID := newResourceID()

	createVariableVersionSQL, createVariableVersionSQLArgs, err := dialect.From("namespace_variable_versions").
		Prepared(true).
		With("namespace_variable_versions",
			dialect.Insert("namespace_variable_versions").
				Rows(goqu.Record{
					"id":          variableVersionID,
					"version":     initialResourceVersion,
					"created_at":  timestamp,
					"updated_at":  timestamp,
					"variable_id": variableID,
					"key":         input.Key,
					"value":       input.Value,
					"hcl":         input.Hcl,
					"secret_data": input.SecretData,
				}).
				Returning("*"),
		).Select(m.getSelectFields()...).
		InnerJoin(goqu.T("namespace_variables"), goqu.On(goqu.Ex{"namespace_variable_versions.variable_id": goqu.I("namespace_variables.id")})).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"namespace_variables.namespace_id": goqu.I("namespaces.id")})).
		ToSQL()

	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	createLatestVariableRowSQL, createLatestVariableRowArgs, err := dialect.Insert("latest_namespace_variable_versions").
		Prepared(true).
		Rows(goqu.Record{
			"variable_id": variableID,
			"version_id":  variableVersionID,
		}).ToSQL()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	tx, err := m.dbClient.getConnection(ctx).Begin(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return nil, err
	}

	// Rollback is safe to call even if the tx is already closed, so if
	// the tx commits successfully, this is a no-op
	defer func() {
		if txErr := tx.Rollback(ctx); txErr != nil && txErr != pgx.ErrTxClosed {
			m.dbClient.logger.WithContextFields(ctx).Errorf("failed to rollback tx for CreateVariable: %v", txErr)
		}
	}()

	// Execute query to create variable
	_, err = tx.Exec(ctx, createVariableSQL, createVariableSQLArgs...)
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				return nil, errors.New(
					"Variable with key %s in namespace %s already exists", input.Key, input.NamespacePath,
					errors.WithErrorCode(errors.EConflict),
					errors.WithSpan(span),
				)
			}
			if isForeignKeyViolation(pgErr) {
				switch pgErr.ConstraintName {
				case "fk_namespace_variables_namespace_id":
					return nil, errors.New("namespace does not exist", errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
				}
			}
		}
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	createdVariable, err := scanVariable(tx.QueryRow(ctx, createVariableVersionSQL, createVariableVersionSQLArgs...))
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isForeignKeyViolation(pgErr) {
				switch pgErr.ConstraintName {
				case "fk_variable_id":
					return nil, errors.New("variable does not exist", errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
				}
			}
		}
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	_, err = tx.Exec(ctx, createLatestVariableRowSQL, createLatestVariableRowArgs...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, errors.Wrap(err, "failed to commit DB transaction", errors.WithSpan(span))
	}

	return createdVariable, nil
}

func (m *variables) CreateVariables(ctx context.Context, namespacePath string, variables []*models.Variable) error {
	ctx, span := tracer.Start(ctx, "db.CreateVariables")
	defer span.End()

	namespace, err := getNamespaceByPath(ctx, m.dbClient.getConnection(ctx), namespacePath)
	if err != nil {
		return errors.Wrap(err, "failed to get namespace by path", errors.WithSpan(span))
	}

	if namespace == nil {
		return errors.New("Namespace not found", errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	timestamp := currentTime()

	variableRecords := []goqu.Record{}
	for _, v := range variables {
		variableRecords = append(variableRecords, goqu.Record{
			"id":           newResourceID(),
			"version":      initialResourceVersion,
			"created_at":   timestamp,
			"updated_at":   timestamp,
			"namespace_id": namespace.id,
			"key":          v.Key,
			"category":     v.Category,
			"sensitive":    v.Sensitive,
		})
	}

	variableVersionRecords := []goqu.Record{}
	for i, v := range variables {
		variableVersionRecords = append(variableVersionRecords, goqu.Record{
			"id":          newResourceID(),
			"version":     initialResourceVersion,
			"created_at":  timestamp,
			"updated_at":  timestamp,
			"variable_id": variableRecords[i]["id"],
			"key":         v.Key,
			"value":       v.Value,
			"hcl":         v.Hcl,
			"secret_data": v.SecretData,
		})
	}

	latestVersionRecords := []goqu.Record{}
	for i := range variables {
		latestVersionRecords = append(latestVersionRecords, goqu.Record{
			"variable_id": variableRecords[i]["id"],
			"version_id":  variableVersionRecords[i]["id"],
		})
	}

	tx, err := m.dbClient.getConnection(ctx).Begin(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return err
	}

	// Rollback is safe to call even if the tx is already closed, so if
	// the tx commits successfully, this is a no-op
	defer func() {
		if txErr := tx.Rollback(ctx); txErr != nil && txErr != pgx.ErrTxClosed {
			m.dbClient.logger.WithContextFields(ctx).Errorf("failed to rollback tx for CreateVariables: %v", txErr)
		}
	}()

	sql, args, err := dialect.Insert("namespace_variables").
		Prepared(true).
		Rows(variableRecords).
		ToSQL()

	if err != nil {
		return errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	if _, err = tx.Exec(ctx, sql, args...); err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				return errors.New(
					"Variable with key already exists in namespace %s", namespacePath,
					errors.WithErrorCode(errors.EConflict),
					errors.WithSpan(span),
				)
			}
			if isForeignKeyViolation(pgErr) {
				switch pgErr.ConstraintName {
				case "fk_namespace_variables_namespace_id":
					return errors.New("namespace does not exist", errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
				}
			}
		}
		return errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	// Insert variable versions
	sql, args, err = dialect.Insert("namespace_variable_versions").
		Prepared(true).
		Rows(variableVersionRecords).
		ToSQL()

	if err != nil {
		return errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	if _, err = tx.Exec(ctx, sql, args...); err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isForeignKeyViolation(pgErr) {
				switch pgErr.ConstraintName {
				case "fk_variable_id":
					return errors.New("namespace does not exist", errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
				}
			}
		}
		return errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	sql, args, err = dialect.Insert("latest_namespace_variable_versions").
		Prepared(true).
		Rows(latestVersionRecords).
		ToSQL()

	if err != nil {
		return errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	if _, err = tx.Exec(ctx, sql, args...); err != nil {
		return errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	if err := tx.Commit(ctx); err != nil {
		return errors.Wrap(err, "failed to commit DB transaction", errors.WithSpan(span))
	}

	return nil
}

func (m *variables) UpdateVariable(ctx context.Context, variable *models.Variable) (*models.Variable, error) {
	ctx, span := tracer.Start(ctx, "db.UpdateVariable")
	defer span.End()

	timestamp := currentTime()

	tx, err := m.dbClient.getConnection(ctx).Begin(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin DB transaction", errors.WithSpan(span))
	}

	// Rollback is safe to call even if the tx is already closed, so if
	// the tx commits successfully, this is a no-op
	defer func() {
		if txErr := tx.Rollback(ctx); txErr != nil && txErr != pgx.ErrTxClosed {
			m.dbClient.logger.WithContextFields(ctx).Errorf("failed to rollback tx for UpdateVariable: %v", txErr)
		}
	}()

	newVersionID := newResourceID()

	sql, args, err := dialect.Insert("namespace_variable_versions").
		Prepared(true).
		Rows(goqu.Record{
			"id":          newVersionID,
			"version":     initialResourceVersion,
			"created_at":  timestamp,
			"updated_at":  timestamp,
			"variable_id": variable.Metadata.ID,
			"key":         variable.Key,
			"value":       variable.Value,
			"hcl":         variable.Hcl,
			"secret_data": variable.SecretData,
		}).ToSQL()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	if _, err = tx.Exec(ctx, sql, args...); err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return nil, ErrOptimisticLockError
		}
		if pgErr := asPgError(err); pgErr != nil {
			if isForeignKeyViolation(pgErr) {
				switch pgErr.ConstraintName {
				case "fk_variable_id":
					return nil, errors.New("variable does not exist", errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
				}
			}
		}
		return nil, err
	}

	sql, args, err = dialect.Update("latest_namespace_variable_versions").
		Prepared(true).
		Set(goqu.Record{
			"variable_id": variable.Metadata.ID,
			"version_id":  newVersionID,
		}).Where(goqu.Ex{"variable_id": variable.Metadata.ID}).ToSQL()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	if _, err = tx.Exec(ctx, sql, args...); err != nil {
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	sql, args, err = dialect.From("namespace_variables").
		Prepared(true).
		With("namespace_variables",
			dialect.Update("namespace_variables").
				Set(goqu.Record{
					"version":    goqu.L("? + ?", goqu.C("version"), 1),
					"updated_at": timestamp,
					"key":        variable.Key,
				}).Where(goqu.Ex{"id": variable.Metadata.ID, "version": variable.Metadata.Version}).
				Returning("*"),
		).Select(m.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"namespace_variables.namespace_id": goqu.I("namespaces.id")})).
		InnerJoin(goqu.T("latest_namespace_variable_versions"), goqu.On(goqu.Ex{"namespace_variables.id": goqu.I("latest_namespace_variable_versions.variable_id")})).
		InnerJoin(goqu.T("namespace_variable_versions"), goqu.On(goqu.Ex{"latest_namespace_variable_versions.version_id": goqu.I("namespace_variable_versions.id")})).
		ToSQL()

	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	updatedVariable, err := scanVariable(tx.QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return nil, ErrOptimisticLockError
		}
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				return nil, errors.New(
					"Variable with key %s in namespace %s already exists", variable.Key, variable.NamespacePath,
					errors.WithErrorCode(errors.EConflict),
					errors.WithSpan(span),
				)
			}
		}
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, errors.Wrap(err, "failed to commit DB transaction", errors.WithSpan(span))
	}

	return updatedVariable, nil
}

func (m *variables) DeleteVariable(ctx context.Context, variable *models.Variable) error {
	ctx, span := tracer.Start(ctx, "db.DeleteVariable")
	defer span.End()

	sql, args, err := dialect.From("namespace_variables").
		Prepared(true).
		With("namespace_variables",
			dialect.Delete("namespace_variables").
				Where(goqu.Ex{"id": variable.Metadata.ID, "version": variable.Metadata.Version}).
				Returning("*"),
		).Select(m.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"namespace_variables.namespace_id": goqu.I("namespaces.id")})).
		InnerJoin(goqu.T("latest_namespace_variable_versions"), goqu.On(goqu.Ex{"namespace_variables.id": goqu.I("latest_namespace_variable_versions.variable_id")})).
		InnerJoin(goqu.T("namespace_variable_versions"), goqu.On(goqu.Ex{"latest_namespace_variable_versions.version_id": goqu.I("namespace_variable_versions.id")})).
		ToSQL()

	if err != nil {
		return errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	if _, err := scanVariable(m.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...)); err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return ErrOptimisticLockError
		}
		return errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return nil
}

func (m *variables) GetVariables(ctx context.Context, input *GetVariablesInput) (*VariableResult, error) {
	ctx, span := tracer.Start(ctx, "db.GetVariables")
	defer span.End()

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

		if input.Filter.Category != nil {
			ex = ex.Append(goqu.I("namespace_variables.category").Eq(string(*input.Filter.Category)))
		}

		if input.Filter.Key != nil {
			ex = ex.Append(goqu.I("namespace_variables.key").Eq(*input.Filter.Key))
		}
	}

	query := dialect.From("namespace_variables").
		Select(m.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"namespace_variables.namespace_id": goqu.I("namespaces.id")})).
		InnerJoin(goqu.T("latest_namespace_variable_versions"), goqu.On(goqu.Ex{"namespace_variables.id": goqu.I("latest_namespace_variable_versions.variable_id")})).
		InnerJoin(goqu.T("namespace_variable_versions"), goqu.On(goqu.Ex{"latest_namespace_variable_versions.version_id": goqu.I("namespace_variable_versions.id")})).
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
		pagination.WithSortByField(sortBy, sortDirection),
	)

	if err != nil {
		return nil, errors.Wrap(err, "failed to create paginated query builder", errors.WithSpan(span))
	}

	rows, err := qBuilder.Execute(ctx, m.dbClient.getConnection(ctx), query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	defer rows.Close()

	// Scan rows
	results := []models.Variable{}
	for rows.Next() {
		item, err := scanVariable(rows)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan row", errors.WithSpan(span))
		}

		results = append(results, *item)
	}

	if err := rows.Finalize(&results); err != nil {
		return nil, errors.Wrap(err, "failed to finalize rows", errors.WithSpan(span))
	}

	result := VariableResult{
		PageInfo:  rows.GetPageInfo(),
		Variables: results,
	}

	return &result, nil
}

func (m *variables) getVariable(ctx context.Context, ex goqu.Ex) (*models.Variable, error) {
	ctx, span := tracer.Start(ctx, "db.getVariable")
	defer span.End()

	sql, _, err := dialect.From("namespace_variables").
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"namespace_variables.namespace_id": goqu.I("namespaces.id")})).
		InnerJoin(goqu.T("latest_namespace_variable_versions"), goqu.On(goqu.Ex{"namespace_variables.id": goqu.I("latest_namespace_variable_versions.variable_id")})).
		InnerJoin(goqu.T("namespace_variable_versions"), goqu.On(goqu.Ex{"latest_namespace_variable_versions.version_id": goqu.I("namespace_variable_versions.id")})).
		Select(m.getSelectFields()...).
		Where(ex).ToSQL()

	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL for query to get variable by ID", errors.WithSpan(span))
	}

	variable, err := scanVariable(m.dbClient.getConnection(ctx).QueryRow(ctx, sql))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}

		if pgErr := asPgError(err); pgErr != nil {
			if isInvalidIDViolation(pgErr) {
				return nil, ErrInvalidID
			}
		}

		return nil, errors.Wrap(err, "failed execute query to get variable by ID", errors.WithSpan(span))
	}

	return variable, nil
}

func (m *variables) getSelectFields() []interface{} {
	selectFields := []interface{}{}

	for _, field := range variableFieldList {
		selectFields = append(selectFields, fmt.Sprintf("namespace_variables.%s", field))
	}

	// Add columns for namespace variable versions
	selectFields = append(selectFields, "namespace_variable_versions.id")
	selectFields = append(selectFields, "namespace_variable_versions.value")
	selectFields = append(selectFields, "namespace_variable_versions.secret_data")
	selectFields = append(selectFields, "namespace_variable_versions.hcl")

	// Add columns for namespaces
	selectFields = append(selectFields, "namespaces.path")

	return selectFields
}

func scanVariable(row scanner) (*models.Variable, error) {
	variable := &models.Variable{}

	fields := []interface{}{
		&variable.Metadata.ID,
		&variable.Metadata.CreationTimestamp,
		&variable.Metadata.LastUpdatedTimestamp,
		&variable.Metadata.Version,
		&variable.Key,
		&variable.Category,
		&variable.Sensitive,
		&variable.LatestVersionID,
		&variable.Value,
		&variable.SecretData,
		&variable.Hcl,
		&variable.NamespacePath,
	}

	err := row.Scan(fields...)
	if err != nil {
		return nil, err
	}

	variable.Metadata.TRN = types.VariableModelType.BuildTRN(variable.NamespacePath, string(variable.Category), variable.Key)

	return variable, nil
}
