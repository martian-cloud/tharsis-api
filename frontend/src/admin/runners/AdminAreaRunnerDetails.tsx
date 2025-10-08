import { Box, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useLazyLoadQuery } from 'react-relay/hooks';
import { useParams } from 'react-router-dom';
import RunnerDetails from '../../runners/RunnerDetails';
import AdminAreaBreadcrumbs from '../AdminAreaBreadcrumbs';
import { GetConnections } from './AdminAreaRunnersList';
import { AdminAreaRunnerDetailsQuery } from './__generated__/AdminAreaRunnerDetailsQuery.graphql';

function AdminAreaRunnerDetails() {
    const runnerId = useParams().runnerId as string;

    const data = useLazyLoadQuery<AdminAreaRunnerDetailsQuery>(graphql`
        query AdminAreaRunnerDetailsQuery($id: String!) {
            node(id: $id) {
                ... on Runner {
                    id
                    name
                    ...RunnerDetailsFragment_runner
                }
            }
        }`, { id: runnerId }, { fetchPolicy: 'store-and-network' });

    const runner = data.node as any;

    if (data.node) {
        return (
            <Box>
                <AdminAreaBreadcrumbs
                    childRoutes={[
                        { title: "runners", path: 'runners' },
                        { title: runner.name, path: runner.id }
                    ]}
                />
                <RunnerDetails fragmentRef={data.node} getConnections={GetConnections} />
            </Box>
        );
    } else {
        return <Box display="flex" justifyContent="center" mt={4}>
            <Typography color="textSecondary">Runner with ID {runnerId} not found</Typography>
        </Box>;
    }
}

export default AdminAreaRunnerDetails
