package db

//go:generate go tool mockery --name PackageVersions --inpackage --case underscore

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v5"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
)

// PackageVersions encapsulates the logic to access policy set versions from the database
type PackageVersions interface {
	GetPackageVersionByID(ctx context.Context, id string) (*models.PackageVersion, error)
	GetPackageVersionByTRN(ctx context.Context, trnValue string) (*models.PackageVersion, error)
	GetPackageVersions(ctx context.Context, input *GetPackageVersionsInput) (*PackageVersionsResult, error)
	CreatePackageVersion(ctx context.Context, packageVersion *models.PackageVersion) (*models.PackageVersion, error)
	UpdatePackageVersion(ctx context.Context, packageVersion *models.PackageVersion) (*models.PackageVersion, error)
	DeletePackageVersion(ctx context.Context, packageVersion *models.PackageVersion) error
}

// PackageVersionSortableField represents the fields that a policy set version can be sorted by
type PackageVersionSortableField string

// PackageVersionSortableField constants
const (
	PackageVersionSortableFieldUpdatedAtAsc  PackageVersionSortableField = "UPDATED_AT_ASC"
	PackageVersionSortableFieldUpdatedAtDesc PackageVersionSortableField = "UPDATED_AT_DESC"
	PackageVersionSortableFieldCreatedAtAsc  PackageVersionSortableField = "CREATED_AT_ASC"
	PackageVersionSortableFieldCreatedAtDesc PackageVersionSortableField = "CREATED_AT_DESC"
)

func (ps PackageVersionSortableField) getFieldDescriptor() *pagination.FieldDescriptor {
	switch ps {
	case PackageVersionSortableFieldUpdatedAtAsc, PackageVersionSortableFieldUpdatedAtDesc:
		return &pagination.FieldDescriptor{Key: "updated_at", Table: "package_versions", Col: "updated_at"}
	case PackageVersionSortableFieldCreatedAtAsc, PackageVersionSortableFieldCreatedAtDesc:
		return &pagination.FieldDescriptor{Key: "created_at", Table: "package_versions", Col: "created_at"}
	default:
		return nil
	}
}

func (ps PackageVersionSortableField) getSortDirection() pagination.SortDirection {
	if strings.HasSuffix(string(ps), "_DESC") {
		return pagination.DescSort
	}
	return pagination.AscSort
}

// PackageVersionFilter contains the supported fields for filtering PackageVersion resources
type PackageVersionFilter struct {
	TimeRangeStart    *time.Time
	PackageID         *string
	Status            *models.PackageVersionStatus
	SemanticVersion   *string
	Latest            *bool
	PackageVersionIDs []string
	Search            *string
}

// GetPackageVersionsInput is the input for listing policy set versions
type GetPackageVersionsInput struct {
	// Sort specifies the field to sort on and direction
	Sort *PackageVersionSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// Filter is used to filter the results
	Filter *PackageVersionFilter
}

// PackageVersionsResult contains the response data and page information
type PackageVersionsResult struct {
	PageInfo        *pagination.PageInfo
	PackageVersions []models.PackageVersion
}

type packageVersions struct {
	dbClient *Client
}

var packageVersionFieldList = append(
	metadataFieldList,
	"package_id",
	"semantic_version",
	"sha_sum",
	"status",
	"error",
	"object_store_key",
	"upload_started_at",
	"size",
	"latest",
	"created_by",
)

// NewPackageVersions returns an instance of the PackageVersions interface
func NewPackageVersions(dbClient *Client) PackageVersions {
	return &packageVersions{dbClient: dbClient}
}

func (p *packageVersions) GetPackageVersionByID(ctx context.Context, id string) (*models.PackageVersion, error) {
	ctx, span := tracer.Start(ctx, "db.GetPackageVersionByID")
	defer span.End()

	return p.getPackageVersion(ctx, goqu.Ex{"package_versions.id": id})
}

func (p *packageVersions) GetPackageVersionByTRN(ctx context.Context, trnValue string) (*models.PackageVersion, error) {
	ctx, span := tracer.Start(ctx, "db.GetPackageVersionByTRN")
	defer span.End()

	parsed, err := trn.TypePackageVersion.Parse(trnValue)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse TRN", errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
	}

	parts := parsed.PathParts()

	if len(parts) < 3 {
		return nil, errors.New("a policy set version TRN must have group path, policy set name, and semver separated by a forward slash",
			errors.WithErrorCode(errors.EInvalid),
			errors.WithSpan(span),
		)
	}

	return p.getPackageVersion(ctx, goqu.Ex{
		"package_versions.semantic_version": parts[len(parts)-1],
		"packages.name":                     parts[len(parts)-2],
		"namespaces.path":                   strings.Join(parts[:len(parts)-2], "/"),
	})
}

