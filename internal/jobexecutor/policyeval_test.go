package jobexecutor

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/jobclient"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/client"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// noopJobLogger is a no-op joblogger.Logger for tests.
type noopJobLogger struct{}

func (noopJobLogger) Close()                              {}
func (noopJobLogger) Errorf(_ string, _ ...interface{})   {}
func (noopJobLogger) Flush()                              {}
func (noopJobLogger) Infof(_ string, _ ...interface{})    {}
func (noopJobLogger) Printf(_ string, _ ...interface{})   {}
func (noopJobLogger) Start()                              {}
func (noopJobLogger) Warningf(_ string, _ ...interface{}) {}
func (noopJobLogger) Write(data []byte) (int, error)      { return len(data), nil }

// makeBundleBytes builds a tar.gz package from a map of file name -> contents.
func makeBundleBytes(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, contents := range files {
		require.NoError(t, tw.WriteHeader(&tar.Header{
			Name:     name,
			Mode:     0o600,
			Size:     int64(len(contents)),
			Typeflag: tar.TypeReg,
		}))
		_, err := tw.Write([]byte(contents))
		require.NoError(t, err)
	}
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())

	return buf.Bytes()
}

func makeBundle(t *testing.T, files map[string]string) io.ReadCloser {
	return io.NopCloser(bytes.NewReader(makeBundleBytes(t, files)))
}

// newTestRun builds a run whose post-plan OPA policy check pins the given policies (mirroring how
// the API populates the Run message from the run model).
func newTestRun(policies []*pb.PolicyCheckPolicy) *pb.Run {
	return &pb.Run{
		Metadata:  &pb.ResourceMetadata{Id: "run-1"},
		PlanId:    "plan-1",
		IsDestroy: true,
		TaskStages: []*pb.RunTaskStage{
			{
				Id:        "post-plan-stage-gid",
				StageName: pb.RunTaskStageName_RUN_TASK_STAGE_NAME_POST_PLAN,
				Status:    pb.RunTaskStageStatus_RUN_TASK_STAGE_STATUS_RUNNING,
				PolicyChecks: []*pb.PolicyCheck{
					{
						Id:        "post-plan-policy-check-gid",
						CheckType: pb.PolicyCheckType_POLICY_CHECK_TYPE_OPA,
						StageName: pb.RunTaskStageName_RUN_TASK_STAGE_NAME_POST_PLAN,
						Status:    pb.PolicyCheckStatus_POLICY_CHECK_STATUS_RUNNING,
						Policies:  policies,
					},
				},
			},
		},
	}
}

// onlyMessage returns the single message an outcome reported. Outcomes are a list of messages now, so
// asserting through this says "one violation, and it reads like so" rather than matching a substring
// of an unknown number of them.
func onlyMessage(t *testing.T, outcome jobclient.RunPolicyOutcomeInput) string {
	t.Helper()
	require.Len(t, outcome.Messages, 1)
	return outcome.Messages[0]
}

func newTestHandler(client jobclient.Client, policies []*pb.PolicyCheckPolicy) *PolicyEvalHandler {
	return &PolicyEvalHandler{
		cancellableCtx: context.Background(),
		run:            newTestRun(policies),
		workspace:      &pb.Workspace{Metadata: &pb.ResourceMetadata{Id: "ws-1"}, FullPath: "group/ws"},
		job: &pb.Job{
			Metadata: &pb.ResourceMetadata{Id: "job-1"},
			JobData: &pb.Job_OpaData{
				OpaData: &pb.OPAJobData{PolicyCheckId: "post-plan-policy-check-gid"},
			},
		},
		jobLogger: noopJobLogger{},
		client:    client,
	}
}

func TestPolicyEvalHandler_Execute_FailingPolicy(t *testing.T) {
	ctx := context.Background()
	client := jobclient.NewMockClient(t)

	client.On("GetRunVariables", ctx, "run-1", false).Return([]*pb.RunVariable{}, nil)
	client.On("DownloadPlanJSON", ctx, "plan-1", mock.Anything).Run(func(args mock.Arguments) {
		w := args.Get(2).(io.Writer)
		_, _ = w.Write([]byte(`{"format_version":"1.0"}`))
	}).Return(nil)

	// A rule that denies when the run is a destroy (the test run is a destroy).
	rego := `package tharsis.test

deny contains msg if {
    input.run.isDestroy
    msg := "destroy runs are not allowed"
}`
	// The run carries a constraint, not a version: the API resolves it, and the resolved version is
	// deliberately different from anything in the constraint so the handler cannot be reading it back.
	client.On("GetPackageVersion", ctx, "security", ">= 1.0.0").Return(
		&pb.PackageVersion{Metadata: &pb.ResourceMetadata{Id: "pv-1"}, Version: "1.2.0"}, nil)
	client.On("DownloadPackage", ctx, "pv-1").Return(
		makeBundle(t, map[string]string{"policy.rego": rego}), nil)

	var captured []jobclient.RunPolicyOutcomeInput
	client.On("ReportRunPolicyOutcomes", ctx, "post-plan-policy-check-gid", mock.Anything).Run(func(args mock.Arguments) {
		captured = args.Get(2).([]jobclient.RunPolicyOutcomeInput)
	}).Return(nil)

	handler := newTestHandler(client, []*pb.PolicyCheckPolicy{
		{
			Id:                       "pol-1",
			PackageSource:            "security",
			PackageVersionConstraint: ">= 1.0.0",
			EnforcementLevel:         pb.PolicyEnforcementLevel_POLICY_ENFORCEMENT_LEVEL_HARD_MANDATORY,
			Provenance:               &pb.PolicyCheckPolicyProvenance{PolicyTrn: "trn:policy:group/security"},
		},
	})
	require.NoError(t, handler.Execute(ctx))

	require.Len(t, captured, 1)
	assert.Equal(t, "pol-1", captured[0].PolicyID)
	assert.False(t, captured[0].Passed)
	assert.Contains(t, onlyMessage(t, captured[0]), "destroy runs are not allowed")
}

