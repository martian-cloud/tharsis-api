package db

//go:generate go tool mockery --name ManagedIdentities --inpackage --case underscore

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v4"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
	"go.opentelemetry.io/otel/attribute"
)

// ManagedIdentities encapsulates the logic to access managed identities from the database
type ManagedIdentities interface {
	GetManagedIdentityByID(ctx context.Context, id string) (*models.ManagedIdentity, error)
	GetManagedIdentityByTRN(ctx context.Context, trn string) (*models.ManagedIdentity, error)
	GetManagedIdentitiesForWorkspace(ctx context.Context, workspaceID string) ([]models.ManagedIdentity, error)
	AddManagedIdentityToWorkspace(ctx context.Context, managedIdentityID string, workspaceID string) error
	RemoveManagedIdentityFromWorkspace(ctx context.Context, managedIdentityID string, workspaceID string) error
	CreateManagedIdentity(ctx context.Context, managedIdentity *models.ManagedIdentity) (*models.ManagedIdentity, error)
	UpdateManagedIdentity(ctx context.Context, managedIdentity *models.ManagedIdentity) (*models.ManagedIdentity, error)
	GetManagedIdentities(ctx context.Context, input *GetManagedIdentitiesInput) (*ManagedIdentitiesResult, error)
	DeleteManagedIdentity(ctx context.Context, managedIdentity *models.ManagedIdentity) error
	GetManagedIdentityAccessRules(ctx context.Context, input *GetManagedIdentityAccessRulesInput) (*ManagedIdentityAccessRulesResult, error)
	GetManagedIdentityAccessRuleByID(ctx context.Context, ruleID string) (*models.ManagedIdentityAccessRule, error)
	GetManagedIdentityAccessRuleByTRN(ctx context.Context, trn string) (*models.ManagedIdentityAccessRule, error)
	CreateManagedIdentityAccessRule(ctx context.Context, rule *models.ManagedIdentityAccessRule) (*models.ManagedIdentityAccessRule, error)
	UpdateManagedIdentityAccessRule(ctx context.Context, rule *models.ManagedIdentityAccessRule) (*models.ManagedIdentityAccessRule, error)
	DeleteManagedIdentityAccessRule(ctx context.Context, rule *models.ManagedIdentityAccessRule) error
}

// ManagedIdentitySortableField represents the fields that a managed identity can be sorted by
type ManagedIdentitySortableField string

// ManagedIdentitySortableField constants
const (
	ManagedIdentitySortableFieldCreatedAtAsc   ManagedIdentitySortableField = "CREATED_AT_ASC"
	ManagedIdentitySortableFieldCreatedAtDesc  ManagedIdentitySortableField = "CREATED_AT_DESC"
	ManagedIdentitySortableFieldUpdatedAtAsc   ManagedIdentitySortableField = "UPDATED_AT_ASC"
	ManagedIdentitySortableFieldUpdatedAtDesc  ManagedIdentitySortableField = "UPDATED_AT_DESC"
	ManagedIdentitySortableFieldGroupLevelAsc  ManagedIdentitySortableField = "GROUP_LEVEL_ASC"
	ManagedIdentitySortableFieldGroupLevelDesc ManagedIdentitySortableField = "GROUP_LEVEL_DESC"
)

func (sf ManagedIdentitySortableField) getFieldDescriptor() *pagination.FieldDescriptor {
	switch sf {
	case ManagedIdentitySortableFieldCreatedAtAsc, ManagedIdentitySortableFieldCreatedAtDesc:
		return &pagination.FieldDescriptor{Key: "created_at", Table: "t1", Col: "created_at"}
	case ManagedIdentitySortableFieldUpdatedAtAsc, ManagedIdentitySortableFieldUpdatedAtDesc:
		return &pagination.FieldDescriptor{Key: "updated_at", Table: "t1", Col: "updated_at"}
	case ManagedIdentitySortableFieldGroupLevelAsc, ManagedIdentitySortableFieldGroupLevelDesc:
		return &pagination.FieldDescriptor{Key: "group_path", Table: "namespaces", Col: "path"}
	default:
		return nil
	}
}

func (sf ManagedIdentitySortableField) getSortDirection() pagination.SortDirection {
	if strings.HasSuffix(string(sf), "_DESC") {
		return pagination.DescSort
	}
	return pagination.AscSort
}

func (sf ManagedIdentitySortableField) getTransformFunc() pagination.SortTransformFunc {
	switch sf {
	case ManagedIdentitySortableFieldGroupLevelAsc, ManagedIdentitySortableFieldGroupLevelDesc:
		return func(s string) string {
			return fmt.Sprintf("array_length(string_to_array(%s, '/'), 1)", s)
		}
	default:
		return nil
	}
}

// ManagedIdentityAccessRuleSortableField represents the fields that a managed identity access rule can be sorted by
type ManagedIdentityAccessRuleSortableField string

func (sf ManagedIdentityAccessRuleSortableField) getRuleFieldDescriptor() *pagination.FieldDescriptor {
	switch sf {
	default:
		return nil
	}
}

func (sf ManagedIdentityAccessRuleSortableField) getRuleSortDirection() pagination.SortDirection {
	if strings.HasSuffix(string(sf), "_DESC") {
		return pagination.DescSort
	}
	return pagination.AscSort
}

// ManagedIdentityFilter contains the supported fields for filtering ManagedIdentity resources
type ManagedIdentityFilter struct {
	Search             *string
	AliasSourceID      *string
	NamespacePaths     []string
	ManagedIdentityIDs []string
}

