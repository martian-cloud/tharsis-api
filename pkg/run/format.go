package run

import (
	"fmt"
	"strings"

	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
)

// Helpers for describing policy findings and gates, shared by the observers that render them. Fields
// are read through their getters so that results missing a field (from an API newer or older than
// this package) describe what they can rather than panic.

// NeedsOverride reports whether a gate can only be cleared by an override: its failed policies
// declared no approvers, so it has no approval rules for anyone to satisfy.
func NeedsOverride(g *pb.RunGate) bool {
	return len(g.GetApprovalRules()) == 0
}

// PolicyName is the policy's resource path (group path and name) from its TRN, falling back to the
// full TRN if it doesn't parse, and to the policy's ID when there is no TRN.
func PolicyName(p *pb.PolicyCheckPolicy) string {
	policyTRN := p.GetProvenance().GetPolicyTrn()
	if policyTRN == "" {
		return p.GetId()
	}
	parsed, err := trn.ParseAny(policyTRN)
	if err != nil {
		return policyTRN
	}
	return parsed.Path()
}

// PolicyNames maps each of a check's policy IDs to its display name, the key a gate's rules use.
func PolicyNames(results *pb.PolicyCheckResults) map[string]string {
	names := map[string]string{}
	for _, r := range results.GetPolicyResults() {
		names[r.GetPolicy().GetId()] = PolicyName(r.GetPolicy())
	}
	return names
}

// AdvisoryFailures names the check's policies that failed at the advisory level.
func AdvisoryFailures(results *pb.PolicyCheckResults) []string {
	var names []string
	for _, r := range results.GetPolicyResults() {
		p := r.GetPolicy()
		if p.GetStatus() == pb.PolicyCheckPolicyStatus_POLICY_CHECK_POLICY_STATUS_FAILED &&
			p.GetEnforcementLevel() == pb.PolicyEnforcementLevel_POLICY_ENFORCEMENT_LEVEL_ADVISORY {
			names = append(names, PolicyName(p))
		}
	}
	return names
}

// FailedPolicyCount counts the check's policies that failed, at any enforcement level.
func FailedPolicyCount(results *pb.PolicyCheckResults) int {
	n := 0
	for _, r := range results.GetPolicyResults() {
		if r.GetPolicy().GetStatus() == pb.PolicyCheckPolicyStatus_POLICY_CHECK_POLICY_STATUS_FAILED {
			n++
		}
	}
	return n
}

// GateRuleLines describes each of a gate's approval rules, with the policy names aligned, for
// example "group/encrypted-buckets  0 of 2 approvals".
func GateRuleLines(g *pb.RunGate, results *pb.PolicyCheckResults) []string {
	names := PolicyNames(results)
	rules := g.GetApprovalRules()

	width := 0
	ruleNames := make([]string, len(rules))
	for i, rule := range rules {
		ruleNames[i] = ruleName(names, rule)
		width = max(width, len(ruleNames[i]))
	}

	lines := make([]string, 0, len(rules))
	for i, rule := range rules {
		lines = append(lines, fmt.Sprintf("%-*s  %s", width, ruleNames[i], ApprovalCount(rule)))
	}
	return lines
}

// GateSummary is a one-line form of a gate's progress, such as "1 of 2 approvals".
func GateSummary(g *pb.RunGate, results *pb.PolicyCheckResults) string {
	names := PolicyNames(results)
	rules := g.GetApprovalRules()
	parts := make([]string, 0, len(rules))
	for _, rule := range rules {
		part := ApprovalCount(rule)
		if len(rules) > 1 {
			part = ruleName(names, rule) + ": " + part
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, "; ")
}

func ruleName(names map[string]string, rule *pb.RunGateApprovalRule) string {
	if name := names[rule.GetName()]; name != "" {
		return name
	}
	return rule.GetName()
}

// ApprovalCount describes a rule's progress, such as "0 of 2 approvals".
func ApprovalCount(rule *pb.RunGateApprovalRule) string {
	return fmt.Sprintf("%d of %d %s", rule.GetApprovals(), rule.GetRequiredApprovals(),
		Plural(int(rule.GetRequiredApprovals()), "approval", "approvals"))
}

// Plural returns one when n is 1, and many otherwise.
func Plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// PolicyCheckTypeName names a check type for a verdict line, such as "OPA policies".
func PolicyCheckTypeName(t pb.PolicyCheckType) string {
	switch t {
	case pb.PolicyCheckType_POLICY_CHECK_TYPE_OPA:
		return "OPA policies"
	case pb.PolicyCheckType_POLICY_CHECK_TYPE_MODULE_ATTESTATION:
		return "Module attestation"
	default:
		return "Policies"
	}
}

// CheckTypeLabel names a check type for display. A type this package was built without has no name,
// so its number is shown.
func CheckTypeLabel(t pb.PolicyCheckType) string {
	if _, ok := pb.PolicyCheckType_name[int32(t)]; !ok {
		return fmt.Sprintf("(%d)", int32(t))
	}
	return strings.ToLower(strings.TrimPrefix(t.String(), "POLICY_CHECK_TYPE_"))
}

// EnforcementLevelName names an enforcement level, such as "soft-mandatory".
func EnforcementLevelName(l pb.PolicyEnforcementLevel) string {
	if _, ok := pb.PolicyEnforcementLevel_name[int32(l)]; !ok {
		// A level added after this package was built.
		return "unknown"
	}
	return strings.ReplaceAll(strings.ToLower(strings.TrimPrefix(l.String(), "POLICY_ENFORCEMENT_LEVEL_")), "_", "-")
}
