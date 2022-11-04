package activityevent

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

type mockDBClient struct {
	*db.Client
	MockTransactions   *db.MockTransactions
	MockActivityEvents *db.MockActivityEvents
}

func buildDBClientWithMocks(t *testing.T) *mockDBClient {
	mockTransactions := db.MockTransactions{}
	mockTransactions.Test(t)

	mockActivityEvents := db.MockActivityEvents{}
	mockActivityEvents.Test(t)

	return &mockDBClient{
		Client: &db.Client{
			Transactions:   &mockTransactions,
			ActivityEvents: &mockActivityEvents,
		},
		MockTransactions:   &mockTransactions,
		MockActivityEvents: &mockActivityEvents,
	}
}

func TestGetActivityEvents(t *testing.T) {

	type testCase struct {
		name                    string
		caller                  string
		allowAllNamespacePolicy bool
	}

	// Test cases
	testCases := []testCase{
		{
			name:                    "verify membership filter is set when the namespace access policy does allow all namespaces",
			caller:                  "user",
			allowAllNamespacePolicy: true,
		},
		{
			name:                    "verify membership filter is not when the namespace access policy does not allow all namespaces",
			caller:                  "serviceAccount",
			allowAllNamespacePolicy: false,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			dbClient := buildDBClientWithMocks(t)

			mockAuthorizer := auth.MockAuthorizer{}
			mockAuthorizer.Test(t)

			mockAuthorizer.On("GetRootNamespaces", mock.Anything).Return([]models.MembershipNamespace{}, nil)

			var testCaller auth.Caller
			switch test.caller {
			case "user":
				testCaller = auth.NewUserCaller(
					&models.User{
						Metadata: models.ResourceMetadata{
							ID: "123",
						},
						Admin:    test.allowAllNamespacePolicy,
						Username: "user1",
					},
					&mockAuthorizer,
					dbClient.Client,
				)
			case "serviceAccount":
				testCaller = auth.NewServiceAccountCaller(
					"sa1",
					"groupA/sa1",
					&mockAuthorizer,
				)
			}

			dbClient.MockActivityEvents.On("GetActivityEvents", mock.Anything, mock.Anything).
				Return(func(_ context.Context, input *db.GetActivityEventsInput) *db.ActivityEventsResult {
					return &db.ActivityEventsResult{
						ActivityEvents: []models.ActivityEvent{},
					}
				}, nil)

			logger, _ := logger.NewForTest()
			service := NewService(dbClient.Client, logger)

			// Call the service function.
			actualOutput, actualError := service.GetActivityEvents(auth.WithCaller(ctx, testCaller), &GetActivityEventsInput{})
			if actualError != nil {
				t.Fatal(actualError)
			}

			assert.Equal(t, &db.ActivityEventsResult{
				ActivityEvents: []models.ActivityEvent{},
			}, actualOutput)

			dbClient.MockActivityEvents.AssertCalled(t, "GetActivityEvents", mock.Anything, mock.MatchedBy(func(input *db.GetActivityEventsInput) bool {
				if !test.allowAllNamespacePolicy {
					return input.Filter.NamespaceMembershipRequirement != nil
				}
				return true
			}))
		})
	}
}

