package resolver

import (
	"context"
	"strconv"

	graphql "github.com/graph-gophers/graphql-go"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/email"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

/* EmailSuppression Query Resolvers */

// EmailSuppressionConnectionQueryArgs contains the arguments for the emailSuppressions query.
type EmailSuppressionConnectionQueryArgs struct {
	ConnectionQueryArgs
	Addresses *[]string
	Search    *string
}

// EmailSuppressionEdgeResolver resolves email suppression edges.
type EmailSuppressionEdgeResolver struct {
	edge Edge
}

// Cursor returns an opaque cursor.
func (r *EmailSuppressionEdgeResolver) Cursor() (string, error) {
	suppression, ok := r.edge.Node.(*models.EmailSuppression)
	if !ok {
		return "", errors.New("failed to convert node type")
	}
	cursor, err := r.edge.CursorFunc(suppression)
	return *cursor, err
}

// Node returns an email suppression node.
func (r *EmailSuppressionEdgeResolver) Node() (*EmailSuppressionResolver, error) {
	suppression, ok := r.edge.Node.(*models.EmailSuppression)
	if !ok {
		return nil, errors.New("failed to convert node type")
	}
	return &EmailSuppressionResolver{suppression: suppression}, nil
}

// EmailSuppressionConnectionResolver resolves an email suppression connection.
type EmailSuppressionConnectionResolver struct {
	connection Connection
}

// NewEmailSuppressionConnectionResolver creates a new EmailSuppressionConnectionResolver.
func NewEmailSuppressionConnectionResolver(ctx context.Context, input *db.GetEmailSuppressionsInput) (*EmailSuppressionConnectionResolver, error) {
	result, err := getServiceCatalog(ctx).EmailService.GetSuppressions(ctx, input)
	if err != nil {
		return nil, err
	}

	suppressions := result.Suppressions

	edges := make([]Edge, len(suppressions))
	for i := range suppressions {
		edges[i] = Edge{CursorFunc: result.PageInfo.Cursor, Node: suppressions[i]}
	}

	pageInfo := PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
	}

	if len(suppressions) > 0 {
		var err error
		pageInfo.StartCursor, err = result.PageInfo.Cursor(suppressions[0])
		if err != nil {
			return nil, err
		}

		pageInfo.EndCursor, err = result.PageInfo.Cursor(suppressions[len(suppressions)-1])
		if err != nil {
			return nil, err
		}
	}

	return &EmailSuppressionConnectionResolver{connection: Connection{
		TotalCount: result.PageInfo.TotalCount,
		PageInfo:   pageInfo,
		Edges:      edges,
	}}, nil
}

// TotalCount returns the total result count for the connection.
func (r *EmailSuppressionConnectionResolver) TotalCount(ctx context.Context) (int32, error) {
	return r.connection.TotalCount(ctx)
}

// PageInfo returns the page information for the connection.
func (r *EmailSuppressionConnectionResolver) PageInfo() *PageInfoResolver {
	return &PageInfoResolver{pageInfo: r.connection.PageInfo}
}

// Edges returns the email suppression edges for the connection.
func (r *EmailSuppressionConnectionResolver) Edges() *[]*EmailSuppressionEdgeResolver {
	resolvers := make([]*EmailSuppressionEdgeResolver, len(r.connection.Edges))
	for i, edge := range r.connection.Edges {
		resolvers[i] = &EmailSuppressionEdgeResolver{edge: edge}
	}
	return &resolvers
}

// EmailSuppressionResolver resolves an EmailSuppression.
type EmailSuppressionResolver struct {
	suppression *models.EmailSuppression
}

// ID resolver.
func (r *EmailSuppressionResolver) ID() graphql.ID {
	return graphql.ID(r.suppression.GetGlobalID())
}

// Metadata resolver.
func (r *EmailSuppressionResolver) Metadata() *MetadataResolver {
	return &MetadataResolver{metadata: &r.suppression.Metadata}
}

// Address resolver.
func (r *EmailSuppressionResolver) Address() string {
	return r.suppression.Address
}

// Cause resolver.
func (r *EmailSuppressionResolver) Cause() string {
	return toGraphqlEnum(string(r.suppression.Cause))
}

func emailSuppressionsQuery(ctx context.Context, args *EmailSuppressionConnectionQueryArgs) (*EmailSuppressionConnectionResolver, error) {
	if err := args.Validate(); err != nil {
		return nil, err
	}

	input := &db.GetEmailSuppressionsInput{
		PaginationOptions: &pagination.Options{First: args.First, Last: args.Last, After: args.After, Before: args.Before},
	}

	if args.Addresses != nil || args.Search != nil {
		filter := &db.EmailSuppressionFilter{Search: args.Search}
		if args.Addresses != nil {
			filter.Addresses = *args.Addresses
		}

		input.Filter = filter
	}

	if args.Sort != nil {
		sort := db.EmailSuppressionSortableField(*args.Sort)
		input.Sort = &sort
	}

	return NewEmailSuppressionConnectionResolver(ctx, input)
}

/* EmailSuppression Mutation Resolvers */

// DeleteEmailSuppressionPayload is the response payload for deleting an email suppression.
type DeleteEmailSuppressionPayload struct {
	ClientMutationID *string
	Suppression      *models.EmailSuppression
	Problems         []Problem
}

// DeleteEmailSuppressionPayloadResolver resolves DeleteEmailSuppressionPayload.
type DeleteEmailSuppressionPayloadResolver struct {
	DeleteEmailSuppressionPayload
}

// Suppression field resolver.
func (r *DeleteEmailSuppressionPayloadResolver) Suppression() *EmailSuppressionResolver {
	if r.DeleteEmailSuppressionPayload.Suppression == nil {
		return nil
	}

	return &EmailSuppressionResolver{suppression: r.DeleteEmailSuppressionPayload.Suppression}
}

// DeleteEmailSuppressionInput contains the input for deleting an email suppression.
type DeleteEmailSuppressionInput struct {
	ClientMutationID *string
	ID               string
	Metadata         *MetadataInput
}

func handleEmailSuppressionMutationProblem(e error, clientMutationID *string) (*DeleteEmailSuppressionPayloadResolver, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}

	payload := DeleteEmailSuppressionPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}
	return &DeleteEmailSuppressionPayloadResolver{DeleteEmailSuppressionPayload: payload}, nil
}

func deleteEmailSuppressionMutation(ctx context.Context, input *DeleteEmailSuppressionInput) (*DeleteEmailSuppressionPayloadResolver, error) {
	serviceCatalog := getServiceCatalog(ctx)

	model, err := serviceCatalog.FetchModel(ctx, string(input.ID))
	if err != nil {
		return nil, err
	}

	gotSuppression, ok := model.(*models.EmailSuppression)
	if !ok {
		return nil, errors.New("email suppression %s not found", input.ID, errors.WithErrorCode(errors.ENotFound))
	}

	deleteInput := &email.DeleteSuppressionInput{ID: gotSuppression.Metadata.ID}

	if input.Metadata != nil {
		v, cErr := strconv.Atoi(input.Metadata.Version)
		if cErr != nil {
			return nil, cErr
		}

		deleteInput.MetadataVersion = &v
	}

	if err = serviceCatalog.EmailService.DeleteSuppression(ctx, deleteInput); err != nil {
		return nil, err
	}

	payload := DeleteEmailSuppressionPayload{ClientMutationID: input.ClientMutationID, Suppression: gotSuppression, Problems: []Problem{}}
	return &DeleteEmailSuppressionPayloadResolver{DeleteEmailSuppressionPayload: payload}, nil
}
