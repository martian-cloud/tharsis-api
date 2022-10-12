package db

//go:generate mockery --name TeamMembers --inpackage --case underscore

import (
	"context"
	"fmt"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/jackc/pgx/v4"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// TeamMembers encapsulates the logic to access team members from the database
type TeamMembers interface {
	GetTeamMemberByID(ctx context.Context, teamMemberID string) (*models.TeamMember, error)
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

func (tms TeamMemberSortableField) getFieldDescriptor() *fieldDescriptor {
	switch tms {
	// Placeholder for any future sorting field assignments.
	default:
		return nil
	}
}

func (tms TeamMemberSortableField) getSortDirection() SortDirection {
	if strings.HasSuffix(string(tms), "_DESC") {
		return DescSort
	}
	return AscSort
}

// GetTeamMembersInput is the input for listing team members
type GetTeamMembersInput struct {
	// Sort specifies the field to sort on and direction
	Sort *TeamMemberSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *PaginationOptions
	// Filter is used to filter the results
	Filter *TeamMemberFilter
}

// TeamMembersResult contains the response data and page information
type TeamMembersResult struct {
	PageInfo    *PageInfo
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
	return tm.getTeamMember(ctx, goqu.Ex{"team_members.id": teamMemberID})
}

func (tm *teamMembers) GetTeamMember(ctx context.Context,
	userID, teamID string) (*models.TeamMember, error) {
	return tm.getTeamMember(ctx, goqu.Ex{"team_members.user_id": userID, "team_members.team_id": teamID})
}

func (tm *teamMembers) GetTeamMembers(ctx context.Context, input *GetTeamMembersInput) (*TeamMembersResult, error) {
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

	query := dialect.From(goqu.T("team_members")).Select(teamMemberFieldList...).Where(ex)

	sortDirection := AscSort

	var sortBy *fieldDescriptor
	if input.Sort != nil {
		sortDirection = input.Sort.getSortDirection()
		sortBy = input.Sort.getFieldDescriptor()
	}

	qBuilder, err := newPaginatedQueryBuilder(
		input.PaginationOptions,
		&fieldDescriptor{key: "id", table: "team_members", col: "id"},
		sortBy,
		sortDirection,
		teamMemberFieldResolver,
	)
	if err != nil {
		return nil, err
	}

	rows, err := qBuilder.execute(ctx, tm.dbClient.getConnection(ctx), query)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	// Scan rows
	results := []models.TeamMember{}
	for rows.Next() {
		item, err := scanTeamMember(rows)
		if err != nil {
			return nil, err
		}

		results = append(results, *item)
	}

	if err := rows.finalize(&results); err != nil {
		return nil, err
	}

	result := TeamMembersResult{
		PageInfo:    rows.getPageInfo(),
		TeamMembers: results,
	}

	return &result, nil
}

func (tm *teamMembers) AddUserToTeam(ctx context.Context, teamMember *models.TeamMember) (*models.TeamMember, error) {
	timestamp := currentTime()

	sql, _, err := dialect.Insert("team_members").
		Rows(goqu.Record{
			"id":            newResourceID(),
			"version":       initialResourceVersion,
			"created_at":    timestamp,
			"updated_at":    timestamp,
			"user_id":       teamMember.UserID,
			"team_id":       teamMember.TeamID,
			"is_maintainer": teamMember.IsMaintainer,
		}).
		Returning(teamMemberFieldList...).ToSQL()
	if err != nil {
		return nil, err
	}

	createdTeamMember, err := scanTeamMember(tm.dbClient.getConnection(ctx).QueryRow(ctx, sql))
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

				return nil, errors.NewError(errors.EConflict,
					fmt.Sprintf("team member of user %s in team %s already exists", username, teamName))
			}
		}
		return nil, err
	}

	return createdTeamMember, nil
}

func (tm *teamMembers) UpdateTeamMember(ctx context.Context, teamMember *models.TeamMember) (*models.TeamMember, error) {
	timestamp := currentTime()

	sql, _, err := dialect.Update("team_members").Set(
		goqu.Record{
			"version":       goqu.L("? + ?", goqu.C("version"), 1),
			"updated_at":    timestamp,
			"is_maintainer": teamMember.IsMaintainer,
		},
	).Where(goqu.Ex{"id": teamMember.Metadata.ID, "version": teamMember.Metadata.Version}).
		Returning(teamMemberFieldList...).ToSQL()
	if err != nil {
		return nil, err
	}

	updatedTeamMember, err := scanTeamMember(tm.dbClient.getConnection(ctx).QueryRow(ctx, sql))

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrOptimisticLockError
		}
		return nil, err
	}

	return updatedTeamMember, nil
}

func (tm *teamMembers) RemoveUserFromTeam(ctx context.Context, teamMember *models.TeamMember) error {

	sql, _, err := dialect.Delete("team_members").Where(
		goqu.Ex{
			"id":      teamMember.Metadata.ID,
			"version": teamMember.Metadata.Version,
		},
	).Returning(teamMemberFieldList...).ToSQL()
	if err != nil {
		return err
	}

	_, err = scanTeamMember(tm.dbClient.getConnection(ctx).QueryRow(ctx, sql))
	if err != nil {
		if err == pgx.ErrNoRows {
			return ErrOptimisticLockError
		}
		return err
	}

	return nil
}

func (tm *teamMembers) getTeamMember(ctx context.Context, exp exp.Expression) (*models.TeamMember, error) {
	query := dialect.From(goqu.T("team_members")).
		Select(teamMemberFieldList...).Where(exp)

	sql, _, err := query.ToSQL()
	if err != nil {
		return nil, err
	}

	teamMember, err := scanTeamMember(tm.dbClient.getConnection(ctx).QueryRow(ctx, sql))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return teamMember, nil
}

func scanTeamMember(row scanner) (*models.TeamMember, error) {
	teamMember := &models.TeamMember{}

	fields := []interface{}{
		&teamMember.Metadata.ID,
		&teamMember.Metadata.CreationTimestamp,
		&teamMember.Metadata.LastUpdatedTimestamp,
		&teamMember.Metadata.Version,
		&teamMember.UserID,
		&teamMember.TeamID,
		&teamMember.IsMaintainer,
	}

	err := row.Scan(fields...)
	if err != nil {
		return nil, err
	}

	return teamMember, nil
}

func teamMemberFieldResolver(key string, model interface{}) (string, error) {
	teamMember, ok := model.(*models.TeamMember)
	if !ok {
		return "", errors.NewError(errors.EInternal, fmt.Sprintf("Expected team member type, got %T", model))
	}

	val, ok := metadataFieldResolver(key, &teamMember.Metadata)
	if !ok {
		return "", errors.NewError(errors.EInternal, fmt.Sprintf("Invalid field key requested %s", key))
	}

	return val, nil
}

// The End.
