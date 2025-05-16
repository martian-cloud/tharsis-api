package db

//go:generate go tool mockery --name TerraformProviderPlatformMirrors --inpackage --case underscore

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

// TerraformProviderPlatformMirrors encapsulates the logic to access TerraformProviderPlatform from the DB.
type TerraformProviderPlatformMirrors interface {
	GetPlatformMirrorByID(ctx context.Context, id string) (*models.TerraformProviderPlatformMirror, error)
	GetPlatformMirrorByTRN(ctx context.Context, trn string) (*models.TerraformProviderPlatformMirror, error)
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

	return t.getPlatformMirror(ctx, goqu.Ex{"terraform_provider_platform_mirrors.id": id})
}

func (t *terraformProviderPlatformMirrors) GetPlatformMirrorByTRN(ctx context.Context, trn string) (*models.TerraformProviderPlatformMirror, error) {
	ctx, span := tracer.Start(ctx, "db.GetPlatformMirrorByTRN")
	defer span.End()

	path, err := types.TerraformProviderPlatformMirrorModelType.ResourcePathFromTRN(trn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse TRN", errors.WithSpan(span))
	}

	parts := strings.Split(path, "/")

	if len(parts) < 7 {
		return nil, errors.New("a Terraform provider platform mirror must have group path, hostname, namespace, type, semantic version, os, and arch separated by a forward slash",
			errors.WithErrorCode(errors.EInvalid),
			errors.WithSpan(span),
		)
	}

	return t.getPlatformMirror(ctx, goqu.Ex{
		"terraform_provider_platform_mirrors.architecture":      parts[len(parts)-1],
		"terraform_provider_platform_mirrors.os":                parts[len(parts)-2],
		"terraform_provider_version_mirrors.semantic_version":   parts[len(parts)-3],
		"terraform_provider_version_mirrors.type":               parts[len(parts)-4],
		"terraform_provider_version_mirrors.registry_namespace": parts[len(parts)-5],
		"terraform_provider_version_mirrors.registry_hostname":  parts[len(parts)-6],
		"namespaces.path": strings.Join(parts[:len(parts)-6], "/"),
	})
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
		Select(t.getSelectFields()...).
		InnerJoin(goqu.T("terraform_provider_version_mirrors"), goqu.On(goqu.I("terraform_provider_platform_mirrors.version_mirror_id").Eq(goqu.I("terraform_provider_version_mirrors.id")))).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.I("terraform_provider_version_mirrors.group_id").Eq(goqu.I("namespaces.group_id")))).
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

	sql, args, err := dialect.From("terraform_provider_platform_mirrors").
		Prepared(true).
		With("terraform_provider_platform_mirrors",
			dialect.Insert("terraform_provider_platform_mirrors").
				Rows(goqu.Record{
					"id":                newResourceID(),
					"version":           initialResourceVersion,
					"created_at":        timestamp,
					"updated_at":        timestamp,
					"os":                platformMirror.OS,
					"architecture":      platformMirror.Architecture,
					"version_mirror_id": platformMirror.VersionMirrorID,
				}).Returning("*"),
		).Select(t.getSelectFields()...).
		InnerJoin(goqu.T("terraform_provider_version_mirrors"), goqu.On(goqu.I("terraform_provider_platform_mirrors.version_mirror_id").Eq(goqu.I("terraform_provider_version_mirrors.id")))).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.I("terraform_provider_version_mirrors.group_id").Eq(goqu.I("namespaces.group_id")))).
		ToSQL()

	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return platformMirror, err
	}

	createdMirror, err := scanPlatformMirror(t.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				tracing.RecordError(span, nil, "terraform provider platform is already mirrored")
				return nil, errors.New("terraform provider platform is already mirrored", errors.WithErrorCode(errors.EConflict))
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

	sql, args, err := dialect.From("terraform_provider_platform_mirrors").
		Prepared(true).
		With("terraform_provider_platform_mirrors",
			dialect.Delete("terraform_provider_platform_mirrors").
				Where(
					goqu.Ex{
						"id":      platformMirror.Metadata.ID,
						"version": platformMirror.Metadata.Version,
					},
				).Returning("*"),
		).Select(t.getSelectFields()...).
		InnerJoin(goqu.T("terraform_provider_version_mirrors"), goqu.On(goqu.I("terraform_provider_platform_mirrors.version_mirror_id").Eq(goqu.I("terraform_provider_version_mirrors.id")))).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.I("terraform_provider_version_mirrors.group_id").Eq(goqu.I("namespaces.group_id")))).
		ToSQL()
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

func (t *terraformProviderPlatformMirrors) getPlatformMirror(ctx context.Context, exp goqu.Ex) (*models.TerraformProviderPlatformMirror, error) {
	query := dialect.From(goqu.T("terraform_provider_platform_mirrors")).
		Prepared(true).
		Select(t.getSelectFields()...).
		InnerJoin(goqu.T("terraform_provider_version_mirrors"), goqu.On(goqu.I("terraform_provider_platform_mirrors.version_mirror_id").Eq(goqu.I("terraform_provider_version_mirrors.id")))).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.I("terraform_provider_version_mirrors.group_id").Eq(goqu.I("namespaces.group_id")))).
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

		if pgErr := asPgError(err); pgErr != nil {
			if isInvalidIDViolation(pgErr) {
				return nil, ErrInvalidID
			}
		}

		return nil, err
	}

	return platformMirror, nil
}

func (*terraformProviderPlatformMirrors) getSelectFields() []interface{} {
	selectFields := []interface{}{}
	for _, field := range terraformProviderPlatformMirrorsFieldList {
		selectFields = append(selectFields, fmt.Sprintf("terraform_provider_platform_mirrors.%s", field))
	}

	selectFields = append(
		selectFields,
		"terraform_provider_version_mirrors.registry_hostname",
		"terraform_provider_version_mirrors.registry_namespace",
		"terraform_provider_version_mirrors.type",
		"terraform_provider_version_mirrors.semantic_version",
		"namespaces.path",
	)

	return selectFields
}

func scanPlatformMirror(row scanner) (*models.TerraformProviderPlatformMirror, error) {
	var namespacePath, registryHostname, registryNamespace, providerType, providerSemVersion string
	platformMirror := &models.TerraformProviderPlatformMirror{}

	fields := []interface{}{
		&platformMirror.Metadata.ID,
		&platformMirror.Metadata.CreationTimestamp,
		&platformMirror.Metadata.LastUpdatedTimestamp,
		&platformMirror.Metadata.Version,
		&platformMirror.OS,
		&platformMirror.Architecture,
		&platformMirror.VersionMirrorID,
		&registryHostname,
		&registryNamespace,
		&providerType,
		&providerSemVersion,
		&namespacePath,
	}

	err := row.Scan(fields...)
	if err != nil {
		return nil, err
	}

	platformMirror.Metadata.TRN = types.TerraformProviderPlatformMirrorModelType.BuildTRN(
		namespacePath,
		registryHostname,
		registryNamespace,
		providerType,
		providerSemVersion,
		platformMirror.OS,
		platformMirror.Architecture,
	)

	return platformMirror, nil
}
