package db

//go:generate go tool mockery --name TerraformProviderVersions --inpackage --case underscore

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v4"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// TerraformProviderVersions encapsulates the logic to access terraform provider versions from the database
type TerraformProviderVersions interface {
	GetProviderVersionByID(ctx context.Context, id string) (*models.TerraformProviderVersion, error)
	GetProviderVersionByTRN(ctx context.Context, trn string) (*models.TerraformProviderVersion, error)
	GetProviderVersions(ctx context.Context, input *GetProviderVersionsInput) (*ProviderVersionsResult, error)
	CreateProviderVersion(ctx context.Context, providerVersion *models.TerraformProviderVersion) (*models.TerraformProviderVersion, error)
	UpdateProviderVersion(ctx context.Context, providerVersion *models.TerraformProviderVersion) (*models.TerraformProviderVersion, error)
	DeleteProviderVersion(ctx context.Context, providerVersion *models.TerraformProviderVersion) error
}

// TerraformProviderVersionSortableField represents the fields that a provider version can be sorted by
type TerraformProviderVersionSortableField string

// TerraformProviderVersionSortableField constants
const (
	// TODO: remove version sortable field
	TerraformProviderVersionSortableFieldVersionAsc    TerraformProviderVersionSortableField = "VERSION_ASC"
	TerraformProviderVersionSortableFieldVersionDesc   TerraformProviderVersionSortableField = "VERSION_DESC"
	TerraformProviderVersionSortableFieldUpdatedAtAsc  TerraformProviderVersionSortableField = "UPDATED_AT_ASC"
	TerraformProviderVersionSortableFieldUpdatedAtDesc TerraformProviderVersionSortableField = "UPDATED_AT_DESC"
	TerraformProviderVersionSortableFieldCreatedAtAsc  TerraformProviderVersionSortableField = "CREATED_AT_ASC"
	TerraformProviderVersionSortableFieldCreatedAtDesc TerraformProviderVersionSortableField = "CREATED_AT_DESC"
)

func (ts TerraformProviderVersionSortableField) getFieldDescriptor() *pagination.FieldDescriptor {
	switch ts {
	case TerraformProviderVersionSortableFieldVersionAsc, TerraformProviderVersionSortableFieldVersionDesc:
		return &pagination.FieldDescriptor{Key: "sem_version", Table: "terraform_provider_versions", Col: "provider_sem_version"}
	case TerraformProviderVersionSortableFieldUpdatedAtAsc, TerraformProviderVersionSortableFieldUpdatedAtDesc:
		return &pagination.FieldDescriptor{Key: "updated_at", Table: "terraform_provider_versions", Col: "updated_at"}
	case TerraformProviderVersionSortableFieldCreatedAtAsc, TerraformProviderVersionSortableFieldCreatedAtDesc:
		return &pagination.FieldDescriptor{Key: "created_at", Table: "terraform_provider_versions", Col: "created_at"}
	default:
		return nil
	}
}

func (ts TerraformProviderVersionSortableField) getSortDirection() pagination.SortDirection {
	if strings.HasSuffix(string(ts), "_DESC") {
		return pagination.DescSort
	}
	return pagination.AscSort
}

// TerraformProviderVersionFilter contains the supported fields for filtering TerraformProviderVersion resources
type TerraformProviderVersionFilter struct {
	TimeRangeStart           *time.Time
	ProviderID               *string
	SHASumsUploaded          *bool
	SHASumsSignatureUploaded *bool
	SemanticVersion          *string
	Latest                   *bool
	ProviderVersionIDs       []string
}

// GetProviderVersionsInput is the input for listing terraform provider versions
type GetProviderVersionsInput struct {
	// Sort specifies the field to sort on and direction
	Sort *TerraformProviderVersionSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// Filter is used to filter the results
	Filter *TerraformProviderVersionFilter
}

// ProviderVersionsResult contains the response data and page information
type ProviderVersionsResult struct {
	PageInfo         *pagination.PageInfo
	ProviderVersions []models.TerraformProviderVersion
}

