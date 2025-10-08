import {
    Avatar,
    Box,
    Link,
    ListItem,
    ListItemIcon,
    ListItemSecondaryAction,
    ListItemText,
    Stack,
    Tooltip,
    Typography
} from '@mui/material';
import teal from '@mui/material/colors/teal';
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

    const data = useFragment(graphql`
        fragment HomeRunListItemFragment_run on Run {
            id
            createdBy
            metadata {
                createdAt
            }
            plan {
                status
            }
            apply {
                status
            }
            workspace {
                fullPath
            }
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
                    bgcolor: teal[200],
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
            <ListItemIcon sx={{ minWidth: 60 }}>
                <RunStageIcons planStatus={data.plan.status} applyStatus={data.apply?.status} runPath={runPath} />
            </ListItemIcon>
            <ListItemText
                primary={
                    <Stack>
                        <Link
                            to={runPath}
                            component={LinkRouter}
                            underline="hover"
                            fontWeight={500}
                            variant="body2"
                            color="textPrimary"
                        >
                            {`${data.id.substring(0, 8)}...`}
                        </Link>
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
                        <Box display="flex">
                            <Typography variant="body2" color="textSecondary">created</Typography>
                            <Timestamp ml={0.5} variant="body2" color="textSecondary" timestamp={data.metadata.createdAt} />
                        </Box>
                    </Stack>}
            />
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
