package resolver

import (
	"context"
	"strconv"

	"github.com/graph-gophers/dataloader"
	graphql "github.com/graph-gophers/graphql-go"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/loader"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/team"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

/* Team Query Resolvers */

// TeamConnectionQueryArgs are used to query a team connection
type TeamConnectionQueryArgs struct {
	ConnectionQueryArgs
	Search *string
}

// TeamQueryArgs are used to query a single team
type TeamQueryArgs struct {
	Name string
}

// TeamEdgeResolver resolves team edges
type TeamEdgeResolver struct {
	edge Edge
}

// Cursor returns an opaque cursor
func (r *TeamEdgeResolver) Cursor() (string, error) {
	team, ok := r.edge.Node.(models.Team)
	if !ok {
		return "", errors.New(errors.EInternal, "Failed to convert node type")
	}
	cursor, err := r.edge.CursorFunc(&team)
	return *cursor, err
}

// Node returns a team node
func (r *TeamEdgeResolver) Node() (*TeamResolver, error) {
	team, ok := r.edge.Node.(models.Team)
	if !ok {
		return nil, errors.New(errors.EInternal, "Failed to convert node type")
	}

	return &TeamResolver{team: &team}, nil
}

// TeamConnectionResolver resolves a team connection
type TeamConnectionResolver struct {
	connection Connection
}

// NewTeamConnectionResolver creates a new TeamConnectionResolver
func NewTeamConnectionResolver(ctx context.Context, input *team.GetTeamsInput) (*TeamConnectionResolver, error) {
	teamService := getTeamService(ctx)

	result, err := teamService.GetTeams(ctx, input)
	if err != nil {
		return nil, err
	}

	teams := result.Teams

	// Create edges
	edges := make([]Edge, len(teams))
	for i, team := range teams {
		edges[i] = Edge{CursorFunc: result.PageInfo.Cursor, Node: team}
	}

	pageInfo := PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
	}

	if len(teams) > 0 {
		var err error
		pageInfo.StartCursor, err = result.PageInfo.Cursor(&teams[0])
		if err != nil {
			return nil, err
		}

		pageInfo.EndCursor, err = result.PageInfo.Cursor(&teams[len(edges)-1])
		if err != nil {
			return nil, err
		}
	}

	connection := Connection{
		TotalCount: result.PageInfo.TotalCount,
		PageInfo:   pageInfo,
		Edges:      edges,
	}

	return &TeamConnectionResolver{connection: connection}, nil
}

// TotalCount returns the total result count for the connection
func (r *TeamConnectionResolver) TotalCount() int32 {
	return r.connection.TotalCount
}

// PageInfo returns the connection page information
func (r *TeamConnectionResolver) PageInfo() *PageInfoResolver {
	return &PageInfoResolver{pageInfo: r.connection.PageInfo}
}

// Edges returns the connection edges
func (r *TeamConnectionResolver) Edges() *[]*TeamEdgeResolver {
	resolvers := make([]*TeamEdgeResolver, len(r.connection.Edges))
	for i, edge := range r.connection.Edges {
		resolvers[i] = &TeamEdgeResolver{edge: edge}
	}
	return &resolvers
}

// TeamResolver resolves a team resource
type TeamResolver struct {
	team *models.Team
}

// ID resolver
func (r *TeamResolver) ID() graphql.ID {
	return graphql.ID(gid.ToGlobalID(gid.TeamType, r.team.Metadata.ID))
}

// Name resolver
func (r *TeamResolver) Name() string {
	return r.team.Name
}

// Description resolver
func (r *TeamResolver) Description() string {
	return r.team.Description
}

// Metadata resolver
func (r *TeamResolver) Metadata() *MetadataResolver {
	return &MetadataResolver{metadata: &r.team.Metadata}
}

// SCIMExternalID resolver
func (r *TeamResolver) SCIMExternalID() *string {
	return &r.team.SCIMExternalID
}

// Members resolver
func (r *TeamResolver) Members(ctx context.Context, args *ConnectionQueryArgs) (*TeamMemberConnectionResolver, error) {
	if err := args.Validate(); err != nil {
		return nil, err
	}

	input := team.GetTeamMembersInput{
		PaginationOptions: &pagination.Options{First: args.First, Last: args.Last, After: args.After, Before: args.Before},
		TeamID:            &r.team.Metadata.ID,
	}

	if args.Sort != nil {
		sort := db.TeamMemberSortableField(*args.Sort)
		input.Sort = &sort
	}

	return NewTeamMemberConnectionResolver(ctx, &input)
}