// ManagedIdentityAccessRuleFilter contains the supported fields for filtering ManagedIdentityAccessRule resources
type ManagedIdentityAccessRuleFilter struct {
	ManagedIdentityID            *string
	ManagedIdentityAccessRuleIDs []string
}

// GetManagedIdentitiesInput is the input for listing managed identities
type GetManagedIdentitiesInput struct {
	// Sort specifies the field to sort on and direction
	Sort *ManagedIdentitySortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// Filter is used to filter the results
	Filter *ManagedIdentityFilter
}

// GetManagedIdentityAccessRulesInput is the input for listing managed identity access rules
type GetManagedIdentityAccessRulesInput struct {
	// Sort specifies the field to sort on and direction
	Sort *ManagedIdentityAccessRuleSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// Filter is used to filter the results
	Filter *ManagedIdentityAccessRuleFilter
}

// ManagedIdentitiesResult contains the response data and page information
type ManagedIdentitiesResult struct {
	PageInfo          *pagination.PageInfo
	ManagedIdentities []models.ManagedIdentity
}

// ManagedIdentityAccessRulesResult contains the response data and page information
type ManagedIdentityAccessRulesResult struct {
	PageInfo                   *pagination.PageInfo
	ManagedIdentityAccessRules []models.ManagedIdentityAccessRule
}

type managedIdentities struct {
	dbClient *Client
}

var (
	managedIdentityFieldList = append(metadataFieldList,
		"name", "description", "type", "group_id", "data", "created_by", "alias_source_id")
	managedIdentityRuleFieldList = append(metadataFieldList,
		"run_stage", "managed_identity_id", "type", "module_attestation_policies", "verify_state_lineage")
)

// Table aliases used with several queries.
var (
	t1 = goqu.From("managed_identities").As("t1")
	t2 = goqu.T("managed_identities").As("t2")
)

// NewManagedIdentities returns an instance of the ManagedIdentity interface
func NewManagedIdentities(dbClient *Client) ManagedIdentities {
	return &managedIdentities{dbClient: dbClient}
}

func (m *managedIdentities) GetManagedIdentityAccessRules(ctx context.Context,
	input *GetManagedIdentityAccessRulesInput,
) (*ManagedIdentityAccessRulesResult, error) {
	ctx, span := tracer.Start(ctx, "db.GetManagedIdentityAccessRules")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	conn := m.dbClient.getConnection(ctx)
	ex := goqu.And()

	if input.Filter != nil {

		if input.Filter.ManagedIdentityID != nil {
			ex = ex.Append(
				goqu.Or(
					goqu.I("managed_identity_id").
						Eq(dialect.From("managed_identities").
							Select("managed_identities.alias_source_id").
							Where(goqu.Ex{"managed_identities.id": input.Filter.ManagedIdentityID})),
					goqu.I("managed_identity_id").
						Eq(dialect.From("managed_identities").
							Select("managed_identities.id").
							Where(goqu.Ex{"managed_identities.id": input.Filter.ManagedIdentityID})),
				),
			)
		}

		if input.Filter.ManagedIdentityAccessRuleIDs != nil {
			ex = ex.Append(goqu.I("managed_identity_rules.id").In(input.Filter.ManagedIdentityAccessRuleIDs))
		}
	}

	query := dialect.From("managed_identity_rules").
		Select(m.getManagedIdentityRuleSelectFields()...).
		InnerJoin(goqu.T("managed_identities"), goqu.On(goqu.Ex{"managed_identity_rules.managed_identity_id": goqu.I("managed_identities.id")})).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"managed_identities.group_id": goqu.I("namespaces.group_id")})).
		Where(ex)

	sortDirection := pagination.AscSort

	var sortBy *pagination.FieldDescriptor
	if input.Sort != nil {
		sortDirection = input.Sort.getRuleSortDirection()
		sortBy = input.Sort.getRuleFieldDescriptor()
	}

	qBuilder, err := pagination.NewPaginatedQueryBuilder(
		input.PaginationOptions,
		&pagination.FieldDescriptor{Key: "id", Table: "managed_identity_rules", Col: "id"},
		pagination.WithSortByField(sortBy, sortDirection),
	)
	if err != nil {
		tracing.RecordError(span, err, "failed to build query")
		return nil, err
	}

	rows, err := qBuilder.Execute(ctx, conn, query)
	if err != nil {
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	defer rows.Close()

	// Scan rows
	rules := []models.ManagedIdentityAccessRule{}
	for rows.Next() {
		rule, err := scanManagedIdentityRule(rows)
		if err != nil {
			tracing.RecordError(span, err, "failed to scan row")
			return nil, err
		}

		rules = append(rules, *rule)
	}

	for i, rule := range rules {
		allowedUserIDs, err := m.getManagedIdentityAccessRuleAllowedUserIDs(ctx, conn, rule.Metadata.ID)
		if err != nil {
			tracing.RecordError(span, err, "failed to get managed identity access rule allowed user IDs")
			return nil, err
		}

		allowedServiceAccountIDs, err := m.getManagedIdentityAccessRuleAllowedServiceAccountIDs(ctx, conn, rule.Metadata.ID)
		if err != nil {
			tracing.RecordError(span, err, "failed to get managed identity access rule allowed service account IDs")
			return nil, err
		}

		allowedTeamIDs, err := m.getManagedIdentityAccessRuleAllowedTeamIDs(ctx, conn, rule.Metadata.ID)
		if err != nil {
			tracing.RecordError(span, err, "failed to get managed identity access rule allowed team IDs")
			return nil, err
		}

		rules[i].AllowedUserIDs = allowedUserIDs
		rules[i].AllowedServiceAccountIDs = allowedServiceAccountIDs
		rules[i].AllowedTeamIDs = allowedTeamIDs
	}

	result := ManagedIdentityAccessRulesResult{
		PageInfo:                   rows.GetPageInfo(),
		ManagedIdentityAccessRules: rules,
	}

	return &result, nil
}

