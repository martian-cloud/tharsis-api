// Package auth package
package auth

//go:generate go tool mockery --name Authenticator --inpackage --case underscore
//go:generate go tool mockery --name tokenAuthenticator --inpackage --case underscore

import (
	"context"
	"fmt"

	"github.com/aws/smithy-go/ptr"
	"github.com/lestrrat-go/jwx/v2/jwt"
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
)

// Valid token types used as private claims for tokens
// issued by Tharsis.
// #nosec: G101 -- false flag.
const (
	JobTokenType               string = "job"
	ServiceAccountTokenType    string = "service_account"
	UserSessionAccessTokenType string = "user_session_access"
	UserSessionCSRFTokenType   string = "user_session_csrf"
	SCIMTokenType              string = "scim"
	VCSWorkspaceLinkTokenType  string = "vcs_workspace_link"
	FederatedRegistryTokenType string = "federated_registry"
)

// SCIM token claim names.
const (
	IDPIssuerURLClaimName = "idp_issuer_url"
)

// Authenticator is used to authenticate JWT tokens
type Authenticator interface {
	// Authenticate verifies the token and returns a Caller
	Authenticate(ctx context.Context, tokenString string, useCache bool) (Caller, error)
}

type tokenAuthenticator interface {
	Use(token jwt.Token) bool
	Authenticate(ctx context.Context, tokenString string, useCache bool) (Caller, error)
}

// Authenticator is used to authenticate JWT tokens
type authenticator struct {
	tokenAuthenticators []tokenAuthenticator
}

// NewAuthenticator creates a new Authenticator instance
func NewAuthenticator(
	userAuth *UserAuth,
	federatedRegistryAuth *FederatedRegistryAuth,
	signingKeyManager SigningKeyManager,
	dbClient *db.Client,
	maintenanceMonitor maintenance.Monitor,
	issuerURL string,
) Authenticator {
	return newAuthenticator(
		[]tokenAuthenticator{
			&tharsisIDPTokenAuthenticator{
				issuerURL:          issuerURL,
				signingKeyManager:  signingKeyManager,
				dbClient:           dbClient,
				maintenanceMonitor: maintenanceMonitor,
			},
			userAuth,
			federatedRegistryAuth,
		},
	)
}

func newAuthenticator(
	tokenAuthenticators []tokenAuthenticator,
) *authenticator {
	return &authenticator{
		tokenAuthenticators: tokenAuthenticators,
	}
}

// Authenticate verifies the token and returns a Caller
func (a *authenticator) Authenticate(ctx context.Context, tokenString string, useCache bool) (Caller, error) {
	if tokenString == "" {
		return nil, errors.New("authentication token is missing", errors.WithErrorCode(errors.EUnauthorized))
	}

	tokenBytes := []byte(tokenString)
	decodedToken, err := jwt.Parse(tokenBytes, jwt.WithVerify(false))
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode token", errors.WithErrorCode(errors.EUnauthorized))
	}

	for _, authenticator := range a.tokenAuthenticators {
		if authenticator.Use(decodedToken) {
			caller, err := authenticator.Authenticate(ctx, tokenString, useCache)
			if err != nil {
				return nil, err
			}
			return caller, nil
		}
	}

	return nil, errors.New("token issuer %s is not allowed", decodedToken.Issuer(), errors.WithErrorCode(errors.EUnauthorized))
}

type tharsisIDPTokenAuthenticator struct {
	issuerURL          string
	signingKeyManager  SigningKeyManager
	dbClient           *db.Client
	maintenanceMonitor maintenance.Monitor
}

func (t *tharsisIDPTokenAuthenticator) Use(token jwt.Token) bool {
	return token.Issuer() == t.issuerURL
}