// TestPolicyEvalHandler_Execute_MultipleViolations verifies each violation is reported as its own
// message rather than one blob, and that the list is sorted: violations are collected by ranging a map
// keyed by rego package, so without sorting an unchanged policy would reorder its findings between
// attempts. The two rules below live in separate packages precisely to exercise that.
func TestPolicyEvalHandler_Execute_MultipleViolations(t *testing.T) {
	ctx := context.Background()
	client := jobclient.NewMockClient(t)

	client.On("GetRunVariables", ctx, "run-1", false).Return([]*pb.RunVariable{}, nil)
	client.On("DownloadPlanJSON", ctx, "plan-1", mock.Anything).Run(func(args mock.Arguments) {
		w := args.Get(2).(io.Writer)
		_, _ = w.Write([]byte(`{"format_version":"1.0"}`))
	}).Return(nil)

	first := `package tharsis.first

deny contains msg if {
    input.run.isDestroy
    msg := "b destroy runs are not allowed"
}`
	second := `package tharsis.second

deny contains msg if {
    input.run.isDestroy
    msg := "a destroys need an approval"
}`
	client.On("GetPackageVersion", ctx, "security", ">= 1.0.0").Return(
		&pb.PackageVersion{Metadata: &pb.ResourceMetadata{Id: "pv-1"}, Version: "1.2.0"}, nil)
	client.On("DownloadPackage", ctx, "pv-1").Return(
		makeBundle(t, map[string]string{"first.rego": first, "second.rego": second}), nil)

	var captured []jobclient.RunPolicyOutcomeInput
	client.On("ReportRunPolicyOutcomes", ctx, "post-plan-policy-check-gid", mock.Anything).Run(func(args mock.Arguments) {
		captured = args.Get(2).([]jobclient.RunPolicyOutcomeInput)
	}).Return(nil)

	handler := newTestHandler(client, []*pb.PolicyCheckPolicy{
		{
			Id:                       "pol-1",
			PackageSource:            "security",
			PackageVersionConstraint: ">= 1.0.0",
			EnforcementLevel:         pb.PolicyEnforcementLevel_POLICY_ENFORCEMENT_LEVEL_HARD_MANDATORY,
			Provenance:               &pb.PolicyCheckPolicyProvenance{PolicyTrn: "trn:policy:group/security"},
		},
	})
	require.NoError(t, handler.Execute(ctx))

	require.Len(t, captured, 1)
	assert.False(t, captured[0].Passed)
	assert.Equal(t, []string{
		"a destroys need an approval",
		"b destroy runs are not allowed",
	}, captured[0].Messages)
}

// TestPolicyEvalHandler_Execute_RejectedReportFailsJob verifies a report the API refuses fails the job.
// The API rejects rather than truncates a policy reporting more messages than it allows, and nothing is
// stored when it does — so the job must not report success and leave a check with no verdict. The
// executor derives the job status from the returned error, which is why this returns rather than
// logging and moving on.
func TestPolicyEvalHandler_Execute_RejectedReportFailsJob(t *testing.T) {
	ctx := context.Background()
	client := jobclient.NewMockClient(t)

	client.On("GetRunVariables", ctx, "run-1", false).Return([]*pb.RunVariable{}, nil)
	client.On("DownloadPlanJSON", ctx, "plan-1", mock.Anything).Run(func(args mock.Arguments) {
		w := args.Get(2).(io.Writer)
		_, _ = w.Write([]byte(`{"format_version":"1.0"}`))
	}).Return(nil)

	rego := `package tharsis.test

deny contains msg if {
    input.run.isDestroy
    msg := "destroy runs are not allowed"
}`
	client.On("GetPackageVersion", ctx, "security", ">= 1.0.0").Return(
		&pb.PackageVersion{Metadata: &pb.ResourceMetadata{Id: "pv-1"}, Version: "1.2.0"}, nil)
	client.On("DownloadPackage", ctx, "pv-1").Return(
		makeBundle(t, map[string]string{"policy.rego": rego}), nil)

	client.On("ReportRunPolicyOutcomes", ctx, "post-plan-policy-check-gid", mock.Anything).Return(
		status.Error(codes.InvalidArgument, "policy pol-1 reported 2097152 bytes of messages, above the 1048576 byte limit"))

	handler := newTestHandler(client, []*pb.PolicyCheckPolicy{
		{
			Id:                       "pol-1",
			PackageSource:            "security",
			PackageVersionConstraint: ">= 1.0.0",
			EnforcementLevel:         pb.PolicyEnforcementLevel_POLICY_ENFORCEMENT_LEVEL_HARD_MANDATORY,
			Provenance:               &pb.PolicyCheckPolicyProvenance{PolicyTrn: "trn:policy:group/security"},
		},
	})

	err := handler.Execute(ctx)
	require.Error(t, err)
	// The reason reaches the job: the runner surfaces what the API said rather than a bare failure.
	assert.Contains(t, err.Error(), "above the 1048576 byte limit")
}

func TestPolicyEvalHandler_Execute_PassingPolicy(t *testing.T) {
	ctx := context.Background()
	client := jobclient.NewMockClient(t)

	client.On("GetRunVariables", ctx, "run-1", false).Return([]*pb.RunVariable{}, nil)
	client.On("DownloadPlanJSON", ctx, "plan-1", mock.Anything).Run(func(args mock.Arguments) {
		w := args.Get(2).(io.Writer)
		_, _ = w.Write([]byte(`{}`))
	}).Return(nil)

	// Denies only when a (non-existent) variable is present, so it never fires here.
	rego := `package tharsis.test

deny contains msg if {
    input.variables[_].key == "forbidden"
    msg := "forbidden variable present"
}`
	// The run carries a constraint, not a version: the API resolves it, and the resolved version is
	// deliberately different from anything in the constraint so the handler cannot be reading it back.
	client.On("GetPackageVersion", ctx, "security", ">= 1.0.0").Return(
		&pb.PackageVersion{Metadata: &pb.ResourceMetadata{Id: "pv-1"}, Version: "1.2.0"}, nil)
	client.On("DownloadPackage", ctx, "pv-1").Return(
		makeBundle(t, map[string]string{"policy.rego": rego}), nil)

	var captured []jobclient.RunPolicyOutcomeInput
	client.On("ReportRunPolicyOutcomes", ctx, "post-plan-policy-check-gid", mock.Anything).Run(func(args mock.Arguments) {
		captured = args.Get(2).([]jobclient.RunPolicyOutcomeInput)
	}).Return(nil)

	handler := newTestHandler(client, []*pb.PolicyCheckPolicy{
		{Id: "pol-1", PackageSource: "security", PackageVersionConstraint: ">= 1.0.0", EnforcementLevel: pb.PolicyEnforcementLevel_POLICY_ENFORCEMENT_LEVEL_ADVISORY, Provenance: &pb.PolicyCheckPolicyProvenance{PolicyTrn: "trn:policy:group/security"}},
	})
	require.NoError(t, handler.Execute(ctx))

	require.Len(t, captured, 1)
	assert.Equal(t, "pol-1", captured[0].PolicyID)
	assert.True(t, captured[0].Passed)
	assert.Empty(t, captured[0].Messages)
}