func (m *managedIdentities) GetManagedIdentityAccessRuleByID(ctx context.Context, ruleID string) (*models.ManagedIdentityAccessRule, error) {
	ctx, span := tracer.Start(ctx, "db.GetManagedIdentityAccessRuleByID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	return m.getManagedIdentityAccessRule(ctx, goqu.Ex{"managed_identity_rules.id": ruleID})
}

func (m *managedIdentities) GetManagedIdentityAccessRuleByTRN(ctx context.Context, trn string) (*models.ManagedIdentityAccessRule, error) {
	ctx, span := tracer.Start(ctx, "db.GetManagedIdentityAccessRuleByTRN")
	defer span.End()

	path, err := types.ManagedIdentityAccessRuleModelType.ResourcePathFromTRN(trn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse TRN", errors.WithSpan(span))
	}

	parts := strings.Split(path, "/")
	if len(parts) < 3 {
		return nil, errors.New("a managed identity access rule TRN must have the group path, managed identity name, and rule GID separated by a forward slash",
			errors.WithErrorCode(errors.EInvalid),
			errors.WithSpan(span),
		)
	}

	return m.getManagedIdentityAccessRule(ctx, goqu.Ex{
		"managed_identity_rules.id": gid.FromGlobalID(parts[len(parts)-1]),
		"managed_identities.name":   parts[len(parts)-2],
		"namespaces.path":           strings.Join(parts[:len(parts)-2], "/"),
	})
}

func (m *managedIdentities) CreateManagedIdentityAccessRule(ctx context.Context, rule *models.ManagedIdentityAccessRule) (*models.ManagedIdentityAccessRule, error) {
	ctx, span := tracer.Start(ctx, "db.CreateManagedIdentityAccessRule")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	timestamp := currentTime()

	tx, err := m.dbClient.getConnection(ctx).Begin(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return nil, err
	}

	// Rollback is safe to call even if the tx is already closed, so if
	// the tx commits successfully, this is a no-op
	defer func() {
		if txErr := tx.Rollback(ctx); txErr != nil && txErr != pgx.ErrTxClosed {
			m.dbClient.logger.WithContextFields(ctx).Errorf("failed to rollback tx for CreateManagedIdentityAccessRule: %v", txErr)
		}
	}()

	var moduleAttestationPolicies interface{}
	if rule.ModuleAttestationPolicies != nil {
		moduleAttestationPolicies, err = json.Marshal(rule.ModuleAttestationPolicies)
		if err != nil {
			tracing.RecordError(span, err, "failed to marshal module attestation policies")
			return nil, err
		}
	}

	// Create rule
	sql, args, err := dialect.From("managed_identity_rules").
		Prepared(true).
		With("managed_identity_rules",
			dialect.Insert("managed_identity_rules").
				Rows(goqu.Record{
					"id":                          newResourceID(),
					"version":                     initialResourceVersion,
					"created_at":                  timestamp,
					"updated_at":                  timestamp,
					"type":                        rule.Type,
					"managed_identity_id":         rule.ManagedIdentityID,
					"run_stage":                   rule.RunStage,
					"module_attestation_policies": moduleAttestationPolicies,
					"verify_state_lineage":        rule.VerifyStateLineage,
				}).
				Returning("*"),
		).Select(m.getManagedIdentityRuleSelectFields()...).
		InnerJoin(goqu.T("managed_identities"), goqu.On(goqu.Ex{"managed_identity_rules.managed_identity_id": goqu.I("managed_identities.id")})).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"managed_identities.group_id": goqu.I("namespaces.group_id")})).
		ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	createdRule, err := scanManagedIdentityRule(tx.QueryRow(ctx, sql, args...))
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				tracing.RecordError(span, nil, "rule for run stage %s already exists", rule.RunStage)
				return nil, errors.New("rule for run stage %s already exists", rule.RunStage, errors.WithErrorCode(errors.EConflict))
			}
		}
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	// Create allowed users
	for _, userID := range rule.AllowedUserIDs {
		sql, args, err := dialect.Insert("managed_identity_rule_allowed_users").
			Prepared(true).
			Rows(goqu.Record{
				"id":      newResourceID(),
				"rule_id": createdRule.Metadata.ID,
				"user_id": userID,
			}).ToSQL()
		if err != nil {
			tracing.RecordError(span, err, "failed to generate SQL")
			return nil, err
		}

		if _, err := tx.Exec(ctx, sql, args...); err != nil {
			tracing.RecordError(span, err, "failed to execute DB query")
			return nil, err
		}
	}

	// Create allowed service accounts
	for _, serviceAccountID := range rule.AllowedServiceAccountIDs {
		sql, args, err := dialect.Insert("managed_identity_rule_allowed_service_accounts").
			Prepared(true).
			Rows(goqu.Record{
				"id":                 newResourceID(),
				"rule_id":            createdRule.Metadata.ID,
				"service_account_id": serviceAccountID,
			}).ToSQL()
		if err != nil {
			tracing.RecordError(span, err, "failed to generate SQL")
			return nil, err
		}

		if _, err := tx.Exec(ctx, sql, args...); err != nil {
			tracing.RecordError(span, err, "failed to execute DB query")
			return nil, err
		}
	}

	// Create allowed teams
	for _, teamID := range rule.AllowedTeamIDs {
		sql, args, err := dialect.Insert("managed_identity_rule_allowed_teams").
			Prepared(true).
			Rows(goqu.Record{
				"id":      newResourceID(),
				"rule_id": createdRule.Metadata.ID,
				"team_id": teamID,
			}).ToSQL()
		if err != nil {
			tracing.RecordError(span, err, "failed to generate SQL")
			return nil, err
		}

		if _, err := tx.Exec(ctx, sql, args...); err != nil {
			tracing.RecordError(span, err, "failed to execute DB query")
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return nil, err
	}

	createdRule.AllowedUserIDs = rule.AllowedUserIDs
	createdRule.AllowedServiceAccountIDs = rule.AllowedServiceAccountIDs
	createdRule.AllowedTeamIDs = rule.AllowedTeamIDs

	return createdRule, nil
}

