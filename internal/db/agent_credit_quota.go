package db

//go:generate go tool mockery --name AgentCreditQuotas --inpackage --case underscore

import (
	"context"
	"fmt"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v4"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// AgentCreditQuotas encapsulates the logic to access agent credit quotas from the database
type AgentCreditQuotas interface {
	GetAgentCreditQuota(ctx context.Context, userID string, monthDate time.Time) (*models.AgentCreditQuota, error)
	CreateAgentCreditQuota(ctx context.Context, quota *models.AgentCreditQuota) (*models.AgentCreditQuota, error)
	AddCredits(ctx context.Context, id string, credits float64) error
}

type agentCreditQuotas struct {
	dbClient *Client
}

var agentCreditQuotaFieldList = append(metadataFieldList, "user_id", "month_date", "total_credits")

// NewAgentCreditQuotas returns an instance of the AgentCreditQuotas interface
func NewAgentCreditQuotas(dbClient *Client) AgentCreditQuotas {
	return &agentCreditQuotas{dbClient: dbClient}
}

func (a *agentCreditQuotas) GetAgentCreditQuota(ctx context.Context, userID string, monthDate time.Time) (*models.AgentCreditQuota, error) {
	ctx, span := tracer.Start(ctx, "db.GetAgentCreditQuota")
	defer span.End()

	sql, args, err := dialect.From(goqu.T("agent_credit_quotas")).
		Prepared(true).
		Select(a.getSelectFields()...).
		Where(goqu.Ex{"user_id": userID, "month_date": monthDate}).
		ToSQL()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	quota, err := scanAgentCreditQuota(a.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to query agent credit quota", errors.WithSpan(span))
	}

	return quota, nil
}

func (a *agentCreditQuotas) CreateAgentCreditQuota(ctx context.Context, quota *models.AgentCreditQuota) (*models.AgentCreditQuota, error) {
	ctx, span := tracer.Start(ctx, "db.CreateAgentCreditQuota")
	defer span.End()

	timestamp := currentTime()

	sql, args, err := dialect.Insert("agent_credit_quotas").Prepared(true).Rows(
		goqu.Record{
			"id":            newResourceID(),
			"version":       initialResourceVersion,
			"created_at":    timestamp,
			"updated_at":    timestamp,
			"user_id":       quota.UserID,
			"month_date":    quota.MonthDate,
			"total_credits": quota.TotalCredits,
		},
	).Returning(a.getSelectFields()...).ToSQL()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	created, err := scanAgentCreditQuota(a.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				return nil, errors.New("agent credit quota already exists", errors.WithErrorCode(errors.EConflict), errors.WithSpan(span))
			}
		}
		return nil, errors.Wrap(err, "failed to create agent credit quota", errors.WithSpan(span))
	}

	return created, nil
}

// AddCredits atomically increments total_credits to avoid race conditions.
func (a *agentCreditQuotas) AddCredits(ctx context.Context, id string, credits float64) error {
	ctx, span := tracer.Start(ctx, "db.AddCredits")
	defer span.End()

	timestamp := currentTime()

	sql, args, err := dialect.Update("agent_credit_quotas").Prepared(true).
		Set(goqu.Record{
			"version":       goqu.L("? + ?", goqu.C("version"), 1),
			"updated_at":    timestamp,
			"total_credits": goqu.L("? + ?", goqu.C("total_credits"), credits),
		}).
		Where(goqu.Ex{"id": id}).
		ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	tag, err := a.dbClient.getConnection(ctx).Exec(ctx, sql, args...)
	if err != nil {
		return errors.Wrap(err, "failed to add credits", errors.WithSpan(span))
	}

	if tag.RowsAffected() == 0 {
		tracing.RecordError(span, nil, "agent credit quota not found")
		return errors.New("agent credit quota not found", errors.WithErrorCode(errors.ENotFound))
	}

	return nil
}

func (*agentCreditQuotas) getSelectFields() []interface{} {
	selectFields := []interface{}{}
	for _, field := range agentCreditQuotaFieldList {
		selectFields = append(selectFields, fmt.Sprintf("agent_credit_quotas.%s", field))
	}
	return selectFields
}

func scanAgentCreditQuota(row scanner) (*models.AgentCreditQuota, error) {
	quota := &models.AgentCreditQuota{}

	fields := []interface{}{
		&quota.Metadata.ID,
		&quota.Metadata.CreationTimestamp,
		&quota.Metadata.LastUpdatedTimestamp,
		&quota.Metadata.Version,
		&quota.UserID,
		&quota.MonthDate,
		&quota.TotalCredits,
	}

	err := row.Scan(fields...)
	if err != nil {
		return nil, err
	}

	quota.Metadata.TRN = types.AgentCreditQuotaModelType.BuildTRN(quota.Metadata.ID)

	return quota, nil
}
