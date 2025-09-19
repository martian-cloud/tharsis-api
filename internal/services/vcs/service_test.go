package vcs

import (
	context "context"
	"io"
	http "net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	mock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/asynctask"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/limits"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/maintenance"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	mtypes "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run"
	types "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/vcs/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/workspace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

const (
	// Used for ResourceMetadata.ID by several test resources.
	resourceUUID = "0e8408de-a2ff-4194-8481-d0bbb0874037"

	tharsisURL          = "https://tharsis.domain"
	vcsOAuthCallbackURL = "https://tharsis.domain/v1/vcs/auth/callback"
)

var (
	sampleProviderURL = url.URL{Scheme: "http", Host: "example.com:8080", Path: "/some/instance"}
	sampleTagRegex    = "\\d+.\\d+$"

	groupPath = "a/resource"

	// Random commit IDs for testing processingWebhook logic.
	sampleBeforeCommit = "64b317c5bcfc637cca23b25f38501571f2a02b21"
	sampleAfterCommit  = "64b317c5bcfc637cca80b25f38501571f2a02b21"

	// OAuth token fields for a vcs provider.
	sampleOAuthAccessToken                    = "an-access-token"
	sampleOAuthRefreshToken                   = "a-refresh-token"
	sampleOAuthAccessTokenExpirationTimestamp = time.Now()
)

func TestGetVCSProviderByID(t *testing.T) {
	sampleProvider := &models.VCSProvider{
		Metadata: models.ResourceMetadata{
			ID: resourceUUID,
		},
		Name: "expected-name",
	}

	testCases := []struct {
		caller            auth.Caller
		expectedProvider  *models.VCSProvider
		name              string
		inputID           string
		expectedErrorCode errors.CodeType
	}{
		{
			name:             "positive: with caller; expect a vcs provider",
			inputID:          resourceUUID,
			caller:           &auth.SystemCaller{},
			expectedProvider: sampleProvider,
		},
		{
			name:              "negative: with caller, no such provider; expect error ENotFound",
			inputID:           resourceUUID,
			caller:            &auth.SystemCaller{},
			expectedErrorCode: errors.ENotFound,
		},
		{
			name:              "negative: without caller; expect error EUnauthorized",
			inputID:           resourceUUID,
			expectedErrorCode: errors.EUnauthorized,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := auth.WithCaller(context.Background(), test.caller)

			mockVCSProviders := db.MockVCSProviders{}
			mockVCSProviders.Test(t)

			// VCSProvider mocks.
			mockVCSProviders.On("GetProviderByID", mock.Anything, test.inputID).Return(test.expectedProvider, nil)

			dbClient := &db.Client{
				VCSProviders: &mockVCSProviders,
			}

			service := newService(nil, dbClient, nil, nil, nil, nil, nil, nil, nil, nil, "", 0)

			provider, err := service.GetVCSProviderByID(ctx, test.inputID)
			if test.expectedErrorCode != "" {
				assert.Equal(t, test.expectedErrorCode, errors.ErrorCode(err))
			} else if err != nil {
				t.Fatal(err)
			} else {
				assert.Equal(t, test.expectedProvider, provider)
			}
		})
	}
}

func TestGetVCSProviderByTRN(t *testing.T) {
	sampleProvider := &models.VCSProvider{
		Metadata: models.ResourceMetadata{
			ID:  "provider-1",
			TRN: mtypes.VCSProviderModelType.BuildTRN("provider-gid-1"),
		},
		Name:               "test-provider",
		OAuthClientID:      "client-id",
		OAuthClientSecret:  "client-secret",
		Type:               models.GitHubProviderType,
		URL:                sampleProviderURL,
		AutoCreateWebhooks: true,
	}

	type testCase struct {
		name            string
		authError       error
		provider        *models.VCSProvider
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:     "successfully get provider by trn",
			provider: sampleProvider,
		},
		{
			name:            "provider not found",
			expectErrorCode: errors.ENotFound,
		},
		{
			name: "subject is not authorized to view provider",
			provider: &models.VCSProvider{
				Metadata:          sampleProvider.Metadata,
				Name:              sampleProvider.Name,
				OAuthClientID:     sampleProvider.OAuthClientID,
				OAuthClientSecret: sampleProvider.OAuthClientSecret,
			},
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockCaller := auth.NewMockCaller(t)
			mockVCSProviders := db.NewMockVCSProviders(t)

			mockVCSProviders.On("GetProviderByTRN", mock.Anything, sampleProvider.Metadata.TRN).Return(test.provider, nil)

			if test.provider != nil {
				mockCaller.On("RequirePermission", mock.Anything, models.ViewVCSProviderPermission, mock.Anything).Return(test.authError)
			}

			dbClient := &db.Client{
				VCSProviders: mockVCSProviders,
			}

			service := &service{
				dbClient: dbClient,
			}

			actualProvider, err := service.GetVCSProviderByTRN(auth.WithCaller(ctx, mockCaller), sampleProvider.Metadata.TRN)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.provider, actualProvider)
		})
	}
}

func TestGetVCSProviders(t *testing.T) {
	sampleProvider := models.VCSProvider{
		Metadata: models.ResourceMetadata{
			ID: resourceUUID,
		},
		Name: "sample-vcs-provider",
	}

	// a sample DB result object.
	sampleResult := &db.VCSProvidersResult{
		PageInfo: &pagination.PageInfo{
			TotalCount:      1,
			HasNextPage:     false,
			HasPreviousPage: false,
		},
		VCSProviders: []models.VCSProvider{sampleProvider},
	}

	testCases := []struct {
		name              string
		input             *GetVCSProvidersInput
		dbInput           *db.GetVCSProvidersInput
		expectedResult    *db.VCSProvidersResult
		caller            auth.Caller
		expectedErrorCode errors.CodeType
	}{
		{
			name:  "positive: nearly empty input and with caller; expect result object",
			input: &GetVCSProvidersInput{},
			dbInput: &db.GetVCSProvidersInput{
				Filter: &db.VCSProviderFilter{
					NamespacePaths: []string{""},
				},
			},
			caller:         &auth.SystemCaller{},
			expectedResult: sampleResult,
		},
		{
			name: "negative: search for provider that doesn't exist; expect empty result object",
			input: &GetVCSProvidersInput{
				Search: ptr.String("non-existent-provider"),
			},
			dbInput: &db.GetVCSProvidersInput{
				Filter: &db.VCSProviderFilter{
					Search:         ptr.String("non-existent-provider"),
					NamespacePaths: []string{""},
				},
			},
			caller:         &auth.SystemCaller{},
			expectedResult: &db.VCSProvidersResult{},
		},
		{
			name:              "negative: without caller; expect error EUnauthorized",
			expectedErrorCode: errors.EUnauthorized,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := auth.WithCaller(context.Background(), test.caller)

			mockVCSProviders := db.MockVCSProviders{}
			mockVCSProviders.Test(t)

			// VCSProvider mocks.
			mockVCSProviders.On("GetProviders", mock.Anything, test.dbInput).Return(test.expectedResult, nil)

			dbClient := &db.Client{
				VCSProviders: &mockVCSProviders,
			}

			service := newService(nil, dbClient, nil, nil, nil, nil, nil, nil, nil, nil, "", 0)

			result, err := service.GetVCSProviders(ctx, test.input)
			if test.expectedErrorCode != "" {
				assert.Equal(t, test.expectedErrorCode, errors.ErrorCode(err))
			} else if err != nil {
				t.Fatal(err)
			} else if test.expectedResult != nil {
				assert.NotNil(t, test.expectedResult, result)
				assert.Equal(t, test.expectedResult, result)
			}
		})
	}
}

func TestGetVCSProvidersByIDs(t *testing.T) {
	sampleProvider := models.VCSProvider{
		Metadata: models.ResourceMetadata{
			ID:  resourceUUID,
			TRN: mtypes.VCSProviderModelType.BuildTRN("some-group/expected-name"),
		},
		Name: "expected-name",
	}

	// a sample DB result object.
	sampleResult := &db.VCSProvidersResult{
		PageInfo: &pagination.PageInfo{
			TotalCount:      1,
			HasNextPage:     false,
			HasPreviousPage: false,
		},
		VCSProviders: []models.VCSProvider{sampleProvider},
	}

	testCases := []struct {
		caller               auth.Caller
		dbInput              *db.GetVCSProvidersInput
		name                 string
		expectedErrorCode    errors.CodeType
		expectedProviderList []models.VCSProvider
		inputIDList          []string
	}{
		{
			name:        "positive: with caller; expect a vcs provider list",
			inputIDList: []string{resourceUUID},
			dbInput: &db.GetVCSProvidersInput{
				Filter: &db.VCSProviderFilter{
					VCSProviderIDs: []string{resourceUUID},
				},
			},
			caller:               &auth.SystemCaller{},
			expectedProviderList: []models.VCSProvider{sampleProvider},
		},
		{
			name:              "negative: without caller; expect error EUnauthorized",
			expectedErrorCode: errors.EUnauthorized,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := auth.WithCaller(context.Background(), test.caller)

			mockVCSProviders := db.MockVCSProviders{}
			mockVCSProviders.Test(t)

			// VCSProvider mocks.
			mockVCSProviders.On("GetProviders", mock.Anything, test.dbInput).Return(sampleResult, nil)

			dbClient := &db.Client{
				VCSProviders: &mockVCSProviders,
			}

			service := newService(nil, dbClient, nil, nil, nil, nil, nil, nil, nil, nil, "", 0)

			providerList, err := service.GetVCSProvidersByIDs(ctx, test.inputIDList)
			if test.expectedErrorCode != "" {
				assert.Equal(t, test.expectedErrorCode, errors.ErrorCode(err))
			} else if err != nil {
				t.Fatal(err)
			} else {
				assert.Equal(t, test.expectedProviderList, providerList)
			}
		})
	}
}

func TestGetVCSEventByID(t *testing.T) {
	sampleEvent := &models.VCSEvent{
		Metadata: models.ResourceMetadata{
			ID:  "event-1",
			TRN: mtypes.VCSEventModelType.BuildTRN("event-gid-1"),
		},
		WorkspaceID: "workspace-1",
	}

	type testCase struct {
		name            string
		authError       error
		event           *models.VCSEvent
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:  "successfully get event by id",
			event: sampleEvent,
		},
		{
			name:            "event not found",
			expectErrorCode: errors.ENotFound,
		},
		{
			name: "subject is not authorized to view event",
			event: &models.VCSEvent{
				Metadata:    sampleEvent.Metadata,
				WorkspaceID: sampleEvent.WorkspaceID,
				Type:        sampleEvent.Type,
			},
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockCaller := auth.NewMockCaller(t)
			mockVCSEvents := db.NewMockVCSEvents(t)

			mockVCSEvents.On("GetEventByID", mock.Anything, sampleEvent.Metadata.ID).Return(test.event, nil)

			if test.event != nil {
				mockCaller.On("RequirePermission", mock.Anything, models.ViewVCSProviderPermission, mock.Anything).Return(test.authError)
			}

			dbClient := &db.Client{
				VCSEvents: mockVCSEvents,
			}

			service := &service{
				dbClient: dbClient,
			}

			actualEvent, err := service.GetVCSEventByID(auth.WithCaller(ctx, mockCaller), sampleEvent.Metadata.ID)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.event, actualEvent)
		})
	}
}

func TestGetVCSEventByTRN(t *testing.T) {
	sampleEvent := &models.VCSEvent{
		Metadata: models.ResourceMetadata{
			ID:  "event-1",
			TRN: mtypes.VCSEventModelType.BuildTRN("event-gid-1"),
		},
		WorkspaceID: "workspace-1",
	}

	type testCase struct {
		name            string
		authError       error
		event           *models.VCSEvent
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:  "successfully get event by trn",
			event: sampleEvent,
		},
		{
			name:            "event not found",
			expectErrorCode: errors.ENotFound,
		},
		{
			name: "subject is not authorized to view event",
			event: &models.VCSEvent{
				Metadata:    sampleEvent.Metadata,
				WorkspaceID: sampleEvent.WorkspaceID,
				Type:        sampleEvent.Type,
			},
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockCaller := auth.NewMockCaller(t)
			mockVCSEvents := db.NewMockVCSEvents(t)

			mockVCSEvents.On("GetEventByTRN", mock.Anything, sampleEvent.Metadata.TRN).Return(test.event, nil)

			if test.event != nil {
				mockCaller.On("RequirePermission", mock.Anything, models.ViewVCSProviderPermission, mock.Anything).Return(test.authError)
			}

			dbClient := &db.Client{
				VCSEvents: mockVCSEvents,
			}

			service := &service{
				dbClient: dbClient,
			}

			actualEvent, err := service.GetVCSEventByTRN(auth.WithCaller(ctx, mockCaller), sampleEvent.Metadata.TRN)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.event, actualEvent)
		})
	}
}

