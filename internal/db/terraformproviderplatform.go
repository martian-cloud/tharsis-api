package db

//go:generate mockery --name TerraformProviderPlatforms --inpackage --case underscore

import (
	"context"
	"fmt"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v4"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// TerraformProviderPlatforms encapsulates the logic to access terraform provider platforms from the database
type TerraformProviderPlatforms interface {
	GetProviderPlatformByID(ctx context.Context, id string) (*models.TerraformProviderPlatform, error)
	GetProviderPlatforms(ctx context.Context, input *GetProviderPlatformsInput) (*ProviderPlatformsResult, error)
	CreateProviderPlatform(ctx context.Context, providerPlatform *models.TerraformProviderPlatform) (*models.TerraformProviderPlatform, error)
	UpdateProviderPlatform(ctx context.Context, providerPlatform *models.TerraformProviderPlatform) (*models.TerraformProviderPlatform, error)
	DeleteProviderPlatform(ctx context.Context, providerPlatform *models.TerraformProviderPlatform) error
}

// TerraformProviderPlatformSortableField represents the fields that a provider platform can be sorted by
type TerraformProviderPlatformSortableField string

// TerraformProviderPlatformSortableField constants
const (
	TerraformProviderPlatformSortableFieldUpdatedAtAsc  TerraformProviderPlatformSortableField = "UPDATED_AT_ASC"
	TerraformProviderPlatformSortableFieldUpdatedAtDesc TerraformProviderPlatformSortableField = "UPDATED_AT_DESC"
)

func (ts TerraformProviderPlatformSortableField) getFieldDescriptor() *fieldDescriptor {
	switch ts {
	case TerraformProviderPlatformSortableFieldUpdatedAtAsc, TerraformProviderPlatformSortableFieldUpdatedAtDesc:
		return &fieldDescriptor{key: "updated_at", table: "terraform_provider_platforms", col: "updated_at"}
	default:
		return nil
	}
}

func (ts TerraformProviderPlatformSortableField) getSortDirection() SortDirection {
	if strings.HasSuffix(string(ts), "_DESC") {
		return DescSort
	}
	return AscSort
}

// TerraformProviderPlatformFilter contains the supported fields for filtering TerraformProviderPlatform resources
type TerraformProviderPlatformFilter struct {
	ProviderID        *string
	ProviderVersionID *string
	BinaryUploaded    *bool
	OperatingSystem   *string
	Architecture      *string
}

// GetProviderPlatformsInput is the input for listing terraform provider platforms
type GetProviderPlatformsInput struct {
	// Sort specifies the field to sort on and direction
	Sort *TerraformProviderPlatformSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *PaginationOptions
	// Filter is used to filter the results
	Filter *TerraformProviderPlatformFilter
}

// ProviderPlatformsResult contains the response data and page information
type ProviderPlatformsResult struct {
	PageInfo          *PageInfo
	ProviderPlatforms []models.TerraformProviderPlatform
}

type terraformProviderPlatforms struct {
	dbClient *Client
}

var providerPlatformFieldList = append(metadataFieldList, "provider_version_id", "os", "arch", "sha_sum", "filename", "binary_uploaded", "created_by")

// NewTerraformProviderPlatforms returns an instance of the TerraformProviderPlatforms interface
func NewTerraformProviderPlatforms(dbClient *Client) TerraformProviderPlatforms {
	return &terraformProviderPlatforms{dbClient: dbClient}
}

func (t *terraformProviderPlatforms) GetProviderPlatformByID(ctx context.Context, id string) (*models.TerraformProviderPlatform, error) {
	return t.getProviderPlatform(ctx, goqu.Ex{"terraform_provider_platforms.id": id})
}

func (t *terraformProviderPlatforms) GetProviderPlatforms(ctx context.Context, input *GetProviderPlatformsInput) (*ProviderPlatformsResult, error) {
	ex := goqu.Ex{}

	if input.Filter != nil {
		if input.Filter.ProviderID != nil {
			ex["terraform_provider_versions.provider_id"] = *input.Filter.ProviderID
		}

		if input.Filter.ProviderVersionID != nil {
			ex["terraform_provider_platforms.provider_version_id"] = *input.Filter.ProviderVersionID
		}

		if input.Filter.BinaryUploaded != nil {
			ex["terraform_provider_platforms.binary_uploaded"] = *input.Filter.BinaryUploaded
		}

		if input.Filter.OperatingSystem != nil {
			ex["terraform_provider_platforms.os"] = *input.Filter.OperatingSystem
		}

		if input.Filter.Architecture != nil {
			ex["terraform_provider_platforms.arch"] = *input.Filter.Architecture
		}
	}

	query := dialect.From(goqu.T("terraform_provider_platforms")).
		InnerJoin(goqu.T("terraform_provider_versions"), goqu.On(goqu.Ex{"terraform_provider_platforms.provider_version_id": goqu.I("terraform_provider_versions.id")})).
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
		&fieldDescriptor{key: "id", table: "terraform_provider_platforms", col: "id"},
		sortBy,
		sortDirection,
		providerPlatformFieldResolver,
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
	results := []models.TerraformProviderPlatform{}
	for rows.Next() {
		item, err := scanTerraformProviderPlatform(rows)
		if err != nil {
			return nil, err
		}

		results = append(results, *item)
	}

	if err := rows.finalize(&results); err != nil {
		return nil, err
	}

	result := ProviderPlatformsResult{
		PageInfo:          rows.getPageInfo(),
		ProviderPlatforms: results,
	}

	return &result, nil
}

func (t *terraformProviderPlatforms) CreateProviderPlatform(ctx context.Context, providerPlatform *models.TerraformProviderPlatform) (*models.TerraformProviderPlatform, error) {
	timestamp := currentTime()

	sql, args, err := dialect.Insert("terraform_provider_platforms").
		Prepared(true).
		Rows(goqu.Record{
			"id":                  newResourceID(),
			"version":             initialResourceVersion,
			"created_at":          timestamp,
			"updated_at":          timestamp,
			"provider_version_id": providerPlatform.ProviderVersionID,
			"os":                  providerPlatform.OperatingSystem,
			"arch":                providerPlatform.Architecture,
			"sha_sum":             providerPlatform.SHASum,
			"filename":            providerPlatform.Filename,
			"binary_uploaded":     providerPlatform.BinaryUploaded,
			"created_by":          providerPlatform.CreatedBy,
		}).
		Returning(providerPlatformFieldList...).ToSQL()
	if err != nil {
		return nil, err
	}

	createdProviderPlatform, err := scanTerraformProviderPlatform(t.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				return nil, errors.NewError(
					errors.EConflict,
					fmt.Sprintf("terraform provider platform %s_%s already exists", providerPlatform.OperatingSystem, providerPlatform.Architecture),
				)
			}
		}
		return nil, err
	}

	return createdProviderPlatform, nil
}

