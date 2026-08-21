package db

//go:generate go tool mockery --name Policies --inpackage --case underscore

import (
	"context"
	"encoding/json"
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

// Policies encapsulates the logic to access policies from the database
type Policies interface {
	GetPolicyByID(ctx context.Context, id string) (*models.Policy, error)
	GetPolicyByTRN(ctx context.Context, trnValue string) (*models.Policy, error)
	GetPolicies(ctx context.Context, input *GetPoliciesInput) (*PoliciesResult, error)
	CreatePolicy(ctx context.Context, policy *models.Policy) (*models.Policy, error)
	UpdatePolicy(ctx context.Context, policy *models.Policy) (*models.Policy, error)
	DeletePolicy(ctx context.Context, policy *models.Policy) error
}

// PolicySortableField represents the fields a policy connection can be sorted by.
type PolicySortableField string

// PolicySortableField constants.
const (
	PolicySortableFieldCreatedAtAsc  PolicySortableField = "CREATED_AT_ASC"
	PolicySortableFieldCreatedAtDesc PolicySortableField = "CREATED_AT_DESC"
)

func (sf PolicySortableField) getFieldDescriptor() *pagination.FieldDescriptor {
	switch sf {
	case PolicySortableFieldCreatedAtAsc, PolicySortableFieldCreatedAtDesc:
		return &pagination.FieldDescriptor{Key: "created_at", Table: "policies", Col: "created_at"}
	default:
		return nil
	}
}

func (sf PolicySortableField) getSortDirection() pagination.SortDirection {
	if strings.HasSuffix(string(sf), "_DESC") {
		return pagination.DescSort
	}
	return pagination.AscSort
}

// PolicyFilter contains the supported fields for filtering Policy resources.
type PolicyFilter struct {
	// GroupIDs filters to policies owned by any of these groups.
	GroupIDs []string
	// GroupPathTree filters to policies owned by the group at this path, by one of its ancestors, or by
	// any group beneath it
	GroupPathTree *string
	// PolicyIDs filters to policies with any of these IDs (used for batch loading).
	PolicyIDs []string
	Stage     *models.RunTaskStageName
}

// GetPoliciesInput is the input for listing policies
type GetPoliciesInput struct {
	Sort              *PolicySortableField
	PaginationOptions *pagination.Options
	Filter            *PolicyFilter
}

// PoliciesResult contains the response data and page information
type PoliciesResult struct {
	PageInfo *pagination.PageInfo
	Policies []*models.Policy
}

type policies struct {
	dbClient *Client
}

var policyFieldList = append(metadataFieldList,
	"group_id",
	"name",
	"description",
	"kind",
	"kind_data",
	"scope",
	"required_approvals",
	"created_by",
)

// NewPolicies returns an instance of the Policies interface
func NewPolicies(dbClient *Client) Policies {
	return &policies{dbClient: dbClient}
}

func (m *policies) GetPolicyByID(ctx context.Context, id string) (*models.Policy, error) {
	ctx, span := tracer.Start(ctx, "db.GetPolicyByID")
	defer span.End()

	return m.getPolicy(ctx, goqu.Ex{"policies.id": id})
}

func (m *policies) GetPolicyByTRN(ctx context.Context, trnValue string) (*models.Policy, error) {
	ctx, span := tracer.Start(ctx, "db.GetPolicyByTRN")
	defer span.End()

	parsed, err := trn.TypePolicy.Parse(trnValue)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse TRN", errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
	}

	if !parsed.HasParent() {
		return nil, errors.New("a policy TRN must have a group path and policy name separated by a forward slash",
			errors.WithErrorCode(errors.EInvalid),
			errors.WithSpan(span),
		)
	}

	return m.getPolicy(ctx, goqu.Ex{
		"policies.name":   parsed.BaseName(),
		"namespaces.path": parsed.ParentPath(),
	})
}

