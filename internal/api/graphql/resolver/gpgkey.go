package resolver

import (
	"context"
	"strconv"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/loader"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/gpgkey"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"

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
		return "", errors.New("Failed to convert node type")
	}
	cursor, err := r.edge.CursorFunc(&gpgKey)
	return *cursor, err
}

// Node returns a gpgKey node
func (r *GPGKeyEdgeResolver) Node() (*GPGKeyResolver, error) {
	gpgKey, ok := r.edge.Node.(models.GPGKey)
	if !ok {
		return nil, errors.New("Failed to convert node type")
	}

	return &GPGKeyResolver{gpgKey: &gpgKey}, nil
}

// GPGKeyConnectionResolver resolves a gpgKey connection
type GPGKeyConnectionResolver struct {
	connection Connection
}

// NewGPGKeyConnectionResolver creates a new GPGKeyConnectionResolver
func NewGPGKeyConnectionResolver(ctx context.Context, input *gpgkey.GetGPGKeysInput) (*GPGKeyConnectionResolver, error) {
	gpgKeyService := getServiceCatalog(ctx).GPGKeyService

	result, err := gpgKeyService.GetGPGKeys(ctx, input)
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
	return graphql.ID(r.gpgKey.GetGlobalID())
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

// GroupPath resolver
func (r *GPGKeyResolver) GroupPath() string {
	return r.gpgKey.GetGroupPath()
}

// ResourcePath resolver
func (r *GPGKeyResolver) ResourcePath() string {
	return r.gpgKey.GetResourcePath()
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
	GroupID          *string
	GroupPath        *string // DEPRECATED: use GroupID instead with a TRN
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
	groupID, err := toModelID(ctx, input.GroupPath, input.GroupID, types.GroupModelType)
	if err != nil {
		return nil, err
	}

	createdGPGKey, err := getServiceCatalog(ctx).GPGKeyService.CreateGPGKey(ctx, &gpgkey.CreateGPGKeyInput{
		GroupID:    groupID,
		ASCIIArmor: input.ASCIIArmor,
	})
	if err != nil {
		return nil, err
	}

	payload := GPGKeyMutationPayload{ClientMutationID: input.ClientMutationID, GPGKey: createdGPGKey, Problems: []Problem{}}
	return &GPGKeyMutationPayloadResolver{GPGKeyMutationPayload: payload}, nil
}

func deleteGPGKeyMutation(ctx context.Context, input *DeleteGPGKeyInput) (*GPGKeyMutationPayloadResolver, error) {
	serviceCatalog := getServiceCatalog(ctx)

	id, err := serviceCatalog.FetchModelID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	gpgKey, err := serviceCatalog.GPGKeyService.GetGPGKeyByID(ctx, id)
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

	if err := serviceCatalog.GPGKeyService.DeleteGPGKey(ctx, gpgKey); err != nil {
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
		return nil, errors.New("Wrong type")
	}

	return &gpgKey, nil
}

func gpgKeyBatchFunc(ctx context.Context, ids []string) (loader.DataBatch, error) {
	gpgKeys, err := getServiceCatalog(ctx).GPGKeyService.GetGPGKeysByIDs(ctx, ids)
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
