import CopyIcon from '@mui/icons-material/ContentCopy';
import HelpOutlineIcon from '@mui/icons-material/HelpOutline';
import { Chip, Stack, Tooltip, Typography } from '@mui/material';
import Box from '@mui/material/Box';
import IconButton from '@mui/material/IconButton';
import graphql from 'babel-plugin-relay/macro';
import React, { useContext, useMemo, useState } from 'react';
import { useFragment, useMutation } from 'react-relay/hooks';
import { Link as LinkRouter } from 'react-router-dom';
import { ApiConfigContext } from '../../ApiConfigContext';
import ConfirmationDialog from '../../common/ConfirmationDialog';
import Drawer from '../../common/Drawer';
import { MutationError } from '../../common/error';
import Gravatar from '../../common/Gravatar';
import Timestamp from '../../common/Timestamp';
import TRNButton from '../../common/TRNButton';
import Link from '../../routes/Link';
import { RunDetailsSidebarFragment_details$key } from './__generated__/RunDetailsSidebarFragment_details.graphql';
import { RunDetailsSidebarSetRunAutoApplyMutation } from './__generated__/RunDetailsSidebarSetRunAutoApplyMutation.graphql';
import RunStageStatusTypes from './RunStageStatusTypes';
import RunStatusChip from './RunStatusChip';
import { taskStagePath } from './runStageNavigation';

interface Props {
    fragmentRef: RunDetailsSidebarFragment_details$key
    stage: string
    open: boolean
    temporary: boolean
    onClose: () => void
    onError: (error: MutationError) => void
}

export const SidebarWidth = 300;

