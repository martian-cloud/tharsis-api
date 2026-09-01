package db

//go:generate go tool mockery --name CleanupPolicies --inpackage --case underscore

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v5"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
)

// CleanupPolicies encapsulates the logic to access cleanup policies from the database
type CleanupPolicies interface {
	GetCleanupPolicyByID(ctx context.Context, id string) (*models.CleanupPolicy, error)
	GetCleanupPolicyByTRN(ctx context.Context, trnValue string) (*models.CleanupPolicy, error)
	GetCleanupPolicies(ctx context.Context, input *GetCleanupPoliciesInput) ([]*models.CleanupPolicy, error)
	CreateCleanupPolicy(ctx context.Context, policy *models.CleanupPolicy) (*models.CleanupPolicy, error)
	UpdateCleanupPolicy(ctx context.Context, policy *models.CleanupPolicy) (*models.CleanupPolicy, error)
	DeleteCleanupPolicy(ctx context.Context, policy *models.CleanupPolicy) error
	ClaimCleanupPoliciesForSweep(ctx context.Context, input *ClaimCleanupPoliciesForSweepInput) ([]models.CleanupPolicy, error)
}

// GetCleanupPoliciesInput is the input for listing cleanup policies
type GetCleanupPoliciesInput struct {
	// NamespacePaths filters to policies belonging to any of these namespace paths.
	NamespacePaths []string
	// CleanupPolicyIDs filters to specific policy IDs.
	CleanupPolicyIDs []string
	// Kind filters to policies of this kind.
	Kind *models.CleanupRuleKind
}

// ClaimCleanupPoliciesForSweepInput is the input for claiming cleanup policies for sweeping
type ClaimCleanupPoliciesForSweepInput struct {
	// ClaimedBefore claims a policy another sweeper already claimed if that claim is older than this
	// time, so a sweeper that died mid-sweep does not hold its policies until they are due again.
	ClaimedBefore time.Time
	// Number of policies to return.
	Limit int32
}

var cleanupPolicyFieldList = append(metadataFieldList,
	"group_id",
	"workspace_id",
	"disabled",
	"kind",
	"kind_data",
	"sweep_claimed_at",
	"sweep_cursor",
	"last_sweep_completed_at",
)

type cleanupPolicies struct {
	dbClient *Client
}

// NewCleanupPolicies returns an instance of the CleanupPolicies interface
func NewCleanupPolicies(dbClient *Client) CleanupPolicies {
	return &cleanupPolicies{dbClient: dbClient}
}

func (n *cleanupPolicies) GetCleanupPolicyByID(ctx context.Context, id string) (*models.CleanupPolicy, error) {
	ctx, span := tracer.Start(ctx, "db.GetCleanupPolicyByID")
	defer span.End()

	return n.getCleanupPolicy(ctx, goqu.Ex{"cleanup_policies.id": id})
}

func (n *cleanupPolicies) GetCleanupPolicyByTRN(ctx context.Context, trnValue string) (*models.CleanupPolicy, error) {
	ctx, span := tracer.Start(ctx, "db.GetCleanupPolicyByTRN")
	defer span.End()

	parsed, err := trn.TypeCleanupPolicy.Parse(trnValue)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse TRN", errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
	}

	if !parsed.HasParent() {
		return nil, errors.New("a cleanup policy TRN must have a namespace path and kind separated by a forward slash",
			errors.WithErrorCode(errors.EInvalid),
			errors.WithSpan(span),
		)
	}

	return n.getCleanupPolicy(ctx, goqu.Ex{
		"cleanup_policies.kind": parsed.BaseName(),
		"namespaces.path":       parsed.ParentPath(),
	})
}

