import { Paper, Stack, Typography } from '@mui/material';
import Box from '@mui/material/Box';
import graphql from 'babel-plugin-relay/macro';
import { RocketLaunchOutline as RunIcon } from 'mdi-material-ui';
import React from 'react';
import { useFragment } from 'react-relay/hooks';
import Link from '../routes/Link';
import RunStageIcons from './runs/RunStageIcons';
import { WorkspaceDetailsCurrentJobFragment_workspace$key } from './__generated__/WorkspaceDetailsCurrentJobFragment_workspace.graphql';

interface Props {
    fragmentRef: WorkspaceDetailsCurrentJobFragment_workspace$key
}

function WorkspaceDetailsCurrentJob(props: Props) {
    const { fragmentRef } = props;

    const data = useFragment<WorkspaceDetailsCurrentJobFragment_workspace$key>(
        graphql`
      fragment WorkspaceDetailsCurrentJobFragment_workspace on Workspace
      {
        id
        fullPath
        currentJob {
            id
            type
            run {
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
      }
    `, fragmentRef);

    return data.currentJob ? (
        <Paper variant="outlined" sx={{ padding: 2 }}>
            <Box display="flex" justifyContent="space-between" alignItems="center">
                <Stack direction="row" spacing={2}>
                    <RunIcon />
                    <Typography component="div">
                        {data.currentJob.type.charAt(0).toUpperCase() + data.currentJob.type.slice(1)} is currently in progress for run{' '}
                        <Link to={`/groups/${data.fullPath}/-/runs/${data.currentJob.run.id}`}>
                            {data.currentJob.run.id.substring(0, 8)}...
                        </Link>
                    </Typography>
                </Stack>
                <RunStageIcons
                    runPath={`/groups/${data.fullPath}/-/runs/${data.currentJob.run.id}`}
                    planStatus={data.currentJob.run.plan.status}
                    applyStatus={data.currentJob.run.apply?.status}
                />
            </Box>
        </Paper>
    ) : null;
}

export default WorkspaceDetailsCurrentJob;
