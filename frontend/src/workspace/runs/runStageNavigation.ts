// Stage navigation for the run details page: which stage should be shown when a run is opened
// without an explicit stage in the URL.
//
// Node statuses arrive from GraphQL with inconsistent casing — plan and apply enums are serialized
// lowercase, while task stage and policy check enums are uppercase — so every comparison here
// normalizes first.

// A stage is in progress once it has been admitted or started and has not reached a terminal state.
// 'pending' is the pre-admission state shared by the plan, the apply and workspace-gated task stages
// ("ready, but waiting for the workspace slot"), so it counts as in progress. 'awaiting_override' is a
// task stage blocked on a human decision, which is still the stage the run sits at.
const IN_PROGRESS_STATUSES = new Set(['pending', 'queued', 'running', 'awaiting_override']);

// A stage counts as completed once it has actually run and reached a terminal state. 'created' and
// 'skipped' are excluded: they mean the stage never started.
const COMPLETED_STATUSES = new Set(['finished', 'completed', 'errored', 'canceled']);

type NodeStatus = string | null | undefined;

interface TaskStageNode {
    readonly stageName: string;
    readonly status: NodeStatus;
}

export interface RunStageNodes {
    readonly plan: { readonly status: NodeStatus };
    readonly apply?: { readonly status: NodeStatus } | null;
    readonly taskStages: readonly TaskStageNode[];
}

interface PipelineStage {
    readonly path: string;
    readonly status: NodeStatus;
}

const normalize = (status: NodeStatus) => (status ?? '').toLowerCase();

// Each task stage has its own route, named after the stage itself, so the path segment is the stage
// name lowercased ('PRE_PLAN' -> 'pre_plan'). Kept here so the routes in RunDetails, the links that
// point at them and the active-stage checks all derive the segment from one place.
export const taskStagePath = (stageName: string) => stageName.toLowerCase();

// The stages that exist on this run, in pipeline order, paired with the route that renders them.
// Only the pre-plan and post-plan task stages are included because those are the only task stages
// with a route today.
function buildPipeline(run: RunStageNodes): PipelineStage[] {
    const prePlanStage = run.taskStages.find(s => s.stageName === 'PRE_PLAN');
    const postPlanStage = run.taskStages.find(s => s.stageName === 'POST_PLAN');

    const pipeline: PipelineStage[] = [];
    if (prePlanStage) {
        pipeline.push({ path: taskStagePath(prePlanStage.stageName), status: prePlanStage.status });
    }
    pipeline.push({ path: 'plan', status: run.plan.status });
    if (postPlanStage) {
        pipeline.push({ path: taskStagePath(postPlanStage.stageName), status: postPlanStage.status });
    }
    if (run.apply) {
        pipeline.push({ path: 'apply', status: run.apply.status });
    }
    return pipeline;
}

/**
 * Returns the route path of the stage the run details page should show by default: the stage that is
 * currently in progress, or — when nothing is in progress — the last stage that completed. Falls back
 * to the plan stage when no stage has started yet (or the run is unavailable).
 *
 * Stages that were never reached ('created') or were bypassed ('skipped') are never selected, which is
 * what keeps a run whose plan is still running from landing on its post-plan policy stage.
 */
export function resolveCurrentStagePath(run: RunStageNodes | null | undefined): string {
    if (!run) {
        return 'plan';
    }

    const pipeline = buildPipeline(run);

    // Only one stage runs at a time, but scanning from the end keeps the furthest-along stage as the
    // answer if statuses are ever momentarily out of step (e.g. a refetch landing mid-transition).
    for (let i = pipeline.length - 1; i >= 0; i--) {
        if (IN_PROGRESS_STATUSES.has(normalize(pipeline[i].status))) {
            return pipeline[i].path;
        }
    }

    for (let i = pipeline.length - 1; i >= 0; i--) {
        if (COMPLETED_STATUSES.has(normalize(pipeline[i].status))) {
            return pipeline[i].path;
        }
    }

    return 'plan';
}
