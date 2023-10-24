package db

//go:generate mockery --name TerraformProviderVersionMirrors --inpackage --case underscore

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v4"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// TerraformProviderVersionMirrors encapsulates the logic to access Terraform provider version mirrors from the DB
type TerraformProviderVersionMirrors interface {
	GetVersionMirrorByID(ctx context.Context, id string) (*models.TerraformProviderVersionMirror, error)
	GetVersionMirrors(ctx context.Context, input *GetProviderVersionMirrorsInput) (*ProviderVersionMirrorsResult, error)
	CreateVersionMirror(ctx context.Context, versionMirror *models.TerraformProviderVersionMirror) (*models.TerraformProviderVersionMirror, error)
	DeleteVersionMirror(ctx context.Context, versionMirror *models.TerraformProviderVersionMirror) error
}

// TerraformProviderVersionMirrorSortableField represents fields
// that a TerraformProviderVersionMirror can be sorted by.
type TerraformProviderVersionMirrorSortableField string

// TerraformProviderVersionMirrorSortableField constants
const (
	TerraformProviderVersionMirrorSortableFieldCreatedAtAsc        TerraformProviderVersionMirrorSortableField = "CREATED_AT_ASC"
	TerraformProviderVersionMirrorSortableFieldCreatedAtDesc       TerraformProviderVersionMirrorSortableField = "CREATED_AT_DESC"
	TerraformProviderVersionMirrorSortableFieldSemanticVersionAsc  TerraformProviderVersionMirrorSortableField = "SEMANTIC_VERSION_ASC"
	TerraformProviderVersionMirrorSortableFieldSemanticVersionDesc TerraformProviderVersionMirrorSortableField = "SEMANTIC_VERSION_DESC"
)

func (ts TerraformProviderVersionMirrorSortableField) getFieldDescriptor() *pagination.FieldDescriptor {
	switch ts {
	case TerraformProviderVersionMirrorSortableFieldCreatedAtAsc, TerraformProviderVersionMirrorSortableFieldCreatedAtDesc:
		return &pagination.FieldDescriptor{Key: "created_at", Table: "terraform_provider_version_mirrors", Col: "created_at"}
	case TerraformProviderVersionMirrorSortableFieldSemanticVersionAsc, TerraformProviderVersionMirrorSortableFieldSemanticVersionDesc:
		return &pagination.FieldDescriptor{Key: "semantic_version", Table: "terraform_provider_version_mirrors", Col: "semantic_version"}
	default:
		return nil
	}
}

func (ts TerraformProviderVersionMirrorSortableField) getSortDirection() pagination.SortDirection {
	if strings.HasSuffix(string(ts), "_DESC") {
		return pagination.DescSort
	}
	return pagination.AscSort
}

// TerraformProviderVersionMirrorFilter represents fields a TerraformProviderVersionMirror can be filtered by.
type TerraformProviderVersionMirrorFilter struct {
	RegistryHostname  *string
	RegistryNamespace *string
	Type              *string
	SemanticVersion   *string
	GroupID           *string
	VersionMirrorIDs  []string
	NamespacePaths    []string
}

// GetProviderVersionMirrorsInput is the input for listing TerraformProviderVersionMirrors.
type GetProviderVersionMirrorsInput struct {
	// Sort specifies the field to sort on and direction
	Sort *TerraformProviderVersionMirrorSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// Filter is used to filter the results
	Filter *TerraformProviderVersionMirrorFilter
}

// ProviderVersionMirrorsResult contains the result of listing TerraformProviderVersionMirrors.
type ProviderVersionMirrorsResult struct {
	PageInfo       *pagination.PageInfo
	VersionMirrors []models.TerraformProviderVersionMirror
}

type terraformProviderVersionMirrors struct {
	dbClient *Client
}

var terraformProviderVersionMirrorFieldList = append(
	metadataFieldList,
	"created_by",
	"type",
	"semantic_version",
	"registry_namespace",
	"registry_hostname",
	"digests",
	"group_id",
)

// NewTerraformProviderVersionMirrors returns a new TerraformProviderVersionMirrors instance
func NewTerraformProviderVersionMirrors(dbClient *Client) TerraformProviderVersionMirrors {
	return &terraformProviderVersionMirrors{
		dbClient: dbClient,
	}
}

func (t *terraformProviderVersionMirrors) GetVersionMirrorByID(ctx context.Context, id string) (*models.TerraformProviderVersionMirror, error) {
	ctx, span := tracer.Start(ctx, "db.GetVersionMirrorByID")
	defer span.End()

	return t.getVersionMirror(ctx, goqu.Ex{"id": id})
}

