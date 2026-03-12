// Package serviceaccount package
package serviceaccount

import (
	"bytes"
	"context"
	"encoding/json"
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
	failedToVerifyJWSSignature = "failed to verify token: could not verify message using any of the signatures or keys"
	expiredTokenDetector       = `"exp" not satisfied`
)

// Grant types for service account tokens
const (
	GrantTypeOIDCRelyingParty  = "oidc_relying_party"
	GrantTypeClientCredentials = "client_credentials"
)

var (
	serviceAccountLoginDuration = 15 * time.Minute

	errFailedCreateClientCredentialsToken = errors.New(
		"failed to create service account token due to one of the following reasons: "+
			"the client credentials are missing; the service account does not exist; "+
			"Client credentials are not enabled for the service account; the client secret is invalid or expired.",
		errors.WithErrorCode(errors.EUnauthorized),
	)
)

// CreateOIDCTokenInput for logging into a service account via OIDC token exchange
type CreateOIDCTokenInput struct {
	ServiceAccountPublicID string // Service account identifier (TRN or GID)
	Token                  []byte
}

// CreateTokenResponse returned after logging into a service account
type CreateTokenResponse struct {
	Token     []byte
	ExpiresIn int32 // seconds
}

// CreateClientCredentialsTokenInput is the input for creating a token using client credentials
type CreateClientCredentialsTokenInput struct {
	ClientID     string
	ClientSecret string
}