type terraformProviderVersions struct {
	dbClient *Client
}

var providerVersionFieldList = append(metadataFieldList, "provider_id", "provider_sem_version", "gpg_key_id", "gpg_ascii_armor", "protocols", "sha_sums_uploaded", "sha_sums_sig_uploaded", "readme_uploaded", "latest", "created_by")

// NewTerraformProviderVersions returns an instance of the TerraformProviderVersions interface
func NewTerraformProviderVersions(dbClient *Client) TerraformProviderVersions {
	return &terraformProviderVersions{dbClient: dbClient}
}

func (t *terraformProviderVersions) GetProviderVersionByID(ctx context.Context, id string) (*models.TerraformProviderVersion, error) {
	ctx, span := tracer.Start(ctx, "db.GetProviderVersionByID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	return t.getProviderVersion(ctx, goqu.Ex{"terraform_provider_versions.id": id})
}

func (t *terraformProviderVersions) GetProviderVersionByTRN(ctx context.Context, trn string) (*models.TerraformProviderVersion, error) {
	ctx, span := tracer.Start(ctx, "db.GetProviderVersionByTRN")
	defer span.End()

	path, err := types.TerraformProviderVersionModelType.ResourcePathFromTRN(trn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse TRN", errors.WithSpan(span))
	}

	parts := strings.Split(path, "/")

	if len(parts) < 3 {
		return nil, errors.New("a Terraform Provider version TRN must have group path, provider name and semver separated by a forward slash",
			errors.WithErrorCode(errors.EInvalid),
			errors.WithSpan(span),
		)
	}

	return t.getProviderVersion(ctx, goqu.Ex{
		"terraform_provider_versions.provider_sem_version": parts[len(parts)-1],
		"terraform_providers.name":                         parts[len(parts)-2],
		"namespaces.path":                                  strings.Join(parts[:len(parts)-2], "/"),
	})
}

func (t *terraformProviderVersions) GetProviderVersions(ctx context.Context, input *GetProviderVersionsInput) (*ProviderVersionsResult, error) {
	ctx, span := tracer.Start(ctx, "db.GetProviderVersions")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	ex := goqu.And()

	if input.Filter != nil {
		if input.Filter.ProviderVersionIDs != nil {
			ex = ex.Append(goqu.I("terraform_provider_versions.id").In(input.Filter.ProviderVersionIDs))
		}
		if input.Filter.ProviderID != nil {
			ex = ex.Append(goqu.I("terraform_provider_versions.provider_id").Eq(*input.Filter.ProviderID))
		}
		if input.Filter.SHASumsUploaded != nil {
			ex = ex.Append(goqu.I("terraform_provider_versions.sha_sums_uploaded").Eq(*input.Filter.SHASumsUploaded))
		}
		if input.Filter.SHASumsSignatureUploaded != nil {
			ex = ex.Append(goqu.I("terraform_provider_versions.sha_sums_sig_uploaded").Eq(*input.Filter.SHASumsSignatureUploaded))
		}
		if input.Filter.SemanticVersion != nil {
			ex = ex.Append(goqu.I("terraform_provider_versions.provider_sem_version").Eq(*input.Filter.SemanticVersion))
		}
		if input.Filter.Latest != nil {
			ex = ex.Append(goqu.I("terraform_provider_versions.latest").Eq(*input.Filter.Latest))
		}
		if input.Filter.TimeRangeStart != nil {
			// Must use UTC here otherwise, queries will return unexpected results.
			ex = ex.Append(goqu.I("terraform_provider_versions.created_at").Gte(input.Filter.TimeRangeStart.UTC()))
		}
	}

	query := dialect.From(goqu.T("terraform_provider_versions")).
		Select(t.getSelectFields()...).
		InnerJoin(goqu.T("terraform_providers"), goqu.On(goqu.I("terraform_providers.id").Eq(goqu.I("terraform_provider_versions.provider_id")))).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"terraform_providers.group_id": goqu.I("namespaces.group_id")})).
		Where(ex)

	sortDirection := pagination.AscSort

	var sortBy *pagination.FieldDescriptor
	if input.Sort != nil {
		sortDirection = input.Sort.getSortDirection()
		sortBy = input.Sort.getFieldDescriptor()
	}

	qBuilder, err := pagination.NewPaginatedQueryBuilder(
		input.PaginationOptions,
		&pagination.FieldDescriptor{Key: "id", Table: "terraform_provider_versions", Col: "id"},
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
	results := []models.TerraformProviderVersion{}
	for rows.Next() {
		item, err := scanTerraformProviderVersion(rows)
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

	result := ProviderVersionsResult{
		PageInfo:         rows.GetPageInfo(),
		ProviderVersions: results,
	}

	return &result, nil
}

func (t *terraformProviderVersions) CreateProviderVersion(ctx context.Context, providerVersion *models.TerraformProviderVersion) (*models.TerraformProviderVersion, error) {
	ctx, span := tracer.Start(ctx, "db.CreateProviderVersion")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	timestamp := currentTime()

	protocolsJSON, err := json.Marshal(providerVersion.Protocols)
	if err != nil {
		tracing.RecordError(span, err, "failed to marshal provider version protocols")
		return nil, err
	}

	sql, args, err := dialect.From("terraform_provider_versions").
		Prepared(true).
		With("terraform_provider_versions",
			dialect.Insert("terraform_provider_versions").
				Rows(goqu.Record{
					"id":                    newResourceID(),
					"version":               initialResourceVersion,
					"created_at":            timestamp,
					"updated_at":            timestamp,
					"provider_id":           providerVersion.ProviderID,
					"provider_sem_version":  providerVersion.SemanticVersion,
					"gpg_key_id":            providerVersion.GPGKeyID,
					"gpg_ascii_armor":       providerVersion.GPGASCIIArmor,
					"protocols":             protocolsJSON,
					"sha_sums_uploaded":     providerVersion.SHASumsUploaded,
					"sha_sums_sig_uploaded": providerVersion.SHASumsSignatureUploaded,
					"readme_uploaded":       providerVersion.ReadmeUploaded,
					"created_by":            providerVersion.CreatedBy,
					"latest":                providerVersion.Latest,
				}).Returning("*"),
		).Select(t.getSelectFields()...).
		InnerJoin(goqu.T("terraform_providers"), goqu.On(goqu.I("terraform_providers.id").Eq(goqu.I("terraform_provider_versions.provider_id")))).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"terraform_providers.group_id": goqu.I("namespaces.group_id")})).
		ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	createdTerraformProviderVersion, err := scanTerraformProviderVersion(t.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				tracing.RecordError(span, nil,
					"terraform provider version %s already exists", providerVersion.SemanticVersion)
				return nil, errors.New("terraform provider version %s already exists", providerVersion.SemanticVersion, errors.WithErrorCode(errors.EConflict))
			}
		}
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	return createdTerraformProviderVersion, nil
}

