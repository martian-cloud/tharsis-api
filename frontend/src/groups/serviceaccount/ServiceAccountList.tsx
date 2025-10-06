import { Box, Button, List, Paper, Typography, useTheme } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import throttle from 'lodash.throttle';
import React, { useMemo, useState } from 'react';
import InfiniteScroll from 'react-infinite-scroll-component';
import { ConnectionHandler, fetchQuery, useFragment, useLazyLoadQuery, usePaginationFragment, useRelayEnvironment } from "react-relay/hooks";
import { Link as RouterLink } from 'react-router-dom';
import SearchInput from '../../common/SearchInput';
import NamespaceBreadcrumbs from '../../namespace/NamespaceBreadcrumbs';
import ListSkeleton from '../../skeletons/ListSkeleton';
import ServiceAccountListItem from './ServiceAccountListItem';
import { ServiceAccountListFragment_group$key } from './__generated__/ServiceAccountListFragment_group.graphql';
import { ServiceAccountListFragment_serviceAccounts$key } from './__generated__/ServiceAccountListFragment_serviceAccounts.graphql';
import { ServiceAccountListPaginationQuery } from './__generated__/ServiceAccountListPaginationQuery.graphql';
import { ServiceAccountListQuery } from './__generated__/ServiceAccountListQuery.graphql';

const DESCRIPTION = 'Service accounts provide access to the Tharsis API using a token from an OpenID Connect-compatible identity provider';
const INITIAL_ITEM_COUNT = 100;

const query = graphql`
    query ServiceAccountListQuery($first: Int, $last: Int, $after: String, $before: String, $groupId: String!, $search: String) {
        node(id: $groupId) {
            ...on Group {
                ...ServiceAccountListFragment_serviceAccounts
            }
        }
    }
`;

interface Props {
    fragmentRef: ServiceAccountListFragment_group$key
}

export function GetConnections(groupId: string): [string] {
    const connectionId = ConnectionHandler.getConnectionID(
        groupId,
        'ServiceAccountList_serviceAccounts',
        { sort: 'GROUP_LEVEL_DESC', includeInherited: true }
    );
    return [connectionId];
}

function ServiceAccountList({ fragmentRef }: Props) {
    const theme = useTheme();
    const [search, setSearch] = useState<string | undefined>('');
    const [isRefreshing, setIsRefreshing] = useState(false);

    const group = useFragment<ServiceAccountListFragment_group$key>(
        graphql`
        fragment ServiceAccountListFragment_group on Group
        {
            id
            fullPath
        }
    `, fragmentRef);

    const queryData = useLazyLoadQuery<ServiceAccountListQuery>(query, { first: INITIAL_ITEM_COUNT, groupId: group.id }, { fetchPolicy: 'store-and-network' })

    const { data, loadNext, hasNext, refetch } = usePaginationFragment<ServiceAccountListPaginationQuery, ServiceAccountListFragment_serviceAccounts$key>(
        graphql`
        fragment ServiceAccountListFragment_serviceAccounts on Group
        @refetchable(queryName: "ServiceAccountListPaginationQuery") {
            serviceAccounts(
                after: $after
                before: $before
                first: $first
                last: $last
                includeInherited: true
                search: $search
                sort: GROUP_LEVEL_DESC
            ) @connection(key: "ServiceAccountList_serviceAccounts") {
                totalCount
                edges {
                    node {
                        id
                        groupPath
                        ...ServiceAccountListItemFragment_serviceAccount
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
                    { title: "service accounts", path: 'service_accounts' }
                ]}
            />
            {(search !== '' || data?.serviceAccounts.edges?.length !== 0) && <Box>
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
                            <Typography variant="h5" gutterBottom>Service Accounts</Typography>
                            <Typography variant="body2">
                                {DESCRIPTION}
                            </Typography>
                        </Box>
                        <Box>
                            <Button sx={{ minWidth: 200 }} component={RouterLink} variant="outlined" to="new">
                                New Service Account
                            </Button>
                        </Box>
                    </Box>
                    <SearchInput
                        sx={{ marginTop: 2, marginBottom: 2 }}
                        placeholder="search for service accounts"
                        fullWidth
                        onChange={onSearchChange}
                        onKeyPress={onKeyPress}
                    />
                </Box>
                <Paper sx={{ borderBottomLeftRadius: 0, borderBottomRightRadius: 0, border: `1px solid ${theme.palette.divider}` }}>
                    <Box padding={2} display="flex" alignItems="center" justifyContent="space-between">
                        <Typography variant="subtitle1">
                            {data?.serviceAccounts.totalCount} service account{data?.serviceAccounts.totalCount === 1 ? '' : 's'}
                        </Typography>
                    </Box>
                </Paper>
                {(data?.serviceAccounts.edges?.length === 0) && search !== '' && <Typography
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
                    No service accounts matching search <strong>{search}</strong>
                </Typography>}
                <InfiniteScroll
                    dataLength={data?.serviceAccounts.edges?.length ?? 0}
                    next={() => loadNext(20)}
                    hasMore={hasNext}
                    loader={<ListSkeleton rowCount={3} />}
                >
                    <List sx={isRefreshing ? { opacity: 0.5 } : null} disablePadding>
                        {data?.serviceAccounts.edges?.map((edge: any) => <ServiceAccountListItem
                            key={edge.node.id}
                            fragmentRef={edge.node}
                            inherited={edge.node.groupPath !== group.fullPath}
                        />)}
                    </List>
                </InfiniteScroll>
            </Box>}
            {search === '' && data?.serviceAccounts.edges?.length === 0 && <Box sx={{ marginTop: 4 }} display="flex" justifyContent="center">
                <Box padding={4} display="flex" flexDirection="column" justifyContent="center" alignItems="center" sx={{ maxWidth: 600 }}>
                    <Typography variant="h6">Get started with service accounts</Typography>
                    <Typography color="textSecondary" align="center" sx={{ marginBottom: 2 }}>
                        {DESCRIPTION}
                    </Typography>
                    <Button component={RouterLink} variant="outlined" to="new">New Service Account</Button>
                </Box>
            </Box>}
        </Box >
    );
}

export default ServiceAccountList;
