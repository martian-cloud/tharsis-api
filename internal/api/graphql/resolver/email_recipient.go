package resolver

import (
	"context"

	graphql "github.com/graph-gophers/graphql-go"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

/* EmailRecipient Query Resolvers */

// EmailRecipientConnectionQueryArgs contains the arguments for the EmailOutboxItem.recipients field.
type EmailRecipientConnectionQueryArgs struct {
	ConnectionQueryArgs
	DeliveryStatuses *[]models.EmailDeliveryStatus
	HasOpened        *bool
	HasIssues        *bool
	Search           *string
}

// EmailRecipientStatsResolver resolves an EmailRecipientStats.
type EmailRecipientStatsResolver struct {
	stats *db.EmailRecipientStatsResult
}

// Total resolver.
func (r *EmailRecipientStatsResolver) Total() int32 {
	return int32(r.stats.Total)
}

// Delivered resolver.
func (r *EmailRecipientStatsResolver) Delivered() int32 {
	return int32(r.stats.Delivered)
}

// Opened resolver.
func (r *EmailRecipientStatsResolver) Opened() int32 {
	return int32(r.stats.Opened)
}

// Clicked resolver.
func (r *EmailRecipientStatsResolver) Clicked() int32 {
	return int32(r.stats.Clicked)
}

// Issues resolver.
func (r *EmailRecipientStatsResolver) Issues() int32 {
	return int32(r.stats.Issues)
}

// EmailRecipientEdgeResolver resolves email recipient edges.
type EmailRecipientEdgeResolver struct {
	edge Edge
}

// Cursor returns an opaque cursor.
func (r *EmailRecipientEdgeResolver) Cursor() (string, error) {
	recipient, ok := r.edge.Node.(models.EmailRecipient)
	if !ok {
		return "", errors.New("failed to convert node type")
	}
	cursor, err := r.edge.CursorFunc(&recipient)
	return *cursor, err
}

// Node returns an email recipient node.
func (r *EmailRecipientEdgeResolver) Node() (*EmailRecipientResolver, error) {
	recipient, ok := r.edge.Node.(models.EmailRecipient)
	if !ok {
		return nil, errors.New("failed to convert node type")
	}
	return &EmailRecipientResolver{recipient: &recipient}, nil
}

// EmailRecipientConnectionResolver resolves an email recipient connection.
type EmailRecipientConnectionResolver struct {
	connection Connection
}

// NewEmailRecipientConnectionResolver creates a new EmailRecipientConnectionResolver.
func NewEmailRecipientConnectionResolver(ctx context.Context, input *db.GetEmailRecipientsInput) (*EmailRecipientConnectionResolver, error) {
	result, err := getServiceCatalog(ctx).EmailService.GetRecipients(ctx, input)
	if err != nil {
		return nil, err
	}

	recipients := result.Recipients

	edges := make([]Edge, len(recipients))
	for i := range recipients {
		edges[i] = Edge{CursorFunc: result.PageInfo.Cursor, Node: recipients[i]}
	}

	pageInfo := PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
	}

	if len(recipients) > 0 {
		var err error
		pageInfo.StartCursor, err = result.PageInfo.Cursor(&recipients[0])
		if err != nil {
			return nil, err
		}

		pageInfo.EndCursor, err = result.PageInfo.Cursor(&recipients[len(recipients)-1])
		if err != nil {
			return nil, err
		}
	}

	return &EmailRecipientConnectionResolver{connection: Connection{
		TotalCount: result.PageInfo.TotalCount,
		PageInfo:   pageInfo,
		Edges:      edges,
	}}, nil
}

// TotalCount returns the total result count for the connection.
func (r *EmailRecipientConnectionResolver) TotalCount(ctx context.Context) (int32, error) {
	return r.connection.TotalCount(ctx)
}

// PageInfo returns the page information for the connection.
func (r *EmailRecipientConnectionResolver) PageInfo() *PageInfoResolver {
	return &PageInfoResolver{pageInfo: r.connection.PageInfo}
}

// Edges returns the email recipient edges for the connection.
func (r *EmailRecipientConnectionResolver) Edges() *[]*EmailRecipientEdgeResolver {
	resolvers := make([]*EmailRecipientEdgeResolver, len(r.connection.Edges))
	for i, edge := range r.connection.Edges {
		resolvers[i] = &EmailRecipientEdgeResolver{edge: edge}
	}
	return &resolvers
}

// EmailRecipientResolver resolves an EmailRecipient.
type EmailRecipientResolver struct {
	recipient *models.EmailRecipient
}

// ID resolver.
func (r *EmailRecipientResolver) ID() graphql.ID {
	return graphql.ID(r.recipient.GetGlobalID())
}

