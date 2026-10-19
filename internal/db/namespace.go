package db

//go:generate go tool mockery --name Namespaces --inpackage --case underscore

import (
	"context"
	"database/sql"
	"strings"
	"unicode/utf8"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/attribute"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// Namespace holds a resolved namespace — exactly one of Group or Workspace is non-nil.
type Namespace struct {
	Type      models.NamespaceType
	Group     *models.Group
	Workspace *models.Workspace
}

// NamespaceSortableField represents the fields that a namespace listing can be sorted by.
type NamespaceSortableField string

const (
	// NamespaceSortableFieldFullPathAsc sorts by full path ascending.
	NamespaceSortableFieldFullPathAsc NamespaceSortableField = "FULL_PATH_ASC"
	// NamespaceSortableFieldFullPathDesc sorts by full path descending.
	NamespaceSortableFieldFullPathDesc NamespaceSortableField = "FULL_PATH_DESC"
)

func (n NamespaceSortableField) getSortDirection() pagination.SortDirection {
	if strings.HasSuffix(string(n), "_DESC") {
		return pagination.DescSort
	}
	return pagination.AscSort
}

// NamespaceFilter contains the supported filter fields for a namespace listing.
type NamespaceFilter struct {
	// ParentPath restricts results to the direct children of the group at this path.
	ParentPath *string
	// IncludeNested widens ParentPath to every descendant of the group rather than only its direct children.
	IncludeNested bool
	// Search is a case-insensitive contains match against: the path below ParentPath when IncludeNested is set;
	// the namespace's own name for the direct children of ParentPath; otherwise the full path.
	Search *string
	// RootNamespaceMemberships restricts to namespaces the caller has membership in.
	// A non-nil empty slice matches nothing; nil means no membership filter (e.g. admin).
	RootNamespaceMemberships []models.MembershipNamespace
	// TopLevel restricts to the top of the hierarchy: the root membership namespaces themselves
	// when RootNamespaceMemberships is set, otherwise groups without a parent.
	TopLevel bool
}

// GetNamespacesInput is the input for listing namespaces.
type GetNamespacesInput struct {
	Sort              *NamespaceSortableField
	PaginationOptions *pagination.Options
	Filter            *NamespaceFilter
}

// NamespacesResult contains the paginated response from GetNamespaces.
type NamespacesResult struct {
	PageInfo   *pagination.PageInfo
	Namespaces []Namespace
}

type namespaceRow struct {
	id          string
	path        string
	groupID     string
	workspaceID string
	version     int
}

var namespaceFieldList = []interface{}{"id", "version", "path", "group_id", "workspace_id"}

// Namespaces exposes namespace-level lookups backed by the namespaces table.
type Namespaces interface {
	// GetNamespace returns the group or workspace at the given path, or nil if there is none.
	GetNamespace(ctx context.Context, path string) (*Namespace, error)
	// GetNamespaces returns a paginated list of groups and workspaces sorted by path.
	GetNamespaces(ctx context.Context, input *GetNamespacesInput) (*NamespacesResult, error)
}

type namespaces struct {
	dbClient *Client
}

// NewNamespaces returns an instance of the Namespaces interface.
func NewNamespaces(dbClient *Client) Namespaces {
	return &namespaces{dbClient: dbClient}
}

func (n *namespaces) GetNamespace(ctx context.Context, path string) (*Namespace, error) {
	ctx, span := tracer.Start(ctx, "db.GetNamespace")
	span.SetAttributes(attribute.String("path", path))
	defer span.End()

	// Columns are in the order scanNamespace expects; the LEFT JOINs leave the side that doesn't match NULL
	fields := []interface{}{"namespaces.path"}
	fields = append(fields, qualifiedFields("groups", groupFieldList)...)
	fields = append(fields, qualifiedFields("workspaces", workspaceFieldList)...)

	sql, args, err := toSQLWithTag("namespace.GetNamespace", dialect.From(goqu.T("namespaces")).
		Prepared(true).
		Select(fields...).
		LeftJoin(goqu.T("groups"), goqu.On(goqu.I("namespaces.group_id").Eq(goqu.I("groups.id")))).
		LeftJoin(goqu.T("workspaces"), goqu.On(goqu.I("namespaces.workspace_id").Eq(goqu.I("workspaces.id")))).
		Where(goqu.Ex{"namespaces.path": path}))
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	namespace, err := scanNamespace(n.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return &namespace, nil
}

func (n *namespaces) GetNamespaces(ctx context.Context, input *GetNamespacesInput) (*NamespacesResult, error) {
	ctx, span := tracer.Start(ctx, "db.GetNamespaces")
	defer span.End()

	ex := goqu.And()

	if input.Filter != nil {
		if input.Filter.ParentPath != nil && !input.Filter.IncludeNested {
			// Drives the query from the children found through the parent_id and group_id indexes and looks their
			// namespaces up by key, rather than scanning every namespace in path order and filtering. An OR of two
			// IN subqueries on the namespaces table can't be turned into a join, so it forces that full scan.
			parentID := dialect.From("namespaces").Select("group_id").Where(goqu.I("path").Eq(*input.Filter.ParentPath))
			childGroups := dialect.From(goqu.T("groups").As("child")).
				Select("ns.id").
				Join(goqu.T("namespaces").As("ns"), goqu.On(goqu.I("ns.group_id").Eq(goqu.I("child.id")))).
				Where(goqu.I("child.parent_id").Eq(parentID))
			childWorkspaces := dialect.From(goqu.T("workspaces").As("child")).
				Select("ns.id").
				Join(goqu.T("namespaces").As("ns"), goqu.On(goqu.I("ns.workspace_id").Eq(goqu.I("child.id")))).
				Where(goqu.I("child.group_id").Eq(parentID))
			ex = ex.Append(goqu.I("namespaces.id").In(childGroups.UnionAll(childWorkspaces)))
		}

		if input.Filter.RootNamespaceMemberships != nil {
			if input.Filter.TopLevel {
				ex = ex.Append(topLevelMembershipFilter(input.Filter.RootNamespaceMemberships))
			} else {
				ex = ex.Append(membershipFilterByRootNamespaces(input.Filter.RootNamespaceMemberships))
			}
		} else if input.Filter.TopLevel {
			ex = ex.Append(
				goqu.I("namespaces.group_id").IsNotNull(),
				goqu.I("groups.parent_id").IsNull(),
			)
		}

		if input.Filter.ParentPath != nil && input.Filter.IncludeNested {
			ex = ex.Append(goqu.I("namespaces.path").Like(escapeLikePattern(*input.Filter.ParentPath) + "/%"))
		}

		if input.Filter.Search != nil && *input.Filter.Search != "" {
			ex = ex.Append(namespaceSearchFilter(*input.Filter.Search, input.Filter.ParentPath, input.Filter.IncludeNested))
		}
	}

	// Columns are in the order scanNamespace expects; the LEFT JOINs leave the side that doesn't match NULL
	fields := []interface{}{"namespaces.path"}
	fields = append(fields, qualifiedFields("groups", groupFieldList)...)
	fields = append(fields, qualifiedFields("workspaces", workspaceFieldList)...)

	query := dialect.From(goqu.T("namespaces")).
		Select(fields...).
		LeftJoin(goqu.T("groups"), goqu.On(goqu.I("namespaces.group_id").Eq(goqu.I("groups.id")))).
		LeftJoin(goqu.T("workspaces"), goqu.On(goqu.I("namespaces.workspace_id").Eq(goqu.I("workspaces.id")))).
		Where(ex)

	sortDir := pagination.AscSort
	if input.Sort != nil {
		sortDir = input.Sort.getSortDirection()
	}

	// namespaces.path is unique — use it as the primary key with no secondary sort.
	qBuilder, err := pagination.NewPaginatedQueryBuilder(
		input.PaginationOptions,
		&pagination.FieldDescriptor{Key: "full_path", Table: "namespaces", Col: "path"},
		pagination.WithSortByField(nil, sortDir),
		pagination.WithQueryTag("namespaces.GetNamespaces"),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build query", errors.WithSpan(span))
	}

	rows, err := qBuilder.Execute(ctx, n.dbClient.getConnection(ctx), query)
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil && isInvalidIDViolation(pgErr) {
			return nil, ErrInvalidID
		}
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}
	defer rows.Close()

	results := []Namespace{}
	for rows.Next() {
		item, err := scanNamespace(rows)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan namespace row", errors.WithSpan(span))
		}
		results = append(results, item)
	}

	if err := rows.Finalize(&results); err != nil {
		if pgErr := asPgError(err); pgErr != nil && isInvalidIDViolation(pgErr) {
			return nil, ErrInvalidID
		}
		return nil, errors.Wrap(err, "failed to finalize rows", errors.WithSpan(span))
	}

	return &NamespacesResult{
		PageInfo:   rows.GetPageInfo(),
		Namespaces: results,
	}, nil
}