func (m *managedIdentities) UpdateManagedIdentityAccessRule(ctx context.Context, rule *models.ManagedIdentityAccessRule) (*models.ManagedIdentityAccessRule, error) {
	ctx, span := tracer.Start(ctx, "db.UpdateManagedIdentityAccessRule")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	timestamp := currentTime()

	tx, err := m.dbClient.getConnection(ctx).Begin(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return nil, err
	}

	// Rollback is safe to call even if the tx is already closed, so if
	// the tx commits successfully, this is a no-op
	defer func() {
		if txErr := tx.Rollback(ctx); txErr != nil && txErr != pgx.ErrTxClosed {
			m.dbClient.logger.WithContextFields(ctx).Errorf("failed to rollback tx for UpdateManagedIdentityAccessRule: %v", txErr)
		}
	}()

	var moduleAttestationPolicies interface{}
	if rule.ModuleAttestationPolicies != nil {
		moduleAttestationPolicies, err = json.Marshal(rule.ModuleAttestationPolicies)
		if err != nil {
			tracing.RecordError(span, err, "failed to marshal module attestation policies")
			return nil, err
		}
	}

	sql, args, err := dialect.From("managed_identity_rules").
		Prepared(true).
		With("managed_identity_rules",
			dialect.Update("managed_identity_rules").
				Set(
					goqu.Record{
						"version":                     goqu.L("? + ?", goqu.C("version"), 1),
						"updated_at":                  timestamp,
						"run_stage":                   rule.RunStage,
						"module_attestation_policies": moduleAttestationPolicies,
						"verify_state_lineage":        rule.VerifyStateLineage,
					},
				).Where(goqu.Ex{"id": rule.Metadata.ID, "version": rule.Metadata.Version}).
				Returning("*"),
		).Select(m.getManagedIdentityRuleSelectFields()...).
		InnerJoin(goqu.T("managed_identities"), goqu.On(goqu.Ex{"managed_identity_rules.managed_identity_id": goqu.I("managed_identities.id")})).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"managed_identities.group_id": goqu.I("namespaces.group_id")})).
		ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	updatedRule, err := scanManagedIdentityRule(tx.QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return nil, ErrOptimisticLockError
		}
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				tracing.RecordError(span, nil, "rule for run stage %s already exists", rule.RunStage)
				return nil, errors.New("rule for run stage %s already exists", rule.RunStage, errors.WithErrorCode(errors.EConflict))
			}
		}
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	// Delete allowed users
	deleteAllowedUsersSQL, args, err := dialect.Delete("managed_identity_rule_allowed_users").
		Prepared(true).
		Where(
			goqu.Ex{
				"rule_id": rule.Metadata.ID,
			},
		).ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	if _, err = tx.Exec(ctx, deleteAllowedUsersSQL, args...); err != nil {
		tracing.RecordError(span, err, "failed to execute DB query")
		return nil, err
	}

	// Delete allowed service accounts
	deleteAllowedServiceAccountsSQL, args, err := dialect.Delete("managed_identity_rule_allowed_service_accounts").
		Prepared(true).
		Where(
			goqu.Ex{
				"rule_id": rule.Metadata.ID,
			},
		).ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	if _, err = tx.Exec(ctx, deleteAllowedServiceAccountsSQL, args...); err != nil {
		tracing.RecordError(span, err, "failed to execute DB query")
		return nil, err
	}

	// Delete allowed teams
	deleteAllowedTeamsSQL, args, err := dialect.Delete("managed_identity_rule_allowed_teams").
		Prepared(true).
		Where(
			goqu.Ex{
				"rule_id": rule.Metadata.ID,
			},
		).ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	if _, err := tx.Exec(ctx, deleteAllowedTeamsSQL, args...); err != nil {
		tracing.RecordError(span, err, "failed to execute DB query")
		return nil, err
	}

	// Create allowed users
	for _, userID := range rule.AllowedUserIDs {
		sql, args, err := dialect.Insert("managed_identity_rule_allowed_users").
			Prepared(true).
			Rows(goqu.Record{
				"id":      newResourceID(),
				"rule_id": rule.Metadata.ID,
				"user_id": userID,
			}).ToSQL()
		if err != nil {
			tracing.RecordError(span, err, "failed to generate SQL")
			return nil, err
		}

		if _, err := tx.Exec(ctx, sql, args...); err != nil {
			tracing.RecordError(span, err, "failed to execute DB query")
			return nil, err
		}
	}

	// Create allowed service accounts
	for _, serviceAccountID := range rule.AllowedServiceAccountIDs {
		sql, args, err := dialect.Insert("managed_identity_rule_allowed_service_accounts").
			Prepared(true).
			Rows(goqu.Record{
				"id":                 newResourceID(),
				"rule_id":            rule.Metadata.ID,
				"service_account_id": serviceAccountID,
			}).ToSQL()
		if err != nil {
			tracing.RecordError(span, err, "failed to generate SQL")
			return nil, err
		}

		if _, err := tx.Exec(ctx, sql, args...); err != nil {
			tracing.RecordError(span, err, "failed to execute DB query")
			return nil, err
		}
	}

	// Create allowed teams
	for _, teamID := range rule.AllowedTeamIDs {
		sql, args, err := dialect.Insert("managed_identity_rule_allowed_teams").
			Prepared(true).
			Rows(goqu.Record{
				"id":      newResourceID(),
				"rule_id": rule.Metadata.ID,
				"team_id": teamID,
			}).ToSQL()
		if err != nil {
			tracing.RecordError(span, err, "failed to generate SQL")
			return nil, err
		}

		if _, err := tx.Exec(ctx, sql, args...); err != nil {
			tracing.RecordError(span, err, "failed to execute DB query")
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return nil, err
	}

	updatedRule.AllowedUserIDs = rule.AllowedUserIDs
	updatedRule.AllowedServiceAccountIDs = rule.AllowedServiceAccountIDs
	updatedRule.AllowedTeamIDs = rule.AllowedTeamIDs

	return updatedRule, nil
}