// Metadata resolver.
func (r *EmailRecipientResolver) Metadata() *MetadataResolver {
	return &MetadataResolver{metadata: &r.recipient.Metadata}
}

// OutboxItem resolves the outbox item this recipient belongs to.
func (r *EmailRecipientResolver) OutboxItem(ctx context.Context) (*EmailOutboxItemResolver, error) {
	outboxItem, err := loadEmailOutboxItem(ctx, r.recipient.EmailOutboxItemID)
	if err != nil {
		return nil, err
	}

	return &EmailOutboxItemResolver{outbox: outboxItem}, nil
}

// Address resolver.
func (r *EmailRecipientResolver) Address() string {
	return r.recipient.Address
}

// DeliveryStatus resolver.
func (r *EmailRecipientResolver) DeliveryStatus() string {
	return toGraphqlEnum(string(r.recipient.DeliveryStatus))
}

// AttemptCount resolver.
func (r *EmailRecipientResolver) AttemptCount() int32 {
	return int32(r.recipient.AttemptCount)
}

// AvailableAt resolver.
func (r *EmailRecipientResolver) AvailableAt() graphql.Time {
	return graphql.Time{Time: r.recipient.AvailableAt}
}

// LastAttemptAt resolver.
func (r *EmailRecipientResolver) LastAttemptAt() *graphql.Time {
	if r.recipient.LastAttemptAt == nil {
		return nil
	}

	return &graphql.Time{Time: *r.recipient.LastAttemptAt}
}

// OpenedAt resolver.
func (r *EmailRecipientResolver) OpenedAt() *graphql.Time {
	if r.recipient.OpenedAt == nil {
		return nil
	}
	return &graphql.Time{Time: *r.recipient.OpenedAt}
}

// ClickedAt resolver.
func (r *EmailRecipientResolver) ClickedAt() *graphql.Time {
	if r.recipient.ClickedAt == nil {
		return nil
	}
	return &graphql.Time{Time: *r.recipient.ClickedAt}
}

// ComplainedAt resolver.
func (r *EmailRecipientResolver) ComplainedAt() *graphql.Time {
	if r.recipient.ComplainedAt == nil {
		return nil
	}
	return &graphql.Time{Time: *r.recipient.ComplainedAt}
}

// FailureReason resolver.
func (r *EmailRecipientResolver) FailureReason() *string {
	return r.recipient.FailureReason
}

/* EmailRecipient Mutation Resolvers */

// MarkEmailRecipientClickedPayload is the response payload for marking an email recipient as clicked.
type MarkEmailRecipientClickedPayload struct {
	ClientMutationID *string
	EmailRecipient   *models.EmailRecipient
	Problems         []Problem
}

// MarkEmailRecipientClickedPayloadResolver resolves MarkEmailRecipientClickedPayload.
type MarkEmailRecipientClickedPayloadResolver struct {
	MarkEmailRecipientClickedPayload
}

// EmailRecipient field resolver.
func (r *MarkEmailRecipientClickedPayloadResolver) EmailRecipient() *EmailRecipientResolver {
	if r.MarkEmailRecipientClickedPayload.EmailRecipient == nil {
		return nil
	}

	return &EmailRecipientResolver{recipient: r.MarkEmailRecipientClickedPayload.EmailRecipient}
}

// MarkEmailRecipientClickedInput contains the input for marking an email recipient as clicked.
type MarkEmailRecipientClickedInput struct {
	ClientMutationID *string
	RecipientID      string
}

func handleMarkEmailRecipientClickedProblem(e error, clientMutationID *string) (*MarkEmailRecipientClickedPayloadResolver, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}

	payload := MarkEmailRecipientClickedPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}
	return &MarkEmailRecipientClickedPayloadResolver{MarkEmailRecipientClickedPayload: payload}, nil
}

// markEmailRecipientClickedMutation decodes the global ID directly rather than going through
// serviceCatalog.FetchModel, since that routes through the admin-only GetRecipientByID query.
func markEmailRecipientClickedMutation(ctx context.Context, input *MarkEmailRecipientClickedInput) (*MarkEmailRecipientClickedPayloadResolver, error) {
	updated, err := getServiceCatalog(ctx).EmailService.MarkRecipientClicked(ctx, gid.FromGlobalID(input.RecipientID))
	if err != nil {
		return nil, err
	}

	payload := MarkEmailRecipientClickedPayload{ClientMutationID: input.ClientMutationID, EmailRecipient: updated, Problems: []Problem{}}
	return &MarkEmailRecipientClickedPayloadResolver{MarkEmailRecipientClickedPayload: payload}, nil
}
