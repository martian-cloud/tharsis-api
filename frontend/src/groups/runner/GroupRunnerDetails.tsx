import { Box, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useFragment, useLazyLoadQuery } from 'react-relay/hooks';
import { useParams } from 'react-router-dom';
import NamespaceBreadcrumbs from '../../namespace/NamespaceBreadcrumbs';
import RunnerDetails from '../../runners/RunnerDetails';
import { GroupRunnerDetailsFragment_group$key } from './__generated__/GroupRunnerDetailsFragment_group.graphql';
import { GroupRunnerDetailsQuery } from './__generated__/GroupRunnerDetailsQuery.graphql';
import { GetConnections } from './GroupRunnersList';

interface Props {
    fragmentRef: GroupRunnerDetailsFragment_group$key;
}

function GroupRunnerDetails({ fragmentRef }: Props) {
    const runnerId = useParams().runnerId as string;

    const group = useFragment<GroupRunnerDetailsFragment_group$key>(graphql`
        fragment GroupRunnerDetailsFragment_group on Group
        {
            id
            fullPath
        }
    `, fragmentRef);

    const data = useLazyLoadQuery<GroupRunnerDetailsQuery>(graphql`
        query GroupRunnerDetailsQuery($id: String!) {
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
                <NamespaceBreadcrumbs
                    namespacePath={group.fullPath}
                    childRoutes={[
                        { title: "runners", path: "runners" },
                        { title: runner.name, path: runner.id }
                    ]}
                />
                <RunnerDetails fragmentRef={data.node} getConnections={() => GetConnections(group.id)} />
            </Box>
        );
    } else {
        return <Box display="flex" justifyContent="center" mt={4}>
            <Typography color="textSecondary">Runner with ID {runnerId} not found</Typography>
        </Box>;
    }
}

export default GroupRunnerDetails