// Response is the response for service account mutations
type Response struct {
	ServiceAccount *models.ServiceAccount
	ClientSecret   *string
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

// CreateServiceAccountInput is the input for creating a service account
type CreateServiceAccountInput struct {
	Name                    string
	Description             string
	GroupID                 string
	OIDCTrustPolicies       []models.OIDCTrustPolicy
	ClientSecretExpiresAt   *time.Time
	EnableClientCredentials bool
}

// UpdateServiceAccountInput is the input for updating a service account
type UpdateServiceAccountInput struct {
	ID                      string
	Description             *string
	OIDCTrustPolicies       []models.OIDCTrustPolicy
	EnableClientCredentials *bool
	ClientSecretExpiresAt   *time.Time
	MetadataVersion         *int
}

// DeleteServiceAccountInput is the input for deleting a service account
type DeleteServiceAccountInput struct {
	ID              string
	MetadataVersion *int
}

// ResetClientCredentialsInput is the input for resetting client credentials
type ResetClientCredentialsInput struct {
	ID                    string
	ClientSecretExpiresAt *time.Time
}

// newLoginError returns an error wrapping with serviceAccount and tokenClaims
func createGenericOIDCLoginError(serviceAccountID string, token jwt.Token) error {
	msg := fmt.Sprintf("failed to create service account token for service account %q due to one of the following reasons: "+
		"the service account does not exist; the JWT token used as input is invalid; the issuer "+
		"for the token is not a valid issuer; the claims in the token do not satisfy the trust policy requirements",
		serviceAccountID)

	claimsJSON, err := extractClaimsFromToken(token)
	if err == nil {
		msg += fmt.Sprintf("; token claims provided: %s", claimsJSON)
	}

	return errors.New(msg, errors.WithErrorCode(errors.EUnauthorized))
}

func extractClaimsFromToken(token jwt.Token) (string, error) {
	claims := token.PrivateClaims()
	if iss := token.Issuer(); iss != "" {
		claims["iss"] = iss
	}
	if aud := token.Audience(); len(aud) > 0 {
		claims["aud"] = aud
	}
	if sub := token.Subject(); sub != "" {
		claims["sub"] = sub
	}
	if exp := token.Expiration(); !exp.IsZero() {
		claims["exp"] = exp.Unix()
	}
	if nbf := token.NotBefore(); !nbf.IsZero() {
		claims["nbf"] = nbf.Unix()
	}
	if iat := token.IssuedAt(); !iat.IsZero() {
		claims["iat"] = iat.Unix()
	}
	if jti := token.JwtID(); jti != "" {
		claims["jti"] = jti
	}
	claimsBytes, marshalErr := json.Marshal(claims)
	if marshalErr != nil {
		return "", errors.Wrap(marshalErr, "unable to marshal token claims")
	}
	return string(claimsBytes), nil
}

// Service implements all service account related functionality
type Service interface {
	GetServiceAccountByTRN(ctx context.Context, trn string) (*models.ServiceAccount, error)
	GetServiceAccountByID(ctx context.Context, id string) (*models.ServiceAccount, error)
	GetServiceAccounts(ctx context.Context, input *GetServiceAccountsInput) (*db.ServiceAccountsResult, error)
	GetServiceAccountsByIDs(ctx context.Context, idList []string) ([]models.ServiceAccount, error)
	CreateServiceAccount(ctx context.Context, input *CreateServiceAccountInput) (*Response, error)
	UpdateServiceAccount(ctx context.Context, input *UpdateServiceAccountInput) (*Response, error)
	DeleteServiceAccount(ctx context.Context, input *DeleteServiceAccountInput) error
	ResetClientCredentials(ctx context.Context, input *ResetClientCredentialsInput) (*Response, error)
	CreateOIDCToken(ctx context.Context, input *CreateOIDCTokenInput) (*CreateTokenResponse, error)
	CreateClientCredentialsToken(ctx context.Context, input *CreateClientCredentialsTokenInput) (*CreateTokenResponse, error)
}

type service struct {
	logger                  logger.Logger
	dbClient                *db.Client
	limitChecker            limits.LimitChecker
	signingKeyManager       auth.SigningKeyManager
	openIDConfigFetcher     auth.OpenIDConfigFetcher
	activityService         activityevent.Service
	buildOIDCTokenVerifier  func(ctx context.Context, issuers []string, oidcConfigFetcher auth.OpenIDConfigFetcher) auth.OIDCTokenVerifier
	secretMaxExpirationDays int
}

// NewService creates an instance of Service
func NewService(
	logger logger.Logger,
	dbClient *db.Client,
	limitChecker limits.LimitChecker,
	signingKeyManager auth.SigningKeyManager,
	openIDConfigFetcher auth.OpenIDConfigFetcher,
	activityService activityevent.Service,
	secretMaxExpirationDays int,
) Service {
	return newService(
		logger,
		dbClient,
		limitChecker,
		signingKeyManager,
		openIDConfigFetcher,
		activityService,
		buildOIDCTokenVerifier,
		secretMaxExpirationDays,
	)
}

func newService(
	logger logger.Logger,
	dbClient *db.Client,
	limitChecker limits.LimitChecker,
	signingKeyManager auth.SigningKeyManager,
	openIDConfigFetcher auth.OpenIDConfigFetcher,
	activityService activityevent.Service,
	buildOIDCTokenVerifier func(ctx context.Context, issuers []string, oidcConfigFetcher auth.OpenIDConfigFetcher) auth.OIDCTokenVerifier,
	secretMaxExpirationDays int,
) Service {
	return &service{
		logger:                  logger,
		dbClient:                dbClient,
		limitChecker:            limitChecker,
		signingKeyManager:       signingKeyManager,
		openIDConfigFetcher:     openIDConfigFetcher,
		activityService:         activityService,
		buildOIDCTokenVerifier:  buildOIDCTokenVerifier,
		secretMaxExpirationDays: secretMaxExpirationDays,
	}
}

func (s *service) GetServiceAccounts(ctx context.Context, input *GetServiceAccountsInput) (*db.ServiceAccountsResult, error) {
	ctx, span := tracer.Start(ctx, "svc.GetServiceAccounts")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "caller authorization failed", errors.WithSpan(span))
	}

	err = caller.RequirePermission(ctx, models.ViewServiceAccountPermission, auth.WithNamespacePath(input.NamespacePath))
	if err != nil {
		return nil, errors.Wrap(err, "permission check failed", errors.WithSpan(span))
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
		return nil, errors.Wrap(err, "failed to get service accounts", errors.WithSpan(span))
	}

	return result, nil
}

