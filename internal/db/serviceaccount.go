package db

//go:generate mockery --name ServiceAccounts --inpackage --case underscore

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/jackc/pgx/v4"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// ServiceAccounts encapsulates the logic to access service accounts from the database
type ServiceAccounts interface {
	GetServiceAccountByID(ctx context.Context, id string) (*models.ServiceAccount, error)
	GetServiceAccountByPath(ctx context.Context, path string) (*models.ServiceAccount, error)
	CreateServiceAccount(ctx context.Context, serviceAccount *models.ServiceAccount) (*models.ServiceAccount, error)
	UpdateServiceAccount(ctx context.Context, serviceAccount *models.ServiceAccount) (*models.ServiceAccount, error)
	GetServiceAccounts(ctx context.Context, input *GetServiceAccountsInput) (*ServiceAccountsResult, error)
	DeleteServiceAccount(ctx context.Context, serviceAccount *models.ServiceAccount) error
}

// ServiceAccountSortableField represents the fields that a service account can be sorted by
type ServiceAccountSortableField string

// ServiceAccountSortableField constants
const (
	ServiceAccountSortableFieldCreatedAtAsc  ServiceAccountSortableField = "CREATED_AT_ASC"
	ServiceAccountSortableFieldCreatedAtDesc ServiceAccountSortableField = "CREATED_AT_DESC"
	ServiceAccountSortableFieldUpdatedAtAsc  ServiceAccountSortableField = "UPDATED_AT_ASC"
	ServiceAccountSortableFieldUpdatedAtDesc ServiceAccountSortableField = "UPDATED_AT_DESC"
)

func (sf ServiceAccountSortableField) getFieldDescriptor() *fieldDescriptor {
	switch sf {
	case ServiceAccountSortableFieldCreatedAtAsc, ServiceAccountSortableFieldCreatedAtDesc:
		return &fieldDescriptor{key: "created_at", table: "service_accounts", col: "created_at"}
	case ServiceAccountSortableFieldUpdatedAtAsc, ServiceAccountSortableFieldUpdatedAtDesc:
		return &fieldDescriptor{key: "updated_at", table: "service_accounts", col: "updated_at"}
	default:
		return nil
	}
}

func (sf ServiceAccountSortableField) getSortDirection() SortDirection {
	if strings.HasSuffix(string(sf), "_DESC") {
		return DescSort
	}
	return AscSort
}

// ServiceAccountFilter contains the supported fields for filtering ServiceAccount resources
type ServiceAccountFilter struct {
	Search            *string
	ServiceAccountIDs []string
	NamespacePaths    []string
}

// oidcTrustPolicyDBType is the type used to store the trust policies in the DB table
type oidcTrustPolicyDBType struct {
	BoundClaims map[string]string `json:"boundClaims"`
	Issuer      string            `json:"issuer"`
}

// GetServiceAccountsInput is the input for listing service accounts
type GetServiceAccountsInput struct {
	// Sort specifies the field to sort on and direction
	Sort *ServiceAccountSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *PaginationOptions
	// Filter is used to filter the results
	Filter *ServiceAccountFilter
}

// ServiceAccountsResult contains the response data and page information
type ServiceAccountsResult struct {
	PageInfo        *PageInfo
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
	return s.getServiceAccount(ctx, goqu.Ex{"service_accounts.id": id})
}

func (s *serviceAccounts) GetServiceAccountByPath(ctx context.Context, path string) (*models.ServiceAccount, error) {
	parts := strings.Split(path, "/")
	name := parts[len(parts)-1]
	namespace := strings.Join(parts[:len(parts)-1], "/")

	return s.getServiceAccount(ctx, goqu.Ex{"service_accounts.name": name, "namespaces.path": namespace})
}

