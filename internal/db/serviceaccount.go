package db

//go:generate go tool mockery --name ServiceAccounts --inpackage --case underscore

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/jackc/pgx/v4"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
	"go.opentelemetry.io/otel/attribute"
)

// ServiceAccounts encapsulates the logic to access service accounts from the database
type ServiceAccounts interface {
	GetServiceAccountByID(ctx context.Context, id string) (*models.ServiceAccount, error)
	GetServiceAccountByTRN(ctx context.Context, trn string) (*models.ServiceAccount, error)
	CreateServiceAccount(ctx context.Context, serviceAccount *models.ServiceAccount) (*models.ServiceAccount, error)
	UpdateServiceAccount(ctx context.Context, serviceAccount *models.ServiceAccount) (*models.ServiceAccount, error)
	GetServiceAccounts(ctx context.Context, input *GetServiceAccountsInput) (*ServiceAccountsResult, error)
	DeleteServiceAccount(ctx context.Context, serviceAccount *models.ServiceAccount) error
	AssignServiceAccountToRunner(ctx context.Context, serviceAccountID string, runnerID string) error
	UnassignServiceAccountFromRunner(ctx context.Context, serviceAccountID string, runnerID string) error
}

// ServiceAccountSortableField represents the fields that a service account can be sorted by
type ServiceAccountSortableField string

// ServiceAccountSortableField constants
const (
	ServiceAccountSortableFieldCreatedAtAsc        ServiceAccountSortableField = "CREATED_AT_ASC"
	ServiceAccountSortableFieldCreatedAtDesc       ServiceAccountSortableField = "CREATED_AT_DESC"
	ServiceAccountSortableFieldUpdatedAtAsc        ServiceAccountSortableField = "UPDATED_AT_ASC"
	ServiceAccountSortableFieldUpdatedAtDesc       ServiceAccountSortableField = "UPDATED_AT_DESC"
	ServiceAccountSortableFieldFieldGroupLevelAsc  ServiceAccountSortableField = "GROUP_LEVEL_ASC"
	ServiceAccountSortableFieldFieldGroupLevelDesc ServiceAccountSortableField = "GROUP_LEVEL_DESC"
)

func (sf ServiceAccountSortableField) getFieldDescriptor() *pagination.FieldDescriptor {
	switch sf {
	case ServiceAccountSortableFieldCreatedAtAsc, ServiceAccountSortableFieldCreatedAtDesc:
		return &pagination.FieldDescriptor{Key: "created_at", Table: "service_accounts", Col: "created_at"}
	case ServiceAccountSortableFieldUpdatedAtAsc, ServiceAccountSortableFieldUpdatedAtDesc:
		return &pagination.FieldDescriptor{Key: "updated_at", Table: "service_accounts", Col: "updated_at"}
	case ServiceAccountSortableFieldFieldGroupLevelAsc, ServiceAccountSortableFieldFieldGroupLevelDesc:
		return &pagination.FieldDescriptor{Key: "group_path", Table: "namespaces", Col: "path"}
	default:
		return nil
	}
}

func (sf ServiceAccountSortableField) getSortDirection() pagination.SortDirection {
	if strings.HasSuffix(string(sf), "_DESC") {
		return pagination.DescSort
	}
	return pagination.AscSort
}

func (sf ServiceAccountSortableField) getTransformFunc() pagination.SortTransformFunc {
	switch sf {
	case ServiceAccountSortableFieldFieldGroupLevelAsc, ServiceAccountSortableFieldFieldGroupLevelDesc:
		return func(s string) string {
			return fmt.Sprintf("array_length(string_to_array(%s, '/'), 1)", s)
		}
	default:
		return nil
	}
}

// ServiceAccountFilter contains the supported fields for filtering ServiceAccount resources
type ServiceAccountFilter struct {
	Search            *string
	RunnerID          *string
	ServiceAccountIDs []string
	NamespacePaths    []string
}

// oidcTrustPolicyDBType is the type used to store the trust policies in the DB table
type oidcTrustPolicyDBType struct {
	BoundClaimsType models.BoundClaimsType `json:"boundClaimsType"`
	BoundClaims     map[string]string      `json:"boundClaims"`
	Issuer          string                 `json:"issuer"`
}

// GetServiceAccountsInput is the input for listing service accounts
type GetServiceAccountsInput struct {
	// Sort specifies the field to sort on and direction
	Sort *ServiceAccountSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// Filter is used to filter the results
	Filter *ServiceAccountFilter
}

// ServiceAccountsResult contains the response data and page information
type ServiceAccountsResult struct {
	PageInfo        *pagination.PageInfo
	ServiceAccounts []models.ServiceAccount
}

type serviceAccounts struct {
	dbClient *Client
}