func (s *service) GetServiceAccountsByIDs(ctx context.Context, idList []string) ([]models.ServiceAccount, error) {
	ctx, span := tracer.Start(ctx, "svc.GetServiceAccountsByIDs")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "caller authorization failed", errors.WithSpan(span))
	}

	result, err := s.dbClient.ServiceAccounts.GetServiceAccounts(ctx, &db.GetServiceAccountsInput{
		Filter: &db.ServiceAccountFilter{
			ServiceAccountIDs: idList,
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get service accounts", errors.WithSpan(span))
	}

	namespacePaths := []string{}
	for _, sa := range result.ServiceAccounts {
		namespacePaths = append(namespacePaths, sa.GetGroupPath())
	}

	if len(namespacePaths) > 0 {
		err = caller.RequireAccessToInheritableResource(ctx, types.ServiceAccountModelType, auth.WithNamespacePaths(namespacePaths))
		if err != nil {
			return nil, errors.Wrap(err, "inheritable resource access check failed", errors.WithSpan(span))
		}
	}

	return result.ServiceAccounts, nil
}

func (s *service) DeleteServiceAccount(ctx context.Context, input *DeleteServiceAccountInput) error {
	ctx, span := tracer.Start(ctx, "svc.DeleteServiceAccount")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return errors.Wrap(err, "caller authorization failed", errors.WithSpan(span))
	}

	serviceAccount, err := s.dbClient.ServiceAccounts.GetServiceAccountByID(ctx, input.ID)
	if err != nil {
		return errors.Wrap(err, "failed to get service account", errors.WithSpan(span))
	}

	if serviceAccount == nil {
		return errors.New("service account not found", errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	err = caller.RequirePermission(ctx, models.DeleteServiceAccountPermission, auth.WithGroupID(serviceAccount.GroupID))
	if err != nil {
		return errors.Wrap(err, "permission check failed", errors.WithSpan(span))
	}

	if input.MetadataVersion != nil {
		serviceAccount.Metadata.Version = *input.MetadataVersion
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to begin DB transaction", errors.WithSpan(span))
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.WithContextFields(ctx).Errorf("failed to rollback tx for service layer DeleteServiceAccount: %v", txErr)
		}
	}()

	err = s.dbClient.ServiceAccounts.DeleteServiceAccount(txContext, serviceAccount)
	if err != nil {
		return errors.Wrap(err, "failed to delete service account", errors.WithSpan(span))
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
		return errors.Wrap(err, "failed to create activity event", errors.WithSpan(span))
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return errors.Wrap(err, "failed to commit DB transaction", errors.WithSpan(span))
	}

	s.logger.WithContextFields(ctx).Infow("Deleted a service account.",
		"serviceAccountID", serviceAccount.Metadata.ID,
	)

	return nil
}

func (s *service) GetServiceAccountByTRN(ctx context.Context, trn string) (*models.ServiceAccount, error) {
	ctx, span := tracer.Start(ctx, "svc.GetServiceAccountByTRN")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "caller authorization failed", errors.WithSpan(span))
	}

	// Get serviceAccount from DB
	serviceAccount, err := s.dbClient.ServiceAccounts.GetServiceAccountByTRN(ctx, trn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get service account by TRN", errors.WithSpan(span))
	}

	if serviceAccount == nil {
		return nil, errors.New("service account with TRN %s not found", trn, errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	err = caller.RequireAccessToInheritableResource(ctx, types.ServiceAccountModelType, auth.WithGroupID(serviceAccount.GroupID))
	if err != nil {
		return nil, errors.Wrap(err, "inheritable resource access check failed", errors.WithSpan(span))
	}

	return serviceAccount, nil
}

func (s *service) GetServiceAccountByID(ctx context.Context, id string) (*models.ServiceAccount, error) {
	ctx, span := tracer.Start(ctx, "svc.GetServiceAccountByID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "caller authorization failed", errors.WithSpan(span))
	}

	// Get serviceAccount from DB
	serviceAccount, err := s.dbClient.ServiceAccounts.GetServiceAccountByID(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get service account by ID", errors.WithSpan(span))
	}

	if serviceAccount == nil {
		return nil, errors.New("service account with ID %s not found", id, errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	err = caller.RequireAccessToInheritableResource(ctx, types.ServiceAccountModelType, auth.WithGroupID(serviceAccount.GroupID))
	if err != nil {
		return nil, errors.Wrap(err, "inheritable resource access check failed", errors.WithSpan(span))
	}

	return serviceAccount, nil
}

func (s *service) CreateServiceAccount(ctx context.Context, input *CreateServiceAccountInput) (*Response, error) {
	ctx, span := tracer.Start(ctx, "svc.CreateServiceAccount")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "caller authorization failed", errors.WithSpan(span))
	}

	err = caller.RequirePermission(ctx, models.CreateServiceAccountPermission, auth.WithGroupID(input.GroupID))
	if err != nil {
		return nil, errors.Wrap(err, "permission check failed", errors.WithSpan(span))
	}

	serviceAccount := &models.ServiceAccount{
		Name:              input.Name,
		Description:       input.Description,
		GroupID:           input.GroupID,
		OIDCTrustPolicies: input.OIDCTrustPolicies,
		CreatedBy:         caller.GetSubject(),
	}

	var clientSecret *string

	if input.EnableClientCredentials {
		secret, gErr := serviceAccount.GenerateClientSecret(input.ClientSecretExpiresAt, s.secretMaxExpirationDays)
		if gErr != nil {
			return nil, errors.Wrap(gErr, "failed to generate client secret", errors.WithSpan(span))
		}

		clientSecret = &secret
	}

	if err = serviceAccount.Validate(); err != nil {
		return nil, errors.Wrap(err, "failed to validate service account model", errors.WithSpan(span))
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin DB transaction", errors.WithSpan(span))
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.WithContextFields(ctx).Errorf("failed to rollback tx for service layer CreateServiceAccount: %v", txErr)
		}
	}()

	createdServiceAccount, err := s.dbClient.ServiceAccounts.CreateServiceAccount(txContext, serviceAccount)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create service account", errors.WithSpan(span))
	}

	groupPath := createdServiceAccount.GetGroupPath()

	newServiceAccounts, err := s.dbClient.ServiceAccounts.GetServiceAccounts(txContext, &db.GetServiceAccountsInput{
		Filter: &db.ServiceAccountFilter{
			NamespacePaths: []string{groupPath},
		},
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(0),
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get group's service accounts", errors.WithSpan(span))
	}

	if err = s.limitChecker.CheckLimit(txContext,
		limits.ResourceLimitServiceAccountsPerGroup, newServiceAccounts.PageInfo.TotalCount); err != nil {
		return nil, errors.Wrap(err, "limit check failed", errors.WithSpan(span))
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &groupPath,
			Action:        models.ActionCreate,
			TargetType:    models.TargetServiceAccount,
			TargetID:      createdServiceAccount.Metadata.ID,
		}); err != nil {
		return nil, errors.Wrap(err, "failed to create activity event", errors.WithSpan(span))
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, errors.Wrap(err, "failed to commit DB transaction", errors.WithSpan(span))
	}

	s.logger.WithContextFields(ctx).Infow("Created a service account.",
		"serviceAccountID", createdServiceAccount.Metadata.ID,
	)

	return &Response{
		ServiceAccount: createdServiceAccount,
		ClientSecret:   clientSecret,
	}, nil
}

