package commands

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/in-toto/in-toto-golang/in_toto"
	ssldsse "github.com/secure-systems-lab/go-securesystemslib/dsse"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/sigstore/sigstore/pkg/signature/dsse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/registry"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/store"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/workspace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

const evalTestModuleDigestHex = "7ae471ed18395339572f5265b835860e28a2f85016455214cb214bafe4422cba"

// signEvalTestAttestation builds a minimal signed DSSE-wrapped in-toto attestation over
// moduleDigestHex, matching the base64-of-JSON-envelope form GetAttestations returns in production.
// Key material is generated fresh in-test rather than committed to the repo.
func signEvalTestAttestation(t *testing.T, priv *ecdsa.PrivateKey, moduleDigestHex string) string {
	t.Helper()

	statement := in_toto.Statement{
		StatementHeader: in_toto.StatementHeader{
			Type:          in_toto.StatementInTotoV01,
			PredicateType: "https://slsa.dev/provenance/v1",
			Subject: []in_toto.Subject{
				{Name: "module", Digest: map[string]string{"sha256": moduleDigestHex}},
			},
		},
		Predicate: map[string]string{},
	}
	payload, err := json.Marshal(statement)
	require.NoError(t, err)

	signer, err := signature.LoadSigner(priv, crypto.SHA256)
	require.NoError(t, err)

	envelopeSigner, err := ssldsse.NewEnvelopeSigner(&dsse.SignerAdapter{SignatureSigner: signer})
	require.NoError(t, err)

	env, err := envelopeSigner.SignPayload(context.Background(), "application/vnd.in-toto+json", payload)
	require.NoError(t, err)

	envBytes, err := json.Marshal(env)
	require.NoError(t, err)
	return base64.StdEncoding.EncodeToString(envBytes)
}

// evalRunPolicyCheckTestEnv bundles the collaborators EvaluateRunPolicyCheck needs, each
// individually wireable, mirroring createTestEnv's style in the run package.
type evalRunPolicyCheckTestEnv struct {
	run            *models.Run
	runStore       *store.RunStore
	mockRuns       *db.MockRuns
	mockWorkspaces *db.MockWorkspaces
	mockResolver   *registry.MockModuleResolver
	artifactStore  *workspace.MockArtifactStore
}

func newEvalRunPolicyCheckTestEnv(t *testing.T, check *models.PolicyCheck) *evalRunPolicyCheckTestEnv {
	moduleDigest, err := hex.DecodeString(evalTestModuleDigestHex)
	require.NoError(t, err)

	run := &models.Run{
		Metadata:      models.ResourceMetadata{ID: "run-1"},
		WorkspaceID:   "ws-1",
		Status:        models.RunPostPlanRunning,
		Plan:          models.Plan{ID: "plan-1", Status: models.PlanFinished, HasChanges: true},
		Apply:         &models.Apply{ID: "apply-1", Status: models.ApplyCreated},
		ModuleSource:  ptr.String("registry.example.com/group/module/aws"),
		ModuleDigest:  moduleDigest,
		ModuleVersion: ptr.String("1.0.0"),
		TaskStages: []*models.RunTaskStage{{
			ID:           "ts-post",
			StageName:    models.RunTaskStageNamePostPlan,
			Status:       models.RunTaskStageRunning,
			PolicyChecks: []*models.PolicyCheck{check},
		}},
	}

	runStore := store.NewRunStore(&db.Client{})
	runStore.AddRun(run)

	mockRuns := db.NewMockRuns(t)
	mockWorkspaces := db.NewMockWorkspaces(t)
	mockResolver := registry.NewMockModuleResolver(t)
	artifactStore := workspace.NewMockArtifactStore(t)

	return &evalRunPolicyCheckTestEnv{
		run:            run,
		runStore:       runStore,
		mockRuns:       mockRuns,
		mockWorkspaces: mockWorkspaces,
		mockResolver:   mockResolver,
		artifactStore:  artifactStore,
	}
}

func (e *evalRunPolicyCheckTestEnv) command(policyCheckID string) *EvaluateRunPolicyCheck {
	dbClient := &db.Client{Runs: e.mockRuns, Workspaces: e.mockWorkspaces}
	return &EvaluateRunPolicyCheck{
		ReportRunPolicyOutcomes: ReportRunPolicyOutcomes{
			dbClient:      dbClient,
			artifactStore: e.artifactStore,
			PolicyCheckID: policyCheckID,
		},
		dbClient:       dbClient,
		moduleResolver: e.mockResolver,
		RunID:          "run-1",
	}
}