// TestPolicyEvalHandler_Execute_UnresolvableConstraintFails verifies a constraint no uploaded version
// satisfies (including a deleted package) fails the policy rather than being skipped: the constraint
// is resolved when the check runs, so this is the first point at which it can be known.
func TestPolicyEvalHandler_Execute_UnresolvableConstraintFails(t *testing.T) {
	ctx := context.Background()
	client := jobclient.NewMockClient(t)

	client.On("GetRunVariables", ctx, "run-1", false).Return([]*pb.RunVariable{}, nil)
	client.On("DownloadPlanJSON", ctx, "plan-1", mock.Anything).Run(func(args mock.Arguments) {
		w := args.Get(2).(io.Writer)
		_, _ = w.Write([]byte(`{}`))
	}).Return(nil)
	client.On("GetPackageVersion", ctx, "gone", ">= 2.0.0").Return(
		nil, status.Error(codes.NotFound, "no matching version"))

	var captured []jobclient.RunPolicyOutcomeInput
	client.On("ReportRunPolicyOutcomes", ctx, "post-plan-policy-check-gid", mock.Anything).Run(func(args mock.Arguments) {
		captured = args.Get(2).([]jobclient.RunPolicyOutcomeInput)
	}).Return(nil)

	handler := newTestHandler(client, []*pb.PolicyCheckPolicy{
		{Id: "pol-1", PackageSource: "gone", PackageVersionConstraint: ">= 2.0.0", EnforcementLevel: pb.PolicyEnforcementLevel_POLICY_ENFORCEMENT_LEVEL_HARD_MANDATORY, Provenance: &pb.PolicyCheckPolicyProvenance{PolicyTrn: "trn:policy:group/gone"}},
	})
	require.NoError(t, handler.Execute(ctx))

	// Reported as a failed outcome (not a job abort) so it flows through the server's
	// enforcement-level verdict.
	require.Len(t, captured, 1)
	assert.Equal(t, "pol-1", captured[0].PolicyID)
	assert.False(t, captured[0].Passed)
	msg := onlyMessage(t, captured[0])
	assert.Contains(t, msg, "not found")
	// The message names the constraint that could not be satisfied.
	assert.Contains(t, msg, ">= 2.0.0")
}

// TestPolicyEvalHandler_Execute_DeletedPolicyFails verifies a package version that resolves but whose
// content has since been deleted fails the policy.
func TestPolicyEvalHandler_Execute_DeletedPolicyFails(t *testing.T) {
	ctx := context.Background()
	client := jobclient.NewMockClient(t)

	client.On("GetRunVariables", ctx, "run-1", false).Return([]*pb.RunVariable{}, nil)
	client.On("DownloadPlanJSON", ctx, "plan-1", mock.Anything).Run(func(args mock.Arguments) {
		w := args.Get(2).(io.Writer)
		_, _ = w.Write([]byte(`{}`))
	}).Return(nil)
	client.On("GetPackageVersion", ctx, "gone", "").Return(
		&pb.PackageVersion{Metadata: &pb.ResourceMetadata{Id: "pv-gone"}, Version: "2.0.0"}, nil)
	client.On("DownloadPackage", ctx, "pv-gone").Return(nil, status.Error(codes.NotFound, "deleted"))

	var captured []jobclient.RunPolicyOutcomeInput
	client.On("ReportRunPolicyOutcomes", ctx, "post-plan-policy-check-gid", mock.Anything).Run(func(args mock.Arguments) {
		captured = args.Get(2).([]jobclient.RunPolicyOutcomeInput)
	}).Return(nil)

	// An empty constraint means latest, which the mock resolves to 2.0.0.
	handler := newTestHandler(client, []*pb.PolicyCheckPolicy{
		{Id: "pol-1", PackageSource: "gone", EnforcementLevel: pb.PolicyEnforcementLevel_POLICY_ENFORCEMENT_LEVEL_HARD_MANDATORY, Provenance: &pb.PolicyCheckPolicyProvenance{PolicyTrn: "trn:policy:group/gone"}},
	})
	require.NoError(t, handler.Execute(ctx))

	require.Len(t, captured, 1)
	assert.Equal(t, "pol-1", captured[0].PolicyID)
	assert.False(t, captured[0].Passed)
	assert.Contains(t, onlyMessage(t, captured[0]), "missing files")
}

// TestPolicyEvalHandler_Execute_DigestMatch verifies a pinned digest matching the downloaded
// package's sha256 lets evaluation proceed.
func TestPolicyEvalHandler_Execute_DigestMatch(t *testing.T) {
	ctx := context.Background()
	client := jobclient.NewMockClient(t)

	rego := `package tharsis.test` // no deny rules -> passes
	pkg := makeBundleBytes(t, map[string]string{"policy.rego": rego})
	sum := sha256.Sum256(pkg)

	client.On("GetRunVariables", ctx, "run-1", false).Return([]*pb.RunVariable{}, nil)
	client.On("DownloadPlanJSON", ctx, "plan-1", mock.Anything).Run(func(args mock.Arguments) {
		_, _ = args.Get(2).(io.Writer).Write([]byte(`{}`))
	}).Return(nil)
	// The run carries a constraint, not a version: the API resolves it, and the resolved version is
	// deliberately different from anything in the constraint so the handler cannot be reading it back.
	client.On("GetPackageVersion", ctx, "security", ">= 1.0.0").Return(
		&pb.PackageVersion{Metadata: &pb.ResourceMetadata{Id: "pv-1"}, Version: "1.2.0"}, nil)
	client.On("DownloadPackage", ctx, "pv-1").Return(io.NopCloser(bytes.NewReader(pkg)), nil)

	var captured []jobclient.RunPolicyOutcomeInput
	client.On("ReportRunPolicyOutcomes", ctx, "post-plan-policy-check-gid", mock.Anything).Run(func(args mock.Arguments) {
		captured = args.Get(2).([]jobclient.RunPolicyOutcomeInput)
	}).Return(nil)

	digestStr := hex.EncodeToString(sum[:])
	handler := newTestHandler(client, []*pb.PolicyCheckPolicy{
		{Id: "pol-1", PackageSource: "security", PackageVersionConstraint: ">= 1.0.0", EnforcementLevel: pb.PolicyEnforcementLevel_POLICY_ENFORCEMENT_LEVEL_HARD_MANDATORY, PackageDigest: &digestStr, Provenance: &pb.PolicyCheckPolicyProvenance{PolicyTrn: "trn:policy:group/security"}},
	})
	require.NoError(t, handler.Execute(ctx))

	require.Len(t, captured, 1)
	assert.True(t, captured[0].Passed)
}

