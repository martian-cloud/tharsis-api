package runner

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth/permissions"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/events"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/limits"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/logstream"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

func TestGetRunnerByID(t *testing.T) {
	runnerID := "runner-1"
	groupID := "group-1"
	// Test cases
	tests := []struct {
		expectRunner  *models.Runner
		name          string
		authError     error
		expectErrCode errors.CodeType
	}{
		{
			name: "all users can view shared runner",
			expectRunner: &models.Runner{
				Metadata: models.ResourceMetadata{ID: runnerID},
				Name:     "test-runner",
				Type:     models.SharedRunnerType,
			},
		},
		{
			name: "get group runner",
			expectRunner: &models.Runner{
				Metadata: models.ResourceMetadata{ID: runnerID},
				GroupID:  &groupID,
				Name:     "test-runner",
				Type:     models.GroupRunnerType,
			},
		},
		{
			name: "subject does not have access to group runner",
			expectRunner: &models.Runner{
				Metadata: models.ResourceMetadata{ID: runnerID},
				GroupID:  &groupID,
				Name:     "test-runner",
				Type:     models.GroupRunnerType,
			},
			authError:     errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrCode: errors.EForbidden,
		},
		{
			name:          "runner not found",
			expectErrCode: errors.ENotFound,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)

			if test.expectRunner != nil && test.expectRunner.GroupID != nil {
				mockCaller.On("RequireAccessToInheritableResource",
					mock.Anything, permissions.RunnerResourceType, mock.Anything, mock.Anything).
					Return(test.authError)
			}

			mockRunners := db.NewMockRunners(t)

			mockRunners.On("GetRunnerByID", mock.Anything, runnerID).Return(test.expectRunner, nil)

			dbClient := db.Client{
				Runners: mockRunners,
			}

			testLogger, _ := logger.NewForTest()

			service := NewService(testLogger, &dbClient, nil, nil, nil, nil)

			runner, err := service.GetRunnerByID(auth.WithCaller(ctx, mockCaller), runnerID)

			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectRunner, runner)
		})
	}
}

func TestGetRunnerByPath(t *testing.T) {
	path := "group-1/runner-1"
	runnerID := "runner-1"
	groupID := "group-1"
	// Test cases
	tests := []struct {
		expectRunner  *models.Runner
		name          string
		authError     error
		expectErrCode errors.CodeType
	}{
		{
			name: "all users can get a shared runner",
			expectRunner: &models.Runner{
				Metadata: models.ResourceMetadata{ID: runnerID},
				Name:     "test-runner",
				Type:     models.SharedRunnerType,
			},
		},
		{
			name: "get group runner",
			expectRunner: &models.Runner{
				Metadata: models.ResourceMetadata{ID: runnerID},
				GroupID:  &groupID,
				Name:     "test-runner",
				Type:     models.GroupRunnerType,
			},
		},
		{
			name: "subject does not have access to group runner",
			expectRunner: &models.Runner{
				Metadata: models.ResourceMetadata{ID: runnerID},
				GroupID:  &groupID,
				Name:     "test-runner",
				Type:     models.GroupRunnerType,
			},
			authError:     errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrCode: errors.EForbidden,
		},
		{
			name:          "runner not found",
			expectErrCode: errors.ENotFound,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)

			if test.expectRunner != nil && test.expectRunner.GroupID != nil {
				mockCaller.On("RequireAccessToInheritableResource",
					mock.Anything, permissions.RunnerResourceType, mock.Anything, mock.Anything).
					Return(test.authError)
			}

			mockRunners := db.NewMockRunners(t)

			mockRunners.On("GetRunnerByPath", mock.Anything, path).Return(test.expectRunner, nil)

			dbClient := db.Client{
				Runners: mockRunners,
			}

			testLogger, _ := logger.NewForTest()

			service := NewService(testLogger, &dbClient, nil, nil, nil, nil)

			runner, err := service.GetRunnerByPath(auth.WithCaller(ctx, mockCaller), path)

			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectRunner, runner)
		})
	}
}

func TestGetRunnersByIDs(t *testing.T) {
	runnerID := "runner-1"
	groupID := "group-1"
	// Test cases
	tests := []struct {
		expectRunner  *models.Runner
		name          string
		authError     error
		expectErrCode errors.CodeType
	}{
		{
			name: "all users can get a shared runner",
			expectRunner: &models.Runner{
				Metadata: models.ResourceMetadata{ID: runnerID},
				Name:     "test-runner",
				Type:     models.SharedRunnerType,
			},
		},
		{
			name: "get group runner",
			expectRunner: &models.Runner{
				Metadata:     models.ResourceMetadata{ID: runnerID},
				GroupID:      &groupID,
				Name:         "test-runner",
				ResourcePath: "some-group/test-runner",
				Type:         models.GroupRunnerType,
			},
		},
		{
			name: "subject does not have access to group runner",
			expectRunner: &models.Runner{
				Metadata:     models.ResourceMetadata{ID: runnerID},
				GroupID:      &groupID,
				Name:         "test-runner",
				ResourcePath: "some-group/test-runner",
				Type:         models.GroupRunnerType,
			},
			authError:     errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrCode: errors.EForbidden,
		},
		{
			name: "runner not found",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)

			if test.expectRunner != nil && test.expectRunner.GroupID != nil {
				mockCaller.On("RequireAccessToInheritableResource",
					mock.Anything, permissions.RunnerResourceType, mock.Anything, mock.Anything).
					Return(test.authError)
			}

			mockRunners := db.NewMockRunners(t)

			getRunnersResponse := db.RunnersResult{
				Runners: []models.Runner{},
			}

			if test.expectRunner != nil {
				getRunnersResponse.Runners = append(getRunnersResponse.Runners, *test.expectRunner)
			}

			mockRunners.On("GetRunners", mock.Anything, &db.GetRunnersInput{
				Filter: &db.RunnerFilter{
					RunnerIDs: []string{runnerID},
				},
			}).Return(&getRunnersResponse, nil)

			dbClient := db.Client{
				Runners: mockRunners,
			}

			testLogger, _ := logger.NewForTest()

			service := NewService(testLogger, &dbClient, nil, nil, nil, nil)

			runners, err := service.GetRunnersByIDs(auth.WithCaller(ctx, mockCaller), []string{runnerID})

			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			if test.expectRunner != nil {
				assert.Equal(t, 1, len(runners))
				assert.Equal(t, test.expectRunner, &runners[0])
			} else {
				assert.Equal(t, 0, len(runners))
			}
		})
	}
}