func (m *policies) GetPolicies(ctx context.Context, input *GetPoliciesInput) (*PoliciesResult, error) {
	ctx, span := tracer.Start(ctx, "db.GetPolicies")
	defer span.End()

	ex := goqu.And()

	if input.Filter != nil {
		if len(input.Filter.GroupIDs) > 0 {
			ex = ex.Append(goqu.I("policies.group_id").In(input.Filter.GroupIDs))
		}
		if len(input.Filter.PolicyIDs) > 0 {
			ex = ex.Append(goqu.I("policies.id").In(input.Filter.PolicyIDs))
		}
		if input.Filter.GroupPathTree != nil {
			// ExpandPath covers the path itself and every ancestor; the prefix covers everything below
			// it. The path is escaped because "_" is a LIKE wildcard and group names may contain it.
			ex = ex.Append(goqu.Or(
				goqu.I("namespaces.path").In(nsutils.ExpandPath(*input.Filter.GroupPathTree)),
				goqu.I("namespaces.path").Like(escapeLikePattern(*input.Filter.GroupPathTree)+"/%"),
			))
		}
		if input.Filter.Stage != nil {
			ex = ex.Append(goqu.L("policies.kind_data->>'stage'").Eq(string(*input.Filter.Stage)))
		}
	}

	query := dialect.From("policies").
		Select(m.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"policies.group_id": goqu.I("namespaces.group_id")})).
		Where(ex)

	sortDirection := pagination.AscSort
	var sortBy *pagination.FieldDescriptor
	if input.Sort != nil {
		sortDirection = input.Sort.getSortDirection()
		sortBy = input.Sort.getFieldDescriptor()
	}

	qBuilder, err := pagination.NewPaginatedQueryBuilder(
		input.PaginationOptions,
		&pagination.FieldDescriptor{Key: "id", Table: "policies", Col: "id"},
		pagination.WithSortByField(sortBy, sortDirection),
		pagination.WithQueryTag("policy.GetPolicies"),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build query", errors.WithSpan(span))
	}

	rows, err := qBuilder.Execute(ctx, m.dbClient.getConnection(ctx), query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	results := []*models.Policy{}
	for rows.Next() {
		item, err := scanPolicy(rows)
		if err != nil {
			rows.Close()
			return nil, errors.Wrap(err, "failed to scan row", errors.WithSpan(span))
		}
		results = append(results, item)
	}

	if err := rows.Finalize(&results); err != nil {
		rows.Close()
		return nil, errors.Wrap(err, "failed to finalize rows", errors.WithSpan(span))
	}

	pageInfo := rows.GetPageInfo()

	// Close the row set before loading approver principals: a new query cannot run while rows are
	// open on the same connection.
	rows.Close()

	if err := m.loadAllowedPrincipalsForPolicies(ctx, results); err != nil {
		return nil, errors.Wrap(err, "failed to load policy approvers", errors.WithSpan(span))
	}

	return &PoliciesResult{
		PageInfo: pageInfo,
		Policies: results,
	}, nil
}