func (s *service) UpdateServiceAccount(ctx context.Context, input *UpdateServiceAccountInput) (*Response, error) {
	ctx, span := tracer.Start(ctx, "svc.UpdateServiceAccount")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "caller authorization failed", errors.WithSpan(span))
	}

	serviceAccount, err := s.dbClient.ServiceAccounts.GetServiceAccountByID(ctx, input.ID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get service account", errors.WithSpan(span))
	}

	if serviceAccount == nil {
		return nil, errors.New("service account not found", errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	err = caller.RequirePermission(ctx, models.UpdateServiceAccountPermission, auth.WithGroupID(serviceAccount.GroupID))
	if err != nil {
		return nil, errors.Wrap(err, "permission check failed", errors.WithSpan(span))
	}

	if input.MetadataVersion != nil {
		serviceAccount.Metadata.Version = *input.MetadataVersion
	}

	if input.Description != nil {
		serviceAccount.Description = *input.Description
	}

	if input.OIDCTrustPolicies != nil {
		serviceAccount.OIDCTrustPolicies = input.OIDCTrustPolicies
	}

	var clientSecret *string

	if input.EnableClientCredentials != nil {
		if *input.EnableClientCredentials && !serviceAccount.ClientCredentialsEnabled() {
			secret, gErr := serviceAccount.GenerateClientSecret(input.ClientSecretExpiresAt, s.secretMaxExpirationDays)
			if gErr != nil {
				return nil, errors.Wrap(gErr, "failed to generate client secret", errors.WithSpan(span))
			}

			clientSecret = &secret
		} else if !*input.EnableClientCredentials {
			serviceAccount.ClientSecretHash = nil
			serviceAccount.ClientSecretExpiresAt = nil
		}
	}

	if err = serviceAccount.Validate(); err != nil {
		return nil, errors.Wrap(err, "failed to validate service account model", errors.WithSpan(span))
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin DB transaction", errors.WithSpan(span))
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.WithContextFields(ctx).Errorf("failed to rollback tx for service layer UpdateServiceAccount: %v", txErr)
		}
	}()

	updatedServiceAccount, err := s.dbClient.ServiceAccounts.UpdateServiceAccount(txContext, serviceAccount)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update service account", errors.WithSpan(span))
	}

	groupPath := updatedServiceAccount.GetGroupPath()

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &groupPath,
			Action:        models.ActionUpdate,
			TargetType:    models.TargetServiceAccount,
			TargetID:      updatedServiceAccount.Metadata.ID,
		}); err != nil {
		return nil, errors.Wrap(err, "failed to create activity event", errors.WithSpan(span))
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, errors.Wrap(err, "failed to commit DB transaction", errors.WithSpan(span))
	}

	s.logger.WithContextFields(ctx).Infow("Updated a service account.",
		"serviceAccountID", serviceAccount.Metadata.ID,
	)

	return &Response{
		ServiceAccount: updatedServiceAccount,
		ClientSecret:   clientSecret,
	}, nil
}

