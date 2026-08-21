import MiddleDot from '@/common/MiddleDot';
import {
    Avatar,
    Box, alpha, Chip, Link,
    ListItem,
    ListItemSecondaryAction,
    ListItemText,
    Stack,
    Tooltip, useTheme
} from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useMemo } from 'react';
import { useFragment } from 'react-relay/hooks';
import { Link as LinkRouter } from 'react-router-dom';
import Gravatar from '../common/Gravatar';
import Timestamp from '../common/Timestamp';
import RunStageIcons from '../workspace/runs/RunStageIcons';
import { HomeRunListItemFragment_run$key } from './__generated__/HomeRunListItemFragment_run.graphql';

const getServiceAccountInitial = (serviceAccount: string): string => {
    const lastSlashIndex = serviceAccount.lastIndexOf('/');
    return serviceAccount.charAt(lastSlashIndex + 1).toUpperCase();
};

interface Props {
    fragmentRef: HomeRunListItemFragment_run$key;
    last?: boolean;
}

function HomeRunListItem({ fragmentRef, last }: Props) {
    const theme = useTheme();

    const data = useFragment(graphql`
        fragment HomeRunListItemFragment_run on Run {
            id
            createdBy
            isDestroy
            metadata {
                createdAt
            }
            workspace {
                fullPath
            }
            ...RunStageIconsFragment_run
        }
    `, fragmentRef);

    const workspacePath = `/groups/${data.workspace.fullPath}`;
    const runPath = `${workspacePath}/-/runs/${data.id}`;

    const formattedWorkspacePath = useMemo(() => {
        const path = data.workspace.fullPath;
        const pathParts = path.split('/');

        return pathParts.length > 3 ? `${pathParts[0]} / ... / ${pathParts[pathParts.length - 1]}` : path;

    }, [data.workspace.fullPath]);

    const avatar = useMemo(() => {
        return data.createdBy.includes('/') ?
            <Avatar variant="rounded"
                sx={{
                    width: 20,
                    height: 20,
                    bgcolor: 'avatar.default',
                    fontSize: 14,
                    fontWeight: 500
                }}>
                {getServiceAccountInitial(data.createdBy)}
            </Avatar>
            :
            <Gravatar width={20} height={20} email={data.createdBy} />
    }, [data.createdBy]);

    return (
        <ListItem
            divider={!last}
        >
            <ListItemText
                primary={
                    <Stack>
                        <Box display="flex" alignItems="center">
                            <Link
                                to={runPath}
                                component={LinkRouter}
                                underline="hover"
                                fontWeight={500}
                                variant="body2"
                                color="textPrimary"
                            >
                                {`${data.id.substring(0, 8)}`}
                            </Link>
                            <MiddleDot />
                            <Timestamp variant="body2" sx={{ color: alpha(theme.palette.text.primary, 0.5) }} timestamp={data.metadata.createdAt} />
                        </Box>
                        <Tooltip title={data.workspace.fullPath}>
                            <Link
                                sx={{ mb: 0.5, wordWrap: 'break-word' }}
                                to={workspacePath}
                                component={LinkRouter}
                                underline="hover"
                                variant="body2"
                                color="textSecondary">
                                {formattedWorkspacePath}
                            </Link>
                        </Tooltip>
                        <Box mt={0.5}>
                            <RunStageIcons fragmentRef={data} />
                        </Box>
                    </Stack>}
            />
            {data.isDestroy && (
                <Chip size="small" label="Destroy" sx={{ position: 'absolute', top: 12, right: 16, color: 'runStatus.destroy' }} />
            )}
            <ListItemSecondaryAction>
                <Tooltip title={data.createdBy}>
                    <Box>
                        {avatar}
                    </Box>
                </Tooltip>
            </ListItemSecondaryAction>
        </ListItem>
    );
}

export default HomeRunListItem;
