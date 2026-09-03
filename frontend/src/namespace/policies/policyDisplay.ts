// Display vocabulary for a policy's kind, stage, and enforcement level — shared by every view that
// shows a policy or a policy check, regardless of feature area (the policy list/card/detail pages
// under namespace/policies, and the run task-stage and approvals views that show a check's policy
// data). Kept separate from PolicyForm, which owns the *_OPTIONS arrays these labels are derived
// from plus the option descriptions and form-only validation — pulling the two apart is what lets a
// non-form view (a run's task stage, an approval gate card) show a policy's kind/stage/enforcement
// without importing form state and its dependencies.

// The kind label, keyed by the GraphQL enum value (e.g. MODULE_ATTESTATION), so every view that shows
// a policy's or a policy check's kind — the policy card, the policy detail page, a run's task stage,
// an approval gate card — agrees on its wording instead of showing the raw constant.
export const KIND_LABELS: Record<string, string> = {
    OPA: 'OPA',
    MODULE_ATTESTATION: 'Module Attestation',
};

// checkTypeLabel is KIND_LABELS with a fallback: an unrecognized checkType (a kind this frontend
// hasn't been taught the label for yet) falls back to the raw value rather than to 'OPA' — showing
// an unfamiliar constant is honest about not knowing what it is, where defaulting to the original
// and most common kind would misidentify a future check type as OPA.
export function checkTypeLabel(checkType: string): string {
    return KIND_LABELS[checkType] ?? checkType;
}

// The stage label, keyed by the GraphQL enum value, so every view that shows a policy's stage agrees
// on its wording instead of showing the raw constant.
export const STAGE_LABELS: Record<string, string> = {
    PRE_PLAN: 'Pre Plan',
    POST_PLAN: 'Post Plan',
    PRE_APPLY: 'Pre Apply',
    POST_APPLY: 'Post Apply',
};

// The enforcement level label, keyed by the GraphQL enum value. Also used for the speculative level,
// whose values are a subset.
export const ENFORCEMENT_LEVEL_LABELS: Record<string, string> = {
    ADVISORY: 'Advisory',
    SOFT_MANDATORY: 'Soft Mandatory',
    HARD_MANDATORY: 'Hard Mandatory',
};