func TestGetRunners(t *testing.T) {
	runnerID := "runner-1"
	groupID := "group-1"

	// Allow a pointer to a constant.
	localSharedRunnerType := models.SharedRunnerType
	localGroupRunnerType := models.GroupRunnerType

	// Test cases
	tests := []struct {
		input         *GetRunnersInput
		expectRunner  *models.Runner
		name          string
		authError     error
		expectErrCode errors.CodeType
		isAdmin       bool
	}{
		{
			name: "filter runners by group and allow access",
			input: &GetRunnersInput{
				NamespacePath: &groupID,
			},
			expectRunner: &models.Runner{
				Type:     models.GroupRunnerType,
				Metadata: models.ResourceMetadata{ID: runnerID},
				GroupID:  &groupID,
				Name:     "test-runner",
			},
		},
		{
			name: "filter runners by shared type, admin user",
			input: &GetRunnersInput{
				RunnerType: &localSharedRunnerType,
			},
			isAdmin: true,
			expectRunner: &models.Runner{
				Type:     models.SharedRunnerType,
				Metadata: models.ResourceMetadata{ID: runnerID},
				Name:     "shared-test-runner",
			},
		},
		{
			name: "filter runners by group type, admin user",
			input: &GetRunnersInput{
				RunnerType: &localGroupRunnerType,
			},
			isAdmin: true,
			expectRunner: &models.Runner{
				Type:     models.GroupRunnerType,
				Metadata: models.ResourceMetadata{ID: runnerID},
				GroupID:  &groupID,
				Name:     "group-test-runner",
			},
		},
		{
			name: "filter runners by shared type, non-admin user",
			input: &GetRunnersInput{
				RunnerType: &localSharedRunnerType,
			},
		},
		{
			name: "filter runners by group type, non-admin user",
			input: &GetRunnersInput{
				RunnerType: &localGroupRunnerType,
			},
			expectErrCode: errors.EInvalid,
		},
		{
			name: "subject does not have viewer role for group",
			input: &GetRunnersInput{
				NamespacePath: &groupID,
			},
			authError:     errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrCode: errors.EForbidden,
		},
		{
			name: "no runners matching filters",
			input: &GetRunnersInput{
				NamespacePath: &groupID,
			},
		},
		{
			name:          "non admin cannot view all runners",
			input:         &GetRunnersInput{},
			expectErrCode: errors.EForbidden,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)

			mockCaller.On("RequirePermission", mock.Anything, permissions.ViewRunnerPermission, mock.Anything).
				Return(test.authError).Maybe()
			mockCaller.On("IsAdmin").Return(test.isAdmin).Maybe()

			mockRunners := db.NewMockRunners(t)

			getRunnersResponse := db.RunnersResult{
				Runners: []models.Runner{},
			}

			if test.expectRunner != nil {
				getRunnersResponse.Runners = append(getRunnersResponse.Runners, *test.expectRunner)
			}

			var filterNamespacePaths []string
			if test.input.NamespacePath != nil {
				filterNamespacePaths = []string{*test.input.NamespacePath}
			}
			mockRunners.On("GetRunners", mock.Anything, &db.GetRunnersInput{
				Sort:              test.input.Sort,
				PaginationOptions: test.input.PaginationOptions,
				Filter: &db.RunnerFilter{
					NamespacePaths: filterNamespacePaths,
					RunnerType:     test.input.RunnerType,
				},
			}).Return(&getRunnersResponse, nil).Maybe()

			dbClient := db.Client{
				Runners: mockRunners,
			}

			testLogger, _ := logger.NewForTest()

			service := NewService(testLogger, &dbClient, nil, nil, nil, nil)

			resp, err := service.GetRunners(auth.WithCaller(ctx, mockCaller), test.input)

			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			if test.expectRunner != nil {
				assert.Equal(t, 1, len(resp.Runners))
				assert.Equal(t, test.expectRunner, &resp.Runners[0])
			} else {
				assert.Equal(t, 0, len(resp.Runners))
			}
		})
	}
}

func TestCreateRunner(t *testing.T) {
	groupID := "group123"

	// Test cases
	tests := []struct {
		authError             error
		expectCreatedRunner   *models.Runner
		name                  string
		expectErrCode         errors.CodeType
		input                 CreateRunnerInput
		limit                 int
		injectRunnersPerGroup int32
		exceedsLimit          bool
		isAdmin               bool
	}{
		{
			name: "create group runner",
			input: CreateRunnerInput{
				Name:            "test-runner",
				GroupID:         groupID,
				Tags:            []string{"some-tag"},
				RunUntaggedJobs: false,
			},
			expectCreatedRunner: &models.Runner{
				Type:            models.GroupRunnerType,
				Name:            "test-runner",
				GroupID:         &groupID,
				CreatedBy:       "mockSubject",
				ResourcePath:    "group-1/test-runner",
				Tags:            []string{"some-tag"},
				RunUntaggedJobs: false,
			},
			limit:                 5,
			injectRunnersPerGroup: 5,
		},
		{
			name: "subject does not have owner role",
			input: CreateRunnerInput{
				Name:    "test-runner",
				GroupID: groupID,
			},
			authError:     errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrCode: errors.EForbidden,
		},
		{
			name: "exceeds limit",
			input: CreateRunnerInput{
				Name:    "test-runner",
				GroupID: groupID,
				Tags:    []string{"some-tag"},
			},
			expectCreatedRunner: &models.Runner{
				Type:         models.GroupRunnerType,
				Name:         "test-runner",
				GroupID:      &groupID,
				CreatedBy:    "mockSubject",
				ResourcePath: "group-1/test-runner",
				Tags:         []string{"some-tag"},
			},
			limit:                 5,
			injectRunnersPerGroup: 6,
			exceedsLimit:          true,
			expectErrCode:         errors.EInvalid,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.MockCaller{}
			mockCaller.Test(t)

			mockCaller.On("RequirePermission", mock.Anything, permissions.CreateRunnerPermission, mock.Anything).Return(test.authError)
			mockCaller.On("IsAdmin").Return(test.isAdmin).Maybe()

			mockCaller.On("GetSubject").Return("mockSubject")

			mockTransactions := db.NewMockTransactions(t)
			mockRunners := db.NewMockRunners(t)
			mockResourceLimits := db.NewMockResourceLimits(t)

			if test.authError == nil {
				mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil).Maybe()
				mockTransactions.On("RollbackTx", mock.Anything).Return(nil).Maybe()
				if !test.exceedsLimit {
					mockTransactions.On("CommitTx", mock.Anything).Return(nil).Maybe()
				}
			}

			if test.expectCreatedRunner != nil {
				mockRunners.On("CreateRunner", mock.Anything, mock.Anything).
					Return(test.expectCreatedRunner, nil)
			}

			dbClient := db.Client{
				Transactions:   mockTransactions,
				Runners:        mockRunners,
				ResourceLimits: mockResourceLimits,
			}

			mockActivityEvents := activityevent.NewMockService(t)

			if test.authError == nil && !test.exceedsLimit {
				mockActivityEvents.On("CreateActivityEvent", mock.Anything, mock.Anything).
					Return(&models.ActivityEvent{}, nil).Maybe()
			}

			// Called inside transaction to check resource limits.
			if test.limit > 0 {
				mockRunners.On("GetRunners", mock.Anything, mock.Anything).Return(&db.GetRunnersInput{
					Filter: &db.RunnerFilter{
						GroupID: &groupID,
					},
					PaginationOptions: &pagination.Options{
						First: ptr.Int32(0),
					},
				}).Return(func(ctx context.Context, input *db.GetRunnersInput) *db.RunnersResult {
					_ = ctx
					_ = input

					return &db.RunnersResult{
						PageInfo: &pagination.PageInfo{
							TotalCount: test.injectRunnersPerGroup,
						},
					}
				}, nil)

				mockResourceLimits.On("GetResourceLimit", mock.Anything, mock.Anything).
					Return(&models.ResourceLimit{Value: test.limit}, nil)
			}

			testLogger, _ := logger.NewForTest()

			service := NewService(testLogger, &dbClient, limits.NewLimitChecker(&dbClient), mockActivityEvents, nil, nil)

			runner, err := service.CreateRunner(auth.WithCaller(ctx, &mockCaller), &test.input)
			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectCreatedRunner, runner)
		})
	}
}

