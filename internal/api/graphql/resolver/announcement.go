package resolver

import (
	"context"
	"strconv"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/announcement"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"

	graphql "github.com/graph-gophers/graphql-go"
)

// AnnouncementConnectionQueryArgs contains the arguments for the announcements query
type AnnouncementConnectionQueryArgs struct {
	ConnectionQueryArgs
	Active *bool
}

/* Announcement Query Resolvers */

// AnnouncementEdgeResolver resolves announcement edges
type AnnouncementEdgeResolver struct {
	edge Edge
}

// Cursor returns an opaque cursor
func (r *AnnouncementEdgeResolver) Cursor() (string, error) {
	announcement, ok := r.edge.Node.(models.Announcement)
	if !ok {
		return "", errors.New("Failed to convert node type")
	}
	cursor, err := r.edge.CursorFunc(&announcement)
	return *cursor, err
}

// Node returns an announcement node
func (r *AnnouncementEdgeResolver) Node() (*AnnouncementResolver, error) {
	announcement, ok := r.edge.Node.(models.Announcement)
	if !ok {
		return nil, errors.New("Failed to convert node type")
	}

	return &AnnouncementResolver{announcement: &announcement}, nil
}

// AnnouncementConnectionResolver resolves an announcement connection
type AnnouncementConnectionResolver struct {
	connection Connection
}

// NewAnnouncementConnectionResolver creates a new AnnouncementConnectionResolver
func NewAnnouncementConnectionResolver(ctx context.Context, input *announcement.GetAnnouncementsInput) (*AnnouncementConnectionResolver, error) {
	service := getServiceCatalog(ctx).AnnouncementService

	result, err := service.GetAnnouncements(ctx, input)
	if err != nil {
		return nil, err
	}

	announcements := result.Announcements

	// Create edges
	edges := make([]Edge, len(announcements))
	for i, announcement := range announcements {
		edges[i] = Edge{CursorFunc: result.PageInfo.Cursor, Node: announcement}
	}

	pageInfo := PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
	}

	if len(announcements) > 0 {
		var err error
		pageInfo.StartCursor, err = result.PageInfo.Cursor(&announcements[0])
		if err != nil {
			return nil, err
		}

		pageInfo.EndCursor, err = result.PageInfo.Cursor(&announcements[len(edges)-1])
		if err != nil {
			return nil, err
		}
	}

	connection := Connection{
		TotalCount: result.PageInfo.TotalCount,
		PageInfo:   pageInfo,
		Edges:      edges,
	}

	return &AnnouncementConnectionResolver{connection: connection}, nil
}

// TotalCount returns the total result count for the connection
func (r *AnnouncementConnectionResolver) TotalCount() int32 {
	return r.connection.TotalCount
}

// PageInfo returns the page information for the connection
func (r *AnnouncementConnectionResolver) PageInfo() *PageInfoResolver {
	return &PageInfoResolver{pageInfo: r.connection.PageInfo}
}

// Edges returns the announcement edges for the connection
func (r *AnnouncementConnectionResolver) Edges() *[]*AnnouncementEdgeResolver {
	resolvers := make([]*AnnouncementEdgeResolver, len(r.connection.Edges))

	for i, edge := range r.connection.Edges {
		resolvers[i] = &AnnouncementEdgeResolver{edge: edge}
	}

	return &resolvers
}

// AnnouncementResolver resolves an Announcement
type AnnouncementResolver struct {
	announcement *models.Announcement
}

// ID resolver
func (r *AnnouncementResolver) ID() graphql.ID {
	return graphql.ID(r.announcement.GetGlobalID())
}

// Message resolver
func (r *AnnouncementResolver) Message() string {
	return r.announcement.Message
}

// StartTime resolver
func (r *AnnouncementResolver) StartTime() graphql.Time {
	return graphql.Time{Time: r.announcement.StartTime}
}

// EndTime resolver
func (r *AnnouncementResolver) EndTime() *graphql.Time {
	if r.announcement.EndTime == nil {
		return nil
	}
	return &graphql.Time{Time: *r.announcement.EndTime}
}

// CreatedBy resolver
func (r *AnnouncementResolver) CreatedBy() string {
	return r.announcement.CreatedBy
}

// Type resolver
func (r *AnnouncementResolver) Type() models.AnnouncementType {
	return r.announcement.Type
}

// Dismissible resolver
func (r *AnnouncementResolver) Dismissible() bool {
	return r.announcement.Dismissible
}

// Active resolver
func (r *AnnouncementResolver) Active() bool {
	return r.announcement.IsActive()
}

// Expired resolver
func (r *AnnouncementResolver) Expired() bool {
	return r.announcement.IsExpired()
}

// Metadata resolver
func (r *AnnouncementResolver) Metadata() *MetadataResolver {
	return &MetadataResolver{metadata: &r.announcement.Metadata}
}

/* Announcement Query Functions */

func announcementsQuery(ctx context.Context, args *AnnouncementConnectionQueryArgs) (*AnnouncementConnectionResolver, error) {
	if err := args.Validate(); err != nil {
		return nil, err
	}

	input := announcement.GetAnnouncementsInput{
		PaginationOptions: &pagination.Options{First: args.First, Last: args.Last, After: args.After, Before: args.Before},
		Active:            args.Active,
	}

	if args.Sort != nil {
		sort := db.AnnouncementSortableField(*args.Sort)
		input.Sort = &sort
	}

	return NewAnnouncementConnectionResolver(ctx, &input)
}

