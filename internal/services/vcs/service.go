package vcs

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/bmatcuk/doublestar/v4"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/google/uuid"
	"github.com/hashicorp/go-getter"
	"github.com/hashicorp/go-slug"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/asynctask"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth/permissions"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/limits"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/vcs/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/workspace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

const (
	// defaultSleepDuration is used when polling the API for a status change.
	defaultSleepDuration = time.Second * 10

	// tokenExpirationLeeway is the headroom given to renew an
	// access token before it expires.
	tokenExpirationLeeway = time.Minute

	// oAuthCallBackEndpoint is the Tharsis endpoint VCS providers use
	// as a callback for completing the OAuth flow.
	oAuthCallBackEndpoint = "v1/vcs/auth/callback"

	// options for creating a temporary TarFile
	tarFlagWrite = os.O_CREATE | os.O_TRUNC | os.O_WRONLY
	tarMode      = 0o600
)

var (
	// Un-tarring of repository archive done with Hashicorp's go-getter library.
	tgz = getter.TarGzipDecompressor{}

	// refPrefixes is a slice of prefixes that _must_ be removed
	// before matching branch, or tag filters.
	refPrefixes = []string{
		"refs/heads/",
		"refs/tags/",

		// More can be added here as needed for other providers.
	}
)

// GetVCSProvidersInput is the input for listing VCS providers.
type GetVCSProvidersInput struct {
	// Sort specifies the field to sort on and direction
	Sort *db.VCSProviderSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// Search returns only the VCS providers with a name or resource path that starts with the value of search
	Search *string
	// NamespacePath is the namespace to return VCS providers for
	NamespacePath string
	// IncludeInherited includes inherited VCS providers in the result
	IncludeInherited bool
}

// GetVCSEventsInput is the input for retrieving VCSEvents.
type GetVCSEventsInput struct {
	Sort              *db.VCSEventSortableField
	PaginationOptions *pagination.Options
	WorkspaceID       string
}

// CreateVCSProviderInput is the input for creating a VCS provider.
type CreateVCSProviderInput struct {
	URL                *string
	Name               string
	Description        string
	GroupID            string
	OAuthClientID      string
	OAuthClientSecret  string
	Type               models.VCSProviderType
	AutoCreateWebhooks bool
}

// UpdateVCSProviderInput is the input for updating a VCS provider.
type UpdateVCSProviderInput struct {
	Provider *models.VCSProvider
}

// DeleteVCSProviderInput is the input for deleting a VCS provider.
type DeleteVCSProviderInput struct {
	Provider *models.VCSProvider
	Force    bool
}

// CreateWorkspaceVCSProviderLinkInput is the input for creating a VCS provider link.
type CreateWorkspaceVCSProviderLinkInput struct {
	Workspace           *models.Workspace
	ModuleDirectory     *string
	Branch              *string
	TagRegex            *string
	ProviderID          string
	RepositoryPath      string
	GlobPatterns        []string
	AutoSpeculativePlan bool
	WebhookDisabled     bool
}

// UpdateWorkspaceVCSProviderLinkInput is the input for updating a VCS provider link.
type UpdateWorkspaceVCSProviderLinkInput struct {
	Link *models.WorkspaceVCSProviderLink
}

// DeleteWorkspaceVCSProviderLinkInput is the input for deleting a workspace VCS provider link.
type DeleteWorkspaceVCSProviderLinkInput struct {
	Link  *models.WorkspaceVCSProviderLink
	Force bool
}

// CreateWorkspaceVCSProviderLinkResponse is the response for creating a workspace vcs provider link.
type CreateWorkspaceVCSProviderLinkResponse struct {
	WebhookURL   *string
	Link         *models.WorkspaceVCSProviderLink
	WebhookToken []byte
}

// CreateVCSRunInput is the input for creating a run via VCS.
type CreateVCSRunInput struct {
	Workspace     *models.Workspace
	ReferenceName *string // Optional branch, commit hash or tag name to clone.
	IsDestroy     bool
}

// ResetVCSProviderOAuthTokenInput is the input for
type ResetVCSProviderOAuthTokenInput struct {
	VCSProvider *models.VCSProvider
}

// ResetVCSProviderOAuthTokenResponse is the response for resetting a VCS OAuth token.
type ResetVCSProviderOAuthTokenResponse struct {
	VCSProvider           *models.VCSProvider
	OAuthAuthorizationURL string
}

// CreateVCSProviderResponse is the response for creating a VCS provider
type CreateVCSProviderResponse struct {
	VCSProvider           *models.VCSProvider
	OAuthAuthorizationURL string
}

// ProcessWebhookEventInput is the input for processing a webhook event.
type ProcessWebhookEventInput struct {
	EventHeader      string
	Action           string // Type of action for a MR / PR.
	SourceRepository string // Repository from which the MR originated.
	SourceBranch     string // Source branch from which the MR originated.
	TargetBranch     string // Branch this MR is for.
	HeadCommitID     string // Head commit for an MR.
	Before           string // Commit SHA before the change (can be empty).
	After            string // Commit SHA after the change  (can be empty).
	Ref              string // Ref name starting with refs/heads or similar.
}

// ProcessOAuthInput is the input for processing OAuth callback.
type ProcessOAuthInput struct {
	AuthorizationCode string
	State             string
}

// handleEventInput is the input for handling a webhook event.
type handleEventInput struct {
	provider            Provider
	processInput        *ProcessWebhookEventInput
	link                *models.WorkspaceVCSProviderLink
	workspace           *models.Workspace
	vcsEvent            *models.VCSEvent
	providerURL         url.URL
	accessToken         string
	repositorySizeLimit int
}

// downloadRepositoryArchiveInput is the input for downloading a repository archive.
type downloadRepositoryArchiveInput struct {
	provider            Provider
	providerURL         url.URL
	accessToken         string
	repositoryPath      string
	referenceName       string
	repositorySizeLimit int
}

// handleVCSRunInput is the input for handling a manual vcs run.
type handleVCSRunInput struct {
	link          *models.WorkspaceVCSProviderLink
	workspace     *models.Workspace
	vcsEvent      *models.VCSEvent
	provider      Provider
	accessToken   string
	providerURL   url.URL
	referenceName string
	isDestroy     bool
}

// createUploadConfigurationVersionInput is the input for creating and uploading
// a configuration version.
type createUploadConfigurationVersionInput struct {
	vcsEvent      *models.VCSEvent
	link          *models.WorkspaceVCSProviderLink
	repoDirectory string
}

// Service implements all the functionality related to version control providers.
type Service interface {
	GetVCSProviderByID(ctx context.Context, id string) (*models.VCSProvider, error)
	GetVCSProviders(ctx context.Context, input *GetVCSProvidersInput) (*db.VCSProvidersResult, error)
	GetVCSProvidersByIDs(ctx context.Context, idList []string) ([]models.VCSProvider, error)
	CreateVCSProvider(ctx context.Context, input *CreateVCSProviderInput) (*CreateVCSProviderResponse, error)
	UpdateVCSProvider(ctx context.Context, input *UpdateVCSProviderInput) (*models.VCSProvider, error)
	DeleteVCSProvider(ctx context.Context, input *DeleteVCSProviderInput) error
	GetWorkspaceVCSProviderLinkByID(ctx context.Context, id string) (*models.WorkspaceVCSProviderLink, error)
	GetWorkspaceVCSProviderLinkByWorkspaceID(ctx context.Context, workspaceID string) (*models.WorkspaceVCSProviderLink, error)
	CreateWorkspaceVCSProviderLink(ctx context.Context, input *CreateWorkspaceVCSProviderLinkInput) (*CreateWorkspaceVCSProviderLinkResponse, error)
	UpdateWorkspaceVCSProviderLink(ctx context.Context, input *UpdateWorkspaceVCSProviderLinkInput) (*models.WorkspaceVCSProviderLink, error)
	DeleteWorkspaceVCSProviderLink(ctx context.Context, input *DeleteWorkspaceVCSProviderLinkInput) error
	GetVCSEventByID(ctx context.Context, id string) (*models.VCSEvent, error)
	GetVCSEvents(ctx context.Context, input *GetVCSEventsInput) (*db.VCSEventsResult, error)
	GetVCSEventsByIDs(ctx context.Context, idList []string) ([]models.VCSEvent, error)
	CreateVCSRun(ctx context.Context, input *CreateVCSRunInput) error
	ProcessWebhookEvent(ctx context.Context, input *ProcessWebhookEventInput) error
	ResetVCSProviderOAuthToken(ctx context.Context, input *ResetVCSProviderOAuthTokenInput) (*ResetVCSProviderOAuthTokenResponse, error)
	ProcessOAuth(ctx context.Context, input *ProcessOAuthInput) error
}

