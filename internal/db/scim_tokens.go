package db

//go:generate mockery --name SCIMTokens --inpackage --case underscore

import (
	"context"
	"fmt"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/jackc/pgx/v4"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
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
	return s.getToken(ctx, goqu.Ex{"scim_tokens.nonce": nonce})
}

func (s *scimTokens) GetTokens(ctx context.Context) ([]models.SCIMToken, error) {
	sql, _, err := goqu.From("scim_tokens").Select(s.getSelectFields()...).ToSQL()
	if err != nil {
		return nil, err
	}

	rows, err := s.dbClient.getConnection(ctx).Query(ctx, sql)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	results := []models.SCIMToken{}
	for rows.Next() {
		item, err := scanToken(rows)
		if err != nil {
			return nil, err
		}

		results = append(results, *item)
	}

	return results, nil
}

func (s *scimTokens) CreateToken(ctx context.Context, token *models.SCIMToken) (*models.SCIMToken, error) {
	timestamp := currentTime()

	sql, _, err := dialect.Insert("scim_tokens").
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
		return nil, err
	}

	createdToken, err := scanToken(s.dbClient.getConnection(ctx).QueryRow(ctx, sql))
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				return nil, errors.NewError(errors.EConflict, "SCIM token already exists")
			}
		}
		return nil, err
	}

	return createdToken, nil
}

func (s *scimTokens) DeleteToken(ctx context.Context, token *models.SCIMToken) error {
	sql, _, err := dialect.Delete("scim_tokens").Where(
		goqu.Ex{
			"id":      token.Metadata.ID,
			"version": token.Metadata.Version,
		},
	).Returning(scimTokensFieldList...).ToSQL()
	if err != nil {
		return err
	}

	if _, err = scanToken(s.dbClient.getConnection(ctx).QueryRow(ctx, sql)); err != nil {
		if err == pgx.ErrNoRows {
			return ErrOptimisticLockError
		}

		return err
	}

	return nil
}

func (s *scimTokens) getToken(ctx context.Context, exp exp.Ex) (*models.SCIMToken, error) {
	sql, _, err := goqu.From("scim_tokens").
		Select(s.getSelectFields()...).
		Where(exp).
		ToSQL()

	if err != nil {
		return nil, err
	}

	token, err := scanToken(s.dbClient.getConnection(ctx).QueryRow(ctx, sql))

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
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

func scanToken(row scanner) (*models.SCIMToken, error) {
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

	return token, nil
}