func (t *tharsisIDPTokenAuthenticator) Authenticate(ctx context.Context, tokenString string, useCache bool) (Caller, error) {
	output, vtErr := t.signingKeyManager.VerifyToken(ctx, tokenString)
	if vtErr != nil {
		return nil, errors.New(errorReason(vtErr), errors.WithErrorCode(errors.EUnauthorized))
	}

	tokenType, ok := output.PrivateClaims["type"]
	if !ok {
		return nil, errors.New("failed to get token type", errors.WithErrorCode(errors.EUnauthorized))
	}

	switch tokenType {
	case UserSessionAccessTokenType:
		user, err := t.dbClient.Users.GetUserByID(ctx, gid.FromGlobalID(output.Token.Subject()))
		if err != nil {
			return nil, err
		}
		if user == nil {
			return nil, errors.New("user not found for token subject %s", output.Token.Subject(), errors.WithErrorCode(errors.EUnauthorized))
		}

		if !user.Active {
			return nil, errors.New(
				"User has been disabled",
				errors.WithErrorCode(errors.EUnauthorized),
			)
		}

		sessionGID := output.PrivateClaims[SessionIDClaim]

		return NewUserCaller(
			user,
			newNamespaceMembershipAuthorizer(t.dbClient, &user.Metadata.ID, nil, useCache),
			t.dbClient,
			t.maintenanceMonitor,
			ptr.String(gid.FromGlobalID(sessionGID)),
		), nil
	case ServiceAccountTokenType:
		serviceAccountID := gid.FromGlobalID(output.PrivateClaims["service_account_id"])
		return NewServiceAccountCaller(
			serviceAccountID,
			output.PrivateClaims["service_account_path"],
			newNamespaceMembershipAuthorizer(t.dbClient, nil, &serviceAccountID, useCache),
			t.dbClient,
			t.maintenanceMonitor,
		), nil
	case JobTokenType:
		return &JobCaller{
			JobID:       gid.FromGlobalID(output.PrivateClaims["job_id"]),
			JobTRN:      output.PrivateClaims["job_trn"],
			RunID:       gid.FromGlobalID(output.PrivateClaims["run_id"]),
			WorkspaceID: gid.FromGlobalID(output.PrivateClaims["workspace_id"]),
			dbClient:    t.dbClient,
		}, nil
	case SCIMTokenType:
		if sErr := t.verifySCIMTokenClaim(ctx, output.Token); sErr != nil {
			return nil, sErr
		}
		return NewSCIMCaller(t.dbClient, t.maintenanceMonitor, output.PrivateClaims[IDPIssuerURLClaimName]), nil
	case VCSWorkspaceLinkTokenType:
		vcsCaller, sErr := t.verifyVCSToken(ctx, output)
		if sErr != nil {
			return nil, sErr
		}
		return vcsCaller, nil
	default:
		return nil, errors.New("unsupported token type received", errors.WithErrorCode(errors.EUnauthorized))
	}
}

// verifySCIMToken verifies the JwtID field is known.
func (t *tharsisIDPTokenAuthenticator) verifySCIMTokenClaim(ctx context.Context, token jwt.Token) error {
	// Get the token claim to verify it is known.
	tokenClaim, err := t.dbClient.SCIMTokens.GetTokenByNonce(ctx, token.JwtID())
	if err != nil {
		return err
	}

	if tokenClaim == nil {
		return errors.New("scim token has an invalid jti claim", errors.WithErrorCode(errors.EUnauthorized))
	}

	return nil
}

// verifyVCSToken verifies a VCS token is known.
func (t *tharsisIDPTokenAuthenticator) verifyVCSToken(ctx context.Context, output *VerifyTokenOutput) (*VCSWorkspaceLinkCaller, error) {
	linkID, ok := output.PrivateClaims["link_id"]
	if !ok {
		return nil, errors.New("failed to get provider link id token claim", errors.WithErrorCode(errors.EUnauthorized))
	}

	link, err := t.dbClient.WorkspaceVCSProviderLinks.GetLinkByID(ctx, gid.FromGlobalID(linkID))
	if err != nil {
		return nil, err
	}

	if link == nil {
		return nil, errors.New("vcs token has invalid vcs provider link id", errors.WithErrorCode(errors.EUnauthorized))
	}

	if link.TokenNonce != output.Token.JwtID() {
		return nil, errors.New("vcs token has an invalid jti claim", errors.WithErrorCode(errors.EUnauthorized))
	}

	provider, err := t.dbClient.VCSProviders.GetProviderByID(ctx, link.ProviderID)
	if err != nil {
		return nil, err
	}

	if provider == nil {
		return nil, fmt.Errorf("failed to get vcs provider associated with link %s", link.Metadata.ID)
	}

	return NewVCSWorkspaceLinkCaller(
		provider,
		link,
		t.dbClient,
		t.maintenanceMonitor,
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