func (t *terraformProviderVersions) UpdateProviderVersion(ctx context.Context, providerVersion *models.TerraformProviderVersion) (*models.TerraformProviderVersion, error) {
	ctx, span := tracer.Start(ctx, "db.UpdateProviderVersion")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	timestamp := currentTime()

	protocolsJSON, err := json.Marshal(providerVersion.Protocols)
	if err != nil {
		tracing.RecordError(span, err, "failed to marshal provider version protocols")
		return nil, err
	}

	sql, args, err := dialect.From("terraform_provider_versions").
		Prepared(true).
		With("terraform_provider_versions",
			dialect.Update("terraform_provider_versions").
				Set(
					goqu.Record{
						"version":               goqu.L("? + ?", goqu.C("version"), 1),
						"updated_at":            timestamp,
						"gpg_key_id":            providerVersion.GPGKeyID,
						"gpg_ascii_armor":       providerVersion.GPGASCIIArmor,
						"protocols":             protocolsJSON,
						"sha_sums_uploaded":     providerVersion.SHASumsUploaded,
						"sha_sums_sig_uploaded": providerVersion.SHASumsSignatureUploaded,
						"readme_uploaded":       providerVersion.ReadmeUploaded,
						"latest":                providerVersion.Latest,
					},
				).Where(goqu.Ex{"id": providerVersion.Metadata.ID, "version": providerVersion.Metadata.Version}).
				Returning("*"),
		).Select(t.getSelectFields()...).
		InnerJoin(goqu.T("terraform_providers"), goqu.On(goqu.I("terraform_providers.id").Eq(goqu.I("terraform_provider_versions.provider_id")))).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"terraform_providers.group_id": goqu.I("namespaces.group_id")})).
		ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	updatedTerraformProviderVersion, err := scanTerraformProviderVersion(t.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return nil, ErrOptimisticLockError
		}
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	return updatedTerraformProviderVersion, nil
}

