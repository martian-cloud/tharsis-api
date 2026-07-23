package run

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/aws/smithy-go/ptr"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/commands"
	runvariables "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/variables"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/workspace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/events"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/maintenance"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plugin/secret"

	corerun "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
)

type mockDBClient struct {
	*db.Client
	MockTransactions          *db.MockTransactions
	MockManagedIdentities     *db.MockManagedIdentities
	MockWorkspaces            *db.MockWorkspaces
	MockWorkspaceAssessments  *db.MockWorkspaceAssessments
	MockVariables             *db.MockVariables
	MockVariableVersions      *db.MockVariableVersions
	MockRuns                  *db.MockRuns
	MockConfigurationVersions *db.MockConfigurationVersions
	MockJobs                  *db.MockJobs
	MockTeams                 *db.MockTeams
	MockTeamMembers           *db.MockTeamMembers
	MockLogStreams            *db.MockLogStreams
	MockResourceLimits        *db.MockResourceLimits
	MockGroups                *db.MockGroups
	MockStateVersions         *db.MockStateVersions
}

func buildDBClientWithMocks(t *testing.T) *mockDBClient {
	mockTransactions := db.MockTransactions{}
	mockTransactions.Test(t)
	// The mocks are enabled by the above function.

	mockManagedIdentities := db.MockManagedIdentities{}
	mockManagedIdentities.Test(t)

	mockWorkspaces := db.MockWorkspaces{}
	mockWorkspaces.Test(t)

	mockWorkspaceAssessments := db.MockWorkspaceAssessments{}
	mockWorkspaceAssessments.Test(t)

	mockVariables := db.MockVariables{}
	mockVariables.Test(t)

	mockVariableVersions := db.MockVariableVersions{}
	mockVariableVersions.Test(t)

	mockRuns := db.MockRuns{}
	mockRuns.Test(t)

	mockConfigurationVersions := db.MockConfigurationVersions{}
	mockConfigurationVersions.Test(t)

	mockJobs := db.MockJobs{}
	mockJobs.Test(t)

	mockTeams := db.MockTeams{}
	mockTeams.Test(t)

	mockTeamMembers := db.MockTeamMembers{}
	mockTeamMembers.Test(t)

	mockLogStreams := db.MockLogStreams{}
	mockLogStreams.Test(t)

	mockResourceLimits := db.MockResourceLimits{}
	mockResourceLimits.Test(t)

	mockGroups := db.MockGroups{}
	mockGroups.Test(t)

	mockStateVersions := db.MockStateVersions{}
	mockStateVersions.Test(t)

	return &mockDBClient{
		Client: &db.Client{
			Transactions:          &mockTransactions,
			ManagedIdentities:     &mockManagedIdentities,
			Workspaces:            &mockWorkspaces,
			WorkspaceAssessments:  &mockWorkspaceAssessments,
			Variables:             &mockVariables,
			VariableVersions:      &mockVariableVersions,
			Runs:                  &mockRuns,
			ConfigurationVersions: &mockConfigurationVersions,
			Jobs:                  &mockJobs,
			Teams:                 &mockTeams,
			TeamMembers:           &mockTeamMembers,
			LogStreams:            &mockLogStreams,
			ResourceLimits:        &mockResourceLimits,
			Groups:                &mockGroups,
			StateVersions:         &mockStateVersions,
		},
		MockTransactions:          &mockTransactions,
		MockManagedIdentities:     &mockManagedIdentities,
		MockWorkspaces:            &mockWorkspaces,
		MockWorkspaceAssessments:  &mockWorkspaceAssessments,
		MockVariables:             &mockVariables,
		MockVariableVersions:      &mockVariableVersions,
		MockRuns:                  &mockRuns,
		MockConfigurationVersions: &mockConfigurationVersions,
		MockJobs:                  &mockJobs,
		MockTeams:                 &mockTeams,
		MockTeamMembers:           &mockTeamMembers,
		MockLogStreams:            &mockLogStreams,
		MockResourceLimits:        &mockResourceLimits,
		MockGroups:                &mockGroups,
		MockStateVersions:         &mockStateVersions,
	}
}

func TestRunByTRN(t *testing.T) {
	sampleRun := &models.Run{
		Metadata: models.ResourceMetadata{
			ID:  "run-id-1",
			TRN: trn.TypeRun.Build("run-gid-1"),
		},
		WorkspaceID: "workspace-1",
		Status:      models.RunPlanned,
	}

	type testCase struct {
		name            string
		authError       error
		run             *models.Run
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name: "successfully get run by trn",
			run:  sampleRun,
		},
		{
			name:            "run not found",
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "subject is not authorized to view run",
			run:             sampleRun,
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockCaller := auth.NewMockCaller(t)
			mockRuns := db.NewMockRuns(t)

			mockRuns.On("GetRunByTRN", mock.Anything, sampleRun.Metadata.TRN).Return(test.run, nil)

			if test.run != nil {
				mockCaller.On("RequirePermission", mock.Anything, models.ViewRunPermission, mock.Anything, mock.Anything).Return(test.authError)
			}

			dbClient := &db.Client{
				Runs: mockRuns,
			}

			service := &service{
				dbClient: dbClient,
			}

			actualRun, err := service.GetRunByTRN(auth.WithCaller(ctx, mockCaller), sampleRun.Metadata.TRN)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.run, actualRun)
		})
	}
}

func TestGetRunByID(t *testing.T) {
	sampleRun := &models.Run{
		Metadata: models.ResourceMetadata{
			ID: "run-id-1",
		},
		WorkspaceID: "workspace-1",
		Status:      models.RunPlanned,
	}

	type testCase struct {
		name            string
		authError       error
		run             *models.Run
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name: "successfully get run by id",
			run:  sampleRun,
		},
		{
			name:            "run not found",
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "subject is not authorized to view run",
			run:             sampleRun,
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()

			mockCaller := auth.NewMockCaller(t)
			mockRuns := db.NewMockRuns(t)

			mockRuns.On("GetRunByID", mock.Anything, sampleRun.Metadata.ID).Return(test.run, nil)

			if test.run != nil {
				mockCaller.On("RequirePermission", mock.Anything, models.ViewRunPermission, mock.Anything, mock.Anything).Return(test.authError)
			}

			dbClient := &db.Client{
				Runs: mockRuns,
			}

			service := &service{
				dbClient: dbClient,
			}

			actualRun, err := service.GetRunByID(auth.WithCaller(ctx, mockCaller), sampleRun.Metadata.ID)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.run, actualRun)
		})
	}
}