func (m *policies) CreatePolicy(ctx context.Context, policy *models.Policy) (*models.Policy, error) {
	ctx, span := tracer.Start(ctx, "db.CreatePolicy")
	defer span.End()

	timestamp := currentTime()

	scopeJSON, err := json.Marshal(policy.Scope)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal policy scope", errors.WithSpan(span))
	}

	kindDataJSON, err := json.Marshal(policy.OPAData)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal policy kind_data", errors.WithSpan(span))
	}

	tx, err := m.dbClient.getConnection(ctx).Begin(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin DB transaction", errors.WithSpan(span))
	}
	defer func() {
		if txErr := tx.Rollback(ctx); txErr != nil && txErr != pgx.ErrTxClosed {
			m.dbClient.logger.WithContextFields(ctx).Errorf("failed to rollback tx for CreatePolicy: %v", txErr)
		}
	}()

	sql, args, err := toSQLWithTag("policy.CreatePolicy", dialect.From("policies").
		Prepared(true).
		With("policies",
			dialect.Insert("policies").
				Rows(goqu.Record{
					"id":                 newResourceID(),
					"version":            initialResourceVersion,
					"created_at":         timestamp,
					"updated_at":         timestamp,
					"group_id":           policy.GroupID,
					"name":               policy.Name,
					"description":        policy.Description,
					"kind":               policy.Kind,
					"kind_data":          kindDataJSON,
					"scope":              scopeJSON,
					"required_approvals": policy.RequiredApprovals,
					"created_by":         policy.CreatedBy,
				}).Returning("*"),
		).Select(m.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"policies.group_id": goqu.I("namespaces.group_id")})))
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	created, err := scanPolicy(tx.QueryRow(ctx, sql, args...))
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				return nil, errors.New("a policy with this name already exists in the group",
					errors.WithErrorCode(errors.EConflict), errors.WithSpan(span))
			}
			if isForeignKeyViolation(pgErr) {
				return nil, errors.New("owner group does not exist",
					errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
			}
		}
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	if err := m.insertAllowedPrincipals(ctx, tx, created.Metadata.ID, policy); err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isForeignKeyViolation(pgErr) {
				return nil, errors.New("an allowed approver (user, service account, or team) does not exist",
					errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
			}
			if isUniqueViolation(pgErr) {
				return nil, errors.New("an approver is listed more than once",
					errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
			}
		}
		return nil, errors.Wrap(err, "failed to insert policy approvers", errors.WithSpan(span))
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, errors.Wrap(err, "failed to commit DB transaction", errors.WithSpan(span))
	}

	// The approver ids just inserted are the caller-supplied ones, which Validate has already checked
	// hold no duplicates.
	created.AllowedUserIDs = policy.AllowedUserIDs
	created.AllowedServiceAccountIDs = policy.AllowedServiceAccountIDs
	created.AllowedTeamIDs = policy.AllowedTeamIDs

	return created, nil
}

func (m *policies) UpdatePolicy(ctx context.Context, policy *models.Policy) (*models.Policy, error) {
	ctx, span := tracer.Start(ctx, "db.UpdatePolicy")
	defer span.End()

	timestamp := currentTime()

	scopeJSON, err := json.Marshal(policy.Scope)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal policy scope", errors.WithSpan(span))
	}

	kindDataJSON, err := json.Marshal(policy.OPAData)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal policy kind_data", errors.WithSpan(span))
	}

	tx, err := m.dbClient.getConnection(ctx).Begin(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin DB transaction", errors.WithSpan(span))
	}
	defer func() {
		if txErr := tx.Rollback(ctx); txErr != nil && txErr != pgx.ErrTxClosed {
			m.dbClient.logger.WithContextFields(ctx).Errorf("failed to rollback tx for UpdatePolicy: %v", txErr)
		}
	}()

	sql, args, err := toSQLWithTag("policy.UpdatePolicy", dialect.From("policies").
		Prepared(true).
		With("policies",
			dialect.Update("policies").
				Set(goqu.Record{
					"version":            goqu.L("version + 1"),
					"updated_at":         timestamp,
					"description":        policy.Description,
					"kind_data":          kindDataJSON,
					"scope":              scopeJSON,
					"required_approvals": policy.RequiredApprovals,
				}).
				Where(goqu.Ex{"id": policy.Metadata.ID, "version": policy.Metadata.Version}).
				Returning("*"),
		).Select(m.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"policies.group_id": goqu.I("namespaces.group_id")})))
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	updated, err := scanPolicy(tx.QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrOptimisticLockError
		}
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	// Replace child approver rows: delete all then reinsert.
	for _, table := range []string{"policy_allowed_users", "policy_allowed_service_accounts", "policy_allowed_teams"} {
		delSQL, delArgs, dErr := toSQLWithTag("policy.UpdatePolicy.deleteApprovers",
			dialect.Delete(table).Prepared(true).Where(goqu.Ex{"policy_id": updated.Metadata.ID}))
		if dErr != nil {
			return nil, errors.Wrap(dErr, "failed to generate SQL", errors.WithSpan(span))
		}
		if _, dErr = tx.Exec(ctx, delSQL, delArgs...); dErr != nil {
			return nil, errors.Wrap(dErr, "failed to delete policy approvers", errors.WithSpan(span))
		}
	}

	if err := m.insertAllowedPrincipals(ctx, tx, updated.Metadata.ID, policy); err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isForeignKeyViolation(pgErr) {
				return nil, errors.New("an allowed approver (user, service account, or team) does not exist",
					errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
			}
			if isUniqueViolation(pgErr) {
				return nil, errors.New("an approver is listed more than once",
					errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
			}
		}
		return nil, errors.Wrap(err, "failed to insert policy approvers", errors.WithSpan(span))
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, errors.Wrap(err, "failed to commit DB transaction", errors.WithSpan(span))
	}

	updated.AllowedUserIDs = policy.AllowedUserIDs
	updated.AllowedServiceAccountIDs = policy.AllowedServiceAccountIDs
	updated.AllowedTeamIDs = policy.AllowedTeamIDs

	return updated, nil
}

