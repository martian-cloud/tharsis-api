import { Box, List, Paper, Typography, useTheme } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import throttle from 'lodash.throttle';
import React, { useMemo, useState } from 'react';
import InfiniteScroll from 'react-infinite-scroll-component';
import { fetchQuery, useFragment, useLazyLoadQuery, usePaginationFragment, useRelayEnvironment } from "react-relay/hooks";
import SearchInput from '../../common/SearchInput';
import NamespaceBreadcrumbs from '../../namespace/NamespaceBreadcrumbs';
import ListSkeleton from '../../skeletons/ListSkeleton';
import TerraformModuleListItem from './TerraformModuleListItem';
import { TerraformModuleListFragment_group$key } from './__generated__/TerraformModuleListFragment_group.graphql';
import { TerraformModuleListFragment_terraformModules$key } from './__generated__/TerraformModuleListFragment_terraformModules.graphql';
import { TerraformModuleListPaginationQuery } from './__generated__/TerraformModuleListPaginationQuery.graphql';
import { TerraformModuleListQuery } from './__generated__/TerraformModuleListQuery.graphql';

const DESCRIPTION = 'Terraform modules provide reusable infrastructure components that can be shared across workspaces';
const INITIAL_ITEM_COUNT = 20;

const query = graphql`
    query TerraformModuleListQuery($first: Int, $last: Int, $after: String, $before: String, $groupId: String!, $search: String) {
        node(id: $groupId) {
            ...on Group {
                ...TerraformModuleListFragment_terraformModules
            }
        }
    }
`;

interface Props {
    fragmentRef: TerraformModuleListFragment_group$key
}

function TerraformModuleList({ fragmentRef }: Props) {
    const theme = useTheme();
    const [search, setSearch] = useState<string | undefined>('');
    const [isRefreshing, setIsRefreshing] = useState(false);

    const group = useFragment<TerraformModuleListFragment_group$key>(
        graphql`
        fragment TerraformModuleListFragment_group on Group {
            id
            fullPath
        }
        `, fragmentRef);

    const queryData = useLazyLoadQuery<TerraformModuleListQuery>(
        query, 
        { first: INITIAL_ITEM_COUNT, groupId: group.id }, 
        { fetchPolicy: 'store-and-network' }
    );

    const { data, loadNext, hasNext, refetch } = usePaginationFragment<TerraformModuleListPaginationQuery, TerraformModuleListFragment_terraformModules$key>(
        graphql`
        fragment TerraformModuleListFragment_terraformModules on Group
        @refetchable(queryName: "TerraformModuleListPaginationQuery") {
            terraformModules(
                after: $after
                before: $before
                first: $first
                last: $last
                search: $search
                sort: NAME_ASC
            ) @connection(key: "TerraformModuleList_terraformModules") {
                totalCount
                edges {
                    node {
                        id
                        ...TerraformModuleListItemFragment_terraformModule
                    }
                }
            }
        }
        `, queryData.node
    );

    // Use all edges - filtering should be done server-side
    const edges = data?.terraformModules?.edges || [];

    const environment = useRelayEnvironment();

    const fetch = useMemo(
        () =>
            throttle(
                (input?: string) => {
                    setIsRefreshing(true);

                    fetchQuery(environment, query, { 
                        first: INITIAL_ITEM_COUNT,
                        groupId: group.id,
                        search: input
                    })
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
                                    search: input,
                                }, {
                                    fetchPolicy: 'store-only'
                                });
                            },
                            error: () => {
                                setIsRefreshing(false);
                            }
                        });
                },
                500,
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
                    { title: "terraform modules", path: 'terraform_modules' }
                ]}
            />
            {(search !== '' || edges.length !== 0) && <Box>
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
                            <Typography variant="h5" gutterBottom>Terraform Modules</Typography>
                            <Typography variant="body2">
                                {DESCRIPTION}
                            </Typography>
                        </Box>
                    </Box>
                    <SearchInput
                        sx={{ marginTop: 2, marginBottom: 2 }}
                        placeholder="search for terraform modules"
                        fullWidth
                        onChange={onSearchChange}
                        onKeyPress={onKeyPress}
                    />
                </Box>
                <Paper sx={{ borderBottomLeftRadius: 0, borderBottomRightRadius: 0, border: `1px solid ${theme.palette.divider}` }}>
                    <Box padding={2} display="flex" alignItems="center" justifyContent="space-between">
                        <Typography variant="subtitle1">
                            {edges.length} terraform module{edges.length === 1 ? '' : 's'}
                        </Typography>
                    </Box>
                </Paper>
                {(edges.length === 0) && search !== '' && <Typography
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
                    No terraform modules matching search <strong>{search}</strong>
                </Typography>}
                <InfiniteScroll
                    dataLength={edges.length}
                    next={() => loadNext(20)}
                    hasMore={hasNext}
                    loader={<ListSkeleton rowCount={3} />}
                >
                    <List sx={isRefreshing ? { opacity: 0.5 } : null} disablePadding>
                        {edges.map((edge: any) => <TerraformModuleListItem
                            key={edge.node.id}
                            fragmentRef={edge.node}
                        />)}
                    </List>
                </InfiniteScroll>
            </Box>}
            {search === '' && edges.length === 0 && <Box sx={{ marginTop: 4 }} display="flex" justifyContent="center">
                <Box padding={4} display="flex" flexDirection="column" justifyContent="center" alignItems="center" sx={{ maxWidth: 600 }}>
                    <Typography variant="h6">No terraform modules found</Typography>
                    <Typography color="textSecondary" align="center" sx={{ marginBottom: 2 }}>
                        {DESCRIPTION}
                    </Typography>
                </Box>
            </Box>}
        </Box >
    );
}

export default TerraformModuleList;
