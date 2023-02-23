package resolver

import (
	"context"
	"strconv"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/loader"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/gpgkey"

	"github.com/graph-gophers/dataloader"
	graphql "github.com/graph-gophers/graphql-go"
)

/* GPGKey Query Resolvers */

// GPGKeysConnectionQueryArgs are used to query a gpgKey connection
type GPGKeysConnectionQueryArgs struct {
	ConnectionQueryArgs
	IncludeInherited *bool
}

// GPGKeyEdgeResolver resolves gpgKey edges
type GPGKeyEdgeResolver struct {
	edge Edge
}

// Cursor returns an opaque cursor
func (r *GPGKeyEdgeResolver) Cursor() (string, error) {
	gpgKey, ok := r.edge.Node.(models.GPGKey)
	if !ok {
		return "", errors.NewError(errors.EInternal, "Failed to convert node type")
	}
	cursor, err := r.edge.CursorFunc(&gpgKey)
	return *cursor, err
}

// Node returns a gpgKey node
func (r *GPGKeyEdgeResolver) Node() (*GPGKeyResolver, error) {
	gpgKey, ok := r.edge.Node.(models.GPGKey)
	if !ok {
		return nil, errors.NewError(errors.EInternal, "Failed to convert node type")
	}

	return &GPGKeyResolver{gpgKey: &gpgKey}, nil
}

// GPGKeyConnectionResolver resolves a gpgKey connection
type GPGKeyConnectionResolver struct {
	connection Connection
}

// NewGPGKeyConnectionResolver creates a new GPGKeyConnectionResolver
func NewGPGKeyConnectionResolver(ctx context.Context, input *gpgkey.GetGPGKeysInput) (*GPGKeyConnectionResolver, error) {
	service := getGPGKeyService(ctx)

	result, err := service.GetGPGKeys(ctx, input)
	if err != nil {
		return nil, err
	}

	gpgKeys := result.GPGKeys

	// Create edges
	edges := make([]Edge, len(gpgKeys))
	for i, gpgKey := range gpgKeys {
		edges[i] = Edge{CursorFunc: result.PageInfo.Cursor, Node: gpgKey}
	}

	pageInfo := PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
	}

	if len(gpgKeys) > 0 {
		var err error
		pageInfo.StartCursor, err = result.PageInfo.Cursor(&gpgKeys[0])
		if err != nil {
			return nil, err
		}

		pageInfo.EndCursor, err = result.PageInfo.Cursor(&gpgKeys[len(edges)-1])
		if err != nil {
			return nil, err
		}
	}

	connection := Connection{
		TotalCount: result.PageInfo.TotalCount,
		PageInfo:   pageInfo,
		Edges:      edges,
	}

	return &GPGKeyConnectionResolver{connection: connection}, nil
}

// TotalCount returns the total result count for the connection
func (r *GPGKeyConnectionResolver) TotalCount() int32 {
	return r.connection.TotalCount
}

// PageInfo returns the connection page information
func (r *GPGKeyConnectionResolver) PageInfo() *PageInfoResolver {
	return &PageInfoResolver{pageInfo: r.connection.PageInfo}
}

// Edges returns the connection edges
func (r *GPGKeyConnectionResolver) Edges() *[]*GPGKeyEdgeResolver {
	resolvers := make([]*GPGKeyEdgeResolver, len(r.connection.Edges))
	for i, edge := range r.connection.Edges {
		resolvers[i] = &GPGKeyEdgeResolver{edge: edge}
	}
	return &resolvers
}

// GPGKeyResolver resolves a gpgKey resource
type GPGKeyResolver struct {
	gpgKey *models.GPGKey
}

// ID resolver
func (r *GPGKeyResolver) ID() graphql.ID {
	return graphql.ID(gid.ToGlobalID(gid.GPGKeyType, r.gpgKey.Metadata.ID))
}

// GPGKeyID resolver
func (r *GPGKeyResolver) GPGKeyID() string {
	return r.gpgKey.GetHexGPGKeyID()
}

// Fingerprint resolver
func (r *GPGKeyResolver) Fingerprint() string {
	return r.gpgKey.Fingerprint
}

// ASCIIArmor resolver
func (r *GPGKeyResolver) ASCIIArmor() string {
	return r.gpgKey.ASCIIArmor
}

// Metadata resolver
func (r *GPGKeyResolver) Metadata() *MetadataResolver {
	return &MetadataResolver{metadata: &r.gpgKey.Metadata}
}

// Group resolver
func (r *GPGKeyResolver) Group(ctx context.Context) (*GroupResolver, error) {
	group, err := loadGroup(ctx, r.gpgKey.GroupID)
	if err != nil {
		return nil, err
	}

	return &GroupResolver{group: group}, nil
}

