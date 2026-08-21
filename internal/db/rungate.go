package db

//go:generate go tool mockery --name RunGates --inpackage --case underscore

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/jackc/pgx/v5"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
)

// RunGates encapsulates the logic to access run gates from the database
type RunGates interface {
	GetRunGateByID(ctx context.Context, id string) (*models.RunGate, error)
	GetRunGateByTRN(ctx context.Context, trnValue string) (*models.RunGate, error)
	GetRunGates(ctx context.Context, input *GetRunGatesInput) (*RunGatesResult, error)
	CreateRunGate(ctx context.Context, gate *models.RunGate) (*models.RunGate, error)
	UpdateRunGate(ctx context.Context, gate *models.RunGate) (*models.RunGate, error)
	DeleteRunGate(ctx context.Context, gate *models.RunGate) error
}

// RunGateSortableField represents the fields a run gate connection can be sorted by.
type RunGateSortableField string

// RunGateSortableField constants.
const (
	RunGateSortableFieldCreatedAtAsc  RunGateSortableField = "CREATED_AT_ASC"
	RunGateSortableFieldCreatedAtDesc RunGateSortableField = "CREATED_AT_DESC"
)

func (sf RunGateSortableField) getFieldDescriptor() *pagination.FieldDescriptor {
	switch sf {
	case RunGateSortableFieldCreatedAtAsc, RunGateSortableFieldCreatedAtDesc:
		return &pagination.FieldDescriptor{Key: "created_at", Table: "run_gates", Col: "created_at"}
	default:
		return nil
	}
}

func (sf RunGateSortableField) getSortDirection() pagination.SortDirection {
	if strings.HasSuffix(string(sf), "_DESC") {
		return pagination.DescSort
	}
	return pagination.AscSort
}

// RunGateEligibilityFilter restricts a run gate query to gates a specific caller may act on and
// excludes gates the caller has already decided. Matching is done against the run_gate_allowed_*
// query-index tables (raw principal IDs) and the run_gate_approvals table. When UserID is set, team
// eligibility is derived from the user's team_members rows via a subquery, so the caller does not
// need to resolve and pass in the user's team memberships.
type RunGateEligibilityFilter struct {
	UserID           *string
	ServiceAccountID *string
}

// RunGateFilter contains the supported fields for filtering RunGate resources.
type RunGateFilter struct {
	RunID *string
	// RunGateIDs restricts to the gates with the given primary IDs.
	RunGateIDs []string
	// PolicyCheckIDs restricts to the gates governing specific policy-check nodes. At most one gate
	// exists per check, so this returns at most one row per ID.
	PolicyCheckIDs []string
	Statuses       []models.RunGateStatus
	// Eligibility, when set, restricts to gates the caller is an allowed subject for and that the
	// caller has not already decided.
	Eligibility *RunGateEligibilityFilter
	// RootNamespaceMemberships limits results to gates in workspaces at or under one of the
	// caller's root member namespaces. An empty (non-nil) slice matches nothing.
	RootNamespaceMemberships []models.MembershipNamespace
}

// GetRunGatesInput is the input for listing run gates.
type GetRunGatesInput struct {
	Sort              *RunGateSortableField
	PaginationOptions *pagination.Options
	Filter            *RunGateFilter
}

// RunGatesResult contains the response data and page information.
type RunGatesResult struct {
	PageInfo *pagination.PageInfo
	RunGates []models.RunGate
}

type runGates struct {
	dbClient *Client
}

var runGateFieldList = append(metadataFieldList,
	"run_id",
	"workspace_id",
	"policy_check_id",
	"type",
	"approval_rules",
	"status",
	"overridden_by",
	"override_comment",
)

// NewRunGates returns an instance of the RunGates interface
func NewRunGates(dbClient *Client) RunGates {
	return &runGates{dbClient: dbClient}
}

func (m *runGates) GetRunGateByID(ctx context.Context, id string) (*models.RunGate, error) {
	ctx, span := tracer.Start(ctx, "db.GetRunGateByID")
	defer span.End()

	return m.getRunGate(ctx, goqu.Ex{"run_gates.id": id})
}

func (m *runGates) GetRunGateByTRN(ctx context.Context, trnValue string) (*models.RunGate, error) {
	ctx, span := tracer.Start(ctx, "db.GetRunGateByTRN")
	defer span.End()

	parsed, err := trn.TypeRunGate.Parse(trnValue)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse TRN", errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
	}

	if !parsed.HasParent() {
		return nil, errors.New("a run gate TRN must have the workspace path, run GID, and gate GID separated by forward slashes",
			errors.WithErrorCode(errors.EInvalid),
			errors.WithSpan(span),
		)
	}

	// parentPath is "<workspace-path>/<run-GID>"; split on the last slash to extract each.
	parentPath := parsed.ParentPath()
	lastSlash := strings.LastIndex(parentPath, "/")
	if lastSlash < 0 {
		return nil, errors.New("a run gate TRN must have the workspace path, run GID, and gate GID separated by forward slashes",
			errors.WithErrorCode(errors.EInvalid),
			errors.WithSpan(span),
		)
	}
	workspacePath := parentPath[:lastSlash]

	return m.getRunGate(ctx, goqu.Ex{
		"run_gates.id":    gid.FromGlobalID(parsed.BaseName()),
		"namespaces.path": workspacePath,
	})
}