func (t *terraformProviderPlatforms) UpdateProviderPlatform(ctx context.Context, providerPlatform *models.TerraformProviderPlatform) (*models.TerraformProviderPlatform, error) {
	timestamp := currentTime()

	sql, args, err := dialect.Update("terraform_provider_platforms").
		Prepared(true).
		Set(
			goqu.Record{
				"version":         goqu.L("? + ?", goqu.C("version"), 1),
				"updated_at":      timestamp,
				"binary_uploaded": providerPlatform.BinaryUploaded,
			},
		).Where(goqu.Ex{"id": providerPlatform.Metadata.ID, "version": providerPlatform.Metadata.Version}).Returning(providerPlatformFieldList...).ToSQL()

	if err != nil {
		return nil, err
	}

	updatedProviderPlatform, err := scanTerraformProviderPlatform(t.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrOptimisticLockError
		}
		return nil, err
	}

	return updatedProviderPlatform, nil
}

func (t *terraformProviderPlatforms) DeleteProviderPlatform(ctx context.Context, providerPlatform *models.TerraformProviderPlatform) error {

	sql, args, err := dialect.Delete("terraform_provider_platforms").
		Prepared(true).
		Where(
			goqu.Ex{
				"id":      providerPlatform.Metadata.ID,
				"version": providerPlatform.Metadata.Version,
			},
		).Returning(providerPlatformFieldList...).ToSQL()
	if err != nil {
		return err
	}

	_, err = scanTerraformProviderPlatform(t.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return ErrOptimisticLockError
		}
		return err
	}

	return nil
}

func (t *terraformProviderPlatforms) getProviderPlatform(ctx context.Context, exp goqu.Ex) (*models.TerraformProviderPlatform, error) {
	query := dialect.From(goqu.T("terraform_provider_platforms")).
		Prepared(true).
		Select(t.getSelectFields()...).
		Where(exp)

	sql, args, err := query.ToSQL()
	if err != nil {
		return nil, err
	}

	providerPlatform, err := scanTerraformProviderPlatform(t.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return providerPlatform, nil
}

func (t *terraformProviderPlatforms) getSelectFields() []interface{} {
	selectFields := []interface{}{}
	for _, field := range providerPlatformFieldList {
		selectFields = append(selectFields, fmt.Sprintf("terraform_provider_platforms.%s", field))
	}

	return selectFields
}

func scanTerraformProviderPlatform(row scanner) (*models.TerraformProviderPlatform, error) {
	providerPlatform := &models.TerraformProviderPlatform{}

	fields := []interface{}{
		&providerPlatform.Metadata.ID,
		&providerPlatform.Metadata.CreationTimestamp,
		&providerPlatform.Metadata.LastUpdatedTimestamp,
		&providerPlatform.Metadata.Version,
		&providerPlatform.ProviderVersionID,
		&providerPlatform.OperatingSystem,
		&providerPlatform.Architecture,
		&providerPlatform.SHASum,
		&providerPlatform.Filename,
		&providerPlatform.BinaryUploaded,
		&providerPlatform.CreatedBy,
	}

	err := row.Scan(fields...)
	if err != nil {
		return nil, err
	}

	return providerPlatform, nil
}

func providerPlatformFieldResolver(key string, model interface{}) (string, error) {
	providerPlatform, ok := model.(*models.TerraformProviderPlatform)
	if !ok {
		return "", errors.NewError(errors.EInternal, fmt.Sprintf("Expected provider platform type, got %T", model))
	}

	val, ok := metadataFieldResolver(key, &providerPlatform.Metadata)
	if !ok {
		return "", errors.NewError(errors.EInternal, fmt.Sprintf("Invalid field key requested %s", key))
	}

	return val, nil
}
