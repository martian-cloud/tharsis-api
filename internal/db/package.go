package db

//go:generate go tool mockery --name Packages --inpackage --case underscore

import (
	"context"
	"fmt"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v5"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	nsutils "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/namespace/utils"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
)

// Packages encapsulates the logic to access packages from the database
type Packages interface {
	GetPackageByID(ctx context.Context, id string) (*models.Package, error)
	GetPackageByTRN(ctx context.Context, trnValue string) (*models.Package, error)
	GetPackages(ctx context.Context, input *GetPackagesInput) (*PackagesResult, error)
	CreatePackage(ctx context.Context, pkg *models.Package) (*models.Package, error)
	UpdatePackage(ctx context.Context, pkg *models.Package) (*models.Package, error)
	DeletePackage(ctx context.Context, pkg *models.Package) error
}

// PackageSortableField represents the fields that a package can be sorted by
type PackageSortableField string

// PackageSortableField constants
const (
	PackageSortableFieldNameAsc        PackageSortableField = "NAME_ASC"
	PackageSortableFieldNameDesc       PackageSortableField = "NAME_DESC"
	PackageSortableFieldUpdatedAtAsc   PackageSortableField = "UPDATED_AT_ASC"
	PackageSortableFieldUpdatedAtDesc  PackageSortableField = "UPDATED_AT_DESC"
	PackageSortableFieldGroupLevelAsc  PackageSortableField = "GROUP_LEVEL_ASC"
	PackageSortableFieldGroupLevelDesc PackageSortableField = "GROUP_LEVEL_DESC"
)

func (ps PackageSortableField) getFieldDescriptor() *pagination.FieldDescriptor {
	switch ps {
	case PackageSortableFieldNameAsc, PackageSortableFieldNameDesc:
		return &pagination.FieldDescriptor{Key: "name", Table: "packages", Col: "name"}
	case PackageSortableFieldUpdatedAtAsc, PackageSortableFieldUpdatedAtDesc:
		return &pagination.FieldDescriptor{Key: "updated_at", Table: "packages", Col: "updated_at"}
	case PackageSortableFieldGroupLevelAsc, PackageSortableFieldGroupLevelDesc:
		return &pagination.FieldDescriptor{Key: "group_path", Table: "namespaces", Col: "path"}
	default:
		return nil
	}
}

func (ps PackageSortableField) getSortDirection() pagination.SortDirection {
	if strings.HasSuffix(string(ps), "_DESC") {
		return pagination.DescSort
	}
	return pagination.AscSort
}

func (ps PackageSortableField) getTransformFunc() pagination.SortTransformFunc {
	switch ps {
	case PackageSortableFieldGroupLevelAsc, PackageSortableFieldGroupLevelDesc:
		return func(s string) string {
			return fmt.Sprintf("array_length(string_to_array(%s, '/'), 1)", s)
		}
	default:
		return nil
	}
}

// PackageFilter contains the supported fields for filtering Package resources
type PackageFilter struct {
	Search  *string
	GroupID *string
	Kind    *models.PackageKind
	// NamespacePaths scopes results to packages owned by one of these exact namespaces. Used for a
	// group listing that includes inherited packages: pass the group's ancestor-or-self paths so
	// packages owned by the group and its ancestors are returned (all of which are visible to it).
	NamespacePaths []string
	PackageIDs     []string
	// VisibleToGroup is a group's full path, scoping results to every package that group may
	// reference: a superset of NamespacePaths made up of packages owned by the group or one of its
	// ancestors, plus root_group-visibility packages sharing its root group, plus global packages
	// anywhere. It is the SQL mirror of core/packageregistry.IsPackageVisibleToNamespace. It takes the
	// path itself rather than pre-derived parts so that no caller can describe two different groups.
	VisibleToGroup *string
	// RootNamespaceMemberships restricts a global (cross-group) listing to packages the caller
	// can access: any package within one of the caller's root member namespace subtrees, plus
	// global-visibility packages (everywhere) and root_group-visibility packages that share a
	// root group with one of those memberships. Non-nil empty = no memberships (global only);
	// nil = no filter (e.g. admin).
	RootNamespaceMemberships []models.MembershipNamespace
}

