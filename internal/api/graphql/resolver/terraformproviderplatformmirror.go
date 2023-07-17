package resolver

import (
	"context"
	"strconv"

	graphql "github.com/graph-gophers/graphql-go"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/providermirror"
)

/* TerraformProviderPlatformMirror Query Resolvers */

// TerraformProviderPlatformMirrorResolver resolves a providerPlatformMirror resource
type TerraformProviderPlatformMirrorResolver struct {
	platformMirror *models.TerraformProviderPlatformMirror
}

// ID resolver
func (r *TerraformProviderPlatformMirrorResolver) ID() graphql.ID {
	return graphql.ID(gid.ToGlobalID(gid.TerraformProviderPlatformMirrorType, r.platformMirror.Metadata.ID))
}

// OS resolver
func (r *TerraformProviderPlatformMirrorResolver) OS() string {
	return r.platformMirror.OS
}

// Arch resolver
func (r *TerraformProviderPlatformMirrorResolver) Arch() string {
	return r.platformMirror.Architecture
}

// Metadata resolver
func (r *TerraformProviderPlatformMirrorResolver) Metadata() *MetadataResolver {
	return &MetadataResolver{metadata: &r.platformMirror.Metadata}
}

// VersionMirror resolver
func (r *TerraformProviderPlatformMirrorResolver) VersionMirror(ctx context.Context) (*TerraformProviderVersionMirrorResolver, error) {
	versionMirror, err := loadTerraformProviderVersionMirror(ctx, r.platformMirror.VersionMirrorID)
	if err != nil {
		return nil, err
	}

	return &TerraformProviderVersionMirrorResolver{versionMirror: versionMirror}, nil
}

/* TerraformProviderPlatformMirror Mutation Resolvers */

// TerraformProviderPlatformMirrorMutationPayload is the response payload for a providerPlatformMirror mutation
type TerraformProviderPlatformMirrorMutationPayload struct {
	ClientMutationID *string
	PlatformMirror   *models.TerraformProviderPlatformMirror
	Problems         []Problem
}

// TerraformProviderPlatformMirrorMutationPayloadResolver resolves a TerraformProviderPlatformMirrorMutationPayload
type TerraformProviderPlatformMirrorMutationPayloadResolver struct {
	TerraformProviderPlatformMirrorMutationPayload
}

// PlatformMirror field resolver
func (r *TerraformProviderPlatformMirrorMutationPayloadResolver) PlatformMirror() *TerraformProviderPlatformMirrorResolver {
	if r.TerraformProviderPlatformMirrorMutationPayload.PlatformMirror == nil {
		return nil
	}

	return &TerraformProviderPlatformMirrorResolver{platformMirror: r.TerraformProviderPlatformMirrorMutationPayload.PlatformMirror}
}

// DeleteTerraformProviderPlatformMirrorInput contains the input for deleting a providerPlatformMirror
type DeleteTerraformProviderPlatformMirrorInput struct {
	ClientMutationID *string
	Metadata         *MetadataInput
	ID               string
}

func handleTerraformProviderPlatformMirrorMutationProblem(e error, clientMutationID *string) (*TerraformProviderPlatformMirrorMutationPayloadResolver, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}

	payload := TerraformProviderPlatformMirrorMutationPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}
	return &TerraformProviderPlatformMirrorMutationPayloadResolver{TerraformProviderPlatformMirrorMutationPayload: payload}, nil
}

func deleteTerraformProviderPlatformMirrorMutation(ctx context.Context, input *DeleteTerraformProviderPlatformMirrorInput) (*TerraformProviderPlatformMirrorMutationPayloadResolver, error) {
	service := getProviderMirrorService(ctx)

	platformMirror, err := service.GetProviderPlatformMirrorByID(ctx, gid.FromGlobalID(input.ID))
	if err != nil {
		return nil, err
	}

	// Check if resource version is specified
	if input.Metadata != nil {
		v, err := strconv.Atoi(input.Metadata.Version)
		if err != nil {
			return nil, err
		}

		platformMirror.Metadata.Version = v
	}

	toDelete := &providermirror.DeleteProviderPlatformMirrorInput{
		PlatformMirror: platformMirror,
	}

	if err := service.DeleteProviderPlatformMirror(ctx, toDelete); err != nil {
		return nil, err
	}

	payload := TerraformProviderPlatformMirrorMutationPayload{ClientMutationID: input.ClientMutationID, PlatformMirror: platformMirror, Problems: []Problem{}}
	return &TerraformProviderPlatformMirrorMutationPayloadResolver{TerraformProviderPlatformMirrorMutationPayload: payload}, nil
}
