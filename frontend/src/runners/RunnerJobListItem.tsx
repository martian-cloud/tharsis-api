import { Box, TableCell, TableRow, Tooltip } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import humanizeDuration from 'humanize-duration';
import moment from 'moment';
import { useFragment } from 'react-relay/hooks';
import Link from '../routes/Link';
import { RunnerJobListItemFragment$key } from './__generated__/RunnerJobListItemFragment.graphql';
import JobStatusChip from '../workspace/runs/JobStatusChip';

interface Props {
    fragmentRef: RunnerJobListItemFragment$key
}

function RunnerJobListItem({ fragmentRef }: Props) {
    const data = useFragment(graphql`
        fragment RunnerJobListItemFragment on Job {
            id
            status
            type
            run {
                id
            }
            timestamps {
                queuedAt
                pendingAt
                runningAt
                finishedAt
            }
            workspace {
                name
                fullPath
            }
            metadata {
                createdAt
                updatedAt
            }
        }
    `, fragmentRef);

    const timestamps = data.timestamps;
    const duration = timestamps?.finishedAt ?
        moment.duration(moment(timestamps.finishedAt as moment.MomentInput).diff(moment(timestamps.runningAt as moment.MomentInput))) : null;

    const jobLink = `/groups/${data.workspace.fullPath}/-/runs/${data.run.id}/${data.type}`;

    return (
        <TableRow sx={{ '&:last-child td, &:last-child th': { border: 0 } }}>
            <TableCell>
                <JobStatusChip status={data.status} to={jobLink} />
            </TableCell>
            <TableCell>
                <Link color="inherit" to={jobLink}>{data.id.substring(0, 8)}...</Link>
            </TableCell>
            <TableCell>
                {data.type}
            </TableCell>
            <TableCell>
                <Link color="inherit" to={`/groups/${data.workspace.fullPath}`}>{data.workspace.name}</Link>
            </TableCell>
            <TableCell>
                {duration ? humanizeDuration(duration.asMilliseconds()) : '--'}
            </TableCell>
            <TableCell>
                <Tooltip title={data.metadata.createdAt}>
                    <Box>{moment(data.metadata.createdAt as moment.MomentInput).fromNow()}</Box>
                </Tooltip>
            </TableCell>
        </TableRow >
    );
}

export default RunnerJobListItem;
