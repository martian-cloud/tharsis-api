package auth

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/ryanuber/go-glob"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/apiserver/config"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth/permissions"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

var (
	// acceptable skew for nbf and exp claims when verifying tokens
	federatedRegistryAcceptableTokenSkew = 5 * time.Second
)

// FederatedRegistryAuth handles authentication for federated registry tokens
type FederatedRegistryAuth struct {
	trustPolicies []config.FederatedRegistryTrustPolicy
	tokenVerifier OIDCTokenVerifier
	dbClient      *db.Client
	logger        logger.Logger
}

// NewFederatedRegistryAuth creates a new instance of FederatedRegistryAuth
func NewFederatedRegistryAuth(
	ctx context.Context,
	trustPolicies []config.FederatedRegistryTrustPolicy,
	logger logger.Logger,
	oidcConfigFetcher OpenIDConfigFetcher,
	dbClient *db.Client,
) *FederatedRegistryAuth {
	issuers := []string{}

	for _, policy := range trustPolicies {
		issuers = append(issuers, policy.IssuerURL)
	}

	oidcTokenVerifier := NewOIDCTokenVerifier(ctx, issuers, oidcConfigFetcher, true)

	return &FederatedRegistryAuth{trustPolicies, oidcTokenVerifier, dbClient, logger}
}

// Use checks if the token is a federated registry token and if the issuer is trusted
func (f *FederatedRegistryAuth) Use(token jwt.Token) bool {
	if typ, ok := token.Get("typ"); ok {
		if tokenTypeClaim, ok := typ.(string); ok {
			// Check token type claim
			if tokenTypeClaim == FederatedRegistryTokenType {
				// Check issuer
				issuer := token.Issuer()
				for _, policy := range f.trustPolicies {
					if policy.IssuerURL == issuer {
						return true
					}
				}
			}
		}
	}
	return false
}

// Authenticate verifies the token and returns a FederatedRegistryCaller
func (f *FederatedRegistryAuth) Authenticate(ctx context.Context, token string, _ bool) (Caller, error) {
	decodedToken, err := f.tokenVerifier.VerifyToken(ctx, token, []jwt.ValidateOption{jwt.WithAcceptableSkew(federatedRegistryAcceptableTokenSkew)})
	if err != nil {
		return nil, errors.New(errorReason(err), errors.WithErrorCode(errors.EUnauthorized))
	}
	// Verify that decoded token matches aud and sub from one of the trust policies
	issuer := decodedToken.Issuer()
	subject := decodedToken.Subject()
	audiences := decodedToken.Audience()

	matchingTrustPolicies := []*config.FederatedRegistryTrustPolicy{}
	for _, policy := range f.trustPolicies {
		if policy.IssuerURL != issuer {
			continue
		}
		if policy.Subject != nil && *policy.Subject != subject {
			continue
		}
		if policy.Audience != nil && !slices.Contains(audiences, *policy.Audience) {
			continue
		}
		matchingTrustPolicies = append(matchingTrustPolicies, &policy)
	}

	if len(matchingTrustPolicies) == 0 {
		return nil, errors.New(
			"no federated trust policies match the token issuer %s, subject %s, and audience %s",
			issuer,
			subject,
			strings.Join(audiences, ", "),
			errors.WithErrorCode(errors.EUnauthorized),
		)
	}

	return NewFederatedRegistryCaller(f.dbClient, matchingTrustPolicies, subject), nil
}

// FederatedRegistryCaller represents a federated registry subject
type FederatedRegistryCaller struct {
	dbClient      *db.Client
	trustPolicies []*config.FederatedRegistryTrustPolicy
	subject       string
}

// NewFederatedRegistryCaller returns a new FederatedRegistryCaller
func NewFederatedRegistryCaller(
	dbClient *db.Client,
	trustPolicies []*config.FederatedRegistryTrustPolicy,
	subject string,
) *FederatedRegistryCaller {
	return &FederatedRegistryCaller{
		dbClient:      dbClient,
		trustPolicies: trustPolicies,
		subject:       subject,
	}
}