var serviceAccountFieldList = append(metadataFieldList, "name", "description", "group_id", "created_by", "oidc_trust_policies")

// NewServiceAccounts returns an instance of the ServiceAccount interface
func NewServiceAccounts(dbClient *Client) ServiceAccounts {
	return &serviceAccounts{dbClient: dbClient}
}

// GetServiceAccount returns a serviceAccount by ID
func (s *serviceAccounts) GetServiceAccountByID(ctx context.Context, id string) (*models.ServiceAccount, error) {
	ctx, span := tracer.Start(ctx, "db.GetServiceAccountByID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	return s.getServiceAccount(ctx, goqu.Ex{"service_accounts.id": id})
}

func (s *serviceAccounts) GetServiceAccountByTRN(ctx context.Context, trn string) (*models.ServiceAccount, error) {
	ctx, span := tracer.Start(ctx, "db.GetServiceAccountByTRN")
	span.SetAttributes(attribute.String("trn", trn))
	defer span.End()

	path, err := types.ServiceAccountModelType.ResourcePathFromTRN(trn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse TRN", errors.WithSpan(span))
	}

	lastSlashIndex := strings.LastIndex(path, "/")

	// A service account TRN should always have the group path and the name of the service account.
	if lastSlashIndex == -1 {
		return nil, errors.New("a service account TRN must have the group path and service account name separated by a forward slash",
			errors.WithErrorCode(errors.EInvalid),
			errors.WithSpan(span),
		)
	}

	return s.getServiceAccount(ctx, goqu.Ex{
		"namespaces.path":       path[:lastSlashIndex],
		"service_accounts.name": path[lastSlashIndex+1:],
	})
}

func (s *serviceAccounts) GetServiceAccounts(ctx context.Context, input *GetServiceAccountsInput) (*ServiceAccountsResult, error) {
	ctx, span := tracer.Start(ctx, "db.GetServiceAccounts")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	ex := goqu.And()

	if input.Filter != nil {
		if input.Filter.ServiceAccountIDs != nil {
			ex = ex.Append(goqu.I("service_accounts.id").In(input.Filter.ServiceAccountIDs))
		}

		if input.Filter.NamespacePaths != nil {
			ex = ex.Append(goqu.I("namespaces.path").In(input.Filter.NamespacePaths))
		}

		if input.Filter.RunnerID != nil {
			ex = ex.Append(goqu.I("service_account_runner_relation.runner_id").In(*input.Filter.RunnerID))
		}

		if input.Filter.Search != nil {
			search := *input.Filter.Search

			lastDelimiterIndex := strings.LastIndex(search, "/")

			if lastDelimiterIndex != -1 {
				namespacePath := search[:lastDelimiterIndex]
				serviceAccountName := search[lastDelimiterIndex+1:]

				if serviceAccountName != "" {
					// An OR condition is used here since the last component of the search path could be part of
					// the namespace or it can be a service account name prefix
					ex = ex.Append(
						goqu.Or(
							goqu.And(
								goqu.I("namespaces.path").Eq(namespacePath),
								goqu.I("service_accounts.name").ILike(serviceAccountName+"%"),
							),
							goqu.Or(
								goqu.I("namespaces.path").ILike(search+"%"),
								goqu.I("service_accounts.name").ILike(serviceAccountName+"%"),
							),
						),
					)
				} else {
					// We know the search is a namespace path since it ends with a "/"
					ex = ex.Append(goqu.I("namespaces.path").ILike(namespacePath + "%"))
				}
			} else {
				// We don't know if the search is for a namespace path or service account name; therefore, use
				// an OR condition to search both
				ex = ex.Append(
					goqu.Or(
						goqu.I("namespaces.path").ILike(search+"%"),
						goqu.I("service_accounts.name").ILike(search+"%"),
					),
				)
			}
		}

	}

	query := dialect.From("service_accounts").
		Select(s.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"service_accounts.group_id": goqu.I("namespaces.group_id")}))

	if input.Filter != nil && input.Filter.RunnerID != nil {
		// Add inner join for runner relation table
		query = query.InnerJoin(goqu.T("service_account_runner_relation"), goqu.On(goqu.Ex{"service_accounts.id": goqu.I("service_account_runner_relation.service_account_id")}))
	}

	query = query.Where(ex)

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
		&pagination.FieldDescriptor{Key: "id", Table: "service_accounts", Col: "id"},
		pagination.WithSortByField(sortBy, sortDirection),
		pagination.WithSortByTransform(sortTransformFunc),
	)
	if err != nil {
		tracing.RecordError(span, err, "failed to build query")
		return nil, err
	}

	rows, err := qBuilder.Execute(ctx, s.dbClient.getConnection(ctx), query)
	if err != nil {
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	defer rows.Close()

	// Scan rows
	results := []models.ServiceAccount{}
	for rows.Next() {
		item, err := scanServiceAccount(rows)
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

	result := ServiceAccountsResult{
		PageInfo:        rows.GetPageInfo(),
		ServiceAccounts: results,
	}

	return &result, nil
}

// CreateServiceAccount creates a new serviceAccount
func (s *serviceAccounts) CreateServiceAccount(ctx context.Context, serviceAccount *models.ServiceAccount) (*models.ServiceAccount, error) {
	ctx, span := tracer.Start(ctx, "db.CreateServiceAccount")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	timestamp := currentTime()

	trustPoliciesJSON, err := s.marshalOIDCTrustPolicies(serviceAccount.OIDCTrustPolicies)
	if err != nil {
		tracing.RecordError(span, err, "failed to marshal OIDC trust policies")
		return nil, err
	}

	sql, args, err := dialect.From("service_accounts").
		Prepared(true).
		With("service_accounts",
			dialect.Insert("service_accounts").Rows(
				goqu.Record{
					"id":                  newResourceID(),
					"version":             initialResourceVersion,
					"created_at":          timestamp,
					"updated_at":          timestamp,
					"name":                serviceAccount.Name,
					"description":         serviceAccount.Description,
					"group_id":            serviceAccount.GroupID,
					"created_by":          serviceAccount.CreatedBy,
					"oidc_trust_policies": trustPoliciesJSON,
				}).Returning("*"),
		).Select(s.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"service_accounts.group_id": goqu.I("namespaces.group_id")})).
		ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	createdServiceAccount, err := scanServiceAccount(s.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				tracing.RecordError(span, nil,
					"Service account with name %s already exists in group %s", serviceAccount.Name, serviceAccount.GroupID)
				return nil, errors.New(
					"Service account with name %s already exists in group %s", serviceAccount.Name, serviceAccount.GroupID,
					errors.WithErrorCode(errors.EConflict),
				)
			}
			if isForeignKeyViolation(pgErr) && pgErr.ConstraintName == "fk_group_id" {
				tracing.RecordError(span, nil, "invalid group: the specified group does not exist")
				return nil, errors.New("invalid group: the specified group does not exist", errors.WithErrorCode(errors.EConflict))
			}
		}
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	return createdServiceAccount, nil
}

