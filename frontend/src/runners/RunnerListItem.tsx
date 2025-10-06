import { Box, TableCell, TableRow, Tooltip, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useFragment } from 'react-relay/hooks';
import Gravatar from '../common/Gravatar';
import Timestamp from '../common/Timestamp';
import Link from '../routes/Link';
import RunnerChip from './RunnerChip';
import { RunnerListItemFragment_runner$key } from './__generated__/RunnerListItemFragment_runner.graphql';

interface Props {
    fragmentRef: RunnerListItemFragment_runner$key
    inherited?: boolean
}

function RunnerListItem({ fragmentRef, inherited }: Props) {

    const data = useFragment<RunnerListItemFragment_runner$key>(graphql`
        fragment RunnerListItemFragment_runner on Runner {
            metadata {
                createdAt
            }
            id
            name
            disabled
            createdBy
            groupPath
        }
    `, fragmentRef);

    return (
        <TableRow sx={{ '&:last-child td, &:last-child th': { border: 0 } }}>
            <TableCell>
                <Typography fontWeight={500}>
                    <Link color="inherit" to={data.id}>{data.name}</Link>
                </Typography>
                {inherited && <Typography mt={0.5} color="textSecondary" variant="caption">Inherited from group <strong>{data.groupPath}</strong></Typography>}
            </TableCell>
            <TableCell>
                <RunnerChip disabled={data.disabled} />
            </TableCell>
            <TableCell>
                <Box display="flex" alignItems="center">
                    <Tooltip title={data.createdBy}>
                        <Box>
                            <Gravatar width={24} height={24} email={data.createdBy} />
                        </Box>
                    </Tooltip>
                    <Box ml={1}>
                        <Timestamp variant="body2" timestamp={data.metadata.createdAt} />
                    </Box>
                </Box>
            </TableCell>
        </TableRow>
    );
}

export default RunnerListItem
