package db

//go:generate mockery --name Teams --inpackage --case underscore

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v4"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// Teams encapsulates the logic to access teams from the database
type Teams interface {
	GetTeamBySCIMExternalID(ctx context.Context, scimExternalID string) (*models.Team, error)
	GetTeamByID(ctx context.Context, id string) (*models.Team, error)
	GetTeamByName(ctx context.Context, name string) (*models.Team, error)
	GetTeams(ctx context.Context, input *GetTeamsInput) (*TeamsResult, error)
	CreateTeam(ctx context.Context, team *models.Team) (*models.Team, error)
	UpdateTeam(ctx context.Context, team *models.Team) (*models.Team, error)
	DeleteTeam(ctx context.Context, team *models.Team) error
}

// TeamSortableField represents the fields that a team can be sorted by
type TeamSortableField string

// TeamSortableField constants
const (
	TeamSortableFieldNameAsc       TeamSortableField = "NAME_ASC"
	TeamSortableFieldNameDesc      TeamSortableField = "NAME_DESC"
	TeamSortableFieldUpdatedAtAsc  TeamSortableField = "UPDATED_AT_ASC"
	TeamSortableFieldUpdatedAtDesc TeamSortableField = "UPDATED_AT_DESC"
)

func (ts TeamSortableField) getFieldDescriptor() *pagination.FieldDescriptor {
	switch ts {
	case TeamSortableFieldNameAsc, TeamSortableFieldNameDesc:
		return &pagination.FieldDescriptor{Key: "name", Table: "teams", Col: "name"}
	case TeamSortableFieldUpdatedAtAsc, TeamSortableFieldUpdatedAtDesc:
		return &pagination.FieldDescriptor{Key: "updated_at", Table: "teams", Col: "updated_at"}
	default:
		return nil
	}
}

func (ts TeamSortableField) getSortDirection() pagination.SortDirection {
	if strings.HasSuffix(string(ts), "_DESC") {
		return pagination.DescSort
	}
	return pagination.AscSort
}

// TeamFilter contains the supported fields for filtering Team resources
type TeamFilter struct {
	TeamNamePrefix *string
	UserID         *string
	TeamIDs        []string
	SCIMExternalID bool
}

// GetTeamsInput is the input for listing teams
type GetTeamsInput struct {
	// Sort specifies the field to sort on and direction
	Sort *TeamSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// Filter is used to filter the results
	Filter *TeamFilter
}

// TeamsResult contains the response data and page information
type TeamsResult struct {
	PageInfo *pagination.PageInfo
	Teams    []models.Team
}

type teams struct {
	dbClient *Client
}

var teamFieldList = append(metadataFieldList, "name", "description", "scim_external_id")

// NewTeams returns an instance of the Teams interface
func NewTeams(dbClient *Client) Teams {
	return &teams{dbClient: dbClient}
}