func (p *packageVersions) GetPackageVersions(ctx context.Context, input *GetPackageVersionsInput) (*PackageVersionsResult, error) {
	ctx, span := tracer.Start(ctx, "db.GetPackageVersions")
	defer span.End()

	ex := goqu.And()

	if input.Filter != nil {
		if input.Filter.PackageVersionIDs != nil {
			ex = ex.Append(goqu.I("package_versions.id").In(input.Filter.PackageVersionIDs))
		}
		if input.Filter.PackageID != nil {
			ex = ex.Append(goqu.I("package_versions.package_id").Eq(*input.Filter.PackageID))
		}
		if input.Filter.Status != nil {
			ex = ex.Append(goqu.I("package_versions.status").Eq(string(*input.Filter.Status)))
		}
		if input.Filter.SemanticVersion != nil {
			ex = ex.Append(goqu.I("package_versions.semantic_version").Eq(*input.Filter.SemanticVersion))
		}
		if input.Filter.Latest != nil {
			ex = ex.Append(goqu.I("package_versions.latest").Eq(*input.Filter.Latest))
		}
		if input.Filter.TimeRangeStart != nil {
			// Must use UTC here otherwise, queries will return unexpected results.
			ex = ex.Append(goqu.I("package_versions.created_at").Gte(input.Filter.TimeRangeStart.UTC()))
		}
		if input.Filter.Search != nil && *input.Filter.Search != "" {
			ex = ex.Append(goqu.I("package_versions.semantic_version").ILike("%" + *input.Filter.Search + "%"))
		}
	}

	query := dialect.From(goqu.T("package_versions")).
		Select(p.getSelectFields()...).
		InnerJoin(goqu.T("packages"), goqu.On(goqu.I("packages.id").Eq(goqu.I("package_versions.package_id")))).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"packages.group_id": goqu.I("namespaces.group_id")})).
		Where(ex)

	sortDirection := pagination.AscSort

	var sortBy *pagination.FieldDescriptor
	if input.Sort != nil {
		sortDirection = input.Sort.getSortDirection()
		sortBy = input.Sort.getFieldDescriptor()
	}

	qBuilder, err := pagination.NewPaginatedQueryBuilder(
		input.PaginationOptions,
		&pagination.FieldDescriptor{Key: "id", Table: "package_versions", Col: "id"},
		pagination.WithSortByField(sortBy, sortDirection),
		pagination.WithQueryTag("packageversion.GetPackageVersions"),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build query", errors.WithSpan(span))
	}

	rows, err := qBuilder.Execute(ctx, p.dbClient.getConnection(ctx), query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	defer rows.Close()

	// Scan rows
	results := []models.PackageVersion{}
	for rows.Next() {
		item, err := scanPackageVersion(rows)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan row", errors.WithSpan(span))
		}

		results = append(results, *item)
	}

	if err := rows.Finalize(&results); err != nil {
		return nil, errors.Wrap(err, "failed to finalize rows", errors.WithSpan(span))
	}

	result := PackageVersionsResult{
		PageInfo:        rows.GetPageInfo(),
		PackageVersions: results,
	}

	return &result, nil
}

func (p *packageVersions) CreatePackageVersion(ctx context.Context, packageVersion *models.PackageVersion) (*models.PackageVersion, error) {
	ctx, span := tracer.Start(ctx, "db.CreatePackageVersion")
	defer span.End()

	timestamp := currentTime()

	record := goqu.Record{
		"id":                newResourceID(),
		"version":           initialResourceVersion,
		"created_at":        timestamp,
		"updated_at":        timestamp,
		"package_id":        packageVersion.PackageID,
		"semantic_version":  packageVersion.SemanticVersion,
		"sha_sum":           packageVersion.SHASum,
		"status":            packageVersion.Status,
		"error":             packageVersion.Error,
		"object_store_key":  packageVersion.ObjectStoreKey,
		"upload_started_at": packageVersion.UploadStartedTimestamp,
		"size":              packageVersion.Size,
		"created_by":        packageVersion.CreatedBy,
		"latest":            packageVersion.Latest,
	}

	sql, args, err := toSQLWithTag("packageversion.CreatePackageVersion", dialect.From("package_versions").
		Prepared(true).
		With("package_versions",
			dialect.Insert("package_versions").
				Rows(record).
				Returning("*"),
		).Select(p.getSelectFields()...).
		InnerJoin(goqu.T("packages"), goqu.On(goqu.I("packages.id").Eq(goqu.I("package_versions.package_id")))).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"packages.group_id": goqu.I("namespaces.group_id")})))
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	createdPackageVersion, err := scanPackageVersion(p.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				switch pgErr.ConstraintName {
				case "index_package_versions_on_latest":
					return nil, errors.New("another policy set version is already marked as the latest for the same policy set", errors.WithErrorCode(errors.EConflict), errors.WithSpan(span))
				case "index_package_versions_on_semantic_version":
					return nil, errors.New("policy set version %s already exists", packageVersion.SemanticVersion, errors.WithErrorCode(errors.EConflict), errors.WithSpan(span))
				default:
					return nil, errors.New("database constraint violated: %s", pgErr.ConstraintName, errors.WithErrorCode(errors.EConflict), errors.WithSpan(span))
				}
			}
		}
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return createdPackageVersion, nil
}