func (m *runGates) GetRunGates(ctx context.Context, input *GetRunGatesInput) (*RunGatesResult, error) {
	ctx, span := tracer.Start(ctx, "db.GetRunGates")
	defer span.End()

	ex := goqu.And()

	if input.Filter != nil {
		f := input.Filter

		if f.RunID != nil {
			ex = ex.Append(goqu.I("run_gates.run_id").Eq(*f.RunID))
		}

		if len(f.RunGateIDs) > 0 {
			ex = ex.Append(goqu.I("run_gates.id").In(f.RunGateIDs))
		}

		if len(f.PolicyCheckIDs) > 0 {
			ex = ex.Append(goqu.I("run_gates.policy_check_id").In(f.PolicyCheckIDs))
		}

		if len(f.Statuses) > 0 {
			statuses := make([]string, len(f.Statuses))
			for i, s := range f.Statuses {
				statuses[i] = string(s)
			}
			ex = ex.Append(goqu.I("run_gates.status").In(statuses))
		}

		if f.Eligibility != nil {
			e := f.Eligibility

			// Eligibility: the caller must be an allowed subject for the gate via at least one of
			// the query-index tables (allowed user, service account, or team).
			eligible := []exp.Expression{}
			if e.UserID != nil {
				// Directly allowed user.
				eligible = append(eligible, goqu.I("run_gates.id").In(
					dialect.From("run_gate_allowed_users").Select("gate_id").Where(goqu.Ex{"user_id": *e.UserID}),
				))
				// Allowed via any team the user belongs to. Resolve the user's team memberships with
				// a subquery on team_members rather than having the caller pass in team IDs.
				eligible = append(eligible, goqu.I("run_gates.id").In(
					dialect.From("run_gate_allowed_teams").Select("gate_id").Where(goqu.I("team_id").In(
						dialect.From("team_members").Select("team_id").Where(goqu.Ex{"user_id": *e.UserID}),
					)),
				))
			}
			if e.ServiceAccountID != nil {
				eligible = append(eligible, goqu.I("run_gates.id").In(
					dialect.From("run_gate_allowed_service_accounts").Select("gate_id").Where(goqu.Ex{"service_account_id": *e.ServiceAccountID}),
				))
			}
			if len(eligible) == 0 {
				// A caller with no matching principal is eligible for nothing. Return a literal
				// false rather than an absent predicate so this never widens the result set.
				ex = ex.Append(goqu.L("false"))
			} else {
				ex = ex.Append(goqu.Or(eligible...))
			}

			// Exclude gates the caller has already decided. Correlate on run_gate_id so the
			// anti-join probes run_gate_approvals via its run_gate_id index for the small set of
			// candidate gates, rather than scanning approvals by principal id.
			decided := []exp.Expression{}
			if e.UserID != nil {
				decided = append(decided, goqu.Ex{"user_id": *e.UserID})
			}
			if e.ServiceAccountID != nil {
				decided = append(decided, goqu.Ex{"service_account_id": *e.ServiceAccountID})
			}
			if len(decided) > 0 {
				ex = ex.Append(goqu.L("NOT EXISTS ?",
					dialect.From("run_gate_approvals").
						Select(goqu.L("1")).
						Where(
							goqu.I("run_gate_approvals.run_gate_id").Eq(goqu.I("run_gates.id")),
							goqu.Or(decided...),
						),
				))
			}
		}

		if f.RootNamespaceMemberships != nil {
			ex = ex.Append(membershipFilterByRootNamespaces(f.RootNamespaceMemberships))
		}
	}

	query := dialect.From("run_gates").
		Select(m.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"run_gates.workspace_id": goqu.I("namespaces.workspace_id")}))

	query = query.Where(ex)

	sortDirection := pagination.AscSort
	var sortBy *pagination.FieldDescriptor
	if input.Sort != nil {
		sortDirection = input.Sort.getSortDirection()
		sortBy = input.Sort.getFieldDescriptor()
	}

	qBuilder, err := pagination.NewPaginatedQueryBuilder(
		input.PaginationOptions,
		&pagination.FieldDescriptor{Key: "id", Table: "run_gates", Col: "id"},
		pagination.WithSortByField(sortBy, sortDirection),
		pagination.WithQueryTag("rungate.GetRunGates"),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build query", errors.WithSpan(span))
	}

	rows, err := qBuilder.Execute(ctx, m.dbClient.getConnection(ctx), query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}
	defer rows.Close()

	results := []models.RunGate{}
	for rows.Next() {
		item, err := scanRunGate(rows)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan row", errors.WithSpan(span))
		}
		results = append(results, *item)
	}

	if err := rows.Finalize(&results); err != nil {
		return nil, errors.Wrap(err, "failed to finalize rows", errors.WithSpan(span))
	}

	return &RunGatesResult{
		PageInfo: rows.GetPageInfo(),
		RunGates: results,
	}, nil
}

