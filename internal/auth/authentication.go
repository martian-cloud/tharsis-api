package auth

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/jwx/jwt"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
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
const (
	JobTokenType            string = "job"
	ServiceAccountTokenType string = "service_account"
	SCIMTokenType           string = "scim"
)

// Authenticator is used to authenticate JWT tokens
type Authenticator struct {
	userAuth  *UserAuth
	idp       *IdentityProvider
	dbClient  *db.Client
	issuerURL string
}

// NewAuthenticator creates a new Authenticator instance
func NewAuthenticator(userAuth *UserAuth, idp *IdentityProvider, dbClient *db.Client, issuerURL string) *Authenticator {
	return &Authenticator{
		userAuth:  userAuth,
		idp:       idp,
		dbClient:  dbClient,
		issuerURL: issuerURL,
	}
}

// Authenticate verifies the token and returns a Caller
func (a *Authenticator) Authenticate(ctx context.Context, tokenString string, useCache bool) (Caller, error) {
	if tokenString == "" {
		return nil, errors.NewError(errors.EUnauthorized, "Authentication token is missing")
	}

	decodedToken, err := jwt.Parse([]byte(tokenString))
	if err != nil {
		return nil, errors.NewError(errors.EUnauthorized, fmt.Sprintf("Failed to decode token %v", err))
	}

	if decodedToken.Issuer() == a.issuerURL {
		// This is a service account token
		output, vtErr := a.idp.VerifyToken(ctx, tokenString)
		if vtErr != nil {
			return nil, errors.NewError(errors.EUnauthorized, errorReason(vtErr))
		}

		// TODO: Update the if conditions to look for 'type' field instead
		// and use the enum strings defined above.
		if serviceAccountGID, ok := output.PrivateClaims["service_account_id"]; ok {
			serviceAccountID := gid.FromGlobalID(serviceAccountGID)
			return NewServiceAccountCaller(
				serviceAccountID,
				output.PrivateClaims["service_account_path"],
				newNamespaceMembershipAuthorizer(a.dbClient, nil, &serviceAccountID, useCache),
			), nil
		} else if jobID, ok := output.PrivateClaims["job_id"]; ok {
			return &JobCaller{
				JobID:       gid.FromGlobalID(jobID),
				RunID:       gid.FromGlobalID(output.PrivateClaims["run_id"]),
				WorkspaceID: gid.FromGlobalID(output.PrivateClaims["workspace_id"]),
				dbClient:    a.dbClient,
			}, nil
		} else if tokenType, ok := output.PrivateClaims["type"]; ok && tokenType == SCIMTokenType {
			scimCaller, sErr := a.verifySCIMTokenClaim(ctx, output.Token)
			if sErr != nil {
				return nil, errors.NewError(errors.EUnauthorized, errorReason(sErr))
			}
			return scimCaller, nil
		} else {
			return nil, errors.NewError(errors.EInternal, "Unsupported token type received")
		}
	}

	// This is a user token
	caller, err := a.userAuth.Authenticate(ctx, tokenString, useCache)
	if err != nil {
		return nil, errors.NewError(errors.EUnauthorized, errorReason(err))
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

	return NewSCIMCaller(a.dbClient), nil
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