// UpdateServiceAccount updates an existing serviceAccount by name
func (s *serviceAccounts) UpdateServiceAccount(ctx context.Context, serviceAccount *models.ServiceAccount) (*models.ServiceAccount, error) {
	ctx, span := tracer.Start(ctx, "db.UpdateServiceAccount")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	trustPoliciesJSON, err := s.marshalOIDCTrustPolicies(serviceAccount.OIDCTrustPolicies)
	if err != nil {
		tracing.RecordError(span, err, "failed to marshal OIDC trust policies")
		return nil, err
	}

	timestamp := currentTime()

	sql, args, err := dialect.From("service_accounts").
		Prepared(true).
		With("service_accounts",
			dialect.Update("service_accounts").
				Set(
					goqu.Record{
						"version":             goqu.L("? + ?", goqu.C("version"), 1),
						"updated_at":          timestamp,
						"description":         serviceAccount.Description,
						"oidc_trust_policies": trustPoliciesJSON,
					},
				).Where(goqu.Ex{"id": serviceAccount.Metadata.ID, "version": serviceAccount.Metadata.Version}).
				Returning("*"),
		).Select(s.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"service_accounts.group_id": goqu.I("namespaces.group_id")})).
		ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	updatedServiceAccount, err := scanServiceAccount(s.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return nil, ErrOptimisticLockError
		}
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	return updatedServiceAccount, nil
}

func (s *serviceAccounts) DeleteServiceAccount(ctx context.Context, serviceAccount *models.ServiceAccount) error {
	ctx, span := tracer.Start(ctx, "db.DeleteServiceAccount")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	sql, args, err := dialect.From("service_accounts").
		Prepared(true).
		With("service_accounts",
			dialect.Delete("service_accounts").
				Where(goqu.Ex{"id": serviceAccount.Metadata.ID, "version": serviceAccount.Metadata.Version}).
				Returning("*"),
		).Select(s.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"service_accounts.group_id": goqu.I("namespaces.group_id")})).
		ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return err
	}

	if _, err := scanServiceAccount(s.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...)); err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return ErrOptimisticLockError
		}

		if pgErr := asPgError(err); pgErr != nil {
			if isForeignKeyViolation(pgErr) {
				tracing.RecordError(span, nil,
					"Service account %s is assigned as a member of a group/workspace", serviceAccount.Name)
				return errors.New(
					"Service account %s is assigned as a member of a group/workspace", serviceAccount.Name,
					errors.WithErrorCode(errors.EConflict),
				)
			}
		}

		tracing.RecordError(span, err, "failed to execute query")
		return err
	}

	return nil
}