/* Announcement Mutation Resolvers */

// AnnouncementMutationPayload is the response payload for announcement mutations
type AnnouncementMutationPayload struct {
	ClientMutationID *string
	Announcement     *models.Announcement
	Problems         []Problem
}

// AnnouncementMutationPayloadResolver resolves AnnouncementMutationPayload
type AnnouncementMutationPayloadResolver struct {
	AnnouncementMutationPayload
}

// Announcement field resolver
func (r *AnnouncementMutationPayloadResolver) Announcement() *AnnouncementResolver {
	if r.AnnouncementMutationPayload.Announcement == nil {
		return nil
	}

	return &AnnouncementResolver{announcement: r.AnnouncementMutationPayload.Announcement}
}

// CreateAnnouncementInput contains the input for creating a new announcement
type CreateAnnouncementInput struct {
	ClientMutationID *string
	Message          string
	StartTime        *graphql.Time
	EndTime          *graphql.Time
	Type             models.AnnouncementType
	Dismissible      bool
}

// UpdateAnnouncementInput contains the input for updating an announcement
type UpdateAnnouncementInput struct {
	ClientMutationID *string
	ID               string
	Message          *string
	StartTime        *graphql.Time
	EndTime          *graphql.Time
	Type             *models.AnnouncementType
	Dismissible      *bool
	Metadata         *MetadataInput
}

// DeleteAnnouncementInput contains the input for deleting an announcement
type DeleteAnnouncementInput struct {
	ClientMutationID *string
	ID               string
	Metadata         *MetadataInput
}

func handleAnnouncementMutationProblem(e error, clientMutationID *string) (*AnnouncementMutationPayloadResolver, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}

	payload := AnnouncementMutationPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}
	return &AnnouncementMutationPayloadResolver{AnnouncementMutationPayload: payload}, nil
}

func createAnnouncementMutation(ctx context.Context, input *CreateAnnouncementInput) (*AnnouncementMutationPayloadResolver, error) {
	service := getServiceCatalog(ctx).AnnouncementService

	createInput := &announcement.CreateAnnouncementInput{
		Message:     input.Message,
		Type:        input.Type,
		Dismissible: input.Dismissible,
	}

	if input.StartTime != nil {
		createInput.StartTime = &input.StartTime.Time
	}

	if input.EndTime != nil {
		createInput.EndTime = &input.EndTime.Time
	}

	createdAnnouncement, err := service.CreateAnnouncement(ctx, createInput)
	if err != nil {
		return nil, err
	}

	payload := AnnouncementMutationPayload{ClientMutationID: input.ClientMutationID, Announcement: createdAnnouncement, Problems: []Problem{}}
	return &AnnouncementMutationPayloadResolver{AnnouncementMutationPayload: payload}, nil
}

func updateAnnouncementMutation(ctx context.Context, input *UpdateAnnouncementInput) (*AnnouncementMutationPayloadResolver, error) {
	service := getServiceCatalog(ctx).AnnouncementService

	id := gid.FromGlobalID(string(input.ID))

	updateInput := &announcement.UpdateAnnouncementInput{
		ID:          id,
		Message:     input.Message,
		Type:        input.Type,
		Dismissible: input.Dismissible,
	}

	if input.StartTime != nil {
		startTime := input.StartTime.Time
		updateInput.StartTime = &startTime
	}

	if input.EndTime != nil {
		updateInput.EndTime = &input.EndTime.Time
	} else {
		updateInput.EndTime = nil
	}

	// Check if resource version is specified
	if input.Metadata != nil {
		v, cErr := strconv.Atoi(input.Metadata.Version)
		if cErr != nil {
			return nil, cErr
		}

		updateInput.MetadataVersion = &v
	}

	updatedAnnouncement, err := service.UpdateAnnouncement(ctx, updateInput)
	if err != nil {
		return nil, err
	}

	payload := AnnouncementMutationPayload{ClientMutationID: input.ClientMutationID, Announcement: updatedAnnouncement, Problems: []Problem{}}
	return &AnnouncementMutationPayloadResolver{AnnouncementMutationPayload: payload}, nil
}

func deleteAnnouncementMutation(ctx context.Context, input *DeleteAnnouncementInput) (*AnnouncementMutationPayloadResolver, error) {
	service := getServiceCatalog(ctx).AnnouncementService

	id := gid.FromGlobalID(string(input.ID))

	// Get the announcement before deleting for the response
	gotAnnouncement, err := service.GetAnnouncementByID(ctx, id)
	if err != nil {
		return nil, err
	}

	deleteInput := &announcement.DeleteAnnouncementInput{
		ID: id,
	}

	// Check if resource version is specified
	if input.Metadata != nil {
		v, cErr := strconv.Atoi(input.Metadata.Version)
		if cErr != nil {
			return nil, cErr
		}

		deleteInput.MetadataVersion = &v
	}

	err = service.DeleteAnnouncement(ctx, deleteInput)
	if err != nil {
		return nil, err
	}

	payload := AnnouncementMutationPayload{ClientMutationID: input.ClientMutationID, Announcement: gotAnnouncement, Problems: []Problem{}}
	return &AnnouncementMutationPayloadResolver{AnnouncementMutationPayload: payload}, nil
}