func (n *cleanupPolicies) GetCleanupPolicies(ctx context.Context, input *GetCleanupPoliciesInput) ([]*models.CleanupPolicy, error) {
	ctx, span := tracer.Start(ctx, "db.GetCleanupPolicies")
	defer span.End()

	ex := goqu.And()

	if len(input.NamespacePaths) > 0 {
		ex = ex.Append(goqu.I("namespaces.path").In(input.NamespacePaths))
	}

	if len(input.CleanupPolicyIDs) > 0 {
		ex = ex.Append(goqu.I("cleanup_policies.id").In(input.CleanupPolicyIDs))
	}

	if input.Kind != nil {
		ex = ex.Append(goqu.I("cleanup_policies.kind").Eq(*input.Kind))
	}

	sqlStr, args, err := toSQLWithTag("cleanupPolicy.GetCleanupPolicies",
		dialect.From("cleanup_policies").
			Prepared(true).
			InnerJoin(goqu.T("namespaces"), goqu.On(
				goqu.Or(
					goqu.Ex{"cleanup_policies.group_id": goqu.I("namespaces.group_id")},
					goqu.Ex{"cleanup_policies.workspace_id": goqu.I("namespaces.workspace_id")},
				)),
			).
			Select(n.getSelectFields()...).
			Where(ex))
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	rows, err := n.dbClient.getConnection(ctx).Query(ctx, sqlStr, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}
	defer rows.Close()

	results := []*models.CleanupPolicy{}
	for rows.Next() {
		item, err := scanCleanupPolicy(rows)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan row", errors.WithSpan(span))
		}

		results = append(results, item)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "failed to read rows", errors.WithSpan(span))
	}

	return results, nil
}

func (n *cleanupPolicies) CreateCleanupPolicy(ctx context.Context, policy *models.CleanupPolicy) (*models.CleanupPolicy, error) {
	ctx, span := tracer.Start(ctx, "db.CreateCleanupPolicy")
	defer span.End()

	data, err := marshalCleanupPolicyData(policy)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal cleanup policy rules", errors.WithSpan(span))
	}

	timestamp := currentTime()

	sql, args, err := toSQLWithTag("cleanupPolicy.CreateCleanupPolicy", dialect.From("cleanup_policies").
		Prepared(true).
		With("cleanup_policies",
			dialect.Insert("cleanup_policies").
				Rows(goqu.Record{
					"id":                      newResourceID(),
					"version":                 initialResourceVersion,
					"created_at":              timestamp,
					"updated_at":              timestamp,
					"group_id":                policy.GroupID,
					"workspace_id":            policy.WorkspaceID,
					"disabled":                policy.Disabled,
					"kind":                    policy.Kind,
					"kind_data":               data,
					"sweep_claimed_at":        policy.SweepClaimedAt,
					"sweep_cursor":            policy.SweepCursor,
					"last_sweep_completed_at": policy.LastSweepCompletedAt,
				}).
				Returning("*"),
		).Select(n.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Or(
			goqu.Ex{"cleanup_policies.group_id": goqu.I("namespaces.group_id")},
			goqu.Ex{"cleanup_policies.workspace_id": goqu.I("namespaces.workspace_id")},
		))))
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	createdPolicy, err := scanCleanupPolicy(n.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				return nil, errors.New(
					"a cleanup policy of this kind already exists for this namespace",
					errors.WithErrorCode(errors.EConflict),
					errors.WithSpan(span),
				)
			}

			if isForeignKeyViolation(pgErr) {
				return nil, errors.New("namespace does not exist", errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
			}
		}

		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return createdPolicy, nil
}

func (n *cleanupPolicies) UpdateCleanupPolicy(ctx context.Context, policy *models.CleanupPolicy) (*models.CleanupPolicy, error) {
	ctx, span := tracer.Start(ctx, "db.UpdateCleanupPolicy")
	defer span.End()

	data, err := marshalCleanupPolicyData(policy)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal cleanup policy rules", errors.WithSpan(span))
	}

	timestamp := currentTime()

	sql, args, err := toSQLWithTag("cleanupPolicy.UpdateCleanupPolicy", dialect.From("cleanup_policies").
		Prepared(true).
		With("cleanup_policies",
			dialect.Update("cleanup_policies").
				Set(goqu.Record{
					"version":                 goqu.L("? + ?", goqu.C("version"), 1),
					"updated_at":              timestamp,
					"disabled":                policy.Disabled,
					"kind_data":               data,
					"sweep_claimed_at":        policy.SweepClaimedAt,
					"sweep_cursor":            policy.SweepCursor,
					"last_sweep_completed_at": policy.LastSweepCompletedAt,
				}).
				Where(goqu.Ex{"id": policy.Metadata.ID, "version": policy.Metadata.Version}).
				Returning("*"),
		).Select(n.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(
			goqu.Or(
				goqu.Ex{"cleanup_policies.group_id": goqu.I("namespaces.group_id")},
				goqu.Ex{"cleanup_policies.workspace_id": goqu.I("namespaces.workspace_id")},
			)),
		))
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	updatedPolicy, err := scanCleanupPolicy(n.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrOptimisticLockError
		}

		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return updatedPolicy, nil
}

