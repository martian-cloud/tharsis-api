package db

//go:generate go tool mockery --name SCIMTokens --inpackage --case underscore

import (
	"context"
	"fmt"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/jackc/pgx/v4"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// SCIMTokens encapsulates the logic to access SCIM tokens from the database
type SCIMTokens interface {
	GetTokenByNonce(ctx context.Context, nonce string) (*models.SCIMToken, error)
	GetTokens(ctx context.Context) ([]models.SCIMToken, error)
	CreateToken(ctx context.Context, token *models.SCIMToken) (*models.SCIMToken, error)
	DeleteToken(ctx context.Context, token *models.SCIMToken) error
}

type scimTokens struct {
	dbClient *Client
}

var scimTokensFieldList = append(metadataFieldList, "created_by", "nonce")

// NewSCIMTokens returns an instance of the SCIMTokens interface.
func NewSCIMTokens(dbClient *Client) SCIMTokens {
	return &scimTokens{dbClient: dbClient}
}

func (s *scimTokens) GetTokenByNonce(ctx context.Context, nonce string) (*models.SCIMToken, error) {
	ctx, span := tracer.Start(ctx, "db.GetTokenByNonce")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	return s.getToken(ctx, goqu.Ex{"scim_tokens.nonce": nonce})
}

func (s *scimTokens) GetTokens(ctx context.Context) ([]models.SCIMToken, error) {
	ctx, span := tracer.Start(ctx, "db.GetTokens")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	sql, args, err := dialect.From("scim_tokens").
		Prepared(true).
		Select(s.getSelectFields()...).
		ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	rows, err := s.dbClient.getConnection(ctx).Query(ctx, sql, args...)
	if err != nil {
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	defer rows.Close()

	results := []models.SCIMToken{}
	for rows.Next() {
		item, err := scanSCIMToken(rows)
		if err != nil {
			tracing.RecordError(span, err, "failed to scan row")
			return nil, err
		}

		results = append(results, *item)
	}

	return results, nil
}

func (s *scimTokens) CreateToken(ctx context.Context, token *models.SCIMToken) (*models.SCIMToken, error) {
	ctx, span := tracer.Start(ctx, "db.CreateToken")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	timestamp := currentTime()

	sql, args, err := dialect.Insert("scim_tokens").
		Prepared(true).
		Rows(goqu.Record{
			"id":         newResourceID(),
			"version":    initialResourceVersion,
			"created_at": timestamp,
			"updated_at": timestamp,
			"created_by": token.CreatedBy,
			"nonce":      token.Nonce,
		}).
		Returning(scimTokensFieldList...).ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	createdToken, err := scanSCIMToken(s.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				tracing.RecordError(span, nil, "SCIM token already exists")
				return nil, errors.New("SCIM token already exists", errors.WithErrorCode(errors.EConflict))
			}
		}
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	return createdToken, nil
}

func (s *scimTokens) DeleteToken(ctx context.Context, token *models.SCIMToken) error {
	ctx, span := tracer.Start(ctx, "db.DeleteToken")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	sql, args, err := dialect.Delete("scim_tokens").
		Prepared(true).
		Where(
			goqu.Ex{
				"id":      token.Metadata.ID,
				"version": token.Metadata.Version,
			},
		).Returning(scimTokensFieldList...).ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return err
	}

	if _, err = scanSCIMToken(s.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...)); err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return ErrOptimisticLockError
		}

		tracing.RecordError(span, err, "failed to execute query")
		return err
	}

	return nil
}

func (s *scimTokens) getToken(ctx context.Context, exp exp.Ex) (*models.SCIMToken, error) {
	ctx, span := tracer.Start(ctx, "db.getToken")
	defer span.End()

	sql, args, err := dialect.From("scim_tokens").
		Prepared(true).
		Select(s.getSelectFields()...).
		Where(exp).
		ToSQL()

	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	token, err := scanSCIMToken(s.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}

		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}
	return token, nil
}

func (s *scimTokens) getSelectFields() []interface{} {
	selectFields := []interface{}{}
	for _, field := range scimTokensFieldList {
		selectFields = append(selectFields, fmt.Sprintf("scim_tokens.%s", field))
	}

	return selectFields
}

func scanSCIMToken(row scanner) (*models.SCIMToken, error) {
	token := &models.SCIMToken{}

	fields := []interface{}{
		&token.Metadata.ID,
		&token.Metadata.CreationTimestamp,
		&token.Metadata.LastUpdatedTimestamp,
		&token.Metadata.Version,
		&token.CreatedBy,
		&token.Nonce,
	}

	err := row.Scan(fields...)
	if err != nil {
		return nil, err
	}

	token.Metadata.TRN = types.SCIMTokenModelType.BuildTRN(token.GetGlobalID())

	return token, nil
}
