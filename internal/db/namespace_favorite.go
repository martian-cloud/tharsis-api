package db

//go:generate go tool mockery --name NamespaceFavorites --inpackage --case underscore

import (
	"context"
	"fmt"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// NamespaceFavorites encapsulates the logic to access namespace favorites from the database
type NamespaceFavorites interface {
	GetNamespaceFavoriteByID(ctx context.Context, id string) (*models.NamespaceFavorite, error)
	GetNamespaceFavoriteByTRN(ctx context.Context, trn string) (*models.NamespaceFavorite, error)
	GetNamespaceFavorites(ctx context.Context, input *GetNamespaceFavoritesInput) (*NamespaceFavoritesResult, error)
	CreateNamespaceFavorite(ctx context.Context, favorite *models.NamespaceFavorite) (*models.NamespaceFavorite, error)
	DeleteNamespaceFavorite(ctx context.Context, favorite *models.NamespaceFavorite) error
}

// NamespaceFavoriteSortableField represents the fields that a namespace favorite can be sorted by
type NamespaceFavoriteSortableField string

// NamespaceFavoriteSortableField constants
const (
	NamespaceFavoriteSortableFieldCreatedAtAsc  NamespaceFavoriteSortableField = "CREATED_AT_ASC"
	NamespaceFavoriteSortableFieldCreatedAtDesc NamespaceFavoriteSortableField = "CREATED_AT_DESC"
)

func (sf NamespaceFavoriteSortableField) getFieldDescriptor() *pagination.FieldDescriptor {
	switch sf {
	case NamespaceFavoriteSortableFieldCreatedAtAsc, NamespaceFavoriteSortableFieldCreatedAtDesc:
		return &pagination.FieldDescriptor{Key: "created_at", Table: "namespace_favorites", Col: "created_at"}
	default:
		return nil
	}
}

func (sf NamespaceFavoriteSortableField) getSortDirection() pagination.SortDirection {
	if strings.HasSuffix(string(sf), "_DESC") {
		return pagination.DescSort
	}
	return pagination.AscSort
}

func (sf NamespaceFavoriteSortableField) getValue() string {
	return string(sf)
}

// NamespaceFavoriteFilter contains the supported fields for filtering NamespaceFavorite resources
type NamespaceFavoriteFilter struct {
	NamespaceFavoriteIDs []string
	UserIDs              []string
	NamespacePath        *string
	Search               *string
}

// GetNamespaceFavoritesInput is the input for listing namespace favorites
type GetNamespaceFavoritesInput struct {
	// Sort specifies the field to sort on and direction
	Sort *NamespaceFavoriteSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// Filter is used to filter the results
	Filter *NamespaceFavoriteFilter
}

// NamespaceFavoritesResult contains the response data and page information
type NamespaceFavoritesResult struct {
	PageInfo           *pagination.PageInfo
	NamespaceFavorites []models.NamespaceFavorite
}

type namespaceFavorites struct {
	dbClient *Client
}

var namespaceFavoriteFieldList = append(metadataFieldList, "user_id", "group_id", "workspace_id")

// NewNamespaceFavorites returns an instance of the NamespaceFavorites interface
func NewNamespaceFavorites(dbClient *Client) NamespaceFavorites {
	return &namespaceFavorites{dbClient: dbClient}
}

func (f *namespaceFavorites) GetNamespaceFavoriteByID(ctx context.Context, id string) (*models.NamespaceFavorite, error) {
	ctx, span := tracer.Start(ctx, "db.GetNamespaceFavoriteByID")
	defer span.End()

	result, err := f.GetNamespaceFavorites(ctx, &GetNamespaceFavoritesInput{
		Filter: &NamespaceFavoriteFilter{NamespaceFavoriteIDs: []string{id}},
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to get namespace favorite")
		return nil, err
	}

	if len(result.NamespaceFavorites) == 0 {
		return nil, nil
	}

	return &result.NamespaceFavorites[0], nil
}

func (f *namespaceFavorites) GetNamespaceFavoriteByTRN(ctx context.Context, trn string) (*models.NamespaceFavorite, error) {
	ctx, span := tracer.Start(ctx, "db.GetNamespaceFavoriteByTRN")
	defer span.End()

	resourcePath, err := types.NamespaceFavoriteModelType.ResourcePathFromTRN(trn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse TRN", errors.WithSpan(span))
	}
	lastSlashIndex := strings.LastIndex(resourcePath, "/")
	if lastSlashIndex == -1 {
		return nil, errors.New("invalid TRN format: missing namespace path", errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
	}

	namespacePath := resourcePath[:lastSlashIndex]
	gidStr := resourcePath[lastSlashIndex+1:]
	parsedGID, err := gid.ParseGlobalID(gidStr)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse GID", errors.WithSpan(span))
	}

	result, err := f.GetNamespaceFavorites(ctx, &GetNamespaceFavoritesInput{
		Filter: &NamespaceFavoriteFilter{
			UserIDs:       []string{parsedGID.ID},
			NamespacePath: &namespacePath,
		},
	})
	if err != nil {
		return nil, err
	}

	if len(result.NamespaceFavorites) == 0 {
		return nil, nil
	}

	return &result.NamespaceFavorites[0], nil
}

func (f *namespaceFavorites) GetNamespaceFavorites(ctx context.Context, input *GetNamespaceFavoritesInput) (*NamespaceFavoritesResult, error) {
	ctx, span := tracer.Start(ctx, "db.GetNamespaceFavorites")
	defer span.End()

	ex := goqu.And()

	if input.Filter != nil {
		if input.Filter.NamespaceFavoriteIDs != nil {
			ex = ex.Append(goqu.I("namespace_favorites.id").In(input.Filter.NamespaceFavoriteIDs))
		}
		if input.Filter.UserIDs != nil {
			ex = ex.Append(goqu.I("namespace_favorites.user_id").In(input.Filter.UserIDs))
		}
		if input.Filter.NamespacePath != nil {
			ex = ex.Append(goqu.I("namespaces.path").Eq(*input.Filter.NamespacePath))
		}
		if input.Filter.Search != nil {
			ex = ex.Append(goqu.I("namespaces.path").ILike("%" + *input.Filter.Search + "%"))
		}
	}

	query := dialect.From(goqu.T("namespace_favorites")).
		Select(f.getSelectFields()...).
		LeftJoin(goqu.T("namespaces"), goqu.On(
			goqu.Or(
				goqu.I("namespace_favorites.group_id").Eq(goqu.I("namespaces.group_id")),
				goqu.I("namespace_favorites.workspace_id").Eq(goqu.I("namespaces.workspace_id")),
			),
		)).
		Where(ex)

	sortDirection := pagination.AscSort

	var sortBy *pagination.FieldDescriptor
	if input.Sort != nil {
		sortDirection = input.Sort.getSortDirection()
		sortBy = input.Sort.getFieldDescriptor()
	}

	qBuilder, err := pagination.NewPaginatedQueryBuilder(
		input.PaginationOptions,
		&pagination.FieldDescriptor{Key: "id", Table: "namespace_favorites", Col: "id"},
		pagination.WithSortByField(sortBy, sortDirection),
	)

	if err != nil {
		tracing.RecordError(span, err, "failed to build query")
		return nil, err
	}

	rows, err := qBuilder.Execute(ctx, f.dbClient.getConnection(ctx), query)
	if err != nil {
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	defer rows.Close()

	// Scan rows
	results := []models.NamespaceFavorite{}
	for rows.Next() {
		item, err := scanNamespaceFavorite(rows)
		if err != nil {
			tracing.RecordError(span, err, "failed to scan row")
			return nil, err
		}

		results = append(results, *item)
	}

	if err := rows.Err(); err != nil {
		tracing.RecordError(span, err, "error during row iteration")
		return nil, err
	}

	if err := rows.Finalize(&results); err != nil {
		tracing.RecordError(span, err, "failed to finalize rows")
		return nil, err
	}

	result := NamespaceFavoritesResult{
		PageInfo:           rows.GetPageInfo(),
		NamespaceFavorites: results,
	}

	return &result, nil
}

func (f *namespaceFavorites) CreateNamespaceFavorite(ctx context.Context, favorite *models.NamespaceFavorite) (*models.NamespaceFavorite, error) {
	ctx, span := tracer.Start(ctx, "db.CreateNamespaceFavorite")
	defer span.End()

	timestamp := currentTime()

	sql, args, err := dialect.From("namespace_favorites").
		Prepared(true).
		With("namespace_favorites",
			dialect.Insert("namespace_favorites").
				Rows(goqu.Record{
					"id":           newResourceID(),
					"version":      initialResourceVersion,
					"created_at":   timestamp,
					"updated_at":   timestamp,
					"user_id":      favorite.UserID,
					"group_id":     favorite.GroupID,
					"workspace_id": favorite.WorkspaceID,
				}).
				Returning("*"),
		).Select(f.getSelectFields()...).
		LeftJoin(goqu.T("namespaces"), goqu.On(
			goqu.Or(
				goqu.I("namespace_favorites.group_id").Eq(goqu.I("namespaces.group_id")),
				goqu.I("namespace_favorites.workspace_id").Eq(goqu.I("namespaces.workspace_id")),
			),
		)).
		ToSQL()

	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	createdNamespaceFavorite, err := scanNamespaceFavorite(f.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				return nil, errors.New("namespace favorite already exists", errors.WithErrorCode(errors.EConflict), errors.WithSpan(span))
			}
			if isForeignKeyViolation(pgErr) {
				switch pgErr.ConstraintName {
				case "fk_user_id":
					return nil, errors.New("user not found", errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
				default:
					return nil, errors.New("invalid foreign key", errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
				}
			}
		}

		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	return createdNamespaceFavorite, nil
}

func (f *namespaceFavorites) DeleteNamespaceFavorite(ctx context.Context, favorite *models.NamespaceFavorite) error {
	ctx, span := tracer.Start(ctx, "db.DeleteNamespaceFavorite")
	defer span.End()

	sql, args, err := dialect.Delete("namespace_favorites").
		Prepared(true).
		Where(
			goqu.Ex{
				"id":      favorite.Metadata.ID,
				"version": favorite.Metadata.Version,
			},
		).
		ToSQL()

	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return err
	}

	result, err := f.dbClient.getConnection(ctx).Exec(ctx, sql, args...)
	if err != nil {
		tracing.RecordError(span, err, "failed to execute query")
		return err
	}

	if result.RowsAffected() == 0 {
		tracing.RecordError(span, nil, "optimistic lock error")
		return ErrOptimisticLockError
	}

	return nil
}

func (f *namespaceFavorites) getSelectFields() []interface{} {
	selectFields := []interface{}{}

	for _, field := range namespaceFavoriteFieldList {
		selectFields = append(selectFields, fmt.Sprintf("namespace_favorites.%s", field))
	}
	selectFields = append(selectFields, "namespaces.path")

	return selectFields
}

func scanNamespaceFavorite(row scanner) (*models.NamespaceFavorite, error) {
	favorite := &models.NamespaceFavorite{}
	var namespacePath string

	if err := row.Scan(
		&favorite.Metadata.ID,
		&favorite.Metadata.CreationTimestamp,
		&favorite.Metadata.LastUpdatedTimestamp,
		&favorite.Metadata.Version,
		&favorite.UserID,
		&favorite.GroupID,
		&favorite.WorkspaceID,
		&namespacePath,
	); err != nil {
		return nil, err
	}

	favorite.Metadata.TRN = types.NamespaceFavoriteModelType.BuildTRN(namespacePath, gid.ToGlobalID(types.UserModelType, favorite.UserID))

	return favorite, nil
}
