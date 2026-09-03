// Shared vocabulary and helpers for the task stage components. Every component reads its own Relay
// fragment, so the helpers below are generic over the minimal shape they need rather than tied to one
// generated type — that way a caller keeps its precise fragment type through the call.

// PolicyCheckPolicy.status is a bare String on the API, not an enum: '' until the check has been
// evaluated, then PENDING | PASSED | FAILED.
export const POLICY_PASSED = 'PASSED';
export const POLICY_FAILED = 'FAILED';

// PolicyCheckStatus values in which the check is still producing job output — everything else
// (PASSED, SOFT_FAILED, ERRORED, CANCELED, OVERRIDDEN, SKIPPED) is a status the job has already
// stopped running to reach.
const NOT_FINAL_POLICY_CHECK_STATUSES = ['CREATED', 'PENDING', 'QUEUED', 'RUNNING'];

export function isPolicyCheckFinal(status: string): boolean {
    return !NOT_FINAL_POLICY_CHECK_STATUSES.includes(status);
}

// PolicyCheckStatus values reached before the check has entered the queue: the run/task stage
// exists, but no policy-eval job has been created yet (mirrors PolicyCheckStatus.NotStarted on the
// backend). There is nothing yet for the panel to show at these statuses — no job, no progress, not
// even something a runner could be waiting to pick up — so callers hold off rendering it until the
// check has moved past them.
const NOT_STARTED_POLICY_CHECK_STATUSES = ['CREATED', 'PENDING'];

export function isPolicyCheckNotStarted(status: string): boolean {
    return NOT_STARTED_POLICY_CHECK_STATUSES.includes(status);
}

// stageLabel turns a RunTaskStageName into the display label used in the stage card and the stage
// header, which have to agree. The return type is narrowed to the literals RunDetailsStageHeader
// accepts, since it shares this vocabulary with the plan and apply stages.
export function stageLabel(stageName: string): 'Pre-Plan' | 'Post-Plan' | 'Pre-Apply' | 'Post-Apply' {
    switch (stageName) {
        case 'PRE_PLAN': return 'Pre-Plan';
        case 'PRE_APPLY': return 'Pre-Apply';
        case 'POST_APPLY': return 'Post-Apply';
        default: return 'Post-Plan';
    }
}


// approvalsForRule returns the decisions that cover a given rule. coveredRules exists precisely
// because one decision can satisfy several rules at once.
export function approvalsForRule<T extends { readonly coveredRules: readonly string[] }>(
    gate: { readonly approvals: readonly T[] } | null | undefined,
    ruleName: string,
): readonly T[] {
    if (!gate) {
        return [];
    }
    return gate.approvals.filter(approval => approval.coveredRules.includes(ruleName));
}

// gateProgress counts a gate's approval progress in policies rather than decisions: a policy is
// cleared once its rule has collected the approvals it requires. Failures no rule governs (a
// hard-mandatory policy is not overridable by approval) are excluded, so gated can be smaller than
// the failed list passed in.
export function gateProgress<
    R extends { readonly name: string, readonly requiredApprovals: number },
    A extends { readonly decision: string, readonly coveredRules: readonly string[] },
>(
    gate: { readonly approvalRules: readonly R[], readonly approvals: readonly A[] } | null | undefined,
    failed: readonly { readonly id: string }[],
): { gated: number, approved: number } {
    if (!gate) {
        return { gated: 0, approved: 0 };
    }

    const rules = failed.map(policy => findRule(gate, policy.id)).filter((rule): rule is R => !!rule);
    const approved = rules.filter(rule =>
        approvalsForRule(gate, rule.name).filter(approval => approval.decision === 'APPROVE').length >= rule.requiredApprovals
    );

    return { gated: rules.length, approved: approved.length };
}

// findRule locates the rule gating a policy within its check's gate. A rule's name is the
// PolicyCheckPolicy id it gates, which is the only join key between the two. Undefined means the
// policy is not gated — it passed, or its failure is not overridable by approval.
export function findRule<T extends { readonly name: string }>(
    gate: { readonly approvalRules: readonly T[] } | null | undefined,
    policyId: string,
): T | undefined {
    return gate?.approvalRules.find(rule => rule.name === policyId);
}
