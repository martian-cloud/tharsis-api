// Package auth package
package auth

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/jwx/jwt"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/maintenance"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

var (
	errUnauthorized = "token is unauthorized"
	errExpired      = "token is expired"
	errNBFInvalid   = "token nbf validation failed"
	errIATInvalid   = "token iat validation failed"
	errNoTokenFound = "no token found"     // nolint
	errAlgoInvalid  = "algorithm mismatch" // nolint
)

// Valid token types used as private claims for tokens
// issued by Tharsis.
// #nosec: G101 -- false flag.
const (
	JobTokenType              string = "job"
	ServiceAccountTokenType   string = "service_account"
	SCIMTokenType             string = "scim"
	VCSWorkspaceLinkTokenType string = "vcs_workspace_link"
)

// Authenticator is used to authenticate JWT tokens
type Authenticator struct {
	userAuth           *UserAuth
	idp                *IdentityProvider
	dbClient           *db.Client
	maintenanceMonitor maintenance.Monitor
	issuerURL          string
}

// NewAuthenticator creates a new Authenticator instance
func NewAuthenticator(
	userAuth *UserAuth,
	idp *IdentityProvider,
	dbClient *db.Client,
	maintenanceMonitor maintenance.Monitor,
	issuerURL string,
) *Authenticator {
	return &Authenticator{
		userAuth:           userAuth,
		idp:                idp,
		dbClient:           dbClient,
		maintenanceMonitor: maintenanceMonitor,
		issuerURL:          issuerURL,
	}
}

// Authenticate verifies the token and returns a Caller
func (a *Authenticator) Authenticate(ctx context.Context, tokenString string, useCache bool) (Caller, error) {
	if tokenString == "" {
		return nil, errors.New("Authentication token is missing", errors.WithErrorCode(errors.EUnauthorized))
	}

	decodedToken, err := jwt.Parse([]byte(tokenString))
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode token", errors.WithErrorCode(errors.EUnauthorized))
	}

	if decodedToken.Issuer() == a.issuerURL {
		// This is a service account token
		output, vtErr := a.idp.VerifyToken(ctx, tokenString)
		if vtErr != nil {
			return nil, errors.New(errorReason(vtErr), errors.WithErrorCode(errors.EUnauthorized))
		}

		tokenType, ok := output.PrivateClaims["type"]
		if !ok {
			return nil, fmt.Errorf("failed to get token type")
		}

		switch tokenType {
		case ServiceAccountTokenType:
			serviceAccountID := gid.FromGlobalID(output.PrivateClaims["service_account_id"])
			return NewServiceAccountCaller(
				serviceAccountID,
				output.PrivateClaims["service_account_path"],
				newNamespaceMembershipAuthorizer(a.dbClient, nil, &serviceAccountID, useCache),
				a.dbClient,
				a.maintenanceMonitor,
			), nil
		case JobTokenType:
			return &JobCaller{
				JobID:       gid.FromGlobalID(output.PrivateClaims["job_id"]),
				RunID:       gid.FromGlobalID(output.PrivateClaims["run_id"]),
				WorkspaceID: gid.FromGlobalID(output.PrivateClaims["workspace_id"]),
				dbClient:    a.dbClient,
			}, nil
		case SCIMTokenType:
			scimCaller, sErr := a.verifySCIMTokenClaim(ctx, output.Token)
			if sErr != nil {
				return nil, errors.New(errorReason(sErr), errors.WithErrorCode(errors.EUnauthorized))
			}
			return scimCaller, nil
		case VCSWorkspaceLinkTokenType:
			vcsCaller, sErr := a.verifyVCSToken(ctx, output)
			if sErr != nil {
				return nil, errors.New(errorReason(sErr), errors.WithErrorCode(errors.EUnauthorized))
			}
			return vcsCaller, nil
		default:
			return nil, errors.New("Unsupported token type received")
		}
	}

	// This is a user token
	caller, err := a.userAuth.Authenticate(ctx, tokenString, useCache)
	if err != nil {
		return nil, errors.New(errorReason(err), errors.WithErrorCode(errors.EUnauthorized))
	}

	return caller, nil
}

// verifySCIMToken verifies the JwtID field is known.
func (a *Authenticator) verifySCIMTokenClaim(ctx context.Context, token jwt.Token) (*SCIMCaller, error) {
	// Get the token claim to verify it is known.
	tokenClaim, err := a.dbClient.SCIMTokens.GetTokenByNonce(ctx, token.JwtID())
	if err != nil {
		return nil, err
	}

	if tokenClaim == nil {
		return nil, fmt.Errorf("scim token has an invalid jti claim")
	}

	return NewSCIMCaller(a.dbClient, a.maintenanceMonitor), nil
}

// verifyVCSToken verifies a VCS token is known.
func (a *Authenticator) verifyVCSToken(ctx context.Context, output *VerifyTokenOutput) (*VCSWorkspaceLinkCaller, error) {
	linkID, ok := output.PrivateClaims["link_id"]
	if !ok {
		return nil, fmt.Errorf("failed to get provider link id token claim")
	}

	link, err := a.dbClient.WorkspaceVCSProviderLinks.GetLinkByID(ctx, gid.FromGlobalID(linkID))
	if err != nil {
		return nil, err
	}

	if link == nil {
		return nil, fmt.Errorf("vcs token has invalid vcs provider link id")
	}

	if link.TokenNonce != output.Token.JwtID() {
		return nil, fmt.Errorf("vcs token has an invalid jti claim")
	}

	provider, err := a.dbClient.VCSProviders.GetProviderByID(ctx, link.ProviderID)
	if err != nil {
		return nil, err
	}

	if provider == nil {
		return nil, fmt.Errorf("failed to get provider")
	}

	return NewVCSWorkspaceLinkCaller(
		provider,
		link,
		a.dbClient,
		a.maintenanceMonitor,
	), nil
}

// ErrorReason will normalize the error message
func errorReason(err error) string {
	switch err.Error() {
	case "exp not satisfied":
		return errExpired
	case "iat not satisfied":
		return errIATInvalid
	case "nbf not satisfied":
		return errNBFInvalid
	default:
		return errUnauthorized
	}
}
