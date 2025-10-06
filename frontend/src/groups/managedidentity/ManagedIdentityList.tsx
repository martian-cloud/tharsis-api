import { Box, Button, List, Paper, Typography, useTheme } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import throttle from 'lodash.throttle';
import React, { useMemo, useState } from 'react';
import InfiniteScroll from 'react-infinite-scroll-component';
import { ConnectionHandler, fetchQuery, useLazyLoadQuery, usePaginationFragment, useFragment, useRelayEnvironment } from "react-relay/hooks";
import { Link as RouterLink } from 'react-router-dom';
import SearchInput from '../../common/SearchInput';
import ListSkeleton from '../../skeletons/ListSkeleton';
import NamespaceBreadcrumbs from '../../namespace/NamespaceBreadcrumbs';
import ManagedIdentityListItem from './ManagedIdentityListItem';
import { ManagedIdentityListFragment_group$key } from './__generated__/ManagedIdentityListFragment_group.graphql';
import { ManagedIdentityListFragment_managedIdentities$key } from './__generated__/ManagedIdentityListFragment_managedIdentities.graphql';
import { ManagedIdentityListPaginationQuery } from './__generated__/ManagedIdentityListPaginationQuery.graphql';
import { ManagedIdentityListQuery } from './__generated__/ManagedIdentityListQuery.graphql';

const DESCRIPTION = 'Managed identities provide credentials to the Terraform providers without having to store credentials.'
const INITIAL_ITEM_COUNT = 100;

const query = graphql`
    query ManagedIdentityListQuery($first: Int, $last: Int, $after: String, $before: String, $groupId: String!, $search: String) {
        node(id: $groupId) {
            ...on Group {
                ...ManagedIdentityListFragment_managedIdentities
            }
        }
    }
`;

interface Props {
    fragmentRef: ManagedIdentityListFragment_group$key
}

export function GetConnections(groupId: string): [string] {
    const connectionId = ConnectionHandler.getConnectionID(
        groupId,
        'ManagedIdentityList_managedIdentities',
        { sort: 'UPDATED_AT_DESC' }
    );
    return [connectionId];
}