// TestPolicyEvalHandler_Execute_DigestMismatch verifies a stale pinned digest fails the policy
// without evaluating it.
func TestPolicyEvalHandler_Execute_DigestMismatch(t *testing.T) {
	ctx := context.Background()
	client := jobclient.NewMockClient(t)

	pkg := makeBundleBytes(t, map[string]string{"policy.rego": `package tharsis.test`})

	client.On("GetRunVariables", ctx, "run-1", false).Return([]*pb.RunVariable{}, nil)
	client.On("DownloadPlanJSON", ctx, "plan-1", mock.Anything).Run(func(args mock.Arguments) {
		_, _ = args.Get(2).(io.Writer).Write([]byte(`{}`))
	}).Return(nil)
	// The run carries a constraint, not a version: the API resolves it, and the resolved version is
	// deliberately different from anything in the constraint so the handler cannot be reading it back.
	client.On("GetPackageVersion", ctx, "security", ">= 1.0.0").Return(
		&pb.PackageVersion{Metadata: &pb.ResourceMetadata{Id: "pv-1"}, Version: "1.2.0"}, nil)
	client.On("DownloadPackage", ctx, "pv-1").Return(io.NopCloser(bytes.NewReader(pkg)), nil)

	var captured []jobclient.RunPolicyOutcomeInput
	client.On("ReportRunPolicyOutcomes", ctx, "post-plan-policy-check-gid", mock.Anything).Run(func(args mock.Arguments) {
		captured = args.Get(2).([]jobclient.RunPolicyOutcomeInput)
	}).Return(nil)

	// A digest that does not match the package content.
	staleDigest := hex.EncodeToString(make([]byte, 32))

	handler := newTestHandler(client, []*pb.PolicyCheckPolicy{
		{Id: "pol-1", PackageSource: "security", PackageVersionConstraint: ">= 1.0.0", EnforcementLevel: pb.PolicyEnforcementLevel_POLICY_ENFORCEMENT_LEVEL_SOFT_MANDATORY, PackageDigest: &staleDigest, Provenance: &pb.PolicyCheckPolicyProvenance{PolicyTrn: "trn:policy:group/security"}},
	})
	require.NoError(t, handler.Execute(ctx))

	require.Len(t, captured, 1)
	assert.Equal(t, "pol-1", captured[0].PolicyID)
	assert.False(t, captured[0].Passed)
	assert.Contains(t, onlyMessage(t, captured[0]), "digest mismatch")
}

func TestPoliciesFromCheck(t *testing.T) {
	// One evalPolicy per policy, keyed by id; policyPath is the path of the owning policy's TRN. The
	// constraint is carried unresolved — resolvedVersion is filled in later, by Execute.
	digest1 := "abc123"
	check := &pb.PolicyCheck{
		Policies: []*pb.PolicyCheckPolicy{
			{
				Id:                       "pol-1",
				PackageSource:            "security",
				PackageVersionConstraint: "~> 1.0",
				PackageDigest:            &digest1,
				Provenance:               &pb.PolicyCheckPolicyProvenance{PolicyTrn: "trn:policy:group/require-tags"},
			},
			{
				Id:            "pol-2",
				PackageSource: "cost",
				Provenance:    &pb.PolicyCheckPolicyProvenance{PolicyTrn: "trn:policy:group/budget"},
			},
		},
	}

	policies, err := policiesFromCheck(check)
	require.NoError(t, err)
	require.Len(t, policies, 2)
	assert.Equal(t, "pol-1", policies[0].policyID)
	assert.Equal(t, "group/require-tags", policies[0].policyPath)
	assert.Equal(t, "security", policies[0].packageSource)
	assert.Equal(t, "~> 1.0", policies[0].versionConstraint)
	assert.Empty(t, policies[0].resolvedVersion)
	assert.Equal(t, "abc123", policies[0].digest)
	assert.Equal(t, "pol-2", policies[1].policyID)
	assert.Equal(t, "group/budget", policies[1].policyPath)
	assert.Empty(t, policies[1].versionConstraint) // empty constraint means latest
}

// TestExtractPolicyBundle_ComputesShaSum verifies extractPolicyBundle records the SHA-256 of the raw
// package stream it read, so the caller can verify the pinned digest against the returned bundle.
func TestExtractPolicyBundle_ComputesShaSum(t *testing.T) {
	pkg := makeBundleBytes(t, map[string]string{"policy.rego": `package tharsis.test`})
	sum := sha256.Sum256(pkg)
	want := hex.EncodeToString(sum[:])

	bundle, err := extractPolicyBundle(io.NopCloser(bytes.NewReader(pkg)))
	require.NoError(t, err)
	assert.Equal(t, want, bundle.ShaSum)
	assert.Contains(t, bundle.Modules, "policy.rego")
}

// TestConstraintLabel verifies the empty constraint is named rather than shown as nothing, since it
// is what a policy tracking the latest uploaded version carries.
func TestConstraintLabel(t *testing.T) {
	assert.Equal(t, "latest", constraintLabel(""))
	assert.Equal(t, "~> 1.0", constraintLabel("~> 1.0"))
}

