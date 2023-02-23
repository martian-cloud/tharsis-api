package resolver

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/moduleregistry"

	graphql "github.com/graph-gophers/graphql-go"
)

/* TerraformModuleAttestation Query Resolvers */

// TerraformModuleAttestationConnectionQueryArgs are used to query a module attestation connection
type TerraformModuleAttestationConnectionQueryArgs struct {
	ConnectionQueryArgs
	Digest *string
}

// TerraformModuleAttestationEdgeResolver resolves module attestation edges
type TerraformModuleAttestationEdgeResolver struct {
	edge Edge
}

// Cursor returns an opaque cursor
func (r *TerraformModuleAttestationEdgeResolver) Cursor() (string, error) {
	moduleAttestation, ok := r.edge.Node.(models.TerraformModuleAttestation)
	if !ok {
		return "", errors.NewError(errors.EInternal, "Failed to convert node type")
	}
	cursor, err := r.edge.CursorFunc(&moduleAttestation)
	return *cursor, err
}

// Node returns a module attestation node
func (r *TerraformModuleAttestationEdgeResolver) Node() (*TerraformModuleAttestationResolver, error) {
	moduleAttestation, ok := r.edge.Node.(models.TerraformModuleAttestation)
	if !ok {
		return nil, errors.NewError(errors.EInternal, "Failed to convert node type")
	}

	return &TerraformModuleAttestationResolver{moduleAttestation: &moduleAttestation}, nil
}

// TerraformModuleAttestationConnectionResolver resolves a module attestation connection
type TerraformModuleAttestationConnectionResolver struct {
	connection Connection
}

// NewTerraformModuleAttestationConnectionResolver creates a new TerraformModuleAttestationConnectionResolver
func NewTerraformModuleAttestationConnectionResolver(ctx context.Context, input *moduleregistry.GetModuleAttestationsInput) (*TerraformModuleAttestationConnectionResolver, error) {
	service := getModuleRegistryService(ctx)

	result, err := service.GetModuleAttestations(ctx, input)
	if err != nil {
		return nil, err
	}

	moduleAttestations := result.ModuleAttestations

	// Create edges
	edges := make([]Edge, len(moduleAttestations))
	for i, moduleAttestation := range moduleAttestations {
		edges[i] = Edge{CursorFunc: result.PageInfo.Cursor, Node: moduleAttestation}
	}

	pageInfo := PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
	}

	if len(moduleAttestations) > 0 {
		var err error
		pageInfo.StartCursor, err = result.PageInfo.Cursor(&moduleAttestations[0])
		if err != nil {
			return nil, err
		}

		pageInfo.EndCursor, err = result.PageInfo.Cursor(&moduleAttestations[len(edges)-1])
		if err != nil {
			return nil, err
		}
	}

	connection := Connection{
		TotalCount: result.PageInfo.TotalCount,
		PageInfo:   pageInfo,
		Edges:      edges,
	}

	return &TerraformModuleAttestationConnectionResolver{connection: connection}, nil
}

// TotalCount returns the total result count for the connection
func (r *TerraformModuleAttestationConnectionResolver) TotalCount() int32 {
	return r.connection.TotalCount
}

// PageInfo returns the connection page information
func (r *TerraformModuleAttestationConnectionResolver) PageInfo() *PageInfoResolver {
	return &PageInfoResolver{pageInfo: r.connection.PageInfo}
}

// Edges returns the connection edges
func (r *TerraformModuleAttestationConnectionResolver) Edges() *[]*TerraformModuleAttestationEdgeResolver {
	resolvers := make([]*TerraformModuleAttestationEdgeResolver, len(r.connection.Edges))
	for i, edge := range r.connection.Edges {
		resolvers[i] = &TerraformModuleAttestationEdgeResolver{edge: edge}
	}
	return &resolvers
}

// TerraformModuleAttestationResolver resolves a module attestation resource
type TerraformModuleAttestationResolver struct {
	moduleAttestation *models.TerraformModuleAttestation
}

// ID resolver
func (r *TerraformModuleAttestationResolver) ID() graphql.ID {
	return graphql.ID(gid.ToGlobalID(gid.TerraformModuleAttestationType, r.moduleAttestation.Metadata.ID))
}

// Metadata resolver
func (r *TerraformModuleAttestationResolver) Metadata() *MetadataResolver {
	return &MetadataResolver{metadata: &r.moduleAttestation.Metadata}
}

// SchemaType resolver
func (r *TerraformModuleAttestationResolver) SchemaType() string {
	return r.moduleAttestation.SchemaType
}

// PredicateType resolver
func (r *TerraformModuleAttestationResolver) PredicateType() string {
	return r.moduleAttestation.PredicateType
}

// Digests resolver
func (r *TerraformModuleAttestationResolver) Digests() []string {
	return r.moduleAttestation.Digests
}

// Description resolver
func (r *TerraformModuleAttestationResolver) Description() string {
	return r.moduleAttestation.Description
}

// CreatedBy resolver
func (r *TerraformModuleAttestationResolver) CreatedBy() string {
	return r.moduleAttestation.CreatedBy
}