func TestUpdateRunner(t *testing.T) {
	groupID := "group123"

	// Test cases
	tests := []struct {
		authError     error
		input         *models.Runner
		name          string
		expectErrCode errors.CodeType
	}{
		{
			name: "update runner",
			input: &models.Runner{
				Type:            models.GroupRunnerType,
				Name:            "test-runner",
				GroupID:         &groupID,
				ResourcePath:    "group123/test-runner",
				Disabled:        false,
				Tags:            []string{"some-tag"},
				RunUntaggedJobs: true,
			},
		},
		{
			name: "subject does not have owner role",
			input: &models.Runner{
				Type:            models.GroupRunnerType,
				Name:            "test-runner",
				GroupID:         &groupID,
				ResourcePath:    "group123/test-runner",
				Disabled:        false,
				Tags:            []string{"some-tag"},
				RunUntaggedJobs: true,
			},
			authError:     errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrCode: errors.EForbidden,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.MockCaller{}
			mockCaller.Test(t)

			mockCaller.On("RequirePermission", mock.Anything, permissions.UpdateRunnerPermission, mock.Anything).Return(test.authError)

			mockCaller.On("GetSubject").Return("mockSubject")

			mockTransactions := db.NewMockTransactions(t)
			mockRunners := db.NewMockRunners(t)

			if test.authError == nil {
				mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
				mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
				mockTransactions.On("CommitTx", mock.Anything).Return(nil)

				mockRunners.On("UpdateRunner", mock.Anything, test.input).
					Return(test.input, nil)
			}

			dbClient := db.Client{
				Transactions: mockTransactions,
				Runners:      mockRunners,
			}

			mockActivityEvents := activityevent.NewMockService(t)

			if test.authError == nil {
				mockActivityEvents.On("CreateActivityEvent", mock.Anything, mock.Anything).Return(&models.ActivityEvent{}, nil)
			}

			testLogger, _ := logger.NewForTest()

			service := NewService(testLogger, &dbClient, nil, mockActivityEvents, nil, nil)

			runner, err := service.UpdateRunner(auth.WithCaller(ctx, &mockCaller), test.input)
			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.input, runner)
		})
	}
}