// TestBuildInputDocument_PrePlanOmitsTfplan verifies the input document carries the stage and omits
// tfplan for a pre-plan stage (there is no plan yet), while a post-plan stage includes it.
func TestBuildInputDocument_PrePlanOmitsTfplan(t *testing.T) {
	h := &PolicyEvalHandler{
		run:       &pb.Run{Metadata: &pb.ResourceMetadata{Id: "run-1"}},
		workspace: &pb.Workspace{Metadata: &pb.ResourceMetadata{Id: "ws-1"}, FullPath: "group/ws"},
	}

	preInput := h.buildInputDocument(pb.RunTaskStageName_RUN_TASK_STAGE_NAME_PRE_PLAN, nil, nil, nil)
	assert.Equal(t, "pre_plan", preInput["stage"])
	_, hasPlan := preInput["tfplan"]
	assert.False(t, hasPlan, "pre-plan input must not include tfplan")
	assert.NotNil(t, preInput["run"])

	postInput := h.buildInputDocument(pb.RunTaskStageName_RUN_TASK_STAGE_NAME_POST_PLAN, nil, map[string]interface{}{"format_version": "1.0"}, nil)
	assert.Equal(t, "post_plan", postInput["stage"])
	assert.NotNil(t, postInput["tfplan"], "post-plan input includes the plan JSON")
	_, hasState := postInput["tfstate"]
	assert.False(t, hasState, "post-plan input must not include tfstate")
}

// TestBuildInputDocument_PostApplyIncludesTfstate verifies a post-apply stage's input document
// carries both tfplan and tfstate, mirroring pre-apply's tfplan-only document plus state.
func TestBuildInputDocument_PostApplyIncludesTfstate(t *testing.T) {
	h := &PolicyEvalHandler{
		run:       &pb.Run{Metadata: &pb.ResourceMetadata{Id: "run-1"}},
		workspace: &pb.Workspace{Metadata: &pb.ResourceMetadata{Id: "ws-1"}, FullPath: "group/ws"},
	}

	preApplyInput := h.buildInputDocument(pb.RunTaskStageName_RUN_TASK_STAGE_NAME_PRE_APPLY, nil, map[string]interface{}{"format_version": "1.0"}, nil)
	assert.Equal(t, "pre_apply", preApplyInput["stage"])
	assert.NotNil(t, preApplyInput["tfplan"], "pre-apply input includes the plan JSON")
	_, hasState := preApplyInput["tfstate"]
	assert.False(t, hasState, "pre-apply input must not include tfstate")

	postApplyInput := h.buildInputDocument(
		pb.RunTaskStageName_RUN_TASK_STAGE_NAME_POST_APPLY,
		nil,
		map[string]interface{}{"format_version": "1.0"},
		map[string]interface{}{"format_version": "1.0", "values": map[string]interface{}{}},
	)
	assert.Equal(t, "post_apply", postApplyInput["stage"])
	assert.NotNil(t, postApplyInput["tfplan"], "post-apply input includes the plan JSON")
	assert.NotNil(t, postApplyInput["tfstate"], "post-apply input includes the state JSON")
}

// TestBuildInputDocument_AssessmentRunFlag verifies a policy can tell a scheduled drift assessment
// apart from a run someone asked for. Both are speculative and refresh-only, so the flag is the only
// thing that distinguishes them.
func TestBuildInputDocument_AssessmentRunFlag(t *testing.T) {
	forRun := func(run *pb.Run) map[string]interface{} {
		h := &PolicyEvalHandler{run: run}
		input := h.buildInputDocument(pb.RunTaskStageName_RUN_TASK_STAGE_NAME_PRE_PLAN, nil, nil, nil)
		runDoc, ok := input["run"].(map[string]interface{})
		require.True(t, ok, "input document carries a run object")
		return runDoc
	}

	assessment := forRun(&pb.Run{
		Metadata:        &pb.ResourceMetadata{Id: "run-1"},
		Speculative:     true,
		RefreshOnly:     true,
		IsAssessmentRun: true,
	})
	assert.Equal(t, true, assessment["isAssessmentRun"])
	assert.Equal(t, true, assessment["speculative"])

	userRun := forRun(&pb.Run{
		Metadata:    &pb.ResourceMetadata{Id: "run-2"},
		Speculative: true,
		RefreshOnly: true,
	})
	assert.Equal(t, false, userRun["isAssessmentRun"])
}

// TestPolicyEvalHandler_Execute_PrePlanSkipsPlanJSON verifies a pre-plan check evaluates without
// downloading the plan JSON (DownloadPlanJSON is never called) and still reports outcomes.
func TestPolicyEvalHandler_Execute_PrePlanSkipsPlanJSON(t *testing.T) {
	ctx := context.Background()
	client := jobclient.NewMockClient(t)

	client.On("GetRunVariables", ctx, "run-1", false).Return([]*pb.RunVariable{}, nil)
	// DownloadPlanJSON is intentionally NOT expected: a pre-plan check must not fetch the plan JSON.
	// The mock would fail the test on an unexpected call.

	// A rule that denies based on the stage input, which is available without a tfplan.
	rego := `package tharsis.test

deny contains msg if {
    input.stage == "pre_plan"
    msg := "pre-plan gate blocks this run"
}`
	// The run carries a constraint, not a version: the API resolves it, and the resolved version is
	// deliberately different from anything in the constraint so the handler cannot be reading it back.
	client.On("GetPackageVersion", ctx, "security", ">= 1.0.0").Return(
		&pb.PackageVersion{Metadata: &pb.ResourceMetadata{Id: "pv-1"}, Version: "1.2.0"}, nil)
	client.On("DownloadPackage", ctx, "pv-1").Return(makeBundle(t, map[string]string{"policy.rego": rego}), nil)

	var captured []jobclient.RunPolicyOutcomeInput
	client.On("ReportRunPolicyOutcomes", ctx, "pre-plan-policy-check-gid", mock.Anything).Run(func(args mock.Arguments) {
		captured = args.Get(2).([]jobclient.RunPolicyOutcomeInput)
	}).Return(nil)

	handler := &PolicyEvalHandler{
		cancellableCtx: context.Background(),
		run: &pb.Run{
			Metadata: &pb.ResourceMetadata{Id: "run-1"},
			PlanId:   "plan-1",
			TaskStages: []*pb.RunTaskStage{{
				Id:        "pre-plan-stage-gid",
				StageName: pb.RunTaskStageName_RUN_TASK_STAGE_NAME_PRE_PLAN,
				Status:    pb.RunTaskStageStatus_RUN_TASK_STAGE_STATUS_RUNNING,
				PolicyChecks: []*pb.PolicyCheck{{
					Id:        "pre-plan-policy-check-gid",
					CheckType: pb.PolicyCheckType_POLICY_CHECK_TYPE_OPA,
					StageName: pb.RunTaskStageName_RUN_TASK_STAGE_NAME_PRE_PLAN,
					Status:    pb.PolicyCheckStatus_POLICY_CHECK_STATUS_RUNNING,
					Policies: []*pb.PolicyCheckPolicy{{
						Id:                       "pol-1",
						PackageSource:            "security",
						PackageVersionConstraint: ">= 1.0.0",
						EnforcementLevel:         pb.PolicyEnforcementLevel_POLICY_ENFORCEMENT_LEVEL_HARD_MANDATORY,
						Provenance:               &pb.PolicyCheckPolicyProvenance{PolicyTrn: "trn:policy:group/security"},
					}},
				}},
			}},
		},
		workspace: &pb.Workspace{Metadata: &pb.ResourceMetadata{Id: "ws-1"}, FullPath: "group/ws"},
		job: &pb.Job{
			Metadata: &pb.ResourceMetadata{Id: "job-1"},
			JobData:  &pb.Job_OpaData{OpaData: &pb.OPAJobData{PolicyCheckId: "pre-plan-policy-check-gid"}},
		},
		jobLogger: noopJobLogger{},
		client:    client,
	}

	require.NoError(t, handler.Execute(ctx))
	require.Len(t, captured, 1)
	assert.False(t, captured[0].Passed)
	assert.Contains(t, onlyMessage(t, captured[0]), "pre-plan gate blocks this run")
}