func (m *runGates) CreateRunGate(ctx context.Context, gate *models.RunGate) (*models.RunGate, error) {
	ctx, span := tracer.Start(ctx, "db.CreateRunGate")
	defer span.End()

	timestamp := currentTime()

	tx, err := m.dbClient.getConnection(ctx).Begin(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin DB transaction", errors.WithSpan(span))
	}

	defer func() {
		if txErr := tx.Rollback(ctx); txErr != nil && txErr != pgx.ErrTxClosed {
			m.dbClient.logger.WithContextFields(ctx).Errorf("failed to rollback tx for CreateRunGate: %v", txErr)
		}
	}()

	gateID := newResourceID()

	approvalRules, err := json.Marshal(gate.ApprovalRules)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal run gate approval rules", errors.WithSpan(span))
	}

	sql, args, err := toSQLWithTag("rungate.CreateRunGate", dialect.Insert("run_gates").
		Prepared(true).
		Rows(goqu.Record{
			"id":              gateID,
			"version":         initialResourceVersion,
			"created_at":      timestamp,
			"updated_at":      timestamp,
			"run_id":          gate.RunID,
			"workspace_id":    gate.WorkspaceID,
			"policy_check_id": gate.PolicyCheckID,
			"type":            gate.Type,
			"approval_rules":  approvalRules,
			"status":          gate.Status,
		}))
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	if _, err := tx.Exec(ctx, sql, args...); err != nil {
		if pgErr := asPgError(err); pgErr != nil && isForeignKeyViolation(pgErr) {
			// run_id and workspace_id both have FKs; name the one that failed rather than always
			// blaming the run.
			switch pgErr.ConstraintName {
			case "fk_run_gates_workspace_id":
				return nil, errors.New("workspace does not exist", errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
			default:
				return nil, errors.New("run does not exist", errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
			}
		}
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	// Seed the query-index tables from the gate's allowed subjects, deduped by type.
	allowedUserIDs, allowedServiceAccountIDs, allowedTeamIDs := allowedSubjectIDsByType(gate)
	inserts := []struct {
		table string
		col   string
		ids   []string
	}{
		{"run_gate_allowed_users", "user_id", allowedUserIDs},
		{"run_gate_allowed_service_accounts", "service_account_id", allowedServiceAccountIDs},
		{"run_gate_allowed_teams", "team_id", allowedTeamIDs},
	}

	for _, ins := range inserts {
		if len(ins.ids) == 0 {
			continue
		}

		rows := make([]interface{}, len(ins.ids))
		for i, id := range ins.ids {
			rows[i] = goqu.Record{
				"id":      newResourceID(),
				"gate_id": gateID,
				ins.col:   id,
			}
		}

		iSQL, iArgs, iErr := toSQLWithTag("rungate.CreateRunGate", dialect.Insert(ins.table).
			Prepared(true).
			Rows(rows...))
		if iErr != nil {
			return nil, errors.Wrap(iErr, "failed to generate SQL", errors.WithSpan(span))
		}
		if _, iErr := tx.Exec(ctx, iSQL, iArgs...); iErr != nil {
			return nil, errors.Wrap(iErr, "failed to execute query", errors.WithSpan(span))
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, errors.Wrap(err, "failed to commit DB transaction", errors.WithSpan(span))
	}

	return m.GetRunGateByID(ctx, gateID)
}

// allowedSubjectIDsByType collects the raw principal IDs from a gate's approval rules' allowed
// subjects, deduped and split by principal type, to seed the run_gate_allowed_* query-index tables.
func allowedSubjectIDsByType(gate *models.RunGate) (userIDs, serviceAccountIDs, teamIDs []string) {
	users := map[string]struct{}{}
	serviceAccounts := map[string]struct{}{}
	teams := map[string]struct{}{}

	for _, rule := range gate.ApprovalRules {
		for _, subject := range rule.AllowedSubjects {
			switch subject.Type {
			case models.RunGateSubjectUser:
				users[subject.ID] = struct{}{}
			case models.RunGateSubjectServiceAccount:
				serviceAccounts[subject.ID] = struct{}{}
			case models.RunGateSubjectTeam:
				teams[subject.ID] = struct{}{}
			}
		}
	}

	return subjectMapKeys(users), subjectMapKeys(serviceAccounts), subjectMapKeys(teams)
}

func (m *runGates) UpdateRunGate(ctx context.Context, gate *models.RunGate) (*models.RunGate, error) {
	ctx, span := tracer.Start(ctx, "db.UpdateRunGate")
	defer span.End()

	timestamp := currentTime()

	sql, args, err := toSQLWithTag("rungate.UpdateRunGate", dialect.From("run_gates").
		Prepared(true).
		With("run_gates",
			dialect.Update("run_gates").
				Set(goqu.Record{
					"version":    goqu.L("? + ?", goqu.C("version"), 1),
					"updated_at": timestamp,
					"status":     gate.Status,
					// Written on every update, not just the override path. Callers update a gate they
					// read fresh from the DB, so an already-recorded override round-trips unchanged.
					"overridden_by":    gate.OverriddenBy,
					"override_comment": gate.OverrideComment,
				}).
				Where(goqu.Ex{"id": gate.Metadata.ID, "version": gate.Metadata.Version}).
				Returning("*"),
		).Select(m.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"run_gates.workspace_id": goqu.I("namespaces.workspace_id")})))
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	updated, err := scanRunGate(m.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrOptimisticLockError
		}
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return updated, nil
}

// DeleteRunGate removes a gate along with its approvals, allowed-subject index rows, and activity
// events, all of which cascade from the gate row. This is how a retried policy check discards its
// gate so a re-evaluation can create a fresh one; a gate on a run that merely reached a terminal
// status is canceled instead, so the decision it recorded survives.
func (m *runGates) DeleteRunGate(ctx context.Context, gate *models.RunGate) error {
	ctx, span := tracer.Start(ctx, "db.DeleteRunGate")
	defer span.End()

	sql, args, err := toSQLWithTag("rungate.DeleteRunGate", dialect.From("run_gates").
		Prepared(true).
		With("run_gates",
			dialect.Delete("run_gates").
				Where(goqu.Ex{"id": gate.Metadata.ID, "version": gate.Metadata.Version}).
				Returning("*"),
		).Select(m.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"run_gates.workspace_id": goqu.I("namespaces.workspace_id")})))
	if err != nil {
		return errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	if _, err := scanRunGate(m.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...)); err != nil {
		if err == pgx.ErrNoRows {
			return ErrOptimisticLockError
		}
		return errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return nil
}