// GetPackagesInput is the input for listing packages
type GetPackagesInput struct {
	// Sort specifies the field to sort on and direction
	Sort *PackageSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// Filter is used to filter the results
	Filter *PackageFilter
}

// PackagesResult contains the response data and page information
type PackagesResult struct {
	PageInfo *pagination.PageInfo
	Packages []models.Package
}

type packages struct {
	dbClient *Client
}

var packageFieldList = append(metadataFieldList,
	"name",
	"description",
	"group_id",
	"kind",
	"visibility",
	"created_by",
	"root_group_id",
	"allow_mutable_versions",
)

// NewPackages returns an instance of the Packages interface
func NewPackages(dbClient *Client) Packages {
	return &packages{dbClient: dbClient}
}

func (p *packages) GetPackageByID(ctx context.Context, id string) (*models.Package, error) {
	ctx, span := tracer.Start(ctx, "db.GetPackageByID")
	defer span.End()

	return p.getPackage(ctx, goqu.Ex{"packages.id": id})
}

func (p *packages) GetPackageByTRN(ctx context.Context, trnValue string) (*models.Package, error) {
	ctx, span := tracer.Start(ctx, "db.GetPackageByTRN")
	defer span.End()

	parsed, err := trn.TypePackage.Parse(trnValue)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse TRN", errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
	}

	if !parsed.HasParent() {
		return nil, errors.New("a package TRN must have a group path and package name separated by a forward slash",
			errors.WithErrorCode(errors.EInvalid),
			errors.WithSpan(span),
		)
	}

	return p.getPackage(ctx, goqu.Ex{
		"packages.name":   parsed.BaseName(),
		"namespaces.path": parsed.ParentPath(),
	})
}

