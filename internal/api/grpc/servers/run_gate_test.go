package servers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
)

// TestToPBRunGate verifies a run gate converts with its enums, a RunNode GID for its policy check,
// and each rule's approval progress counted by distinct approving principal.
func TestToPBRunGate(t *testing.T) {
	now := time.Now()
	u1, u2 := "u1", "u2"
	gate := &models.RunGate{
		RunID:         "run-1",
		PolicyCheckID: "check-1",
		Type:          models.RunGateTypeOPAPolicy,
		Status:        models.RunGatePending,
		ApprovalRules: []*models.RunGateApprovalRule{
			{Name: "p1", RequiredApprovals: 2},
			{Name: "p2", RequiredApprovals: 1},
		},
		Metadata: models.ResourceMetadata{ID: "gate-1", CreationTimestamp: &now, LastUpdatedTimestamp: &now},
	}
	approvals := []models.RunGateApproval{
		{UserID: &u1, Decision: models.RunGateDecisionApprove, CoveredRules: []string{"p1"}},
		{UserID: &u1, Decision: models.RunGateDecisionApprove, CoveredRules: []string{"p1"}},
		{UserID: &u2, Decision: models.RunGateDecisionReject, CoveredRules: []string{"p1", "p2"}},
	}

	pbGate := toPBRunGate(gate, approvals)

	require.NotNil(t, pbGate.Metadata)
	assert.Equal(t, gate.GetGlobalID(), pbGate.Metadata.Id)
	assert.Equal(t, models.RunNodeGID("check-1"), pbGate.PolicyCheckId)
	assert.Equal(t, gid.ToGlobalID(types.RunModelType, "run-1"), pbGate.RunId)
	assert.Nil(t, pbGate.OverriddenBy)
	assert.Empty(t, pbGate.OverrideComment)
	assert.Equal(t, pb.RunGateType_RUN_GATE_TYPE_OPA_POLICY, pbGate.Type)
	assert.Equal(t, pb.RunGateStatus_RUN_GATE_STATUS_PENDING, pbGate.Status)
	require.Len(t, pbGate.ApprovalRules, 2)
	assert.Equal(t, &pb.RunGateApprovalRule{Name: "p1", RequiredApprovals: 2, Approvals: 1}, pbGate.ApprovalRules[0])
	assert.Equal(t, &pb.RunGateApprovalRule{Name: "p2", RequiredApprovals: 1, Approvals: 0}, pbGate.ApprovalRules[1])
}

// TestToPBRunGate_Override verifies an overridden gate reports who overrode it and why.
func TestToPBRunGate_Override(t *testing.T) {
	now := time.Now()
	by, comment := "admin@example.com", "hotfix"
	gate := &models.RunGate{
		Status:          models.RunGateOverridden,
		OverriddenBy:    &by,
		OverrideComment: &comment,
		Metadata:        models.ResourceMetadata{ID: "gate-1", CreationTimestamp: &now, LastUpdatedTimestamp: &now},
	}

	pbGate := toPBRunGate(gate, nil)

	require.NotNil(t, pbGate.OverriddenBy)
	assert.Equal(t, by, *pbGate.OverriddenBy)
	assert.Equal(t, comment, pbGate.OverrideComment)
}

// TestRunGateDecisionFromPB verifies every protobuf decision maps to its domain constant.
func TestRunGateDecisionFromPB(t *testing.T) {
	assert.Equal(t, models.RunGateDecisionApprove, enumFromPB[models.RunGateDecision](pb.RunGateDecision_RUN_GATE_DECISION_APPROVE, "RUN_GATE_DECISION_"))
	assert.Equal(t, models.RunGateDecisionReject, enumFromPB[models.RunGateDecision](pb.RunGateDecision_RUN_GATE_DECISION_REJECT, "RUN_GATE_DECISION_"))
}

// TestToPBRunGate_StatusesAreMapped verifies every run gate status and type has a protobuf counterpart.
func TestToPBRunGate_StatusesAreMapped(t *testing.T) {
	now := time.Now()
	for _, status := range []models.RunGateStatus{models.RunGatePending, models.RunGateApproved, models.RunGateOverridden, models.RunGateCanceled} {
		for _, gateType := range []models.RunGateType{models.RunGateTypeOPAPolicy, models.RunGateTypeModuleAttestation} {
			gate := &models.RunGate{
				Type:     gateType,
				Status:   status,
				Metadata: models.ResourceMetadata{ID: "gate-1", CreationTimestamp: &now, LastUpdatedTimestamp: &now},
			}
			pbGate := toPBRunGate(gate, nil)
			assert.NotEqualf(t, pb.RunGateStatus_RUN_GATE_STATUS_UNSPECIFIED, pbGate.Status, "status %q is unmapped", status)
			assert.NotEqualf(t, pb.RunGateType_RUN_GATE_TYPE_UNSPECIFIED, pbGate.Type, "type %q is unmapped", gateType)
		}
	}
}
