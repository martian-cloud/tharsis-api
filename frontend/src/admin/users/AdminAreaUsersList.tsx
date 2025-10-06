import { Paper, Table, TableBody, TableCell, TableContainer, TableHead, TableRow, Typography, useTheme } from '@mui/material';
import Box from '@mui/material/Box';
import graphql from 'babel-plugin-relay/macro';
import throttle from 'lodash.throttle';
import React, { useMemo, useState } from 'react';
import InfiniteScroll from 'react-infinite-scroll-component';
import { fetchQuery, useLazyLoadQuery, usePaginationFragment, useRelayEnvironment } from 'react-relay/hooks';
import SearchInput from '../../common/SearchInput';
import ListSkeleton from '../../skeletons/ListSkeleton';
import AdminAreaBreadcrumbs from '../AdminAreaBreadcrumbs';
import { AdminAreaUsersListFragment_users$key } from './__generated__/AdminAreaUsersListFragment_users.graphql';
import { AdminAreaUsersListPaginationQuery } from './__generated__/AdminAreaUsersListPaginationQuery.graphql';
import { AdminAreaUsersListQuery } from './__generated__/AdminAreaUsersListQuery.graphql';
import AdminAreaUserListItem from './AdminAreaUserListItem';

const INITIAL_ITEM_COUNT = 50;

const query = graphql`
    query AdminAreaUsersListQuery($first: Int!, $after: String, $search: String) {
        ...AdminAreaUsersListFragment_users
    }
`;

function AdminAreaUsersList() {
    const theme = useTheme();
    const queryData = useLazyLoadQuery<AdminAreaUsersListQuery>(query, { first: INITIAL_ITEM_COUNT }, { fetchPolicy: 'store-and-network' });

    const { data, loadNext, hasNext, refetch } = usePaginationFragment<AdminAreaUsersListPaginationQuery, AdminAreaUsersListFragment_users$key>(
        graphql`
            fragment AdminAreaUsersListFragment_users on Query
            @refetchable(queryName: "AdminAreaUsersListPaginationQuery")
             {
                users(
                    first: $first
                    after: $after
                    search: $search
                ) @connection(key: "AdminAreaUsersList_users") {
                    totalCount
                    edges {
                        node {
                            id
                            ...AdminAreaUserListItemFragment_user
                        }
                    }
                }
            }
        `, queryData);

    const [search, setSearch] = useState<string | undefined>('');
    const [isRefreshing, setIsRefreshing] = useState(false);

    const environment = useRelayEnvironment();

    const fetch = useMemo(
        () =>
            throttle(
                (input?: string) => {
                    setIsRefreshing(true);

                    fetchQuery(environment, query, { first: INITIAL_ITEM_COUNT, search: input })
                        .subscribe({
                            complete: () => {
                                setIsRefreshing(false);
                                setSearch(input);

                                // *After* the query has been fetched, we call
                                // refetch again to re-render with the updated data.
                                // At this point the data for the query should
                                // be cached, so we use the 'store-only'
                                // fetchPolicy to avoid suspending.
                                refetch({ first: INITIAL_ITEM_COUNT, search: input }, { fetchPolicy: 'store-only' });
                            },
                            error: () => {
                                setIsRefreshing(false);
                            }
                        });
                },
                2000,
                { leading: false, trailing: true }
            ),
        [environment, refetch],
    );

    const onSearchChange = (event: React.ChangeEvent<HTMLInputElement>) => {
        fetch(event.target.value.toLowerCase());
    };

    const onKeyPress = (event: React.KeyboardEvent<HTMLInputElement>) => {
        // Only handle enter key type
        if (event.which === 13) {
            fetch.flush();
        }
    };

    const edges = data?.users?.edges ?? [];

    return (
        <Box>
            <AdminAreaBreadcrumbs
                childRoutes={[
                    { title: "users", path: 'users' }
                ]}
            />
            {(search !== '' || edges.length !== 0) && <React.Fragment>
                <Typography variant="h5" sx={{ marginBottom: 2 }}>Users</Typography>
                <Box marginBottom={2}>
                    <SearchInput
                        fullWidth
                        placeholder="search for users"
                        onChange={onSearchChange}
                        onKeyPress={onKeyPress}
                    />
                </Box>
                <Paper sx={{ borderBottomLeftRadius: 0, borderBottomRightRadius: 0, border: `1px solid ${theme.palette.divider}` }}>
                    <Box padding={2} display="flex" alignItems="center" justifyContent="space-between">
                        <Typography variant="subtitle1">
                            {data?.users.totalCount} user{data?.users.totalCount === 1 ? '' : 's'}
                        </Typography>
                    </Box>
                </Paper>
                <Box sx={{
                    borderBottom: `1px solid ${theme.palette.divider}`,
                    borderLeft: `1px solid ${theme.palette.divider}`,
                    borderRight: `1px solid ${theme.palette.divider}`,
                    borderBottomLeftRadius: 4,
                    borderBottomRightRadius: 4
                }}>
                    {(edges.length === 0) && search !== '' && <Typography
                        align="center"
                        color="textSecondary"
                        padding={4}
                    >
                        No users matching search <strong>{search}</strong>
                    </Typography>}
                    {edges.length > 0 && <InfiniteScroll
                        dataLength={data?.users?.edges?.length ?? 0}
                        next={() => loadNext(20)}
                        hasMore={hasNext}
                        loader={<ListSkeleton rowCount={3} />}
                    >
                        <TableContainer>
                            <Table sx={isRefreshing ? { opacity: 0.5 } : null}>
                                <TableHead>
                                    <TableRow>
                                        <TableCell>Name</TableCell>
                                        <TableCell>SCIM</TableCell>
                                        <TableCell>Created</TableCell>
                                        <TableCell></TableCell>
                                    </TableRow>
                                </TableHead>
                                <TableBody>
                                    {data?.users?.edges?.map((edge: any) => (
                                        <AdminAreaUserListItem key={edge.node.id} fragmentRef={edge.node} />
                                    ))}
                                </TableBody>
                            </Table>
                        </TableContainer>
                    </InfiniteScroll>}
                </Box>
            </React.Fragment>}
        </Box>
    );
}

export default AdminAreaUsersList;
