package resolver

import "context"

// CreateSCIMTokenInput contains the input for creating a SCIM token
type CreateSCIMTokenInput struct {
	ClientMutationID *string
	IdpIssuerURL     string
}

// CreateSCIMTokenPayload is the response payload for a SCIM token mutation.
type CreateSCIMTokenPayload struct {
	ClientMutationID *string
	Token            *string
	Problems         []Problem
}

func handleSCIMMutationProblem(e error, clientMutationID *string) (*CreateSCIMTokenPayload, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}

	return &CreateSCIMTokenPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}, nil
}

func createSCIMTokenMutation(ctx context.Context, input *CreateSCIMTokenInput) (*CreateSCIMTokenPayload, error) {
	tokenBytes, err := getServiceCatalog(ctx).SCIMService.CreateSCIMToken(ctx, input.IdpIssuerURL)
	if err != nil {
		return nil, err
	}

	stringToken := string(tokenBytes)

	return &CreateSCIMTokenPayload{ClientMutationID: input.ClientMutationID, Token: &stringToken}, nil
}