func TestCreateVCSProvider(t *testing.T) {
	sampleOAuthState, err := uuid.NewRandom()
	assert.Nil(t, err)

	testCases := []struct {
		caller                auth.Caller
		input                 *CreateVCSProviderInput
		toCreate              *models.VCSProvider
		buildAuthCodeURLInput *types.BuildOAuthAuthorizationURLInput
		activityInput         *activityevent.CreateActivityEventInput
		expectedProvider      *models.VCSProvider
		name                  string
		expectedErrorCode     errors.CodeType
		limit                 int
		injectProviders       int32
		exceedsLimit          bool
	}{
		{
			name:   "positive: GitLab provider with URL, manual; expect provider created with values",
			caller: &auth.SystemCaller{},
			input: &CreateVCSProviderInput{
				Name:               "a-sample-gitlab-provider",
				Description:        "",
				GroupID:            "group-id",
				OAuthClientID:      "a-sample-client-id",
				OAuthClientSecret:  "a-sample-client-secret",
				Type:               models.GitLabProviderType,
				URL:                ptr.String(sampleProviderURL.String()),
				AutoCreateWebhooks: false,
			},
			buildAuthCodeURLInput: &types.BuildOAuthAuthorizationURLInput{
				ProviderURL:        sampleProviderURL,
				OAuthClientID:      "a-sample-client-id",
				OAuthState:         sampleOAuthState.String(),
				RedirectURL:        oAuthCallBackEndpoint,
				UseReadWriteScopes: false,
			},
			toCreate: &models.VCSProvider{
				Name:               "a-sample-gitlab-provider",
				Description:        "",
				GroupID:            "group-id",
				OAuthClientID:      "a-sample-client-id",
				OAuthClientSecret:  "a-sample-client-secret",
				OAuthState:         ptr.String(sampleOAuthState.String()),
				Type:               models.GitLabProviderType,
				URL:                sampleProviderURL,
				CreatedBy:          "system",
				AutoCreateWebhooks: false,
			},
			activityInput: &activityevent.CreateActivityEventInput{
				NamespacePath: ptr.String("a/resource"),
				Action:        models.ActionCreate,
				TargetType:    models.TargetVCSProvider,
				TargetID:      resourceUUID,
			},
			expectedProvider: &models.VCSProvider{
				Metadata: models.ResourceMetadata{
					ID:  resourceUUID,
					TRN: mtypes.VCSProviderModelType.BuildTRN("a/resource/path"),
				},
				Name:               "a-sample-gitlab-provider",
				Description:        "",
				GroupID:            "group-id",
				OAuthClientID:      "a-sample-client-id",
				OAuthClientSecret:  "a-sample-client-secret",
				OAuthState:         ptr.String(sampleOAuthState.String()),
				Type:               models.GitLabProviderType,
				URL:                sampleProviderURL,
				CreatedBy:          "sample@sample-email",
				AutoCreateWebhooks: false,
			},
			limit:           5,
			injectProviders: 5,
		},
		{
			name:   "positive: GitHub provider, no URL, manual; expect provider created with defaults",
			caller: &auth.SystemCaller{},
			input: &CreateVCSProviderInput{
				Name:               "a-sample-github-provider",
				Description:        "",
				GroupID:            "group-id",
				OAuthClientID:      "a-sample-client-id",
				OAuthClientSecret:  "a-sample-client-secret",
				Type:               models.GitHubProviderType,
				AutoCreateWebhooks: false,
			},
			toCreate: &models.VCSProvider{
				Name:               "a-sample-github-provider",
				Description:        "",
				GroupID:            "group-id",
				OAuthClientID:      "a-sample-client-id",
				OAuthClientSecret:  "a-sample-client-secret",
				OAuthState:         ptr.String(sampleOAuthState.String()),
				URL:                sampleProviderURL,
				CreatedBy:          "system",
				Type:               models.GitHubProviderType,
				AutoCreateWebhooks: false,
			},
			activityInput: &activityevent.CreateActivityEventInput{
				NamespacePath: ptr.String("a/resource"),
				Action:        models.ActionCreate,
				TargetType:    models.TargetVCSProvider,
				TargetID:      resourceUUID,
			},
			buildAuthCodeURLInput: &types.BuildOAuthAuthorizationURLInput{
				ProviderURL:        sampleProviderURL,
				OAuthClientID:      "a-sample-client-id",
				OAuthState:         sampleOAuthState.String(),
				RedirectURL:        oAuthCallBackEndpoint,
				UseReadWriteScopes: false,
			},
			expectedProvider: &models.VCSProvider{
				Metadata: models.ResourceMetadata{
					ID:  resourceUUID,
					TRN: mtypes.VCSProviderModelType.BuildTRN("a/resource/path"),
				},
				Name:               "a-sample-github-provider",
				Description:        "",
				GroupID:            "group-id",
				OAuthClientID:      "a-sample-client-id",
				OAuthClientSecret:  "a-sample-client-secret",
				OAuthState:         ptr.String(sampleOAuthState.String()),
				Type:               models.GitHubProviderType,
				URL:                sampleProviderURL,
				CreatedBy:          "system",
				AutoCreateWebhooks: false,
			},
			limit:           5,
			injectProviders: 5,
		},
		{
			name:   "negative: unsupported provider type; expect error EInvalid",
			caller: &auth.SystemCaller{},
			input: &CreateVCSProviderInput{
				Name:               "an-unsupported-provider",
				Description:        "",
				GroupID:            "group-id",
				OAuthClientID:      "a-sample-client-id",
				OAuthClientSecret:  "a-sample-client-secret",
				Type:               "unsupported",
				AutoCreateWebhooks: true,
			},
			expectedErrorCode: errors.EInvalid,
		},
		{
			name:   "negative: not a valid URL; expect error EInvalid",
			caller: &auth.SystemCaller{},
			input: &CreateVCSProviderInput{
				Type: models.GitHubProviderType,
				URL:  ptr.String("not-valid"),
			},
			expectedErrorCode: errors.EInvalid,
		},
		{
			name:              "negative: without caller; expect error EUnauthorized",
			input:             &CreateVCSProviderInput{},
			expectedErrorCode: errors.EUnauthorized,
		},
		{
			name:   "negative - exceeds limit",
			caller: &auth.SystemCaller{},
			input: &CreateVCSProviderInput{
				Name:               "a-sample-gitlab-provider",
				Description:        "",
				GroupID:            "group-id",
				OAuthClientID:      "a-sample-client-id",
				OAuthClientSecret:  "a-sample-client-secret",
				Type:               models.GitLabProviderType,
				URL:                ptr.String(sampleProviderURL.String()),
				AutoCreateWebhooks: false,
			},
			buildAuthCodeURLInput: &types.BuildOAuthAuthorizationURLInput{
				ProviderURL:        sampleProviderURL,
				OAuthClientID:      "a-sample-client-id",
				OAuthState:         sampleOAuthState.String(),
				RedirectURL:        oAuthCallBackEndpoint,
				UseReadWriteScopes: false,
			},
			toCreate: &models.VCSProvider{
				Name:               "a-sample-gitlab-provider",
				Description:        "",
				GroupID:            "group-id",
				OAuthClientID:      "a-sample-client-id",
				OAuthClientSecret:  "a-sample-client-secret",
				OAuthState:         ptr.String(sampleOAuthState.String()),
				Type:               models.GitLabProviderType,
				URL:                sampleProviderURL,
				CreatedBy:          "system",
				AutoCreateWebhooks: false,
			},
			activityInput: &activityevent.CreateActivityEventInput{
				NamespacePath: ptr.String("a/resource"),
				Action:        models.ActionCreate,
				TargetType:    models.TargetVCSProvider,
				TargetID:      resourceUUID,
			},
			expectedProvider: &models.VCSProvider{
				Metadata: models.ResourceMetadata{
					ID:  resourceUUID,
					TRN: mtypes.VCSProviderModelType.BuildTRN("a/resource/path"),
				},
				Name:               "a-sample-gitlab-provider",
				Description:        "",
				GroupID:            "group-id",
				OAuthClientID:      "a-sample-client-id",
				OAuthClientSecret:  "a-sample-client-secret",
				OAuthState:         ptr.String(sampleOAuthState.String()),
				Type:               models.GitLabProviderType,
				URL:                sampleProviderURL,
				CreatedBy:          "sample@sample-email",
				AutoCreateWebhooks: false,
			},
			limit:             5,
			injectProviders:   6,
			exceedsLimit:      true,
			expectedErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := auth.WithCaller(context.Background(), test.caller)

			mockProviders := MockProvider{}
			mockVCSProviders := db.MockVCSProviders{}
			mockTransactions := db.MockTransactions{}
			mockActivityEventService := activityevent.MockService{}

			mockProviders.Test(t)
			mockVCSProviders.Test(t)
			mockTransactions.Test(t)
			mockActivityEventService.Test(t)
			mockResourceLimits := db.NewMockResourceLimits(t)

			mockProviders.On("DefaultURL").Return(sampleProviderURL)
			mockProviders.On("BuildOAuthAuthorizationURL", mock.Anything).Return("https://redirect-url", nil)

			mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
			mockTransactions.On("RollbackTx", mock.Anything).Return(nil)

			if (test.expectedErrorCode == "") || test.exceedsLimit {
				mockVCSProviders.On("CreateProvider", mock.Anything, test.toCreate).Return(test.expectedProvider, nil)

				mockTransactions.On("CommitTx", mock.Anything).Return(nil)

				mockActivityEventService.On("CreateActivityEvent", mock.Anything, test.activityInput).Return(nil, nil)
			}

			// Called inside transaction to check resource limits.
			if test.limit > 0 {
				mockVCSProviders.On("GetProviders", mock.Anything, mock.Anything).Return(&db.GetVCSProvidersInput{
					Filter: &db.VCSProviderFilter{
						// empty
					},
					PaginationOptions: &pagination.Options{
						First: ptr.Int32(0),
					},
				}).Return(func(ctx context.Context, input *db.GetVCSProvidersInput) *db.VCSProvidersResult {
					_ = ctx
					_ = input

					return &db.VCSProvidersResult{
						PageInfo: &pagination.PageInfo{
							TotalCount: test.injectProviders,
						},
					}
				}, nil)

				mockResourceLimits.On("GetResourceLimit", mock.Anything, mock.Anything).
					Return(&models.ResourceLimit{Value: test.limit}, nil)
			}

			dbClient := &db.Client{
				VCSProviders:   &mockVCSProviders,
				Transactions:   &mockTransactions,
				ResourceLimits: mockResourceLimits,
			}

			logger, _ := logger.NewForTest()

			providerMap := map[models.VCSProviderType]Provider{
				models.GitLabProviderType: &mockProviders,
				models.GitHubProviderType: &mockProviders,
			}

			// Override state generator.
			stateGeneratorFunc := func() (uuid.UUID, error) {
				return sampleOAuthState, nil
			}

			service := newService(logger, dbClient, limits.NewLimitChecker(dbClient), nil, providerMap, &mockActivityEventService, nil, nil, nil, stateGeneratorFunc, "", 0)

			response, err := service.CreateVCSProvider(ctx, test.input)
			if test.expectedErrorCode != "" {
				assert.Equal(t, test.expectedErrorCode, errors.ErrorCode(err))
			} else if err != nil {
				t.Fatal(err)
			} else {
				assert.NotNil(t, response)
				assert.Equal(t, test.expectedProvider, response.VCSProvider)
			}
		})
	}
}

func TestUpdateVCSProvider(t *testing.T) {
	testCases := []struct {
		caller            auth.Caller
		input             *UpdateVCSProviderInput
		activityInput     *activityevent.CreateActivityEventInput
		name              string
		expectedErrorCode errors.CodeType
	}{
		{
			name:   "positive: update description; expect updated provider",
			caller: &auth.SystemCaller{},
			input: &UpdateVCSProviderInput{
				&models.VCSProvider{
					Metadata: models.ResourceMetadata{
						ID:  resourceUUID,
						TRN: mtypes.VCSProviderModelType.BuildTRN("a/resource/path"),
					},
					Name:               "a-sample-github-provider",
					Description:        "this-is-the-new-description",
					GroupID:            "group-id",
					OAuthClientID:      "a-sample-client-id",
					OAuthClientSecret:  "a-sample-client-secret",
					OAuthState:         ptr.String("sample-state"),
					Type:               models.GitHubProviderType,
					URL:                sampleProviderURL,
					CreatedBy:          "sample@sample-email",
					AutoCreateWebhooks: true,
				},
			},
			activityInput: &activityevent.CreateActivityEventInput{
				NamespacePath: ptr.String("a/resource"),
				Action:        models.ActionUpdate,
				TargetType:    models.TargetVCSProvider,
				TargetID:      resourceUUID,
			},
		},
		{
			name:              "negative: without caller; expect error EUnauthorized",
			input:             &UpdateVCSProviderInput{},
			expectedErrorCode: errors.EUnauthorized,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := auth.WithCaller(context.Background(), test.caller)

			mockVCSProviders := db.MockVCSProviders{}
			mockTransactions := db.MockTransactions{}
			mockActivityEventService := activityevent.MockService{}

			mockVCSProviders.Test(t)
			mockTransactions.Test(t)
			mockActivityEventService.Test(t)

			mockVCSProviders.On("UpdateProvider", mock.Anything, test.input.Provider).Return(test.input.Provider, nil)

			mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
			mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
			mockTransactions.On("CommitTx", mock.Anything).Return(nil)

			mockActivityEventService.On("CreateActivityEvent", mock.Anything, test.activityInput).Return(nil, nil)

			dbClient := &db.Client{
				VCSProviders: &mockVCSProviders,
				Transactions: &mockTransactions,
			}

			logger, _ := logger.NewForTest()

			service := newService(logger, dbClient, nil, nil, nil, &mockActivityEventService, nil, nil, nil, nil, "", 0)

			provider, err := service.UpdateVCSProvider(ctx, test.input)
			if test.expectedErrorCode != "" {
				assert.Equal(t, test.expectedErrorCode, errors.ErrorCode(err))
			} else if err != nil {
				t.Fatal(err)
			} else {
				assert.NotNil(t, provider)
				assert.Equal(t, test.input.Provider, provider)
			}
		})
	}
}

