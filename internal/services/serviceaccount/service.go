package serviceaccount

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/lestrrat-go/jwx/jwa"
	"github.com/lestrrat-go/jwx/jwk"
	"github.com/lestrrat-go/jwx/jwt"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

var (
	failedLoginError = errors.NewError(errors.EUnauthorized, "Failed to login to service account due to one of the "+
		"following reasons: the service account does not exist; the JWT token used to login is invalid; the issuer "+
		"for the token is not a valid issuer.")
)

// LoginInput for logging into a service account
type LoginInput struct {
	// ServiceAccount ID or resource path
	ServiceAccount string
	Token          []byte
}

// LoginResponse returned after logging into a service account
type LoginResponse struct {
	Token []byte
}

// GetServiceAccountsInput is the input for querying a list of service accounts
type GetServiceAccountsInput struct {
	// Sort specifies the field to sort on and direction
	Sort *db.ServiceAccountSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *db.PaginationOptions
	// Search returns only the service accounts with a name or resource path that starts with the value of search
	Search *string
	// NamespacePath is the namespace to return service accounts for
	NamespacePath string
	// IncludeInherited includes inherited services accounts in the result
	IncludeInherited bool
}

// Service implements all service account related functionality
type Service interface {
	GetServiceAccountByPath(ctx context.Context, path string) (*models.ServiceAccount, error)
	GetServiceAccountByID(ctx context.Context, id string) (*models.ServiceAccount, error)
	GetServiceAccounts(ctx context.Context, input *GetServiceAccountsInput) (*db.ServiceAccountsResult, error)
	GetServiceAccountsByIDs(ctx context.Context, idList []string) ([]models.ServiceAccount, error)
	CreateServiceAccount(ctx context.Context, input *models.ServiceAccount) (*models.ServiceAccount, error)
	UpdateServiceAccount(ctx context.Context, serviceAccount *models.ServiceAccount) (*models.ServiceAccount, error)
	DeleteServiceAccount(ctx context.Context, serviceAccount *models.ServiceAccount) error
	Login(ctx context.Context, input *LoginInput) (*LoginResponse, error)
}

type service struct {
	logger        logger.Logger
	dbClient      *db.Client
	idp           *auth.IdentityProvider
	getKeySetFunc func(ctx context.Context, issuer string) (jwk.Set, error)
}

// NewService creates an instance of Service
func NewService(
	logger logger.Logger,
	dbClient *db.Client,
	idp *auth.IdentityProvider,
) Service {
	return newService(
		logger,
		dbClient,
		idp,
		getKeySet,
	)
}

func newService(
	logger logger.Logger,
	dbClient *db.Client,
	idp *auth.IdentityProvider,
	getKeySetFunc func(ctx context.Context, issuer string) (jwk.Set, error),
) Service {
	return &service{
		logger:        logger,
		dbClient:      dbClient,
		idp:           idp,
		getKeySetFunc: getKeySetFunc,
	}
}

