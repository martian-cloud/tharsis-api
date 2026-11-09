package run

import (
	"testing"

	"github.com/stretchr/testify/assert"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/run/runtest"
)

func TestGateSummary(t *testing.T) {
	results := runtest.SoftFailedResults("c1", 2, 1)
	results.RunGate.ApprovalRules = append(results.RunGate.ApprovalRules, &pb.RunGateApprovalRule{Name: "unknown", RequiredApprovals: 1})

	summary := GateSummary(results.RunGate, results)

	assert.Contains(t, summary, "group/encrypted-buckets: ")
	assert.Contains(t, summary, "unknown: ", "a rule with no matching policy falls back to its name")
}

func TestGateRuleLines(t *testing.T) {
	results := runtest.SoftFailedResults("c1", 2, 1)
	results.PolicyResults = append(results.PolicyResults, &pb.PolicyCheckPolicyResult{Policy: &pb.PolicyCheckPolicy{
		Id:         "p2",
		Provenance: &pb.PolicyCheckPolicyProvenance{PolicyTrn: "trn:policy:group/tags"},
	}})
	results.RunGate.ApprovalRules = append(results.RunGate.ApprovalRules, &pb.RunGateApprovalRule{Name: "p2", RequiredApprovals: 1})

	assert.Equal(t, []string{
		"group/encrypted-buckets  1 of 2 approvals",
		"group/tags               0 of 1 approval",
	}, GateRuleLines(results.RunGate, results))
}

func TestPolicyName(t *testing.T) {
	assert.Equal(t, "group/a", PolicyName(&pb.PolicyCheckPolicy{Provenance: &pb.PolicyCheckPolicyProvenance{PolicyTrn: "trn:policy:group/a"}}))
	assert.Equal(t, "not-a-trn", PolicyName(&pb.PolicyCheckPolicy{Provenance: &pb.PolicyCheckPolicyProvenance{PolicyTrn: "not-a-trn"}}))
	assert.Equal(t, "p1", PolicyName(&pb.PolicyCheckPolicy{Id: "p1"}))
}

func TestLabels(t *testing.T) {
	assert.Equal(t, "soft-mandatory", EnforcementLevelName(pb.PolicyEnforcementLevel_POLICY_ENFORCEMENT_LEVEL_SOFT_MANDATORY))
	assert.Equal(t, "unknown", EnforcementLevelName(pb.PolicyEnforcementLevel(42)))
	assert.Equal(t, "opa", CheckTypeLabel(pb.PolicyCheckType_POLICY_CHECK_TYPE_OPA))
	assert.Equal(t, "(99)", CheckTypeLabel(pb.PolicyCheckType(99)))
}
