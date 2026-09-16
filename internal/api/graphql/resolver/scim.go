package resolver

import (
	"context"

	graphql "github.com/graph-gophers/graphql-go"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// SCIMTokenResolver resolves a SCIM token without exposing its nonce.
type SCIMTokenResolver struct {
	token *models.SCIMToken
}

// ID resolver
func (r *SCIMTokenResolver) ID() graphql.ID {
	return graphql.ID(r.token.GetGlobalID())
}

// Metadata resolver
func (r *SCIMTokenResolver) Metadata() *MetadataResolver {
	return &MetadataResolver{metadata: &r.token.Metadata}
}

// CreatedBy resolver
func (r *SCIMTokenResolver) CreatedBy() string {
	return r.token.CreatedBy
}

// CreateSCIMTokenInput contains the input for creating a SCIM token
type CreateSCIMTokenInput struct {
	ClientMutationID *string
	IdpIssuerURL     string
}

// CreateSCIMTokenPayload is the response payload for a SCIM token mutation.
type CreateSCIMTokenPayload struct {
	ClientMutationID *string
	SCIMToken        *models.SCIMToken
	Problems         []Problem
}

// CreateSCIMTokenPayloadResolver resolves a CreateSCIMTokenPayload.
type CreateSCIMTokenPayloadResolver struct {
	plaintextToken []byte
	CreateSCIMTokenPayload
}

// SCIMToken field resolver
func (r *CreateSCIMTokenPayloadResolver) SCIMToken() *SCIMTokenResolver {
	if r.CreateSCIMTokenPayload.SCIMToken == nil {
		return nil
	}

	return &SCIMTokenResolver{token: r.CreateSCIMTokenPayload.SCIMToken}
}

// TokenText field resolver
func (r *CreateSCIMTokenPayloadResolver) TokenText() *string {
	if r.plaintextToken == nil {
		return nil
	}

	tokenText := string(r.plaintextToken)
	return &tokenText
}

func scimTokenQuery(ctx context.Context) (*SCIMTokenResolver, error) {
	token, err := getServiceCatalog(ctx).SCIMService.GetSCIMToken(ctx)
	if err != nil {
		if errors.ErrorCode(err) == errors.ENotFound {
			return nil, nil
		}

		return nil, err
	}

	return &SCIMTokenResolver{token: token}, nil
}

func handleSCIMMutationProblem(e error, clientMutationID *string) (*CreateSCIMTokenPayloadResolver, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}

	payload := CreateSCIMTokenPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}
	return &CreateSCIMTokenPayloadResolver{CreateSCIMTokenPayload: payload}, nil
}

func createSCIMTokenMutation(ctx context.Context, input *CreateSCIMTokenInput) (*CreateSCIMTokenPayloadResolver, error) {
	output, err := getServiceCatalog(ctx).SCIMService.CreateSCIMToken(ctx, input.IdpIssuerURL)
	if err != nil {
		return nil, err
	}

	payload := CreateSCIMTokenPayload{
		ClientMutationID: input.ClientMutationID,
		SCIMToken:        output.SCIMToken,
		Problems:         []Problem{},
	}

	return &CreateSCIMTokenPayloadResolver{
		CreateSCIMTokenPayload: payload,
		plaintextToken:         output.PlaintextToken,
	}, nil
}
