import Chip from '@mui/material/Chip';
import TableCell from '@mui/material/TableCell';
import TableRow from '@mui/material/TableRow';
import Tooltip from '@mui/material/Tooltip';
import red from '@mui/material/colors/red';
import Box from '@mui/system/Box';
import graphql from 'babel-plugin-relay/macro';
import { useFragment } from "react-relay/hooks";
import Gravatar from '../../common/Gravatar';
import Timestamp from '../../common/Timestamp';
import Link from '../../routes/Link';
import RunStageIcons from './RunStageIcons';
import RunStatusChip from './RunStatusChip';
import TRNButton from '../../common/TRNButton';
import { RunListItemFragment_run$key } from './__generated__/RunListItemFragment_run.graphql';

interface Props {
    runKey: RunListItemFragment_run$key
    workspacePath: string
}

function RunListItem(props: Props) {
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
            plan {
                status
            }
            apply {
                status
            }
        }
    `, props.runKey)

    const runPath = `/groups/${props.workspacePath}/-/runs/${data.id}`;

    return (
        <TableRow
            sx={{ '&:last-child td, &:last-child th': { border: 0 } }}
        >
            <TableCell>
                <RunStatusChip to={runPath} status={data.status} />
            </TableCell>
            <TableCell>
                <Link color="inherit" to={runPath}>{data.id.substring(0, 8)}...</Link>
            </TableCell>
            <TableCell>
                {!data.isDestroy && data.apply && <Chip size="small" label="Apply" />}
                {data.isDestroy && <Chip size="small" label="Destroy" sx={{ color: red[500] }} />}
                {!data.apply && <Chip size="small" label={data.assessment ? "Assessment" : "Speculative"} />}
            </TableCell>
            <TableCell>
                <Box display="flex" alignItems="center">
                    <Tooltip title={data.createdBy}>
                        <Box>
                            <Gravatar width={24} height={24} email={data.createdBy} />
                        </Box>
                    </Tooltip>
                    <Timestamp ml={1} timestamp={data.metadata.createdAt} />
                </Box>
            </TableCell>
            <TableCell>
                <RunStageIcons planStatus={data.plan.status} applyStatus={data.apply?.status} runPath={runPath} />
            </TableCell>
            <TableCell align="right">
                <TRNButton trn={data.metadata.trn} size="small"/>
            </TableCell>
        </TableRow>
    );
}

export default RunListItem;
