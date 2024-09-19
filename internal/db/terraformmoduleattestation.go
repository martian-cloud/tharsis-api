package db

//go:generate mockery --name TerraformModuleAttestations --inpackage --case underscore

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v4"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// TerraformModuleAttestations encapsulates the logic to access terraform module attestationsfrom the database
type TerraformModuleAttestations interface {
	GetModuleAttestationByID(ctx context.Context, id string) (*models.TerraformModuleAttestation, error)
	GetModuleAttestations(ctx context.Context, input *GetModuleAttestationsInput) (*ModuleAttestationsResult, error)
	CreateModuleAttestation(ctx context.Context, moduleAttestation *models.TerraformModuleAttestation) (*models.TerraformModuleAttestation, error)
	UpdateModuleAttestation(ctx context.Context, moduleAttestation *models.TerraformModuleAttestation) (*models.TerraformModuleAttestation, error)
	DeleteModuleAttestation(ctx context.Context, moduleAttestation *models.TerraformModuleAttestation) error
}

// TerraformModuleAttestationSortableField represents the fields that a moduleAttestation can be sorted by
type TerraformModuleAttestationSortableField string

// TerraformModuleAttestationSortableField constants
const (
	TerraformModuleAttestationSortableFieldPredicateAsc  TerraformModuleAttestationSortableField = "PREDICATE_ASC"
	TerraformModuleAttestationSortableFieldPredicateDesc TerraformModuleAttestationSortableField = "PREDICATE_DESC"
	TerraformModuleAttestationSortableFieldCreatedAtAsc  TerraformModuleAttestationSortableField = "CREATED_AT_ASC"
	TerraformModuleAttestationSortableFieldCreatedAtDesc TerraformModuleAttestationSortableField = "CREATED_AT_DESC"
)

func (ts TerraformModuleAttestationSortableField) getFieldDescriptor() *pagination.FieldDescriptor {
	switch ts {
	case TerraformModuleAttestationSortableFieldPredicateAsc, TerraformModuleAttestationSortableFieldPredicateDesc:
		return &pagination.FieldDescriptor{Key: "predicate", Table: "terraform_module_attestations", Col: "predicate_type"}
	case TerraformModuleAttestationSortableFieldCreatedAtAsc, TerraformModuleAttestationSortableFieldCreatedAtDesc:
		return &pagination.FieldDescriptor{Key: "created_at", Table: "terraform_module_attestations", Col: "created_at"}
	default:
		return nil
	}
}

func (ts TerraformModuleAttestationSortableField) getSortDirection() pagination.SortDirection {
	if strings.HasSuffix(string(ts), "_DESC") {
		return pagination.DescSort
	}
	return pagination.AscSort
}

// TerraformModuleAttestationFilter contains the supported fields for filtering TerraformModuleAttestation resources
type TerraformModuleAttestationFilter struct {
	TimeRangeStart       *time.Time
	Digest               *string
	ModuleID             *string
	ModuleAttestationIDs []string
}

// GetModuleAttestationsInput is the input for listing terraform moduleAttestations
type GetModuleAttestationsInput struct {
	// Sort specifies the field to sort on and direction
	Sort *TerraformModuleAttestationSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// Filter is used to filter the results
	Filter *TerraformModuleAttestationFilter
}

// ModuleAttestationsResult contains the response data and page information
type ModuleAttestationsResult struct {
	PageInfo           *pagination.PageInfo
	ModuleAttestations []models.TerraformModuleAttestation
}

type terraformModuleAttestations struct {
	dbClient *Client
}

var moduleAttestationFieldList = append(metadataFieldList, "module_id", "description", "data", "data_sha_sum", "schema_type", "predicate_type", "digests", "created_by")

// NewTerraformModuleAttestations returns an instance of the TerraformModuleAttestations interface
func NewTerraformModuleAttestations(dbClient *Client) TerraformModuleAttestations {
	return &terraformModuleAttestations{dbClient: dbClient}
}

