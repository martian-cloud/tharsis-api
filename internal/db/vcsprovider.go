package db

//go:generate go tool mockery --name VCSProviders --inpackage --case underscore

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v4"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// VCSProviders encapsulates the logic to access VCS providers from the database.
type VCSProviders interface {
	GetProviderByID(ctx context.Context, id string) (*models.VCSProvider, error)
	GetProviderByTRN(ctx context.Context, trn string) (*models.VCSProvider, error)
	GetProviderByOAuthState(ctx context.Context, state string) (*models.VCSProvider, error)
	GetProviders(ctx context.Context, input *GetVCSProvidersInput) (*VCSProvidersResult, error)
	CreateProvider(ctx context.Context, provider *models.VCSProvider) (*models.VCSProvider, error)
	UpdateProvider(ctx context.Context, provider *models.VCSProvider) (*models.VCSProvider, error)
	DeleteProvider(ctx context.Context, provider *models.VCSProvider) error
}

// VCSProviderSortableField represents the fields that a VCS provider can be sorted by.
type VCSProviderSortableField string

// VCSProviderSortableField constants
const (
	VCSProviderSortableFieldCreatedAtAsc   VCSProviderSortableField = "CREATED_AT_ASC"
	VCSProviderSortableFieldCreatedAtDesc  VCSProviderSortableField = "CREATED_AT_DESC"
	VCSProviderSortableFieldUpdatedAtAsc   VCSProviderSortableField = "UPDATED_AT_ASC"
	VCSProviderSortableFieldUpdatedAtDesc  VCSProviderSortableField = "UPDATED_AT_DESC"
	VCSProviderSortableFieldGroupLevelAsc  VCSProviderSortableField = "GROUP_LEVEL_ASC"
	VCSProviderSortableFieldGroupLevelDesc VCSProviderSortableField = "GROUP_LEVEL_DESC"
)

func (sf VCSProviderSortableField) getFieldDescriptor() *pagination.FieldDescriptor {
	switch sf {
	case VCSProviderSortableFieldCreatedAtAsc, VCSProviderSortableFieldCreatedAtDesc:
		return &pagination.FieldDescriptor{Key: "created_at", Table: "vcs_providers", Col: "created_at"}
	case VCSProviderSortableFieldUpdatedAtAsc, VCSProviderSortableFieldUpdatedAtDesc:
		return &pagination.FieldDescriptor{Key: "updated_at", Table: "vcs_providers", Col: "updated_at"}
	case VCSProviderSortableFieldGroupLevelAsc, VCSProviderSortableFieldGroupLevelDesc:
		return &pagination.FieldDescriptor{Key: "group_path", Table: "namespaces", Col: "path"}
	default:
		return nil
	}
}

func (sf VCSProviderSortableField) getSortDirection() pagination.SortDirection {
	if strings.HasSuffix(string(sf), "_DESC") {
		return pagination.DescSort
	}
	return pagination.AscSort
}

func (sf VCSProviderSortableField) getTransformFunc() pagination.SortTransformFunc {
	switch sf {
	case VCSProviderSortableFieldGroupLevelAsc, VCSProviderSortableFieldGroupLevelDesc:
		return func(s string) string {
			return fmt.Sprintf("array_length(string_to_array(%s, '/'), 1)", s)
		}
	default:
		return nil
	}
}

// VCSProviderFilter contains the supported fields for filtering VCSProvider resources.
type VCSProviderFilter struct {
	Search         *string
	VCSProviderIDs []string
	NamespacePaths []string
}

// GetVCSProvidersInput is the input for listing VCS providers.
type GetVCSProvidersInput struct {
	// Sort specifies the field to sort on and direction
	Sort *VCSProviderSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// Filter is used to filter the results
	Filter *VCSProviderFilter
}

// VCSProvidersResult contains the response data and page information.
type VCSProvidersResult struct {
	PageInfo     *pagination.PageInfo
	VCSProviders []models.VCSProvider
}

