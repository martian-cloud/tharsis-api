package resolver

import (
	"context"

	graphql "github.com/graph-gophers/graphql-go"

	"github.com/graph-gophers/dataloader"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/loader"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

/* EmailOutboxItem Query Resolvers */

// EmailOutboxItemConnectionQueryArgs contains the arguments for the emailOutboxItems query.
type EmailOutboxItemConnectionQueryArgs struct {
	ConnectionQueryArgs
	Ephemeral     *bool
	SubjectSearch *string
}

// EmailOutboxItemEdgeResolver resolves email outbox edges.
type EmailOutboxItemEdgeResolver struct {
	edge Edge
}

// Cursor returns an opaque cursor.
func (r *EmailOutboxItemEdgeResolver) Cursor() (string, error) {
	outbox, ok := r.edge.Node.(models.EmailOutboxItem)
	if !ok {
		return "", errors.New("failed to convert node type")
	}
	cursor, err := r.edge.CursorFunc(&outbox)
	return *cursor, err
}

// Node returns an email outbox node.
func (r *EmailOutboxItemEdgeResolver) Node() (*EmailOutboxItemResolver, error) {
	outbox, ok := r.edge.Node.(models.EmailOutboxItem)
	if !ok {
		return nil, errors.New("failed to convert node type")
	}
	return &EmailOutboxItemResolver{outbox: &outbox}, nil
}

// EmailOutboxItemConnectionResolver resolves an email outbox connection.
type EmailOutboxItemConnectionResolver struct {
	connection Connection
}

// NewEmailOutboxItemConnectionResolver creates a new EmailOutboxItemConnectionResolver.
func NewEmailOutboxItemConnectionResolver(ctx context.Context, input *db.GetEmailOutboxItemsInput) (*EmailOutboxItemConnectionResolver, error) {
	result, err := getServiceCatalog(ctx).EmailService.GetOutboxItems(ctx, input)
	if err != nil {
		return nil, err
	}

	outboxes := result.OutboxItems

	edges := make([]Edge, len(outboxes))
	for i := range outboxes {
		edges[i] = Edge{CursorFunc: result.PageInfo.Cursor, Node: outboxes[i]}
	}

	pageInfo := PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
	}

	if len(outboxes) > 0 {
		var err error
		pageInfo.StartCursor, err = result.PageInfo.Cursor(&outboxes[0])
		if err != nil {
			return nil, err
		}

		pageInfo.EndCursor, err = result.PageInfo.Cursor(&outboxes[len(outboxes)-1])
		if err != nil {
			return nil, err
		}
	}

	return &EmailOutboxItemConnectionResolver{connection: Connection{
		TotalCount: result.PageInfo.TotalCount,
		PageInfo:   pageInfo,
		Edges:      edges,
	}}, nil
}

// TotalCount returns the total result count for the connection.
func (r *EmailOutboxItemConnectionResolver) TotalCount(ctx context.Context) (int32, error) {
	return r.connection.TotalCount(ctx)
}

// PageInfo returns the page information for the connection.
func (r *EmailOutboxItemConnectionResolver) PageInfo() *PageInfoResolver {
	return &PageInfoResolver{pageInfo: r.connection.PageInfo}
}

// Edges returns the email outbox edges for the connection.
func (r *EmailOutboxItemConnectionResolver) Edges() *[]*EmailOutboxItemEdgeResolver {
	resolvers := make([]*EmailOutboxItemEdgeResolver, len(r.connection.Edges))
	for i, edge := range r.connection.Edges {
		resolvers[i] = &EmailOutboxItemEdgeResolver{edge: edge}
	}
	return &resolvers
}

// EmailOutboxItemResolver resolves an EmailOutboxItem.
type EmailOutboxItemResolver struct {
	outbox *models.EmailOutboxItem
}

// ID resolver.
func (r *EmailOutboxItemResolver) ID() graphql.ID {
	return graphql.ID(r.outbox.GetGlobalID())
}

// Metadata resolver.
func (r *EmailOutboxItemResolver) Metadata() *MetadataResolver {
	return &MetadataResolver{metadata: &r.outbox.Metadata}
}

// EmailType resolver.
func (r *EmailOutboxItemResolver) EmailType() string {
	return toGraphqlEnum(string(r.outbox.EmailType))
}