func (t *teams) GetTeamBySCIMExternalID(ctx context.Context, scimExternalID string) (*models.Team, error) {
	ctx, span := tracer.Start(ctx, "db.GetTeamBySCIMExternalID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	return t.getTeam(ctx, goqu.Ex{"teams.scim_external_id": scimExternalID})
}

func (t *teams) GetTeamByID(ctx context.Context, id string) (*models.Team, error) {
	ctx, span := tracer.Start(ctx, "db.GetTeamByID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	return t.getTeam(ctx, goqu.Ex{"teams.id": id})
}

func (t *teams) GetTeamByName(ctx context.Context, name string) (*models.Team, error) {
	ctx, span := tracer.Start(ctx, "db.GetTeamByName")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	return t.getTeam(ctx, goqu.Ex{"teams.name": name})
}

func (t *teams) GetTeams(ctx context.Context, input *GetTeamsInput) (*TeamsResult, error) {
	ctx, span := tracer.Start(ctx, "db.GetTeams")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	ex := goqu.Ex{}

	if input.Filter != nil {
		if input.Filter.TeamIDs != nil {
			ex["teams.id"] = input.Filter.TeamIDs
		}
		if input.Filter.TeamNamePrefix != nil && *input.Filter.TeamNamePrefix != "" {
			ex["teams.name"] = goqu.Op{"like": *input.Filter.TeamNamePrefix + "%"}
		}
		if input.Filter.UserID != nil {
			ex["team_members.user_id"] = *input.Filter.UserID
		}
		if input.Filter.SCIMExternalID {
			ex["teams.scim_external_id"] = goqu.Op{"isNot": nil}
		}
	}

	query := dialect.From(goqu.T("teams")).
		Select(teamFieldList...).
		Where(ex)

	// Don't want to pay the cost to do an inner join unless necessary.
	if (input.Filter != nil) && (input.Filter.UserID != nil) {
		query = dialect.From("teams").
			Select(t.getSelectFields()...).
			InnerJoin(goqu.T("team_members"), goqu.On(goqu.Ex{"teams.id": goqu.I("team_members.team_id")})).
			Where(ex)
	}

	sortDirection := pagination.AscSort

	var sortBy *pagination.FieldDescriptor
	if input.Sort != nil {
		sortDirection = input.Sort.getSortDirection()
		sortBy = input.Sort.getFieldDescriptor()
	}

	qBuilder, err := pagination.NewPaginatedQueryBuilder(
		input.PaginationOptions,
		&pagination.FieldDescriptor{Key: "id", Table: "teams", Col: "id"},
		sortBy,
		sortDirection,
	)
	if err != nil {
		tracing.RecordError(span, err, "failed to build query")
		return nil, err
	}

	rows, err := qBuilder.Execute(ctx, t.dbClient.getConnection(ctx), query)
	if err != nil {
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	defer rows.Close()

	// Scan rows
	results := []models.Team{}
	for rows.Next() {
		item, err := scanTeam(rows)
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

	result := TeamsResult{
		PageInfo: rows.GetPageInfo(),
		Teams:    results,
	}

	return &result, nil
}

func (t *teams) CreateTeam(ctx context.Context, team *models.Team) (*models.Team, error) {
	ctx, span := tracer.Start(ctx, "db.CreateTeam")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	timestamp := currentTime()

	sql, args, err := dialect.Insert("teams").
		Prepared(true).
		Rows(goqu.Record{
			"id":               newResourceID(),
			"version":          initialResourceVersion,
			"created_at":       timestamp,
			"updated_at":       timestamp,
			"name":             team.Name,
			"description":      team.Description,
			"scim_external_id": nullableString(team.SCIMExternalID),
		}).
		Returning(teamFieldList...).ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	createdTeam, err := scanTeam(t.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				tracing.RecordError(span, nil, "team with name %s already exists", team.Name)
				return nil, errors.New(errors.EConflict, "team with name %s already exists", team.Name)
			}
		}
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	return createdTeam, nil
}

func (t *teams) UpdateTeam(ctx context.Context, team *models.Team) (*models.Team, error) {
	ctx, span := tracer.Start(ctx, "db.UpdateTeam")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	timestamp := currentTime()

	sql, args, err := dialect.Update("teams").
		Prepared(true).
		Set(
			goqu.Record{
				"version":          goqu.L("? + ?", goqu.C("version"), 1),
				"updated_at":       timestamp,
				"description":      team.Description,
				"scim_external_id": nullableString(team.SCIMExternalID),
			},
		).Where(goqu.Ex{"id": team.Metadata.ID, "version": team.Metadata.Version}).Returning(teamFieldList...).ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	updatedTeam, err := scanTeam(t.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return nil, ErrOptimisticLockError
		}
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	return updatedTeam, nil
}

func (t *teams) DeleteTeam(ctx context.Context, team *models.Team) error {
	ctx, span := tracer.Start(ctx, "db.DeleteTeam")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	sql, args, err := dialect.Delete("teams").
		Prepared(true).
		Where(
			goqu.Ex{
				"id":      team.Metadata.ID,
				"version": team.Metadata.Version,
			},
		).Returning(teamFieldList...).ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return err
	}

	_, err = scanTeam(t.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err == pgx.ErrNoRows {
		tracing.RecordError(span, err, "optimistic lock error")
		return ErrOptimisticLockError
	}

	return nil
}

func (t *teams) getTeam(ctx context.Context, exp goqu.Ex) (*models.Team, error) {
	query := dialect.From(goqu.T("teams")).
		Prepared(true).
		Select(teamFieldList...).
		Where(exp)

	sql, args, err := query.ToSQL()
	if err != nil {
		return nil, err
	}

	team, err := scanTeam(t.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return team, nil
}

func (t *teams) getSelectFields() []interface{} {
	selectFields := []interface{}{}
	for _, field := range teamFieldList {
		selectFields = append(selectFields, fmt.Sprintf("teams.%s", field))
	}

	return selectFields
}

func scanTeam(row scanner) (*models.Team, error) {
	var scimExternalID sql.NullString
	team := &models.Team{}

	fields := []interface{}{
		&team.Metadata.ID,
		&team.Metadata.CreationTimestamp,
		&team.Metadata.LastUpdatedTimestamp,
		&team.Metadata.Version,
		&team.Name,
		&team.Description,
		&scimExternalID,
	}

	err := row.Scan(fields...)
	if err != nil {
		return nil, err
	}

	if scimExternalID.Valid {
		team.SCIMExternalID = scimExternalID.String
	}

	return team, nil
}