func attestationPolicy(id string, level models.PolicyEnforcementLevel, publicKeyPEM string) *models.PolicyCheckPolicy {
	return &models.PolicyCheckPolicy{
		ID:                    id,
		ModuleAttestationData: &models.PolicyCheckModuleAttestationData{PublicKey: publicKeyPEM},
		EnforcementLevel:      level,
		Status:                models.PolicyCheckPolicyPending,
	}
}

func TestEvaluateRunPolicyCheck_HappyPath(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	pemBytes, err := cryptoutils.MarshalPublicKeyToPEM(&priv.PublicKey)
	require.NoError(t, err)
	attestation := signEvalTestAttestation(t, priv, evalTestModuleDigestHex)

	check := &models.PolicyCheck{
		ID:        "check-1",
		StageName: models.RunTaskStageNamePostPlan,
		CheckType: models.PolicyKindModuleAttestation,
		Status:    models.PolicyCheckQueued,
		Policies:  []*models.PolicyCheckPolicy{attestationPolicy("a", models.PolicyEnforcementHardMandatory, string(pemBytes))},
	}
	env := newEvalRunPolicyCheckTestEnv(t, check)

	env.mockRuns.On("GetRunByID", mock.Anything, "run-1").Return(env.run, nil)
	env.mockRuns.On("GetRunByNodeID", mock.Anything, "check-1").Return(env.run, nil)
	env.mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "ws-1").Return(&models.Workspace{Metadata: models.ResourceMetadata{ID: "ws-1"}}, nil)

	mockSource := registry.NewMockModuleRegistrySource(t)
	mockSource.On("IsTharsisModule").Return(true)
	mockSource.On("GetAttestations", mock.Anything, "1.0.0", evalTestModuleDigestHex).Return([]string{attestation}, nil)
	env.mockResolver.On("ParseModuleRegistrySource", mock.Anything, "registry.example.com/group/module/aws", mock.Anything, mock.Anything).
		Return(mockSource, nil)

	cmd := env.command("check-1")

	require.NoError(t, cmd.Prepare(context.Background()))
	require.NoError(t, cmd.Execute(context.Background(), &types.ExecuteInput{RunStore: env.runStore}))

	assert.Equal(t, models.PolicyCheckPassed, check.Status)
	assert.Nil(t, check.LatestJobID, "an in-API check never gets a job")
}

func TestEvaluateRunPolicyCheck_HardMandatoryFailureErrorsCheck(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	otherPriv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	pemBytes, err := cryptoutils.MarshalPublicKeyToPEM(&priv.PublicKey)
	require.NoError(t, err)
	// Signed with a different key than the policy requires, so verification fails.
	attestation := signEvalTestAttestation(t, otherPriv, evalTestModuleDigestHex)

	check := &models.PolicyCheck{
		ID:        "check-1",
		StageName: models.RunTaskStageNamePostPlan,
		CheckType: models.PolicyKindModuleAttestation,
		Status:    models.PolicyCheckQueued,
		Policies:  []*models.PolicyCheckPolicy{attestationPolicy("a", models.PolicyEnforcementHardMandatory, string(pemBytes))},
	}
	env := newEvalRunPolicyCheckTestEnv(t, check)

	env.mockRuns.On("GetRunByID", mock.Anything, "run-1").Return(env.run, nil)
	env.mockRuns.On("GetRunByNodeID", mock.Anything, "check-1").Return(env.run, nil)
	env.mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "ws-1").Return(&models.Workspace{Metadata: models.ResourceMetadata{ID: "ws-1"}}, nil)

	mockSource := registry.NewMockModuleRegistrySource(t)
	mockSource.On("IsTharsisModule").Return(true)
	mockSource.On("GetAttestations", mock.Anything, "1.0.0", evalTestModuleDigestHex).Return([]string{attestation}, nil)
	env.mockResolver.On("ParseModuleRegistrySource", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(mockSource, nil)

	env.artifactStore.On("UploadPolicyCheckPolicyMessages", mock.Anything, env.run, mock.Anything).
		Return(db.RetainObjectRefFunc(func(_ context.Context, _ string) error { return nil }), "policy_messages/obj.json", nil)

	cmd := env.command("check-1")

	require.NoError(t, cmd.Prepare(context.Background()))
	require.NoError(t, cmd.Execute(context.Background(), &types.ExecuteInput{RunStore: env.runStore}))

	assert.Equal(t, models.PolicyCheckErrored, check.Status)
	assert.Equal(t, models.PolicyCheckPolicyFailed, check.Policies[0].Status)
}