func teamQuery(ctx context.Context, args *TeamQueryArgs) (*TeamResolver, error) {
	teamService := getTeamService(ctx)

	team, err := teamService.GetTeamByName(ctx, args.Name)
	if err != nil {
		if errors.ErrorCode(err) == errors.ENotFound {
			return nil, nil
		}

		return nil, err
	}

	return &TeamResolver{team: team}, nil
}

func teamsQuery(ctx context.Context, args *TeamConnectionQueryArgs) (*TeamConnectionResolver, error) {
	if err := args.Validate(); err != nil {
		return nil, err
	}

	input := team.GetTeamsInput{
		PaginationOptions: &pagination.Options{First: args.First, Last: args.Last, After: args.After, Before: args.Before},
		TeamNamePrefix:    args.Search,
	}

	if args.Sort != nil {
		sort := db.TeamSortableField(*args.Sort)
		input.Sort = &sort
	}

	return NewTeamConnectionResolver(ctx, &input)
}

/* Team Mutation Resolvers */

// TeamMutationPayload is the response payload for a team mutation
type TeamMutationPayload struct {
	ClientMutationID *string
	Team             *models.Team
	Problems         []Problem
}

// TeamMutationPayloadResolver resolves a TeamMutationPayload
type TeamMutationPayloadResolver struct {
	TeamMutationPayload
}

// Team field resolver
func (r *TeamMutationPayloadResolver) Team() *TeamResolver {
	if r.TeamMutationPayload.Team == nil {
		return nil
	}

	return &TeamResolver{team: r.TeamMutationPayload.Team}
}

// CreateTeamInput contains the input for creating a new team
type CreateTeamInput struct {
	ClientMutationID *string
	Name             string
	Description      string
}

// UpdateTeamInput contains the input for updating a team
type UpdateTeamInput struct {
	ClientMutationID *string
	Metadata         *MetadataInput
	Name             string
	Description      string
}

// DeleteTeamInput contains the input for deleting a team
type DeleteTeamInput struct {
	ClientMutationID *string
	Metadata         *MetadataInput
	Name             string
}

func handleTeamMutationProblem(e error, clientMutationID *string) (*TeamMutationPayloadResolver, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}
	payload := TeamMutationPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}
	return &TeamMutationPayloadResolver{TeamMutationPayload: payload}, nil
}

func createTeamMutation(ctx context.Context, input *CreateTeamInput) (*TeamMutationPayloadResolver, error) {
	teamService := getTeamService(ctx)

	toCreate := &team.CreateTeamInput{
		Name:        input.Name,
		Description: input.Description,
	}

	createdTeam, err := teamService.CreateTeam(ctx, toCreate)
	if err != nil {
		return nil, err
	}

	payload := TeamMutationPayload{ClientMutationID: input.ClientMutationID, Team: createdTeam, Problems: []Problem{}}
	return &TeamMutationPayloadResolver{TeamMutationPayload: payload}, nil
}

func updateTeamMutation(ctx context.Context, input *UpdateTeamInput) (*TeamMutationPayloadResolver, error) {
	teamService := getTeamService(ctx)

	toUpdate := &team.UpdateTeamInput{
		Name:        input.Name,
		Description: &input.Description,
	}

	// Check if resource version is specified
	if input.Metadata != nil {
		v, cErr := strconv.Atoi(input.Metadata.Version)
		if cErr != nil {
			return nil, cErr
		}

		toUpdate.MetadataVersion = &v
	}

	team, err := teamService.UpdateTeam(ctx, toUpdate)
	if err != nil {
		return nil, err
	}

	payload := TeamMutationPayload{ClientMutationID: input.ClientMutationID, Team: team, Problems: []Problem{}}
	return &TeamMutationPayloadResolver{TeamMutationPayload: payload}, nil
}

func deleteTeamMutation(ctx context.Context, input *DeleteTeamInput) (*TeamMutationPayloadResolver, error) {
	teamService := getTeamService(ctx)

	gotTeam, err := teamService.GetTeamByName(ctx, input.Name)
	if err != nil {
		return nil, err
	}

	// Check if resource version is specified
	if input.Metadata != nil {
		v, err := strconv.Atoi(input.Metadata.Version)
		if err != nil {
			return nil, err
		}

		gotTeam.Metadata.Version = v
	}

	toDelete := &team.DeleteTeamInput{
		Team: gotTeam,
	}

	if err := teamService.DeleteTeam(ctx, toDelete); err != nil {
		return nil, err
	}

	payload := TeamMutationPayload{ClientMutationID: input.ClientMutationID, Team: gotTeam, Problems: []Problem{}}
	return &TeamMutationPayloadResolver{TeamMutationPayload: payload}, nil
}

/* Team loader */

const teamLoaderKey = "team"

// RegisterTeamLoader registers a team loader function
func RegisterTeamLoader(collection *loader.Collection) {
	collection.Register(teamLoaderKey, teamBatchFunc)
}