function RunDetailsSidebar(props: Props) {
    const { stage, open, temporary, onClose, onError } = props;
    const apiConfig = useContext(ApiConfigContext);

    const data = useFragment<RunDetailsSidebarFragment_details$key>(
        graphql`
    fragment RunDetailsSidebarFragment_details on Run
    {
        id
        status
        createdBy
        isDestroy
        assessment
        autoApply
        hasAdvisoryFailures
        moduleSource
        moduleVersion
        workspace {
          fullPath
        }
        metadata {
          createdAt
          trn
        }
        configurationVersion {
          id
        }
        plan {
          status
          metadata {
            createdAt
          }
          currentJob {
            runnerPath
            cancelRequested
          }
        }
        taskStages {
          stageName
          status
          policyChecks {
            status
            stageName
          }
        }
        apply {
          status
          metadata {
            createdAt
          }
          currentJob {
            runnerPath
            cancelRequested
          }
        }
    }
  `, props.fragmentRef)

    const [commitSetAutoApply, setAutoApplyInFlight] = useMutation<RunDetailsSidebarSetRunAutoApplyMutation>(graphql`
        mutation RunDetailsSidebarSetRunAutoApplyMutation($input: SetRunAutoApplyInput!) {
            setRunAutoApply(input: $input) {
                run {
                    ...RunDetailsSidebarFragment_details
                }
                problems {
                    message
                    field
                    type
                }
            }
        }
    `);

    const [editingAutoApply, setEditingAutoApply] = useState(false);

    // Auto-apply only takes effect when the plan finishes, so it can only be changed while the
    // run is still planning and its apply phase has not started. The pre-plan policy stage runs
    // before the plan, so it counts too. Mirrors the server-side guard in SetRunAutoApply.
    const canEditAutoApply = !!data.apply
        && data.apply.status === 'created'
        && [
            'pending',
            'pre_plan_queuing', 'pre_plan_running', 'pre_plan_awaiting_decision', 'pre_plan_completed',
            'plan_queuing', 'plan_queued', 'planning', 'post_plan_running', 'post_plan_awaiting_decision',
        ].includes(data.status);

    const confirmAutoApply = () => {
        commitSetAutoApply({
            variables: {
                input: {
                    runId: data.id,
                    autoApply: !data.autoApply,
                },
            },
            onCompleted: response => {
                if (response.setRunAutoApply.problems.length) {
                    onError({
                        severity: 'warning',
                        message: response.setRunAutoApply.problems.map(problem => problem.message).join('; ')
                    });
                    return;
                }
                setEditingAutoApply(false);
            },
            onError: error => {
                onError({
                    severity: 'error',
                    message: `Unexpected Error Occurred: ${error.message}`
                });
            }
        });
    };

    // If module source references a module in the tharsis registry than strip the host
    const moduleSource = useMemo(
        () => (data.moduleSource && data.moduleSource?.startsWith(apiConfig.serviceDiscoveryHost)) ? data.moduleSource.substring(apiConfig.serviceDiscoveryHost.length + 1) : data.moduleSource,
        [data.moduleSource, apiConfig.serviceDiscoveryHost]
    );

    const isTharsisModule = useMemo(() => moduleSource && moduleSource.length != data.moduleSource?.length, [moduleSource, data.moduleSource]);

    // Stage-nav entries reflect the aggregate task stage status (the pre-plan stage runs before the
    // plan, the post-plan stage after it). Each stage is its own route, so `stage` alone says which
    // entry is active.
    const prePlanStage = data.taskStages.find(s => s.stageName === 'PRE_PLAN');
    const postPlanStage = data.taskStages.find(s => s.stageName === 'POST_PLAN');
    const preApplyStage = data.taskStages.find(s => s.stageName === 'PRE_APPLY');
    const postApplyStage = data.taskStages.find(s => s.stageName === 'POST_APPLY');
    const PlanStatusIcon = RunStageStatusTypes[data.plan.status.toLowerCase()].icon;
    const PrePlanStatusIcon = prePlanStage ? RunStageStatusTypes[prePlanStage.status.toLowerCase()].icon : null;
    const PolicyStatusIcon = postPlanStage ? RunStageStatusTypes[postPlanStage.status.toLowerCase()].icon : null;
    const PreApplyStatusIcon = preApplyStage ? RunStageStatusTypes[preApplyStage.status.toLowerCase()].icon : null;
    const ApplyStatusIcon = data.apply ? RunStageStatusTypes[data.apply.status.toLowerCase()].icon : null;
    const PostApplyStatusIcon = postApplyStage ? RunStageStatusTypes[postApplyStage.status.toLowerCase()].icon : null;

    return (
        <Drawer
            width={SidebarWidth}
            temporary={temporary}
            variant={temporary ? 'temporary' : 'permanent'}
            open={open}
            hideBackdrop={false}
            anchor='right'
            onClose={onClose}
        >
            <Box padding={2}>
                <Box marginBottom={2} display="flex" alignItems="center" justifyContent="space-between">
                    <Typography variant="h6">Run Details</Typography>
                    <TRNButton size="small" trn={data.metadata.trn} />
                </Box>
                <Box marginBottom={3}>
                    <Typography sx={{ marginBottom: 1 }}>Status</Typography>
                    <RunStatusChip status={data.status} hasAdvisoryFailures={data.hasAdvisoryFailures} />
                </Box>
                <Box marginBottom={3}>
                    <Typography sx={{ marginBottom: 1 }}>Type</Typography>
                    {!data.isDestroy && data.apply && <Chip size="small" label="Apply" />}
                    {data.isDestroy && <Chip size="small" label="Destroy" sx={{ color: 'runStatus.destroy' }} />}
                    {!data.apply && <Chip size="small" label={data.assessment ? "Assessment" : "Speculative"} />}
                </Box>
                {data.apply && <Box marginBottom={3}>
                    <Box display="flex" alignItems="center" sx={{ marginBottom: 1 }}>
                        <Typography>Auto Apply</Typography>
                        <Tooltip title="When auto apply is enabled, the apply stage starts automatically after the plan completes with changes. When disabled, the run waits at the planned state for a user to manually start the apply.">
                            <HelpOutlineIcon sx={{
                                width: 16,
                                height: 16,
                                marginLeft: '6px',
                                opacity: '40%',
                                transition: 'ease',
                                transitionDuration: '300ms',
                                ":hover": {
                                    opacity: '100%'
                                }
                            }} />
                        </Tooltip>
                    </Box>
                    {canEditAutoApply
                        ? <Tooltip title="Click to change">
                            <Chip
                                size="small"
                                label={data.autoApply ? 'Enabled' : 'Disabled'}
                                onClick={() => setEditingAutoApply(true)}
                            />
                        </Tooltip>
                        : <Chip size="small" label={data.autoApply ? 'Enabled' : 'Disabled'} />}
                </Box>}
                <Box marginBottom={3}>
                    <Typography sx={{ marginBottom: 1 }}>Created</Typography>
                    <Box display="flex" alignItems="center">
                        <Timestamp variant="subtitle1" sx={{ marginRight: 1 }} timestamp={data.metadata.createdAt} />
                        <Tooltip title={data.createdBy}>
                            <Box>
                                <Gravatar width={20} height={20} email={data.createdBy} />
                            </Box>
                        </Tooltip>
                    </Box>
                </Box>
                {data.configurationVersion && <Box marginBottom={3}>
                    <Typography sx={{ marginBottom: 1 }}>Configuration Version</Typography>
                    <Stack direction="row" spacing={1} alignItems="center">
                        <Tooltip title="view files">
                            <Link
                                color="secondary"
                                underline="none"
                                sx={{ wordBreak: 'break-all' }}
                                to={`/groups/${data.workspace.fullPath}/-/configuration_versions/${data.configurationVersion.id}`}
                            >
                                {data.configurationVersion.id.substring(0, 8)}...
                            </Link>
                        </Tooltip>
                    </Stack>
                </Box>}
                {moduleSource && <Box marginBottom={3}>
                    <Typography sx={{ marginBottom: 1 }}>Module Source</Typography>
                    {!isTharsisModule && <React.Fragment>
                        <Stack direction="row" spacing={1} alignItems="center">
                            <Tooltip title={data.moduleSource}>
                                <Typography sx={{ wordBreak: 'break-all' }}>
                                    {`${moduleSource.substring(0, 24)}...`}
                                </Typography>
                            </Tooltip>
                            <IconButton sx={{ padding: '4px' }} onClick={() => navigator.clipboard.writeText(data.moduleSource ?? '')}>
                                <CopyIcon sx={{ width: 16, height: 16 }} />
                            </IconButton>
                        </Stack>
                    </React.Fragment>}
                    {isTharsisModule && <React.Fragment>
                        <Tooltip title={moduleSource}>
                            <Typography color="secondary" component="p" noWrap>
                                <Link color="inherit" noWrap underline="hover" to={`/module-registry/${moduleSource}/${data.moduleVersion}`}>
                                    {moduleSource}
                                </Link>
                            </Typography>
                        </Tooltip>
                    </React.Fragment>}
                </Box>}
                {data.moduleVersion && <Box marginBottom={3}>
                    <Typography sx={{ marginBottom: 1 }}>Module Version</Typography>
                    <Chip size="small" label={data.moduleVersion} />
                </Box>}
                {(data as any)[stage]?.currentJob?.runnerPath && <Box marginBottom={3}>
                    <Typography sx={{ marginBottom: 1 }}>Runner</Typography>
                    <Tooltip title={(data as any)[stage].currentJob.runnerPath} >
                        <Chip size="small" label={(data as any)[stage].currentJob.runnerPath} />
                    </Tooltip>
                </Box>}
                <Box marginBottom={3}>
                    <Typography sx={{ marginBottom: 1 }}>Stages</Typography>
                    <Box sx={{ py: 0.5 }}>
                        {/* Pre-Plan Policy (runs before the plan) */}
                        {prePlanStage && PrePlanStatusIcon && <>
                            <Box
                                component={LinkRouter}
                                to={`/groups/${data.workspace.fullPath}/-/runs/${data.id}/${taskStagePath(prePlanStage.stageName)}`}
                                replace
                                sx={{ display: 'flex', alignItems: 'center', gap: 1.25, px: 1, py: 0.75, borderRadius: 1, textDecoration: 'none', bgcolor: stage === taskStagePath(prePlanStage.stageName) ? 'action.selected' : 'transparent', '&:hover': { bgcolor: 'action.hover' } }}
                            >
                                <PrePlanStatusIcon sx={{ flexShrink: 0 }} />
                                <Typography variant="body2" color="text.primary">Pre-Plan</Typography>
                            </Box>
                            <Box sx={{ ml: '19px', width: 2, height: 10, bgcolor: 'divider' }} />
                        </>}

                        {/* Plan */}
                        <Box
                            component={LinkRouter}
                            to={`/groups/${data.workspace.fullPath}/-/runs/${data.id}/plan`}
                            replace
                            sx={{ display: 'flex', alignItems: 'center', gap: 1.25, px: 1, py: 0.75, borderRadius: 1, textDecoration: 'none', bgcolor: stage === 'plan' ? 'action.selected' : 'transparent', '&:hover': { bgcolor: 'action.hover' } }}
                        >
                            <PlanStatusIcon sx={{ flexShrink: 0 }} />
                            <Typography variant="body2" color="text.primary">Plan</Typography>
                        </Box>

                        {/* Connector: Plan → Post-Plan (or Plan → Pre-Apply/Apply if no post-plan policy) */}
                        {(postPlanStage || preApplyStage || data.apply) && (
                            <Box sx={{ ml: '19px', width: 2, height: 10, bgcolor: 'divider' }} />
                        )}

                        {/* Policy (post-plan) */}
                        {postPlanStage && PolicyStatusIcon && <>
                            <Box
                                component={LinkRouter}
                                to={`/groups/${data.workspace.fullPath}/-/runs/${data.id}/${taskStagePath(postPlanStage.stageName)}`}
                                replace
                                sx={{ display: 'flex', alignItems: 'center', gap: 1.25, px: 1, py: 0.75, borderRadius: 1, textDecoration: 'none', bgcolor: stage === taskStagePath(postPlanStage.stageName) ? 'action.selected' : 'transparent', '&:hover': { bgcolor: 'action.hover' } }}
                            >
                                <PolicyStatusIcon sx={{ flexShrink: 0 }} />
                                <Typography variant="body2" color="text.primary">Post-Plan</Typography>
                            </Box>
                            {(preApplyStage || data.apply) && (
                                <Box sx={{ ml: '19px', width: 2, height: 10, bgcolor: 'divider' }} />
                            )}
                        </>}

                        {/* Policy (pre-apply) */}
                        {preApplyStage && PreApplyStatusIcon && <>
                            <Box
                                component={LinkRouter}
                                to={`/groups/${data.workspace.fullPath}/-/runs/${data.id}/${taskStagePath(preApplyStage.stageName)}`}
                                replace
                                sx={{ display: 'flex', alignItems: 'center', gap: 1.25, px: 1, py: 0.75, borderRadius: 1, textDecoration: 'none', bgcolor: stage === taskStagePath(preApplyStage.stageName) ? 'action.selected' : 'transparent', '&:hover': { bgcolor: 'action.hover' } }}
                            >
                                <PreApplyStatusIcon sx={{ flexShrink: 0 }} />
                                <Typography variant="body2" color="text.primary">Pre-Apply</Typography>
                            </Box>
                            {data.apply && (
                                <Box sx={{ ml: '19px', width: 2, height: 10, bgcolor: 'divider' }} />
                            )}
                        </>}

                        {/* Apply */}
                        {data.apply && ApplyStatusIcon && <>
                            <Box
                                component={LinkRouter}
                                to={`/groups/${data.workspace.fullPath}/-/runs/${data.id}/apply`}
                                replace
                                sx={{ display: 'flex', alignItems: 'center', gap: 1.25, px: 1, py: 0.75, borderRadius: 1, textDecoration: 'none', bgcolor: stage === 'apply' ? 'action.selected' : 'transparent', '&:hover': { bgcolor: 'action.hover' } }}
                            >
                                <ApplyStatusIcon sx={{ flexShrink: 0 }} />
                                <Typography variant="body2" color="text.primary">Apply</Typography>
                            </Box>
                            {postApplyStage && (
                                <Box sx={{ ml: '19px', width: 2, height: 10, bgcolor: 'divider' }} />
                            )}
                        </>}

                        {/* Policy (post-apply) */}
                        {postApplyStage && PostApplyStatusIcon && (
                            <Box
                                component={LinkRouter}
                                to={`/groups/${data.workspace.fullPath}/-/runs/${data.id}/${taskStagePath(postApplyStage.stageName)}`}
                                replace
                                sx={{ display: 'flex', alignItems: 'center', gap: 1.25, px: 1, py: 0.75, borderRadius: 1, textDecoration: 'none', bgcolor: stage === taskStagePath(postApplyStage.stageName) ? 'action.selected' : 'transparent', '&:hover': { bgcolor: 'action.hover' } }}
                            >
                                <PostApplyStatusIcon sx={{ flexShrink: 0 }} />
                                <Typography variant="body2" color="text.primary">Post-Apply</Typography>
                            </Box>
                        )}
                    </Box>
                </Box>
            </Box>
            {editingAutoApply && <ConfirmationDialog
                title="Edit Auto Apply"
                confirmColor="primary"
                confirmLabel={data.autoApply ? 'Disable' : 'Enable'}
                confirmInProgress={setAutoApplyInFlight}
                onConfirm={confirmAutoApply}
                onClose={() => setEditingAutoApply(false)}
            >
                <Typography>
                    {data.autoApply
                        ? 'Disabling auto apply means this run will wait at the planned state for a user to manually start the apply.'
                        : 'Enabling auto apply means this run\'s apply phase will start automatically once the plan completes with changes.'}
                </Typography>
            </ConfirmationDialog>}
        </Drawer>
    );
}

export default RunDetailsSidebar;