// newApplyPhaseCheckRun builds a run whose single OPA policy check is at the given apply-phase stage
// (pre_apply or post_apply), mirroring newTestRun's shape for the post-plan stage.
func newApplyPhaseCheckRun(stage pb.RunTaskStageName, policies []*pb.PolicyCheckPolicy) *pb.Run {
	return &pb.Run{
		Metadata: &pb.ResourceMetadata{Id: "run-1"},
		PlanId:   "plan-1",
		TaskStages: []*pb.RunTaskStage{
			{
				Id:        "stage-gid",
				StageName: stage,
				Status:    pb.RunTaskStageStatus_RUN_TASK_STAGE_STATUS_RUNNING,
				PolicyChecks: []*pb.PolicyCheck{
					{
						Id:        "check-gid",
						CheckType: pb.PolicyCheckType_POLICY_CHECK_TYPE_OPA,
						StageName: stage,
						Status:    pb.PolicyCheckStatus_POLICY_CHECK_STATUS_RUNNING,
						Policies:  policies,
					},
				},
			},
		},
	}
}

// TestPolicyEvalHandler_Execute_PreApplySkipsStateVersion verifies a pre-apply check downloads the
// plan JSON but never the state version (DownloadStateVersion is not expected — the mock would fail
// the test on an unexpected call), since state has not been written yet at that stage.
func TestPolicyEvalHandler_Execute_PreApplySkipsStateVersion(t *testing.T) {
	ctx := context.Background()
	client := jobclient.NewMockClient(t)

	client.On("GetRunVariables", ctx, "run-1", false).Return([]*pb.RunVariable{}, nil)
	client.On("DownloadPlanJSON", ctx, "plan-1", mock.Anything).Run(func(args mock.Arguments) {
		w := args.Get(2).(io.Writer)
		_, _ = w.Write([]byte(`{"format_version":"1.0"}`))
	}).Return(nil)

	rego := `package tharsis.test

deny contains msg if {
    input.stage == "pre_apply"
    msg := "pre-apply gate"
}`
	client.On("GetPackageVersion", ctx, "security", "").Return(
		&pb.PackageVersion{Metadata: &pb.ResourceMetadata{Id: "pv-1"}, Version: "1.0.0"}, nil)
	client.On("DownloadPackage", ctx, "pv-1").Return(makeBundle(t, map[string]string{"policy.rego": rego}), nil)

	var captured []jobclient.RunPolicyOutcomeInput
	client.On("ReportRunPolicyOutcomes", ctx, "check-gid", mock.Anything).Run(func(args mock.Arguments) {
		captured = args.Get(2).([]jobclient.RunPolicyOutcomeInput)
	}).Return(nil)

	handler := &PolicyEvalHandler{
		cancellableCtx: context.Background(),
		run: newApplyPhaseCheckRun(pb.RunTaskStageName_RUN_TASK_STAGE_NAME_PRE_APPLY, []*pb.PolicyCheckPolicy{
			{
				Id:            "pol-1",
				PackageSource: "security",
				Provenance:    &pb.PolicyCheckPolicyProvenance{PolicyTrn: "trn:policy:group/security"},
			},
		}),
		workspace: &pb.Workspace{Metadata: &pb.ResourceMetadata{Id: "ws-1"}, FullPath: "group/ws"},
		job: &pb.Job{
			Metadata: &pb.ResourceMetadata{Id: "job-1"},
			JobData:  &pb.Job_OpaData{OpaData: &pb.OPAJobData{PolicyCheckId: "check-gid"}},
		},
		jobLogger: noopJobLogger{},
		client:    client,
	}

	require.NoError(t, handler.Execute(ctx))
	require.Len(t, captured, 1)
	assert.False(t, captured[0].Passed)
	assert.Contains(t, onlyMessage(t, captured[0]), "pre-apply gate")
}