func (n *cleanupPolicies) DeleteCleanupPolicy(ctx context.Context, policy *models.CleanupPolicy) error {
	ctx, span := tracer.Start(ctx, "db.DeleteCleanupPolicy")
	defer span.End()

	sql, args, err := toSQLWithTag("cleanupPolicy.DeleteCleanupPolicy", dialect.From("cleanup_policies").
		Prepared(true).
		With("cleanup_policies",
			dialect.Delete("cleanup_policies").
				Where(goqu.Ex{"id": policy.Metadata.ID, "version": policy.Metadata.Version}).
				Returning("*"),
		).Select(n.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(
			goqu.Or(
				goqu.Ex{"cleanup_policies.group_id": goqu.I("namespaces.group_id")},
				goqu.Ex{"cleanup_policies.workspace_id": goqu.I("namespaces.workspace_id")},
			)),
		))
	if err != nil {
		return errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	if _, err = scanCleanupPolicy(n.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...)); err != nil {
		if err == pgx.ErrNoRows {
			return ErrOptimisticLockError
		}

		return errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return nil
}

// ClaimCleanupPoliciesForSweep claims and returns up to Limit policies that are due, marking
// each as being swept so that a sweeper on another instance claims a different batch.
func (n *cleanupPolicies) ClaimCleanupPoliciesForSweep(ctx context.Context, input *ClaimCleanupPoliciesForSweepInput) ([]models.CleanupPolicy, error) {
	ctx, span := tracer.Start(ctx, "db.ClaimCleanupPoliciesForSweep")
	defer span.End()

	timestamp := currentTime()

	claimedBefore := input.ClaimedBefore.UTC()

	// Inner CTE: pick a batch of due policies, row-locked with SKIP LOCKED so a concurrent sweeper
	// selects a different batch rather than blocking.
	claimable := dialect.From(goqu.T("cleanup_policies")).
		Select(goqu.I("cleanup_policies.id")).
		Where(
			// A policy is due once it's never been swept or its claim is old enough to have expired.
			goqu.Or(
				goqu.I("cleanup_policies.sweep_claimed_at").IsNull(),
				goqu.I("cleanup_policies.sweep_claimed_at").Lt(claimedBefore),
			),
			// Skip disabled policies.
			goqu.I("cleanup_policies.disabled").IsFalse(),
		).
		Order(goqu.I("cleanup_policies.sweep_claimed_at").Asc().NullsFirst()).
		Limit(uint(input.Limit)).
		ForUpdate(goqu.SkipLocked, goqu.T("cleanup_policies"))

	// Outer UPDATE: stamp the claim on exactly the locked rows and return them, which also keeps another
	// sweeper off a policy while this one works through it. The version is left alone on purpose, since a
	// sweep is bookkeeping and must not invalidate a caller's open edit.
	query := dialect.Update("cleanup_policies").
		Prepared(true).
		With("claimable", claimable).
		Set(goqu.Record{"sweep_claimed_at": timestamp}).
		From("claimable", "namespaces").
		Where(
			goqu.I("cleanup_policies.id").Eq(goqu.I("claimable.id")),
			goqu.Or(
				goqu.Ex{"cleanup_policies.group_id": goqu.I("namespaces.group_id")},
				goqu.Ex{"cleanup_policies.workspace_id": goqu.I("namespaces.workspace_id")},
			),
		).
		Returning(n.getSelectFields()...)

	sql, args, err := toSQLWithTag("cleanupPolicy.ClaimCleanupPoliciesForSweep", query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	rows, err := n.dbClient.getConnection(ctx).Query(ctx, sql, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}
	defer rows.Close()

	results := []models.CleanupPolicy{}
	for rows.Next() {
		item, err := scanCleanupPolicy(rows)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan row", errors.WithSpan(span))
		}

		results = append(results, *item)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "failed to read rows", errors.WithSpan(span))
	}

	return results, nil
}

