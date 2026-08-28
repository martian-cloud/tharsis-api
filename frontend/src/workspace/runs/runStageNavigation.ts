// Stage navigation for the run details page: which stage should be shown when a run is opened
// without an explicit stage in the URL.
//
// The run status is the answer for every run that is still going: the status names the phase the run
// is in ('post_plan_awaiting_decision', 'apply_queuing', ...), so the stage is a lookup rather than a
// scan over the run's nodes. Node statuses are only consulted for a run that stopped short, where the
// status ('errored', 'canceled', 'discarded') says the run ended but not where.

// Each task stage has its own route, named after the stage itself, so the path segment is the stage
// name lowercased ('PRE_PLAN' -> 'pre_plan'). Kept here so the routes in RunDetails, the links that
// point at them and the active-stage checks all derive the segment from one place.
export const taskStagePath = (stageName: string) => stageName.toLowerCase();

const PLAN = 'plan';

// The stage each run status sits at. Every status the API can return is listed, so a missing entry
// means the status is one of the three that does not name a stage (see resolveCurrentStagePath).
//
// 'planned' maps to the plan rather than to the post-plan stage that just passed: a planned run is
// waiting on the apply decision, and the plan is where the diff and the apply action are. A post-plan
// stage that did not pass leaves the run in 'post_plan_awaiting_decision' or 'errored' instead, so
// this never hides a stage that needs attention.
const STAGE_BY_RUN_STATUS: Record<string, string> = {
    // Nothing has started yet; the plan is the run's headline stage.
    pending: PLAN,

    pre_plan_queuing: 'pre_plan',
    pre_plan_running: 'pre_plan',
    pre_plan_awaiting_decision: 'pre_plan',
    pre_plan_completed: 'pre_plan',

    plan_queuing: PLAN,
    plan_queued: PLAN,
    planning: PLAN,

    post_plan_running: 'post_plan',
    post_plan_awaiting_decision: 'post_plan',
    post_plan_completed: 'post_plan',

    planned: PLAN,

    pre_apply_queuing: 'pre_apply',
    pre_apply_running: 'pre_apply',
    pre_apply_awaiting_decision: 'pre_apply',
    pre_apply_completed: 'pre_apply',

    apply_queuing: 'apply',
    apply_queued: 'apply',
    applying: 'apply',

    post_apply_running: 'post_apply',
    post_apply_completed: 'post_apply',

    // Terminal, and each still names where the run ended: an applied run ended at its apply, and a
    // run that finished without applying (speculative, or nothing to change) ended at its plan.
    applied: 'apply',
    planned_and_finished: PLAN,
};

// The statuses of a node that failed or was stopped. A run reaching 'errored', 'canceled' or
// 'discarded' does not say which stage it stopped at, and these are what identify it: the run settles
// the stage it was at to errored or canceled and skips the nodes that never started, so the stopped
// node is the run's stopping point. Plan and apply serialize their statuses lowercase while task
// stages serialize theirs uppercase, so comparisons normalize first.
const STOPPED_NODE_STATUSES = new Set(['errored', 'canceled']);

type NodeStatus = string | null | undefined;

interface TaskStageNode {
    readonly stageName: string;
    readonly status: NodeStatus;
}

export interface RunStageNodes {
    readonly status: string;
    readonly plan: { readonly status: NodeStatus };
    readonly apply?: { readonly status: NodeStatus } | null;
    readonly taskStages: readonly TaskStageNode[];
}

const normalize = (status: NodeStatus) => (status ?? '').toLowerCase();

/**
 * Returns the route path of the stage the run details page should show by default: the stage the run
 * status names, or — for a run that errored, was canceled or was discarded, where the status names no
 * stage — the furthest stage that failed or was stopped. Falls back to the plan stage when neither
 * applies (a discard from 'planned' stops no stage, and the run is unavailable on first render).
 */
export function resolveCurrentStagePath(run: RunStageNodes | null | undefined): string {
    if (!run) {
        return PLAN;
    }

    const stage = STAGE_BY_RUN_STATUS[normalize(run.status)];
    if (stage) {
        return stage;
    }

    // 'errored', 'canceled' or 'discarded'. The stage the run stopped at carries the matching node
    // status, while the stages after it were never reached ('created') or were bypassed ('skipped').
    // Scanning from the end of the pipeline picks the furthest one, so a stage stopped after an
    // earlier failure was retried and cleared still wins.
    //
    // A stage the run does not have contributes no status, which is what keeps a speculative run (no
    // apply, no apply-phase stages) from matching on one.
    const taskStage = (stageName: string) => [
        taskStagePath(stageName),
        run.taskStages.find(s => s.stageName === stageName)?.status,
    ] as const;

    // Pipeline order.
    const pipeline: readonly (readonly [string, NodeStatus])[] = [
        taskStage('PRE_PLAN'),
        [PLAN, run.plan.status],
        taskStage('POST_PLAN'),
        taskStage('PRE_APPLY'),
        ['apply', run.apply?.status],
        taskStage('POST_APPLY'),
    ];

    for (let i = pipeline.length - 1; i >= 0; i--) {
        const [path, status] = pipeline[i];
        if (STOPPED_NODE_STATUSES.has(normalize(status))) {
            return path;
        }
    }

    return PLAN;
}