// TestPolicyEvalHandler_Execute_PostApplyDownloadsStateVersion verifies a post-apply check downloads
// both the plan JSON and the JSON representation of the state version *this run* created, and that the
// state document is available to the policy as input.tfstate. The workspace's current state version is
// deliberately a different one: the two diverge whenever anything writes state after the apply, and the
// policy must see what the run produced.
func TestPolicyEvalHandler_Execute_PostApplyDownloadsStateVersion(t *testing.T) {
	ctx := context.Background()
	client := jobclient.NewMockClient(t)

	client.On("GetRunVariables", ctx, "run-1", false).Return([]*pb.RunVariable{}, nil)
	client.On("DownloadPlanJSON", ctx, "plan-1", mock.Anything).Run(func(args mock.Arguments) {
		w := args.Get(2).(io.Writer)
		_, _ = w.Write([]byte(`{"format_version":"1.0"}`))
	}).Return(nil)
	client.On("GetRunStateVersion", ctx, "run-1").Return(
		&pb.StateVersion{Metadata: &pb.ResourceMetadata{Id: "sv-run-1"}}, nil)
	client.On("DownloadStateVersionJSON", ctx, "sv-run-1", mock.Anything).Run(func(args mock.Arguments) {
		w := args.Get(2).(io.Writer)
		_, _ = w.Write([]byte(`{"format_version":"1.0","values":{"root_module":{}}}`))
	}).Return(nil)

	rego := `package tharsis.test

deny contains msg if {
    input.tfstate.format_version == "1.0"
    msg := "state was present"
}`
	client.On("GetPackageVersion", ctx, "security", "").Return(
		&pb.PackageVersion{Metadata: &pb.ResourceMetadata{Id: "pv-1"}, Version: "1.0.0"}, nil)
	client.On("DownloadPackage", ctx, "pv-1").Return(makeBundle(t, map[string]string{"policy.rego": rego}), nil)

	var captured []jobclient.RunPolicyOutcomeInput
	client.On("ReportRunPolicyOutcomes", ctx, "check-gid", mock.Anything).Run(func(args mock.Arguments) {
		captured = args.Get(2).([]jobclient.RunPolicyOutcomeInput)
	}).Return(nil)

	handler := &PolicyEvalHandler{
		cancellableCtx: context.Background(),
		run: newApplyPhaseCheckRun(pb.RunTaskStageName_RUN_TASK_STAGE_NAME_POST_APPLY, []*pb.PolicyCheckPolicy{
			{
				Id:            "pol-1",
				PackageSource: "security",
				Provenance:    &pb.PolicyCheckPolicyProvenance{PolicyTrn: "trn:policy:group/security"},
			},
		}),
		workspace: &pb.Workspace{
			Metadata: &pb.ResourceMetadata{Id: "ws-1"},
			FullPath: "group/ws",
			// A different state version than the run's: nothing may fall back to it.
			CurrentStateVersionId: "sv-workspace-current",
		},
		job: &pb.Job{
			Metadata: &pb.ResourceMetadata{Id: "job-1"},
			JobData:  &pb.Job_OpaData{OpaData: &pb.OPAJobData{PolicyCheckId: "check-gid"}},
		},
		jobLogger: noopJobLogger{},
		client:    client,
	}

	require.NoError(t, handler.Execute(ctx))
	require.Len(t, captured, 1)
	assert.False(t, captured[0].Passed, "the deny rule should have fired against input.tfstate")
	assert.Contains(t, onlyMessage(t, captured[0]), "state was present")
}

// TestPolicyEvalHandler_Execute_PostApplyWithoutStateVersionOmitsTFState verifies a post-apply check
// still evaluates, with input.tfstate absent, when the run created no state version — an apply that
// wrote no state. The check is not failed over the missing key, and the workspace's current state
// version (which belongs to some earlier run) is not substituted for it.
func TestPolicyEvalHandler_Execute_PostApplyWithoutStateVersionOmitsTFState(t *testing.T) {
	ctx := context.Background()
	client := jobclient.NewMockClient(t)

	client.On("GetRunVariables", ctx, "run-1", false).Return([]*pb.RunVariable{}, nil)
	client.On("DownloadPlanJSON", ctx, "plan-1", mock.Anything).Run(func(args mock.Arguments) {
		w := args.Get(2).(io.Writer)
		_, _ = w.Write([]byte(`{"format_version":"1.0"}`))
	}).Return(nil)
	client.On("GetRunStateVersion", ctx, "run-1").Return(nil, status.Error(codes.NotFound, "run has no state version"))
	// DownloadStateVersionJSON is intentionally NOT expected: with no state version for this run there
	// is nothing to fetch, so the handler must skip the call rather than fall back to the workspace's.

	captured := expectStateAbsentEvaluation(ctx, t, client)

	handler := &PolicyEvalHandler{
		cancellableCtx: context.Background(),
		run: newApplyPhaseCheckRun(pb.RunTaskStageName_RUN_TASK_STAGE_NAME_POST_APPLY, []*pb.PolicyCheckPolicy{
			{
				Id:            "pol-1",
				PackageSource: "security",
				Provenance:    &pb.PolicyCheckPolicyProvenance{PolicyTrn: "trn:policy:group/security"},
			},
		}),
		workspace: &pb.Workspace{
			Metadata: &pb.ResourceMetadata{Id: "ws-1"},
			FullPath: "group/ws",
			// A state version from an earlier run; the handler must not reach for it.
			CurrentStateVersionId: "sv-workspace-current",
		},
		job: &pb.Job{
			Metadata: &pb.ResourceMetadata{Id: "job-1"},
			JobData:  &pb.Job_OpaData{OpaData: &pb.OPAJobData{PolicyCheckId: "check-gid"}},
		},
		jobLogger: noopJobLogger{},
		client:    client,
	}

	require.NoError(t, handler.Execute(ctx))
	require.Len(t, *captured, 1)
	assert.False(t, (*captured)[0].Passed, "the deny rule keyed on an absent input.tfstate should have fired")
	assert.Contains(t, onlyMessage(t, (*captured)[0]), "state was absent")
}

