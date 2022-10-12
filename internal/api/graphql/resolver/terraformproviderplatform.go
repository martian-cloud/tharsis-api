package resolver

import (
	"context"
	"strconv"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/providerregistry"

	graphql "github.com/graph-gophers/graphql-go"
)

/* TerraformProviderPlatform Query Resolvers */

// TerraformProviderPlatformResolver resolves a providerPlatform resource
type TerraformProviderPlatformResolver struct {
	providerPlatform *models.TerraformProviderPlatform
}

// ID resolver
func (r *TerraformProviderPlatformResolver) ID() graphql.ID {
	return graphql.ID(gid.ToGlobalID(gid.TerraformProviderPlatformType, r.providerPlatform.Metadata.ID))
}

// OS resolver
func (r *TerraformProviderPlatformResolver) OS() string {
	return r.providerPlatform.OperatingSystem
}

// Arch resolver
func (r *TerraformProviderPlatformResolver) Arch() string {
	return r.providerPlatform.Architecture
}

// SHASum resolver
func (r *TerraformProviderPlatformResolver) SHASum() string {
	return r.providerPlatform.SHASum
}

// Filename resolver
func (r *TerraformProviderPlatformResolver) Filename() string {
	return r.providerPlatform.Filename
}

// BinaryUploaded resolver
func (r *TerraformProviderPlatformResolver) BinaryUploaded() bool {
	return r.providerPlatform.BinaryUploaded
}

// Metadata resolver
func (r *TerraformProviderPlatformResolver) Metadata() *MetadataResolver {
	return &MetadataResolver{metadata: &r.providerPlatform.Metadata}
}

// ProviderVersion resolver
func (r *TerraformProviderPlatformResolver) ProviderVersion(ctx context.Context) (*TerraformProviderVersionResolver, error) {
	providerVersion, err := loadTerraformProviderVersion(ctx, r.providerPlatform.ProviderVersionID)
	if err != nil {
		return nil, err
	}

	return &TerraformProviderVersionResolver{providerVersion: providerVersion}, nil
}

/* TerraformProviderPlatform Mutation Resolvers */

// TerraformProviderPlatformMutationPayload is the response payload for a providerPlatform mutation
type TerraformProviderPlatformMutationPayload struct {
	ClientMutationID *string
	ProviderPlatform *models.TerraformProviderPlatform
	Problems         []Problem
}

// TerraformProviderPlatformMutationPayloadResolver resolvers a TerraformProviderPlatformMutationPayload
type TerraformProviderPlatformMutationPayloadResolver struct {
	TerraformProviderPlatformMutationPayload
}

// ProviderPlatform field resolver
func (r *TerraformProviderPlatformMutationPayloadResolver) ProviderPlatform(ctx context.Context) *TerraformProviderPlatformResolver {
	if r.TerraformProviderPlatformMutationPayload.ProviderPlatform == nil {
		return nil
	}
	return &TerraformProviderPlatformResolver{providerPlatform: r.TerraformProviderPlatformMutationPayload.ProviderPlatform}
}

// CreateTerraformProviderPlatformInput contains the input for creating a new providerPlatform
type CreateTerraformProviderPlatformInput struct {
	ClientMutationID  *string
	ProviderVersionID string
	OS                string
	Arch              string
	SHASum            string
	Filename          string
}

// DeleteTerraformProviderPlatformInput contains the input for deleting a providerPlatform
type DeleteTerraformProviderPlatformInput struct {
	ClientMutationID *string
	Metadata         *MetadataInput
	ID               string
}

func handleTerraformProviderPlatformMutationProblem(e error, clientMutationID *string) (*TerraformProviderPlatformMutationPayloadResolver, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}
	payload := TerraformProviderPlatformMutationPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}
	return &TerraformProviderPlatformMutationPayloadResolver{TerraformProviderPlatformMutationPayload: payload}, nil
}

func createTerraformProviderPlatformMutation(ctx context.Context, input *CreateTerraformProviderPlatformInput) (*TerraformProviderPlatformMutationPayloadResolver, error) {
	service := getProviderRegistryService(ctx)

	createdProviderPlatform, err := service.CreateProviderPlatform(ctx, &providerregistry.CreateProviderPlatformInput{
		ProviderVersionID: gid.FromGlobalID(input.ProviderVersionID),
		OperatingSystem:   input.OS,
		Architecture:      input.Arch,
		SHASum:            input.SHASum,
		Filename:          input.Filename,
	})
	if err != nil {
		return nil, err
	}

	payload := TerraformProviderPlatformMutationPayload{ClientMutationID: input.ClientMutationID, ProviderPlatform: createdProviderPlatform, Problems: []Problem{}}
	return &TerraformProviderPlatformMutationPayloadResolver{TerraformProviderPlatformMutationPayload: payload}, nil
}

func deleteTerraformProviderPlatformMutation(ctx context.Context, input *DeleteTerraformProviderPlatformInput) (*TerraformProviderPlatformMutationPayloadResolver, error) {
	service := getProviderRegistryService(ctx)

	providerPlatform, err := service.GetProviderPlatformByID(ctx, gid.FromGlobalID(input.ID))
	if err != nil {
		return nil, err
	}

	// Check if resource version is specified
	if input.Metadata != nil {
		v, err := strconv.Atoi(input.Metadata.Version)
		if err != nil {
			return nil, err
		}

		providerPlatform.Metadata.Version = v
	}

	if err := service.DeleteProviderPlatform(ctx, providerPlatform); err != nil {
		return nil, err
	}

	payload := TerraformProviderPlatformMutationPayload{ClientMutationID: input.ClientMutationID, ProviderPlatform: providerPlatform, Problems: []Problem{}}
	return &TerraformProviderPlatformMutationPayloadResolver{TerraformProviderPlatformMutationPayload: payload}, nil
}