func TestDeleteVCSProvider(t *testing.T) {
	sampleOAuthState, err := uuid.NewRandom()
	assert.Nil(t, err)

	sampleAutomaticProvider := &models.VCSProvider{
		Name: "some-provider",
		Metadata: models.ResourceMetadata{
			ID:      resourceUUID,
			Version: 1,
			TRN:     mtypes.VCSProviderModelType.BuildTRN(groupPath + "/some-provider"),
		},
		URL:                sampleProviderURL,
		OAuthClientID:      "a-sample-client-id",
		OAuthClientSecret:  "a-sample-client-secret",
		OAuthState:         ptr.String(sampleOAuthState.String()),
		OAuthAccessToken:   &sampleOAuthAccessToken, // GitHub only supports AccessToken.
		Type:               models.GitHubProviderType,
		GroupID:            "group-id",
		AutoCreateWebhooks: true, // Automatically configured.
	}

	sampleManualProvider := &models.VCSProvider{
		Name: "some-provider",
		Metadata: models.ResourceMetadata{
			ID:      resourceUUID,
			Version: 1,
			TRN:     mtypes.VCSProviderModelType.BuildTRN(groupPath + "/some-provider"),
		},
		URL:                sampleProviderURL,
		OAuthClientID:      "a-sample-client-id",
		OAuthClientSecret:  "a-sample-client-secret",
		OAuthState:         ptr.String(sampleOAuthState.String()),
		OAuthAccessToken:   &sampleOAuthAccessToken,
		Type:               models.GitHubProviderType,
		GroupID:            "group-id",
		AutoCreateWebhooks: false, // Manually configured.
	}

	testCases := []struct {
		caller             auth.Caller
		input              *DeleteVCSProviderInput
		activityInput      *activityevent.CreateActivityEventInput
		deleteWebhookInput *types.DeleteWebhookInput
		name               string
		expectedErrorCode  errors.CodeType
		links              []models.WorkspaceVCSProviderLink
	}{
		{
			name:   "positive: provider is not linked to any workspaces; expect no errors",
			caller: &auth.SystemCaller{},
			input: &DeleteVCSProviderInput{
				Provider: sampleAutomaticProvider,
			},
			links: []models.WorkspaceVCSProviderLink{},
			activityInput: &activityevent.CreateActivityEventInput{
				NamespacePath: &groupPath,
				Action:        models.ActionDeleteChildResource,
				TargetType:    models.TargetGroup,
				TargetID:      "group-id",
				Payload: &models.ActivityEventDeleteChildResourcePayload{
					Name: sampleAutomaticProvider.Name,
					ID:   sampleAutomaticProvider.Metadata.ID,
					Type: string(models.TargetVCSProvider),
				},
			},
		},
		{
			name:   "positive: provider is linked to workspaces(s), automatically configured, force option is used; expect no errors",
			caller: &auth.SystemCaller{},
			input: &DeleteVCSProviderInput{
				Provider: sampleAutomaticProvider,
				Force:    true,
			},
			deleteWebhookInput: &types.DeleteWebhookInput{
				ProviderURL:    sampleProviderURL,
				AccessToken:    "an-access-token",
				RepositoryPath: "owner/repository",
				WebhookID:      "webhook-id",
			},
			links: []models.WorkspaceVCSProviderLink{
				{
					RepositoryPath: "owner/repository",
					WebhookID:      "webhook-id",
				},
			},
			activityInput: &activityevent.CreateActivityEventInput{
				NamespacePath: &groupPath,
				Action:        models.ActionDeleteChildResource,
				TargetType:    models.TargetGroup,
				TargetID:      "group-id",
				Payload: &models.ActivityEventDeleteChildResourcePayload{
					Name: sampleAutomaticProvider.Name,
					ID:   sampleAutomaticProvider.Metadata.ID,
					Type: string(models.TargetVCSProvider),
				},
			},
		},
		{
			name:   "positive: provider is linked to workspaces(s), manually configured and force option is used; expect no errors",
			caller: &auth.SystemCaller{},
			input: &DeleteVCSProviderInput{
				Provider: sampleManualProvider,
				Force:    true,
			},
			links: []models.WorkspaceVCSProviderLink{
				{},
			},
			activityInput: &activityevent.CreateActivityEventInput{
				NamespacePath: &groupPath,
				Action:        models.ActionDeleteChildResource,
				TargetType:    models.TargetGroup,
				TargetID:      "group-id",
				Payload: &models.ActivityEventDeleteChildResourcePayload{
					Name: sampleAutomaticProvider.Name,
					ID:   sampleAutomaticProvider.Metadata.ID,
					Type: string(models.TargetVCSProvider),
				},
			},
		},
		{
			name:   "negative: provider is linked to workspace(s) and force option is not used; expect error EConflict",
			caller: &auth.SystemCaller{},
			input: &DeleteVCSProviderInput{
				Provider: sampleAutomaticProvider,
			},
			links: []models.WorkspaceVCSProviderLink{
				{},
			},
			expectedErrorCode: errors.EConflict,
		},
		{
			name: "negative: without caller; expect error EUnauthorized",
			input: &DeleteVCSProviderInput{
				Provider: sampleAutomaticProvider,
			},
			expectedErrorCode: errors.EUnauthorized,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := auth.WithCaller(context.Background(), test.caller)

			mockProviders := MockProvider{}
			mockVCSProviders := db.MockVCSProviders{}
			mockTransactions := db.MockTransactions{}
			mockWorkspaceVCSProviderLinks := db.MockWorkspaceVCSProviderLinks{}
			mockActivityEventService := activityevent.MockService{}

			mockProviders.Test(t)
			mockVCSProviders.Test(t)
			mockTransactions.Test(t)
			mockWorkspaceVCSProviderLinks.Test(t)
			mockActivityEventService.Test(t)

			createAccessTokenInput := &types.CreateAccessTokenInput{
				ProviderURL:  test.input.Provider.URL,
				ClientID:     test.input.Provider.OAuthClientID,
				ClientSecret: test.input.Provider.OAuthClientSecret,
				RedirectURI:  vcsOAuthCallbackURL,
			}

			createAccessTokenPayload := &types.AccessTokenPayload{AccessToken: "an-access-token"}

			mockProviders.On("CreateAccessToken", mock.Anything, createAccessTokenInput).Return(createAccessTokenPayload, nil)
			mockProviders.On("DeleteWebhook", mock.Anything, test.deleteWebhookInput).Return(nil)

			mockVCSProviders.On("UpdateProvider", mock.Anything, test.input.Provider).Return(&models.VCSProvider{}, nil)
			mockVCSProviders.On("DeleteProvider", mock.Anything, test.input.Provider).Return(nil)

			mockWorkspaceVCSProviderLinks.On("GetLinksByProviderID", mock.Anything, resourceUUID).Return(test.links, nil)

			mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
			mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
			mockTransactions.On("CommitTx", mock.Anything).Return(nil)

			mockActivityEventService.On("CreateActivityEvent", mock.Anything, test.activityInput).Return(nil, nil)

			dbClient := &db.Client{
				VCSProviders:              &mockVCSProviders,
				WorkspaceVCSProviderLinks: &mockWorkspaceVCSProviderLinks,
				Transactions:              &mockTransactions,
			}

			providerMap := map[models.VCSProviderType]Provider{
				models.GitLabProviderType: &mockProviders,
				models.GitHubProviderType: &mockProviders,
			}

			// Override state generator.
			stateGeneratorFunc := func() (uuid.UUID, error) {
				return sampleOAuthState, nil
			}

			logger, _ := logger.NewForTest()
			service := newService(logger, dbClient, nil, nil, providerMap, &mockActivityEventService, nil, nil, nil, stateGeneratorFunc, tharsisURL, 0)

			err := service.DeleteVCSProvider(ctx, test.input)
			if test.expectedErrorCode != "" {
				assert.Equal(t, test.expectedErrorCode, errors.ErrorCode(err))
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestGetVCSProviderLinkByWorkspaceID(t *testing.T) {
	sampleProviderLink := &models.WorkspaceVCSProviderLink{
		Metadata: models.ResourceMetadata{
			ID: resourceUUID,
		},
	}

	testCases := []struct {
		caller            auth.Caller
		expectedLink      *models.WorkspaceVCSProviderLink
		name              string
		workspaceID       string
		expectedErrorCode errors.CodeType
	}{
		{
			name:         "positive: with caller; expect a vcs provider",
			workspaceID:  resourceUUID,
			caller:       &auth.SystemCaller{},
			expectedLink: sampleProviderLink,
		},
		{
			name:              "negative: with caller, workspace is not linked; expect ENotFound",
			workspaceID:       resourceUUID,
			caller:            &auth.SystemCaller{},
			expectedErrorCode: errors.ENotFound,
		},
		{
			name:              "negative: without caller; expect error EUnauthorized",
			workspaceID:       resourceUUID,
			expectedErrorCode: errors.EUnauthorized,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := auth.WithCaller(context.Background(), test.caller)

			mockWorkspaceVCSProviderLinks := db.MockWorkspaceVCSProviderLinks{}
			mockWorkspaceVCSProviderLinks.Test(t)

			// VCSProvider mocks.
			mockWorkspaceVCSProviderLinks.On("GetLinkByWorkspaceID", mock.Anything, test.workspaceID).Return(test.expectedLink, nil)

			dbClient := &db.Client{
				WorkspaceVCSProviderLinks: &mockWorkspaceVCSProviderLinks,
			}

			service := newService(nil, dbClient, nil, nil, nil, nil, nil, nil, nil, nil, "", 0)

			link, err := service.GetWorkspaceVCSProviderLinkByWorkspaceID(ctx, test.workspaceID)
			if test.expectedErrorCode != "" {
				assert.Equal(t, test.expectedErrorCode, errors.ErrorCode(err))
			} else if err != nil {
				t.Fatal(err)
			} else {
				assert.Equal(t, test.expectedLink, link)
			}
		})
	}
}

func TestGetWorkspaceVCSProviderLinkByID(t *testing.T) {
	sampleProviderLink := &models.WorkspaceVCSProviderLink{
		Metadata: models.ResourceMetadata{
			ID: resourceUUID,
		},
	}

	testCases := []struct {
		caller            auth.Caller
		expectedLink      *models.WorkspaceVCSProviderLink
		name              string
		inputID           string
		expectedErrorCode errors.CodeType
	}{
		{
			name:         "positive: with caller; expect a vcs provider",
			inputID:      resourceUUID,
			caller:       &auth.SystemCaller{},
			expectedLink: sampleProviderLink,
		},
		{
			name:              "negative: with caller, no such link; expect ENotFound",
			inputID:           resourceUUID,
			caller:            &auth.SystemCaller{},
			expectedErrorCode: errors.ENotFound,
		},
		{
			name:              "negative: without caller; expect error EUnauthorized",
			inputID:           resourceUUID,
			expectedErrorCode: errors.EUnauthorized,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := auth.WithCaller(context.Background(), test.caller)

			mockWorkspaceVCSProviderLinks := db.MockWorkspaceVCSProviderLinks{}
			mockWorkspaceVCSProviderLinks.Test(t)

			// VCSProvider mocks.
			mockWorkspaceVCSProviderLinks.On("GetLinkByID", mock.Anything, test.inputID).Return(test.expectedLink, nil)

			dbClient := &db.Client{
				WorkspaceVCSProviderLinks: &mockWorkspaceVCSProviderLinks,
			}

			service := newService(nil, dbClient, nil, nil, nil, nil, nil, nil, nil, nil, "", 0)

			link, err := service.GetWorkspaceVCSProviderLinkByID(ctx, test.inputID)
			if test.expectedErrorCode != "" {
				assert.Equal(t, test.expectedErrorCode, errors.ErrorCode(err))
			} else if err != nil {
				t.Fatal(err)
			} else {
				assert.Equal(t, test.expectedLink, link)
			}
		})
	}
}

func TestGetWorkspaceVCSProviderLinkByTRN(t *testing.T) {
	sampleLink := &models.WorkspaceVCSProviderLink{
		Metadata: models.ResourceMetadata{
			ID:  "link-1",
			TRN: mtypes.WorkspaceVCSProviderLinkModelType.BuildTRN("link-gid-1"),
		},
		WorkspaceID:     "workspace-1",
		ProviderID:      "provider-1",
		RepositoryPath:  "owner/repo",
		Branch:          "main",
		WebhookID:       "webhook-1",
		WebhookDisabled: false,
	}

	type testCase struct {
		name            string
		authError       error
		link            *models.WorkspaceVCSProviderLink
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name: "successfully get link by trn",
			link: sampleLink,
		},
		{
			name:            "link not found",
			expectErrorCode: errors.ENotFound,
		},
		{
			name: "subject is not authorized to view link",
			link: &models.WorkspaceVCSProviderLink{
				Metadata:       sampleLink.Metadata,
				WorkspaceID:    sampleLink.WorkspaceID,
				ProviderID:     sampleLink.ProviderID,
				RepositoryPath: sampleLink.RepositoryPath,
			},
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockCaller := auth.NewMockCaller(t)
			mockWorkspaceVCSProviderLinks := db.NewMockWorkspaceVCSProviderLinks(t)

			mockWorkspaceVCSProviderLinks.On("GetLinkByTRN", mock.Anything, sampleLink.Metadata.TRN).Return(test.link, nil)

			if test.link != nil {
				mockCaller.On("RequirePermission", mock.Anything, models.ViewWorkspacePermission, mock.Anything).Return(test.authError)
			}

			dbClient := &db.Client{
				WorkspaceVCSProviderLinks: mockWorkspaceVCSProviderLinks,
			}

			service := &service{
				dbClient: dbClient,
			}

			actualLink, err := service.GetWorkspaceVCSProviderLinkByTRN(auth.WithCaller(ctx, mockCaller), sampleLink.Metadata.TRN)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.link, actualLink)
		})
	}
}