func (t *terraformProviderVersions) DeleteProviderVersion(ctx context.Context, providerVersion *models.TerraformProviderVersion) error {
	ctx, span := tracer.Start(ctx, "db.DeleteProviderVersion")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	sql, args, err := dialect.From("terraform_provider_versions").
		Prepared(true).
		With("terraform_provider_versions",
			dialect.Delete("terraform_provider_versions").
				Where(
					goqu.Ex{
						"id":      providerVersion.Metadata.ID,
						"version": providerVersion.Metadata.Version,
					},
				).Returning("*"),
		).Select(t.getSelectFields()...).
		InnerJoin(goqu.T("terraform_providers"), goqu.On(goqu.I("terraform_providers.id").Eq(goqu.I("terraform_provider_versions.provider_id")))).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"terraform_providers.group_id": goqu.I("namespaces.group_id")})).
		ToSQL()

	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return err
	}

	_, err = scanTerraformProviderVersion(t.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
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

func (t *terraformProviderVersions) getProviderVersion(ctx context.Context, exp goqu.Ex) (*models.TerraformProviderVersion, error) {
	query := dialect.From(goqu.T("terraform_provider_versions")).
		Prepared(true).
		Select(t.getSelectFields()...).
		InnerJoin(goqu.T("terraform_providers"), goqu.On(goqu.I("terraform_providers.id").Eq(goqu.I("terraform_provider_versions.provider_id")))).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"terraform_providers.group_id": goqu.I("namespaces.group_id")})).
		Where(exp)

	sql, args, err := query.ToSQL()
	if err != nil {
		return nil, err
	}

	providerVersion, err := scanTerraformProviderVersion(t.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
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

	return providerVersion, nil
}

func (t *terraformProviderVersions) getSelectFields() []interface{} {
	selectFields := []interface{}{}
	for _, field := range providerVersionFieldList {
		selectFields = append(selectFields, fmt.Sprintf("terraform_provider_versions.%s", field))
	}

	selectFields = append(selectFields,
		"namespaces.path",
		"terraform_providers.name",
	)

	return selectFields
}

func scanTerraformProviderVersion(row scanner) (*models.TerraformProviderVersion, error) {
	var groupPath, providerName string
	providerVersion := &models.TerraformProviderVersion{}

	fields := []interface{}{
		&providerVersion.Metadata.ID,
		&providerVersion.Metadata.CreationTimestamp,
		&providerVersion.Metadata.LastUpdatedTimestamp,
		&providerVersion.Metadata.Version,
		&providerVersion.ProviderID,
		&providerVersion.SemanticVersion,
		&providerVersion.GPGKeyID,
		&providerVersion.GPGASCIIArmor,
		&providerVersion.Protocols,
		&providerVersion.SHASumsUploaded,
		&providerVersion.SHASumsSignatureUploaded,
		&providerVersion.ReadmeUploaded,
		&providerVersion.Latest,
		&providerVersion.CreatedBy,
		&groupPath,
		&providerName,
	}

	err := row.Scan(fields...)
	if err != nil {
		return nil, err
	}

	providerVersion.Metadata.TRN = types.TerraformProviderVersionModelType.BuildTRN(
		groupPath,
		providerName,
		providerVersion.SemanticVersion,
	)

	return providerVersion, nil
}
