package resolver

import (
	"context"
	"strconv"

	"github.com/graph-gophers/dataloader"
	graphql "github.com/graph-gophers/graphql-go"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/loader"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/vcs"
)

// VCSProviderConnectionQueryArgs are used to query a vcsProvider connection
type VCSProviderConnectionQueryArgs struct {
	ConnectionQueryArgs
	IncludeInherited *bool
	Search           *string
}

// VCSProviderEdgeResolver resolves vcsProvider edges
type VCSProviderEdgeResolver struct {
	edge Edge
}

// Cursor returns an opaque cursor
func (r *VCSProviderEdgeResolver) Cursor() (string, error) {
	vcsProvider, ok := r.edge.Node.(models.VCSProvider)
	if !ok {
		return "", errors.NewError(errors.EInternal, "Failed to convert node type")
	}
	cursor, err := r.edge.CursorFunc(&vcsProvider)
	return *cursor, err
}

// Node returns a vcsProvider node
func (r *VCSProviderEdgeResolver) Node(ctx context.Context) (*VCSProviderResolver, error) {
	vcsProvider, ok := r.edge.Node.(models.VCSProvider)
	if !ok {
		return nil, errors.NewError(errors.EInternal, "Failed to convert node type")
	}

	return &VCSProviderResolver{vcsProvider: &vcsProvider}, nil
}

// VCSProviderConnectionResolver resolves a vcs provider connection
type VCSProviderConnectionResolver struct {
	connection Connection
}

// NewVCSProviderConnectionResolver creates a new VCSProviderConnectionResolver
func NewVCSProviderConnectionResolver(ctx context.Context, input *vcs.GetVCSProvidersInput) (*VCSProviderConnectionResolver, error) {
	vcsService := getVCSService(ctx)

	result, err := vcsService.GetVCSProviders(ctx, input)
	if err != nil {
		return nil, err
	}

	vcsProviders := result.VCSProviders

	// Create edges
	edges := make([]Edge, len(vcsProviders))
	for i, vcsProvider := range vcsProviders {
		edges[i] = Edge{CursorFunc: result.PageInfo.Cursor, Node: vcsProvider}
	}

	pageInfo := PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
	}

	if len(vcsProviders) > 0 {
		var err error
		pageInfo.StartCursor, err = result.PageInfo.Cursor(&vcsProviders[0])
		if err != nil {
			return nil, err
		}

		pageInfo.EndCursor, err = result.PageInfo.Cursor(&vcsProviders[len(edges)-1])
		if err != nil {
			return nil, err
		}
	}

	connection := Connection{
		TotalCount: result.PageInfo.TotalCount,
		PageInfo:   pageInfo,
		Edges:      edges,
	}

	return &VCSProviderConnectionResolver{connection: connection}, nil
}

// TotalCount returns the total result count for the connection
func (r *VCSProviderConnectionResolver) TotalCount() int32 {
	return r.connection.TotalCount
}

// PageInfo returns the connection page information
func (r *VCSProviderConnectionResolver) PageInfo() *PageInfoResolver {
	return &PageInfoResolver{pageInfo: r.connection.PageInfo}
}

// Edges returns the connection edges
func (r *VCSProviderConnectionResolver) Edges() *[]*VCSProviderEdgeResolver {
	resolvers := make([]*VCSProviderEdgeResolver, len(r.connection.Edges))
	for i, edge := range r.connection.Edges {
		resolvers[i] = &VCSProviderEdgeResolver{edge: edge}
	}
	return &resolvers
}

// VCSProviderResolver resolves a vcsProvider resource
type VCSProviderResolver struct {
	vcsProvider *models.VCSProvider
}

// ID resolver
func (r *VCSProviderResolver) ID() graphql.ID {
	return graphql.ID(gid.ToGlobalID(gid.VCSProviderType, r.vcsProvider.Metadata.ID))
}

// GroupPath resolver
func (r *VCSProviderResolver) GroupPath() string {
	return r.vcsProvider.GetGroupPath()
}

// ResourcePath resolver
func (r *VCSProviderResolver) ResourcePath() string {
	return r.vcsProvider.ResourcePath
}

// Name resolver
func (r *VCSProviderResolver) Name() string {
	return r.vcsProvider.Name
}

// Description resolver
func (r *VCSProviderResolver) Description() string {
	return r.vcsProvider.Description
}

// Metadata resolver
func (r *VCSProviderResolver) Metadata() *MetadataResolver {
	return &MetadataResolver{metadata: &r.vcsProvider.Metadata}
}

// CreatedBy resolver
func (r *VCSProviderResolver) CreatedBy() string {
	return r.vcsProvider.CreatedBy
}

// Group resolver
func (r *VCSProviderResolver) Group(ctx context.Context) (*GroupResolver, error) {
	group, err := loadGroup(ctx, r.vcsProvider.GroupID)
	if err != nil {
		return nil, err
	}

	return &GroupResolver{group: group}, nil
}

