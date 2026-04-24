package db

//go:generate go tool mockery --name AgentSessions --inpackage --case underscore

import (
	"context"
	"fmt"

	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v4"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// AgentSessions encapsulates the logic to access agent sessions from the database
type AgentSessions interface {
	GetAgentSessionByID(ctx context.Context, id string) (*models.AgentSession, error)
	GetAgentSessionByTRN(ctx context.Context, trn string) (*models.AgentSession, error)
	CreateAgentSession(ctx context.Context, session *models.AgentSession) (*models.AgentSession, error)
	UpdateAgentSession(ctx context.Context, session *models.AgentSession) (*models.AgentSession, error)
	DeleteOldestSessionsByUserID(ctx context.Context, userID string, keepCount int) error
}

type agentSessions struct {
	dbClient *Client
}

var agentSessionFieldList = append(metadataFieldList, "user_id", "total_credits")

// NewAgentSessions returns an instance of the AgentSessions interface
func NewAgentSessions(dbClient *Client) AgentSessions {
	return &agentSessions{dbClient: dbClient}
}

func (a *agentSessions) GetAgentSessionByID(ctx context.Context, id string) (*models.AgentSession, error) {
	ctx, span := tracer.Start(ctx, "db.GetAgentSessionByID")
	defer span.End()

	query := dialect.From(goqu.T("agent_sessions")).
		Prepared(true).
		Select(a.getSelectFields()...).
		Where(goqu.Ex{"agent_sessions.id": id})

	sql, args, err := query.ToSQL()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	session, err := scanAgentSession(a.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
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

	return session, nil
}

func (a *agentSessions) GetAgentSessionByTRN(ctx context.Context, trn string) (*models.AgentSession, error) {
	ctx, span := tracer.Start(ctx, "db.GetAgentSessionByTRN")
	defer span.End()

	path, err := types.AgentSessionModelType.ResourcePathFromTRN(trn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse TRN", errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
	}

	return a.GetAgentSessionByID(ctx, gid.FromGlobalID(path))
}

func (a *agentSessions) CreateAgentSession(ctx context.Context, session *models.AgentSession) (*models.AgentSession, error) {
	ctx, span := tracer.Start(ctx, "db.CreateAgentSession")
	defer span.End()

	timestamp := currentTime()

	sql, args, err := dialect.Insert("agent_sessions").Rows(
		goqu.Record{
			"id":            newResourceID(),
			"version":       initialResourceVersion,
			"created_at":    timestamp,
			"updated_at":    timestamp,
			"user_id":       session.UserID,
			"total_credits": session.TotalCredits,
		},
	).Returning(agentSessionFieldList...).ToSQL()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	created, err := scanAgentSession(a.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isForeignKeyViolation(pgErr) {
				return nil, errors.New("invalid user ID", errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
			}
			if isInvalidIDViolation(pgErr) {
				return nil, errors.Wrap(pgErr, "invalid ID; %s", pgErr.Message, errors.WithSpan(span), errors.WithErrorCode(errors.EInvalid))
			}
		}
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return created, nil
}

func (a *agentSessions) UpdateAgentSession(ctx context.Context, session *models.AgentSession) (*models.AgentSession, error) {
	ctx, span := tracer.Start(ctx, "db.UpdateAgentSession")
	defer span.End()

	timestamp := currentTime()

	sql, args, err := dialect.Update("agent_sessions").
		Set(goqu.Record{
			"version":       goqu.L("? + ?", goqu.C("version"), 1),
			"updated_at":    timestamp,
			"total_credits": session.TotalCredits,
		}).
		Where(goqu.Ex{"id": session.Metadata.ID, "version": session.Metadata.Version}).
		Returning(agentSessionFieldList...).
		ToSQL()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	updated, err := scanAgentSession(a.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return nil, ErrOptimisticLockError
		}
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return updated, nil
}

// DeleteOldestSessionsByUserID deletes the oldest sessions for a user, keeping only keepCount most recent.
func (a *agentSessions) DeleteOldestSessionsByUserID(ctx context.Context, userID string, keepCount int) error {
	ctx, span := tracer.Start(ctx, "db.DeleteOldestSessionsByUserID")
	defer span.End()

	// Delete all sessions except the N most recent (by created_at desc)
	sql, args, err := dialect.Delete("agent_sessions").
		Prepared(true).
		Where(
			goqu.C("user_id").Eq(userID),
			goqu.C("id").NotIn(
				dialect.From("agent_sessions").
					Select("id").
					Where(goqu.C("user_id").Eq(userID)).
					Order(goqu.C("created_at").Desc()).
					Limit(uint(keepCount)),
			),
		).ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	if _, err := a.dbClient.getConnection(ctx).Exec(ctx, sql, args...); err != nil {
		return errors.Wrap(err, "failed to delete old sessions", errors.WithSpan(span))
	}

	return nil
}

func (*agentSessions) getSelectFields() []interface{} {
	selectFields := []interface{}{}
	for _, field := range agentSessionFieldList {
		selectFields = append(selectFields, fmt.Sprintf("agent_sessions.%s", field))
	}
	return selectFields
}

func scanAgentSession(row scanner) (*models.AgentSession, error) {
	session := &models.AgentSession{}

	fields := []interface{}{
		&session.Metadata.ID,
		&session.Metadata.CreationTimestamp,
		&session.Metadata.LastUpdatedTimestamp,
		&session.Metadata.Version,
		&session.UserID,
		&session.TotalCredits,
	}

	err := row.Scan(fields...)
	if err != nil {
		return nil, err
	}

	session.Metadata.TRN = types.AgentSessionModelType.BuildTRN(session.GetGlobalID())

	return session, nil
}