func loadTeam(ctx context.Context, id string) (*models.Team, error) {
	ldr, err := loader.Extract(ctx, teamLoaderKey)
	if err != nil {
		return nil, err
	}

	data, err := ldr.Load(ctx, dataloader.StringKey(id))()
	if err != nil {
		return nil, err
	}

	team, ok := data.(models.Team)
	if !ok {
		return nil, errors.New(errors.EInternal, "Wrong type")
	}

	return &team, nil
}

func teamBatchFunc(ctx context.Context, ids []string) (loader.DataBatch, error) {
	teams, err := getTeamService(ctx).GetTeamsByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	// Build map of results
	batch := loader.DataBatch{}
	for _, result := range teams {
		batch[result.Metadata.ID] = result
	}

	return batch, nil
}

/* Team Member Query Resolvers */

// TeamMemberEdgeResolver resolves team member edges
type TeamMemberEdgeResolver struct {
	edge Edge
}

// Cursor returns an opaque cursor
func (r *TeamMemberEdgeResolver) Cursor() (string, error) {
	teamMember, ok := r.edge.Node.(models.TeamMember)
	if !ok {
		return "", errors.New(errors.EInternal, "Failed to convert node type")
	}
	cursor, err := r.edge.CursorFunc(&teamMember)
	return *cursor, err
}

// Node returns a team member node
func (r *TeamMemberEdgeResolver) Node() (*TeamMemberResolver, error) {
	teamMember, ok := r.edge.Node.(models.TeamMember)
	if !ok {
		return nil, errors.New(errors.EInternal, "Failed to convert node type")
	}

	return &TeamMemberResolver{teamMember: &teamMember}, nil
}

// TeamMemberConnectionResolver resolves a team member connection
type TeamMemberConnectionResolver struct {
	connection Connection
}

// NewTeamMemberConnectionResolver creates a new TeamMemberConnectionResolver
func NewTeamMemberConnectionResolver(ctx context.Context,
	input *team.GetTeamMembersInput,
) (*TeamMemberConnectionResolver, error) {
	result, err := getTeamService(ctx).GetTeamMembers(ctx,
		&db.GetTeamMembersInput{Filter: &db.TeamMemberFilter{TeamIDs: []string{*input.TeamID}}})
	if err != nil {
		return nil, err
	}

	teamMembers := result.TeamMembers

	// Create edges
	edges := make([]Edge, len(teamMembers))
	for i, teamMember := range teamMembers {
		edges[i] = Edge{CursorFunc: result.PageInfo.Cursor, Node: teamMember}
	}

	pageInfo := PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
	}

	if len(teamMembers) > 0 {
		var err error
		pageInfo.StartCursor, err = result.PageInfo.Cursor(&teamMembers[0])
		if err != nil {
			return nil, err
		}

		pageInfo.EndCursor, err = result.PageInfo.Cursor(&teamMembers[len(edges)-1])
		if err != nil {
			return nil, err
		}
	}

	connection := Connection{
		TotalCount: result.PageInfo.TotalCount,
		PageInfo:   pageInfo,
		Edges:      edges,
	}

	return &TeamMemberConnectionResolver{connection: connection}, nil
}

// TotalCount returns the total result count for the connection
func (r *TeamMemberConnectionResolver) TotalCount() int32 {
	return r.connection.TotalCount
}

// PageInfo returns the connection page information
func (r *TeamMemberConnectionResolver) PageInfo() *PageInfoResolver {
	return &PageInfoResolver{pageInfo: r.connection.PageInfo}
}

// Edges returns the connection edges
func (r *TeamMemberConnectionResolver) Edges() *[]*TeamMemberEdgeResolver {
	resolvers := make([]*TeamMemberEdgeResolver, len(r.connection.Edges))
	for i, edge := range r.connection.Edges {
		resolvers[i] = &TeamMemberEdgeResolver{edge: edge}
	}
	return &resolvers
}

// TeamMemberResolver resolves a team member resource
type TeamMemberResolver struct {
	teamMember *models.TeamMember
}

// ID resolver
func (r *TeamMemberResolver) ID() graphql.ID {
	return graphql.ID(gid.ToGlobalID(gid.TeamMemberType, r.teamMember.Metadata.ID))
}

// User resolver
func (r *TeamMemberResolver) User(ctx context.Context) (*UserResolver, error) {
	user, err := loadUser(ctx, r.teamMember.UserID)
	if err != nil {
		return nil, err
	}

	return &UserResolver{user: user}, nil
}

// Team resolver
func (r *TeamMemberResolver) Team(ctx context.Context) (*TeamResolver, error) {
	team, err := loadTeam(ctx, r.teamMember.TeamID)
	if err != nil {
		return nil, err
	}

	return &TeamResolver{team: team}, nil
}