func TestGetRunVariables(t *testing.T) {
	ctx := context.Background()

	runID := "run1"
	run := &models.Run{
		Metadata: models.ResourceMetadata{
			ID: runID,
		},
		WorkspaceID: "ws1",
	}

	runVariables := []runvariables.Variable{
		{
			Key:       "var1",
			Value:     ptr.String("value1"),
			Category:  models.TerraformVariableCategory,
			Sensitive: false,
		},
		{
			Key:       "var2",
			Category:  models.EnvironmentVariableCategory,
			Sensitive: true,
			VersionID: ptr.String("1"),
		},
	}

	marshaledRunVariables, err := json.Marshal(runVariables)
	require.NoError(t, err)

	variableVersions := []models.VariableVersion{}
	variableVersionIDs := []string{}
	// Add variable version for each sensitive variable
	for i, v := range runVariables {
		if v.Sensitive {
			id := strconv.Itoa(i)
			variableVersionIDs = append(variableVersionIDs, id)
			variableVersions = append(variableVersions, models.VariableVersion{
				Metadata:   models.ResourceMetadata{ID: id},
				Key:        v.Key,
				SecretData: []byte(fmt.Sprintf("%s-encrypted", v.Key)),
			})
		}
	}

	tests := []struct {
		name                            string
		includeSensitiveValues          bool
		expectedVariables               []runvariables.Variable
		hasViewVariableValuePermissions bool
		authError                       error
		expectedErrorCode               errors.CodeType
	}{
		{
			name:                            "include sensitive values for caller with view variable value permission",
			includeSensitiveValues:          true,
			hasViewVariableValuePermissions: true,
			expectedVariables: []runvariables.Variable{
				{
					Key:       "var1",
					Value:     ptr.String("value1"),
					Category:  models.TerraformVariableCategory,
					Sensitive: false,
				},
				{
					Key:       "var2",
					Value:     ptr.String("var2-plaintext"),
					Category:  models.EnvironmentVariableCategory,
					Sensitive: true,
					VersionID: ptr.String("1"),
				},
			},
		},
		{
			name:                            "don't include sensitive values for caller with view variable value permission",
			includeSensitiveValues:          false,
			hasViewVariableValuePermissions: true,
			expectedVariables: []runvariables.Variable{
				{
					Key:       "var1",
					Value:     ptr.String("value1"),
					Category:  models.TerraformVariableCategory,
					Sensitive: false,
				},
				{
					Key:       "var2",
					Category:  models.EnvironmentVariableCategory,
					Sensitive: true,
					VersionID: ptr.String("1"),
				},
			},
		},
		{
			name:                            "don't include any values for caller without view variable value permission",
			includeSensitiveValues:          false,
			hasViewVariableValuePermissions: false,
			expectedVariables: []runvariables.Variable{
				{
					Key:       "var1",
					Category:  models.TerraformVariableCategory,
					Sensitive: false,
				},
				{
					Key:       "var2",
					Category:  models.EnvironmentVariableCategory,
					Sensitive: true,
					VersionID: ptr.String("1"),
				},
			},
		},
		{
			name:                            "return error if caller without view variable value permission requests sensitive values",
			includeSensitiveValues:          true,
			hasViewVariableValuePermissions: false,
			expectedErrorCode:               errors.EForbidden,
		},
		{
			name:                            "return error if caller doesn't have view variable permission",
			hasViewVariableValuePermissions: false,
			authError:                       errors.New("no permission", errors.WithErrorCode(errors.EForbidden)),
			expectedErrorCode:               errors.EForbidden,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockDBClient := buildDBClientWithMocks(t)
			mockArtifactStore := workspace.NewMockArtifactStore(t)
			mockSecretManager := secret.NewMockManager(t)
			mockCaller := auth.NewMockCaller(t)

			service := &service{
				dbClient:         mockDBClient.Client,
				artifactStore:    mockArtifactStore,
				variablesBuilder: runvariables.NewBuilder(mockDBClient.Client, mockSecretManager, mockArtifactStore),
			}

			if test.hasViewVariableValuePermissions {
				mockCaller.On("RequirePermission", mock.Anything, models.ViewVariableValuePermission, mock.Anything).Return(nil)
			} else {
				mockCaller.On("RequirePermission", mock.Anything, models.ViewVariableValuePermission, mock.Anything).Return(errors.New("no permission", errors.WithErrorCode(errors.EForbidden)))
				mockCaller.On("RequirePermission", mock.Anything, models.ViewVariablePermission, mock.Anything).Return(test.authError)
			}

			mockDBClient.MockRuns.On("GetRunByID", mock.Anything, runID).Return(run, nil)

			mockArtifactStore.On("GetRunVariables", mock.Anything, run).Return(io.NopCloser(bytes.NewReader(marshaledRunVariables)), nil).Maybe()

			if test.includeSensitiveValues && test.expectedErrorCode == "" {
				mockDBClient.MockVariableVersions.On("GetVariableVersions", mock.Anything, &db.GetVariableVersionsInput{
					Filter: &db.VariableVersionFilter{
						VariableVersionIDs: variableVersionIDs,
					},
				}).Return(&db.VariableVersionResult{
					VariableVersions: variableVersions,
				}, nil)

				for _, v := range variableVersions {
					mockSecretManager.On("Get", mock.Anything, v.Key, v.SecretData).Return(fmt.Sprintf("%s-plaintext", v.Key), nil)
				}
			}

			vars, err := service.GetRunVariables(auth.WithCaller(ctx, mockCaller), runID, test.includeSensitiveValues)
			if test.expectedErrorCode != "" {
				require.Error(t, err)
				require.Equal(t, test.expectedErrorCode, errors.ErrorCode(err))
			} else {
				require.NoError(t, err)
				require.Equal(t, test.expectedVariables, vars)
			}
		})
	}
}

func TestGetStateVersionsByRunIDs(t *testing.T) {
	workspaceID := "ws1"

	type testCase struct {
		authError       error
		name            string
		expectErrorCode errors.CodeType
		runIDs          []string
	}

	testCases := []testCase{
		{
			name:   "get state versions by run ids",
			runIDs: []string{"run1", "run2"},
		},
		{
			name: "no run ids",
		},
		{
			name:            "subject does not have permission to view run",
			runIDs:          []string{"run1"},
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockRuns := db.NewMockRuns(t)
			mockStateVersions := db.NewMockStateVersions(t)

			runsCount := len(test.runIDs)

			mockRuns.On("GetRuns", mock.Anything, &db.GetRunsInput{
				Filter: &db.RunFilter{
					RunIDs: test.runIDs,
				},
			}).Return(func(_ context.Context, _ *db.GetRunsInput) (*db.RunsResult, error) {
				// Create runs
				runs := make([]*models.Run, runsCount)
				for i := 0; i < runsCount; i++ {
					runs[i] = &models.Run{
						Metadata: models.ResourceMetadata{
							ID: test.runIDs[i],
						},
						WorkspaceID: workspaceID,
					}
				}

				return &db.RunsResult{
					Runs: runs,
					PageInfo: &pagination.PageInfo{
						TotalCount: pagination.StaticCount(int32(runsCount)),
						HasResults: runsCount > 0,
					},
				}, nil
			})

			if runsCount > 0 {
				mockCaller.On("RequirePermission", mock.Anything, models.ViewRunPermission, mock.Anything, mock.Anything).Return(test.authError).Times(runsCount)

				if test.authError == nil {
					mockStateVersions.On("GetStateVersions", mock.Anything, &db.GetStateVersionsInput{
						Filter: &db.StateVersionFilter{
							RunIDs: test.runIDs,
						},
					}).Return(func(_ context.Context, _ *db.GetStateVersionsInput) (*db.StateVersionsResult, error) {
						// Create state versions
						stateVersions := make([]models.StateVersion, runsCount)
						for i := 0; i < runsCount; i++ {
							stateVersions[i] = models.StateVersion{
								Metadata: models.ResourceMetadata{
									ID: fmt.Sprintf("sv%d", i),
								},
								WorkspaceID: workspaceID,
								RunID:       &test.runIDs[i],
							}
						}

						return &db.StateVersionsResult{
							StateVersions: stateVersions,
							PageInfo: &pagination.PageInfo{
								TotalCount: pagination.StaticCount(int32(runsCount)),
							},
						}, nil
					})
				}
			}

			dbClient := &db.Client{
				Runs:          mockRuns,
				StateVersions: mockStateVersions,
			}

			service := &service{
				dbClient: dbClient,
			}

			result, err := service.GetStateVersionsByRunIDs(auth.WithCaller(ctx, mockCaller), test.runIDs)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Len(t, result, runsCount)
		})
	}
}

