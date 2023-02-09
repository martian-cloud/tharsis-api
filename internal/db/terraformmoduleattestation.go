package db

//go:generate mockery --name TerraformModuleAttestations --inpackage --case underscore

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

func (ts TerraformModuleAttestationSortableField) getFieldDescriptor() *fieldDescriptor {
	switch ts {
	case TerraformModuleAttestationSortableFieldPredicateAsc, TerraformModuleAttestationSortableFieldPredicateDesc:
		return &fieldDescriptor{key: "predicate", table: "terraform_module_attestations", col: "predicate_type"}
	case TerraformModuleAttestationSortableFieldCreatedAtAsc, TerraformModuleAttestationSortableFieldCreatedAtDesc:
		return &fieldDescriptor{key: "created_at", table: "terraform_module_attestations", col: "created_at"}
	default:
		return nil
	}
}

func (ts TerraformModuleAttestationSortableField) getSortDirection() SortDirection {
	if strings.HasSuffix(string(ts), "_DESC") {
		return DescSort
	}
	return AscSort
}

// TerraformModuleAttestationFilter contains the supported fields for filtering TerraformModuleAttestation resources
type TerraformModuleAttestationFilter struct {
	Digest               *string
	ModuleID             *string
	ModuleAttestationIDs []string
}

// GetModuleAttestationsInput is the input for listing terraform moduleAttestations
type GetModuleAttestationsInput struct {
	// Sort specifies the field to sort on and direction
	Sort *TerraformModuleAttestationSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *PaginationOptions
	// Filter is used to filter the results
	Filter *TerraformModuleAttestationFilter
}

// ModuleAttestationsResult contains the response data and page information
type ModuleAttestationsResult struct {
	PageInfo           *PageInfo
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
	return t.getModuleAttestation(ctx, goqu.Ex{"terraform_module_attestations.id": id})
}

func (t *terraformModuleAttestations) GetModuleAttestations(ctx context.Context, input *GetModuleAttestationsInput) (*ModuleAttestationsResult, error) {
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
	}

	query := dialect.From(goqu.T("terraform_module_attestations")).
		Select(t.getSelectFields()...).
		Where(ex)

	sortDirection := AscSort

	var sortBy *fieldDescriptor
	if input.Sort != nil {
		sortDirection = input.Sort.getSortDirection()
		sortBy = input.Sort.getFieldDescriptor()
	}

	qBuilder, err := newPaginatedQueryBuilder(
		input.PaginationOptions,
		&fieldDescriptor{key: "id", table: "terraform_module_attestations", col: "id"},
		sortBy,
		sortDirection,
		moduleAttestationFieldResolver,
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
	results := []models.TerraformModuleAttestation{}
	for rows.Next() {
		item, err := scanTerraformModuleAttestation(rows)
		if err != nil {
			return nil, err
		}

		results = append(results, *item)
	}

	if err := rows.finalize(&results); err != nil {
		return nil, err
	}

	result := ModuleAttestationsResult{
		PageInfo:           rows.getPageInfo(),
		ModuleAttestations: results,
	}

	return &result, nil
}

func (t *terraformModuleAttestations) CreateModuleAttestation(ctx context.Context, moduleAttestation *models.TerraformModuleAttestation) (*models.TerraformModuleAttestation, error) {
	timestamp := currentTime()

	digests, err := json.Marshal(moduleAttestation.Digests)
	if err != nil {
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
		return nil, err
	}

	createdModuleAttestation, err := scanTerraformModuleAttestation(t.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				switch pgErr.ConstraintName {
				case "index_terraform_module_attestations_on_module_and_data_sha_sum":
					return nil, errors.NewError(errors.EConflict, "another module attestation with the same data already exists for this module")
				default:
					return nil, errors.NewError(errors.EConflict, fmt.Sprintf("database constraint violated: %s", pgErr.ConstraintName))
				}
			}
		}
		return nil, err
	}

	return createdModuleAttestation, nil
}

func (t *terraformModuleAttestations) UpdateModuleAttestation(ctx context.Context, moduleAttestation *models.TerraformModuleAttestation) (*models.TerraformModuleAttestation, error) {
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
		return nil, err
	}

	updatedModuleAttestation, err := scanTerraformModuleAttestation(t.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrOptimisticLockError
		}
		return nil, err
	}

	return updatedModuleAttestation, nil
}

func (t *terraformModuleAttestations) DeleteModuleAttestation(ctx context.Context, moduleAttestation *models.TerraformModuleAttestation) error {

	sql, args, err := dialect.Delete("terraform_module_attestations").
		Prepared(true).
		Where(
			goqu.Ex{
				"id":      moduleAttestation.Metadata.ID,
				"version": moduleAttestation.Metadata.Version,
			},
		).Returning(moduleAttestationFieldList...).ToSQL()
	if err != nil {
		return err
	}

	_, err = scanTerraformModuleAttestation(t.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return ErrOptimisticLockError
		}
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

func moduleAttestationFieldResolver(key string, model interface{}) (string, error) {
	moduleAttestation, ok := model.(*models.TerraformModuleAttestation)
	if !ok {
		return "", errors.NewError(errors.EInternal, fmt.Sprintf("Expected moduleAttestation type, got %T", model))
	}

	val, ok := metadataFieldResolver(key, &moduleAttestation.Metadata)
	if !ok {
		switch key {
		case "predicate":
			val = moduleAttestation.PredicateType
		default:
			return "", errors.NewError(errors.EInternal, fmt.Sprintf("Invalid field key requested %s", key))
		}
	}

	return val, nil
}
