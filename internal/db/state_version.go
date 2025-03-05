package db

//go:generate go tool mockery --name StateVersions --inpackage --case underscore

import (
	"context"
	"strings"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v4"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// StateVersionSortableField represents the fields that a list of state versions can be sorted by
type StateVersionSortableField string

// StateVersionSortableField constants
const (
	StateVersionSortableFieldUpdatedAtAsc  StateVersionSortableField = "UPDATED_AT_ASC"
	StateVersionSortableFieldUpdatedAtDesc StateVersionSortableField = "UPDATED_AT_DESC"
)

func (sf StateVersionSortableField) getFieldDescriptor() *pagination.FieldDescriptor {
	switch sf {
	case StateVersionSortableFieldUpdatedAtAsc, StateVersionSortableFieldUpdatedAtDesc:
		return &pagination.FieldDescriptor{Key: "updated_at", Table: "state_versions", Col: "updated_at"}
	default:
		return nil
	}
}

func (sf StateVersionSortableField) getSortDirection() pagination.SortDirection {
	if strings.HasSuffix(string(sf), "_DESC") {
		return pagination.DescSort
	}
	return pagination.AscSort
}

// StateVersionFilter contains the supported fields for filtering StateVersion resources
type StateVersionFilter struct {
	TimeRangeStart  *time.Time
	WorkspaceID     *string
	RunIDs          []string
	StateVersionIDs []string
}

// GetStateVersionsInput is the input for listing state versions
type GetStateVersionsInput struct {
	// Sort specifies the field to sort on and direction
	Sort *StateVersionSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// Filter is used to filter the results
	Filter *StateVersionFilter
}

// StateVersionsResult contains the response data and page information
type StateVersionsResult struct {
	PageInfo      *pagination.PageInfo
	StateVersions []models.StateVersion
}

// StateVersions encapsulates the logic to access stateVersions from the database
type StateVersions interface {
	GetStateVersions(ctx context.Context, input *GetStateVersionsInput) (*StateVersionsResult, error)
	// GetStateVersion returns a stateVersion by ID
	GetStateVersion(ctx context.Context, id string) (*models.StateVersion, error)
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
	ctx, span := tracer.Start(ctx, "db.GetStateVersions")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	ex := goqu.And()

	if input.Filter != nil {
		if input.Filter.StateVersionIDs != nil {
			ex = ex.Append(goqu.I("state_versions.id").In(input.Filter.StateVersionIDs))
		}
		if input.Filter.WorkspaceID != nil {
			ex = ex.Append(goqu.I("state_versions.workspace_id").Eq(*input.Filter.WorkspaceID))
		}
		if len(input.Filter.RunIDs) > 0 {
			ex = ex.Append(goqu.I("state_versions.run_id").In(input.Filter.RunIDs))
		}
		if input.Filter.TimeRangeStart != nil {
			// Must use UTC here otherwise, queries will return unexpected results.
			ex = ex.Append(goqu.I("state_versions.created_at").Gte(input.Filter.TimeRangeStart.UTC()))
		}
	}

	query := dialect.From("state_versions").
		Select(stateVersionFieldList...).
		Where(ex)

	sortDirection := pagination.AscSort

	var sortBy *pagination.FieldDescriptor
	if input.Sort != nil {
		sortDirection = input.Sort.getSortDirection()
		sortBy = input.Sort.getFieldDescriptor()
	}

	qBuilder, err := pagination.NewPaginatedQueryBuilder(
		input.PaginationOptions,
		&pagination.FieldDescriptor{Key: "id", Table: "state_versions", Col: "id"},
		pagination.WithSortByField(sortBy, sortDirection),
	)

	if err != nil {
		tracing.RecordError(span, err, "failed to build query")
		return nil, err
	}

	rows, err := qBuilder.Execute(ctx, s.dbClient.getConnection(ctx), query)
	if err != nil {
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	defer rows.Close()

	// Scan rows
	results := []models.StateVersion{}
	for rows.Next() {
		item, err := scanStateVersion(rows)
		if err != nil {
			tracing.RecordError(span, err, "failed to scan row")
			return nil, err
		}

		results = append(results, *item)
	}

	if err := rows.Finalize(&results); err != nil {
		tracing.RecordError(span, err, "failed to finalize rows")
		return nil, err
	}

	result := StateVersionsResult{
		PageInfo:      rows.GetPageInfo(),
		StateVersions: results,
	}

	return &result, nil
}

func (s *stateVersions) GetStateVersion(ctx context.Context, id string) (*models.StateVersion, error) {
	ctx, span := tracer.Start(ctx, "db.GetStateVersion")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	sql, args, err := dialect.From("state_versions").
		Prepared(true).
		Select(stateVersionFieldList...).
		Where(goqu.Ex{"id": id}).
		ToSQL()

	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	stateVersion, err := scanStateVersion(s.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}
	return stateVersion, nil
}

func (s *stateVersions) CreateStateVersion(ctx context.Context, stateVersion *models.StateVersion) (*models.StateVersion, error) {
	ctx, span := tracer.Start(ctx, "db.CreateStateVersion")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

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
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	createdStateVersion, err := scanStateVersion(s.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))

	if err != nil {
		s.dbClient.logger.Error(err)
		tracing.RecordError(span, err, "failed to execute query")
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
