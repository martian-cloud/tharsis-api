package db

//go:generate mockery --name TerraformProviders --inpackage --case underscore

import (
	"context"
	"fmt"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v4"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// TerraformProviders encapsulates the logic to access terraform providers from the database
type TerraformProviders interface {
	GetProviderByID(ctx context.Context, id string) (*models.TerraformProvider, error)
	GetProviderByPath(ctx context.Context, path string) (*models.TerraformProvider, error)
	GetProviders(ctx context.Context, input *GetProvidersInput) (*ProvidersResult, error)
	CreateProvider(ctx context.Context, provider *models.TerraformProvider) (*models.TerraformProvider, error)
	UpdateProvider(ctx context.Context, provider *models.TerraformProvider) (*models.TerraformProvider, error)
	DeleteProvider(ctx context.Context, provider *models.TerraformProvider) error
}

// TerraformProviderSortableField represents the fields that a provider can be sorted by
type TerraformProviderSortableField string

// TerraformProviderSortableField constants
const (
	TerraformProviderSortableFieldNameAsc       TerraformProviderSortableField = "NAME_ASC"
	TerraformProviderSortableFieldNameDesc      TerraformProviderSortableField = "NAME_DESC"
	TerraformProviderSortableFieldUpdatedAtAsc  TerraformProviderSortableField = "UPDATED_AT_ASC"
	TerraformProviderSortableFieldUpdatedAtDesc TerraformProviderSortableField = "UPDATED_AT_DESC"
)

func (ts TerraformProviderSortableField) getFieldDescriptor() *fieldDescriptor {
	switch ts {
	case TerraformProviderSortableFieldNameAsc, TerraformProviderSortableFieldNameDesc:
		return &fieldDescriptor{key: "name", table: "terraform_providers", col: "name"}
	case TerraformProviderSortableFieldUpdatedAtAsc, TerraformProviderSortableFieldUpdatedAtDesc:
		return &fieldDescriptor{key: "updated_at", table: "terraform_providers", col: "updated_at"}
	default:
		return nil
	}
}

func (ts TerraformProviderSortableField) getSortDirection() SortDirection {
	if strings.HasSuffix(string(ts), "_DESC") {
		return DescSort
	}
	return AscSort
}

// TerraformProviderFilter contains the supported fields for filtering TerraformProvider resources
type TerraformProviderFilter struct {
	Search               *string
	Name                 *string
	RootGroupID          *string
	GroupID              *string
	UserID               *string
	ServiceAccountID     *string
	TerraformProviderIDs []string
}

// GetProvidersInput is the input for listing terraform providers
type GetProvidersInput struct {
	// Sort specifies the field to sort on and direction
	Sort *TerraformProviderSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *PaginationOptions
	// Filter is used to filter the results
	Filter *TerraformProviderFilter
}

// ProvidersResult contains the response data and page information
type ProvidersResult struct {
	PageInfo  *PageInfo
	Providers []models.TerraformProvider
}

type terraformProviders struct {
	dbClient *Client
}

var providerFieldList = append(metadataFieldList, "group_id", "root_group_id", "name", "private", "repo_url", "created_by")

// NewTerraformProviders returns an instance of the TerraformProviders interface
func NewTerraformProviders(dbClient *Client) TerraformProviders {
	return &terraformProviders{dbClient: dbClient}
}

func (t *terraformProviders) GetProviderByID(ctx context.Context, id string) (*models.TerraformProvider, error) {
	return t.getProvider(ctx, goqu.Ex{"terraform_providers.id": id})
}

func (t *terraformProviders) GetProviderByPath(ctx context.Context, path string) (*models.TerraformProvider, error) {
	index := strings.LastIndex(path, "/")
	return t.getProvider(ctx, goqu.Ex{"terraform_providers.name": path[index+1:], "namespaces.path": path[:index]})
}

func (t *terraformProviders) GetProviders(ctx context.Context, input *GetProvidersInput) (*ProvidersResult, error) {
	ex := goqu.And()

	if input.Filter != nil {
		if input.Filter.TerraformProviderIDs != nil {
			ex = ex.Append(goqu.I("terraform_providers.id").In(input.Filter.TerraformProviderIDs))
		}
		if input.Filter.Search != nil && *input.Filter.Search != "" {
			search := *input.Filter.Search

			lastDelimiterIndex := strings.LastIndex(search, "/")

			if lastDelimiterIndex != -1 {
				registryNamespace := search[:lastDelimiterIndex]
				providerName := search[lastDelimiterIndex+1:]

				if providerName != "" {
					// An AND condition is used here since the first part of the search is the registry namespace path
					// and the second part is the provider name
					ex = ex.Append(
						goqu.And(
							goqu.I("namespaces.path").Like(registryNamespace+"%"),
							goqu.I("terraform_providers.name").Like(providerName+"%"),
						),
					)
				} else {
					// We know the search is a namespace path since it ends with a "/"
					ex = ex.Append(goqu.I("namespaces.path").Like(registryNamespace + "%"))
				}
			} else {
				// We don't know if the search is for a namespace path or provider name; therefore, use
				// an OR condition to search both
				ex = ex.Append(
					goqu.Or(
						goqu.I("namespaces.path").Like(search+"%"),
						goqu.I("terraform_providers.name").Like(search+"%"),
					),
				)
			}
		}
		if input.Filter.GroupID != nil {
			ex = ex.Append(goqu.I("terraform_providers.group_id").Eq(*input.Filter.GroupID))
		}
		if input.Filter.RootGroupID != nil {
			ex = ex.Append(goqu.I("terraform_providers.root_group_id").Eq(*input.Filter.RootGroupID))
		}
		if input.Filter.Name != nil {
			ex = ex.Append(goqu.I("terraform_providers.name").Eq(*input.Filter.Name))
		}
		if input.Filter.UserID != nil {
			ex = ex.Append(
				goqu.Or(
					goqu.I("terraform_providers.private").Eq(false),
					namespaceMembershipExpressionBuilder{
						userID: input.Filter.UserID,
					}.build(),
				))
		}
		if input.Filter.ServiceAccountID != nil {
			ex = ex.Append(
				goqu.Or(
					goqu.I("terraform_providers.private").Eq(false),
					namespaceMembershipExpressionBuilder{
						serviceAccountID: input.Filter.ServiceAccountID,
					}.build(),
				))
		}
	}

	query := dialect.From(goqu.T("terraform_providers")).
		Select(t.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"terraform_providers.group_id": goqu.I("namespaces.group_id")})).
		Where(ex)

	sortDirection := AscSort

	var sortBy *fieldDescriptor
	if input.Sort != nil {
		sortDirection = input.Sort.getSortDirection()
		sortBy = input.Sort.getFieldDescriptor()
	}

	qBuilder, err := newPaginatedQueryBuilder(
		input.PaginationOptions,
		&fieldDescriptor{key: "id", table: "terraform_providers", col: "id"},
		sortBy,
		sortDirection,
		providerFieldResolver,
	)

	if err != nil {
		return nil, err
	}

	rows, err := qBuilder.execute(ctx, t.dbClient.getConnection(ctx), query)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	// Scan rows
	results := []models.TerraformProvider{}
	for rows.Next() {
		item, err := scanTerraformProvider(rows, true)
		if err != nil {
			return nil, err
		}

		results = append(results, *item)
	}

	if err := rows.finalize(&results); err != nil {
		return nil, err
	}

	result := ProvidersResult{
		PageInfo:  rows.getPageInfo(),
		Providers: results,
	}

	return &result, nil
}

