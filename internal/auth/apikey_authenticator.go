package auth

import (
	"context"
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth/apikey"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/maintenance"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// apiKeyAuthenticator authenticates opaque API keys by looking up their hash.
type apiKeyAuthenticator struct {
	dbClient           *db.Client
	maintenanceMonitor maintenance.Monitor
}

func (a *apiKeyAuthenticator) Authenticate(ctx context.Context, key string) (Caller, error) {
	keyType, hash, err := apikey.Parse(key)
	if err != nil {
		return nil, err
	}

	switch keyType {
	case apikey.TypeSCIM:
		return a.authenticateSCIM(ctx, hash)
	case apikey.TypeVCS:
		return a.authenticateVCS(ctx, hash)
	default:
		return nil, errors.New("unsupported API key type", errors.WithErrorCode(errors.EUnauthorized))
	}
}

func (a *apiKeyAuthenticator) authenticateSCIM(ctx context.Context, hash string) (Caller, error) {
	token, err := a.dbClient.SCIMTokens.GetTokenByHash(ctx, hash)
	if err != nil {
		return nil, err
	}

	if token == nil {
		return nil, errors.New("scim token is unknown", errors.WithErrorCode(errors.EUnauthorized))
	}

	if token.ExpiresAt != nil && time.Now().UTC().After(*token.ExpiresAt) {
		return nil, errors.New("scim token is expired", errors.WithErrorCode(errors.EUnauthorized))
	}

	return NewSCIMCaller(a.maintenanceMonitor, token.IDPIssuerURL), nil
}

func (a *apiKeyAuthenticator) authenticateVCS(ctx context.Context, hash string) (Caller, error) {
	link, err := a.dbClient.WorkspaceVCSProviderLinks.GetLinkByTokenHash(ctx, hash)
	if err != nil {
		return nil, err
	}

	if link == nil {
		return nil, errors.New("vcs token is unknown", errors.WithErrorCode(errors.EUnauthorized))
	}

	provider, err := a.dbClient.VCSProviders.GetProviderByID(ctx, link.ProviderID)
	if err != nil {
		return nil, err
	}

	if provider == nil {
		return nil, errors.New("failed to get vcs provider associated with link %s", link.Metadata.ID)
	}

	return NewVCSWorkspaceLinkCaller(provider, link, a.dbClient, a.maintenanceMonitor), nil
}
