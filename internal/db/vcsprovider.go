package db

//go:generate mockery --name VCSProviders --inpackage --case underscore

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v4"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// VCSProviders encapsulates the logic to access VCS providers from the database.
type VCSProviders interface {
	GetProviderByID(ctx context.Context, id string) (*models.VCSProvider, error)
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
	VCSProviderSortableFieldCreatedAtAsc  VCSProviderSortableField = "CREATED_AT_ASC"
	VCSProviderSortableFieldCreatedAtDesc VCSProviderSortableField = "CREATED_AT_DESC"
	VCSProviderSortableFieldUpdatedAtAsc  VCSProviderSortableField = "UPDATED_AT_ASC"
	VCSProviderSortableFieldUpdatedAtDesc VCSProviderSortableField = "UPDATED_AT_DESC"
)

func (sf VCSProviderSortableField) getFieldDescriptor() *fieldDescriptor {
	switch sf {
	case VCSProviderSortableFieldCreatedAtAsc, VCSProviderSortableFieldCreatedAtDesc:
		return &fieldDescriptor{key: "created_at", table: "vcs_providers", col: "created_at"}
	case VCSProviderSortableFieldUpdatedAtAsc, VCSProviderSortableFieldUpdatedAtDesc:
		return &fieldDescriptor{key: "updated_at", table: "vcs_providers", col: "updated_at"}
	default:
		return nil
	}
}

func (sf VCSProviderSortableField) getSortDirection() SortDirection {
	if strings.HasSuffix(string(sf), "_DESC") {
		return DescSort
	}
	return AscSort
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
	PaginationOptions *PaginationOptions
	// Filter is used to filter the results
	Filter *VCSProviderFilter
}

// VCSProvidersResult contains the response data and page information.
type VCSProvidersResult struct {
	PageInfo     *PageInfo
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
	"hostname",
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
	return vp.getProvider(ctx, goqu.Ex{"vcs_providers.id": id})
}

func (vp *vcsProviders) GetProviderByOAuthState(ctx context.Context, state string) (*models.VCSProvider, error) {
	return vp.getProvider(ctx, goqu.Ex{"vcs_providers.oauth_state": state})
}

func (vp *vcsProviders) GetProviders(ctx context.Context, input *GetVCSProvidersInput) (*VCSProvidersResult, error) {
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
								goqu.I("vcs_providers.name").Like(vcsProviderName+"%%"),
							),
							goqu.Or(
								goqu.I("namespaces.path").Like(search+"%"),
								goqu.I("vcs_providers.name").Like(vcsProviderName+"%%"),
							),
						),
					)
				} else {
					// We know the search is a namespace path since it ends with a "/"
					ex = ex.Append(goqu.I("namespaces.path").Like(namespacePath + "%%"))
				}
			} else {
				// We don't know if the search is for a namespace path or VCS provider name; therefore, use
				// an OR condition to search both
				ex = ex.Append(
					goqu.Or(
						goqu.I("namespaces.path").Like(search+"%%"),
						goqu.I("vcs_providers.name").Like(search+"%%"),
					),
				)
			}
		}

	}

	query := dialect.From("vcs_providers").
		Select(vp.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"vcs_providers.group_id": goqu.I("namespaces.group_id")})).
		Where(ex)

	sortDirection := AscSort

	var sortBy *fieldDescriptor
	if input.Sort != nil {
		sortDirection = input.Sort.getSortDirection()
		sortBy = input.Sort.getFieldDescriptor()
	}

	qBuilder, err := newPaginatedQueryBuilder(
		input.PaginationOptions,
		&fieldDescriptor{key: "id", table: "vcs_providers", col: "id"},
		sortBy,
		sortDirection,
		vcsProviderFieldResolver,
	)

	if err != nil {
		return nil, err
	}

	rows, err := qBuilder.execute(ctx, vp.dbClient.getConnection(ctx), query)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	// Scan rows
	results := []models.VCSProvider{}
	for rows.Next() {
		item, err := scanVCSProvider(rows, true)
		if err != nil {
			return nil, err
		}

		results = append(results, *item)
	}

	if err := rows.finalize(&results); err != nil {
		return nil, err
	}

	result := VCSProvidersResult{
		PageInfo:     rows.getPageInfo(),
		VCSProviders: results,
	}

	return &result, nil
}