// Subject resolver.
func (r *EmailOutboxItemResolver) Subject() string {
	return r.outbox.Subject
}

// Ephemeral resolver.
func (r *EmailOutboxItemResolver) Ephemeral() bool {
	return r.outbox.Ephemeral
}

// Status resolver.
func (r *EmailOutboxItemResolver) Status() string {
	return toGraphqlEnum(string(r.outbox.Status))
}

// RecipientStats resolves the outbox's aggregate delivery breakdown.
func (r *EmailOutboxItemResolver) RecipientStats(ctx context.Context) (*EmailRecipientStatsResolver, error) {
	stats, err := getServiceCatalog(ctx).EmailService.GetRecipientStats(ctx, r.outbox.Metadata.ID)
	if err != nil {
		return nil, err
	}

	return &EmailRecipientStatsResolver{stats: stats}, nil
}

// Recipients resolves the outbox's recipients connection.
func (r *EmailOutboxItemResolver) Recipients(ctx context.Context, args *EmailRecipientConnectionQueryArgs) (*EmailRecipientConnectionResolver, error) {
	if err := args.Validate(); err != nil {
		return nil, err
	}

	filter := &db.EmailRecipientFilter{
		EmailOutboxItemID: &r.outbox.Metadata.ID,
		Search:            args.Search,
		HasOpened:         args.HasOpened,
		HasIssues:         args.HasIssues,
	}

	if args.DeliveryStatuses != nil {
		statuses := make([]models.EmailDeliveryStatus, len(*args.DeliveryStatuses))
		for i, s := range *args.DeliveryStatuses {
			statuses[i] = models.EmailDeliveryStatus(fromGraphqlEnum(string(s)))
		}

		filter.DeliveryStatuses = statuses
	}

	input := &db.GetEmailRecipientsInput{
		PaginationOptions: &pagination.Options{First: args.First, Last: args.Last, After: args.After, Before: args.Before},
		Filter:            filter,
	}

	if args.Sort != nil {
		sort := db.EmailRecipientSortableField(*args.Sort)
		input.Sort = &sort
	}

	return NewEmailRecipientConnectionResolver(ctx, input)
}

func emailOutboxItemsQuery(ctx context.Context, args *EmailOutboxItemConnectionQueryArgs) (*EmailOutboxItemConnectionResolver, error) {
	if err := args.Validate(); err != nil {
		return nil, err
	}

	input := &db.GetEmailOutboxItemsInput{
		PaginationOptions: &pagination.Options{First: args.First, Last: args.Last, After: args.After, Before: args.Before},
	}

	if args.Ephemeral != nil || args.SubjectSearch != nil {
		input.Filter = &db.EmailOutboxItemFilter{Ephemeral: args.Ephemeral, SubjectSearch: args.SubjectSearch}
	}

	if args.Sort != nil {
		sort := db.EmailOutboxItemSortableField(*args.Sort)
		input.Sort = &sort
	}

	return NewEmailOutboxItemConnectionResolver(ctx, input)
}

/* EmailOutboxItem loader */

const emailOutboxItemLoaderKey = "emailOutboxItem"

// RegisterEmailOutboxItemLoader registers an email outbox item loader function.
func RegisterEmailOutboxItemLoader(collection *loader.Collection) {
	collection.Register(emailOutboxItemLoaderKey, emailOutboxItemBatchFunc)
}

func loadEmailOutboxItem(ctx context.Context, id string) (*models.EmailOutboxItem, error) {
	ldr, err := loader.Extract(ctx, emailOutboxItemLoaderKey)
	if err != nil {
		return nil, err
	}

	data, err := ldr.Load(ctx, dataloader.StringKey(id))()
	if err != nil {
		return nil, err
	}

	outboxItem, ok := data.(models.EmailOutboxItem)
	if !ok {
		return nil, errors.New("wrong type")
	}

	return &outboxItem, nil
}

func emailOutboxItemBatchFunc(ctx context.Context, ids []string) (loader.DataBatch, error) {
	outboxItems, err := getServiceCatalog(ctx).EmailService.GetOutboxItemsByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	// Build map of results
	batch := loader.DataBatch{}
	for _, result := range outboxItems {
		batch[result.Metadata.ID] = result
	}

	return batch, nil
}