type vcsProviders struct {
	dbClient *Client
}

var vcsProvidersFieldList = append(
	metadataFieldList,
	"created_by",
	"name",
	"description",
	"type",
	"url",
	"oauth_client_id",
	"oauth_client_secret",
	"oauth_state",
	"oauth_access_token",
	"oauth_refresh_token",
	"oauth_access_token_expires_at",
	"auto_create_webhooks",
	"group_id",
)

// NewVCSProviders returns an instance of the VCSProviders interface.
func NewVCSProviders(dbClient *Client) VCSProviders {
	return &vcsProviders{dbClient: dbClient}
}

func (vp *vcsProviders) GetProviderByID(ctx context.Context, id string) (*models.VCSProvider, error) {
	ctx, span := tracer.Start(ctx, "db.GetProviderByID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	return vp.getProvider(ctx, goqu.Ex{"vcs_providers.id": id})
}

func (vp *vcsProviders) GetProviderByTRN(ctx context.Context, trn string) (*models.VCSProvider, error) {
	ctx, span := tracer.Start(ctx, "db.GetProviderByTRN")
	defer span.End()

	path, err := types.VCSProviderModelType.ResourcePathFromTRN(trn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse TRN", errors.WithSpan(span))
	}

	lastSlashIndex := strings.LastIndex(path, "/")

	if lastSlashIndex == -1 {
		return nil, errors.New("a VCS provider TRN must have a group path and vcs provider name separated by a forward slash",
			errors.WithErrorCode(errors.EInvalid),
			errors.WithSpan(span),
		)
	}

	return vp.getProvider(ctx, goqu.Ex{
		"namespaces.path":    path[:lastSlashIndex],
		"vcs_providers.name": path[lastSlashIndex+1:],
	})
}

func (vp *vcsProviders) GetProviderByOAuthState(ctx context.Context, state string) (*models.VCSProvider, error) {
	ctx, span := tracer.Start(ctx, "db.GetProviderByOAuthState")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	return vp.getProvider(ctx, goqu.Ex{"vcs_providers.oauth_state": state})
}

func (vp *vcsProviders) GetProviders(ctx context.Context, input *GetVCSProvidersInput) (*VCSProvidersResult, error) {
	ctx, span := tracer.Start(ctx, "db.GetProviders")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	ex := goqu.And()

	if input.Filter != nil {
		if input.Filter.VCSProviderIDs != nil {
			ex = ex.Append(goqu.I("vcs_providers.id").In(input.Filter.VCSProviderIDs))
		}

		if input.Filter.NamespacePaths != nil {
			ex = ex.Append(goqu.I("namespaces.path").In(input.Filter.NamespacePaths))
		}

		if input.Filter.Search != nil {
			search := *input.Filter.Search

			lastDelimiterIndex := strings.LastIndex(search, "/")

			if lastDelimiterIndex != -1 {
				namespacePath := search[:lastDelimiterIndex]
				vcsProviderName := search[lastDelimiterIndex+1:]

				if vcsProviderName != "" {
					// An OR condition is used here since the last component of the search path could be part of
					// the namespace or it can be a VCS provider name prefix
					ex = ex.Append(
						goqu.Or(
							goqu.And(
								goqu.I("namespaces.path").Eq(namespacePath),
								goqu.I("vcs_providers.name").ILike(vcsProviderName+"%"),
							),
							goqu.Or(
								goqu.I("namespaces.path").ILike(search+"%"),
								goqu.I("vcs_providers.name").ILike(vcsProviderName+"%"),
							),
						),
					)
				} else {
					// We know the search is a namespace path since it ends with a "/"
					ex = ex.Append(goqu.I("namespaces.path").ILike(namespacePath + "%"))
				}
			} else {
				// We don't know if the search is for a namespace path or VCS provider name; therefore, use
				// an OR condition to search both
				ex = ex.Append(
					goqu.Or(
						goqu.I("namespaces.path").ILike(search+"%"),
						goqu.I("vcs_providers.name").ILike(search+"%"),
					),
				)
			}
		}

	}

	query := dialect.From("vcs_providers").
		Select(vp.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"vcs_providers.group_id": goqu.I("namespaces.group_id")})).
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
		&pagination.FieldDescriptor{Key: "id", Table: "vcs_providers", Col: "id"},
		pagination.WithSortByField(sortBy, sortDirection),
		pagination.WithSortByTransform(sortTransformFunc),
	)
	if err != nil {
		tracing.RecordError(span, err, "failed to build query")
		return nil, err
	}

	rows, err := qBuilder.Execute(ctx, vp.dbClient.getConnection(ctx), query)
	if err != nil {
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	defer rows.Close()

	// Scan rows
	results := []models.VCSProvider{}
	for rows.Next() {
		item, err := scanVCSProvider(rows)
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

	result := VCSProvidersResult{
		PageInfo:     rows.GetPageInfo(),
		VCSProviders: results,
	}

	return &result, nil
}

func (vp *vcsProviders) CreateProvider(ctx context.Context, provider *models.VCSProvider) (*models.VCSProvider, error) {
	ctx, span := tracer.Start(ctx, "db.CreateProvider")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	timestamp := currentTime()

	sql, args, err := dialect.From("vcs_providers").
		Prepared(true).
		With("vcs_providers",
			dialect.Insert("vcs_providers").
				Rows(goqu.Record{
					"id":                            newResourceID(),
					"version":                       initialResourceVersion,
					"created_at":                    timestamp,
					"updated_at":                    timestamp,
					"created_by":                    provider.CreatedBy,
					"name":                          provider.Name,
					"description":                   nullableString(provider.Description),
					"type":                          provider.Type,
					"url":                           provider.URL.String(),
					"oauth_client_id":               provider.OAuthClientID,
					"oauth_client_secret":           provider.OAuthClientSecret,
					"oauth_state":                   provider.OAuthState,
					"oauth_access_token":            provider.OAuthAccessToken,
					"oauth_refresh_token":           provider.OAuthRefreshToken,
					"oauth_access_token_expires_at": provider.OAuthAccessTokenExpiresAt,
					"auto_create_webhooks":          provider.AutoCreateWebhooks,
					"group_id":                      provider.GroupID,
				}).Returning("*"),
		).Select(vp.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"vcs_providers.group_id": goqu.I("namespaces.group_id")})).
		ToSQL()

	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	createdProvider, err := scanVCSProvider(vp.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				tracing.RecordError(span, nil, "vcs provider already exists in the specified group")
				return nil, errors.New("vcs provider already exists in the specified group", errors.WithErrorCode(errors.EConflict))
			}

			if isForeignKeyViolation(pgErr) && pgErr.ConstraintName == "fk_group_id" {
				tracing.RecordError(span, nil, "invalid group: the specified group does not exist")
				return nil, errors.New("invalid group: the specified group does not exist", errors.WithErrorCode(errors.EConflict))
			}
		}
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	return createdProvider, nil
}