func (p *packages) GetPackages(ctx context.Context, input *GetPackagesInput) (*PackagesResult, error) {
	ctx, span := tracer.Start(ctx, "db.GetPackages")
	defer span.End()

	ex := goqu.And()

	if input.Filter != nil {
		if input.Filter.PackageIDs != nil {
			ex = ex.Append(goqu.I("packages.id").In(input.Filter.PackageIDs))
		}
		if input.Filter.GroupID != nil {
			ex = ex.Append(goqu.I("packages.group_id").Eq(*input.Filter.GroupID))
		}
		if input.Filter.Kind != nil {
			ex = ex.Append(goqu.I("packages.kind").Eq(string(*input.Filter.Kind)))
		}
		if input.Filter.Search != nil && *input.Filter.Search != "" {
			// Search the fully-qualified package source rather than the bare name. namespaces.path is
			// the owning group path, so this CONCAT is exactly the "<groupPath>/<name>" source a policy
			// stores, which is what a listing that spans several groups displays and searches by.
			searchExp := goqu.L("CONCAT(namespaces.path, '/', packages.name)")
			ex = ex.Append(searchExp.ILike("%" + *input.Filter.Search + "%"))
		}
		if input.Filter.NamespacePaths != nil {
			ex = ex.Append(goqu.I("namespaces.path").In(input.Filter.NamespacePaths))
		}
		if input.Filter.VisibleToGroup != nil {
			groupPath := *input.Filter.VisibleToGroup

			// The three ways a package can be visible to one group. This has to be a single OR group
			// rather than a reuse of NamespacePaths, which is ANDed into the predicate.
			ex = ex.Append(goqu.Or(
				// Owned by the group or one of its ancestors, whatever its visibility.
				goqu.I("namespaces.path").In(nsutils.ExpandPath(groupPath)),
				goqu.I("packages.visibility").Eq(string(models.PackageVisibilityGlobal)),
				goqu.And(
					goqu.I("packages.visibility").Eq(string(models.PackageVisibilityRootGroup)),
					goqu.I("packages.root_group_id").In(
						dialect.From(goqu.T("namespaces").As("root_ns")).
							Select("root_ns.group_id").
							Where(goqu.I("root_ns.path").Eq(nsutils.RootGroupPath(groupPath))),
					),
				),
			))
		}
		if input.Filter.RootNamespaceMemberships != nil {
			// This OR condition will find all global packages or packages in the membership groups
			visibilityOr := goqu.Or(
				goqu.I("packages.visibility").Eq(string(models.PackageVisibilityGlobal)),
				membershipFilterByRootNamespaces(input.Filter.RootNamespaceMemberships),
			)

			rootPaths := []string{}
			seen := map[string]struct{}{}
			for _, ns := range input.Filter.RootNamespaceMemberships {
				root := strings.Split(ns.Path, "/")[0]
				if _, ok := seen[root]; !ok {
					seen[root] = struct{}{}
					rootPaths = append(rootPaths, root)
				}
			}
			if len(rootPaths) > 0 {
				// This condition will find all packages with root group visibility that match the root path of the namespace memberships
				visibilityOr = visibilityOr.Append(
					goqu.And(
						goqu.I("packages.visibility").Eq(string(models.PackageVisibilityRootGroup)),
						goqu.I("packages.root_group_id").In(
							dialect.From(goqu.T("namespaces").As("root_ns")).
								Select("root_ns.group_id").
								Where(goqu.I("root_ns.path").In(rootPaths)),
						),
					),
				)
			}
			ex = ex.Append(visibilityOr)
		}
	}

	query := dialect.From(goqu.T("packages")).
		Select(p.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"packages.group_id": goqu.I("namespaces.group_id")})).
		Where(ex)

	sortDirection := pagination.AscSort

	var sortBy *pagination.FieldDescriptor
	var sortTransformFunc pagination.SortTransformFunc
	if input.Sort != nil {
		sortDirection = input.Sort.getSortDirection()
		sortBy = input.Sort.getFieldDescriptor()
		sortTransformFunc = input.Sort.getTransformFunc()
	}

	qBuilder, err := pagination.NewPaginatedQueryBuilder(
		input.PaginationOptions,
		&pagination.FieldDescriptor{Key: "id", Table: "packages", Col: "id"},
		pagination.WithSortByField(sortBy, sortDirection),
		pagination.WithSortByTransform(sortTransformFunc),
		pagination.WithQueryTag("pkg.GetPackages"),
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
	results := []models.Package{}
	for rows.Next() {
		item, err := scanPackage(rows)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan row", errors.WithSpan(span))
		}

		results = append(results, *item)
	}

	if err := rows.Finalize(&results); err != nil {
		return nil, errors.Wrap(err, "failed to finalize rows", errors.WithSpan(span))
	}

	result := PackagesResult{
		PageInfo: rows.GetPageInfo(),
		Packages: results,
	}

	return &result, nil
}

func (p *packages) CreatePackage(ctx context.Context, pkg *models.Package) (*models.Package, error) {
	ctx, span := tracer.Start(ctx, "db.CreatePackage")
	defer span.End()

	timestamp := currentTime()

	sql, args, err := toSQLWithTag("pkg.CreatePackage", dialect.From("packages").
		Prepared(true).
		With("packages",
			dialect.Insert("packages").
				Rows(goqu.Record{
					"id":                     newResourceID(),
					"version":                initialResourceVersion,
					"created_at":             timestamp,
					"updated_at":             timestamp,
					"name":                   pkg.Name,
					"description":            pkg.Description,
					"group_id":               pkg.GroupID,
					"root_group_id":          pkg.RootGroupID,
					"kind":                   pkg.Kind,
					"visibility":             pkg.Visibility,
					"created_by":             pkg.CreatedBy,
					"allow_mutable_versions": pkg.AllowMutableVersions,
				}).Returning("*"),
		).Select(p.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"packages.group_id": goqu.I("namespaces.group_id")})))
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	createdPackage, err := scanPackage(p.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				return nil, errors.New("package with name %s already exists in the specified group", pkg.Name, errors.WithErrorCode(errors.EConflict), errors.WithSpan(span))
			}
			if isForeignKeyViolation(pgErr) {
				// group_id and root_group_id both reference groups; name the one that failed so the
				// caller gets ENotFound (matching CreatePolicy/CreateRunGate) instead of an opaque
				// internal error.
				switch pgErr.ConstraintName {
				case "fk_packages_root_group_id":
					return nil, errors.New("root group does not exist", errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
				default:
					return nil, errors.New("owner group does not exist", errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
				}
			}
		}
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return createdPackage, nil
}