func TestDeleteRunner(t *testing.T) {
	groupID := "group123"

	// Test cases
	tests := []struct {
		authError     error
		input         *models.Runner
		name          string
		expectErrCode errors.CodeType
	}{
		{
			name: "delete runner",
			input: &models.Runner{
				Type:         models.GroupRunnerType,
				Name:         "test-runner",
				GroupID:      &groupID,
				ResourcePath: "group123/test-runner",
				Disabled:     false,
			},
		},
		{
			name: "subject does not have deployer role",
			input: &models.Runner{
				Type:         models.GroupRunnerType,
				Name:         "test-runner",
				GroupID:      &groupID,
				ResourcePath: "group123/test-runner",
				Disabled:     false,
			},
			authError:     errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrCode: errors.EForbidden,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.MockCaller{}
			mockCaller.Test(t)

			mockCaller.On("RequirePermission", mock.Anything, permissions.DeleteRunnerPermission, mock.Anything).Return(test.authError)

			mockCaller.On("GetSubject").Return("mockSubject")

			mockTransactions := db.NewMockTransactions(t)
			mockRunners := db.NewMockRunners(t)

			if test.authError == nil {
				mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
				mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
				mockTransactions.On("CommitTx", mock.Anything).Return(nil)

				mockRunners.On("DeleteRunner", mock.Anything, test.input).
					Return(nil)
			}

			dbClient := db.Client{
				Transactions: mockTransactions,
				Runners:      mockRunners,
			}

			mockActivityEvents := activityevent.NewMockService(t)

			if test.authError == nil {
				mockActivityEvents.On("CreateActivityEvent", mock.Anything, mock.Anything).Return(&models.ActivityEvent{}, nil)
			}

			testLogger, _ := logger.NewForTest()

			service := NewService(testLogger, &dbClient, nil, mockActivityEvents, nil, nil)

			err := service.DeleteRunner(auth.WithCaller(ctx, &mockCaller), test.input)
			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestAssignServiceAccountToRunner(t *testing.T) {
	groupID := "group123"

	type testCase struct {
		name           string
		runner         *models.Runner
		serviceAccount *models.ServiceAccount
		authError      error
		expectErrCode  errors.CodeType
		isAdmin        bool
	}

	testCases := []testCase{
		{
			name: "successfully assign group service account to group runner",
			runner: &models.Runner{
				GroupID:      &groupID,
				ResourcePath: "group123/runner-1",
				Type:         models.GroupRunnerType,
			},
			serviceAccount: &models.ServiceAccount{
				GroupID:      groupID,
				ResourcePath: "group123/sa-1",
			},
		},
		{
			name: "cannot assign global service account to shared runner", // unlike in Phobos, where this is allowed
			runner: &models.Runner{
				Type: models.SharedRunnerType,
			},
			serviceAccount: &models.ServiceAccount{
				ResourcePath: "global/sa-1",
			},
			isAdmin:       true,
			expectErrCode: errors.EInvalid,
		},
		{
			name: "cannot assign group service account to shared runner",
			runner: &models.Runner{
				Type: models.SharedRunnerType,
			},
			serviceAccount: &models.ServiceAccount{
				GroupID:      groupID,
				ResourcePath: "group123/sa-1",
			},
			expectErrCode: errors.EInvalid,
		},
		{
			name: "cannot assign global service account to group runner",
			runner: &models.Runner{
				GroupID:      &groupID,
				ResourcePath: "group123/runner-1",
				Type:         models.GroupRunnerType,
			},
			serviceAccount: &models.ServiceAccount{
				ResourcePath: "global/sa-1",
			},
			expectErrCode: errors.EInvalid,
		},
		{
			name: "group service account and group runner are not in the same group",
			runner: &models.Runner{
				GroupID:      &groupID,
				ResourcePath: "group123/runner-1",
				Type:         models.GroupRunnerType,
			},
			serviceAccount: &models.ServiceAccount{
				GroupID:      "group456",
				ResourcePath: "group456/sa-1",
			},
			expectErrCode: errors.EInvalid,
		},
		{
			name:          "runner not found",
			expectErrCode: errors.ENotFound,
		},
		{
			name: "service account not found",
			runner: &models.Runner{
				GroupID:      &groupID,
				ResourcePath: "group123/runner-1",
				Type:         models.GroupRunnerType,
			},
			expectErrCode: errors.ENotFound,
		},
		{
			name: "global service account cannot be assigned to a shared runner, since caller is not an admin",
			runner: &models.Runner{
				Type: models.SharedRunnerType,
			},
			serviceAccount: &models.ServiceAccount{
				ResourcePath: "global/sa-1",
			},
			expectErrCode: errors.EInvalid, // unlike in Phobos, where this is forbidden
		},
		{
			name: "subject does not have permissions to assign a group service account to a group runner",
			runner: &models.Runner{
				GroupID:      &groupID,
				ResourcePath: "group123/runner-1",
				Type:         models.GroupRunnerType,
			},
			serviceAccount: &models.ServiceAccount{
				GroupID:      groupID,
				ResourcePath: "group123/sa-1",
			},
			authError:     errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrCode: errors.EForbidden,
		},
		{
			name: "global service account cannot be assigned to a shared runner, since caller is not a user",
			runner: &models.Runner{
				Type: models.SharedRunnerType,
			},
			serviceAccount: &models.ServiceAccount{
				ResourcePath: "global/sa-1",
			},
			expectErrCode: errors.EInvalid, // unlike in Phobos, where this is forbidden
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockRunners := db.NewMockRunners(t)
			mockServiceAccounts := db.NewMockServiceAccounts(t)

			mockRunners.On("GetRunnerByID", mock.Anything, mock.Anything).Return(test.runner, nil)

			if test.runner != nil {
				mockServiceAccounts.On("GetServiceAccountByID", mock.Anything, mock.Anything).
					Return(test.serviceAccount, nil).Maybe()
			}

			mockCaller.On("RequirePermission", mock.Anything, permissions.UpdateRunnerPermission, mock.Anything).Return(test.authError).Maybe()
			mockCaller.On("IsAdmin").Return(test.isAdmin).Maybe()

			if test.expectErrCode == "" {
				mockServiceAccounts.On("AssignServiceAccountToRunner", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			}

			dbClient := &db.Client{
				Runners:         mockRunners,
				ServiceAccounts: mockServiceAccounts,
			}

			service := &service{
				dbClient: dbClient,
			}

			err := service.AssignServiceAccountToRunner(auth.WithCaller(ctx, mockCaller), "sa-1", "runner-1")

			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestUnassignServiceAccountFromRunner(t *testing.T) {
	groupID := "group123"

	type testCase struct {
		name          string
		runner        *models.Runner
		authError     error
		expectErrCode errors.CodeType
		isAdmin       bool
	}

	testCases := []testCase{
		{
			name: "successfully unassign service account from shared runner",
			runner: &models.Runner{
				Type: models.SharedRunnerType,
			},
			isAdmin:       true,
			expectErrCode: errors.EInvalid, // unlike Phobos, where this is allowed
		},
		{
			name: "successfully unassign service account from group runner",
			runner: &models.Runner{
				GroupID: &groupID,
				Type:    models.GroupRunnerType,
			},
		},
		{
			name: "subject does not have permissions to unassign a service account from a group runner",
			runner: &models.Runner{
				GroupID: &groupID,
				Type:    models.GroupRunnerType,
			},
			authError:     errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrCode: errors.EForbidden,
		},
		{
			name:          "runner not found",
			expectErrCode: errors.ENotFound,
		},
		{
			name: "global service account cannot be unassigned from a shared runner, since caller is not an admin",
			runner: &models.Runner{
				Type: models.SharedRunnerType,
			},
			expectErrCode: errors.EInvalid, // unlike Phobos, where this is forbidden
			isAdmin:       false,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockRunners := db.NewMockRunners(t)
			mockServiceAccounts := db.NewMockServiceAccounts(t)

			mockRunners.On("GetRunnerByID", mock.Anything, mock.Anything).Return(test.runner, nil)

			if test.runner != nil {
				if test.runner.Type.Equals(models.GroupRunnerType) {
					mockCaller.On("RequirePermission", mock.Anything, permissions.UpdateRunnerPermission, mock.Anything).Return(test.authError)
				} else {
					mockCaller.On("IsAdmin").Return(test.isAdmin).Maybe()
				}
			}

			mockServiceAccounts.On("UnassignServiceAccountFromRunner", mock.Anything, mock.Anything, mock.Anything).
				Return(nil).Maybe()

			dbClient := &db.Client{
				Runners:         mockRunners,
				ServiceAccounts: mockServiceAccounts,
			}

			service := &service{
				dbClient: dbClient,
			}

			err := service.UnassignServiceAccountFromRunner(auth.WithCaller(ctx, mockCaller), "sa-1", "runner-1")

			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCreateRunnerSession(t *testing.T) {
	runnerPath := "runner123"
	runnerSessionID := "runner-session-123"

	// Test cases
	tests := []struct {
		authError                 error
		limitError                error
		input                     *CreateRunnerSessionInput
		createdRunnerSession      *models.RunnerSession
		name                      string
		expectErrCode             errors.CodeType
		allExistingSessionsActive bool
	}{
		{
			name: "create runner session",
			input: &CreateRunnerSessionInput{
				RunnerPath: runnerPath,
			},
			createdRunnerSession: &models.RunnerSession{
				Metadata: models.ResourceMetadata{ID: runnerSessionID},
				RunnerID: runnerPath,
			},
		},
		{
			name: "subject does not have permissions to create a runner",
			input: &CreateRunnerSessionInput{
				RunnerPath: runnerPath,
			},
			authError:     errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrCode: errors.EForbidden,
		},
		{
			name: "oldest session should be deleted when limit is reached",
			input: &CreateRunnerSessionInput{
				RunnerPath: runnerPath,
			},
			createdRunnerSession: &models.RunnerSession{
				Metadata: models.ResourceMetadata{ID: runnerSessionID},
				RunnerID: runnerPath,
			},
			limitError: errors.New("mocked limit violation", errors.WithErrorCode(errors.EInvalid)),
		},
		{
			name: "create session should fail because limit is reached and all sessions are currently active",
			input: &CreateRunnerSessionInput{
				RunnerPath: runnerPath,
			},
			createdRunnerSession: &models.RunnerSession{
				Metadata: models.ResourceMetadata{ID: runnerSessionID},
				RunnerID: runnerPath,
			},
			allExistingSessionsActive: true,
			limitError:                errors.New("mocked limit violation", errors.WithErrorCode(errors.EInvalid)),
			expectErrCode:             errors.ETooManyRequests,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockRunners := db.NewMockRunners(t)
			mockRunnerSessions := db.NewMockRunnerSessions(t)
			mockLogStreams := db.NewMockLogStreams(t)
			mockTransactions := db.NewMockTransactions(t)
			mockLimitChecker := limits.NewMockLimitChecker(t)

			mockCaller.On("RequirePermission", mock.Anything, permissions.CreateRunnerSessionPermission, mock.Anything).
				Return(test.authError)

			mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil).Maybe()
			mockTransactions.On("RollbackTx", mock.Anything).Return(nil).Maybe()

			mockRunners.On("GetRunnerByPath", mock.Anything, runnerPath).Return(&models.Runner{
				Metadata: models.ResourceMetadata{ID: runnerPath},
			}, nil)

			mockRunnerSessions.On("GetRunnerSessions", mock.Anything, &db.GetRunnerSessionsInput{
				Filter: &db.RunnerSessionFilter{
					RunnerID: &runnerPath,
				},
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(0),
				},
			}).Return(&db.RunnerSessionsResult{
				PageInfo: &pagination.PageInfo{},
			}, nil).Maybe()

			mockLimitChecker.On("CheckLimit", mock.Anything,
				limits.ResourceLimitRunnerSessionsPerRunner, int32(0)).
				Return(test.limitError).Maybe()

			if test.createdRunnerSession != nil {
				mockRunnerSessions.On("CreateRunnerSession", mock.Anything, mock.Anything).Return(test.createdRunnerSession, nil)

				sessionID := test.createdRunnerSession.Metadata.ID
				mockLogStreams.On("CreateLogStream", mock.Anything, &models.LogStream{
					RunnerSessionID: &sessionID,
				}).Return(nil, nil)
			}

			if test.limitError != nil {
				existingSession := models.RunnerSession{
					Metadata: models.ResourceMetadata{ID: "existing-session"},
					RunnerID: runnerPath,
				}

				if test.allExistingSessionsActive {
					existingSession.LastContactTimestamp = time.Now().UTC()
				}

				sortBy := db.RunnerSessionSortableFieldLastContactedAtAsc
				mockRunnerSessions.On("GetRunnerSessions", mock.Anything, &db.GetRunnerSessionsInput{
					Sort: &sortBy,
					Filter: &db.RunnerSessionFilter{
						RunnerID: &runnerPath,
					},
					PaginationOptions: &pagination.Options{
						First: ptr.Int32(1),
					},
				}).Return(&db.RunnerSessionsResult{
					PageInfo:       &pagination.PageInfo{},
					RunnerSessions: []models.RunnerSession{existingSession},
				}, nil)

				if !test.allExistingSessionsActive {
					mockRunnerSessions.On("DeleteRunnerSession", mock.Anything, &existingSession).Return(nil)
				}
			}

			if test.authError == nil && test.expectErrCode == "" {
				mockTransactions.On("CommitTx", mock.Anything).Return(nil)
			}

			dbClient := &db.Client{
				Transactions:   mockTransactions,
				Runners:        mockRunners,
				RunnerSessions: mockRunnerSessions,
				LogStreams:     mockLogStreams,
			}

			testLogger, _ := logger.NewForTest()

			service := &service{
				logger:       testLogger,
				dbClient:     dbClient,
				limitChecker: mockLimitChecker,
			}

			session, err := service.CreateRunnerSession(auth.WithCaller(ctx, mockCaller), test.input)

			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.createdRunnerSession, session)
		})
	}
}

func TestGetRunnerSessions(t *testing.T) {
	runnerID := "runner123"

	// Test cases
	tests := []struct {
		input         *GetRunnerSessionsInput
		runner        *models.Runner
		name          string
		authError     error
		expectErrCode errors.CodeType
		isAdmin       bool
	}{
		{
			name:  "successfully get sessions for a group runner",
			input: &GetRunnerSessionsInput{RunnerID: runnerID},
			runner: &models.Runner{
				Type:    models.GroupRunnerType,
				GroupID: ptr.String("group123"),
			},
		},
		{
			name:  "successfully get sessiosn for a shared runner",
			input: &GetRunnerSessionsInput{RunnerID: runnerID},
			runner: &models.Runner{
				Type: models.SharedRunnerType,
			},
			isAdmin: true,
		},
		{
			name:  "subject is not authorized to query sessions for a group runner",
			input: &GetRunnerSessionsInput{RunnerID: runnerID},
			runner: &models.Runner{
				Type:    models.GroupRunnerType,
				GroupID: ptr.String("group123"),
			},
			authError:     errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrCode: errors.EForbidden,
		},
		{
			name:          "runner not found",
			input:         &GetRunnerSessionsInput{RunnerID: runnerID},
			expectErrCode: errors.ENotFound,
		},
		{
			name:  "cannot query sessions for a shared runner because subject is not an admin",
			input: &GetRunnerSessionsInput{RunnerID: runnerID},
			runner: &models.Runner{
				Type: models.SharedRunnerType,
			},
			expectErrCode: errors.EForbidden,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockRunners := db.NewMockRunners(t)
			mockRunnerSessions := db.NewMockRunnerSessions(t)

			mockRunners.On("GetRunnerByID", mock.Anything, runnerID).Return(test.runner, nil)

			if test.runner != nil {
				if test.runner.Type == models.GroupRunnerType {
					mockCaller.On("RequireAccessToInheritableResource",
						mock.Anything, permissions.RunnerResourceType, mock.Anything, mock.Anything).
						Return(test.authError)
				} else {
					mockCaller.On("IsAdmin").Return(test.isAdmin)
				}
			}

			mockRunnerSessions.On("GetRunnerSessions", mock.Anything, &db.GetRunnerSessionsInput{
				Filter: &db.RunnerSessionFilter{
					RunnerID: &runnerID,
				},
			}).Return(&db.RunnerSessionsResult{}, nil).Maybe()

			dbClient := &db.Client{
				Runners:        mockRunners,
				RunnerSessions: mockRunnerSessions,
			}

			service := &service{
				dbClient: dbClient,
			}

			resp, err := service.GetRunnerSessions(auth.WithCaller(ctx, mockCaller), test.input)

			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			require.NotNil(t, resp)
		})
	}
}

func TestGetRunnerSessionByID(t *testing.T) {
	runnerID := "runner123"
	runnerSessionID := "runner-session-123"

	// Test cases
	tests := []struct {
		runner        *models.Runner
		name          string
		authError     error
		expectErrCode errors.CodeType
		isAdmin       bool
	}{
		{
			name: "successfully get session by ID for a group runner",
			runner: &models.Runner{
				Type:    models.GroupRunnerType,
				GroupID: ptr.String("group123"),
			},
		},
		{
			name: "successfully get session for a shared runner",
			runner: &models.Runner{
				Type: models.SharedRunnerType,
			},
			isAdmin: true,
		},
		{
			name: "subject is not authorized to query session for a group runner",
			runner: &models.Runner{
				Type:    models.GroupRunnerType,
				GroupID: ptr.String("group123"),
			},
			authError:     errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrCode: errors.EForbidden,
		},
		{
			name:          "runner not found",
			expectErrCode: errors.ENotFound,
		},
		{
			name: "cannot query session for a shared runner because subject is not an admin",
			runner: &models.Runner{
				Type: models.SharedRunnerType,
			},
			expectErrCode: errors.EForbidden,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockRunners := db.NewMockRunners(t)
			mockRunnerSessions := db.NewMockRunnerSessions(t)

			mockRunners.On("GetRunnerByID", mock.Anything, runnerID).Return(test.runner, nil)

			if test.runner != nil {
				if test.runner.Type == models.GroupRunnerType {
					mockCaller.On("RequireAccessToInheritableResource",
						mock.Anything, permissions.RunnerResourceType, mock.Anything, mock.Anything).
						Return(test.authError)
				} else {
					mockCaller.On("IsAdmin").Return(test.isAdmin)
				}
			}

			mockRunnerSessions.On("GetRunnerSessionByID", mock.Anything, runnerSessionID).Return(&models.RunnerSession{
				Metadata: models.ResourceMetadata{ID: runnerSessionID},
				RunnerID: runnerID,
			}, nil).Maybe()

			dbClient := &db.Client{
				Runners:        mockRunners,
				RunnerSessions: mockRunnerSessions,
			}

			service := &service{
				dbClient: dbClient,
			}

			resp, err := service.GetRunnerSessionByID(auth.WithCaller(ctx, mockCaller), runnerSessionID)

			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			require.NotNil(t, resp)
		})
	}
}

func TestAcceptRunnerSessionHeartbeat(t *testing.T) {
	runnerSessionID := "runner-session-123"

	runnerSession := models.RunnerSession{
		Metadata: models.ResourceMetadata{ID: runnerSessionID},
		RunnerID: "runner123",
	}

	// Test cases
	tests := []struct {
		name          string
		authError     error
		expectErrCode errors.CodeType
	}{
		{
			name: "successfully accept heartbeat for a runner session",
		},
		{
			name:          "subject is not authorized",
			authError:     errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrCode: errors.EForbidden,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockRunnerSessions := db.NewMockRunnerSessions(t)

			mockCaller.On("RequirePermission", mock.Anything, permissions.UpdateRunnerSessionPermission, mock.Anything).Return(test.authError)

			mockRunnerSessions.On("GetRunnerSessionByID", mock.Anything, runnerSessionID).Return(&runnerSession, nil)

			if test.authError == nil {
				matcher := mock.MatchedBy(func(session *models.RunnerSession) bool {
					return session.LastContactTimestamp.After(time.Now().UTC().Add(-time.Minute))
				})
				mockRunnerSessions.On("UpdateRunnerSession", mock.Anything, matcher).Return(nil, nil)
			}

			dbClient := &db.Client{
				RunnerSessions: mockRunnerSessions,
			}

			service := &service{
				dbClient: dbClient,
			}

			err := service.AcceptRunnerSessionHeartbeat(auth.WithCaller(ctx, mockCaller), runnerSessionID)

			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCreateRunnerSessionError(t *testing.T) {
	runnerID := "runner1"
	runnerSessionID := "runner-session-1"
	logStreamID := "log-stream-1"
	errorCount := 1
	message := "runner failed to claim job"

	// Test cases
	tests := []struct {
		authError     error
		name          string
		expectErrCode errors.CodeType
		logStreamSize int
	}{
		{
			name:          "create runner session error",
			logStreamSize: 100,
		},
		{
			name:          "subject does not have permission to create session error",
			authError:     errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrCode: errors.EForbidden,
		},
		{
			name:          "fail to create error because error log has exceeded the limit",
			logStreamSize: runnerErrorLogsBytesLimit,
			expectErrCode: errors.ETooLarge,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockRunnerSessions := db.NewMockRunnerSessions(t)
			mockLogStreams := db.NewMockLogStreams(t)
			mockTransactions := db.NewMockTransactions(t)
			mockLimitChecker := limits.NewMockLimitChecker(t)
			mockLogStreamManager := logstream.NewMockManager(t)

			mockCaller.On("RequirePermission", mock.Anything, permissions.UpdateRunnerSessionPermission, mock.Anything).
				Return(test.authError)

			mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil).Maybe()
			mockTransactions.On("RollbackTx", mock.Anything).Return(nil).Maybe()

			mockLogStreams.On("GetLogStreamByRunnerSessionID", mock.Anything, runnerSessionID).Return(&models.LogStream{
				Metadata: models.ResourceMetadata{ID: logStreamID},
				Size:     test.logStreamSize,
			}, nil).Maybe()

			mockRunnerSessions.On("GetRunnerSessionByID", mock.Anything, runnerSessionID).Return(&models.RunnerSession{
				Metadata:   models.ResourceMetadata{ID: runnerSessionID},
				RunnerID:   runnerID,
				ErrorCount: errorCount,
			}, nil).Maybe()

			mockRunnerSessions.On("UpdateRunnerSession",
				mock.Anything,
				mock.MatchedBy(func(session *models.RunnerSession) bool {
					// Verify error count was updated
					return session.ErrorCount > errorCount
				})).Return(nil, nil).Maybe()

			mockLogStreamManager.On("WriteLogs",
				mock.Anything,
				logStreamID,
				test.logStreamSize,
				mock.MatchedBy(func(buf []byte) bool {
					return strings.Contains(string(buf), message)
				}),
			).Return(nil, nil).Maybe()

			if test.authError == nil && test.expectErrCode == "" {
				mockTransactions.On("CommitTx", mock.Anything).Return(nil)
			}

			dbClient := &db.Client{
				Transactions:   mockTransactions,
				RunnerSessions: mockRunnerSessions,
				LogStreams:     mockLogStreams,
			}

			testLogger, _ := logger.NewForTest()

			service := &service{
				logger:           testLogger,
				dbClient:         dbClient,
				limitChecker:     mockLimitChecker,
				logStreamManager: mockLogStreamManager,
			}

			err := service.CreateRunnerSessionError(auth.WithCaller(ctx, mockCaller), runnerSessionID, message)

			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestReadRunnerSessionErrorLog(t *testing.T) {
	runnerID := "runner123"
	runnerSessionID := "runner-session-123"
	logStreamID := "log-stream-1"

	// Test cases
	tests := []struct {
		runner        *models.Runner
		name          string
		authError     error
		expectErrCode errors.CodeType
		isAdmin       bool
	}{
		{
			name: "successfully read logs for a group runner",
			runner: &models.Runner{
				Type:    models.GroupRunnerType,
				GroupID: ptr.String("group123"),
			},
		},
		{
			name: "successfully read logs for a shared runner",
			runner: &models.Runner{
				Type: models.SharedRunnerType,
			},
			isAdmin: true,
		},
		{
			name: "subject is not authorized to read logs for a group runner",
			runner: &models.Runner{
				Type:    models.GroupRunnerType,
				GroupID: ptr.String("group123"),
			},
			authError:     errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrCode: errors.EForbidden,
		},
		{
			name:          "runner not found",
			expectErrCode: errors.ENotFound,
		},
		{
			name: "cannot read logs for a shared runner because subject is not an admin",
			runner: &models.Runner{
				Type: models.SharedRunnerType,
			},
			expectErrCode: errors.EForbidden,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockRunners := db.NewMockRunners(t)
			mockRunnerSessions := db.NewMockRunnerSessions(t)
			mockLogStream := db.NewMockLogStreams(t)
			mockLogStreamManager := logstream.NewMockManager(t)

			mockRunners.On("GetRunnerByID", mock.Anything, runnerID).Return(test.runner, nil)

			if test.runner != nil {
				if test.runner.Type == models.GroupRunnerType {
					mockCaller.On("RequireAccessToInheritableResource",
						mock.Anything, permissions.RunnerResourceType, mock.Anything, mock.Anything).
						Return(test.authError)
				} else {
					mockCaller.On("IsAdmin").Return(test.isAdmin)
				}
			}

			mockRunnerSessions.On("GetRunnerSessionByID", mock.Anything, runnerSessionID).Return(&models.RunnerSession{
				Metadata: models.ResourceMetadata{ID: runnerSessionID},
				RunnerID: runnerID,
			}, nil).Maybe()

			mockLogStream.On("GetLogStreamByRunnerSessionID", mock.Anything, runnerSessionID).Return(&models.LogStream{
				Metadata: models.ResourceMetadata{ID: logStreamID},
				Size:     100,
			}, nil).Maybe()

			mockLogStreamManager.On("ReadLogs",
				mock.Anything,
				logStreamID,
				0,
				100,
			).Return([]byte("hello"), nil).Maybe()

			dbClient := db.Client{
				Runners:        mockRunners,
				RunnerSessions: mockRunnerSessions,
				LogStreams:     mockLogStream,
			}

			service := &service{
				dbClient:         &dbClient,
				logStreamManager: mockLogStreamManager,
			}

			resp, err := service.ReadRunnerSessionErrorLog(auth.WithCaller(ctx, mockCaller), runnerSessionID, 0, 100)

			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			require.NotNil(t, resp)
		})
	}
}

func TestGetLogStreamsByRunnerSessionIDs(t *testing.T) {
	runnerID := "runner123"

	// Test cases
	tests := []struct {
		authError     error
		runner        *models.Runner
		name          string
		expectErrCode errors.CodeType
		sessionIDs    []string
		isAdmin       bool
	}{
		{
			name:       "get log streams for a group runner",
			sessionIDs: []string{"session-1", "session-2"},
			runner: &models.Runner{
				Metadata: models.ResourceMetadata{ID: runnerID},
				Type:     models.GroupRunnerType,
				GroupID:  ptr.String("group123"),
			},
		},
		{
			name:       "get log streams for a shared runner",
			sessionIDs: []string{"session-1", "session-2"},
			runner: &models.Runner{
				Metadata: models.ResourceMetadata{ID: runnerID},
				Type:     models.SharedRunnerType,
			},
			isAdmin: true,
		},
		{
			name:       "caller is not authorized to get log streams for group runner",
			sessionIDs: []string{"session-1", "session-2"},
			runner: &models.Runner{
				Metadata: models.ResourceMetadata{ID: runnerID},
				Type:     models.GroupRunnerType,
				GroupID:  ptr.String("group123"),
			},
			authError:     errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrCode: errors.EForbidden,
		},
		{
			name:       "caller is not authorized to get log streams for shared runner",
			sessionIDs: []string{"session-1", "session-2"},
			runner: &models.Runner{
				Metadata: models.ResourceMetadata{ID: runnerID},
				Type:     models.SharedRunnerType,
			},
			expectErrCode: errors.EForbidden,
		},
		{
			name:       "handle empty session ID list",
			sessionIDs: []string{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockRunners := db.NewMockRunners(t)
			mockRunnerSessions := db.NewMockRunnerSessions(t)
			mockLogStream := db.NewMockLogStreams(t)

			if test.runner != nil {
				if test.runner.Type == models.GroupRunnerType {
					mockCaller.On("RequireAccessToInheritableResource",
						mock.Anything, permissions.RunnerResourceType, mock.Anything, mock.Anything).
						Return(test.authError)
				} else {
					mockCaller.On("IsAdmin").Return(test.isAdmin)
				}
			}

			logStreams := []models.LogStream{}

			if len(test.sessionIDs) > 0 {
				runnerSessions := []models.RunnerSession{}
				for _, sessionID := range test.sessionIDs {
					runnerSessions = append(runnerSessions, models.RunnerSession{
						Metadata: models.ResourceMetadata{ID: sessionID},
						RunnerID: runnerID,
					})
				}

				mockRunnerSessions.On("GetRunnerSessions", mock.Anything, &db.GetRunnerSessionsInput{
					Filter: &db.RunnerSessionFilter{
						RunnerSessionIDs: test.sessionIDs,
					},
				}).Return(&db.RunnerSessionsResult{
					RunnerSessions: runnerSessions,
				}, nil)

				mockRunners.On("GetRunners", mock.Anything, &db.GetRunnersInput{
					Filter: &db.RunnerFilter{
						RunnerIDs: []string{runnerID},
					},
				}).Return(&db.RunnersResult{
					Runners: []models.Runner{*test.runner},
				}, nil)

				for idx, sessionID := range test.sessionIDs {
					sessionIDCopy := sessionID
					logStreams = append(logStreams, models.LogStream{
						Metadata:        models.ResourceMetadata{ID: fmt.Sprintf("log-stream-%d", idx)},
						RunnerSessionID: &sessionIDCopy,
					})
				}

				mockLogStream.On("GetLogStreams", mock.Anything, &db.GetLogStreamsInput{
					Filter: &db.LogStreamFilter{
						RunnerSessionIDs: test.sessionIDs,
					},
				}).Return(&db.LogStreamsResult{
					LogStreams: logStreams,
				}, nil).Maybe()
			}

			dbClient := &db.Client{
				Runners:        mockRunners,
				RunnerSessions: mockRunnerSessions,
				LogStreams:     mockLogStream,
			}

			service := &service{
				dbClient: dbClient,
			}

			resp, err := service.GetLogStreamsByRunnerSessionIDs(auth.WithCaller(ctx, mockCaller), test.sessionIDs)

			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			require.NotNil(t, resp)
			assert.Equal(t, logStreams, resp)
		})
	}
}

func TestSubscribeToRunnerSessionErrorLog(t *testing.T) {
	runnerID := "runner123"
	runnerSessionID := "runner-session-123"
	logStreamID := "log-stream-1"

	// Test cases
	tests := []struct {
		runner          *models.Runner
		lastSeenLogSize *int
		name            string
		authError       error
		expectErrCode   errors.CodeType
		isAdmin         bool
	}{
		{
			name: "successfully subscribe to log stream events for a group runner",
			runner: &models.Runner{
				Type:    models.GroupRunnerType,
				GroupID: ptr.String("group123"),
			},
			lastSeenLogSize: ptr.Int(100),
		},
		{
			name: "successfully subscribe to log stream events for a shared runner",
			runner: &models.Runner{
				Type: models.SharedRunnerType,
			},
			isAdmin: true,
		},
		{
			name: "subject is not authorized to subscribe to events for a group runner",
			runner: &models.Runner{
				Type:    models.GroupRunnerType,
				GroupID: ptr.String("group123"),
			},
			authError:     errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrCode: errors.EForbidden,
		},
		{
			name:          "runner not found",
			expectErrCode: errors.ENotFound,
		},
		{
			name: "subject is not authorized to subscribe to events for a shared runner because subject is not an admin",
			runner: &models.Runner{
				Type: models.SharedRunnerType,
			},
			expectErrCode: errors.EForbidden,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockRunners := db.NewMockRunners(t)
			mockRunnerSessions := db.NewMockRunnerSessions(t)
			mockLogStream := db.NewMockLogStreams(t)
			mockLogStreamManager := logstream.NewMockManager(t)

			mockRunners.On("GetRunnerByID", mock.Anything, runnerID).Return(test.runner, nil)

			if test.runner != nil {
				if test.runner.Type == models.GroupRunnerType {
					mockCaller.On("RequireAccessToInheritableResource",
						mock.Anything, permissions.RunnerResourceType, mock.Anything, mock.Anything).
						Return(test.authError)
				} else {
					mockCaller.On("IsAdmin").Return(test.isAdmin)
				}
			}

			mockRunnerSessions.On("GetRunnerSessionByID", mock.Anything, runnerSessionID).Return(&models.RunnerSession{
				Metadata: models.ResourceMetadata{ID: runnerSessionID},
				RunnerID: runnerID,
			}, nil).Maybe()

			mockLogStream.On("GetLogStreamByRunnerSessionID", mock.Anything, runnerSessionID).Return(&models.LogStream{
				Metadata: models.ResourceMetadata{ID: logStreamID},
				Size:     100,
			}, nil).Maybe()

			eventChan := make(<-chan *logstream.LogEvent)
			mockLogStreamManager.On("Subscribe",
				mock.Anything,
				&logstream.SubscriptionOptions{
					LastSeenLogSize: test.lastSeenLogSize,
					LogStreamID:     logStreamID,
				},
			).Return(eventChan, nil).Maybe()

			dbClient := db.Client{
				Runners:        mockRunners,
				RunnerSessions: mockRunnerSessions,
				LogStreams:     mockLogStream,
			}

			service := &service{
				dbClient:         &dbClient,
				logStreamManager: mockLogStreamManager,
			}

			resp, err := service.SubscribeToRunnerSessionErrorLog(auth.WithCaller(ctx, mockCaller), &SubscribeToRunnerSessionErrorLogInput{
				RunnerSessionID: runnerSessionID,
				LastSeenLogSize: test.lastSeenLogSize,
			})

			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			require.NotNil(t, resp)
		})
	}
}

func TestSubscribeToRunnerSessions(t *testing.T) {
	// Test cases
	tests := []struct {
		authError      error
		input          *SubscribeToRunnerSessionsInput
		name           string
		expectErrCode  errors.CodeType
		runners        []models.Runner
		sendEvents     []SessionEvent
		expectedEvents []SessionEvent
		isAdmin        bool
	}{
		{
			name: "subscribe to runner session events for a group",
			input: &SubscribeToRunnerSessionsInput{
				GroupID: ptr.String("group1"),
			},
			sendEvents: []SessionEvent{
				{
					RunnerSession: &models.RunnerSession{
						Metadata: models.ResourceMetadata{ID: "session1"},
						RunnerID: "runner1",
					},
				},
				{
					RunnerSession: &models.RunnerSession{
						Metadata: models.ResourceMetadata{ID: "session2"},
						RunnerID: "runner2",
					},
				},
			},
			expectedEvents: []SessionEvent{
				{
					RunnerSession: &models.RunnerSession{
						Metadata: models.ResourceMetadata{ID: "session1"},
						RunnerID: "runner1",
					},
					Action: "UPDATE",
				},
			},
			runners: []models.Runner{
				{
					Metadata: models.ResourceMetadata{ID: "runner1"},
					Type:     models.GroupRunnerType,
					GroupID:  ptr.String("group1"),
				},
				{
					Metadata: models.ResourceMetadata{ID: "runner2"},
					Type:     models.GroupRunnerType,
					GroupID:  ptr.String("group2"),
				},
			},
		},
		{
			name: "not authorized to subscribe to events for a group",
			input: &SubscribeToRunnerSessionsInput{
				GroupID: ptr.String("group1"),
			},
			authError:     errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrCode: errors.EForbidden,
		},
		{
			name: "subscribe to events for a group runner",
			input: &SubscribeToRunnerSessionsInput{
				RunnerID: ptr.String("runner1"),
			},
			sendEvents: []SessionEvent{
				{
					RunnerSession: &models.RunnerSession{
						Metadata: models.ResourceMetadata{ID: "session1"},
						RunnerID: "runner1",
					},
				},
				{
					RunnerSession: &models.RunnerSession{
						Metadata: models.ResourceMetadata{ID: "session2"},
						RunnerID: "runner2",
					},
				},
			},
			expectedEvents: []SessionEvent{
				{
					RunnerSession: &models.RunnerSession{
						Metadata: models.ResourceMetadata{ID: "session1"},
						RunnerID: "runner1",
					},
					Action: "UPDATE",
				},
			},
			runners: []models.Runner{
				{
					Metadata: models.ResourceMetadata{ID: "runner1"},
					Type:     models.GroupRunnerType,
					GroupID:  ptr.String("group1"),
				},
				{
					Metadata: models.ResourceMetadata{ID: "runner2"},
					Type:     models.GroupRunnerType,
					GroupID:  ptr.String("group1"),
				},
			},
		},
		{
			name: "not authorized to subscribe to events for a group runner",
			input: &SubscribeToRunnerSessionsInput{
				RunnerID: ptr.String("runner1"),
			},
			runners: []models.Runner{
				{
					Metadata: models.ResourceMetadata{ID: "runner1"},
					Type:     models.GroupRunnerType,
					GroupID:  ptr.String("group1"),
				},
			},
			authError:     errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrCode: errors.EForbidden,
		},
		{
			name: "subscribe to events for a shared runner",
			input: &SubscribeToRunnerSessionsInput{
				RunnerID: ptr.String("runner1"),
			},
			runners: []models.Runner{
				{
					Metadata: models.ResourceMetadata{ID: "runner1"},
					Type:     models.SharedRunnerType,
				},
			},
			isAdmin: true,
		},
		{
			name: "not authorized to subscribe to events for a shared runner",
			input: &SubscribeToRunnerSessionsInput{
				RunnerID: ptr.String("runner1"),
			},
			runners: []models.Runner{
				{
					Metadata: models.ResourceMetadata{ID: "runner1"},
					Type:     models.SharedRunnerType,
				},
			},
			expectErrCode: errors.EForbidden,
		},
		{
			name:    "subscribe to all runner session events",
			input:   &SubscribeToRunnerSessionsInput{},
			isAdmin: true,
			sendEvents: []SessionEvent{
				{
					RunnerSession: &models.RunnerSession{
						Metadata: models.ResourceMetadata{ID: "session1"},
						RunnerID: "runner1",
					},
				},
				{
					RunnerSession: &models.RunnerSession{
						Metadata: models.ResourceMetadata{ID: "session2"},
						RunnerID: "runner2",
					},
				},
			},
			expectedEvents: []SessionEvent{
				{
					RunnerSession: &models.RunnerSession{
						Metadata: models.ResourceMetadata{ID: "session1"},
						RunnerID: "runner1",
					},
					Action: "UPDATE",
				},
				{
					RunnerSession: &models.RunnerSession{
						Metadata: models.ResourceMetadata{ID: "session2"},
						RunnerID: "runner2",
					},
					Action: "UPDATE",
				},
			},
			runners: []models.Runner{
				{
					Metadata: models.ResourceMetadata{ID: "runner1"},
					Type:     models.GroupRunnerType,
					GroupID:  ptr.String("group1"),
				},
				{
					Metadata: models.ResourceMetadata{ID: "runner2"},
					Type:     models.GroupRunnerType,
					GroupID:  ptr.String("group2"),
				},
			},
		},
		{
			name:          "not authorized to subscribe to all runner session events",
			input:         &SubscribeToRunnerSessionsInput{},
			expectErrCode: errors.EForbidden,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockRunners := db.NewMockRunners(t)
			mockRunnerSessions := db.NewMockRunnerSessions(t)
			mockEvents := db.NewMockEvents(t)

			mockEventChannel := make(chan db.Event, 1)
			var roEventChan <-chan db.Event = mockEventChannel
			mockEvents.On("Listen", mock.Anything).Return(roEventChan, make(<-chan error)).Maybe()

			for _, runner := range test.runners {
				runnerCopy := runner
				mockRunners.On("GetRunnerByID", mock.Anything, runner.Metadata.ID).Return(&runnerCopy, nil).Maybe()
			}

			if test.input.GroupID != nil {
				mockCaller.On("RequireAccessToInheritableResource", mock.Anything, permissions.RunnerResourceType, mock.Anything).
					Return(test.authError)
			} else if test.input.RunnerID != nil {
				mockCaller.On("RequireAccessToInheritableResource",
					mock.Anything, permissions.RunnerResourceType, mock.Anything, mock.Anything).
					Return(test.authError).Maybe()
				mockCaller.On("IsAdmin").Return(test.isAdmin).Maybe()
			} else {
				mockCaller.On("IsAdmin").Return(test.isAdmin)
			}

			for _, e := range test.sendEvents {
				mockRunnerSessions.On("GetRunnerSessionByID", mock.Anything, e.RunnerSession.Metadata.ID).Return(e.RunnerSession, nil).Maybe()
			}

			dbClient := db.Client{
				Runners:        mockRunners,
				RunnerSessions: mockRunnerSessions,
				Events:         mockEvents,
			}

			logger, _ := logger.NewForTest()
			eventManager := events.NewEventManager(&dbClient, logger)
			eventManager.Start(ctx)

			service := &service{
				dbClient:     &dbClient,
				eventManager: eventManager,
				logger:       logger,
			}

			events, err := service.SubscribeToRunnerSessions(auth.WithCaller(ctx, mockCaller), test.input)

			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			receivedEvents := []*SessionEvent{}

			go func() {
				for _, e := range test.sendEvents {
					mockEventChannel <- db.Event{
						Table:  "runner_sessions",
						Action: "UPDATE",
						ID:     e.RunnerSession.Metadata.ID,
					}
				}
			}()

			if len(test.expectedEvents) > 0 {
				for e := range events {
					eCopy := e
					receivedEvents = append(receivedEvents, eCopy)

					if len(receivedEvents) == len(test.expectedEvents) {
						break
					}
				}
			}

			require.Equal(t, len(test.expectedEvents), len(receivedEvents))
			for i, e := range test.expectedEvents {
				assert.Equal(t, e, *receivedEvents[i])
			}
		})
	}
}
