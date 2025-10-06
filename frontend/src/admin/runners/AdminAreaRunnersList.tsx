import Box from '@mui/material/Box';
import graphql from 'babel-plugin-relay/macro';
import { ConnectionHandler, useLazyLoadQuery, usePaginationFragment } from 'react-relay/hooks';
import AdminAreaBreadcrumbs from '../AdminAreaBreadcrumbs';
import RunnerList from '../../runners/RunnerList';
import { AdminAreaRunnersListQuery } from './__generated__/AdminAreaRunnersListQuery.graphql';
import { AdminAreaRunnersListPaginationQuery } from './__generated__/AdminAreaRunnersListPaginationQuery.graphql';
import { AdminAreaRunnersListFragment_sharedRunners$key } from './__generated__/AdminAreaRunnersListFragment_sharedRunners.graphql';

export const INITIAL_ITEM_COUNT = 100;

export function GetConnections(): [string] {
    const connectionId = ConnectionHandler.getConnectionID(
        'root',
        'AdminAreaRunnersList_sharedRunners'
    );
    return [connectionId];
}

const query = graphql`
    query AdminAreaRunnersListQuery($first: Int!, $after: String) {
        ...AdminAreaRunnersListFragment_sharedRunners
    }
`;

function AdminAreaRunnersList() {

    const queryData = useLazyLoadQuery<AdminAreaRunnersListQuery>(query, { first: INITIAL_ITEM_COUNT });

    const { data, loadNext, hasNext } = usePaginationFragment<AdminAreaRunnersListPaginationQuery, AdminAreaRunnersListFragment_sharedRunners$key>(
        graphql`
            fragment AdminAreaRunnersListFragment_sharedRunners on Query
            @refetchable(queryName: "AdminAreaRunnersListPaginationQuery")
             {
                sharedRunners(
                    first: $first
                    after: $after
                ) @connection(key: "AdminAreaRunnersList_sharedRunners") {
                    edges {
                        node {
                            id
                        }
                    }
                    ...RunnerListFragment_runners
                }
            }
        `, queryData);

    return (
        <Box>
            <AdminAreaBreadcrumbs
                childRoutes={[
                    { title: "runners", path: 'runners' }
                ]}
            />
            <Box>
                <RunnerList
                    fragmentRef={data.sharedRunners}
                    hasNext={hasNext}
                    loadNext={loadNext}
                    hideNewRunnerButton
                />
            </Box>
        </Box>
    );
}

export default AdminAreaRunnersList
