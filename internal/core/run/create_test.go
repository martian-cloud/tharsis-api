package run

import (
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/rules"
	runvariables "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/variables"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/workspace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/limits"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// createTestEnv bundles the collaborators that corerun.Create needs so each test
// can wire only the behavior it cares about.
type createTestEnv struct {
	dbClient      *db.Client
	workspaces    *db.MockWorkspaces
	managedIDs    *db.MockManagedIdentities
	configVers    *db.MockConfigurationVersions
	runs          *db.MockRuns
	activityEvts  *db.MockActivityEvents
	ruleEnforcer  *rules.MockRuleEnforcer
	limitChecker  *limits.MockLimitChecker
	artifactStore *workspace.MockArtifactStore
}

func newCreateTestEnv(t *testing.T) *createTestEnv {
	env := &createTestEnv{
		workspaces:    db.NewMockWorkspaces(t),
		managedIDs:    db.NewMockManagedIdentities(t),
		configVers:    db.NewMockConfigurationVersions(t),
		runs:          db.NewMockRuns(t),
		activityEvts:  db.NewMockActivityEvents(t),
		ruleEnforcer:  rules.NewMockRuleEnforcer(t),
		limitChecker:  limits.NewMockLimitChecker(t),
		artifactStore: workspace.NewMockArtifactStore(t),
	}
	env.dbClient = &db.Client{
		Workspaces:            env.workspaces,
		ManagedIdentities:     env.managedIDs,
		ConfigurationVersions: env.configVers,
		Runs:                  env.runs,
		ActivityEvents:        env.activityEvts,
	}
	return env
}

func (e *createTestEnv) create(ctx context.Context, input *CreateRunInput) (*models.Run, error) {
	return Create(ctx, e.dbClient, "", e.ruleEnforcer, e.limitChecker, e.artifactStore, input)
}

// callerCtx returns a context carrying a service-account caller so the activity
// event writer attributes the run-creation event.
func callerCtx() context.Context {
	return auth.WithCaller(context.Background(),
		auth.NewServiceAccountCaller("sa-1", "group/sa", nil, nil, nil))
}

// stubCreatePersistence wires the persistence/limit/activity/upload calls that the
// happy-path portion of Create makes after validation succeeds.
func (e *createTestEnv) stubCreatePersistence(ctx context.Context) {
	e.runs.On("CreateRun", ctx, mock.Anything).Return(func(_ context.Context, run *models.Run) *models.Run {
		run.Metadata.ID = "run-1"
		run.Metadata.CreationTimestamp = ptr.Time(time.Now().UTC())
		return run
	}, nil)
	e.runs.On("GetRuns", ctx, mock.Anything).
		Return(&db.RunsResult{PageInfo: &pagination.PageInfo{TotalCount: pagination.StaticCount(1)}}, nil)
	e.limitChecker.On("CheckLimit", ctx, limits.ResourceLimitRunsPerWorkspacePerTimePeriod, mock.Anything).Return(nil)
	e.activityEvts.On("CreateActivityEvent", mock.Anything, mock.MatchedBy(func(in *models.ActivityEvent) bool {
		return in.Action == models.ActionCreate && in.TargetType == models.TargetRun
	})).Return(&models.ActivityEvent{}, nil)
	e.artifactStore.On("UploadRunVariables", ctx, mock.Anything, mock.Anything).Return(nil)
}

func TestCreate_HappyPath_NonSpeculative(t *testing.T) {
	ctx := callerCtx()
	env := newCreateTestEnv(t)

	env.workspaces.On("GetWorkspaceByID", ctx, "ws-1").
		Return(&models.Workspace{FullPath: "group/ws", TerraformVersion: "1.5.0"}, nil)
	env.managedIDs.On("GetManagedIdentitiesForWorkspace", ctx, "ws-1").Return(nil, nil)
	env.stubCreatePersistence(ctx)

	run, err := env.create(ctx, &CreateRunInput{
		Subject:     "user@example.com",
		WorkspaceID: "ws-1",
	})
	require.NoError(t, err)
	require.NotNil(t, run)
	// No terraform version supplied -> defaults to the workspace's version.
	assert.Equal(t, "1.5.0", run.TerraformVersion)
	assert.Equal(t, models.RunPending, run.Status)
	assert.Equal(t, models.PlanCreated, run.Plan.Status)
	// Non-speculative runs get an apply node.
	require.NotNil(t, run.Apply)
	assert.Equal(t, models.ApplyCreated, run.Apply.Status)
}

func TestCreate_SkipActivityEvent(t *testing.T) {
	ctx := callerCtx()
	env := newCreateTestEnv(t)

	env.workspaces.On("GetWorkspaceByID", ctx, "ws-1").
		Return(&models.Workspace{FullPath: "group/ws", TerraformVersion: "1.5.0"}, nil)
	env.managedIDs.On("GetManagedIdentitiesForWorkspace", ctx, "ws-1").Return(nil, nil)

	// Wire the persistence path without a CreateActivityEvent expectation;
	// NewMockActivityEvents(t) fails the test if the suppressed event is created anyway.
	env.runs.On("CreateRun", ctx, mock.Anything).Return(func(_ context.Context, run *models.Run) *models.Run {
		run.Metadata.ID = "run-1"
		run.Metadata.CreationTimestamp = ptr.Time(time.Now().UTC())
		return run
	}, nil)
	env.runs.On("GetRuns", ctx, mock.Anything).
		Return(&db.RunsResult{PageInfo: &pagination.PageInfo{TotalCount: pagination.StaticCount(1)}}, nil)
	env.limitChecker.On("CheckLimit", ctx, limits.ResourceLimitRunsPerWorkspacePerTimePeriod, mock.Anything).Return(nil)
	env.artifactStore.On("UploadRunVariables", ctx, mock.Anything, mock.Anything).Return(nil)

	run, err := env.create(ctx, &CreateRunInput{
		Subject:           "system",
		WorkspaceID:       "ws-1",
		SkipActivityEvent: true,
	})
	require.NoError(t, err)
	require.NotNil(t, run)
	env.activityEvts.AssertNotCalled(t, "CreateActivityEvent", mock.Anything, mock.Anything)
}

func TestCreate_Speculative_HasNoApplyNode(t *testing.T) {
	ctx := callerCtx()
	env := newCreateTestEnv(t)

	env.workspaces.On("GetWorkspaceByID", ctx, "ws-1").
		Return(&models.Workspace{FullPath: "group/ws", TerraformVersion: "1.5.0"}, nil)
	env.managedIDs.On("GetManagedIdentitiesForWorkspace", ctx, "ws-1").Return(nil, nil)
	env.stubCreatePersistence(ctx)

	run, err := env.create(ctx, &CreateRunInput{
		Subject:      "user@example.com",
		WorkspaceID:  "ws-1",
		ModuleSource: ptr.String("registry.example.com/ns/name/aws"),
		Speculative:  ptr.Bool(true),
	})
	require.NoError(t, err)
	assert.Nil(t, run.Apply)
}

func TestCreate_DestroyOnPreventDestroyWorkspace_Forbidden(t *testing.T) {
	ctx := callerCtx()
	env := newCreateTestEnv(t)

	env.workspaces.On("GetWorkspaceByID", ctx, "ws-1").
		Return(&models.Workspace{FullPath: "group/ws", TerraformVersion: "1.5.0", PreventDestroyPlan: true}, nil)

	run, err := env.create(ctx, &CreateRunInput{
		Subject:     "user@example.com",
		WorkspaceID: "ws-1",
		IsDestroy:   true,
	})
	require.Error(t, err)
	assert.Equal(t, errors.EForbidden, errors.ErrorCode(err))
	assert.Nil(t, run)
}

func TestCreate_WorkspaceNotFound(t *testing.T) {
	ctx := callerCtx()
	env := newCreateTestEnv(t)
	env.workspaces.On("GetWorkspaceByID", ctx, "ws-1").Return(nil, nil)

	run, err := env.create(ctx, &CreateRunInput{Subject: "u", WorkspaceID: "ws-1"})
	require.Error(t, err)
	assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
	assert.Nil(t, run)
}

func TestCreate_ManagedIdentityRuleDenied(t *testing.T) {
	ctx := callerCtx()
	env := newCreateTestEnv(t)

	env.workspaces.On("GetWorkspaceByID", ctx, "ws-1").
		Return(&models.Workspace{FullPath: "group/ws", TerraformVersion: "1.5.0"}, nil)
	env.managedIDs.On("GetManagedIdentitiesForWorkspace", ctx, "ws-1").
		Return([]models.ManagedIdentity{{Metadata: models.ResourceMetadata{ID: "mi-1"}}}, nil)
	env.ruleEnforcer.On("EnforceRules", ctx, mock.Anything, mock.Anything).
		Return(errors.New("rule violation", errors.WithErrorCode(errors.EForbidden)))

	run, err := env.create(ctx, &CreateRunInput{Subject: "u", WorkspaceID: "ws-1"})
	require.Error(t, err)
	assert.Equal(t, errors.EForbidden, errors.ErrorCode(err))
	assert.Nil(t, run)
}

func TestCreate_RunLimitExceeded(t *testing.T) {
	ctx := callerCtx()
	env := newCreateTestEnv(t)

	env.workspaces.On("GetWorkspaceByID", ctx, "ws-1").
		Return(&models.Workspace{FullPath: "group/ws", TerraformVersion: "1.5.0"}, nil)
	env.managedIDs.On("GetManagedIdentitiesForWorkspace", ctx, "ws-1").Return(nil, nil)
	env.runs.On("CreateRun", ctx, mock.Anything).Return(func(_ context.Context, run *models.Run) *models.Run {
		run.Metadata.ID = "run-1"
		run.Metadata.CreationTimestamp = ptr.Time(time.Now().UTC())
		return run
	}, nil)
	env.runs.On("GetRuns", ctx, mock.Anything).
		Return(&db.RunsResult{PageInfo: &pagination.PageInfo{TotalCount: pagination.StaticCount(101)}}, nil)
	env.limitChecker.On("CheckLimit", ctx, limits.ResourceLimitRunsPerWorkspacePerTimePeriod, mock.Anything).
		Return(errors.New("limit exceeded", errors.WithErrorCode(errors.EInvalid)))

	run, err := env.create(ctx, &CreateRunInput{Subject: "u", WorkspaceID: "ws-1"})
	require.Error(t, err)
	assert.Equal(t, errors.EInvalid, errors.ErrorCode(err))
	assert.Nil(t, run)
}

func TestCreate_SensitiveVariablesAreMaskedBeforeUpload(t *testing.T) {
	ctx := callerCtx()
	env := newCreateTestEnv(t)

	env.workspaces.On("GetWorkspaceByID", ctx, "ws-1").
		Return(&models.Workspace{FullPath: "group/ws", TerraformVersion: "1.5.0"}, nil)
	env.managedIDs.On("GetManagedIdentitiesForWorkspace", ctx, "ws-1").Return(nil, nil)
	env.runs.On("CreateRun", ctx, mock.Anything).Return(func(_ context.Context, run *models.Run) *models.Run {
		run.Metadata.ID = "run-1"
		run.Metadata.CreationTimestamp = ptr.Time(time.Now().UTC())
		return run
	}, nil)
	env.runs.On("GetRuns", ctx, mock.Anything).
		Return(&db.RunsResult{PageInfo: &pagination.PageInfo{TotalCount: pagination.StaticCount(1)}}, nil)
	env.limitChecker.On("CheckLimit", ctx, limits.ResourceLimitRunsPerWorkspacePerTimePeriod, mock.Anything).Return(nil)
	env.activityEvts.On("CreateActivityEvent", mock.Anything, mock.Anything).Return(&models.ActivityEvent{}, nil)

	// Capture the uploaded variable payload to confirm sensitive values are stripped.
	var uploaded []runvariables.Variable
	env.artifactStore.On("UploadRunVariables", ctx, mock.Anything, mock.Anything).
		Return(nil).Run(func(args mock.Arguments) {
		body := args.Get(2).(io.Reader)
		data, err := io.ReadAll(body)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(data, &uploaded))
	})

	secretVal := "super-secret"
	plainVal := "plain"
	_, err := env.create(ctx, &CreateRunInput{
		Subject:     "u",
		WorkspaceID: "ws-1",
		Variables: []runvariables.Variable{
			{Key: "secret", Value: &secretVal, Sensitive: true, Category: models.TerraformVariableCategory},
			{Key: "plain", Value: &plainVal, Sensitive: false, Category: models.TerraformVariableCategory},
		},
	})
	require.NoError(t, err)

	require.Len(t, uploaded, 2)
	for _, v := range uploaded {
		if v.Sensitive {
			assert.Nil(t, v.Value, "sensitive variable value must be stripped before upload")
		} else {
			require.NotNil(t, v.Value)
			assert.Equal(t, "plain", *v.Value)
		}
	}
}