func (m *runGates) getRunGate(ctx context.Context, ex goqu.Ex) (*models.RunGate, error) {
	sql, args, err := toSQLWithTag("rungate.getRunGate", dialect.From("run_gates").
		Prepared(true).
		Select(m.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"run_gates.workspace_id": goqu.I("namespaces.workspace_id")})).
		Where(ex))
	if err != nil {
		return nil, err
	}

	gate, err := scanRunGate(m.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		if pgErr := asPgError(err); pgErr != nil && isInvalidIDViolation(pgErr) {
			return nil, ErrInvalidID
		}
		return nil, err
	}

	return gate, nil
}

func (m *runGates) getSelectFields() []interface{} {
	selectFields := []interface{}{}
	for _, field := range runGateFieldList {
		selectFields = append(selectFields, fmt.Sprintf("run_gates.%s", field))
	}
	selectFields = append(selectFields, "namespaces.path")
	return selectFields
}

func scanRunGate(row scanner) (*models.RunGate, error) {
	gate := &models.RunGate{}

	var approvalRules []byte
	var workspacePath string

	fields := []interface{}{
		&gate.Metadata.ID,
		&gate.Metadata.CreationTimestamp,
		&gate.Metadata.LastUpdatedTimestamp,
		&gate.Metadata.Version,
		&gate.RunID,
		&gate.WorkspaceID,
		&gate.PolicyCheckID,
		&gate.Type,
		&approvalRules,
		&gate.Status,
		&gate.OverriddenBy,
		&gate.OverrideComment,
		&workspacePath,
	}

	if err := row.Scan(fields...); err != nil {
		return nil, err
	}

	if len(approvalRules) > 0 {
		if err := json.Unmarshal(approvalRules, &gate.ApprovalRules); err != nil {
			return nil, err
		}
	}

	runGID := gid.ToGlobalID(types.RunModelType, gate.RunID)
	gate.Metadata.TRN = trn.TypeRunGate.Build(workspacePath, runGID, gate.GetGlobalID())

	return gate, nil
}

// subjectMapKeys returns the subjectMapKeys of a set as a slice.
func subjectMapKeys(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	return out
}
