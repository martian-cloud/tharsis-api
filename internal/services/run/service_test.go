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
		name                                     string
		includeSensitiveValues                   bool
		expectedVariables                        []runvariables.Variable
		hasViewSensitiveVariableValuePermissions bool
		authError                                error
		expectedErrorCode                        errors.CodeType
	}{
		{
			name:                                     "include sensitive values for caller with view variable value permission",
			includeSensitiveValues:                   true,
			hasViewSensitiveVariableValuePermissions: true,
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
			name:                                     "don't include sensitive values for caller with view variable value permission",
			includeSensitiveValues:                   false,
			hasViewSensitiveVariableValuePermissions: true,
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
			name:                                     "include non-sensitive values for caller without view variable value permission",
			includeSensitiveValues:                   false,
			hasViewSensitiveVariableValuePermissions: false,
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
			name:                                     "return error if caller without view variable value permission requests sensitive values",
			includeSensitiveValues:                   true,
			hasViewSensitiveVariableValuePermissions: false,
			expectedErrorCode:                        errors.EForbidden,
		},
		{
			name:                                     "return error if caller doesn't have view variable permission",
			hasViewSensitiveVariableValuePermissions: false,
			authError:                                errors.New("no permission", errors.WithErrorCode(errors.EForbidden)),
			expectedErrorCode:                        errors.EForbidden,
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

			if test.hasViewSensitiveVariableValuePermissions {
				mockCaller.On("RequirePermission", mock.Anything, models.ViewSensitiveVariableValuePermission, mock.Anything).Return(nil)
			} else {
				mockCaller.On("RequirePermission", mock.Anything, models.ViewSensitiveVariableValuePermission, mock.Anything).Return(errors.New("no permission", errors.WithErrorCode(errors.EForbidden)))
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

func TestGetRunGatesByIDs(t *testing.T) {
	type testCase struct {
		authError       error
		name            string
		expectErrorCode errors.CodeType
		runGateIDs      []string
		gates           []models.RunGate
		// expectPermissionChecks is the number of RequirePermission calls expected: one per distinct
		// run across the returned gates, not one per gate.
		expectPermissionChecks int
	}

	testCases := []testCase{
		{
			name:       "gates on the same run are authorized once",
			runGateIDs: []string{"gate1", "gate2"},
			gates: []models.RunGate{
				{Metadata: models.ResourceMetadata{ID: "gate1"}, RunID: "run1", WorkspaceID: "ws1"},
				{Metadata: models.ResourceMetadata{ID: "gate2"}, RunID: "run1", WorkspaceID: "ws1"},
			},
			expectPermissionChecks: 1,
		},
		{
			name:       "gates spanning two runs are authorized per run",
			runGateIDs: []string{"gate1", "gate2"},
			gates: []models.RunGate{
				{Metadata: models.ResourceMetadata{ID: "gate1"}, RunID: "run1", WorkspaceID: "ws1"},
				{Metadata: models.ResourceMetadata{ID: "gate2"}, RunID: "run2", WorkspaceID: "ws2"},
			},
			expectPermissionChecks: 2,
		},
		{
			name:       "no matching gates authorizes nothing",
			runGateIDs: []string{"gate1"},
		},
		{
			name:       "subject does not have permission to view run",
			runGateIDs: []string{"gate1"},
			gates: []models.RunGate{
				{Metadata: models.ResourceMetadata{ID: "gate1"}, RunID: "run1", WorkspaceID: "ws1"},
			},
			expectPermissionChecks: 1,
			authError:              errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode:        errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockRunGates := db.NewMockRunGates(t)

			mockRunGates.On("GetRunGates", mock.Anything, &db.GetRunGatesInput{
				Filter: &db.RunGateFilter{
					RunGateIDs: test.runGateIDs,
				},
			}).Return(&db.RunGatesResult{
				RunGates: test.gates,
				PageInfo: &pagination.PageInfo{
					TotalCount: pagination.StaticCount(int32(len(test.gates))),
					HasResults: len(test.gates) > 0,
				},
			}, nil)

			if test.expectPermissionChecks > 0 {
				mockCaller.On("RequirePermission", mock.Anything, models.ViewRunPermission, mock.Anything, mock.Anything).
					Return(test.authError).Times(test.expectPermissionChecks)
			}

			service := &service{
				dbClient: &db.Client{RunGates: mockRunGates},
			}

			result, err := service.GetRunGatesByIDs(auth.WithCaller(ctx, mockCaller), test.runGateIDs)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Len(t, result, len(test.gates))
			for i := range result {
				assert.Equal(t, test.gates[i].Metadata.ID, result[i].Metadata.ID)
			}
		})
	}
}

func TestGetRunGatesByPolicyCheckIDs(t *testing.T) {
	type testCase struct {
		authError       error
		name            string
		expectErrorCode errors.CodeType
		policyCheckIDs  []string
		gates           []models.RunGate
		// expectPermissionChecks is the number of RequirePermission calls expected: one per distinct
		// run across the returned gates, not one per gate.
		expectPermissionChecks int
	}

	testCases := []testCase{
		{
			name:           "gates for checks on the same run are authorized once",
			policyCheckIDs: []string{"check1", "check2"},
			gates: []models.RunGate{
				{Metadata: models.ResourceMetadata{ID: "gate1"}, RunID: "run1", WorkspaceID: "ws1", PolicyCheckID: "check1"},
				{Metadata: models.ResourceMetadata{ID: "gate2"}, RunID: "run1", WorkspaceID: "ws1", PolicyCheckID: "check2"},
			},
			expectPermissionChecks: 1,
		},
		{
			name:           "gates spanning two runs are authorized per run",
			policyCheckIDs: []string{"check1", "check2"},
			gates: []models.RunGate{
				{Metadata: models.ResourceMetadata{ID: "gate1"}, RunID: "run1", WorkspaceID: "ws1", PolicyCheckID: "check1"},
				{Metadata: models.ResourceMetadata{ID: "gate2"}, RunID: "run2", WorkspaceID: "ws2", PolicyCheckID: "check2"},
			},
			expectPermissionChecks: 2,
		},
		{
			name:           "checks with no gates authorize nothing",
			policyCheckIDs: []string{"check1"},
		},
		{
			name:           "subject does not have permission to view run",
			policyCheckIDs: []string{"check1"},
			gates: []models.RunGate{
				{Metadata: models.ResourceMetadata{ID: "gate1"}, RunID: "run1", WorkspaceID: "ws1", PolicyCheckID: "check1"},
			},
			expectPermissionChecks: 1,
			authError:              errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode:        errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockRunGates := db.NewMockRunGates(t)

			mockRunGates.On("GetRunGates", mock.Anything, &db.GetRunGatesInput{
				Filter: &db.RunGateFilter{
					PolicyCheckIDs: test.policyCheckIDs,
				},
			}).Return(&db.RunGatesResult{
				RunGates: test.gates,
				PageInfo: &pagination.PageInfo{
					TotalCount: pagination.StaticCount(int32(len(test.gates))),
					HasResults: len(test.gates) > 0,
				},
			}, nil)

			if test.expectPermissionChecks > 0 {
				mockCaller.On("RequirePermission", mock.Anything, models.ViewRunPermission, mock.Anything, mock.Anything).
					Return(test.authError).Times(test.expectPermissionChecks)
			}

			service := &service{
				dbClient: &db.Client{RunGates: mockRunGates},
			}

			result, err := service.GetRunGatesByPolicyCheckIDs(auth.WithCaller(ctx, mockCaller), test.policyCheckIDs)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Len(t, result, len(test.gates))
			for i := range result {
				assert.Equal(t, test.gates[i].Metadata.ID, result[i].Metadata.ID)
			}
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

func TestGetPolicyCheckPolicyMessages(t *testing.T) {
	workspaceID := "ws1"
	runID := "run1"
	checkID := "check-1"

	// runWith builds a run whose post-plan check pins one policy, with the given messages key on it.
	runWith := func(key *string) *models.Run {
		return &models.Run{
			Metadata:    models.ResourceMetadata{ID: runID},
			WorkspaceID: workspaceID,
			TaskStages: []*models.RunTaskStage{{
				ID:        "stage-1",
				StageName: models.RunTaskStageNamePostPlan,
				PolicyChecks: []*models.PolicyCheck{{
					ID:        checkID,
					StageName: models.RunTaskStageNamePostPlan,
					CheckType: models.PolicyKindOPA,
					Policies: []*models.PolicyCheckPolicy{
						{ID: "pol-1", MessagesObjectStoreKey: key},
					},
				}},
			}},
		}
	}

	messagesKey := ptr.String("workspaces/ws1/runs/run1/policy_messages/obj-1.json")

	type testCase struct {
		name            string
		authError       error
		runError        error
		run             *models.Run
		policyID        string
		artifactError   error
		object          string
		skipCaller      bool
		expectResult    []string
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:            "auth failure",
			skipCaller:      true,
			policyID:        "pol-1",
			expectErrorCode: errors.EUnauthorized,
		},
		{
			name:            "permission denied",
			run:             runWith(messagesKey),
			policyID:        "pol-1",
			authError:       errors.New("forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
		{
			name:            "run not found",
			run:             nil,
			policyID:        "pol-1",
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "failed to get run by policy check ID",
			runError:        errors.New("db error", errors.WithErrorCode(errors.EInternal)),
			policyID:        "pol-1",
			expectErrorCode: errors.EInternal,
		},
		{
			name:            "policy not pinned by the check",
			run:             runWith(messagesKey),
			policyID:        "pol-missing",
			expectErrorCode: errors.ENotFound,
		},
		{
			// A policy that reported nothing has no key, which is an empty list and no object read —
			// the mock would fail the test if one were attempted.
			name:         "no messages reported",
			run:          runWith(nil),
			policyID:     "pol-1",
			expectResult: []string{},
		},
		{
			name:            "artifact store error",
			run:             runWith(messagesKey),
			policyID:        "pol-1",
			artifactError:   errors.New("store error", errors.WithErrorCode(errors.EInternal)),
			expectErrorCode: errors.EInternal,
		},
		{
			name:            "invalid stored JSON",
			run:             runWith(messagesKey),
			policyID:        "pol-1",
			object:          "not-json",
			expectErrorCode: errors.EInternal,
		},
		{
			name:         "messages are returned in stored order",
			run:          runWith(messagesKey),
			policyID:     "pol-1",
			object:       `["denied by rule X","denied by rule Y"]`,
			expectResult: []string{"denied by rule X", "denied by rule Y"},
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
				mockRuns.On("GetRunByNodeID", mock.Anything, checkID).Return(test.run, test.runError)

				if test.run != nil && test.runError == nil {
					mockCaller.On("RequirePermission", mock.Anything, models.ViewRunPermission, mock.Anything, mock.Anything).Return(test.authError)
				}

				if test.authError == nil && test.run != nil && test.runError == nil {
					var reader io.ReadCloser
					if test.object != "" {
						reader = io.NopCloser(strings.NewReader(test.object))
					}
					mockArtifactStore.On("GetPolicyCheckPolicyMessages", mock.Anything, mock.Anything).
						Return(reader, test.artifactError).Maybe()
				}
			}

			service := &service{
				dbClient:      &db.Client{Runs: mockRuns},
				artifactStore: mockArtifactStore,
			}

			callCtx := ctx
			if !test.skipCaller {
				callCtx = auth.WithCaller(ctx, mockCaller)
			}

			result, err := service.GetPolicyCheckPolicyMessages(callCtx, checkID, test.policyID)

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
			mockCaller.On("RequirePermission", mock.Anything, models.UpdateRunPermission, mock.Anything, mock.Anything).Return(test.authError)

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

			mockRuns.On("GetRunByNodeID", mock.Anything, planID).Return(run, nil)

			mockCaller.On("RequirePermission", mock.Anything, models.UpdateRunPermission, mock.Anything, mock.Anything).Return(test.authError)

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
				mockCaller.On("RequirePermission", mock.Anything, models.UpdateRunPermission, mock.Anything, mock.Anything).Return(tc.authError)

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

			mockRuns.On("GetRunByNodeID", mock.Anything, applyID).Return(&models.Run{Metadata: models.ResourceMetadata{ID: "run-1"}, Apply: &models.Apply{ID: applyID}}, nil)

			mockCaller.On("RequirePermission", mock.Anything, models.UpdateRunPermission, mock.Anything, mock.Anything).Return(test.authError)

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

			mockRuns.On("GetRunByNodeID", mock.Anything, planID).Return(&models.Run{Metadata: models.ResourceMetadata{ID: "run-1"}}, nil)

			mockCaller.On("RequirePermission", mock.Anything, models.UpdateRunPermission, mock.Anything, mock.Anything).Return(test.authError)

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

func TestApproveRunGateAuthorization(t *testing.T) {
	// Approver eligibility is decided inside the command, against the gate's snapshotted allowed
	// subjects. The service's job is the check that eligibility cannot substitute for: that the
	// caller can see the run at all. A denied caller must short-circuit before the command is
	// processed, so no approval row is ever written for a gate they could not have fetched.
	sampleGate := &models.RunGate{
		Metadata:    models.ResourceMetadata{ID: "gate-1", TRN: "trn:run_gate:acme/ws/gate-1"},
		RunID:       "run-1",
		WorkspaceID: "workspace-1",
		Status:      models.RunGatePending,
	}

	type testCase struct {
		gate            *models.RunGate
		authError       error
		name            string
		decision        models.RunGateDecision
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:     "subject is authorized",
			gate:     sampleGate,
			decision: models.RunGateDecisionApprove,
		},
		{
			name:            "subject cannot view the run",
			gate:            sampleGate,
			decision:        models.RunGateDecisionApprove,
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
		{
			name:            "a reject is authorized the same way as an approve",
			gate:            sampleGate,
			decision:        models.RunGateDecisionReject,
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
		{
			// Resolved before the permission check, so a missing gate cannot be told apart from
			// one the caller may not see — but it also must not reach the command.
			name:            "gate does not exist",
			decision:        models.RunGateDecisionApprove,
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "unsupported decision is rejected before the gate is fetched",
			decision:        models.RunGateDecision("bogus"),
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockRunGates := db.NewMockRunGates(t)
			mockRuns := db.NewMockRuns(t)
			mockWorkspaces := db.NewMockWorkspaces(t)
			mockCaller := auth.NewMockCaller(t)

			if test.expectErrorCode != errors.EInvalid {
				mockRunGates.On("GetRunGateByID", mock.Anything, "gate-1").Return(test.gate, nil)
			}

			if test.gate != nil {
				// requireRunViewAccess resolves the run's workspace before checking the permission.
				mockRuns.On("GetWorkspaceIDForRun", mock.Anything, "run-1").Return("workspace-1", nil)
				mockCaller.On("RequirePermission", mock.Anything, models.ViewRunPermission, mock.Anything, mock.Anything).
					Return(test.authError)
			}

			testLogger, _ := logger.NewForTest()
			mockProcessor := engine.NewMockCmdProcessor(t)
			if test.gate != nil && test.authError == nil {
				mockCaller.On("GetSubject").Return("user@example.com").Maybe()
				mockProcessor.On("ProcessCommand", mock.Anything, mock.Anything).
					Run(func(args mock.Arguments) {
						if c, ok := args.Get(1).(*commands.SetRunGateDecision); ok {
							c.UpdatedGate = test.gate
						}
					}).Return(nil)
				// A nil workspace skips the activity event, which is not what this test covers.
				mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "workspace-1").Return(nil, nil)
			}

			service := &service{
				logger: testLogger,
				dbClient: &db.Client{
					RunGates:   mockRunGates,
					Runs:       mockRuns,
					Workspaces: mockWorkspaces,
				},
				cmdProcessor: mockProcessor,
				cmdFactory:   commands.NewFactory(testLogger, &db.Client{}, nil, nil, nil, "", nil, nil),
			}

			gate, err := service.ApproveRunGate(auth.WithCaller(context.Background(), mockCaller), &ApproveRunGateInput{
				GateID:   "gate-1",
				Decision: test.decision,
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, "gate-1", gate.Metadata.ID)
		})
	}
}

func TestOverrideRunGateAuthorization(t *testing.T) {
	// An override needs two separate rights: seeing the run, and UpdatePolicyPermission in the group
	// defining each soft-mandatory policy that failed on the gate's check. The view check comes first
	// and short-circuits before the gate's state is resolved, so a caller who cannot see the run cannot
	// learn the gate's status or whether its check is awaiting an override.
	gateWithRules := &models.RunGate{
		Metadata:      models.ResourceMetadata{ID: "gate-1", TRN: "trn:run_gate:acme/ws/gate-1"},
		RunID:         "run-1",
		WorkspaceID:   "workspace-1",
		PolicyCheckID: "check-1",
		Status:        models.RunGatePending,
		ApprovalRules: []*models.RunGateApprovalRule{
			{Name: "policy-1", RequiredApprovals: 1},
		},
	}

	// The gate a soft failure with no approvers produces. It carries no rules, so nobody can approve
	// it — but the policy permission is still required, because the override lifts the same policy's
	// requirement either way.
	gateWithoutRules := &models.RunGate{
		Metadata:      models.ResourceMetadata{ID: "gate-1", TRN: "trn:run_gate:acme/ws/gate-1"},
		RunID:         "run-1",
		WorkspaceID:   "workspace-1",
		PolicyCheckID: "check-1",
		Status:        models.RunGatePending,
		ApprovalRules: []*models.RunGateApprovalRule{},
	}

	sampleRun := &models.Run{
		Metadata:    models.ResourceMetadata{ID: "run-1"},
		WorkspaceID: "workspace-1",
		TaskStages: []*models.RunTaskStage{
			{
				ID:        "stage-1",
				StageName: models.RunTaskStageNamePostPlan,
				PolicyChecks: []*models.PolicyCheck{
					{
						ID:        "check-1",
						StageName: models.RunTaskStageNamePostPlan,
						CheckType: models.PolicyKindOPA,
						Status:    models.PolicyCheckSoftFailed,
						Policies: []*models.PolicyCheckPolicy{
							{
								ID:               "policy-1",
								Status:           models.PolicyCheckPolicyFailed,
								EnforcementLevel: models.PolicyEnforcementSoftMandatory,
								Provenance:       models.PolicyCheckPolicyProvenance{GroupID: "group-1"},
							},
						},
					},
				},
			},
		},
	}

	type testCase struct {
		gate            *models.RunGate
		viewError       error
		overrideError   error
		name            string
		expectErrorCode errors.CodeType
		isAdmin         bool
	}

	testCases := []testCase{
		{
			name: "subject holds both rights",
			gate: gateWithRules,
		},
		{
			// Admin mode skips the override permission, but the view check still runs — it is
			// satisfied by RequirePermission's own admin-mode short-circuit, not by skipping it.
			name:    "admin mode",
			gate:    gateWithRules,
			isAdmin: true,
		},
		{
			name:            "subject cannot view the run",
			gate:            gateWithRules,
			viewError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
		{
			name:            "subject can view the run but cannot update the policy",
			gate:            gateWithRules,
			overrideError:   errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
		{
			name: "rule-less gate is overridable with the policy permission",
			gate: gateWithoutRules,
		},
		{
			// The check keys on the check's failed policies, not the gate's rules, so a gate with no
			// rules is not a free pass.
			name:            "rule-less gate still requires the policy permission",
			gate:            gateWithoutRules,
			overrideError:   errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockRunGates := db.NewMockRunGates(t)
			mockRuns := db.NewMockRuns(t)
			mockWorkspaces := db.NewMockWorkspaces(t)
			mockCaller := auth.NewMockCaller(t)

			mockRunGates.On("GetRunGateByID", mock.Anything, "gate-1").Return(test.gate, nil)

			// requireRunViewAccess resolves the run's workspace before checking the permission.
			mockRuns.On("GetWorkspaceIDForRun", mock.Anything, "run-1").Return("workspace-1", nil)
			mockCaller.On("RequirePermission", mock.Anything, models.ViewRunPermission, mock.Anything, mock.Anything).
				Return(test.viewError)

			if test.viewError == nil {
				mockRuns.On("GetRunByID", mock.Anything, "run-1").Return(sampleRun, nil)
				mockCaller.On("IsAdminModeActivated", mock.Anything).Return(test.isAdmin)
				if !test.isAdmin {
					mockCaller.On("RequirePermission", mock.Anything, models.UpdatePolicyPermission, mock.Anything).
						Return(test.overrideError)
				}
			}

			testLogger, _ := logger.NewForTest()
			mockProcessor := engine.NewMockCmdProcessor(t)
			if test.expectErrorCode == "" {
				mockCaller.On("GetSubject").Return("user@example.com").Maybe()
				mockProcessor.On("ProcessCommand", mock.Anything, mock.Anything).
					Run(func(args mock.Arguments) {
						if c, ok := args.Get(1).(*commands.SetRunGateDecision); ok {
							c.UpdatedGate = test.gate
						}
					}).Return(nil)
				// A nil workspace skips the activity event, which is not what this test covers.
				mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "workspace-1").Return(nil, nil)
			}

			service := &service{
				logger: testLogger,
				dbClient: &db.Client{
					RunGates:   mockRunGates,
					Runs:       mockRuns,
					Workspaces: mockWorkspaces,
				},
				cmdProcessor: mockProcessor,
				cmdFactory:   commands.NewFactory(testLogger, &db.Client{}, nil, nil, nil, "", nil, nil),
			}

			gate, err := service.OverrideRunGate(auth.WithCaller(context.Background(), mockCaller), "gate-1", nil)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, "gate-1", gate.Metadata.ID)
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

func TestService_DownloadPlanJSON(t *testing.T) {
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
				// On the authorized path the plan JSON is streamed from the artifact store.
				mockArtifactStore.On("GetPlanJSON", mock.Anything, sampleRun).
					Return(io.NopCloser(strings.NewReader("plan-json-data")), nil)
			}

			service := &service{
				dbClient:      &db.Client{Runs: mockRuns},
				artifactStore: mockArtifactStore,
			}

			result, err := service.DownloadPlanJSON(auth.WithCaller(context.Background(), mockCaller), "plan-1")

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			data, rErr := io.ReadAll(result)
			require.NoError(t, rErr)
			assert.Equal(t, "plan-json-data", string(data))
		})
	}
}

func TestService_ReportRunPolicyOutcomes(t *testing.T) {
	sampleRun := &models.Run{
		Metadata:    models.ResourceMetadata{ID: "run-1"},
		WorkspaceID: "workspace-1",
		TaskStages: []*models.RunTaskStage{
			{
				StageName: models.RunTaskStageNamePostPlan,
				PolicyChecks: []*models.PolicyCheck{
					{ID: "check-1", StageName: models.RunTaskStageNamePostPlan, CheckType: models.PolicyKindOPA},
				},
			},
		},
	}

	type testCase struct {
		run                 *models.Run
		authError           error
		name                string
		expectErrorCode     errors.CodeType
		expectPermissionSet bool
	}

	testCases := []testCase{
		{
			name:                "subject is authorized and the policy check exists",
			run:                 sampleRun,
			expectPermissionSet: true,
		},
		{
			name:            "run for the policy check node ID does not exist",
			run:             nil,
			expectErrorCode: errors.ENotFound,
		},
		{
			name: "run exists but has no policy check with that ID",
			run: &models.Run{
				Metadata:    models.ResourceMetadata{ID: "run-1"},
				WorkspaceID: "workspace-1",
			},
			expectErrorCode: errors.ENotFound,
		},
		{
			name:                "subject is not authorized",
			run:                 sampleRun,
			authError:           errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode:     errors.EForbidden,
			expectPermissionSet: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockCaller := auth.NewMockCaller(t)
			mockRuns := db.NewMockRuns(t)

			mockRuns.On("GetRunByNodeID", mock.Anything, "check-1").Return(test.run, nil)

			testLogger, _ := logger.NewForTest()
			mockProcessor := engine.NewMockCmdProcessor(t)
			if test.expectPermissionSet {
				mockCaller.On("RequirePermission", mock.Anything, models.UpdateRunPermission, mock.Anything, mock.Anything).Return(test.authError)
			}
			if test.expectErrorCode == "" {
				mockProcessor.On("ProcessCommand", mock.Anything, mock.Anything).Return(nil)
			}

			service := &service{
				logger:       testLogger,
				dbClient:     &db.Client{Runs: mockRuns},
				cmdProcessor: mockProcessor,
				cmdFactory:   commands.NewFactory(testLogger, &db.Client{}, nil, nil, nil, "", nil, nil),
			}

			err := service.ReportRunPolicyOutcomes(auth.WithCaller(context.Background(), mockCaller), &ReportRunPolicyOutcomesInput{
				PolicyCheckID: "check-1",
				Outcomes: []*PolicyOutcome{
					{PolicyID: "policy-1", Messages: []string{"ok"}, Passed: true},
				},
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestService_GetRunGateByTRN(t *testing.T) {
	sampleGate := &models.RunGate{
		Metadata: models.ResourceMetadata{ID: "gate-1", TRN: "trn:run_gate:acme/ws/gate-1"},
		RunID:    "run-1",
	}

	type testCase struct {
		gate            *models.RunGate
		authError       error
		name            string
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name: "subject is authorized",
			gate: sampleGate,
		},
		{
			name:            "gate does not exist",
			gate:            nil,
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "subject cannot view the run",
			gate:            sampleGate,
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockCaller := auth.NewMockCaller(t)
			mockRunGates := db.NewMockRunGates(t)
			mockRuns := db.NewMockRuns(t)

			mockRunGates.On("GetRunGateByTRN", mock.Anything, "trn:run_gate:acme/ws/gate-1").Return(test.gate, nil)
			if test.gate != nil {
				// requireRunViewAccess resolves the run's workspace before checking the permission.
				mockRuns.On("GetWorkspaceIDForRun", mock.Anything, "run-1").Return("workspace-1", nil)
				mockCaller.On("RequirePermission", mock.Anything, models.ViewRunPermission, mock.Anything, mock.Anything).
					Return(test.authError)
			}

			service := &service{
				dbClient: &db.Client{
					RunGates: mockRunGates,
					Runs:     mockRuns,
				},
			}

			result, err := service.GetRunGateByTRN(auth.WithCaller(context.Background(), mockCaller), "trn:run_gate:acme/ws/gate-1")

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, "gate-1", result.Metadata.ID)
		})
	}
}

func TestService_GetRunGateApprovalByID(t *testing.T) {
	sampleApproval := &models.RunGateApproval{
		Metadata:  models.ResourceMetadata{ID: "approval-1"},
		RunGateID: "gate-1",
	}
	sampleGate := &models.RunGate{
		Metadata: models.ResourceMetadata{ID: "gate-1"},
		RunID:    "run-1",
	}

	type testCase struct {
		approval        *models.RunGateApproval
		gate            *models.RunGate
		authError       error
		name            string
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:     "subject is authorized",
			approval: sampleApproval,
			gate:     sampleGate,
		},
		{
			name:            "approval does not exist",
			approval:        nil,
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "the approval's gate does not exist",
			approval:        sampleApproval,
			gate:            nil,
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "subject cannot view the run",
			approval:        sampleApproval,
			gate:            sampleGate,
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockCaller := auth.NewMockCaller(t)
			mockRunGates := db.NewMockRunGates(t)
			mockRunGateApprovals := db.NewMockRunGateApprovals(t)
			mockRuns := db.NewMockRuns(t)

			mockRunGateApprovals.On("GetRunGateApprovalByID", mock.Anything, "approval-1").Return(test.approval, nil)
			if test.approval != nil {
				mockRunGates.On("GetRunGateByID", mock.Anything, "gate-1").Return(test.gate, nil)
			}
			if test.gate != nil {
				mockRuns.On("GetWorkspaceIDForRun", mock.Anything, "run-1").Return("workspace-1", nil)
				mockCaller.On("RequirePermission", mock.Anything, models.ViewRunPermission, mock.Anything, mock.Anything).
					Return(test.authError)
			}

			service := &service{
				dbClient: &db.Client{
					RunGates:         mockRunGates,
					RunGateApprovals: mockRunGateApprovals,
					Runs:             mockRuns,
				},
			}

			result, err := service.GetRunGateApprovalByID(auth.WithCaller(context.Background(), mockCaller), "approval-1")

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, "approval-1", result.Metadata.ID)
		})
	}
}

func TestService_GetRunGateApprovalByTRN(t *testing.T) {
	sampleApproval := &models.RunGateApproval{
		Metadata:  models.ResourceMetadata{ID: "approval-1", TRN: "trn:run_gate_approval:acme/ws/gate-1/approval-1"},
		RunGateID: "gate-1",
	}
	sampleGate := &models.RunGate{
		Metadata: models.ResourceMetadata{ID: "gate-1"},
		RunID:    "run-1",
	}

	type testCase struct {
		approval        *models.RunGateApproval
		gate            *models.RunGate
		authError       error
		name            string
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:     "subject is authorized",
			approval: sampleApproval,
			gate:     sampleGate,
		},
		{
			name:            "approval does not exist",
			approval:        nil,
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "the approval's gate does not exist",
			approval:        sampleApproval,
			gate:            nil,
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "subject cannot view the run",
			approval:        sampleApproval,
			gate:            sampleGate,
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockCaller := auth.NewMockCaller(t)
			mockRunGates := db.NewMockRunGates(t)
			mockRunGateApprovals := db.NewMockRunGateApprovals(t)
			mockRuns := db.NewMockRuns(t)

			mockRunGateApprovals.On("GetRunGateApprovalByTRN", mock.Anything, "trn:run_gate_approval:acme/ws/gate-1/approval-1").Return(test.approval, nil)
			if test.approval != nil {
				mockRunGates.On("GetRunGateByID", mock.Anything, "gate-1").Return(test.gate, nil)
			}
			if test.gate != nil {
				mockRuns.On("GetWorkspaceIDForRun", mock.Anything, "run-1").Return("workspace-1", nil)
				mockCaller.On("RequirePermission", mock.Anything, models.ViewRunPermission, mock.Anything, mock.Anything).
					Return(test.authError)
			}

			service := &service{
				dbClient: &db.Client{
					RunGates:         mockRunGates,
					RunGateApprovals: mockRunGateApprovals,
					Runs:             mockRuns,
				},
			}

			result, err := service.GetRunGateApprovalByTRN(auth.WithCaller(context.Background(), mockCaller), "trn:run_gate_approval:acme/ws/gate-1/approval-1")

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, "approval-1", result.Metadata.ID)
		})
	}
}

func TestService_GetRunGateApprovalsByGateID(t *testing.T) {
	sampleGate := &models.RunGate{
		Metadata: models.ResourceMetadata{ID: "gate-1"},
		RunID:    "run-1",
	}
	sampleApprovals := []models.RunGateApproval{
		{Metadata: models.ResourceMetadata{ID: "approval-1"}, RunGateID: "gate-1"},
		{Metadata: models.ResourceMetadata{ID: "approval-2"}, RunGateID: "gate-1"},
	}

	type testCase struct {
		gate            *models.RunGate
		authError       error
		name            string
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name: "subject is authorized",
			gate: sampleGate,
		},
		{
			name:            "gate does not exist",
			gate:            nil,
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "subject cannot view the run",
			gate:            sampleGate,
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockCaller := auth.NewMockCaller(t)
			mockRunGates := db.NewMockRunGates(t)
			mockRunGateApprovals := db.NewMockRunGateApprovals(t)
			mockRuns := db.NewMockRuns(t)

			mockRunGates.On("GetRunGateByID", mock.Anything, "gate-1").Return(test.gate, nil)
			if test.gate != nil {
				mockRuns.On("GetWorkspaceIDForRun", mock.Anything, "run-1").Return("workspace-1", nil)
				mockCaller.On("RequirePermission", mock.Anything, models.ViewRunPermission, mock.Anything, mock.Anything).
					Return(test.authError)
			}
			if test.expectErrorCode == "" {
				mockRunGateApprovals.On("GetRunGateApprovalsByGateID", mock.Anything, "gate-1").Return(sampleApprovals, nil)
			}

			service := &service{
				dbClient: &db.Client{
					RunGates:         mockRunGates,
					RunGateApprovals: mockRunGateApprovals,
					Runs:             mockRuns,
				},
			}

			result, err := service.GetRunGateApprovalsByGateID(auth.WithCaller(context.Background(), mockCaller), "gate-1")

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Len(t, result, len(sampleApprovals))
		})
	}
}

func TestService_GetRunGatesAwaitingDecision(t *testing.T) {
	rootNamespacePath := "root-namespace-path"

	type testCase struct {
		isAdmin           bool
		useServiceAccount bool
		name              string
	}

	testCases := []testCase{
		{
			name: "a non-admin user is scoped to their root namespace memberships",
		},
		{
			name:    "an admin user is not scoped to root namespace memberships",
			isAdmin: true,
		},
		{
			name:              "a service account caller is supported",
			useServiceAccount: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockRunGates := db.NewMockRunGates(t)
			mockAuthorizer := auth.NewMockAuthorizer(t)
			mockMaintenanceMonitor := maintenance.NewMockMonitor(t)
			mockUsers := db.NewMockUsers(t)

			dbClient := &db.Client{
				RunGates: mockRunGates,
				Users:    mockUsers,
			}

			var rootNamespaceMemberships []models.MembershipNamespace
			if !test.isAdmin {
				rootNamespaceMemberships = []models.MembershipNamespace{
					{ID: "root-namespace-1", Path: rootNamespacePath},
				}
				mockAuthorizer.On("GetRootNamespaces", mock.Anything).Return(rootNamespaceMemberships, nil).Maybe()
			}

			eligibility := &db.RunGateEligibilityFilter{}
			var testCaller auth.Caller
			if test.useServiceAccount {
				saID := "service-account-1"
				eligibility.ServiceAccountID = &saID
				testCaller = auth.NewServiceAccountCaller(saID, "sa/service-account-1", mockAuthorizer, dbClient, mockMaintenanceMonitor)
			} else {
				userID := "user-1"
				eligibility.UserID = &userID
				callerUser := &models.User{
					Metadata: models.ResourceMetadata{ID: userID},
					Admin:    test.isAdmin,
					AdminModeExpiration: func() *time.Time {
						if test.isAdmin {
							t := time.Now().Add(time.Hour)
							return &t
						}
						return nil
					}(),
				}
				mockUsers.On("GetUserByID", mock.Anything, userID).Return(callerUser, nil).Maybe()
				testCaller = auth.NewUserCaller(callerUser, mockAuthorizer, dbClient, mockMaintenanceMonitor, nil)
			}

			expectedSort := db.RunGateSortableFieldCreatedAtDesc
			mockRunGates.On("GetRunGates", mock.Anything, &db.GetRunGatesInput{
				Sort: &expectedSort,
				Filter: &db.RunGateFilter{
					Statuses:                 []models.RunGateStatus{models.RunGatePending},
					Eligibility:              eligibility,
					RootNamespaceMemberships: rootNamespaceMemberships,
				},
			}).Return(&db.RunGatesResult{
				PageInfo: &pagination.PageInfo{},
			}, nil)

			service := &service{
				dbClient: dbClient,
			}

			result, err := service.GetRunGatesAwaitingDecision(auth.WithCaller(context.Background(), testCaller), &GetRunGatesAwaitingDecisionInput{})

			require.NoError(t, err)
			assert.NotNil(t, result)
		})
	}
}