func (s *service) CreateOIDCToken(ctx context.Context, input *CreateOIDCTokenInput) (*CreateTokenResponse, error) {
	ctx, span := tracer.Start(ctx, "svc.CreateOIDCToken")
	defer span.End()

	// Validate service account ID
	var validationErrors []string
	serviceAccountID := strings.TrimSpace(input.ServiceAccountPublicID)

	if serviceAccountID == "" {
		validationErrors = append(validationErrors, "service account ID is empty")
	} else if types.IsTRN(serviceAccountID) {
		_, resourcePathErr := types.ServiceAccountModelType.ResourcePathFromTRN(serviceAccountID)
		if resourcePathErr != nil {
			validationErrors = append(validationErrors, fmt.Sprintf("service account path is not valid - %s", resourcePathErr.Error()))
		}
	}

	// Check if token is empty
	if len(bytes.TrimSpace(input.Token)) == 0 {
		validationErrors = append(validationErrors, "service account token is empty")
	}

	// If there are validation errors, return them all
	if len(validationErrors) > 0 {
		errorMsg := strings.Join(validationErrors, "; ")
		s.logger.WithContextFields(ctx).Infof("Failed to create token for service account: %s", errorMsg)
		return nil, errors.New(errorMsg, errors.WithErrorCode(errors.EUnauthorized))
	}

	// Parse token
	token, err := jwt.Parse(input.Token, jwt.WithVerify(false))
	if err != nil {
		// Check if the error is due to token expiration
		if strings.Contains(err.Error(), expiredTokenDetector) {
			s.logger.WithContextFields(ctx).Infof("Failed to create token for service account %s; token is expired", input.ServiceAccountPublicID)
			return nil, errors.New("failed to create token for service account %s - token is expired", input.ServiceAccountPublicID,
				errors.WithErrorCode(errors.EUnauthorized),
			)
		}
		s.logger.WithContextFields(ctx).Infof("Failed to create token for service account %s; token is not a valid JWT", input.ServiceAccountPublicID)
		return nil, errors.Wrap(err, "failed to create token for service account %s - token is not a valid JWT", input.ServiceAccountPublicID,
			errors.WithErrorCode(errors.EUnauthorized),
		)
	}

	// Check if token is from a valid issuer associated with the service account
	issuer := token.Issuer()
	if issuer == "" {
		return nil, errors.Wrap(
			err,
			"failed to create token for service account %s - issuer claim in token is empty",
			input.ServiceAccountPublicID,
			errors.WithErrorCode(errors.EUnauthorized),
		)
	}

	// Get service account based on the ID type (TRN or GID)
	var serviceAccount *models.ServiceAccount
	if types.IsTRN(input.ServiceAccountPublicID) {
		serviceAccount, err = s.dbClient.ServiceAccounts.GetServiceAccountByTRN(ctx, input.ServiceAccountPublicID)
	} else {
		serviceAccount, err = s.dbClient.ServiceAccounts.GetServiceAccountByID(ctx, gid.FromGlobalID(input.ServiceAccountPublicID))
	}

	if err != nil || serviceAccount == nil {
		s.logger.WithContextFields(ctx).Infof("Failed to create token for service account; service account %s does not exist", input.ServiceAccountPublicID)
		return nil, createGenericOIDCLoginError(input.ServiceAccountPublicID, token)
	}

	trustPolicies := s.findMatchingTrustPolicies(issuer, serviceAccount.OIDCTrustPolicies)
	if len(trustPolicies) == 0 {
		s.logger.WithContextFields(ctx).Infof("Failed to create token for service account %s; issuer %s not found in trust policy", serviceAccount.Metadata.TRN, issuer)
		return nil, createGenericOIDCLoginError(input.ServiceAccountPublicID, token)
	}

	// One satisfied trust policy is sufficient for service account token creation.
	// However, must keep all the failures in case everything fails.
	mismatchesFound := []string{}
	for _, trustPolicy := range trustPolicies {

		err := s.verifyOneTrustPolicy(ctx, input.Token, trustPolicy)
		if err != nil {

			// Catch bubbled-up invalid token signature errors here.
			if strings.Contains(err.Error(), failedToVerifyJWSSignature) {
				s.logger.WithContextFields(ctx).Infof("Failed to create token for service account %s due to invalid token signature",
					serviceAccount.Metadata.TRN)
				return nil, createGenericOIDCLoginError(input.ServiceAccountPublicID, token)
			}

			// Record this claim mismatch in case no other, later trust policy is satisfied
			mismatchesFound = append(mismatchesFound, err.Error())
		} else {
			// The input token satisfied this trust policy, so the service account token creation succeeded.
			return s.generateToken(ctx, serviceAccount, GrantTypeOIDCRelyingParty)
		}
	}

	// Log all the mismatches found so we can look them up if needed.
	s.logger.WithContextFields(ctx).Infof("failed to create service account token for issuer %s; %s", issuer, strings.Join(mismatchesFound, "; "))

	// We know there was at least one trust policy checked, otherwise we would have returned before the for loop.
	// To get here, all of the trust policies that were checked must have failed.
	return nil, createGenericOIDCLoginError(input.ServiceAccountPublicID, token)
}