func (m *policies) DeletePolicy(ctx context.Context, policy *models.Policy) error {
	ctx, span := tracer.Start(ctx, "db.DeletePolicy")
	defer span.End()

	sql, args, err := toSQLWithTag("policy.DeletePolicy", dialect.From("policies").
		Prepared(true).
		With("policies",
			dialect.Delete("policies").
				Where(goqu.Ex{"id": policy.Metadata.ID, "version": policy.Metadata.Version}).
				Returning("*"),
		).Select(m.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"policies.group_id": goqu.I("namespaces.group_id")})))
	if err != nil {
		return errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	if _, err := scanPolicy(m.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...)); err != nil {
		if err == pgx.ErrNoRows {
			return ErrOptimisticLockError
		}
		return errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return nil
}

func (m *policies) getPolicy(ctx context.Context, ex goqu.Ex) (*models.Policy, error) {
	sql, args, err := toSQLWithTag("policy.getPolicy", dialect.From("policies").
		Prepared(true).
		Select(m.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"policies.group_id": goqu.I("namespaces.group_id")})).
		Where(ex))
	if err != nil {
		return nil, err
	}

	policy, err := scanPolicy(m.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
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

	if err := m.loadAllowedPrincipalsForPolicies(ctx, []*models.Policy{policy}); err != nil {
		return nil, err
	}

	return policy, nil
}

func (m *policies) getSelectFields() []interface{} {
	selectFields := []interface{}{}
	for _, field := range policyFieldList {
		selectFields = append(selectFields, fmt.Sprintf("policies.%s", field))
	}
	selectFields = append(selectFields, "namespaces.path")
	return selectFields
}

// insertAllowedPrincipals writes the policy's approver principals into the three child tables. A
// repeated principal is rejected by the unique index on (policy_id, <principal>); Policy.Validate turns
// that away first, so reaching it means a caller bypassed validation.
func (m *policies) insertAllowedPrincipals(ctx context.Context, tx pgx.Tx, policyID string, policy *models.Policy) error {
	inserts := []struct {
		table string
		col   string
		ids   []string
	}{
		{"policy_allowed_users", "user_id", policy.AllowedUserIDs},
		{"policy_allowed_service_accounts", "service_account_id", policy.AllowedServiceAccountIDs},
		{"policy_allowed_teams", "team_id", policy.AllowedTeamIDs},
	}

	for _, ins := range inserts {
		if len(ins.ids) == 0 {
			continue
		}

		rows := make([]interface{}, len(ins.ids))
		for i, id := range ins.ids {
			rows[i] = goqu.Record{
				"id":        newResourceID(),
				"policy_id": policyID,
				ins.col:     id,
			}
		}

		sql, args, err := toSQLWithTag("policy.insertAllowedPrincipals", dialect.Insert(ins.table).
			Prepared(true).
			Rows(rows...))
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, sql, args...); err != nil {
			return err
		}
	}

	return nil
}