func (s *service) GetServiceAccounts(ctx context.Context, input *GetServiceAccountsInput) (*db.ServiceAccountsResult, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	if err = caller.RequireAccessToNamespace(ctx, input.NamespacePath, models.ViewerRole); err != nil {
		return nil, err
	}

	filter := &db.ServiceAccountFilter{
		Search: input.Search,
	}

	if input.IncludeInherited {
		pathParts := strings.Split(input.NamespacePath, "/")

		paths := []string{}
		for len(pathParts) > 0 {
			paths = append(paths, strings.Join(pathParts, "/"))
			// Remove last element
			pathParts = pathParts[:len(pathParts)-1]
		}

		filter.NamespacePaths = paths
	} else {
		// This will return an empty result for workspace namespaces because workspaces
		// don't have service accounts directly associated (i.e. only group namespaces do)
		filter.NamespacePaths = []string{input.NamespacePath}
	}

	result, err := s.dbClient.ServiceAccounts.GetServiceAccounts(ctx, &db.GetServiceAccountsInput{
		Sort:              input.Sort,
		PaginationOptions: input.PaginationOptions,
		Filter:            filter,
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *service) GetServiceAccountsByIDs(ctx context.Context, idList []string) ([]models.ServiceAccount, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	result, err := s.dbClient.ServiceAccounts.GetServiceAccounts(ctx, &db.GetServiceAccountsInput{
		Filter: &db.ServiceAccountFilter{
			ServiceAccountIDs: idList,
		},
	})
	if err != nil {
		return nil, err
	}

	namespaces := []string{}
	for _, sa := range result.ServiceAccounts {
		parts := strings.Split(sa.ResourcePath, "/")
		namespaces = append(namespaces, strings.Join(parts[:len(parts)-1], "/"))
	}

	for _, ns := range namespaces {
		if err := caller.RequireAccessToInheritedNamespaceResource(ctx, ns); err != nil {
			return nil, err
		}
	}

	return result.ServiceAccounts, nil
}

func (s *service) DeleteServiceAccount(ctx context.Context, serviceAccount *models.ServiceAccount) error {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return err
	}

	if err := caller.RequireAccessToGroup(ctx, serviceAccount.GroupID, models.DeployerRole); err != nil {
		return err
	}

	s.logger.Infow("Requested deletion of a service account.",
		"caller", caller.GetSubject(),
		"groupID", serviceAccount.GroupID,
		"serviceAccountID", serviceAccount.Metadata.ID,
	)
	return s.dbClient.ServiceAccounts.DeleteServiceAccount(ctx, serviceAccount)
}

func (s *service) GetServiceAccountByPath(ctx context.Context, path string) (*models.ServiceAccount, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	// Get serviceAccount from DB
	serviceAccount, err := s.dbClient.ServiceAccounts.GetServiceAccountByPath(ctx, path)
	if err != nil {
		return nil, err
	}

	if serviceAccount == nil {
		return nil, errors.NewError(errors.ENotFound, fmt.Sprintf("service account with path %s not found", path))
	}

	if err := caller.RequireAccessToInheritedGroupResource(ctx, serviceAccount.GroupID); err != nil {
		return nil, err
	}

	return serviceAccount, nil
}

func (s *service) GetServiceAccountByID(ctx context.Context, id string) (*models.ServiceAccount, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	// Get serviceAccount from DB
	serviceAccount, err := s.dbClient.ServiceAccounts.GetServiceAccountByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if serviceAccount == nil {
		return nil, errors.NewError(errors.ENotFound, fmt.Sprintf("service account with ID %s not found", id))
	}

	if err := caller.RequireAccessToInheritedGroupResource(ctx, serviceAccount.GroupID); err != nil {
		return nil, err
	}

	return serviceAccount, nil
}

func (s *service) CreateServiceAccount(ctx context.Context, input *models.ServiceAccount) (*models.ServiceAccount, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	if err := caller.RequireAccessToGroup(ctx, input.GroupID, models.DeployerRole); err != nil {
		return nil, err
	}

	// Validate model
	if err := input.Validate(); err != nil {
		return nil, err
	}

	input.CreatedBy = caller.GetSubject()

	s.logger.Infow("Requested creation of a service account.",
		"caller", caller.GetSubject(),
		"groupID", input.GroupID,
		"serviceAccountName", input.Name,
	)

	// Store service account in DB
	return s.dbClient.ServiceAccounts.CreateServiceAccount(ctx, input)
}

func (s *service) UpdateServiceAccount(ctx context.Context, serviceAccount *models.ServiceAccount) (*models.ServiceAccount, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	if err := caller.RequireAccessToGroup(ctx, serviceAccount.GroupID, models.DeployerRole); err != nil {
		return nil, err
	}

	// Validate model
	if err := serviceAccount.Validate(); err != nil {
		return nil, err
	}

	s.logger.Infow("Requested an update to a service account.",
		"caller", caller.GetSubject(),
		"groupID", serviceAccount.GroupID,
		"serviceAccountID", serviceAccount.Metadata.ID,
	)

	// Store serviceAccount in DB
	return s.dbClient.ServiceAccounts.UpdateServiceAccount(ctx, serviceAccount)
}

func (s *service) Login(ctx context.Context, input *LoginInput) (*LoginResponse, error) {
	// Parse token
	token, err := jwt.Parse(input.Token)
	if err != nil {
		return nil, errors.NewError(errors.EUnauthorized, fmt.Sprintf("Failed to decode token %v", err))
	}

	// Check if token if from a valid issuer associated with the service account
	if token.Issuer() == "" {
		return nil, errors.NewError(errors.EUnauthorized, "JWT is missing issuer claim")
	}

	// Get service account
	serviceAccount, err := s.dbClient.ServiceAccounts.GetServiceAccountByPath(ctx, input.ServiceAccount)
	if err != nil || serviceAccount == nil {
		s.logger.Infof("Failed login to service account; resource path %s does not exist", input.ServiceAccount)
		return nil, failedLoginError
	}

	issuer := token.Issuer()

	trustPolicy := s.findMatchingTrustPolicy(issuer, serviceAccount.OIDCTrustPolicies)
	if trustPolicy == nil {
		s.logger.Infof("Failed login to service account %s; issuer %s not found in trust policy", serviceAccount.ResourcePath, issuer)
		return nil, failedLoginError
	}

	// Get issuer JWK response
	keySet, err := s.getKeySetFunc(ctx, trustPolicy.Issuer)
	if err != nil {
		return nil, err
	}

	// Set default key to RS256 if it's not specified in JWK set
	iter := keySet.Iterate(ctx)
	for iter.Next(ctx) {
		key := iter.Pair().Value.(jwk.Key)
		if err = key.Set(jwk.AlgorithmKey, jwa.RS256); err != nil {
			return nil, err
		}
	}

	options := []jwt.ParseOption{
		jwt.WithKeySet(keySet),
		jwt.WithValidate(true),
	}
	for k, v := range trustPolicy.BoundClaims {
		if k == "aud" {
			options = append(options, jwt.WithAudience(v))
		} else {
			options = append(options, jwt.WithClaimValue(k, v))
		}
	}

	// Parse and Verify token
	if _, err = jwt.Parse(input.Token, options...); err != nil {
		if strings.Contains(err.Error(), "failed to verify jws signature") {
			s.logger.Infof("Login to service account %s failed due to invalid token signature", serviceAccount.ResourcePath)
			return nil, failedLoginError
		}
		return nil, errors.NewError(errors.EUnauthorized, fmt.Sprintf("Failed to verify token %v", err))
	}

	// Generate service account token
	expiration := time.Now().Add(1 * time.Hour)
	serviceAccountToken, err := s.idp.GenerateToken(ctx, &auth.TokenInput{
		Expiration: &expiration,
		Subject:    serviceAccount.ResourcePath,
		Claims: map[string]string{
			"service_account_name": serviceAccount.Name,
			"service_account_path": serviceAccount.ResourcePath,
			"service_account_id":   gid.ToGlobalID(gid.ServiceAccountType, serviceAccount.Metadata.ID),
			"type":                 auth.ServiceAccountTokenType,
		},
	})
	if err != nil {
		return nil, err
	}

	return &LoginResponse{
		Token: serviceAccountToken,
	}, nil
}

func (s *service) findMatchingTrustPolicy(issuer string, policies []models.OIDCTrustPolicy) *models.OIDCTrustPolicy {
	normalizedIssuer := issuer
	if !strings.HasPrefix(issuer, "https://") {
		normalizedIssuer = fmt.Sprintf("https://%s", issuer)
	}
	for _, p := range policies {
		if normalizedIssuer == p.Issuer {
			return &p
		}
	}
	return nil
}

func getKeySet(ctx context.Context, issuer string) (jwk.Set, error) {
	fetchCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	oidcConfig, err := auth.GetOpenIDConfig(fetchCtx, issuer)
	if err != nil {
		return nil, errors.NewError(errors.EInternal, fmt.Sprintf("Failed to get OIDC discovery document for issuer %s; %v", issuer, err))
	}

	fetchCtx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Get issuer JWK response
	keySet, err := jwk.Fetch(fetchCtx, oidcConfig.JwksURI)
	if err != nil {
		return nil, errors.NewError(errors.EInternal, fmt.Sprintf("Failed to query JWK URL %s; %v", oidcConfig.JwksURI, err))
	}

	return keySet, nil
}