func (vp *vcsProviders) UpdateProvider(ctx context.Context, provider *models.VCSProvider) (*models.VCSProvider, error) {
	ctx, span := tracer.Start(ctx, "db.UpdateProvider")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	timestamp := currentTime()

	sql, args, err := dialect.From("vcs_providers").
		Prepared(true).
		With("vcs_providers",
			dialect.Update("vcs_providers").
				Set(
					goqu.Record{
						"version":                       goqu.L("? + ?", goqu.C("version"), 1),
						"updated_at":                    timestamp,
						"description":                   nullableString(provider.Description),
						"oauth_client_id":               provider.OAuthClientID,
						"oauth_client_secret":           provider.OAuthClientSecret,
						"oauth_state":                   provider.OAuthState,
						"oauth_access_token":            provider.OAuthAccessToken,
						"oauth_refresh_token":           provider.OAuthRefreshToken,
						"oauth_access_token_expires_at": provider.OAuthAccessTokenExpiresAt,
					},
				).Where(goqu.Ex{"id": provider.Metadata.ID, "version": provider.Metadata.Version}).
				Returning("*"),
		).Select(vp.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"vcs_providers.group_id": goqu.I("namespaces.group_id")})).
		ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	updatedProvider, err := scanVCSProvider(vp.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return nil, ErrOptimisticLockError
		}
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	return updatedProvider, nil
}

