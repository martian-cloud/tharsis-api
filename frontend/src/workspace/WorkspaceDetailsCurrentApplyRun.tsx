import { Paper, Stack, Typography } from '@mui/material';
import Box from '@mui/material/Box';
import graphql from 'babel-plugin-relay/macro';
import { RocketLaunchOutline as RunIcon } from 'mdi-material-ui';
import React from 'react';
import { useFragment } from 'react-relay/hooks';
import Link from '../routes/Link';
import RunStageIcons from './runs/RunStageIcons';
import { WorkspaceDetailsCurrentApplyRunFragment_workspace$key } from './__generated__/WorkspaceDetailsCurrentApplyRunFragment_workspace.graphql';

interface Props {
    fragmentRef: WorkspaceDetailsCurrentApplyRunFragment_workspace$key
}

function WorkspaceDetailsCurrentApplyRun(props: Props) {
    const { fragmentRef } = props;

    const data = useFragment<WorkspaceDetailsCurrentApplyRunFragment_workspace$key>(
        graphql`
      fragment WorkspaceDetailsCurrentApplyRunFragment_workspace on Workspace
      {
        id
        fullPath
        currentApplyRun {
            id
            status
            createdBy
            isDestroy
            moduleSource
            moduleVersion
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
            }
            apply {
                status
                triggeredBy
                metadata {
                    createdAt
                    updatedAt
                }
            }
        }
      }
    `, fragmentRef);

    return data.currentApplyRun ? (
        <Paper variant="outlined" sx={{ padding: 2 }}>
            <Box display="flex" justifyContent="space-between" alignItems="center">
                <Stack direction="row" spacing={2}>
                    <RunIcon />
                    <Typography component="div">
                        Run{' '}
                        <Link to={`/groups/${data.fullPath}/-/runs/${data.currentApplyRun.id}`}>
                            {data.currentApplyRun.id.substring(0, 8)}...
                        </Link>
                        {' '}is currently in progress
                    </Typography>
                </Stack>
                <RunStageIcons
                    runPath={`/groups/${data.fullPath}/-/runs/${data.currentApplyRun.id}`}
                    planStatus={data.currentApplyRun.plan.status}
                    applyStatus={data.currentApplyRun.apply?.status}
                />
            </Box>
        </Paper>
    ) : null;
}

export default WorkspaceDetailsCurrentApplyRun;