func TestCreateActivityEvent(t *testing.T) {

	positiveActivityEventU := models.ActivityEvent{
		Metadata: models.ResourceMetadata{
			ID: "activityEvent-id-1", // okay that this is not a valid UUID
		},
		UserID:        ptr.String("user-id-1"),
		NamespacePath: ptr.String("namespace-path-u-1"),
		Action:        models.ActionCreate,
		TargetType:    models.TargetGroup,
		Payload:       fillPayload(map[string]string{"attributes": "u-1"}),
	}

	positiveActivityEventSA := models.ActivityEvent{
		Metadata: models.ResourceMetadata{
			ID: "activityEvent-id-1", // okay that this is not a valid UUID
		},
		ServiceAccountID: ptr.String("service-account-id-1"),
		NamespacePath:    ptr.String("namespace-path-sa-1"),
		Action:           models.ActionCreate,
		TargetType:       models.TargetGroup,
		Payload:          fillPayload(map[string]string{"attributes": "sa-1"}),
	}

	positiveCallerUser := models.User{
		Metadata: models.ResourceMetadata{
			ID: "test-caller-user-id-1",
		},
		Admin:    false,
		Username: "username-1",
	}

	invalidCallerUser := models.User{
		Metadata: models.ResourceMetadata{
			ID: "test-caller-invalid-user-id",
		},
		Admin:    false,
		Username: "invalid",
	}

	positiveCallerServiceAccount := models.ServiceAccount{
		Metadata: models.ResourceMetadata{
			ID: "test-caller-service-account-id-1",
		},
		ResourcePath: "service-account-1-full-path",
		Name:         "service-account-name-1",
		Description:  "This is test service account 1.",
	}

	invalidCallerServiceAccount := models.ServiceAccount{
		Metadata: models.ResourceMetadata{
			ID: "test-caller-invalid-service-account-id",
		},
		ResourcePath: "invalid-service-account-full-path",
		Name:         "invalid",
		Description:  "This is invalid test service account 1.",
	}

	negativeError := fmt.Errorf("this is a negative error")
	fakeErrorInvalidUsername := fmt.Errorf("fake error: invalid username")
	fakeErrorInvalidServiceAccountName := fmt.Errorf("fake error: invalid service account name")
	fakeErrorInvalidNamespace := fmt.Errorf("fake error: invalid namespace path")
	fakeErrorNoPermission := fmt.Errorf("fake error: user has no permission in namespace")

	// Start mocking out the necessary functions.
	mockAuthorizer := auth.MockAuthorizer{}
	mockAuthorizer.Test(t)

	mockAuthorizer.On("RequireAccessToNamespace", mock.Anything, "namespace-path-u-1", models.DeployerRole).
		Return(nil)
	mockAuthorizer.On("RequireAccessToNamespace", mock.Anything, "namespace-path-sa-1", models.DeployerRole).
		Return(nil)
	mockAuthorizer.On("RequireAccessToNamespace", mock.Anything, "invalid", models.DeployerRole).
		Return(nil)
	mockAuthorizer.On("RequireAccessToNamespace", mock.Anything, "namespace-path-2", models.DeployerRole).
		Return(fakeErrorNoPermission)

	type testCase struct {
		expectError          error
		directError          error
		callerUser           *models.User
		callerServiceAccount *models.ServiceAccount
		directOutput         *models.ActivityEvent
		input                *models.ActivityEvent
		expectOutput         *models.ActivityEvent
		name                 string
		injectID             string
	}

	/*
		template test case:
		{
			name                 string
			input                *models.ActivityEvent
			callerUser           *models.User
			callerServiceAccount *models.ServiceAccount
			directOutput         *models.ActivityEvent
			directError          error
			injectID             string
			expectError          error
			expectOutput         *models.ActivityEvent
		}
	*/

	// Test cases
	testCases := []testCase{

		{
			name:         "direct pass-through, something",
			input:        &positiveActivityEventU,
			callerUser:   &positiveCallerUser,
			directOutput: &positiveActivityEventU,
			directError:  nil,
			expectOutput: &positiveActivityEventU,
			expectError:  nil,
		},

		{
			name:         "direct pass-through, error",
			input:        &positiveActivityEventU,
			callerUser:   &positiveCallerUser,
			directOutput: nil,
			directError:  negativeError,
			expectOutput: nil,
			expectError:  negativeError,
		},

		{
			name:       "not direct, basic positive, user caller",
			input:      &positiveActivityEventU,
			callerUser: &positiveCallerUser,
			injectID:   "activityEvent-id-1",
			expectOutput: replaceFields(positiveActivityEventU, models.ActivityEvent{
				UserID:  &positiveCallerUser.Metadata.ID,
				Payload: fillExpectedPayload(positiveActivityEventU.Payload),
			}),
			expectError: nil,
		},

		{
			name:                 "not direct, basic positive, service account caller",
			input:                &positiveActivityEventSA,
			callerServiceAccount: &positiveCallerServiceAccount,
			injectID:             "activityEvent-id-1",
			expectOutput: replaceFields(positiveActivityEventSA, models.ActivityEvent{
				ServiceAccountID: &positiveCallerServiceAccount.Metadata.ID,
				Payload:          fillExpectedPayload(positiveActivityEventSA.Payload),
			}),
			expectError: nil,
		},

		{
			name:         "not direct, negative, invalid user",
			input:        &positiveActivityEventU,
			callerUser:   &invalidCallerUser,
			expectOutput: nil,
			expectError:  fakeErrorInvalidUsername,
		},

		{
			name:                 "not direct, negative, invalid service account",
			input:                &positiveActivityEventSA,
			callerServiceAccount: &invalidCallerServiceAccount,
			expectOutput:         nil,
			expectError:          fakeErrorInvalidServiceAccountName,
		},

		{
			name: "not direct, negative, invalid namespace path",
			input: replaceFields(positiveActivityEventU, models.ActivityEvent{
				NamespacePath: ptr.String("invalid"),
			}),
			callerUser:   &positiveCallerUser,
			expectOutput: nil,
			expectError:  fakeErrorInvalidNamespace,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			dbClient := buildDBClientWithMocks(t)

			var testCaller auth.Caller
			switch {
			case test.callerUser != nil:
				newUserCaller := auth.NewUserCaller(test.callerUser, &mockAuthorizer, dbClient.Client)
				testCaller = auth.Caller(newUserCaller)
			case test.callerServiceAccount != nil:
				newServiceAccountCaller := auth.NewServiceAccountCaller(test.callerServiceAccount.Metadata.ID,
					test.callerServiceAccount.ResourcePath, &mockAuthorizer)
				testCaller = auth.Caller(newServiceAccountCaller)
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			dbClient.MockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
			dbClient.MockTransactions.On("CommitTx", mock.Anything).Return(nil)

			// mockFunc returns something.
			// If either direct result or direct error are supplied, it returns those.
			// Otherwise, it returns a result based on whether any fields are "invalid".
			mockFunc := func(ctx context.Context, input *models.ActivityEvent) (*models.ActivityEvent, error) {
				_ = ctx

				if (test.directOutput != nil) || (test.directError != nil) {
					return test.directOutput, test.directError
				}

				switch {
				case (input.UserID != nil) && (*input.UserID == "test-caller-invalid-user-id"):
					return nil, fakeErrorInvalidUsername
				case (input.ServiceAccountID != nil) && (*input.ServiceAccountID == "test-caller-invalid-service-account-id"):
					return nil, fakeErrorInvalidServiceAccountName
				case (input.NamespacePath != nil) && (*input.NamespacePath == "invalid"):
					return nil, fakeErrorInvalidNamespace
				default:
					input.Metadata.ID = test.injectID
					return input, nil
				}
			}

			mockFunc0 := func(ctx context.Context, input *models.ActivityEvent) *models.ActivityEvent {
				result, _ := mockFunc(ctx, input)
				return result
			}

			mockFunc1 := func(ctx context.Context, input *models.ActivityEvent) error {
				_, err := mockFunc(ctx, input)
				return err
			}

			dbClient.MockActivityEvents.On("CreateActivityEvent", mock.Anything, mock.Anything).
				Return(mockFunc0, mockFunc1)

			logger, _ := logger.NewForTest()
			service := NewService(dbClient.Client, logger)

			// Call the service function.
			toCreate := CreateActivityEventInput{
				NamespacePath: test.input.NamespacePath,
				Action:        test.input.Action,
				TargetType:    test.input.TargetType,
				Payload:       test.input.Payload,
			}

			actualOutput, actualError := service.CreateActivityEvent(auth.WithCaller(ctx, testCaller), &toCreate)

			assert.Equal(t, test.expectError, actualError)
			assert.Equal(t, test.expectOutput, actualOutput)
		})
	}
}

