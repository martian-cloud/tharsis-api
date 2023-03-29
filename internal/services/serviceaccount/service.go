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
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
)

const (
	failedToVerifyJWSSignature = "failed to verify jws signature"
	expiredTokenDetector       = "Failed to verify token exp not satisfied"
)

var (
	serviceAccountLoginDuration = 1 * time.Hour

	failedCreateTokenError = errors.NewError(errors.EUnauthorized, "Failed to create service account token due to one of the "+
		"following reasons: the service account does not exist; the JWT token used as input is invalid; the issuer "+
		"for the token is not a valid issuer.")

	expiredTokenError = errors.NewError(errors.EUnauthorized,
		"Failed to create service account token due to an expired token.")
)

// CreateTokenInput for logging into a service account
type CreateTokenInput struct {
	// ServiceAccount ID or resource path
	ServiceAccount string
	Token          []byte
}

// CreateTokenResponse returned after logging into a service account
type CreateTokenResponse struct {
	Token     []byte
	ExpiresIn int32 // seconds
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
	// RunnerID will filter service accounts that are assigned to the specified runner
	RunnerID *string
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
	CreateToken(ctx context.Context, input *CreateTokenInput) (*CreateTokenResponse, error)
}

type service struct {
	logger              logger.Logger
	dbClient            *db.Client
	idp                 *auth.IdentityProvider
	openIDConfigFetcher *auth.OpenIDConfigFetcher
	getKeySetFunc       func(ctx context.Context, issuer string, configFetcher *auth.OpenIDConfigFetcher) (jwk.Set, error)
	activityService     activityevent.Service
}

// NewService creates an instance of Service
func NewService(
	logger logger.Logger,
	dbClient *db.Client,
	idp *auth.IdentityProvider,
	openIDConfigFetcher *auth.OpenIDConfigFetcher,
	activityService activityevent.Service,
) Service {
	return newService(
		logger,
		dbClient,
		idp,
		openIDConfigFetcher,
		getKeySet,
		activityService,
	)
}

func newService(
	logger logger.Logger,
	dbClient *db.Client,
	idp *auth.IdentityProvider,
	openIDConfigFetcher *auth.OpenIDConfigFetcher,
	getKeySetFunc func(ctx context.Context, issuer string, configFetcher *auth.OpenIDConfigFetcher) (jwk.Set, error),
	activityService activityevent.Service,
) Service {
	return &service{
		logger:              logger,
		dbClient:            dbClient,
		idp:                 idp,
		openIDConfigFetcher: openIDConfigFetcher,
		getKeySetFunc:       getKeySetFunc,
		activityService:     activityService,
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
		Search:   input.Search,
		RunnerID: input.RunnerID,
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

	if rErr := caller.RequireAccessToGroup(ctx, serviceAccount.GroupID, models.DeployerRole); rErr != nil {
		return rErr
	}

	s.logger.Infow("Requested deletion of a service account.",
		"caller", caller.GetSubject(),
		"groupID", serviceAccount.GroupID,
		"serviceAccountID", serviceAccount.Metadata.ID,
	)

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer DeleteServiceAccount: %v", txErr)
		}
	}()

	err = s.dbClient.ServiceAccounts.DeleteServiceAccount(txContext, serviceAccount)
	if err != nil {
		return err
	}

	groupPath := serviceAccount.GetGroupPath()

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &groupPath,
			Action:        models.ActionDeleteChildResource,
			TargetType:    models.TargetGroup,
			TargetID:      serviceAccount.GroupID,
			Payload: &models.ActivityEventDeleteChildResourcePayload{
				Name: serviceAccount.Name,
				ID:   serviceAccount.Metadata.ID,
				Type: string(models.TargetServiceAccount),
			},
		}); err != nil {
		return err
	}

	return s.dbClient.Transactions.CommitTx(txContext)
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

	if err = caller.RequireAccessToGroup(ctx, input.GroupID, models.DeployerRole); err != nil {
		return nil, err
	}

	// Validate model
	if err = input.Validate(); err != nil {
		return nil, err
	}

	input.CreatedBy = caller.GetSubject()

	s.logger.Infow("Requested creation of a service account.",
		"caller", caller.GetSubject(),
		"groupID", input.GroupID,
		"serviceAccountName", input.Name,
	)

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer CreateServiceAccount: %v", txErr)
		}
	}()

	// Store service account in DB
	createdServiceAccount, err := s.dbClient.ServiceAccounts.CreateServiceAccount(txContext, input)
	if err != nil {
		return nil, err
	}

	groupPath := createdServiceAccount.GetGroupPath()

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &groupPath,
			Action:        models.ActionCreate,
			TargetType:    models.TargetServiceAccount,
			TargetID:      createdServiceAccount.Metadata.ID,
		}); err != nil {
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, err
	}

	return createdServiceAccount, nil
}

func (s *service) UpdateServiceAccount(ctx context.Context, serviceAccount *models.ServiceAccount) (*models.ServiceAccount, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	if err = caller.RequireAccessToGroup(ctx, serviceAccount.GroupID, models.DeployerRole); err != nil {
		return nil, err
	}

	// Validate model
	if err = serviceAccount.Validate(); err != nil {
		return nil, err
	}

	s.logger.Infow("Requested an update to a service account.",
		"caller", caller.GetSubject(),
		"groupID", serviceAccount.GroupID,
		"serviceAccountID", serviceAccount.Metadata.ID,
	)

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer UpdateServiceAccount: %v", txErr)
		}
	}()

	// Store serviceAccount in DB
	updatedServiceAccount, err := s.dbClient.ServiceAccounts.UpdateServiceAccount(txContext, serviceAccount)
	if err != nil {
		return nil, err
	}

	groupPath := updatedServiceAccount.GetGroupPath()

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &groupPath,
			Action:        models.ActionUpdate,
			TargetType:    models.TargetServiceAccount,
			TargetID:      updatedServiceAccount.Metadata.ID,
		}); err != nil {
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, err
	}

	return updatedServiceAccount, nil
}

