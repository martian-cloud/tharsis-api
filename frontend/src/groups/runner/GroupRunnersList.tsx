import graphql from 'babel-plugin-relay/macro';
import { ConnectionHandler, useFragment, useLazyLoadQuery, usePaginationFragment } from "react-relay";
import RunnerList from '../../runners/RunnerList';
import { GroupRunnersListFragment_group$key } from './__generated__/GroupRunnersListFragment_group.graphql';
import { GroupRunnersListFragment_runners$key } from './__generated__/GroupRunnersListFragment_runners.graphql';
import { GroupRunnersListPaginationQuery } from './__generated__/GroupRunnersListPaginationQuery.graphql';
import { GroupRunnersListQuery } from './__generated__/GroupRunnersListQuery.graphql';

export function GetConnections(groupId: string): [string] {
    const connectionId = ConnectionHandler.getConnectionID(
        groupId,
        'GroupRunnersList_runners',
    );
    return [connectionId];
}

interface Props {
    fragmentRef: GroupRunnersListFragment_group$key
}

function GroupRunnersList({ fragmentRef }: Props) {
    const group = useFragment<GroupRunnersListFragment_group$key>(
        graphql`
            fragment GroupRunnersListFragment_group on Group {
                id
            }
        `,
        fragmentRef
    );

    const queryData = useLazyLoadQuery<GroupRunnersListQuery>(graphql`
        query GroupRunnersListQuery($id: String!, $first: Int!, $after: String) {
            node(id: $id) {
                ... on Group {
                    ...GroupRunnersListFragment_runners
                    fullPath
                }
            }
        }`, { id: group.id, first: 20 }, { fetchPolicy: 'store-and-network' });

    const { data, loadNext, hasNext } = usePaginationFragment<GroupRunnersListPaginationQuery, GroupRunnersListFragment_runners$key>(
        graphql`
            fragment GroupRunnersListFragment_runners on Group
            @refetchable(queryName: "GroupRunnersListPaginationQuery") {
                runners(
                    after: $after
                    first: $first
                    includeInherited: true
                    sort: GROUP_LEVEL_DESC
                ) @connection(key: "GroupRunnersList_runners") {
                    edges {
                        node {
                            id
                        }
                    }
                    ...RunnerListFragment_runners
                }
            }
        `, queryData.node);

    return data ? (
        <RunnerList fragmentRef={data.runners} loadNext={loadNext} hasNext={hasNext} groupPath={queryData?.node?.fullPath} />
    ) : null;
}

export default GroupRunnersList;
