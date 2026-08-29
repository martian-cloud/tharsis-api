import WarningIcon from '@mui/icons-material/Error';
import { alpha, Box, Theme, Tooltip, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import React, { useMemo } from 'react';
import { useFragment } from 'react-relay/hooks';
import { Link as RouterLink } from 'react-router-dom';
import RunStageStatusTypes from './RunStageStatusTypes';
import { STATUS_MAP } from './RunStatusChip';
import { RunStageIconsFragment_run$key } from './__generated__/RunStageIconsFragment_run.graphql';
import { taskStagePath } from './runStageNavigation';

const ADVISORY_FAILURES_TOOLTIP = "One or more advisory policies failed.";

interface Props {
    fragmentRef: RunStageIconsFragment_run$key
}

const IDLE_STATUSES = new Set([
    'created', 'pending', 'queued', 'plan_queued', 'apply_queued',
    'plan_queuing', 'apply_queuing', 'pre_plan_queuing', 'pre_apply_queuing',
]);

const SUCCESS_STATUSES = new Set([
    // 'completed' is the task stage aggregate success status; the rest are plan/apply statuses.
    'finished', 'applied', 'planned_and_finished', 'completed',
]);

const RUNNING_STATUSES = new Set([
    'running', 'planning', 'applying', 'planned',
]);

function RunStageIcons({ fragmentRef }: Props) {
    const data = useFragment<RunStageIconsFragment_run$key>(graphql`
        fragment RunStageIconsFragment_run on Run {
            id
            status
            # Qualifies the status readout below the bar with a warning glyph, exactly as RunStatusChip
            # does. An advisory failure never blocks a run, so no stage segment can express one: the
            # stage it belongs to still completes, and the run still reaches a success status.
            hasAdvisoryFailures
            plan {
                status
            }
            taskStages {
                stageName
                status
            }
            apply {
                status
            }
            # Only to build the segment links. The bar already owns the rest of the stage route shape,
            # so it owns the prefix too rather than taking it as a prop.
            workspace {
                fullPath
            }
        }
    `, fragmentRef);

    const runPath = `/groups/${data.workspace.fullPath}/-/runs/${data.id}`;
    const runStatusLabel = STATUS_MAP[data.status]?.label?.toLowerCase() || 'unknown';
    const stages = useMemo(() => {
        const s = [];
        // A pre-plan policy stage runs BEFORE the plan, so it leads the bar when present.
        const prePlan = data.taskStages.find(t => t.stageName === 'PRE_PLAN');
        if (prePlan) s.push({ name: 'Pre-Plan', status: prePlan.status, path: `${runPath}/${taskStagePath(prePlan.stageName)}` });
        s.push({ name: 'Plan', status: data.plan.status, path: `${runPath}/plan` });
        const postPlan = data.taskStages.find(t => t.stageName === 'POST_PLAN');
        if (postPlan) s.push({ name: 'Post-Plan', status: postPlan.status, path: `${runPath}/${taskStagePath(postPlan.stageName)}` });
        const preApply = data.taskStages.find(t => t.stageName === 'PRE_APPLY');
        if (preApply) s.push({ name: 'Pre-Apply', status: preApply.status, path: `${runPath}/${taskStagePath(preApply.stageName)}` });
        // A speculative run has no apply at all, which is what makes the segment conditional rather
        // than the apply's status.
        if (data.apply) s.push({ name: 'Apply', status: data.apply.status, path: `${runPath}/apply` });
        const postApply = data.taskStages.find(t => t.stageName === 'POST_APPLY');
        if (postApply) s.push({ name: 'Post-Apply', status: postApply.status, path: `${runPath}/${taskStagePath(postApply.stageName)}` });
        return s;
    }, [data, runPath]);

    const activeIndex = useMemo(() => {
        let ai = 0;
        stages.forEach((s, i) => { if (!IDLE_STATUSES.has(s.status.toLowerCase())) ai = i; });
        return ai;
    }, [stages]);

    const activeStage = stages[activeIndex];
    const activeInfo = RunStageStatusTypes[activeStage.status.toLowerCase()] ?? RunStageStatusTypes.created;
    const successCount = stages.filter(s => SUCCESS_STATUSES.has(s.status.toLowerCase())).length;

    // The dot, the status word, and the advisory glyph read as one unit, so they are grouped and
    // hovered as one — matching RunStatusChip, where the tooltip covers the whole chip rather than
    // the glyph alone. The stage tally to their right is deliberately left outside it.
    const statusReadout = (
        <Box display="flex" alignItems="center" sx={{ gap: '8px', minWidth: 0 }}>
            <Box sx={{ width: 7, height: 7, borderRadius: '50%', bgcolor: activeInfo.color, flexShrink: 0 }} />
            <Typography variant="caption" color="text.secondary">
                {runStatusLabel}
            </Typography>
            {/* Amber regardless of the run's status colour: the glyph is the advisory signal, the
                label is still the status. */}
            {data.hasAdvisoryFailures && (
                <WarningIcon sx={{ width: 16, height: 16, color: 'warning.main', flexShrink: 0 }} />
            )}
        </Box>
    );

    // activeInfo.color is a theme palette path (e.g. 'runStatus.running'); resolve it to a
    // concrete color so we can derive a translucent fill with alpha().
    const resolveColor = (theme: Theme, path: string): string =>
        path.split('.').reduce<any>((o, k) => (o == null ? o : o[k]), theme.palette) as string;

    return (
        <Box>
            <Box display="flex" sx={{ gap: '3px', mb: 0.5 }}>
                {stages.map(stage => {
                    const info = RunStageStatusTypes[stage.status.toLowerCase()] ?? RunStageStatusTypes.created;
                    const isRunning = RUNNING_STATUSES.has(stage.status.toLowerCase());
                    const opacity = IDLE_STATUSES.has(stage.status.toLowerCase()) ? 0.25
                        // Skipped/canceled segments are dimmed to recede, but their colours are now
                        // themselves recessive (warm inert / red), so they need more opacity than
                        // before to stay visible against the paper background.
                        : (stage.status.toLowerCase() === 'skipped' || stage.status.toLowerCase() === 'canceled') ? 0.55
                            : 0.6;
                    return (
                        <Tooltip key={stage.name} title={`${stage.name}: ${info.label}`}>
                            <Box
                                component={RouterLink}
                                to={stage.path}
                                sx={(theme) => {
                                    const c = resolveColor(theme, info.color);
                                    return {
                                        flex: 1,
                                        display: 'block',
                                        height: '8px',
                                        borderRadius: '4px',
                                        border: `1px solid ${c}`,
                                        bgcolor: alpha(c, 0.25),
                                        opacity,
                                        textDecoration: 'none',
                                        ...(isRunning && {
                                            '@keyframes stageBarPulse': {
                                                '0%, 100%': { opacity: 1 },
                                                '50%': { opacity: 0.45 },
                                            },
                                            animation: 'stageBarPulse 4.0s ease-in-out infinite',
                                        }),
                                    };
                                }}
                            />
                        </Tooltip>
                    );
                })}
            </Box>
            <Box display="flex" alignItems="center" sx={{ gap: '8px' }}>
                {data.hasAdvisoryFailures
                    ? <Tooltip title={ADVISORY_FAILURES_TOOLTIP}>{statusReadout}</Tooltip>
                    : statusReadout}
                <Typography variant="caption" sx={{ ml: 'auto', color: 'text.disabled', fontFamily: 'monospace' }}>
                    {successCount}/{stages.length}
                </Typography>
            </Box>
        </Box>
    );
}

export default RunStageIcons;