func (t *terraformProviderVersionMirrors) GetVersionMirrors(ctx context.Context, input *GetProviderVersionMirrorsInput) (*ProviderVersionMirrorsResult, error) {
	ctx, span := tracer.Start(ctx, "db.GetVersionMirrors")
	defer span.End()

	ex := goqu.And()

	if input.Filter != nil {
		if len(input.Filter.NamespacePaths) > 0 {
			ex = ex.Append(goqu.I("namespaces.path").In(input.Filter.NamespacePaths))
		}
		if input.Filter.RegistryHostname != nil {
			ex = ex.Append(goqu.I("terraform_provider_version_mirrors.registry_hostname").Eq(*input.Filter.RegistryHostname))
		}
		if input.Filter.RegistryNamespace != nil {
			ex = ex.Append(goqu.I("terraform_provider_version_mirrors.registry_namespace").Eq(*input.Filter.RegistryNamespace))
		}
		if input.Filter.Type != nil {
			ex = ex.Append(goqu.I("terraform_provider_version_mirrors.type").Eq(*input.Filter.Type))
		}
		if input.Filter.SemanticVersion != nil {
			ex = ex.Append(goqu.I("terraform_provider_version_mirrors.semantic_version").Eq(*input.Filter.SemanticVersion))
		}
		if len(input.Filter.VersionMirrorIDs) > 0 {
			ex = ex.Append(goqu.I("terraform_provider_version_mirrors.id").In(input.Filter.VersionMirrorIDs))
		}
		if input.Filter.GroupID != nil {
			// GroupID is mainly for convenience as it avoids querying for a group prior to calling this function.
			ex = ex.Append(goqu.I("terraform_provider_version_mirrors.group_id").Eq(*input.Filter.GroupID))
		}
	}

	query := dialect.From(goqu.T("terraform_provider_version_mirrors")).
		Select(t.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"terraform_provider_version_mirrors.group_id": goqu.I("namespaces.group_id")})).
		Where(ex)

	sortDirection := pagination.AscSort

	var sortBy *pagination.FieldDescriptor
	if input.Sort != nil {
		sortDirection = input.Sort.getSortDirection()
		sortBy = input.Sort.getFieldDescriptor()
	}

	qBuilder, err := pagination.NewPaginatedQueryBuilder(
		input.PaginationOptions,
		&pagination.FieldDescriptor{Key: "id", Table: "terraform_provider_version_mirrors", Col: "id"},
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
	results := []models.TerraformProviderVersionMirror{}
	for rows.Next() {
		item, err := scanVersionMirror(rows)
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

	result := &ProviderVersionMirrorsResult{
		PageInfo:       rows.GetPageInfo(),
		VersionMirrors: results,
	}

	return result, nil
}

func (t *terraformProviderVersionMirrors) CreateVersionMirror(ctx context.Context, versionMirror *models.TerraformProviderVersionMirror) (*models.TerraformProviderVersionMirror, error) {
	ctx, span := tracer.Start(ctx, "db.CreateVersionMirror")
	defer span.End()

	timestamp := currentTime()

	digests, err := json.Marshal(versionMirror.Digests)
	if err != nil {
		tracing.RecordError(span, err, "failed to marshal provider version mirror digests")
		return nil, err
	}

	sql, args, err := dialect.Insert("terraform_provider_version_mirrors").
		Prepared(true).
		Rows(goqu.Record{
			"id":                 newResourceID(),
			"version":            initialResourceVersion,
			"created_at":         timestamp,
			"updated_at":         timestamp,
			"created_by":         versionMirror.CreatedBy,
			"type":               versionMirror.Type,
			"semantic_version":   versionMirror.SemanticVersion,
			"registry_namespace": versionMirror.RegistryNamespace,
			"registry_hostname":  versionMirror.RegistryHostname,
			"digests":            digests,
			"group_id":           versionMirror.GroupID,
		}).
		Returning(terraformProviderVersionMirrorFieldList...).ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	createdMirror, err := scanVersionMirror(t.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				tracing.RecordError(span, nil, "terraform provider version is already mirrored")
				return nil, errors.New("terraform provider version is already mirrored", errors.WithErrorCode(errors.EConflict))
			}
		}
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	return createdMirror, nil
}

func (t *terraformProviderVersionMirrors) DeleteVersionMirror(ctx context.Context, versionMirror *models.TerraformProviderVersionMirror) error {
	ctx, span := tracer.Start(ctx, "db.DeleteVersionMirror")
	defer span.End()

	sql, args, err := dialect.Delete("terraform_provider_version_mirrors").
		Prepared(true).
		Where(
			goqu.Ex{
				"id":      versionMirror.Metadata.ID,
				"version": versionMirror.Metadata.Version,
			},
		).Returning(terraformProviderVersionMirrorFieldList...).ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return err
	}

	if _, err = scanVersionMirror(t.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...)); err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return ErrOptimisticLockError
		}
		tracing.RecordError(span, err, "failed to execute query")
		return err
	}

	return nil
}

func (t *terraformProviderVersionMirrors) getVersionMirror(ctx context.Context, exp goqu.Ex) (*models.TerraformProviderVersionMirror, error) {
	query := dialect.From(goqu.T("terraform_provider_version_mirrors")).
		Prepared(true).
		Select(terraformProviderVersionMirrorFieldList...).
		Where(exp)

	sql, args, err := query.ToSQL()
	if err != nil {
		return nil, err
	}

	versionMirror, err := scanVersionMirror(t.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return versionMirror, nil
}

func (*terraformProviderVersionMirrors) getSelectFields() []interface{} {
	selectFields := []interface{}{}
	for _, field := range terraformProviderVersionMirrorFieldList {
		selectFields = append(selectFields, fmt.Sprintf("terraform_provider_version_mirrors.%s", field))
	}

	return selectFields
}

func scanVersionMirror(row scanner) (*models.TerraformProviderVersionMirror, error) {
	versionMirror := &models.TerraformProviderVersionMirror{}

	fields := []interface{}{
		&versionMirror.Metadata.ID,
		&versionMirror.Metadata.CreationTimestamp,
		&versionMirror.Metadata.LastUpdatedTimestamp,
		&versionMirror.Metadata.Version,
		&versionMirror.CreatedBy,
		&versionMirror.Type,
		&versionMirror.SemanticVersion,
		&versionMirror.RegistryNamespace,
		&versionMirror.RegistryHostname,
		&versionMirror.Digests,
		&versionMirror.GroupID,
	}

	err := row.Scan(fields...)
	if err != nil {
		return nil, err
	}

	return versionMirror, nil
}