func (s *serviceAccounts) GetServiceAccounts(ctx context.Context, input *GetServiceAccountsInput) (*ServiceAccountsResult, error) {
	ex := goqu.And()

	if input.Filter != nil {
		if input.Filter.ServiceAccountIDs != nil {
			ex = ex.Append(goqu.I("service_accounts.id").In(input.Filter.ServiceAccountIDs))
		}

		if input.Filter.NamespacePaths != nil {
			ex = ex.Append(goqu.I("namespaces.path").In(input.Filter.NamespacePaths))
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
								goqu.I("service_accounts.name").Like(serviceAccountName+"%%"),
							),
							goqu.Or(
								goqu.I("namespaces.path").Like(search+"%"),
								goqu.I("service_accounts.name").Like(serviceAccountName+"%%"),
							),
						),
					)
				} else {
					// We know the search is a namespace path since it ends with a "/"
					ex = ex.Append(goqu.I("namespaces.path").Like(namespacePath + "%%"))
				}
			} else {
				// We don't know if the search is for a namespace path or service account name; therefore, use
				// an OR condition to search both
				ex = ex.Append(
					goqu.Or(
						goqu.I("namespaces.path").Like(search+"%%"),
						goqu.I("service_accounts.name").Like(search+"%%"),
					),
				)
			}
		}

	}

	query := dialect.From("service_accounts").
		Select(s.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"service_accounts.group_id": goqu.I("namespaces.group_id")})).
		Where(ex)

	sortDirection := AscSort

	var sortBy *fieldDescriptor
	if input.Sort != nil {
		sortDirection = input.Sort.getSortDirection()
		sortBy = input.Sort.getFieldDescriptor()
	}

	qBuilder, err := newPaginatedQueryBuilder(
		input.PaginationOptions,
		&fieldDescriptor{key: "id", table: "service_accounts", col: "id"},
		sortBy,
		sortDirection,
		serviceAccountFieldResolver,
	)

	if err != nil {
		return nil, err
	}

	rows, err := qBuilder.execute(ctx, s.dbClient.getConnection(ctx), query)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	// Scan rows
	results := []models.ServiceAccount{}
	for rows.Next() {
		item, err := scanServiceAccount(rows, true)
		if err != nil {
			return nil, err
		}

		results = append(results, *item)
	}

	if err := rows.finalize(&results); err != nil {
		return nil, err
	}

	result := ServiceAccountsResult{
		PageInfo:        rows.getPageInfo(),
		ServiceAccounts: results,
	}

	return &result, nil
}

// CreateServiceAccount creates a new serviceAccount
func (s *serviceAccounts) CreateServiceAccount(ctx context.Context, serviceAccount *models.ServiceAccount) (*models.ServiceAccount, error) {
	timestamp := currentTime()

	tx, err := s.dbClient.getConnection(ctx).Begin(ctx)
	if err != nil {
		return nil, err
	}

	// Rollback is safe to call even if the tx is already closed, so if
	// the tx commits successfully, this is a no-op
	defer func() {
		if txErr := tx.Rollback(ctx); txErr != nil && txErr != pgx.ErrTxClosed {
			s.dbClient.logger.Errorf("failed to rollback tx for CreateServiceAccount: %v", txErr)
		}
	}()

	trustPoliciesJSON, err := s.marshalOIDCTrustPolicies(serviceAccount.OIDCTrustPolicies)
	if err != nil {
		return nil, err
	}

	sql, _, err := dialect.Insert("service_accounts").
		Rows(goqu.Record{
			"id":                  newResourceID(),
			"version":             initialResourceVersion,
			"created_at":          timestamp,
			"updated_at":          timestamp,
			"name":                serviceAccount.Name,
			"description":         serviceAccount.Description,
			"group_id":            serviceAccount.GroupID,
			"created_by":          serviceAccount.CreatedBy,
			"oidc_trust_policies": trustPoliciesJSON,
		}).
		Returning(serviceAccountFieldList...).ToSQL()

	if err != nil {
		return nil, err
	}

	createdServiceAccount, err := scanServiceAccount(tx.QueryRow(ctx, sql), false)

	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				return nil, errors.NewError(
					errors.EConflict,
					fmt.Sprintf("Service account with name %s already exists in group %s", serviceAccount.Name, serviceAccount.GroupID),
				)
			}
			if isForeignKeyViolation(pgErr) && pgErr.ConstraintName == "fk_group_id" {
				return nil, errors.NewError(errors.EConflict, "invalid group: the specified group does not exist")
			}
		}
		return nil, err
	}

	// Lookup namespace for group
	namespace, err := getNamespaceByGroupID(ctx, tx, createdServiceAccount.GroupID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	createdServiceAccount.ResourcePath = buildServiceAccountResourcePath(namespace.path, createdServiceAccount.Name)

	return createdServiceAccount, nil
}