func (s *service) CreateToken(ctx context.Context, input *CreateTokenInput) (*CreateTokenResponse, error) {
	// Parse token
	token, err := jwt.Parse(input.Token)
	if err != nil {
		return nil, errors.NewError(errors.EUnauthorized, fmt.Sprintf("Failed to decode token %v", err))
	}

	// Check if token is from a valid issuer associated with the service account
	issuer := token.Issuer()
	if issuer == "" {
		return nil, errors.NewError(errors.EUnauthorized, "JWT is missing issuer claim")
	}

	// Get service account
	serviceAccount, err := s.dbClient.ServiceAccounts.GetServiceAccountByPath(ctx, input.ServiceAccount)
	if err != nil || serviceAccount == nil {
		s.logger.Infof("Failed to create token for service account; resource path %s does not exist", input.ServiceAccount)
		return nil, failedCreateTokenError
	}

	trustPolicies := s.findMatchingTrustPolicies(issuer, serviceAccount.OIDCTrustPolicies)
	if len(trustPolicies) == 0 {
		s.logger.Infof("Failed to create token for service account %s; issuer %s not found in trust policy", serviceAccount.ResourcePath, issuer)
		return nil, failedCreateTokenError
	}

	// One satisfied trust policy is sufficient for service account token creation.
	// However, must keep all the failures in case everything fails.
	mismatchesFound := []string{}
	for _, trustPolicy := range trustPolicies {

		err := s.verifyOneTrustPolicy(ctx, input.Token, trustPolicy, serviceAccount)
		if err != nil {

			// Catch bubbled-up invalid token signature errors here.
			if strings.Contains(err.Error(), failedToVerifyJWSSignature) {
				s.logger.Infof("Failed to create token for service account %s due to invalid token signature",
					serviceAccount.ResourcePath)
				return nil, failedCreateTokenError
			}

			// Catch token expiration here.  An expired token will be expired for all trust policies.
			if strings.Contains(err.Error(), expiredTokenDetector) {
				s.logger.Infof("Failed to create token for service account %s due to expired token",
					serviceAccount.ResourcePath)
				return nil, expiredTokenError
			}

			// Record this claim mismatch in case no other, later trust policy is satisfied
			mismatchesFound = append(mismatchesFound, err.Error())
		} else {
			// The input token satisfied this trust policy, so the service account token creation succeeded.

			// Generate service account token
			expiration := time.Now().Add(serviceAccountLoginDuration)
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

			return &CreateTokenResponse{
				Token:     serviceAccountToken,
				ExpiresIn: int32(serviceAccountLoginDuration / time.Second),
			}, nil
		}
	}

	// Log all the mismatches found so we can look them up if needed.
	s.logger.Infof("failed to create service account token for issuer %s; %s", issuer, strings.Join(mismatchesFound, "; "))

	// We know there was at least one trust policy checked, otherwise we would have returned before the for loop.
	// To get here, all of the trust policies that were checked must have failed.
	return nil, errors.NewError(errors.EUnauthorized,
		fmt.Sprintf("of the trust policies for issuer %s, none was satisfied", issuer))
}

// findMatchingTrustPolicies returns a slice of the policies that have a matching issuer.
// If no match is found, it returns an empty slice.
// Trailing forward slashes are ignored on both sides of the comparison.
// Claims are not checked.
func (s *service) findMatchingTrustPolicies(issuer string, policies []models.OIDCTrustPolicy) []models.OIDCTrustPolicy {
	result := []models.OIDCTrustPolicy{}
	normalizedIssuer := issuer
	if !strings.HasPrefix(issuer, "https://") {
		normalizedIssuer = fmt.Sprintf("https://%s", issuer)
	}
	normalizedIssuer = strings.TrimSuffix(normalizedIssuer, "/")
	for _, p := range policies {
		if normalizedIssuer == strings.TrimSuffix(p.Issuer, "/") {
			result = append(result, p)
		}
	}
	return result
}

func getKeySet(ctx context.Context, issuer string, configFetcher *auth.OpenIDConfigFetcher) (jwk.Set, error) {
	fetchCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	oidcConfig, err := configFetcher.GetOpenIDConfig(fetchCtx, issuer)
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

// verifyOneTrustPolicy verifies a token vs. one trust policy.
func (s *service) verifyOneTrustPolicy(ctx context.Context, inputToken []byte, trustPolicy models.OIDCTrustPolicy,
	serviceAccount *models.ServiceAccount) error {

	// Get issuer JWK response
	keySet, err := s.getKeySetFunc(ctx, trustPolicy.Issuer, s.openIDConfigFetcher)
	if err != nil {
		return err
	}

	// Set default key to RS256 if it's not specified in JWK set
	iter := keySet.Iterate(ctx)
	for iter.Next(ctx) {
		key := iter.Pair().Value.(jwk.Key)
		if err = key.Set(jwk.AlgorithmKey, jwa.RS256); err != nil {
			return err
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
	if _, err = jwt.Parse(inputToken, options...); err != nil {
		return errors.NewError(errors.EUnauthorized,
			fmt.Sprintf("Failed to verify token %v", err))
	}

	return nil
}
