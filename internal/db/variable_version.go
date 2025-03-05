package db

//go:generate go tool mockery --name VariableVersions --inpackage --case underscore

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

// VariableVersions encapsulates the logic to access variableVersions from the database
type VariableVersions interface {
	GetVariableVersions(ctx context.Context, input *GetVariableVersionsInput) (*VariableVersionResult, error)
	GetVariableVersionByID(ctx context.Context, id string) (*models.VariableVersion, error)
}

// VariableVersionSortableField represents the fields that a variable can be sorted by
type VariableVersionSortableField string

// VariableVersionSortableField constants
const (
	VariableVersionSortableFieldCreatedAtAsc  VariableVersionSortableField = "CREATED_AT_ASC"
	VariableVersionSortableFieldCreatedAtDesc VariableVersionSortableField = "CREATED_AT_DESC"
)

func (sf VariableVersionSortableField) getFieldDescriptor() *pagination.FieldDescriptor {
	switch sf {
	case VariableVersionSortableFieldCreatedAtAsc, VariableVersionSortableFieldCreatedAtDesc:
		return &pagination.FieldDescriptor{Key: "created_at", Table: "namespace_variable_versions", Col: "created_at"}
	default:
		return nil
	}
}

func (sf VariableVersionSortableField) getSortDirection() pagination.SortDirection {
	if strings.HasSuffix(string(sf), "_DESC") {
		return pagination.DescSort
	}
	return pagination.AscSort
}

// VariableVersionFilter contains the supported fields for filtering VariableVersion resources
type VariableVersionFilter struct {
	VariableID         *string
	VariableVersionIDs []string
}

// GetVariableVersionsInput is the input for listing variableVersions
type GetVariableVersionsInput struct {
	// Sort specifies the field to sort on and direction
	Sort *VariableVersionSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// Filter is used to filter the results
	Filter *VariableVersionFilter
}

// VariableVersionResult contains the response data and page information
type VariableVersionResult struct {
	PageInfo         *pagination.PageInfo
	VariableVersions []models.VariableVersion
}

type variableVersions struct {
	dbClient *Client
}

var variableVersionFieldList = append(metadataFieldList, "variable_id", "key", "value", "hcl", "secret_data")

// NewVariableVersions returns an instance of the VariableVersions interface
func NewVariableVersions(dbClient *Client) VariableVersions {
	return &variableVersions{dbClient: dbClient}
}

func (m *variableVersions) GetVariableVersionByID(ctx context.Context, id string) (*models.VariableVersion, error) {
	ctx, span := tracer.Start(ctx, "db.GetVariableVersionByID")
	defer span.End()

	sql, _, err := dialect.From("namespace_variable_versions").
		Select(m.getSelectFields()...).
		Where(goqu.Ex{"namespace_variable_versions.id": id}).ToSQL()

	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	variable, err := scanVariableVersion(m.dbClient.getConnection(ctx).QueryRow(ctx, sql))

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		if pgErr := asPgError(err); pgErr != nil {
			if isInvalidIDViolation(pgErr) {
				return nil, errors.Wrap(pgErr, "invalid ID; %s", pgErr.Message, errors.WithSpan(span), errors.WithErrorCode(errors.EInvalid))
			}
		}
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return variable, nil
}

func (m *variableVersions) GetVariableVersions(ctx context.Context, input *GetVariableVersionsInput) (*VariableVersionResult, error) {
	ctx, span := tracer.Start(ctx, "db.GetVariableVersions")
	defer span.End()

	ex := goqu.And()

	if input.Filter != nil {
		if input.Filter.VariableID != nil {
			ex = ex.Append(goqu.I("namespace_variable_versions.variable_id").Eq(*input.Filter.VariableID))
		}

		if len(input.Filter.VariableVersionIDs) > 0 {
			ex = ex.Append(goqu.I("namespace_variable_versions.id").In(input.Filter.VariableVersionIDs))
		}
	}

	query := dialect.From("namespace_variable_versions").
		Select(m.getSelectFields()...).
		Where(ex)

	sortDirection := pagination.AscSort

	var sortBy *pagination.FieldDescriptor
	if input.Sort != nil {
		sortDirection = input.Sort.getSortDirection()
		sortBy = input.Sort.getFieldDescriptor()
	}

	qBuilder, err := pagination.NewPaginatedQueryBuilder(
		input.PaginationOptions,
		&pagination.FieldDescriptor{Key: "id", Table: "namespace_variable_versions", Col: "id"},
		pagination.WithSortByField(sortBy, sortDirection),
	)

	if err != nil {
		return nil, errors.Wrap(err, "failed to build query", errors.WithSpan(span))
	}

	rows, err := qBuilder.Execute(ctx, m.dbClient.getConnection(ctx), query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	defer rows.Close()

	// Scan rows
	results := []models.VariableVersion{}
	for rows.Next() {
		item, err := scanVariableVersion(rows)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan row", errors.WithSpan(span))
		}

		results = append(results, *item)
	}

	if err := rows.Finalize(&results); err != nil {
		return nil, errors.Wrap(err, "failed to finalize rows", errors.WithSpan(span))
	}

	result := VariableVersionResult{
		PageInfo:         rows.GetPageInfo(),
		VariableVersions: results,
	}

	return &result, nil
}

func (m *variableVersions) getSelectFields() []interface{} {
	selectFields := []interface{}{}

	for _, field := range variableVersionFieldList {
		selectFields = append(selectFields, fmt.Sprintf("namespace_variable_versions.%s", field))
	}

	return selectFields
}

func scanVariableVersion(row scanner) (*models.VariableVersion, error) {
	variableVersion := &models.VariableVersion{}

	fields := []interface{}{
		&variableVersion.Metadata.ID,
		&variableVersion.Metadata.CreationTimestamp,
		&variableVersion.Metadata.LastUpdatedTimestamp,
		&variableVersion.Metadata.Version,
		&variableVersion.VariableID,
		&variableVersion.Key,
		&variableVersion.Value,
		&variableVersion.Hcl,
		&variableVersion.SecretData,
	}

	err := row.Scan(fields...)
	if err != nil {
		return nil, err
	}

	return variableVersion, nil
}
