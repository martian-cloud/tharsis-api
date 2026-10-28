import { ListItem, ListItemIcon, ListItemSecondaryAction, ListItemText, Stack, Typography } from '@mui/material';
import Chip from '@mui/material/Chip';
import TableCell from '@mui/material/TableCell';
import TableRow from '@mui/material/TableRow';
import Tooltip from '@mui/material/Tooltip';
import Box from '@mui/system/Box';
import graphql from 'babel-plugin-relay/macro';
import { useMemo } from 'react';
import { useFragment } from "react-relay/hooks";
import { Link as LinkRouter } from 'react-router-dom';
import ActorDisplay from '../../common/ActorDisplay';
import TRNButton from '../../common/TRNButton';
import Timestamp from '../../common/Timestamp';
import Link from '../../routes/Link';
import RunAnnotations from './RunAnnotations';
import RunStageIcons from './RunStageIcons';
import RunStatusChip from './RunStatusChip';
import { RunListItemFragment_run$key } from './__generated__/RunListItemFragment_run.graphql';

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
            createdByActor {
                ...ActorDisplayFragment_actor
            }
            status
            isDestroy
            assessment
            hasAdvisoryFailures
            workspace {
                fullPath
            }
            apply {
                status
            }
            ...RunStageIconsFragment_run
            ...RunAnnotationsFragment_run
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

    return !mobile ? (
        <TableRow
            sx={{ '&:last-child td, &:last-child th': { border: 0 } }}
        >
            <TableCell>
                <RunStatusChip to={runPath} status={data.status} hasAdvisoryFailures={data.hasAdvisoryFailures} />
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
                {data.isDestroy && <Chip size="small" label="Destroy" sx={{ color: 'runStatus.destroy' }} />}
                {!data.apply && <Chip size="small" label={data.assessment ? "Assessment" : "Speculative"} />}
            </TableCell>
            <TableCell>
                <RunAnnotations fragmentRef={data} emptyPlaceholder="--" maxVisible={3} />
            </TableCell>
            <TableCell>
                <Box display="flex" alignItems="center" gap={1}>
                    <Box display="flex" flexShrink={0}>
                        <ActorDisplay fragmentRef={data.createdByActor} variant="avatar" size={24} />
                    </Box>
                    <Box minWidth={0}>
                        <Timestamp timestamp={data.metadata.createdAt} />
                    </Box>
                </Box>
            </TableCell>
            <TableCell>
                <RunStageIcons fragmentRef={data} />
            </TableCell>
            <TableCell align="right">
                <TRNButton trn={data.metadata.trn} size="small" />
            </TableCell>
        </TableRow>
    ) : (
        <ListItem divider={!last}>
            <ListItemIcon sx={{ minWidth: 80 }}>
                <RunStatusChip to={runPath} status={data.status} hasAdvisoryFailures={data.hasAdvisoryFailures} />
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
                        <Typography variant="body2" color="textSecondary">
                            created <Timestamp variant="inherit" color="inherit" timestamp={data.metadata.createdAt} />
                        </Typography>
                        <Box mt={0.5}>
                            <RunAnnotations fragmentRef={data} maxVisible={3} />
                        </Box>
                    </Stack>
                }
            />
            <ListItemSecondaryAction sx={{ right: 20 }}>
                <ActorDisplay fragmentRef={data.createdByActor} variant="avatar" size={24} />
            </ListItemSecondaryAction>
        </ListItem>
    );
}

export default RunListItem;