func (t *terraformModuleAttestations) GetModuleAttestationByID(ctx context.Context, id string) (*models.TerraformModuleAttestation, error) {
	ctx, span := tracer.Start(ctx, "db.GetModuleAttestationByID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	return t.getModuleAttestation(ctx, goqu.Ex{"terraform_module_attestations.id": id})
}

func (t *terraformModuleAttestations) GetModuleAttestations(ctx context.Context, input *GetModuleAttestationsInput) (*ModuleAttestationsResult, error) {
	ctx, span := tracer.Start(ctx, "db.GetModuleAttestations")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	ex := goqu.And()

	if input.Filter != nil {
		if input.Filter.ModuleAttestationIDs != nil {
			ex = ex.Append(goqu.I("terraform_module_attestations.id").In(input.Filter.ModuleAttestationIDs))
		}
		if input.Filter.ModuleID != nil {
			ex = ex.Append(goqu.I("terraform_module_attestations.module_id").Eq(*input.Filter.ModuleID))
		}
		if input.Filter.Digest != nil {
			ex = ex.Append(goqu.L(fmt.Sprintf("terraform_module_attestations.digests @> '\"%s\"'", *input.Filter.Digest)))
		}
		if input.Filter.TimeRangeStart != nil {
			// Must use UTC here otherwise, queries will return unexpected results.
			ex = ex.Append(goqu.I("terraform_module_attestations.created_at").Gte(input.Filter.TimeRangeStart.UTC()))
		}
	}

	query := dialect.From(goqu.T("terraform_module_attestations")).
		Select(t.getSelectFields()...).
		Where(ex)

	sortDirection := pagination.AscSort

	var sortBy *pagination.FieldDescriptor
	if input.Sort != nil {
		sortDirection = input.Sort.getSortDirection()
		sortBy = input.Sort.getFieldDescriptor()
	}

	qBuilder, err := pagination.NewPaginatedQueryBuilder(
		input.PaginationOptions,
		&pagination.FieldDescriptor{Key: "id", Table: "terraform_module_attestations", Col: "id"},
		pagination.WithSortByField(sortBy, sortDirection),
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
	results := []models.TerraformModuleAttestation{}
	for rows.Next() {
		item, err := scanTerraformModuleAttestation(rows)
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

	result := ModuleAttestationsResult{
		PageInfo:           rows.GetPageInfo(),
		ModuleAttestations: results,
	}

	return &result, nil
}

func (t *terraformModuleAttestations) CreateModuleAttestation(ctx context.Context, moduleAttestation *models.TerraformModuleAttestation) (*models.TerraformModuleAttestation, error) {
	ctx, span := tracer.Start(ctx, "db.CreateModuleAttestation")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	timestamp := currentTime()

	digests, err := json.Marshal(moduleAttestation.Digests)
	if err != nil {
		tracing.RecordError(span, err, "failed to marshal module attestation digests")
		return nil, err
	}

	sql, args, err := dialect.Insert("terraform_module_attestations").
		Prepared(true).
		Rows(goqu.Record{
			"id":             newResourceID(),
			"version":        initialResourceVersion,
			"created_at":     timestamp,
			"updated_at":     timestamp,
			"module_id":      moduleAttestation.ModuleID,
			"description":    nullableString(moduleAttestation.Description),
			"data":           moduleAttestation.Data,
			"data_sha_sum":   moduleAttestation.DataSHASum,
			"schema_type":    moduleAttestation.SchemaType,
			"predicate_type": moduleAttestation.PredicateType,
			"digests":        digests,
			"created_by":     moduleAttestation.CreatedBy,
		}).
		Returning(moduleAttestationFieldList...).ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	createdModuleAttestation, err := scanTerraformModuleAttestation(t.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				switch pgErr.ConstraintName {
				case "index_terraform_module_attestations_on_module_and_data_sha_sum":
					tracing.RecordError(span, nil,
						"another module attestation with the same data already exists for this module")
					return nil, errors.New("another module attestation with the same data already exists for this module", errors.WithErrorCode(errors.EConflict))
				default:
					tracing.RecordError(span, nil,
						"database constraint violated: %s", pgErr.ConstraintName)
					return nil, errors.New("database constraint violated: %s", pgErr.ConstraintName, errors.WithErrorCode(errors.EConflict))
				}
			}
		}
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	return createdModuleAttestation, nil
}

func (t *terraformModuleAttestations) UpdateModuleAttestation(ctx context.Context, moduleAttestation *models.TerraformModuleAttestation) (*models.TerraformModuleAttestation, error) {
	ctx, span := tracer.Start(ctx, "db.UpdateModuleAttestation")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	timestamp := currentTime()

	record := goqu.Record{
		"version":     goqu.L("? + ?", goqu.C("version"), 1),
		"updated_at":  timestamp,
		"description": nullableString(moduleAttestation.Description),
	}

	sql, args, err := dialect.Update("terraform_module_attestations").
		Prepared(true).
		Set(record).
		Where(goqu.Ex{"id": moduleAttestation.Metadata.ID, "version": moduleAttestation.Metadata.Version}).Returning(moduleAttestationFieldList...).ToSQL()

	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	updatedModuleAttestation, err := scanTerraformModuleAttestation(t.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return nil, ErrOptimisticLockError
		}
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	return updatedModuleAttestation, nil
}

func (t *terraformModuleAttestations) DeleteModuleAttestation(ctx context.Context, moduleAttestation *models.TerraformModuleAttestation) error {
	ctx, span := tracer.Start(ctx, "db.DeleteModuleAttestation")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	sql, args, err := dialect.Delete("terraform_module_attestations").
		Prepared(true).
		Where(
			goqu.Ex{
				"id":      moduleAttestation.Metadata.ID,
				"version": moduleAttestation.Metadata.Version,
			},
		).Returning(moduleAttestationFieldList...).ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return err
	}

	_, err = scanTerraformModuleAttestation(t.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
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

func (t *terraformModuleAttestations) getModuleAttestation(ctx context.Context, exp goqu.Ex) (*models.TerraformModuleAttestation, error) {
	query := dialect.From(goqu.T("terraform_module_attestations")).
		Prepared(true).
		Select(t.getSelectFields()...).
		Where(exp)

	sql, args, err := query.ToSQL()
	if err != nil {
		return nil, err
	}

	moduleAttestation, err := scanTerraformModuleAttestation(t.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return moduleAttestation, nil
}

func (t *terraformModuleAttestations) getSelectFields() []interface{} {
	selectFields := []interface{}{}
	for _, field := range moduleAttestationFieldList {
		selectFields = append(selectFields, fmt.Sprintf("terraform_module_attestations.%s", field))
	}

	return selectFields
}

func scanTerraformModuleAttestation(row scanner) (*models.TerraformModuleAttestation, error) {
	var description sql.NullString

	moduleAttestation := &models.TerraformModuleAttestation{
		Digests: []string{},
	}

	fields := []interface{}{
		&moduleAttestation.Metadata.ID,
		&moduleAttestation.Metadata.CreationTimestamp,
		&moduleAttestation.Metadata.LastUpdatedTimestamp,
		&moduleAttestation.Metadata.Version,
		&moduleAttestation.ModuleID,
		&description,
		&moduleAttestation.Data,
		&moduleAttestation.DataSHASum,
		&moduleAttestation.SchemaType,
		&moduleAttestation.PredicateType,
		&moduleAttestation.Digests,
		&moduleAttestation.CreatedBy,
	}

	err := row.Scan(fields...)
	if err != nil {
		return nil, err
	}

	if description.Valid {
		moduleAttestation.Description = description.String
	}

	return moduleAttestation, nil
}
