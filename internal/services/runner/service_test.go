package runner

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth/permissions"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestGetRunnerByID(t *testing.T) {
	runnerID := "runner-1"
	groupID := "group-1"
	// Test cases
	tests := []struct {
		expectRunner  *models.Runner
		name          string
		authError     error
		expectErrCode string
	}{
		{
			name: "get shared runner",
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
			authError:     errors.New(errors.EForbidden, "Unauthorized"),
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
				mockCaller.On("RequireAccessToInheritableResource", mock.Anything, permissions.RunnerResourceType, mock.Anything).Return(test.authError)
			}

			mockRunners := db.NewMockRunners(t)

			mockRunners.On("GetRunnerByID", mock.Anything, runnerID).Return(test.expectRunner, nil)

			dbClient := db.Client{
				Runners: mockRunners,
			}

			testLogger, _ := logger.NewForTest()

			service := NewService(testLogger, &dbClient, nil)

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
		expectErrCode string
	}{
		{
			name: "get shared runner",
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
			authError:     errors.New(errors.EForbidden, "Unauthorized"),
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
				mockCaller.On("RequireAccessToInheritableResource", mock.Anything, permissions.RunnerResourceType, mock.Anything).Return(test.authError)
			}

			mockRunners := db.NewMockRunners(t)

			mockRunners.On("GetRunnerByPath", mock.Anything, path).Return(test.expectRunner, nil)

			dbClient := db.Client{
				Runners: mockRunners,
			}

			testLogger, _ := logger.NewForTest()

			service := NewService(testLogger, &dbClient, nil)

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
		expectErrCode string
	}{
		{
			name: "get shared runner",
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
			authError:     errors.New(errors.EForbidden, "Unauthorized"),
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
				mockCaller.On("RequireAccessToInheritableResource", mock.Anything, permissions.RunnerResourceType, mock.Anything).Return(test.authError)
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

			service := NewService(testLogger, &dbClient, nil)

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
	groupID := "group-1"
	// Test cases
	tests := []struct {
		input         *GetRunnersInput
		expectRunner  *models.Runner
		name          string
		authError     error
		expectErrCode string
	}{
		{
			name: "filter runners by group and allow access",
			input: &GetRunnersInput{
				NamespacePath: "group-1",
			},
			expectRunner: &models.Runner{
				Type:     models.GroupRunnerType,
				Metadata: models.ResourceMetadata{ID: groupID},
				GroupID:  &groupID,
				Name:     "test-runner",
			},
		},
		{
			name: "subject does not have viewer role for group",
			input: &GetRunnersInput{
				NamespacePath: "group-1",
			},
			authError:     errors.New(errors.EForbidden, "Unauthorized"),
			expectErrCode: errors.EForbidden,
		},
		{
			name: "no runners matching filters",
			input: &GetRunnersInput{
				NamespacePath: "group-1",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)

			mockCaller.On("RequirePermission", mock.Anything, permissions.ViewRunnerPermission, mock.Anything).Return(test.authError)

			mockRunners := db.NewMockRunners(t)

			getRunnersResponse := db.RunnersResult{
				Runners: []models.Runner{},
			}

			if test.expectRunner != nil {
				getRunnersResponse.Runners = append(getRunnersResponse.Runners, *test.expectRunner)
			}

			mockRunners.On("GetRunners", mock.Anything, &db.GetRunnersInput{
				Sort:              test.input.Sort,
				PaginationOptions: test.input.PaginationOptions,
				Filter: &db.RunnerFilter{
					NamespacePaths: []string{test.input.NamespacePath},
				},
			}).Return(&getRunnersResponse, nil).Maybe()

			dbClient := db.Client{
				Runners: mockRunners,
			}

			testLogger, _ := logger.NewForTest()

			service := NewService(testLogger, &dbClient, nil)

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
		authError           error
		expectCreatedRunner *models.Runner
		name                string
		expectErrCode       string
		input               CreateRunnerInput
	}{
		{
			name: "create group runner",
			input: CreateRunnerInput{
				Name:    "test-runner",
				GroupID: groupID,
			},
			expectCreatedRunner: &models.Runner{
				Type:         models.GroupRunnerType,
				Name:         "test-runner",
				GroupID:      &groupID,
				CreatedBy:    "mockSubject",
				ResourcePath: "group-1/test-runner",
			},
		},
		{
			name: "subject does not have owner role",
			input: CreateRunnerInput{
				Name:    "test-runner",
				GroupID: groupID,
			},
			authError:     errors.New(errors.EForbidden, "Unauthorized"),
			expectErrCode: errors.EForbidden,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.MockCaller{}
			mockCaller.Test(t)

			mockCaller.On("RequirePermission", mock.Anything, permissions.CreateRunnerPermission, mock.Anything).Return(test.authError)

			mockCaller.On("GetSubject").Return("mockSubject")

			mockTransactions := db.NewMockTransactions(t)
			mockRunners := db.NewMockRunners(t)

			if test.authError == nil {
				mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
				mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
				mockTransactions.On("CommitTx", mock.Anything).Return(nil)
			}

			if test.expectCreatedRunner != nil {
				mockRunners.On("CreateRunner", mock.Anything, mock.Anything).
					Return(test.expectCreatedRunner, nil)
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

			service := NewService(testLogger, &dbClient, mockActivityEvents)

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
		expectErrCode string
	}{
		{
			name: "update runner",
			input: &models.Runner{
				Type:         models.GroupRunnerType,
				Name:         "test-runner",
				GroupID:      &groupID,
				ResourcePath: "group123/test-runner",
			},
		},
		{
			name: "subject does not have owner role",
			input: &models.Runner{
				Type:         models.GroupRunnerType,
				Name:         "test-runner",
				GroupID:      &groupID,
				ResourcePath: "group123/test-runner",
			},
			authError:     errors.New(errors.EForbidden, "Unauthorized"),
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

			service := NewService(testLogger, &dbClient, mockActivityEvents)

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
		expectErrCode string
	}{
		{
			name: "delete runner",
			input: &models.Runner{
				Type:         models.GroupRunnerType,
				Name:         "test-runner",
				GroupID:      &groupID,
				ResourcePath: "group123/test-runner",
			},
		},
		{
			name: "subject does not have deployer role",
			input: &models.Runner{
				Type:         models.GroupRunnerType,
				Name:         "test-runner",
				GroupID:      &groupID,
				ResourcePath: "group123/test-runner",
			},
			authError:     errors.New(errors.EForbidden, "Unauthorized"),
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

			service := NewService(testLogger, &dbClient, mockActivityEvents)

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
