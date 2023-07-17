package db

//go:generate mockery --name TerraformProviderPlatformMirrors --inpackage --case underscore

import (
	"context"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v4"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// TerraformProviderPlatformMirrors encapsulates the logic to access TerraformProviderPlatform from the DB.
type TerraformProviderPlatformMirrors interface {
	GetPlatformMirrorByID(ctx context.Context, id string) (*models.TerraformProviderPlatformMirror, error)
	GetPlatformMirrors(ctx context.Context, input *GetProviderPlatformMirrorsInput) (*ProviderPlatformMirrorsResult, error)
	CreatePlatformMirror(ctx context.Context, platformMirror *models.TerraformProviderPlatformMirror) (*models.TerraformProviderPlatformMirror, error)
	DeletePlatformMirror(ctx context.Context, platformMirror *models.TerraformProviderPlatformMirror) error
}

// TerraformProviderPlatformMirrorSortableField represents fields
// that a TerraformProviderPlatformMirror can be sorted by.
type TerraformProviderPlatformMirrorSortableField string

// TerraformProviderPlatformMirrorSortableField constants
const (
	TerraformProviderPlatformMirrorSortableFieldCreatedAtAsc  TerraformProviderPlatformMirrorSortableField = "CREATED_AT_ASC"
	TerraformProviderPlatformMirrorSortableFieldCreatedAtDesc TerraformProviderPlatformMirrorSortableField = "CREATED_AT_DESC"
)

func (ts TerraformProviderPlatformMirrorSortableField) getFieldDescriptor() *pagination.FieldDescriptor {
	switch ts {
	case TerraformProviderPlatformMirrorSortableFieldCreatedAtAsc, TerraformProviderPlatformMirrorSortableFieldCreatedAtDesc:
		return &pagination.FieldDescriptor{Key: "created_at", Table: "terraform_provider_platform_mirrors", Col: "created_at"}
	default:
		return nil
	}
}

func (ts TerraformProviderPlatformMirrorSortableField) getSortDirection() pagination.SortDirection {
	if strings.HasSuffix(string(ts), "_DESC") {
		return pagination.DescSort
	}
	return pagination.AscSort
}

// TerraformProviderPlatformMirrorFilter represents fields a TerraformProviderPlatformMirror can be filtered by.
type TerraformProviderPlatformMirrorFilter struct {
	VersionMirrorID *string
	OS              *string
	Architecture    *string
}

// GetProviderPlatformMirrorsInput is the input for listing TerraformProviderPlatformMirror.
type GetProviderPlatformMirrorsInput struct {
	// Sort specifies the field to sort on and direction
	Sort *TerraformProviderPlatformMirrorSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// Filter is used to filter the results
	Filter *TerraformProviderPlatformMirrorFilter
}

// ProviderPlatformMirrorsResult contains the result of listing TerraformProviderPlatformMirror.
type ProviderPlatformMirrorsResult struct {
	PageInfo        *pagination.PageInfo
	PlatformMirrors []models.TerraformProviderPlatformMirror
}

type terraformProviderPlatformMirrors struct {
	dbClient *Client
}

var terraformProviderPlatformMirrorsFieldList = append(
	metadataFieldList,
	"os",
	"architecture",
	"version_mirror_id",
)

// NewTerraformProviderPlatformMirrors returns a new instance of TerraformProviderPlatformMirrors
func NewTerraformProviderPlatformMirrors(dbClient *Client) TerraformProviderPlatformMirrors {
	return &terraformProviderPlatformMirrors{
		dbClient: dbClient,
	}
}

func (t *terraformProviderPlatformMirrors) GetPlatformMirrorByID(ctx context.Context, id string) (*models.TerraformProviderPlatformMirror, error) {
	ctx, span := tracer.Start(ctx, "db.GetPlatformMirrorByID")
	defer span.End()

	return t.getVersionMirror(ctx, goqu.Ex{"id": id})
}

func (t *terraformProviderPlatformMirrors) GetPlatformMirrors(ctx context.Context, input *GetProviderPlatformMirrorsInput) (*ProviderPlatformMirrorsResult, error) {
	ctx, span := tracer.Start(ctx, "db.GetPlatformMirrors")
	defer span.End()

	ex := goqu.And()

	if input.Filter != nil {
		if input.Filter.VersionMirrorID != nil {
			ex = ex.Append(goqu.I("version_mirror_id").Eq(*input.Filter.VersionMirrorID))
		}
		if input.Filter.OS != nil {
			ex = ex.Append(goqu.I("os").Eq(*input.Filter.OS))
		}
		if input.Filter.Architecture != nil {
			ex = ex.Append(goqu.I("architecture").Eq(*input.Filter.Architecture))
		}
	}

	query := dialect.From(goqu.T("terraform_provider_platform_mirrors")).
		Select(terraformProviderPlatformMirrorsFieldList...).
		Where(ex)

	sortDirection := pagination.AscSort

	var sortBy *pagination.FieldDescriptor
	if input.Sort != nil {
		sortDirection = input.Sort.getSortDirection()
		sortBy = input.Sort.getFieldDescriptor()
	}

	qBuilder, err := pagination.NewPaginatedQueryBuilder(
		input.PaginationOptions,
		&pagination.FieldDescriptor{Key: "id", Table: "terraform_provider_platform_mirrors", Col: "id"},
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
	results := []models.TerraformProviderPlatformMirror{}
	for rows.Next() {
		item, err := scanPlatformMirror(rows)
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

	result := &ProviderPlatformMirrorsResult{
		PageInfo:        rows.GetPageInfo(),
		PlatformMirrors: results,
	}

	return result, nil
}

func (t *terraformProviderPlatformMirrors) CreatePlatformMirror(ctx context.Context, platformMirror *models.TerraformProviderPlatformMirror) (*models.TerraformProviderPlatformMirror, error) {
	ctx, span := tracer.Start(ctx, "db.CreatePlatformMirror")
	defer span.End()

	timestamp := currentTime()

	sql, args, err := dialect.Insert("terraform_provider_platform_mirrors").
		Prepared(true).
		Rows(goqu.Record{
			"id":                newResourceID(),
			"version":           initialResourceVersion,
			"created_at":        timestamp,
			"updated_at":        timestamp,
			"os":                platformMirror.OS,
			"architecture":      platformMirror.Architecture,
			"version_mirror_id": platformMirror.VersionMirrorID,
		}).
		Returning(terraformProviderPlatformMirrorsFieldList...).ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return platformMirror, err
	}

	createdMirror, err := scanPlatformMirror(t.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				tracing.RecordError(span, nil, "terraform provider platform is already mirrored")
				return nil, errors.New(errors.EConflict, "terraform provider platform is already mirrored")
			}
		}
		tracing.RecordError(span, err, "failed to execute query")
		return platformMirror, err
	}

	return createdMirror, nil
}