func (s *service) CreateClientCredentialsToken(ctx context.Context, input *CreateClientCredentialsTokenInput) (*CreateTokenResponse, error) {
	ctx, span := tracer.Start(ctx, "svc.CreateClientCredentialsToken")
	defer span.End()

	if input.ClientID == "" || input.ClientSecret == "" {
		tracing.RecordError(span, nil, "client credentials are required")
		return nil, errFailedCreateClientCredentialsToken
	}

	var serviceAccount *models.ServiceAccount
	var err error
	if types.IsTRN(input.ClientID) {
		serviceAccount, err = s.dbClient.ServiceAccounts.GetServiceAccountByTRN(ctx, input.ClientID)
	} else {
		serviceAccount, err = s.dbClient.ServiceAccounts.GetServiceAccountByID(ctx, gid.FromGlobalID(input.ClientID))
	}

	if err != nil {
		tracing.RecordError(span, err, "failed to get service account")
		return nil, errFailedCreateClientCredentialsToken
	}

	if serviceAccount == nil {
		s.logger.WithContextFields(ctx).Infof("Service account %s not found for client credentials authentication.", input.ClientID)
		tracing.RecordError(span, nil, "service account not found")
		return nil, errFailedCreateClientCredentialsToken
	}

	if !serviceAccount.ClientCredentialsEnabled() {
		s.logger.WithContextFields(ctx).Infof("Client credentials not enabled for service account %s.", serviceAccount.Metadata.ID)
		tracing.RecordError(span, nil, "client credentials not enabled")
		return nil, errFailedCreateClientCredentialsToken
	}

	if !serviceAccount.VerifyClientSecret(input.ClientSecret) {
		s.logger.WithContextFields(ctx).Infof("Invalid or expired client secret for service account %s.", serviceAccount.Metadata.ID)
		tracing.RecordError(span, nil, "invalid client credentials")
		return nil, errFailedCreateClientCredentialsToken
	}

	return s.generateToken(ctx, serviceAccount, GrantTypeClientCredentials)
}