// NOTE: The "unsupported Terraform version" branch is intentionally not unit-tested
// here: it calls terraform.GetCLIVersions, which performs live HashiCorp releases-API
// I/O and is covered by integration tests. All unit cases above leave the input
// TerraformVersion empty so Create uses the workspace's version without that call.

func TestGetFederatedRegistry(t *testing.T) {
	ws := &models.Workspace{FullPath: "group/sub/ws"}
	hostname := "registry.example.com"

	t.Run("multiple registries returns the first selected by the getter", func(t *testing.T) {
		ctx := context.Background()
		fedRegs := db.NewMockFederatedRegistries(t)
		groups := db.NewMockGroups(t)
		dbClient := &db.Client{FederatedRegistries: fedRegs, Groups: groups}

		// Two registries sharing the same hostname under an ancestor and its descendant
		// group. GetFederatedRegistries dedups by hostname, preferring the descendant.
		ancestor := &models.FederatedRegistry{
			Metadata: models.ResourceMetadata{ID: "fr-ancestor"},
			Hostname: hostname,
			GroupID:  "g-ancestor",
		}
		descendant := &models.FederatedRegistry{
			Metadata: models.ResourceMetadata{ID: "fr-descendant"},
			Hostname: hostname,
			GroupID:  "g-descendant",
		}
		fedRegs.On("GetFederatedRegistries", ctx, mock.Anything).
			Return(&db.FederatedRegistriesResult{
				FederatedRegistries: []*models.FederatedRegistry{ancestor, descendant},
			}, nil)
		groups.On("GetGroups", ctx, mock.Anything).
			Return(&db.GroupsResult{
				Groups: []models.Group{
					{Metadata: models.ResourceMetadata{ID: "g-ancestor"}, FullPath: "group"},
					{Metadata: models.ResourceMetadata{ID: "g-descendant"}, FullPath: "group/sub"},
				},
			}, nil)

		got, err := GetFederatedRegistry(dbClient, ws)(ctx, hostname)
		require.NoError(t, err)
		require.NotNil(t, got)
		// The dedup collapses to a single registry, which the getter returns as [0].
		assert.Equal(t, descendant, got)
	})

	t.Run("no registries returns nil", func(t *testing.T) {
		ctx := context.Background()
		fedRegs := db.NewMockFederatedRegistries(t)
		dbClient := &db.Client{FederatedRegistries: fedRegs}

		fedRegs.On("GetFederatedRegistries", ctx, mock.Anything).
			Return(&db.FederatedRegistriesResult{
				FederatedRegistries: []*models.FederatedRegistry{},
			}, nil)

		got, err := GetFederatedRegistry(dbClient, ws)(ctx, hostname)
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("db error is wrapped", func(t *testing.T) {
		ctx := context.Background()
		fedRegs := db.NewMockFederatedRegistries(t)
		dbClient := &db.Client{FederatedRegistries: fedRegs}

		fedRegs.On("GetFederatedRegistries", ctx, mock.Anything).
			Return(nil, errors.New("boom"))

		got, err := GetFederatedRegistry(dbClient, ws)(ctx, hostname)
		require.Error(t, err)
		assert.Nil(t, got)
		assert.Contains(t, err.Error(), "failed to get federated registries")
	})
}
