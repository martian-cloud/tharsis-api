import React, { useEffect, useMemo, useState } from 'react';
import { Box, List, Typography } from '@mui/material';
import throttle from 'lodash.throttle';
import graphql from 'babel-plugin-relay/macro';
import InfiniteScroll from 'react-infinite-scroll-component';
import { ConnectionHandler, fetchQuery, useFragment, useLazyLoadQuery, usePaginationFragment, useRelayEnvironment } from "react-relay/hooks";
import ListSkeleton from '../skeletons/ListSkeleton';
import WorkspaceListItem from './WorkspaceListItem';
import SearchInput from '../common/SearchInput';
import { WorkspaceListFragment_workspaces$key } from './__generated__/WorkspaceListFragment_workspaces.graphql';
import { WorkspaceListPaginationQuery } from './__generated__/WorkspaceListPaginationQuery.graphql';
import { WorkspaceListQuery } from './__generated__/WorkspaceListQuery.graphql';
import { WorkspaceListFragment_group$key } from './__generated__/WorkspaceListFragment_group.graphql';

const INITIAL_ITEM_COUNT = 100;

interface Props {
    fragmentRef: WorkspaceListFragment_group$key
}

const query = graphql`
    query WorkspaceListQuery($first: Int, $last: Int, $after: String, $before: String, $groupId: String!, $search: String) {
        ...WorkspaceListFragment_workspaces
    }
`;

export function GetConnections(groupPath: string): [string] {
    const connectionId = ConnectionHandler.getConnectionID(
        "root",
        "WorkspaceList_workspaces",
        { groupPath: groupPath, sort: 'FULL_PATH_ASC' }
    );
    return [connectionId];
}

function WorkspaceList({ fragmentRef }: Props) {
    const [search, setSearch] = useState<string | undefined>('');
    const [isRefreshing, setIsRefreshing] = useState(false);

        const group = useFragment<WorkspaceListFragment_group$key >(
            graphql`
            fragment WorkspaceListFragment_group on Group
            {
                id
            }
        `, fragmentRef);

    const queryData = useLazyLoadQuery<WorkspaceListQuery>(query, { first: INITIAL_ITEM_COUNT, groupId: group.id }, { fetchPolicy: 'store-and-network' });

    const { data, loadNext, hasNext, refetch } = usePaginationFragment<WorkspaceListPaginationQuery, WorkspaceListFragment_workspaces$key>(
        graphql`
        fragment WorkspaceListFragment_workspaces on Query
        @refetchable(queryName: "WorkspaceListPaginationQuery") {
            workspaces(
                after: $after
                before: $before
                first: $first
                last: $last
                groupId: $groupId
                search: $search
                sort: FULL_PATH_ASC
            ) @connection(key: "WorkspaceList_workspaces") {
                totalCount
                edges {
                    node {
                        id
                        ...WorkspaceListItemFragment_workspace
                    }
                }
            }
        }
    `, queryData);

    const environment = useRelayEnvironment();

    const fetch = useMemo(
        () =>
            throttle(
                (input?: string) => {
                    setIsRefreshing(true);

                    const normalizedInput = input?.trim();

                    fetchQuery(environment, query, { first: INITIAL_ITEM_COUNT, groupId: group.id, search: normalizedInput })
                        .subscribe({
                            complete: () => {
                                setIsRefreshing(false);
                                setSearch(input);

                                // *After* the query has been fetched, we call
                                // refetch again to re-render with the updated data.
                                // At this point the data for the query should
                                // be cached, so we use the 'store-only'
                                // fetchPolicy to avoid suspending.
                                refetch({ first: INITIAL_ITEM_COUNT, search: normalizedInput }, { fetchPolicy: 'store-only' });
                            },
                            error: () => {
                                setIsRefreshing(false);
                            }
                        });
                },
                2000,
                { leading: false, trailing: true }
            ),
        [environment, refetch, group.id],
    );

    useEffect(() => {
        return () => {
            fetch.cancel()
        }
    }, [fetch]);

    const onSearchChange = (event: React.ChangeEvent<HTMLInputElement>) => {
        fetch(event.target.value.toLowerCase());
    };

    const onKeyPress = (event: React.KeyboardEvent<HTMLInputElement>) => {
        // Only handle enter key type
        if (event.which === 13) {
            fetch.flush();
        }
    };

    const edgeCount = (data.workspaces.edges?.length ?? 0) - 1

    return (
        <Box>
            <SearchInput
                sx={{ marginTop: 2, marginBottom: 2 }}
                placeholder="search for workspaces"
                fullWidth
                onChange={onSearchChange}
                onKeyPress={onKeyPress}
            />
            {(data.workspaces.edges?.length === 0) && search !== '' && <Typography
                sx={{ p: 4 }}
                align="center"
                color="textSecondary"
            >
                No workspaces matching search <strong>{search}</strong>
            </Typography>}
            <InfiniteScroll
                dataLength={data.workspaces.edges?.length ?? 0}
                next={() => loadNext(100)}
                hasMore={hasNext}
                loader={<ListSkeleton rowCount={3} />}
            >
                <List sx={isRefreshing ? { opacity: 0.5 } : null} disablePadding>
                    {data.workspaces.edges?.map((edge: any, index: number) => <WorkspaceListItem key={edge.node.id} workspaceKey={edge.node} last={index === edgeCount} />)}
                </List>
            </InfiniteScroll>
        </Box>
    );
}

export default WorkspaceList;