// IsMaintainer resolver
func (r *TeamMemberResolver) IsMaintainer() bool {
	return r.teamMember.IsMaintainer
}

// Metadata resolver
func (r *TeamMemberResolver) Metadata() *MetadataResolver {
	return &MetadataResolver{metadata: &r.teamMember.Metadata}
}

/* Team Member Mutation Resolvers */

// TeamMemberMutationPayload is the response payload for a team member mutation
type TeamMemberMutationPayload struct {
	ClientMutationID *string
	TeamMember       *models.TeamMember
	Problems         []Problem
}

// TeamMemberMutationPayloadResolver resolves a TeamMemberMutationPayload
type TeamMemberMutationPayloadResolver struct {
	TeamMemberMutationPayload
}

// TeamMember field resolver
func (r *TeamMemberMutationPayloadResolver) TeamMember() *TeamMemberResolver {
	if r.TeamMemberMutationPayload.TeamMember == nil {
		return nil
	}

	return &TeamMemberResolver{teamMember: r.TeamMemberMutationPayload.TeamMember}
}

// AddUserToTeamInput is the input for adding a user to a team.
type AddUserToTeamInput struct {
	ClientMutationID *string
	Username         string
	TeamName         string
	IsMaintainer     bool
}

// UpdateTeamMemberInput is the input for updating a team member
type UpdateTeamMemberInput struct {
	ClientMutationID *string
	Metadata         *MetadataInput
	Username         string
	TeamName         string
	IsMaintainer     bool
}

// RemoveUserFromTeamInput is the input for removing a user from a team.
type RemoveUserFromTeamInput struct {
	ClientMutationID *string
	Metadata         *MetadataInput
	Username         string
	TeamName         string
}

func handleTeamMemberMutationProblem(e error, clientMutationID *string) (*TeamMemberMutationPayloadResolver, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}
	payload := TeamMemberMutationPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}
	return &TeamMemberMutationPayloadResolver{TeamMemberMutationPayload: payload}, nil
}

func addUserToTeamMutation(ctx context.Context, input *AddUserToTeamInput) (*TeamMemberMutationPayloadResolver, error) {
	createOptions := &team.AddUserToTeamInput{
		TeamName:     input.TeamName,
		Username:     input.Username,
		IsMaintainer: input.IsMaintainer,
	}

	teamMember, err := getTeamService(ctx).AddUserToTeam(ctx, createOptions)
	if err != nil {
		return nil, err
	}

	payload := TeamMemberMutationPayload{ClientMutationID: input.ClientMutationID, TeamMember: teamMember, Problems: []Problem{}}
	return &TeamMemberMutationPayloadResolver{TeamMemberMutationPayload: payload}, nil
}

func updateTeamMemberMutation(ctx context.Context, input *UpdateTeamMemberInput) (*TeamMemberMutationPayloadResolver, error) {
	service := getTeamService(ctx)

	toUpdate := &team.UpdateTeamMemberInput{
		TeamName:     input.TeamName,
		Username:     input.Username,
		IsMaintainer: input.IsMaintainer,
	}

	// Check if resource version is specified
	if input.Metadata != nil {
		v, cErr := strconv.Atoi(input.Metadata.Version)
		if cErr != nil {
			return nil, cErr
		}

		toUpdate.MetadataVersion = &v
	}

	teamMember, err := service.UpdateTeamMember(ctx, toUpdate)
	if err != nil {
		return nil, err
	}

	payload := TeamMemberMutationPayload{ClientMutationID: input.ClientMutationID, TeamMember: teamMember, Problems: []Problem{}}
	return &TeamMemberMutationPayloadResolver{TeamMemberMutationPayload: payload}, nil
}

func removeUserFromTeamMutation(ctx context.Context, input *RemoveUserFromTeamInput) (*TeamMemberMutationPayloadResolver, error) {
	service := getTeamService(ctx)

	teamMember, err := service.GetTeamMember(ctx, input.Username, input.TeamName)
	if err != nil {
		return nil, err
	}

	// Check if resource version is specified
	if input.Metadata != nil {
		v, cErr := strconv.Atoi(input.Metadata.Version)
		if cErr != nil {
			return nil, cErr
		}

		teamMember.Metadata.Version = v
	}

	toDelete := &team.RemoveUserFromTeamInput{
		TeamMember: teamMember,
	}

	if err = service.RemoveUserFromTeam(ctx, toDelete); err != nil {
		return nil, err
	}

	payload := TeamMemberMutationPayload{ClientMutationID: input.ClientMutationID, TeamMember: teamMember, Problems: []Problem{}}
	return &TeamMemberMutationPayloadResolver{TeamMemberMutationPayload: payload}, nil
}