func TestCreateWorkspaceVCSProviderLink(t *testing.T) {
	sampleOAuthState, err := uuid.NewRandom()
	assert.Nil(t, err)

	sampleProjectPayload := &types.GetProjectPayload{
		DefaultBranch: "main",
	}

	sampleWebhookPayload := &types.WebhookPayload{
		WebhookID: "webhook-id",
	}

	sampleWorkspace := &models.Workspace{
		Metadata: models.ResourceMetadata{
			ID: "workspace-id",
		},
		FullPath: "full/path/to/workspace",
	}

	testCases := []struct {
		input              *CreateWorkspaceVCSProviderLinkInput
		getProjectInput    *types.GetProjectInput
		createWebhookInput *types.CreateWebhookInput
		createdLink        *models.WorkspaceVCSProviderLink
		updatedLink        *models.WorkspaceVCSProviderLink
		expectedResponse   *CreateWorkspaceVCSProviderLinkResponse
		existingProvider   *models.VCSProvider
		name               string
		expectedErrorCode  errors.CodeType
	}{
		{
			name: "positive: AutoCreateWebhooks is true; expect Tharsis configures webhook, no url in response",
			input: &CreateWorkspaceVCSProviderLinkInput{
				Workspace:       sampleWorkspace,
				ProviderID:      "provider-id",
				RepositoryPath:  "owner/repository",
				Branch:          ptr.String("main"),
				ModuleDirectory: ptr.String("hidden/../../path"),
				GlobPatterns:    []string{"/**/directory/*"},
				WebhookDisabled: true, // Tharsis should still configure the webhook.
			},
			getProjectInput: &types.GetProjectInput{
				ProviderURL:    sampleProviderURL,
				AccessToken:    "an-access-token",
				RepositoryPath: "owner/repository",
			},
			createWebhookInput: &types.CreateWebhookInput{
				ProviderURL:    sampleProviderURL,
				AccessToken:    "an-access-token",
				RepositoryPath: "owner/repository",
				WebhookToken:   []byte("signed-token"),
			},
			existingProvider: &models.VCSProvider{
				Metadata: models.ResourceMetadata{
					ID:  "provider-id",
					TRN: mtypes.VCSEventModelType.BuildTRN("full/path/to/provider"),
				},
				URL:                sampleProviderURL,
				OAuthClientID:      "a-sample-client-id",
				OAuthClientSecret:  "a-sample-client-secret",
				OAuthState:         ptr.String(sampleOAuthState.String()),
				OAuthAccessToken:   &sampleOAuthAccessToken,
				Type:               models.GitHubProviderType,
				AutoCreateWebhooks: true, // Tharsis configures the Git provider.
			},
			createdLink: &models.WorkspaceVCSProviderLink{
				Metadata: models.ResourceMetadata{
					ID: resourceUUID,
				},
				WorkspaceID:     sampleWorkspace.Metadata.ID,
				ProviderID:      "provider-id",
				RepositoryPath:  "owner/repository",
				Branch:          "main",
				CreatedBy:       "some@some-email",
				TokenNonce:      "some-token-nonce",
				ModuleDirectory: ptr.String("hidden/../../path"), // Should be cleaned.
				GlobPatterns:    []string{"/**/directory/*"},
				// Other fields should remain unchanged.
			},
			updatedLink: &models.WorkspaceVCSProviderLink{
				Metadata: models.ResourceMetadata{
					ID: resourceUUID,
				},
				WorkspaceID:     sampleWorkspace.Metadata.ID,
				ProviderID:      "provider-id",
				RepositoryPath:  "owner/repository",
				Branch:          "main",
				CreatedBy:       "some@some-email",
				WebhookID:       "webhook-id", // Only present if Tharsis configures Git provider.
				TokenNonce:      "some-token-nonce",
				ModuleDirectory: ptr.String("../path"), // Should be cleaned.
				GlobPatterns:    []string{"/**/directory/*"},
				WebhookDisabled: true,
				// Other fields should remain unchanged.
			},
			expectedResponse: &CreateWorkspaceVCSProviderLinkResponse{
				Link: &models.WorkspaceVCSProviderLink{
					Metadata: models.ResourceMetadata{
						ID: resourceUUID,
					},
					WorkspaceID:     sampleWorkspace.Metadata.ID,
					ProviderID:      "provider-id",
					RepositoryPath:  "owner/repository",
					Branch:          "main",
					CreatedBy:       "some@some-email",
					WebhookID:       "webhook-id", // Only present if Tharsis configures Git provider.
					TokenNonce:      "some-token-nonce",
					ModuleDirectory: ptr.String("../path"), // Should be cleaned.
					GlobPatterns:    []string{"/**/directory/*"},
					WebhookDisabled: true,
					// Other fields should remain unchanged.
				},
			},
		},
		{
			name: "positive: valid link, no branch, AutoCreateWebhooks is false; expect webhook url in response",
			input: &CreateWorkspaceVCSProviderLinkInput{
				Workspace:       sampleWorkspace,
				ProviderID:      "provider-id",
				RepositoryPath:  "owner/repository",
				TagRegex:        &sampleTagRegex,
				WebhookDisabled: false,
				// No branch here means, Tharsis should use the default
				// branch from the projectPayload.
			},
			getProjectInput: &types.GetProjectInput{
				ProviderURL:    sampleProviderURL,
				AccessToken:    "an-access-token",
				RepositoryPath: "owner/repository",
			},
			existingProvider: &models.VCSProvider{
				Metadata: models.ResourceMetadata{
					ID:  "provider-id",
					TRN: mtypes.VCSEventModelType.BuildTRN("full/path/to/provider"),
				},
				URL:                sampleProviderURL,
				OAuthClientID:      "a-sample-client-id",
				OAuthClientSecret:  "a-sample-client-secret",
				OAuthState:         ptr.String(sampleOAuthState.String()),
				OAuthAccessToken:   &sampleOAuthAccessToken,
				Type:               models.GitHubProviderType,
				AutoCreateWebhooks: false, // User configures the Git provider.
			},
			createdLink: &models.WorkspaceVCSProviderLink{
				Metadata: models.ResourceMetadata{
					ID: resourceUUID,
				},
				WorkspaceID:    sampleWorkspace.Metadata.ID,
				ProviderID:     "provider-id",
				RepositoryPath: "owner/repository",
				Branch:         "main", // Default branch.
				CreatedBy:      "some@some-email",
				TokenNonce:     "some-token-nonce",
				TagRegex:       &sampleTagRegex,
				// Other fields should remain unchanged.
			},
			expectedResponse: &CreateWorkspaceVCSProviderLinkResponse{
				Link: &models.WorkspaceVCSProviderLink{
					Metadata: models.ResourceMetadata{
						ID: resourceUUID,
					},
					WorkspaceID:    sampleWorkspace.Metadata.ID,
					ProviderID:     "provider-id",
					RepositoryPath: "owner/repository",
					Branch:         "main",
					CreatedBy:      "some@some-email",
					TokenNonce:     "some-token-nonce",
					TagRegex:       &sampleTagRegex,
				},
				WebhookURL: ptr.String("https://tharsis.domain/v1/vcs/events?token=signed-token"),
				// No token for GitHub.
			},
		},
		{
			name: "negative: vcs provider is not in the same group hierarchy; expect error EInvalid",
			input: &CreateWorkspaceVCSProviderLinkInput{
				Workspace:  sampleWorkspace,
				ProviderID: "provider-id",
			},
			existingProvider: &models.VCSProvider{
				Metadata: models.ResourceMetadata{
					ID:  "provider-id",
					TRN: mtypes.VCSProviderModelType.BuildTRN("some/resource/path"), // Different from workspace.
				},
			},
			expectedErrorCode: errors.EInvalid,
		},
		{
			name: "negative: OAuth flow hasn't been completed; expect error EInvalid",
			input: &CreateWorkspaceVCSProviderLinkInput{
				Workspace:  sampleWorkspace,
				ProviderID: "provider-id",
			},
			existingProvider: &models.VCSProvider{
				Metadata: models.ResourceMetadata{
					ID:  "provider-id",
					TRN: mtypes.VCSEventModelType.BuildTRN("path/provider"),
				},
			},
			expectedErrorCode: errors.EInvalid,
		},
		{
			name: "negative: no such provider; expect error EInvalid",
			input: &CreateWorkspaceVCSProviderLinkInput{
				Workspace:      sampleWorkspace,
				ProviderID:     "provider-id",
				RepositoryPath: "owner/repository",
			},
			existingProvider: &models.VCSProvider{
				Metadata: models.ResourceMetadata{
					ID:  "provider-id",
					TRN: mtypes.VCSEventModelType.BuildTRN("path/provider"),
				},
			},
			expectedErrorCode: errors.EInvalid,
		},
		{
			name: "negative: invalid repository path; expect error EInvalid",
			input: &CreateWorkspaceVCSProviderLinkInput{
				Workspace:      sampleWorkspace,
				ProviderID:     "provider-id",
				RepositoryPath: "owner",
			},
			existingProvider: &models.VCSProvider{
				Metadata: models.ResourceMetadata{
					ID:  "provider-id",
					TRN: mtypes.VCSEventModelType.BuildTRN("path/provider"),
				},
			},
			expectedErrorCode: errors.EInvalid,
		},
		{
			name: "negative: invalid glob pattern; expect error EInvalid",
			input: &CreateWorkspaceVCSProviderLinkInput{
				Workspace:      sampleWorkspace,
				ProviderID:     "provider-id",
				RepositoryPath: "owner/repository",
				GlobPatterns:   []string{"[invalid"},
			},
			existingProvider: &models.VCSProvider{
				Metadata: models.ResourceMetadata{
					ID:  "provider-id",
					TRN: mtypes.VCSEventModelType.BuildTRN("path/provider"),
				},
			},
			expectedErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockCaller := auth.MockCaller{}
			mockProviders := MockProvider{}
			mockTransactions := db.MockTransactions{}
			mockVCSProviders := db.MockVCSProviders{}

			mockWorkspaceVCSProviderLinks := db.MockWorkspaceVCSProviderLinks{}

			mockCaller.Test(t)
			mockProviders.Test(t)
			mockTransactions.Test(t)
			mockVCSProviders.Test(t)
			mockWorkspaceVCSProviderLinks.Test(t)
			MockSigningKeyManager := auth.NewMockSigningKeyManager(t)

			mockCaller.On("GetSubject").Return("testsubject")
			mockCaller.On("RequirePermission", mock.Anything, models.UpdateWorkspacePermission, mock.Anything).Return(nil)
			ctx := auth.WithCaller(context.Background(), &mockCaller)

			createAccessTokenInput := &types.CreateAccessTokenInput{
				ProviderURL:  test.existingProvider.URL,
				ClientID:     test.existingProvider.OAuthClientID,
				ClientSecret: test.existingProvider.OAuthClientSecret,
				RedirectURI:  vcsOAuthCallbackURL,
			}

			createAccessTokenPayload := &types.AccessTokenPayload{AccessToken: "an-access-token"}

			mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
			mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
			mockTransactions.On("CommitTx", mock.Anything).Return(nil)

			mockProviders.On("CreateAccessToken", mock.Anything, createAccessTokenInput).Return(createAccessTokenPayload, nil)
			mockProviders.On("GetProject", mock.Anything, test.getProjectInput).Return(sampleProjectPayload, nil)
			mockProviders.On("CreateWebhook", mock.Anything, test.createWebhookInput).Return(sampleWebhookPayload, nil)

			mockVCSProviders.On("GetProviderByID", mock.Anything, "provider-id").Return(test.existingProvider, nil)

			MockSigningKeyManager.On("GenerateToken", mock.Anything, mock.Anything).Return([]byte("signed-token"), nil).Maybe()

			mockWorkspaceVCSProviderLinks.On("CreateLink", mock.Anything, mock.Anything).Return(test.createdLink, nil)
			mockWorkspaceVCSProviderLinks.On("UpdateLink", mock.Anything, test.createdLink).Return(test.updatedLink, nil)

			dbClient := &db.Client{
				VCSProviders:              &mockVCSProviders,
				WorkspaceVCSProviderLinks: &mockWorkspaceVCSProviderLinks,
				Transactions:              &mockTransactions,
			}

			providerMap := map[models.VCSProviderType]Provider{
				models.GitLabProviderType: &mockProviders,
				models.GitHubProviderType: &mockProviders,
			}

			// Override state generator.
			stateGeneratorFunc := func() (uuid.UUID, error) {
				return sampleOAuthState, nil
			}

			logger, _ := logger.NewForTest()
			service := newService(logger, dbClient, nil, MockSigningKeyManager, providerMap, nil, nil, nil, nil, stateGeneratorFunc, tharsisURL, 0)

			response, err := service.CreateWorkspaceVCSProviderLink(ctx, test.input)
			if test.expectedErrorCode != "" {
				assert.Equal(t, test.expectedErrorCode, errors.ErrorCode(err))
			} else if err != nil {
				t.Fatal(err)
			} else {
				assert.Equal(t, test.expectedResponse, response)
			}
		})
	}
}

