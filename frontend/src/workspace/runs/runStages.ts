// Stage statuses that count as done: 'completed' is the task stage aggregate success status; the
// rest are plan/apply statuses.
export const SUCCESS_STATUSES = new Set([
    'finished', 'applied', 'planned_and_finished', 'completed',
]);

export type RunStageKey = 'PRE_PLAN' | 'PLAN' | 'POST_PLAN' | 'PRE_APPLY' | 'APPLY' | 'POST_APPLY';

export interface RunStage {
    key: RunStageKey
    name: string
    status: string
}

interface RunStagesInput {
    plan: { readonly status: string }
    apply: { readonly status: string } | null | undefined
    taskStages: ReadonlyArray<{ readonly stageName: string, readonly status: string }>
}

// buildRunStages returns a run's stages in execution order, so the stage bar and any stage tally
// agree on what the stages are.
export function buildRunStages({ plan, apply, taskStages }: RunStagesInput): RunStage[] {
    const task = (key: RunStageKey, name: string): RunStage[] => {
        const stage = taskStages.find(t => t.stageName === key);
        return stage ? [{ key, name, status: stage.status }] : [];
    };
    return [
        // A pre-plan policy stage runs BEFORE the plan, so it leads when present.
        ...task('PRE_PLAN', 'Pre-Plan'),
        { key: 'PLAN', name: 'Plan', status: plan.status },
        ...task('POST_PLAN', 'Post-Plan'),
        ...task('PRE_APPLY', 'Pre-Apply'),
        // A speculative run has no apply at all, which is what makes the stage conditional rather
        // than the apply's status.
        ...(apply ? [{ key: 'APPLY' as const, name: 'Apply', status: apply.status }] : []),
        ...task('POST_APPLY', 'Post-Apply'),
    ];
}

// countCompletedStages returns how many of the stages have finished successfully.
export function countCompletedStages(stages: RunStage[]): number {
    return stages.filter(s => SUCCESS_STATUSES.has(s.status.toLowerCase())).length;
}