func (m *managedIdentities) DeleteManagedIdentityAccessRule(ctx context.Context, rule *models.ManagedIdentityAccessRule) error {
	ctx, span := tracer.Start(ctx, "db.DeleteManagedIdentityAccessRule")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	sql, args, err := dialect.From("managed_identity_rules").
		Prepared(true).
		With("managed_identity_rules",
			dialect.Delete("managed_identity_rules").
				Where(
					goqu.Ex{
						"id":      rule.Metadata.ID,
						"version": rule.Metadata.Version,
					},
				).Returning("*"),
		).Select(m.getManagedIdentityRuleSelectFields()...).
		InnerJoin(goqu.T("managed_identities"), goqu.On(goqu.Ex{"managed_identity_rules.managed_identity_id": goqu.I("managed_identities.id")})).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"managed_identities.group_id": goqu.I("namespaces.group_id")})).
		ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return err
	}

	if _, err := scanManagedIdentityRule(m.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...)); err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return ErrOptimisticLockError
		}

		tracing.RecordError(span, err, "failed to execute query")
		return err
	}

	return nil
}

func (m *managedIdentities) GetManagedIdentitiesForWorkspace(ctx context.Context, workspaceID string) ([]models.ManagedIdentity, error) {
	ctx, span := tracer.Start(ctx, "db.GetManagedIdentitiesForWorkspace")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	sql, args, err := dialect.From(t1).
		Prepared(true).
		Select(m.getManagedIdentitySelectFields(true)...).
		InnerJoin(goqu.T("workspace_managed_identity_relation"), goqu.On(goqu.Ex{"t1.id": goqu.I("workspace_managed_identity_relation.managed_identity_id")})).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"t1.group_id": goqu.I("namespaces.group_id")})).
		LeftJoin(t2, goqu.On(goqu.Ex{"t1.alias_source_id": goqu.I("t2.id")})).
		Where(goqu.Ex{"workspace_managed_identity_relation.workspace_id": workspaceID}).ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	rows, err := m.dbClient.getConnection(ctx).Query(ctx, sql, args...)
	if err != nil {
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	defer rows.Close()

	// Scan rows
	results := []models.ManagedIdentity{}
	for rows.Next() {
		item, err := scanManagedIdentity(rows, true)
		if err != nil {
			tracing.RecordError(span, err, "failed to scan row")
			return nil, err
		}

		results = append(results, *item)
	}

	return results, nil
}

func (m *managedIdentities) AddManagedIdentityToWorkspace(ctx context.Context, managedIdentityID string, workspaceID string) error {
	ctx, span := tracer.Start(ctx, "db.AddManagedIdentityToWorkspace")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	sql, args, err := dialect.Insert("workspace_managed_identity_relation").
		Prepared(true).
		Rows(goqu.Record{
			"managed_identity_id": managedIdentityID,
			"workspace_id":        workspaceID,
		}).ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return err
	}

	if _, err = m.dbClient.getConnection(ctx).Exec(ctx, sql, args...); err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				tracing.RecordError(span, nil, "managed identity already assigned to workspace")
				return errors.New("managed identity already assigned to workspace", errors.WithErrorCode(errors.EConflict))
			}
		}
		tracing.RecordError(span, err, "failed to execute DB query")
		return err
	}

	return nil
}

func (m *managedIdentities) RemoveManagedIdentityFromWorkspace(ctx context.Context, managedIdentityID string, workspaceID string) error {
	ctx, span := tracer.Start(ctx, "db.RemoveManagedIdentityFromWorkspace")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	sql, args, err := dialect.Delete("workspace_managed_identity_relation").
		Prepared(true).
		Where(
			goqu.Ex{
				"managed_identity_id": managedIdentityID,
				"workspace_id":        workspaceID,
			},
		).ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return err
	}

	if _, err = m.dbClient.getConnection(ctx).Exec(ctx, sql, args...); err != nil {
		tracing.RecordError(span, err, "failed to execute DB query")
		return err
	}

	return nil
}

