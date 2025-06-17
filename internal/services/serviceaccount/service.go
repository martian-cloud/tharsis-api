// Package serviceaccount package
package serviceaccount

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/lestrrat-go/jwx/v2/jwt"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/limits"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

const (
	failedToVerifyJWSSignature = "Failed to verify token could not verify message using any of the signatures or keys"
	expiredTokenDetector       = "Failed to verify token exp not satisfied"
)

var (
	serviceAccountLoginDuration = 1 * time.Hour

	errFailedCreateToken = errors.New(
		"Failed to create service account token due to one of the "+
			"following reasons: the service account does not exist; the JWT token used as input is invalid; the issuer "+
			"for the token is not a valid issuer.",
		errors.WithErrorCode(errors.EUnauthorized),
	)

	errExpiredToken = errors.New(
		"failed to create service account token due to an expired token",
		errors.WithErrorCode(errors.EUnauthorized),
	)
)

// CreateTokenInput for logging into a service account
type CreateTokenInput struct {
	ServiceAccountPublicID string // Service account identifier (TRN or GID)
	Token                  []byte
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
	PaginationOptions *pagination.Options
	// Search returns only the service accounts with a name or resource path that starts with the value of search
	Search *string
	// RunnerID will filter service accounts that are assigned to the specified runner
	RunnerID *string
	// NamespacePath is the namespace to return service accounts for
	NamespacePath string
	// IncludeInherited includes inherited services accounts in the result
	IncludeInherited bool
}

// Service implements all service account related functionality
type Service interface {
	GetServiceAccountByTRN(ctx context.Context, trn string) (*models.ServiceAccount, error)
	GetServiceAccountByID(ctx context.Context, id string) (*models.ServiceAccount, error)
	GetServiceAccounts(ctx context.Context, input *GetServiceAccountsInput) (*db.ServiceAccountsResult, error)
	GetServiceAccountsByIDs(ctx context.Context, idList []string) ([]models.ServiceAccount, error)
	CreateServiceAccount(ctx context.Context, input *models.ServiceAccount) (*models.ServiceAccount, error)
	UpdateServiceAccount(ctx context.Context, serviceAccount *models.ServiceAccount) (*models.ServiceAccount, error)
	DeleteServiceAccount(ctx context.Context, serviceAccount *models.ServiceAccount) error
	CreateToken(ctx context.Context, input *CreateTokenInput) (*CreateTokenResponse, error)
}

type service struct {
	logger                 logger.Logger
	dbClient               *db.Client
	limitChecker           limits.LimitChecker
	idp                    auth.IdentityProvider
	openIDConfigFetcher    auth.OpenIDConfigFetcher
	activityService        activityevent.Service
	buildOIDCTokenVerifier func(ctx context.Context, issuers []string, oidcConfigFetcher auth.OpenIDConfigFetcher) auth.OIDCTokenVerifier
}

// NewService creates an instance of Service
func NewService(
	logger logger.Logger,
	dbClient *db.Client,
	limitChecker limits.LimitChecker,
	idp auth.IdentityProvider,
	openIDConfigFetcher auth.OpenIDConfigFetcher,
	activityService activityevent.Service,
) Service {
	return newService(
		logger,
		dbClient,
		limitChecker,
		idp,
		openIDConfigFetcher,
		activityService,
		buildOIDCTokenVerifier,
	)
}

func newService(
	logger logger.Logger,
	dbClient *db.Client,
	limitChecker limits.LimitChecker,
	idp auth.IdentityProvider,
	openIDConfigFetcher auth.OpenIDConfigFetcher,
	activityService activityevent.Service,
	buildOIDCTokenVerifier func(ctx context.Context, issuers []string, oidcConfigFetcher auth.OpenIDConfigFetcher) auth.OIDCTokenVerifier,
) Service {
	return &service{
		logger:                 logger,
		dbClient:               dbClient,
		limitChecker:           limitChecker,
		idp:                    idp,
		openIDConfigFetcher:    openIDConfigFetcher,
		activityService:        activityService,
		buildOIDCTokenVerifier: buildOIDCTokenVerifier,
	}
}