func (n *cleanupPolicies) getCleanupPolicy(ctx context.Context, ex goqu.Ex) (*models.CleanupPolicy, error) {
	ctx, span := tracer.Start(ctx, "db.getCleanupPolicy")
	defer span.End()

	sql, args, err := toSQLWithTag("cleanupPolicy.getCleanupPolicy",
		dialect.From("cleanup_policies").
			Prepared(true).
			InnerJoin(goqu.T("namespaces"), goqu.On(
				goqu.Or(
					goqu.Ex{"cleanup_policies.group_id": goqu.I("namespaces.group_id")},
					goqu.Ex{"cleanup_policies.workspace_id": goqu.I("namespaces.workspace_id")},
				)),
			).
			Select(n.getSelectFields()...).
			Where(ex))
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL for query to get cleanup policy", errors.WithSpan(span))
	}

	policy, err := scanCleanupPolicy(n.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}

		if pgErr := asPgError(err); pgErr != nil {
			if isInvalidIDViolation(pgErr) {
				return nil, ErrInvalidID
			}
		}

		return nil, errors.Wrap(err, "failed execute query to get cleanup policy", errors.WithSpan(span))
	}

	return policy, nil
}

func (n *cleanupPolicies) getSelectFields() []interface{} {
	selectFields := []any{}

	for _, field := range cleanupPolicyFieldList {
		selectFields = append(selectFields, fmt.Sprintf("cleanup_policies.%s", field))
	}

	selectFields = append(selectFields, "namespaces.path")

	return selectFields
}

// marshalCleanupPolicyData marshals whichever data struct is set for policy.Kind.
func marshalCleanupPolicyData(policy *models.CleanupPolicy) ([]byte, error) {
	switch policy.Kind {
	case models.CleanupRuleKindTerraformModules:
		return json.Marshal(policy.TerraformModulePolicyData)
	case models.CleanupRuleKindTerraformProviders:
		return json.Marshal(policy.TerraformProviderPolicyData)
	case models.CleanupRuleKindRuns:
		return json.Marshal(policy.RunPolicyData)
	default:
		return nil, errors.New("unsupported kind %q", policy.Kind, errors.WithErrorCode(errors.EInvalid))
	}
}

func scanCleanupPolicy(row scanner) (*models.CleanupPolicy, error) {
	var data []byte
	var namespacePath string

	policy := &models.CleanupPolicy{}

	err := row.Scan(
		&policy.Metadata.ID,
		&policy.Metadata.CreationTimestamp,
		&policy.Metadata.LastUpdatedTimestamp,
		&policy.Metadata.Version,
		&policy.GroupID,
		&policy.WorkspaceID,
		&policy.Disabled,
		&policy.Kind,
		&data,
		&policy.SweepClaimedAt,
		&policy.SweepCursor,
		&policy.LastSweepCompletedAt,
		&namespacePath,
	)
	if err != nil {
		return nil, err
	}

	switch policy.Kind {
	case models.CleanupRuleKindTerraformModules:
		policy.TerraformModulePolicyData = &models.TerraformModuleCleanupPolicyData{}
		if err := json.Unmarshal(data, policy.TerraformModulePolicyData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal terraform module cleanup policy data: %w", err)
		}
	case models.CleanupRuleKindTerraformProviders:
		policy.TerraformProviderPolicyData = &models.TerraformProviderCleanupPolicyData{}
		if err := json.Unmarshal(data, policy.TerraformProviderPolicyData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal terraform provider cleanup policy data: %w", err)
		}
	case models.CleanupRuleKindRuns:
		policy.RunPolicyData = &models.RunCleanupPolicyData{}
		if err := json.Unmarshal(data, policy.RunPolicyData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal run cleanup policy data: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported kind %q in database", policy.Kind)
	}

	policy.Metadata.TRN = trn.TypeCleanupPolicy.Build(namespacePath, string(policy.Kind))

	return policy, nil
}