// GetManagedIdentityByID returns a managedIdentity by ID
func (m *managedIdentities) GetManagedIdentityByID(ctx context.Context, id string) (*models.ManagedIdentity, error) {
	ctx, span := tracer.Start(ctx, "db.GetManagedIdentityByID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	return m.getManagedIdentity(ctx, goqu.Ex{"t1.id": id})
}

// GetManagedIdentityByTRN returns a managedIdentity by TRN
func (m *managedIdentities) GetManagedIdentityByTRN(ctx context.Context, trn string) (*models.ManagedIdentity, error) {
	ctx, span := tracer.Start(ctx, "db.GetManagedIdentityByTRN")
	span.SetAttributes(attribute.String("trn", trn))
	defer span.End()

	path, err := types.ManagedIdentityModelType.ResourcePathFromTRN(trn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse TRN", errors.WithSpan(span))
	}

	lastSlashIndex := strings.LastIndex(path, "/")

	if lastSlashIndex == -1 {
		return nil, errors.New("a managed identity TRN must have a group path and identity name separated by a forward slash",
			errors.WithErrorCode(errors.EInvalid),
			errors.WithSpan(span),
		)
	}

	return m.getManagedIdentity(ctx, goqu.Ex{
		"t1.name":         path[lastSlashIndex+1:],
		"namespaces.path": path[:lastSlashIndex],
	})
}

func (m *managedIdentities) GetManagedIdentities(ctx context.Context, input *GetManagedIdentitiesInput) (*ManagedIdentitiesResult, error) {
	ctx, span := tracer.Start(ctx, "db.GetManagedIdentities")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	ex := goqu.And()

	if input.Filter != nil {
		if input.Filter.NamespacePaths != nil {
			ex = ex.Append(goqu.I("namespaces.path").In(input.Filter.NamespacePaths))
		}

		if input.Filter.Search != nil {
			search := *input.Filter.Search

			lastDelimiterIndex := strings.LastIndex(search, "/")

			if lastDelimiterIndex != -1 {
				namespacePath := search[:lastDelimiterIndex]
				managedIdentityName := search[lastDelimiterIndex+1:]

				if managedIdentityName != "" {
					// An OR condition is used here since the last component of the search path could be part of
					// the namespace or it can be a managed identity name prefix
					ex = ex.Append(
						goqu.Or(
							goqu.And(
								goqu.I("namespaces.path").Eq(namespacePath),
								goqu.I("t1.name").ILike(managedIdentityName+"%"),
							),
							goqu.Or(
								goqu.I("namespaces.path").ILike(search+"%"),
								goqu.I("t1.name").ILike(managedIdentityName+"%"),
							),
						),
					)
				} else {
					// We know the search is a namespace path since it ends with a "/"
					ex = ex.Append(goqu.I("namespaces.path").ILike(namespacePath + "%"))
				}
			} else {
				// We don't know if the search is for a namespace path or managed identity name; therefore, use
				// an OR condition to search both
				ex = ex.Append(
					goqu.Or(
						goqu.I("namespaces.path").ILike(search+"%"),
						goqu.I("t1.name").ILike(search+"%"),
					),
				)
			}
		}

		if input.Filter.AliasSourceID != nil {
			ex = ex.Append(goqu.Ex{"t1.alias_source_id": *input.Filter.AliasSourceID})
		}

		if input.Filter.ManagedIdentityIDs != nil {
			// This check avoids an SQL syntax error if an empty slice is provided.
			if len(input.Filter.ManagedIdentityIDs) > 0 {
				ex = ex.Append(goqu.I("t1.id").In(input.Filter.ManagedIdentityIDs))
			}
		}
	}

	query := dialect.From(t1).
		Select(m.getManagedIdentitySelectFields(true)...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"t1.group_id": goqu.I("namespaces.group_id")})).
		LeftJoin(t2, goqu.On(goqu.Ex{"t1.alias_source_id": goqu.I("t2.id")})).
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
		&pagination.FieldDescriptor{Key: "id", Table: "t1", Col: "id"},
		pagination.WithSortByField(sortBy, sortDirection),
		pagination.WithSortByTransform(sortTransformFunc),
	)
	if err != nil {
		tracing.RecordError(span, err, "failed to build query")
		return nil, err
	}

	rows, err := qBuilder.Execute(ctx, m.dbClient.getConnection(ctx), query)
	if err != nil {
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	defer rows.Close()

	// Scan rows
	results := []models.ManagedIdentity{}
	for rows.Next() {
		item, err := scanManagedIdentity(rows, true)
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

	result := ManagedIdentitiesResult{
		PageInfo:          rows.GetPageInfo(),
		ManagedIdentities: results,
	}

	return &result, nil
}

// CreateManagedIdentity creates a new managedIdentity
func (m *managedIdentities) CreateManagedIdentity(ctx context.Context, managedIdentity *models.ManagedIdentity) (*models.ManagedIdentity, error) {
	ctx, span := tracer.Start(ctx, "db.CreateManagedIdentity")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	timestamp := currentTime()
	createdID := newResourceID()

	tx, err := m.dbClient.getConnection(ctx).Begin(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return nil, err
	}

	// Rollback is safe to call even if the tx is already closed, so if
	// the tx commits successfully, this is a no-op
	defer func() {
		if txErr := tx.Rollback(ctx); txErr != nil && txErr != pgx.ErrTxClosed {
			m.dbClient.logger.WithContextFields(ctx).Errorf("failed to rollback tx for CreateManagedIdentity: %v", txErr)
		}
	}()

	sql, args, err := dialect.Insert("managed_identities").
		Prepared(true).
		Rows(goqu.Record{
			"id":              createdID,
			"version":         initialResourceVersion,
			"created_at":      timestamp,
			"updated_at":      timestamp,
			"name":            managedIdentity.Name,
			"description":     managedIdentity.Description,
			"type":            managedIdentity.Type,
			"group_id":        managedIdentity.GroupID,
			"data":            managedIdentity.Data,
			"created_by":      managedIdentity.CreatedBy,
			"alias_source_id": managedIdentity.AliasSourceID,
		}).ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	if _, err = tx.Exec(ctx, sql, args...); err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				tracing.RecordError(span, nil, "managed identity already exists in the specified group")
				return nil, errors.New("managed identity already exists in the specified group", errors.WithErrorCode(errors.EConflict))
			}
		}
		tracing.RecordError(span, err, "failed to execute DB query")
		return nil, err
	}

	// A separate query allows backfilling empty columns in the alias with that of the source managed identity.
	sql, args, err = dialect.From(t1).
		Prepared(true).
		Select(m.getManagedIdentitySelectFields(true)...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"t1.group_id": goqu.I("namespaces.group_id")})).
		LeftJoin(t2, goqu.On(goqu.Ex{"t1.alias_source_id": goqu.I("t2.id")})).
		Where(goqu.Ex{"t1.id": createdID}).
		ToSQL()

	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	createdManagedIdentity, err := scanManagedIdentity(tx.QueryRow(ctx, sql, args...), true)
	if err != nil {
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return nil, err
	}

	return createdManagedIdentity, nil
}

