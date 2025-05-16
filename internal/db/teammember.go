package db

//go:generate go tool mockery --name TeamMembers --inpackage --case underscore

import (
	"context"
	"fmt"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/jackc/pgx/v4"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// TeamMembers encapsulates the logic to access team members from the database
type TeamMembers interface {
	GetTeamMember(ctx context.Context, userID, teamID string) (*models.TeamMember, error)
	GetTeamMembers(ctx context.Context, input *GetTeamMembersInput) (*TeamMembersResult, error)
	AddUserToTeam(ctx context.Context, teamMember *models.TeamMember) (*models.TeamMember, error)
	UpdateTeamMember(ctx context.Context, teamMember *models.TeamMember) (*models.TeamMember, error)
	RemoveUserFromTeam(ctx context.Context, teamMember *models.TeamMember) error
}

// TeamMemberFilter contains the supported fields for filtering TeamMember resources
type TeamMemberFilter struct {
	UserID         *string
	TeamIDs        []string
	MaintainerOnly bool
}

// TeamMemberSortableField represents the fields that a team member can be sorted by
type TeamMemberSortableField string

// TeamMemberSortableField constants
const (
	UsernameAsc  TeamMemberSortableField = "USERNAME_ASC"
	UsernameDesc TeamMemberSortableField = "USERNAME_DESC"
)

func (tms TeamMemberSortableField) getFieldDescriptor() *pagination.FieldDescriptor {
	switch tms {
	// Placeholder for any future sorting field assignments.
	default:
		return nil
	}
}

func (tms TeamMemberSortableField) getSortDirection() pagination.SortDirection {
	if strings.HasSuffix(string(tms), "_DESC") {
		return pagination.DescSort
	}
	return pagination.AscSort
}

// GetTeamMembersInput is the input for listing team members
type GetTeamMembersInput struct {
	// Sort specifies the field to sort on and direction
	Sort *TeamMemberSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// Filter is used to filter the results
	Filter *TeamMemberFilter
}

// TeamMembersResult contains the response data and page information
type TeamMembersResult struct {
	PageInfo    *pagination.PageInfo
	TeamMembers []models.TeamMember
}

var teamMemberFieldList = append(metadataFieldList, "user_id", "team_id", "is_maintainer")

type teamMembers struct {
	dbClient *Client
}

// NewTeamMembers returns an instance of the TeamMembers interface
func NewTeamMembers(dbClient *Client) TeamMembers {
	return &teamMembers{dbClient: dbClient}
}

func (tm *teamMembers) GetTeamMemberByID(ctx context.Context,
	teamMemberID string) (*models.TeamMember, error) {
	ctx, span := tracer.Start(ctx, "db.GetTeamMemberByID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	return tm.getTeamMember(ctx, goqu.Ex{"team_members.id": teamMemberID})
}

func (tm *teamMembers) GetTeamMember(ctx context.Context,
	userID, teamID string) (*models.TeamMember, error) {
	ctx, span := tracer.Start(ctx, "db.GetTeamMember")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	return tm.getTeamMember(ctx, goqu.Ex{"team_members.user_id": userID, "team_members.team_id": teamID})
}

func (tm *teamMembers) GetTeamMembers(ctx context.Context, input *GetTeamMembersInput) (*TeamMembersResult, error) {
	ctx, span := tracer.Start(ctx, "db.GetTeamMembers")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	ex := goqu.Ex{}

	if input.Filter != nil {
		if input.Filter.UserID != nil {
			ex["team_members.user_id"] = *input.Filter.UserID
		}

		if input.Filter.TeamIDs != nil {
			ex["team_members.team_id"] = input.Filter.TeamIDs
		}

		if input.Filter.MaintainerOnly {
			ex["team_members.maintainer_only"] = true
		}
	}

	query := dialect.From(goqu.T("team_members")).
		Select(tm.getSelectFields()...).
		InnerJoin(goqu.T("teams"), goqu.On(goqu.I("team_members.team_id").Eq(goqu.I("teams.id")))).
		InnerJoin(goqu.T("users"), goqu.On(goqu.I("team_members.user_id").Eq(goqu.I("users.id")))).
		Where(ex)

	sortDirection := pagination.AscSort

	var sortBy *pagination.FieldDescriptor
	if input.Sort != nil {
		sortDirection = input.Sort.getSortDirection()
		sortBy = input.Sort.getFieldDescriptor()
	}

	qBuilder, err := pagination.NewPaginatedQueryBuilder(
		input.PaginationOptions,
		&pagination.FieldDescriptor{Key: "id", Table: "team_members", Col: "id"},
		pagination.WithSortByField(sortBy, sortDirection),
	)
	if err != nil {
		tracing.RecordError(span, err, "failed to build query")
		return nil, err
	}

	rows, err := qBuilder.Execute(ctx, tm.dbClient.getConnection(ctx), query)
	if err != nil {
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	defer rows.Close()

	// Scan rows
	results := []models.TeamMember{}
	for rows.Next() {
		item, err := scanTeamMember(rows)
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

	result := TeamMembersResult{
		PageInfo:    rows.GetPageInfo(),
		TeamMembers: results,
	}

	return &result, nil
}

func (tm *teamMembers) AddUserToTeam(ctx context.Context, teamMember *models.TeamMember) (*models.TeamMember, error) {
	ctx, span := tracer.Start(ctx, "db.AddUserToTeam")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	timestamp := currentTime()

	sql, args, err := dialect.From("team_members").
		Prepared(true).
		With("team_members",
			dialect.Insert("team_members").
				Rows(goqu.Record{
					"id":            newResourceID(),
					"version":       initialResourceVersion,
					"created_at":    timestamp,
					"updated_at":    timestamp,
					"user_id":       teamMember.UserID,
					"team_id":       teamMember.TeamID,
					"is_maintainer": teamMember.IsMaintainer,
				}).
				Returning("*"),
		).Select(tm.getSelectFields()...).
		InnerJoin(goqu.T("teams"), goqu.On(goqu.I("team_members.team_id").Eq(goqu.I("teams.id")))).
		InnerJoin(goqu.T("users"), goqu.On(goqu.I("team_members.user_id").Eq(goqu.I("users.id")))).
		ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	createdTeamMember, err := scanTeamMember(tm.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {

				// Don't have the username or team name, so need to fetch them for printing the error message.
				username := ""
				userRecord, uErr := tm.dbClient.Users.GetUserByID(ctx, teamMember.UserID)
				if uErr != nil {
					// Return original error but with user ID.
					username = fmt.Sprintf("ID %s", teamMember.UserID)
				} else {
					username = userRecord.Username
				}

				teamName := ""
				teamRecord, tErr := tm.dbClient.Teams.GetTeamByID(ctx, teamMember.TeamID)
				if tErr != nil {
					// Return the original error but with the team ID.
					teamName = fmt.Sprintf("ID %s", teamMember.TeamID)
				} else {
					teamName = teamRecord.Name
				}

				tracing.RecordError(span, nil,
					"team member of user %s in team %s already exists", username, teamName)
				return nil, errors.New(fmt.Sprintf("team member of user %s in team %s already exists", username, teamName), errors.WithErrorCode(errors.EConflict))
			}
		}
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	return createdTeamMember, nil
}

func (tm *teamMembers) UpdateTeamMember(ctx context.Context, teamMember *models.TeamMember) (*models.TeamMember, error) {
	ctx, span := tracer.Start(ctx, "db.UpdateTeamMember")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	timestamp := currentTime()

	sql, args, err := dialect.From("team_members").
		Prepared(true).
		With("team_members",
			dialect.Update("team_members").
				Set(
					goqu.Record{
						"version":       goqu.L("? + ?", goqu.C("version"), 1),
						"updated_at":    timestamp,
						"is_maintainer": teamMember.IsMaintainer,
					},
				).Where(goqu.Ex{"id": teamMember.Metadata.ID, "version": teamMember.Metadata.Version}).
				Returning("*"),
		).Select(tm.getSelectFields()...).
		InnerJoin(goqu.T("teams"), goqu.On(goqu.I("team_members.team_id").Eq(goqu.I("teams.id")))).
		InnerJoin(goqu.T("users"), goqu.On(goqu.I("team_members.user_id").Eq(goqu.I("users.id")))).
		ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	updatedTeamMember, err := scanTeamMember(tm.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))

	if err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return nil, ErrOptimisticLockError
		}
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	return updatedTeamMember, nil
}

func (tm *teamMembers) RemoveUserFromTeam(ctx context.Context, teamMember *models.TeamMember) error {
	ctx, span := tracer.Start(ctx, "db.RemoveUserFromTeam")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	sql, args, err := dialect.From("team_members").
		Prepared(true).
		With("team_members",
			dialect.Delete("team_members").
				Where(
					goqu.Ex{
						"id":      teamMember.Metadata.ID,
						"version": teamMember.Metadata.Version,
					},
				).Returning("*"),
		).Select(tm.getSelectFields()...).
		InnerJoin(goqu.T("teams"), goqu.On(goqu.I("team_members.team_id").Eq(goqu.I("teams.id")))).
		InnerJoin(goqu.T("users"), goqu.On(goqu.I("team_members.user_id").Eq(goqu.I("users.id")))).
		ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return err
	}

	_, err = scanTeamMember(tm.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return ErrOptimisticLockError
		}
		tracing.RecordError(span, err, "failed to execute query")
		return err
	}

	return nil
}