type service struct {
	logger              logger.Logger
	dbClient            *db.Client
	limitChecker        limits.LimitChecker
	idp                 auth.IdentityProvider
	vcsProviderMap      map[models.VCSProviderType]Provider
	activityService     activityevent.Service
	runService          run.Service
	workspaceService    workspace.Service
	taskManager         asynctask.Manager
	oAuthStateGenerator func() (uuid.UUID, error) // Overriding for unit tests.
	tharsisURL          string
	repositorySizeLimit int
}

// NewService creates an instance of Service
func NewService(
	ctx context.Context,
	logger logger.Logger,
	dbClient *db.Client,
	limitChecker limits.LimitChecker,
	idp auth.IdentityProvider,
	httpClient *http.Client,
	activityService activityevent.Service,
	runService run.Service,
	workspaceService workspace.Service,
	taskManager asynctask.Manager,
	tharsisURL string,
	repositorySizeLimit int,
) (Service, error) {
	vcsProviderMap, err := NewVCSProviderMap(ctx, logger, httpClient, tharsisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize vcs provider map %v", err)
	}

	return newService(
		logger,
		dbClient,
		limitChecker,
		idp,
		vcsProviderMap,
		activityService,
		runService,
		workspaceService,
		taskManager,
		uuid.NewRandom,
		tharsisURL,
		repositorySizeLimit,
	), nil
}

func newService(
	logger logger.Logger,
	dbClient *db.Client,
	limitChecker limits.LimitChecker,
	idp auth.IdentityProvider,
	vcsProviderMap map[models.VCSProviderType]Provider,
	activityService activityevent.Service,
	runService run.Service,
	workspaceService workspace.Service,
	taskManager asynctask.Manager,
	oAuthStateGenerator func() (uuid.UUID, error),
	tharsisURL string,
	repositorySizeLimit int,
) Service {
	return &service{
		logger,
		dbClient,
		limitChecker,
		idp,
		vcsProviderMap,
		activityService,
		runService,
		workspaceService,
		taskManager,
		oAuthStateGenerator,
		tharsisURL,
		repositorySizeLimit,
	}
}

func (s *service) GetVCSProviderByID(ctx context.Context, id string) (*models.VCSProvider, error) {
	ctx, span := tracer.Start(ctx, "svc.GetVCSProviderByID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	provider, err := s.dbClient.VCSProviders.GetProviderByID(ctx, id)
	if err != nil {
		tracing.RecordError(span, err, "failed to get provider by ID")
		return nil, err
	}

	if provider == nil {
		tracing.RecordError(span, nil, "VCS provider with ID %s not found", id)
		return nil, errors.New("VCS provider with ID %s not found", id, errors.WithErrorCode(errors.ENotFound))
	}

	err = caller.RequirePermission(ctx, permissions.ViewVCSProviderPermission, auth.WithGroupID(provider.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	return provider, nil
}

func (s *service) GetVCSProviders(ctx context.Context, input *GetVCSProvidersInput) (*db.VCSProvidersResult, error) {
	ctx, span := tracer.Start(ctx, "svc.GetVCSProviders")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.ViewVCSProviderPermission, auth.WithNamespacePath(input.NamespacePath))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	filter := &db.VCSProviderFilter{
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
		// don't have VCS providers directly associated (i.e. only group namespaces do)
		filter.NamespacePaths = []string{input.NamespacePath}
	}

	result, err := s.dbClient.VCSProviders.GetProviders(ctx, &db.GetVCSProvidersInput{
		Sort:              input.Sort,
		PaginationOptions: input.PaginationOptions,
		Filter:            filter,
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to get providers")
		return nil, err
	}

	return result, nil
}

func (s *service) GetVCSProvidersByIDs(ctx context.Context, idList []string) ([]models.VCSProvider, error) {
	ctx, span := tracer.Start(ctx, "svc.GetVCSProvidersByIDs")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	result, err := s.dbClient.VCSProviders.GetProviders(ctx, &db.GetVCSProvidersInput{
		Filter: &db.VCSProviderFilter{
			VCSProviderIDs: idList,
		},
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to get providers")
		return nil, err
	}

	namespacePaths := []string{}
	for _, vp := range result.VCSProviders {
		namespacePaths = append(namespacePaths, vp.GetGroupPath())
	}

	if len(namespacePaths) > 0 {
		err = caller.RequireAccessToInheritableResource(ctx, permissions.VCSProviderResourceType, auth.WithNamespacePaths(namespacePaths))
		if err != nil {
			tracing.RecordError(span, err, "inheritable resource access check failed")
			return nil, err
		}
	}

	return result.VCSProviders, nil
}

func (s *service) CreateVCSProvider(ctx context.Context, input *CreateVCSProviderInput) (*CreateVCSProviderResponse, error) {
	ctx, span := tracer.Start(ctx, "svc.CreateVCSProvider")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.CreateVCSProviderPermission, auth.WithGroupID(input.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	// Check if provider is supported.
	provider, err := s.getVCSProvider(input.Type)
	if err != nil {
		tracing.RecordError(span, err, "failed to get VCS provider")
		return nil, err
	}

	// Use the default providerURL if nothing provided.
	var providerURL url.URL
	if input.URL == nil {
		providerURL = provider.DefaultURL()
	} else {
		parsedURL, uErr := url.Parse(*input.URL)
		if uErr != nil || (parsedURL.Scheme == "") || (parsedURL.Host == "") {
			tracing.RecordError(span, nil, "Invalid provider URL")
			return nil, errors.New("Invalid provider URL", errors.WithErrorCode(errors.EInvalid))
		}

		// Remove any trailing backslash.
		parsedURL.Path = strings.TrimSuffix(parsedURL.Path, "/")
		providerURL = *parsedURL
	}

	// Use a UUID for the state.
	oAuthState, err := s.oAuthStateGenerator()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate an OAuth state")
		return nil, err
	}

	// Must be a pointer.
	oAuthStateString := oAuthState.String()

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer CreateVCSProvider: %v", txErr)
		}
	}()

	toCreate := &models.VCSProvider{
		Name:               input.Name,
		Description:        input.Description,
		CreatedBy:          caller.GetSubject(),
		GroupID:            input.GroupID,
		URL:                providerURL,
		OAuthClientID:      input.OAuthClientID,
		OAuthClientSecret:  input.OAuthClientSecret,
		OAuthState:         &oAuthStateString,
		Type:               input.Type,
		AutoCreateWebhooks: input.AutoCreateWebhooks,
	}

	if err = toCreate.Validate(); err != nil {
		tracing.RecordError(span, err, "failed to validate VCS provider model")
		return nil, err
	}

	createdProvider, err := s.dbClient.VCSProviders.CreateProvider(txContext, toCreate)
	if err != nil {
		tracing.RecordError(span, err, "failed to create provider")
		return nil, err
	}

	groupPath := createdProvider.GetGroupPath()

	// Get the number of VCS providers in the group to check whether we just violated the limit.
	newVCSProviders, err := s.dbClient.VCSProviders.GetProviders(txContext, &db.GetVCSProvidersInput{
		Filter: &db.VCSProviderFilter{
			NamespacePaths: []string{groupPath},
		},
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(0),
		},
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to get VCS providers")
		return nil, err
	}
	if err = s.limitChecker.CheckLimit(txContext,
		limits.ResourceLimitVCSProvidersPerGroup, newVCSProviders.PageInfo.TotalCount); err != nil {
		tracing.RecordError(span, err, "limit check failed")
		return nil, err
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &groupPath,
			Action:        models.ActionCreate,
			TargetType:    models.TargetVCSProvider,
			TargetID:      createdProvider.Metadata.ID,
		}); err != nil {
		tracing.RecordError(span, err, "failed to create activity event")
		return nil, err
	}

	if err = s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return nil, err
	}

	s.logger.Infow("Created a VCS provider.",
		"caller", caller.GetSubject(),
		"name", input.Name,
		"groupID", input.GroupID,
		"type", input.Type,
	)

	authorizationURL, err := s.getOAuthAuthorizationURL(ctx, createdProvider)
	if err != nil {
		tracing.RecordError(span, err, "failed to get authorization URL")
		return nil, err
	}

	return &CreateVCSProviderResponse{
		VCSProvider:           createdProvider,
		OAuthAuthorizationURL: authorizationURL,
	}, nil
}