// Hostname resolver
func (r *VCSProviderResolver) Hostname() string {
	return r.vcsProvider.Hostname
}

// Type resolver
func (r *VCSProviderResolver) Type() string {
	return string(r.vcsProvider.Type)
}

// AutoCreateWebhooks resolver
func (r *VCSProviderResolver) AutoCreateWebhooks() bool {
	return r.vcsProvider.AutoCreateWebhooks
}

/* VCSProvider Mutation Resolvers */

// ResetVCSProviderOAuthTokenMutationPayload is the response payload for
// resetting a OAuth token.
type ResetVCSProviderOAuthTokenMutationPayload struct {
	ClientMutationID      *string
	VCSProvider           *models.VCSProvider
	OAuthAuthorizationURL string
	Problems              []Problem
}

// ResetVCSProviderOAuthTokenMutationPayloadResolver resolves a ResetVCSProviderOAuthTokenPayload
type ResetVCSProviderOAuthTokenMutationPayloadResolver struct {
	ResetVCSProviderOAuthTokenMutationPayload
}

// VCSProvider field resolver
func (r *ResetVCSProviderOAuthTokenMutationPayloadResolver) VCSProvider(ctx context.Context) *VCSProviderResolver {
	if r.ResetVCSProviderOAuthTokenMutationPayload.VCSProvider == nil {
		return nil
	}

	return &VCSProviderResolver{vcsProvider: r.ResetVCSProviderOAuthTokenMutationPayload.VCSProvider}
}

// VCSProviderMutationPayload is the response payload for a vcsProvider mutation
type VCSProviderMutationPayload struct {
	ClientMutationID      *string
	VCSProvider           *models.VCSProvider
	OAuthAuthorizationURL string
	Problems              []Problem
}

// VCSProviderMutationPayloadResolver resolves a VCSProviderMutationPayload
type VCSProviderMutationPayloadResolver struct {
	VCSProviderMutationPayload
}

// VCSProvider field resolver
func (r *VCSProviderMutationPayloadResolver) VCSProvider(ctx context.Context) *VCSProviderResolver {
	if r.VCSProviderMutationPayload.VCSProvider == nil {
		return nil
	}

	return &VCSProviderResolver{vcsProvider: r.VCSProviderMutationPayload.VCSProvider}
}

// ResetVCSProviderOAuthTokenInput is the input for resetting a
// VCS provider's OAuth token.
type ResetVCSProviderOAuthTokenInput struct {
	ClientMutationID *string
	ProviderID       string
}

// CreateVCSProviderInput is the input for creating a VCS provider.
type CreateVCSProviderInput struct {
	ClientMutationID   *string
	Hostname           *string
	Name               string
	Description        string
	GroupPath          string
	OAuthClientID      string
	OAuthClientSecret  string
	Type               models.VCSProviderType
	AutoCreateWebhooks bool
}

// UpdateVCSProviderInput is the input for updating a VCS provider.
type UpdateVCSProviderInput struct {
	ClientMutationID  *string
	Metadata          *MetadataInput
	Description       *string
	OAuthClientID     *string
	OAuthClientSecret *string
	ID                string
}

// DeleteVCSProviderInput is the input for deleting a VCS provider.
type DeleteVCSProviderInput struct {
	ClientMutationID *string
	Metadata         *MetadataInput
	Force            *bool
	ID               string
}

func handleVCSProviderMutationProblem(e error, clientMutationID *string) (*VCSProviderMutationPayloadResolver, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}

	payload := VCSProviderMutationPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}
	return &VCSProviderMutationPayloadResolver{VCSProviderMutationPayload: payload}, nil
}

func handleResetVCSProviderOAuthTokenMutationProblem(e error, clientMutationID *string) (*ResetVCSProviderOAuthTokenMutationPayloadResolver, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}

	payload := ResetVCSProviderOAuthTokenMutationPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}
	return &ResetVCSProviderOAuthTokenMutationPayloadResolver{ResetVCSProviderOAuthTokenMutationPayload: payload}, nil
}

func resetVCSProviderOAuthTokenMutation(ctx context.Context, input *ResetVCSProviderOAuthTokenInput) (*ResetVCSProviderOAuthTokenMutationPayloadResolver, error) {
	service := getVCSService(ctx)

	provider, err := service.GetVCSProviderByID(ctx, gid.FromGlobalID(input.ProviderID))
	if err != nil {
		return nil, err
	}

	response, err := service.ResetVCSProviderOAuthToken(ctx, &vcs.ResetVCSProviderOAuthTokenInput{
		VCSProvider: provider,
	})
	if err != nil {
		return nil, err
	}

	payload := ResetVCSProviderOAuthTokenMutationPayload{
		ClientMutationID:      input.ClientMutationID,
		VCSProvider:           response.VCSProvider,
		OAuthAuthorizationURL: response.OAuthAuthorizationURL,
		Problems:              []Problem{},
	}
	return &ResetVCSProviderOAuthTokenMutationPayloadResolver{ResetVCSProviderOAuthTokenMutationPayload: payload}, nil
}

