import { Box, Tooltip, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useFragment } from 'react-relay/hooks';
import Gravatar from '../common/Gravatar';
import { ResponsiveRow } from '../common/ResponsiveTable';
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
                updatedAt
            }
            id
            name
            disabled
            createdBy
            groupPath
        }
    `, fragmentRef);

    const name = (
        <>
            <Typography fontWeight={500}>
                <Link color="inherit" to={data.id}>{data.name}</Link>
            </Typography>
            {inherited && <Typography mt={0.5} color="textSecondary" variant="caption">Inherited from group <strong>{data.groupPath}</strong></Typography>}
        </>
    );

    const created = (
        <Box display="flex" alignItems="center" gap={0.5}>
            <Timestamp variant="body2" timestamp={data.metadata.createdAt} />
            <Typography variant="body2">by</Typography>
            <Tooltip title={data.createdBy}>
                <Box display="flex">
                    <Gravatar width={20} height={20} email={data.createdBy} />
                </Box>
            </Tooltip>
        </Box>
    );

    return (
        <ResponsiveRow cells={[
            { primary: true, content: name },
            { label: 'Status', content: <RunnerChip disabled={data.disabled} /> },
            { label: 'Created', content: created },
            { label: 'Last Updated', content: <Timestamp variant="body2" timestamp={data.metadata.updatedAt} /> },
        ]} />
    );
}

export default RunnerListItem