func (s *service) UpdateVCSProvider(ctx context.Context, input *UpdateVCSProviderInput) (*models.VCSProvider, error) {
	ctx, span := tracer.Start(ctx, "svc.UpdateVCSProvider")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.UpdateVCSProviderPermission, auth.WithGroupID(input.Provider.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	if err = input.Provider.Validate(); err != nil {
		tracing.RecordError(span, err, "failed to validate VCS provider model")
		return nil, err
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer UpdateVCSProvider: %v", txErr)
		}
	}()

	updatedProvider, err := s.dbClient.VCSProviders.UpdateProvider(txContext, input.Provider)
	if err != nil {
		tracing.RecordError(span, err, "failed to update provider")
		return nil, err
	}

	groupPath := updatedProvider.GetGroupPath()

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &groupPath,
			Action:        models.ActionUpdate,
			TargetType:    models.TargetVCSProvider,
			TargetID:      updatedProvider.Metadata.ID,
		}); err != nil {
		tracing.RecordError(span, err, "failed to create activity event")
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return nil, err
	}

	s.logger.Infow("Updated a VCS provider.",
		"caller", caller.GetSubject(),
		"name", input.Provider.Name,
		"groupID", input.Provider.GroupID,
		"type", input.Provider.Type,
	)

	return updatedProvider, nil
}

func (s *service) DeleteVCSProvider(ctx context.Context, input *DeleteVCSProviderInput) error {
	ctx, span := tracer.Start(ctx, "svc.DeleteVCSProvider")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return err
	}

	err = caller.RequirePermission(ctx, permissions.DeleteVCSProviderPermission, auth.WithGroupID(input.Provider.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return err
	}

	// Verify the provider does not have any links.
	links, gErr := s.dbClient.WorkspaceVCSProviderLinks.GetLinksByProviderID(ctx, input.Provider.Metadata.ID)
	if gErr != nil {
		tracing.RecordError(span, gErr, "failed to commit DB transaction")
		return gErr
	}

	if !input.Force && len(links) > 0 {
		tracing.RecordError(span, nil,
			"This VCS provider can't be deleted because it's currently linked to %d workspaces. "+
				"Setting force to true will automatically remove all associated links for this provider.", len(links))
		return errors.New(
			"This VCS provider can't be deleted because it's currently linked to %d workspaces. "+
				"Setting force to true will automatically remove all associated links for this provider.", len(links),
			errors.WithErrorCode(errors.EConflict),
		)
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer DeleteVCSProvider: %v", txErr)
		}
	}()

	err = s.dbClient.VCSProviders.DeleteProvider(txContext, input.Provider)
	if err != nil {
		tracing.RecordError(span, err, "failed to delete provider")
		return err
	}

	// Delete all webhooks associated with provider.
	if input.Provider.AutoCreateWebhooks && len(links) > 0 {
		provider, gErr := s.getVCSProvider(input.Provider.Type)
		if gErr != nil {
			tracing.RecordError(span, gErr, "failed to get VCS provider")
			return gErr
		}

		// Get a new access token.
		accessToken, rErr := s.refreshOAuthToken(ctx, provider, input.Provider, true)
		if rErr != nil {
			tracing.RecordError(span, rErr, "failed to refresh OAuth token")
			return fmt.Errorf("failed to refresh access token: %v", rErr)
		}

		for _, link := range links {
			err = provider.DeleteWebhook(ctx, &types.DeleteWebhookInput{
				ProviderURL:    input.Provider.URL,
				AccessToken:    accessToken,
				RepositoryPath: link.RepositoryPath,
				WebhookID:      link.WebhookID,
			})
			if err != nil {
				tracing.RecordError(span, err, "failed to delete webhook")
				return err
			}
		}
	}

	groupPath := input.Provider.GetGroupPath()
	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &groupPath,
			Action:        models.ActionDeleteChildResource,
			TargetType:    models.TargetGroup,
			TargetID:      input.Provider.GroupID,
			Payload: &models.ActivityEventDeleteChildResourcePayload{
				Name: input.Provider.Name,
				ID:   input.Provider.Metadata.ID,
				Type: string(models.TargetVCSProvider),
			},
		}); err != nil {
		tracing.RecordError(span, err, "failed to create activity event")
		return err
	}

	s.logger.Infow("Deleted a VCS provider.",
		"caller", caller.GetSubject(),
		"name", input.Provider.Name,
		"groupID", input.Provider.GroupID,
		"type", input.Provider.Type,
	)

	return s.dbClient.Transactions.CommitTx(txContext)
}

func (s *service) GetWorkspaceVCSProviderLinkByWorkspaceID(ctx context.Context, workspaceID string) (*models.WorkspaceVCSProviderLink, error) {
	ctx, span := tracer.Start(ctx, "svc.GetWorkspaceVCSProviderLinkByWorkspaceID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.ViewWorkspacePermission, auth.WithWorkspaceID(workspaceID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	link, err := s.dbClient.WorkspaceVCSProviderLinks.GetLinkByWorkspaceID(ctx, workspaceID)
	if err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return nil, err
	}

	if link == nil {
		tracing.RecordError(span, nil, "failed to commit DB transaction")
		return nil, errors.New("workspace vcs provider link for workspace ID %s not found", workspaceID, errors.WithErrorCode(errors.ENotFound))
	}

	return link, nil
}