func TestGetRuns(t *testing.T) {
	workspace := &models.Workspace{
		Metadata: models.ResourceMetadata{
			ID: "workspace-1",
		},
	}

	group := &models.Group{
		Metadata: models.ResourceMetadata{
			ID: "group-1",
		},
	}

	userID := "userID"

	type testCase struct {
		authError       error
		input           *GetRunsInput
		name            string
		expectErrorCode errors.CodeType
		isAdmin         bool
	}

	testCases := []testCase{
		{
			name: "filter by workspace",
			input: &GetRunsInput{
				Workspace: workspace,
			},
		},
		{
			name: "filter by group",
			input: &GetRunsInput{
				Group: group,
			},
		},
		{
			name:    "admin user queries for all runs",
			input:   &GetRunsInput{},
			isAdmin: true,
		},
		{
			name: "caller does not have access to workspace",
			input: &GetRunsInput{
				Workspace: workspace,
			},
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
		{
			name: "caller does not have access to group",
			input: &GetRunsInput{
				Group: group,
			},
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockRuns := db.NewMockRuns(t)
			mockAuthorizer := auth.NewMockAuthorizer(t)
			mockMaintenanceMonitor := maintenance.NewMockMonitor(t)
			mockUsers := db.NewMockUsers(t)

			filter := &db.RunFilter{}

			rootNamespacePath := "root-namespace-path"

			switch {
			case test.input.Workspace != nil:
				filter.WorkspaceID = ptr.String(test.input.Workspace.Metadata.ID)
			case test.input.Group != nil:
				filter.GroupID = ptr.String(test.input.Group.Metadata.ID)
			default:
				if !test.isAdmin {
					filter.RootNamespaceMemberships = []models.MembershipNamespace{
						{ID: "root-namespace-1", Path: rootNamespacePath},
					}
					mockAuthorizer.On("GetRootNamespaces", mock.Anything).Return([]models.MembershipNamespace{
						{ID: "root-namespace-1", Path: rootNamespacePath},
					}, nil).Maybe()
				}
			}

			mockMaintenanceMonitor.On("InMaintenanceMode", mock.Anything).Return(false, nil).Maybe()
			mockAuthorizer.On("RequireAccess", mock.Anything, []models.Permission{models.ViewRunPermission}, mock.Anything).Return(test.authError).Maybe()

			if test.expectErrorCode == "" {
				mockRuns.On("GetRuns", mock.Anything, &db.GetRunsInput{
					Sort:              test.input.Sort,
					PaginationOptions: test.input.PaginationOptions,
					Filter:            filter,
				}).Return(&db.RunsResult{}, nil)
			}

			dbClient := &db.Client{
				Runs:  mockRuns,
				Users: mockUsers,
			}

			service := &service{
				dbClient: dbClient,
			}

			callerUser := &models.User{
				Metadata: models.ResourceMetadata{
					ID: userID,
				},
				Admin: test.isAdmin,
				AdminModeExpiration: func() *time.Time {
					if test.isAdmin {
						t := time.Now().Add(time.Hour)
						return &t
					}
					return nil
				}(),
			}

			mockUsers.On("GetUserByID", mock.Anything, callerUser.Metadata.ID).Return(callerUser, nil).Maybe()

			userCaller := auth.NewUserCaller(
				callerUser,
				mockAuthorizer,
				dbClient,
				mockMaintenanceMonitor,
				nil,
			)

			runsResult, err := service.GetRuns(auth.WithCaller(ctx, userCaller), test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, errors.ErrorCode(err), test.expectErrorCode)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, runsResult)
		})
	}
}

func TestGetPlanDiff(t *testing.T) {
	workspaceID := "ws1"
	runID := "run1"

	run := &models.Run{
		Metadata: models.ResourceMetadata{
			ID: runID,
		},
		WorkspaceID: workspaceID,
		Plan:        models.Plan{ID: "plan-1", DiffObjectStoreKey: ptr.String("workspaces/ws1/runs/run1/plan/diff.json")},
	}

	planID := run.Plan.GetID()

	type testCase struct {
		authError       error
		name            string
		expectErrorCode errors.CodeType
		expectedDiff    *plan.Diff
	}

	testCases := []testCase{
		{
			name:            "subject does not have permission to view run",
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
		{
			name:         "get plan diff",
			expectedDiff: &plan.Diff{},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockRuns := db.NewMockRuns(t)
			mockArtifactStore := workspace.NewMockArtifactStore(t)

			mockRuns.On("GetRunByNodeID", mock.Anything, planID).Return(run, nil)

			mockCaller.On("RequirePermission", mock.Anything, models.ViewRunPermission, mock.Anything, mock.Anything).Return(test.authError)

			planDiffBuf, err := json.Marshal(test.expectedDiff)
			require.NoError(t, err)

			mockArtifactStore.On("GetPlanDiff", mock.Anything, run).Return(io.NopCloser(bytes.NewReader(planDiffBuf)), nil).Maybe()

			dbClient := &db.Client{
				Runs: mockRuns,
			}

			service := &service{
				dbClient:      dbClient,
				artifactStore: mockArtifactStore,
			}

			actualDiff, err := service.GetPlanDiff(auth.WithCaller(ctx, mockCaller), planID)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)

			assert.Equal(t, test.expectedDiff, actualDiff)
		})
	}
}

func TestGetPlanCheckResults(t *testing.T) {
	workspaceID := "ws1"
	runID := "run1"
	planID := "plan-1"

	run := &models.Run{
		Metadata: models.ResourceMetadata{
			ID: runID,
		},
		WorkspaceID: workspaceID,
		Plan:        models.Plan{ID: planID, JSONObjectStoreKey: ptr.String("workspaces/ws1/runs/run1/plan/plan.json")},
	}

	type testCase struct {
		name            string
		authError       error
		runError        error
		run             *models.Run
		artifactError   error
		planJSON        string
		skipCaller      bool
		expectResult    []corerun.CheckResult
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:            "auth failure",
			skipCaller:      true,
			expectErrorCode: errors.EUnauthorized,
		},
		{
			name:            "permission denied",
			run:             run,
			authError:       errors.New("forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
		{
			name:            "run not found",
			run:             nil,
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "failed to get run by plan ID",
			runError:        errors.New("db error", errors.WithErrorCode(errors.EInternal)),
			expectErrorCode: errors.EInternal,
		},
		{
			name:            "artifact store error",
			run:             run,
			artifactError:   errors.New("store error", errors.WithErrorCode(errors.EInternal)),
			expectErrorCode: errors.EInternal,
		},
		{
			name:            "invalid plan JSON",
			run:             run,
			planJSON:        "not-json",
			expectErrorCode: errors.EInternal,
		},
		{
			name:         "empty checks",
			run:          run,
			planJSON:     `{"format_version":"1.0","checks":[]}`,
			expectResult: []corerun.CheckResult{},
		},
		{
			name:         "no checks field",
			run:          run,
			planJSON:     `{"format_version":"1.0"}`,
			expectResult: []corerun.CheckResult{},
		},
		{
			name: "check with failure messages",
			run:  run,
			planJSON: `{"format_version":"1.0","checks":[
				{"address":{"to_display":"check.health","kind":"check"},"status":"fail",
				 "instances":[{"address":{"to_display":"check.health"},"status":"fail",
				   "problems":[{"message":"Service returned 503"}]}]}
			]}`,
			expectResult: []corerun.CheckResult{
				{
					Name:   "check.health",
					Status: "fail",
					Objects: []corerun.CheckResultObject{
						{Address: "check.health", Status: "fail", FailureMessages: []string{"Service returned 503"}},
					},
				},
			},
		},
		{
			name: "check with multiple instances preserves per-object detail",
			run:  run,
			planJSON: `{"format_version":"1.0","checks":[
				{"address":{"to_display":"check.multi","kind":"check"},"status":"fail",
				 "instances":[
				   {"address":{"to_display":"check.multi[0]"},"status":"fail","problems":[{"message":"First failed"}]},
				   {"address":{"to_display":"check.multi[1]"},"status":"fail","problems":[{"message":"Second failed"}]}
				 ]}
			]}`,
			expectResult: []corerun.CheckResult{
				{
					Name:   "check.multi",
					Status: "fail",
					Objects: []corerun.CheckResultObject{
						{Address: "check.multi[0]", Status: "fail", FailureMessages: []string{"First failed"}},
						{Address: "check.multi[1]", Status: "fail", FailureMessages: []string{"Second failed"}},
					},
				},
			},
		},
		{
			name: "mixed instance results - one pass one fail",
			run:  run,
			planJSON: `{"format_version":"1.0","checks":[
				{"address":{"to_display":"null_resource.web","kind":"resource"},"status":"fail",
				 "instances":[
				   {"address":{"to_display":"null_resource.web[0]"},"status":"pass"},
				   {"address":{"to_display":"null_resource.web[1]"},"status":"fail","problems":[{"message":"port too low"}]}
				 ]}
			]}`,
			expectResult: []corerun.CheckResult{
				{
					Name:   "null_resource.web",
					Status: "fail",
					Objects: []corerun.CheckResultObject{
						{Address: "null_resource.web[0]", Status: "pass", FailureMessages: nil},
						{Address: "null_resource.web[1]", Status: "fail", FailureMessages: []string{"port too low"}},
					},
				},
			},
		},
		{
			name: "passing check has no failure messages",
			run:  run,
			planJSON: `{"format_version":"1.0","checks":[
				{"address":{"to_display":"check.cert","kind":"check"},"status":"pass",
				 "instances":[{"address":{"to_display":"check.cert"},"status":"pass"}]}
			]}`,
			expectResult: []corerun.CheckResult{
				{
					Name:   "check.cert",
					Status: "pass",
					Objects: []corerun.CheckResultObject{
						{Address: "check.cert", Status: "pass", FailureMessages: nil},
					},
				},
			},
		},
		{
			name: "unknown status at plan time",
			run:  run,
			planJSON: `{"format_version":"1.0","checks":[
				{"address":{"to_display":"check.pending","kind":"check"},"status":"unknown",
				 "instances":[{"address":{"to_display":"check.pending"},"status":"unknown"}]}
			]}`,
			expectResult: []corerun.CheckResult{
				{
					Name:   "check.pending",
					Status: "unknown",
					Objects: []corerun.CheckResultObject{
						{Address: "check.pending", Status: "unknown", FailureMessages: nil},
					},
				},
			},
		},
		{
			name: "unrecognized status normalized to unknown",
			run:  run,
			planJSON: `{"format_version":"1.0","checks":[
				{"address":{"to_display":"check.future","kind":"check"},"status":"skipped",
				 "instances":[{"address":{"to_display":"check.future"},"status":"skipped"}]}
			]}`,
			expectResult: []corerun.CheckResult{
				{
					Name:   "check.future",
					Status: "unknown",
					Objects: []corerun.CheckResultObject{
						{Address: "check.future", Status: "unknown", FailureMessages: nil},
					},
				},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockRuns := db.NewMockRuns(t)
			mockArtifactStore := workspace.NewMockArtifactStore(t)

			if !test.skipCaller {
				mockRuns.On("GetRunByNodeID", mock.Anything, planID).Return(test.run, test.runError)

				if test.run != nil && test.runError == nil {
					mockCaller.On("RequirePermission", mock.Anything, models.ViewRunPermission, mock.Anything, mock.Anything).Return(test.authError)
				}

				if test.authError == nil && test.run != nil && test.runError == nil {
					var reader io.ReadCloser
					if test.planJSON != "" {
						reader = io.NopCloser(strings.NewReader(test.planJSON))
					}
					mockArtifactStore.On("GetPlanJSON", mock.Anything, test.run).Return(reader, test.artifactError).Maybe()
				}
			}

			dbClient := &db.Client{
				Runs: mockRuns,
			}

			service := &service{
				dbClient:      dbClient,
				artifactStore: mockArtifactStore,
			}

			callCtx := ctx
			if !test.skipCaller {
				callCtx = auth.WithCaller(ctx, mockCaller)
			}

			result, err := service.GetPlanCheckResults(callCtx, planID)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.expectResult, result)
		})
	}
}

