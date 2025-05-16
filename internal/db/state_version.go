package db

//go:generate go tool mockery --name StateVersions --inpackage --case underscore

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v4"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
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
	// GetStateVersionByID returns a stateVersion by ID
	GetStateVersionByID(ctx context.Context, id string) (*models.StateVersion, error)
	// GetStateVersionByTRN returns a state version by TRN
	GetStateVersionByTRN(ctx context.Context, trn string) (*models.StateVersion, error)
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
		Select(s.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"state_versions.workspace_id": goqu.I("namespaces.workspace_id")})).
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

func (s *stateVersions) GetStateVersionByID(ctx context.Context, id string) (*models.StateVersion, error) {
	ctx, span := tracer.Start(ctx, "db.GetStateVersionByID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	return s.getStateVersion(ctx, goqu.Ex{"state_versions.id": id})
}

func (s *stateVersions) GetStateVersionByTRN(ctx context.Context, trn string) (*models.StateVersion, error) {
	ctx, span := tracer.Start(ctx, "db.GetStateVersionByTRN")
	defer span.End()

	path, err := types.StateVersionModelType.ResourcePathFromTRN(trn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse TRN", errors.WithSpan(span))
	}

	lastSlashIndex := strings.LastIndex(path, "/")
	if lastSlashIndex == -1 {
		return nil, errors.New("a state version TRN must have the workspace path and GID separated by a forward slash",
			errors.WithErrorCode(errors.EInvalid),
			errors.WithSpan(span),
		)
	}

	return s.getStateVersion(ctx, goqu.Ex{
		"state_versions.id": gid.FromGlobalID(path[lastSlashIndex+1:]),
		"namespaces.path":   path[:lastSlashIndex],
	})
}

func (s *stateVersions) CreateStateVersion(ctx context.Context, stateVersion *models.StateVersion) (*models.StateVersion, error) {
	ctx, span := tracer.Start(ctx, "db.CreateStateVersion")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	timestamp := currentTime()

	sql, args, err := dialect.From("state_versions").
		Prepared(true).
		With("state_versions",
			dialect.Insert("state_versions").
				Rows(goqu.Record{
					"id":           newResourceID(),
					"version":      initialResourceVersion,
					"created_at":   timestamp,
					"updated_at":   timestamp,
					"workspace_id": stateVersion.WorkspaceID,
					"run_id":       stateVersion.RunID,
					"created_by":   stateVersion.CreatedBy,
				}).Returning("*"),
		).Select(s.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"state_versions.workspace_id": goqu.I("namespaces.workspace_id")})).
		ToSQL()

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

func (s *stateVersions) getStateVersion(ctx context.Context, ex goqu.Ex) (*models.StateVersion, error) {
	ctx, span := tracer.Start(ctx, "db.getStateVersion")
	defer span.End()

	sql, args, err := dialect.From("state_versions").
		Prepared(true).
		Select(s.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"state_versions.workspace_id": goqu.I("namespaces.workspace_id")})).
		Where(ex).
		ToSQL()

	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	stateVersion, err := scanStateVersion(s.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
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

	return stateVersion, nil
}

func (s *stateVersions) getSelectFields() []interface{} {
	selectFields := []interface{}{}
	for _, field := range stateVersionFieldList {
		selectFields = append(selectFields, fmt.Sprintf("state_versions.%s", field))
	}

	selectFields = append(selectFields, "namespaces.path")

	return selectFields
}

func scanStateVersion(row scanner) (*models.StateVersion, error) {
	var workspacePath string
	stateVersion := &models.StateVersion{}

	err := row.Scan(
		&stateVersion.Metadata.ID,
		&stateVersion.Metadata.CreationTimestamp,
		&stateVersion.Metadata.LastUpdatedTimestamp,
		&stateVersion.Metadata.Version,
		&stateVersion.WorkspaceID,
		&stateVersion.RunID,
		&stateVersion.CreatedBy,
		&workspacePath,
	)
	if err != nil {
		return nil, err
	}

	stateVersion.Metadata.TRN = types.StateVersionModelType.BuildTRN(workspacePath, stateVersion.GetGlobalID())

	return stateVersion, nil
}