func (s *service) GetWorkspaceVCSProviderLinkByID(ctx context.Context, id string) (*models.WorkspaceVCSProviderLink, error) {
	ctx, span := tracer.Start(ctx, "svc.GetWorkspaceVCSProviderLinkByID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	link, err := s.dbClient.WorkspaceVCSProviderLinks.GetLinkByID(ctx, id)
	if err != nil {
		tracing.RecordError(span, err, "failed to get link by ID")
		return nil, err
	}

	if link == nil {
		tracing.RecordError(span, nil, "workspace vcs provider link with ID %s not found", id)
		return nil, errors.New("workspace vcs provider link with ID %s not found", id, errors.WithErrorCode(errors.ENotFound))
	}

	err = caller.RequirePermission(ctx, permissions.ViewWorkspacePermission, auth.WithWorkspaceID(link.WorkspaceID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	return link, nil
}

func (s *service) CreateWorkspaceVCSProviderLink(ctx context.Context, input *CreateWorkspaceVCSProviderLinkInput) (*CreateWorkspaceVCSProviderLinkResponse, error) {
	ctx, span := tracer.Start(ctx, "svc.CreateWorkspaceVCSProviderLink")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.UpdateWorkspacePermission, auth.WithWorkspaceID(input.Workspace.Metadata.ID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	// Make sure the VCS provider exists. Also, used to configure it.
	vp, err := s.dbClient.VCSProviders.GetProviderByID(ctx, input.ProviderID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get provider by ID")
		return nil, err
	}

	if vp == nil {
		tracing.RecordError(span, nil, "failed to get provider by ID")
		return nil, errors.New("vcs provider with id %s not found", input.ProviderID, errors.WithErrorCode(errors.EInvalid))
	}

	// Get the group path.
	groupPath := vp.ResourcePath[:strings.LastIndex(vp.ResourcePath, "/")+1]

	// Verify that the vcs provider's group is in the same hierarchy as the workspace.
	if !strings.HasPrefix(input.Workspace.FullPath, groupPath) {
		tracing.RecordError(span, nil,
			"VCS provider %s is not available to workspace %s", vp.ResourcePath, input.Workspace.FullPath)
		return nil, errors.New("VCS provider %s is not available to workspace %s", vp.ResourcePath, input.Workspace.FullPath, errors.WithErrorCode(errors.EInvalid))
	}

	// Make sure the token is there, otherwise user forgot to complete
	// the OAuth flow for the VCS provider.
	if vp.OAuthAccessToken == nil {
		tracing.RecordError(span, nil,
			"OAuth flow must be completed before linking a workspace to a VCS provider. "+
				"Either use the original authorization URL when VCS provider was created "+
				"or request another one",
		)
		return nil, errors.New(
			"OAuth flow must be completed before linking a workspace to a VCS provider. "+
				"Either use the original authorization URL when VCS provider was created "+
				"or request another one",
			errors.WithErrorCode(errors.EInvalid),
		)
	}

	provider, cErr := s.getVCSProvider(vp.Type)
	if cErr != nil {
		tracing.RecordError(span, cErr, "failed to get VCS provider")
		return nil, cErr
	}

	// Get a new access token.
	accessToken, err := s.refreshOAuthToken(ctx, provider, vp, false)
	if err != nil {
		tracing.RecordError(span, err, "failed to refresh access token")
		return nil, fmt.Errorf("failed to refresh access token: %v", err)
	}

	// Get the project, this also validates the repository exists.
	payload, gErr := provider.GetProject(ctx, &types.GetProjectInput{
		ProviderURL:    vp.URL,
		AccessToken:    accessToken,
		RepositoryPath: input.RepositoryPath,
	})
	if gErr != nil {
		tracing.RecordError(span, gErr, "failed to get projects")
		return nil, gErr
	}

	branch := payload.DefaultBranch
	if input.Branch != nil && *input.Branch != "" {
		branch = *input.Branch
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer CreateWorkspaceVCSProviderLink: %v", txErr)
		}
	}()

	jwtID := uuid.New().String()

	toCreate := &models.WorkspaceVCSProviderLink{
		CreatedBy:           caller.GetSubject(),
		WorkspaceID:         input.Workspace.Metadata.ID,
		ProviderID:          input.ProviderID,
		TokenNonce:          jwtID,
		Branch:              branch,
		RepositoryPath:      input.RepositoryPath,
		TagRegex:            input.TagRegex,
		GlobPatterns:        input.GlobPatterns,
		AutoSpeculativePlan: input.AutoSpeculativePlan,
		WebhookDisabled:     input.WebhookDisabled,
	}

	// Clean module directory path. Attempting to clean an
	// empty directory will return '.'.
	if input.ModuleDirectory != nil && *input.ModuleDirectory != "" {
		moduleDirectory := filepath.Clean(*input.ModuleDirectory)
		toCreate.ModuleDirectory = &moduleDirectory
	}

	if err = toCreate.Validate(); err != nil {
		tracing.RecordError(span, err, "failed to validate workspace VCS provider link model")
		return nil, err
	}

	createdLink, err := s.dbClient.WorkspaceVCSProviderLinks.CreateLink(txContext, toCreate)
	if err != nil {
		tracing.RecordError(span, err, "failed to create link")
		return nil, err
	}

	response := &CreateWorkspaceVCSProviderLinkResponse{}

	// Create the token and configure webhook if using them.
	// Generate a token with a UUID claim.
	token, gErr := s.idp.GenerateToken(ctx, &auth.TokenInput{
		Subject: vp.ResourcePath,
		JwtID:   createdLink.TokenNonce,
		Claims: map[string]string{
			"type":    auth.VCSWorkspaceLinkTokenType,
			"link_id": gid.ToGlobalID(gid.WorkspaceVCSProviderLinkType, createdLink.Metadata.ID),
		},
	})
	if gErr != nil {
		tracing.RecordError(span, gErr, "failed to generate token with a UUID claim")
		return nil, gErr
	}

	// If provider was set to automatically create webhook, create it.
	if vp.AutoCreateWebhooks {
		// Create the webhook.
		payload, cErr := provider.CreateWebhook(ctx, &types.CreateWebhookInput{
			ProviderURL:    vp.URL,
			AccessToken:    accessToken,
			RepositoryPath: createdLink.RepositoryPath,
			WebhookToken:   token,
		})
		if cErr != nil {
			tracing.RecordError(span, cErr, "failed to create webhook")
			return nil, cErr
		}

		// Set the webhook ID to the one just created.
		createdLink.WebhookID = payload.WebhookID

		createdLink, err = s.dbClient.WorkspaceVCSProviderLinks.UpdateLink(txContext, createdLink)
		if err != nil {
			tracing.RecordError(span, err, "failed to update link")
			return nil, err
		}
	} else {
		// Get the webhook URL based on the provider type. GitLab supports
		// passing in a token whereas GitHub does not. It must be added
		// as a query parameter for the latter.
		var webhookToken []byte
		switch vp.Type {
		case models.GitLabProviderType:
			response.WebhookToken = token // Only set token field for GitLab.
		case models.GitHubProviderType:
			webhookToken = token // For GitHub. include token as a query param.
		}

		webhookURL, wErr := getTharsisWebhookURL(s.tharsisURL, webhookToken)
		if wErr != nil {
			tracing.RecordError(span, wErr, "failed to get webhook URL")
			return nil, wErr
		}

		response.WebhookURL = &webhookURL
	}

	// Set the created link.
	response.Link = createdLink

	if err = s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return nil, err
	}

	s.logger.Infow("Created a workspace vcs provider link.",
		"caller", caller.GetSubject(),
		"workspacePath", input.Workspace.FullPath,
		"linkID", createdLink.Metadata.ID,
		"providerPath", vp.ResourcePath,
	)

	return response, nil
}

func (s *service) UpdateWorkspaceVCSProviderLink(ctx context.Context, input *UpdateWorkspaceVCSProviderLinkInput) (*models.WorkspaceVCSProviderLink, error) {
	ctx, span := tracer.Start(ctx, "svc.UpdateWorkspaceVCSProviderLink")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.UpdateWorkspacePermission, auth.WithWorkspaceID(input.Link.WorkspaceID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	if err = input.Link.Validate(); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return nil, err
	}

	s.logger.Infow("Requested an update to a workspace vcs provider link.",
		"caller", caller.GetSubject(),
		"workspaceID", input.Link.WorkspaceID,
		"linkID", input.Link.Metadata.ID,
	)

	return s.dbClient.WorkspaceVCSProviderLinks.UpdateLink(ctx, input.Link)
}

func (s *service) DeleteWorkspaceVCSProviderLink(ctx context.Context, input *DeleteWorkspaceVCSProviderLinkInput) error {
	ctx, span := tracer.Start(ctx, "svc.DeleteWorkspaceVCSProviderLink")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return err
	}

	err = caller.RequirePermission(ctx, permissions.UpdateWorkspacePermission, auth.WithWorkspaceID(input.Link.WorkspaceID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return err
	}

	// Get the provider, so we can reconfigure it.
	vp, err := s.dbClient.VCSProviders.GetProviderByID(ctx, input.Link.ProviderID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get provider by ID")
		return err
	}

	if vp == nil {
		tracing.RecordError(span, nil, "failed to get provider by ID")
		return errors.New("vcs provider with id %s not found", input.Link.ProviderID)
	}

	// If the provider was automatically configured, delete the webhook
	// that is associated with the link.
	if vp.AutoCreateWebhooks {
		provider, err := s.getVCSProvider(vp.Type)
		if err != nil {
			tracing.RecordError(span, err, "failed to get VCS provider")
			return err
		}

		// Get a new access token.
		accessToken, err := s.refreshOAuthToken(ctx, provider, vp, false)
		if err != nil && !input.Force {
			tracing.RecordError(span, nil, "failed to validate workspace VCS provider link model")
			return fmt.Errorf("error refreshing access token. "+
				"Setting force to true will delete this link but webhooks may have to be deleted manually: %v", err)
		}

		// Delete the existing webhook.
		if err = provider.DeleteWebhook(ctx, &types.DeleteWebhookInput{
			ProviderURL:    vp.URL,
			AccessToken:    accessToken,
			RepositoryPath: input.Link.RepositoryPath,
			WebhookID:      input.Link.WebhookID,
		}); err != nil && !input.Force {
			tracing.RecordError(span, nil,
				"error deleting webhook. "+
					"Setting force to true will delete this link but webhooks may have to be deleted manually: %v", err)
			return fmt.Errorf("error deleting webhook. "+
				"Setting force to true will delete this link but webhooks may have to be deleted manually: %v", err)
		}
	}

	s.logger.Infow("Requested to delete a workspace vcs provider link.",
		"caller", caller.GetSubject(),
		"workspaceID", input.Link.WorkspaceID,
		"providerName", vp.Name,
	)

	return s.dbClient.WorkspaceVCSProviderLinks.DeleteLink(ctx, input.Link)
}

func (s *service) GetVCSEventByID(ctx context.Context, id string) (*models.VCSEvent, error) {
	ctx, span := tracer.Start(ctx, "svc.GetVCSEventByID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	event, err := s.dbClient.VCSEvents.GetEventByID(ctx, id)
	if err != nil {
		tracing.RecordError(span, err, "failed to get event by ID")
		return nil, err
	}

	if event == nil {
		tracing.RecordError(span, nil, "vcs event with id %s not found", id)
		return nil, errors.New("vcs event with id %s not found", id, errors.WithErrorCode(errors.ENotFound))
	}

	err = caller.RequirePermission(ctx, permissions.ViewVCSProviderPermission, auth.WithWorkspaceID(event.WorkspaceID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	return event, nil
}

func (s *service) GetVCSEvents(ctx context.Context, input *GetVCSEventsInput) (*db.VCSEventsResult, error) {
	ctx, span := tracer.Start(ctx, "svc.GetVCSEvents")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.ViewVCSProviderPermission, auth.WithWorkspaceID(input.WorkspaceID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	dbInput := &db.GetVCSEventsInput{
		Sort:              input.Sort,
		PaginationOptions: input.PaginationOptions,
		Filter: &db.VCSEventFilter{
			WorkspaceID: &input.WorkspaceID,
		},
	}

	return s.dbClient.VCSEvents.GetEvents(ctx, dbInput)
}

func (s *service) GetVCSEventsByIDs(ctx context.Context, idList []string) ([]models.VCSEvent, error) {
	ctx, span := tracer.Start(ctx, "svc.GetVCSEventsByIDs")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	result, err := s.dbClient.VCSEvents.GetEvents(ctx, &db.GetVCSEventsInput{
		Filter: &db.VCSEventFilter{
			VCSEventIDs: idList,
		},
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to get events")
		return nil, err
	}

	for _, ve := range result.VCSEvents {
		err = caller.RequirePermission(ctx, permissions.ViewVCSProviderPermission, auth.WithWorkspaceID(ve.WorkspaceID))
		if err != nil {
			tracing.RecordError(span, err, "permission check failed")
			return nil, err
		}
	}

	return result.VCSEvents, nil
}

func (s *service) CreateVCSRun(ctx context.Context, input *CreateVCSRunInput) error {
	ctx, span := tracer.Start(ctx, "svc.CreateVCSRun")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return err
	}

	err = caller.RequirePermission(ctx, permissions.CreateRunPermission, auth.WithWorkspaceID(input.Workspace.Metadata.ID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return err
	}

	// Make sure the workspace is linked to a VCS provider.
	link, err := s.dbClient.WorkspaceVCSProviderLinks.GetLinkByWorkspaceID(ctx, input.Workspace.Metadata.ID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get link by workspace ID")
		return err
	}

	if link == nil {
		tracing.RecordError(span, nil,
			"Workspace %s is not linked to a VCS provider", input.Workspace.FullPath)
		return errors.New("Workspace %s is not linked to a VCS provider", input.Workspace.FullPath, errors.WithErrorCode(errors.EInvalid))
	}

	// Get the provider associated with the link.
	vp, err := s.dbClient.VCSProviders.GetProviderByID(ctx, link.ProviderID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get provider by ID")
		return err
	}

	// Shouldn't happen.
	if vp == nil {
		tracing.RecordError(span, nil, "failed to get provider by ID")
		return errors.New("VCS provider associated with link ID %s not found", link.Metadata.ID)
	}

	// Get the appropriate provider from the map, so we can download from it.
	provider, err := s.getVCSProvider(vp.Type)
	if err != nil {
		tracing.RecordError(span, err, "failed to get VCS provider")
		return err
	}

	accessToken, err := s.refreshOAuthToken(ctx, provider, vp, false)
	if err != nil {
		tracing.RecordError(span, err, "failed to refresh OAuth token")
		return err
	}

	var referenceName string
	if input.ReferenceName != nil && *input.ReferenceName != "" {
		// Use the provided reference name.
		referenceName = *input.ReferenceName
	} else {
		// Otherwise, use the branch on the link as default.
		referenceName = link.Branch
	}

	var (
		eventCommitID  *string
		eventSourceRef *string
	)

	if plumbing.IsHash(referenceName) {
		// Set the CommitID instead since a commit hash is provided.
		eventCommitID = &referenceName
	} else {
		// Otherwise, use the branch or tag name as SourceReferenceName.
		eventSourceRef = &referenceName
	}

	repoURL, err := provider.BuildRepositoryURL(&types.BuildRepositoryURLInput{
		ProviderURL:    vp.URL,
		RepositoryPath: link.RepositoryPath,
	})
	if err != nil {
		tracing.RecordError(span, nil, "failed to build repository url")
		return fmt.Errorf("failed to build repository url: %w", err)
	}

	// Create the VCS event with 'pending' status.
	createdEvent, err := s.dbClient.VCSEvents.CreateEvent(ctx, &models.VCSEvent{
		CommitID:            eventCommitID,
		SourceReferenceName: eventSourceRef,
		WorkspaceID:         input.Workspace.Metadata.ID,
		Type:                models.ManualEventType,
		Status:              models.VCSEventPending,
		RepositoryURL:       repoURL,
	})
	if err != nil {
		tracing.RecordError(span, nil, "failed to create a vcs event")
		return fmt.Errorf("failed to create a vcs event: %w", err)
	}

	handleVCSRunCallback := func(ctx context.Context) {
		// Update the status field beforehand.
		createdEvent.Status = models.VCSEventFinished

		if err := s.handleVCSRun(auth.WithCaller(ctx, caller), &handleVCSRunInput{
			providerURL:   vp.URL,
			accessToken:   accessToken,
			link:          link,
			workspace:     input.Workspace,
			referenceName: referenceName,
			isDestroy:     input.IsDestroy,
			vcsEvent:      createdEvent,
			provider:      provider,
		}); err != nil {
			if errors.ErrorCode(err) != errors.EForbidden {
				s.logger.Errorf("failed to process manual vcs run: %v", err)
			} else {
				// To avoid polluting the logs with false errors an Info level is used here.
				s.logger.Info(err)
			}

			// Update the status and error message on the event.
			errorMessage := err.Error() // ErrorMessage must be a pointer.
			createdEvent.Status = models.VCSEventErrored
			createdEvent.ErrorMessage = &errorMessage
		}

		// Update the vcs event. Returned model is not needed.
		if _, err := s.dbClient.VCSEvents.UpdateEvent(ctx, createdEvent); err != nil {
			s.logger.Error(
				"failed to update event for repository %s archive for workspace %s and workspace vcs provider link ID %s: %v",
				link.RepositoryPath,
				input.Workspace.FullPath,
				link.Metadata.ID,
				err,
			)
		}
	}

	// Download and create the run in a goroutine.
	s.taskManager.StartTask(handleVCSRunCallback)

	return nil
}

func (s *service) ProcessWebhookEvent(ctx context.Context, input *ProcessWebhookEventInput) error {
	ctx, span := tracer.Start(ctx, "svc.ProcessWebhookEvent")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return err
	}

	vcsCaller, ok := caller.(*auth.VCSWorkspaceLinkCaller)
	if !ok {
		tracing.RecordError(span, nil, "Invalid caller; only version control systems can invoke webhook")
		return errors.New("Invalid caller; only version control systems can invoke webhook", errors.WithErrorCode(errors.EInvalid))
	}

	// Require permission for creating plan runs.
	err = caller.RequirePermission(ctx, permissions.CreateRunPermission, auth.WithWorkspaceID(vcsCaller.Link.WorkspaceID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return err
	}

	// Get workspace so errors can be printed using the workspace path instead.
	// Mainly just to allow easier debugging incase things do go wrong.
	workspace, err := s.workspaceService.GetWorkspaceByID(ctx, vcsCaller.Link.WorkspaceID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get workspace by ID")
		return err
	}

	if vcsCaller.Link.WebhookDisabled {
		s.logger.Infof("Skipping webhook event since webhook is disabled for link ID %s, workspace %s and repository %s",
			vcsCaller.Link.Metadata.ID,
			workspace.FullPath,
			vcsCaller.Link.RepositoryPath,
		)

		// Only process webhook events if webhook is not disabled on the link.
		return nil
	}

	provider, err := s.getVCSProvider(vcsCaller.Provider.Type)
	if err != nil {
		tracing.RecordError(span, err, "failed to get VCS provider")
		return err
	}

	eventType := provider.ToVCSEventType(&types.ToVCSEventTypeInput{
		EventHeader: input.EventHeader,
		Ref:         input.Ref,
	})
	if eventType == "" {
		// Silently ignore the request rather than throwing an error.
		// This prevents GitHub from thinking the webhook is invalid
		// when it first attempts to ping it.
		return nil
	}

	// If the event ref does not match the defined filters
	// on the link, no further action is required.
	if !refMatches(input, eventType, vcsCaller.Link, provider) {
		return nil
	}

	// If the after hash is zero and this is not a merge request event,
	// then there are no changes to evaluate.
	if !eventType.Equals(models.MergeRequestEventType) && plumbing.NewHash(input.After).IsZero() {
		return nil
	}

	accessToken, err := s.refreshOAuthToken(ctx, provider, vcsCaller.Provider, false)
	if err != nil {
		tracing.RecordError(span, err, "failed to refresh access token")
		return fmt.Errorf("failed to refresh access token: %v", err)
	}

	ref := input.Ref
	commitID := input.After

	// Use the ref and commit ID appropriate for an MR / PR.
	if eventType.Equals(models.MergeRequestEventType) {
		ref = input.SourceBranch
		commitID = input.HeadCommitID
	}

	repoURL, err := provider.BuildRepositoryURL(&types.BuildRepositoryURLInput{
		ProviderURL:    vcsCaller.Provider.URL,
		RepositoryPath: vcsCaller.Link.RepositoryPath,
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to build repository URL")
		return fmt.Errorf("failed to build repository URL: %w", err)
	}

	// Create the VCS event with 'pending' status.
	createdEvent, err := s.dbClient.VCSEvents.CreateEvent(ctx, &models.VCSEvent{
		SourceReferenceName: &ref,
		CommitID:            &commitID,
		WorkspaceID:         workspace.Metadata.ID,
		Type:                eventType,
		Status:              models.VCSEventPending,
		RepositoryURL:       repoURL,
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to create a vcs event")
		return fmt.Errorf("failed to create a vcs event: %v", err)
	}

	// Build a callback for taskManager.
	handleEventCallback := func(ctx context.Context) {
		// Update the status field beforehand.
		createdEvent.Status = models.VCSEventFinished

		if err := s.handleEvent(auth.WithCaller(ctx, caller), &handleEventInput{
			providerURL:         vcsCaller.Provider.URL,
			accessToken:         accessToken,
			provider:            provider,
			processInput:        input,
			link:                vcsCaller.Link,
			workspace:           workspace,
			vcsEvent:            createdEvent,
			repositorySizeLimit: s.repositorySizeLimit,
		}); err != nil {
			if errors.ErrorCode(err) != errors.EForbidden {
				s.logger.Errorf("failed to process %s webhook event: %v", vcsCaller.Provider.Type, err)
			} else {
				// To avoid polluting the logs with false errors an Info level is used here.
				s.logger.Info(err)
			}

			// Update the status and error message on the event.
			errorMessage := err.Error() // ErrorMessage must be a pointer.
			createdEvent.Status = models.VCSEventErrored
			createdEvent.ErrorMessage = &errorMessage
		}

		// Update the vcs event. Returned model is not needed.
		if _, err := s.dbClient.VCSEvents.UpdateEvent(ctx, createdEvent); err != nil {
			s.logger.Error(
				"failed to update event for repository %s archive for workspace %s and workspace vcs provider link ID %s: %v",
				vcsCaller.Link.RepositoryPath,
				workspace.FullPath,
				vcsCaller.Link.Metadata.ID,
				err,
			)
		}
	}

	// Processing the event in its own goroutine.
	s.taskManager.StartTask(handleEventCallback)

	return nil
}

func (s *service) ResetVCSProviderOAuthToken(ctx context.Context, input *ResetVCSProviderOAuthTokenInput) (*ResetVCSProviderOAuthTokenResponse, error) {
	ctx, span := tracer.Start(ctx, "svc.ResetVCSProviderOAuthToken")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.UpdateVCSProviderPermission, auth.WithGroupID(input.VCSProvider.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	// Use a UUID for the state.
	oAuthState, err := s.oAuthStateGenerator()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate an OAuth state")
		return nil, err
	}

	oAuthStateString := oAuthState.String()

	// Update fields with state value / reset fields.
	input.VCSProvider.OAuthAccessToken = nil
	input.VCSProvider.OAuthRefreshToken = nil
	input.VCSProvider.OAuthAccessTokenExpiresAt = nil
	input.VCSProvider.OAuthState = &oAuthStateString

	updatedProvider, err := s.dbClient.VCSProviders.UpdateProvider(ctx, input.VCSProvider)
	if err != nil {
		tracing.RecordError(span, err, "failed to update provider")
		return nil, err
	}

	authorizationURL, err := s.getOAuthAuthorizationURL(ctx, updatedProvider)
	if err != nil {
		tracing.RecordError(span, err, "failed to get OAuth authorization URL")
		return nil, err
	}

	return &ResetVCSProviderOAuthTokenResponse{
		VCSProvider:           updatedProvider,
		OAuthAuthorizationURL: authorizationURL,
	}, nil
}

func (s *service) getOAuthAuthorizationURL(ctx context.Context, vcsProvider *models.VCSProvider) (string, error) {
	// Check if a valid state value is available.
	if vcsProvider.OAuthState == nil {
		return "", errors.New("oauth state is not set")
	}

	redirectURL, err := s.getOAuthCallBackURL(ctx)
	if err != nil {
		return "", err
	}

	provider, err := s.getVCSProvider(vcsProvider.Type)
	if err != nil {
		return "", err
	}

	authorizationURLInput := &types.BuildOAuthAuthorizationURLInput{
		ProviderURL:        vcsProvider.URL,
		OAuthClientID:      vcsProvider.OAuthClientID,
		OAuthState:         *vcsProvider.OAuthState,
		RedirectURL:        redirectURL,
		UseReadWriteScopes: vcsProvider.AutoCreateWebhooks,
	}

	// Build authorization code URL for the provider which
	// identity provider can use to complete OAuth flow.
	return provider.BuildOAuthAuthorizationURL(authorizationURLInput)
}

func (s *service) ProcessOAuth(ctx context.Context, input *ProcessOAuthInput) error {
	ctx, span := tracer.Start(ctx, "svc.ProcessOAuth")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return err
	}

	// Make sure the state value if a valid UUID. Avoids
	// a DB query for random calls to the endpoint.
	if _, err = uuid.Parse(input.State); err != nil {
		tracing.RecordError(span, err, "failed to get VCS provider")
		return err
	}

	// Validate the state value.
	vp, err := s.dbClient.VCSProviders.GetProviderByOAuthState(ctx, input.State)
	if err != nil {
		tracing.RecordError(span, err, "failed to get provider by OAuth state")
		return err
	}

	if vp == nil {
		tracing.RecordError(span, nil, "VCS provider not found")
		return errors.New("VCS provider not found", errors.WithErrorCode(errors.ENotFound))
	}

	// Require UpdateVCSProviderPermission since we're updating the provider's values.
	err = caller.RequirePermission(ctx, permissions.UpdateVCSProviderPermission, auth.WithGroupID(vp.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return err
	}

	provider, err := s.getVCSProvider(vp.Type)
	if err != nil {
		tracing.RecordError(span, err, "failed to get VCS provider")
		return err
	}

	redirectURL, err := s.getOAuthCallBackURL(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to get Tharsis OAuth callback URL")
		return fmt.Errorf("failed to get Tharsis OAuth callback URL: %v", err)
	}

	// Create the access token with the provider.
	payload, err := provider.CreateAccessToken(ctx, &types.CreateAccessTokenInput{
		ProviderURL:       vp.URL,
		ClientID:          vp.OAuthClientID,
		ClientSecret:      vp.OAuthClientSecret,
		AuthorizationCode: input.AuthorizationCode,
		RedirectURI:       redirectURL,
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to create access token")
		return err
	}

	// Test the access token incase the value wasn't retrieved for some reason.
	if err = provider.TestConnection(ctx, &types.TestConnectionInput{
		ProviderURL: vp.URL,
		AccessToken: payload.AccessToken,
	}); err != nil {
		tracing.RecordError(span, err, "access token connection test failed")
		return err
	}

	// Update provider's fields.
	vp.OAuthState = nil
	vp.OAuthAccessToken = &payload.AccessToken

	// Not all provider's (e.g. GitHub) support refresh tokens for OAuth apps.
	if payload.RefreshToken != "" {
		vp.OAuthRefreshToken = &payload.RefreshToken
		vp.OAuthAccessTokenExpiresAt = payload.ExpirationTimestamp
	}

	// Update the provider.
	_, err = s.dbClient.VCSProviders.UpdateProvider(ctx, vp)
	if err != nil {
		tracing.RecordError(span, err, "failed to update VCS provider in service layer ProcessOAuth")
		return fmt.Errorf("failed to update VCS provider in service layer ProcessOAuth: %v", err)
	}

	return nil
}

func (s *service) getOAuthCallBackURL(_ context.Context) (string, error) {
	tharsisURL, err := url.Parse(s.tharsisURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse Tharsis URL: %v", err)
	}

	// Add the callback endpoint to the URL.
	tharsisURL.Path = oAuthCallBackEndpoint

	return tharsisURL.String(), nil
}

// refreshOAuthToken renews the access token used to interact with the provider.
// skipUpdate can be set to true when provider isn't to be updated.
func (s *service) refreshOAuthToken(ctx context.Context, provider Provider, vp *models.VCSProvider, skipUpdate bool) (string, error) {
	if vp.OAuthAccessToken == nil {
		// OAuthAccessToken could be nil if OAuth token has been reset, but
		// OAuth flow hasn't been completed yet.
		return "", errors.New(
			"No available access token, please complete OAuth flow first",
			errors.WithErrorCode(errors.EInternal))
	}

	if vp.OAuthRefreshToken == nil {
		// Since no refresh token is available, use the token on the provider.
		return *vp.OAuthAccessToken, nil
	}

	if vp.OAuthAccessTokenExpiresAt != nil && vp.OAuthAccessTokenExpiresAt.After(time.Now().Add(tokenExpirationLeeway)) {
		// Since the access token hasn't expired yet, continue to use it.
		return *vp.OAuthAccessToken, nil
	}

	redirectURI, err := s.getOAuthCallBackURL(ctx)
	if err != nil {
		return "", err
	}

	// Renew the access token.
	payload, err := provider.CreateAccessToken(ctx, &types.CreateAccessTokenInput{
		ProviderURL:  vp.URL,
		ClientID:     vp.OAuthClientID,
		ClientSecret: vp.OAuthClientSecret,
		RedirectURI:  redirectURI,
		RefreshToken: *vp.OAuthRefreshToken, // We're renewing the access token.
	})
	if err != nil {
		return "", err
	}

	// Update fields.
	vp.OAuthAccessToken = &payload.AccessToken
	vp.OAuthRefreshToken = &payload.RefreshToken
	vp.OAuthAccessTokenExpiresAt = payload.ExpirationTimestamp

	// Update provider.
	if !skipUpdate {
		if _, err = s.dbClient.VCSProviders.UpdateProvider(ctx, vp); err != nil {
			return "", err
		}
	}

	return payload.AccessToken, nil
}

func (s *service) getVCSProvider(providerType models.VCSProviderType) (Provider, error) {
	provider, ok := s.vcsProviderMap[providerType]
	if !ok {
		return nil, errors.New("VCS provider with type %s is not supported", providerType, errors.WithErrorCode(errors.EInvalid))
	}

	return provider, nil
}

func (s *service) handleVCSRun(ctx context.Context, input *handleVCSRunInput) error {
	// Download the repository archive and get the path to the local repo.
	parentDirectory, repoDirectory, err := downloadRepositoryArchive(ctx, &downloadRepositoryArchiveInput{
		providerURL:         input.providerURL,
		accessToken:         input.accessToken,
		provider:            input.provider,
		repositoryPath:      input.link.RepositoryPath,
		referenceName:       input.referenceName,
		repositorySizeLimit: s.repositorySizeLimit,
	})
	if err != nil {
		// Remove the temp directory.
		os.RemoveAll(parentDirectory)
		return fmt.Errorf(
			"failed to download repository %s archive for workspace %s and workspace vcs provider link ID %s: %v",
			input.link.RepositoryPath,
			input.workspace.FullPath,
			input.link.Metadata.ID,
			err,
		)
	}

	// Defer removing temporary parent directory.
	defer func() {
		if err = os.RemoveAll(parentDirectory); err != nil {
			s.logger.Errorf(
				"failed to delete temp repository directory for repository %s for workspace %s and workspace vcs provider link ID %s: %v",
				input.link.RepositoryPath,
				input.workspace.FullPath,
				input.link.Metadata.ID,
				err,
			)
		}
	}()

	// Create and upload the configuration version.
	configurationVersionID, err := s.createUploadConfigurationVersion(ctx, &createUploadConfigurationVersionInput{
		vcsEvent:      input.vcsEvent,
		link:          input.link,
		repoDirectory: repoDirectory,
	})
	if err != nil {
		return fmt.Errorf(
			"failed to create and upload configuration version for repository %s for workspace %s and workspace vcs provider link ID %s: %v",
			input.link.RepositoryPath,
			input.workspace.FullPath,
			input.link.Metadata.ID,
			err,
		)
	}

	if _, err = s.runService.CreateRun(ctx, &run.CreateRunInput{
		ConfigurationVersionID: &configurationVersionID,
		WorkspaceID:            input.link.WorkspaceID,
		IsDestroy:              input.isDestroy,
	}); err != nil {
		return fmt.Errorf(
			"failed to create a run for repository %s for workspace %s and workspace vcs provider link ID %s: %v",
			input.link.RepositoryPath,
			input.workspace.FullPath,
			input.link.Metadata.ID,
			err,
		)
	}

	return nil
}

// handleEvent fetches a list of changed files from the provider's
// API and determines if a run is required. Dispatches functions to
// create / upload the configuration version and creates the run.
func (s *service) handleEvent(ctx context.Context, input *handleEventInput) error {
	var (
		alteredFiles map[string]struct{}
		err          error
	)

	// Find changed files if this is not a tag event and glob patterns are being used.
	if !input.vcsEvent.Type.Equals(models.TagEventType) && len(input.link.GlobPatterns) > 0 {
		alteredFiles, err = getAlteredFiles(ctx, input)
		if err != nil {
			s.logger.Errorf(
				"failed to get altered files for repository %s for workspace %s and workspace vcs provider link ID %s: %v",
				input.link.RepositoryPath,
				input.workspace.FullPath,
				input.link.Metadata.ID,
				err,
			)
			// If we can't get the list of changes, we'll create a run anyway.
		}
	}

	referenceName := input.processInput.Ref
	if input.vcsEvent.Type.Equals(models.MergeRequestEventType) {
		referenceName = input.processInput.SourceBranch // Clone the source branch for MRs.
	}

	downloadInput := &downloadRepositoryArchiveInput{
		providerURL:         input.providerURL,
		accessToken:         input.accessToken,
		provider:            input.provider,
		repositoryPath:      input.link.RepositoryPath,
		referenceName:       referenceName,
		repositorySizeLimit: input.repositorySizeLimit,
	}

	// Download the repository archive and get the path to the local repo.
	parentDirectory, repoDirectory, err := downloadRepositoryArchive(ctx, downloadInput)
	if err != nil {
		// Remove the temp directory.
		os.RemoveAll(parentDirectory)
		return fmt.Errorf(
			"failed to download repository %s archive for workspace %s and workspace vcs provider link ID %s: %v",
			input.link.RepositoryPath,
			input.workspace.FullPath,
			input.link.Metadata.ID,
			err,
		)
	}

	// Defer removing temporary parent directory.
	defer func() {
		if err = os.RemoveAll(parentDirectory); err != nil {
			s.logger.Errorf(
				"failed to delete temp repository directory for repository %s for workspace %s and workspace vcs provider link ID %s: %v",
				input.link.RepositoryPath,
				input.workspace.FullPath,
				input.link.Metadata.ID,
				err,
			)
		}
	}()

	// If none of the glob patterns match, no run is required.
	if len(alteredFiles) > 0 && !globsMatch(repoDirectory, alteredFiles, input.link.GlobPatterns) {
		return nil
	}

	// Create and upload the configuration version.
	configurationVersionID, err := s.createUploadConfigurationVersion(ctx, &createUploadConfigurationVersionInput{
		vcsEvent:      input.vcsEvent,
		link:          input.link,
		repoDirectory: repoDirectory,
	})
	if err != nil {
		return fmt.Errorf(
			"failed to create and upload configuration version for repository %s for workspace %s and workspace vcs provider link ID %s: %v",
			input.link.RepositoryPath,
			input.workspace.FullPath,
			input.link.Metadata.ID,
			err,
		)
	}

	if _, err = s.runService.CreateRun(ctx, &run.CreateRunInput{
		ConfigurationVersionID: &configurationVersionID,
		WorkspaceID:            input.link.WorkspaceID,
	}); err != nil {
		return fmt.Errorf(
			"failed to create a run for repository %s for workspace %s and workspace vcs provider link ID %s: %v",
			input.link.RepositoryPath,
			input.workspace.FullPath,
			input.link.Metadata.ID,
			err,
		)
	}

	return nil
}

// createUploadConfigurationVersion creates a configuration version, uploads it
// and waits for the upload to finish. Returns the configuration version ID and
// any errors encountered.
func (s *service) createUploadConfigurationVersion(ctx context.Context,
	input *createUploadConfigurationVersionInput,
) (string, error) {
	// Create the configuration version.
	cv, err := s.workspaceService.CreateConfigurationVersion(ctx, &workspace.CreateConfigurationVersionInput{
		VCSEventID:  &input.vcsEvent.Metadata.ID,
		WorkspaceID: input.link.WorkspaceID,
		Speculative: input.vcsEvent.Type.Equals(models.MergeRequestEventType), // Set to speculative for MRs.
	})
	if err != nil {
		return "", err
	}

	moduleDirectory := ""
	if input.link.ModuleDirectory != nil {
		moduleDirectory = *input.link.ModuleDirectory
	}

	// Create a tar of the Terraform module, if moduleDirectory is not
	// set then the root of the repo directory is used.
	moduleTar, err := makeModuleTar(filepath.Join(input.repoDirectory, moduleDirectory))
	if err != nil {
		return "", err
	}

	// Open a reader on the tar.gz file.
	tarRdr, err := os.Open(moduleTar) // nosemgrep: gosec.G304-1
	if err != nil {
		return "", err
	}
	defer tarRdr.Close()
	defer os.Remove(tarRdr.Name())

	err = s.workspaceService.UploadConfigurationVersion(ctx, cv.Metadata.ID, tarRdr)
	if err != nil {
		return "", err
	}

	// Wait for the upload to complete.
	var updatedConfigurationVersion *models.ConfigurationVersion
	for {
		updatedConfigurationVersion, err = s.workspaceService.GetConfigurationVersion(ctx, cv.Metadata.ID)
		if err != nil {
			return "", fmt.Errorf("failed to check for completion of configuration upload: %s", err)
		}
		if updatedConfigurationVersion.Status != models.ConfigurationPending {
			break
		}

		// Sleep some time before polling again.
		time.Sleep(defaultSleepDuration)
	}

	if updatedConfigurationVersion.Status != models.ConfigurationUploaded {
		return "", fmt.Errorf("configuration upload failed; status is %s", updatedConfigurationVersion.Status)
	}

	return cv.Metadata.ID, nil
}

// downloadRepositoryArchive downloads the repository archive
// and returns the path to the repo's directory.
func downloadRepositoryArchive(ctx context.Context, input *downloadRepositoryArchiveInput) (string, string, error) {
	// Download the repository archive.
	archiveResp, err := input.provider.GetArchive(ctx, &types.GetArchiveInput{
		ProviderURL:    input.providerURL,
		AccessToken:    input.accessToken,
		RepositoryPath: input.repositoryPath,
		Ref:            input.referenceName,
	})
	if err != nil {
		return "", "", err
	}
	defer archiveResp.Body.Close()

	// Create the final destination directory where archive will be unpacked.
	tmpDownloadDir, err := os.MkdirTemp("", "repository")
	if err != nil {
		return "", "", fmt.Errorf("failed to create temp repository directory: %v", err)
	}

	// Create a temporary file to download the archive to.
	destinationFile, err := os.CreateTemp("", "*-repository.tar.gz")
	if err != nil {
		return tmpDownloadDir, "", fmt.Errorf("failed to create temporary file to download repository: %v", err)
	}
	defer os.Remove(destinationFile.Name())

	// Download the repository in chunks.
	if err = copyToDestination(destinationFile, archiveResp.Body, int64(input.repositorySizeLimit)); err != nil {
		return tmpDownloadDir, "", err
	}

	// Rewind file to start
	if _, err = destinationFile.Seek(0, io.SeekStart); err != nil {
		return tmpDownloadDir, "", err
	}

	// Decompress the tar file.
	err = tgz.Decompress(tmpDownloadDir, destinationFile.Name(), true, 0o000)
	if err != nil {
		return tmpDownloadDir, "", err
	}

	// Get a list of all files in the directory. When decompressing,
	// the actual repo contents are in a child directory, the name
	// of which can differ from provider to provider.
	files, err := os.ReadDir(tmpDownloadDir)
	if err != nil {
		return tmpDownloadDir, "", err
	}

	if len(files) == 0 {
		return tmpDownloadDir, "", fmt.Errorf("failed to decompress repository tarball")
	}

	// Repository directory is the child of tmpDownloadDir.
	return tmpDownloadDir, filepath.Join(tmpDownloadDir, files[0].Name()), nil
}

// getAlteredFiles returns a list of directories / files that
// have been altered by running a diff on the 'before' and
// after commit IDs. For cases, such as a first commit
// in a branch where the 'before' commit may be empty, it
// simply retrieves the files from the most-recent commit ID.
// For merge requests, it uses the head commit ID.
func getAlteredFiles(ctx context.Context, input *handleEventInput) (map[string]struct{}, error) {
	var alteredFiles map[string]struct{}

	if !plumbing.NewHash(input.processInput.Before).IsZero() {
		// Since the 'before' commit is not empty, we can
		// run a diff on 'before' and 'after' commits.
		payload, err := input.provider.GetDiffs(ctx, &types.GetDiffsInput{
			ProviderURL:    input.providerURL,
			AccessToken:    input.accessToken,
			RepositoryPath: input.link.RepositoryPath,
			BaseRef:        input.processInput.Before,
			HeadRef:        input.processInput.After,
		})
		if err != nil {
			return nil, err
		}

		alteredFiles = payload.AlteredFiles
	} else {
		// Use the after or head commit for a branch unless this is
		// an MR event, for that we can use the latest commit of the MR.
		ref := input.processInput.After
		if input.vcsEvent.Type.Equals(models.MergeRequestEventType) {
			ref = input.processInput.HeadCommitID
		}

		// No parent or 'before' hash i.e. first branch commit.
		// Get the diff for the 'head' commit ID.
		payload, err := input.provider.GetDiff(ctx, &types.GetDiffInput{
			ProviderURL:    input.providerURL,
			AccessToken:    input.accessToken,
			RepositoryPath: input.link.RepositoryPath,
			Ref:            ref,
		})
		if err != nil {
			return nil, err
		}

		alteredFiles = payload.AlteredFiles
	}

	return alteredFiles, nil
}

// globsMatch determines if the files that changed match
// the glob patterns. Returns true on the earliest match.
// Multiple patterns act as an OR condition.
func globsMatch(repoDirectory string, alteredFiles map[string]struct{}, globs []string) bool {
	// Must add a trailing slash, so globs without a leading slash work properly.
	// FilepathGlob will properly clean the path before using it incase of double slashes.
	repoDirectory += "/"
	for _, glob := range globs {
		// Only possible error returned is when pattern is malformed.
		// Since this was validated when created, we can ignore it.
		matches, _ := doublestar.FilepathGlob(repoDirectory + glob)

		for _, match := range matches {
			// Remove the directory name and a trailing '/' prefix as
			// filepaths in alteredFiles won't have it.
			if _, ok := alteredFiles[strings.TrimPrefix(match, repoDirectory)]; ok {
				return ok
			}
		}
	}

	return false
}

// refMatches performs some preliminary checks to make sure
// the branch or tag events match what's defined on the
// provider link.
func refMatches(
	input *ProcessWebhookEventInput,
	eventType models.VCSEventType,
	link *models.WorkspaceVCSProviderLink,
	provider Provider,
) bool {
	// Trim the prefix before pattern matching. Necessary
	// incase the pattern supplied contains '^' or '$'.
	ref := trimRefPrefix(input.Ref)

	// Tag event.
	if eventType.Equals(models.TagEventType) {
		if link.TagRegex == nil {
			// Since there isn't a regex we could match the tag
			// to, no run will be created.
			return false
		}

		if *link.TagRegex != "" {
			// Regex has already been validated at the time of creation.
			tagRegex, _ := regexp.Compile(*link.TagRegex)
			return tagRegex.MatchString(ref)
		}
	}

	// Merge request event.
	if eventType.Equals(models.MergeRequestEventType) {
		// Allow runs only if PR is not from a fork,
		// MR action is supported, auto speculative plan is enabled
		// on the link and it's for the link's configured branch.
		return input.SourceRepository == link.RepositoryPath &&
			provider.MergeRequestActionIsSupported(input.Action) &&
			link.AutoSpeculativePlan &&
			input.TargetBranch == link.Branch
	}

	// Branch event.
	return eventType.Equals(models.BranchEventType) && ref == link.Branch
}

// makeModuleTar creates a tar of the location specified by the module path.
func makeModuleTar(modulePath string) (string, error) {
	// Create the temporary tar.gz file.
	tarFile, err := os.CreateTemp("", "*-uploadCV.tgz")
	if err != nil {
		return "", err
	}
	tarPath := tarFile.Name()

	// Open a writer to the temporary tar.gz file.
	tgzFileWriter, err := os.OpenFile(tarPath, tarFlagWrite, tarMode) // nosemgrep: gosec.G304-1
	if err != nil {
		return "", err
	}
	defer tgzFileWriter.Close()

	_, err = slug.Pack(modulePath, tgzFileWriter, false)
	if err != nil {
		return "", err
	}

	return tarPath, err
}

// copyToDestination copies from source to destination in chunks.
// Returns an error if bytes received exceed repositorySizeLimit.
func copyToDestination(destinationFile *os.File, sourceFile io.ReadCloser, repositorySizeLimit int64) error {
	var totalWrittenBytes int64

	for {
		writtenBytes, err := io.CopyN(destinationFile, sourceFile, 1024)
		if err != nil {
			if err == io.EOF {
				// We've reached the end of the file i.e. download complete.
				break
			}
			return err
		}

		totalWrittenBytes += writtenBytes

		// Make sure downloaded amount doesn't exceed repositorySizeLimit.
		if totalWrittenBytes > repositorySizeLimit {
			return fmt.Errorf(
				"download size %d bytes exceeds the maximum configured size limit of %d bytes",
				totalWrittenBytes,
				repositorySizeLimit,
			)
		}
	}

	return nil
}

// trimRefPrefix removes any ref prefix.
func trimRefPrefix(ref string) string {
	for _, prefix := range refPrefixes {
		ref = strings.TrimPrefix(ref, prefix)
	}

	return ref
}

// getTharsisWebhookURL returns the Tharsis webhook URL with an optional
// token as a query parameter (used for GitHub).
func getTharsisWebhookURL(tharsisURL string, token []byte) (string, error) {
	endpoint, err := url.Parse(tharsisURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse Tharsis URL: %v", err)
	}
	endpoint.Path = types.V1WebhookEndpoint

	// Add the token if present.
	if token != nil {
		queries := endpoint.Query()
		queries.Set("token", string(token))
		endpoint.RawQuery = queries.Encode()
	}

	return endpoint.String(), nil
}
