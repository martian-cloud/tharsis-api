package resolver

import "context"

// SCIMTokenPayload is the response payload for a SCIM token mutation.
type SCIMTokenPayload struct {
	ClientMutationID *string
	Token            *string
	Problems         []Problem
}

func handleSCIMMutationProblem(e error, clientMutationID *string) (*SCIMTokenPayload, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}

	return &SCIMTokenPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}, nil
}

func createSCIMTokenMutation(ctx context.Context) (*SCIMTokenPayload, error) {
	tokenBytes, err := getServiceCatalog(ctx).SCIMService.CreateSCIMToken(ctx)
	if err != nil {
		return nil, err
	}

	stringToken := string(tokenBytes)

	return &SCIMTokenPayload{Token: &stringToken}, nil
}