func TestUpdateWorkspaceVCSProviderLink(t *testing.T) {
	testCases := []struct {
		name              string
		caller            auth.Caller
		input             *UpdateWorkspaceVCSProviderLinkInput
		existingLink      *models.WorkspaceVCSProviderLink
		expectedLink      *models.WorkspaceVCSProviderLink
		expectedErrorCode errors.CodeType
	}{
		{
			name:   "positive: valid update input; expect no errors",
			caller: &auth.SystemCaller{},
			input: &UpdateWorkspaceVCSProviderLinkInput{
				Link: &models.WorkspaceVCSProviderLink{
					Metadata: models.ResourceMetadata{
						ID: resourceUUID,
					},
					RepositoryPath: "owner/repository",
					Branch:         "main",
				},
			},
			existingLink: &models.WorkspaceVCSProviderLink{
				Metadata: models.ResourceMetadata{
					ID: resourceUUID,
				},
				RepositoryPath: "owner/repository",
				Branch:         "feature/branch",
			},
			expectedLink: &models.WorkspaceVCSProviderLink{
				Metadata: models.ResourceMetadata{
					ID: resourceUUID,
				},
				RepositoryPath: "owner/repository",
				Branch:         "main",
			},
		},
		{
			name:   "negative: invalid glob pattern; expect error EInvalid",
			caller: &auth.SystemCaller{},
			input: &UpdateWorkspaceVCSProviderLinkInput{
				&models.WorkspaceVCSProviderLink{
					Metadata: models.ResourceMetadata{
						ID: resourceUUID,
					},
					RepositoryPath: "owner/repository",
					GlobPatterns:   []string{"[invalid"},
				},
			},
			expectedErrorCode: errors.EInvalid,
		},
		{
			name:              "negative: without caller; expect error EUnauthorized",
			input:             &UpdateWorkspaceVCSProviderLinkInput{},
			expectedErrorCode: errors.EUnauthorized,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := auth.WithCaller(context.Background(), test.caller)

			mockWorkspaceVCSProviderLinks := db.MockWorkspaceVCSProviderLinks{}

			mockWorkspaceVCSProviderLinks.Test(t)

			mockWorkspaceVCSProviderLinks.On("UpdateLink", mock.Anything, test.input.Link).Return(test.expectedLink, nil)

			dbClient := &db.Client{
				WorkspaceVCSProviderLinks: &mockWorkspaceVCSProviderLinks,
			}

			logger, _ := logger.NewForTest()
			service := newService(logger, dbClient, nil, nil, nil, nil, nil, nil, nil, nil, "", 0)

			link, err := service.UpdateWorkspaceVCSProviderLink(ctx, test.input)
			if test.expectedErrorCode != "" {
				assert.Equal(t, test.expectedErrorCode, errors.ErrorCode(err))
			} else if err != nil {
				t.Fatal(err)
			} else {
				assert.Equal(t, test.expectedLink, link)
			}
		})
	}
}

