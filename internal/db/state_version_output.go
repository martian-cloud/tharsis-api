package db

import (
	"context"

	"github.com/doug-martin/goqu/v9"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// StateVersionOutputs encapsulates the logic to access state version outputs from the database
type StateVersionOutputs interface {
	CreateStateVersionOutput(ctx context.Context, stateVersionOutput *models.StateVersionOutput) (*models.StateVersionOutput, error)
	GetStateVersionOutputs(ctx context.Context, stateVersionID string) ([]models.StateVersionOutput, error)
	GetStateVersionOutputByName(ctx context.Context, stateVersionID, outputName string) (*models.StateVersionOutput, error)
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
func (ro *stateVersionOutputs) CreateStateVersionOutput(ctx context.Context,
	stateVersionOutput *models.StateVersionOutput) (*models.StateVersionOutput, error) {
	timestamp := currentTime()

	sql, args, err := dialect.Insert("state_version_outputs").
		Prepared(true).
		Rows(goqu.Record{
			"id":               newResourceID(),
			"version":          initialResourceVersion,
			"created_at":       timestamp,
			"updated_at":       timestamp,
			"name":             stateVersionOutput.Name,
			"value":            stateVersionOutput.Value,
			"type":             stateVersionOutput.Type,
			"sensitive":        stateVersionOutput.Sensitive,
			"state_version_id": nullableString(stateVersionOutput.StateVersionID),
		}).
		Returning(stateVersionOutputFieldList...).ToSQL()

	if err != nil {
		return nil, err
	}

	createdStateVersionOutput, err := scanStateVersionOutput(ro.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))

	if err != nil {
		ro.dbClient.logger.Error(err)
		return nil, err
	}
	return createdStateVersionOutput, nil
}

// GetStateVersionOutputs returns a slice of state version outputs.  It does _NOT_ do pagination.
func (ro *stateVersionOutputs) GetStateVersionOutputs(ctx context.Context,
	stateVersionID string) ([]models.StateVersionOutput, error) {
	sql, args, err := dialect.From("state_version_outputs").
		Prepared(true).
		Select(stateVersionOutputFieldList...).
		Where(goqu.Ex{"state_version_id": stateVersionID}).
		ToSQL()

	if err != nil {
		return nil, err
	}

	rows, err := ro.dbClient.getConnection(ctx).Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Scan rows
	results := []models.StateVersionOutput{}
	for rows.Next() {
		item, err := scanStateVersionOutput(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, *item)
	}

	return results, nil
}

// GetStateVersionOutputByName returns a state version output by name
func (ro *stateVersionOutputs) GetStateVersionOutputByName(ctx context.Context,
	stateVersionID, outputName string) (*models.StateVersionOutput, error) {
	sql, args, err := dialect.From("state_version_outputs").
		Select(stateVersionOutputFieldList...).
		Where(goqu.Ex{"state_version_id": stateVersionID, "name": outputName}).
		ToSQL()

	if err != nil {
		return nil, err
	}

	stateVersionOutput, err := scanStateVersionOutput(ro.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))

	if err != nil {
		return nil, err
	}
	return stateVersionOutput, nil
}

func scanStateVersionOutput(row scanner) (*models.StateVersionOutput, error) {

	stateVersionOutput := &models.StateVersionOutput{}

	err := row.Scan(
		&stateVersionOutput.Metadata.ID,
		&stateVersionOutput.Metadata.CreationTimestamp,
		&stateVersionOutput.Metadata.LastUpdatedTimestamp,
		&stateVersionOutput.Metadata.Version,
		&stateVersionOutput.Name,
		&stateVersionOutput.Value,
		&stateVersionOutput.Type,
		&stateVersionOutput.Sensitive,
		&stateVersionOutput.StateVersionID, // cannot be null
	)
	if err != nil {
		return nil, err
	}

	return stateVersionOutput, nil
}
