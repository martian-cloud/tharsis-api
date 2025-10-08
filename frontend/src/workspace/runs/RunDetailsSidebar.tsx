import CopyIcon from '@mui/icons-material/ContentCopy';
import { LoadingButton } from '@mui/lab';
import { Chip, Link, List, Stack, Tooltip, Typography } from '@mui/material';
import Box from '@mui/material/Box';
import { red } from '@mui/material/colors';
import IconButton from '@mui/material/IconButton';
import ListItem from '@mui/material/ListItem';
import ListItemIcon from '@mui/material/ListItemIcon';
import ListItemText from '@mui/material/ListItemText';
import graphql from 'babel-plugin-relay/macro';
import { useContext } from 'react';
import { useFragment, useMutation } from 'react-relay/hooks';
import { Link as LinkRouter } from 'react-router-dom';
import AuthenticationService from '../../auth/AuthenticationService';
import AuthServiceContext from '../../auth/AuthServiceContext';
import cfg from '../../common/config';
import Drawer from '../../common/Drawer';
import { MutationError } from '../../common/error';
import downloadFile from '../../common/filedownload';
import Gravatar from '../../common/Gravatar';
import Timestamp from '../../common/Timestamp';
import { RunDetailsSidebarCancelRunMutation } from './__generated__/RunDetailsSidebarCancelRunMutation.graphql';
import { RunDetailsSidebarFragment_details$key } from './__generated__/RunDetailsSidebarFragment_details.graphql';
import RunStageStatusTypes from './RunStageStatusTypes';

interface Props {
    fragmentRef: RunDetailsSidebarFragment_details$key
    stage: string
    open: boolean
    temporary: boolean
    onClose: () => void
    onError: (error: MutationError) => void
}

export const SidebarWidth = 240;

const RUN_FINALITY_STATES = ['planned_and_finished', 'applied', 'errored', 'canceled']

function RunDetailsSidebar(props: Props) {
    const { stage, open, temporary, onClose, onError } = props;
    const authService = useContext<AuthenticationService>(AuthServiceContext);

    const data = useFragment<RunDetailsSidebarFragment_details$key>(
        graphql`
    fragment RunDetailsSidebarFragment_details on Run
    {
        id
        status
        createdBy
        isDestroy
        assessment
        moduleSource
        moduleVersion
        workspace {
          fullPath
        }
        metadata {
          createdAt
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

    const [commitCancelRun, commitCancelRunInFlight] = useMutation<RunDetailsSidebarCancelRunMutation>(graphql`
        mutation RunDetailsSidebarCancelRunMutation($input: CancelRunInput!) {
            cancelRun(input: $input) {
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

    const cancelRun = () => {
        commitCancelRun({
            variables: {
                input: {
                    runId: data.id
                },
            },
            onCompleted: data => {
                if (data.cancelRun.problems.length) {
                    onError({
                        severity: 'warning',
                        message: data.cancelRun.problems.map(problem => problem.message).join('; ')
                    });
                }
            },
            onError: error => {
                onError({
                    severity: 'error',
                    message: `Unexpected Error Occurred: ${error.message}`
                });
            }
        })
    }

    const onDownloadConfigVersion = async (configVersionId: string) => {
        try {
            const response = await authService.fetchWithAuth(`${cfg.apiUrl}/tfe/v2/configuration-versions/${configVersionId}/content`, {
                method: 'GET',
            });

            if (!response.ok) {
                throw new Error(`request for configuration version content returned status ${response.status}`);
            }

            const blob = await response.blob();
            downloadFile(`${configVersionId}.tar.gz`, blob);
        } catch (error) {
            onError({ message: `failed to download: ${error}`, severity: 'error' })
        }
    }

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
                    {!RUN_FINALITY_STATES.includes(data.status) &&
                        !data.plan.currentJob?.cancelRequested &&
                        !data.apply?.currentJob?.cancelRequested &&
                        <LoadingButton loading={commitCancelRunInFlight} size="small" variant="outlined" color="error" onClick={cancelRun}>Cancel</LoadingButton>}
                </Box>
                <Box marginBottom={3}>
                    <Typography sx={{ marginBottom: 1 }}>Type</Typography>
                    {!data.isDestroy && data.apply && <Chip size="small" label="Apply" />}
                    {data.isDestroy && <Chip size="small" label="Destroy" sx={{ color: red[500] }} />}
                    {!data.apply && <Chip size="small" label={data.assessment ? "Assessment" : "Speculative"} />}
                </Box>
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
                        <Tooltip title="download">
                            <Link
                                color="textPrimary"
                                underline="none"
                                sx={{ wordBreak: 'break-all', cursor: 'pointer' }}
                                onClick={() => onDownloadConfigVersion(data.configurationVersion?.id as string)}
                            >
                                {data.configurationVersion.id.substring(0, 8)}...
                            </Link>
                        </Tooltip>
                        <IconButton sx={{ padding: 0 }} onClick={() => navigator.clipboard.writeText(data.configurationVersion?.id ?? '')}>
                            <CopyIcon sx={{ width: 16, height: 16 }} />
                        </IconButton>
                    </Stack>
                </Box>}
                {data.moduleSource && <Box marginBottom={3}>
                    <Typography sx={{ marginBottom: 1 }}>Module Source</Typography>
                    <Stack direction="row" spacing={1} alignItems="center">
                        <Tooltip title={data.moduleSource}>
                            <Typography sx={{ wordBreak: 'break-all' }}>
                                {`${data.moduleSource.substring(0, 16)}...`}
                            </Typography>
                        </Tooltip>
                        <IconButton sx={{ padding: '4px' }} onClick={() => navigator.clipboard.writeText(data.moduleSource ?? '')}>
                            <CopyIcon sx={{ width: 16, height: 16 }} />
                        </IconButton>
                    </Stack>
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
                        <ListItem button selected={stage === 'plan'} component={LinkRouter} replace to={`/groups/${data.workspace.fullPath}/-/runs/${data.id}/plan`}>
                            <ListItemIcon>
                                <PlanStatusIcon />
                            </ListItemIcon>
                            <ListItemText primary="Plan" />
                        </ListItem>
                        {data.apply && <ListItem button selected={stage === 'apply'} component={LinkRouter} replace to={`/groups/${data.workspace.fullPath}/-/runs/${data.id}/apply`}>
                            <ListItemIcon>
                                <ApplyStatusIcon />
                            </ListItemIcon>
                            <ListItemText primary="Apply" />
                        </ListItem>}
                    </List>
                </Box>
            </Box>
        </Drawer>
    );
}

export default RunDetailsSidebar;