func (p *packages) UpdatePackage(ctx context.Context, pkg *models.Package) (*models.Package, error) {
	ctx, span := tracer.Start(ctx, "db.UpdatePackage")
	defer span.End()

	timestamp := currentTime()

	sql, args, err := toSQLWithTag("pkg.UpdatePackage", dialect.From("packages").
		Prepared(true).
		With("packages",
			dialect.Update("packages").
				Set(
					goqu.Record{
						"version":                goqu.L("? + ?", goqu.C("version"), 1),
						"updated_at":             timestamp,
						"description":            pkg.Description,
						"visibility":             pkg.Visibility,
						"allow_mutable_versions": pkg.AllowMutableVersions,
					},
				).Where(goqu.Ex{"id": pkg.Metadata.ID, "version": pkg.Metadata.Version}).
				Returning("*"),
		).Select(p.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"packages.group_id": goqu.I("namespaces.group_id")})))
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	updatedPackage, err := scanPackage(p.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrOptimisticLockError
		}
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return updatedPackage, nil
}

func (p *packages) DeletePackage(ctx context.Context, pkg *models.Package) error {
	ctx, span := tracer.Start(ctx, "db.DeletePackage")
	defer span.End()

	sql, args, err := toSQLWithTag("pkg.DeletePackage", dialect.From("packages").
		Prepared(true).
		With("packages",
			dialect.Delete("packages").
				Where(
					goqu.Ex{
						"id":      pkg.Metadata.ID,
						"version": pkg.Metadata.Version,
					},
				).Returning("*"),
		).Select(p.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"packages.group_id": goqu.I("namespaces.group_id")})))
	if err != nil {
		return errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	if _, err := scanPackage(p.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...)); err != nil {
		if err == pgx.ErrNoRows {
			return ErrOptimisticLockError
		}
		return errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return nil
}

func (p *packages) getPackage(ctx context.Context, exp goqu.Ex) (*models.Package, error) {
	query := dialect.From(goqu.T("packages")).
		Prepared(true).
		Select(p.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"packages.group_id": goqu.I("namespaces.group_id")})).
		Where(exp)

	sql, args, err := toSQLWithTag("pkg.getPackage", query)
	if err != nil {
		return nil, err
	}

	pkg, err := scanPackage(p.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
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

	return pkg, nil
}

func (p *packages) getSelectFields() []interface{} {
	selectFields := []interface{}{}
	for _, field := range packageFieldList {
		selectFields = append(selectFields, fmt.Sprintf("packages.%s", field))
	}

	selectFields = append(selectFields, "namespaces.path")

	return selectFields
}

func scanPackage(row scanner) (*models.Package, error) {
	var groupPath string
	pkg := &models.Package{}

	fields := []interface{}{
		&pkg.Metadata.ID,
		&pkg.Metadata.CreationTimestamp,
		&pkg.Metadata.LastUpdatedTimestamp,
		&pkg.Metadata.Version,
		&pkg.Name,
		&pkg.Description,
		&pkg.GroupID,
		&pkg.Kind,
		&pkg.Visibility,
		&pkg.CreatedBy,
		&pkg.RootGroupID,
		&pkg.AllowMutableVersions,
		&groupPath,
	}

	err := row.Scan(fields...)
	if err != nil {
		return nil, err
	}

	pkg.Metadata.TRN = trn.TypePackage.Build(groupPath, pkg.Name)

	return pkg, nil
}
