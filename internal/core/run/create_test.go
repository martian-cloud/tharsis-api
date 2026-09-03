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
	dbClient          *db.Client
	workspaces        *db.MockWorkspaces
	managedIDs        *db.MockManagedIdentities
	configVers        *db.MockConfigurationVersions
	runs              *db.MockRuns
	activityEvts      *db.MockActivityEvents
	objectStoreRefs   *db.MockObjectStoreRefs
	ruleEnforcer      *rules.MockRuleEnforcer
	limitChecker      *limits.MockLimitChecker
	groups            *db.MockGroups
	policies          *db.MockPolicies
	policySets        *db.MockPackages
	policySetVersions *db.MockPackageVersions
}

func newCreateTestEnv(t *testing.T) *createTestEnv {
	env := &createTestEnv{
		workspaces:        db.NewMockWorkspaces(t),
		managedIDs:        db.NewMockManagedIdentities(t),
		configVers:        db.NewMockConfigurationVersions(t),
		runs:              db.NewMockRuns(t),
		activityEvts:      db.NewMockActivityEvents(t),
		objectStoreRefs:   db.NewMockObjectStoreRefs(t),
		ruleEnforcer:      rules.NewMockRuleEnforcer(t),
		limitChecker:      limits.NewMockLimitChecker(t),
		groups:            db.NewMockGroups(t),
		policies:          db.NewMockPolicies(t),
		policySetVersions: db.NewMockPackageVersions(t),
	}
	env.dbClient = &db.Client{
		Workspaces:            env.workspaces,
		ManagedIdentities:     env.managedIDs,
		ConfigurationVersions: env.configVers,
		Runs:                  env.runs,
		ActivityEvents:        env.activityEvts,
		Groups:                env.groups,
		Policies:              env.policies,
		PackageVersions:       env.policySetVersions,
		ObjectStoreRefs:       env.objectStoreRefs,
	}
	return env
}

