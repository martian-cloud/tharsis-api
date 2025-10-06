import { Box, Link, List, ListItem, Typography, useTheme } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import throttle from 'lodash.throttle';
import { useEffect, useMemo, useState } from 'react';
import { fetchQuery, useLazyLoadQuery, usePaginationFragment, useRelayEnvironment } from 'react-relay/hooks';
import SearchInput from '../common/SearchInput';
import HomeWorkspaceListItem from './HomeWorkspaceListItem';
import { HomeWorkspaceListFragment_workspaces$key } from './__generated__/HomeWorkspaceListFragment_workspaces.graphql';
import { HomeWorkspaceListPaginationQuery } from './__generated__/HomeWorkspaceListPaginationQuery.graphql';
import { HomeWorkspaceListQuery } from './__generated__/HomeWorkspaceListQuery.graphql';

const INITIAL_ITEM_COUNT = 7;

const query = graphql`
    query HomeWorkspaceListQuery($first: Int!, $after: String, $search: String) {
        ...HomeWorkspaceListFragment_workspaces
    }
`;

function HomeWorkspaceList() {
    const theme = useTheme();
    const [search, setSearch] = useState<string | undefined>('');
    const [isRefreshing, setIsRefreshing] = useState(false);

    const queryData = useLazyLoadQuery<HomeWorkspaceListQuery>(query, { first: INITIAL_ITEM_COUNT }, { fetchPolicy: 'store-and-network' });

    const { data, loadNext, hasNext, refetch } = usePaginationFragment<HomeWorkspaceListPaginationQuery, HomeWorkspaceListFragment_workspaces$key>(
        graphql`
        fragment HomeWorkspaceListFragment_workspaces on Query
        @refetchable(queryName: "HomeWorkspaceListPaginationQuery") {
            workspaces(
                first: $first
                after: $after
                search: $search
                sort: UPDATED_AT_DESC
                ) @connection(key: "HomeWorkspaceList_workspaces") {
                    totalCount
                    edges {
                        node {
                            id
                            ...HomeWorkspaceListItemFragment_workspace
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

                    fetchQuery(environment, query, { first: INITIAL_ITEM_COUNT, search: normalizedInput })
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
                        }
                        );
                },
                2000,
                { leading: false, trailing: true }
            ),
        [environment, refetch],
    );

    useEffect(() => {
        return () => {
            fetch.cancel()
        }
    }, [fetch]);

    const onSearchChange = (event: React.ChangeEvent<HTMLInputElement>) => {
        fetch(event.target.value.toLowerCase().trim());
    };

    const onKeyDown = (event: React.KeyboardEvent<HTMLInputElement>) => {
        // Only handle enter key type
        if (event.key === 'Enter') {
            fetch.flush();
        }
    };

    const edges = data?.workspaces?.edges ?? [];
    const edgeCount = edges.length;

    return (
        <Box>
            {search === '' && edgeCount === 0 && <Box sx={{ p: 2, border: `1px dashed ${theme.palette.divider}`, borderRadius: 2 }}>
                <Typography color="textSecondary" variant="body2">You are not currently a member of any workspaces.</Typography>
            </Box>}
            {(search !== '' || edgeCount !== 0) && <Box>
                {(edgeCount > 1 || search !== '') && <SearchInput
                    placeholder="search by workspace name"
                    fullWidth
                    onChange={onSearchChange}
                    onKeyDown={onKeyDown}
                />}
                {(edgeCount === 0 && search !== '') && <Typography
                    sx={{ p: 2 }}
                    variant="body2"
                    align="center"
                    color="textSecondary"
                >
                    No Workspaces matching search <strong>{search}</strong>
                </Typography>}
                <List sx={isRefreshing ? { opacity: 0.5 } : null} >
                    {edges.map((edge: any, index: number) => <HomeWorkspaceListItem
                        key={edge.node.id}
                        fragmentRef={edge.node}
                        last={index === (edgeCount - 1)}
                    />)}
                    {hasNext && <ListItem>
                        <Link
                            variant="body2"
                            color="textSecondary"
                            sx={{ cursor: 'pointer' }}
                            underline="hover"
                            onClick={() => loadNext(INITIAL_ITEM_COUNT)}
                        >
                            Show more
                        </Link>
                    </ListItem>}
                </List>
            </Box>}
        </Box>
    );
}

export default HomeWorkspaceList;
