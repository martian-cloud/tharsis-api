package db

import (
	"context"
	"fmt"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v4"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// StateVersionSortableField represents the fields that a list of state versions can be sorted by
type StateVersionSortableField string

// StateVersionSortableField constants
const (
	StateVersionSortableFieldUpdatedAtAsc  StateVersionSortableField = "UPDATED_AT_ASC"
	StateVersionSortableFieldUpdatedAtDesc StateVersionSortableField = "UPDATED_AT_DESC"
)

func (sf StateVersionSortableField) getFieldDescriptor() *fieldDescriptor {
	switch sf {
	case StateVersionSortableFieldUpdatedAtAsc, StateVersionSortableFieldUpdatedAtDesc:
		return &fieldDescriptor{key: "updated_at", table: "state_versions", col: "updated_at"}
	default:
		return nil
	}
}

func (sf StateVersionSortableField) getSortDirection() SortDirection {
	if strings.HasSuffix(string(sf), "_DESC") {
		return DescSort
	}
	return AscSort
}

// StateVersionFilter contains the supported fields for filtering StateVersion resources
type StateVersionFilter struct {
	WorkspaceID     *string
	StateVersionIDs []string
}

// GetStateVersionsInput is the input for listing state versions
type GetStateVersionsInput struct {
	// Sort specifies the field to sort on and direction
	Sort *StateVersionSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *PaginationOptions
	// Filter is used to filter the results
	Filter *StateVersionFilter
}

// StateVersionsResult contains the response data and page information
type StateVersionsResult struct {
	PageInfo      *PageInfo
	StateVersions []models.StateVersion
}

// StateVersions encapsulates the logic to access stateVersions from the database
type StateVersions interface {
	GetStateVersions(ctx context.Context, input *GetStateVersionsInput) (*StateVersionsResult, error)
	// GetStateVersion returns a stateVersion by ID
	GetStateVersion(ctx context.Context, id string) (*models.StateVersion, error)
	// GetStateVersionByRunID returns the state version associated with the specified run
	GetStateVersionByRunID(ctx context.Context, runID string) (*models.StateVersion, error)
	// CreateStateVersion will create a new stateVersion
	CreateStateVersion(ctx context.Context, stateVersion *models.StateVersion) (*models.StateVersion, error)
}

type stateVersions struct {
	dbClient *Client
}

var stateVersionFieldList = append(metadataFieldList, "workspace_id", "run_id", "created_by")

// NewStateVersions returns an instance of the StateVersion interface
func NewStateVersions(dbClient *Client) StateVersions {
	return &stateVersions{dbClient: dbClient}
}

func (s *stateVersions) GetStateVersions(ctx context.Context,
	input *GetStateVersionsInput) (*StateVersionsResult, error) {
	ex := goqu.Ex{}

	if input.Filter != nil {
		if input.Filter.StateVersionIDs != nil {
			ex["state_versions.id"] = input.Filter.StateVersionIDs
		}
		if input.Filter.WorkspaceID != nil {
			ex["state_versions.workspace_id"] = *input.Filter.WorkspaceID
		}
	}

	query := dialect.From("state_versions").
		Select(stateVersionFieldList...).
		Where(ex)

	sortDirection := AscSort

	var sortBy *fieldDescriptor
	if input.Sort != nil {
		sortDirection = input.Sort.getSortDirection()
		sortBy = input.Sort.getFieldDescriptor()
	}

	qBuilder, err := newPaginatedQueryBuilder(
		input.PaginationOptions,
		&fieldDescriptor{key: "id", table: "state_versions", col: "id"},
		sortBy,
		sortDirection,
		stateVersionFieldResolver,
	)

	if err != nil {
		return nil, err
	}

	rows, err := qBuilder.execute(ctx, s.dbClient.getConnection(ctx), query)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	// Scan rows
	results := []models.StateVersion{}
	for rows.Next() {
		item, err := scanStateVersion(rows)
		if err != nil {
			return nil, err
		}

		results = append(results, *item)
	}

	if err := rows.finalize(&results); err != nil {
		return nil, err
	}

	result := StateVersionsResult{
		PageInfo:      rows.getPageInfo(),
		StateVersions: results,
	}

	return &result, nil
}

func (s *stateVersions) GetStateVersionByRunID(ctx context.Context, runID string) (*models.StateVersion, error) {
	sql, args, err := dialect.From("state_versions").
		Prepared(true).
		Select(stateVersionFieldList...).
		Where(goqu.Ex{"run_id": runID}).
		ToSQL()

	if err != nil {
		return nil, err
	}

	stateVersion, err := scanStateVersion(s.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return stateVersion, nil
}

func (s *stateVersions) GetStateVersion(ctx context.Context, id string) (*models.StateVersion, error) {

	sql, args, err := dialect.From("state_versions").
		Prepared(true).
		Select(stateVersionFieldList...).
		Where(goqu.Ex{"id": id}).
		ToSQL()

	if err != nil {
		return nil, err
	}

	stateVersion, err := scanStateVersion(s.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return stateVersion, nil
}

func (s *stateVersions) CreateStateVersion(ctx context.Context, stateVersion *models.StateVersion) (*models.StateVersion, error) {
	timestamp := currentTime()

	sql, args, err := dialect.Insert("state_versions").
		Prepared(true).
		Rows(goqu.Record{
			"id":           newResourceID(),
			"version":      initialResourceVersion,
			"created_at":   timestamp,
			"updated_at":   timestamp,
			"workspace_id": stateVersion.WorkspaceID,
			"run_id":       stateVersion.RunID,
			"created_by":   stateVersion.CreatedBy,
		}).
		Returning(stateVersionFieldList...).ToSQL()

	if err != nil {
		return nil, err
	}

	createdStateVersion, err := scanStateVersion(s.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))

	if err != nil {
		s.dbClient.logger.Error(err)
		return nil, err
	}
	return createdStateVersion, nil
}

func scanStateVersion(row scanner) (*models.StateVersion, error) {
	stateVersion := &models.StateVersion{}

	err := row.Scan(
		&stateVersion.Metadata.ID,
		&stateVersion.Metadata.CreationTimestamp,
		&stateVersion.Metadata.LastUpdatedTimestamp,
		&stateVersion.Metadata.Version,
		&stateVersion.WorkspaceID,
		&stateVersion.RunID,
		&stateVersion.CreatedBy,
	)
	if err != nil {
		return nil, err
	}

	return stateVersion, nil
}

func stateVersionFieldResolver(key string, model interface{}) (string, error) {
	stateVersion, ok := model.(*models.StateVersion)
	if !ok {
		return "", errors.NewError(errors.EInternal, fmt.Sprintf("Expected stateVersion type, got %T", model))
	}

	val, ok := metadataFieldResolver(key, &stateVersion.Metadata)
	if !ok {
		return "", errors.NewError(errors.EInternal, fmt.Sprintf("Invalid field key requested %s", key))
	}

	return val, nil
}

// The End.