func getNamespaceByGroupID(ctx context.Context, conn connection, groupID string) (*namespaceRow, error) {
	ctx, span := tracer.Start(ctx, "db.getNamespaceByGroupID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	return getNamespace(ctx, conn, goqu.Ex{"group_id": groupID})
}

func getNamespaceByWorkspaceID(ctx context.Context, conn connection, workspaceID string) (*namespaceRow, error) {
	ctx, span := tracer.Start(ctx, "db.getNamespaceByWorkspaceID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	return getNamespace(ctx, conn, goqu.Ex{"workspace_id": workspaceID})
}

func getNamespaceByPath(ctx context.Context, conn connection, path string) (*namespaceRow, error) {
	ctx, span := tracer.Start(ctx, "db.getNamespaceByPath")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	return getNamespace(ctx, conn, goqu.Ex{"path": path})
}

func getNamespace(ctx context.Context, conn connection, ex goqu.Ex) (*namespaceRow, error) {
	ctx, span := tracer.Start(ctx, "db.getNamespace")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	sql, args, err := toSQLWithTag("namespace.getNamespace", dialect.From("namespaces").
		Prepared(true).
		Select(namespaceFieldList...).
		Where(ex))
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	namespace, err := scanNamespaceRow(conn.QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return namespace, nil
}

// insertWithNamespace builds one statement that runs insert, creates the new row's namespace at path and
// selects the row joined to that namespace. The CTEs are named after their tables so selectFields
// (e.g. "groups.id", "namespaces.path") read the inserted rows; the base tables wouldn't show them yet,
// since every part of a statement sees the database as it was before the statement ran.
func insertWithNamespace(table, namespaceColumn, path string, insert *goqu.InsertDataset, selectFields []interface{}) *goqu.SelectDataset {
	return dialect.From(goqu.T(table)).
		Prepared(true).
		With(table, insert.Returning("*")).
		With("namespaces", dialect.Insert("namespaces").
			Cols("id", "version", "created_at", "updated_at", "path", namespaceColumn).
			FromQuery(dialect.From(goqu.T(table)).Select(
				goqu.L("?::uuid", newResourceID()),
				goqu.L("?::integer", initialResourceVersion),
				goqu.I(table+".created_at"),
				goqu.I(table+".updated_at"),
				goqu.L("?::varchar", path),
				goqu.I(table+".id"),
			)).
			Returning("*")).
		Select(selectFields...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.I("namespaces."+namespaceColumn).Eq(goqu.I(table+".id"))))
}

// migrateNamespaces migrates all namespaces that either exactly match an old path or have the old path as a prefix.
func migrateNamespaces(ctx context.Context, conn connection, oldPath, newPath string) error {
	ctx, span := tracer.Start(ctx, "db.migrateNamespaces")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	timestamp := currentTime()

	sql, args, err := toSQLWithTag("namespace.migrateNamespaces", dialect.Update("namespaces").
		Prepared(true).
		Set(
			goqu.Record{
				"version":    goqu.L("? + ?", goqu.C("version"), 1),
				"updated_at": timestamp,
				// Swap the old prefix for the new one literally. A regex replace would treat oldPath
				// as a pattern ('.' and similar match any character) and match it anywhere in the
				// path, not just at the start.
				"path": goqu.L("?::text || SUBSTRING(? FROM CHAR_LENGTH(?::text) + 1)", newPath, goqu.C("path"), oldPath),
			},
		).Where(goqu.Or(
		goqu.I("path").Eq(oldPath),
		// Escaped because '_' and '%' are LIKE wildcards and '_' is valid in a namespace name, so an
		// unescaped "a_b/%" would also match paths under "axb".
		goqu.I("path").Like(escapeLikePattern(oldPath)+"/%"),
	)).Returning(namespaceFieldList...))

	if err != nil {
		return errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	_, err = conn.Exec(ctx, sql, args...)
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) && pgErr.ConstraintName == "namespaces_path_key" {
				return errors.New("namespace %s already exists", newPath,
					errors.WithErrorCode(errors.EConflict), errors.WithSpan(span))
			}
		}
		return errors.Wrap(err, "failed to execute DB query", errors.WithSpan(span))
	}

	return nil
}