//////////////////////////////////////////////////////////////////////////////

// fillExpectedPayload returns a byte slice with double quotation marks around a base64-encoded
// version of the input.
func fillExpectedPayload(input []byte) []byte {
	result := []byte{'"'}
	result = append(result, []byte(base64.StdEncoding.EncodeToString(input))...)
	result = append(result, '"')
	return result
}

// fillPayload returns a byte slice to put in a payload field
// for purposes of the tests in this module, the input is intended to be
// either a string-to-string map or a db.GetActivityEventsInput.
func fillPayload(input interface{}) []byte {

	serializedInput, err := json.Marshal(input)
	if err != nil {
		// This error should not be expected.
		serializedInput = []byte(err.Error())
	}

	return serializedInput
}

// replaceFields returns a modified copy of an activity event with one or more fields replaced.
func replaceFields(old, delta models.ActivityEvent) *models.ActivityEvent {
	result := old

	if delta.Metadata.ID != "" {
		result.Metadata.ID = delta.Metadata.ID
	}
	if delta.Metadata.Version != 0 {
		result.Metadata.Version = delta.Metadata.Version
	}
	if delta.UserID != nil {
		result.UserID = delta.UserID
	}
	if delta.ServiceAccountID != nil {
		result.ServiceAccountID = delta.ServiceAccountID
	}
	if delta.NamespacePath != nil {
		result.NamespacePath = delta.NamespacePath
	}

	if delta.Action != "" {
		result.Action = delta.Action
	}

	if delta.TargetType != "" {
		result.TargetType = delta.TargetType
	}

	if len(delta.Payload) != 0 {
		result.Payload = delta.Payload
	}

	return &result
}

// The End.
