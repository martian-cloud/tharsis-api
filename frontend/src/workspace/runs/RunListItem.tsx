import { Avatar, ListItem, ListItemIcon, ListItemSecondaryAction, ListItemText, Stack, Typography } from '@mui/material';
import Chip from '@mui/material/Chip';
import TableCell from '@mui/material/TableCell';
import TableRow from '@mui/material/TableRow';
import Tooltip from '@mui/material/Tooltip';
import { teal } from '@mui/material/colors';
import red from '@mui/material/colors/red';
import Box from '@mui/system/Box';
import graphql from 'babel-plugin-relay/macro';
import { useMemo } from 'react';
import { useFragment } from "react-relay/hooks";
import { Link as LinkRouter } from 'react-router-dom';
import Gravatar from '../../common/Gravatar';
import TRNButton from '../../common/TRNButton';
import Timestamp from '../../common/Timestamp';
import Link from '../../routes/Link';
import RunStageIcons from './RunStageIcons';
import RunStatusChip from './RunStatusChip';
import { RunListItemFragment_run$key } from './__generated__/RunListItemFragment_run.graphql';

const getServiceAccountInitial = (serviceAccount: string): string => {
    const lastSlashIndex = serviceAccount.lastIndexOf('/');
    return serviceAccount.charAt(lastSlashIndex + 1).toUpperCase();
};

interface Props {
    runFragment: RunListItemFragment_run$key
    mobile: boolean
    displayWorkspacePath?: boolean
    last?: boolean
}

function RunListItem({ runFragment, displayWorkspacePath, mobile, last }: Props) {
    const data = useFragment<RunListItemFragment_run$key>(graphql`
        fragment RunListItemFragment_run on Run {
            metadata {
                createdAt
                trn
            }
            id
            createdBy
            status
            isDestroy
            assessment
            workspace {
                fullPath
            }
            plan {
                status
            }
            apply {
                status
            }
        }
    `, runFragment)

    const workspacePath = `/groups/${data.workspace.fullPath}`;
    const runPath = `${workspacePath}/-/runs/${data.id}`;

    const formattedWorkspacePath = useMemo(() => {
        if (!displayWorkspacePath) {
            return '';
        }
        const path = data.workspace.fullPath;
        const pathParts = path.split('/');

        return pathParts.length > 3 ? `${pathParts[0]} / ... / ${pathParts[pathParts.length - 1]}` : path;

    }, [data.workspace.fullPath, displayWorkspacePath]);

    const avatar = useMemo(() => {
        return data.createdBy.includes('/') ?
            <Avatar variant="rounded"
                sx={{
                    width: 24,
                    height: 24,
                    bgcolor: teal[200],
                    fontSize: 14,
                    fontWeight: 500
                }}>
                {getServiceAccountInitial(data.createdBy)}
            </Avatar>
            :
            <Gravatar width={24} height={24} email={data.createdBy} />
    }, [data.createdBy]);

    return !mobile ? (
        <TableRow
            sx={{ '&:last-child td, &:last-child th': { border: 0 } }}
        >
            <TableCell>
                <RunStatusChip to={runPath} status={data.status} />
            </TableCell>
            <TableCell>
                <Link color="inherit" to={runPath}>{data.id.substring(0, 8)}...</Link>
            </TableCell>
            {displayWorkspacePath && <TableCell>
                <Tooltip title={data.workspace.fullPath} placement="bottom-start">
                    <Box>
                        <Link color="inherit" sx={{ wordWrap: 'break-word' }} to={workspacePath}>{formattedWorkspacePath}</Link>
                    </Box>
                </Tooltip>
            </TableCell>}
            <TableCell>
                {!data.isDestroy && data.apply && <Chip size="small" label="Apply" />}
                {data.isDestroy && <Chip size="small" label="Destroy" sx={{ color: red[500] }} />}
                {!data.apply && <Chip size="small" label={data.assessment ? "Assessment" : "Speculative"} />}
            </TableCell>
            <TableCell>
                <Box display="flex" alignItems="center">
                    <Tooltip title={data.createdBy}>
                        <Box>
                            {avatar}
                        </Box>
                    </Tooltip>
                    <Timestamp ml={1} timestamp={data.metadata.createdAt} />
                </Box>
            </TableCell>
            <TableCell>
                <RunStageIcons planStatus={data.plan.status} applyStatus={data.apply?.status} runPath={runPath} />
            </TableCell>
            <TableCell align="right">
                <TRNButton trn={data.metadata.trn} size="small" />
            </TableCell>
        </TableRow>
    ) : (
        <ListItem divider={!last}>
            <ListItemIcon sx={{ minWidth: 80 }}>
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
                        {displayWorkspacePath && <Tooltip title={data.workspace.fullPath} placement="bottom-start">
                            <Box>
                                <Link
                                    sx={{ mb: 0.5, wordWrap: 'break-word' }}
                                    to={workspacePath}
                                    component={LinkRouter}
                                    underline="hover"
                                    variant="body2"
                                    color="textSecondary">
                                    {formattedWorkspacePath}
                                </Link>
                            </Box>
                        </Tooltip>}
                        <Box display="flex">
                            <Typography variant="body2" color="textSecondary">created</Typography>
                            <Timestamp ml={0.5} variant="body2" color="textSecondary" timestamp={data.metadata.createdAt} />
                        </Box>
                    </Stack>
                }
            />
            <ListItemSecondaryAction sx={{ right: 20 }}>
                <Tooltip title={data.createdBy}>
                    <Box>
                        {avatar}
                    </Box>
                </Tooltip>
            </ListItemSecondaryAction>
        </ListItem>
    );
}

export default RunListItem;
