package db

//go:generate go tool mockery --name RunGateApprovals --inpackage --case underscore

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v5"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
)

// RunGateApprovals encapsulates the logic to access run gate approvals from the database
type RunGateApprovals interface {
	GetRunGateApprovalByID(ctx context.Context, id string) (*models.RunGateApproval, error)
	GetRunGateApprovalByTRN(ctx context.Context, trnValue string) (*models.RunGateApproval, error)
	GetRunGateApprovalsByGateID(ctx context.Context, gateID string) ([]models.RunGateApproval, error)
	CreateRunGateApproval(ctx context.Context, approval *models.RunGateApproval) (*models.RunGateApproval, error)
	UpdateRunGateApproval(ctx context.Context, approval *models.RunGateApproval) (*models.RunGateApproval, error)
}

type runGateApprovals struct {
	dbClient *Client
}

var runGateApprovalFieldList = append(metadataFieldList,
	"run_gate_id",
	"user_id",
	"service_account_id",
	"created_by",
	"decision",
	"comment",
	"covered_rules",
)

// NewRunGateApprovals returns an instance of the RunGateApprovals interface
func NewRunGateApprovals(dbClient *Client) RunGateApprovals {
	return &runGateApprovals{dbClient: dbClient}
}

func (m *runGateApprovals) GetRunGateApprovalByID(ctx context.Context, id string) (*models.RunGateApproval, error) {
	ctx, span := tracer.Start(ctx, "db.GetRunGateApprovalByID")
	defer span.End()

	return m.getRunGateApproval(ctx, goqu.Ex{"run_gate_approvals.id": id})
}

func (m *runGateApprovals) GetRunGateApprovalByTRN(ctx context.Context, trnValue string) (*models.RunGateApproval, error) {
	ctx, span := tracer.Start(ctx, "db.GetRunGateApprovalByTRN")
	defer span.End()

	parsed, err := trn.TypeRunGateApproval.Parse(trnValue)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse TRN", errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
	}

	return m.getRunGateApproval(ctx, goqu.Ex{"run_gate_approvals.id": gid.FromGlobalID(parsed.BaseName())})
}

func (m *runGateApprovals) GetRunGateApprovalsByGateID(ctx context.Context, gateID string) ([]models.RunGateApproval, error) {
	ctx, span := tracer.Start(ctx, "db.GetRunGateApprovalsByGateID")
	defer span.End()

	sql, args, err := toSQLWithTag("rungateapproval.GetRunGateApprovalsByGateID", dialect.From("run_gate_approvals").
		Prepared(true).
		Select(m.getSelectFields()...).
		Where(goqu.Ex{"run_gate_approvals.run_gate_id": gateID}))
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	rows, err := m.dbClient.getConnection(ctx).Query(ctx, sql, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}
	defer rows.Close()

	results := []models.RunGateApproval{}
	for rows.Next() {
		item, err := scanRunGateApproval(rows)
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

func (m *runGateApprovals) CreateRunGateApproval(ctx context.Context, approval *models.RunGateApproval) (*models.RunGateApproval, error) {
	ctx, span := tracer.Start(ctx, "db.CreateRunGateApproval")
	defer span.End()

	timestamp := currentTime()

	coveredRules, err := json.Marshal(approval.CoveredRules)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal covered rules", errors.WithSpan(span))
	}

	sql, args, err := toSQLWithTag("rungateapproval.CreateRunGateApproval", dialect.Insert("run_gate_approvals").
		Prepared(true).
		Rows(goqu.Record{
			"id":                 newResourceID(),
			"version":            initialResourceVersion,
			"created_at":         timestamp,
			"updated_at":         timestamp,
			"run_gate_id":        approval.RunGateID,
			"user_id":            approval.UserID,
			"service_account_id": approval.ServiceAccountID,
			"created_by":         approval.CreatedBy,
			"decision":           approval.Decision,
			"comment":            approval.Comment,
			"covered_rules":      coveredRules,
		}).Returning(m.getSelectFields()...))
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	created, err := scanRunGateApproval(m.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isForeignKeyViolation(pgErr) {
				return nil, errors.New("run gate does not exist", errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
			}
			if isUniqueViolation(pgErr) {
				return nil, errors.New("subject has already submitted an approval for this gate", errors.WithErrorCode(errors.EConflict), errors.WithSpan(span))
			}
		}
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return created, nil
}

func (m *runGateApprovals) UpdateRunGateApproval(ctx context.Context, approval *models.RunGateApproval) (*models.RunGateApproval, error) {
	ctx, span := tracer.Start(ctx, "db.UpdateRunGateApproval")
	defer span.End()

	timestamp := currentTime()

	coveredRules, err := json.Marshal(approval.CoveredRules)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal covered rules", errors.WithSpan(span))
	}

	sql, args, err := toSQLWithTag("rungateapproval.UpdateRunGateApproval", dialect.From("run_gate_approvals").
		Prepared(true).
		With("run_gate_approvals",
			dialect.Update("run_gate_approvals").
				Set(goqu.Record{
					"version":       goqu.L("? + ?", goqu.C("version"), 1),
					"updated_at":    timestamp,
					"decision":      approval.Decision,
					"comment":       approval.Comment,
					"covered_rules": coveredRules,
				}).
				Where(goqu.Ex{"id": approval.Metadata.ID, "version": approval.Metadata.Version}).
				Returning("*"),
		).Select(m.getSelectFields()...))
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	updated, err := scanRunGateApproval(m.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrOptimisticLockError
		}
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return updated, nil
}

func (m *runGateApprovals) getRunGateApproval(ctx context.Context, ex goqu.Ex) (*models.RunGateApproval, error) {
	sql, args, err := toSQLWithTag("rungateapproval.getRunGateApproval", dialect.From("run_gate_approvals").
		Prepared(true).
		Select(m.getSelectFields()...).
		Where(ex))
	if err != nil {
		return nil, err
	}

	approval, err := scanRunGateApproval(m.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		if pgErr := asPgError(err); pgErr != nil && isInvalidIDViolation(pgErr) {
			return nil, ErrInvalidID
		}
		return nil, err
	}

	return approval, nil
}

func (m *runGateApprovals) getSelectFields() []interface{} {
	selectFields := []interface{}{}
	for _, field := range runGateApprovalFieldList {
		selectFields = append(selectFields, fmt.Sprintf("run_gate_approvals.%s", field))
	}
	return selectFields
}

func scanRunGateApproval(row scanner) (*models.RunGateApproval, error) {
	approval := &models.RunGateApproval{}

	var coveredRules []byte

	fields := []interface{}{
		&approval.Metadata.ID,
		&approval.Metadata.CreationTimestamp,
		&approval.Metadata.LastUpdatedTimestamp,
		&approval.Metadata.Version,
		&approval.RunGateID,
		&approval.UserID,
		&approval.ServiceAccountID,
		&approval.CreatedBy,
		&approval.Decision,
		&approval.Comment,
		&coveredRules,
	}

	if err := row.Scan(fields...); err != nil {
		return nil, err
	}

	if len(coveredRules) > 0 {
		if err := json.Unmarshal(coveredRules, &approval.CoveredRules); err != nil {
			return nil, err
		}
	}

	gateGID := gid.ToGlobalID(types.RunGateModelType, approval.RunGateID)
	approval.Metadata.TRN = trn.TypeRunGateApproval.Build(gateGID, approval.GetGlobalID())

	return approval, nil
}