func (t *terraformProviderPlatformMirrors) DeletePlatformMirror(ctx context.Context, platformMirror *models.TerraformProviderPlatformMirror) error {
	ctx, span := tracer.Start(ctx, "db.DeletePlatformMirror")
	defer span.End()

	sql, args, err := dialect.Delete("terraform_provider_platform_mirrors").
		Prepared(true).
		Where(
			goqu.Ex{
				"id":      platformMirror.Metadata.ID,
				"version": platformMirror.Metadata.Version,
			},
		).Returning(terraformProviderPlatformMirrorsFieldList...).ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return err
	}

	if _, err = scanPlatformMirror(t.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...)); err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return ErrOptimisticLockError
		}
		tracing.RecordError(span, err, "failed to execute query")
		return err
	}

	return nil
}

func (t *terraformProviderPlatformMirrors) getVersionMirror(ctx context.Context, exp goqu.Ex) (*models.TerraformProviderPlatformMirror, error) {
	query := dialect.From(goqu.T("terraform_provider_platform_mirrors")).
		Prepared(true).
		Select(terraformProviderPlatformMirrorsFieldList...).
		Where(exp)

	sql, args, err := query.ToSQL()
	if err != nil {
		return nil, err
	}

	platformMirror, err := scanPlatformMirror(t.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return platformMirror, nil
}

func scanPlatformMirror(row scanner) (*models.TerraformProviderPlatformMirror, error) {
	platformMirror := &models.TerraformProviderPlatformMirror{}

	fields := []interface{}{
		&platformMirror.Metadata.ID,
		&platformMirror.Metadata.CreationTimestamp,
		&platformMirror.Metadata.LastUpdatedTimestamp,
		&platformMirror.Metadata.Version,
		&platformMirror.OS,
		&platformMirror.Architecture,
		&platformMirror.VersionMirrorID,
	}

	err := row.Scan(fields...)
	if err != nil {
		return nil, err
	}

	return platformMirror, nil
}
