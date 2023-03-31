// Package vcs package
package vcs

//go:generate mockery --name Provider --inpackage --case underscore

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/vcs/github"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/vcs/gitlab"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/vcs/types"
)

// Provider handles the logic for a specific type of vcs provider.
type Provider interface {
	DefaultURL() url.URL
	MergeRequestActionIsSupported(action string) bool
	ToVCSEventType(input *types.ToVCSEventTypeInput) models.VCSEventType
	BuildOAuthAuthorizationURL(input *types.BuildOAuthAuthorizationURLInput) (string, error)
	BuildRepositoryURL(input *types.BuildRepositoryURLInput) (string, error)
	TestConnection(ctx context.Context, input *types.TestConnectionInput) error
	GetProject(ctx context.Context, input *types.GetProjectInput) (*types.GetProjectPayload, error)
	GetDiff(ctx context.Context, input *types.GetDiffInput) (*types.GetDiffsPayload, error)
	GetDiffs(ctx context.Context, input *types.GetDiffsInput) (*types.GetDiffsPayload, error)
	GetArchive(ctx context.Context, input *types.GetArchiveInput) (*http.Response, error)
	CreateAccessToken(ctx context.Context, input *types.CreateAccessTokenInput) (*types.AccessTokenPayload, error)
	CreateWebhook(ctx context.Context, input *types.CreateWebhookInput) (*types.WebhookPayload, error)
	DeleteWebhook(ctx context.Context, input *types.DeleteWebhookInput) error
}

// NewVCSProviderMap returns a map containing a handler for each VCS provider type.
func NewVCSProviderMap(
	ctx context.Context,
	logger logger.Logger,
	client *http.Client,
	tharsisURL string,
) (
	map[models.VCSProviderType]Provider,
	error,
) {
	gitLabHandler, err := gitlab.New(ctx, logger, client, tharsisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize %s vcs provider handler %v", models.GitLabProviderType, err)
	}

	gitHubHandler, err := github.New(ctx, logger, client, tharsisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize %s vcs provider handler %v", models.GitLabProviderType, err)
	}

	return map[models.VCSProviderType]Provider{
		models.GitLabProviderType: gitLabHandler,
		models.GitHubProviderType: gitHubHandler,
	}, nil
}