// TestPolicyEvalHandler_Execute_PostApplyStateVersionLookupErrorFails verifies that a failure other
// than NotFound while resolving the run's state version fails the job. Only "this run has no state
// version" is tolerated; a lookup that broke must not be read as an absence.
func TestPolicyEvalHandler_Execute_PostApplyStateVersionLookupErrorFails(t *testing.T) {
	ctx := context.Background()
	mockClient := jobclient.NewMockClient(t)

	mockClient.On("GetRunVariables", ctx, "run-1", false).Return([]*pb.RunVariable{}, nil)
	mockClient.On("DownloadPlanJSON", ctx, "plan-1", mock.Anything).Run(func(args mock.Arguments) {
		w := args.Get(2).(io.Writer)
		_, _ = w.Write([]byte(`{"format_version":"1.0"}`))
	}).Return(nil)
	mockClient.On("GetRunStateVersion", ctx, "run-1").Return(nil, status.Error(codes.Unavailable, "backend down"))

	handler := &PolicyEvalHandler{
		cancellableCtx: context.Background(),
		run: newApplyPhaseCheckRun(pb.RunTaskStageName_RUN_TASK_STAGE_NAME_POST_APPLY, []*pb.PolicyCheckPolicy{
			{
				Id:            "pol-1",
				PackageSource: "security",
				Provenance:    &pb.PolicyCheckPolicyProvenance{PolicyTrn: "trn:policy:group/security"},
			},
		}),
		workspace: &pb.Workspace{Metadata: &pb.ResourceMetadata{Id: "ws-1"}, FullPath: "group/ws"},
		job: &pb.Job{
			Metadata: &pb.ResourceMetadata{Id: "job-1"},
			JobData:  &pb.Job_OpaData{OpaData: &pb.OPAJobData{PolicyCheckId: "check-gid"}},
		},
		jobLogger: noopJobLogger{},
		client:    mockClient,
	}

	err := handler.Execute(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get the run's state version")
}

// TestPolicyEvalHandler_Execute_PostApplyStateVersionJSONNotFoundOmitsTFState verifies that a state
// version with no stored JSON representation — which is what an older job executor produces, since it
// uploads only the raw state — leaves input.tfstate absent instead of failing the check. This is what
// lets policy evaluation keep working while runners are upgraded.
func TestPolicyEvalHandler_Execute_PostApplyStateVersionJSONNotFoundOmitsTFState(t *testing.T) {
	ctx := context.Background()
	mockClient := jobclient.NewMockClient(t)

	mockClient.On("GetRunVariables", ctx, "run-1", false).Return([]*pb.RunVariable{}, nil)
	mockClient.On("DownloadPlanJSON", ctx, "plan-1", mock.Anything).Run(func(args mock.Arguments) {
		w := args.Get(2).(io.Writer)
		_, _ = w.Write([]byte(`{"format_version":"1.0"}`))
	}).Return(nil)
	mockClient.On("GetRunStateVersion", ctx, "run-1").Return(
		&pb.StateVersion{Metadata: &pb.ResourceMetadata{Id: "sv-run-1"}}, nil)
	mockClient.On("DownloadStateVersionJSON", ctx, "sv-run-1", mock.Anything).
		Return(fmt.Errorf("%w: no rendering", client.ErrStateVersionJSONNotFound))

	captured := expectStateAbsentEvaluation(ctx, t, mockClient)

	handler := &PolicyEvalHandler{
		cancellableCtx: context.Background(),
		run: newApplyPhaseCheckRun(pb.RunTaskStageName_RUN_TASK_STAGE_NAME_POST_APPLY, []*pb.PolicyCheckPolicy{
			{
				Id:            "pol-1",
				PackageSource: "security",
				Provenance:    &pb.PolicyCheckPolicyProvenance{PolicyTrn: "trn:policy:group/security"},
			},
		}),
		workspace: &pb.Workspace{
			Metadata: &pb.ResourceMetadata{Id: "ws-1"},
			FullPath: "group/ws",
			// A different state version than the run's: nothing may fall back to it.
			CurrentStateVersionId: "sv-workspace-current",
		},
		job: &pb.Job{
			Metadata: &pb.ResourceMetadata{Id: "job-1"},
			JobData:  &pb.Job_OpaData{OpaData: &pb.OPAJobData{PolicyCheckId: "check-gid"}},
		},
		jobLogger: noopJobLogger{},
		client:    mockClient,
	}

	require.NoError(t, handler.Execute(ctx))
	require.Len(t, *captured, 1)
	assert.False(t, (*captured)[0].Passed, "the deny rule keyed on an absent input.tfstate should have fired")
	assert.Contains(t, onlyMessage(t, (*captured)[0]), "state was absent")
}

// TestPolicyEvalHandler_Execute_PostApplyStateVersionJSONErrorFails verifies that a transport failure
// fetching the JSON representation still fails the job. Only a documented absence is tolerated; an
// error that might mean "the artifact exists but we could not read it" must not be mistaken for one.
func TestPolicyEvalHandler_Execute_PostApplyStateVersionJSONErrorFails(t *testing.T) {
	ctx := context.Background()
	mockClient := jobclient.NewMockClient(t)

	mockClient.On("GetRunVariables", ctx, "run-1", false).Return([]*pb.RunVariable{}, nil)
	mockClient.On("DownloadPlanJSON", ctx, "plan-1", mock.Anything).Run(func(args mock.Arguments) {
		w := args.Get(2).(io.Writer)
		_, _ = w.Write([]byte(`{"format_version":"1.0"}`))
	}).Return(nil)
	mockClient.On("GetRunStateVersion", ctx, "run-1").Return(
		&pb.StateVersion{Metadata: &pb.ResourceMetadata{Id: "sv-run-1"}}, nil)
	mockClient.On("DownloadStateVersionJSON", ctx, "sv-run-1", mock.Anything).
		Return(errors.New("connection reset"))

	handler := &PolicyEvalHandler{
		cancellableCtx: context.Background(),
		run: newApplyPhaseCheckRun(pb.RunTaskStageName_RUN_TASK_STAGE_NAME_POST_APPLY, []*pb.PolicyCheckPolicy{
			{
				Id:            "pol-1",
				PackageSource: "security",
				Provenance:    &pb.PolicyCheckPolicyProvenance{PolicyTrn: "trn:policy:group/security"},
			},
		}),
		workspace: &pb.Workspace{
			Metadata: &pb.ResourceMetadata{Id: "ws-1"},
			FullPath: "group/ws",
			// A different state version than the run's: nothing may fall back to it.
			CurrentStateVersionId: "sv-workspace-current",
		},
		job: &pb.Job{
			Metadata: &pb.ResourceMetadata{Id: "job-1"},
			JobData:  &pb.Job_OpaData{OpaData: &pb.OPAJobData{PolicyCheckId: "check-gid"}},
		},
		jobLogger: noopJobLogger{},
		client:    mockClient,
	}

	err := handler.Execute(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to download state version JSON")
}

// expectStateAbsentEvaluation registers the package/bundle/report expectations for a policy whose deny
// rule fires only when input.tfstate is absent, and returns where the reported outcomes land.
func expectStateAbsentEvaluation(ctx context.Context, t *testing.T, mockClient *jobclient.MockClient) *[]jobclient.RunPolicyOutcomeInput {
	rego := `package tharsis.test

deny contains msg if {
    not input.tfstate
    msg := "state was absent"
}`
	mockClient.On("GetPackageVersion", ctx, "security", "").Return(
		&pb.PackageVersion{Metadata: &pb.ResourceMetadata{Id: "pv-1"}, Version: "1.0.0"}, nil)
	mockClient.On("DownloadPackage", ctx, "pv-1").Return(makeBundle(t, map[string]string{"policy.rego": rego}), nil)

	captured := &[]jobclient.RunPolicyOutcomeInput{}
	mockClient.On("ReportRunPolicyOutcomes", ctx, "check-gid", mock.Anything).Run(func(args mock.Arguments) {
		*captured = args.Get(2).([]jobclient.RunPolicyOutcomeInput)
	}).Return(nil)

	return captured
}