func createVCSProviderMutation(ctx context.Context, input *CreateVCSProviderInput) (*VCSProviderMutationPayloadResolver, error) {
	group, err := getGroupService(ctx).GetGroupByFullPath(ctx, input.GroupPath)
	if err != nil {
		return nil, err
	}

	vcsProviderCreateOptions := &vcs.CreateVCSProviderInput{
		Name:               input.Name,
		Description:        input.Description,
		GroupID:            group.Metadata.ID,
		Hostname:           input.Hostname,
		OAuthClientID:      input.OAuthClientID,
		OAuthClientSecret:  input.OAuthClientSecret,
		Type:               input.Type,
		AutoCreateWebhooks: input.AutoCreateWebhooks,
	}

	vcsService := getVCSService(ctx)

	response, err := vcsService.CreateVCSProvider(ctx, vcsProviderCreateOptions)
	if err != nil {
		return nil, err
	}

	payload := VCSProviderMutationPayload{
		ClientMutationID:      input.ClientMutationID,
		VCSProvider:           response.VCSProvider,
		OAuthAuthorizationURL: response.OAuthAuthorizationURL,
		Problems:              []Problem{},
	}
	return &VCSProviderMutationPayloadResolver{VCSProviderMutationPayload: payload}, nil
}

func updateVCSProviderMutation(ctx context.Context, input *UpdateVCSProviderInput) (*VCSProviderMutationPayloadResolver, error) {
	vcsService := getVCSService(ctx)

	vcsProvider, err := vcsService.GetVCSProviderByID(ctx, gid.FromGlobalID(input.ID))
	if err != nil {
		return nil, err
	}

	// Check if resource version is specified
	if input.Metadata != nil {
		v, cErr := strconv.Atoi(input.Metadata.Version)
		if cErr != nil {
			return nil, cErr
		}

		vcsProvider.Metadata.Version = v
	}

	if input.Description != nil {
		vcsProvider.Description = *input.Description
	}

	if input.OAuthClientID != nil {
		vcsProvider.OAuthClientID = *input.OAuthClientID
	}

	if input.OAuthClientSecret != nil {
		vcsProvider.OAuthClientSecret = *input.OAuthClientSecret
	}

	updatedProvider, err := vcsService.UpdateVCSProvider(ctx, &vcs.UpdateVCSProviderInput{Provider: vcsProvider})
	if err != nil {
		return nil, err
	}

	payload := VCSProviderMutationPayload{ClientMutationID: input.ClientMutationID, VCSProvider: updatedProvider, Problems: []Problem{}}
	return &VCSProviderMutationPayloadResolver{VCSProviderMutationPayload: payload}, nil
}

func deleteVCSProviderMutation(ctx context.Context, input *DeleteVCSProviderInput) (*VCSProviderMutationPayloadResolver, error) {
	vcsService := getVCSService(ctx)

	vcsProvider, err := vcsService.GetVCSProviderByID(ctx, gid.FromGlobalID(input.ID))
	if err != nil {
		return nil, err
	}

	// Check if resource version is specified
	if input.Metadata != nil {
		v, err := strconv.Atoi(input.Metadata.Version)
		if err != nil {
			return nil, err
		}

		vcsProvider.Metadata.Version = v
	}

	deleteOptions := vcs.DeleteVCSProviderInput{
		Provider: vcsProvider,
	}

	if input.Force != nil {
		deleteOptions.Force = *input.Force
	}

	if err := vcsService.DeleteVCSProvider(ctx, &deleteOptions); err != nil {
		return nil, err
	}

	payload := VCSProviderMutationPayload{ClientMutationID: input.ClientMutationID, VCSProvider: vcsProvider, Problems: []Problem{}}
	return &VCSProviderMutationPayloadResolver{VCSProviderMutationPayload: payload}, nil
}

/* VCSProvider loader */

const vcsProviderLoaderKey = "vcsProvider"

// RegisterVCSProviderLoader registers a VCS provider loader function
func RegisterVCSProviderLoader(collection *loader.Collection) {
	collection.Register(vcsProviderLoaderKey, vcsProviderBatchFunc)
}

func loadVCSProvider(ctx context.Context, id string) (*models.VCSProvider, error) {
	ldr, err := loader.Extract(ctx, vcsProviderLoaderKey)
	if err != nil {
		return nil, err
	}

	data, err := ldr.Load(ctx, dataloader.StringKey(id))()
	if err != nil {
		return nil, err
	}

	vp, ok := data.(models.VCSProvider)
	if !ok {
		return nil, errors.NewError(errors.EInternal, "Wrong type")
	}

	return &vp, nil
}

func vcsProviderBatchFunc(ctx context.Context, ids []string) (loader.DataBatch, error) {
	service := getVCSService(ctx)

	providers, err := service.GetVCSProvidersByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	// Build map of results
	batch := loader.DataBatch{}
	for _, result := range providers {
		batch[result.Metadata.ID] = result
	}

	return batch, nil
}
