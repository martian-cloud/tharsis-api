package models

import (
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func approval(userID string, decision RunGateDecision, coveredRules ...string) RunGateApproval {
	uid := userID
	return RunGateApproval{UserID: &uid, Decision: decision, CoveredRules: coveredRules}
}

func TestRunGate_Satisfied(t *testing.T) {
	tests := []struct {
		name      string
		rules     []*RunGateApprovalRule
		approvals []RunGateApproval
		want      bool
	}{
		{
			name:  "no rules is satisfied",
			rules: nil,
			want:  true,
		},
		{
			name:  "single rule below threshold is not satisfied",
			rules: []*RunGateApprovalRule{{Name: "p1", RequiredApprovals: 2}},
			approvals: []RunGateApproval{
				approval("u1", RunGateDecisionApprove, "p1"),
			},
			want: false,
		},
		{
			name:  "single rule meeting threshold is satisfied",
			rules: []*RunGateApprovalRule{{Name: "p1", RequiredApprovals: 2}},
			approvals: []RunGateApproval{
				approval("u1", RunGateDecisionApprove, "p1"),
				approval("u2", RunGateDecisionApprove, "p1"),
			},
			want: true,
		},
		{
			name:  "duplicate approvals from the same principal count once",
			rules: []*RunGateApprovalRule{{Name: "p1", RequiredApprovals: 2}},
			approvals: []RunGateApproval{
				approval("u1", RunGateDecisionApprove, "p1"),
				approval("u1", RunGateDecisionApprove, "p1"),
			},
			want: false,
		},
		{
			name:  "reject decisions never count toward a rule",
			rules: []*RunGateApprovalRule{{Name: "p1", RequiredApprovals: 1}},
			approvals: []RunGateApproval{
				approval("u1", RunGateDecisionReject, "p1"),
			},
			want: false,
		},
		{
			name: "all rules must be met",
			rules: []*RunGateApprovalRule{
				{Name: "p1", RequiredApprovals: 1},
				{Name: "p2", RequiredApprovals: 1},
			},
			approvals: []RunGateApproval{
				approval("u1", RunGateDecisionApprove, "p1"),
			},
			want: false,
		},
		{
			name: "one approval covering multiple rules counts for each",
			rules: []*RunGateApprovalRule{
				{Name: "p1", RequiredApprovals: 1},
				{Name: "p2", RequiredApprovals: 1},
			},
			approvals: []RunGateApproval{
				approval("u1", RunGateDecisionApprove, "p1", "p2"),
			},
			want: true,
		},
		{
			name:  "rule with zero required approvals is met",
			rules: []*RunGateApprovalRule{{Name: "p1", RequiredApprovals: 0}},
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gate := &RunGate{ApprovalRules: tt.rules}
			assert.Equal(t, tt.want, gate.Satisfied(tt.approvals))
		})
	}
}

func TestRunGate_Validate(t *testing.T) {
	valid := func(mutate func(*RunGate)) *RunGate {
		g := &RunGate{
			Type:   RunGateTypeOPAPolicy,
			Status: RunGatePending,
		}
		if mutate != nil {
			mutate(g)
		}
		return g
	}

	tests := []struct {
		name            string
		gate            *RunGate
		expectErrorCode errors.CodeType
	}{
		{
			name: "valid pending gate",
			gate: valid(nil),
		},
		{
			name: "valid overridden gate",
			gate: valid(func(g *RunGate) {
				g.Status = RunGateOverridden
				g.OverriddenBy = ptr.String("admin@example.com")
				g.OverrideComment = ptr.String("bypassed for incident response")
			}),
		},
		{
			name:            "unsupported type",
			gate:            valid(func(g *RunGate) { g.Type = RunGateType("bogus") }),
			expectErrorCode: errors.EInvalid,
		},
		{
			name:            "unsupported status",
			gate:            valid(func(g *RunGate) { g.Status = RunGateStatus("bogus") }),
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "overridden gate without a subject is invalid",
			gate: valid(func(g *RunGate) {
				g.Status = RunGateOverridden
			}),
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "overriddenBy set on a non-overridden gate is invalid",
			gate: valid(func(g *RunGate) {
				g.Status = RunGateApproved
				g.OverriddenBy = ptr.String("admin@example.com")
			}),
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "overrideComment set on a non-overridden gate is invalid",
			gate: valid(func(g *RunGate) {
				g.OverrideComment = ptr.String("why")
			}),
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "override comment too long is invalid",
			gate: valid(func(g *RunGate) {
				g.Status = RunGateOverridden
				g.OverriddenBy = ptr.String("admin@example.com")
				long := make([]byte, maxOverrideCommentLength+1)
				for i := range long {
					long[i] = 'a'
				}
				g.OverrideComment = ptr.String(string(long))
			}),
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.gate.Validate()
			if tt.expectErrorCode != "" {
				require.Error(t, err)
				assert.Equal(t, tt.expectErrorCode, errors.ErrorCode(err))
				return
			}
			require.NoError(t, err)
		})
	}
}