// UpdateServiceAccount updates an existing serviceAccount by name
func (s *serviceAccounts) UpdateServiceAccount(ctx context.Context, serviceAccount *models.ServiceAccount) (*models.ServiceAccount, error) {
	trustPoliciesJSON, err := s.marshalOIDCTrustPolicies(serviceAccount.OIDCTrustPolicies)
	if err != nil {
		return nil, err
	}

	timestamp := currentTime()

	tx, err := s.dbClient.getConnection(ctx).Begin(ctx)
	if err != nil {
		return nil, err
	}

	// Rollback is safe to call even if the tx is already closed, so if
	// the tx commits successfully, this is a no-op
	defer func() {
		if txErr := tx.Rollback(ctx); txErr != nil && txErr != pgx.ErrTxClosed {
			s.dbClient.logger.Errorf("failed to rollback tx for UpdateServiceAccount: %v", txErr)
		}
	}()

	sql, _, err := goqu.Update("service_accounts").Set(
		goqu.Record{
			"version":             goqu.L("? + ?", goqu.C("version"), 1),
			"updated_at":          timestamp,
			"description":         serviceAccount.Description,
			"oidc_trust_policies": trustPoliciesJSON,
		},
	).Where(goqu.Ex{"id": serviceAccount.Metadata.ID, "version": serviceAccount.Metadata.Version}).Returning(serviceAccountFieldList...).ToSQL()

	if err != nil {
		return nil, err
	}

	updatedServiceAccount, err := scanServiceAccount(tx.QueryRow(ctx, sql), false)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrOptimisticLockError
		}
		return nil, err
	}

	// Lookup namespace for group
	namespace, err := getNamespaceByGroupID(ctx, tx, updatedServiceAccount.GroupID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	updatedServiceAccount.ResourcePath = buildServiceAccountResourcePath(namespace.path, updatedServiceAccount.Name)

	return updatedServiceAccount, nil
}

func (s *serviceAccounts) DeleteServiceAccount(ctx context.Context, serviceAccount *models.ServiceAccount) error {
	sql, _, err := dialect.Delete("service_accounts").Where(
		goqu.Ex{
			"id":      serviceAccount.Metadata.ID,
			"version": serviceAccount.Metadata.Version,
		},
	).Returning(serviceAccountFieldList...).ToSQL()

	if err != nil {
		return err
	}

	if _, err := scanServiceAccount(s.dbClient.getConnection(ctx).QueryRow(ctx, sql), false); err != nil {
		if err == pgx.ErrNoRows {
			return ErrOptimisticLockError
		}

		if pgErr := asPgError(err); pgErr != nil {
			if isForeignKeyViolation(pgErr) {
				return errors.NewError(
					errors.EConflict,
					fmt.Sprintf("Service account %s is assigned as a member of a group/workspace", serviceAccount.Name),
				)
			}
		}

		return err
	}

	return nil
}

func (s *serviceAccounts) getServiceAccount(ctx context.Context, exp exp.Ex) (*models.ServiceAccount, error) {
	sql, _, err := goqu.From("service_accounts").
		Select(s.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"service_accounts.group_id": goqu.I("namespaces.group_id")})).
		Where(exp).
		ToSQL()

	if err != nil {
		return nil, err
	}

	serviceAccount, err := scanServiceAccount(s.dbClient.getConnection(ctx).QueryRow(ctx, sql), true)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return serviceAccount, nil
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
			Issuer:      p.Issuer,
			BoundClaims: p.BoundClaims,
		})
	}
	trustPoliciesJSON, err := json.Marshal(policies)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal trust policies to JSON %v", err)
	}
	return trustPoliciesJSON, nil
}

func buildServiceAccountResourcePath(groupPath string, name string) string {
	return fmt.Sprintf("%s/%s", groupPath, name)
}

func scanServiceAccount(row scanner, withResourcePath bool) (*models.ServiceAccount, error) {
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
	}
	var path string
	if withResourcePath {
		fields = append(fields, &path)
	}

	err := row.Scan(fields...)
	if err != nil {
		return nil, err
	}

	serviceAccount.OIDCTrustPolicies = []models.OIDCTrustPolicy{}
	for _, p := range policies {
		serviceAccount.OIDCTrustPolicies = append(serviceAccount.OIDCTrustPolicies, models.OIDCTrustPolicy{
			Issuer:      p.Issuer,
			BoundClaims: p.BoundClaims,
		})
	}

	if withResourcePath {
		serviceAccount.ResourcePath = buildServiceAccountResourcePath(path, serviceAccount.Name)
	}

	return serviceAccount, nil
}

func serviceAccountFieldResolver(key string, model interface{}) (string, error) {
	serviceAccount, ok := model.(*models.ServiceAccount)
	if !ok {
		return "", errors.NewError(errors.EInternal, fmt.Sprintf("Expected ServiceAccount type, got %T", model))
	}

	val, ok := metadataFieldResolver(key, &serviceAccount.Metadata)
	if !ok {
		return "", errors.NewError(errors.EInternal, fmt.Sprintf("Invalid field key requested %s", key))
	}

	return val, nil
}
