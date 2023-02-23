package resolver

import (
	"context"

	"github.com/graph-gophers/dataloader"
	graphql "github.com/graph-gophers/graphql-go"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/loader"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/vcs"
)

// VCSEventConnectionQueryArgs are used to query a vcsEvent connection
type VCSEventConnectionQueryArgs struct {
	ConnectionQueryArgs
	WorkspacePath string
}

// VCSEventEdgeResolver resolves vcsEvent edges
type VCSEventEdgeResolver struct {
	edge Edge
}

// Cursor returns an opaque cursor
func (r *VCSEventEdgeResolver) Cursor() (string, error) {
	vcsEvent, ok := r.edge.Node.(models.VCSEvent)
	if !ok {
		return "", errors.NewError(errors.EInternal, "Failed to convert node type")
	}
	cursor, err := r.edge.CursorFunc(&vcsEvent)
	return *cursor, err
}

// Node returns a vcsEvent node
func (r *VCSEventEdgeResolver) Node() (*VCSEventResolver, error) {
	vcsEvent, ok := r.edge.Node.(models.VCSEvent)
	if !ok {
		return nil, errors.NewError(errors.EInternal, "Failed to convert node type")
	}

	return &VCSEventResolver{vcsEvent: &vcsEvent}, nil
}

// VCSEventConnectionResolver resolves a vcs event connection
type VCSEventConnectionResolver struct {
	connection Connection
}

// NewVCSEventConnectionResolver creates a new VCSEventConnectionResolver
func NewVCSEventConnectionResolver(ctx context.Context, input *vcs.GetVCSEventsInput) (*VCSEventConnectionResolver, error) {
	vcsService := getVCSService(ctx)

	result, err := vcsService.GetVCSEvents(ctx, input)
	if err != nil {
		return nil, err
	}

	vcsEvents := result.VCSEvents

	// Create edges
	edges := make([]Edge, len(vcsEvents))
	for i, vcsEvent := range vcsEvents {
		edges[i] = Edge{CursorFunc: result.PageInfo.Cursor, Node: vcsEvent}
	}

	pageInfo := PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
	}

	if len(vcsEvents) > 0 {
		var err error
		pageInfo.StartCursor, err = result.PageInfo.Cursor(&vcsEvents[0])
		if err != nil {
			return nil, err
		}

		pageInfo.EndCursor, err = result.PageInfo.Cursor(&vcsEvents[len(edges)-1])
		if err != nil {
			return nil, err
		}
	}

	connection := Connection{
		TotalCount: result.PageInfo.TotalCount,
		PageInfo:   pageInfo,
		Edges:      edges,
	}

	return &VCSEventConnectionResolver{connection: connection}, nil
}

// TotalCount returns the total result count for the connection
func (r *VCSEventConnectionResolver) TotalCount() int32 {
	return r.connection.TotalCount
}

// PageInfo returns the connection page information
func (r *VCSEventConnectionResolver) PageInfo() *PageInfoResolver {
	return &PageInfoResolver{pageInfo: r.connection.PageInfo}
}

// Edges returns the connection edges
func (r *VCSEventConnectionResolver) Edges() *[]*VCSEventEdgeResolver {
	resolvers := make([]*VCSEventEdgeResolver, len(r.connection.Edges))
	for i, edge := range r.connection.Edges {
		resolvers[i] = &VCSEventEdgeResolver{edge: edge}
	}
	return &resolvers
}

// VCSEventResolver resolves a vcsEvent resource
type VCSEventResolver struct {
	vcsEvent *models.VCSEvent
}

// ID resolver
func (r *VCSEventResolver) ID() graphql.ID {
	return graphql.ID(gid.ToGlobalID(gid.VCSEventType, r.vcsEvent.Metadata.ID))
}

// Metadata resolver
func (r *VCSEventResolver) Metadata() *MetadataResolver {
	return &MetadataResolver{metadata: &r.vcsEvent.Metadata}
}

// Status resolver
func (r *VCSEventResolver) Status() string {
	return string(r.vcsEvent.Status)
}

// Type resolver
func (r *VCSEventResolver) Type() string {
	return string(r.vcsEvent.Type)
}

// RepositoryURL resolver
func (r *VCSEventResolver) RepositoryURL() string {
	return r.vcsEvent.RepositoryURL
}

// Workspace resolver
func (r *VCSEventResolver) Workspace(ctx context.Context) (*WorkspaceResolver, error) {
	workspace, err := loadWorkspace(ctx, r.vcsEvent.WorkspaceID)
	if err != nil {
		return nil, err
	}

	return &WorkspaceResolver{workspace: workspace}, nil
}

// SourceReferenceName resolver
func (r *VCSEventResolver) SourceReferenceName() *string {
	return r.vcsEvent.SourceReferenceName
}

// CommitID resolver
func (r *VCSEventResolver) CommitID() *string {
	return r.vcsEvent.CommitID
}

// ErrorMessage resolver
func (r *VCSEventResolver) ErrorMessage() *string {
	return r.vcsEvent.ErrorMessage
}

/* VCSEvent loader */

const vcsEventLoaderKey = "vcsEvent"

// RegisterVCSEventLoader registers a VCS event loader function
func RegisterVCSEventLoader(collection *loader.Collection) {
	collection.Register(vcsEventLoaderKey, vcsEventBatchFunc)
}

func loadVCSEvent(ctx context.Context, id string) (*models.VCSEvent, error) {
	ldr, err := loader.Extract(ctx, vcsEventLoaderKey)
	if err != nil {
		return nil, err
	}

	data, err := ldr.Load(ctx, dataloader.StringKey(id))()
	if err != nil {
		return nil, err
	}

	ve, ok := data.(models.VCSEvent)
	if !ok {
		return nil, errors.NewError(errors.EInternal, "Wrong type")
	}

	return &ve, nil
}

func vcsEventBatchFunc(ctx context.Context, ids []string) (loader.DataBatch, error) {
	service := getVCSService(ctx)

	events, err := service.GetVCSEventsByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	// Build map of results
	batch := loader.DataBatch{}
	for _, result := range events {
		batch[result.Metadata.ID] = result
	}

	return batch, nil
}