func TestEvaluateRunPolicyCheck_AdvisoryFailurePassesCheckButFlagsRun(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	otherPriv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	pemBytes, err := cryptoutils.MarshalPublicKeyToPEM(&priv.PublicKey)
	require.NoError(t, err)
	attestation := signEvalTestAttestation(t, otherPriv, evalTestModuleDigestHex)

	check := &models.PolicyCheck{
		ID:        "check-1",
		StageName: models.RunTaskStageNamePostPlan,
		CheckType: models.PolicyKindModuleAttestation,
		Status:    models.PolicyCheckQueued,
		Policies:  []*models.PolicyCheckPolicy{attestationPolicy("a", models.PolicyEnforcementAdvisory, string(pemBytes))},
	}
	env := newEvalRunPolicyCheckTestEnv(t, check)

	env.mockRuns.On("GetRunByID", mock.Anything, "run-1").Return(env.run, nil)
	env.mockRuns.On("GetRunByNodeID", mock.Anything, "check-1").Return(env.run, nil)
	env.mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "ws-1").Return(&models.Workspace{Metadata: models.ResourceMetadata{ID: "ws-1"}}, nil)

	mockSource := registry.NewMockModuleRegistrySource(t)
	mockSource.On("IsTharsisModule").Return(true)
	mockSource.On("GetAttestations", mock.Anything, "1.0.0", evalTestModuleDigestHex).Return([]string{attestation}, nil)
	env.mockResolver.On("ParseModuleRegistrySource", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(mockSource, nil)

	env.artifactStore.On("UploadPolicyCheckPolicyMessages", mock.Anything, env.run, mock.Anything).
		Return(db.RetainObjectRefFunc(func(_ context.Context, _ string) error { return nil }), "policy_messages/obj.json", nil)

	cmd := env.command("check-1")

	require.NoError(t, cmd.Prepare(context.Background()))
	require.NoError(t, cmd.Execute(context.Background(), &types.ExecuteInput{RunStore: env.runStore}))

	assert.Equal(t, models.PolicyCheckPassed, check.Status)
	assert.True(t, env.run.HasAdvisoryFailures)
}

// TestEvaluateRunPolicyCheck_RedeliveryNoOp verifies a work item redelivered after the check already
// reached a terminal status (a prior attempt committed but the item's ack was lost, or the check was
// independently retried/discarded) is a no-op: no module resolution, no verdict change.
func TestEvaluateRunPolicyCheck_RedeliveryNoOp(t *testing.T) {
	check := &models.PolicyCheck{
		ID:        "check-1",
		StageName: models.RunTaskStageNamePostPlan,
		CheckType: models.PolicyKindModuleAttestation,
		Status:    models.PolicyCheckPassed,
		Policies:  []*models.PolicyCheckPolicy{attestationPolicy("a", models.PolicyEnforcementHardMandatory, "unused")},
	}
	env := newEvalRunPolicyCheckTestEnv(t, check)

	env.mockRuns.On("GetRunByID", mock.Anything, "run-1").Return(env.run, nil)
	// No other expectations: a redelivery must not touch the workspace, the resolver, or storage.

	cmd := env.command("check-1")

	require.NoError(t, cmd.Prepare(context.Background()))
	assert.True(t, cmd.skip)
	require.NoError(t, cmd.Execute(context.Background(), &types.ExecuteInput{RunStore: env.runStore}))

	assert.Equal(t, models.PolicyCheckPassed, check.Status, "status must be untouched by the no-op")
}

// TestEvaluateRunPolicyCheck_RegistryErrorReturnsWithoutVerdict verifies that a registry/network
// failure while resolving the module source is returned as an error (so the work item is
// redelivered) rather than being recorded as a failed outcome.
func TestEvaluateRunPolicyCheck_RegistryErrorReturnsWithoutVerdict(t *testing.T) {
	check := &models.PolicyCheck{
		ID:        "check-1",
		StageName: models.RunTaskStageNamePostPlan,
		CheckType: models.PolicyKindModuleAttestation,
		Status:    models.PolicyCheckQueued,
		Policies:  []*models.PolicyCheckPolicy{attestationPolicy("a", models.PolicyEnforcementHardMandatory, "unused")},
	}
	env := newEvalRunPolicyCheckTestEnv(t, check)

	env.mockRuns.On("GetRunByID", mock.Anything, "run-1").Return(env.run, nil)
	env.mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "ws-1").Return(&models.Workspace{Metadata: models.ResourceMetadata{ID: "ws-1"}}, nil)
	env.mockResolver.On("ParseModuleRegistrySource", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("registry unreachable"))

	cmd := env.command("check-1")

	err := cmd.Prepare(context.Background())
	require.Error(t, err)

	// No verdict was ever set: the check is exactly as it started.
	assert.Equal(t, models.PolicyCheckQueued, check.Status)
}