func (vp *vcsProviders) CreateProvider(ctx context.Context, provider *models.VCSProvider) (*models.VCSProvider, error) {
	timestamp := currentTime()

	tx, err := vp.dbClient.getConnection(ctx).Begin(ctx)
	if err != nil {
		return nil, err
	}

	// Rollback is safe to call even if the tx is already closed, so if
	// the tx commits successfully, this is a no-op
	defer func() {
		if txErr := tx.Rollback(ctx); txErr != nil && txErr != pgx.ErrTxClosed {
			vp.dbClient.logger.Errorf("failed to rollback tx for CreateProvider: %v", txErr)
		}
	}()

	sql, _, err := dialect.Insert("vcs_providers").
		Rows(goqu.Record{
			"id":                            newResourceID(),
			"version":                       initialResourceVersion,
			"created_at":                    timestamp,
			"updated_at":                    timestamp,
			"created_by":                    provider.CreatedBy,
			"name":                          provider.Name,
			"description":                   nullableString(provider.Description),
			"type":                          provider.Type,
			"hostname":                      provider.Hostname,
			"oauth_client_id":               provider.OAuthClientID,
			"oauth_client_secret":           provider.OAuthClientSecret,
			"oauth_state":                   provider.OAuthState,
			"oauth_access_token":            provider.OAuthAccessToken,
			"oauth_refresh_token":           provider.OAuthRefreshToken,
			"oauth_access_token_expires_at": provider.OAuthAccessTokenExpiresAt,
			"auto_create_webhooks":          provider.AutoCreateWebhooks,
			"group_id":                      provider.GroupID,
		}).
		Returning(vcsProvidersFieldList...).ToSQL()
	if err != nil {
		return nil, err
	}

	createdProvider, err := scanVCSProvider(tx.QueryRow(ctx, sql), false)
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				return nil, errors.NewError(errors.EConflict, "vcs provider already exists in the specified group")
			}

			if isForeignKeyViolation(pgErr) && pgErr.ConstraintName == "fk_group_id" {
				return nil, errors.NewError(errors.EConflict, "invalid group: the specified group does not exist")
			}
		}
		return nil, err
	}

	// Lookup namespace for group
	namespace, err := getNamespaceByGroupID(ctx, tx, createdProvider.GroupID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	createdProvider.ResourcePath = buildVCSProviderResourcePath(namespace.path, createdProvider.Name)

	return createdProvider, nil
}

func (vp *vcsProviders) UpdateProvider(ctx context.Context, provider *models.VCSProvider) (*models.VCSProvider, error) {
	timestamp := currentTime()

	tx, err := vp.dbClient.getConnection(ctx).Begin(ctx)
	if err != nil {
		return nil, err
	}

	// Rollback is safe to call even if the tx is already closed, so if
	// the tx commits successfully, this is a no-op
	defer func() {
		if txErr := tx.Rollback(ctx); txErr != nil && txErr != pgx.ErrTxClosed {
			vp.dbClient.logger.Errorf("failed to rollback tx for UpdateProvider: %v", txErr)
		}
	}()

	sql, _, err := dialect.Update("vcs_providers").Set(
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
		Returning(vcsProvidersFieldList...).ToSQL()

	if err != nil {
		return nil, err
	}

	updatedProvider, err := scanVCSProvider(tx.QueryRow(ctx, sql), false)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrOptimisticLockError
		}
		return nil, err
	}

	// Lookup namespace for group
	namespace, err := getNamespaceByGroupID(ctx, tx, updatedProvider.GroupID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	updatedProvider.ResourcePath = buildVCSProviderResourcePath(namespace.path, updatedProvider.Name)

	return updatedProvider, nil
}

func (vp *vcsProviders) DeleteProvider(ctx context.Context, provider *models.VCSProvider) error {
	sql, _, err := dialect.Delete("vcs_providers").Where(
		goqu.Ex{
			"id":      provider.Metadata.ID,
			"version": provider.Metadata.Version,
		},
	).Returning(vcsProvidersFieldList...).ToSQL()

	if err != nil {
		return err
	}

	if _, err := scanVCSProvider(vp.dbClient.getConnection(ctx).QueryRow(ctx, sql), false); err != nil {
		if err == pgx.ErrNoRows {
			return ErrOptimisticLockError
		}

		if pgErr := asPgError(err); pgErr != nil {
			if isForeignKeyViolation(pgErr) && pgErr.ConstraintName == "fk_workspace_id" {
				return errors.NewError(
					errors.EConflict,
					fmt.Sprintf("VCS provider %s has workspace configurations", provider.Name),
				)
			}
		}

		return err
	}

	return nil
}

func (vp *vcsProviders) getProvider(ctx context.Context, exp goqu.Ex) (*models.VCSProvider, error) {
	sql, _, err := dialect.From(goqu.T("vcs_providers")).
		Select(vp.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"vcs_providers.group_id": goqu.I("namespaces.group_id")})).
		Where(exp).
		ToSQL()

	if err != nil {
		return nil, err
	}

	provider, err := scanVCSProvider(vp.dbClient.getConnection(ctx).QueryRow(ctx, sql), true)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
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

func buildVCSProviderResourcePath(groupPath string, name string) string {
	return fmt.Sprintf("%s/%s", groupPath, name)
}

func scanVCSProvider(row scanner, withResourcePath bool) (*models.VCSProvider, error) {
	var description sql.NullString
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
		&vp.Hostname,
		&vp.OAuthClientID,
		&vp.OAuthClientSecret,
		&vp.OAuthState,
		&vp.OAuthAccessToken,
		&vp.OAuthRefreshToken,
		&vp.OAuthAccessTokenExpiresAt,
		&vp.AutoCreateWebhooks,
		&vp.GroupID,
	}

	var path string
	if withResourcePath {
		fields = append(fields, &path)
	}

	err := row.Scan(fields...)

	if err != nil {
		return nil, err
	}

	if description.Valid {
		vp.Description = description.String
	}

	if withResourcePath {
		vp.ResourcePath = buildVCSProviderResourcePath(path, vp.Name)
	}

	return vp, nil
}

func vcsProviderFieldResolver(key string, model interface{}) (string, error) {
	vcsProvider, ok := model.(*models.VCSProvider)
	if !ok {
		return "", errors.NewError(errors.EInternal, fmt.Sprintf("Expected VCSProvider type, got %T", model))
	}

	val, ok := metadataFieldResolver(key, &vcsProvider.Metadata)
	if !ok {
		return "", errors.NewError(errors.EInternal, fmt.Sprintf("Invalid field key requested %s", key))
	}

	return val, nil
}