func (p *packageVersions) UpdatePackageVersion(ctx context.Context, packageVersion *models.PackageVersion) (*models.PackageVersion, error) {
	ctx, span := tracer.Start(ctx, "db.UpdatePackageVersion")
	defer span.End()

	timestamp := currentTime()

	record := goqu.Record{
		"version":           goqu.L("? + ?", goqu.C("version"), 1),
		"updated_at":        timestamp,
		"sha_sum":           packageVersion.SHASum,
		"status":            packageVersion.Status,
		"error":             packageVersion.Error,
		"object_store_key":  packageVersion.ObjectStoreKey,
		"upload_started_at": packageVersion.UploadStartedTimestamp,
		"size":              packageVersion.Size,
		"latest":            packageVersion.Latest,
	}

	sql, args, err := toSQLWithTag("packageversion.UpdatePackageVersion", dialect.From("package_versions").
		Prepared(true).
		With("package_versions",
			dialect.Update("package_versions").
				Set(record).
				Where(goqu.Ex{"id": packageVersion.Metadata.ID, "version": packageVersion.Metadata.Version}).
				Returning("*"),
		).Select(p.getSelectFields()...).
		InnerJoin(goqu.T("packages"), goqu.On(goqu.I("packages.id").Eq(goqu.I("package_versions.package_id")))).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"packages.group_id": goqu.I("namespaces.group_id")})))
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	updatedPackageVersion, err := scanPackageVersion(p.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrOptimisticLockError
		}
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				switch pgErr.ConstraintName {
				case "index_package_versions_on_latest":
					return nil, errors.New("another policy set version is already marked as the latest for the same policy set", errors.WithErrorCode(errors.EConflict), errors.WithSpan(span))
				default:
					return nil, errors.New("database constraint violated: %s", pgErr.ConstraintName, errors.WithErrorCode(errors.EConflict), errors.WithSpan(span))
				}
			}
		}
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return updatedPackageVersion, nil
}

func (p *packageVersions) DeletePackageVersion(ctx context.Context, packageVersion *models.PackageVersion) error {
	ctx, span := tracer.Start(ctx, "db.DeletePackageVersion")
	defer span.End()

	sql, args, err := toSQLWithTag("packageversion.DeletePackageVersion", dialect.From("package_versions").
		Prepared(true).
		With("package_versions",
			dialect.Delete("package_versions").
				Where(goqu.Ex{
					"id":      packageVersion.Metadata.ID,
					"version": packageVersion.Metadata.Version,
				}).Returning("*"),
		).Select(p.getSelectFields()...).
		InnerJoin(goqu.T("packages"), goqu.On(goqu.I("packages.id").Eq(goqu.I("package_versions.package_id")))).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"packages.group_id": goqu.I("namespaces.group_id")})))
	if err != nil {
		return errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	_, err = scanPackageVersion(p.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return ErrOptimisticLockError
		}
		return errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return nil
}

func (p *packageVersions) getPackageVersion(ctx context.Context, exp goqu.Ex) (*models.PackageVersion, error) {
	query := dialect.From(goqu.T("package_versions")).
		Prepared(true).
		Select(p.getSelectFields()...).
		InnerJoin(goqu.T("packages"), goqu.On(goqu.I("packages.id").Eq(goqu.I("package_versions.package_id")))).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"packages.group_id": goqu.I("namespaces.group_id")})).
		Where(exp)

	sql, args, err := toSQLWithTag("packageversion.getPackageVersion", query)
	if err != nil {
		return nil, err
	}

	packageVersion, err := scanPackageVersion(p.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
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

	return packageVersion, nil
}

func (p *packageVersions) getSelectFields() []interface{} {
	selectFields := []interface{}{}
	for _, field := range packageVersionFieldList {
		selectFields = append(selectFields, fmt.Sprintf("package_versions.%s", field))
	}

	selectFields = append(selectFields,
		"namespaces.path",
		"packages.name",
	)

	return selectFields
}

func scanPackageVersion(row scanner) (*models.PackageVersion, error) {
	packageVersion := &models.PackageVersion{}

	var errorMessage sql.NullString
	var uploadStartedAt sql.NullTime
	var groupPath string
	var packageName string

	fields := []interface{}{
		&packageVersion.Metadata.ID,
		&packageVersion.Metadata.CreationTimestamp,
		&packageVersion.Metadata.LastUpdatedTimestamp,
		&packageVersion.Metadata.Version,
		&packageVersion.PackageID,
		&packageVersion.SemanticVersion,
		&packageVersion.SHASum,
		&packageVersion.Status,
		&errorMessage,
		&packageVersion.ObjectStoreKey,
		&uploadStartedAt,
		&packageVersion.Size,
		&packageVersion.Latest,
		&packageVersion.CreatedBy,
		&groupPath,
		&packageName,
	}

	err := row.Scan(fields...)
	if err != nil {
		return nil, err
	}

	if errorMessage.Valid {
		packageVersion.Error = &errorMessage.String
	}

	if uploadStartedAt.Valid {
		packageVersion.UploadStartedTimestamp = &uploadStartedAt.Time
	}

	packageVersion.Metadata.TRN = trn.TypePackageVersion.Build(
		groupPath,
		packageName,
		packageVersion.SemanticVersion,
	)

	return packageVersion, nil
}