// CreatedBy resolver
func (r *GPGKeyResolver) CreatedBy() string {
	return r.gpgKey.CreatedBy
}

/* GPGKey Mutation Resolvers */

// GPGKeyMutationPayload is the response payload for a gpgKey mutation
type GPGKeyMutationPayload struct {
	ClientMutationID *string
	GPGKey           *models.GPGKey
	Problems         []Problem
}

// GPGKeyMutationPayloadResolver resolves a GPGKeyMutationPayload
type GPGKeyMutationPayloadResolver struct {
	GPGKeyMutationPayload
}

// GPGKey field resolver
func (r *GPGKeyMutationPayloadResolver) GPGKey() *GPGKeyResolver {
	if r.GPGKeyMutationPayload.GPGKey == nil {
		return nil
	}
	return &GPGKeyResolver{gpgKey: r.GPGKeyMutationPayload.GPGKey}
}

// CreateGPGKeyInput contains the input for creating a new gpgKey
type CreateGPGKeyInput struct {
	ClientMutationID *string
	GroupPath        string
	ASCIIArmor       string
}

// DeleteGPGKeyInput contains the input for deleting a gpgKey
type DeleteGPGKeyInput struct {
	ClientMutationID *string
	Metadata         *MetadataInput
	ID               string
}

func handleGPGKeyMutationProblem(e error, clientMutationID *string) (*GPGKeyMutationPayloadResolver, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}
	payload := GPGKeyMutationPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}
	return &GPGKeyMutationPayloadResolver{GPGKeyMutationPayload: payload}, nil
}

func createGPGKeyMutation(ctx context.Context, input *CreateGPGKeyInput) (*GPGKeyMutationPayloadResolver, error) {
	group, err := getGroupService(ctx).GetGroupByFullPath(ctx, input.GroupPath)
	if err != nil {
		return nil, err
	}

	service := getGPGKeyService(ctx)

	createdGPGKey, err := service.CreateGPGKey(ctx, &gpgkey.CreateGPGKeyInput{
		GroupID:    group.Metadata.ID,
		ASCIIArmor: input.ASCIIArmor,
	})
	if err != nil {
		return nil, err
	}

	payload := GPGKeyMutationPayload{ClientMutationID: input.ClientMutationID, GPGKey: createdGPGKey, Problems: []Problem{}}
	return &GPGKeyMutationPayloadResolver{GPGKeyMutationPayload: payload}, nil
}

func deleteGPGKeyMutation(ctx context.Context, input *DeleteGPGKeyInput) (*GPGKeyMutationPayloadResolver, error) {
	service := getGPGKeyService(ctx)

	gpgKey, err := service.GetGPGKeyByID(ctx, gid.FromGlobalID(input.ID))
	if err != nil {
		return nil, err
	}

	// Check if resource version is specified
	if input.Metadata != nil {
		v, err := strconv.Atoi(input.Metadata.Version)
		if err != nil {
			return nil, err
		}

		gpgKey.Metadata.Version = v
	}

	if err := service.DeleteGPGKey(ctx, gpgKey); err != nil {
		return nil, err
	}

	payload := GPGKeyMutationPayload{ClientMutationID: input.ClientMutationID, GPGKey: gpgKey, Problems: []Problem{}}
	return &GPGKeyMutationPayloadResolver{GPGKeyMutationPayload: payload}, nil
}

/* GPG key loader */

const gpgKeyLoaderKey = "gpgKey"

// RegisterGPGKeyLoader registers a GPG key loader function
func RegisterGPGKeyLoader(collection *loader.Collection) {
	collection.Register(gpgKeyLoaderKey, gpgKeyBatchFunc)
}

func loadGPGKey(ctx context.Context, id string) (*models.GPGKey, error) {
	ldr, err := loader.Extract(ctx, gpgKeyLoaderKey)
	if err != nil {
		return nil, err
	}

	data, err := ldr.Load(ctx, dataloader.StringKey(id))()
	if err != nil {
		return nil, err
	}

	gpgKey, ok := data.(models.GPGKey)
	if !ok {
		return nil, errors.NewError(errors.EInternal, "Wrong type")
	}

	return &gpgKey, nil
}

func gpgKeyBatchFunc(ctx context.Context, ids []string) (loader.DataBatch, error) {
	gpgKeys, err := getGPGKeyService(ctx).GetGPGKeysByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	// Build map of results
	batch := loader.DataBatch{}
	for _, result := range gpgKeys {
		batch[result.Metadata.ID] = result
	}

	return batch, nil
}