// Module resolver
func (r *TerraformModuleAttestationResolver) Module(ctx context.Context) (*TerraformModuleResolver, error) {
	module, err := loadTerraformModule(ctx, r.moduleAttestation.ModuleID)
	if err != nil {
		return nil, err
	}
	return &TerraformModuleResolver{module: module}, nil
}

// Data resolver
func (r *TerraformModuleAttestationResolver) Data() string {
	return r.moduleAttestation.Data
}

/* TerraformModuleAttestation Mutation Resolvers */

// TerraformModuleAttestationMutationPayload is the response payload for module attestation mutation
type TerraformModuleAttestationMutationPayload struct {
	ClientMutationID  *string
	ModuleAttestation *models.TerraformModuleAttestation
	Problems          []Problem
}

// TerraformModuleAttestationMutationPayloadResolver resolves a TerraformModuleAttestationMutationPayload
type TerraformModuleAttestationMutationPayloadResolver struct {
	TerraformModuleAttestationMutationPayload
}

// ModuleAttestation field resolver
func (r *TerraformModuleAttestationMutationPayloadResolver) ModuleAttestation() *TerraformModuleAttestationResolver {
	if r.TerraformModuleAttestationMutationPayload.ModuleAttestation == nil {
		return nil
	}
	return &TerraformModuleAttestationResolver{moduleAttestation: r.TerraformModuleAttestationMutationPayload.ModuleAttestation}
}

// CreateTerraformModuleAttestationInput contains the input for creating a moduleAttestation
type CreateTerraformModuleAttestationInput struct {
	ClientMutationID *string
	ModulePath       string
	Description      *string
	AttestationData  string
}

// UpdateTerraformModuleAttestationInput contains the input for updating a moduleAttestation
type UpdateTerraformModuleAttestationInput struct {
	ClientMutationID *string
	Description      string
	ID               string
}

// DeleteTerraformModuleAttestationInput contains the input for deleting a moduleAttestation
type DeleteTerraformModuleAttestationInput struct {
	ClientMutationID *string
	ID               string
}

func handleTerraformModuleAttestationMutationProblem(e error, clientMutationID *string) (*TerraformModuleAttestationMutationPayloadResolver, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}
	payload := TerraformModuleAttestationMutationPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}
	return &TerraformModuleAttestationMutationPayloadResolver{TerraformModuleAttestationMutationPayload: payload}, nil
}

func createTerraformModuleAttestationMutation(ctx context.Context, input *CreateTerraformModuleAttestationInput) (*TerraformModuleAttestationMutationPayloadResolver, error) {
	service := getModuleRegistryService(ctx)

	module, err := service.GetModuleByPath(ctx, input.ModulePath)
	if err != nil {
		return nil, err
	}

	createOptions := moduleregistry.CreateModuleAttestationInput{
		ModuleID:        module.Metadata.ID,
		AttestationData: input.AttestationData,
	}

	if input.Description != nil {
		createOptions.Description = *input.Description
	}

	moduleAttestation, err := service.CreateModuleAttestation(ctx, &createOptions)
	if err != nil {
		return nil, err
	}

	payload := TerraformModuleAttestationMutationPayload{ClientMutationID: input.ClientMutationID, ModuleAttestation: moduleAttestation, Problems: []Problem{}}
	return &TerraformModuleAttestationMutationPayloadResolver{TerraformModuleAttestationMutationPayload: payload}, nil
}

func updateTerraformModuleAttestationMutation(ctx context.Context, input *UpdateTerraformModuleAttestationInput) (*TerraformModuleAttestationMutationPayloadResolver, error) {
	service := getModuleRegistryService(ctx)

	attestation, err := service.GetModuleAttestationByID(ctx, gid.FromGlobalID(input.ID))
	if err != nil {
		return nil, err
	}

	attestation.Description = input.Description

	updatedModuleAttestation, err := service.UpdateModuleAttestation(ctx, attestation)
	if err != nil {
		return nil, err
	}

	payload := TerraformModuleAttestationMutationPayload{ClientMutationID: input.ClientMutationID, ModuleAttestation: updatedModuleAttestation, Problems: []Problem{}}
	return &TerraformModuleAttestationMutationPayloadResolver{TerraformModuleAttestationMutationPayload: payload}, nil
}

func deleteTerraformModuleAttestationMutation(ctx context.Context, input *DeleteTerraformModuleAttestationInput) (*TerraformModuleAttestationMutationPayloadResolver, error) {
	service := getModuleRegistryService(ctx)

	moduleAttestation, err := service.GetModuleAttestationByID(ctx, gid.FromGlobalID(input.ID))
	if err != nil {
		return nil, err
	}

	if err := service.DeleteModuleAttestation(ctx, moduleAttestation); err != nil {
		return nil, err
	}

	payload := TerraformModuleAttestationMutationPayload{ClientMutationID: input.ClientMutationID, ModuleAttestation: moduleAttestation, Problems: []Problem{}}
	return &TerraformModuleAttestationMutationPayloadResolver{TerraformModuleAttestationMutationPayload: payload}, nil
}