func TestDeleteWorkspaceVCSProviderLink(t *testing.T) {
	sampleOAuthState, err := uuid.NewRandom()
	assert.Nil(t, err)

	testCases := []struct {
		caller             auth.Caller
		name               string
		input              *DeleteWorkspaceVCSProviderLinkInput
		deleteWebhookInput *types.DeleteWebhookInput
		existingProvider   *models.VCSProvider
		expectedErrorCode  errors.CodeType
	}{
		{
			name:   "positive: valid input, manually configured provider; expect no errors",
			caller: &auth.SystemCaller{},
			input: &DeleteWorkspaceVCSProviderLinkInput{
				Link: &models.WorkspaceVCSProviderLink{
					Metadata: models.ResourceMetadata{
						ID: resourceUUID,
					},
					ProviderID:  "provider-id",
					WorkspaceID: "workspace-id",
				},
				Force: true,
			},
			existingProvider: &models.VCSProvider{
				AutoCreateWebhooks: false, // Manually configured provider.
			},
		},
		{
			name:   "positive: valid input, automatically configured provider; expect no errors",
			caller: &auth.SystemCaller{},
			input: &DeleteWorkspaceVCSProviderLinkInput{
				Link: &models.WorkspaceVCSProviderLink{
					Metadata: models.ResourceMetadata{
						ID: resourceUUID,
					},
					ProviderID:     "provider-id",
					WorkspaceID:    "workspace-id",
					RepositoryPath: "owner/repository",
					WebhookID:      "webhook-id",
				},
				Force: false,
			},
			deleteWebhookInput: &types.DeleteWebhookInput{
				ProviderURL:    sampleProviderURL,
				AccessToken:    "an-access-token",
				RepositoryPath: "owner/repository",
				WebhookID:      "webhook-id",
			},
			existingProvider: &models.VCSProvider{
				URL:                sampleProviderURL,
				OAuthClientID:      "a-sample-client-id",
				OAuthClientSecret:  "a-sample-client-secret",
				OAuthState:         ptr.String(sampleOAuthState.String()),
				OAuthAccessToken:   &sampleOAuthAccessToken,
				Type:               models.GitHubProviderType,
				AutoCreateWebhooks: true, // Automatically configured provider.
			},
		},
		{
			name:              "negative: without caller; expect error EUnauthorized",
			input:             &DeleteWorkspaceVCSProviderLinkInput{Link: &models.WorkspaceVCSProviderLink{}},
			existingProvider:  &models.VCSProvider{},
			expectedErrorCode: errors.EUnauthorized,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := auth.WithCaller(context.Background(), test.caller)

			mockProviders := MockProvider{}
			mockVCSProviders := db.MockVCSProviders{}
			mockWorkspaceVCSProviderLinks := db.MockWorkspaceVCSProviderLinks{}

			mockProviders.Test(t)
			mockVCSProviders.Test(t)
			mockWorkspaceVCSProviderLinks.Test(t)

			createAccessTokenInput := &types.CreateAccessTokenInput{
				ProviderURL:  test.existingProvider.URL,
				ClientID:     test.existingProvider.OAuthClientID,
				ClientSecret: test.existingProvider.OAuthClientSecret,
				RedirectURI:  oAuthCallBackEndpoint,
			}

			createAccessTokenPayload := &types.AccessTokenPayload{AccessToken: "an-access-token"}

			mockProviders.On("CreateAccessToken", mock.Anything, createAccessTokenInput).Return(createAccessTokenPayload, nil)
			mockProviders.On("DeleteWebhook", mock.Anything, test.deleteWebhookInput).Return(nil)

			mockVCSProviders.On("GetProviderByID", mock.Anything, test.input.Link.ProviderID).Return(test.existingProvider, nil)
			mockVCSProviders.On("UpdateProvider", mock.Anything, test.existingProvider).Return(&models.VCSProvider{}, nil)

			mockWorkspaceVCSProviderLinks.On("DeleteLink", mock.Anything, test.input.Link).Return(nil)

			dbClient := &db.Client{
				VCSProviders:              &mockVCSProviders,
				WorkspaceVCSProviderLinks: &mockWorkspaceVCSProviderLinks,
			}

			providerMap := map[models.VCSProviderType]Provider{
				models.GitLabProviderType: &mockProviders,
				models.GitHubProviderType: &mockProviders,
			}

			oAuthStateGenerator := func() (uuid.UUID, error) {
				return sampleOAuthState, nil
			}

			logger, _ := logger.NewForTest()
			service := newService(logger, dbClient, nil, nil, providerMap, nil, nil, nil, nil, oAuthStateGenerator, "", 0)

			err := service.DeleteWorkspaceVCSProviderLink(ctx, test.input)
			if test.expectedErrorCode != "" {
				assert.Equal(t, test.expectedErrorCode, errors.ErrorCode(err))
			} else if err != nil {
				t.Fatal(err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestCreateVCSRun(t *testing.T) {
	sampleOAuthState, err := uuid.NewRandom()
	assert.Nil(t, err)

	sampleRepositoryURL := "https://some.gitlab.instance.com/owner/repository"

	sampleWorkspace := &models.Workspace{
		Metadata: models.ResourceMetadata{
			ID: "workspace-id",
		},
		FullPath: "path/to/workspace",
	}

	sampleLink := &models.WorkspaceVCSProviderLink{
		Metadata: models.ResourceMetadata{
			ID: "link-id",
		},
		Branch:         "feature/another-branch", // API should default to this, when referenceName is nil.
		RepositoryPath: "owner/repository",
		WorkspaceID:    "workspace-id",
	}

	sampleVCSProvider := &models.VCSProvider{
		URL:               sampleProviderURL,
		OAuthClientID:     "a-sample-client-id",
		OAuthClientSecret: "a-sample-client-secret",
		OAuthState:        ptr.String(sampleOAuthState.String()),
		OAuthAccessToken:  &sampleOAuthAccessToken,
		Type:              models.GitLabProviderType,
	}

	testCases := []struct {
		caller            auth.Caller
		input             *CreateVCSRunInput
		existingLink      *models.WorkspaceVCSProviderLink
		name              string
		expectedErrorCode errors.CodeType
	}{
		{
			name:   "positive: no referenceName, not destroy; expect no errors",
			caller: &auth.SystemCaller{},
			input: &CreateVCSRunInput{
				Workspace: sampleWorkspace,
				// ReferenceName will be nil and isDestroy false.
			},
			existingLink: sampleLink,
		},
		{
			name:   "positive: referenceName, not destroy; expect no errors",
			caller: &auth.SystemCaller{},
			input: &CreateVCSRunInput{
				Workspace:     sampleWorkspace,
				ReferenceName: ptr.String("feature/branch"),
				// IsDestroy is false here.
			},
			existingLink: sampleLink,
		},
		{
			name:   "positive: referenceName, is destroy; expect no errors",
			caller: &auth.SystemCaller{},
			input: &CreateVCSRunInput{
				Workspace:     sampleWorkspace,
				ReferenceName: ptr.String("feature/branch"),
				IsDestroy:     true,
			},
			existingLink: sampleLink,
		},
		{
			name:   "negative: workspace is not linked to a VCS provider; expect error EInvalid",
			caller: &auth.SystemCaller{},
			input: &CreateVCSRunInput{
				Workspace: sampleWorkspace,
				// Other fields won't matter.
			},
			expectedErrorCode: errors.EInvalid,
			// existingLink will be nil.
		},
		{
			name:              "negative: without caller; expect error EUnauthorized",
			input:             &CreateVCSRunInput{},
			expectedErrorCode: errors.EUnauthorized,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := auth.WithCaller(context.Background(), test.caller)

			mockProviders := MockProvider{}
			mockVCSEvents := db.MockVCSEvents{}
			mockManager := asynctask.MockManager{}
			mockVCSProviders := db.MockVCSProviders{}
			mockWorkspaceVCSProviderLinks := db.MockWorkspaceVCSProviderLinks{}

			mockProviders.Test(t)
			mockVCSEvents.Test(t)
			mockManager.Test(t)
			mockVCSProviders.Test(t)
			mockWorkspaceVCSProviderLinks.Test(t)

			referenceName := sampleLink.Branch
			if test.input.ReferenceName != nil {
				referenceName = *test.input.ReferenceName
			}

			createEventInput := &models.VCSEvent{
				SourceReferenceName: &referenceName,
				WorkspaceID:         sampleWorkspace.Metadata.ID,
				Type:                models.ManualEventType,
				Status:              models.VCSEventPending,
				RepositoryURL:       sampleRepositoryURL,
			}

			createAccessTokenInput := &types.CreateAccessTokenInput{
				ProviderURL:  sampleVCSProvider.URL,
				ClientID:     sampleVCSProvider.OAuthClientID,
				ClientSecret: sampleVCSProvider.OAuthClientSecret,
				RedirectURI:  oAuthCallBackEndpoint,
			}

			buildRepositoryURLInput := &types.BuildRepositoryURLInput{
				ProviderURL:    sampleVCSProvider.URL,
				RepositoryPath: sampleLink.RepositoryPath,
			}

			createAccessTokenPayload := &types.AccessTokenPayload{AccessToken: "an-access-token"}

			mockProviders.On("CreateAccessToken", mock.Anything, createAccessTokenInput).Return(createAccessTokenPayload, nil)
			mockProviders.On("BuildRepositoryURL", buildRepositoryURLInput).Return(sampleRepositoryURL, nil)

			mockVCSEvents.On("CreateEvent", mock.Anything, createEventInput).Return(&models.VCSEvent{}, nil)

			mockVCSProviders.On("GetProviderByID", mock.Anything, mock.Anything).Return(sampleVCSProvider, nil)
			mockVCSProviders.On("UpdateProvider", mock.Anything, sampleVCSProvider).Return(&models.VCSProvider{}, nil)

			mockWorkspaceVCSProviderLinks.On("GetLinkByWorkspaceID", mock.Anything, mock.Anything).Return(test.existingLink, nil)

			mockManager.On("StartTask", mock.Anything)

			dbClient := &db.Client{
				VCSProviders:              &mockVCSProviders,
				WorkspaceVCSProviderLinks: &mockWorkspaceVCSProviderLinks,
				VCSEvents:                 &mockVCSEvents,
			}

			providerMap := map[models.VCSProviderType]Provider{
				models.GitLabProviderType: &mockProviders,
				models.GitHubProviderType: &mockProviders,
			}

			oAuthStateGenerator := func() (uuid.UUID, error) {
				return sampleOAuthState, nil
			}

			logger, _ := logger.NewForTest()
			service := newService(logger, dbClient, nil, nil, providerMap, nil, nil, nil, &mockManager, oAuthStateGenerator, "", 5000)

			err := service.CreateVCSRun(ctx, test.input)
			if test.expectedErrorCode != "" {
				assert.Equal(t, test.expectedErrorCode, errors.ErrorCode(err))
			} else if err != nil {
				t.Fatal(err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestProcessWebhookEvent(t *testing.T) {
	sampleRepositoryURL := "https://github.com/owner/repository"

	sampleWorkspace := &models.Workspace{
		Metadata: models.ResourceMetadata{
			ID: "workspace-id",
		},
		FullPath: "path/to/workspace",
	}

	sampleOAuthState, err := uuid.NewRandom()
	assert.Nil(t, err)

	sampleVCSProvider := &models.VCSProvider{
		Type:              models.GitHubProviderType,
		URL:               sampleProviderURL,
		OAuthClientID:     "a-sample-client-id",
		OAuthClientSecret: "a-sample-client-secret",
		OAuthState:        ptr.String(sampleOAuthState.String()),
		OAuthAccessToken:  &sampleOAuthAccessToken,
	}

	// SourceReferenceName for VCS events.
	sampleReferenceNames := []string{
		"refs/heads/main",
		"refs/tags/v0.1",
		"feature/branch",
	}

	testCases := []struct {
		link                *models.WorkspaceVCSProviderLink
		input               *ProcessWebhookEventInput
		createEventInput    *models.VCSEvent
		equivalentEventType models.VCSEventType
		name                string
		expectedErrorCode   errors.CodeType
	}{
		{
			name: "positive: valid branch push event, mostly empty link and provider setup; expect no errors",
			link: &models.WorkspaceVCSProviderLink{
				RepositoryPath:      "owner/repository",
				WorkspaceID:         "workspace-id",
				Branch:              "main", // Only allow events for main branch.
				AutoSpeculativePlan: false,
			},
			input: &ProcessWebhookEventInput{
				EventHeader: "push",                     // Corresponds to a GitHub push event.
				Before:      plumbing.ZeroHash.String(), // Represents an empty hash.
				After:       sampleAfterCommit,
				Ref:         "refs/heads/main", // Happening on the main branch.
			},
			equivalentEventType: models.BranchEventType,
			createEventInput: &models.VCSEvent{
				SourceReferenceName: &sampleReferenceNames[0],
				CommitID:            &sampleAfterCommit,
				WorkspaceID:         sampleWorkspace.Metadata.ID,
				Type:                models.BranchEventType,
				Status:              models.VCSEventPending,
				RepositoryURL:       sampleRepositoryURL,
			},
		},
		{
			name: "positive: valid branch push event, mostly empty link and provider setup; expect no errors",
			link: &models.WorkspaceVCSProviderLink{
				RepositoryPath:      "owner/repository",
				WorkspaceID:         "workspace-id",
				Branch:              "main", // Only allow events for main branch.
				AutoSpeculativePlan: false,
			},
			input: &ProcessWebhookEventInput{
				EventHeader: "push", // Corresponds to a GitHub push event.
				Before:      sampleBeforeCommit,
				After:       sampleAfterCommit,
				Ref:         "refs/heads/main", // Happening on the main branch.
			},
			equivalentEventType: models.BranchEventType,
			createEventInput: &models.VCSEvent{
				SourceReferenceName: &sampleReferenceNames[0],
				CommitID:            &sampleAfterCommit,
				WorkspaceID:         sampleWorkspace.Metadata.ID,
				Type:                models.BranchEventType,
				Status:              models.VCSEventPending,
				RepositoryURL:       sampleRepositoryURL,
			},
		},
		{
			name: "positive: valid tag event, no tag regex defined on link; expect no errors",
			link: &models.WorkspaceVCSProviderLink{
				RepositoryPath:      "owner/repository",
				WorkspaceID:         "workspace-id",
				Branch:              "main",
				AutoSpeculativePlan: false,
				// No tag regex, meaning no run.
			},
			input: &ProcessWebhookEventInput{
				EventHeader: "push", // Corresponds to a GitHub push event.
				Ref:         "refs/tags/v0.1",
				After:       sampleAfterCommit,
			},
			equivalentEventType: models.TagEventType,
			createEventInput: &models.VCSEvent{
				SourceReferenceName: &sampleReferenceNames[1],
				CommitID:            &sampleAfterCommit,
				WorkspaceID:         sampleWorkspace.Metadata.ID,
				Type:                models.TagEventType,
				Status:              models.VCSEventPending,
				RepositoryURL:       sampleRepositoryURL,
			},
		},
		{
			name: "positive: valid tag event, with tag regex defined on link; expect no errors",
			link: &models.WorkspaceVCSProviderLink{
				RepositoryPath:      "owner/repository",
				WorkspaceID:         "workspace-id",
				Branch:              "main",
				AutoSpeculativePlan: false,
				TagRegex:            &sampleTagRegex,
			},
			input: &ProcessWebhookEventInput{
				EventHeader: "push", // Corresponds to a GitHub push event.
				Ref:         "refs/tags/v0.1",
				After:       sampleAfterCommit,
			},
			equivalentEventType: models.TagEventType,
			createEventInput: &models.VCSEvent{
				SourceReferenceName: &sampleReferenceNames[1],
				CommitID:            &sampleAfterCommit,
				WorkspaceID:         sampleWorkspace.Metadata.ID,
				Type:                models.TagEventType,
				Status:              models.VCSEventPending,
				RepositoryURL:       sampleRepositoryURL,
			},
		},
		{
			name: "positive: valid PR event, auto-speculative is false on link; expect no errors",
			link: &models.WorkspaceVCSProviderLink{
				RepositoryPath:      "owner/repository",
				WorkspaceID:         "workspace-id",
				Branch:              "main",
				AutoSpeculativePlan: false, // No PR's allowed here.
				TagRegex:            &sampleTagRegex,
			},
			input: &ProcessWebhookEventInput{
				EventHeader:      "pull_request", // Corresponds to a GitHub PR event.
				SourceRepository: "owner/repository",
				SourceBranch:     "feature/branch",
				TargetBranch:     "main",
				Action:           "opened",
				HeadCommitID:     "sample-commit-id",
			},
			equivalentEventType: models.MergeRequestEventType,
			createEventInput: &models.VCSEvent{
				SourceReferenceName: &sampleReferenceNames[2],
				CommitID:            ptr.String("sample-commit-id"),
				WorkspaceID:         sampleWorkspace.Metadata.ID,
				Type:                models.MergeRequestEventType,
				Status:              models.VCSEventPending,
				RepositoryURL:       sampleRepositoryURL,
			},
		},
		{
			name: "positive: valid PR event, auto-speculative is true on link; expect no errors",
			link: &models.WorkspaceVCSProviderLink{
				RepositoryPath:      "owner/repository",
				WorkspaceID:         "workspace-id",
				Branch:              "main",
				AutoSpeculativePlan: true, // PR's allowed here.
				TagRegex:            &sampleTagRegex,
			},
			input: &ProcessWebhookEventInput{
				EventHeader:      "pull_request", // Corresponds to a GitHub PR event.
				SourceRepository: "owner/repository",
				SourceBranch:     "feature/branch",
				TargetBranch:     "main",
				Action:           "opened",
				HeadCommitID:     "sample-commit-id",
			},
			equivalentEventType: models.MergeRequestEventType,
			createEventInput: &models.VCSEvent{
				SourceReferenceName: &sampleReferenceNames[2],
				CommitID:            ptr.String("sample-commit-id"),
				WorkspaceID:         sampleWorkspace.Metadata.ID,
				Type:                models.MergeRequestEventType,
				Status:              models.VCSEventPending,
				RepositoryURL:       sampleRepositoryURL,
			},
		},
		{
			name: "positive: webhook is disabled on the link; expect no errors",
			link: &models.WorkspaceVCSProviderLink{
				WorkspaceID:     "workspace-id",
				WebhookDisabled: true, // Webhook is disabled.
			},
			input: &ProcessWebhookEventInput{},
		},
		{
			name: "negative: invalid webhook event; expect no errors",
			link: &models.WorkspaceVCSProviderLink{
				WorkspaceID: "workspace-id",
			},
			input: &ProcessWebhookEventInput{
				EventHeader: "unknown", // An event not supported.
			},
			// Expect error to be nil.
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockProviders := MockProvider{}
			mockVCSProviders := db.MockVCSProviders{}
			mockVCSEvents := db.MockVCSEvents{}
			mockManager := asynctask.MockManager{}
			mockWorkspaceService := workspace.MockService{}
			mockMaintenanceMonitor := maintenance.MockMonitor{}

			mockProviders.Test(t)
			mockVCSProviders.Test(t)
			mockVCSEvents.Test(t)
			mockWorkspaceService.Test(t)
			mockManager.Test(t)
			mockMaintenanceMonitor.Test(t)

			createAccessTokenInput := &types.CreateAccessTokenInput{
				ProviderURL:  sampleVCSProvider.URL,
				ClientID:     sampleVCSProvider.OAuthClientID,
				ClientSecret: sampleVCSProvider.OAuthClientSecret,
				RedirectURI:  oAuthCallBackEndpoint,
			}

			createAccessTokenPayload := &types.AccessTokenPayload{AccessToken: "an-access-token"}

			toVCSEventInput := &types.ToVCSEventTypeInput{
				EventHeader: test.input.EventHeader,
				Ref:         test.input.Ref,
			}

			buildRepositoryURLInput := &types.BuildRepositoryURLInput{
				ProviderURL:    sampleVCSProvider.URL,
				RepositoryPath: "owner/repository",
			}

			mockProviders.On("ToVCSEventType", toVCSEventInput).Return(test.equivalentEventType)
			mockProviders.On("MergeRequestActionIsSupported", test.input.Action).Return(test.input.Action == "opened")
			mockProviders.On("CreateAccessToken", mock.Anything, createAccessTokenInput).Return(createAccessTokenPayload, nil)
			mockProviders.On("BuildRepositoryURL", buildRepositoryURLInput).Return(sampleRepositoryURL, nil)

			mockVCSProviders.On("UpdateProvider", mock.Anything, sampleVCSProvider).Return(&models.VCSProvider{}, nil)

			mockWorkspaceService.On("GetWorkspaceByID", mock.Anything, mock.Anything).Return(sampleWorkspace, nil)
			mockVCSEvents.On("CreateEvent", mock.Anything, test.createEventInput).Return(&models.VCSEvent{}, nil)

			mockMaintenanceMonitor.On("InMaintenanceMode", mock.Anything).Return(false, nil)

			mockManager.On("StartTask", mock.Anything)

			dbClient := &db.Client{
				VCSEvents:    &mockVCSEvents,
				VCSProviders: &mockVCSProviders,
			}

			caller := auth.NewVCSWorkspaceLinkCaller(sampleVCSProvider, test.link, dbClient, &mockMaintenanceMonitor)

			providerMap := map[models.VCSProviderType]Provider{
				models.GitLabProviderType: &mockProviders,
				models.GitHubProviderType: &mockProviders,
			}

			oAuthStateGenerator := func() (uuid.UUID, error) {
				return sampleOAuthState, nil
			}

			logger, _ := logger.NewForTest()
			service := newService(logger, dbClient, nil, nil, providerMap, nil, nil, &mockWorkspaceService, &mockManager, oAuthStateGenerator, "", 5000)

			err := service.ProcessWebhookEvent(auth.WithCaller(context.Background(), caller), test.input)
			if test.expectedErrorCode != "" {
				assert.Equal(t, test.expectedErrorCode, errors.ErrorCode(err))
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestResetVCSProviderOAuthToken(t *testing.T) {
	sampleOAuthState, err := uuid.NewRandom()
	assert.Nil(t, err)

	sampleVCSProvider := &models.VCSProvider{
		Type:              models.GitHubProviderType,
		URL:               sampleProviderURL,
		OAuthClientID:     "a-sample-client-id",
		OAuthClientSecret: "a-sample-client-secret",
		OAuthState:        ptr.String("sample-state"),
	}

	testCases := []struct {
		expectedResponse  *ResetVCSProviderOAuthTokenResponse
		input             *ResetVCSProviderOAuthTokenInput
		caller            auth.Caller
		name              string
		expectedErrorCode errors.CodeType
	}{
		{
			name:   "positive: with caller; expect no errors",
			caller: &auth.SystemCaller{},
			input: &ResetVCSProviderOAuthTokenInput{
				VCSProvider: sampleVCSProvider,
			},
			expectedResponse: &ResetVCSProviderOAuthTokenResponse{
				VCSProvider: &models.VCSProvider{
					Type:       models.GitHubProviderType,
					URL:        sampleProviderURL,
					OAuthState: ptr.String("a-new-state"),
				},
				OAuthAuthorizationURL: "expected-url",
			},
		},
		{
			name: "negative: without caller; expect error EUnauthorized",
			input: &ResetVCSProviderOAuthTokenInput{
				VCSProvider: sampleVCSProvider,
			},
			expectedResponse:  &ResetVCSProviderOAuthTokenResponse{},
			expectedErrorCode: errors.EUnauthorized,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := auth.WithCaller(context.Background(), test.caller)

			mockVCSProviders := db.MockVCSProviders{}
			mockVCSProviders.Test(t)

			mockVCSProviders.On("UpdateProvider", mock.Anything, sampleVCSProvider).Return(test.expectedResponse.VCSProvider, nil)

			mockProviders := MockProvider{}
			mockProviders.Test(t)

			mockProviders.On("BuildOAuthAuthorizationURL", mock.Anything).Return("expected-url", nil)

			providerMap := map[models.VCSProviderType]Provider{
				models.GitLabProviderType: &mockProviders,
				models.GitHubProviderType: &mockProviders,
			}

			dbClient := &db.Client{
				VCSProviders: &mockVCSProviders,
			}

			oAuthStateGenerator := func() (uuid.UUID, error) {
				return sampleOAuthState, nil
			}

			logger, _ := logger.NewForTest()
			service := newService(logger, dbClient, nil, nil, providerMap, nil, nil, nil, nil, oAuthStateGenerator, "", 5000)

			response, err := service.ResetVCSProviderOAuthToken(ctx, test.input)
			if test.expectedErrorCode != "" {
				assert.Equal(t, test.expectedErrorCode, errors.ErrorCode(err))
			} else if err != nil {
				t.Fatal(err)
			} else {
				assert.Equal(t, test.expectedResponse, response)
			}
		})
	}
}

func TestGetOAuthAuthorizationURL(t *testing.T) {
	testCases := []struct {
		caller            auth.Caller
		input             *models.VCSProvider
		expectedURL       string
		name              string
		expectedErrorCode errors.CodeType
	}{
		{
			name:   "positive: vcs provider has a non-nil state value; expect no errors",
			caller: &auth.SystemCaller{},
			input: &models.VCSProvider{
				URL:               sampleProviderURL,
				OAuthClientID:     "sample-client-id",
				OAuthClientSecret: "sample-client-secret",
				GroupID:           "group-id",
				OAuthState:        ptr.String("a-state-value"),
				Type:              models.GitLabProviderType,
			},
			expectedURL: "expected-url",
		},
		{
			name:   "positive: vcs provider has a nil state value; expect no errors",
			caller: &auth.SystemCaller{},
			input: &models.VCSProvider{
				GroupID: "group-id",
			},
			expectedErrorCode: errors.EInternal,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := auth.WithCaller(context.Background(), test.caller)

			mockProviders := MockProvider{}
			mockProviders.Test(t)

			buildAuthURLInput := &types.BuildOAuthAuthorizationURLInput{
				ProviderURL:        test.input.URL,
				OAuthClientID:      test.input.OAuthClientID,
				OAuthState:         "a-state-value",
				RedirectURL:        vcsOAuthCallbackURL,
				UseReadWriteScopes: false,
			}

			mockProviders.On("BuildOAuthAuthorizationURL", buildAuthURLInput).Return("expected-url", nil)

			providerMap := map[models.VCSProviderType]Provider{
				models.GitLabProviderType: &mockProviders,
				models.GitHubProviderType: &mockProviders,
			}

			service := &service{
				vcsProviderMap:      providerMap,
				tharsisURL:          tharsisURL,
				repositorySizeLimit: 5000,
			}

			authURL, err := service.getOAuthAuthorizationURL(ctx, test.input)
			if test.expectedErrorCode != "" {
				assert.Equal(t, test.expectedErrorCode, errors.ErrorCode(err))
			} else if err != nil {
				t.Fatal(err)
			} else {
				assert.Equal(t, test.expectedURL, authURL)
			}
		})
	}
}

func TestProcessOAuth(t *testing.T) {
	sampleOAuthOldState, err := uuid.NewRandom()
	assert.Nil(t, err)

	testCases := []struct {
		caller                   auth.Caller
		input                    *ProcessOAuthInput
		existingProvider         *models.VCSProvider
		createAccessTokenPayload *types.AccessTokenPayload
		toUpdateInput            *models.VCSProvider
		name                     string
		expectedErrorCode        errors.CodeType
	}{
		{
			name:   "positive: provider returns both access and refresh token; expect both stored",
			caller: &auth.SystemCaller{},
			input: &ProcessOAuthInput{
				AuthorizationCode: "an-authorization-code",
				State:             sampleOAuthOldState.String(),
			},
			existingProvider: &models.VCSProvider{
				Type:              models.GitLabProviderType,
				URL:               sampleProviderURL,
				OAuthClientID:     "a-sample-client-id",
				OAuthClientSecret: "a-sample-client-secret",
				OAuthState:        ptr.String("sample-state"),
			},
			createAccessTokenPayload: &types.AccessTokenPayload{
				AccessToken:         sampleOAuthAccessToken,
				RefreshToken:        sampleOAuthRefreshToken,
				ExpirationTimestamp: &sampleOAuthAccessTokenExpirationTimestamp,
			},
			toUpdateInput: &models.VCSProvider{
				Type:                      models.GitLabProviderType,
				URL:                       sampleProviderURL,
				OAuthClientID:             "a-sample-client-id",
				OAuthClientSecret:         "a-sample-client-secret",
				OAuthAccessToken:          &sampleOAuthAccessToken,
				OAuthRefreshToken:         &sampleOAuthRefreshToken,
				OAuthAccessTokenExpiresAt: &sampleOAuthAccessTokenExpirationTimestamp,
			},
		},
		{
			name:   "positive: provider returns only access token; expect only access token stored",
			caller: &auth.SystemCaller{},
			input: &ProcessOAuthInput{
				AuthorizationCode: "an-authorization-code",
				State:             sampleOAuthOldState.String(),
			},
			existingProvider: &models.VCSProvider{
				Type:              models.GitLabProviderType,
				URL:               sampleProviderURL,
				OAuthClientID:     "a-sample-client-id",
				OAuthClientSecret: "a-sample-client-secret",
				OAuthState:        ptr.String("sample-state"),
			},
			createAccessTokenPayload: &types.AccessTokenPayload{
				AccessToken: sampleOAuthAccessToken,
				// No refresh token here.
			},
			toUpdateInput: &models.VCSProvider{
				Type:              models.GitLabProviderType,
				URL:               sampleProviderURL,
				OAuthClientID:     "a-sample-client-id",
				OAuthClientSecret: "a-sample-client-secret",
				OAuthAccessToken:  &sampleOAuthAccessToken,
			},
		},
		{
			name: "negative: without caller; expect error EUnauthorized",
			input: &ProcessOAuthInput{
				AuthorizationCode: "an-authorization-code",
				State:             sampleOAuthOldState.String(),
			},
			existingProvider:         &models.VCSProvider{},
			createAccessTokenPayload: &types.AccessTokenPayload{},
			expectedErrorCode:        errors.EUnauthorized,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := auth.WithCaller(context.Background(), test.caller)

			mockProviders := MockProvider{}
			mockVCSProviders := db.MockVCSProviders{}

			mockProviders.Test(t)
			mockVCSProviders.Test(t)

			createAccessTokenInput := &types.CreateAccessTokenInput{
				ProviderURL:       test.existingProvider.URL,
				ClientID:          test.existingProvider.OAuthClientID,
				ClientSecret:      test.existingProvider.OAuthClientSecret,
				AuthorizationCode: test.input.AuthorizationCode,
				RedirectURI:       vcsOAuthCallbackURL,
			}

			testConnectionInput := &types.TestConnectionInput{
				ProviderURL: test.existingProvider.URL,
				AccessToken: test.createAccessTokenPayload.AccessToken,
			}

			mockProviders.On("CreateAccessToken", mock.Anything, createAccessTokenInput).Return(test.createAccessTokenPayload, nil)
			mockProviders.On("TestConnection", mock.Anything, testConnectionInput).Return(nil)

			mockVCSProviders.On("GetProviderByOAuthState", mock.Anything, test.input.State).Return(test.existingProvider, nil)
			mockVCSProviders.On("UpdateProvider", mock.Anything, test.toUpdateInput).Return(&models.VCSProvider{}, nil)

			dbClient := &db.Client{
				VCSProviders: &mockVCSProviders,
			}

			providerMap := map[models.VCSProviderType]Provider{
				models.GitLabProviderType: &mockProviders,
				models.GitHubProviderType: &mockProviders,
			}

			service := newService(nil, dbClient, nil, nil, providerMap, nil, nil, nil, nil, nil, tharsisURL, 5000)

			err := service.ProcessOAuth(ctx, test.input)
			if test.expectedErrorCode != "" {
				assert.Equal(t, test.expectedErrorCode, errors.ErrorCode(err))
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func Test_handleEvent(t *testing.T) {
	ctx := context.Background()

	sampleWorkspace := &models.Workspace{
		FullPath: "path/to/workspace",
	}

	sampleDiffsPayload := &types.GetDiffsPayload{
		AlteredFiles: map[string]struct{}{
			"file.txt":    {},
			"changed.txt": {},
		},
	}

	createdCV := &models.ConfigurationVersion{
		Metadata: models.ResourceMetadata{
			ID: "cv-id",
		},
		Status: models.ConfigurationUploaded,
	}

	testCases := []struct {
		updatedCV       *models.ConfigurationVersion
		getDiffsPayload *types.GetDiffsPayload
		input           *handleEventInput
		name            string
		expectedError   string
	}{
		{
			name: "positive: valid branch push event, mostly empty link and provider setup; expect no errors",
			input: &handleEventInput{
				providerURL: sampleProviderURL,
				accessToken: "an-access-token",
				link: &models.WorkspaceVCSProviderLink{
					RepositoryPath:      "owner/repository",
					WorkspaceID:         "workspace-id",
					Branch:              "main", // Only allow events for main branch.
					AutoSpeculativePlan: false,
				},
				processInput: &ProcessWebhookEventInput{
					EventHeader: "push",                     // Corresponds to a GitHub push event.
					Before:      plumbing.ZeroHash.String(), // Represents an empty hash.
					After:       sampleAfterCommit,
					Ref:         "refs/heads/main", // Happening on the main branch.
				},
				workspace: sampleWorkspace,
				vcsEvent: &models.VCSEvent{
					Metadata: models.ResourceMetadata{
						ID: "event-id",
					},
					Type: models.BranchEventType,
				},
				repositorySizeLimit: 5000,
			},
			updatedCV:       createdCV,
			getDiffsPayload: sampleDiffsPayload,
		},
		{
			name: "positive: valid branch push event, mostly empty link and provider setup; expect no errors",
			input: &handleEventInput{
				providerURL: sampleProviderURL,
				accessToken: "an-access-token",
				link: &models.WorkspaceVCSProviderLink{
					RepositoryPath:      "owner/repository",
					WorkspaceID:         "workspace-id",
					Branch:              "main", // Only allow events for main branch.
					AutoSpeculativePlan: false,
				},
				processInput: &ProcessWebhookEventInput{
					EventHeader: "push", // Corresponds to a GitHub push event.
					Before:      sampleBeforeCommit,
					After:       sampleAfterCommit,
					Ref:         "refs/heads/main", // Happening on the main branch.
				},
				workspace: sampleWorkspace,
				vcsEvent: &models.VCSEvent{
					Metadata: models.ResourceMetadata{
						ID: "event-id",
					},
					Type: models.BranchEventType,
				},
				repositorySizeLimit: 5000,
			},
			updatedCV:       createdCV,
			getDiffsPayload: sampleDiffsPayload,
		},
		{
			name: "positive: valid tag event, no tag regex defined on link; expect no errors",
			input: &handleEventInput{
				providerURL: sampleProviderURL,
				accessToken: "an-access-token",
				link: &models.WorkspaceVCSProviderLink{
					RepositoryPath:      "owner/repository",
					WorkspaceID:         "workspace-id",
					Branch:              "main",
					AutoSpeculativePlan: false,
					// No tag regex, meaning no run.
				},
				processInput: &ProcessWebhookEventInput{
					EventHeader: "push", // Corresponds to a GitHub push event.
					Ref:         "refs/tags/v0.1",
				},
				workspace: sampleWorkspace,
				vcsEvent: &models.VCSEvent{
					Metadata: models.ResourceMetadata{
						ID: "event-id",
					},
					Type: models.TagEventType,
				},
				repositorySizeLimit: 5000,
			},
			updatedCV:       createdCV,
			getDiffsPayload: sampleDiffsPayload,
		},
		{
			name: "positive: valid tag event, with tag regex defined on link; expect no errors",
			input: &handleEventInput{
				providerURL: sampleProviderURL,
				accessToken: "an-access-token",
				link: &models.WorkspaceVCSProviderLink{
					RepositoryPath:      "owner/repository",
					WorkspaceID:         "workspace-id",
					Branch:              "main",
					AutoSpeculativePlan: false,
					TagRegex:            &sampleTagRegex,
				},
				processInput: &ProcessWebhookEventInput{
					EventHeader: "push", // Corresponds to a GitHub push event.
					Ref:         "refs/tags/v0.1",
				},
				workspace: sampleWorkspace,
				vcsEvent: &models.VCSEvent{
					Metadata: models.ResourceMetadata{
						ID: "event-id",
					},
					Type: models.TagEventType,
				},
				repositorySizeLimit: 5000,
			},
			updatedCV:       createdCV,
			getDiffsPayload: sampleDiffsPayload,
		},
		{
			name: "positive: valid PR event, auto-speculative is false on link; expect no errors",
			input: &handleEventInput{
				providerURL: sampleProviderURL,
				accessToken: "an-access-token",
				link: &models.WorkspaceVCSProviderLink{
					RepositoryPath:      "owner/repository",
					WorkspaceID:         "workspace-id",
					Branch:              "main",
					AutoSpeculativePlan: false, // No PR's allowed here.
					TagRegex:            &sampleTagRegex,
				},
				processInput: &ProcessWebhookEventInput{
					EventHeader:      "pull_request", // Corresponds to a GitHub PR event.
					SourceRepository: "owner/repository",
					SourceBranch:     "feature/branch",
					TargetBranch:     "main",
					Action:           "opened",
					HeadCommitID:     "some-commit-id",
				},
				workspace: sampleWorkspace,
				vcsEvent: &models.VCSEvent{
					Metadata: models.ResourceMetadata{
						ID: "event-id",
					},
					Type: models.MergeRequestEventType,
				},
				repositorySizeLimit: 5000,
			},
			updatedCV:       createdCV,
			getDiffsPayload: sampleDiffsPayload,
		},
		{
			name: "positive: valid PR event, auto-speculative is true on link; expect no errors",
			input: &handleEventInput{
				providerURL: sampleProviderURL,
				accessToken: "an-access-token",
				link: &models.WorkspaceVCSProviderLink{
					RepositoryPath:      "owner/repository",
					WorkspaceID:         "workspace-id",
					Branch:              "main",
					AutoSpeculativePlan: true, // PR's allowed here.
					TagRegex:            &sampleTagRegex,
				},
				processInput: &ProcessWebhookEventInput{
					EventHeader:      "pull_request", // Corresponds to a GitHub PR event.
					SourceRepository: "owner/repository",
					SourceBranch:     "feature/branch",
					TargetBranch:     "main",
					Action:           "opened",
					HeadCommitID:     "some-commit-id",
				},
				workspace: sampleWorkspace,
				vcsEvent: &models.VCSEvent{
					Metadata: models.ResourceMetadata{
						ID: "event-id",
					},
					Type: models.MergeRequestEventType,
				},
				repositorySizeLimit: 5000,
			},
			updatedCV:       createdCV,
			getDiffsPayload: sampleDiffsPayload,
		},
		{
			// Just to test the for-loop logic.
			name: "negative: configuration version failed to upload; expect error",
			input: &handleEventInput{
				providerURL: sampleProviderURL,
				accessToken: "an-access-token",
				link: &models.WorkspaceVCSProviderLink{
					RepositoryPath:      "owner/repository",
					WorkspaceID:         "workspace-id",
					Branch:              "main",
					AutoSpeculativePlan: true, // PR's allowed here.
					TagRegex:            &sampleTagRegex,
				},
				processInput: &ProcessWebhookEventInput{
					EventHeader:      "pull_request", // Corresponds to a GitHub PR event.
					SourceRepository: "owner/repository",
					SourceBranch:     "feature/branch",
					TargetBranch:     "main",
					Action:           "opened",
					HeadCommitID:     "some-commit-id",
				},
				workspace: sampleWorkspace,
				vcsEvent: &models.VCSEvent{
					Metadata: models.ResourceMetadata{
						ID: "event-id",
					},
					Type: models.MergeRequestEventType,
				},
				repositorySizeLimit: 5000,
			},
			updatedCV: &models.ConfigurationVersion{
				Status: models.ConfigurationErrored,
				// Other fields won't matter.
			},
			getDiffsPayload: sampleDiffsPayload,
			expectedError:   "failed to create and upload configuration version for repository owner/repository for workspace path/to/workspace and workspace vcs provider link ID : configuration upload failed; status is errored",
		},
		{
			// Run should still be created regardless.
			name: "negative: unable to get altered files; expect no errors",
			input: &handleEventInput{
				providerURL: sampleProviderURL,
				accessToken: "an-access-token",
				link: &models.WorkspaceVCSProviderLink{
					RepositoryPath:      "owner/repository",
					WorkspaceID:         "workspace-id",
					Branch:              "main",
					AutoSpeculativePlan: true, // PR's allowed here.
					TagRegex:            &sampleTagRegex,
				},
				processInput: &ProcessWebhookEventInput{
					EventHeader:      "pull_request", // Corresponds to a GitHub PR event.
					SourceRepository: "owner/repository",
					SourceBranch:     "feature/branch",
					TargetBranch:     "main",
					Action:           "opened",
					HeadCommitID:     "some-commit-id",
				},
				workspace: sampleWorkspace,
				vcsEvent: &models.VCSEvent{
					Metadata: models.ResourceMetadata{
						ID: "event-id",
					},
					Type: models.MergeRequestEventType,
				},
				repositorySizeLimit: 5000,
			},
			updatedCV: createdCV,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockProvider := MockProvider{}
			mockRunService := run.MockService{}
			mockWorkspaceService := workspace.MockService{}

			mockProvider.Test(t)
			mockRunService.Test(t)
			mockWorkspaceService.Test(t)

			runInput := &run.CreateRunInput{
				ConfigurationVersionID: &createdCV.Metadata.ID,
				WorkspaceID:            test.input.link.WorkspaceID,
			}

			cvInput := &workspace.CreateConfigurationVersionInput{
				WorkspaceID: test.input.link.WorkspaceID,
				Speculative: test.input.vcsEvent.Type.Equals(models.MergeRequestEventType),
				VCSEventID:  &test.input.vcsEvent.Metadata.ID,
			}

			getDiffsInput := &types.GetDiffsInput{
				ProviderURL:    test.input.providerURL,
				AccessToken:    test.input.accessToken,
				RepositoryPath: test.input.link.RepositoryPath,
				BaseRef:        test.input.processInput.Before,
				HeadRef:        test.input.processInput.After,
			}

			getDiffInput := &types.GetDiffInput{
				ProviderURL:    test.input.providerURL,
				AccessToken:    test.input.accessToken,
				RepositoryPath: test.input.link.RepositoryPath,
				Ref:            test.input.processInput.After,
			}

			if test.input.vcsEvent.Type.Equals(models.MergeRequestEventType) {
				getDiffInput.Ref = test.input.processInput.HeadCommitID
			}

			getArchiveInput := &types.GetArchiveInput{
				ProviderURL:    test.input.providerURL,
				AccessToken:    test.input.accessToken,
				RepositoryPath: test.input.link.RepositoryPath,
				Ref:            test.input.processInput.Ref,
			}

			if test.input.vcsEvent.Type.Equals(models.MergeRequestEventType) {
				getArchiveInput.Ref = test.input.processInput.SourceBranch
			}

			tarFile, err := createRepositoryArchive()
			require.Nil(t, err)
			defer tarFile.Close()
			defer os.Remove(tarFile.Name())

			sampleGetArchiveResponse := &http.Response{
				Body: io.NopCloser(tarFile),
			}

			mockProvider.On("GetArchive", mock.Anything, getArchiveInput).Return(sampleGetArchiveResponse, nil)
			mockProvider.On("GetDiffs", mock.Anything, getDiffsInput).Return(test.getDiffsPayload, nil)
			mockProvider.On("GetDiff", mock.Anything, getDiffInput).Return(test.getDiffsPayload, nil)

			mockRunService.On("CreateRun", mock.Anything, runInput).Return(&models.Run{}, nil)

			mockWorkspaceService.On("GetConfigurationVersionByID", mock.Anything, createdCV.Metadata.ID).Return(test.updatedCV, nil)
			mockWorkspaceService.On("CreateConfigurationVersion", mock.Anything, cvInput).Return(createdCV, nil)
			mockWorkspaceService.On("UploadConfigurationVersion", mock.Anything, createdCV.Metadata.ID, mock.Anything).Return(nil)

			// Update input with mocks.
			test.input.provider = &mockProvider
			logger, _ := logger.NewForTest()
			s := service{
				logger:              logger,
				dbClient:            nil,
				signingKeyManager:   nil,
				vcsProviderMap:      nil,
				activityService:     nil,
				runService:          &mockRunService,
				workspaceService:    &mockWorkspaceService,
				taskManager:         nil,
				oAuthStateGenerator: nil,
				tharsisURL:          "",
				repositorySizeLimit: 0,
			}

			err = s.handleEvent(ctx, test.input)
			if test.expectedError != "" {
				assert.Equal(t, test.expectedError, err.Error())
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}

// createRepositoryArchive creates a sample tar.gz file which is used
// as the GetArchive response payload.
func createRepositoryArchive() (*os.File, error) {
	// Create temp directory and file so we can test tar.gz functionality.
	// This will be used as a mock tar.gz archive returned by a vcs provider.
	parentDir, err := os.MkdirTemp("", "parent-test-dir")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(parentDir)

	// Create at least one file to be compressed otherwise,
	// decompression will fail later on.
	tempFile, err := os.CreateTemp(parentDir, "sample-file")
	if err != nil {
		return nil, err
	}
	defer tempFile.Close()

	// Make the tar.gz file.
	tarFilePath, err := makeModuleTar(parentDir)
	if err != nil {
		return nil, err
	}

	return os.Open(tarFilePath)
}
