import { useMemo, useState } from 'react';
import { Box, Button, List, Paper, Typography, useTheme } from '@mui/material'
import InfiniteScroll from 'react-infinite-scroll-component';
import ListSkeleton from '../../skeletons/ListSkeleton';
import graphql from 'babel-plugin-relay/macro';
import throttle from 'lodash.throttle';
import { ConnectionHandler, fetchQuery, useFragment, useLazyLoadQuery, usePaginationFragment, useRelayEnvironment } from 'react-relay/hooks';
import { Link as RouterLink } from 'react-router-dom';
import SearchInput from '../../common/SearchInput';
import NamespaceBreadcrumbs from '../../namespace/NamespaceBreadcrumbs';
import VCSProviderListItem from './VCSProviderListItem';
import { VCSProviderListFragment_group$key } from './__generated__/VCSProviderListFragment_group.graphql'
import { VCSProviderListQuery } from './__generated__/VCSProviderListQuery.graphql';
import { VCSProviderListPaginationQuery } from './__generated__/VCSProviderListPaginationQuery.graphql';
import { VCSProviderListFragment_vcsProviders$key } from './__generated__/VCSProviderListFragment_vcsProviders.graphql';

const DESCRIPTION = 'Version control system (VCS) providers allow connections between workspaces and Git repositories.';
const INITIAL_ITEM_COUNT = 100;

const query = graphql`
    query VCSProviderListQuery($first: Int, $last: Int, $after: String, $before: String, $groupId: String!, $search: String) {
        node(id: $groupId) {
            ...on Group {
                ...VCSProviderListFragment_vcsProviders
            }
        }
    }
`;

interface Props {
    fragmentRef: VCSProviderListFragment_group$key
}

export function GetConnections(groupId: string): [string] {
    const connectionId = ConnectionHandler.getConnectionID(
        groupId,
        'VCSProviderList_vcsProviders',
        { includeInherited: false }
    );
    return [connectionId];
}

function VCSProviderList(props: Props) {
    const theme = useTheme();
    const [search, setSearch] = useState<string | undefined>('');
    const [isRefreshing, setIsRefreshing] = useState(false);

    const group = useFragment<VCSProviderListFragment_group$key>(
        graphql`
        fragment VCSProviderListFragment_group on Group
        {
            id
            fullPath
        }
    `, props.fragmentRef);

    const queryData = useLazyLoadQuery<VCSProviderListQuery>(query, { first: INITIAL_ITEM_COUNT, groupId: group.id }, { fetchPolicy: 'store-and-network' })

    const { data, loadNext, hasNext, refetch } = usePaginationFragment<VCSProviderListPaginationQuery,VCSProviderListFragment_vcsProviders$key>(
        graphql`
      fragment VCSProviderListFragment_vcsProviders on Group
      @refetchable(queryName: "VCSProviderListPaginationQuery") {
            vcsProviders(
                after: $after
                before: $before
                first: $first
                last: $last
                search: $search
                includeInherited: true
                sort: GROUP_LEVEL_DESC
            ) @connection(key: "VCSProviderList_vcsProviders") {
                totalCount
                edges {
                    node {
                        id
                        groupPath
                        ...VCSProviderListItemFragment_vcsProvider
                    }
                }
            }
      }
    `, queryData.node);

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
                    { title: "vcs providers", path: "vcs_providers" }
                ]}
            />
            {(search !== '' || data?.vcsProviders.edges?.length !== 0) && <Box>
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
                            <Typography variant="h5" gutterBottom>VCS Providers</Typography>
                            <Typography variant="body2">
                                {DESCRIPTION}
                            </Typography>
                        </Box>
                        <Box>
                            <Button sx={{ minWidth: 200 }} component={RouterLink} variant="outlined" to="new">
                                New VCS Provider
                            </Button>
                        </Box>
                    </Box>
                    <SearchInput
                        sx={{ marginTop: 2, marginBottom: 2 }}
                        placeholder="search for VCS providers"
                        fullWidth
                        onChange={onSearchChange}
                        onKeyPress={onKeyPress}
                    />
                </Box>
                <Paper sx={{ borderBottomLeftRadius: 0, borderBottomRightRadius: 0, border: `1px solid ${theme.palette.divider}` }}>
                    <Box padding={2} display="flex" alignItems="center" justifyContent="space-between">
                        <Typography variant="subtitle1">
                            {data?.vcsProviders.totalCount} VCS provider{data?.vcsProviders.totalCount === 1 ? '' : 's'}
                        </Typography>
                    </Box>
                </Paper>
                {(data?.vcsProviders.edges?.length === 0) && search !== '' && <Typography
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
                    No VCS providers matching search <strong>{search}</strong>
                </Typography>}
                <InfiniteScroll
                    dataLength={data?.vcsProviders.edges?.length ?? 0}
                    next={() => loadNext(20)}
                    hasMore={hasNext}
                    loader={<ListSkeleton rowCount={3} />}
                >
                    <List sx={isRefreshing ? { opacity: 0.5 } : null} disablePadding>
                        {data?.vcsProviders.edges?.map((edge: any) => <VCSProviderListItem
                            key={edge.node.id}
                            fragmentRef={edge.node}
                            inherited={group.fullPath !== edge.node.groupPath}
                        />)}
                    </List>
                </InfiniteScroll>
            </Box>}
            {search === '' && data?.vcsProviders.edges?.length === 0 && <Box sx={{ mt: 4 }} display="flex" justifyContent="center">
                <Box padding={4} display="flex" flexDirection="column" justifyContent="center" alignItems="center" sx={{ maxWidth: 600 }}>
                    <Typography variant="h6">Get started with VCS providers</Typography>
                    <Typography color="textSecondary" align="center" sx={{ mb: 2 }}>{DESCRIPTION}</Typography>
                    <Button component={RouterLink} variant="outlined" to="new">New VCS Provider</Button>
                </Box>
            </Box>}
        </Box>
    );
}

export default VCSProviderList;
