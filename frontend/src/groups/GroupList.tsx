import React, { useEffect, useMemo, useState } from 'react';
import { Box, List, Typography } from '@mui/material';
import throttle from 'lodash.throttle';
import graphql from 'babel-plugin-relay/macro';
import InfiniteScroll from 'react-infinite-scroll-component';
import { ConnectionHandler, fetchQuery, useLazyLoadQuery, usePaginationFragment, useRelayEnvironment } from "react-relay/hooks";
import ListSkeleton from '../skeletons/ListSkeleton';
import GroupListItem from './GroupListItem';
import SearchInput from '../common/SearchInput';
import { GroupListFragment_groups$key } from './__generated__/GroupListFragment_groups.graphql';
import { GroupListPaginationQuery } from './__generated__/GroupListPaginationQuery.graphql';
import { GroupListQuery } from './__generated__/GroupListQuery.graphql';

const INITIAL_ITEM_COUNT = 100;

interface Props {
    groupPath: string
}

const query = graphql`
    query GroupListQuery($first: Int, $last: Int, $after: String, $before: String, $parentPath: String, $search: String) {
        ...GroupListFragment_groups
    }
`;

export function GetConnections(parentPath: any): [any] {
    const connectionId = ConnectionHandler.getConnectionID(
        "root",
        'GroupList_groups',
        { parentPath, sort: "FULL_PATH_ASC" }
    );
    return [connectionId];
}

function GroupList(props: Props) {
    const { groupPath } = props;
    const [search, setSearch] = useState<string | undefined>('');
    const [isRefreshing, setIsRefreshing] = useState(false);

    const queryData = useLazyLoadQuery<GroupListQuery>(query, { first: 100, parentPath: groupPath }, { fetchPolicy: 'store-and-network' });

    const { data, loadNext, hasNext, refetch } = usePaginationFragment<GroupListPaginationQuery, GroupListFragment_groups$key>(
        graphql`
        fragment GroupListFragment_groups on Query
        @refetchable(queryName: "GroupListPaginationQuery") {
            groups(
                after: $after
                before: $before
                first: $first
                last: $last
                parentPath: $parentPath
                search: $search
                sort:FULL_PATH_ASC
            ) @connection(key: "GroupList_groups") {
                totalCount
                edges {
                    node {
                        id
                        ...GroupListItemFragment_group
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

                    fetchQuery(environment, query, { first: INITIAL_ITEM_COUNT, parentPath: groupPath, search: normalizedInput })
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
        [environment, refetch, groupPath],
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

    const edgeCount = (data.groups.edges?.length ?? 0) - 1

    return (
        <Box>
            <SearchInput
                sx={{ marginTop: 2, marginBottom: 2 }}
                placeholder="search for subgroups"
                fullWidth
                onChange={onSearchChange}
                onKeyPress={onKeyPress}
            />
            {(data.groups.edges?.length === 0) && search !== '' && <Typography
                sx={{ p: 4 }}
                align="center"
                color="textSecondary"
            >
                No subgroups matching search <strong>{search}</strong>
            </Typography>}
            <InfiniteScroll
                dataLength={data.groups.edges?.length ?? 0}
                next={() => loadNext(100)}
                hasMore={hasNext}
                loader={<ListSkeleton rowCount={3} />}
            >
                <List sx={isRefreshing ? { opacity: 0.5 } : null} disablePadding>
                    {data.groups.edges?.map((edge: any, index: number) => <GroupListItem key={edge.node.id} groupKey={edge.node} last={index === edgeCount} />)}
                </List>
            </InfiniteScroll>
        </Box>
    );
}

export default GroupList;