// GetSubject returns the subject identifier for this caller
func (f *FederatedRegistryCaller) GetSubject() string {
	return f.subject
}

// IsAdmin returns true if the caller is an admin
func (f *FederatedRegistryCaller) IsAdmin() bool {
	return false
}

// UnauthorizedError returns the unauthorized error for this specific caller type
func (f *FederatedRegistryCaller) UnauthorizedError(_ context.Context, hasViewerAccess bool) error {
	// If subject has at least viewer permissions then return 403, if not, return 404
	if hasViewerAccess {
		return errors.New(
			"federated registry subject %s is not authorized to perform the requested operation",
			f.GetSubject(),
			errors.WithErrorCode(errors.EForbidden),
		)
	}

	return errors.New(
		"either the requested resource does not exist or the federated registry subject %s is not authorized to perform the requested operation",
		f.GetSubject(),
		errors.WithErrorCode(errors.ENotFound),
	)
}

// GetNamespaceAccessPolicy returns the namespace access policy for this caller
func (f *FederatedRegistryCaller) GetNamespaceAccessPolicy(_ context.Context) (*NamespaceAccessPolicy, error) {
	return &NamespaceAccessPolicy{
		// RootNamespaceIDs is empty to indicate the caller doesn't have access to any root namespaces.
		RootNamespaceIDs: []string{},
	}, nil
}

// RequirePermission will return an error if the caller doesn't have the specified permissions.
func (f *FederatedRegistryCaller) RequirePermission(ctx context.Context, _ permissions.Permission, _ ...func(*constraints),
) error {
	// Federated caller only supports read-only permissions on inheritable resources (i.e. modules and providers)
	return f.UnauthorizedError(ctx, false)
}

// RequireAccessToInheritableResource will return an error if caller doesn't have permissions to inherited resources.
func (f *FederatedRegistryCaller) RequireAccessToInheritableResource(ctx context.Context,
	resourceType permissions.ResourceType, checks ...func(*constraints),
) error {
	if resourceType != permissions.TerraformModuleResourceType && resourceType != permissions.TerraformProviderResourceType {
		return errors.New("unsupported resource type %s for federated registry caller", resourceType)
	}

	requestedNamespacePaths, err := f.getRequestedNamespacePaths(ctx, getConstraints(checks...))
	if err != nil {
		return err
	}

	for _, policy := range f.trustPolicies {
		if f.trustPolicySatisfied(requestedNamespacePaths, policy.GroupGlobPatterns) {
			return nil
		}
	}

	return f.UnauthorizedError(ctx, false)
}

func (f *FederatedRegistryCaller) trustPolicySatisfied(requestedNamespacePaths []string, globPatterns []string) bool {
	for _, requested := range requestedNamespacePaths {
		match := false
		for _, pattern := range globPatterns {
			match = glob.Glob(pattern, requested)
			if match {
				break
			}
		}
		// No match was found for this requested path
		if !match {
			return false
		}
	}

	return true
}

// requestedGroupPaths returns all group paths specified in the constraints.  It does NOT remove duplicates.
func (f *FederatedRegistryCaller) getRequestedNamespacePaths(ctx context.Context, constraints *constraints) ([]string, error) {
	result := []string{}
	// If constraint is for a group, get the path of that group.
	if constraints.groupID != nil {
		group, err := f.dbClient.Groups.GetGroupByID(ctx, *constraints.groupID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get group")
		}

		if group != nil {
			result = append(result, group.FullPath)
		}
	}

	// If constraint is for namespace paths, add the paths.
	if constraints.namespacePaths != nil {
		result = append(result, constraints.namespacePaths...)
	}

	if len(result) == 0 {
		return nil, errMissingConstraints
	}

	return result, nil
}