function ManagedIdentityList(props: Props) {
    const theme = useTheme();
    const [search, setSearch] = useState<string | undefined>('');
    const [isRefreshing, setIsRefreshing] = useState(false);

    const group = useFragment<ManagedIdentityListFragment_group$key>(
        graphql`
        fragment ManagedIdentityListFragment_group on Group
        {
            id
            fullPath
        }
    `, props.fragmentRef);

    const queryData = useLazyLoadQuery<ManagedIdentityListQuery>(query, { first: INITIAL_ITEM_COUNT, groupId: group.id }, { fetchPolicy: 'store-and-network' })

    const { data, loadNext, hasNext, refetch } = usePaginationFragment<ManagedIdentityListPaginationQuery, ManagedIdentityListFragment_managedIdentities$key>(
        graphql`
        fragment ManagedIdentityListFragment_managedIdentities on Group
        @refetchable(queryName: "ManagedIdentityListPaginationQuery") {
            managedIdentities(
                after: $after
                before: $before
                first: $first
                last: $last
                search: $search
                includeInherited: true
                sort: GROUP_LEVEL_DESC
            ) @connection(key: "ManagedIdentityList_managedIdentities") {
                totalCount
                edges {
                    node {
                        id
                        groupPath
                        ...ManagedIdentityListItemFragment_managedIdentity
                    }
                }
            }
        }
        `, queryData.node
    );

    const environment = useRelayEnvironment();

    const fetch = useMemo(
        () =>
            throttle(
                (input?: string) => {
                    setIsRefreshing(true);

                    fetchQuery(environment, query, { first: INITIAL_ITEM_COUNT, groupId: group.id, search: input })
                        .subscribe({
                            complete: () => {
                                setIsRefreshing(false);
                                setSearch(input);

                                // *After* the query has been fetched, we call
                                // refetch again to re-render with the updated data.
                                // At this point the data for the query should
                                // be cached, so we use the 'store-only'
                                // fetchPolicy to avoid suspending.
                                refetch({
                                    first: INITIAL_ITEM_COUNT,
                                    search: input
                                }, {
                                    fetchPolicy: 'store-only'
                                });
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


    const onSearchChange = (event: React.ChangeEvent<HTMLInputElement>) => {
        fetch(event.target.value.toLowerCase());
    };

    const onKeyPress = (event: React.KeyboardEvent<HTMLInputElement>) => {
        // Only handle enter key type
        if (event.which === 13) {
            fetch.flush();
        }
    };

    return (
        <Box>
            <NamespaceBreadcrumbs
                namespacePath={group.fullPath}
                childRoutes={[
                    { title: "managed identities", path: 'managed_identities' }
                ]}
            />
            {(search !== '' || data?.managedIdentities.edges?.length !== 0) && <Box>
                <Box>
                    <Box sx={{
                        display: 'flex',
                        flexDirection: 'row',
                        justifyContent: 'space-between',
                        [theme.breakpoints.down('md')]: {
                            flexDirection: 'column',
                            alignItems: 'flex-start',
                            '& > *': { marginBottom: 2 },
                        }
                    }}>
                        <Box>
                            <Typography variant="h5" gutterBottom>Managed Identities</Typography>
                            <Typography variant="body2">
                                {DESCRIPTION}
                            </Typography>
                        </Box>
                        <Box>
                            <Button sx={{ minWidth: 220 }} component={RouterLink} variant="outlined" to="new">New Managed Identity</Button>
                        </Box>
                    </Box>
                    <SearchInput
                        sx={{ marginTop: 2, marginBottom: 2 }}
                        placeholder="search for managed identities"
                        fullWidth
                        onChange={onSearchChange}
                        onKeyPress={onKeyPress}
                    />
                </Box>
                <Paper sx={{ borderBottomLeftRadius: 0, borderBottomRightRadius: 0, border: `1px solid ${theme.palette.divider}` }}>
                    <Box padding={2} display="flex" alignItems="center" justifyContent="space-between">
                        <Typography variant="subtitle1">
                            {data?.managedIdentities.edges?.length} managed identit{data?.managedIdentities.edges?.length === 1 ? 'y' : 'ies'}
                        </Typography>
                    </Box>
                </Paper>
                {(data?.managedIdentities.edges?.length === 0) && search !== '' && <Typography
                    sx={{
                        padding: 4,
                        borderBottom: `1px solid ${theme.palette.divider}`,
                        borderLeft: `1px solid ${theme.palette.divider}`,
                        borderRight: `1px solid ${theme.palette.divider}`,
                        borderBottomLeftRadius: 4,
                        borderBottomRightRadius: 4
                    }}
                    align="center"
                    color="textSecondary"
                >
                    No managed identities matching search <strong>{search}</strong>
                </Typography>}
                <InfiniteScroll
                    dataLength={data?.managedIdentities.edges?.length ?? 0}
                    next={() => loadNext(20)}
                    hasMore={hasNext}
                    loader={<ListSkeleton rowCount={3} />}
                >
                    <List sx={isRefreshing ? { opacity: 0.5 } : null} disablePadding>
                        {data?.managedIdentities.edges?.map((edge: any) => <ManagedIdentityListItem
                            key={edge.node.id}
                            fragmentRef={edge.node}
                            inherited={edge.node.groupPath !== group.fullPath}
                        />)}
                    </List>
                </InfiniteScroll>
            </Box>}
            {search === '' && data?.managedIdentities.edges?.length === 0 && <Box sx={{ marginTop: 4 }} display="flex" justifyContent="center">
                <Box padding={4} display="flex" flexDirection="column" justifyContent="center" alignItems="center" sx={{ maxWidth: 600 }}>
                    <Typography variant="h6">Get started with managed identities</Typography>
                    <Typography color="textSecondary" align="center" sx={{ marginBottom: 2 }}>
                        {DESCRIPTION}
                    </Typography>
                    <Button component={RouterLink} variant="outlined" to="new">New Managed Identity</Button>
                </Box>
            </Box>}
        </Box>
    );
}

export default ManagedIdentityList;
