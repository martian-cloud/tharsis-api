package resourcelimit

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/limits"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

func TestUpdateResourceLimit(t *testing.T) {
	userMemberID := "this-is-a-fake-user-member-ID"
	serviceAccountMemberID := "this is a fake-service-account-member-ID"
	serviceAccountPath := "this/is/a/fake/service/account/path"
	testBadInput := &UpdateResourceLimitInput{
		Name:  "not-a-valid-resource-limit-name",
		Value: 56,
	}

	testGoodInput := &UpdateResourceLimitInput{
		Name:  string(limits.ResourceLimitGroupTreeDepth),
		Value: 47,
	}

	testUpdatedInput := &UpdateResourceLimitInput{
		Name:  string(limits.ResourceLimitGroupTreeDepth),
		Value: 91,
	}

	testGoodLimit := &models.ResourceLimit{
		Name:  string(testGoodInput.Name),
		Value: testGoodInput.Value,
	}

	testUpdatedLimit := &models.ResourceLimit{
		Name:  string(testUpdatedInput.Name),
		Value: testUpdatedInput.Value,
	}

	type testCase struct {
		name            string
		callerType      string // "admin", "user", "service-account"
		input           *UpdateResourceLimitInput
		willGetLimit    bool
		injectOldLimit  *models.ResourceLimit
		injectNewLimit  *models.ResourceLimit
		expectLimit     *models.ResourceLimit
		expectErrorCode string
	}

	/*
		template test case:
		{
			name            string
			callerType      string // "admin", "user", "service-account"
			input           *UpdateResourceLimitInput
			willGetLimit    bool
			injectOldLimit  *models.ResourceLimit
			injectNewLimit  *models.ResourceLimit
			expectLimit     *models.ResourceLimit
			expectErrorCode string
		}
	*/

	// Test cases
	testCases := []testCase{
		{
			name:            "service account cannot create/update limits",
			callerType:      "service-account",
			input:           testGoodInput,
			expectErrorCode: errors.EForbidden,
		},
		{
			name:            "non-admin user cannot create/update limits",
			callerType:      "user",
			input:           testGoodInput,
			expectErrorCode: errors.EForbidden,
		},
		{
			name:            "admin user cannot create/update a limit with a bad name",
			callerType:      "admin",
			input:           testBadInput,
			willGetLimit:    true,
			injectOldLimit:  nil,
			expectErrorCode: errors.EInvalid,
		},
		// There is no such thing as creating a new limit in the DB, because they are all pre-populated.
		{
			name:           "admin user can update an existing limit",
			callerType:     "admin",
			input:          testUpdatedInput,
			willGetLimit:   true,
			injectOldLimit: testGoodLimit,
			injectNewLimit: testUpdatedLimit,
			expectLimit:    testUpdatedLimit,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockAuthorizer := auth.MockAuthorizer{}
			mockAuthorizer.Test(t)

			mockTransactions := db.NewMockTransactions(t)
			mockTransactions.Test(t)

			mockResourceLimits := db.NewMockResourceLimits(t)
			mockResourceLimits.Test(t)

			if test.willGetLimit {
				mockResourceLimits.On("GetResourceLimit", mock.Anything, test.input.Name).Return(test.injectOldLimit, nil)
			}

			if (test.expectErrorCode == "") && (test.injectOldLimit != nil) {
				// for the update existing limit case
				mockResourceLimits.On("UpdateResourceLimit", mock.Anything, mock.Anything).Return(test.injectNewLimit, nil)
			}

			dbClient := &db.Client{
				Transactions:   mockTransactions,
				ResourceLimits: mockResourceLimits,
			}

			var testCaller auth.Caller
			switch test.callerType {
			case "admin":
				testCaller = auth.NewUserCaller(
					&models.User{
						Metadata: models.ResourceMetadata{
							ID: userMemberID,
						},
						Admin:    true,
						Username: "user1",
					},
					&mockAuthorizer,
					dbClient,
				)
			case "user":
				testCaller = auth.NewUserCaller(
					&models.User{
						Metadata: models.ResourceMetadata{
							ID: userMemberID,
						},
						Admin:    false,
						Username: "user1",
					},
					&mockAuthorizer,
					dbClient,
				)
			case "service-account":
				testCaller = auth.NewServiceAccountCaller(
					serviceAccountMemberID,
					serviceAccountPath,
					&mockAuthorizer,
					dbClient,
				)
			default:
				assert.Fail(t, "invalid caller type in test")
			}

			logger, _ := logger.NewForTest()
			service := NewService(logger, dbClient)

			// Call the service function.
			actualOutput, actualError := service.UpdateResourceLimit(auth.WithCaller(ctx, testCaller), test.input)

			assert.Equal(t, (test.expectErrorCode == ""), (actualError == nil))
			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(actualError))
			}

			assert.Equal(t, test.expectLimit, actualOutput)
		})
	}
}
