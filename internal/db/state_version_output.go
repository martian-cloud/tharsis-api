package db

import (
	"context"
	"fmt"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v4"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"go.opentelemetry.io/otel/attribute"
)

// StateVersionOutputs encapsulates the logic to access state version outputs from the database
type StateVersionOutputs interface {
	GetStateVersionOutputByID(ctx context.Context, id string) (*models.StateVersionOutput, error)
	GetStateVersionOutputByTRN(ctx context.Context, trn string) (*models.StateVersionOutput, error)
	CreateStateVersionOutput(ctx context.Context, stateVersionOutput *models.StateVersionOutput) (*models.StateVersionOutput, error)
	GetStateVersionOutputs(ctx context.Context, stateVersionID string) ([]models.StateVersionOutput, error)
}

type stateVersionOutputs struct {
	dbClient *Client
}

var stateVersionOutputFieldList = append(metadataFieldList,
	"name", "value", "type", "sensitive", "state_version_id")

// NewStateVersionOutputs returns an instance of the StateVersionOutput interface
func NewStateVersionOutputs(dbClient *Client) StateVersionOutputs {
	return &stateVersionOutputs{dbClient: dbClient}
}

// CreateStateVersionOutput creates a new state version output by name
func (s *stateVersionOutputs) CreateStateVersionOutput(ctx context.Context,
	stateVersionOutput *models.StateVersionOutput) (*models.StateVersionOutput, error) {
	ctx, span := tracer.Start(ctx, "db.CreateStateVersionOutput")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	timestamp := currentTime()

	sql, args, err := dialect.From("state_version_outputs").
		Prepared(true).
		With("state_version_outputs",
			dialect.Insert("state_version_outputs").
				Rows(goqu.Record{
					"id":               newResourceID(),
					"version":          initialResourceVersion,
					"created_at":       timestamp,
					"updated_at":       timestamp,
					"name":             stateVersionOutput.Name,
					"value":            stateVersionOutput.Value,
					"type":             stateVersionOutput.Type,
					"sensitive":        stateVersionOutput.Sensitive,
					"state_version_id": stateVersionOutput.StateVersionID,
				}).
				Returning("*"),
		).Select(s.getSelectFields()...).
		InnerJoin(goqu.T("state_versions"), goqu.On(goqu.Ex{"state_version_outputs.state_version_id": goqu.I("state_versions.id")})).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"state_versions.workspace_id": goqu.I("namespaces.workspace_id")})).
		ToSQL()

	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	createdStateVersionOutput, err := scanStateVersionOutput(s.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))

	if err != nil {
		s.dbClient.logger.Error(err)
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}
	return createdStateVersionOutput, nil
}

// GetStateVersionOutputs returns a slice of state version outputs.  It does _NOT_ do pagination.
func (s *stateVersionOutputs) GetStateVersionOutputs(ctx context.Context,
	stateVersionID string) ([]models.StateVersionOutput, error) {
	ctx, span := tracer.Start(ctx, "db.GetStateVersionOutputs")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	sql, args, err := dialect.From("state_version_outputs").
		Prepared(true).
		Select(s.getSelectFields()...).
		InnerJoin(goqu.T("state_versions"), goqu.On(goqu.Ex{"state_version_outputs.state_version_id": goqu.I("state_versions.id")})).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"state_versions.workspace_id": goqu.I("namespaces.workspace_id")})).
		Where(goqu.Ex{"state_version_id": stateVersionID}).
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

	// Scan rows
	results := []models.StateVersionOutput{}
	for rows.Next() {
		item, err := scanStateVersionOutput(rows)
		if err != nil {
			tracing.RecordError(span, err, "failed to scan row")
			return nil, err
		}
		results = append(results, *item)
	}

	return results, nil
}

func (s *stateVersionOutputs) GetStateVersionOutputByID(ctx context.Context, id string) (*models.StateVersionOutput, error) {
	ctx, span := tracer.Start(ctx, "db.GetStateVersionOutputByID")
	span.SetAttributes(attribute.String("id", id))
	defer span.End()

	return s.getStateVersionOutput(ctx, goqu.Ex{"state_version_outputs.id": id})
}

func (s *stateVersionOutputs) GetStateVersionOutputByTRN(ctx context.Context, trn string) (*models.StateVersionOutput, error) {
	ctx, span := tracer.Start(ctx, "db.GetStateVersionOutputByTRN")
	span.SetAttributes(attribute.String("trn", trn))
	defer span.End()

	path, err := types.StateVersionOutputModelType.ResourcePathFromTRN(trn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse TRN", errors.WithSpan(span))
	}

	parts := strings.Split(path, "/")
	if len(parts) < 3 {
		return nil, errors.New("a state version outputs TRN must have the workspace path, state version GID and output name separated by a forward slash",
			errors.WithErrorCode(errors.EInvalid),
			errors.WithSpan(span),
		)
	}

	return s.getStateVersionOutput(ctx,
		goqu.Ex{
			"state_version_outputs.name": parts[len(parts)-1],
			"state_versions.id":          gid.FromGlobalID(parts[len(parts)-2]),
			"namespaces.path":            strings.Join(parts[:len(parts)-2], "/"),
		},
	)
}

func (s *stateVersionOutputs) getStateVersionOutput(ctx context.Context, ex goqu.Ex) (*models.StateVersionOutput, error) {
	ctx, span := tracer.Start(ctx, "db.getStateVersionOutput")
	defer span.End()

	sql, args, err := dialect.From("state_version_outputs").
		Prepared(true).
		Select(s.getSelectFields()...).
		InnerJoin(goqu.T("state_versions"), goqu.On(goqu.Ex{"state_version_outputs.state_version_id": goqu.I("state_versions.id")})).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"state_versions.workspace_id": goqu.I("namespaces.workspace_id")})).
		Where(ex).
		ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
	}

	stateVersionOutput, err := scanStateVersionOutput(s.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
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

	return stateVersionOutput, nil
}

func (s *stateVersionOutputs) getSelectFields() []interface{} {
	selectFields := []interface{}{}
	for _, field := range stateVersionOutputFieldList {
		selectFields = append(selectFields, fmt.Sprintf("state_version_outputs.%s", field))
	}

	selectFields = append(selectFields, "state_versions.id", "namespaces.path")

	return selectFields
}

func scanStateVersionOutput(row scanner) (*models.StateVersionOutput, error) {
	var stateVersionID, workspacePath string
	stateVersionOutput := &models.StateVersionOutput{}

	fields := []interface{}{
		&stateVersionOutput.Metadata.ID,
		&stateVersionOutput.Metadata.CreationTimestamp,
		&stateVersionOutput.Metadata.LastUpdatedTimestamp,
		&stateVersionOutput.Metadata.Version,
		&stateVersionOutput.Name,
		&stateVersionOutput.Value,
		&stateVersionOutput.Type,
		&stateVersionOutput.Sensitive,
		&stateVersionOutput.StateVersionID,
		&stateVersionID,
		&workspacePath,
	}

	if err := row.Scan(fields...); err != nil {
		return nil, err
	}

	stateVersionOutput.Metadata.TRN = types.StateVersionOutputModelType.BuildTRN(
		workspacePath,
		gid.ToGlobalID(types.StateVersionModelType, stateVersionID),
		stateVersionOutput.Name,
	)

	return stateVersionOutput, nil
}