func (tm *teamMembers) getTeamMember(ctx context.Context, exp exp.Expression) (*models.TeamMember, error) {
	ctx, span := tracer.Start(ctx, "db.getTeamMember")
	defer span.End()

	query := dialect.From(goqu.T("team_members")).
		Prepared(true).
		Select(tm.getSelectFields()...).
		InnerJoin(goqu.T("teams"), goqu.On(goqu.I("team_members.team_id").Eq(goqu.I("teams.id")))).
		InnerJoin(goqu.T("users"), goqu.On(goqu.I("team_members.user_id").Eq(goqu.I("users.id")))).
		Where(exp)

	sql, args, err := query.ToSQL()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	teamMember, err := scanTeamMember(tm.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
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

	return teamMember, nil
}

func (tm *teamMembers) getSelectFields() []interface{} {
	selectFields := []interface{}{}
	for _, field := range teamMemberFieldList {
		selectFields = append(selectFields, fmt.Sprintf("team_members.%s", field))
	}

	selectFields = append(selectFields, "teams.name", "users.username")

	return selectFields
}

func scanTeamMember(row scanner) (*models.TeamMember, error) {
	var teamName, userName string
	teamMember := &models.TeamMember{}

	fields := []interface{}{
		&teamMember.Metadata.ID,
		&teamMember.Metadata.CreationTimestamp,
		&teamMember.Metadata.LastUpdatedTimestamp,
		&teamMember.Metadata.Version,
		&teamMember.UserID,
		&teamMember.TeamID,
		&teamMember.IsMaintainer,
		&teamName,
		&userName,
	}

	err := row.Scan(fields...)
	if err != nil {
		return nil, err
	}

	teamMember.Metadata.TRN = types.TeamMemberModelType.BuildTRN(teamName, userName)

	return teamMember, nil
}
