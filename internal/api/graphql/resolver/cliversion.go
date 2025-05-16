package resolver

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/cli"
)

/* TerraformCLIVersion query resolvers */

// TerraformCLIVersionsResolver resolves TerraformCLIVersions.
type TerraformCLIVersionsResolver struct {
	versions []string
}

// Versions resolver
func (r *TerraformCLIVersionsResolver) Versions() []string {
	return r.versions
}

func terraformCLIVersionsQuery(ctx context.Context) (*TerraformCLIVersionsResolver, error) {
	cliVersions, err := getServiceCatalog(ctx).CLIService.GetTerraformCLIVersions(ctx)
	if err != nil {
		return nil, err
	}

	return &TerraformCLIVersionsResolver{versions: cliVersions}, nil
}

// TerraformCLIMutationPayload is the response payload for a Terraform CLI mutation.
type TerraformCLIMutationPayload struct {
	ClientMutationID *string
	DownloadURL      string
	Problems         []Problem
}

func handleTerraformCLIMutationProblem(e error, clientMutationID *string) (*TerraformCLIMutationPayload, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}

	return &TerraformCLIMutationPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}, nil
}

// CreateTerraformCLIDownloadURLInput is the input for createTerraformCLIDownloadURLMutation.
type CreateTerraformCLIDownloadURLInput struct {
	ClientMutationID *string
	Version          string
	OS               string
	Architecture     string
}

func createTerraformCLIDownloadURLMutation(ctx context.Context,
	input *CreateTerraformCLIDownloadURLInput,
) (*TerraformCLIMutationPayload, error) {
	// Prepare input.
	downloadInput := &cli.TerraformCLIVersionsInput{
		Version:      input.Version,
		OS:           input.OS,
		Architecture: input.Architecture,
	}

	downloadURL, err := getServiceCatalog(ctx).CLIService.CreateTerraformCLIDownloadURL(ctx, downloadInput)
	if err != nil {
		return nil, err
	}

	return &TerraformCLIMutationPayload{DownloadURL: downloadURL}, nil
}
