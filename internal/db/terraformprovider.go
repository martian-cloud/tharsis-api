package db

//go:generate go tool mockery --name TerraformProviders --inpackage --case underscore

import (
	"context"
	"fmt"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v4"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
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

func (ts TerraformProviderSortableField) getFieldDescriptor() *pagination.FieldDescriptor {
	switch ts {
	case TerraformProviderSortableFieldNameAsc, TerraformProviderSortableFieldNameDesc:
		return &pagination.FieldDescriptor{Key: "name", Table: "terraform_providers", Col: "name"}
	case TerraformProviderSortableFieldUpdatedAtAsc, TerraformProviderSortableFieldUpdatedAtDesc:
		return &pagination.FieldDescriptor{Key: "updated_at", Table: "terraform_providers", Col: "updated_at"}
	default:
		return nil
	}
}

func (ts TerraformProviderSortableField) getSortDirection() pagination.SortDirection {
	if strings.HasSuffix(string(ts), "_DESC") {
		return pagination.DescSort
	}
	return pagination.AscSort
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
	PaginationOptions *pagination.Options
	// Filter is used to filter the results
	Filter *TerraformProviderFilter
}

// ProvidersResult contains the response data and page information
type ProvidersResult struct {
	PageInfo  *pagination.PageInfo
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
	ctx, span := tracer.Start(ctx, "db.GetProviderByID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	return t.getProvider(ctx, goqu.Ex{"terraform_providers.id": id})
}

func (t *terraformProviders) GetProviderByPath(ctx context.Context, path string) (*models.TerraformProvider, error) {
	ctx, span := tracer.Start(ctx, "db.GetProviderByPath")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	index := strings.LastIndex(path, "/")
	return t.getProvider(ctx, goqu.Ex{"terraform_providers.name": path[index+1:], "namespaces.path": path[:index]})
}

func (t *terraformProviders) GetProviders(ctx context.Context, input *GetProvidersInput) (*ProvidersResult, error) {
	ctx, span := tracer.Start(ctx, "db.GetProviders")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

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
							goqu.I("namespaces.path").ILike(registryNamespace+"%"),
							goqu.I("terraform_providers.name").ILike(providerName+"%"),
						),
					)
				} else {
					// We know the search is a namespace path since it ends with a "/"
					ex = ex.Append(goqu.I("namespaces.path").ILike(registryNamespace + "%"))
				}
			} else {
				// We don't know if the search is for a namespace path or provider name; therefore, use
				// an OR condition to search both
				ex = ex.Append(
					goqu.Or(
						goqu.I("namespaces.path").ILike(search+"%"),
						goqu.I("terraform_providers.name").ILike(search+"%"),
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

	sortDirection := pagination.AscSort

	var sortBy *pagination.FieldDescriptor
	if input.Sort != nil {
		sortDirection = input.Sort.getSortDirection()
		sortBy = input.Sort.getFieldDescriptor()
	}

	qBuilder, err := pagination.NewPaginatedQueryBuilder(
		input.PaginationOptions,
		&pagination.FieldDescriptor{Key: "id", Table: "terraform_providers", Col: "id"},
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
	results := []models.TerraformProvider{}
	for rows.Next() {
		item, err := scanTerraformProvider(rows, true)
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

	result := ProvidersResult{
		PageInfo:  rows.GetPageInfo(),
		Providers: results,
	}

	return &result, nil
}

func (t *terraformProviders) CreateProvider(ctx context.Context, provider *models.TerraformProvider) (*models.TerraformProvider, error) {
	ctx, span := tracer.Start(ctx, "db.CreateProvider")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	timestamp := currentTime()

	tx, err := t.dbClient.getConnection(ctx).Begin(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return nil, err
	}

	// Rollback is safe to call even if the tx is already closed, so if
	// the tx commits successfully, this is a no-op
	defer func() {
		if txErr := tx.Rollback(ctx); txErr != nil && txErr != pgx.ErrTxClosed {
			t.dbClient.logger.Errorf("failed to rollback tx: %v", txErr)
		}
	}()

	sql, args, err := dialect.Insert("terraform_providers").
		Prepared(true).
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
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	createdProvider, err := scanTerraformProvider(tx.QueryRow(ctx, sql, args...), false)
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				tracing.RecordError(span, nil,
					"terraform provider with name %s already exists", provider.Name)
				return nil, errors.New("terraform provider with name %s already exists", provider.Name, errors.WithErrorCode(errors.EConflict))
			}
		}
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	// Lookup namespace for group
	namespace, err := getNamespaceByGroupID(ctx, tx, provider.GroupID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get namespace by group ID")
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return nil, err
	}

	createdProvider.ResourcePath = buildTerraformProviderResourcePath(namespace.path, provider.Name)

	return createdProvider, nil
}

func (t *terraformProviders) UpdateProvider(ctx context.Context, provider *models.TerraformProvider) (*models.TerraformProvider, error) {
	ctx, span := tracer.Start(ctx, "db.UpdateProvider")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	timestamp := currentTime()

	tx, err := t.dbClient.getConnection(ctx).Begin(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return nil, err
	}

	// Rollback is safe to call even if the tx is already closed, so if
	// the tx commits successfully, this is a no-op
	defer func() {
		if txErr := tx.Rollback(ctx); txErr != nil && txErr != pgx.ErrTxClosed {
			t.dbClient.logger.Errorf("failed to rollback tx: %v", txErr)
		}
	}()

	sql, args, err := dialect.Update("terraform_providers").
		Prepared(true).
		Set(
			goqu.Record{
				"version":    goqu.L("? + ?", goqu.C("version"), 1),
				"updated_at": timestamp,
				"private":    provider.Private,
				"repo_url":   provider.RepositoryURL,
			},
		).Where(goqu.Ex{"id": provider.Metadata.ID, "version": provider.Metadata.Version}).Returning(providerFieldList...).ToSQL()

	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	updatedProvider, err := scanTerraformProvider(tx.QueryRow(ctx, sql, args...), false)

	if err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return nil, ErrOptimisticLockError
		}
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	// Lookup namespace for group
	namespace, err := getNamespaceByGroupID(ctx, tx, provider.GroupID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get namespace by group ID")
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return nil, err
	}

	updatedProvider.ResourcePath = buildTerraformProviderResourcePath(namespace.path, provider.Name)

	return updatedProvider, nil
}

func (t *terraformProviders) DeleteProvider(ctx context.Context, provider *models.TerraformProvider) error {
	ctx, span := tracer.Start(ctx, "db.DeleteProvider")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	sql, args, err := dialect.Delete("terraform_providers").
		Prepared(true).
		Where(
			goqu.Ex{
				"id":      provider.Metadata.ID,
				"version": provider.Metadata.Version,
			},
		).Returning(providerFieldList...).ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return err
	}

	_, err = scanTerraformProvider(t.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...), false)
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

func (t *terraformProviders) getProvider(ctx context.Context, exp goqu.Ex) (*models.TerraformProvider, error) {
	query := dialect.From(goqu.T("terraform_providers")).
		Prepared(true).
		Select(t.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"terraform_providers.group_id": goqu.I("namespaces.group_id")})).
		Where(exp)

	sql, args, err := query.ToSQL()
	if err != nil {
		return nil, err
	}

	provider, err := scanTerraformProvider(t.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...), true)
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