func (s *service) ResetClientCredentials(ctx context.Context, input *ResetClientCredentialsInput) (*Response, error) {
	ctx, span := tracer.Start(ctx, "svc.ResetClientCredentials")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "caller authorization failed", errors.WithSpan(span))
	}

	serviceAccount, err := s.dbClient.ServiceAccounts.GetServiceAccountByID(ctx, input.ID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get service account", errors.WithSpan(span))
	}

	if serviceAccount == nil {
		return nil, errors.New("service account not found", errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	err = caller.RequirePermission(ctx, models.UpdateServiceAccountPermission, auth.WithGroupID(serviceAccount.GroupID))
	if err != nil {
		return nil, errors.Wrap(err, "permission check failed", errors.WithSpan(span))
	}

	if !serviceAccount.ClientCredentialsEnabled() {
		return nil, errors.New("client credentials are not enabled for this service account", errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
	}

	secret, err := serviceAccount.GenerateClientSecret(input.ClientSecretExpiresAt, s.secretMaxExpirationDays)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate client secret", errors.WithSpan(span))
	}

	updatedServiceAccount, err := s.dbClient.ServiceAccounts.UpdateServiceAccount(ctx, serviceAccount)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update service account", errors.WithSpan(span))
	}

	s.logger.WithContextFields(ctx).Infow("Reset client credentials for service account.",
		"serviceAccountID", serviceAccount.Metadata.ID,
	)

	return &Response{
		ServiceAccount: updatedServiceAccount,
		ClientSecret:   &secret,
	}, nil
}

// generateToken creates a service account token for the given service account
func (s *service) generateToken(ctx context.Context, serviceAccount *models.ServiceAccount, grantType string) (*CreateTokenResponse, error) {
	expiration := time.Now().Add(serviceAccountLoginDuration)
	token, err := s.signingKeyManager.GenerateToken(ctx, &auth.TokenInput{
		Expiration: &expiration,
		Subject:    serviceAccount.GetResourcePath(),
		Claims: map[string]string{
			"service_account_name": serviceAccount.Name,
			"service_account_path": serviceAccount.GetResourcePath(),
			"service_account_id":   serviceAccount.GetGlobalID(),
			"type":                 auth.ServiceAccountTokenType,
			"grant_type":           grantType,
		},
	})
	if err != nil {
		return nil, err
	}

	return &CreateTokenResponse{
		Token:     token,
		ExpiresIn: int32(serviceAccountLoginDuration / time.Second),
	}, nil
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