func scanNamespaceRow(row scanner) (*namespaceRow, error) {
	var groupID sql.NullString
	var workspaceID sql.NullString

	namespace := &namespaceRow{}

	err := row.Scan(
		&namespace.id,
		&namespace.version,
		&namespace.path,
		&groupID,
		&workspaceID,
	)

	if err != nil {
		return nil, err
	}

	if groupID.Valid {
		namespace.groupID = groupID.String
	}

	if workspaceID.Valid {
		namespace.workspaceID = workspaceID.String
	}

	return namespace, nil
}

// scanNamespace scans a single namespace row: the namespace path followed by every group column and every workspace column.
func scanNamespace(row scanner) (Namespace, error) {
	var fullPath string
	g := &groupRow{}
	w := &workspaceRow{}

	fields := append([]interface{}{&fullPath}, g.targets()...)
	if err := row.Scan(append(fields, w.targets()...)...); err != nil {
		return Namespace{}, err
	}

	if g.valid() {
		return Namespace{Type: models.NamespaceTypeGroup, Group: g.toModel(fullPath)}, nil
	}

	ws, err := w.toModel(fullPath)
	if err != nil {
		return Namespace{}, err
	}

	return Namespace{Type: models.NamespaceTypeWorkspace, Workspace: ws}, nil
}