func (vp *vcsProviders) DeleteProvider(ctx context.Context, provider *models.VCSProvider) error {
	ctx, span := tracer.Start(ctx, "db.DeleteProvider")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	sql, args, err := dialect.From("vcs_providers").
		Prepared(true).
		With("vcs_providers",
			dialect.Delete("vcs_providers").
				Where(
					goqu.Ex{
						"id":      provider.Metadata.ID,
						"version": provider.Metadata.Version,
					},
				).Returning("*"),
		).Select(vp.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"vcs_providers.group_id": goqu.I("namespaces.group_id")})).
		ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return err
	}

	if _, err := scanVCSProvider(vp.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...)); err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return ErrOptimisticLockError
		}

		if pgErr := asPgError(err); pgErr != nil {
			if isForeignKeyViolation(pgErr) && pgErr.ConstraintName == "fk_workspace_id" {
				tracing.RecordError(span, nil,
					"VCS provider %s has workspace configurations", provider.Name)
				return errors.New(
					"VCS provider %s has workspace configurations", provider.Name,
					errors.WithErrorCode(errors.EConflict),
				)
			}
		}

		tracing.RecordError(span, err, "failed to execute query")
		return err
	}

	return nil
}

func (vp *vcsProviders) getProvider(ctx context.Context, exp goqu.Ex) (*models.VCSProvider, error) {
	sql, args, err := dialect.From(goqu.T("vcs_providers")).
		Prepared(true).
		Select(vp.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"vcs_providers.group_id": goqu.I("namespaces.group_id")})).
		Where(exp).
		ToSQL()
	if err != nil {
		return nil, err
	}

	provider, err := scanVCSProvider(vp.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
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

	return provider, nil
}

func (vp *vcsProviders) getSelectFields() []interface{} {
	selectFields := []interface{}{}
	for _, field := range vcsProvidersFieldList {
		selectFields = append(selectFields, fmt.Sprintf("vcs_providers.%s", field))
	}

	selectFields = append(selectFields, "namespaces.path")

	return selectFields
}

func scanVCSProvider(row scanner) (*models.VCSProvider, error) {
	var description sql.NullString
	var providerURL string
	var namespacePath string
	vp := &models.VCSProvider{}

	fields := []interface{}{
		&vp.Metadata.ID,
		&vp.Metadata.CreationTimestamp,
		&vp.Metadata.LastUpdatedTimestamp,
		&vp.Metadata.Version,
		&vp.CreatedBy,
		&vp.Name,
		&description,
		&vp.Type,
		&providerURL,
		&vp.OAuthClientID,
		&vp.OAuthClientSecret,
		&vp.OAuthState,
		&vp.OAuthAccessToken,
		&vp.OAuthRefreshToken,
		&vp.OAuthAccessTokenExpiresAt,
		&vp.AutoCreateWebhooks,
		&vp.GroupID,
		&namespacePath,
	}

	err := row.Scan(fields...)
	if err != nil {
		return nil, err
	}

	if description.Valid {
		vp.Description = description.String
	}

	parsedURL, err := url.Parse(providerURL)
	if err != nil {
		return nil, err
	}
	vp.URL = *parsedURL

	vp.Metadata.TRN = types.VCSProviderModelType.BuildTRN(namespacePath, vp.Name)

	return vp, nil
}