// UpdateManagedIdentity updates an existing managedIdentity by ID.
// It updates the description, the data, and the group ID (to move a managed identity to another group).
func (m *managedIdentities) UpdateManagedIdentity(ctx context.Context,
	managedIdentity *models.ManagedIdentity) (*models.ManagedIdentity, error) {
	ctx, span := tracer.Start(ctx, "db.UpdateManagedIdentity")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	timestamp := currentTime()

	sql, args, err := dialect.From(t1).
		Prepared(true).
		With("managed_identities",
			dialect.Update("managed_identities").
				Set(
					goqu.Record{
						"version":     goqu.L("? + ?", goqu.C("version"), 1),
						"updated_at":  timestamp,
						"description": managedIdentity.Description,
						"data":        managedIdentity.Data,
						"group_id":    managedIdentity.GroupID,
					},
				).Where(goqu.Ex{"id": managedIdentity.Metadata.ID, "version": managedIdentity.Metadata.Version}).
				Returning("*"),
		).Select(m.getManagedIdentitySelectFields(false)...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"t1.group_id": goqu.I("namespaces.group_id")})).
		ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	updatedManagedIdentity, err := scanManagedIdentity(m.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...), false)
	if err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return nil, ErrOptimisticLockError
		}
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	return updatedManagedIdentity, nil
}

func (m *managedIdentities) DeleteManagedIdentity(ctx context.Context, managedIdentity *models.ManagedIdentity) error {
	ctx, span := tracer.Start(ctx, "db.DeleteManagedIdentity")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	sql, args, err := dialect.From(t1).
		Prepared(true).
		With("managed_identities",
			dialect.Delete("managed_identities").
				Where(
					goqu.Ex{
						"id":      managedIdentity.Metadata.ID,
						"version": managedIdentity.Metadata.Version,
					},
				).Returning("*"),
		).Select(m.getManagedIdentitySelectFields(false)...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"t1.group_id": goqu.I("namespaces.group_id")})).
		ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return err
	}

	if _, err := scanManagedIdentity(m.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...), false); err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return ErrOptimisticLockError
		}

		if pgErr := asPgError(err); pgErr != nil {
			if isForeignKeyViolation(pgErr) {
				tracing.RecordError(span, nil, "managed identity is still assigned to a workspace")
				return errors.New("managed identity is still assigned to a workspace", errors.WithErrorCode(errors.EConflict))
			}
		}

		tracing.RecordError(span, err, "failed to execute query")
		return err
	}

	return nil
}

func (m *managedIdentities) getManagedIdentitySelectFields(withAliasFields bool) []interface{} {
	selectFields := []interface{}{}
	for _, field := range managedIdentityFieldList {
		selectFields = append(selectFields, fmt.Sprintf("t1.%s", field))
	}

	selectFields = append(selectFields, "namespaces.path")

	if withAliasFields {
		selectFields = append(selectFields, "t2.description", "t2.type", "t2.data")
	}

	return selectFields
}

func (m *managedIdentities) getManagedIdentityRuleSelectFields() []interface{} {
	selectFields := []interface{}{}
	for _, field := range managedIdentityRuleFieldList {
		selectFields = append(selectFields, fmt.Sprintf("managed_identity_rules.%s", field))
	}

	selectFields = append(selectFields, "managed_identities.name", "namespaces.path")

	return selectFields
}

func (m *managedIdentities) getManagedIdentity(ctx context.Context, ex goqu.Ex) (*models.ManagedIdentity, error) {
	ctx, span := tracer.Start(ctx, "db.getManagedIdentity")
	defer span.End()

	sql, args, err := dialect.From(t1).
		Prepared(true).
		Select(m.getManagedIdentitySelectFields(true)...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"t1.group_id": goqu.I("namespaces.group_id")})).
		LeftJoin(t2, goqu.On(goqu.Ex{"t1.alias_source_id": goqu.I("t2.id")})).
		Where(ex).
		ToSQL()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	managedIdentity, err := scanManagedIdentity(m.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...), true)
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

	return managedIdentity, nil
}