func TestUploadPlanBinary(t *testing.T) {
	workspaceID := "ws1"
	runID := "run1"

	run := &models.Run{
		Metadata:    models.ResourceMetadata{ID: runID},
		WorkspaceID: workspaceID,
		Plan:        models.Plan{ID: "plan-1"},
	}

	planID := run.Plan.GetID()
	cacheKey := "workspaces/ws1/runs/run1/plan/plan-1"

	type testCase struct {
		authError       error
		linkRefErr      error
		name            string
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:            "subject does not have permission to upload plan binary",
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
		{
			name: "upload plan binary",
		},
		{
			name:            "retainFn error is propagated",
			linkRefErr:      errors.New("link failed", errors.WithErrorCode(errors.EInternal)),
			expectErrorCode: errors.EInternal,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockCaller.On("RequirePermission", mock.Anything, models.UpdatePlanPermission, mock.Anything).Return(test.authError)

			mockRuns := db.NewMockRuns(t)
			mockRuns.On("GetRunByNodeID", mock.Anything, planID).Return(run, nil).Maybe()
			mockRuns.On("UpdateRun", mock.Anything, run, run.Plan.GetID()).Return(run, nil).Maybe()

			mockTransactions := db.NewMockTransactions(t)
			mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil).Maybe()
			mockTransactions.On("RollbackTx", mock.Anything).Return(nil).Maybe()
			mockTransactions.On("CommitTx", mock.Anything).Return(nil).Maybe()

			mockObjectStoreRefs := db.NewMockObjectStoreRefs(t)
			mockObjectStoreRefs.On("LinkRef", mock.Anything, cacheKey, db.ObjectStoreRefOwnerRun, runID).Return(test.linkRefErr).Maybe()

			mockArtifactStore := workspace.NewMockArtifactStore(t)
			mockArtifactStore.On("UploadPlanCache", mock.Anything, run, mock.Anything).
				Return(db.RetainObjectRefFunc(func(ctx context.Context, ownerID string) error {
					return mockObjectStoreRefs.LinkRef(ctx, cacheKey, db.ObjectStoreRefOwnerRun, ownerID)
				}), cacheKey, nil).Maybe()

			testLogger, _ := logger.NewForTest()

			service := &service{
				logger:        testLogger,
				dbClient:      &db.Client{Runs: mockRuns, Transactions: mockTransactions},
				artifactStore: mockArtifactStore,
			}

			err := service.UploadPlanBinary(auth.WithCaller(ctx, mockCaller), planID, strings.NewReader("test data"))

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, &cacheKey, run.Plan.CacheObjectStoreKey)
		})
	}
}

