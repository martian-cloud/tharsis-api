import CopyIcon from '@mui/icons-material/ContentCopy';
import HelpOutlineIcon from '@mui/icons-material/HelpOutline';
import { Chip, List, Stack, Tooltip, Typography } from '@mui/material';
import Box from '@mui/material/Box';
import IconButton from '@mui/material/IconButton';
import ListItemButton from '@mui/material/ListItemButton';
import ListItemIcon from '@mui/material/ListItemIcon';
import ListItemText from '@mui/material/ListItemText';
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
    // run is still planning and its apply phase has not started.
    const canEditAutoApply = !!data.apply
        && data.apply.status === 'created'
        && ['pending', 'queuing', 'plan_queued', 'planning'].includes(data.status);

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

    const PlanStatusIcon = RunStageStatusTypes[data.plan.status].icon;
    const ApplyStatusIcon = data.apply ? RunStageStatusTypes[data.apply.status].icon : null;

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
                    <RunStatusChip status={data.status} />
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
                {(data as any)[stage].currentJob?.runnerPath && <Box marginBottom={3}>
                    <Typography sx={{ marginBottom: 1 }}>Runner</Typography>
                    <Tooltip title={(data as any)[stage].currentJob.runnerPath} >
                        <Chip size="small" label={(data as any)[stage].currentJob.runnerPath} />
                    </Tooltip>
                </Box>}
                <Box marginBottom={3}>
                    <Typography sx={{ marginBottom: 1 }}>Stages</Typography>
                    <List>
                        <ListItemButton selected={stage === 'plan'} component={LinkRouter} replace to={`/groups/${data.workspace.fullPath}/-/runs/${data.id}/plan`}>
                            <ListItemIcon>
                                <PlanStatusIcon />
                            </ListItemIcon>
                            <ListItemText primary="Plan" />
                        </ListItemButton>
                        {data.apply && <ListItemButton selected={stage === 'apply'} component={LinkRouter} replace to={`/groups/${data.workspace.fullPath}/-/runs/${data.id}/apply`}>
                            <ListItemIcon>
                                <ApplyStatusIcon />
                            </ListItemIcon>
                            <ListItemText primary="Apply" />
                        </ListItemButton>}
                    </List>
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