func (m *managedIdentities) getManagedIdentityAccessRule(ctx context.Context, ex goqu.Ex) (*models.ManagedIdentityAccessRule, error) {
	ctx, span := tracer.Start(ctx, "db.getManagedIdentityAccessRule")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	conn := m.dbClient.getConnection(ctx)

	sql, args, err := dialect.From("managed_identity_rules").
		Prepared(true).
		Select(m.getManagedIdentityRuleSelectFields()...).
		InnerJoin(goqu.T("managed_identities"), goqu.On(goqu.Ex{"managed_identity_rules.managed_identity_id": goqu.I("managed_identities.id")})).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"managed_identities.group_id": goqu.I("namespaces.group_id")})).
		Where(ex).ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	rule, err := scanManagedIdentityRule(conn.QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}

		if pgErr := asPgError(err); pgErr != nil {
			if isInvalidIDViolation(pgErr) {
				return nil, ErrInvalidID
			}
		}

		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	if rule == nil {
		// Short circuit if rule isn't found
		return nil, nil
	}

	allowedUserIDs, err := m.getManagedIdentityAccessRuleAllowedUserIDs(ctx, conn, rule.Metadata.ID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get managed identity access rule allowed user IDs")
		return nil, err
	}

	allowedServiceAccountIDs, err := m.getManagedIdentityAccessRuleAllowedServiceAccountIDs(ctx, conn, rule.Metadata.ID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get managed identity access rule allowed service account IDs")
		return nil, err
	}

	allowedTeamIDs, err := m.getManagedIdentityAccessRuleAllowedTeamIDs(ctx, conn, rule.Metadata.ID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get managed identity access rule allowed team IDs")
		return nil, err
	}

	rule.AllowedUserIDs = allowedUserIDs
	rule.AllowedServiceAccountIDs = allowedServiceAccountIDs
	rule.AllowedTeamIDs = allowedTeamIDs

	return rule, nil
}

func (m *managedIdentities) getManagedIdentityAccessRuleAllowedUserIDs(ctx context.Context, conn connection, ruleID string) ([]string, error) {
	sql, args, err := dialect.From("managed_identity_rule_allowed_users").
		Prepared(true).
		Select("user_id").
		Where(goqu.Ex{"rule_id": ruleID}).ToSQL()
	if err != nil {
		return nil, err
	}

	rows, err := conn.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	// Scan rows
	results := []string{}
	for rows.Next() {
		var userID string
		err := rows.Scan(&userID)
		if err != nil {
			return nil, err
		}

		results = append(results, userID)
	}

	return results, nil
}

func (m *managedIdentities) getManagedIdentityAccessRuleAllowedServiceAccountIDs(ctx context.Context, conn connection, ruleID string) ([]string, error) {
	sql, args, err := dialect.From("managed_identity_rule_allowed_service_accounts").
		Prepared(true).
		Select("service_account_id").
		Where(goqu.Ex{"rule_id": ruleID}).ToSQL()
	if err != nil {
		return nil, err
	}

	rows, err := conn.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	// Scan rows
	results := []string{}
	for rows.Next() {
		var serviceAccountID string
		err := rows.Scan(&serviceAccountID)
		if err != nil {
			return nil, err
		}

		results = append(results, serviceAccountID)
	}

	return results, nil
}

func (m *managedIdentities) getManagedIdentityAccessRuleAllowedTeamIDs(ctx context.Context, conn connection, ruleID string) ([]string, error) {
	sql, args, err := dialect.From("managed_identity_rule_allowed_teams").
		Prepared(true).
		Select("team_id").
		Where(goqu.Ex{"rule_id": ruleID}).ToSQL()
	if err != nil {
		return nil, err
	}

	rows, err := conn.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	// Scan rows
	results := []string{}
	for rows.Next() {
		var teamID string
		err := rows.Scan(&teamID)
		if err != nil {
			return nil, err
		}

		results = append(results, teamID)
	}

	return results, nil
}

func scanManagedIdentity(row scanner, withAliasFields bool) (*models.ManagedIdentity, error) {
	var (
		aliasSourceDescription sql.NullString
		aliasSourceType        sql.NullString
		aliasSourceData        sql.NullString
		namespacePath          string
	)

	managedIdentity := &models.ManagedIdentity{}

	fields := []interface{}{
		&managedIdentity.Metadata.ID,
		&managedIdentity.Metadata.CreationTimestamp,
		&managedIdentity.Metadata.LastUpdatedTimestamp,
		&managedIdentity.Metadata.Version,
		&managedIdentity.Name,
		&managedIdentity.Description,
		&managedIdentity.Type,
		&managedIdentity.GroupID,
		&managedIdentity.Data,
		&managedIdentity.CreatedBy,
		&managedIdentity.AliasSourceID,
		&namespacePath,
	}

	if withAliasFields {
		fields = append(fields, &aliasSourceDescription)
		fields = append(fields, &aliasSourceType)
		fields = append(fields, &aliasSourceData)
	}

	err := row.Scan(fields...)
	if err != nil {
		return nil, err
	}

	if aliasSourceDescription.Valid {
		managedIdentity.Description = aliasSourceDescription.String
	}

	if aliasSourceType.Valid {
		managedIdentity.Type = models.ManagedIdentityType(aliasSourceType.String)
	}

	if aliasSourceData.Valid {
		managedIdentity.Data = []byte(aliasSourceData.String)
	}

	managedIdentity.Metadata.TRN = types.ManagedIdentityModelType.BuildTRN(namespacePath, managedIdentity.Name)

	return managedIdentity, nil
}

func scanManagedIdentityRule(row scanner) (*models.ManagedIdentityAccessRule, error) {
	var groupPath, managedIdentityName string
	rule := &models.ManagedIdentityAccessRule{}

	fields := []interface{}{
		&rule.Metadata.ID,
		&rule.Metadata.CreationTimestamp,
		&rule.Metadata.LastUpdatedTimestamp,
		&rule.Metadata.Version,
		&rule.RunStage,
		&rule.ManagedIdentityID,
		&rule.Type,
		&rule.ModuleAttestationPolicies,
		&rule.VerifyStateLineage,
		&managedIdentityName,
		&groupPath,
	}

	err := row.Scan(fields...)
	if err != nil {
		return nil, err
	}

	rule.Metadata.TRN = types.ManagedIdentityAccessRuleModelType.BuildTRN(groupPath, managedIdentityName, rule.GetGlobalID())

	return rule, nil
}