// topLevelMembershipFilter matches only the root membership namespaces themselves, not their descendants.
func topLevelMembershipFilter(rootNamespaces []models.MembershipNamespace) exp.Expression {
	if len(rootNamespaces) == 0 {
		return goqu.L("false")
	}

	paths := make([]string, len(rootNamespaces))
	for i, ns := range rootNamespaces {
		paths[i] = ns.Path
	}

	return goqu.I("namespaces.path").In(paths)
}

// namespaceSearchFilter matches the term against the part of the path a user can't already infer from where they are
// searching, so a search within a group can't match every descendant through the group's own path segments: the path
// below parentPath when nested, the namespace's own name for direct children (their paths differ from the parent's
// only in the last segment), and the full path when there is no parent.
func namespaceSearchFilter(search string, parentPath *string, includeNested bool) exp.Expression {
	pattern := "%" + escapeLikePattern(search) + "%"

	if parentPath == nil {
		return goqu.I("namespaces.path").ILike(pattern)
	}

	if includeNested {
		// substr is 1-indexed; skip the parent path and the "/" after it
		return goqu.Func("substr", goqu.I("namespaces.path"), utf8.RuneCountInString(*parentPath)+2).ILike(pattern)
	}

	return goqu.Func("coalesce", goqu.I("groups.name"), goqu.I("workspaces.name")).ILike(pattern)
}