func TestProcessPlanData(t *testing.T) {
	workspaceID := "ws1"
	runID := "run1"

	run := &models.Run{
		Metadata: models.ResourceMetadata{
			ID: runID,
		},
		WorkspaceID: workspaceID,
		Plan:        models.Plan{ID: "plan-1"},
	}

	planID := run.Plan.GetID()

	type testCase struct {
		authError         error
		name              string
		expectErrorCode   errors.CodeType
		tfPlan            *tfjson.Plan
		tfProviderSchemas *tfjson.ProviderSchemas
	}

	testCases := []testCase{
		{
			name:            "subject does not have permission to update plan",
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
		{
			name: "process plan data",
			tfPlan: &tfjson.Plan{
				FormatVersion: "0.1",
				OutputChanges: map[string]*tfjson.Change{
					"test": {
						Actions: tfjson.Actions{tfjson.ActionCreate},
					},
				},
			},
			tfProviderSchemas: &tfjson.ProviderSchemas{
				FormatVersion: "0.1",
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)

			mockRuns := db.NewMockRuns(t)

			mockCaller.On("GetSubject").Return("testsubject").Maybe()

			mockCaller.On("RequirePermission", mock.Anything, models.UpdatePlanPermission, mock.Anything).Return(test.authError)

			dbClient := &db.Client{
				Runs: mockRuns,
			}

			mockCmdProcessor := engine.NewMockCmdProcessor(t)
			mockCmdProcessor.On("ProcessCommand", mock.Anything, mock.Anything).Return(nil).Maybe()

			testLogger, _ := logger.NewForTest()

			service := &service{
				logger:       testLogger,
				dbClient:     dbClient,
				cmdProcessor: mockCmdProcessor,
				cmdFactory:   &commands.Factory{},
			}

			err := service.ProcessPlanData(auth.WithCaller(ctx, mockCaller), planID, test.tfPlan, test.tfProviderSchemas)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestSubscribeToRunEvents(t *testing.T) {
	userID := "user1"
	rootNamespacePath := "root-namespace-path"

	// Test cases
	tests := []struct {
		authError      error
		input          *EventSubscriptionOptions
		name           string
		expectErrCode  errors.CodeType
		runner         *models.Runner
		sendEventData  []*db.RunEventData
		expectedEvents []Event
		// workspacePaths maps an event's workspace ID to the workspace's full path, used to
		// drive the non-admin root namespace membership access check. When nil, every workspace
		// resolves to rootNamespacePath (so the membership check passes).
		workspacePaths map[string]string
		isAdmin        bool
		useUserCaller  bool
		nilUserMember  bool
		nilWorkspaceID bool
	}{
		{
			name: "subscribe to run events for a workspace",
			input: &EventSubscriptionOptions{
				WorkspaceID: ptr.String("workspace1"),
			},
			sendEventData: []*db.RunEventData{
				{
					ID:          "run1",
					WorkspaceID: "workspace1",
				},
				{
					ID:          "run2",
					WorkspaceID: "workspace1",
				},
			},
			expectedEvents: []Event{
				{
					Run: &models.Run{
						Metadata: models.ResourceMetadata{
							ID: "run1",
						},
					},
					Action: "UPDATE",
				},
				{
					Run: &models.Run{
						Metadata: models.ResourceMetadata{
							ID: "run2",
						},
					},
					Action: "UPDATE",
				},
			},
		},
		{
			name: "not authorized to subscribe to run events for a workspace",
			input: &EventSubscriptionOptions{
				WorkspaceID: ptr.String("workspace1"),
			},
			authError:     errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrCode: errors.EForbidden,
		},
		{
			name: "subscribe to run events for a run",
			input: &EventSubscriptionOptions{
				RunID: ptr.String("run1"),
			},
			useUserCaller:  true,
			nilWorkspaceID: true,
			sendEventData: []*db.RunEventData{
				{
					ID: "run1",
				},
				{
					ID: "run2",
				},
			},
			expectedEvents: []Event{
				{
					Run: &models.Run{
						Metadata: models.ResourceMetadata{
							ID: "run1",
						},
					},
					Action: "UPDATE",
				},
			},
			runner: &models.Runner{
				Metadata: models.ResourceMetadata{ID: "runner1"},
				Type:     models.GroupRunnerType,
				GroupID:  ptr.String("group1"),
			},
		},
		{
			name: "not authorized to subscribe to run events for a run",
			input: &EventSubscriptionOptions{
				RunID: ptr.String("run1"),
			},
			runner: &models.Runner{
				Metadata: models.ResourceMetadata{ID: "runner1"},
				Type:     models.GroupRunnerType,
				GroupID:  ptr.String("group1"),
			},
			authError:     errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrCode: errors.EForbidden,
		},
		{
			name:    "subscribe to all run events",
			input:   &EventSubscriptionOptions{},
			isAdmin: true,
			sendEventData: []*db.RunEventData{
				{
					ID: "run1",
				},
				{
					ID: "run2",
				},
			},
			useUserCaller:  true,
			nilUserMember:  true,
			nilWorkspaceID: true,
			expectedEvents: []Event{
				{
					Run: &models.Run{
						Metadata: models.ResourceMetadata{
							ID: "run1",
						},
					},
					Action: "UPDATE",
				},
				{
					Run: &models.Run{
						Metadata: models.ResourceMetadata{
							ID: "run2",
						},
					},
					Action: "UPDATE",
				},
			},
		},
		{
			name:          "not authorized to subscribe to all run events",
			input:         &EventSubscriptionOptions{},
			expectErrCode: errors.EForbidden,
		},
		{
			name:           "non-admin user only receives runs in workspaces under their root namespace memberships",
			input:          &EventSubscriptionOptions{},
			useUserCaller:  true,
			nilWorkspaceID: true,
			sendEventData: []*db.RunEventData{
				// run1's workspace is a descendant of the caller's root namespace, so it is delivered.
				{
					ID:          "run1",
					WorkspaceID: "workspace1",
				},
				// run2's workspace is outside the caller's root namespace, so it is filtered out.
				{
					ID:          "run2",
					WorkspaceID: "workspace2",
				},
			},
			workspacePaths: map[string]string{
				"workspace1": rootNamespacePath + "/child",
				"workspace2": "other-root/child",
			},
			expectedEvents: []Event{
				{
					Run: &models.Run{
						Metadata: models.ResourceMetadata{
							ID: "run1",
						},
					},
					Action: "UPDATE",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockRunners := db.NewMockRunners(t)
			mockRuns := db.NewMockRuns(t)
			mockWorkspaces := db.NewMockWorkspaces(t)
			mockEvents := db.NewMockEvents(t)
			mockUsers := db.NewMockUsers(t)

			mockAuthorizer := auth.NewMockAuthorizer(t)
			mockMaintenanceMonitor := maintenance.NewMockMonitor(t)

			mockEventChannel := make(chan db.Event, 1)
			var roEventChan <-chan db.Event = mockEventChannel
			mockEvents.On("Listen", mock.Anything).Return(roEventChan, make(<-chan error)).Maybe()

			if test.input.WorkspaceID != nil {
				mockCaller.On("RequirePermission", mock.Anything, models.ViewRunPermission, mock.Anything).
					Return(test.authError)
			}

			for _, d := range test.sendEventData {
				dCopy := d

				mockRuns.On("GetRunByID", mock.Anything, dCopy.ID).
					Return(&models.Run{
						Metadata: models.ResourceMetadata{
							ID: dCopy.ID,
						},
					}, nil).Maybe()
			}

			if test.useUserCaller && !test.nilUserMember {
				// Non-admin user callers verify each run's workspace path against their root
				// namespace memberships before delivering the event.
				if test.workspacePaths != nil {
					for wsID, fullPath := range test.workspacePaths {
						mockWorkspaces.On("GetWorkspaceByID", mock.Anything, wsID).
							Return(&models.Workspace{
								FullPath: fullPath,
							}, nil).Maybe()
					}
				} else {
					// The workspace path matches the membership path so the access check passes.
					mockWorkspaces.On("GetWorkspaceByID", mock.Anything, mock.Anything).
						Return(&models.Workspace{
							FullPath: rootNamespacePath,
						}, nil).Maybe()
				}
			}

			dbClient := db.Client{
				Runners:    mockRunners,
				Runs:       mockRuns,
				Workspaces: mockWorkspaces,
				Events:     mockEvents,
				Users:      mockUsers,
			}

			logger, _ := logger.NewForTest()
			eventManager := events.NewEventManager(&dbClient, logger)
			eventManager.Start(ctx)

			service := &service{
				dbClient:     &dbClient,
				eventManager: eventManager,
				logger:       logger,
			}

			var useCaller auth.Caller = mockCaller
			if test.useUserCaller {
				callerUser := &models.User{
					Metadata: models.ResourceMetadata{
						ID: userID,
					},
					Admin: test.isAdmin,
					AdminModeExpiration: func() *time.Time {
						if test.isAdmin {
							t := time.Now().Add(time.Hour)
							return &t
						}
						return nil
					}(),
				}

				mockUsers.On("GetUserByID", mock.Anything, callerUser.Metadata.ID).Return(callerUser, nil).Maybe()

				// Non-admin user callers querying without a workspace/group resolve their root
				// namespace memberships.
				mockAuthorizer.On("GetRootNamespaces", mock.Anything).Return([]models.MembershipNamespace{
					{ID: "root-namespace-1", Path: rootNamespacePath},
				}, nil).Maybe()

				useCaller = auth.NewUserCaller(
					callerUser,
					mockAuthorizer,
					&dbClient,
					mockMaintenanceMonitor,
					nil,
				)
			}

			eventChannel, err := service.SubscribeToRunEvents(auth.WithCaller(ctx, useCaller), test.input)

			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			receivedEvents := []*Event{}

			go func() {
				for _, d := range test.sendEventData {
					encoded, err := json.Marshal(d)
					require.Nil(t, err)

					mockEventChannel <- db.Event{
						Table:  "runs",
						Action: "UPDATE",
						ID:     d.ID,
						Data:   encoded,
					}
				}
			}()

			if len(test.expectedEvents) > 0 {
				for e := range eventChannel {
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

func TestCallerHasRootNamespaceAccess(t *testing.T) {
	roots := []models.MembershipNamespace{
		{ID: "ns-1", Path: "group-a"},
		{ID: "ns-2", Path: "group-b/sub"},
	}

	tests := []struct {
		name                     string
		workspacePath            string
		rootNamespaceMemberships []models.MembershipNamespace
		expectAccess             bool
	}{
		{
			name:                     "exact match on a root membership",
			workspacePath:            "group-a",
			rootNamespaceMemberships: roots,
			expectAccess:             true,
		},
		{
			name:                     "immediate descendant of a root membership",
			workspacePath:            "group-a/workspace",
			rootNamespaceMemberships: roots,
			expectAccess:             true,
		},
		{
			name:                     "deeply nested descendant of a root membership",
			workspacePath:            "group-a/sub/deeper/workspace",
			rootNamespaceMemberships: roots,
			expectAccess:             true,
		},
		{
			name:                     "exact match on a nested root membership",
			workspacePath:            "group-b/sub",
			rootNamespaceMemberships: roots,
			expectAccess:             true,
		},
		{
			name:                     "descendant of a nested root membership",
			workspacePath:            "group-b/sub/workspace",
			rootNamespaceMemberships: roots,
			expectAccess:             true,
		},
		{
			name:                     "unrelated path is denied",
			workspacePath:            "group-c/workspace",
			rootNamespaceMemberships: roots,
			expectAccess:             false,
		},
		{
			name:                     "path with a matching prefix but no segment boundary is denied",
			workspacePath:            "group-a-other",
			rootNamespaceMemberships: roots,
			expectAccess:             false,
		},
		{
			name:                     "ancestor of a root membership is denied",
			workspacePath:            "group-b",
			rootNamespaceMemberships: roots,
			expectAccess:             false,
		},
		{
			name:                     "empty memberships deny all access",
			workspacePath:            "group-a",
			rootNamespaceMemberships: []models.MembershipNamespace{},
			expectAccess:             false,
		},
		{
			name:                     "nil memberships deny all access",
			workspacePath:            "group-a",
			rootNamespaceMemberships: nil,
			expectAccess:             false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expectAccess, callerHasRootNamespaceAccess(test.workspacePath, test.rootNamespaceMemberships))
		})
	}
}

func TestSetVariablesIncludedInTFConfig(t *testing.T) {
	runID := "run-1"

	type testCase struct {
		name            string
		run             *models.Run
		authError       error
		linkRefErr      error
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name: "set run variables",
			run: &models.Run{
				Metadata: models.ResourceMetadata{
					ID: runID,
				},
				Plan: models.Plan{ID: "plan-1"},
			},
		},
		{
			name: "not authorized to set run variables",
			run: &models.Run{
				Metadata: models.ResourceMetadata{
					ID: runID,
				},
				Plan: models.Plan{ID: "plan-1"},
			},
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
		{
			name:            "run not found",
			expectErrorCode: errors.ENotFound,
		},
		{
			name: "retainFn error is propagated",
			run: &models.Run{
				Metadata: models.ResourceMetadata{
					ID: runID,
				},
				Plan: models.Plan{ID: "plan-1"},
			},
			linkRefErr:      errors.New("link failed", errors.WithErrorCode(errors.EInternal)),
			expectErrorCode: errors.EInternal,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockRuns := db.NewMockRuns(t)
			mockCaller := auth.NewMockCaller(t)
			mockArtifactStore := workspace.NewMockArtifactStore(t)

			sampleVariables := []runvariables.Variable{
				{
					Key:           "my_var",
					Value:         ptr.String("my value"),
					NamespacePath: ptr.String("group/workspace"),
					Category:      models.TerraformVariableCategory,
				},
				{
					Key:      "my_var2",
					Value:    ptr.String("my value2"),
					Category: models.TerraformVariableCategory,
				},
				{
					Key:      "my_var",
					Value:    ptr.String("my value"),
					Category: models.EnvironmentVariableCategory,
				},
			}

			mockRuns.On("GetRunByID", mock.Anything, runID).Return(tc.run, nil)

			if tc.run != nil {
				mockCaller.On("RequirePermission", mock.Anything, models.UpdatePlanPermission, mock.Anything).Return(tc.authError)

				if tc.authError == nil {
					data, err := json.Marshal(sampleVariables)
					require.NoError(t, err)

					mockArtifactStore.On("GetRunVariables", mock.Anything, tc.run).Return(io.NopCloser(bytes.NewReader(data)), nil)

					// Should only mark first variable as used.
					sampleVariables[0].IncludedInTFConfig = ptr.Bool(true)
					sampleVariables[1].IncludedInTFConfig = ptr.Bool(false)

					data, err = json.Marshal(sampleVariables)
					require.NoError(t, err)

					mockArtifactStore.On("UploadRunVariables", mock.Anything, tc.run, bytes.NewReader(data)).
						Return(db.RetainObjectRefFunc(func(_ context.Context, _ string) error { return tc.linkRefErr }), "workspaces/ws1/runs/run1/variables.json", nil)
				}
			}

			dbClient := &db.Client{
				Runs: mockRuns,
			}

			service := &service{
				dbClient:      dbClient,
				artifactStore: mockArtifactStore,
			}

			err := service.SetVariablesIncludedInTFConfig(auth.WithCaller(ctx, mockCaller), &SetVariablesIncludedInTFConfigInput{
				RunID:        runID,
				VariableKeys: []string{"my_var"},
			})

			if tc.expectErrorCode != "" {
				assert.Equal(t, tc.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestUpdateApply(t *testing.T) {
	applyID := "apply-1"

	testCases := []struct {
		name            string
		input           *UpdateApplyInput
		authError       error
		expectErrorCode errors.CodeType
	}{
		{
			name: "update apply with valid UTF-8 error message",
			input: &UpdateApplyInput{
				ApplyID:      applyID,
				ErrorMessage: ptr.String("Valid UTF-8 error message"),
			},
		},
		{
			name: "update apply with invalid UTF-8 error message gets sanitized",
			input: &UpdateApplyInput{
				ApplyID:      applyID,
				ErrorMessage: ptr.String("Invalid UTF-8: \xff\xfe\xfd"),
			},
		},
		{
			name: "update apply with mixed valid and invalid UTF-8",
			input: &UpdateApplyInput{
				ApplyID:      applyID,
				ErrorMessage: ptr.String("Valid text \xff invalid \xfe more valid text"),
			},
		},
		{
			name: "update apply without permission",
			input: &UpdateApplyInput{
				ApplyID:      applyID,
				ErrorMessage: ptr.String("Error message"),
			},
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockCaller := auth.NewMockCaller(t)
			mockRuns := db.NewMockRuns(t)

			mockCaller.On("RequirePermission", mock.Anything, models.UpdateApplyPermission, mock.Anything).Return(test.authError)

			dbClient := &db.Client{
				Runs: mockRuns,
			}

			mockCmdProcessor := engine.NewMockCmdProcessor(t)
			if test.authError == nil {
				// Simulate the processor executing the command and populating its result.
				mockCmdProcessor.On("ProcessCommand", mock.Anything, mock.Anything).
					Run(func(args mock.Arguments) {
						cmd := args.Get(1).(*commands.UpdateApply)
						cmd.Updated = &models.Apply{ID: cmd.ApplyID, Status: models.ApplyErrored}
					}).Return(nil)
			}

			logger, _ := logger.NewForTest()
			service := &service{
				dbClient:     dbClient,
				cmdProcessor: mockCmdProcessor,
				cmdFactory:   &commands.Factory{},
				logger:       logger,
			}

			result, err := service.UpdateApply(auth.WithCaller(ctx, mockCaller), test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)
		})
	}
}

func TestUpdatePlan(t *testing.T) {
	planID := "plan-1"

	testCases := []struct {
		name            string
		input           *UpdatePlanInput
		authError       error
		expectErrorCode errors.CodeType
	}{
		{
			name: "update plan with valid UTF-8 error message",
			input: &UpdatePlanInput{
				PlanID:       planID,
				HasChanges:   true,
				ErrorMessage: ptr.String("Valid UTF-8 error message"),
			},
		},
		{
			name: "update plan with invalid UTF-8 error message gets sanitized",
			input: &UpdatePlanInput{
				PlanID:       planID,
				HasChanges:   false,
				ErrorMessage: ptr.String("Invalid UTF-8: \xff\xfe\xfd"),
			},
		},
		{
			name: "update plan with mixed valid and invalid UTF-8",
			input: &UpdatePlanInput{
				PlanID:       planID,
				HasChanges:   false,
				ErrorMessage: ptr.String("Valid text \xff invalid \xfe more valid text"),
			},
		},
		{
			name: "update plan without permission",
			input: &UpdatePlanInput{
				PlanID:       planID,
				HasChanges:   false,
				ErrorMessage: ptr.String("Error message"),
			},
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockCaller := auth.NewMockCaller(t)
			mockRuns := db.NewMockRuns(t)

			mockCaller.On("RequirePermission", mock.Anything, models.UpdatePlanPermission, mock.Anything).Return(test.authError)

			dbClient := &db.Client{
				Runs: mockRuns,
			}

			mockCmdProcessor := engine.NewMockCmdProcessor(t)
			if test.authError == nil {
				// Simulate the processor executing the command and populating its result.
				mockCmdProcessor.On("ProcessCommand", mock.Anything, mock.Anything).
					Run(func(args mock.Arguments) {
						cmd := args.Get(1).(*commands.UpdatePlan)
						cmd.Updated = &models.Plan{ID: cmd.PlanID, Status: models.PlanErrored, HasChanges: cmd.HasChanges}
					}).Return(nil)
			}

			logger, _ := logger.NewForTest()
			service := &service{
				dbClient:     dbClient,
				cmdProcessor: mockCmdProcessor,
				cmdFactory:   &commands.Factory{},
				logger:       logger,
			}

			result, err := service.UpdatePlan(auth.WithCaller(ctx, mockCaller), test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)
		})
	}
}
func TestCreateRunInputValidate(t *testing.T) {
	tests := []struct {
		name            string
		input           CreateRunInput
		expectError     string
		expectErrorCode errors.CodeType
	}{
		{
			name: "empty module version",
			input: CreateRunInput{
				ModuleSource:  ptr.String("test-source"),
				ModuleVersion: ptr.String(""),
			},
			expectError:     "module version cannot be empty; please specify a valid semantic version",
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "latest module version",
			input: CreateRunInput{
				ModuleSource:  ptr.String("test-source"),
				ModuleVersion: ptr.String("latest"),
			},
			expectError:     "'latest' is not a valid module version; please specify a valid semantic version",
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "valid exact module version",
			input: CreateRunInput{
				ModuleSource:  ptr.String("test-source"),
				ModuleVersion: ptr.String("1.0.0"),
			},
		},
		{
			name: "valid prerelease module version",
			input: CreateRunInput{
				ModuleSource:  ptr.String("test-source"),
				ModuleVersion: ptr.String("1.0.0-rc.1"),
			},
		},
		{
			name: "valid constraint expression",
			input: CreateRunInput{
				ModuleSource:  ptr.String("test-source"),
				ModuleVersion: ptr.String(">= 1.0.0"),
			},
		},
		{
			name: "valid constraint range",
			input: CreateRunInput{
				ModuleSource:  ptr.String("test-source"),
				ModuleVersion: ptr.String(">= 1.0.0, < 2.0.0"),
			},
		},
		{
			name: "invalid module version string",
			input: CreateRunInput{
				ModuleSource:  ptr.String("test-source"),
				ModuleVersion: ptr.String("not-a-version"),
			},
			expectError:     "module version is not a valid semver version or constraint expression",
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "valid v-prefixed module version",
			input: CreateRunInput{
				ModuleSource:  ptr.String("test-source"),
				ModuleVersion: ptr.String("v1.0.0"),
			},
		},
		{
			name: "valid constraint range with v-prefixed versions",
			input: CreateRunInput{
				ModuleSource:  ptr.String("test-source"),
				ModuleVersion: ptr.String(">= v1.0.0, < v2.0.0"),
			},
		},
		{
			name: "invalid v-prefixed operator string",
			input: CreateRunInput{
				ModuleSource:  ptr.String("test-source"),
				ModuleVersion: ptr.String("v>= 1.0.0"),
			},
			expectError:     "module version is not a valid semver version or constraint expression",
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.input.Validate()

			if test.expectError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), test.expectError)
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestRunCreationAuthorization verifies that the run-creation methods all deny
// callers that lack CreateRunPermission. Each case exercises a single method via
// its call closure.
func TestRunCreationAuthorization(t *testing.T) {
	type testCase struct {
		call            func(context.Context, *service) error
		authError       error
		name            string
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name: "CreateRun: subject is not authorized",
			call: func(ctx context.Context, s *service) error {
				_, err := s.CreateRun(ctx, &CreateRunInput{WorkspaceID: "workspace-1"})
				return err
			},
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
		{
			name: "CreateAssessmentRunForWorkspace: subject is not authorized",
			call: func(ctx context.Context, s *service) error {
				_, err := s.CreateAssessmentRunForWorkspace(ctx, &CreateAssessmentRunForWorkspaceInput{WorkspaceID: "workspace-1"})
				return err
			},
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
		{
			name: "CreateDestroyRunForWorkspace: subject is not authorized",
			call: func(ctx context.Context, s *service) error {
				_, err := s.CreateDestroyRunForWorkspace(ctx, &CreateDestroyRunForWorkspaceInput{WorkspaceID: "workspace-1"})
				return err
			},
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
		{
			name: "CreateReconcileRunForWorkspace: subject is not authorized",
			call: func(ctx context.Context, s *service) error {
				_, err := s.CreateReconcileRunForWorkspace(ctx, &CreateReconcileRunForWorkspaceInput{WorkspaceID: "workspace-1"})
				return err
			},
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
		{
			name: "CreateRun: subject is authorized",
			call: func(ctx context.Context, s *service) error {
				// A valid input needs exactly one of configuration version / module source.
				_, err := s.CreateRun(ctx, &CreateRunInput{WorkspaceID: "workspace-1", ConfigurationVersionID: ptr.String("cv-1")})
				return err
			},
		},
		{
			name: "CreateAssessmentRunForWorkspace: subject is authorized",
			call: func(ctx context.Context, s *service) error {
				_, err := s.CreateAssessmentRunForWorkspace(ctx, &CreateAssessmentRunForWorkspaceInput{WorkspaceID: "workspace-1"})
				return err
			},
		},
		{
			name: "CreateDestroyRunForWorkspace: subject is authorized",
			call: func(ctx context.Context, s *service) error {
				_, err := s.CreateDestroyRunForWorkspace(ctx, &CreateDestroyRunForWorkspaceInput{WorkspaceID: "workspace-1"})
				return err
			},
		},
		{
			name: "CreateReconcileRunForWorkspace: subject is authorized",
			call: func(ctx context.Context, s *service) error {
				_, err := s.CreateReconcileRunForWorkspace(ctx, &CreateReconcileRunForWorkspaceInput{WorkspaceID: "workspace-1"})
				return err
			},
		},
	}

	sampleRun := &models.Run{Metadata: models.ResourceMetadata{ID: "run-1"}, WorkspaceID: "workspace-1"}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockCaller := auth.NewMockCaller(t)
			mockCaller.On("RequirePermission", mock.Anything, models.CreateRunPermission, mock.Anything).Return(test.authError)
			// Some run-creation methods stamp the run with the caller's subject.
			mockCaller.On("GetSubject").Return("user@example.com").Maybe()

			testLogger, _ := logger.NewForTest()
			mockProcessor := engine.NewMockCmdProcessor(t)
			if test.authError == nil {
				// On the authorized path the command is dispatched; populate the
				// command's result the way the real processor would so the method
				// can read/log the created run.
				mockProcessor.On("ProcessCommand", mock.Anything, mock.Anything).
					Run(func(args mock.Arguments) {
						switch c := args.Get(1).(type) {
						case *commands.CreateRun:
							c.Created = sampleRun
						case *commands.CreateAssessmentRun:
							c.Created = sampleRun
						case *commands.CreateDestroyRun:
							c.Created = sampleRun
						case *commands.CreateReconcileRun:
							c.Created = sampleRun
						}
					}).Return(nil)
			}

			service := &service{
				logger:       testLogger,
				dbClient:     &db.Client{},
				cmdProcessor: mockProcessor,
				cmdFactory:   commands.NewFactory(testLogger, &db.Client{}, nil, nil, nil, "", nil, nil),
			}

			err := test.call(auth.WithCaller(context.Background(), mockCaller), service)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestRunMutationAuthorization(t *testing.T) {
	// Every run-mutation entrypoint authorizes through authorizeRunMutation, which
	// fetches the run and then requires CreateRunPermission on its workspace. A denied
	// caller must short-circuit before any command is processed.
	type testCase struct {
		call            func(context.Context, *service) error
		authError       error
		name            string
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name: "ApplyRun: subject is not authorized",
			call: func(ctx context.Context, s *service) error {
				_, err := s.ApplyRun(ctx, "run-1", nil)
				return err
			},
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
		{
			name: "SetRunAutoApply: subject is not authorized",
			call: func(ctx context.Context, s *service) error {
				_, err := s.SetRunAutoApply(ctx, "run-1", true)
				return err
			},
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
		{
			name: "CancelRun: subject is not authorized",
			call: func(ctx context.Context, s *service) error {
				_, err := s.CancelRun(ctx, &CancelRunInput{RunID: "run-1"})
				return err
			},
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
		{
			name: "RetryRunNode: subject is not authorized",
			call: func(ctx context.Context, s *service) error {
				_, err := s.RetryRunNode(ctx, &RetryRunNodeInput{RunID: "run-1"})
				return err
			},
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
		{
			name: "DiscardRun: subject is not authorized",
			call: func(ctx context.Context, s *service) error {
				_, err := s.DiscardRun(ctx, &DiscardRunInput{RunID: "run-1"})
				return err
			},
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
		{
			name: "UndiscardRun: subject is not authorized",
			call: func(ctx context.Context, s *service) error {
				_, err := s.UndiscardRun(ctx, &UndiscardRunInput{RunID: "run-1"})
				return err
			},
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
		{
			name: "ApplyRun: subject is authorized",
			call: func(ctx context.Context, s *service) error {
				_, err := s.ApplyRun(ctx, "run-1", nil)
				return err
			},
		},
		{
			name: "SetRunAutoApply: subject is authorized",
			call: func(ctx context.Context, s *service) error {
				_, err := s.SetRunAutoApply(ctx, "run-1", true)
				return err
			},
		},
		{
			name: "CancelRun: subject is authorized",
			call: func(ctx context.Context, s *service) error {
				_, err := s.CancelRun(ctx, &CancelRunInput{RunID: "run-1"})
				return err
			},
		},
		{
			name: "RetryRunNode: subject is authorized",
			call: func(ctx context.Context, s *service) error {
				_, err := s.RetryRunNode(ctx, &RetryRunNodeInput{RunID: "run-1"})
				return err
			},
		},
		{
			name: "DiscardRun: subject is authorized",
			call: func(ctx context.Context, s *service) error {
				_, err := s.DiscardRun(ctx, &DiscardRunInput{RunID: "run-1"})
				return err
			},
		},
		{
			name: "UndiscardRun: subject is authorized",
			call: func(ctx context.Context, s *service) error {
				_, err := s.UndiscardRun(ctx, &UndiscardRunInput{RunID: "run-1"})
				return err
			},
		},
	}

	sampleRun := &models.Run{Metadata: models.ResourceMetadata{ID: "run-1"}, WorkspaceID: "workspace-1", Status: models.RunApplied}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockRuns := db.NewMockRuns(t)
			// authorizeRunMutation fetches the run before checking the permission.
			mockRuns.On("GetRunByID", mock.Anything, "run-1").Return(sampleRun, nil)

			mockCaller := auth.NewMockCaller(t)
			mockCaller.On("RequirePermission", mock.Anything, models.CreateRunPermission, mock.Anything).Return(test.authError)
			// Some run-mutation methods stamp the command with the caller's subject.
			mockCaller.On("GetSubject").Return("user@example.com").Maybe()

			testLogger, _ := logger.NewForTest()
			mockProcessor := engine.NewMockCmdProcessor(t)
			if test.authError == nil {
				// On the authorized path the mutation command is dispatched; populate
				// its Updated result the way the real processor would.
				mockProcessor.On("ProcessCommand", mock.Anything, mock.Anything).
					Run(func(args mock.Arguments) {
						switch c := args.Get(1).(type) {
						case *commands.StartApply:
							c.Updated = sampleRun
						case *commands.SetRunAutoApply:
							c.Updated = sampleRun
						case *commands.CancelRun:
							c.Updated = sampleRun
						case *commands.RetryRunNode:
							c.Updated = sampleRun
						case *commands.DiscardRun:
							c.Updated = sampleRun
						case *commands.UndiscardRun:
							c.Updated = sampleRun
						}
					}).Return(nil)
			}

			service := &service{
				logger:       testLogger,
				dbClient:     &db.Client{Runs: mockRuns},
				cmdProcessor: mockProcessor,
				cmdFactory:   commands.NewFactory(testLogger, &db.Client{}, nil, nil, nil, "", nil, nil),
			}

			err := test.call(auth.WithCaller(context.Background(), mockCaller), service)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestGetRunByNodeID(t *testing.T) {
	sampleRun := &models.Run{
		Metadata:    models.ResourceMetadata{ID: "run-id-1"},
		WorkspaceID: "workspace-1",
	}

	type testCase struct {
		authError       error
		name            string
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:            "subject is not authorized",
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
		{
			name: "subject is authorized",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockCaller := auth.NewMockCaller(t)
			mockRuns := db.NewMockRuns(t)

			mockRuns.On("GetRunByNodeID", mock.Anything, "node-1").Return(sampleRun, nil)
			mockCaller.On("RequirePermission", mock.Anything, models.ViewRunPermission, mock.Anything, mock.Anything).Return(test.authError)

			service := &service{
				dbClient: &db.Client{Runs: mockRuns},
			}

			_, err := service.GetRunByNodeID(auth.WithCaller(context.Background(), mockCaller), "node-1")

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestGetRunsByIDs(t *testing.T) {
	sampleRun := &models.Run{
		Metadata:    models.ResourceMetadata{ID: "run-id-1"},
		WorkspaceID: "workspace-1",
	}

	type testCase struct {
		authError       error
		name            string
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:            "subject is not authorized",
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
		{
			name: "subject is authorized",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockCaller := auth.NewMockCaller(t)
			mockRuns := db.NewMockRuns(t)

			mockRuns.On("GetRuns", mock.Anything, mock.Anything).Return(&db.RunsResult{Runs: []*models.Run{sampleRun}}, nil)
			mockCaller.On("RequirePermission", mock.Anything, models.ViewRunPermission, mock.Anything, mock.Anything).Return(test.authError)

			service := &service{
				dbClient: &db.Client{Runs: mockRuns},
			}

			_, err := service.GetRunsByIDs(auth.WithCaller(context.Background(), mockCaller), []string{"run-id-1"})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestDownloadPlan(t *testing.T) {
	sampleRun := &models.Run{
		Metadata:    models.ResourceMetadata{ID: "run-id-1"},
		WorkspaceID: "workspace-1",
	}

	type testCase struct {
		authError       error
		name            string
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:            "subject is not authorized",
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
		{
			name: "subject is authorized",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockCaller := auth.NewMockCaller(t)
			mockRuns := db.NewMockRuns(t)
			mockArtifactStore := workspace.NewMockArtifactStore(t)

			mockRuns.On("GetRunByNodeID", mock.Anything, "plan-1").Return(sampleRun, nil)
			mockCaller.On("RequirePermission", mock.Anything, models.ViewRunPermission, mock.Anything, mock.Anything).Return(test.authError)
			if test.expectErrorCode == "" {
				// On the authorized path the plan cache is streamed from the artifact store.
				mockArtifactStore.On("GetPlanCache", mock.Anything, sampleRun).
					Return(io.NopCloser(strings.NewReader("plan-data")), nil)
			}

			service := &service{
				dbClient:      &db.Client{Runs: mockRuns},
				artifactStore: mockArtifactStore,
			}

			_, err := service.DownloadPlan(auth.WithCaller(context.Background(), mockCaller), "plan-1")

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
		})
	}
}