// loadAllowedPrincipalsForPolicies populates the approver id slices for every policy in three queries
// total (one per child table) rather than three per policy, which is what a per-policy loadAllowedPrincipals
// call would cost across a page of results.
func (m *policies) loadAllowedPrincipalsForPolicies(ctx context.Context, policies []*models.Policy) error {
	if len(policies) == 0 {
		return nil
	}

	ids := make([]string, len(policies))
	for i, policy := range policies {
		ids[i] = policy.Metadata.ID
	}

	usersByPolicy, err := m.scanChildIDsByPolicy(ctx, "policy_allowed_users", "user_id", ids)
	if err != nil {
		return err
	}
	serviceAccountsByPolicy, err := m.scanChildIDsByPolicy(ctx, "policy_allowed_service_accounts", "service_account_id", ids)
	if err != nil {
		return err
	}
	teamsByPolicy, err := m.scanChildIDsByPolicy(ctx, "policy_allowed_teams", "team_id", ids)
	if err != nil {
		return err
	}

	for i := range policies {
		id := policies[i].Metadata.ID
		policies[i].AllowedUserIDs = usersByPolicy[id]
		policies[i].AllowedServiceAccountIDs = serviceAccountsByPolicy[id]
		policies[i].AllowedTeamIDs = teamsByPolicy[id]
	}

	return nil
}

// scanChildIDsByPolicy reads a single id column from a policy_allowed_* child table for multiple
// policies at once, grouped by policy_id. Every requested policy id is present in the returned map,
// even with no rows, so callers don't need a presence check before indexing in.
func (m *policies) scanChildIDsByPolicy(ctx context.Context, table, col string, policyIDs []string) (map[string][]string, error) {
	result := make(map[string][]string, len(policyIDs))
	for _, id := range policyIDs {
		result[id] = []string{}
	}

	sql, args, err := toSQLWithTag("policy."+table, dialect.From(table).
		Prepared(true).
		Select("policy_id", col).
		Where(goqu.I("policy_id").In(policyIDs)))
	if err != nil {
		return nil, err
	}

	rows, err := m.dbClient.getConnection(ctx).Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var policyID, childID string
		if err := rows.Scan(&policyID, &childID); err != nil {
			return nil, err
		}
		result[policyID] = append(result[policyID], childID)
	}
	return result, rows.Err()
}

func scanPolicy(row scanner) (*models.Policy, error) {
	policy := &models.Policy{}

	var scopeJSON, kindDataJSON []byte
	var groupPath string

	fields := []interface{}{
		&policy.Metadata.ID,
		&policy.Metadata.CreationTimestamp,
		&policy.Metadata.LastUpdatedTimestamp,
		&policy.Metadata.Version,
		&policy.GroupID,
		&policy.Name,
		&policy.Description,
		&policy.Kind,
		&kindDataJSON,
		&scopeJSON,
		&policy.RequiredApprovals,
		&policy.CreatedBy,
		&groupPath,
	}

	if err := row.Scan(fields...); err != nil {
		return nil, err
	}

	if len(kindDataJSON) > 0 {
		switch policy.Kind {
		case models.PolicyKindOPA:
			policy.OPAData = &models.OPAPolicyData{}
			if err := json.Unmarshal(kindDataJSON, policy.OPAData); err != nil {
				return nil, fmt.Errorf("failed to unmarshal policy kind_data: %w", err)
			}
		}
	}

	if len(scopeJSON) > 0 {
		if err := json.Unmarshal(scopeJSON, &policy.Scope); err != nil {
			return nil, fmt.Errorf("failed to unmarshal policy scope: %w", err)
		}
	}
	if policy.Scope == nil {
		policy.Scope = []*models.ScopeRule{}
	}

	policy.Metadata.TRN = trn.TypePolicy.Build(groupPath, policy.Name)

	return policy, nil
}