func (s *service) GetServiceAccounts(ctx context.Context, input *GetServiceAccountsInput) (*db.ServiceAccountsResult, error) {
	ctx, span := tracer.Start(ctx, "svc.GetServiceAccounts")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	err = caller.RequirePermission(ctx, models.ViewServiceAccountPermission, auth.WithNamespacePath(input.NamespacePath))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
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
		tracing.RecordError(span, err, "failed to get service accounts")
		return nil, err
	}

	return result, nil
}

func (s *service) GetServiceAccountsByIDs(ctx context.Context, idList []string) ([]models.ServiceAccount, error) {
	ctx, span := tracer.Start(ctx, "svc.GetServiceAccountsByIDs")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	result, err := s.dbClient.ServiceAccounts.GetServiceAccounts(ctx, &db.GetServiceAccountsInput{
		Filter: &db.ServiceAccountFilter{
			ServiceAccountIDs: idList,
		},
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to get service accounts")
		return nil, err
	}

	namespacePaths := []string{}
	for _, sa := range result.ServiceAccounts {
		namespacePaths = append(namespacePaths, sa.GetGroupPath())
	}

	if len(namespacePaths) > 0 {
		err = caller.RequireAccessToInheritableResource(ctx, types.ServiceAccountModelType, auth.WithNamespacePaths(namespacePaths))
		if err != nil {
			tracing.RecordError(span, err, "inheritable resource access check failed")
			return nil, err
		}
	}

	return result.ServiceAccounts, nil
}

func (s *service) DeleteServiceAccount(ctx context.Context, serviceAccount *models.ServiceAccount) error {
	ctx, span := tracer.Start(ctx, "svc.DeleteServiceAccount")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return err
	}

	err = caller.RequirePermission(ctx, models.DeleteServiceAccountPermission, auth.WithGroupID(serviceAccount.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return err
	}

	s.logger.Infow("Requested deletion of a service account.",
		"caller", caller.GetSubject(),
		"groupID", serviceAccount.GroupID,
		"serviceAccountID", serviceAccount.Metadata.ID,
	)

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer DeleteServiceAccount: %v", txErr)
		}
	}()

	err = s.dbClient.ServiceAccounts.DeleteServiceAccount(txContext, serviceAccount)
	if err != nil {
		tracing.RecordError(span, err, "failed to delete service account")
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
		tracing.RecordError(span, err, "failed to create activity event")
		return err
	}

	return s.dbClient.Transactions.CommitTx(txContext)
}

func (s *service) GetServiceAccountByTRN(ctx context.Context, trn string) (*models.ServiceAccount, error) {
	ctx, span := tracer.Start(ctx, "svc.GetServiceAccountByTRN")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	// Get serviceAccount from DB
	serviceAccount, err := s.dbClient.ServiceAccounts.GetServiceAccountByTRN(ctx, trn)
	if err != nil {
		tracing.RecordError(span, err, "failed to get service account by TRN")
		return nil, err
	}

	if serviceAccount == nil {
		tracing.RecordError(span, nil, "service account with TRN %s not found", trn)
		return nil, errors.New("service account with TRN %s not found", trn, errors.WithErrorCode(errors.ENotFound))
	}

	err = caller.RequireAccessToInheritableResource(ctx, types.ServiceAccountModelType, auth.WithGroupID(serviceAccount.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "inheritable resource access check failed")
		return nil, err
	}

	return serviceAccount, nil
}