func (t *terraformProviders) CreateProvider(ctx context.Context, provider *models.TerraformProvider) (*models.TerraformProvider, error) {
	timestamp := currentTime()

	tx, err := t.dbClient.getConnection(ctx).Begin(ctx)
	if err != nil {
		return nil, err
	}

	// Rollback is safe to call even if the tx is already closed, so if
	// the tx commits successfully, this is a no-op
	defer func() {
		if txErr := tx.Rollback(ctx); txErr != nil && txErr != pgx.ErrTxClosed {
			t.dbClient.logger.Errorf("failed to rollback tx: %v", txErr)
		}
	}()

	sql, _, err := dialect.Insert("terraform_providers").
		Rows(goqu.Record{
			"id":            newResourceID(),
			"version":       initialResourceVersion,
			"created_at":    timestamp,
			"updated_at":    timestamp,
			"group_id":      provider.GroupID,
			"root_group_id": provider.RootGroupID,
			"name":          provider.Name,
			"private":       provider.Private,
			"repo_url":      provider.RepositoryURL,
			"created_by":    provider.CreatedBy,
		}).
		Returning(providerFieldList...).ToSQL()
	if err != nil {
		return nil, err
	}

	createdProvider, err := scanTerraformProvider(tx.QueryRow(ctx, sql), false)
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				return nil, errors.NewError(errors.EConflict, fmt.Sprintf("terraform provider with name %s already exists", provider.Name))
			}
		}
		return nil, err
	}

	// Lookup namespace for group
	namespace, err := getNamespaceByGroupID(ctx, tx, provider.GroupID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	createdProvider.ResourcePath = buildTerraformProviderResourcePath(namespace.path, provider.Name)

	return createdProvider, nil
}