func (e *createTestEnv) create(ctx context.Context, input *CreateRunInput) (*models.Run, error) {
	return Create(ctx, e.dbClient, "", e.ruleEnforcer, e.limitChecker, input)
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
	// No policies by default, so the run-policy aggregation writes nothing.
	// GetGroups returns empty → resolveRunPolicies early-exits → GetPolicies may not be called.
	e.groups.On("GetGroups", ctx, mock.Anything).Return(&db.GroupsResult{Groups: []models.Group{}}, nil).Maybe()
	e.policies.On("GetPolicies", ctx, mock.Anything).Return(&db.PoliciesResult{Policies: []*models.Policy{}}, nil).Maybe()
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

// TestCreate_AttachesPostPlanStageWhenPolicyApplies verifies that when a post_plan policy applies to
// the run, the run model gets a post-plan policy check node (created) before persistence, carrying
// the policy snapshot on its Policies field with its version constraint left unresolved.
//
// No Packages or PackageVersions mocks are set up on purpose: run creation must not touch the package
// registry at all now that the constraint is resolved by the policy-eval job. The mocks would fail
// the test on an unexpected call.
func TestCreate_AttachesPostPlanStageWhenPolicyApplies(t *testing.T) {
	ctx := callerCtx()
	env := newCreateTestEnv(t)

	env.workspaces.On("GetWorkspaceByID", ctx, "ws-1").
		Return(&models.Workspace{Metadata: models.ResourceMetadata{ID: "ws-1"}, FullPath: "group/ws", TerraformVersion: "1.5.0"}, nil)
	env.managedIDs.On("GetManagedIdentitiesForWorkspace", ctx, "ws-1").Return(nil, nil)
	// Ancestor group resolution: workspace path "group/ws" → ancestor path "group".
	env.groups.On("GetGroups", ctx, mock.Anything).Return(&db.GroupsResult{Groups: []models.Group{
		{Metadata: models.ResourceMetadata{ID: "g-ancestor"}, FullPath: "group"},
	}}, nil)
	// Run creation resolves every stage's policies with a single query (no stage filter); only the
	// post_plan policy is returned, so only that stage's check attaches.
	env.policies.On("GetPolicies", ctx, mock.MatchedBy(func(in *db.GetPoliciesInput) bool {
		return in.Filter != nil && in.Filter.Stage == nil
	})).Return(&db.PoliciesResult{Policies: []*models.Policy{
		{
			Metadata:    models.ResourceMetadata{ID: "attach-1"},
			GroupID:     "g-ancestor",
			Kind:        models.PolicyKindOPA,
			Name:        "deny-destroy-production",
			Description: ptr.String("Destroy runs are not permitted in production workspaces."),
			OPAData: &models.OPAPolicyData{
				PackageSource:            "my-group/my-package",
				PackageVersionConstraint: ptr.String("~> 1.0"),
				Stage:                    models.RunTaskStageNamePostPlan,
				EnforcementLevel:         models.PolicyEnforcementSoftMandatory,
				// Deliberately different from the declared level: this run has an apply, so the
				// assertion below proves the declared level is the one that gets snapshotted.
				SpeculativeRunEnforcementLevel: models.PolicyEnforcementAdvisory,
			},
		},
	}}, nil)

	// Capture the run model handed to CreateRun to assert the stage node is attached.
	var capturedRun *models.Run
	env.runs.On("CreateRun", ctx, mock.Anything).Return(func(_ context.Context, run *models.Run) *models.Run {
		capturedRun = run
		run.Metadata.ID = "run-1"
		run.Metadata.CreationTimestamp = ptr.Time(time.Now().UTC())
		return run
	}, nil)
	env.runs.On("GetRuns", ctx, mock.Anything).
		Return(&db.RunsResult{PageInfo: &pagination.PageInfo{TotalCount: pagination.StaticCount(1)}}, nil)
	env.limitChecker.On("CheckLimit", ctx, limits.ResourceLimitRunsPerWorkspacePerTimePeriod, mock.Anything).Return(nil)
	env.activityEvts.On("CreateActivityEvent", mock.Anything, mock.Anything).Return(&models.ActivityEvent{}, nil)

	_, err := env.create(ctx, &CreateRunInput{Subject: "u", WorkspaceID: "ws-1"})
	require.NoError(t, err)

	require.NotNil(t, capturedRun)
	require.Len(t, capturedRun.TaskStages, 1, "post-plan task stage should be attached when a post_plan policy resolves")
	stage := capturedRun.TaskStages[0]
	assert.Equal(t, models.RunTaskStageNamePostPlan, stage.StageName)
	assert.Equal(t, models.RunTaskStageCreated, stage.Status)
	require.Len(t, stage.PolicyChecks, 1, "the stage should own one OPA policy check")
	check := stage.PolicyChecks[0]
	assert.Equal(t, models.RunTaskStageNamePostPlan, check.StageName)
	assert.Equal(t, models.PolicyKindOPA, check.CheckType)
	assert.Equal(t, models.PolicyCheckCreated, check.Status)
	// The policy snapshot is carried on the check (one entry per policy, keyed by the policy id, with
	// the owning policy captured as the single Source and the version constraint copied verbatim
	// rather than resolved). Name and Description are copied too, so the run keeps describing the
	// policy as it was evaluated.
	require.Len(t, check.Policies, 1)
	assert.Equal(t, &models.PolicyCheckPolicy{
		ID:          "attach-1",
		Name:        "deny-destroy-production",
		Description: "Destroy runs are not permitted in production workspaces.",
		OPAData: &models.PolicyCheckOPAData{
			PackageSource:            "my-group/my-package",
			PackageVersionConstraint: "~> 1.0",
		},
		EnforcementLevel: models.PolicyEnforcementSoftMandatory,
		Provenance: models.PolicyCheckPolicyProvenance{
			GroupID:   "g-ancestor",
			PolicyTRN: "",
		},
		Status: models.PolicyCheckPolicyPending,
	}, check.Policies[0])
}

// TestCreate_AttachesAllFourStagesWhenPoliciesApply verifies that a non-speculative run resolves
// policies at all four stages — pre_plan, post_plan, pre_apply, post_apply — attaching one task
// stage per stage with an applicable policy, in run order, from a single batched query.
func TestCreate_AttachesAllFourStagesWhenPoliciesApply(t *testing.T) {
	ctx := callerCtx()
	env := newCreateTestEnv(t)

	env.workspaces.On("GetWorkspaceByID", ctx, "ws-1").
		Return(&models.Workspace{Metadata: models.ResourceMetadata{ID: "ws-1"}, FullPath: "group/ws", TerraformVersion: "1.5.0"}, nil)
	env.managedIDs.On("GetManagedIdentitiesForWorkspace", ctx, "ws-1").Return(nil, nil)
	env.groups.On("GetGroups", ctx, mock.Anything).Return(&db.GroupsResult{Groups: []models.Group{
		{Metadata: models.ResourceMetadata{ID: "g-ancestor"}, FullPath: "group"},
	}}, nil)

	policyForStage := func(id string, stage models.RunTaskStageName) *models.Policy {
		return &models.Policy{
			Metadata: models.ResourceMetadata{ID: id},
			GroupID:  "g-ancestor",
			Kind:     models.PolicyKindOPA,
			Name:     "policy-" + id,
			OPAData: &models.OPAPolicyData{
				PackageSource:                  "my-group/my-package",
				Stage:                          stage,
				EnforcementLevel:               models.PolicyEnforcementAdvisory,
				SpeculativeRunEnforcementLevel: models.PolicyEnforcementAdvisory,
			},
		}
	}
	var allPolicies []*models.Policy
	for _, stage := range []models.RunTaskStageName{
		models.RunTaskStageNamePrePlan, models.RunTaskStageNamePostPlan,
		models.RunTaskStageNamePreApply, models.RunTaskStageNamePostApply,
	} {
		allPolicies = append(allPolicies, policyForStage("p-"+string(stage), stage))
	}
	env.policies.On("GetPolicies", ctx, mock.MatchedBy(func(in *db.GetPoliciesInput) bool {
		return in.Filter != nil && in.Filter.Stage == nil
	})).Return(&db.PoliciesResult{Policies: allPolicies}, nil)

	var capturedRun *models.Run
	env.runs.On("CreateRun", ctx, mock.Anything).Return(func(_ context.Context, run *models.Run) *models.Run {
		capturedRun = run
		run.Metadata.ID = "run-1"
		run.Metadata.CreationTimestamp = ptr.Time(time.Now().UTC())
		return run
	}, nil)
	env.runs.On("GetRuns", ctx, mock.Anything).
		Return(&db.RunsResult{PageInfo: &pagination.PageInfo{TotalCount: pagination.StaticCount(1)}}, nil)
	env.limitChecker.On("CheckLimit", ctx, limits.ResourceLimitRunsPerWorkspacePerTimePeriod, mock.Anything).Return(nil)
	env.activityEvts.On("CreateActivityEvent", mock.Anything, mock.Anything).Return(&models.ActivityEvent{}, nil)

	_, err := env.create(ctx, &CreateRunInput{Subject: "u", WorkspaceID: "ws-1"})
	require.NoError(t, err)

	require.NotNil(t, capturedRun)
	require.Len(t, capturedRun.TaskStages, 4, "all four stages should attach for a non-speculative run with policies at each")
	gotOrder := make([]models.RunTaskStageName, len(capturedRun.TaskStages))
	for i, s := range capturedRun.TaskStages {
		gotOrder[i] = s.StageName
	}
	assert.Equal(t, []models.RunTaskStageName{
		models.RunTaskStageNamePrePlan, models.RunTaskStageNamePostPlan,
		models.RunTaskStageNamePreApply, models.RunTaskStageNamePostApply,
	}, gotOrder, "stages must attach in run order")
}

// TestCreate_SpeculativeRunSkipsApplyPhaseStages verifies that a speculative run never attaches a
// pre_apply/post_apply task stage, even though the single batched query fetches policies at every
// stage: resolveRunPolicies is only asked to resolve pre_plan/post_plan for a speculative run, so an
// apply-phase policy present in that same batched result is dropped rather than attached — a
// speculative run has no apply node for such a stage to ever gate.
func TestCreate_SpeculativeRunSkipsApplyPhaseStages(t *testing.T) {
	ctx := callerCtx()
	env := newCreateTestEnv(t)

	env.workspaces.On("GetWorkspaceByID", ctx, "ws-1").
		Return(&models.Workspace{Metadata: models.ResourceMetadata{ID: "ws-1"}, FullPath: "group/ws", TerraformVersion: "1.5.0"}, nil)
	env.managedIDs.On("GetManagedIdentitiesForWorkspace", ctx, "ws-1").Return(nil, nil)
	env.groups.On("GetGroups", ctx, mock.Anything).Return(&db.GroupsResult{Groups: []models.Group{
		{Metadata: models.ResourceMetadata{ID: "g-ancestor"}, FullPath: "group"},
	}}, nil)
	// The single query returns a policy scoped to pre_apply, proving it is dropped in application
	// code (because a speculative run never asks to resolve that stage) rather than never fetched.
	env.policies.On("GetPolicies", ctx, mock.Anything).Return(&db.PoliciesResult{Policies: []*models.Policy{
		{
			Metadata: models.ResourceMetadata{ID: "apply-only"},
			GroupID:  "g-ancestor",
			Kind:     models.PolicyKindOPA,
			Name:     "apply-only",
			OPAData: &models.OPAPolicyData{
				PackageSource:                  "my-group/my-package",
				Stage:                          models.RunTaskStageNamePreApply,
				EnforcementLevel:               models.PolicyEnforcementAdvisory,
				SpeculativeRunEnforcementLevel: models.PolicyEnforcementAdvisory,
			},
		},
	}}, nil)

	var capturedRun *models.Run
	env.runs.On("CreateRun", ctx, mock.Anything).Return(func(_ context.Context, run *models.Run) *models.Run {
		capturedRun = run
		run.Metadata.ID = "run-1"
		run.Metadata.CreationTimestamp = ptr.Time(time.Now().UTC())
		return run
	}, nil)
	env.runs.On("GetRuns", ctx, mock.Anything).
		Return(&db.RunsResult{PageInfo: &pagination.PageInfo{TotalCount: pagination.StaticCount(1)}}, nil)
	env.limitChecker.On("CheckLimit", ctx, limits.ResourceLimitRunsPerWorkspacePerTimePeriod, mock.Anything).Return(nil)
	env.activityEvts.On("CreateActivityEvent", mock.Anything, mock.Anything).Return(&models.ActivityEvent{}, nil)

	moduleSource := "registry/group/module"
	speculative := true
	_, err := env.create(ctx, &CreateRunInput{
		Subject:      "u",
		WorkspaceID:  "ws-1",
		ModuleSource: &moduleSource,
		Speculative:  &speculative,
	})
	require.NoError(t, err)

	require.NotNil(t, capturedRun)
	assert.Empty(t, capturedRun.TaskStages, "a speculative run must never attach an apply-phase stage, even for a pre_apply policy present in the batched fetch")
	assert.Nil(t, capturedRun.Apply, "a speculative run has no apply node")
}

// TestCreate_OPAKindWithNilOPADataFailsRunCreation verifies that an OPA-kind policy whose OPAData
// failed to hydrate is treated as a data-integrity violation, not a skippable policy kind: Create
// returns an internal error and never reaches persistence, so a mandatory gate can't be silently
// dropped and the run proceed as if the policy didn't exist.
func TestCreate_OPAKindWithNilOPADataFailsRunCreation(t *testing.T) {
	ctx := callerCtx()
	env := newCreateTestEnv(t)

	env.workspaces.On("GetWorkspaceByID", ctx, "ws-1").
		Return(&models.Workspace{Metadata: models.ResourceMetadata{ID: "ws-1"}, FullPath: "group/ws", TerraformVersion: "1.5.0"}, nil)
	env.managedIDs.On("GetManagedIdentitiesForWorkspace", ctx, "ws-1").Return(nil, nil)
	env.groups.On("GetGroups", ctx, mock.Anything).Return(&db.GroupsResult{Groups: []models.Group{
		{Metadata: models.ResourceMetadata{ID: "g-ancestor"}, FullPath: "group"},
	}}, nil)
	// The stages are resolved in order (pre_plan first); return the malformed policy from any stage
	// query so resolution hits the OPA-kind-with-nil-OPAData policy. post_plan may not be reached
	// once the first stage errors, so leave its expectation optional.
	env.policies.On("GetPolicies", ctx, mock.Anything).Return(&db.PoliciesResult{Policies: []*models.Policy{
		{
			Metadata: models.ResourceMetadata{ID: "broken-1", TRN: "trn:policy:group/broken"},
			GroupID:  "g-ancestor",
			Kind:     models.PolicyKindOPA,
			Name:     "hydration-regression",
			OPAData:  nil, // OPA kind but no data: a deserialization bug / partial migration.
		},
	}}, nil).Maybe()

	run, err := env.create(ctx, &CreateRunInput{Subject: "u", WorkspaceID: "ws-1"})

	require.Error(t, err)
	assert.Nil(t, run)
	assert.Equal(t, errors.EInternal, errors.ErrorCode(err))
	// The run must never be persisted when a mandatory gate can't be evaluated.
	env.runs.AssertNotCalled(t, "CreateRun", mock.Anything, mock.Anything)
}

// TestCreate_UnsupportedKindFailsRunCreation verifies that a policy whose kind is neither OPA nor
// module_attestation is treated as a data-integrity violation rather than silently skipped: Policy.
// Validate already rejects any other kind at write time, so reaching this code with one means the
// gate was dropped some other way (direct DB write, a downgrade after a kind's removal), and Create
// must fail rather than let the run proceed as if the policy never existed.
func TestCreate_UnsupportedKindFailsRunCreation(t *testing.T) {
	ctx := callerCtx()
	env := newCreateTestEnv(t)

	env.workspaces.On("GetWorkspaceByID", ctx, "ws-1").
		Return(&models.Workspace{Metadata: models.ResourceMetadata{ID: "ws-1"}, FullPath: "group/ws", TerraformVersion: "1.5.0"}, nil)
	env.managedIDs.On("GetManagedIdentitiesForWorkspace", ctx, "ws-1").Return(nil, nil)
	env.groups.On("GetGroups", ctx, mock.Anything).Return(&db.GroupsResult{Groups: []models.Group{
		{Metadata: models.ResourceMetadata{ID: "g-ancestor"}, FullPath: "group"},
	}}, nil)
	// The stages are resolved in order (pre_plan first); return the unsupported-kind policy from any
	// stage query so resolution hits it. post_plan may not be reached once the first stage errors, so
	// leave its expectation optional.
	env.policies.On("GetPolicies", ctx, mock.Anything).Return(&db.PoliciesResult{Policies: []*models.Policy{
		{
			Metadata: models.ResourceMetadata{ID: "future-1", TRN: "trn:policy:group/future"},
			GroupID:  "g-ancestor",
			Kind:     models.PolicyKind("sentinel"),
			Name:     "future-kind",
		},
	}}, nil).Maybe()

	run, err := env.create(ctx, &CreateRunInput{Subject: "u", WorkspaceID: "ws-1"})

	require.Error(t, err)
	assert.Nil(t, run)
	assert.Equal(t, errors.EInternal, errors.ErrorCode(err))
	env.runs.AssertNotCalled(t, "CreateRun", mock.Anything, mock.Anything)
}

// TestCreate_UnsupportedKindAtAnUnrelatedStageDoesNotFailRunCreation verifies that a policy of an
// unrecognized kind attached to a stage the run is not resolving is skipped rather than hard-failing
// run creation. This is the regression case for resolveRunPolicies checking kind support before
// filtering by stage: with that ordering, a future-kind policy scoped to pre_apply/post_apply would
// have hard-failed even a speculative run that never touches either -- a landmine reachable the
// moment a third kind exists, even though it never fires while only OPA and module_attestation do.
func TestCreate_UnsupportedKindAtAnUnrelatedStageDoesNotFailRunCreation(t *testing.T) {
	ctx := callerCtx()
	env := newCreateTestEnv(t)

	env.workspaces.On("GetWorkspaceByID", ctx, "ws-1").
		Return(&models.Workspace{Metadata: models.ResourceMetadata{ID: "ws-1"}, FullPath: "group/ws", TerraformVersion: "1.5.0"}, nil)
	env.managedIDs.On("GetManagedIdentitiesForWorkspace", ctx, "ws-1").Return(nil, nil)
	env.groups.On("GetGroups", ctx, mock.Anything).Return(&db.GroupsResult{Groups: []models.Group{
		{Metadata: models.ResourceMetadata{ID: "g-ancestor"}, FullPath: "group"},
	}}, nil)
	// Scoped to pre_apply, so it is present in the batched fetch (see resolveRunPolicies' comment on
	// GetWorkspaceAssignedPolicies) but a speculative run never asks to resolve that stage -- the same
	// setup TestCreate_SpeculativeRunSkipsApplyPhaseStages uses for a *known* kind. OPAData is only
	// present here to give the fixture a resolvable stage (Policy.Stage() reads it regardless of
	// Kind); a real future kind would carry its own stage field the same way.
	env.policies.On("GetPolicies", ctx, mock.Anything).Return(&db.PoliciesResult{Policies: []*models.Policy{
		{
			Metadata: models.ResourceMetadata{ID: "future-2", TRN: "trn:policy:group/future-2"},
			GroupID:  "g-ancestor",
			Kind:     models.PolicyKind("sentinel"),
			Name:     "future-kind-unrelated-stage",
			OPAData:  &models.OPAPolicyData{Stage: models.RunTaskStageNamePreApply},
		},
	}}, nil)

	var capturedRun *models.Run
	env.runs.On("CreateRun", ctx, mock.Anything).Return(func(_ context.Context, run *models.Run) *models.Run {
		capturedRun = run
		run.Metadata.ID = "run-1"
		run.Metadata.CreationTimestamp = ptr.Time(time.Now().UTC())
		return run
	}, nil)
	env.runs.On("GetRuns", ctx, mock.Anything).
		Return(&db.RunsResult{PageInfo: &pagination.PageInfo{TotalCount: pagination.StaticCount(1)}}, nil)
	env.limitChecker.On("CheckLimit", ctx, limits.ResourceLimitRunsPerWorkspacePerTimePeriod, mock.Anything).Return(nil)
	env.activityEvts.On("CreateActivityEvent", mock.Anything, mock.Anything).Return(&models.ActivityEvent{}, nil)

	moduleSource := "registry/group/module"
	speculative := true
	_, err := env.create(ctx, &CreateRunInput{
		Subject:      "u",
		WorkspaceID:  "ws-1",
		ModuleSource: &moduleSource,
		Speculative:  &speculative,
	})
	require.NoError(t, err)

	require.NotNil(t, capturedRun)
	assert.Empty(t, capturedRun.TaskStages, "the unsupported-kind policy's pre_apply stage is not one a speculative run resolves, so it must be dropped rather than hard-failing")
}

// TestCreate_ModuleAttestationKindWithNilDataFailsRunCreation mirrors the OPA data-integrity guard
// for the module_attestation kind: a policy of that kind with no ModuleAttestationData must fail run
// creation rather than silently dropping the gate.
func TestCreate_ModuleAttestationKindWithNilDataFailsRunCreation(t *testing.T) {
	ctx := callerCtx()
	env := newCreateTestEnv(t)

	env.workspaces.On("GetWorkspaceByID", ctx, "ws-1").
		Return(&models.Workspace{Metadata: models.ResourceMetadata{ID: "ws-1"}, FullPath: "group/ws", TerraformVersion: "1.5.0"}, nil)
	env.managedIDs.On("GetManagedIdentitiesForWorkspace", ctx, "ws-1").Return(nil, nil)
	env.groups.On("GetGroups", ctx, mock.Anything).Return(&db.GroupsResult{Groups: []models.Group{
		{Metadata: models.ResourceMetadata{ID: "g-ancestor"}, FullPath: "group"},
	}}, nil)
	env.policies.On("GetPolicies", ctx, mock.Anything).Return(&db.PoliciesResult{Policies: []*models.Policy{
		{
			Metadata:              models.ResourceMetadata{ID: "broken-1", TRN: "trn:policy:group/broken"},
			GroupID:               "g-ancestor",
			Kind:                  models.PolicyKindModuleAttestation,
			Name:                  "hydration-regression",
			ModuleAttestationData: nil,
		},
	}}, nil).Maybe()

	run, err := env.create(ctx, &CreateRunInput{Subject: "u", WorkspaceID: "ws-1"})

	require.Error(t, err)
	assert.Nil(t, run)
	assert.Equal(t, errors.EInternal, errors.ErrorCode(err))
	env.runs.AssertNotCalled(t, "CreateRun", mock.Anything, mock.Anything)
}

// TestCreate_StageWithBothKindsProducesTwoChecksWithDistinctPaths verifies that a stage with both an
// OPA and a module attestation policy produces two sibling PolicyCheck nodes under one task stage,
// each with its own path and holding only its own kind's policy.
func TestCreate_StageWithBothKindsProducesTwoChecksWithDistinctPaths(t *testing.T) {
	ctx := callerCtx()
	env := newCreateTestEnv(t)

	env.workspaces.On("GetWorkspaceByID", ctx, "ws-1").
		Return(&models.Workspace{Metadata: models.ResourceMetadata{ID: "ws-1"}, FullPath: "group/ws", TerraformVersion: "1.5.0"}, nil)
	env.managedIDs.On("GetManagedIdentitiesForWorkspace", ctx, "ws-1").Return(nil, nil)
	env.groups.On("GetGroups", ctx, mock.Anything).Return(&db.GroupsResult{Groups: []models.Group{
		{Metadata: models.ResourceMetadata{ID: "g-ancestor"}, FullPath: "group"},
	}}, nil)

	predicateType := "https://slsa.dev/provenance/v1"
	prePlanPolicies := []*models.Policy{
		{
			Metadata: models.ResourceMetadata{ID: "opa-1", TRN: "trn:policy:group/opa"},
			GroupID:  "g-ancestor",
			Kind:     models.PolicyKindOPA,
			Name:     "opa-policy",
			OPAData: &models.OPAPolicyData{
				PackageSource:    "my-package",
				EnforcementLevel: models.PolicyEnforcementHardMandatory,
				Stage:            models.RunTaskStageNamePrePlan,
			},
		},
		{
			Metadata: models.ResourceMetadata{ID: "attest-1", TRN: "trn:policy:group/attest"},
			GroupID:  "g-ancestor",
			Kind:     models.PolicyKindModuleAttestation,
			Name:     "attestation-policy",
			ModuleAttestationData: &models.ModuleAttestationPolicyData{
				PublicKey:        "-----BEGIN PUBLIC KEY-----\nkey\n-----END PUBLIC KEY-----",
				PredicateType:    &predicateType,
				EnforcementLevel: models.PolicyEnforcementHardMandatory,
				Stage:            models.RunTaskStageNamePrePlan,
			},
		},
	}
	env.policies.On("GetPolicies", ctx, mock.Anything).Return(&db.PoliciesResult{Policies: prePlanPolicies}, nil)

	var capturedRun *models.Run
	env.runs.On("CreateRun", ctx, mock.Anything).Return(func(_ context.Context, run *models.Run) *models.Run {
		capturedRun = run
		run.Metadata.ID = "run-1"
		run.Metadata.CreationTimestamp = ptr.Time(time.Now().UTC())
		return run
	}, nil)
	env.runs.On("GetRuns", ctx, mock.Anything).
		Return(&db.RunsResult{PageInfo: &pagination.PageInfo{TotalCount: pagination.StaticCount(1)}}, nil)
	env.limitChecker.On("CheckLimit", ctx, limits.ResourceLimitRunsPerWorkspacePerTimePeriod, mock.Anything).Return(nil)
	env.activityEvts.On("CreateActivityEvent", mock.Anything, mock.Anything).Return(&models.ActivityEvent{}, nil)

	_, err := env.create(ctx, &CreateRunInput{Subject: "u", WorkspaceID: "ws-1"})
	require.NoError(t, err)

	require.NotNil(t, capturedRun)
	var prePlanStage *models.RunTaskStage
	for _, stage := range capturedRun.TaskStages {
		if stage.StageName == models.RunTaskStageNamePrePlan {
			prePlanStage = stage
		}
	}
	require.NotNil(t, prePlanStage, "pre_plan stage must be attached")
	require.Len(t, prePlanStage.PolicyChecks, 2)

	opaCheck := prePlanStage.PolicyChecks[0]
	attestationCheck := prePlanStage.PolicyChecks[1]
	assert.Equal(t, models.PolicyKindOPA, opaCheck.CheckType)
	assert.Equal(t, models.PolicyKindModuleAttestation, attestationCheck.CheckType)
	assert.NotEqual(t, opaCheck.GetPath(), attestationCheck.GetPath())
	require.Len(t, opaCheck.Policies, 1)
	require.Len(t, attestationCheck.Policies, 1)
	assert.Equal(t, "opa-1", opaCheck.Policies[0].ID)
	assert.Equal(t, "attest-1", attestationCheck.Policies[0].ID)
}

// TestCreate_NoStageNodeWhenNoPolicies verifies a run with no policies gets no
// post-plan stage node (it behaves exactly as before the feature).
func TestCreate_NoStageNodeWhenNoPolicies(t *testing.T) {
	ctx := callerCtx()
	env := newCreateTestEnv(t)

	env.workspaces.On("GetWorkspaceByID", ctx, "ws-1").
		Return(&models.Workspace{FullPath: "group/ws", TerraformVersion: "1.5.0"}, nil)
	env.managedIDs.On("GetManagedIdentitiesForWorkspace", ctx, "ws-1").Return(nil, nil)

	var capturedRun *models.Run
	env.runs.On("CreateRun", ctx, mock.Anything).Return(func(_ context.Context, run *models.Run) *models.Run {
		capturedRun = run
		run.Metadata.ID = "run-1"
		run.Metadata.CreationTimestamp = ptr.Time(time.Now().UTC())
		return run
	}, nil)
	env.runs.On("GetRuns", ctx, mock.Anything).
		Return(&db.RunsResult{PageInfo: &pagination.PageInfo{TotalCount: pagination.StaticCount(1)}}, nil)
	env.limitChecker.On("CheckLimit", ctx, limits.ResourceLimitRunsPerWorkspacePerTimePeriod, mock.Anything).Return(nil)
	env.activityEvts.On("CreateActivityEvent", mock.Anything, mock.Anything).Return(&models.ActivityEvent{}, nil)
	env.groups.On("GetGroups", ctx, mock.Anything).Return(&db.GroupsResult{Groups: []models.Group{}}, nil).Maybe()
	env.policies.On("GetPolicies", ctx, mock.Anything).Return(&db.PoliciesResult{Policies: []*models.Policy{}}, nil).Maybe()

	_, err := env.create(ctx, &CreateRunInput{Subject: "u", WorkspaceID: "ws-1"})
	require.NoError(t, err)

	require.NotNil(t, capturedRun)
	assert.Empty(t, capturedRun.TaskStages, "no task stage should be attached when there are no policies")
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
	env.groups.On("GetGroups", ctx, mock.Anything).Return(&db.GroupsResult{Groups: []models.Group{}}, nil).Maybe()
	env.policies.On("GetPolicies", ctx, mock.Anything).Return(&db.PoliciesResult{Policies: []*models.Policy{}}, nil).Maybe()

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

// TestCreate_SpeculativeUsesSpeculativeEnforcementLevel verifies that a run with no apply is enforced
// at each policy's speculative level rather than its declared one, at every stage. The two policies
// here declare soft mandatory — the level a speculative run can never be enforced at, since nothing
// would act on an approval — and name different speculative levels, so the snapshot shows the level
// came from the policy rather than from a rule about which stage is being resolved.
func TestCreate_SpeculativeUsesSpeculativeEnforcementLevel(t *testing.T) {
	ctx := callerCtx()
	env := newCreateTestEnv(t)

	env.workspaces.On("GetWorkspaceByID", ctx, "ws-1").
		Return(&models.Workspace{Metadata: models.ResourceMetadata{ID: "ws-1"}, FullPath: "group/ws", TerraformVersion: "1.5.0"}, nil)
	env.managedIDs.On("GetManagedIdentitiesForWorkspace", ctx, "ws-1").Return(nil, nil)
	env.groups.On("GetGroups", ctx, mock.Anything).Return(&db.GroupsResult{Groups: []models.Group{
		{Metadata: models.ResourceMetadata{ID: "g-ancestor"}, FullPath: "group"},
	}}, nil)

	// A single query returns every stage's policies at once; return one per stage to exercise both.
	policyAt := func(id string, stage models.RunTaskStageName, speculative models.PolicyEnforcementLevel) *models.Policy {
		return &models.Policy{
			Metadata: models.ResourceMetadata{ID: id},
			GroupID:  "g-ancestor",
			Kind:     models.PolicyKindOPA,
			Name:     id,
			OPAData: &models.OPAPolicyData{
				PackageSource:                  "my-group/my-package",
				Stage:                          stage,
				EnforcementLevel:               models.PolicyEnforcementSoftMandatory,
				SpeculativeRunEnforcementLevel: speculative,
			},
		}
	}
	env.policies.On("GetPolicies", ctx, mock.Anything).Return(&db.PoliciesResult{
		Policies: []*models.Policy{
			policyAt("pre-1", models.RunTaskStageNamePrePlan, models.PolicyEnforcementHardMandatory),
			policyAt("post-1", models.RunTaskStageNamePostPlan, models.PolicyEnforcementAdvisory),
		},
	}, nil)

	var capturedRun *models.Run
	env.runs.On("CreateRun", ctx, mock.Anything).Return(func(_ context.Context, run *models.Run) *models.Run {
		capturedRun = run
		run.Metadata.ID = "run-1"
		run.Metadata.CreationTimestamp = ptr.Time(time.Now().UTC())
		return run
	}, nil)
	env.runs.On("GetRuns", ctx, mock.Anything).
		Return(&db.RunsResult{PageInfo: &pagination.PageInfo{TotalCount: pagination.StaticCount(1)}}, nil)
	env.limitChecker.On("CheckLimit", ctx, limits.ResourceLimitRunsPerWorkspacePerTimePeriod, mock.Anything).Return(nil)
	env.activityEvts.On("CreateActivityEvent", mock.Anything, mock.Anything).Return(&models.ActivityEvent{}, nil)

	_, err := env.create(ctx, &CreateRunInput{
		Subject:      "user@example.com",
		WorkspaceID:  "ws-1",
		ModuleSource: ptr.String("registry.example.com/ns/name/aws"),
		Speculative:  ptr.Bool(true),
	})
	require.NoError(t, err)

	require.NotNil(t, capturedRun)
	require.Len(t, capturedRun.TaskStages, 2, "both stages resolved a policy")
	enforcementByStage := map[models.RunTaskStageName]models.PolicyEnforcementLevel{}
	for _, stage := range capturedRun.TaskStages {
		require.Len(t, stage.PolicyChecks, 1)
		require.Len(t, stage.PolicyChecks[0].Policies, 1)
		enforcementByStage[stage.StageName] = stage.PolicyChecks[0].Policies[0].EnforcementLevel
	}
	assert.Equal(t, models.PolicyEnforcementHardMandatory, enforcementByStage[models.RunTaskStageNamePrePlan],
		"pre-plan must enforce at the speculative level the policy names: it decides whether the config runs at all")
	assert.Equal(t, models.PolicyEnforcementAdvisory, enforcementByStage[models.RunTaskStageNamePostPlan],
		"post-plan must enforce at the speculative level the policy names")
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
	// Policy resolution runs before the run is created (to gate the stage node), so it
	// happens even on the limit-exceeded path.
	env.groups.On("GetGroups", ctx, mock.Anything).Return(&db.GroupsResult{Groups: []models.Group{}}, nil).Maybe()
	env.policies.On("GetPolicies", ctx, mock.Anything).Return(&db.PoliciesResult{Policies: []*models.Policy{}}, nil).Maybe()
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

func TestUploadRunVariables_MasksSensitiveValues(t *testing.T) {
	ctx := context.Background()
	artifactStore := workspace.NewMockArtifactStore(t)

	// Capture the uploaded payload to confirm sensitive values are stripped.
	var uploaded []runvariables.Variable
	artifactStore.On("UploadRunVariables", ctx, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			body := args.Get(2).(io.Reader)
			data, err := io.ReadAll(body)
			require.NoError(t, err)
			require.NoError(t, json.Unmarshal(data, &uploaded))
		}).
		Return(db.RetainObjectRefFunc(func(_ context.Context, _ string) error { return nil }), "workspaces/ws-1/run_variables/key.json", nil)

	secretVal := "super-secret"
	plainVal := "plain"
	retainFn, key, err := UploadRunVariables(ctx, artifactStore, "ws-1", []runvariables.Variable{
		{Key: "secret", Value: &secretVal, Sensitive: true, Category: models.TerraformVariableCategory},
		{Key: "plain", Value: &plainVal, Sensitive: false, Category: models.TerraformVariableCategory},
	})
	require.NoError(t, err)
	assert.Equal(t, "workspaces/ws-1/run_variables/key.json", key)
	assert.NotNil(t, retainFn)

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