func (s *service) GetServiceAccountByID(ctx context.Context, id string) (*models.ServiceAccount, error) {
	ctx, span := tracer.Start(ctx, "svc.GetServiceAccountByID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	// Get serviceAccount from DB
	serviceAccount, err := s.dbClient.ServiceAccounts.GetServiceAccountByID(ctx, id)
	if err != nil {
		tracing.RecordError(span, err, "failed to get service account by ID")
		return nil, err
	}

	if serviceAccount == nil {
		tracing.RecordError(span, nil, "service account with ID %s not found", id)
		return nil, errors.New("service account with ID %s not found", id, errors.WithErrorCode(errors.ENotFound))
	}

	err = caller.RequireAccessToInheritableResource(ctx, types.ServiceAccountModelType, auth.WithGroupID(serviceAccount.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "inheritable resource access check failed")
		return nil, err
	}

	return serviceAccount, nil
}

func (s *service) CreateServiceAccount(ctx context.Context, input *models.ServiceAccount) (*models.ServiceAccount, error) {
	ctx, span := tracer.Start(ctx, "svc.CreateServiceAccount")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	err = caller.RequirePermission(ctx, models.CreateServiceAccountPermission, auth.WithGroupID(input.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	// Validate model
	if err = input.Validate(); err != nil {
		tracing.RecordError(span, err, "failed to validate service account model")
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
		tracing.RecordError(span, err, "failed to begin DB transaction")
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
		tracing.RecordError(span, err, "failed to create service account")
		return nil, err
	}

	groupPath := createdServiceAccount.GetGroupPath()

	// Get the number of service accounts in the group to check whether we just violated the limit.
	newServiceAccounts, err := s.dbClient.ServiceAccounts.GetServiceAccounts(txContext, &db.GetServiceAccountsInput{
		Filter: &db.ServiceAccountFilter{
			NamespacePaths: []string{groupPath},
		},
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(0),
		},
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to get group's service accounts")
		return nil, err
	}
	if err = s.limitChecker.CheckLimit(txContext,
		limits.ResourceLimitServiceAccountsPerGroup, newServiceAccounts.PageInfo.TotalCount); err != nil {
		tracing.RecordError(span, err, "limit check failed")
		return nil, err
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &groupPath,
			Action:        models.ActionCreate,
			TargetType:    models.TargetServiceAccount,
			TargetID:      createdServiceAccount.Metadata.ID,
		}); err != nil {
		tracing.RecordError(span, err, "failed to create activity event")
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return nil, err
	}

	return createdServiceAccount, nil
}

func (s *service) UpdateServiceAccount(ctx context.Context, serviceAccount *models.ServiceAccount) (*models.ServiceAccount, error) {
	ctx, span := tracer.Start(ctx, "svc.UpdateServiceAccount")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	err = caller.RequirePermission(ctx, models.UpdateServiceAccountPermission, auth.WithGroupID(serviceAccount.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	// Validate model
	if err = serviceAccount.Validate(); err != nil {
		tracing.RecordError(span, err, "failed to validate service account model")
		return nil, err
	}

	s.logger.Infow("Requested an update to a service account.",
		"caller", caller.GetSubject(),
		"groupID", serviceAccount.GroupID,
		"serviceAccountID", serviceAccount.Metadata.ID,
	)

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
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
		tracing.RecordError(span, err, "failed to update service account")
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
		tracing.RecordError(span, err, "failed to create activity event")
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return nil, err
	}

	return updatedServiceAccount, nil
}

func (s *service) CreateToken(ctx context.Context, input *CreateTokenInput) (*CreateTokenResponse, error) {
	ctx, span := tracer.Start(ctx, "svc.CreateToken")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	// Parse token
	token, err := jwt.Parse(input.Token, jwt.WithVerify(false))
	if err != nil {
		tracing.RecordError(span, err, "failed to decode token")
		return nil, errors.Wrap(err, "failed to decode token", errors.WithErrorCode(errors.EUnauthorized))
	}

	// Check if token is from a valid issuer associated with the service account
	issuer := token.Issuer()
	if issuer == "" {
		tracing.RecordError(span, nil, "JWT is missing issuer claim")
		return nil, errors.New("JWT is missing issuer claim", errors.WithErrorCode(errors.EUnauthorized))
	}

	if input.ServiceAccountPublicID == "" {
		s.logger.Infof("Failed to create token for service account; service account ID is empty")
		tracing.RecordError(span, nil, "service account ID is empty")
		return nil, errFailedCreateToken
	}

	// Get service account based on the ID type (TRN or GID)
	var serviceAccount *models.ServiceAccount
	if types.IsTRN(input.ServiceAccountPublicID) {
		serviceAccount, err = s.dbClient.ServiceAccounts.GetServiceAccountByTRN(ctx, input.ServiceAccountPublicID)
	} else {
		serviceAccount, err = s.dbClient.ServiceAccounts.GetServiceAccountByID(ctx, gid.FromGlobalID(input.ServiceAccountPublicID))
	}

	if err != nil || serviceAccount == nil {
		s.logger.Infof("Failed to create token for service account; service account %s does not exist", input.ServiceAccountPublicID)
		tracing.RecordError(span, nil,
			"failed to create token for service account; service account does not exist")
		return nil, errFailedCreateToken
	}

	trustPolicies := s.findMatchingTrustPolicies(issuer, serviceAccount.OIDCTrustPolicies)
	if len(trustPolicies) == 0 {
		s.logger.Infof("Failed to create token for service account %s; issuer %s not found in trust policy", serviceAccount.GetResourcePath(), issuer)
		tracing.RecordError(span, nil,
			"failed to create token for service account; issuer not found in trust policy")
		return nil, errFailedCreateToken
	}

	// One satisfied trust policy is sufficient for service account token creation.
	// However, must keep all the failures in case everything fails.
	mismatchesFound := []string{}
	for _, trustPolicy := range trustPolicies {

		err := s.verifyOneTrustPolicy(ctx, input.Token, trustPolicy)
		if err != nil {

			// Catch bubbled-up invalid token signature errors here.
			if strings.Contains(err.Error(), failedToVerifyJWSSignature) {
				s.logger.Infof("Failed to create token for service account %s due to invalid token signature",
					serviceAccount.GetResourcePath())
				tracing.RecordError(span, nil,
					"failed to create token for service account; invalid token signature")
				return nil, errFailedCreateToken
			}

			// Catch token expiration here.  An expired token will be expired for all trust policies.
			if strings.Contains(err.Error(), expiredTokenDetector) {
				s.logger.Infof("Failed to create token for service account %s due to expired token",
					serviceAccount.GetResourcePath())
				tracing.RecordError(span, nil,
					"failed to create token for service account; expired token")
				return nil, errExpiredToken
			}

			// Record this claim mismatch in case no other, later trust policy is satisfied
			mismatchesFound = append(mismatchesFound, err.Error())
		} else {
			// The input token satisfied this trust policy, so the service account token creation succeeded.

			// Generate service account token
			expiration := time.Now().Add(serviceAccountLoginDuration)
			serviceAccountToken, err := s.idp.GenerateToken(ctx, &auth.TokenInput{
				Expiration: &expiration,
				Subject:    serviceAccount.GetResourcePath(),
				Typ:        auth.ServiceAccountTokenType,
				Claims: map[string]string{
					"service_account_name": serviceAccount.Name,
					"service_account_path": serviceAccount.GetResourcePath(),
					"service_account_id":   serviceAccount.GetGlobalID(),
					"type":                 auth.ServiceAccountTokenType,
				},
			})
			if err != nil {
				tracing.RecordError(span, err, "failed to generate token for service account")
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
	tracing.RecordError(span, nil, "of the trust policies for issuer, none was satisfied")
	return nil, errors.New(
		fmt.Sprintf("of the trust policies for issuer %s, none was satisfied", issuer),
		errors.WithErrorCode(errors.EUnauthorized),
	)
}

// findMatchingTrustPolicies returns a slice of the policies that have a matching issuer.
// If no match is found, it returns an empty slice.
// Trailing forward slashes are ignored on both sides of the comparison.
// Claims are not checked.
func (s *service) findMatchingTrustPolicies(issuer string, policies []models.OIDCTrustPolicy) []models.OIDCTrustPolicy {
	result := []models.OIDCTrustPolicy{}
	normalizedIssuer := auth.NormalizeOIDCIssuer(issuer)
	for _, p := range policies {
		if normalizedIssuer == auth.NormalizeOIDCIssuer(p.Issuer) {
			result = append(result, p)
		}
	}
	return result
}

// verifyOneTrustPolicy verifies a token vs. one trust policy.
func (s *service) verifyOneTrustPolicy(ctx context.Context, inputToken []byte, trustPolicy models.OIDCTrustPolicy) error {
	verifier := s.buildOIDCTokenVerifier(ctx, []string{trustPolicy.Issuer}, s.openIDConfigFetcher)

	options := []jwt.ValidateOption{}
	for k, v := range trustPolicy.BoundClaims {
		options = append(options, jwt.WithValidator(newClaimValueValidator(k, v, trustPolicy.BoundClaimsType == models.BoundClaimsTypeGlob)))
	}

	_, err := verifier.VerifyToken(ctx, string(inputToken), options)
	return err
}

func buildOIDCTokenVerifier(ctx context.Context, issuers []string, oidcConfigFetcher auth.OpenIDConfigFetcher) auth.OIDCTokenVerifier {
	oidcTokenVerifier := auth.NewOIDCTokenVerifier(ctx, issuers, oidcConfigFetcher, false)
	return oidcTokenVerifier
}