func (t *terraformProviders) UpdateProvider(ctx context.Context, provider *models.TerraformProvider) (*models.TerraformProvider, error) {
	timestamp := currentTime()

	tx, err := t.dbClient.getConnection(ctx).Begin(ctx)
	if err != nil {
		return nil, err
	}

	// Rollback is safe to call even if the tx is already closed, so if
	// the tx commits successfully, this is a no-op
	defer func() {
		if txErr := tx.Rollback(ctx); txErr != nil && txErr != pgx.ErrTxClosed {
			t.dbClient.logger.Errorf("failed to rollback tx: %v", txErr)
		}
	}()

	sql, _, err := dialect.Update("terraform_providers").Set(
		goqu.Record{
			"version":    goqu.L("? + ?", goqu.C("version"), 1),
			"updated_at": timestamp,
			"name":       provider.Name,
			"private":    provider.Private,
			"repo_url":   provider.RepositoryURL,
		},
	).Where(goqu.Ex{"id": provider.Metadata.ID, "version": provider.Metadata.Version}).Returning(providerFieldList...).ToSQL()

	if err != nil {
		return nil, err
	}

	updatedProvider, err := scanTerraformProvider(tx.QueryRow(ctx, sql), false)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrOptimisticLockError
		}
		return nil, err
	}

	// Lookup namespace for group
	namespace, err := getNamespaceByGroupID(ctx, tx, provider.GroupID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	updatedProvider.ResourcePath = buildTerraformProviderResourcePath(namespace.path, provider.Name)

	return updatedProvider, nil
}

func (t *terraformProviders) DeleteProvider(ctx context.Context, provider *models.TerraformProvider) error {

	sql, _, err := dialect.Delete("terraform_providers").Where(
		goqu.Ex{
			"id":      provider.Metadata.ID,
			"version": provider.Metadata.Version,
		},
	).Returning(providerFieldList...).ToSQL()
	if err != nil {
		return err
	}

	_, err = scanTerraformProvider(t.dbClient.getConnection(ctx).QueryRow(ctx, sql), false)
	if err != nil {
		if err == pgx.ErrNoRows {
			return ErrOptimisticLockError
		}
		return err
	}

	return nil
}

func (t *terraformProviders) getProvider(ctx context.Context, exp goqu.Ex) (*models.TerraformProvider, error) {
	query := dialect.From(goqu.T("terraform_providers")).
		Select(t.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"terraform_providers.group_id": goqu.I("namespaces.group_id")})).
		Where(exp)

	sql, _, err := query.ToSQL()
	if err != nil {
		return nil, err
	}

	provider, err := scanTerraformProvider(t.dbClient.getConnection(ctx).QueryRow(ctx, sql), true)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return provider, nil
}

func (t *terraformProviders) getSelectFields() []interface{} {
	selectFields := []interface{}{}
	for _, field := range providerFieldList {
		selectFields = append(selectFields, fmt.Sprintf("terraform_providers.%s", field))
	}

	selectFields = append(selectFields, "namespaces.path")

	return selectFields
}

func buildTerraformProviderResourcePath(groupPath string, name string) string {
	return fmt.Sprintf("%s/%s", groupPath, name)
}

func scanTerraformProvider(row scanner, withResourcePath bool) (*models.TerraformProvider, error) {
	provider := &models.TerraformProvider{}

	fields := []interface{}{
		&provider.Metadata.ID,
		&provider.Metadata.CreationTimestamp,
		&provider.Metadata.LastUpdatedTimestamp,
		&provider.Metadata.Version,
		&provider.GroupID,
		&provider.RootGroupID,
		&provider.Name,
		&provider.Private,
		&provider.RepositoryURL,
		&provider.CreatedBy,
	}

	var path string
	if withResourcePath {
		fields = append(fields, &path)
	}

	err := row.Scan(fields...)
	if err != nil {
		return nil, err
	}

	if withResourcePath {
		provider.ResourcePath = buildTerraformProviderResourcePath(path, provider.Name)
	}

	return provider, nil
}

func providerFieldResolver(key string, model interface{}) (string, error) {
	provider, ok := model.(*models.TerraformProvider)
	if !ok {
		return "", errors.NewError(errors.EInternal, fmt.Sprintf("Expected provider type, got %T", model))
	}

	val, ok := metadataFieldResolver(key, &provider.Metadata)
	if !ok {
		switch key {
		case "name":
			val = provider.Name
		default:
			return "", errors.NewError(errors.EInternal, fmt.Sprintf("Invalid field key requested %s", key))
		}
	}

	return val, nil
}