func (s *serviceAccounts) getServiceAccount(ctx context.Context, exp exp.Ex) (*models.ServiceAccount, error) {
	ctx, span := tracer.Start(ctx, "db.getServiceAccount")
	defer span.End()

	sql, args, err := dialect.From("service_accounts").
		Prepared(true).
		Select(s.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"service_accounts.group_id": goqu.I("namespaces.group_id")})).
		Where(exp).
		ToSQL()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	serviceAccount, err := scanServiceAccount(s.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}

		if pgErr := asPgError(err); pgErr != nil {
			if isInvalidIDViolation(pgErr) {
				return nil, ErrInvalidID
			}
		}

		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return serviceAccount, nil
}

func (s *serviceAccounts) AssignServiceAccountToRunner(ctx context.Context, serviceAccountID string, runnerID string) error {
	ctx, span := tracer.Start(ctx, "db.AssignServiceAccountToRunner")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	sql, args, err := dialect.Insert("service_account_runner_relation").
		Prepared(true).
		Rows(goqu.Record{
			"service_account_id": serviceAccountID,
			"runner_id":          runnerID,
		}).ToSQL()

	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return err
	}

	if _, err = s.dbClient.getConnection(ctx).Exec(ctx, sql, args...); err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				tracing.RecordError(span, nil, "service account already assigned to runner")
				return errors.New("service account already assigned to runner", errors.WithErrorCode(errors.EConflict))
			}
		}
		tracing.RecordError(span, err, "failed to execute query")
		return err
	}

	return nil
}

func (s *serviceAccounts) UnassignServiceAccountFromRunner(ctx context.Context, serviceAccountID string, runnerID string) error {
	ctx, span := tracer.Start(ctx, "db.UnassignServiceAccountFromRunner")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	sql, args, err := dialect.Delete("service_account_runner_relation").
		Prepared(true).
		Where(
			goqu.Ex{
				"service_account_id": serviceAccountID,
				"runner_id":          runnerID,
			},
		).ToSQL()

	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return err
	}

	if _, err = s.dbClient.getConnection(ctx).Exec(ctx, sql, args...); err != nil {
		tracing.RecordError(span, err, "failed to execute query")
		return err
	}

	return nil
}

func (s *serviceAccounts) getSelectFields() []interface{} {
	selectFields := []interface{}{}
	for _, field := range serviceAccountFieldList {
		selectFields = append(selectFields, fmt.Sprintf("service_accounts.%s", field))
	}

	selectFields = append(selectFields, "namespaces.path")

	return selectFields
}

func (s *serviceAccounts) marshalOIDCTrustPolicies(input []models.OIDCTrustPolicy) ([]byte, error) {
	policies := []oidcTrustPolicyDBType{}
	for _, p := range input {
		policies = append(policies, oidcTrustPolicyDBType{
			Issuer:          p.Issuer,
			BoundClaimsType: p.BoundClaimsType,
			BoundClaims:     p.BoundClaims,
		})
	}
	trustPoliciesJSON, err := json.Marshal(policies)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal trust policies to JSON %v", err)
	}
	return trustPoliciesJSON, nil
}

func scanServiceAccount(row scanner) (*models.ServiceAccount, error) {
	var groupPath string
	serviceAccount := &models.ServiceAccount{}

	policies := []oidcTrustPolicyDBType{}

	fields := []interface{}{
		&serviceAccount.Metadata.ID,
		&serviceAccount.Metadata.CreationTimestamp,
		&serviceAccount.Metadata.LastUpdatedTimestamp,
		&serviceAccount.Metadata.Version,
		&serviceAccount.Name,
		&serviceAccount.Description,
		&serviceAccount.GroupID,
		&serviceAccount.CreatedBy,
		&policies,
		&groupPath,
	}

	err := row.Scan(fields...)
	if err != nil {
		return nil, err
	}

	serviceAccount.OIDCTrustPolicies = []models.OIDCTrustPolicy{}
	for _, p := range policies {
		// Default bound claims type to string if it's not defined
		boundClaimsType := p.BoundClaimsType
		if boundClaimsType == "" {
			boundClaimsType = models.BoundClaimsTypeString
		}

		serviceAccount.OIDCTrustPolicies = append(serviceAccount.OIDCTrustPolicies, models.OIDCTrustPolicy{
			Issuer:          p.Issuer,
			BoundClaimsType: boundClaimsType,
			BoundClaims:     p.BoundClaims,
		})
	}

	serviceAccount.Metadata.TRN = types.ServiceAccountModelType.BuildTRN(groupPath, serviceAccount.Name)

	return serviceAccount, nil
}
